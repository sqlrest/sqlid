package app

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sqlrest/sqlid/internal/constants"
)

const (
	wantID   = "dmrrk1sbj01z"
	wantHash = "1891139647"
)

func run(t *testing.T, args []string, stdin io.Reader) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Run(context.Background(), append([]string{"sqlid"}, args...), stdin, &out, &errb)
	return code, out.String(), errb.String()
}

func empty() io.Reader { return strings.NewReader("") }

// failReader fails on every read.
type failReader struct{}

func (failReader) Read([]byte) (int, error) { return 0, io.ErrClosedPipe }

// failWriter fails on every write.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

// TestRunDefault exercises the happy path: flag parsing, config building, the
// non-terminal stdin branch, osFS, and writing to stdout.
func TestRunDefault(t *testing.T) {
	code, out, _ := run(t, []string{"--no-stdin", "select 1"}, empty())
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if out != wantID+" "+wantHash+" arg[0]\n" {
		t.Errorf("output = %q", out)
	}
}

// TestRunReadsTerminalLikeFileStdin drives the *os.File branch of isTerminal
// with a regular file (which term.IsTerminal reports as not a terminal).
func TestRunReadsTerminalLikeFileStdin(t *testing.T) {
	path := filepath.Join(t.TempDir(), "in.sql")
	if err := os.WriteFile(path, []byte("select 1"), 0o644); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	code, out, _ := run(t, nil, file)
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if out != wantID+" "+wantHash+"\n" {
		t.Errorf("output = %q", out)
	}
}

func TestRunWritesOutputFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.txt")
	code, out, _ := run(t, []string{"-o", path, "--no-stdin", "select 1"}, empty())
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if out != "" {
		t.Errorf("stdout = %q, want empty", out)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != wantID+" "+wantHash+" arg[0]\n" {
		t.Errorf("file = %q", data)
	}
}

func TestRunWriteFileErrorExitsNonZero(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-dir", "out.txt")
	code, _, errOut := run(t, []string{"-o", missing, "--no-stdin", "select 1"}, empty())
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errOut, constants.ErrWriteFile.Error()) {
		t.Errorf("stderr = %q, want it to mention %q", errOut, constants.ErrWriteFile)
	}
}

func TestRunFlagParseErrorExitsNonZero(t *testing.T) {
	code, _, errOut := run(t, []string{"--bogus"}, empty())
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if errOut == "" {
		t.Error("expected an error message on stderr")
	}
}

func TestRunStdinReadErrorExitsNonZero(t *testing.T) {
	code, _, errOut := run(t, nil, failReader{})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errOut, constants.ErrReadStdin.Error()) {
		t.Errorf("stderr = %q, want it to mention %q", errOut, constants.ErrReadStdin)
	}
}

func TestRunStdoutWriteErrorExitsNonZero(t *testing.T) {
	var errb bytes.Buffer
	code := Run(context.Background(), []string{"sqlid", "--no-stdin", "select 1"}, empty(), failWriter{}, &errb)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
}
