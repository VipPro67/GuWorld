export interface QuestReward {
  type: string;
  id?: string;
  amount: number;
  label: string;
}

export interface QuestDef {
  id: string;
  title: string;
  desc: string;
  req_desc: string;
  rewards: QuestReward[];
}

export function formatRewards(rewards: QuestReward[]): string {
  if (!rewards || rewards.length === 0) return "";
  const labels = rewards.map((r) => r.label).filter(Boolean);
  return labels.length > 0 ? labels.join(", ") : "";
}

export function checkQuest(
  questId: string,
  player: any,
  guDictionary: any
): boolean {
  const rank = player.rank;
  const guInventory = player.gu_inventory || [];
  const clearedCells = player.cleared_cells || {};
  const materialsInventory = player.materials_inventory || {};

  switch (questId) {
    case "quest_1_awaken":
      return rank >= 1;
    case "quest_2_vital_gu":
      return guInventory.some((g: any) => g.is_vital || g.equipped);
    case "quest_3_rank_2":
      return rank >= 2;
    case "quest_4_refine_gu":
      return guInventory.some((g: any) => {
        const spec = guDictionary?.[g.gu_id];
        return spec && spec.rank >= 2;
      });
    case "quest_5_fame":
      return Object.keys(clearedCells).length >= 5;
    case "quest_6_rank_5":
      return rank >= 5;
    case "quest_7_stone_hoarder":
      return player.primeval_stones >= 5000;
    case "quest_8_immortal_capital":
      return player.immortal_stones >= 100;
    case "quest_9_liquor_worm":
      return guInventory.some((g: any) => g.gu_id === "GU_LIQUOR_WORM");
    case "quest_10_storage_tusita":
      return guInventory.some((g: any) => g.gu_id === "GU_TUSITA_FLOWER");
    case "quest_11_gu_collector":
      return guInventory.length >= 10;
    case "quest_12_immortal_gu_owner":
      return guInventory.some((g: any) => {
        const spec = guDictionary?.[g.gu_id];
        return spec && spec.rank >= 6;
      });
    case "quest_13_gather_lan_flower":
      return (materialsInventory["nguyet_lan_hoa"] || 0) >= 50;
    case "quest_14_immortal_material":
      return (materialsInventory["tu_kim_than_bi"] || 0) >= 1;
    default:
      return false;
  }
}
