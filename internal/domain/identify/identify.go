// Package identify orchestrates the sqlid command: it collects SQL inputs from
// arguments and standard input, computes each statement's SQL ID, hash, and
// normalized form via the reusable github.com/sqlrest/sqlid library, and renders
// the requested output lines.
//
// This is the domain tier between the app tier (internal/app, the urfave/cli
// wiring) and the implementation tier (the public github.com/sqlrest/sqlid
// library). It contains no CLI, flag-parsing, or process-I/O logic; the file
// system and input streams are injected so it stays testable.
package identify

import (
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/sqlrest/sqlid"
	"github.com/sqlrest/sqlid/internal/constants"
)

// FileSystem abstracts the file operations the domain performs so they can be
// substituted in tests. The app tier injects the os-backed implementation.
type FileSystem struct {
	Stat func(string) (fs.FileInfo, error)
	Read func(string) ([]byte, error)
}

// Output captures the selected output mode.
type Output struct {
	Format       string
	IsIDOnly     bool
	IsHashOnly   bool
	HasFormat    bool
	TabsEnabled  bool
	IsVerbose    bool
	NameDisabled bool
}

// Config holds the resolved settings for one invocation: the normalization
// toggles (negated flags), the output mode, and whether standard input should
// be read.
type Config struct {
	Output              Output
	ShouldKeepCase      bool
	UncommentDisabled   bool
	CompressDisabled    bool
	NewlineDisabled     bool
	ShouldKeepWith      bool
	ShouldKeepConst     bool
	ShouldKeepSemicolon bool
	ShouldReadStdin     bool
}

// options translates the configuration's toggles into normalization options.
func (c Config) options() []sqlid.Option {
	return []sqlid.Option{
		sqlid.Lowercase(!c.ShouldKeepCase),
		sqlid.Uncomment(!c.UncommentDisabled && !c.CompressDisabled),
		sqlid.StripSemicolon(!c.ShouldKeepSemicolon),
		sqlid.Compress(!c.CompressDisabled),
		sqlid.Newline(!c.NewlineDisabled),
		sqlid.RewriteWith(!c.ShouldKeepWith),
		sqlid.StripConstants(!c.ShouldKeepConst),
	}
}

// argument is one positional CLI argument: a file path or literal SQL text.
type argument string

// position is an argument's zero-based place on the command line.
type position int

// statement is a single input to process, with its display name and origin.
type statement struct {
	name        string
	sql         sqlid.Statement
	isFromStdin bool
}

// result holds the computed fields available to the output formatters.
type result struct {
	id         sqlid.ID
	name       string
	normalized sqlid.Statement
	original   sqlid.Statement
	hash       sqlid.Hash
}

// fromArg resolves a positional argument to a file's contents or a literal SQL
// string.
func fromArg(filesys FileSystem, arg argument, index position) (statement, error) {
	if info, err := filesys.Stat(string(arg)); err == nil && !info.IsDir() {
		data, err := filesys.Read(string(arg))
		if err != nil {
			return statement{}, constants.ErrReadFile.With(nil, string(arg))
		}
		return statement{name: string(arg), sql: sqlid.Statement(data)}, nil
	}
	return statement{name: fmt.Sprintf("arg[%d]", int(index)), sql: sqlid.Statement(arg)}, nil
}

// collect gathers all inputs from positional arguments and, when the
// configuration asks for it, standard input.
func (c Config) collect(filesys FileSystem, args []string, stdin io.Reader) ([]statement, error) {
	statements := make([]statement, 0, len(args)+1)
	for index, arg := range args {
		in, err := fromArg(filesys, argument(arg), position(index))
		if err != nil {
			return nil, err
		}
		statements = append(statements, in)
	}
	if !c.ShouldReadStdin {
		return statements, nil
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		return nil, constants.ErrReadStdin.With(err)
	}
	return append(statements, statement{name: "--", sql: sqlid.Statement(data), isFromStdin: true}), nil
}

// field resolves a single format-string character to its result field.
func (r result) field(char rune) string {
	switch char {
	case 'i':
		return string(r.id)
	case 'h':
		return fmt.Sprint(r.hash)
	case 'n':
		return r.name
	case 'q', 'c':
		return string(r.normalized)
	case 's':
		return string(r.original)
	}
	return string(char)
}

// formatLine renders a custom format string against the result.
func (r result) formatLine(template string) string {
	template = strings.ReplaceAll(template, `\n`, "\n")
	template = strings.ReplaceAll(template, `\t`, "\t")
	var b strings.Builder
	for _, char := range template {
		// strings.Builder.WriteString never returns a non-nil error.
		_, _ = b.WriteString(r.field(char))
	}
	return b.String()
}

// renderer holds the resolved output context applied to every statement: the
// normalization options, the output mode, the column separator, and whether
// the bare (single stdin input) form applies.
type renderer struct {
	opts   []sqlid.Option
	sep    string
	out    Output
	isBare bool
}

// compute normalizes a statement and derives its identifiers.
func (rd renderer) compute(in statement) result {
	normalized := sqlid.Normalize(in.sql, rd.opts...)
	name := in.name
	if rd.out.NameDisabled {
		name = ""
	}
	return result{
		id:         sqlid.SQLRawID(normalized),
		hash:       sqlid.SQLRawHash(normalized),
		name:       name,
		normalized: normalized,
		original:   in.sql,
	}
}

// pair renders a value optionally followed by a separator and name.
func (rd renderer) pair(value, name string) string {
	if rd.isBare {
		return value
	}
	return value + rd.sep + name
}

// fullLine renders the default id/hash/name columns (plus normalized when verbose).
func (rd renderer) fullLine(r result) string {
	columns := []string{string(r.id), fmt.Sprint(r.hash), r.name}
	if rd.out.IsVerbose {
		columns = append(columns, string(r.normalized))
	}
	if rd.isBare {
		return strings.Join(columns[:2], rd.sep)
	}
	return strings.Join(columns, rd.sep)
}

// line renders one result according to the output mode.
func (rd renderer) line(r result) string {
	if rd.out.IsIDOnly {
		return rd.pair(string(r.id), r.name)
	}
	if rd.out.IsHashOnly {
		return rd.pair(fmt.Sprint(r.hash), r.name)
	}
	return rd.fullLine(r)
}

// renderOne computes and renders a single statement.
func (rd renderer) renderOne(in statement) string {
	r := rd.compute(in)
	if rd.out.HasFormat {
		return r.formatLine(rd.out.Format)
	}
	return rd.line(r)
}

// render produces one output line per non-blank input.
func render(statements []statement, opts []sqlid.Option, o Output) []string {
	rd := renderer{
		opts:   opts,
		sep:    " ",
		out:    o,
		isBare: len(statements) == 1 && statements[0].isFromStdin,
	}
	if o.TabsEnabled {
		rd.sep = "\t"
	}
	lines := make([]string, 0, len(statements))
	for _, in := range statements {
		if strings.TrimSpace(string(in.sql)) == "" {
			continue
		}
		lines = append(lines, rd.renderOne(in))
	}
	return lines
}

// join concatenates lines, each terminated by a newline.
func join(lines []string) string {
	var b strings.Builder
	for _, l := range lines {
		// strings.Builder writes never return a non-nil error.
		_, _ = b.WriteString(l)
		_ = b.WriteByte('\n')
	}
	return b.String()
}

// Run collects the inputs, computes and renders each statement, and returns the
// combined text. It validates nothing beyond what collection requires and
// delegates all SQL work to the sqlid library.
func Run(cfg Config, filesys FileSystem, args []string, stdin io.Reader) (string, error) {
	statements, err := cfg.collect(filesys, args, stdin)
	if err != nil {
		return "", err
	}
	return join(render(statements, cfg.options(), cfg.Output)), nil
}
