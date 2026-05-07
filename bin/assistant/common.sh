#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ENV_FILE="${ROOT_DIR}/.env"
RUNTIME_DIR="${ROOT_DIR}/services/var/run"
PID_FILE="${RUNTIME_DIR}/assistant.pid"
LOG_FILE="${ROOT_DIR}/services/var/logs/assistant.out.log"

required_vars=(
  INWORLD_API_KEY
  CARTESIA_API_KEY
  CARTESIA_VOICE_ID
)

optional_vars=(
  INWORLD_STT_MODEL
  CARTESIA_MODEL_ID
  ASSISTANT_AUDIO_SAMPLE_RATE
  ASSISTANT_CHUNK_MS
  ASSISTANT_TTS_DIR
  ASSISTANT_AUDIO_REC_DIR
  ASSISTANT_STT_DIR
  ASSISTANT_PERF_LOG
)

load_from_env_file() {
  local key="$1"
  local line value
  line="$(rg -m1 "^${key}=" "${ENV_FILE}" || true)"
  if [[ -z "${line}" ]]; then
    return 0
  fi
  value="${line#*=}"
  value="${value%\"}"
  value="${value#\"}"
  value="${value%\'}"
  value="${value#\'}"
  export "${key}=${value}"
}

load_assistant_env() {
  mkdir -p "${RUNTIME_DIR}" "$(dirname "${LOG_FILE}")"
  if [[ -f "${ENV_FILE}" ]]; then
    for k in "${required_vars[@]}" "${optional_vars[@]}"; do
      if [[ -z "${!k:-}" ]]; then
        load_from_env_file "${k}"
      fi
    done
  fi

  : "${INWORLD_STT_MODEL:=base}"
  : "${CARTESIA_MODEL_ID:=sonic-3.5}"
  : "${ASSISTANT_AUDIO_SAMPLE_RATE:=16000}"
  : "${ASSISTANT_CHUNK_MS:=100}"
  : "${ASSISTANT_TTS_DIR:=var/tss}"
  : "${ASSISTANT_AUDIO_REC_DIR:=var/audio-rec}"
  : "${ASSISTANT_STT_DIR:=var/stt}"
  : "${ASSISTANT_PERF_LOG:=var/logs/performance.jsonl}"

  for k in "${required_vars[@]}"; do
    if [[ -z "${!k:-}" ]]; then
      echo "assistant: missing required env var: ${k}" >&2
      echo "Set it in ${ENV_FILE} or export it before run." >&2
      exit 1
    fi
  done
}

is_running() {
  if [[ ! -f "${PID_FILE}" ]]; then
    return 1
  fi
  local pid
  pid="$(<"${PID_FILE}")"
  [[ -n "${pid}" ]] && kill -0 "${pid}" 2>/dev/null
}

