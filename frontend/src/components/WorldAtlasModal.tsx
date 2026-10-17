"use client";

import { useEffect, useMemo, useState, useCallback } from "react";
import { createPortal } from "react-dom";
import { ArrowLeft, Compass, MapPin, X, Search, Shield, Skull, Crown, Sparkles, Bug, Package, Layers } from "lucide-react";
import type { Region } from "@/store/useGameStore";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";

interface WorldAtlasModalProps {
  regions: Region[];
  currentRegionId: string;
  x: number;
  y: number;
  onClose: () => void;
}

export interface RegionMonsterDetail {
  id: string;
  name: string;
  type: string;
  min_rank: number;
  max_rank: number;
  is_boss: boolean;
  is_capturable: boolean;
  max_hp: number;
  base_attack: number;
  base_defense: number;
  max_hp_max: number;
  attack_max: number;
  defense_max: number;
  description: string;
}

export interface RegionDropDetail {
  id: string;
  name: string;
  kind: "item" | "gu" | "resource";
  rank?: number;
  chance: number;
  min_qty: number;
  max_qty: number;
  sources?: string[];
}

export interface NamedResource {
  id: string;
  name: string;
}

export interface RegionDetailsData {
  region_id: string;
  name: string;
  zone_type: string;
  min_rank: number;
  max_rank: number;
  description: string;
  monsters: RegionMonsterDetail[];
  drops: RegionDropDetail[];
  wild_gus: NamedResource[];
  resources: NamedResource[];
  has_boss: boolean;
  monster_count: number;
}

const WORLD_SIZE = 30_000;
const REGION_COLORS = [
  "#0f766e", "#155e75", "#365314", "#854d0e", "#6b21a8", "#9f1239", "#1d4ed8",
];

// Rank-scaled monster stats get very large (rank 5 = x100, rank 6 = x10^4,
// rank 7 = x10^6 ...). Format with thousands separators, and collapse to a
// single value when the min/max ranks produce the same number.
function formatMonsterStat(min: number, max: number): string {
  const lo = Math.round(min);
  const hi = Math.round(max);
  if (!Number.isFinite(lo) || lo <= 0) return "—";
  if (!Number.isFinite(hi) || hi <= lo) return lo.toLocaleString("vi-VN");
  return `${lo.toLocaleString("vi-VN")} – ${hi.toLocaleString("vi-VN")}`;
}

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

  const [selectedMajorRegionId, setSelectedMajorRegionId] = useState(currentMajorRegion?.id ?? majorRegions[0]?.id ?? "");
  const [zoomedRegionId, setZoomedRegionId] = useState<string | null>(null);
  const [inspectedSubRegionId, setInspectedSubRegionId] = useState<string>(currentRegionId);
  const [searchQuery, setSearchQuery] = useState("");
  const [activeTab, setActiveTab] = useState<"monsters" | "drops" | "list">("monsters");

  // Region details cache & loading state
  const [detailsCache, setDetailsCache] = useState<Record<string, RegionDetailsData>>({});
  const [loadingDetails, setLoadingDetails] = useState(false);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  const selectedMajorRegion = majorRegions.find((region) => region.id === selectedMajorRegionId) ?? currentMajorRegion;
  const mapRegion = majorRegions.find((region) => region.id === zoomedRegionId);

  const subRegions = useMemo(() => {
    return regions
      .filter((region) => region.parent_id === (mapRegion?.id ?? selectedMajorRegion?.id) && region.is_active)
      .sort((a, b) => (a.min_y ?? 0) - (b.min_y ?? 0) || (a.min_x ?? 0) - (b.min_x ?? 0));
  }, [regions, mapRegion?.id, selectedMajorRegion?.id]);

  const mapSubRegions = useMemo(() => {
    return mapRegion
      ? regions.filter((region) => region.parent_id === mapRegion.id && region.zone_type !== "HUB" && region.is_active
        && region.min_x !== undefined && region.min_y !== undefined && region.max_x !== undefined && region.max_y !== undefined)
      : [];
  }, [regions, mapRegion]);

  const mapHubs = useMemo(() => {
    return mapRegion
      ? regions.filter((region) => region.parent_id === mapRegion.id && (region.zone_type === "HUB" || region.zone_type === "SAFE_ZONE") && region.is_active)
      : [];
  }, [regions, mapRegion]);

  const mapViewBox = mapRegion
    ? `${mapRegion.min_x ?? 0} ${mapRegion.min_y ?? 0} ${Math.max(1, (mapRegion.max_x ?? 0) - (mapRegion.min_x ?? 0) + 1)} ${Math.max(1, (mapRegion.max_y ?? 0) - (mapRegion.min_y ?? 0) + 1)}`
    : `0 0 ${WORLD_SIZE} ${WORLD_SIZE}`;

  const hasPlayerPosition = x >= 0 && x <= WORLD_SIZE && y >= 0 && y <= WORLD_SIZE;
  const playerInZoomedRegion = !!mapRegion
    && x >= (mapRegion.min_x ?? Infinity) && x <= (mapRegion.max_x ?? -Infinity)
    && y >= (mapRegion.min_y ?? Infinity) && y <= (mapRegion.max_y ?? -Infinity);

  const inspectedRegion = useMemo(() => {
    return regions.find((r) => r.id === inspectedSubRegionId) || currentRegion || subRegions[0];
  }, [inspectedSubRegionId, regions, currentRegion, subRegions]);

  // Fetch region details
  const fetchDetails = useCallback(async (regionId: string) => {
    if (!regionId || detailsCache[regionId]) return;
    setLoadingDetails(true);
    try {
      const res = await authFetch(`${BACKEND_URL}/api/explore/regions/${regionId}/details`);
      if (res.ok) {
        const data: RegionDetailsData = await res.json();
        setDetailsCache((prev) => ({ ...prev, [regionId]: data }));
      }
    } catch (err) {
      console.warn("fetch region details error:", err);
    } finally {
      setLoadingDetails(false);
    }
  }, [detailsCache]);

  useEffect(() => {
    if (inspectedRegion?.id) {
      void fetchDetails(inspectedRegion.id);
    }
  }, [inspectedRegion?.id, fetchDetails]);

  useEffect(() => {
    if (currentMajorRegion) {
      setSelectedMajorRegionId(currentMajorRegion.id);
    }
  }, [currentMajorRegion?.id]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [onClose]);

  const currentDetails = inspectedRegion ? detailsCache[inspectedRegion.id] : null;

  // Search filter
  const cleanSearch = searchQuery.trim().toLowerCase();
  const isRegionMatch = useCallback((r: Region) => {
    if (!cleanSearch) return true;
    if (r.name.toLowerCase().includes(cleanSearch) || r.id.toLowerCase().includes(cleanSearch)) {
      return true;
    }
    const cached = detailsCache[r.id];
    if (cached) {
      if (cached.monsters?.some((m) => m.name.toLowerCase().includes(cleanSearch))) return true;
      if (cached.drops?.some((d) => d.name.toLowerCase().includes(cleanSearch))) return true;
      if (cached.wild_gus?.some((g) => g.name.toLowerCase().includes(cleanSearch))) return true;
      if (cached.resources?.some((res) => res.name.toLowerCase().includes(cleanSearch))) return true;
    }
    return false;
  }, [cleanSearch, detailsCache]);

  if (!mounted) return null;

  return createPortal(
    <div
      className="fixed inset-0 z-[100] h-[100dvh] flex items-center justify-center bg-black/80 px-2 sm:px-4 md:px-6 pt-[calc(env(safe-area-inset-top)+0.5rem)] pb-[calc(env(safe-area-inset-bottom)+0.5rem)] backdrop-blur-md"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <section
        role="dialog"
        aria-modal="true"
        aria-labelledby="world-atlas-title"
        className="flex max-h-[calc(100dvh-env(safe-area-inset-top)-env(safe-area-inset-bottom)-1rem)] w-full max-w-6xl flex-col overflow-hidden rounded-2xl border border-cyan-800/50 bg-[#061017] text-zinc-100 shadow-2xl shadow-cyan-950/40"
      >
        {/* Header */}
        <header className="flex flex-wrap items-center justify-between gap-3 border-b border-white/10 px-4 py-3 sm:px-5">
          <div className="flex items-center gap-3">
            <div className="grid h-9 w-9 place-items-center rounded-xl border border-cyan-400/30 bg-cyan-500/10 text-cyan-300 shadow-inner">
              <Compass className="h-5 w-5" />
            </div>
            <div>
              <h2 id="world-atlas-title" className="font-serif text-base font-bold text-cyan-50 sm:text-lg">
                Bản Đồ Cổ Giới
              </h2>
              <p className="text-[11px] text-zinc-400">
                {mapRegion ? `Chi tiết: ${mapRegion.name}` : "Toàn cảnh 5 Đại Vực & Phân bố Yêu thú, Chiến lợi phẩm"}
              </p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            {/* Search input */}
            <div className="relative">
              <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-zinc-400" />
              <input
                type="text"
                placeholder="Tìm quái, item, Cổ..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="h-8 w-36 sm:w-44 lg:w-56 rounded-lg border border-cyan-900/60 bg-[#091b24] pl-8 pr-7 text-xs text-zinc-100 placeholder:text-zinc-500 focus:border-cyan-400 focus:outline-none"
              />
              {searchQuery && (
                <button
                  onClick={() => setSearchQuery("")}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-200"
                >
                  <X className="h-3 w-3" />
                </button>
              )}
            </div>

            <button
              onClick={onClose}
              aria-label="Đóng bản đồ"
              className="grid h-8 w-8 place-items-center rounded-lg border border-white/10 bg-white/[0.04] text-zinc-400 transition hover:bg-rose-500/20 hover:text-rose-200"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </header>

        {/* Body Grid */}
        <div className="flex flex-col lg:grid lg:grid-cols-[minmax(0,1.25fr)_minmax(330px,0.85fr)] min-h-0 flex-1 overflow-y-auto lg:overflow-hidden custom-scrollbar">
          {/* Left Column: Map Screen */}
          <div className="flex shrink-0 flex-col border-b border-white/10 p-2.5 sm:p-4 lg:border-b-0 lg:border-r lg:min-h-0 lg:shrink">
            {/* Map Top Navigation Bar */}
            <div className="mb-2 flex items-center justify-between gap-2">
              {mapRegion ? (
                <button
                  onClick={() => setZoomedRegionId(null)}
                  className="inline-flex items-center gap-1.5 rounded-lg border border-cyan-400/30 bg-cyan-950/60 px-3 py-1.5 text-xs font-semibold text-cyan-200 shadow transition hover:border-cyan-400 hover:bg-cyan-900/80"
                >
                  <ArrowLeft className="h-3.5 w-3.5" /> Toàn cảnh Đại Vực
                </button>
              ) : (
                <span className="text-xs font-medium text-zinc-400">
                  Nhấp vào một Đại Vực để mở lưới khu vực con
                </span>
              )}

              {mapRegion && (
                <div className="flex items-center gap-2">
                  <span className="text-xs font-semibold text-cyan-300">{mapRegion.name}</span>
                  <span className="rounded bg-cyan-900/50 px-2 py-0.5 text-[10px] font-mono text-cyan-200 border border-cyan-700/40">
                    C{mapRegion.min_rank}–C{mapRegion.max_rank}
                  </span>
                </div>
              )}
            </div>

            {/* SVG Map Container */}
            <div className="relative mx-auto flex w-full max-w-[340px] sm:max-w-[460px] lg:max-w-[560px] flex-1 items-center justify-center overflow-hidden rounded-xl border border-cyan-950/80 bg-[#05131a] shadow-inner">
              <div
                className="pointer-events-none absolute inset-0 opacity-25"
                style={{
                  backgroundImage:
                    "radial-gradient(circle at 50% 50%, rgba(34,211,238,.15), transparent 70%), linear-gradient(rgba(34,211,238,.07) 1px, transparent 1px), linear-gradient(90deg, rgba(34,211,238,.07) 1px, transparent 1px)",
                  backgroundSize: "auto, 32px 32px, 32px 32px",
                }}
              />

              <svg
                viewBox={mapViewBox}
                className="relative block aspect-square w-full"
                aria-label={mapRegion ? `Bản đồ ${mapRegion.name}` : "Bản đồ các đại vực"}
              >
                {/* 1. Large 5 Major Regions view */}
                {!mapRegion &&
                  majorRegions.map((region, index) => {
                    const minX = region.min_x ?? 0;
                    const minY = region.min_y ?? 0;
                    const width = (region.max_x ?? minX) - minX + 1;
                    const height = (region.max_y ?? minY) - minY + 1;
                    const isSelected = selectedMajorRegionId === region.id;
                    const isCurrent = currentMajorRegion?.id === region.id;
                    const label = region.name.replace(/^Đại Vực\s*/i, "");
                    const isMatched = isRegionMatch(region);

                    return (
                      <g
                        key={region.id}
                        role="button"
                        tabIndex={0}
                        aria-label={`Chọn ${region.name}`}
                        onClick={() => {
                          setSelectedMajorRegionId(region.id);
                          setZoomedRegionId(region.id);
                        }}
                        className={`cursor-pointer transition duration-200 hover:brightness-110 ${!isMatched ? "opacity-30" : "opacity-100"}`}
                      >
                        <rect
                          x={minX + 90}
                          y={minY + 90}
                          width={Math.max(0, width - 180)}
                          height={Math.max(0, height - 180)}
                          rx="480"
                          fill={REGION_COLORS[index % REGION_COLORS.length]}
                          fillOpacity={isSelected ? 0.72 : 0.42}
                          stroke={isSelected ? "#38bdf8" : isCurrent ? "#fbbf24" : "#0e7490"}
                          strokeOpacity={isSelected || isCurrent ? 0.95 : 0.4}
                          strokeWidth={isSelected ? 120 : 60}
                        />
                        <text
                          x={minX + width / 2}
                          y={minY + height / 2 - 80}
                          textAnchor="middle"
                          dominantBaseline="middle"
                          fill="#f0fdfa"
                          fontSize="520"
                          fontWeight="700"
                          className="pointer-events-none select-none"
                        >
                          {label}
                        </text>
                        <text
                          x={minX + width / 2}
                          y={minY + height / 2 + 380}
                          textAnchor="middle"
                          fill="#67e8f9"
                          fontSize="260"
                          fontWeight="600"
                          className="pointer-events-none select-none"
                        >
                          C{region.min_rank}–C{region.max_rank}
                        </text>
                        {isCurrent && (
                          <text
                            x={minX + width / 2}
                            y={minY + height / 2 + 750}
                            textAnchor="middle"
                            fill="#fcd34d"
                            fontSize="280"
                            fontWeight="700"
                            className="pointer-events-none select-none"
                          >
                            ĐANG Ở ĐÂY
                          </text>
                        )}
                      </g>
                    );
                  })}

                {/* 2. Sub-regions grid view */}
                {mapRegion &&
                  mapSubRegions.map((region, index) => {
                    const minX = region.min_x ?? 0;
                    const minY = region.min_y ?? 0;
                    const width = (region.max_x ?? minX) - minX + 1;
                    const height = (region.max_y ?? minY) - minY + 1;
                    const centerX = minX + width / 2;
                    const centerY = minY + height / 2;

                    const isInspected = inspectedRegion?.id === region.id;
                    const isCurrent = currentRegion?.id === region.id;
                    const isMatched = isRegionMatch(region);
                    const isSafe = region.zone_type === "SAFE_ZONE" || region.zone_type === "HUB";
                    const hasBoss = region.has_boss || region.zone_type === "BOSS_ZONE" || region.max_rank >= 7;

                    // Formatted labels (split into lines)
                    const words = region.name.split(/\s+/);
                    let firstLine = region.name;
                    let secondLine = "";
                    if (words.length > 3) {
                      firstLine = words.slice(0, Math.ceil(words.length / 2)).join(" ");
                      secondLine = words.slice(Math.ceil(words.length / 2)).join(" ");
                    } else if (words.length === 3 && region.name.length > 12) {
                      firstLine = words.slice(0, 2).join(" ");
                      secondLine = words.slice(2).join(" ");
                    }

                    const fontSize = firstLine.length > 14 ? 175 : 210;

                    return (
                      <g
                        key={region.id}
                        role="button"
                        tabIndex={0}
                        aria-label={`Khu vực ${region.name}`}
                        onClick={() => {
                          setInspectedSubRegionId(region.id);
                        }}
                        className={`cursor-pointer transition duration-150 hover:brightness-125 ${!isMatched ? "opacity-25" : "opacity-100"}`}
                      >
                        <title>{`${region.name} (C${region.min_rank}-C${region.max_rank})`}</title>
                        <rect
                          x={minX + 35}
                          y={minY + 35}
                          width={Math.max(1, width - 70)}
                          height={Math.max(1, height - 70)}
                          rx="90"
                          fill={REGION_COLORS[index % REGION_COLORS.length]}
                          fillOpacity={isInspected ? 0.8 : 0.45}
                          stroke={isInspected ? "#38bdf8" : isCurrent ? "#fb7185" : hasBoss ? "#f59e0b" : "#0e7490"}
                          strokeOpacity={isInspected || isCurrent ? 1 : 0.6}
                          strokeWidth={isInspected ? 65 : isCurrent ? 45 : 24}
                        />

                        {/* Badges on corner */}
                        {hasBoss && (
                          <g transform={`translate(${minX + width - 160}, ${minY + 60}) scale(0.65)`}>
                            <rect width="180" height="180" rx="40" fill="#78350f" fillOpacity="0.8" />
                            <text x="90" y="125" textAnchor="middle" fontSize="110">👑</text>
                          </g>
                        )}
                        {isSafe && (
                          <g transform={`translate(${minX + 60}, ${minY + 60}) scale(0.65)`}>
                            <rect width="180" height="180" rx="40" fill="#065f46" fillOpacity="0.8" />
                            <text x="90" y="125" textAnchor="middle" fontSize="110">🛡️</text>
                          </g>
                        )}

                        {/* Name text */}
                        <text
                          x={centerX}
                          y={secondLine ? centerY - 75 : centerY}
                          textAnchor="middle"
                          dominantBaseline="middle"
                          fill={isInspected ? "#ffffff" : isCurrent ? "#fff1f2" : "#ecfeff"}
                          fontSize={fontSize}
                          fontWeight="700"
                        >
                          <tspan x={centerX}>{firstLine}</tspan>
                          {secondLine && <tspan x={centerX} dy="1.25em">{secondLine}</tspan>}
                        </text>

                        {/* Rank indicator at bottom of cell */}
                        <text
                          x={centerX}
                          y={minY + height - 85}
                          textAnchor="middle"
                          fill={isInspected ? "#7dd3fc" : "#94a3b8"}
                          fontSize="140"
                          fontWeight="600"
                        >
                          C{region.min_rank}–C{region.max_rank}
                        </text>
                      </g>
                    );
                  })}

                {/* 3. Hubs */}
                {mapHubs.map((hub) => {
                  const hubX = hub.entry_x ?? ((hub.min_x ?? 0) + (hub.max_x ?? 0)) / 2;
                  const hubY = hub.entry_y ?? ((hub.min_y ?? 0) + (hub.max_y ?? 0)) / 2;
                  const isInspected = inspectedRegion?.id === hub.id;
                  return (
                    <g
                      key={hub.id}
                      onClick={() => setInspectedSubRegionId(hub.id)}
                      className="cursor-pointer transition hover:scale-105"
                    >
                      <title>{hub.name}</title>
                      <circle cx={hubX} cy={hubY} r="250" fill="#eab308" stroke="#fef08a" strokeWidth={isInspected ? 70 : 40} />
                      <text x={hubX} y={hubY + 500} textAnchor="middle" fill="#fef08a" fontSize="230" fontWeight="700">
                        {hub.name}
                      </text>
                    </g>
                  );
                })}

                {/* 4. Player Pointer */}
                {hasPlayerPosition && (!mapRegion || playerInZoomedRegion) && (
                  <g className="pointer-events-none">
                    <circle cx={x} cy={y} r="520" fill="#f43f5e" fillOpacity="0.25" />
                    <circle cx={x} cy={y} r="210" fill="#f43f5e" stroke="#fff1f2" strokeWidth="80" />
                    <path d={`M ${x} ${y - 750} L ${x - 170} ${y - 140} L ${x + 170} ${y - 140} Z`} fill="#fff1f2" />
                  </g>
                )}
              </svg>

              {/* Map Footer status */}
              <div className="absolute bottom-2 left-2 rounded-md border border-white/10 bg-black/70 px-2.5 py-1 text-[10px] font-mono text-zinc-300">
                BẮC ↑ <span className="mx-1 text-zinc-600">•</span>
                {mapRegion ? "Nhấp ô để xem Quái & Vật phẩm rơi" : "Nhấp Đại Vực để xem chi tiết"}
              </div>
            </div>
          </div>

          {/* Right Column: Detailed Info Panel */}
          <aside className="flex flex-col shrink-0 lg:shrink lg:min-h-0 lg:flex-1 bg-[#07141d]/70 p-3 sm:p-4 pb-12 sm:pb-6 lg:pb-4">
            {/* 1. Current Player Region Badge */}
            <div className="mb-2.5 rounded-xl border border-rose-500/25 bg-rose-500/[0.07] px-3 py-2 shrink-0">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider text-rose-300">
                  <MapPin className="h-3 w-3" /> Vị trí hiện tại
                </div>
                <span className="font-mono text-[10px] text-zinc-400">
                  X: {Math.round(x).toLocaleString("vi-VN")} · Y: {Math.round(y).toLocaleString("vi-VN")}
                </span>
              </div>
              <div className="font-serif text-xs font-semibold text-zinc-100">
                {currentRegion?.name ?? currentRegionId ?? "Chưa xác định"}
                {currentMajorRegion && <span className="ml-1.5 text-[10px] font-normal text-zinc-400">({currentMajorRegion.name})</span>}
              </div>
            </div>

            {/* 2. Inspected Region Banner */}
            <div className="rounded-xl border border-cyan-800/40 bg-cyan-950/20 p-3 shadow shrink-0">
              <div className="flex items-start justify-between gap-2">
                <div>
                  <div className="flex items-center gap-1.5">
                    <h3 className="font-serif text-sm font-bold text-cyan-100">
                      {inspectedRegion?.name ?? "Chưa chọn khu vực"}
                    </h3>
                    {inspectedRegion?.has_boss && (
                      <span className="flex items-center gap-0.5 rounded bg-amber-500/20 px-1.5 py-0.5 text-[9px] font-semibold text-amber-300 border border-amber-500/30">
                        <Crown className="h-2.5 w-2.5" /> Có Boss
                      </span>
                    )}
                  </div>
                  <div className="mt-1 flex items-center gap-2 text-[10px] text-zinc-400">
                    <span className="rounded bg-white/[0.06] px-1.5 py-0.5 font-mono text-cyan-300">
                      Cấp C{inspectedRegion?.min_rank}–C{inspectedRegion?.max_rank}
                    </span>
                    <span>
                      {inspectedRegion?.zone_type === "SAFE_ZONE" || inspectedRegion?.zone_type === "HUB"
                        ? "Khu An Toàn"
                        : inspectedRegion?.zone_type === "BOSS_ZONE"
                        ? "Khu Lĩnh Chủ"
                        : "Khu Vực Dã Ngoại"}
                    </span>
                  </div>
                </div>
              </div>
              {inspectedRegion?.description && (
                <p className="mt-2 line-clamp-2 text-[11px] leading-relaxed text-zinc-400">
                  {inspectedRegion.description}
                </p>
              )}
            </div>

            {/* 3. Navigation Tabs */}
            <div className="mt-2.5 flex border-b border-white/10 text-xs shrink-0 overflow-x-auto">
              <button
                onClick={() => setActiveTab("monsters")}
                className={`flex items-center gap-1.5 border-b-2 px-3 pb-2 font-medium transition ${
                  activeTab === "monsters"
                    ? "border-cyan-400 text-cyan-300"
                    : "border-transparent text-zinc-400 hover:text-zinc-200"
                }`}
              >
                <Skull className="h-3.5 w-3.5" /> Yêu Thú ({currentDetails?.monsters?.length ?? inspectedRegion?.monster_count ?? 0})
              </button>
              <button
                onClick={() => setActiveTab("drops")}
                className={`flex items-center gap-1.5 border-b-2 px-3 pb-2 font-medium transition ${
                  activeTab === "drops"
                    ? "border-amber-400 text-amber-300"
                    : "border-transparent text-zinc-400 hover:text-zinc-200"
                }`}
              >
                <Package className="h-3.5 w-3.5" /> Rơi Đồ & Cổ (
                {(currentDetails?.drops?.length ?? 0) + (currentDetails?.wild_gus?.length ?? 0) + (currentDetails?.resources?.length ?? 0)}
                )
              </button>
              <button
                onClick={() => setActiveTab("list")}
                className={`flex items-center gap-1.5 border-b-2 px-3 pb-2 font-medium transition ${
                  activeTab === "list"
                    ? "border-emerald-400 text-emerald-300"
                    : "border-transparent text-zinc-400 hover:text-zinc-200"
                }`}
              >
                <Layers className="h-3.5 w-3.5" /> Khu Vực ({subRegions.length})
              </button>
            </div>

            {/* 4. Tab Content Area */}
            <div className="mt-2.5 min-h-[160px] max-h-80 lg:max-h-none lg:min-h-0 lg:flex-1 overflow-y-auto pr-1">
              {loadingDetails && !currentDetails ? (
                <div className="flex h-32 items-center justify-center text-xs text-zinc-500">
                  <div className="h-4 w-4 animate-spin rounded-full border-2 border-cyan-400 border-t-transparent mr-2" />
                  Đang tải thông tin khu vực...
                </div>
              ) : activeTab === "monsters" ? (
                <div className="space-y-2">
                  {currentDetails?.monsters && currentDetails.monsters.length > 0 ? (
                    currentDetails.monsters.map((monster) => {
                      const isBoss = monster.is_boss || monster.type === "Boss";
                      const isElite = monster.type === "Elite";

                      return (
                        <div
                          key={monster.id}
                          className={`rounded-lg border p-2.5 transition ${
                            isBoss
                              ? "border-amber-500/30 bg-amber-950/20 shadow-sm"
                              : isElite
                              ? "border-purple-500/25 bg-purple-950/15"
                              : "border-white/[0.08] bg-white/[0.02]"
                          }`}
                        >
                          <div className="flex items-start justify-between gap-2">
                            <div>
                              <div className="flex items-center gap-1.5 font-semibold text-xs text-zinc-100">
                                {isBoss && <Crown className="h-3.5 w-3.5 text-amber-400" />}
                                {isElite && <Sparkles className="h-3.5 w-3.5 text-purple-400" />}
                                <span className={isBoss ? "text-amber-200" : isElite ? "text-purple-200" : "text-zinc-200"}>
                                  {monster.name}
                                </span>
                              </div>
                              <div className="mt-0.5 flex flex-wrap items-center gap-1.5 text-[10px] text-zinc-400">
                                <span className="font-mono text-cyan-300">Rank {monster.max_rank > monster.min_rank ? `${monster.min_rank}–${monster.max_rank}` : monster.min_rank}</span>
                                <span>•</span>
                                <span>{isBoss ? "Thủ Lĩnh / Boss" : isElite ? "Tinh Anh" : "Yêu thú thường"}</span>
                                {monster.is_capturable && (
                                  <>
                                    <span>•</span>
                                    <span className="text-emerald-400 font-medium">Có thể bắt 🐾</span>
                                  </>
                                )}
                              </div>
                            </div>
                            {monster.max_hp > 0 && (
                              <div className="text-right text-[10px] font-mono text-zinc-400">
                                <div>Máu: <span className="text-rose-300">{formatMonsterStat(monster.max_hp, monster.max_hp_max)}</span></div>
                                <div>Công: <span className="text-amber-300">{formatMonsterStat(monster.base_attack, monster.attack_max)}</span></div>
                              </div>
                            )}
                          </div>
                          {monster.description && (
                            <p className="mt-1 text-[10px] leading-relaxed text-zinc-400 italic">
                              "{monster.description}"
                            </p>
                          )}
                        </div>
                      );
                    })
                  ) : (
                    <div className="rounded-lg border border-dashed border-white/10 p-4 text-center text-xs text-zinc-500">
                      {inspectedRegion?.zone_type === "SAFE_ZONE" || inspectedRegion?.zone_type === "HUB"
                        ? "Khu an toàn không có yêu thú sinh sống."
                        : "Chưa ghi nhận yêu thú dã ngoại tại khu vực này."}
                    </div>
                  )}
                </div>
              ) : activeTab === "drops" ? (
                <div className="space-y-3">
                  {/* Drops from Monsters */}
                  <div>
                    <div className="mb-1.5 text-[10px] font-semibold uppercase tracking-wider text-amber-300">
                      🎁 Chiến Lợi Phẩm ({currentDetails?.drops?.length ?? 0})
                    </div>
                    {currentDetails?.drops && currentDetails.drops.length > 0 ? (
                      <div className="grid grid-cols-1 gap-1.5 sm:grid-cols-2">
                        {currentDetails.drops.map((drop, idx) => {
                          const rarityColor =
                            drop.chance >= 0.7
                              ? "text-zinc-200 border-white/10"
                              : drop.chance >= 0.3
                              ? "text-cyan-300 border-cyan-800/40 bg-cyan-950/20"
                              : "text-amber-300 border-amber-800/40 bg-amber-950/20";

                          return (
                            <div
                              key={`${drop.id}-${idx}`}
                              className={`rounded-lg border p-2 flex flex-col justify-between ${rarityColor}`}
                            >
                              <div className="flex items-center justify-between gap-1">
                                <span className="font-medium text-xs truncate">{drop.name}</span>
                                <span className="text-[9px] font-mono text-zinc-400 shrink-0">
                                  {Math.round(drop.chance * 100)}%
                                </span>
                              </div>
                              <div className="mt-1 flex items-center justify-between text-[9px] text-zinc-500">
                                <span>SL: {drop.min_qty}–{drop.max_qty}</span>
                                {drop.sources && drop.sources.length > 0 && (
                                  <span className="truncate max-w-[100px] text-right" title={drop.sources.join(", ")}>
                                    {drop.sources[0]}
                                  </span>
                                )}
                              </div>
                            </div>
                          );
                        })}
                      </div>
                    ) : (
                      <div className="rounded-lg border border-dashed border-white/10 p-3 text-center text-xs text-zinc-500">
                        Chưa có dữ liệu chiến lợi phẩm quái rơi.
                      </div>
                    )}
                  </div>

                  {/* Wild Gu */}
                  {currentDetails?.wild_gus && currentDetails.wild_gus.length > 0 && (
                    <div>
                      <div className="mb-1.5 flex items-center gap-1 text-[10px] font-semibold uppercase tracking-wider text-purple-300">
                        <Bug className="h-3 w-3" /> Cổ Trùng Hoang Dã ({currentDetails.wild_gus.length})
                      </div>
                      <div className="flex flex-wrap gap-1.5">
                        {currentDetails.wild_gus.map((gu) => (
                          <span
                            key={gu.id}
                            className="rounded-md border border-purple-500/30 bg-purple-950/30 px-2 py-1 text-[10px] text-purple-200"
                          >
                            {gu.name}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Resources */}
                  {currentDetails?.resources && currentDetails.resources.length > 0 && (
                    <div>
                      <div className="mb-1.5 text-[10px] font-semibold uppercase tracking-wider text-emerald-300">
                        🌿 Linh Tài & Khoáng Vật ({currentDetails.resources.length})
                      </div>
                      <div className="flex flex-wrap gap-1.5">
                        {currentDetails.resources.map((res) => (
                          <span
                            key={res.id}
                            className="rounded-md border border-emerald-500/30 bg-emerald-950/30 px-2 py-1 text-[10px] text-emerald-200"
                          >
                            {res.name}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              ) : (
                /* Subregions List */
                <div className="space-y-1.5">
                  {subRegions.map((region) => {
                    const isSelected = inspectedRegion?.id === region.id;
                    const isCurrent = currentRegion?.id === region.id;

                    return (
                      <div
                        key={region.id}
                        role="button"
                        tabIndex={0}
                        onClick={() => setInspectedSubRegionId(region.id)}
                        className={`flex cursor-pointer items-center justify-between rounded-lg border px-3 py-2 transition ${
                          isSelected
                            ? "border-cyan-400/50 bg-cyan-900/30 text-cyan-100"
                            : isCurrent
                            ? "border-rose-400/30 bg-rose-400/[0.07] text-rose-200"
                            : "border-white/[0.07] bg-white/[0.025] text-zinc-300 hover:bg-white/[0.06]"
                        }`}
                      >
                        <div className="flex items-center gap-1.5 truncate">
                          {region.has_boss && <Crown className="h-3 w-3 text-amber-400 shrink-0" />}
                          <span className="truncate text-xs font-medium">{region.name}</span>
                        </div>
                        <div className="flex items-center gap-2 shrink-0">
                          {isCurrent ? (
                            <span className="text-[9px] font-semibold text-rose-300">ĐANG Ở ĐÂY</span>
                          ) : (
                            <span className="text-[9px] font-mono text-zinc-500">
                              C{region.min_rank}–C{region.max_rank}
                            </span>
                          )}
                        </div>
                      </div>
                    );
                  })}
                  {subRegions.length === 0 && (
                    <div className="rounded-lg border border-dashed border-white/10 p-3 text-center text-xs text-zinc-500">
                      Chưa có khu vực con.
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* Footer Status */}
            <div className="mt-3 flex items-center justify-between border-t border-white/10 pt-2 pb-1 text-[10px] text-zinc-400 shrink-0">
              <div className="flex items-center gap-2">
                <span className={`inline-block h-2 w-2 rounded-full ${mapRegion && !playerInZoomedRegion ? "bg-zinc-600" : "bg-rose-400"}`} />
                <span>{mapRegion && !playerInZoomedRegion ? "Bạn đang ở đại vực khác" : "Vị trí của bạn"}</span>
              </div>
              <span>{subRegions.length} khu vực</span>
            </div>
          </aside>
        </div>
      </section>
    </div>,
    document.body
  );
}
