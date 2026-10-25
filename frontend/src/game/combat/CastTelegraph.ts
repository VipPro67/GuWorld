import Phaser from "phaser";
import { PIXELS_PER_UNIT, worldToPixels, type WorldPoint } from "../world/WorldUnits.ts";
import type { WorldSkillInfo } from "../network/Protocol.ts";

/** Draws authoritative range/shape previews while aiming a skill. */
export class CastTelegraph {
  private readonly graphics: Phaser.GameObjects.Graphics;

  constructor(scene: Phaser.Scene) {
    this.graphics = scene.add.graphics().setDepth(-6_000);
  }

  public draw(skill: WorldSkillInfo, self: WorldPoint, pointer: WorldPoint): void {
    this.graphics.clear();
    const from = worldToPixels(self);
    const raw = worldToPixels(pointer);
    const distance = Math.hypot(pointer.x - self.x, pointer.y - self.y);
    const rangePx = skill.range * PIXELS_PER_UNIT;
    const limited = distance > skill.range && distance > 0 ? rangePx / distance : 1;
    const to = {
      x: from.x + (raw.x - from.x) * limited,
      y: from.y + (raw.y - from.y) * limited,
    };
    if (skill.mode === "DIRECTION") {
      this.graphics.lineStyle(3, 0x60a5fa, 0.75);
      this.graphics.beginPath();
      this.graphics.moveTo(from.x, from.y);
      this.graphics.lineTo(to.x, to.y);
      this.graphics.strokePath();
      this.graphics.lineStyle(2, 0x93c5fd, 0.9);
      this.graphics.strokeCircle(to.x, to.y, 6);
    } else if (skill.mode === "GROUND") {
      this.graphics.lineStyle(1, 0xfde68a, 0.5);
      this.graphics.strokeCircle(from.x, from.y, rangePx);

      if (skill.shape === "CONE") {
        const aimAngle = Math.atan2(to.y - from.y, to.x - from.x);
        const halfAngle = ((skill.angle_deg ?? 60) * Math.PI) / 360;
        const coneRadius = Math.max(16, (skill.radius_units ?? skill.range) * PIXELS_PER_UNIT);

        this.graphics.fillStyle(0xf59e0b, 0.15);
        this.graphics.lineStyle(2, 0xfbbf24, 0.9);
        this.graphics.beginPath();
        this.graphics.moveTo(from.x, from.y);
        this.graphics.arc(from.x, from.y, coneRadius, aimAngle - halfAngle, aimAngle + halfAngle, false);
        this.graphics.closePath();
        this.graphics.fillPath();
        this.graphics.strokePath();
      } else if (skill.shape === "RECTANGLE" || skill.shape === "WALL") {
        const aimAngle = Math.atan2(to.y - from.y, to.x - from.x);
        const length = (skill.length_units ?? skill.range) * PIXELS_PER_UNIT;
        const width = (skill.width_units ?? 2) * PIXELS_PER_UNIT;

        const perpX = Math.cos(aimAngle + Math.PI / 2) * (width / 2);
        const perpY = Math.sin(aimAngle + Math.PI / 2) * (width / 2);
        const dirX = Math.cos(aimAngle) * length;
        const dirY = Math.sin(aimAngle) * length;

        this.graphics.fillStyle(0xf59e0b, 0.15);
        this.graphics.lineStyle(2, 0xfbbf24, 0.9);
        this.graphics.beginPath();
        this.graphics.moveTo(from.x + perpX, from.y + perpY);
        this.graphics.lineTo(from.x + dirX + perpX, from.y + dirY + perpY);
        this.graphics.lineTo(from.x + dirX - perpX, from.y + dirY - perpY);
        this.graphics.lineTo(from.x - perpX, from.y - perpY);
        this.graphics.closePath();
        this.graphics.fillPath();
        this.graphics.strokePath();
      } else if (skill.shape === "DONUT") {
        const radiusPx = Math.max(8, (skill.radius_units ?? 2.5) * PIXELS_PER_UNIT);
        const innerPx = Math.max(4, (skill.inner_radius ?? (skill.radius_units ?? 2.5) * 0.5) * PIXELS_PER_UNIT);

        this.graphics.fillStyle(0xf59e0b, 0.15);
        this.graphics.fillCircle(to.x, to.y, radiusPx);
        this.graphics.lineStyle(2, 0xfbbf24, 0.9);
        this.graphics.strokeCircle(to.x, to.y, radiusPx);
        this.graphics.lineStyle(2, 0xef4444, 0.9);
        this.graphics.strokeCircle(to.x, to.y, innerPx);
      } else if (skill.shape === "CROSS") {
        const armPx = (skill.length_units ?? (skill.radius_units ?? 2.5)) * PIXELS_PER_UNIT;
        const widthPx = (skill.width_units ?? 1.5) * PIXELS_PER_UNIT;

        this.graphics.fillStyle(0xf59e0b, 0.15);
        this.graphics.lineStyle(2, 0xfbbf24, 0.9);
        // Horizontal bar
        this.graphics.fillRect(to.x - armPx, to.y - widthPx / 2, armPx * 2, widthPx);
        this.graphics.strokeRect(to.x - armPx, to.y - widthPx / 2, armPx * 2, widthPx);
        // Vertical bar
        this.graphics.fillRect(to.x - widthPx / 2, to.y - armPx, widthPx, armPx * 2);
        this.graphics.strokeRect(to.x - widthPx / 2, to.y - armPx, widthPx, armPx * 2);
      } else {
        const radiusPx = Math.max(8, (skill.radius_units ?? 2.5) * PIXELS_PER_UNIT);
        this.graphics.fillStyle(0xf59e0b, 0.12);
        this.graphics.fillCircle(to.x, to.y, radiusPx);
        this.graphics.lineStyle(2, 0xfbbf24, 0.9);
        this.graphics.strokeCircle(to.x, to.y, radiusPx);
      }
    } else if (skill.mode === "SELF") {
      this.graphics.lineStyle(2, 0x34d399, 0.9);
      this.graphics.strokeCircle(from.x, from.y, 26);
    }
  }

  public clear(): void {
    this.graphics.clear();
  }
}
