"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { Gem, Loader2, MapPin, Pickaxe } from "lucide-react";
import { useGameStore, type ApertureResourcePoint } from "@/store/useGameStore";
import { useWorldStore } from "@/store/useWorldStore";

// Backend worldHarvestRange (Chebyshev cells). Kept in sync with
// backend/internal/handlers/aperture_points.go.
const WORLD_HARVEST_RANGE = 6;
// Canon transplantation ritual tables (worldTransplantMana* / worldTransplantRitual*).
const TRANSPLANT_MANA: Record<string, number> = { LOW: 20, MID: 30, TOP: 50 };
const TRANSPLANT_RITUAL_SECS: Record<string, number> = { LOW: 180, MID: 300, TOP: 480 };
const RITUAL_POLL_MS = 10000;

function transplantManaCost(grade: string, rank: number): number {
  const base = TRANSPLANT_MANA[grade] ?? TRANSPLANT_MANA.LOW;
  return Math.round(base * (rank > 6 ? 1 + 0.5 * (rank - 6) : 1));
}

function transplantRitualSeconds(grade: string, rank: number): number {
  const base = TRANSPLANT_RITUAL_SECS[grade] ?? TRANSPLANT_RITUAL_SECS.LOW;
  if (rank <= 6) return base;
  return Math.max(60, Math.round(base * Math.max(0.35, 1 - 0.12 * (rank - 6))));
}

function formatDuration(totalSeconds: number): string {
  const secs = Math.max(0, Math.ceil(totalSeconds));
  if (secs < 60) return `${secs} giây`;
  const mins = Math.floor(secs / 60);
  const rest = secs % 60;
  return rest === 0 ? `${mins} phút` : `${mins} phút ${rest} giây`;
}

function formatClock(totalSeconds: number): string {
  const secs = Math.max(0, Math.ceil(totalSeconds));
  return `${Math.floor(secs / 60)}:${String(secs % 60).padStart(2, "0")}`;
}

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
  const addToast = useGameStore((s) => s.addToast);
  const resourcePoints = useWorldStore((s) => s.resourcePoints);
  const resourcePointsRegion = useWorldStore((s) => s.resourcePointsRegion);
  const fetchWorldResourcePoints = useWorldStore((s) => s.fetchWorldResourcePoints);
  const collectWorldResourcePoint = useWorldStore((s) => s.collectWorldResourcePoint);
  const transplantWorldResourcePoint = useWorldStore((s) => s.transplantWorldResourcePoint);

  const [busyId, setBusyId] = useState<string | null>(null);
  const [confirmTransplant, setConfirmTransplant] = useState<string | null>(null);
  const [now, setNow] = useState(() => Date.now());
  const autoCompleteRef = useRef("");
  const autoCompleteRetryRef = useRef(0);
  const previousRitualRef = useRef<{ pointId: string } | null>(null);

  const region = player?.current_region || "";
  const rank = player?.rank ?? 6;
  const mana = Math.floor(player?.current_mana ?? 0);

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
  const activeRitual = useMemo(
    () => resourcePoints.find((point) => point.transplant)?.transplant ?? null,
    [resourcePoints],
  );
  const activeRitualPointId = activeRitual?.point_id ?? "";
  const activeRitualDueAt = activeRitual?.completes_at ?? "";

  // Tick only while a ritual is running so the countdown stays live.
  useEffect(() => {
    if (!activeRitualPointId) return;
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, [activeRitualPointId, activeRitualDueAt]);

  // Presence is authoritative on the server: the listing call is what detects
  // that the cultivator walked away (and breaks the ritual).
  useEffect(() => {
    if (!activeRitualPointId || !region) return;
    const id = window.setInterval(() => void fetchWorldResourcePoints(region), RITUAL_POLL_MS);
    return () => window.clearInterval(id);
  }, [activeRitualPointId, region, fetchWorldResourcePoints]);

  // Once the timer is up and we are still standing at the mine, finish it.
  useEffect(() => {
    if (!activeRitual) {
      autoCompleteRef.current = "";
      autoCompleteRetryRef.current = 0;
      return;
    }
    if (busyId) return;
    const due = new Date(activeRitual.completes_at).getTime();
    // Small buffer: our clock may run ahead of the server's.
    if (!Number.isFinite(due) || now < due + 1500) return;
    if (now < autoCompleteRetryRef.current) return;
    const row = rows.find((entry) => entry.point.id === activeRitual.point_id);
    if (!row || row.distance > WORLD_HARVEST_RANGE) return;
    const key = `${activeRitual.point_id}:${activeRitual.completes_at}`;
    if (autoCompleteRef.current === key) return;
    void (async () => {
      autoCompleteRef.current = key;
      setBusyId(activeRitual.point_id);
      const ok = await transplantWorldResourcePoint(activeRitual.point_id);
      setBusyId(null);
      if (!ok) {
        // Server clock behind ours, or the aperture filled up: allow a retry
        // instead of stranding the ritual until the player clicks manually.
        autoCompleteRef.current = "";
        autoCompleteRetryRef.current = Date.now() + 5000;
      }
    })();
  }, [now, activeRitual, rows, busyId, transplantWorldResourcePoint]);

  // A ritual that vanishes while its mine is still in the world was broken by
  // the server (the cultivator left); if the mine also left the listing, the
  // ritual simply completed and needs no warning.
  useEffect(() => {
    const previous = previousRitualRef.current;
    previousRitualRef.current = activeRitual ? { pointId: activeRitual.point_id } : null;
    if (!previous || activeRitual) return;
    if (!rows.some((entry) => entry.point.id === previous.pointId)) return;
    addToast("Nghi thức di dời đã đứt vì đạo hữu rời khỏi mỏ — Tiên Nguyên đã tiêu không thể thu hồi.", "error");
  }, [activeRitual, rows, addToast]);

  const startOrFinishTransplant = async (pointId: string) => {
    setBusyId(pointId);
    await transplantWorldResourcePoint(pointId);
    setBusyId(null);
    setConfirmTransplant(null);
  };

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-[#c9b28b] bg-[#f6ecd8] px-3 py-2">
        <div>
          <div className="flex items-center gap-1.5 text-sm font-bold text-[#5a3d21]">
            <Pickaxe className="h-4 w-4" /> Mỏ Tài Nguyên Công Khai
          </div>
          <div className="text-[11px] text-[#7a6144]">
            Vùng {region || "—"} · {rows.length} mỏ · {inRangeCount} có thể khai thác ngay (≤ {WORLD_HARVEST_RANGE} ô) · Tiên Nguyên{" "}
            {mana.toLocaleString("vi-VN")}
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
            const ritual = point.transplant;
            const ritualDueAt = ritual ? new Date(ritual.completes_at).getTime() : 0;
            const ritualActive = Boolean(ritual) && now < ritualDueAt;
            const ritualRemaining = ritual ? Math.max(0, (ritualDueAt - now) / 1000) : 0;
            const ritualTotal = ritual
              ? Math.max(1, (ritualDueAt - new Date(ritual.started_at).getTime()) / 1000)
              : 1;
            const almostDone = ritual ? Math.min(100, Math.max(0, (1 - ritualRemaining / ritualTotal) * 100)) : 0;
            const busy = busyId === point.id;
            const cost = transplantManaCost(point.grade, rank);
            const ritualSecs = transplantRitualSeconds(point.grade, rank);
            const otherRitual = Boolean(activeRitual && activeRitual.point_id !== point.id);
            const canStart = inRange && point.vitality >= 70 && mana >= cost && !otherRitual;
            const canFinish = Boolean(ritual) && !ritualActive && inRange;
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
                      {ritual && (
                        <span className="rounded border border-amber-600/60 bg-amber-500/15 px-1.5 py-0.5 text-[9px] font-bold text-amber-700">
                          Đang làm nghi thức
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
                  {ritual ? (
                    <button
                      type="button"
                      disabled={!canFinish || busy}
                      onClick={() => void startOrFinishTransplant(point.id)}
                      className="rounded border border-amber-700 bg-amber-600 px-2.5 py-1 text-[11px] font-semibold text-white hover:bg-amber-700 disabled:opacity-40"
                      title={
                        ritualActive
                          ? `Nghi thức còn ${formatClock(ritualRemaining)}`
                          : !inRange
                            ? `Phải đứng trong ${WORLD_HARVEST_RANGE} ô để hoàn tất nghi thức`
                            : "Hoàn tất nghi thức di dời"
                      }
                    >
                      {busy ? <Loader2 className="mr-1 inline h-3 w-3 animate-spin" /> : null}
                      {ritualActive ? `Nghi thức · ${formatClock(ritualRemaining)}` : "Hoàn tất di dời"}
                    </button>
                  ) : (
                    <button
                      type="button"
                      disabled={!canStart || busy}
                      onClick={() => setConfirmTransplant(point.id)}
                      className="rounded border border-[#a88952] px-2.5 py-1 text-[11px] font-semibold text-[#6b4f2d] hover:bg-[#efe1c6] disabled:opacity-40"
                      title={
                        otherRitual
                          ? "Đang có một nghi thức di dời khác"
                          : !inRange
                            ? `Phải đứng trong ${WORLD_HARVEST_RANGE} ô để bắt đầu nghi thức`
                            : point.vitality < 70
                              ? "Sinh khí dưới 70% — chưa thể di dời"
                              : mana < cost
                                ? `Không đủ Tiên Nguyên (cần ${cost.toLocaleString("vi-VN")})`
                                : `Tiêu hao ${cost.toLocaleString("vi-VN")} Tiên Nguyên, đứng tại mỏ ${formatDuration(ritualSecs)}`
                      }
                    >
                      Di dời · {cost.toLocaleString("vi-VN")} tiên nguyên
                    </button>
                  )}
                  {!inRange && <span className="text-[10px] text-[#9a7c55]">Hãy đến gần mỏ để thao tác.</span>}
                </div>

                {ritual && (
                  <div className="mt-2 rounded border border-amber-600/50 bg-amber-100 px-2 py-1.5 text-[10px] text-[#5a3d21]">
                    <div className="flex items-center justify-between gap-2">
                      <span>
                        {ritualActive
                          ? `Đang di dời — còn ${formatDuration(ritualRemaining)}. Đứng yên tại mỏ, rời đi sẽ hỏng nghi thức.`
                          : "Nghi thức đã đủ thời gian — hoàn tất ngay khi còn đứng tại mỏ."}
                      </span>
                      <span className="font-mono font-semibold">{Math.round(almostDone)}%</span>
                    </div>
                    <div className="mt-1 h-1.5 w-full overflow-hidden rounded bg-[#e0cda8]">
                      <div className="h-full rounded bg-amber-600 transition-all" style={{ width: `${almostDone}%` }} />
                    </div>
                  </div>
                )}

                {confirmTransplant === point.id && (
                  <div className="mt-2 rounded border border-amber-600/50 bg-amber-100 px-2 py-1.5 text-[10px] text-[#5a3d21]">
                    Bắt đầu nghi thức di dời <strong>{point.name}</strong> vào Tiên Khiếu?
                    <div className="mt-1">
                      Tiêu hao <strong>{cost.toLocaleString("vi-VN")} Tiên Nguyên</strong> ngay khi bắt đầu và phải đứng tại mỏ{" "}
                      <strong>{formatDuration(ritualSecs)}</strong>. Rời khỏi mỏ giữa chừng sẽ hỏng nghi thức và mất số Tiên Nguyên đã tiêu.
                    </div>
                    <div className="mt-1 flex gap-2">
                      <button
                        type="button"
                        className="rounded bg-amber-600 px-2 py-0.5 font-bold text-white disabled:opacity-40"
                        disabled={busy}
                        onClick={() => void startOrFinishTransplant(point.id)}
                      >
                        Bắt đầu nghi thức
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
        Di dời là <strong>nghi thức</strong>: tiêu hao Tiên Nguyên và phải đứng tại mỏ tới khi xong — mỏ vào Tiên Khiếu sẽ sản xuất nhanh hơn
        nhờ <code>time_rate</code> và Đạo Ngân.
      </div>
    </div>
  );
}
