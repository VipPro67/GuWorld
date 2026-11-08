import type { WorldEntityState } from "./Protocol.ts";

interface Sample {
  tick: number;
  x: number;
  y: number;
}

const MAX_SAMPLES_PER_ENTITY = 24;
const MIN_INTERPOLATION_DELAY_TICKS = 2;
const MAX_INTERPOLATION_DELAY_TICKS = 6;
const MAX_EXTRAPOLATION_TICKS = 2;
const MAX_EXTRAPOLATION_SAMPLE_GAP_TICKS = 2;
const MAX_EXTRAPOLATION_SPEED_PER_TICK = 1;
const MAX_EXTRAPOLATION_DISTANCE = 1;

/**
 * Tick-based interpolation for remote actors. The render buffer adapts to
 * snapshot gaps, with only short, bounded extrapolation from recent samples.
 */
export class RemoteInterpolator {
  private readonly samples = new Map<string, Sample[]>();
  private lastSnapshotTick: number | null = null;
  private interpolationDelayTicks = MIN_INTERPOLATION_DELAY_TICKS;

  public push(tick: number, entities: WorldEntityState[]): void {
    if (!Number.isFinite(tick)) return;
    if (this.lastSnapshotTick !== null && tick <= this.lastSnapshotTick) return;
    if (this.lastSnapshotTick !== null) {
      const gap = tick - this.lastSnapshotTick;
      if (gap > 1) {
        this.interpolationDelayTicks = Math.min(
          MAX_INTERPOLATION_DELAY_TICKS,
          Math.max(this.interpolationDelayTicks, gap + 1),
        );
      } else {
        this.interpolationDelayTicks = Math.max(
          MIN_INTERPOLATION_DELAY_TICKS,
          this.interpolationDelayTicks - 0.125,
        );
      }
    }
    this.lastSnapshotTick = tick;

    const seen = new Set<string>();
    for (const entity of entities) {
      seen.add(entity.entity_id);
      const list = this.samples.get(entity.entity_id) ?? [];
      list.push({ tick, x: entity.x, y: entity.y });
      if (list.length > MAX_SAMPLES_PER_ENTITY) list.splice(0, list.length - MAX_SAMPLES_PER_ENTITY);
      this.samples.set(entity.entity_id, list);
    }
    for (const id of [...this.samples.keys()]) {
      if (!seen.has(id)) this.samples.delete(id);
    }
  }

  public getInterpolationDelayTicks(): number {
    return this.interpolationDelayTicks;
  }

  public positionAt(entityId: string, renderTick: number): { x: number; y: number } | null {
    const list = this.samples.get(entityId);
    if (!list || list.length === 0) return null;
    if (list.length === 1 || renderTick <= list[0].tick) return { x: list[0].x, y: list[0].y };
    const last = list[list.length - 1];
    if (renderTick >= last.tick) return this.extrapolate(list, renderTick);
    for (let index = 1; index < list.length; index += 1) {
      const next = list[index];
      if (renderTick > next.tick) continue;
      const previous = list[index - 1];
      const span = next.tick - previous.tick;
      const ratio = span <= 0 ? 1 : (renderTick - previous.tick) / span;
      return {
        x: previous.x + (next.x - previous.x) * ratio,
        y: previous.y + (next.y - previous.y) * ratio,
      };
    }
    return { x: last.x, y: last.y };
  }

  public reset(): void {
    this.samples.clear();
    this.lastSnapshotTick = null;
    this.interpolationDelayTicks = MIN_INTERPOLATION_DELAY_TICKS;
  }

  public size(): number {
    return this.samples.size;
  }

  private extrapolate(list: Sample[], renderTick: number): { x: number; y: number } {
    const last = list[list.length - 1];
    const previous = list[list.length - 2];
    if (!previous) return { x: last.x, y: last.y };

    const sampleGap = last.tick - previous.tick;
    const ticksAhead = Math.min(MAX_EXTRAPOLATION_TICKS, renderTick - last.tick);
    if (sampleGap <= 0 || sampleGap > MAX_EXTRAPOLATION_SAMPLE_GAP_TICKS || ticksAhead <= 0) {
      return { x: last.x, y: last.y };
    }

    const velocityX = (last.x - previous.x) / sampleGap;
    const velocityY = (last.y - previous.y) / sampleGap;
    const speed = Math.hypot(velocityX, velocityY);
    const distance = speed * ticksAhead;
    if (speed > MAX_EXTRAPOLATION_SPEED_PER_TICK || distance === 0) {
      return { x: last.x, y: last.y };
    }

    const scale = Math.min(1, MAX_EXTRAPOLATION_DISTANCE / distance);
    return {
      x: last.x + velocityX * ticksAhead * scale,
      y: last.y + velocityY * ticksAhead * scale,
    };
  }
}
