#!/usr/bin/env bash
# Stop Neo4j, Qdrant, Matrix bot, and optionally Elasticsearch stack (same services as bin/run.sh).
# Does not stop the docs (MkDocs) service unless you pass STOP_DOCS=1.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if ! docker compose version &>/dev/null; then
  echo "error: docker compose not found" >&2
  exit 1
fi

echo "Stopping neo4j, qdrant, matrix-bot ..."
docker compose --profile kg --profile bot stop neo4j qdrant matrix-bot 2>/dev/null || true

if [[ "${STOP_ELASTIC:-1}" == "1" ]]; then
  echo "Stopping elasticsearch, kibana, filebeat (if running) ..."
  docker compose --profile elastic stop elasticsearch kibana filebeat 2>/dev/null || true
fi

if [[ "${STOP_DOCS:-0}" == "1" ]]; then
  echo "Stopping docs ..."
  docker compose stop docs 2>/dev/null || true
fi

echo "Done."
