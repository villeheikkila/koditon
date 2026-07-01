# Backend Agent Guidelines

## Project Overview

Go backend service using pgx, Huma v2 for HTTP API, sqlc for type-safe database queries, and tern for migrations. The application supports dual-mode operation (API server + background consumer) with a reactive task queue system.

## Build, Test, and Development Commands

### Agent Tooling
- Use Chrome DevTools MCP for browser testing and web flow verification.
- Use the PostgreSQL MCP connection for database inspection and SQL queries.

### Development
- `mise run dev`: run server with file watching and auto-reload (from `backend/`)
- `go run ./cmd`: start the application once without watching
- `mise run build`: build production binary

### Testing
- `mise run test`: run all tests
- `go test ./internal/domain/auth`: test specific package
- `go test -v -run TestSignInAnonymous ./internal/domain/auth`: run single test with verbose output
- `mise run test:race`: run tests with race detector
- `mise run test:cover`: run tests with coverage report

### Linting and Formatting
- `mise run lint`: static analysis
- `mise run fmt`: check formatting
- `mise run fix`: autofix formatting and lints

### Database Commands
Run from `backend/` (see root AGENTS.md for details):
- `NAME=description mise run db:new`: create new timestamped migration
- `mise run db:migrate`: apply pending migrations
- `mise run db:status`: show current migration state
- `mise run db:generate`: regenerate sqlc code from queries

## Project Structure

```
backend/
├── cmd/
│   ├── main.go              # Entrypoint; keep minimal, delegate to internal/app
│   └── cli/                 # CLI binary and private implementation
├── internal/
│   ├── app/                 # Application bootstrap, wiring, lifecycle
│   ├── clients/             # External provider API clients
│   ├── db/                  # Generated sqlc storage package (DO NOT EDIT)
│   ├── domain/              # Core feature/domain packages
│   ├── platform/            # Config, logging, queue, schema, utility infrastructure
│   ├── sync/                # Sync engine services, flows, and consumers
│   └── transport/           # HTTP/OpenAPI, OAuth, MCP, web, health transports
├── db/
│   ├── migrations/          # Tern migration files
│   └── queries/             # sqlc query files grouped by feature
└── sqlc.yaml                # sqlc configuration
```

### Package Boundaries
- `cmd/main.go` imports `internal/app`; binary-specific CLI implementation belongs under `cmd/cli/internal`.
- `internal/app` may import all backend subsystems for wiring.
- `internal/clients/*` contains external API wire types and fetch/parse behavior; it must not import `internal/db`, `internal/sync`, or `internal/transport`.
- `internal/sync/*` owns provider synchronization, mapping, and DB upserts; it may import `internal/clients/*`, `internal/db`, and `internal/platform/*`.
- `internal/transport/*` owns request/response adapters and may import domain, sync, db, and platform packages.
- `internal/platform/*` should stay reusable and avoid importing domain, sync, or transport packages.

## Code Style and Conventions

### Imports
- Use goimports grouping: stdlib, external, internal
- Example:
  ```go
  import (
      "context"
      "fmt"
      
      "github.com/jackc/pgx/v5"
      "github.com/jackc/pgx/v5/pgxpool"
      
      "koditon/internal/domain/auth"
      "koditon/internal/platform/config"
  )
  ```

### Naming
- Exported identifiers: `CamelCase` with doc comment starting with the name
- Unexported helpers: descriptive and scoped appropriately
- Database types: use sqlc-generated types with package prefixes (e.g., `authdb.AuthUser`)
- Avoid stuttering: `auth.Service` not `auth.AuthService`

### Types and Nullability
- Use `*T` for nullable primitives (e.g., `*string`, `*int64`)
- Use `pgtype.UUID` for database UUIDs, convert with `pgToUUID()` / `uuidToPg()`
- Use `json.RawMessage` for JSONB columns
- Prefer concrete types over `interface{}`

### Error Handling
- Return errors, don't panic in request paths
- Wrap errors with context: `fmt.Errorf("operation failed: %w", err)`
- Define sentinel errors: `var ErrNotFound = errors.New("not found")`
- Check errors explicitly; use `errors.Is()` and `errors.As()` for sentinel/typed errors
- Always check `defer`red cleanup errors (e.g., `resp.Body.Close()`)

### Logging
- Use structured logging with slog: `logger.Info("msg", "key", value)`
- Add component context: `logger.With("component", "auth")`
- Log levels: Debug for verbose, Info for normal, Warn for recoverable, Error for failures
- Don't log sensitive data (tokens, passwords, PII)

### Database
- Generated code in `internal/db/` is READ ONLY - never hand-edit
- SQL queries go in `db/queries/<feature>/*.sql` with sqlc annotations
- Use transactions for multi-step operations; always `defer tx.Rollback(ctx)`
- Migrations are forward-only; never add down migrations
- Use `FOR UPDATE` when reading rows that will be updated in the same transaction

### Comments
- Minimal and purposeful; avoid obvious comments
- Focus on "why", not "what"
- Exported identifiers require doc comments starting with the name
- Example:
  ```go
  // NewService creates a new authentication service with the given configuration.
  func NewService(cfg Config) (*Service, error) {
  ```

### Formatting
- No blank lines within function bodies or code blocks
- Use single blank lines to separate logical sections
- Let gofmt handle indentation and alignment

### Testing (when adding tests)
- Table-driven tests for multiple scenarios
- Use `t.Parallel()` when tests are independent
- Test file naming: `*_test.go` in the same package
- Use `testdata/` directory for fixtures

## Commit and Pull Request Guidelines

### Commits
- Use conventional commits with a lowercase type prefix such as `feat:`, `fix:`, `refactor:`, or `chore:`
- Keep the summary short, imperative, and lowercase after the prefix
- No trailing punctuation
- Prefer `type: concise summary`, for example `feat: add session refresh token rotation`
- Don't mix unrelated changes in a single commit

### Pull Requests
- Include short summary (2-3 sentences)
- Testing notes: document how to test changes
- Run `go test ./...` and `go vet ./...` before submitting
- Update `.env.template` when adding new environment variables
- Screenshots for user-facing changes

## Architecture Notes

### Authentication System
- JWT-based with access + refresh token pairs
- Refresh token rotation for security (detects reuse)
- Session tracking with device management
- Supports Apple Sign In and anonymous auth

### Task Queue System
- Database-driven reactive architecture
- Application only processes tasks from queue, never initiates syncs
- pg_cron jobs create tasks; workers process them
- Automatic retry with exponential backoff (up to 3 attempts)
- Stuck tasks requeued every 5 minutes

### Configuration
- Environment-based using `github.com/caarlos0/env`
- `.env` files loaded automatically
- All config in `internal/platform/config/config.go`
- Required fields fail fast on startup

### HTTP Framework
- Huma v2 for OpenAPI-first design
- Automatic request/response validation
- Type-safe handlers with input/output structs
- Middleware for auth, logging, CORS

## Common Patterns

### Service Creation
```go
type Service struct {
    logger  *slog.Logger
    pool    *pgxpool.Pool
    queries *db.Queries
}

func NewService(logger *slog.Logger, pool *pgxpool.Pool) *Service {
    return &Service{
        logger:  logger.With("component", "myservice"),
        pool:    pool,
        queries: db.New(pool),
    }
}
```

### Transaction Pattern
```go
tx, err := s.pool.Begin(ctx)
if err != nil {
    return fmt.Errorf("begin transaction: %w", err)
}
defer tx.Rollback(ctx)
qtx := s.queries.WithTx(tx)
// ... do work with qtx ...
if err := tx.Commit(ctx); err != nil {
    return fmt.Errorf("commit transaction: %w", err)
}
```

### UUID Conversion
```go
// pgtype.UUID <-> github.com/google/uuid
func pgToUUID(pg pgtype.UUID) uuid.UUID {
    if !pg.Valid {
        return uuid.Nil
    }
    return uuid.UUID(pg.Bytes)
}

func uuidToPg(u uuid.UUID) pgtype.UUID {
    return pgtype.UUID{Bytes: u, Valid: true}
}
```
