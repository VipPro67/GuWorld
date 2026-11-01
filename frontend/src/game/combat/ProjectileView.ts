import Phaser from "phaser";
import type { WorldProjectileState } from "../network/Protocol.ts";
import { PIXELS_PER_UNIT, worldToPixels } from "../world/WorldUnits.ts";

/** Renders server-authored projectiles from snapshots (never simulates hits). */
export class ProjectileView {
  private readonly scene: Phaser.Scene;
  private readonly nodes = new Map<number, { core: Phaser.GameObjects.Arc; halo: Phaser.GameObjects.Arc }>();

  constructor(scene: Phaser.Scene) {
    this.scene = scene;
  }

  public sync(projectiles: WorldProjectileState[]): void {
    const seen = new Set<number>();
    for (const projectile of projectiles) {
      seen.add(projectile.id);
      const pixels = worldToPixels({ x: projectile.x, y: projectile.y });
      let nodes = this.nodes.get(projectile.id);
      if (!nodes) {
        const tint = projectileTint(projectile.projectile_vfx);
        const radius = Math.max(4, projectile.radius * PIXELS_PER_UNIT * 0.8);
        const halo = this.scene.add.circle(pixels.x, pixels.y, radius * 1.8, tint, 0.2).setDepth(pixels.y + 38);
        const core = this.scene.add.circle(pixels.x, pixels.y, radius, tint, 0.95)
          .setStrokeStyle(1.5, 0xffffff, 0.85)
          .setDepth(pixels.y + 40);
        nodes = { core, halo };
        this.nodes.set(projectile.id, nodes);
      }
      nodes.core.setPosition(pixels.x, pixels.y).setDepth(pixels.y + 40);
      nodes.halo.setPosition(pixels.x, pixels.y).setDepth(pixels.y + 38);
    }
    for (const [id, node] of this.nodes) {
      if (seen.has(id)) continue;
      node.core.destroy();
      node.halo.destroy();
      this.nodes.delete(id);
    }
  }

  public destroy(): void {
    for (const node of this.nodes.values()) {
      node.core.destroy();
      node.halo.destroy();
    }
    this.nodes.clear();
  }
}

function projectileTint(vfx?: string): number {
  if (vfx?.includes("fireball")) return 0xff793f;
  if (vfx?.includes("spark")) return 0xf5df55;
  if (vfx?.includes("poison")) return 0x8fd653;
  if (vfx?.includes("dark")) return 0xb18cff;
  if (vfx?.includes("moon")) return 0xc5e8ff;
  if (vfx?.includes("wind")) return 0x9be7c1;
  return 0x70d9ee;
}
