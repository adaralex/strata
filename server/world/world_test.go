package world

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/adaralex/strata/server/world/h3x"
)

func loadCivs(t *testing.T) (*Civs, []byte) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "rules", "civs.json"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := ParseCivs(b)
	if err != nil {
		t.Fatal(err)
	}
	return c, b
}

func TestRecordRoundTrip(t *testing.T) {
	r := Record{
		Civ: [TopK]uint8{6, 4, 0, NoCiv}, Cost: [TopK]uint16{150, 358, 660, CostUnreached},
		ResidualCost: CostUnreached, ServiceMask: 0b1011, POIDensity: 40, BeaconGrade: 3,
		BeaconStart: 123456, BeaconN: 2, ExclChildren: 7, TerrainFlags: 0b0001_1001, ElevationBand: 6,
	}
	buf := make([]byte, RecordSize)
	r.MarshalTo(buf)
	var back Record
	if err := back.UnmarshalFrom(buf); err != nil {
		t.Fatal(err)
	}
	if back != r {
		t.Fatalf("round trip mismatch:\n%+v\n%+v", r, back)
	}
}

func TestDeriveCugnauxLike(t *testing.T) {
	civs, _ := loadCivs(t)
	hal, _ := civs.ByKey("hallstatt")
	pho, _ := civs.ByKey("phoenicia")
	etr, _ := civs.ByKey("etruria")
	r := EmptyRecord()
	r.Civ = [TopK]uint8{hal, pho, etr, NoCiv}
	r.Cost = [TopK]uint16{150, 358, 660, CostUnreached}
	day := Derive(&r, civs, 0)
	if !day.Reached || len(day.Top) != 3 {
		t.Fatalf("expected three civs, got %+v", day)
	}
	var sum float64
	for _, w := range day.Top {
		sum += w.Weight
	}
	if math.Abs(sum+day.Residual-1) > 1e-9 {
		t.Fatalf("weights must normalise to 1, got %f", sum+day.Residual)
	}
	if day.Top[0].Civ != hal || day.Purity != day.Top[0].Weight {
		t.Fatalf("Hallstatt should dominate: %+v", day)
	}
	night := Derive(&r, civs, 1)
	if night.Purity >= day.Purity {
		t.Fatalf("raising propagation must lower purity: day %.3f night %.3f", day.Purity, night.Purity)
	}
	// The distant civilization gains share at night: that is PLAN §5's whole point.
	if night.Top[2].Weight <= day.Top[2].Weight {
		t.Fatalf("Etruria should gain at night: day %.3f night %.3f", day.Top[2].Weight, night.Top[2].Weight)
	}
}

func TestDeriveUnreached(t *testing.T) {
	civs, _ := loadCivs(t)
	r := EmptyRecord()
	if w := Derive(&r, civs, 0); w.Reached {
		t.Fatalf("empty record must report unreached, got %+v", w)
	}
}

func TestFoldResidual(t *testing.T) {
	civs, _ := loadCivs(t)
	tail := []CivWeight{{Civ: 1, Weight: 0.05}, {Civ: 2, Weight: 0.03}}
	cost := FoldResidual(tail, civs, 150)
	back := math.Exp(-float64(cost) * civs.CostUnitKm / civs.MeanLambdaEff(0, 150))
	if math.Abs(back-0.08) > 0.002 {
		t.Fatalf("residual should reproduce the tail weight 0.08, got %f", back)
	}
	if FoldResidual(nil, civs, 0) != CostUnreached {
		t.Fatal("empty tail must be unreached")
	}
}

func TestDensityByte(t *testing.T) {
	if DensityByte(0) != 0 {
		t.Fatal("zero")
	}
	if DensityByte(1) != 16 {
		t.Fatalf("one POI -> 16, got %d", DensityByte(1))
	}
	if DensityByte(1<<20) != 255 {
		t.Fatal("saturates")
	}
	if d := DensityCount(DensityByte(31)); d < 28 || d > 34 {
		t.Fatalf("inverse of 31 should be ~31, got %f", d)
	}
}

func buildTestSnapshot(t *testing.T, civs *Civs) *Snapshot {
	t.Helper()
	cells, err := h3x.BBoxCells(43.50, 1.30, 43.57, 1.40, h3x.ResWeight)
	if err != nil {
		t.Fatal(err)
	}
	h3x.SortCells(cells)
	s := &Snapshot{BuildID: "test", CostUnitKm: civs.CostUnitKm, Classes: []string{"hearth", "vault", "beacon"}}
	hal, _ := civs.ByKey("hallstatt")
	for _, c := range cells {
		r := EmptyRecord()
		r.Civ[0], r.Cost[0] = hal, 150
		r.ServiceMask = 0b101
		s.Keys = append(s.Keys, uint64(c))
		s.Records = append(s.Records, r)
	}
	// One point beacon adjacent to the Cugnaux centre cell.
	s.Beacons = []Beacon{{ID: 0, Ref: "n/1", Name: "Test museum", Grade: 3, Source: "cell_soil", Lat: 43.5365, Lon: 1.3444, RangeM: 60}}
	centre, _ := h3x.FromLatLng(43.5365, 1.3444, h3x.ResWeight)
	i := s.Find(uint64(centre))
	s.BeaconAdj = []uint32{0}
	s.Records[i].BeaconStart, s.Records[i].BeaconN, s.Records[i].BeaconGrade = 0, 1, 3
	r10, _ := h3x.FromLatLng(43.5365, 1.3444, h3x.ResPlace)
	s.Excl = []uint64{uint64(r10)}
	s.ExclZone = []uint8{0}
	s.ExclZoneNames = []string{"test"}
	s.Zones = []Zone{{ID: 0, Kind: "test", Ref: "n/9", BufferM: 20, Lat: 43.5365, Lon: 1.3444}}
	s.ZoneIdx, s.ZoneIdxZone = []uint64{uint64(r10)}, []uint32{0}
	s.Records[i].ExclChildren = 1
	return s
}

func TestSnapshotRoundTripAndLookup(t *testing.T) {
	civs, civsJSON := loadCivs(t)
	s := buildTestSnapshot(t, civs)
	dir := t.TempDir()
	if err := s.Write(dir, civsJSON); err != nil {
		t.Fatal(err)
	}
	w, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Snap.Keys) != len(s.Keys) || w.Snap.BuildID != "test" || len(w.Snap.Classes) != 3 {
		t.Fatalf("header mismatch: %+v", w.Snap)
	}
	res, err := w.Lookup(43.5365, 1.3444, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Found || len(res.Weights.Top) != 1 || res.Weights.Top[0].Civ != 6 {
		t.Fatalf("lookup: %+v", res)
	}
	if len(res.Services) != 2 || res.Services[0] != "hearth" || res.Services[1] != "beacon" {
		t.Fatalf("services: %v", res.Services)
	}
	if len(res.Beacons) != 1 || res.Beacons[0].Beacon.Name != "Test museum" {
		t.Fatalf("beacon hit: %+v", res.Beacons)
	}
	if !res.ExcludedCell || res.ExcludedCellBy != "test" {
		t.Fatalf("point sits on an excluded r10 cell: %v %q", res.ExcludedCell, res.ExcludedCellBy)
	}
	if !res.Excluded || res.Zone == nil || res.Zone.Kind != "test" {
		t.Fatalf("point is inside the zone buffer: %+v", res.Zone)
	}
	// 200 m away: same r8 cell, not excluded, beacon out of range.
	res2, _ := w.Lookup(43.5365, 1.3470, 0)
	if res2.Excluded || res2.ExcludedCell || len(res2.Beacons) != 0 {
		t.Fatalf("200 m away: %+v", res2)
	}
	out, _ := w.Lookup(48.85, 2.35, 0)
	if out.Found {
		t.Fatal("Paris is not in the snapshot")
	}
	near := w.Nearby(43.5365, 1.3444, 3000)
	if len(near) != 1 || near[0].Kind != "beacon" {
		t.Fatalf("nearby: %+v", near)
	}
}

// BenchmarkLookup is the track 1 acceptance check: a lat/lon must resolve
// to a weight vector in well under 5 ms.
func BenchmarkLookup(b *testing.B) {
	bts, err := os.ReadFile(filepath.Join("..", "..", "rules", "civs.json"))
	if err != nil {
		b.Fatal(err)
	}
	civs, err := ParseCivs(bts)
	if err != nil {
		b.Fatal(err)
	}
	cells, err := h3x.BBoxCells(43.50, 1.30, 43.57, 1.40, h3x.ResWeight)
	if err != nil {
		b.Fatal(err)
	}
	h3x.SortCells(cells)
	s := &Snapshot{CostUnitKm: 1, Classes: []string{"hearth"}}
	for _, c := range cells {
		r := EmptyRecord()
		r.Civ, r.Cost = [TopK]uint8{6, 4, 0, NoCiv}, [TopK]uint16{150, 358, 660, CostUnreached}
		s.Keys = append(s.Keys, uint64(c))
		s.Records = append(s.Records, r)
	}
	w := &World{Snap: s, Civs: civs}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := w.Lookup(43.5365, 1.3444, 0.3); err != nil {
			b.Fatal(err)
		}
	}
}
