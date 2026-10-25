import Phaser from "phaser";
import type { WorldRuntime } from "../core/WorldRuntime.ts";

/** F3 debug overlay; polled at 4 Hz, never per frame. */
export class RealtimeDebugHUD {
  private readonly scene: Phaser.Scene;
  private readonly runtime: WorldRuntime;
  private readonly text: Phaser.GameObjects.Text;
  private visible = false;
  private lastUpdate = 0;

  constructor(scene: Phaser.Scene, runtime: WorldRuntime) {
    this.scene = scene;
    this.runtime = runtime;
    this.text = scene.add
      .text(16, 200, "", {
        fontFamily: "monospace",
        fontSize: "11px",
        color: "#bae6fd",
        backgroundColor: "#020617d9",
        padding: { x: 8, y: 6 },
      })
      .setScrollFactor(0)
      .setDepth(120_000)
      .setVisible(false);
  }

  public toggle(): void {
    this.visible = !this.visible;
    this.text.setVisible(this.visible);
    if (this.visible) this.update(true);
  }

  public isVisible(): boolean {
    return this.visible;
  }

  public update(force = false): void {
    if (!this.visible) return;
    const now = this.scene.time.now;
    if (!force && now - this.lastUpdate < 250) return;
    this.lastUpdate = now;
    const stats = this.runtime.getDebugStats();
    const age = stats.network.lastMessageAt > 0 ? `${Date.now() - stats.network.lastMessageAt}ms` : "—";
    const kb = stats.network.bytes / 1024;
    this.text.setText([
      `F3 DEBUG · ${stats.status.toUpperCase()}`,
      `tick ${stats.tick} @ ${stats.tickRate}Hz · entities ${stats.entities} · projectiles ${stats.projectiles}`,
      `self ${stats.selfX.toFixed(2)}, ${stats.selfY.toFixed(2)} · correction ${stats.lastCorrection.toFixed(2)}u`,
      `net msgs ${stats.network.messages} · snapshots ${stats.network.snapshots} · ${kb.toFixed(1)}KB · last ${age}`,
      `fps ${Math.round(this.scene.game.loop.actualFps)}`,
    ].join("\n"));
  }

  public destroy(): void {
    this.text.destroy();
  }
}
