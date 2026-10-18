"use client";

import React, { useState, useEffect, useRef, useCallback } from "react";
import { guSpecFor, useGameStore, Player, Region } from "@/store/useGameStore";
import { Compass, Users, Map as MapIcon, ChevronLeft, ChevronRight, ChevronUp, ChevronDown, Flag, User, Swords, ArrowRight, Activity, ArrowLeftRight, Leaf, Sparkles, MapPin, ShieldAlert, Award, Footprints, Flame, HelpCircle, Crown, Home, Gift, Skull, Package, Shield, X, DoorOpen, Landmark } from "lucide-react";

import { LogConsole } from "./LogConsole";
import { SharedSpatialRealmControls } from "./SharedSpatialRealmControls";
import { SpatialRealmHUD } from "./SpatialRealmHUD";
import { InheritanceAltarModal } from "./InheritanceAltarModal";
import { GamePanel, GameButton } from "@/components/ui";
import { usePolling } from "@/lib/usePolling";
import { getWorldResourceGroup, WORLD_RESOURCE_ASSETS } from "@/lib/worldResourceAssets";
// UI migrated to shared components

interface ExplorationMapProps {
  player: Player;
  actionLoading: boolean;
  onExplore: (nodeId: string) => Promise<{ player: ApiResponseData; log: string; combat_triggered: boolean; skip_result?: ApiResponseData } | null>;
  onBack?: () => void;
}

// Dynamic mapping based on array index instead of hardcoding
const getColorClass = (index: number) => {
  const colors = [
    "text-emerald-400 border-emerald-500/20 hover:border-emerald-500/50 bg-emerald-950/10",
    "text-blue-400 border-blue-500/20 hover:border-blue-500/50 bg-blue-950/10",
    "text-purple-400 border-purple-500/20 hover:border-purple-500/50 bg-purple-950/10",
    "text-amber-400 border-amber-500/20 hover:border-amber-500/50 bg-amber-950/10",
    "text-rose-400 border-rose-500/20 hover:border-rose-500/50 bg-rose-950/10"
  ];
  return colors[index % colors.length];
};

const getBgGlow = (index: number) => {
  const glows = [
    "from-emerald-500/5 to-transparent",
    "from-blue-500/5 to-transparent",
    "from-purple-500/5 to-transparent",
    "from-amber-500/5 to-transparent",
    "from-rose-500/5 to-transparent"
  ];
  return glows[index % glows.length];
};

const MAIN_WORLD_REGION_IDS = new Set([
  "dai_vuc_nam_cuong",
  "dai_vuc_trung_chau",
  "dai_vuc_bac_nguyen",
  "dai_vuc_tay_mac",
  "dai_vuc_dong_hai",
]);

const regionRestrictionSummary = (region?: Region): string[] => {
  const rules = region?.effective_rules || region?.rules;
  if (!rules) return [];
  const summary: string[] = [];
  if ((rules.rank?.min || 0) > 0) summary.push(`Rank tối thiểu ${rules.rank?.min}`);
  if ((rules.rank?.max || 0) > 0) summary.push(`Rank tối đa ${rules.rank?.max}`);
  if (rules.paths?.allow?.length) summary.push(`Chỉ Cổ đạo: ${rules.paths.allow.join(", ")}`);
  if (rules.paths?.deny?.length) summary.push(`Cấm Cổ đạo: ${rules.paths.deny.join(", ")}`);
  if ((rules.gu?.max_rank || 0) > 0) summary.push(`Cổ tối đa rank ${rules.gu?.max_rank}`);
  if (rules.gu?.deny_ids?.length) summary.push(`Cấm ${rules.gu.deny_ids.length} loại Cổ`);
  if (rules.items?.deny_ids?.length) summary.push(`Cấm ${rules.items.deny_ids.length} vật phẩm`);
  const actionNames: Record<string, string> = {
    pvp: "PvP", teleport: "dịch chuyển", gu_use: "dùng Cổ", item_use: "dùng vật phẩm",
    combat: "giao chiến", monster_combat: "đánh quái", resource_gathering: "thu thập tài nguyên", gu_capture: "bắt Cổ",
  };
  if (rules.actions?.deny?.length) {
    summary.push(`Cấm: ${rules.actions.deny.map((action) => actionNames[action] || action).join(", ")}`);
  }
  return summary;
};

export const ExplorationMap: React.FC<ExplorationMapProps> = ({
  player,
  actionLoading,
  onExplore,
  onBack,
}) => {
  const [lastLog, setLastLog] = useState<string | null>(null);
  const [activeNode, setActiveNode] = useState<string | null>(null);
  const lastMoveTimeRef = useRef(0);
  const [showRegionsList, setShowRegionsList] = useState(false);
  const [gridData, setGridData] = useState<{
    x: number;
    y: number;
    vision_radius: number;
    cells: ApiResponseData[];
    stats: ApiResponseData;
    players_on_map?: ApiResponseData[];
    world_width?: number;
    world_height?: number;
  } | null>(null);

  const { fetchGrid, movePlayer, party, partyInvitations, createParty, fetchPartyStatus, fetchPartyInvitations, invitePlayer, respondPartyInvitation, leaveParty, retreat, fetchPlayerProfile, showConfirm, addToast, groundDrops, fetchGroundDrops, lootGroundDrop, sectMap, fetchSectsMap, attackSect, sectMembers, fetchSectMembers, mySectStatus } = useGameStore();
  const [inviteUsername, setInviteUsername] = useState("");
  const [inviteError, setInviteError] = useState<string | null>(null);
  const [partyLoading, setPartyLoading] = useState(false);
  const [attackTargetSect, setAttackTargetSect] = useState<ApiResponseData | null>(null);
  const [attackPlayerId, setAttackPlayerId] = useState("");
  const logs = useGameStore((state) => state.logs);
  const guDictionary = useGameStore((state) => state.guDictionary);
  const itemDictionary = useGameStore((state) => state.itemDictionary);
  const monsterDictionary = useGameStore((state) => state.monsterDictionary);
  const fetchAltarInfo = useGameStore((state) => state.fetchAltarInfo);

  const handleExitMap = async () => {
    if (isExploringActiveRegion) {
      if (await showConfirm("Đạo hữu có chắc chắn muốn rời khỏi bản đồ khám phá và trở về Trấn?")) {
        await retreat();
        if (onBack) {
          onBack();
        }
      }
    } else {
      if (onBack) {
        onBack();
      }
    }
  };

  const regions = useGameStore((state) => state.regions);
  const regionsLoaded = useGameStore((state) => state.regionsLoaded);
  const fetchRegions = useGameStore((state) => state.fetchRegions);

  useEffect(() => {
    if (!regionsLoaded) {
      fetchRegions();
    }
  }, [regionsLoaded, fetchRegions]);

  const isExploringActiveRegion = player.current_region !== "hub_nam_cuong" && player.current_region !== "Lobby" && regions.find((r) => r.id === player.current_region)?.zone_type !== "HUB";
  const isPartyMemberNotLeader = party !== null && party.leader_id !== player.id;

  const loadGrid = async () => {
    if (!isExploringActiveRegion) return;
    const data = await fetchGrid();
    if (data && Array.isArray(data.cells)) {
      setGridData({ ...data, cells: data.cells });
    }
    fetchGroundDrops({
      x: data?.x ?? player.x,
      y: data?.y ?? player.y,
      radius: data?.vision_radius ?? 25,
    });
    fetchSectsMap();
  };

  useEffect(() => {
    fetchPartyStatus();
    fetchPartyInvitations();
  }, []);

  useEffect(() => {
    loadGrid();
  }, [player.current_region, player.x, player.y, isExploringActiveRegion]);
  usePolling(loadGrid, 10000, isExploringActiveRegion);

  const handleCreateParty = async () => {
    setPartyLoading(true);
    try {
      await createParty();
    } finally {
      setPartyLoading(false);
    }
  };

  const handleInvitePlayer = async () => {
    if (!inviteUsername.trim()) return;
    setPartyLoading(true);
    setInviteError(null);
    try {
      const ok = await invitePlayer(inviteUsername.trim());
      if (ok) {
        setInviteUsername("");
      } else {
        setInviteError("Mời thất bại. Hãy đảm bảo đối hiệu đúng và cự ly / tổ đội chưa có.");
      }
    } finally {
      setPartyLoading(false);
    }
  };

  const handleLeaveParty = async () => {
    setPartyLoading(true);
    try {
      await leaveParty();
    } finally {
      setPartyLoading(false);
    }
  };

  const handleNodeClick = async (nodeId: string) => {
    if (actionLoading || player.active_combat) return;

    setActiveNode(nodeId);
    setLastLog(null);

    try {
      const res = await onExplore(nodeId);
      if (res) {
        setLastLog(res.log);
        setShowRegionsList(false);
      }
    } finally {
      setActiveNode(null);
    }
  };

  const handleGridCellClick = async (cx: number, cy: number) => {
    if (actionLoading || player.active_combat || isPartyMemberNotLeader) return;

    const now = Date.now();
    if (now - lastMoveTimeRef.current < 1000) return;

    // Check distance range
    const dx = Math.abs(cx - player.x);
    const dy = Math.abs(cy - player.y);
    const dist = Math.max(dx, dy);

    const movementLimit = gridData?.stats?.movement || 1;

    if (dist > movementLimit) return; // Too far

    lastMoveTimeRef.current = now;
    setLastLog(null);
    const res = await movePlayer(cx, cy);
    if (res) {
      setLastLog(res.log);
    }
  };

  const getEntityName = (type: string, content: string) => {
    if (type === "EXIT_PORTAL" || type === "PORTAL_EXIT") return "Cổng trở về";
    if (type === "ENTRY_PORTAL") {
      const realmName = content.replace(/^spatial_realm_/, "").replace(/_/g, " ");
      return `Cổng vào bí cảnh: ${realmName}`;
    }
    if (type === "INHERITANCE_ALTAR") {
      const altarName = content.replace(/^altar_/, "").replace(/_/g, " ");
      return `Đài Truyền Thừa: ${altarName}`;
    }
    if (type === "Monster" || type === "Boss") {
      return monsterDictionary[content]?.name || content.split(" ")[0];
    }
    if (type === "Resource") {
      return itemDictionary[content]?.name || content.replace(/_/g, " ").replace(/\b\w/g, (c: string) => c.toUpperCase());
    }
    if (type === "WildGu") {
      return guDictionary[content]?.name || content;
    }
    return content;
  };

  // Helper to format map cell content icon
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
      case "PVP":
        return <User className="w-5 h-5 text-purple-400" />;
      case "EXIT_PORTAL":
      case "PORTAL_EXIT":
        return <DoorOpen className="w-5 h-5 text-violet-400" />;
      case "ENTRY_PORTAL":
        return <DoorOpen className="w-5 h-5 text-amber-300" />;
      case "INHERITANCE_ALTAR":
        return <Landmark className="w-5 h-5 text-amber-400 animate-pulse drop-shadow-[0_0_8px_rgba(245,158,11,0.8)]" />;
      default:
        return null;
    }
  };

  const getMapTitle = (id: string): string => {
    const node = regions.find((n) => n.id === id);
    return node ? node.name : "Thế Giới Cổ Sư";
  };

  if (!isExploringActiveRegion || showRegionsList) {
    return (
      <div className="glass-panel-gold rounded-3xl p-6 w-full mx-auto space-y-6 text-left">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
          <div className="flex items-center gap-2">
            <Compass className="w-5 h-5 text-amber-500 animate-spin" />
            <h3 className="text-lg font-bold text-zinc-100 font-serif">
              Bản Đồ Thế Giới - Lục Đại Vực
            </h3>
          </div>
          <div className="flex items-center gap-4">
            {isExploringActiveRegion && (
              <button
                onClick={() => setShowRegionsList(false)}
                className="py-1 px-3 border border-zinc-800 rounded-xl text-[10px] text-zinc-400 hover:bg-zinc-900 cursor-pointer font-mono"
              >
                Về Bản Đồ Hiện Tại
              </button>
            )}
             {onBack && (
              <button
                type="button"
                onClick={handleExitMap}
                disabled={actionLoading}
                className={`py-1 px-3 border rounded-xl text-[10px] cursor-pointer font-mono font-bold transition-all ${
                  isExploringActiveRegion
                    ? "bg-emerald-950/40 text-emerald-400 border-emerald-900/60 hover:bg-emerald-950/60 hover:border-emerald-700 shadow-[0_0_10px_rgba(16,185,129,0.1)]"
                    : "border-zinc-800 text-zinc-400 hover:bg-zinc-900"
                }`}
              >
                {isExploringActiveRegion ? "Thoát & Về Trấn" : "Trở Về"}
              </button>
            )}
          </div>
        </div>
        {/* World Overview: the five core regions; the two heavenly realms stay out of this selector. */}
        {(() => {
          const largeZones = regions
            .filter((r) => r.zone_type === "NGU_VUC" && MAIN_WORLD_REGION_IDS.has(r.id))
            .sort((a, b) => (a.min_y || 0) - (b.min_y || 0) || (a.min_x || 0) - (b.min_x || 0));
          const namedZones = regions.filter((r) => MAIN_WORLD_REGION_IDS.has(r.parent_id || "") && r.zone_type === "KHU_VUC");
          return (
            <div className="space-y-6">
              <div>
                <h3 className="text-sm font-bold text-zinc-100 font-serif">5 Đại Vực</h3>
                <p className="mt-1 text-[10px] text-zinc-500">Chọn một khu vực bên trong đại vực để bắt đầu khám phá.</p>
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
                {largeZones.map((node, index) => {
                  const children = namedZones.filter((n) => n.parent_id === node.id);
                  return (
                    <div
                      key={node.id}
                      className={`border rounded-2xl p-4 flex flex-col gap-4 min-h-60 transition-all bg-gradient-to-b ${getBgGlow(index)} to-zinc-950 relative overflow-hidden group ${
                        getColorClass(index)
                      } ${!!player.active_combat ? "opacity-60" : ""}`}
                    >
                      <div className="space-y-2">
                        <div className="flex items-start justify-between">
                          <span className="text-[9px] text-zinc-500 uppercase tracking-widest font-mono">
                            Rank {node.min_rank} - {node.max_rank}
                          </span>
                          <MapPin className="w-3.5 h-3.5 opacity-60 group-hover:scale-110 transition-transform" />
                        </div>
                        <h4 className="text-base font-bold text-zinc-100 font-serif group-hover:text-amber-400 transition-colors">
                          {node.name}
                        </h4>
                        <div className="text-[10px] text-zinc-400 font-serif leading-relaxed line-clamp-3">
                          {node.description}
                        </div>
                        {children.length > 0 && (
                          <div className="grid grid-cols-1 gap-1.5 pt-1">
                            {children.map((c) => {
                              const restrictions = regionRestrictionSummary(c);
                              return (
                                <div key={c.id}>
                                  <button
                                    type="button"
                                    disabled={actionLoading || !!player.active_combat}
                                    onClick={() => void handleNodeClick(c.id)}
                                    className="flex w-full items-center justify-between gap-2 rounded-lg border border-zinc-800/80 bg-zinc-950/50 px-2.5 py-2 text-left text-[10px] text-zinc-300 transition hover:border-emerald-500/40 hover:bg-zinc-900 disabled:cursor-not-allowed disabled:opacity-50"
                                  >
                                    <span className="flex items-center gap-2">
                                      {activeNode === c.id ? <Compass className="h-3 w-3 animate-spin text-amber-400" /> : <Footprints className="h-3 w-3 text-emerald-500" />}
                                      {c.name}
                                    </span>
                                    <span className="shrink-0 text-[8px] text-zinc-600">Rank {c.min_rank}–{c.max_rank}</span>
                                  </button>
                                  {restrictions.length > 0 && <p className="px-2.5 pt-1 text-[9px] leading-relaxed text-amber-400/80">Cấm chế: {restrictions.join(" · ")}</p>}
                                </div>
                              );
                            })}
                          </div>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>

              {isExploringActiveRegion && (
                <button
                  type="button"
                  disabled={actionLoading || !!player.active_combat}
                  onClick={() => handleNodeClick("hub_nam_cuong")}
                  className="w-full py-2.5 px-3 text-xs font-bold rounded-xl flex items-center justify-center gap-1.5 transition-all cursor-pointer bg-emerald-950/40 text-emerald-400 border border-emerald-900/60 hover:bg-emerald-950/60 hover:border-emerald-700"
                >
                  <Home className="w-3.5 h-3.5" />
                  Trở Về Khu An Toàn
                </button>
              )}
            </div>
          );
        })()}

        {/* last log display */}
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
      </div>
    );
  }

  // 100x100 Coordinates Viewport Grid rendering
  // Cap the UI vision radius to 2 (corresponding to a 5x5 grid) to prevent UI breakage
  const visionRadius = Math.min(gridData?.vision_radius || 1, 2);
  const cols = 2 * visionRadius + 1;

  // Generate a complete 2D array grid centered on the player (size cols x cols)
  // to ensure perfect alignment even at map boundaries (where out-of-bounds cells are returned as empty placeholder cells).
  interface GridCellItem {
    x: number;
    y: number;
    isOutOfBounds: boolean;
    cellData: {
      x: number;
      y: number;
      cell_type: string;
      content: string;
    };
  }

  const worldW = gridData?.world_width || 30000;
  const worldH = gridData?.world_height || 20000;
  const gridCells: GridCellItem[] = [];
  if (gridData) {
    for (let dy = -visionRadius; dy <= visionRadius; dy++) {
      for (let dx = -visionRadius; dx <= visionRadius; dx++) {
        const cx = player.x + dx;
        const cy = player.y + dy;
        const outOfBounds = cx < 0 || cx >= worldW || cy < 0 || cy >= worldH;

        const existingCell = gridData.cells.find((c: { x: number; y: number; cell_type: string; content: string }) => c.x === cx && c.y === cy);
        gridCells.push({
          x: cx,
          y: cy,
          isOutOfBounds: outOfBounds,
          cellData: existingCell || {
            x: cx,
            y: cy,
            cell_type: "Empty",
            content: "",
          },
        });
      }
    }
  }

  const movementRange = gridData?.stats?.movement || 1;

  return (
    <GamePanel variant="gold" className="rounded-3xl p-3 sm:p-6 w-full mx-auto space-y-6 text-left relative">
      {/* Grid Map Header */}
      <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
        <div className="flex items-center gap-2.5">
          <Compass className="w-5 h-5 text-amber-400 animate-spin" />
          <div>
            <h3 className="text-base font-bold text-zinc-100 font-serif leading-none">
              {getMapTitle(player.current_region)}
            </h3>
            <span className="text-[10px] text-zinc-500 font-mono mt-1 block">
              Tọa độ Không Khiếu: ({player.x}, {player.y})
            </span>
            {regionRestrictionSummary(regions.find((region) => region.id === player.current_region)).length > 0 && (
              <span className="text-[9px] text-amber-400/80 mt-1 block">
                Cấm chế: {regionRestrictionSummary(regions.find((region) => region.id === player.current_region)).join(" · ")}
              </span>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          <GameButton
            onClick={() => setShowRegionsList(true)}
            variant="secondary"
            className="py-1.5 px-3 text-[10px] font-mono"
          >
            Đổi Bản Đồ
          </GameButton>
          {onBack && (
            <GameButton
              onClick={handleExitMap}
              variant="success"
              className="py-1.5 px-3 text-[10px] font-mono font-bold"
            >
              Thoát & Về Trấn
            </GameButton>
          )}
        </div>
      </div>

      <SharedSpatialRealmControls enabled={isExploringActiveRegion} currentRegion={player.current_region} x={player.x} y={player.y} />
      <SpatialRealmHUD />

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        {/* Left Column: Map Grid & Logs */}
        <div className="lg:col-span-8 space-y-6">
          <div className="text-[10px] text-zinc-500 font-mono max-w-xl">
            * Nhấp vào một ô trong phạm vi thân pháp (<span className="text-emerald-400 font-bold">{movementRange} ô</span>) để di chuyển.
          </div>
          {isPartyMemberNotLeader && (
            <div className="text-[10px] text-amber-500 font-mono max-w-xl">
              * Chỉ có Trưởng nhóm mới có thể di chuyển trên bản đồ.
            </div>
          )}

          {/* Grid Canvas */}
          <div className="flex justify-center items-center py-2">
            {gridData ? (
              <div
                className="grid gap-2 border border-zinc-900 bg-zinc-950/20 p-4 rounded-2xl shadow-inner max-w-md w-full"
                style={{
                  gridTemplateColumns: `repeat(${cols}, minmax(0, 1fr))`,
                }}
              >
                {gridCells.map(({ x, y, isOutOfBounds, cellData }) => {
                  const isPlayerCell = x === player.x && y === player.y;

                  const dx = Math.abs(x - player.x);
                  const dy = Math.abs(y - player.y);
                  const distance = Math.max(dx, dy);

                  const isMovable = !isOutOfBounds && distance > 0 && distance <= movementRange && !isPartyMemberNotLeader;
                  const hasEntity = cellData.cell_type !== "Empty";

                  // Other players on this cell
                  const otherPlayersOnCell = isOutOfBounds
                    ? []
                    : gridData?.players_on_map?.filter(
                        (p: { x: number; y: number; id: string; username: string; avatar_url: string }) => p.x === x && p.y === y
                      ) || [];

                  const cellDrops = isOutOfBounds ? [] : groundDrops.filter((d) => d.x === x && d.y === y);
                  const sectOnCell = isOutOfBounds ? null : sectMap.find((s: ApiResponseData) => s.region === player.current_region && s.map_x === x && s.map_y === y);

                  return (
                    <div
                      key={`${x},${y}`}
                      onClick={() => {
                        if (cellData.cell_type === "INHERITANCE_ALTAR") {
                          fetchAltarInfo(cellData.content);
                          return;
                        }
                        if (isMovable) handleGridCellClick(x, y);
                      }}
                      className={`aspect-square rounded-xl border flex flex-col justify-center items-center relative transition-all duration-300 ${
                        isPlayerCell
                          ? "bg-emerald-500/10 border-emerald-500 shadow-[0_0_12px_rgba(16,185,129,0.2)] animate-pulse"
                          : cellData.cell_type === "INHERITANCE_ALTAR"
                          ? "bg-amber-950/40 border-amber-500/60 shadow-[0_0_15px_rgba(245,158,11,0.3)] hover:scale-105 cursor-pointer"
                          : isMovable
                          ? "bg-zinc-950/30 border-zinc-800 hover:bg-emerald-950/20 hover:border-emerald-500/30 cursor-pointer"
                          : isOutOfBounds
                          ? "bg-zinc-950/80 border-transparent opacity-20 cursor-not-allowed"
                          : "bg-zinc-950/50 border-zinc-950 opacity-40 cursor-not-allowed"
                      }`}
                      title={
                        isOutOfBounds
                          ? "Hư vô (Ngoại biên bản đồ)"
                          : `Tọa độ: (${x}, ${y}) ${
                              isPlayerCell ? "[Vị trí của bạn]" : ""
                            }${
                              sectOnCell ? ` | Tông Môn: ${sectOnCell.name} (Cấp ${sectOnCell.level})` : ""
                            }${
                              cellData.cell_type !== "Empty"
                                ? ` | ${
                                    cellData.cell_type === "Monster" || cellData.cell_type === "Boss"
                                      ? "Quái vật"
                                      : cellData.cell_type === "Resource"
                                      ? "Linh tài"
                                      : cellData.cell_type === "WildGu"
                                      ? "Cổ trùng hoang dã"
                                      : cellData.cell_type
                                  }: ${getEntityName(cellData.cell_type, cellData.content)}`
                                : ""
                            }${
                              cellDrops.length > 0
                                ? ` | Chiến lợi phẩm rơi từ: ${cellDrops.map((d) => d.loser_name).join(", ")}`
                                : ""
                            }${
                              otherPlayersOnCell.length > 0
                                ? ` | Cổ sư: ${otherPlayersOnCell.map((p: { username: string }) => p.username).join(", ")}`
                                : ""
                            }`
                      }
                    >
                      {/* Coordinates indicator overlay */}
                      {!isOutOfBounds && (
                        <span className="text-[7px] text-zinc-700 font-mono absolute top-1.5">
                          {x},{y}
                        </span>
                      )}

                      {/* Cell Entity View */}
                      <div className="flex gap-1 relative z-10">
                        {isPlayerCell && (
                          <div className="w-7 h-7 rounded-full bg-emerald-500 flex items-center justify-center text-[10px] text-zinc-950 font-bold border-2 border-emerald-300 shadow-[0_0_10px_rgba(52,211,153,0.6)] animate-bounce overflow-hidden">
                            {player.avatar_url ? (
                              <img src={player.avatar_url} alt="" className="w-full h-full object-cover" />
                            ) : (
                              "Ta"
                            )}
                          </div>
                        )}
                        {sectOnCell && (
                          <div
                            onClick={(e) => {
                              e.stopPropagation();
                              setAttackTargetSect(sectOnCell);
                              setAttackPlayerId(player.id);
                              fetchSectMembers();
                            }}
                            className="w-7 h-7 rounded-xl bg-amber-950/90 flex items-center justify-center text-[12px] font-bold border border-amber-500/60 shadow-[0_0_10px_rgba(245,158,11,0.5)] cursor-pointer hover:scale-110 transition-transform overflow-hidden"
                            title={`Tông Môn: ${sectOnCell.name} (Cấp ${sectOnCell.level}) - Tọa độ (${sectOnCell.map_x}, ${sectOnCell.map_y})\nBấm để Khiêu Chiến Tông Môn!`}
                          >
                            🏯
                          </div>
                        )}
                        {otherPlayersOnCell.length > 0 && (
                          <div
                            onClick={async (e) => {
                              e.stopPropagation();
                              if (await showConfirm(`Đạo hữu muốn khiêu chiến (PVP) với ${otherPlayersOnCell[0].username}?\n\nChọn [OK] để Tấn Công.\nChọn [Hủy] để Trinh Thám.`)) {
                                movePlayer(x, y, otherPlayersOnCell[0].id);
                              } else {
                                fetchPlayerProfile(otherPlayersOnCell[0].username);
                              }
                            }}
                            className="w-7 h-7 rounded-full bg-purple-500/80 flex items-center justify-center text-[10px] text-zinc-100 font-bold border-2 border-purple-300 shadow-[0_0_10px_rgba(139,92,246,0.6)] animate-pulse cursor-pointer hover:scale-110 transition-transform overflow-hidden"
                            title={`Bấm để Tấn Công / Trinh Thám cổ sư ${otherPlayersOnCell[0].username}`}
                          >
                            {otherPlayersOnCell[0].avatar_url ? (
                              <img src={otherPlayersOnCell[0].avatar_url} alt="" className="w-full h-full object-cover" />
                            ) : (
                              otherPlayersOnCell[0].username.slice(0, 2).toUpperCase()
                            )}
                          </div>
                        )}
                        {!isOutOfBounds && cellDrops.length > 0 && (
                          <span title={`Túi đồ rơi từ ${cellDrops.map((d) => d.loser_name).join(", ")}`}>
                            <Gift className="w-5 h-5 text-amber-400 animate-bounce drop-shadow-[0_0_8px_rgba(251,191,36,0.8)]" />
                          </span>
                        )}
                        {!isOutOfBounds && getCellIcon(cellData.cell_type, cellData.content)}
                      </div>

                      {/* Tiny text label */}
                      {hasEntity && !isOutOfBounds && (
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

          {/* Exploration Log Overlay */}


          {player.active_combat && (
            <div className="bg-rose-950/20 border border-rose-900/40 rounded-2xl p-4 flex items-center justify-between animate-pulse">
              <div className="flex items-center gap-2.5">
                <ShieldAlert className="w-5 h-5 text-rose-500" />
                <div className="text-left">
                  <div className="text-xs font-bold text-rose-400 font-serif">Đang chạm trán dã thú/đối thủ!</div>
                  <div className="text-[10px] text-zinc-500">
                    Hãy trở về Căn cứ để kích hoạt sát chiêu tiêu diệt kẻ thù.
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Right Column: Party Manager & Ground Drops */}
        <div className="lg:col-span-4 space-y-6">
          {/* Ground Drops Panel */}
          {(() => {
            const currentTileDrops = groundDrops.filter((d) => d.x === player.x && d.y === player.y);
            return (
              <div className="bg-zinc-950/40 border border-amber-900/40 rounded-3xl p-5 space-y-4">
                <div className="flex items-center justify-between border-b border-zinc-900 pb-2">
                  <h4 className="font-serif font-bold text-sm text-amber-400 flex items-center gap-1.5">
                    <Gift className="w-4 h-4 text-amber-400" /> CHIẾN LỢI PHẨM RƠI ({currentTileDrops.length})
                  </h4>
                </div>

                {currentTileDrops.length === 0 ? (
                  <div className="text-[11px] text-zinc-500 italic py-2 text-center">
                    Không có chiến lợi phẩm nào tại vị trí ({player.x}, {player.y}).
                  </div>
                ) : (
                  <div className="space-y-3 max-h-[350px] overflow-y-auto pr-1">
                    {currentTileDrops.map((drop) => {
                      const data = drop.loot_data || {};
                      const primevalStones = data.primeval_stones ?? 0;
                      const immortalStones = data.immortal_stones ?? 0;
                      return (
                        <div key={drop.id} className="bg-zinc-950/80 border border-amber-500/30 rounded-2xl p-3.5 space-y-2 text-left shadow-lg">
                          <div className="flex items-center justify-between">
                            <span className="text-xs font-bold text-amber-300 font-serif flex items-center gap-1">
                              💀 Hài Cốt: {drop.loser_name}
                            </span>
                            {drop.is_protected ? (
                              <span className="text-[9px] text-amber-400 bg-amber-950/50 border border-amber-800/50 px-2 py-0.5 rounded font-mono">
                                🔒 Khóa Killer
                              </span>
                            ) : (
                              <span className="text-[9px] text-emerald-400 bg-emerald-950/50 border border-emerald-800/50 px-2 py-0.5 rounded font-mono">
                                🔓 Tự do nhặt
                              </span>
                            )}
                          </div>

                          {drop.killer_name && (
                            <div className="text-[10px] text-zinc-400 font-mono">
                              Hạ gục bởi: <span className="text-rose-400 font-semibold">{drop.killer_name}</span>
                            </div>
                          )}

                          {/* GroundLootData Content Display */}
                          <div className="bg-zinc-900/80 rounded-xl p-2.5 space-y-1 text-[11px] font-mono border border-zinc-800">
                            {primevalStones > 0 && (
                              <div className="text-emerald-400 font-semibold">❖ +{primevalStones.toLocaleString()} Nguyên Thạch</div>
                            )}
                            {immortalStones > 0 && (
                              <div className="text-purple-400 font-semibold">❖ +{immortalStones.toLocaleString()} Tiên Nguyên Thạch</div>
                            )}
                            {data.items && data.items.map((it: ApiResponseData, idx: number) => (
                              <div key={idx} className="text-amber-300">
                                📦 {it.quantity}x {itemDictionary[it.item_id]?.name || it.item_id}
                              </div>
                            ))}
                            {data.gus && data.gus.map((g: ApiResponseData, idx: number) => (
                              <div key={idx} className="text-cyan-300">
                                🐛 Cổ trùng: {guSpecFor(g.gu_id, g.rank)?.name || g.gu_id} (Rank {g.rank})
                              </div>
                            ))}
                            {data.recipes && data.recipes.map((r: string, idx: number) => (
                              <div key={idx} className="text-indigo-300">
                                📜 Bí phương: {guDictionary[r]?.name || r}
                              </div>
                            ))}
                          </div>

                          <button
                            type="button"
                            onClick={async () => {
                              await lootGroundDrop(drop.id);
                            }}
                            disabled={actionLoading || drop.is_protected}
                            className={`w-full py-2 rounded-xl text-xs font-bold font-serif transition-all cursor-pointer ${
                              drop.is_protected
                                ? "bg-zinc-900 border border-zinc-800 text-zinc-500 cursor-not-allowed"
                                : "bg-amber-950/40 border border-amber-500/40 text-amber-300 hover:bg-amber-950/70 hover:border-amber-400 shadow-[0_0_10px_rgba(245,158,11,0.1)] active:scale-[0.98]"
                            }`}
                          >
                            {drop.is_protected ? `Đang Bảo Hộ Cho ${drop.killer_name || "Killer"}` : "Nhặt Tất Cả Chiến Lợi Phẩm"}
                          </button>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            );
          })()}
          <div className="bg-zinc-950/40 border border-zinc-900 rounded-3xl p-5 space-y-4">
            <div className="flex items-center justify-between border-b border-zinc-900 pb-2">
              <h4 className="font-serif font-bold text-sm text-amber-400 flex items-center gap-1.5">
                <Users className="w-4 h-4" /> TỔ ĐỘI CHIẾN HỮU
              </h4>
              {party && (
                <span className="text-[9px] font-mono text-zinc-500 uppercase tracking-widest">
                  {party.members.length}/4 Cổ sư
                </span>
              )}
            </div>

            {!party ? (
              <div className="text-center py-4 space-y-3">
                <p className="text-[11px] text-zinc-400 leading-normal">
                  Đạo hữu chưa gia nhập tổ đội. Lập tổ đội để đi chung bản đồ và cùng chia sẻ Nguyên Thạch & Vật phẩm rơi ra khi chiến thắng.
                </p>
                <button
                  type="button"
                  onClick={handleCreateParty}
                  disabled={partyLoading}
                  className="w-full py-2 bg-emerald-950/30 border border-emerald-500/30 text-emerald-400 hover:bg-emerald-950/50 font-bold rounded-xl text-xs transition-all active:scale-[0.98] cursor-pointer"
                >
                  {partyLoading ? "Đang lập tổ..." : "Lập Tổ Đội Mới"}
                </button>

                {partyInvitations && partyInvitations.length > 0 && (
                  <div className="pt-4 border-t border-zinc-900/60 text-left space-y-2">
                    <span className="text-[10px] text-zinc-500 uppercase tracking-wider font-mono block">
                      Lời mời vào tổ đội ({partyInvitations.length})
                    </span>
                    <div className="space-y-1.5 max-h-[150px] overflow-y-auto pr-1">
                      {partyInvitations.map((inv) => (
                        <div
                          key={inv.id}
                          className="flex items-center justify-between bg-zinc-950/60 border border-zinc-900 rounded-lg p-2.5 text-xs"
                        >
                          <div className="flex items-center gap-2">
                            <div className="w-5 h-5 rounded-full bg-zinc-800 text-zinc-300 flex items-center justify-center font-bold text-[8px] overflow-hidden shrink-0">
                              {inv.inviter_avatar ? (
                                <img src={inv.inviter_avatar} alt="" className="w-full h-full object-cover" />
                              ) : (
                                inv.inviter_name.slice(0, 2).toUpperCase()
                              )}
                            </div>
                            <span className="font-semibold text-zinc-300">
                              {inv.inviter_name}
                            </span>
                          </div>
                          <div className="flex gap-1.5 shrink-0">
                            <button
                              type="button"
                              onClick={() => respondPartyInvitation(inv.id, "accept")}
                              disabled={partyLoading}
                              className="px-2 py-1 bg-emerald-950/30 border border-emerald-500/30 text-emerald-400 font-bold rounded text-[10px] hover:bg-emerald-950/50 cursor-pointer"
                            >
                              Đồng ý
                            </button>
                            <button
                              type="button"
                              onClick={() => respondPartyInvitation(inv.id, "reject")}
                              disabled={partyLoading}
                              className="px-2 py-1 bg-rose-950/15 border border-rose-950/40 text-rose-400 font-bold rounded text-[10px] hover:bg-rose-950/30 cursor-pointer"
                            >
                              Từ chối
                            </button>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            ) : (
              <div className="space-y-4">
                {/* Member list */}
                <div className="space-y-2">
                  {party.members.map((member) => {
                    const isLeader = member.id === party.leader_id;
                    const isMe = member.id === player.id;
                    return (
                      <div
                        key={member.id}
                        className="flex items-center justify-between bg-zinc-950/60 border border-zinc-900 rounded-lg p-2.5 text-xs"
                      >
                        <div
                          onClick={() => !isMe && fetchPlayerProfile(member.username)}
                          className={`flex items-center gap-2 ${!isMe ? "cursor-pointer hover:text-white" : ""}`}
                          title={!isMe ? "Bấm để trinh thám cổ sư" : undefined}
                        >
                          <div className={`w-6 h-6 rounded-full flex items-center justify-center font-bold text-[9px] overflow-hidden ${
                            isMe ? "bg-emerald-500 text-zinc-950" : "bg-zinc-800 text-zinc-300"
                          }`}>
                            {member.avatar_url ? (
                              <img src={member.avatar_url} alt="" className="w-full h-full object-cover" />
                            ) : (
                              member.username.slice(0, 2).toUpperCase()
                            )}
                          </div>
                          <span className={`font-semibold ${isMe ? "text-emerald-400" : "text-zinc-300 hover:underline decoration-dotted"}`}>
                            {member.username} {isMe && "(Bạn)"}
                          </span>
                        </div>
                        {isLeader && (
                          <span className="text-[9px] text-amber-500 bg-amber-950/30 border border-amber-900/30 px-1.5 py-0.5 rounded font-mono uppercase tracking-wider flex items-center gap-1">
                            <Crown className="w-2.5 h-2.5" /> Trưởng nhóm
                          </span>
                        )}
                      </div>
                    );
                  })}
                </div>

                {/* Invite Guidance for Leader */}
                {party.leader_id === player.id && (
                  <div className="pt-2 border-t border-zinc-900/60">
                    <p className="text-[11px] text-amber-400/90 font-serif bg-amber-950/20 border border-amber-800/30 rounded-xl p-2.5 flex items-center gap-2">
                      <span>💡</span>
                      <span>Nhấp chọn Cổ sư khác trực tiếp trên bản đồ Ngũ Vực để gửi lời mời vào tổ đội.</span>
                    </p>
                  </div>
                )}

                {/* Leave Button */}
                <button
                  type="button"
                  onClick={handleLeaveParty}
                  disabled={partyLoading}
                  className="w-full py-2 bg-rose-950/15 border border-rose-950/40 text-rose-400 hover:bg-rose-950/30 font-bold rounded-xl text-xs transition-all active:scale-[0.98] cursor-pointer"
                >
                  Rời Khỏi Tổ Đội
                </button>
              </div>
            )}
          </div>

        </div>
        <div className="lg:col-span-12">
          <LogConsole logs={logs} />
        </div>
      </div>

      {/* Challenge Sect Modal */}
      {attackTargetSect && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
          <div className="w-full max-w-md bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4 text-left">
            <div className="flex items-center justify-between border-b border-zinc-900 pb-3">
              <div>
                <h3 className="text-sm font-bold text-zinc-100 font-serif flex items-center gap-2">
                  <span>🏯</span> Khiêu Chiến Tông Môn — {attackTargetSect.name}
                </h3>
                <span className="text-[10px] font-mono text-zinc-500 block mt-0.5">Tọa độ: ({attackTargetSect.map_x}, {attackTargetSect.map_y}) · Cấp {attackTargetSect.level}</span>
              </div>
              <button onClick={() => setAttackTargetSect(null)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer"><X className="w-4 h-4 text-zinc-500" /></button>
            </div>

            <div className="flex flex-col gap-1 text-xs font-mono bg-zinc-900/40 p-3 rounded-xl border border-zinc-900">
              <div className="flex justify-between">
                <span className="text-zinc-500">Sức Phòng Thủ Hộ Sơn:</span>
                <span className="text-rose-400 font-bold">🛡 {Math.round(attackTargetSect.total_defense_power || 0).toLocaleString()}</span>
              </div>
              {attackTargetSect.shield_until && new Date(attackTargetSect.shield_until).getTime() > Date.now() && (
                <div className="flex justify-between text-rose-400 font-bold mt-1">
                  <span>Đại Trận Bảo Hộ:</span>
                  <span>{new Date(attackTargetSect.shield_until).toLocaleTimeString()}</span>
                </div>
              )}
            </div>

            <div className="flex flex-col gap-1.5">
              <label className="text-[11px] font-bold text-zinc-500 font-mono">Chọn đệ tử ra trận công kích</label>
              <select
                value={attackPlayerId}
                onChange={(e) => setAttackPlayerId(e.target.value)}
                className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-rose-500/50"
              >
                {sectMembers
                  .filter((m) => m.member_rank !== "NGOAI_MON")
                  .map((m) => (
                    <option key={m.player_id} value={m.player_id}>
                      {m.username} ({m.member_rank}) — ⚔ {Math.round(m.power || 0).toLocaleString()}
                    </option>
                  ))}
              </select>
            </div>

            <button
              disabled={actionLoading || !attackPlayerId || (attackTargetSect.shield_until ? new Date(attackTargetSect.shield_until).getTime() > Date.now() : false)}
              onClick={async () => {
                const ok = await attackSect(attackTargetSect.id, attackPlayerId);
                if (ok) setAttackTargetSect(null);
              }}
              className="w-full py-2.5 bg-gradient-to-r from-rose-600 to-red-600 hover:from-rose-500 hover:to-red-500 disabled:opacity-50 text-white rounded-xl font-serif font-bold text-xs shadow-md transition-all cursor-pointer"
            >
              {attackTargetSect.shield_until && new Date(attackTargetSect.shield_until).getTime() > Date.now() ? "Đang Bảo Hộ" : "Phát Động Công Kích Tông Môn"}
            </button>
          </div>
        </div>
      )}

      <InheritanceAltarModal />
    </GamePanel>
  );
};
