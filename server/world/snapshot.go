package world

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"

	"github.com/adaralex/strata/server/world/h3x"
)

// Snapshot file names inside a snapshot directory. cells.bin is the fixed
// width part; the JSON side tables are small and variable-width. civs.json is
// copied in at build time so a snapshot is self-contained.
const (
	FileCells   = "cells.bin"
	FileBeacons = "beacons.json"
	FilePOIs    = "pois.json"
	FileZones   = "zones.json"
	FileCivs    = "civs.json"

	snapshotMagic   = "STR8"
	snapshotVersion = 3
)

// Snapshot is one immutable world build (PLAN.md §14 step 4/5).
type Snapshot struct {
	BuildID    string
	CostUnitKm float64
	// Classes names the POI classes in ServiceMask bit order.
	Classes []string
	// Keys are sorted r8 indices; Records[i] belongs to Keys[i].
	Keys    []uint64
	Records []Record
	// BeaconAdj holds beacon ids; a record's BeaconStart/BeaconN slice it.
	BeaconAdj []uint32
	// Excl is the sorted set of hard-excluded r10 cells. ExclZone[i] indexes
	// ExclZoneNames with the exclusion that claimed Excl[i] first.
	Excl          []uint64
	ExclZone      []uint8
	ExclZoneNames []string
	// ZoneIdx and ZoneIdxZone are pairs (r10 cell, zone id) sorted by cell:
	// the spatial index for the exact geometry test in Lookup. It covers one
	// ring beyond the excluded cells so a point just outside still gets tested.
	ZoneIdx     []uint64
	ZoneIdxZone []uint32

	Beacons []Beacon
	POIs    []POI
	// Zones are the exclusion geometries (PLAN.md §11) with their buffers.
	Zones []Zone
}

// Zone is one exclusion feature: the geometry that no gameplay may touch,
// plus the buffer around it.
type Zone struct {
	ID       uint32            `json:"id"`
	Kind     string            `json:"kind"` // exclusion id: school, worship, blocklist...
	Ref      string            `json:"ref"`
	Name     string            `json:"name,omitempty"`
	BufferM  float64           `json:"buffer_m"`
	Lat      float64           `json:"lat"`
	Lon      float64           `json:"lon"`
	Geometry *geojson.Geometry `json:"geometry,omitempty"`
}

// DistanceM is the distance from p to the zone geometry, 0 inside.
func (z *Zone) DistanceM(p orb.Point) float64 {
	if z.Geometry != nil {
		switch g := z.Geometry.Geometry().(type) {
		case orb.Polygon:
			return h3x.DistanceToPolygonM(p, g)
		case orb.MultiPolygon:
			return h3x.DistanceToMultiPolygonM(p, g)
		case orb.LineString:
			return h3x.DistanceToLineM(p, g)
		}
	}
	return h3x.DistanceM(p, orb.Point{z.Lon, z.Lat})
}

// Contains reports whether p is inside the zone or its buffer.
func (z *Zone) Contains(p orb.Point) bool { return z.DistanceM(p) <= z.BufferM }

// Beacon is a museum, gallery or historic site that shifts spawn tier and
// carries its own civilization vector (PLAN.md §6, §9).
type Beacon struct {
	ID     uint32 `json:"id"`
	Ref    string `json:"ref"` // OSM ref: n/123, w/456, r/789
	Name   string `json:"name,omitempty"`
	Grade  uint8  `json:"grade"`
	Source string `json:"civ_source"` // curated | historic:civilization | cell_soil
	// Civs is a weight vector over civilization keys. Empty means the beacon
	// inherits the soil vector of its cell and only shifts tier.
	Civs      map[string]float64 `json:"civs,omitempty"`
	Lat       float64            `json:"lat"`
	Lon       float64            `json:"lon"`
	RangeM    float64            `json:"range_m"`
	CooldownS int                `json:"cooldown_s"`
	Geometry  *geojson.Geometry  `json:"geometry,omitempty"`
}

// Polygon returns the beacon footprint, or nil for a point beacon.
func (b *Beacon) Polygon() orb.Polygon {
	if b.Geometry == nil {
		return nil
	}
	switch g := b.Geometry.Geometry().(type) {
	case orb.Polygon:
		return g
	case orb.MultiPolygon:
		if len(g) > 0 {
			return g[0]
		}
	}
	return nil
}

// POI is one classified city-service point (PLAN.md §9).
type POI struct {
	Ref      string  `json:"ref"`
	Name     string  `json:"name,omitempty"`
	Class    string  `json:"class"`
	Subclass string  `json:"subclass"`
	Potency  float64 `json:"potency"`
	Grade    uint8   `json:"grade,omitempty"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	RangeM   float64 `json:"range_m"`
	Brand    string  `json:"brand_wikidata,omitempty"`
	Hours    string  `json:"opening_hours,omitempty"`
	Cell     uint64  `json:"cell"`
	// Excluded names the exclusion that suppressed this POI; empty = live.
	Excluded string `json:"excluded,omitempty"`
}

// Write stores the snapshot into dir, creating it.
func (s *Snapshot) Write(dir string, civsJSON []byte) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if len(s.Keys) != len(s.Records) {
		return fmt.Errorf("snapshot: %d keys but %d records", len(s.Keys), len(s.Records))
	}
	if !sort.SliceIsSorted(s.Keys, func(i, j int) bool { return s.Keys[i] < s.Keys[j] }) {
		return fmt.Errorf("snapshot: keys must be sorted")
	}
	f, err := os.Create(filepath.Join(dir, FileCells))
	if err != nil {
		return err
	}
	w := bufio.NewWriterSize(f, 1<<20)
	if err := s.encodeCells(w); err != nil {
		_ = f.Close()
		return err
	}
	if err := w.Flush(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, FileBeacons), s.Beacons); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, FilePOIs), s.POIs); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(dir, FileZones), s.Zones); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, FileCivs), civsJSON, 0o644)
}

func (s *Snapshot) encodeCells(w io.Writer) error {
	put := func(v any) error { return binary.Write(w, binary.LittleEndian, v) }
	if _, err := w.Write([]byte(snapshotMagic)); err != nil {
		return err
	}
	if err := put(uint16(snapshotVersion)); err != nil {
		return err
	}
	if err := putString(w, s.BuildID); err != nil {
		return err
	}
	if err := put(s.CostUnitKm); err != nil {
		return err
	}
	if len(s.Classes) > 16 {
		return fmt.Errorf("snapshot: ServiceMask holds 16 classes, got %d", len(s.Classes))
	}
	if err := put(uint8(len(s.Classes))); err != nil {
		return err
	}
	for _, c := range s.Classes {
		if err := putString(w, c); err != nil {
			return err
		}
	}
	if err := put(uint32(len(s.Keys))); err != nil {
		return err
	}
	if err := put(uint32(len(s.BeaconAdj))); err != nil {
		return err
	}
	if err := put(uint32(len(s.Excl))); err != nil {
		return err
	}
	if len(s.ExclZoneNames) > 255 {
		return fmt.Errorf("snapshot: at most 255 exclusion zone names")
	}
	if err := put(uint8(len(s.ExclZoneNames))); err != nil {
		return err
	}
	for _, n := range s.ExclZoneNames {
		if err := putString(w, n); err != nil {
			return err
		}
	}
	if err := put(s.Keys); err != nil {
		return err
	}
	buf := make([]byte, RecordSize)
	for i := range s.Records {
		s.Records[i].MarshalTo(buf)
		if _, err := w.Write(buf); err != nil {
			return err
		}
	}
	if err := put(s.BeaconAdj); err != nil {
		return err
	}
	if err := put(s.Excl); err != nil {
		return err
	}
	if len(s.ExclZone) != len(s.Excl) {
		return fmt.Errorf("snapshot: %d excluded cells but %d zone ids", len(s.Excl), len(s.ExclZone))
	}
	if err := put(s.ExclZone); err != nil {
		return err
	}
	if len(s.ZoneIdxZone) != len(s.ZoneIdx) {
		return fmt.Errorf("snapshot: %d zone index cells but %d zone ids", len(s.ZoneIdx), len(s.ZoneIdxZone))
	}
	if err := put(uint32(len(s.ZoneIdx))); err != nil {
		return err
	}
	if err := put(s.ZoneIdx); err != nil {
		return err
	}
	return put(s.ZoneIdxZone)
}

// Read loads a snapshot directory.
func Read(dir string) (*Snapshot, error) {
	f, err := os.Open(filepath.Join(dir, FileCells))
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	s, err := decodeCells(bufio.NewReaderSize(f, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("snapshot %s: %w", dir, err)
	}
	if err := readJSON(filepath.Join(dir, FileBeacons), &s.Beacons); err != nil {
		return nil, err
	}
	if err := readJSON(filepath.Join(dir, FilePOIs), &s.POIs); err != nil {
		return nil, err
	}
	if err := readJSON(filepath.Join(dir, FileZones), &s.Zones); err != nil {
		return nil, err
	}
	return s, nil
}

func decodeCells(r io.Reader) (*Snapshot, error) {
	get := func(v any) error { return binary.Read(r, binary.LittleEndian, v) }
	magic := make([]byte, 4)
	if _, err := io.ReadFull(r, magic); err != nil {
		return nil, err
	}
	if string(magic) != snapshotMagic {
		return nil, fmt.Errorf("bad magic %q", magic)
	}
	var version uint16
	if err := get(&version); err != nil {
		return nil, err
	}
	if version != snapshotVersion {
		return nil, fmt.Errorf("snapshot version %d, want %d", version, snapshotVersion)
	}
	s := &Snapshot{}
	var err error
	if s.BuildID, err = getString(r); err != nil {
		return nil, err
	}
	if err := get(&s.CostUnitKm); err != nil {
		return nil, err
	}
	var nClasses uint8
	if err := get(&nClasses); err != nil {
		return nil, err
	}
	for i := 0; i < int(nClasses); i++ {
		c, err := getString(r)
		if err != nil {
			return nil, err
		}
		s.Classes = append(s.Classes, c)
	}
	var nCells, nAdj, nExcl uint32
	if err := get(&nCells); err != nil {
		return nil, err
	}
	if err := get(&nAdj); err != nil {
		return nil, err
	}
	if err := get(&nExcl); err != nil {
		return nil, err
	}
	var nZones uint8
	if err := get(&nZones); err != nil {
		return nil, err
	}
	for i := 0; i < int(nZones); i++ {
		n, err := getString(r)
		if err != nil {
			return nil, err
		}
		s.ExclZoneNames = append(s.ExclZoneNames, n)
	}
	s.Keys = make([]uint64, nCells)
	if err := get(s.Keys); err != nil {
		return nil, err
	}
	s.Records = make([]Record, nCells)
	buf := make([]byte, RecordSize)
	for i := range s.Records {
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		if err := s.Records[i].UnmarshalFrom(buf); err != nil {
			return nil, err
		}
	}
	s.BeaconAdj = make([]uint32, nAdj)
	if err := get(s.BeaconAdj); err != nil {
		return nil, err
	}
	s.Excl = make([]uint64, nExcl)
	if err := get(s.Excl); err != nil {
		return nil, err
	}
	s.ExclZone = make([]uint8, nExcl)
	if err := get(s.ExclZone); err != nil {
		return nil, err
	}
	var nIdx uint32
	if err := get(&nIdx); err != nil {
		return nil, err
	}
	s.ZoneIdx = make([]uint64, nIdx)
	if err := get(s.ZoneIdx); err != nil {
		return nil, err
	}
	s.ZoneIdxZone = make([]uint32, nIdx)
	if err := get(s.ZoneIdxZone); err != nil {
		return nil, err
	}
	return s, nil
}

// Find returns the record index for an r8 cell key, or -1.
func (s *Snapshot) Find(key uint64) int {
	i := sort.Search(len(s.Keys), func(i int) bool { return s.Keys[i] >= key })
	if i < len(s.Keys) && s.Keys[i] == key {
		return i
	}
	return -1
}

// IsExcluded reports whether an r10 cell is in the hard-exclusion set.
func (s *Snapshot) IsExcluded(r10 uint64) bool {
	_, ok := s.ExcludedBy(r10)
	return ok
}

// ExcludedBy returns the exclusion zone id that claimed an r10 cell.
func (s *Snapshot) ExcludedBy(r10 uint64) (string, bool) {
	i := sort.Search(len(s.Excl), func(i int) bool { return s.Excl[i] >= r10 })
	if i >= len(s.Excl) || s.Excl[i] != r10 {
		return "", false
	}
	if i < len(s.ExclZone) && int(s.ExclZone[i]) < len(s.ExclZoneNames) {
		return s.ExclZoneNames[s.ExclZone[i]], true
	}
	return "?", true
}

// ClassBit returns the ServiceMask bit for a class name, or 0 if unknown.
func (s *Snapshot) ClassBit(class string) uint16 {
	for i, c := range s.Classes {
		if c == class {
			return 1 << uint(i)
		}
	}
	return 0
}

// ServiceNames expands a ServiceMask into class names.
func (s *Snapshot) ServiceNames(mask uint16) []string {
	var out []string
	for i, c := range s.Classes {
		if mask&(1<<uint(i)) != 0 {
			out = append(out, c)
		}
	}
	return out
}

func putString(w io.Writer, s string) error {
	if len(s) > 255 {
		return fmt.Errorf("string too long: %d", len(s))
	}
	if _, err := w.Write([]byte{uint8(len(s))}); err != nil {
		return err
	}
	_, err := w.Write([]byte(s))
	return err
}

func getString(r io.Reader) (string, error) {
	var n [1]byte
	if _, err := io.ReadFull(r, n[:]); err != nil {
		return "", err
	}
	b := make([]byte, n[0])
	if _, err := io.ReadFull(r, b); err != nil {
		return "", err
	}
	return string(b), nil
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// ZonesAt returns the ids of the exclusion zones indexed on an r10 cell.
func (s *Snapshot) ZonesAt(r10 uint64) []uint32 {
	i := sort.Search(len(s.ZoneIdx), func(i int) bool { return s.ZoneIdx[i] >= r10 })
	j := i
	for j < len(s.ZoneIdx) && s.ZoneIdx[j] == r10 {
		j++
	}
	return s.ZoneIdxZone[i:j]
}

// ZoneContaining returns the first zone whose buffered geometry contains
// the point, using the r10 index. This is the exact test behind "no
// gameplay at" a place; the r10 set is only the cheap raster for spawns.
func (s *Snapshot) ZoneContaining(lat, lon float64) (*Zone, bool) {
	r10, err := h3x.FromLatLng(lat, lon, h3x.ResPlace)
	if err != nil {
		return nil, false
	}
	p := orb.Point{lon, lat}
	for _, id := range s.ZonesAt(uint64(r10)) {
		if int(id) < len(s.Zones) && s.Zones[id].Contains(p) {
			return &s.Zones[id], true
		}
	}
	return nil, false
}
