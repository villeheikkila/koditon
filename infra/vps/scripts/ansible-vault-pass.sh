#!/usr/bin/env bash
set -euo pipefail

account="${ANSIBLE_VAULT_KEYCHAIN_ACCOUNT:-$USER}"
service="${ANSIBLE_VAULT_KEYCHAIN_SERVICE:-koditon-ansible-vault}"

if ! password="$(security find-generic-password -a "$account" -s "$service" -w 2>/dev/null)"; then
  echo "Vault password not found in Keychain (account=$account, service=$service)." >&2
  echo "Add it with: security add-generic-password -U -a \"$account\" -s \"$service\" -w '<PASSWORD>'" >&2
  exit 1
fi

printf '%s' "$password"
