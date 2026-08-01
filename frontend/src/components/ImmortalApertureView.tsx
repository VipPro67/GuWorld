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
  Users,
  Leaf,
} from "lucide-react";
import { translatePath } from "@/lib/translate";

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
    variantHumans,
    activeTribulation,
    guDictionary,
    itemDictionary,
    actionLoading,
    fetchAperture,
    fetchDaoMarks,
    fetchApertureNodes,
    fetchVariantHumans,
    placeApertureNode,
    collectResourceNode,
    uninstallApertureNode,
    deleteApertureNode,
    setApertureTimeRate,
    setApertureLabor,
    repairApertureNode,
    fetchActiveTribulation,
    attackTribulationEnemy,
    resolveTribulation,
    setView,
    showConfirm,
  } = useGameStore();

  const [activeTab, setActiveTab] = useState<"overview" | "map" | "dao" | "tribulation" | "ecosystem" | "variant_humans">("overview");
  const [selectedCell, setSelectedCell] = useState<{ x: number; y: number } | null>(null);
  const [selectedNode, setSelectedNode] = useState<ApertureResourceNode | null>(null);
  const [selectedEnemy, setSelectedEnemy] = useState<TribulationEnemy | null>(null);
  const [selectedGu, setSelectedGu] = useState<string>("");

  const [sliderRate, setSliderRate] = useState<number>(aperture?.time_rate || 10.0);
  const [buildCategory, setBuildCategory] = useState<"plants" | "beasts" | "humans">("plants");

  const essenceProd = aperture?.essence_production_per_year ?? 10;
  const rank = player?.rank ?? 6;
  const subRank = player?.sub_rank ?? 1;
  const subRankFactor = 1.0 + (Math.min(Math.max(subRank, 1), 4) - 1) * 0.15;
  const timeRate = aperture?.time_rate ?? player?.time_speed_multiplier ?? 10;
  const hourlyRegen = (essenceProd / 120.0) * subRankFactor * timeRate;
  const dailyRegen = Math.round(hourlyRegen * 24);

  const MANA_QUALITY_NAMES: Record<number, string> = {
    6: "Thanh Đề Tiên Nguyên",
    7: "Hồng Tảo Tiên Nguyên",
    8: "Bạch Lệ Tiên Nguyên",
    9: "Hoàng Hạnh Tiên Nguyên",
  };
  const manaName = MANA_QUALITY_NAMES[rank] || "Tiên Nguyên";

  useEffect(() => {
    if (aperture) {
      setSliderRate(aperture.time_rate);
    }
  }, [aperture]);

  useEffect(() => {
    fetchAperture();
    fetchDaoMarks();
    fetchApertureNodes();
    fetchActiveTribulation();
    fetchVariantHumans();
  }, []);

  // Auto-refresh when tribulation is active to show enemy movement
  useEffect(() => {
    if (aperture?.state === "TRIBULATION_ACTIVE") {
      const interval = setInterval(() => {
        fetchApertureNodes();
        fetchActiveTribulation();
        // Also fetch aperture to update aperture HP if taking damage
        fetchAperture();
      }, 15000);
      return () => clearInterval(interval);
    }
  }, [aperture?.state]);

  const handleSetTimeRate = async () => {
    await setApertureTimeRate(sliderRate);
  };

  const handleSetLabor = async (race: string, intensity: string) => {
    await setApertureLabor(race, intensity);
  };

  const handleRepair = async (nodeId: string) => {
    const success = await repairApertureNode(nodeId);
    if (success) {
      setSelectedNode(null);
      setSelectedCell(null);
    }
  };

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
      setActiveTab("map");
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
    if (await showConfirm(`Đạo hữu có chắc chắn muốn tháo dỡ ${selectedNode.name}? Mỏ sẽ được thu hồi thành vật phẩm cất vào túi đồ.`)) {
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
    if (await showConfirm(confirmMsg)) {
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
  const activeGus = player.gu_inventory.filter((gu) => gu.status === "ACTIVE" && gu.equipped === true);

  // Helper to check for path conflict with other nodes
  const checkConflict = (nd: ApertureResourceNode, all: ApertureResourceNode[]) => {
    for (const other of all) {
      if (other.id === nd.id || other.is_destroyed) continue;
      const dx = Math.abs(nd.x - other.x);
      const dy = Math.abs(nd.y - other.y);
      if (dx <= 2 && dy <= 2) {
        const p1 = nd.path.toLowerCase().trim();
        const p2 = other.path.toLowerCase().trim();
        if (p1 !== p2) {
          const key1 = p1 < p2 ? `${p1}:${p2}` : `${p2}:${p1}`;
          const conflicts: Record<string, boolean> = {
            "fire path:water path": true,
            "metal path:wood path": true,
            "earth path:wood path": true,
            "fire path:ice path": true,
          };
          if (conflicts[key1]) return other;
        }
      }
    }
    return null;
  };

  const plantCount = apertureNodes.filter(n => !n.is_destroyed && ["MOON_ORCHID_FIELD", "GREEN_BAMBOO_GROVE", "SWORD_LEAF_GRASS"].includes(n.node_type)).length;
  const herbCount = apertureNodes.filter(n => !n.is_destroyed && n.node_type === "SPIRIT_DEER_PASTURE").length;
  const carnCount = apertureNodes.filter(n => !n.is_destroyed && n.node_type === "SILVER_WOLF_DEN").length;

  const starvationHerb = herbCount > 0 && plantCount === 0;
  const starvationCarn = carnCount > 0 && herbCount === 0;
  const overgrazing = herbCount > plantCount && carnCount === 0 && plantCount > 0;

  // Total Food Production
  const totalFoodProd = apertureNodes.reduce((sum, n) => {
    if (!n.is_destroyed && ["MOON_ORCHID_FIELD", "GREEN_BAMBOO_GROVE", "SWORD_LEAF_GRASS", "SPIRIT_DEER_PASTURE"].includes(n.node_type)) {
      return sum + n.production_amount;
    }
    return sum;
  }, 0);

  // Total Variant Human Population and Food Consumption
  const totalVariantHumanPop = variantHumans.reduce((sum, v) => sum + v.population, 0);
  const foodNeeded = totalVariantHumanPop / 100.0;
  const foodSurplus = totalFoodProd >= foodNeeded;

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
          className={`flex items-center gap-2 px-5 py-3 rounded-lg text-sm font-semibold transition-all whitespace-nowrap relative ${
            activeTab === "map"
              ? "bg-gradient-to-r from-amber-500/20 to-yellow-600/20 border border-amber-500/40 text-amber-300"
              : "text-slate-400 hover:text-slate-200"
          }`}
        >
          <MapIcon className="w-4 h-4" />
          Bản Đồ & Phòng Thủ
          {aperture?.state === "TRIBULATION_ACTIVE" && (
            <span className="absolute top-1 right-1 flex h-2.5 w-2.5">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-rose-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-rose-500"></span>
            </span>
          )}
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
          onClick={() => setActiveTab("ecosystem")}
          className={`flex items-center gap-2 px-5 py-3 rounded-lg text-sm font-semibold transition-all whitespace-nowrap ${
            activeTab === "ecosystem"
              ? "bg-gradient-to-r from-amber-500/20 to-yellow-600/20 border border-amber-500/40 text-amber-300"
              : "text-slate-400 hover:text-slate-200"
          }`}
        >
          <Leaf className="w-4 h-4" />
          Hệ Sinh Thái
        </button>
        <button
          onClick={() => setActiveTab("variant_humans")}
          className={`flex items-center gap-2 px-5 py-3 rounded-lg text-sm font-semibold transition-all whitespace-nowrap ${
            activeTab === "variant_humans"
              ? "bg-gradient-to-r from-amber-500/20 to-yellow-600/20 border border-amber-500/40 text-amber-300"
              : "text-slate-400 hover:text-slate-200"
          }`}
        >
          <Users className="w-4 h-4" />
          Dị Nhân
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
                  <div className="text-slate-400 text-xs font-semibold">Gia Tốc Thời Gian</div>
                  <div className="text-lg font-bold text-cyan-400 mt-1">x{aperture.time_rate} lần</div>
                </div>

                <div className="bg-slate-950/50 p-4 rounded-xl border border-slate-800/80">
                  <div className="text-slate-400 text-xs font-semibold">{manaName} Dự Trữ</div>
                  <div className="text-lg font-bold text-yellow-400 mt-1 flex items-center gap-1">
                    <Coins className="w-5 h-5 text-yellow-500" />
                    {Math.floor(player.current_mana)} viên
                  </div>
                  <div className="text-[10px] text-slate-400 mt-1 font-mono">
                    Hồi phục: +{hourlyRegen.toFixed(1)} viên/giờ
                  </div>
                </div>

                <div className="bg-slate-950/50 p-4 rounded-xl border border-slate-800/80 col-span-2">
                  <div className="text-slate-400 text-xs font-semibold">Diện Tích Đất Đai</div>
                  <div className="text-lg font-bold text-slate-200 mt-1">{(aperture.area_size / 666.67).toFixed(1)} Mẫu (đất cát, rừng hồ...)</div>
                </div>
              </div>

              {/* Sinh Cảnh Tiên Khiếu */}
              <div className="border-t border-slate-800/60 pt-4 space-y-3">
                <h3 className="text-sm font-bold text-slate-350 flex items-center gap-2 uppercase tracking-wider font-mono">
                  🌍 Sinh cảnh & Khí hậu Tiên Khiếu
                </h3>
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs text-left">
                  <div className="bg-slate-950/40 p-3 rounded-xl border border-slate-800/60 flex flex-col justify-between">
                    <span className="text-slate-500 font-mono block mb-1">Nhiệt độ</span>
                    <span className={`text-sm font-bold font-mono ${
                      (aperture.temperature ?? 25.0) > 35 ? 'text-rose-400' :
                      (aperture.temperature ?? 25.0) < 15 ? 'text-blue-400' :
                      'text-emerald-450'
                    }`}>
                      {(aperture.temperature ?? 25.0).toFixed(1)}°C
                    </span>
                  </div>
                  <div className="bg-slate-950/40 p-3 rounded-xl border border-slate-800/60 flex flex-col justify-between">
                    <span className="text-slate-500 font-mono block mb-1">Độ ẩm</span>
                    <span className={`text-sm font-bold font-mono ${
                      (aperture.humidity ?? 50.0) > 70 ? 'text-blue-400' :
                      (aperture.humidity ?? 50.0) < 30 ? 'text-amber-500' :
                      'text-cyan-450'
                    }`}>
                      {(aperture.humidity ?? 50.0).toFixed(1)}%
                    </span>
                  </div>
                  <div className="bg-slate-950/40 p-3 rounded-xl border border-slate-800/60 flex flex-col justify-between">
                    <span className="text-slate-500 font-mono block mb-1">Tiên Khí</span>
                    <span className="text-sm font-bold text-purple-400 font-mono">
                      {Math.round(aperture.qi ?? 100.0)}
                    </span>
                  </div>
                  <div className="bg-slate-950/40 p-3 rounded-xl border border-slate-800/60 flex flex-col justify-between">
                    <span className="text-slate-500 font-mono block mb-1">Địa Hình</span>
                    <span className="text-[11px] font-bold text-amber-300 leading-tight">
                      {aperture.terrain || "Thảo Nguyên Trù Phú"}
                    </span>
                  </div>
                </div>
              </div>

              {/* Integrity HP Bar */}
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-slate-400 font-medium">Độ Ổn Định Linh Khiếu</span>
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

              {/* Time Flow Slider */}
              <div className="bg-slate-950/50 p-6 rounded-xl border border-slate-800 space-y-4">
                {/* <div className="flex justify-between items-center">
                  <div>
                    <h3 className="text-md font-bold text-slate-200 flex items-center gap-1.5">
                      <Hourglass className="w-4 h-4 text-cyan-400" />
                      Điều Chỉnh Dòng Thời Gian
                    </h3>
                    <p className="text-xs text-slate-400 mt-0.5">
                      Gia tốc dòng thời gian giúp sản xuất tài nguyên cực nhanh, nhưng tiêu tốn Thọ Nguyên tương ứng.
                    </p>
                  </div>
                  <span className="text-xl font-bold text-cyan-400">x{sliderRate.toFixed(1)}</span>
                </div> */}

                {/* <div className="flex items-center gap-4">
                  <input
                    type="range"
                    min="1"
                    max="100"
                    step="0.5"
                    value={sliderRate}
                    onChange={(e) => setSliderRate(parseFloat(e.target.value))}
                    className="w-full h-1.5 bg-slate-800 rounded-lg appearance-none cursor-pointer accent-cyan-500"
                  />
                  <button
                    onClick={handleSetTimeRate}
                    disabled={actionLoading || sliderRate === aperture.time_rate}
                    className="px-4 py-1.5 bg-cyan-600 hover:bg-cyan-500 disabled:bg-slate-800 disabled:text-slate-500 text-white font-bold text-xs rounded-lg transition whitespace-nowrap"
                  >
                    Xác Nhận
                  </button>
                </div> */}



                {sliderRate > 30 && (
                  <div className="p-3 bg-amber-500/10 border border-amber-500/20 rounded-lg text-xs text-amber-300 flex items-start gap-1.5">
                    <AlertTriangle className="w-4 h-4 text-amber-400 shrink-0 mt-0.5" />
                    <span>
                      Cảnh báo: Đạo hữu đang tăng tốc dòng thời gian lên mức cao ({sliderRate}x). Điều này sẽ đẩy nhanh Kiếp nạn ập đến!
                    </span>
                  </div>
                )}
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
                  {aperture.earth_calamity_count > 0 && (
                    <div className="flex justify-between items-center text-sm">
                      <span className="text-slate-400">Địa Tai</span>
                      <span className="text-emerald-400 font-semibold">{aperture.earth_calamity_count} lần</span>
                    </div>
                  )}
                  {aperture.heavenly_tribulation_count > 0 && (
                    <div className="flex justify-between items-center text-sm">
                      <span className="text-slate-400">Thiên Kiếp</span>
                      <span className="text-amber-400 font-semibold">{aperture.heavenly_tribulation_count} lần</span>
                    </div>
                  )}
                  {aperture.grand_tribulation_count > 0 && (
                    <div className="flex justify-between items-center text-sm">
                      <span className="text-slate-400">Hạo Kiếp</span>
                      <span className="text-red-400 font-semibold">{aperture.grand_tribulation_count} lần</span>
                    </div>
                  )}
                  {aperture.myriad_tribulation_count > 0 && (
                    <div className="flex justify-between items-center text-sm">
                      <span className="text-slate-400">Vạn Kiếp</span>
                      <span className="text-purple-400 font-semibold">{aperture.myriad_tribulation_count} lần</span>
                    </div>
                  )}
                  {aperture.chaos_calamity_count > 0 && (
                    <div className="flex justify-between items-center text-sm">
                      <span className="text-slate-400">Hỗn Độn Đại Nạn</span>
                      <span className="text-orange-400 font-semibold">{aperture.chaos_calamity_count} lần</span>
                    </div>
                  )}
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
                    onClick={() => setActiveTab("map")}
                    className="w-full py-2 bg-rose-600 hover:bg-rose-500 text-white font-semibold text-sm rounded-lg transition"
                  >
                    Vào Vượt Kiếp
                  </button>
                </div>
              ) : (''
                // <div className="p-4 bg-slate-950/50 border border-slate-850 rounded-xl text-center">
                //   <Check className="w-8 h-8 text-emerald-500 mx-auto mb-2" />
                //   <p className="text-xs text-slate-400">
                //     Tiên Khiếu hòa bình. 1 giờ thực sẽ trôi qua {aperture.time_rate / 10} năm Tiên Khiếu.
                //   </p>
                // </div>
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

                      const checkConflictInJS = (nd: ApertureResourceNode) => {
                        for (const other of apertureNodes) {
                          if (other.id === nd.id || other.is_destroyed) continue;
                          const dx = Math.abs(nd.x - other.x);
                          const dy = Math.abs(nd.y - other.y);
                          if (dx <= 2 && dy <= 2) {
                            const p1 = nd.path.toLowerCase().trim();
                            const p2 = other.path.toLowerCase().trim();
                            if (p1 !== p2) {
                              const key1 = p1 < p2 ? `${p1}:${p2}` : `${p2}:${p1}`;
                              const conflicts: Record<string, boolean> = {
                                "fire path:water path": true,
                                "metal path:wood path": true,
                                "earth path:wood path": true,
                                "fire path:ice path": true,
                              };
                              if (conflicts[key1]) return true;
                            }
                          }
                        }
                        return false;
                      };

                      const hasConflict = node ? checkConflictInJS(node) : false;
                      const isPlant = node ? ["MOON_ORCHID_FIELD", "GREEN_BAMBOO_GROVE", "SWORD_LEAF_GRASS"].includes(node.node_type) : false;
                      const isHerb = node ? node.node_type === "SPIRIT_DEER_PASTURE" : false;
                      const isCarn = node ? node.node_type === "SILVER_WOLF_DEN" : false;

                      const pCount = apertureNodes.filter(n => !n.is_destroyed && ["MOON_ORCHID_FIELD", "GREEN_BAMBOO_GROVE", "SWORD_LEAF_GRASS"].includes(n.node_type)).length;
                      const hCount = apertureNodes.filter(n => !n.is_destroyed && n.node_type === "SPIRIT_DEER_PASTURE").length;
                      const cCount = apertureNodes.filter(n => !n.is_destroyed && n.node_type === "SILVER_WOLF_DEN").length;

                      const overgrazed = isPlant && hCount > pCount && cCount === 0;
                      const starvingHerb = isHerb && pCount === 0;
                      const starvingCarn = isCarn && hCount === 0;
                      const isEcosystemStressed = overgrazed || starvingHerb || starvingCarn;

                      if (enemy) {
                        cellIcon = "💀";
                        cellColor = "bg-rose-950/60 border-rose-500/40 animate-pulse cursor-pointer shadow-[0_0_8px_rgba(239,68,68,0.2)]";
                      } else if (node) {
                        if (node.is_destroyed) {
                          cellIcon = "🪦";
                          cellColor = "bg-slate-950 border-slate-900 opacity-60";
                        } else {
                          switch (node.node_type) {
                            case "MOON_ORCHID_FIELD":
                              cellIcon = "🌸";
                              cellColor = "bg-fuchsia-950/40 border-fuchsia-500/30";
                              break;
                            case "GREEN_BAMBOO_GROVE":
                              cellIcon = "🎋";
                              cellColor = "bg-emerald-950/40 border-emerald-500/30";
                              break;
                            case "SWORD_LEAF_GRASS":
                              cellIcon = "⚔️";
                              cellColor = "bg-slate-800/60 border-slate-500/30";
                              break;
                            case "JADE_STONE_MINE":
                              cellIcon = "💎";
                              cellColor = "bg-cyan-950/40 border-cyan-500/30";
                              break;
                            case "PRIMEVAL_MINE":
                              cellIcon = "🪙";
                              cellColor = "bg-amber-950/40 border-amber-500/30";
                              break;
                            case "SPIRIT_DEER_PASTURE":
                              cellIcon = "🦌";
                              cellColor = "bg-amber-950/30 border-amber-600/30";
                              break;
                            case "SILVER_WOLF_DEN":
                              cellIcon = "🐺";
                              cellColor = "bg-rose-950/30 border-rose-600/30";
                              break;
                            case "HAIRY_MAN_SETTLEMENT":
                              cellIcon = "🧔";
                              cellColor = "bg-indigo-950/40 border-indigo-500/30";
                              break;
                            case "ROCKMAN_TRIBE":
                              cellIcon = "🗿";
                              cellColor = "bg-stone-900 border-stone-600/40";
                              break;
                            case "FEATHERMAN_CANYON":
                              cellIcon = "🦅";
                              cellColor = "bg-sky-950/40 border-sky-500/30";
                              break;
                            default:
                              cellIcon = "🏡";
                              cellColor = "bg-slate-800 border-slate-700";
                          }

                          if (hasConflict) {
                            cellColor += " border-red-500 animate-pulse shadow-[0_0_6px_rgba(239,68,68,0.5)]";
                          } else if (isEcosystemStressed) {
                            cellColor += " border-amber-500 shadow-[0_0_6px_rgba(245,158,11,0.5)]";
                          }
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

            {/* Chi tiết ô chọn / Xây dựng HOẶC Vượt Kiếp */}
            <div className="bg-slate-900/60 backdrop-blur-md border border-slate-700/40 rounded-2xl p-6 flex flex-col justify-between space-y-6">
              {activeTribulation && aperture?.state === "TRIBULATION_ACTIVE" ? (
                // COMBAT AND RESOLVE CONTROLS FOR TRIBULATION
                <div className="space-y-6">
                  {/* Calamity general info / combat actions */}
                  <div className="space-y-4">
                    <h3 className="text-md font-bold text-slate-200 border-b border-slate-800 pb-2 flex items-center gap-2">
                      <Sparkles className="w-4 h-4 text-amber-400" />
                      Chi Tiết Tai Kiếp
                    </h3>

                    <div className="p-4 bg-rose-950/20 border border-rose-500/20 rounded-xl space-y-2 text-xs">
                      <div className="flex justify-between items-center">
                        <span className="font-bold text-rose-300">
                          {activeTribulation.event.tribulation_type === "EARTH_CALAMITY" ? "Địa Tai" : "Thiên Kiếp"}
                        </span>
                        <span className="text-slate-400">Độ khó: <span className="text-rose-400 font-bold">{activeTribulation.event.difficulty}</span></span>
                      </div>
                      <div className="text-sm font-bold text-slate-200 uppercase tracking-wide">
                        {activeTribulation.event.path.toUpperCase()} PATH
                      </div>
                      <div className="text-yellow-400 font-extrabold flex items-center gap-1 pt-1 border-t border-rose-950/50">
                        Phần thưởng: +{activeTribulation.event.reward_dao_marks} Đạo Ngân
                      </div>
                    </div>

                    {/* Target Select Dropdown or small list */}
                    <div className="space-y-3">
                      <div className="text-xs font-bold text-slate-300 flex justify-between items-center">
                        <span>Chọn Mục Tiêu Tấn Công</span>
                        <span className="text-slate-500 font-normal">Còn lại: {activeTribulation.enemies.length} quái</span>
                      </div>

                      <div className="space-y-2 max-h-40 overflow-y-auto border border-slate-800 rounded-xl p-2 bg-slate-950/50">
                        {activeTribulation.enemies.map((enemy) => {
                          const isSelected = selectedEnemy?.id === enemy.id;
                          return (
                            <div
                              key={enemy.id}
                              onClick={() => setSelectedEnemy(enemy)}
                              className={`p-2 rounded-lg text-xs flex justify-between items-center cursor-pointer border transition ${
                                isSelected
                                  ? "bg-rose-950/40 border-rose-500/40 text-rose-200"
                                  : "bg-slate-900 border-transparent hover:border-slate-800 text-slate-400"
                              }`}
                            >
                              <div>
                                <div className="font-bold flex items-center gap-1">
                                  {enemy.is_boss ? "👑" : "👾"} {enemy.name}
                                </div>
                                <div className="text-[10px] text-slate-500">Tọa độ: ({enemy.x}, {enemy.y})</div>
                              </div>
                              <div className="text-right">
                                <div className="font-bold text-rose-400">{enemy.hp} HP</div>
                              </div>
                            </div>
                          );
                        })}
                      </div>
                    </div>

                    {/* Gu Select */}
                    <div className="space-y-2">
                      <label className="text-xs text-slate-300 font-bold">Chọn Cổ Trùng (Chỉ Cổ Đang Trang Bị)</label>
                      <div className="space-y-2 max-h-40 overflow-y-auto border border-slate-800 rounded-xl p-2 bg-slate-950/50">
                        {activeGus.length === 0 ? (
                          <div className="text-center text-slate-500 py-6 text-xs leading-normal">
                            Không tìm thấy Cổ trùng đang trang bị (equipped) và ACTIVE.
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
                                  <div className="text-[10px] text-slate-500">
                                    Cấp: {spec.rank} | {spec.base_mana_cost} Mana
                                  </div>
                                </div>
                                <div className="text-right text-[10px] text-slate-400">
                                  Đói: {instance.hunger}%
                                </div>
                              </div>
                            );
                          })
                        )}
                      </div>
                    </div>

                    <button
                      onClick={handleAttack}
                      disabled={actionLoading || !selectedEnemy || !selectedGu}
                      className="w-full py-3 bg-gradient-to-r from-rose-600 to-amber-600 hover:from-rose-500 hover:to-amber-500 disabled:from-slate-850 disabled:to-slate-900 disabled:text-slate-500 text-white font-bold text-xs rounded-xl transition flex items-center justify-center gap-2 shadow-lg"
                    >
                      <Sword className="w-4 h-4" />
                      {actionLoading ? "Đang tiến hành công kích..." : "Phát Động Công Kích"}
                    </button>
                  </div>

                  {/* Resolve Panel */}
                  <div className="space-y-4 pt-4 border-t border-slate-800">
                    <h3 className="text-md font-bold text-slate-200 flex items-center gap-2">
                      <Shield className="w-4 h-4 text-emerald-400" />
                      Kết Toán Vượt Kiếp
                    </h3>
                    <p className="text-[11px] text-slate-400 leading-relaxed">
                      Để kết thúc tai kiếp hoàn toàn, hãy tiêu diệt toàn bộ kẻ địch trên bản đồ lưới trước khi nhấn kết toán để nhận **Đạo Ngân tối đa**. 
                    </p>
                    <p className="text-[11px] text-amber-500 font-semibold leading-relaxed">
                      ⚠️ Cảnh báo: Kết toán khi còn quái vật sống chỉ có 25-50% cơ hội thắng lợi chật vật (nhận 1/2 quà), và có nguy cơ cao bị phá hủy 1 mỏ tài nguyên!
                    </p>

                    <button
                      onClick={handleResolve}
                      disabled={actionLoading}
                      className="w-full py-3 bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white font-bold text-xs rounded-xl transition shadow-lg"
                    >
                      {actionLoading ? "Đang xử lý..." : "Yêu Cầu Kết Toán Tai Kiếp"}
                    </button>
                  </div>
                </div>
              ) : (
                // STANDARD LAND ACTION PANEL
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
                            Loại sản sinh: <span className="text-slate-300 font-medium">{selectedNode.production_type === "Material" ? "Vật phẩm" : "Nguyên Thạch"}</span>
                          </div>

                          <div className="flex justify-between text-xs text-slate-400">
                            <span>Độ bền: {selectedNode.hp}/{selectedNode.max_hp}</span>
                            {selectedNode.is_destroyed && <span className="text-rose-500 font-bold">Bị phá hủy</span>}
                          </div>

                          {/* Ecosystem details */}
                          {!selectedNode.is_destroyed && (() => {
                            const checkConflict = (nd: ApertureResourceNode) => {
                              for (const other of apertureNodes) {
                                if (other.id === nd.id || other.is_destroyed) continue;
                                const dx = Math.abs(nd.x - other.x);
                                const dy = Math.abs(nd.y - other.y);
                                if (dx <= 2 && dy <= 2) {
                                  const p1 = nd.path.toLowerCase().trim();
                                  const p2 = other.path.toLowerCase().trim();
                                  if (p1 !== p2) {
                                    const key1 = p1 < p2 ? `${p1}:${p2}` : `${p2}:${p1}`;
                                    const conflicts: Record<string, boolean> = {
                                      "fire path:water path": true,
                                      "metal path:wood path": true,
                                      "earth path:wood path": true,
                                      "fire path:ice path": true,
                                    };
                                    if (conflicts[key1]) return true;
                                  }
                                }
                              }
                              return false;
                            };

                            const isConflicting = checkConflict(selectedNode);
                            const isPlant = ["MOON_ORCHID_FIELD", "GREEN_BAMBOO_GROVE", "SWORD_LEAF_GRASS"].includes(selectedNode.node_type);
                            const isHerb = selectedNode.node_type === "SPIRIT_DEER_PASTURE";
                            const isCarn = selectedNode.node_type === "SILVER_WOLF_DEN";
                            const isMine = ["JADE_STONE_MINE", "PRIMEVAL_MINE"].includes(selectedNode.node_type);

                            const pCount = apertureNodes.filter(n => !n.is_destroyed && ["MOON_ORCHID_FIELD", "GREEN_BAMBOO_GROVE", "SWORD_LEAF_GRASS"].includes(n.node_type)).length;
                            const hCount = apertureNodes.filter(n => !n.is_destroyed && n.node_type === "SPIRIT_DEER_PASTURE").length;
                            const cCount = apertureNodes.filter(n => !n.is_destroyed && n.node_type === "SILVER_WOLF_DEN").length;

                            const overgrazed = isPlant && hCount > pCount && cCount === 0;
                            const starvingHerb = isHerb && pCount === 0;
                            const starvingCarn = isCarn && hCount === 0;

                            const pathDaoMarks = daoMarks.find(d => d.path === selectedNode.path)?.amount || 0;
                            const daoBonusPercent = pathDaoMarks * 0.5;

                            const rockmanVH = variantHumans.find(v => v.race === "ROCKMAN");
                            const rockmanPop = rockmanVH ? rockmanVH.population : 0;
                            const rockmanLabor = rockmanVH ? rockmanVH.labor_intensity : "NORMAL";
                            const laborLabels: Record<string, string> = { LOW: "x0.7 (Nhàn)", NORMAL: "x1.0", HIGH: "x1.5", EXPLOIT: "x2.0 (Bóc lột)" };

                            return (
                              <div className="p-3 bg-slate-900/40 rounded-lg border border-slate-805 text-[10px] space-y-1 text-slate-400">
                                <div className="font-bold text-slate-300">Phân Tích Hiệu Suất Sản Lượng:</div>
                                <div>• Đạo ngân bổ trợ ({selectedNode.path}): <span className="text-emerald-400 font-medium">+{daoBonusPercent.toFixed(1)}%</span></div>
                                {isConflicting && <div className="text-rose-400">• Xung đột Đạo ngân: <span className="font-medium">-50% hiệu suất, mất 10% HP/năm</span></div>}
                                {overgrazed && <div className="text-amber-400">• Quá tải gặm nhấm (Ecosystem): <span className="font-medium">-50% hiệu suất, mất 5% HP/năm</span></div>}
                                {starvingHerb && <div className="text-rose-400">• Thiếu cỏ thực vật (Herbivore): <span className="font-medium">-80% hiệu suất</span></div>}
                                {starvingCarn && <div className="text-rose-400">• Thiếu con mồi (Carnivore): <span className="font-medium">-80% hiệu suất</span></div>}
                                {isMine && rockmanPop > 0 && (
                                  <div>
                                    <div>• Dân số Người Đá: <span className="text-cyan-400 font-medium">+{Math.min(50, rockmanPop * 0.1).toFixed(1)}%</span></div>
                                    <div>• Cường độ lao động: <span className="text-amber-400 font-medium">{laborLabels[rockmanLabor]}</span></div>
                                  </div>
                                )}
                              </div>
                            );
                          })()}

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

                          {/* Repair Node Section */}
                          {(selectedNode.hp < selectedNode.max_hp || selectedNode.is_destroyed) && (() => {
                            const repairCost = selectedNode.is_destroyed ? selectedNode.max_hp * 2 : selectedNode.max_hp - selectedNode.hp;
                            const canRepair = player.primeval_stones >= repairCost;
                            return (
                              <div className="pt-2 border-t border-slate-800">
                                <button
                                  onClick={() => handleRepair(selectedNode.id)}
                                  disabled={actionLoading || !canRepair}
                                  className="w-full py-2 bg-gradient-to-r from-cyan-600 to-blue-600 disabled:from-slate-800 disabled:to-slate-900 disabled:text-slate-500 text-white font-bold text-xs rounded-lg transition"
                                >
                                  {actionLoading ? "Đang sửa..." : selectedNode.is_destroyed ? `Tái thiết phế tích (${repairCost} Đá)` : `Sửa chữa độ bền (${repairCost} Đá)`}
                                </button>
                              </div>
                            );
                          })()}

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
                      const qtySword = player.materials_inventory?.["node_sword_leaf_grass"] || 0;
                      const qtyJade = player.materials_inventory?.["node_jade_stone_mine"] || 0;
                      const qtyPrimeval = player.materials_inventory?.["node_primeval_mine"] || 0;
                      const qtyDeer = player.materials_inventory?.["node_spirit_deer_pasture"] || 0;
                      const qtyWolf = player.materials_inventory?.["node_silver_wolf_den"] || 0;
                      const qtyHairy = player.materials_inventory?.["node_hairy_man_settlement"] || 0;
                      const qtyRock = player.materials_inventory?.["node_rockman_tribe"] || 0;
                      const qtyFeather = player.materials_inventory?.["node_featherman_canyon"] || 0;

                      type NodeOption = {
                        id: string;
                        name: string;
                        desc: string;
                        cost: number;
                        qty: number;
                      };

                      const categoryData: Record<string, NodeOption[]> = {
                        plants: [
                          { id: "MOON_ORCHID_FIELD", name: "🌸 Cánh Đồng Hoa Nguyệt Lan", desc: "Sản xuất Nguyệt Lan Hoa. Phí: 500 Đá.", cost: 500, qty: qtyMoon },
                          { id: "GREEN_BAMBOO_GROVE", name: "🎋 Rừng Trúc Thanh Trúc", desc: "Sản xuất Thanh Trúc Rượu. Phí: 600 Đá.", cost: 600, qty: qtyBamboo },
                          { id: "SWORD_LEAF_GRASS", name: "⚔️ Đồng Cỏ Kiếm Diệp", desc: "Sản xuất Cỏ Kiếm Diệp (Kiếm Đạo). Phí: 800 Đá.", cost: 800, qty: qtySword },
                        ],
                        beasts: [
                          { id: "SPIRIT_DEER_PASTURE", name: "🦌 Bãi Chăn Thả Hươu Linh", desc: "Sản sinh Thịt Hươu Linh. Cần Thực vật. Phí: 1200 Đá.", cost: 1200, qty: qtyDeer },
                          { id: "SILVER_WOLF_DEN", name: "🐺 Hang Sói Bạc", desc: "Sản sinh Da Sói Bạc. Cần Thú ăn cỏ. Phí: 2000 Đá.", cost: 2000, qty: qtyWolf },
                        ],
                        humans: [
                          { id: "HAIRY_MAN_SETTLEMENT", name: "🧔 Bộ Lạc Người Lông", desc: "Chiêu mộ Người Lông, tăng Refine. Phí: 3000 Đá.", cost: 3000, qty: qtyHairy },
                          { id: "ROCKMAN_TRIBE", name: "🗿 Bản Doanh Người Đá", desc: "Chiêu mộ Người Đá, tăng sản lượng Mỏ. Phí: 3000 Đá.", cost: 3000, qty: qtyRock },
                          { id: "FEATHERMAN_CANYON", name: "🦅 Hẻm Núi Người Lông Vũ", desc: "Chiêu mộ Người Lông Vũ, giảm sát thương Kiếp. Phí: 3000 Đá.", cost: 3000, qty: qtyFeather },
                          { id: "JADE_STONE_MINE", name: "💎 Mỏ Ngọc Thạch Bản Địa", desc: "Sản xuất Ngọc Thạch. Phí: 1000 Đá.", cost: 1000, qty: qtyJade },
                          { id: "PRIMEVAL_MINE", name: "🪙 Mạch Nguyên Thạch Tiên Khiếu", desc: "Sản sinh Nguyên Thạch cực nhanh. Phí: 1500 Đá.", cost: 1500, qty: qtyPrimeval },
                        ],
                      };

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

                          {/* Category Tabs */}
                          <div className="flex bg-slate-950/80 p-1 rounded-lg border border-slate-800 text-[10px]">
                            {["plants", "beasts", "humans"].map((cat) => (
                              <button
                                key={cat}
                                onClick={() => setBuildCategory(cat as any)}
                                className={`flex-1 py-1.5 font-bold rounded capitalize transition-all ${
                                  buildCategory === cat ? "bg-slate-800 text-amber-400" : "text-slate-400"
                                }`}
                              >
                                {cat === "plants" ? "Thực Vật" : cat === "beasts" ? "Sinh Vật" : "Khai Khoáng & Dị Nhân"}
                              </button>
                            ))}
                          </div>

                          <div className="space-y-3 pt-2 max-h-[350px] overflow-y-auto pr-1">
                            {categoryData[buildCategory]?.map((option) => {
                              const nodeItemID = "node_" + option.id.toLowerCase();
                              const hasModel = option.qty > 0;
                              const canBuild = hasModel || player.primeval_stones >= option.cost;

                              return (
                                <div key={option.id} className="flex items-center justify-between p-3 bg-slate-950/60 hover:bg-slate-950 border border-slate-850 hover:border-slate-800 rounded-xl transition gap-2">
                                  <div className="flex-grow min-w-0">
                                    <div className="text-xs font-bold text-slate-200 truncate">{option.name}</div>
                                    <div className="text-[10px] text-slate-400 mt-0.5 line-clamp-2">{option.desc}</div>
                                    {hasModel && (
                                      <div className="text-[9px] text-emerald-400 font-semibold mt-0.5">Sẵn có mô hình ({option.qty}x) | Miễn phí đặt</div>
                                    )}
                                  </div>
                                  <button
                                    onClick={() => handleBuildNode(option.id)}
                                    disabled={actionLoading || isAtLimit || !canBuild}
                                    className="px-3 py-1.5 bg-gradient-to-r from-amber-500 to-yellow-600 disabled:from-slate-800 disabled:to-slate-900 disabled:text-slate-500 hover:brightness-110 text-slate-950 font-bold text-[10px] rounded-lg transition whitespace-nowrap shrink-0"
                                  >
                                    {hasModel ? "Đặt" : "Xây"}
                                  </button>
                                </div>
                              );
                            })}
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
              )}

              {!activeTribulation && selectedCell && (
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
              Đạo ngân là dấu vết quy tắc tiên thiên khắc sâu vào Tiên Khiếu khi vượt kiếp nạn. Đạo ngân tích lũy giúp tăng cường vĩnh viễn sát thương và phòng thủ của Cổ sư khi sử dụng thuộc tính tương ứng. Khi vượt quá 10,000 Đạo Ngân hệ sẽ bị giới hạn gia tăng.
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



        {/* TAB 5: ECOSYSTEM */}
        {activeTab === "ecosystem" && (
          <div className="space-y-6">
            <div className="bg-slate-900/60 backdrop-blur-md border border-slate-700/40 rounded-2xl p-6 space-y-6">
              <div>
                <h2 className="text-xl font-bold text-slate-100 border-b border-slate-800 pb-3 flex items-center gap-2">
                  <Leaf className="w-5 h-5 text-emerald-400" />
                  Quản Lý Chuỗi Sinh Thái Tiên Khiếu
                </h2>
                <p className="text-sm text-slate-400 mt-2 leading-relaxed">
                  Tiên Khiếu là một hệ sinh thái thu nhỏ. Đạo hữu phải duy trì sự cân bằng giữa các loài Thực vật, Thú ăn cỏ, Thú ăn thịt và Dị nhân để tối ưu hóa sản lượng tài nguyên và tránh tàn phá Tiên Khiếu.
                </p>
              </div>

              {/* Status Banner */}
              {(() => {
                if (starvationHerb) {
                  return (
                    <div className="p-4 bg-rose-500/10 border border-rose-500/30 rounded-xl flex items-start gap-3 text-rose-300">
                      <AlertTriangle className="w-5 h-5 shrink-0 mt-0.5 animate-pulse text-rose-400" />
                      <div>
                        <div className="font-bold text-sm">Thú Ăn Cỏ Đang Chết Đói!</div>
                        <div className="text-xs mt-1">Không có bất kỳ nguồn Thực vật nào đang hoạt động. Bãi Chăn Thả Hươu Linh bị giảm 80% hiệu năng sản sinh tài nguyên. Hãy trồng thêm thực vật ngay lập tức!</div>
                      </div>
                    </div>
                  );
                }
                if (starvationCarn) {
                  return (
                    <div className="p-4 bg-rose-500/10 border border-rose-500/30 rounded-xl flex items-start gap-3 text-rose-300">
                      <AlertTriangle className="w-5 h-5 shrink-0 mt-0.5 animate-pulse text-rose-400" />
                      <div>
                        <div className="font-bold text-sm">Thú Ăn Thịt Đang Chết Đói!</div>
                        <div className="text-xs mt-1">Không có bãi chăn thả Thú ăn cỏ nào khả dụng. Hang Sói Bạc bị giảm 80% hiệu năng sản sinh tài nguyên do cạn kiệt thức ăn. Hãy chăn thả Thú ăn cỏ!</div>
                      </div>
                    </div>
                  );
                }
                if (overgrazing) {
                  return (
                    <div className="p-4 bg-amber-500/10 border border-amber-500/30 rounded-xl flex items-start gap-3 text-amber-300">
                      <AlertTriangle className="w-5 h-5 shrink-0 mt-0.5 animate-pulse text-amber-400" />
                      <div>
                        <div className="font-bold text-sm">Hiện Tượng Quá Tải Gặm Nhấm!</div>
                        <div className="text-xs mt-1">Dân số Thú ăn cỏ vượt quá lượng Thực vật hiện có, và thiếu Thú ăn thịt để kiểm soát. Các nguồn Thực vật bị giảm 50% hiệu suất và sẽ mất 5% độ bền (HP) mỗi năm Tiên Khiếu. Hãy bổ sung Thú ăn thịt hoặc giảm bớt Thú ăn cỏ!</div>
                      </div>
                    </div>
                  );
                }
                if (plantCount > 0 || herbCount > 0 || carnCount > 0) {
                  return (
                    <div className="p-4 bg-emerald-500/10 border border-emerald-500/30 rounded-xl flex items-start gap-3 text-emerald-300">
                      <Check className="w-5 h-5 shrink-0 mt-0.5 text-emerald-400" />
                      <div>
                        <div className="font-bold text-sm">Hệ Sinh Thái Cân Bằng</div>
                        <div className="text-xs mt-1">Chuỗi thức ăn đang vận hành ổn định. {herbCount > 0 && !overgrazing && "Nhờ thú ăn cỏ chăn thả tự nhiên, các nguồn Thực vật nhận được phân hữu cơ giúp gia tăng +25% sản lượng."}</div>
                      </div>
                    </div>
                  );
                }
                return (
                  <div className="p-4 bg-slate-950 border border-slate-800 rounded-xl text-center text-xs text-slate-500">
                    Hệ sinh thái trống rỗng. Hãy xây dựng các điểm tài nguyên Thực vật để khởi đầu sinh mệnh.
                  </div>
                );
              })()}

              {/* Grid Metrics */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div className="bg-slate-950/60 p-4 rounded-xl border border-slate-850 flex items-center justify-between">
                  <div className="space-y-1">
                    <span className="text-xs text-slate-400 font-medium">Thực Vật (Plants)</span>
                    <div className="text-2xl font-black text-emerald-400 flex items-baseline gap-1">
                      {plantCount} <span className="text-xs text-slate-500 font-medium">mỏ</span>
                    </div>
                    <span className="text-[10px] text-slate-500">Nguyệt Lan, Thanh Trúc, Kiếm Diệp</span>
                  </div>
                  <Leaf className="w-10 h-10 text-emerald-500/20 shrink-0" />
                </div>

                <div className="bg-slate-950/60 p-4 rounded-xl border border-slate-850 flex items-center justify-between">
                  <div className="space-y-1">
                    <span className="text-xs text-slate-400 font-medium">Thú Ăn Cỏ (Herbivores)</span>
                    <div className="text-2xl font-black text-amber-400 flex items-baseline gap-1">
                      {herbCount} <span className="text-xs text-slate-500 font-medium">mỏ</span>
                    </div>
                    <span className="text-[10px] text-slate-500">Bãi chăn thả Hươu Linh</span>
                  </div>
                  <span className="text-3xl select-none opacity-20 shrink-0">🦌</span>
                </div>

                <div className="bg-slate-950/60 p-4 rounded-xl border border-slate-850 flex items-center justify-between">
                  <div className="space-y-1">
                    <span className="text-xs text-slate-400 font-medium">Thú Ăn Thịt (Carnivores)</span>
                    <div className="text-2xl font-black text-rose-400 flex items-baseline gap-1">
                      {carnCount} <span className="text-xs text-slate-500 font-medium">mỏ</span>
                    </div>
                    <span className="text-[10px] text-slate-500">Hang Sói Bạc dã hoang</span>
                  </div>
                  <span className="text-3xl select-none opacity-20 shrink-0">🐺</span>
                </div>
              </div>

              {/* Food Chain Diagram */}
              <div className="bg-slate-950/40 border border-slate-850 rounded-xl p-6 space-y-4">
                <h3 className="text-sm font-bold text-slate-200">Sơ Đồ Chuỗi Cung Ứng Thức Ăn</h3>
                
                <div className="flex flex-col md:flex-row items-center justify-center gap-6 py-4">
                  {/* Plant Card */}
                  <div className={`w-full md:w-56 p-4 rounded-xl border text-center space-y-2 ${
                    plantCount === 0 ? "bg-slate-950/40 border-slate-800 text-slate-500" :
                    overgrazing ? "bg-amber-950/20 border-amber-500/40 text-amber-300" :
                    "bg-emerald-950/20 border-emerald-500/40 text-emerald-300"
                  }`}>
                    <Leaf className="w-6 h-6 mx-auto mb-1 opacity-80" />
                    <div className="font-bold text-xs uppercase tracking-wider">Mức 1: Thực Vật</div>
                    <div className="text-lg font-black">{plantCount} Mỏ Hoạt Động</div>
                    <div className="text-[10px] opacity-75">Tạo Lương Thực & Nguyên Liệu</div>
                  </div>

                  {/* Arrow 1 */}
                  <div className="text-slate-600 font-bold text-lg rotate-90 md:rotate-0">➔</div>

                  {/* Herbivore Card */}
                  <div className={`w-full md:w-56 p-4 rounded-xl border text-center space-y-2 ${
                    herbCount === 0 ? "bg-slate-950/40 border-slate-800 text-slate-500" :
                    starvationHerb ? "bg-rose-950/20 border-rose-500/40 text-rose-300" :
                    "bg-amber-950/20 border-amber-500/40 text-amber-300"
                  }`}>
                    <span className="text-xl block mb-1">🦌</span>
                    <div className="font-bold text-xs uppercase tracking-wider">Mức 2: Thú Ăn Cỏ</div>
                    <div className="text-lg font-black">{herbCount} Mỏ Hoạt Động</div>
                    <div className="text-[10px] opacity-75">
                      {starvationHerb ? "Đang đói khát" : "Cần 1+ Thực Vật"}
                    </div>
                  </div>

                  {/* Arrow 2 */}
                  <div className="text-slate-600 font-bold text-lg rotate-90 md:rotate-0">➔</div>

                  {/* Carnivore Card */}
                  <div className={`w-full md:w-56 p-4 rounded-xl border text-center space-y-2 ${
                    carnCount === 0 ? "bg-slate-950/40 border-slate-800 text-slate-500" :
                    starvationCarn ? "bg-rose-950/20 border-rose-500/40 text-rose-300" :
                    "bg-rose-950/10 border-rose-500/30 text-rose-300"
                  }`}>
                    <span className="text-xl block mb-1">🐺</span>
                    <div className="font-bold text-xs uppercase tracking-wider">Mức 3: Thú Ăn Thịt</div>
                    <div className="text-lg font-black">{carnCount} Mỏ Hoạt Động</div>
                    <div className="text-[10px] opacity-75">
                      {starvationCarn ? "Đang đói khát" : "Cần 1+ Thú Ăn Cỏ"}
                    </div>
                  </div>
                </div>
              </div>

              {/* Food & Population Balance Card */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                {/* Food balance info */}
                <div className="bg-slate-950/50 p-5 rounded-xl border border-slate-800 space-y-4">
                  <h3 className="font-bold text-sm text-slate-200">Cung Cấp Lương Thực Dị Nhân</h3>
                  <p className="text-xs text-slate-400">
                    Dị nhân định cư trong Tiên Khiếu cần lương thực từ Thực vật và Thú ăn cỏ để sinh sống. Thiếu lương thực sẽ làm giảm nghiêm trọng Hạnh Phúc (-10/năm) và tăng nguy cơ nổi loạn.
                  </p>

                  <div className="space-y-3 pt-2">
                    <div className="flex justify-between items-center text-xs">
                      <span className="text-slate-400 font-medium">Tổng sản lượng lương thực:</span>
                      <span className="text-emerald-400 font-bold">{totalFoodProd} đơn vị / năm</span>
                    </div>

                    <div className="flex justify-between items-center text-xs">
                      <span className="text-slate-400 font-medium">Tiêu thụ của Dị nhân ({totalVariantHumanPop} người):</span>
                      <span className="text-rose-400 font-bold">{foodNeeded.toFixed(2)} đơn vị / năm</span>
                    </div>

                    <div className="pt-2 border-t border-slate-850 flex justify-between items-center text-xs">
                      <span className="text-slate-400 font-medium">Cân đối lương thực:</span>
                      {foodSurplus ? (
                        <span className="px-2 py-0.5 bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 font-bold rounded">
                          Thặng Dư (+2 Hạnh Phúc/năm)
                        </span>
                      ) : (
                        <span className="px-2 py-0.5 bg-rose-500/10 border border-rose-500/30 text-rose-400 font-bold rounded animate-pulse">
                          Thiếu Thụt (-10 Hạnh Phúc/năm)
                        </span>
                      )}
                    </div>
                  </div>
                </div>

                {/* Conflicts summary */}
                <div className="bg-slate-950/50 p-5 rounded-xl border border-slate-800 space-y-3">
                  <h3 className="font-bold text-sm text-slate-200 flex items-center gap-1.5">
                    <AlertTriangle className="w-4 h-4 text-amber-500" />
                    Cơ Chế Xung Đột Đạo Ngân Địa Hình
                  </h3>
                  <p className="text-xs text-slate-400">
                    Đạo ngân của các thuộc tính đối nghịch đặt cạnh nhau (khoảng cách Chebyshev ≤ 2) sẽ gây bài xích, giảm 50% sản lượng và gây 10% sát thương HP/năm lên các điểm tài nguyên đó.
                  </p>

                  {/* Conflict list */}
                  {(() => {
                    const conflictsList: { n1: ApertureResourceNode; n2: ApertureResourceNode }[] = [];
                    const checkedPairs = new Set<string>();

                    apertureNodes.forEach((n1) => {
                      if (n1.is_destroyed) return;
                      const conflictingOther = checkConflict(n1, apertureNodes);
                      if (conflictingOther) {
                        const pairKey = n1.id < conflictingOther.id ? `${n1.id}:${conflictingOther.id}` : `${conflictingOther.id}:${n1.id}`;
                        if (!checkedPairs.has(pairKey)) {
                          checkedPairs.add(pairKey);
                          conflictsList.push({ n1, n2: conflictingOther });
                        }
                      }
                    });

                    if (conflictsList.length === 0) {
                      return (
                        <div className="p-3 bg-emerald-500/5 border border-emerald-500/10 rounded-lg text-emerald-400 text-xs text-center font-medium">
                          Tuyệt vời! Tiên Khiếu không có xung đột Đạo ngân nào.
                        </div>
                      );
                    }

                    return (
                      <div className="space-y-2 max-h-32 overflow-y-auto pr-1">
                        {conflictsList.map(({ n1, n2 }, idx) => (
                          <div key={idx} className="p-2 bg-rose-950/20 border border-rose-900/30 rounded-lg text-[10px] text-rose-300 flex justify-between items-center gap-2">
                            <span>
                              <strong>{n1.name}</strong> ({n1.x},{n1.y}) ❌ <strong>{n2.name}</strong> ({n2.x},{n2.y})
                            </span>
                            <span className="text-rose-500 font-bold shrink-0">Xung Đột</span>
                          </div>
                        ))}
                      </div>
                    );
                  })()}
                </div>
              </div>
            </div>
          </div>
        )}

        {/* TAB 6: VARIANT HUMANS */}
        {activeTab === "variant_humans" && (
          <div className="space-y-6">
            <div className="bg-slate-900/60 backdrop-blur-md border border-slate-700/40 rounded-2xl p-6 space-y-6">
              <div>
                <h2 className="text-xl font-bold text-slate-100 border-b border-slate-800 pb-3 flex items-center gap-2">
                  <Users className="w-5 h-5 text-amber-400" />
                  Bảng Điều Khiển Bộ Tộc Dị Nhân
                </h2>
                <p className="text-sm text-slate-400 mt-2 leading-relaxed">
                  Nuôi dưỡng, cai quản các tộc Dị nhân định cư trong Tiên Khiếu. Thiết lập cường độ lao động để gia tăng sản xuất hoặc ổn định hạnh phúc của bộ tộc nhằm triệt tiêu nguy cơ nổi loạn.
                </p>
              </div>

              {/* Variant Humans List */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                {(() => {
                  const racesList: {
                    race: 'HAIRY_MAN' | 'ROCKMAN' | 'FEATHERMAN';
                    name: string;
                    desc: string;
                    icon: string;
                    settlementType: string;
                    buffLabel: string;
                  }[] = [
                    {
                      race: "HAIRY_MAN",
                      name: "Tộc Người Lông",
                      desc: "Kẻ lập dị giỏi luyện Cổ trùng. Dân số sinh ra nô lệ giúp giảm thời gian và chi phí luyện Cổ.",
                      icon: "🧔",
                      settlementType: "HAIRY_MAN_SETTLEMENT",
                      buffLabel: "Tạo nguyên liệu: Nô Lệ Người Lông"
                    },
                    {
                      race: "ROCKMAN",
                      name: "Tộc Người Đá",
                      desc: "Tộc nhân sống sâu dưới lòng đất. Dân số gia tăng sản lượng khai khoáng của mỏ Ngọc Thạch & Nguyên Thạch.",
                      icon: "🗿",
                      settlementType: "ROCKMAN_TRIBE",
                      buffLabel: "Sản lượng Quặng Mỏ"
                    },
                    {
                      race: "FEATHERMAN",
                      name: "Tộc Người Lông Vũ",
                      desc: "Bộ tộc tự do có cánh trên lưng. Giúp giảm thiểu sát thương của thiên địa tai kiếp lên Tiên Khiếu.",
                      icon: "🦅",
                      settlementType: "FEATHERMAN_CANYON",
                      buffLabel: "Giảm sát thương Kiếp nạn"
                    }
                  ];

                  return racesList.map((raceInfo) => {
                    const record = variantHumans.find((v) => v.race === raceInfo.race);
                    const population = record ? record.population : 0;
                    const happiness = record ? record.happiness : 70;
                    const oppression = record ? record.oppression : 0;
                    const laborIntensity = record ? record.labor_intensity : "NORMAL";

                    // Calculate settlement counts and max capacity
                    const settlementCount = apertureNodes.filter(
                      (n) => !n.is_destroyed && n.node_type === raceInfo.settlementType
                    ).length;
                    const maxPop = 200 * settlementCount;

                    // rebellion rate matching backend
                    const rebellionRate = (100.0 - happiness + oppression) / 4.0;
                    const showRebellionAlert = happiness < 30.0 || oppression > 70.0;

                    // Calculate Buff Percentage
                    let buffValue = "";
                    if (raceInfo.race === "HAIRY_MAN") {
                      buffValue = `Sản sinh nô lệ luyện Cổ`;
                    } else if (raceInfo.race === "ROCKMAN") {
                      const rockmanBonus = Math.min(50, population * 0.1);
                      buffValue = `+${rockmanBonus.toFixed(1)}% sản lượng`;
                    } else if (raceInfo.race === "FEATHERMAN") {
                      const featherReduction = Math.min(50, population * 0.1);
                      buffValue = `-${featherReduction.toFixed(1)}% HP hao tổn`;
                    }

                    const laborLabels: Record<string, string> = {
                      LOW: "Nhàn Hạ (x0.7)",
                      NORMAL: "Bình Thường (x1.0)",
                      HIGH: "Lao Dịch (x1.5)",
                      EXPLOIT: "Bóc Lột (x2.0)"
                    };

                    const handleLaborChange = async (intensity: string) => {
                      let confirmText = `Đạo hữu muốn đổi cường độ lao động của tộc ${raceInfo.name} thành ${laborLabels[intensity]}?`;
                      if (intensity === "HIGH") {
                        confirmText = `⚠️ CẢNH BÁO: Thiết lập cường độ 'Lao Dịch' (x1.5 sản lượng) sẽ giảm 10 Hạnh phúc và tăng 10 Áp bức của tộc này mỗi năm. Đạo hữu có chắc chắn?`;
                      } else if (intensity === "EXPLOIT") {
                        confirmText = `💥 CẢNH BÁO CỰC KỲ NGUY HIỂM: Thiết lập cường độ 'Bóc Lột' (x2.0 sản lượng) sẽ khiến tộc nhân bị kiệt quệ, giảm 25 Hạnh phúc và tăng 25 Áp bức mỗi năm! Tỷ lệ nổi loạn phá hủy Tiên Khiếu sẽ tăng vọt! Đạo hữu có chắc chắn?`;
                      }
                      
                      if (await showConfirm(confirmText)) {
                        await handleSetLabor(raceInfo.race, intensity);
                      }
                    };

                    return (
                      <div key={raceInfo.race} className="bg-slate-950/60 border border-slate-850 hover:border-slate-800 rounded-2xl p-5 flex flex-col justify-between space-y-5 transition-all">
                        <div className="space-y-3">
                          {/* Name & Icon */}
                          <div className="flex items-center gap-3">
                            <span className="text-3xl select-none">{raceInfo.icon}</span>
                            <div>
                              <h3 className="font-extrabold text-sm text-slate-100">{raceInfo.name}</h3>
                              <span className="text-[10px] text-slate-500">Định cư: {settlementCount} bộ lạc</span>
                            </div>
                          </div>

                          <p className="text-[10px] text-slate-400 leading-relaxed min-h-[40px]">
                            {raceInfo.desc}
                          </p>

                          {/* Population */}
                          <div className="space-y-1.5">
                            <div className="flex justify-between text-xs">
                              <span className="text-slate-400">Dân số:</span>
                              <span className="font-bold text-slate-200">
                                {population} / {maxPop}
                              </span>
                            </div>
                            <div className="w-full bg-slate-900 h-2 rounded-full overflow-hidden border border-slate-800">
                              <div
                                className="h-full bg-cyan-500 transition-all duration-500"
                                style={{ width: `${maxPop > 0 ? Math.min(100, (population / maxPop) * 100) : 0}%` }}
                              />
                            </div>
                            {settlementCount === 0 && (
                              <div className="text-[9px] text-rose-400 font-semibold leading-tight flex items-start gap-1">
                                <AlertTriangle className="w-3.5 h-3.5 shrink-0 mt-0.5" />
                                <span>Chưa có cơ sở định cư! Dân số sẽ hao hụt 10% / năm.</span>
                              </div>
                            )}
                          </div>

                          {/* Happiness & Oppression */}
                          <div className="space-y-2 pt-1">
                            {/* Happiness */}
                            <div className="space-y-1">
                              <div className="flex justify-between text-[10px]">
                                <span className="text-slate-400">Độ Hạnh Phúc:</span>
                                <span className={`font-bold ${happiness < 30 ? "text-rose-400" : happiness > 60 ? "text-emerald-400" : "text-amber-400"}`}>
                                  {happiness.toFixed(1)} / 100
                                </span>
                              </div>
                              <div className="w-full bg-slate-900 h-1.5 rounded-full overflow-hidden">
                                <div
                                  className={`h-full transition-all duration-500 ${happiness < 30 ? "bg-rose-500" : happiness > 60 ? "bg-emerald-500" : "bg-amber-500"}`}
                                  style={{ width: `${happiness}%` }}
                                />
                              </div>
                            </div>

                            {/* Oppression */}
                            <div className="space-y-1">
                              <div className="flex justify-between text-[10px]">
                                <span className="text-slate-400">Độ Áp Bức:</span>
                                <span className={`font-bold ${oppression > 70 ? "text-rose-400" : "text-slate-400"}`}>
                                  {oppression.toFixed(1)} / 100
                                </span>
                              </div>
                              <div className="w-full bg-slate-900 h-1.5 rounded-full overflow-hidden">
                                <div
                                  className={`h-full transition-all duration-500 ${oppression > 70 ? "bg-rose-500" : "bg-slate-500"}`}
                                  style={{ width: `${oppression}%` }}
                                />
                              </div>
                            </div>
                          </div>

                          {/* Rebellion risk indicator */}
                          <div className="p-2.5 bg-slate-900/60 border border-slate-800/80 rounded-lg flex items-center justify-between text-[10px]">
                            <span className="text-slate-400">Nguy cơ nổi loạn:</span>
                            <div className="flex items-center gap-1.5">
                              {showRebellionAlert ? (
                                <span className="flex h-2 w-2 shrink-0">
                                  <span className="animate-ping absolute inline-flex h-2 w-2 rounded-full bg-rose-400 opacity-75"></span>
                                  <span className="relative inline-flex rounded-full h-2 w-2 bg-rose-500"></span>
                                </span>
                              ) : null}
                              <span className={`font-black ${showRebellionAlert ? "text-rose-400 animate-pulse font-extrabold" : "text-amber-400"}`}>
                                {rebellionRate.toFixed(1)}% {showRebellionAlert ? "(Nguy hiểm)" : "(Ổn định)"}
                              </span>
                            </div>
                          </div>
                        </div>

                        {/* Labor Intensity Selector & Global Buffs */}
                        <div className="space-y-4 pt-3 border-t border-slate-850">
                          {/* Selector */}
                          <div className="space-y-1.5">
                            <label className="text-[10px] text-slate-400 font-semibold block">Cường Độ Lao Động</label>
                            <div className="grid grid-cols-2 gap-1 text-[9px] font-bold">
                              {["LOW", "NORMAL", "HIGH", "EXPLOIT"].map((mode) => (
                                <button
                                  key={mode}
                                  onClick={() => handleLaborChange(mode)}
                                  disabled={actionLoading}
                                  className={`py-1.5 px-1 rounded transition border ${
                                    laborIntensity === mode
                                      ? "bg-amber-500/20 border-amber-500/40 text-amber-300"
                                      : "bg-slate-900 border-transparent text-slate-400 hover:text-slate-200"
                                  }`}
                                >
                                  {mode === "LOW" ? "Nhàn Nhã" : mode === "NORMAL" ? "Bình Thường" : mode === "HIGH" ? "Lao Dịch" : "Bóc Lột"}
                                </button>
                              ))}
                            </div>
                          </div>

                          {/* Buff value card */}
                          <div className="p-3 bg-slate-900/40 border border-slate-850 rounded-xl text-[10px] flex items-center justify-between gap-2">
                            <div>
                              <div className="text-slate-400 font-medium">{raceInfo.buffLabel}</div>
                              <div className="text-slate-500 text-[8px] mt-0.5">Sinh hiệu toàn bản đồ</div>
                            </div>
                            <div className="text-emerald-400 font-extrabold text-right shrink-0">{buffValue}</div>
                          </div>
                        </div>
                      </div>
                    );
                  });
                })()}
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
