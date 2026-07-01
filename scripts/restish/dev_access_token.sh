#!/usr/bin/env bash
set -euo pipefail

if [ -n "${KODITON_RESTISH_ACCESS_TOKEN:-}" ]; then
  printf '%s\n' "${KODITON_RESTISH_ACCESS_TOKEN}"
  exit 0
fi

API_BASE="${API_PUBLIC_BASE_URL:-http://localhost:8080}"
WEB_BASE="${WEB_BASE_URL:-http://localhost:5173}"
EMAIL="${KODITON_RESTISH_EMAIL:-agent@koditon.local}"
DEVICE_ID="${KODITON_RESTISH_DEVICE_ID:-}"
export EMAIL

if [ -z "${DEVICE_ID}" ]; then
  DEVICE_ID="$(python3 - <<'PY'
import uuid
print(uuid.uuid4())
PY
)"
fi

payload="$(python3 - <<'PY'
import json
import os
print(json.dumps({"email": os.environ["EMAIL"]}))
PY
)"

response="$(
  curl -fsS \
    -H "Content-Type: application/json" \
    -H "Origin: ${WEB_BASE}" \
    -H "X-Device-ID: ${DEVICE_ID}" \
    --data "${payload}" \
    "${API_BASE%/}/auth/dev/web"
)"

export RESPONSE="${response}"
python3 - <<'PY'
import json
import os
import sys
data = json.loads(os.environ["RESPONSE"])
token = data.get("access_token", "")
if not token:
    raise SystemExit("auth response did not include access_token")
print(token)
PY
