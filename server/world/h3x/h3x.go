// Package h3x is the one spatial helper in the repo. Every H3 operation and
// every piece of lat/lon geometry goes through here (CLAUDE.md conventions:
// "no ad-hoc lat/lon maths"). It wraps uber/h3-go v4 and the small amount of
// spherical geometry the world build needs.
package h3x

import (
	"fmt"
	"math"
	"sort"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
	"github.com/paulmach/orb/planar"
	h3 "github.com/uber/h3-go/v4"
)

// Resolutions fixed by CLAUDE.md ("Locked decisions").
const (
	ResWeather = 5  // weather cache key
	ResWeight  = 8  // cell weight vectors, the cell record
	ResSpawn   = 9  // spawn placement, coarse
	ResPlace   = 10 // spawn placement, fine; also the exclusion set
)

// ChildrenPerLevel is the number of children a hexagon has one resolution
// down. Two levels down (r8 to r10) that is 49, which is why the exclusion
// child count in the cell record fits in a byte.
const ChildrenPerLevel = 7

// EarthRadiusM is the mean Earth radius used for all great-circle maths here.
const EarthRadiusM = 6371008.8

// Cell is an H3 cell index.
type Cell = h3.Cell

// FromLatLng returns the cell containing the point at the given resolution.
func FromLatLng(lat, lon float64, res int) (Cell, error) {
	return h3.LatLngToCell(h3.NewLatLng(lat, lon), res)
}

// FromPoint is FromLatLng for an orb.Point (lon, lat order).
func FromPoint(p orb.Point, res int) (Cell, error) {
	return FromLatLng(p.Lat(), p.Lon(), res)
}

// Center returns the cell's centre as (lat, lon).
func Center(c Cell) (lat, lon float64, err error) {
	ll, err := c.LatLng()
	if err != nil {
		return 0, 0, err
	}
	return ll.Lat, ll.Lng, nil
}

// CenterPoint returns the cell's centre as an orb.Point (lon, lat).
func CenterPoint(c Cell) (orb.Point, error) {
	lat, lon, err := Center(c)
	if err != nil {
		return orb.Point{}, err
	}
	return orb.Point{lon, lat}, nil
}

// Disk returns the cells within k steps of the origin, origin included.
func Disk(c Cell, k int) ([]Cell, error) { return c.GridDisk(k) }

// Children returns the descendants of c at the given resolution.
func Children(c Cell, res int) ([]Cell, error) { return c.Children(res) }

// Parent returns the ancestor of c at the given resolution.
func Parent(c Cell, res int) (Cell, error) { return c.Parent(res) }

// Boundary returns the cell's outline as an orb.Ring (closed).
func Boundary(c Cell) (orb.Ring, error) {
	b, err := c.Boundary()
	if err != nil {
		return nil, err
	}
	r := make(orb.Ring, 0, len(b)+1)
	for _, ll := range b {
		r = append(r, orb.Point{ll.Lng, ll.Lat})
	}
	r = append(r, r[0])
	return r, nil
}

// PolyfillOverlapping returns every cell at res that overlaps the polygon at
// any point. Small polygons that contain no cell centre still return the
// cells they touch, which is what exclusion zones need.
func PolyfillOverlapping(p orb.Polygon, res int) ([]Cell, error) {
	gp, err := toGeoPolygon(p)
	if err != nil {
		return nil, err
	}
	return h3.PolygonToCellsExperimental(gp, res, h3.ContainmentOverlapping)
}

// PolyfillCenter returns the cells at res whose centre lies inside the polygon.
func PolyfillCenter(p orb.Polygon, res int) ([]Cell, error) {
	gp, err := toGeoPolygon(p)
	if err != nil {
		return nil, err
	}
	return h3.PolygonToCells(gp, res)
}

// BBoxCells returns every cell at res overlapping the lat/lon box.
func BBoxCells(minLat, minLon, maxLat, maxLon float64, res int) ([]Cell, error) {
	ring := orb.Ring{
		{minLon, minLat}, {maxLon, minLat}, {maxLon, maxLat}, {minLon, maxLat}, {minLon, minLat},
	}
	return PolyfillOverlapping(orb.Polygon{ring}, res)
}

// LineCells returns the cells at res along a line, sampling every stepM
// metres so that no cell the line crosses is skipped when stepM is below the
// cell edge length.
func LineCells(ls orb.LineString, res int, stepM float64) ([]Cell, error) {
	seen := map[Cell]struct{}{}
	var out []Cell
	add := func(p orb.Point) error {
		c, err := FromPoint(p, res)
		if err != nil {
			return err
		}
		if _, ok := seen[c]; !ok {
			seen[c] = struct{}{}
			out = append(out, c)
		}
		return nil
	}
	for i := 0; i < len(ls); i++ {
		if err := add(ls[i]); err != nil {
			return nil, err
		}
		if i+1 >= len(ls) {
			break
		}
		d := DistanceM(ls[i], ls[i+1])
		n := int(math.Ceil(d / stepM))
		for j := 1; j < n; j++ {
			t := float64(j) / float64(n)
			if err := add(lerp(ls[i], ls[i+1], t)); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

// Contains reports whether the point (lon, lat) lies inside the polygon.
// Planar containment on lon/lat is exact enough for anything smaller than a
// hemisphere, which is everything in this repo.
func Contains(p orb.Polygon, pt orb.Point) bool { return planar.PolygonContains(p, pt) }

// DistanceM is the great-circle distance in metres between two points.
func DistanceM(a, b orb.Point) float64 {
	la1, lo1 := a.Lat()*math.Pi/180, a.Lon()*math.Pi/180
	la2, lo2 := b.Lat()*math.Pi/180, b.Lon()*math.Pi/180
	dla, dlo := la2-la1, lo2-lo1
	h := math.Sin(dla/2)*math.Sin(dla/2) + math.Cos(la1)*math.Cos(la2)*math.Sin(dlo/2)*math.Sin(dlo/2)
	return 2 * EarthRadiusM * math.Asin(math.Min(1, math.Sqrt(h)))
}

// DistanceToSegmentM is the great-circle distance from p to the geodesic
// segment ab (cross-track distance, clamped to the endpoints).
func DistanceToSegmentM(p, a, b orb.Point) float64 {
	d12 := DistanceM(a, b)
	if d12 == 0 {
		return DistanceM(p, a)
	}
	d13 := DistanceM(a, p)
	if d13 == 0 {
		return 0
	}
	t12 := bearingRad(a, b)
	t13 := bearingRad(a, p)
	dxt := math.Asin(math.Sin(d13/EarthRadiusM)*math.Sin(t13-t12)) * EarthRadiusM
	// Along-track distance; negative means p is behind a.
	cosDxt := math.Cos(dxt / EarthRadiusM)
	var dat float64
	if cosDxt == 0 {
		dat = 0
	} else {
		dat = math.Acos(clamp(math.Cos(d13/EarthRadiusM)/cosDxt, -1, 1)) * EarthRadiusM
	}
	if math.Cos(t13-t12) < 0 {
		dat = -dat
	}
	switch {
	case dat < 0:
		return d13
	case dat > d12:
		return DistanceM(p, b)
	default:
		return math.Abs(dxt)
	}
}

// DistanceToLineM is the distance from p to the nearest point of a polyline.
func DistanceToLineM(p orb.Point, ls orb.LineString) float64 {
	if len(ls) == 0 {
		return math.Inf(1)
	}
	if len(ls) == 1 {
		return DistanceM(p, ls[0])
	}
	best := math.Inf(1)
	for i := 0; i+1 < len(ls); i++ {
		if d := DistanceToSegmentM(p, ls[i], ls[i+1]); d < best {
			best = d
		}
	}
	return best
}

// DistanceToPolygonM is 0 when p is inside the polygon (and outside its
// holes), otherwise the distance to the nearest edge.
func DistanceToPolygonM(p orb.Point, poly orb.Polygon) float64 {
	if len(poly) == 0 {
		return math.Inf(1)
	}
	if Contains(poly, p) {
		return 0
	}
	best := math.Inf(1)
	for _, ring := range poly {
		if d := DistanceToLineM(p, orb.LineString(ring)); d < best {
			best = d
		}
	}
	return best
}

// DistanceToMultiPolygonM is the minimum DistanceToPolygonM over the parts.
func DistanceToMultiPolygonM(p orb.Point, mp orb.MultiPolygon) float64 {
	best := math.Inf(1)
	for _, poly := range mp {
		if d := DistanceToPolygonM(p, poly); d < best {
			best = d
		}
	}
	return best
}

// Centroid is the area-weighted planar centroid of a polygon's outer ring,
// good enough to place a marker on a building.
func Centroid(poly orb.Polygon) orb.Point {
	if len(poly) == 0 || len(poly[0]) == 0 {
		return orb.Point{}
	}
	c, _ := planar.CentroidArea(poly)
	return c
}

// SortCells sorts cells by index so the snapshot arrays are binary-searchable.
func SortCells(cells []Cell) {
	sort.Slice(cells, func(i, j int) bool { return cells[i] < cells[j] })
}

func toGeoPolygon(p orb.Polygon) (h3.GeoPolygon, error) {
	if len(p) == 0 || len(p[0]) < 4 {
		return h3.GeoPolygon{}, fmt.Errorf("h3x: polygon needs a closed outer ring with at least 3 distinct points")
	}
	gp := h3.GeoPolygon{GeoLoop: toLoop(p[0])}
	for _, hole := range p[1:] {
		gp.Holes = append(gp.Holes, toLoop(hole))
	}
	return gp, nil
}

func toLoop(r orb.Ring) h3.GeoLoop {
	n := len(r)
	if n > 1 && r[0] == r[n-1] {
		n-- // H3 loops are implicitly closed
	}
	loop := make(h3.GeoLoop, n)
	for i := 0; i < n; i++ {
		loop[i] = h3.NewLatLng(r[i].Lat(), r[i].Lon())
	}
	return loop
}

func bearingRad(a, b orb.Point) float64 {
	la1, lo1 := a.Lat()*math.Pi/180, a.Lon()*math.Pi/180
	la2, lo2 := b.Lat()*math.Pi/180, b.Lon()*math.Pi/180
	y := math.Sin(lo2-lo1) * math.Cos(la2)
	x := math.Cos(la1)*math.Sin(la2) - math.Sin(la1)*math.Cos(la2)*math.Cos(lo2-lo1)
	return math.Atan2(y, x)
}

func lerp(a, b orb.Point, t float64) orb.Point {
	return orb.Point{a[0] + (b[0]-a[0])*t, a[1] + (b[1]-a[1])*t}
}

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

// Offset returns the point at distM metres from (lat, lon) along bearingDeg
// (0 = north, clockwise).
func Offset(lat, lon, bearingDeg, distM float64) (float64, float64) {
	p := geo.PointAtBearingAndDistance(orb.Point{lon, lat}, bearingDeg, distM)
	return p.Lat(), p.Lon()
}

// ParseCell parses an H3 index string such as "8839601945fffff". The second
// result is false when the string is not a valid cell.
func ParseCell(s string) (Cell, bool) {
	c := h3.CellFromString(s)
	return c, c.IsValid()
}
