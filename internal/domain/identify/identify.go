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
	Format    string
	IDOnly    bool
	HashOnly  bool
	HasFormat bool
	Tabs      bool
	Verbose   bool
	NoName    bool
}

// Config holds the resolved settings for one invocation: the normalization
// toggles (negated flags), the output mode, and whether standard input should
// be read.
type Config struct {
	Output      Output
	KeepCase    bool
	NoUncomment bool
	NoCompress  bool
	NoNewline   bool
	KeepWith    bool
	KeepConst   bool
	Semicolon   bool
	UseStdin    bool
}

// options translates the configuration's toggles into normalization options.
func (c Config) options() []sqlid.Option {
	return []sqlid.Option{
		sqlid.Lowercase(!c.KeepCase),
		sqlid.Uncomment(!c.NoUncomment && !c.NoCompress),
		sqlid.StripSemicolon(!c.Semicolon),
		sqlid.Compress(!c.NoCompress),
		sqlid.Newline(!c.NoNewline),
		sqlid.RewriteWith(!c.KeepWith),
		sqlid.StripConstants(!c.KeepConst),
	}
}

// statement is a single input to process, with its display name and origin.
type statement struct {
	name      string
	sql       sqlid.Statement
	fromStdin bool
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
func fromArg(filesys FileSystem, arg string, index int) (statement, error) {
	if info, err := filesys.Stat(arg); err == nil && !info.IsDir() {
		data, err := filesys.Read(arg)
		if err != nil {
			return statement{}, constants.ErrReadFile.With(nil, arg)
		}
		return statement{name: arg, sql: sqlid.Statement(data)}, nil
	}
	return statement{name: fmt.Sprintf("arg[%d]", index), sql: sqlid.Statement(arg)}, nil
}

// collect gathers all inputs from positional arguments and standard input.
func collect(filesys FileSystem, args []string, stdin io.Reader, useStdin bool) ([]statement, error) {
	statements := make([]statement, 0, len(args)+1)
	for index, arg := range args {
		in, err := fromArg(filesys, arg, index)
		if err != nil {
			return nil, err
		}
		statements = append(statements, in)
	}
	if !useStdin {
		return statements, nil
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		return nil, constants.ErrReadStdin.With(err)
	}
	return append(statements, statement{name: "--", sql: sqlid.Statement(data), fromStdin: true}), nil
}

// compute normalizes a statement and derives its identifiers.
func compute(in statement, opts []sqlid.Option, noName bool) result {
	normalized := sqlid.Normalize(in.sql, opts...)
	name := in.name
	if noName {
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

// field resolves a single format-string character to its result field.
func field(char rune, r result) string {
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

// formatLine renders a custom format string against a result.
func formatLine(template string, r result) string {
	template = strings.ReplaceAll(template, `\n`, "\n")
	template = strings.ReplaceAll(template, `\t`, "\t")
	var b strings.Builder
	for _, char := range template {
		// strings.Builder.WriteString never returns a non-nil error.
		_, _ = b.WriteString(field(char, r))
	}
	return b.String()
}

// pair renders a value optionally followed by a separator and name.
func pair(value, name, sep string, bare bool) string {
	if bare {
		return value
	}
	return value + sep + name
}

// fullLine renders the default id/hash/name columns (plus normalized when verbose).
func fullLine(r result, o Output, sep string, bare bool) string {
	columns := []string{string(r.id), fmt.Sprint(r.hash), r.name}
	if o.Verbose {
		columns = append(columns, string(r.normalized))
	}
	if bare {
		return strings.Join(columns[:2], sep)
	}
	return strings.Join(columns, sep)
}

// line renders one result according to the output mode.
func line(r result, o Output, sep string, bare bool) string {
	if o.IDOnly {
		return pair(string(r.id), r.name, sep, bare)
	}
	if o.HashOnly {
		return pair(fmt.Sprint(r.hash), r.name, sep, bare)
	}
	return fullLine(r, o, sep, bare)
}

// renderOne computes and renders a single statement.
func renderOne(in statement, opts []sqlid.Option, o Output, sep string, bare bool) string {
	r := compute(in, opts, o.NoName)
	if o.HasFormat {
		return formatLine(o.Format, r)
	}
	return line(r, o, sep, bare)
}

// render produces one output line per non-blank input.
func render(statements []statement, opts []sqlid.Option, o Output) []string {
	sep := " "
	if o.Tabs {
		sep = "\t"
	}
	bare := len(statements) == 1 && statements[0].fromStdin
	lines := make([]string, 0, len(statements))
	for _, in := range statements {
		if strings.TrimSpace(string(in.sql)) == "" {
			continue
		}
		lines = append(lines, renderOne(in, opts, o, sep, bare))
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
	statements, err := collect(filesys, args, stdin, cfg.UseStdin)
	if err != nil {
		return "", err
	}
	return join(render(statements, cfg.options(), cfg.Output)), nil
}
