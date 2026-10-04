import test from "node:test";
import assert from "node:assert/strict";
import {
  extractGuSearchableText,
  matchesGuEffect,
  extractGuEffectBadges,
  EFFECT_CATEGORIES,
} from "./gu-effect-search.ts";
import type { GuDictionaryItem, ItemSpec } from "./gu-dict.ts";

const sampleGuStun: GuDictionaryItem = {
  id: "GU_TEST_STUN",
  name: "Định Thần Cổ",
  rank: 3,
  path: ["Wisdom Path"],
  gu_type: "Active",
  base_mana_cost: 15,
  effects: {
    blocks: [
      {
        id: "SKILL_1",
        trigger: "ON_ACTION",
        target: { selector: "TARGET" },
        operations: [
          {
            op: "STATUS",
            status_id: "STUN",
            turns: 2,
          },
        ],
      },
    ],
  },
  food_types: ["item_beast_blood"],
  hunger_rate: 1,
  max_hunger: 100,
  description: "Cổ trùng làm choáng đối thủ trong 2 lượt.",
  rarity: 3,
  is_immortal: false,
  primary_role: "CONTROL",
  combat_roles: ["control"],
  combat_tags: ["stun"],
  support_tags: [],
};

const sampleGuLifesteal: GuDictionaryItem = {
  id: "GU_TEST_DRAIN",
  name: "Huyết Dực Cổ",
  rank: 4,
  path: ["Blood Path"],
  gu_type: "Active",
  base_mana_cost: 20,
  effects: {
    blocks: [
      {
        id: "SKILL_DRAIN",
        trigger: "ON_HIT",
        target: { selector: "SELF" },
        operations: [
          {
            op: "HEAL",
            sub_type: "LIFESTEAL",
            value: 0.25,
          },
        ],
      },
    ],
  },
  food_types: ["item_spirit_water"],
  hunger_rate: 2,
  max_hunger: 200,
  description: "Hút 25% sinh lực khi đánh trúng.",
  rarity: 4,
  is_immortal: false,
  primary_role: "ATTACK",
  combat_roles: ["damage"],
  combat_tags: ["lifesteal", "blood"],
  support_tags: [],
};

const mockItemMap = new Map<string, ItemSpec>([
  [
    "item_beast_blood",
    {
      id: "item_beast_blood",
      name: "Huyết Thú Tinh Hoa",
      item_type: "Food",
      rank: 2,
      description: "Thức ăn bổ huyết.",
    },
  ],
  [
    "item_spirit_water",
    {
      id: "item_spirit_water",
      name: "Linh Tuyền Thủy",
      item_type: "Food",
      rank: 3,
      description: "Nước suối linh khí.",
    },
  ],
]);

test("extractGuSearchableText includes effect status names, path translation, and food item names", () => {
  const textStun = extractGuSearchableText(sampleGuStun, mockItemMap);
  // Vietnamese status name
  assert.ok(textStun.includes("choang") || textStun.includes("choáng"));
  // Path translation (Wisdom Path -> Trí Đạo)
  assert.ok(textStun.includes("tri") || textStun.includes("trí"));
  // Food item name
  assert.ok(textStun.includes("huyet thu") || textStun.includes("huyết thú"));

  const textDrain = extractGuSearchableText(sampleGuLifesteal, mockItemMap);
  assert.ok(textDrain.includes("lifesteal") || textDrain.includes("hut mau"));
  assert.ok(textDrain.includes("linh tuyen") || textDrain.includes("linh tuyền"));
});

test("matchesGuEffect correctly categorizes stun and lifesteal effects", () => {
  assert.equal(matchesGuEffect(sampleGuStun, "stun"), true);
  assert.equal(matchesGuEffect(sampleGuStun, "lifesteal"), false);
  assert.equal(matchesGuEffect(sampleGuStun, "all"), true);

  assert.equal(matchesGuEffect(sampleGuLifesteal, "lifesteal"), true);
  assert.equal(matchesGuEffect(sampleGuLifesteal, "stun"), false);
  assert.equal(matchesGuEffect(sampleGuLifesteal, "all"), true);
});

test("extractGuEffectBadges returns badge summaries with icons and labels", () => {
  const badgesStun = extractGuEffectBadges(sampleGuStun, 3);
  assert.equal(badgesStun.length, 1);
  assert.equal(badgesStun[0].icon, "⚡");
  assert.ok(badgesStun[0].text.includes("Choáng"));

  const badgesDrain = extractGuEffectBadges(sampleGuLifesteal, 3);
  assert.equal(badgesDrain.length, 1);
  assert.equal(badgesDrain[0].icon, "🩸");
  assert.ok(badgesDrain[0].text.includes("Hút máu 25%"));
});

test("EFFECT_CATEGORIES contains major combat effects", () => {
  const categoryIds = EFFECT_CATEGORIES.map((c) => c.id);
  assert.ok(categoryIds.includes("all"));
  assert.ok(categoryIds.includes("stun"));
  assert.ok(categoryIds.includes("lifesteal"));
  assert.ok(categoryIds.includes("poison"));
  assert.ok(categoryIds.includes("burn"));
  assert.ok(categoryIds.includes("shield"));
  assert.ok(categoryIds.includes("heal"));
  assert.ok(categoryIds.includes("mana"));
});
