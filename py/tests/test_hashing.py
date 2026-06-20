"""Tests for the raw SQL ID and SQL hash primitives."""

from __future__ import annotations

from sqlid import hashing
from sqlid.hashing import sql_hash_raw, sql_id_raw


def test_sql_id_raw_matches_reference() -> None:
    assert sql_id_raw("select 1") == "y30pf6xwqt3x"
    assert sql_id_raw("select * from table") == "9nq4tw9gnts86"


def test_sql_hash_raw_matches_reference() -> None:
    assert sql_hash_raw("select 1") == 3150668925


def test_nul_byte_changes_result() -> None:
    # The trailing NUL byte means these two inputs are not collapsed together.
    assert sql_id_raw("a") != sql_id_raw("a\x00")


def test_base32_of_zero_is_first_alphabet_character() -> None:
    assert hashing._base32(0) == hashing.ALPHABET[0]


def test_base32_round_trips_a_known_value() -> None:
    # 32 -> "10": one full radix carry into the next digit.
    assert hashing._base32(32) == "10"
