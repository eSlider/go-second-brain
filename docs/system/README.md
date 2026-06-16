# System documentation index

Read this first when working on the **go-second-brain** SDK (Go services + optional local Markdown corpus).

## Order of reading

1. [architecture.md](./architecture.md) — packages, data flow, public API surface
2. [configuration.md](./configuration.md) — `config.yaml` + `.env` merge
3. [corpus-format.md](./corpus-format.md) — Markdown ID conventions and layout
4. [SECURITY.md](./SECURITY.md) — secrets policy and credential rotation
5. [adr/README.md](./adr/README.md) — architecture decision records

## Agent rules

- Apply **Ponytail** (lazy senior dev): `.cursor/rules/ponytail.mdc`
- Do not commit proprietary domain knowledge to the public repo; use a private corpus path via `docs.root`
- Synthetic fixture only: [`examples/corpus`](../../examples/corpus/)

## Commands (from repo root)

```bash
cp config.yaml.example config.yaml   # optional local overrides
cp .env.example .env                 # secrets
make kg-up ingest bot
cd services && go test ./...
```
