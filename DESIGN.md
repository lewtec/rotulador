# Design: Rotulador app shell

## Mode

**Operate** — multi-hour labeling sessions. Home/help are thin doors and reference; annotate is the product.

## Scope (this pass)

Structure + quiet craft. Not a rebrand. Steal Backstage layout primitives, not festival brand.

## Constraints

- UI is lived in for **hours** (low chrome noise, stable controls, no decorative motion).
- Images **never crop** (`object-contain` only).
- Layout shift must **never** move annotate action keys.

## Shell geometry

Viewport-locked column:

```
body.app-frame: height 100vh (fallback) / 100dvh when supported
  header: shrink-0  (navbar bg-base-100 border-b border-base-300)
  main#app-main: flex-1 min-h-0  (scroll or flex; toast host lives here)
  dock?: shrink-0   (annotate only: progress + action buttons; safe-area padding)
```

- **`.app-frame`**: `100vh` so desktop always fills the window; `100dvh` under `@supports` so mobile URL bars do not cover the dock. No `max-height` cap.
- Dock bottom padding includes `env(safe-area-inset-bottom)`; viewport meta has `viewport-fit=cover`.
- **No product footer.**
- Toast host is **scoped to main** so nothing under main (the dock) is covered.
- Surface: field `bg-base-200`, chrome `bg-base-100`, border edge, little/no heavy shadow.

## Primitives (`internal/ui/layout`)

| Primitive | Role |
|-----------|------|
| `Document` | HTML document + theme + HTMX + CSS |
| `Shell` | Header + scrolling main (home, help, complete) |
| `ShellColumn` | Header + free column; page supplies `main` + optional dock (annotate) |
| `PageHeader` | Title, optional lead, crumbs, action slot |
| `PageBody` / `PageBodyNarrow` | Content stacks under header |
| `HeaderBtn` / `HeaderBtnPrimary` | Shared header action classes |

**Density**

- **Annotate:** compact page chrome (`PageHeader` compact): smaller title, tight gaps.
- **Home / help:** slightly airier headers; same components.

## Annotate layout

```
header (fixed)
main#app-main (flex-1, flex col, min-h-0, relative; toast here):
  compact meta (shrink-0):
    breadcrumbs
    PageHeader: task title | Help (btn btn-sm / HeaderBtn)
    subtitle: mono image id (+ optional count)
  image region (flex-1 min-h-0):
    img object-contain max-w-full max-h-full centered
dock (shrink-0):
  progress bar (segment list) glued to top of dock
  class buttons + Not sure  — stable Y, never moves
```

## Progress bar

`components.Progress` = `{ Total, Segments[] }` with count, percent, i18n label, class.

## Out of scope

- Full visual rebrand / new palette world
- Sidebar, festival chrome, loud motion
- Changing HTMX / labeling semantics
