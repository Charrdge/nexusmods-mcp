# План тестирования

Из корня репозитория (например `~/nexusmods-mcp` в WSL).

## Go

```bash
cd ~/nexusmods-mcp
go vet ./...
go test ./... -count=1
```

Опционально с гонками:

```bash
go test ./... -race -count=1
```

## Docker и образ

```bash
docker build -t nexus-mcp:local .
```

Проверка без ключа (процесс должен завершиться с сообщением о необходимости ключа):

```bash
docker run --rm -i nexus-mcp:local
# ожидается: NEXUSMODS_API_KEY is required
```

С ключом процесс **блокируется** на stdin — это нормально для stdio MCP:

```bash
docker run --rm -i --env-file /absolute/path/to/.env nexus-mcp:local
```

## MCP smoke (stdio + реальные вызовы Nexus)

Требуется валидный `NEXUSMODS_API_KEY` в окружении бинарника или в `.env` для Docker.

Через Docker:

```bash
go run ./cmd/mcp-smoke -docker -env-file "/absolute/path/to/.env"
```

Локальный бинарь (переменные уже в shell или в `.env` не подхватываются автоматически — экспортируй `NEXUSMODS_API_KEY`):

```bash
go build -o nexusmods-mcp ./cmd/server
export NEXUSMODS_API_KEY=...
go run ./cmd/mcp-smoke -bin ./nexusmods-mcp
```

Ожидаемый вывод включает `OK initialize`, строку `OK tools/list contains all 16 tools`, вызовы tools и в конце `ALL_OK`. Для `nexus_search_mods` при сбое GraphQL возможно предупреждение `WARN` (smoke не падает только на этом шаге); остальные tool-вызовы должны завершиться без `fatalf`.

## Переменные для интеграционных проверок

| Переменная | Назначение |
|------------|------------|
| `NEXUSMODS_API_KEY` | Обязательна для любых запросов к Nexus |
| Остальные `NEXUSMODS_*` | См. [docs/MCP.md](docs/MCP.md) |

Для `mcp-smoke -docker` все нужные переменные задаются через `--env-file`.

## Матрица сбоев

| Симптом | Проверить |
|---------|-----------|
| `NEXUSMODS_API_KEY is required` | Ключ в `env` / `--env-file` / секреты IDE |
| `connection refused` / сеть | Доступность `api.nexusmods.com`, прокси, firewall |
| 401 / invalid API key | Ключ в [Account → API](https://www.nexusmods.com/users/myaccount?tab=api) |
| MCP-клиент не коннектится к Docker | Флаг `-i` у `docker run`, корректный `command`/`args` в настройках клиента |
| `timeout waiting for jsonrpc` | Логи stderr сервера, не завершился ли процесс с `log.Fatal` |
| В ответе GraphQL `errors`, `data`: null | Неверный `game_domain` / `mod_id`; для зависимостей см. [docs/MCP.md](docs/MCP.md) (`nexus_get_mod_requirements`) |
| HTTP-режим недоступен снаружи | `MCP_HTTP_ADDR=0.0.0.0:8080`, порт проброшен; помни про безопасность |

## Дополнительно

- [MCP Inspector](https://github.com/modelcontextprotocol/inspector) — ручная проверка списка tools и вызовов.
- Пошаговый чеклист: [AGENT-TEST-PLAYBOOK.md](AGENT-TEST-PLAYBOOK.md).
