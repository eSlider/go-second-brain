# Руководство для контрибьюторов и агентов

Этот документ фиксирует **инженерный слой** поверх базы знаний в `docs/project/`: граф знаний, векторный поиск и Matrix-бот. Основные правила по контенту и терминологии — в [AGENTS.md](AGENTS.md) и [README.md](README.md).

## Go-модуль и расположение кода

- **Модуль:** `git.produktor.io/edelweiss/docs/services` (путь согласован с `git remote`: репозиторий `edelweiss/docs`, код Go — в каталоге [`services/`](services/)).
- **Go:** 1.26 (см. [`services/go.mod`](services/go.mod)).
- **Сборка образов:** [`services/Dockerfile`](services/Dockerfile) — два target: `ingestor` и `bot` (CGO + `libolm` для Matrix E2E).

Структура пакетов (кратко):

- `cmd/ingestor` — обход документации, запись в Neo4j и Qdrant.
- `cmd/bot` — Matrix-бот на [go-matrix-bot](https://github.com/eSlider/go-matrix-bot), RAG через `internal/rag`.
- `internal/docsparse` — разбор Markdown, ID (`SUBJ-*`, `UC-*`, `PAIN-*`, и т.д.), чанки.
- `internal/graph` — Neo4j.
- `internal/vectorstore` — Qdrant (HTTP).
- `internal/embed`, `internal/llm` — Ollama (`/api/embeddings`, `/api/generate`).
- `internal/config`, `internal/httpjson`, `internal/slogx` — конфиг и утилиты.
- `integration/` — интеграционные тесты (тег сборки `integration`).

## Docker Compose

Корневой [`compose.yml`](compose.yml) расширен **профилями**:

- **`docs`** (по умолчанию) — MkDocs, как раньше.
- **`kg`** — Neo4j, Qdrant, одноразовый job `kg-ingestor` (индексация).
- **`bot`** — сервис `matrix-bot` (после индексации).

Ollama **не** входит в compose: ожидается на хосте; из контейнеров доступ через `host.docker.internal:11434` (см. переменные в compose для `kg-ingestor` и `matrix-bot`).

Удобные цели: [`Makefile`](Makefile) — `make kg-up`, `make ingest`, `make bot`, `make test-integration`, `make ci`.

## Переменные окружения

- Шаблон без секретов: [`.env.example`](.env.example).
- Рабочий файл — **`.env`** (в [.gitignore](.gitignore), не коммитить).
- Matrix: `MATRIX_API_URL`, `MATRIX_USER`, `MATRIX_PASSWORD` (или `MATRIX_API_USER` / `MATRIX_API_PASS`).
- Ollama: `OLLAMA_URL`, `GEN_MODEL`, `EMBED_MODEL`.
- Хранилища: `NEO4J_URI`, `NEO4J_USER`, `NEO4J_PASSWORD`, `QDRANT_URL`, `QDRANT_COLLECTION`.
- Бот: `BOT_COMMAND_PREFIX` (например `!edel`), `MATRIX_BOT_DB` (SQLite для crypto state бота).

## Тесты

- Обычный прогон: `cd services && go test ./...` — пакет `integration` без тега не собирается.
- Интеграционные (Docker + при необходимости Ollama на localhost):  
  `go test -tags=integration ./integration/...` или `make test-integration`.
- Полный сценарий RAG в тестах — только при `RUN_RAG_FUNCTIONAL=1` (долго, см. `integration/rag_functional_test.go`).

## Линт и стиль

- Конфиг: [`services/.golangci.yml`](services/.golangci.yml).
- `golangci-lint` должен быть собран **Go ≥ 1.26** (иначе ошибка «targeted Go version (1.26)»). Обновление: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`.
- Форматирование: `make fmt` или `gofmt` в `services/`.

## Что не дублировать

- Факты о Pflegedienst и процессах — только из `docs/project/` и отчётов; не выдумывать детали домена в коде или в ответах бота без опоры на индексированные документы.
- Сырые `docs/*.stt.txt` не редактировать (см. [AGENTS.md](AGENTS.md)).

## Полезные ссылки для агентов

- Обзор возможностей стека в [README.md](README.md) (раздел про knowledge graph и Matrix-бот).
- Neo4j Browser при локальном запуске: `http://localhost:7474` (учётные данные из `NEO4J_*`).
