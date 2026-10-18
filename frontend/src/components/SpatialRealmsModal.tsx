"use client";

import React, { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import {
  Compass,
  X,
  Users,
  DoorOpen,
  Shield,
  Swords,
  MapPin,
  Clock,
  Sparkles,
  ArrowRight,
  RefreshCw,
} from "lucide-react";

interface SpatialRealmsModalProps {
  isOpen?: boolean;
  onClose?: () => void;
}

export const SpatialRealmsModal: React.FC<SpatialRealmsModalProps> = ({ isOpen, onClose }) => {
  const spatialDirectoryOpen = useGameStore((state) => state.spatialDirectoryOpen);
  const setSpatialDirectoryOpen = useGameStore((state) => state.setSpatialDirectoryOpen);
  const spatialRealms = useGameStore((state) => state.spatialRealms);
  const fetchSpatialRealms = useGameStore((state) => state.fetchSpatialRealms);
  const enterSpatialRealm = useGameStore((state) => state.enterSpatialRealm);
  const spatialLoading = useGameStore((state) => state.spatialLoading);
  const player = useGameStore((state) => state.player);
  const addToast = useGameStore((state) => state.addToast);

  const isModalOpen = isOpen !== undefined ? isOpen : spatialDirectoryOpen;
  const handleClose = () => {
    if (onClose) onClose();
    setSpatialDirectoryOpen(false);
  };

  const [filterScope, setFilterScope] = useState<string>("ALL");
  const [search, setSearch] = useState<string>("");

  useEffect(() => {
    if (isModalOpen) {
      void fetchSpatialRealms();
    }
  }, [isModalOpen, fetchSpatialRealms]);

  if (!isModalOpen) {
    return null;
  }

  const filteredRealms = spatialRealms.filter((realm) => {
    if (filterScope !== "ALL" && realm.scope !== filterScope) {
      return false;
    }
    if (search.trim()) {
      const q = search.toLowerCase();
      return (
        realm.name.toLowerCase().includes(q) ||
        realm.description.toLowerCase().includes(q)
      );
    }
    return true;
  });

  const handleEnter = async (realm: (typeof spatialRealms)[0]) => {
    // Check if player is near entry portal
    if (
      player &&
      (player.current_region !== realm.entry_region_id ||
        Math.abs(player.x - realm.entry_x) > 1 ||
        Math.abs(player.y - realm.entry_y) > 1)
    ) {
      addToast(
        `Đạo hữu cần di chuyển đến Cổng Vào tại khu vực [${realm.entry_region_id}] tọa độ (${realm.entry_x}, ${realm.entry_y}) để tiến vào!`,
        "info"
      );
      return;
    }

    const success = await enterSpatialRealm(realm.template_id);
    if (success) {
      handleClose();
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm animate-fade-in">
      <div className="relative w-full max-w-3xl max-h-[90vh] flex flex-col bg-gradient-to-b from-zinc-950 via-zinc-900 to-zinc-950 border border-amber-500/50 rounded-3xl p-6 shadow-[0_0_50px_rgba(245,158,11,0.25)] text-left">
        {/* Close Button */}
        <button
          onClick={handleClose}
          className="absolute top-4 right-4 p-2 text-zinc-400 hover:text-zinc-100 hover:bg-zinc-800/60 rounded-full transition-all"
        >
          <X className="w-5 h-5" />
        </button>

        {/* Header */}
        <div className="flex items-center justify-between border-b border-amber-500/20 pb-4 mb-4">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 rounded-2xl bg-amber-500/10 border border-amber-500/40 flex items-center justify-center text-amber-400 shadow-[0_0_15px_rgba(245,158,11,0.3)]">
              <Compass className="w-6 h-6 animate-spin-slow" />
            </div>
            <div>
              <h2 className="text-xl font-bold font-serif text-amber-100 tracking-wide">
                Bí Cảnh Không Gian & Truyền Thừa
              </h2>
              <p className="text-xs text-amber-400/80">
                Danh mục tiểu giới độc lập & vùng khảo nghiệm di sản tiền nhân trên khắp Ngũ Vực
              </p>
            </div>
          </div>

          <button
            onClick={() => fetchSpatialRealms()}
            disabled={spatialLoading}
            className="p-2 text-amber-400 hover:text-amber-300 hover:bg-amber-500/10 rounded-xl transition-all disabled:opacity-50"
            title="Làm mới danh sách"
          >
            <RefreshCw className={`w-4 h-4 ${spatialLoading ? "animate-spin" : ""}`} />
          </button>
        </div>

        {/* Filters */}
        <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
          <div className="flex items-center gap-1.5 bg-zinc-900/60 p-1 rounded-xl border border-zinc-800 text-xs">
            {["ALL", "SHARED", "PARTY", "SOLO"].map((sc) => (
              <button
                key={sc}
                onClick={() => setFilterScope(sc)}
                className={`px-3 py-1 rounded-lg font-semibold transition-all ${
                  filterScope === sc
                    ? "bg-amber-500 text-zinc-950 shadow-md"
                    : "text-zinc-400 hover:text-zinc-200"
                }`}
              >
                {sc === "ALL"
                  ? "Tất cả"
                  : sc === "SHARED"
                  ? "Bí Cảnh Chung"
                  : sc === "PARTY"
                  ? "Tổ Đội"
                  : "Cá Nhân"}
              </button>
            ))}
          </div>

          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Tìm kiếm bí cảnh..."
            className="px-3 py-1.5 text-xs bg-zinc-900/60 border border-zinc-800 rounded-xl text-zinc-200 placeholder-zinc-500 focus:outline-none focus:border-amber-500/50 w-full sm:w-56"
          />
        </div>

        {/* List of Realms */}
        <div className="flex-1 overflow-y-auto space-y-3 pr-1">
          {filteredRealms.length === 0 ? (
            <div className="text-center py-12 text-zinc-500 text-sm">
              Không tìm thấy bí cảnh hoặc truyền thừa nào phù hợp.
            </div>
          ) : (
            filteredRealms.map((realm) => {
              const isAtPortal =
                player &&
                player.current_region === realm.entry_region_id &&
                Math.abs(player.x - realm.entry_x) <= 1 &&
                Math.abs(player.y - realm.entry_y) <= 1;

              const isSafe = (realm.rules?.pvp_mode || "SAFE").toUpperCase() === "SAFE";

              return (
                <div
                  key={realm.template_id}
                  className="p-4 rounded-2xl bg-zinc-900/50 border border-zinc-800 hover:border-amber-500/40 transition-all flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4"
                >
                  <div className="space-y-1.5 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-bold text-amber-200 text-base">
                        {realm.name}
                      </span>
                      <span
                        className={`px-2 py-0.5 text-[10px] font-bold rounded-full border ${
                          realm.scope === "SHARED"
                            ? "bg-purple-950/40 text-purple-300 border-purple-500/40"
                            : realm.scope === "PARTY"
                            ? "bg-cyan-950/40 text-cyan-300 border-cyan-500/40"
                            : "bg-emerald-950/40 text-emerald-300 border-emerald-500/40"
                        }`}
                      >
                        {realm.scope === "SHARED"
                          ? "Bí Cảnh Chung"
                          : realm.scope === "PARTY"
                          ? "Tổ Đội"
                          : "Cá Nhân"}
                      </span>
                      <span
                        className={`flex items-center gap-1 px-2 py-0.5 text-[10px] font-medium rounded-full border ${
                          isSafe
                            ? "bg-emerald-950/30 text-emerald-300 border-emerald-500/30"
                            : "bg-rose-950/30 text-rose-300 border-rose-500/30"
                        }`}
                      >
                        {isSafe ? <Shield className="w-3 h-3" /> : <Swords className="w-3 h-3" />}
                        {isSafe ? "An Toàn" : "Tự Do PK"}
                      </span>
                    </div>

                    <p className="text-xs text-zinc-400 line-clamp-2">
                      {realm.description}
                    </p>

                    <div className="flex flex-wrap items-center gap-3 text-xs text-zinc-500 font-mono">
                      <span className="flex items-center gap-1 text-amber-400/90">
                        <MapPin className="w-3.5 h-3.5" />
                        <span>
                          {realm.entry_region_id} ({realm.entry_x}, {realm.entry_y})
                        </span>
                      </span>

                      <span className="flex items-center gap-1 text-zinc-400">
                        <Clock className="w-3.5 h-3.5" />
                        <span>{realm.duration_minutes} phút</span>
                      </span>

                      {realm.required_rank > 0 && (
                        <span className="text-zinc-400">
                          Yêu cầu: Chuyển {realm.required_rank}
                        </span>
                      )}

                      {realm.player_count > 0 && (
                        <span className="flex items-center gap-1 text-zinc-400">
                          <Users className="w-3.5 h-3.5" />
                          <span>{realm.player_count} người</span>
                        </span>
                      )}
                    </div>
                  </div>

                  <button
                    onClick={() => handleEnter(realm)}
                    disabled={spatialLoading}
                    className={`w-full sm:w-auto px-4 py-2.5 rounded-xl font-bold text-xs flex items-center justify-center gap-2 transition-all shadow-md active:scale-95 disabled:opacity-50 ${
                      isAtPortal
                        ? "bg-gradient-to-r from-amber-500 to-yellow-600 hover:from-amber-400 hover:to-yellow-500 text-zinc-950 shadow-[0_0_15px_rgba(245,158,11,0.4)]"
                        : "bg-zinc-800 hover:bg-zinc-700 text-amber-300 border border-zinc-700"
                    }`}
                  >
                    <DoorOpen className="w-4 h-4" />
                    <span>{isAtPortal ? "Tiến Vào Ngay" : "Chỉ Dẫn Cổng Vào"}</span>
                  </button>
                </div>
              );
            })
          )}
        </div>
      </div>
    </div>
  );
};
