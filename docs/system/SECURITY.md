# Security

## Credential rotation (2026-06)

Historical private forks may have committed a `.env` with real credentials.
Those entries were purged with `git filter-repo` before the public release, but **removal from git does not revoke tokens**.

Rotate any credentials that were ever stored in a committed `.env`:

- Social / portal auth tokens
- Self-hosted service passwords
- API keys for third-party STT/TTS providers

## Local development

- Copy `.env.example` → `.env` and `config.yaml.example` → `config.yaml`.
- Never commit `.env` or `config.yaml` with secrets (both are in `.gitignore`).
- Run `gitleaks detect --source .` before pushing.
