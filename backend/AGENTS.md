# Repository Guidelines

## Project Structure & Module Organization
- `cmd/app/`: service entrypoint; keep `main.go` small and delegate to internal packages.
- `internal/config/`: configuration loading and validation; use typed structs over maps.
- `internal/server/`: HTTP routing, handlers, and middleware; group by feature as it grows.
- Database: `db/migrations/` (tern files), `db/schema/` (types for sqlc), `db/queries/` (SQL for codegen), generated code lives in `internal/db/` (do not hand-edit).
- Root helpers: `Taskfile.yml` for common tasks, `.env.template` documents env vars, `go.mod` pins toolchain.

## Build, Test, and Development Commands
- `task dev`: run the server with reload on Go source, `go.mod`, or `.env` changes (`CGO_ENABLED=0`).
- `go run ./cmd/app`: start the app once without file watching.
- `go test ./...`: run unit and integration tests.
- `go vet ./...`: static analysis; run before proposing changes.
- Database tooling: `task tools:db` installs tern + sqlc; `task db:new NAME=...` creates timestamped migration; `task db:up` applies pending migrations; `task db:status` shows state; `task db:force VERSION=...` pins migration state; `task sqlc` regenerates `internal/db`.

## Coding Style & Naming Conventions
- Go fmt first: `task gofmt` or editor save hook; use `goimports` style imports.
- Follow Go naming: exported identifiers CamelCase with doc comments starting with the name; unexported helpers stay scoped and descriptive.
- Prefer dependency injection over globals; log with structured fields; avoid panics in request paths—return errors instead.
- **Minimal code comments**: Keep comments minimal and purposeful. Avoid obvious comments that restate what the code does.
- **No spaces inside code blocks**: Do not add blank lines within function bodies or code blocks for "readability".
- **When comments are needed**: Split code sections with short, descriptive comments that explain *why*, not *what*.

## Frontdoor Task Sync System
The Frontdoor scraper uses a fully database-driven, reactive task queue system to sync ads and buildings daily.

### Architecture (Database-Driven)
- **Reactive Application**: The application is purely reactive - it only processes tasks from the queue, never initiates syncs on startup.
- **Database-Driven Workflow**: All sync operations are triggered by pg_cron jobs that create tasks in the queue.
- **Automatic Retry**: Failed tasks are retried up to 3 times; stuck tasks are requeued every 5 minutes by pg_cron.

### Key Components
- **Task Queue** (`internal/taskqueue/`): PGMQ-based distributed queue with task state tracking.
- **Worker Pool** (`internal/taskqueue/worker.go`): Concurrent task processors with configurable parallelism.

## Commit & Pull Request Guidelines
- Commits: imperative, concise subject (`Add server health check`); avoid mixing unrelated refactors.
- PRs: include short summary, testing notes (`go test ./...`), and screenshots for user-facing changes; link issues when applicable.
- Update docs or `.env.template` when changing configuration; call out new env vars and defaults.
