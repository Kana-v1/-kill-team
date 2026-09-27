#!/usr/bin/env python3
"""Audit a team census against its official PDF text.

Reads data/teams/<team>.json and the pipeline's extractions
(data/extracted/<team>.txt for datacard rows, <team>.cols.txt for cards) and
writes data/extracted/<team>.audit.md: every place the census and the PDF
disagree, for a human to reconcile. It never edits the census.

Checks:
  - datacards: APL / Move / Save / Wounds and every weapon row, field by field
  - effects: each ploy / equipment / rule found by title; numbers (3", D3, 7+ ...)
    present in one text but not the other; how much of the census wording the
    PDF card actually contains
  - PDF cards the census doesn't have
  - open verify[] items and secondary-source entries

Usage: audit.py <team>
"""
import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SECONDARY = ("ktdash", "ktdojo", "wargamebuilder", "wahapedia")
CARD_HEADERS = ("STRATEGY PLOY", "FIREFIGHT PLOY", "FACTION EQUIPMENT", "FACTION RULE")
STOP = re.compile(r"^(=====|.*UPDATE LOG|ERRATA|PREVIOUS ERRATAS|.*(STRATEGY|FIREFIGHT)\s+PLOY$|.*FACTION\s+(EQUIPMENT|RULE)$)")


def norm(s):
    s = s.replace("’", "'").replace("″", '"').replace("–", "-").replace("—", "-").replace("−", "-")
    return re.sub(r"\s+", " ", s).strip()


def words(s):
    return set(re.findall(r"[a-z]+", norm(s).lower())) - {"the", "a", "an", "of", "to", "and", "or", "it", "its", "is", "that", "this", "if", "in", "on", "be", "for", "with", "you", "your"}


def numbers(s):
    s = norm(s)
    return set(re.findall(r'\d+D\d+|D\d+|\d+\+|\d+"|\d+CP|\d+AP', s))


def weapon_rules(s):
    toks = [t.strip().rstrip("*").lower() for t in norm(s).replace("·", ",").split(",")]
    return {t for t in toks if t and t not in ("-", "—")}


def find_datacards(txt):
    """Map collapsed upper-case operative header -> (stats, weapons, block text)."""
    lines = txt.splitlines()
    cards = {}
    for i, ln in enumerate(lines):
        if "WOUNDS" not in ln or "APL" not in ln:
            continue
        name = norm(ln.split("APL")[0]).upper()
        stats, weapons, block = None, [], [ln]
        for nxt in lines[i + 1:i + 30]:
            if "WOUNDS" in nxt and "APL" in nxt:
                break
            block.append(nxt)
            n = norm(nxt)
            m = re.search(r'(\d)\s+(\d+")\s+(\d\+)\s+(\d+)$', n)
            if stats is None and m:
                stats = {"apl": int(m.group(1)), "move": m.group(2), "save": m.group(3), "wounds": int(m.group(4))}
                continue
            w = re.match(r'^(.+?)\s+(\d+)\s+(\d\+)\s+(\d+/\d+)\s*(.*)$', n)
            if not w and weapons and weapons[-1]["rules"].endswith(",") and n and len(n) < 60:
                weapons[-1]["rules"] += " " + n  # rules wrapped onto the next line
                continue
            if w and not n.startswith("NAME"):
                weapons.append({"name": w.group(1).strip(), "atk": int(w.group(2)), "hit": w.group(3),
                                "dmg": w.group(4), "rules": w.group(5).strip()})
        cards.setdefault(name, {"stats": stats, "weapons": weapons, "block": "\n".join(block)})
    return cards


def card_body(cols, title):
    """The text of the card titled `title` in reading order, or None."""
    lines = cols.splitlines()
    want = norm(title).upper()
    for i, ln in enumerate(lines):
        if norm(ln).upper() == want:
            body = []
            for nxt in lines[i + 1:i + 40]:
                s = norm(nxt)
                if body and (STOP.match(s) or re.fullmatch(r"(PLAGUE MARINES?|ANGELS? OF DEATH)", s)):
                    break
                body.append(nxt)
            return norm(" ".join(body))
    return None


def pdf_card_titles(cols):
    """Card titles: the line after a card-type header line."""
    lines = [norm(l) for l in cols.splitlines()]
    out = []
    for i, ln in enumerate(lines[:-1]):
        if any(ln.endswith(h) for h in CARD_HEADERS):
            t = lines[i + 1]
            if t and t.isupper() and len(t) < 40:
                out.append(t)
    return out


def main():
    team = sys.argv[1]
    census = json.load(open(ROOT / f"data/teams/{team}.json", encoding="utf-8"))
    txt = (ROOT / f"data/extracted/{team}.txt").read_text(encoding="utf-8")
    cols = (ROOT / f"data/extracted/{team}.cols.txt").read_text(encoding="utf-8")
    out = [f"# Rules audit — {census['meta']['team']}", "",
           f"Census: `data/teams/{team}.json` · PDF text: `data/extracted/{team}.txt`", ""]
    issues = 0

    # ---- datacards
    out += ["## Datacards", ""]
    cards = find_datacards(txt)
    for op in census["operatives"]:
        key = norm(op["name"]).upper()
        card = cards.get(key) or next((c for k, c in cards.items() if key.endswith(k) or k.endswith(key)), None)
        if not card:
            out.append(f"- **{op['name']}** — datacard not found in PDF text")
            issues += 1
            continue
        probs = []
        if card["stats"]:
            for f in ("apl", "move", "save", "wounds"):
                cv, pv = op["stats"][f], card["stats"][f]
                if norm(str(cv)) != norm(str(pv)):
                    probs.append(f"{f.upper()}: census `{cv}` vs PDF `{pv}`")
        else:
            probs.append("stat line not parsed from PDF")
        pdfw = {norm(w["name"]).lower(): w for w in card["weapons"]}
        cw = {norm(w["name"]).lower(): w for w in op["weapons"]}
        for n in sorted(set(cw) - set(pdfw)):
            probs.append(f"weapon `{cw[n]['name']}` is in the census but NOT on the PDF datacard")
        for n in sorted(set(pdfw) - set(cw)):
            probs.append(f"weapon `{pdfw[n]['name']}` is on the PDF datacard but missing from the census")
        for n in sorted(set(cw) & set(pdfw)):
            a, b = cw[n], pdfw[n]
            for f in ("atk", "hit", "dmg"):
                if norm(str(a[f])) != norm(str(b[f])):
                    probs.append(f"`{a['name']}` {f.upper()}: census `{a[f]}` vs PDF `{b[f]}`")
            ra, rb = weapon_rules(a["rules"]), weapon_rules(b["rules"])
            if ra != rb:
                probs.append(f"`{a['name']}` rules: census `{a['rules']}` vs PDF `{b['rules']}`")
        issues += len(probs)
        out.append(f"- **{op['name']}** — " + ("OK" if not probs else ""))
        out += [f"  - {p}" for p in probs]
    out.append("")

    # ---- effects
    out += ["## Ploys, equipment, faction rules", ""]
    seen = set()
    for e in census["effects"]:
        if e["kind"] not in ("strategy_ploy", "firefight_ploy", "equipment", "faction_rule"):
            continue
        body = card_body(cols, e["name"])
        seen.add(norm(e["name"]).upper())
        if body is None:
            if not e.get("universal"):
                out.append(f"- **{e['name']}** ({e['kind']}) — title not found in PDF")
                issues += 1
            continue
        probs = []
        extra_pdf = numbers(body) - numbers(e["text"])
        extra_cen = numbers(e["text"]) - numbers(body)
        if extra_cen:
            probs.append(f"numbers in census, not on the card: {', '.join(sorted(extra_cen))}")
        if extra_pdf:
            probs.append(f"numbers on the card, not in census: {', '.join(sorted(extra_pdf))}")
        cw = words(e["text"])
        recall = len(cw & words(body)) / max(1, len(cw))
        if recall < 0.85:
            probs.append(f"only {recall:.0%} of the census wording appears on the card")
        only = sorted(cw - words(body))
        if len(only) >= 2:
            probs.append(f"words only in the census (check for changed rules): {', '.join(only)}")
        issues += len(probs)
        out.append(f"- **{e['name']}** ({e['kind']}) — " + ("OK" if not probs else ""))
        if probs:
            out += [f"  - {p}" for p in probs]
            out += [f"  - census: {e['text']}", f"  - PDF: {body}"]
    out.append("")

    missing = [t for t in pdf_card_titles(cols) if t not in seen]
    out += ["## On the PDF, not in the census", ""]
    out += [f"- {t}" for t in missing] or ["- none"]
    issues += len(missing)
    out.append("")

    # ---- open questions
    out += ["## Open verify[] items", ""]
    for e in census["effects"]:
        for v in e.get("verify", []):
            out.append(f"- **{e['name']}**: {v}")
    out += ["", "## Entries sourced from secondary sites", ""]
    sec = [e for e in census["effects"] if any(s in e.get("source", "") for s in SECONDARY)]
    out += [f"- **{e['name']}** — `{e['source']}`" for e in sec] or ["- none"]
    out += ["", f"_{issues} discrepancies flagged._", ""]

    dest = ROOT / f"data/extracted/{team}.audit.md"
    dest.write_text("\n".join(out), encoding="utf-8")
    print(f"{team}: {issues} discrepancies → {dest.relative_to(ROOT)}")


if __name__ == "__main__":
    main()
