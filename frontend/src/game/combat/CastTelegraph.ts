import Phaser from "phaser";
import { PIXELS_PER_UNIT, worldToPixels, type WorldPoint } from "../world/WorldUnits.ts";
import type { WorldSkillInfo } from "../network/Protocol.ts";

/** Draws authoritative range/shape previews while aiming or hovering a skill. */
export class CastTelegraph {
  private readonly graphics: Phaser.GameObjects.Graphics;

  constructor(scene: Phaser.Scene) {
    this.graphics = scene.add.graphics().setDepth(-6_000);
  }

  /**
   * Draws active aiming indicators (skillshot arrow, ground AoE, target reticle, self aura).
   */
  public draw(skill: WorldSkillInfo, self: WorldPoint, pointer: WorldPoint): void {
    this.graphics.clear();
    const from = worldToPixels(self);
    const raw = worldToPixels(pointer);
    const distance = Math.hypot(pointer.x - self.x, pointer.y - self.y);
    const rangePx = Math.max(16, (skill.range || 3) * PIXELS_PER_UNIT);
    const limited = distance > skill.range && distance > 0 ? rangePx / distance : 1;
    const to = {
      x: from.x + (raw.x - from.x) * limited,
      y: from.y + (raw.y - from.y) * limited,
    };

    if (skill.mode === "DIRECTION") {
      // 1. Max Range Circle boundary
      this.graphics.lineStyle(1.5, 0x38bdf8, 0.45);
      this.graphics.strokeCircle(from.x, from.y, rangePx);
      this.graphics.fillStyle(0x0284c7, 0.03);
      this.graphics.fillCircle(from.x, from.y, rangePx);

      // 2. Skillshot direction corridor & arrow
      const aimAngle = Math.atan2(to.y - from.y, to.x - from.x);
      const widthPx = Math.max(16, (skill.width_units ?? 1.2) * PIXELS_PER_UNIT);
      const perpX = Math.cos(aimAngle + Math.PI / 2) * (widthPx / 2);
      const perpY = Math.sin(aimAngle + Math.PI / 2) * (widthPx / 2);

      // Trajectory end point at maximum range
      const endX = from.x + Math.cos(aimAngle) * rangePx;
      const endY = from.y + Math.sin(aimAngle) * rangePx;

      // Draw corridor shaft
      this.graphics.fillStyle(0x38bdf8, 0.16);
      this.graphics.lineStyle(2, 0x60a5fa, 0.85);
      this.graphics.beginPath();
      this.graphics.moveTo(from.x + perpX, from.y + perpY);
      this.graphics.lineTo(endX + perpX, endY + perpY);
      this.graphics.lineTo(endX - perpX, endY - perpY);
      this.graphics.lineTo(from.x - perpX, from.y - perpY);
      this.graphics.closePath();
      this.graphics.fillPath();
      this.graphics.strokePath();

      // Arrowhead at the tip
      const arrowLength = Math.max(18, widthPx * 1.2);
      const arrowTipX = endX + Math.cos(aimAngle) * arrowLength;
      const arrowTipY = endY + Math.sin(aimAngle) * arrowLength;
      const arrowWingL_X = endX + perpX * 1.8;
      const arrowWingL_Y = endY + perpY * 1.8;
      const arrowWingR_X = endX - perpX * 1.8;
      const arrowWingR_Y = endY - perpY * 1.8;

      this.graphics.fillStyle(0x60a5fa, 0.4);
      this.graphics.lineStyle(2, 0x93c5fd, 0.95);
      this.graphics.beginPath();
      this.graphics.moveTo(arrowTipX, arrowTipY);
      this.graphics.lineTo(arrowWingL_X, arrowWingL_Y);
      this.graphics.lineTo(endX + perpX, endY + perpY);
      this.graphics.lineTo(endX - perpX, endY - perpY);
      this.graphics.lineTo(arrowWingR_X, arrowWingR_Y);
      this.graphics.closePath();
      this.graphics.fillPath();
      this.graphics.strokePath();

      // Center aim line
      this.graphics.lineStyle(1.5, 0xbae6fd, 0.9);
      this.graphics.beginPath();
      this.graphics.moveTo(from.x, from.y);
      this.graphics.lineTo(endX, endY);
      this.graphics.strokePath();
    } else if (skill.mode === "GROUND") {
      // 1. Max Range Circle
      this.graphics.lineStyle(1.5, 0xfde68a, 0.6);
      this.graphics.strokeCircle(from.x, from.y, rangePx);
      this.graphics.fillStyle(0xf59e0b, 0.03);
      this.graphics.fillCircle(from.x, from.y, rangePx);

      // Range line to center of AoE
      this.graphics.lineStyle(1, 0xfde68a, 0.4);
      this.graphics.beginPath();
      this.graphics.moveTo(from.x, from.y);
      this.graphics.lineTo(to.x, to.y);
      this.graphics.strokePath();

      // 2. Specific Ground AoE shapes
      if (skill.shape === "CONE") {
        const aimAngle = Math.atan2(to.y - from.y, to.x - from.x);
        const halfAngle = ((skill.angle_deg ?? 60) * Math.PI) / 360;
        const coneRadius = Math.max(16, (skill.radius_units ?? skill.range) * PIXELS_PER_UNIT);

        this.graphics.fillStyle(0xf59e0b, 0.2);
        this.graphics.lineStyle(2, 0xfbbf24, 0.95);
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

        this.graphics.fillStyle(0xf59e0b, 0.2);
        this.graphics.lineStyle(2, 0xfbbf24, 0.95);
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

        this.graphics.fillStyle(0xf59e0b, 0.2);
        this.graphics.fillCircle(to.x, to.y, radiusPx);
        this.graphics.lineStyle(2, 0xfbbf24, 0.95);
        this.graphics.strokeCircle(to.x, to.y, radiusPx);
        this.graphics.lineStyle(2, 0xef4444, 0.95);
        this.graphics.strokeCircle(to.x, to.y, innerPx);
      } else if (skill.shape === "CROSS") {
        const armPx = (skill.length_units ?? (skill.radius_units ?? 2.5)) * PIXELS_PER_UNIT;
        const widthPx = (skill.width_units ?? 1.5) * PIXELS_PER_UNIT;

        this.graphics.fillStyle(0xf59e0b, 0.2);
        this.graphics.lineStyle(2, 0xfbbf24, 0.95);
        this.graphics.fillRect(to.x - armPx, to.y - widthPx / 2, armPx * 2, widthPx);
        this.graphics.strokeRect(to.x - armPx, to.y - widthPx / 2, armPx * 2, widthPx);
        this.graphics.fillRect(to.x - widthPx / 2, to.y - armPx, widthPx, armPx * 2);
        this.graphics.strokeRect(to.x - widthPx / 2, to.y - armPx, widthPx, armPx * 2);
      } else {
        // Standard Circular AoE
        const radiusPx = Math.max(8, (skill.radius_units ?? 2.5) * PIXELS_PER_UNIT);
        this.graphics.fillStyle(0xf59e0b, 0.2);
        this.graphics.fillCircle(to.x, to.y, radiusPx);
        this.graphics.lineStyle(2, 0xfbbf24, 0.95);
        this.graphics.strokeCircle(to.x, to.y, radiusPx);

        // Center crosshair
        this.graphics.fillStyle(0xfef08a, 0.9);
        this.graphics.fillCircle(to.x, to.y, 3.5);
      }
    } else if (skill.mode === "TARGET") {
      // 1. Max Range Circle
      this.graphics.lineStyle(2, 0x38bdf8, 0.85);
      this.graphics.strokeCircle(from.x, from.y, rangePx);
      this.graphics.fillStyle(0x0284c7, 0.04);
      this.graphics.fillCircle(from.x, from.y, rangePx);

      // Line from player to cursor/target
      this.graphics.lineStyle(1.5, 0x7dd3fc, 0.65);
      this.graphics.beginPath();
      this.graphics.moveTo(from.x, from.y);
      this.graphics.lineTo(to.x, to.y);
      this.graphics.strokePath();

      // Target reticle crosshair at target point
      this.graphics.lineStyle(2, 0x38bdf8, 0.95);
      this.graphics.strokeCircle(to.x, to.y, 16);
      this.graphics.beginPath();
      this.graphics.moveTo(to.x - 22, to.y);
      this.graphics.lineTo(to.x - 9, to.y);
      this.graphics.moveTo(to.x + 9, to.y);
      this.graphics.lineTo(to.x + 22, to.y);
      this.graphics.moveTo(to.x, to.y - 22);
      this.graphics.lineTo(to.x, to.y - 9);
      this.graphics.moveTo(to.x, to.y + 9);
      this.graphics.lineTo(to.x, to.y + 22);
      this.graphics.strokePath();

      this.graphics.fillStyle(0x38bdf8, 0.3);
      this.graphics.fillCircle(to.x, to.y, 4);
    } else if (skill.mode === "SELF") {
      this.graphics.fillStyle(0x34d399, 0.15);
      this.graphics.fillCircle(from.x, from.y, 34);
      this.graphics.lineStyle(2.5, 0x34d399, 0.95);
      this.graphics.strokeCircle(from.x, from.y, 34);
      this.graphics.lineStyle(1.5, 0xa7f3d0, 0.6);
      this.graphics.strokeCircle(from.x, from.y, 44);
    }
  }

  /**
   * Draws a hovering range indicator preview around the player when hovering over a skill button.
   */
  public drawPreview(skill: WorldSkillInfo, self: WorldPoint): void {
    this.graphics.clear();
    const from = worldToPixels(self);
    const rangePx = Math.max(16, (skill.range || 3) * PIXELS_PER_UNIT);

    let color = 0x38bdf8;
    if (skill.mode === "DIRECTION") color = 0x60a5fa;
    else if (skill.mode === "GROUND") color = 0xf59e0b;
    else if (skill.mode === "SELF") color = 0x34d399;

    // Glowing double ring
    this.graphics.fillStyle(color, 0.05);
    this.graphics.fillCircle(from.x, from.y, rangePx);

    this.graphics.lineStyle(2, color, 0.85);
    this.graphics.strokeCircle(from.x, from.y, rangePx);

    this.graphics.lineStyle(1, color, 0.35);
    this.graphics.strokeCircle(from.x, from.y, rangePx + 5);
  }

  public clear(): void {
    this.graphics.clear();
  }
}
