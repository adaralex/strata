package spawn

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/adaralex/strata/server/cond"
	"github.com/adaralex/strata/server/world"
	"github.com/adaralex/strata/server/world/h3x"
)

// Collapse errors. The HTTP layer maps them to status codes.
var (
	ErrBadCell    = errors.New("cell is not an r8 index")
	ErrNotFound   = errors.New("no such spawn in that cell and epoch")
	ErrExpired    = errors.New("spawn epoch is over")
	ErrTooFar     = errors.New("fix is outside interaction range")
	ErrExcluded   = errors.New("no interaction here")
	ErrClaimed    = errors.New("already collapsed by this device")
	ErrSpeedGate  = errors.New("moving too fast for interaction")
	ErrNoWorld    = errors.New("cell outside the snapshot")
	ErrEpochAhead = errors.New("spawn epoch is in the future")
)

// CollapseRequest is a claim: this device says it beat this spawn from here.
// The client never says what it looted (non-negotiable 1); the server
// re-derives the spawn and decides the drop.
type CollapseRequest struct {
	Device  string
	SpawnID string
	Cell    string
	Epoch   int64
	Lat     float64
	Lon     float64
	At      time.Time
}

// CollapseResult is the adjudicated outcome.
type CollapseResult struct {
	Spawn  Spawn         `json:"spawn"`
	Item   Item          `json:"item"`
	Beacon *world.Beacon `json:"beacon,omitempty"`
}

// State is the only mutable state phase 0 keeps: one claim per device per
// spawn, and the last fix per device for the speed gate. In memory here;
// Postgres and Redis later.
type State interface {
	// Claim records the claim and reports whether it was the first.
	Claim(device, spawnID string) bool
	// Observe records a fix and reports whether the device is speed-gated.
	Observe(device string, lat, lon float64, at time.Time, maxSpeedKmh, minIntervalS float64) bool
}

// MemoryState is the in-process State.
type MemoryState struct {
	mu     sync.Mutex
	claims map[string]struct{}
	fixes  map[string]fix
}

type fix struct {
	lat, lon float64
	at       time.Time
	gated    bool
}

// NewMemoryState returns an empty in-memory state.
func NewMemoryState() *MemoryState {
	return &MemoryState{claims: map[string]struct{}{}, fixes: map[string]fix{}}
}

// Claim implements State.
func (m *MemoryState) Claim(device, spawnID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := device + "|" + spawnID
	if _, ok := m.claims[k]; ok {
		return false
	}
	m.claims[k] = struct{}{}
	return true
}

// Observe implements State: a pair of fixes at least minIntervalS apart and
// faster than maxSpeedKmh gates the device until a slower pair arrives.
func (m *MemoryState) Observe(device string, lat, lon float64, at time.Time, maxSpeedKmh, minIntervalS float64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	prev, ok := m.fixes[device]
	cur := fix{lat: lat, lon: lon, at: at, gated: prev.gated}
	if ok {
		dt := at.Sub(prev.at).Seconds()
		if dt >= minIntervalS {
			d := h3x.DistanceM(pt(prev.lon, prev.lat), pt(lon, lat))
			cur.gated = d/dt*3.6 > maxSpeedKmh
		} else if dt < 0 {
			cur = prev // out-of-order fix: keep the last one
		}
	}
	m.fixes[device] = cur
	return cur.gated
}

// Collapse adjudicates a claim and derives the drop.
func (s *Spawner) Collapse(req CollapseRequest, st State) (*CollapseResult, error) {
	if req.At.IsZero() {
		req.At = time.Now()
	}
	cell, ok := h3x.ParseCell(req.Cell)
	if !ok || cell.Resolution() != h3x.ResWeight {
		return nil, ErrBadCell
	}
	now := cond.EpochOf(req.At)
	if req.Epoch > now {
		return nil, ErrEpochAhead
	}
	if req.Epoch < now-s.Content.Rules.Placement.ClaimGraceEpochs {
		return nil, ErrExpired
	}
	spawns, _, err := s.SpawnsAtEpoch(cell, req.Epoch)
	if err != nil {
		return nil, err
	}
	var sp *Spawn
	for i := range spawns {
		if spawns[i].ID == req.SpawnID {
			sp = &spawns[i]
			break
		}
	}
	if sp == nil {
		return nil, ErrNotFound
	}
	if st != nil && st.Observe(req.Device, req.Lat, req.Lon, req.At, s.Content.Rules.SpeedGate.MaxSpeedKmh, s.Content.Rules.SpeedGate.MinIntervalS) {
		return nil, ErrSpeedGate
	}
	if d := distM(req.Lat, req.Lon, *sp); d > s.Content.Rules.Placement.InteractionRangeM {
		return nil, fmt.Errorf("%w: %.0f m", ErrTooFar, d)
	}
	res, err := s.World.Lookup(req.Lat, req.Lon, 0)
	if err != nil {
		return nil, err
	}
	if !res.Found {
		return nil, ErrNoWorld
	}
	if res.Excluded {
		return nil, fmt.Errorf("%w: %s %s", ErrExcluded, res.Zone.Kind, res.Zone.Name)
	}
	if st != nil && !st.Claim(req.Device, sp.ID) {
		return nil, ErrClaimed
	}
	var beacon *world.Beacon
	if len(res.Beacons) > 0 {
		beacon = res.Beacons[0].Beacon
	}
	item := s.Drop(*sp, req.Device, beacon, req.At)
	return &CollapseResult{Spawn: *sp, Item: item, Beacon: beacon}, nil
}

// Affix is one stat bonus (PLAN §8: Might, Ward, Resonance, Fortune).
type Affix struct {
	K string  `json:"k"`
	V float64 `json:"v"`
}

// Provenance says where an item came from, permanently (PLAN §6, §7).
type Provenance struct {
	Kind       string    `json:"kind"` // soil | museum | site
	Ref        string    `json:"ref,omitempty"`
	Name       string    `json:"name,omitempty"`
	Cell       string    `json:"cell"`
	AcquiredAt time.Time `json:"acquired_at"`
}

// Item is a drop (PLAN §7). Fact and Fiction are separate fields on purpose;
// a hybrid has no Fact (non-negotiable 5).
type Item struct {
	ID           string     `json:"id"`
	Kind         string     `json:"kind"` // gear | edible
	Civ          string     `json:"civ"`
	Secondary    string     `json:"secondary,omitempty"`
	Archetype    string     `json:"archetype,omitempty"`
	Slot         string     `json:"slot,omitempty"`
	Name         string     `json:"name"`
	Tier         int        `json:"tier"`
	TierName     string     `json:"tier_name"`
	Authenticity string     `json:"authenticity"`
	Affixes      []Affix    `json:"affixes,omitempty"`
	Provenance   Provenance `json:"provenance"`
	Fact         string     `json:"fact,omitempty"`
	Fiction      string     `json:"fiction"`
	Edible       *Edible    `json:"edible,omitempty"`
	CodexRef     string     `json:"codex_ref,omitempty"`
}

// Drop derives the item for a spawn collapsed by a device, optionally inside
// a beacon's range. Deterministic in (secret, spawn, device), so a retried
// request yields the same item.
func (s *Spawner) Drop(sp Spawn, device string, beacon *world.Beacon, at time.Time) Item {
	r := s.Content.Rules
	items := s.Content.Items
	rg := newRNG(keyedHash(s.Secret, sp.id, hashString(device)))
	m := s.Content.Bestiary.ByID(sp.Kind)

	civKey := sp.Civ
	auth := sp.Authenticity
	prov := Provenance{Kind: "soil", Cell: sp.Cell, AcquiredAt: at.UTC()}
	if beacon != nil {
		auth = Accessioned
		prov = Provenance{Kind: "museum", Ref: beacon.Ref, Name: beacon.Name, Cell: sp.Cell, AcquiredAt: at.UTC()}
		if s.isSite(beacon) {
			auth = Sited
			prov.Kind = "site"
		}
		if len(beacon.Civs) > 0 {
			civKey = pickCiv(beacon.Civs, rg)
		}
	}
	tier := sp.Tier
	if m != nil && m.LootBias != nil {
		tier += m.LootBias.TierShift
	}
	tier = max(0, min(len(r.Tiers.Names)-1, tier))
	item := Item{
		ID: fmt.Sprintf("%016x", rg.next()), Civ: civKey, Tier: tier, TierName: r.Tiers.Names[tier],
		Authenticity: auth, Provenance: prov,
	}

	// Edible drop from an edible-tagged monster.
	if m != nil && m.HasTag("edible") && rg.float() < r.Drops.EdibleChanceWhenTagged {
		e := s.edibleFor(civKey, rg)
		item.Kind = "edible"
		item.Name = e.Name
		item.Fact = e.Fact
		item.Fiction = e.Fiction
		item.Edible = e
		return item
	}

	// Gear.
	item.Kind = "gear"
	var arch *Archetype
	if m != nil && m.LootBias != nil && rg.float() < r.Drops.LootBiasChance {
		arch = items.byID[m.LootBias.Archetype]
	}
	if arch == nil {
		arch = &items.Archetypes[rg.intn(len(items.Archetypes))]
	}
	item.Archetype = arch.ID
	item.Slot = arch.Slot
	name, ok := arch.Names[civKey]
	if !ok {
		name = arch.Names["generic"]
	}
	item.Name = name
	item.Fiction = arch.Fiction
	if auth == Drift && sp.Secondary != "" {
		// A hybrid: dominant supplies the archetype, secondary the epithet.
		// No fact panel at all (PLAN §7 "hybrids must never pretend to be real").
		item.Secondary = sp.Secondary
		if ep, ok := items.HybridEpithets[civKey+">"+sp.Secondary]; ok {
			item.Name = ep + " " + lower(name)
		} else {
			item.Name = "unnamed hybrid " + lower(name)
		}
		item.Fiction = "Made by the transmission. It never existed above ground. " + arch.Fiction
	} else {
		item.Fact = arch.Fact[civKey]
	}
	n := r.Drops.AffixCountByTier[tier]
	keys := append([]string(nil), r.Drops.AffixKeys...)
	for i := 0; i < n && len(keys) > 0; i++ {
		k := rg.intn(len(keys))
		v := r.Drops.AffixValueByTier[tier] * (0.8 + 0.4*rg.float())
		item.Affixes = append(item.Affixes, Affix{K: keys[k], V: math.Round(v)})
		keys = append(keys[:k], keys[k+1:]...)
	}
	if auth != Drift && rg.float() < r.Drops.CodexChanceByTier[tier] {
		item.CodexRef = fmt.Sprintf("%s.f%03d", civKey, 1+rg.intn(40))
	}
	return item
}

func (s *Spawner) edibleFor(civ string, rg *rng) *Edible {
	var own, generic []*Edible
	for i := range s.Content.Items.Edibles {
		e := &s.Content.Items.Edibles[i]
		switch e.Civ {
		case civ:
			own = append(own, e)
		case "generic":
			generic = append(generic, e)
		}
	}
	if len(own) > 0 {
		return own[rg.intn(len(own))]
	}
	if len(generic) > 0 {
		return generic[rg.intn(len(generic))]
	}
	return &s.Content.Items.Edibles[0]
}

// isSite reports whether a beacon is an archaeological site, from its POI
// row's subclass.
func (s *Spawner) isSite(b *world.Beacon) bool {
	for i := range s.World.Snap.POIs {
		p := &s.World.Snap.POIs[i]
		if p.Ref == b.Ref && p.Class == "beacon" {
			return p.Subclass == "historic=archaeological_site"
		}
	}
	return false
}

func pickCiv(civs map[string]float64, rg *rng) string {
	keys := make([]string, 0, len(civs))
	for k := range civs {
		keys = append(keys, k)
	}
	sortStrings(keys)
	w := make([]float64, len(keys))
	for i, k := range keys {
		w[i] = civs[k]
	}
	if i := rg.pick(w); i >= 0 {
		return keys[i]
	}
	return keys[0]
}

func hashString(s string) uint64 {
	h := sha256.Sum256([]byte(s))
	return binary.LittleEndian.Uint64(h[:8])
}
