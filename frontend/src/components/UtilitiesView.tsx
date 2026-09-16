"use client";

import React, { useEffect, useState, useMemo } from "react";
import { useGameStore, GuInstance, guSpecFor } from "@/store/useGameStore";
import { Lock, Unlock, Search, Sparkles, AlertCircle, Layers } from "lucide-react";
import { translatePath } from "@/lib/translate";
import { getPathColor as _getPathColor } from "@/lib/utils";

const getPathColor = (path: string): string => {
  const c = _getPathColor(path);
  return `${c.bg} ${c.text} ${c.border}`;
};

const getRankPFactor = (rank: number): number => {
  switch (rank) {
    case 1: return 1;
    case 2: return 2;
    case 3: return 5;
    case 4: return 20;
    case 5: return 100;
    case 6: return 10000;
    case 7: return 1000000;
    case 8: return 100000000;
    case 9: return 10000000000;
    default: return rank < 1 ? 1 : 10000000000;
  }
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
  const [processingId, setProcessingId] = useState<string | null>(null);

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

    const neededStones = getRankPFactor(gu.rank);
    if (sealingStoneQty < neededStones) {
      addToast(`Đạo hữu không có đủ Đá Phong Ấn (sealing_stone)! Cần ${neededStones.toLocaleString("vi-VN")}x Đá Phong Ấn.`, "error");
      return;
    }

    const confirmMsg = `Phong ấn Cổ trùng "${name}" (Rank ${gu.rank}) sẽ tiêu hao ${neededStones.toLocaleString("vi-VN")}x Đá Phong Ấn.\nĐạo hữu chắc chắn muốn phong ấn chứ?`;

    if (!(await showConfirm(confirmMsg))) return;

    setProcessingId(gu.instance_id);
    try {
      await updateGuStatus(gu.instance_id, "SEALED");
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
    } finally {
      setProcessingId(null);
    }
  };

  return (
    <div className="glass-panel-gold rounded-3xl p-6 w-full mx-auto space-y-6 text-left relative overflow-hidden group">
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
            <span className="font-bold text-amber-400">{sealingStoneQty}</span>
          </div>
          <button
            onClick={() => setView("dashboard")}
            className="px-3 py-1.5 rounded-xl bg-zinc-900 border border-zinc-800 hover:bg-zinc-800 text-xs text-zinc-300 transition-all cursor-pointer font-serif"
          >
            Đóng
          </button>
        </div>
      </div>

      {/* Navigation tabs */}
      <div className="flex border-b border-zinc-800/60 pb-px gap-2 overflow-x-auto">
        <button
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
          Phong Ấn Cổ
        </button>

        <button
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
        <div className="space-y-4">
          {/* Search & Filters */}
          <div className="flex flex-col sm:flex-row gap-3">
            <div className="relative flex-grow">
              <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-500" />
              <input
                type="text"
                placeholder="Tìm kiếm Cổ trùng..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full pl-10 pr-4 py-2 rounded-xl bg-zinc-900/50 border border-zinc-800 text-zinc-100 text-sm focus:outline-none focus:border-amber-500/50"
              />
            </div>

            <select
              value={selectedRank}
              onChange={(e) => setSelectedRank(e.target.value)}
              className="px-3 py-2 rounded-xl bg-zinc-900/50 border border-zinc-800 text-zinc-300 text-sm focus:outline-none focus:border-amber-500/50"
            >
              <option value="all">Tất cả Rank</option>
              {[1, 2, 3, 4, 5, 6, 7, 8, 9].map((r) => (
                <option key={r} value={r}>
                  Rank {r}
                </option>
              ))}
            </select>
          </div>

          {/* Description banner */}
          <div className="p-3 rounded-xl bg-zinc-900/40 border border-zinc-800/60 flex items-start gap-2.5 text-xs text-zinc-400">
            <AlertCircle className="w-4 h-4 text-amber-500/80 shrink-0 mt-0.5" />
            <div>
              {activeTab === "seal" ? (
                <p>
                  <strong>Phong Ấn Cổ:</strong> Phong ấn Cổ trùng của đạo hữu vào một trạng thái ngủ sâu, <span className="text-gray-300">
                    nhưng chúng
                    <span className="font-bold text-red-500"> VẪN BỊ ĐÓI và CHẾT</span>,
                    chỉ là
                    <span className="font-semibold text-yellow-400"> giảm độ no chậm hơn</span>.
                  </span> Cần tiêu hao <span className="text-amber-400 font-semibold">Lượng Đá Phong Ấn theo Hệ số Rank P</span> cho mỗi lần thực hiện.
                </p>
              ) : (
                <p>
                  <strong>Giải Phong Ấn:</strong> Giải phóng Cổ trùng ra khỏi trạng thái phong ấn để có thể tiếp tục nuôi dưỡng và trang bị vào Không Khiếu. Hành động này không tốn kém tài nguyên.
                </p>
              )}
            </div>
          </div>

          {/* Gu List Grid */}
          {filteredGuList.length > 0 ? (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {filteredGuList.map((gu) => {
                const spec = guSpecFor(gu.gu_id, gu.rank);
                const name = spec?.name || gu.gu_id;
                const isProcessing = processingId === gu.instance_id;
                const neededStones = getRankPFactor(gu.rank);

                return (
                  <div
                    key={gu.instance_id}
                    className="p-4 rounded-2xl bg-zinc-900/40 border border-zinc-800/80 hover:border-zinc-700/60 transition-all flex justify-between items-center gap-4"
                  >
                    <div className="space-y-1.5 text-left">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="font-serif font-bold text-zinc-200 text-sm">
                          {name}
                        </span>
                        <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-amber-500/10 text-amber-400 border border-amber-500/20">
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

                    <div>
                      {activeTab === "seal" ? (
                        <button
                          disabled={actionLoading || isProcessing}
                          onClick={() => handleSealGu(gu, name)}
                          className="px-3.5 py-1.5 rounded-xl bg-amber-600 hover:bg-amber-500 text-zinc-950 font-semibold text-xs transition-all disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer flex items-center gap-1.5"
                        >
                          <Lock className="w-3.5 h-3.5" />
                          {isProcessing ? "Đang xử lý..." : `Phong Ấn (${neededStones.toLocaleString("vi-VN")} đá)`}
                        </button>
                      ) : (
                        <button
                          disabled={actionLoading || isProcessing}
                          onClick={() => handleUnsealGu(gu, name)}
                          className="px-3.5 py-1.5 rounded-xl bg-emerald-600 hover:bg-emerald-500 text-zinc-950 font-semibold text-xs transition-all disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer flex items-center gap-1.5"
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
          ) : (
            <div className="py-12 text-center text-zinc-500 text-sm font-mono border border-dashed border-zinc-800/80 rounded-2xl">
              Không tìm thấy Cổ trùng nào phù hợp.
            </div>
          )}
        </div>
      ) : (
        /* Future extensions placeholder */
        <div className="py-16 text-center space-y-4 max-w-md mx-auto">
          <div className="w-12 h-12 rounded-full bg-zinc-900/80 border border-zinc-800 flex items-center justify-center mx-auto">
            <Sparkles className="w-6 h-6 text-zinc-600" />
          </div>
          <div className="space-y-1">
            <h4 className="text-zinc-300 font-serif font-bold">Tính Năng Đang Phát Triển</h4>
            <p className="text-xs text-zinc-500 leading-relaxed font-mono">
              Các tiện ích đặc thù khác ... sẽ được cập nhật trong tương lai.
            </p>
          </div>
        </div>
      )}
    </div>
  );
};
