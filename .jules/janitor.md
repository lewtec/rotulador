# Janitor Journal

## Date: $(date)
**Target:** Addressed technical debt in tooling configuration (`mise.toml`).
**Action:** The `gen:i18n` task produced fatal errors during string extraction because stale `active.*.json` files were carrying over reserved keys (like 'Description') causing conflicts during the merge step. Modified `mise.toml` to clean up these stale `active.*.json` files from `annotation/locales/` before extraction.
**Verification:** Running `mise run gen:i18n` now correctly extracts strings without failing on reserved key conflicts.

