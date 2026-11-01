"use client";

import React, { useEffect, useMemo, useState } from "react";
import { Gem, RefreshCw, Sprout, TrendingUp, Users } from "lucide-react";
import { GameButton, GamePanel, GameProgressBar } from "@/components/ui";
import { translatePath } from "@/lib/translate";
import { formatNumber } from "@/lib/utils";
import { useGameStore } from "@/store/useGameStore";
import type { ApertureResourcePoint } from "@/store/useGameStore";

const POINT_GLYPHS: Record<string, string> = {
  HERB_VALLEY: "🌿",
  BEAST_PASTURE: "🐄",
  ORE_VEIN: "⛏️",
  SPIRIT_SPRING: "💧",
  GOLD_SAND_SHOAL: "🏖️",
  MOON_BOWL: "🌙",
  YEAR_ESSENCE_POOL: "🦬",
  DANG_HON_PEAK: "👻",
  JADE_SNAKE_CAVE: "🐍",
  PHOENIX_PEAK: "🔥",
  FROSTY_PEAK: "❄️",
  CORPSE_RIDGE: "🪦",
};

export function pointGlyph(point: Pick<ApertureResourcePoint, "point_type">): string {
  return POINT_GLYPHS[point.point_type] || "💎";
}

export function pointStockTotal(point: Pick<ApertureResourcePoint, "stock">): number {
  return Object.values(point.stock || {}).reduce((sum, amount) => sum + (Number.isFinite(amount) ? amount : 0), 0);
}

export function pointIsReady(point: Pick<ApertureResourcePoint, "stock" | "state">): boolean {
  return point.state === "ACTIVE" && pointStockTotal(point) > 0;
}

const YIELD_FACTOR_LABELS: Record<string, string> = {
  level: "Cấp điểm",
  dao: "Đạo Ngân cùng phái",
  area: "Diện tích Tiên Khiếu",
  conflict: "Xung đột Đạo Ngân",
  ecology: "Độ thuần khiết sinh thái",
  labor: "Cường độ lao động",
  pop: "Dân số dị nhân",
};

const GRADE_STYLES: Record<string, string> = {
  LOW: "border-zinc-700 bg-zinc-900/70 text-zinc-300",
  MID: "border-cyan-500/40 bg-cyan-500/10 text-cyan-300",
  TOP: "border-amber-500/40 bg-amber-500/10 text-amber-300",
};

function gradeBadgeClass(grade: string): string {
  return GRADE_STYLES[grade] || GRADE_STYLES.LOW;
}

function pointUpgradeCost(level: number): number {
  return Math.floor(30 * level * 1.5);
}

function stockLabel(itemID: string, itemDictionary: Record<string, { name?: string }>): string {
  return itemDictionary?.[itemID]?.name || itemID;
}

interface WorkerAssignmentProps {
  point: ApertureResourcePoint;
}

function PointWorkerEditor({ point }: WorkerAssignmentProps) {
  const variantHumans = useGameStore((s) => s.variantHumans);
  const pointWorkersByPoint = useGameStore((s) => s.pointWorkers);
  const pointWorkers = pointWorkersByPoint[point.id] ?? [];
  const fetchPointWorkers = useGameStore((s) => s.fetchPointWorkers);
  const assignWorkerToPoint = useGameStore((s) => s.assignWorkerToPoint);
  const removeWorkerFromPoint = useGameStore((s) => s.removeWorkerFromPoint);

  const [variantHumanID, setVariantHumanID] = useState("");
  const [population, setPopulation] = useState(100);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    void fetchPointWorkers(point.id);
  }, [point.id, fetchPointWorkers]);

  useEffect(() => {
    if (!variantHumanID && variantHumans.length > 0) {
      const preferred = variantHumans.find((v) => v.race === point.worker_race) || variantHumans[0];
      setVariantHumanID(preferred.id);
    }
  }, [variantHumans, variantHumanID, point.worker_race]);

  const assignedHere = pointWorkers
    .filter((w) => w.variant_human_id === variantHumanID)
    .reduce((sum, w) => sum + w.assigned_population, 0);

  const selectedHuman = variantHumans.find((v) => v.id === variantHumanID);

  return (
    <div className="mt-3 space-y-2 rounded-lg border border-zinc-800 bg-zinc-950/60 p-2.5">
      <div className="flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-wide text-zinc-400">
        <Users className="h-3.5 w-3.5 text-cyan-400" /> Phân công lao động
      </div>
      {variantHumans.length === 0 ? (
        <div className="text-[10px] text-zinc-500">Tiên Khiếu chưa có dị nhân nào để phân công.</div>
      ) : (
        <>
          <div className="flex flex-wrap items-center gap-1.5">
            <select
              className="min-w-0 flex-1 rounded border border-zinc-700 bg-zinc-950 px-2 py-1 text-[10px] text-zinc-200 outline-none focus:border-amber-500"
              value={variantHumanID}
              onChange={(e) => setVariantHumanID(e.target.value)}
            >
              {variantHumans.map((human) => (
                <option key={human.id} value={human.id}>
                  {human.race} · dân số {formatNumber(human.population, 0)}
                </option>
              ))}
            </select>
            <input
              type="number"
              min={0}
              max={selectedHuman?.population ?? 0}
              value={population}
              onChange={(e) => setPopulation(Math.max(0, Number(e.target.value) || 0))}
              className="w-20 rounded border border-zinc-700 bg-zinc-950 px-2 py-1 text-[10px] text-zinc-200 outline-none focus:border-amber-500"
            />
            <GameButton
              size="xs"
              variant="secondary"
              loading={busy}
              onClick={async () => {
                if (!variantHumanID) return;
                setBusy(true);
                await assignWorkerToPoint(point.id, variantHumanID, population);
                setBusy(false);
              }}
            >
              Giao việc
            </GameButton>
          </div>
          <div className="text-[9px] text-zinc-500">
            Điểm này đang nhận {formatNumber(assignedHere, 0)} / {formatNumber(selectedHuman?.population ?? 0, 0)} dân ·
            mỗi 500 lao động +25% sản lượng (tối đa +25%).
          </div>
          {pointWorkers.length > 0 && (
            <div className="space-y-1">
              {pointWorkers.map((worker) => (
                <div key={worker.id} className="flex items-center justify-between rounded bg-zinc-900/70 px-2 py-1 text-[10px]">
                  <span className="text-zinc-300">
                    {worker.race} · {formatNumber(worker.assigned_population, 0)} dân
                  </span>
                  <button
                    className="text-rose-400 hover:text-rose-300"
                    onClick={async () => {
                      setBusy(true);
                      await removeWorkerFromPoint(point.id, worker.id);
                      setBusy(false);
                    }}
                  >
                    Bỏ
                  </button>
                </div>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
}

interface PointCardProps {
  point: ApertureResourcePoint;
  itemDictionary: Record<string, { name?: string }>;
  immortalStones: number;
  actionLoading: boolean;
}

function PointCard({ point, itemDictionary, immortalStones, actionLoading }: PointCardProps) {
  const collectAperturePoint = useGameStore((s) => s.collectAperturePoint);
  const revitalizeAperturePoint = useGameStore((s) => s.revitalizeAperturePoint);
  const upgradeAperturePoint = useGameStore((s) => s.upgradeAperturePoint);
  const [showWorkers, setShowWorkers] = useState(false);

  const totalStock = pointStockTotal(point);
  const ready = pointIsReady(point);
  const stockPercent = point.stock_cap > 0 ? Math.min(100, (totalStock / point.stock_cap) * 100) : 0;
  const upgradeCost = pointUpgradeCost(point.level);
  const canUpgrade = point.level < 5 && immortalStones >= upgradeCost;
  const preview = point.yield_preview;

  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-950/50 p-3 transition hover:border-zinc-700">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <div className="flex items-center gap-1.5">
            <span className="text-base leading-none">{pointGlyph(point)}</span>
            <span className="truncate text-xs font-bold text-zinc-100">{point.name}</span>
            {point.point_class === "FAMOUS" && (
              <span className="rounded-full border border-amber-500/40 bg-amber-500/10 px-1.5 py-0.5 text-[9px] font-bold text-amber-300">
                Núi Danh
              </span>
            )}
          </div>
          <div className="mt-1 flex flex-wrap items-center gap-1 text-[9px] text-zinc-500">
            <span className={`rounded border px-1.5 py-0.5 font-semibold ${gradeBadgeClass(point.grade)}`}>
              {point.grade}
            </span>
            <span className="rounded border border-zinc-800 px-1.5 py-0.5">Cấp {point.level}</span>
            <span className="rounded border border-zinc-800 px-1.5 py-0.5">{translatePath(point.path)}</span>
            <span className="rounded border border-zinc-800 px-1.5 py-0.5">
              ({point.x}, {point.y})
            </span>
            {point.state === "EXHAUSTED" && (
              <span className="rounded border border-rose-500/40 bg-rose-500/10 px-1.5 py-0.5 font-semibold text-rose-300">
                Cạn kiệt
              </span>
            )}
          </div>
        </div>
        <div className="shrink-0 text-right">
          <div className="text-[10px] font-bold text-emerald-400">
            {formatNumber(totalStock, 0)} / {formatNumber(point.stock_cap, 0)}
          </div>
          <div className="text-[9px] text-zinc-500">tồn kho</div>
        </div>
      </div>

      <div className="mt-2 space-y-1.5">
        <GameProgressBar value={totalStock} max={Math.max(1, point.stock_cap)} size="xs" variant={ready ? "success" : "default"} />
        <div className="flex items-center gap-2">
          <GameProgressBar
            value={point.vitality}
            max={100}
            size="xs"
            variant={point.vitality < 30 ? "danger" : point.vitality < 50 ? "warning" : "cultivation"}
            className="flex-1"
          />
          <span className="w-20 text-right text-[9px] text-zinc-400">Sinh khí {point.vitality}%</span>
        </div>
      </div>

      <div className="mt-2 space-y-0.5 text-[10px]">
        {Object.entries(point.stock || {}).filter(([, amount]) => amount > 0).length === 0 ? (
          <div className="text-zinc-500">Chưa tích lũy tài nguyên.</div>
        ) : (
          Object.entries(point.stock || {})
            .filter(([, amount]) => amount > 0)
            .map(([itemID, amount]) => (
              <div key={itemID} className="flex justify-between text-zinc-300">
                <span className="truncate">{stockLabel(itemID, itemDictionary)}</span>
                <span className="font-semibold text-emerald-300">×{formatNumber(amount, 0)}</span>
              </div>
            ))
        )}
      </div>

      {preview && (
        <details className="mt-2 rounded border border-zinc-800 bg-zinc-900/40 px-2 py-1">
          <summary className="cursor-pointer list-none text-[10px] font-semibold text-zinc-400">
            <TrendingUp className="mr-1 inline h-3 w-3 text-emerald-400" />
            Sản lượng {formatNumber(preview.expected, 1)}/{point.production_interval_hours || 12} giờ nội bộ · ~
            {formatNumber(preview.per_hour, 1)}/giờ thật
          </summary>
          <div className="mt-1.5 space-y-1 text-[9px]">
            {preview.factors.map((factor) => (
              <div key={factor.key} className="flex justify-between text-zinc-400">
                <span>{YIELD_FACTOR_LABELS[factor.key] || factor.key}</span>
                <span className={factor.value < 0.999 ? "text-rose-400" : "text-emerald-400"}>
                  ×{factor.value.toFixed(2)}
                </span>
              </div>
            ))}
            <div className="flex justify-between border-t border-zinc-800 pt-1 text-zinc-400">
              <span>Trần hệ số</span>
              <span className={preview.capped ? "text-amber-400" : "text-zinc-300"}>
                ×{preview.bonus.toFixed(2)} / ×50{preview.capped ? " (chạm trần)" : ""}
              </span>
            </div>
          </div>
        </details>
      )}

      <div className="mt-2 grid grid-cols-2 gap-1.5">
        <GameButton
          size="xs"
          variant="success"
          loading={actionLoading}
          disabled={!ready}
          onClick={() => void collectAperturePoint(point.id)}
        >
          Thu hoạch
        </GameButton>
        <GameButton
          size="xs"
          variant="secondary"
          loading={actionLoading}
          disabled={point.vitality >= 100}
          onClick={() => void revitalizeAperturePoint(point.id)}
        >
          Bồi dưỡng
        </GameButton>
        <GameButton
          size="xs"
          variant="secondary"
          loading={actionLoading}
          disabled={!canUpgrade}
          onClick={() => void upgradeAperturePoint(point.id)}
        >
          Nâng cấp · {formatNumber(upgradeCost, 0)} đá
        </GameButton>
        <GameButton size="xs" variant="ghost" onClick={() => setShowWorkers((open) => !open)}>
          <Users className="mr-1 h-3 w-3" /> Lao động
        </GameButton>
      </div>

      {showWorkers && <PointWorkerEditor point={point} />}
    </div>
  );
}

export function AperturePointsPanel() {
  const aperturePoints = useGameStore((s) => s.aperturePoints);
  const itemDictionary = useGameStore((s) => s.itemDictionary);
  const player = useGameStore((s) => s.player);
  const actionLoading = useGameStore((s) => s.actionLoading);
  const fetchAperturePoints = useGameStore((s) => s.fetchAperturePoints);
  const fetchVariantHumans = useGameStore((s) => s.fetchVariantHumans);
  const collectAllAperturePoints = useGameStore((s) => s.collectAllAperturePoints);

  useEffect(() => {
    void fetchAperturePoints();
    void fetchVariantHumans();
  }, [fetchAperturePoints, fetchVariantHumans]);

  const sortedPoints = useMemo(() => {
    const rank = (point: ApertureResourcePoint) => {
      if (pointIsReady(point)) return 0;
      if (point.state === "EXHAUSTED") return 2;
      return 1;
    };
    return [...aperturePoints].sort((a, b) => rank(a) - rank(b) || a.name.localeCompare(b.name));
  }, [aperturePoints]);

  const readyCount = sortedPoints.filter(pointIsReady).length;
  const totalStock = sortedPoints.reduce((sum, point) => sum + pointStockTotal(point), 0);

  return (
    <GamePanel className="p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <div className="flex items-center gap-2 text-sm font-bold text-amber-300">
            <Sprout className="h-4 w-4" /> Điểm Tài Nguyên
          </div>
          <div className="mt-0.5 text-[10px] text-zinc-400">
            {sortedPoints.length} điểm · {readyCount} sẵn sàng thu hoạch · tồn kho {formatNumber(totalStock, 0)} món
          </div>
        </div>
        <div className="flex items-center gap-1.5">
          <GameButton size="xs" variant="ghost" onClick={() => void fetchAperturePoints(true)}>
            <RefreshCw className="h-3 w-3" />
          </GameButton>
          <GameButton
            size="xs"
            variant="primary"
            loading={actionLoading}
            disabled={readyCount === 0}
            onClick={() => void collectAllAperturePoints()}
          >
            <Gem className="mr-1 h-3 w-3" /> Thu hoạch tất cả ({readyCount})
          </GameButton>
        </div>
      </div>

      {sortedPoints.length === 0 ? (
        <div className="py-8 text-center text-xs text-zinc-500">
          Tiên Khiếu chưa có điểm tài nguyên. Mỗi lần đột phá độ kiếp sẽ sinh thêm một điểm mới.
        </div>
      ) : (
        <div className="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2 xl:grid-cols-3">
          {sortedPoints.map((point) => (
            <PointCard
              key={point.id}
              point={point}
              itemDictionary={itemDictionary}
              immortalStones={player?.immortal_stones ?? 0}
              actionLoading={actionLoading}
            />
          ))}
        </div>
      )}
    </GamePanel>
  );
}
