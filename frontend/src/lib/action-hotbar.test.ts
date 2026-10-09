import test from "node:test";
import assert from "node:assert/strict";
import { dispatchHotbarAction, normalizeHotbarPreferences, reorderHotbarIds } from "./action-hotbar.ts";

test("default hotbar pins open inventory, Gu, cultivation, character and map in one dispatch", () => {
  const defaults = normalizeHotbarPreferences(null, ["inventory", "inventory-gu", "cultivation", "practice-room", "character", "world-atlas"]);
  assert.deepEqual(defaults.pinnedIds, ["inventory", "inventory-gu", "cultivation", "character", "world-atlas"]);
  const opened: string[] = [];
  const handlers = {
    setView: (id: string) => opened.push(id),
    openGeneralSettings: () => opened.push("general-settings"),
    openTeleport: () => opened.push("teleport"),
    openWorldAtlas: () => opened.push("world-atlas"),
    openTeleportDirectory: () => opened.push("teleport-directory"),
    openTeleportBuild: () => opened.push("teleport-build"),
    openDivination: () => opened.push("divination"),
    openActivityLog: () => opened.push("activity-log"),
  };
  for (const id of defaults.pinnedIds) dispatchHotbarAction(id, handlers);
  assert.deepEqual(opened, defaults.pinnedIds);
  dispatchHotbarAction("practice-room", handlers);
  assert.equal(opened.at(-1), "practice-room", "Phòng Tập stays a separate action");
});

test("custom modal actions call their existing handler once without a category step", () => {
  const calls: string[] = [];
  const handlers = {
    setView: (id: string) => calls.push(`view:${id}`),
    openGeneralSettings: () => calls.push("settings"),
    openTeleport: () => calls.push("teleport"),
    openWorldAtlas: () => calls.push("atlas"),
    openTeleportDirectory: () => calls.push("directory"),
    openTeleportBuild: () => calls.push("build"),
    openDivination: () => calls.push("divination"),
    openActivityLog: () => calls.push("activity-log"),
  };
  const cases = [
    ["general-settings", "settings"], ["teleport", "teleport"], ["world-atlas", "atlas"],
    ["teleport-directory", "directory"], ["teleport-build", "build"],
    ["divination", "divination"], ["activity-log", "activity-log"],
  ];
  for (const [id, expected] of cases) {
    calls.length = 0;
    dispatchHotbarAction(id, handlers);
    assert.deepEqual(calls, [expected], `${id} should dispatch directly to its existing handler`);
  }
});

test("preferences discard deleted actions and preserve compact, visibility and hotkeys", () => {
  assert.deepEqual(normalizeHotbarPreferences({
    pinnedIds: ["inventory", "removed", "inventory", 4], compact: true, visible: false,
    hotkeys: { inventory: "i", removed: "r", "inventory-gu": "g" },
  }, ["inventory", "inventory-gu"]), {
    pinnedIds: ["inventory"], compact: true, visible: false,
    hotkeys: { inventory: "i", "inventory-gu": "g" },
  });
});

test("legacy Tu Luyện pin migrates from practice room to cultivation", () => {
  const preferences = normalizeHotbarPreferences({ pinnedIds: ["inventory", "practice-room"] }, ["inventory", "cultivation", "practice-room"]);
  assert.deepEqual(preferences.pinnedIds, ["inventory", "cultivation"]);
});

test("shortcut order can be moved and ignores missing or identical ids", () => {
  assert.deepEqual(reorderHotbarIds(["inventory", "practice-room", "character"], "character", "inventory"), ["character", "inventory", "practice-room"]);
  const ids = ["inventory", "practice-room"];
  assert.equal(reorderHotbarIds(ids, "missing", "inventory"), ids);
  assert.equal(reorderHotbarIds(ids, "inventory", "inventory"), ids);
});
