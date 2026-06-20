package identify

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sqlrest/sqlid/internal/constants"
)

const (
	wantID   = "dmrrk1sbj01z"
	wantHash = "1891139647"
)

func realFS() FileSystem { return FileSystem{Stat: os.Stat, Read: os.ReadFile} }

func empty() io.Reader { return strings.NewReader("") }

func TestRunOutputModes(t *testing.T) {
	cases := []struct {
		name  string
		cfg   Config
		args  []string
		stdin io.Reader
		want  string
	}{
		{"default literal", Config{}, []string{"select 1"}, empty(), wantID + " " + wantHash + " arg[0]\n"},
		{"stdin is bare", Config{UseStdin: true}, nil, strings.NewReader("select 1"), wantID + " " + wantHash + "\n"},
		{"id only", Config{Output: Output{IDOnly: true}}, []string{"select 1"}, empty(), wantID + " arg[0]\n"},
		{"hash only", Config{Output: Output{HashOnly: true}}, []string{"select 1"}, empty(), wantHash + " arg[0]\n"},
		{"id only stdin bare", Config{UseStdin: true, Output: Output{IDOnly: true}}, nil, strings.NewReader("select 1"), wantID + "\n"},
		{"hash only stdin bare", Config{UseStdin: true, Output: Output{HashOnly: true}}, nil, strings.NewReader("select 1"), wantHash + "\n"},
		{"no name", Config{Output: Output{NoName: true}}, []string{"select 1"}, empty(), wantID + " " + wantHash + " \n"},
		{"verbose", Config{Output: Output{Verbose: true}}, []string{"select 1"}, empty(), wantID + " " + wantHash + " arg[0] select ? \n"},
		{"tabs", Config{Output: Output{Tabs: true}}, []string{"select 1"}, empty(), wantID + "\t" + wantHash + "\targ[0]\n"},
		{"format literal and fields", Config{Output: Output{Format: "i:s", HasFormat: true}}, []string{"select 1"}, empty(), wantID + ":select 1\n"},
		{"format hash name", Config{Output: Output{Format: "h n", HasFormat: true}}, []string{"select 1"}, empty(), wantHash + " arg[0]\n"},
		{"format normalized", Config{Output: Output{Format: "q", HasFormat: true}}, []string{"select 1"}, empty(), "select ? \n"},
		{"blank input skipped", Config{}, []string{"   ", "select 1"}, empty(), wantID + " " + wantHash + " arg[1]\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := Run(c.cfg, realFS(), c.args, c.stdin)
			if err != nil {
				t.Fatalf("Run error = %v", err)
			}
			if out != c.want {
				t.Errorf("output = %q, want %q", out, c.want)
			}
		})
	}
}

func TestRunReadsFileInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "q.sql")
	if err := os.WriteFile(path, []byte("select 1"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := Run(Config{}, realFS(), []string{path}, empty())
	if err != nil {
		t.Fatalf("Run error = %v", err)
	}
	if out != wantID+" "+wantHash+" "+path+"\n" {
		t.Errorf("output = %q", out)
	}
}

func TestRunTreatsDirectoryAsLiteral(t *testing.T) {
	dir := t.TempDir()
	out, err := Run(Config{}, realFS(), []string{dir}, empty())
	if err != nil {
		t.Fatalf("Run error = %v", err)
	}
	if !strings.HasSuffix(out, " arg[0]\n") {
		t.Errorf("output = %q, want a literal arg name", out)
	}
}

func TestRunKeepConstChangesResult(t *testing.T) {
	out, err := Run(Config{KeepConst: true, Output: Output{IDOnly: true}}, realFS(), []string{"select 1"}, empty())
	if err != nil {
		t.Fatalf("Run error = %v", err)
	}
	if out == wantID+" arg[0]\n" {
		t.Error("keeping constants should change the SQL ID")
	}
}

func TestRunStdinReadError(t *testing.T) {
	_, err := Run(Config{UseStdin: true}, realFS(), nil, failReader{})
	if !errors.Is(err, constants.ErrReadStdin) {
		t.Errorf("error = %v, want %v", err, constants.ErrReadStdin)
	}
}

func TestCollectPropagatesReadError(t *testing.T) {
	filesys := FileSystem{
		Stat: func(string) (fs.FileInfo, error) { return fakeInfo{}, nil },
		Read: func(string) ([]byte, error) { return nil, io.ErrUnexpectedEOF },
	}
	_, err := collect(filesys, []string{"x"}, empty(), false)
	if !errors.Is(err, constants.ErrReadFile) {
		t.Errorf("error = %v, want %v", err, constants.ErrReadFile)
	}
}

func TestFromArgReadErrorIsWrapped(t *testing.T) {
	filesys := FileSystem{
		Stat: func(string) (fs.FileInfo, error) { return fakeInfo{}, nil },
		Read: func(string) ([]byte, error) { return nil, io.ErrUnexpectedEOF },
	}
	_, err := fromArg(filesys, "x", 0)
	if !errors.Is(err, constants.ErrReadFile) {
		t.Errorf("error = %v, want %v", err, constants.ErrReadFile)
	}
}

// failReader fails on every read.
type failReader struct{}

func (failReader) Read([]byte) (int, error) { return 0, io.ErrClosedPipe }

// fakeInfo is a non-directory fs.FileInfo for filesystem injection.
type fakeInfo struct{}

func (fakeInfo) Name() string       { return "x" }
func (fakeInfo) Size() int64        { return 0 }
func (fakeInfo) Mode() fs.FileMode  { return 0 }
func (fakeInfo) ModTime() time.Time { return time.Time{} }
func (fakeInfo) IsDir() bool        { return false }
func (fakeInfo) Sys() any           { return nil }
