import Phaser from "phaser";
import type { WorldProjectileState } from "../network/Protocol.ts";
import { PIXELS_PER_UNIT, worldToPixels } from "../world/WorldUnits.ts";

/** Renders server-authored projectiles from snapshots (never simulates hits). */
export class ProjectileView {
  private readonly scene: Phaser.Scene;
  private readonly nodes = new Map<number, ProjectileNodes>();
  private readonly available: ProjectileNodes[] = [];
  private readonly allNodes: ProjectileNodes[] = [];

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
        const acquired = this.acquire();
        if (!acquired) continue;
        nodes = acquired;
        this.nodes.set(projectile.id, acquired);
      }
      const tint = projectileTint(projectile.projectile_vfx);
      const radius = Math.max(4, projectile.radius * PIXELS_PER_UNIT * 0.8);
      if (nodes.tint !== tint || nodes.radius !== radius) {
        nodes.tint = tint;
        nodes.radius = radius;
        nodes.halo.setRadius(radius * 1.8).setFillStyle(tint, 0.2);
        nodes.core.setRadius(radius).setFillStyle(tint, 0.95).setStrokeStyle(1.5, 0xffffff, 0.85);
      }
      nodes.core.setPosition(pixels.x, pixels.y).setDepth(pixels.y + 40);
      nodes.halo.setPosition(pixels.x, pixels.y).setDepth(pixels.y + 38);
    }
    for (const [id, node] of this.nodes) {
      if (seen.has(id)) continue;
      node.core.setActive(false).setVisible(false);
      node.halo.setActive(false).setVisible(false);
      this.nodes.delete(id);
      this.available.push(node);
    }
  }

  public destroy(): void {
    for (const node of this.allNodes) {
      node.core.destroy();
      node.halo.destroy();
    }
    this.nodes.clear();
    this.available.length = 0;
    this.allNodes.length = 0;
  }

  private acquire(): ProjectileNodes | null {
    const pooled = this.available.pop();
    if (pooled) {
      pooled.core.setActive(true).setVisible(true);
      pooled.halo.setActive(true).setVisible(true);
      return pooled;
    }
    if (this.allNodes.length >= MAX_PROJECTILE_NODES) return null;
    const halo = this.scene.add.circle(0, 0, 4).setActive(false).setVisible(false);
    const core = this.scene.add.circle(0, 0, 4).setActive(false).setVisible(false);
    const nodes: ProjectileNodes = { core, halo, tint: -1, radius: -1 };
    this.allNodes.push(nodes);
    core.setActive(true).setVisible(true);
    halo.setActive(true).setVisible(true);
    return nodes;
  }
}

interface ProjectileNodes {
  core: Phaser.GameObjects.Arc;
  halo: Phaser.GameObjects.Arc;
  tint: number;
  radius: number;
}

const MAX_PROJECTILE_NODES = 128;

function projectileTint(vfx?: string): number {
  if (vfx?.includes("fireball")) return 0xff793f;
  if (vfx?.includes("spark")) return 0xf5df55;
  if (vfx?.includes("poison")) return 0x8fd653;
  if (vfx?.includes("dark")) return 0xb18cff;
  if (vfx?.includes("moon")) return 0xc5e8ff;
  if (vfx?.includes("wind")) return 0x9be7c1;
  return 0x70d9ee;
}
