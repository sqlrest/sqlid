"""Tests for SQL normalization, including the WITH-clause rewrite parser."""

from __future__ import annotations

from sqlid import Options, normalize, sql_hash, sql_id
from sqlid.normalize import _nesting


def test_default_normalization_reference_values() -> None:
    assert normalize("select 1") == "select ? "
    assert normalize("select * from table") == "select * from table\n"
    assert normalize("SELECT  1 ;") == "select ? "
    assert normalize("/* c */ select 1") == "select ? "
    assert normalize("select 'x' from t where id = 5") == "select ? from t where id = ? "


def test_with_clause_aliases_become_positional_tokens() -> None:
    statement = "with a as (select 1), b as (select 2) select * from a,b"
    expected = "with ^0001^ as (select 1), ^0002^ as (select 2) select * from a,b\n"
    assert normalize(statement) == expected


def test_hint_comment_is_preserved() -> None:
    assert normalize("/*+ index(t) */ select 1") == "/*+ index(t) */ select ? "


def test_lowercase_can_be_disabled() -> None:
    assert normalize("SELECT 1", Options(lowercase=False)) == "SELECT ? "


def test_comments_can_be_kept() -> None:
    assert normalize("/* c */ select 1", Options(uncomment=False)) == "/* c */ select ? "


def test_constants_can_be_kept() -> None:
    assert normalize("select 1", Options(strip_constants=False)) == "select 1\n"


def test_semicolon_can_be_kept() -> None:
    assert normalize("select 1;", Options(strip_semicolon=False)) == "select 1;\n"


def test_newline_can_be_disabled() -> None:
    assert normalize("select x", Options(newline=False)) == "select x"


def test_with_rewrite_can_be_disabled() -> None:
    statement = "with a as (select 1) select * from a"
    assert "^0001^" not in normalize(statement, Options(rewrite_with=False))


def test_sql_id_and_hash_use_normalization() -> None:
    assert sql_id("select 1") == "dmrrk1sbj01z"
    assert sql_hash("select 1") == 1891139647
    # Equivalent statements collapse to the same ID.
    assert sql_id("SELECT   1") == sql_id("select 1")


def test_nesting_splits_text_and_groups() -> None:
    assert _nesting("a(b)c.") == ["a", ["b"], "c"]


def test_nesting_skips_trailing_group_segment() -> None:
    # A statement ending in ')' leaves no trailing text segment to append.
    assert _nesting("(a)") == ["", ["a"]]


def test_nesting_ignores_parentheses_inside_quotes() -> None:
    assert _nesting("'(a)'.") == ["'(a)'"]


def test_nesting_handles_mixed_quotes() -> None:
    # A single quote inside a double-quoted region does not end the region.
    assert _nesting("\"a'b\"x") == ["\"a'b\"x"[:-1]]
