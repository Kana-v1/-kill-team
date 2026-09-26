#!/usr/bin/env python3
"""Add the iOS engine's model fields to both censuses (idempotent).

Per effect / chapter tactic:
  when          activation | attack | defence | any — which group it shows in
  appliesTo     team | self | weapons — who it applies to (self = its requiresOperative)
  weaponMatch   for appliesTo=weapons: case-insensitive substrings of weapon names
  hint          one action-first line for mid-game (the old per-trigger prompts)
  grantsWeaponRules  [{match, rules, condition?}] — weapon-row notes; match is
                "*", "ranged", "melee" or weapon-name substrings
Per team: glossary of faction terms (definitions copied from the verified census text).

Values come from the rule text as verified against the August '26 PDFs.
"""
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
T, S, W = "team", "self", "weapons"

AOD = {
    "core.ff.command_reroll": ("any", T, "After rolling your attack or defence dice, re-roll one of them."),
    "aod.rule.astartes": ("activation", T, "Two Shoot or two Fight actions (one Shoot with a bolt weapon; a second shot with the same bolt sniper rifle or heavy bolter costs +1 AP). Can counteract regardless of order."),
    "aod.strat.combat_doctrine": ("attack", T, "Your weapons have Balanced while your doctrine's condition is met."),
    "aod.strat.and_they_shall_know_no_fear": ("any", T, "Injured operatives use their normal stats this turning point (weapons included)."),
    "aod.strat.indomitus": ("defence", T, "When shot: two or more fails? Discard one to retain another as a normal success."),
    "aod.strat.adaptive_tactics": ("any", T, "Swap your secondary chapter tactic until the end of the turning point."),
    "aod.ff.shock_assault": ("attack", T, "Fighting after a Charge: Shock, and +1 damage on your first strike (max 7)."),
    "aod.ff.transhuman_physiology": ("defence", T, "When shot, in the Roll Defence Dice step: retain one normal success as a critical."),
    "aod.ff.wrath_of_vengeance": ("activation", T, "When counteracting: one extra free 1AP action (both actions different)."),
    "aod.ff.adjust_doctrine": ("activation", T, "Before or after an action: change the Combat Doctrine you selected this turning point."),
    "aod.eq.purity_seals": ("attack", T, "Once per turning point, shooting, fighting or retaliating: two or more fails? Discard one to retain another as a normal success."),
    "aod.eq.auspex": ("attack", T, "Once per turning point when shooting: enemies within 8\" of this operative can't be obscured until its activation ends."),
    "aod.eq.tilting_shields": ("defence", T, "Once per turning point, fighting or retaliating, after they roll: they can't retain results below 6 as criticals this sequence."),
    "aod.eq.chapter_reliquaries": ("activation", T, "Wrath of Vengeance is free if the operative has an Engage order."),
    "aod.op.heroic_leader": ("any", T, "Once per turning point: a firefight ploy free for the Captain (not Command Re-roll), Combat Doctrine on activation, or Adjust Doctrine free."),
    "aod.op.iron_halo": ("defence", S, "Once per battle: ignore one attack die's Normal Dmg on the Captain."),
    "aod.op.optics": ("activation", S, "1AP: until its next activation, when this operative shoots, enemies can't be obscured. Not in enemy control range."),
    "aod.op.doctrine_warfare_asltsgt": ("any", T, "Once per battle each: Combat Doctrine is free when you pick Assault or Tactical (Sergeant in the killzone)."),
    "aod.op.doctrine_warfare_intsgt": ("any", T, "Once per battle each: Combat Doctrine is free when you pick Devastator or Tactical (Sergeant in the killzone)."),
    "aod.op.camo_cloak": ("defence", S, "When shot, ignore Saturate. Has Stealthy — and can use both Stealthy options if you picked it."),
    "aod.op.grenadier": ("attack", S, "Uses frag and krak grenades free of their limited uses, at +1 Hit."),
}
AOD_GRANTS = {
    "aod.ff.shock_assault": [{"match": ["melee"], "rules": ["Shock"], "condition": "fighting after a Charge"}],
}
AOD_OPTION_GRANTS = {
    "devastator": [{"match": ["ranged"], "rules": ["Balanced"], "condition": "target more than 6\" away"}],
    "tactical": [{"match": ["ranged"], "rules": ["Balanced"], "condition": "target within 6\""}],
    "assault": [{"match": ["melee"], "rules": ["Balanced"], "condition": "fighting or retaliating"}],
}
AOD_TACTICS = {
    "aggressive": ("attack", {"match": ["melee"], "rules": ["Rending"]}),
    "dueller": ("attack", None),
    "resolute": ("any", None),
    "stealthy": ("defence", None),
    "mobile": ("activation", None),
    "hardy": ("defence", None),
    "sharpshooter": ("attack", {"match": ["bolt"], "rules": ["Accurate 1", "Severe"],
                                "condition": "if it hasn't Charged, Fallen Back or Repositioned this activation"}),
    "siege_specialist": ("attack", {"match": ["ranged"], "rules": ["Saturate"]}),
}

PM = {
    "core.ff.command_reroll": ("any", T, "After rolling your attack or defence dice, re-roll one of them."),
    "pm.rule.astartes": ("activation", T, "Two Shoot or two Fight actions (one Shoot with a bolt pistol, boltgun or Psychic weapon; not the same Psychic ranged weapon twice). Can counteract regardless of order."),
    "pm.rule.poison": ("attack", T, "Poison weapon deals damage → the target (not a Plague Marine) gains a Poison token. A poisoned enemy takes 1 damage when it activates."),
    "pm.rule.disgustingly_resilient": ("defence", T, "Took 3+ damage on one die? Roll a D6 — on 4+, subtract 1 from that damage."),
    "pm.strat.contagion": ("defence", T, "Enemy −2\" Move and −1 Hit while poisoned within 3\" of a Plague Marine, or within 3\" of your Icon Bearer."),
    "pm.strat.lumbering_death": ("attack", T, "Ceaseless when shooting or fighting after moving ≤3\" this activation, or when retaliating."),
    "pm.strat.cloud_of_flies": ("defence", T, "A Plague Marine more than 3\" from its shooter and wholly within 1\" of your marker is obscured."),
    "pm.strat.nurglings": ("any", T, "One enemy within 3\" of a Plague Marine (or poisoned within 7\"): −1 APL until its next activation ends."),
    "pm.ff.virulent_poison": ("activation", T, "Before or after an action: an enemy within 3\", or visible and within 7\", gains a Poison token."),
    "pm.ff.poisonous_demise": ("any", T, "When a Plague Marine is incapacitated: enemies within 3\" gain a Poison token; already-poisoned ones take 1 damage instead."),
    "pm.ff.sickening_resilience": ("defence", T, "Until this activation ends, Disgustingly Resilient subtracts 1 automatically (min 2) — no roll."),
    "pm.ff.curse_of_rot": ("attack", T, "After they roll, each 3 deals 1 damage and can't be retained as a success or re-rolled. Enemy within 3\", 7\" if poisoned."),
    "pm.eq.plague_bells": ("any", T, "Ignore injured-stat changes on your Plague Marines (weapons included)."),
    "pm.eq.plague_rounds": ("attack", W, "Boltguns and bolt pistols have Poison and Severe."),
    "pm.eq.blight_grenades": ("attack", T, "Blight grenade: ATK 4 · Hit 4+ · Dmg 2/4 · Range 6\", Blast 2\", Saturate, Severe, Poison. Twice per battle."),
    "pm.eq.poison_vents": ("any", T, "Enemy activating within 3\" of a Plague Marine: unpoisoned → roll D3, on 3 it gains a token; poisoned → D3 damage instead of 1."),
    "pm.op.grandfathers_blessing": ("any", S, "A poisoned enemy loses wounds within 7\" of the Champion? It regains that many (max 3 per turning point)."),
    "pm.op.icon_bearer": ("any", S, "Counts as 1 higher APL when determining marker control."),
    "pm.op.icon_of_contagion": ("any", S, "Contagion is free while the Icon Bearer is in your opponent's territory."),
    "pm.op.repulsive_fortitude": ("defence", S, "When shot, defence dice of 5+ are critical successes."),
    "pm.op.grenadier": ("attack", S, "Uses blight and krak grenades free of their limited uses, at +1 Hit; blight grenades gain Toxic."),
    "pm.op.poisonous_miasma": ("activation", S, "1AP Psychic: an enemy within 7\" gains a Poison token; if already poisoned, 3 damage instead. Not in enemy control range."),
    "pm.op.putrescent_vitality": ("activation", S, "1AP Psychic: a friendly operative within 3\" rolls 2D6 — a 7 regains 7 wounds, else the highest D6. Once per turning point; not in enemy control range."),
    "pm.op.flail": ("attack", S, "1AP (counts as Fight): D3+2 damage to each other operative visible and within 2\"; on a D3 of 3, enemies also gain a Poison token. Not on Conceal."),
}
PM_WEAPON_MATCH = {"pm.eq.plague_rounds": ["boltgun", "bolt pistol"]}
PM_GRANTS = {
    "pm.strat.lumbering_death": [{"match": ["*"], "rules": ["Ceaseless"],
                                  "condition": "if it hasn't moved more than 3\" this activation, or when retaliating"}],
    "pm.eq.plague_rounds": [{"match": ["boltgun", "bolt pistol"], "rules": ["Poison", "Severe"]}],
}
PM_GLOSSARY = {
    "Poison": {"kind": "weapon rule · Plague Marines",
               "def": "In the Resolve Attack Dice step, if you inflict damage with any successes, the operative this weapon is being used against (excluding friendly Plague Marines) gains one of your Poison tokens (if it doesn't already have one). Whenever an operative that has one of your Poison tokens is activated, inflict 1 damage on it."},
    "Poison token": {"kind": "token · Plague Marines",
                     "def": "An enemy operative with one of your Poison tokens takes 1 damage whenever it's activated. Many Plague Marine rules get stronger against poisoned enemies."},
    "Toxic": {"kind": "weapon rule · Plague Marines",
              "def": "Whenever this operative is using this weapon against an enemy operative that had one of your Poison tokens at the start of that action, add 1 to both Dmg stats of this weapon."},
}

VOCAB_ADD = {
    "phases": ["strategy", "firefight"],
    "when": ["activation", "attack", "defence", "any"],
    "appliesTo": ["team", "self", "weapons"],
}


def apply(team, table, grants, weapon_match, option_grants=None, tactics=None, glossary=None):
    p = ROOT / f"data/teams/{team}.json"
    d = json.loads(p.read_text(encoding="utf-8"))
    d["vocab"].update(VOCAB_ADD)
    ids = {e["id"] for e in d["effects"]}
    missing = ids - set(table)
    extra = set(table) - ids
    assert not missing and not extra, f"{team}: table mismatch missing={missing} extra={extra}"
    for e in d["effects"]:
        when, applies, hint = table[e["id"]]
        e["when"], e["appliesTo"], e["hint"] = when, applies, hint
        if applies == W:
            e["weaponMatch"] = weapon_match[e["id"]]
        if e["id"] in grants:
            e["grantsWeaponRules"] = grants[e["id"]]
        for o in e.get("options", []):
            if option_grants and o["id"] in option_grants:
                o["grantsWeaponRules"] = option_grants[o["id"]]
            o["hint"] = o.get("prompt", "")
    for t in d.get("chapterTactics", []):
        when, g = tactics[t["id"]]
        t["when"], t["hint"] = when, t.get("prompt", t["text"])
        if g:
            t["grantsWeaponRules"] = [g]
    if glossary:
        d["glossary"] = glossary
    p.write_text(json.dumps(d, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    print(f"{team}: {len(d['effects'])} effects, {len(d.get('chapterTactics', []))} tactics annotated")


apply("aod", AOD, AOD_GRANTS, {}, AOD_OPTION_GRANTS, AOD_TACTICS)
apply("plague_marines", PM, PM_GRANTS, PM_WEAPON_MATCH, glossary=PM_GLOSSARY)
