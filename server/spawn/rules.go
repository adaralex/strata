// Package spawn derives the monsters of a cell for an epoch from a hash, with
// no stored spawn rows (PLAN.md §4 "Deterministic spawns instead of a spawn
// table", §5 rules as data), and decides the drop when one collapses (§7,
// non-negotiable 1).
package spawn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adaralex/strata/server/cond"
	"github.com/adaralex/strata/server/world"
)

// Rules is rules/spawn.json.
type Rules struct {
	Version      int              `json:"version"`
	EpochSeconds int              `json:"epoch_seconds"`
	Propagation  cond.Propagation `json:"propagation"`
	Count        struct {
		Base                      float64 `json:"base"`
		PerLog2Density            float64 `json:"per_log2_density"`
		Max                       int     `json:"max"`
		WildMul                   float64 `json:"wild_mul"`
		AfterMidnightMul          float64 `json:"after_midnight_mul"`
		AfterMidnightUrbanBandMax int     `json:"after_midnight_urban_band_max"`
	} `json:"count"`
	Value struct {
		ReferenceDensity float64 `json:"reference_density"`
		MaxMul           float64 `json:"max_mul"`
	} `json:"value"`
	Tiers struct {
		Names             []string  `json:"names"`
		BaseDistribution  []float64 `json:"base_distribution"`
		BeaconShiftByGrad []int     `json:"beacon_shift_by_grade"`
		WildShift         int       `json:"wild_shift"`
		ValueShiftPerMul  float64   `json:"value_shift_per_mul"`
	} `json:"tiers"`
	Ranks struct {
		Elite struct {
			Chance     float64 `json:"chance"`
			MaxPerCell int     `json:"max_per_cell"`
			TierFloor  int     `json:"tier_floor"`
			ValueMul   float64 `json:"value_mul"`
		} `json:"elite"`
		Boss struct {
			Chance        float64 `json:"chance"`
			ExclusiveRing int     `json:"exclusive_ring"`
			TierFloor     int     `json:"tier_floor"`
			ValueMul      float64 `json:"value_mul"`
			TierShift     float64 `json:"tier_shift"`
		} `json:"boss"`
	} `json:"ranks"`
	Authenticity struct {
		GroundedMinPurity float64 `json:"grounded_min_purity"`
		DriftMaxPurity    float64 `json:"drift_max_purity"`
		ConfluenceSpread  float64 `json:"confluence_spread"`
	} `json:"authenticity"`
	Placement struct {
		JitterM           float64 `json:"jitter_m"`
		StreetJitterM     float64 `json:"street_jitter_m"`
		InteractionRangeM float64 `json:"interaction_range_m"`
		SenseRangeM       float64 `json:"sense_range_m"`
		ClaimGraceEpochs  int64   `json:"claim_grace_epochs"`
	} `json:"placement"`
	SpeedGate struct {
		MaxSpeedKmh  float64 `json:"max_speed_kmh"`
		MinIntervalS float64 `json:"min_interval_s"`
	} `json:"speed_gate"`
	Drops struct {
		EdibleChanceWhenTagged float64   `json:"edible_chance_when_tagged"`
		LootBiasChance         float64   `json:"loot_bias_chance"`
		CodexChanceByTier      []float64 `json:"codex_chance_by_tier"`
		AffixCountByTier       []int     `json:"affix_count_by_tier"`
		AffixKeys              []string  `json:"affix_keys"`
		AffixValueByTier       []float64 `json:"affix_value_by_tier"`
	} `json:"drops"`
}

// Monster is one rules/bestiary.json entry.
type Monster struct {
	ID         string          `json:"id"`
	Civ        string          `json:"civ"`
	Rank       string          `json:"rank"` // common | elite | boss; empty = common
	Name       string          `json:"name"`
	Weight     float64         `json:"weight"`
	Tags       []string        `json:"tags"`
	When       json.RawMessage `json:"when"`
	WeightMods []struct {
		If  json.RawMessage `json:"if"`
		Mul float64         `json:"mul"`
	} `json:"weight_mods"`
	LootBias *struct {
		Archetype string `json:"archetype"`
		TierShift int    `json:"tier_shift"`
	} `json:"loot_bias"`
	Note    string `json:"note"`
	Fiction string `json:"fiction"`

	when Predicate
	mods []Predicate
	civ  uint8
}

// HasTag reports whether the monster carries a tag.
func (m *Monster) HasTag(tag string) bool {
	for _, t := range m.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// Bestiary is rules/bestiary.json, indexed by civilization id.
type Bestiary struct {
	Monsters []Monster
	byCiv    [world.NumCivs]map[string][]*Monster // rank -> monsters
	byID     map[string]*Monster
}

// Ranks, from the bulk to the rarest.
const (
	RankCommon = "common"
	RankElite  = "elite"
	RankBoss   = "boss"
)

// ByID finds a monster.
func (b *Bestiary) ByID(id string) *Monster { return b.byID[id] }

// Archetype is one rules/items.json archetype.
type Archetype struct {
	ID      string            `json:"id"`
	Slot    string            `json:"slot"`
	Names   map[string]string `json:"names"`
	Fact    map[string]string `json:"fact"`
	Fiction string            `json:"fiction"`
}

// Edible is a consumable drop.
type Edible struct {
	ID   string `json:"id"`
	Civ  string `json:"civ"`
	Name string `json:"name"`
	Buff struct {
		Stat    string  `json:"stat"`
		Pct     float64 `json:"pct"`
		Minutes int     `json:"minutes"`
	} `json:"buff"`
	Fact    string `json:"fact,omitempty"`
	Fiction string `json:"fiction"`
}

// Items is rules/items.json.
type Items struct {
	Version        int               `json:"version"`
	Archetypes     []Archetype       `json:"archetypes"`
	HybridEpithets map[string]string `json:"hybrid_epithets"`
	Edibles        []Edible          `json:"edibles"`
	byID           map[string]*Archetype
}

// Content is everything the spawner reads from /rules.
type Content struct {
	Rules    *Rules
	Bestiary *Bestiary
	Items    *Items
}

// LoadContent reads spawn.json, bestiary.json and items.json from rulesDir
// and validates them against the roster.
func LoadContent(rulesDir string, civs *world.Civs) (*Content, error) {
	var c Content
	if err := readJSON(filepath.Join(rulesDir, "spawn.json"), &c.Rules); err != nil {
		return nil, err
	}
	if err := c.Rules.validate(); err != nil {
		return nil, err
	}
	var bf struct {
		Monsters []Monster `json:"monsters"`
	}
	if err := readJSON(filepath.Join(rulesDir, "bestiary.json"), &bf); err != nil {
		return nil, err
	}
	var items Items
	if err := readJSON(filepath.Join(rulesDir, "items.json"), &items); err != nil {
		return nil, err
	}
	items.byID = map[string]*Archetype{}
	for i := range items.Archetypes {
		a := &items.Archetypes[i]
		if a.ID == "" || a.Names["generic"] == "" {
			return nil, fmt.Errorf("items: archetype %d needs an id and a generic name", i)
		}
		items.byID[a.ID] = a
	}
	if len(items.Archetypes) == 0 || len(items.Edibles) == 0 {
		return nil, fmt.Errorf("items: need at least one archetype and one edible")
	}
	c.Items = &items

	b := &Bestiary{Monsters: bf.Monsters, byID: map[string]*Monster{}}
	for i := range b.Monsters {
		m := &b.Monsters[i]
		id, ok := civs.ByKey(m.Civ)
		if !ok {
			return nil, fmt.Errorf("bestiary: %s: unknown civ %q", m.ID, m.Civ)
		}
		if m.Weight <= 0 {
			return nil, fmt.Errorf("bestiary: %s: weight must be > 0", m.ID)
		}
		if m.Rank == "" {
			m.Rank = RankCommon
		}
		if m.Rank != RankCommon && m.Rank != RankElite && m.Rank != RankBoss {
			return nil, fmt.Errorf("bestiary: %s: rank must be common, elite or boss, got %q", m.ID, m.Rank)
		}
		if m.LootBias != nil && items.byID[m.LootBias.Archetype] == nil {
			return nil, fmt.Errorf("bestiary: %s: loot_bias archetype %q not in items.json", m.ID, m.LootBias.Archetype)
		}
		p, err := CompilePredicate(m.When)
		if err != nil {
			return nil, fmt.Errorf("bestiary: %s: when: %w", m.ID, err)
		}
		m.when = p
		for j, wm := range m.WeightMods {
			mp, err := CompilePredicate(wm.If)
			if err != nil {
				return nil, fmt.Errorf("bestiary: %s: weight_mods[%d]: %w", m.ID, j, err)
			}
			m.mods = append(m.mods, mp)
		}
		m.civ = id
		if b.byID[m.ID] != nil {
			return nil, fmt.Errorf("bestiary: duplicate id %q", m.ID)
		}
		b.byID[m.ID] = m
		if b.byCiv[id] == nil {
			b.byCiv[id] = map[string][]*Monster{}
		}
		b.byCiv[id][m.Rank] = append(b.byCiv[id][m.Rank], m)
	}
	for id := range b.byCiv {
		if len(b.byCiv[id][RankCommon]) == 0 {
			return nil, fmt.Errorf("bestiary: civilization %s has no common monsters", civs.List[id].Key)
		}
	}
	c.Bestiary = b
	return &c, nil
}

func (r *Rules) validate() error {
	n := len(r.Tiers.Names)
	if n == 0 || len(r.Tiers.BaseDistribution) != n || len(r.Drops.CodexChanceByTier) != n ||
		len(r.Drops.AffixCountByTier) != n || len(r.Drops.AffixValueByTier) != n {
		return fmt.Errorf("spawn.json: tier tables must all have %d entries", n)
	}
	if len(r.Tiers.BeaconShiftByGrad) < 5 {
		return fmt.Errorf("spawn.json: beacon_shift_by_grade needs grades 0..4")
	}
	if r.Count.Max <= 0 || r.Placement.JitterM <= 0 || r.Placement.InteractionRangeM <= 0 {
		return fmt.Errorf("spawn.json: count.max, placement.jitter_m and interaction_range_m must be > 0")
	}
	if r.EpochSeconds != cond.EpochSeconds {
		return fmt.Errorf("spawn.json: epoch_seconds %d must match cond.EpochSeconds %d", r.EpochSeconds, cond.EpochSeconds)
	}
	return nil
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}
