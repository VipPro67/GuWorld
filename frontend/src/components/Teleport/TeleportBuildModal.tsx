"use client";

import React, { useState, useEffect } from "react";
import { useGameStore, Player } from "@/store/useGameStore";
import { X, Sparkles, Compass, Shield, Coins, Layers, ArrowRightLeft, Lock, Users, Globe } from "lucide-react";
import { GameButton } from "@/components/ui";

interface TeleportBuildModalProps {
  player: Player;
  onClose: () => void;
  onSuccess?: () => void;
}

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

  useEffect(() => {
    if (teleportTemplates.length === 0) {
      fetchTeleportTemplates();
    }
  }, [teleportTemplates.length, fetchTeleportTemplates]);

  const activeTpl = teleportTemplates.find((t) => t.tier === selectedTier) || teleportTemplates[0];

  const handleBuild = async () => {
    if (!activeTpl) return;
    const ok = await buildTeleportArray({
      name: arrayName.trim() || activeTpl.name,
      tier: selectedTier,
      direction_type: directionType,
      access_policy: accessPolicy,
      fee_enabled: feeEnabled,
      fee_currency: feeCurrency,
      fee_amount: Number(feeAmount) || 0,
    });
    if (ok) {
      if (onSuccess) onSuccess();
      onClose();
    }
  };

  return (
    <div className="fixed inset-0 z-[9999] flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/75 backdrop-blur-md" onClick={onClose} />
      <div className="relative bg-zinc-950 border border-amber-500/30 rounded-3xl shadow-2xl w-full max-w-xl overflow-hidden animate-in fade-in zoom-in-95 duration-200 text-left">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-zinc-800/80 px-6 py-4 bg-gradient-to-r from-amber-950/20 via-zinc-900/40 to-transparent">
          <div className="flex items-center gap-2.5">
            <div className="p-2 rounded-xl bg-amber-500/10 border border-amber-500/30 text-amber-400">
              <Compass className="w-5 h-5 animate-spin" />
            </div>
            <div>
              <h3 className="text-base font-bold text-zinc-100 font-serif">Bố Trí Truyền Tống Trận</h3>
              <p className="text-[11px] text-zinc-400 font-mono">
                Tọa độ hiện tại: ({player.x}, {player.y}) — Vực: {player.current_region}
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-xl text-zinc-500 hover:text-zinc-200 hover:bg-zinc-900 transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-6 space-y-5 max-h-[75vh] overflow-y-auto custom-scrollbar">
          {/* Tier Selection */}
          <div className="space-y-2">
            <label className="text-xs font-bold text-zinc-300 uppercase tracking-wider font-mono flex items-center gap-1.5">
              <Layers className="w-3.5 h-3.5 text-amber-400" />
              Chọn Phẩm Cấp Trận Pháp
            </label>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-2.5">
              {teleportTemplates.map((tpl) => {
                const isSelected = selectedTier === tpl.tier;
                const canAffordStones = player.primeval_stones >= tpl.cost_primeval_stones;
                const canAffordImmortal = player.immortal_stones >= tpl.cost_immortal_stones;
                const meetsRank = player.rank >= tpl.min_rank;
                const isReady = canAffordStones && canAffordImmortal && meetsRank;

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
                      <div className="flex items-center justify-between">
                        <span className="text-[10px] font-mono uppercase text-amber-400/90 font-bold">
                          {tpl.tier === "TIER_1" ? "Vi Trận" : tpl.tier === "TIER_2" ? "Đại Vực Trận" : "Tiên Trận"}
                        </span>
                        {!meetsRank && (
                          <span className="text-[9px] font-mono text-rose-400 bg-rose-950/40 px-1 rounded border border-rose-900/50">
                            Cần Rank {tpl.min_rank}
                          </span>
                        )}
                      </div>
                      <h4 className="text-xs font-bold text-zinc-100 font-serif mt-1">{tpl.name}</h4>
                      <p className="text-[10px] text-zinc-400 mt-1 leading-relaxed line-clamp-2">{tpl.description}</p>
                    </div>

                    <div className="mt-3 pt-2 border-t border-zinc-800/60 space-y-0.5 text-[10px] font-mono">
                      <div className="text-zinc-400">
                        Phạm vi: <span className="text-emerald-400 font-bold">{tpl.is_cross_region ? "Ngũ Vực" : `${tpl.max_range.toLocaleString()} ô`}</span>
                      </div>
                      <div className={canAffordStones ? "text-amber-300" : "text-rose-400"}>
                        {tpl.cost_primeval_stones.toLocaleString()} Nguyên Thạch
                      </div>
                      {tpl.cost_immortal_stones > 0 && (
                        <div className={canAffordImmortal ? "text-cyan-300" : "text-rose-400"}>
                          {tpl.cost_immortal_stones} Tiên Nguyên
                        </div>
                      )}
                    </div>
                  </button>
                );
              })}
            </div>
          </div>

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
        <div className="border-t border-zinc-800/80 px-6 py-4 flex items-center justify-end gap-3 bg-zinc-950/80">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 rounded-xl text-xs font-mono text-zinc-400 hover:bg-zinc-900 border border-zinc-800 transition-colors"
          >
            Hủy Bỏ
          </button>
          <GameButton
            onClick={handleBuild}
            variant="primary"
            disabled={actionLoading || !activeTpl || player.rank < activeTpl.min_rank || player.primeval_stones < activeTpl.cost_primeval_stones || player.immortal_stones < activeTpl.cost_immortal_stones}
            className="px-6 py-2 text-xs font-bold font-serif"
          >
            {actionLoading ? "Đang Bố Trí..." : "Bố Trí Trận Pháp"}
          </GameButton>
        </div>
      </div>
    </div>
  );
};
