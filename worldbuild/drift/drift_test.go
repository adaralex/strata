package drift

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/paulmach/orb"

	"github.com/adaralex/strata/server/world"
	"github.com/adaralex/strata/server/world/h3x"
	"github.com/adaralex/strata/worldbuild/cells"
)

func TestGridFill(t *testing.T) {
	g := NewGrid(0.5)
	// A square with a hole.
	sq := orb.Polygon{
		{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {0, 0}},
		{{4, 4}, {6, 4}, {6, 6}, {4, 6}, {4, 4}},
	}
	g.Fill(sq, Land, nil)
	if g.At(2, 2) != Land || g.At(8, 8) != Land {
		t.Fatal("inside the square should be land")
	}
	if g.At(5, 5) != Ocean {
		t.Fatal("the hole should stay ocean")
	}
	if g.At(-2, 2) != Ocean || g.At(2, 12) != Ocean {
		t.Fatal("outside stays ocean")
	}
	only := Land
	g.Fill(orb.Polygon{{{-5, 1}, {3, 1}, {3, 3}, {-5, 3}, {-5, 1}}}, Ice, &only)
	if g.At(2, 2) != Ice {
		t.Fatal("ice paints over land")
	}
	if g.At(2, -3) != Ocean {
		t.Fatal("ice must not paint over ocean when restricted to land")
	}
}

// syntheticPlanet is a small r5 region: land everywhere except an ocean
// band between two longitudes, with one river running east-west.
func syntheticPlanet(t *testing.T) *Planet {
	t.Helper()
	g := NewGrid(0.1)
	g.Fill(orb.Polygon{{{-10, 30}, {10, 30}, {10, 50}, {-10, 50}, {-10, 30}}}, Land, nil)
	g.Fill(orb.Polygon{{{-2, 30}, {2, 30}, {2, 50}, {-2, 50}, {-2, 30}}}, Ocean, nil)
	terr := &Terrain{Grid: g, Rivers: []orb.LineString{{{-9, 45}, {-3, 45}}}}
	cs, err := h3x.BBoxCells(31, -9, 49, 9, 5)
	if err != nil {
		t.Fatal(err)
	}
	h3x.SortCells(cs)
	p, err := BuildPlanet(5, cs, terr, nil)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func testRules(t *testing.T) (*Rules, *world.Civs) {
	t.Helper()
	root := filepath.Join("..", "..")
	r, err := LoadRules(filepath.Join(root, "rules", "drift.json"))
	if err != nil {
		t.Fatal(err)
	}
	civs, err := world.LoadCivs(filepath.Join(root, "rules", "civs.json"))
	if err != nil {
		t.Fatal(err)
	}
	return r, civs
}

func TestPlanetClasses(t *testing.T) {
	p := syntheticPlanet(t)
	counts := p.ClassCounts()
	for _, want := range []string{"land", "ocean", "coast", "river"} {
		if counts[want] == 0 {
			t.Fatalf("synthetic planet has no %s cells: %v", want, counts)
		}
	}
	// Coast cells are land cells next to ocean.
	for i, c := range p.Class {
		if c != Coast {
			continue
		}
		ocean := false
		for k := 0; k < 6; k++ {
			if j := p.Nbr[i*6+k]; j >= 0 && p.Class[j] == Ocean {
				ocean = true
			}
		}
		if !ocean {
			t.Fatal("a coast cell must touch ocean")
		}
	}
}

func TestDijkstraTerrain(t *testing.T) {
	p := syntheticPlanet(t)
	r, civs := testRules(t)
	// Source: one land cell far west.
	src, _ := h3x.FromLatLng(40, -8, 5)
	sources := []Source{{Idx: p.Index(src), CostKm: 0}}
	if sources[0].Idx < 0 {
		t.Fatal("source not in planet")
	}
	hallstatt, _ := civs.ByKey("hallstatt")
	phoenicia, _ := civs.ByKey("phoenicia")
	land := Dijkstra(p, sources, r.UnitCosts(civs.List[hallstatt]))
	sea := Dijkstra(p, sources, r.UnitCosts(civs.List[phoenicia]))
	if land[sources[0].Idx] != 0 {
		t.Fatal("source cost must be 0")
	}
	near, _ := h3x.FromLatLng(40, -7, 5)
	far, _ := h3x.FromLatLng(40, -4, 5)
	if !(land[p.Index(near)] < land[p.Index(far)]) {
		t.Fatalf("cost must grow with distance: %.0f vs %.0f", land[p.Index(near)], land[p.Index(far)])
	}
	// Straight-line distance on flat land ~ 1.0 unit per km, within the
	// hexagonal detour factor.
	d := haversineKm(40, -8, 40, -4)
	if got := float64(land[p.Index(far)]); got < d*0.95 || got > d*1.35 {
		t.Fatalf("land cost %.0f for %.0f km", got, d)
	}
	// Across the ocean band a landlocked civilization pays much more than
	// a maritime one.
	east, _ := h3x.FromLatLng(40, 6, 5)
	if !(sea[p.Index(east)] < land[p.Index(east)]*0.6) {
		t.Fatalf("maritime crossing should be far cheaper: sea %.0f land %.0f", sea[p.Index(east)], land[p.Index(east)])
	}
	// Along the river is cheaper than parallel to it on plain land.
	onRiver, _ := h3x.FromLatLng(45, -4, 5)
	offRiver, _ := h3x.FromLatLng(35, -4, 5)
	riverSrc, _ := h3x.FromLatLng(45, -8, 5)
	plainSrc, _ := h3x.FromLatLng(35, -8, 5)
	dr := Dijkstra(p, []Source{{Idx: p.Index(riverSrc)}}, r.UnitCosts(civs.List[hallstatt]))
	dp := Dijkstra(p, []Source{{Idx: p.Index(plainSrc)}}, r.UnitCosts(civs.List[hallstatt]))
	if !(dr[p.Index(onRiver)] < dp[p.Index(offRiver)]*0.9) {
		t.Fatalf("river valley should be cheaper: river %.0f plain %.0f", dr[p.Index(onRiver)], dp[p.Index(offRiver)])
	}
	// Unreached cells are +Inf: an isolated planet subset never touches them.
	if !math.IsInf(float64(Dijkstra(p, nil, r.UnitCosts(civs.List[hallstatt]))[0]), 1) {
		t.Fatal("no sources means everything unreached")
	}
}

func TestSolveAndField(t *testing.T) {
	p := syntheticPlanet(t)
	r, civs := testRules(t)
	// Layers: the real Hallstatt and Phoenicia layers intersect this region;
	// give every other civilization a synthetic core so all fifteen solve.
	root := filepath.Join("..", "..")
	layers, err := cells.LoadLayers(filepath.Join(root, "data", "cores"), civs)
	if err != nil {
		t.Fatal(err)
	}
	for id := 0; id < world.NumCivs; id++ {
		has := false
		for _, l := range layers {
			if l.Civ == uint8(id) {
				for _, poly := range l.Multi {
					if cs, _ := h3x.PolyfillOverlapping(poly, 5); len(cs) > 0 {
						for _, c := range cs {
							if p.Index(c) >= 0 {
								has = true
							}
						}
					}
				}
			}
		}
		if !has {
			lon := -9 + float64(id)
			layers = append(layers, cells.Layer{Civ: uint8(id), Kind: "core", Multi: orb.MultiPolygon{{{{lon, 31}, {lon + 0.5, 31}, {lon + 0.5, 31.5}, {lon, 31.5}, {lon, 31}}}}})
		}
	}
	field, err := Solve(p, r, civs, layers, nil)
	if err != nil {
		t.Fatal(err)
	}
	pairs, _ := field.Unreached()
	if pairs != 0 {
		t.Fatalf("%d unreached pairs in a connected region", pairs)
	}
	path := filepath.Join(t.TempDir(), "f.bin")
	if err := field.Write(path); err != nil {
		t.Fatal(err)
	}
	back, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if back.Res != 5 || len(back.Keys) != len(field.Keys) || back.Costs[100] != field.Costs[100] {
		t.Fatal("field round trip")
	}
	costs, ok := back.At(43.5365, 1.3444)
	if !ok {
		t.Fatal("Cugnaux is inside the synthetic region")
	}
	hal, _ := civs.ByKey("hallstatt")
	if costs[hal] > 400 || costs[hal] < 100 {
		t.Fatalf("Cugnaux Hallstatt drift cost %.0f; periphery starts at 150", costs[hal])
	}
	rec := RecordFromCosts(costs, civs)
	if rec.Civ[0] != hal || rec.ResidualCost == world.CostUnreached {
		t.Fatalf("record: %+v", rec)
	}
	w := world.Derive(&rec, civs, 0)
	if !w.Reached || w.Residual <= 0 {
		t.Fatalf("all fifteen present means a residual: %+v", w)
	}
	if _, ok := back.At(-40, 100); ok {
		t.Fatal("outside the region must report false")
	}
	// Interpolation is continuous: two points 2 km apart differ by little.
	a, _ := back.At(43.50, 1.30)
	b, _ := back.At(43.52, 1.30)
	if math.Abs(a[hal]-b[hal]) > 30 {
		t.Fatalf("interpolation jump %.0f over 2 km", math.Abs(a[hal]-b[hal]))
	}
}
