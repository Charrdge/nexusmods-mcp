# nexusmods-mcp

MCP-сервер на Go для [Nexus Mods](https://www.nexusmods.com/): список игр, поиск модов (GraphQL v2), карточка мода, файлы (REST v1), зависимости и «кто зависит от мода» (GraphQL `mod.modRequirements`).

- **stdio** (по умолчанию): `docker run -i` — Cursor, OpenClaw, `mcp-smoke`.
- **HTTP** (`MCP_TRANSPORT=http`): streamable HTTP для клиентов с поддержкой; см. [docs/MCP.md](docs/MCP.md). Не публикуй порт без защиты.

Код: `cmd/server`, `internal/nexus`, `internal/toolreg`.

**Tools (5):** `nexus_games`, `nexus_search_mods`, `nexus_get_mod`, `nexus_list_mod_files`, `nexus_get_mod_requirements` — см. [docs/MCP.md](docs/MCP.md).

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

Шаблон переменных: [.env.example](.env.example). Ключ API: [Account → API](https://www.nexusmods.com/users/myaccount?tab=api).

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
