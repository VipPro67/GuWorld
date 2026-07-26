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
  rarity: string;
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
