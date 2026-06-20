package constants_test

import (
	"errors"
	"io"
	"testing"

	"github.com/sqlrest/sqlid/internal/constants"
)

func TestErrorMessage(t *testing.T) {
	if got := constants.ErrWriteFile.Error(); got != "writing output file failed" {
		t.Errorf("Error() = %q", got)
	}
}

func TestErrorWithAppendsArgsWithoutCause(t *testing.T) {
	err := constants.ErrReadFile.With(nil, "q.sql")
	if !errors.Is(err, constants.ErrReadFile) {
		t.Errorf("error %v is not %v", err, constants.ErrReadFile)
	}
	if got := err.Error(); got != "reading input file failed: q.sql" {
		t.Errorf("Error() = %q", got)
	}
}

func TestErrorWithWrapsCause(t *testing.T) {
	err := constants.ErrReadStdin.With(io.EOF)
	if !errors.Is(err, constants.ErrReadStdin) {
		t.Errorf("error %v is not %v", err, constants.ErrReadStdin)
	}
	if !errors.Is(err, io.EOF) {
		t.Errorf("error %v does not wrap io.EOF", err)
	}
}

func TestErrorWithWrapsCauseAndArgs(t *testing.T) {
	err := constants.ErrWriteFile.With(io.EOF, "out.txt")
	if !errors.Is(err, constants.ErrWriteFile) {
		t.Errorf("error %v is not %v", err, constants.ErrWriteFile)
	}
	if !errors.Is(err, io.EOF) {
		t.Errorf("error %v does not wrap io.EOF", err)
	}
	if got := err.Error(); got != "writing output file failed: EOF: out.txt" {
		t.Errorf("Error() = %q", got)
	}
}
