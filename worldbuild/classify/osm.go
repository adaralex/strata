package classify

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"
	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmpbf"
	"github.com/paulmach/osm/osmxml"
)

// Feature is one OSM element with resolved geometry and the tags that made
// it interesting. Point is always set (the node, or a centroid). Polygon is
// set for closed ways and multipolygon relations; Line for open ways.
type Feature struct {
	Ref     string // n/123, w/456, r/789
	Tags    map[string]string
	Point   orb.Point
	Polygon orb.Polygon
	Multi   orb.MultiPolygon
	Line    orb.LineString
}

// Name is the feature's OSM name, if any.
func (f *Feature) Name() string { return f.Tags["name"] }

// BBox is a lat/lon box.
type BBox struct{ MinLat, MinLon, MaxLat, MaxLon float64 }

// Extend grows the box to include p.
func (b *BBox) Extend(p orb.Point) {
	if b.MinLat == 0 && b.MaxLat == 0 && b.MinLon == 0 && b.MaxLon == 0 {
		b.MinLat, b.MaxLat, b.MinLon, b.MaxLon = p.Lat(), p.Lat(), p.Lon(), p.Lon()
		return
	}
	if p.Lat() < b.MinLat {
		b.MinLat = p.Lat()
	}
	if p.Lat() > b.MaxLat {
		b.MaxLat = p.Lat()
	}
	if p.Lon() < b.MinLon {
		b.MinLon = p.Lon()
	}
	if p.Lon() > b.MaxLon {
		b.MaxLon = p.Lon()
	}
}

// Extract is everything the reader pulled out of one OSM file.
type Extract struct {
	Features []Feature
	// Bounds is the file header bbox when present, else the bbox of what was read.
	Bounds    BBox
	HasHeader bool
	Stats     ReadStats
}

// ReadStats counts what each pass saw.
type ReadStats struct {
	Nodes, Ways, Relations                             int64
	TaggedNodes, InterestingWays, InterestingRelations int64
	AssembledPolygons, AssembledLines                  int64
	DroppedRelations                                   int64
}

type scanner interface {
	Scan() bool
	Object() osm.Object
	Err() error
	Close() error
}

type opener func(pass string) (scanner, func(), error)

// ReadOSM streams an .osm.pbf or .osm (XML) file in three passes (relations,
// ways, nodes) so only the node coordinates the interesting features need
// are ever held in memory. progress, if non-nil, receives one line per pass.
func ReadOSM(ctx context.Context, path string, rules *Rules, progress io.Writer) (*Extract, error) {
	open, header, err := openerFor(ctx, path)
	if err != nil {
		return nil, err
	}
	ex := &Extract{}
	if header != nil && header.Bounds != nil {
		ex.Bounds = BBox{header.Bounds.MinLat, header.Bounds.MinLon, header.Bounds.MaxLat, header.Bounds.MaxLon}
		ex.HasHeader = true
	}
	log := func(format string, a ...any) {
		if progress != nil {
			_, _ = fmt.Fprintf(progress, format+"\n", a...)
		}
	}

	// Pass 1: relations. Multipolygons with interesting tags; remember their
	// outer ways.
	type rel struct {
		id     osm.RelationID
		tags   map[string]string
		outers []osm.WayID
	}
	var rels []rel
	needWay := map[osm.WayID]struct{}{}
	{
		sc, done, err := open("relations")
		if err != nil {
			return nil, err
		}
		for sc.Scan() {
			r, ok := sc.Object().(*osm.Relation)
			if !ok {
				continue
			}
			ex.Stats.Relations++
			tags := tagMap(r.Tags)
			if tags["type"] != "multipolygon" || !rules.Interesting(tags) {
				continue
			}
			ex.Stats.InterestingRelations++
			rr := rel{id: r.ID, tags: tags}
			for _, m := range r.Members {
				if m.Type == osm.TypeWay && (m.Role == "outer" || m.Role == "") {
					rr.outers = append(rr.outers, osm.WayID(m.Ref))
					needWay[osm.WayID(m.Ref)] = struct{}{}
				}
			}
			if len(rr.outers) > 0 {
				rels = append(rels, rr)
			}
		}
		err = sc.Err()
		_ = sc.Close()
		done()
		if err != nil {
			return nil, fmt.Errorf("relations pass: %w", err)
		}
		log("relations: %d scanned, %d interesting multipolygons", ex.Stats.Relations, ex.Stats.InterestingRelations)
	}

	// Pass 2: ways. Interesting ways and the ways relations need; remember
	// their nodes.
	type way struct {
		id    osm.WayID
		tags  map[string]string
		nodes []osm.NodeID
		own   bool // interesting in its own right
	}
	ways := map[osm.WayID]*way{}
	needNode := map[osm.NodeID]struct{}{}
	{
		sc, done, err := open("ways")
		if err != nil {
			return nil, err
		}
		for sc.Scan() {
			w, ok := sc.Object().(*osm.Way)
			if !ok {
				continue
			}
			ex.Stats.Ways++
			_, needed := needWay[w.ID]
			tags := tagMap(w.Tags)
			own := rules.Interesting(tags)
			if !needed && !own {
				continue
			}
			if own {
				ex.Stats.InterestingWays++
			}
			ww := &way{id: w.ID, tags: tags, own: own, nodes: make([]osm.NodeID, len(w.Nodes))}
			for i, n := range w.Nodes {
				ww.nodes[i] = n.ID
				needNode[n.ID] = struct{}{}
			}
			ways[w.ID] = ww
		}
		err = sc.Err()
		_ = sc.Close()
		done()
		if err != nil {
			return nil, fmt.Errorf("ways pass: %w", err)
		}
		log("ways: %d scanned, %d interesting, %d kept, %d nodes needed", ex.Stats.Ways, ex.Stats.InterestingWays, len(ways), len(needNode))
	}

	// Pass 3: nodes. Tagged interesting nodes become features; needed nodes
	// give coordinates.
	coords := make(map[osm.NodeID]orb.Point, len(needNode))
	var bounds BBox
	{
		sc, done, err := open("nodes")
		if err != nil {
			return nil, err
		}
		for sc.Scan() {
			obj := sc.Object()
			if b, ok := obj.(*osm.Bounds); ok && !ex.HasHeader {
				ex.Bounds = BBox{b.MinLat, b.MinLon, b.MaxLat, b.MaxLon}
				ex.HasHeader = true
				continue
			}
			n, ok := obj.(*osm.Node)
			if !ok {
				continue
			}
			ex.Stats.Nodes++
			p := orb.Point{n.Lon, n.Lat}
			if _, needed := needNode[n.ID]; needed {
				coords[n.ID] = p
				bounds.Extend(p)
			}
			if len(n.Tags) == 0 {
				continue
			}
			tags := tagMap(n.Tags)
			if !rules.Interesting(tags) {
				continue
			}
			ex.Stats.TaggedNodes++
			bounds.Extend(p)
			ex.Features = append(ex.Features, Feature{Ref: fmt.Sprintf("n/%d", n.ID), Tags: tags, Point: p})
		}
		err = sc.Err()
		_ = sc.Close()
		done()
		if err != nil {
			return nil, fmt.Errorf("nodes pass: %w", err)
		}
		log("nodes: %d scanned, %d interesting, %d coordinates kept", ex.Stats.Nodes, ex.Stats.TaggedNodes, len(coords))
	}
	if !ex.HasHeader {
		ex.Bounds = bounds
	}

	// Assemble ways.
	wayLine := func(w *way) orb.LineString {
		ls := make(orb.LineString, 0, len(w.nodes))
		for _, id := range w.nodes {
			if p, ok := coords[id]; ok {
				ls = append(ls, p)
			}
		}
		return ls
	}
	for _, w := range ways {
		if !w.own {
			continue
		}
		ls := wayLine(w)
		if len(ls) < 2 {
			continue
		}
		f := Feature{Ref: fmt.Sprintf("w/%d", w.id), Tags: w.tags}
		if len(ls) >= 4 && ls[0] == ls[len(ls)-1] && !isLinear(w.tags) {
			f.Polygon = orb.Polygon{orb.Ring(ls)}
			f.Point = centroid(f.Polygon)
			ex.Stats.AssembledPolygons++
		} else {
			f.Line = ls
			f.Point = ls[len(ls)/2]
			ex.Stats.AssembledLines++
		}
		ex.Features = append(ex.Features, f)
	}

	// Assemble relations: chain outer ways into rings.
	for _, r := range rels {
		var parts []orb.LineString
		for _, wid := range r.outers {
			if w, ok := ways[wid]; ok {
				if ls := wayLine(w); len(ls) >= 2 {
					parts = append(parts, ls)
				}
			}
		}
		rings := chainRings(parts)
		if len(rings) == 0 {
			ex.Stats.DroppedRelations++
			continue
		}
		f := Feature{Ref: fmt.Sprintf("r/%d", r.id), Tags: r.tags}
		var best orb.Polygon
		bestArea := -1.0
		for _, ring := range rings {
			poly := orb.Polygon{ring}
			f.Multi = append(f.Multi, poly)
			if a := planar.Area(poly); a > bestArea {
				bestArea, best = a, poly
			}
		}
		f.Polygon = best
		f.Point = centroid(best)
		ex.Stats.AssembledPolygons++
		ex.Features = append(ex.Features, f)
	}
	log("features: %d (%d polygons, %d lines, %d relations dropped)", len(ex.Features), ex.Stats.AssembledPolygons, ex.Stats.AssembledLines, ex.Stats.DroppedRelations)
	return ex, nil
}

func openerFor(ctx context.Context, path string) (opener, *osmpbf.Header, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".pbf":
		f, err := os.Open(path)
		if err != nil {
			return nil, nil, err
		}
		hs := osmpbf.New(ctx, f, 1)
		header, err := hs.Header()
		_ = hs.Close()
		_ = f.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("pbf header: %w", err)
		}
		open := func(pass string) (scanner, func(), error) {
			f, err := os.Open(path)
			if err != nil {
				return nil, nil, err
			}
			sc := osmpbf.New(ctx, f, runtime.NumCPU())
			switch pass {
			case "relations":
				sc.SkipNodes, sc.SkipWays = true, true
			case "ways":
				sc.SkipNodes, sc.SkipRelations = true, true
			case "nodes":
				sc.SkipWays, sc.SkipRelations = true, true
			}
			return sc, func() { _ = f.Close() }, nil
		}
		return open, header, nil
	case ".osm", ".xml":
		open := func(string) (scanner, func(), error) {
			f, err := os.Open(path)
			if err != nil {
				return nil, nil, err
			}
			return osmxml.New(ctx, f), func() { _ = f.Close() }, nil
		}
		return open, nil, nil
	default:
		return nil, nil, fmt.Errorf("unsupported OSM file %q: want .osm.pbf or .osm", path)
	}
}

func tagMap(tags osm.Tags) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	m := make(map[string]string, len(tags))
	for _, t := range tags {
		m[t.Key] = t.Value
	}
	return m
}

// isLinear says a closed way is still a line: a roundabout or circular
// railway is not a polygon. area=yes overrides.
func isLinear(tags map[string]string) bool {
	if tags["area"] == "yes" {
		return false
	}
	_, hw := tags["highway"]
	_, rw := tags["railway"]
	_, ww := tags["waterway"]
	_, bar := tags["barrier"]
	return hw || rw || ww || bar
}

func centroid(p orb.Polygon) orb.Point {
	c, _ := planar.CentroidArea(p)
	return c
}

// chainRings joins open way segments end to end into closed rings. Segments
// that never close are dropped; a multipolygon with a broken outer ring is
// better skipped than guessed at in phase 0.
func chainRings(parts []orb.LineString) []orb.Ring {
	var rings []orb.Ring
	used := make([]bool, len(parts))
	for i := range parts {
		if used[i] {
			continue
		}
		used[i] = true
		ring := append(orb.LineString{}, parts[i]...)
		for {
			if len(ring) >= 4 && ring[0] == ring[len(ring)-1] {
				rings = append(rings, orb.Ring(ring))
				break
			}
			tail := ring[len(ring)-1]
			found := false
			for j := range parts {
				if used[j] {
					continue
				}
				p := parts[j]
				switch {
				case p[0] == tail:
					ring = append(ring, p[1:]...)
				case p[len(p)-1] == tail:
					for k := len(p) - 2; k >= 0; k-- {
						ring = append(ring, p[k])
					}
				default:
					continue
				}
				used[j] = true
				found = true
				break
			}
			if !found {
				break // open ring, dropped
			}
		}
	}
	return rings
}
