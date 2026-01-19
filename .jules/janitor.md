## 2024-07-26 - Handle Unchecked `w.Write` Error in Favicon Handler

**Issue:** The `errcheck` linter identified an unhandled error returned by `w.Write` when serving the favicon in `annotation/app.go`.

**Root Cause:** The original code did not check the return value of `w.Write`. While errors during favicon serving are rare, network issues or client connection problems can cause `w.Write` to fail. Ignoring such errors can lead to silent failures and make debugging difficult.

**Solution:** I wrapped the `w.Write` call in an `if` statement to check for a non-nil error. If an error occurs, it is now logged using `log.Printf`, making the failure visible for debugging purposes.

**Pattern:** Always check the error return values of I/O operations, such as `w.Write`, `f.Close()`, and `io.Copy`. In cases where the application can gracefully continue, log the error for observability.

## 2026-01-16 - Fix Build Errors and Cleanup annotation/app.go

**Issue:** The project failed to build due to an undefined `log` package reference in `annotation/app.go` and a missing `hashPasswordCmd` definition in `cmd/rotulador`. Additionally, `annotation/app.go` contained an unchecked error from `migrate.NewWithInstance` and unused code (`i18n` field, `stringOr` function).

**Root Cause:** The `log` package was used without import (likely intended to be `slog`). The `hashPasswordCmd` variable was referenced in `root.go` but the implementation was missing. The error return from the migration instance creation was assigned but ignored (shadowed).

**Solution:**
1.  Replaced `log.Printf` with `a.Logger.Warn` to use the configured `slog` logger.
2.  Implemented the missing `hashPasswordCmd` in `cmd/rotulador/hash_password.go`.
3.  Added proper error checking for `migrate.NewWithInstance`.
4.  Removed unused `i18n` field and `stringOr` helper.

**Pattern:** Ensure all referenced symbols are defined and imported. Always check error returns, especially when assigning them to variables that are immediately shadowed. Remove unused code to keep the codebase clean.
