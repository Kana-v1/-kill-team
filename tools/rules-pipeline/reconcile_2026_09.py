#!/usr/bin/env python3
"""One-off: reconcile both censuses with the official August '26 documents.

Every change here was checked against the rendered PDF pages (2026-09-26 audit):
  Plague Marines  eng_plague_marines_online_rules-wqjvit50wj-jvekzllwdl.pdf (Errata August '26)
  Angels of Death eng_26-08_killteam_angels_of_death_online_rules-...pdf     (Errata August '26)
  Core defaults   Kill Team Lite Rules (ploys 1CP; each ploy except Command Re-roll once per TP)
Kept as a record of what changed and why; safe to re-run (idempotent).
"""
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
PDF = {
    "plague_marines": ("eng_plague_marines_online_rules-wqjvit50wj-jvekzllwdl.pdf", "August '26"),
    "aod": ("eng_26-08_killteam_angels_of_death_online_rules-1rwlnicmkz-qjtykwlybg.pdf", "August '26"),
}
RULE_NAMES = [(" · 2nd Shoot costs +1 AP", ""), (" · Grenadier +1 Hit", ""), ("Rng ", "Range "),
              ("Piercing Crit 1", "Piercing Crits 1"), ("Heavy (Dash)", "Heavy (Dash only)"),
              ("Dev 3", "Devastating 3")]


def eff(d, i):
    return next(e for e in d["effects"] if e["id"] == i)


def op(d, i):
    return next(o for o in d["operatives"] if o["id"] == i)


def common(d, team):
    pdf, ver = PDF[team]
    src = f"official PDF, {ver}"
    d["meta"]["sourcePdf"] = pdf
    d["meta"]["rulesVersion"] = ver
    d["meta"]["unresolved"] = []
    for e in d["effects"]:
        if e["kind"] in ("strategy_ploy", "firefight_ploy") and e["id"] != "core.ff.command_reroll":
            e["once_per"] = "turning_point"  # Lite rules: each ploy once per TP
        v = [x for x in e.get("verify", []) if "CP cost" not in x]
        if v:
            e["verify"] = v
        else:
            e.pop("verify", None)
        if e["id"] != "core.ff.command_reroll":
            e["source"] = src
    for o in d["operatives"]:
        for w in o["weapons"]:
            for a, b in RULE_NAMES:
                w["rules"] = w["rules"].replace(a, b)
    for w in d.get("universalEquipment", []):
        for a, b in RULE_NAMES:
            w["rules"] = w["rules"].replace(a, b)


def plague_marines(d):
    d["meta"]["note"] = ("Transcribed from the official August '26 PDF (linked from the Kill Team downloads page) "
                         "and checked page by page against the rendered PDF. Ploy costs: 1CP (core rules; cards print none).")
    d["meta"]["sources"] = [f"official: {PDF['plague_marines'][0]}"]
    e = eff(d, "pm.strat.contagion")
    e["text"] = ("Subtract 2\" from the Move stat of an enemy operative and worsen the Hit stat of its weapons by 1 "
                 "(not cumulative with being injured) whenever either is true: it has one of your Poison tokens and is "
                 "visible to (or vice versa) and within 3\" of friendly Plague Marines; or it's visible to (or vice versa) "
                 "and within 3\" of a friendly Plague Marine Icon Bearer.")
    e["prompts"] = {"*": "Enemy −2\" Move and −1 Hit while poisoned within 3\" of a Plague Marine, or within 3\" of your Icon Bearer."}
    e = eff(d, "pm.strat.cloud_of_flies")
    e["text"] = ("Place one of your Cloud of Flies markers in the killzone. Whenever an operative is shooting a friendly "
                 "Plague Marine that's more than 3\" from it, if that friendly operative is wholly within 1\" of that marker, "
                 "it is obscured. Remove the marker in the Ready step of the next Strategy phase.")
    e["prompts"] = {"*": "A Plague Marine more than 3\" from its shooter and wholly within 1\" of your marker is obscured."}
    e = eff(d, "pm.ff.virulent_poison")
    e["text"] = ("Use during a friendly Plague Marine's activation or counteraction, before or after it performs an action. "
                 "One enemy operative within 3\" of, or visible to and within 7\" of, that operative gains one of your Poison "
                 "tokens (if it doesn't already have one).")
    e["prompts"] = {"*": "Before/after an action: an enemy within 3\", or visible and within 7\", gains a Poison token."}
    e = eff(d, "pm.ff.curse_of_rot")
    e["text"] = ("Use when a friendly Plague Marine is shooting against or fighting against an enemy operative within 3\" of it "
                 "(or within 7\" if that enemy has one of your Poison tokens), after your opponent rolls their attack or defence "
                 "dice. For each result of 3 they roll, inflict 1 damage on that enemy; that result cannot be retained as a "
                 "success and they cannot re-roll it.")
    e["prompts"] = {"*": "After they roll, each 3 deals 1 damage and can't be retained as a success or re-rolled. Enemy within 3\", 7\" if poisoned."}
    e = eff(d, "pm.eq.poison_vents")
    e["text"] = ("Whenever an enemy operative is activated within 3\" of a friendly Plague Marine: if it doesn't have one of "
                 "your Poison tokens, roll one D3 — on a 3 it gains one; if it has one of your Poison tokens, inflict D3 "
                 "damage on it (instead of 1).")
    e["prompts"] = {"*": "Enemy activating within 3\" of a Plague Marine: unpoisoned → roll D3, on 3 it gains a token; poisoned → D3 damage instead of 1."}
    e = eff(d, "pm.op.flail")
    e["text"] = ("Action (1AP): inflict D3+2 damage on each other operative both visible to and within 2\" of this operative. "
                 "Roll separately for each: if it's an enemy operative and the D3 result is a 3, it also gains one of your "
                 "Poison tokens (if it doesn't already have one). For action restrictions and the Astartes rule, this counts "
                 "as a Fight action. Cannot be performed while this operative has a Conceal order.")
    e["prompts"] = {"*": "1AP (counts as Fight): D3+2 damage to each other operative visible and within 2\"; on a D3 of 3, enemies also gain a Poison token. Not on Conceal."}


def aod(d):
    d["meta"]["note"] = ("Checked page by page against the official August '26 PDF (linked from the Kill Team downloads page). "
                         "Ploy costs: 1CP (core rules; cards print none).")
    d["meta"]["sources"] = [f"official: {PDF['aod'][0]}"]
    for w in op(d, "intercessor_sergeant")["weapons"]:
        if w["name"] == "Chainsword":
            w["hit"] = "3+"
    for w in op(d, "intercessor_gunner")["weapons"]:
        w["name"] = w["name"].replace("Aux grenade launcher", "Auxiliary grenade launcher")
    g = op(d, "assault_intercessor_grenadier")
    g["weapons"] = [w for w in g["weapons"] if w["name"] not in ("Frag grenade", "Krak grenade")]

    e = eff(d, "aod.rule.astartes")
    e["text"] = ("During each friendly operative's activation, it can perform either two Shoot actions or two Fight actions. "
                 "If two Shoot, a bolt weapon must be selected for at least one; if it's a bolt sniper rifle or heavy bolter, "
                 "1 additional AP must be spent for the second action if both use that weapon. Each friendly operative can "
                 "counteract regardless of its order.")
    e["prompts"]["activation.start"] = ("Take TWO Shoot or TWO Fight actions (one Shoot with a bolt weapon; a second shot "
                                        "with the same bolt sniper rifle or heavy bolter costs +1 AP).")
    e = eff(d, "aod.ff.shock_assault")
    e["text"] = ("Use when a friendly operative is performing the Fight action during an activation in which it performed the "
                 "Charge action, at the start of the Resolve Attack Dice step. Until the end of that action its melee weapon "
                 "has Shock, and the first time you strike during that sequence, inflict 1 additional damage (max 7).")
    e.pop("verify", None)
    e = eff(d, "aod.ff.wrath_of_vengeance")
    e["text"] = ("Use when a friendly operative is counteracting. It can perform an additional 1AP action for free during "
                 "that counteraction, but both actions must be different.")
    e = eff(d, "aod.eq.chapter_reliquaries")
    for k in ("window", "windows", "once_per"):
        e.pop(k, None)
    e.update({"alwaysOn": True, "duration": "battle",
              "text": "You can use the Wrath of Vengeance firefight ploy for 0CP if the specified friendly operative has an Engage order."})
    e.pop("verify", None)
    e = eff(d, "aod.op.optics")
    e["text"] = ("Action (1AP): until the start of this operative's next activation, whenever it's shooting, enemy operatives "
                 "cannot be obscured. Cannot be performed while within control range of an enemy operative.")
    e["prompts"] = {"*": "1AP: until its next activation, when this operative shoots, enemies can't be obscured. Not in enemy control range."}
    e.pop("verify", None)

    have = {x["id"] for x in d["effects"]}
    new = [
        {"id": "aod.op.camo_cloak", "name": "Camo Cloak", "kind": "operative_ability",
         "requiresOperative": "eliminator_sniper", "cost": {"cp": 0}, "alwaysOn": True, "duration": "battle",
         "text": ("Whenever an operative is shooting this operative, ignore the Saturate weapon rule. This operative has the "
                  "Stealthy chapter tactic; if you selected that chapter tactic, you can do both of its options (retain two "
                  "cover saves — one normal and one critical success)."),
         "remind_at": ["defence.shooting.before_roll"],
         "prompts": {"*": "When this operative is shot, ignore Saturate. It has Stealthy — and can use both Stealthy options if you picked it."}},
        {"id": "aod.op.grenadier", "name": "Grenadier", "kind": "operative_ability",
         "requiresOperative": "assault_intercessor_grenadier", "cost": {"cp": 0}, "alwaysOn": True, "duration": "battle",
         "text": ("This operative can use frag and krak grenades (see universal equipment). Doing so doesn't count towards any "
                  "limited uses you have. Whenever it's doing so, improve the Hit stat of that weapon by 1."),
         "remind_at": ["combat.shoot.before_roll"],
         "prompts": {"*": "Grenadier uses frag and krak grenades free of their limited uses, at +1 Hit."}},
    ]
    src = f"official PDF, {PDF['aod'][1]}"
    for n in new:
        if n["id"] not in have:
            n["source"] = src
            d["effects"].append(n)


for team, fn in (("plague_marines", plague_marines), ("aod", aod)):
    p = ROOT / f"data/teams/{team}.json"
    d = json.loads(p.read_text(encoding="utf-8"))
    common(d, team)
    fn(d)
    p.write_text(json.dumps(d, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    print(f"{team}: reconciled")
