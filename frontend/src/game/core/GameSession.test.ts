import assert from "node:assert/strict";
import test from "node:test";
import { GameSession } from "./GameSession.ts";

test("GameSession follows CONNECTING -> JOINED -> LEAVING -> IDLE", () => {
  const session = new GameSession();
  assert.equal(session.getStatus(), "IDLE");

  assert.equal(session.transition("CONNECTING"), true);
  assert.equal(session.transition("JOINED"), true);
  assert.equal(session.isJoined(), true);

  assert.equal(session.transition("LEAVING"), true);
  assert.equal(session.reset(), true);
  assert.equal(session.getStatus(), "IDLE");
});

test("GameSession rejects illegal transitions", () => {
  const session = new GameSession();
  assert.equal(session.transition("JOINED"), false);
  assert.equal(session.transition("LEAVING"), false);
  assert.equal(session.getStatus(), "IDLE");
});

test("GameSession resets out of ERROR and notifies subscribers", () => {
  const session = new GameSession();
  const seen: string[] = [];
  const unsubscribe = session.subscribe((status) => seen.push(status));

  session.transition("CONNECTING");
  session.transition("ERROR");
  assert.equal(session.transition("JOINED"), false);
  assert.equal(session.reset(), true);

  unsubscribe();
  session.transition("CONNECTING");
  assert.deepEqual(seen, ["CONNECTING", "ERROR", "IDLE"]);
});
