import { useCallback, useEffect, useRef, useState } from "react";
import { api, ApiError, type TeamSummary } from "./api";
import type { EventParams, Rules, View } from "./types";
import { TopBar } from "./components/TopBar";
import { ContextChips } from "./components/ContextChips";
import { Prompts, SummaryBanner } from "./components/Prompts";
import { StatPanel } from "./components/StatPanel";
import { UsableList, RunningList, SpentList } from "./components/Lists";
import { Setup } from "./components/Setup";

const GAME_KEY = "kt-aod-game-v1";

function loadGameId(): string | null {
  try {
    return localStorage.getItem(GAME_KEY);
  } catch {
    return null;
  }
}
function saveGameId(id: string) {
  try {
    localStorage.setItem(GAME_KEY, id);
  } catch {
    /* best-effort */
  }
}

export default function App() {
  const [rules, setRules] = useState<Rules | null>(null);
  const [view, setView] = useState<View | null>(null);
  const [teams, setTeams] = useState<TeamSummary[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [setupOpen, setSetupOpen] = useState(false);
  const gameId = useRef<string | null>(null);

  // Boot: resume the saved game (loading its team's rules) or create a new one
  // for the server's active team.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const teamList = await api.listTeams();
        if (!cancelled) setTeams(teamList);

        let v: View;
        let freshGame = false;
        const existing = loadGameId();
        if (existing) {
          try {
            v = await api.getGame(existing);
            gameId.current = existing;
          } catch (e) {
            if (e instanceof ApiError && e.status === 404) {
              const created = await api.createGame();
              gameId.current = created.id;
              saveGameId(created.id);
              v = created.view;
              freshGame = true;
            } else {
              throw e;
            }
          }
        } else {
          const created = await api.createGame();
          gameId.current = created.id;
          saveGameId(created.id);
          v = created.view;
          freshGame = true;
        }

        // Load the rules for whichever team this game belongs to.
        const r = await api.rules(v.team);
        if (!cancelled) {
          setRules(r);
          setView(v);
          if (freshGame) setSetupOpen(true);
        }
      } catch (e) {
        if (!cancelled) setError(errMessage(e));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  // Switching team starts a fresh game for it (a game is bound to one team) and
  // loads that team's rules.
  const switchTeam = useCallback(
    async (teamId: string) => {
      try {
        const created = await api.createGame(teamId);
        gameId.current = created.id;
        saveGameId(created.id);
        const r = await api.rules(teamId);
        setRules(r);
        setView(created.view);
        setSetupOpen(true);
        setError(null);
      } catch (e) {
        setError(errMessage(e));
      }
    },
    [],
  );

  const run = useCallback(async (fn: () => Promise<View>) => {
    try {
      const v = await fn();
      setView(v);
      setError(null);
    } catch (e) {
      setError(errMessage(e));
    }
  }, []);

  const onEvent = useCallback(
    (t: string, p?: Record<string, unknown>) => {
      const id = gameId.current;
      if (!id) return;
      void run(() => api.event(id, { t, p: (p ?? {}) as EventParams }));
    },
    [run],
  );

  const onUndo = useCallback(() => {
    const id = gameId.current;
    if (!id) return;
    void run(() => api.undo(id));
  }, [run]);

  const onReset = useCallback(() => {
    const id = gameId.current;
    if (!id) return;
    void run(() => api.reset(id));
    setSetupOpen(true);
  }, [run]);

  if (error && !view) {
    return (
      <div className="app">
        <p className="banner">Could not reach the tracker backend: {error}</p>
        <p className="setnote">
          Start it with <code>go run ./cmd/server</code> in <code>backend/</code>, then reload.
        </p>
      </div>
    );
  }
  if (!rules || !view) {
    return (
      <div className="app">
        <p className="loading">Loading tracker…</p>
      </div>
    );
  }

  return (
    <div className="app">
      <TopBar view={view} onEvent={onEvent} onUndo={onUndo} />
      <ContextChips
        contexts={rules.vocab.contexts}
        active={view.ctx}
        onSelect={(ctx) => onEvent("CTX", { ctx })}
      />
      {error && <p className="banner">Action failed: {error}</p>}
      <Prompts prompts={view.prompts} onDismiss={(k) => onEvent("DISMISS", { k })} />
      <SummaryBanner summary={view.summary} onClear={() => onEvent("CLEAR_SUMMARY")} />
      <StatPanel
        operatives={rules.operatives}
        view={view}
        onSelectOp={(id) => onEvent("OP", { id })}
        onToggleDown={(id) => onEvent("DOWN", { id })}
      />
      <UsableList cards={view.usable} onActivate={(id, opt) => onEvent("ACTIVATE", opt ? { id, opt } : { id })} />
      <RunningList
        cards={view.running}
        onEnd={(id) => onEvent("END", { id })}
        onMarkUsed={(id) => onEvent("USE_BATTLE", { id })}
      />
      <SpentList cards={view.spent} />
      <Setup
        rules={rules}
        view={view}
        teams={teams}
        open={setupOpen}
        onEvent={onEvent}
        onReset={onReset}
        onSwitchTeam={switchTeam}
      />
    </div>
  );
}

function errMessage(e: unknown): string {
  if (e instanceof Error) return e.message;
  return String(e);
}
