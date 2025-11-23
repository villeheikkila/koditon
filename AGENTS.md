# Repository Guidelines

## Project Structure & Module Organization
- `cmd/app/`: service entrypoint; keep `main.go` lean by delegating setup to internal packages.
- `internal/config/`: configuration loading and validation; prefer typed structs over raw maps.
- `internal/server/`: HTTP handlers, routing, and middleware; group files by feature when it grows.
- `db/migrations/`: tern migration files; `db/schema/` holds the canonical schema for sqlc inference; `db/queries/` stores SQL for codegen.
- `internal/db/`: generated sqlc package; treat it as build output and regenerate instead of hand-editing.
- Root files: `go.mod` pins toolchain; `Taskfile.yml` defines common tasks; `.env.example` documents runtime and DB env vars.

## Build, Test, and Development Commands
- `task dev`: run the server with automatic reload on Go source, `go.mod`, or `.env` changes; sets `CGO_ENABLED=0` for reproducible builds.
- `task tools:db`: install tern + sqlc (requires network and Go toolchain).
- `task db:new NAME=add_users`: create a timestamped migration in `db/migrations/`.
- `task db:up`: apply pending migrations using `tern.conf`; `task db:status` shows applied vs pending.
- `task sqlc`: generate Go code in `internal/db` from `db/queries` using `db/schema` for types.
- `go run ./cmd/app`: start the application without file watching; useful for quick checks.
- `go test ./...`: execute unit tests and integration tests; ensure new packages include coverage.
- `go vet ./...`: static analysis; run before proposing changes to catch common pitfalls.

## Coding Style & Naming Conventions
- Use `gofmt` and `goimports` on all Go files; prefer `task gofmt` or an editor save hook.
- Follow Go naming: exported identifiers use CamelCase and doc comments starting with the name; unexported helpers stay scoped and descriptive.
- Keep functions small; prefer dependency injection over globals, especially for config and clients.
- Log with structured fields; avoid panics in request paths—return errors instead.

## Testing Guidelines
- Add `_test.go` files near the code they cover; name tests `TestXxx` and subtests for scenarios.
- Use table-driven tests for handlers and business logic; keep fixtures in `testdata/` alongside the package when needed.
- Aim for meaningful coverage on new code paths, especially around error handling and config parsing.
- Run `go test ./...` locally and ensure tests pass in a clean environment (no reliance on local state).

## Commit & Pull Request Guidelines
- Commits: imperative, concise subject lines (`Add server health check`); group related changes and avoid mixing refactors with feature work.
- PRs: include a short summary, testing notes (`go test ./...`), screenshots for user-facing changes, and linked issues if applicable.
- Describe configuration impacts (new env vars, defaults) and update docs or examples in the same PR.
- Request review early for interface changes; prefer smaller, focused PRs for easier review.
