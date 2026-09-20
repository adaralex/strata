// Package classify turns OSM features into the POI classes of PLAN.md §9,
// applying the exclusions of §11 first. The class definitions are data in
// rules/poi_classes.json; this package only evaluates them.
package classify

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Rules is the parsed rules/poi_classes.json.
type Rules struct {
	Version int `json:"version"`
	// Skip matchers drop a feature from every class, never from exclusions
	// or terrain: private land is still private, and a private lake is still water.
	Skip       []Matcher   `json:"skip"`
	Classes    []Class     `json:"classes"`
	Exclusions []Exclusion `json:"exclusions"`
	Terrain    []Terrain   `json:"terrain"`
}

// Class is one POI function.
type Class struct {
	ID          string          `json:"id"`
	Priority    int             `json:"priority"`
	Stackable   bool            `json:"stackable"`
	Match       []Matcher       `json:"match"`
	Geometry    Geometry        `json:"geometry"`
	Interaction json.RawMessage `json:"interaction"`
	Spawn       Spawn           `json:"spawn"`
	CivSource   []string        `json:"civ_source"`
	rangeM      float64
}

// Matcher is one tag pattern. Every key must be present with one of the
// listed values; "*" matches any value.
type Matcher struct {
	Tags map[string][]string `json:"tags"`
	// Not vetoes the match when any listed key carries one of the listed values.
	Not           map[string][]string `json:"not"`
	Grade         uint8               `json:"grade"`
	Potency       float64             `json:"potency"`
	WhitelistOnly bool                `json:"whitelist_only"`
}

// Geometry says how a feature of this class is placed.
type Geometry struct {
	Prefer  string  `json:"prefer"` // point | polygon
	BufferM float64 `json:"buffer_m"`
}

// Spawn holds the spawn-side effects; read by track 3, carried here as data.
type Spawn struct {
	Zone             bool               `json:"zone"`
	DensityMul       float64            `json:"density_mul"`
	TierShift        int                `json:"tier_shift"`
	TierShiftByGrade []int              `json:"tier_shift_by_grade"`
	RarityMulByGrade []float64          `json:"rarity_mul_by_grade"`
	TagBias          map[string]float64 `json:"tag_bias"`
}

// Exclusion is a no-gameplay selector (PLAN.md §11).
type Exclusion struct {
	ID      string              `json:"id"`
	Tags    map[string][]string `json:"tags"`
	BufferM float64             `json:"buffer_m"`
	// OptIn names a data file listing refs that are allowed to host despite
	// matching (places of worship). Whitelist is the same for memorials.
	OptIn     string `json:"opt_in"`
	Whitelist string `json:"whitelist"`
	// PlacementOnly exclusions are not baked into the r10 set; the spawn
	// placer checks them against geometry.
	PlacementOnly bool `json:"placement_only"`
}

// Terrain selectors feed the cell record's terrain flags.
type Terrain struct {
	ID   string              `json:"id"`
	Tags map[string][]string `json:"tags"`
}

// LoadRules reads rules/poi_classes.json.
func LoadRules(path string) (*Rules, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseRules(b)
}

// ParseRules parses the rules file and validates it.
func ParseRules(b []byte) (*Rules, error) {
	var r Rules
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("poi_classes: %w", err)
	}
	if len(r.Classes) == 0 || len(r.Classes) > 16 {
		return nil, fmt.Errorf("poi_classes: need 1..16 classes for the 16-bit service mask, got %d", len(r.Classes))
	}
	seen := map[string]bool{}
	for i := range r.Classes {
		c := &r.Classes[i]
		if seen[c.ID] {
			return nil, fmt.Errorf("poi_classes: duplicate class %q", c.ID)
		}
		seen[c.ID] = true
		if len(c.Match) == 0 {
			return nil, fmt.Errorf("poi_classes: class %q has no matchers", c.ID)
		}
		for j := range c.Match {
			if c.Match[j].Potency == 0 {
				c.Match[j].Potency = 1
			}
			if len(c.Match[j].Tags) == 0 {
				return nil, fmt.Errorf("poi_classes: class %q matcher %d has no tags", c.ID, j)
			}
		}
		if len(c.Interaction) > 0 {
			var ia struct {
				RangeM float64 `json:"range_m"`
			}
			if err := json.Unmarshal(c.Interaction, &ia); err != nil {
				return nil, fmt.Errorf("poi_classes: class %q interaction: %w", c.ID, err)
			}
			c.rangeM = ia.RangeM
		}
	}
	// Class order is the ServiceMask bit order: keep file order, it is stable.
	for _, e := range r.Exclusions {
		if e.ID == "" || len(e.Tags) == 0 {
			return nil, fmt.Errorf("poi_classes: exclusion without id or tags")
		}
	}
	return &r, nil
}

// ClassIDs returns class ids in file order, which is ServiceMask bit order.
func (r *Rules) ClassIDs() []string {
	out := make([]string, len(r.Classes))
	for i, c := range r.Classes {
		out[i] = c.ID
	}
	return out
}

// ClassByID finds a class.
func (r *Rules) ClassByID(id string) *Class {
	for i := range r.Classes {
		if r.Classes[i].ID == id {
			return &r.Classes[i]
		}
	}
	return nil
}

// RangeM is the class's interaction range.
func (c *Class) RangeM() float64 { return c.rangeM }

// Hit is a class match on a feature.
type Hit struct {
	Class    *Class
	Subclass string
	Grade    uint8
	Potency  float64
	// WhitelistOnly hits stand only if the feature is whitelisted.
	WhitelistOnly bool
}

// MatchTags evaluates one matcher against a tag map. It returns the
// "key=value" pair that matched, for the subclass label.
func MatchTags(m map[string][]string, tags map[string]string) (string, bool) {
	// Deterministic order so the subclass label is stable.
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	label := ""
	labelSpecific := false
	for _, k := range keys {
		v, ok := tags[k]
		if !ok {
			return "", false
		}
		matched := false
		for _, want := range m[k] {
			if want == "*" || want == v {
				matched = true
				break
			}
		}
		if !matched {
			return "", false
		}
		specific := m[k][0] != "*"
		switch {
		case specific && !labelSpecific:
			label, labelSpecific = k+"="+v, true
		case !labelSpecific && (label == "" || primaryKeyRank(k) < primaryKeyRank(labelKey(label))):
			label = k + "=" + v
		}
	}
	return label, true
}

// MatchClasses returns the classes a tag set belongs to: the highest-priority
// match plus any stackable ones, highest priority first.
func (r *Rules) MatchClasses(tags map[string]string) []Hit {
	for i := range r.Skip {
		if r.Skip[i].Matches(tags) {
			return nil
		}
	}
	var hits []Hit
	for i := range r.Classes {
		c := &r.Classes[i]
		for j := range c.Match {
			m := &c.Match[j]
			if label, ok := m.match(tags); ok {
				hits = append(hits, Hit{Class: c, Subclass: label, Grade: m.Grade, Potency: m.Potency, WhitelistOnly: m.WhitelistOnly})
				break
			}
		}
	}
	if len(hits) == 0 {
		return nil
	}
	sort.SliceStable(hits, func(a, b int) bool { return hits[a].Class.Priority > hits[b].Class.Priority })
	out := hits[:1]
	for _, h := range hits[1:] {
		if h.Class.Stackable {
			out = append(out, h)
		}
	}
	return out
}

// MatchExclusion returns the first exclusion a tag set matches.
func (r *Rules) MatchExclusion(tags map[string]string) (*Exclusion, bool) {
	for i := range r.Exclusions {
		if _, ok := MatchTags(r.Exclusions[i].Tags, tags); ok {
			return &r.Exclusions[i], true
		}
	}
	return nil, false
}

// MatchTerrain returns the terrain ids a tag set matches.
func (r *Rules) MatchTerrain(tags map[string]string) []string {
	var out []string
	for _, t := range r.Terrain {
		if _, ok := MatchTags(t.Tags, tags); ok {
			out = append(out, t.ID)
		}
	}
	return out
}

// Interesting reports whether a tag set matters to the build at all.
func (r *Rules) Interesting(tags map[string]string) bool {
	if len(tags) == 0 {
		return false
	}
	if _, ok := r.MatchExclusion(tags); ok {
		return true
	}
	if len(r.MatchTerrain(tags)) > 0 {
		return true
	}
	return len(r.MatchClasses(tags)) > 0
}

// primaryKeys orders OSM feature keys for the subclass label when a matcher
// is all wildcards (shop=* with repair=*): the feature key wins.
var primaryKeys = []string{"amenity", "shop", "tourism", "historic", "craft", "leisure", "natural", "landuse", "healthcare", "man_made", "railway", "aeroway", "public_transport"}

func primaryKeyRank(k string) int {
	for i, p := range primaryKeys {
		if p == k {
			return i
		}
	}
	return len(primaryKeys)
}

func labelKey(label string) string {
	if i := strings.IndexByte(label, '='); i >= 0 {
		return label[:i]
	}
	return label
}

// Matches reports whether the matcher accepts a tag set.
func (m *Matcher) Matches(tags map[string]string) bool {
	_, ok := m.match(tags)
	return ok
}

func (m *Matcher) match(tags map[string]string) (string, bool) {
	label, ok := MatchTags(m.Tags, tags)
	if !ok {
		return "", false
	}
	for k, vals := range m.Not {
		v, present := tags[k]
		if !present {
			continue
		}
		for _, bad := range vals {
			if bad == "*" || bad == v {
				return "", false
			}
		}
	}
	return label, true
}
