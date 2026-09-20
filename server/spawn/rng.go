package spawn

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
)

// keyedHash is the seed function of PLAN §4: a keyed hash of the cell, the
// epoch and the condition digest. PLAN says siphash; HMAC-SHA-256 truncated
// to 64 bits is in the standard library and runs once per cell, so the cost
// is irrelevant. Nobody without the secret can predict a cell's spawns.
func keyedHash(secret []byte, parts ...uint64) uint64 {
	mac := hmac.New(sha256.New, secret)
	var buf [8]byte
	for _, p := range parts {
		binary.LittleEndian.PutUint64(buf[:], p)
		mac.Write(buf[:])
	}
	return binary.LittleEndian.Uint64(mac.Sum(nil))
}

// rng is splitmix64: tiny, fast, and identical on every platform, which is
// what "two devices in the same cell see the same monsters" needs.
type rng struct{ s uint64 }

func newRNG(seed uint64) *rng { return &rng{s: seed} }

func (r *rng) next() uint64 {
	r.s += 0x9E3779B97F4A7C15
	z := r.s
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// float returns a uniform value in [0, 1).
func (r *rng) float() float64 { return float64(r.next()>>11) / (1 << 53) }

// intn returns a uniform value in [0, n).
func (r *rng) intn(n int) int {
	if n <= 1 {
		return 0
	}
	return int(r.next() % uint64(n))
}

// pick returns an index drawn in proportion to weights; -1 if all are zero.
func (r *rng) pick(weights []float64) int {
	var sum float64
	for _, w := range weights {
		if w > 0 {
			sum += w
		}
	}
	if sum <= 0 {
		return -1
	}
	x := r.float() * sum
	for i, w := range weights {
		if w <= 0 {
			continue
		}
		if x < w {
			return i
		}
		x -= w
	}
	for i := len(weights) - 1; i >= 0; i-- {
		if weights[i] > 0 {
			return i
		}
	}
	return -1
}

// mix derives an independent sub-seed for slot i of a seed.
func mix(seed uint64, i uint64) uint64 {
	r := rng{s: seed ^ (i+1)*0xD1B54A32D192ED03}
	return r.next()
}
