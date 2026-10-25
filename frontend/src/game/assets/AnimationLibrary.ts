import type Phaser from "phaser";
import {
  ACTOR_ASSETS,
  ANIMATION_STATES,
  animationFrameKeys,
  animationFrameRate,
  animationKey,
  type ActorAnimation,
  type ActorKind,
} from "./AssetManifest.ts";

/** Registers idempotent Phaser animations for generated actor frames. */
export function registerActorAnimations(scene: Phaser.Scene, kind: ActorKind): void {
  for (const animation of ANIMATION_STATES) {
    const key = animationKey(kind, animation);
    if (scene.anims.exists(key)) continue;
    const frames = animationFrameKeys(kind, animation).map((texture) => ({ key: texture }));
    scene.anims.create({
      key,
      frames,
      frameRate: animationFrameRate(animation),
      repeat: animation === "idle" || animation === "walk" ? -1 : 0,
    });
  }
}

export function actorAnimationKey(kind: ActorKind, animation: ActorAnimation): string {
  return animationKey(kind, animation);
}

export function actorFrameCount(kind: ActorKind, animation: ActorAnimation): number {
  return ACTOR_ASSETS[kind].animations[animation];
}
