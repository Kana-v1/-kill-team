import type { Operative, View } from "../types";
import { Glyph } from "../icons";

interface Props {
  operatives: Operative[];
  view: View;
  onSelectOp: (inst: string) => void;
  onToggleDown: (inst: string) => void;
}

const typeOf = (inst: string) => inst.split("#")[0];

function shortName(name: string): string {
  return name
    .replace("Assault Intercessor", "Asslt Int")
    .replace("Heavy Intercessor", "Hvy Int")
    .replace("Intercessor", "Int")
    .replace("Eliminator", "Elim")
    .replace("Space Marine", "SM")
    .replace("Malignant Plaguecaster", "Plaguecaster")
    .replace("Plague Marine ", "");
}

// The datacard panel: roster chips plus the selected operative's stats and the
// weapons relevant to the current combat context. Pure presentation — the
// backend already told us the roster, selection and context.
export function StatPanel({ operatives, view, onSelectOp, onToggleDown }: Props) {
  const opById = (inst: string) => operatives.find((o) => o.id === typeOf(inst));
  const countOf = (inst: string) => view.roster.filter((x) => typeOf(x) === typeOf(inst)).length;

  const chips = view.roster.map((inst) => {
    const op = opById(inst);
    if (!op) return null;
    let short = shortName(op.name);
    if (countOf(inst) > 1) short += " " + inst.split("#")[1];
    return (
      <button
        key={inst}
        className={`opchip ${view.dead[inst] ? "down" : ""}`}
        aria-pressed={inst === view.op}
        onClick={() => onSelectOp(inst)}
      >
        <Glyph op={op} />
        {short}
      </button>
    );
  });

  const o = opById(view.op);
  if (!o) {
    return (
      <div className="stats">
        <div className="oprow">{chips}</div>
      </div>
    );
  }

  const melee = view.ctx === "combat.fight" || view.ctx === "combat.retaliate";
  const ranged = view.ctx === "combat.shoot";
  let weapons = o.weapons;
  if (melee) weapons = o.weapons.filter((w) => w.type === "melee");
  if (ranged) weapons = o.weapons.filter((w) => w.type === "ranged");

  const dup = countOf(view.op) > 1 ? " " + view.op.split("#")[1] : "";
  const hitHead = melee ? "WS" : ranged ? "BS" : "Hit";
  const weaponHead = melee ? "Melee" : ranged ? "Ranged" : "Weapon";

  return (
    <div className="stats">
      <div className="oprow">{chips}</div>
      <div className="ophead">
        <span className="badge">
          <Glyph op={o} />
        </span>
        <span>
          <span className="who">
            {o.name}
            {dup}
          </span>
          <br />
          <span className="role">
            {o.role || "Operative"}
            {view.dead[view.op] ? " · incapacitated" : ""}
          </span>
        </span>
      </div>
      <div className="statline">
        <div>
          <span className="v">{o.stats.apl}</span>
          <span className="k">APL</span>
        </div>
        <div>
          <span className="v">{o.stats.move}</span>
          <span className="k">Move</span>
        </div>
        <div>
          <span className="v">{o.stats.save}</span>
          <span className="k">Save</span>
        </div>
        <div>
          <span className="v">{o.stats.wounds}</span>
          <span className="k">Wounds</span>
        </div>
      </div>
      <table className="wtab">
        <thead>
          <tr>
            <th>{weaponHead}</th>
            <th className="n">A</th>
            <th className="n">{hitHead}</th>
            <th className="n">D</th>
          </tr>
        </thead>
        <tbody>
          {weapons.map((w, i) => (
            <tr key={i}>
              <td>
                {w.name}
                <span className="wr">{w.rules}</span>
              </td>
              <td className="n">{w.atk}</td>
              <td className="n">{w.hit}</td>
              <td className="n">{w.dmg}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <div className="opfoot">
        <span className="ab">{o.abilities.join(" · ")}</span>
        <button onClick={() => onToggleDown(view.op)}>
          {view.dead[view.op] ? "Back up" : "Incapacitated"}
        </button>
      </div>
    </div>
  );
}
