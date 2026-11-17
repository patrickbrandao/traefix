package fingerprint

import (
	"crypto/rand"
	"time"
)

const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// newULID returns a ULID (github.com/ulid/spec): 48 bits of milliseconds and 80 random bits,
// in 26 characters of Crockford base32, so that the IDs sort by accept time.
func newULID(t time.Time) string {
	var b [16]byte
	ms := uint64(t.UnixMilli())
	for i := 5; i >= 0; i-- {
		b[i] = byte(ms)
		ms >>= 8
	}
	_, _ = rand.Read(b[6:])

	// The 128 bits are padded with 2 leading zero bits to 130 = 26 * 5, the first character carrying 3 bits.
	var out [26]byte
	var acc uint64
	bits := 2
	j := 0
	for _, v := range b {
		acc = acc<<8 | uint64(v)
		bits += 8
		for bits >= 5 {
			bits -= 5
			out[j] = crockford[(acc>>uint(bits))&0x1f]
			j++
		}
	}

	return string(out[:])
}
