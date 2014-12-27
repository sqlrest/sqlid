package sqlid

import "testing"

//
type T struct {
	*testing.T
}

//
func (t T) id(sql, expect string) {
	id := SQLId(sql)
	var f func(string, ...interface{})
	if id != expect {
		f = t.Fatalf
	} else {
		f = t.Logf
	}
	f("%+v = %+v", sql, SQLId(sql))
}

//
func TestSelect1(t *testing.T) {
	T{t}.id("select 1", "y30pf6xwqt3x")
}

//
func TestSelectFrom(t *testing.T) {
	T{t}.id("select * from table", "9nq4tw9gnts86")
}
