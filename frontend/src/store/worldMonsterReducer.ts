export type WorldMonsterState = "IDLE" | "CHASE" | "ATTACK" | "RETURN" | "DEFEATED";

export interface WorldMonster {
  instance_id: string;
  spawn_id: string;
  monster_template_id: string;
  is_boss?: boolean;
  rank?: number;
  sub_rank?: number;
  region_id: string;
  x: number;
  y: number;
  spawn_x: number;
  spawn_y: number;
  hp: number;
  max_hp: number;
  state: WorldMonsterState;
  target_id?: string;
  revision: number;
  last_update_at: string;
}

export interface WorldCombatEvent {
  event_id: string;
  server_time: string;
  region_id: string;
  entity_id: string;
  entity_revision: number;
  event_type: string;
  position: { x: number; y: number };
  payload?: Record<string, unknown>;
}

export interface MonsterSelection {
  instanceId?: string;
  x: number;
  y: number;
  hp: number;
  maxHp: number;
}

export function legacyMonsterHasRuntimeReplacement(
  cell: { x: number; y: number },
  monsters: Record<string, WorldMonster>,
): boolean {
  return Object.values(monsters).some((monster) =>
    monster.state !== "DEFEATED" && Math.floor(monster.spawn_x) === cell.x && Math.floor(monster.spawn_y) === cell.y,
  );
}

// Static cell chunks only need repainting when the set of runtime spawn
// replacements changes. Movement, HP, and revision updates deliberately do not
// affect this key, avoiding a full chunk repaint on every chase tick.
export function runtimeMonsterReplacementKey(monsters: Record<string, WorldMonster>): string {
  return [...new Set(Object.values(monsters)
    .filter((monster) => monster.state !== "DEFEATED")
    .map((monster) => `${monster.region_id}:${Math.floor(monster.spawn_x)},${Math.floor(monster.spawn_y)}`))]
    .sort()
    .join("|");
}

export interface WorldMonsterReducerState<TSelection extends MonsterSelection = MonsterSelection> {
  monsters: Record<string, WorldMonster>;
  seenCombatEvents: string[];
  selectedTarget: TSelection | null;
  targetingGu: unknown | null;
}

const authoritativeStates = new Set<WorldMonsterState>(["IDLE", "CHASE", "ATTACK", "RETURN", "DEFEATED"]);

function payloadMonster(event: WorldCombatEvent): WorldMonster | null {
  const entity = event.payload?.entity;
  if (!entity || typeof entity !== "object") return null;
  return entity as WorldMonster;
}

export function reduceWorldCombatEvent<TSelection extends MonsterSelection>(
  state: WorldMonsterReducerState<TSelection>,
  event: WorldCombatEvent,
): WorldMonsterReducerState<TSelection> {
  if (!event.event_id || state.seenCombatEvents.includes(event.event_id)) return state;
  const seenCombatEvents = [...state.seenCombatEvents, event.event_id].slice(-1000);
  const current = state.monsters[event.entity_id];
  if (current && event.entity_revision <= current.revision) return { ...state, seenCombatEvents };

  const removed = event.event_type === "monster_defeated" || event.event_type === "monster_despawned";
  if (removed) {
    if (!current) return { ...state, seenCombatEvents };
    const monsters = { ...state.monsters };
    delete monsters[event.entity_id];
    const selectedRemoved = state.selectedTarget?.instanceId === event.entity_id;
    return {
      ...state,
      monsters,
      seenCombatEvents,
      selectedTarget: selectedRemoved ? null : state.selectedTarget,
      targetingGu: selectedRemoved ? null : state.targetingGu,
    };
  }

  const fullEntity = payloadMonster(event);
  let base = current ?? fullEntity;
  if (!base) return { ...state, seenCombatEvents };

  const payloadState = event.payload?.state;
  const nextState = typeof payloadState === "string" && authoritativeStates.has(payloadState as WorldMonsterState)
    ? payloadState as WorldMonsterState
    : base.state;
  const hp = typeof event.payload?.hp === "number" ? event.payload.hp : base.hp;
  const maxHp = typeof event.payload?.max_hp === "number" ? event.payload.max_hp : base.max_hp;
  const targetID = typeof event.payload?.target_id === "string" ? event.payload.target_id : base.target_id;
  const next: WorldMonster = {
    ...base,
    ...(fullEntity ?? {}),
    instance_id: event.entity_id,
    region_id: event.region_id || base.region_id,
    x: event.position.x,
    y: event.position.y,
    hp,
    max_hp: maxHp,
    state: nextState,
    target_id: targetID || undefined,
    revision: event.entity_revision,
    last_update_at: event.server_time,
  };

  const monsters = { ...state.monsters };
  let replacedSelected = false;
  for (const [id, monster] of Object.entries(monsters)) {
    if (id !== next.instance_id && monster.spawn_id === next.spawn_id) {
      delete monsters[id];
      replacedSelected ||= state.selectedTarget?.instanceId === id;
    }
  }
  monsters[next.instance_id] = next;
  const selectedTarget = replacedSelected ? null : state.selectedTarget?.instanceId === next.instance_id
    ? { ...state.selectedTarget, x: next.x, y: next.y, hp: next.hp, maxHp: next.max_hp }
    : state.selectedTarget;
  return {
    ...state,
    monsters,
    seenCombatEvents,
    selectedTarget,
    targetingGu: replacedSelected ? null : state.targetingGu,
  };
}
