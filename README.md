# Kill Team live tracker

A phone-first tracker for Warhammer 40,000: Kill Team (2024). It answers one
question mid-game: **"what can I use right now?"** — the ploys, chapter tactics,
equipment and operative abilities legal in the current moment, plus what is
already running and when it expires, and the active operative's datacard stats.

Ships with two kill teams — **Angels of Death** and **Plague Marines** — and a
team picker in Setup to switch between them (each has its own census in
`data/teams/`).

It is a **bookkeeper, not a rules arbiter**: it tracks state and surfaces
reminders; it never models the board, dice, damage, or anything requiring input
the player won't give mid-game. See [CLAUDE.md](CLAUDE.md) for the full design
rationale — read it before adding features.

## Architecture

Originally a single client-only HTML file, now split into a Go backend that owns
all the rules logic and a thin React frontend that renders it.

```
              ┌────────────────────────── backend (Go) ───────────────────────────┐
  browser     │  events[] ──fold()──▶ State ──derive()+rules──▶ View (hydrated)     │
 ┌────────┐   │    append-only log     pure reducer      what's usable/running/…    │
 │ React  │──▶│  POST /api/games/{id}/events  ─────────────────────▶  returns View  │
 │  SPA   │◀──│  every tap is one event; the server folds + derives and replies     │
 └────────┘   │  rules census: data/teams/<id>.json   games persisted to games/     │
              └────────────────────────────────────────────────────────────────────┘
```

The engine (`backend/internal/engine`) is a direct, tested port of the original
client-side reducer. The frontend holds **no** rules logic: it posts events and
renders the derived view, plus a presentation-only join for the stat panel.

| Path | Role |
| --- | --- |
| `backend/` | Go API server: rules loader, engine (fold + derive), game store, HTTP API. |
| `backend/internal/engine` | The canonical rules engine and its test suite. |
| `frontend/` | React + TypeScript + Vite thin client. |
| `data/teams/*.json` | Per-team rules census (`aod.json`, `plague_marines.json`) — the source of truth. Hand-verified. |
| `tools/rules-pipeline/` | Fetch official PDFs and extract text for verification (see its README). |
| `legacy/` | The original single-file app, kept for provenance. |

## Running it

**Backend** (needs Go 1.23+):

```bash
cd backend
go test ./...                 # fold scripted event sequences, assert derive output
go run ./cmd/server           # serves http://localhost:8080
```

All loaded teams are served at once — the client picks one in Setup and each
game is bound to its team. Useful flags / env: `-addr` (`KT_ADDR`), `-data`
(`KT_DATA_DIR`, the teams dir), `-games` (`KT_GAMES_DIR`, persistence dir; empty
disables it), `-team` (`KT_TEAM`, the *default* team for new games; when several
are loaded and none is given it defaults to the first alphabetically),
`-static` (`KT_STATIC_DIR`, serve a built frontend).

**Frontend** (needs Node 18+; this repo uses `pnpm` via `corepack`):

```bash
cd frontend
corepack pnpm install
corepack pnpm dev             # http://localhost:5173, proxies /api to :8080
corepack pnpm build           # type-check + production build into dist/
```

**One process (prod-style):** build the frontend, then point the backend at it:

```bash
cd frontend && corepack pnpm build && cd ..
go -C backend run ./cmd/server -static ../frontend/dist    # app + API on :8080
```

## API

| Method & path | Purpose |
| --- | --- |
| `GET /api/teams` | list loaded teams |
| `GET /api/teams/{id}/rules` | full rules census for a team |
| `POST /api/games` | create a game → `{id, view}`; optional body `{"team": "plague_marines"}` (defaults to the active team) |
| `GET /api/games/{id}` | current derived view (its `team` says which team's rules to load) |
| `POST /api/games/{id}/events` | append one event → view |
| `POST /api/games/{id}/undo` | drop the last event → view |
| `POST /api/games/{id}/reset` | clear the log → view |
| `GET /api/games/{id}/log` | raw event log (debug/export) |

An **event** is `{"t": "ACTIVATE", "p": {"id": "aod.strat.combat_doctrine", "opt": "devastator"}}`.
The event vocabulary is defined in `backend/internal/engine/state.go`.

## Updating rules from official PDFs

```bash
tools/rules-pipeline/update-rules.sh aod              # or: plague_marines, or --all
```

See [tools/rules-pipeline/README.md](tools/rules-pipeline/README.md). The
pipeline extracts trustworthy source text; editing the census stays a manual,
card-verified step by design.

## Adding a team

The Plague Marines census (`data/teams/plague_marines.json`) is a worked
example: add the team's PDF to `tools/rules-pipeline/sources.json`, run the
pipeline, transcribe the extracted text into a new `data/teams/<id>.json`
(keeping `vocab` unchanged and setting `meta.defaultRoster`), and it is picked
up automatically — the engine, store and team picker are all team-agnostic.
