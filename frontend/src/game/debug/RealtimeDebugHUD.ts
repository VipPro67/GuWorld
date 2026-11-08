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
    const age = stats.network.lastSnapshotAt > 0 ? `${Date.now() - stats.network.lastSnapshotAt}ms` : "—";
    const kb = stats.network.bytes / 1024;
    const m = stats.metrics;
    this.text.setText([
      `F3 DEBUG · ${stats.status.toUpperCase()}`,
      `tick ${stats.tick} @ ${stats.tickRate}Hz · entities ${stats.entities} · projectiles ${stats.projectiles}`,
      `interp buffer ${stats.interpolationDelayTicks}t (~${Math.round(stats.interpolationDelayTicks * 1000 / stats.tickRate)}ms)`,
      `self ${stats.selfX.toFixed(2)}, ${stats.selfY.toFixed(2)} · correction ${stats.lastCorrection.toFixed(2)}u`,
      `net msgs ${stats.network.messages} · snapshots ${stats.network.snapshots} · ${kb.toFixed(1)}KB · snap ${age}`,
      `rate ${m.rateHz.toFixed(1)}/s · gap avg ${m.gapAvgMs.toFixed(1)} p95 ${m.gapP95Ms.toFixed(1)} p99 ${m.gapP99Ms.toFixed(1)} max ${m.gapMaxMs.toFixed(0)}ms`,
      `delay ${m.renderDelayAvgMs.toFixed(1)}ms · latency ${m.latencyAvgMs.toFixed(1)}ms · jitter ${m.jitterMs.toFixed(1)}ms`,
      `lost ~${m.estimatedLost} · stalls ${m.stalls} · snaps hard ${m.hardSnaps} · blend ${m.blendCount} · out ${(m.bytesOut / 1024).toFixed(1)}KB`,
      `fps ${Math.round(this.scene.game.loop.actualFps)}`,
    ].join("\n"));
  }

  public destroy(): void {
    this.text.destroy();
  }
}
