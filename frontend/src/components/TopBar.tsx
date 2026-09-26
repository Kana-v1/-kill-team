import type { View } from "../types";

interface Props {
  view: View;
  onEvent: (t: string, p?: Record<string, unknown>) => void;
  onUndo: () => void;
}

// The sticky control bar: turning-point and command-point gauges, undo, reset TP.
export function TopBar({ view, onEvent, onUndo }: Props) {
  return (
    <div className="bar">
      <div className="gauge" id="tpGauge">
        <button onClick={() => onEvent("TP_PREV")} disabled={!view.canPrevTP} aria-label="Previous turning point">
          −
        </button>
        <div className="mid">
          <span className="v">{view.tp}</span>
          <span className="k">Turn Pt</span>
        </div>
        <button onClick={() => onEvent("TP_NEXT")} aria-label="Next turning point">
          +
        </button>
      </div>
      <div className="gauge" id="cpGauge">
        <button onClick={() => onEvent("CP", { d: -1 })} aria-label="Spend a command point">
          −
        </button>
        <div className="mid">
          <span className="v">{view.cp}</span>
          <span className="k">Command</span>
        </div>
        <button onClick={() => onEvent("CP", { d: 1 })} aria-label="Gain a command point">
          +
        </button>
      </div>
      <div className="mini">
        <button className="minibtn" onClick={onUndo} disabled={!view.canUndo}>
          Undo
        </button>
        <button className="minibtn" onClick={() => onEvent("TP_RESET")}>
          Reset TP
        </button>
      </div>
    </div>
  );
}
