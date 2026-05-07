#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

if is_running; then
  pid="$(<"${PID_FILE}")"
  echo "assistant running (pid ${pid})"
  echo "log: ${LOG_FILE}"
  exit 0
fi

echo "assistant stopped"
exit 1

