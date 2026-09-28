import assert from "node:assert/strict";
import test from "node:test";
import { formatEffectTarget, renderableEffectBlocks } from "./effect-presentation.ts";

test("target formatter includes selector and bypass-frontline metadata", () => {
  assert.equal(formatEffectTarget({ selector: "ALL_ENEMIES" }), "Toàn bộ địch");
  assert.equal(formatEffectTarget({ selector: "SELF", bypass_frontline: true }), "Bản thân (Xuyên hàng)");
  assert.equal(formatEffectTarget({ selector: "SELF" }), null);
});

test("renderer presentation accepts v2 blocks and reports unsupported schemas", () => {
  const block = { id: "A", trigger: "ON_ACTION", target: { selector: "TARGET" }, operations: [{ op: "DAMAGE", value: 1 }] };
  assert.deepEqual(renderableEffectBlocks({ schema_version: 2, blocks: [block] }).blocks, [block]);
  assert.deepEqual(renderableEffectBlocks({ blocks: [block] }).blocks, [block]);
  assert.deepEqual(renderableEffectBlocks([block]).blocks, [block]);
  assert.deepEqual(renderableEffectBlocks({ schema_version: 3, blocks: [] } as never), { blocks: [], unsupportedSchema: 3 });
});
