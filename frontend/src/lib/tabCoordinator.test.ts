import assert from "node:assert/strict";
import test from "node:test";
import { defaultTabCoordinator as authTabCoordinator } from "./tabCoordinator.ts";
import { defaultTabCoordinator as realtimeTabCoordinator } from "./realtime/RealtimeClient.ts";

test("auth and realtime share one gameplay tab coordinator", () => {
  assert.strictEqual(realtimeTabCoordinator, authTabCoordinator);
});
