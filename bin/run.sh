#!/usr/bin/env bash
# Start Neo4j, Qdrant, and Matrix bot (compose profiles kg + bot).
# Optional: INCLUDE_ELASTIC=1 to also start Elasticsearch + Kibana + Filebeat.
# Requires: Docker, compose file at repo root, .env for bot credentials (copy from .env.example).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if ! docker compose version &>/dev/null; then
  echo "error: docker compose not found" >&2
  exit 1
fi

if [[ ! -f .env ]]; then
  echo "warning: .env missing; copy .env.example to .env and set Matrix / secrets." >&2
fi

profiles=(--profile kg --profile bot)
services=(neo4j qdrant matrix-bot)

if [[ "${INCLUDE_ELASTIC:-0}" == "1" ]]; then
  profiles+=(--profile elastic)
  services+=(elasticsearch kibana filebeat)
fi

echo "Starting: ${services[*]}"
docker compose "${profiles[@]}" up -d "${services[@]}"

echo ""
echo "Compose status (all listed services):"
docker compose "${profiles[@]}" ps "${services[@]}"

echo ""
echo "Done. Bot uses Ollama on host (OLLAMA_URL in .env). Ensure Ollama is running."
echo "This script exits after starting detached containers; services keep running."
if [[ "${INCLUDE_ELASTIC:-0}" == "1" ]]; then
  echo "Kibana: http://127.0.0.1:${KIBANA_PORT:-5601}"
fi
