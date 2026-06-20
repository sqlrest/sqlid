"""SQL normalization.

Normalization rewrites a statement so that semantically equivalent queries
collapse to the same text before hashing: case is folded, comments and trailing
semicolons are removed, whitespace is compressed, ``WITH``-clause aliases are
replaced by positional tokens, and string and numeric literals are replaced by
``?`` placeholders. Hint comments (``/*+ ... */``) are deliberately preserved.
"""

from __future__ import annotations

import re
from collections.abc import Callable
from dataclasses import dataclass

# C-style comments that are not optimizer hints (``/*+ ... */``).
_COMMENT_RE = re.compile(r"/[*][^+](?:(?![*]/).)*[*]/")
# Any run of whitespace.
_WHITESPACE_RE = re.compile(r"[ \t\n\r\v\f]+")
# A trailing semicolon and any whitespace after it.
_SEMICOLON_RE = re.compile(r";[ \t\n\r\v\f]*$")
# A ``WITH``-clause alias: ``with x as`` or ``, x as`` anchored to the segment.
_WITH_RE = re.compile(r"^(?:with\s+|,\s*)([^\s.]+)\s+as", re.IGNORECASE)
# A single-quoted string literal, tolerating embedded doubled single quotes.
_STRING_RE = re.compile(r"'((?:[^']+|'')*)('?)(?!')")
# A whitespace-delimited integer literal.
_NUMBER_RE = re.compile(r"\s\d+\s")


@dataclass(frozen=True, slots=True)
class Options:
    """Toggles for each normalization step; every step is on by default."""

    lowercase: bool = True
    uncomment: bool = True
    strip_semicolon: bool = True
    compress: bool = True
    newline: bool = True
    rewrite_with: bool = True
    strip_constants: bool = True


def _nesting(stmt: str) -> list[object]:
    """Split ``stmt`` into top-level text segments and parenthesized sub-lists.

    Quoted regions are treated as opaque so their parentheses do not nest. The
    final trailing character (the appended newline) is dropped, matching the
    historical behavior the WITH-clause rewrite depends on.
    """
    root: list[object] = []
    stack: list[list[object]] = [root]
    quote: str | None = None
    start = 0
    for index, char in enumerate(stmt):
        if char in ("'", '"'):
            quote = None if char == quote else (quote or char)
        if quote is not None:
            continue
        if char in ("(", ")"):
            stack[-1].append(stmt[start:index])
            _descend(stack, char)
            start = index + 1
    if start != len(stmt):
        root.append(stmt[start : len(stmt) - 1])
    return root


def _descend(stack: list[list[object]], char: str) -> None:
    """Push a new group on ``(`` or pop and attach the current group on ``)``."""
    if char == "(":
        stack.append([])
        return
    node = stack.pop()
    stack[-1].append(node)


def _rewrite_with(stmt: str) -> str:
    """Replace each top-level ``WITH`` alias with a positional ``^NNNN^`` token."""
    if not _WITH_RE.search(stmt):
        return stmt
    sequence = 1
    for segment in _nesting(stmt):
        if isinstance(segment, list):
            continue
        for match in _WITH_RE.finditer(segment):
            stmt = stmt.replace(f"{match.group(1)} ", f"^{sequence:04X}^ ")
            sequence += 1
    return stmt


def _strip_constants(stmt: str) -> str:
    """Replace string and numeric literals with ``?`` placeholders."""
    return _NUMBER_RE.sub(" ? ", _STRING_RE.sub("?", stmt))


def _steps(options: Options) -> list[tuple[bool, Callable[[str], str]]]:
    """Return the ordered (enabled, transform) pipeline for ``options``."""
    return [
        (options.lowercase, str.lower),
        (options.uncomment, lambda s: _COMMENT_RE.sub("", s)),
        (options.strip_semicolon, lambda s: _SEMICOLON_RE.sub("", s)),
        (options.compress, lambda s: _WHITESPACE_RE.sub(" ", s).strip()),
        (options.newline, lambda s: s + "\n"),
        (options.rewrite_with, _rewrite_with),
        (options.strip_constants, _strip_constants),
    ]


def normalize(stmt: str, options: Options = Options()) -> str:
    """Apply the enabled normalization steps to ``stmt`` in order."""
    for enabled, transform in _steps(options):
        if enabled:
            stmt = transform(stmt)
    return stmt
