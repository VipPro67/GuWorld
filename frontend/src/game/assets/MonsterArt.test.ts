import assert from "node:assert/strict";
import test from "node:test";
import { getMonsterArt } from "./MonsterArt.ts";

test("monster atlas lookup keeps row and column for full and partial atlases", () => {
  assert.deepEqual(getMonsterArt("boss_wolf_king"), {
    src: "/assets/world/monsters/monster-atlas-batch-01.webp",
    column: 0, row: 0, columns: 4, rows: 4,
  });
  assert.deepEqual(getMonsterArt("monster_sound_r3_normal"), {
    src: "/assets/world/monsters/monster-atlas-batch-32.webp",
    column: 1, row: 2, columns: 4, rows: 3,
  });
  assert.equal(getMonsterArt("missing-template"), null);
});
