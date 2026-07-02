package identify

import (
	"bytes"
	"context"
	"errors"
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

func run(t *testing.T, args []string, stdin io.Reader) (string, error) {
	t.Helper()
	var out, errb bytes.Buffer
	err := Command("dev", stdin, &out, &errb).Run(context.Background(), append([]string{Name}, args...))
	return out.String(), err
}

func empty() io.Reader { return strings.NewReader("") }

// failReader fails on every read.
type failReader struct{}

func (failReader) Read([]byte) (int, error) { return 0, io.ErrClosedPipe }

// failWriter fails on every write.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

// TestCommandVersion asserts --version prints "sqlid version <version>" with
// the build-time version string and succeeds.
func TestCommandVersion(t *testing.T) {
	var out, errb bytes.Buffer
	err := Command("9.9.9", empty(), &out, &errb).Run(context.Background(), []string{Name, "--version"})
	if err != nil {
		t.Fatalf("Run error = %v", err)
	}
	if !strings.Contains(out.String(), "sqlid version 9.9.9") {
		t.Errorf("version output = %q, want it to contain %q", out.String(), "sqlid version 9.9.9")
	}
}

// TestCommandDefault exercises the happy path: flag parsing, config building,
// the non-terminal stdin branch, osFS, and writing to stdout.
func TestCommandDefault(t *testing.T) {
	out, err := run(t, []string{"--no-stdin", "select 1"}, empty())
	if err != nil {
		t.Fatalf("Run error = %v", err)
	}
	if out != wantID+" "+wantHash+" arg[0]\n" {
		t.Errorf("output = %q", out)
	}
}

// TestCommandReadsTerminalLikeFileStdin drives the *os.File branch of
// isTerminal with a regular file (which term.IsTerminal reports as not a
// terminal).
func TestCommandReadsTerminalLikeFileStdin(t *testing.T) {
	path := filepath.Join(t.TempDir(), "in.sql")
	if err := os.WriteFile(path, []byte("select 1"), 0o644); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	out, runErr := run(t, nil, file)
	if runErr != nil {
		t.Fatalf("Run error = %v", runErr)
	}
	if out != wantID+" "+wantHash+"\n" {
		t.Errorf("output = %q", out)
	}
}

func TestCommandWritesOutputFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.txt")
	out, err := run(t, []string{"-o", path, "--no-stdin", "select 1"}, empty())
	if err != nil {
		t.Fatalf("Run error = %v", err)
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

func TestCommandWriteFileErrorIsWrapped(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-dir", "out.txt")
	_, err := run(t, []string{"-o", missing, "--no-stdin", "select 1"}, empty())
	if !errors.Is(err, constants.ErrWriteFile) {
		t.Errorf("error = %v, want %v", err, constants.ErrWriteFile)
	}
}

func TestCommandStdinReadErrorIsWrapped(t *testing.T) {
	_, err := run(t, nil, failReader{})
	if !errors.Is(err, constants.ErrReadStdin) {
		t.Errorf("error = %v, want %v", err, constants.ErrReadStdin)
	}
}

func TestCommandStdoutWriteErrorPropagates(t *testing.T) {
	var errb bytes.Buffer
	err := Command("dev", empty(), failWriter{}, &errb).
		Run(context.Background(), []string{Name, "--no-stdin", "select 1"})
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Errorf("error = %v, want %v", err, io.ErrClosedPipe)
	}
}
