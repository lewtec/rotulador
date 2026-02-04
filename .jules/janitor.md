## 2024-07-26 - Handle Unchecked `w.Write` Error in Favicon Handler

**Issue:** The `errcheck` linter identified an unhandled error returned by `w.Write` when serving the favicon in `annotation/app.go`.

**Root Cause:** The original code did not check the return value of `w.Write`. While errors during favicon serving are rare, network issues or client connection problems can cause `w.Write` to fail. Ignoring such errors can lead to silent failures and make debugging difficult.

**Solution:** I wrapped the `w.Write` call in an `if` statement to check for a non-nil error. If an error occurs, it is now logged using `log.Printf`, making the failure visible for debugging purposes.

**Pattern:** Always check the error return values of I/O operations, such as `w.Write`, `f.Close()`, and `io.Copy`. In cases where the application can gracefully continue, log the error for observability.

## 2026-01-24 - Deduplicate Dependency Handling Logic in Annotation App

**Issue:** Significant code duplication (approx. 40-50 lines per instance) in `annotation/app.go` for finding task indices and pre-fetching dependency image hashes, appearing in 4 different methods.
**Root Cause:** As the application grew, the logic for handling task dependencies (the `If` field in tasks) was copy-pasted into every method that needed to filter images based on previous annotations.
**Solution:** Extracted `findTaskIndex` and `getDependencyImageHashes` into a new `annotation/helpers.go` file. Refactored `CountEligibleImages`, `CountAvailableImages`, `GetPhaseProgressStats`, and `NextAnnotationStep` to use these helpers.
**Pattern:** Complex configuration traversal and dependency resolution logic should be centralized in helper methods, especially when used in loops or multiple contexts. This reduces the risk of inconsistent behavior when logic changes.
