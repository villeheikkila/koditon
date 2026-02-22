# Backend Agent Guidelines

## Project Overview

Go backend service using pgx, Huma v2 for HTTP API, sqlc for type-safe database queries, and tern for migrations. The application supports dual-mode operation (API server + background consumer) with a reactive task queue system.

## Build, Test, and Development Commands

### Development
- `task backend:dev`: run server with file watching and auto-reload (uses reflex)
- `go run ./cmd`: start the application once without watching
- `go build -o server ./cmd`: build production binary

### Testing
- `go test ./...`: run all tests
- `go test ./internal/auth`: test specific package
- `go test -v -run TestSignInAnonymous ./internal/auth`: run single test with verbose output
- `go test -race ./...`: run tests with race detector
- `go test -cover ./...`: run tests with coverage report
- `go test -coverprofile=coverage.out ./...`: generate coverage profile

### Linting and Formatting
- `go vet ./...`: static analysis (run before committing)
- `go fmt ./...`: format all Go code (use goimports style)
- `goimports -w .`: organize imports and format (preferred over go fmt)

### Database Commands
Use task commands from the root directory (see root AGENTS.md for details):
- `task db:new NAME=description`: create new timestamped migration
- `task db:migrate`: apply pending migrations
- `task db:status`: show current migration state
- `task db:generate`: regenerate sqlc code from queries
- `task db:query QUERY="SELECT ..."`: run SQL query
- `task db:reset`: drop, recreate, and migrate database (destructive)

## Project Structure

```
backend/
├── cmd/
│   └── main.go              # Entrypoint; keep minimal, delegate to internal/
├── internal/
│   ├── auth/                # Authentication and authorization
│   │   ├── db/              # Generated sqlc code (DO NOT EDIT)
│   │   ├── apple/           # Apple Sign In client
│   │   ├── service.go       # Core auth business logic
│   │   ├── jwt.go           # JWT token handling
│   │   └── middleware.go    # HTTP auth middleware
│   ├── config/              # Configuration loading and validation
│   ├── server/              # HTTP server, routes, handlers
│   ├── taskqueue/           # PGMQ-based task queue system
│   ├── consumers/           # Background task consumers
│   └── [domain]/            # Domain packages (prices, postal, etc.)
│       ├── db/              # Generated sqlc code (DO NOT EDIT)
│       │   ├── schema.sql   # Table definitions for sqlc
│       │   ├── queries.sql  # SQL queries for sqlc
│       │   └── *.go         # Generated code
│       ├── client/          # External API clients
│       └── service.go       # Business logic
├── db/
│   └── migrations/          # Tern migration files
└── sqlc.yaml                # sqlc configuration
```

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
      
      "koditon-go/internal/auth"
      "koditon-go/internal/config"
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
- Generated code in `internal/*/db/` is READ ONLY - never hand-edit
- SQL queries go in `internal/*/db/queries.sql` with sqlc annotations
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
- Imperative mood: "Add feature" not "Added feature" or "Adds feature"
- Concise subject line (50 chars or less)
- Example: "Add session refresh token rotation"
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
- All config in `internal/config/config.go`
- Required fields fail fast on startup

### HTTP Framework
- Huma v2 for OpenAPI-first design
- Automatic request/response validation
- Type-safe handlers with input/output structs
- Middleware for auth, logging, CORS

### TUI Architecture (`internal/tui`)
- Entrypoint is `NewApp(runner).Model()` used by `cmd/tui/main.go`; do not reintroduce a monolithic model.
- Navigation uses stack router primitives in `router.go` with `Screen` + `Navigator` contracts.
- Shell layout is centralized in `shell.go`; screens provide `ShellState()` and body content only.
- Shared UI primitives live in `primitives_*.go` (list, input, fuzzy picker, job view). Reuse primitives before adding screen-local widget logic.
- Action execution lifecycle is centralized in `runtime.go` (`jobRuntime`), including single-active-job enforcement and cancellation.
- Screen flow is split by concern in `screens_*.go`: home, actions, city picker, prompt, job.
- Domain actions remain in `actions.go`; keep business execution there and keep screens focused on interaction state.
- Global behavior: `q` quits; `ctrl+c` cancels active job first, then quits when no active job exists.
- When adding screens, preserve push/pop/replace navigation semantics and keep breadcrumb/help text consistent through shell state.
- Keep rendering deterministic for tests; avoid time/random-dependent output in snapshot paths without test injection.

### TUI Testing
- Run `go test ./internal/tui/...` for router/runtime/snapshot coverage.
- Snapshot fixtures are in `internal/tui/testdata/*.golden`.
- Regenerate snapshots with `UPDATE_GOLDEN=1 go test ./internal/tui -run TestScreenSnapshots`.

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
