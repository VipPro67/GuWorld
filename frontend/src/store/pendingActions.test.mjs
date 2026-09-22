import assert from "node:assert/strict";
import test from "node:test";
import { acquirePendingAction, releasePendingAction } from "./pendingActions.ts";

function deferred() {
  let resolve;
  const promise = new Promise((done) => { resolve = done; });
  return { promise, resolve };
}

function createActionRunner() {
  let pending = {};
  return {
    get pending() { return pending; },
    async run(key, action) {
      const acquired = acquirePendingAction(pending, key);
      if (!acquired) return false;
      pending = acquired;
      try {
        await action();
        return true;
      } finally {
        pending = releasePendingAction(pending, key);
      }
    },
  };
}

test("independent actions can run concurrently", async () => {
  const runner = createActionRunner();
  const inventoryResponse = deferred();
  const profileResponse = deferred();
  const inventoryAction = runner.run("player:inventory", () => inventoryResponse.promise);
  const profileAction = runner.run("player:profile", () => profileResponse.promise);

  assert.deepEqual(runner.pending, {
    "player:inventory": true,
    "player:profile": true,
  });
  inventoryResponse.resolve();
  profileResponse.resolve();
  assert.deepEqual(await Promise.all([inventoryAction, profileAction]), [true, true]);
});

test("competing actions on the same inventory are blocked", async () => {
  const runner = createActionRunner();
  const feedResponse = deferred();
  const feedAction = runner.run("player:inventory", () => feedResponse.promise);
  let competingActionStarted = false;
  const competingAction = runner.run("player:inventory", async () => {
    competingActionStarted = true;
  });

  assert.equal(await competingAction, false);
  assert.equal(competingActionStarted, false);
  feedResponse.resolve();
  assert.equal(await feedAction, true);
});
