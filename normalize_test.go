package sqlid

import (
	"reflect"
	"testing"
)

func TestNormalizeOptions(t *testing.T) {
	cases := []struct {
		name   string
		stmt   Statement
		option Option
		want   Statement
	}{
		{"lowercase", "SELECT 1", Lowercase(false), "SELECT ? "},
		{"uncomment", "/* c */ select 1", Uncomment(false), "/* c */ select ? "},
		{"strip-constants", "select 1", StripConstants(false), "select 1\n"},
		{"strip-semicolon", "select 1;", StripSemicolon(false), "select 1;\n"},
		{"newline", "select x", Newline(false), "select x"},
		{
			"rewrite-with",
			"with a as (select 1) select * from a",
			RewriteWith(false),
			"with a as (select 1) select * from a\n",
		},
		{"compress", "select   1  x", Compress(false), "select   ?  x\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Normalize(c.stmt, c.option); got != c.want {
				t.Errorf("Normalize(%q, %s) = %q, want %q", c.stmt, c.name, got, c.want)
			}
		})
	}
}

func TestTopLevelSegments(t *testing.T) {
	cases := []struct {
		name string
		in   Statement
		want []string
	}{
		{"flat groups", "a(b)c(d)e.", []string{"a", "c", "e"}},
		{"nested group", "a(b(c)d)e.", []string{"a", "e"}},
		{"trailing group", "(a)", []string{""}},
		{"parens in quotes", "'(a)'b.", []string{"'(a)'b"}},
		{"mixed quotes", "\"a'b\"c.", []string{"\"a'b\"c"}},
		{"unbalanced close", ")a.", []string{"a"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := topLevelSegments(c.in); !reflect.DeepEqual(got, c.want) {
				t.Errorf("topLevelSegments(%q) = %#v, want %#v", c.in, got, c.want)
			}
		})
	}
}
