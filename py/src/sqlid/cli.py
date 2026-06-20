"""Command-line interface for computing SQL IDs and hashes.

Each input is either a path to a file containing SQL or a literal SQL string;
when standard input is not a terminal it is read as an additional statement.
Every input is normalized (subject to the ``--keep-*``/``--no-*`` toggles) and
reported as its SQL ID and SQL hash.
"""

from __future__ import annotations

import argparse
import sys
from collections.abc import Sequence
from dataclasses import dataclass
from pathlib import Path
from typing import TextIO

from sqlid.hashing import sql_hash_raw, sql_id_raw
from sqlid.normalize import Options, normalize

_DESCRIPTION = "Calculate the SQL ID and SQL hash of each SQL statement."


@dataclass(frozen=True, slots=True)
class _Input:
    """A single statement to process, with its display name and origin."""

    name: str
    sql: str
    from_stdin: bool


@dataclass(frozen=True, slots=True)
class _Result:
    """The computed fields available to the output formatter."""

    id: str
    hash: int
    name: str
    normalized: str
    original: str


def _parser() -> argparse.ArgumentParser:
    """Build the argument parser mirroring the historical option set."""
    parser = argparse.ArgumentParser(prog="sqlid", description=_DESCRIPTION)
    parser.add_argument("inputs", nargs="*", help="SQL file paths or literal SQL")
    parser.add_argument("-i", "--id", action="store_true", help="only output the SQL ID")
    parser.add_argument("-a", "--hash", action="store_true", help="only output the SQL hash")
    parser.add_argument("-F", "--format", help="format string of i,h,n,q,s characters")
    parser.add_argument("-t", "--tabs", action="store_true", help="separate fields with tabs")
    parser.add_argument("-v", "--verbose", action="store_true", help="also output the normalized SQL")
    parser.add_argument("-N", "--no-name", action="store_true", help="omit the input name")
    parser.add_argument("-I", "--case", action="store_true", help="keep case (do not lowercase)")
    parser.add_argument("-C", "--no-uncomment", action="store_true", help="keep comments")
    parser.add_argument("-Z", "--no-compress", action="store_true", help="do not compress whitespace or comments")
    parser.add_argument("-L", "--no-newline", action="store_true", help="do not append a trailing newline")
    parser.add_argument("-W", "--keep-with", action="store_true", help="keep WITH-clause aliases")
    parser.add_argument("-R", "--keep-const", action="store_true", help="keep string and numeric literals")
    parser.add_argument("-S", "--semicolon", action="store_true", help="keep a trailing semicolon")
    parser.add_argument("-x", "--no-stdin", action="store_true", help="do not read standard input")
    parser.add_argument("-o", "--output", type=Path, help="write output to a file instead of stdout")
    return parser


def _options(args: argparse.Namespace) -> Options:
    """Translate parsed flags into a normalization :class:`Options`."""
    return Options(
        lowercase=not args.case,
        uncomment=not (args.no_uncomment or args.no_compress),
        strip_semicolon=not args.semicolon,
        compress=not args.no_compress,
        newline=not args.no_newline,
        rewrite_with=not args.keep_with,
        strip_constants=not args.keep_const,
    )


def _from_arg(arg: str, index: int) -> _Input:
    """Resolve a positional argument to a file's contents or a literal string."""
    path = Path(arg)
    if path.is_file():
        return _Input(name=str(path), sql=path.read_text(), from_stdin=False)
    return _Input(name=f"arg[{index}]", sql=arg, from_stdin=False)


def _collect(args: argparse.Namespace, stdin: TextIO) -> list[_Input]:
    """Gather all inputs from positional arguments and standard input."""
    inputs = [_from_arg(arg, index) for index, arg in enumerate(args.inputs)]
    if not args.no_stdin and not stdin.isatty():
        inputs.append(_Input(name="--", sql=stdin.read(), from_stdin=True))
    return inputs


def _result(inp: _Input, options: Options, no_name: bool) -> _Result:
    """Compute the SQL ID, hash and normalized form for one input."""
    normalized = normalize(inp.sql, options)
    return _Result(
        id=sql_id_raw(normalized),
        hash=sql_hash_raw(normalized),
        name="" if no_name else inp.name,
        normalized=normalized,
        original=inp.sql,
    )


def _format(template: str, result: _Result) -> str:
    """Render ``result`` through a user format string of field characters."""
    template = template.replace("\\n", "\n").replace("\\t", "\t")
    fields = {
        "i": result.id,
        "h": str(result.hash),
        "n": result.name,
        "q": result.normalized,
        "c": result.normalized,
        "s": result.original,
    }
    return "".join(fields.get(char, char) for char in template)


def _line(result: _Result, args: argparse.Namespace, sep: str, bare: bool) -> str:
    """Render one output line for the selected (non-format) output mode."""
    if args.id:
        return result.id if bare else f"{result.id}{sep}{result.name}"
    if args.hash:
        return str(result.hash) if bare else f"{result.hash}{sep}{result.name}"
    columns = [result.id, str(result.hash), result.name]
    if args.verbose:
        columns.append(result.normalized)
    return sep.join(columns[:2] if bare else columns)


def _render(inputs: list[_Input], args: argparse.Namespace, options: Options) -> list[str]:
    """Produce one output line per non-blank input."""
    sep = "\t" if args.tabs else " "
    bare = len(inputs) == 1 and inputs[0].from_stdin
    statements = (inp for inp in inputs if inp.sql.strip())
    results = (_result(inp, options, args.no_name) for inp in statements)
    if args.format is not None:
        return [_format(args.format, result) for result in results]
    return [_line(result, args, sep, bare) for result in results]


def main(
    argv: Sequence[str] | None = None,
    *,
    stdin: TextIO | None = None,
    stdout: TextIO | None = None,
) -> int:
    """Entry point; returns the process exit code."""
    args = _parser().parse_args(argv)
    inputs = _collect(args, stdin if stdin is not None else sys.stdin)
    lines = _render(inputs, args, _options(args))
    text = "".join(f"{line}\n" for line in lines)
    if args.output is not None:
        args.output.write_text(text)
        return 0
    (stdout if stdout is not None else sys.stdout).write(text)
    return 0
