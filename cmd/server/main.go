// Command server is the Nexus Mods MCP server: stdio (default) or HTTP streamable (daemon).
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/charrdge/nexusmods-mcp/internal/nexus"
	"github.com/charrdge/nexusmods-mcp/internal/toolreg"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	cfg, err := nexus.ConfigFromEnv()
	if err != nil {
		log.Fatalf("nexusmods-mcp: %v", err)
	}
	client := nexus.NewClient(cfg)
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "nexusmods-mcp",
		Version: "0.1.0",
	}, nil)
	toolreg.Register(server, client)

	mode := strings.ToLower(strings.TrimSpace(os.Getenv("MCP_TRANSPORT")))
	if mode == "" || mode == "stdio" {
		if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			log.Fatalf("server: %v", err)
		}
		return
	}

	if mode != "http" {
		log.Fatalf("nexusmods-mcp: unknown MCP_TRANSPORT=%q (use stdio or http)", mode)
	}

	addr := strings.TrimSpace(os.Getenv("MCP_HTTP_ADDR"))
	if addr == "" {
		addr = ":8080"
	}
	handler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return server
	}, nil)
	log.Printf("nexusmods-mcp streamable HTTP on %s (set MCP_TRANSPORT=stdio for Cursor docker -i)", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
