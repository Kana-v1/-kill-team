import { useState } from "react";
import type { RunningCard, SpentCard, UsableCard } from "../types";

// ---- Usable ----

interface UsableProps {
  cards: UsableCard[];
  onActivate: (id: string, opt?: string) => void;
}

export function UsableList({ cards, onActivate }: UsableProps) {
  return (
    <section id="usable">
      <div className="head">
        <h2>Usable now</h2>
        <span className="ct">{cards.length || ""}</span>
      </div>
      <div>
        {cards.length === 0 ? (
          <p className="empty">Nothing available in this window.</p>
        ) : (
          cards.map((c) => <Usable key={c.id} card={c} onActivate={onActivate} />)
        )}
      </div>
    </section>
  );
}

function Usable({ card, onActivate }: { card: UsableCard; onActivate: (id: string, opt?: string) => void }) {
  const [open, setOpen] = useState(false);
  const hasOptions = (card.options?.length ?? 0) > 0;
  const cost = card.afford ? card.costShown : card.costShown + " — short";

  const click = () => {
    if (!card.afford) return;
    if (hasOptions) setOpen((v) => !v);
    else onActivate(card.id);
  };

  return (
    <button className={`card ${card.afford ? "" : "locked"}`} onClick={click} disabled={!card.afford && !hasOptions}>
      <span className="kind">
        <span>{card.kindLabel}</span>
      </span>
      <span className="top">
        <span className="nm">{card.name}</span>
        <span className="cost">{cost}</span>
      </span>
      <span className="tx">{card.text}</span>
      {card.condition && (
        <span className="cond">
          {card.from}: {card.condition}
        </span>
      )}
      {card.maybe && card.maybe.length > 0 && (
        <>
          <span className="freepill">Can be free</span>
          {card.maybe.map((r, i) => (
            <span className="cond" key={i}>
              {r.from}
              {r.options ? " · " + r.options.join(" / ") : ""}: {r.condition}
              {r.needs ? " — select that operative above" : ""}
              {r.disputed ? " — UNVERIFIED" : ""}
            </span>
          ))}
        </>
      )}
      {open && hasOptions && (
        <span className="opts">
          {card.options!.map((o) => (
            <span
              className="opt"
              key={o.id}
              role="button"
              tabIndex={0}
              onClick={(e) => {
                e.stopPropagation();
                onActivate(card.id, o.id);
              }}
            >
              {o.name}
              <small>{o.condition}</small>
              <b>{o.free ? "free" : o.cp + " CP"}</b>
            </span>
          ))}
        </span>
      )}
    </button>
  );
}

// ---- Running ----

interface RunningProps {
  cards: RunningCard[];
  onEnd: (id: string) => void;
  onMarkUsed: (id: string) => void;
}

export function RunningList({ cards, onEnd, onMarkUsed }: RunningProps) {
  const now = cards.filter((c) => c.now);
  const later = cards.filter((c) => !c.now);
  const ct = now.length ? `${now.length} now · ${cards.length} total` : cards.length || "";
  return (
    <section id="running">
      <div className="head">
        <h2>Running</h2>
        <span className="ct">{ct}</span>
      </div>
      <div>
        {cards.length === 0 && <p className="empty">Nothing active.</p>}
        {now.length > 0 && (
          <>
            <p className="sub" style={{ color: "var(--go)" }}>
              Applies right now
            </p>
            {now.map((c) => (
              <Running key={c.id} card={c} onEnd={onEnd} onMarkUsed={onMarkUsed} />
            ))}
          </>
        )}
        {later.length > 0 && (
          <>
            <p className="sub">Active, not in this window</p>
            {later.map((c) => (
              <Running key={c.id} card={c} onEnd={onEnd} onMarkUsed={onMarkUsed} />
            ))}
          </>
        )}
      </div>
    </section>
  );
}

function Running({
  card,
  onEnd,
  onMarkUsed,
}: {
  card: RunningCard;
  onEnd: (id: string) => void;
  onMarkUsed: (id: string) => void;
}) {
  return (
    <div className={`card ${card.now ? "on" : "off"}`}>
      <span className="kind">
        <span>{card.kindLabel}</span>
        {card.now && <span className="now">Applies now</span>}
      </span>
      <span className="top">
        <span className="nm">{card.name}</span>
        {card.canEnd && (
          <button className="cost" onClick={() => onEnd(card.id)}>
            end ×
          </button>
        )}
        {card.canMarkUsed && (
          <button className="cost" onClick={() => onMarkUsed(card.id)}>
            mark used ×
          </button>
        )}
      </span>
      <span className="tx">{card.text}</span>
      {card.disputed && (
        <>
          <span className="unver">Unverified</span>
          <span className="cond">Sources disagree on this rule — check your datacard.</span>
        </>
      )}
      <span className="meta">{card.meta}</span>
    </div>
  );
}

// ---- Spent ----

export function SpentList({ cards }: { cards: SpentCard[] }) {
  return (
    <section id="spent">
      <div className="head">
        <h2>Spent this turning point</h2>
        <span className="ct">{cards.length || ""}</span>
      </div>
      <div>
        {cards.length === 0 ? (
          <p className="empty">Nothing spent yet.</p>
        ) : (
          cards.map((c) => (
            <div className="card" key={c.id}>
              <span className="kind">
                <span>{c.kindLabel}</span>
              </span>
              <span className="top">
                <span className="nm">{c.name}</span>
                <span className="cost">used</span>
              </span>
            </div>
          ))
        )}
      </div>
    </section>
  );
}
