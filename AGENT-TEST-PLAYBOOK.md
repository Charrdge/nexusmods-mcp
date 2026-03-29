# Playbook: проверка стека nexusmods-mcp

Из корня клона, например `~/nexusmods-mcp` в WSL.

## A — Go

```bash
go vet ./...
go test ./... -count=1
```

## B — Docker (образ и env)

```bash
docker build -t nexus-mcp:local .
docker run --rm -i nexus-mcp:local
```

Ожидание: процесс завершается с `NEXUSMODS_API_KEY is required`.

Повтори с реальным ключом (например `--env-file` с абсолютным путём к `.env`): процесс **висит** на stdin — норма для stdio.

## C — MCP smoke

```bash
go run ./cmd/mcp-smoke -docker -env-file "/absolute/path/to/.env"
```

Ожидание: цепочка `OK ...` (в т.ч. `OK tools/list contains all 17 tools`, вызовы включая `nexus_get_mod_requirements` и новые REST/GraphQL tools) и финальный `ALL_OK`.

Альтернатива без Docker:

```bash
go build -o nexusmods-mcp ./cmd/server
export NEXUSMODS_API_KEY=...
go run ./cmd/mcp-smoke -bin ./nexusmods-mcp
```

## D — Cursor / другой MCP-клиент

- Ключ в `args`: [cursor-mcp.json.example](cursor-mcp.json.example).
- Только `--env-file`: [cursor-mcp-envfile.example.json](cursor-mcp-envfile.example.json) (абсолютный путь к `.env`).

После правок конфигурации перезапусти MCP в IDE.

## Не делать

- Не коммить `.env` с реальным `NEXUSMODS_API_KEY`.
- Не публиковать `MCP_TRANSPORT=http` в открытую сеть без TLS и контроля доступа.
