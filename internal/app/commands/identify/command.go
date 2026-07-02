// Package identify defines the sqlid CLI command: the urfave/cli command and
// flags, the mapping from parsed flags to the identify domain configuration,
// and the action that runs the domain and writes its output to a file or
// stdout.
//
// This is the command tier between the app entrypoint (internal/app) and the
// identify domain (internal/domain/identify); it contains no SQL-computation
// or rendering logic.
package identify

import (
	"context"
	"io"
	"os"

	"github.com/urfave/cli/v3"
	"golang.org/x/term"

	"github.com/sqlrest/sqlid/internal/constants"
	domain "github.com/sqlrest/sqlid/internal/domain/identify"
)

// Name is the command's program name, used for the CLI command and argv[0].
const Name = "sqlid"

// Version is the build-time version string exposed via --version as
// "sqlid version <version>".
type Version string

func flags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{Name: "id", Aliases: []string{"i"}, Usage: "only output the SQL ID"},
		&cli.BoolFlag{Name: "hash", Aliases: []string{"a"}, Usage: "only output the SQL hash"},
		&cli.StringFlag{Name: "format", Aliases: []string{"F"}, Usage: "format string of i,h,n,q,s characters"},
		&cli.BoolFlag{Name: "tabs", Aliases: []string{"t"}, Usage: "separate fields with tabs"},
		&cli.BoolFlag{Name: "verbose", Aliases: []string{"v"}, Usage: "also output the normalized SQL"},
		&cli.BoolFlag{Name: "no-name", Aliases: []string{"N"}, Usage: "omit the input name"},
		&cli.BoolFlag{Name: "case", Aliases: []string{"I"}, Usage: "keep case (do not lowercase)"},
		&cli.BoolFlag{Name: "no-uncomment", Aliases: []string{"C"}, Usage: "keep comments"},
		&cli.BoolFlag{Name: "no-compress", Aliases: []string{"Z"}, Usage: "do not compress whitespace or comments"},
		&cli.BoolFlag{Name: "no-newline", Aliases: []string{"L"}, Usage: "do not append a trailing newline"},
		&cli.BoolFlag{Name: "keep-with", Aliases: []string{"W"}, Usage: "keep WITH-clause aliases"},
		&cli.BoolFlag{Name: "keep-const", Aliases: []string{"R"}, Usage: "keep string and numeric literals"},
		&cli.BoolFlag{Name: "semicolon", Aliases: []string{"S"}, Usage: "keep a trailing semicolon"},
		&cli.BoolFlag{Name: "no-stdin", Aliases: []string{"x"}, Usage: "do not read standard input"},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "write output to a file instead of stdout"},
	}
}

// isTerminal reports whether the reader is an interactive terminal.
func isTerminal(reader io.Reader) bool {
	file, ok := reader.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

// config builds the domain configuration from the parsed flags. Standard input
// is read only when not disabled and not connected to a terminal.
func config(cmd *cli.Command, stdin io.Reader) domain.Config {
	return domain.Config{
		ShouldKeepCase:      cmd.Bool("case"),
		UncommentDisabled:   cmd.Bool("no-uncomment"),
		CompressDisabled:    cmd.Bool("no-compress"),
		NewlineDisabled:     cmd.Bool("no-newline"),
		ShouldKeepWith:      cmd.Bool("keep-with"),
		ShouldKeepConst:     cmd.Bool("keep-const"),
		ShouldKeepSemicolon: cmd.Bool("semicolon"),
		Output: domain.Output{
			IsIDOnly:     cmd.Bool("id"),
			IsHashOnly:   cmd.Bool("hash"),
			Format:       cmd.String("format"),
			HasFormat:    cmd.IsSet("format"),
			TabsEnabled:  cmd.Bool("tabs"),
			IsVerbose:    cmd.Bool("verbose"),
			NameDisabled: cmd.Bool("no-name"),
		},
		ShouldReadStdin: !cmd.Bool("no-stdin") && !isTerminal(stdin),
	}
}

// osFS returns the production file system backed by the os package.
func osFS() domain.FileSystem {
	return domain.FileSystem{Stat: os.Stat, Read: os.ReadFile}
}

// execute runs the CLI's work: render the inputs and write them to a file or stdout.
func execute(cmd *cli.Command, filesys domain.FileSystem, stdin io.Reader, stdout io.Writer) error {
	text, err := domain.Run(config(cmd, stdin), filesys, cmd.Args().Slice(), stdin)
	if err != nil {
		return err
	}
	if path := cmd.String("output"); path != "" {
		if writeErr := os.WriteFile(path, []byte(text), 0o600); writeErr != nil {
			return constants.ErrWriteFile.With(writeErr, path)
		}
		return nil
	}
	_, err = io.WriteString(stdout, text)
	return err
}

// Command builds the sqlid command wired to the given streams.
func Command(version Version, stdin io.Reader, stdout, stderr io.Writer) *cli.Command {
	// sqlid uses -v for --verbose, so drop the version flag's default -v alias.
	cli.VersionFlag = &cli.BoolFlag{Name: "version", Usage: "print the version and exit"}
	return &cli.Command{
		Name:           Name,
		Version:        string(version),
		Usage:          "Calculate the SQL ID and SQL hash of each SQL statement",
		ArgsUsage:      "[SQL|FILE]...",
		Flags:          flags(),
		Reader:         stdin,
		Writer:         stdout,
		ErrWriter:      stderr,
		ExitErrHandler: func(context.Context, *cli.Command, error) {},
		Action: func(_ context.Context, cmd *cli.Command) error {
			return execute(cmd, osFS(), stdin, stdout)
		},
	}
}
