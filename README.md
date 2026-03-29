# nexusmods-mcp

MCP-сервер на Go для [Nexus Mods](https://www.nexusmods.com/): REST v1 (игры, мод, файлы, changelog, ленты, отслеживаемые моды, лимиты) и GraphQL v2 (поиск с фильтрами, зависимости мода, расширенная карточка `mod`).

- **stdio** (по умолчанию): `docker run -i` — Cursor, OpenClaw, `mcp-smoke`.
- **HTTP** (`MCP_TRANSPORT=http`): streamable HTTP для клиентов с поддержкой; см. [docs/MCP.md](docs/MCP.md). Не публикуй порт без защиты.

Код: `cmd/server`, `internal/nexus`, `internal/toolreg`.

**Tools (17):** `nexus_games`, `nexus_search_mods`, `nexus_get_mod`, `nexus_get_mod_graphql`, `nexus_list_mod_files`, `nexus_get_mod_file`, `nexus_get_mod_changelog`, `nexus_get_mod_requirements`, `nexus_get_game`, `nexus_game_categories`, `nexus_mods_latest_updated`, `nexus_mods_latest_added`, `nexus_mods_trending`, `nexus_mods_recently_updated`, `nexus_get_tracked_mods`, `nexus_get_rate_limits`, `nexus_invalidate_cache` — контракт в [docs/MCP.md](docs/MCP.md).

## Быстрый старт

Из каталога репозитория (удобно держать клон на ext4 в WSL, например `~/nexusmods-mcp`):

```bash
cd ~/nexusmods-mcp
docker build -t nexus-mcp:local .
docker run --rm -i \
  -e NEXUSMODS_API_KEY="YOUR_KEY" \
  -e NEXUSMODS_APPLICATION_NAME="nexusmods-mcp-local" \
  -e NEXUSMODS_APPLICATION_VERSION="0.1.0" \
  nexus-mcp:local
```

Флаг `-i` нужен, чтобы клиент подключился к stdin MCP.

Шаблон переменных: [.env.example](.env.example). Ключ API: [Account → API](https://www.nexusmods.com/users/myaccount?tab=api). Ответы API по умолчанию кешируются в памяти 24h (`NEXUSMODS_CACHE_TTL`) — подробности в [docs/MCP.md](docs/MCP.md).

После обновления кода снова собери образ или бинарь — иначе Cursor продолжит показывать старый список tools из `tools/list`. См. [docs/MCP.md](docs/MCP.md) (типичные ошибки).

## Документация

- [docs/MCP.md](docs/MCP.md) — транспорты, переменные окружения, контракт tools, Cursor, Claude Desktop, отладка, типичные ошибки
- [TESTING.md](TESTING.md) — `go test`, Docker, `mcp-smoke`, матрица сбоев
- [AGENT-TEST-PLAYBOOK.md](AGENT-TEST-PLAYBOOK.md) — короткий чеклист A→D

## Примеры конфигурации MCP-клиентов

| Файл | Назначение |
|------|------------|
| [cursor-mcp.json.example](cursor-mcp.json.example) | Cursor: ключ в `-e` |
| [cursor-mcp-envfile.example.json](cursor-mcp-envfile.example.json) | Cursor: `--env-file` + абсолютный путь к `.env` |
| [claude_desktop_config.json.example](claude_desktop_config.json.example) | Claude Desktop: Docker + `--env-file` |

Не коммить реальные ключи. [Nexus API acceptable use](https://help.nexusmods.com/article/114-api-acceptable-use-policy).

## Локальная разработка (Go 1.26+)

```bash
go build -o nexusmods-mcp ./cmd/server
NEXUSMODS_API_KEY=... ./nexusmods-mcp
```

Smoke по stdio: см. [TESTING.md](TESTING.md).
