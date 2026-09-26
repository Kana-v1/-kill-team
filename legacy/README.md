# Legacy single-file tracker

These are the original client-only app, superseded by the `backend/` (Go) +
`frontend/` (React) split. They are kept for provenance only.

- `tracker_src.html` — the whole app: style block, markup, and the
  `events[] → fold() → derive() → render()` script inline. The Go engine in
  `backend/internal/engine` is a direct, tested port of this script's reducer.
- `build.js` — the old injector that inlined `aod.json` into the template.

The canonical rules data (`data/teams/aod.json`) is unchanged and is now served
by the backend instead of being injected at build time.
