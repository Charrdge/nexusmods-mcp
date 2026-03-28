# Nexus Mods MCP

Сервер [Model Context Protocol](https://modelcontextprotocol.io/) для [Nexus Mods](https://www.nexusmods.com/): список игр, поиск модов (GraphQL v2), карточка мода и список файлов (REST v1). Реализация на Go: [`github.com/modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk) v1.2.0 (см. `go.mod`). Имя и версия сервера в протоколе: `nexusmods-mcp` / `0.1.0` — [`cmd/server/main.go`](../cmd/server/main.go).

Имена и аргументы tools: [`internal/toolreg/register.go`](../internal/toolreg/register.go). HTTP-клиент Nexus: [`internal/nexus/client.go`](../internal/nexus/client.go).

## Транспорт и сборка

- **stdio** (по умолчанию): клиент запускает процесс; обмен — newline-delimited JSON по stdin/stdout. Для Docker обязателен интерактивный stdin: `docker run -i`.
- **HTTP** (streamable): `MCP_TRANSPORT=http`, слушает `MCP_HTTP_ADDR` (по умолчанию `:8080`). Удобно, если клиент умеет streamable HTTP; **не выставляй порт в интернет без TLS и авторизации**.

Сборка бинарника из корня репозитория:

```bash
go build -o nexusmods-mcp ./cmd/server
```

Образ Docker: см. [`Dockerfile`](../Dockerfile) и [`README.md`](../README.md) (быстрый старт).

## Переменные окружения

| Переменная | Обязательная | По умолчанию | Описание |
|------------|--------------|--------------|----------|
| `NEXUSMODS_API_KEY` | да | — | Ключ API: [Account → API](https://www.nexusmods.com/users/myaccount?tab=api) |
| `NEXUSMODS_APPLICATION_NAME` | нет | `nexusmods-mcp-local` | Заголовок `Application-Name` |
| `NEXUSMODS_APPLICATION_VERSION` | нет | `0.1.0` | Заголовок `Application-Version` (semver) |
| `NEXUSMODS_PROTOCOL_VERSION` | нет | `1.5.2` | Заголовок `Protocol-Version` (как у `@nexusmods/nexus-api`) |
| `NEXUSMODS_REST_BASE` | нет | `https://api.nexusmods.com/v1` | Базовый URL REST v1 |
| `NEXUSMODS_GRAPHQL_URL` | нет | `https://api.nexusmods.com/v2/graphql` | Endpoint GraphQL (поиск модов) |
| `MCP_TRANSPORT` | нет | `stdio` | `stdio` или `http` |
| `MCP_HTTP_ADDR` | нет | `:8080` | Адрес в режиме `http`, например `0.0.0.0:8080` |

Логика загрузки: [`internal/nexus/config.go`](../internal/nexus/config.go).

## Соответствие API Nexus

- **REST v1:** `games.json`, `games/{domain}/mods/{id}.json`, `.../files.json` — заголовки `APIKEY`, `Application-Name`, `Application-Version`, `Protocol-Version`.
- **GraphQL v2:** поиск по имени мода (`nexus_search_mods`); в REST v1 отдельного текстового поиска нет.

Соблюдай [Nexus API acceptable use policy](https://help.nexusmods.com/article/114-api-acceptable-use-policy). Не коммить реальные ключи.

## Tools (контракт)

Успешный ответ — текстовый JSON (при возможности с отступами) в содержимом результата tool. Ошибки валидации аргументов или ответа API Nexus возвращаются как результат tool с `isError: true` и текстом сообщения.

### `nexus_games`

Список всех игр (домен, id и метаданные). Нужен для выбора `game_domain` в остальных tools.

| Аргумент MCP | Обязательный |
|--------------|--------------|
| — | — |

**REST:** `GET {NEXUSMODS_REST_BASE}/games.json`

### `nexus_search_mods`

Поиск модов по имени для игры (GraphQL).

| Аргумент MCP | Обязательный | Описание |
|--------------|--------------|----------|
| `game_domain` | да | Домен игры, например `skyrimspecialedition` |
| `query` | да | Произвольная строка поиска (wildcard по имени мода на стороне API) |
| `offset` | нет | Смещение результатов; неотрицательное целое; по умолчанию `0` |
| `count` | нет | Размер страницы; при указании должно быть от 1 до 50; по умолчанию сервер использует 20 |

Некорректные `offset` / `count` дают ошибку tool без запроса к API.

### `nexus_get_mod`

Детали мода (REST).

| Аргумент MCP | Обязательный | Описание |
|--------------|--------------|----------|
| `game_domain` | да | Домен игры |
| `mod_id` | да | Числовой id мода на Nexus |

### `nexus_list_mod_files`

Список файлов мода (архивы, версии, категории).

| Аргумент MCP | Обязательный | Описание |
|--------------|--------------|----------|
| `game_domain` | да | Домен игры |
| `mod_id` | да | Числовой id мода |

Пустой `game_domain` или нечисловой `mod_id` обрабатываются как ошибка tool или API согласно [`internal/nexus/client.go`](../internal/nexus/client.go).

## Ограничения и типичные ошибки

- **`NEXUSMODS_API_KEY is required`** — ключ не передан в окружение процесса (Docker / IDE).
- **401 / отказ API** — неверный или отозванный ключ; проверь ключ в [настройках аккаунта](https://www.nexusmods.com/users/myaccount?tab=api).
- **`nexus API ...`** — HTTP-статус вне 2xx; тело ответа обрезается в сообщении об ошибке.
- **`invalid offset` / `invalid count`** — проверь формат и диапазон для `nexus_search_mods`.
- **Docker без `-i`** — клиент не сможет вести stdio-сессию MCP.
- **HTTP-режим** — клиент должен поддерживать streamable HTTP; публичный порт без защиты не использовать.

## Подключение клиентов

### Cursor

- С ключом в аргументах `docker`: готовый фрагмент — [`cursor-mcp.json.example`](../cursor-mcp.json.example).
- Без ключа в JSON: `--env-file` и **абсолютный** путь к `.env`, который видит Docker — [`cursor-mcp-envfile.example.json`](../cursor-mcp-envfile.example.json).

Пути в WSL: например `/home/<user>/nexusmods-mcp/.env`. На Windows с Docker Desktop проверь тот же `docker run ... --env-file`, что использует Cursor.

### Claude Desktop

Файл конфигурации: `claude_desktop_config.json` (macOS: `~/Library/Application Support/Claude/`; Windows: `%APPDATA%\Claude\`). Пример: [`claude_desktop_config.json.example`](../claude_desktop_config.json.example). Замени путь к `--env-file` или к бинарнику и при необходимости переменные окружения.

Шаблон переменных без секретов в репозитории: [`.env.example`](../.env.example).

## Отладка

- Автоматический smoke по stdio и реальным вызовам Nexus: [`TESTING.md`](../TESTING.md).
- [MCP Inspector](https://github.com/modelcontextprotocol/inspector) — тот же `docker run --rm -i ...`, что для Cursor.
