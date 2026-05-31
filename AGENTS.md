# Agent Guidelines

This repository contains multiple sub-projects with their own agent guidelines.

## General Agent Behavior

- **Do NOT write summary documents or other markdown junk.** When completing tasks, make direct code changes and provide concise explanations inline. Avoid creating unnecessary documentation files.
- **Minimal code comments**: Keep comments minimal and purposeful. Avoid obvious comments that restate what the code does.
- **No spaces inside code blocks**: Do not add blank lines within function bodies or code blocks for "readability".
- **When comments are needed**: Split code sections with short, descriptive comments that explain *why*, not *what*.
- **Modern Go patterns**: Keep the codebase up to date with latest Go idioms and best practices. This includes proper error handling (e.g., checking `resp.Body.Close()` errors), using contemporary standard library features, and following current Go conventions.

## Agent Tooling

- Use Chrome DevTools MCP for browser testing and web flow verification.
- Use the PostgreSQL MCP connection for database inspection and SQL queries.

## Backend

For guidelines on working with the Go backend, see [backend/AGENTS.md](./backend/AGENTS.md).

This includes:
- Project structure and module organization
- Build, test, and development commands
- Database tooling and migration workflows
- Coding style and naming conventions
- Testing guidelines
- Commit and pull request guidelines
- TUI architecture (`internal/tui`) including router, primitives, screen flow, and snapshot tests

## mise Tasks

Tasks are managed with [mise](https://mise.jdx.dev/). Run `mise tasks` in any directory to see what's available.

### Development
- `mise run backend:dev`: runs the backend server with file watching and auto-reload (from project root)
- `mise run dev`: same, when already in `backend/`

### Database
Run from project root using `backend:` prefix, or from `backend/` without prefix:
- `NAME=... mise run backend:db:new`: creates a new timestamped migration file
- `mise run backend:db:migrate`: applies pending database migrations
- `mise run backend:db:status`: shows current migration state
- `mise run backend:db:generate`: regenerates `internal/*/db` from `db/schema` and package-level `queries.sql` files

### Web (run from anywhere)
- `mise run web:dev`: starts the Vite dev server
- `mise run web:build`: type-checks and builds the web bundle
- `mise run web:generate`: regenerates the API client with Orval

### iOS (Native Swift App in `koditon/`)

Use `task` (Go Task) directly from the `koditon/` directory:
- `task build`: builds the iOS app using xcede
- `task run`: runs the iOS app on the simulator (does not exit on its own - must be terminated manually)
- `task build-run`: builds and runs the iOS app in one command (does not exit on its own - must be terminated manually)

The device ID is read from `IOS_DEVICE_ID` in `.env`. You can override the platform with `IOS_PLATFORM=device` for physical devices.

## General Repository Notes

- Each sub-project may have its own task definitions and conventions
- Refer to the specific AGENTS.md file in each sub-project directory for detailed guidance
- SQL queries are co-located with their packages (e.g., `internal/pgmq/queries.sql`) for better maintainability
