package main

import "testing"

// TestMainRunsCleanly exercises the process entry point with a captured exit
// code so the single delegating statement in main is covered.
func TestMainRunsCleanly(t *testing.T) {
	osArgs = []string{"sqlid", "--no-stdin", "select 1"}
	code := -1
	osExit = func(c int) { code = c }
	main()
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}
