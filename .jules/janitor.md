## 2024-07-26 - Handle Unchecked `w.Write` Error in Favicon Handler

**Issue:** The `errcheck` linter identified an unhandled error returned by `w.Write` when serving the favicon in `annotation/app.go`.

**Root Cause:** The original code did not check the return value of `w.Write`. While errors during favicon serving are rare, network issues or client connection problems can cause `w.Write` to fail. Ignoring such errors can lead to silent failures and make debugging difficult.

**Solution:** I wrapped the `w.Write` call in an `if` statement to check for a non-nil error. If an error occurs, it is now logged using `log.Printf`, making the failure visible for debugging purposes.

**Pattern:** Always check the error return values of I/O operations, such as `w.Write`, `f.Close()`, and `io.Copy`. In cases where the application can gracefully continue, log the error for observability.

## 2026-01-21 - Eliminate Code Duplication in Dependency Checking

**Issue:** The logic for checking task dependencies (fetching and filtering image hashes based on `task.If` conditions) was duplicated in four different methods within `annotation/app.go`: `CountEligibleImages`, `CountAvailableImages`, `GetPhaseProgressStats`, and `NextAnnotationStep`. This violates the DRY principle and makes maintenance error-prone.

**Root Cause:** As features were added, the dependency checking block was likely copy-pasted into each new method that required it.

**Solution:** I extracted the common logic into two helper methods in a new file `annotation/helpers.go`:
1. `findTaskIndex(taskID string) int`: Encapsulates the loop to find a task's index by ID.
2. `getDependencyImageHashes(ctx, task) (map[string]map[string]bool, error)`: Encapsulates the complex logic of iterating through dependencies, finding their stage indices, and fetching valid image hashes from the repository.

**Pattern:** When a block of logic (especially involving database queries or complex filtering) is repeated more than twice, extract it into a dedicated helper method. This not only reduces lines of code but also centralizes the logic, making future optimizations or fixes easier to apply universally.
