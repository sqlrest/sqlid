// Package app implements the sqlid command-line entrypoint: it assembles the
// identify command with the process streams and turns its error into a process
// exit code.
//
// This is the app tier; the command definition lives in
// internal/app/commands/identify and the work in internal/domain/identify.
package app

import (
	"context"
	"fmt"
	"io"

	"github.com/sqlrest/sqlid/internal/app/commands/identify"
)

// Version is the build-time version string passed through to the command.
type Version = identify.Version

// Run executes the sqlid CLI with the given version, arguments, and streams,
// returning the process exit code.
func Run(ctx context.Context, version Version, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if err := identify.Command(version, stdin, stdout, stderr).Run(ctx, args); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
