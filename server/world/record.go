package world

import (
	"encoding/binary"
	"fmt"
)

// RecordSize is the fixed width of one r8 cell record in bytes.
const RecordSize = 32

// CostUnreached marks a civilization that no layer reaches (yet). Track 2's
// drift solve is what removes these; track 1 leaves them for anything the
// hand-drawn layers do not cover.
const CostUnreached = 0xFFFF

// Terrain flag bits in Record.TerrainFlags.
const (
	// Water proximity band, two bits: 0 = water in this cell, 1 = adjacent
	// cell, 2 = two cells away, 3 = farther.
	TerrainWaterMask  uint8 = 0b0000_0011
	TerrainCoast      uint8 = 0b0000_0100
	TerrainWild       uint8 = 0b0000_1000 // park, wood, forest, beach
	TerrainUrbanMask  uint8 = 0b0011_0000 // two bits from poi density
	TerrainUrbanShift       = 4
	TerrainLit        uint8 = 0b0100_0000 // not populated in phase 0
)

// Record is the per-cell payload. Costs are stored, weights are derived
// (PLAN.md §5: "the costs are static; only lambda moves").
//
//	off  field           type
//	 0   Civ[4]          4 x u8   civ ids, NoCiv = empty, ranked by cost/lambda
//	 4   Cost[4]         4 x u16  km-equivalent; CostUnreached = 0xFFFF
//	12   ResidualCost    u16      the other eleven folded into one pseudo-cost
//	14   ServiceMask     u16      one bit per POI class present in the cell
//	16   POIDensity      u8       16*log2(1+n) over the cell and its k-ring 1
//	17   BeaconGrade     u8       max beacon grade touching the cell
//	18   BeaconStart     u32      offset into the beacon adjacency array
//	22   BeaconN         u8       adjacency count
//	23   ExclChildren    u8       hard-excluded r10 children, 0..49
//	24   TerrainFlags    u8       see Terrain* bits
//	25   ElevationBand   u8       25 m steps (0 until a DEM is wired in)
//	26   reserved        6 x u8
type Record struct {
	Civ           [TopK]uint8
	Cost          [TopK]uint16
	ResidualCost  uint16
	ServiceMask   uint16
	POIDensity    uint8
	BeaconGrade   uint8
	BeaconStart   uint32
	BeaconN       uint8
	ExclChildren  uint8
	TerrainFlags  uint8
	ElevationBand uint8
}

// EmptyRecord is a record with no civilization reached and nothing in it.
func EmptyRecord() Record {
	var r Record
	for i := range r.Civ {
		r.Civ[i] = NoCiv
		r.Cost[i] = CostUnreached
	}
	r.ResidualCost = CostUnreached
	return r
}

// MarshalTo writes the record into b, which must be at least RecordSize long.
func (r *Record) MarshalTo(b []byte) {
	_ = b[RecordSize-1]
	copy(b[0:4], r.Civ[:])
	for i := 0; i < TopK; i++ {
		binary.LittleEndian.PutUint16(b[4+2*i:], r.Cost[i])
	}
	binary.LittleEndian.PutUint16(b[12:], r.ResidualCost)
	binary.LittleEndian.PutUint16(b[14:], r.ServiceMask)
	b[16] = r.POIDensity
	b[17] = r.BeaconGrade
	binary.LittleEndian.PutUint32(b[18:], r.BeaconStart)
	b[22] = r.BeaconN
	b[23] = r.ExclChildren
	b[24] = r.TerrainFlags
	b[25] = r.ElevationBand
	for i := 26; i < RecordSize; i++ {
		b[i] = 0
	}
}

// UnmarshalFrom reads a record from b.
func (r *Record) UnmarshalFrom(b []byte) error {
	if len(b) < RecordSize {
		return fmt.Errorf("record: need %d bytes, got %d", RecordSize, len(b))
	}
	copy(r.Civ[:], b[0:4])
	for i := 0; i < TopK; i++ {
		r.Cost[i] = binary.LittleEndian.Uint16(b[4+2*i:])
	}
	r.ResidualCost = binary.LittleEndian.Uint16(b[12:])
	r.ServiceMask = binary.LittleEndian.Uint16(b[14:])
	r.POIDensity = b[16]
	r.BeaconGrade = b[17]
	r.BeaconStart = binary.LittleEndian.Uint32(b[18:])
	r.BeaconN = b[22]
	r.ExclChildren = b[23]
	r.TerrainFlags = b[24]
	r.ElevationBand = b[25]
	return nil
}

// WaterBand returns the water proximity band (0 nearest, 3 farthest).
func (r *Record) WaterBand() uint8 { return r.TerrainFlags & TerrainWaterMask }

// IsWild reports whether a park, wood, forest or beach touches the cell.
func (r *Record) IsWild() bool { return r.TerrainFlags&TerrainWild != 0 }

// UrbanBand returns the coarse urban density band (0..3).
func (r *Record) UrbanBand() uint8 { return (r.TerrainFlags & TerrainUrbanMask) >> TerrainUrbanShift }

// DensityByte quantises a POI count as 16*log2(1+n), saturating at 255.
func DensityByte(n int) uint8 {
	if n <= 0 {
		return 0
	}
	v := 16 * log2(1+float64(n))
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}

// DensityCount inverts DensityByte approximately.
func DensityCount(b uint8) float64 {
	return pow2(float64(b)/16) - 1
}
