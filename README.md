# Kill Team live tracker

A phone-first iOS app for Warhammer 40,000: Kill Team (2024) that answers one
question mid-game: **"what can I use right now?"** It shows:
- the ploys, equipment, chapter tactics and abilities available now, and their price
- what's in play for the operative who's acting
- the rules those effects add to its weapons
- the acting operative's datacard

Rule jargon (Ceaseless, Severe, Poison…) is tappable for its definition.

It's a **bookkeeper, not a rules arbiter**: it tracks state and reminds you,
and never models the board, dice or damage. Read [CLAUDE.md](CLAUDE.md) before
adding features.

Teams: **Angels of Death**, **Plague Marines**. Rules are checked against the
official August '26 PDFs.

## Layout

| Path | What |
| --- | --- |
| `ios/KTEngine/` | The rules engine: a Foundation-only Swift package. Event log → fold → derive. Tested on Linux. |
| `ios/KillTeam/` | The SwiftUI app. |
| `ios/project.yml` | XcodeGen spec; the Xcode project is generated on CI. |
| `data/teams/*.json` | Per-team rules census, the source of truth, bundled into the app as-is. |
| `data/core/glossary.json` | Core weapon-rule definitions, from the official Lite rules. |
| `tools/rules-pipeline/` | Fetch the official PDFs, extract and audit (see its README). |
| `legacy/` | The earlier single-file app and the Go/React web version. |

## Build and test

There's no Mac: **GitHub Actions is the iOS compiler.** Every push touching
`ios/`, `data/teams/` or `data/core/` does two things (`.github/workflows/ios.yml`):
- runs the engine tests
- builds an unsigned `KillTeam.ipa` artifact, which you install with Sideloadly

```bash
gh run list -R Kana-v1/-kill-team            # builds
gh run download <run-id> -R Kana-v1/-kill-team
gh run view <run-id> -R Kana-v1/-kill-team --log-failed
```

The engine tests also run locally on Linux, with Swift installed at `~/sdk/swift`:

```bash
cd ios/KTEngine && swift test
KT_UPDATE_GOLDEN=1 swift test    # after a reviewed rules change, refresh the applicability tables
```

## Updating rules

```bash
tools/rules-pipeline/update-rules.sh --sync --all   # current links from the downloads page, then extract
python3 tools/rules-pipeline/audit.py <team>        # census vs PDF
```

Then read the rendered pages, and edit the census only after the discrepancies
have been reviewed. See [tools/rules-pipeline/README.md](tools/rules-pipeline/README.md).

## Operative photos

Photos are the player's own and stay on the phone: they are never committed or bundled. Add them
in Setup → Operative photos → **Import photos…** (files matched by operative name), by dropping
files into *Files → On My iPhone → Kill Team*, or per operative by tapping its portrait.
