import assert from "node:assert/strict";
import test from "node:test";
import { centeredCameraBounds } from "./cameraBounds.ts";

test("a small private map is centered within a larger viewport", () => {
  assert.deepEqual(centeredCameraBounds(0, 0, 960, 960, 1877, 1000), {
    x: -458.5, y: -20, width: 1877, height: 1000,
  });
});

test("a large map keeps its own bounds", () => {
  assert.deepEqual(centeredCameraBounds(0, 0, 24000, 24000, 400, 900), {
    x: 0, y: 0, width: 24000, height: 24000,
  });
});
