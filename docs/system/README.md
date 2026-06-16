# System documentation index

Read this first when working on the **go-second-brain** SDK (Go services + demo knowledge corpus).

## Order of reading

1. [architecture.md](./architecture.md) — packages, data flow, public API surface
2. [configuration.md](./configuration.md) — `config.yaml` + `.env` merge
3. [SECURITY.md](./SECURITY.md) — secrets policy and credential rotation
4. [adr/README.md](./adr/README.md) — architecture decision records
5. [../project/overview.md](../project/overview.md) — optional demo domain corpus (DemoCare)

## Agent rules

- Apply **Ponytail** (lazy senior dev): `.cursor/rules/ponytail.mdc`
- Domain facts live only in `docs/project/` demo files — do not invent Pflegedienst details in code
- Raw `*.stt.txt` are not used in the public repo; structured content is in `docs/reports/` and `docs/project/`

## Commands (from repo root)

```bash
cp config.yaml.example config.yaml   # optional local overrides
cp .env.example .env                 # secrets
make kg-up ingest bot
cd services && go test ./...
```
