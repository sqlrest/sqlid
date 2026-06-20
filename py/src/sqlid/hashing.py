"""Core SQL ID and SQL hash computation.

The algorithm mirrors Oracle's ``SQL_ID``: MD5 the statement text (with a
trailing NUL byte), interpret the last 8 bytes of the digest as a 64-bit
little-endian integer, then base-32 encode it with Oracle's alphabet. It makes
no attempt to reproduce a specific database's ``SQL_ID`` value; its purpose is
to deterministically identify the *same* statement across processes.
"""

from __future__ import annotations

import hashlib
import math
import struct

# Oracle's base-32 alphabet for SQL_ID: digits plus lowercase letters with the
# vowel-like characters ``e``, ``i``, ``l`` and ``o`` removed.
ALPHABET = "0123456789abcdfghjkmnpqrstuvwxyz"

_RADIX = len(ALPHABET)
_DIGEST_LAYOUT = "<IIII"


def _digest(stmt: str) -> bytes:
    """Return the MD5 digest of ``stmt`` with the trailing NUL byte applied."""
    return hashlib.md5((stmt + "\x00").encode("utf-8")).digest()


def _words(stmt: str) -> tuple[int, int, int, int]:
    """Return the four little-endian 32-bit words of the statement's digest."""
    return struct.unpack(_DIGEST_LAYOUT, _digest(stmt))


def _base32(value: int) -> str:
    """Base-32 encode ``value`` using :data:`ALPHABET`, most significant first."""
    if value == 0:
        return ALPHABET[0]
    width = int(math.log(value) / math.log(_RADIX) + 1)
    digits = (ALPHABET[(value // _RADIX**i) % _RADIX] for i in range(width))
    return "".join(reversed(list(digits)))


def sql_id_raw(stmt: str) -> str:
    """Return the SQL ID of ``stmt`` exactly as given, without normalization."""
    _, _, most, least = _words(stmt)
    return _base32((most << 32) | least)


def sql_hash_raw(stmt: str) -> int:
    """Return the SQL hash of ``stmt`` exactly as given, without normalization."""
    return _words(stmt)[3]
