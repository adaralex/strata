// Package world holds the H3 r8 cell record, the snapshot codec and the
// lookup that turns a lat/lon into a civilization weight vector. It is shared
// by the offline world build (worldbuild/) and the world service.
package world

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
)

// NumCivs is the fixed roster size (PLAN.md §2). NoCiv marks an empty slot.
const (
	NumCivs = 15
	NoCiv   = 15
	// TopK is how many civilizations a cell record keeps explicitly; the rest
	// are folded into one residual pseudo-cost.
	TopK = 4
)

// Civ is one entry of rules/civs.json.
type Civ struct {
	ID       uint8   `json:"id"`
	Key      string  `json:"key"`
	Name     string  `json:"name"`
	LambdaKm float64 `json:"lambda_km"`
	K        float64 `json:"k"`
	Maritime bool    `json:"maritime"`
	// OceanMul scales the ocean unit cost for this civilization in the drift
	// solve; 1 is neutral, Lapita is below 1 (PLAN §4 "reach constant").
	OceanMul float64 `json:"ocean_mul"`
}

// Civs is the loaded roster, indexed by id.
type Civs struct {
	CostUnitKm float64
	// Smear widens every lambda by smear * c_min so remote ground blends
	// (decision record 0005). 0 disables it.
	Smear float64
	List  [NumCivs]Civ
	byKey map[string]uint8
}

type civsFile struct {
	CostUnitKm float64 `json:"cost_unit_km"`
	Smear      float64 `json:"smear"`
	Civs       []Civ   `json:"civs"`
}

// LoadCivs reads rules/civs.json.
func LoadCivs(path string) (*Civs, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseCivs(b)
}

// ParseCivs parses the contents of rules/civs.json.
func ParseCivs(b []byte) (*Civs, error) {
	var f civsFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("civs: %w", err)
	}
	if len(f.Civs) != NumCivs {
		return nil, fmt.Errorf("civs: want exactly %d civilizations, got %d", NumCivs, len(f.Civs))
	}
	c := &Civs{CostUnitKm: f.CostUnitKm, Smear: math.Max(0, f.Smear), byKey: map[string]uint8{}}
	if c.CostUnitKm <= 0 {
		c.CostUnitKm = 1
	}
	seen := [NumCivs]bool{}
	for _, civ := range f.Civs {
		if civ.ID >= NumCivs {
			return nil, fmt.Errorf("civs: id %d out of range", civ.ID)
		}
		if seen[civ.ID] {
			return nil, fmt.Errorf("civs: duplicate id %d", civ.ID)
		}
		if civ.LambdaKm <= 0 {
			return nil, fmt.Errorf("civs: %s needs lambda_km > 0", civ.Key)
		}
		if civ.OceanMul <= 0 {
			civ.OceanMul = 1
		}
		seen[civ.ID] = true
		c.List[civ.ID] = civ
		c.byKey[civ.Key] = civ.ID
	}
	return c, nil
}

// ByKey resolves a civilization key such as "hallstatt".
func (c *Civs) ByKey(key string) (uint8, bool) {
	id, ok := c.byKey[key]
	return id, ok
}

// Lambda is the decay constant for civ id under a propagation scalar in
// [0,1] (PLAN.md §5): lambda_i = lambda_base * (1 + k_i * propagation).
func (c *Civs) Lambda(id uint8, propagation float64) float64 {
	civ := c.List[id]
	return civ.LambdaKm * (1 + civ.K*propagation)
}

// LambdaEff is Lambda widened by the smear term: lambda + smear * cMinKm,
// where cMinKm is the cheapest cost at the cell being derived.
func (c *Civs) LambdaEff(id uint8, propagation, cMinKm float64) float64 {
	return c.Lambda(id, propagation) + c.Smear*cMinKm
}

// MeanLambda is the base lambda used for the residual pseudo-cost.
func (c *Civs) MeanLambda(propagation float64) float64 {
	var s float64
	for i := range c.List {
		s += c.Lambda(uint8(i), propagation)
	}
	return s / NumCivs
}

// MeanLambdaEff is MeanLambda with the smear term.
func (c *Civs) MeanLambdaEff(propagation, cMinKm float64) float64 {
	return c.MeanLambda(propagation) + c.Smear*cMinKm
}
