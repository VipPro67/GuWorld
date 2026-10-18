"use client";

import { useEffect, useState, useMemo } from "react";
import { createPortal } from "react-dom";
import { canCreatePortal, portalRange, usesRandomPortal } from "@/lib/effect-blocks";
import { guSpecFor, useGameStore } from "@/store/useGameStore";
import { useWorldStore } from "@/store/useWorldStore";
import { SearchableSelect } from "@/components/ui";

export interface RegionInfo {
  id: string;
  name: string;
  zone_type: string;
  min_x: number;
  min_y: number;
  max_x: number;
  max_y: number;
  entry_x: number;
  entry_y: number;
  safe_zone_x: number;
  safe_zone_y: number;
  is_active: boolean;
}

function TeleportModal({ isOpen, onClose }: { isOpen: boolean; onClose: () => void }) {
  const [regions, setRegions] = useState<RegionInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [teleporting, setTeleporting] = useState(false);
  const [targetX, setTargetX] = useState<string>("");
  const [targetY, setTargetY] = useState<string>("");
  const [selectedRegionId, setSelectedRegionId] = useState<string>("");

  const player = useGameStore((s) => s.player);
  const storeRegions = useGameStore((s) => s.regions);
  const fetchRegionsStore = useGameStore((s) => s.fetchRegions);
  const currentX = useWorldStore((s) => s.liveX);
  const currentY = useWorldStore((s) => s.liveY);
  const useGuOnMap = useWorldStore((s) => s.useGuOnMap);

  // Filter Portal Gu
  const portalGus = player?.gu_inventory?.filter((gu) => {
    const spec = guSpecFor(gu.gu_id, gu.rank);
    return gu.equipped && gu.status === "ACTIVE" && (canCreatePortal(spec?.effects) || usesRandomPortal(spec?.effects));
  }) ?? [];
  const [selectedGu, setSelectedGu] = useState<string>("");
  const activeGu = player?.gu_inventory?.find(g => g.instance_id === selectedGu);
  const activeSpec = activeGu ? guSpecFor(activeGu.gu_id, activeGu.rank) : null;
  const isRandomPortal = usesRandomPortal(activeSpec?.effects);
  const teleportRange = activeSpec ? portalRange(activeSpec.effects, activeGu?.rank ?? player?.rank ?? 1) : 0;

  useEffect(() => {
    if (portalGus.length > 0 && !selectedGu) {
      const eligible = portalGus.find((g) => g.rank >= (player?.rank ?? 1));
      setSelectedGu(eligible ? eligible.instance_id : portalGus[0].instance_id);
    }
  }, [portalGus, selectedGu, player?.rank]);

  // Sync store regions to local state
  useEffect(() => {
    if (storeRegions && storeRegions.length > 0) {
      setRegions(storeRegions as unknown as RegionInfo[]);
    }
  }, [storeRegions]);

  // Fetch regions on open for target region coordinate selection helper
  useEffect(() => {
    if (!isOpen) return;
    if (storeRegions && storeRegions.length > 0) {
      setRegions(storeRegions as unknown as RegionInfo[]);
      return;
    }
    void fetchRegionsStore();
  }, [isOpen, storeRegions, fetchRegionsStore]);

  const handleSelectRegion = (regionId: string) => {
    setSelectedRegionId(regionId);
    const region = regions.find((r) => r.id === regionId);
    if (region) {
      const sx = region.safe_zone_x ?? region.entry_x;
      const sy = region.safe_zone_y ?? region.entry_y;
      setTargetX(String(sx));
      setTargetY(String(sy));
    }
  };

  const isRankSufficient = Boolean(activeGu && player && activeGu.rank >= player.rank);

  const handleGuTeleport = async () => {
    if (!selectedGu || !isRankSufficient) return;
    if (!isRandomPortal && (!targetX || !targetY)) return;

    const tx = isRandomPortal ? 0 : parseInt(targetX);
    const ty = isRandomPortal ? 0 : parseInt(targetY);
    if (!isRandomPortal && teleportRange > 0) {
      const dist = Math.max(Math.abs(tx - currentX), Math.abs(ty - currentY));
      if (dist > teleportRange) {
        setError(`Cổ trùng ${activeSpec?.name ?? "này"} chỉ dịch chuyển tối đa ${teleportRange} ô (mục tiêu cách ${dist} ô).`);
        return;
      }
    }

    setTeleporting(true);
    setError(null);
    try {
      const res = await useGuOnMap(selectedGu, tx, ty);
      if (res) {
        useGameStore.getState().addToast(res.message, "success");
        onClose();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setTeleporting(false);
    }
  };

  const [mounted, setMounted] = useState(false);
  useEffect(() => {
    setMounted(true);
  }, []);

  const availableRegions = regions
    .filter((r) => r.is_active && (r.zone_type === "KHU_VUC" || r.zone_type === "NGU_VUC"))
    .sort((a, b) => a.name.localeCompare(b.name));

  const portalGuOptions = useMemo(() => {
    return portalGus.map((g) => {
      const spec = guSpecFor(g.gu_id, g.rank);
      const meetsRank = g.rank >= (player?.rank ?? 1);
      return {
        value: g.instance_id,
        label: `${spec?.name || g.gu_id} (Rank ${g.rank})`,
        badge: meetsRank ? "Hợp Lệ" : `Yêu Cầu Rank ≥ ${player?.rank ?? 1}`,
        badgeColor: meetsRank ? "bg-emerald-950 text-emerald-300 border-emerald-800" : "bg-red-950 text-red-300 border-red-800",
      };
    });
  }, [portalGus, player?.rank]);

  const regionOptions = useMemo(() => {
    return availableRegions.map((r) => ({
      value: r.id,
      label: r.name,
      subLabel: `An toàn: (${r.safe_zone_x ?? r.entry_x}, ${r.safe_zone_y ?? r.entry_y})`,
    }));
  }, [availableRegions]);

  // All hooks must run on every render, so the render gate sits after them.
  if (!isOpen || !mounted) return null;

  const currentRegionId = player?.current_region ?? "";

  return createPortal(
    <div className="fixed inset-0 z-[9999] h-[100dvh] bg-black/75 backdrop-blur-md flex items-center justify-center px-2 pt-[calc(env(safe-area-inset-top)+0.5rem)] pb-[calc(env(safe-area-inset-bottom)+0.5rem)] sm:px-4">
      <div className="bg-zinc-950 border border-purple-500/40 rounded-3xl w-full max-w-lg max-h-[calc(100dvh-env(safe-area-inset-top)-env(safe-area-inset-bottom)-1rem)] overflow-hidden flex flex-col shadow-2xl animate-in fade-in zoom-in-95 duration-200 text-left">
        <div className="shrink-0 flex items-center justify-between px-6 py-4 border-b border-zinc-800 bg-gradient-to-r from-purple-950/30 via-zinc-900/40 to-transparent">
          <div>
            <h2 className="text-base font-bold text-purple-300 font-serif">Dịch Chuyển Cổ Trùng</h2>
            <p className="text-[11px] text-zinc-400 font-mono">Yêu cầu: Cổ dịch chuyển Rank ≥ Rank Cổ sư (Rank {player?.rank ?? 1})</p>
          </div>
          <button onClick={onClose} className="p-1.5 rounded-xl text-zinc-400 hover:text-white hover:bg-zinc-900 transition-colors cursor-pointer">
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg>
          </button>
        </div>

        <div className="shrink-0 px-6 py-2.5 border-b border-zinc-800/80 bg-zinc-900/50 flex flex-wrap gap-4 text-xs font-mono">
          <div className="text-zinc-400">Vị trí hiện tại: <span className="text-emerald-400 font-bold">({currentX}, {currentY})</span></div>
          <div className="text-zinc-400">Khu vực: <span className="text-amber-400 font-bold">{currentRegionId || "Không xác định"}</span></div>
          <div className="text-zinc-400">Cảnh giới: <span className="text-cyan-400 font-bold">Rank {player?.rank ?? 1}</span></div>
          <div className="text-zinc-400">Tầm dịch chuyển: <span className="text-purple-400 font-bold">{!activeGu ? "—" : teleportRange > 0 ? `${teleportRange} ô` : "Vô hạn"}</span></div>
        </div>

        {error && <div className="shrink-0 px-6 py-2.5 text-xs font-mono text-red-400 bg-red-950/30 border-b border-red-900/50">{error}</div>}

        <div className="flex-1 min-h-0 overflow-y-auto p-4 sm:p-6 custom-scrollbar">
          {portalGus.length === 0 ? (
            <div className="p-4 bg-amber-950/40 border border-amber-500/40 rounded-lg text-amber-200 text-sm space-y-2">
              <div className="font-bold text-amber-300 text-base">⚠️ Chưa có Cổ trùng dịch chuyển phù hợp</div>
              <p>
                Cơ chế dịch chuyển bằng Linh Thạch / Nguyên Thạch đã bị hủy bỏ.
              </p>
              <p>
                Để thực hiện dịch chuyển, đạo hữu cần trang bị Cổ trùng có khả năng Dịch Chuyển / Tạo Cổng (như Định Tiên Du, Môn Vực Cổ, Thần Du Fake,...) và thiết lập ở trạng thái <span className="text-green-400 font-semibold">ACTIVE</span>.
              </p>
            </div>
          ) : (
            <div className="space-y-4">
              <div>
                <label className="block text-sm text-gray-400 mb-1 font-medium font-serif">Chọn Cổ Trùng Dịch Chuyển</label>
                <SearchableSelect
                  options={portalGuOptions}
                  value={selectedGu}
                  onChange={setSelectedGu}
                  placeholder="Chọn Cổ trùng dịch chuyển..."
                  searchPlaceholder="Tìm Cổ trùng..."
                />
              </div>

              {!isRankSufficient && activeGu && player && (
                <div className="p-3 bg-red-950/40 border border-red-500/40 rounded-xl text-xs font-mono text-red-300 space-y-1">
                  <div className="font-bold flex items-center gap-1.5 text-red-400">
                    <span>⚠️</span>
                    <span>Phẩm cấp Cổ trùng không đủ</span>
                  </div>
                  <p className="text-[11px] leading-relaxed text-zinc-300">
                    Cổ trùng <strong>{activeSpec?.name}</strong> có phẩm cấp (Rank {activeGu.rank}) thấp hơn cảnh giới bản thân (Rank {player.rank}). Để dịch chuyển bản thể, Cổ trùng dịch chuyển phải có phẩm cấp tối thiểu bằng cảnh giới Cổ sư (Rank ≥ {player.rank})!
                  </p>
                </div>
              )}

              {isRandomPortal ? (
                <div className="p-3 bg-purple-950/40 border border-purple-500/30 rounded text-purple-300 text-sm">
                  ✨ Cổ trùng này sẽ dịch chuyển đạo hữu tới một vị trí ngẫu nhiên
                  {teleportRange > 0 ? ` trong phạm vi ${teleportRange} ô` : " trên thế giới"}. Không cần nhập tọa độ.
                </div>
              ) : (
                <div className="space-y-3">
                  <div>
                    <label className="block text-sm text-gray-400 mb-1 font-medium font-serif">Chọn nhanh vùng an toàn khu vực</label>
                    <SearchableSelect
                      options={regionOptions}
                      value={selectedRegionId}
                      onChange={handleSelectRegion}
                      placeholder="-- Chọn khu vực mục tiêu (Tùy chọn) --"
                      searchPlaceholder="Tìm khu vực..."
                      clearable
                      disabled={loading}
                    />
                  </div>

                  <div className="flex gap-4">
                    <div className="flex-1">
                      <label className="block text-xs text-gray-400 mb-1 font-mono">Tọa độ X</label>
                      <input
                        type="number"
                        value={targetX}
                        onChange={e => setTargetX(e.target.value)}
                        className="w-full bg-zinc-900 border border-zinc-700 rounded-xl p-2 text-white text-xs font-mono"
                        placeholder="Ví dụ: 100"
                      />
                    </div>
                    <div className="flex-1">
                      <label className="block text-xs text-gray-400 mb-1 font-mono">Tọa độ Y</label>
                      <input
                        type="number"
                        value={targetY}
                        onChange={e => setTargetY(e.target.value)}
                        className="w-full bg-zinc-900 border border-zinc-700 rounded-xl p-2 text-white text-xs font-mono"
                        placeholder="Ví dụ: 100"
                      />
                    </div>
                  </div>
                </div>
              )}

              <button
                onClick={handleGuTeleport}
                disabled={teleporting || !isRankSufficient || (!isRandomPortal && (!targetX || !targetY)) || !selectedGu}
                className="w-full mt-4 bg-purple-600 hover:bg-purple-500 disabled:bg-zinc-800 disabled:text-zinc-500 border border-purple-400/30 disabled:border-zinc-700 text-white p-3 rounded-xl font-bold font-serif disabled:opacity-60 transition shadow-lg shadow-purple-950/50 cursor-pointer disabled:cursor-not-allowed"
              >
                {teleporting
                  ? "Đang Dịch Chuyển..."
                  : !isRankSufficient
                  ? `Yêu Cầu Cổ Dịch Chuyển Rank ≥ ${player?.rank ?? 1}`
                  : "Kích Hoạt Cổ Trùng Dịch Chuyển"}
              </button>
            </div>
          )}
        </div>

        {teleporting && (
          <div className="shrink-0 p-3 border-t border-zinc-800 bg-zinc-900/60 text-center text-xs font-mono text-purple-300 font-medium">
            Đang thôi động Cổ trùng xé rách không gian...
          </div>
        )}
      </div>
    </div>,
    document.body
  );
}

export default TeleportModal;

