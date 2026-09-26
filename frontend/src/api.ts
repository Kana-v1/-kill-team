// Typed client for the Go backend. Every game mutation posts an event and gets
// back the fresh derived view; the client never re-implements rules logic.

import type { GameEvent, Rules, View } from "./types";

const BASE = "/api";

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(BASE + path, {
    ...init,
    headers: { "Content-Type": "application/json", ...(init?.headers ?? {}) },
  });
  if (!res.ok) {
    let msg = `${res.status} ${res.statusText}`;
    try {
      const body = (await res.json()) as { error?: string };
      if (body.error) msg = body.error;
    } catch {
      /* non-JSON error body */
    }
    throw new ApiError(res.status, msg);
  }
  return (await res.json()) as T;
}

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = "ApiError";
  }
}

export interface TeamSummary {
  id: string;
  name: string;
  edition: string;
  active: boolean;
}

export const api = {
  listTeams: () => req<TeamSummary[]>("/teams"),
  rules: (teamId: string) => req<Rules>(`/teams/${teamId}/rules`),

  createGame: (team?: string) =>
    req<{ id: string; view: View }>("/games", {
      method: "POST",
      body: JSON.stringify(team ? { team } : {}),
    }),
  getGame: (id: string) => req<View>(`/games/${id}`),
  event: (id: string, ev: GameEvent) =>
    req<View>(`/games/${id}/events`, { method: "POST", body: JSON.stringify(ev) }),
  undo: (id: string) => req<View>(`/games/${id}/undo`, { method: "POST" }),
  reset: (id: string) => req<View>(`/games/${id}/reset`, { method: "POST" }),
};
