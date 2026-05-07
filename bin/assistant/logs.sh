#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

if [[ ! -f "${LOG_FILE}" ]]; then
  echo "log file not found: ${LOG_FILE}"
  exit 1
fi

tail -n 100 -f "${LOG_FILE}"

