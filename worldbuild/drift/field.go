package drift

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"sort"

	"github.com/adaralex/strata/server/world"
	"github.com/adaralex/strata/server/world/h3x"
)

const (
	fieldMagic   = "DRFT"
	fieldVersion = 1
)

// Field is the solved planet: sorted cell keys and fifteen u16 costs per
// cell, in km-equivalent, CostUnreached where no source reaches.
type Field struct {
	Res          int
	RulesVersion int
	Keys         []uint64
	Costs        []uint16 // len(Keys) * NumCivs
}

// Write stores the field.
func (f *Field) Write(path string) error {
	fh, err := os.Create(path)
	if err != nil {
		return err
	}
	w := bufio.NewWriterSize(fh, 1<<20)
	put := func(v any) error { return binary.Write(w, binary.LittleEndian, v) }
	if _, err := w.Write([]byte(fieldMagic)); err != nil {
		return err
	}
	for _, v := range []any{uint16(fieldVersion), uint8(f.Res), uint8(world.NumCivs), uint32(f.RulesVersion), uint32(len(f.Keys))} {
		if err := put(v); err != nil {
			return err
		}
	}
	if err := put(f.Keys); err != nil {
		return err
	}
	if err := put(f.Costs); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}
	return fh.Close()
}

// Read loads a field.
func Read(path string) (*Field, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = fh.Close() }()
	r := bufio.NewReaderSize(fh, 1<<20)
	get := func(v any) error { return binary.Read(r, binary.LittleEndian, v) }
	magic := make([]byte, 4)
	if _, err := io.ReadFull(r, magic); err != nil {
		return nil, err
	}
	if string(magic) != fieldMagic {
		return nil, fmt.Errorf("%s: not a drift field", path)
	}
	var version uint16
	var res, ncivs uint8
	var rulesVersion, n uint32
	for _, v := range []any{&version, &res, &ncivs, &rulesVersion, &n} {
		if err := get(v); err != nil {
			return nil, err
		}
	}
	if version != fieldVersion || int(ncivs) != world.NumCivs {
		return nil, fmt.Errorf("%s: version %d with %d civs, want %d and %d", path, version, ncivs, fieldVersion, world.NumCivs)
	}
	f := &Field{Res: int(res), RulesVersion: int(rulesVersion), Keys: make([]uint64, n), Costs: make([]uint16, int(n)*world.NumCivs)}
	if err := get(f.Keys); err != nil {
		return nil, err
	}
	if err := get(f.Costs); err != nil {
		return nil, err
	}
	return f, nil
}

// Find returns a cell's index or -1.
func (f *Field) Find(c h3x.Cell) int {
	k := uint64(c)
	i := sort.Search(len(f.Keys), func(i int) bool { return f.Keys[i] >= k })
	if i < len(f.Keys) && f.Keys[i] == k {
		return i
	}
	return -1
}

// CostsOf returns a cell's fifteen costs.
func (f *Field) CostsOf(i int) []uint16 { return f.Costs[i*world.NumCivs : (i+1)*world.NumCivs] }

// At interpolates the fifteen costs at a point: inverse-distance weights
// over the containing cell and its ring, so r8 cells inside one r5 cell do
// not all read the same number. Returns false outside the field.
func (f *Field) At(lat, lon float64) ([world.NumCivs]float64, bool) {
	var out [world.NumCivs]float64
	c, err := h3x.FromLatLng(lat, lon, f.Res)
	if err != nil {
		return out, false
	}
	disk, err := h3x.Disk(c, 1)
	if err != nil {
		return out, false
	}
	var wsum float64
	var acc [world.NumCivs]float64
	var reached [world.NumCivs]bool
	for _, d := range disk {
		i := f.Find(d)
		if i < 0 {
			continue
		}
		clat, clon, err := h3x.Center(d)
		if err != nil {
			continue
		}
		dist := haversineKm(lat, lon, clat, clon) + 0.5 // km, floor avoids a singularity
		w := 1 / (dist * dist)
		wsum += w
		for k, v := range f.CostsOf(i) {
			if v != world.CostUnreached {
				acc[k] += w * float64(v)
				reached[k] = true
			}
		}
	}
	if wsum == 0 {
		return out, false
	}
	for k := range out {
		if reached[k] {
			out[k] = acc[k] / wsum
		} else {
			out[k] = math.Inf(1)
		}
	}
	return out, true
}

// Unreached counts (cell, civ) pairs no source reaches and cells with any.
func (f *Field) Unreached() (pairs, cellsWithAny int) {
	for i := 0; i < len(f.Keys); i++ {
		any := false
		for _, v := range f.CostsOf(i) {
			if v == world.CostUnreached {
				pairs++
				any = true
			}
		}
		if any {
			cellsWithAny++
		}
	}
	return
}
