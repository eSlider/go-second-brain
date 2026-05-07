#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

if is_running; then
  echo "assistant already running (pid $(<"${PID_FILE}"))"
  exit 0
fi

load_assistant_env

cd "${ROOT_DIR}/services"
nohup go run ./cmd/assistant >>"${LOG_FILE}" 2>&1 &
pid=$!
echo "${pid}" >"${PID_FILE}"
echo "assistant started (pid ${pid})"
echo "logs: ${LOG_FILE}"

