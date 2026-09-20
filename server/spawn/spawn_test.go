package spawn

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/adaralex/strata/server/cond"
	"github.com/adaralex/strata/server/internal/fixture"
	"github.com/adaralex/strata/server/world"
	"github.com/adaralex/strata/server/world/h3x"
)

var (
	root     = fixture.Root()
	secret   = []byte("phase-zero-test-secret-0123456789")
	townHall = [2]float64{43.5365, 1.3444}
	noon     = time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
)

func newSpawner(t *testing.T) *Spawner {
	t.Helper()
	w := fixture.World(t)
	c, err := LoadContent(filepath.Join(root, "rules"), w.Civs)
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(w, c, secret)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestContentLoads(t *testing.T) {
	s := newSpawner(t)
	if len(s.Content.Bestiary.Monsters) < 20 {
		t.Fatalf("bestiary has %d monsters", len(s.Content.Bestiary.Monsters))
	}
	if s.Content.Bestiary.ByID("hallstatt.tectosage_warden") == nil {
		t.Fatal("Tectosage warden missing")
	}
	if len(s.Content.Items.Archetypes) != 9 {
		t.Fatalf("archetypes: %d", len(s.Content.Items.Archetypes))
	}
}

func TestPredicates(t *testing.T) {
	facts := &cellFacts{cr: &CellResult{
		Cond:   cond.Vector{IsNight: true, Phase: cond.Night, PhaseName: "night", MoonIllum: 0.9, Propagation: 0.9, Weather: "fair"},
		Record: world.Record{TerrainFlags: 0b0000_1001, BeaconGrade: 2}, // water band 1, wild
	}}
	cases := map[string]bool{
		`{}`:                                     true,
		`{"is_night": true}`:                     true,
		`{"is_night": false}`:                    false,
		`{"phase": "night"}`:                     true,
		`{"phase": {"in": ["day", "twilight"]}}`: false,
		`{"water_band": {"lte": 1}}`:             true,
		`{"all": [{"wild": true}, {"beacon_grade": {"gte": 2}}]}`:         true,
		`{"any": [{"wild": false}, {"moon_illumination": {"gte": 0.8}}]}`: true,
		`{"urban_band": {"gte": 2}}`:                                      false,
		`{"weather": "fair", "storm": false}`:                             true,
		`{"nonexistent": 1}`:                                              false,
	}
	for src, want := range cases {
		p, err := CompilePredicate([]byte(src))
		if err != nil {
			t.Fatalf("%s: %v", src, err)
		}
		if got := p(facts); got != want {
			t.Fatalf("%s: got %v want %v", src, got, want)
		}
	}
	if _, err := CompilePredicate([]byte(`{"x": {"between": 1}}`)); err == nil {
		t.Fatal("unknown operator must fail to compile")
	}
}

func TestDeterministic(t *testing.T) {
	s := newSpawner(t)
	cell, _ := h3x.FromLatLng(townHall[0], townHall[1], h3x.ResWeight)
	a, va, err := s.Spawns(cell, noon)
	if err != nil {
		t.Fatal(err)
	}
	b, _, _ := s.Spawns(cell, noon.Add(7*time.Minute)) // same epoch
	if len(a) == 0 {
		t.Fatal("town centre must spawn something at noon")
	}
	if len(a) != len(b) {
		t.Fatalf("two devices, same cell and epoch: %d vs %d spawns", len(a), len(b))
	}
	for i := range a {
		ja, _ := json.Marshal(a[i])
		jb, _ := json.Marshal(b[i])
		if string(ja) != string(jb) {
			t.Fatalf("spawn %d differs between devices:\n%s\n%s", i, ja, jb)
		}
	}
	// A different epoch changes the set.
	c, _, _ := s.Spawns(cell, noon.Add(15*time.Minute))
	same := len(c) == len(a)
	if same {
		for i := range a {
			if a[i].ID != c[i].ID || a[i].Kind != c[i].Kind || a[i].Lat != c[i].Lat {
				same = false
				break
			}
		}
	}
	if same {
		t.Fatal("next epoch produced identical spawns")
	}
	// A different secret changes the set.
	other, _ := New(s.World, s.Content, []byte("another-secret-that-is-long-enough"))
	d, _, _ := other.Spawns(cell, noon)
	if len(d) == len(a) && d[0].ID == a[0].ID {
		t.Fatal("different secret produced the same spawn ids")
	}
	if va.Digest() == 0 {
		t.Fatal("digest")
	}
}

func TestSpawnShape(t *testing.T) {
	s := newSpawner(t)
	cell, _ := h3x.FromLatLng(townHall[0], townHall[1], h3x.ResWeight)
	r := s.Content.Rules
	seen := map[string]int{}
	total := 0
	for e := int64(0); e < 40; e++ {
		sps, _, err := s.Spawns(cell, noon.Add(time.Duration(e)*15*time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		if len(sps) > r.Count.Max {
			t.Fatalf("count %d above max", len(sps))
		}
		for _, sp := range sps {
			total++
			seen[sp.Civ]++
			if sp.Civ != "hallstatt" && sp.Civ != "phoenicia" && sp.Civ != "etruria" {
				t.Fatalf("unexpected civ %s", sp.Civ)
			}
			if !strings.HasPrefix(sp.Kind, sp.Civ+".") {
				t.Fatalf("monster %s does not belong to %s", sp.Kind, sp.Civ)
			}
			if sp.Tier < 0 || sp.Tier > 4 || sp.TierName == "" {
				t.Fatalf("tier %d %q", sp.Tier, sp.TierName)
			}
			if sp.Authenticity != Grounded && sp.Authenticity != Drift && sp.Authenticity != Confluence {
				t.Fatalf("authenticity %q at spawn time", sp.Authenticity)
			}
			// Placement: inside the r8 cell, never on an excluded r10 cell.
			r10, _ := h3x.FromLatLng(sp.Lat, sp.Lon, h3x.ResPlace)
			if s.World.Snap.IsExcluded(uint64(r10)) {
				t.Fatalf("spawn placed on an excluded r10 cell: %+v", sp)
			}
			place, _ := h3x.ParseCell(sp.PlaceCell)
			plat, plon, _ := h3x.Center(place)
			if d := h3x.DistanceM(pt(plon, plat), pt(sp.Lon, sp.Lat)); d > r.Placement.JitterM+1 {
				t.Fatalf("spawn %.0f m from its place cell centre", d)
			}
			if len(sp.ID) != 16 {
				t.Fatalf("id %q", sp.ID)
			}
		}
	}
	if total < 60 {
		t.Fatalf("40 epochs produced only %d spawns", total)
	}
	// Hallstatt dominates Cugnaux soil, Phoenicia is second, Etruria present.
	if seen["hallstatt"] <= seen["phoenicia"] || seen["phoenicia"] <= seen["etruria"] || seen["etruria"] == 0 {
		t.Fatalf("civ mix over 40 epochs: %v", seen)
	}
}

func TestNightChangesTheBestiary(t *testing.T) {
	s := newSpawner(t)
	cell, _ := h3x.FromLatLng(townHall[0], townHall[1], h3x.ResWeight)
	kinds := func(t0 time.Time) map[string]bool {
		out := map[string]bool{}
		for e := int64(0); e < 30; e++ {
			sps, _, _ := s.Spawns(cell, t0.Add(time.Duration(e)*15*time.Minute))
			for _, sp := range sps {
				out[sp.Kind] = true
			}
		}
		return out
	}
	day := kinds(time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC))
	night := kinds(time.Date(2026, 6, 20, 22, 30, 0, 0, time.UTC))
	if day["etruria.charun"] {
		t.Fatal("Charun is night-only")
	}
	if !night["etruria.charun"] && !night["hallstatt.carnyx_howler"] && !night["phoenicia.glass_wraith"] {
		t.Fatalf("no night-gated monster appeared in 30 night epochs: %v", night)
	}
	if !day["phoenicia.cocagne_shade"] && !day["hallstatt.tantugou"] && !day["phoenicia.tanit_sentinel"] {
		t.Fatalf("no day-gated monster appeared in 30 day epochs: %v", day)
	}
}

func TestShiftTiers(t *testing.T) {
	base := []float64{0.55, 0.28, 0.12, 0.04, 0.01}
	up := shiftTiers(base, 2)
	if up[0] >= base[0] || up[4] <= base[4] {
		t.Fatalf("shift up: %v", up)
	}
	down := shiftTiers(base, -1)
	if down[0] <= base[0] {
		t.Fatalf("shift down: %v", down)
	}
	var sum float64
	for _, p := range up {
		sum += p
	}
	if sum < 0.999 || sum > 1.001 {
		t.Fatalf("mass not conserved: %f", sum)
	}
}

func TestCollapse(t *testing.T) {
	s := newSpawner(t)
	cell, _ := h3x.FromLatLng(townHall[0], townHall[1], h3x.ResWeight)
	sps, v, err := s.Spawns(cell, noon)
	if err != nil || len(sps) == 0 {
		t.Fatal(err, len(sps))
	}
	// Find a spawn that is not inside an exclusion zone at its own position.
	var sp *Spawn
	for i := range sps {
		res, _ := s.World.Lookup(sps[i].Lat, sps[i].Lon, 0)
		if !res.Excluded {
			sp = &sps[i]
			break
		}
	}
	if sp == nil {
		t.Skip("every fixture spawn landed in a zone this epoch")
	}
	st := NewMemoryState()
	req := CollapseRequest{Device: "dev-a", SpawnID: sp.ID, Cell: cell.String(), Epoch: v.Epoch, Lat: sp.Lat, Lon: sp.Lon, At: noon.Add(3 * time.Minute)}
	res, err := s.Collapse(req, st)
	if err != nil {
		t.Fatalf("collapse: %v", err)
	}
	if res.Item.Name == "" || res.Item.Civ == "" || res.Item.Fiction == "" {
		t.Fatalf("item: %+v", res.Item)
	}
	if res.Item.Kind == "gear" && res.Item.Authenticity == Drift && res.Item.Fact != "" {
		t.Fatal("a hybrid must carry no fact")
	}
	if res.Item.Kind == "gear" && res.Item.Authenticity != Drift && res.Item.Fact == "" {
		t.Fatalf("a grounded item carries a fact: %+v", res.Item)
	}
	if len(res.Item.Affixes) != s.Content.Rules.Drops.AffixCountByTier[res.Item.Tier] {
		t.Fatalf("affix count %d for tier %d", len(res.Item.Affixes), res.Item.Tier)
	}
	// The same device retrying gets "already claimed"; another device gets
	// a different but equally deterministic drop.
	if _, err := s.Collapse(req, st); err != ErrClaimed {
		t.Fatalf("second claim: %v", err)
	}
	again := s.Drop(*sp, "dev-a", nil, noon)
	if again.ID != res.Item.ID || again.Name != res.Item.Name {
		t.Fatal("drop must be deterministic per device")
	}
	reqB := req
	reqB.Device = "dev-b"
	resB, err := s.Collapse(reqB, st)
	if err != nil {
		t.Fatal(err)
	}
	if resB.Item.ID == res.Item.ID {
		t.Fatal("two devices should not draw the same item id")
	}
	// Too far.
	far := req
	far.Device, far.Lat = "dev-c", sp.Lat+0.002
	if _, err := s.Collapse(far, st); err == nil || !strings.Contains(err.Error(), ErrTooFar.Error()) {
		t.Fatalf("200 m away: %v", err)
	}
	// Wrong id, expired epoch, future epoch.
	bad := req
	bad.Device, bad.SpawnID = "dev-d", "0000000000000000"
	if _, err := s.Collapse(bad, st); err != ErrNotFound {
		t.Fatalf("bad id: %v", err)
	}
	old := req
	old.Device, old.At = "dev-e", noon.Add(45*time.Minute)
	if _, err := s.Collapse(old, st); err != ErrExpired {
		t.Fatalf("expired: %v", err)
	}
	// One epoch of grace is fine.
	grace := req
	grace.Device, grace.At = "dev-f", noon.Add(20*time.Minute)
	if _, err := s.Collapse(grace, st); err != nil {
		t.Fatalf("grace epoch: %v", err)
	}
	future := req
	future.Device, future.Epoch = "dev-g", v.Epoch+3
	if _, err := s.Collapse(future, st); err != ErrEpochAhead {
		t.Fatalf("future: %v", err)
	}
}

func TestSpeedGate(t *testing.T) {
	st := NewMemoryState()
	t0 := noon
	if st.Observe("d", 43.5365, 1.3444, t0, 15, 20) {
		t.Fatal("first fix never gates")
	}
	// 1 km in 60 s = 60 km/h.
	if !st.Observe("d", 43.5455, 1.3444, t0.Add(60*time.Second), 15, 20) {
		t.Fatal("60 km/h must gate")
	}
	// Still gated on a fix too soon to judge.
	if !st.Observe("d", 43.5456, 1.3444, t0.Add(65*time.Second), 15, 20) {
		t.Fatal("gate holds until a judgeable pair arrives")
	}
	// 50 m in 60 s = 3 km/h clears it.
	if st.Observe("d", 43.5460, 1.3444, t0.Add(125*time.Second), 15, 20) {
		t.Fatal("walking pace clears the gate")
	}
}

func TestAccessionedDrop(t *testing.T) {
	s := newSpawner(t)
	cell, _ := h3x.FromLatLng(43.5350, 1.3481, h3x.ResWeight) // the fixture museum
	sps, _, _ := s.Spawns(cell, noon)
	if len(sps) == 0 {
		t.Skip("no spawn in the museum cell this epoch")
	}
	var museum *world.Beacon
	for i := range s.World.Snap.Beacons {
		if s.World.Snap.Beacons[i].Ref == "w/102" {
			museum = &s.World.Snap.Beacons[i]
		}
	}
	item := s.Drop(sps[0], "dev", museum, noon)
	if item.Authenticity != Accessioned || item.Provenance.Kind != "museum" || item.Provenance.Name != "Musée test de Cugnaux" {
		t.Fatalf("accessioned drop: %+v", item)
	}
	var site *world.Beacon
	for i := range s.World.Snap.Beacons {
		if s.World.Snap.Beacons[i].Ref == "n/10" {
			site = &s.World.Snap.Beacons[i]
		}
	}
	sited := s.Drop(sps[0], "dev", site, noon)
	if sited.Authenticity != Sited || sited.Civ != "hallstatt" {
		t.Fatalf("sited drop at the oppidum: %+v", sited)
	}
}

func BenchmarkSpawns(b *testing.B) {
	t := &testing.T{}
	s := newSpawner(t)
	cell, _ := h3x.FromLatLng(townHall[0], townHall[1], h3x.ResWeight)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := s.Spawns(cell, noon); err != nil {
			b.Fatal(err)
		}
	}
}

func TestNightMakesHybridsLikelyNotUniversal(t *testing.T) {
	s := newSpawner(t)
	cell, _ := h3x.FromLatLng(townHall[0], townHall[1], h3x.ResWeight)
	count := func(t0 time.Time) (drift, total int) {
		for e := int64(0); e < 40; e++ {
			sps, _, _ := s.Spawns(cell, t0.Add(time.Duration(e)*15*time.Minute))
			for _, sp := range sps {
				total++
				if sp.Authenticity == Drift {
					drift++
					if sp.Secondary == "" || sp.Secondary == sp.Civ {
						t.Fatalf("hybrid without a partner: %+v", sp)
					}
				} else if sp.Secondary != "" {
					t.Fatalf("non-hybrid with a partner: %+v", sp)
				}
			}
		}
		return
	}
	dd, dt := count(time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC))
	nd, nt := count(time.Date(2026, 6, 20, 22, 30, 0, 0, time.UTC))
	dayFrac, nightFrac := float64(dd)/float64(dt), float64(nd)/float64(nt)
	if !(dayFrac > 0.05 && dayFrac < 0.6) {
		t.Fatalf("day drift fraction %.2f should be a minority", dayFrac)
	}
	if !(nightFrac > dayFrac && nightFrac < 0.95) {
		t.Fatalf("night drift fraction %.2f should rise above day %.2f without becoming universal", nightFrac, dayFrac)
	}
}

func TestRanks(t *testing.T) {
	s := newSpawner(t)
	r := s.Content.Rules
	centre, _ := h3x.FromLatLng(townHall[0], townHall[1], h3x.ResWeight)
	disk, _ := h3x.Disk(centre, 1)
	counts := map[string]int{}
	bossCells := 0
	for e := int64(0); e < 60; e++ {
		at := noon.Add(time.Duration(e) * 15 * time.Minute)
		bossesInDisk := map[h3x.Cell]bool{}
		for _, c := range disk {
			sps, _, err := s.Spawns(c, at)
			if err != nil {
				t.Fatal(err)
			}
			elites, bosses := 0, 0
			for _, sp := range sps {
				counts[sp.Rank]++
				m := s.Content.Bestiary.ByID(sp.Kind)
				if m.Rank != sp.Rank {
					t.Fatalf("spawn rank %s but monster %s is %s", sp.Rank, sp.Kind, m.Rank)
				}
				switch sp.Rank {
				case RankBoss:
					bosses++
					if sp.Tier < r.Ranks.Boss.TierFloor {
						t.Fatalf("boss below tier floor: %+v", sp)
					}
				case RankElite:
					elites++
					if sp.Tier < r.Ranks.Elite.TierFloor {
						t.Fatalf("elite below tier floor: %+v", sp)
					}
				}
			}
			if elites > r.Ranks.Elite.MaxPerCell || bosses > 1 {
				t.Fatalf("cell %s epoch %d: %d elites, %d bosses", c, e, elites, bosses)
			}
			if bosses == 1 {
				bossesInDisk[c] = true
				if c == centre {
					bossCells++
				}
			}
		}
		// No two adjacent cells hold a boss in the same epoch.
		for c := range bossesInDisk {
			ring, _ := c.GridDisk(1)
			for _, n := range ring {
				if n != c && bossesInDisk[n] {
					t.Fatalf("epoch %d: adjacent cells %s and %s both hold a boss", e, c, n)
				}
			}
		}
	}
	if counts[RankCommon] <= counts[RankElite]+counts[RankBoss] {
		t.Fatalf("commons must be the bulk: %v", counts)
	}
	if counts[RankBoss] == 0 || counts[RankElite] == 0 {
		t.Fatalf("expected some elites and bosses over 60 epochs: %v", counts)
	}
	if bossCells > 30 {
		t.Fatalf("centre cell held a boss in %d of 60 epochs; bosses should be rare", bossCells)
	}
}
