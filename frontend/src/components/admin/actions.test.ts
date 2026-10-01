import assert from "node:assert/strict";
import test from "node:test";
import { ADMIN_ACTIONS, canPerformAdminAction } from "./actions.ts";

test("every admin action explicitly declares the complete safety policy", () => {
  for (const [key, action] of Object.entries(ADMIN_ACTIONS)) {
    assert.ok(action.permission, `${key} must declare a permission`);
    assert.ok(["low", "medium", "high", "destructive"].includes(action.riskLevel));
    assert.equal(typeof action.confirmationRequired, "boolean");
    assert.equal(typeof action.reasonRequired, "boolean");
    assert.equal(typeof action.reauthenticationRequired, "boolean");
  }
});

test("known sensitive actions have the expected policies", () => {
  assert.deepEqual(ADMIN_ACTIONS.grantPlayerInventory, { permission: "players.inventory.grant", riskLevel: "high", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false });
  assert.equal(ADMIN_ACTIONS.importAllGameData.riskLevel, "destructive");
  assert.equal(ADMIN_ACTIONS.flushRedis.permission, "cache.flush");
});

test("visibility accepts exact permissions and the admin wildcard", () => {
  assert.equal(canPerformAdminAction(["players.read"], "viewPlayer"), true);
  assert.equal(canPerformAdminAction(["players.read"], "updatePlayer"), false);
  assert.equal(canPerformAdminAction(["admin.*"], "flushRedis"), true);
});
