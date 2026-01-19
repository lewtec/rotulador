## 2024-07-26 - Handle Unchecked `w.Write` Error in Favicon Handler

**Issue:** The `errcheck` linter identified an unhandled error returned by `w.Write` when serving the favicon in `annotation/app.go`.

**Root Cause:** The original code did not check the return value of `w.Write`. While errors during favicon serving are rare, network issues or client connection problems can cause `w.Write` to fail. Ignoring such errors can lead to silent failures and make debugging difficult.

**Solution:** I wrapped the `w.Write` call in an `if` statement to check for a non-nil error. If an error occurs, it is now logged using `log.Printf`, making the failure visible for debugging purposes.

**Pattern:** Always check the error return values of I/O operations, such as `w.Write`, `f.Close()`, and `io.Copy`. In cases where the application can gracefully continue, log the error for observability.

## 2026-01-19 - Resolve "Description" Localization Key Conflict

**Issue:** The localization key "Description" was conflicting with `go-i18n`'s internal metadata handling, preventing proper translation of the project description header on the Help page.

**Root Cause:** The `go-i18n` library treats "Description" as a reserved keyword for message metadata when parsing translation files, causing it to be ignored or mishandled as a message ID.

**Solution:** Renamed the localization key from "Description" to "ProjectDescription" in all locale files (`annotation/locales/*.json`) and the Help page template (`annotation/templates/pages/help.html`). Also fixed an unrelated issue where `log.Printf` was used instead of `slog` and restored the missing `hash-password` CLI command.

**Pattern:** Avoid using common words like "Description", "ID", or "Other" as message IDs in `go-i18n` if they overlap with the library's reserved struct fields. Use specific prefixes (e.g., "ProjectDescription") to avoid conflicts.
