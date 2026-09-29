import assert from "node:assert/strict";
import test from "node:test";

import {
  buildBuyOrderExecution,
  reconcileGuSelection,
  toggleGuSelection,
} from "./marketplaceGuSelection.ts";

test("selling Gu into a buy order includes the selected Gu instance", () => {
  assert.deepEqual(
    buildBuyOrderExecution("order-1", 1, "Gu", ["gu-instance-1"]),
    {
      orderId: "order-1",
      quantity: 1,
      guInstanceIds: ["gu-instance-1"],
    },
  );
});

test("selling Gu in bulk includes every selected instance", () => {
  assert.deepEqual(
    buildBuyOrderExecution("order-1", 4, "Gu", ["gu-1", "gu-2", "gu-3", "gu-4"]),
    {
      orderId: "order-1",
      quantity: 4,
      guInstanceIds: ["gu-1", "gu-2", "gu-3", "gu-4"],
    },
  );
});

test("selling Gu requires exactly as many selected instances as the quantity", () => {
  assert.throws(
    () => buildBuyOrderExecution("order-1", 2, "Gu", ["gu-instance-1"]),
    /Cần chọn đúng 2 Cổ trùng để bán/,
  );
});

test("non-Gu order execution does not include Gu instance IDs", () => {
  assert.deepEqual(
    buildBuyOrderExecution("order-1", 3, "Material", ["ignored"]),
    { orderId: "order-1", quantity: 3 },
  );
});

test("buying Gu from a sell order does not require seller-owned instance IDs", () => {
  assert.deepEqual(
    buildBuyOrderExecution("order-1", 3, "Gu", [], "SELL"),
    { orderId: "order-1", quantity: 3 },
  );
});

test("typing a Gu quantity preserves selected instances and auto-selects the rest", () => {
  assert.deepEqual(
    reconcileGuSelection(["gu-1", "gu-2", "gu-3", "gu-4"], ["gu-3"], 3),
    ["gu-3", "gu-1", "gu-2"],
  );
  assert.deepEqual(
    reconcileGuSelection(["gu-1", "gu-2", "gu-3", "gu-4"], ["gu-3", "gu-1", "gu-2"], 1),
    ["gu-3"],
  );
  assert.deepEqual(
    reconcileGuSelection(["gu-1", "gu-2"], [], 99),
    ["gu-1", "gu-2"],
  );
});

test("checking and unchecking a Gu updates the selected list used as quantity", () => {
  assert.deepEqual(toggleGuSelection(["gu-1"], "gu-2"), ["gu-1", "gu-2"]);
  assert.deepEqual(toggleGuSelection(["gu-1", "gu-2"], "gu-1"), ["gu-2"]);
});
