import type { WorldPoint } from "../world/WorldUnits.ts";
import { ClickToMove } from "../input/ClickToMove.ts";

export interface MotionHost {
  /** Issue a server move-target request. Returns false when the world is not joined. */
  sendMoveTarget(target: WorldPoint): boolean;
  /** Cancel the active move target by requesting the current position. */
  cancelMoveTarget(): void;
  getSelfPosition(): WorldPoint;
  /** Server-side tracked target used to suppress duplicate waypoint sends. */
  getActiveMoveTarget(): WorldPoint | null;
}

export interface ApproachOptions {
  /** Minimum ms between re-issuing an approach move while a plan is in flight. */
  throttleMs?: number;
}

const WAYPOINT_ARRIVAL_EPSILON = 0.4;
const WAYPOINT_LOOKAHEAD = 1.2;
const APPROACH_THROTTLE_MS = 900;

/**
 * Deep module arbitrating the three independent movement sources (manual
 * steer vectors, click-to-move A* plans and auto-farm approach legs) behind a
 * small intent interface. Priority is fixed: Manual > Click > Approach.
 *
 * The controller owns the waypoint queue, arrival epsilon, lookahead re-issue
 * and network throttling; the scene only issues intents and renders markers
 * from the accessor view.
 */
export class MotionController {
  private readonly host: MotionHost;
  private readonly click = new ClickToMove();
  private queue: WorldPoint[] = [];
  private goalPoint: WorldPoint | null = null;
  private approachMoving = false;
  private nextApproachAt = 0;

  constructor(host: MotionHost) {
    this.host = host;
  }

  /**
   * Manual steering outranks everything: a nonzero vector drops any click or
   * approach plan. The runtime clears its own target when it receives the
   * vector, so no extra network cancel is needed here.
   */
  public setManualVector(vector: WorldPoint): void {
    if (vector.x === 0 && vector.y === 0) return;
    this.clearPlan();
    this.approachMoving = false;
  }

  /**
   * Start a click navigation plan. `path` carries the found A* route (its
   * first point is the current position and is dropped); pass null when no
   * map is loaded and the destination is sent as one hop.
   */
  public navigateTo(goal: WorldPoint, path: WorldPoint[] | null): boolean {
    this.clearPlan();
    if (!this.click.begin(goal)) return false;
    this.goalPoint = goal;
    if (path) {
      this.queue = path.slice(1);
      return this.sendNextWaypoint();
    }
    // Open-map fallback: single hop with no waypoint queue.
    return this.host.sendMoveTarget(goal);
  }

  /**
   * Auto-farm approach leg toward a monster. While a plan is in flight it is
   * kept until the throttle window expires so chasing does not spam
   * move-target packets each frame; a dropped or rejected plan is retried
   * immediately.
   */
  public approach(goal: WorldPoint, path: WorldPoint[] | null, now: number, options: ApproachOptions = {}): boolean {
    const throttleMs = options.throttleMs ?? APPROACH_THROTTLE_MS;
    const planInFlight = this.queue.length > 0
      || (this.approachMoving && this.host.getActiveMoveTarget() !== null);
    if (now < this.nextApproachAt && planInFlight) return true;
    this.nextApproachAt = now + throttleMs;
    this.clearPlan();
    if (!path) {
      // Open-map fallback: single hop with no destination marker.
      this.approachMoving = this.host.sendMoveTarget(goal);
      return this.approachMoving;
    }
    this.click.begin(goal);
    this.queue = path.slice(1);
    this.goalPoint = goal;
    this.approachMoving = true;
    return this.sendNextWaypoint();
  }

  /** Caught-up stop inside attack range: unconditionally ask the server to stop. */
  public disengage(): void {
    this.clearPlan();
    this.approachMoving = false;
    this.host.cancelMoveTarget();
  }

  /** Stop only the approach leg; click navigation plans are left untouched. */
  public stopApproach(): void {
    if (!this.approachMoving) return;
    this.approachMoving = false;
    this.clearPlan();
    this.host.cancelMoveTarget();
  }

  /** Drop the whole plan and tell the server to stand still (Esc, blur, rejected move). */
  public stopAll(): void {
    this.clearPlan();
    this.host.cancelMoveTarget();
  }

  /**
   * Per-frame hook: advances the waypoint plan (lookahead re-issue) and
   * completes the single-hop destination marker near arrival.
   */
  public tick(): void {
    if (this.queue.length === 0) {
      if (this.click.hasTarget()) this.click.completeIfNear(this.host.getSelfPosition());
      return;
    }
    const self = this.host.getSelfPosition();
    const next = this.queue[0];
    if (Math.hypot(next.x - self.x, next.y - self.y) <= WAYPOINT_LOOKAHEAD) {
      this.sendNextWaypoint();
    }
  }

  // ---------- accessor view (for the scene's marker/HUD rendering) ----------

  public waypoints(): WorldPoint[] {
    return this.queue.map((point) => ({ ...point }));
  }

  public goal(): WorldPoint | null {
    return this.goalPoint ? { ...this.goalPoint } : null;
  }

  public clickTarget(): WorldPoint | null {
    return this.click.getTarget();
  }

  public isApproaching(): boolean {
    return this.approachMoving;
  }

  // ---------- internals ----------

  private sendNextWaypoint(): boolean {
    const self = this.host.getSelfPosition();
    while (
      this.queue.length > 0
      && Math.hypot(this.queue[0].x - self.x, this.queue[0].y - self.y) <= WAYPOINT_ARRIVAL_EPSILON
    ) {
      this.queue.shift();
    }
    if (this.queue.length === 0) {
      this.clearPlan();
      return false;
    }
    const target = this.queue[0];
    const active = this.host.getActiveMoveTarget();
    if (active && Math.hypot(active.x - target.x, active.y - target.y) < 0.001) return true;
    return this.host.sendMoveTarget(target);
  }

  private clearPlan(): void {
    this.queue = [];
    this.goalPoint = null;
    this.click.cancel();
  }
}
