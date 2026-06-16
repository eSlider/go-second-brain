---
title: AI
---

# Agent instructions — go-second-brain

Public SDK monorepo: Go services + optional local Markdown corpus. System docs in **English** under `docs/system/`.

## Read first

```
README.md
 └── docs/system/README.md
 └── docs/system/architecture.md
 └── docs/system/configuration.md
 └── docs/system/corpus-format.md
 └── examples/corpus/              # synthetic fixture only (public)
```

## Conventions

- Stable IDs in corpora: `SUBJ-*`, `UC-*`, `PAIN-*`, `AUTO-*`, `AGENT-*`, `ROAD-*` — see [corpus-format](./docs/system/corpus-format.md)
- Do not commit proprietary domain knowledge to this public repository
- Mark unknowns as `TODO: clarify` — do not fabricate

## Code

- Module: `github.com/eSlider/go-second-brain/services`
- Config: `config.yaml.example` + `.env` — see [configuration](./docs/system/configuration.md)
- Ponytail rules: `.cursor/rules/ponytail.mdc`
- ADRs: `docs/system/adr/`

## Forbidden

- Committing `.env`, `config.yaml` with secrets
- Adding real company PII or client-specific business logic to the public repo
- Publishing interview transcripts or operational domain documentation here
