# infra/vps Ansible

## Usage

```bash
cp group_vars/vps.secrets.yml.example group_vars/vps.secrets.yml
ansible-vault encrypt group_vars/vps.secrets.yml
ansible-playbook -i inventory.ini playbook.yml --vault-password-file ../scripts/ansible-vault-pass.sh
```

Or from `infra/vps`:

```bash
task ansible:apply
```

Run a post-reboot/deploy healthcheck:

```bash
task ansible:compose:healthcheck
```

## Notes

- `inventory.ini` targets `koditon.bytesized.solutions` so Ansible follows Pulumi-managed DNS instead of a pinned VPS IP.
- `ghcr_login=true` requires non-empty `ghcr_username` and `ghcr_token`; Ansible fails fast with a clear assertion if either is missing.
- `infra_env` in `group_vars/vps.secrets.yml` is rendered to `/srv/koditon/infra/.env.prod`.
- Caddy runs only in compose. TLS is automatic via Let's Encrypt.
- Unattended upgrades are configured for daily security updates with automatic reboot at `04:00` UTC.
- `deploy_password_hash` is optional and updates the `deploy` user's Unix password. Generate hash with `openssl passwd -6`.
- Vault password lookup defaults to Keychain service `koditon-ansible-vault` and account `$USER`.
  Override with `ANSIBLE_VAULT_KEYCHAIN_SERVICE` and `ANSIBLE_VAULT_KEYCHAIN_ACCOUNT`.
