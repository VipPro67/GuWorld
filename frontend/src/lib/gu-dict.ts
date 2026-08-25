/* eslint-disable @typescript-eslint/no-explicit-any */
export interface ActiveEffect {
  damage_flat?: number;
  damage_mult?: number;
  shield_flat?: number;
  range?: number;
  mana_regen?: number;
  wood_transformation?: boolean;
  blood_drain?: boolean;
  replicate?: boolean;
  apply_status?: string;
  status_tick_damage?: number;
  status_duration_ticks?: number;
  lifesteal_rate?: number;
  heal_flat?: number;
  refine_mana_tier?: number;
  cleansing_power?: number;
  summon_phantom?: boolean;
  generate_gu?: boolean;
  curve_attack?: boolean;
  summon_growth_phantoms?: boolean;
  summon_count_per_tick?: number;
  phantom_damage_scale?: number;
  [key: string]: any;
}

export interface PassiveEffect {
  stat_base_attack?: number;
  stat_base_defense?: number;
  soul_refinement?: number;
  storage_capacity?: number;
  movement?: number;
  investigation?: number;
  hearing?: number;
  evasion?: number;
  refine_mana_tier?: number;
  cleansing_power?: number;
  damage_multiplier_on_hit?: number;
  path_conflict_reduction?: number;
  [key: string]: any;
}

export interface SpecialMechanic {
  event_trigger: string;
  action_code: string;
  params: Record<string, any>;
  chance: number;
}

export interface GrowthEffect {
  max_attack?: number;
  max_defense?: number;
  max_hp?: number;
  max_speed?: number;
  hours_to_max?: number;
}

export interface GuEffects {
  active?: ActiveEffect | null;
  passive?: PassiveEffect | null;
  growth?: GrowthEffect | null;
  other?: SpecialMechanic[] | null;
  [key: string]: any;
}

export interface GuDictionaryItem {
  id: string;
  name: string;
  rank: number;
  path: string[];
  gu_type: string;
  base_mana_cost: number;
  effects: GuEffects;
  food_types: string[];
  hunger_rate: number;
  max_hunger: number;
  description: string;
  rarity: number;
  is_immortal: boolean;
  primary_role: string;
  combat_roles: string[];
  combat_tags: string[];
  support_tags: string[];
  target_type?: string;
}

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

export async function fetchAllGus(): Promise<GuDictionaryItem[]> {
  try {
    const res = await fetch(`${BACKEND_URL}/api/dictionary`, {
      cache: "force-cache",
      next: { revalidate: 3600 },
    });
    if (!res.ok) return [];
    const data = await res.json();
    const gu: Record<string, GuDictionaryItem> = data.gu || {};
    return Object.values(gu);
  } catch {
    return [];
  }
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
  effects?: Record<string, number>;
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
}> {
  try {
    const res = await fetch(`${BACKEND_URL}/api/dictionary`, {
      cache: "force-cache",
      next: { revalidate: 3600 },
    });
    if (!res.ok) return { gus: [], items: [], monsters: [], lootTables: [], regions: [], monsterSpawns: [] };
    const data = await res.json();
    const gu: Record<string, GuDictionaryItem> = data.gu || {};
    const itemsRecord: Record<string, ItemSpec> = data.items || {};
    const monstersRecord: Record<string, MonsterTemplate> = data.monsters || {};
    
    const gus = Object.values(gu);
    const items = Object.values(itemsRecord).map((item) => ({
      ...item,
      item_type: item.item_type || item.type || "Material",
    }));
    const monsters = Object.values(monstersRecord);
    const lootTables: LootTable[] = data.loot_tables || [];
    const regions: WikiRegion[] = data.regions || [];
    const monsterSpawns: WikiMonsterSpawn[] = data.monster_spawns || [];
    
    return { gus, items, monsters, lootTables, regions, monsterSpawns };
  } catch {
    return { gus: [], items: [], monsters: [], lootTables: [], regions: [], monsterSpawns: [] };
  }
}

export async function fetchMasterRecipes(): Promise<RefiningRecipe[]> {
  try {
    const res = await fetch(`${BACKEND_URL}/api/recipes?all=true`, {
      cache: "force-cache",
      next: { revalidate: 3600 },
    });
    if (!res.ok) return [];
    const data = await res.json();
    return data || [];
  } catch {
    return [];
  }
}
