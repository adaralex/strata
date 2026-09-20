package spawn

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/adaralex/strata/server/cond"
	"github.com/adaralex/strata/server/world"
	"github.com/adaralex/strata/server/world/h3x"
)

// Authenticity values (PLAN §7). Accessioned and Sited are set at collapse.
const (
	Grounded    = "grounded"
	Drift       = "drift"
	Confluence  = "confluence"
	Accessioned = "accessioned"
	Sited       = "sited"
)

// Spawn is one monster in one cell for one epoch. It is never stored; any
// server re-derives it from (cell, epoch, digest) and the secret.
type Spawn struct {
	ID        string `json:"id"` // hex, stable for the epoch
	Cell      string `json:"cell"`
	Epoch     int64  `json:"epoch"`
	ExpiresAt int64  `json:"expires_at"` // unix seconds, end of the grace window
	Slot      int    `json:"slot"`

	Civ          string   `json:"civ"`
	Secondary    string   `json:"secondary,omitempty"` // Drift hybrids
	Kind         string   `json:"kind"`
	Name         string   `json:"name"`
	Tags         []string `json:"tags,omitempty"`
	Fiction      string   `json:"fiction"`
	Rank         string   `json:"rank"`
	Tier         int      `json:"tier"`
	TierName     string   `json:"tier_name"`
	Authenticity string   `json:"authenticity"`
	ValueMul     float64  `json:"value_mul"`

	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	PlaceCell string  `json:"place_cell"` // r10

	id uint64
}

// Spawner derives spawns for a loaded world.
type Spawner struct {
	World   *world.World
	Content *Content
	Secret  []byte
}

// New wires a spawner. The secret must be the same on every server that
// answers for the same world, or players see different monsters.
func New(w *world.World, c *Content, secret []byte) (*Spawner, error) {
	if len(secret) < 16 {
		return nil, fmt.Errorf("spawn: secret must be at least 16 bytes")
	}
	return &Spawner{World: w, Content: c, Secret: secret}, nil
}

// Conditions computes the condition vector for a cell at time t.
func (s *Spawner) Conditions(cell h3x.Cell, t time.Time) (cond.Vector, error) {
	lat, lon, err := h3x.Center(cell)
	if err != nil {
		return cond.Vector{}, err
	}
	return cond.At(lat, lon, t, s.Content.Rules.Propagation), nil
}

// CellResult is what the spawner knows about a cell before rolling.
type CellResult struct {
	Cell    h3x.Cell
	Record  world.Record
	Weights world.Weights
	Cond    cond.Vector
}

// Spawns returns the monsters of an r8 cell at time t.
func (s *Spawner) Spawns(cell h3x.Cell, t time.Time) ([]Spawn, cond.Vector, error) {
	v, err := s.Conditions(cell, t)
	if err != nil {
		return nil, v, err
	}
	out, err := s.spawnsAt(cell, v)
	return out, v, err
}

// SpawnsAtEpoch re-derives an epoch's spawns, for claim validation.
func (s *Spawner) SpawnsAtEpoch(cell h3x.Cell, epoch int64) ([]Spawn, cond.Vector, error) {
	return s.Spawns(cell, cond.EpochStart(epoch))
}

func (s *Spawner) spawnsAt(cell h3x.Cell, v cond.Vector) ([]Spawn, error) {
	i := s.World.Snap.Find(uint64(cell))
	if i < 0 {
		return nil, nil
	}
	rec := s.World.Snap.Records[i]
	if rec.ExclChildren >= h3x.ChildrenPerLevel*h3x.ChildrenPerLevel {
		return nil, nil
	}
	weights := world.Derive(&rec, s.World.Civs, v.Propagation)
	if !weights.Reached {
		return nil, nil
	}
	cr := &CellResult{Cell: cell, Record: rec, Weights: weights, Cond: v}
	facts := &cellFacts{cr: cr}
	r := s.Content.Rules

	// Count and value (PLAN §10, §11).
	poiCount := world.DensityCount(rec.POIDensity)
	n := r.Count.Base + math.Floor(math.Log2(1+poiCount)*r.Count.PerLog2Density)
	if rec.IsWild() {
		n *= r.Count.WildMul
	}
	if v.AfterMidnight && int(rec.UrbanBand()) <= r.Count.AfterMidnightUrbanBandMax {
		n *= r.Count.AfterMidnightMul
	}
	count := int(math.Min(float64(r.Count.Max), math.Floor(n+0.5)))
	valueMul := math.Sqrt(r.Value.ReferenceDensity / (1 + poiCount))
	valueMul = math.Max(1, math.Min(r.Value.MaxMul, valueMul))

	// Placement candidates: the non-excluded r10 children, in H3 order.
	children, err := h3x.Children(cell, h3x.ResPlace)
	if err != nil {
		return nil, err
	}
	free := children[:0:0]
	var street []h3x.Cell // free children that carry an on-street point
	for _, c := range children {
		if s.World.Snap.IsExcluded(uint64(c)) {
			continue
		}
		free = append(free, c)
		if _, _, ok := s.World.Snap.WalkPoint(uint64(c)); ok {
			street = append(street, c)
		}
	}
	if len(free) == 0 {
		return nil, nil
	}

	// Tier distribution for this cell.
	shift := float64(r.Tiers.BeaconShiftByGrad[min(int(rec.BeaconGrade), len(r.Tiers.BeaconShiftByGrad)-1)])
	if rec.IsWild() {
		shift += float64(r.Tiers.WildShift)
	}
	shift += (valueMul - 1) * r.Tiers.ValueShiftPerMul
	tierDist := shiftTiers(r.Tiers.BaseDistribution, shift)

	// Civ pick weights: the top-K, renormalised without the residual.
	civWeights := make([]float64, len(weights.Top))
	for k, cw := range weights.Top {
		civWeights[k] = cw.Weight
	}
	confluence, driftP := s.authenticity(weights)

	seed := keyedHash(s.Secret, uint64(cell), uint64(v.Epoch), v.Digest())
	expires := cond.EpochStart(v.Epoch + 1 + r.Placement.ClaimGraceEpochs).Unix()

	// Ranks: slot 0 is the area boss when this cell wins the neighbourhood
	// roll, the next slot an elite with its own chance, the rest commons.
	ranks := make([]string, count)
	for i := range ranks {
		ranks[i] = RankCommon
	}
	next := 0
	if next < count && s.bossHere(cell, v) {
		ranks[next] = RankBoss
		next++
	}
	if next < count && r.Ranks.Elite.MaxPerCell > 0 && newRNG(mix(seed, 0xE117)).float() < r.Ranks.Elite.Chance {
		ranks[next] = RankElite
	}
	bossDist := shiftTiers(tierDist, r.Ranks.Boss.TierShift)

	var out []Spawn
	elites := 0
	for slot := 0; slot < count; slot++ {
		rg := newRNG(mix(seed, uint64(slot)))
		ci := rg.pick(civWeights)
		if ci < 0 {
			continue
		}
		civID := weights.Top[ci].Civ
		m, rank := s.pickMonster(civID, ranks[slot], facts, rg)
		if rank == RankElite {
			// A boss slot that fell back to an elite still respects the cap.
			if elites >= r.Ranks.Elite.MaxPerCell {
				m, rank = s.pickMonster(civID, RankCommon, facts, rg)
			} else {
				elites++
			}
		}
		if m == nil {
			continue
		}
		auth := Grounded
		if confluence {
			auth = Confluence
		} else if rg.float() < driftP {
			auth = Drift
		}
		dist, floor, rankMul := tierDist, 0, 1.0
		switch rank {
		case RankBoss:
			dist, floor, rankMul = bossDist, r.Ranks.Boss.TierFloor, r.Ranks.Boss.ValueMul
		case RankElite:
			floor, rankMul = r.Ranks.Elite.TierFloor, r.Ranks.Elite.ValueMul
		}
		tier := max(rg.pick(dist), min(floor, len(r.Tiers.Names)-1))
		// Prefer a child with a pedestrian way and stand on it; fall back to
		// anywhere non-excluded in a cell with no ways at all.
		var place h3x.Cell
		var plat, plon, jitter float64
		if len(street) > 0 {
			place = street[rg.intn(len(street))]
			plat, plon, _ = s.World.Snap.WalkPoint(uint64(place))
			jitter = r.Placement.StreetJitterM
		} else {
			place = free[rg.intn(len(free))]
			var err error
			if plat, plon, err = h3x.Center(place); err != nil {
				continue
			}
			jitter = r.Placement.JitterM
		}
		plat, plon = h3x.Offset(plat, plon, rg.float()*360, math.Sqrt(rg.float())*jitter)
		sp := Spawn{
			Cell: cell.String(), Epoch: v.Epoch, ExpiresAt: expires, Slot: slot,
			Civ: s.World.CivName(civID), Kind: m.ID, Name: m.Name, Tags: m.Tags, Fiction: m.Fiction,
			Rank: rank, Tier: tier, TierName: r.Tiers.Names[tier], Authenticity: auth, ValueMul: math.Round(valueMul*rankMul*100) / 100,
			Lat: plat, Lon: plon, PlaceCell: place.String(),
			id: mix(seed, 0x5150+uint64(slot)),
		}
		if auth == Drift {
			if sec := strongestOther(weights, civID); sec != world.NoCiv {
				sp.Secondary = s.World.CivName(sec)
			} else {
				sp.Authenticity = Grounded // nothing to hybridise with
			}
		}
		sp.ID = fmt.Sprintf("%016x", sp.id)
		out = append(out, sp)
	}
	return out, nil
}

// pickMonster chooses among the civilization's eligible monsters of a rank,
// falling back to the next rank down when none is eligible (a civilization
// with no boss for these conditions still fills the slot).
func (s *Spawner) pickMonster(civ uint8, rank string, facts *cellFacts, rg *rng) (*Monster, string) {
	for _, rk := range ranksFrom(rank) {
		if m := s.pickMonsterOfRank(civ, rk, facts, rg); m != nil {
			return m, rk
		}
	}
	return nil, ""
}

func ranksFrom(rank string) []string {
	switch rank {
	case RankBoss:
		return []string{RankBoss, RankElite, RankCommon}
	case RankElite:
		return []string{RankElite, RankCommon}
	default:
		return []string{RankCommon}
	}
}

// bossHere is the area-boss rule: the cell's boss roll must beat every
// neighbour within the exclusive ring and fall under the chance, so two
// adjacent cells never both hold a boss in one epoch. Neighbour rolls need
// only the neighbour's index, not its record.
func (s *Spawner) bossHere(cell h3x.Cell, v cond.Vector) bool {
	b := s.Content.Rules.Ranks.Boss
	if b.Chance <= 0 {
		return false
	}
	roll := func(c h3x.Cell) float64 {
		return float64(keyedHash(s.Secret, uint64(c), uint64(v.Epoch), v.Digest(), 0xB055)>>11) / (1 << 53)
	}
	mine := roll(cell)
	if mine >= b.Chance {
		return false
	}
	disk, err := h3x.Disk(cell, max(1, b.ExclusiveRing))
	if err != nil {
		return false
	}
	for _, n := range disk {
		if n != cell && roll(n) <= mine {
			return false
		}
	}
	return true
}

func (s *Spawner) pickMonsterOfRank(civ uint8, rank string, facts *cellFacts, rg *rng) *Monster {
	cands := s.Content.Bestiary.byCiv[civ][rank]
	var eligible []*Monster
	var weights []float64
	hearth := facts.cr.Record.ServiceMask&s.World.Snap.ClassBit("hearth") != 0
	for _, m := range cands {
		if !m.when(facts) {
			continue
		}
		w := m.Weight
		for j, p := range m.mods {
			if p(facts) {
				w *= m.WeightMods[j].Mul
			}
		}
		if hearth && m.HasTag("edible") {
			w *= s.hearthEdibleBias()
		}
		eligible = append(eligible, m)
		weights = append(weights, w)
	}
	i := rg.pick(weights)
	if i < 0 {
		return nil
	}
	return eligible[i]
}

// hearthEdibleBias is the hearth class's spawn tag bias from poi_classes.json,
// carried as a constant here until the rules loader is shared (phase 0).
func (s *Spawner) hearthEdibleBias() float64 { return 1.5 }

// authenticity reads purity into the cell's authenticity odds (PLAN §7):
// whether every spawn is a Confluence, and otherwise the per-spawn chance of
// a Drift hybrid, 0 at grounded_min_purity, 1 at drift_max_purity.
func (s *Spawner) authenticity(w world.Weights) (confluence bool, driftP float64) {
	a := s.Content.Rules.Authenticity
	if len(w.Top) >= 3 && w.Top[0].Weight-w.Top[2].Weight <= a.ConfluenceSpread {
		return true, 0
	}
	span := a.GroundedMinPurity - a.DriftMaxPurity
	if span <= 0 {
		return false, 0
	}
	driftP = (a.GroundedMinPurity - w.Purity) / span
	return false, math.Max(0, math.Min(1, driftP))
}

// strongestOther is the hybrid partner: the heaviest civilization that is
// not the spawn's own.
func strongestOther(w world.Weights, civ uint8) uint8 {
	for _, cw := range w.Top {
		if cw.Civ != civ {
			return cw.Civ
		}
	}
	return world.NoCiv
}

// shiftTiers moves probability mass up (shift > 0) or down the tier ladder.
// Each whole step moves 60% of every tier's mass one tier; a fractional
// step interpolates.
func shiftTiers(base []float64, shift float64) []float64 {
	out := append([]float64(nil), base...)
	steps := int(math.Abs(shift))
	frac := math.Abs(shift) - float64(steps)
	dir := 1
	if shift < 0 {
		dir = -1
	}
	apply := func(d []float64, amount float64) []float64 {
		n := len(d)
		next := make([]float64, n)
		for i, p := range d {
			j := i + dir
			if j < 0 || j >= n {
				next[i] += p
				continue
			}
			next[i] += p * (1 - amount)
			next[j] += p * amount
		}
		return next
	}
	for i := 0; i < steps; i++ {
		out = apply(out, 0.6)
	}
	if frac > 0 {
		out = apply(out, 0.6*frac)
	}
	return out
}

// cellFacts exposes the condition vector and cell statics to predicates.
type cellFacts struct{ cr *CellResult }

func (f *cellFacts) Fact(name string) (float64, string, bool) {
	v, rec := f.cr.Cond, &f.cr.Record
	b := func(x bool) float64 {
		if x {
			return 1
		}
		return 0
	}
	switch name {
	case "is_night":
		return b(v.IsNight), "", true
	case "phase":
		return float64(v.Phase), v.PhaseName, true
	case "moon_illumination":
		return v.MoonIllum, "", true
	case "propagation":
		return v.Propagation, "", true
	case "after_midnight":
		return b(v.AfterMidnight), "", true
	case "solar_alt_deg":
		return v.SolarAltDeg, "", true
	case "weather":
		return 0, v.Weather, true
	case "precip":
		return b(v.Precip), "", true
	case "storm":
		return b(v.Storm), "", true
	case "fog":
		return b(v.Fog), "", true
	case "frost":
		return b(v.Frost), "", true
	case "water_band":
		return float64(rec.WaterBand()), "", true
	case "wild":
		return b(rec.IsWild()), "", true
	case "urban_band":
		return float64(rec.UrbanBand()), "", true
	case "beacon_grade":
		return float64(rec.BeaconGrade), "", true
	case "poi_density":
		return world.DensityCount(rec.POIDensity), "", true
	case "purity":
		return f.cr.Weights.Purity, "", true
	}
	return 0, "", false
}

// Near returns the spawns of the cell under a point and its k-ring 1,
// nearest first, with their distance from the point.
func (s *Spawner) Near(lat, lon float64, t time.Time) ([]Spawn, cond.Vector, error) {
	centre, err := h3x.FromLatLng(lat, lon, h3x.ResWeight)
	if err != nil {
		return nil, cond.Vector{}, err
	}
	disk, err := h3x.Disk(centre, 1)
	if err != nil {
		return nil, cond.Vector{}, err
	}
	v, err := s.Conditions(centre, t)
	if err != nil {
		return nil, v, err
	}
	var out []Spawn
	for _, c := range disk {
		cv := v
		if c != centre {
			// Same sky within a kilometre: reuse the centre's vector so the
			// whole neighbourhood shares one digest.
			cv.Epoch = v.Epoch
		}
		sp, err := s.spawnsAt(c, cv)
		if err != nil {
			return nil, v, err
		}
		out = append(out, sp...)
	}
	sort.Slice(out, func(i, j int) bool {
		return distM(lat, lon, out[i]) < distM(lat, lon, out[j])
	})
	return out, v, nil
}

func distM(lat, lon float64, sp Spawn) float64 {
	return h3x.DistanceM(pt(lon, lat), pt(sp.Lon, sp.Lat))
}

// CellOf returns the r8 cell under a point.
func (s *Spawner) CellOf(lat, lon float64) (h3x.Cell, error) {
	return h3x.FromLatLng(lat, lon, h3x.ResWeight)
}

// DistanceM is the distance from a point to a spawn.
func (s *Spawner) DistanceM(lat, lon float64, sp Spawn) float64 { return distM(lat, lon, sp) }
