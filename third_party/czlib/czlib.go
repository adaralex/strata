// Package czlib is a pure-Go stand-in for github.com/DataDog/czlib, wired in
// through a replace directive in the root go.mod.
//
// github.com/paulmach/osm/osmpbf selects the C zlib binding whenever CGO is
// enabled, and CGO is enabled in this repo because the H3 binding needs it.
// The C binding needs pkg-config and the zlib headers on every machine that
// builds the pipeline. osmpbf uses exactly one function from it, so this
// package provides that function over compress/zlib instead. Decompression
// is slower than C zlib, but PBF blocks decode in parallel and a regional
// extract is still a matter of minutes.
package czlib

import (
	"compress/zlib"
	"io"
)

// NewReader returns a reader that decompresses zlib data from r.
func NewReader(r io.Reader) (io.ReadCloser, error) {
	return zlib.NewReader(r)
}
