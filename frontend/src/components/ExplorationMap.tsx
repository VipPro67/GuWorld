"use client";

import React, { useState, useEffect, useRef, useCallback } from "react";
import { useGameStore, Player } from "@/store/useGameStore";
import { Compass, Users, Map as MapIcon, ChevronLeft, ChevronRight, ChevronUp, ChevronDown, Flag, User, Swords, ArrowRight, Activity, ArrowLeftRight, Leaf, Sparkles, MapPin, ShieldAlert, Award, Footprints, Flame, Gem, Flower2, HelpCircle, Wine, Crown, Package, Home } from "lucide-react";
import { translateItemType, translatePath } from "@/lib/translate";
import { LogConsole } from "./LogConsole";

interface ExplorationMapProps {
  player: Player;
  actionLoading: boolean;
  onExplore: (nodeId: string) => Promise<{ player: any; log: string; combat_triggered: boolean } | null>;
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

export const ExplorationMap: React.FC<ExplorationMapProps> = ({
  player,
  actionLoading,
  onExplore,
  onBack,
}) => {
  const [lastLog, setLastLog] = useState<string | null>(null);
  const [activeNode, setActiveNode] = useState<string | null>(null);
  const [showRegionsList, setShowRegionsList] = useState(false);
  const [gridData, setGridData] = useState<{
    x: number;
    y: number;
    vision_radius: number;
    cells: any[];
    stats: any;
    players_on_map?: any[];
    world_width?: number;
    world_height?: number;
  } | null>(null);

  const { fetchGrid, movePlayer, party, partyInvitations, createParty, fetchPartyStatus, fetchPartyInvitations, invitePlayer, respondPartyInvitation, leaveParty, retreat, fetchPlayerProfile, showConfirm } = useGameStore();
  const [inviteUsername, setInviteUsername] = useState("");
  const [inviteError, setInviteError] = useState<string | null>(null);
  const [partyLoading, setPartyLoading] = useState(false);
  const logs = useGameStore((state) => state.logs);
  const guDictionary = useGameStore((state) => state.guDictionary);
  const itemDictionary = useGameStore((state) => state.itemDictionary);
  const monsterDictionary = useGameStore((state) => state.monsterDictionary);

  const handleExitMap = async () => {
    if (isExploringActiveRegion) {
      if (await showConfirm("Đạo hữu có chắc chắn muốn rời khỏi bản đồ khám phá? Tất cả Nguyên Thạch, Cổ trùng và tài nguyên thu thập tạm thời sẽ được thu hoạch an toàn về túi đồ.")) {
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

  const isExploringActiveRegion = player.current_region !== "Southern_Border" && player.current_region !== "Lobby";
  const isPartyMemberNotLeader = party !== null && party.leader_id !== player.id;

  const loadGrid = async () => {
    if (!isExploringActiveRegion) return;
    const data = await fetchGrid();
    if (data) {
      setGridData(data);
    }
  };

  useEffect(() => {
    fetchPartyStatus();
    fetchPartyInvitations();
  }, []);

  useEffect(() => {
    loadGrid();
    
    let intervalId: NodeJS.Timeout;
    if (isExploringActiveRegion) {
      intervalId = setInterval(() => {
        loadGrid();
      }, 5000);
    }
    
    return () => {
      if (intervalId) clearInterval(intervalId);
    };
  }, [player.current_region, player.x, player.y, isExploringActiveRegion]);

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

    // Check distance range
    const dx = Math.abs(cx - player.x);
    const dy = Math.abs(cy - player.y);
    const dist = Math.max(dx, dy);

    const movementLimit = gridData?.stats?.movement || 1;

    if (dist > movementLimit) return; // Too far

    setLastLog(null);
    const res = await movePlayer(cx, cy);
    if (res) {
      setLastLog(res.log);
    }
  };

  const getEntityName = (type: string, content: string) => {
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
      case "Resource":
        if (content === "moon_orchid_petal") return <Flower2 className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />;
        if (content === "green_bamboo_wine" || content.includes("wine")) return <Wine className="w-5 h-5 text-amber-500" />;
        return <Gem className="w-5 h-5 text-cyan-400" />;
      case "Monster":
        return <Flame className="w-5 h-5 text-rose-500 animate-pulse" />;
      case "WildGu":
        return <Sparkles className="w-5 h-5 text-amber-400 animate-bounce" />;
      case "PVP":
        return <User className="w-5 h-5 text-purple-400" />;
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
      <div className="glass-panel-gold rounded-3xl p-6 w-full max-w-4xl mx-auto space-y-6 text-left">
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
                {isExploringActiveRegion ? "Thoát & Nhận Đồ" : "Trở Về"}
              </button>
            )}
          </div>
        </div>
        {/* World Overview: 3x2 large zones (Đại Vực) */}
        {(() => {
          const largeZones = regions
            .filter((r) => r.zone_type === "DAI_VUC")
            .sort((a, b) => (a.min_y || 0) - (b.min_y || 0) || (a.min_x || 0) - (b.min_x || 0));
          const namedZones = regions.filter((r) => r.zone_type === "NGU_VUC");
          return (
            <div className="space-y-6">
              <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                {largeZones.map((node, index) => {
                  const isCurrentExploring = activeNode === node.id;
                  const children = namedZones.filter((n) => n.parent_id === node.id);
                  return (
                    <div
                      key={node.id}
                      className={`border rounded-2xl p-4 flex flex-col justify-between h-60 transition-all bg-gradient-to-b ${getBgGlow(index)} to-zinc-950 relative overflow-hidden group ${
                        getColorClass(index)
                      } ${!!player.active_combat ? "opacity-40 cursor-not-allowed" : ""}`}
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
                          <div className="flex flex-wrap gap-1 pt-1">
                            {children.map((c) => (
                              <span key={c.id} className="text-[8px] font-mono text-zinc-500 bg-zinc-900/60 border border-zinc-800 px-1.5 py-0.5 rounded">
                                {c.name}
                              </span>
                            ))}
                          </div>
                        )}
                      </div>
                      <div className="mt-4 pt-3 border-t border-zinc-900/60">
                        <button
                          type="button"
                          disabled={actionLoading || !!player.active_combat}
                          onClick={() => handleNodeClick(node.id)}
                          className={`w-full py-2 px-3 text-xs font-bold rounded-xl flex items-center justify-center gap-1.5 transition-all cursor-pointer ${
                            !player.active_combat
                              ? "bg-zinc-900 border border-zinc-800 hover:bg-zinc-800 text-zinc-200 active:scale-[0.98]"
                              : "bg-zinc-950 text-zinc-600 border border-zinc-900/80 cursor-not-allowed"
                          }`}
                        >
                          {isCurrentExploring ? (
                            <>
                              <Compass className="w-3.5 h-3.5 animate-spin text-amber-400" />
                              <span>Đang di hành...</span>
                            </>
                          ) : (
                            <>
                              <Footprints className="w-3.5 h-3.5 text-emerald-400" />
                              <span>Cất Bước Khám Phá</span>
                            </>
                          )}
                        </button>
                      </div>
                    </div>
                  );
                })}
              </div>

              {isExploringActiveRegion && (
                <button
                  type="button"
                  disabled={actionLoading || !!player.active_combat}
                  onClick={() => handleNodeClick("Southern_Border")}
                  className="w-full py-2.5 px-3 text-xs font-bold rounded-xl flex items-center justify-center gap-1.5 transition-all cursor-pointer bg-emerald-950/40 text-emerald-400 border border-emerald-900/60 hover:bg-emerald-950/60 hover:border-emerald-700"
                >
                  <Home className="w-3.5 h-3.5" />
                  Trở Về Trấn Thanh Bản (An Toàn)
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
    <div className="glass-panel-gold rounded-3xl p-6 w-full max-w-6xl mx-auto space-y-6 text-left relative">
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
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowRegionsList(true)}
            className="py-1.5 px-3 bg-zinc-950 border border-zinc-800 hover:bg-zinc-900 rounded-xl text-[10px] text-zinc-400 hover:text-zinc-200 cursor-pointer font-mono"
          >
            Đổi Bản Đồ
          </button>
          {onBack && (
            <button
              onClick={handleExitMap}
              className="py-1.5 px-3 bg-emerald-950/40 text-emerald-400 border border-emerald-900/60 hover:bg-emerald-950/60 hover:border-emerald-700 rounded-xl text-[10px] cursor-pointer font-mono font-bold transition-all shadow-[0_0_10px_rgba(16,185,129,0.1)]"
            >
              Thoát & Nhận Đồ
            </button>
          )}
        </div>
      </div>

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

                  return (
                    <div
                      key={`${x},${y}`}
                      onClick={() => isMovable && handleGridCellClick(x, y)}
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
                          : `Tọa độ: (${x}, ${y}) ${
                              isPlayerCell ? "[Vị trí của bạn]" : ""
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
                        {!isPlayerCell && otherPlayersOnCell.length === 0 && !isOutOfBounds && getCellIcon(cellData.cell_type, cellData.content)}
                      </div>

                      {/* Tiny text label */}
                      {hasEntity && !isPlayerCell && otherPlayersOnCell.length === 0 && !isOutOfBounds && (
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

        {/* Right Column: Party Manager, Temporary Loot & Cultivation Log */}
        <div className="lg:col-span-4 space-y-6">
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

                {/* Invite Form for Leader */}
                {party.leader_id === player.id && (
                  <div className="pt-2 border-t border-zinc-900/60 space-y-2">
                    <span className="text-[10px] text-zinc-500 uppercase tracking-wider font-mono block">
                      Chiêu mộ thành viên
                    </span>
                    <div className="flex gap-2">
                      <input
                        type="text"
                        placeholder="Đạo hiệu đối phương..."
                        value={inviteUsername}
                        onChange={(e) => {
                          setInviteUsername(e.target.value);
                          setInviteError(null);
                        }}
                        className="bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-1.5 text-xs text-zinc-100 placeholder-zinc-750 flex-grow focus:outline-none focus:border-zinc-800 transition-colors"
                      />
                      <button
                        type="button"
                        onClick={handleInvitePlayer}
                        disabled={partyLoading || !inviteUsername.trim()}
                        className="bg-zinc-900 border border-zinc-800 hover:bg-zinc-800 text-zinc-250 hover:text-zinc-100 font-bold px-3 py-1.5 rounded-xl text-xs transition-all active:scale-[0.98] cursor-pointer font-mono shrink-0"
                      >
                        Mời
                      </button>
                    </div>
                    {inviteError && (
                      <p className="text-[9px] text-rose-600 dark:text-rose-400 font-mono">{inviteError}</p>
                    )}
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

          {/* Temporary Loot Panel */}
          <div className="bg-zinc-950/40 border border-zinc-900 rounded-3xl p-5 space-y-4">
            <div className="flex items-center justify-between border-b border-zinc-900/60 pb-2">
              <h4 className="font-serif font-bold text-sm text-emerald-600 dark:text-emerald-400 flex items-center gap-1.5">
                <Package className="w-4 h-4 text-emerald-400" /> CHIẾN LỢI PHẨM TẠM THỜI
              </h4>
              <span className="text-[8px] font-mono text-zinc-500 uppercase tracking-widest bg-emerald-950/30 border border-emerald-900/40 px-1.5 py-0.5 rounded">
                Tạm thời
              </span>
            </div>

            {(() => {
              const tempLoot = player.temp_loot || { gu_inventory: [], materials_inventory: {}, primeval_stones: 0 };
              const isEmptyLoot =
                tempLoot.primeval_stones === 0 &&
                tempLoot.gu_inventory.length === 0 &&
                Object.values(tempLoot.materials_inventory).every((qty) => qty <= 0);

              if (isEmptyLoot) {
                return (
                  <div className="text-center py-3">
                    <p className="text-[11px] text-zinc-500 leading-normal italic">
                      Đạo hữu chưa thu thập được gì. Di chuyển trên lưới bản đồ để tìm tài nguyên hoặc đánh bại đối thủ.
                    </p>
                  </div>
                );
              }

              return (
                <div className="space-y-2 max-h-[160px] overflow-y-auto custom-scrollbar pr-1">
                  {/* Primeval Stones */}
                  {tempLoot.primeval_stones > 0 && (
                    <div className="flex items-center justify-between bg-zinc-950/60 border border-amber-900/20 rounded-xl p-2 text-xs">
                      <div className="flex items-center gap-2">
                        <div className="w-6 h-6 rounded-lg bg-amber-500/10 flex items-center justify-center">
                          <Gem className="w-3.5 h-3.5 text-amber-500" />
                        </div>
                        <span className="font-semibold text-zinc-300">Nguyên Thạch</span>
                      </div>
                      <span className="font-mono font-bold text-amber-400">+{tempLoot.primeval_stones}</span>
                    </div>
                  )}

                  {/* Gu Inventory */}
                  {tempLoot.gu_inventory.map((gu, idx) => {
                    const spec = guDictionary[gu.gu_id] || { name: gu.gu_id, path: ["Không rõ"] };
                    return (
                      <div
                        key={gu.instance_id || idx}
                        className="flex items-center justify-between bg-zinc-950/60 border border-emerald-900/20 rounded-xl p-2 text-xs"
                      >
                        <div className="flex items-center gap-2">
                          <div className="w-6 h-6 rounded-lg bg-emerald-500/10 flex items-center justify-center">
                            <Sparkles className="w-3.5 h-3.5 text-emerald-400 animate-pulse" />
                          </div>
                          <div>
                            <span className="font-semibold text-zinc-300 block leading-tight">{spec.name}</span>
                            <span className="text-[8px] text-zinc-550 font-mono">
                              {spec.path && (Array.isArray(spec.path) ? spec.path.map(translatePath).join(" / ") : translatePath(spec.path))}
                            </span>
                          </div>
                        </div>
                        <span className="text-[8px] font-mono text-emerald-400 bg-emerald-950/20 border border-emerald-900/30 px-1 py-0.5 rounded uppercase tracking-wide shrink-0">
                          Cổ trùng
                        </span>
                      </div>
                    );
                  })}

                  {/* Materials */}
                  {Object.entries(tempLoot.materials_inventory)
                    .filter(([_, qty]) => qty > 0)
                    .map(([matId, qty]) => {
                      const spec = itemDictionary[matId] || { name: matId, item_type: "Nguyên Liệu" };
                      return (
                        <div
                          key={matId}
                          className="flex items-center justify-between bg-zinc-950/60 border border-zinc-900 rounded-xl p-2 text-xs"
                        >
                          <div className="flex items-center gap-2">
                            <div className="w-6 h-6 rounded-lg bg-zinc-900/60 flex items-center justify-center">
                              {matId.includes("wine") ? (
                                <Wine className="w-3.5 h-3.5 text-amber-500" />
                              ) : matId.includes("petal") || matId.includes("flower") ? (
                                <Flower2 className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400" />
                              ) : (
                                <Gem className="w-3.5 h-3.5 text-cyan-400" />
                              )}
                            </div>
                            <div>
                              <span className="font-semibold text-zinc-300 block leading-tight">{spec.name}</span>
                              <span className="text-[8px] text-zinc-550 font-mono">{translateItemType(spec.item_type)}</span>
                            </div>
                          </div>
                          <span className="font-mono font-bold text-zinc-400 shrink-0">x{qty}</span>
                        </div>
                      );
                    })}
                </div>
              );
            })()}
          </div>

        </div>
        <div className="lg:col-span-12">
          <LogConsole logs={logs} />
        </div>
      </div>
    </div>
  );
};
