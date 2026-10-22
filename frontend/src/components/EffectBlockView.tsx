import React from "react";
import type { EffectBlock, EffectEnvelope, EffectLike, EffectOperation, EffectTarget } from "@/lib/effect-blocks";
import { translatePath } from "@/lib/translate";
import { formatEffectTarget, renderableEffectBlocks } from "@/lib/effect-presentation";

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
  CRIT_RATE: "Bạo Kích",
  TENACITY: "Kiên Cường",
  CC_RESIST: "Kháng Khống Chế",
  LUCK: "Vận Khí",
  ALL: "Toàn Chỉ Số",
  DAMAGE_DEALT: "Sát Thương Gây Ra",
  MOVEMENT: "Di Chuyển (ô)",
  INVESTIGATION: "Điều Tra",
  HEARING: "Thính Giác",
  DISGUISE: "Ngụy Trang",
  TIME_SPEED: "Tốc Độ Thời Gian",
  REFINE_SUCCESS: "Tỷ Lệ Luyện Cổ",
  REFINE_MANA_TIER: "Tinh Luyện Chân Nguyên",
  STORAGE: "Sức Chứa",
  DAO_MARK: "Đạo Ngân",
  PATH_CONFLICT_REDUCTION: "Giảm Xung Đột Đạo",
  PATH_RESISTANCE: "Kháng Lưu Phái",
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
  INVINCIBLE: "Bất Hoại",
  LAST_STAND: "Bất Tử",
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
  INSTANT_WIN: "Chiến Thắng Tuyệt Đối",
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
  PORTAL_CREATE: "Tạo Cổng Dịch Chuyển",
  RANDOM_PORTAL: "Cổng Dịch Chuyển Ngẫu Nhiên",
  CAPTURE_BONUS: "Tăng Tỷ Lệ Thu Phục",
  DEDUCTION_INSTANT_FINISH: "Đốn Ngộ Thôi Diễn Bí Phương",
  REVERSE_CALAMITY: "Nghịch Chuyển Càn Khôn",
  SUPPRESS_PRIORITY: "Cấm Tuyệt Tiên Thủ",
  GROWTH_DURATION_HOURS: "Thời Gian Trưởng Thành Cổ",
  IMMUNE_PATH_STATUS: "Miễn Nhiễm Trạng Thái Lưu Phái",
  SOVEREIGN_BODY_TRANSFORM: "Cải Tạo Tiên Thể Chí Tôn",
};

export const formatTarget = formatEffectTarget;

export function formatOperation(op: EffectOperation): { text: string; color: string; icon: string } {
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
      const isPathConflictReduction = op.stat?.toUpperCase() === "PATH_CONFLICT_REDUCTION";
      const text = isPathConflictReduction
        ? `${val >= 0 ? "+" : ""}${val}% ${statName}`
        : isMult
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
      const val = op.params?.value !== undefined ? op.params.value : op.value;

      if (code === "STAT_MODS" && val && typeof val === "object") {
        const parts = Object.entries(val as Record<string, number>).map(([k, v]) => {
          const sName = STAT_NAMES[k.toUpperCase()] || k;
          return `${sName} ${v >= 0 ? "+" : ""}${Math.round(v * 100)}%`;
        });
        const isPositive = Object.values(val as Record<string, number>).every((v) => v >= 0);
        return { text: parts.join(", "), color: isPositive ? "text-amber-300" : "text-rose-400", icon: "📊" };
      }
      if (code === "DAO_MARK_MODIFIER" && val && typeof val === "object") {
        const parts = Object.entries(val as Record<string, number>).map(([path, v]) => {
          return `Đạo Ngân [${translatePath(path)}] ${v >= 0 ? "+" : ""}${Math.round(v * 100)}%`;
        });
        return { text: parts.join(", "), color: "text-indigo-300", icon: "⚡" };
      }
      if (code === "PATH_DAMAGE_MODIFIER" && val && typeof val === "object") {
        const parts = Object.entries(val as Record<string, number>).map(([path, v]) => {
          return `Sát Thương [${translatePath(path)}] ${v >= 0 ? "+" : ""}${Math.round(v * 100)}%`;
        });
        return { text: parts.join(", "), color: "text-rose-300", icon: "⚔️" };
      }
      if (code === "PATH_RESISTANCE_MODIFIER" && val && typeof val === "object") {
        const parts = Object.entries(val as Record<string, number>).map(([path, v]) => {
          return `Kháng [${translatePath(path)}] ${v >= 0 ? "+" : ""}${Math.round(v * 100)}%`;
        });
        return { text: parts.join(", "), color: "text-sky-300", icon: "🛡️" };
      }
      if (code === "MANA_REGEN") {
        const num = Number(val) || 0;
        return { text: `Hồi Chân Nguyên ${num >= 0 ? "+" : ""}${num}`, color: "text-cyan-300", icon: "💧" };
      }
      if (code === "MAX_MANA") {
        const num = Number(val) || 0;
        return { text: `Chân Nguyên Tối Đa +${Math.round(num * 100)}%`, color: "text-cyan-300", icon: "💧" };
      }
      if (code === "HEALING_RECEIVED") {
        const num = Number(val) || 0;
        return { text: `Hiệu Quả Hồi Máu ${num >= 0 ? "+" : ""}${Math.round(num * 100)}%`, color: num >= 0 ? "text-emerald-300" : "text-rose-400", icon: "💚" };
      }
      if (code === "HP_REGEN") {
        const num = Number(val) || 0;
        return { text: `Hồi Sinh Lực ${num >= 0 ? "+" : ""}${num}/giờ`, color: "text-emerald-300", icon: "💚" };
      }
      if (code === "REFINE_SUCCESS") {
        const num = Number(val) || 0;
        return { text: `Tỷ Lệ Luyện Cổ ${num >= 0 ? "+" : ""}${Math.round(num * 100)}%`, color: num >= 0 ? "text-amber-300" : "text-rose-400", icon: "🧪" };
      }
      if (code === "LUCK") {
        const num = Number(val) || 0;
        return { text: `Vận Khí ${num >= 0 ? "+" : ""}${num}`, color: "text-yellow-300", icon: "🍀" };
      }
      if (code === "CULTIVATION_EFFICIENCY") {
        const num = Number(val) || 0;
        return { text: `Hiệu Suất Tu Luyện ${num >= 0 ? "+" : ""}${Math.round(num * 100)}%`, color: num >= 0 ? "text-purple-300" : "text-rose-400", icon: "📈" };
      }
      if (code === "DAO_MARK_GAIN") {
        const num = Number(val) || 0;
        return { text: `Thu Hoạch Đạo Ngân +${Math.round(num * 100)}%`, color: "text-violet-300", icon: "✨" };
      }
      if (code === "MENTAL_RESISTANCE") {
        const num = Number(val) || 0;
        return { text: `Kháng Tâm Trí / Khống Chế +${Math.round(num * 100)}%`, color: "text-teal-300", icon: "🧘" };
      }
      if (code === "SOUL_DEFENSE") {
        const num = Number(val) || 0;
        return { text: `Phòng Ngự Hồn Phách +${Math.round(num * 100)}%`, color: "text-indigo-300", icon: "👻" };
      }
      if (code === "SOUL_REGEN") {
        const num = Number(val) || 0;
        return { text: `Dưỡng Hồn +${Math.round(num * 100)}%`, color: "text-indigo-300", icon: "👻" };
      }
      if (code === "TENACITY") {
        const num = Number(val) || 0;
        return { text: `Kiên Cường +${Math.round(num * 100)}%`, color: "text-blue-300", icon: "🛡️" };
      }
      if (code === "CC_RESIST") {
        const num = Number(val) || 0;
        return { text: `Kháng Khống Chế +${Math.round(num * 100)}%`, color: "text-blue-300", icon: "🛡️" };
      }
      if (code === "APERTURE_STRESS") {
        const num = Number(val) || 0;
        return { text: `Áp Lực Không Khiếu +${Math.round(num * 100)}%`, color: "text-red-300", icon: "💥" };
      }
      if (code === "NON_CONFLICTING_DAO_MARKS") {
        return { text: `Đạo Ngân Không Xung Đột`, color: "text-amber-300", icon: "☯️" };
      }
      if (code === "IMMORTAL") {
        return { text: `Thọ Nguyên Vô Tận (Bất Tử)`, color: "text-yellow-300 font-semibold", icon: "♾️" };
      }
      if (code === "DISABLE_HP_REGEN") {
        return { text: `Không Thể Hồi Sinh Lực Tự Nhiên`, color: "text-rose-400", icon: "🚫" };
      }
      if (code === "DISABLE_MANA_REGEN") {
        return { text: `Không Thể Hồi Chân Nguyên Tự Nhiên`, color: "text-rose-400", icon: "🚫" };
      }
      if (code === "DISABLE_RESOURCE_REGEN") {
        return { text: `Không Thể Hồi Tiên/Chân Nguyên`, color: "text-rose-400", icon: "🚫" };
      }
      if (code === "DISABLE_SOUL_REGEN") {
        return { text: `Không Thể Phục Hồi Hồn Phách`, color: "text-rose-400", icon: "🚫" };
      }
      if (code === "DEDUCTION_INSTANT_FINISH") {
        const p = op.params || {};
        const base = Number(p.base_chance ?? 0.05);
        const step = Number(p.per_rank_step ?? 0.02);
        return { text: `Đốn ngộ hoàn thành thôi diễn ngay (${Math.round(base * 100)}% + ${Math.round(step * 100)}%/rank chênh lệch)`, color: "text-amber-300", icon: "💡" };
      }
      if (code === "REVERSE_CALAMITY") {
        return { text: `Đảo nghịch mọi trạng thái bất lợi về hư vô (chỉ nhận sát thương)`, color: "text-slate-200 font-semibold", icon: "☯️" };
      }
      if (code === "SUPPRESS_PRIORITY") {
        return { text: `Trấn áp ưu tiên hành động của địch — trấn áp thành công thì đi trước Lượt 1`, color: "text-red-300 font-semibold", icon: "⛔" };
      }

      const codeName = UNIQUE_NAMES[code] || code;
      const paramStr = op.params && Object.keys(op.params).length > 0 ? ` ${JSON.stringify(op.params)}` : "";
      return { text: `${codeName}${paramStr}`, color: "text-fuchsia-300", icon: "🔮" };
    }
    default:
      return { text: `${opType}: ${JSON.stringify(op)}`, color: "text-zinc-300", icon: "•" };
  }
}

const operationMeta = (op: EffectOperation): string[] => {
  const parts: string[] = [];
  const target = formatTarget(op.target);
  if (target) parts.push(`mục tiêu: ${target}`);
  if (op.unit) parts.push(`đơn vị: ${op.unit}`);
  if (typeof op.chance === "number") parts.push(`tỉ lệ: ${Math.round(op.chance * 100)}%`);
  if (typeof op.turns === "number") parts.push(`${op.turns} lượt`);
  if (op.stack_policy) parts.push(`cộng dồn: ${op.stack_policy}`);
  return parts;
};

export const EffectBlockItem: React.FC<{ block: EffectBlock; index?: number }> = ({ block }) => {
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

      {block.conditions?.map((condition, index) => (
        <span key={`condition-${index}`} className="px-1.5 py-0.5 rounded bg-slate-900/70 text-slate-300 border border-slate-700/60 text-[9px] font-mono">
          nếu {condition.field} {condition.operator} {condition.value}
        </span>
      ))}

      {block.operations && block.operations.map((op, i) => {
        const { text, color, icon } = formatOperation(op);
        const metadata = operationMeta(op);
        return (
          <span
            key={i}
            className={`px-1.5 py-0.5 rounded bg-zinc-900/40 border border-zinc-800/40 text-[9.5px] font-mono flex items-center gap-1 ${color}`}
          >
            <span className="opacity-80 text-[9px]">{icon}</span>
            <span>{text}</span>
            {metadata.length > 0 && <span className="text-zinc-500">· {metadata.join(" · ")}</span>}
          </span>
        );
      })}
    </div>
  );
};

export const EffectBlocksList: React.FC<{ blocks?: EffectBlock[] | null }> = ({ blocks }) => {
  if (!blocks || !Array.isArray(blocks) || blocks.length === 0) return null;
  return (
    <div className="flex flex-col gap-1 w-full">
      {blocks.map((b, i) => (
        <EffectBlockItem key={b.id || i} block={b} index={i} />
      ))}
    </div>
  );
};

/** Canonical renderer. Unknown envelopes never throw and remain inspectable. */
export const EffectEnvelopeView: React.FC<{ effects?: EffectLike | null }> = ({ effects }) => {
  const presentation = renderableEffectBlocks(effects);
  if (presentation.unsupportedSchema !== undefined) {
    return <div className="text-[10px] text-amber-400">Effect schema không được hỗ trợ: {String(presentation.unsupportedSchema)}</div>;
  }
  return <EffectBlocksList blocks={presentation.blocks} />;
};
