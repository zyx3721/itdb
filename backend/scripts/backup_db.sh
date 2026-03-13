#!/usr/bin/env bash
set -euo pipefail

SOURCE="${1:-../data/itdb.db}"
TARGET_DIR="${2:-../data/backups}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOURCE_PATH="$(cd "${SCRIPT_DIR}" && cd "$(dirname "${SOURCE}")" && pwd)/$(basename "${SOURCE}")"
TARGET_PATH="${SCRIPT_DIR}/${TARGET_DIR}"

mkdir -p "${TARGET_PATH}"
STAMP="$(date +%Y%m%d-%H%M%S)"
DEST="${TARGET_PATH}/itdb-${STAMP}.db"

cp "${SOURCE_PATH}" "${DEST}"
echo "Backup created: ${DEST}"
