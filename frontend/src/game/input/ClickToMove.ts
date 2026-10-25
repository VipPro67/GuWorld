import type { WorldPoint } from "../world/WorldUnits.ts";

export interface MoveTargetRequest {
  requestId: number;
  target: WorldPoint;
}

/**
 * Client-side intent for click-to-move. The destination is only a request; the
 * server validates/clamps it and owns the resulting movement. The marker is
 * cleared when the predicted self position reaches it or another input wins.
 */
export class ClickToMove {
  private target: WorldPoint | null = null;
  private requestId = 0;

  public begin(target: WorldPoint): MoveTargetRequest | null {
    if (!Number.isFinite(target.x) || !Number.isFinite(target.y)) return null;
    if (Math.abs(target.x) > 1_000_000 || Math.abs(target.y) > 1_000_000) return null;
    this.requestId += 1;
    this.target = { ...target };
    return { requestId: this.requestId, target: { ...this.target } };
  }

  public cancel(): void {
    this.target = null;
  }

  public getTarget(): WorldPoint | null {
    return this.target ? { ...this.target } : null;
  }

  public hasTarget(): boolean {
    return this.target !== null;
  }

  /** Returns true when the actor is close enough to drop the destination marker. */
  public completeIfNear(position: WorldPoint, epsilon = 0.35): boolean {
    if (!this.target) return false;
    if (Math.hypot(this.target.x - position.x, this.target.y - position.y) > epsilon) return false;
    this.target = null;
    return true;
  }
}
