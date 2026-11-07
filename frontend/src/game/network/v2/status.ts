const STATUS_ID_TO_CODE: Record<string, number> = {
  BURN: 1, BLEED: 2, BIND: 3, POISON: 4, STUN: 5, CONFUSE: 6, FEAR: 7,
  EVASION: 8, FREEZE: 9, BLIND: 10, SILENCE: 11, CHARM: 12, SLEEP: 13,
  INVINCIBLE: 14, SLOW: 15, HASTE: 16, SLOW1: 17, FLYING: 18, FLIGHT: 18,
  STEALTH: 19, INVISIBILITY: 19, REVIVE_LOCK: 20, WEAK: 21, FORM_WOLF_PARTIAL: 22,
};
const STATUS_CODE_TO_ID = new Map<number, string>([
  [1, "BURN"], [2, "BLEED"], [3, "BIND"], [4, "POISON"], [5, "STUN"], [6, "CONFUSE"], [7, "FEAR"],
  [8, "EVASION"], [9, "FREEZE"], [10, "BLIND"], [11, "SILENCE"], [12, "CHARM"], [13, "SLEEP"],
  [14, "INVINCIBLE"], [15, "SLOW"], [16, "HASTE"], [17, "SLOW1"], [18, "FLYING"], [19, "STEALTH"],
  [20, "REVIVE_LOCK"], [21, "WEAK"], [22, "FORM_WOLF_PARTIAL"],
  [23, "SLOW_DAMAGE"], [24, "THORNS"], [25, "REGEN"], [26, "ATK_UP"], [27, "DEF_UP"],
  [28, "TAUNT"], [29, "ARMOR_BREAK"], [30, "DAMAGE_REDUCTION_UP"], [31, "LAST_STAND"],
  [32, "FORGET_EGO"], [33, "FORGET_REALITY"], [34, "PHI_HANH"], [35, "ESSENCE_LEAK"],
  [36, "GU_SEAL"], [37, "SOUL_WOUND"], [38, "THOUGHT_CHAOS"], [39, "DAO_CONFLICT"],
  [40, "BACKLASH"], [41, "FORM_TURTLE_PARTIAL"], [42, "ROOT"], [43, "PETRIFY"],
]);

export function statusCode(id: string): number | undefined { return STATUS_ID_TO_CODE[id]; }
export function statusId(code: number): string | undefined { return STATUS_CODE_TO_ID.get(code); }
