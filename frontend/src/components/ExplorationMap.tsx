"use client";

import React, { useState, useEffect } from "react";
import { Player, useGameStore } from "@/store/useGameStore";
import { Compass, Sparkles, MapPin, ShieldAlert, Award, Footprints, Flame, User, Gem, Flower2, HelpCircle, Wine, Users, Crown } from "lucide-react";

interface ExplorationMapProps {
  player: Player;
  actionLoading: boolean;
  onExplore: (nodeId: string) => Promise<{ player: any; log: string; combat_triggered: boolean } | null>;
  onBack?: () => void;
}

interface MapNode {
  id: string;
  name: string;
  region: string;
  recommendedRank: string;
  description: string;
  colorClass: string;
  bgGlow: string;
}

const MAP_NODES: MapNode[] = [
  {
    id: "qing_mao_mountain",
    name: "Thanh Mao Sơn",
    region: "Southern Border (Nam Cương)",
    recommendedRank: "Nhất Chuyển (Rank 1)",
    description: "Ngọn núi phủ đầy trúc xanh, nơi có ba gia tộc lớn sinh sống. Có dã thú cấp thấp rình rập trong rừng sâu.",
    colorClass: "text-emerald-400 border-emerald-500/20 hover:border-emerald-500/50 bg-emerald-950/10",
    bgGlow: "from-emerald-500/5 to-transparent",
  },
  {
    id: "gu_luan_slope",
    name: "Cổ Loạn Sườn",
    region: "Southern Border (Nam Cương)",
    recommendedRank: "Nhất - Nhị Chuyển (Rank 1-2)",
    description: "Khu vực núi đá hiểm trở hoang sơ, chứa nhiều bộ xương khô dã thú cổ đại và các bầy thú dữ lang thang.",
    colorClass: "text-blue-400 border-blue-500/20 hover:border-blue-500/50 bg-blue-950/10",
    bgGlow: "from-blue-500/5 to-transparent",
  },
  {
    id: "three_kings_inheritance",
    name: "Tam Vương Truyền Thừa",
    region: "Southern Border (Nam Cương)",
    recommendedRank: "Nhị - Tam Chuyển (Rank 2-3)",
    description: "Bảo cảnh kế thừa của ba vị ma đạo cổ sư: Thiết Vương, Khương Vương, và Tín Vương. Vô cùng hung hiểm.",
    colorClass: "text-purple-400 border-purple-500/20 hover:border-purple-500/50 bg-purple-950/10",
    bgGlow: "from-purple-500/5 to-transparent",
  },
];

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
    explored_cells: string[];
    stats: any;
    players_on_map?: any[];
  } | null>(null);

  const { fetchGrid, movePlayer, party, createParty, fetchPartyStatus, invitePlayer, leaveParty } = useGameStore();
  const [inviteUsername, setInviteUsername] = useState("");
  const [inviteError, setInviteError] = useState<string | null>(null);
  const [partyLoading, setPartyLoading] = useState(false);

  const isExploringActiveRegion =
    player.current_region === "qing_mao_mountain" ||
    player.current_region === "gu_luan_slope" ||
    player.current_region === "three_kings_inheritance";

  const loadGrid = async () => {
    if (!isExploringActiveRegion) return;
    const data = await fetchGrid();
    if (data) {
      setGridData(data);
    }
    fetchPartyStatus();
  };

  useEffect(() => {
    loadGrid();
  }, [player.current_region, player.x, player.y]);

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

  const handleNodeClick = async (node: MapNode) => {
    if (actionLoading || player.active_combat) return;

    setActiveNode(node.id);
    setLastLog(null);

    try {
      const res = await onExplore(node.id);
      if (res) {
        setLastLog(res.log);
        setShowRegionsList(false);
      }
    } finally {
      setActiveNode(null);
    }
  };

  const handleGridCellClick = async (cx: number, cy: number) => {
    if (actionLoading || player.active_combat) return;

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
      if (res.combat_triggered) {
        // Switch view to dashboard where combat renders
        useGameStore.getState().setView("dashboard");
      }
    }
  };

  // Helper to format map cell content icon
  const getCellIcon = (type: string, content: string) => {
    switch (type) {
      case "Resource":
        if (content === "moon_orchid_petal") return <Flower2 className="w-5 h-5 text-emerald-450" />;
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
    const node = MAP_NODES.find((n) => n.id === id);
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
              Chọn Lưu Vực Du Ngoạn (Select Region Map)
            </h3>
          </div>
          <div className="flex items-center gap-4">
            {isExploringActiveRegion && (
              <button
                onClick={() => setShowRegionsList(false)}
                className="py-1 px-3 border border-zinc-800 rounded-xl text-[10px] text-zinc-450 hover:bg-zinc-900 cursor-pointer font-mono"
              >
                Về Bản Đồ Hiện Tại
              </button>
            )}
            {onBack && (
              <button
                type="button"
                onClick={onBack}
                disabled={actionLoading}
                className="py-1 px-3 border border-zinc-800 rounded-xl text-[10px] text-zinc-450 hover:bg-zinc-900 transition-all cursor-pointer font-mono font-bold"
              >
                Trở Về
              </button>
            )}
          </div>
        </div>

        <div className="text-xs text-zinc-500 max-w-xl">
          * Chọn vùng đất để cất bước hành trình. Di chuyển sang khu vực mới sẽ đưa đạo hữu về làng trung tâm (50, 50) của vùng đất đó và tiêu hao <span className="text-emerald-400 font-bold">5 Chân Nguyên</span>.
        </div>

        {/* Grid Nodes */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {MAP_NODES.map((node) => {
            const isCurrentExploring = activeNode === node.id;
            const hasMana = player.current_mana >= 5;

            return (
              <div
                key={node.id}
                className={`border rounded-2xl p-4 flex flex-col justify-between h-56 transition-all bg-gradient-to-b ${node.bgGlow} to-zinc-950 relative overflow-hidden group ${
                  node.colorClass
                } ${!!player.active_combat ? "opacity-40 cursor-not-allowed" : ""}`}
              >
                <div className="space-y-2">
                  <div className="flex items-start justify-between">
                    <span className="text-[9px] text-zinc-500 uppercase tracking-widest font-mono">
                      {node.region}
                    </span>
                    <MapPin className="w-3.5 h-3.5 opacity-60 group-hover:scale-110 transition-transform" />
                  </div>

                  <h4 className="text-base font-bold text-zinc-100 font-serif group-hover:text-amber-400 transition-colors">
                    {node.name}
                  </h4>

                  <div className="flex items-center gap-1 text-[10px] text-zinc-500 font-mono">
                    <Award className="w-3 h-3 text-amber-500" />
                    Khuyên dùng: {node.recommendedRank}
                  </div>

                  <p className="text-[11px] text-zinc-400 leading-normal line-clamp-3">
                    {node.description}
                  </p>
                </div>

                <div className="mt-4 pt-3 border-t border-zinc-900/60">
                  <button
                    type="button"
                    disabled={actionLoading || !!player.active_combat || !hasMana}
                    onClick={() => handleNodeClick(node)}
                    className={`w-full py-2 px-3 text-xs font-bold rounded-xl flex items-center justify-center gap-1.5 transition-all cursor-pointer ${
                      hasMana && !player.active_combat
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
  const visionRadius = gridData?.vision_radius || 1;
  const cols = 2 * visionRadius + 1;

  // Sort cells top-to-bottom (Y), left-to-right (X)
  const sortedCells = gridData
    ? [...gridData.cells].sort((a, b) => {
        if (a.y !== b.y) return a.y - b.y;
        return a.x - b.x;
      })
    : [];

  const movementRange = gridData?.stats?.movement || 1;

  return (
    <div className="glass-panel-gold rounded-3xl p-6 w-full max-w-5xl mx-auto space-y-6 text-left relative">
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
              onClick={onBack}
              className="py-1.5 px-3 border border-zinc-800 hover:bg-zinc-900 rounded-xl text-[10px] text-zinc-400 cursor-pointer font-mono font-bold"
            >
              Trở Về
            </button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        {/* Left Column: Map Grid & Logs */}
        <div className="lg:col-span-8 space-y-6">
          <div className="text-[10px] text-zinc-500 font-mono max-w-xl">
            * Nhấp vào một ô trong phạm vi thân pháp (<span className="text-emerald-400 font-bold">{movementRange} ô</span>) để di chuyển. 
            Chân Nguyên: 1 bước mất <span className="text-emerald-450 font-bold">2 mana</span>, 2 bước mất <span className="text-emerald-450 font-bold">3 mana</span>.
          </div>

          {/* Grid Canvas */}
          <div className="flex justify-center items-center py-2">
            {gridData ? (
              <div
                className="grid gap-2 border border-zinc-900 bg-zinc-950/20 p-4 rounded-2xl shadow-inner max-w-md w-full"
                style={{
                  gridTemplateColumns: `repeat(${cols}, minmax(0, 1fr))`,
                }}
              >
                {sortedCells.map((cell) => {
                  const isPlayerCell = cell.x === player.x && cell.y === player.y;

                  const dx = Math.abs(cell.x - player.x);
                  const dy = Math.abs(cell.y - player.y);
                  const distance = Math.max(dx, dy);

                  const isMovable = distance > 0 && distance <= movementRange;
                  const hasEntity = cell.cell_type !== "Empty";

                  // Other players on this cell
                  const otherPlayersOnCell = gridData?.players_on_map?.filter(
                    (p: any) => p.x === cell.x && p.y === cell.y
                  ) || [];

                  return (
                    <div
                      key={`${cell.x},${cell.y}`}
                      onClick={() => isMovable && handleGridCellClick(cell.x, cell.y)}
                      className={`aspect-square rounded-xl border flex flex-col justify-center items-center relative transition-all duration-300 ${
                        isPlayerCell
                          ? "bg-emerald-500/10 border-emerald-500 shadow-[0_0_12px_rgba(16,185,129,0.2)] animate-pulse"
                          : isMovable
                          ? "bg-zinc-950/30 border-zinc-800 hover:bg-emerald-950/20 hover:border-emerald-500/30 cursor-pointer"
                          : "bg-zinc-950/50 border-zinc-950 opacity-40 cursor-not-allowed"
                      }`}
                      title={`Tọa độ: (${cell.x}, ${cell.y}) ${
                        isPlayerCell ? "[Vị trí của bạn]" : ""
                      }${
                        otherPlayersOnCell.length > 0
                          ? ` | Cổ sư: ${otherPlayersOnCell.map((p: any) => p.username).join(", ")}`
                          : ""
                      }`}
                    >
                      {/* Coordinates indicator overlay */}
                      <span className="text-[7px] text-zinc-700 font-mono absolute top-1.5">
                        {cell.x},{cell.y}
                      </span>

                      {/* Cell Entity View */}
                      {isPlayerCell ? (
                        <div className="w-5 h-5 rounded-full bg-emerald-500 flex items-center justify-center text-[9px] text-zinc-950 font-bold border border-white/20 animate-bounce">
                          Ta
                        </div>
                      ) : otherPlayersOnCell.length > 0 ? (
                        <div className="w-5 h-5 rounded-full bg-purple-500/80 flex items-center justify-center text-[9px] text-zinc-100 font-bold border border-white/20 shadow-[0_0_8px_rgba(139,92,246,0.6)] animate-pulse">
                          {otherPlayersOnCell[0].username.slice(0, 2).toUpperCase()}
                        </div>
                      ) : (
                        getCellIcon(cell.cell_type, cell.content)
                      )}

                      {/* Tiny text label */}
                      {hasEntity && !isPlayerCell && otherPlayersOnCell.length === 0 && (
                        <span className="text-[7px] font-mono text-zinc-400 absolute bottom-1 px-1 truncate max-w-full text-center">
                          {cell.cell_type === "Resource" 
                            ? cell.content.replace(/_/g, " ").replace(/\b\w/g, (c: string) => c.toUpperCase()) 
                            : cell.content.split(" ")[0]}
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
          {lastLog && (
            <div className="bg-zinc-950/70 border border-zinc-900 rounded-2xl p-4 flex items-start gap-3">
              <div className="w-7 h-7 rounded-lg bg-amber-500/10 flex items-center justify-center shrink-0 mt-0.5">
                <Sparkles className="w-4 h-4 text-amber-400" />
              </div>
              <div className="space-y-1">
                <span className="text-[9px] text-zinc-550 uppercase tracking-widest block font-mono">Nhật ký hành quân</span>
                <p className="text-xs text-zinc-300 font-mono leading-relaxed">{lastLog}</p>
              </div>
            </div>
          )}

          {player.active_combat && (
            <div className="bg-rose-950/20 border border-rose-900/40 rounded-2xl p-4 flex items-center justify-between animate-pulse">
              <div className="flex items-center gap-2.5">
                <ShieldAlert className="w-5 h-5 text-rose-500" />
                <div className="text-left">
                  <div className="text-xs font-bold text-rose-400 font-serif">Đang chạm trán dã thú/đối thủ!</div>
                  <div className="text-[10px] text-zinc-500">
                    Hãy trở về Căn cứ (Dashboard) để kích hoạt sát chiêu tiêu diệt kẻ thù.
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Right Column: Party Manager */}
        <div className="lg:col-span-4 bg-zinc-950/40 border border-zinc-900 rounded-3xl p-5 space-y-4">
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
                Đạo hữu chưa gia nhập tổ đội. Lập tổ đội để đi chung bản đồ và cùng chia sẻ Nguyên thạch & Vật phẩm rơi ra khi chiến thắng.
              </p>
              <button
                type="button"
                onClick={handleCreateParty}
                disabled={partyLoading}
                className="w-full py-2 bg-emerald-950/30 border border-emerald-500/30 text-emerald-450 hover:bg-emerald-950/50 font-bold rounded-xl text-xs transition-all active:scale-[0.98] cursor-pointer"
              >
                {partyLoading ? "Đang lập tổ..." : "Lập Tổ Đội Mới"}
              </button>
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
                      <div className="flex items-center gap-2">
                        <div className={`w-6 h-6 rounded-full flex items-center justify-center font-bold text-[9px] ${
                          isMe ? "bg-emerald-500 text-zinc-950" : "bg-zinc-800 text-zinc-300"
                        }`}>
                          {member.username.slice(0, 2).toUpperCase()}
                        </div>
                        <span className={`font-semibold ${isMe ? "text-emerald-400" : "text-zinc-300"}`}>
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
                    <p className="text-[9px] text-rose-450 font-mono">{inviteError}</p>
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
      </div>
    </div>
  );
};
