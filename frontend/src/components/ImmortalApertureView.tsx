import React, { useState, useEffect } from "react";
import { useGameStore, ApertureResourceNode, TribulationEnemy, ImmortalAperture } from "@/store/useGameStore";
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
  RefreshCw,
} from "lucide-react";
import { translatePath } from "@/lib/translate";
import { formatNumber } from "@/lib/utils";
import { Num } from "./Num";
import { GamePanel, GameTabs, GameButton, GameCard, GameModal, GameProgressBar, GameSection } from "@/components/ui";
export function getPathIconAndColor(path: string | undefined, nodeType: string | undefined): { icon: string; color: string } {
  if (nodeType === "WALL") return { icon: "🧱", color: "bg-orange-950/50 border-orange-500/40" };
  if (nodeType === "WATCHTOWER") return { icon: "🗼", color: "bg-blue-950/50 border-blue-500/40" };
  if (nodeType === "GUARDIAN_STATUE") return { icon: "🗿", color: "bg-emerald-950/50 border-emerald-500/40" };
  if (nodeType === "BARRACKS") return { icon: "⚔️", color: "bg-red-950/50 border-red-500/40" };
  if (nodeType === "FORCE_FIELD") return { icon: "🔮", color: "bg-purple-950/50 border-purple-500/40" };

  if (nodeType === "MOON_ORCHID_FIELD") return { icon: "🌸", color: "bg-fuchsia-950/40 border-fuchsia-500/30" };
  if (nodeType === "GREEN_BAMBOO_GROVE") return { icon: "🎋", color: "bg-emerald-950/40 border-emerald-500/30" };
  if (nodeType === "JADE_STONE_MINE") return { icon: "💎", color: "bg-cyan-950/40 border-cyan-500/30" };
  if (nodeType === "PRIMEVAL_MINE") return { icon: "🪙", color: "bg-amber-950/40 border-amber-500/30" };
  if (nodeType === "SPIRIT_DEER_PASTURE") return { icon: "🦌", color: "bg-amber-950/30 border-amber-600/30" };
  if (nodeType === "SILVER_WOLF_DEN") return { icon: "🐺", color: "bg-rose-950/30 border-rose-600/30" };

  const norm = (path || "").toLowerCase().trim();

  if (norm.includes("wisdom") || norm.includes("trí")) return { icon: "🧠", color: "bg-violet-950/40 border-violet-500/30" };
  if (norm.includes("wood") || norm.includes("mộc")) return { icon: "🌱", color: "bg-emerald-950/40 border-emerald-500/30" };
  if (norm.includes("fire") || norm.includes("hỏa")) return { icon: "🔥", color: "bg-red-950/40 border-red-500/30" };
  if (norm.includes("earth") || norm.includes("thổ")) return { icon: "🪨", color: "bg-amber-950/40 border-amber-600/30" };
  if (norm.includes("metal") || norm.includes("kim")) return { icon: "🔩", color: "bg-slate-800/60 border-slate-500/30" };
  if (norm.includes("water") || norm.includes("thủy")) return { icon: "💧", color: "bg-blue-950/40 border-blue-500/30" };
  if (norm.includes("wind") || norm.includes("phong")) return { icon: "🌪️", color: "bg-sky-950/40 border-sky-500/30" };
  if (norm.includes("light") || norm.includes("quang")) return { icon: "☀️", color: "bg-yellow-950/40 border-yellow-500/30" };
  if (norm.includes("dark") || norm.includes("ám")) return { icon: "🌑", color: "bg-zinc-950/40 border-zinc-500/30" };
  if (norm.includes("ice") || norm.includes("băng")) return { icon: "❄️", color: "bg-sky-950/50 border-sky-400/40" };
  if (norm.includes("time") || norm.includes("thời") || norm.includes("trụ")) return { icon: "🕰️", color: "bg-purple-950/40 border-purple-500/30" };
  if (norm.includes("space") || norm.includes("không") || norm.includes("vũ")) return { icon: "🌌", color: "bg-indigo-950/40 border-indigo-500/30" };
  if (norm.includes("blood") || norm.includes("huyết")) return { icon: "🩸", color: "bg-rose-950/40 border-rose-500/30" };
  if (norm.includes("blade") || norm.includes("đao")) return { icon: "🗡️", color: "bg-slate-900 border-slate-600/40" };
  if (norm.includes("sword") || norm.includes("kiếm")) return { icon: "⚔️", color: "bg-slate-800/60 border-slate-500/30" };
  if (norm.includes("food") || norm.includes("thực")) return { icon: "🍖", color: "bg-amber-950/30 border-amber-600/30" };
  if (norm.includes("star") || norm.includes("tinh")) return { icon: "⭐", color: "bg-fuchsia-950/40 border-fuchsia-500/30" };
  if (norm.includes("poison") || norm.includes("độc")) return { icon: "☠️", color: "bg-green-950/40 border-green-500/30" };
  if (norm.includes("human") || norm.includes("nhân")) return { icon: "🧔", color: "bg-indigo-950/40 border-indigo-500/30" };
  if (norm.includes("enslavement") || norm.includes("nô") || norm.includes("thú")) return { icon: "🐄", color: "bg-amber-950/40 border-amber-500/30" };
  if (norm.includes("strength") || norm.includes("lực")) return { icon: "💪", color: "bg-red-950/40 border-red-500/30" };
  if (norm.includes("bone") || norm.includes("cốt")) return { icon: "🦴", color: "bg-stone-950/40 border-stone-500/30" };
  if (norm.includes("transformation") || norm.includes("biến hóa")) return { icon: "🎭", color: "bg-purple-950/40 border-purple-500/30" };
  if (norm.includes("rule") || norm.includes("luật")) return { icon: "⚖️", color: "bg-indigo-950/40 border-indigo-500/30" };
  if (norm.includes("luck") || norm.includes("vận")) return { icon: "🍀", color: "bg-emerald-950/40 border-emerald-500/30" };
  if (norm.includes("refinement") || norm.includes("luyện")) return { icon: "🔨", color: "bg-amber-950/40 border-amber-500/30" };
  if (norm.includes("soul") || norm.includes("hồn")) return { icon: "👻", color: "bg-purple-950/40 border-purple-500/30" };
  if (norm.includes("sound") || norm.includes("âm")) return { icon: "🔊", color: "bg-purple-950/40 border-purple-500/30" };
  if (norm.includes("information") || norm.includes("faith") || norm.includes("tín")) return { icon: "📜", color: "bg-amber-950/40 border-amber-500/30" };
  if (norm.includes("phantom") || norm.includes("ảo") || norm.includes("huyễn")) return { icon: "🌫️", color: "bg-slate-900 border-slate-600/40" };
  if (norm.includes("formation") || norm.includes("trận")) return { icon: "⛩️", color: "bg-cyan-950/40 border-cyan-500/30" };
  if (norm.includes("theft") || norm.includes("thâu")) return { icon: "🗝️", color: "bg-zinc-900 border-zinc-600" };
  if (norm.includes("pill") || norm.includes("đan")) return { icon: "💊", color: "bg-rose-950/40 border-rose-500/30" };
  if (norm.includes("qi") || norm.includes("khí")) return { icon: "💨", color: "bg-sky-950/40 border-sky-500/30" };

  if (nodeType === "FARMING_ZONE") return { icon: "🌱", color: "bg-emerald-950/40 border-emerald-500/30" };
  if (nodeType === "BREEDING_ZONE") return { icon: "🐄", color: "bg-amber-950/40 border-amber-500/30" };
  if (nodeType === "MINING_ZONE") return { icon: "⛏️", color: "bg-cyan-950/40 border-cyan-500/30" };

  return { icon: "📦", color: "bg-slate-800 border-slate-700" };
}

export function getItemZoneCategory(item: { id: string; name: string; item_type?: string; type?: string; description?: string; path?: string }): "khu_trong" | "khu_nuoi" | "khu_khai_thac" {
  const text = `${item.id} ${item.name} ${item.description || ''} ${item.path || ''}`.toLowerCase();

  // 1. Check Trồng Trọt (Farming / Plants / Flowers / Grass / Trees / Herbs / Wine / Seeds / Dew / Wood)
  const farmingKeywords = [
    "trúc", "bamboo", "rượu", "wine", "cỏ", "grass", "thảo", "herb", "hoa", "flower", 
    "cây", "tree", "diệp", "leaf", "lá", "dịch", "fluid", "sâm", "ginseng", 
    "quả", "fruit", "hạt", "seed", "lộ", "dew", "chi", "nấm", "mushroom", 
    "mầm", "sprout", "tảo", "algae", "mạc", "nha", "trà", "tea", "hương", "incense", 
    "họa", "painting", "trục", "scroll", "mực", "ink", "bột màu", "gỗ", "wood", 
    "bạch quả", "liên", "bọt mây", "vân ti", "tơ mây"
  ];
  if (farmingKeywords.some(kw => text.includes(kw))) {
    return "khu_trong";
  }

  // 2. Check Chăn Nuôi (Breeding / Animals / Beast / Meat / Food / Organisms / Insects / Blood / Bone)
  const breedingKeywords = [
    "thịt", "meat", "thú", "beast", "hươu", "deer", "sói", "wolf", "dơi", "bat", 
    "ngư", "fish", "long ngư", "máu", "blood", "huyết", "xương", "bone", "tủy", "marrow",
    "noãn", "egg", "cánh", "wing", "gấu", "bear", "lợn", "boar", "trư", "mãng", "python",
    "lông", "fur", "scale", "vảy", "mật", "gall", "tạng", "gan", "tâm", "tằm", "silkworm", 
    "côn trùng", "insect", "sâu", "worm", "bọ", "hạc", "crane", "cọp", "tiger", 
    "báo", "leopard", "ngưu", "ox", "bull", "xà", "snake", "hạt", "scorpion", 
    "nhện", "spider", "tổ", "nest", "gân", "sinew", "nanh", "tusk", "rùa", "turtle", 
    "long", "dragon", "phượng", "phoenix", "mao nhân", "hairy_man", "vũ nhân", "featherman", 
    "thạch nhân", "rockman", "dị nhân", "tộc", "dạ minh sa", "bức", "nọc độc"
  ];
  if (breedingKeywords.some(kw => text.includes(kw))) {
    return "khu_nuoi";
  }

  // 3. Check Khai Thác (Mining / Minerals / Ores / Metals / Cores / Gem / Oil / Dust / Sand / Ice)
  const miningKeywords = [
    "đá", "stone", "khoáng", "ore", "thạch", "jade", "tinh thiết", "thiết", "iron", 
    "đồng", "copper", "ngọc", "kim", "metal", "hạch", "core", "cát", "sand", 
    "dầu", "oil", "bùn", "mud", "phấn", "dust", "tinh thể", "crystal", "băng", "ice",
    "ma ngọc", "nham", "hỏa dung", "thiết sa", "thiết bì", "bán thạch", "lưu ly", "glass",
    "hóa thạch", "fossil", "kim cương", "diamond", "ngô đồng thiết", "giới thạch", "chân không thạch",
    "diệt thế", "lôi hỏa tương", "hồng hoang trận hạch", "xu đấu trường", "tiền"
  ];
  if (miningKeywords.some(kw => text.includes(kw))) {
    return "khu_khai_thac";
  }

  // Fallback check by item type
  const itemType = (item.item_type || item.type || "").toLowerCase();
  if (itemType === "food") return "khu_nuoi";

  return "khu_khai_thac";
}

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

interface TribulationRealTimeInfoProps {
  aperture: ImmortalAperture;
  timeSpeedMultiplier?: number;
  isCompact?: boolean;
}

const TribulationRealTimeInfo: React.FC<TribulationRealTimeInfoProps> = ({ aperture, timeSpeedMultiplier = 1, isCompact }) => {
  const [timeLeftText, setTimeLeftText] = useState<string>("");
  const [arrivalTimeText, setArrivalTimeText] = useState<string>("");
  const [isPassed, setIsPassed] = useState<boolean>(false);

  useEffect(() => {
    const updateTime = () => {
      if (aperture.state === "TRIBULATION_ACTIVE") {
        setTimeLeftText("Đang diễn ra");
        setArrivalTimeText("Đang chiến đấu");
        setIsPassed(true);
        return;
      }

      const rate = (aperture.time_rate || 1.0) * (timeSpeedMultiplier > 0 ? timeSpeedMultiplier : 1.0);
      const internalYearsLeft = aperture.next_tribulation_year - aperture.internal_years_passed;
      if (internalYearsLeft <= 0) {
        setTimeLeftText("Đang ập đến");
        setArrivalTimeText("Bất cứ lúc nào");
        setIsPassed(true);
        return;
      }

      const remainingRealSeconds = (internalYearsLeft * 36000.0) / rate;
      const updatedAtMs = new Date(aperture.updated_at).getTime();
      const arrivalTimeMs = updatedAtMs + remainingRealSeconds * 1000;
      const now = Date.now();
      const diff = arrivalTimeMs - now;

      if (diff <= 0) {
        setTimeLeftText("Đang ập đến");
        setArrivalTimeText("Bất cứ lúc nào");
        setIsPassed(true);
        return;
      }

      setIsPassed(false);

      const arrivalDate = new Date(arrivalTimeMs);
      const hours = String(arrivalDate.getHours()).padStart(2, '0');
      const minutes = String(arrivalDate.getMinutes()).padStart(2, '0');
      const seconds = String(arrivalDate.getSeconds()).padStart(2, '0');
      const day = String(arrivalDate.getDate()).padStart(2, '0');
      const month = String(arrivalDate.getMonth() + 1).padStart(2, '0');
      const year = arrivalDate.getFullYear();
      
      setArrivalTimeText(`${hours}:${minutes}:${seconds} ${day}/${month}/${year}`);

      const diffHours = Math.floor(diff / (60 * 60 * 1000));
      const diffMinutes = Math.floor((diff % (60 * 60 * 1000)) / (60 * 1000));
      const diffSeconds = Math.floor((diff % (60 * 1000)) / 1000);

      const parts = [];
      if (diffHours > 0) parts.push(`${diffHours}g`);
      if (diffMinutes > 0 || diffHours > 0) parts.push(`${diffMinutes}p`);
      parts.push(`${diffSeconds}s`);
      setTimeLeftText(parts.join(" "));
    };

    updateTime();
    const interval = setInterval(updateTime, 1000);
    return () => clearInterval(interval);
  }, [aperture.next_tribulation_year, aperture.internal_years_passed, aperture.time_rate, aperture.state, aperture.updated_at, timeSpeedMultiplier]);

  if (isCompact) {
    return (
      <div className="flex justify-between items-center text-[10px] text-zinc-400 font-mono">
        <span>Thời gian thực:</span>
        <span className={isPassed ? "text-rose-400 font-bold animate-pulse" : "text-amber-400 font-bold"}>
          {arrivalTimeText} ({timeLeftText})
        </span>
      </div>
    );
  }

  return (
    <div className="space-y-1 bg-slate-950/40 p-3 rounded-lg border border-slate-800/50 mt-2">
      <div className="flex justify-between items-center text-xs">
        <span className="text-slate-400">Thời gian thực tế đến:</span>
        <span className={`font-mono font-semibold ${isPassed ? "text-rose-400 font-bold animate-pulse" : "text-amber-400 font-bold"}`}>
          {arrivalTimeText}
        </span>
      </div>
      <div className="flex justify-between items-center text-xs">
        <span className="text-slate-400">Còn lại :</span>
        <span className={`font-mono font-semibold ${isPassed ? "text-rose-400 font-bold animate-pulse" : "text-cyan-400 font-bold"}`}>
          {timeLeftText}
        </span>
      </div>
    </div>
  );
};

export const ImmortalApertureView: React.FC = () => {
  const {
    player,
    aperture,
    daoMarks,
    apertureNodes,
    variantHumans,
    activeTribulation,
    itemDictionary,
    actionLoading,
    fetchAperture,
    fetchDaoMarks,
    fetchApertureNodes,
    fetchVariantHumans,
    placeApertureNode,
    collectResourceNode,
    collectAllResourceNodes,
    uninstallApertureNode,
    deleteApertureNode,
    setApertureTimeRate,
    setApertureLabor,
    repairApertureNode,
    repairAllApertureNodes,
    upgradeApertureNode,
    fetchActiveTribulation,
    engageTribulationCombat,
    resolveTribulation,
    useItems: consumeItems,
    setView,
    showConfirm,
  } = useGameStore();

  const [activeTab, setActiveTab] = useState<"overview" | "map" | "dao" | "tribulation" | "ecosystem" | "variant_humans">("overview");
  const [selectedCell, setSelectedCell] = useState<{ x: number; y: number } | null>(null);
  const [selectedNode, setSelectedNode] = useState<ApertureResourceNode | null>(null);
  const [showHealMenu, setShowHealMenu] = useState(false);

  const [sliderRate, setSliderRate] = useState<number>(aperture?.time_rate || 10.0);
  const [buildCategory, setBuildCategory] = useState<"khu_trong" | "khu_nuoi" | "khu_khai_thac" | "defense">("khu_trong");
  const [pendingBuildZone, setPendingBuildZone] = useState<string | null>(null);
  const [buildPath, setBuildPath] = useState<string>("Wisdom Path");
  const [buildProdKey, setBuildProdKey] = useState<string>("");
  const [pathSearch, setPathSearch] = useState<string>("");
  const [itemSearch, setItemSearch] = useState<string>("");
  const [showAllZoneItems, setShowAllZoneItems] = useState<boolean>(false);

  const essenceProd = aperture?.essence_production_per_year ?? 10;
  const rank = player?.rank ?? 6;
  const subRank = player?.sub_rank ?? 1;
  const subRankFactor = 1.0 + (Math.min(Math.max(subRank, 1), 4) - 1) * 0.15;
  const timeRate = aperture?.time_rate ?? 10;
  const timeSpeedMultiplier = player?.time_speed_multiplier ?? 1;
  const effectiveTimeRate = timeRate * timeSpeedMultiplier;
  const hourlyRegen = (essenceProd / 120.0) * subRankFactor * effectiveTimeRate;
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

  const [isReloading, setIsReloading] = useState(false);

  useEffect(() => {
    fetchAperture();
    fetchDaoMarks();
    fetchActiveTribulation();
  }, []);

  useEffect(() => {
    // Fetch Dao Marks whenever tab changes to ensure latest data on map, tribulation (phòng thủ), dao tabs
    fetchDaoMarks();

    if (activeTab === "map") {
      fetchApertureNodes();
    } else if (activeTab === "tribulation") {
      fetchActiveTribulation();
    } else if (activeTab === "ecosystem" || activeTab === "variant_humans") {
      fetchVariantHumans();
      fetchApertureNodes();
    }
  }, [activeTab]);

  const handleReloadAll = async () => {
    setIsReloading(true);
    await Promise.all([
      fetchAperture(),
      fetchDaoMarks(),
      fetchApertureNodes(),
      fetchActiveTribulation(),
      fetchVariantHumans(),
    ]);
    setIsReloading(false);
  };

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

  const handleUpgrade = async (nodeId: string) => {
    const success = await upgradeApertureNode(nodeId);
    if (success) {
      // Refresh selected node data
      const updated = apertureNodes.find(n => n.id === nodeId);
      if (updated) setSelectedNode(updated);
    }
  };

  const handleUseHealingItem = async (itemId: string) => {
    const success = await consumeItems([itemId]);
    if (success) {
      await fetchAperture();
      setShowHealMenu(false);
    }
  };

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

    setSelectedCell({ x, y });
    setSelectedNode(node || null);
  };

  const handleBuildNode = async (nodeType: string, customPath?: string, customProdKey?: string) => {
    if (!selectedCell) return;
    const success = await placeApertureNode(nodeType, selectedCell.x, selectedCell.y, customPath, customProdKey);
    if (success) {
      setSelectedCell(null);
      setSelectedNode(null);
      setPendingBuildZone(null);
    }
  };

  const handleCollect = async (nodeId: string) => {
    const success = await collectResourceNode(nodeId);
    if (success) {
      setSelectedCell(null);
      setSelectedNode(null);
    }
  };

  const handleCollectAll = async () => {
    const success = await collectAllResourceNodes();
    if (success) {
      setSelectedCell(null);
      setSelectedNode(null);
    }
  };

  const handleRepairAll = async () => {
    const success = await repairAllApertureNodes();
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

  const handleEngage = async () => {
    if (!activeTribulation) return;
    await engageTribulationCombat(activeTribulation.event.id);
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

  const plantCount = apertureNodes.filter(n => !n.is_destroyed && 
    ["MOON_ORCHID_FIELD", "GREEN_BAMBOO_GROVE", "SWORD_LEAF_GRASS",
     "TRI_FARM", "TRI_FARM_ADV", "HOA_FARM", "TIN_FARM", "AM_FARM", "DOC_FARM",
     "BAC_FARM", "QUANG_FARM", "KIEM_FARM", "THUY_FARM", "CHUA_FARM", "CAM_FARM",
     "KIM_FARM", "VANG_FARM", "HUYET_FARM", "THAT_FARM", "THAY_THUAT_FARM", "LUAT_FARM",
     "THUAT_FARM", "THOI_THUAT_FARM", "LUC_FARM", "THOI_FARM", "PHONG_FARM",
     "NO_FARM", "NO_FARM_ADV"].includes(n.node_type)
  ).length;
  const herbCount = apertureNodes.filter(n => !n.is_destroyed && n.node_type === "SPIRIT_DEER_PASTURE").length;
  const carnCount = apertureNodes.filter(n => !n.is_destroyed && 
    ["SILVER_WOLF_DEN", "FEATHERMAN_CANYON", "HAIRY_MAN_SETTLEMENT", "ROCKMAN_TRIBE"].includes(n.node_type)
  ).length;

  const starvationHerb = herbCount > 0 && plantCount === 0;
  const starvationCarn = carnCount > 0 && herbCount === 0;
  const overgrazing = herbCount > plantCount && carnCount === 0 && plantCount > 0;

  // Bulk action helpers
  const bulkCollectableNodes = apertureNodes.filter(n => !n.is_destroyed && n.node_category !== "DEFENSE" && canCollect(n, aperture?.time_rate ?? 1.0));
  const bulkReadyCount = bulkCollectableNodes.length;
  const bulkDamagedNodes = apertureNodes.filter(n => n.is_destroyed || n.hp < n.max_hp);
  const bulkRepairCount = bulkDamagedNodes.length;
  const bulkRepairCost = bulkDamagedNodes.reduce((sum, n) => {
    const baseCost = n.is_destroyed ? Math.ceil((n.max_hp * 2) / 50) : Math.ceil((n.max_hp - n.hp) / 50);
    return sum + Math.ceil(baseCost * 1.3);
  }, 0);
  const canAffordBulkRepair = player.immortal_stones >= bulkRepairCost;

  // Total Food Production
  const totalFoodProd = apertureNodes.reduce((sum, n) => {
    if (!n.is_destroyed && 
        ["MOON_ORCHID_FIELD", "GREEN_BAMBOO_GROVE", "SWORD_LEAF_GRASS", "SPIRIT_DEER_PASTURE",
         "TRI_FARM", "TRI_FARM_ADV", "HOA_FARM", "TIN_FARM", "AM_FARM", "DOC_FARM",
         "BAC_FARM", "QUANG_FARM", "KIEM_FARM", "THUY_FARM", "CHUA_FARM", "CAM_FARM",
         "KIM_FARM", "VANG_FARM", "HUYET_FARM", "THAT_FARM", "THAY_THUAT_FARM", "LUAT_FARM",
         "THUAT_FARM", "THOI_THUAT_FARM", "LUC_FARM", "THOI_FARM", "PHONG_FARM",
         "NO_FARM", "NO_FARM_ADV", "SILVER_WOLF_DEN", "FEATHERMAN_CANYON", "HAIRY_MAN_SETTLEMENT", "ROCKMAN_TRIBE"].includes(n.node_type)
    ) {
      return sum + n.production_amount;
    }
    return sum;
  }, 0);

  // Total Variant Human Population and Food Consumption
  const totalVariantHumanPop = variantHumans.reduce((sum, v) => sum + v.population, 0);
  const foodNeeded = totalVariantHumanPop / 100.0;
  const foodSurplus = totalFoodProd >= foodNeeded;

return (
    <GamePanel variant="default" className="w-full mx-auto space-y-6">
      {/* Header Panel */}
      <div className="relative overflow-hidden bg-slate-900/80 backdrop-blur-md border border-slate-700/50 rounded-2xl p-6 shadow-xl">
        <div className="absolute top-0 right-0 w-64 h-64 bg-amber-500/5 rounded-full blur-3xl -z-10" />
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <h1 className="text-2xl sm:text-3xl font-extrabold bg-gradient-to-r from-amber-200 via-yellow-400 to-amber-500 bg-clip-text text-transparent flex items-center gap-2">
              <Sparkles className="w-6 h-6 sm:w-8 sm:h-8 text-amber-400" />
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
              onClick={handleReloadAll}
              disabled={isReloading}
              className="px-3 py-2 bg-slate-800 hover:bg-slate-700 text-slate-200 text-sm font-medium rounded-xl border border-slate-700 transition disabled:opacity-50"
              title="Tải lại dữ liệu"
            >
              <RefreshCw className={`w-4 h-4 ${isReloading ? "animate-spin" : ""}`} />
            </button>
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
      <GameTabs
        items={[
          { id: "overview", label: "Tổng Quan", icon: <Activity className="w-4 h-4 shrink-0" />, disabled: false },
          { id: "map", label: "Bản Đồ & Phòng Thủ", icon: <MapIcon className="w-4 h-4 shrink-0" />, disabled: false, badge: aperture?.state === "TRIBULATION_ACTIVE" ? <span className="absolute top-0 right-0 flex h-2.5 w-2.5"><span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-rose-400 opacity-75"></span><span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-rose-500"></span></span> : undefined },
          { id: "dao", label: "Bảng Đạo Ngân", icon: <Layers className="w-4 h-4 shrink-0" />, disabled: false },
          { id: "ecosystem", label: "Hệ Sinh Thái", icon: <Leaf className="w-4 h-4 shrink-0" />, disabled: false },
          { id: "variant_humans", label: "Dị Nhân", icon: <Users className="w-4 h-4 shrink-0" />, disabled: false },
        ]}
        activeId={activeTab}
        onChange={(id) => setActiveTab(id as any)}
        variant="pills"
        size="sm"
        scrollable={true}
      >
      </GameTabs>

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
                  <div className="text-slate-400 text-xs font-semibold">Tốc Độ Cá Nhân</div>
                  <div className="text-lg font-bold text-violet-400 mt-1">x{timeSpeedMultiplier.toFixed(2)} lần</div>
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


              {/* Time Flow Slider */}
              {/* <div className="bg-slate-950/50 p-6 rounded-xl border border-slate-800 space-y-4">
                <div className="flex justify-between items-center">
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
                </div> 

                <div className="flex items-center gap-4">
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
                </div>



                {sliderRate > 30 && (
                  <div className="p-3 bg-amber-500/10 border border-amber-500/20 rounded-lg text-xs text-amber-300 flex items-start gap-1.5">
                    <AlertTriangle className="w-4 h-4 text-amber-400 shrink-0 mt-0.5" />
                    <span>
                      Cảnh báo: Đạo hữu đang tăng tốc dòng thời gian lên mức cao ({sliderRate}x). Điều này sẽ đẩy nhanh Kiếp nạn ập đến!
                    </span>
                  </div>
                )}
              </div> */}
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
                    <span className="text-slate-200 font-semibold">{Math.floor(aperture.internal_years_passed)} năm</span>
                  </div>
                  <div className="flex justify-between items-center text-sm">
                    <span className="text-slate-400">Mốc tai kiếp kế tiếp</span>
                    <span className="text-rose-400 font-semibold">{Math.round(aperture.next_tribulation_year)} năm</span>
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
                  <TribulationRealTimeInfo aperture={aperture} timeSpeedMultiplier={player?.time_speed_multiplier ?? 1} isCompact={false} />
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
              ) : aperture.state === "TRIBULATION_PENDING" ? (
                <div className="p-4 bg-amber-500/10 border border-amber-500/30 rounded-xl text-center space-y-2">
                  <Hourglass className="w-8 h-8 text-amber-400 mx-auto" />
                  <p className="text-sm text-amber-200 font-medium">
                    Tai kiếp đang chờ. Vì đạo hữu vắng mặt, quái tai kiếp chưa xuất hiện để phá hoại — chúng sẽ giáng xuống khi bạn online. Trì hoãn quá 24 giờ sẽ làm giảm phần thưởng Đạo Ngân.
                  </p>
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
        {activeTab === "map" && aperture && (
          <div className="space-y-6">
            {/* Aperture Stability Card */}
            <div className="bg-slate-900/60 backdrop-blur-md border border-slate-700/40 rounded-2xl p-6 relative z-50">
              <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
                <div className="flex-1 space-y-2">
                  <div className="flex justify-between text-sm">
                    <span className="text-slate-400 font-medium flex items-center gap-1.5">
                      <Shield className="w-4 h-4 text-emerald-400" />
                      Độ Ổn Định Linh Khiếu
                    </span>
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

                {/* Healing Button */}
                {aperture.aperture_hp < aperture.max_aperture_hp && (() => {
                  const healingItems = Object.entries(player.materials_inventory || {})
                    .filter(([, qty]) => qty > 0)
                    .map(([itemId, qty]) => ({
                      itemId,
                      qty,
                      spec: itemDictionary[itemId],
                    }))
                    .filter(item => item.spec && item.spec.effects && item.spec.effects.heal_aperture !== undefined);

                  if (healingItems.length === 0) return null;

                  return (
                    <div className="relative shrink-0">
                      {healingItems.length === 1 ? (
                        <button
                          onClick={() => handleUseHealingItem(healingItems[0].itemId)}
                          disabled={actionLoading}
                          className="px-4 py-2.5 bg-gradient-to-r from-emerald-500 to-teal-600 hover:brightness-110 text-slate-950 font-bold text-xs rounded-lg transition flex items-center gap-2 shadow-lg shadow-emerald-950/20"
                        >
                          <Heart className="w-3.5 h-3.5 fill-current text-slate-950" />
                          Hồi Phục ({healingItems[0].spec.name})
                        </button>
                      ) : (
                        <div>
                          <button
                            onClick={() => setShowHealMenu(!showHealMenu)}
                            disabled={actionLoading}
                            className="px-4 py-2.5 bg-gradient-to-r from-emerald-500 to-teal-600 hover:brightness-110 text-slate-950 font-bold text-xs rounded-lg transition flex items-center gap-2 shadow-lg shadow-emerald-950/20"
                          >
                            <Heart className="w-3.5 h-3.5 fill-current text-slate-950" />
                            Hồi Phục Linh Khiếu
                          </button>
                          {showHealMenu && (
                            <div className="absolute right-0 mt-2 w-64 bg-slate-900 border border-slate-750 rounded-xl shadow-2xl p-2 z-20 space-y-1">
                              <div className="text-[10px] text-slate-500 font-semibold px-2 py-1 uppercase tracking-wider">Chọn dược phẩm</div>
                              {healingItems.map(item => (
                                <button
                                  key={item.itemId}
                                  onClick={() => handleUseHealingItem(item.itemId)}
                                  disabled={actionLoading}
                                  className="w-full text-left px-2 py-2 hover:bg-slate-800 rounded-lg text-xs flex justify-between items-center transition text-slate-200"
                                >
                                  <div>
                                    <div className="font-bold">{item.spec.name}</div>
                                    <div className="text-[10px] text-slate-400">Số lượng: {item.qty}</div>
                                  </div>
                                  <div className="text-emerald-400 font-bold">+{item.spec.effects?.heal_aperture} HP</div>
                                </button>
                              ))}
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  );
                })()}
              </div>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* Lưới Grid Bản Đồ */}
            <div className="lg:col-span-2 bg-slate-900/60 backdrop-blur-md border border-slate-700/40 rounded-2xl p-6 flex flex-col items-center">
              <h2 className="text-xl font-bold text-slate-100 border-b border-slate-800 pb-3 w-full mb-4 flex items-center gap-2">
                <MapIcon className="w-5 h-5 text-amber-400" />
                Lưới Địa Hình Tiên Khiếu
              </h2>

              {/* Bulk actions */}
              <div className="w-full flex flex-col sm:flex-row gap-2 mb-4">
                <button
                  onClick={handleCollectAll}
                  disabled={actionLoading || bulkReadyCount === 0}
                  className="flex-1 py-2 bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 disabled:from-slate-800 disabled:to-slate-900 disabled:text-slate-500 text-white font-bold text-xs rounded-lg transition"
                  title="Thu hoạch mọi mỏ đã đủ thời gian tích tụ (sản lượng -20%, Nhân Khí +1/mỏ)"
                >
                  {actionLoading ? "Đang xử lý..." : `Thu Hoạch Tất Cả (${bulkReadyCount} mỏ, −20%)`}
                </button>
                <button
                  onClick={handleRepairAll}
                  disabled={actionLoading || bulkRepairCount === 0 || !canAffordBulkRepair}
                  className="flex-1 py-2 bg-gradient-to-r from-cyan-600 to-blue-600 hover:from-cyan-500 hover:to-blue-500 disabled:from-slate-800 disabled:to-slate-900 disabled:text-slate-500 text-white font-bold text-xs rounded-lg transition"
                  title="Sửa chữa mọi công trình hư tổn (phí +30%)"
                >
                  {actionLoading ? "Đang xử lý..." : `Sửa Chữa Tất Cả (${bulkRepairCount} công trình, ${bulkRepairCost.toLocaleString()} Đá)`}
                </button>
              </div>

              {/* Grid 20x20 container */}
              <div
                className="bg-slate-950/90 p-3 rounded-2xl border border-slate-800/60 overflow-auto max-w-full shadow-[inset_0_4px_20px_rgba(0,0,0,0.6)] relative"
                style={{
                  backgroundImage: "radial-gradient(circle at center, rgba(16, 185, 129, 0.08) 0%, rgba(30, 41, 59, 0.15) 50%, rgba(15, 23, 42, 0.9) 100%)",
                }}
              >
                <div
                  className="grid gap-1 select-none mx-auto"
                  style={{
                    gridTemplateColumns: "repeat(20, minmax(28px, 1fr))",
                    width: "fit-content",
                  }}
                >
                  {Array.from({ length: 20 }).map((_, y) =>
                    Array.from({ length: 20 }).map((_, x) => {
                      const coordKey = `${x},${y}`;
                      const node = nodeMap.get(coordKey);
                      const enemy = enemyMap.get(coordKey);
                      const isSelected = selectedCell?.x === x && selectedCell?.y === y;

                      // Determine cell contents / look
                      let cellIcon = null;
                      let cellColor = "bg-slate-900/60 hover:bg-slate-800 border-slate-800/80";

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
                          const res = getPathIconAndColor(node.path, node.node_type);
                          cellIcon = res.icon;
                          cellColor = res.color;

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
                          className={`w-full aspect-square flex items-center justify-center text-sm font-bold rounded cursor-pointer border transition-all ${cellColor}`}
                          title={`Tọa độ: (${x}, ${y})`}
                          style={{ minWidth: "28px" }}
                        >
                          {cellIcon}
                        </div>
                      );
                    })
                  )}
                </div>
              </div>

              {/* Dynamic Map Legend */}
              <div className="flex gap-3 mt-4 text-xs text-slate-400 flex-wrap justify-center">
                {(() => {
                  const activePaths = Array.from(
                    new Set(apertureNodes.filter(n => !n.is_destroyed && n.node_category !== "DEFENSE" && n.path).map(n => n.path))
                  );
                  
                  const defaultLegendPaths = [
                    "Wood Path", "Fire Path", "Earth Path", "Metal Path", "Water Path",
                    "Wind Path", "Wisdom Path", "Sword Path", "Blood Path", "Poison Path"
                  ];

                  const displayPaths = activePaths.length > 0 ? activePaths : defaultLegendPaths;

                  return (
                    <>
                      {displayPaths.map(p => {
                        const res = getPathIconAndColor(p, undefined);
                        return (
                          <div key={p} className="flex items-center gap-1">
                            <span>{res.icon}</span>
                            <span>{translatePath(p)}</span>
                          </div>
                        );
                      })}
                      <div className="flex items-center gap-1 text-rose-400">💀 Quái Tai Kiếp</div>
                      <div className="flex items-center gap-1 text-orange-400">🧱 Tường Thành</div>
                      <div className="flex items-center gap-1 text-blue-400">🗼 Tháp Canh</div>
                      <div className="flex items-center gap-1 text-emerald-400">🗿 Tượng Hộ Mệnh</div>
                    </>
                  );
                })()}
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

                    {/* Enemies summary + engage */}
                    <div className="space-y-3">
                      <div className="text-xs font-bold text-slate-300 flex justify-between items-center">
                        <span>Quái Tai Kiếp</span>
                        <span className="text-slate-500 font-normal">Còn lại: {activeTribulation.enemies.length} quái</span>
                      </div>

                      <div className="space-y-2 max-h-40 overflow-y-auto border border-slate-800 rounded-xl p-2 bg-slate-950/50">
                        {activeTribulation.enemies.map((enemy) => (
                          <div
                            key={enemy.id}
                            className="p-2 rounded-lg text-xs flex justify-between items-center bg-slate-900 border border-transparent text-slate-400"
                          >
                            <div>
                              <div className="font-bold flex items-center gap-1">
                                {enemy.is_boss ? "👑" : "👾"} {enemy.name}
                              </div>
                              <div className="text-[10px] text-slate-500 flex items-center gap-2 flex-wrap mt-0.5">
                                <span>Tọa độ: ({enemy.x}, {enemy.y})</span>
                                <span className="text-rose-400 font-semibold">⚔️ Công: <Num value={enemy.attack} /></span>
                                <span className="text-emerald-400 font-semibold">🛡️ Thủ: <Num value={enemy.defense} /></span>
                              </div>
                            </div>
                            <div className="text-right">
                              <div className="font-bold text-rose-400"><Num value={enemy.hp} /> HP</div>
                            </div>
                          </div>
                        ))}
                      </div>

                      <p className="text-[11px] text-amber-500 font-semibold leading-relaxed">
                        Xuất trận sẽ mở giao diện chiến đấu thông thường. Giáp và máu của đạo hữu tiêu hao trước; sát thương tràn qua mới làm tổn hại Tiên Khiếu.
                      </p>

                      <button
                        onClick={handleEngage}
                        disabled={actionLoading}
                        className="w-full py-3 bg-gradient-to-r from-rose-600 to-amber-600 hover:from-rose-500 hover:to-amber-500 disabled:from-slate-850 disabled:to-slate-900 disabled:text-slate-500 text-white font-bold text-xs rounded-xl transition flex items-center justify-center gap-2 shadow-lg"
                      >
                        <Sword className="w-4 h-4" />
                        {actionLoading ? "Đang xuất trận..." : "Xuất Trận Chiến Đấu"}
                      </button>
                    </div>
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
                    <p className="text-[11px] text-cyan-400 font-semibold leading-relaxed">
                      Lưu ý Đột Phá: Đảm bảo Cảnh Giới Lưu Phái đạt yêu cầu (Thất: Tông Sư, Bát: Đại Tông Sư, Cửu: Vô Thượng Đại Tông Sư ). Nếu thiếu, việc vượt kiếp vẫn thành công nhưng sẽ KHÔNG thể thăng Chuyển tu vi dù đủ số lần tai kiếp.
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
                  <h2 className="text-sm font-bold text-slate-100 border-b border-slate-800/80 pb-1.5 mb-2.5 flex items-center gap-1.5">
                    <Plus className="w-4 h-4 text-cyan-400" />
                    Hành Động Ô Đất
                  </h2>

                {selectedCell ? (
                  <div className="space-y-3">
                    <div className="bg-slate-950/60 px-3 py-1.5 rounded-lg border border-slate-850 flex items-center justify-between text-xs">
                      <span className="text-slate-400 font-medium">Tọa độ đang chọn:</span>
                      <span className="font-bold text-amber-300">X: {selectedCell.x}, Y: {selectedCell.y}</span>
                    </div>

                    {selectedNode ? (
                      // Node Interaction details
                      <div className="space-y-4">
                        <div className="p-4 bg-slate-950/50 rounded-xl border border-slate-800 space-y-3">
                          <div className="flex justify-between items-center">
                            <h3 className="font-bold text-slate-200">{selectedNode.name}</h3>
                            <span className="text-xs px-2 py-0.5 bg-slate-800 text-slate-400 rounded">Lv. {selectedNode.level}</span>
                          </div>

                          <div className="flex items-center gap-2 text-xs text-slate-400">
                            <span>Khu: </span>
                            <span className="text-slate-300 font-medium capitalize">{selectedNode.zone_type || "unknown"}</span>
                            <span className="text-slate-500">|</span>
                            <span>Tier: </span>
                            <span className="text-slate-300 font-medium">{selectedNode.item_tier || "T1"}</span>
                            <span className="text-slate-500">|</span>
                            <span>Food Path: </span>
                            <span className="text-slate-300 font-medium capitalize">{selectedNode.food_path || "none"}</span>
                          </div>

                          <div className="text-xs text-slate-400">
                            Loại sản sinh: <span className="text-slate-300 font-medium">{selectedNode.production_type === "Material" ? "Vật phẩm" : "Nguyên Thạch"}</span>
                          </div>

                          <div className="flex justify-between text-xs text-slate-400">
                            <span>Độ bền: {selectedNode.hp}/{selectedNode.max_hp}</span>
                            {selectedNode.is_destroyed && <span className="text-rose-500 font-bold">Bị phá hủy</span>}
                          </div>

                          {/* Ecosystem details - only for resource nodes */}
                          {!selectedNode.is_destroyed && selectedNode.node_category !== "DEFENSE" && (() => {
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
                            const isPlant = selectedNode.node_type === "FARMING_ZONE" || selectedNode.zone_type === "khu_trong";
                            const isHerb = selectedNode.node_type === "BREEDING_ZONE" || selectedNode.zone_type === "khu_nuoi";
                            const isCarn = false;
                            const isMine = selectedNode.node_type === "MINING_ZONE" || selectedNode.zone_type === "khu_khai_thac";

                            const pCount = apertureNodes.filter(n => !n.is_destroyed && (n.node_type === "FARMING_ZONE" || n.zone_type === "khu_trong")).length;
                            const hCount = apertureNodes.filter(n => !n.is_destroyed && (n.node_type === "BREEDING_ZONE" || n.zone_type === "khu_nuoi")).length;
                            const cCount = apertureNodes.filter(n => !n.is_destroyed && (n.node_type === "MINING_ZONE" || n.zone_type === "khu_khai_thac")).length;

                            const overgrazed = isPlant && hCount > pCount && cCount === 0;
                            const starvingHerb = isHerb && pCount === 0;
                            const starvingCarn = isCarn && hCount === 0;

                             const parseItemPaths = (pathVal: any): string[] => {
                              if (!pathVal) return [];
                              if (Array.isArray(pathVal)) return pathVal.map(p => String(p).trim());
                              if (typeof pathVal === "string") {
                                const s = pathVal.trim();
                                if (s.startsWith("[") && s.endsWith("]")) {
                                  try {
                                    const arr = JSON.parse(s);
                                    if (Array.isArray(arr)) return arr.map(p => String(p).trim());
                                  } catch (e) {}
                                }
                                const cleaned = s.replace(/[\[\]"]/g, "").trim();
                                return cleaned ? [cleaned] : [];
                              }
                              return [];
                            };

                            const getDaoMarkAmount = (pathStr: string | undefined): number => {
                              if (!pathStr) return 0;
                              const paths = parseItemPaths(pathStr);
                              if (!paths || paths.length === 0) return 0;

                              let maxMarks = 0;
                              for (const p of paths) {
                                const target = p.toLowerCase().replace(/ path$/i, "").replace(/_path$/i, "").trim();
                                for (const d of daoMarks) {
                                  if (!d || !d.path) continue;
                                  const dp = d.path.toLowerCase().replace(/ path$/i, "").replace(/_path$/i, "").trim();
                                  if (dp === target && d.amount > maxMarks) {
                                    maxMarks = d.amount;
                                  }
                                }
                                const vietToKey: Record<string, string> = {
                                  "vận": "luck", "vận đạo": "luck",
                                  "mộc": "wood", "mộc đạo": "wood",
                                  "hỏa": "fire", "hỏa đạo": "fire",
                                  "thủy": "water", "thủy đạo": "water",
                                  "kim": "metal", "kim đạo": "metal",
                                  "thổ": "earth", "thổ đạo": "earth",
                                  "lôi": "lightning", "lôi đạo": "lightning",
                                  "băng": "ice", "băng đạo": "ice",
                                  "phong": "wind", "phong đạo": "wind",
                                  "quang": "light", "quang đạo": "light",
                                  "ám": "dark", "ám đạo": "dark",
                                  "kiếm": "sword", "kiếm đạo": "sword",
                                  "đao": "blade", "đao đạo": "blade",
                                  "thực": "food", "thực đạo": "food",
                                  "biến hóa": "transformation", "nô": "enslavement",
                                  "tín": "information", "huyết": "blood", "độc": "poison",
                                  "nhân": "human", "mộng": "dream", "huyễn": "phantom",
                                  "sát": "killing", "lực": "strength", "trí": "wisdom",
                                };
                                const mappedKey = vietToKey[target];
                                if (mappedKey) {
                                  for (const d of daoMarks) {
                                    if (!d || !d.path) continue;
                                    const dp = d.path.toLowerCase().replace(/ path$/i, "").replace(/_path$/i, "").trim();
                                    if (dp === mappedKey && d.amount > maxMarks) {
                                      maxMarks = d.amount;
                                    }
                                  }
                                }
                              }
                              return maxMarks;
                            };

                            const hasSovereignFetus = (player?.gu_inventory || []).some(
                              (g: any) => (g.gu_id || g.id) === "GU_SOVEREIGN_FETUS" && g.equipped
                            );

                            const levelMultiplier = 1.0 + (Math.max(1, selectedNode.level) - 1) * 0.20;
                            const nodePathMarks = getDaoMarkAmount(selectedNode.path);
                            const daoMultiplier = 1.0 + nodePathMarks * 0.01;
                            const conflictMult = (isConflicting && !hasSovereignFetus) ? 0.5 : 1.0;
                            let ecoMult = 1.0;
                            if (overgrazed) ecoMult = 0.5;
                            else if (starvingHerb || starvingCarn) ecoMult = 0.2;
                            else if (isPlant && hCount > 0) ecoMult = 1.25;

                            let vhLaborMult = 1.0;
                            let vhPopBonus = 1.0;
                            const rockmanVH = variantHumans.find(v => v.race === "ROCKMAN");
                            const rockmanPop = rockmanVH ? rockmanVH.population : 0;
                            const rockmanLabor = rockmanVH ? rockmanVH.labor_intensity : "NORMAL";
                            const laborLabels: Record<string, string> = { LOW: "x0.7 (Nhàn)", NORMAL: "x1.0", HIGH: "x1.5", EXPLOIT: "x2.0 (Bóc lột)" };

                            if (isMine && rockmanPop > 0) {
                              vhPopBonus = 1.0 + Math.min(0.5, rockmanPop * 0.001);
                              if (rockmanLabor === "LOW") vhLaborMult = 0.7;
                              if (rockmanLabor === "HIGH") vhLaborMult = 1.5;
                              if (rockmanLabor === "EXPLOIT") vhLaborMult = 2.0;
                            }

                            const getManaVal = (rank: number): number => {
                              const table: Record<number, number> = {
                                1: 1,
                                2: 2,
                                3: 5,
                                4: 20,
                                5: 100,
                                6: 10000,
                                7: 1000000,
                                8: 100000000,
                                9: 10000000000,
                              };
                              return table[Math.min(9, Math.max(1, rank))] || 1;
                            };

                            const areaSize = aperture?.area_size || 0;
                            const areaMultiplier = 1.0 + areaSize / 10000000.0;

                            let itemRankDivisor = 1.0;
                            let itemFoodPathMultiplier = 1.0;
                            let itemSpecName = "";
                            let itemSpecRank = 1;
                            let itemPathName = "";
                            let itemPathMarks = 0;

                            if (selectedNode.production_type === "Material") {
                              let prodKey = selectedNode.production_key;
                              if (!prodKey && selectedNode.name && selectedNode.name.includes("(") && selectedNode.name.includes(")")) {
                                const match = selectedNode.name.match(/\(([^)]+)\)/);
                                if (match && match[1]) {
                                  const itemName = match[1].trim().toLowerCase();
                                  for (const [id, spec] of Object.entries(itemDictionary || {})) {
                                    if (spec.name.toLowerCase() === itemName) {
                                      prodKey = id;
                                      break;
                                    }
                                  }
                                }
                              }

                              if (prodKey && itemDictionary[prodKey]) {
                                const item = itemDictionary[prodKey];
                                itemSpecName = item.name;
                                const effectiveRank = Math.max(item.rank || 0, item.rarity || 0, 1);
                                itemSpecRank = effectiveRank;
                                itemRankDivisor = getManaVal(effectiveRank);
                                if (item.path) {
                                  itemPathName = item.path;
                                  itemPathMarks = getDaoMarkAmount(item.path);
                                  itemFoodPathMultiplier = 1.0 + itemPathMarks * 0.01;
                                }
                              }
                            }

                            const combinedMultiplier = (areaMultiplier * daoMultiplier * conflictMult * ecoMult * vhLaborMult * vhPopBonus * itemFoodPathMultiplier * levelMultiplier) / itemRankDivisor;
                            const baseProd = selectedNode.production_amount || 0;
                            let rawExpected = baseProd > 0 ? Math.max(1, Math.floor(baseProd * combinedMultiplier)) : 0;
                            if (selectedNode.production_type === "Material" && rawExpected > 10000) {
                              rawExpected = 10000;
                            }
                            const expectedYield = rawExpected;

                            const efficiencyPercent = Math.round(combinedMultiplier * 100);

                            // Determine displayed path for Dao Marks row
                            const displayPath = (itemPathName && (itemPathMarks > 0 || !nodePathMarks)) ? itemPathName : (selectedNode.path || "");
                            const displayMarks = (itemPathName && (itemPathMarks > 0 || !nodePathMarks)) ? itemPathMarks : nodePathMarks;

                            return (
                              <div className="p-3 bg-slate-950/80 rounded-xl border border-amber-500/30 text-xs space-y-2.5 shadow-md">
                                <div className="flex items-center justify-between border-b border-slate-800 pb-2">
                                  <div className="font-bold text-amber-400 flex items-center gap-1.5">
                                    <TrendingUp className="w-4 h-4 text-emerald-400" />
                                    Phân Tích Hiệu Suất Sản Lượng
                                  </div>
                                  <span className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                                    combinedMultiplier >= 1.0 
                                      ? "bg-emerald-500/20 text-emerald-400 border border-emerald-500/30" 
                                      : "bg-rose-500/20 text-rose-400 border border-rose-500/30"
                                  }`}>
                                    Hiệu suất thực tế: {efficiencyPercent}% ({combinedMultiplier >= 1.0 ? `+${Math.round((combinedMultiplier - 1) * 100)}%` : `-${Math.round((1 - combinedMultiplier) * 100)}%`})
                                  </span>
                                </div>

                                {/* Banner Sản Lượng Dự Tính (1 lần thu hoạch) */}
                                <div className="p-2.5 bg-gradient-to-r from-emerald-950/70 to-slate-900 border border-emerald-500/40 rounded-lg flex items-center justify-between shadow-inner">
                                  <div>
                                    <div className="text-[10px] text-emerald-400 font-bold uppercase tracking-wider flex items-center gap-1.5">
                                      <Package className="w-3.5 h-3.5 text-emerald-300" />
                                      Sản Lượng Dự Tính (1 Lần Thu Hoạch)
                                    </div>
                                    <div className="text-xs text-slate-300 mt-0.5">
                                      Gốc: <span className="font-semibold text-slate-200">{baseProd}</span> {selectedNode.production_type === "Stones" ? "Nguyên Thạch" : (itemSpecName || "vật phẩm")}
                                    </div>
                                  </div>
                                  <div className="text-right">
                                    <div className="text-sm font-extrabold text-emerald-300">
                                      +{expectedYield.toLocaleString()} {selectedNode.production_type === "Stones" ? "Đá" : "món"}
                                    </div>
                                    {selectedNode.production_type === "Material" && itemSpecRank > 1 && (
                                      <div className="text-[10px] text-amber-400 font-medium">
                                        (Rank {itemSpecRank}: /{itemRankDivisor >= 10000 ? formatNumber(itemRankDivisor) : itemRankDivisor.toLocaleString("vi-VN")})
                                      </div>
                                    )}
                                  </div>
                                </div>

                                <div className="space-y-1.5 text-[11px]">
                                  {/* 1. Dao Marks Bonus */}
                                  <div className="flex justify-between items-center bg-slate-900/60 px-2.5 py-1.5 rounded-lg border border-slate-850">
                                    <span className="text-slate-300 flex items-center gap-1.5 shrink-0">
                                      <span>☯️</span> Đạo ngân bổ trợ ({translatePath(displayPath)}):
                                    </span>
                                    <span
                                      className={displayMarks > 0 ? "text-emerald-400 font-bold text-right" : "text-slate-400 text-right"}
                                      title={`${displayMarks.toLocaleString("vi-VN")} vết Đạo ngân`}
                                    >
                                      {((displayMarks * 1.0) / itemRankDivisor) >= 10000
                                        ? `+${formatNumber((displayMarks * 1.0) / itemRankDivisor)}%`
                                        : `+${((displayMarks * 1.0) / itemRankDivisor).toFixed(1)}%`}
                                      {itemRankDivisor > 1 && (
                                        <span className="text-[10px] text-amber-400 font-normal ml-1">
                                          (chia Rank /{itemRankDivisor >= 10000 ? formatNumber(itemRankDivisor) : itemRankDivisor.toLocaleString("vi-VN")})
                                        </span>
                                      )}
                                    </span>
                                  </div>

                                  {/* Secondary Node Path Dao Marks if different from item path */}
                                  {itemPathName && selectedNode.path && itemPathName !== selectedNode.path && nodePathMarks > 0 && (
                                    <div className="flex justify-between items-center bg-slate-900/60 px-2.5 py-1.5 rounded-lg border border-slate-850 text-[10px]">
                                      <span className="text-slate-400 flex items-center gap-1.5 shrink-0">
                                        <span>☯️</span> Đạo ngân mỏ ({translatePath(selectedNode.path)}):
                                      </span>
                                      <span
                                        className="text-emerald-400 font-bold text-right"
                                        title={`${nodePathMarks.toLocaleString("vi-VN")} vết Đạo ngân`}
                                      >
                                        {nodePathMarks >= 10000 ? `+${formatNumber(nodePathMarks)}%` : `+${(nodePathMarks * 1.0).toFixed(1)}%`}
                                      </span>
                                    </div>
                                  )}

                                  {/* 2. Node Level Bonus */}
                                  <div className="flex justify-between items-center bg-slate-900/60 px-2.5 py-1.5 rounded-lg border border-slate-850">
                                    <span className="text-slate-300 flex items-center gap-1.5">
                                      <span>⭐</span> Cấp độ mỏ (Lv.{selectedNode.level}):
                                    </span>
                                    <span className="text-amber-400 font-bold">
                                      +{((selectedNode.level - 1) * 20).toFixed(0)}% sản lượng
                                    </span>
                                  </div>

                                  {/* 3. Dao Conflict Status */}
                                  <div className="flex justify-between items-center bg-slate-900/60 px-2.5 py-1.5 rounded-lg border border-slate-850">
                                    <span className="text-slate-300 flex items-center gap-1.5">
                                      <span>⚔️</span> Xung đột Đạo ngân:
                                    </span>
                                    {hasSovereignFetus ? (
                                      <span className="text-purple-400 font-bold">Thân thể Chí Tôn (Miễn xung đột)</span>
                                    ) : isConflicting ? (
                                      <span className="text-rose-400 font-bold">-50% sản lượng (-10% HP/năm)</span>
                                    ) : (
                                      <span className="text-emerald-400 font-semibold">Hài hòa (100%)</span>
                                    )}
                                  </div>

                                  {/* 4. Ecosystem Status */}
                                  {(overgrazed || starvingHerb || starvingCarn || (isPlant && hCount > 0)) && (
                                    <div className="flex justify-between items-center bg-slate-900/60 px-2.5 py-1.5 rounded-lg border border-slate-850">
                                      <span className="text-slate-300 flex items-center gap-1.5">
                                        <span>🌿</span> Trạng thái Sinh Thái:
                                      </span>
                                      {overgrazed && <span className="text-amber-400 font-bold">Quá tải gặm nhấm (-50%)</span>}
                                      {starvingHerb && <span className="text-rose-400 font-bold">Thiếu cỏ thực vật (-80%)</span>}
                                      {starvingCarn && <span className="text-rose-400 font-bold">Thiếu con mồi (-80%)</span>}
                                      {!overgrazed && !starvingHerb && !starvingCarn && isPlant && hCount > 0 && (
                                        <span className="text-emerald-400 font-bold">Phân bón sinh thái (+25%)</span>
                                      )}
                                    </div>
                                  )}

                                  {/* 5. Variant Humans Labor Status */}
                                  {isMine && rockmanPop > 0 && (
                                    <div className="space-y-1 bg-slate-900/60 p-2 rounded-lg border border-slate-850 text-[10px]">
                                      <div className="flex justify-between items-center">
                                        <span className="text-slate-300">🪨 Dân số Người Đá:</span>
                                        <span className="text-cyan-400 font-bold">{rockmanPop} (+{(Math.min(0.5, rockmanPop * 0.001) * 100).toFixed(1)}%)</span>
                                      </div>
                                      <div className="flex justify-between items-center">
                                        <span className="text-slate-300">⚙️ Cường độ lao động:</span>
                                        <span className="text-amber-400 font-bold">{laborLabels[rockmanLabor]}</span>
                                      </div>
                                    </div>
                                  )}
                                </div>
                              </div>
                            );
                          })()}

                          {/* Defense Structure Details */}
                          {!selectedNode.is_destroyed && selectedNode.node_category === "DEFENSE" && (() => {
                            return (
                              <div className="p-3 bg-slate-900/40 rounded-lg border border-slate-805 text-[10px] space-y-1 text-slate-400">
                                <div className="font-bold text-slate-300">Thông Tin Công Trình Phòng Thủ:</div>
                                {selectedNode.blocks_movement && (
                                  <div>• <span className="text-amber-400 font-medium">Tường Thành</span>: Chặn kẻ thù di chuyển qua ô này</div>
                                )}
                                {selectedNode.defense_power > 0 && (
                                  <div>• <span className="text-cyan-400 font-medium">Tháp Canh</span>: Gây {selectedNode.defense_power} sát thương/kẻ thù trong phạm vi {selectedNode.attack_range} ô</div>
                                )}
                                {selectedNode.hp_bonus > 0 && (
                                  <div>• <span className="text-emerald-400 font-medium">Tượng Hộ Mệnh</span>: Tăng +{selectedNode.hp_bonus} HP cho công trình trong phạm vi {selectedNode.attack_range} ô</div>
                                )}
                                {selectedNode.damage_reduction > 0 && selectedNode.attack_range === 0 && (
                                  <div>• <span className="text-blue-400 font-medium">Doanh Trại</span>: Giảm {selectedNode.damage_reduction}% sát thương cho TẤT CẢ công trình</div>
                                )}
                                {selectedNode.damage_reduction > 0 && selectedNode.attack_range > 0 && (
                                  <div>• <span className="text-purple-400 font-medium">Trường Lực</span>: Giảm {selectedNode.damage_reduction}% sát thương cho công trình trong phạm vi {selectedNode.attack_range} ô</div>
                                )}
                                {selectedNode.required_gu_id && (
                                  <div>• Cổ trùng đã hy sinh: <span className="text-rose-400 font-medium">{selectedNode.required_gu_id}</span> (rank {selectedNode.required_gu_rank}+)</div>
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

                          {/* Collect timing indicator - only for resource nodes */}
                          {!selectedNode.is_destroyed && selectedNode.node_category !== "DEFENSE" && (
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
                            const repairCost = selectedNode.is_destroyed ? Math.ceil((selectedNode.max_hp * 2) / 50) : Math.ceil((selectedNode.max_hp - selectedNode.hp) / 50);
                            const canRepair = player.immortal_stones >= repairCost;
                            return (
                              <div className="pt-2 border-t border-slate-800">
                                <button
                                  onClick={() => handleRepair(selectedNode.id)}
                                  disabled={actionLoading || !canRepair}
                                  className="w-full py-2 bg-gradient-to-r from-cyan-600 to-blue-600 disabled:from-slate-800 disabled:to-slate-900 disabled:text-slate-500 text-white font-bold text-xs rounded-lg transition"
                                >
                                  {actionLoading ? "Đang sửa..." : selectedNode.is_destroyed ? `Tái thiết phế tích (${repairCost} Tiên Nguyên Thạch)` : `Sửa chữa độ bền (${repairCost} Tiên Nguyên Thạch)`}
                                </button>
                              </div>
                            );
                          })()}

                          {/* Upgrade Node Section */}
                          {!selectedNode.is_destroyed && (() => {
                            const MAX_LEVEL = 5;
                            const isMaxLevel = selectedNode.level >= MAX_LEVEL;
                            const template = (() => {
                              const costs: Record<string, number> = {
                                FARMING_ZONE: 10, BREEDING_ZONE: 24, MINING_ZONE: 20,
                                WALL: 16, WATCHTOWER: 24, GUARDIAN_STATUE: 30, BARRACKS: 40, FORCE_FIELD: 50,
                              };
                              return costs[selectedNode.node_type] || 10;
                            })();
                            const upgradeCost = Math.max(5, Math.floor(template * selectedNode.level * 1.5));
                            const canUpgrade = player.immortal_stones >= upgradeCost;

                            return (
                              <div className="pt-2 border-t border-slate-800">
                                {isMaxLevel ? (
                                  <div className="text-center text-xs text-emerald-400 font-semibold py-2">
                                    ⭐ Đã đạt cấp tối đa (Lv.{MAX_LEVEL})
                                  </div>
                                ) : (
                                  <div className="space-y-1">
                                    <div className="text-[10px] text-slate-400 flex justify-between">
                                      <span>Nâng cấp lên Lv.{selectedNode.level + 1}:</span>
                                      <span className={canUpgrade ? "text-amber-400" : "text-rose-400"}>
                                        {upgradeCost} Tiên Nguyên Thạch
                                      </span>
                                    </div>
                                    <div className="text-[10px] text-slate-500">
                                      {selectedNode.node_category === "RESOURCE"
                                        ? `+50% sản lượng, +500 HP`
                                        : `+25% hiệu ứng phòng thủ, +500 HP`}
                                    </div>
                                    <button
                                      onClick={() => handleUpgrade(selectedNode.id)}
                                      disabled={actionLoading || !canUpgrade}
                                      className="w-full py-2 bg-gradient-to-r from-amber-600 to-yellow-600 disabled:from-slate-800 disabled:to-slate-900 disabled:text-slate-500 text-white font-bold text-xs rounded-lg transition"
                                    >
                                      {actionLoading ? "Đang nâng cấp..." : `Nâng Cấp (Lv.${selectedNode.level} → ${selectedNode.level + 1})`}
                                    </button>
                                  </div>
                                )}
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

                      const qtyWall = player.materials_inventory?.["node_wall"] || 0;
                      const qtyWatchtower = player.materials_inventory?.["node_watchtower"] || 0;
                      const qtyGuardianStatue = player.materials_inventory?.["node_guardian_statue"] || 0;
                      const qtyBarracks = player.materials_inventory?.["node_barracks"] || 0;
                      const qtyForceField = player.materials_inventory?.["node_force_field"] || 0;

                      type NodeOption = {
                        id: string;
                        name: string;
                        desc: string;
                        cost: number;
                        qty: number;
                        zone: string;
                      };

                      const categoryData: Record<"khu_trong" | "khu_nuoi" | "khu_khai_thac" | "defense", NodeOption[]> = {
                        khu_trong: [
                          { id: "FARMING_ZONE", name: "🌱 Khu Trồng Trọt (Tùy chỉnh)", desc: "Trồng tài nguyên thực vật theo Đại Đạo. Phí: 10 Tiên Nguyên Thạch.", cost: 10, qty: 0, zone: "khu_trong" },
                        ],
                        khu_nuoi: [
                          { id: "BREEDING_ZONE", name: "🐄 Khu Chăn Nuôi (Tùy chỉnh)", desc: "Nuôi thú theo Đại Đạo. Phí: 24 Tiên Nguyên Thạch.", cost: 24, qty: 0, zone: "khu_nuoi" },
                        ],
                        khu_khai_thac: [
                          { id: "MINING_ZONE", name: "⛏️ Khu Khai Thác (Tùy chỉnh)", desc: "Khai thác khoáng thạch theo Đại Đạo. Phí: 20 Tiên Nguyên Thạch.", cost: 20, qty: 0, zone: "khu_khai_thac" },
                        ],
                        defense: [
                          { id: "WALL", name: "🧱 Tường Thành", desc: "Chặn kẻ thù di chuyển. HP: 3000. Chi phí: 16 Tiên Nguyên Thạch + 2 Cổ trùng (Thú Bì, Thạch Bì).", cost: 16, qty: qtyWall, zone: "defense" },
                          { id: "WATCHTOWER", name: "🗼 Tháp Canh", desc: "Bắn kẻ thù phạm vi 3 ô. HP: 2000. Chi phí: 24 Tiên Nguyên Thạch + 2 Cổ trùng (Thiết Bì, Bán Thạch).", cost: 24, qty: qtyWatchtower, zone: "defense" },
                          { id: "GUARDIAN_STATUE", name: "🗿 Tượng Hộ Mệnh", desc: "Tăng 500 HP node lân cận (phạm vi 2 ô). HP: 2500. Chi phí: 30 Tiên Nguyên Thạch + 2 Cổ trùng (Bạch Ngọc, Nguyệt Nghê).", cost: 30, qty: qtyGuardianStatue, zone: "defense" },
                          { id: "BARRACKS", name: "⚔️ Doanh Trại", desc: "Giảm 15% sát thương toàn bộ Tiên Khiếu. HP: 4000. Chi phí: 40 Tiên Nguyên Thạch + 2 Cổ trùng (Kim Chung, Tượng Nha).", cost: 40, qty: qtyBarracks, zone: "defense" },
                          { id: "FORCE_FIELD", name: "🔮 Trường Lực", desc: "Giảm 30% sát thương phạm vi 2 ô. HP: 1500. Chi phí: 50 Tiên Nguyên Thạch + 2 Cổ trùng (Chí Tôn, Kim Chung).", cost: 50, qty: qtyForceField, zone: "defense" },
                        ],
                      };

                      return (
                        <div className="space-y-2.5">
                          <div className="bg-slate-950/60 px-3 py-1.5 rounded-lg border border-slate-850 flex justify-between items-center text-xs">
                            <span className="text-slate-400 font-medium">Số lượng mỏ tài nguyên:</span>
                            <span className={`${isAtLimit ? 'text-rose-400' : 'text-amber-400'} font-bold`}>
                              {currentNodesCount} / {maxNodes} mỏ
                            </span>
                          </div>

                          {isAtLimit && (
                            <div className="p-2 bg-rose-500/10 border border-rose-500/20 rounded-lg text-center text-xs text-rose-300">
                              Quy mô Tiên Khiếu đạt giới hạn mỏ. Hãy tháo dỡ hoặc mở rộng thêm đất đai.
                            </div>
                          )}

                          {/* Category Tabs */}
                          <div className="flex bg-slate-950/80 p-1 rounded-lg border border-slate-800 text-[10px]">
                            {["khu_trong", "khu_nuoi", "khu_khai_thac", "defense"].map((cat) => (
                              <button
                                key={cat}
                                onClick={() => {
                                  setBuildCategory(cat as any);
                                  setPendingBuildZone(null); // Tắt bảng chọn tùy chỉnh khi chuyển tab!
                                }}
                                className={`flex-1 py-1 font-bold rounded capitalize transition-all ${
                                  buildCategory === cat ? "bg-slate-800 text-amber-400" : "text-slate-400"
                                }`}
                              >
                                {cat === "khu_trong" ? "🌱 Khu Trồng" : cat === "khu_nuoi" ? "🐄 Khu Nuôi" : cat === "khu_khai_thac" ? "⛏️ Khu Khai Thác" : "🛡️ Phòng Thủ"}
                              </button>
                            ))}
                          </div>

                          <div className="space-y-3 pt-1">
                            {pendingBuildZone ? (
                              <div className="p-3 bg-slate-900/80 rounded-xl border border-slate-700/60 space-y-3 shadow-md">
                                <div className="flex items-center justify-between border-b border-slate-800 pb-2">
                                  <div className="text-xs font-bold text-amber-400 flex items-center gap-1.5">
                                    {pendingBuildZone === "FARMING_ZONE" ? "🌱 Tùy chỉnh Khu Trồng Trọt" : pendingBuildZone === "BREEDING_ZONE" ? "🐄 Tùy chỉnh Khu Chăn Nuôi" : pendingBuildZone === "MINING_ZONE" ? "⛏️ Tùy chỉnh Khu Khai Thác" : "⚙️ Tùy chỉnh Mỏ Tài Nguyên"}
                                  </div>
                                  <span className="text-[10px] text-slate-400 font-medium">Chi phí: 10 Tiên Nguyên Thạch</span>
                                </div>
                                
                                {(() => {
                                  // Parse path helper
                                  const parseItemPaths = (pathVal: any): string[] => {
                                    if (!pathVal) return [];
                                    if (Array.isArray(pathVal)) return pathVal.map(p => String(p).trim());
                                    if (typeof pathVal === "string") {
                                      const s = pathVal.trim();
                                      if (s.startsWith("[") && s.endsWith("]")) {
                                        try {
                                          const arr = JSON.parse(s);
                                          if (Array.isArray(arr)) return arr.map(p => String(p).trim());
                                        } catch (e) {}
                                      }
                                      const cleaned = s.replace(/[\[\]"]/g, "").trim();
                                      return cleaned ? [cleaned] : [];
                                    }
                                    return [];
                                  };

                                  // Compute valid paths from itemDictionary
                                  const pathsSet = new Set<string>();
                                  const defaultPaths = [
                                    "Wisdom Path", "Wood Path", "Fire Path", "Earth Path", "Metal Path", "Water Path",
                                    "Wind Path", "Light Path", "Dark Path", "Ice Path", "Time Path",
                                    "Space Path", "Blood Path", "Blade Path", "Sword Path",
                                    "Food Path", "Star Path", "Poison Path", "Human Path", "Enslavement Path",
                                    "Strength Path", "Bone Path", "Transformation Path", "Rule Path",
                                    "Luck Path", "Refinement Path", "Soul Path", "Sound Path", "Faith Path"
                                  ];
                                  defaultPaths.forEach(dp => pathsSet.add(dp));

                                  Object.values(itemDictionary || {}).forEach(item => {
                                    const paths = parseItemPaths(item.path);
                                    paths.forEach(p => {
                                      if (p) {
                                        let formatted = p.trim();
                                        if (!formatted.toLowerCase().endsWith("path")) {
                                          formatted = formatted.charAt(0).toUpperCase() + formatted.slice(1) + " Path";
                                        } else {
                                          formatted = formatted.split(" ").map(w => w.charAt(0).toUpperCase() + w.slice(1).toLowerCase()).join(" ");
                                        }
                                        pathsSet.add(formatted);
                                      }
                                    });
                                  });

                                  // Helper to count items for a path
                                  const getItemCountForPath = (pathName: string) => {
                                    const rawPath = pathName.replace(/ Path$/i, '').trim().toLowerCase();
                                    return Object.values(itemDictionary || {}).filter(item => {
                                      const itemPaths = parseItemPaths(item.path).map(p => p.toLowerCase().replace(/ path$/i, '').trim());
                                      return itemPaths.includes(rawPath);
                                    }).length;
                                  };

                                  // Target zone category
                                  const targetZone = pendingBuildZone === "FARMING_ZONE" ? "khu_trong"
                                    : pendingBuildZone === "BREEDING_ZONE" ? "khu_nuoi"
                                    : pendingBuildZone === "MINING_ZONE" ? "khu_khai_thac"
                                    : buildCategory;

                                  const targetZoneTitle = targetZone === "khu_trong" ? "🌱 Trồng Trọt"
                                    : targetZone === "khu_nuoi" ? "🐄 Chăn Nuôi"
                                    : targetZone === "khu_khai_thac" ? "⛏️ Khai Thác"
                                    : "Tất cả";

                                  // Sort paths alphabetically by Vietnamese translation
                                  const sortedPaths = Array.from(pathsSet).sort((a, b) => {
                                    const nameA = translatePath(a);
                                    const nameB = translatePath(b);
                                    return nameA.localeCompare(nameB, "vi");
                                  });

                                  // Filter paths by pathSearch query
                                  const searchP = pathSearch.trim().toLowerCase();
                                  const filteredPaths = sortedPaths.filter(p => {
                                    if (!searchP) return true;
                                    const viName = translatePath(p).toLowerCase();
                                    const enName = p.toLowerCase();
                                    return viName.includes(searchP) || enName.includes(searchP);
                                  });
                                  
                                  // Compute available items for the currently selected path & target zone
                                  const selectedRawPath = buildPath.replace(/ Path$/i, '').trim().toLowerCase();
                                  const availableItems = Object.values(itemDictionary || {}).filter(item => {
                                    const itemPaths = parseItemPaths(item.path).map(p => p.toLowerCase().replace(/ path$/i, '').trim());
                                    const matchesPath = buildPath === "ALL" || itemPaths.includes(selectedRawPath);
                                    const matchesZone = showAllZoneItems || !["khu_trong", "khu_nuoi", "khu_khai_thac"].includes(targetZone)
                                      ? true
                                      : getItemZoneCategory(item) === targetZone;
                                    return matchesPath && matchesZone;
                                  }).sort((a, b) => (a.rank || 0) - (b.rank || 0) || a.name.localeCompare(b.name, "vi"));

                                  // Filter items by itemSearch query
                                  const searchI = itemSearch.trim().toLowerCase();
                                  const filteredItems = availableItems.filter(item => {
                                    if (!searchI) return true;
                                    return item.name.toLowerCase().includes(searchI) || item.id.toLowerCase().includes(searchI);
                                  });

                                  const selectedItemSpec = buildProdKey ? itemDictionary[buildProdKey] : null;

                                  return (
                                    <div className="space-y-3">
                                      <div>
                                        <div className="flex justify-between items-center mb-1">
                                          <label className="text-[11px] font-semibold text-slate-300">1. Chọn Đại Đạo (Path)</label>
                                          <span className="text-[10px] text-amber-400 font-bold">{translatePath(buildPath)}</span>
                                        </div>

                                        {/* Search Input for Path */}
                                        <input 
                                          type="text"
                                          placeholder="🔍 Tìm Đại Đạo (vd: Huyết Đạo, Trí Đạo...)"
                                          className="w-full bg-slate-950 border border-slate-700 text-xs text-slate-200 rounded-lg p-2 mb-1.5 focus:border-amber-500 outline-none transition"
                                          value={pathSearch}
                                          onChange={(e) => setPathSearch(e.target.value)}
                                        />

                                        <select 
                                          className="w-full bg-slate-950 border border-slate-700 text-xs text-slate-200 rounded-lg p-2 focus:border-amber-500 outline-none cursor-pointer"
                                          value={buildPath} 
                                          onChange={(e) => {
                                            setBuildPath(e.target.value);
                                            setBuildProdKey(""); // reset item when path changes
                                            setItemSearch("");
                                          }}
                                        >
                                          <option value="ALL">✨ Tất cả Đại Đạo (All Paths)</option>
                                          {filteredPaths.map(p => {
                                            const icon = getPathIconAndColor(p, "RESOURCE").icon;
                                            const count = getItemCountForPath(p);
                                            return (
                                              <option key={p} value={p}>
                                                {icon} {translatePath(p)} ({p}) — [{count} vật phẩm]
                                              </option>
                                            );
                                          })}
                                        </select>
                                      </div>
                                      
                                      <div>
                                        <div className="flex justify-between items-center mb-1">
                                          <label className="text-[11px] font-semibold text-slate-300">
                                            2. Chọn vật phẩm sản xuất {targetZoneTitle}
                                          </label>
                                          <span className="text-[10px] text-slate-400 font-medium">({filteredItems.length} vật phẩm)</span>
                                        </div>

                                        {/* Search Input for Item */}
                                        <input 
                                          type="text"
                                          placeholder="🔍 Tìm theo tên vật phẩm..."
                                          className="w-full bg-slate-950 border border-slate-700 text-xs text-slate-200 rounded-lg p-2 mb-1.5 focus:border-amber-500 outline-none transition"
                                          value={itemSearch}
                                          onChange={(e) => setItemSearch(e.target.value)}
                                        />

                                        <select 
                                          className="w-full bg-slate-950 border border-slate-700 text-xs text-slate-200 rounded-lg p-2 focus:border-amber-500 outline-none cursor-pointer"
                                          value={buildProdKey} 
                                          onChange={(e) => setBuildProdKey(e.target.value)}
                                        >
                                          <option value="" disabled>-- Chọn vật phẩm ({filteredItems.length} vật phẩm) --</option>
                                          {filteredItems.map(item => {
                                            const itemCat = getItemZoneCategory(item);
                                            const catTag = itemCat === "khu_trong" ? "🌱 Trồng" : itemCat === "khu_nuoi" ? "🐄 Nuôi" : "⛏️ Khai Thác";
                                            const rankTag = item.rank ? (item.rank >= 6 ? `Rank ${item.rank} ⭐ Tiên Cấp` : `Rank ${item.rank}`) : "Rank 1";
                                            return (
                                              <option key={item.id} value={item.id}>
                                                [{catTag}] [{rankTag}] {item.name}
                                              </option>
                                            );
                                          })}
                                        </select>

                                        {filteredItems.length === 0 && (
                                          <div className="mt-2 p-2.5 bg-amber-500/10 border border-amber-500/20 rounded-lg text-[10px] text-amber-300 text-center space-y-1">
                                            <div>Chưa tìm thấy vật phẩm thuộc nhóm <strong>{targetZoneTitle}</strong> trong <strong>{translatePath(buildPath)}</strong>.</div>
                                            <button 
                                              type="button"
                                              onClick={() => setShowAllZoneItems(true)} 
                                              className="text-cyan-400 underline font-semibold cursor-pointer"
                                            >
                                              Bấm vào đây để xem tất cả vật phẩm của {translatePath(buildPath)}
                                            </button>
                                          </div>
                                        )}
                                      </div>

                                      {/* Selected Item Preview Card */}
                                      {selectedItemSpec && (
                                        <div className="p-3 bg-slate-950/90 rounded-xl border border-amber-500/40 space-y-1.5 shadow-lg">
                                          <div className="flex items-center justify-between">
                                            <span className="text-xs font-bold text-amber-400">{selectedItemSpec.name}</span>
                                            <span className="text-[10px] px-2 py-0.5 rounded bg-amber-500/20 text-amber-300 border border-amber-500/30 font-bold">
                                              Rank {selectedItemSpec.rank || 1} {(selectedItemSpec.rank && selectedItemSpec.rank >= 6) ? "⭐ Tiên Cấp" : ""}
                                            </span>
                                          </div>
                                          {selectedItemSpec.description && (
                                            <div className="text-[10px] text-slate-300 leading-relaxed">{selectedItemSpec.description}</div>
                                          )}
                                          <div className="text-[9px] text-slate-400 pt-1.5 flex items-center justify-between border-t border-slate-800">
                                            <span>Đại Đạo: <strong className="text-slate-200">{translatePath(buildPath)}</strong></span>
                                            <span>Chu kỳ thu: <strong className="text-emerald-400">10 tiếng/lần</strong></span>
                                          </div>
                                        </div>
                                      )}
                                    </div>
                                  );
                                })()}
                                <div className="flex gap-2 pt-2">
                                  <button
                                    onClick={() => setPendingBuildZone(null)}
                                    className="flex-1 py-1.5 bg-slate-800 text-slate-300 text-[10px] font-bold rounded hover:bg-slate-700 transition"
                                  >
                                    Hủy
                                  </button>
                                  <button
                                    onClick={() => handleBuildNode(pendingBuildZone, buildPath, buildProdKey)}
                                    disabled={actionLoading || !buildProdKey}
                                    className="flex-1 py-1.5 bg-gradient-to-r from-amber-500 to-yellow-600 disabled:from-slate-800 disabled:text-slate-500 text-slate-950 text-[10px] font-bold rounded transition"
                                  >
                                    {actionLoading ? "Đang xây..." : "Xác nhận Xây"}
                                  </button>
                                </div>
                              </div>
                            ) : (
                              categoryData[buildCategory]?.map((option) => {
                                const nodeItemID = "node_" + option.id.toLowerCase();
                                const hasModel = option.qty > 0;
                                const canBuild = hasModel || player.immortal_stones >= option.cost;

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
                                      onClick={() => {
                                        if (["FARMING_ZONE", "BREEDING_ZONE", "MINING_ZONE"].includes(option.id)) {
                                          setPendingBuildZone(option.id);
                                        } else {
                                          handleBuildNode(option.id);
                                        }
                                      }}
                                      disabled={actionLoading || isAtLimit || !canBuild}
                                      className="px-3 py-1.5 bg-gradient-to-r from-amber-500 to-yellow-600 disabled:from-slate-800 disabled:to-slate-900 disabled:text-slate-500 hover:brightness-110 text-slate-950 font-bold text-[10px] rounded-lg transition whitespace-nowrap shrink-0"
                                    >
                                      {hasModel ? "Đặt" : "Xây"}
                                    </button>
                                  </div>
                                );
                              })
                            )}
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
                        <div className="text-xs text-slate-400" title={`${mark.amount.toLocaleString("vi-VN")} Đạo Ngân`}>
                          Tích lũy: <span className="text-amber-400 font-bold">{mark.amount >= 100000 ? formatNumber(mark.amount) : mark.amount.toLocaleString("vi-VN")}</span> Đạo Ngân
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
    </GamePanel>
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
