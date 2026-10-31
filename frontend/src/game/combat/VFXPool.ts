import Phaser from "phaser";
import type { WorldSkillInfo } from "../network/Protocol.ts";
import { PIXELS_PER_UNIT } from "../world/WorldUnits.ts";

const MAX_TEXTS = 64;
const MAX_RINGS = 48;
const MAX_ACTION_MARKS = 32;

export interface FloatingTextOptions {
  fontSize?: string;
  isCrit?: boolean;
  isShield?: boolean;
  isHeal?: boolean;
  isSkill?: boolean;
  jitterX?: boolean;
  offsetY?: number;
}

/** Small object pool for floating combat text and impact rings. */
export class VFXPool {
  private readonly scene: Phaser.Scene;
  private readonly texts: Phaser.GameObjects.Text[] = [];
  private readonly rings: Phaser.GameObjects.Arc[] = [];
  private readonly actionMarks: Phaser.GameObjects.Graphics[] = [];
  private activeTexts = 0;
  private activeRings = 0;
  private activeActionMarks = 0;

  constructor(scene: Phaser.Scene) {
    this.scene = scene;
  }

  public spawnText(x: number, y: number, text: string, color: string, options?: FloatingTextOptions): void {
    if (this.activeTexts >= MAX_TEXTS) return;
    const node = this.texts.find((item) => !item.active) ?? this.createText();
    if (!node) return;
    this.activeTexts += 1;

    const jitter = options?.jitterX !== false ? (Math.random() - 0.5) * 16 : 0;
    const startX = x + jitter;
    const startY = y + (options?.offsetY ?? 0);
    const fontSize = options?.fontSize ?? (options?.isCrit ? "20px" : options?.isSkill ? "13px" : "15px");
    const fontStyle = options?.isSkill ? "italic" : "bold";
    const strokeThickness = options?.isCrit ? 4 : 3;

    node.setActive(true).setVisible(true);
    node.setText(text)
      .setColor(color)
      .setFontSize(fontSize)
      .setFontStyle(fontStyle)
      .setStroke("#020617", strokeThickness)
      .setPosition(startX, startY)
      .setAlpha(1)
      .setDepth(90_000);

    if (options?.isCrit) {
      node.setScale(1.4);
      this.scene.tweens.add({
        targets: node,
        scale: 1.0,
        duration: 140,
        ease: "Back.easeOut",
      });
      this.scene.tweens.add({
        targets: node,
        y: startY - 48,
        alpha: 0,
        duration: 1100,
        ease: "Cubic.easeOut",
        onComplete: () => {
          node.setScale(1).setActive(false).setVisible(false);
          this.activeTexts = Math.max(0, this.activeTexts - 1);
        },
      });
    } else if (options?.isSkill) {
      node.setScale(1.0);
      this.scene.tweens.add({
        targets: node,
        y: startY - 32,
        alpha: 0,
        duration: 1200,
        ease: "Quad.easeOut",
        onComplete: () => {
          node.setScale(1).setActive(false).setVisible(false);
          this.activeTexts = Math.max(0, this.activeTexts - 1);
        },
      });
    } else {
      node.setScale(1.2);
      this.scene.tweens.add({
        targets: node,
        scale: 1.0,
        duration: 110,
        ease: "Quad.easeOut",
      });
      this.scene.tweens.add({
        targets: node,
        y: startY - 40,
        alpha: 0,
        duration: 950,
        ease: "Cubic.easeOut",
        onComplete: () => {
          node.setScale(1).setActive(false).setVisible(false);
          this.activeTexts = Math.max(0, this.activeTexts - 1);
        },
      });
    }
  }

  public spawnRing(x: number, y: number, color: number): void {
    if (this.activeRings >= MAX_RINGS) return;
    const ring = this.rings.find((item) => !item.active) ?? this.createRing();
    if (!ring) return;
    this.activeRings += 1;
    ring.setActive(true).setVisible(true).setPosition(x, y).setScale(0.4).setAlpha(0.9).setDepth(y + 50);
    this.scene.tweens.add({
      targets: ring,
      scale: 2.2,
      alpha: 0,
      duration: 380,
      ease: "Quad.easeOut",
      onComplete: () => {
        ring.setActive(false).setVisible(false);
        this.activeRings = Math.max(0, this.activeRings - 1);
      },
    });
    ring.setStrokeStyle(2, color);
  }

  /** Small placeholder melee swipe, kept deliberately simple for later VFX replacement. */
  public spawnSlash(x: number, y: number, directionX: number, directionY: number): void {
    const length = Math.hypot(directionX, directionY) || 1;
    const dx = directionX / length;
    const dy = directionY / length;
    const mark = this.acquireActionMark();
    if (!mark) return;
    mark.lineStyle(3, 0xffe6a3, 0.95);
    mark.beginPath();
    mark.moveTo(dx * 10 - dy * 18, dy * 10 + dx * 18);
    mark.lineTo(dx * 22 + dy * 3, dy * 22 - dx * 3);
    mark.lineTo(dx * 34 - dy * 11, dy * 34 + dx * 11);
    mark.strokePath();
    this.animateActionMark(mark, x, y, 170, 1.25);
  }

  /** Generic cast pulse around the caster until per-skill VFX are available. */
  public spawnCastPulse(x: number, y: number, color = 0x93c5fd): void {
    const mark = this.acquireActionMark();
    if (!mark) return;
    mark.lineStyle(2, color, 0.9);
    mark.strokeCircle(0, 0, 16);
    mark.lineStyle(1, color, 0.55);
    mark.strokeCircle(0, 0, 23);
    this.animateActionMark(mark, x, y, 340, 1.8);
  }

  /** Confirms a selected world point with a brief crosshair pulse. */
  public spawnTargetMark(x: number, y: number): void {
    const mark = this.acquireActionMark();
    if (!mark) return;
    mark.lineStyle(2, 0xfbbf24, 0.95);
    mark.strokeCircle(0, 0, 13);
    mark.lineBetween(-20, 0, -8, 0);
    mark.lineBetween(8, 0, 20, 0);
    mark.lineBetween(0, -20, 0, -8);
    mark.lineBetween(0, 8, 0, 20);
    this.animateActionMark(mark, x, y, 260, 1.35);
  }

  /** Brief impact preview that follows the skill's authored area shape. */
  public spawnSkillArea(x: number, y: number, skill: WorldSkillInfo): void {
    const mark = this.acquireActionMark();
    if (!mark) return;
    const color = skill.mode === "GROUND" ? 0xfbbf24 : 0x60a5fa;
    const radius = Math.max(10, (skill.radius_units ?? 2.5) * PIXELS_PER_UNIT);
    const length = Math.max(16, (skill.length_units ?? skill.range ?? 3) * PIXELS_PER_UNIT);
    const width = Math.max(10, (skill.width_units ?? 1.5) * PIXELS_PER_UNIT);
    mark.lineStyle(2, color, 0.9);
    mark.fillStyle(color, 0.16);
    switch (skill.shape) {
      case "CONE": {
        const halfAngle = ((skill.angle_deg ?? 60) * Math.PI) / 360;
        mark.beginPath();
        mark.moveTo(0, 0);
        mark.arc(0, 0, radius, -Math.PI / 2 - halfAngle, -Math.PI / 2 + halfAngle, false);
        mark.closePath();
        mark.fillPath();
        mark.strokePath();
        break;
      }
      case "RECTANGLE":
      case "WALL":
        mark.fillRect(-length / 2, -width / 2, length, width);
        mark.strokeRect(-length / 2, -width / 2, length, width);
        break;
      case "DONUT":
        mark.strokeCircle(0, 0, radius);
        mark.strokeCircle(0, 0, Math.max(4, (skill.inner_radius ?? radius / PIXELS_PER_UNIT / 2) * PIXELS_PER_UNIT));
        break;
      case "CROSS":
        mark.fillRect(-length / 2, -width / 2, length, width);
        mark.fillRect(-width / 2, -length / 2, width, length);
        mark.strokeRect(-length / 2, -width / 2, length, width);
        mark.strokeRect(-width / 2, -length / 2, width, length);
        break;
      default:
        mark.fillCircle(0, 0, radius);
        mark.strokeCircle(0, 0, radius);
        break;
    }
    this.animateActionMark(mark, x, y, 420, 1.12);
  }

  public destroy(): void {
    for (const node of this.texts) node.destroy();
    for (const ring of this.rings) ring.destroy();
    for (const mark of this.actionMarks) mark.destroy();
    this.texts.length = 0;
    this.rings.length = 0;
    this.actionMarks.length = 0;
  }

  private createText(): Phaser.GameObjects.Text | null {
    if (this.texts.length >= MAX_TEXTS) return null;
    const node = this.scene.add
      .text(0, 0, "", {
        fontFamily: "monospace",
        fontSize: "15px",
        color: "#fca5a5",
        stroke: "#020617",
        strokeThickness: 3,
      })
      .setOrigin(0.5, 1)
      .setActive(false)
      .setVisible(false);
    this.texts.push(node);
    return node;
  }

  private createRing(): Phaser.GameObjects.Arc | null {
    if (this.rings.length >= MAX_RINGS) return null;
    const ring = this.scene.add
      .circle(0, 0, 18)
      .setStrokeStyle(2, 0xfca5a5, 0.9)
      .setFillStyle(0, 0)
      .setActive(false)
      .setVisible(false);
    this.rings.push(ring);
    return ring;
  }

  private acquireActionMark(): Phaser.GameObjects.Graphics | null {
    if (this.activeActionMarks >= MAX_ACTION_MARKS) return null;
    const mark = this.actionMarks.find((item) => !item.active) ?? (() => {
      if (this.actionMarks.length >= MAX_ACTION_MARKS) return null;
      const graphics = this.scene.add.graphics().setActive(false).setVisible(false);
      this.actionMarks.push(graphics);
      return graphics;
    })();
    if (!mark) return null;
    this.activeActionMarks += 1;
    mark.clear().setPosition(0, 0).setScale(1).setAlpha(1).setActive(true).setVisible(true).setDepth(90_001);
    return mark;
  }

  private animateActionMark(mark: Phaser.GameObjects.Graphics, x: number, y: number, duration: number, scale: number): void {
    mark.setPosition(x, y);
    this.scene.tweens.add({
      targets: mark,
      scale,
      alpha: 0,
      duration,
      ease: "Quad.easeOut",
      onComplete: () => {
        mark.clear().setActive(false).setVisible(false);
        this.activeActionMarks = Math.max(0, this.activeActionMarks - 1);
      },
    });
  }
}
