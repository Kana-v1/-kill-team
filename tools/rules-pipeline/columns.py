#!/usr/bin/env python3
"""Rebuild reading-order text from lit's JSON output.

lit's plain-text mode projects each page onto a character grid, so two cards
printed side by side come out interleaved line by line ("CONTAGION   LUMBERING
DEATH", then both first lines, ...). This uses the per-item x/y coordinates in
lit's JSON to split two-column pages at the gutter and emit each column whole.

Datacard pages (they carry the APL/MOVE/SAVE/WOUNDS header) mix both: the
stat header, weapon table and keyword footer are rows that must stay whole,
while the ability text below the table runs in two columns. Table lines are
kept intact; everything else is split at the gutter like any other page.

Usage: columns.py data/extracted/<team>.json > data/extracted/<team>.cols.txt
"""
import json
import re
import sys


def lines_of(items, tol=3.0):
    """Group items into visual lines by y, top to bottom."""
    lines = []
    for it in sorted(items, key=lambda i: (i["y"], i["x"])):
        if lines and abs(lines[-1]["y"] - it["y"]) <= tol:
            lines[-1]["items"].append(it)
        else:
            lines.append({"y": it["y"], "h": it.get("height", 10), "items": [it]})
    for ln in lines:
        ln["items"].sort(key=lambda i: i["x"])
    return lines


def join(items):
    return " ".join(i["text"].strip() for i in items if i["text"].strip())


WEAPON_ROW = re.compile(r".+\s\d+\s+\d\+\s+\d+/\d+")


def is_table_line(text):
    """A datacard line that must stay whole: header, stats, weapon row, footer."""
    t = text.strip()
    return bool(
        re.search(r"\bAPL\b.*\bMOVE\b|\bNAME\b.*\bATK\b", t)
        or WEAPON_ROW.match(t)
        or re.fullmatch(r'[\d\s"”+]+', t)
        or ("," in t and t.upper() == t and re.search(r"\d+$", t))
        or "RULES CONTINUE ON OTHER SIDE" in t
    )


def page_text(page):
    items = [i for i in page["textItems"] if i["text"].strip()]
    lines = lines_of(items)
    datacard = any("WOUNDS" in i["text"] for i in items)

    mid = page["width"] / 2
    out, left, right = [], [], []
    prev_y = None

    def flush():
        out.extend(left)
        if left and right:
            out.append("")
        out.extend(right)
        left.clear()
        right.clear()

    for ln in lines:
        gap = (ln["y"] - prev_y) if prev_y is not None else 0
        prev_y = ln["y"]
        L = [i for i in ln["items"] if i["x"] + i["width"] <= mid + 2]
        R = [i for i in ln["items"] if i["x"] >= mid - 2]
        spanning = [i for i in ln["items"] if i not in L and i not in R]
        if spanning or (datacard and is_table_line(join(ln["items"]))):
            flush()
            out.append(join(ln["items"]))
            continue
        # a big vertical gap starts a new block of cards in both columns
        if gap > 2.2 * max(ln["h"], 8):
            if left:
                left.append("")
            if right:
                right.append("")
        if L:
            left.append(join(L))
        if R:
            right.append(join(R))
    flush()
    return "\n".join(out)


def main():
    doc = json.load(open(sys.argv[1], encoding="utf-8"))
    for page in doc["pages"]:
        print(f"===== page {page['page']} =====")
        print(page_text(page))


if __name__ == "__main__":
    main()
