// Package drift solves PLAN.md §4's drift field: for every cell on the
// planet, the cheapest path cost from each civilization's core over a
// terrain graph where land is cheap, rivers cheaper, mountains, deserts and
// ice expensive, and the sea is priced by who is sailing. Costs are static
// (§5); the record stores them and lambda moves at query time.
package drift

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
)

// Class is a terrain class per cell.
type Class uint8

// Terrain classes. Order matters only for the unit-cost table.
const (
	Ocean Class = iota
	Land
	Coast
	River
	Lake
	Mountain
	Desert
	Ice
	numClasses
)

// ClassNames in Class order, matching the keys of rules/drift.json.
var ClassNames = [numClasses]string{"ocean", "land", "coast", "river", "lake", "mountain", "desert", "ice"}

func (c Class) String() string {
	if int(c) < len(ClassNames) {
		return ClassNames[c]
	}
	return fmt.Sprintf("class(%d)", c)
}

// Grid is an equirectangular raster of terrain classes. Polygons are
// scanline-filled into it once; cell centres then read it. This is the one
// place in the repo with its own lon/lat geometry, kept inside the drift
// package because H3's polygon fill cannot take a million-vertex coastline.
type Grid struct {
	Cols, Rows int
	Step       float64 // degrees per pixel
	Data       []Class
}

// NewGrid allocates a grid with the given pixel size in degrees.
func NewGrid(stepDeg float64) *Grid {
	cols := int(math.Round(360 / stepDeg))
	rows := int(math.Round(180 / stepDeg))
	return &Grid{Cols: cols, Rows: rows, Step: stepDeg, Data: make([]Class, cols*rows)}
}

// At returns the class under a point.
func (g *Grid) At(lat, lon float64) Class {
	c, r := g.index(lat, lon)
	return g.Data[r*g.Cols+c]
}

func (g *Grid) index(lat, lon float64) (col, row int) {
	col = int(math.Floor((lon + 180) / g.Step))
	row = int(math.Floor((90 - lat) / g.Step))
	if col < 0 {
		col = 0
	}
	if col >= g.Cols {
		col = g.Cols - 1
	}
	if row < 0 {
		row = 0
	}
	if row >= g.Rows {
		row = g.Rows - 1
	}
	return col, row
}

// Fill paints class into every pixel whose centre is inside the polygon,
// even-odd rule over all rings, so holes come out unpainted. When only is
// non-nil, only pixels currently of that class are painted (a glacier is
// ice only where there is land).
func (g *Grid) Fill(poly orb.Polygon, class Class, only *Class) {
	if len(poly) == 0 {
		return
	}
	type edge struct{ x0, y0, x1, y1 float64 }
	minY, maxY := math.Inf(1), math.Inf(-1)
	var edges []edge
	for _, ring := range poly {
		for i := 0; i+1 < len(ring); i++ {
			a, b := ring[i], ring[i+1]
			if a[1] == b[1] {
				continue // horizontal edges never cross a scanline centre
			}
			edges = append(edges, edge{a[0], a[1], b[0], b[1]})
			minY = math.Min(minY, math.Min(a[1], b[1]))
			maxY = math.Max(maxY, math.Max(a[1], b[1]))
		}
	}
	if len(edges) == 0 {
		return
	}
	_, rowTop := g.index(maxY, 0)
	_, rowBot := g.index(minY, 0)
	// Bucket edges by the rows they span.
	buckets := make([][]int32, rowBot-rowTop+1)
	for ei, e := range edges {
		lo, hi := math.Min(e.y0, e.y1), math.Max(e.y0, e.y1)
		_, rTop := g.index(hi, 0)
		_, rBot := g.index(lo, 0)
		for r := rTop; r <= rBot; r++ {
			buckets[r-rowTop] = append(buckets[r-rowTop], int32(ei))
		}
	}
	var xs []float64
	for r := rowTop; r <= rowBot; r++ {
		y := 90 - (float64(r)+0.5)*g.Step
		xs = xs[:0]
		for _, ei := range buckets[r-rowTop] {
			e := edges[ei]
			lo, hi := e.y0, e.y1
			if lo > hi {
				lo, hi = hi, lo
			}
			if y < lo || y >= hi { // half-open so a vertex counts once
				continue
			}
			t := (y - e.y0) / (e.y1 - e.y0)
			xs = append(xs, e.x0+t*(e.x1-e.x0))
		}
		if len(xs) < 2 {
			continue
		}
		sort.Float64s(xs)
		for i := 0; i+1 < len(xs); i += 2 {
			c0, _ := g.index(y, xs[i])
			c1, _ := g.index(y, xs[i+1])
			// Pixel centres strictly inside the span.
			for c := c0; c <= c1; c++ {
				cx := -180 + (float64(c)+0.5)*g.Step
				if cx < xs[i] || cx > xs[i+1] {
					continue
				}
				k := r*g.Cols + c
				if only != nil && g.Data[k] != *only {
					continue
				}
				g.Data[k] = class
			}
		}
	}
}

// FillGeometry paints a Polygon or MultiPolygon.
func (g *Grid) FillGeometry(geom orb.Geometry, class Class, only *Class) {
	switch gg := geom.(type) {
	case orb.Polygon:
		g.Fill(gg, class, only)
	case orb.MultiPolygon:
		for _, p := range gg {
			g.Fill(p, class, only)
		}
	}
}

// Sources names the Natural Earth 1:10m files the terrain is built from.
// They are public domain; the tool fetches them from the GitHub mirror of
// the Natural Earth vector repository (decision record 0005).
var Sources = []string{
	"ne_10m_land",
	"ne_10m_glaciated_areas",
	"ne_10m_geography_regions_polys",
	"ne_10m_rivers_lake_centerlines",
	"ne_10m_lakes",
}

// SourceURL is where a source file is fetched from.
func SourceURL(name string) string {
	return "https://raw.githubusercontent.com/nvkelso/natural-earth-vector/master/geojson/" + name + ".geojson"
}

// Terrain is the rasterised planet plus the river lines.
type Terrain struct {
	Grid   *Grid
	Rivers []orb.LineString
	Stats  map[string]int
}

// LoadTerrain reads the Natural Earth files from dir and rasterises them at
// stepDeg. Order: land, then glaciers, mountain ranges and deserts on land,
// then lakes.
func LoadTerrain(dir string, stepDeg float64) (*Terrain, error) {
	t := &Terrain{Grid: NewGrid(stepDeg), Stats: map[string]int{}}
	land, err := readFC(dir, "ne_10m_land")
	if err != nil {
		return nil, err
	}
	for _, f := range land.Features {
		t.Grid.FillGeometry(f.Geometry, Land, nil)
		t.Stats["land"]++
	}
	onlyLand := Land
	glaciers, err := readFC(dir, "ne_10m_glaciated_areas")
	if err != nil {
		return nil, err
	}
	for _, f := range glaciers.Features {
		t.Grid.FillGeometry(f.Geometry, Ice, &onlyLand)
		t.Stats["ice"]++
	}
	regions, err := readFC(dir, "ne_10m_geography_regions_polys")
	if err != nil {
		return nil, err
	}
	for _, f := range regions.Features {
		cla, _ := f.Properties["FEATURECLA"].(string)
		if cla == "" {
			cla, _ = f.Properties["featurecla"].(string)
		}
		switch cla {
		case "Range/mtn":
			t.Grid.FillGeometry(f.Geometry, Mountain, &onlyLand)
			t.Stats["mountain"]++
		case "Desert":
			t.Grid.FillGeometry(f.Geometry, Desert, &onlyLand)
			t.Stats["desert"]++
		}
	}
	lakes, err := readFC(dir, "ne_10m_lakes")
	if err != nil {
		return nil, err
	}
	for _, f := range lakes.Features {
		t.Grid.FillGeometry(f.Geometry, Lake, nil)
		t.Stats["lake"]++
	}
	rivers, err := readFC(dir, "ne_10m_rivers_lake_centerlines")
	if err != nil {
		return nil, err
	}
	for _, f := range rivers.Features {
		switch gg := f.Geometry.(type) {
		case orb.LineString:
			t.Rivers = append(t.Rivers, gg)
		case orb.MultiLineString:
			t.Rivers = append(t.Rivers, gg...)
		}
	}
	t.Stats["river_lines"] = len(t.Rivers)
	return t, nil
}

func readFC(dir, name string) (*geojson.FeatureCollection, error) {
	path := dir + "/" + name + ".geojson"
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w (run: drift fetch -out %s)", path, err, dir)
	}
	fc, err := geojson.UnmarshalFeatureCollection(b)
	if err != nil {
		// Some NE files carry a "crs" member the strict decoder rejects.
		var loose struct {
			Features []json.RawMessage `json:"features"`
		}
		if err2 := json.Unmarshal(b, &loose); err2 != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		fc = geojson.NewFeatureCollection()
		for _, raw := range loose.Features {
			f, err := geojson.UnmarshalFeature(raw)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			fc.Append(f)
		}
	}
	return fc, nil
}
