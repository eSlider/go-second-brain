# Contributor guide

Engineering layer for **go-second-brain**: knowledge graph, vector search, Matrix bot, voice assistant. Content rules: [AGENTS.md](AGENTS.md). System architecture: [docs/system/](docs/system/).

## Go module

- **Module:** `github.com/eSlider/go-second-brain/services`
- **Go:** 1.26+ ([`services/go.mod`](services/go.mod))
- **Dockerfile:** [`services/Dockerfile`](services/Dockerfile) — targets `ingestor`, `bot`

### Packages

- `cmd/ingestor` — walk `docs/project`, write Neo4j + Qdrant
- `cmd/bot` — Matrix bot with RAG ([go-matrix-bot](https://github.com/eSlider/go-matrix-bot))
- `cmd/assistant` — STT/TTS CLI loop
- `pkg/*` — public SDK surface (see ADR-0003)
- `internal/*` — config, docsparse, rag, graph

## Configuration

1. `cp config.yaml.example config.yaml` (optional overrides)
2. `cp .env.example .env` — set secrets
3. Load order documented in [docs/system/configuration.md](docs/system/configuration.md)

## Docker Compose

Profiles in [`compose.yml`](compose.yml):

- `docs` — MkDocs (default)
- `kg` — Neo4j, Qdrant, `kg-ingestor`
- `bot` — `matrix-bot`

`make kg-up`, `make ingest`, `make bot`, `make test-integration`

## Tests

From repo root:

```bash
make fmt lint
cd services && go test ./...
make test-integration
```

RAG functional (slow, needs Ollama): `make test-rag-e2e`

## Lint

[`services/.golangci.yml`](services/.golangci.yml) — requires Go ≥ 1.26 for golangci-lint.

## Domain content

Demo corpus only (`docs/project/`). Do not add real company PII. SDK docs belong in `docs/system/`.
