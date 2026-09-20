// Package cells builds the H3 r8 cell snapshot from classified OSM features
// and the hand-drawn civilization layers (PLAN.md §4 layers 1-4, §14 step 4).
package cells

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"

	"github.com/adaralex/strata/server/world"
	"github.com/adaralex/strata/server/world/h3x"
)

// Default layer costs in km-equivalent when a feature sets none.
const (
	costCore      = 0
	costPeriphery = 150
	costCorridor  = 300
)

// Layer is one feature of data/cores/*.geojson: a core or periphery polygon
// or a corridor line, with the cost of standing on it.
type Layer struct {
	Civ    uint8
	Kind   string
	Name   string
	CostKm float64
	Multi  orb.MultiPolygon
	Line   orb.LineString
}

// CostKmAt is the layer's cost at a point: its base cost plus the distance to
// the feature. Track 1 uses great-circle distance; track 2 replaces this with
// the terrain-aware Dijkstra cost and nothing downstream changes.
func (l *Layer) CostKmAt(p orb.Point) float64 {
	var d float64
	if l.Line != nil {
		d = h3x.DistanceToLineM(p, l.Line)
	} else {
		d = h3x.DistanceToMultiPolygonM(p, l.Multi)
	}
	return l.CostKm + d/1000
}

// LoadLayers reads every *.geojson in dir.
func LoadLayers(dir string, civs *world.Civs) ([]Layer, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.geojson"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	var out []Layer
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		fc, err := geojson.UnmarshalFeatureCollection(b)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		for i, f := range fc.Features {
			l, err := layerFromFeature(f, civs)
			if err != nil {
				return nil, fmt.Errorf("%s feature %d: %w", p, i, err)
			}
			out = append(out, l)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no civilization layers found in %s", dir)
	}
	return out, nil
}

func layerFromFeature(f *geojson.Feature, civs *world.Civs) (Layer, error) {
	key, _ := f.Properties["civ"].(string)
	id, ok := civs.ByKey(key)
	if !ok {
		return Layer{}, fmt.Errorf("unknown civ %q", key)
	}
	kind, _ := f.Properties["kind"].(string)
	l := Layer{Civ: id, Kind: kind}
	l.Name, _ = f.Properties["name"].(string)
	switch kind {
	case "core":
		l.CostKm = costCore
	case "periphery":
		l.CostKm = costPeriphery
	case "corridor":
		l.CostKm = costCorridor
	default:
		return Layer{}, fmt.Errorf("kind must be core, periphery or corridor, got %q", kind)
	}
	if c, ok := f.Properties["cost_km"].(float64); ok {
		l.CostKm = c
	}
	switch g := f.Geometry.(type) {
	case orb.Polygon:
		l.Multi = orb.MultiPolygon{g}
	case orb.MultiPolygon:
		l.Multi = g
	case orb.LineString:
		l.Line = g
	default:
		return Layer{}, fmt.Errorf("geometry must be Polygon, MultiPolygon or LineString, got %T", f.Geometry)
	}
	if kind == "corridor" && l.Line == nil {
		return Layer{}, fmt.Errorf("a corridor must be a LineString")
	}
	return l, nil
}

// Curated is one entry of data/beacons/*.json.
type Curated struct {
	Match struct {
		Ref          string `json:"ref"`
		NameContains string `json:"name_contains"`
	} `json:"match"`
	Grade  uint8              `json:"grade"`
	Civs   map[string]float64 `json:"civs"`
	Note   string             `json:"note"`
	Verify bool               `json:"verify"`
}

// LoadCurated reads every *.json in dir.
func LoadCurated(dir string, civs *world.Civs) ([]Curated, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	var out []Curated
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		var f struct {
			Beacons []Curated `json:"beacons"`
		}
		if err := json.Unmarshal(b, &f); err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		for _, c := range f.Beacons {
			if c.Match.Ref == "" && c.Match.NameContains == "" {
				return nil, fmt.Errorf("%s: curated beacon needs match.ref or match.name_contains", p)
			}
			for k := range c.Civs {
				if _, ok := civs.ByKey(k); !ok {
					return nil, fmt.Errorf("%s: unknown civ %q", p, k)
				}
			}
			out = append(out, c)
		}
	}
	return out, nil
}

// Find returns the curated entry matching a beacon by ref or name.
func findCurated(list []Curated, ref, name string) *Curated {
	lname := strings.ToLower(name)
	for i := range list {
		c := &list[i]
		if c.Match.Ref != "" && c.Match.Ref == ref {
			return c
		}
		if c.Match.NameContains != "" && lname != "" && strings.Contains(lname, strings.ToLower(c.Match.NameContains)) {
			return c
		}
	}
	return nil
}

// CivTags maps OSM tag values to civilization keys (data/civ_tags.json).
type CivTags map[string]map[string]string

// LoadCivTags reads data/civ_tags.json; a missing file is an empty map.
func LoadCivTags(path string, civs *world.Civs) (CivTags, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return CivTags{}, nil
	}
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	out := CivTags{}
	for k, v := range raw {
		if strings.HasPrefix(k, "$") {
			continue
		}
		var m map[string]string
		if err := json.Unmarshal(v, &m); err != nil {
			return nil, fmt.Errorf("%s: %s: %w", path, k, err)
		}
		for tagValue, civ := range m {
			if _, ok := civs.ByKey(civ); !ok {
				return nil, fmt.Errorf("%s: %s=%s maps to unknown civ %q", path, k, tagValue, civ)
			}
		}
		out[k] = m
	}
	return out, nil
}

// Resolve returns the civ key a tag set points at, if any.
func (ct CivTags) Resolve(tags map[string]string) (string, bool) {
	for key, m := range ct {
		if v, ok := tags[key]; ok {
			if civ, ok := m[strings.ToLower(v)]; ok {
				return civ, true
			}
		}
	}
	return "", false
}

// LoadBlocklist reads data/blocklist.geojson polygons; missing file = none.
func LoadBlocklist(path string) ([]orb.Polygon, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	fc, err := geojson.UnmarshalFeatureCollection(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	var out []orb.Polygon
	for _, f := range fc.Features {
		switch g := f.Geometry.(type) {
		case orb.Polygon:
			out = append(out, g)
		case orb.MultiPolygon:
			out = append(out, g...)
		}
	}
	return out, nil
}
