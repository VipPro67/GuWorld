import test from "node:test";
import assert from "node:assert/strict";
import { indexGus, guAt } from "./gu-index.ts";
import type { GuDictionaryItem } from "./gu-dict.ts";

function gu(id: string, rank: number, name: string): GuDictionaryItem {
  return {
    id,
    rank,
    name,
    path: [],
    gu_type: "Active",
    base_mana_cost: 0,
    effects: { blocks: [] },
    food_types: [],
    hunger_rate: 1,
    max_hunger: 100,
    description: "",
    rarity: 0,
    primary_role: "",
    combat_roles: [],
    combat_tags: [],
    support_tags: [],
  };
}

test("guAt resolves names of duplicate Gu ids by rank", () => {
  const index = indexGus([
    gu("GU_GOLDEN_SUPPORT_I", 6, "Kim Hỗ Trợ Tiên Cổ"),
    gu("GU_GOLDEN_SUPPORT_I", 8, "Kim Hỗ Trợ Tiên Cổ"),
    gu("GU_GOLDEN_SUPPORT_I", 9, "Cửu Chuyển Kim Hỗ Trợ Tiên Cổ"),
  ]);

  assert.equal(guAt(index, "GU_GOLDEN_SUPPORT_I", 6)?.name, "Kim Hỗ Trợ Tiên Cổ");
  assert.equal(guAt(index, "GU_GOLDEN_SUPPORT_I", 8)?.name, "Kim Hỗ Trợ Tiên Cổ");
  assert.equal(guAt(index, "GU_GOLDEN_SUPPORT_I", 9)?.name, "Cửu Chuyển Kim Hỗ Trợ Tiên Cổ");
});

test("guAt falls back to the highest rank when rank is unknown", () => {
  const index = indexGus([gu("GU_X", 6, "Kim Hỗ Trợ"), gu("GU_X", 9, "Cửu Chuyển Kim Hỗ Trợ")]);

  assert.equal(guAt(index, "GU_X")?.name, "Cửu Chuyển Kim Hỗ Trợ");
  assert.equal(guAt(index, "GU_X", 7)?.name, "Cửu Chuyển Kim Hỗ Trợ");
  assert.equal(guAt(index, "GU_MISSING", 9), undefined);
});
