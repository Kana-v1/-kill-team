# Legacy

Superseded versions of the tracker, kept for provenance. Nothing here is built
or maintained; the app is `ios/`.

- `tracker_src.html` + `build.js` — the original single-file, client-only app
  (`events[] → fold() → derive() → render()` inline).
- `web/` — the Go backend + React client that replaced it (2026-09), before the
  native iOS app. The Swift engine in `ios/KTEngine` descends from
  `web/backend/internal/engine`, with the later model changes (Strategy /
  Firefight, per-unit applicability, glossary, initiative CP).
