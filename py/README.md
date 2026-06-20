# sqlid (Python)

Compute an Oracle-style **SQL ID** that identifies the *same* SQL statement across processes — regardless of `WITH`-clause aliases and literal constants.

The algorithm mirrors Oracle's `SQL_ID`: MD5 the statement (with a trailing NUL byte), read the last 8 bytes of the digest as a 64-bit little-endian integer, and base-32 encode it with Oracle's alphabet (`0123456789abcdfghjkmnpqrstuvwxyz`). It does not attempt to reproduce any specific database's `SQL_ID` value; its purpose is deterministic, normalization-aware identification.

This package is the modern Python 3.14 port of the original Python 2 implementation; it targets the latest stable CPython and runs in a [uv](https://docs.astral.sh/uv/)-managed virtual environment (`.venv`), never the system interpreter. The pinned version lives in [.python-version](.python-version). The companion Go implementation lives in the [repository root](../).

## Install

This project uses [uv](https://docs.astral.sh/uv/) for environment and dependency management.

```sh
uv sync
```

## Library

```python
from sqlid import sql_id, sql_hash, normalize, Options

sql_id("select 1")                     # normalized SQL ID
sql_hash("select 1")                   # normalized SQL hash (int)
normalize("SELECT  1 ;")               # "select ?\n"
sql_id("select 1", Options(strip_constants=False))
```

`sql_id` and `sql_hash` normalize first; `sql_id_raw` and `sql_hash_raw` operate on the exact text given.

## CLI

```sh
uv run sqlid 'select 1' 'select * from table'
echo 'select 1' | uv run sqlid --id
uv run sqlid --format 'i s\n' query.sql
```

Run `uv run sqlid --help` for the full option list. Each input is a file path or a literal SQL string; standard input is read as an additional statement when it is not a terminal.

## Test

```sh
uv run pytest
```

Tests enforce 100% statement coverage.
