# sqlid — agent instructions

`sqlid` computes Oracle-style SQL IDs and hashes that identify the *same* SQL statement across processes regardless of `WITH`-clause aliases and literal constants. It is a **library first** (the public `github.com/sqlrest/sqlid` package at the module root) with a thin CLI on top, and it ships a parity-checked Python implementation in [`py/`](../py).

It follows the [gomatic/template.cli](https://github.com/gomatic/template.cli) tiered architecture, adapted to a single-command, text-output, library-first tool. **Read [`docs/architecture.md`](../docs/architecture.md) before changing anything.**

## The tiers (app → domain → implementation)

Dependencies flow one direction only.

1. **app** — [`internal/app`](../internal/app). CLI only: `urfave/cli` flags and command, terminal detection, and process I/O (stdin, file/stdout). **No SQL or rendering logic.** It builds an `identify.Config` from the flags and calls the domain.
2. **domain** — [`internal/domain/identify`](../internal/domain/identify). Orchestration only: a `Config` and `Run(cfg, filesys, args, stdin) (string, error)` that collects inputs, computes identifiers via the library, and renders text. **Never import `urfave/cli`; never touch `os` directly** — the `FileSystem` and input stream are injected.
3. **implementation** — the public root [`sqlid`](..) package plus [`internal/constants`](../internal/constants). The reusable SQL work; **no knowledge of the CLI.**

## Hard rules

- **The root `sqlid` package is a public API — keep it importable and at the module root. Do not move it under `internal/`.** It is the implementation tier and the shared source of truth for the Go and Python implementations (parity is enforced by [`testdata/parity.json`](../testdata/parity.json)).
- Keep `internal/app` purely CLI + I/O; keep the domain free of `urfave/cli` and `os`; put reusable work in the library.
- Constant sentinel errors in [`internal/constants`](../internal/constants); match with `errors.Is`, never by string. Wrap with `.With`.
- Tests use the standard `testing` package only — **do not add `testify`** or other assertion libraries.
- 100% statement coverage per package; cognitive complexity ≤ 7 per function; `gofumpt`-clean.
- `make check` must pass before a change is complete. Build tooling (`Makefile`, `scripts/`, `go.mod` tool stanza, `.goreleaser.yaml`, CI) is maintained separately — do not change it as part of a feature change.
- Any change to normalization or hashing must keep the Go and Python implementations in parity; update [`testdata/parity.json`](../testdata/parity.json) deliberately.

To add a flag, follow the checklist at the end of [`docs/architecture.md`](../docs/architecture.md).
