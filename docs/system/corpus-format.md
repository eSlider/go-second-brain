# Markdown corpus format

go-second-brain indexes a directory of `.md` files via **docsparse**. The format is domain-agnostic: you bring your own content; the repo ships only a tiny synthetic fixture under [`examples/corpus`](../../examples/corpus/).

## Stable IDs

| Prefix | Kind | Example |
|--------|------|---------|
| `SUBJ-*` | Subject (actor) | `SUBJ-ACTOR` |
| `UC-NN` | Use case | `UC-01` |
| `PAIN-NN` | Pain point | `PAIN-01` |
| `AUTO-NN` | Automation idea | `AUTO-01` |
| `AGENT-NN` | Agent idea | `AGENT-01` |
| `ROAD-NN` | Roadmap item | `ROAD-01` |

Inline references (`**SUBJ-ACTOR**`, links to `UC-01-*.md`) become Neo4j edges (`REFS`, `INVOLVES`, `AFFECTS`, …).

## Recommended layout

```
corpus/
├── glossary.md              # TERM:* from "- **term** — desc {: #anchor}"
├── cases/UC-NN-slug.md
├── subjects/subj-*.md       # title must contain SUBJ-*
├── processes/*.md           # PROC-* from filename
└── optimization/
    ├── pain-points.md       # **PAIN-NN** list items
    ├── automation-opportunities.md
    ├── ai-agent-ideas.md
    └── roadmap.md
```

## Configuration

Set corpus root via YAML or env:

```yaml
docs:
  root: examples/corpus
```

```bash
export DOCS_ROOT=examples/corpus
```

## Security

Do **not** commit proprietary or personal data to the public SDK repository. Keep production corpora in a private repo or local path and point `docs.root` at it.
