package h3x

import (
	"math"
	"testing"

	"github.com/paulmach/orb"
)

var cugnaux = orb.Point{1.3444, 43.5365}

func TestDistanceM(t *testing.T) {
	toulouse := orb.Point{1.4442, 43.6045}
	d := DistanceM(cugnaux, toulouse)
	if d < 10000 || d > 12000 {
		t.Fatalf("Cugnaux to Toulouse should be ~11 km, got %.0f m", d)
	}
	if DistanceM(cugnaux, cugnaux) != 0 {
		t.Fatal("zero distance to self")
	}
}

func TestDistanceToSegmentM(t *testing.T) {
	// A north-south segment 1 degree of longitude east of the point.
	a := orb.Point{2.3444, 43.0}
	b := orb.Point{2.3444, 44.0}
	d := DistanceToSegmentM(cugnaux, a, b)
	want := DistanceM(cugnaux, orb.Point{2.3444, 43.5365})
	if math.Abs(d-want) > 200 {
		t.Fatalf("cross-track distance %.0f, want ~%.0f", d, want)
	}
	// Point beyond the end clamps to the endpoint.
	far := orb.Point{2.3444, 46.0}
	if got := DistanceToSegmentM(far, a, b); math.Abs(got-DistanceM(far, b)) > 1 {
		t.Fatalf("beyond end: %.0f", got)
	}
	behind := orb.Point{2.3444, 41.0}
	if got := DistanceToSegmentM(behind, a, b); math.Abs(got-DistanceM(behind, a)) > 1 {
		t.Fatalf("behind start: %.0f", got)
	}
}

func TestDistanceToPolygonM(t *testing.T) {
	sq := orb.Polygon{{{1.0, 43.0}, {2.0, 43.0}, {2.0, 44.0}, {1.0, 44.0}, {1.0, 43.0}}}
	if d := DistanceToPolygonM(cugnaux, sq); d != 0 {
		t.Fatalf("inside should be 0, got %f", d)
	}
	outside := orb.Point{3.0, 43.5}
	d := DistanceToPolygonM(outside, sq)
	want := DistanceM(outside, orb.Point{2.0, 43.5})
	if math.Abs(d-want) > 500 {
		t.Fatalf("outside distance %.0f want ~%.0f", d, want)
	}
}

func TestPolyfillOverlappingSmallPolygon(t *testing.T) {
	// A 40 m square: no r10 centre inside, but it still overlaps a cell.
	d := 0.00025
	sq := orb.Polygon{{
		{cugnaux[0] - d, cugnaux[1] - d}, {cugnaux[0] + d, cugnaux[1] - d},
		{cugnaux[0] + d, cugnaux[1] + d}, {cugnaux[0] - d, cugnaux[1] + d},
		{cugnaux[0] - d, cugnaux[1] - d},
	}}
	cells, err := PolyfillOverlapping(sq, ResPlace)
	if err != nil {
		t.Fatal(err)
	}
	if len(cells) == 0 {
		t.Fatal("overlapping polyfill returned nothing for a small polygon")
	}
}

func TestChildrenCount(t *testing.T) {
	c, err := FromPoint(cugnaux, ResWeight)
	if err != nil {
		t.Fatal(err)
	}
	kids, err := Children(c, ResPlace)
	if err != nil {
		t.Fatal(err)
	}
	if len(kids) != ChildrenPerLevel*ChildrenPerLevel {
		t.Fatalf("r8 cell has %d r10 children, want 49", len(kids))
	}
}

func TestLineCells(t *testing.T) {
	ls := orb.LineString{{1.30, 43.53}, {1.40, 43.53}} // ~8 km east-west
	cells, err := LineCells(ls, ResWeight, 200)
	if err != nil {
		t.Fatal(err)
	}
	if len(cells) < 8 || len(cells) > 25 {
		t.Fatalf("8 km line should cross roughly 10-20 r8 cells, got %d", len(cells))
	}
}
