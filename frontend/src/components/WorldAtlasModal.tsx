"use client";

import { useEffect, useMemo, useState } from "react";
import { ArrowLeft, Compass, MapPin, X } from "lucide-react";
import type { Region } from "@/store/useGameStore";

interface WorldAtlasModalProps {
  regions: Region[];
  currentRegionId: string;
  x: number;
  y: number;
  onClose: () => void;
}

const WORLD_SIZE = 30_000;
const REGION_COLORS = [
  "#0f766e", "#155e75", "#365314", "#854d0e", "#6b21a8", "#9f1239", "#1d4ed8",
];

export function WorldAtlasModal({ regions, currentRegionId, x, y, onClose }: WorldAtlasModalProps) {
  const majorRegions = useMemo(
    () => regions.filter((region) => region.zone_type === "NGU_VUC" && region.is_active),
    [regions],
  );
  const currentRegion = regions.find((region) => region.id === currentRegionId);
  const currentMajorRegion = majorRegions.find((region) =>
    region.id === currentRegion?.parent_id
    || (x >= (region.min_x ?? Infinity) && x <= (region.max_x ?? -Infinity)
      && y >= (region.min_y ?? Infinity) && y <= (region.max_y ?? -Infinity)),
  );
  const [selectedRegionId, setSelectedRegionId] = useState(currentMajorRegion?.id ?? majorRegions[0]?.id ?? "");
  const [zoomedRegionId, setZoomedRegionId] = useState<string | null>(null);
  const selectedRegion = majorRegions.find((region) => region.id === selectedRegionId) ?? currentMajorRegion;
  const subRegions = regions
    .filter((region) => region.parent_id === selectedRegion?.id && region.is_active)
    .sort((a, b) => (a.min_y ?? 0) - (b.min_y ?? 0) || (a.min_x ?? 0) - (b.min_x ?? 0));
  const mapRegion = majorRegions.find((region) => region.id === zoomedRegionId);
  const mapSubRegions = mapRegion
    ? regions.filter((region) => region.parent_id === mapRegion.id && region.zone_type === "KHU_VUC" && region.is_active
      && region.min_x !== undefined && region.min_y !== undefined && region.max_x !== undefined && region.max_y !== undefined)
    : [];
  const mapHubs = mapRegion
    ? regions.filter((region) => region.parent_id === mapRegion.id && region.zone_type === "HUB" && region.is_active)
    : [];
  const mapViewBox = mapRegion
    ? `${mapRegion.min_x ?? 0} ${mapRegion.min_y ?? 0} ${Math.max(1, (mapRegion.max_x ?? 0) - (mapRegion.min_x ?? 0) + 1)} ${Math.max(1, (mapRegion.max_y ?? 0) - (mapRegion.min_y ?? 0) + 1)}`
    : `0 0 ${WORLD_SIZE} ${WORLD_SIZE}`;
  const hasPlayerPosition = x >= 0 && x <= WORLD_SIZE && y >= 0 && y <= WORLD_SIZE;
  const playerInZoomedRegion = !!mapRegion
    && x >= (mapRegion.min_x ?? Infinity) && x <= (mapRegion.max_x ?? -Infinity)
    && y >= (mapRegion.min_y ?? Infinity) && y <= (mapRegion.max_y ?? -Infinity);

  useEffect(() => {
    if (currentMajorRegion) setSelectedRegionId(currentMajorRegion.id);
  }, [currentMajorRegion?.id]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [onClose]);

  return (
    <div
      className="fixed inset-0 z-[95] flex items-center justify-center bg-black/75 p-3 backdrop-blur-sm sm:p-6"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <section
        role="dialog"
        aria-modal="true"
        aria-labelledby="world-atlas-title"
        className="flex max-h-[94dvh] w-full max-w-5xl flex-col overflow-hidden rounded-2xl border border-cyan-900/60 bg-[#07121a] text-zinc-100 shadow-2xl"
      >
        <header className="flex items-center justify-between border-b border-white/10 px-4 py-3 sm:px-5">
          <div className="flex items-center gap-3">
            <div className="grid h-9 w-9 place-items-center rounded-xl border border-cyan-400/20 bg-cyan-400/10 text-cyan-300">
              <Compass className="h-5 w-5" />
            </div>
            <div>
              <h2 id="world-atlas-title" className="font-serif text-base font-bold sm:text-lg">Bản đồ thế giới</h2>
              <p className="text-[11px] text-zinc-500">Tổng quan các đại vực và vị trí hiện tại</p>
            </div>
          </div>
          <button onClick={onClose} aria-label="Đóng bản đồ" className="grid h-9 w-9 place-items-center rounded-lg text-zinc-400 transition hover:bg-white/10 hover:text-white">
            <X className="h-4 w-4" />
          </button>
        </header>

        <div className="grid min-h-0 flex-1 overflow-y-auto lg:grid-cols-[minmax(0,1.35fr)_minmax(280px,0.85fr)]">
          <div className="border-b border-white/10 p-3 sm:p-5 lg:border-b-0 lg:border-r">
            <div className="relative mx-auto w-full max-w-[620px] overflow-hidden rounded-2xl border border-cyan-950 bg-[#081b22]">
              <div className="pointer-events-none absolute inset-0 opacity-30" style={{ backgroundImage: "radial-gradient(circle at 50% 50%, rgba(34,211,238,.12), transparent 65%), linear-gradient(rgba(34,211,238,.08) 1px, transparent 1px), linear-gradient(90deg, rgba(34,211,238,.08) 1px, transparent 1px)", backgroundSize: "auto, 36px 36px, 36px 36px" }} />
              {mapRegion && (
                <div className="absolute left-2 right-2 top-2 z-10 flex items-center justify-between gap-2 sm:left-3 sm:right-3 sm:top-3">
                  <div className="rounded-lg border border-white/15 bg-[#07121a]/90 px-2.5 py-1.5 text-[10px] font-semibold text-cyan-100 shadow-lg sm:text-xs">
                    {mapRegion.name}
                  </div>
                  <button
                    onClick={() => setZoomedRegionId(null)}
                    className="inline-flex items-center gap-1.5 rounded-lg border border-cyan-300/20 bg-[#07121a]/90 px-2.5 py-1.5 text-[10px] font-medium text-cyan-100 shadow-lg transition hover:bg-cyan-950 sm:text-xs"
                  >
                    <ArrowLeft className="h-3.5 w-3.5" /> Toàn cảnh
                  </button>
                </div>
              )}
              <svg viewBox={mapViewBox} className="relative block aspect-square w-full" aria-label={mapRegion ? `Bản đồ ${mapRegion.name}` : "Bản đồ các đại vực"}>
                {majorRegions.filter((region) => !mapRegion || region.id === mapRegion.id).map((region, index) => {
                  const minX = region.min_x ?? 0;
                  const minY = region.min_y ?? 0;
                  const width = (region.max_x ?? minX) - minX + 1;
                  const height = (region.max_y ?? minY) - minY + 1;
                  const isSelected = selectedRegion?.id === region.id;
                  const isCurrent = currentMajorRegion?.id === region.id;
                  const label = region.name.replace(/^Đại Vực\s*/i, "");
                  return (
                    <g
                      key={region.id}
                      role="button"
                      tabIndex={0}
                      aria-label={`Chọn ${region.name}`}
                      onClick={() => {
                        setSelectedRegionId(region.id);
                        setZoomedRegionId(region.id);
                      }}
                      onKeyDown={(event) => {
                        if (event.key === "Enter" || event.key === " ") {
                          setSelectedRegionId(region.id);
                          setZoomedRegionId(region.id);
                        }
                      }}
                      className="cursor-pointer"
                    >
                      <rect
                        x={minX + 90}
                        y={minY + 90}
                        width={Math.max(0, width - 180)}
                        height={Math.max(0, height - 180)}
                        rx="480"
                        fill={REGION_COLORS[index % REGION_COLORS.length]}
                        fillOpacity={isSelected ? 0.68 : 0.42}
                        stroke={isSelected ? "#67e8f9" : isCurrent ? "#fbbf24" : "#67e8f9"}
                        strokeOpacity={isSelected || isCurrent ? 0.95 : 0.35}
                        strokeWidth={isSelected ? 110 : 55}
                      />
                      {!mapRegion && <text
                        x={minX + width / 2}
                        y={minY + height / 2}
                        textAnchor="middle"
                        dominantBaseline="middle"
                        fill="#ecfeff"
                        fontSize="510"
                        fontWeight="700"
                        className="pointer-events-none"
                      >
                        {label}
                      </text>}
                      {!mapRegion && isCurrent && (
                        <text x={minX + width / 2} y={minY + height / 2 + 650} textAnchor="middle" fill="#fcd34d" fontSize="300" fontWeight="700" className="pointer-events-none">
                          ĐẠO HỮU ĐANG Ở ĐÂY
                        </text>
                      )}
                    </g>
                  );
                })}
                {mapRegion && mapSubRegions.map((region, index) => {
                  const minX = region.min_x ?? 0;
                  const minY = region.min_y ?? 0;
                  const width = (region.max_x ?? minX) - minX + 1;
                  const height = (region.max_y ?? minY) - minY + 1;
                  const centerX = minX + width / 2;
                  const centerY = minY + height / 2;
                  const words = region.name.split(/\s+/);
                  const firstLine = words.length > 2 ? words.slice(0, Math.ceil(words.length / 2)).join(" ") : region.name;
                  const secondLine = words.length > 2 ? words.slice(Math.ceil(words.length / 2)).join(" ") : "";
                  const isCurrent = currentRegion?.id === region.id;
                  return (
                    <g key={region.id} className="pointer-events-none">
                      <title>{region.name}</title>
                      <rect
                        x={minX + 55}
                        y={minY + 55}
                        width={Math.max(1, width - 110)}
                        height={Math.max(1, height - 110)}
                        rx="110"
                        fill={REGION_COLORS[index % REGION_COLORS.length]}
                        fillOpacity="0.52"
                        stroke={isCurrent ? "#fda4af" : "#a5f3fc"}
                        strokeOpacity={isCurrent ? 1 : 0.7}
                        strokeWidth={isCurrent ? 45 : 24}
                      />
                      <text x={centerX} y={secondLine ? centerY - 100 : centerY} textAnchor="middle" dominantBaseline="middle" fill={isCurrent ? "#fff1f2" : "#ecfeff"} fontSize="270" fontWeight="600">
                        <tspan x={centerX}>{firstLine}</tspan>
                        {secondLine && <tspan x={centerX} dy="1.2em">{secondLine}</tspan>}
                      </text>
                    </g>
                  );
                })}
                {mapHubs.map((hub) => {
                  const hubX = hub.entry_x ?? ((hub.min_x ?? 0) + (hub.max_x ?? 0)) / 2;
                  const hubY = hub.entry_y ?? ((hub.min_y ?? 0) + (hub.max_y ?? 0)) / 2;
                  return (
                    <g key={hub.id} className="pointer-events-none">
                      <title>{hub.name}</title>
                      <circle cx={hubX} cy={hubY} r="260" fill="#fbbf24" stroke="#fffbeb" strokeWidth="55" />
                      <text x={hubX} y={hubY + 520} textAnchor="middle" fill="#fef3c7" fontSize="250" fontWeight="700">{hub.name}</text>
                    </g>
                  );
                })}
                {hasPlayerPosition && (!mapRegion || playerInZoomedRegion) && (
                  <g className="pointer-events-none">
                    <circle cx={x} cy={y} r="560" fill="#f43f5e" fillOpacity="0.2" />
                    <circle cx={x} cy={y} r="230" fill="#fb7185" stroke="#fff1f2" strokeWidth="90" />
                    <path d={`M ${x} ${y - 800} L ${x - 180} ${y - 150} L ${x + 180} ${y - 150} Z`} fill="#fff1f2" />
                  </g>
                )}
              </svg>
              <div className="absolute bottom-2 left-2 rounded-md border border-white/10 bg-black/60 px-2 py-1 text-[9px] font-mono text-zinc-400 sm:bottom-3 sm:left-3 sm:text-[10px]">
                BẮC ↑ <span className="mx-1 text-zinc-600">•</span> {mapRegion ? "Các khu vực trong đại vực" : "Chọn đại vực để xem khu vực"}
              </div>
            </div>
          </div>

          <aside className="flex min-h-0 flex-col gap-4 p-4 sm:p-5">
            <section className="rounded-xl border border-rose-400/20 bg-rose-400/[0.06] p-3">
              <div className="mb-1 flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-rose-300">
                <MapPin className="h-3.5 w-3.5" /> Vị trí hiện tại
              </div>
              <div className="font-serif text-sm font-semibold text-zinc-100">{currentRegion?.name ?? currentRegionId ?? "Chưa xác định"}</div>
              <div className="mt-1 font-mono text-[10px] text-zinc-400">X: {Math.round(x).toLocaleString("vi-VN")} · Y: {Math.round(y).toLocaleString("vi-VN")}</div>
              {currentMajorRegion && <div className="mt-1 text-[10px] text-zinc-500">{currentMajorRegion.name}</div>}
            </section>

            <section className="flex min-h-0 flex-1 flex-col">
              <div className="mb-2 flex items-center justify-between gap-2">
                <div>
                  <h3 className="font-serif text-sm font-semibold text-cyan-100">{selectedRegion?.name ?? "Các đại vực"}</h3>
                  <p className="text-[10px] text-zinc-500">{subRegions.length} khu vực đã biết</p>
                </div>
                {selectedRegion && <span className="shrink-0 rounded-md border border-cyan-500/20 bg-cyan-500/10 px-2 py-1 text-[9px] text-cyan-200">C{selectedRegion.min_rank}–C{selectedRegion.max_rank}</span>}
              </div>
              {selectedRegion?.description && <p className="mb-3 line-clamp-3 text-[11px] leading-relaxed text-zinc-400">{selectedRegion.description}</p>}
              <div className="min-h-0 space-y-1.5 overflow-y-auto pr-1">
                {subRegions.map((region) => {
                  const isCurrent = currentRegion?.id === region.id;
                  return (
                    <div key={region.id} className={`rounded-lg border px-3 py-2 ${isCurrent ? "border-rose-400/30 bg-rose-400/[0.07]" : "border-white/[0.07] bg-white/[0.025]"}`}>
                      <div className="flex items-center justify-between gap-2">
                        <span className={`truncate text-xs font-medium ${isCurrent ? "text-rose-200" : "text-zinc-200"}`}>{region.name}</span>
                        {isCurrent ? <span className="shrink-0 text-[9px] font-semibold text-rose-300">ĐANG Ở ĐÂY</span> : <span className="shrink-0 text-[9px] text-zinc-500">C{region.min_rank}–C{region.max_rank}</span>}
                      </div>
                    </div>
                  );
                })}
                {subRegions.length === 0 && <div className="rounded-lg border border-dashed border-white/10 p-3 text-xs text-zinc-500">Chưa có khu vực con.</div>}
              </div>
            </section>

            <div className="flex items-center gap-2 border-t border-white/10 pt-3 text-[10px] text-zinc-500">
              <span className={`inline-block h-2 w-2 rounded-full ${mapRegion && !playerInZoomedRegion ? "bg-zinc-600" : "bg-rose-400"}`} />
              {mapRegion && !playerInZoomedRegion ? "Bạn đang ở đại vực khác" : "Vị trí của bạn"}
              <span className="ml-auto">{mapRegion ? `${mapSubRegions.length} khu vực trên bản đồ` : `${majorRegions.length} đại vực`}</span>
            </div>
          </aside>
        </div>
      </section>
    </div>
  );
}
