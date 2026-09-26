import type { Prompt, Summary } from "../types";

interface PromptsProps {
  prompts: Prompt[];
  onDismiss: (key: string) => void;
}

// The amber banners: reminders that fire in the current window. One-tap dismiss.
export function Prompts({ prompts, onDismiss }: PromptsProps) {
  if (prompts.length === 0) return null;
  return (
    <div className="prompts">
      {prompts.map((p) => (
        <div className="prompt" key={p.key}>
          <div>
            <span className="src">{p.name}</span>
            <p>{p.text}</p>
          </div>
          <button className="x" aria-label="Dismiss" onClick={() => onDismiss(p.key)}>
            ×
          </button>
        </div>
      ))}
    </div>
  );
}

interface SummaryProps {
  summary: Summary | null;
  onClear: () => void;
}

// The end-of-turning-point "never used" recap.
export function SummaryBanner({ summary, onClear }: SummaryProps) {
  if (!summary) return null;
  return (
    <div className="summary">
      <h3>Turning point {summary.tp} — never used</h3>
      <ul>
        {summary.names.map((n, i) => (
          <li key={i}>{n}</li>
        ))}
      </ul>
      <button onClick={onClear}>Dismiss</button>
    </div>
  );
}
