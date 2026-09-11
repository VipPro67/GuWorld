import { create } from "zustand";
import FingerprintJS from "@fingerprintjs/fingerprintjs";
import type { RequiredGus } from "@/lib/gu-dict";
import type { EffectDictionaryEntry } from "@/lib/effects";
import { useUIStore } from "@/store/useUIStore";
import { wsManager } from "@/lib/websocket";
let fetchAperturePromise: Promise<void> | null = null;
let fetchApertureNodesPromise: Promise<void> | null = null;
let fetchVariantHumansPromise: Promise<void> | null = null;
let fetchGuInventoryPromise: Promise<void> | null = null;
let fetchMySectStatusPromise: Promise<void> | null = null;

export interface GuInstance {
  instance_id: string;
  gu_id: string;
  rank: number;
  hunger: number;
  status: string; // "ACTIVE", "SLEEPING", "DEAD"
  equipped: boolean;
  is_vital: boolean;
  integration_time?: number;
  growth_hours?: number;
  equipped_at?: string;
}

export interface RefineRateBreakdown {
  base_rate: number;
  completion_penalty: number;
  combo_accuracy: boolean;
  material_quality: number;
  refinement_dao_marks: number;
  refinement_mastery: number;
  target_path_mastery: number;
  luck_bonus: number;
  support_bonus: number;
  status_penalty?: number;
  env_bonus: number;
  final_rate: number;
  refinement_realm: number;
}


export interface Combatant {
  id: string;
  name: string;
  kind: string; // "Player", "Normal", "Boss", etc.
  rank: number;
  sub_rank: number;
  hp: number;
  max_hp: number;
  current_mana: number;
  max_mana: number;
  base_attack: number;
  base_defense: number;
  gu_inventory?: GuInstance[];
  dao_marks?: Record<string, number>;
  path_masteries?: Record<string, number>;
  ai_profile?: string;
  loot_table_id?: string;
  grid_x?: number;
  grid_y?: number;
  speed?: number;
  virtual_shield?: number;
  avatar_url?: string;
}

export interface CombatState {
  enemy_name: string;
  enemy_hp: number;
  enemy_max_hp: number;
  enemy_damage: number;
  enemy_defense: number;
  charge_points: number;
  target_player_id?: string;
  enemy_template_id?: string;
  allies?: Combatant[];
  enemies?: Combatant[];
  enemy_type?: string;
  enemy_rank?: number;
  enemy_sub_rank?: number;
  combat_mode?: string;
  mine_slot_id?: number;
  tribulation_event_id?: string;
  aperture_hp?: number;
  aperture_max_hp?: number;
}

export interface PlayerPet {
  id: string;
  player_id: string;
  monster_template_id: string;
  nickname: string;
  rank: number;
  sub_rank: number;
  hp: number;
  max_hp: number;
  base_attack: number;
  base_defense: number;
  speed: number;
  count: number;
  hunger: number;
  max_hunger: number;
  pet_hunger_per_hour: number;
  state: string;
  formation_slot: number;
  captured_at: string;
  updated_at: string;
}

export interface CombatHistoryEntry {
  id: string;
  player_id: string;
  winner: string;
  logs: any[];
  summary: {
    total_damage_dealt: number;
    total_damage_taken: number;
    shields_used: number;
    final_player_hp: number;
    items_used: number;
    stones_used: number;
    loot_count: number;
  };
  combatants: any[];
  enemy_name: string;
  enemy_type: string;
  region: string;
  created_at: string;
}

// Death Arena Types
export interface DeathArenaTournament {
  id: string;
  status: string;
  scheduled_at: string;
  registration_started_at: string;
  registration_closed_at: string;
  started_at?: string;
  finished_at?: string;
  total_players: number;
  total_teams: number;
  current_round: number;
  total_rounds: number;
  champion_team_id?: string;
  created_at: string;
}

export interface DeathArenaTeam {
  id: string;
  tournament_id: string;
  team_name: string;
  seed_number: number;
  eliminated: boolean;
  eliminated_at_round?: number;
  final_placement?: number;
  member_count: number;
  avg_hp: number;
  created_at: string;
}

export interface DeathArenaTeamMember {
  id: string;
  team_id: string;
  player_id: string;
  username: string;
  avatar_url: string;
  rank: number;
  current_hp: number;
  max_hp: number;
  current_mana: number;
  max_mana: number;
  eliminated: boolean;
  eliminated_at_round?: number;
  joined_at: string;
}

export interface DeathArenaMatch {
  id: string;
  tournament_id: string;
  round_number: number;
  bracket_position: number;
  team_a_id: string;
  team_a_name: string;
  team_b_id?: string;
  team_b_name?: string;
  winner_team_id?: string;
  status: string;
  reward_stones_per_member: number;
  created_at: string;
  finished_at?: string;
}

export interface DeathArenaReward {
  id: string;
  tournament_id: string;
  player_id: string;
  username: string;
  placement: number;
  team_id: string;
  team_name: string;
  title: string;
  reward_stones: number;
  reward_immortal_stones: number;
  reward_gu_id?: string;
  reward_gu_rank?: number;
  reward_items: any[];
  claimed: boolean;
  created_at: string;
}

export interface Player {
  id: string;
  username: string;
  rank: number;
  sub_rank: number;
  aptitude_type: string;
  aptitude_percent: number;
  current_mana: number;
  max_mana: number;
  primeval_stones: number;
  immortal_stones: number;
  gu_inventory: GuInstance[];
  materials_inventory: Record<string, number>;
  wall_progress?: number;
  human_qi?: number;
  next_tribulation_time?: string | null;
  hp: number;
  base_hp: number;
  body_mods: string[];
  current_region: string;
  dao_marks: Record<string, number>;
  path_masteries: Record<string, number>;
  x: number;
  y: number;
  equipped_gus: string[];
  cleared_cells: Record<string, string>;
  visited_subzones?: string[];
  active_combat: CombatState | null;
  last_mana_update: string;
  created_at: string;
  lifespan: number;
  time_speed_multiplier: number;
  party_id?: string | null;
  arena_rank?: number | null;
  completed_quests: string[];
  aperture_stress?: number;
  last_cultivate_time?: string;
  soul_hp: number;
  max_soul: number;
  soul_tier: string;
  needs_soul_refine: boolean;
  unlocked_recipes?: { recipe_id: string; completion_rate: number }[];
  email?: string;
  google_id?: string;
  bio?: string;
  fake_name?: string;
  fake_rank?: number;
  fake_sub_rank?: number;
  fake_vital_gu?: string;
  is_disguised?: boolean;
  is_admin?: boolean;
  is_sparse_gu_inventory?: boolean;
  avatar_url?: string;
  combat_settings?: AutoCombatConfig;
  needs_onboarding?: boolean;
  current_thought: number;
  max_thought: number;
  thought_regen_rate: number;
  mana_regen_rate: number;
  hp_regen_rate: number;
  soul_regen_rate?: number;
  growth_data?: {
    boosts: { attack: number; defense: number; hp: number; speed: number };
    fully_grown: { gu_id: string; rank: number; attack: number; defense: number; hp: number; speed: number }[];
  };
  race: string;
  gender: string;
  constitution: string;
  statuses: PlayerStatusInstance[];
}

export interface PlayerStatusInstance {
  id: string;
  expires_at?: string | null;
  source?: string;
  value?: number;
}


export interface GuTactic {
  gu_id: string;
  priority: number;
  use_condition: string;
  condition_value: number;
}

export interface KillerMoveTactic {
  killer_move_id: string;
  priority: number;
  use_condition: string;
  condition_value: number;
}

export interface AutoCombatConfig {
  auto_heal_hp: boolean;
  heal_threshold: number;
  auto_restore_mana: boolean;
  mana_threshold: number;
  category_priorities?: string[];
  gu_priorities: GuTactic[];
  km_priorities: KillerMoveTactic[];
  skip_battle: boolean;
}

export interface KillerMoveSlot {
  id: string;
  killer_move_id: string;
  player_gu_id: string;
  slot_type: "CORE" | "SUPPORT" | string;
  slot_index: number;
  created_at: string;
}

export interface KillerMove {
  id: string;
  player_id: string;
  name: string;
  path: string;
  move_type: string;
  rank_required: number;
  core_player_gu_id: string;
  total_mana_cost: number;
  stability: number;
  damage_value: number;
  defense_value: number;
  cooldown: number;
  cast_time: number;
  effects?: Record<string, number> | string | null;
  is_mastered: boolean;
  is_stable: boolean;
  is_instant_cast: boolean;
  created_at: string;
  updated_at: string;
  slots?: KillerMoveSlot[];
}

export interface ManaSnapshot {
  player_id: string;
  current_mana: number;
  max_mana: number;
  last_mana_update: string;
  time_speed_multiplier: number;
  time_rate?: number;
  rank: number;
  sub_rank: number;
  aptitude_type?: string;
  lifespan?: number;
  essence_production_per_year?: number;
  captured_at: number;
  hp: number;
  base_hp: number;
  hp_regen_rate: number;
  mana_regen_tick_per_hour?: number;
  mana_regen_mult?: number;
}

// manaRegenScale is the per-rank "cost" used to scale MANA_REGEN_TICK so a Gu
// gives its full value only when its rank matches the player's, and less when
// the player outranks it. Index 0 = rank 1. Mirrors backend models.ManaRegenScale.
const manaRegenScale = [1, 2, 5, 20, 100, 1000, 100000, 10000000, 100000000];

// manaRegenTickFactor returns the multiplier applied to a MANA_REGEN_TICK
// value: 1.0 when player and Gu ranks match, otherwise cost[guRank]/cost[playerRank].
const manaRegenTickFactor = (playerRank: number, guRank: number): number => {
  if (playerRank === guRank) return 1;
  if (playerRank < 1 || playerRank > manaRegenScale.length || guRank < 1 || guRank > manaRegenScale.length) return 1;
  return manaRegenScale[guRank - 1] / manaRegenScale[playerRank - 1];
};

// manaRegenTickPerHour sums the per-hour mana contribution of every equipped
// ACTIVE Gu carrying an ON_TICK MANA_REGEN_TICK effect.
const manaRegenTickPerHour = (player: Player, guDictionary: Record<string, GuSpec>): number => {
  if (!player.gu_inventory) return 0;
  let total = 0;
  for (const gu of player.gu_inventory) {
    if (!gu.equipped || gu.status !== "ACTIVE") continue;
    const spec = guDictionary[gu.gu_id];
    if (!spec || !spec.effects?.other) continue;
    for (const mech of spec.effects.other) {
      if (mech.event_trigger !== "ON_TICK" || mech.action_code !== "MANA_REGEN_TICK") continue;
      let val = mech.params?.value;
      if (typeof val !== "number" || val <= 0) val = 0.5;
      total += val * manaRegenTickFactor(player.rank, gu.rank);
    }
  }
  return total;
};

const getManaRegenRatePerHour = (snapshot: ManaSnapshot): number => {
  let rate: number;
  if (snapshot.rank >= 6) {
    const subRank = Math.min(4, Math.max(1, snapshot.sub_rank || 1));
    const subRankFactor = 1.0 + (subRank - 1) * 0.15;
    const essenceProduction = snapshot.essence_production_per_year ?? 10;
    rate = (essenceProduction / 120.0) * subRankFactor;
  } else {
    switch (snapshot.aptitude_type) {
      case "Giáp":
        rate = 8.0;
        break;
      case "Ất":
        rate = 6.0;
        break;
      case "Bính":
        rate = 4.0;
        break;
      case "Đinh":
        rate = 2.0;
        break;
      default:
        rate = 2.0;
    }
  }

  rate += snapshot.mana_regen_tick_per_hour || 0;
  rate *= snapshot.mana_regen_mult || 1;
  return rate;
};

export const estimateManaFromSnapshot = (snapshot: ManaSnapshot | null, nowMs = Date.now()): number | null => {
  if (!snapshot) return null;
  const isImmortal = snapshot.rank >= 6;
  if (snapshot.max_mana <= 0 && !isImmortal) return Math.max(0, snapshot.current_mana);

  const lastUpdateMs = new Date(snapshot.last_mana_update).getTime();
  if (!Number.isFinite(lastUpdateMs)) {
    if (isImmortal) {
      return Math.max(0, snapshot.current_mana);
    }
    return Math.min(snapshot.max_mana, Math.max(0, snapshot.current_mana));
  }

  const elapsedSeconds = Math.max(0, (nowMs - lastUpdateMs) / 1000);
  const timeSpeedMultiplier = snapshot.time_speed_multiplier > 0 ? snapshot.time_speed_multiplier : 1;
  const timeRate = snapshot.time_rate && snapshot.time_rate > 0 ? snapshot.time_rate : 10.0;
  const multiplier = isImmortal ? timeSpeedMultiplier * timeRate : timeSpeedMultiplier;
  const regenRatePerHour = getManaRegenRatePerHour(snapshot);
  const lifespanModifier = snapshot.lifespan !== undefined && snapshot.lifespan <= 0 ? 0.5 : 1;
  const regeneratedMana = ((elapsedSeconds * multiplier) / 3600) * regenRatePerHour * lifespanModifier;

  const newMana = snapshot.current_mana + regeneratedMana;
  if (isImmortal) {
    return Math.max(0, newMana);
  }
  return Math.min(snapshot.max_mana, Math.max(0, newMana));
};

export const estimateHPFromSnapshot = (snapshot: ManaSnapshot | null, nowMs = Date.now()): number | null => {
  if (!snapshot) return null;
  if (snapshot.hp <= 0) return 0;
  if (snapshot.hp >= snapshot.base_hp) return snapshot.base_hp;

  const lastUpdateMs = new Date(snapshot.last_mana_update).getTime();
  if (!Number.isFinite(lastUpdateMs)) {
    return Math.min(snapshot.base_hp, Math.max(0, snapshot.hp));
  }

  const elapsedSeconds = Math.max(0, (nowMs - lastUpdateMs) / 1000);
  const timeSpeedMultiplier = snapshot.time_speed_multiplier > 0 ? snapshot.time_speed_multiplier : 1;
  const isImmortal = snapshot.rank >= 6;
  const timeRate = snapshot.time_rate && snapshot.time_rate > 0 ? snapshot.time_rate : 10.0;
  const multiplier = isImmortal ? timeSpeedMultiplier * timeRate : timeSpeedMultiplier;
  const regenHPPerHour = snapshot.hp_regen_rate || 0;
  const regeneratedHP = ((elapsedSeconds * multiplier) / 3600) * regenHPPerHour;

  return Math.min(snapshot.base_hp, Math.max(0, snapshot.hp + regeneratedHP));
};

const createManaSnapshotFromPlayer = (player: Player, guDictionary: Record<string, GuSpec>, constitutionDictionary?: Record<string, EffectDictionaryEntry>, timeRate?: number): ManaSnapshot => ({
  player_id: player.id,
  current_mana: player.current_mana,
  max_mana: player.max_mana,
  last_mana_update: player.last_mana_update,
  time_speed_multiplier: player.time_speed_multiplier || 1,
  time_rate: timeRate ?? 10.0,
  rank: player.rank,
  sub_rank: player.sub_rank,
  aptitude_type: player.aptitude_type,
  lifespan: player.lifespan,
  captured_at: Date.now(),
  hp: player.hp,
  base_hp: player.base_hp,
  hp_regen_rate: player.hp_regen_rate || 0,
  mana_regen_tick_per_hour: manaRegenTickPerHour(player, guDictionary),
  mana_regen_mult: 1 + (player.constitution ? (constitutionDictionary?.[player.constitution]?.effects?.mana_regen ?? 1) : 0),
});

export interface ImmortalAperture {
  id: string;
  player_id: string;
  grade: 'LOW' | 'MID' | 'HIGH' | 'EXTREME';
  time_rate: number;
  aperture_hp: number;
  max_aperture_hp: number;
  area_size: number;
  immortal_essence: number;
  essence_production_per_year: number;
  internal_years_passed: number;
  next_tribulation_year: number;
  tribulation_count: number;
  rank_start_year: number;
  earth_calamity_count: number;
  heavenly_tribulation_count: number;
  grand_tribulation_count: number;
  myriad_tribulation_count: number;
  chaos_calamity_count: number;
  state: 'STABLE' | 'TRIBULATION_PENDING' | 'TRIBULATION_ACTIVE' | 'DAMAGED' | 'DESTROYED';
  created_at: string;
  updated_at: string;
  temperature?: number;
  humidity?: number;
  qi?: number;
  terrain?: string;
}

export interface DaoMark {
  id: string;
  player_id: string;
  path: string;
  amount: number;
  created_at: string;
  updated_at: string;
}

export interface ApertureResourceNode {
  id: string;
  aperture_id: string;
  node_type: string;
  name: string;
  path: string;
  level: number;
  hp: number;
  max_hp: number;
  x: number;
  y: number;
  production_type: string;
  production_amount: number;
  production_interval_hours: number;
  last_collected_at: string | null;
  is_destroyed: boolean;
  created_at: string;
  updated_at: string;
  node_category: string;
  defense_power: number;
  attack_range: number;
  hp_bonus: number;
  damage_reduction: number;
  blocks_movement: boolean;
  required_gu_id: string;
  required_gu_rank: number;
  zone_type: string;
  food_path: string;
  item_tier: string;
  production_key: string;
}

export interface ApertureVariantHuman {
  id: string;
  aperture_id: string;
  race: 'HAIRY_MAN' | 'ROCKMAN' | 'FEATHERMAN';
  population: number;
  happiness: number;
  oppression: number;
  rebellion_rate: number;
  labor_intensity: 'LOW' | 'NORMAL' | 'HIGH' | 'EXPLOIT';
  created_at: string;
  updated_at: string;
}

export interface TribulationEvent {
  id: string;
  aperture_id: string;
  player_id: string;
  tribulation_type: 'EARTH_CALAMITY' | 'HEAVENLY_TRIBULATION' | 'GRAND_TRIBULATION' | 'MYRIAD_TRIBULATION' | 'CHAOS_CALAMITY';
  path: string;
  difficulty: number;
  status: 'ACTIVE' | 'SUCCESS' | 'FAILED';
  aperture_hp_before: number;
  aperture_hp_after: number | null;
  reward_dao_marks: number;
  started_at: string;
  ended_at: string | null;
}

export interface TribulationEnemy {
  id: string;
  event_id: string;
  name: string;
  enemy_type: string;
  path: string;
  hp: number;
  max_hp: number;
  attack: number;
  defense: number;
  target_node_id: string | null;
  x: number;
  y: number;
  is_boss: boolean;
  is_dead: boolean;
  rank?: number;
  sub_rank?: number;
  created_at: string;
  updated_at: string;
}

export interface SpecialMechanic {
  event_trigger: string;
  action_code: string;
  params: Record<string, any>;
}

export interface GuEffects {
  active?: Record<string, any>;
  support?: Record<string, any>;
  passive?: Record<string, any>;
  house?: Record<string, any>;
  growth?: {
    max_attack?: number;
    max_defense?: number;
    max_hp?: number;
    max_speed?: number;
    hours_to_max?: number;
  };
  other?: SpecialMechanic[];
}

export interface GuSpec {
  id: string;
  name: string;
  rank: number;
  path: string[];
  gu_type: string;
  base_mana_cost: number;
  effects: GuEffects;
  hunger_rate: number;
  food_types: string[];
  max_hunger: number;
  description: string;
  rarity: number;
  is_immortal: boolean;
  primary_role: string;
  combat_roles: string[];
  combat_tags: string[];
  support_tags: string[];
  base_price?: number;
  image_url?: string;
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

export interface MonsterSpec {
  id: string;
  name: string;
  type: string;
  rank: number;
  sub_rank: number;
  path: string;
  max_hp: number;
  base_attack: number;
  base_defense: number;
  max_mana: number;
  current_mana: number;
  ai_profile: string;
  loot_table_id: string;
  description: string;
  is_capturable?: boolean;
  capture_chance?: number;
  pet_hunger_per_hour?: number;
}


export interface InstanceOption {
  text: string;
  next_stage: number;
  requirements: Record<string, any>;
}

export interface InstanceStage {
  stage_index: number;
  description: string;
  options: InstanceOption[];
}

export interface InstanceTemplate {
  id: string;
  name: string;
  type: string;
  required_rank: number;
  entry_item: string;
  description: string;
  rewards: Record<string, any>;
  map_data: InstanceStage[];
  cooldown_hours?: number;
  cooldown_remaining?: number;
}

export interface InstanceListItem {
  id: string;
  name: string;
  type: string;
  required_rank: number;
  entry_item: string;
  description: string;
  rewards: Record<string, any>;
  cooldown_hours: number;
  cooldown_remaining: number;
}

export interface PlayerInstanceSession {
  id: string;
  player_id: string;
  instance_id: string;
  current_stage: number;
  status: "ACTIVE" | "COMPLETED" | "FAILED";
  temp_inventory: any[];
}

export interface SectMineDefender {
  id: string;
  defender_type: 'member' | 'beast';
  player_id: string | null;
  name: string;
  monster_id: string | null;
  monster_rank: number;
  monster_sub_rank: number;
  power: number;
  assigned_at: string;
}

export interface SectMine {
  id: number;
  name: string;
  rarity: string;
  occupier_sect_id: string | null;
  occupier_name: string | null;
  occupied_at: string | null;
  shield_until: string | null;
  last_claimed_at: string | null;
  defenders: SectMineDefender[];
  total_defense_power: number;
  output_per_hour: { funds_per_hour: number; contribution_per_hour: number } | null;
}

export interface SectVaultGuItem {
  id: string;
  rank?: number;
  cost: number;
  quantity: number;
  hunger?: number;
}

export interface SectVaultRecipe {
  recipe_id: string;
  completion_rate: number;
}

export interface Sect {
  id: string;
  name: string;
  description: string;
  region: string;
  vault_gu: SectVaultGuItem[];
  vault_materials: Record<string, number>;
  vault_recipes: SectVaultRecipe[];
  funds: number;
  level: number;
  require_approval?: boolean;
  require_promotion_approval?: boolean;
}

export interface SectJoinRequest {
  id: string;
  sect_id: string;
  player_id: string;
  status: 'PENDING' | 'APPROVED' | 'REJECTED';
  created_at: string;
  username?: string;
  avatar_url?: string;
  player_rank?: number;
}

export interface SectPromotionRequest {
  id: string;
  sect_id: string;
  player_id: string;
  current_rank: 'NGOAI_MON' | 'NOI_MON' | 'TINH_ANH' | 'CHAN_TRUYEN' | 'TRUONG_LAO' | 'HO_PHAP' | 'CHUONG_MON';
  target_rank: 'NGOAI_MON' | 'NOI_MON' | 'TINH_ANH' | 'CHAN_TRUYEN' | 'TRUONG_LAO' | 'HO_PHAP' | 'CHUONG_MON';
  status: 'PENDING' | 'APPROVED' | 'REJECTED';
  created_at: string;
  username?: string;
  avatar_url?: string;
  player_rank?: number;
}

export interface SectMember {
  id: string;
  sect_id: string;
  player_id: string;
  member_rank: 'NGOAI_MON' | 'NOI_MON' | 'TINH_ANH' | 'CHAN_TRUYEN' | 'TRUONG_LAO' | 'HO_PHAP' | 'CHUONG_MON';
  contribution_points: number;
  master_id: string | null;
  joined_at: string;
}

export interface SectMission {
  id: string;
  sect_id: string;
  title: string;
  description: string;
  mission_type: 'GATHER' | 'COMBAT' | 'REFINE' | 'DONATE';
  target_id: string;
  target_amount: number;
  reward_contribution: number;
  required_member_rank: 'NGOAI_MON' | 'NOI_MON' | 'TINH_ANH' | 'CHAN_TRUYEN' | 'TRUONG_LAO' | 'HO_PHAP' | 'CHUONG_MON';
  expires_at?: string | null;
}

export interface SectLeaderboardEntry {
  username: string;
  avatar_url?: string;
  rank: 'NGOAI_MON' | 'NOI_MON' | 'TINH_ANH' | 'CHAN_TRUYEN' | 'TRUONG_LAO' | 'HO_PHAP' | 'CHUONG_MON';
  contribution: number;
}

export interface MySectStatus {
  in_sect: boolean;
  member?: SectMember;
  sect?: Sect;
  has_active: boolean;
  active_mission?: SectMission;
  active_progress?: number;
  active_pm_id?: string;
  leaderboard?: SectLeaderboardEntry[];
  has_pending_join?: boolean;
  pending_sect_id?: string;
  pending_sect_name?: string;
  has_pending_promotion?: boolean;
  pending_promotion_target?: string;
}

export interface SectMemberInfo {
  player_id: string;
  username: string;
  avatar_url?: string;
  member_rank: 'NGOAI_MON' | 'NOI_MON' | 'TINH_ANH' | 'CHAN_TRUYEN' | 'TRUONG_LAO' | 'HO_PHAP' | 'CHUONG_MON';
  contribution_points: number;
  joined_at: string;
  player_rank: number;
  power?: number;
}

export interface Feedback {
  id: string;
  user_id: string;
  username?: string;
  title: string;
  description: string;
  category: string;
  status: string;
  total_votes: number;
  comment_count: number;
  developer_response: string | null;
  rewarded: boolean;
  created_at: string;
  user_vote?: number | null;
}

export interface FeedbackComment {
  id: string;
  feedback_id: string;
  user_id: string;
  username?: string;
  content: string;
  created_at: string;
}

interface AuthResponse {
  player: Player;
  token?: string;
}

interface ExploreResponse {
  player: Player;
  reward_stones: number;
  log: string;
}

interface FeedResponse {
  player: Player;
  message: string;
}

export interface RefiningRecipe {
  id: string;
  target_gu_id: string;
  target_rank?: number;
  required_gus: RequiredGus;
  required_materials: Record<string, number>;
  success_rate: number;
  primeval_stones_cost: number;
  immortal_essence_stones_cost: number;
  completion_rate?: number;
}

export interface ToastMessage {
  id: string;
  message: string;
  type: "success" | "error" | "info";
  duration?: number;
}

export interface PlayerNotification {
  id: string;
  player_id: string;
  type: string;
  title: string;
  content: string;
  is_read: boolean;
  created_at: string;
}

export interface Region {
  id: string;
  name: string;
  description: string;
  width: number;
  height: number;
  min_rank: number;
  max_rank: number;
  is_active: boolean;
  parent_id?: string;
  zone_type?: string; // "HUB" | "NGU_VUC" | "KHU_VUC"
  min_x?: number;
  min_y?: number;
  max_x?: number;
  max_y?: number;
  entry_x?: number;
  entry_y?: number;
}

export interface WorldConfig {
  width: number;
  height: number;
  large_zone_size: number;
  sub_zone_size: number;
  hub_region_id: string;
  hub_x: number;
  hub_y: number;
}

interface GameState {
  player: Player | null;
  manaSnapshot: ManaSnapshot | null;
  token: string | null;
  isLoading: boolean;
  actionLoading: boolean;
  error: string | null;
  logs: string[];
  guDictionary: Record<string, GuSpec>;
  guByRank: Record<string, Record<number, GuSpec>>;
  itemDictionary: Record<string, ItemSpec>;
  monsterDictionary: Record<string, MonsterSpec>;
  constitutionDictionary: Record<string, EffectDictionaryEntry>;
  raceDictionary: Record<string, EffectDictionaryEntry>;
  dictionaryLoaded: boolean;
  recipes: RefiningRecipe[];
  recipesLoaded: boolean;
  regions: Region[];
  regionsLoaded: boolean;
  world: WorldConfig | null;
  fetchRegions: () => Promise<void>;
  fetchWorld: () => Promise<WorldConfig | null>;
  killerMoves: KillerMove[];
  killerMovesLoaded: boolean;
  fetchKillerMoves: () => Promise<void>;
  deleteKillerMove: (id: string) => Promise<boolean>;
  castKillerMove: (id: string, targetId?: string, usedExtraGuThisTurn?: boolean) => Promise<{ player: Player; logs: string[]; victory?: boolean; defeat?: boolean } | null>;
  inspectedProfile: any | null;
  inspectedUsernameQuery: string | null;
  profileLoading: boolean;
  fetchPlayerProfile: (username: string) => Promise<void>;
  clearInspectedProfile: () => void;
  updateProfileSettings: (settings: { 
    username: string; 
    bio: string; 
    new_password?: string;
    fake_name?: string;
    fake_rank?: number;
    fake_sub_rank?: number;
    fake_vital_gu?: string;
    is_disguised?: boolean;
  }) => Promise<boolean>;
  uploadAvatar: (file: File) => Promise<string | null>;
  login: (email: string, password: string) => Promise<boolean>;
  sendOTP: (email: string) => Promise<boolean>;
  getBrowserFingerprint: () => Promise<string>;
  register: (
    username: string,
    password: string,
    email: string,
    startingRegion: string,
    setVitalNow: boolean,
    referralCode?: string,
    attackGuId?: string,
    defenseGuId?: string,
    vitalGuChoice?: string,
    otp?: string,
    race?: string,
    gender?: string
  ) => Promise<boolean>;
  logout: () => void;
  clearError: () => void;
  completeOnboarding: (data: {
    starting_region: string;
    attack_gu_id: string;
    defense_gu_id: string;
    vital_gu_choice: string;
  }) => Promise<boolean>;
  updateManaSnapshot: (player: Player | null) => void;
  setPlayerFromServer: (player: Player | Partial<Player> | null) => void;
  getEstimatedMana: () => number | null;
  fetchPlayerStatus: (id: string) => Promise<void>;
  fetchGuInventory: () => Promise<void>;
  guInventoryLoaded: boolean;
  subscribeEvents: () => void;
  unsubscribeEvents: () => void;
  subscribePlayerEvents: () => void;
  unsubscribePlayerEvents: () => void;
  playerEventsConnected: boolean;
  playerEventsRetries: number;
  subscribeTradeEvents: () => void;
  unsubscribeTradeEvents: () => void;
  subscribeBaoHoangMessages: (channel: string) => void;
  unsubscribeBaoHoangMessages: () => void;
  initializeSession: () => Promise<void>;
  explore: () => Promise<void>;
  feedGu: (guInstanceId: string) => Promise<void>;
  feedGuGroup: (guId: string, rank?: number) => Promise<void>;
  feedAllGu: () => Promise<void>;
  fetchDictionary: () => Promise<void>;
  fetchRecipes: () => Promise<void>;
  quests: any[];
  fetchQuests: () => Promise<void>;
  refineGu: (recipeId: string, selectedInstanceIds: string[]) => Promise<{ status: "success" | "failed"; message: string; new_gu?: GuInstance; rate_breakdown?: RefineRateBreakdown } | null>;
  deduceRecipe: (recipeId: string) => Promise<{ success: boolean; message: string; completion_rate: number } | null>;
  breakWall: (manaSpend: number) => Promise<{ success: boolean; message: string } | null>;
  useRelicGu: (relicKey: string) => Promise<{ success: boolean; message: string } | null>;
  ascension: () => Promise<{ success: boolean; message: string } | null>;
  triggerTribulation: () => Promise<{ success: boolean; message: string } | null>;
  completeQuest: (questId: string) => Promise<{ success: boolean; message: string } | null>;
  redeemGiftCode: (code: string) => Promise<{ success: boolean; message: string } | null>;
  gachaPools: any[];
  fetchGachaPools: () => Promise<void>;
  openGacha: (poolId: string, quantity?: number) => Promise<{ success: boolean; message: string; reward?: any; rewards?: any[] } | null>;
  currentBettingRound: any | null;
  myBetTeam: string | null;
  myBetAmount: number | null;
  bettingHistory: any[];
  fetchCurrentBettingRound: () => Promise<void>;
  fetchBettingHistory: () => Promise<void>;
  placeBeastBet: (team: string, amount: number) => Promise<{ success: boolean; message: string } | null>;
  rechargeWithStones: (stones: number) => Promise<{ success: boolean; message: string } | null>;
  clearStress: () => Promise<{ success: boolean; message: string } | null>;
  setShowChangelog: (v: boolean) => void;
  achievements: any[];
  earnedAchievements: any[];
  selectedTitle: string;
  selectedFrame: string;
  fetchAchievements: () => Promise<void>;
  claimAchievement: (achievementId: string) => Promise<{ success: boolean; message: string } | null>;
  selectTitleFrame: (title: string, frame: string) => Promise<{ success: boolean; message: string } | null>;
  sacrificeNPC: (npcId: string) => Promise<{ success: boolean; message: string } | null>;
  exploreNode: (nodeId: string) => Promise<{ player: Player; log: string; combat_triggered: boolean; skip_result?: any } | null>;
  executeCombatTurn: (equippedGuIds: string[], isCharge?: boolean, targetId?: string, targetX?: number, targetY?: number) => Promise<{ player: Player; logs: string[]; enemy: CombatState | null; victory: boolean; defeat: boolean } | null>;
  executeAutoCombat: (loadout: any) => Promise<{ logs: any[]; winner: string; combatants?: any[] } | null>;
  executePracticeCombat: (loadout: any, dummyConfig: any) => Promise<{ logs: any[]; winner: string; combatants?: any[], player_won?: boolean } | null>;
  combatHistory: CombatHistoryEntry[];
  combatHistoryLoading: boolean;
  fetchCombatHistory: () => Promise<void>;
  fetchCombatHistoryEntry: (id: string) => Promise<CombatHistoryEntry | null>;
  auditLogs: any[];
  auditLogsLoading: boolean;
  fetchAuditLogs: (filters?: { action?: string; entity_type?: string; success?: boolean; limit?: number; offset?: number }) => Promise<void>;
  referralsData: {
    ref_code: string;
    referees: any[];
    total_ref: number;
    has_referrer?: boolean;
    can_bind_ref?: boolean;
  } | null;
  fetchReferrals: () => Promise<void>;
  claimReferralReward: (refereeId: string, milestone: number) => Promise<{success: boolean; message?: string}>;
  bindReferralCode: (refCode: string) => Promise<{success: boolean; message?: string}>;
  passiveStats: { stats: Record<string, number>; details: { key: string; total: number; sources: { source: string; name: string; value: number; mult: number; dao: number; final: number }[] }[] } | null;
  fetchPassiveStats: () => Promise<void>;
  simulateDeposit: (stones: number) => Promise<boolean>;
  addLog: (message: string) => void;
  setView: (view: string) => void;
  retreat: () => Promise<void>;
  fleeCombat: () => Promise<any>;
  activeCombatReplay: { logs: any[]; winner: string; combatants: any[]; enemy: CombatState } | null;
  setActiveCombatReplay: (replay: { logs: any[]; winner: string; combatants: any[]; enemy: CombatState } | null) => void;
  combatSettings: AutoCombatConfig | null;
  combatSettingsLoading: boolean;
  fetchCombatSettings: () => Promise<void>;
  saveCombatSettings: (settings: AutoCombatConfig) => Promise<boolean>;
  equipGu: (instanceId: string, equip: boolean) => Promise<void>;
  unequipAllGu: () => Promise<void>;
  updateGuStatus: (instanceId: string, status: string) => Promise<void>;
  setVitalGu: (instanceId: string) => Promise<void>;
  useGu: (instanceId: string) => Promise<void>;
  useGus: (instanceIds: string[]) => Promise<boolean>;
  useItem: (itemId: string) => Promise<void>;
  useItems: (itemIds: string[]) => Promise<boolean>;
  movePlayer: (x: number, y: number, attackTargetId?: string) => Promise<{ player: Player; log: string; combat_triggered: boolean; skip_result?: any } | null>;
  fetchGrid: () => Promise<{ x: number; y: number; vision_radius: number; cells: any[]; stats: any; players_on_map?: any[]; world_width?: number; world_height?: number } | null>;
  dropGu: (instanceId: string) => Promise<boolean>;
  dropGus: (instanceIds: string[]) => Promise<boolean>;
  buyShopItem: (itemId: string, quantity: number, rank?: number) => Promise<boolean>;
  sellShopItem: (itemId: string, quantity: number, itemType: "Gu" | "Material" | "Recipe", rank?: number) => Promise<boolean>;
  sellShopItemsBulk: (items: { item_id: string; quantity: number; item_type: "Gu" | "Material" | "Recipe"; rank?: number }[]) => Promise<boolean>;
  marketplaceListings: any[];
  fetchMarketplaceListings: () => Promise<void>;
  createMarketplaceListing: (payload: { itemType: "Gu" | "Material" | "Recipe"; guInstanceId?: string; guInstanceIds?: string[]; itemId?: string; quantity: number; price: number; durationHours: number; currency: "primeval" | "immortal"; listingType: "MARKET" | "AUCTION" | "TRADE"; targetBuyerUsername?: string }) => Promise<boolean>;
  placeMarketplaceBid: (listingId: string, amount: number) => Promise<boolean>;
  cancelMarketplaceListing: (listingId: string) => Promise<boolean>;
  buyMarketplaceListing: (listingId: string, quantity?: number) => Promise<boolean>;
  activeTrade: any | null;
  fetchActiveTrade: () => Promise<void>;
  initiateTrade: (receiverUsername: string) => Promise<boolean>;
  acceptTrade: (tradeId: string) => Promise<boolean>;
  declineTrade: (tradeId: string) => Promise<boolean>;
  cancelTrade: (tradeId: string) => Promise<boolean>;
  updateTradeOffer: (tradeId: string, offer: { primeval_stones: number; immortal_stones: number; gus: string[]; materials: Record<string, number>; recipes: Record<string, number> }) => Promise<boolean>;
  lockTrade: (tradeId: string) => Promise<boolean>;
  confirmTrade: (tradeId: string) => Promise<boolean>;

  // Party State & Actions
  party: { id: string; leader_id: string; members: { id: string; username: string; avatar_url?: string }[] } | null;
  partyInvitations: { id: string; party_id: string; inviter_name: string; inviter_avatar?: string }[];
  createParty: () => Promise<boolean>;
  fetchPartyStatus: () => Promise<void>;
  fetchPartyInvitations: () => Promise<void>;
  invitePlayer: (username: string) => Promise<boolean>;
  respondPartyInvitation: (id: string, action: "accept" | "reject") => Promise<boolean>;
  leaveParty: () => Promise<boolean>;

  // Arena State & Actions
  arenaLeaderboard: any[];
  arenaDailyChallenges: number;
  arenaShopItems: any[];
  fetchArenaLeaderboard: () => Promise<void>;
  fetchArenaShop: () => Promise<void>;
  challengeArena: (targetPlayerId: string) => Promise<boolean>;
  buyArenaItem: (itemId: string, quantity?: number) => Promise<boolean>;

  // Death Arena State & Actions
  deathArenaTournament: DeathArenaTournament | null;
  deathArenaBracket: DeathArenaMatch[];
  deathArenaMyTeam: DeathArenaTeam | null;
  deathArenaMyTeamMembers: DeathArenaTeamMember[];
  deathArenaMyMatches: DeathArenaMatch[];
  deathArenaRewards: DeathArenaReward[];
  deathArenaHistory: DeathArenaTournament[];
  deathArenaLoading: boolean;
  fetchDeathArenaCurrent: () => Promise<void>;
  registerDeathArena: () => Promise<boolean>;
  unregisterDeathArena: () => Promise<boolean>;
  fetchDeathArenaBracket: () => Promise<void>;
  fetchDeathArenaMyTeam: () => Promise<void>;
  fetchDeathArenaMyMatches: () => Promise<void>;
  fetchDeathArenaRewards: () => Promise<void>;
  fetchDeathArenaHistory: () => Promise<void>;
  claimDeathArenaReward: (rewardId: string) => Promise<boolean>;

  // Leaderboard State & Actions
  leaderboardEntries: any[];
  leaderboardCategory: string;
  myLeaderboardRank: { rank: number; value: number } | null;
  leaderboardTotalPlayers: number;
  leaderboardShowOnLeaderboard: boolean;
  leaderboardLoading: boolean;
  leaderboardLastUpdated: string;
  fetchLeaderboard: (category: string) => Promise<void>;
  toggleLeaderboardVisibility: () => Promise<void>;
  fetchLeaderboardSettings: () => Promise<void>;

  // Chat/Mail State & Actions
  inbox: any[];
  baoHoangMessages: any[];
  activeChannel: "world" | "general" | "sect";
  chatOpen: boolean;
  chatMessages: { world: any[]; general: any[]; sect: any[] };
  unreadChannels: { world: boolean; general: boolean; sect: boolean };
  setActiveChannel: (channel: "world" | "general" | "sect") => void;
  setChatOpen: (open: boolean) => void;
  subscribeAllBaoHoangMessages: () => void;
  unsubscribeAllBaoHoangMessages: () => void;
  sendPrivateMail: (receiverUsername: string, content: string) => Promise<boolean>;
  fetchInbox: () => Promise<void>;
  sendBaoHoangChat: (content: string, channel: string) => Promise<boolean>;
  fetchBaoHoangMessages: (channel: string) => Promise<void>;

  // Notification State & Actions
  notifications: PlayerNotification[];
  notificationsLoaded: boolean;
  unreadNotifications: number;
  fetchNotifications: () => Promise<void>;
  fetchUnreadNotificationsCount: () => Promise<void>;
  markNotificationRead: (id: string) => Promise<void>;
  markAllNotificationsRead: () => Promise<void>;

  // Mine State & Actions
  mineSlots: MineSlot[];
  fetchMineSlots: () => Promise<void>;
  occupyMineSlot: (slotId: number) => Promise<boolean>;
  claimMineRewards: (slotId: number) => Promise<boolean>;
  abandonMineSlot: (slotId: number) => Promise<boolean>;
  attackMineSlot: (slotId: number) => Promise<boolean>;

  // Pet State & Actions
  playerPets: PlayerPet[];
  playerPetsLoaded: boolean;
  fetchPlayerPets: () => Promise<void>;
  feedPet: (petId: string, itemId: string) => Promise<boolean>;
  setPetFormation: (petIds: string[]) => Promise<boolean>;
  setPetSleep: (petId: string, sleeping: boolean) => Promise<boolean>;
  releasePet: (petId: string) => Promise<boolean>;
  mergePets: (targetId: string, sourceIds: string[]) => Promise<boolean>;
  captureMonster: () => Promise<{ captured: boolean; message: string } | null>;

  // Toast System Actions (state lives in useUIStore)
  addToast: (message: string, type?: "success" | "error" | "info", duration?: number) => void;
  removeToast: (id: string) => void;

  // Confirm Dialog Actions (state lives in useUIStore)
  showConfirm: (message: string, options?: { confirmText?: string; cancelText?: string }) => Promise<boolean>;
  closeConfirm: (result: boolean) => void;

  // Immortal Aperture State & Actions
  aperture: ImmortalAperture | null;
  daoMarks: DaoMark[];
  apertureNodes: ApertureResourceNode[];
  variantHumans: ApertureVariantHuman[];
  activeTribulation: { event: TribulationEvent; enemies: TribulationEnemy[] } | null;
  fetchAperture: () => Promise<void>;
  fetchDaoMarks: () => Promise<void>;
  fetchApertureNodes: () => Promise<void>;
  fetchVariantHumans: () => Promise<void>;
  placeApertureNode: (nodeType: string, x: number, y: number, path?: string, prodKey?: string) => Promise<boolean>;
  collectResourceNode: (nodeId: string) => Promise<boolean>;
  collectAllResourceNodes: () => Promise<boolean>;
  uninstallApertureNode: (nodeId: string) => Promise<boolean>;
  deleteApertureNode: (nodeId: string) => Promise<boolean>;
  setApertureTimeRate: (rate: number) => Promise<boolean>;
  setApertureLabor: (race: string, intensity: string) => Promise<boolean>;
  repairApertureNode: (nodeId: string) => Promise<boolean>;
  repairAllApertureNodes: () => Promise<boolean>;
  upgradeApertureNode: (nodeId: string) => Promise<boolean>;
  fetchActiveTribulation: () => Promise<void>;
  engageTribulationCombat: (eventId: string) => Promise<boolean>;
  resolveTribulation: (eventId: string) => Promise<boolean>;

  // Soul foundation actions
  useGutsGu: () => Promise<boolean>;
  refineSoul: () => Promise<boolean>;
  // Instance actions
  activeInstanceSession: PlayerInstanceSession | null;
  activeInstanceTemplate: InstanceTemplate | null;
  availableInstances: InstanceListItem[];
  instanceDailyEntriesUsed: number;
  instanceDailyEntriesMax: number;
  fetchInstances: () => Promise<void>;
  fetchActiveInstance: () => Promise<void>;
  enterInstance: (instanceId: string) => Promise<boolean>;
  actionInstance: (optionIndex: number) => Promise<boolean>;

  // Sect State & Actions
  sects: Sect[];
  mySectStatus: MySectStatus | null;
  sectMissions: SectMission[];
  sectMembers: SectMemberInfo[];
  sectJoinRequests: SectJoinRequest[];
  sectPromotionRequests: SectPromotionRequest[];
  sectMines: SectMine[];
  fetchSects: () => Promise<void>;
  joinSect: (sectId: string) => Promise<boolean>;
  cancelSectJoinRequest: () => Promise<boolean>;
  leaveSect: () => Promise<boolean>;
  fetchMySectStatus: () => Promise<void>;
  promoteSectRank: () => Promise<boolean>;
  donateSectStones: (amount: number) => Promise<boolean>;
  donateSectMaterials: (materialId: string, amount: number) => Promise<boolean>;
  donateSectGu: (guInstanceId: string) => Promise<boolean>;
  donateSectRecipe: (recipeId: string) => Promise<boolean>;
  donateSectBulk: (payload: {
    materials: { material_id: string; amount: number }[];
    gus: { gu_instance_id: string }[];
    recipes: { recipe_id: string }[];
  }) => Promise<boolean>;
  buyFromSectVault: (itemType: string, itemId: string, quantity?: number, rank?: number) => Promise<boolean>;
  fetchSectMissions: () => Promise<void>;
  acceptSectMission: (missionId: string) => Promise<boolean>;
  completeSectMission: (missionId: string) => Promise<boolean>;
  deleteSectMission: (missionId: string) => Promise<boolean>;
  abandonSectMission: (missionId: string) => Promise<boolean>;
  createSect: (name: string, description: string, region: string) => Promise<boolean>;
  fetchSectMembers: () => Promise<void>;
  setMemberRank: (memberId: string, rank: string) => Promise<boolean>;
  kickMember: (memberId: string) => Promise<boolean>;
  createCustomMission: (payload: {
    title: string;
    description: string;
    mission_type: string;
    target_id: string;
    target_amount: number;
    reward_contribution: number;
    required_member_rank: string;
    duration_hours?: number;
  }) => Promise<boolean>;
  stockSectVault: (itemType: 'gu' | 'material' | 'recipe', itemId: string, quantity: number) => Promise<boolean>;
  stockSectVaultBulk: (items: { item_type: 'gu' | 'material' | 'recipe'; item_id: string; quantity: number }[]) => Promise<boolean>;
  upgradeSect: () => Promise<boolean>;
  toggleSectApproval: (requireApproval: boolean) => Promise<boolean>;
  toggleSectPromotionApproval: (requirePromotionApproval: boolean) => Promise<boolean>;
  paySectSalary: () => Promise<boolean>;
  fetchJoinRequests: () => Promise<void>;
  actionJoinRequest: (requestId: string, action: 'approve' | 'reject') => Promise<boolean>;
  fetchPromotionRequests: () => Promise<void>;
  actionPromotionRequest: (requestId: string, action: "approve" | "reject") => Promise<boolean>;
  fetchSectMines: () => Promise<void>;
  attackSectMine: (slotId: number, attackerId?: string) => Promise<boolean>;
  assignSectMineDefender: (mineId: number, defenderType: 'member' | 'beast', playerId?: string, monsterId?: string, monsterRank?: number) => Promise<boolean>;
  removeSectMineDefender: (defenderId: string) => Promise<boolean>;

  // Feedback State & Actions
  feedbacks: Feedback[];
  feedbackComments: Record<string, FeedbackComment[]>;
  fetchFeedbacks: (category?: string, status?: string) => Promise<void>;
  submitFeedback: (title: string, description: string, category: string) => Promise<boolean>;
  voteFeedback: (feedbackId: string, voteType: number) => Promise<boolean>;
  updateFeedbackStatus: (feedbackId: string, status: string, developerResponse: string) => Promise<boolean>;
  fetchFeedbackComments: (feedbackId: string) => Promise<boolean>;
  addFeedbackComment: (feedbackId: string, content: string) => Promise<boolean>;
  deleteFeedbackComment: (feedbackId: string, commentId: string) => Promise<boolean>;
}

export interface MineSlot {
  id: number;
  rarity: string;
  mine_type?: string;
  occupier_id?: string;
  occupier_name?: string;
  monster_id?: string;
  occupied_at?: string;
  last_claimed_at?: string;
  accumulated_stones: number;
  accumulated_qi: number;
  accumulated_chests: number;
}


const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
const AUTHENTICATED_SESSION = "cookie-session";

const getCookie = (name: string): string | null => {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(new RegExp(`(?:^|; )${name}=([^;]*)`));
  return match ? decodeURIComponent(match[1]) : null;
};

// Track latest request timestamp per URL to reject stale responses
const pendingTimestamps = new Map<string, number>();
const getAbortSignal = (key: string, timeoutMs = 30000): AbortSignal => {
  const controller = new AbortController();
  const id = Date.now();
  pendingTimestamps.set(key, id);
  setTimeout(() => controller.abort(), timeoutMs);
  const originalAddEventListener = controller.signal.addEventListener.bind(controller.signal);
  controller.signal.addEventListener = (type: string, listener: EventListenerOrEventListenerObject, options?: AddEventListenerOptions | boolean) => {
    if (type === "abort") {
      originalAddEventListener("abort", () => {
        if (pendingTimestamps.get(key) === id) pendingTimestamps.delete(key);
      });
    }
    originalAddEventListener(type, listener, options);
  };
  return controller.signal;
};

// Deduplicate concurrent fetches by key so multiple callers share one in-flight request.
const pendingFetches = new Map<string, Promise<void>>();
const fetchWithDedupe = async (key: string, fetcher: () => Promise<void>): Promise<void> => {
  const existing = pendingFetches.get(key);
  if (existing) return existing;
  const promise = fetcher().finally(() => pendingFetches.delete(key));
  pendingFetches.set(key, promise);
  return promise;
};

const authFetch = (input: RequestInfo | URL, init: RequestInit & { timeoutMs?: number } = {}) => {
  const csrfToken = getCookie("gu_csrf_token");
  const headers: Record<string, string> = {};
  if (init.headers) {
    // Flatten headers
    if (Array.isArray(init.headers)) {
      for (const [k, v] of init.headers) headers[k] = v;
    } else if (init.headers instanceof Headers) {
      init.headers.forEach((v, k) => { headers[k] = v; });
    } else {
      Object.assign(headers, init.headers);
    }
  }
  if (csrfToken && init.method && init.method !== "GET" && init.method !== "HEAD") {
    headers["X-CSRF-Token"] = csrfToken;
  }
  if (typeof window !== "undefined") {
    const token = localStorage.getItem("gu_auth_token");
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }
  }
  // Add abort signal with timeout, keyed by URL string (default 30s)
  const key = typeof input === "string" ? input : input.toString();
  const signal = getAbortSignal(key, init.timeoutMs ?? 30000);
  return fetch(input, {
    ...init,
    headers,
    credentials: "include",
    signal,
  });
};

// ── apiFetch: wraps authFetch with automatic error toasts ────────────
// Usage: const res = await apiFetch(url, opts)
//   opts.silent = true  → skip auto-toast (handler manages errors itself)
//   opts.raw     = true → return raw Response without auto error handling
interface ApiFetchInit extends RequestInit {
  silent?: boolean;
  raw?: boolean;
}

const apiFetch = async (input: RequestInfo | URL, init: ApiFetchInit = {}): Promise<Response> => {
  const { silent, raw, ...fetchInit } = init;
  try {
    const response = await authFetch(input, fetchInit);

    if (raw) return response;

    if (response.status === 401) {
      if (!silent) {
        const state = useGameStore.getState();
        let errorMsg = "Phiên làm việc hết hạn. Vui lòng đăng nhập lại.";
        try {
          const errData = await response.clone().json();
          if (errData.error) {
            errorMsg = errData.error;
          }
        } catch {}
        state.addToast(errorMsg, "error");
        state.logout();
      }
      return response;
    }

    if (!response.ok && !silent) {
      let errorMsg = "Lỗi từ server";
      try {
        const errData = await response.clone().json();
        errorMsg = errData.error || errorMsg;
      } catch {}
      useGameStore.getState().addToast(errorMsg, "error");
    }

    return response;
  } catch (err: any) {
    if (!silent) {
      const msg = err?.name === "AbortError" ? "Yêu cầu đã hết thời gian" : (err.message || "Lỗi kết nối");
      useGameStore.getState().addToast(msg, "error");
    }
    throw err;
  }
};

const PLAYER_EVENT_TYPES = [
  "mana.snapshot",
  "player.updated",
  "combat.started",
  "combat.ended",
  "trade.updated",
  "player.heartbeat",
] as const;

type PlayerEventType = (typeof PLAYER_EVENT_TYPES)[number];

type PlayerRealtimeEvent = {
  type: PlayerEventType;
  data: Partial<Player> & { id: string; aperture_stress_level?: string };
};

export function guSpecFor(guId?: string, rank?: number): GuSpec | undefined {
  if (!guId) return undefined;
  const state = useGameStore.getState();
  const byRank = state.guByRank[guId];
  if (byRank) {
    if (rank != null && byRank[rank]) return byRank[rank];
    const ranks = Object.keys(byRank).map(Number);
    const maxRank = ranks.length ? Math.max(...ranks) : 0;
    if (maxRank && byRank[maxRank]) return byRank[maxRank];
  }
  return state.guDictionary[guId];
}

export function formatRecipeName(recipeId: string): string {
  const targetGuID = recipeId.replace("recipe_", "").replace(/_\d+_\d+$/, "").toUpperCase();
  const match = recipeId.match(/_(\d+)_(\d+)$/);
  const state = useGameStore.getState();
  const name = state.guDictionary[targetGuID]?.name || recipeId;
  
  if (match) {
    const rank = match[1];
    const variant = match[2];
    if (variant === "0" || variant === "1") {
      return `Bí phương: ${name} (Bậc ${rank})`;
    }
    return `Bí phương: ${name} (Bậc ${rank}, Phân nhánh ${variant})`;
  }
  return `Bí phương: ${name}`;
}

export const useGameStore = create<GameState>((set, get) => {
  const normalizePlayer = (player: Player | null): Player | null => {
    if (!player) return null;
    const guInventory = (player.gu_inventory || []).map((g) => ({ ...g, equipped: !!g.equipped || !!g.is_vital }));
    const equippedSet = new Set<string>(player.equipped_gus || []);
    const vital = guInventory.find((g) => g.is_vital);
    if (vital && vital.status !== "DEAD") equippedSet.add(vital.instance_id);
    return {
      ...player,
      current_thought: player.current_thought ?? 100,
      max_thought: player.max_thought ?? 100,
      thought_regen_rate: player.thought_regen_rate ?? 1.0,
      gu_inventory: guInventory,
      equipped_gus: Array.from(equippedSet),
    };
  };

  const setPlayerFromServer = (player: Player | Partial<Player> | null, extraState: Partial<GameState> = {}) => {
    if (!player) {
      set({ ...extraState, player: null, manaSnapshot: null });
      return;
    }
    const currentPlayer = get().player;
    const fullPlayer: Player = currentPlayer ? {
      ...currentPlayer,
      ...player,
      gu_inventory: player.gu_inventory ?? currentPlayer.gu_inventory,
      materials_inventory: player.materials_inventory ?? currentPlayer.materials_inventory,
      completed_quests: player.completed_quests ?? currentPlayer.completed_quests,
      unlocked_recipes: player.unlocked_recipes ?? currentPlayer.unlocked_recipes,
    } : (player as Player);

    const normalizedPlayer = normalizePlayer(fullPlayer);
    if (normalizedPlayer && !normalizedPlayer.materials_inventory) {
      normalizedPlayer.materials_inventory = {};
    }
    // Preserve a richer gu_inventory already in state when the incoming
    // snapshot is sparse (e.g. equipped-only from /auth/me or challenge).
    const incomingInv = normalizedPlayer?.gu_inventory || [];
    const currentInv = currentPlayer?.gu_inventory || [];
    let guInventory = incomingInv;
    let guComplete = false;
    if (normalizedPlayer?.is_sparse_gu_inventory && currentInv.length > 0 && incomingInv.length > 0 && currentInv.length > incomingInv.length) {
      guInventory = currentInv;
      guComplete = true;
    }
    if (normalizedPlayer) {
      normalizedPlayer.gu_inventory = guInventory;
    }
    set({
      ...extraState,
      player: normalizedPlayer,
      guInventoryLoaded: guComplete || (normalizedPlayer ? incomingInv.length > 0 : false),
      manaSnapshot: normalizedPlayer ? createManaSnapshotFromPlayer(normalizedPlayer, get().guDictionary, get().constitutionDictionary, get().aperture?.time_rate) : null,
    });
    if (normalizedPlayer) {
      get().subscribeAllBaoHoangMessages();
    }
  };

  const translateCache = new Map<string, string>();

  const translateIDs = (msg: string): string => {
    if (!msg) return "";
    const cacheKey = `${msg}\x00${JSON.stringify(get().guDictionary || {})}\x00${JSON.stringify(get().itemDictionary || {})}\x00${JSON.stringify(get().monsterDictionary || {})}`;
    const cached = translateCache.get(cacheKey);
    if (cached) return cached;

    let formatted = msg;
    const { guDictionary, itemDictionary, monsterDictionary } = get();

    // Replace Gu IDs
    Object.keys(guDictionary || {}).forEach((id) => {
      const name = guDictionary[id]?.name;
      if (name && formatted.includes(id)) {
        formatted = formatted.split(id).join(name);
      }
    });

    // Replace Item IDs
    Object.keys(itemDictionary || {}).forEach((id) => {
      const name = itemDictionary[id]?.name;
      if (name && formatted.includes(id)) {
        formatted = formatted.split(id).join(name);
      }
    });

    // Replace Monster IDs
    Object.keys(monsterDictionary || {}).forEach((id) => {
      const name = monsterDictionary[id]?.name;
      if (name && formatted.includes(id)) {
        formatted = formatted.split(id).join(name);
      }
    });

    translateCache.set(cacheKey, formatted);
    if (translateCache.size > 500) {
      const firstKey = translateCache.keys().next().value;
      if (firstKey) translateCache.delete(firstKey);
    }
    return formatted;
  };

  return ({
  player: null,
  manaSnapshot: null,
  token: null,
  isLoading: false,
  actionLoading: false,
  error: null,
  arenaDailyChallenges: 0,
  playerEventsConnected: false,
  playerEventsRetries: 0,
  guInventoryLoaded: false,
  logs: ["Chào mừng đạo hữu đến với Cổ Giới. Hãy tu luyện và sinh tồn."],
  referralsData: null,
  passiveStats: null,
  guDictionary: {},
  guByRank: {},
  itemDictionary: {},
  monsterDictionary: {},
  constitutionDictionary: {},
  raceDictionary: {},
  dictionaryLoaded: false,
  recipes: [],
  recipesLoaded: false,
  quests: [],
  gachaPools: [],
  currentBettingRound: null,
  myBetTeam: null,
  myBetAmount: null,
  bettingHistory: [],
  achievements: [],
  earnedAchievements: [],
  selectedTitle: '',
  selectedFrame: '',
  regions: [],
  regionsLoaded: false,
  world: null,
  killerMoves: [],
  killerMovesLoaded: false,
  combatSettings: null,
  combatSettingsLoading: false,
  combatHistory: [],
  combatHistoryLoading: false,
  inspectedProfile: null,
  inspectedUsernameQuery: null,
  profileLoading: false,
  currentView: "dashboard",
  marketplaceListings: [],
  activeTrade: null,
  party: null,
  partyInvitations: [],
  arenaLeaderboard: [],
  arenaShopItems: [],
  deathArenaTournament: null,
  deathArenaBracket: [],
  deathArenaMyTeam: null,
  deathArenaMyTeamMembers: [],
  deathArenaMyMatches: [],
  deathArenaRewards: [],
  deathArenaHistory: [],
  deathArenaLoading: false,
  leaderboardEntries: [],
  leaderboardCategory: "rank",
  myLeaderboardRank: null,
  leaderboardTotalPlayers: 0,
  leaderboardShowOnLeaderboard: true,
  leaderboardLoading: false,
  leaderboardLastUpdated: "",
  inbox: [],
  baoHoangMessages: [],
  activeChannel: "world",
  chatOpen: false,
  chatMessages: { world: [], general: [], sect: [] },
  unreadChannels: { world: false, general: false, sect: false },
  notifications: [],
  notificationsLoaded: false,
  unreadNotifications: 0,
  mineSlots: [],
  aperture: null,
  daoMarks: [],
  apertureNodes: [],
  variantHumans: [],
  activeTribulation: null,
  activeInstanceSession: null,
  activeInstanceTemplate: null,
  availableInstances: [],
  instanceDailyEntriesUsed: 0,
  instanceDailyEntriesMax: 2,
  sects: [],
  mySectStatus: null,
  sectMissions: [],
  sectMembers: [],
  sectJoinRequests: [],
  sectPromotionRequests: [],
  sectMines: [],
  feedbacks: [],
  feedbackComments: {},

  setView: (view) => useUIStore.getState().setView(view),
  setShowChangelog: (v) => useUIStore.getState().setShowChangelog(v),
  addToast: (message, type = "info", duration = 4000) => {
    const formatted = translateIDs(message);
    useUIStore.getState().addToast(formatted, type, duration);
  },
  removeToast: (id) => useUIStore.getState().removeToast(id),
  showConfirm: async (message, options) => {
    return useUIStore.getState().showConfirm(message, options);
  },
  closeConfirm: (result) => {
    useUIStore.getState().closeConfirm(result);
  },

  login: async (email, password) => {
    set({ isLoading: true, error: null });
    try {
      const fingerprint = await get().getBrowserFingerprint();
      const response = await authFetch(`${BACKEND_URL}/api/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password, fingerprint }),
      });

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || "Mật khẩu hoặc Email không chính xác.");
      }

      const data: AuthResponse = await response.json();
      setPlayerFromServer(data.player, { token: data.token || AUTHENTICATED_SESSION });
      if (data.token) {
        localStorage.setItem("gu_auth_token", data.token);
      }
      localStorage.setItem("gu_player_id", data.player.id);
      
      void get().fetchRecipes();
      void get().fetchGuInventory();
      get().subscribePlayerEvents();
      get().subscribeTradeEvents();
      void get().fetchNotifications();
      get().addLog(`Đồng chí ${data.player.username} đã quy vị!`);
      const today = new Date().toISOString().split('T')[0];
      const hiddenUntil = typeof window !== "undefined" ? localStorage.getItem("gu_changelog_hide_until") : null;
      if (hiddenUntil !== today) {
        useUIStore.getState().setShowChangelog(true);
      }
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      return false;
    } finally {
      set({ isLoading: false });
    }
  },

  sendOTP: async (email) => {
    set({ isLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/auth/send-otp`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email }),
      });

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || "Gửi OTP thất bại.");
      }

      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      return false;
    } finally {
      set({ isLoading: false });
    }
  },
  
  getBrowserFingerprint: async (): Promise<string> => {
    if (typeof window === "undefined") {
      return "server-ssr-fallback";
    }
    try {
      const fp = await FingerprintJS.load();
      const result = await fp.get();
      return result.visitorId;
    } catch (e) {
      return "fingerprint-error";
    }
  },

  register: async (
    username,
    password,
    email,
    startingRegion,
    setVitalNow,
    referralCode,
    attackGuId,
    defenseGuId,
    vitalGuChoice,
    otp,
    race?,
    gender?
  ) => {
    set({ isLoading: true, error: null });
    try {
      const fingerprint = await get().getBrowserFingerprint();
      const response = await authFetch(`${BACKEND_URL}/api/auth/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          username,
          password,
          email,
          starting_region: startingRegion,
          set_vital_now: setVitalNow,
          referral_code: referralCode || "",
          attack_gu_id: attackGuId,
          defense_gu_id: defenseGuId,
          vital_gu_choice: vitalGuChoice,
          otp: otp || "",
          race: race || "human",
          gender: gender || "male",
          fingerprint,
        }),
      });

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || "Không thể khai mở Không Khiếu.");
      }

      const data: AuthResponse = await response.json();
      setPlayerFromServer(data.player, { token: data.token || AUTHENTICATED_SESSION });
      if (data.token) {
        localStorage.setItem("gu_auth_token", data.token);
      }
      localStorage.setItem("gu_player_id", data.player.id);

      void get().fetchRecipes();
      void get().fetchGuInventory();
      get().subscribePlayerEvents();
      get().subscribeTradeEvents();
      void get().fetchNotifications();
      get().addLog(`Cổ sư ${data.player.username} khai mở Không Khiếu! Tư chất: ${data.player.aptitude_type} (${data.player.aptitude_percent}%)`);
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      return false;
    } finally {
      set({ isLoading: false });
    }
  },

  completeOnboarding: async (data) => {
    set({ isLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/complete-onboarding`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
      });

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || "Không thể hoàn tất thiết lập.");
      }

      const player: Player = await response.json();
      setPlayerFromServer(player);
      get().addLog(`Thiết lập ban đầu hoàn tất! Khu vực: ${player.current_region}`);
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      return false;
    } finally {
      set({ isLoading: false });
    }
  },

  logout: () => {
    get().unsubscribePlayerEvents();
    get().unsubscribeTradeEvents();
    get().unsubscribeBaoHoangMessages();
    void authFetch(`${BACKEND_URL}/api/auth/logout`, { method: "POST" });
    set({ player: null, manaSnapshot: null, token: null, recipes: [], recipesLoaded: true, quests: [], guInventoryLoaded: false });
    localStorage.removeItem("gu_auth_token");
    localStorage.removeItem("gu_player_id");
    get().addLog("Đã rời khỏi thế giới Cổ sư.");
  },

  clearError: () => set({ error: null }),
  updateManaSnapshot: (player) => {
    const normalizedPlayer = normalizePlayer(player);
    set({
      manaSnapshot: normalizedPlayer ? createManaSnapshotFromPlayer(normalizedPlayer, get().guDictionary, get().constitutionDictionary, get().aperture?.time_rate) : null,
    });
  },
  setPlayerFromServer: (player) => setPlayerFromServer(player),
  getEstimatedMana: () => estimateManaFromSnapshot(get().manaSnapshot),

  subscribeEvents: () => {
    if (typeof window === "undefined" || !get().token) return;
    if (get().playerEventsConnected) return; // Already connected

    const token = get().token!;

    // Acquire connection with reference counting
    const releaseConnection = wsManager.acquireConnection(token);

    try {
      wsManager.connect(token).then(() => {
        set({ playerEventsConnected: true, playerEventsRetries: 0 });
      }).catch((err) => {
        console.warn("[WebSocket] Connection failed", err);
        releaseConnection();
      });

      // Listen for player updates
      const unsubscribePlayer = wsManager.subscribe("player", (payload: any) => {
        try {
          const eventData = payload?.data || payload;
          const eventType = payload?.type;

          // Handle delta events (efficient updates)
          if (eventType === "player.position") {
            const currentPlayer = get().player;
            if (!currentPlayer) return;
            const newX = eventData?.x ?? currentPlayer.x;
            const newY = eventData?.y ?? currentPlayer.y;
            if (newX !== currentPlayer.x || newY !== currentPlayer.y) {
              set({
                player: { ...currentPlayer, x: newX, y: newY },
              });
            }
            return;
          }

          if (eventType === "player.hp") {
            const currentPlayer = get().player;
            if (!currentPlayer) return;
            const newHp = eventData?.hp ?? currentPlayer.hp;
            const newMaxHp = eventData?.max_hp ?? currentPlayer.base_hp;
            set({
              player: { ...currentPlayer, hp: newHp, base_hp: newMaxHp },
              manaSnapshot: createManaSnapshotFromPlayer(
                { ...currentPlayer, hp: newHp, base_hp: newMaxHp },
                get().guDictionary,
                get().constitutionDictionary,
                get().aperture?.time_rate
              ),
            });
            return;
          }

          if (eventType === "player.mana") {
            const currentPlayer = get().player;
            if (!currentPlayer) return;
            const newCurrentMana = eventData?.current_mana ?? currentPlayer.current_mana;
            const newMaxMana = eventData?.max_mana ?? currentPlayer.max_mana;
            set({
              player: { ...currentPlayer, current_mana: newCurrentMana, max_mana: newMaxMana },
              manaSnapshot: createManaSnapshotFromPlayer(
                { ...currentPlayer, current_mana: newCurrentMana, max_mana: newMaxMana },
                get().guDictionary,
                get().constitutionDictionary,
                get().aperture?.time_rate
              ),
            });
            return;
          }

          if (eventType === "player.combat_state" || eventType === "combat.started" || eventType === "combat.ended") {
            const currentPlayer = get().player;
            if (!currentPlayer) return;
            const nextCombat = eventData?.active_combat || null;
            const updatedPlayer = {
              ...currentPlayer,
              active_combat: nextCombat,
            };
            set({
              player: updatedPlayer,
              ...(nextCombat === null || eventType === "combat.ended" ? { activeCombatReplay: null } : {}),
              manaSnapshot: updatedPlayer
                ? createManaSnapshotFromPlayer(
                    updatedPlayer,
                    get().guDictionary,
                    get().constitutionDictionary,
                    get().aperture?.time_rate
                  )
                : get().manaSnapshot,
            });
            if (eventType === "combat.started") {
              get().addLog("Chiến đấu realtime đã bắt đầu.");
            } else if (eventType === "combat.ended") {
              get().addLog("Chiến đấu realtime đã kết thúc.");
            }
            return;
          }

          // Full snapshot fallback (for initial connection and other events)
          if (!eventData?.id) return;

          const currentPlayer = get().player;
          if (!currentPlayer || currentPlayer.id !== eventData.id) return;

          if (eventType === "session.kicked") {
            const state = get();
            state.addToast("Tài khoản đã đăng nhập từ thiết bị khác. Bạn đã bị đăng xuất.", "error");
            state.logout();
            return;
          }

          const { active_combat, ...rest } = eventData;
          const nextCombat = active_combat !== undefined ? active_combat : currentPlayer.active_combat;
          const updatedPlayer = normalizePlayer({
            ...currentPlayer,
            ...rest,
            active_combat: nextCombat,
          });
          set({
            player: updatedPlayer,
            ...(nextCombat === null || eventType === "combat.ended" ? { activeCombatReplay: null } : {}),
            manaSnapshot: updatedPlayer
              ? createManaSnapshotFromPlayer(
                  updatedPlayer,
                  get().guDictionary,
                  get().constitutionDictionary,
                  get().aperture?.time_rate
                )
              : get().manaSnapshot,
          });

          if (eventType === "combat.started") {
            get().addLog("Chiến đấu realtime đã bắt đầu.");
          } else if (eventType === "combat.ended") {
            get().addLog("Chiến đấu realtime đã kết thúc.");
          }
        } catch (err) {
          console.error("Failed to process player websocket event", err);
        }
      });

      // Listen for trade updates
      const unsubscribeTrade = wsManager.subscribe("trade", (payload: any) => {
        try {
          set({ activeTrade: payload?.trade ?? null });
        } catch (err) {
          console.error("Failed to process trade event", err);
        }
      });

      // Listen for world position
      wsManager.subscribe("world:pos", (payload: any) => {
        try {
          if (payload && payload.id) {
            window.dispatchEvent(new CustomEvent("gu_world_pos", { detail: payload }));
          }
        } catch (err) {
          console.error("Failed to process world position event", err);
        }
      });

      // Listen for world cell
      wsManager.subscribe("world:cell", (payload: any) => {
        try {
          if (payload) {
            window.dispatchEvent(new CustomEvent("gu_world_cell", { detail: payload }));
          }
        } catch (err) {
          console.error("Failed to process world cell event", err);
        }
      });

      // Listen for chat:world
      wsManager.subscribe("chat:world", (payload: any) => {
        try {
          // Handle append-only updates (single new message)
          if (payload?.message && !Array.isArray(payload?.messages)) {
            set((state) => {
              const existing = state.chatMessages.world || [];
              const newMsg = payload.message;
              // Avoid duplicates
              if (existing.find((m: any) => m.id === newMsg.id)) {
                return {};
              }
              const merged = [...existing, newMsg];
              const active = state.activeChannel || "world";
              const open = state.chatOpen;
              const isCurrent = active === "world" && open;
              const newUnread = { ...state.unreadChannels };
              if (!isCurrent) {
                newUnread.world = true;
              }
              return {
                chatMessages: { ...state.chatMessages, world: merged },
                unreadChannels: newUnread,
                baoHoangMessages: active === "world" && open ? merged : state.baoHoangMessages,
              };
            });
            return;
          }

          // Handle full history (initial load or fallback)
          const messages = Array.isArray(payload?.messages) ? payload.messages : (Array.isArray(payload) ? payload : []);
          set((state) => {
            const active = state.activeChannel || "world";
            const open = state.chatOpen;
            const isCurrent = active === "world" && open;
            const newUnread = { ...state.unreadChannels };
            if (!isCurrent) {
              newUnread.world = true;
            }
            return {
              chatMessages: { ...state.chatMessages, world: messages },
              unreadChannels: newUnread,
              baoHoangMessages: active === "world" ? messages : state.baoHoangMessages,
            };
          });
        } catch (err) {
          console.error("Failed to process world chat event", err);
        }
      });

      // Listen for chat:general
      wsManager.subscribe("chat:general", (payload: any) => {
        try {
          // Handle append-only updates (single new message)
          if (payload?.message && !Array.isArray(payload?.messages)) {
            set((state) => {
              const existing = state.chatMessages.general || [];
              const newMsg = payload.message;
              if (existing.find((m: any) => m.id === newMsg.id)) {
                return {};
              }
              const merged = [...existing, newMsg];
              const active = state.activeChannel || "world";
              const open = state.chatOpen;
              const isCurrent = active === "general" && open;
              const newUnread = { ...state.unreadChannels };
              if (!isCurrent) {
                newUnread.general = true;
              }
              return {
                chatMessages: { ...state.chatMessages, general: merged },
                unreadChannels: newUnread,
                baoHoangMessages: active === "general" && open ? merged : state.baoHoangMessages,
              };
            });
            return;
          }

          // Handle full history (initial load or fallback)
          const messages = Array.isArray(payload?.messages) ? payload.messages : (Array.isArray(payload) ? payload : []);
          set((state) => {
            const active = state.activeChannel || "world";
            const open = state.chatOpen;
            const isCurrent = active === "general" && open;
            const newUnread = { ...state.unreadChannels };
            if (!isCurrent) {
              newUnread.general = true;
            }
            return {
              chatMessages: { ...state.chatMessages, general: messages },
              unreadChannels: newUnread,
              baoHoangMessages: active === "general" ? messages : state.baoHoangMessages,
            };
          });
        } catch (err) {
          console.error("Failed to process general chat event", err);
        }
      });

      // Listen for chat:sect
      wsManager.subscribe("chat:sect", (payload: any) => {
        try {
          // Handle append-only updates (single new message)
          if (payload?.message && !Array.isArray(payload?.messages)) {
            set((state) => {
              const existing = state.chatMessages.sect || [];
              const newMsg = payload.message;
              if (existing.find((m: any) => m.id === newMsg.id)) {
                return {};
              }
              const merged = [...existing, newMsg];
              const active = state.activeChannel || "world";
              const open = state.chatOpen;
              const isCurrent = active === "sect" && open;
              const newUnread = { ...state.unreadChannels };
              if (!isCurrent) {
                newUnread.sect = true;
              }
              return {
                chatMessages: { ...state.chatMessages, sect: merged },
                unreadChannels: newUnread,
                baoHoangMessages: active === "sect" && open ? merged : state.baoHoangMessages,
              };
            });
            return;
          }

          // Handle full history (initial load or fallback)
          const messages = Array.isArray(payload?.messages) ? payload.messages : (Array.isArray(payload) ? payload : []);
          set((state) => {
            const active = state.activeChannel || "world";
            const open = state.chatOpen;
            const isCurrent = active === "sect" && open;
            const newUnread = { ...state.unreadChannels };
            if (!isCurrent) {
              newUnread.sect = true;
            }
            return {
              chatMessages: { ...state.chatMessages, sect: messages },
              unreadChannels: newUnread,
              baoHoangMessages: active === "sect" ? messages : state.baoHoangMessages,
            };
          });
        } catch (err) {
          console.error("Failed to process sect chat event", err);
        }
      });

      // Listen for notification
      wsManager.subscribe("notification", (payload: any) => {
        try {
          if (!payload?.id) return;
          set((state) => {
            const exists = state.notifications.some((n) => n.id === payload.id);
            if (exists) return {};
            return {
              notifications: [payload, ...state.notifications].slice(0, 50),
              unreadNotifications: state.unreadNotifications + 1,
            };
          });
        } catch (err) {
          console.error("Failed to process notification event", err);
        }
      });

      // Store cleanup functions in a global variable
      (window as any).__gu_ws_cleanup = () => {
        unsubscribePlayer();
        unsubscribeTrade();
        releaseConnection();
      };

    } catch (wsErr) {
      console.warn("[WebSocket] Exception starting WS:", wsErr);
      releaseConnection();
    }
  },

  unsubscribeEvents: () => {
    try {
      const cleanup = (window as any).__gu_ws_cleanup;
      if (cleanup) {
        cleanup();
        (window as any).__gu_ws_cleanup = null;
      } else {
        wsManager.disconnect();
      }
    } catch {
      // ignore
    }
    set({ playerEventsConnected: false });
  },

  subscribePlayerEvents: () => {
    get().subscribeEvents();
  },

  unsubscribePlayerEvents: () => {
    get().unsubscribeEvents();
  },

  subscribeTradeEvents: () => {
    get().subscribeEvents();
  },

  unsubscribeTradeEvents: () => {
    get().unsubscribeEvents();
  },

  setActiveChannel: (channel) => {
    set((state) => {
      const newUnread = { ...state.unreadChannels };
      newUnread[channel] = false;
      return {
        activeChannel: channel,
        unreadChannels: newUnread,
        baoHoangMessages: state.chatMessages[channel] || [],
      };
    });
  },

  setChatOpen: (open) => {
    set((state) => {
      const active = state.activeChannel || "world";
      const newUnread = { ...state.unreadChannels };
      if (open) {
        newUnread[active] = false;
      }
      return { chatOpen: open, unreadChannels: newUnread };
    });
  },

  subscribeBaoHoangMessages: (channel) => {
    get().subscribeEvents();
    get().setActiveChannel(channel as any);
  },

  unsubscribeBaoHoangMessages: () => {
    // Handled at main connection level
  },

  subscribeAllBaoHoangMessages: () => {
    get().subscribeEvents();
  },

  unsubscribeAllBaoHoangMessages: () => {
    // Handled at main connection level
  },

  fetchReferrals: async () => {
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/referrals`);
      if (response.ok) {
        const data = await response.json();
        set({ referralsData: data });
      }
    } catch (err) {
      console.error("Failed to fetch referrals", err);
    }
  },

  claimReferralReward: async (refereeId: string, milestone: number) => {
    set({ actionLoading: true });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/referrals/claim`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ referee_id: refereeId, milestone }),
      });
      const data = await response.json();
      set({ actionLoading: false });
      if (response.ok) {
        get().addLog(`[Hệ thống] ${data.message}`);
        // Reload referrals to update status
        await get().fetchReferrals();
        // Also refresh player stats (like primeval stones)
        await get().fetchPlayerProfile(get().player?.id || "");
        return { success: true, message: data.message };
      }
      return { success: false, message: data.error };
    } catch (err: any) {
      console.error("Failed to claim referral reward", err);
      set({ actionLoading: false });
      return { success: false, message: "Lỗi kết nối máy chủ." };
    }
  },

  bindReferralCode: async (refCode: string) => {
    set({ actionLoading: true });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/referrals/bind`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ref_code: refCode }),
      });
      const data = await response.json();
      set({ actionLoading: false });
      if (response.ok) {
        get().addLog(`[Hệ thống] ${data.message}`);
        await get().fetchReferrals();
        return { success: true, message: data.message };
      }
      return { success: false, message: data.error };
    } catch (err: any) {
      console.error("Failed to bind referral code", err);
      set({ actionLoading: false });
      return { success: false, message: "Lỗi kết nối máy chủ." };
    }
  },

  fetchPassiveStats: async () => {
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/passive-stats`);
      if (response.ok) {
        const data = await response.json();
        set({ passiveStats: data });
      }
    } catch (err) {
      console.error("Failed to fetch passive stats", err);
    }
  },

  simulateDeposit: async (stones) => {
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/simulate-deposit`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ stones }),
      });
      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || "Giao dịch giả lập thất bại.");
      }
      const data = await response.json();
      await get().fetchPlayerStatus(get().player?.id || "");
      void get().fetchReferrals();
      get().addLog(`[Giả Lập Nạp] Đã nạp thành công $10 (tặng ${stones} Nguyên Thạch)!`);
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Giao dịch giả lập thất bại.", "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  // --- Sect Mines ---

  fetchSectMines: async () => {
    const { token } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/mines`, { method: "GET" });
      if (!response.ok) throw new Error("Failed to fetch sect mines");
      const data = await response.json();
      set({ sectMines: data || [] });
    } catch (err) {
      console.error("Failed to fetch sect mines", err);
    }
  },

  attackSectMine: async (slotId: number, attackerId?: string) => {
    const { token, actionLoading, logout, addToast, addLog } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/mines/attack`, {
        method: "POST",
        body: JSON.stringify({ mine_id: slotId, attacker_id: attackerId || "" }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Lỗi khi tấn công Lãnh địa");
      addToast(data.message || "Đã tấn công Lãnh địa Tông môn!", data.won ? "success" : "info");
      addLog(data.message || "Tấn công Lãnh địa Tông môn.");
      get().fetchPlayerProfile(
        get().player?.username || ""
      ); // To refresh combat state
      get().fetchSectMines();
      return true;
    } catch (err: any) {
      addToast(err.message || "Lỗi khi tấn công Lãnh địa", "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  assignSectMineDefender: async (mineId, defenderType, playerId, monsterId, monsterRank) => {
    const { token, actionLoading, logout, addToast, addLog } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/mines/defend`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          mine_id: mineId,
          defender_type: defenderType,
          player_id: playerId || "",
          monster_id: monsterId || "",
          monster_rank: monsterRank || 1,
        }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Bố trí phòng thủ thất bại.");
      addToast(data.message || "Đã bố trí phòng thủ!", "success");
      addLog(data.message || "Bố trí phòng thủ lãnh địa.");
      get().fetchSectMines();
      get().fetchMySectStatus();
      return true;
    } catch (err: any) {
      addToast(err.message || "Lỗi bố trí phòng thủ", "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  removeSectMineDefender: async (defenderId: string) => {
    const { token, actionLoading, logout, addToast, addLog } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/mines/undefend`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ defender_id: defenderId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Rút phòng thủ thất bại.");
      addToast(data.message || "Đã rút phòng thủ!", "success");
      addLog(data.message || "Rút phòng thủ lãnh địa.");
      get().fetchSectMines();
      get().fetchMySectStatus();
      return true;
    } catch (err: any) {
      addToast(err.message || "Lỗi rút phòng thủ", "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  addLog: (message) => {
    const formatted = translateIDs(message);
    set((state) => ({
      logs: [formatted, ...state.logs].slice(0, 50),
    }));
  },

  initializeSession: async () => {
    if (typeof window !== "undefined") {
      const urlParams = new URLSearchParams(window.location.search);
      const tokenFromUrl = urlParams.get("token");
      if (tokenFromUrl) {
        localStorage.setItem("gu_auth_token", tokenFromUrl);
        const cleanUrl = window.location.pathname;
        window.history.replaceState({}, document.title, cleanUrl);
      }
      const errorFromUrl = urlParams.get("error");
      if (errorFromUrl) {
        set({ error: errorFromUrl });
        get().addToast(errorFromUrl, "error");
        const cleanUrl = window.location.pathname;
        window.history.replaceState({}, document.title, cleanUrl);
      }
    }

    try {
      const response = await authFetch(`${BACKEND_URL}/api/auth/me`);
      if (response.status === 401) {
        set({ player: null, manaSnapshot: null, token: null });
        return;
      }
      if (!response.ok) {
        throw new Error("Failed to restore session");
      }

      const player: Player = await response.json();
      const localToken = (typeof window !== "undefined" ? localStorage.getItem("gu_auth_token") : null) || AUTHENTICATED_SESSION;
      set({ activeCombatReplay: null });
      setPlayerFromServer(player, { token: localToken });
      if (typeof window !== "undefined") {
        localStorage.setItem("gu_player_id", player.id);
      }
      void get().fetchMySectStatus();
      void get().fetchGuInventory();
      try {
        get().subscribePlayerEvents();
        get().subscribeTradeEvents();
      } catch (wsErr) {
        console.warn("[WebSocket] Failed to subscribe events during session restore:", wsErr);
      }
      const today = new Date().toISOString().split('T')[0];
      const hiddenUntil = typeof window !== "undefined" ? localStorage.getItem("gu_changelog_hide_until") : null;
      if (hiddenUntil !== today) {
        useUIStore.getState().setShowChangelog(true);
      }
    } catch (err: any) {
      console.error("Session restore failed:", err.message);
      set({ player: null, manaSnapshot: null, token: null });
    }
  },

  fetchPlayerStatus: async (id) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return;

    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/status/${id}`);

      if (response.status === 401) {
        logout();
        return;
      }

      if (!response.ok) {
        throw new Error("Failed to fetch status");
      }

      const player: Player = await response.json();
      const currentPlayer = get().player;
      if (currentPlayer && currentPlayer.id === id) {
        // The status endpoint only projects equipped/vital Gus. Preserve the
        // full inventory already in state so it is not clobbered by the sparse
        // snapshot.
        setPlayerFromServer({
          ...player,
          gu_inventory: currentPlayer.gu_inventory?.length ? currentPlayer.gu_inventory : player.gu_inventory,
          equipped_gus: currentPlayer.gu_inventory?.length ? currentPlayer.equipped_gus : player.equipped_gus,
        });
      }
    } catch (err: any) {
      console.error("Status fetch failed:", err.message);
    }
  },

  fetchGuInventory: async () => {
    if (fetchGuInventoryPromise) return fetchGuInventoryPromise;
    const executeFetch = async () => {
      const { token, logout } = get();
      if (!token) return;

      try {
        const response = await authFetch(`${BACKEND_URL}/api/player/inventory/gus`);

        if (response.status === 401) {
          logout();
          return;
        }

        if (!response.ok) {
          throw new Error("Failed to fetch inventory");
        }

        const data: { gu_inventory: GuInstance[]; equipped_gus: string[] } = await response.json();
        const currentPlayer = get().player;
        if (!currentPlayer) return;

        setPlayerFromServer({
          ...currentPlayer,
          gu_inventory: data.gu_inventory,
          equipped_gus: data.equipped_gus,
        });
        set({ guInventoryLoaded: true });
      } catch (err: any) {
        console.error("Gu inventory fetch failed:", err.message);
      } finally {
        fetchGuInventoryPromise = null;
      }
    };
    fetchGuInventoryPromise = executeFetch();
    return fetchGuInventoryPromise;
  },

  fetchPlayerProfile: async (username) => {
    const { token, logout } = get();
    if (!token) return;

    set({ profileLoading: true });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/profile/${encodeURIComponent(username)}`);

      if (response.status === 401) {
        logout();
        return;
      }

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || "Failed to load player profile");
      }

      const data = await response.json();
      set({ inspectedProfile: data, inspectedUsernameQuery: username });
    } catch (err: any) {
      console.error("Profile fetch failed:", err.message);
      get().addToast("Lỗi tải hồ sơ.", "error");
      get().addToast(`Không thể trinh thám hồ sơ: ${err.message}`, "error");
    } finally {
      set({ profileLoading: false });
    }
  },

  clearInspectedProfile: () => set({ inspectedProfile: null, inspectedUsernameQuery: null }),

  updateProfileSettings: async (settings) => {
    const { token, actionLoading, addToast } = get();
    if (!token || actionLoading) return false;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/profile/settings`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(settings),
      });

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Failed to update profile settings");
      }

      setPlayerFromServer(data.player);
      addToast(data.message || "Cập nhật hồ sơ thành công!", "success");
      return true;
    } catch (err: any) {
      console.error("Profile settings update failed:", err.message);
      get().addToast("Lỗi cập nhật hồ sơ.", "error");
      set({ error: err.message });
      addToast(`Cập nhật thất bại: ${err.message}`, "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  uploadAvatar: async (file: File) => {
    const { token, actionLoading, addToast } = get();
    if (!token || actionLoading) return null;

    set({ actionLoading: true, error: null });
    try {
      const formData = new FormData();
      formData.append("avatar", file);

      const response = await authFetch(`${BACKEND_URL}/api/player/avatar`, {
        method: "POST",
        body: formData,
      });

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Upload thất bại");
      }

      // Update player avatar_url in store
      const { player } = get();
      if (player) {
        set({ player: { ...player, avatar_url: data.avatar_url } });
      }
      addToast(data.message || "Cập nhật ảnh đại diện thành công!", "success");
      return data.avatar_url as string;
    } catch (err: any) {
      console.error("Avatar upload failed:", err.message);
      get().addToast("Lỗi tải ảnh đại diện.", "error");
      set({ error: err.message });
      addToast(`Upload thất bại: ${err.message}`, "error");
      return null;
    } finally {
      set({ actionLoading: false });
    }
  },

  explore: async () => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/explore/${player.id}`, {
        method: "POST",
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || "Khám phá thất bại.");
      }

      const data: ExploreResponse = await response.json();
      setPlayerFromServer(data.player);
      get().addLog(data.log);
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Khám phá thất bại: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      set({ actionLoading: false });
    }
  },

  feedGu: async (guInstanceId) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/inventory/feed`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ gu_instance_id: guInstanceId }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || "Không thể cho ăn.");
      }

      const data: FeedResponse = await response.json();
      setPlayerFromServer(data.player);
      get().addLog(data.message);
      get().addToast(data.message, "success");
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Cho Cổ ăn thất bại: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      set({ actionLoading: false });
    }
  },

  feedGuGroup: async (guId, rank) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/inventory/feed-group`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ gu_id: guId, ...(rank ? { rank } : {}) }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || "Không thể cho ăn hàng loạt.");
      }

      const data: FeedResponse = await response.json();
      setPlayerFromServer(data.player);
      get().addLog(data.message);
      get().addToast(data.message, "success");
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Cho ăn hàng loạt thất bại: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      set({ actionLoading: false });
    }
  },

  feedAllGu: async () => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/inventory/feed-all`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      if (!response.ok) {
        const errData = await response.json();
        throw new Error(errData.error || "Không thể cho tất cả Cổ trùng ăn.");
      }

      const data: FeedResponse = await response.json();
      setPlayerFromServer(data.player);
      get().addLog(data.message);
      get().addToast(data.message, "success");
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Cho tất cả Cổ trùng ăn thất bại: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      set({ actionLoading: false });
    }
  },

  fetchDictionary: async () => {
    await fetchWithDedupe("fetchDictionary", async () => {
      try {
        const response = await authFetch(`${BACKEND_URL}/api/dictionary`);
        if (!response.ok) {
          throw new Error("Failed to load global metadata");
        }
        const data = await response.json();
        const guArray = data.gu || [];
        const guMap: Record<string, GuSpec> = {};
        const guByRank: Record<string, Record<number, GuSpec>> = {};
        if (Array.isArray(guArray)) {
          for (const item of guArray) {
            if (!guByRank[item.id]) guByRank[item.id] = {};
            guByRank[item.id][item.rank] = item;
            const existing = guMap[item.id];
            if (!existing || item.rank > existing.rank) {
              guMap[item.id] = item;
            }
          }
        }
        const constitutionDict: Record<string, EffectDictionaryEntry> = {};
        const constitutionArr = Array.isArray(data.constitutions) ? data.constitutions : [];
        for (const c of constitutionArr) {
          constitutionDict[c.key] = c;
        }
        const raceDict: Record<string, EffectDictionaryEntry> = {};
        const raceArr = Array.isArray(data.races) ? data.races : [];
        for (const r of raceArr) {
          raceDict[r.key] = r;
        }
        set({
          guDictionary: guMap,
          guByRank,
          itemDictionary: data.items || {},
          monsterDictionary: data.monsters || {},
          constitutionDictionary: constitutionDict,
          raceDictionary: raceDict,
          dictionaryLoaded: true,
        });
      } catch (err: any) {
        console.error("Dictionary fetch failed:", err.message);
        set({ dictionaryLoaded: true });
      }
    });
  },

  fetchRecipes: async () => {
    if (get().recipesLoaded) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/recipes`);
      if (response.status === 401) {
        set({ recipes: [], recipesLoaded: true });
        return;
      }
      if (!response.ok) {
        throw new Error("Failed to load recipes");
      }
      const data = await response.json();
      set({ recipes: data || [], recipesLoaded: true });
    } catch (err: any) {
      console.error("Recipes sync error:", err.message);
      set({ recipesLoaded: true });
    }
  },

  fetchQuests: async () => {
    try {
      const response = await authFetch(`${BACKEND_URL}/api/quest`);
      if (response.status === 401) {
        set({ quests: [] });
        return;
      }
      if (!response.ok) {
        throw new Error("Failed to load quests");
      }
      const data = await response.json();
      set({ quests: data.quests || [] });
    } catch (err: any) {
      console.error("Quests sync error:", err.message);
    }
  },

  fetchRegions: async () => {
    try {
      const response = await authFetch(`${BACKEND_URL}/api/explore/regions`);
      if (response.status === 401) {
        set({ regions: [], regionsLoaded: true });
        return;
      }
      if (!response.ok) {
        throw new Error("Failed to load regions");
      }
      const data = await response.json();
      set({ regions: data || [], regionsLoaded: true });
    } catch (err: any) {
      console.error("Regions sync error:", err.message);
      set({ regionsLoaded: true });
    }
  },

  fetchWorld: async () => {
    try {
      const response = await authFetch(`${BACKEND_URL}/api/explore/world`);
      if (!response.ok) {
        throw new Error("Failed to load world config");
      }
      const data = await response.json();
      set({ world: data || null });
      return data || null;
    } catch (err: any) {
      console.error("World config sync error:", err.message);
      return null;
    }
  },

  refineGu: async (recipeId, selectedInstanceIds) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/refine`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          recipe_id: recipeId,
          selected_gu_instance_ids: selectedInstanceIds,
        }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Hợp luyện thất bại.");
      }

      setPlayerFromServer(data.player);
      get().addLog(data.message);
      get().addToast(data.message, data.status === "success" ? "success" : "error");
      return data;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Hợp luyện thất bại: ${err.message}`);
      get().addToast(err.message, "error");
      return { status: "failed", message: err.message };
    } finally {
      set({ actionLoading: false });
    }
  },

  deduceRecipe: async (recipeId) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/refine/deduce`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          recipe_id: recipeId,
        }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Thôi diễn thất bại.");
      }

      setPlayerFromServer(data.player);
      // update completion rate in local recipes state
      set((state) => ({
        recipes: state.recipes.map((r) =>
          r.id === recipeId ? { ...r, completion_rate: data.completion_rate } : r
        ),
      }));

      get().addLog(data.message);
      get().addToast(data.message, "success");
      return { success: true, message: data.message, completion_rate: data.completion_rate };
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Thôi diễn thất bại: ${err.message}`);
      get().addToast(err.message, "error");
      return { success: false, message: err.message, completion_rate: 0 };
    } finally {
      set({ actionLoading: false });
    }
  },

  breakWall: async (manaSpend) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    // Client-side validation to prevent unnecessary requests
    if (manaSpend < 0 || manaSpend > Math.floor(player.current_mana)) {
      set({ error: "Số lượng Chân Nguyên không hợp lệ." });
      return null;
    }

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/cultivate/break-wall`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ mana_spend: manaSpend}),
      });

      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Trùng kích thất bại."); }

      setPlayerFromServer(data.player);
      get().addLog(data.message || "Trùng kích hoàn tất.");
      return { success: data.success, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Trùng kích lỗi: ${err.message}`);
      return { success: false, message: err.message };
    } finally { set({ actionLoading: false }); }
  },

  rechargeWithStones: async (stones) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/recharge`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ player_id: player.id, stones: stones }),
      });

      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Nạp nguyên thạch thất bại.");

      setPlayerFromServer(data.player);
      get().addLog(data.message || "Nạp nguyên thạch thành công.");
      return { success: true, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Nạp nguyên thạch lỗi: ${err.message}`);
      return { success: false, message: err.message };
    } finally {
      set({ actionLoading: false });
    }
  },

  useRelicGu: async (relicKey) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/cultivate/use-relic-gu`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ relic_key: relicKey }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Không thể dùng Cổ Xá Lợi."); }
      setPlayerFromServer(data.player);
      get().addLog(data.message || "Đã sử dụng Cổ Xá Lợi.");
      return { success: true, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Dùng Cổ Xá Lợi lỗi: ${err.message}`);
      return { success: false, message: err.message };
    } finally { set({ actionLoading: false }); }
  },

  ascension: async () => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/cultivate/ascension`, {
        method: "POST",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Thăng tiên thất bại."); }
      setPlayerFromServer(data.player);
      get().addLog(data.message || "Thăng tiên hoàn tất.");
      return { success: data.success, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Thăng tiên lỗi: ${err.message}`);
      return { success: false, message: err.message };
    } finally { set({ actionLoading: false }); }
  },

  completeQuest: async (questId) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/quest/complete`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ quest_id: questId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Không thể nhận thưởng nhiệm vụ."); }
      setPlayerFromServer(data.player);
      get().addToast(data.message || "Hoàn thành nhiệm vụ.", "success");
      return { success: true, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(`Nhận thưởng nhiệm vụ lỗi: ${err.message}`, "error");
      return { success: false, message: err.message };
    } finally { set({ actionLoading: false }); }
  },

  redeemGiftCode: async (code) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/redeem-gift-code`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ code }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Mã quà không hợp lệ."); }
      setPlayerFromServer(data.player);
      get().addLog(data.message || "Đổi mã quà thành công!");
      return { success: true, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Đổi mã quà lỗi: ${err.message}`);
      return { success: false, message: err.message };
    } finally { set({ actionLoading: false }); }
  },

  fetchAchievements: async () => {
    const { token } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/achievements`);
      if (response.status === 401) return;
      if (response.ok) {
        const data = await response.json();
        set({ achievements: data.achievements || [], earnedAchievements: data.earned || [], selectedTitle: data.selected_title || '', selectedFrame: data.selected_frame || '' });
      }
    } catch (err: any) {
      console.error("Achievements fetch error:", err.message);
      get().addToast("Lỗi tải thành tích.", "error");
    }
  },
  claimAchievement: async (achievementId) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/achievements/claim`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ achievement_id: achievementId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Nhận thưởng thất bại."); }
      setPlayerFromServer(data.player);
      get().fetchAchievements();
      get().addLog(data.message || "Nhận thưởng thành tựu thành công!");
      return { success: true, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Nhận thưởng thành tựu lỗi: ${err.message}`);
      return { success: false, message: err.message };
    } finally { set({ actionLoading: false }); }
  },
  selectTitleFrame: async (title, frame) => {
    const { token } = get();
    if (!token) return null;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/achievements/select`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ title, frame }),
      });
      if (response.status === 401) return null;
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Cập nhật thất bại.");
      set({ selectedTitle: title, selectedFrame: frame });
      get().addLog(data.message || "Đã cập nhật danh hiệu và frame.");
      return { success: true, message: data.message };
    } catch (err: any) {
      console.error("Select title/frame error:", err.message);
      get().addToast("Lỗi chọn danh hiệu.", "error");
      return { success: false, message: err.message };
    }
  },

  fetchGachaPools: async () => {
    const { token } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/gacha/pools`);
      if (response.status === 401) return;
      if (response.ok) {
        const data = await response.json();
        set({ gachaPools: data.pools || [] });
      }
    } catch (err: any) {
      console.error("Gacha pools error:", err.message);
      get().addToast("Lỗi tải bách khoa.", "error");
    }
  },

  openGacha: async (poolId, quantity = 1) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/gacha/open`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ pool_id: poolId, quantity }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Mở Đá  thất bại."); }
      setPlayerFromServer(data.player);
      get().addLog(data.message || "Mở Đá  thành công!");
      return { success: true, message: data.message, reward: data.reward, rewards: data.rewards };
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Mở Đá  lỗi: ${err.message}`);
      return { success: false, message: err.message };
    } finally { set({ actionLoading: false }); }
  },

  fetchCurrentBettingRound: async () => {
    const { token } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/beast-betting/current`);
      if (response.status === 401) return;
      if (response.ok) {
        const data = await response.json();
        set({
          currentBettingRound: data.round,
          myBetTeam: data.my_bet_team || null,
          myBetAmount: data.my_bet_amount || null,
        });
      }
    } catch (err: any) {
      console.error("Fetch betting round error:", err.message);
    }
  },

  fetchBettingHistory: async () => {
    const { token } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/beast-betting/history`);
      if (response.status === 401) return;
      if (response.ok) {
        const data = await response.json();
        set({ bettingHistory: data.history || [] });
      }
    } catch (err: any) {
      console.error("Fetch betting history error:", err.message);
    }
  },

  placeBeastBet: async (team, amount) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/beast-betting/bet`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ team, amount }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Đặt cược thất bại."); }

      // Update player stones locally
      if (player.primeval_stones !== undefined) {
        set({
          player: {
            ...player,
            primeval_stones: player.primeval_stones - amount
          }
        });
      }

      get().addLog(data.message || "Đặt cược thành công!");
      get().addToast(data.message || "Đặt cược thành công!", "success");

      // Refresh current betting round
      await get().fetchCurrentBettingRound();

      return { success: true, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Đặt cược lỗi: ${err.message}`);
      return { success: false, message: err.message };
    } finally { set({ actionLoading: false }); }
  },

  triggerTribulation: async () => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/cultivate/tribulation`, {
        method: "POST",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Không thể kích hoạt thiên kiếp."); }
      setPlayerFromServer(data.player);
      get().addLog(data.message || "Thiên kiếp giáng lâm.");
      return { success: true, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Lỗi độ kiếp: ${err.message}`);
      return { success: false, message: err.message };
    } finally { set({ actionLoading: false }); }
  },

  exploreNode: async (nodeId) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/explore`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          player_id: player.id,
          node_id: nodeId,
        }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Khám phá thất bại.");
      }

      setPlayerFromServer(data.player);
      get().addLog(data.log);
      
      // Handle skip battle result from explore
      if (data.skip_result) {
        const skipResult = data.skip_result;
        if (skipResult.winner === "Phe Ta") {
          get().addToast(`Tự động chiến thắng ${skipResult.enemy_name || "kẻ thù"}!`, "success");
          if (skipResult.mine_occupied) {
            get().addToast(`Đã chiếm mỏ ${skipResult.mine_occupied}!`, "info");
          }
        } else {
          get().addToast(`Tự động chiến đấu thất bại! ${skipResult.enemy_name || "Kẻ thù"} quá mạnh.`, "error");
        }
      }
      
      return data;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Khám phá thất bại: ${err.message}`);
      get().addToast(err.message, "error");
      return null;
    } finally {
      set({ actionLoading: false });
    }
  },

  clearStress: async () => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/clear-stress`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ player_id: player.id }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Tịnh hóa Không khiếu thất bại.");

      setPlayerFromServer(data.player);
      get().addLog(data.message || "Tịnh hóa Không khiếu hoàn tất.");
      return { success: true, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Tịnh hóa lỗi: ${err.message}`);
      return { success: false, message: err.message };
    } finally {
      set({ actionLoading: false });
    }
  },

  sacrificeNPC: async (npcId: string) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/sacrifice`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ npc_id: npcId }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hiến tế thất bại.");

      const updatedPlayer = { ...get().player };
      if (updatedPlayer) {
        updatedPlayer.aptitude_percent = data.new_aptitude;
        updatedPlayer.aptitude_type = data.aptitude_type;
        updatedPlayer.aperture_stress = data.aperture_stress;
        set({ player: updatedPlayer as any });
      }

      get().addLog(data.message || "Hiến tế hoàn tất.");
      return { success: true, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Hiến tế lỗi: ${err.message}`);
      return { success: false, message: err.message };
    } finally {
      set({ actionLoading: false });
    }
  },

  fetchKillerMoves: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/killer-moves`);
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Không thể tải sát chiêu.");
      set({ killerMoves: data.killer_moves || [], killerMovesLoaded: true });
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Không thể tải sát chiêu.";
      set({ error: message, killerMovesLoaded: true });
      get().addLog(`Tải sát chiêu thất bại: ${message}`);
    }
  },

  fetchCombatSettings: async () => {
    const { token, logout } = get();
    if (!token) return;
    set({ combatSettingsLoading: true });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/combat-settings`);
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Không thể tải cài đặt chiến đấu.");
      set({ combatSettings: data.combat_settings || null, combatSettingsLoading: false });
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Không thể tải cài đặt chiến đấu.";
      set({ error: message, combatSettingsLoading: false });
    }
  },

  saveCombatSettings: async (settings) => {
    const { token, logout } = get();
    if (!token) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/combat-settings`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ combat_settings: settings }),
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Lưu cài đặt thất bại.");
      set({ combatSettings: data.combat_settings || settings, actionLoading: false });
      get().addLog(data.message || "Đã lưu cài đặt chiến đấu.");
      return true;
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Lưu cài đặt thất bại.";
      set({ error: message, actionLoading: false });
      get().addLog(`Lưu cài đặt thất bại: ${message}`);
      return false;
    }
  },

  deleteKillerMove: async (id) => {
    const { token, logout } = get();
    if (!token) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/killer-moves/${id}`, { method: "DELETE" });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Không thể xóa sát chiêu.");
      set((state) => ({ killerMoves: state.killerMoves.filter((move) => move.id !== id) }));
      get().addLog(data.message || "Đã xóa sát chiêu.");
      return true;
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Không thể xóa sát chiêu.";
      set({ error: message });
      get().addLog(`Xóa sát chiêu thất bại: ${message}`);
      return false;
    }
  },

  castKillerMove: async (id, targetId, usedExtraGuThisTurn = false) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/combat/cast-killer-move`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          killer_move_id: id,
          target_id: targetId || "",
          used_extra_gu_this_turn: usedExtraGuThisTurn,
        }),
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Thi triển sát chiêu thất bại.");
      const updatedPlayer = data.player
        ? data.player
        : {
            ...player,
            active_combat: data.active_combat ?? player.active_combat,
            current_mana: typeof data.current_mana === "number" ? data.current_mana : player.current_mana,
            hp: typeof data.hp === "number" ? data.hp : player.hp,
          };
      setPlayerFromServer(updatedPlayer);
      if (data.logs && Array.isArray(data.logs)) {
        data.logs.forEach((logLine: string) => get().addLog(logLine));
      }
      if (data.compound_effects && Array.isArray(data.compound_effects)) {
        data.compound_effects.forEach((ce: { name: string; description: string }) => {
          get().addLog(`⟐ [${ce.name}] ${ce.description}`);
        });
      }
      return data;
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Thi triển sát chiêu thất bại.";
      set({ error: message });
      get().addLog(`Thi triển sát chiêu thất bại: ${message}`);
      return null;
    } finally {
      set({ actionLoading: false });
    }
  },

  executeCombatTurn: async (equippedGuIds, isCharge = false, targetId, targetX, targetY) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/combat/attack`, {
        method: "POST",
        // Combat resolution can include party, pet, loot and persistence work.
        // Do not cancel a valid turn at the generic 30-second API timeout.
        timeoutMs: 120000,
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          player_id: player.id,
          equipped_gu_ids: equippedGuIds,
          is_charge: isCharge,
          target_id: targetId || "",
          target_x: targetX || 0,
          target_y: targetY || 0,
        }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Tấn công thất bại.");
      }

      setPlayerFromServer(data.player);
      if (data.logs && Array.isArray(data.logs)) {
        data.logs.forEach((logLine: string) => {
          get().addLog(logLine);
        });
      }
      return data;
    } catch (err: any) {
      const message = err?.name === "AbortError"
        ? "Yêu cầu tấn công quá thời gian. Trạng thái nhân vật sẽ được đồng bộ lại."
        : (err?.message || "Thao tác thất bại.");
      set({ error: message });
      get().addToast(message, "error");
      get().addLog(`Tấn công thất bại: ${message}`);
      if (err?.name === "AbortError") {
        setTimeout(() => void get().fetchPlayerStatus(player.id), 0);
      }
      return null;
    } finally {
      set({ actionLoading: false });
    }
  },

  executeAutoCombat: async (loadout: any) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/combat/start`, {
        method: "POST",
        // Full auto-combat simulations legitimately take longer than ordinary
        // CRUD requests, especially for parties and multi-enemy encounters.
        timeoutMs: 120000,
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ loadout }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        if (response.status === 400 && (data.error?.includes("Không trong trạng thái chiến đấu") || data.error?.includes("Không có trận chiến nào"))) {
          const currentPlayer = get().player;
          if (currentPlayer && currentPlayer.active_combat) {
            set({
              player: { ...currentPlayer, active_combat: null },
              activeCombatReplay: null,
            });
          }
        }
        throw new Error(data.error || "Bắt đầu mô phỏng thất bại.");
      }

      // We just return the logs and winner, we don't need to update player immediately
      // The frontend handles the replay animation
      return data;
    } catch (err: any) {
      const message = err?.name === "AbortError"
        ? "Mô phỏng chiến đấu quá thời gian. Trạng thái nhân vật sẽ được đồng bộ lại."
        : (err?.message || "Thao tác thất bại.");
      set({ error: message });
      get().addToast(message, "error");
      get().addLog(`Mô phỏng thất bại: ${message}`);
      if (err?.name === "AbortError") {
        setTimeout(() => void get().fetchPlayerStatus(player.id), 0);
      }
      return null;
    } finally {
      set({ actionLoading: false });
    }
  },

  executePracticeCombat: async (loadout: any, dummyConfig: any) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/combat/practice`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          loadout,
          dummy_hp: parseInt(dummyConfig.dummyHP as any) || 100,
          dummy_shield: parseInt(dummyConfig.dummyShield as any) || 0,
          dummy_atk: parseInt(dummyConfig.dummyAtk as any) || 10,
          dummy_def: parseInt(dummyConfig.dummyDef as any) || 5,
          dummy_speed: parseFloat(dummyConfig.dummySpeed as any) || 1.0
        }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Bắt đầu mô phỏng thất bại.");
      }

      return data;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Mô phỏng thất bại: ${err.message}`);
      return null;
    } finally {
      set({ actionLoading: false });
    }
  },

  activeCombatReplay: null,
  setActiveCombatReplay: (replay) => set({ activeCombatReplay: replay }),

  retreat: async () => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/explore/retreat/${player.id}`, {
        method: "POST",
        timeoutMs: 30000,
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Rút lui thất bại.");
      }

      set({ activeCombatReplay: null });
      setPlayerFromServer(data.player);
      useUIStore.getState().setView("dashboard");
      get().addLog(data.log || "Đạo hữu đã rút lui an toàn.");
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Rút lui thất bại: ${err.message}`);
    } finally {
      set({ actionLoading: false });
    }
  },

  fleeCombat: async () => {
    const { player, token, actionLoading, logout, showConfirm } = get();
    if (!player || !token || actionLoading) return;

    // HP quá thấp (< 20% max): bỏ chạy trừ 20% Max HP — gần như chắc chắn ngất
    // xỉu, mất toàn bộ tài nguyên thu thập. Cảnh báo và chặn lại cho đạo hữu cân nhắc.
    // Mine và PvP được bỏ chạy tự do nên bỏ qua cảnh báo này.
    const maxHP = player.base_hp || player.hp || 1;
    const isMine = !!player.active_combat?.mine_slot_id;
    const isPVP = player.active_combat?.enemy_type === "PVP";
    if (!isMine && !isPVP && player.hp <= 0.2 * maxHP) {
      set({ actionLoading: true });
      const proceed = await showConfirm(
        "HP của đạo hữu đang rất thấp. Bỏ chạy khỏi chiến đấu sẽ khiến đạo hữu chịu thêm 20% sát thương theo HP tối đa — gần như chắc chắn sẽ ngất xỉu, làm rơi mất toàn bộ Nguyên Thạch, Cổ trùng và tài nguyên thu thập. Vẫn muốn bỏ chạy?",
        { confirmText: "Vẫn bỏ chạy", cancelText: "Hủy" },
      );
      set({ actionLoading: false });
      if (!proceed) return;
    }

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/explore/flee/${player.id}`, {
        method: "POST",
        timeoutMs: 30000,
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        if (response.status === 400 && data.error?.includes("Không có trận chiến nào")) {
          const currentPlayer = get().player;
          if (currentPlayer && currentPlayer.active_combat) {
            set({
              player: { ...currentPlayer, active_combat: null },
              activeCombatReplay: null,
            });
          } else {
            set({ activeCombatReplay: null });
          }
        }
        throw new Error(data.error || "Bỏ chạy thất bại.");
      }

      if (data.flee_success === false) {
        setPlayerFromServer(data.player);
        if (data.logs && Array.isArray(data.logs)) {
          data.logs.forEach((logLine: string) => {
            get().addLog(logLine);
          });
        } else {
          get().addLog("Rút lui thất bại!");
        }
        if (data.victory) {
          get().addToast("Địch nhân tự chuốc lấy diệt vong! Chiến thắng bất ngờ!", "success");
        } else if (data.defeat) {
          get().addToast("Rút lui thất bại! Đạo hữu bị đánh bại!", "error");
        } else {
          get().addToast("Cố gắng rút lui thất bại! Địch nhân thừa cơ tấn công!", "info");
        }
        return data;
      }

      const combat = player.active_combat;
      let returnView = useUIStore.getState().currentView;
      if (combat) {
        if (combat.combat_mode === "virtual") returnView = "arena";
        else if (combat.mine_slot_id) returnView = "mine";
        else if (combat.tribulation_event_id) returnView = "aperture";
      }
      set({ activeCombatReplay: null });
      setPlayerFromServer(data.player);
      useUIStore.getState().setView(returnView);
      get().addLog(data.log || "Đạo hữu đã bỏ chạy an toàn.");
      return data;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Bỏ chạy thất bại: ${err.message}`);
    } finally {
      set({ actionLoading: false });
    }
  },

  fetchCombatHistory: async () => {
    const { token, logout } = get();
    if (!token) return;

    set({ combatHistoryLoading: true });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/combat/history`, { method: "GET" });
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      if (response.ok) {
        set({ combatHistory: data.entries || [] });
      }
    } catch {
      // Non-critical
    } finally {
      set({ combatHistoryLoading: false });
    }
  },

  fetchCombatHistoryEntry: async (id: string) => {
    const { token, logout } = get();
    if (!token) return null;

    try {
      const response = await authFetch(`${BACKEND_URL}/api/combat/history/${id}`, { method: "GET" });
      if (response.status === 401) { logout(); return null; }
      const data = await response.json();
      if (response.ok) return data;
    } catch {
      return null;
    }
  },

  auditLogs: [],
  auditLogsLoading: false,
  fetchAuditLogs: async (filters) => {
    const { token, logout } = get();
    if (!token) return;

    set({ auditLogsLoading: true });
    try {
      let url = `${BACKEND_URL}/api/player/audit-logs`;
      const queryParams: string[] = [];
      if (filters) {
        if (filters.action) queryParams.push(`action=${encodeURIComponent(filters.action)}`);
        if (filters.entity_type) queryParams.push(`entity_type=${encodeURIComponent(filters.entity_type)}`);
        if (filters.success !== undefined) queryParams.push(`success=${filters.success}`);
        if (filters.limit !== undefined) queryParams.push(`limit=${filters.limit}`);
        if (filters.offset !== undefined) queryParams.push(`offset=${filters.offset}`);
      }
      if (queryParams.length > 0) {
        url += `?${queryParams.join("&")}`;
      }

      const response = await authFetch(url, { method: "GET" });
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      if (response.ok) {
        set({ auditLogs: data || [] });
      }
    } catch {
      // Non-critical
    } finally {
      set({ auditLogsLoading: false });
    }
  },



  fetchMarketplaceListings: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/bao-hoang/listings`, {});
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ marketplaceListings: Array.isArray(data) ? data : [] });
    } catch (err) {
      console.error("market listings failed", err);
      get().addToast("Lỗi tải thị trường.", "error");
    }
  },

  createMarketplaceListing: async (payload) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/bao-hoang/listings`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          item_type: payload.itemType,
          gu_instance_id: payload.guInstanceId || "",
          gu_instance_ids: payload.guInstanceIds || [],
          item_id: payload.itemId || "",
          quantity: payload.quantity,
          price: payload.price,
          duration_hours: payload.durationHours,
          currency: payload.currency,
          listing_type: payload.listingType,
          target_buyer_username: payload.targetBuyerUsername || "",
        }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Đăng bán thất bại");
      get().addLog(data.message || "Đăng bán thành công");
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Đăng bán thất bại: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  placeMarketplaceBid: async (listingId, amount) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/bao-hoang/bid`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ listing_id: listingId, amount }) });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Đặt giá thất bại");
      get().addLog(data.message || "Đặt giá thành công");
      await get().initializeSession();
      await get().fetchMarketplaceListings();
      return true;
    } catch (err: any) { set({ error: err.message }); get().addLog(`Đặt giá thất bại: ${err.message}`); get().addToast(err.message || "Đặt giá thất bại.", "error"); return false; } finally { set({ actionLoading: false }); }
  },

  cancelMarketplaceListing: async (listingId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/bao-hoang/cancel`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ listing_id: listingId }) });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hủy listing thất bại");
      get().addLog(data.message || "Đã hủy");
      await get().initializeSession();
      await get().fetchMarketplaceListings();
      return true;
    } catch (err: any) { set({ error: err.message }); get().addLog(`Hủy listing thất bại: ${err.message}`); get().addToast(err.message || "Hủy listing thất bại.", "error"); return false; } finally { set({ actionLoading: false }); }
  },

  buyMarketplaceListing: async (listingId, quantity) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/bao-hoang/buy`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ listing_id: listingId, quantity }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Mua hàng thất bại");
      get().addLog(data.message || "Mua hàng thành công");
      await get().initializeSession();
      await get().fetchMarketplaceListings();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Mua hàng thất bại: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchActiveTrade: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/trade/active`);
      if (response.status === 401) { logout(); return; }
      if (!response.ok) return;
      const data = await response.json();
      set({ activeTrade: data.trade });
    } catch (err) {
      console.error("Failed to fetch active trade", err);
      get().addToast("Lỗi tải giao dịch.", "error");
    }
  },

  initiateTrade: async (receiverUsername) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/trade/initiate`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ receiver_username: receiverUsername }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Gửi yêu cầu giao dịch thất bại");
      get().addLog(data.message || "Yêu cầu giao dịch đã được gửi!");
      get().addToast(data.message || "Yêu cầu giao dịch đã được gửi!", "success");
      await get().fetchActiveTrade();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      get().addLog(`Giao dịch thất bại: ${err.message}`);
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  acceptTrade: async (tradeId) => {
    const { token, logout } = get();
    if (!token) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/trade/accept/${tradeId}`, { method: "POST" });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Chấp nhận giao dịch thất bại");
      get().addLog(data.message || "Đã chấp nhận giao dịch!");
      get().addToast("Đã thiết lập kết nối Thông Thiên Cổ!", "success");
      await get().fetchActiveTrade();
      return true;
    } catch (err: any) {
      get().addToast(err.message, "error");
      return false;
    }
  },

  declineTrade: async (tradeId) => {
    const { token, logout } = get();
    if (!token) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/trade/decline/${tradeId}`, { method: "POST" });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Từ chối giao dịch thất bại");
      get().addLog(data.message || "Đã từ chối giao dịch!");
      set({ activeTrade: null });
      return true;
    } catch (err: any) {
      get().addToast(err.message, "error");
      return false;
    }
  },

  cancelTrade: async (tradeId) => {
    const { token, logout } = get();
    if (!token) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/trade/cancel/${tradeId}`, { method: "POST" });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hủy giao dịch thất bại");
      get().addLog(data.message || "Đã hủy giao dịch!");
      get().addToast("Đã hủy kết nối giao dịch.", "info");
      set({ activeTrade: null });
      return true;
    } catch (err: any) {
      get().addToast(err.message, "error");
      return false;
    }
  },

  updateTradeOffer: async (tradeId, offer) => {
    const { token, logout } = get();
    if (!token) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/trade/offer/${tradeId}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(offer),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Cập nhật ưu đãi thất bại");
      await get().fetchActiveTrade();
      return true;
    } catch (err: any) {
      get().addToast(err.message, "error");
      return false;
    }
  },

  lockTrade: async (tradeId) => {
    const { token, logout } = get();
    if (!token) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/trade/lock/${tradeId}`, { method: "POST" });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Khóa giao dịch thất bại");
      get().addLog("Đã khóa ưu đãi giao dịch!");
      get().addToast("Đã khóa ưu đãi giao dịch!", "success");
      await get().fetchActiveTrade();
      return true;
    } catch (err: any) {
      get().addToast(err.message, "error");
      return false;
    }
  },

  confirmTrade: async (tradeId) => {
    const { token, logout } = get();
    if (!token) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/trade/confirm/${tradeId}`, { method: "POST" });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Xác nhận giao dịch thất bại");
      
      const bothConfirmed = data.sender_confirmed && data.receiver_confirmed;
      if (bothConfirmed) {
        get().addLog("Giao dịch thành công!");
        get().addToast("Giao dịch thành công!", "success");
        set({ activeTrade: null });
        await get().initializeSession();
      } else {
        get().addLog("Đã xác nhận giao dịch. Chờ đối phương xác nhận...");
        get().addToast("Đã xác nhận giao dịch. Chờ đối phương xác nhận...", "info");
        await get().fetchActiveTrade();
      }
      return true;
    } catch (err: any) {
      get().addToast(err.message, "error");
      return false;
    }
  },

  buyShopItem: async (itemId, quantity, rank = 0) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return false;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/buy`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          player_id: player.id,
          item_id: itemId,
          quantity: quantity,
          rank: rank,
        }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Giao dịch thất bại.");
      }

      setPlayerFromServer(data.player);
      get().addLog(data.message || "Mua hàng thành công!");
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Mua hàng thất bại: ${err.message}`);
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },


  sellShopItem: async (itemId, quantity, itemType, rank = 0) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/sell`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ player_id: player.id, item_id: itemId, quantity, item_type: itemType, rank }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Bán hàng thất bại.");
      setPlayerFromServer(data.player);
      get().addLog(data.message || "Bán hàng thành công!");
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Bán hàng thất bại: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  sellShopItemsBulk: async (items) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/sell-bulk`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ player_id: player.id, items }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Bán hàng thất bại.");
      setPlayerFromServer(data.player);
      get().addLog(data.message || "Bán hàng thành công!");
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Bán hàng thất bại: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  dropGu: async (instanceId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/inventory/drop`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ gu_instance_id: instanceId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Vứt Cổ thất bại.");
      setPlayerFromServer(data.player);
      get().addLog(data.message || "Đã vứt Cổ trùng.");
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Vứt Cổ thất bại: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },
  dropGus: async (instanceIds) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/inventory/drop-batch`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ gu_instance_ids: instanceIds }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Vứt Cổ thất bại.");
      setPlayerFromServer(data.player);
      get().addLog(data.message || "Đã vứt Cổ trùng hàng loạt.");
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Vứt Cổ hàng loạt thất bại: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },
  equipGu: async (instanceId, equip) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/inventory/equip`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          instance_id: instanceId,
          equip: equip,
        }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Thao tác trang bị thất bại.");
      }

      setPlayerFromServer(data.player);
      const successMsg = data.message || (equip ? "Trang bị Cổ trùng thành công!" : "Tháo Cổ trùng thành công!");
      get().addLog(successMsg);
      get().addToast(successMsg, "success");
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Thao tác trang bị thất bại: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      set({ actionLoading: false });
    }
  },

  unequipAllGu: async () => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/inventory/unequip-all`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Tháo cổ trùng thất bại.");
      }

      setPlayerFromServer(data.player);
      if (data.message) {
        get().addLog(data.message);
        get().addToast(data.message, "success");
      }
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Tháo cổ trùng thất bại: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      set({ actionLoading: false });
    }
  },

  updateGuStatus: async (instanceId, status) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/inventory/status`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          instance_id: instanceId,
          status: status,
        }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Cập nhật trạng thái thất bại.");
      }

      setPlayerFromServer(data.player);
      get().addLog(data.message || "Cập nhật trạng thái Cổ trùng thành công!");
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Lỗi cập nhật trạng thái: ${err.message}`);
    } finally {
      set({ actionLoading: false });
    }
  },

  setVitalGu: async (instanceId) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/inventory/set-vital`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          gu_instance_id: instanceId,
        }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Thiết lập Bản Mệnh Cổ thất bại.");
      }

      setPlayerFromServer(data.player);
      const successMsg = data.message || "Thiết lập Bản Mệnh Cổ thành công!";
      get().addLog(successMsg);
      get().addToast(successMsg, "success");
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Lỗi thiết lập Bản Mệnh Cổ: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      set({ actionLoading: false });
    }
  },

  useGu: async (instanceId) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/inventory/use-gu`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          gu_instance_id: instanceId,
        }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Sử dụng Cổ trùng thất bại.");
      }

      setPlayerFromServer(data.player);
      const successMsg = data.message || "Sử dụng Cổ trùng thành công!";
      get().addLog(successMsg);
      get().addToast(successMsg, "success");
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Lỗi sử dụng Cổ trùng: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      set({ actionLoading: false });
    }
  },

  useItem: async (itemId) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/use-item`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          item_id: itemId,
        }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Sử dụng vật phẩm thất bại.");
      }

      setPlayerFromServer(data.player);
      const successMsg = data.message || "Sử dụng vật phẩm thành công!";
      get().addLog(successMsg);
      get().addToast(successMsg, "success");
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Sử dụng vật phẩm thất bại: ${err.message}`);
      get().addToast(err.message, "error");
    } finally {
      set({ actionLoading: false });
    }
  },

  useItems: async (itemIds) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading || !itemIds?.length) return false;

    set({ actionLoading: true, error: null });

    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/use-items`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ item_ids: itemIds }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        set({ error: data.error });
        get().addLog(`Sử dụng vật phẩm thất bại: ${data.error}`);
        get().addToast(data.error || "Không thể sử dụng vật phẩm.", "error");
        return false;
      }

      setPlayerFromServer(data.player);
      const successMsg = data.message || "Đã sử dụng vật phẩm.";
      get().addLog(successMsg);
      get().addToast(successMsg, "success");
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Sử dụng vật phẩm thất bại: ${err.message}`);
      get().addToast(err.message, "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  useGus: async (instanceIds) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading || !instanceIds?.length) return false;

    set({ actionLoading: true, error: null });
    let successCount = 0;
    let failedCount = 0;

    for (const guInstanceId of instanceIds) {
      try {
        const response = await authFetch(`${BACKEND_URL}/api/inventory/use-gu`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({ gu_instance_id: guInstanceId }),
        });

        if (response.status === 401) {
          logout();
          throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
        }

        const data = await response.json();
        if (!response.ok) {
          failedCount += 1;
          get().addToast(data.error || `Không thể sử dụng Cổ trùng ${guInstanceId}.`, "error");
          continue;
        }

        setPlayerFromServer(data.player);
        const successMsg = data.message || `Đã sử dụng Cổ trùng ${guInstanceId}.`;
        get().addLog(successMsg);
        get().addToast(successMsg, "success");
        successCount += 1;
      } catch (err: any) {
        failedCount += 1;
        set({ error: err.message });
        get().addLog(`Sử dụng Cổ trùng thất bại: ${err.message}`);
        get().addToast(err.message, "error");
      }
    }

    set({ actionLoading: false });
    return successCount > 0 && failedCount === 0;
  },

  movePlayer: async (x, y, attackTargetId?) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/explore/move`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          x: x,
          y: y,
          attack_target_id: attackTargetId,
        }),
      });

      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Di chuyển thất bại.");
      }

      setPlayerFromServer(data.player);
      get().addLog(data.log || "Di chuyển thành công.");
      
      // Handle skip battle result
      if (data.skip_result) {
        const skipResult = data.skip_result;
        if (skipResult.winner === "Phe Ta") {
          get().addToast(`Tự động chiến thắng ${skipResult.enemy_name || "kẻ thù"}!`, "success");
          if (skipResult.mine_occupied) {
            get().addToast(`Đã chiếm mỏ ${skipResult.mine_occupied}!`, "info");
          }
        } else {
          get().addToast(`Tự động chiến đấu thất bại! ${skipResult.enemy_name || "Kẻ thù"} quá mạnh.`, "error");
        }
      }
      
      return data;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Di chuyển thất bại: ${err.message}`);
      return null;
    } finally {
      set({ actionLoading: false });
    }
  },

  fetchGrid: async () => {
    const { token, logout } = get();
    if (!token) return null;

    try {
      const response = await authFetch(`${BACKEND_URL}/api/explore/grid`, {
        headers: {
        },
      });

      if (response.status === 401) {
        logout();
        return null;
      }

      if (!response.ok) {
        throw new Error("Failed to load map grid viewport");
      }

      const data = await response.json();
      return data;
    } catch (err: any) {
      console.error("Grid fetch failed:", err.message);
      get().addToast("Lỗi tải bản đồ.", "error");
      return null;
    }
  },

  createParty: async () => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/party/create`, {
        method: "POST",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Tạo tổ đội thất bại");
      set({ party: data.party });
      get().addLog(data.message || "Đã lập tổ đội thành công!");
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Lỗi tạo party: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchPartyStatus: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/party/status`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ party: data.party });
    } catch (err) {
      console.error("fetch party status failed", err);
      get().addToast("Lỗi tải trạng thái đội.", "error");
    }
  },

  invitePlayer: async (username) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/party/invite`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Không thể mời");
      set({ party: data.party });
      get().addLog(data.message || `Đã mời người chơi ${username} vào tổ đội.`);
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Mời tổ đội thất bại: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  leaveParty: async () => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/party/leave`, {
        method: "POST",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Rời tổ đội thất bại");
      set({ party: null });
      get().addLog(data.message || "Bạn đã rời khỏi tổ đội.");
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Rời tổ đội thất bại: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchPartyInvitations: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/party/invitations`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ partyInvitations: data.invitations || [] });
    } catch (err) {
      console.error("fetch party invitations failed", err);
    }
  },

  respondPartyInvitation: async (id, action) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/party/invitations/${id}/respond`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Thao tác thất bại.");
      if (action === "accept") {
        set({ party: data.party });
        await get().initializeSession();
      }
      set((state) => ({
        partyInvitations: state.partyInvitations.filter((inv) => inv.id !== id),
      }));
      get().addLog(data.message || `Đã ${action === "accept" ? "chấp nhận" : "từ chối"} lời mời.`);
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchArenaLeaderboard: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/arena/leaderboard`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ 
        arenaLeaderboard: Array.isArray(data.leaderboard) ? data.leaderboard : [],
        arenaDailyChallenges: data.daily_challenges || 0 
      });
    } catch (err) {
      console.error("fetch arena leaderboard failed", err);
      get().addToast("Lỗi tải bảng xếp hạng.", "error");
    }
  },

  fetchArenaShop: async () => {
    const { token } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/arena/shop`);
      if (response.ok) {
        const data = await response.json();
        set({ arenaShopItems: data.items || [] });
      }
    } catch (err) {
      console.error("fetch arena shop failed", err);
      get().addToast("Lỗi tải cửa hàng đấu trường.", "error");
    }
  },

  buyArenaItem: async (itemId, quantity) => {
    const { token, actionLoading } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/arena/shop/buy`, {
        method: "POST",
        body: JSON.stringify({ item_id: itemId, quantity: quantity || 1 }),
      });
      const data = await response.json();
      if (response.ok) {
        get().addLog(`Đổi vật phẩm thành công! Đã tiêu hao ${data.cost} Xu Đấu Trường.`);
        // @ts-ignore
        get().addToast(data.message, "success");
        await get().fetchPlayerStatus(get().player!.id);
        return true;
      } else {
        set({ error: data.error });
        // @ts-ignore
        get().addToast(data.error || "Lỗi giao dịch", "error");
        return false;
      }
    } catch (err: any) {
      set({ error: err.message });
      // @ts-ignore
      get().addToast("Lỗi kết nối", "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  // Death Arena Actions
  fetchDeathArenaCurrent: async () => {
    const { token, logout } = get();
    if (!token) return;
    set({ deathArenaLoading: true });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/death-arena/current`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({
        deathArenaTournament: data.tournament || null,
        deathArenaMyTeam: data.team || null,
        deathArenaMyTeamMembers: data.members || [],
      });
    } catch (err) {
      console.error("fetch death arena current failed", err);
      get().addToast("Lỗi tải thông tin Đấu Trường Sinh Tử.", "error");
    } finally {
      set({ deathArenaLoading: false });
    }
  },

  registerDeathArena: async () => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/death-arena/register`, {
        method: "POST",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Đăng ký thất bại");
      // @ts-ignore
      get().addToast(data.message || "Đăng ký thành công!", "success");
      get().addLog(data.message || "Đã đăng ký Đấu Trường Sinh Tử!");
      await get().fetchDeathArenaCurrent();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Đăng ký thất bại.", "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  unregisterDeathArena: async () => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/death-arena/register`, {
        method: "DELETE",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hủy đăng ký thất bại");
      // @ts-ignore
      get().addToast(data.message || "Hủy đăng ký thành công!", "success");
      get().addLog(data.message || "Đã hủy đăng ký Đấu Trường Sinh Tử!");
      await get().fetchDeathArenaCurrent();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Hủy đăng ký thất bại.", "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  fetchDeathArenaBracket: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/death-arena/bracket`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({
        deathArenaTournament: data.tournament || null,
        deathArenaBracket: data.matches || [],
      });
    } catch (err) {
      console.error("fetch death arena bracket failed", err);
      get().addToast("Lỗi tải bảng đấu.", "error");
    }
  },

  fetchDeathArenaMyTeam: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/death-arena/my-team`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({
        deathArenaMyTeam: data.team || null,
        deathArenaMyTeamMembers: data.members || [],
      });
    } catch (err) {
      console.error("fetch death arena my team failed", err);
      get().addToast("Lỗi tải thông tin đội.", "error");
    }
  },

  fetchDeathArenaMyMatches: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/death-arena/matches`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ deathArenaMyMatches: data.matches || [] });
    } catch (err) {
      console.error("fetch death arena matches failed", err);
      get().addToast("Lỗi tải trận đấu.", "error");
    }
  },

  fetchDeathArenaRewards: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/death-arena/rewards`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ deathArenaRewards: data.rewards || [] });
    } catch (err) {
      console.error("fetch death arena rewards failed", err);
      get().addToast("Lỗi tải phần thưởng.", "error");
    }
  },

  fetchDeathArenaHistory: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/death-arena/history`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ deathArenaHistory: data.tournaments || [] });
    } catch (err) {
      console.error("fetch death arena history failed", err);
      get().addToast("Lỗi tải lịch sử.", "error");
    }
  },

  claimDeathArenaReward: async (rewardId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/death-arena/claim-reward`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ reward_id: rewardId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Nhận thưởng thất bại");
      // @ts-ignore
      get().addToast(data.message || "Nhận thưởng thành công!", "success");
      get().addLog(data.message || "Đã nhận phần thưởng Đấu Trường Sinh Tử!");
      await get().fetchDeathArenaRewards();
      await get().fetchPlayerStatus(get().player!.id);
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Nhận thưởng thất bại.", "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  challengeArena: async (targetPlayerId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/arena/challenge`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ target_player_id: targetPlayerId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Khiêu chiến thất bại");
      setPlayerFromServer(data.player);
      useUIStore.getState().setView("arena");
      get().addLog(data.message || "Đã thách đấu đối thủ!");
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Khiêu chiến võ đài lỗi: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchLeaderboard: async (category) => {
    const { token, logout } = get();
    if (!token) return;
    set({ leaderboardLoading: true, leaderboardCategory: category });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/leaderboard/${category}`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({
        leaderboardEntries: Array.isArray(data.entries) ? data.entries : [],
        myLeaderboardRank: data.my_rank ? { rank: data.my_rank, value: data.my_value } : null,
        leaderboardTotalPlayers: data.total_players || 0,
        leaderboardLastUpdated: data.last_updated || "",
      });
    } catch (err) {
      console.error("fetch leaderboard failed", err);
      get().addToast("Lỗi tải bảng xếp hạng.", "error");
    } finally {
      set({ leaderboardLoading: false });
    }
  },

  toggleLeaderboardVisibility: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/leaderboard/toggle`, { method: "POST" });
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ leaderboardShowOnLeaderboard: data.show_on_leaderboard });
      get().addLog(data.show_on_leaderboard ? "Đã tham gia bảng xếp hạng" : "Đã rút khỏi bảng xếp hạng");
    } catch (err) {
      console.error("toggle leaderboard visibility failed", err);
      get().addToast("Lỗi thay đổi hiển thị.", "error");
    }
  },

  fetchLeaderboardSettings: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/leaderboard/settings`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ leaderboardShowOnLeaderboard: data.show_on_leaderboard });
    } catch (err) {
      console.error("fetch leaderboard settings failed", err);
      get().addToast("Lỗi tải cài đặt bảng xếp hạng.", "error");
    }
  },

  sendPrivateMail: async (receiverUsername, content) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/chat/mail/send`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ receiver_username: receiverUsername, content }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Gửi thư thất bại");
      get().addLog(data.message || "Đã gửi thư truyền tin!");
      if (data.player) setPlayerFromServer(data.player);
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Truyền tin lỗi: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchInbox: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/chat/mail/inbox`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      set({ inbox: Array.isArray(data.inbox) ? data.inbox : [] });
    } catch (err) {
      console.error("fetch inbox failed", err);
      get().addToast("Lỗi tải hộp thư.", "error");
    }
  },

  sendBaoHoangChat: async (content, channel) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/chat/bao-hoang/send`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ content, channel }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Gửi tin nhắn thất bại");
      get().addLog(data.message || "Đã phát truyền tin nhắn!");
      if (data.player) setPlayerFromServer(data.player);
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      get().addLog(`Lỗi gửi tin nhắn: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchBaoHoangMessages: async (channel) => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/chat/bao-hoang/messages?channel=${channel}`);
      if (response.status === 401) { return; }
      const data = await response.json();
      set({ baoHoangMessages: Array.isArray(data.messages) ? data.messages : [] });
    } catch (err) {
      console.error("fetch chat messages failed", err);
      get().addToast("Lỗi tải tin nhắn.", "error");
    }
  },

  fetchNotifications: async () => {
    const { token, logout } = get();
    if (!token) return;
    await fetchWithDedupe("fetchNotifications", async () => {
      try {
        const response = await authFetch(`${BACKEND_URL}/api/player/notifications?page=1&pageSize=50`);
        if (response.status === 401) { logout(); return; }
        const data = await response.json();
        const list = Array.isArray(data.notifications) ? data.notifications : [];
        set({
          notifications: list,
          notificationsLoaded: true,
          unreadNotifications: list.filter((n: PlayerNotification) => !n.is_read).length,
        });
      } catch (err) {
        console.error("fetch notifications failed", err);
      }
    });
  },

  fetchUnreadNotificationsCount: async () => {
    const { token } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/notifications/unread-count`);
      if (!response.ok) return;
      const data = await response.json();
      set({ unreadNotifications: typeof data.unread === "number" ? data.unread : 0 });
    } catch (err) {
      console.error("fetch unread count failed", err);
    }
  },

  markNotificationRead: async (id) => {
    const { token } = get();
    if (!token) return;
    set((state) => ({
      notifications: state.notifications.map((n) => (n.id === id ? { ...n, is_read: true } : n)),
      unreadNotifications: Math.max(0, state.unreadNotifications - 1),
    }));
    try {
      await authFetch(`${BACKEND_URL}/api/player/notifications/read`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ids: [id] }),
      });
    } catch (err) {
      console.error("mark notification read failed", err);
    }
  },

  markAllNotificationsRead: async () => {
    const { token } = get();
    if (!token) return;
    set((state) => ({
      notifications: state.notifications.map((n) => ({ ...n, is_read: true })),
      unreadNotifications: 0,
    }));
    try {
      await authFetch(`${BACKEND_URL}/api/player/notifications/read-all`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({}),
      });
    } catch (err) {
      console.error("mark all notifications read failed", err);
    }
  },

  fetchMineSlots: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/mine/slots`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      if (response.ok) {
        set({ mineSlots: Array.isArray(data) ? data : [] });
      }
    } catch (err) {
      console.error("fetch mine slots failed", err);
      get().addToast("Lỗi tải mỏ.", "error");
    }
  },

  occupyMineSlot: async (slotId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/mine/occupy`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ slot_id: slotId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Chiếm mỏ thất bại");
      get().addLog(data.message || "Chiếm mỏ thành công!");
      await get().initializeSession();
      await get().fetchMineSlots();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Chiếm mỏ lỗi: ${err.message}`);
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  claimMineRewards: async (slotId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/mine/claim`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ slot_id: slotId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Nhận thưởng thất bại");
      get().addLog(data.message || "Đã nhận thưởng tích lũy!");
      await get().initializeSession();
      await get().fetchMineSlots();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Nhận thưởng mỏ lỗi: ${err.message}`);
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  abandonMineSlot: async (slotId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/mine/abandon`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ slot_id: slotId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Rút lui thất bại");
      get().addLog(data.message || "Đã rời khỏi mỏ và nhận hết thưởng!");
      await get().initializeSession();
      await get().fetchMineSlots();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Rời mỏ lỗi: ${err.message}`);
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  attackMineSlot: async (slotId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/mine/attack`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ slot_id: slotId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Tấn công mỏ thất bại");
      get().addLog(data.message || "Bắt đầu thách đấu chiếm mỏ!");
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Tấn công mỏ lỗi: ${err.message}`);
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  // ── Pet Actions ──
  playerPets: [],
  playerPetsLoaded: false,

  fetchPlayerPets: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/pets`);
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      if (response.ok) {
        set({ playerPets: data.pets || [], playerPetsLoaded: true });
      }
    } catch {}
  },

  feedPet: async (petId, itemId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/pets/feed`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ pet_id: petId, item_id: itemId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Cho ăn thất bại");
      get().addToast(data.message || `Đã cho ăn +${data.restored} hunger`, "success");
      await get().fetchPlayerPets();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  setPetFormation: async (petIds) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/pets/formation`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ slots: petIds }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Cập nhật đội hình thất bại");
      get().addToast("Đội hình đã cập nhật", "success");
      await get().fetchPlayerPets();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  setPetSleep: async (petId, sleeping) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/pets/sleep`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ pet_id: petId, sleeping }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Cập nhật trạng thái thất bại");
      await get().fetchPlayerPets();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  releasePet: async (petId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/pets/release`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ pet_id: petId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Thả pet thất bại");
      get().addToast("Đã thả pet", "success");
      await get().fetchPlayerPets();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  mergePets: async (targetId, sourceIds) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/pets/merge`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ target_id: targetId, source_ids: sourceIds }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Gộp bầy thất bại");
      get().addToast(data.message || `Đã gộp thành ${data.count} cá thể!`, "success");
      await get().fetchPlayerPets();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  captureMonster: async () => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/combat/capture`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ player_id: player.id }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Bắt thất bại");
      setPlayerFromServer(data.player);
      if (data.captured) {
        get().addToast(data.message || "Bắt thành công!", "success");
        await get().fetchPlayerPets();
      } else {
        get().addToast(data.message || "Bắt thất bại!", "error");
      }
      return { captured: data.captured, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return null;
    } finally { set({ actionLoading: false }); }
  },

  fetchAperture: async () => {
    if (fetchAperturePromise) return fetchAperturePromise;
    const executeFetch = async () => {
      const { token, logout } = get();
      if (!token) return;
      try {
        const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture`);
        if (response.status === 401) {
          logout();
          return;
        }
        if (!response.ok) {
          set({ aperture: null });
          return;
        }
        const data = await response.json();
        set({ aperture: data });
        const player = get().player;
        if (player) {
          set({ manaSnapshot: createManaSnapshotFromPlayer(player, get().guDictionary, get().constitutionDictionary, data.time_rate) });
        }
      } catch (err) {
        console.error("Failed to fetch aperture:", err);
        get().addToast("Lỗi tải tiên thể.", "error");
        set({ aperture: null });
      } finally {
        fetchAperturePromise = null;
      }
    };
    fetchAperturePromise = executeFetch();
    return fetchAperturePromise;
  },

  fetchDaoMarks: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/dao-marks`);
      if (response.status === 401) {
        logout();
        return;
      }
      if (response.ok) {
        const data = await response.json();
        set({ daoMarks: data });
      }
    } catch (err) {
      console.error("Failed to fetch Dao Marks:", err);
      get().addToast("Lỗi tải dấu đạo.", "error");
    }
  },

  fetchApertureNodes: async () => {
    if (fetchApertureNodesPromise) return fetchApertureNodesPromise;
    const executeFetch = async () => {
      const { token, logout } = get();
      if (!token) return;
      try {
        const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes`);
        if (response.status === 401) {
          logout();
          return;
        }
        if (response.ok) {
          const data = await response.json();
          set({ apertureNodes: data });
        }
      } catch (err) {
        console.error("Failed to fetch aperture nodes:", err);
        get().addToast("Lỗi tải tiên thể.", "error");
      } finally {
        fetchApertureNodesPromise = null;
      }
    };
    fetchApertureNodesPromise = executeFetch();
    return fetchApertureNodesPromise;
  },

  placeApertureNode: async (nodeType: string, x: number, y: number, path?: string, prodKey?: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ node_type: nodeType, x, y, path, prod_key: prodKey }),
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể xây dựng điểm tài nguyên");
      }
      get().addToast(data.message || "Xây dựng mỏ tài nguyên thành công!", "success");
      get().addLog(data.message || "Đã đặt điểm tài nguyên mới.");
      await get().initializeSession();
      await get().fetchApertureNodes();
      await get().fetchVariantHumans();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  collectResourceNode: async (nodeId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes/${nodeId}/collect`, {
        method: "POST",
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể thu hoạch tài nguyên");
      }
      get().addToast(data.message || "Thu hoạch thành công!", "success");
      get().addLog(data.message || "Thu hoạch tài nguyên thành công.");
      await get().initializeSession();
      await get().fetchApertureNodes();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  collectAllResourceNodes: async () => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes/collect-all`, {
        method: "POST",
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể thu hoạch tài nguyên");
      }
      get().addToast(data.message || "Thu hoạch hàng loạt thành công!", "success");
      get().addLog(data.message || "Thu hoạch hàng loạt tài nguyên thành công.");
      await get().initializeSession();
      await get().fetchApertureNodes();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  uninstallApertureNode: async (nodeId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes/${nodeId}/uninstall`, {
        method: "POST",
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể tháo dỡ mỏ tài nguyên");
      }
      get().addToast(data.message || "Tháo dỡ thành công!", "success");
      get().addLog(data.message || "Đã tháo dỡ mỏ tài nguyên.");
      await get().initializeSession();
      await get().fetchApertureNodes();
      await get().fetchVariantHumans();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  deleteApertureNode: async (nodeId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes/${nodeId}`, {
        method: "DELETE",
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể san phẳng mỏ tài nguyên");
      }
      get().addToast(data.message || "San phẳng thành công!", "success");
      get().addLog(data.message || "Đã san phẳng mỏ tài nguyên.");
      await get().initializeSession();
      await get().fetchApertureNodes();
      await get().fetchVariantHumans();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  fetchVariantHumans: async () => {
    if (fetchVariantHumansPromise) return fetchVariantHumansPromise;
    const executeFetch = async () => {
      const { token, logout } = get();
      if (!token) return;
      try {
        const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/variant-humans`);
        if (response.status === 401) {
          logout();
          return;
        }
        if (response.ok) {
          const data = await response.json();
          set({ variantHumans: data });
        }
      } catch (err) {
        console.error("Failed to fetch variant humans:", err);
        get().addToast("Lỗi tải tiên thể.", "error");
      } finally {
        fetchVariantHumansPromise = null;
      }
    };
    fetchVariantHumansPromise = executeFetch();
    return fetchVariantHumansPromise;
  },

  setApertureTimeRate: async (rate: number) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/time-rate`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ time_rate: rate }),
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể điều chỉnh dòng thời gian");
      }
      get().addToast(data.message || "Điều chỉnh dòng thời gian thành công!", "success");
      get().addLog(data.message || "Đã điều chỉnh dòng thời gian Tiên Khiếu.");
      await get().fetchAperture();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  setApertureLabor: async (race: string, intensity: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/labor`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ race, labor_intensity: intensity }),
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể điều chỉnh cường độ lao động");
      }
      get().addToast(data.message || "Điều chỉnh cường độ lao động thành công!", "success");
      get().addLog(data.message || "Đã điều chỉnh cường độ lao động của dị nhân.");
      await get().fetchVariantHumans();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  repairApertureNode: async (nodeId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes/${nodeId}/repair`, {
        method: "POST",
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể sửa chữa điểm tài nguyên");
      }
      get().addToast(data.message || "Sửa chữa thành công!", "success");
      get().addLog(data.message || "Đã sửa chữa điểm tài nguyên.");
      await get().initializeSession();
      await get().fetchApertureNodes();
      await get().fetchVariantHumans();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  repairAllApertureNodes: async () => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes/repair-all`, {
        method: "POST",
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể sửa chữa công trình");
      }
      get().addToast(data.message || "Sửa chữa hàng loạt thành công!", "success");
      get().addLog(data.message || "Đã sửa chữa hàng loạt công trình.");
      await get().initializeSession();
      await get().fetchApertureNodes();
      await get().fetchVariantHumans();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  upgradeApertureNode: async (nodeId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes/${nodeId}/upgrade`, {
        method: "POST",
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Không thể nâng cấp công trình");
      }
      get().addToast(data.message || "Nâng cấp thành công!", "success");
      get().addLog(data.message || "Đã nâng cấp công trình.");
      await get().initializeSession();
      await get().fetchApertureNodes();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  fetchActiveTribulation: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/tribulations/active`);
      if (response.status === 401) {
        logout();
        return;
      }
      if (response.ok) {
        const data = await response.json();
        set({ activeTribulation: data });
      }
    } catch (err) {
      console.error("Failed to fetch active tribulation:", err);
      get().addToast("Lỗi tải thiên kiếp.", "error");
    }
  },

  engageTribulationCombat: async (eventId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/tribulations/${eventId}/engage`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Xuất trận thất bại");
      }
      get().addLog(data.message || "Xuất trận nghênh chiến quái tai kiếp.");
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  resolveTribulation: async (eventId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/tribulations/${eventId}/resolve`, {
        method: "POST",
      });
      if (response.status === 401) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Kết toán tai kiếp thất bại");
      }
      get().addToast(data.message || "Kết toán tai kiếp hoàn thành!", "success");
      get().addLog(data.message || "Kết toán tai kiếp hoàn thành.");
      await get().initializeSession();
      await get().fetchAperture();
      await get().fetchDaoMarks();
      set({ activeTribulation: null });
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },

  useGutsGu: async () => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/soul/use-guts`, { method: "POST" });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Sử dụng Cổ trùng Can Đảm thất bại.");
      setPlayerFromServer(data.player);
      get().addToast(data.message || "Tráng hồn thành công!", "success");
      get().addLog(data.message || "Đã sử dụng 1x Can Đảm Cổ để tráng đại linh hồn.");
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  refineSoul: async () => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/soul/refine`, { method: "POST" });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Luyện hồn thất bại.");
      setPlayerFromServer(data.player);
      get().addToast(data.message || "Luyện hồn thành công!", "success");
      get().addLog(data.message || "Đã luyện hồn tại Lạc Phách Cốc, nén ép linh hồn ngưng thực.");
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchInstances: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/instance/list`);
      if (response.status === 401) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({
          availableInstances: data.instances || [],
          instanceDailyEntriesUsed: data.daily_entries_used ?? 0,
          instanceDailyEntriesMax: data.daily_entries_max ?? 2,
        });
      }
    } catch (err) {
      console.error("Failed to fetch instances:", err);
    }
  },

  fetchActiveInstance: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/instance/active`);
      if (response.status === 401) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        if (data.active) {
          set({
            activeInstanceSession: data.session,
            activeInstanceTemplate: data.template,
          });
        } else {
          set({
            activeInstanceSession: null,
            activeInstanceTemplate: null,
          });
        }
      }
    } catch (err) {
      console.error("Failed to fetch active instance:", err);
      get().addToast("Lỗi tải phó bản.", "error");
    }
  },

  enterInstance: async (instanceId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/instance/enter`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ instance_id: instanceId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Không thể tiến vào phó bản.");
      set({
        activeInstanceSession: data.session,
        activeInstanceTemplate: data.template,
      });
      get().addToast(data.message || "Tiến vào phó bản thành công!", "success");
      get().addLog(data.message || `Tiến vào phó bản: ${instanceId}`);
      await get().initializeSession();
      await get().fetchInstances();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  actionInstance: async (optionIndex: number) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/instance/action`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ option_index: optionIndex }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hành động phó bản thất bại.");
      
      setPlayerFromServer(data.player);
      if (data.session) {
        set({ activeInstanceSession: data.session });
        if (data.session.status !== "ACTIVE") {
          set({
            activeInstanceSession: null,
            activeInstanceTemplate: null,
          });
        }
      }
      get().addToast(data.message || "Thực hiện hành động thành công!", "success");
      get().addLog(data.message || "Đã hành động trong phó bản.");
      await get().initializeSession();
      if (data.session?.status !== "ACTIVE") {
        await get().fetchInstances();
      }
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchSects: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects`);
      if (response.status === 401) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({ sects: data || [] });
      }
    } catch (err) {
      console.error("Failed to fetch sects:", err);
      get().addToast("Lỗi tải danh sách tông môn.", "error");
    }
  },

  joinSect: async (sectId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/${sectId}/join`, {
        method: "POST",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Gia nhập tông môn thất bại.");
      get().addToast(data.message || "Gia nhập tông môn thành công!", "success");
      get().addLog(data.message || `Gia nhập tông môn: ${sectId}`);
      await get().fetchMySectStatus();
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  leaveSect: async () => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/leave`, {
        method: "POST",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Rời tông môn thất bại.");
      get().addToast(data.message || "Đã rời tông môn!", "info");
      get().addLog(data.message || "Rời tông môn.");
      set({ mySectStatus: null });
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  cancelSectJoinRequest: async () => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/join-request/cancel`, {
        method: "POST",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hủy đơn xin gia nhập thất bại.");
      get().addToast(data.message || "Đã hủy đơn xin gia nhập tông môn!", "info");
      get().addLog(data.message || "Hủy đơn xin gia nhập tông môn.");
      await get().fetchMySectStatus();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchMySectStatus: async () => {
    if (fetchMySectStatusPromise) return fetchMySectStatusPromise;
    const executeFetch = async () => {
      const { token, logout } = get();
      if (!token) return;
      try {
        const response = await authFetch(`${BACKEND_URL}/api/sects/me`);
        if (response.status === 401) { logout(); return; }
        if (response.ok) {
          const data = await response.json();
          set({ mySectStatus: data });
        }
      } catch (err) {
        console.error("Failed to fetch my sect status:", err);
        get().addToast("Lỗi tải thông tin tông môn.", "error");
      } finally {
        fetchMySectStatusPromise = null;
      }
    };
    fetchMySectStatusPromise = executeFetch();
    return fetchMySectStatusPromise;
  },

  promoteSectRank: async () => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/promote`, {
        method: "POST",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Khảo hạch thăng cấp thất bại.");
      get().addToast(data.message || "Thăng cấp thành công!", "success");
      get().addLog(data.message || "Thăng cấp tông môn.");
      await get().fetchMySectStatus();
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  donateSectStones: async (amount: number) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/donate/stones`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ amount }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Quyên góp thất bại.");
      get().addToast(data.message || "Quyên góp Nguyên Thạch thành công!", "success");
      get().addLog(data.message || `Quyên góp ${amount} Nguyên Thạch.`);
      await get().fetchMySectStatus();
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  donateSectMaterials: async (materialId: string, amount: number) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/donate/materials`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ material_id: materialId, amount }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Quyên góp thất bại.");
      get().addToast(data.message || "Quyên góp nguyên liệu thành công!", "success");
      get().addLog(data.message || `Quyên góp ${amount}x ${materialId}.`);
      await get().fetchMySectStatus();
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  donateSectGu: async (guInstanceId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/donate/gu`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ gu_instance_id: guInstanceId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Quyên góp thất bại.");
      get().addToast(data.message || "Quyên góp Cổ trùng thành công!", "success");
      get().addLog(data.message || `Quyên góp Cổ trùng.`);
      await get().fetchMySectStatus();
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  donateSectRecipe: async (recipeId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/donate/recipe`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ recipe_id: recipeId }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Quyên góp thất bại.");
      get().addToast(data.message || "Quyên góp Bí phương thành công!", "success");
      get().addLog(data.message || `Quyên góp Bí phương.`);
      await get().fetchMySectStatus();
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  donateSectBulk: async (payload: {
    materials: { material_id: string; amount: number }[];
    gus: { gu_instance_id: string }[];
    recipes: { recipe_id: string }[];
  }) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/donate/bulk`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Quyên góp hàng loạt thất bại.");
      get().addToast(data.message || "Quyên góp hàng loạt thành công!", "success");
      get().addLog(data.message || `Quyên góp hàng loạt vật phẩm.`);
      await get().fetchMySectStatus();
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  buyFromSectVault: async (itemType: string, itemId: string, quantity = 1, rank?: number) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/vault/buy`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ item_type: itemType, item_id: itemId, quantity, rank }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Trao đổi vật phẩm thất bại.");
      get().addToast(data.message || "Trao đổi vật phẩm thành công!", "success");
      get().addLog(data.message || `Đổi ${itemType}: ${itemId} x${quantity} từ bảo khố.`);
      await get().fetchMySectStatus();
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchSectMissions: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/missions`);
      if (response.status === 401) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({ sectMissions: data || [] });
      }
    } catch (err) {
      console.error("Failed to fetch sect missions:", err);
      get().addToast("Lỗi tải nhiệm vụ tông môn.", "error");
    }
  },

  acceptSectMission: async (missionId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/missions/${missionId}/accept`, {
        method: "POST",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Nhận nhiệm vụ thất bại.");
      get().addToast(data.message || "Nhận nhiệm vụ thành công!", "success");
      get().addLog(data.message || `Nhận nhiệm vụ tông môn: ${missionId}`);
      await get().fetchMySectStatus();
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  completeSectMission: async (missionId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/missions/${missionId}/complete`, {
        method: "POST",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hoàn thành nhiệm vụ thất bại.");
      get().addToast(data.message || "Hoàn thành nhiệm vụ thành công!", "success");
      get().addLog(data.message || `Hoàn thành nhiệm vụ tông môn: ${missionId}`);
      await get().fetchMySectStatus();
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  deleteSectMission: async (missionId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/missions/${missionId}`, {
        method: "DELETE",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Xóa nhiệm vụ thất bại.");
      get().addToast(data.message || "Đã xóa nhiệm vụ tông môn!", "success");
      get().addLog(data.message || `Xóa nhiệm vụ tông môn: ${missionId}`);
      await get().fetchSectMissions();
      await get().fetchMySectStatus();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  abandonSectMission: async (missionId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/missions/${missionId}/abandon`, {
        method: "DELETE",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hủy nhiệm vụ thất bại.");
      get().addToast(data.message || "Đã hủy nhiệm vụ tông môn!", "success");
      get().addLog(data.message || `Hủy nhiệm vụ tông môn: ${missionId}`);
      await get().fetchMySectStatus();
      await get().fetchSectMissions();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  createSect: async (name: string, description: string, region: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/create`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, description, region }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Lập phái thất bại.");
      get().addToast(data.message || "Khai tông lập phái thành công!", "success");
      get().addLog(data.message || `Lập phái: ${name}`);
      await get().fetchMySectStatus();
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchSectMembers: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/me/members`);
      if (response.status === 401) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({ sectMembers: data || [] });
      }
    } catch (err) {
      console.error("Failed to fetch sect members:", err);
      get().addToast("Lỗi tải thành viên tông môn.", "error");
    }
  },

  setMemberRank: async (memberId: string, rank: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/members/${memberId}/rank`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ rank }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Điều chỉnh chức vụ thất bại.");
      get().addToast(data.message || "Đã điều chỉnh chức vụ!", "success");
      get().addLog(data.message || `Điều chỉnh chức vụ thành viên: ${memberId} sang ${rank}`);
      await get().fetchSectMembers();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  kickMember: async (memberId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/members/${memberId}/kick`, {
        method: "POST",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Trục xuất đệ tử thất bại.");
      get().addToast(data.message || "Đã trục xuất đệ tử!", "info");
      get().addLog(data.message || `Trục xuất đệ tử: ${memberId}`);
      await get().fetchSectMembers();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  createCustomMission: async (payload) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/missions/create`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          title: payload.title,
          description: payload.description,
          mission_type: payload.mission_type,
          target_id: payload.target_id,
          target_amount: payload.target_amount,
          reward_contribution: payload.reward_contribution,
          required_member_rank: payload.required_member_rank,
          duration_hours: payload.duration_hours,
        }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Phát hành nhiệm vụ thất bại.");
      get().addToast(data.message || "Đã phát hành nhiệm vụ môn phái!", "success");
      get().addLog(data.message || `Phát hành nhiệm vụ: ${payload.title}`);
      await get().fetchSectMissions();
      await get().fetchMySectStatus();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  stockSectVault: async (itemType, itemId, quantity) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/vault/stock`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          item_type: itemType,
          item_id: itemId,
          quantity: quantity,
        }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Nhập kho bảo khố thất bại.");
      get().addToast(data.message || "Đã nhập kho bảo khố thành công!", "success");
      get().addLog(`Nhập bảo khố: ${quantity}x ${itemId}`);
      await get().fetchMySectStatus();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  stockSectVaultBulk: async (items) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/vault/stock/bulk`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ items }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Nhập kho bảo khố thất bại.");
      get().addToast(data.message || "Đã nhập kho bảo khố thành công!", "success");
      get().addLog(`Nhập bảo khố hàng loạt: ${items.length} mặt hàng`);
      await get().fetchMySectStatus();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  upgradeSect: async () => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/upgrade`, {
        method: "POST",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Nâng cấp tông môn thất bại.");
      get().addToast(data.message || "Nâng cấp tông môn thành công!", "success");
      get().addLog(`Nâng cấp tông môn: Cấp ${data.new_level}`);
      await get().fetchMySectStatus();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  toggleSectApproval: async (requireApproval) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/toggle-approval`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ require_approval: requireApproval }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Cài đặt thất bại.");
      get().addToast(data.message || "Đã cập nhật cài đặt duyệt thành viên!", "success");
      await get().fetchMySectStatus();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  toggleSectPromotionApproval: async (requirePromotionApproval) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/toggle-promotion-approval`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ require_promotion_approval: requirePromotionApproval }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Cài đặt thất bại.");
      get().addToast(data.message || "Đã cập nhật cài đặt duyệt lên cấp!", "success");
      await get().fetchMySectStatus();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  paySectSalary: async () => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/pay-salary`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ confirm: true }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Phát lương thất bại.");
      get().addToast(data.message || "Đã phát lương thành công!", "success");
      await get().fetchMySectStatus();
      await get().fetchSectMembers();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchJoinRequests: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/join-requests`);
      if (response.status === 401) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({ sectJoinRequests: data || [] });
      }
    } catch (err) {
      console.error("Failed to fetch join requests:", err);
      get().addToast("Lỗi tải yêu cầu gia nhập.", "error");
    }
  },

  actionJoinRequest: async (requestId, action) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/join-requests/${requestId}/action`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Thao tác thất bại.");
      get().addToast(data.message || "Thao tác thành công!", "success");
      await get().fetchJoinRequests();
      await get().fetchSectMembers();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchPromotionRequests: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/promotion-requests`);
      if (response.status === 401) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({ sectPromotionRequests: data || [] });
      }
    } catch (err) {
      console.error("Failed to fetch promotion requests:", err);
      get().addToast("Lỗi tải yêu cầu thăng chức.", "error");
    }
  },

  actionPromotionRequest: async (requestId, action) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/promotion-requests/${requestId}/action`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Thao tác thất bại.");
      get().addToast(data.message || "Thao tác thành công!", "success");
      await get().fetchPromotionRequests();
      await get().fetchSectMembers();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchFeedbacks: async (category, status) => {
    const { token, logout } = get();
    if (!token) return;
    try {
      let url = `${BACKEND_URL}/api/feedback`;
      const params = [];
      if (category) params.push(`category=${encodeURIComponent(category)}`);
      if (status) params.push(`status=${encodeURIComponent(status)}`);
      if (params.length > 0) {
        url += `?${params.join("&")}`;
      }
      const response = await authFetch(url);
      if (response.status === 401) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({ feedbacks: data || [] });
      }
    } catch (err) {
      console.error("Failed to fetch feedbacks:", err);
      get().addToast("Lỗi tải phản hồi.", "error");
    }
  },

  submitFeedback: async (title, description, category) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/feedback`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ title, description, category }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Gửi góp ý thất bại.");
      get().addToast(data.message || "Gửi góp ý thành công!", "success");
      await get().fetchFeedbacks();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  voteFeedback: async (feedbackId, voteType) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });

    // Optimistic UI update
    const prevFeedbacks = [...get().feedbacks];
    const updatedFeedbacks = prevFeedbacks.map((f) => {
      if (f.id === feedbackId) {
        const prevVote = f.user_vote;
        let newTotal = f.total_votes;
        if (prevVote === undefined || prevVote === null) {
          newTotal += voteType;
        } else if (prevVote === voteType) {
          newTotal -= voteType; // Toggle off
        } else {
          newTotal += 2 * voteType; // Changed vote type
        }
        return {
          ...f,
          total_votes: newTotal,
          user_vote: prevVote === voteType ? null : voteType,
        };
      }
      return f;
    });
    set({ feedbacks: updatedFeedbacks });

    try {
      const response = await authFetch(`${BACKEND_URL}/api/feedback/${feedbackId}/vote`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ vote_type: voteType }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Bình chọn thất bại.");
      get().addToast(data.message || "Bình chọn thành công!", "success");
      // Update with exact values from server
      set((state) => ({
        feedbacks: state.feedbacks.map((f) =>
          f.id === feedbackId ? { ...f, total_votes: data.total_votes, user_vote: data.user_vote } : f
        ),
      }));
      return true;
    } catch (err: any) {
      // Revert on error
      set({ feedbacks: prevFeedbacks, error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  updateFeedbackStatus: async (feedbackId, status, developerResponse) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/feedback/${feedbackId}/status`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ status, developer_response: developerResponse }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Cập nhật trạng thái thất bại.");
      get().addToast(data.message || "Cập nhật trạng thái thành công!", "success");
      await get().fetchFeedbacks();
      // Also refresh player stats in case they received a reward
      if (get().player) {
        await get().fetchPlayerStatus(get().player!.id);
      }
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchFeedbackComments: async (feedbackId) => {
    const { token, logout } = get();
    if (!token) return false;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/feedback/${feedbackId}/comments`);
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Tải góp ý thất bại.");
      set((state) => ({ feedbackComments: { ...state.feedbackComments, [feedbackId]: data || [] } }));
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message || "Thao tác thất bại.", "error");
      return false;
    }
  },

  addFeedbackComment: async (feedbackId, content) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/feedback/${feedbackId}/comments`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ content }),
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Góp ý thất bại.");
      get().addToast(data.message || "Đã góp ý!", "success");
      await get().fetchFeedbackComments(feedbackId);
      set((state) => ({
        feedbacks: state.feedbacks.map((f) =>
          f.id === feedbackId ? { ...f, comment_count: f.comment_count + 1 } : f
        ),
      }));
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  deleteFeedbackComment: async (feedbackId, commentId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/feedback/${feedbackId}/comments/${commentId}`, {
        method: "DELETE",
      });
      if (response.status === 401) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Xóa góp ý thất bại.");
      get().addToast(data.message || "Đã xóa góp ý!", "success");
      await get().fetchFeedbackComments(feedbackId);
      set((state) => ({
        feedbacks: state.feedbacks.map((f) =>
          f.id === feedbackId ? { ...f, comment_count: Math.max(0, f.comment_count - 1) } : f
        ),
      }));
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },
})});
