# Consistently Ignored Changes

This file lists patterns of changes that have been consistently rejected by human reviewers. All agents MUST consult this file before proposing a new change. If a planned change matches any pattern described below, it MUST be abandoned.

---

## IGNORE: Do not create pull requests for code generation changes

**- Pattern:** Do not add CI steps that automatically create pull requests for code generation changes.
**- Justification:** This change has been proposed multiple times and rejected. Developers are expected to run code generation tasks (`mise gen`) and commit the results locally before pushing their changes.
**- Files Affected:** `.github/workflows/autorelease.yml`
