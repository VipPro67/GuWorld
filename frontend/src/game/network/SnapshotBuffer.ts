import {
  parseWorldSnapshotPayload,
  type WorldEntityState,
  type WorldProjectileState,
  type WorldSnapshotPayload,
} from "./Protocol.ts";

export interface BufferedWorldState {
  tick: number;
  ackSeq: number;
  baselineId: number;
  baseTick: number;
  entities: WorldEntityState[];
  projectiles: WorldProjectileState[];
}

export type SnapshotApplyResult =
  | { status: "applied"; state: BufferedWorldState }
  | { status: "ignored"; reason: "stale_tick" | "old_baseline" }
  | { status: "resync_required"; reason: string };

/** Applies full AOI baselines and baseline-relative deltas from protocol v1. */
export class SnapshotBuffer {
  private baselineId = 0;
  private baseTick = 0;
  private lastTick = -1;
  private ackSeq = 0;
  private entities = new Map<string, WorldEntityState>();
  private projectiles: WorldProjectileState[] = [];

  public apply(input: WorldSnapshotPayload | unknown): SnapshotApplyResult {
    let snapshot: WorldSnapshotPayload;
    try {
      snapshot = parseWorldSnapshotPayload(input);
    } catch (error) {
      return { status: "resync_required", reason: error instanceof Error ? error.message : "Invalid snapshot" };
    }

    if (snapshot.is_baseline) return this.applyBaseline(snapshot);
    return this.applyDelta(snapshot);
  }

  public getState(): BufferedWorldState | null {
    if (this.baselineId === 0) return null;
    return this.stateView();
  }

  public reset(): void {
    this.baselineId = 0;
    this.baseTick = 0;
    this.lastTick = -1;
    this.ackSeq = 0;
    this.entities.clear();
    this.projectiles = [];
  }

  private applyBaseline(snapshot: WorldSnapshotPayload): SnapshotApplyResult {
    if (snapshot.base_tick !== undefined && snapshot.base_tick !== snapshot.tick) {
      return { status: "resync_required", reason: "Baseline tick does not match base_tick" };
    }
    if (snapshot.tick <= this.lastTick) return { status: "ignored", reason: "stale_tick" };
    if (snapshot.baseline_id < this.baselineId) return { status: "ignored", reason: "old_baseline" };
    if (snapshot.baseline_id === this.baselineId && this.baselineId !== 0) {
      return { status: "resync_required", reason: "Baseline ID was reused for different state" };
    }
    if (snapshot.ack_seq < this.ackSeq) return { status: "resync_required", reason: "Baseline ack_seq moved backwards" };

    this.entities = new Map(snapshot.entities.map((entity) => [entity.entity_id, { ...entity }]));
    this.baselineId = snapshot.baseline_id;
    this.baseTick = snapshot.tick;
    this.lastTick = snapshot.tick;
    this.ackSeq = snapshot.ack_seq;
    this.projectiles = (snapshot.projectiles ?? []).map((projectile) => ({ ...projectile }));
    return { status: "applied", state: this.stateView() };
  }

  private applyDelta(snapshot: WorldSnapshotPayload): SnapshotApplyResult {
    if (this.baselineId === 0) return { status: "resync_required", reason: "Delta arrived before a baseline" };
    if (snapshot.baseline_id < this.baselineId) return { status: "ignored", reason: "old_baseline" };
    if (snapshot.baseline_id !== this.baselineId || snapshot.base_tick !== this.baseTick) {
      return { status: "resync_required", reason: "Delta references an unknown baseline" };
    }
    if (snapshot.tick <= this.lastTick) return { status: "ignored", reason: "stale_tick" };
    if (snapshot.ack_seq < this.ackSeq) return { status: "resync_required", reason: "Delta ack_seq moved backwards" };

    const next = new Map(this.entities);
    for (const entity of snapshot.entities) {
      const previous = next.get(entity.entity_id);
      if (previous && entity.entity_revision < previous.entity_revision) {
        return { status: "resync_required", reason: `Entity revision moved backwards: ${entity.entity_id}` };
      }
      next.set(entity.entity_id, { ...entity });
    }
    for (const entityId of snapshot.removed ?? []) next.delete(entityId);

    this.entities = next;
    this.lastTick = snapshot.tick;
    this.ackSeq = snapshot.ack_seq;
    this.projectiles = (snapshot.projectiles ?? []).map((projectile) => ({ ...projectile }));
    return { status: "applied", state: this.stateView() };
  }

  private stateView(): BufferedWorldState {
    return {
      tick: this.lastTick,
      ackSeq: this.ackSeq,
      baselineId: this.baselineId,
      baseTick: this.baseTick,
      entities: [...this.entities.values()]
        .map((entity) => ({ ...entity }))
        .sort((left, right) => left.entity_id.localeCompare(right.entity_id)),
      projectiles: this.projectiles.map((projectile) => ({ ...projectile })),
    };
  }
}
