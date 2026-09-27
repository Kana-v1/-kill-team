"""Draws the app icon (an original design: a d6 in an objective-marker hex).
Run with a Python that has PyMuPDF: python tools/app_icon.py"""
import math
import pymupdf as fitz
from pathlib import Path
S = 1024
doc = fitz.open(); page = doc.new_page(width=S, height=S)
def rgb(h): return tuple(((h >> k) & 255) / 255 for k in (16, 8, 0))
sh = page.new_shape()
# background: dark vertical gradient
top, bot = rgb(0x2A2A30), rgb(0x050506)
for i in range(128):
    t = i / 127
    c = tuple(a + (b - a) * t for a, b in zip(top, bot))
    sh.draw_rect(fitz.Rect(0, i * S / 128, S, (i + 1) * S / 128 + 1)); sh.finish(fill=c, color=None, width=0)
cx, cy = S / 2, S / 2
def hexagon(r, rot=0):
    return [fitz.Point(cx + r * math.cos(math.radians(60 * k + rot)), cy + r * math.sin(math.radians(60 * k + rot))) for k in range(7)]
gold, amber, white = rgb(0xE8C27A), rgb(0xFF9F0A), rgb(0xF2F2F7)
# objective-marker hexagon: soft glow, then the ring
for r, w, o in [(372, 90, 0.10), (372, 64, 0.18)]:
    sh.draw_polyline(hexagon(r, 30)); sh.finish(color=gold, width=w, closePath=True, lineJoin=1, stroke_opacity=o)
sh.draw_polyline(hexagon(372, 30)); sh.finish(color=gold, width=40, closePath=True, lineJoin=1)
sh.draw_polyline(hexagon(300, 30)); sh.finish(fill=rgb(0x141416), color=None, closePath=True)
# a d6, tilted, rolled a 6 — one pip lit amber (the critical)
rot = math.radians(-14)
def P(x, y):
    return fitz.Point(cx + x * math.cos(rot) - y * math.sin(rot), cy + x * math.sin(rot) + y * math.cos(rot))
h, rad = 190, 58
pts = []
for (qx, qy, a0) in [(h - rad, -h + rad, -90), (h - rad, h - rad, 0), (-h + rad, h - rad, 90), (-h + rad, -h + rad, 180)]:
    for k in range(13):
        a = math.radians(a0 + 90 * k / 12)
        pts.append(P(qx + rad * math.cos(a), qy + rad * math.sin(a)))
pts.append(pts[0])
sh.draw_polyline([fitz.Point(p.x + 14, p.y + 22) for p in pts]); sh.finish(fill=(0, 0, 0), color=None, closePath=True, fill_opacity=0.45)
sh.draw_polyline(pts); sh.finish(fill=white, color=None, closePath=True)
pip = rgb(0x1C1C1E)
for i, (x, y) in enumerate([(-92, -100), (-92, 0), (-92, 100), (92, -100), (92, 0), (92, 100)]):
    sh.draw_circle(P(x, y), 34); sh.finish(fill=amber if i == 3 else pip, color=None)
sh.commit()
page.get_pixmap(matrix=fitz.Matrix(1, 1), alpha=False).save(Path(__file__).resolve().parents[1] / "ios/KillTeam/Assets.xcassets/AppIcon.appiconset/AppIcon.png")
