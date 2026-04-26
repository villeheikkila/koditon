#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
DUMP_SQL_FILE="${SCRIPT_DIR}/schema_dump.sql"
INLINE_CONSTRAINTS_SCRIPT="${SCRIPT_DIR}/schema_inline_constraints.pl"
OUT_FILE="${BACKEND_DIR}/db/schema.sql"

if [[ -n "${DATABASE_URL:-}" ]]; then
  CONN="${DATABASE_URL}"
else
  : "${DB_USER:?DB_USER is required when DATABASE_URL is not set}"
  : "${DB_PASSWORD:?DB_PASSWORD is required when DATABASE_URL is not set}"
  : "${DB_HOST:?DB_HOST is required when DATABASE_URL is not set}"
  : "${DB_PORT:?DB_PORT is required when DATABASE_URL is not set}"
  : "${DB_NAME:?DB_NAME is required when DATABASE_URL is not set}"
  CONN="postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}"
fi

if ! command -v psql >/dev/null 2>&1; then
  echo "ERROR: psql is not installed or not on PATH" >&2
  exit 1
fi

if ! command -v perl >/dev/null 2>&1; then
  echo "ERROR: perl is not installed or not on PATH" >&2
  exit 1
fi

mkdir -p "$(dirname "${OUT_FILE}")"
TMP_FILE="$(mktemp)"
trap 'rm -f "${TMP_FILE}"' EXIT

psql -q -X -A -t -v ON_ERROR_STOP=1 "${CONN}" -f "${DUMP_SQL_FILE}" | perl "${INLINE_CONSTRAINTS_SCRIPT}" > "${TMP_FILE}"
perl -0pi -e 's/\n+\z/\n/' "${TMP_FILE}"
mv "${TMP_FILE}" "${OUT_FILE}"

echo "Wrote ${OUT_FILE}"
