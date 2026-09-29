import assert from "node:assert/strict";
import test from "node:test";

import { buildBuyOrderExecution } from "./marketplaceGuSelection.ts";

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
