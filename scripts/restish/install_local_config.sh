#!/usr/bin/env bash
set -euo pipefail

API_NAME="${RESTISH_API_NAME:-koditon}"
API_BASE="${API_PUBLIC_BASE_URL:-http://localhost:8080}"
CONFIG_DIR="${RESTISH_CONFIG_DIR:-${HOME}/Library/Application Support/restish}"
APIS_FILE="${CONFIG_DIR}/apis.json"
export API_NAME API_BASE APIS_FILE

mkdir -p "${CONFIG_DIR}"

python3 - <<'PY'
import json
import os
from pathlib import Path

path = Path(os.environ["APIS_FILE"])
api_name = os.environ["API_NAME"]
api_base = os.environ["API_BASE"]

if path.exists():
    with path.open("r", encoding="utf-8") as f:
        data = json.load(f)
else:
    data = {"$schema": "https://rest.sh/schemas/apis.json"}

api = data.get(api_name, {})
profiles = api.get("profiles", {})
profiles["local"] = {
    "auth": {
        "name": "bearer",
        "params": {
            "token": "env:KODITON_RESTISH_ACCESS_TOKEN",
        },
    },
}
api["base"] = api_base
api["profiles"] = profiles
data[api_name] = api

tmp_path = path.with_suffix(path.suffix + ".tmp")
with tmp_path.open("w", encoding="utf-8") as f:
    json.dump(data, f, indent=2)
    f.write("\n")
tmp_path.replace(path)
PY

restish api show "${API_NAME}"
