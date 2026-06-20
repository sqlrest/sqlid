"""Compute Oracle-style SQL IDs that identify equivalent SQL statements.

The public API offers normalization-aware :func:`sql_id` and :func:`sql_hash`
helpers plus the raw, no-normalization primitives they build on.
"""

from __future__ import annotations

from sqlid.hashing import ALPHABET, sql_hash_raw, sql_id_raw
from sqlid.normalize import Options, normalize

__all__ = [
    "ALPHABET",
    "Options",
    "normalize",
    "sql_hash",
    "sql_hash_raw",
    "sql_id",
    "sql_id_raw",
]


def sql_id(stmt: str, options: Options = Options()) -> str:
    """Return the SQL ID of ``stmt`` after normalizing it with ``options``."""
    return sql_id_raw(normalize(stmt, options))


def sql_hash(stmt: str, options: Options = Options()) -> int:
    """Return the SQL hash of ``stmt`` after normalizing it with ``options``."""
    return sql_hash_raw(normalize(stmt, options))
