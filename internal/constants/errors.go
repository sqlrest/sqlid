// Package constants holds the package-wide sentinel errors for sqlid's CLI.
//
// Every error the command-line tool can emit is declared here as a constant of
// the immutable Error type, so callers test for them with errors.Is rather than
// by comparing strings.
package constants

// Keep these constants sorted alphabetically.
const (
	// ErrReadFile is returned when an input file cannot be read.
	ErrReadFile Error = "reading input file failed"
	// ErrReadStdin is returned when standard input cannot be read.
	ErrReadStdin Error = "reading standard input failed"
	// ErrWriteFile is returned when the output file cannot be written.
	ErrWriteFile Error = "writing output file failed"
)
