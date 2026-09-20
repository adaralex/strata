package cells

import (
	"fmt"
	"io"
	"math"
	"runtime"
	"sort"
	"sync"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"

	"github.com/adaralex/strata/server/world"
	"github.com/adaralex/strata/server/world/h3x"
	"github.com/adaralex/strata/worldbuild/classify"
)

// MaxCells guards against a header bbox that covers a continent; pass an
// explicit bbox instead.
const MaxCells = 5_000_000

// Inputs is everything one build consumes.
type Inputs struct {
	BuildID    string
	Civs       *world.Civs
	Rules      *classify.Rules
	Layers     []Layer
	Curated    []Curated
	CivTags    CivTags
	Blocklist  []orb.Polygon
	Classified *classify.Result
	Bounds     classify.BBox
	Progress   io.Writer
}

// Stats summarises a build.
type Stats struct {
	Cells, ExclCells, Beacons, LivePOIs, ZonedPOIs int
	Reached                                        int // cells with at least one civ
}

// Build assembles the snapshot.
func Build(in Inputs) (*world.Snapshot, Stats, error) {
	var st Stats
	log := func(format string, a ...any) {
		if in.Progress != nil {
			_, _ = fmt.Fprintf(in.Progress, format+"\n", a...)
		}
	}
	b := in.Bounds
	cells, err := h3x.BBoxCells(b.MinLat, b.MinLon, b.MaxLat, b.MaxLon, h3x.ResWeight)
	if err != nil {
		return nil, st, fmt.Errorf("bbox cells: %w", err)
	}
	if len(cells) > MaxCells {
		return nil, st, fmt.Errorf("bbox covers %d r8 cells, above the %d guard; pass a smaller -bbox", len(cells), MaxCells)
	}
	h3x.SortCells(cells)
	index := make(map[h3x.Cell]int, len(cells))
	for i, c := range cells {
		index[c] = i
	}
	snap := &world.Snapshot{BuildID: in.BuildID, CostUnitKm: in.Civs.CostUnitKm, Classes: in.Rules.ClassIDs()}
	snap.Keys = make([]uint64, len(cells))
	snap.Records = make([]world.Record, len(cells))
	for i, c := range cells {
		snap.Keys[i] = uint64(c)
		snap.Records[i] = world.EmptyRecord()
	}
	st.Cells = len(cells)
	log("cells: %d r8 cells over %.3f,%.3f - %.3f,%.3f", len(cells), b.MinLat, b.MinLon, b.MaxLat, b.MaxLon)

	// 1. Soil: costs from the civilization layers, in parallel.
	reached := soil(cells, snap.Records, in.Layers, in.Civs)
	st.Reached = reached
	log("soil: %d/%d cells reached by at least one layer", reached, len(cells))

	// 2. Exclusion zones. The r10 set is what spawn placement checks. It is
	// also the spatial index for the exact geometry test on POIs below, so a
	// bakery on the same square as the war memorial stays live unless it is
	// inside the memorial's own buffer.
	var zones []zone
	for i := range in.Classified.Items {
		it := &in.Classified.Items[i]
		if it.IsExclusionZone {
			zones = append(zones, zone{ID: it.Excluded, BufferM: it.Exclusion.BufferM, F: it.Feature})
		}
	}
	for i, poly := range in.Blocklist {
		f := &classify.Feature{Ref: fmt.Sprintf("blocklist/%d", i), Polygon: poly, Point: h3x.Centroid(poly)}
		zones = append(zones, zone{ID: "blocklist", BufferM: 100, F: f})
	}
	excl := map[uint64]string{}
	zoneIndex := map[uint64][]int{}
	for zi := range zones {
		z := &zones[zi]
		k := int(math.Round(z.BufferM / 100))
		// A zero-buffer zone (a place of worship) excludes only the cells
		// whose centre is inside it: at 66 m cells, overlap would swallow
		// the square in front of every church.
		cells := featureCells(z.F, h3x.ResPlace, 30)
		if z.BufferM == 0 {
			cells = featureCellsCentre(z.F, h3x.ResPlace, 30)
		}
		for _, c := range cells {
			excluded := []h3x.Cell{c}
			if k > 0 {
				if d, err := h3x.Disk(c, k); err == nil {
					excluded = d
				}
			}
			for _, cc := range excluded {
				if _, ok := excl[uint64(cc)]; !ok {
					excl[uint64(cc)] = z.ID
				}
			}
			// Index one ring beyond what is excluded so a POI just over a
			// cell edge is still tested against the geometry.
			if d, err := h3x.Disk(c, k+1); err == nil {
				for _, cc := range d {
					zoneIndex[uint64(cc)] = append(zoneIndex[uint64(cc)], zi)
				}
			}
		}
	}
	snap.Excl = make([]uint64, 0, len(excl))
	for c := range excl {
		snap.Excl = append(snap.Excl, c)
	}
	sort.Slice(snap.Excl, func(i, j int) bool { return snap.Excl[i] < snap.Excl[j] })
	zoneID := map[string]uint8{}
	snap.ExclZone = make([]uint8, len(snap.Excl))
	for i, c := range snap.Excl {
		name := excl[c]
		id, ok := zoneID[name]
		if !ok {
			if len(snap.ExclZoneNames) >= 255 {
				return nil, st, fmt.Errorf("more than 255 exclusion zone ids")
			}
			id = uint8(len(snap.ExclZoneNames))
			zoneID[name] = id
			snap.ExclZoneNames = append(snap.ExclZoneNames, name)
		}
		snap.ExclZone[i] = id
	}
	st.ExclCells = len(snap.Excl)
	for _, r10 := range snap.Excl {
		parent, err := h3x.Parent(h3x.Cell(r10), h3x.ResWeight)
		if err != nil {
			continue
		}
		if i, ok := index[parent]; ok && snap.Records[i].ExclChildren < 255 {
			snap.Records[i].ExclChildren++
		}
	}
	log("exclusion: %d r10 cells", len(snap.Excl))

	// 3. POIs: every class hit becomes a POI row; those inside a buffered
	// exclusion geometry are suppressed with the zone's id.
	counts := make([]int, len(cells))
	var beaconItems []int
	for i := range in.Classified.Items {
		it := &in.Classified.Items[i]
		if len(it.Hits) == 0 {
			continue
		}
		f := it.Feature
		r8, err := h3x.FromPoint(f.Point, h3x.ResWeight)
		if err != nil {
			continue
		}
		excludedBy := it.Excluded
		if excludedBy == "" {
			if id, ok := insideZone(f.Point, zones, zoneIndex); ok {
				excludedBy = "zone:" + id
				st.ZonedPOIs++
			}
		}
		live := excludedBy == ""
		ci, inRegion := index[r8]
		for _, h := range it.Hits {
			poi := world.POI{
				Ref: f.Ref, Name: f.Name(), Class: h.Class.ID, Subclass: h.Subclass, Potency: h.Potency, Grade: h.Grade,
				Lat: f.Point.Lat(), Lon: f.Point.Lon(), RangeM: h.Class.RangeM(), Brand: f.Tags["brand:wikidata"],
				Hours: f.Tags["opening_hours"], Cell: uint64(r8), Excluded: excludedBy,
			}
			if h.Class.ID == "beacon" {
				poi.RangeM = h.Class.Geometry.BufferM
			}
			snap.POIs = append(snap.POIs, poi)
			if !live || !inRegion {
				continue
			}
			st.LivePOIs++
			snap.Records[ci].ServiceMask |= snap.ClassBit(h.Class.ID)
			if len(h.Class.Interaction) > 0 {
				counts[ci]++
			}
			if h.Class.Spawn.Zone {
				for _, c := range featureCells(f, h3x.ResWeight, 200) {
					if j, ok := index[c]; ok {
						snap.Records[j].TerrainFlags |= world.TerrainWild
					}
				}
			}
		}
		if live && inRegion && it.Hits[0].Class.ID == "beacon" {
			beaconItems = append(beaconItems, i)
		}
	}
	log("pois: %d rows, %d live, %d suppressed by an exclusion zone", len(snap.POIs), st.LivePOIs, st.ZonedPOIs)

	// 4. Density over the cell and its k-ring 1; urban band from it.
	for i, c := range cells {
		n := counts[i]
		if disk, err := h3x.Disk(c, 1); err == nil {
			for _, d := range disk {
				if d == c {
					continue
				}
				if j, ok := index[d]; ok {
					n += counts[j]
				}
			}
		}
		rec := &snap.Records[i]
		rec.POIDensity = world.DensityByte(n)
		rec.TerrainFlags |= urbanBand(n) << world.TerrainUrbanShift
	}

	// 5. Beacons and their cell adjacency.
	perCell := make([][]uint32, len(cells))
	beaconClass := in.Rules.ClassByID("beacon")
	for _, i := range beaconItems {
		it := &in.Classified.Items[i]
		f := it.Feature
		h := it.Hits[0]
		bc := world.Beacon{
			ID: uint32(len(snap.Beacons)), Ref: f.Ref, Name: f.Name(), Grade: h.Grade,
			Lat: f.Point.Lat(), Lon: f.Point.Lon(), RangeM: beaconClass.Geometry.BufferM,
		}
		if len(beaconClass.Interaction) > 0 {
			bc.CooldownS = interactionInt(beaconClass, "cooldown_s")
		}
		if f.Polygon != nil {
			bc.Geometry = geojson.NewGeometry(f.Polygon)
		}
		switch cur := findCurated(in.Curated, f.Ref, f.Name()); {
		case cur != nil:
			bc.Source = "curated"
			bc.Civs = normalise(cur.Civs)
			if cur.Grade > 0 {
				bc.Grade = cur.Grade
			}
		default:
			if key, ok := in.CivTags.Resolve(f.Tags); ok {
				bc.Source = "historic:civilization"
				bc.Civs = map[string]float64{key: 1}
			} else {
				bc.Source = "cell_soil"
			}
		}
		touch := map[h3x.Cell]struct{}{}
		for _, c := range featureCells(f, h3x.ResWeight, 200) {
			if d, err := h3x.Disk(c, 1); err == nil {
				for _, dc := range d {
					touch[dc] = struct{}{}
				}
			}
		}
		for c := range touch {
			if j, ok := index[c]; ok {
				perCell[j] = append(perCell[j], bc.ID)
				if bc.Grade > snap.Records[j].BeaconGrade {
					snap.Records[j].BeaconGrade = bc.Grade
				}
			}
		}
		snap.Beacons = append(snap.Beacons, bc)
	}
	for i := range cells {
		ids := perCell[i]
		if len(ids) == 0 {
			continue
		}
		sort.Slice(ids, func(a, b int) bool { return ids[a] < ids[b] })
		if len(ids) > 255 {
			ids = ids[:255]
		}
		snap.Records[i].BeaconStart = uint32(len(snap.BeaconAdj))
		snap.Records[i].BeaconN = uint8(len(ids))
		snap.BeaconAdj = append(snap.BeaconAdj, ids...)
	}
	st.Beacons = len(snap.Beacons)
	log("beacons: %d, %d cell adjacencies", len(snap.Beacons), len(snap.BeaconAdj))

	// 6. Terrain: water proximity band and coast.
	water := map[h3x.Cell]struct{}{}
	coast := map[h3x.Cell]struct{}{}
	for i := range in.Classified.Items {
		it := &in.Classified.Items[i]
		for _, t := range it.Terrain {
			var set map[h3x.Cell]struct{}
			switch t {
			case "water":
				set = water
			case "coast":
				set = coast
			default:
				continue
			}
			for _, c := range featureCells(it.Feature, h3x.ResWeight, 200) {
				set[c] = struct{}{}
			}
		}
	}
	for i, c := range cells {
		rec := &snap.Records[i]
		rec.TerrainFlags = rec.TerrainFlags&^world.TerrainWaterMask | waterBand(c, water)
		if _, ok := coast[c]; ok {
			rec.TerrainFlags |= world.TerrainCoast
		}
	}
	log("terrain: %d water cells, %d coast cells", len(water), len(coast))
	return snap, st, nil
}

// soil fills Civ/Cost/ResidualCost for every cell from the layers and
// returns how many cells at least one layer reaches.
func soil(cells []h3x.Cell, recs []world.Record, layers []Layer, civs *world.Civs) int {
	workers := runtime.NumCPU()
	var wg sync.WaitGroup
	var reachedMu sync.Mutex
	reached := 0
	chunk := (len(cells) + workers - 1) / workers
	for w := 0; w < workers; w++ {
		lo, hi := w*chunk, (w+1)*chunk
		if hi > len(cells) {
			hi = len(cells)
		}
		if lo >= hi {
			continue
		}
		wg.Add(1)
		go func(lo, hi int) {
			defer wg.Done()
			local := 0
			for i := lo; i < hi; i++ {
				p, err := h3x.CenterPoint(cells[i])
				if err != nil {
					continue
				}
				if fillSoil(&recs[i], p, layers, civs) {
					local++
				}
			}
			reachedMu.Lock()
			reached += local
			reachedMu.Unlock()
		}(lo, hi)
	}
	wg.Wait()
	return reached
}

type civCost struct {
	civ  uint8
	cost float64 // cost units
	norm float64 // cost / lambda_base
}

func fillSoil(rec *world.Record, p orb.Point, layers []Layer, civs *world.Civs) bool {
	var best [world.NumCivs]float64
	for i := range best {
		best[i] = math.Inf(1)
	}
	for i := range layers {
		l := &layers[i]
		if c := l.CostKmAt(p) / civs.CostUnitKm; c < best[l.Civ] {
			best[l.Civ] = c
		}
	}
	var ranked []civCost
	for id, c := range best {
		if math.IsInf(c, 1) {
			continue
		}
		ranked = append(ranked, civCost{civ: uint8(id), cost: c, norm: c * civs.CostUnitKm / civs.Lambda(uint8(id), 0)})
	}
	if len(ranked) == 0 {
		return false
	}
	sort.Slice(ranked, func(a, b int) bool { return ranked[a].norm < ranked[b].norm })
	for i := 0; i < world.TopK; i++ {
		if i < len(ranked) {
			rec.Civ[i] = ranked[i].civ
			rec.Cost[i] = world.QuantiseCost(ranked[i].cost)
		} else {
			rec.Civ[i] = world.NoCiv
			rec.Cost[i] = world.CostUnreached
		}
	}
	var tail []world.CivWeight
	for _, r := range ranked[min(world.TopK, len(ranked)):] {
		tail = append(tail, world.CivWeight{Civ: r.civ, Weight: math.Exp(-r.cost * civs.CostUnitKm / civs.Lambda(r.civ, 0))})
	}
	rec.ResidualCost = world.FoldResidual(tail, civs)
	return true
}

// featureCells rasterises a feature at res: polygon parts by overlap, lines
// by sampling every stepM, points by their cell. It never returns nothing
// for a feature with a point.
func featureCells(f *classify.Feature, res int, stepM float64) []h3x.Cell {
	seen := map[h3x.Cell]struct{}{}
	var out []h3x.Cell
	add := func(cs []h3x.Cell) {
		for _, c := range cs {
			if _, ok := seen[c]; !ok {
				seen[c] = struct{}{}
				out = append(out, c)
			}
		}
	}
	switch {
	case len(f.Multi) > 0:
		for _, poly := range f.Multi {
			if cs, err := h3x.PolyfillOverlapping(poly, res); err == nil {
				add(cs)
			}
		}
	case f.Polygon != nil:
		if cs, err := h3x.PolyfillOverlapping(f.Polygon, res); err == nil {
			add(cs)
		}
	case f.Line != nil:
		if cs, err := h3x.LineCells(f.Line, res, stepM); err == nil {
			add(cs)
		}
	}
	if c, err := h3x.FromPoint(f.Point, res); err == nil {
		add([]h3x.Cell{c})
	}
	return out
}

func waterBand(c h3x.Cell, water map[h3x.Cell]struct{}) uint8 {
	if _, ok := water[c]; ok {
		return 0
	}
	for k := 1; k <= 2; k++ {
		ring, err := c.GridRing(k)
		if err != nil {
			continue
		}
		for _, r := range ring {
			if _, ok := water[r]; ok {
				return uint8(k)
			}
		}
	}
	return 3
}

func urbanBand(n int) uint8 {
	switch {
	case n < 2:
		return 0
	case n < 8:
		return 1
	case n < 32:
		return 2
	default:
		return 3
	}
}

func normalise(m map[string]float64) map[string]float64 {
	var s float64
	for _, v := range m {
		s += v
	}
	if s <= 0 {
		return nil
	}
	out := make(map[string]float64, len(m))
	for k, v := range m {
		out[k] = v / s
	}
	return out
}

func interactionInt(c *classify.Class, key string) int {
	var m map[string]any
	if err := jsonUnmarshal(c.Interaction, &m); err != nil {
		return 0
	}
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

// zone is one exclusion geometry with its buffer.
type zone struct {
	ID      string
	BufferM float64
	F       *classify.Feature
}

// distanceM is the distance from p to the zone's geometry, 0 inside.
func (z *zone) distanceM(p orb.Point) float64 {
	switch {
	case len(z.F.Multi) > 0:
		return h3x.DistanceToMultiPolygonM(p, z.F.Multi)
	case z.F.Polygon != nil:
		return h3x.DistanceToPolygonM(p, z.F.Polygon)
	case z.F.Line != nil:
		return h3x.DistanceToLineM(p, z.F.Line)
	default:
		return h3x.DistanceM(p, z.F.Point)
	}
}

// insideZone tests a point against the exclusion zones indexed on its r10
// cell and returns the first zone whose buffered geometry contains it.
func insideZone(p orb.Point, zones []zone, index map[uint64][]int) (string, bool) {
	r10, err := h3x.FromPoint(p, h3x.ResPlace)
	if err != nil {
		return "", false
	}
	for _, zi := range index[uint64(r10)] {
		z := &zones[zi]
		if z.distanceM(p) <= z.BufferM {
			return z.ID, true
		}
	}
	return "", false
}

// featureCellsCentre is featureCells with centre containment for polygons:
// only cells whose centre lies inside. A polygon too small to hold a centre
// still yields the cell under its centroid.
func featureCellsCentre(f *classify.Feature, res int, stepM float64) []h3x.Cell {
	if f.Polygon == nil && len(f.Multi) == 0 {
		return featureCells(f, res, stepM)
	}
	seen := map[h3x.Cell]struct{}{}
	var out []h3x.Cell
	add := func(cs []h3x.Cell) {
		for _, c := range cs {
			if _, ok := seen[c]; !ok {
				seen[c] = struct{}{}
				out = append(out, c)
			}
		}
	}
	polys := f.Multi
	if len(polys) == 0 {
		polys = orb.MultiPolygon{f.Polygon}
	}
	for _, poly := range polys {
		if cs, err := h3x.PolyfillCenter(poly, res); err == nil {
			add(cs)
		}
	}
	if c, err := h3x.FromPoint(f.Point, res); err == nil {
		add([]h3x.Cell{c})
	}
	return out
}
