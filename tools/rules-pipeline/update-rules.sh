#!/usr/bin/env bash
#
# update-rules.sh — fetch a team's official rules PDF and extract authoritative
# text from it with `lit` (liteparse), staging the result for a card-verification
# pass. It never writes the census (data/teams/<id>.json) directly: per the
# project's sourcing rules, only human-verified card text lands there. This
# pipeline just gets you trustworthy source text to verify against, replacing the
# PDF-summarising WebFetch that used to fabricate rules.
#
# Usage:
#   tools/rules-pipeline/update-rules.sh <teamId>            # url from sources.json
#   tools/rules-pipeline/update-rules.sh <teamId> <pdfUrl>   # override / add a url
#   tools/rules-pipeline/update-rules.sh --all               # every team in sources.json
#   tools/rules-pipeline/update-rules.sh --sync [...]        # first refresh sources.json from the downloads page
#
# Outputs (relative to repo root):
#   data/pdf/<teamId>.pdf              raw PDF
#   data/extracted/<teamId>.txt        extracted text (for reading/diffing)
#   data/extracted/<teamId>.json       extracted per-page text (for tooling)
#
# Env:
#   LIT           override the lit binary (default: `lit` on PATH)
#   LIT_OCR=1     enable lit's OCR (needs an OCR server; see --ocr-server-url)
#   OCR_SERVER    OCR server url passed to lit when LIT_OCR=1
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SOURCES="$REPO_ROOT/tools/rules-pipeline/sources.json"
PDF_DIR="$REPO_ROOT/data/pdf"
OUT_DIR="$REPO_ROOT/data/extracted"
LIT="${LIT:-lit}"

die() { echo "error: $*" >&2; exit 1; }
have() { command -v "$1" >/dev/null 2>&1; }

have python3 || die "python3 is required to read sources.json"
have "$LIT" || die "lit (liteparse) not found on PATH; install it or set \$LIT"

# url_for <teamId> — print the configured PDF url, or empty.
url_for() {
  python3 - "$SOURCES" "$1" <<'PY'
import json, sys
with open(sys.argv[1]) as f:
    data = json.load(f)
t = sys.argv[2]
print((data.get("teams", {}).get(t) or data.get("core", {}).get(t) or {}).get("url", ""))
PY
}

# all_teams — print every configured teamId, one per line.
all_teams() {
  python3 - "$SOURCES" <<'PY'
import json, sys
with open(sys.argv[1]) as f:
    data = json.load(f)
print("\n".join(data.get("teams", {}).keys()))
PY
}

# lit runs as a Windows binary under WSL: it rejects UNC paths, so stage the PDF
# on a real Windows drive and run it from there with bare filenames.
lit_is_windows() { case "$(command -v "$LIT")" in /mnt/*) return 0;; *) return 1;; esac }

win_temp_wsl() {
  local t
  # </dev/null: cmd.exe otherwise swallows the caller's stdin (the --all team list)
  t="$(cmd.exe /c 'echo %TEMP%' </dev/null 2>/dev/null | tr -d '\r\n')" || return 0
  [ -n "$t" ] && wslpath -u "$t" 2>/dev/null || true
}

lit_flags() {
  if [ "${LIT_OCR:-0}" = "1" ]; then
    [ -n "${OCR_SERVER:-}" ] && printf -- '--ocr-server-url\n%s\n' "$OCR_SERVER"
  else
    printf -- '--no-ocr\n'
  fi
}

# extract <pdf> <out_txt> <out_json>
extract() {
  local pdf="$1" out_txt="$2" out_json="$3"
  mapfile -t flags < <(lit_flags)
  if lit_is_windows; then
    local wt; wt="$(win_temp_wsl)"
    [ -n "$wt" ] || die "could not locate a Windows temp dir for the Windows lit binary"
    local stage="$wt/kt-rules"; mkdir -p "$stage"
    local base="_kt_$(basename "$pdf")" stem
    stem="${base%.pdf}"
    cp "$pdf" "$stage/$base"
    ( cd "$stage" && "$LIT" parse "$base" --format text "${flags[@]}" -q -o "$stem.txt" \
                  && "$LIT" parse "$base" --format json "${flags[@]}" -q -o "$stem.json" )
    cp "$stage/$stem.txt" "$out_txt"
    cp "$stage/$stem.json" "$out_json"
    rm -f "$stage/$base" "$stage/$stem.txt" "$stage/$stem.json"
  else
    "$LIT" parse "$pdf" --format text "${flags[@]}" -q -o "$out_txt"
    "$LIT" parse "$pdf" --format json "${flags[@]}" -q -o "$out_json"
  fi
}

process() {
  local team="$1" url="${2:-}"
  [ -n "$url" ] || url="$(url_for "$team")"
  [ -n "$url" ] || die "no url for team '$team' — add it to sources.json or pass it as the 2nd argument"

  mkdir -p "$PDF_DIR" "$OUT_DIR"
  local pdf="$PDF_DIR/$team.pdf"
  echo "→ $team: downloading"
  curl -fsSL "$url" -o "$pdf" || die "download failed for $url"
  echo "  $(du -h "$pdf" | cut -f1) $pdf"

  echo "  extracting with lit"
  extract "$pdf" "$OUT_DIR/$team.txt" "$OUT_DIR/$team.json"
  echo "  wrote $OUT_DIR/$team.txt ($(wc -l < "$OUT_DIR/$team.txt") lines) and $team.json"
  python3 "$REPO_ROOT/tools/rules-pipeline/columns.py" "$OUT_DIR/$team.json" > "$OUT_DIR/$team.cols.txt"
  echo "  wrote $OUT_DIR/$team.cols.txt (column-aware reading order)"

  # Flag census entries still carrying unverified assumptions.
  local census="$REPO_ROOT/data/teams/$team.json"
  if [ -f "$census" ]; then
    python3 - "$census" <<'PY'
import json, sys
with open(sys.argv[1]) as f:
    d = json.load(f)
flagged = [e["id"] for e in d.get("effects", []) if e.get("verify")]
if flagged:
    print(f"  {len(flagged)} census entries still carry verify[] assumptions to reconcile:")
    for i in flagged:
        print(f"    · {i}")
PY
  fi
}

main() {
  [ "$#" -ge 1 ] || die "usage: update-rules.sh [--sync] <teamId> [pdfUrl] | --all"
  if [ "$1" = "--sync" ]; then
    python3 "$REPO_ROOT/tools/rules-pipeline/sync_sources.py" || die "sync failed"
    shift
    [ "$#" -ge 1 ] || exit 0
  fi
  if [ "$1" = "--all" ]; then
    local teams
    mapfile -t teams < <(all_teams)
    for team in "${teams[@]}"; do
      [ -n "$team" ] && process "$team" </dev/null
    done
  else
    process "$1" "${2:-}"
  fi
  echo
  echo "Done. Next: read data/extracted/<team>.txt against the datacards, then update"
  echo "data/teams/<team>.json by hand. Only card-verified text belongs in the census."
}

main "$@"
