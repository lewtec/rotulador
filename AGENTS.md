# Project Guidelines

## Mise First
- Before doing anything, you MUST install `mise` and use it for all task execution.
- `mise` is a non-negotiable basic necessity.

## Error Handling
- **Never ignore errors.** You must NEVER leave an empty catch block or ignore a returned error value.
- **Centralized Error Reporting:** The project MUST have a single, centralized error-reporting function (e.g. `ReportError`). All code paths that handle unexpected errors MUST funnel through this function.
- **Logging:** All application logging must use the `slog` library. Do not use the `log` package or `fmt.Println` for logging purposes.
- **No silent failures:** Every `catch` block, every `.catch()`, every error callback that is not an expected/recoverable condition MUST call the centralized error-reporting function.

## Code Generation
- Prefer `mise run gen` / `go generate ./...` (CSS, sqlc, goi18n via `go tool`).
- Do not modify the database schema defined in `internal/db/migrations` or rename columns.

## Database
- Database schema changes must be defined in `internal/db/migrations` using `golang-migrate`.
- The test database in `internal/repository/testing.go` automatically applies these migrations.

## Frontend
- Frontend formatting is handled by `prettier`.
- File serving handlers must prevent path traversal.
