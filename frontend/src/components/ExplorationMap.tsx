"use client";

import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Compass, Flame, Sparkles, User, DoorOpen, Landmark, ShieldAlert, Footprints } from "lucide-react";
import { useGameStore, Player } from "@/store/useGameStore";
import { useWorldStore } from "@/store/useWorldStore";
import { GamePanel } from "@/components/ui";
import { usePolling } from "@/lib/usePolling";
import { getWorldResourceGroup, WORLD_RESOURCE_ASSETS } from "@/lib/worldResourceAssets";

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
  cells: GridCell[];
  stats: { movement?: number; investigation?: number };
  players_on_map?: { id: string; username: string; avatar_url: string; x: number; y: number }[];
  world_width?: number;
  world_height?: number;
}

/**
 * Simple world map: the player's current viewport with click-to-move.
 * No region selector, no side panels. Used by WorldPage in "simple" map mode.
 */
export const ExplorationMap: React.FC<ExplorationMapProps> = ({ player }) => {
  const regions = useGameStore((state) => state.regions);
  const regionsLoaded = useGameStore((state) => state.regionsLoaded);
  const fetchRegions = useGameStore((state) => state.fetchRegions);
  const fetchGrid = useGameStore((state) => state.fetchGrid);
  const movePlayer = useGameStore((state) => state.movePlayer);
  const harvestVein = useWorldStore((state) => state.harvestVein);
  const addToast = useGameStore((state) => state.addToast);
  const actionLoading = useGameStore((state) => state.actionLoading);
  const party = useGameStore((state) => state.party);
  const guDictionary = useGameStore((state) => state.guDictionary);
  const itemDictionary = useGameStore((state) => state.itemDictionary);
  const monsterDictionary = useGameStore((state) => state.monsterDictionary);

  const [gridData, setGridData] = useState<GridData | null>(null);
  const [lastLog, setLastLog] = useState<string | null>(null);
  const lastMoveTimeRef = useRef(0);

  useEffect(() => {
    if (!regionsLoaded) fetchRegions();
  }, [regionsLoaded, fetchRegions]);

  const loadGrid = useCallback(async () => {
    const data = await fetchGrid();
    if (data && Array.isArray(data.cells)) {
      setGridData(data as GridData);
    }
  }, [fetchGrid]);

  useEffect(() => {
    void loadGrid();
  }, [player.current_region, player.x, player.y, loadGrid]);
  usePolling(loadGrid, 10000);

  const isMemberNotLeader = Boolean(player.party_id) && (party === null || party.leader_id !== player.id);
  const movementRange = gridData?.stats?.movement || 1;
  const regionName = regions.find((r) => r.id === player.current_region)?.name || "Thế Giới Cổ Sư";

  const handleCellClick = async (x: number, y: number, isMovable: boolean, cell?: GridCell) => {
    if (cell?.cell_type === "PrimevalVein"
      && Math.max(Math.abs(x - player.x), Math.abs(y - player.y)) <= 3) {
      await harvestVein(cell.content);
      await loadGrid();
      return;
    }
    if (!isMovable || actionLoading || player.active_combat) return;
    if (isMemberNotLeader) {
      addToast("Chỉ có Trưởng nhóm mới có quyền di chuyển trên bản đồ!", "error");
      return;
    }

    const now = Date.now();
    if (now - lastMoveTimeRef.current < 1000) return;
    lastMoveTimeRef.current = now;

    setLastLog(null);
    const res = await movePlayer(x, y);
    if (res) setLastLog(res.log);
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
        return <Flame className="w-5 h-5 text-rose-500 animate-pulse" />;
      case "WildGu":
        return <Sparkles className="w-5 h-5 text-amber-400 animate-bounce" />;
      case "PrimevalVein":
        return <Sparkles className="w-5 h-5 text-emerald-300 drop-shadow-[0_0_6px_rgba(52,211,153,0.8)]" />;
      case "PVP":
        return <User className="w-5 h-5 text-purple-400" />;
      case "EXIT_PORTAL":
      case "PORTAL_EXIT":
      case "ENTRY_PORTAL":
        return <DoorOpen className="w-5 h-5 text-violet-400" />;
      case "INHERITANCE_ALTAR":
        return <Landmark className="w-5 h-5 text-amber-400 animate-pulse drop-shadow-[0_0_8px_rgba(245,158,11,0.8)]" />;
      default:
        return null;
    }
  };

  // Cap the viewport to 5x5 so the grid stays readable on small screens.
  const visionRadius = Math.min(gridData?.vision_radius || 1, 2);
  const cols = 2 * visionRadius + 1;
  const worldW = gridData?.world_width || 30000;
  const worldH = gridData?.world_height || 20000;

  // Viewport lookup is bounded (≤ 5x5 cells), but build an O(1) key map instead of
  // scanning the full cells array per rendered cell.
  const gridCellLookup = useMemo(() => {
    const lookup = new Map<string, GridCell>();
    if (gridData?.cells) {
      for (const c of gridData.cells) {
        lookup.set(`${c.x},${c.y}`, c);
      }
    }
    return lookup;
  }, [gridData]);

  const gridCells: { x: number; y: number; isOutOfBounds: boolean; cellData: GridCell }[] = [];
  if (gridData) {
    for (let dy = -visionRadius; dy <= visionRadius; dy++) {
      for (let dx = -visionRadius; dx <= visionRadius; dx++) {
        const cx = player.x + dx;
        const cy = player.y + dy;
        const existingCell = gridCellLookup.get(`${cx},${cy}`);
        gridCells.push({
          x: cx,
          y: cy,
          isOutOfBounds: cx < 0 || cx >= worldW || cy < 0 || cy >= worldH,
          cellData: existingCell || { x: cx, y: cy, cell_type: "Empty", content: "" },
        });
      }
    }
  }

  return (
    <GamePanel variant="gold" className="rounded-3xl p-3 sm:p-6 w-full mx-auto space-y-4 text-left relative">
      <div className="flex items-center justify-between gap-2 border-b border-zinc-800 pb-3">
        <div className="flex items-center gap-2.5">
          <Compass className="w-5 h-5 text-amber-400 animate-spin" />
          <div>
            <h3 className="text-base font-bold text-zinc-100 font-serif leading-none">{regionName}</h3>
            <span className="text-[10px] text-zinc-500 font-mono mt-1 block">
              Tọa độ Không Khiếu: ({player.x}, {player.y})
            </span>
          </div>
        </div>
        <span className="text-[10px] text-zinc-500 font-mono flex items-center gap-1">
          <Footprints className="w-3.5 h-3.5 text-emerald-400" /> {movementRange} ô
        </span>
      </div>

      <div className="text-[10px] text-zinc-500 font-mono">
        * Nhấp vào một ô trong phạm vi thân pháp (<span className="text-emerald-400 font-bold">{movementRange} ô</span>) để di chuyển.
      </div>
      {isMemberNotLeader && (
        <div className="text-[10px] text-amber-500 font-mono">* Chỉ có Trưởng nhóm mới có thể di chuyển trên bản đồ.</div>
      )}

      <div className="flex justify-center items-center py-2">
        {gridData ? (
          <div
            className="grid gap-2 border border-zinc-900 bg-zinc-950/20 p-4 rounded-2xl shadow-inner max-w-md w-full"
            style={{ gridTemplateColumns: `repeat(${cols}, minmax(0, 1fr))` }}
          >
            {gridCells.map(({ x, y, isOutOfBounds, cellData }) => {
              const isPlayerCell = x === player.x && y === player.y;
              const distance = Math.max(Math.abs(x - player.x), Math.abs(y - player.y));
              const isMovable = !isOutOfBounds && distance > 0 && distance <= movementRange && !isMemberNotLeader;
              const otherPlayersOnCell = isOutOfBounds
                ? []
                : gridData.players_on_map?.filter((p) => p.x === x && p.y === y) || [];

              return (
                <div
                  key={`${x},${y}`}
                  onClick={() => handleCellClick(x, y, isMovable, cellData)}
                  className={`aspect-square rounded-xl border flex flex-col justify-center items-center relative transition-all duration-300 ${
                    isPlayerCell
                      ? "bg-emerald-500/10 border-emerald-500 shadow-[0_0_12px_rgba(16,185,129,0.2)] animate-pulse"
                      : isMovable
                      ? "bg-zinc-950/30 border-zinc-800 hover:bg-emerald-950/20 hover:border-emerald-500/30 cursor-pointer"
                      : isOutOfBounds
                      ? "bg-zinc-950/80 border-transparent opacity-20 cursor-not-allowed"
                      : "bg-zinc-950/50 border-zinc-950 opacity-40 cursor-not-allowed"
                  }`}
                  title={
                    isOutOfBounds
                      ? "Hư vô (Ngoại biên bản đồ)"
                      : `Tọa độ: (${x}, ${y})${isPlayerCell ? " [Vị trí của bạn]" : ""}${
                          cellData.cell_type !== "Empty" ? ` | ${getEntityName(cellData.cell_type, cellData.content)}` : ""
                        }${otherPlayersOnCell.length > 0 ? ` | Cổ sư: ${otherPlayersOnCell.map((p) => p.username).join(", ")}` : ""}`
                  }
                >
                  {!isOutOfBounds && <span className="text-[7px] text-zinc-700 font-mono absolute top-1.5">{x},{y}</span>}

                  <div className="flex gap-1 relative z-10">
                    {isPlayerCell && (
                      <div className="w-7 h-7 rounded-full bg-emerald-500 flex items-center justify-center text-[10px] text-zinc-950 font-bold border-2 border-emerald-300 shadow-[0_0_10px_rgba(52,211,153,0.6)] animate-bounce overflow-hidden">
                        {player.avatar_url ? <img src={player.avatar_url} alt="" className="w-full h-full object-cover" /> : "Ta"}
                      </div>
                    )}
                    {otherPlayersOnCell.length > 0 && (
                      <div
                        className="w-7 h-7 rounded-full bg-purple-500/80 flex items-center justify-center text-[10px] text-zinc-100 font-bold border-2 border-purple-300 shadow-[0_0_10px_rgba(139,92,246,0.6)] overflow-hidden"
                        title={`Cổ sư: ${otherPlayersOnCell[0].username}`}
                      >
                        {otherPlayersOnCell[0].avatar_url ? (
                          <img src={otherPlayersOnCell[0].avatar_url} alt="" className="w-full h-full object-cover" />
                        ) : (
                          otherPlayersOnCell[0].username.slice(0, 2).toUpperCase()
                        )}
                      </div>
                    )}
                    {!isOutOfBounds && getCellIcon(cellData.cell_type, cellData.content)}
                  </div>

                  {cellData.cell_type !== "Empty" && !isOutOfBounds && (
                    <span className="text-[7px] font-mono text-zinc-400 absolute bottom-1 px-1 truncate max-w-full text-center">
                      {getEntityName(cellData.cell_type, cellData.content)}
                    </span>
                  )}
                </div>
              );
            })}
          </div>
        ) : (
          <div className="text-xs text-zinc-600 py-12 italic">Đang đồng bộ địa hình trận địa...</div>
        )}
      </div>

      {player.active_combat && (
        <div className="bg-rose-950/20 border border-rose-900/40 rounded-2xl p-4 flex items-center gap-2.5">
          <ShieldAlert className="w-5 h-5 text-rose-500" />
          <div className="text-xs font-bold text-rose-400 font-serif">Đang chạm trán dã thú/đối thủ! Hãy xử lý trận chiến trước.</div>
        </div>
      )}

      {lastLog && (
        <div className="bg-zinc-950/70 border border-zinc-900 rounded-2xl p-4 flex items-start gap-3">
          <div className="w-7 h-7 rounded-lg bg-amber-500/10 flex items-center justify-center shrink-0 mt-0.5">
            <Sparkles className="w-4 h-4 text-amber-400" />
          </div>
          <div className="space-y-1">
            <span className="text-[9px] text-zinc-500 uppercase tracking-widest block font-mono">Ghi chép du hành</span>
            <p className="text-xs text-zinc-300 font-mono">{lastLog}</p>
          </div>
        </div>
      )}
    </GamePanel>
  );
};
