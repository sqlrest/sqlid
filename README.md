# sqlid

[![CI](https://github.com/sqlrest/sqlid/actions/workflows/ci.yml/badge.svg)](https://github.com/sqlrest/sqlid/actions/workflows/ci.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/sqlrest/sqlid)](https://goreportcard.com/report/github.com/sqlrest/sqlid) [![Release](https://img.shields.io/github/v/release/sqlrest/sqlid)](https://github.com/sqlrest/sqlid/releases/latest)

Compute an Oracle-style **SQL ID** that identifies the *same* SQL statement across processes — regardless of `WITH`-clause aliases and literal constants.

The algorithm mirrors Oracle's `SQL_ID`: MD5 the statement text (with a trailing NUL byte), read the last 8 bytes of the digest as a 64-bit little-endian integer, then base-32 encode it with Oracle's alphabet (`0123456789abcdfghjkmnpqrstuvwxyz`). It makes no attempt to reproduce any specific database's `SQL_ID` value; its purpose is deterministic, normalization-aware identification — two statements that differ only in CTE alias names, literal values, comments, case, or whitespace collapse to the same ID.

This repository ships two implementations that produce identical results: the Go module described below, and the modern Python 3 package in [py/](py/). Their agreement is enforced by a shared golden corpus in [testdata/parity.json](testdata/parity.json).

## Install

```sh
go get github.com/sqlrest/sqlid
```

Or build the CLI from source:

```sh
make build   # produces bin/sqlid
```

## Library

```go
import "github.com/sqlrest/sqlid"

id := sqlid.SQLID("select 1")                 // normalized SQL ID
h := sqlid.SQLHash("select 1")                // normalized SQL hash
n := sqlid.Normalize("SELECT  1 ;")           // "select ? "

raw := sqlid.SQLRawID("select 1")             // ID of the exact text, no normalization
custom := sqlid.SQLID("select 1", sqlid.StripConstants(false))
```

`SQLID` and `SQLHash` normalize the statement first; `SQLRawID` and `SQLRawHash` operate on the exact text given. Normalization is controlled by functional options ([options.go](options.go)): `Lowercase`, `Uncomment`, `StripSemicolon`, `Compress`, `Newline`, `RewriteWith` and `StripConstants`, each enabled by default.

## CLI

```sh
sqlid 'select 1' 'select * from table'
echo 'select 1' | sqlid --id
sqlid --format 'i s\n' query.sql
```

Each argument is a file path or a literal SQL string; standard input is read as an additional statement when it is not a terminal. Run `sqlid --help` for the full option list.

## Develop

The full quality gate runs with:

```sh
make check
```

It enforces `gofumpt` formatting, `go vet`, `staticcheck`, `govulncheck`, a cognitive-complexity ceiling of 7 ([gocognit](https://github.com/uudashr/gocognit)), 100% statement coverage, and a valid [GoReleaser](https://goreleaser.com) configuration. Run `make help` to list every target.

### Layout

The code follows the [gomatic/template.cli](https://github.com/gomatic/template.cli) tiered layout (**app → domain → implementation**); see [docs/architecture.md](docs/architecture.md). The public library at the module root is the implementation tier and is intentionally importable, so the CLI is layered on top rather than the library being hidden under `internal/`.

```text
.                          the public sqlid library (implementation tier)
cmd/sqlid                  entry point (CLI wiring)
internal/app               urfave/cli wiring: flags, command, process I/O
internal/domain/identify   command orchestration: collect, compute, render (domain tier)
internal/constants         sentinel errors
py/                        the parity-checked Python implementation
```

The Python implementation has its own gate; see [py/README.md](py/README.md).
