package world

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/paulmach/orb"

	"github.com/adaralex/strata/server/world/h3x"
)

// World is a loaded snapshot plus the roster, ready to answer queries.
type World struct {
	Snap *Snapshot
	Civs *Civs
}

// Load reads a snapshot directory and its embedded civs.json.
func Load(dir string) (*World, error) {
	snap, err := Read(dir)
	if err != nil {
		return nil, err
	}
	civs, err := LoadCivs(filepath.Join(dir, FileCivs))
	if err != nil {
		return nil, err
	}
	return &World{Snap: snap, Civs: civs}, nil
}

// BeaconHit is a beacon whose footprint plus range covers the query point.
type BeaconHit struct {
	Beacon    *Beacon
	DistanceM float64 // 0 inside the footprint
}

// Result is what a lat/lon resolves to.
type Result struct {
	Cell     h3x.Cell
	Found    bool // false when the cell is outside the snapshot
	Record   Record
	Weights  Weights
	Services []string
	Beacons  []BeaconHit
	// Excluded is true when the r10 cell under the point is hard-excluded;
	// ExcludedBy names the zone.
	Excluded   bool
	ExcludedBy string
}

// Lookup resolves a point under a propagation scalar (PLAN.md §4, §5).
func (w *World) Lookup(lat, lon, propagation float64) (Result, error) {
	cell, err := h3x.FromLatLng(lat, lon, h3x.ResWeight)
	if err != nil {
		return Result{}, err
	}
	res := Result{Cell: cell}
	i := w.Snap.Find(uint64(cell))
	if i < 0 {
		return res, nil
	}
	res.Found = true
	res.Record = w.Snap.Records[i]
	res.Weights = Derive(&res.Record, w.Civs, propagation)
	res.Services = w.Snap.ServiceNames(res.Record.ServiceMask)
	pt := orb.Point{lon, lat}
	rec := &res.Record
	end := int(rec.BeaconStart) + int(rec.BeaconN)
	if end > len(w.Snap.BeaconAdj) {
		return res, fmt.Errorf("lookup: beacon adjacency out of range for cell %s", cell)
	}
	for _, id := range w.Snap.BeaconAdj[rec.BeaconStart:end] {
		if int(id) >= len(w.Snap.Beacons) {
			continue
		}
		b := &w.Snap.Beacons[id]
		var d float64
		if poly := b.Polygon(); poly != nil {
			d = h3x.DistanceToPolygonM(pt, poly)
		} else {
			d = h3x.DistanceM(pt, orb.Point{b.Lon, b.Lat})
		}
		if d <= b.RangeM {
			res.Beacons = append(res.Beacons, BeaconHit{Beacon: b, DistanceM: d})
		}
	}
	sort.Slice(res.Beacons, func(a, b int) bool { return res.Beacons[a].DistanceM < res.Beacons[b].DistanceM })
	if rec.ExclChildren > 0 {
		r10, err := h3x.FromLatLng(lat, lon, h3x.ResPlace)
		if err != nil {
			return res, err
		}
		res.ExcludedBy, res.Excluded = w.Snap.ExcludedBy(uint64(r10))
	}
	return res, nil
}

// Near is a POI or beacon within a radius of a query point.
type Near struct {
	Kind      string // "poi" or "beacon"
	DistanceM float64
	POI       *POI
	Beacon    *Beacon
}

// Nearby lists live POIs and beacons within radiusM, nearest first. It scans
// the side tables, which is fine for a CLI and a region-sized snapshot.
func (w *World) Nearby(lat, lon, radiusM float64) []Near {
	pt := orb.Point{lon, lat}
	var out []Near
	for i := range w.Snap.Beacons {
		b := &w.Snap.Beacons[i]
		d := h3x.DistanceM(pt, orb.Point{b.Lon, b.Lat})
		if d <= radiusM {
			out = append(out, Near{Kind: "beacon", DistanceM: d, Beacon: b})
		}
	}
	for i := range w.Snap.POIs {
		p := &w.Snap.POIs[i]
		if p.Excluded != "" {
			continue
		}
		d := h3x.DistanceM(pt, orb.Point{p.Lon, p.Lat})
		if d <= radiusM {
			out = append(out, Near{Kind: "poi", DistanceM: d, POI: p})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DistanceM < out[j].DistanceM })
	return out
}

// CivName resolves a civ id to its key for display.
func (w *World) CivName(id uint8) string {
	if id >= NumCivs {
		return "-"
	}
	return w.Civs.List[id].Key
}
