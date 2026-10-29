import Phaser from "phaser";

const MIN_ZOOM = 0.6;
const MAX_ZOOM = 1.6;
const ZOOM_STEP = 0.15;

/**
 * Owns camera follow/deadzone/zoom/shake for the world scene. Camera positions
 * are render-only; they never affect authoritative movement.
 */
export class CameraController {
  private readonly scene: Phaser.Scene;
  private lastBounds = { x: 0, y: 0, width: 0, height: 0 };
  private readonly onWheel = (pointer: Phaser.Input.Pointer, _objects: unknown, _dx: number, dy: number): void => {
    this.setZoom(this.camera.zoom - Math.sign(dy) * ZOOM_STEP);
  };

  constructor(scene: Phaser.Scene) {
    this.scene = scene;
    scene.input.on(Phaser.Input.Events.POINTER_WHEEL, this.onWheel);
  }

  private get camera(): Phaser.Cameras.Scene2D.Camera {
    return this.scene.cameras.main;
  }

  public follow(target: Phaser.GameObjects.Container): void {
    this.camera.startFollow(target, true, 0.22, 0.22);
    const mobile = this.scene.scale.width < 768;
    this.camera.setDeadzone(mobile ? 72 : 140, mobile ? 56 : 90);
  }

  public stopFollow(): void {
    this.camera.stopFollow();
  }

  public setBounds(x: number, y: number, width: number, height: number): void {
    this.lastBounds = { x, y, width, height };
    const viewportWidth = this.scene.scale.width / this.camera.zoom;
    const viewportHeight = this.scene.scale.height / this.camera.zoom;
    this.camera.setBounds(x, y, Math.max(width, viewportWidth), Math.max(height, viewportHeight));
  }

  public setZoom(zoom: number): void {
    this.camera.setZoom(Math.max(MIN_ZOOM, Math.min(MAX_ZOOM, zoom)));
    if (this.lastBounds.width > 0 && this.lastBounds.height > 0) {
      const viewportWidth = this.scene.scale.width / this.camera.zoom;
      const viewportHeight = this.scene.scale.height / this.camera.zoom;
      this.camera.setBounds(
        this.lastBounds.x,
        this.lastBounds.y,
        Math.max(this.lastBounds.width, viewportWidth),
        Math.max(this.lastBounds.height, viewportHeight),
      );
    }
  }

  public getZoom(): number {
    return Math.round(this.camera.zoom * 100) / 100;
  }

  public zoomIn(): number {
    const next = Math.min(MAX_ZOOM, Math.round((this.camera.zoom + ZOOM_STEP) * 100) / 100);
    this.setZoom(next);
    return next;
  }

  public zoomOut(): number {
    const next = Math.max(MIN_ZOOM, Math.round((this.camera.zoom - ZOOM_STEP) * 100) / 100);
    this.setZoom(next);
    return next;
  }

  public shake(intensity = 0.0025): void {
    this.camera.shake(80, intensity, true);
  }

  /** Arrow from screen center toward an offscreen target. */
  public offscreenDirection(targetPx: { x: number; y: number }): { x: number; y: number } | null {
    const view = this.camera.worldView;
    const margin = 24;
    if (targetPx.x >= view.x - margin && targetPx.x <= view.right + margin && targetPx.y >= view.y - margin && targetPx.y <= view.bottom + margin) {
      return null;
    }
    const center = this.camera.midPoint;
    const dx = targetPx.x - center.x;
    const dy = targetPx.y - center.y;
    const length = Math.hypot(dx, dy);
    return length > 0 ? { x: dx / length, y: dy / length } : null;
  }

  public destroy(): void {
    this.scene.input.off(Phaser.Input.Events.POINTER_WHEEL, this.onWheel);
  }
}
