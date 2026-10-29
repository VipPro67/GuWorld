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

  constructor(sprite: Phaser.GameObjects.Sprite, kind: ActorKind) {
    this.sprite = sprite;
    this.kind = kind;
    this.play("idle");
  }

  public update(now: number, moved: boolean, dead: boolean, castUntil: number, hitUntil: number, dx: number): void {
    if (dx < -0.01) this.facingLeft = true;
    else if (dx > 0.01) this.facingLeft = false;
    this.sprite.setFlipX(this.facingLeft);
    if (this.staticTexture) return;
    let next: ActorAnimation = "idle";
    if (dead) next = "death";
    else if (now < castUntil) next = "cast";
    else if (now < hitUntil) next = "hit";
    else if (moved) next = "walk";
    this.play(next);
  }

  public play(animation: ActorAnimation): void {
    if (this.staticTexture) return;
    if (this.current === animation) return;
    this.current = animation;
    const key = actorAnimationKey(this.kind, animation);
    if (this.sprite.scene.anims.exists(key)) {
      this.sprite.play(key, true);
    }
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
  }
}
