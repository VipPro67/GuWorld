export const WORLD_PROTOCOL_VERSION = 1;
export const WORLD_MAX_MESSAGE_BYTES = 16 * 1024;

export type WorldMessageType =
  | "world.join"
  | "world.welcome"
  | "world.input"
  | "world.move_target"
  | "world.cast"
  | "world.attack"
  | "world.events"
  | "world.snapshot"
  | "world.transfer"
  | "world.reject"
  | "world.resync";

export interface WorldEnvelope<TPayload = unknown> {
  v: typeof WORLD_PROTOCOL_VERSION;
  type: WorldMessageType;
  request_id?: string;
  payload: TPayload;
}

export interface WorldInputPayload {
  seq: number;
  move?: { x: number; y: number };
}

export interface MoveTargetPayload {
  seq: number;
  x: number;
  y: number;
}

export interface WorldEntityState {
  entity_id: string;
  entity_revision: number;
  kind?: "player" | "monster";
  party_id?: string;
  template_id?: string;
  name?: string;
  ai_state?: string;
  aggro_target_id?: string;
  x: number;
  y: number;
  hp: number;
  max_hp?: number;
  shield?: number;
  mana?: number;
  max_mana?: number;
  thought?: number;
  max_thought?: number;
  cooldowns?: number[];
  cast_skill_id?: string;
  cast_end_tick?: number;
  cast_windup_ticks?: number;
  statuses?: string[];
  stealthed?: boolean;
  flying?: boolean;
}

export interface WorldProjectileState {
  id: number;
  source_id: string;
  skill_id: string;
  x: number;
  y: number;
  radius: number;
}

export interface WorldSkillInfo {
  slot: number;
  id: string;
  name: string;
  mode: "TARGET" | "DIRECTION" | "GROUND" | "SELF";
  shape?: "CIRCLE" | "CONE" | "RECTANGLE" | "DONUT" | "CROSS" | "WALL";
  angle_deg?: number;
  length_units?: number;
  width_units?: number;
  inner_radius?: number;
  mana_cost: number;
  range: number;
  radius_units?: number;
  cooldown_ticks: number;
  windup_ticks: number;
  is_toggle?: boolean;
}

export interface CastPayload {
  seq: number;
  slot: number;
  mode: "TARGET" | "DIRECTION" | "GROUND" | "SELF";
  target_id?: string;
  aim_x?: number;
  aim_y?: number;
  x?: number;
  y?: number;
}

export interface BasicAttackPayload {
  seq: number;
  target_id: string;
}

export interface WorldEvent {
  event_id: number;
  tick: number;
  type: string;
  source_id?: string;
  target_id?: string;
  payload?: Record<string, unknown>;
}

export interface WorldEventBatch {
  from_event_id: number;
  events: WorldEvent[];
}

export interface WorldSnapshotPayload {
  tick: number;
  ack_seq: number;
  baseline_id: number;
  base_tick?: number;
  is_baseline: boolean;
  entities: WorldEntityState[];
  removed?: string[];
  projectiles?: WorldProjectileState[];
}

export interface WorldWelcomePayload {
  session_id: string;
  zone_id: string;
  map_version: string;
  content_version: string;
  tick_rate: number;
  server_tick: number;
  entity_id: string;
  entity_revision: number;
  baseline_id: number;
  map_hash?: string;
  skills?: WorldSkillInfo[];
  loadout_revision?: string;
  snapshot: WorldSnapshotPayload;
}

export interface WorldJoinPayload {
  zone_id: string;
  map_version: string;
  content_version: string;
}

export interface WorldRejectPayload {
  code: string;
  message?: string;
  retryable?: boolean;
}

/**
 * Server-owned border handoff. target_x/target_y are MMO map-local cells;
 * legacy_x/legacy_y are the same point projected into the players.x/y world
 * space. Callers must persist the legacy pair, never the map-local one.
 */
export interface WorldTransferPayload {
  from_zone_id?: string;
  to_zone_id: string;
  transfer_id?: string;
  target_x: number;
  target_y: number;
  legacy_x: number;
  legacy_y: number;
  hp: number;
}

const knownTypes = new Set<WorldMessageType>([
  "world.join",
  "world.welcome",
  "world.input",
  "world.move_target",
  "world.cast",
  "world.attack",
  "world.events",
  "world.snapshot",
  "world.transfer",
  "world.reject",
  "world.resync",
]);

/**
 * The legacy hub multiplexes chat/trade/presence frames (`event`, `pong`,
 * `response`, ...) on the same `/api/ws` socket as the realtime world
 * protocol. Only `world.*` frames belong to this protocol; callers must skip
 * everything else instead of treating it as a protocol error.
 */
export function isWorldProtocolMessage(value: unknown): boolean {
  return isRecord(value) && typeof value.type === "string" && value.type.startsWith("world.");
}

/** Validates only the envelope; callers must validate the payload for its type. */
export function decodeWorldEnvelope(input: string): WorldEnvelope {
  if (new TextEncoder().encode(input).byteLength > WORLD_MAX_MESSAGE_BYTES) {
    throw new Error("Realtime world message is too large");
  }
  const value: unknown = JSON.parse(input);
  if (!isRecord(value)) throw new Error("Realtime world message must be an object");
  if (value.v !== WORLD_PROTOCOL_VERSION) throw new Error("Unsupported realtime world protocol version");
  if (typeof value.type !== "string" || !knownTypes.has(value.type as WorldMessageType)) {
    throw new Error("Unknown realtime world message type");
  }
  if (!isRecord(value.payload)) throw new Error("Realtime world payload must be an object");
  if (value.request_id !== undefined && (typeof value.request_id !== "string" || value.request_id.length > 64)) {
    throw new Error("Invalid realtime world request_id");
  }
  return {
    v: WORLD_PROTOCOL_VERSION,
    type: value.type as WorldMessageType,
    request_id: value.request_id as string | undefined,
    payload: value.payload,
  };
}

export function parseWorldSnapshotPayload(value: unknown): WorldSnapshotPayload {
  if (!isRecord(value)) throw new Error("World snapshot payload must be an object");
  const { tick, ack_seq: ackSeq, baseline_id: baselineId, base_tick: baseTick, is_baseline: isBaseline } = value;
  if (!isSafeNonNegativeInteger(tick) || !isSafeNonNegativeInteger(ackSeq) || !isSafePositiveInteger(baselineId) || typeof isBaseline !== "boolean") {
    throw new Error("World snapshot tick, ack, baseline or mode is invalid");
  }
  if (baseTick !== undefined && !isSafeNonNegativeInteger(baseTick)) throw new Error("World snapshot base_tick is invalid");
  if (isBaseline && baseTick !== undefined && baseTick !== tick) throw new Error("World baseline base_tick must match tick");
  if (!isBaseline && (!isSafeNonNegativeInteger(baseTick) || baseTick > tick)) throw new Error("World delta requires a valid base_tick");
  if (!Array.isArray(value.entities) || value.entities.length > 1000) throw new Error("World snapshot entity list is invalid");
  const entities = value.entities.map(parseEntityState);
  const ids = new Set<string>();
  for (const entity of entities) {
    if (ids.has(entity.entity_id)) throw new Error("World snapshot contains a duplicate entity");
    ids.add(entity.entity_id);
  }
  let removed: string[] | undefined;
  if (value.removed !== undefined) {
    if (!Array.isArray(value.removed) || value.removed.length > 1000) throw new Error("World snapshot removed list is invalid");
    removed = value.removed.map((id) => {
      if (typeof id !== "string" || id.length === 0 || id.length > 80 || id.trim() !== id) throw new Error("World snapshot contains an invalid removed entity ID");
      return id;
    });
    if (new Set(removed).size !== removed.length || removed.some((id) => ids.has(id))) throw new Error("World snapshot has duplicate or conflicting removals");
  }
  let projectiles: WorldProjectileState[] | undefined;
  if (value.projectiles !== undefined) {
    if (!Array.isArray(value.projectiles) || value.projectiles.length > 200) throw new Error("World snapshot projectiles are invalid");
    projectiles = value.projectiles.map((raw) => {
      if (!isRecord(raw) || !isSafePositiveInteger(raw.id) || typeof raw.source_id !== "string" || raw.source_id.length > 80 || typeof raw.skill_id !== "string" || raw.skill_id.length > 80) {
        throw new Error("World snapshot projectile identity is invalid");
      }
      if (!isFiniteNumber(raw.x) || !isFiniteNumber(raw.y) || !isFiniteNumber(raw.radius) || raw.radius <= 0 || Math.abs(raw.x) > 1_000_000 || Math.abs(raw.y) > 1_000_000) {
        throw new Error("World snapshot projectile transform is invalid");
      }
      return {
        id: raw.id, source_id: raw.source_id, skill_id: raw.skill_id,
        x: raw.x, y: raw.y, radius: raw.radius,
      };
    });
  }
  return {
    tick,
    ack_seq: ackSeq,
    baseline_id: baselineId,
    base_tick: baseTick as number | undefined,
    is_baseline: isBaseline,
    entities,
    removed,
    projectiles,
  };
}

export function parseWorldWelcomePayload(value: unknown): WorldWelcomePayload {
  if (!isRecord(value)) throw new Error("World welcome payload must be an object");
  const stringFields = [value.session_id, value.zone_id, value.map_version, value.content_version, value.entity_id];
  if (stringFields.some((field) => typeof field !== "string" || field.length === 0)) throw new Error("World welcome identity is invalid");
  if (!isSafePositiveInteger(value.tick_rate) || !isSafeNonNegativeInteger(value.server_tick) || !isSafeNonNegativeInteger(value.entity_revision) || !isSafePositiveInteger(value.baseline_id)) {
    throw new Error("World welcome timing or baseline is invalid");
  }
  const snapshot = parseWorldSnapshotPayload(value.snapshot);
  if (!snapshot.is_baseline || snapshot.baseline_id !== value.baseline_id || snapshot.tick !== value.server_tick || !snapshot.entities.some((entity) => entity.entity_id === value.entity_id)) {
    throw new Error("World welcome does not contain a matching owner baseline");
  }
  if (value.map_hash !== undefined && (typeof value.map_hash !== "string" || !/^[0-9a-f]{64}$/.test(value.map_hash))) {
    throw new Error("World welcome map_hash is invalid");
  }
  if (value.loadout_revision !== undefined && (typeof value.loadout_revision !== "string" || value.loadout_revision.length > 128)) {
    throw new Error("World welcome loadout revision is invalid");
  }
  let skills: WorldSkillInfo[] | undefined;
  if (value.skills !== undefined) {
    if (!Array.isArray(value.skills) || value.skills.length > 6) throw new Error("World welcome skills are invalid");
    const slots = new Set<number>();
    skills = value.skills.map((raw) => {
      if (!isRecord(raw) || !isSafePositiveInteger(raw.slot) || raw.slot > 6 || slots.has(raw.slot)) throw new Error("World welcome skill slot is invalid");
      slots.add(raw.slot);
      const mode = raw.mode;
      if (mode !== "TARGET" && mode !== "DIRECTION" && mode !== "GROUND" && mode !== "SELF") throw new Error("World welcome skill mode is invalid");
      if (typeof raw.id !== "string" || raw.id.length === 0 || raw.id.length > 80 || typeof raw.name !== "string" || raw.name.length > 120) {
        throw new Error("World welcome skill identity is invalid");
      }
      if (!isFiniteNumber(raw.mana_cost) || raw.mana_cost < 0 || !isFiniteNumber(raw.range) || !isSafeNonNegativeInteger(raw.cooldown_ticks) || !isSafeNonNegativeInteger(raw.windup_ticks)) {
        throw new Error("World welcome skill stats are invalid");
      }
      if (raw.radius_units !== undefined && (!isFiniteNumber(raw.radius_units) || raw.radius_units < 0)) {
        throw new Error("World welcome skill radius is invalid");
      }
      return {
        slot: raw.slot, id: raw.id, name: raw.name, mode,
        shape: typeof raw.shape === "string" ? raw.shape as WorldSkillInfo["shape"] : undefined,
        angle_deg: isFiniteNumber(raw.angle_deg) ? raw.angle_deg : undefined,
        length_units: isFiniteNumber(raw.length_units) ? raw.length_units : undefined,
        width_units: isFiniteNumber(raw.width_units) ? raw.width_units : undefined,
        inner_radius: isFiniteNumber(raw.inner_radius) ? raw.inner_radius : undefined,
        mana_cost: raw.mana_cost, range: raw.range,
        radius_units: raw.radius_units as number | undefined,
        cooldown_ticks: raw.cooldown_ticks, windup_ticks: raw.windup_ticks,
      };
    });
  }
  return {
    session_id: value.session_id as string,
    zone_id: value.zone_id as string,
    map_version: value.map_version as string,
    content_version: value.content_version as string,
    tick_rate: value.tick_rate,
    server_tick: value.server_tick,
    entity_id: value.entity_id as string,
    entity_revision: value.entity_revision,
    baseline_id: value.baseline_id,
    map_hash: value.map_hash as string | undefined,
    skills,
    loadout_revision: value.loadout_revision as string | undefined,
    snapshot,
  };
}

export function parseWorldTransferPayload(value: unknown): WorldTransferPayload {
  if (!isRecord(value)) throw new Error("World transfer payload must be an object");
  if (typeof value.to_zone_id !== "string" || value.to_zone_id.length === 0 || value.to_zone_id.length > 80 || value.to_zone_id.trim() !== value.to_zone_id) {
    throw new Error("World transfer destination is invalid");
  }
  if (value.from_zone_id !== undefined && (typeof value.from_zone_id !== "string" || value.from_zone_id.length > 80)) {
    throw new Error("World transfer source is invalid");
  }
  if (value.transfer_id !== undefined && (typeof value.transfer_id !== "string" || value.transfer_id.length > 80)) {
    throw new Error("World transfer id is invalid");
  }
  for (const field of [value.target_x, value.target_y, value.hp]) {
    if (!isFiniteNumber(field) || field < 0) throw new Error("World transfer coordinate or hp is invalid");
  }
  for (const field of [value.legacy_x, value.legacy_y]) {
    if (!isSafeNonNegativeInteger(field)) throw new Error("World transfer legacy coordinate is invalid");
  }
  return {
    from_zone_id: value.from_zone_id as string | undefined,
    to_zone_id: value.to_zone_id,
    transfer_id: value.transfer_id as string | undefined,
    target_x: value.target_x as number,
    target_y: value.target_y as number,
    legacy_x: value.legacy_x as number,
    legacy_y: value.legacy_y as number,
    hp: value.hp as number,
  };
}

function parseEntityState(value: unknown): WorldEntityState {
  if (!isRecord(value) || typeof value.entity_id !== "string" || value.entity_id.length === 0 || value.entity_id.length > 80 || value.entity_id.trim() !== value.entity_id) {
    throw new Error("World snapshot entity ID is invalid");
  }
  if (!isSafeNonNegativeInteger(value.entity_revision) || !isFiniteNumber(value.x) || !isFiniteNumber(value.y) || !isFiniteNumber(value.hp) || value.hp < 0 || Math.abs(value.x) > 1_000_000 || Math.abs(value.y) > 1_000_000) {
    throw new Error("World snapshot entity state is invalid");
  }
  if (value.kind !== undefined && value.kind !== "player" && value.kind !== "monster") {
    throw new Error("World snapshot entity kind is invalid");
  }
  if (value.max_hp !== undefined && (!isFiniteNumber(value.max_hp) || value.max_hp <= 0)) {
    throw new Error("World snapshot entity max_hp is invalid");
  }
  if (value.shield !== undefined && (!isFiniteNumber(value.shield) || value.shield < 0)) {
    throw new Error("World snapshot entity shield is invalid");
  }
  for (const field of [value.mana, value.max_mana]) {
    if (field !== undefined && (!isFiniteNumber(field) || field < 0)) throw new Error("World snapshot combat resource is invalid");
  }
  for (const field of [value.cast_end_tick, value.cast_windup_ticks]) {
    if (field !== undefined && !isSafeNonNegativeInteger(field)) throw new Error("World snapshot combat timing is invalid");
  }
  if (value.cast_skill_id !== undefined && (typeof value.cast_skill_id !== "string" || value.cast_skill_id.length > 80)) {
    throw new Error("World snapshot cast skill id is invalid");
  }
  if (value.cooldowns !== undefined) {
    if (!Array.isArray(value.cooldowns) || value.cooldowns.length > 7 || value.cooldowns.some((entry) => !isSafeNonNegativeInteger(entry))) {
      throw new Error("World snapshot cooldowns are invalid");
    }
  }
  const entity: WorldEntityState = {
    entity_id: value.entity_id,
    entity_revision: value.entity_revision,
    x: value.x,
    y: value.y,
    hp: value.hp,
  };
  if (value.kind !== undefined) entity.kind = value.kind as "player" | "monster";
  if (value.party_id !== undefined && (typeof value.party_id !== "string" || value.party_id.length > 80)) {
    throw new Error("World snapshot party_id is invalid");
  }
  if (value.party_id !== undefined) entity.party_id = value.party_id as string;
  for (const field of ["template_id", "name", "ai_state", "aggro_target_id"] as const) {
    const raw = value[field];
    if (raw !== undefined) {
      if (typeof raw !== "string" || raw.length > 120) throw new Error(`World snapshot ${field} is invalid`);
      entity[field] = raw;
    }
  }
  if (value.max_hp !== undefined) entity.max_hp = value.max_hp as number;
  if (value.shield !== undefined) entity.shield = value.shield as number;
  if (value.mana !== undefined) entity.mana = value.mana as number;
  if (value.max_mana !== undefined) entity.max_mana = value.max_mana as number;
  if (value.cooldowns !== undefined) entity.cooldowns = (value.cooldowns as number[]).slice();
  if (value.cast_skill_id !== undefined) entity.cast_skill_id = value.cast_skill_id as string;
  if (value.cast_end_tick !== undefined) entity.cast_end_tick = value.cast_end_tick as number;
  if (value.cast_windup_ticks !== undefined) entity.cast_windup_ticks = value.cast_windup_ticks as number;
  return entity;
}

export function parseWorldEventBatch(value: unknown): WorldEventBatch {
  if (!isRecord(value) || !isSafeNonNegativeInteger(value.from_event_id) || !Array.isArray(value.events) || value.events.length > 500) {
    throw new Error("World events batch is invalid");
  }
  const events = value.events.map((raw): WorldEvent => {
    if (!isRecord(raw) || !isSafePositiveInteger(raw.event_id) || !isSafeNonNegativeInteger(raw.tick)) {
      throw new Error("World event identity is invalid");
    }
    if (typeof raw.type !== "string" || raw.type.length === 0 || raw.type.length > 64) {
      throw new Error("World event type is invalid");
    }
    for (const field of [raw.source_id, raw.target_id]) {
      if (field !== undefined && (typeof field !== "string" || field.length > 80 || field.trim() !== field)) {
        throw new Error("World event entity ID is invalid");
      }
    }
    if (raw.payload !== undefined && !isRecord(raw.payload)) throw new Error("World event payload is invalid");
    return {
      event_id: raw.event_id,
      tick: raw.tick,
      type: raw.type,
      source_id: raw.source_id as string | undefined,
      target_id: raw.target_id as string | undefined,
      payload: raw.payload as Record<string, unknown> | undefined,
    };
  });
  return { from_event_id: value.from_event_id, events };
}

function isSafeNonNegativeInteger(value: unknown): value is number {
  return typeof value === "number" && Number.isSafeInteger(value) && value >= 0;
}

function isSafePositiveInteger(value: unknown): value is number {
  return isSafeNonNegativeInteger(value) && value > 0;
}

function isFiniteNumber(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
