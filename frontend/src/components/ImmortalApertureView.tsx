import React, { useState, useEffect } from "react";
import { useGameStore, ApertureResourceNode, TribulationEnemy, GuInstance } from "@/store/useGameStore";
import {
  Shield,
  Activity,
  Map as MapIcon,
  Layers,
  Flame,
  Coins,
  Hourglass,
  Skull,
  AlertTriangle,
  Heart,
  Zap,
  Sparkles,
  Plus,
  Check,
  X,
  Sword,
  TrendingUp,
  Package,
  Lock as LockIcon,
} from "lucide-react";

const CountdownTimer: React.FC<{ node: ApertureResourceNode; timeRate: number; onComplete?: () => void }> = ({ node, timeRate, onComplete }) => {
  const [timeLeft, setTimeLeft] = useState<string>("");

  useEffect(() => {
    const calculateTime = () => {
      if (!node.last_collected_at) {
        setTimeLeft("Sẵn sàng");
        return;
      }
      const lastTime = new Date(node.last_collected_at).getTime();
      const rate = timeRate <= 0 ? 1.0 : timeRate;
      const nextCollectTime = lastTime + (node.production_interval_hours * 60 * 60 * 1000) / rate;
      const diff = nextCollectTime - Date.now();

      if (diff <= 0) {
        setTimeLeft("Sẵn sàng");
        if (onComplete) onComplete();
        return;
      }

      const hours = Math.floor(diff / (60 * 60 * 1000));
      const minutes = Math.floor((diff % (60 * 60 * 1000)) / (60 * 1000));
      const seconds = Math.floor((diff % (60 * 1000)) / 1000);

      const parts = [];
      if (hours > 0) parts.push(`${hours}h`);
      if (minutes > 0 || hours > 0) parts.push(`${minutes}m`);
      parts.push(`${seconds}s`);

      setTimeLeft(parts.join(" "));
    };

    calculateTime();
    const interval = setInterval(calculateTime, 1000);
    return () => clearInterval(interval);
  }, [node, timeRate]);

  return <span>{timeLeft}</span>;
};

export const ImmortalApertureView: React.FC = () => {
  const {
    player,
    aperture,
    daoMarks,
    apertureNodes,
    activeTribulation,
    guDictionary,
    itemDictionary,
    actionLoading,
    fetchAperture,
    fetchDaoMarks,
    fetchApertureNodes,
    placeApertureNode,
    collectResourceNode,
    uninstallApertureNode,
    deleteApertureNode,
    fetchActiveTribulation,
    attackTribulationEnemy,
    resolveTribulation,
    setView,
  } = useGameStore();

  const [activeTab, setActiveTab] = useState<"overview" | "map" | "dao" | "tribulation">("overview");
  const [selectedCell, setSelectedCell] = useState<{ x: number; y: number } | null>(null);
  const [selectedNode, setSelectedNode] = useState<ApertureResourceNode | null>(null);
  const [selectedEnemy, setSelectedEnemy] = useState<TribulationEnemy | null>(null);
  const [selectedGu, setSelectedGu] = useState<string>("");

  useEffect(() => {
    fetchAperture();
    fetchDaoMarks();
    fetchApertureNodes();
    fetchActiveTribulation();
  }, []);

  // Sync state if activeTribulation changes
  useEffect(() => {
    if (activeTribulation && activeTribulation.enemies.length > 0) {
      // Set default selected enemy to boss if exists, else first enemy
      const boss = activeTribulation.enemies.find((e) => e.is_boss);
      setSelectedEnemy(boss || activeTribulation.enemies[0]);
    } else {
      setSelectedEnemy(null);
    }
  }, [activeTribulation]);

  if (!player) {
    return (
      <div className="flex flex-col items-center justify-center p-8 bg-slate-900/80 backdrop-blur-md rounded-2xl border border-slate-700/50 max-w-lg mx-auto text-center">
        <Loader />
        <p className="mt-4 text-slate-400">Đang tải thông tin Cổ sư...</p>
      </div>
    );
  }

  if (player.rank < 6) {
    return (
      <div className="flex flex-col items-center justify-center p-8 bg-slate-900/80 backdrop-blur-md rounded-2xl border border-slate-700/50 max-w-lg mx-auto text-center">
        <LockIcon className="w-16 h-16 text-amber-500 mb-4 animate-pulse" />
        <h2 className="text-2xl font-bold bg-gradient-to-r from-amber-200 to-yellow-500 bg-clip-text text-transparent mb-2">
          Chưa Khai Mở Tiên Khiếu
        </h2>
        <p className="text-slate-300 mb-6 leading-relaxed">
          Đạo hữu hiện tại tu vi mới là **Rank {player.rank}**. 
          Chỉ sau khi đột phá **Rank 5 Đỉnh Phong**, tụ khí tam hoa thăng thiên thành công lên **Rank 6 (Hồn Tiên/Yêu Tiên)**, đạo hữu mới có thể phá vỡ phàm khiếu, khai mở **Tiên Khiếu** (Phúc Địa) của riêng mình.
        </p>
        <button
          onClick={() => setView("dashboard")}
          className="px-6 py-2 bg-gradient-to-r from-slate-700 to-slate-800 hover:from-slate-600 hover:to-slate-700 text-slate-100 rounded-lg border border-slate-600 transition-all"
        >
          Quay lại Động Phủ
        </button>
      </div>
    );
  }

  // Define nodes dict for checking cell contents
  const nodeMap = new Map<string, ApertureResourceNode>();
  apertureNodes.forEach((node) => {
    nodeMap.set(`${node.x},${node.y}`, node);
  });

  // Define enemies dict for grid overlay
  const enemyMap = new Map<string, TribulationEnemy>();
  if (activeTribulation) {
    activeTribulation.enemies.forEach((enemy) => {
      enemyMap.set(`${enemy.x},${enemy.y}`, enemy);
    });
  }

  const handleCellClick = (x: number, y: number) => {
    const coordKey = `${x},${y}`;
    const node = nodeMap.get(coordKey);
    const enemy = enemyMap.get(coordKey);

    if (enemy) {
      setSelectedEnemy(enemy);
      setActiveTab("tribulation");
      return;
    }

    setSelectedCell({ x, y });
    setSelectedNode(node || null);
  };

  const handleBuildNode = async (nodeType: string) => {
    if (!selectedCell) return;
    const success = await placeApertureNode(nodeType, selectedCell.x, selectedCell.y);
    if (success) {
      setSelectedCell(null);
      setSelectedNode(null);
    }
  };

  const handleCollect = async (nodeId: string) => {
    const success = await collectResourceNode(nodeId);
    if (success) {
      setSelectedCell(null);
      setSelectedNode(null);
    }
  };

  const handleUninstall = async () => {
    if (!selectedNode) return;
    if (window.confirm(`Đạo hữu có chắc chắn muốn tháo dỡ ${selectedNode.name}? Mỏ sẽ được thu hồi thành vật phẩm cất vào túi đồ.`)) {
      const success = await uninstallApertureNode(selectedNode.id);
      if (success) {
        setSelectedCell(null);
        setSelectedNode(null);
      }
    }
  };

  const handleDelete = async () => {
    if (!selectedNode) return;
    const confirmMsg = selectedNode.is_destroyed
      ? `Mỏ này đã bị phá hủy. Đạo hữu muốn dọn dẹp phế tích và san phẳng ô đất chứ?`
      : `CẢNH BÁO: Đạo hữu có chắc chắn muốn san phẳng ${selectedNode.name}? Hành động này sẽ xóa vĩnh viễn mỏ và KHÔNG hoàn trả lại mô hình mỏ.`;
    if (window.confirm(confirmMsg)) {
      const success = await deleteApertureNode(selectedNode.id);
      if (success) {
        setSelectedCell(null);
        setSelectedNode(null);
      }
    }
  };

  const handleAttack = async () => {
    if (!activeTribulation || !selectedEnemy || !selectedGu) return;
    await attackTribulationEnemy(activeTribulation.event.id, selectedEnemy.id, selectedGu);
  };

  const handleResolve = async () => {
    if (!activeTribulation) return;
    await resolveTribulation(activeTribulation.event.id);
  };

  // Helper helper to format names nicely
  const getGradeName = (grade?: string) => {
    switch (grade) {
      case "LOW":
        return "Hạ Đẳng";
      case "MID":
        return "Trung Đẳng";
      case "HIGH":
        return "Thượng Đẳng";
      case "EXTREME":
        return "Chí Cao";
      default:
        return "Không Xác Định";
    }
  };

  const getStateName = (state?: string) => {
    switch (state) {
      case "STABLE":
        return { text: "Ổn Định", color: "text-emerald-400 bg-emerald-500/10 border-emerald-500/30" };
      case "TRIBULATION_PENDING":
        return { text: "Kiếp Lực Tích Lũy", color: "text-yellow-400 bg-yellow-500/10 border-yellow-500/30" };
      case "TRIBULATION_ACTIVE":
        return { text: "Tai Kiếp Hoành Hành", color: "text-rose-400 bg-rose-500/10 border-rose-500/30 animate-pulse border" };
      case "DAMAGED":
        return { text: "Hư Tổn Nặng", color: "text-amber-400 bg-amber-500/10 border-amber-500/30" };
      case "DESTROYED":
        return { text: "Đã Bị Huỷ", color: "text-red-500 bg-red-500/10 border-red-500/30" };
      default:
        return { text: "Ổn Định", color: "text-emerald-400 bg-emerald-500/10" };
    }
  };

  const getApertureState = getStateName(aperture?.state);

  // Filter player active Gu bugs
  const activeGus = player.gu_inventory.filter((gu) => gu.status === "ACTIVE");

  return (
    <div className="w-full max-w-6xl mx-auto space-y-6">
      {/* Header Panel */}
      <div className="relative overflow-hidden bg-slate-900/80 backdrop-blur-md border border-slate-700/50 rounded-2xl p-6 shadow-xl">
        <div className="absolute top-0 right-0 w-64 h-64 bg-amber-500/5 rounded-full blur-3xl -z-10" />
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <h1 className="text-3xl font-extrabold bg-gradient-to-r from-amber-200 via-yellow-400 to-amber-500 bg-clip-text text-transparent flex items-center gap-2">
              <Sparkles className="w-8 h-8 text-amber-400" />
              Tiên Khiếu Phúc Địa
            </h1>
            <p className="text-slate-400 mt-1 text-sm">
              Thế giới nội bộ độc lập của Cổ sư Rank 6. Tích lũy tài nguyên, tăng phúc Đạo Ngân và chống chọi thiên đạo kiếp nạn.
            </p>
          </div>

          <div className="flex items-center gap-3">
            <div className={`px-4 py-2 rounded-xl text-sm font-semibold border ${getApertureState.color}`}>
              Trạng thái: {getApertureState.text}
            </div>
            <button
              onClick={() => setView("dashboard")}
              className="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-200 text-sm font-medium rounded-xl border border-slate-700 transition"
            >
              Về Động Phủ
            </button>
          </div>
        </div>
      </div>

      {/* Tabs Navigation */}
      <div className="flex bg-slate-950/40 p-1.5 rounded-xl border border-slate-800/80 gap-1 overflow-x-auto">
        <button
          onClick={() => setActiveTab("overview")}
          className={`flex items-center gap-2 px-5 py-3 rounded-lg text-sm font-semibold transition-all whitespace-nowrap ${
            activeTab === "overview"
              ? "bg-gradient-to-r from-amber-500/20 to-yellow-600/20 border border-amber-500/40 text-amber-300"
              : "text-slate-400 hover:text-slate-200"
          }`}
        >
          <Activity className="w-4 h-4" />
          Tổng Quan
        </button>
        <button
          onClick={() => setActiveTab("map")}
          className={`flex items-center gap-2 px-5 py-3 rounded-lg text-sm font-semibold transition-all whitespace-nowrap ${
            activeTab === "map"
              ? "bg-gradient-to-r from-amber-500/20 to-yellow-600/20 border border-amber-500/40 text-amber-300"
              : "text-slate-400 hover:text-slate-200"
          }`}
        >
          <MapIcon className="w-4 h-4" />
          Bản Đồ Lưới (20x20)
        </button>
        <button
          onClick={() => setActiveTab("dao")}
          className={`flex items-center gap-2 px-5 py-3 rounded-lg text-sm font-semibold transition-all whitespace-nowrap ${
            activeTab === "dao"
              ? "bg-gradient-to-r from-amber-500/20 to-yellow-600/20 border border-amber-500/40 text-amber-300"
              : "text-slate-400 hover:text-slate-200"
          }`}
        >
          <Layers className="w-4 h-4" />
          Bảng Đạo Ngân
        </button>
        <button
          onClick={() => setActiveTab("tribulation")}
          className={`flex items-center gap-2 px-5 py-3 rounded-lg text-sm font-semibold transition-all whitespace-nowrap relative ${
            activeTab === "tribulation"
              ? "bg-gradient-to-r from-amber-500/20 to-yellow-600/20 border border-amber-500/40 text-amber-300"
              : "text-slate-400 hover:text-slate-200"
          }`}
        >
          <Skull className="w-4 h-4" />
          Tai Kiếp & Phòng Thủ
          {aperture?.state === "TRIBULATION_ACTIVE" && (
            <span className="absolute top-1 right-1 flex h-2.5 w-2.5">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-rose-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-rose-500"></span>
            </span>
          )}
        </button>
      </div>

      {/* Tab Contents */}
      <div className="grid grid-cols-1 gap-6">
        {/* TAB 1: OVERVIEW */}
        {activeTab === "overview" && aperture && (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {/* Core Stats Card */}
            <div className="md:col-span-2 bg-slate-900/60 backdrop-blur-md border border-slate-700/40 rounded-2xl p-6 space-y-6">
              <h2 className="text-xl font-bold text-slate-100 border-b border-slate-800 pb-3 flex items-center gap-2">
                <Sparkles className="w-5 h-5 text-amber-400" />
                Thông Số Tiên Khiếu
              </h2>

              <div className="grid grid-cols-2 gap-4">
                <div className="bg-slate-950/50 p-4 rounded-xl border border-slate-800/80">
                  <div className="text-slate-400 text-xs font-semibold">Cấp Phúc Địa</div>
                  <div className="text-lg font-bold text-amber-400 mt-1">{getGradeName(aperture.grade)}</div>
                </div>

                <div className="bg-slate-950/50 p-4 rounded-xl border border-slate-800/80">
                  <div className="text-slate-400 text-xs font-semibold">Gia Tốc Thời Gian (Time Rate)</div>
                  <div className="text-lg font-bold text-cyan-400 mt-1">x{aperture.time_rate} lần</div>
                </div>

                <div className="bg-slate-950/50 p-4 rounded-xl border border-slate-800/80">
                  <div className="text-slate-400 text-xs font-semibold">Tiên Nguyên Dự Trữ</div>
                  <div className="text-lg font-bold text-yellow-400 mt-1 flex items-center gap-1">
                    <Coins className="w-5 h-5 text-yellow-500" />
                    {aperture.immortal_essence} hạt
                  </div>
                </div>

                <div className="bg-slate-950/50 p-4 rounded-xl border border-slate-800/80">
                  <div className="text-slate-400 text-xs font-semibold">Sản Lượng Tiên Nguyên</div>
                  <div className="text-lg font-bold text-emerald-400 mt-1">{aperture.essence_production_per_year} hạt / năm</div>
                </div>

                <div className="bg-slate-950/50 p-4 rounded-xl border border-slate-800/80 col-span-2">
                  <div className="text-slate-400 text-xs font-semibold">Diện Tích Đất Đai</div>
                  <div className="text-lg font-bold text-slate-200 mt-1">{(aperture.area_size / 666.67).toFixed(1)} Mẫu (đất cát, rừng hồ...)</div>
                </div>
              </div>

              {/* Integrity HP Bar */}
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-slate-400 font-medium">Độ Ổn Định Linh Khiếu (HP)</span>
                  <span className="text-slate-200 font-bold">
                    {aperture.aperture_hp} / {aperture.max_aperture_hp}
                  </span>
                </div>
                <div className="w-full bg-slate-950/80 h-3 rounded-full overflow-hidden border border-slate-800">
                  <div
                    className="h-full bg-gradient-to-r from-rose-500 to-emerald-500 transition-all duration-500"
                    style={{ width: `${(aperture.aperture_hp / aperture.max_aperture_hp) * 100}%` }}
                  />
                </div>
              </div>
            </div>

            {/* Time / Next Tribulation Year */}
            <div className="bg-slate-900/60 backdrop-blur-md border border-slate-700/40 rounded-2xl p-6 space-y-6 flex flex-col justify-between">
              <div className="space-y-4">
                <h2 className="text-xl font-bold text-slate-100 border-b border-slate-800 pb-3 flex items-center gap-2">
                  <Hourglass className="w-5 h-5 text-cyan-400" />
                  Mốc Khai Linh
                </h2>

                <div className="space-y-3">
                  <div className="flex justify-between items-center text-sm">
                    <span className="text-slate-400">Số năm nội bộ đã trôi qua</span>
                    <span className="text-slate-200 font-semibold">{aperture.internal_years_passed.toFixed(4)} năm</span>
                  </div>
                  <div className="flex justify-between items-center text-sm">
                    <span className="text-slate-400">Mốc tai kiếp kế tiếp</span>
                    <span className="text-rose-400 font-semibold">{aperture.next_tribulation_year.toFixed(1)} năm</span>
                  </div>
                  <div className="flex justify-between items-center text-sm">
                    <span className="text-slate-400">Tổng số kiếp nạn vượt qua</span>
                    <span className="text-emerald-400 font-semibold">{aperture.tribulation_count} lần</span>
                  </div>
                </div>

                {/* Progress to next Tribulation */}
                <div className="space-y-2 pt-2">
                  <div className="flex justify-between text-xs">
                    <span className="text-slate-400">Tiến độ tích kiếp lực</span>
                    <span className="text-amber-400 font-semibold">
                      {Math.min(100, Math.max(0, (aperture.internal_years_passed / aperture.next_tribulation_year) * 100)).toFixed(1)}%
                    </span>
                  </div>
                  <div className="w-full bg-slate-950/80 h-2 rounded-full overflow-hidden border border-slate-800">
                    <div
                      className="h-full bg-gradient-to-r from-cyan-500 to-amber-500"
                      style={{ width: `${Math.min(100, (aperture.internal_years_passed / aperture.next_tribulation_year) * 100)}%` }}
                    />
                  </div>
                </div>
              </div>

              {aperture.state === "TRIBULATION_ACTIVE" ? (
                <div className="p-4 bg-rose-500/10 border border-rose-500/30 rounded-xl text-center space-y-2">
                  <AlertTriangle className="w-8 h-8 text-rose-500 mx-auto animate-bounce" />
                  <p className="text-sm text-rose-200 font-medium">
                    Một tai kiếp cuồng bạo đã ập đến! Mau chóng vào phòng thủ bảo vệ Tiên Khiếu.
                  </p>
                  <button
                    onClick={() => setActiveTab("tribulation")}
                    className="w-full py-2 bg-rose-600 hover:bg-rose-500 text-white font-semibold text-sm rounded-lg transition"
                  >
                    Vào Vượt Kiếp
                  </button>
                </div>
              ) : (
                <div className="p-4 bg-slate-950/50 border border-slate-850 rounded-xl text-center">
                  <Check className="w-8 h-8 text-emerald-500 mx-auto mb-2" />
                  <p className="text-xs text-slate-400">
                    Tiên Khiếu hòa bình. 1 giờ thực sẽ trôi qua {aperture.time_rate / 10} năm Tiên Khiếu.
                  </p>
                </div>
              )}
            </div>
          </div>
        )}

        {/* TAB 2: RESOURCE GRID MAP */}
        {activeTab === "map" && (
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* Lưới Grid Bản Đồ */}
            <div className="lg:col-span-2 bg-slate-900/60 backdrop-blur-md border border-slate-700/40 rounded-2xl p-6 flex flex-col items-center">
              <h2 className="text-xl font-bold text-slate-100 border-b border-slate-800 pb-3 w-full mb-4 flex items-center gap-2">
                <MapIcon className="w-5 h-5 text-amber-400" />
                Lưới Địa Hình Tiên Khiếu
              </h2>

              {/* Grid 20x20 container */}
              <div className="bg-slate-950 p-3 rounded-2xl border border-slate-850 overflow-auto max-w-full">
                <div
                  className="grid gap-1 select-none"
                  style={{ gridTemplateColumns: "repeat(20, minmax(0, 1fr))", width: "fit-content" }}
                >
                  {Array.from({ length: 20 }).map((_, y) =>
                    Array.from({ length: 20 }).map((_, x) => {
                      const coordKey = `${x},${y}`;
                      const node = nodeMap.get(coordKey);
                      const enemy = enemyMap.get(coordKey);
                      const isSelected = selectedCell?.x === x && selectedCell?.y === y;

                      // Determine cell contents / look
                      let cellIcon = null;
                      let cellColor = "bg-slate-900 hover:bg-slate-800 border-slate-800/40";

                      if (enemy) {
                        cellIcon = "💀";
                        cellColor = "bg-rose-950/60 border-rose-500/40 animate-pulse cursor-pointer shadow-[0_0_8px_rgba(239,68,68,0.2)]";
                      } else if (node) {
                        switch (node.node_type) {
                          case "MOON_ORCHID_FIELD":
                            cellIcon = "🌸";
                            cellColor = "bg-fuchsia-950/40 border-fuchsia-500/30";
                            break;
                          case "GREEN_BAMBOO_GROVE":
                            cellIcon = "🎋";
                            cellColor = "bg-emerald-950/40 border-emerald-500/30";
                            break;
                          case "JADE_STONE_MINE":
                            cellIcon = "💎";
                            cellColor = "bg-cyan-950/40 border-cyan-500/30";
                            break;
                          case "PRIMEVAL_MINE":
                            cellIcon = "🪙";
                            cellColor = "bg-amber-950/40 border-amber-500/30";
                            break;
                          default:
                            cellIcon = "🏡";
                            cellColor = "bg-slate-800 border-slate-700";
                        }
                      }

                      if (isSelected) {
                        cellColor += " ring-2 ring-amber-500 border-transparent scale-110 z-10";
                      }

                      return (
                        <div
                          key={coordKey}
                          onClick={() => handleCellClick(x, y)}
                          className={`w-7 h-7 sm:w-8 sm:h-8 flex items-center justify-center text-xs font-bold rounded cursor-pointer border transition-all ${cellColor}`}
                          title={`Tọa độ: (${x}, ${y})`}
                        >
                          {cellIcon}
                        </div>
                      );
                    })
                  )}
                </div>
              </div>
              <div className="flex gap-4 mt-4 text-xs text-slate-400 flex-wrap justify-center">
                <div className="flex items-center gap-1">🌸 Cánh hoa Nguyệt Lan</div>
                <div className="flex items-center gap-1">🎋 Trúc Thanh Trúc</div>
                <div className="flex items-center gap-1">💎 Ngọc Thạch</div>
                <div className="flex items-center gap-1">🪙 Mạch Nguyên Thạch</div>
                <div className="flex items-center gap-1 text-rose-400">💀 Quái Tai Kiếp</div>
              </div>
            </div>

            {/* Chi tiết ô chọn / Xây dựng */}
            <div className="bg-slate-900/60 backdrop-blur-md border border-slate-700/40 rounded-2xl p-6 flex flex-col justify-between">
              <div>
                <h2 className="text-xl font-bold text-slate-100 border-b border-slate-800 pb-3 mb-4 flex items-center gap-2">
                  <Plus className="w-5 h-5 text-cyan-400" />
                  Hành Động Ô Đất
                </h2>

                {selectedCell ? (
                  <div className="space-y-6">
                    <div className="bg-slate-950/50 p-4 rounded-xl border border-slate-850">
                      <div className="text-xs text-slate-400 font-semibold">Tọa độ đang chọn</div>
                      <div className="text-lg font-bold text-amber-300 mt-1">X: {selectedCell.x}, Y: {selectedCell.y}</div>
                    </div>

                    {selectedNode ? (
                      // Node Interaction details
                      <div className="space-y-4">
                        <div className="p-4 bg-slate-950/50 rounded-xl border border-slate-800 space-y-3">
                          <div className="flex justify-between items-center">
                            <h3 className="font-bold text-slate-200">{selectedNode.name}</h3>
                            <span className="text-xs px-2 py-0.5 bg-slate-800 text-slate-400 rounded">Lv. {selectedNode.level}</span>
                          </div>

                          <div className="text-xs text-slate-400">
                            Loại sản sinh: <span className="text-slate-300 font-medium">{selectedNode.production_type === "Material" ? "Vật phẩm" : "Nguyên thạch"}</span>
                          </div>

                          <div className="flex justify-between text-xs text-slate-400">
                            <span>Độ bền: {selectedNode.hp}/{selectedNode.max_hp}</span>
                            {selectedNode.is_destroyed && <span className="text-rose-500 font-bold">Bị phá hủy</span>}
                          </div>

                          {/* Countdown Timer */}
                          {!selectedNode.is_destroyed && (
                            <div className="text-xs text-slate-400 flex justify-between">
                              <span>Thời gian tích tụ:</span>
                              <span className="text-amber-400 font-semibold">
                                <CountdownTimer node={selectedNode} timeRate={aperture?.time_rate || 1.0} />
                              </span>
                            </div>
                          )}

                          {/* Collect timing indicator */}
                          {!selectedNode.is_destroyed && (
                            <div className="pt-2">
                              {canCollect(selectedNode, aperture?.time_rate) ? (
                                <button
                                  onClick={() => handleCollect(selectedNode.id)}
                                  disabled={actionLoading}
                                  className="w-full py-2 bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white font-bold text-sm rounded-lg transition"
                                >
                                  {actionLoading ? "Đang thu hoạch..." : "Thu Hoạch Tài Nguyên"}
                                </button>
                              ) : (
                                <div className="text-center text-xs text-slate-500 border border-slate-800/40 p-2 rounded">
                                  Linh khí đang tích tụ (Chưa đạt mốc thời gian)
                                </div>
                              )}
                            </div>
                          )}

                          {/* Action Buttons: Uninstall and Bulldoze */}
                          <div className="pt-2 border-t border-slate-800 space-y-2">
                            {!selectedNode.is_destroyed && (
                              <button
                                onClick={handleUninstall}
                                disabled={actionLoading}
                                className="w-full py-1.5 bg-slate-805 hover:bg-slate-800 text-slate-300 font-semibold text-xs rounded-lg border border-slate-700 transition"
                                title="Tháo dỡ mỏ tài nguyên về dạng mô hình vật phẩm cất vào túi đồ"
                              >
                                {actionLoading ? "Đang xử lý..." : "Tháo Dỡ (Thu hồi mỏ)"}
                              </button>
                            )}
                            <button
                              onClick={handleDelete}
                              disabled={actionLoading}
                              className="w-full py-1.5 bg-rose-950/40 hover:bg-rose-900/40 text-rose-300 border border-rose-800/50 font-semibold text-xs rounded-lg transition"
                              title="Xóa vĩnh viễn mỏ tài nguyên (không hoàn trả)"
                            >
                              {actionLoading ? "Đang xử lý..." : selectedNode.is_destroyed ? "Dọn phế tích (San phẳng)" : "San Phẳng (Xóa vĩnh viễn)"}
                            </button>
                          </div>
                        </div>
                      </div>
                    ) : (() => {
                      // Build Node menu
                      const maxNodes = Math.min(200, Math.floor((aperture?.area_size || 0) / 100000));
                      const currentNodesCount = apertureNodes.length;
                      const isAtLimit = currentNodesCount >= maxNodes;

                      const qtyMoon = player.materials_inventory?.["node_moon_orchid_field"] || 0;
                      const qtyBamboo = player.materials_inventory?.["node_green_bamboo_grove"] || 0;
                      const qtyJade = player.materials_inventory?.["node_jade_stone_mine"] || 0;
                      const qtyPrimeval = player.materials_inventory?.["node_primeval_mine"] || 0;

                      return (
                        <div className="space-y-4">
                          <div className="text-sm font-semibold text-slate-300">Xây dựng điểm tài nguyên</div>

                          <div className="bg-slate-950/60 p-3 rounded-xl border border-slate-850 flex justify-between items-center text-xs">
                            <span className="text-slate-400">Số lượng điểm tài nguyên:</span>
                            <span className={`${isAtLimit ? 'text-rose-400 font-bold' : 'text-amber-400'} font-bold`}>
                              {currentNodesCount} / {maxNodes} mỏ
                            </span>
                          </div>

                          {isAtLimit && (
                            <div className="p-3 bg-rose-500/10 border border-rose-500/20 rounded-xl text-center text-xs text-rose-300">
                              Quy mô Tiên Khiếu đạt giới hạn mỏ tài nguyên. Hãy tháo dỡ hoặc mở rộng thêm đất đai.
                            </div>
                          )}

                          <div className="space-y-3 pt-2">
                            {/* Moon Orchid Field */}
                            <div className="flex items-center justify-between p-3 bg-slate-950/60 hover:bg-slate-950 border border-slate-850 hover:border-slate-800 rounded-xl transition">
                              <div>
                                <div className="text-sm font-bold text-slate-200">🌸 Cánh Đồng Hoa Nguyệt Lan</div>
                                {qtyMoon > 0 ? (
                                  <div className="text-xs text-emerald-400 font-medium">Có sẵn mô hình (Còn: {qtyMoon}x) | Miễn phí xây dựng</div>
                                ) : (
                                  <div className="text-xs text-slate-400">Phí: 500 Nguyên Thạch | Sản lượng: 10 Cánh Hoa / 4h</div>
                                )}
                              </div>
                              <button
                                onClick={() => handleBuildNode("MOON_ORCHID_FIELD")}
                                disabled={actionLoading || isAtLimit || (qtyMoon <= 0 && player.primeval_stones < 500)}
                                className="px-3 py-1.5 bg-gradient-to-r from-amber-500 to-yellow-600 disabled:from-slate-800 disabled:to-slate-900 disabled:text-slate-500 hover:brightness-110 text-slate-950 font-bold text-xs rounded-lg transition whitespace-nowrap"
                              >
                                {qtyMoon > 0 ? "Đặt" : "Xây"}
                              </button>
                            </div>

                            {/* Green Bamboo Grove */}
                            <div className="flex items-center justify-between p-3 bg-slate-950/60 hover:bg-slate-950 border border-slate-850 hover:border-slate-800 rounded-xl transition">
                              <div>
                                <div className="text-sm font-bold text-slate-200">🎋 Rừng Trúc Thanh Trúc</div>
                                {qtyBamboo > 0 ? (
                                  <div className="text-xs text-emerald-400 font-medium">Có sẵn mô hình (Còn: {qtyBamboo}x) | Miễn phí xây dựng</div>
                                ) : (
                                  <div className="text-xs text-slate-400">Phí: 600 Nguyên Thạch | Sản lượng: 5 Thanh Trúc Tửu / 6h</div>
                                )}
                              </div>
                              <button
                                onClick={() => handleBuildNode("GREEN_BAMBOO_GROVE")}
                                disabled={actionLoading || isAtLimit || (qtyBamboo <= 0 && player.primeval_stones < 600)}
                                className="px-3 py-1.5 bg-gradient-to-r from-amber-500 to-yellow-600 disabled:from-slate-800 disabled:to-slate-900 disabled:text-slate-500 hover:brightness-110 text-slate-950 font-bold text-xs rounded-lg transition whitespace-nowrap"
                              >
                                {qtyBamboo > 0 ? "Đặt" : "Xây"}
                              </button>
                            </div>

                            {/* Jade Stone Mine */}
                            <div className="flex items-center justify-between p-3 bg-slate-950/60 hover:bg-slate-950 border border-slate-850 hover:border-slate-800 rounded-xl transition">
                              <div>
                                <div className="text-sm font-bold text-slate-200">💎 Mỏ Ngọc Thạch Bản Địa</div>
                                {qtyJade > 0 ? (
                                  <div className="text-xs text-emerald-400 font-medium">Có sẵn mô hình (Còn: {qtyJade}x) | Miễn phí xây dựng</div>
                                ) : (
                                  <div className="text-xs text-slate-400">Phí: 1,000 Nguyên Thạch | Sản lượng: 3 Ngọc Thạch / 8h</div>
                                )}
                              </div>
                              <button
                                onClick={() => handleBuildNode("JADE_STONE_MINE")}
                                disabled={actionLoading || isAtLimit || (qtyJade <= 0 && player.primeval_stones < 1000)}
                                className="px-3 py-1.5 bg-gradient-to-r from-amber-500 to-yellow-600 disabled:from-slate-800 disabled:to-slate-900 disabled:text-slate-500 hover:brightness-110 text-slate-950 font-bold text-xs rounded-lg transition whitespace-nowrap"
                              >
                                {qtyJade > 0 ? "Đặt" : "Xây"}
                              </button>
                            </div>

                            {/* Primeval Mine */}
                            <div className="flex items-center justify-between p-3 bg-slate-950/60 hover:bg-slate-950 border border-slate-850 hover:border-slate-800 rounded-xl transition">
                              <div>
                                <div className="text-sm font-bold text-slate-200">🪙 Mạch Nguyên Thạch Tiên Khiếu</div>
                                {qtyPrimeval > 0 ? (
                                  <div className="text-xs text-emerald-400 font-medium">Có sẵn mô hình (Còn: {qtyPrimeval}x) | Miễn phí xây dựng</div>
                                ) : (
                                  <div className="text-xs text-slate-400">Phí: 1,500 Nguyên Thạch | Sản lượng: 200 Nguyên Thạch / 12h</div>
                                )}
                              </div>
                              <button
                                onClick={() => handleBuildNode("PRIMEVAL_MINE")}
                                disabled={actionLoading || isAtLimit || (qtyPrimeval <= 0 && player.primeval_stones < 1500)}
                                className="px-3 py-1.5 bg-gradient-to-r from-amber-500 to-yellow-600 disabled:from-slate-800 disabled:to-slate-900 disabled:text-slate-500 hover:brightness-110 text-slate-950 font-bold text-xs rounded-lg transition whitespace-nowrap"
                              >
                                {qtyPrimeval > 0 ? "Đặt" : "Xây"}
                              </button>
                            </div>
                          </div>
                        </div>
                      );
                    })()}
                  </div>
                ) : (
                  <div className="flex flex-col items-center justify-center py-12 text-slate-500 text-center">
                    <Plus className="w-12 h-12 text-slate-700 mb-2" />
                    <p className="text-sm">Click vào một ô trên bản đồ Tiên Khiếu để xây dựng mỏ hoặc xem thông tin.</p>
                  </div>
                )}
              </div>

              {selectedCell && (
                <button
                  onClick={() => { setSelectedCell(null); setSelectedNode(null); }}
                  className="mt-6 w-full py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-lg text-sm transition"
                >
                  Đóng
                </button>
              )}
            </div>
          </div>
        )}

        {/* TAB 3: DAO MARKS */}
        {activeTab === "dao" && (
          <div className="bg-slate-900/60 backdrop-blur-md border border-slate-700/40 rounded-2xl p-6 space-y-6">
            <h2 className="text-xl font-bold text-slate-100 border-b border-slate-800 pb-3 flex items-center gap-2">
              <Layers className="w-5 h-5 text-amber-400" />
              Thống Kê Đạo Ngân Tiên Khiếu
            </h2>

            <p className="text-sm text-slate-400 leading-relaxed">
              Đạo ngân là dấu vết quy tắc tiên thiên khắc sâu vào Tiên Khiếu khi vượt kiếp nạn. Đạo ngân tích lũy giúp tăng cường vĩnh viễn sát thương và phòng thủ của Cổ sư khi sử dụng thuộc tính tương ứng. Khi vượt quá 10,000 Đạo Ngân hệ sẽ bị giới hạn gia tăng (Soft Cap).
            </p>

            {daoMarks.length === 0 ? (
              <div className="p-8 text-center text-slate-500 border border-slate-850 rounded-xl">
                Chưa sở hữu bất kỳ Đạo Ngân nào. Hãy phòng thủ và tiêu diệt các quái vật tai kiếp để thu được Đạo Ngân đầu tiên!
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {daoMarks.map((mark) => {
                  const multiplier = calculateDaoMarkMultiplier(mark.amount);
                  const isCap = mark.amount > 10000;

                  return (
                    <div
                      key={mark.id}
                      className="p-4 bg-slate-950/60 border border-slate-850 hover:border-slate-800 rounded-xl flex items-center justify-between transition"
                    >
                      <div className="space-y-1">
                        <div className="flex items-center gap-2">
                          <Flame className="w-4 h-4 text-amber-400" />
                          <span className="font-bold text-slate-200 uppercase">{mark.path} Path</span>
                        </div>
                        <div className="text-xs text-slate-400">
                          Tích lũy: <span className="text-amber-400 font-bold">{mark.amount.toLocaleString()}</span> Đạo Ngân
                        </div>
                      </div>

                      <div className="text-right">
                        <div className="text-emerald-400 font-extrabold text-lg flex items-center gap-1 justify-end">
                          <TrendingUp className="w-4 h-4" />
                          x{multiplier.toFixed(1)}
                        </div>
                        <div className="text-[10px] text-slate-500">
                          {isCap ? "Đã chạm Soft Cap (+200% / 10K)" : "Hệ số cường hoá sát thương"}
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        )}

        {/* TAB 4: TRIBULATION DEFENSE */}
        {activeTab === "tribulation" && (
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* Calamity general info / combat actions */}
            <div className="lg:col-span-2 bg-slate-900/60 backdrop-blur-md border border-slate-700/40 rounded-2xl p-6 space-y-6">
              <h2 className="text-xl font-bold text-slate-100 border-b border-slate-800 pb-3 flex items-center gap-2">
                <Skull className="w-5 h-5 text-rose-500 animate-pulse" />
                Kiểm Soát Vượt Kiếp
              </h2>

              {activeTribulation ? (
                <div className="space-y-6">
                  {/* Calamity Details */}
                  <div className="p-4 bg-rose-950/20 border border-rose-500/20 rounded-xl space-y-3">
                    <div className="flex justify-between items-center flex-wrap gap-2">
                      <div>
                        <span className="text-xs text-rose-400 font-bold uppercase tracking-wider">Tai Kiếp Đang Diễn Ra</span>
                        <h3 className="text-lg font-bold text-rose-200 mt-0.5">
                          {activeTribulation.event.tribulation_type === "EARTH_CALAMITY" ? "Địa Tai" : "Thiên Kiếp"}: {activeTribulation.event.path.toUpperCase()} Path
                        </h3>
                      </div>
                      <div className="text-right">
                        <span className="text-xs text-slate-400">Phần Thưởng Đạo Ngân</span>
                        <div className="text-yellow-400 font-extrabold flex items-center gap-1 justify-end text-sm">
                          <Sparkles className="w-4 h-4" />
                          +{activeTribulation.event.reward_dao_marks} Đạo Ngân
                        </div>
                      </div>
                    </div>

                    <div className="flex gap-4 text-xs text-slate-400">
                      <div>Độ khó: <span className="text-rose-400 font-semibold">{activeTribulation.event.difficulty}</span></div>
                      <div>Bắt đầu lúc: <span className="text-slate-300">{new Date(activeTribulation.event.started_at).toLocaleTimeString()} {new Date(activeTribulation.event.started_at).toLocaleDateString()}</span></div>
                    </div>
                  </div>

                  {/* Combat panel */}
                  <div className="space-y-4">
                    <h4 className="font-bold text-slate-200 text-sm">Thi Triển Cổ Trùng Tấn Công</h4>

                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      {/* Enemy Select */}
                      <div className="space-y-2">
                        <label className="text-xs text-slate-400 font-semibold">Chọn Kẻ Địch Mục Tiêu</label>
                        <div className="space-y-2 max-h-48 overflow-y-auto border border-slate-800 rounded-xl p-2 bg-slate-950/50">
                          {activeTribulation.enemies.map((enemy) => {
                            const isSelected = selectedEnemy?.id === enemy.id;
                            return (
                              <div
                                key={enemy.id}
                                onClick={() => setSelectedEnemy(enemy)}
                                className={`p-2.5 rounded-lg text-xs flex justify-between items-center cursor-pointer border transition ${
                                  isSelected
                                    ? "bg-rose-950/40 border-rose-500/40 text-rose-200"
                                    : "bg-slate-900 border-transparent hover:border-slate-800 text-slate-400"
                                }`}
                              >
                                <div>
                                  <div className="font-bold flex items-center gap-1">
                                    {enemy.is_boss ? "👑" : "👾"} {enemy.name}
                                  </div>
                                  <div className="text-[10px] text-slate-500 mt-0.5">Vị trí: ({enemy.x}, {enemy.y})</div>
                                </div>
                                <div className="text-right">
                                  <div className="font-semibold text-rose-400">{enemy.hp} HP</div>
                                  <div className="text-[10px] text-slate-500">Atk: {enemy.attack} | Def: {enemy.defense}</div>
                                </div>
                              </div>
                            );
                          })}
                        </div>
                      </div>

                      {/* Gu Select */}
                      <div className="space-y-2">
                        <label className="text-xs text-slate-400 font-semibold">Chọn Cổ Trùng Sử Dụng</label>
                        <div className="space-y-2 max-h-48 overflow-y-auto border border-slate-800 rounded-xl p-2 bg-slate-950/50">
                          {activeGus.length === 0 ? (
                            <div className="text-center text-slate-500 py-6 text-xs">
                              Không có Cổ trùng nào ở trạng thái ACTIVE (Khả dụng).
                            </div>
                          ) : (
                            activeGus.map((instance) => {
                              const spec = guDictionary[instance.gu_id];
                              if (!spec) return null;
                              const isSelected = selectedGu === instance.gu_id;

                              return (
                                <div
                                  key={instance.instance_id}
                                  onClick={() => setSelectedGu(instance.gu_id)}
                                  className={`p-2 rounded-lg text-xs flex justify-between items-center cursor-pointer border transition ${
                                    isSelected
                                      ? "bg-amber-950/40 border-amber-500/40 text-amber-200"
                                      : "bg-slate-900 border-transparent hover:border-slate-800 text-slate-400"
                                  }`}
                                >
                                  <div>
                                    <div className="font-bold">{spec.name}</div>
                                    <div className="text-[10px] text-slate-500 mt-0.5">
                                      Hệ: {spec.path} | Cấp: {spec.rank}
                                    </div>
                                  </div>
                                  <div className="text-right">
                                    <div className="text-cyan-400 font-semibold">{spec.base_mana_cost} Mana</div>
                                    <div className="text-[10px] text-slate-500">Độ đói: {instance.hunger}%</div>
                                  </div>
                                </div>
                              );
                            })
                          )}
                        </div>
                      </div>
                    </div>

                    <button
                      onClick={handleAttack}
                      disabled={actionLoading || !selectedEnemy || !selectedGu}
                      className="w-full py-3 bg-gradient-to-r from-rose-600 to-amber-600 hover:from-rose-500 hover:to-amber-500 disabled:from-slate-800 disabled:to-slate-900 disabled:text-slate-500 text-white font-bold text-sm rounded-xl transition flex items-center justify-center gap-2"
                    >
                      <Sword className="w-5 h-5" />
                      {actionLoading ? "Đang tiến hành công kích..." : "Phát Động Công Kích"}
                    </button>
                  </div>
                </div>
              ) : (
                <div className="py-12 text-center space-y-4">
                  <Check className="w-16 h-16 text-emerald-500 mx-auto animate-pulse" />
                  <h3 className="text-lg font-bold text-slate-200">Tiên Khiếu Bình Yên</h3>
                  <p className="text-slate-400 max-w-md mx-auto text-xs">
                    Hiện tại Tiên Khiếu Phúc Địa của đạo hữu đang được bảo hộ tuyệt đối bởi quy tắc tiên giới. Tai kiếp chỉ xuất hiện định kỳ khi mốc năm nội bộ được lấp đầy.
                  </p>
                </div>
              )}
            </div>

            {/* Resolve or info summary panel */}
            <div className="bg-slate-900/60 backdrop-blur-md border border-slate-700/40 rounded-2xl p-6 flex flex-col justify-between">
              <div>
                <h2 className="text-xl font-bold text-slate-100 border-b border-slate-800 pb-3 mb-4 flex items-center gap-2">
                  <Shield className="w-5 h-5 text-amber-400" />
                  Kết Toán Vượt Kiếp
                </h2>

                {activeTribulation ? (
                  <div className="space-y-4">
                    <p className="text-xs text-slate-400 leading-relaxed">
                      Để kết thúc tai kiếp, người chơi cần tiêu diệt toàn bộ kẻ thù (Minion & Boss). 
                      Nếu đã tiêu diệt hết, việc kết toán sẽ đảm bảo **Thắng lợi tuyệt đối** và nhận toàn bộ Đạo Ngân.
                    </p>
                    <p className="text-xs text-amber-500 font-medium">
                      ⚠️ Cảnh báo: Nếu bạn cố tình nhấn Kết toán khi vẫn còn kẻ thù còn sống, hệ thống sẽ thực hiện giải quyết khẩn cấp (Offline Resolve) với cơ hội thành công cực thấp (25%-50%), và có nguy cơ bị phá hủy mỏ tài nguyên nặng nề!
                    </p>

                    <button
                      onClick={handleResolve}
                      disabled={actionLoading}
                      className="w-full mt-4 py-3 bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white font-bold text-sm rounded-xl transition shadow-lg"
                    >
                      {actionLoading ? "Đang xử lý kết toán..." : "Yêu Cầu Kết Toán Tai Kiếp"}
                    </button>
                  </div>
                ) : (
                  <div className="flex flex-col items-center justify-center py-12 text-slate-500 text-center">
                    <Shield className="w-12 h-12 text-slate-700 mb-2" />
                    <p className="text-sm">Chưa có Tai Kiếp nào cần kết toán.</p>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

// Simple spinning loader
const Loader: React.FC = () => (
  <svg
    className="animate-spin -ml-1 mr-3 h-10 w-10 text-amber-500"
    xmlns="http://www.w3.org/2000/svg"
    fill="none"
    viewBox="0 0 24 24"
  >
    <circle
      className="opacity-25"
      cx="12"
      cy="12"
      r="10"
      stroke="currentColor"
      strokeWidth="4"
    />
    <path
      className="opacity-75"
      fill="currentColor"
      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
    />
  </svg>
);

// Helper helper for checking if enough time passed to collect node resources
function canCollect(node: ApertureResourceNode, timeRate: number = 1.0): boolean {
  if (!node.last_collected_at) return true;
  const lastTime = new Date(node.last_collected_at).getTime();
  const rate = timeRate <= 0 ? 1.0 : timeRate;
  const nextCollectTime = lastTime + (node.production_interval_hours * 60 * 60 * 1000) / rate;
  return Date.now() >= nextCollectTime;
}

// Calculate Dao Mark multipliers matching Go backend formula
function calculateDaoMarkMultiplier(amount: number): number {
  if (amount <= 10000) {
    return 1.0 + amount / 100.0;
  }
  return 101.0 + (amount - 10000.0) / 500.0;
}
