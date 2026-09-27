// Types mirroring the Go backend's JSON. The View is the whole derived
// snapshot; Rules is the static census used for the stat panel and Setup.

export interface Cost {
  cp: number;
}

export interface Weapon {
  name: string;
  type: "ranged" | "melee";
  atk: number;
  hit: string;
  dmg: string;
  rules: string;
}

export interface Stats {
  apl: number;
  move: string;
  save: string;
  wounds: number;
}

export interface Operative {
  id: string;
  name: string;
  stats: Stats;
  abilities: string[];
  weapons: Weapon[];
  icon: string;
  accent: string;
  photo: string | null;
  leader: boolean;
  multiple: boolean;
  role: string;
  chapterVeteran: boolean;
}

export interface ChapterTactic {
  id: string;
  name: string;
  text: string;
  remind_at: string[];
  prompt: string;
}

export interface Context {
  id: string;
  label: string;
}

export interface EffectLite {
  id: string;
  name: string;
  kind: string;
}

export interface Rules {
  meta: { team: string; teamId: string; edition: string; unresolved?: string[] };
  vocab: { contexts: Context[] };
  effects: (EffectLite & { kind: string })[];
  operatives: Operative[];
  chapterTactics: ChapterTactic[];
}

// ---- derived view (from the engine) ----

export interface Prompt {
  id: string;
  name: string;
  text: string;
  key: string;
  paid: boolean;
}

export interface MaybeRoute {
  from: string;
  options?: string[];
  condition: string;
  needs: boolean;
  disputed: boolean;
}

export interface OptionQuote {
  id: string;
  name: string;
  condition: string;
  cp: number;
  free: boolean;
}

export interface UsableCard {
  id: string;
  name: string;
  kind: string;
  kindLabel: string;
  text: string;
  costBase: number;
  costShown: string;
  reduced: boolean;
  free: boolean;
  afford: boolean;
  condition?: string;
  from?: string;
  maybe?: MaybeRoute[];
  options?: OptionQuote[];
}

export interface RunningCard {
  id: string;
  name: string;
  kindLabel: string;
  text: string;
  meta: string;
  now: boolean;
  always: boolean;
  canEnd: boolean;
  canMarkUsed: boolean;
  disputed: boolean;
}

export interface SpentCard {
  id: string;
  name: string;
  kindLabel: string;
}

export interface Summary {
  tp: number;
  names: string[];
}

export interface RosterStatus {
  total: number;
  leaders: number;
  ok: boolean;
}

export interface View {
  team: string;
  tp: number;
  cp: number;
  ctx: string;
  roster: string[];
  op: string;
  dead: Record<string, boolean>;
  equip: Record<string, boolean>;
  tactics: { primary: string; secondary: string; extra: string };
  canUndo: boolean;
  canPrevTP: boolean;
  rosterStatus: RosterStatus;
  veterans: string[];
  prompts: Prompt[];
  summary: Summary | null;
  usable: UsableCard[];
  running: RunningCard[];
  spent: SpentCard[];
  verifyNote: { flagged: number; unresolved: string[] | null };
}

// ---- event payloads posted to the backend ----

export interface EventParams {
  ctx?: string;
  d?: number;
  id?: string;
  opt?: string;
  slot?: string;
  k?: string;
}

export interface GameEvent {
  t: string;
  p: EventParams;
}
