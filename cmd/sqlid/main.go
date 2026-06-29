// Command sqlid computes Oracle-style SQL IDs and SQL hashes for SQL statements.
package main

import (
	"context"
	"os"

	"github.com/sqlrest/sqlid/internal/app"
)

// version is injected at build time via ldflags -X main.version={{.Version}}.
var version = "dev"

// Indirections over the process environment so main is exercised in tests.
var (
	osArgs = os.Args
	osExit = os.Exit
)

func main() {
	osExit(app.Run(context.Background(), version, osArgs, os.Stdin, os.Stdout, os.Stderr))
}
