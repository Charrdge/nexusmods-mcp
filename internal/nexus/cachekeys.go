package nexus

import (
	"fmt"
	"sort"
	"strings"
)

// cacheKindPrefixes maps logical kind names to key prefixes used in withCache (see client.go).
var cacheKindPrefixes = map[string][]string{
	"games":            {"nx|games"},
	"game":             {"nx|game|"},
	"mod":              {"nx|mod|"},
	"modfiles":         {"nx|modfiles|"},
	"modfile":          {"nx|modfile|"},
	"changelog":        {"nx|changelog|"},
	"feeds":            {"nx|latest_updated|", "nx|latest_added|", "nx|trending|", "nx|recently_updated|"},
	"modreq":           {"nx|modreq|"},
	"mod_requirements": {"nx|modreq|"},
}

// KnownCacheKinds returns sorted valid kind names for InvalidateCacheKind errors and docs.
func KnownCacheKinds() []string {
	out := make([]string, 0, len(cacheKindPrefixes))
	for k := range cacheKindPrefixes {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func validateCachePrefix(p string) error {
	p = strings.TrimSpace(p)
	if p == "" {
		return fmt.Errorf("prefix must be non-empty")
	}
	if !strings.HasPrefix(p, "nx|") {
		return fmt.Errorf("prefix must start with nx|")
	}
	return nil
}
