package classify

import (
	"context"
	"path/filepath"
	"testing"
)

func repoRules(t *testing.T) *Rules {
	t.Helper()
	r, err := LoadRules(filepath.Join("..", "..", "rules", "poi_classes.json"))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestRulesFileLoads(t *testing.T) {
	r := repoRules(t)
	if got := r.ClassIDs()[0]; got != "beacon" {
		t.Fatalf("first class (bit 0) should be beacon, got %s", got)
	}
	if r.ClassByID("hearth").RangeM() != 35 {
		t.Fatal("hearth range should be 35 m")
	}
}

func TestMatchClasses(t *testing.T) {
	r := repoRules(t)
	cases := []struct {
		tags map[string]string
		want []string
		sub  string
	}{
		{map[string]string{"shop": "bakery"}, []string{"hearth"}, "shop=bakery"},
		{map[string]string{"amenity": "pharmacy"}, []string{"apothecary"}, "amenity=pharmacy"},
		{map[string]string{"tourism": "museum", "amenity": "library"}, []string{"beacon", "scriptorium"}, "tourism=museum"},
		{map[string]string{"shop": "bicycle", "repair": "yes"}, []string{"forge"}, "shop=bicycle"},
		{map[string]string{"shop": "car_repair"}, nil, ""},
		{map[string]string{"shop": "car_repair", "repair": "yes"}, nil, ""},
		{map[string]string{"shop": "mobile_phone", "repair": "yes"}, nil, ""},
		{map[string]string{"shop": "electronics", "repair": "yes"}, []string{"forge"}, "shop=electronics"},
		{map[string]string{"shop": "bakery", "access": "private"}, nil, ""},
		{map[string]string{"leisure": "garden"}, nil, ""},
		{map[string]string{"leisure": "garden", "garden:type": "community"}, []string{"wild"}, ""},
		{map[string]string{"tourism": "information"}, nil, ""},
		{map[string]string{"tourism": "information", "information": "office"}, []string{"scriptorium"}, ""},
		{map[string]string{"highway": "residential"}, nil, ""},
		{map[string]string{"historic": "memorial"}, []string{"beacon"}, "historic=memorial"},
	}
	for _, c := range cases {
		hits := r.MatchClasses(c.tags)
		var got []string
		for _, h := range hits {
			got = append(got, h.Class.ID)
		}
		if len(got) != len(c.want) {
			t.Fatalf("%v: got %v want %v", c.tags, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("%v: got %v want %v", c.tags, got, c.want)
			}
		}
		if c.sub != "" && hits[0].Subclass != c.sub {
			t.Fatalf("%v: subclass %q want %q", c.tags, hits[0].Subclass, c.sub)
		}
	}
	if h := r.MatchClasses(map[string]string{"historic": "memorial"}); !h[0].WhitelistOnly {
		t.Fatal("memorial beacons are whitelist-only")
	}
	if h := r.MatchClasses(map[string]string{"tourism": "museum"}); h[0].Grade != 2 {
		t.Fatalf("uncurated museum grade %d, want 2", h[0].Grade)
	}
	if h := r.MatchClasses(map[string]string{"tourism": "gallery"}); h[0].Grade != 1 {
		t.Fatalf("gallery grade %d, want 1", h[0].Grade)
	}
	if h := r.MatchClasses(map[string]string{"shop": "supermarket"}); h[0].Potency != 0.4 {
		t.Fatalf("supermarket potency %f", h[0].Potency)
	}
}

func TestMatchExclusion(t *testing.T) {
	r := repoRules(t)
	if e, ok := r.MatchExclusion(map[string]string{"amenity": "school"}); !ok || e.ID != "school" || e.BufferM != 30 {
		t.Fatalf("school: %+v %v", e, ok)
	}
	if e, ok := r.MatchExclusion(map[string]string{"building": "house"}); !ok || !e.PlacementOnly {
		t.Fatalf("house must be a placement-only exclusion: %+v", e)
	}
	if _, ok := r.MatchExclusion(map[string]string{"shop": "bakery"}); ok {
		t.Fatal("bakery is not excluded")
	}
}

func TestReadAndClassifyFixture(t *testing.T) {
	rules := repoRules(t)
	ex, err := ReadOSM(context.Background(), filepath.Join("..", "testdata", "cugnaux.osm"), rules, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !ex.HasHeader || ex.Bounds.MinLat != 43.52 {
		t.Fatalf("bounds: %+v", ex.Bounds)
	}
	byRef := map[string]*Feature{}
	for i := range ex.Features {
		byRef[ex.Features[i].Ref] = &ex.Features[i]
	}
	if f := byRef["w/100"]; f == nil || f.Polygon == nil {
		t.Fatal("school way should be a polygon")
	}
	if f := byRef["w/103"]; f == nil || f.Line == nil || f.Polygon != nil {
		t.Fatal("stream should be a line")
	}
	if f := byRef["r/300"]; f == nil || len(f.Polygon) != 1 || len(f.Polygon[0]) != 5 {
		t.Fatalf("forest relation should chain into one 4-point ring, got %+v", byRef["r/300"])
	}
	if f := byRef["w/110"]; f == nil || f.Line == nil || f.Polygon != nil {
		t.Fatal("the roundabout is a walkable residential way and must stay a line, never a polygon")
	}

	lists, err := LoadLists(filepath.Join("..", "..", "data"), rules)
	if err != nil {
		t.Fatal(err)
	}
	if len(lists.BlockNames) == 0 {
		t.Fatal("blocklist names should load")
	}
	res := Run(ex, rules, lists)
	got := map[string]*Classified{}
	for i := range res.Items {
		got[res.Items[i].Feature.Ref] = &res.Items[i]
	}
	check := func(ref, class, excluded string) {
		t.Helper()
		c := got[ref]
		if c == nil {
			t.Fatalf("%s missing", ref)
		}
		if class != "" && (len(c.Hits) == 0 || c.Hits[0].Class.ID != class) {
			t.Fatalf("%s: want class %s, got %+v", ref, class, c.Hits)
		}
		if c.Excluded != excluded {
			t.Fatalf("%s: excluded %q want %q", ref, c.Excluded, excluded)
		}
	}
	check("n/1", "hearth", "")
	check("n/2", "apothecary", "")
	check("n/9", "", "memorial")
	check("n/10", "beacon", "")
	check("w/111", "", "worship")
	if c := got["n/17"]; c != nil && len(c.Hits) > 0 {
		t.Fatalf("a garage is not a forge any more: %+v", c.Hits)
	}
	if c := got["n/18"]; c != nil && len(c.Hits) > 0 {
		t.Fatalf("access=private drops the feature from every class: %+v", c.Hits)
	}
	if c := got["w/112"]; c != nil && len(c.Hits) > 0 {
		t.Fatalf("a private garden is not wild: %+v", c.Hits)
	}
	check("n/14", "beacon", "blocklist:name")
	check("n/16", "forge", "")
	check("w/100", "", "school")
	check("w/101", "wild", "")
	check("w/102", "beacon", "")
	check("w/109", "", "home")
	if len(got["w/102"].Hits) != 2 || got["w/102"].Hits[1].Class.ID != "scriptorium" {
		t.Fatalf("museum with a library stacks scriptorium: %+v", got["w/102"].Hits)
	}
	if got["w/109"].IsExclusionZone {
		t.Fatal("a house is placement-only, never in the r10 set")
	}
	if !got["w/100"].IsExclusionZone || !got["w/104"].IsExclusionZone {
		t.Fatal("school and motorway go into the r10 set")
	}
	if len(got["n/9"].Hits) != 0 {
		t.Fatal("non-whitelisted memorial must not be a beacon")
	}
	if len(got["w/103"].Terrain) != 1 || got["w/103"].Terrain[0] != "water" {
		t.Fatalf("stream is water terrain: %v", got["w/103"].Terrain)
	}
	if res.Stats.ByClass["hearth"] != 3 { // bakery, restaurant, supermarket (excluded later by geometry, not tags)
		t.Fatalf("hearth count %d", res.Stats.ByClass["hearth"])
	}
}
