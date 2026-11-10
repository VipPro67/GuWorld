"use client";

import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Compass, Sparkles, DoorOpen, Landmark, ShieldAlert, Map as MapIcon, Package, Eye, Mountain, Waypoints } from "lucide-react";
import { useGameStore, Player } from "@/store/useGameStore";
import { useWorldStore } from "@/store/useWorldStore";
import { usePolling } from "@/lib/usePolling";
import { getWorldResourceGroup, WORLD_RESOURCE_ASSETS } from "@/lib/worldResourceAssets";
import { WorldAtlasModal } from "@/components/WorldAtlasModal";
import { useUIStore } from "@/store/useUIStore";
import { MonsterSprite } from "@/components/MonsterSprite";

interface ExplorationMapProps {
  player: Player;
}

interface GridCell {
  x: number;
  y: number;
  cell_type: string;
  content: string;
}

interface GridData {
  x: number;
  y: number;
  vision_radius: number;
  max_vision_radius?: number;
  cells: GridCell[];
  stats: { movement?: number; investigation?: number };
  players_on_map?: { id: string; username: string; avatar_url: string; x: number; y: number }[];
  ground_drops?: { id: string; x: number; y: number; loser_name?: string }[];
  world_width?: number;
  world_height?: number;
}

export const ExplorationMap: React.FC<ExplorationMapProps> = ({ player }) => {
  const regions = useGameStore((state) => state.regions);
  const regionsLoaded = useGameStore((state) => state.regionsLoaded);
  const fetchRegions = useGameStore((state) => state.fetchRegions);
  const fetchGrid = useGameStore((state) => state.fetchGrid);
  const movePlayer = useGameStore((state) => state.movePlayer);
  const enterAperturePortal = useGameStore((state) => state.enterPortal);
  const harvestVein = useWorldStore((state) => state.harvestVein);
  const mapMarkers = useWorldStore((state) => state.zoneMarkers);
  const mapMarkersRegion = useWorldStore((state) => state.zoneMarkersRegion);
  const fetchZoneMarkers = useWorldStore((state) => state.fetchZoneMarkers);
  const resourcePoints = useWorldStore((state) => state.resourcePoints);
  const resourcePointsRegion = useWorldStore((state) => state.resourcePointsRegion);
  const fetchWorldResourcePoints = useWorldStore((state) => state.fetchWorldResourcePoints);
  const collectWorldResourcePoint = useWorldStore((state) => state.collectWorldResourcePoint);
  const lootGroundDrop = useGameStore((state) => state.lootGroundDrop);
  const addToast = useGameStore((state) => state.addToast);
  const actionLoading = useGameStore((state) => state.actionLoading);
  const party = useGameStore((state) => state.party);
  const guDictionary = useGameStore((state) => state.guDictionary);
  const itemDictionary = useGameStore((state) => state.itemDictionary);
  const monsterDictionary = useGameStore((state) => state.monsterDictionary);

  const [gridData, setGridData] = useState<GridData | null>(null);
  const [preferredRadius, setPreferredRadius] = useState<number>(() => {
    if (typeof window !== "undefined") {
      const saved = localStorage.getItem("gu_explore_vision_radius");
      if (saved) {
        const val = parseInt(saved, 10);
        if (val >= 2) return val;
      }
    }
    return 2; // Mặc định 5x5
  });
  const [lastLog, setLastLog] = useState<string | null>(null);
  const [showAtlas, setShowAtlas] = useState(false);
  const lastMoveTimeRef = useRef(0);
  const setView = useUIStore((state) => state.setView);

  useEffect(() => {
    if (!regionsLoaded) fetchRegions();
  }, [regionsLoaded, fetchRegions]);

  const loadGrid = useCallback(async () => {
    const data = await fetchGrid(preferredRadius);
    if (data && Array.isArray(data.cells)) {
      setGridData(data as GridData);
    }
  }, [fetchGrid, preferredRadius]);

  const handleRadiusChange = (newRadius: number) => {
    setPreferredRadius(newRadius);
    if (typeof window !== "undefined") {
      localStorage.setItem("gu_explore_vision_radius", String(newRadius));
    }
  };

  useEffect(() => {
    void loadGrid();
  }, [player.current_region, player.x, player.y, loadGrid]);
  usePolling(loadGrid, 10000);

  useEffect(() => {
    if (mapMarkersRegion !== player.current_region) void fetchZoneMarkers(player.current_region);
    if (resourcePointsRegion !== player.current_region) void fetchWorldResourcePoints(player.current_region);
  }, [player.current_region, mapMarkersRegion, resourcePointsRegion, fetchZoneMarkers, fetchWorldResourcePoints]);

  const isMemberNotLeader = Boolean(player.party_id) && (party === null || party.leader_id !== player.id);
  const currentRegion = regions.find((r) => r.id === player.current_region);
  const regionName = currentRegion?.name || "Thế Giới Cổ Sư";

  const handleCellClick = async (x: number, y: number, isMovable: boolean, cell?: GridCell, marker?: { kind: string; id: string; isMine?: boolean }, resourcePointId?: string) => {
    if (marker?.kind === "aperture_portal" && !marker.isMine && Math.hypot(x - player.x, y - player.y) <= 3) {
      await enterAperturePortal(marker.id);
      await loadGrid();
      return;
    }
    if (resourcePointId && Math.max(Math.abs(x - player.x), Math.abs(y - player.y)) <= 6) {
      if (await collectWorldResourcePoint(resourcePointId)) await fetchWorldResourcePoints(player.current_region);
      return;
    }
    if (cell?.cell_type === "PrimevalVein"
      && Math.max(Math.abs(x - player.x), Math.abs(y - player.y)) <= 3) {
      await harvestVein(cell.content);
      await loadGrid();
      return;
    }

    // Check ground drops on cell
    const dropOnCell = gridData?.ground_drops?.find((d) => d.x === x && d.y === y);
    if (dropOnCell && Math.max(Math.abs(x - player.x), Math.abs(y - player.y)) <= 1) {
      const res = await lootGroundDrop(dropOnCell.id);
      if (res?.success) {
        await loadGrid();
        return;
      }
    }

    if (!isMovable || actionLoading || player.active_combat) return;
    if (isMemberNotLeader) {
      addToast("Chỉ có Trưởng nhóm mới có quyền di chuyển trên bản đồ!", "error");
      return;
    }

    const now = Date.now();
    if (now - lastMoveTimeRef.current < 600) return;
    lastMoveTimeRef.current = now;

    setLastLog(null);
    const res = await movePlayer(x, y);
    if (res) {
      setLastLog(res.log);
      await loadGrid();
    }
  };

  const getEntityName = (type: string, content: string) => {
    if (type === "EXIT_PORTAL" || type === "PORTAL_EXIT") return "Cổng trở về";
    if (type === "ENTRY_PORTAL") return `Cổng vào bí cảnh: ${content.replace(/^spatial_realm_/, "").replace(/_/g, " ")}`;
    if (type === "INHERITANCE_ALTAR") return `Đài Truyền Thừa: ${content.replace(/^altar_/, "").replace(/_/g, " ")}`;
    if (type === "Monster" || type === "Boss") return monsterDictionary[content]?.name || content.split(" ")[0];
    if (type === "Resource") return itemDictionary[content]?.name || content.replace(/_/g, " ").replace(/\b\w/g, (c: string) => c.toUpperCase());
    if (type === "WildGu") return guDictionary[content]?.name || content;
    if (type === "PrimevalVein") return "Nguyên Tuyền";
    return content;
  };

  const getCellIcon = (type: string, content: string) => {
    switch (type) {
      case "Resource": {
        const item = itemDictionary[content];
        const group = getWorldResourceGroup(content, item?.item_type || item?.type);
        return <img src={WORLD_RESOURCE_ASSETS[group]} alt="" className="h-6 w-6 object-contain" />;
      }
      case "Monster":
      case "Boss":
        return (
          <MonsterSprite
            templateId={content}
            alt={getEntityName(type, content)}
            className={`${type === "Boss" ? "w-[84%] h-[84%]" : "w-[72%] h-[72%]"} object-contain drop-shadow-[0_0_8px_rgba(251,146,60,0.45)]`}
          />
        );
      case "WildGu":
        return <Sparkles className="w-5 h-5 text-amber-400 drop-shadow-[0_0_8px_rgba(251,191,36,0.5)] animate-spin" />;
      case "PrimevalVein":
        return <Landmark className="w-5 h-5 text-cyan-400 drop-shadow-[0_0_8px_rgba(34,211,238,0.5)]" />;
      case "ENTRY_PORTAL":
      case "EXIT_PORTAL":
      case "PORTAL_EXIT":
        return <DoorOpen className="w-5 h-5 text-emerald-400 drop-shadow-[0_0_8px_rgba(52,211,153,0.5)]" />;
      case "INHERITANCE_ALTAR":
        return <Landmark className="w-5 h-5 text-purple-400 drop-shadow-[0_0_8px_rgba(192,132,252,0.5)]" />;
      default:
        return null;
    }
  };

  const maxAllowedRadius = Math.max(2, gridData?.max_vision_radius || 2);
  const radius = Math.min(maxAllowedRadius, Math.max(2, preferredRadius));
  const diameter = radius * 2 + 1;
  const maxContainerWidth = radius <= 2 ? "max-w-[560px]" : radius === 3 ? "max-w-[660px]" : "max-w-[760px]";

  const cellMap = useMemo(() => {
    const map = new Map<string, GridCell>();
    if (gridData?.cells) {
      for (const cell of gridData.cells) {
        map.set(`${cell.x},${cell.y}`, cell);
      }
    }
    return map;
  }, [gridData?.cells]);

  const markersByCell = useMemo(() => {
    const map = new Map<string, { kind: string; id: string; name: string; isMine?: boolean }[]>();
    if (mapMarkersRegion === player.current_region) {
      for (const marker of mapMarkers) {
        const key = `${marker.x},${marker.y}`;
        map.set(key, [...(map.get(key) || []), {
          kind: marker.kind,
          id: marker.id,
          name: marker.name,
          isMine: marker.detail?.is_mine === true,
        }]);
      }
    }
    if (resourcePointsRegion === player.current_region) {
      for (const point of resourcePoints) {
        if (point.region_id && point.region_id !== player.current_region) continue;
        if (point.state === "GONE") continue;
        const key = `${Math.round(point.x)},${Math.round(point.y)}`;
        map.set(key, [...(map.get(key) || []), { kind: "resource_point", id: point.id, name: point.name }]);
      }
    }
    return map;
  }, [mapMarkers, mapMarkersRegion, player.current_region, resourcePoints, resourcePointsRegion]);

  const gridArray = useMemo(() => {
    const arr = [];
    for (let dy = -radius; dy <= radius; dy++) {
      for (let dx = -radius; dx <= radius; dx++) {
        const x = player.x + dx;
        const y = player.y + dy;
        const isOutOfBounds =
          x < 0 || y < 0 || (gridData?.world_width && x >= gridData.world_width) || (gridData?.world_height && y >= gridData.world_height);
        arr.push({ x, y, isOutOfBounds });
      }
    }
    return arr;
  }, [player.x, player.y, radius, gridData?.world_width, gridData?.world_height]);

  return (
    <div className="w-full flex flex-col gap-4 max-w-5xl mx-auto p-2 sm:p-4">
      {/* Header Bar */}
      <div className="flex flex-wrap items-center justify-between gap-3 bg-zinc-950/80 border border-zinc-800 rounded-2xl p-4 shadow-xl backdrop-blur-md">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400">
            <Compass className="w-6 h-6 animate-spin-slow" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="text-xs uppercase font-mono tracking-widest text-emerald-500 font-semibold">Khu vực hiện tại</span>
              <span className="text-xs text-zinc-500">·</span>
              <span className="text-xs font-mono text-zinc-400">({player.x}, {player.y})</span>
            </div>
            <h2 className="text-lg font-serif font-bold text-zinc-100">{regionName}</h2>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2.5 sm:gap-3">
          {/* Vision Range Selector */}
          <div className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-xl bg-zinc-900/90 border border-zinc-800 text-xs shadow-md">
            <Eye className="w-4 h-4 text-emerald-400 shrink-0" />
            <span className="text-zinc-400 font-serif hidden sm:inline">Tầm nhìn:</span>
            <select
              value={radius}
              onChange={(e) => handleRadiusChange(Number(e.target.value))}
              className="bg-zinc-950 border border-zinc-700/60 rounded-lg px-2 py-1 text-emerald-400 font-mono text-xs font-semibold focus:outline-none cursor-pointer"
            >
              <option value={2}>5×5 (Ô to chuẩn)</option>
              {maxAllowedRadius >= 3 && <option value={3}>7×7 (Mở rộng)</option>}
              {maxAllowedRadius >= 4 && <option value={4}>9×9 (Xa)</option>}
              {maxAllowedRadius >= 5 && <option value={5}>11×11 (Toàn cảnh)</option>}
            </select>
          </div>

          <button
            onClick={() => setView("dashboard")}
            className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-zinc-900 border border-zinc-700/60 text-zinc-300 hover:text-white hover:bg-zinc-800 transition-all font-serif text-xs font-semibold cursor-pointer shadow-lg"
          >
            <span>⬅ Sảnh</span>
          </button>
          <button
            onClick={() => setShowAtlas(true)}
            className="flex items-center gap-2 px-3.5 py-1.5 rounded-xl bg-amber-500/10 border border-amber-500/30 text-amber-400 hover:bg-amber-500/20 transition-all font-serif text-xs font-semibold shadow-lg shadow-amber-950/20 cursor-pointer"
          >
            <MapIcon className="w-3.5 h-3.5" />
            <span>Đại Vực</span>
          </button>
        </div>
      </div>

      {/* Exploration Grid Canvas */}
      <div className="relative bg-zinc-950/90 border border-zinc-900 rounded-3xl p-2 sm:p-3 shadow-2xl overflow-hidden flex flex-col items-center justify-center">
        {gridData ? (
          <div
            className={`grid gap-px w-full ${maxContainerWidth} aspect-square transition-all duration-300`}
            style={{
              gridTemplateColumns: `repeat(${diameter}, minmax(0, 1fr))`,
              gridTemplateRows: `repeat(${diameter}, minmax(0, 1fr))`,
            }}
          >
            {gridArray.map(({ x, y, isOutOfBounds }) => {
              const cellData = cellMap.get(`${x},${y}`) || { x, y, cell_type: "Empty", content: "" };
              const isPlayerCell = x === player.x && y === player.y;
              const isMovable = !isOutOfBounds && !isPlayerCell && !isMemberNotLeader;
              const otherPlayersOnCell = isOutOfBounds
                ? []
                : gridData.players_on_map?.filter((p) => p.x === x && p.y === y) || [];
              const dropOnCell = isOutOfBounds ? null : gridData.ground_drops?.find((d) => d.x === x && d.y === y);
              const markersOnCell = isOutOfBounds ? [] : markersByCell.get(`${x},${y}`) || [];
              const primaryMarker = markersOnCell.find((marker) => marker.kind === "aperture_portal")
                || markersOnCell.find((marker) => marker.kind === "teleport")
                || markersOnCell.find((marker) => marker.kind === "sect")
                || markersOnCell[0];
              const portalMarker = markersOnCell.find((marker) => marker.kind === "aperture_portal");
              const resourcePoint = markersOnCell.find((marker) => marker.kind === "resource_point");
              const markerTitle = markersOnCell.map((marker) => marker.name).join(", ");

              return (
                <div
                  key={`${x},${y}`}
                  onClick={() => handleCellClick(x, y, isMovable, cellData, portalMarker || primaryMarker, resourcePoint?.id)}
                  className={`aspect-square rounded-md border flex flex-col justify-center items-center relative transition-all duration-200 ${
                    isPlayerCell
                      ? "bg-emerald-500/25 border-emerald-400 shadow-[0_0_16px_rgba(16,185,129,0.35)] z-20"
                    : isMovable
                      ? "bg-emerald-950/30 border-emerald-500/40 hover:bg-emerald-900/50 hover:border-emerald-300 cursor-pointer z-10 shadow-sm"
                      : isOutOfBounds
                      ? "bg-zinc-950 border-zinc-900/30 opacity-20 cursor-not-allowed"
                      : "bg-zinc-950/60 border-zinc-800/60 opacity-40 cursor-default"
                  }`}
                  title={
                    isOutOfBounds
                      ? "Hư vô (Ngoại biên bản đồ)"
                      : `Tọa độ: (${x}, ${y})${isPlayerCell ? " [Vị trí của bạn]" : ""}${
                          cellData.cell_type !== "Empty" ? ` | ${getEntityName(cellData.cell_type, cellData.content)}` : ""
                        }${markerTitle ? ` | ${markerTitle}` : ""}${otherPlayersOnCell.length > 0 ? ` | Cổ sư: ${otherPlayersOnCell.map((p) => p.username).join(", ")}` : ""}${dropOnCell ? " | Chiến lợi phẩm" : ""}`
                  }
                >
                  <div className="w-full h-full flex items-center justify-center gap-1 relative z-10 pb-3">
                    {isPlayerCell && (
                      <div className="w-7 h-7 sm:w-8 sm:h-8 rounded-full bg-emerald-500 flex items-center justify-center text-[10px] text-zinc-950 font-bold border-2 border-emerald-300 shadow-[0_0_12px_rgba(52,211,153,0.8)] overflow-hidden">
                        {player.avatar_url ? <img src={player.avatar_url} alt="" className="w-full h-full object-cover" /> : "Ta"}
                      </div>
                    )}
                    {otherPlayersOnCell.length > 0 && !isPlayerCell && (
                      <div
                        className="w-6 h-6 sm:w-7 sm:h-7 rounded-full bg-purple-600/80 flex items-center justify-center text-[9px] text-zinc-100 font-bold border border-purple-400 shadow-[0_0_10px_rgba(168,85,247,0.5)] overflow-hidden"
                        title={`Cổ sư: ${otherPlayersOnCell[0].username}`}
                      >
                        {otherPlayersOnCell[0].avatar_url ? (
                          <img src={otherPlayersOnCell[0].avatar_url} alt="" className="w-full h-full object-cover" />
                        ) : (
                          otherPlayersOnCell[0].username.slice(0, 2).toUpperCase()
                        )}
                      </div>
                    )}
                    {!isOutOfBounds && !isPlayerCell && getCellIcon(cellData.cell_type, cellData.content)}
                    {primaryMarker && (
                      <span className="absolute right-0.5 top-0.5 rounded-full bg-zinc-950/90 border border-zinc-600/70 p-0.5 shadow" title={markerTitle}>
                        {primaryMarker.kind === "sect" ? <ShieldAlert className="w-3.5 h-3.5 text-amber-300" />
                          : primaryMarker.kind === "teleport" ? <Waypoints className="w-3.5 h-3.5 text-sky-300" />
                            : primaryMarker.kind === "aperture_portal" ? <DoorOpen className="w-3.5 h-3.5 text-violet-300" />
                              : <Mountain className="w-3.5 h-3.5 text-lime-300" />}
                      </span>
                    )}
                    {markersOnCell.length > 1 && (
                      <span className="absolute left-0.5 top-0.5 rounded-full bg-zinc-950/90 px-1 text-[7px] leading-3 text-zinc-200 border border-zinc-600/70">
                        +{markersOnCell.length - 1}
                      </span>
                    )}
                    {dropOnCell && !isPlayerCell && (
                      <Package className="w-4 h-4 text-amber-400 animate-bounce drop-shadow" />
                    )}
                  </div>

                  {(cellData.cell_type !== "Empty" || primaryMarker) && !isOutOfBounds && !isPlayerCell && (
                    <span className="text-[6px] leading-tight font-mono text-zinc-400 absolute bottom-0.5 px-0.5 truncate max-w-full text-center">
                      {cellData.cell_type !== "Empty" ? getEntityName(cellData.cell_type, cellData.content) : primaryMarker.name}
                    </span>
                  )}
                </div>
              );
            })}
          </div>
        ) : (
          <div className="text-sm text-zinc-500 py-16 flex flex-col items-center gap-3">
            <Compass className="w-8 h-8 animate-spin text-zinc-600" />
            <span>Đang cảm ứng thiên địa, giải tỏa sương mù...</span>
          </div>
        )}
      </div>

      {/* Combat Warning Banner */}
      {player.active_combat && (
        <div className="bg-rose-950/40 border border-rose-900/60 rounded-2xl p-4 flex items-center gap-3 shadow-lg animate-pulse">
          <ShieldAlert className="w-6 h-6 text-rose-500 shrink-0" />
          <div className="text-sm font-bold text-rose-300 font-serif">
            Đang chạm trán dã thú/đối thủ! Chuyển tiếp vào trận đấu để ứng chiến.
          </div>
        </div>
      )}

      {/* Exploration Log */}
      {lastLog && (
        <div className="bg-zinc-950/80 border border-zinc-800 rounded-2xl p-4 flex items-start gap-3 shadow-md">
          <div className="w-8 h-8 rounded-xl bg-amber-500/10 border border-amber-500/20 flex items-center justify-center shrink-0 mt-0.5 text-amber-400">
            <Sparkles className="w-4 h-4" />
          </div>
          <div className="space-y-1">
            <span className="text-[10px] text-zinc-500 uppercase tracking-widest block font-mono font-semibold">Ghi chép du hành</span>
            <p className="text-xs text-zinc-300 font-mono leading-relaxed whitespace-pre-line">{lastLog}</p>
          </div>
        </div>
      )}

      {/* World Atlas Modal */}
      {showAtlas && (
        <WorldAtlasModal
          regions={regions}
          currentRegionId={player.current_region}
          x={player.x}
          y={player.y}
          onClose={() => setShowAtlas(false)}
        />
      )}
    </div>
  );
};
