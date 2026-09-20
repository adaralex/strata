package classify

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Lists are the hand-maintained allow and deny lists under /data that the
// exclusion rules reference (PLAN.md §6, §11).
type Lists struct {
	// OptIn and Whitelist map exclusion id -> set of OSM refs allowed anyway.
	OptIn map[string]map[string]bool
	// BlockNames are case-insensitive substrings; a beacon whose name
	// contains one is suppressed pending human review (memorial museums).
	BlockNames []string
}

// LoadLists reads the list files an exclusion rule names, relative to dataDir.
// A missing file is an empty list, which is the conservative default.
func LoadLists(dataDir string, rules *Rules) (*Lists, error) {
	l := &Lists{OptIn: map[string]map[string]bool{}}
	for _, e := range rules.Exclusions {
		for _, name := range []string{e.OptIn, e.Whitelist} {
			if name == "" {
				continue
			}
			refs, err := readRefList(filepath.Join(dataDir, name))
			if err != nil {
				return nil, err
			}
			l.OptIn[e.ID] = refs
		}
	}
	names, err := readStringList(filepath.Join(dataDir, "blocklist_names.json"))
	if err != nil {
		return nil, err
	}
	for _, n := range names {
		l.BlockNames = append(l.BlockNames, strings.ToLower(n))
	}
	return l, nil
}

func readRefList(path string) (map[string]bool, error) {
	refs, err := readStringList(path)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, r := range refs {
		out[r] = true
	}
	return out, nil
}

func readStringList(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var v struct {
		Refs  []string `json:"refs"`
		Names []string `json:"names"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return append(v.Refs, v.Names...), nil
}

// Classified is one feature with its class hits, or its exclusion.
type Classified struct {
	Feature *Feature
	Hits    []Hit
	// Excluded is the exclusion id that suppressed this feature's own
	// classes; empty when live.
	Excluded string
	// IsExclusionZone marks features whose geometry goes into the r10 set.
	IsExclusionZone bool
	Exclusion       *Exclusion
	Terrain         []string
}

// Result is the classified extract.
type Result struct {
	Items []Classified
	Stats ClassifyStats
}

// ClassifyStats counts outcomes per class and exclusion.
type ClassifyStats struct {
	ByClass     map[string]int
	ByExclusion map[string]int
	Excluded    int
	Terrain     map[string]int
}

// Run classifies every feature of an extract.
func Run(ex *Extract, rules *Rules, lists *Lists) *Result {
	res := &Result{Stats: ClassifyStats{ByClass: map[string]int{}, ByExclusion: map[string]int{}, Terrain: map[string]int{}}}
	for i := range ex.Features {
		f := &ex.Features[i]
		c := Classified{Feature: f, Terrain: rules.MatchTerrain(f.Tags)}
		for _, t := range c.Terrain {
			res.Stats.Terrain[t]++
		}
		whitelisted := false
		if e, ok := rules.MatchExclusion(f.Tags); ok {
			if lists != nil && lists.OptIn[e.ID][f.Ref] {
				whitelisted = true
			} else {
				c.Exclusion = e
				c.Excluded = e.ID
				c.IsExclusionZone = !e.PlacementOnly
				res.Stats.ByExclusion[e.ID]++
			}
		}
		hits := rules.MatchClasses(f.Tags)
		kept := hits[:0]
		for _, h := range hits {
			if h.WhitelistOnly && !whitelisted {
				continue
			}
			kept = append(kept, h)
		}
		c.Hits = kept
		if len(c.Hits) > 0 && c.Excluded == "" && lists != nil {
			name := strings.ToLower(f.Name())
			if c.Hits[0].Class.ID == "beacon" && name != "" {
				for _, bad := range lists.BlockNames {
					if strings.Contains(name, bad) {
						c.Excluded = "blocklist:name"
						break
					}
				}
			}
		}
		if len(c.Hits) > 0 {
			if c.Excluded != "" {
				res.Stats.Excluded++
			} else {
				for _, h := range c.Hits {
					res.Stats.ByClass[h.Class.ID]++
				}
			}
		}
		if len(c.Hits) == 0 && c.Excluded == "" && len(c.Terrain) == 0 {
			continue
		}
		res.Items = append(res.Items, c)
	}
	return res
}
