import type { GuDictionaryItem, ItemSpec } from "./gu-dict.ts";
import type { EffectOperation } from "./effect-blocks.ts";
import { translatePath } from "./translate.ts";

export function removeAccents(str: string): string {
  return str
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/đ/g, "d")
    .replace(/Đ/g, "D");
}

export const STATUS_NAMES_VI: Record<string, string> = {
  BLEED: "Chảy Máu",
  POISON: "Trúng Độc",
  BURN: "Thiêu Đốt",
  STUN: "Choáng",
  FREEZE: "Đóng Băng",
  SILENCE: "Câm Lặng",
  GU_SEAL: "Khóa Cổ",
  HASTE: "Tăng Tốc",
  SLOW: "Làm Chậm",
  SLOW1: "Làm Chậm",
  SLOW_DAMAGE: "Sát Thương Chậm",
  WEAK: "Suy Yếu",
  REGEN: "Hồi Phục",
  THORNS: "Gai Phản Đòn",
  TAUNT: "Khiêu Khích",
  INVINCIBLE: "Bất Tử",
  EVASION: "Né Tránh",
  CONFUSE: "Hỗn Loạn",
  CONFUSION: "Hỗn Loạn",
  FEAR: "Hoảng Sợ",
  STEALTH: "Tàng Hình",
  SLEEP: "Ngủ Say",
  BLIND: "Mù Lòa",
  CHARM: "Mê Hoặc",
  BIND: "Trói Buộc",
  ROOT: "Bất Động",
  DECAY: "Hủ Bại",
  FORGET_EGO: "Lãng Quên",
  ARMOR_BREAK: "Phá Giáp",
  DAMAGE_REDUCTION_UP: "Giảm Thương",
  REVIVE_LOCK: "Khóa Hồi Sinh",
  ESSENCE_LEAK: "Thất Thoát Chân Nguyên",
  SOUL_WOUND: "Tổn Thương Hồn",
  THOUGHT_CHAOS: "Tạp Niệm",
  DAO_CONFLICT: "Xung Đột Đạo",
  BACKLASH: "Phản Phệ",
};

export const STAT_NAMES_VI: Record<string, string> = {
  ATTACK: "Công",
  DEFENSE: "Thủ",
  HP: "Sinh Lực",
  SPEED: "Tốc Độ",
  CRIT_RATE: "Bạo Kích",
  TENACITY: "Kiên Cường",
  CC_RESIST: "Kháng Khống Chế",
  LUCK: "Vận Khí",
  ALL: "Toàn Chỉ Số",
  DAMAGE_DEALT: "Sát Thương Gây Ra",
  MOVEMENT: "Di Chuyển",
  STORAGE: "Sức Chứa",
  DAO_MARK: "Đạo Ngân",
};

export const UNIQUE_NAMES_VI: Record<string, string> = {
  REBORN: "Hồi Sinh",
  BYPASS_SHIELD: "Xuyên Khiên",
  SUMMON_PHANTOM: "Triệu Hồi Hư Ảnh",
  SUMMON_WOLF_FRONTLINE: "Triệu Hồi Sói",
  SWORD_DRAGON_TRANSFORM: "Kiếm Long Biến",
  FULL_SOUL_HEAL: "Đầy Hồn Phục",
  SEIZE_INITIATIVE: "Đoạt Tiên Cơ",
  FIRST_TURN_EVADE: "Né Tuyệt Đối Lượt Đầu",
  STEALTH: "Tàng Hình",
  MANA_REGEN: "Hồi Chân Nguyên",
  MAX_MANA: "Chân Nguyên Tối Đa",
  HEALING_RECEIVED: "Hiệu Quả Hồi Máu",
  HP_REGEN: "Hồi Sinh Lực",
  REFINE_SUCCESS: "Tỷ Lệ Luyện Cổ",
  LUCK: "Vận Khí",
  DOUBLE_DAMAGE_DEALT: "Nhân Đôi Sát Thương",
};

export interface EffectCategoryOption {
  id: string;
  label: string;
  icon: string;
  keywords: string[];
}

export const EFFECT_CATEGORIES: EffectCategoryOption[] = [
  { id: "all", label: "Tất cả hiệu ứng", icon: "✨", keywords: [] },
  { id: "stun", label: "Choáng", icon: "⚡", keywords: ["stun", "choang"] },
  { id: "lifesteal", label: "Hút máu", icon: "🩸", keywords: ["lifesteal", "hut mau", "blood_drain"] },
  { id: "poison", label: "Trúng độc", icon: "☠️", keywords: ["poison", "trung doc", "doc"] },
  { id: "burn", label: "Thiêu đốt", icon: "🔥", keywords: ["burn", "thieu dot"] },
  { id: "bleed", label: "Chảy máu", icon: "🗡️", keywords: ["bleed", "chay mau"] },
  { id: "freeze", label: "Đóng băng", icon: "❄️", keywords: ["freeze", "dong bang"] },
  { id: "silence", label: "Câm lặng / Khóa", icon: "🤫", keywords: ["silence", "cam lang", "gu_seal", "khoa co", "bind", "troi buoc"] },
  { id: "shield", label: "Khiên / Bảo hộ", icon: "🛡️", keywords: ["shield", "khien", "barrier", "bao ho", "damage_reduction", "giam thuong"] },
  { id: "heal", label: "Hồi máu (HP)", icon: "💚", keywords: ["heal", "hoi mau", "hoi hp", "regen", "hoi phuc"] },
  { id: "mana", label: "Chân nguyên", icon: "💧", keywords: ["mana", "chan nguyen", "mana_regen"] },
  { id: "stealth", label: "Tàng hình", icon: "👻", keywords: ["stealth", "tang hinh"] },
  { id: "invincible", label: "Bất tử / Hồi sinh", icon: "🌟", keywords: ["invincible", "bat tu", "reborn", "hoi sinh"] },
  { id: "direct_hp", label: "Sát thương trực tiếp", icon: "💥", keywords: ["direct_hp", "xuyen khien", "sat thuong truc tiep"] },
  { id: "crit_evade", label: "Bạo kích / Né", icon: "🎯", keywords: ["crit", "bao kich", "evasion", "ne tranh"] },
  { id: "stat_buff", label: "Tăng chỉ số (Công/Thủ)", icon: "📊", keywords: ["stat_mod", "tang cong", "tang thu", "attack", "defense"] },
];

/**
 * Extracts a normalized searchable string covering:
 * - Basic info (name, id, description, path, gu_type)
 * - Effects operations, statuses, stats, triggers, unique abilities
 * - Combat & support tags
 * - Food item names
 */
export function extractGuSearchableText(gu: GuDictionaryItem, itemMap?: Map<string, ItemSpec>): string {
  const parts: string[] = [];

  // 1. Basic text
  if (gu.name) parts.push(gu.name);
  if (gu.id) parts.push(gu.id);
  if (gu.description) parts.push(gu.description);
  if (gu.gu_type) parts.push(gu.gu_type);

  // 2. Paths
  if (gu.path && gu.path.length > 0) {
    for (const p of gu.path) {
      parts.push(p);
      parts.push(translatePath(p));
    }
  }

  // 3. Tags & Roles
  if (gu.primary_role) parts.push(gu.primary_role);
  if (gu.combat_roles) parts.push(...gu.combat_roles);
  if (gu.combat_tags) parts.push(...gu.combat_tags);
  if (gu.support_tags) parts.push(...gu.support_tags);

  // 4. Effects
  const blocks = gu.effects?.blocks || [];
  for (const block of blocks) {
    if (block.trigger) parts.push(block.trigger);
    for (const op of block.operations || []) {
      if (op.op) parts.push(op.op);
      if (op.sub_type) parts.push(op.sub_type);
      if (op.status_id) {
        parts.push(op.status_id);
        const stName = STATUS_NAMES_VI[op.status_id.toUpperCase()];
        if (stName) parts.push(stName);
      }
      if (op.stat) {
        parts.push(op.stat);
        const sName = STAT_NAMES_VI[op.stat.toUpperCase()];
        if (sName) parts.push(sName);
      }
      if (op.code) {
        parts.push(op.code);
        const uName = UNIQUE_NAMES_VI[op.code.toUpperCase()];
        if (uName) parts.push(uName);
      }
      if (op.path) {
        parts.push(op.path);
        parts.push(translatePath(op.path));
      }
    }
  }

  // 5. Food types and names
  if (gu.food_types) {
    for (const food of gu.food_types) {
      parts.push(food);
      if (itemMap) {
        const item = itemMap.get(food);
        if (item?.name) parts.push(item.name);
      }
    }
  }

  const combined = parts.join(" ").toLowerCase();
  return `${combined} ${removeAccents(combined)}`;
}

/**
 * Checks whether a Gu item matches a specific effect category filter.
 */
export function matchesGuEffect(gu: GuDictionaryItem, effectId: string): boolean {
  if (!effectId || effectId === "all") return true;

  const category = EFFECT_CATEGORIES.find((c) => c.id === effectId);
  if (!category) return true;

  const blocks = gu.effects?.blocks || [];
  const tags = [...(gu.combat_tags || []), ...(gu.combat_roles || []), ...(gu.support_tags || [])].map((t) =>
    t.toLowerCase()
  );

  for (const block of blocks) {
    for (const op of block.operations || []) {
      const opType = (op.op || "").toUpperCase();
      const subType = (op.sub_type || "").toUpperCase();
      const statusId = (op.status_id || "").toUpperCase();
      const code = (op.code || "").toUpperCase();
      const stat = (op.stat || "").toUpperCase();

      switch (effectId) {
        case "stun":
          if (statusId === "STUN") return true;
          break;
        case "lifesteal":
          if (subType === "LIFESTEAL" || opType === "LIFESTEAL") return true;
          break;
        case "poison":
          if (statusId === "POISON") return true;
          break;
        case "burn":
          if (statusId === "BURN") return true;
          break;
        case "bleed":
          if (statusId === "BLEED") return true;
          break;
        case "freeze":
          if (statusId === "FREEZE") return true;
          break;
        case "silence":
          if (statusId === "SILENCE" || statusId === "GU_SEAL" || statusId === "BIND" || statusId === "ROOT") return true;
          break;
        case "shield":
          if (opType === "SHIELD" || opType === "BARRIER" || statusId === "DAMAGE_REDUCTION_UP") return true;
          break;
        case "heal":
          if ((opType === "HEAL" && subType !== "MANA") || statusId === "REGEN" || code === "HP_REGEN") return true;
          break;
        case "mana":
          if (subType === "MANA" || code === "MANA_REGEN" || code === "MAX_MANA") return true;
          break;
        case "stealth":
          if (statusId === "STEALTH" || code === "STEALTH") return true;
          break;
        case "invincible":
          if (statusId === "INVINCIBLE" || code === "REBORN" || opType === "BARRIER") return true;
          break;
        case "direct_hp":
          if (opType === "DIRECT_HP" || subType === "MAX_HP_PERCENT" || code === "BYPASS_SHIELD") return true;
          break;
        case "crit_evade":
          if (stat === "CRIT_RATE" || statusId === "EVASION" || code === "FIRST_TURN_EVADE") return true;
          break;
        case "stat_buff":
          if (opType === "STAT_MOD" || code === "STAT_MODS") return true;
          break;
      }
    }
  }

  // Also check combat tags
  for (const kw of category.keywords) {
    if (tags.some((t) => t.includes(kw))) return true;
  }

  return false;
}

/**
 * Extract up to maxCount prominent effect summary badges for rendering in cards.
 */
export function extractGuEffectBadges(
  gu: GuDictionaryItem,
  maxCount: number = 3
): { text: string; icon: string; color: string }[] {
  const badges: { text: string; icon: string; color: string }[] = [];
  const blocks = gu.effects?.blocks || [];

  for (const block of blocks) {
    for (const op of block.operations || []) {
      if (badges.length >= maxCount) break;

      const opType = (op.op || "").toUpperCase();
      const subType = (op.sub_type || "").toUpperCase();
      const statusId = (op.status_id || "").toUpperCase();
      const code = (op.code || "").toUpperCase();
      const stat = (op.stat || "").toUpperCase();

      if (statusId) {
        const sName = STATUS_NAMES_VI[statusId] || statusId;
        const turns = op.turns ? ` (${op.turns} lượt)` : "";
        badges.push({
          text: `${sName}${turns}`,
          icon: statusId === "STUN" ? "⚡" : statusId === "POISON" ? "☠️" : statusId === "BURN" ? "🔥" : "✨",
          color: "bg-indigo-950/30 text-indigo-300 border-indigo-500/20",
        });
        continue;
      }

      if (opType === "HEAL") {
        if (subType === "LIFESTEAL") {
          badges.push({
            text: `Hút máu ${Math.round((op.value ?? 0) * 100)}%`,
            icon: "🩸",
            color: "bg-rose-950/30 text-rose-300 border-rose-500/20",
          });
        } else if (subType === "MANA") {
          badges.push({
            text: `Hồi ${op.value ?? 0} Chân Nguyên`,
            icon: "💧",
            color: "bg-cyan-950/30 text-cyan-300 border-cyan-500/20",
          });
        } else {
          badges.push({
            text: `Hồi +${op.value ?? 0} HP`,
            icon: "💚",
            color: "bg-emerald-950/30 text-emerald-300 border-emerald-500/20",
          });
        }
        continue;
      }

      if (opType === "SHIELD") {
        badges.push({
          text: `Khiên +${op.value ?? 0}`,
          icon: "🛡️",
          color: "bg-sky-950/30 text-sky-300 border-sky-500/20",
        });
        continue;
      }

      if (opType === "DIRECT_HP") {
        badges.push({
          text: subType === "MAX_HP_PERCENT" ? `Xuyên khiên ${Math.round((op.value ?? 0) * 100)}% HP` : "Sát thương trực tiếp",
          icon: "⚡",
          color: "bg-purple-950/30 text-purple-300 border-purple-500/20",
        });
        continue;
      }

      if (code) {
        const uName = UNIQUE_NAMES_VI[code] || code;
        badges.push({
          text: uName,
          icon: code === "REBORN" ? "🌟" : code === "BYPASS_SHIELD" ? "🗡️" : "✨",
          color: "bg-amber-950/30 text-amber-300 border-amber-500/20",
        });
        continue;
      }

      if (opType === "STAT_MOD") {
        const sName = STAT_NAMES_VI[stat] || stat;
        const val = op.value ?? 0;
        const isMult = subType === "MULTIPLY_PERCENT";
        const valStr = isMult ? `${val >= 0 ? "+" : ""}${Math.round(val * 100)}%` : `${val >= 0 ? "+" : ""}${val}`;
        badges.push({
          text: `${valStr} ${sName}`,
          icon: "📊",
          color: val >= 0 ? "bg-amber-950/30 text-amber-300 border-amber-500/20" : "bg-rose-950/30 text-rose-300 border-rose-500/20",
        });
        continue;
      }

      if (opType === "DAMAGE" && op.value) {
        const pathStr = op.path ? ` [${translatePath(op.path)}]` : "";
        badges.push({
          text: subType === "MULTIPLIER" ? `ST ×${op.value}${pathStr}` : `ST ${op.value}${pathStr}`,
          icon: "⚔️",
          color: "bg-rose-950/30 text-rose-300 border-rose-500/20",
        });
        continue;
      }
    }
  }

  return badges;
}
