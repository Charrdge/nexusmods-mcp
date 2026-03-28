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

// postGraphQL sends a JSON body to the Nexus GraphQL endpoint and returns the raw response bytes.
func (c *Client) postGraphQL(ctx context.Context, query string, variables map[string]any) (json.RawMessage, error) {
	payload := map[string]any{
		"query":     query,
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

// ModChangelog returns changelog entries for a mod (REST).
func (c *Client) ModChangelog(ctx context.Context, gameDomain string, modID int64) (json.RawMessage, error) {
	gameDomain = strings.TrimSpace(gameDomain)
	if gameDomain == "" {
		return nil, fmt.Errorf("game_domain is required")
	}
	u := fmt.Sprintf("%s/games/%s/mods/%d/changelogs", c.cfg.RESTBaseURL, url.PathEscape(gameDomain), modID)
	data, _, err := c.getJSON(ctx, u)
	return data, err
}

// ModFile returns metadata for a single mod file by file_id (REST).
func (c *Client) ModFile(ctx context.Context, gameDomain string, modID, fileID int64) (json.RawMessage, error) {
	gameDomain = strings.TrimSpace(gameDomain)
	if gameDomain == "" {
		return nil, fmt.Errorf("game_domain is required")
	}
	u := fmt.Sprintf("%s/games/%s/mods/%d/files/%d", c.cfg.RESTBaseURL, url.PathEscape(gameDomain), modID, fileID)
	data, _, err := c.getJSON(ctx, u)
	return data, err
}

// Game returns details for one game by domain (REST), including category tree.
func (c *Client) Game(ctx context.Context, gameDomain string) (json.RawMessage, error) {
	gameDomain = strings.TrimSpace(gameDomain)
	if gameDomain == "" {
		return nil, fmt.Errorf("game_domain is required")
	}
	u := fmt.Sprintf("%s/games/%s.json", c.cfg.RESTBaseURL, url.PathEscape(gameDomain))
	data, _, err := c.getJSON(ctx, u)
	return data, err
}

// GameCategories returns only the categories array from GET /games/{domain}.json.
func (c *Client) GameCategories(ctx context.Context, gameDomain string) (json.RawMessage, error) {
	data, err := c.Game(ctx, gameDomain)
	if err != nil {
		return nil, err
	}
	var obj struct {
		Categories json.RawMessage `json:"categories"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("game json: %w", err)
	}
	if obj.Categories == nil {
		return json.RawMessage(`{"categories":[]}`), nil
	}
	wrapped := map[string]json.RawMessage{"categories": obj.Categories}
	out, err := json.Marshal(wrapped)
	return json.RawMessage(out), err
}

// ModsLatestUpdated returns recently updated mods for a game (REST).
func (c *Client) ModsLatestUpdated(ctx context.Context, gameDomain string) (json.RawMessage, error) {
	gameDomain = strings.TrimSpace(gameDomain)
	if gameDomain == "" {
		return nil, fmt.Errorf("game_domain is required")
	}
	u := fmt.Sprintf("%s/games/%s/mods/latest_updated.json", c.cfg.RESTBaseURL, url.PathEscape(gameDomain))
	data, _, err := c.getJSON(ctx, u)
	return data, err
}

// ModsLatestAdded returns recently added mods for a game (REST).
func (c *Client) ModsLatestAdded(ctx context.Context, gameDomain string) (json.RawMessage, error) {
	gameDomain = strings.TrimSpace(gameDomain)
	if gameDomain == "" {
		return nil, fmt.Errorf("game_domain is required")
	}
	u := fmt.Sprintf("%s/games/%s/mods/latest_added.json", c.cfg.RESTBaseURL, url.PathEscape(gameDomain))
	data, _, err := c.getJSON(ctx, u)
	return data, err
}

// ModsTrending returns trending mods for a game (REST).
func (c *Client) ModsTrending(ctx context.Context, gameDomain string) (json.RawMessage, error) {
	gameDomain = strings.TrimSpace(gameDomain)
	if gameDomain == "" {
		return nil, fmt.Errorf("game_domain is required")
	}
	u := fmt.Sprintf("%s/games/%s/mods/trending.json", c.cfg.RESTBaseURL, url.PathEscape(gameDomain))
	data, _, err := c.getJSON(ctx, u)
	return data, err
}

// ModsRecentlyUpdated returns mods updated in a server-cached period: 1d, 1w, or 1m (REST).
func (c *Client) ModsRecentlyUpdated(ctx context.Context, gameDomain, period string) (json.RawMessage, error) {
	gameDomain = strings.TrimSpace(gameDomain)
	if gameDomain == "" {
		return nil, fmt.Errorf("game_domain is required")
	}
	period = strings.TrimSpace(strings.ToLower(period))
	if period != "1d" && period != "1w" && period != "1m" {
		return nil, fmt.Errorf("period must be 1d, 1w, or 1m")
	}
	u := fmt.Sprintf("%s/games/%s/mods/updated.json?period=%s", c.cfg.RESTBaseURL, url.PathEscape(gameDomain), url.QueryEscape(period))
	data, _, err := c.getJSON(ctx, u)
	return data, err
}

// TrackedMods returns mods tracked by the API key owner (REST).
func (c *Client) TrackedMods(ctx context.Context) (json.RawMessage, error) {
	u := c.cfg.RESTBaseURL + "/user/tracked_mods.json"
	data, _, err := c.getJSON(ctx, u)
	return data, err
}

// RateLimitHeaders performs a lightweight GET and returns x-rl-* response headers as JSON.
func (c *Client) RateLimitHeaders(ctx context.Context) (json.RawMessage, error) {
	u := c.cfg.RESTBaseURL + "/games.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header = c.baseHeaders()
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("nexus API %s", resp.Status)
	}
	out := make(map[string]string)
	for k, vals := range resp.Header {
		kl := strings.ToLower(k)
		if strings.HasPrefix(kl, "x-rl-") && len(vals) > 0 {
			out[k] = vals[0]
		}
	}
	raw, err := json.Marshal(out)
	return json.RawMessage(raw), err
}

// SearchMods runs GraphQL `mods` with game domain + optional name wildcard and optional author/categoryName (EQUALS).
func (c *Client) SearchMods(ctx context.Context, gameDomain, query, author, categoryName string, offset, count int) (json.RawMessage, error) {
	gameDomain = strings.TrimSpace(gameDomain)
	if gameDomain == "" {
		return nil, fmt.Errorf("game_domain is required")
	}
	query = strings.TrimSpace(query)
	author = strings.TrimSpace(author)
	categoryName = strings.TrimSpace(categoryName)
	if query == "" && author == "" && categoryName == "" {
		return nil, fmt.Errorf("provide query and/or author and/or category_name")
	}
	count = pageCountOrDefault(count)
	if offset < 0 {
		offset = 0
	}
	filter := map[string]any{
		"op": "AND",
		"gameDomainName": []map[string]string{
			{"value": gameDomain, "op": "EQUALS"},
		},
	}
	if query != "" {
		pattern := "*" + escapeGraphQLWildcard(query) + "*"
		filter["name"] = []map[string]string{
			{"value": pattern, "op": "WILDCARD"},
		}
	}
	if author != "" {
		filter["author"] = []map[string]string{
			{"value": author, "op": "EQUALS"},
		}
	}
	if categoryName != "" {
		filter["categoryName"] = []map[string]string{
			{"value": categoryName, "op": "EQUALS"},
		}
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
	return c.postGraphQL(ctx, gqlQuery, variables)
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
	return c.postGraphQL(ctx, gqlQuery, variables)
}

// ModGraphQL returns selected GraphQL mod fields including viewer* flags (requires same API key as other calls).
func (c *Client) ModGraphQL(ctx context.Context, gameDomain string, modID int64) (json.RawMessage, error) {
	gid, err := c.GameIDForDomain(ctx, gameDomain)
	if err != nil {
		return nil, err
	}
	gqlQuery := `query ModGraph($modId: ID!, $gameId: ID!) {
  mod(modId: $modId, gameId: $gameId) {
    modId
    name
    summary
    description
    version
    updatedAt
    createdAt
    downloads
    endorsements
    status
    viewerUpdateAvailable
    viewerTracked
    viewerEndorsed
    viewerDownloaded
    viewerBlocked
    viewerIsBlocked
  }
}`
	variables := map[string]any{
		"modId":  strconv.FormatInt(modID, 10),
		"gameId": strconv.FormatInt(gid, 10),
	}
	return c.postGraphQL(ctx, gqlQuery, variables)
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
