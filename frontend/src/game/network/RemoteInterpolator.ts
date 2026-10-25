import type { WorldEntityState } from "./Protocol.ts";

interface Sample {
  tick: number;
  x: number;
  y: number;
}

const MAX_SAMPLES_PER_ENTITY = 24;

/**
 * Tick-based interpolation for remote actors. Rendering lags the newest
 * snapshot by a small buffer, so movement stays smooth without trusting client
 * positions. There is no extrapolation beyond the newest sample.
 */
export class RemoteInterpolator {
  private readonly samples = new Map<string, Sample[]>();

  public push(tick: number, entities: WorldEntityState[]): void {
    if (!Number.isFinite(tick)) return;
    const seen = new Set<string>();
    for (const entity of entities) {
      seen.add(entity.entity_id);
      const list = this.samples.get(entity.entity_id) ?? [];
      if (list.length > 0 && tick <= list[list.length - 1].tick) continue;
      list.push({ tick, x: entity.x, y: entity.y });
      if (list.length > MAX_SAMPLES_PER_ENTITY) list.splice(0, list.length - MAX_SAMPLES_PER_ENTITY);
      this.samples.set(entity.entity_id, list);
    }
    for (const id of [...this.samples.keys()]) {
      if (!seen.has(id)) this.samples.delete(id);
    }
  }

  public positionAt(entityId: string, renderTick: number): { x: number; y: number } | null {
    const list = this.samples.get(entityId);
    if (!list || list.length === 0) return null;
    if (list.length === 1 || renderTick <= list[0].tick) return { x: list[0].x, y: list[0].y };
    const last = list[list.length - 1];
    if (renderTick >= last.tick) return { x: last.x, y: last.y };
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
  }

  public size(): number {
    return this.samples.size;
  }
}
