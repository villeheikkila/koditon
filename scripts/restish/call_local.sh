#!/usr/bin/env bash
set -euo pipefail

API_NAME="${RESTISH_API_NAME:-koditon}"
API_BASE="${API_PUBLIC_BASE_URL:-http://localhost:8080}"
TOKEN="${KODITON_RESTISH_ACCESS_TOKEN:-}"

if [ -z "${TOKEN}" ]; then
  TOKEN="$(scripts/restish/dev_access_token.sh)"
fi

if [ "$#" -eq 0 ]; then
  exec restish "${API_NAME}" --rsh-server "${API_BASE}" -H "authorization:Bearer ${TOKEN}"
fi

target="$1"
shift

case "${target}" in
  http://*|https://*)
    exec restish "${target}" -H "authorization:Bearer ${TOKEN}" "$@"
    ;;
  /*)
    exec restish "${API_BASE%/}${target}" -H "authorization:Bearer ${TOKEN}" "$@"
    ;;
  *)
    exec restish "${API_NAME}" --rsh-server "${API_BASE}" -H "authorization:Bearer ${TOKEN}" "${target}" "$@"
    ;;
esac
