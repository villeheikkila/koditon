# Repository Guidelines

## Project Structure & Module Organization
- `cmd/app/`: service entrypoint; keep `main.go` small and delegate to internal packages.
- `internal/config/`: configuration loading and validation; use typed structs over maps.
- `internal/server/`: HTTP routing, handlers, and middleware; group by feature as it grows.
- Database: `db/migrations/` (tern files), `db/schema/` (types for sqlc), `db/queries/` (SQL for codegen), generated code lives in `internal/db/` (do not hand-edit).
- Root helpers: `Taskfile.yml` for common tasks, `.env.example` documents env vars, `go.mod` pins toolchain.

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

## Testing Guidelines
- Place `_test.go` beside code; use table-driven tests and subtests for scenarios.
- Name tests `TestXxx`; keep fixtures in `testdata/` near the package when needed.
- Aim for coverage on new code paths, especially error handling and config parsing.
- Run `go test ./...` locally before pushing; keep tests free of local state.

## Commit & Pull Request Guidelines
- Commits: imperative, concise subject (`Add server health check`); avoid mixing unrelated refactors.
- PRs: include short summary, testing notes (`go test ./...`), and screenshots for user-facing changes; link issues when applicable.
- Update docs or `.env.example` when changing configuration; call out new env vars and defaults.
