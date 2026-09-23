"use client";

import React, { useState, useId } from "react";
import {
  Zap,
  Shield,
  Heart,
  Plus,
  Trash2,
  Copy,
  ChevronDown,
  ChevronUp,
  FileCode,
  Sliders,
  Sparkles,
  Info,
  Check,
  AlertCircle,
  Eye,
  Activity,
  Layers,
} from "lucide-react";
import type { ComposableEffectBlock, AtomicOperation, TargetScope } from "@/lib/gu-dict";

interface Props {
  effects: ApiResponseData;
  onChange: (effects: ApiResponseData) => void;
}

const TRIGGER_OPTIONS = [
  { value: "ON_ACTION", label: "⚡ ON_ACTION (Chủ động khi xuất chiêu)", desc: "Kích hoạt khi Cổ trùng được kích hoạt trong lượt" },
  { value: "ALWAYS_ON", label: "🧘 ALWAYS_ON (Bị động duy trì)", desc: "Hiệu ứng nội tại có hiệu lực liên tục trong suốt trận đấu" },
  { value: "ON_HIT", label: "💥 ON_HIT (Khi bị trúng đòn)", desc: "Kích hoạt phản đòn hoặc hộ thân khi bản thân nhận sát thương" },
  { value: "ON_TURN_START", label: "🔄 ON_TURN_START (Bắt đầu hiệp)", desc: "Kích hoạt tự động khi bắt đầu mỗi hiệp đấu" },
  { value: "ON_DEATH", label: "💀 ON_DEATH (Khi tử trận)", desc: "Kích hoạt lần cuối khi bản thân bị hạ gục" },
  { value: "CAPABILITY", label: "🧩 CAPABILITY (Khả năng đặc thù)", desc: "Nội tại hoặc tính năng chuyên biệt ngoài giao tranh" },
];

const TARGET_SELECTORS = [
  { value: "TARGET", label: "🎯 TARGET (Mục tiêu hiện tại)" },
  { value: "SELF", label: "👤 SELF (Bản thân người dùng)" },
  { value: "ALL_ENEMIES", label: "👥 ALL_ENEMIES (Toàn bộ phe địch)" },
  { value: "ALL_ALLIES", label: "🛡️ ALL_ALLIES (Toàn bộ phe ta)" },
  { value: "ROW", label: "↔️ ROW (Hàng ngang mục tiêu)" },
  { value: "COLUMN", label: "↕️ COLUMN (Hàng dọc mục tiêu)" },
];

const OP_TYPES = [
  { value: "DAMAGE", label: "⚔️ DAMAGE (Sát thương)", color: "text-rose-400" },
  { value: "SHIELD", label: "🛡️ SHIELD (Hộ thuẫn)", color: "text-amber-400" },
  { value: "HEAL", label: "💚 HEAL (Hồi phục)", color: "text-emerald-400" },
  { value: "DIRECT_HP", label: "🩸 DIRECT_HP (Sinh lực trực tiếp)", color: "text-red-500" },
  { value: "STAT_MOD", label: "📈 STAT_MOD (Tăng/giảm chỉ số)", color: "text-sky-400" },
  { value: "STATUS", label: "💫 STATUS (Trạng thái hiệu ứng)", color: "text-purple-400" },
  { value: "BARRIER", label: "🌀 BARRIER (Kết giới)", color: "text-indigo-400" },
  { value: "CLEANSE", label: "✨ CLEANSE (Tẩy trừ)", color: "text-teal-400" },
  { value: "UNIQUE", label: "⭐ UNIQUE (Cơ chế đặc thù)", color: "text-yellow-400" },
];

const COMMON_STATUSES = [
  { id: "FREEZE", label: "Đóng Băng (FREEZE)" },
  { id: "STUN", label: "Choáng Váng (STUN)" },
  { id: "SILENCE", label: "Câm Lặng (SILENCE)" },
  { id: "BLEED", label: "Xuất Huyết (BLEED)" },
  { id: "POISON", label: "Trúng Độc (POISON)" },
  { id: "BURN", label: "Thiêu Đốt (BURN)" },
  { id: "WEAK", label: "Suy Yếu (WEAK)" },
  { id: "SLOW", label: "Làm Chậm (SLOW)" },
  { id: "REGEN", label: "Hồi Phục Liên Tục (REGEN)" },
  { id: "THORNS", label: "Gai Phản Đòn (THORNS)" },
  { id: "TAUNT", label: "Khiêu Khích (TAUNT)" },
  { id: "ATK_UP", label: "Tăng Công (ATK_UP)" },
  { id: "DEF_UP", label: "Tăng Thủ (DEF_UP)" },
  { id: "STEALTH", label: "Tàng Hình (STEALTH)" },
  { id: "BLIND", label: "Mù Lòa (BLIND)" },
];

const STAT_OPTIONS = [
  { value: "ATTACK", label: "Tấn Công (ATTACK)" },
  { value: "DEFENSE", label: "Phòng Thủ (DEFENSE)" },
  { value: "HP", label: "Sinh Lực Tối Đa (HP)" },
  { value: "SPEED", label: "Tốc Độ (SPEED)" },
  { value: "EVASION", label: "Né Tránh (EVASION)" },
  { value: "CRIT_RATE", label: "Bạo Kích (CRIT_RATE)" },
];

const DAO_PATHS = [
  "blood", "bone", "dark", "dream", "earth", "enchantment", "fire", "formation",
  "heaven", "ice", "illusion", "information", "kill", "light", "lightning", "luck",
  "metal", "moon", "phantom", "poison", "refinement", "rule", "shadow", "slave",
  "soul", "sound", "space", "star", "strength", "sun", "sword", "time", "transformation",
  "water", "wind", "wisdom", "wood"
];

// Presets for quick block creation
const PRESET_TEMPLATES = [
  {
    name: "⚔️ Đòn tấn công đơn",
    block: {
      trigger: "ON_ACTION",
      target: { selector: "TARGET", bypass_frontline: false },
      operations: [{ op: "DAMAGE", value: 30, chance: 1, path: "moon" }],
    },
  },
  {
    name: "💥 Sát thương toàn thể (AOE)",
    block: {
      trigger: "ON_ACTION",
      target: { selector: "ALL_ENEMIES", bypass_frontline: true },
      operations: [{ op: "DAMAGE", value: 25, chance: 1 }],
    },
  },
  {
    name: "🛡️ Tạo Hộ Thuẫn",
    block: {
      trigger: "ON_ACTION",
      target: { selector: "SELF", bypass_frontline: false },
      operations: [{ op: "SHIELD", value: 50, turns: 2 }],
    },
  },
  {
    name: "💚 Hồi phục Sinh Lực",
    block: {
      trigger: "ON_ACTION",
      target: { selector: "SELF", bypass_frontline: false },
      operations: [{ op: "HEAL", sub_type: "HP", value: 40 }],
    },
  },
  {
    name: "❄️ Tấn công kèm Khống Chế",
    block: {
      trigger: "ON_ACTION",
      target: { selector: "TARGET", bypass_frontline: false },
      operations: [
        { op: "DAMAGE", value: 20, chance: 1 },
        { op: "STATUS", status_id: "FREEZE", turns: 2, chance: 0.5 },
      ],
    },
  },
  {
    name: "🧘 Bị động tăng chỉ số",
    block: {
      trigger: "ALWAYS_ON",
      target: { selector: "SELF", bypass_frontline: false },
      operations: [{ op: "STAT_MOD", stat: "ATTACK", value: 15 }],
    },
  },
  {
    name: "🩸 Bị động phản đòn",
    block: {
      trigger: "ON_HIT",
      target: { selector: "TARGET", bypass_frontline: false },
      operations: [{ op: "DAMAGE", value: 15, chance: 0.8 }],
    },
  },
  {
    name: "🔄 Đầu hiệp hồi Chân Nguyên",
    block: {
      trigger: "ON_TURN_START",
      target: { selector: "SELF", bypass_frontline: false },
      operations: [{ op: "HEAL", sub_type: "MANA", value: 10 }],
    },
  },
];

export function GuEffectsEditor({ effects, onChange }: Props) {
  const [tab, setTab] = useState<"visual" | "json">("visual");
  const [jsonText, setJsonText] = useState("");
  const [jsonError, setJsonError] = useState<string | null>(null);
  const [collapsedBlocks, setCollapsedBlocks] = useState<Record<number, boolean>>({});
  const datalistId = useId();

  const blocks: ComposableEffectBlock[] = Array.isArray(effects.blocks) ? effects.blocks : [];
  const growth = (effects.growth || {}) as Record<string, number>;
  const house = effects.house || {};
  const legacyActive = effects.active && Object.keys(effects.active).length > 0 ? effects.active : null;
  const legacyPassive = effects.passive && Object.keys(effects.passive).length > 0 ? effects.passive : null;
  const legacyOther = Array.isArray(effects.other) && effects.other.length > 0 ? effects.other : null;

  const handleSwitchToJson = () => {
    setJsonText(JSON.stringify(effects, null, 2));
    setJsonError(null);
    setTab("json");
  };

  const handleSwitchToVisual = () => {
    try {
      const parsed = JSON.parse(jsonText);
      if (typeof parsed !== "object" || parsed === null) {
        setJsonError("Dữ liệu JSON phải là một đối tượng (Object)");
        return;
      }
      onChange(parsed);
      setJsonError(null);
      setTab("visual");
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Cú pháp JSON không hợp lệ";
      setJsonError(`Lỗi cú pháp: ${msg}`);
    }
  };

  const updateBlocks = (newBlocks: ComposableEffectBlock[]) => {
    onChange({
      ...effects,
      blocks: newBlocks,
    });
  };

  const addEmptyBlock = () => {
    const newBlock: ComposableEffectBlock = {
      trigger: "ON_ACTION",
      target: { selector: "TARGET", bypass_frontline: false },
      operations: [
        {
          op: "DAMAGE",
          value: 10,
          chance: 1,
        },
      ],
    };
    updateBlocks([...blocks, newBlock]);
  };

  const addPresetBlock = (presetIndex: number) => {
    const preset = PRESET_TEMPLATES[presetIndex];
    if (!preset) return;
    const cloned = JSON.parse(JSON.stringify(preset.block));
    updateBlocks([...blocks, cloned]);
  };

  const removeBlock = (index: number) => {
    const updated = blocks.filter((_, i) => i !== index);
    updateBlocks(updated);
  };

  const duplicateBlock = (index: number) => {
    const cloned = JSON.parse(JSON.stringify(blocks[index]));
    const updated = [...blocks];
    updated.splice(index + 1, 0, cloned);
    updateBlocks(updated);
  };

  const toggleCollapse = (index: number) => {
    setCollapsedBlocks((prev) => ({ ...prev, [index]: !prev[index] }));
  };

  const updateBlockField = (index: number, patch: Partial<ComposableEffectBlock>) => {
    const updated = blocks.map((b, i) => (i === index ? { ...b, ...patch } : b));
    updateBlocks(updated);
  };

  const updateBlockTarget = (index: number, patch: Partial<TargetScope>) => {
    const block = blocks[index];
    const currentTarget = block.target || { selector: "TARGET", bypass_frontline: false };
    updateBlockField(index, {
      target: { ...currentTarget, ...patch },
    });
  };

  const addOperation = (blockIndex: number) => {
    const block = blocks[blockIndex];
    const newOp: AtomicOperation = {
      op: "DAMAGE",
      value: 10,
      chance: 1,
    };
    updateBlockField(blockIndex, {
      operations: [...block.operations, newOp],
    });
  };

  const removeOperation = (blockIndex: number, opIndex: number) => {
    const block = blocks[blockIndex];
    updateBlockField(blockIndex, {
      operations: block.operations.filter((_, i) => i !== opIndex),
    });
  };

  const updateOperation = (blockIndex: number, opIndex: number, patch: Partial<AtomicOperation>) => {
    const block = blocks[blockIndex];
    const updatedOps = block.operations.map((op, i) => (i === opIndex ? { ...op, ...patch } : op));
    updateBlockField(blockIndex, { operations: updatedOps });
  };

  const updateGrowth = (key: string, val: number) => {
    const newGrowth = { ...growth, [key]: val };
    onChange({ ...effects, growth: newGrowth });
  };

  return (
    <div className="mt-4 border-t border-zinc-800 pt-4">
      {/* Header with View Tabs */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <Layers className="w-4 h-4 text-amber-500" />
          <h4 className="text-xs font-bold text-zinc-300 uppercase tracking-wider">
            Hiệu Ứng Cổ Trùng (Composable Blocks)
          </h4>
          <span className="text-[10px] px-1.5 py-0.5 rounded bg-amber-950/60 text-amber-400 border border-amber-800/40 font-mono">
            {blocks.length} blocks
          </span>
        </div>

        <div className="flex items-center gap-1 bg-zinc-800/60 p-0.5 rounded-lg border border-zinc-700/50">
          <button
            type="button"
            onClick={() => {
              if (tab === "json") handleSwitchToVisual();
              else setTab("visual");
            }}
            className={`flex items-center gap-1 px-2.5 py-1 rounded text-[11px] font-medium transition-colors cursor-pointer ${
              tab === "visual"
                ? "bg-amber-600 text-white shadow-sm"
                : "text-zinc-400 hover:text-zinc-200"
            }`}
          >
            <Sliders className="w-3 h-3" />
            Trực quan
          </button>
          <button
            type="button"
            onClick={handleSwitchToJson}
            className={`flex items-center gap-1 px-2.5 py-1 rounded text-[11px] font-medium transition-colors cursor-pointer ${
              tab === "json"
                ? "bg-amber-600 text-white shadow-sm"
                : "text-zinc-400 hover:text-zinc-200"
            }`}
          >
            <FileCode className="w-3 h-3" />
            JSON thô
          </button>
        </div>
      </div>

      {/* JSON Mode */}
      {tab === "json" && (
        <div className="space-y-2 mb-4">
          <div className="flex items-center justify-between text-[11px] text-zinc-400">
            <span>Chỉnh sửa trực tiếp cấu trúc JSON của Cổ trùng:</span>
            <button
              type="button"
              onClick={handleSwitchToVisual}
              className="text-amber-400 hover:text-amber-300 cursor-pointer font-medium"
            >
              Áp dụng & chuyển sang Trực quan →
            </button>
          </div>
          {jsonError && (
            <div className="p-2.5 rounded-lg bg-red-950/40 border border-red-800/60 text-xs text-red-400 flex items-center gap-2">
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span>{jsonError}</span>
            </div>
          )}
          <textarea
            value={jsonText}
            onChange={(e) => {
              setJsonText(e.target.value);
              setJsonError(null);
            }}
            rows={14}
            className="w-full px-3 py-2 bg-zinc-950 border border-zinc-700 rounded-lg text-xs text-zinc-200 font-mono leading-relaxed focus:outline-none focus:border-amber-500"
          />
        </div>
      )}

      {/* Visual Mode */}
      {tab === "visual" && (
        <div className="space-y-4">
          {/* Quick Presets Picker */}
          <div className="bg-zinc-900/60 p-3 rounded-xl border border-zinc-800/80">
            <div className="flex items-center justify-between mb-2">
              <span className="text-[11px] font-bold text-zinc-400 uppercase tracking-wider flex items-center gap-1.5">
                <Sparkles className="w-3 h-3 text-amber-400" />
                Mẫu tạo nhanh (Presets)
              </span>
              <span className="text-[10px] text-zinc-500">Bấm để thêm nhanh mẫu block chuẩn</span>
            </div>
            <div className="flex flex-wrap gap-1.5">
              {PRESET_TEMPLATES.map((preset, idx) => (
                <button
                  key={idx}
                  type="button"
                  onClick={() => addPresetBlock(idx)}
                  className="px-2.5 py-1 rounded-lg bg-zinc-800 hover:bg-zinc-700 border border-zinc-700/60 text-[11px] text-zinc-300 hover:text-amber-300 transition-colors flex items-center gap-1 cursor-pointer"
                >
                  <Plus className="w-3 h-3 text-amber-500" />
                  {preset.name}
                </button>
              ))}
            </div>
          </div>

          {/* Blocks List */}
          <div className="space-y-3">
            {blocks.map((block, bIdx) => {
              const isCollapsed = !!collapsedBlocks[bIdx];
              return (
                <div
                  key={bIdx}
                  className="rounded-xl border border-zinc-700/60 bg-zinc-900/80 overflow-hidden shadow-sm"
                >
                  {/* Block Header */}
                  <div className="flex items-center justify-between p-3 bg-zinc-800/60 border-b border-zinc-800">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="w-5 h-5 rounded-full bg-amber-500/20 text-amber-400 text-[10px] font-bold flex items-center justify-center border border-amber-500/40">
                        {bIdx + 1}
                      </span>
                      <span className="text-xs font-bold text-zinc-200">
                        Khối #{bIdx + 1}:
                      </span>
                      <span className="px-2 py-0.5 rounded text-[10px] font-mono font-bold bg-amber-950/70 text-amber-400 border border-amber-600/30">
                        {block.trigger}
                      </span>
                      <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-zinc-800 text-zinc-300 border border-zinc-700">
                        {block.target?.selector || "TARGET"}
                        {block.target?.bypass_frontline ? " (Xuyên hàng)" : ""}
                      </span>
                      <span className="text-[10px] text-zinc-500 font-mono">
                        ({block.operations?.length || 0} hành vi)
                      </span>
                    </div>

                    <div className="flex items-center gap-1.5">
                      <button
                        type="button"
                        onClick={() => duplicateBlock(bIdx)}
                        title="Nhân bản khối này"
                        className="p-1 rounded text-zinc-400 hover:text-zinc-200 hover:bg-zinc-700/60 transition-colors cursor-pointer"
                      >
                        <Copy className="w-3.5 h-3.5" />
                      </button>
                      <button
                        type="button"
                        onClick={() => removeBlock(bIdx)}
                        title="Xóa khối này"
                        className="p-1 rounded text-red-400 hover:text-red-300 hover:bg-zinc-700/60 transition-colors cursor-pointer"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                      <button
                        type="button"
                        onClick={() => toggleCollapse(bIdx)}
                        className="p-1 rounded text-zinc-400 hover:text-zinc-200 hover:bg-zinc-700/60 transition-colors cursor-pointer"
                      >
                        {isCollapsed ? (
                          <ChevronDown className="w-4 h-4" />
                        ) : (
                          <ChevronUp className="w-4 h-4" />
                        )}
                      </button>
                    </div>
                  </div>

                  {/* Block Body */}
                  {!isCollapsed && (
                    <div className="p-3 space-y-3">
                      {/* Trigger & Target Row */}
                      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                        <div>
                          <label className="text-[10px] font-bold text-zinc-400 uppercase tracking-wider block mb-1">
                            Thời điểm kích hoạt (Trigger)
                          </label>
                          <select
                            value={block.trigger}
                            onChange={(e) => updateBlockField(bIdx, { trigger: e.target.value })}
                            className="w-full px-2.5 py-1.5 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500"
                          >
                            {TRIGGER_OPTIONS.map((opt) => (
                              <option key={opt.value} value={opt.value}>
                                {opt.label}
                              </option>
                            ))}
                          </select>
                        </div>

                        <div>
                          <label className="text-[10px] font-bold text-zinc-400 uppercase tracking-wider block mb-1">
                            Phạm vi mục tiêu mặc định (Target Scope)
                          </label>
                          <div className="flex gap-2">
                            <select
                              value={block.target?.selector || "TARGET"}
                              onChange={(e) => updateBlockTarget(bIdx, { selector: e.target.value })}
                              className="flex-1 px-2.5 py-1.5 bg-zinc-800 border border-zinc-700 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500"
                            >
                              {TARGET_SELECTORS.map((opt) => (
                                <option key={opt.value} value={opt.value}>
                                  {opt.label}
                                </option>
                              ))}
                            </select>
                            <label className="flex items-center gap-1.5 px-2.5 py-1 bg-zinc-800 border border-zinc-700 rounded-lg text-[11px] text-zinc-300 cursor-pointer shrink-0">
                              <input
                                type="checkbox"
                                checked={!!block.target?.bypass_frontline}
                                onChange={(e) => updateBlockTarget(bIdx, { bypass_frontline: e.target.checked })}
                                className="rounded bg-zinc-900 border-zinc-600 text-amber-600 focus:ring-0"
                              />
                              <span>Xuyên hàng</span>
                            </label>
                          </div>
                        </div>
                      </div>

                      {/* Operations Section */}
                      <div className="pt-2 border-t border-zinc-800/80">
                        <div className="flex items-center justify-between mb-2">
                          <span className="text-[11px] font-bold text-zinc-400 uppercase tracking-wider">
                            Các hành vi nguyên tử (Atomic Operations)
                          </span>
                          <button
                            type="button"
                            onClick={() => addOperation(bIdx)}
                            className="text-[11px] font-medium text-amber-400 hover:text-amber-300 flex items-center gap-1 cursor-pointer"
                          >
                            <Plus className="w-3 h-3" />
                            Thêm hành vi
                          </button>
                        </div>

                        {(!block.operations || block.operations.length === 0) && (
                          <div className="p-3 text-center border border-dashed border-zinc-800 rounded-lg text-xs text-zinc-500">
                            Chưa có hành vi nào trong khối này. Hãy thêm hành vi (Sát thương, Hộ thuẫn, Trạng thái...).
                          </div>
                        )}

                        <div className="space-y-2">
                          {block.operations?.map((op, opIdx) => (
                            <div
                              key={opIdx}
                              className="p-2.5 bg-zinc-950/60 border border-zinc-800 rounded-lg text-xs space-y-2"
                            >
                              <div className="flex items-center justify-between gap-2">
                                <div className="flex items-center gap-2 flex-1">
                                  <span className="w-4 h-4 rounded bg-zinc-800 text-[9px] font-bold text-zinc-400 flex items-center justify-center">
                                    {opIdx + 1}
                                  </span>
                                  <select
                                    value={op.op}
                                    onChange={(e) => updateOperation(bIdx, opIdx, { op: e.target.value })}
                                    className="px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200 font-bold focus:outline-none focus:border-amber-500"
                                  >
                                    {OP_TYPES.map((t) => (
                                      <option key={t.value} value={t.value}>
                                        {t.label}
                                      </option>
                                    ))}
                                  </select>
                                </div>
                                <button
                                  type="button"
                                  onClick={() => removeOperation(bIdx, opIdx)}
                                  className="text-red-400 hover:text-red-300 p-1 cursor-pointer"
                                  title="Xóa hành vi này"
                                >
                                  <Trash2 className="w-3.5 h-3.5" />
                                </button>
                              </div>

                              {/* Form fields depending on op type */}
                              <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 pt-1 border-t border-zinc-900 text-[11px]">
                                {/* DAMAGE Fields */}
                                {op.op === "DAMAGE" && (
                                  <>
                                    <div>
                                      <label className="text-zinc-500 block text-[10px]">Sát thương (Value)</label>
                                      <input
                                        type="number"
                                        step="any"
                                        value={op.value ?? 0}
                                        onChange={(e) => updateOperation(bIdx, opIdx, { value: parseFloat(e.target.value) || 0 })}
                                        className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200"
                                      />
                                    </div>
                                    <div>
                                      <label className="text-zinc-500 block text-[10px]">Lưu phái (Path)</label>
                                      <input
                                        list={`dao-paths-${datalistId}`}
                                        value={op.path ?? ""}
                                        onChange={(e) => updateOperation(bIdx, opIdx, { path: e.target.value })}
                                        placeholder="Tự do / Hệ Cổ"
                                        className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200 font-mono"
                                      />
                                    </div>
                                    <div>
                                      <label className="text-zinc-500 block text-[10px]">Tỉ lệ kích hoạt (0..1)</label>
                                      <input
                                        type="number"
                                        step="0.05"
                                        min="0"
                                        max="1"
                                        value={op.chance ?? 1}
                                        onChange={(e) => updateOperation(bIdx, opIdx, { chance: parseFloat(e.target.value) || 0 })}
                                        className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200"
                                      />
                                    </div>
                                  </>
                                )}

                                {/* SHIELD Fields */}
                                {op.op === "SHIELD" && (
                                  <>
                                    <div>
                                      <label className="text-zinc-500 block text-[10px]">Lượng Khiên (Value)</label>
                                      <input
                                        type="number"
                                        step="any"
                                        value={op.value ?? 0}
                                        onChange={(e) => updateOperation(bIdx, opIdx, { value: parseFloat(e.target.value) || 0 })}
                                        className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200"
                                      />
                                    </div>
                                    <div>
                                      <label className="text-zinc-500 block text-[10px]">Số lượt (Turns)</label>
                                      <input
                                        type="number"
                                        min="1"
                                        value={op.turns ?? 2}
                                        onChange={(e) => updateOperation(bIdx, opIdx, { turns: parseInt(e.target.value) || 1 })}
                                        className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200"
                                      />
                                    </div>
                                    <div>
                                      <label className="text-zinc-500 block text-[10px]">Tỉ lệ (0..1)</label>
                                      <input
                                        type="number"
                                        step="0.05"
                                        min="0"
                                        max="1"
                                        value={op.chance ?? 1}
                                        onChange={(e) => updateOperation(bIdx, opIdx, { chance: parseFloat(e.target.value) || 0 })}
                                        className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200"
                                      />
                                    </div>
                                  </>
                                )}

                                {/* HEAL Fields */}
                                {op.op === "HEAL" && (
                                  <>
                                    <div>
                                      <label className="text-zinc-500 block text-[10px]">Loại Hồi Phục</label>
                                      <select
                                        value={op.sub_type ?? "HP"}
                                        onChange={(e) => updateOperation(bIdx, opIdx, { sub_type: e.target.value })}
                                        className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200"
                                      >
                                        <option value="HP">Sinh Lực (HP)</option>
                                        <option value="MANA">Chân Nguyên (MANA)</option>
                                        <option value="LIFESTEAL">Hút Máu (LIFESTEAL)</option>
                                      </select>
                                    </div>
                                    <div>
                                      <label className="text-zinc-500 block text-[10px]">Lượng hồi (Value)</label>
                                      <input
                                        type="number"
                                        step="any"
                                        value={op.value ?? 0}
                                        onChange={(e) => updateOperation(bIdx, opIdx, { value: parseFloat(e.target.value) || 0 })}
                                        className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200"
                                      />
                                    </div>
                                    <div>
                                      <label className="text-zinc-500 block text-[10px]">Tỉ lệ (0..1)</label>
                                      <input
                                        type="number"
                                        step="0.05"
                                        min="0"
                                        max="1"
                                        value={op.chance ?? 1}
                                        onChange={(e) => updateOperation(bIdx, opIdx, { chance: parseFloat(e.target.value) || 0 })}
                                        className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200"
                                      />
                                    </div>
                                  </>
                                )}

                                {/* DIRECT_HP Fields */}
                                {op.op === "DIRECT_HP" && (
                                  <>
                                    <div>
                                      <label className="text-zinc-500 block text-[10px]">Phân loại</label>
                                      <select
                                        value={op.sub_type ?? "HP_PERCENT"}
                                        onChange={(e) => updateOperation(bIdx, opIdx, { sub_type: e.target.value })}
                                        className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200"
                                      >
                                        <option value="HP_PERCENT">% Max HP</option>
                                        <option value="SHIELD_BREAK">Phá Giáp / Trừ Khiên</option>
                                      </select>
                                    </div>
                                    <div>
                                      <label className="text-zinc-500 block text-[10px]">Giá trị</label>
                                      <input
                                        type="number"
                                        step="any"
                                        value={op.value ?? 0}
                                        onChange={(e) => updateOperation(bIdx, opIdx, { value: parseFloat(e.target.value) || 0 })}
                                        className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200"
                                      />
                                    </div>
                                  </>
                                )}

                                {/* STAT_MOD Fields */}
                                {op.op === "STAT_MOD" && (
                                  <>
                                    <div>
                                      <label className="text-zinc-500 block text-[10px]">Chỉ số</label>
                                      <select
                                        value={op.stat ?? "ATTACK"}
                                        onChange={(e) => updateOperation(bIdx, opIdx, { stat: e.target.value })}
                                        className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200"
                                      >
                                        {STAT_OPTIONS.map((st) => (
                                          <option key={st.value} value={st.value}>
                                            {st.label}
                                          </option>
                                        ))}
                                      </select>
                                    </div>
                                    <div>
                                      <label className="text-zinc-500 block text-[10px]">Giá trị (+ / -)</label>
                                      <input
                                        type="number"
                                        step="any"
                                        value={op.value ?? 0}
                                        onChange={(e) => updateOperation(bIdx, opIdx, { value: parseFloat(e.target.value) || 0 })}
                                        className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200"
                                      />
                                    </div>
                                    <div>
                                      <label className="text-zinc-500 block text-[10px]">Số lượt (0 = Vĩnh viễn)</label>
                                      <input
                                        type="number"
                                        min="0"
                                        value={op.turns ?? 0}
                                        onChange={(e) => updateOperation(bIdx, opIdx, { turns: parseInt(e.target.value) || 0 })}
                                        className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200"
                                      />
                                    </div>
                                  </>
                                )}

                                {/* STATUS Fields */}
                                {op.op === "STATUS" && (
                                  <>
                                    <div>
                                      <label className="text-zinc-500 block text-[10px]">Mã Trạng Thái (status_id)</label>
                                      <input
                                        list={`common-statuses-${datalistId}`}
                                        value={op.status_id ?? ""}
                                        onChange={(e) => updateOperation(bIdx, opIdx, { status_id: e.target.value.toUpperCase() })}
                                        placeholder="FREEZE, STUN..."
                                        className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200 font-mono uppercase"
                                      />
                                    </div>
                                    <div>
                                      <label className="text-zinc-500 block text-[10px]">Số lượt (Turns)</label>
                                      <input
                                        type="number"
                                        min="1"
                                        value={op.turns ?? 2}
                                        onChange={(e) => updateOperation(bIdx, opIdx, { turns: parseInt(e.target.value) || 1 })}
                                        className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200"
                                      />
                                    </div>
                                    <div>
                                      <label className="text-zinc-500 block text-[10px]">Tỉ lệ (0..1)</label>
                                      <input
                                        type="number"
                                        step="0.05"
                                        min="0"
                                        max="1"
                                        value={op.chance ?? 1}
                                        onChange={(e) => updateOperation(bIdx, opIdx, { chance: parseFloat(e.target.value) || 0 })}
                                        className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200"
                                      />
                                    </div>
                                    <div>
                                      <label className="text-zinc-500 block text-[10px]">DoT / Giá trị tick (tùy chọn)</label>
                                      <input
                                        type="number"
                                        step="any"
                                        value={op.value ?? 0}
                                        onChange={(e) => updateOperation(bIdx, opIdx, { value: parseFloat(e.target.value) || 0 })}
                                        className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200"
                                      />
                                    </div>
                                  </>
                                )}

                                {/* BARRIER / CLEANSE / UNIQUE Fields */}
                                {(op.op === "BARRIER" || op.op === "CLEANSE" || op.op === "UNIQUE") && (
                                  <>
                                    <div>
                                      <label className="text-zinc-500 block text-[10px]">Mã cơ chế (Code)</label>
                                      <input
                                        value={op.code ?? ""}
                                        onChange={(e) => updateOperation(bIdx, opIdx, { code: e.target.value })}
                                        placeholder="Mã hành động..."
                                        className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200 font-mono"
                                      />
                                    </div>
                                    <div>
                                      <label className="text-zinc-500 block text-[10px]">Giá trị (Value)</label>
                                      <input
                                        type="number"
                                        step="any"
                                        value={op.value ?? 0}
                                        onChange={(e) => updateOperation(bIdx, opIdx, { value: parseFloat(e.target.value) || 0 })}
                                        className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200"
                                      />
                                    </div>
                                    {op.op !== "CLEANSE" && (
                                      <div>
                                        <label className="text-zinc-500 block text-[10px]">Số lượt (Turns)</label>
                                        <input
                                          type="number"
                                          min="0"
                                          value={op.turns ?? 0}
                                          onChange={(e) => updateOperation(bIdx, opIdx, { turns: parseInt(e.target.value) || 0 })}
                                          className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200"
                                        />
                                      </div>
                                    )}
                                  </>
                                )}
                              </div>
                            </div>
                          ))}
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>

          {/* Add Block Button */}
          <button
            type="button"
            onClick={addEmptyBlock}
            className="w-full py-2.5 rounded-xl border border-dashed border-amber-600/50 bg-amber-950/20 hover:bg-amber-950/40 text-amber-400 hover:text-amber-300 text-xs font-semibold flex items-center justify-center gap-1.5 transition-colors cursor-pointer"
          >
            <Plus className="w-4 h-4" />
            + Thêm Khối Hiệu Ứng (Composable Block)
          </button>

          {/* Growth & House Accordion */}
          <div className="pt-2 border-t border-zinc-800 space-y-2">
            <details className="bg-zinc-800/30 rounded-lg">
              <summary className="px-3 py-2 text-xs font-bold text-zinc-400 uppercase cursor-pointer flex items-center justify-between">
                <span>🌱 Nuôi Dưỡng (Growth)</span>
                <span className="text-[10px] text-zinc-500 font-mono">
                  {growth.hours_to_max ? `${growth.hours_to_max}h max` : "Chưa thiết lập"}
                </span>
              </summary>
              <div className="px-3 pb-3 grid grid-cols-2 sm:grid-cols-5 gap-2 pt-1 text-xs">
                <div>
                  <label className="text-zinc-500 block text-[10px]">Thời gian max (giờ)</label>
                  <input
                    type="number"
                    step="any"
                    value={growth.hours_to_max ?? ""}
                    onChange={(e) => updateGrowth("hours_to_max", parseFloat(e.target.value) || 0)}
                    placeholder="vd: 24"
                    className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-zinc-200 text-xs"
                  />
                </div>
                <div>
                  <label className="text-zinc-500 block text-[10px]">Công tối đa</label>
                  <input
                    type="number"
                    step="any"
                    value={growth.max_attack ?? ""}
                    onChange={(e) => updateGrowth("max_attack", parseFloat(e.target.value) || 0)}
                    className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-zinc-200 text-xs"
                  />
                </div>
                <div>
                  <label className="text-zinc-500 block text-[10px]">Thủ tối đa</label>
                  <input
                    type="number"
                    step="any"
                    value={growth.max_defense ?? ""}
                    onChange={(e) => updateGrowth("max_defense", parseFloat(e.target.value) || 0)}
                    className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-zinc-200 text-xs"
                  />
                </div>
                <div>
                  <label className="text-zinc-500 block text-[10px]">Máu tối đa</label>
                  <input
                    type="number"
                    step="any"
                    value={growth.max_hp ?? ""}
                    onChange={(e) => updateGrowth("max_hp", parseFloat(e.target.value) || 0)}
                    className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-zinc-200 text-xs"
                  />
                </div>
                <div>
                  <label className="text-zinc-500 block text-[10px]">Tốc tối đa</label>
                  <input
                    type="number"
                    step="any"
                    value={growth.max_speed ?? ""}
                    onChange={(e) => updateGrowth("max_speed", parseFloat(e.target.value) || 0)}
                    className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-zinc-200 text-xs"
                  />
                </div>
              </div>
            </details>

            {/* Legacy Data Notice if present */}
            {(legacyActive || legacyPassive || legacyOther) && (
              <details className="bg-zinc-800/30 rounded-lg">
                <summary className="px-3 py-2 text-xs font-bold text-amber-500/80 uppercase cursor-pointer flex items-center justify-between">
                  <span>⚠️ Dữ liệu hiệu ứng cũ (Legacy Active / Passive / Other)</span>
                  <span className="text-[10px] text-zinc-500 font-mono">Đã có Composable Blocks</span>
                </summary>
                <div className="px-3 pb-3 space-y-2 text-xs">
                  <p className="text-[11px] text-zinc-400 italic">
                    Cổ trùng này chứa dữ liệu dạng phẳng cũ. Hệ thống ưu tiên đọc <strong>blocks</strong>. Bạn có thể xóa các trường cũ này sau khi đã chuyển sang blocks đầy đủ.
                  </p>
                  <div className="flex gap-2">
                    <button
                      type="button"
                      onClick={() => {
                        const { active, passive, other, ...rest } = effects;
                        onChange(rest);
                      }}
                      className="px-3 py-1 bg-red-950/60 hover:bg-red-900 border border-red-800/60 rounded text-xs text-red-300 cursor-pointer"
                    >
                      Dọn dẹp (Xóa bỏ toàn bộ active/passive/other cũ)
                    </button>
                  </div>
                </div>
              </details>
            )}
          </div>
        </div>
      )}

      {/* Datalists for autocompletion */}
      <datalist id={`common-statuses-${datalistId}`}>
        {COMMON_STATUSES.map((st) => (
          <option key={st.id} value={st.id}>
            {st.label}
          </option>
        ))}
      </datalist>
      <datalist id={`dao-paths-${datalistId}`}>
        {DAO_PATHS.map((p) => (
          <option key={p} value={p}>
            {p}
          </option>
        ))}
      </datalist>
    </div>
  );
}
