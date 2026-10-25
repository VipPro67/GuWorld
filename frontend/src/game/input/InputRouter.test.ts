import assert from "node:assert/strict";
import test from "node:test";
import { InputRouter } from "./InputRouter.ts";

test("InputRouter normalizes WASD and arrow diagonals", () => {
  const router = new InputRouter();
  router.setKey("KeyW", true);
  router.setKey("KeyD", true);
  const movement = router.movement();
  assert.ok(Math.abs(movement.x - Math.SQRT1_2) < 1e-12);
  assert.ok(Math.abs(movement.y + Math.SQRT1_2) < 1e-12);
  assert.equal(router.hasMovementKeys(), true);
});

test("InputRouter obeys UI, modal and aiming precedence", () => {
  const router = new InputRouter();
  router.setKey("KeyW", true);
  assert.equal(router.blocker(), "none");
  router.setAiming(true);
  assert.equal(router.blocker(), "aiming");
  assert.deepEqual(router.movement(), { x: 0, y: 0 });
  router.setModal(true);
  assert.equal(router.blocker(), "modal");
  assert.equal(router.canClickMove(), false);
  router.setUIFocused(true);
  assert.equal(router.blocker(), "ui");
  assert.equal(InputRouter.strongest(["aiming", "ui", "none"]), "ui");
});

test("InputRouter reset clears held keys and aim state", () => {
  const router = new InputRouter();
  router.setKey("KeyA", true);
  router.setAiming(true);
  router.reset();
  assert.equal(router.hasMovementKeys(), false);
  assert.equal(router.blocker(), "none");
  assert.deepEqual(router.movement(), { x: 0, y: 0 });
});

test("InputRouter ignores unrelated keys", () => {
  const router = new InputRouter();
  assert.equal(router.setKey("KeyQ", true), false);
  assert.equal(router.hasMovementKeys(), false);
});

test("InputRouter accepts and normalizes joystick vector", () => {
  const router = new InputRouter();
  router.setJoystick({ x: 3, y: 4 });
  assert.equal(router.hasMovementKeys(), true);
  const movement = router.movement();
  assert.ok(Math.abs(movement.x - 0.6) < 1e-6);
  assert.ok(Math.abs(movement.y - 0.8) < 1e-6);

  // Aiming blocks joystick movement too
  router.setAiming(true);
  assert.deepEqual(router.movement(), { x: 0, y: 0 });

  // Reset clears joystick
  router.reset();
  assert.equal(router.hasMovementKeys(), false);
  assert.deepEqual(router.movement(), { x: 0, y: 0 });
});
