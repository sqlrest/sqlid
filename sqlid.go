package sqlid

import (
	"crypto/md5"
	"encoding/binary"
	"math"
)

const (
	alphabet = "0123456789abcdfghjkmnpqrstuvwxyz"
)

func SQLId(data ...string) string {
	h := md5.New()
	for _, d := range data {
		h.Write([]byte(d))
	}
	h.Write([]byte{0})
	sum := h.Sum(nil)
	msi, lsi := binary.LittleEndian.Uint32(sum[8:12]), binary.LittleEndian.Uint32(sum[12:16])
	sqln := uint64(msi)<<32 + uint64(lsi)
	stop := uint8(math.Log(float64(sqln))/math.Log(32) + 1)
	sqlid := make([]byte, stop, stop)
	for i, stop := uint8(0), uint8(math.Log(float64(sqln))/math.Log(32)+1); i < stop; i += 1 {
		sqlid[stop-i-1] = alphabet[(sqln/uint64(math.Pow(32, float64(i))))%32]
	}
	return string(sqlid)
}
