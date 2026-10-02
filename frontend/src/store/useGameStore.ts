import { create } from "zustand";
import FingerprintJS from "@fingerprintjs/fingerprintjs";
import type { RequiredGus } from "@/lib/gu-dict";
import { manaRegenPerTick, nativeNumericValue } from "@/lib/effect-blocks";
import type { EffectDictionaryEntry, EffectEnvelope } from "@/lib/effect-blocks";
import { useUIStore } from "@/store/useUIStore";
import { wsManager } from "@/lib/websocket";
import { authFetch } from "@/lib/auth";
import { fetchWithDedupe } from "@/lib/fetch";
import { BACKEND_URL } from "@/lib/config";
import { getDictionaryCache, setDictionaryCache } from "@/lib/db-cache";
import { createNotificationsSlice, type NotificationsSlice } from "@/store/slices/notificationsSlice";
import { createMineSlice, type MineSlice } from "@/store/slices/mineSlice";
import { createPartySlice, type PartySlice } from "@/store/slices/partySlice";
import { createChatSlice, type ChatSlice } from "@/store/slices/chatSlice";
import { createArenaSlice, type ArenaSlice } from "@/store/slices/arenaSlice";
import { createFeedbackSlice, type FeedbackSlice } from "@/store/slices/feedbackSlice";
import { createBettingSlice, type BettingSlice } from "@/store/slices/bettingSlice";
import { createGachaSlice, type GachaSlice } from "@/store/slices/gachaSlice";
import { createAchievementsSlice, type AchievementsSlice } from "@/store/slices/achievementsSlice";
import { createPetsSlice, type PetsSlice } from "@/store/slices/petsSlice";
import { createInstanceSlice, type InstanceSlice } from "@/store/slices/instanceSlice";
import { createMarketplaceSlice, type MarketplaceSlice, type BuybackOrder, type AsyncTradeOffer, type TradeOfferData } from "@/store/slices/marketplaceSlice";
export type { BuybackOrder, AsyncTradeOffer, TradeOfferData };
import { createSectSlice, type SectSlice } from "@/store/slices/sectSlice";
import { createApertureSlice, type ApertureSlice } from "@/store/slices/apertureSlice";
import { createTradeSlice, type TradeSlice } from "@/store/slices/tradeSlice";
import { createInventorySlice, type InventorySlice } from "@/store/slices/inventorySlice";
import { createCultivationSlice, type CultivationSlice } from "@/store/slices/cultivationSlice";
import { createProfileSlice, type ProfileSlice } from "@/store/slices/profileSlice";
import { createAuthSlice, type AuthSlice } from "@/store/slices/authSlice";
import { createReferralsSlice, type ReferralsSlice } from "@/store/slices/referralsSlice";
import { createMiscSlice, type MiscSlice } from "@/store/slices/miscSlice";
import { createCombatSlice, type CombatSlice } from "@/store/slices/combatSlice";
import { createTeleportSlice, type TeleportSlice } from "@/store/slices/teleportSlice";
import { createScoutingSlice, type ScoutingSlice } from "@/store/slices/scoutingSlice";
import { shouldApplyPlayerRevision } from "@/store/playerRevision";
import { acquirePendingAction, releasePendingAction, type PendingActions } from "@/store/pendingActions";

export interface GuInstance {
  instance_id: string;
  gu_id: string;
  rank: number;
  hunger: number;
  status: string; // "ACTIVE", "SLEEPING", "DEAD"
  equipped: boolean;
  is_vital: boolean;
  is_locked?: boolean;
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
  insight_bonus?: number;
  insight_stacks?: number;
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
  target_sect_id?: string;
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

// List responses omit logs/combatants (fetched per entry via /combat/history/:id);
// only logs_count is included for the list view.
export interface CombatHistoryEntry {
  id: string;
  player_id: string;
  winner: string;
  logs?: ApiResponseData[];
  logs_count?: number;
  summary: {
    total_damage_dealt: number;
    total_damage_taken: number;
    shields_used: number;
    final_player_hp: number;
    items_used: number;
    stones_used: number;
    loot_count: number;
  };
  combatants?: ApiResponseData[];
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
  reward_items: ApiResponseData[];
  claimed: boolean;
  created_at: string;
}

export interface Player {
  id: string;
  state_revision: number;
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
  is_dead?: boolean;
  is_admin?: boolean;
  can_access_admin?: boolean;
  is_sparse_gu_inventory?: boolean;
  avatar_url?: string;
  active_concealment?: boolean;
  last_sever_clues_at?: string;
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
  version?: number;
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
  mana_regen_rate?: number;
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
    const spec = guSpecFor(gu.gu_id, gu.rank);
    if (!spec) continue;
    total += manaRegenPerTick(spec.effects) * manaRegenTickFactor(player.rank, gu.rank);
  }
  return total;
};

const getManaRegenRatePerHour = (snapshot: ManaSnapshot): number => {
  if (snapshot.mana_regen_rate !== undefined) return snapshot.mana_regen_rate;

  let rate: number;
  if (snapshot.rank >= 6) {
    const essenceProduction = snapshot.essence_production_per_year ?? 10;
    rate = essenceProduction / 10.0;
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

export const createManaSnapshotFromPlayer = (player: Player, guDictionary: Record<string, GuSpec>, constitutionDictionary?: Record<string, EffectDictionaryEntry>, timeRate?: number): ManaSnapshot => ({
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
  mana_regen_rate: player.mana_regen_rate,
  mana_regen_tick_per_hour: manaRegenTickPerHour(player, guDictionary),
  mana_regen_mult: 1 + (player.constitution ? nativeNumericValue(constitutionDictionary?.[player.constitution], "MANA_REGEN") : 0),
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
  next_tribulation_at?: string | null;
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

export interface DaoNganPathDetail {
  path: string;
  amount: number;
  multiplier: number;
  base_multiplier?: number;
  power: number;
  attainment_level: number;
  attainment_name: string;
  realm_control_pct: number;
  realm_efficiency: number;
  internal_conflict_penalty_pct: number;
  natural_resistance_pct: number;
  purity_pct: number;
}

export interface DaoNganResult {
  total_dao_ngan: number;
  total_marks: number;
  unique_path_count: number;
  average_multiplier: number;
  max_multiplier: number;
  max_path: string;
  dominant_path: string;
  grace_threshold: number;
  has_non_conflicting_dao: boolean;
  paths: DaoNganPathDetail[];
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
  prosperity: number;
  gu_master_count: number;
  gu_immortal_count: number;
  food_stored: number;
  food_per_capita: number;
  population_growth_rate: number;
  mortality_rate: number;
  created_at: string;
  updated_at: string;
}

export interface ApertureNodeWorker {
  id: string;
  node_id: string;
  variant_human_id: string;
  race: string;
  population: number;
  assigned_population: number;
  created_at: string;
  updated_at: string;
}

export interface ApertureRefinementJob {
  id: string;
  aperture_id: string;
  recipe_id: string;
  mode: 'FIXED_COUNT' | 'UNTIL_SUCCESS' | 'UNTIL_MATERIAL_EMPTY' | 'MAINTAIN_STOCK';
  target_quantity: number;
  current_quantity: number;
  priority: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface ApertureSettlement {
  id: string;
  aperture_id: string;
  race: string;
  name: string;
  settlement_type: string;
  population_capacity: number;
  housing: number;
  food_storage: number;
  security: number;
  created_at: string;
  updated_at: string;
}

export interface ApertureVariantNPC {
  id: string;
  aperture_id: string;
  race: string;
  name: string;
  rank: number;
  sub_rank: number;
  path: string;
  role: string;
  loyalty: number;
  talent: number;
  status: string;
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

export interface GroundDrop {
  id: string;
  region_id: string;
  x: number;
  y: number;
  loser_id: string;
  loser_name: string;
  killer_id?: string;
  killer_name?: string;
  loot_data: {
    primeval_stones?: number;
    immortal_stones?: number;
    items?: Array<{ item_id: string; category: string; quantity: number }>;
    gus?: ApiResponseData[];
    recipes?: string[];
  };
  is_protected: boolean;
  protected_until?: string;
  expires_at: string;
  created_at: string;
}


export interface GuSpec {
  id: string;
  name: string;
  rank: number;
  path: string[];
  gu_type: string;
  base_mana_cost: number;
  effects: EffectEnvelope;
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
  effects?: EffectEnvelope;
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
  requirements: Record<string, ApiResponseData>;
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
  rewards: Record<string, ApiResponseData>;
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
  rewards: Record<string, ApiResponseData>;
  cooldown_hours: number;
  cooldown_remaining: number;
}

export interface PlayerInstanceSession {
  id: string;
  player_id: string;
  instance_id: string;
  current_stage: number;
  status: "ACTIVE" | "COMPLETED" | "FAILED";
  temp_inventory: ApiResponseData[];
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
  buildings?: Record<string, number>;
  map_x?: number;
  map_y?: number;
  shield_until?: string;
  require_approval?: boolean;
  require_promotion_approval?: boolean;
  last_relocated_at?: string;
}

export interface SectRelocationEstimate {
  can_relocate: boolean;
  reason?: string;
  is_leader: boolean;
  cooldown_remaining_seconds: number;
  cooldown_total_seconds: number;
  current_funds: number;
  funds_needed: number;
  cost_breakdown: {
    base_cost: number;
    member_count: number;
    member_cost: number;
    materials_count: number;
    materials_cost: number;
    mortal_gu_count: number;
    mortal_gu_cost: number;
    immortal_gu_count: number;
    immortal_gu_cost: number;
    recipes_count: number;
    recipes_cost: number;
    treasury_level: number;
    treasury_discount: number;
    total_cost: number;
  };
  current_sect_location: {
    region: string;
    x: number;
    y: number;
  };
  target_location: {
    region: string;
    x: number;
    y: number;
    radius: number;
  };
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
  is_defending: boolean;
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
  estimated_daily_maintenance?: number;
  maintenance_breakdown?: {
    level_fee: number;
    sect_level: number;
    member_fee: number;
    member_count: number;
    materials_fee: number;
    materials_count: number;
    mortal_gu_fee: number;
    mortal_gu_count: number;
    immortal_gu_fee: number;
    immortal_gu_count: number;
    recipes_fee: number;
    recipes_count: number;
    treasury_level: number;
    treasury_discount: number;
    total_fee: number;
  };
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
  last_activity?: string;
  is_online?: boolean;
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

export interface AuthResponse {
  player: Player;
  token?: string;
}

// Slim action response carrying only inventory deltas.
// Player stat changes (hp, mana, stones, rank, etc.) arrive via WebSocket.
interface ActionInventoryResponse {
  success?: boolean;
  message?: string;
  gu_inventory?: GuInstance[];
  materials_inventory?: Record<string, number>;
  hp?: number;
  current_thought?: number;
  soul_hp?: number;
  lifespan?: number;
}

export interface ExploreResponse extends ActionInventoryResponse {
  reward_stones?: number;
  log: string;
}

export interface FeedResponse extends ActionInventoryResponse {
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

// ToastMessage is defined in useUIStore — re-export for backwards compat.
export type { ToastMessage } from "@/store/useUIStore";

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

interface GameState extends NotificationsSlice, MineSlice, PartySlice, ChatSlice, ArenaSlice, FeedbackSlice, BettingSlice, GachaSlice, AchievementsSlice, PetsSlice, InstanceSlice, MarketplaceSlice, SectSlice, ApertureSlice, TradeSlice, InventorySlice, CombatSlice, CultivationSlice, ProfileSlice, AuthSlice, ReferralsSlice, MiscSlice, TeleportSlice, ScoutingSlice {
  player: Player | null;
  playerStateRevision: number | null;
  manaSnapshot: ManaSnapshot | null;
  token: string | null;
  isLoading: boolean;
  actionLoading: boolean;
  pendingActions: PendingActions;
  playerRevision: number;
  beginAction: (key: string) => boolean;
  endAction: (key: string) => void;
  isActionPending: (key: string) => boolean;
  error: string | null;
  logs: string[];
  guDictionary: Record<string, GuSpec>;
  guByRank: Record<string, Record<number, GuSpec>>;
  itemDictionary: Record<string, ItemSpec>;
  monsterDictionary: Record<string, MonsterSpec>;
  constitutionDictionary: Record<string, EffectDictionaryEntry>;
  raceDictionary: Record<string, EffectDictionaryEntry>;
  statusDictionary: Record<string, EffectDictionaryEntry>;
  dictionaryLoaded: boolean;
  dictionaryLoading: boolean;
  dictionaryError: string | null;
  dictionaryPartsLoaded: {
    gus: boolean;
    items: boolean;
    monsters: boolean;
    constitutions: boolean;
    races: boolean;
    statuses: boolean;
  };
  fetchGuDictionary: () => Promise<void>;
  fetchItemDictionary: () => Promise<void>;
  fetchMonsterDictionary: () => Promise<void>;
  fetchConstitutionDictionary: () => Promise<void>;
  fetchRaceDictionary: () => Promise<void>;
  fetchStatusDictionary: () => Promise<void>;
  updateManaSnapshot: (player: Player | null) => void;
  setPlayerFromServer: (player: Player | Partial<Player> | null, extraState?: Partial<GameState>, expectedRevision?: number) => boolean;
  applyInventoryFromResponse: (data: ApiResponseData) => void;
  getEstimatedMana: () => number | null;
  subscribeEvents: () => void;
  unsubscribeEvents: () => void;
  subscribePlayerEvents: () => void;
  unsubscribePlayerEvents: () => void;
  playerEventsConnected: boolean;
  playerEventsRetries: number;
  reconciliationRevision: number;
  reconciliationInFlight: boolean;
  subscribeTradeEvents: () => void;
  unsubscribeTradeEvents: () => void;
  subscribeBaoHoangMessages: (channel: string) => void;
  unsubscribeBaoHoangMessages: () => void;
  fetchDictionary: (forceFresh?: boolean) => Promise<void>;

  setShowChangelog: (v: boolean) => void;

  simulateDeposit: (stones: number) => Promise<boolean>;
  addLog: (message: string) => void;
  setView: (view: string) => void;




  // Toast System Actions (state lives in useUIStore)
  addToast: (message: string, type?: "success" | "error" | "info", duration?: number) => void;
  removeToast: (id: string) => void;

  // Confirm Dialog Actions (state lives in useUIStore)
  showConfirm: (message: string, options?: { confirmText?: string; cancelText?: string }) => Promise<boolean>;
  closeConfirm: (result: boolean) => void;


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


export const AUTHENTICATED_SESSION = "cookie-session";

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
  } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
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
  "player.position",
  "player.hp",
  "player.mana",
  "player.combat_state",
  "session.kicked",
] as const;

type PlayerEventType = (typeof PLAYER_EVENT_TYPES)[number];

type PlayerEventDomain = "player" | "gu_inventory" | "item_inventory" | "recipes" | "combat_stats";

type PlayerRealtimeEvent = {
  type: PlayerEventType;
  revision?: number;
  state_revision?: number;
  changed_domains?: PlayerEventDomain[];
  data?: Partial<Player> & { id?: string; aperture_stress_level?: string };
  patch?: Partial<Player> & { id?: string; aperture_stress_level?: string };
};

const lastPlayerEventRevision = new Map<string, number>();

export function guSpecFor(guId?: string, rank?: number): GuSpec | undefined {
  if (!guId) return undefined;
  const state = useGameStore.getState();
  const byRank = state.guByRank[guId];
  if (byRank) {
    if (rank != null) return byRank[rank];
    const ranks = Object.keys(byRank).map(Number);
    const maxRank = ranks.length ? Math.max(...ranks) : 0;
    if (maxRank && byRank[maxRank]) return byRank[maxRank];
  }
  return rank == null ? state.guDictionary[guId] : undefined;
}

export function formatRecipeName(recipeId: string): string {
  const targetGuID = recipeId.replace("recipe_", "").replace(/_\d+_\d+$/, "").toUpperCase();
  const match = recipeId.match(/_(\d+)_(\d+)$/);
  const name = guSpecFor(targetGuID, match ? Number(match[1]) : undefined)?.name || recipeId;

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
    const equippedSet = new Set<string>(player.equipped_gus || []);
    const vital = player.gu_inventory?.find((g) => g.is_vital);
    if (vital && vital.status !== "DEAD") equippedSet.add(vital.instance_id);
    const guInventory = (player.gu_inventory || []).map((g) => {
      const isEquipped = !!g.equipped || !!g.is_vital || equippedSet.has(g.instance_id);
      if (isEquipped && g.status !== "DEAD") {
        equippedSet.add(g.instance_id);
      }
      return { ...g, equipped: isEquipped };
    });
    return {
      ...player,
      current_thought: player.current_thought ?? 100,
      max_thought: player.max_thought ?? 100,
      thought_regen_rate: player.thought_regen_rate ?? 1.0,
      gu_inventory: guInventory,
      equipped_gus: Array.from(equippedSet),
    };
  };

  const setPlayerFromServer = (player: Player | Partial<Player> | null, extraState: Partial<GameState> = {}, expectedRevision?: number) => {
    if (expectedRevision !== undefined && get().playerRevision !== expectedRevision) return false;
    if (!player) {
      set((state) => ({ ...extraState, player: null, playerStateRevision: null, manaSnapshot: null, playerRevision: state.playerRevision + 1 }));
      return true;
    }
    const currentPlayer = get().player;
    const incomingRevision = player.state_revision;
    const samePlayer = !currentPlayer || !player.id || currentPlayer.id === player.id;
    if (samePlayer && !shouldApplyPlayerRevision(get().playerStateRevision, incomingRevision)) {
      set(extraState);
      return false;
    }
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
      if (guComplete) {
        normalizedPlayer.is_sparse_gu_inventory = false;
      }
    }
    set((state) => ({
      ...extraState,
      player: normalizedPlayer,
      playerStateRevision: incomingRevision ?? (currentPlayer?.id === normalizedPlayer?.id ? get().playerStateRevision : null),
      playerRevision: state.playerRevision + 1,
      guInventoryLoaded: guComplete || (normalizedPlayer ? !normalizedPlayer.is_sparse_gu_inventory : false),
      manaSnapshot: normalizedPlayer ? createManaSnapshotFromPlayer(normalizedPlayer, get().guDictionary, get().constitutionDictionary, get().aperture?.time_rate) : null,
    }));
    if (normalizedPlayer) {
      get().subscribeAllBaoHoangMessages();
    }
    return true;
  };

  // applyInventoryFromResponse updates only the inventory-related player fields
  // (gu_inventory, materials_inventory) from a slim action API response.
  // All other player state changes (hp, mana, stones, combat, position, rank, etc.)
  // are handled by the WebSocket player.updated / delta events.
  const applyInventoryFromResponse = (data: ApiResponseData) => {
    const currentPlayer = get().player;
    if (!currentPlayer) return;
    const incomingRevision = typeof data.state_revision === "number" ? data.state_revision : undefined;
    if (!shouldApplyPlayerRevision(get().playerStateRevision, incomingRevision)) return;
    const updates: Partial<Player> = {};
    if (data.gu_inventory !== undefined) {
      // Re-use the same sparse/equipped logic from setPlayerFromServer
      const incomingInv: GuInstance[] = data.gu_inventory || [];
      updates.gu_inventory = incomingInv;
      updates.is_sparse_gu_inventory = false;
      if (Array.isArray(data.equipped_gus)) {
        updates.equipped_gus = data.equipped_gus;
      } else {
        const equippedSet = new Set<string>();
        incomingInv.forEach((g) => { if (g.equipped) equippedSet.add(g.instance_id); });
        updates.equipped_gus = Array.from(equippedSet);
      }
    }
    if (data.materials_inventory !== undefined) {
      updates.materials_inventory = data.materials_inventory ?? {};
    }
    if (data.primeval_stones !== undefined) {
      updates.primeval_stones = Number(data.primeval_stones);
    }
    if (data.immortal_stones !== undefined) {
      updates.immortal_stones = Number(data.immortal_stones);
    }
    if (data.human_qi !== undefined) {
      updates.human_qi = Number(data.human_qi);
    }
    if (data.hp !== undefined) {
      updates.hp = Number(data.hp);
    }
    if (data.current_thought !== undefined) {
      updates.current_thought = Number(data.current_thought);
    }
    if (data.soul_hp !== undefined) {
      updates.soul_hp = Number(data.soul_hp);
    }
    if (data.lifespan !== undefined) {
      updates.lifespan = Number(data.lifespan);
    }
    if (data.recipes !== undefined) {
      updates.unlocked_recipes = data.recipes;
    }
    if (Object.keys(updates).length > 0) {
      const merged = normalizePlayer({ ...currentPlayer, ...updates });
      if (merged) {
        set((state) => ({
          player: merged,
          playerStateRevision: incomingRevision ?? get().playerStateRevision,
          playerRevision: state.playerRevision + 1,
          guInventoryLoaded: updates.gu_inventory !== undefined ? true : get().guInventoryLoaded,
          manaSnapshot: createManaSnapshotFromPlayer(merged, get().guDictionary, get().constitutionDictionary, get().aperture?.time_rate),
        }));
      }
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

    // Replace Gu IDs (sorted by length descending so longer IDs match first)
    Object.keys(guDictionary || {})
      .sort((a, b) => b.length - a.length)
      .forEach((id) => {
        const name = guDictionary[id]?.name;
        if (name && formatted.includes(id)) {
          formatted = formatted.split(id).join(name);
        }
      });

    // Replace Item IDs (sorted by length descending so longer IDs match first)
    Object.keys(itemDictionary || {})
      .sort((a, b) => b.length - a.length)
      .forEach((id) => {
        const name = itemDictionary[id]?.name;
        if (name && formatted.includes(id)) {
          formatted = formatted.split(id).join(name);
        }
      });

    // Replace Monster IDs (sorted by length descending so longer IDs match first)
    Object.keys(monsterDictionary || {})
      .sort((a, b) => b.length - a.length)
      .forEach((id) => {
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

  type DictionaryPart = keyof GameState["dictionaryPartsLoaded"];

  const markDictionaryPartLoaded = (part: DictionaryPart) => {
    set((state) => {
      const dictionaryPartsLoaded = {
        ...state.dictionaryPartsLoaded,
        [part]: true,
      };
      return {
        dictionaryPartsLoaded,
        dictionaryLoaded: Object.values(dictionaryPartsLoaded).every(Boolean),
      };
    });
  };

  interface UnifiedDictionaryPayload {
    gu?: GuSpec[] | Record<string, GuSpec>;
    items?: Record<string, ItemSpec>;
    monsters?: Record<string, MonsterSpec>;
    constitutions?: EffectDictionaryEntry[];
    races?: EffectDictionaryEntry[];
    statuses?: EffectDictionaryEntry[];
  }

  const populateDictionaries = (data: UnifiedDictionaryPayload) => {
    if (!data) return;

    let guMap: Record<string, GuSpec> = {};
    let guByRank: Record<string, Record<number, GuSpec>> = {};
    if (data.gu) {
      const guArray = Array.isArray(data.gu) ? data.gu : Object.values(data.gu || {});
      for (const item of guArray as GuSpec[]) {
        if (!guByRank[item.id]) guByRank[item.id] = {};
        guByRank[item.id][item.rank] = item;
        if (!guMap[item.id] || item.rank > guMap[item.id].rank) {
          guMap[item.id] = item;
        }
      }
    } else {
      guMap = get().guDictionary;
      guByRank = get().guByRank;
    }

    const itemDictionary = data.items || get().itemDictionary;
    const monsterDictionary = data.monsters || get().monsterDictionary;

    let constitutionDictionary = get().constitutionDictionary;
    if (data.constitutions) {
      constitutionDictionary = {};
      for (const entry of (Array.isArray(data.constitutions) ? data.constitutions : [])) {
        constitutionDictionary[entry.key] = entry;
      }
    }

    let raceDictionary = get().raceDictionary;
    if (data.races) {
      raceDictionary = {};
      for (const entry of (Array.isArray(data.races) ? data.races : [])) {
        raceDictionary[entry.key] = entry;
      }
    }

    let statusDictionary = get().statusDictionary;
    if (data.statuses) {
      statusDictionary = {};
      for (const entry of (Array.isArray(data.statuses) ? data.statuses : [])) {
        statusDictionary[entry.key] = entry;
      }
    }

    set({
      guDictionary: guMap,
      guByRank,
      itemDictionary,
      monsterDictionary,
      constitutionDictionary,
      raceDictionary,
      statusDictionary,
      dictionaryLoaded: true,
      dictionaryPartsLoaded: {
        gus: true,
        items: true,
        monsters: true,
        constitutions: true,
        races: true,
        statuses: true,
      },
    });
  };

  const fetchDictionaryPart = async (part: DictionaryPart): Promise<void> => {
    if (get().dictionaryPartsLoaded[part]) return;

    await fetchWithDedupe(`fetchDictionary:${part}`, async () => {
      try {
        const response = await authFetch(`${BACKEND_URL}/api/dictionary/${part}`);
        if (!response.ok) throw new Error(`Failed to load ${part} dictionary`);
        const data = await response.json();

        if (part === "gus") {
          const guArray = Array.isArray(data) ? data : Object.values(data || {});
          const guMap: Record<string, GuSpec> = {};
          const guByRank: Record<string, Record<number, GuSpec>> = {};
          for (const item of guArray as GuSpec[]) {
            if (!guByRank[item.id]) guByRank[item.id] = {};
            guByRank[item.id][item.rank] = item;
            if (!guMap[item.id] || item.rank > guMap[item.id].rank) {
              guMap[item.id] = item;
            }
          }
          set({ guDictionary: guMap, guByRank });
        } else if (part === "items") {
          set({ itemDictionary: data || {} });
        } else if (part === "monsters") {
          set({ monsterDictionary: data || {} });
        } else if (part === "constitutions") {
          const constitutionDictionary: Record<string, EffectDictionaryEntry> = {};
          for (const entry of (Array.isArray(data) ? data : [])) {
            constitutionDictionary[entry.key] = entry;
          }
          set({ constitutionDictionary });
        } else if (part === "races") {
          const raceDictionary: Record<string, EffectDictionaryEntry> = {};
          for (const entry of (Array.isArray(data) ? data : [])) {
            raceDictionary[entry.key] = entry;
          }
          set({ raceDictionary });
        } else if (part === "statuses") {
          const statusDictionary: Record<string, EffectDictionaryEntry> = {};
          for (const entry of (Array.isArray(data) ? data : [])) {
            statusDictionary[entry.key] = entry;
          }
          set({ statusDictionary });
        }

        markDictionaryPartLoaded(part);
      } catch (caught: unknown) {
        const err = caught instanceof Error ? caught : new Error(String(caught));
        console.error(`${part} dictionary fetch failed:`, err.message);
      }
    });
  };

  return ({
  ...createNotificationsSlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  ...createMineSlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  ...createPartySlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  ...createChatSlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  ...createArenaSlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  ...createFeedbackSlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  ...createBettingSlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  ...createGachaSlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  ...createAchievementsSlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  ...createPetsSlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  ...createInstanceSlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  ...createMarketplaceSlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  ...createSectSlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  ...createApertureSlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  ...createTradeSlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  ...createInventorySlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  ...createCultivationSlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  ...createProfileSlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  ...createAuthSlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  ...createReferralsSlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  ...createMiscSlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  ...createCombatSlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  ...createTeleportSlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  ...createScoutingSlice(set as ApiResponseData, get as ApiResponseData, {} as ApiResponseData),
  applyInventoryFromResponse,
  player: null,
  playerStateRevision: null,
  manaSnapshot: null,
  token: null,
  isLoading: false,
  actionLoading: false,
  pendingActions: {},
  playerRevision: 0,
  beginAction: (key) => {
    const pendingActions = acquirePendingAction(get().pendingActions, key);
    if (!pendingActions) return false;
    set({ pendingActions, actionLoading: true });
    return true;
  },
  endAction: (key) => set((state) => {
    const pendingActions = releasePendingAction(state.pendingActions, key);
    if (pendingActions === state.pendingActions) return state;
    return { pendingActions, actionLoading: Object.keys(pendingActions).length > 0 };
  }),
  isActionPending: (key) => !!get().pendingActions[key],
  error: null,
  playerEventsConnected: false,
  playerEventsRetries: 0,
  reconciliationRevision: 0,
  reconciliationInFlight: false,
  logs: ["Chào mừng đạo hữu đến với Cổ Giới. Hãy tu luyện và sinh tồn."],
  guDictionary: {},
  guByRank: {},
  itemDictionary: {},
  monsterDictionary: {},
  constitutionDictionary: {},
  raceDictionary: {},
  statusDictionary: {},
  dictionaryLoaded: false,
  dictionaryLoading: false,
  dictionaryError: null,
  dictionaryPartsLoaded: {
    gus: false,
    items: false,
    monsters: false,
    constitutions: false,
    races: false,
    statuses: false,
  },
  currentView: "dashboard",

  fetchGuDictionary: () => fetchDictionaryPart("gus"),
  fetchItemDictionary: () => fetchDictionaryPart("items"),
  fetchMonsterDictionary: () => fetchDictionaryPart("monsters"),
  fetchConstitutionDictionary: () => fetchDictionaryPart("constitutions"),
  fetchRaceDictionary: () => fetchDictionaryPart("races"),
  fetchStatusDictionary: () => fetchDictionaryPart("statuses"),

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

  updateManaSnapshot: (player) => {
    const normalizedPlayer = normalizePlayer(player);
    set({
      manaSnapshot: normalizedPlayer ? createManaSnapshotFromPlayer(normalizedPlayer, get().guDictionary, get().constitutionDictionary, get().aperture?.time_rate) : null,
    });
  },
  setPlayerFromServer: (player, extraState, expectedRevision) => setPlayerFromServer(player, extraState, expectedRevision),
  getEstimatedMana: () => estimateManaFromSnapshot(get().manaSnapshot),

  subscribeEvents: () => {
    if (typeof window === "undefined" || !get().token) return;
    if (get().playerEventsConnected) return; // Already connected

    const token = get().token!;

    // Mark the shared subscription as claimed before awaiting the socket.
    // Login and page effects can request it in the same render cycle.
    set({ playerEventsConnected: true });

    // Acquire connection with reference counting
    const releaseConnection = wsManager.acquireConnection(token);

    try {
      wsManager.connect(token).then(() => {
        set({ playerEventsRetries: 0 });
      }).catch((err) => {
        console.warn("[WebSocket] Connection failed", err);
        set({ playerEventsConnected: false });
        releaseConnection();
      });

      const unsubscribeOpen = wsManager.onOpen(({ reconnected }) => {
        if (reconnected) void get().reconcilePlayer(true);
      });

      // Listen for player updates
      const unsubscribePlayer = wsManager.subscribe("player", (rawPayload: ApiResponseData) => {
        try {
          const payload = rawPayload as PlayerRealtimeEvent;
          const eventData = payload?.patch || payload?.data || rawPayload;
          const eventType = payload?.type;
          const eventVersion = typeof eventData?.version === "number" ? eventData.version : undefined;
          const incomingRevision = typeof (payload?.state_revision ?? eventData?.state_revision) === "number"
            ? (payload?.state_revision ?? eventData.state_revision)
            : undefined;
          if (eventType !== "session.kicked" && !shouldApplyPlayerRevision(get().playerStateRevision, incomingRevision)) {
            return;
          }
          const currentPlayer = get().player;

          if (eventType === "session.kicked") {
            const state = get();
            state.addToast("Tài khoản đã đăng nhập từ thiết bị khác. Bạn đã bị đăng xuất.", "error");
            state.logout();
            return;
          }

          if (currentPlayer && payload.revision !== undefined) {
            const previousRevision = lastPlayerEventRevision.get(currentPlayer.id) || 0;
            if (payload.revision <= previousRevision) return;
            lastPlayerEventRevision.set(currentPlayer.id, payload.revision);
          }

          const changedDomains = new Set(payload.changed_domains || []);
          const hasAuthoritativePatch = payload.patch !== undefined || payload.data !== undefined;
          const invalidatesStatus =
            changedDomains.has("combat_stats") ||
            (!hasAuthoritativePatch && changedDomains.has("player")) ||
            (changedDomains.has("item_inventory") && eventData?.materials_inventory === undefined);
          if (currentPlayer && invalidatesStatus) {
            void get().fetchPlayerStatus(currentPlayer.id, true);
          }
          if (changedDomains.has("gu_inventory") && eventData?.gu_inventory === undefined) {
            void get().fetchGuInventory();
          }
          if (
            changedDomains.has("recipes") &&
            eventData?.unlocked_recipes === undefined &&
            (eventData as ApiResponseData)?.recipes === undefined
          ) {
            void get().fetchRecipes(true);
          }

          // Handle delta events (efficient updates)
          if (eventType === "player.position") {
            if (!currentPlayer) return;
            const newX = eventData?.x ?? currentPlayer.x;
            const newY = eventData?.y ?? currentPlayer.y;
            if (newX !== currentPlayer.x || newY !== currentPlayer.y || eventVersion !== undefined) {
              set({
                player: { ...currentPlayer, x: newX, y: newY, ...(eventVersion !== undefined ? { version: eventVersion } : {}) },
                playerStateRevision: incomingRevision ?? get().playerStateRevision,
              });
            } else if (incomingRevision !== undefined) {
              set({ playerStateRevision: incomingRevision });
            }
            return;
          }

          if (eventType === "player.hp") {
            if (!currentPlayer) return;
            const newHp = eventData?.hp ?? currentPlayer.hp;
            const newMaxHp = eventData?.max_hp ?? currentPlayer.base_hp;
            set({
              playerStateRevision: incomingRevision ?? get().playerStateRevision,
              player: { ...currentPlayer, hp: newHp, base_hp: newMaxHp, ...(eventVersion !== undefined ? { version: eventVersion } : {}) },
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
            if (!currentPlayer) return;
            const newCurrentMana = eventData?.current_mana ?? currentPlayer.current_mana;
            const newMaxMana = eventData?.max_mana ?? currentPlayer.max_mana;
            set({
              playerStateRevision: incomingRevision ?? get().playerStateRevision,
              player: { ...currentPlayer, current_mana: newCurrentMana, max_mana: newMaxMana, ...(eventVersion !== undefined ? { version: eventVersion } : {}) },
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
            if (!currentPlayer) return;
            const nextCombat = eventData?.active_combat || null;
            const updatedPlayer = {
              ...currentPlayer,
              active_combat: nextCombat,
              ...(eventVersion !== undefined ? { version: eventVersion } : {}),
            };
            set((state) => ({
              player: updatedPlayer,
              playerStateRevision: incomingRevision ?? get().playerStateRevision,
              playerRevision: state.playerRevision + 1,
              ...(nextCombat === null || eventType === "combat.ended" ? { activeCombatReplay: null } : {}),
              manaSnapshot: updatedPlayer
                ? createManaSnapshotFromPlayer(
                    updatedPlayer,
                    get().guDictionary,
                    get().constitutionDictionary,
                    get().aperture?.time_rate
                  )
                : get().manaSnapshot,
            }));
            if (eventType === "combat.started") {
              get().addLog("Chiến đấu realtime đã bắt đầu.");
            } else if (eventType === "combat.ended") {
              get().addLog("Chiến đấu realtime đã kết thúc.");
            }
            return;
          }

          // Full snapshot fallback (for initial connection and other events)
          if (!eventData?.id && !payload.patch) return;

          if (!currentPlayer || (eventData.id && currentPlayer.id !== eventData.id)) return;

          const { active_combat, ...rest } = eventData;
          const nextCombat = active_combat !== undefined ? active_combat : currentPlayer.active_combat;
          const updatedPlayer = normalizePlayer({
            ...currentPlayer,
            ...rest,
            id: currentPlayer.id,
            active_combat: nextCombat,
          });
          set((state) => ({
            player: updatedPlayer,
            playerStateRevision: incomingRevision ?? get().playerStateRevision,
            playerRevision: state.playerRevision + 1,
            ...(nextCombat === null ? { activeCombatReplay: null } : {}),
            manaSnapshot: updatedPlayer
              ? createManaSnapshotFromPlayer(
                  updatedPlayer,
                  get().guDictionary,
                  get().constitutionDictionary,
                  get().aperture?.time_rate
                )
              : get().manaSnapshot,
          }));
        } catch (err) {
          console.error("Failed to process player websocket event", err);
        }
      });

      // Listen for trade updates
      const unsubscribeTrade = wsManager.subscribe("trade", (payload: ApiResponseData) => {
        try {
          const prevTrade = get().activeTrade;
          const newTrade = payload?.trade ?? null;
          set({ activeTrade: newTrade });
          // When trade just completed (had active trade, now null),
          // Refresh only the resources affected by the completed trade.
          if (prevTrade && !newTrade) {
            void Promise.all([
              get().refreshPlayerSummary(),
              get().fetchGuInventory(),
              get().refreshRecipeProgress(),
            ]);
          }
        } catch (err) {
          console.error("Failed to process trade event", err);
        }
      });

      // Listen for world position
      wsManager.subscribe("world:pos", (payload: ApiResponseData) => {
        try {
          if (payload && payload.id) {
            window.dispatchEvent(new CustomEvent("gu_world_pos", { detail: payload }));
          }
        } catch (err) {
          console.error("Failed to process world position event", err);
        }
      });

      // Listen for world cell
      wsManager.subscribe("world:cell", (payload: ApiResponseData) => {
        try {
          if (payload) {
            window.dispatchEvent(new CustomEvent("gu_world_cell", { detail: payload }));
          }
        } catch (err) {
          console.error("Failed to process world cell event", err);
        }
      });

      // Listen for world:cells — per-player cell snapshots pushed after movement
      wsManager.subscribe("world:cells", (payload: ApiResponseData) => {
        try {
          if (payload?.cells) {
            window.dispatchEvent(new CustomEvent("gu_world_cells", { detail: payload }));
          }
        } catch (err) {
          console.error("Failed to process world cells event", err);
        }
      });

      // Listen for chat:world
      wsManager.subscribe("chat:world", (payload: ApiResponseData) => {
        try {
          // Handle append-only updates (single new message)
          if (payload?.message && !Array.isArray(payload?.messages)) {
            set((state) => {
              const existing = state.chatMessages.world || [];
              const newMsg = payload.message;
              // Avoid duplicates
              if (existing.find((m: ApiResponseData) => m.id === newMsg.id)) {
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

      const currentPlayer = get().player;
      const hasConnectingHeaven = currentPlayer?.gu_inventory?.some(
        (gu) => gu.gu_id === "GU_CONNECTING_HEAVEN" && gu.status !== "DEAD"
      ) ?? false;
      const hasDivineSense = currentPlayer?.gu_inventory?.some(
        (gu) => gu.gu_id === "GU_DIVINE_SENSE" && gu.status !== "DEAD"
      ) ?? false;

      // Bảo Hoàng Thiên is restricted to players owning both required Gu.
      const unsubscribeGeneralChat = hasConnectingHeaven && hasDivineSense
        ? wsManager.subscribe("chat:general", (payload: ApiResponseData) => {
        try {
          // Handle append-only updates (single new message)
          if (payload?.message && !Array.isArray(payload?.messages)) {
            set((state) => {
              const existing = state.chatMessages.general || [];
              const newMsg = payload.message;
              if (existing.find((m: ApiResponseData) => m.id === newMsg.id)) {
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
      })
        : () => {};

      // Listen only to the current sect's channel. The backend publishes sect
      // chat on chat:sect:<sect_id>; a shared channel would leak messages.
      const sectStatus = get().mySectStatus;
      const sectId = sectStatus?.member?.sect_id ?? sectStatus?.sect?.id;
      const sectChannel = sectId ? `chat:sect:${sectId}` : null;
      const unsubscribeSectChat = sectChannel ? wsManager.subscribe(sectChannel, (payload: ApiResponseData) => {
        try {
          // Handle append-only updates (single new message)
          if (payload?.message && !Array.isArray(payload?.messages)) {
            set((state) => {
              const existing = state.chatMessages.sect || [];
              const newMsg = payload.message;
              if (existing.find((m: ApiResponseData) => m.id === newMsg.id)) {
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
      }) : () => {};

      // Listen for notification
      wsManager.subscribe("notification", (payload: ApiResponseData) => {
        try {
          get()._onNotificationEvent(payload);
        } catch (err) {
          console.error("Failed to process notification event", err);
        }
      });

      // Listen for death_arena events
      const unsubscribeDeathArena = wsManager.subscribe("death_arena", (payload: ApiResponseData) => {
        try {
          const eventType = payload?.type;
          const eventData = payload?.data;
          if (eventType === "match_updated" && eventData?.match_id) {
            set((state) => {
              const currentMatches = state.deathArenaBracket || [];
              const updated = currentMatches.map((m) =>
                m.id === eventData.match_id
                  ? { ...m, winner_team_id: eventData.winner_team_id, status: eventData.status }
                  : m
              );
              return { deathArenaBracket: updated };
            });
          } else if (eventType === "round_advanced" || eventType === "tournament_started") {
            get().fetchDeathArenaBracket();
            get().fetchDeathArenaCurrent();
          } else if (eventType === "tournament_finished") {
            set((state) => {
              if (!state.deathArenaTournament) return {};
              return {
                deathArenaTournament: {
                  ...state.deathArenaTournament,
                  status: "FINISHED",
                  champion_team_id: eventData?.champion_team_id || state.deathArenaTournament.champion_team_id,
                },
              };
            });
            get().fetchDeathArenaBracket();
          }
        } catch (err) {
          console.error("Failed to process death arena event", err);
        }
      });

      const unsubscribeCatalog = wsManager.subscribe("catalog", (payload: ApiResponseData) => {
        const event = payload?.data || payload;
        if (event?.type === "catalog.updated" && event?.catalog === "recipes") {
          // Keep the last valid recipes visible while revalidating the changed catalog.
          void get().fetchRecipes(true);
        }
      });

      // Store cleanup functions in a global variable
      (window as ApiResponseData).__gu_ws_cleanup = () => {
        unsubscribePlayer();
        unsubscribeOpen();
        unsubscribeTrade();
        unsubscribeGeneralChat();
        unsubscribeSectChat();
        unsubscribeDeathArena();
        unsubscribeCatalog();
        releaseConnection();
      };

    } catch (wsErr) {
      console.warn("[WebSocket] Exception starting WS:", wsErr);
      releaseConnection();
    }
  },

  unsubscribeEvents: () => {
    try {
      const cleanup = (window as ApiResponseData).__gu_ws_cleanup;
      if (cleanup) {
        cleanup();
        (window as ApiResponseData).__gu_ws_cleanup = null;
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

  subscribeBaoHoangMessages: (channel) => {
    get().subscribeEvents();
    get().setActiveChannel(channel as ApiResponseData);
  },

  unsubscribeBaoHoangMessages: () => {
    // Handled at main connection level
  },


  simulateDeposit: async (stones) => {
    if (!get().beginAction("player:wallet")) return false;
    set({ error: null });
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
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      set({ error: err.message });
      get().addToast(err.message || "Giao dịch giả lập thất bại.", "error");
      return false;
    } finally {
      get().endAction("player:wallet");
    }
  },

  // --- Sect Mines ---


  addLog: (message) => {
    const formatted = translateIDs(message);
    set((state) => ({
      logs: [formatted, ...state.logs].slice(0, 50),
    }));
  },




  fetchDictionary: async (forceFresh = false) => {
    let cachedEtag: string | undefined;

    // Trigger regions preload concurrently
    void get().fetchRegions();

    // 1. Instant Cache Hydration: Read from IndexedDB first
    if (!forceFresh) {
      try {
        const cached = await getDictionaryCache<UnifiedDictionaryPayload>("game_dictionary_unified");
        if (cached && cached.data) {
          cachedEtag = cached.etag;
          populateDictionaries(cached.data);
          set({ dictionaryLoaded: true, dictionaryError: null });
        }
      } catch (err) {
        console.warn("[Dictionary] Failed to read from IndexedDB:", err);
      }
    }

    const alreadyHasData = get().dictionaryLoaded;
    if (!alreadyHasData) {
      set({ dictionaryLoading: true, dictionaryError: null });
    }

    // 2. Network Fetch with Deduplication & Exponential Backoff Retry
    await fetchWithDedupe("fetchDictionary:unified", async () => {
      const maxRetries = 3;
      let lastError: Error | null = null;

      for (let attempt = 1; attempt <= maxRetries; attempt++) {
        try {
          const headers: Record<string, string> = {};
          if (cachedEtag && !forceFresh) {
            headers["If-None-Match"] = cachedEtag;
          }

          const response = await fetch(`${BACKEND_URL}/api/dictionary`, {
            headers,
          });

          if (response.status === 304) {
            // Server data has not changed, current cache is fresh
            set({ dictionaryLoaded: true, dictionaryLoading: false, dictionaryError: null });
            return;
          }

          if (!response.ok) {
            throw new Error(`Máy chủ phản hồi mã ${response.status}`);
          }

          const data = (await response.json()) as UnifiedDictionaryPayload;
          const etag = response.headers.get("ETag") || undefined;

          populateDictionaries(data);
          set({ dictionaryLoaded: true, dictionaryLoading: false, dictionaryError: null });

          // Save fresh data to IndexedDB
          void setDictionaryCache("game_dictionary_unified", data, etag);
          return;
        } catch (caught: unknown) {
          lastError = caught instanceof Error ? caught : new Error(String(caught));
          console.warn(`[Dictionary] Fetch attempt ${attempt}/${maxRetries} failed:`, lastError.message);
          if (attempt < maxRetries) {
            await new Promise((resolve) => setTimeout(resolve, attempt * 1000));
          }
        }
      }

      // If all network attempts failed:
      if (get().dictionaryLoaded) {
        // Fall back gracefully to the existing cached data
        set({ dictionaryLoading: false });
        console.info("[Dictionary] Using existing local cache after network failure.");
      } else {
        set({
          dictionaryLoading: false,
          dictionaryError: lastError?.message || "Không thể kết nối đến máy chủ để tải dữ liệu game. Vui lòng kiểm tra mạng và thử lại.",
        });
      }
    });
  },











  // Death Arena Actions


})});
