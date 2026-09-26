# Rules-update pipeline

Fetches a team's **official** rules PDF and extracts authoritative text with
[`lit`](https://www.npmjs.com/package/liteparse) (liteparse), staging it for a
card-verification pass.

This exists because the old approach — `WebFetch` on the PDF — did not return
the PDF; it returned a small model's *description* of it, which fabricated rules
(see CLAUDE.md, "Sourcing rules"). `lit` parses the PDF's real text layer, so
what you read is what the document says.

## Prerequisites

- `lit` on `PATH` (`liteparse` 2.x). `lit --version` should print `2.x`.
- `python3`, `curl`.
- Under WSL with a Windows-installed `lit`, the script auto-stages the PDF on a
  Windows-accessible temp dir (the Windows binary rejects UNC `\\wsl.localhost`
  paths); nothing extra to configure.

## The flow (the only sanctioned way to get rules)

```bash
tools/rules-pipeline/update-rules.sh --sync --all      # 1. refresh links from the downloads page, 2. extract
python3 tools/rules-pipeline/audit.py <teamId>         # 3. census vs PDF → data/extracted/<teamId>.audit.md
python3 tools/rules-pipeline/sync_sources.py --check   # anytime: exit 1 if GW published anything newer
```

1. **Sync.** `sync_sources.py` asks the downloads page's own search API for the
   current list. It sends a POST to `https://www.warhammer-community.com/api/search/downloads/`
   with `{"index":"downloads_v2","searchTerm":"","gameSystem":"kill-team","language":"english"}`.
   It then rewrites `sources.json`, with `teams.<id>` for every team-rules PDF and
   `core.*` for the Lite rules, Universal Equipment and the Core Rules update log.
   **Never web-search for a PDF link or hand-edit one.** A search result once
   supplied January Plague Marines rules while the page already linked the
   August update, and the "fixes" drafted from it would have deleted real weapons.
2. **Extract.** `update-rules.sh <id>` writes `<id>.txt` (datacard rows intact),
   `<id>.cols.txt` (cards in reading order, via `columns.py`) and `<id>.json`.
3. **Audit.** `audit.py <id>` diffs datacard stats and weapons field by field, and
   ploy/equipment/rule text by numbers and wording. It also lists PDF cards
   missing from the census and the entries still sourced from secondary sites.
4. **Look at the pages.** Text extraction can't see strikethrough: deleted errata
   text reads as live. Render the pages (`lit screenshot`) and read the errata boxes,
   plus any datacard you change. The Kill Team Selection photo pages scramble
   which weapon label belongs to which operative in extracted text.
5. **Show the discrepancies, then edit** the census. Set `meta.sourcePdf` and
   `meta.rulesVersion` (the ERRATA month). `reconcile_2026_09.py` is the worked
   example of a reviewed batch of edits.

Outputs (git-ignored):

| Path | What |
| --- | --- |
| `data/pdf/<id>.pdf` | the downloaded PDF |
| `data/extracted/<id>.txt` | extracted text (rows intact) |
| `data/extracted/<id>.cols.txt` | column-aware reading order |
| `data/extracted/<id>.json` | per-page text items with coordinates |
| `data/extracted/<id>.audit.md` | census-vs-PDF report |

Authority: the newest official PDF, as linked on the downloads page, errata
included. Secondary sites (ktdash, ktdojo, wahapedia) are hints, never sources:
they have been wrong.

## What the pipeline does NOT do

It never writes `data/teams/<teamId>.json`. Turning prose into the structured
census is a human judgement call, and per the project's sourcing rules only
**card-verified** text is allowed to settle there. The script instead:

1. gets you trustworthy source text, and
2. lists which census entries still carry `verify[]` assumptions to reconcile.

Read the extracted text against the datacards, then edit the census by hand.

## OCR

The text layer is extracted without OCR by default (fast, exact). A few values
are printed as graphics (notably ploy base CP costs) and won't appear in the
text. To try OCR, run with `LIT_OCR=1 OCR_SERVER=<url>` — `lit`'s OCR needs an
HTTP OCR server (`--ocr-server-url`).
