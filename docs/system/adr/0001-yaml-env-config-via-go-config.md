# ADR-0001: YAML + env config via go-config

- Status: Accepted
- Context: Configuration was env-only with duplicated defaults in `.env.example`. Public SDK needs committed defaults without secrets.
- Decision: Use `github.com/eSlider/go-config` — load `config.yaml.example` and optional `config.yaml`, then merge `.env` and process environment. YAML is the preferred format for non-secret defaults.
- Consequences: Two-file workflow for operators (`config.yaml` + `.env`). Compose mounts repo root so containers share the same merge order.
