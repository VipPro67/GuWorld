
import type { EffectEnvelope } from "@/lib/effect-blocks";
export type { EffectBlock as ComposableEffectBlock, EffectOperation as AtomicOperation, EffectTarget as TargetScope, EffectCondition } from "@/lib/effect-blocks";

export interface GuDictionaryItem {
  id: string;
  name: string;
  rank: number;
  path: string[];
  gu_type: string;
  base_mana_cost: number;
  effects: EffectEnvelope;
  food_types: string[];
  hunger_rate: number;
  max_hunger: number;
  description: string;
  rarity: number;
  primary_role: string;
  combat_roles: string[];
  combat_tags: string[];
  support_tags: string[];
  target_type?: string;
}

import { INTERNAL_BACKEND_URL as BACKEND_URL } from "./config";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let cachedWikiData: any = null;
export function getStaticWikiData() {
  if (cachedWikiData) return cachedWikiData;
  try {
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    cachedWikiData = require("@/data/wiki-cache.json");
    return cachedWikiData;
  } catch {
    return null;
  }
}

export async function fetchAllGus(): Promise<GuDictionaryItem[]> {
  const staticData = getStaticWikiData();
  return staticData?.gus || [];
}

export async function fetchGuById(id: string): Promise<GuDictionaryItem | null> {
  const gus = await fetchAllGus();
  return gus.find((g) => g.id === id) || null;
}

export interface ItemSpec {
  id: string;
  name: string;
  item_type: string;
  type?: string;
  rarity?: number;
  rank?: number;
  path?: string;
  effects?: EffectEnvelope;
  tags?: string[];
  description: string;
  base_price?: number;
  image_url?: string;
}

export interface MonsterTemplate {
  id: string;
  name: string;
  type: string;
  max_hp: number;
  base_attack: number;
  base_defense: number;
  loot_table_id: string;
  description: string;
  is_capturable: boolean;
  capture_chance: number;
}

export interface LootEntry {
  item_id?: string;
  gu_id?: string;
  min_qty?: number;
  max_qty?: number;
  chance: number;
}

export interface LootTable {
  id: string;
  name: string;
  entries: LootEntry[];
}

export interface WikiRegion {
  id: string;
  name: string;
  description: string;
  min_rank: number;
  max_rank: number;
  parent_id?: string;
  zone_type: string;
  min_x?: number;
  min_y?: number;
  max_x?: number;
  max_y?: number;
  entry_x?: number;
  entry_y?: number;
  pool_weights: {
    cell_type_weights?: Record<string, number>;
    resources?: string[];
    wild_gus?: string[];
  };
}

export interface WikiMonsterSpawn {
  region_id: string;
  monster_id: string;
  chance: number;
  min_rank: number;
  max_rank: number;
  is_boss_spawn: boolean;
}

export interface WikiRegionLoot {
  region_id: string;
  loot_table_id: string;
  monster_type: string;
  min_rank: number;
  max_rank: number;
  chance: number;
}


export interface RequiredGuItem {
  rank: number; // 0 = any rank (legacy recipe)
  qty: number;
}

// required_gus values are either the legacy quantity number or the ranked form.
export type RequiredGus = Record<string, number | RequiredGuItem>;

export function reqGuQty(v: number | RequiredGuItem): number {
  if (typeof v === "number") return v;
  return v?.qty || 1;
}

export function reqGuRank(v: number | RequiredGuItem): number {
  if (typeof v === "number") return 0; // legacy: any rank
  return v?.rank || 0;
}

export interface RefiningRecipe {
  id: string;
  target_gu_id: string;
  target_rank?: number;
  variant?: number;
  required_gus: RequiredGus;
  required_materials: Record<string, number>;
  success_rate: number;
  primeval_stones_cost: number;
  immortal_essence_stones_cost: number;
  completion_rate?: number;
}

export async function fetchDictionaryUnified(): Promise<{
  gus: GuDictionaryItem[];
  items: ItemSpec[];
  monsters: MonsterTemplate[];
  lootTables: LootTable[];
  regions: WikiRegion[];
  monsterSpawns: WikiMonsterSpawn[];
  regionLoot: WikiRegionLoot[];
}> {
  const staticData = getStaticWikiData();
  if (staticData) {
    return {
      gus: staticData.gus || [],
      items: staticData.items || [],
      monsters: staticData.monsters || [],
      lootTables: staticData.lootTables || [],
      regions: staticData.regions || [],
      monsterSpawns: staticData.monsterSpawns || [],
      regionLoot: staticData.regionLoot || [],
    };
  }

  return { gus: [], items: [], monsters: [], lootTables: [], regions: [], monsterSpawns: [], regionLoot: [] };
}

export async function fetchMasterRecipes(): Promise<RefiningRecipe[]> {
  const staticData = getStaticWikiData();
  return staticData?.recipes || [];
}

export async function fetchAllItems(): Promise<ItemSpec[]> {
  const { items } = await fetchDictionaryUnified();
  return items;
}

export async function fetchItemMap(): Promise<Map<string, ItemSpec>> {
  const items = await fetchAllItems();
  const map = new Map<string, ItemSpec>();
  for (const item of items) {
    map.set(item.id, item);
  }
  return map;
}
