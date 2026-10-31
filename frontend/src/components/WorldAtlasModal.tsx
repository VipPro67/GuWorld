"use client";

import { useEffect, useMemo, useState, useCallback } from "react";
import { createPortal } from "react-dom";
import { ArrowLeft, Compass, MapPin, X, Search, Shield, Skull, Crown, Sparkles, Bug, Package, Layers } from "lucide-react";
import type { Region } from "@/store/useGameStore";
import { getRegionDetailsFromCache, getStaticRegions } from "@/lib/atlas";

interface WorldAtlasModalProps {
  regions?: Region[];
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

const WORLD_WIDTH = 15_000;
const WORLD_HEIGHT = 10_000;
const REGION_COLORS = [
  "#355a50", "#5b442b", "#475635", "#70502b", "#594467", "#713f43", "#3e536b",
];

interface GreatDomainLayout {
  id: string;
  name: string;
  shortName: string;
  subTitle: string;
  path: string;
  labelX: number;
  labelY: number;
  gradientId: string;
  baseColor: string;
  strokeColor: string;
  wallName: string;
}

const GREAT_DOMAINS: Record<string, GreatDomainLayout> = {
  dai_vuc_trung_chau: {
    id: "dai_vuc_trung_chau",
    name: "Đại Vực Trung Châu",
    shortName: "TRUNG CHÂU",
    subTitle: "Thập Đại Cổ Phái",
    path: "M 360 260 L 640 260 L 690 380 L 630 490 L 370 490 L 310 380 Z",
    labelX: 500,
    labelY: 375,
    gradientId: "grad-trung-chau",
    baseColor: "#543e1d",
    strokeColor: "#f59e0b",
    wallName: "Bạch Khí Giới Bích",
  },
  dai_vuc_bac_nguyen: {
    id: "dai_vuc_bac_nguyen",
    name: "Đại Vực Bắc Nguyên",
    shortName: "BẮC NGUYÊN",
    subTitle: "Hoàng Kim Bộ Lạc",
    path: "M 180 115 L 280 40 L 720 40 L 820 115 L 640 260 L 360 260 Z",
    labelX: 500,
    labelY: 155,
    gradientId: "grad-bac-nguyen",
    baseColor: "#1d472c",
    strokeColor: "#10b981",
    wallName: "Liệp Phong Giới Bích",
  },
  dai_vuc_dong_hai: {
    id: "dai_vuc_dong_hai",
    name: "Đại Vực Đông Hải",
    shortName: "ĐÔNG HẢI",
    subTitle: "Vạn Đảo Hạo Hãn",
    path: "M 820 115 L 960 220 L 960 530 L 860 620 L 630 490 L 690 380 L 640 260 Z",
    labelX: 810,
    labelY: 375,
    gradientId: "grad-dong-hai",
    baseColor: "#0f4566",
    strokeColor: "#06b6d4",
    wallName: "Thương Thủy Giới Bích",
  },
  dai_vuc_nam_cuong: {
    id: "dai_vuc_nam_cuong",
    name: "Đại Vực Nam Cương",
    shortName: "NAM CƯƠNG",
    subTitle: "Thập Vạn Đại Sơn",
    path: "M 370 490 L 630 490 L 860 620 L 750 710 L 250 710 L 140 620 Z",
    labelX: 500,
    labelY: 600,
    gradientId: "grad-nam-cuong",
    baseColor: "#1a4538",
    strokeColor: "#34d399",
    wallName: "Chướng Khí Giới Bích",
  },
  dai_vuc_tay_mac: {
    id: "dai_vuc_tay_mac",
    name: "Đại Vực Tây Mạc",
    shortName: "TÂY MẠC",
    subTitle: "Vô Tận Sa Hải",
    path: "M 180 115 L 360 260 L 310 380 L 370 490 L 140 620 L 40 530 L 40 220 Z",
    labelX: 190,
    labelY: 375,
    gradientId: "grad-tay-mac",
    baseColor: "#573d1b",
    strokeColor: "#d97706",
    wallName: "Cuồng Cát Giới Bích",
  },
};

function normalizeSearch(value: string): string {
  return value.normalize("NFD").replace(/[\u0300-\u036f]/g, "").replace(/[đĐ]/g, "d").toLocaleLowerCase("vi-VN");
}

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

export function WorldAtlasModal({ regions: propRegions, currentRegionId, x, y, onClose }: WorldAtlasModalProps) {
  // Lấy dữ liệu build sẵn trực tiếp từ Region.csv (build-wiki-cache)
  // Bản đồ mở tức thì (0ms), toạ độ chuẩn xác 100%, không cần query database/API.
  const staticRegions = useMemo(() => getStaticRegions(), []);

  const regions = useMemo(() => {
    if (staticRegions && staticRegions.length > 0) {
      return staticRegions;
    }
    return propRegions || [];
  }, [staticRegions, propRegions]);

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

  // Region details cache
  const [detailsCache, setDetailsCache] = useState<Record<string, RegionDetailsData>>({});
  const loadingDetails = false;
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  const selectedMajorRegion = majorRegions.find((region) => region.id === selectedMajorRegionId) ?? currentMajorRegion;
  const mapRegion = majorRegions.find((region) => region.id === zoomedRegionId);

  const subRegions = useMemo(() => {
    return regions
      .filter((region) => region.parent_id === (mapRegion?.id ?? selectedMajorRegion?.id) && region.is_active)
      .sort((a, b) => ((a.atlas_y ?? a.min_y ?? 0) - (b.atlas_y ?? b.min_y ?? 0)) || ((a.atlas_x ?? a.min_x ?? 0) - (b.atlas_x ?? b.min_x ?? 0)));
  }, [regions, mapRegion?.id, selectedMajorRegion?.id]);

  const mapSubRegions = useMemo(() => {
    return mapRegion
      ? regions.filter((region) => region.parent_id === mapRegion.id && region.zone_type !== "HUB" && region.is_active)
      : [];
  }, [regions, mapRegion]);

  const mapHubs = useMemo(() => {
    return mapRegion
      ? regions.filter((region) => region.parent_id === mapRegion.id && region.zone_type === "HUB" && region.is_active)
      : [];
  }, [regions, mapRegion]);

  const mapRegionX = mapRegion?.atlas_x !== undefined && mapRegion?.atlas_x !== null ? mapRegion.atlas_x : (mapRegion?.min_x ?? 0);
  const mapRegionY = mapRegion?.atlas_y !== undefined && mapRegion?.atlas_y !== null ? mapRegion.atlas_y : (mapRegion?.min_y ?? 0);
  const mapRegionW = mapRegion?.width || 5000;
  const mapRegionH = mapRegion?.height || 5000;

  const mapViewBox = useMemo(() => {
    if (!mapRegion) {
      return "0 0 1000 750";
    }
    return `${mapRegionX} ${mapRegionY} ${mapRegionW} ${mapRegionH}`;
  }, [mapRegion, mapRegionX, mapRegionY, mapRegionW, mapRegionH]);

  const overviewPlayerPos = useMemo(() => {
    if (!currentMajorRegion) return null;
    const layout = GREAT_DOMAINS[currentMajorRegion.id];
    if (!layout) return null;
    return { x: layout.labelX, y: layout.labelY - 50 };
  }, [currentMajorRegion]);

  const playerAtlasX = currentRegion?.atlas_x !== undefined ? currentRegion.atlas_x + x : (currentRegion?.min_x !== undefined ? currentRegion.min_x + x : x);
  const playerAtlasY = currentRegion?.atlas_y !== undefined ? currentRegion.atlas_y + y : (currentRegion?.min_y !== undefined ? currentRegion.min_y + y : y);

  const hasPlayerPosition = playerAtlasX >= 0 && playerAtlasX <= WORLD_WIDTH && playerAtlasY >= 0 && playerAtlasY <= WORLD_HEIGHT;
  const playerInZoomedRegion = !!mapRegion
    && playerAtlasX >= mapRegionX
    && playerAtlasX <= mapRegionX + mapRegionW
    && playerAtlasY >= mapRegionY
    && playerAtlasY <= mapRegionY + mapRegionH;

  const inspectedRegion = useMemo(() => {
    return regions.find((r) => r.id === inspectedSubRegionId) || currentRegion || subRegions[0];
  }, [inspectedSubRegionId, regions, currentRegion, subRegions]);

  // Resolve region details directly from CSV wiki-cache (instant 0ms response)
  const getDetails = useCallback((regionId: string): RegionDetailsData | null => {
    if (!regionId) return null;
    if (detailsCache[regionId]) return detailsCache[regionId];
    const resolved = getRegionDetailsFromCache(regionId, regions);
    if (resolved) {
      setDetailsCache((prev) => ({ ...prev, [regionId]: resolved }));
      return resolved;
    }
    return null;
  }, [detailsCache, regions]);

  useEffect(() => {
    if (inspectedRegion?.id) {
      getDetails(inspectedRegion.id);
    }
  }, [inspectedRegion?.id, getDetails]);

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

  const currentDetails = inspectedRegion
    ? (detailsCache[inspectedRegion.id] || getRegionDetailsFromCache(inspectedRegion.id, regions))
    : null;

  // Search filter
  const cleanSearch = normalizeSearch(searchQuery.trim());
  const matchesContent = useCallback((r: Region) => {
    if (normalizeSearch(`${r.name} ${r.id}`).includes(cleanSearch)) {
      return true;
    }
    const cached = detailsCache[r.id] || getRegionDetailsFromCache(r.id, regions);
    if (cached) {
      if (cached.monsters?.some((m) => normalizeSearch(m.name).includes(cleanSearch))) return true;
      if (cached.drops?.some((d) => normalizeSearch(d.name).includes(cleanSearch))) return true;
      if (cached.wild_gus?.some((g) => normalizeSearch(g.name).includes(cleanSearch))) return true;
      if (cached.resources?.some((res) => normalizeSearch(res.name).includes(cleanSearch))) return true;
    }
    return false;
  }, [cleanSearch, detailsCache, regions]);

  const searchResults = useMemo(() => cleanSearch
    ? regions.filter((region) => region.is_active && matchesContent(region))
    : [], [cleanSearch, matchesContent, regions]);

  const isRegionMatch = useCallback((region: Region) => {
    if (!cleanSearch) return true;
    return searchResults.some((result) => result.id === region.id || result.parent_id === region.id);
  }, [cleanSearch, searchResults]);

  const selectSearchResult = (region: Region) => {
    const majorId = region.zone_type === "NGU_VUC" ? region.id : region.parent_id;
    if (majorId) {
      setSelectedMajorRegionId(majorId);
      setZoomedRegionId(majorId);
    }
    const inspectedId = region.zone_type === "NGU_VUC"
      ? regions.find((child) => child.parent_id === region.id && child.is_active)?.id ?? region.id
      : region.id;
    setInspectedSubRegionId(inspectedId);
    const details = getRegionDetailsFromCache(inspectedId, regions);
    if (details?.monsters.some((monster) => normalizeSearch(monster.name).includes(cleanSearch))) setActiveTab("monsters");
    else if (details?.drops.some((drop) => normalizeSearch(drop.name).includes(cleanSearch)) || details?.wild_gus.some((gu) => normalizeSearch(gu.name).includes(cleanSearch)) || details?.resources.some((resource) => normalizeSearch(resource.name).includes(cleanSearch))) setActiveTab("drops");
    else setActiveTab(region.zone_type === "NGU_VUC" ? "list" : "monsters");
    setSearchQuery("");
  };

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
        className="mmo-panel-theme mmo-paper-theme mmo-paper-surface flex max-h-[calc(100dvh-env(safe-area-inset-top)-env(safe-area-inset-bottom)-1rem)] w-full max-w-6xl flex-col overflow-hidden rounded-xl text-[#49331f]"
      >
        {/* Header */}
        <header className="mmo-paper-heading flex flex-wrap items-center justify-between gap-3 px-4 py-3 sm:px-5">
          <div className="flex items-center gap-3">
            <div className="mmo-paper-icon">
              <Compass className="h-5 w-5" />
            </div>
            <div>
              <h2 id="world-atlas-title" className="font-serif text-base font-bold text-[#49331f] sm:text-lg">
                Bản Đồ Cổ Giới
              </h2>
              <p className="text-[11px] text-[#795a34]">
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
                aria-label="Tìm khu vực, quái, vật phẩm hoặc Cổ"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="h-8 w-36 sm:w-44 lg:w-56 rounded-lg border border-cyan-900/60 bg-[#302318] pl-8 pr-7 text-xs text-zinc-100 placeholder:text-zinc-500 focus:border-cyan-400 focus:outline-none"
              />
              {searchQuery && (
                <button
                  onClick={() => setSearchQuery("")}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-200"
                >
                  <X className="h-3 w-3" />
                </button>
              )}
              {cleanSearch && <div className="absolute right-0 top-full z-30 mt-1 max-h-64 w-[min(80vw,340px)] overflow-y-auto rounded-lg border border-[#a88952]/60 bg-[#211a12] p-1 shadow-xl">
                <div className="px-2 py-1 text-[10px] text-[#b9a584]">{searchResults.length} khu vực phù hợp</div>
                {searchResults.slice(0, 40).map((region) => <button key={region.id} type="button" onClick={() => selectSearchResult(region)} className="flex w-full items-center justify-between gap-2 rounded px-2 py-1.5 text-left text-xs text-[#f1dfb8] hover:bg-[#665033]/70">
                  <span className="truncate">{region.name}</span><span className="shrink-0 text-[9px] text-[#b9a584]">C{region.min_rank}–C{region.max_rank}</span>
                </button>)}
                {searchResults.length > 40 && <div className="px-2 py-1 text-[10px] text-[#b9a584]">Nhập cụ thể hơn để xem thêm kết quả.</div>}
              </div>}
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
            <div className="relative mx-auto flex w-full max-w-[340px] sm:max-w-[460px] lg:max-w-[560px] flex-1 items-center justify-center overflow-hidden rounded-xl border border-cyan-950/80 bg-[#201a12] shadow-inner">
              <div
                className="pointer-events-none absolute inset-0 opacity-25"
                style={{
                  backgroundImage:
                    "radial-gradient(circle at 50% 50%, rgba(222,180,104,.17), transparent 70%), linear-gradient(rgba(222,180,104,.08) 1px, transparent 1px), linear-gradient(90deg, rgba(222,180,104,.08) 1px, transparent 1px)",
                  backgroundSize: "auto, 32px 32px, 32px 32px",
                }}
              />

              <svg
                viewBox={mapViewBox}
                className={`relative block w-full ${mapRegion ? "aspect-square" : "aspect-[4/3]"}`}
                aria-label={mapRegion ? `Bản đồ ${mapRegion.name}` : "Bản đồ các đại vực"}
              >
                <defs>
                  {/* Glowing filter for Regional Walls */}
                  <filter id="wall-glow" x="-20%" y="-20%" width="140%" height="140%">
                    <feGaussianBlur stdDeviation="3.5" result="blur" />
                    <feComposite in="SourceGraphic" in2="blur" operator="over" />
                  </filter>

                  {/* Domain Background Gradients */}
                  <radialGradient id="grad-trung-chau" cx="50%" cy="50%" r="60%">
                    <stop offset="0%" stopColor="#8d6832" stopOpacity="0.9" />
                    <stop offset="65%" stopColor="#5a421e" stopOpacity="0.9" />
                    <stop offset="100%" stopColor="#36260f" stopOpacity="0.95" />
                  </radialGradient>

                  <linearGradient id="grad-bac-nguyen" x1="50%" y1="0%" x2="50%" y2="100%">
                    <stop offset="0%" stopColor="#25643e" stopOpacity="0.9" />
                    <stop offset="60%" stopColor="#1a4a2d" stopOpacity="0.9" />
                    <stop offset="100%" stopColor="#102e1b" stopOpacity="0.95" />
                  </linearGradient>

                  <linearGradient id="grad-dong-hai" x1="100%" y1="50%" x2="0%" y2="50%">
                    <stop offset="0%" stopColor="#156e99" stopOpacity="0.9" />
                    <stop offset="60%" stopColor="#0f4c6e" stopOpacity="0.9" />
                    <stop offset="100%" stopColor="#0a2a3b" stopOpacity="0.95" />
                  </linearGradient>

                  <linearGradient id="grad-nam-cuong" x1="50%" y1="100%" x2="50%" y2="0%">
                    <stop offset="0%" stopColor="#215e4a" stopOpacity="0.9" />
                    <stop offset="60%" stopColor="#184637" stopOpacity="0.9" />
                    <stop offset="100%" stopColor="#0d261e" stopOpacity="0.95" />
                  </linearGradient>

                  <linearGradient id="grad-tay-mac" x1="0%" y1="50%" x2="100%" y2="50%">
                    <stop offset="0%" stopColor="#7e5722" stopOpacity="0.9" />
                    <stop offset="60%" stopColor="#583c16" stopOpacity="0.9" />
                    <stop offset="100%" stopColor="#33210a" stopOpacity="0.95" />
                  </linearGradient>
                </defs>

                {/* 1. Large 5 Major Regions view (Bố cục Ngũ Vực Chuẩn Cổ Chân Nhân) */}
                {!mapRegion && (
                  <g className="transition-all duration-300">
                    {/* Background Domain Polygons */}
                    {majorRegions.map((region) => {
                      const layout = GREAT_DOMAINS[region.id];
                      if (!layout) return null;
                      const isSelected = selectedMajorRegionId === region.id;
                      const isCurrent = currentMajorRegion?.id === region.id;
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
                            setInspectedSubRegionId(regions.find((child) => child.parent_id === region.id && child.is_active)?.id ?? region.id);
                          }}
                          onKeyDown={(event) => {
                            if (event.key === "Enter" || event.key === " ") {
                              event.preventDefault();
                              setSelectedMajorRegionId(region.id);
                              setZoomedRegionId(region.id);
                              setInspectedSubRegionId(regions.find((child) => child.parent_id === region.id && child.is_active)?.id ?? region.id);
                            }
                          }}
                          className={`cursor-pointer transition-all duration-200 hover:brightness-125 ${!isMatched ? "opacity-30" : "opacity-100"}`}
                        >
                          <path
                            d={layout.path}
                            fill={`url(#${layout.gradientId})`}
                            stroke={isSelected ? "#f59e0b" : isCurrent ? "#ef4444" : layout.strokeColor}
                            strokeWidth={isSelected ? 4 : isCurrent ? 3.5 : 2}
                            strokeOpacity={isSelected || isCurrent ? 1 : 0.6}
                            className="transition-all duration-200"
                          />

                          {/* Domain Center Label Group */}
                          <g transform={`translate(${layout.labelX}, ${layout.labelY})`}>
                            {/* Short Name */}
                            <text
                              x="0"
                              y="-12"
                              textAnchor="middle"
                              dominantBaseline="middle"
                              fill={isSelected ? "#fffbeb" : "#fef3c7"}
                              fontSize="21"
                              fontWeight="800"
                              letterSpacing="2.5"
                              className="pointer-events-none select-none drop-shadow-md"
                            >
                              {layout.shortName}
                            </text>

                            {/* Subtitle / Characteristic */}
                            <text
                              x="0"
                              y="10"
                              textAnchor="middle"
                              dominantBaseline="middle"
                              fill={isSelected ? "#fde68a" : "#d8b485"}
                              fontSize="11"
                              fontWeight="500"
                              className="pointer-events-none select-none"
                            >
                              {layout.subTitle}
                            </text>

                            {/* Rank range badge */}
                            <g transform="translate(0, 27)">
                              <rect x="-32" y="-8.5" width="64" height="17" rx="8.5" fill="#000000" fillOpacity="0.5" stroke={layout.strokeColor} strokeWidth="1" strokeOpacity="0.5" />
                              <text
                                x="0"
                                y="1"
                                textAnchor="middle"
                                dominantBaseline="middle"
                                fill="#fcd34d"
                                fontSize="10"
                                fontWeight="700"
                                fontFamily="monospace"
                                className="pointer-events-none select-none"
                              >
                                C{region.min_rank}–C{region.max_rank}
                              </text>
                            </g>

                            {/* Is Current Player Badge */}
                            {isCurrent && (
                              <g transform="translate(0, 48)">
                                <rect x="-42" y="-8.5" width="84" height="17" rx="8.5" fill="#ef4444" fillOpacity="0.9" stroke="#fee2e2" strokeWidth="0.8" />
                                <text
                                  x="0"
                                  y="1"
                                  textAnchor="middle"
                                  dominantBaseline="middle"
                                  fill="#ffffff"
                                  fontSize="9"
                                  fontWeight="800"
                                  letterSpacing="0.5"
                                  className="pointer-events-none select-none"
                                >
                                  ĐANG Ở ĐÂY
                                </text>
                              </g>
                            )}
                          </g>
                        </g>
                      );
                    })}

                    {/* Regional Wall Glowing Lines (Ngũ Vực Giới Bích) */}
                    <g className="pointer-events-none" filter="url(#wall-glow)">
                      {/* Trung Châu Ring Wall */}
                      <path
                        d="M 360 260 L 640 260 L 690 380 L 630 490 L 370 490 L 310 380 Z"
                        fill="none"
                        stroke="#ef4444"
                        strokeWidth="3.5"
                        strokeLinejoin="round"
                        opacity="0.85"
                      />
                      {/* Corner Radiating Walls */}
                      <line x1="180" y1="115" x2="360" y2="260" stroke="#ef4444" strokeWidth="3" strokeLinecap="round" opacity="0.85" />
                      <line x1="820" y1="115" x2="640" y2="260" stroke="#ef4444" strokeWidth="3" strokeLinecap="round" opacity="0.85" />
                      <line x1="140" y1="620" x2="370" y2="490" stroke="#ef4444" strokeWidth="3" strokeLinecap="round" opacity="0.85" />
                      <line x1="860" y1="620" x2="630" y2="490" stroke="#ef4444" strokeWidth="3" strokeLinecap="round" opacity="0.85" />
                    </g>

                    {/* Regional Wall Name Tags */}
                    <g className="pointer-events-none select-none">
                      {/* Liệp Phong (Bắc Nguyên ↔ Tây Mạc) */}
                      <g transform="translate(265, 182)">
                        <rect x="-35" y="-8" width="70" height="16" rx="4" fill="#18181b" fillOpacity="0.85" stroke="#ef4444" strokeWidth="1" />
                        <text x="0" y="1" textAnchor="middle" dominantBaseline="middle" fill="#fca5a5" fontSize="8" fontWeight="600">Liệp Phong Giới</text>
                      </g>

                      {/* Thương Thủy (Bắc Nguyên ↔ Đông Hải) */}
                      <g transform="translate(735, 182)">
                        <rect x="-36" y="-8" width="72" height="16" rx="4" fill="#18181b" fillOpacity="0.85" stroke="#ef4444" strokeWidth="1" />
                        <text x="0" y="1" textAnchor="middle" dominantBaseline="middle" fill="#fca5a5" fontSize="8" fontWeight="600">Thương Thủy Giới</text>
                      </g>

                      {/* Cuồng Cát (Nam Cương ↔ Tây Mạc) */}
                      <g transform="translate(250, 560)">
                        <rect x="-35" y="-8" width="70" height="16" rx="4" fill="#18181b" fillOpacity="0.85" stroke="#ef4444" strokeWidth="1" />
                        <text x="0" y="1" textAnchor="middle" dominantBaseline="middle" fill="#fca5a5" fontSize="8" fontWeight="600">Cuồng Cát Giới</text>
                      </g>

                      {/* Chướng Khí (Nam Cương ↔ Đông Hải) */}
                      <g transform="translate(750, 560)">
                        <rect x="-36" y="-8" width="72" height="16" rx="4" fill="#18181b" fillOpacity="0.85" stroke="#ef4444" strokeWidth="1" />
                        <text x="0" y="1" textAnchor="middle" dominantBaseline="middle" fill="#fca5a5" fontSize="8" fontWeight="600">Chướng Khí Giới</text>
                      </g>

                      {/* Bạch Khí (Bao quanh Trung Châu) */}
                      <g transform="translate(500, 260)">
                        <rect x="-35" y="-8" width="70" height="16" rx="4" fill="#18181b" fillOpacity="0.85" stroke="#ef4444" strokeWidth="1" />
                        <text x="0" y="1" textAnchor="middle" dominantBaseline="middle" fill="#fca5a5" fontSize="8" fontWeight="600">Bạch Khí Giới</text>
                      </g>
                    </g>

                    {/* Player Pointer on Overview Mode */}
                    {overviewPlayerPos && (
                      <g transform={`translate(${overviewPlayerPos.x}, ${overviewPlayerPos.y})`} className="pointer-events-none">
                        <circle cx="0" cy="0" r="16" fill="#f43f5e" fillOpacity="0.3" />
                        <circle cx="0" cy="0" r="7" fill="#f43f5e" stroke="#fff1f2" strokeWidth="2.5" />
                        <path d="M 0 -18 L -4 -6 L 4 -6 Z" fill="#fff1f2" />
                      </g>
                    )}
                  </g>
                )}

                {/* 2. Sub-regions grid view */}
                {mapRegion &&
                  mapSubRegions.map((region, index) => {
                    const minX = region.atlas_x !== undefined && region.atlas_x !== null ? region.atlas_x : (region.min_x ?? 0);
                    const minY = region.atlas_y !== undefined && region.atlas_y !== null ? region.atlas_y : (region.min_y ?? 0);
                    const width = region.width || 1000;
                    const height = region.height || 1000;
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

                    const fontSize = firstLine.length > 14 ? 90 : 110;

                    return (
                      <g
                        key={region.id}
                        role="button"
                        tabIndex={0}
                        aria-label={`Khu vực ${region.name}`}
                        onClick={() => {
                          setInspectedSubRegionId(region.id);
                        }}
                        onKeyDown={(event) => {
                          if (event.key === "Enter" || event.key === " ") {
                            event.preventDefault();
                            setInspectedSubRegionId(region.id);
                          }
                        }}
                        className={`cursor-pointer transition duration-150 hover:brightness-125 ${!isMatched ? "opacity-25" : "opacity-100"}`}
                      >
                        <title>{`${region.name} (C${region.min_rank}-C${region.max_rank})`}</title>
                        <rect
                          x={minX + 20}
                          y={minY + 20}
                          width={Math.max(1, width - 40)}
                          height={Math.max(1, height - 40)}
                          rx="50"
                          fill={REGION_COLORS[index % REGION_COLORS.length]}
                          fillOpacity={isInspected ? 0.8 : 0.45}
                          stroke={isInspected ? "#e9bd6c" : isCurrent ? "#fb7185" : hasBoss ? "#f59e0b" : "#9b7644"}
                          strokeOpacity={isInspected || isCurrent ? 1 : 0.6}
                          strokeWidth={isInspected ? 35 : isCurrent ? 25 : 12}
                        />

                        {/* Badges on corner */}
                        {hasBoss && (
                          <g transform={`translate(${minX + width - 90}, ${minY + 30}) scale(0.35)`}>
                            <rect width="180" height="180" rx="40" fill="#78350f" fillOpacity="0.8" />
                            <text x="90" y="125" textAnchor="middle" fontSize="110">👑</text>
                          </g>
                        )}
                        {isSafe && (
                          <g transform={`translate(${minX + 30}, ${minY + 30}) scale(0.35)`}>
                            <rect width="180" height="180" rx="40" fill="#065f46" fillOpacity="0.8" />
                            <text x="90" y="125" textAnchor="middle" fontSize="110">🛡️</text>
                          </g>
                        )}

                        {/* Name text */}
                        <text
                          x={centerX}
                          y={secondLine ? centerY - 40 : centerY}
                          textAnchor="middle"
                          dominantBaseline="middle"
                          fill={isInspected ? "#fff3d8" : isCurrent ? "#fff1f2" : "#f4e8cb"}
                          fontSize={fontSize}
                          fontWeight="700"
                        >
                          <tspan x={centerX}>{firstLine}</tspan>
                          {secondLine && <tspan x={centerX} dy="1.25em">{secondLine}</tspan>}
                        </text>

                        {/* Rank indicator at bottom of cell */}
                        <text
                          x={centerX}
                          y={minY + height - 45}
                          textAnchor="middle"
                          fill={isInspected ? "#e9c77d" : "#b9a584"}
                          fontSize="70"
                          fontWeight="600"
                        >
                          C{region.min_rank}–C{region.max_rank}
                        </text>
                      </g>
                    );
                  })}

                {/* 3. Hubs */}
                {mapHubs.map((hub) => {
                  const hubX = hub.atlas_x !== undefined ? hub.atlas_x + (hub.width || 1000) / 2 : hub.entry_x ?? ((hub.min_x ?? 0) + (hub.max_x ?? 0)) / 2;
                  const hubY = hub.atlas_y !== undefined ? hub.atlas_y + (hub.height || 1000) / 2 : hub.entry_y ?? ((hub.min_y ?? 0) + (hub.max_y ?? 0)) / 2;
                  const isInspected = inspectedRegion?.id === hub.id;
                  return (
                    <g
                      key={hub.id}
                      onClick={() => setInspectedSubRegionId(hub.id)}
                      className="cursor-pointer transition hover:scale-105"
                    >
                      <title>{hub.name}</title>
                      <circle cx={hubX} cy={hubY} r="125" fill="#eab308" stroke="#fef08a" strokeWidth={isInspected ? 35 : 20} />
                      <text x={hubX} y={hubY + 250} textAnchor="middle" fill="#fef08a" fontSize="120" fontWeight="700">
                        {hub.name}
                      </text>
                    </g>
                  );
                })}

                {/* 4. Player Pointer */}
                {hasPlayerPosition && (!mapRegion || playerInZoomedRegion) && (
                  <g className="pointer-events-none">
                    <circle cx={playerAtlasX} cy={playerAtlasY} r={mapRegion ? 120 : 260} fill="#f43f5e" fillOpacity="0.25" />
                    <circle cx={playerAtlasX} cy={playerAtlasY} r={mapRegion ? 50 : 100} fill="#f43f5e" stroke="#fff1f2" strokeWidth={mapRegion ? 20 : 40} />
                    <path d={`M ${playerAtlasX} ${playerAtlasY - (mapRegion ? 180 : 360)} L ${playerAtlasX - (mapRegion ? 40 : 80)} ${playerAtlasY - (mapRegion ? 35 : 70)} L ${playerAtlasX + (mapRegion ? 40 : 80)} ${playerAtlasY - (mapRegion ? 35 : 70)} Z`} fill="#fff1f2" />
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
          <aside className="flex flex-col shrink-0 lg:shrink lg:min-h-0 lg:flex-1 bg-[#241a12]/70 p-3 sm:p-4 pb-12 sm:pb-6 lg:pb-4">
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
