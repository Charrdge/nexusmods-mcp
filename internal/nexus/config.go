package nexus

import (
	"fmt"
	"os"
	"strings"
)

const (
	defaultRESTBase   = "https://api.nexusmods.com/v1"
	defaultGraphQLURL = "https://api.nexusmods.com/v2/graphql"
	// Protocol-Version matches @nexusmods/nexus-api package version (see node-nexus-api package.json).
	defaultProtocolVersion = "1.5.2"
)

// Config holds Nexus API credentials and client identity headers.
type Config struct {
	APIKey             string
	ApplicationName    string
	ApplicationVersion string
	ProtocolVersion    string
	RESTBaseURL        string
	GraphQLURL         string
}

// ConfigFromEnv loads configuration from environment variables.
func ConfigFromEnv() (Config, error) {
	key := strings.TrimSpace(os.Getenv("NEXUSMODS_API_KEY"))
	if key == "" {
		return Config{}, fmt.Errorf("NEXUSMODS_API_KEY is required")
	}
	app := strings.TrimSpace(os.Getenv("NEXUSMODS_APPLICATION_NAME"))
	if app == "" {
		app = "nexusmods-mcp-local"
	}
	ver := strings.TrimSpace(os.Getenv("NEXUSMODS_APPLICATION_VERSION"))
	if ver == "" {
		ver = "0.1.0"
	}
	proto := strings.TrimSpace(os.Getenv("NEXUSMODS_PROTOCOL_VERSION"))
	if proto == "" {
		proto = defaultProtocolVersion
	}
	rest := strings.TrimSpace(os.Getenv("NEXUSMODS_REST_BASE"))
	if rest == "" {
		rest = defaultRESTBase
	}
	rest = strings.TrimRight(rest, "/")
	gql := strings.TrimSpace(os.Getenv("NEXUSMODS_GRAPHQL_URL"))
	if gql == "" {
		gql = defaultGraphQLURL
	}
	return Config{
		APIKey:             key,
		ApplicationName:    app,
		ApplicationVersion: ver,
		ProtocolVersion:    proto,
		RESTBaseURL:        rest,
		GraphQLURL:         gql,
	}, nil
}
