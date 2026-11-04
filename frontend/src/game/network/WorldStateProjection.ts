import type { WorldEntityState, WorldProjectileState, WorldSnapshotPayload } from "./Protocol.ts";
import { dequantize, type DeltaEntity, type Projectile, type Spawn } from "./v2/Codec.ts";
import { statusId } from "./v2/status.ts";

type FullSnapshot = {
  tick: number;
  baselineId: number;
  baseTick: number;
  ackSeq: number;
  spawns: Spawn[];
  states: DeltaEntity[];
  removed?: number[];
  projectiles: Projectile[];
};

/** Owns protocol entity identity and projects full/delta data into world snapshots. */
export class WorldStateProjection {
  private readonly netToSpawn = new Map<number, Spawn>();
  private readonly entityByNet = new Map<number, WorldEntityState>();
  private readonly netByEntity = new Map<string, number>();

  public reset(): void {
    this.netToSpawn.clear();
    this.entityByNet.clear();
    this.netByEntity.clear();
  }

  public entityId(netId: number): string | undefined {
    return this.netToSpawn.get(netId)?.entityId;
  }

  public netId(entityId: string): number | undefined {
    return this.netByEntity.get(entityId);
  }

  public applyBaseline(full: Pick<FullSnapshot, "spawns" | "states">): void {
    this.reset();
    for (const spawn of full.spawns) {
      this.netToSpawn.set(spawn.netId, spawn);
      this.netByEntity.set(spawn.entityId, spawn.netId);
    }
    const states = new Map(full.states.map((state) => [state.netId, state]));
    for (const spawn of full.spawns) {
      const state = states.get(spawn.netId);
      if (state) this.entityByNet.set(spawn.netId, entityFrom(spawn, state, this.netToSpawn));
    }
  }

  public addSpawns(spawns: Spawn[]): void {
    for (const spawn of spawns) {
      this.netToSpawn.set(spawn.netId, spawn);
      this.netByEntity.set(spawn.entityId, spawn.netId);
    }
  }

  public projectDelta(delta: Pick<FullSnapshot, "tick" | "baselineId" | "ackSeq" | "states" | "removed" | "projectiles">, baseTick: number): WorldSnapshotPayload {
    return this.project({ ...delta, baseTick, spawns: [] }, false, baseTick);
  }

  public projectFull(full: FullSnapshot, isBaseline: boolean, previousBaseTick: number): WorldSnapshotPayload {
    this.addSpawns(full.spawns);
    return this.project(full, isBaseline, previousBaseTick);
  }

  private project(full: FullSnapshot, isBaseline: boolean, previousBaseTick: number): WorldSnapshotPayload {
    const entities: WorldEntityState[] = [];
    if (isBaseline) {
      const states = new Map(full.states.map((state) => [state.netId, state]));
      for (const spawn of full.spawns) {
        const state = states.get(spawn.netId);
        if (state) entities.push(entityFrom(spawn, state, this.netToSpawn));
      }
    } else {
      for (const state of full.states) {
        const spawn = this.netToSpawn.get(state.netId);
        if (!spawn) continue;
        const merged = mergeEntity(spawn, this.entityByNet.get(state.netId), state, this.netToSpawn);
        this.entityByNet.set(state.netId, merged);
        entities.push(merged);
      }
    }
    const removed: string[] = [];
    const seen = new Set<string>();
    for (const netId of full.removed ?? []) {
      const spawn = this.netToSpawn.get(netId);
      if (spawn && !seen.has(spawn.entityId)) {
        seen.add(spawn.entityId);
        removed.push(spawn.entityId);
      }
      this.netToSpawn.delete(netId);
      this.entityByNet.delete(netId);
      if (spawn) this.netByEntity.delete(spawn.entityId);
    }
    return {
      tick: full.tick,
      ack_seq: full.ackSeq,
      baseline_id: full.baselineId,
      base_tick: isBaseline ? full.tick : (full.baseTick || previousBaseTick || full.tick),
      is_baseline: isBaseline,
      entities,
      removed: removed.length ? removed : undefined,
      projectiles: mapProjectiles(full.projectiles, this.netToSpawn),
    };
  }
}

function mapProjectiles(projectiles: Projectile[], spawns: Map<number, Spawn>): WorldProjectileState[] {
  return projectiles.map((p) => ({
    id: p.id, source_id: spawns.get(p.sourceNetId)?.entityId ?? "", skill_id: p.skillId,
    projectile_vfx: p.vfx || undefined, x: dequantize(p.x), y: dequantize(p.y), radius: p.radiusX10 / 10,
  }));
}

function entityFrom(spawn: Spawn, state: DeltaEntity, spawns: Map<number, Spawn>): WorldEntityState {
  const ownerId = spawn.ownerNetId ? spawns.get(spawn.ownerNetId)?.entityId : undefined;
  const aiState = state.hasExt && (state.ext & 1) ? aiStateName(state.aiState) : undefined;
  const aggroTargetId = state.hasExt && (state.ext & 2) && state.aggroNetId ? spawns.get(state.aggroNetId)?.entityId : undefined;
  const stance = state.hasExt && (state.ext & 16) ? stanceName(state.stance) : (spawn.companionKind ? stanceName(spawn.stance) : undefined);
  return {
    entity_id: spawn.entityId, entity_revision: state.revision, kind: kindName(spawn.kind), party_id: spawn.partyId || undefined,
    pvp_enabled: (spawn.flags & 1) !== 0, template_id: spawn.templateId || undefined, name: spawn.name || undefined,
    ai_state: aiState, aggro_target_id: aggroTargetId, x: dequantize(state.x), y: dequantize(state.y), hp: state.hp,
    max_hp: spawn.maxHp || undefined, shield: state.shield || undefined, mana: state.mana, max_mana: spawn.maxMana || undefined,
    thought: state.thought, max_thought: spawn.maxThought || undefined, cast_end_tick: (state.mask & 16) ? state.castEndTick : undefined,
    cast_windup_ticks: (state.mask & 16) ? state.castWindup : undefined, stealthed: (spawn.flags & 2) !== 0 || undefined,
    statuses: state.statuses.map(statusId).filter((id): id is string => !!id),
    flying: (spawn.flags & 4) !== 0 || undefined, form_active: (spawn.flags & 8) !== 0 || undefined,
    hasted: (spawn.flags & 16) !== 0 || undefined, owner_id: ownerId,
    companion_kind: spawn.companionKind || undefined, stance,
  };
}

function mergeEntity(spawn: Spawn, previous: WorldEntityState | undefined, state: DeltaEntity, spawns: Map<number, Spawn>): WorldEntityState {
  const next = { ...(previous ?? entityFrom(spawn, state, spawns)) };
  if (state.mask & 4) next.entity_revision = state.revision;
  if (state.mask & 1) { next.x = dequantize(state.x); next.y = dequantize(state.y); }
  if (state.mask & 2) {
    if (state.vitalsSub & 1) next.hp = state.hp;
    if (state.vitalsSub & 2) next.shield = state.shield;
    if (state.vitalsSub & 4) next.mana = state.mana;
    if (state.vitalsSub & 8) next.thought = state.thought;
  }
  if (state.mask & 16) { next.cast_end_tick = state.castEndTick || undefined; next.cast_windup_ticks = state.castWindup || undefined; }
  if (state.mask & 8) next.statuses = state.statuses.map(statusId).filter((id): id is string => !!id);
  if (state.mask & 32) {
    next.pvp_enabled = (state.flags & 1) !== 0; next.stealthed = (state.flags & 2) !== 0 || undefined;
    next.flying = (state.flags & 4) !== 0 || undefined; next.form_active = (state.flags & 8) !== 0 || undefined;
    next.hasted = (state.flags & 16) !== 0 || undefined;
  }
  if (state.hasExt) {
    if (state.ext & 1) next.ai_state = aiStateName(state.aiState);
    if (state.ext & 2) next.aggro_target_id = state.aggroNetId ? spawns.get(state.aggroNetId)?.entityId : undefined;
    if (state.ext & 16) next.stance = stanceName(state.stance);
  }
  return next;
}

// Protocol display mappings are kept local to the projection implementation.
function kindName(kind: number): WorldEntityState["kind"] {
  return kind === 1 ? "monster" : kind === 2 ? "companion" : "player";
}
function stanceName(stance: number): WorldEntityState["stance"] {
  return stance === 1 ? "FOCUS" : stance === 2 ? "FOLLOW" : "DEFEND";
}
function aiStateName(code: number): string | undefined {
  return (["", "idle", "patrol", "aggro", "chase", "attack", "leash", "dead", "respawn"] as const)[code] || undefined;
}
