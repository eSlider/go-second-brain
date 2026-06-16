---
title: AI
---

# Agent instructions — go-second-brain

SDK + demo knowledge base monorepo. Primary language: **Russian** for domain demo docs; system/SDK docs in **English** under `docs/system/`.

## Read first

```
README.md
 └── docs/system/README.md
 └── docs/system/architecture.md
 └── docs/system/configuration.md
 └── docs/project/overview.md      # optional demo corpus
 └── docs/project/glossary.md
```

## Conventions

- German domain terms stay in original spelling (`Verordnung`, `Pflegegrad`, …) — see [glossary](./docs/project/glossary.md)
- Stable IDs: `SUBJ-*`, `UC-*`, `PAIN-*`, `AUTO-*`, `AGENT-*`, `ROAD-*`
- Do not invent domain facts; demo content is synthetic
- Mark unknowns as `TODO: clarify` — do not fabricate

## Code

- Module: `github.com/eSlider/go-second-brain/services`
- Config: `config.yaml.example` + `.env` — see [configuration](./docs/system/configuration.md)
- Ponytail rules: `.cursor/rules/ponytail.mdc`
- ADRs: `docs/system/adr/`

## Forbidden

- Committing `.env`, `config.yaml` with secrets
- Adding Edelweiss or real company PII to the public repo
- Editing non-existent `*.stt.txt` (removed from public repo)
