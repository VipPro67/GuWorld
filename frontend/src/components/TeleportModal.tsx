"use client";

import { useEffect, useState } from "react";
import { guSpecFor, useGameStore } from "@/store/useGameStore";
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

function TeleportModal({ isOpen, onClose }: { isOpen: boolean; onClose: () => void }) {
  const [regions, setRegions] = useState<RegionInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [teleporting, setTeleporting] = useState(false);
  const [targetX, setTargetX] = useState<string>("");
  const [targetY, setTargetY] = useState<string>("");
  const [selectedRegionId, setSelectedRegionId] = useState<string>("");

  const player = useGameStore((s) => s.player);
  const currentX = useWorldStore((s) => s.liveX);
  const currentY = useWorldStore((s) => s.liveY);
  const useGuOnMap = useWorldStore((s) => s.useGuOnMap);

  // Filter Portal Gu
  const portalGus = player?.gu_inventory?.filter((gu) => {
    const spec = guSpecFor(gu.gu_id, gu.rank);
    return gu.equipped && gu.status === "ACTIVE" && (spec?.effects?.active?.portal_create === true || spec?.effects?.active?.random_portal === true);
  }) ?? [];
  const [selectedGu, setSelectedGu] = useState<string>("");
  const activeGu = player?.gu_inventory?.find(g => g.instance_id === selectedGu);
  const activeSpec = activeGu ? guSpecFor(activeGu.gu_id, activeGu.rank) : null;
  const isRandomPortal = activeSpec?.effects?.active?.random_portal === true;

  useEffect(() => {
    if (portalGus.length > 0 && !selectedGu) {
      setSelectedGu(portalGus[0].instance_id);
    }
  }, [portalGus, selectedGu]);

  // Fetch regions on open for target region coordinate selection helper
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

  const handleGuTeleport = async () => {
    if (!selectedGu) return;
    if (!isRandomPortal && (!targetX || !targetY)) return;

    setTeleporting(true);
    setError(null);
    try {
      const tx = isRandomPortal ? 0 : parseInt(targetX);
      const ty = isRandomPortal ? 0 : parseInt(targetY);
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

  if (!isOpen) return null;

  const availableRegions = regions
    .filter((r) => r.is_active && (r.zone_type === "KHU_VUC" || r.zone_type === "NGU_VUC"))
    .sort((a, b) => a.name.localeCompare(b.name));

  const currentRegionId = player?.current_region ?? "";

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-gray-900 border border-gray-700 rounded-lg w-full max-w-lg max-h-[80vh] overflow-hidden flex flex-col">
        <div className="flex items-center justify-between p-4 border-b border-gray-700">
          <h2 className="text-xl font-bold text-purple-300">Dịch Chuyển Cổ Trùng</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-white">
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" /></svg>
          </button>
        </div>

        <div className="p-4 border-b border-gray-700 bg-gray-800/50 flex flex-wrap gap-4 text-sm">
          <div className="text-gray-300">Vị trí hiện tại: <span className="text-white">({currentX}, {currentY})</span></div>
          <div className="text-gray-300">Khu vực: <span className="text-white">{currentRegionId || "Không xác định"}</span></div>
        </div>

        {error && <div className="p-4 text-red-400 border-b border-red-900/50">{error}</div>}

        <div className="flex-1 overflow-y-auto p-4">
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
                <label className="block text-sm text-gray-400 mb-1 font-medium">Chọn Cổ Trùng Dịch Chuyển</label>
                <select
                  className="w-full bg-gray-800 border border-purple-500/50 rounded p-2.5 text-white focus:outline-none focus:border-purple-400"
                  value={selectedGu}
                  onChange={(e) => setSelectedGu(e.target.value)}
                >
                  {portalGus.map(g => (
                    <option key={g.instance_id} value={g.instance_id}>
                      {guSpecFor(g.gu_id, g.rank)?.name || g.gu_id} (Cấp {g.rank})
                    </option>
                  ))}
                </select>
              </div>

              {isRandomPortal ? (
                <div className="p-3 bg-purple-950/40 border border-purple-500/30 rounded text-purple-300 text-sm">
                  ✨ Cổ trùng này sẽ dịch chuyển đạo hữu tới một vị trí ngẫu nhiên trên thế giới. Không cần nhập tọa độ.
                </div>
              ) : (
                <div className="space-y-3">
                  <div>
                    <label className="block text-sm text-gray-400 mb-1 font-medium">Chọn nhanh vùng an toàn khu vực</label>
                    <select
                      className="w-full bg-gray-800 border border-gray-600 rounded p-2 text-white text-sm"
                      value={selectedRegionId}
                      onChange={(e) => handleSelectRegion(e.target.value)}
                      disabled={loading}
                    >
                      <option value="">-- Chọn khu vực mục tiêu (Tùy chọn) --</option>
                      {availableRegions.map((r) => (
                        <option key={r.id} value={r.id}>
                          {r.name} (An toàn: {r.safe_zone_x ?? r.entry_x}, {r.safe_zone_y ?? r.entry_y})
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
                </div>
              )}

              <button
                onClick={handleGuTeleport}
                disabled={teleporting || (!isRandomPortal && (!targetX || !targetY)) || !selectedGu}
                className="w-full mt-4 bg-purple-900/60 border border-purple-500 hover:bg-purple-800 text-purple-100 p-3 rounded-lg font-bold disabled:opacity-50 transition shadow-lg shadow-purple-950/50"
              >
                Kích Hoạt Cổ Trùng Dịch Chuyển
              </button>
            </div>
          )}
        </div>

        {teleporting && (
          <div className="p-4 border-t border-gray-700 bg-gray-800/50 text-center text-purple-300 font-medium">
            Đang thôi động Cổ trùng xé rách không gian...
          </div>
        )}
      </div>
    </div>
  );
}

export default TeleportModal;

