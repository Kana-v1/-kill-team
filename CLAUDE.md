# Kill Team live tracker — Angels of Death

A phone-first tracker for Warhammer 40,000: Kill Team (2024). It answers one question mid-game:
**"what can I use right now?"** — ploys, chapter tactics, equipment and operative abilities that are
legal in the current moment, plus what is already running and when it expires. It also shows the
active operative's datacard stats so you don't reach for the card.

**Structure:** a Go backend that owns all rules logic and a thin React frontend that renders it.
Read `README.md` for how to run both. The engine in `backend/internal/engine` is a direct, tested
port of the original client-only reducer; the old single-file app is preserved under `legacy/`
(and was once published as the Claude artifact `88351e67-f1d8-4c9b-a55f-543f6775c66f`).

## Scope — read this before adding features

This is a **bookkeeper, not a rules arbiter**. It tracks state and surfaces reminders; it never
resolves the game. This is the single most important design decision and it came out of prior art
(Gloomhaven Helper explicitly refuses to apply damage or validate legality; two decades of
open-source Magic engines show what the alternative costs).

Concretely, the app must never model:

- the board — positions, distances, line of sight, cover, control range
- dice resolution, damage, or wound totals
- anything requiring input the player won't give mid-game

A ploy only has to declare three things: **when it's legal, how long it lasts, and which moments
should surface it.** Nothing about what it mechanically does.

**Prompts state conditions, they never assert them.** The app cannot see the killzone, so write
"Balanced if the target is more than 6\" away", not "you have Balanced". Same for every condition
involving position or range. The one exception is a condition the app genuinely knows — see
`selectedIs` below.

## Files

| File | Role |
| --- | --- |
| `data/teams/aod.json` | All rules data. Effects, chapter tactics, operatives, weapons, vocab. The single source of truth; hand-verified. |
| `backend/internal/rules/rules.go` | Structs mirroring the JSON + the loader. |
| `backend/internal/engine/state.go` | `Event`, `State`, and `fold`/`apply` — the reducer. |
| `backend/internal/engine/derive.go` | `routes`/`quote`/`derive` → the hydrated view the client renders. |
| `backend/internal/engine/engine_test.go` | Scripted event sequences; the verification harness. |
| `backend/internal/store/store.go` | Per-game event logs, `SURFACE`/undo handling, best-effort file persistence. |
| `backend/internal/api/api.go` | HTTP JSON API. |
| `backend/cmd/server/main.go` | Server entrypoint and flags. |
| `frontend/src/` | React + TS thin client. `App.tsx` orchestrates; `components/` render; `api.ts` is the typed client. No rules logic lives here. |
| `tools/rules-pipeline/` | Fetch official PDFs and extract text with `lit` for verification. |
| `legacy/` | The original single-file app, superseded. |

## Architecture

The Go backend owns the whole pipeline; the frontend posts events and renders the returned view.

```
events[]        append-only log per game: CTX, CP, TP_NEXT, ACTIVATE, DOWN, LEADER, COUNT, TACTIC, …
  ↓ fold()      pure reducer (engine/state.go) → State{ tp, cp, ctx, roster, dead, equip, tactics, active, … }
  ↓ derive()    + rules (engine/derive.go) → View{ usable[], running[], spent[], prompts[], … }  ← hydrated
              the React client renders the View verbatim; every tap is one POSTed event
```

Invariants, in order of importance:

1. **Derive, never mutate.** No "+1 on apply, −1 on expiry" anywhere. Expiry is not an event: an
   effect with `duration: "end_of_turning_point"` simply stops matching once the TP counter moves.
   Adding subtract-on-expiry logic is the classic buff-system bug and will corrupt state permanently.
2. **The log is the state.** Every change is an event; current state is a function of the log.
   Undo drops the last event and re-folds. Never mutate `State` outside `apply()`.
3. **Surfacing is the one side effect.** When a paid effect is shown at a matching moment, the store
   appends a `SURFACE` event, which feeds the end-of-turning-point "never used" summary. `Derive`
   itself stays pure and reports fresh surfaces as advisory output; the store persists them. Undo
   skips trailing `SURFACE` events for this reason.
4. **Persistence is best-effort.** Game logs mirror to `backend/games/<id>.json`; a failed read or
   write must never take the server down. The frontend keeps only the game id in `localStorage`,
   wrapped in try/catch.

## Data model

Every entry in `effects[]`:

| Field | Meaning |
| --- | --- |
| `kind` | `strategy_ploy` · `firefight_ploy` · `equipment` · `faction_rule` · `operative_ability` |
| `cost.cp` | Base cost. All ploys are 1CP unless a card says otherwise (core rules default). |
| `window` / `windows[]` | Contexts where it's legal. `any` for always. Use `windows[]` for several. |
| `duration` | When it stops applying. See vocab below. |
| `once_per` | `turning_point` or `battle`. Omit for unlimited. |
| `remind_at[]` | Trigger moments that should surface it. **This is the product.** |
| `prompts` | Keyed by trigger, or `"*"` for all. The text shown in the amber banner. |
| `options[]` | Sub-selections (Combat Doctrine's three doctrines). Each may carry its own `remind_at`. |
| `requiresOperative` | Only exists while that operative is in the roster and not incapacitated. |
| `alwaysOn` | Passive; goes straight to Running, never Usable. |
| `costOverrides[]` | Discounts this entry grants to *other* entries — see below. |
| `source` | Where it came from. `operative datacard, verified by Maksym` is the gold standard. |
| `verify[]` | Open questions about this entry. |
| `disputed` | Sources conflict. Renders a red **Unverified** tag in the UI. |

### Cost overrides

An ability can change another effect's price. Fields on each override:

- `effect` or `kind` — what it discounts
- `options[]` — only certain sub-selections (Doctrine Warfare's specific doctrines)
- `excludes[]` — explicit exemptions (Heroic Leader does not discount Command Re-roll)
- `once_per` + `group` — a shared `group` means one use consumes *all* options of that ability
  (Heroic Leader is "one of the following" per turning point)
- `selectedIs` — only when the currently selected operative is that one. This is the only
  condition the app can actually evaluate, because the player has already told it who is acting.
  Non-matching routes appear as a dim "can be free" hint rather than a changed price.
- `condition` — human-readable text for what the app can't check

### Closed vocabularies

Do not invent new values. If a card genuinely needs one, add it to `vocab` and say so.

- **windows** — `strategy_phase` `activation.own` `combat.shoot` `combat.fight` `combat.retaliate`
  `defence.shooting` `counteract` `any`
- **durations** — `instant` `this_sequence` `this_activation` `this_counteraction`
  `end_of_turning_point` `battle`
- **triggers** — the seven contexts × `before_roll` / `after_roll`, plus `turning_point.start`,
  `activation.start`, `counteract.available`, `turning_point.end`

`before_roll` vs `after_roll` matters: Indomitus and Transhuman Physiology only make sense once
you've seen your dice. Retaliating and counteracting are distinct from fighting and defending —
they are the opponent's-turn moments, which is exactly when players forget their buffs.

## Sourcing rules — non-negotiable

**The newest official PDF, as linked on the downloads page, is the authority** (errata included).
Secondary sites (ktdash, ktdojo, wahapedia) are hints only; they have been wrong. A ktdash weapon
list and a web-searched January PDF each nearly put wrong datacards into the census.

Get rules **only** through the pipeline (`tools/rules-pipeline/README.md`):
`update-rules.sh --sync --all`, then `audit.py <team>`, then a visual pass on the rendered pages,
then a reviewed edit.
- **Never web-search for a PDF link or hand-edit one.** `sync_sources.py` reads the downloads
  page's own API.
- **Never `WebFetch` a PDF.** It returns a model's summary, which invented rules (Doctrine Warfare
  on the Captain).
- **Text extraction can't see strikethrough,** so deleted errata text looks live. Read the errata
  boxes on the rendered page images.
- Anything unresolved gets `verify[]`, and `disputed: true` if sources conflict. The UI shows
  **Unverified**. Never quietly present an unverified rule as fact.

### Currently unresolved

- Nothing. Ploy costs are settled: team cards print no cost, and the official Lite rules say
  ploys cost 1CP and each ploy except Command Re-roll is once per turning point. Both censuses
  were checked page by page against the August '26 PDFs on 2026-09-26.

## Team composition (Angels of Death)

1 leader — Space Marine Captain, Intercessor Sergeant, or Assault Intercessor Sergeant — plus 5
others. Other than **Warrior** operatives, each may be taken only once; Warriors repeat, so roster
entries are instances (`intercessor_warrior#2`), and `typeOf()` strips the suffix.

**Leader choice changes the rules**, and this is load-bearing:

| Leader | Grants |
| --- | --- |
| Captain | Heroic Leader, Iron Halo |
| Intercessor Sergeant | Doctrine Warfare (Devastator / Tactical free), Chapter Veteran |
| Assault Intercessor Sergeant | Doctrine Warfare (Assault / Tactical free), Chapter Veteran |

Chapter Veteran grants that operative one *additional* chapter tactic, so the extra tactic slot in
Setup appears only when a Sergeant leads. The Heavy Intercessor Gunner has no additional rules.

## UI conventions

- Committed dark single-theme. Gaming tables are dim; do not add a light theme.
- Chips **wrap**, never scroll. A hidden-scrollbar row cost the user half the controls once.
- Every card carries a type badge, because "usable now" is meaningless without knowing whether it's
  a strategy ploy, an ability, or equipment.
- Running splits into *Applies right now* (lit) and *Active, not in this window* (dimmed).
- Reminders are one-tap dismissible and dismissal persists for that duration. Nagging is a
  documented reason companion apps get abandoned.
- Budget one tap for the common path. If logging a thing costs more than remembering it, the app loses.

## Frontend conventions

- Operative art is generated inline SVG (`ICONS` in `frontend/src/icons.tsx`). Do not scrape and
  embed GW artwork. Each operative has a `photo` slot — a data URI there replaces the glyph. The
  user's own model photos are the intended source.
- The frontend holds no rules logic. It posts events and renders the `View`. The only join it does
  locally is presentation: the stat panel filters the selected operative's weapons by context.
- External images, stylesheets and fetches beyond Google Fonts are unnecessary — keep it that way.

## Adding another kill team

The backend is multi-team: one engine per loaded census, and each game is bound to a team (the
store records it, `View.team` reports the route key so the client loads the right rules). The
server serves every `data/teams/*.json`; `-team` only sets the *default* for new games. The
frontend's Setup has a team picker; switching starts a fresh game for that team.

To add one: add the team's PDF url to `tools/rules-pipeline/sources.json`, run the pipeline, and
transcribe the extracted text into `data/teams/<newId>.json` — keep `vocab` unchanged and set
`meta.defaultRoster` to the starting six (the engine derives one if it's absent, but be explicit).
Budget a card-reading verification pass as part of the work, not a nicety at the end. `Plague
Marines` (`data/teams/plague_marines.json`) is the worked example: it has no chapter tactics (an
empty `chapterTactics[]`, and the Setup hides those sections), and its Poison mechanic is surfaced
as reminders only — the app tracks no tokens. The vocabulary held for both teams without additions;
if a new team needs a new window, duration or trigger, that's a real finding worth noting rather
than a one-off patch.

## Verification

The test suite is `backend/internal/engine/engine_test.go`: it folds scripted event sequences and
asserts `Derive` output — the same harness idea as before, now in Go. Run `go test ./...` in
`backend/` for anything touching `quote`, `routes`, `apply`, or expiry.
Worth running for anything touching `quote()`, `routes()` or expiry.
