// Package sqlid computes an Oracle-style SQL ID that identifies the same SQL
// statement across processes, regardless of WITH-clause aliases and literal
// constants.
//
// The algorithm mirrors Oracle's SQL_ID: MD5 the statement text (with a
// trailing NUL byte), read the last 8 bytes of the digest as a 64-bit
// little-endian integer, then base-32 encode it with Oracle's alphabet. It
// makes no attempt to reproduce a specific database's SQL_ID value.
package sqlid

import (
	"crypto/md5"
	"encoding/binary"
	"math"
)

// Statement is SQL statement text to be identified.
type Statement string

// Id is a base-32 SQL identifier.
type Id string

// Hash is the 32-bit SQL hash derived from a statement's digest.
type Hash uint32

// alphabet is Oracle's base-32 SQL_ID alphabet: digits plus lowercase letters
// with the vowel-like characters e, i, l and o removed.
const alphabet = "0123456789abcdfghjkmnpqrstuvwxyz"

// radix is the size of alphabet.
const radix = 32

// words returns the third and fourth little-endian 32-bit words of the
// statement's MD5 digest (after appending the trailing NUL byte).
func (s Statement) words() (most uint32, least uint32) {
	sum := md5.Sum(append([]byte(s), 0))
	return binary.LittleEndian.Uint32(sum[8:12]), binary.LittleEndian.Uint32(sum[12:16])
}

// base32 encodes value with alphabet, most significant digit first.
func base32(value uint64) Id {
	if value == 0 {
		return Id(alphabet[:1])
	}
	width := int(math.Log(float64(value))/math.Log(radix) + 1)
	out := make([]byte, width)
	power := uint64(1)
	for i := range width {
		out[width-1-i] = alphabet[(value/power)%radix]
		power *= radix
	}
	return Id(out)
}

// SQLIdRaw returns the SQL ID of the statement exactly as given, without
// normalization.
func SQLIdRaw(s Statement) Id {
	most, least := s.words()
	return base32(uint64(most)<<32 | uint64(least))
}

// SQLHashRaw returns the SQL hash of the statement exactly as given, without
// normalization.
func SQLHashRaw(s Statement) Hash {
	_, least := s.words()
	return Hash(least)
}

// SQLId normalizes the statement with the given options and returns its SQL ID.
func SQLId(s Statement, options ...Option) Id {
	return SQLIdRaw(Normalize(s, options...))
}

// SQLHash normalizes the statement with the given options and returns its SQL
// hash.
func SQLHash(s Statement, options ...Option) Hash {
	return SQLHashRaw(Normalize(s, options...))
}
