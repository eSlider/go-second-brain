# Sample corpus (synthetic fixture)

Minimal Markdown tree for testing **docsparse**, **ingestor**, and RAG integration tests.

This is **not** real business data. Replace `examples/corpus` with your own knowledge base locally.

## Layout

| Path | Purpose |
|------|---------|
| `glossary.md` | Term nodes (`TERM:*`) |
| `cases/UC-*.md` | Use-case nodes |
| `subjects/*.md` | Subject nodes (`SUBJ-*`) |
| `optimization/pain-points.md` | Pain nodes (`PAIN-*`) |

See [corpus format](../../docs/system/corpus-format.md) for ID conventions.
