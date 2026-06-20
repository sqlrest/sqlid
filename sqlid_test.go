package sqlid

import "testing"

// The raw contract values predate normalization and must never change.
func TestSQLIdRawContract(t *testing.T) {
	cases := map[Statement]Id{
		"select 1":            "y30pf6xwqt3x",
		"select * from table": "9nq4tw9gnts86",
	}
	for stmt, want := range cases {
		if got := SQLIdRaw(stmt); got != want {
			t.Errorf("SQLIdRaw(%q) = %q, want %q", stmt, got, want)
		}
	}
}

func TestSQLHashRawContract(t *testing.T) {
	if got := SQLHashRaw("select 1"); got != Hash(3150668925) {
		t.Errorf("SQLHashRaw(select 1) = %d, want 3150668925", got)
	}
}

func TestSQLIdAndHashNormalize(t *testing.T) {
	if got := SQLId("select 1"); got != Id("dmrrk1sbj01z") {
		t.Errorf("SQLId(select 1) = %q, want dmrrk1sbj01z", got)
	}
	if got := SQLHash("select 1"); got != Hash(1891139647) {
		t.Errorf("SQLHash(select 1) = %d, want 1891139647", got)
	}
	// Equivalent statements collapse to the same ID.
	if SQLId("SELECT   1") != SQLId("select 1") {
		t.Error("normalized SQLId differs for equivalent statements")
	}
}

func TestBase32OfZero(t *testing.T) {
	if got := base32(0); got != Id(alphabet[:1]) {
		t.Errorf("base32(0) = %q, want %q", got, alphabet[:1])
	}
}

func TestBase32Carry(t *testing.T) {
	if got := base32(radix); got != Id("10") {
		t.Errorf("base32(32) = %q, want 10", got)
	}
}
