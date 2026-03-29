package nexus

import (
	"fmt"
	"os"
	"strings"
	"time"
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
	// CacheTTL is the time-to-live for in-memory response caching. Zero disables caching.
	CacheTTL time.Duration
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
	cacheTTL, err := parseCacheTTL(os.Getenv("NEXUSMODS_CACHE_TTL"))
	if err != nil {
		return Config{}, err
	}
	return Config{
		APIKey:             key,
		ApplicationName:    app,
		ApplicationVersion: ver,
		ProtocolVersion:    proto,
		RESTBaseURL:        rest,
		GraphQLURL:         gql,
		CacheTTL:           cacheTTL,
	}, nil
}

func parseCacheTTL(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 24 * time.Hour, nil
	}
	low := strings.ToLower(raw)
	if low == "0" || low == "off" || low == "false" || low == "disable" || low == "disabled" {
		return 0, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("NEXUSMODS_CACHE_TTL: %w", err)
	}
	if d < 0 {
		return 0, fmt.Errorf("NEXUSMODS_CACHE_TTL must be >= 0")
	}
	return d, nil
}
