"use client";

import { useEffect, useMemo, useState } from "react";
import { Gem, Loader2, MapPin, Pickaxe } from "lucide-react";
import { useGameStore, type ApertureResourcePoint } from "@/store/useGameStore";
import { useWorldStore } from "@/store/useWorldStore";

// Backend worldHarvestRange (Chebyshev cells). Kept in sync with
// backend/internal/handlers/aperture_points.go.
const WORLD_HARVEST_RANGE = 6;
const TRANSPLANT_PILL_ITEM = "ITEM_APERTURE_EXPANSION_PILL";
const TRANSPLANT_STONES: Record<string, number> = { LOW: 100, MID: 200, TOP: 400 };

function chebyshev(ax: number, ay: number, bx: number, by: number): number {
  return Math.max(Math.abs(ax - bx), Math.abs(ay - by));
}

function stockTotal(point: Pick<ApertureResourcePoint, "stock">): number {
  return Object.values(point.stock || {}).reduce((sum, amount) => sum + (Number.isFinite(amount) ? amount : 0), 0);
}

function stockText(point: Pick<ApertureResourcePoint, "stock">, dictionary: Record<string, { name?: string }>): string {
  const entries = Object.entries(point.stock || {}).filter(([, amount]) => amount > 0);
  if (entries.length === 0) return "Chưa tích lũy";
  return entries
    .map(([itemID, amount]) => `${dictionary[itemID]?.name || itemID} ×${amount}`)
    .join(", ");
}

function gradeLabel(grade: string): string {
  if (grade === "TOP") return "Thượng phẩm";
  if (grade === "MID") return "Trung phẩm";
  return "Hạ phẩm";
}

interface WorldResourcePointsPanelProps {
  position?: { x: number; y: number } | null;
}

export function WorldResourcePointsPanel({ position }: WorldResourcePointsPanelProps) {
  const player = useGameStore((s) => s.player);
  const itemDictionary = useGameStore((s) => s.itemDictionary);
  const resourcePoints = useWorldStore((s) => s.resourcePoints);
  const resourcePointsRegion = useWorldStore((s) => s.resourcePointsRegion);
  const fetchWorldResourcePoints = useWorldStore((s) => s.fetchWorldResourcePoints);
  const collectWorldResourcePoint = useWorldStore((s) => s.collectWorldResourcePoint);
  const transplantWorldResourcePoint = useWorldStore((s) => s.transplantWorldResourcePoint);

  const [busyId, setBusyId] = useState<string | null>(null);
  const [confirmTransplant, setConfirmTransplant] = useState<string | null>(null);

  const region = player?.current_region || "";

  useEffect(() => {
    if (region) void fetchWorldResourcePoints(region);
  }, [region, fetchWorldResourcePoints]);

  const here = position ?? { x: player?.x ?? 0, y: player?.y ?? 0 };

  const rows = useMemo(() => {
    return resourcePoints
      .map((point) => ({
        point,
        distance: chebyshev(here.x, here.y, point.x, point.y),
        total: stockTotal(point),
      }))
      .sort((a, b) => {
        const aInRange = a.distance <= WORLD_HARVEST_RANGE ? 0 : 1;
        const bInRange = b.distance <= WORLD_HARVEST_RANGE ? 0 : 1;
        if (aInRange !== bInRange) return aInRange - bInRange;
        if (a.distance !== b.distance) return a.distance - b.distance;
        if (a.point.point_class !== b.point.point_class) return a.point.point_class === "FAMOUS" ? -1 : 1;
        return a.point.name.localeCompare(b.point.name, "vi");
      });
  }, [resourcePoints, here.x, here.y]);

  const inRangeCount = rows.filter((row) => row.distance <= WORLD_HARVEST_RANGE && row.total > 0).length;
  const stones = Math.floor(player?.immortal_stones ?? 0);
  const pills = player?.materials_inventory?.[TRANSPLANT_PILL_ITEM] ?? 0;

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-[#c9b28b] bg-[#f6ecd8] px-3 py-2">
        <div>
          <div className="flex items-center gap-1.5 text-sm font-bold text-[#5a3d21]">
            <Pickaxe className="h-4 w-4" /> Mỏ Tài Nguyên Công Khai
          </div>
          <div className="text-[11px] text-[#7a6144]">
            Vùng {region || "—"} · {rows.length} mỏ · {inRangeCount} có thể khai thác ngay (≤ {WORLD_HARVEST_RANGE} ô)
          </div>
        </div>
        <button
          type="button"
          onClick={() => region && void fetchWorldResourcePoints(region)}
          className="mmo-bamboo-action px-2 py-1 text-[11px]"
        >
          Làm mới
        </button>
      </div>

      {rows.length === 0 ? (
        <div className="rounded-lg border border-[#c9b28b] bg-[#f9f1e2] px-4 py-8 text-center text-xs text-[#7a6144]">
          Vùng này chưa có mỏ công khai. Mỏ mới sẽ mọc theo thời gian — hãy quay lại sau.
        </div>
      ) : (
        <div className="space-y-2">
          {rows.map(({ point, distance, total }) => {
            const ready = point.state === "ACTIVE" && total > 0;
            const inRange = distance <= WORLD_HARVEST_RANGE;
            const famous = point.point_class === "FAMOUS";
            const stoneCost = TRANSPLANT_STONES[point.grade] ?? TRANSPLANT_STONES.LOW;
            const canTransplant = inRange && point.vitality >= 70 && stones >= stoneCost && pills > 0;
            const busy = busyId === point.id;
            return (
              <div
                key={point.id}
                className={`rounded-lg border px-3 py-2 ${famous ? "border-amber-500/70 bg-amber-50" : "border-[#c9b28b] bg-[#f9f1e2]"}`}
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-1.5">
                      <span className="truncate text-xs font-bold text-[#4a3117]">{point.name}</span>
                      <span className="rounded border border-[#bfa77f] px-1.5 py-0.5 text-[9px] font-semibold text-[#6b4f2d]">
                        {gradeLabel(point.grade)}
                      </span>
                      {famous && (
                        <span className="rounded border border-amber-600/60 bg-amber-500/15 px-1.5 py-0.5 text-[9px] font-bold text-amber-700">
                          Núi Danh
                        </span>
                      )}
                      {point.state === "EXHAUSTED" && (
                        <span className="rounded border border-rose-500/50 bg-rose-500/10 px-1.5 py-0.5 text-[9px] font-bold text-rose-600">
                          Cạn kiệt
                        </span>
                      )}
                    </div>
                    <div className="mt-1 flex flex-wrap items-center gap-2 text-[10px] text-[#7a6144]">
                      <span className="inline-flex items-center gap-0.5">
                        <MapPin className="h-3 w-3" />({point.x}, {point.y})
                      </span>
                      <span className={inRange ? "font-semibold text-emerald-700" : ""}>
                        cách {distance} ô{inRange ? " · đủ tầm" : ""}
                      </span>
                      <span>·</span>
                      <span>Sinh khí {point.vitality}%</span>
                      <span>·</span>
                      <span>
                        Tồn kho {total}/{point.stock_cap}
                      </span>
                    </div>
                    <div className="mt-1 text-[10px] text-[#6b4f2d]">{stockText(point, itemDictionary)}</div>
                  </div>
                </div>

                <div className="mt-2 flex flex-wrap items-center gap-1.5">
                  <button
                    type="button"
                    disabled={!ready || !inRange || busy}
                    onClick={async () => {
                      setBusyId(point.id);
                      await collectWorldResourcePoint(point.id);
                      setBusyId(null);
                    }}
                    className="mmo-bamboo-action inline-flex items-center gap-1 px-2.5 py-1 text-[11px] disabled:opacity-40"
                    title={!inRange ? `Phải đứng trong ${WORLD_HARVEST_RANGE} ô` : !ready ? "Chưa có tài nguyên để khai thác" : "Khai thác mỏ"}
                  >
                    {busy ? <Loader2 className="h-3 w-3 animate-spin" /> : <Gem className="h-3 w-3" />} Khai thác
                  </button>
                  <button
                    type="button"
                    disabled={!canTransplant || busy}
                    onClick={() => setConfirmTransplant(point.id)}
                    className="rounded border border-[#a88952] px-2.5 py-1 text-[11px] font-semibold text-[#6b4f2d] hover:bg-[#efe1c6] disabled:opacity-40"
                    title={`Cần 1 Thác Khiếu Đan + ${stoneCost} Tiên Nguyên Thạch, sinh khí ≥ 70%`}
                  >
                    Di dời · {stoneCost} đá + 1 đan
                  </button>
                  {!inRange && <span className="text-[10px] text-[#9a7c55]">Hãy đến gần mỏ để thao tác.</span>}
                </div>

                {confirmTransplant === point.id && (
                  <div className="mt-2 rounded border border-amber-600/50 bg-amber-100 px-2 py-1.5 text-[10px] text-[#5a3d21]">
                    Di dời <strong>{point.name}</strong> vào Tiên Khiếu? Mỏ sẽ rời khỏi vùng này và bắt đầu ăn gia tốc thời gian + Đạo Ngân của đạo hữu.
                    <div className="mt-1 flex gap-2">
                      <button
                        type="button"
                        className="rounded bg-amber-600 px-2 py-0.5 font-bold text-white"
                        onClick={async () => {
                          setBusyId(point.id);
                          await transplantWorldResourcePoint(point.id);
                          setBusyId(null);
                          setConfirmTransplant(null);
                        }}
                      >
                        Xác nhận
                      </button>
                      <button type="button" className="rounded border border-[#a88952] px-2 py-0.5" onClick={() => setConfirmTransplant(null)}>
                        Hủy
                      </button>
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      <div className="rounded-lg border border-[#d8c6a4] bg-[#fbf5e8] px-3 py-2 text-[10px] leading-relaxed text-[#7a6144]">
        Mỏ công khai chạy theo thời gian thực (không hưởng gia tốc Tiên Khiếu). Quái canh giữ ô phải bị đánh bại trước khi khai thác.
        Di dời cần <strong>Thác Khiếu Đan</strong> (đang có {pills}) — mỏ vào Tiên Khiếu sẽ sản xuất nhanh hơn nhờ <code>time_rate</code> và Đạo Ngân.
      </div>
    </div>
  );
}
