import type Phaser from "phaser";
import type { ActorAnimation, ActorKind } from "../assets/AssetManifest.ts";
import { actorAnimationKey } from "../assets/AnimationLibrary.ts";

/** Maps server movement/combat state to generated actor animations. */
export class EntityAnimator {
  private readonly sprite: Phaser.GameObjects.Sprite;
  private readonly kind: ActorKind;
  private current: ActorAnimation = "idle";
  private facingLeft = false;
  private staticTexture = "";
  private baseScaleX: number;
  private baseScaleY: number;

  constructor(sprite: Phaser.GameObjects.Sprite, kind: ActorKind) {
    this.sprite = sprite;
    this.kind = kind;
    this.baseScaleX = sprite.scaleX;
    this.baseScaleY = sprite.scaleY;
    this.animatePose("idle");
  }

  public update(now: number, moved: boolean, dead: boolean, castUntil: number, hitUntil: number, dx: number): void {
    if (dx < -0.01) this.facingLeft = true;
    else if (dx > 0.01) this.facingLeft = false;
    this.sprite.setFlipX(this.facingLeft);
    let next: ActorAnimation = "idle";
    if (dead) next = "death";
    else if (now < castUntil) next = "cast";
    else if (now < hitUntil) next = "hit";
    else if (moved) next = "walk";
    this.play(next);
  }

  public play(animation: ActorAnimation): void {
    if (this.current === animation) return;
    this.current = animation;
    const key = actorAnimationKey(this.kind, animation);
    if (!this.staticTexture && this.sprite.scene.anims.exists(key)) {
      this.sprite.play(key, true);
    }
    this.animatePose(animation);
  }

  public getState(): ActorAnimation {
    return this.current;
  }

  public getFacingLeft(): boolean {
    return this.facingLeft;
  }

  public faceToward(dx: number): void {
    if (dx < -0.01) this.facingLeft = true;
    else if (dx > 0.01) this.facingLeft = false;
    this.sprite.setFlipX(this.facingLeft);
  }

  public setStaticTexture(key: string): void {
    if (this.staticTexture === key) return;
    this.staticTexture = key;
    this.sprite.stop();
    this.sprite.setTexture(key).setDisplaySize(58, 58);
    this.baseScaleX = this.sprite.scaleX;
    this.baseScaleY = this.sprite.scaleY;
    this.animatePose(this.current);
  }

  private animatePose(animation: ActorAnimation): void {
    const tweens = this.sprite.scene.tweens;
    tweens.killTweensOf(this.sprite);
    this.sprite.setPosition(0, 0).setRotation(0).setAlpha(1).setScale(this.baseScaleX, this.baseScaleY);
    const scaleX = this.baseScaleX;
    const scaleY = this.baseScaleY;
    switch (animation) {
      case "idle":
        tweens.add({ targets: this.sprite, y: -1.5, scaleY: scaleY * 1.025, duration: 650, yoyo: true, repeat: -1, ease: "Sine.easeInOut" });
        break;
      case "walk":
        tweens.add({ targets: this.sprite, y: -3, rotation: 0.045, duration: 110, yoyo: true, repeat: -1, ease: "Sine.easeInOut" });
        break;
      case "cast":
        tweens.add({ targets: this.sprite, y: -4, scaleX: scaleX * 1.07, scaleY: scaleY * 1.06, duration: 180, yoyo: true, repeat: -1, ease: "Sine.easeInOut" });
        break;
      case "hit":
        this.sprite.setAlpha(0.65);
        tweens.add({ targets: this.sprite, x: 3, alpha: 1, duration: 90, yoyo: true, ease: "Quad.easeOut" });
        break;
      case "death":
        this.sprite.setAlpha(0.55).setRotation(0.35).setScale(scaleX, scaleY * 0.72);
        break;
    }
  }
}
