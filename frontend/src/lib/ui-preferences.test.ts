import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { DEFAULT_LOBBY_GRID_ITEMS, LOBBY_MENU_VERSION } from "./lobby-menu.ts";
import { normalizeUiPreferences } from "./ui-preferences.ts";

describe("ui-preferences lobby grid & v1->v2 migration", () => {
  it("defaults to 11 lobby grid items and limit 12 for fresh users", () => {
    const prefs = normalizeUiPreferences(null);
    assert.deepEqual(prefs.lobbyGridItems, [...DEFAULT_LOBBY_GRID_ITEMS]);
    assert.equal(prefs.lobbyGridItems.length, 11);
    assert.equal(prefs.lobbyGridLimit, 12);
    assert.equal(prefs.lobbyMenuVersion, LOBBY_MENU_VERSION);
  });

  it("upgrades v1 preferences with empty hiddenLobbyItems to the 11 default items", () => {
    const prefs = normalizeUiPreferences({
      lobbyItems: ["shop", "character"],
      hiddenLobbyItems: [],
      soundEnabled: false,
    });
    assert.deepEqual(prefs.lobbyGridItems, [...DEFAULT_LOBBY_GRID_ITEMS]);
    assert.equal(prefs.lobbyGridLimit, 12);
    assert.equal(prefs.lobbyMenuVersion, 2);
    assert.equal(prefs.soundEnabled, false);
  });

  it("preserves non-hidden core items and visible groups when migrating v1 with hiddenLobbyItems", () => {
    const prefs = normalizeUiPreferences({
      hiddenLobbyItems: ["pets", "sect", "shop", "premium-store", "auction"],
    });
    assert.equal(prefs.lobbyGridItems.includes("pets"), false);
    assert.equal(prefs.lobbyGridItems.includes("sect"), false);
    assert.equal(prefs.lobbyGridItems.includes("commerce-hub"), false);
    assert.equal(prefs.lobbyGridItems.includes("character"), true);
    assert.equal(prefs.lobbyGridItems.includes("combat-hub"), true);
  });

  it("filters invalid IDs, deduplicates, allows empty lobbyGridItems, and falls back invalid limit", () => {
    const prefs = normalizeUiPreferences({
      lobbyMenuVersion: 2,
      lobbyGridItems: ["character", "non-existent", "character", "combat-hub"],
      lobbyGridLimit: 99,
    });
    assert.deepEqual(prefs.lobbyGridItems, ["character", "combat-hub"]);
    assert.equal(prefs.lobbyGridLimit, 12);

    const emptyPrefs = normalizeUiPreferences({
      lobbyMenuVersion: 2,
      lobbyGridItems: [],
      lobbyGridLimit: 0,
    });
    assert.deepEqual(emptyPrefs.lobbyGridItems, []);
    assert.equal(emptyPrefs.lobbyGridLimit, 0);
  });
});
