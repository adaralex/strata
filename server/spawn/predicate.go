package spawn

import (
	"encoding/json"
	"fmt"
)

// Facts is what a predicate reads: the condition vector plus cell statics.
// Numeric fields come back as float64; booleans as 0/1; strings as strings.
type Facts interface {
	Fact(name string) (num float64, str string, ok bool)
}

// Predicate is a compiled rule condition (PLAN.md §5 rule language):
//
//	{}                                  always true
//	{ "all": [p, ...] }                 every p
//	{ "any": [p, ...] }                 at least one p
//	{ "field": value }                  equality (number, bool or string)
//	{ "field": { "lt"|"lte"|"gt"|"gte"|"eq": n } }
//	{ "field": { "in": [v, ...] } }
//
// Several fields in one object are ANDed.
type Predicate func(Facts) bool

// CompilePredicate parses a rule condition. An empty or null message is
// always true.
func CompilePredicate(raw json.RawMessage) (Predicate, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return func(Facts) bool { return true }, nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	var parts []Predicate
	for key, val := range obj {
		switch key {
		case "all", "any":
			var list []json.RawMessage
			if err := json.Unmarshal(val, &list); err != nil {
				return nil, fmt.Errorf("%s: %w", key, err)
			}
			subs := make([]Predicate, 0, len(list))
			for _, item := range list {
				p, err := CompilePredicate(item)
				if err != nil {
					return nil, err
				}
				subs = append(subs, p)
			}
			isAll := key == "all"
			parts = append(parts, func(f Facts) bool {
				for _, s := range subs {
					if s(f) != isAll {
						return !isAll
					}
				}
				return isAll
			})
		default:
			p, err := compileField(key, val)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", key, err)
			}
			parts = append(parts, p)
		}
	}
	return func(f Facts) bool {
		for _, p := range parts {
			if !p(f) {
				return false
			}
		}
		return true
	}, nil
}

func compileField(field string, val json.RawMessage) (Predicate, error) {
	var scalar any
	if err := json.Unmarshal(val, &scalar); err != nil {
		return nil, err
	}
	switch v := scalar.(type) {
	case bool:
		want := 0.0
		if v {
			want = 1
		}
		return func(f Facts) bool { n, _, ok := f.Fact(field); return ok && n == want }, nil
	case float64:
		return func(f Facts) bool { n, _, ok := f.Fact(field); return ok && n == v }, nil
	case string:
		return func(f Facts) bool { _, s, ok := f.Fact(field); return ok && s == v }, nil
	case map[string]any:
		var ops []Predicate
		for op, arg := range v {
			switch op {
			case "in":
				list, ok := arg.([]any)
				if !ok {
					return nil, fmt.Errorf("in wants a list")
				}
				ops = append(ops, func(f Facts) bool {
					n, s, ok := f.Fact(field)
					if !ok {
						return false
					}
					for _, item := range list {
						switch it := item.(type) {
						case string:
							if it == s {
								return true
							}
						case float64:
							if it == n {
								return true
							}
						}
					}
					return false
				})
			case "lt", "lte", "gt", "gte", "eq":
				x, ok := arg.(float64)
				if !ok {
					return nil, fmt.Errorf("%s wants a number", op)
				}
				op := op
				ops = append(ops, func(f Facts) bool {
					n, _, ok := f.Fact(field)
					if !ok {
						return false
					}
					switch op {
					case "lt":
						return n < x
					case "lte":
						return n <= x
					case "gt":
						return n > x
					case "gte":
						return n >= x
					default:
						return n == x
					}
				})
			default:
				return nil, fmt.Errorf("unknown operator %q", op)
			}
		}
		return func(f Facts) bool {
			for _, p := range ops {
				if !p(f) {
					return false
				}
			}
			return true
		}, nil
	default:
		return nil, fmt.Errorf("unsupported value %v", scalar)
	}
}
