"use client";

import React, { useEffect, useState } from "react";
import { useGameStore, Player } from "@/store/useGameStore";
import { X, Compass, MapPin, Sparkles, AlertCircle, Coins, Shield, ArrowRight, Settings, RefreshCw } from "lucide-react";
import { GameButton } from "@/components/ui";

interface TeleportDirectoryModalProps {
  player: Player;
  onClose: () => void;
  onOpenManage?: (arrayId: string) => void;
}

export const TeleportDirectoryModal: React.FC<TeleportDirectoryModalProps> = ({
  player,
  onClose,
  onOpenManage,
}) => {
  const {
    teleportDirectory,
    fetchTeleportDirectory,
    travelTeleport,
    teleportLoading,
    actionLoading,
    mySectStatus,
  } = useGameStore();

  const [filter, setFilter] = useState<"ALL" | "SECT" | "MINE">("ALL");

  useEffect(() => {
    fetchTeleportDirectory();
  }, [fetchTeleportDirectory]);

  const source = teleportDirectory?.source_array;
  const destinations = teleportDirectory?.destinations || [];

  const mySectId = mySectStatus?.sect?.id;

  const publicDestinations = destinations.filter((dest) => dest.array.access_policy === "PUBLIC");
  const sectDestinations = destinations.filter(
    (dest) => dest.array.access_policy === "SECT" || (dest.array.sect_id && dest.array.sect_id === mySectId)
  );
  const myDestinations = destinations.filter((dest) => dest.array.owner_id === player.id);

  const filteredDestinations =
    filter === "MINE"
      ? myDestinations
      : filter === "SECT"
      ? sectDestinations
      : publicDestinations;

  // Usable links first, keeping the server order otherwise.
  const sortedDestinations = [...filteredDestinations].sort(
    (a, b) => Number(b.can_use) - Number(a.can_use)
  );

  const handleTravel = async (targetArrayId: string) => {
    const res = await travelTeleport(targetArrayId);
    if (res.success) {
      onClose();
    }
  };

  return (
    <div className="fixed inset-0 z-[9999] h-[100dvh] flex items-center justify-center px-2 pt-[calc(env(safe-area-inset-top)+0.5rem)] pb-[calc(env(safe-area-inset-bottom)+0.5rem)] sm:px-4">
      <div className="absolute inset-0 bg-black/75 backdrop-blur-md" onClick={onClose} />
      <div className="relative bg-zinc-950 border border-amber-500/30 rounded-3xl shadow-2xl w-full max-w-2xl max-h-[calc(100dvh-env(safe-area-inset-top)-env(safe-area-inset-bottom)-1rem)] flex flex-col overflow-hidden animate-in fade-in zoom-in-95 duration-200 text-left">
        {/* Header */}
        <div className="shrink-0 flex items-center justify-between border-b border-zinc-800/80 px-4 sm:px-6 py-3 sm:py-4 bg-gradient-to-r from-blue-950/20 via-zinc-900/40 to-transparent">
          <div className="flex items-center gap-2.5">
            <div className="p-2 rounded-xl bg-blue-500/10 border border-blue-500/30 text-blue-400">
              <Compass className="w-5 h-5 animate-spin" />
            </div>
            <div>
              <h3 className="text-base font-bold text-zinc-100 font-serif">Danh Bạ Truyền Tống Trận</h3>
              <p className="text-[11px] text-zinc-400 font-mono">
                {source ? (
                  <>
                    Điểm gốc: <span className="text-amber-400 font-bold">{source.name}</span> ({source.x}, {source.y}) — Vực: {source.region_id}
                  </>
                ) : (
                  "Đang xác định điểm kết nối không gian..."
                )}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {source && source.owner_id === player.id && onOpenManage && (
              <button
                type="button"
                onClick={() => {
                  onClose();
                  onOpenManage(source.id);
                }}
                className="p-2 rounded-xl bg-zinc-900 border border-zinc-700/80 text-zinc-300 hover:text-amber-400 hover:border-amber-500/50 transition-all text-xs font-mono flex items-center gap-1 cursor-pointer"
                title="Quản lý trận pháp hiện tại"
              >
                <Settings className="w-3.5 h-3.5" />
                <span className="hidden sm:inline">Quản Lý Trận</span>
              </button>
            )}
            <button
              onClick={() => fetchTeleportDirectory()}
              disabled={teleportLoading}
              className="p-2 rounded-xl text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900 transition-colors cursor-pointer"
              title="Làm mới"
            >
              <RefreshCw className={`w-4 h-4 ${teleportLoading ? "animate-spin" : ""}`} />
            </button>
            <button
              onClick={onClose}
              className="p-1.5 rounded-xl text-zinc-500 hover:text-zinc-200 hover:bg-zinc-900 transition-colors cursor-pointer"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Filters */}
        <div className="shrink-0 px-4 sm:px-6 pt-3 pb-2 flex items-center gap-2 border-b border-zinc-900/60 bg-zinc-950/40 overflow-x-auto">
          <button
            type="button"
            onClick={() => setFilter("ALL")}
            className={`px-3 py-1 rounded-xl text-xs font-mono transition-all cursor-pointer ${
              filter === "ALL"
                ? "bg-amber-500/20 border border-amber-500/50 text-amber-300 font-bold"
                : "text-zinc-500 hover:text-zinc-300 bg-zinc-900/50"
            }`}
          >
            Tất Cả ({publicDestinations.length})
          </button>
          <button
            type="button"
            onClick={() => setFilter("SECT")}
            className={`px-3 py-1 rounded-xl text-xs font-mono transition-all cursor-pointer ${
              filter === "SECT"
                ? "bg-amber-500/20 border border-amber-500/50 text-amber-300 font-bold"
                : "text-zinc-500 hover:text-zinc-300 bg-zinc-900/50"
            }`}
          >
            Tông Môn ({sectDestinations.length})
          </button>
          <button
            type="button"
            onClick={() => setFilter("MINE")}
            className={`px-3 py-1 rounded-xl text-xs font-mono transition-all cursor-pointer ${
              filter === "MINE"
                ? "bg-amber-500/20 border border-amber-500/50 text-amber-300 font-bold"
                : "text-zinc-500 hover:text-zinc-300 bg-zinc-900/50"
            }`}
          >
            Của Ta ({myDestinations.length})
          </button>
        </div>

        {/* Content List */}
        <div className="p-4 sm:p-6 space-y-3 flex-1 min-h-0 overflow-y-auto custom-scrollbar">
          {teleportLoading ? (
            <div className="py-12 text-center text-zinc-500 font-mono text-xs flex flex-col items-center gap-2">
              <Compass className="w-6 h-6 animate-spin text-amber-500" />
              Đang rò quét kết giới không gian...
            </div>
          ) : !source ? (
            <div className="py-12 text-center text-zinc-500 font-mono text-xs space-y-2">
              <AlertCircle className="w-8 h-8 mx-auto text-amber-500/60" />
              <p>Đạo hữu cần đứng trên một Truyền Tống Trận để kích hoạt danh bạ.</p>
            </div>
          ) : filteredDestinations.length === 0 ? (
            <div className="py-12 text-center text-zinc-500 font-mono text-xs space-y-2">
              <MapPin className="w-8 h-8 mx-auto text-zinc-600" />
              <p>Chưa có điểm dịch chuyển nào khác trong tầm phủ sóng.</p>
            </div>
          ) : (
            sortedDestinations.map(({ array: dest, distance, is_cross_region_jump, can_use, reason, fee_to_pay, fee_currency }) => {
              const isOwner = dest.owner_id === player.id;
              const hasFee = fee_to_pay > 0;

              return (
                <div
                  key={dest.id}
                  className={`p-4 rounded-2xl border transition-all flex flex-col sm:flex-row sm:items-center justify-between gap-3 ${
                    can_use
                      ? "bg-zinc-900/50 border-zinc-800 hover:border-amber-500/40 hover:bg-zinc-900/80"
                      : "bg-zinc-950/40 border-zinc-900 opacity-60"
                  }`}
                >
                  <div className="space-y-1.5 flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <h4 className="text-xs font-bold text-zinc-100 font-serif truncate">{dest.name}</h4>
                      {isOwner && (
                        <span className="text-[9px] font-mono px-1.5 py-0.5 rounded bg-amber-950/60 text-amber-400 border border-amber-800/40">
                          Sở hữu
                        </span>
                      )}
                      <span className="text-[9px] font-mono px-1.5 py-0.5 rounded bg-zinc-800 text-zinc-400">
                        {dest.access_policy === "PUBLIC" ? "Công cộng" : dest.access_policy === "SECT" ? "Tông Môn" : "Riêng tư"}
                      </span>
                      {is_cross_region_jump && (
                        <span className="text-[9px] font-mono px-1.5 py-0.5 rounded bg-purple-950/60 text-purple-300 border border-purple-800/40">
                          Xuyên Vực
                        </span>
                      )}
                    </div>

                    <div className="flex items-center gap-3 text-[10px] font-mono text-zinc-400 flex-wrap">
                      <span className="flex items-center gap-1 text-zinc-300">
                        <MapPin className="w-3 h-3 text-emerald-400" />
                        ({dest.x}, {dest.y}) · {dest.region_name || dest.region_id}
                      </span>
                      <span>
                        Cự ly: <strong className="text-zinc-200">{is_cross_region_jump ? "Xuyên Ngũ Vực" : `${distance.toLocaleString()} ô`}</strong>
                      </span>
                      <span>
                        Độ bền: <span className={dest.current_durability > 10 ? "text-emerald-400" : "text-rose-400"}>{dest.current_durability}/{dest.max_durability}</span>
                      </span>
                    </div>

                    {!can_use && reason && (
                      <p className="text-[10px] font-mono text-rose-400 flex items-center gap-1">
                        <AlertCircle className="w-3 h-3 shrink-0" />
                        {reason}
                      </p>
                    )}
                  </div>

                  {/* Right: Fee & Action */}
                  <div className="flex sm:flex-col items-center sm:items-end justify-between sm:justify-center gap-2 shrink-0 pt-2 sm:pt-0 border-t sm:border-t-0 border-zinc-800/60">
                    <div className="text-[11px] font-mono">
                      {hasFee ? (
                        <span className="text-amber-400 font-bold flex items-center gap-1">
                          <Coins className="w-3.5 h-3.5" />
                          {fee_to_pay.toLocaleString()} {fee_currency === "immortal_stones" ? "Tiên Nguyên" : "Nguyên Thạch"}
                        </span>
                      ) : (
                        <span className="text-emerald-400 font-bold">Miễn Phí</span>
                      )}
                    </div>

                    <GameButton
                      onClick={() => handleTravel(dest.id)}
                      disabled={!can_use || actionLoading}
                      variant={can_use ? "primary" : "secondary"}
                      className="py-1.5 px-4 text-xs font-bold font-serif flex items-center gap-1"
                    >
                      <ArrowRight className="w-3.5 h-3.5" />
                      Truyền Tống
                    </GameButton>
                  </div>
                </div>
              );
            })
          )}
        </div>
      </div>
    </div>
  );
};
