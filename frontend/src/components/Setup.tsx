import type { TeamSummary } from "../api";
import type { Rules, View } from "../types";
import { Glyph } from "../icons";

interface Props {
  rules: Rules;
  view: View;
  teams: TeamSummary[];
  open: boolean;
  onEvent: (t: string, p?: Record<string, unknown>) => void;
  onReset: () => void;
  onSwitchTeam: (teamId: string) => void;
}

const typeOf = (inst: string) => inst.split("#")[0];
const slots = ["primary", "secondary", "extra"] as const;

// The roster / equipment / tactics editor. All choices are events; the backend
// enforces composition rules and reports validity in view.rosterStatus.
export function Setup({ rules, view, teams, open, onEvent, onReset, onSwitchTeam }: Props) {
  const count = (id: string) => view.roster.filter((x) => typeOf(x) === id).length;

  const leaders = rules.operatives.filter((o) => o.leader);
  const specialists = rules.operatives.filter((o) => !o.leader && !o.multiple);
  const warriors = rules.operatives.filter((o) => o.multiple);
  const equipment = rules.effects.filter((e) => e.kind === "equipment");
  const hasTactics = rules.chapterTactics.length > 0;

  const rosterOk = view.rosterStatus.ok;
  const rosterLabel =
    `— ${view.rosterStatus.total} of 6 selected` + (view.rosterStatus.leaders === 1 ? "" : ", no leader");

  return (
    <details open={open}>
      <summary>Setup — roster &amp; equipment</summary>

      {teams.length > 1 && (
        <>
          <p className="lbl" style={{ margin: "14px 0 0" }}>
            Kill team
          </p>
          <div className="setgrid">
            {teams.map((t) => (
              <button
                key={t.id}
                className="toggle"
                aria-pressed={t.id === view.team}
                onClick={() => {
                  if (t.id !== view.team) onSwitchTeam(t.id);
                }}
              >
                {t.name}
              </button>
            ))}
          </div>
          <p className="setnote" style={{ margin: "6px 0 0" }}>
            Switching kill team starts a new game.
          </p>
        </>
      )}

      <p className="lbl" style={{ margin: "16px 0 0" }}>
        Leader <span style={{ color: rosterOk ? "var(--go)" : "var(--hot)" }}>{rosterLabel}</span>
      </p>
      <div className="setgrid">
        {leaders.map((o) => (
          <button
            key={o.id}
            className="toggle"
            aria-pressed={count(o.id) > 0}
            onClick={() => onEvent("LEADER", { id: o.id })}
          >
            <Glyph op={o} />
            {o.name}
          </button>
        ))}
      </div>

      <p className="lbl" style={{ margin: "16px 0 0" }}>
        Specialists — one of each
      </p>
      <div className="setgrid">
        {specialists.map((o) => (
          <button
            key={o.id}
            className="toggle"
            aria-pressed={count(o.id) > 0}
            onClick={() => onEvent("ROSTER", { id: o.id })}
          >
            <Glyph op={o} />
            {o.name}
          </button>
        ))}
      </div>

      <p className="lbl" style={{ margin: "16px 0 0" }}>
        Warriors — take as many as you like
      </p>
      <div className="setgrid">
        {warriors.map((o) => (
          <span className="counter" key={o.id} aria-label={o.name}>
            <button onClick={() => onEvent("COUNT", { id: o.id, d: -1 })} disabled={count(o.id) === 0}>
              −
            </button>
            <Glyph op={o} />
            <span className="cname">{o.name}</span>
            <span className="cnum">{count(o.id)}</span>
            <button onClick={() => onEvent("COUNT", { id: o.id, d: 1 })}>+</button>
          </span>
        ))}
      </div>

      {hasTactics && (
        <>
          <TacticGrid label="Primary chapter tactic" slot="primary" rules={rules} view={view} onEvent={onEvent} />
          <TacticGrid label="Secondary chapter tactic" slot="secondary" rules={rules} view={view} onEvent={onEvent} />
        </>
      )}

      {hasTactics && view.veterans.length > 0 && (
        <>
          <p className="lbl" style={{ margin: "16px 0 0" }}>
            Extra tactic <span>— Chapter Veteran, applies to {view.veterans.join(" / ")} only</span>
          </p>
          <div className="setgrid">
            {rules.chapterTactics.map((t) => (
              <button
                key={t.id}
                className="toggle"
                aria-pressed={view.tactics.extra === t.id}
                onClick={() => onEvent("TACTIC", { slot: "extra", id: t.id })}
              >
                {t.name}
              </button>
            ))}
          </div>
        </>
      )}

      <p className="lbl" style={{ margin: "16px 0 0" }}>
        Faction equipment taken
      </p>
      <div className="setgrid">
        {equipment.map((e) => (
          <button
            key={e.id}
            className="toggle"
            aria-pressed={!!view.equip[e.id]}
            onClick={() => onEvent("EQUIP", { id: e.id })}
          >
            {e.name}
          </button>
        ))}
      </div>

      <p className="setnote">
        Read through a PDF-summarising model, not parsed from the PDF itself. {view.verifyNote.flagged} entries carry
        assumptions.
        <br />
        <br />
        <b style={{ color: "var(--hot)" }}>Unresolved:</b>
        <br />
        {(view.verifyNote.unresolved ?? []).map((u, i) => (
          <span key={i}>
            · {u}
            <br />
          </span>
        ))}
      </p>
      <button className="danger" onClick={onReset}>
        Reset whole game
      </button>
    </details>
  );
}

function TacticGrid({
  label,
  slot,
  rules,
  view,
  onEvent,
}: {
  label: string;
  slot: (typeof slots)[number];
  rules: Rules;
  view: View;
  onEvent: (t: string, p?: Record<string, unknown>) => void;
}) {
  return (
    <>
      <p className="lbl" style={{ margin: "16px 0 0" }}>
        {label}
      </p>
      <div className="setgrid">
        {rules.chapterTactics.map((t) => (
          <button
            key={t.id}
            className="toggle"
            aria-pressed={view.tactics[slot] === t.id}
            onClick={() => onEvent("TACTIC", { slot, id: t.id })}
          >
            {t.name}
          </button>
        ))}
      </div>
    </>
  );
}
