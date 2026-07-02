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
		stdin io.Reader
		name  string
		want  string
		args  []string
		cfg   Config
	}{
		{
			name:  "default literal",
			cfg:   Config{},
			args:  []string{"select 1"},
			stdin: empty(),
			want:  wantID + " " + wantHash + " arg[0]\n",
		},
		{
			name:  "stdin is bare",
			cfg:   Config{ShouldReadStdin: true},
			args:  nil,
			stdin: strings.NewReader("select 1"),
			want:  wantID + " " + wantHash + "\n",
		},
		{
			name:  "id only",
			cfg:   Config{Output: Output{IsIDOnly: true}},
			args:  []string{"select 1"},
			stdin: empty(),
			want:  wantID + " arg[0]\n",
		},
		{
			name:  "hash only",
			cfg:   Config{Output: Output{IsHashOnly: true}},
			args:  []string{"select 1"},
			stdin: empty(),
			want:  wantHash + " arg[0]\n",
		},
		{
			name:  "id only stdin bare",
			cfg:   Config{ShouldReadStdin: true, Output: Output{IsIDOnly: true}},
			args:  nil,
			stdin: strings.NewReader("select 1"),
			want:  wantID + "\n",
		},
		{
			name:  "hash only stdin bare",
			cfg:   Config{ShouldReadStdin: true, Output: Output{IsHashOnly: true}},
			args:  nil,
			stdin: strings.NewReader("select 1"),
			want:  wantHash + "\n",
		},
		{
			name:  "no name",
			cfg:   Config{Output: Output{NameDisabled: true}},
			args:  []string{"select 1"},
			stdin: empty(),
			want:  wantID + " " + wantHash + " \n",
		},
		{
			name:  "verbose",
			cfg:   Config{Output: Output{IsVerbose: true}},
			args:  []string{"select 1"},
			stdin: empty(),
			want:  wantID + " " + wantHash + " arg[0] select ? \n",
		},
		{
			name:  "tabs",
			cfg:   Config{Output: Output{TabsEnabled: true}},
			args:  []string{"select 1"},
			stdin: empty(),
			want:  wantID + "\t" + wantHash + "\targ[0]\n",
		},
		{
			name:  "format literal and fields",
			cfg:   Config{Output: Output{Format: "i:s", HasFormat: true}},
			args:  []string{"select 1"},
			stdin: empty(),
			want:  wantID + ":select 1\n",
		},
		{
			name:  "format hash name",
			cfg:   Config{Output: Output{Format: "h n", HasFormat: true}},
			args:  []string{"select 1"},
			stdin: empty(),
			want:  wantHash + " arg[0]\n",
		},
		{
			name:  "format normalized",
			cfg:   Config{Output: Output{Format: "q", HasFormat: true}},
			args:  []string{"select 1"},
			stdin: empty(),
			want:  "select ? \n",
		},
		{
			name:  "blank input skipped",
			cfg:   Config{},
			args:  []string{"   ", "select 1"},
			stdin: empty(),
			want:  wantID + " " + wantHash + " arg[1]\n",
		},
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
	out, err := Run(
		Config{ShouldKeepConst: true, Output: Output{IsIDOnly: true}},
		realFS(),
		[]string{"select 1"},
		empty(),
	)
	if err != nil {
		t.Fatalf("Run error = %v", err)
	}
	if out == wantID+" arg[0]\n" {
		t.Error("keeping constants should change the SQL ID")
	}
}

func TestRunStdinReadError(t *testing.T) {
	_, err := Run(Config{ShouldReadStdin: true}, realFS(), nil, failReader{})
	if !errors.Is(err, constants.ErrReadStdin) {
		t.Errorf("error = %v, want %v", err, constants.ErrReadStdin)
	}
}

func TestCollectPropagatesReadError(t *testing.T) {
	filesys := FileSystem{
		Stat: func(string) (fs.FileInfo, error) { return fakeInfo{}, nil },
		Read: func(string) ([]byte, error) { return nil, io.ErrUnexpectedEOF },
	}
	_, err := Config{}.collect(filesys, []string{"x"}, empty())
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
