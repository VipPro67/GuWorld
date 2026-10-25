import Phaser from "phaser";
import type { WorldProjectileState } from "../network/Protocol.ts";
import { PIXELS_PER_UNIT, worldToPixels } from "../world/WorldUnits.ts";

/** Renders server-authored projectiles from snapshots (never simulates hits). */
export class ProjectileView {
  private readonly scene: Phaser.Scene;
  private readonly nodes = new Map<number, Phaser.GameObjects.Arc>();

  constructor(scene: Phaser.Scene) {
    this.scene = scene;
  }

  public sync(projectiles: WorldProjectileState[]): void {
    const seen = new Set<number>();
    for (const projectile of projectiles) {
      seen.add(projectile.id);
      const pixels = worldToPixels({ x: projectile.x, y: projectile.y });
      let node = this.nodes.get(projectile.id);
      if (!node) {
        node = this.scene.add
          .circle(pixels.x, pixels.y, Math.max(4, projectile.radius * PIXELS_PER_UNIT * 0.8), 0x93c5fd, 0.95)
          .setDepth(pixels.y + 40);
        this.nodes.set(projectile.id, node);
      }
      node.setPosition(pixels.x, pixels.y);
      node.setDepth(pixels.y + 40);
    }
    for (const [id, node] of this.nodes) {
      if (seen.has(id)) continue;
      node.destroy();
      this.nodes.delete(id);
    }
  }

  public destroy(): void {
    for (const node of this.nodes.values()) node.destroy();
    this.nodes.clear();
  }
}
