# Architecture

`sqlid` follows the [gomatic/template.cli](https://github.com/gomatic/template.cli) tiered architecture (**app → domain → implementation**), adapted to two facts about this repository:

1. **`sqlid` is a library first.** The Oracle-style SQL-ID computation is the public API at the module root (`import "github.com/sqlrest/sqlid"`). That package is the *implementation tier* and is deliberately **not** hidden under `internal/`, so downstream code can depend on it. The CLI is layered on top.
2. **`sqlid` is a single command.** There is no command tree and no structured (JSON/YAML) result; the tool emits text. So the template's `internal/app` machinery (`Runner`/`Default`/`output`) and per-subcommand directories do not apply. What carries over is the *separation of tiers* and the sentinel-error convention.

## The tiers

Dependencies flow one direction only: a tier depends only on the tier to its right.

| Tier | Location | Sole responsibility | Forbidden |
| --- | --- | --- | --- |
| **app** | [`internal/app`](../internal/app) | The CLI surface: flag definitions, the `urfave/cli` command, terminal detection, and process I/O (reading stdin, writing to a file or stdout). | SQL computation, normalization, or output rendering. |
| **domain** | [`internal/domain/identify`](../internal/domain/identify) | Orchestration: a `Config` (the resolved flag values) and `Run`, which collects inputs, computes each statement's identifiers via the library, and renders the output text. | Importing `urfave/cli`; process-I/O or flag parsing. |
| **implementation** | the root [`sqlid`](..) package (public) plus [`internal/constants`](../internal/constants) | The actual work: normalization and SQL-ID/hash computation, reusable and named for the concept. | Any knowledge of the CLI. |

`internal/app` and `internal/domain` are reserved names with exactly these meanings. `internal/constants` holds the sentinel errors.

### Why the split exists

- A reader opening [`internal/app/app.go`](../internal/app/app.go) sees the *entire* CLI surface — flags, the command, and I/O — and nothing else.
- A reader opening [`internal/domain/identify/identify.go`](../internal/domain/identify/identify.go) sees *what the command does*, expressed as orchestration over injected dependencies (a `FileSystem` and an input stream), with all SQL work delegated to the library. This is what makes it testable to 100% without touching the real filesystem.
- The root `sqlid` library is reusable by any caller and is the shared source of truth — the Go CLI and the [Python implementation](../py) agree against [`testdata/parity.json`](../testdata/parity.json).

## The seam

The app tier builds an [`identify.Config`](../internal/domain/identify/identify.go) from the parsed flags (negating the `--no-*`/`--keep-*` flags into positive toggles) and decides whether to read stdin (`!--no-stdin && !isTerminal`). It then calls:

```go
func Run(cfg Config, filesys FileSystem, args []string, stdin io.Reader) (string, error)
```

`Run` returns the rendered text; the app tier writes it to `--output` or stdout. The domain never imports `urfave/cli` and never touches `os` directly — the `FileSystem` and stream are injected.

## Quality gate

Run `make check`. It must exit zero, and every package holds **100% statement coverage**, every function has **cognitive complexity ≤ 7**, errors are constant sentinels in [`internal/constants`](../internal/constants) matched with `errors.Is`, and code is `gofumpt`-clean. The `cmd/sqlid` shim is the process entry point and is covered by its own test.

## Adding a flag

1. Add the `cli.Flag` in [`internal/app/app.go`](../internal/app/app.go) `flags()`.
2. Map it into [`identify.Config`](../internal/domain/identify/identify.go) in the app tier's `config()`.
3. Consume it in the domain (`Config.options()` for a normalization toggle, or the renderers for an output mode), with tests to 100%.
4. `make check` is green.
