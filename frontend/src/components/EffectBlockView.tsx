import React from "react";
import type { ComposableEffectBlock, AtomicOperation, TargetScope } from "@/lib/gu-dict";
import { translatePath } from "@/lib/translate";

const TRIGGER_CONFIG: Record<string, { label: string; color: string; icon: string }> = {
  ALWAYS_ON: { label: "Nội Tại", color: "bg-purple-950/60 text-purple-300 border-purple-800/50", icon: "💎" },
  PASSIVE: { label: "Nội Tại", color: "bg-purple-950/60 text-purple-300 border-purple-800/50", icon: "💎" },
  COMBAT_PASSIVE: { label: "Nội Tại Chiến Đấu", color: "bg-purple-950/60 text-purple-300 border-purple-800/50", icon: "⚡" },
  ON_ACTION: { label: "Khi Ra Chiêu", color: "bg-amber-950/60 text-amber-300 border-amber-800/50", icon: "⚔️" },
  ON_HIT: { label: "Khi Đánh Trúng", color: "bg-rose-950/60 text-rose-300 border-rose-800/50", icon: "🎯" },
  BEFORE_HIT: { label: "Trước Khi Bị Đánh", color: "bg-cyan-950/60 text-cyan-300 border-cyan-800/50", icon: "🛡️" },
  ON_TURN_START: { label: "Đầu Lượt", color: "bg-emerald-950/60 text-emerald-300 border-emerald-800/50", icon: "⏳" },
  ON_BATTLE_START: { label: "Vào Trận", color: "bg-yellow-950/60 text-yellow-300 border-yellow-800/50", icon: "🚩" },
  BEFORE_ENEMY_ATTACK: { label: "Trước Khi Địch Đánh", color: "bg-orange-950/60 text-orange-300 border-orange-800/50", icon: "👁️" },
  ON_DEATH: { label: "Khi Tử Vong", color: "bg-red-950/70 text-red-300 border-red-800/60", icon: "☠️" },
  ON_TICK: { label: "Mỗi Chu Kỳ", color: "bg-teal-950/60 text-teal-300 border-teal-800/50", icon: "🔄" },
  ON_CAST: { label: "Khi Niệm", color: "bg-indigo-950/60 text-indigo-300 border-indigo-800/50", icon: "✨" },
  CAPABILITY: { label: "Năng Lực", color: "bg-sky-950/60 text-sky-300 border-sky-800/50", icon: "🌐" },
  ON_CONSUME: { label: "Khi Nuốt", color: "bg-emerald-950/60 text-emerald-300 border-emerald-800/50", icon: "🧪" },
  ON_USE: { label: "Khi Kích Hoạt", color: "bg-amber-950/60 text-amber-300 border-amber-800/50", icon: "📜" },
};

const STAT_NAMES: Record<string, string> = {
  ATTACK: "Công",
  DEFENSE: "Thủ",
  HP: "Sinh Lực",
  SPEED: "Tốc Độ",
  EVASION: "Né Tránh",
  ACCURACY: "Chính Xác",
  CRIT_RATE: "Bạo Kích",
  TENACITY: "Kiên Cường",
  CC_RESIST: "Kháng Khống Chế",
  LUCK: "Vận Khí",
  ALL: "Toàn Chỉ Số",
  DAMAGE_DEALT: "Sát Thương Gây Ra",
};

const STATUS_NAMES: Record<string, string> = {
  BLEED: "Chảy Máu",
  POISON: "Trúng Độc",
  BURN: "Thiêu Đốt",
  STUN: "Choáng",
  FREEZE: "Đóng Băng",
  SILENCE: "Câm Lặng",
  GU_SEAL: "Khóa Cổ",
  HASTE: "Tăng Tốc",
  SLOW: "Làm Chậm",
  SLOW1: "Làm Chậm Cấp 1",
  SLOW_DAMAGE: "Sát Thương Chậm",
  WEAK: "Suy Yếu",
  REGEN: "Hồi Phục",
  THORNS: "Gai Phản Đòn",
  TAUNT: "Khiêu Khích",
  INVINCIBLE: "Bất Tử",
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
  FORGET_EGO: "Lãng Quên Bản Ngã",
  FORGET_REALITY: "Lãng Quên Thực Tại",
  ARMOR_BREAK: "Phá Giáp",
  DAMAGE_REDUCTION_UP: "Giảm Thương",
  REVIVE_LOCK: "Khóa Hồi Sinh",
  ESSENCE_LEAK: "Thất Thoát Chân Nguyên",
  SOUL_WOUND: "Tổn Thương Hồn",
  THOUGHT_CHAOS: "Tạp Niệm",
  DAO_CONFLICT: "Xung Đột Đạo",
  BACKLASH: "Phản Phệ",
};

const UNIQUE_NAMES: Record<string, string> = {
  REBORN: "Hồi Sinh Khi Chết",
  BYPASS_SHIELD: "Xuyên Hộ Thể (Bỏ Qua Khiên)",
  SUMMON_PHANTOM: "Triệu Hồi Hư Ảnh",
  SUMMON_WOLF_FRONTLINE: "Triệu Hồi Sói Tiền Hàng",
  SWORD_DRAGON_TRANSFORM: "Kiếm Long Biến",
  FULL_SOUL_HEAL: "Đầy Hồn Phục",
  BORROW_PAST_POWER: "Mượn Lực Quá Khứ",
  UNRAVEL_RIDDLE: "Giải Đoán Câu Đố",
  CHAOS_GAMBLE: "Hỗn Độn Đánh Cược",
  SEIZE_INITIATIVE: "Đoạt Tiên Cơ (Đánh Trước)",
  FIRST_TURN_EVADE: "Né Tuyệt Đối Lượt Đầu",
  FORGET_INTENT_INTERRUPT: "Đoán Ý & Ngắt Chiêu",
  FORGET_WILL_CONFUSE: "Ý Niệm Hỗn Loạn",
  DISGUISE: "Ngụy Trang Khí Tức",
  CHANGE_FORM_DISGUISE: "Biến Hóa Ngụy Trang",
  PERFECT_DISGUISE: "Ngụy Trang Hoàn Mỹ",
  ARENA_IDENTITY_MASK: "Mặt Nạ Đấu Trường",
  REMOTE_TRADE: "Giao Dịch Từ Xa",
  MARKETPLACE_ACCESS: "Truy Cập Chợ Trời",
  CHAT_CONNECT: "Kết Nối Thiên Cơ",
  CHAT_PERCEIVE: "Cảm Ứng Thiên Cơ",
  IGNORE_MOVEMENT_RANGE: "Vượt Giới Hạn Di Chuyển",
  LOOT_PREVIEW: "Thấu Thị Chiến Lợi Phẩm",
  SACRIFICE_NPC: "Tế Lễ Sinh Linh",
  POEM_INTENT: "Thi Ý",
  DOUBLE_DAMAGE_DEALT: "Nhân Đôi Sát Thương",
  MANA_REGEN_TICK: "Hồi Chân Nguyên Định Kỳ",
  THOUGHT_REGEN_TICK: "Hồi Niệm Lực Định Kỳ",
  SOUL_REGEN_TICK: "Hồi Hồn Phách Định Kỳ",
  GENERATE_STONES_TICK: "Ngưng Tụ Nguyên Thạch",
  GENERATE_IMMORTAL_STONES_TICK: "Ngưng Tụ Tiên Nguyên Thạch",
  ADD_LIFESPAN_YEARS: "Tăng Thọ Nguyên",
  RANK_PROGRESS: "Tiến Cảnh Giới",
  SACRIFICE_APTITUDE: "Tế Lễ Tư Chất",
};

export function formatTarget(target?: TargetScope): string | null {
  if (!target || !target.selector) return null;
  const sel = target.selector.toUpperCase();
  let label = "";
  switch (sel) {
    case "TARGET":
    case "SINGLE":
      label = "Mục tiêu";
      break;
    case "ROW":
      label = "Hàng ngang";
      break;
    case "COL":
    case "COLUMN":
      label = "Hàng dọc";
      break;
    case "ALL_ENEMIES":
      label = "Toàn bộ địch";
      break;
    case "ALL_ALLIES":
      label = "Toàn bộ đồng đội";
      break;
    case "ALL":
      label = "Toàn trận";
      break;
    case "SELF":
      label = target.bypass_frontline ? "Bản thân" : "";
      break;
    default:
      label = sel;
  }
  if (!label && !target.bypass_frontline) return null;
  if (target.bypass_frontline) {
    return label ? `${label} (Xuyên hàng)` : "Xuyên hàng";
  }
  return label;
}

export function formatOperation(op: AtomicOperation): { text: string; color: string; icon: string } {
  const opType = (op.op || "").toUpperCase();
  switch (opType) {
    case "DAMAGE": {
      const pathStr = op.path ? ` [${translatePath(op.path)}]` : "";
      if (op.sub_type === "MULTIPLIER") {
        return { text: `Sát thương ×${op.value}${pathStr}`, color: "text-rose-300", icon: "⚔️" };
      }
      return { text: `Sát thương ${op.value ?? 0}${pathStr}`, color: "text-rose-300", icon: "⚔️" };
    }
    case "SHIELD": {
      if (op.sub_type === "MULTIPLIER") {
        return { text: `Khiên ×${op.value}`, color: "text-sky-300", icon: "🛡️" };
      }
      return { text: `Khiên ảo +${op.value ?? 0}`, color: "text-sky-300", icon: "🛡️" };
    }
    case "HEAL": {
      if (op.sub_type === "MANA") {
        return { text: `Hồi ${op.value ?? 0} Chân Nguyên`, color: "text-cyan-300", icon: "💧" };
      }
      if (op.sub_type === "LIFESTEAL") {
        return { text: `Hút máu ${Math.round((op.value ?? 0) * 100)}%`, color: "text-emerald-300", icon: "🩸" };
      }
      return { text: `Hồi +${op.value ?? 0} HP`, color: "text-emerald-300", icon: "💚" };
    }
    case "DIRECT_HP": {
      if (op.sub_type === "MAX_HP_PERCENT") {
        return { text: `Sát thương trực tiếp ${Math.round((op.value ?? 0) * 100)}% Max HP (xuyên khiên)`, color: "text-purple-300 font-semibold", icon: "⚡" };
      }
      if (op.sub_type === "SHIELD_STRIP") {
        return { text: `Hủy ${Math.round((op.value ?? 0) * 100)}% khiên mục tiêu`, color: "text-amber-300 font-semibold", icon: "💥" };
      }
      return { text: `Sát thương trực tiếp vào máu`, color: "text-purple-300", icon: "⚡" };
    }
    case "STAT_MOD": {
      const statName = STAT_NAMES[op.stat?.toUpperCase() || ""] || op.stat || "Chỉ số";
      const isMult = op.sub_type === "MULTIPLY_PERCENT";
      const val = op.value ?? 0;
      const text = isMult
        ? `${val >= 0 ? "+" : ""}${Math.round(val * 100)}% ${statName}`
        : `${val >= 0 ? "+" : ""}${val} ${statName}`;
      return { text, color: val >= 0 ? "text-amber-300" : "text-rose-400", icon: "📊" };
    }
    case "STATUS": {
      const sId = (op.status_id || "").toUpperCase();
      const sName = STATUS_NAMES[sId] || sId;
      const parts: string[] = [sName];
      if (op.value && op.value > 0) parts.push(`[${op.value}]`);
      if (op.turns && op.turns > 0) parts.push(`(${op.turns} lượt)`);
      if (op.chance && op.chance > 0 && op.chance < 1) parts.push(`[${Math.round(op.chance * 100)}%]`);
      return { text: `Trạng thái: ${parts.join(" ")}`, color: "text-indigo-300", icon: "✨" };
    }
    case "BARRIER": {
      const turnsStr = op.turns ? ` (${op.turns} lượt)` : "";
      return { text: `Bảo hộ: ${op.sub_type || "Bất tử"}${turnsStr}`, color: "text-yellow-300 font-semibold", icon: "🌟" };
    }
    case "CLEANSE": {
      return { text: `Thanh tẩy: ${op.sub_type || "Giải trừ hiệu ứng"}`, color: "text-cyan-300", icon: "🍃" };
    }
    case "UNIQUE": {
      const code = (op.code || "").toUpperCase();
      const codeName = UNIQUE_NAMES[code] || code;
      const paramStr = op.params && Object.keys(op.params).length > 0 ? ` ${JSON.stringify(op.params)}` : "";
      return { text: `${codeName}${paramStr}`, color: "text-fuchsia-300", icon: "🔮" };
    }
    default:
      return { text: `${opType}: ${JSON.stringify(op)}`, color: "text-zinc-300", icon: "•" };
  }
}

export const EffectBlockItem: React.FC<{ block: ComposableEffectBlock; index?: number }> = ({ block }) => {
  const triggerKey = (block.trigger || "").toUpperCase();
  const cfg = TRIGGER_CONFIG[triggerKey] || {
    label: block.trigger || "Hiệu ứng",
    color: "bg-zinc-900/60 text-zinc-300 border-zinc-800/50",
    icon: "•",
  };
  const targetLabel = formatTarget(block.target);

  return (
    <div className="flex flex-wrap items-center gap-1.5 p-1.5 rounded-lg bg-zinc-950/60 border border-zinc-900/80 text-[10px]">
      <span className={`px-1.5 py-0.5 rounded border text-[9px] font-mono font-medium flex items-center gap-1 ${cfg.color}`}>
        <span>{cfg.icon}</span>
        <span>{cfg.label}</span>
      </span>

      {targetLabel && (
        <span className="px-1.5 py-0.5 rounded bg-zinc-900/80 text-zinc-400 border border-zinc-800/60 text-[9px] font-mono">
          🎯 {targetLabel}
        </span>
      )}

      {block.operations && block.operations.map((op, i) => {
        const { text, color, icon } = formatOperation(op);
        return (
          <span
            key={i}
            className={`px-1.5 py-0.5 rounded bg-zinc-900/40 border border-zinc-800/40 text-[9.5px] font-mono flex items-center gap-1 ${color}`}
          >
            <span className="opacity-80 text-[9px]">{icon}</span>
            <span>{text}</span>
          </span>
        );
      })}
    </div>
  );
};

export const EffectBlocksList: React.FC<{ blocks?: ComposableEffectBlock[] | null }> = ({ blocks }) => {
  if (!blocks || !Array.isArray(blocks) || blocks.length === 0) return null;
  return (
    <div className="flex flex-col gap-1 w-full">
      {blocks.map((b, i) => (
        <EffectBlockItem key={b.id || i} block={b} index={i} />
      ))}
    </div>
  );
};
