import { ToastContainer } from "@/components/ToastContainer";
import { create } from "zustand";

let confirmResolve: ((value: boolean) => void) | null = null;

export interface GuInstance {
  instance_id: string;
  gu_id: string;
  hunger: number;
  status: string; // "ACTIVE", "SLEEPING", "DEAD"
  equipped: boolean;
  is_vital: boolean;
  integration_time?: number;
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
  path_masteries?: Record<string, string>;
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
  tribulation_count?: number;
  hp: number;
  base_hp: number;
  body_mods: string[];
  current_region: string;
  dao_marks: Record<string, number>;
  path_masteries: Record<string, string>;
  x: number;
  y: number;
  explored_cells: string[];
  equipped_gus: string[];
  cleared_cells: Record<string, string>;
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
  temp_loot?: {
    gu_inventory: GuInstance[];
    materials_inventory: Record<string, number>;
    primeval_stones: number;
  };
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
  avatar_url?: string;
  combat_settings?: AutoCombatConfig;
  needs_onboarding?: boolean;
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
  gu_priorities: GuTactic[];
  km_priorities: KillerMoveTactic[];
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
  rank: number;
  sub_rank: number;
  aptitude_type?: string;
  lifespan?: number;
  essence_production_per_year?: number;
  captured_at: number;
  hp: number;
  base_hp: number;
  healing_stat: number;
}

const getManaRegenRatePerHour = (snapshot: ManaSnapshot): number => {
  if (snapshot.rank >= 6) {
    const subRank = Math.min(4, Math.max(1, snapshot.sub_rank || 1));
    const subRankFactor = 1.0 + (subRank - 1) * 0.15;
    const essenceProduction = snapshot.essence_production_per_year ?? 10;
    return (essenceProduction / 20.0) * subRankFactor;
  }

  switch (snapshot.aptitude_type) {
    case "Giáp":
      return 8.0;
    case "Ất":
      return 6.0;
    case "Bính":
      return 4.0;
    case "Đinh":
      return 2.0;
    default:
      return 2.0;
  }
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
  const multiplier = snapshot.time_speed_multiplier > 0 ? snapshot.time_speed_multiplier : 1;
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
  const multiplier = snapshot.time_speed_multiplier > 0 ? snapshot.time_speed_multiplier : 1;
  const regenHPPerHour = (snapshot.base_hp * 0.05) + (snapshot.healing_stat || 0);
  const regeneratedHP = ((elapsedSeconds * multiplier) / 3600) * regenHPPerHour;

  return Math.min(snapshot.base_hp, Math.max(0, snapshot.hp + regeneratedHP));
};

const calculateHealingStat = (player: Player, guDictionary: Record<string, GuSpec>): number => {
  let healing = 0;
  if (!player.gu_inventory) return healing;

  player.gu_inventory.forEach((gu) => {
    if (gu.equipped && gu.status === "ACTIVE") {
      const spec = guDictionary[gu.gu_id];
      if (spec && spec.effects) {
        if (spec.effects.active?.refine_mana_tier) healing += spec.effects.active?.refine_mana_tier * 10;
        if (spec.effects.active?.cleansing_power) healing += spec.effects.active?.cleansing_power * 15;
      }
    }
  });
  return healing;
};

const createManaSnapshotFromPlayer = (player: Player, guDictionary: Record<string, GuSpec>): ManaSnapshot => ({
  player_id: player.id,
  current_mana: player.current_mana,
  max_mana: player.max_mana,
  last_mana_update: player.last_mana_update,
  time_speed_multiplier: player.time_speed_multiplier || 1,
  rank: player.rank,
  sub_rank: player.sub_rank,
  aptitude_type: player.aptitude_type,
  lifespan: player.lifespan,
  captured_at: Date.now(),
  hp: player.hp,
  base_hp: player.base_hp,
  healing_stat: calculateHealingStat(player, guDictionary),
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
  tribulation_type: 'EARTH_CALAMITY' | 'HEAVENLY_TRIBULATION' | 'GRAND_TRIBULATION' | 'MYRIAD_TRIBULATION';
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
  created_at: string;
  updated_at: string;
}

export interface GuEffects {
  active?: Record<string, number>;
  support?: Record<string, number>;
  passive?: Record<string, number>;
  house?: Record<string, number>;
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
  rarity: string;
  is_immortal: boolean;
  primary_role: string;
  combat_roles: string[];
  combat_tags: string[];
  support_tags: string[];
  base_price?: number;
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
}

export interface PlayerInstanceSession {
  id: string;
  player_id: string;
  instance_id: string;
  current_stage: number;
  status: "ACTIVE" | "COMPLETED" | "FAILED";
  temp_inventory: any[];
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
}

export interface SectVaultGuItem {
  id: string;
  cost: number;
  quantity: number;
}

export interface Sect {
  id: string;
  name: string;
  description: string;
  region: string;
  vault_gu: SectVaultGuItem[];
  vault_materials: Record<string, number>;
  vault_recipes: string[];
  funds: number;
  level: number;
  require_approval?: boolean;
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
  current_rank: 'NGOAI_MON' | 'NOI_MON' | 'TINH_ANH' | 'CHAN_TRUYEN' | 'TRUONG_LAO' | 'CHUONG_MON';
  target_rank: 'NGOAI_MON' | 'NOI_MON' | 'TINH_ANH' | 'CHAN_TRUYEN' | 'TRUONG_LAO' | 'CHUONG_MON';
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
  member_rank: 'NGOAI_MON' | 'NOI_MON' | 'TINH_ANH' | 'CHAN_TRUYEN' | 'TRUONG_LAO' | 'CHUONG_MON';
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
  required_member_rank: 'NGOAI_MON' | 'NOI_MON' | 'TINH_ANH' | 'CHAN_TRUYEN' | 'TRUONG_LAO' | 'CHUONG_MON';
}

export interface SectLeaderboardEntry {
  username: string;
  avatar_url?: string;
  rank: 'NGOAI_MON' | 'NOI_MON' | 'TINH_ANH' | 'CHAN_TRUYEN' | 'TRUONG_LAO' | 'CHUONG_MON';
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
  member_rank: 'NGOAI_MON' | 'NOI_MON' | 'TINH_ANH' | 'CHAN_TRUYEN' | 'TRUONG_LAO' | 'CHUONG_MON';
  contribution_points: number;
  joined_at: string;
  player_rank: number;
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
  developer_response: string | null;
  rewarded: boolean;
  created_at: string;
  user_vote?: number | null;
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
  required_gus: Record<string, number>;
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

export interface Region {
  id: string;
  name: string;
  description: string;
  width: number;
  height: number;
  min_rank: number;
  max_rank: number;
  is_active: boolean;
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
  itemDictionary: Record<string, ItemSpec>;
  monsterDictionary: Record<string, MonsterSpec>;
  dictionaryLoaded: boolean;
  recipes: RefiningRecipe[];
  recipesLoaded: boolean;
  regions: Region[];
  regionsLoaded: boolean;
  fetchRegions: () => Promise<void>;
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
    otp?: string
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
  setPlayerFromServer: (player: Player | null) => void;
  getEstimatedMana: () => number | null;
  fetchPlayerStatus: (id: string) => Promise<void>;
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
  feedGuGroup: (guId: string) => Promise<void>;
  fetchDictionary: () => Promise<void>;
  fetchRecipes: () => Promise<void>;
  quests: any[];
  fetchQuests: () => Promise<void>;
  refineGu: (recipeId: string, selectedInstanceIds: string[]) => Promise<{ status: "success" | "failed"; message: string; new_gu?: GuInstance } | null>;
  deduceRecipe: (recipeId: string) => Promise<{ success: boolean; message: string; completion_rate: number } | null>;
  breakWall: (manaSpend: number) => Promise<{ success: boolean; message: string } | null>;
  useRelicGu: (relicKey: string) => Promise<{ success: boolean; message: string } | null>;
  ascension: () => Promise<{ success: boolean; message: string } | null>;
  triggerTribulation: () => Promise<{ success: boolean; message: string } | null>;
  completeQuest: (questId: string) => Promise<{ success: boolean; message: string } | null>;
  redeemGiftCode: (code: string) => Promise<{ success: boolean; message: string } | null>;
  gachaPools: any[];
  fetchGachaPools: () => Promise<void>;
  openGacha: (poolId: string) => Promise<{ success: boolean; message: string; reward?: any } | null>;
  openGacha10: (poolId: string) => Promise<{ success: boolean; message: string; rewards?: any[] } | null>;
  rechargeWithStones: (stones: number) => Promise<{ success: boolean; message: string } | null>;
  clearStress: () => Promise<{ success: boolean; message: string } | null>;
  showChangelog: boolean;
  setShowChangelog: (v: boolean) => void;
  achievements: any[];
  earnedAchievements: any[];
  selectedTitle: string;
  selectedFrame: string;
  fetchAchievements: () => Promise<void>;
  claimAchievement: (achievementId: string) => Promise<{ success: boolean; message: string } | null>;
  selectTitleFrame: (title: string, frame: string) => Promise<{ success: boolean; message: string } | null>;
  sacrificeNPC: (npcId: string) => Promise<{ success: boolean; message: string } | null>;
  exploreNode: (nodeId: string) => Promise<{ player: Player; log: string; combat_triggered: boolean } | null>;
  executeCombatTurn: (equippedGuIds: string[], isCharge?: boolean, targetId?: string, targetX?: number, targetY?: number) => Promise<{ player: Player; logs: string[]; enemy: CombatState | null; victory: boolean; defeat: boolean } | null>;
  executeAutoCombat: (loadout: any) => Promise<{ logs: any[]; winner: string; combatants?: any[] } | null>;
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
  } | null;
  fetchReferrals: () => Promise<void>;
  simulateDeposit: (stones: number) => Promise<boolean>;
  addLog: (message: string) => void;
  currentView: string;
  setView: (view: string) => void;
  retreat: () => Promise<void>;
  fleeCombat: () => Promise<void>;
  combatSettings: AutoCombatConfig | null;
  combatSettingsLoading: boolean;
  fetchCombatSettings: () => Promise<void>;
  saveCombatSettings: (settings: AutoCombatConfig) => Promise<boolean>;
  equipGu: (instanceId: string, equip: boolean) => Promise<void>;
  updateGuStatus: (instanceId: string, status: string) => Promise<void>;
  setVitalGu: (instanceId: string) => Promise<void>;
  useGu: (instanceId: string) => Promise<void>;
  useItem: (itemId: string) => Promise<void>;
  movePlayer: (x: number, y: number, attackTargetId?: string) => Promise<{ player: Player; log: string; combat_triggered: boolean } | null>;
  fetchGrid: () => Promise<{ x: number; y: number; vision_radius: number; cells: any[]; explored_cells: string[]; stats: any; players_on_map?: any[] } | null>;
  dropGu: (instanceId: string) => Promise<boolean>;
  buyShopItem: (itemId: string, quantity: number) => Promise<boolean>;
  sellShopItem: (itemId: string, quantity: number, itemType: "Gu" | "Material") => Promise<boolean>;
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
  updateTradeOffer: (tradeId: string, offer: { primeval_stones: number; immortal_stones: number; gus: string[]; materials: Record<string, number> }) => Promise<boolean>;
  lockTrade: (tradeId: string) => Promise<boolean>;
  confirmTrade: (tradeId: string) => Promise<boolean>;

  // Party State & Actions
  party: { id: string; leader_id: string; members: { id: string; username: string; avatar_url?: string }[] } | null;
  createParty: () => Promise<boolean>;
  fetchPartyStatus: () => Promise<void>;
  invitePlayer: (username: string) => Promise<boolean>;
  leaveParty: () => Promise<boolean>;

  // Arena State & Actions
  arenaLeaderboard: any[];
  arenaDailyChallenges: number;
  arenaShopItems: any[];
  fetchArenaLeaderboard: () => Promise<void>;
  fetchArenaShop: () => Promise<void>;
  challengeArena: (targetPlayerId: string) => Promise<boolean>;
  buyArenaItem: (itemId: string) => Promise<boolean>;

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

  // Mine State & Actions
  mineSlots: MineSlot[];
  fetchMineSlots: () => Promise<void>;
  occupyMineSlot: (slotId: number) => Promise<boolean>;
  claimMineRewards: (slotId: number) => Promise<boolean>;
  abandonMineSlot: (slotId: number) => Promise<boolean>;
  attackMineSlot: (slotId: number) => Promise<boolean>;

  // Toast System State & Actions
  toasts: ToastMessage[];
  addToast: (message: string, type?: "success" | "error" | "info", duration?: number) => void;
  removeToast: (id: string) => void;

  // Confirm Dialog
  confirmDialog: { open: boolean; message: string; confirmText: string; cancelText: string };
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
  placeApertureNode: (nodeType: string, x: number, y: number) => Promise<boolean>;
  collectResourceNode: (nodeId: string) => Promise<boolean>;
  uninstallApertureNode: (nodeId: string) => Promise<boolean>;
  deleteApertureNode: (nodeId: string) => Promise<boolean>;
  setApertureTimeRate: (rate: number) => Promise<boolean>;
  setApertureLabor: (race: string, intensity: string) => Promise<boolean>;
  repairApertureNode: (nodeId: string) => Promise<boolean>;
  fetchActiveTribulation: () => Promise<void>;
  attackTribulationEnemy: (eventId: string, enemyId: string, guId: string) => Promise<boolean>;
  resolveTribulation: (eventId: string) => Promise<boolean>;

  // Soul foundation actions
  useGutsGu: () => Promise<boolean>;
  refineSoul: () => Promise<boolean>;
  calmSoul: () => Promise<boolean>;

  // Instance actions
  activeInstanceSession: PlayerInstanceSession | null;
  activeInstanceTemplate: InstanceTemplate | null;
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
  leaveSect: () => Promise<boolean>;
  fetchMySectStatus: () => Promise<void>;
  promoteSectRank: () => Promise<boolean>;
  donateSectStones: (amount: number) => Promise<boolean>;
  donateSectMaterials: (materialId: string, amount: number) => Promise<boolean>;
  donateSectGu: (guInstanceId: string) => Promise<boolean>;
  buyFromSectVault: (itemType: string, itemId: string) => Promise<boolean>;
  fetchSectMissions: () => Promise<void>;
  acceptSectMission: (missionId: string) => Promise<boolean>;
  completeSectMission: (missionId: string) => Promise<boolean>;
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
  }) => Promise<boolean>;
  stockSectVault: (itemType: 'gu' | 'material', itemId: string, quantity: number) => Promise<boolean>;
  upgradeSect: () => Promise<boolean>;
  toggleSectApproval: (requireApproval: boolean) => Promise<boolean>;
  fetchJoinRequests: () => Promise<void>;
  actionJoinRequest: (requestId: string, action: 'approve' | 'reject') => Promise<boolean>;
  fetchPromotionRequests: () => Promise<void>;
  actionPromotionRequest: (requestId: string, action: "approve" | "reject") => Promise<boolean>;
  fetchSectMines: () => Promise<void>;
  attackSectMine: (slotId: number) => Promise<boolean>;
  claimSectMineRewards: (slotId: number) => Promise<boolean>;

  // Feedback State & Actions
  feedbacks: Feedback[];
  fetchFeedbacks: (category?: string, status?: string) => Promise<void>;
  submitFeedback: (title: string, description: string, category: string) => Promise<boolean>;
  voteFeedback: (feedbackId: string, voteType: number) => Promise<boolean>;
  updateFeedbackStatus: (feedbackId: string, status: string, developerResponse: string) => Promise<boolean>;
}

export interface MineSlot {
  id: number;
  rarity: string;
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
  // Attach cleanup to signal
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

const authFetch = (input: RequestInfo | URL, init: RequestInit = {}) => {
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
  // Add abort signal with 30s timeout, keyed by URL string
  const key = typeof input === "string" ? input : input.toString();
  const signal = getAbortSignal(key);
  return fetch(input, {
    ...init,
    headers,
    credentials: "include",
    signal,
  });
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

let playerEventsSource: EventSource | null = null;
let tradeEventsSource: EventSource | null = null;
let worldMessagesSource: EventSource | null = null;
let generalMessagesSource: EventSource | null = null;
let sectMessagesSource: EventSource | null = null;

export const useGameStore = create<GameState>((set, get) => {
  const normalizePlayer = (player: Player | null): Player | null => {
    if (!player) return null;
    const guInventory = (player.gu_inventory || []).map((g) => ({ ...g, equipped: !!g.equipped || !!g.is_vital }));
    const equippedSet = new Set<string>(player.equipped_gus || []);
    const vital = guInventory.find((g) => g.is_vital);
    if (vital && vital.status !== "DEAD") equippedSet.add(vital.instance_id);
    return { ...player, gu_inventory: guInventory, equipped_gus: Array.from(equippedSet) };
  };

  const setPlayerFromServer = (player: Player | null, extraState: Partial<GameState> = {}) => {
    const normalizedPlayer = normalizePlayer(player);
    set({
      ...extraState,
      player: normalizedPlayer,
      manaSnapshot: normalizedPlayer ? createManaSnapshotFromPlayer(normalizedPlayer, get().guDictionary) : null,
    });
    if (normalizedPlayer) {
      get().subscribeAllBaoHoangMessages();
    }
  };

  const translateIDs = (msg: string): string => {
    if (!msg) return "";
    let formatted = msg;
    const { guDictionary, itemDictionary, monsterDictionary } = get();

    // Replace Gu IDs
    Object.keys(guDictionary || {}).forEach((id) => {
      const name = guDictionary[id]?.name;
      if (name) {
        formatted = formatted.split(id).join(name);
      }
    });

    // Replace Item IDs
    Object.keys(itemDictionary || {}).forEach((id) => {
      const name = itemDictionary[id]?.name;
      if (name) {
        formatted = formatted.split(id).join(name);
      }
    });

    // Replace Monster IDs
    Object.keys(monsterDictionary || {}).forEach((id) => {
      const name = monsterDictionary[id]?.name;
      if (name) {
        formatted = formatted.split(id).join(name);
      }
    });

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
  arenaShopItems: [],
  playerEventsConnected: false,
  playerEventsRetries: 0,
  logs: ["Chào mừng đạo hữu đến với Cổ Giới. Hãy tu luyện và sinh tồn."],
  referralsData: null,
  guDictionary: {},
  itemDictionary: {},
  monsterDictionary: {},
  dictionaryLoaded: false,
  recipes: [],
  recipesLoaded: false,
  quests: [],
  gachaPools: [],
  achievements: [],
  earnedAchievements: [],
  selectedTitle: '',
  selectedFrame: '',
  regions: [],
  regionsLoaded: false,
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
  arenaLeaderboard: [],
  inbox: [],
  baoHoangMessages: [],
  showChangelog: false,
  activeChannel: "world",
  chatOpen: false,
  chatMessages: { world: [], general: [], sect: [] },
  unreadChannels: { world: false, general: false, sect: false },
  mineSlots: [],
  aperture: null,
  daoMarks: [],
  apertureNodes: [],
  variantHumans: [],
  activeTribulation: null,
  activeInstanceSession: null,
  activeInstanceTemplate: null,
  sects: [],
  mySectStatus: null,
  sectMissions: [],
  sectMembers: [],
  sectJoinRequests: [],
  sectPromotionRequests: [],
  sectMines: [],
  feedbacks: [],
  setView: (view) => set({ currentView: view }),
  setShowChangelog: (v) => set({ showChangelog: v }),

  toasts: [],
  addToast: (message, type = "info", duration = 4000) => {
    const formatted = translateIDs(message);
    const id = Math.random().toString(36).substring(2, 9);
    set((state) => ({
      toasts: [...state.toasts, { id, message: formatted, type, duration }],
    }));
    setTimeout(() => {
      get().removeToast(id);
    }, duration);
  },
  removeToast: (id) => {
    set((state) => ({
      toasts: state.toasts.filter((t) => t.id !== id),
    }));
  },

  confirmDialog: { open: false, message: "", confirmText: "Xác nhận", cancelText: "Hủy" },
  showConfirm: async (message, options) => {
    return new Promise((resolve) => {
      confirmResolve = resolve;
      set({
        confirmDialog: {
          open: true,
          message,
          confirmText: options?.confirmText || "Xác nhận",
          cancelText: options?.cancelText || "Hủy",
        },
      });
    });
  },
  closeConfirm: (result) => {
    if (confirmResolve) {
      confirmResolve(result);
      confirmResolve = null;
    }
    set({ confirmDialog: { open: false, message: "", confirmText: "Xác nhận", cancelText: "Hủy" } });
  },

  login: async (email, password) => {
    set({ isLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
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
      get().subscribePlayerEvents();
      get().subscribeTradeEvents();
      get().addLog(`Đồng chí ${data.player.username} đã quy vị!`);
      set({ showChangelog: true });
      return true;
    } catch (err: any) {
      set({ error: err.message });
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
      return false;
    } finally {
      set({ isLoading: false });
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
    otp
  ) => {
    set({ isLoading: true, error: null });
    try {
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
      get().subscribePlayerEvents();
      get().subscribeTradeEvents();
      get().addLog(`Cổ sư ${data.player.username} khai mở Không Khiếu! Tư chất: ${data.player.aptitude_type} (${data.player.aptitude_percent}%)`);
      return true;
    } catch (err: any) {
      set({ error: err.message });
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
    set({ player: null, manaSnapshot: null, token: null, recipes: [], recipesLoaded: true, quests: [] });
    localStorage.removeItem("gu_auth_token");
    localStorage.removeItem("gu_player_id");
    get().addLog("Đã rời khỏi thế giới Cổ sư.");
  },

  clearError: () => set({ error: null }),
  updateManaSnapshot: (player) => {
    const normalizedPlayer = normalizePlayer(player);
    set({
      manaSnapshot: normalizedPlayer ? createManaSnapshotFromPlayer(normalizedPlayer, get().guDictionary) : null,
    });
  },
  setPlayerFromServer: (player) => setPlayerFromServer(player),
  getEstimatedMana: () => estimateManaFromSnapshot(get().manaSnapshot),

  subscribePlayerEvents: () => {
    if (typeof window === "undefined" || !get().token) return;

    if (playerEventsSource) {
      playerEventsSource.close();
      playerEventsSource = null;
    }

    const source = new EventSource(`${BACKEND_URL}/api/player/events?token=${encodeURIComponent(get().token!)}`);
    playerEventsSource = source;

    const mergePlayerEvent = (event: MessageEvent<string>) => {
      try {
        const payload: PlayerRealtimeEvent = JSON.parse(event.data);
        if (!payload?.data?.id) return;

        const currentPlayer = get().player;
        if (!currentPlayer || currentPlayer.id !== payload.data.id) return;

        // Only merge scalar fields from SSE to avoid overwriting arrays with stale data
        const { active_combat, ...rest } = payload.data;
        set({
          player: normalizePlayer({
            ...currentPlayer,
            ...rest,
            ...(active_combat !== undefined ? { active_combat } : {}),
          }),
        });

        if (payload.type === "combat.started") {
          get().addLog("Chiến đấu realtime đã bắt đầu.");
        } else if (payload.type === "combat.ended") {
          get().addLog("Chiến đấu realtime đã kết thúc.");
        }
      } catch (err) {
        console.error("Failed to process player realtime event", err);
      }
    };

    PLAYER_EVENT_TYPES.forEach((eventType) => {
      source.addEventListener(eventType, mergePlayerEvent as EventListener);
    });

    source.onopen = () => {
      set({ playerEventsConnected: true, playerEventsRetries: 0 });
    };
    source.onerror = () => {
      set({ playerEventsConnected: false });
      source.close();
      // Auto-reconnect with exponential backoff
      const retries = get().playerEventsRetries || 0;
      const delay = Math.min(1000 * Math.pow(2, retries), 30000);
      set({ playerEventsRetries: retries + 1 });
      setTimeout(() => {
        if (get().token) get().subscribePlayerEvents();
      }, delay);
    };
  },

  unsubscribePlayerEvents: () => {
    if (playerEventsSource) {
      playerEventsSource.close();
      playerEventsSource = null;
    }
    set({ playerEventsConnected: false });
  },


  subscribeTradeEvents: () => {
    if (typeof window === "undefined" || !get().token) return;

    if (tradeEventsSource) {
      tradeEventsSource.close();
      tradeEventsSource = null;
    }

    const source = new EventSource(`${BACKEND_URL}/api/trade/events?token=${encodeURIComponent(get().token!)}`);
    tradeEventsSource = source;

    source.addEventListener("trade.updated", (event) => {
      try {
        const payload = JSON.parse((event as MessageEvent<string>).data);
        set({ activeTrade: payload.trade ?? null });
      } catch (err) {
        console.error("Failed to process trade event", err);
      }
    });
  },

  unsubscribeTradeEvents: () => {
    if (tradeEventsSource) {
      tradeEventsSource.close();
      tradeEventsSource = null;
    }
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
    get().subscribeAllBaoHoangMessages();
    get().setActiveChannel(channel as any);
  },

  unsubscribeBaoHoangMessages: () => {
    get().unsubscribeAllBaoHoangMessages();
  },

  subscribeAllBaoHoangMessages: () => {
    if (typeof window === "undefined" || !get().token) return;

    // 1. World Channel
    if (!worldMessagesSource) {
      const source = new EventSource(`${BACKEND_URL}/api/chat/bao-hoang/events?channel=world&token=${encodeURIComponent(get().token!)}`);
      worldMessagesSource = source;
      source.addEventListener("chat.messages", (event) => {
        try {
          const payload = JSON.parse((event as MessageEvent<string>).data);
          const messages = Array.isArray(payload.messages) ? payload.messages : [];
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
    }

    // 2. General Channel (Bảo Hoàng Thiên)
    const player = get().player;
    const hasConnectingHeaven = player?.gu_inventory?.some(g => g.gu_id === "GU_CONNECTING_HEAVEN" && g.status !== "DEAD") ?? false;
    const hasDivineSense = player?.gu_inventory?.some(g => g.gu_id === "GU_DIVINE_SENSE" && g.status !== "DEAD") ?? false;
    if (hasConnectingHeaven && hasDivineSense) {
      if (!generalMessagesSource) {
        const source = new EventSource(`${BACKEND_URL}/api/chat/bao-hoang/events?channel=general&token=${encodeURIComponent(get().token!)}`);
        generalMessagesSource = source;
        source.addEventListener("chat.messages", (event) => {
          try {
            const payload = JSON.parse((event as MessageEvent<string>).data);
            const messages = Array.isArray(payload.messages) ? payload.messages : [];
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
      }
    } else {
      if (generalMessagesSource) {
        generalMessagesSource.close();
        generalMessagesSource = null;
      }
    }

    // 3. Sect Channel – always try to connect; backend validates membership
    if (!sectMessagesSource) {
      const source = new EventSource(`${BACKEND_URL}/api/chat/bao-hoang/events?channel=sect&token=${encodeURIComponent(get().token!)}`);
      sectMessagesSource = source;
      source.addEventListener("chat.messages", (event) => {
        try {
          const payload = JSON.parse((event as MessageEvent<string>).data);
          const messages = Array.isArray(payload.messages) ? payload.messages : [];
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
      source.addEventListener("error", () => {
        // Backend rejected (not in sect) – clean up silently
        source.close();
        sectMessagesSource = null;
      });
    }
  },

  unsubscribeAllBaoHoangMessages: () => {
    if (worldMessagesSource) {
      worldMessagesSource.close();
      worldMessagesSource = null;
    }
    if (generalMessagesSource) {
      generalMessagesSource.close();
      generalMessagesSource = null;
    }
    if (sectMessagesSource) {
      sectMessagesSource.close();
      sectMessagesSource = null;
    }
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

  attackSectMine: async (slotId: number) => {
    const { token, actionLoading, logout, addToast } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/mines/attack`, {
        method: "POST",
        body: JSON.stringify({ mine_id: slotId }),
      });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Lỗi khi tấn công Lãnh địa");
      addToast(data.message || "Bắt đầu tấn công Lãnh địa Tông môn!", "success");
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

  claimSectMineRewards: async (slotId: number) => {
    const { token, actionLoading, logout, addToast } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/mines/claim`, {
        method: "POST",
        body: JSON.stringify({ mine_id: slotId }),
      });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Lỗi khi thu hoạch Lãnh địa");
      addToast(data.message || "Thu hoạch Lãnh địa thành công!", "success");
      get().fetchMySectStatus();
      get().fetchSectMines();
      return true;
    } catch (err: any) {
      addToast(err.message || "Lỗi khi thu hoạch Lãnh địa", "error");
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
    }

    try {
      const response = await authFetch(`${BACKEND_URL}/api/auth/me`);
      if (response.status === 401 || response.status === 403) {
        set({ player: null, manaSnapshot: null, token: null });
        return;
      }
      if (!response.ok) {
        throw new Error("Failed to restore session");
      }

      const player: Player = await response.json();
      const localToken = (typeof window !== "undefined" ? localStorage.getItem("gu_auth_token") : null) || AUTHENTICATED_SESSION;
      setPlayerFromServer(player, { token: localToken });
      if (typeof window !== "undefined") {
        localStorage.setItem("gu_player_id", player.id);
      }
      void get().fetchMySectStatus();
      void get().fetchRecipes();
      get().subscribePlayerEvents();
      get().subscribeTradeEvents();
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

      if (response.status === 401 || response.status === 403) {
        logout();
        return;
      }

      if (!response.ok) {
        throw new Error("Failed to fetch status");
      }

      const player: Player = await response.json();
      const currentPlayer = get().player;
      if (currentPlayer && currentPlayer.id === id) {
        setPlayerFromServer(player);
      }
    } catch (err: any) {
      console.error("Status fetch failed:", err.message);
    }
  },

  fetchPlayerProfile: async (username) => {
    const { token, logout } = get();
    if (!token) return;

    set({ profileLoading: true });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/profile/${encodeURIComponent(username)}`);

      if (response.status === 401 || response.status === 403) {
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

      if (response.status === 401 || response.status === 403) {
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

      if (response.status === 401 || response.status === 403) {
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

  feedGuGroup: async (guId) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/inventory/feed-group`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ gu_id: guId }),
      });

      if (response.status === 401 || response.status === 403) {
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

  fetchDictionary: async () => {
    try {
      const response = await authFetch(`${BACKEND_URL}/api/dictionary`);
      if (!response.ok) {
        throw new Error("Failed to load global metadata");
      }
      const data = await response.json();
      set({
        guDictionary: data.gu || {},
        itemDictionary: data.items || {},
        monsterDictionary: data.monsters || {},
        dictionaryLoaded: true,
      });
    } catch (err: any) {
      console.error("Dictionary sync error:", err.message);
      set({ dictionaryLoaded: true }); // Don't block the app on dictionary failure
    }
  },

  fetchRecipes: async () => {
    try {
      const response = await authFetch(`${BACKEND_URL}/api/recipes`);
      if (response.status === 401 || response.status === 403) {
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
      if (response.status === 401 || response.status === 403) {
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
      if (response.status === 401 || response.status === 403) {
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

      if (response.status === 401 || response.status === 403) {
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

      if (response.status === 401 || response.status === 403) {
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

      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Trùng kích thất bại."); }

      setPlayerFromServer(data.player);
      get().addLog(data.message || "Trùng kích hoàn tất.");
      return { success: data.success, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
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

      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Nạp nguyên thạch thất bại.");

      setPlayerFromServer(data.player);
      get().addLog(data.message || "Nạp nguyên thạch thành công.");
      return { success: true, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Không thể dùng Cổ Xá Lợi."); }
      setPlayerFromServer(data.player);
      get().addLog(data.message || "Đã sử dụng Cổ Xá Lợi.");
      return { success: true, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Thăng tiên thất bại."); }
      setPlayerFromServer(data.player);
      get().addLog(data.message || "Thăng tiên hoàn tất.");
      return { success: data.success, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Không thể nhận thưởng nhiệm vụ."); }
      setPlayerFromServer(data.player);
      get().addLog(data.message || "Hoàn thành nhiệm vụ.");
      return { success: true, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Nhận thưởng nhiệm vụ lỗi: ${err.message}`);
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Mã quà không hợp lệ."); }
      setPlayerFromServer(data.player);
      get().addLog(data.message || "Đổi mã quà thành công!");
      return { success: true, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Đổi mã quà lỗi: ${err.message}`);
      return { success: false, message: err.message };
    } finally { set({ actionLoading: false }); }
  },

  fetchAchievements: async () => {
    const { token } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/achievements`);
      if (response.status === 401 || response.status === 403) return;
      if (response.ok) {
        const data = await response.json();
        set({ achievements: data.achievements || [], earnedAchievements: data.earned || [], selectedTitle: data.selected_title || '', selectedFrame: data.selected_frame || '' });
      }
    } catch (err: any) {
      console.error("Achievements fetch error:", err.message);
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Nhận thưởng thất bại."); }
      setPlayerFromServer(data.player);
      get().fetchAchievements();
      get().addLog(data.message || "Nhận thưởng thành tựu thành công!");
      return { success: true, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
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
      if (response.status === 401 || response.status === 403) return null;
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Cập nhật thất bại.");
      set({ selectedTitle: title, selectedFrame: frame });
      get().addLog(data.message || "Đã cập nhật danh hiệu và frame.");
      return { success: true, message: data.message };
    } catch (err: any) {
      console.error("Select title/frame error:", err.message);
      return { success: false, message: err.message };
    }
  },

  fetchGachaPools: async () => {
    const { token } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/gacha/pools`);
      if (response.status === 401 || response.status === 403) return;
      if (response.ok) {
        const data = await response.json();
        set({ gachaPools: data.pools || [] });
      }
    } catch (err: any) {
      console.error("Gacha pools error:", err.message);
    }
  },

  openGacha: async (poolId) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/gacha/open`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ pool_id: poolId }),
      });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Mở cổ từ thất bại."); }
      setPlayerFromServer(data.player);
      get().addLog(data.message || "Mở cổ từ thành công!");
      return { success: true, message: data.message, reward: data.reward };
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Mở cổ từ lỗi: ${err.message}`);
      return { success: false, message: err.message };
    } finally { set({ actionLoading: false }); }
  },

  openGacha10: async (poolId) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return null;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/gacha/open10`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ pool_id: poolId }),
      });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Mở 10 cổ từ thất bại."); }
      setPlayerFromServer(data.player);
      get().addLog(data.message || "Mở 10 cổ từ thành công!");
      return { success: true, message: data.message, rewards: data.rewards };
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Mở 10 cổ từ lỗi: ${err.message}`);
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) { throw new Error(data.error || "Không thể kích hoạt thiên kiếp."); }
      setPlayerFromServer(data.player);
      get().addLog(data.message || "Thiên kiếp giáng lâm.");
      return { success: true, message: data.message };
    } catch (err: any) {
      set({ error: err.message });
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

      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Khám phá thất bại.");
      }

      setPlayerFromServer(data.player);
      get().addLog(data.log);
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

      if (response.status === 401 || response.status === 403) {
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

      if (response.status === 401 || response.status === 403) {
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
      if (response.status === 401 || response.status === 403) {
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
      if (response.status === 401 || response.status === 403) {
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
      if (response.status === 401 || response.status === 403) {
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
      if (response.status === 401 || response.status === 403) {
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
      if (response.status === 401 || response.status === 403) {
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

      if (response.status === 401 || response.status === 403) {
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
      set({ error: err.message });
      get().addLog(`Tấn công thất bại: ${err.message}`);
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
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ loadout }),
      });

      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Bắt đầu mô phỏng thất bại.");
      }

      // We just return the logs and winner, we don't need to update player immediately
      // The frontend handles the replay animation
      return data;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Mô phỏng thất bại: ${err.message}`);
      return null;
    } finally {
      set({ actionLoading: false });
    }
  },

  retreat: async () => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/explore/retreat/${player.id}`, {
        method: "POST",
      });

      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Rút lui thất bại.");
      }

      setPlayerFromServer(data.player, { currentView: "dashboard" });
      get().addLog(data.log || "Đạo hữu đã rút lui an toàn.");
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Rút lui thất bại: ${err.message}`);
    } finally {
      set({ actionLoading: false });
    }
  },

  fleeCombat: async () => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return;

    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/explore/flee/${player.id}`, {
        method: "POST",
      });

      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Bỏ chạy thất bại.");
      }

      setPlayerFromServer(data.player, { currentView: "explore" });
      get().addLog(data.log || "Đạo hữu đã bỏ chạy an toàn.");
    } catch (err: any) {
      set({ error: err.message });
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
      if (response.status === 401 || response.status === 403) { logout(); return; }
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
      if (response.status === 401 || response.status === 403) { logout(); return null; }
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
      if (response.status === 401 || response.status === 403) { logout(); return; }
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
      if (response.status === 401 || response.status === 403) { logout(); return; }
      const data = await response.json();
      set({ marketplaceListings: Array.isArray(data) ? data : [] });
    } catch (err) {
      console.error("market listings failed", err);
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Đăng bán thất bại");
      get().addLog(data.message || "Đăng bán thành công");
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Đặt giá thất bại");
      get().addLog(data.message || "Đặt giá thành công");
      await get().initializeSession();
      await get().fetchMarketplaceListings();
      return true;
    } catch (err: any) { set({ error: err.message }); get().addLog(`Đặt giá thất bại: ${err.message}`); return false; } finally { set({ actionLoading: false }); }
  },

  cancelMarketplaceListing: async (listingId) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/bao-hoang/cancel`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ listing_id: listingId }) });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hủy listing thất bại");
      get().addLog(data.message || "Đã hủy");
      await get().initializeSession();
      await get().fetchMarketplaceListings();
      return true;
    } catch (err: any) { set({ error: err.message }); get().addLog(`Hủy listing thất bại: ${err.message}`); return false; } finally { set({ actionLoading: false }); }
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Mua hàng thất bại");
      get().addLog(data.message || "Mua hàng thành công");
      await get().initializeSession();
      await get().fetchMarketplaceListings();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Mua hàng thất bại: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchActiveTrade: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/trade/active`);
      if (response.status === 401 || response.status === 403) { logout(); return; }
      if (!response.ok) return;
      const data = await response.json();
      set({ activeTrade: data.trade });
    } catch (err) {
      console.error("Failed to fetch active trade", err);
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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

  buyShopItem: async (itemId, quantity) => {
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
        }),
      });

      if (response.status === 401 || response.status === 403) {
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
      get().addLog(`Mua hàng thất bại: ${err.message}`);
      return false;
    } finally {
      set({ actionLoading: false });
    }
  },


  sellShopItem: async (itemId, quantity, itemType) => {
    const { player, token, actionLoading, logout } = get();
    if (!player || !token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/shop/sell`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ player_id: player.id, item_id: itemId, quantity, item_type: itemType }),
      });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Bán hàng thất bại.");
      setPlayerFromServer(data.player);
      get().addLog(data.message || "Bán hàng thành công!");
      return true;
    } catch (err: any) {
      set({ error: err.message });
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Vứt Cổ thất bại.");
      setPlayerFromServer(data.player);
      get().addLog(data.message || "Đã vứt Cổ trùng.");
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Vứt Cổ thất bại: ${err.message}`);
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

      if (response.status === 401 || response.status === 403) {
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

      if (response.status === 401 || response.status === 403) {
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

      if (response.status === 401 || response.status === 403) {
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

      if (response.status === 401 || response.status === 403) {
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

      if (response.status === 401 || response.status === 403) {
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

      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn hoặc không có quyền.");
      }

      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Di chuyển thất bại.");
      }

      setPlayerFromServer(data.player);
      get().addLog(data.log || "Di chuyển thành công.");
      return data;
    } catch (err: any) {
      set({ error: err.message });
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

      if (response.status === 401 || response.status === 403) {
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Tạo tổ đội thất bại");
      set({ party: data.party });
      get().addLog(data.message || "Đã lập tổ đội thành công!");
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Lỗi tạo party: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchPartyStatus: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/party/status`);
      if (response.status === 401 || response.status === 403) { logout(); return; }
      const data = await response.json();
      set({ party: data.party });
    } catch (err) {
      console.error("fetch party status failed", err);
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Không thể mời");
      set({ party: data.party });
      get().addLog(data.message || `Đã mời người chơi ${username} vào tổ đội.`);
      return true;
    } catch (err: any) {
      set({ error: err.message });
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Rời tổ đội thất bại");
      set({ party: null });
      get().addLog(data.message || "Bạn đã rời khỏi tổ đội.");
      await get().initializeSession();
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Rời tổ đội thất bại: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchArenaLeaderboard: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/arena/leaderboard`);
      if (response.status === 401 || response.status === 403) { logout(); return; }
      const data = await response.json();
      set({ 
        arenaLeaderboard: Array.isArray(data.leaderboard) ? data.leaderboard : [],
        arenaDailyChallenges: data.daily_challenges || 0 
      });
    } catch (err) {
      console.error("fetch arena leaderboard failed", err);
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
    }
  },

  buyArenaItem: async (itemId) => {
    const { token, actionLoading } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/arena/shop/buy`, {
        method: "POST",
        body: JSON.stringify({ item_id: itemId }),
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Khiêu chiến thất bại");
      setPlayerFromServer(data.player, { currentView: "dashboard" }); // Force update player state which triggers combat screen
      get().addLog(data.message || "Đã thách đấu đối thủ!");
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Khiêu chiến võ đài lỗi: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Gửi thư thất bại");
      get().addLog(data.message || "Đã gửi thư truyền tin!");
      await get().initializeSession(); // update stone count
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Truyền tin lỗi: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchInbox: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/chat/mail/inbox`);
      if (response.status === 401 || response.status === 403) { logout(); return; }
      const data = await response.json();
      set({ inbox: Array.isArray(data.inbox) ? data.inbox : [] });
    } catch (err) {
      console.error("fetch inbox failed", err);
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Gửi tin nhắn thất bại");
      get().addLog(data.message || "Đã phát truyền tin nhắn!");
      await get().initializeSession(); // refresh player state (mana, inventory, etc)
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addLog(`Lỗi gửi tin nhắn: ${err.message}`);
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchBaoHoangMessages: async (channel) => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/chat/bao-hoang/messages?channel=${channel}`);
      if (response.status === 401 || response.status === 403) { return; }
      const data = await response.json();
      set({ baoHoangMessages: Array.isArray(data.messages) ? data.messages : [] });
    } catch (err) {
      console.error("fetch chat messages failed", err);
    }
  },

  fetchMineSlots: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/mine/slots`);
      if (response.status === 401 || response.status === 403) { logout(); return; }
      const data = await response.json();
      if (response.ok) {
        set({ mineSlots: Array.isArray(data) ? data : [] });
      }
    } catch (err) {
      console.error("fetch mine slots failed", err);
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn hoặc không có quyền."); }
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

  fetchAperture: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture`);
      if (response.status === 401 || response.status === 403) {
        logout();
        return;
      }
      if (!response.ok) {
        set({ aperture: null });
        return;
      }
      const data = await response.json();
      set({ aperture: data });
    } catch (err) {
      console.error("Failed to fetch aperture:", err);
      set({ aperture: null });
    }
  },

  fetchDaoMarks: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/dao-marks`);
      if (response.status === 401 || response.status === 403) {
        logout();
        return;
      }
      if (response.ok) {
        const data = await response.json();
        set({ daoMarks: data });
      }
    } catch (err) {
      console.error("Failed to fetch Dao Marks:", err);
    }
  },

  fetchApertureNodes: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes`);
      if (response.status === 401 || response.status === 403) {
        logout();
        return;
      }
      if (response.ok) {
        const data = await response.json();
        set({ apertureNodes: data });
      }
    } catch (err) {
      console.error("Failed to fetch aperture nodes:", err);
    }
  },

  placeApertureNode: async (nodeType: string, x: number, y: number) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ node_type: nodeType, x, y }),
      });
      if (response.status === 401 || response.status === 403) {
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
      if (response.status === 401 || response.status === 403) {
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

  uninstallApertureNode: async (nodeId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/nodes/${nodeId}/uninstall`, {
        method: "POST",
      });
      if (response.status === 401 || response.status === 403) {
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
      if (response.status === 401 || response.status === 403) {
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
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/immortal-aperture/variant-humans`);
      if (response.status === 401 || response.status === 403) {
        logout();
        return;
      }
      if (response.ok) {
        const data = await response.json();
        set({ variantHumans: data });
      }
    } catch (err) {
      console.error("Failed to fetch variant humans:", err);
    }
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
      if (response.status === 401 || response.status === 403) {
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
      if (response.status === 401 || response.status === 403) {
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
      if (response.status === 401 || response.status === 403) {
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

  fetchActiveTribulation: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/tribulations/active`);
      if (response.status === 401 || response.status === 403) {
        logout();
        return;
      }
      if (response.ok) {
        const data = await response.json();
        set({ activeTribulation: data });
      }
    } catch (err) {
      console.error("Failed to fetch active tribulation:", err);
    }
  },

  attackTribulationEnemy: async (eventId: string, enemyId: string, guId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/tribulations/${eventId}/attack`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ enemy_id: enemyId, gu_id: guId }),
      });
      if (response.status === 401 || response.status === 403) {
        logout();
        throw new Error("Phiên làm việc hết hạn.");
      }
      const data = await response.json();
      if (!response.ok) {
        throw new Error(data.error || "Tấn công thất bại");
      }
      get().addToast(data.message || "Tấn công kẻ địch tai kiếp!", "success");
      get().addLog(data.message || "Gây sát thương lên kẻ địch tai kiếp.");
      await get().initializeSession();
      await get().fetchActiveTribulation();
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

  resolveTribulation: async (eventId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/tribulations/${eventId}/resolve`, {
        method: "POST",
      });
      if (response.status === 401 || response.status === 403) {
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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

  calmSoul: async () => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/soul/calm`, { method: "POST" });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "An hồn thất bại.");
      setPlayerFromServer(data.player);
      get().addToast(data.message || "An hồn thành công!", "success");
      get().addLog(data.message || "Đã an ổn hồn phách bằng nước hồ An Hồn.");
      return true;
    } catch (err: any) {
      set({ error: err.message });
      get().addToast(err.message, "error");
      return false;
    } finally { set({ actionLoading: false }); }
  },

  fetchActiveInstance: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/instance/active`);
      if (response.status === 401 || response.status === 403) { logout(); return; }
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Không thể tiến vào phó bản.");
      set({
        activeInstanceSession: data.session,
        activeInstanceTemplate: data.template,
      });
      get().addToast(data.message || "Tiến vào phó bản thành công!", "success");
      get().addLog(data.message || `Tiến vào phó bản: ${instanceId}`);
      await get().initializeSession();
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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
      if (response.status === 401 || response.status === 403) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({ sects: data || [] });
      }
    } catch (err) {
      console.error("Failed to fetch sects:", err);
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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

  fetchMySectStatus: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/me`);
      if (response.status === 401 || response.status === 403) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({ mySectStatus: data });
      }
    } catch (err) {
      console.error("Failed to fetch my sect status:", err);
    }
  },

  promoteSectRank: async () => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/promote`, {
        method: "POST",
      });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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

  buyFromSectVault: async (itemType: string, itemId: string) => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/vault/buy`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ item_type: itemType, item_id: itemId }),
      });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Trao đổi vật phẩm thất bại.");
      get().addToast(data.message || "Trao đổi vật phẩm thành công!", "success");
      get().addLog(data.message || `Đổi ${itemType}: ${itemId} từ bảo khố.`);
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
      if (response.status === 401 || response.status === 403) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({ sectMissions: data || [] });
      }
    } catch (err) {
      console.error("Failed to fetch sect missions:", err);
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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
      if (response.status === 401 || response.status === 403) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({ sectMembers: data || [] });
      }
    } catch (err) {
      console.error("Failed to fetch sect members:", err);
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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
        }),
      });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Phát hành nhiệm vụ thất bại.");
      get().addToast(data.message || "Đã phát hành nhiệm vụ môn phái!", "success");
      get().addLog(data.message || `Phát hành nhiệm vụ: ${payload.title}`);
      await get().fetchSectMissions();
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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

  upgradeSect: async () => {
    const { token, actionLoading, logout } = get();
    if (!token || actionLoading) return false;
    set({ actionLoading: true, error: null });
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/upgrade`, {
        method: "POST",
      });
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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

  fetchJoinRequests: async () => {
    const { token, logout } = get();
    if (!token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/sects/join-requests`);
      if (response.status === 401 || response.status === 403) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({ sectJoinRequests: data || [] });
      }
    } catch (err) {
      console.error("Failed to fetch join requests:", err);
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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
      if (response.status === 401 || response.status === 403) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({ sectPromotionRequests: data || [] });
      }
    } catch (err) {
      console.error("Failed to fetch promotion requests:", err);
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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
      if (response.status === 401 || response.status === 403) { logout(); return; }
      if (response.ok) {
        const data = await response.json();
        set({ feedbacks: data || [] });
      }
    } catch (err) {
      console.error("Failed to fetch feedbacks:", err);
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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
      if (response.status === 401 || response.status === 403) { logout(); throw new Error("Phiên làm việc hết hạn."); }
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
})});
