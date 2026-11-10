"use client";

import React, { useEffect, useState, useMemo } from "react";
import { createPortal } from "react-dom";
import { useGameStore, GuInstance, guSpecFor } from "@/store/useGameStore";
import { Lock, Unlock, Search, Sparkles, AlertCircle, Layers, LayoutGrid, List, X, Info } from "lucide-react";
import { translatePath } from "@/lib/translate";
import { getPathColor as _getPathColor } from "@/lib/utils";
import { getSealingStoneCost } from "@/lib/balance";
import { getGuIconUrl } from "@/lib/gu-icons";
import { getGuRankColor } from "@/lib/gu-frames";

const getPathColor = (path: string): string => {
  const c = _getPathColor(path);
  return `${c.bg} ${c.text} ${c.border}`;
};

export const UtilitiesView: React.FC = () => {
  const {
    player,
    actionLoading,
    updateGuStatus,
    addToast,
    showConfirm,
    setView,
  } = useGameStore();

  const [activeTab, setActiveTab] = useState<"seal" | "unseal" | "future">("seal");
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedRank, setSelectedRank] = useState<string>("all");
  const [viewMode, setViewMode] = useState<"grid" | "list">("grid");
  const [processingId, setProcessingId] = useState<string | null>(null);
  const [inspectedGu, setInspectedGu] = useState<GuInstance | null>(null);

  // Lazy-load the full inventory if it isn't loaded yet
  const fetchGuInventory = useGameStore((state) => state.fetchGuInventory);
  const guInventoryLoaded = useGameStore((state) => state.guInventoryLoaded);
  useEffect(() => {
    if (!guInventoryLoaded) void fetchGuInventory();
  }, [fetchGuInventory, guInventoryLoaded]);

  const sealingStoneQty = player?.materials_inventory?.["sealing_stone"] || 0;
  const guList = useMemo(() => player?.gu_inventory || [], [player?.gu_inventory]);

  // Filter Gu bugs depending on action tab
  const filteredGuList = useMemo(() => {
    return guList.filter((gu) => {
      const spec = guSpecFor(gu.gu_id, gu.rank);
      const name = spec?.name || gu.gu_id;
      const matchesSearch = name.toLowerCase().includes(searchTerm.toLowerCase());
      const matchesRank = selectedRank === "all" || gu.rank.toString() === selectedRank;

      if (activeTab === "seal") {
        // Seal tab: show active or sleeping Gu bugs that are NOT already sealed, NOT dead, NOT vital, and NOT equipped
        return (
          matchesSearch &&
          matchesRank &&
          gu.status !== "SEALED" &&
          gu.status !== "DEAD" &&
          !gu.is_vital &&
          !gu.equipped
        );
      } else if (activeTab === "unseal") {
        // Unseal tab: show only sealed Gu bugs
        return matchesSearch && matchesRank && gu.status === "SEALED";
      }
      return false;
    });
  }, [guList, searchTerm, selectedRank, activeTab]);

  if (!player) return null;

  const handleSealGu = async (gu: GuInstance, name: string) => {
    if (actionLoading || processingId) return;

    if (gu.equipped) {
      addToast("Không thể phong ấn Cổ trùng đang được trang bị. Hãy tháo ra trước!", "error");
      return;
    }

    const neededStones = getSealingStoneCost(gu.rank);
    if (sealingStoneQty < neededStones) {
      addToast(`Đạo hữu không có đủ Đá Phong Ấn (sealing_stone)! Cần ${neededStones.toLocaleString("vi-VN")}x Đá Phong Ấn.`, "error");
      return;
    }

    const confirmMsg = `Phong ấn Cổ trùng "${name}" (Rank ${gu.rank}) sẽ tiêu hao ${neededStones.toLocaleString("vi-VN")}x Đá Phong Ấn.\nĐạo hữu chắc chắn muốn phong ấn chứ?`;

    if (!(await showConfirm(confirmMsg))) return;

    setProcessingId(gu.instance_id);
    try {
      await updateGuStatus(gu.instance_id, "SEALED");
      if (inspectedGu?.instance_id === gu.instance_id) {
        setInspectedGu(null);
      }
    } finally {
      setProcessingId(null);
    }
  };

  const handleUnsealGu = async (gu: GuInstance, name: string) => {
    if (actionLoading || processingId) return;

    const confirmMsg = `Đạo hữu muốn giải phong ấn cho Cổ trùng "${name}" (Rank ${gu.rank})? Trạng thái của Cổ trùng sẽ chuyển về KÍCH HOẠT (ACTIVE).`;
    if (!(await showConfirm(confirmMsg))) return;

    setProcessingId(gu.instance_id);
    try {
      await updateGuStatus(gu.instance_id, "ACTIVE");
      if (inspectedGu?.instance_id === gu.instance_id) {
        setInspectedGu(null);
      }
    } finally {
      setProcessingId(null);
    }
  };

  const inspectedSpec = inspectedGu ? guSpecFor(inspectedGu.gu_id, inspectedGu.rank) : null;
  const inspectedName = inspectedSpec?.name || inspectedGu?.gu_id || "";
  const inspectedRankColor = inspectedGu ? getGuRankColor(inspectedGu.rank) : null;
  const inspectedNeededStones = inspectedGu ? getSealingStoneCost(inspectedGu.rank) : 0;

  return (
    <div className="glass-panel-gold rounded-3xl p-4 sm:p-6 w-full mx-auto space-y-6 text-left relative overflow-hidden group">
      {/* Decorative background grid pattern */}
      <div className="absolute inset-0 bg-[linear-gradient(to_right,rgba(217,119,6,0.01)_1px,transparent_1px),linear-gradient(to_bottom,rgba(217,119,6,0.01)_1px,transparent_1px)] bg-[size:16px_28px] pointer-events-none" />

      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between border-b border-zinc-800 pb-4 gap-4 relative z-10">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-amber-950/40 border border-amber-500/20 flex items-center justify-center">
            <Layers className="w-5 h-5 text-amber-400 animate-pulse" />
          </div>
          <div>
            <h3 className="text-lg font-bold text-zinc-100 font-serif">Tiện Ích Cổ Giới</h3>
            <p className="text-xs text-zinc-500">Các phương thức đặc thù bổ trợ nuôi dưỡng và điều khiển Cổ trùng.</p>
          </div>
        </div>

        {/* Back and Stones Info */}
        <div className="flex items-center gap-3 self-end sm:self-auto">
          <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-zinc-900/60 border border-zinc-800/80 text-xs font-mono">
            <span className="text-zinc-400">Đá Phong Ấn:</span>
            <span className="font-bold text-amber-400">{sealingStoneQty.toLocaleString()}</span>
          </div>
          <button
            type="button"
            onClick={() => setView("dashboard")}
            className="px-3 py-1.5 rounded-xl bg-zinc-900 border border-zinc-800 hover:bg-zinc-800 text-xs text-zinc-300 transition-all cursor-pointer font-serif"
          >
            Đóng
          </button>
        </div>
      </div>

      {/* Navigation tabs */}
      <div className="flex border-b border-zinc-800/60 pb-px gap-2 overflow-x-auto relative z-10">
        <button
          type="button"
          onClick={() => {
            setActiveTab("seal");
            setSearchTerm("");
          }}
          className={`px-4 py-2 border-b-2 font-serif text-sm font-semibold transition-all shrink-0 flex items-center gap-2 cursor-pointer ${
            activeTab === "seal"
              ? "border-amber-500 text-amber-500"
              : "border-transparent text-zinc-400 hover:text-zinc-200"
          }`}
        >
          <Lock className="w-4 h-4" />
          Phong Ấn Cổ Trùng
        </button>

        <button
          type="button"
          onClick={() => {
            setActiveTab("unseal");
            setSearchTerm("");
          }}
          className={`px-4 py-2 border-b-2 font-serif text-sm font-semibold transition-all shrink-0 flex items-center gap-2 cursor-pointer ${
            activeTab === "unseal"
              ? "border-amber-500 text-amber-500"
              : "border-transparent text-zinc-400 hover:text-zinc-200"
          }`}
        >
          <Unlock className="w-4 h-4" />
          Giải Phong Ấn
        </button>

        <button
          type="button"
          onClick={() => setActiveTab("future")}
          className={`px-4 py-2 border-b-2 font-serif text-sm font-semibold transition-all shrink-0 flex items-center gap-2 cursor-pointer ${
            activeTab === "future"
              ? "border-amber-500 text-amber-500"
              : "border-transparent text-zinc-500 hover:text-zinc-400"
          }`}
        >
          <Sparkles className="w-4 h-4" />
          Tiện Ích Khác...
        </button>
      </div>

      {activeTab !== "future" ? (
        <div className="space-y-4 relative z-10">
          {/* Search, Filter & View Mode */}
          <div className="flex flex-col sm:flex-row gap-3 items-stretch sm:items-center">
            <div className="relative flex-grow">
              <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-500" />
              <input
                type="text"
                placeholder="Tìm kiếm Cổ trùng..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full pl-10 pr-4 py-2 rounded-xl bg-zinc-900/50 border border-zinc-800 text-zinc-100 text-sm focus:outline-none focus:border-amber-500/50 font-mono"
              />
            </div>

            <div className="flex items-center gap-2">
              <select
                value={selectedRank}
                onChange={(e) => setSelectedRank(e.target.value)}
                className="px-3 py-2 rounded-xl bg-zinc-900/50 border border-zinc-800 text-zinc-300 text-sm focus:outline-none focus:border-amber-500/50 font-mono cursor-pointer"
              >
                <option value="all">Tất cả Rank</option>
                {[1, 2, 3, 4, 5, 6, 7, 8, 9].map((r) => (
                  <option key={r} value={r}>
                    Rank {r}
                  </option>
                ))}
              </select>

              <div className="flex items-center bg-zinc-900 border border-zinc-800 rounded-xl p-0.5">
                <button
                  type="button"
                  onClick={() => setViewMode("grid")}
                  className={`p-1.5 rounded-lg transition-colors cursor-pointer ${
                    viewMode === "grid" ? "bg-amber-500/20 text-amber-400 font-bold" : "text-zinc-500 hover:text-zinc-300"
                  }`}
                  title="Dạng Lưới RPG"
                >
                  <LayoutGrid className="w-4 h-4" />
                </button>
                <button
                  type="button"
                  onClick={() => setViewMode("list")}
                  className={`p-1.5 rounded-lg transition-colors cursor-pointer ${
                    viewMode === "list" ? "bg-amber-500/20 text-amber-400 font-bold" : "text-zinc-500 hover:text-zinc-300"
                  }`}
                  title="Dạng Danh Sách"
                >
                  <List className="w-4 h-4" />
                </button>
              </div>
            </div>
          </div>

          {/* Description banner */}
          <div className="p-3 rounded-xl bg-zinc-900/40 border border-zinc-800/60 flex items-start gap-2.5 text-xs text-zinc-400 font-mono">
            <AlertCircle className="w-4 h-4 text-amber-500/80 shrink-0 mt-0.5" />
            <div>
              {activeTab === "seal" ? (
                <p>
                  <strong className="text-zinc-200">Phong Ấn Cổ:</strong> Phong ấn Cổ trùng vào trạng thái ngủ sâu,{" "}
                  <span className="text-zinc-300">
                    giúp <span className="font-semibold text-yellow-400">giảm tốc độ đói đi nhiều lần</span>.
                  </span>{" "}
                  Tiêu hao <span className="text-amber-400 font-semibold">Đá Phong Ấn theo Rank</span> khi phong ấn.
                </p>
              ) : (
                <p>
                  <strong className="text-zinc-200">Giải Phong Ấn:</strong> Giải phóng Cổ trùng ra khỏi trạng thái phong ấn để tiếp tục nuôi dưỡng và xuất chiến. Hành động này không tốn kém tài nguyên.
                </p>
              )}
            </div>
          </div>

          {/* Gu List Grid or List */}
          {filteredGuList.length > 0 ? (
            viewMode === "grid" ? (
              <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3">
                {filteredGuList.map((gu) => {
                  const spec = guSpecFor(gu.gu_id, gu.rank);
                  const name = spec?.name || gu.gu_id;
                  const rankColors = getGuRankColor(gu.rank);
                  const isProcessing = processingId === gu.instance_id;
                  const neededStones = getSealingStoneCost(gu.rank);
                  const iconUrl = getGuIconUrl(gu.gu_id, name);
                  const canAfford = activeTab === "seal" ? sealingStoneQty >= neededStones : true;

                  return (
                    <div
                      key={gu.instance_id}
                      onClick={() => setInspectedGu(gu)}
                      className={`group relative rounded-2xl border p-3 flex flex-col justify-between transition-all duration-200 cursor-pointer hover:-translate-y-1 ${rankColors.border} ${rankColors.bgGradient} hover:shadow-lg ${rankColors.glow}`}
                    >
                      {/* Art & Badges */}
                      <div className="relative aspect-square w-full rounded-xl bg-black/40 border border-zinc-800/60 overflow-hidden flex items-center justify-center p-2 mb-2">
                        <span className={`absolute top-1 left-1 px-1.5 py-0.5 rounded text-[9px] font-mono font-bold uppercase border ${rankColors.badge}`}>
                          R{gu.rank}
                        </span>
                        {spec?.path && (
                          <span className="absolute top-1 right-1 px-1.5 py-0.5 rounded text-[8px] font-mono text-zinc-300 bg-zinc-950/80 border border-zinc-800">
                            {translatePath(Array.isArray(spec.path) ? spec.path[0] : spec.path)}
                          </span>
                        )}
                        <img
                          src={iconUrl}
                          alt={name}
                          loading="lazy"
                          decoding="async"
                          className="w-12 h-12 sm:w-14 sm:h-14 object-contain transition-transform group-hover:scale-110 duration-200 drop-shadow-md"
                        />
                        <span className={`absolute bottom-1 right-1 px-1.5 py-0.2 rounded text-[8px] font-mono font-bold ${
                          gu.status === "SEALED" ? "bg-amber-950/80 text-amber-300 border border-amber-500/30" : "bg-emerald-950/80 text-emerald-300 border border-emerald-500/30"
                        }`}>
                          {gu.status}
                        </span>
                      </div>

                      {/* Info & Action */}
                      <div className="space-y-1.5">
                        <h4 className={`text-xs font-serif font-bold truncate group-hover:text-amber-300 transition-colors ${rankColors.text}`}>
                          {name}
                        </h4>

                        <div className="flex items-center justify-between text-[10px] font-mono text-zinc-400">
                          <span>Đói: {gu.hunger}%</span>
                          {activeTab === "seal" && (
                            <span className={canAfford ? "text-amber-400 font-bold" : "text-rose-400 font-bold"}>
                              {neededStones} đá
                            </span>
                          )}
                        </div>

                        <div className="pt-1 border-t border-zinc-800/40">
                          {activeTab === "seal" ? (
                            <button
                              type="button"
                              disabled={actionLoading || isProcessing || !canAfford}
                              onClick={(e) => {
                                e.stopPropagation();
                                handleSealGu(gu, name);
                              }}
                              className={`w-full py-1 rounded-lg text-[10px] font-mono font-bold flex items-center justify-center gap-1 transition-all ${
                                canAfford
                                  ? "bg-amber-600 hover:bg-amber-500 text-zinc-950 shadow-sm"
                                  : "bg-zinc-900 border border-zinc-800 text-zinc-500 opacity-60"
                              }`}
                            >
                              <Lock className="w-3 h-3" />
                              {isProcessing ? "..." : "Phong Ấn"}
                            </button>
                          ) : (
                            <button
                              type="button"
                              disabled={actionLoading || isProcessing}
                              onClick={(e) => {
                                e.stopPropagation();
                                handleUnsealGu(gu, name);
                              }}
                              className="w-full py-1 rounded-lg text-[10px] font-mono font-bold bg-emerald-600 hover:bg-emerald-500 text-zinc-950 flex items-center justify-center gap-1 transition-all shadow-sm"
                            >
                              <Unlock className="w-3 h-3" />
                              {isProcessing ? "..." : "Giải Phong"}
                            </button>
                          )}
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : (
              <div className="space-y-2">
                {filteredGuList.map((gu) => {
                  const spec = guSpecFor(gu.gu_id, gu.rank);
                  const name = spec?.name || gu.gu_id;
                  const rankColors = getGuRankColor(gu.rank);
                  const isProcessing = processingId === gu.instance_id;
                  const neededStones = getSealingStoneCost(gu.rank);
                  const iconUrl = getGuIconUrl(gu.gu_id, name);
                  const canAfford = activeTab === "seal" ? sealingStoneQty >= neededStones : true;

                  return (
                    <div
                      key={gu.instance_id}
                      onClick={() => setInspectedGu(gu)}
                      className={`p-3 rounded-2xl bg-zinc-950/60 border hover:border-zinc-700 transition-all flex justify-between items-center gap-4 cursor-pointer ${rankColors.border}`}
                    >
                      <div className="flex items-center gap-3 min-w-0">
                        <div className="w-12 h-12 rounded-xl bg-black/40 border border-zinc-800 flex items-center justify-center p-1.5 shrink-0">
                          <img
                            src={iconUrl}
                            alt={name}
                            loading="lazy"
                            decoding="async"
                            className="w-9 h-9 object-contain"
                          />
                        </div>
                        <div className="space-y-1 text-left min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="font-serif font-bold text-zinc-200 text-sm truncate">
                              {name}
                            </span>
                            <span className={`px-2 py-0.5 rounded text-[10px] font-bold uppercase border ${rankColors.badge}`}>
                              Rank {gu.rank}
                            </span>
                            {spec?.path && (
                              <div className="flex flex-wrap gap-1">
                                {Array.isArray(spec.path) ? (
                                  spec.path.map((p) => (
                                    <span key={p} className={`px-2 py-0.5 rounded text-[10px] font-semibold border ${getPathColor(p)}`}>
                                      {translatePath(p)}
                                    </span>
                                  ))
                                ) : (
                                  <span className={`px-2 py-0.5 rounded text-[10px] font-semibold border ${getPathColor(spec.path as string)}`}>
                                    {translatePath(spec.path as string)}
                                  </span>
                                )}
                              </div>
                            )}
                          </div>

                          <div className="flex items-center gap-4 text-xs text-zinc-500 font-mono">
                            <div>Trạng thái: <span className={gu.status === "ACTIVE" ? "text-emerald-400" : "text-amber-500"}>{gu.status}</span></div>
                            <div>Độ đói: <span className={gu.hunger > 80 ? "text-red-400" : "text-zinc-400"}>{gu.hunger}%</span></div>
                          </div>
                        </div>
                      </div>

                      <div className="shrink-0">
                        {activeTab === "seal" ? (
                          <button
                            type="button"
                            disabled={actionLoading || isProcessing || !canAfford}
                            onClick={(e) => {
                              e.stopPropagation();
                              handleSealGu(gu, name);
                            }}
                            className={`px-3.5 py-1.5 rounded-xl font-semibold text-xs transition-all flex items-center gap-1.5 cursor-pointer ${
                              canAfford
                                ? "bg-amber-600 hover:bg-amber-500 text-zinc-950"
                                : "bg-zinc-900 border border-zinc-800 text-zinc-500 opacity-60"
                            }`}
                          >
                            <Lock className="w-3.5 h-3.5" />
                            {isProcessing ? "Đang xử lý..." : `Phong Ấn (${neededStones.toLocaleString("vi-VN")} đá)`}
                          </button>
                        ) : (
                          <button
                            type="button"
                            disabled={actionLoading || isProcessing}
                            onClick={(e) => {
                              e.stopPropagation();
                              handleUnsealGu(gu, name);
                            }}
                            className="px-3.5 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-zinc-950 font-semibold text-xs transition-all cursor-pointer flex items-center gap-1.5"
                          >
                            <Unlock className="w-3.5 h-3.5" />
                            {isProcessing ? "Đang xử lý..." : "Giải Phong"}
                          </button>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            )
          ) : (
            <div className="py-12 text-center text-zinc-500 text-sm font-mono border border-dashed border-zinc-800/80 rounded-2xl">
              Không tìm thấy Cổ trùng nào phù hợp.
            </div>
          )}
        </div>
      ) : (
        /* Future extensions placeholder */
        <div className="py-16 text-center space-y-4 max-w-md mx-auto relative z-10">
          <div className="w-12 h-12 rounded-full bg-zinc-900/80 border border-zinc-800 flex items-center justify-center mx-auto">
            <Sparkles className="w-6 h-6 text-zinc-600" />
          </div>
          <div className="space-y-1">
            <h4 className="text-zinc-300 font-serif font-bold">Tính Năng Đang Phát Triển</h4>
            <p className="text-xs text-zinc-500 leading-relaxed font-mono">
              Các tiện ích đặc thù khác (nâng cấp túi, tẩy luyện hồn ấn) sẽ được cập nhật trong tương lai.
            </p>
          </div>
        </div>
      )}

      {/* Gu Inspector Modal */}
      {inspectedGu && inspectedRankColor && (
        createPortal(
          <div
            className="fixed inset-0 bg-black/80 flex items-center justify-center p-4 z-50 animate-fade-in"
            onClick={() => setInspectedGu(null)}
          >
            <div
              className={`bg-zinc-950 border ${inspectedRankColor.border} rounded-3xl p-6 max-w-sm w-full space-y-4 shadow-2xl relative ${inspectedRankColor.glow}`}
              onClick={(e) => e.stopPropagation()}
            >
              <button
                type="button"
                onClick={() => setInspectedGu(null)}
                className="absolute top-4 right-4 p-1.5 text-zinc-500 hover:text-zinc-200 rounded-full hover:bg-zinc-900 transition-colors cursor-pointer"
              >
                <X className="w-4 h-4" />
              </button>

              {/* Gu Visual Header */}
              <div className="flex items-center gap-3 pt-1">
                <div className={`w-16 h-16 rounded-2xl bg-black/60 border ${inspectedRankColor.border} flex items-center justify-center p-2 shrink-0 relative shadow-inner`}>
                  <img
                    src={getGuIconUrl(inspectedGu.gu_id, inspectedName)}
                    alt={inspectedName}
                    className="w-12 h-12 object-contain"
                  />
                  <span className={`absolute bottom-1 right-1 text-[8px] font-mono font-bold px-1 rounded ${
                    inspectedGu.status === "SEALED" ? "bg-amber-950 text-amber-300" : "bg-emerald-950 text-emerald-300"
                  }`}>
                    {inspectedGu.status}
                  </span>
                </div>
                <div className="min-w-0">
                  <h4 className="text-base font-bold text-zinc-100 font-serif truncate">
                    {inspectedName}
                  </h4>
                  <div className="flex items-center gap-1.5 mt-1">
                    <span className={`px-2 py-0.5 rounded text-[10px] font-mono font-bold uppercase border ${inspectedRankColor.badge}`}>
                      Rank {inspectedGu.rank}
                    </span>
                    {inspectedSpec?.path && (
                      <span className="px-2 py-0.5 rounded text-[10px] font-mono text-zinc-400 bg-zinc-900 border border-zinc-800">
                        {translatePath(Array.isArray(inspectedSpec.path) ? inspectedSpec.path[0] : inspectedSpec.path)}
                      </span>
                    )}
                  </div>
                </div>
              </div>

              {/* Lore / Description */}
              <div className="bg-zinc-900/40 border border-zinc-800/80 rounded-xl p-3 text-xs text-zinc-400 font-mono leading-relaxed max-h-24 overflow-y-auto">
                <div className="flex items-start gap-1.5 mb-1 text-zinc-500 text-[10px] uppercase font-bold tracking-wider">
                  <Info className="w-3 h-3 text-amber-500/70 shrink-0 mt-0.5" />
                  Bí Thuật & Mô Tả
                </div>
                {inspectedSpec?.description || "Cổ trùng kỳ dị trong thiên địa."}
              </div>

              {/* Status details */}
              <div className="bg-zinc-900/60 border border-zinc-800 rounded-xl p-3 space-y-1.5 text-xs font-mono">
                <div className="flex justify-between">
                  <span className="text-zinc-400">Độ no:</span>
                  <span className={`font-bold ${inspectedGu.hunger > 80 ? "text-rose-400" : "text-emerald-400"}`}>
                    {100 - inspectedGu.hunger}% (Đói: {inspectedGu.hunger}%)
                  </span>
                </div>
                {activeTab === "seal" && (
                  <div className="flex justify-between border-t border-zinc-800/60 pt-1.5">
                    <span className="text-zinc-400">Chi phí phong ấn:</span>
                    <span className="font-bold text-amber-400">
                      {inspectedNeededStones} Đá Phong Ấn
                    </span>
                  </div>
                )}
                {activeTab === "seal" && (
                  <div className="flex justify-between">
                    <span className="text-zinc-400">Đá hiện có:</span>
                    <span className={`font-bold ${sealingStoneQty >= inspectedNeededStones ? "text-emerald-400" : "text-rose-400"}`}>
                      {sealingStoneQty} viên
                    </span>
                  </div>
                )}
              </div>

              {/* Action buttons */}
              <div className="grid grid-cols-2 gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setInspectedGu(null)}
                  className="py-2.5 rounded-xl border border-zinc-800 text-zinc-400 hover:bg-zinc-900 font-semibold text-xs active:scale-[0.98] transition-all cursor-pointer"
                >
                  Đóng
                </button>
                {activeTab === "seal" ? (
                  <button
                    type="button"
                    disabled={actionLoading || processingId === inspectedGu.instance_id || sealingStoneQty < inspectedNeededStones}
                    onClick={() => handleSealGu(inspectedGu, inspectedName)}
                    className="py-2.5 rounded-xl bg-amber-600 hover:bg-amber-500 disabled:opacity-50 text-zinc-950 font-bold text-xs active:scale-[0.98] transition-all cursor-pointer shadow-lg flex items-center justify-center gap-1.5"
                  >
                    <Lock className="w-3.5 h-3.5" />
                    Phong Ấn
                  </button>
                ) : (
                  <button
                    type="button"
                    disabled={actionLoading || processingId === inspectedGu.instance_id}
                    onClick={() => handleUnsealGu(inspectedGu, inspectedName)}
                    className="py-2.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-zinc-950 font-bold text-xs active:scale-[0.98] transition-all cursor-pointer shadow-lg flex items-center justify-center gap-1.5"
                  >
                    <Unlock className="w-3.5 h-3.5" />
                    Giải Phong
                  </button>
                )}
              </div>
            </div>
          </div>,
          document.body
        )
      )}
    </div>
  );
};
