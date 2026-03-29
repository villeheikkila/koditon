# koditon

Full-stack project: Go backend, React web frontend, Swift iOS app, and VPS infrastructure.

## Prerequisites

[mise](https://mise.jdx.dev/) manages tool versions and runs tasks.

```bash
# Install mise (if not already installed)
curl https://mise.run | sh
```

## Setup

```bash
# From project root — installs Go and Node
mise install

# From backend/ — installs backend dev tools (air, golangci-lint, sqlc, tern, etc.)
cd backend && mise install
```

Copy and fill in environment variables:

```bash
cp .env.template .env
cp backend/.env.template backend/.env  # if needed
```

Start local services (Postgres + Redis):

```bash
docker compose up -d
```

## Development

### Backend

```bash
# From project root
mise run backend:dev      # hot reload server
mise run backend:test     # run all tests
mise run backend:lint     # static analysis
mise run backend:build    # production binary → backend/server

# Or from backend/ (shorter, full task list available)
mise run dev
mise run test
```

### Web (run from anywhere)

```bash
mise run web:dev          # Vite dev server
mise run web:build        # type-check + bundle
mise run web:generate     # regenerate API client (Orval)
```

## Database

```bash
# From project root
mise run backend:db:migrate
mise run backend:db:status
NAME=add_users mise run backend:db:new
mise run backend:db:generate

# From backend/ (full set of db tasks available)
mise run db:migrate
QUERY="SELECT 1" mise run db:query
FILE=path/to/file.sql mise run db:query
```

## All tasks

```bash
mise tasks        # list available tasks (context-sensitive by directory)
```
