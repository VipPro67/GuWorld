import assert from "node:assert/strict";
import test from "node:test";
import { EntityAnimator } from "./EntityAnimator.ts";

function makeSprite(width: number, height: number, displayWidth: number, displayHeight: number) {
  const sprite = {
    width, height,
    scaleX: displayWidth / width,
    scaleY: displayHeight / height,
    scene: {
      anims: { exists: () => false },
      tweens: { killTweensOf: () => {}, add: () => {} },
    },
    setPosition() { return this; },
    setRotation() { return this; },
    setAlpha() { return this; },
    setScale(x: number, y = x) { this.scaleX = x; this.scaleY = y; return this; },
    setFlipX() { return this; },
    setTexture() { this.width = 320; this.height = 320; return this; },
    setDisplaySize(w: number, h: number) { this.scaleX = w / this.width; this.scaleY = h / this.height; return this; },
    stop() { return this; },
  };
  return sprite;
}

function assertDisplaySize(sprite: ReturnType<typeof makeSprite>, width: number, height: number) {
  assert.ok(Math.abs(sprite.width * sprite.scaleX - width) < 0.001);
  assert.ok(Math.abs(sprite.height * sprite.scaleY - height) < 0.001);
}

test("first sync and pose changes keep the player's requested display size", () => {
  const sprite = makeSprite(495, 512, 52, 66);
  const animator = new EntityAnimator(sprite as unknown as ConstructorParameters<typeof EntityAnimator>[0], "player");
  animator.setModelScale(1);
  assertDisplaySize(sprite, 52, 66);

  animator.setModelScale(1.4);
  animator.play("walk");
  assertDisplaySize(sprite, 52 * 1.4, 66 * 1.4);
});

test("monster atlas arriving later keeps the current model multiplier", () => {
  const sprite = makeSprite(512, 512, 58, 58);
  const animator = new EntityAnimator(sprite as unknown as ConstructorParameters<typeof EntityAnimator>[0], "monster");
  animator.setModelScale(1.4);
  animator.setStaticTexture("monster-art-test");
  assertDisplaySize(sprite, 58 * 1.4, 58 * 1.4);
});
