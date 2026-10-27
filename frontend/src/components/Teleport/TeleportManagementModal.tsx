"use client";

import React, { useState } from "react";
import { useGameStore, Player } from "@/store/useGameStore";
import { TeleportArray } from "@/store/slices/teleportSlice";
import { X, Settings, Coins, Wrench, Trash2, Shield, ArrowRightLeft, Check, AlertTriangle, Award, Sparkles } from "lucide-react";
import { GameButton } from "@/components/ui";
import { getFormationInfo } from "./TeleportBuildModal";

interface TeleportManagementModalProps {
  player: Player;
  array: TeleportArray;
  onClose: () => void;
  onUpdated?: () => void;
}

export const TeleportManagementModal: React.FC<TeleportManagementModalProps> = ({
  player,
  array,
  onClose,
  onUpdated,
}) => {
  const {
    manageTeleportArray,
    repairTeleportArray,
    withdrawTeleportVault,
    dismantleTeleportArray,
    actionLoading,
    showConfirm,
  } = useGameStore();

  const [name, setName] = useState<string>(array.name);
  const [directionType, setDirectionType] = useState<string>(array.direction_type);
  const [accessPolicy, setAccessPolicy] = useState<string>(array.access_policy);
  const [feeEnabled, setFeeEnabled] = useState<boolean>(array.fee_enabled);
  const [feeCurrency, setFeeCurrency] = useState<string>(array.fee_currency || "primeval_stones");
  const [feeAmount, setFeeAmount] = useState<number>(array.fee_amount || 0);

  // Formation Path & Dao Mark calculations
  const formationPts = player.path_masteries?.["formation"] ?? player.path_masteries?.["formation path"] ?? 0;
  const formationInfo = getFormationInfo(formationPts);
  const repairDiscount = Math.min(50, Math.round(formationInfo.level * 12.5));
  const repairCritChance = formationInfo.level * 10;

  const durPercent = Math.min(100, Math.round((array.current_durability / array.max_durability) * 100));

  const handleSaveSettings = async () => {
    const ok = await manageTeleportArray({
      array_id: array.id,
      name: name.trim() || array.name,
      direction_type: directionType,
      access_policy: accessPolicy,
      fee_enabled: feeEnabled,
      fee_currency: feeCurrency,
      fee_amount: Number(feeAmount) || 0,
    });
    if (ok && onUpdated) onUpdated();
  };

  const handleRepair = async () => {
    const missing = array.max_durability - array.current_durability;
    if (missing <= 0) return;
    const ok = await repairTeleportArray(array.id, missing);
    if (ok && onUpdated) onUpdated();
  };

  const handleWithdraw = async () => {
    if (array.vault_balance <= 0) return;
    const ok = await withdrawTeleportVault(array.id, 0);
    if (ok && onUpdated) onUpdated();
  };

  const handleDismantle = async () => {
    const confirmed = await showConfirm(
      `Đạo hữu có chắc chắn muốn THU HỒI [${array.name}]? Trận pháp sẽ bị tháo dỡ và hoàn lại một phần nguyên liệu cùng toàn bộ số dư Két Sắt.`
    );
    if (confirmed) {
      const ok = await dismantleTeleportArray(array.id);
      if (ok) {
        if (onUpdated) onUpdated();
        onClose();
      }
    }
  };

  return (
    <div className="pointer-events-auto fixed inset-0 z-[9999] h-[100dvh] flex items-center justify-center px-2 pt-[calc(env(safe-area-inset-top)+0.5rem)] pb-[calc(env(safe-area-inset-bottom)+0.5rem)] sm:px-4">
      <div className="absolute inset-0 bg-black/75 backdrop-blur-md" onClick={onClose} />
      <div className="relative bg-zinc-950 border border-amber-500/30 rounded-3xl shadow-2xl w-full max-w-lg max-h-[calc(100dvh-env(safe-area-inset-top)-env(safe-area-inset-bottom)-1rem)] flex flex-col overflow-hidden animate-in fade-in zoom-in-95 duration-200 text-left">
        {/* Header */}
        <div className="shrink-0 flex items-center justify-between border-b border-zinc-800/80 px-4 sm:px-6 py-3 sm:py-4 bg-gradient-to-r from-amber-950/20 via-zinc-900/40 to-transparent">
          <div className="flex items-center gap-2.5">
            <div className="p-2 rounded-xl bg-amber-500/10 border border-amber-500/30 text-amber-400">
              <Settings className="w-5 h-5 animate-spin" />
            </div>
            <div>
              <h3 className="text-base font-bold text-zinc-100 font-serif">Quản Lý Truyền Tống Trận</h3>
              <p className="text-[11px] text-zinc-400 font-mono">
                {array.name} ({array.x}, {array.y}) · Vực {array.region_id}
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

        {/* Formation Status Bar */}
        <div className="shrink-0 px-4 sm:px-6 py-2 bg-zinc-900/60 border-b border-zinc-800/80 flex flex-wrap items-center justify-between gap-2 text-xs font-mono">
          <div className="flex items-center gap-2">
            <span className="text-zinc-400 flex items-center gap-1">
              <Award className="w-3.5 h-3.5 text-amber-400" />
              Trận Đạo:
            </span>
            <span className={`px-2 py-0.5 rounded-md font-bold text-[11px] border ${formationInfo.bg} ${formationInfo.border} ${formationInfo.color}`}>
              {formationInfo.name}
            </span>
          </div>
          {repairDiscount > 0 && (
            <div className="text-emerald-400 font-bold text-[11px] flex items-center gap-1">
              <Sparkles className="w-3 h-3" />
              Giảm {repairDiscount}% Chi Phí Sửa
            </div>
          )}
        </div>

        <div className="p-4 sm:p-6 space-y-4 sm:space-y-5 flex-1 min-h-0 overflow-y-auto custom-scrollbar">
          {/* Status & Vault overview */}
          <div className="grid grid-cols-2 gap-3">
            {/* Durability Card */}
            <div className="p-3.5 rounded-2xl bg-zinc-900/50 border border-zinc-800 space-y-2">
              <div className="flex items-center justify-between text-xs font-mono">
                <span className="text-zinc-400 flex items-center gap-1">
                  <Wrench className="w-3.5 h-3.5 text-blue-400" />
                  Độ Bền Trận
                </span>
                <span className="font-bold text-zinc-200">
                  {array.current_durability}/{array.max_durability}
                </span>
              </div>
              <div className="w-full h-2 bg-zinc-800 rounded-full overflow-hidden">
                <div
                  className={`h-full transition-all ${
                    durPercent > 50 ? "bg-emerald-500" : durPercent > 20 ? "bg-amber-500" : "bg-rose-500"
                  }`}
                  style={{ width: `${durPercent}%` }}
                />
              </div>
              {array.current_durability < array.max_durability && (
                <div className="space-y-1">
                  <button
                    type="button"
                    onClick={handleRepair}
                    disabled={actionLoading}
                    className="w-full py-1.5 mt-1 rounded-xl bg-blue-950/40 border border-blue-800/60 text-blue-300 hover:bg-blue-900/60 text-[11px] font-mono font-bold transition-all flex items-center justify-center gap-1"
                  >
                    <Wrench className="w-3 h-3" />
                    Sửa Chữa (+{array.max_durability - array.current_durability})
                  </button>
                  {repairCritChance > 0 && (
                    <div className="text-[10px] font-mono text-cyan-400/90 text-center">
                      ✨ {repairCritChance}% tỉ lệ Bạo Kích Sửa Chữa
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* Vault Balance Card */}
            <div className="p-3.5 rounded-2xl bg-zinc-900/50 border border-zinc-800 space-y-2 flex flex-col justify-between">
              <div>
                <div className="flex items-center justify-between text-xs font-mono">
                  <span className="text-zinc-400 flex items-center gap-1">
                    <Coins className="w-3.5 h-3.5 text-amber-400" />
                    Két Sắt Doanh Thu
                  </span>
                </div>
                <div className="text-sm font-bold font-mono text-amber-300 mt-1">
                  {array.vault_balance.toLocaleString()} {array.fee_currency === "immortal_stones" ? "Tiên Nguyên" : "Nguyên Thạch"}
                </div>
              </div>
              <button
                type="button"
                onClick={handleWithdraw}
                disabled={actionLoading || array.vault_balance <= 0}
                className={`w-full py-1.5 rounded-xl border text-[11px] font-mono font-bold transition-all flex items-center justify-center gap-1 ${
                  array.vault_balance > 0
                    ? "bg-amber-950/40 border-amber-800/60 text-amber-300 hover:bg-amber-900/60"
                    : "bg-zinc-900 text-zinc-600 border-zinc-800 cursor-not-allowed"
                }`}
              >
                <Coins className="w-3 h-3" />
                Rút Về Túi Đồ
              </button>
            </div>
          </div>

          {/* Form Settings */}
          <div className="space-y-4 bg-zinc-900/30 border border-zinc-800/60 p-4 rounded-2xl">
            <div>
              <label className="text-xs font-mono text-zinc-300 block mb-1">Tên Trận Pháp</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                maxLength={40}
                className="w-full bg-zinc-900 border border-zinc-700/80 rounded-xl px-3.5 py-2 text-xs text-zinc-100 font-mono focus:outline-none focus:border-amber-500"
              />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="text-xs font-mono text-zinc-300 block mb-1 flex items-center gap-1">
                  <ArrowRightLeft className="w-3.5 h-3.5 text-blue-400" />
                  Hướng Kết Nối
                </label>
                <select
                  value={directionType}
                  onChange={(e) => setDirectionType(e.target.value)}
                  className="w-full bg-zinc-900 border border-zinc-700/80 rounded-xl px-3 py-2 text-xs text-zinc-200 font-mono focus:outline-none focus:border-amber-500"
                >
                  <option value="TWO_WAY">Hai Chiều</option>
                  <option value="ONE_WAY_SEND">Chỉ Gửi Đi</option>
                  <option value="ONE_WAY_RECEIVE">Chỉ Tiếp Nhận</option>
                </select>
              </div>

              <div>
                <label className="text-xs font-mono text-zinc-300 block mb-1 flex items-center gap-1">
                  <Shield className="w-3.5 h-3.5 text-purple-400" />
                  Quyền Truy Cập
                </label>
                <select
                  value={accessPolicy}
                  onChange={(e) => setAccessPolicy(e.target.value)}
                  className="w-full bg-zinc-900 border border-zinc-700/80 rounded-xl px-3 py-2 text-xs text-zinc-200 font-mono focus:outline-none focus:border-amber-500"
                >
                  <option value="PUBLIC">Công Cộng</option>
                  <option value="SECT">Tông Môn</option>
                  <option value="PERSONAL">Riêng Tư</option>
                </select>
              </div>
            </div>

            {/* Fee Switch & Pricing */}
            <div className="border-t border-zinc-800/80 pt-3 space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-xs font-mono text-zinc-300 flex items-center gap-1.5">
                  <Coins className="w-3.5 h-3.5 text-amber-400" />
                  Thu Phí Dịch Chuyển
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
                <div className="grid grid-cols-2 gap-3 pt-1">
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
                    <label className="text-[11px] font-mono text-zinc-400 block mb-1">Loại Tiền</label>
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

            <GameButton
              onClick={handleSaveSettings}
              variant="secondary"
              disabled={actionLoading}
              className="w-full py-2 text-xs font-mono font-bold flex items-center justify-center gap-1.5"
            >
              <Check className="w-3.5 h-3.5 text-emerald-400" />
              Lưu Cấu Hình
            </GameButton>
          </div>

          {/* Danger Zone: Dismantle */}
          <div className="border border-rose-950/60 bg-rose-950/10 p-4 rounded-2xl space-y-2">
            <div className="flex items-center gap-1.5 text-xs font-bold text-rose-400 font-mono">
              <AlertTriangle className="w-4 h-4" />
              Khu Vực Nguy Hiểm
            </div>
            <p className="text-[10px] text-zinc-400 leading-relaxed font-mono">
              Thu hồi trận pháp sẽ hoàn trả 50% nguyên liệu chế tạo (tỉ lệ theo % độ bền hiện tại) và rút toàn bộ số dư Két Sắt.
            </p>
            <button
              type="button"
              onClick={handleDismantle}
              disabled={actionLoading}
              className="w-full py-2 rounded-xl bg-rose-950/50 border border-rose-800/80 text-rose-300 hover:bg-rose-900/60 hover:text-white text-xs font-mono font-bold transition-all flex items-center justify-center gap-1.5"
            >
              <Trash2 className="w-3.5 h-3.5" />
              Thu Hồi & Phá Dỡ Trận Pháp
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
