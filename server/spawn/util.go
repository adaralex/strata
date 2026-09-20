package spawn

import (
	"sort"
	"strings"
)

func sortStrings(s []string) { sort.Strings(s) }

func lower(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}
