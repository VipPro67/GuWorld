import Phaser from "phaser";

const MAX_TEXTS = 64;
const MAX_RINGS = 48;

/** Small object pool for floating combat text and impact rings. */
export class VFXPool {
  private readonly scene: Phaser.Scene;
  private readonly texts: Phaser.GameObjects.Text[] = [];
  private readonly rings: Phaser.GameObjects.Arc[] = [];
  private activeTexts = 0;
  private activeRings = 0;

  constructor(scene: Phaser.Scene) {
    this.scene = scene;
  }

  public spawnText(x: number, y: number, text: string, color: string): void {
    if (this.activeTexts >= MAX_TEXTS) return;
    const node = this.texts.find((item) => !item.active) ?? this.createText();
    if (!node) return;
    this.activeTexts += 1;
    node.setActive(true).setVisible(true);
    node.setText(text).setColor(color).setPosition(x, y).setAlpha(1).setDepth(90_000);
    this.scene.tweens.add({
      targets: node,
      y: y - 38,
      alpha: 0,
      duration: 900,
      ease: "Cubic.easeOut",
      onComplete: () => {
        node.setActive(false).setVisible(false);
        this.activeTexts = Math.max(0, this.activeTexts - 1);
      },
    });
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

  public destroy(): void {
    for (const node of this.texts) node.destroy();
    for (const ring of this.rings) ring.destroy();
    this.texts.length = 0;
    this.rings.length = 0;
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
}
