package toolreg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/charrdge/nexusmods-mcp/internal/nexus"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register wires Nexus-backed tools onto the MCP server.
func Register(server *mcp.Server, nx *nexus.Client) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "nexus_games",
		Description: "List all games on Nexus Mods (domain name, id, and metadata). Use this to resolve game_domain for other tools.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		data, err := nx.Games(ctx)
		return jsonResult(data, err)
	})

	type searchArgs struct {
		GameDomain   string `json:"game_domain" jsonschema:"Nexus game domain, e.g. skyrimspecialedition"`
		Query        string `json:"query,omitempty" jsonschema:"Optional; wildcard on stemmed mod name (GraphQL nameStemmed). Use with author/category_name or alone."`
		Author       string `json:"author,omitempty" jsonschema:"Optional; exact match on author display name (GraphQL ModsFilter)"`
		CategoryName string `json:"category_name,omitempty" jsonschema:"Optional; exact match on category name (GraphQL ModsFilter)"`
		Offset       string `json:"offset,omitempty" jsonschema:"Optional result offset (default 0)"`
		Count        string `json:"count,omitempty" jsonschema:"Optional page size 1–50 (default 20)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "nexus_search_mods",
		Description: "Search mods for a game via GraphQL: optional stemmed-name wildcard (query → ModsFilter nameStemmed), optional exact author and category_name. At least one of query, author, category_name is required.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args searchArgs) (*mcp.CallToolResult, any, error) {
		off := 0
		if strings.TrimSpace(args.Offset) != "" {
			v, err := strconv.Atoi(strings.TrimSpace(args.Offset))
			if err != nil || v < 0 {
				return toolErr("invalid offset"), nil, nil
			}
			off = v
		}
		cnt := 0
		if strings.TrimSpace(args.Count) != "" {
			v, err := strconv.Atoi(strings.TrimSpace(args.Count))
			if err != nil || v < 1 || v > 50 {
				return toolErr("invalid count (use 1–50)"), nil, nil
			}
			cnt = v
		}
		data, err := nx.SearchMods(ctx, args.GameDomain, args.Query, args.Author, args.CategoryName, off, cnt)
		return jsonResult(data, err)
	})

	type modArgs struct {
		GameDomain string `json:"game_domain" jsonschema:"Nexus game domain, e.g. skyrimspecialedition"`
		ModID      string `json:"mod_id" jsonschema:"Numeric Nexus mod id"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "nexus_get_mod",
		Description: "Get mod details (REST) for a game domain and mod id.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args modArgs) (*mcp.CallToolResult, any, error) {
		id, err := nexus.ParseInt(args.ModID, "mod_id")
		if err != nil {
			return toolErr(err.Error()), nil, nil
		}
		data, err := nx.Mod(ctx, args.GameDomain, id)
		return jsonResult(data, err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "nexus_list_mod_files",
		Description: "List all files for a mod (REST): archives, versions, categories.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args modArgs) (*mcp.CallToolResult, any, error) {
		id, err := nexus.ParseInt(args.ModID, "mod_id")
		if err != nil {
			return toolErr(err.Error()), nil, nil
		}
		data, err := nx.ModFiles(ctx, args.GameDomain, id)
		return jsonResult(data, err)
	})

	type modReqArgs struct {
		GameDomain         string `json:"game_domain" jsonschema:"Nexus game domain, e.g. skyrimspecialedition"`
		ModID              string `json:"mod_id" jsonschema:"Numeric Nexus mod id"`
		RequirementsOffset string `json:"requirements_offset,omitempty" jsonschema:"Optional offset for required-mods list (default 0)"`
		RequirementsCount  string `json:"requirements_count,omitempty" jsonschema:"Optional page size 1–50 for required mods (default 20)"`
		DependentsOffset   string `json:"dependents_offset,omitempty" jsonschema:"Optional offset for mods-requiring-this list (default 0)"`
		DependentsCount    string `json:"dependents_count,omitempty" jsonschema:"Optional page size 1–50 for dependents (default 20)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "nexus_get_mod_requirements",
		Description: "Mod dependencies via GraphQL: mods this mod requires (nexusRequirements), mods that require this mod (modsRequiringThisMod), and DLC requirements. Uses game_domain like other tools; resolves internal game id automatically.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args modReqArgs) (*mcp.CallToolResult, any, error) {
		id, err := nexus.ParseInt(args.ModID, "mod_id")
		if err != nil {
			return toolErr(err.Error()), nil, nil
		}
		parseOff := func(s string) (int, error) {
			s = strings.TrimSpace(s)
			if s == "" {
				return 0, nil
			}
			v, err := strconv.Atoi(s)
			if err != nil || v < 0 {
				return 0, fmt.Errorf("invalid offset")
			}
			return v, nil
		}
		parseCnt := func(s string) (int, error) {
			s = strings.TrimSpace(s)
			if s == "" {
				return 0, nil
			}
			v, err := strconv.Atoi(s)
			if err != nil || v < 1 || v > 50 {
				return 0, fmt.Errorf("invalid count (use 1–50)")
			}
			return v, nil
		}
		reqOff, err := parseOff(args.RequirementsOffset)
		if err != nil {
			return toolErr(err.Error()), nil, nil
		}
		reqCnt, err := parseCnt(args.RequirementsCount)
		if err != nil {
			return toolErr(err.Error()), nil, nil
		}
		depOff, err := parseOff(args.DependentsOffset)
		if err != nil {
			return toolErr(err.Error()), nil, nil
		}
		depCnt, err := parseCnt(args.DependentsCount)
		if err != nil {
			return toolErr(err.Error()), nil, nil
		}
		data, err := nx.ModRequirements(ctx, args.GameDomain, id, reqOff, reqCnt, depOff, depCnt)
		return jsonResult(data, err)
	})

	type fileArgs struct {
		GameDomain string `json:"game_domain" jsonschema:"Nexus game domain"`
		ModID      string `json:"mod_id" jsonschema:"Numeric Nexus mod id"`
		FileID     string `json:"file_id" jsonschema:"Numeric Nexus file id (from list mod files)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "nexus_get_mod_changelog",
		Description: "Changelog history for a mod (REST).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args modArgs) (*mcp.CallToolResult, any, error) {
		id, err := nexus.ParseInt(args.ModID, "mod_id")
		if err != nil {
			return toolErr(err.Error()), nil, nil
		}
		data, err := nx.ModChangelog(ctx, args.GameDomain, id)
		return jsonResult(data, err)
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "nexus_get_mod_file",
		Description: "Metadata for one mod file by file_id (REST); avoids downloading full files list when you already know file_id.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args fileArgs) (*mcp.CallToolResult, any, error) {
		mid, err := nexus.ParseInt(args.ModID, "mod_id")
		if err != nil {
			return toolErr(err.Error()), nil, nil
		}
		fid, err := nexus.ParseInt(args.FileID, "file_id")
		if err != nil {
			return toolErr(err.Error()), nil, nil
		}
		data, err := nx.ModFile(ctx, args.GameDomain, mid, fid)
		return jsonResult(data, err)
	})

	type gameDomainArg struct {
		GameDomain string `json:"game_domain" jsonschema:"Nexus game domain, e.g. skyrimspecialedition"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "nexus_get_game",
		Description: "Single game record by domain (REST), including category tree and stats.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args gameDomainArg) (*mcp.CallToolResult, any, error) {
		data, err := nx.Game(ctx, args.GameDomain)
		return jsonResult(data, err)
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "nexus_game_categories",
		Description: "Category tree for a game (subset of nexus_get_game): {\"categories\":[...]}.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args gameDomainArg) (*mcp.CallToolResult, any, error) {
		data, err := nx.GameCategories(ctx, args.GameDomain)
		return jsonResult(data, err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "nexus_mods_latest_updated",
		Description: "Latest updated mods for a game (REST feed).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args gameDomainArg) (*mcp.CallToolResult, any, error) {
		data, err := nx.ModsLatestUpdated(ctx, args.GameDomain)
		return jsonResult(data, err)
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "nexus_mods_latest_added",
		Description: "Latest added mods for a game (REST feed).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args gameDomainArg) (*mcp.CallToolResult, any, error) {
		data, err := nx.ModsLatestAdded(ctx, args.GameDomain)
		return jsonResult(data, err)
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "nexus_mods_trending",
		Description: "Trending mods for a game (REST feed).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args gameDomainArg) (*mcp.CallToolResult, any, error) {
		data, err := nx.ModsTrending(ctx, args.GameDomain)
		return jsonResult(data, err)
	})

	type recentArgs struct {
		GameDomain string `json:"game_domain" jsonschema:"Nexus game domain"`
		Period     string `json:"period" jsonschema:"One of: 1d, 1w, 1m (server-cached update windows)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "nexus_mods_recently_updated",
		Description: "Mods updated in a cached time window for a game (REST): period 1d, 1w, or 1m.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args recentArgs) (*mcp.CallToolResult, any, error) {
		data, err := nx.ModsRecentlyUpdated(ctx, args.GameDomain, args.Period)
		return jsonResult(data, err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "nexus_get_tracked_mods",
		Description: "Mods tracked by the Nexus account tied to this API key (REST). Read-only.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		data, err := nx.TrackedMods(ctx)
		return jsonResult(data, err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "nexus_get_mod_graphql",
		Description: "Mod details via GraphQL (viewerUpdateAvailable, viewerTracked, description, dates, etc.). Uses numeric game id resolved from game_domain.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, args modArgs) (*mcp.CallToolResult, any, error) {
		id, err := nexus.ParseInt(args.ModID, "mod_id")
		if err != nil {
			return toolErr(err.Error()), nil, nil
		}
		data, err := nx.ModGraphQL(ctx, args.GameDomain, id)
		return jsonResult(data, err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "nexus_get_rate_limits",
		Description: "Current API rate-limit headers (x-rl-*) from a lightweight GET /games.json. For debugging; respect Nexus acceptable use policy.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		data, err := nx.RateLimitHeaders(ctx)
		return jsonResult(data, err)
	})
}

func toolErr(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		IsError: true,
	}
}

func jsonResult(data json.RawMessage, err error) (*mcp.CallToolResult, any, error) {
	if err != nil {
		return toolErr(err.Error()), nil, nil
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: buf.String()}},
	}, nil, nil
}
