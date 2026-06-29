// Package app implements the sqlid command-line interface: it defines the
// urfave/cli command and flags, builds the domain configuration from the parsed
// flags, and handles process I/O (standard input, terminal detection, and
// writing output to a file or stdout).
//
// This is the app tier; it depends on the identify domain and the constants
// package, and contains no SQL-computation or rendering logic.
package app

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli/v3"
	"golang.org/x/term"

	"github.com/sqlrest/sqlid/internal/constants"
	"github.com/sqlrest/sqlid/internal/domain/identify"
)

// name is the command's program name, used for the CLI command and argv[0].
const name = "sqlid"

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
func config(cmd *cli.Command, stdin io.Reader) identify.Config {
	return identify.Config{
		KeepCase:    cmd.Bool("case"),
		NoUncomment: cmd.Bool("no-uncomment"),
		NoCompress:  cmd.Bool("no-compress"),
		NoNewline:   cmd.Bool("no-newline"),
		KeepWith:    cmd.Bool("keep-with"),
		KeepConst:   cmd.Bool("keep-const"),
		Semicolon:   cmd.Bool("semicolon"),
		Output: identify.Output{
			IDOnly:    cmd.Bool("id"),
			HashOnly:  cmd.Bool("hash"),
			Format:    cmd.String("format"),
			HasFormat: cmd.IsSet("format"),
			Tabs:      cmd.Bool("tabs"),
			Verbose:   cmd.Bool("verbose"),
			NoName:    cmd.Bool("no-name"),
		},
		UseStdin: !cmd.Bool("no-stdin") && !isTerminal(stdin),
	}
}

// osFS returns the production file system backed by the os package.
func osFS() identify.FileSystem {
	return identify.FileSystem{Stat: os.Stat, Read: os.ReadFile}
}

// execute runs the CLI's work: render the inputs and write them to a file or stdout.
func execute(cmd *cli.Command, filesys identify.FileSystem, stdin io.Reader, stdout io.Writer) error {
	text, err := identify.Run(config(cmd, stdin), filesys, cmd.Args().Slice(), stdin)
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

// command builds the sqlid command wired to the given streams. version is the
// build-time version exposed via --version as "sqlid version <version>".
func command(version string, stdin io.Reader, stdout, stderr io.Writer) *cli.Command {
	// sqlid uses -v for --verbose, so drop the version flag's default -v alias.
	cli.VersionFlag = &cli.BoolFlag{Name: "version", Usage: "print the version and exit"}
	return &cli.Command{
		Name:           name,
		Version:        version,
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

// Run executes the sqlid CLI with the given version, arguments, and streams,
// returning the process exit code.
func Run(ctx context.Context, version string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if err := command(version, stdin, stdout, stderr).Run(ctx, args); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
