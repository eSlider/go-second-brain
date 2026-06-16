# Configuration

## Files

| File | Committed | Purpose |
|------|-----------|---------|
| `config.yaml.example` | yes | Non-secret defaults (models, URLs, paths, RAG prompt) |
| `config.yaml` | no (gitignored) | Local overrides |
| `.env.example` | yes | Secret placeholders |
| `.env` | no (gitignored) | Secrets and runtime overrides |

## Merge semantics

Implemented in [`services/internal/config/config.go`](../../services/internal/config/config.go) using [go-config](https://github.com/eSlider/go-config):

1. YAML sources merge deepest-first: `config.yaml.example`, then `config.yaml` if present
2. Environment merges on top (`.env` file, then process env)
3. `github.com/mcuadros/go-defaults` fills remaining struct defaults
4. `github.com/go-viper/mapstructure/v2` decodes nested maps into Go structs

Scalar leaves: **last source wins**. Nested maps: **deep merge**.

## Quick start

```bash
cp config.yaml.example config.yaml   # optional
cp .env.example .env
# edit .env — set NEO4J_PASSWORD, MATRIX_PASSWORD, API keys
```

## Key environment variables

Secrets and overrides (see `.env.example`):

- `NEO4J_PASSWORD`, `MATRIX_PASSWORD`, `INWORLD_API_KEY`, `CARTESIA_API_KEY`
- `OLLAMA_URL`, `NEO4J_URI`, `QDRANT_URL`, `QDRANT_COLLECTION`
- `BOT_COMMAND_PREFIX` (default `!brain`)
- `CONFIG_PATH` — optional path to repo root or config file directory

## Docker Compose

Services `kg-ingestor` and `matrix-bot` mount the repo at `/repo` and set `CONFIG_PATH=/repo/config.yaml`. Ensure `config.yaml` or `config.yaml.example` exists at repo root.

## RAG system prompt

Configured under `rag.system_prompt` in YAML. Override in `config.yaml` for your domain tone without code changes.
