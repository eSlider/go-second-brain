# ADR-0003: pkg/ as public SDK surface

- Status: Accepted
- Context: Repository ships both demo app binaries (`cmd/`) and reusable clients (`pkg/`).
- Decision: External consumers import only `github.com/eSlider/go-second-brain/services/pkg/...`. `internal/*` holds ingest, RAG orchestration, and assistant runtime — not a stability promise.
- Consequences: Breaking changes in `internal/` are allowed. Semver tags apply to module `services` as a whole; document `pkg/` APIs in README.
