package nexus

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// testClient builds a Client that talks to srv; same package so fields are settable.
func testClient(srv *httptest.Server, cacheTTL time.Duration) *Client {
	ttl := cacheTTL
	var cache *apiCache
	if ttl > 0 {
		cache = newAPICache(ttl)
	}
	return &Client{
		cfg: Config{
			APIKey:             "test-key",
			ApplicationName:    "test",
			ApplicationVersion: "0",
			ProtocolVersion:    "1",
			RESTBaseURL:        strings.TrimRight(srv.URL, "/"),
			GraphQLURL:         strings.TrimRight(srv.URL, "/") + "/graphql",
			CacheTTL:           ttl,
		},
		http:  srv.Client(),
		ua:    "test/0 (nexusmods-mcp)",
		cache: cache,
	}
}

func TestClientGamesSecondCallUsesCache(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/games.json" {
			http.NotFound(w, r)
			return
		}
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":42,"domain_name":"skyrim"}]`))
	}))
	defer srv.Close()

	cl := testClient(srv, time.Hour)
	ctx := context.Background()

	_, err := cl.Games(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = cl.Games(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected 1 HTTP GET, got %d (cache should skip second request)", calls.Load())
	}
}

func TestClientGamesNoCacheTTLTwoCalls(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/games.json" {
			http.NotFound(w, r)
			return
		}
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	cl := testClient(srv, 0)
	ctx := context.Background()
	_, _ = cl.Games(ctx)
	_, _ = cl.Games(ctx)
	if calls.Load() != 2 {
		t.Fatalf("with cache off, expected 2 HTTP calls, got %d", calls.Load())
	}
}

func TestClientModFilesCachesPerKey(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/files.json") {
			http.NotFound(w, r)
			return
		}
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"files":[]}`))
	}))
	defer srv.Close()

	cl := testClient(srv, time.Hour)
	ctx := context.Background()

	_, _ = cl.ModFiles(ctx, "skyrim", 1)
	_, _ = cl.ModFiles(ctx, "skyrim", 1)
	_, _ = cl.ModFiles(ctx, "skyrim", 2)
	if calls.Load() != 2 {
		t.Fatalf("expected 2 distinct mod file list requests, got %d", calls.Load())
	}
}

func TestClientSearchModsNeverUsesResponseCache(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/graphql" {
			http.NotFound(w, r)
			return
		}
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"mods":{"totalCount":0,"nodes":[]}}}`))
	}))
	defer srv.Close()

	cl := testClient(srv, time.Hour)
	ctx := context.Background()

	_, _ = cl.SearchMods(ctx, "g", "q", "", "", 0, 10)
	_, _ = cl.SearchMods(ctx, "g", "q", "", "", 0, 10)
	if calls.Load() != 2 {
		t.Fatalf("SearchMods should not cache; expected 2 GraphQL POSTs, got %d", calls.Load())
	}
}

func TestClientGamesBypassForcesNetworkAndRefills(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/games.json" {
			http.NotFound(w, r)
			return
		}
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	cl := testClient(srv, time.Hour)
	ctx := context.Background()
	_, _ = cl.Games(ctx)
	_, _ = cl.Games(ctx)
	if calls.Load() != 1 {
		t.Fatalf("first two Games: want 1 HTTP call, got %d", calls.Load())
	}
	_, _ = cl.Games(WithCacheBypass(ctx, true))
	if calls.Load() != 2 {
		t.Fatalf("bypass Games: want 2 HTTP calls total, got %d", calls.Load())
	}
	_, _ = cl.Games(ctx)
	if calls.Load() != 2 {
		t.Fatalf("after bypass refill, cached read: want 2 HTTP calls total, got %d", calls.Load())
	}
}

func TestClientInvalidateCacheAll(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/games.json" {
			http.NotFound(w, r)
			return
		}
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	cl := testClient(srv, time.Hour)
	ctx := context.Background()
	_, _ = cl.Games(ctx)
	_, _ = cl.Games(ctx)
	rm := cl.InvalidateCacheAll()
	if rm != 1 {
		t.Fatalf("InvalidateCacheAll: want 1 removed, got %d", rm)
	}
	_, _ = cl.Games(ctx)
	if calls.Load() != 2 {
		t.Fatalf("after invalidate, want 2 HTTP GETs total, got %d", calls.Load())
	}
}

func TestClientInvalidateCachePrefixInvalid(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	cl := testClient(srv, time.Hour)
	_, err := cl.InvalidateCachePrefix("bad")
	if err == nil {
		t.Fatal("expected error for prefix without nx|")
	}
}

func TestClientInvalidateCacheKindUnknown(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	cl := testClient(srv, time.Hour)
	_, err := cl.InvalidateCacheKind("not_a_kind")
	if err == nil {
		t.Fatal("expected error")
	}
}
