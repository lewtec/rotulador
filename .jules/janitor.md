## 2024-07-26 - Handle Unchecked `w.Write` Error in Favicon Handler

**Issue:** The `errcheck` linter identified an unhandled error returned by `w.Write` when serving the favicon in `annotation/app.go`.

**Root Cause:** The original code did not check the return value of `w.Write`. While errors during favicon serving are rare, network issues or client connection problems can cause `w.Write` to fail. Ignoring such errors can lead to silent failures and make debugging difficult.

**Solution:** I wrapped the `w.Write` call in an `if` statement to check for a non-nil error. If an error occurs, it is now logged using `log.Printf`, making the failure visible for debugging purposes.

**Pattern:** Always check the error return values of I/O operations, such as `w.Write`, `f.Close()`, and `io.Copy`. In cases where the application can gracefully continue, log the error for observability.

## 2026-01-24 - Reduce Complexity in Dependency Logic & Fix Build Error

**Issue:** Duplicate logic for resolving task dependencies and finding task indices was scattered across 5 methods in `annotation/app.go`. Also, a build error existed due to using `log` package instead of `slog`.

**Root Cause:** Copy-paste programming for dependency handling logic.

**Solution:** Extracted `findTaskIndex` and `getDependencyImageHashes` into `annotation/helpers.go` and refactored `app.go` to use them. Replaced `log.Printf` with `a.Logger.Warn`.

**Pattern:** Extract complex, repeated logic (especially involving loops and error handling) into private helper methods on the struct. Always use the project's standard logger (`slog`).
