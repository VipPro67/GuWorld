import type { CastPayload, WorldSkillInfo } from "../network/Protocol.ts";

export interface CastPoint {
  x: number;
  y: number;
}

type CastMode = Pick<WorldSkillInfo, "slot" | "mode">;

/** Builds the wire intent after the caller has resolved its targeting policy. */
export function buildCastIntent(
  skill: CastMode,
  caster: CastPoint,
  aimPoint?: CastPoint,
  targetId?: string,
): Omit<CastPayload, "seq"> | null {
  switch (skill.mode) {
    case "SELF":
      return { slot: skill.slot, mode: "SELF" };
    case "TARGET": {
      if (targetId) return { slot: skill.slot, mode: "TARGET", target_id: targetId };
      if (!isPoint(aimPoint)) return null;
      const aimX = aimPoint.x - caster.x;
      const aimY = aimPoint.y - caster.y;
      if (Math.hypot(aimX, aimY) <= 0.01) return null;
      return { slot: skill.slot, mode: "TARGET", aim_x: aimX, aim_y: aimY };
    }
    case "DIRECTION": {
      if (!isPoint(aimPoint)) return null;
      const aimX = aimPoint.x - caster.x;
      const aimY = aimPoint.y - caster.y;
      const length = Math.hypot(aimX, aimY);
      if (length < 0.1) return null;
      return { slot: skill.slot, mode: "DIRECTION", aim_x: aimX / length, aim_y: aimY / length };
    }
    case "GROUND":
      return isPoint(aimPoint)
        ? { slot: skill.slot, mode: "GROUND", x: aimPoint.x, y: aimPoint.y }
        : null;
  }
}

function isPoint(point: CastPoint | undefined): point is CastPoint {
  return point !== undefined && Number.isFinite(point.x) && Number.isFinite(point.y);
}