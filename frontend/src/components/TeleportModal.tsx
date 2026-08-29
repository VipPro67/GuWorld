"use client";

import { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import { useWorldStore } from "@/store/useWorldStore";

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

export interface TeleportResponse {
  player: any;
  x: number;
  y: number;
  region: string;
  cost: number;
  currency: string;
  message: string;
}

function TeleportModal({ isOpen, onClose }: { isOpen: boolean; onClose: () => void }) {
  const [regions, setRegions] = useState<RegionInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [teleporting, setTeleporting] = useState(false);
  const [tab, setTab] = useState<"standard" | "gu">("standard");
  const [targetX, setTargetX] = useState<string>("");
  const [targetY, setTargetY] = useState<string>("");

  const player = useGameStore((s) => s.player);
  const guDictionary = useGameStore((s) => s.guDictionary);
  const currentX = useWorldStore((s) => s.liveX);
  const currentY = useWorldStore((s) => s.liveY);
  const teleportToRegion = useWorldStore((s) => s.teleportToRegion);
  const useGuOnMap = useWorldStore((s) => s.useGuOnMap);

  // Filter Portal Gu
  const portalGus = player?.gu_inventory?.filter((gu) => {
    const spec = guDictionary[gu.gu_id];
    return spec?.effects?.active?.portal_create === true;
  }) ?? [];
  const [selectedGu, setSelectedGu] = useState<string>("");

  useEffect(() => {
    if (portalGus.length > 0 && !selectedGu) {
      setSelectedGu(portalGus[0].instance_id);
    }
  }, [portalGus, selectedGu]);

  const rank = player?.rank ?? 1;
  const subRank = player?.sub_rank ?? 1;
  const inventorySize = player?.gu_inventory?.length ?? 0;
  const storage = 100; // base storage, actual may vary with passives
  const primevalStones = player?.primeval_stones ?? 0;
  const immortalStones = player?.immortal_stones ?? 0;

  // Fetch regions on open
  useEffect(() => {
    if (!isOpen) return;
    const fetchRegions = async () => {
      setLoading(true);
      try {
        const res = await fetch(`${process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080"}/api/explore/regions`, {
          headers: { Authorization: `Bearer ${localStorage.getItem("gu_auth_token")}` },
        });
        if (!res.ok) throw new Error("Failed to fetch regions");
        const data = await res.json();
        setRegions(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        setLoading(false);
      }
    };
    fetchRegions();
  }, [isOpen]);

  // Calculate cost for a region
  const calculateCost = (region: RegionInfo) => {
    if (!player) return 0;
    const safeX = region.safe_zone_x ?? region.entry_x;
    const safeY = region.safe_zone_y ?? region.entry_y;
    const dx = Math.abs(currentX - safeX);
    const dy = Math.abs(currentY - safeY);
    const distance = Math.max(dx, dy);

    const baseCost = distance * 0.5;
    const rankMult = 1.0 + (rank - 1) * 0.15 + subRank * 0.03;
    const maxSlots = rank >= 6 ? 999 : storage;
    const inventoryRatio = maxSlots > 0 ? inventorySize / maxSlots : 0;
    const inventoryMult = 1.0 + inventoryRatio * 0.5;

    let total = baseCost * rankMult * inventoryMult;
    if (total < 10) total = 10;
    if (total > 10000) total = 10000;
    return Math.round(total);
  };

  const handleTeleport = async (region: RegionInfo) => {
    if (teleporting) return;
    setTeleporting(true);
    setError(null);
    try {
      const resp = await teleportToRegion(region.id);
      if (resp) {
        onClose();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setTeleporting(false);
    }
  };

  const handleGuTeleport = async () => {
    if (!selectedGu || !targetX || !targetY) return;
    setTeleporting(true);
    setError(null);
    try {
      const res = await useGuOnMap(selectedGu, parseInt(targetX), parseInt(targetY));
      if (res) {
        useGameStore.getState().addToast({ type: "success", title: "Thành công", message: res.message });
        onClose();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setTeleporting(false);
    }
  };

  if (!isOpen) return null;

  // Filter to KHU_VUC and NGU_VUC zones that are active
  const availableRegions = regions
    .filter((r) => r.is_active && (r.zone_type === "KHU_VUC" || r.zone_type === "NGU_VUC"))
    .map((r) => ({
      ...r,
      cost: calculateCost(r),
      currency: rank >= 6 && calculateCost(r) > 5000 ? "immortal_stones" : "primeval_stones",
    }))
    .sort((a, b) => a.cost - b.cost);

  const currentRegionId = player?.current_region ?? "";

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-gray-900 border border-gray-700 rounded-lg w-full max-w-2xl max-h-[80vh] overflow-hidden flex flex-col">
        <div className="flex items-center justify-between p-4 border-b border-gray-700">
          <h2 className="text-xl font-bold text-yellow-300">Di Chuyển Nhanh</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-white">
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg>
          </button>
        </div>

        <div className="flex border-b border-gray-700">
          <button
            className={`flex-1 py-2 text-sm font-medium ${tab === "standard" ? "text-yellow-300 border-b-2 border-yellow-300" : "text-gray-400 hover:text-white"}`}
            onClick={() => setTab("standard")}
          >
            Trạm Dịch Chuyển
          </button>
          {portalGus.length > 0 && (
            <button
              className={`flex-1 py-2 text-sm font-medium ${tab === "gu" ? "text-purple-300 border-b-2 border-purple-300" : "text-gray-400 hover:text-white"}`}
              onClick={() => setTab("gu")}
            >
              Cổ Trùng Dịch Chuyển
            </button>
          )}
        </div>

        <div className="p-4 border-b border-gray-700 bg-gray-800/50 flex flex-wrap gap-4 text-sm">
          <div className="text-gray-300">Vị trí hiện tại: <span className="text-white">({currentX}, {currentY})</span></div>
          <div className="text-gray-300">Khu vực: <span className="text-white">{currentRegionId || "Không xác định"}</span></div>
          <div className="text-gray-300">Primeval: <span className="text-yellow-300">{primevalStones.toLocaleString()}</span></div>
          <div className="text-gray-300">Immortal: <span className="text-purple-300">{immortalStones.toLocaleString()}</span></div>
          <div className="text-gray-300">Tu vi: <span className="text-green-300">Cấp {rank}.{subRank}</span></div>
          <div className="text-gray-300">Túi đồ: <span className="text-blue-300">{inventorySize}/{rank >= 6 ? "∞" : storage}</span></div>
        </div>

        {error && <div className="p-4 text-red-400 border-b border-red-900/50">{error}</div>}

        <div className="flex-1 overflow-y-auto p-4">
          {tab === "standard" ? (
            loading ? (
              <div className="text-center text-gray-400 py-8">Đang tải danh sách khu vực...</div>
            ) : (
              <div className="space-y-2">
                {availableRegions.map((region) => {
                  const canAfford = region.currency === "primeval_stones"
                    ? primevalStones >= region.cost
                    : immortalStones >= region.cost;
                  const isCurrent = region.id === currentRegionId;
                  return (
                    <button
                      key={region.id}
                      onClick={() => handleTeleport(region)}
                      disabled={teleporting || !canAfford || isCurrent}
                      className={`w-full text-left p-3 rounded border transition ${isCurrent
                        ? "bg-gray-800 border-gray-700 text-gray-500 cursor-not-allowed"
                        : canAfford
                        ? "bg-gray-800 border-gray-600 hover:border-yellow-400 hover:bg-gray-700"
                        : "bg-gray-800 border-gray-700 text-gray-500 cursor-not-allowed opacity-50"}`}
                    >
                      <div className="flex justify-between items-start">
                        <div>
                          <div className="font-medium text-white">{region.name}</div>
                          <div className="text-xs text-gray-400">{region.id} • {region.zone_type}</div>
                          <div className="text-xs text-gray-500">An toàn: ({region.safe_zone_x ?? region.entry_x}, {region.safe_zone_y ?? region.entry_y})</div>
                        </div>
                        <div className="text-right">
                          <div className={`font-bold ${canAfford ? "text-green-300" : "text-red-300"}`}>
                            {region.cost.toLocaleString()}
                          </div>
                          <div className="text-xs text-gray-400">
                            {region.currency === "primeval_stones" ? "💎 Primeval" : "💜 Immortal"}
                          </div>
                        </div>
                      </div>
                      {isCurrent && <div className="text-xs text-gray-500 mt-1">Đang ở khu vực này</div>}
                      {!canAfford && !isCurrent && (
                        <div className="text-xs text-red-400 mt-1">Không đủ tiền</div>
                      )}
                    </button>
                  );
                })}
                {availableRegions.length === 0 && (
                  <div className="text-center text-gray-400 py-8">Không có khu vực khả dụng</div>
                )}
              </div>
            )
          ) : (
            <div className="space-y-4">
              <div>
                <label className="block text-sm text-gray-400 mb-1">Chọn Cổ Trùng</label>
                <select 
                  className="w-full bg-gray-800 border border-gray-600 rounded p-2 text-white"
                  value={selectedGu}
                  onChange={(e) => setSelectedGu(e.target.value)}
                >
                  {portalGus.map(g => (
                    <option key={g.instance_id} value={g.instance_id}>
                      {guDictionary[g.gu_id]?.name || g.gu_id} (Rank {g.rank})
                    </option>
                  ))}
                </select>
              </div>
              <div className="flex gap-4">
                <div className="flex-1">
                  <label className="block text-sm text-gray-400 mb-1">Tọa độ X</label>
                  <input 
                    type="number" 
                    value={targetX} 
                    onChange={e => setTargetX(e.target.value)}
                    className="w-full bg-gray-800 border border-gray-600 rounded p-2 text-white" 
                    placeholder="Ví dụ: 100" 
                  />
                </div>
                <div className="flex-1">
                  <label className="block text-sm text-gray-400 mb-1">Tọa độ Y</label>
                  <input 
                    type="number" 
                    value={targetY} 
                    onChange={e => setTargetY(e.target.value)}
                    className="w-full bg-gray-800 border border-gray-600 rounded p-2 text-white" 
                    placeholder="Ví dụ: 100" 
                  />
                </div>
              </div>
              <button 
                onClick={handleGuTeleport}
                disabled={teleporting || !targetX || !targetY || !selectedGu}
                className="w-full mt-4 bg-purple-900/50 border border-purple-500/50 hover:bg-purple-800/50 text-purple-200 p-3 rounded font-bold disabled:opacity-50"
              >
                Kích Hoạt Dịch Chuyển
              </button>
            </div>
          )}
        </div>

        {teleporting && (
          <div className="p-4 border-t border-gray-700 bg-gray-800/50 text-center text-yellow-300">
            Đang dịch chuyển...
          </div>
        )}
      </div>
    </div>
  );
}

export default TeleportModal;