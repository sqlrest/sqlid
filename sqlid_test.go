package sqlid

import "testing"

// The raw contract values predate normalization and must never change.
func TestSQLRawIDContract(t *testing.T) {
	cases := map[Statement]ID{
		"select 1":            "y30pf6xwqt3x",
		"select * from table": "9nq4tw9gnts86",
	}
	for stmt, want := range cases {
		if got := SQLRawID(stmt); got != want {
			t.Errorf("SQLRawID(%q) = %q, want %q", stmt, got, want)
		}
	}
}

func TestSQLRawHashContract(t *testing.T) {
	if got := SQLRawHash("select 1"); got != Hash(3150668925) {
		t.Errorf("SQLRawHash(select 1) = %d, want 3150668925", got)
	}
}

func TestSQLIDAndHashNormalize(t *testing.T) {
	if got := SQLID("select 1"); got != ID("dmrrk1sbj01z") {
		t.Errorf("SQLID(select 1) = %q, want dmrrk1sbj01z", got)
	}
	if got := SQLHash("select 1"); got != Hash(1891139647) {
		t.Errorf("SQLHash(select 1) = %d, want 1891139647", got)
	}
	// Equivalent statements collapse to the same ID.
	if SQLID("SELECT   1") != SQLID("select 1") {
		t.Error("normalized SQLID differs for equivalent statements")
	}
}

func TestBase32OfZero(t *testing.T) {
	if got := base32(0); got != ID(alphabet[:1]) {
		t.Errorf("base32(0) = %q, want %q", got, alphabet[:1])
	}
}

func TestBase32Carry(t *testing.T) {
	if got := base32(radix); got != ID("10") {
		t.Errorf("base32(32) = %q, want 10", got)
	}
}
