import assert from "node:assert/strict";
import test from "node:test";
import { MotionController, type MotionHost } from "./MotionController.ts";
import type { WorldPoint } from "../world/WorldUnits.ts";

class RecordingHost implements MotionHost {
  public sent: WorldPoint[] = [];
  public cancels = 0;
  public self: WorldPoint = { x: 0, y: 0 };
  public active: WorldPoint | null = null;

  public sendMoveTarget(target: WorldPoint): boolean {
    this.sent.push({ ...target });
    this.active = { ...target };
    return true;
  }

  public cancelMoveTarget(): void {
    this.cancels += 1;
    this.active = null;
  }

  public getSelfPosition(): WorldPoint {
    return this.self;
  }

  public getActiveMoveTarget(): WorldPoint | null {
    return this.active;
  }
}

test("navigateTo drops the current position and sends the first waypoint", () => {
  const host = new RecordingHost();
  const motion = new MotionController(host);
  const ok = motion.navigateTo(
    { x: 10, y: 0 },
    [{ x: 0, y: 0 }, { x: 5, y: 0 }, { x: 10, y: 0 }],
  );
  assert.equal(ok, true);
  assert.deepEqual(host.sent, [{ x: 5, y: 0 }]);
  assert.deepEqual(motion.waypoints(), [{ x: 5, y: 0 }, { x: 10, y: 0 }]);
  assert.deepEqual(motion.goal(), { x: 10, y: 0 });
});

test("navigateTo suppresses a duplicate send when the active target matches", () => {
  const host = new RecordingHost();
  const motion = new MotionController(host);
  const path = [{ x: 0, y: 0 }, { x: 5, y: 0 }];
  motion.navigateTo({ x: 5, y: 0 }, path);
  motion.navigateTo({ x: 5, y: 0 }, path);
  assert.equal(host.sent.length, 1);
});

test("tick advances at the lookahead threshold and arrival epsilon", () => {
  const host = new RecordingHost();
  host.self = { x: 0, y: 0 };
  const motion = new MotionController(host);
  motion.navigateTo(
    { x: 10, y: 0 },
    [{ x: 0, y: 0 }, { x: 5, y: 0 }, { x: 10, y: 0 }],
  );
  assert.deepEqual(host.sent, [{ x: 5, y: 0 }]);

  // Within lookahead but not arrived: dedup, no packet.
  host.self = { x: 4.4, y: 0 };
  motion.tick();
  assert.deepEqual(host.sent, [{ x: 5, y: 0 }]);

  // Beyond arrival epsilon: shift to the next waypoint and send it.
  host.self = { x: 4.9, y: 0 };
  motion.tick();
  assert.deepEqual(host.sent, [{ x: 5, y: 0 }, { x: 10, y: 0 }]);
  assert.deepEqual(motion.waypoints(), [{ x: 10, y: 0 }]);
});

test("settling into the waypoint queue end clears the plan without a network cancel", () => {
  const host = new RecordingHost();
  host.self = { x: 0, y: 0 };
  const motion = new MotionController(host);
  motion.navigateTo({ x: 2, y: 0 }, [{ x: 0, y: 0 }, { x: 2, y: 0 }]);
  host.self = { x: 2, y: 0 };
  motion.tick();
  assert.deepEqual(motion.waypoints(), []);
  assert.equal(motion.clickTarget(), null);
  assert.equal(host.cancels, 0);
});

test("single-hop navigation completes its marker near arrival", () => {
  const host = new RecordingHost();
  host.self = { x: 0, y: 0 };
  const motion = new MotionController(host);
  motion.navigateTo({ x: 3, y: 0 }, null);
  assert.deepEqual(host.sent, [{ x: 3, y: 0 }]);
  assert.deepEqual(motion.clickTarget(), { x: 3, y: 0 });
  host.self = { x: 3, y: 0 };
  motion.tick();
  assert.equal(motion.clickTarget(), null);
});
test("a nonzero manual vector drops click and approach plans", () => {
  const host = new RecordingHost();
  const motion = new MotionController(host);
  motion.navigateTo({ x: 5, y: 0 }, [{ x: 0, y: 0 }, { x: 5, y: 0 }]);
  motion.setManualVector({ x: 1, y: 0 });
  assert.deepEqual(motion.waypoints(), []);
  assert.equal(motion.clickTarget(), null);

  motion.approach({ x: 5, y: 0 }, null, 0);
  motion.setManualVector({ x: 0, y: 1 });
  assert.equal(motion.isApproaching(), false);
});

test("a zero manual vector keeps the active plan", () => {
  const host = new RecordingHost();
  const motion = new MotionController(host);
  motion.navigateTo({ x: 5, y: 0 }, [{ x: 0, y: 0 }, { x: 5, y: 0 }]);
  motion.setManualVector({ x: 0, y: 0 });
  assert.equal(motion.waypoints().length, 1);
});

test("approach with a path keeps the plan inside the throttle window", () => {
  const host = new RecordingHost();
  const motion = new MotionController(host);
  const ok = motion.approach({ x: 10, y: 0 }, [{ x: 0, y: 0 }, { x: 5, y: 0 }], 1_000);
  assert.equal(ok, true);
  assert.deepEqual(host.sent, [{ x: 5, y: 0 }]);
  assert.equal(motion.isApproaching(), true);
  assert.deepEqual(motion.clickTarget(), { x: 10, y: 0 });

  // Re-approach while throttled: kept, no new packet.
  assert.equal(motion.approach({ x: 12, y: 0 }, [{ x: 0, y: 0 }, { x: 7, y: 0 }], 1_100), true);
  assert.deepEqual(host.sent, [{ x: 5, y: 0 }]);

  // After the window expires the new leg wins.
  motion.approach({ x: 12, y: 0 }, [{ x: 0, y: 0 }, { x: 7, y: 0 }], 2_000);
  assert.deepEqual(host.sent, [{ x: 5, y: 0 }, { x: 7, y: 0 }]);
});

test("approach without a map single-hops and retries once the server target drops", () => {
  const host = new RecordingHost();
  host.self = { x: 0, y: 0 };
  const motion = new MotionController(host);
  motion.approach({ x: 4, y: 0 }, null, 1_000);
  assert.deepEqual(host.sent, [{ x: 4, y: 0 }]);
  assert.equal(motion.clickTarget(), null);

  // Throttled while the server still tracks the move.
  motion.approach({ x: 4, y: 0 }, null, 1_100);
  assert.equal(host.sent.length, 1);

  // Target dropped (runtime clears its own localTarget on arrival): immediate retry with the fresh goal.
  host.active = null;
  motion.approach({ x: 6, y: 0 }, null, 1_200);
  assert.deepEqual(host.sent, [{ x: 4, y: 0 }, { x: 6, y: 0 }]);
});

test("disengage stops the server move unconditionally", () => {
  const host = new RecordingHost();
  const motion = new MotionController(host);
  motion.navigateTo({ x: 5, y: 0 }, [{ x: 0, y: 0 }, { x: 5, y: 0 }]);
  motion.disengage();
  assert.equal(host.cancels, 1);
  assert.deepEqual(motion.waypoints(), []);
});

test("stopApproach only cancels a live approach leg", () => {
  const host = new RecordingHost();
  const motion = new MotionController(host);
  motion.stopApproach();
  assert.equal(host.cancels, 0);

  motion.approach({ x: 4, y: 0 }, null, 0);
  motion.stopApproach();
  assert.equal(host.cancels, 1);
  assert.equal(motion.isApproaching(), false);
});

test("stopAll clears the plan and asks the server to stand still", () => {
  const host = new RecordingHost();
  host.self = { x: 0, y: 0 };
  const motion = new MotionController(host);
  motion.navigateTo({ x: 5, y: 0 }, [{ x: 0, y: 0 }, { x: 5, y: 0 }]);
  motion.stopAll();
  assert.equal(host.cancels, 1);
  assert.deepEqual(motion.waypoints(), []);
  assert.equal(motion.clickTarget(), null);
});

test("navigateTo rejects a non-finite destination", () => {
  const host = new RecordingHost();
  const motion = new MotionController(host);
  assert.equal(motion.navigateTo({ x: Number.NaN, y: 0 }, null), false);
  assert.deepEqual(host.sent, []);
});
