# nexusmods-mcp

MCP server for [Nexus Mods](https://www.nexusmods.com/) — games list, mod search (GraphQL), mod details and files (REST v1).

- **stdio** (default): `docker run -i` — для Cursor, OpenClaw и `mcp-smoke`.
- **HTTP** (`MCP_TRANSPORT=http`): опционально, если клиент умеет streamable HTTP; пример:  
  `docker run -d --restart unless-stopped -p 8080:8080 --env-file /abs/path/.env -e MCP_TRANSPORT=http -e MCP_HTTP_ADDR=0.0.0.0:8080 nexus-mcp:local`  
  Порт не публикуй без защиты.

Layout: `cmd/server`, `internal/nexus`, `internal/toolreg` (MCP tools).

## Environment variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NEXUSMODS_API_KEY` | yes | — | API key from [Account → API](https://www.nexusmods.com/users/myaccount?tab=api) |
| `NEXUSMODS_APPLICATION_NAME` | no | `nexusmods-mcp-local` | `Application-Name` header |
| `NEXUSMODS_APPLICATION_VERSION` | no | `0.1.0` | `Application-Version` header (semver) |
| `NEXUSMODS_PROTOCOL_VERSION` | no | `1.5.2` | `Protocol-Version` header (aligned with `@nexusmods/nexus-api`) |
| `NEXUSMODS_REST_BASE` | no | `https://api.nexusmods.com/v1` | REST base URL |
| `NEXUSMODS_GRAPHQL_URL` | no | `https://api.nexusmods.com/v2/graphql` | GraphQL endpoint (search) |
| `MCP_TRANSPORT` | no | `stdio` | `stdio` или `http` (streamable) |
| `MCP_HTTP_ADDR` | no | `:8080` | Адрес прослушивания в режиме `http`, напр. `0.0.0.0:8080` |

## Tools

- **nexus_games** — all games (domain ↔ id).
- **nexus_search_mods** — search mods by name (`game_domain`, `query`, optional `offset`, `count`).
- **nexus_get_mod** — mod JSON (`game_domain`, `mod_id`).
- **nexus_list_mod_files** — mod files (`game_domain`, `mod_id`).

Use `game_domain` like `skyrimspecialedition` (from `nexus_games`).

## Build (WSL)

From the project directory (copy under `~/nexusmods-mcp` if you keep sources only on ext4):

```bash
cd ~/nexusmods-mcp   # or: cd "/mnt/s/Skyrim SE tools/nexusmods-mcp"
docker build -t nexus-mcp:local .
```

## Run (stdio for Cursor)

```bash
docker run --rm -i \
  -e NEXUSMODS_API_KEY="YOUR_KEY" \
  -e NEXUSMODS_APPLICATION_NAME="nexusmods-mcp-local" \
  -e NEXUSMODS_APPLICATION_VERSION="0.1.0" \
  nexus-mcp:local
```

`-i` is required so the client can attach stdin for MCP.

## Cursor MCP snippet

```json
{
  "mcpServers": {
    "nexusmods": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-e", "NEXUSMODS_API_KEY=YOUR_KEY",
        "-e", "NEXUSMODS_APPLICATION_NAME=nexusmods-mcp-local",
        "-e", "NEXUSMODS_APPLICATION_VERSION=0.1.0",
        "nexus-mcp:local"
      ]
    }
  }
}
```

Do not commit real API keys. Follow [Nexus API acceptable use](https://help.nexusmods.com/article/114-api-acceptable-use-policy).

### Cursor with `.env` (no key in `mcp.json`)

Use `--env-file` and an **absolute** path Docker can read:

- WSL: `"/home/charrdge/nexusmods-mcp/.env"` or `"/mnt/s/Skyrim SE tools/nexusmods-mcp/.env"`
- Windows (Docker Desktop): e.g. `S:\\Skyrim SE tools\\nexusmods-mcp\\.env` (test `docker run ... --env-file "S:\..."` in the same environment Cursor uses)

Example fragment: [cursor-mcp-envfile.example.json](cursor-mcp-envfile.example.json). Merge into your Cursor MCP settings, replace the path, restart MCP.

## Testing

Automated smoke (MCP **newline-delimited JSON** over stdio + real Nexus calls):

```bash
cd "/mnt/s/Skyrim SE tools/nexusmods-mcp"   # or ~/nexusmods-mcp
go run ./cmd/mcp-smoke -docker -env-file "/absolute/path/to/.env"
```

Expect: `OK initialize` … `ALL_OK`.

**Level 1 (manual):**

- Without key: `docker run --rm -i nexus-mcp:local` → exits with `NEXUSMODS_API_KEY is required`.
- With key: `docker run --rm -i --env-file .env nexus-mcp:local` → blocks on stdin (normal).

Optional: [MCP Inspector](https://github.com/modelcontextprotocol/inspector) against the same `docker run ...` command.

## Local dev (Go 1.26+)

```bash
go build -o nexusmods-mcp ./cmd/server
NEXUSMODS_API_KEY=... ./nexusmods-mcp
```
