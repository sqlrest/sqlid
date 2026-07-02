package app

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/sqlrest/sqlid/internal/app/commands/identify"
)

func run(t *testing.T, args []string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Run(
		context.Background(),
		"dev",
		append([]string{identify.Name}, args...),
		strings.NewReader(""),
		&out,
		&errb,
	)
	return code, out.String(), errb.String()
}

// TestRunExitsZeroOnSuccess asserts a successful command run maps to exit
// code 0 with the rendered output on stdout.
func TestRunExitsZeroOnSuccess(t *testing.T) {
	code, out, errOut := run(t, []string{"--no-stdin", "select 1"})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if out != "dmrrk1sbj01z 1891139647 arg[0]\n" {
		t.Errorf("output = %q", out)
	}
	if errOut != "" {
		t.Errorf("stderr = %q, want empty", errOut)
	}
}

// TestRunExitsOneOnError asserts a command error maps to exit code 1 with the
// error printed to stderr.
func TestRunExitsOneOnError(t *testing.T) {
	code, _, errOut := run(t, []string{"--bogus"})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if errOut == "" {
		t.Error("expected an error message on stderr")
	}
}

// TestRunStdoutWriteErrorExitsNonZero drives the error path through the
// command's stdout writer.
func TestRunStdoutWriteErrorExitsNonZero(t *testing.T) {
	var errb bytes.Buffer
	code := Run(
		context.Background(),
		"dev",
		[]string{identify.Name, "--no-stdin", "select 1"},
		strings.NewReader(""),
		failWriter{},
		&errb,
	)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
}

// failWriter fails on every write.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
