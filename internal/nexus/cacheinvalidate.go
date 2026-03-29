package nexus

import (
	"fmt"
	"strings"
)

// CacheEnabled reports whether this client stores API responses in memory.
func (c *Client) CacheEnabled() bool {
	return c.cache != nil
}

// InvalidateCacheAll drops every cached Nexus response for this process. Returns entries removed.
func (c *Client) InvalidateCacheAll() int {
	if c.cache == nil {
		return 0
	}
	return c.cache.clearAll()
}

// InvalidateCachePrefix removes entries whose keys start with prefix (must begin with nx|).
func (c *Client) InvalidateCachePrefix(prefix string) (int, error) {
	if c.cache == nil {
		return 0, nil
	}
	p := strings.TrimSpace(prefix)
	if err := validateCachePrefix(p); err != nil {
		return 0, err
	}
	return c.cache.deletePrefix(p), nil
}

// InvalidateCacheKind removes entries for a logical group (games, mod, feeds, …).
func (c *Client) InvalidateCacheKind(kind string) (int, error) {
	k := strings.ToLower(strings.TrimSpace(kind))
	if c.cache == nil {
		return 0, nil
	}
	prefixes, ok := cacheKindPrefixes[k]
	if !ok {
		return 0, fmt.Errorf("unknown cache kind %q; valid: %s", kind, strings.Join(KnownCacheKinds(), ", "))
	}
	n := 0
	for _, p := range prefixes {
		n += c.cache.deletePrefix(p)
	}
	return n, nil
}
