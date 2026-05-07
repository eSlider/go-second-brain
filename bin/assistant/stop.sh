#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

if ! is_running; then
  rm -f "${PID_FILE}"
  echo "assistant is not running"
  exit 0
fi

pid="$(<"${PID_FILE}")"
kill "${pid}" 2>/dev/null || true

for _ in {1..20}; do
  if ! kill -0 "${pid}" 2>/dev/null; then
    rm -f "${PID_FILE}"
    echo "assistant stopped"
    exit 0
  fi
  sleep 0.1
done

kill -9 "${pid}" 2>/dev/null || true
rm -f "${PID_FILE}"
echo "assistant force-stopped"

