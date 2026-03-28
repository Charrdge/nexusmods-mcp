package toolreg

import (
	"bytes"
	"context"
	"encoding/json"
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
		GameDomain string `json:"game_domain" jsonschema:"Nexus game domain, e.g. skyrimspecialedition"`
		Query      string `json:"query" jsonschema:"Free-text search; matched against mod names (GraphQL wildcard search)"`
		Offset     string `json:"offset,omitempty" jsonschema:"Optional result offset (default 0)"`
		Count      string `json:"count,omitempty" jsonschema:"Optional page size 1–50 (default 20)"`
	}
	mcp.AddTool(server, &mcp.Tool{
		Name:        "nexus_search_mods",
		Description: "Search mods by name for a game (uses Nexus GraphQL; REST v1 has no text search).",
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
		data, err := nx.SearchMods(ctx, args.GameDomain, args.Query, off, cnt)
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
