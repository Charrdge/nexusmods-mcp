package nexus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func pageCountOrDefault(n int) int {
	if n <= 0 {
		return 20
	}
	if n > 50 {
		return 50
	}
	return n
}

// Client calls Nexus Mods REST v1 and GraphQL v2 with required headers.
type Client struct {
	cfg  Config
	http *http.Client
	ua   string
}

// NewClient builds an HTTP client with Nexus authentication headers.
func NewClient(cfg Config) *Client {
	ua := fmt.Sprintf("%s/%s (nexusmods-mcp)", cfg.ApplicationName, cfg.ApplicationVersion)
	return &Client{
		cfg: cfg,
		http: &http.Client{
			Timeout: 60 * time.Second,
		},
		ua: ua,
	}
}

func (c *Client) baseHeaders() http.Header {
	h := make(http.Header)
	h.Set("APIKEY", c.cfg.APIKey)
	h.Set("Application-Name", c.cfg.ApplicationName)
	h.Set("Application-Version", c.cfg.ApplicationVersion)
	h.Set("Protocol-Version", c.cfg.ProtocolVersion)
	h.Set("Accept", "application/json")
	h.Set("User-Agent", c.ua)
	return h
}

func (c *Client) getJSON(ctx context.Context, rawURL string) (json.RawMessage, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header = c.baseHeaders()
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("nexus API %s: %s", resp.Status, truncate(string(body), 500))
	}
	return json.RawMessage(body), resp.StatusCode, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// Games returns all games (same as GET /v1/games.json).
func (c *Client) Games(ctx context.Context) (json.RawMessage, error) {
	u := c.cfg.RESTBaseURL + "/games.json"
	data, _, err := c.getJSON(ctx, u)
	return data, err
}

// GameIDForDomain resolves the numeric game id from REST games.json domain_name (e.g. skyrimspecialedition).
// Nexus GraphQL mod(...) expects this id as gameId, not the domain string.
func (c *Client) GameIDForDomain(ctx context.Context, domain string) (int64, error) {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return 0, fmt.Errorf("game_domain is required")
	}
	data, err := c.Games(ctx)
	if err != nil {
		return 0, err
	}
	var games []struct {
		ID         int64  `json:"id"`
		DomainName string `json:"domain_name"`
	}
	if err := json.Unmarshal(data, &games); err != nil {
		return 0, fmt.Errorf("games.json: %w", err)
	}
	for _, g := range games {
		if strings.EqualFold(g.DomainName, domain) {
			return g.ID, nil
		}
	}
	return 0, fmt.Errorf("unknown game_domain %q", domain)
}

// Mod returns mod metadata for a game domain and mod ID.
func (c *Client) Mod(ctx context.Context, gameDomain string, modID int64) (json.RawMessage, error) {
	gameDomain = strings.TrimSpace(gameDomain)
	if gameDomain == "" {
		return nil, fmt.Errorf("game_domain is required")
	}
	u := fmt.Sprintf("%s/games/%s/mods/%d.json", c.cfg.RESTBaseURL, url.PathEscape(gameDomain), modID)
	data, _, err := c.getJSON(ctx, u)
	return data, err
}

// ModFiles returns file list for a mod.
func (c *Client) ModFiles(ctx context.Context, gameDomain string, modID int64) (json.RawMessage, error) {
	gameDomain = strings.TrimSpace(gameDomain)
	if gameDomain == "" {
		return nil, fmt.Errorf("game_domain is required")
	}
	u := fmt.Sprintf("%s/games/%s/mods/%d/files.json", c.cfg.RESTBaseURL, url.PathEscape(gameDomain), modID)
	data, _, err := c.getJSON(ctx, u)
	return data, err
}

// SearchMods runs GraphQL `mods` with game domain + wildcard name filter (REST v1 has no text search).
func (c *Client) SearchMods(ctx context.Context, gameDomain, query string, offset, count int) (json.RawMessage, error) {
	gameDomain = strings.TrimSpace(gameDomain)
	query = strings.TrimSpace(query)
	if gameDomain == "" {
		return nil, fmt.Errorf("game_domain is required")
	}
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	count = pageCountOrDefault(count)
	if offset < 0 {
		offset = 0
	}
	// Wildcard partial match on mod name (see ModsFilter.name + WILDCARD).
	pattern := "*" + escapeGraphQLWildcard(query) + "*"
	filter := map[string]any{
		"op": "AND",
		"gameDomainName": []map[string]string{
			{"value": gameDomain, "op": "EQUALS"},
		},
		"name": []map[string]string{
			{"value": pattern, "op": "WILDCARD"},
		},
	}
	variables := map[string]any{
		"filter": filter,
		"offset": offset,
		"count":  count,
	}
	gqlQuery := `query ModSearch($filter: ModsFilter!, $offset: Int, $count: Int) {
  mods(filter: $filter, offset: $offset, count: $count) {
    totalCount
    nodes {
      modId
      name
      summary
      downloads
      endorsements
    }
  }
}`
	payload := map[string]any{
		"query":     gqlQuery,
		"variables": variables,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.GraphQLURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header = c.baseHeaders()
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("nexus GraphQL %s: %s", resp.Status, truncate(string(respBody), 800))
	}
	return json.RawMessage(respBody), nil
}

// ModRequirements returns GraphQL mod.modRequirements: mods this one requires (nexusRequirements),
// mods that require this one (modsRequiringThisMod), and DLC requirements.
func (c *Client) ModRequirements(ctx context.Context, gameDomain string, modID int64, reqOff, reqCount, depOff, depCount int) (json.RawMessage, error) {
	gid, err := c.GameIDForDomain(ctx, gameDomain)
	if err != nil {
		return nil, err
	}
	if reqOff < 0 {
		reqOff = 0
	}
	if depOff < 0 {
		depOff = 0
	}
	reqCount = pageCountOrDefault(reqCount)
	depCount = pageCountOrDefault(depCount)

	gqlQuery := `query ModRequirements($modId: ID!, $gameId: ID!, $reqOff: Int!, $reqCnt: Int!, $depOff: Int!, $depCnt: Int!) {
  mod(modId: $modId, gameId: $gameId) {
    modId
    name
    modRequirements {
      dlcRequirements {
        notes
        gameExpansion { id name }
      }
      nexusRequirements(offset: $reqOff, count: $reqCnt) {
        totalCount
        nodes {
          modId
          modName
          url
          notes
          externalRequirement
        }
      }
      modsRequiringThisMod(offset: $depOff, count: $depCnt) {
        totalCount
        nodes {
          modId
          modName
          url
          notes
          externalRequirement
        }
      }
    }
  }
}`
	variables := map[string]any{
		"modId":  strconv.FormatInt(modID, 10),
		"gameId": strconv.FormatInt(gid, 10),
		"reqOff": reqOff,
		"reqCnt": reqCount,
		"depOff": depOff,
		"depCnt": depCount,
	}
	payload := map[string]any{
		"query":     gqlQuery,
		"variables": variables,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.GraphQLURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header = c.baseHeaders()
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("nexus GraphQL %s: %s", resp.Status, truncate(string(respBody), 800))
	}
	return json.RawMessage(respBody), nil
}

func escapeGraphQLWildcard(s string) string {
	// Avoid injecting extra wildcards; strip characters that break the pattern.
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "?", "")
	return s
}

// ParseInt parses a string as int64 for tool args.
func ParseInt(s string, name string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("%s is required", name)
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("invalid %s", name)
	}
	return n, nil
}
