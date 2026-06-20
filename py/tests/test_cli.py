"""Tests for the command-line interface."""

from __future__ import annotations

import io
import sys
from pathlib import Path

from sqlid.cli import main

_ID = "dmrrk1sbj01z"
_HASH = "1891139647"


def run(argv: list[str], stdin_text: str = "") -> tuple[int, str]:
    """Run the CLI with injected streams and return (exit code, stdout)."""
    out = io.StringIO()
    code = main(argv, stdin=io.StringIO(stdin_text), stdout=out)
    return code, out.getvalue()


def test_default_output_for_literal_argument() -> None:
    code, out = run(["--no-stdin", "select 1"])
    assert code == 0
    assert out == f"{_ID} {_HASH} arg[0]\n"


def test_id_only_for_argument() -> None:
    _, out = run(["--id", "--no-stdin", "select 1"])
    assert out == f"{_ID} arg[0]\n"


def test_hash_only_for_argument() -> None:
    _, out = run(["--hash", "--no-stdin", "select 1"])
    assert out == f"{_HASH} arg[0]\n"


def test_default_output_for_stdin_is_bare() -> None:
    _, out = run([], stdin_text="select 1")
    assert out == f"{_ID} {_HASH}\n"


def test_id_only_for_stdin_is_bare() -> None:
    _, out = run(["--id"], stdin_text="select 1")
    assert out == f"{_ID}\n"


def test_hash_only_for_stdin_is_bare() -> None:
    _, out = run(["--hash"], stdin_text="select 1")
    assert out == f"{_HASH}\n"


def test_no_name_omits_the_name() -> None:
    _, out = run(["--no-name", "--no-stdin", "select 1"])
    assert out == f"{_ID} {_HASH} \n"


def test_verbose_appends_normalized_sql() -> None:
    _, out = run(["--verbose", "--no-stdin", "select 1"])
    assert out == f"{_ID} {_HASH} arg[0] select ? \n"


def test_tabs_separator() -> None:
    _, out = run(["--tabs", "--no-stdin", "select 1"])
    assert out == f"{_ID}\t{_HASH}\targ[0]\n"


def test_format_string_with_literal_and_fields() -> None:
    _, out = run(["--format", "i:q", "--no-stdin", "select 1"])
    assert out == f"{_ID}:select ? \n"


def test_format_string_emits_original_sql() -> None:
    _, out = run(["--format", "s", "--no-stdin", "select 1"])
    assert out == "select 1\n"


def test_blank_inputs_are_skipped() -> None:
    _, out = run(["--no-stdin", "   ", "select 1"])
    assert out == f"{_ID} {_HASH} arg[1]\n"


def test_file_input_is_read(tmp_path: Path) -> None:
    query = tmp_path / "q.sql"
    query.write_text("select 1")
    _, out = run([str(query), "--no-stdin"])
    assert out == f"{_ID} {_HASH} {query}\n"


def test_output_file_receives_result(tmp_path: Path) -> None:
    target = tmp_path / "out.txt"
    code, out = run(["-o", str(target), "--no-stdin", "select 1"])
    assert code == 0
    assert out == ""
    assert target.read_text() == f"{_ID} {_HASH} arg[0]\n"


def test_keep_const_flag_changes_normalization() -> None:
    _, out = run(["--keep-const", "--id", "--no-stdin", "select 1"])
    # With literals kept, "select 1\n" hashes differently from the default.
    assert out != f"{_ID} arg[0]\n"


def test_defaults_fall_back_to_standard_streams(monkeypatch) -> None:
    captured = io.StringIO()
    monkeypatch.setattr(sys, "stdin", io.StringIO("ignored"))
    monkeypatch.setattr(sys, "stdout", captured)
    code = main(["--no-stdin", "select 1"])
    assert code == 0
    assert captured.getvalue() == f"{_ID} {_HASH} arg[0]\n"
