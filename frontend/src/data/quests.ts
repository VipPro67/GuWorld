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

  player: ApiResponseData,

  guDictionary: ApiResponseData
): boolean {
  if (!player) return false;
  const rank = (player.rank as number) || 0;
  const guInventory = (player.gu_inventory as Array<{ gu_id?: string; is_vital?: boolean; equipped?: boolean; rank?: number }>) || [];
  const clearedCells = (player.cleared_cells as Record<string, unknown>) || {};
  const materialsInventory = (player.materials_inventory as Record<string, number>) || {};
  const visitedSubZones = (player.visited_subzones as string[]) || [];

  switch (questId) {
    case "quest_1_awaken":
      return rank >= 1;
    case "quest_2_vital_gu":
      return guInventory.some((g) => g.is_vital || g.equipped);
    case "quest_3_rank_2":
      return rank >= 2;
    case "quest_4_refine_gu":
      return guInventory.some((g) => {
        const spec = guDictionary?.[g.gu_id || ""];
        return (spec && (spec.rank || 0) >= 2) || ((g.rank || 0) >= 2);
      });
    case "quest_5_fame":
      return Object.keys(clearedCells).length >= 5;
    case "quest_6_rank_3":
      return rank >= 3;
    case "quest_7_stone_hoarder":
      return ((player.primeval_stones as number) || 0) >= 5000;
    case "quest_8_rank_4":
      return rank >= 4;
    case "quest_9_liquor_worm":
      return guInventory.some((g) => g.gu_id === "GU_LIQUOR_WORM");
    case "quest_10_storage_tusita":
      return guInventory.some((g) => g.gu_id === "GU_TUSITA_FLOWER");
    case "quest_11_gu_collector":
      return guInventory.length >= 10;
    case "quest_12_rank_5":
      return rank >= 5;
    case "quest_13_gather_lan_flower":
      return (materialsInventory["moon_orchid_petal"] || 0) >= 50;
    case "quest_14_immortal_capital":
      return ((player.immortal_stones as number) || 0) >= 100;
    case "quest_15_immortal_gu_owner":
      return guInventory.some((g) => {
        const spec = guDictionary?.[g.gu_id || ""];
        return (spec && (spec.rank || 0) >= 6) || ((g.rank || 0) >= 6);
      });
    case "quest_16_immortal_material":
      return (materialsInventory["immortal_blood"] || 0) >= 1;
    case "quest_17_rank_6":
      return rank >= 6;
    case "quest_18_human_qi":
      return ((player.human_qi as number) || 0) >= 500;
    case "quest_19_soul_growth":
      return ((player.soul_hp as number) || 0) >= 100;
    case "quest_20_exploration":
      return visitedSubZones.length >= 20;
    default:
      return false;
  }
}


export function isClaimableAchievement(achId: string, player: ApiResponseData, aperture?: ApiResponseData | null): boolean {
  if (!player) return false;
  const rank = (player.rank as number) || 0;
  const guInventory = (player.gu_inventory as Array<{ gu_id?: string }>) || [];
  const clearedCells = (player.cleared_cells as Record<string, unknown>) || {};
  const visitedSubZones = (player.visited_subzones as string[]) || [];

  switch (achId) {
    case "first_combat":
      return Object.keys(clearedCells).length > 0 || ((player.explore_count as number) || 0) > 0;
    case "rank_2":
      return rank >= 2;
    case "rank_3":
      return rank >= 3;
    case "rank_4":
      return rank >= 4;
    case "rank_5":
      return rank >= 5;
    case "rank_6":
      return rank >= 6;
    case "rank_7":
      return rank >= 7;
    case "rank_8":
      return rank >= 8;
    case "rank_9":
      return rank >= 9;
    case "tribulation_1":
      return rank >= 6 && tribulationCount(aperture) >= 1;
    case "tribulation_5":
      return rank >= 6 && tribulationCount(aperture) >= 5;
    case "tribulation_10":
      return rank >= 6 && tribulationCount(aperture) >= 10;
    case "gu_10":
      return new Set(guInventory.map((g) => g.gu_id)).size >= 10;
    case "gu_25":
      return new Set(guInventory.map((g) => g.gu_id)).size >= 25;
    case "gu_50":
      return new Set(guInventory.map((g) => g.gu_id)).size >= 50;
    case "explorer_100":
      return visitedSubZones.length >= 100;
    case "explorer_500":
      return visitedSubZones.length >= 500;
    case "explorer_1000":
      return visitedSubZones.length >= 600;
    case "millionaire":
      return ((player.primeval_stones as number) || 0) >= 1000000;
    case "human_qi_1000":
      return ((player.human_qi as number) || 0) >= 1000;
    case "human_qi_10000":
      return ((player.human_qi as number) || 0) >= 10000;
    case "human_qi_100000":
      return ((player.human_qi as number) || 0) >= 100000;
    case "soul_awaken":
      return ((player.soul_hp as number) || 0) >= 100;
    case "wall_crack_50":
      return ((player.wall_progress as number) || 0) >= 0.5;
    case "wall_crack_100":
      return ((player.wall_progress as number) || 0) >= 1.0;
    default:
      return false;
  }
}

function tribulationCount(aperture?: ApiResponseData | null): number {
  if (!aperture) return 0;
  return ([
    "earth_calamity_count",
    "heavenly_tribulation_count",
    "grand_tribulation_count",
    "myriad_tribulation_count",
    "chaos_calamity_count",
  ] as const).reduce((total, key) => total + (Number(aperture[key]) || 0), 0);
}
