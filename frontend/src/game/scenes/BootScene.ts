import Phaser from "phaser";

/**
 * Minimal boot scene. The preview slice has no remote assets yet; only the
 * scene lifecycle itself is smoke-tested here.
 */
export class BootScene extends Phaser.Scene {
  constructor() {
    super({ key: "BootScene" });
  }

  create(): void {
    this.scene.start("PreloadScene");
  }
}
