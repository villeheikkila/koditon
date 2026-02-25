# infra/vps

Canonical VPS provisioning and configuration package for the production stack in
`infra/docker-compose.prod.yml`.

## What it does

- Provisions Hetzner VPS + DNS with Pulumi
- Configures host baseline and Docker with Ansible
- Deploys the existing production compose stack under `/srv/koditon/infra`
- Runs Caddy inside `docker-compose.prod.yml` as the public ingress on ports `80/443`

## Pulumi defaults

- domain: `koditon.bytesized.solutions`
- server: `cpx22` in `hel1`, name `koditon-vps`
- SSH user: `deploy`

SSH public key path resolution order:

1. `SSH_PUBLIC_KEY_PATH` env var
2. `~/.ssh/id_ed25519.pub`
3. `~/.ssh/id_rsa.pub`

## Quick start

1. Provision server and DNS:

```bash
cd infra/vps
task pulumi:preview
task pulumi:up
```

2. Configure server and deploy compose:

```bash
cd infra/vps
cp ansible/group_vars/vps.secrets.yml.example ansible/group_vars/vps.secrets.yml
ansible-vault encrypt ansible/group_vars/vps.secrets.yml

# fill ansible/group_vars/vps.yml and encrypted vps.secrets.yml

task ansible:apply
```

## Caddy routing

- `api.koditon.bytesized.solutions` -> `backend:8080`

## Notes

- Caddy runs in compose and terminates TLS directly on `80/443`.
- Internal service ports are private to the compose network and are not published on the host.
- `infra/docker-compose.prod.yml` remains the runtime source of truth.
- Optional: set `deploy_password_hash` in `ansible/group_vars/vps.secrets.yml` to
  rotate the `deploy` user's Unix password (hash via `openssl passwd -6`).
