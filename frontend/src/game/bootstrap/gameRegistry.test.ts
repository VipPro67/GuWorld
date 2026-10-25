import assert from "node:assert/strict";
import test from "node:test";
import { attachGame, detachGame, getAttachedGame, type DestroyableGame } from "./gameRegistry.ts";

function fakeGame(): DestroyableGame & { destroyed: boolean; calls: number } {
  return {
    destroyed: false,
    calls: 0,
    destroy() {
      this.destroyed = true;
      this.calls += 1;
    },
  };
}

test("attachGame destroys a stale game before attaching the new one", () => {
  const host = {};
  const first = fakeGame();
  const second = fakeGame();

  attachGame(host, first);
  attachGame(host, second);

  assert.equal(first.calls, 1);
  assert.equal(second.calls, 0);
  assert.equal(getAttachedGame(host), second);
});

test("attaching the same game twice does not destroy it", () => {
  const host = {};
  const game = fakeGame();
  attachGame(host, game);
  attachGame(host, game);
  assert.equal(game.calls, 0);
});

test("detachGame never destroys a newer game when given a stale reference", () => {
  const host = {};
  const stale = fakeGame();
  const current = fakeGame();
  attachGame(host, stale);
  attachGame(host, current);

  assert.equal(detachGame(host, stale), false);
  assert.equal(current.calls, 0);
  assert.equal(stale.calls, 1);
});

test("detachGame removes and destroys the current game once", () => {
  const host = {};
  const game = fakeGame();
  attachGame(host, game);

  assert.equal(detachGame(host), true);
  assert.equal(detachGame(host), false);
  assert.equal(game.calls, 1);
  assert.equal(getAttachedGame(host), null);
});
