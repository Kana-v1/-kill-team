import type { Context } from "../types";

interface Props {
  contexts: Context[];
  active: string;
  onSelect: (ctxId: string) => void;
}

// The moment selector: which part of the turn we're in right now.
export function ContextChips({ contexts, active, onSelect }: Props) {
  return (
    <div className="ctx">
      {contexts.map((c) => (
        <button
          key={c.id}
          className="chip"
          aria-pressed={c.id === active}
          onClick={() => onSelect(c.id)}
        >
          {c.label}
        </button>
      ))}
    </div>
  );
}
