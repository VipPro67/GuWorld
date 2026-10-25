import assert from "node:assert/strict";
import test from "node:test";
import { ADMIN_ACTIONS, canPerformAdminAction, prepareAdminAction } from "./actions.ts";
import { ADMIN_NAVIGATION } from "./navigation.ts";

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
  assert.equal(ADMIN_ACTIONS.importAllGameData.permission, "game_data.import_all");
  assert.equal(ADMIN_ACTIONS.importGameDataset.permission, "game_data.import");
  assert.equal(ADMIN_ACTIONS.flushRedis.permission, "cache.flush");
	assert.equal(ADMIN_ACTIONS.flushRedis.reauthenticationRequired, true);
	assert.equal(ADMIN_ACTIONS.adjustPlayerResources.permission, "players.resources.adjust");
	assert.equal(ADMIN_ACTIONS.assignPlayerRoles.permission, "admin_roles.assign");
  assert.equal(ADMIN_ACTIONS.mutatePayment.permission, "billing.mutate");
  assert.deepEqual(ADMIN_ACTIONS.updateArenaRanking, { permission: "arena_rankings.update", riskLevel: "high", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false });
});

test("admin actions accept an explicit reason and confirmation from an in-app form", () => {
  assert.equal(prepareAdminAction("grantPlayerInventory", "cấp Cổ", "admin", { reason: "  ", confirmed: true }), null);
  const prepared = prepareAdminAction("grantPlayerInventory", "cấp Cổ", "admin", { reason: "MMO test", confirmed: true });
  assert.equal(prepared?.reason, "MMO test");
  assert.equal(prepared?.confirmation_target, "admin");
  assert.ok(prepared?.idempotencyKey);
});

test("visibility accepts exact permissions and the admin wildcard", () => {
  assert.equal(canPerformAdminAction(["players.read"], "viewPlayer"), true);
  assert.equal(canPerformAdminAction(["players.read"], "updatePlayer"), false);
  assert.equal(canPerformAdminAction(["admin.*"], "flushRedis"), true);
});

const rolePermissions: Record<string, readonly string[]> = {
  anonymous: [],
  player: [],
  observer: ADMIN_NAVIGATION.filter((item) => item.permission !== "billing.read").map((item) => item.permission),
  support_admin: ["players.read", "players.update", "players.inventory.grant", "players.resources.adjust", "gus.read", "audit.read"],
  content_admin: ["game_data.read", "game_data.import", "game_data.import_all", "cache.reload"],
  game_admin: ["players.read", "players.update", "players.inventory.grant", "players.resources.adjust", "sects.read", "apertures.read", "gus.read", "combat_history.read", "map_drops.read", "map_drops.update", "map_drops.delete", "arena_rankings.read", "arena_rankings.update", "shop.read", "sect_mines.read", "gift_codes.read", "audit.read", "monitoring.read", "world_bosses.read", "world_bosses.update", "world_bosses.delete"],
  finance_viewer: ["billing.read"],
  super_admin: ["admin.*"],
};

test("role matrix exposes only permitted navigation resources", () => {
  const expected: Record<string, readonly string[]> = {
    anonymous: [], player: [],
    observer: ["players", "sects", "apertures", "gu_history", "combat_history", "map_drops", "arena_rankings", "world_bosses", "data", "shop", "mines", "gift_codes", "logs", "monitoring"],
    support_admin: ["players", "gu_history", "logs"],
    content_admin: ["data"],
    game_admin: ["players", "sects", "apertures", "gu_history", "combat_history", "map_drops", "arena_rankings", "world_bosses", "shop", "mines", "gift_codes", "logs", "monitoring"],
    finance_viewer: ["topup_wallet"],
    super_admin: ADMIN_NAVIGATION.map((item) => item.key),
  };
  for (const [role, permissions] of Object.entries(rolePermissions)) {
    const visible = ADMIN_NAVIGATION.filter((item) => permissions.includes("admin.*") || permissions.includes(item.permission)).map((item) => item.key);
    assert.deepEqual(visible, expected[role], role);
  }
});

test("UI action policy hides forbidden actions for every role", () => {
  assert.equal(canPerformAdminAction(rolePermissions.observer, "updatePlayer"), false);
  assert.equal(canPerformAdminAction(rolePermissions.finance_viewer, "adjustPlayerResources"), false);
  assert.equal(canPerformAdminAction(rolePermissions.support_admin, "updatePlayer"), true);
  assert.equal(canPerformAdminAction(rolePermissions.support_admin, "importGameDataset"), false);
  assert.equal(canPerformAdminAction(rolePermissions.content_admin, "importGameDataset"), true);
  assert.equal(canPerformAdminAction(rolePermissions.game_admin, "grantPlayerInventory"), true);
  assert.equal(canPerformAdminAction(rolePermissions.super_admin, "assignPlayerRoles"), true);
});
