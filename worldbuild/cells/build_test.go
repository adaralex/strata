package cells

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/adaralex/strata/server/world"
	"github.com/adaralex/strata/server/world/h3x"
	"github.com/adaralex/strata/worldbuild/classify"
)

func buildFixture(t *testing.T) (*world.World, Stats) {
	t.Helper()
	root := filepath.Join("..", "..")
	civsJSON, err := os.ReadFile(filepath.Join(root, "rules", "civs.json"))
	if err != nil {
		t.Fatal(err)
	}
	civs, err := world.ParseCivs(civsJSON)
	if err != nil {
		t.Fatal(err)
	}
	rules, err := classify.LoadRules(filepath.Join(root, "rules", "poi_classes.json"))
	if err != nil {
		t.Fatal(err)
	}
	layers, err := LoadLayers(filepath.Join(root, "data", "cores"), civs)
	if err != nil {
		t.Fatal(err)
	}
	curated, err := LoadCurated(filepath.Join(root, "data", "beacons"), civs)
	if err != nil {
		t.Fatal(err)
	}
	civTags, err := LoadCivTags(filepath.Join(root, "data", "civ_tags.json"), civs)
	if err != nil {
		t.Fatal(err)
	}
	block, err := LoadBlocklist(filepath.Join(root, "data", "blocklist.geojson"))
	if err != nil {
		t.Fatal(err)
	}
	lists, err := classify.LoadLists(filepath.Join(root, "data"), rules)
	if err != nil {
		t.Fatal(err)
	}
	ex, err := classify.ReadOSM(context.Background(), filepath.Join("..", "testdata", "cugnaux.osm"), rules, nil)
	if err != nil {
		t.Fatal(err)
	}
	res := classify.Run(ex, rules, lists)
	snap, st, err := Build(Inputs{
		BuildID: "fixture", Civs: civs, Rules: rules, Layers: layers, Curated: curated, CivTags: civTags,
		Blocklist: block, Classified: res, Bounds: ex.Bounds,
	})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := snap.Write(dir, civsJSON); err != nil {
		t.Fatal(err)
	}
	w, err := world.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return w, st
}

func TestBuildFixture(t *testing.T) {
	w, st := buildFixture(t)
	if st.Cells < 10 || st.Reached != st.Cells {
		t.Fatalf("every cell in Cugnaux must be reached: %+v", st)
	}
	// Town centre: Hallstatt periphery dominant, Phoenicia via the isthmus
	// corridor second, Etruria a distant third.
	res, err := w.Lookup(43.5365, 1.3444, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Found || !res.Weights.Reached {
		t.Fatalf("centre not resolved: %+v", res)
	}
	names := func(ws []world.CivWeight) []string {
		var out []string
		for _, x := range ws {
			out = append(out, w.CivName(x.Civ))
		}
		return out
	}
	got := names(res.Weights.Top)
	if len(got) != 3 || got[0] != "hallstatt" || got[1] != "phoenicia" || got[2] != "etruria" {
		t.Fatalf("civ order %v, weights %+v", got, res.Weights.Top)
	}
	if res.Record.Cost[0] != 150 {
		t.Fatalf("Cugnaux is Hallstatt periphery, cost 150, got %d", res.Record.Cost[0])
	}
	if res.Record.Cost[1] < 250 || res.Record.Cost[1] > 280 {
		t.Fatalf("Phoenicia isthmus corridor cost should be 250 + a few km, got %d", res.Record.Cost[1])
	}
	if res.Weights.Purity < 0.4 || res.Weights.Purity > 0.6 {
		t.Fatalf("purity %f outside the expected band", res.Weights.Purity)
	}
	// Services in the centre cell.
	has := map[string]bool{}
	for _, s := range res.Services {
		has[s] = true
	}
	for _, want := range []string{"hearth", "apothecary", "vault", "scriptorium", "spring", "wardrobe"} {
		if !has[want] {
			t.Fatalf("centre cell should offer %s, got %v", want, res.Services)
		}
	}
	if res.Record.POIDensity == 0 || res.Record.UrbanBand() == 0 {
		t.Fatalf("centre should be dense: density %d urban %d", res.Record.POIDensity, res.Record.UrbanBand())
	}
	// The memorial and the church are exclusion zones: the r10 cell under
	// the church is excluded, and the record counts children.
	church, _ := w.Lookup(43.5363, 1.3438, 0)
	if !church.Excluded || church.Record.ExclChildren == 0 {
		t.Fatalf("church must be excluded: %+v", church.Record)
	}
	// The supermarket inside the school is suppressed by the zone.
	var super *world.POI
	for i := range w.Snap.POIs {
		if w.Snap.POIs[i].Ref == "n/11" {
			super = &w.Snap.POIs[i]
		}
	}
	if super == nil || super.Excluded != "zone:school" {
		t.Fatalf("supermarket in the school: %+v", super)
	}
	// Beacons: the museum with a library, the oppidum. The Résistance museum
	// and the memorial are not beacons.
	if st.Beacons != 2 {
		t.Fatalf("want 2 beacons, got %d: %+v", st.Beacons, w.Snap.Beacons)
	}
	for _, b := range w.Snap.Beacons {
		switch b.Ref {
		case "w/102":
			if b.Grade != 3 || b.Source != "cell_soil" || b.Polygon() == nil {
				t.Fatalf("museum beacon: %+v", b)
			}
		case "n/10":
			if b.Grade != 2 || b.Source != "historic:civilization" || b.Civs["hallstatt"] != 1 {
				t.Fatalf("oppidum beacon: %+v", b)
			}
		default:
			t.Fatalf("unexpected beacon %+v", b)
		}
	}
	inMuseum, _ := w.Lookup(43.5350, 1.3481, 0)
	if len(inMuseum.Beacons) != 1 || inMuseum.Beacons[0].DistanceM != 0 || inMuseum.Record.BeaconGrade != 3 {
		t.Fatalf("inside the museum: %+v grade %d", inMuseum.Beacons, inMuseum.Record.BeaconGrade)
	}
	farFromMuseum, _ := w.Lookup(43.5365, 1.3444, 0)
	if len(farFromMuseum.Beacons) != 0 {
		t.Fatal("the town centre is 300 m from the museum, outside its 60 m halo")
	}
	// Terrain: the lake cell is water band 0, the park cell is wild.
	lake, _ := w.Lookup(43.5290, 1.3352, 0)
	if lake.Record.WaterBand() != 0 {
		t.Fatalf("lake water band %d", lake.Record.WaterBand())
	}
	park, _ := w.Lookup(43.5330, 1.3422, 0)
	if !park.Record.IsWild() {
		t.Fatal("park cell should be wild")
	}
	// Raising propagation pulls Etruria in.
	night, _ := w.Lookup(43.5365, 1.3444, 1)
	if night.Weights.Top[2].Weight <= res.Weights.Top[2].Weight {
		t.Fatal("night should raise the distant civilization's share")
	}
	// Nearby lists the live services within 3 km, none of the excluded ones.
	for _, n := range w.Nearby(43.5365, 1.3444, 3000) {
		if n.Kind == "poi" && (n.POI.Ref == "n/11" || n.POI.Ref == "n/14") {
			t.Fatalf("excluded POI listed as nearby: %+v", n.POI)
		}
	}
	_ = h3x.ResWeight
}
