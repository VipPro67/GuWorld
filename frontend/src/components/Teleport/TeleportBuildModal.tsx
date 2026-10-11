"use client";

import React, { useState, useEffect } from "react";
import { useGameStore, Player } from "@/store/useGameStore";
import { X, Sparkles, Compass, Shield, Coins, Layers, ArrowRightLeft, Award, AlertTriangle, CheckCircle2 } from "lucide-react";
import { GameButton } from "@/components/ui";

interface TeleportBuildModalProps {
  player: Player;
  onClose: () => void;
  onSuccess?: () => void;
}

export const getFormationInfo = (pts: number = 0) => {
  if (pts >= 100000) return { level: 4, name: "Vô Thượng Đại Tông Sư", color: "text-purple-400", border: "border-purple-500/40", bg: "bg-purple-950/30" };
  if (pts >= 10000) return { level: 3, name: "Đại Tông Sư", color: "text-amber-400", border: "border-amber-500/40", bg: "bg-amber-950/30" };
  if (pts >= 1000) return { level: 2, name: "Tông Sư", color: "text-cyan-400", border: "border-cyan-500/40", bg: "bg-cyan-950/30" };
  if (pts >= 100) return { level: 1, name: "Đại Sư", color: "text-blue-400", border: "border-blue-500/40", bg: "bg-blue-950/30" };
  return { level: 0, name: "Nhập Môn", color: "text-zinc-400", border: "border-zinc-700/40", bg: "bg-zinc-900/40" };
};

export const getFormationRealmName = (level: number) => {
  switch (level) {
    case 4: return "Vô Thượng Đại Tông Sư";
    case 3: return "Đại Tông Sư";
    case 2: return "Tông Sư";
    case 1: return "Đại Sư";
    default: return "Nhập Môn";
  }
};

export const TeleportBuildModal: React.FC<TeleportBuildModalProps> = ({
  player,
  onClose,
  onSuccess,
}) => {
  const { teleportTemplates, fetchTeleportTemplates, buildTeleportArray, actionLoading } = useGameStore();

  const [selectedTier, setSelectedTier] = useState<string>("TIER_1");
  const [arrayName, setArrayName] = useState<string>("");
  const [directionType, setDirectionType] = useState<string>("TWO_WAY");
  const [accessPolicy, setAccessPolicy] = useState<string>("PUBLIC");
  const [feeEnabled, setFeeEnabled] = useState<boolean>(false);
  const [feeCurrency, setFeeCurrency] = useState<string>("primeval_stones");
  const [feeAmount, setFeeAmount] = useState<number>(100);

  // Formation Path & Dao Mark calculations
  const formationPts = player.path_masteries?.["formation"] ?? player.path_masteries?.["formation path"] ?? 0;
  const daoMarks = player.dao_marks?.["formation"] ?? 0;
  const formationInfo = getFormationInfo(formationPts);
  const discountPercent = Math.min(60, formationInfo.level * 15);

  useEffect(() => {
    if (teleportTemplates.length === 0) {
      fetchTeleportTemplates();
    }
  }, [teleportTemplates.length, fetchTeleportTemplates]);

  const activeTpl = teleportTemplates.find((t) => t.tier === selectedTier) || teleportTemplates[0];

  const getDiscountedCost = (cost: number) => {
    if (discountPercent <= 0) return cost;
    return Math.max(1, Math.round(cost * (1 - discountPercent / 100)));
  };

  const calculateSuccessBreakdown = (tpl: typeof activeTpl) => {
    if (!tpl) return { base: 70, mastery: 0, rank: 0, dao: 0, total: 70 };
    const base = tpl.base_success_rate ?? 70;
    const mastery = formationInfo.level * 15;
    const rank = Math.max(0, player.rank - tpl.min_rank) * 3;
    const dao = Math.min(10, Math.floor(daoMarks / 500));
    const total = Math.min(100, Math.max(10, base + mastery + rank + dao));
    return { base, mastery, rank, dao, total };
  };

  const activeBreakdown = calculateSuccessBreakdown(activeTpl);
  const activeDiscountedStones = activeTpl ? getDiscountedCost(activeTpl.cost_primeval_stones) : 0;
  const activeDiscountedImmortal = activeTpl && activeTpl.cost_immortal_stones > 0 ? getDiscountedCost(activeTpl.cost_immortal_stones) : 0;

  const canAffordActiveStones = activeTpl ? player.primeval_stones >= activeDiscountedStones : false;
  const canAffordActiveImmortal = activeTpl ? player.immortal_stones >= activeDiscountedImmortal : false;
  const meetsActiveRank = activeTpl ? player.rank >= activeTpl.min_rank : false;
  const meetsActiveFormation = activeTpl ? formationInfo.level >= (activeTpl.min_formation_realm ?? 0) : true;
  const canBuildActive = activeTpl && canAffordActiveStones && canAffordActiveImmortal && meetsActiveRank && meetsActiveFormation;

  const handleBuild = async () => {
    if (!activeTpl) return;
    const res = await buildTeleportArray({
      name: arrayName.trim() || activeTpl.name,
      tier: selectedTier,
      direction_type: directionType,
      access_policy: accessPolicy,
      fee_enabled: feeEnabled,
      fee_currency: feeCurrency,
      fee_amount: Number(feeAmount) || 0,
    });
    if (res && res.success) {
      if (onSuccess) onSuccess();
      onClose();
    }
  };

  return (
    <div className="fixed inset-0 z-[9999] h-[100dvh] flex items-center justify-center px-2 pt-[calc(env(safe-area-inset-top)+0.5rem)] pb-[calc(env(safe-area-inset-bottom)+0.5rem)] sm:px-4">
      <div className="absolute inset-0 bg-black/75 backdrop-blur-md" onClick={onClose} />
      <div className="relative bg-zinc-950 border border-amber-500/30 rounded-3xl shadow-2xl w-full max-w-xl max-h-[calc(100dvh-env(safe-area-inset-top)-env(safe-area-inset-bottom)-1rem)] flex flex-col overflow-hidden animate-in fade-in zoom-in-95 duration-200 text-left">
        {/* Header */}
        <div className="shrink-0 flex items-center justify-between border-b border-zinc-800/80 px-4 sm:px-6 py-3 sm:py-4 bg-gradient-to-r from-amber-950/20 via-zinc-900/40 to-transparent">
          <div className="flex items-center gap-2.5">
            <div className="p-2 rounded-xl bg-amber-500/10 border border-amber-500/30 text-amber-400">
              <Compass className="w-5 h-5 animate-spin" />
            </div>
            <div>
              <h3 className="text-base font-bold text-zinc-100 font-serif">Bố Trí Truyền Tống Trận</h3>
              <p className="text-[11px] text-zinc-400 font-mono">
                Tọa độ: ({player.x}, {player.y}) — Vực: {player.current_region}
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-xl text-zinc-500 hover:text-zinc-200 hover:bg-zinc-900 transition-colors cursor-pointer"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Formation Mastery Status Banner */}
        <div className="shrink-0 px-4 sm:px-6 py-2.5 bg-zinc-900/60 border-b border-zinc-800/80 flex flex-wrap items-center justify-between gap-2 text-xs font-mono">
          <div className="flex items-center gap-2">
            <span className="text-zinc-400 flex items-center gap-1">
              <Award className="w-3.5 h-3.5 text-amber-400" />
              Trận Đạo:
            </span>
            <span className={`px-2 py-0.5 rounded-md font-bold text-[11px] border ${formationInfo.bg} ${formationInfo.border} ${formationInfo.color}`}>
              {formationInfo.name} ({formationPts.toLocaleString()} Chân Ý)
            </span>
            {daoMarks > 0 && (
              <span className="text-zinc-400 text-[10px]">
                · {daoMarks.toLocaleString()} Đạo Ngân
              </span>
            )}
          </div>
          {discountPercent > 0 && (
            <div className="text-emerald-400 font-bold text-[11px] flex items-center gap-1">
              <Sparkles className="w-3 h-3" />
              Giảm {discountPercent}% Chi Phí
            </div>
          )}
        </div>

        <div className="p-4 sm:p-6 space-y-4 sm:space-y-5 flex-1 min-h-0 overflow-y-auto custom-scrollbar">
          {/* Tier Selection */}
          <div className="space-y-2">
            <label className="text-xs font-bold text-zinc-300 uppercase tracking-wider font-mono flex items-center gap-1.5">
              <Layers className="w-3.5 h-3.5 text-amber-400" />
              Chọn Phẩm Cấp Trận Pháp
            </label>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-2.5">
              {teleportTemplates.map((tpl) => {
                const isSelected = selectedTier === tpl.tier;
                const discStones = getDiscountedCost(tpl.cost_primeval_stones);
                const discImmortal = tpl.cost_immortal_stones > 0 ? getDiscountedCost(tpl.cost_immortal_stones) : 0;

                const canAffordStones = player.primeval_stones >= discStones;
                const canAffordImmortal = player.immortal_stones >= discImmortal;
                const meetsRank = player.rank >= tpl.min_rank;
                const minRealm = tpl.min_formation_realm ?? 0;
                const meetsFormation = formationInfo.level >= minRealm;
                const breakdown = calculateSuccessBreakdown(tpl);

                return (
                  <button
                    key={tpl.tier}
                    type="button"
                    onClick={() => setSelectedTier(tpl.tier)}
                    className={`p-3 rounded-2xl border text-left transition-all relative flex flex-col justify-between ${
                      isSelected
                        ? "bg-amber-950/25 border-amber-500/60 shadow-[0_0_15px_rgba(245,158,11,0.15)] ring-1 ring-amber-500/30"
                        : "bg-zinc-900/40 border-zinc-800/80 hover:border-zinc-700 hover:bg-zinc-900/70"
                    }`}
                  >
                    <div>
                      <div className="flex items-center justify-between gap-1 flex-wrap">
                        <span className="text-[10px] font-mono uppercase text-amber-400/90 font-bold">
                          {tpl.tier === "TIER_1" ? "Vi Trận" : tpl.tier === "TIER_2" ? "Đại Vực Trận" : "Tiên Trận"}
                        </span>
                        <div className="flex items-center gap-1">
                          {!meetsRank && (
                            <span className="text-[9px] font-mono text-rose-400 bg-rose-950/40 px-1 rounded border border-rose-900/50">
                              Rank {tpl.min_rank}
                            </span>
                          )}
                          {!meetsFormation && (
                            <span className="text-[9px] font-mono text-rose-400 bg-rose-950/40 px-1 rounded border border-rose-900/50">
                              {getFormationRealmName(minRealm)}
                            </span>
                          )}
                        </div>
                      </div>
                      <h4 className="text-xs font-bold text-zinc-100 font-serif mt-1">{tpl.name}</h4>
                      <p className="text-[10px] text-zinc-400 mt-1 leading-relaxed line-clamp-2">{tpl.description}</p>
                    </div>

                    <div className="mt-3 pt-2 border-t border-zinc-800/60 space-y-1 text-[10px] font-mono">
                      <div className="flex justify-between items-center text-zinc-400">
                        <span>Phạm vi:</span>
                        <span className="text-emerald-400 font-bold">{tpl.is_cross_region ? "Ngũ Vực" : `${tpl.max_range.toLocaleString()} ô`}</span>
                      </div>
                      
                      <div className="flex justify-between items-center">
                        <span className="text-zinc-400">Tỉ lệ thành công:</span>
                        <span className={`font-bold ${breakdown.total >= 80 ? "text-emerald-400" : breakdown.total >= 50 ? "text-amber-400" : "text-rose-400"}`}>
                          {breakdown.total}%
                        </span>
                      </div>

                      <div className="pt-1 border-t border-zinc-800/40">
                        <div className={`flex justify-between items-center ${canAffordStones ? "text-amber-300" : "text-rose-400"}`}>
                          <span>Nguyên Thạch:</span>
                          <span className="flex items-center gap-1">
                            {discountPercent > 0 && (
                              <span className="line-through text-zinc-500 text-[9px]">{tpl.cost_primeval_stones.toLocaleString()}</span>
                            )}
                            <span className="font-bold">{discStones.toLocaleString()}</span>
                          </span>
                        </div>
                        {tpl.cost_immortal_stones > 0 && (
                          <div className={`flex justify-between items-center ${canAffordImmortal ? "text-cyan-300" : "text-rose-400"}`}>
                            <span>Tiên Nguyên:</span>
                            <span className="flex items-center gap-1">
                              {discountPercent > 0 && (
                                <span className="line-through text-zinc-500 text-[9px]">{tpl.cost_immortal_stones}</span>
                              )}
                              <span className="font-bold">{discImmortal}</span>
                            </span>
                          </div>
                        )}
                      </div>
                    </div>
                  </button>
                );
              })}
            </div>
          </div>

          {/* Success Rate & Formation Details Card */}
          {activeTpl && (
            <div className="p-3.5 bg-zinc-900/50 border border-zinc-800/80 rounded-2xl space-y-2.5 font-mono text-xs">
              <div className="flex items-center justify-between">
                <span className="text-zinc-300 font-bold flex items-center gap-1.5">
                  <CheckCircle2 className={`w-4 h-4 ${activeBreakdown.total >= 80 ? "text-emerald-400" : activeBreakdown.total >= 50 ? "text-amber-400" : "text-rose-400"}`} />
                  Tỉ Lệ Thành Công Dự Kiến: {activeBreakdown.total}%
                </span>
                <span className="text-[10px] text-zinc-400">
                  +{activeTpl.tier === "TIER_3" ? "300" : activeTpl.tier === "TIER_2" ? "100" : "30"} Chân Ý Trận Đạo
                </span>
              </div>

              {/* Progress Bar */}
              <div className="w-full h-2 bg-zinc-800 rounded-full overflow-hidden">
                <div
                  className={`h-full transition-all duration-300 ${
                    activeBreakdown.total >= 80 ? "bg-emerald-500" : activeBreakdown.total >= 50 ? "bg-amber-500" : "bg-rose-500"
                  }`}
                  style={{ width: `${activeBreakdown.total}%` }}
                />
              </div>

              {/* Breakdown details */}
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-1.5 text-[10px] text-zinc-400">
                <div className="bg-zinc-950/60 p-1.5 rounded-lg border border-zinc-800 text-center">
                  Gốc: <span className="text-zinc-200 font-bold">{activeBreakdown.base}%</span>
                </div>
                <div className="bg-zinc-950/60 p-1.5 rounded-lg border border-zinc-800 text-center">
                  Trận Đạo: <span className="text-emerald-400 font-bold">+{activeBreakdown.mastery}%</span>
                </div>
                <div className="bg-zinc-950/60 p-1.5 rounded-lg border border-zinc-800 text-center">
                  Tu Vi: <span className="text-cyan-400 font-bold">+{activeBreakdown.rank}%</span>
                </div>
                <div className="bg-zinc-950/60 p-1.5 rounded-lg border border-zinc-800 text-center">
                  Đạo Ngân: <span className="text-purple-400 font-bold">+{activeBreakdown.dao}%</span>
                </div>
              </div>

              {activeBreakdown.total < 100 && (
                <div className="flex items-start gap-1.5 p-2 rounded-xl bg-amber-950/20 border border-amber-900/40 text-[10px] text-amber-300/90 leading-relaxed">
                  <AlertTriangle className="w-3.5 h-3.5 text-amber-400 shrink-0 mt-0.5" />
                  <span>
                    Nếu sụp đổ trận pháp khi bố trí: tổn thất 50% Nguyên thạch / Tiên nguyên, Cổ trùng <strong>KHÔNG</strong> bị tổn hại và vẫn lĩnh ngộ 50% Chân Ý.
                  </span>
                </div>
              )}
            </div>
          )}

          {/* Form Settings */}
          <div className="space-y-4 bg-zinc-900/30 border border-zinc-800/60 p-4 rounded-2xl">
            <div>
              <label className="text-xs font-mono text-zinc-300 block mb-1">Tên Trận Pháp</label>
              <input
                type="text"
                value={arrayName}
                onChange={(e) => setArrayName(e.target.value)}
                placeholder={activeTpl?.name || "Nhập tên trận pháp..."}
                maxLength={40}
                className="w-full bg-zinc-900 border border-zinc-700/80 rounded-xl px-3.5 py-2 text-xs text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-amber-500 font-mono"
              />
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              {/* Direction Type */}
              <div>
                <label className="text-xs font-mono text-zinc-300 block mb-1 flex items-center gap-1">
                  <ArrowRightLeft className="w-3.5 h-3.5 text-blue-400" />
                  Hướng Kết Nối
                </label>
                <select
                  value={directionType}
                  onChange={(e) => setDirectionType(e.target.value)}
                  className="w-full bg-zinc-900 border border-zinc-700/80 rounded-xl px-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500 font-mono"
                >
                  <option value="TWO_WAY">Hai Chiều (Gửi & Tiếp Nhận)</option>
                  <option value="ONE_WAY_SEND">Một Chiều (Chỉ Gửi Đi)</option>
                  <option value="ONE_WAY_RECEIVE">Một Chiều (Chỉ Tiếp Nhận)</option>
                </select>
              </div>

              {/* Access Policy */}
              <div>
                <label className="text-xs font-mono text-zinc-300 block mb-1 flex items-center gap-1">
                  <Shield className="w-3.5 h-3.5 text-purple-400" />
                  Quyền Truy Cập
                </label>
                <select
                  value={accessPolicy}
                  onChange={(e) => setAccessPolicy(e.target.value)}
                  className="w-full bg-zinc-900 border border-zinc-700/80 rounded-xl px-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500 font-mono"
                >
                  <option value="PUBLIC">Công Cộng (Mọi người dùng)</option>
                  <option value="SECT">Tông Môn (Chỉ nội bộ Tông)</option>
                  <option value="PERSONAL">Riêng Tư (Chỉ bản thân)</option>
                </select>
              </div>
            </div>

            {/* Fee Settings */}
            <div className="border-t border-zinc-800/80 pt-3 space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-xs font-mono text-zinc-300 flex items-center gap-1.5">
                  <Coins className="w-3.5 h-3.5 text-amber-400" />
                  Thu Phí Dịch Chuyển Người Khác
                </span>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={feeEnabled}
                    onChange={(e) => setFeeEnabled(e.target.checked)}
                    className="sr-only peer"
                  />
                  <div className="w-9 h-5 bg-zinc-800 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-amber-600"></div>
                </label>
              </div>

              {feeEnabled && (
                <div className="grid grid-cols-2 gap-3 pt-1 animate-in fade-in duration-150">
                  <div>
                    <label className="text-[11px] font-mono text-zinc-400 block mb-1">Mức Phí/Lượt</label>
                    <input
                      type="number"
                      min={0}
                      value={feeAmount}
                      onChange={(e) => setFeeAmount(Math.max(0, parseInt(e.target.value) || 0))}
                      className="w-full bg-zinc-900 border border-zinc-700/80 rounded-xl px-3 py-1.5 text-xs text-zinc-100 font-mono"
                    />
                  </div>
                  <div>
                    <label className="text-[11px] font-mono text-zinc-400 block mb-1">Loại Tiền Tệ</label>
                    <select
                      value={feeCurrency}
                      onChange={(e) => setFeeCurrency(e.target.value)}
                      className="w-full bg-zinc-900 border border-zinc-700/80 rounded-xl px-3 py-1.5 text-xs text-zinc-200 font-mono"
                    >
                      <option value="primeval_stones">Nguyên Thạch</option>
                      <option value="immortal_stones">Tiên Nguyên Thạch</option>
                    </select>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Footer Actions */}
        <div className="shrink-0 border-t border-zinc-800/80 px-4 sm:px-6 py-3 sm:py-4 flex items-center justify-between gap-3 bg-zinc-950/90 backdrop-blur-sm">
          <div className="text-[11px] font-mono text-zinc-400">
            {!meetsActiveRank ? (
              <span className="text-rose-400">Yêu cầu Rank {activeTpl?.min_rank}</span>
            ) : !meetsActiveFormation ? (
              <span className="text-rose-400">Yêu cầu Trận Đạo {getFormationRealmName(activeTpl?.min_formation_realm ?? 0)}</span>
            ) : !canAffordActiveStones || !canAffordActiveImmortal ? (
              <span className="text-rose-400">Không đủ tài nguyên bố trận</span>
            ) : (
              <span className="text-emerald-400">Đủ điều kiện bố trận</span>
            )}
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 rounded-xl text-xs font-mono text-zinc-400 hover:bg-zinc-900 border border-zinc-800 transition-colors cursor-pointer"
            >
              Hủy Bỏ
            </button>
            <GameButton
              onClick={handleBuild}
              variant="primary"
              disabled={actionLoading || !canBuildActive}
              className="px-6 py-2 text-xs font-bold font-serif cursor-pointer"
            >
              {actionLoading ? "Đang Bố Trí..." : "Bố Trí Trận Pháp"}
            </GameButton>
          </div>
        </div>
      </div>
    </div>
  );
};

