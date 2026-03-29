# Nexus Mods MCP

Сервер [Model Context Protocol](https://modelcontextprotocol.io/) для [Nexus Mods](https://www.nexusmods.com/): REST v1 (игры, моды, файлы, changelog, ленты, отслеживаемые моды, лимиты) и GraphQL v2 (поиск с фильтрами, `mod.modRequirements`, расширенная карточка `mod`). Реализация на Go: [`github.com/modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk) v1.2.0 (см. `go.mod`). Имя и версия сервера в протоколе: `nexusmods-mcp` / `0.1.0` — [`cmd/server/main.go`](../cmd/server/main.go).

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
| `NEXUSMODS_GRAPHQL_URL` | нет | `https://api.nexusmods.com/v2/graphql` | GraphQL: поиск, `mod`, `mod.modRequirements` |
| `NEXUSMODS_CACHE_TTL` | нет | `24h` | In-memory кеш ответов Nexus (`time.ParseDuration`: `24h`, `30m`, …). Пусто = 24h. `0`, `off`, `false`, `disable` — без кеша |
| `MCP_TRANSPORT` | нет | `stdio` | `stdio` или `http` |
| `MCP_HTTP_ADDR` | нет | `:8080` | Адрес в режиме `http`, например `0.0.0.0:8080` |

Логика загрузки: [`internal/nexus/config.go`](../internal/nexus/config.go).

## Кеширование ответов API

По умолчанию ответы Nexus кешируются **в памяти процесса** с TTL из `NEXUSMODS_CACHE_TTL` (по умолчанию **24 часа**). Это снижает число запросов к API и нагрузку на лимиты; данные могут отставать от сайта до истечения TTL.

Реализация: [`internal/nexus/apicache.go`](../internal/nexus/apicache.go), обвязка в [`internal/nexus/client.go`](../internal/nexus/client.go).

| Кешируется (TTL) | Не кешируется |
|------------------|---------------|
| `Games` (`nexus_games`), карточка мода, список файлов, один файл, changelog, игра/категории, ленты (`latest_*`, `trending`, `recently_updated` с `period`), GraphQL `mod.modRequirements` (`nexus_get_mod_requirements`) | Поиск `nexus_search_mods`, `nexus_get_mod_graphql` (поля viewer\*), `nexus_get_tracked_mods`, `nexus_get_rate_limits` |

Отключить кеш: `NEXUSMODS_CACHE_TTL=0` или `off` / `false` / `disable` (см. [`config.go`](../internal/nexus/config.go)). У нескольких процессов (несколько инстансов HTTP-транспорта) у каждого свой кеш.

Автотесты без сети: `go test ./internal/nexus/ -count=1` (в т.ч. [`client_cache_test.go`](../internal/nexus/client_cache_test.go)).

## Соответствие API Nexus

- **REST v1** (по `game_domain` в пути, как в [официальном клиенте](https://github.com/Nexus-Mods/node-nexus-api)): `games.json`, `games/{domain}.json`, `games/{domain}/mods/...`, `user/tracked_mods.json`, ленты `latest_updated` / `latest_added` / `trending` / `updated?period=`, changelog, один файл по id. Заголовки: `APIKEY`, `Application-Name`, `Application-Version`, `Protocol-Version`.
- **GraphQL v2:** `mods` (фильтр `ModsFilter`), `mod` (в т.ч. viewer\*-поля), `mod.modRequirements`. Числовой `gameId` для `mod(...)` сервер подставляет из `games.json` по `game_domain`.

Соблюдай [Nexus API acceptable use policy](https://help.nexusmods.com/article/114-api-acceptable-use-policy). Не коммить реальные ключи.

## Вне скоупа публичного API

Вкладки **POSTS / BUGS** на странице мода: у типа `Mod` в GraphQL нет ссылки на тред комментариев; REST-карточка мода не отдаёт URL форума. Читать эти ветки через этот MCP нельзя — только запрос фичи у Nexus или просмотр сайта вручную.

## Tools (контракт)

Успешный ответ — текстовый JSON (при возможности с отступами) в содержимом результата tool. Ошибки валидации аргументов или ответа API Nexus возвращаются как результат tool с `isError: true` и текстом сообщения.

### `nexus_games`

Список всех игр (домен, id и метаданные). Нужен для выбора `game_domain` в остальных tools.

| Аргумент MCP | Обязательный |
|--------------|--------------|
| — | — |

**REST:** `GET {NEXUSMODS_REST_BASE}/games.json`

### `nexus_search_mods`

Поиск модов для игры (GraphQL `mods`). Нужен **хотя бы один** из фильтров: `query` (wildcard по **stemmed** имени, поле API `nameStemmed` — токены, пунктуация не учитывается), `author` (точное совпадение), `category_name` (точное совпадение).

| Аргумент MCP | Обязательный | Описание |
|--------------|--------------|----------|
| `game_domain` | да | Домен игры |
| `query` | нет* | Wildcard по stemmed-имени (`nameStemmed` в ModsFilter) |
| `author` | нет* | Точное совпадение автора |
| `category_name` | нет* | Точное совпадение названия категории |
| `offset` | нет | Смещение; по умолчанию `0` |
| `count` | нет | 1–50; по умолчанию `20` |

\* Один или несколько из `query` / `author` / `category_name` должны быть непустыми.

### `nexus_get_mod`

Детали мода (REST).

| Аргумент MCP | Обязательный | Описание |
|--------------|--------------|----------|
| `game_domain` | да | Домен игры |
| `mod_id` | да | Числовой id мода |

### `nexus_get_mod_graphql`

Те же `game_domain` + `mod_id`, но ответ GraphQL `mod`: описание, даты, `viewerUpdateAvailable`, `viewerTracked`, и т.д.

### `nexus_list_mod_files`

Список файлов мода (REST).

| Аргумент MCP | Обязательный | Описание |
|--------------|--------------|----------|
| `game_domain` | да | Домен игры |
| `mod_id` | да | Числовой id мода |

### `nexus_get_mod_file`

Один файл по `file_id` (REST), без полного списка.

| Аргумент MCP | Обязательный | Описание |
|--------------|--------------|----------|
| `game_domain` | да | Домен игры |
| `mod_id` | да | Числовой id мода |
| `file_id` | да | id файла (из списка файлов) |

### `nexus_get_mod_changelog`

Changelog мода (REST).

| Аргумент MCP | Обязательный |
|--------------|--------------|
| `game_domain` | да |
| `mod_id` | да |

### `nexus_get_mod_requirements`

Зависимости мода (GraphQL `mod.modRequirements`). См. прежний контракт: `requirements_*`, `dependents_*` для пагинации.

### `nexus_get_game`

Одна игра по домену (REST), включая дерево категорий в поле `categories`.

| Аргумент MCP | Обязательный |
|--------------|--------------|
| `game_domain` | да |

### `nexus_game_categories`

Только `{"categories":[...]}` из `nexus_get_game` (удобнее для агентов).

| Аргумент MCP | Обязательный |
|--------------|--------------|
| `game_domain` | да |

### `nexus_mods_latest_updated` / `nexus_mods_latest_added` / `nexus_mods_trending`

Ленты модов по игре (REST).

| Аргумент MCP | Обязательный |
|--------------|--------------|
| `game_domain` | да |

### `nexus_mods_recently_updated`

Кэшированный список обновлений за период (REST).

| Аргумент MCP | Обязательный | Описание |
|--------------|--------------|----------|
| `game_domain` | да | Домен игры |
| `period` | да | Одно из: `1d`, `1w`, `1m` |

### `nexus_get_tracked_mods`

Моды, отслеживаемые аккаунтом владельца API-ключа (REST, только чтение).

### `nexus_get_rate_limits`

Лёгкий `GET .../games.json` и возврат заголовков `x-rl-*` в JSON (для отладки квот).

## Ограничения и типичные ошибки

- **`NEXUSMODS_API_KEY is required`** — ключ не передан в окружение процесса (Docker / IDE).
- **401 / отказ API** — неверный или отозванный ключ; проверь ключ в [настройках аккаунта](https://www.nexusmods.com/users/myaccount?tab=api).
- **`nexus API ...`** — HTTP-статус вне 2xx; тело ответа обрезается в сообщении об ошибке.
- **`invalid offset` / `invalid count`** — для `nexus_search_mods` или `requirements_*` / `dependents_*` в `nexus_get_mod_requirements`.
- **`provide query and/or author and/or category_name`** — пустой поиск в `nexus_search_mods`.
- **`period must be 1d, 1w, or 1m`** — `nexus_mods_recently_updated`.
- **GraphQL `Mod not found` в JSON** — неверная пара игра/мод или `mod_id`; для `mod(...)` числовой `gameId` подставляется по `game_domain`.
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
