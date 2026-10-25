import assert from "node:assert/strict";
import test from "node:test";
import {
  ACTOR_ASSETS,
  ANIMATION_STATES,
  ASSET_VERSION,
  animationFrameKeys,
  animationFrameRate,
  validateAssetManifest,
} from "./AssetManifest.ts";

test("placeholder asset manifest is complete and collision-free", () => {
  assert.ok(ASSET_VERSION.length > 0);
  assert.deepEqual(validateAssetManifest(), []);
  for (const asset of Object.values(ACTOR_ASSETS)) {
    assert.ok(asset.license.length > 0);
    assert.ok(asset.fallbackTexture.length > 0);
    for (const state of ANIMATION_STATES) {
      assert.ok(asset.animations[state] >= 1);
      assert.equal(animationFrameKeys(asset.kind, state).length, asset.animations[state]);
      assert.ok(animationFrameRate(state) > 0);
    }
  }
  assert.equal(new Set(animationFrameKeys("player", "walk")).size, 4);
});

test("placeholder frames use stable animation keys", () => {
  assert.deepEqual(animationFrameKeys("monster", "idle"), ["monster-idle-0", "monster-idle-1"]);
  assert.equal(animationFrameKeys("npc", "walk")[0], "npc-walk-0");
});
