# Consistently Ignored Changes

This file lists patterns of changes that have been consistently rejected by human reviewers. All agents MUST consult this file before proposing a new change. If a planned change matches any pattern described below, it MUST be abandoned.

---

## IGNORE: Do not vendor tool installers

**- Pattern:** Adding standalone installer scripts (e.g., `install_mise.sh`, `install_go.sh`) to the repository.
**- Justification:** Use standard actions (e.g., `jdx/mise-action`) or system package managers. Committing vendor scripts creates unnecessary maintenance burden and noise.
**- Files Affected:** `*.sh`

## IGNORE: Ad-hoc Error Handling

**- Pattern:** Using direct logger calls (e.g., `slog.Error`) for error handling, instead of the centralized `annotation.ReportError` function.
**- Justification:** The project mandates a centralized error reporting function (e.g., `annotation.ReportError`) to ensure consistent error handling, stack trace capture, and observability integration (Sentry).
**- Files Affected:** `*.go`

## IGNORE: Direct Task Execution in CI

**- Pattern:** Configuring CI workflows to run `npm install`, `go build`, or `go test` directly.
**- Justification:** All tasks must be executed via `mise run <task>` to ensure environment consistency between development and CI.
**- Files Affected:** `.github/workflows/*.yml`

## IGNORE: Weak Path Traversal Checks

**- Pattern:** Using `filepath.Clean` or `path.Join` for validating file paths against a base directory without resolving to an absolute path (`filepath.Abs`).
**- Justification:** `filepath.Abs` provides a stronger guarantee against traversal attacks by anchoring the path to the filesystem root, whereas `Clean` alone may not resolve all relative path ambiguities.
**- Files Affected:** `*.go`

## IGNORE: Manual Modification of Generated Files

**- Pattern:** Manually editing generated files such as `active.*.json` or SQLc output.
**- Justification:** Generated files should only be updated by running the appropriate generation task (e.g., `mise run codegen:i18n`). Manual edits will be overwritten and can lead to inconsistencies.
**- Files Affected:** `annotation/locales/active.*.json`, `internal/sqlc/*.go`

## IGNORE: Refactoring annotation/app.go

**- Pattern:** Splitting `annotation/app.go` into multiple files (e.g., `handlers.go`, `logic.go`, `setup.go`) or extracting private methods to other files.
**- Justification:** Multiple refactoring attempts involving `annotation/app.go` have been consistently rejected. The project prefers keeping the application logic consolidated in `app.go`.
**- Files Affected:** `annotation/app.go`, `annotation/*.go`

## IGNORE: Custom CSRF Middleware

**- Pattern:** Implementing a custom Double Submit Cookie CSRF middleware in `annotation/csrf.go` and wrapping the handler in `annotation/app.go`.
**- Justification:** Attempts to introduce a custom CSRF implementation have been rejected.
**- Files Affected:** `annotation/csrf.go`, `annotation/app.go`

## IGNORE: Renaming Description i18n key

**- Pattern:** Renaming the `Description` localization key to `ProjectDescription` or similar.
**- Justification:** The codebase uses "Description" and attempts to rename it have been rejected.
**- Files Affected:** `annotation/locales/*.json`, `annotation/templates/**/*.html`

## IGNORE: Scoped Test Execution

**- Pattern:** Changing global test commands like `go test ./...` to scoped ones (e.g., `go test ./annotation/...`).
**- Justification:** Attempts to limit test execution scope in CI/tasks have been rejected.
**- Files Affected:** `mise.toml`, `.github/workflows/*.yml`
