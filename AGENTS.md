# Agent Guidelines

This repository contains multiple sub-projects with their own agent guidelines.

## General Agent Behavior

- **Do NOT write summary documents or other markdown junk.** When completing tasks, make direct code changes and provide concise explanations inline. Avoid creating unnecessary documentation files.
- **Minimal code comments**: Keep comments minimal and purposeful. Avoid obvious comments that restate what the code does.
- **No spaces inside code blocks**: Do not add blank lines within function bodies or code blocks for "readability".
- **When comments are needed**: Split code sections with short, descriptive comments that explain *why*, not *what*.
- **Modern Go patterns**: Keep the codebase up to date with latest Go idioms and best practices. This includes proper error handling (e.g., checking `resp.Body.Close()` errors), using contemporary standard library features, and following current Go conventions.

## Backend

For guidelines on working with the Go backend, see [backend/AGENTS.md](./backend/AGENTS.md).

This includes:
- Project structure and module organization
- Build, test, and development commands
- Database tooling and migration workflows
- Coding style and naming conventions
- Testing guidelines
- Commit and pull request guidelines

## Taskfile Shortcuts

The repository root contains a `Taskfile.yml` with shortcuts for common operations:

### Development
- `task backend:dev`: runs the backend server with file watching and auto-reload
- `task client:dev`: runs the frontend development server

### Database
- `task db:new NAME=...`: creates a new timestamped migration file
- `task db:migrate`: applies pending database migrations
- `task db:status`: shows current migration state
- `task db:generate`: regenerates `internal/*/db` from `db/schema` and package-level `queries.sql` files

### Client Build & Quality
- `task client:generate`: rebuilds the client SDK when API contracts or return types change
- `task client:build`: builds the client app for production
- `task client:lint`: lints the client app
- `task client:preview`: previews the built client

## General Repository Notes

- Each sub-project may have its own task definitions and conventions
- Refer to the specific AGENTS.md file in each sub-project directory for detailed guidance
- SQL queries are co-located with their packages (e.g., `internal/pgmq/queries.sql`) for better maintainability