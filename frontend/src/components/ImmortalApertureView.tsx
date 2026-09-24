import React, { useState, useEffect } from "react";
import { useGameStore, ApertureResourceNode, TribulationEnemy } from "@/store/useGameStore";
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
  Info,
  Search,
  Star,
  Award,
  ShieldCheck,
  ShieldAlert,
  BookOpen,
} from "lucide-react";
import { translatePath } from "@/lib/translate";
import { getRankScale } from "@/lib/balance";
import { formatNumber, calculateTribulationProgress } from "@/lib/utils";
import { Num } from "./Num";
import { GamePanel, GameTabs, GameButton, GameCard, GameModal, GameProgressBar, GameSection } from "@/components/ui";
import { CountdownTimer } from "./CountdownTimer";
import { TribulationRealTimeInfo } from "./TribulationRealTimeInfo";

export interface DaoGroupDef {
  id: string;
  name: string;
  icon: string;
  desc: string;
  color: string;
  badgeBg: string;
}

export const DAO_GROUPS: DaoGroupDef[] = [
  { id: "ALL", name: "Tất Cả", icon: "🌌", desc: "Tất cả các lưu phái Đạo Ngân", color: "text-amber-300 border-amber-500/40 bg-amber-950/30", badgeBg: "bg-amber-500/20 text-amber-300 border-amber-500/30" },
  { id: "ELEMENTAL", name: "Nguyên Tố", icon: "🔥", desc: "Kim, Mộc, Thủy, Hỏa, Thổ, Phong, Lôi, Băng, Vân, Quang, Ám, Tinh, Âm", color: "text-red-300 border-red-500/40 bg-red-950/30", badgeBg: "bg-red-500/20 text-red-300 border-red-500/30" },
  { id: "SPACE_TIME", name: "Không - Thời", icon: "⏳", desc: "Vũ Đạo (Không), Trụ Đạo (Thời)", color: "text-cyan-300 border-cyan-500/40 bg-cyan-950/30", badgeBg: "bg-cyan-500/20 text-cyan-300 border-cyan-500/30" },
  { id: "RULE_ORDER", name: "Quy Luật & Trận", icon: "📜", desc: "Luật Đạo, Cấm Đạo, Trận Đạo", color: "text-emerald-300 border-emerald-500/40 bg-emerald-950/30", badgeBg: "bg-emerald-500/20 text-emerald-300 border-emerald-500/30" },
  { id: "KINETIC_WEAPON", name: "Binh Khí & Sát", icon: "⚔️", desc: "Kiếm Đạo, Đao Đạo, Binh Đạo, Sát Đạo", color: "text-slate-200 border-slate-400/40 bg-slate-900/50", badgeBg: "bg-slate-500/20 text-slate-300 border-slate-400/30" },
  { id: "BODY_LIFE", name: "Thể Phách & Sinh", icon: "💪", desc: "Lực Đạo, Cốt Đạo, Biến Hóa Đạo, Thực Đạo", color: "text-orange-300 border-orange-500/40 bg-orange-950/30", badgeBg: "bg-orange-500/20 text-orange-300 border-orange-500/30" },
  { id: "SOUL_MIND", name: "Hồn Phách & Trí", icon: "🧠", desc: "Hồn Đạo, Trí Đạo, Tình Đạo, Nô Đạo, Mộng Đạo", color: "text-purple-300 border-purple-500/40 bg-purple-950/30", badgeBg: "bg-purple-500/20 text-purple-300 border-purple-500/30" },
  { id: "INFORMATION_STEALTH", name: "Tín Đạo & Vận", icon: "🕵️", desc: "Tín Đạo, Vận Đạo, Thâu Đạo, Huyễn Đạo, Họa Đạo", color: "text-pink-300 border-pink-500/40 bg-pink-950/30", badgeBg: "bg-pink-500/20 text-pink-300 border-pink-500/30" },
  { id: "ORGANIC_CORRUPTION", name: "Huyết & Độc", icon: "🩸", desc: "Huyết Đạo, Độc Đạo", color: "text-rose-300 border-rose-500/40 bg-rose-950/30", badgeBg: "bg-rose-500/20 text-rose-300 border-rose-500/30" },
  { id: "CRAFT_ALCHEMY", name: "Luyện & Đan", icon: "⚗️", desc: "Luyện Đạo, Đan Đạo", color: "text-yellow-300 border-yellow-500/40 bg-yellow-950/30", badgeBg: "bg-yellow-500/20 text-yellow-300 border-yellow-500/30" },
  { id: "HEAVEN_HUMAN", name: "Thiên Nhân & Khí", icon: "☯️", desc: "Thiên Đạo, Nhân Đạo, Khí Đạo", color: "text-blue-300 border-blue-500/40 bg-blue-950/30", badgeBg: "bg-blue-500/20 text-blue-300 border-blue-500/30" },
];

export const PATH_TO_DAO_GROUP: Record<string, string> = {
  metal: "ELEMENTAL", wood: "ELEMENTAL", water: "ELEMENTAL", fire: "ELEMENTAL", earth: "ELEMENTAL",
  wind: "ELEMENTAL", lightning: "ELEMENTAL", ice: "ELEMENTAL", cloud: "ELEMENTAL", light: "ELEMENTAL",
  dark: "ELEMENTAL", star: "ELEMENTAL", sound: "ELEMENTAL",
  space: "SPACE_TIME", time: "SPACE_TIME",
  rule: "RULE_ORDER", restriction: "RULE_ORDER", formation: "RULE_ORDER",
  sword: "KINETIC_WEAPON", blade: "KINETIC_WEAPON", weapon: "KINETIC_WEAPON", killing: "KINETIC_WEAPON", kill: "KINETIC_WEAPON",
  strength: "BODY_LIFE", bone: "BODY_LIFE", transformation: "BODY_LIFE", food: "BODY_LIFE",
  soul: "SOUL_MIND", wisdom: "SOUL_MIND", emotion: "SOUL_MIND", enslavement: "SOUL_MIND", dream: "SOUL_MIND",
  information: "INFORMATION_STEALTH", luck: "INFORMATION_STEALTH", theft: "INFORMATION_STEALTH", phantom: "INFORMATION_STEALTH", painting: "INFORMATION_STEALTH", investigation: "INFORMATION_STEALTH",
  blood: "ORGANIC_CORRUPTION", poison: "ORGANIC_CORRUPTION", death: "ORGANIC_CORRUPTION",
  refinement: "CRAFT_ALCHEMY", pill: "CRAFT_ALCHEMY", creation: "CRAFT_ALCHEMY",
  destruction: "RULE_ORDER",
  heaven: "HEAVEN_HUMAN", human: "HEAVEN_HUMAN", qi: "HEAVEN_HUMAN",
};

export function getAttainmentBadge(name?: string, level?: number): { text: string; bg: string; textCol: string; border: string } {
  const normName = (name || "").toLowerCase();
  if (normName.includes("vô thượng") || level === 5) {
    return { text: name || "Vô Thượng Đại Tông Sư", bg: "bg-rose-950/80", textCol: "text-rose-300", border: "border-rose-500/50" };
  }
  if (normName.includes("đại tông sư") || level === 4) {
    return { text: name || "Đại Tông Sư", bg: "bg-amber-950/80", textCol: "text-amber-300", border: "border-amber-500/50" };
  }
  if (normName.includes("tông sư") || level === 3) {
    return { text: name || "Tông Sư", bg: "bg-purple-950/80", textCol: "text-purple-300", border: "border-purple-500/50" };
  }
  if (normName.includes("đại sư") || level === 2) {
    return { text: name || "Đại Sư", bg: "bg-emerald-950/80", textCol: "text-emerald-300", border: "border-emerald-500/50" };
  }
  if (normName.includes("chuẩn") || level === 1) {
    return { text: name || "Chuẩn Đại Sư", bg: "bg-cyan-950/80", textCol: "text-cyan-300", border: "border-cyan-500/50" };
  }
  return { text: name || "Phổ Thông", bg: "bg-slate-800/80", textCol: "text-slate-400", border: "border-slate-700/50" };
}

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

export function getTribulationTypeName(type: string): string {
  switch (type) {
    case "EARTH_CALAMITY": return "Địa Tai";
    case "HEAVENLY_TRIBULATION": return "Thiên Kiếp";
    case "GRAND_TRIBULATION": return "Hạo Kiếp";
    case "MYRIAD_TRIBULATION": return "Vạn Kiếp";
    case "CHAOS_CALAMITY": return "Hỗn Độn Đại Nạn";
    default: return type;
  }
}

export function getRankName(rank: number): string {
  switch (rank) {
    case 6: return "Lục Chuyển";
    case 7: return "Thất Chuyển";
    case 8: return "Bát Chuyển";
    case 9: return "Cửu Chuyển";
    default: return `${rank} Chuyển`;
  }
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

export const ImmortalApertureView: React.FC = () => {
  const {
    player,
    aperture,
    daoMarks,
    daoNgan,
    apertureNodes,
    variantHumans,
    settlements,
    variantNPCs,
    refinementQueue,
    activeTribulation,
    itemDictionary,
    actionLoading,
    fetchAperture,
    fetchDaoMarks,
    fetchDaoNgan,
    fetchApertureNodes,
    fetchVariantHumans,
    fetchSettlements,
    buildSettlement,
    upgradeSettlement,
    fetchNPCs,
    fetchRefinementQueue,
    createRefinementJob,
    deleteRefinementJob,
    recipes,
    fetchRecipes,
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
    addToast,
    fetchDictionary,
    dictionaryLoaded,
    guDictionary,
  } = useGameStore();

  const [activeTab, setActiveTab] = useState<"overview" | "map" | "dao" | "tribulation" | "variant_humans">("overview");
  const [selectedCell, setSelectedCell] = useState<{ x: number; y: number } | null>(null);
  const [selectedNode, setSelectedNode] = useState<ApertureResourceNode | null>(null);
  const [showHealMenu, setShowHealMenu] = useState(false);

  const [daoGroupFilter, setDaoGroupFilter] = useState<string>("ALL");
  const [daoSearch, setDaoSearch] = useState<string>("" );
  const [showDaoGuide, setShowDaoGuide] = useState<boolean>(false);

  const [sliderRate, setSliderRate] = useState<number>(aperture?.time_rate || 10.0);
  const [variantSubTab, setVariantSubTab] = useState<"overview" | "settlements" | "npcs" | "refine" | "ecosystem">("overview");

  useEffect(() => {
    if (variantSubTab === "refine") {
      fetchRecipes();
    }
  }, [variantSubTab, fetchRecipes]);
  const [showBuildSettlementModal, setShowBuildSettlementModal] = useState<boolean>(false);
  const [buildRace, setBuildRace] = useState<string>("HAIRY_MAN");
  const [buildType, setBuildType] = useState<string>("VILLAGE");
  const [buildName, setBuildName] = useState<string>("");

  const [showBuildRefineModal, setShowBuildRefineModal] = useState<boolean>(false);
  const [refineRecipeId, setRefineRecipeId] = useState<string>("GU_LIQUOR_WORM");
  const [refineMode, setRefineMode] = useState<string>("FIXED_COUNT");
  const [refineTargetQty, setRefineTargetQty] = useState<number>(10);
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
    if (!dictionaryLoaded) {
      fetchDictionary();
    }
    fetchAperture();
    fetchDaoMarks();
    fetchDaoNgan();
    fetchActiveTribulation();
  }, []);

  useEffect(() => {
    if (activeTab === "dao") {
      fetchDaoMarks();
      fetchDaoNgan();
    } else if (activeTab === "map") {
      fetchApertureNodes();
    } else if (activeTab === "tribulation") {
      fetchActiveTribulation();
    } else if (activeTab === "variant_humans") {
      fetchVariantHumans();
      fetchApertureNodes();
      fetchSettlements();
      fetchNPCs();
      fetchRefinementQueue();
    }
  }, [activeTab]);

  const handleReloadAll = async () => {
    setIsReloading(true);
    await Promise.all([
      fetchAperture(),
      fetchDaoMarks(),
      fetchDaoNgan(),
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
        if (typeof document !== "undefined" && document.visibilityState !== "visible") return;
        fetchApertureNodes(true);
        fetchActiveTribulation(true);
        // Also fetch aperture to update aperture HP if taking damage
        fetchAperture(true);
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
  const bulkCollectableNodes = apertureNodes.filter(n => !n.is_destroyed && n.node_category !== "DEFENSE" && canCollect(n, effectiveTimeRate));
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
          { id: "variant_humans", label: "Dị Nhân", icon: <Users className="w-4 h-4 shrink-0" />, disabled: false },
        ]}
        activeId={activeTab}
        onChange={(id) => setActiveTab(id as ApiResponseData)}
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
                    <span className="text-rose-400 font-semibold">
                      {player?.race === "cuong_thi" ? "Tử Khiếu (Ngưng trệ)" : `${Math.round(aperture.next_tribulation_year)} năm`}
                    </span>
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
                      {player?.race === "cuong_thi" ? "0.0% (Tử Khiếu)" : `${calculateTribulationProgress(aperture, player?.race).toFixed(1)}%`}
                    </span>
                  </div>
                  <div className="w-full bg-slate-950/80 h-2 rounded-full overflow-hidden border border-slate-800">
                    <div
                      className="h-full bg-gradient-to-r from-cyan-500 to-amber-500"
                      style={{ width: `${calculateTribulationProgress(aperture, player?.race)}%` }}
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
                        <span className="font-bold text-rose-300 text-sm">
                          {getRankName(rank)} - {getTribulationTypeName(activeTribulation.event.tribulation_type)}
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
                              <div className="font-bold flex items-center gap-1.5">
                                <span>{enemy.is_boss ? "👑" : "👾"}</span>
                                <span>{enemy.name}</span>
                                {enemy.rank && (
                                  <span className="text-[10px] px-1.5 py-0.2 rounded bg-rose-950/80 border border-rose-500/30 text-rose-300 font-semibold">
                                    {getRankName(enemy.rank)}
                                  </span>
                                )}
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

                             const parseItemPaths = (pathVal: ApiResponseData): string[] => {
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
                                    if ((dp === mappedKey || (mappedKey === "killing" && dp === "kill")) && d.amount > maxMarks) {
                                      maxMarks = d.amount;
                                    }
                                  }
                                }
                              }
                              return maxMarks;
                            };

                            const hasSovereignFetus = (player?.gu_inventory || []).some(
                              (g: ApiResponseData) => (g.gu_id || g.id) === "GU_SOVEREIGN_FETUS" && g.equipped
                            );

                            const levelMultiplier = 1.0 + (Math.max(1, selectedNode.level) - 1) * 0.20;
                            const nodePathMarks = getDaoMarkAmount(selectedNode.path);
                            const daoMultiplier = 1.0 + nodePathMarks * 0.0001;
                            const conflictMult = (isConflicting && !hasSovereignFetus) ? 0.5 : 1.0;
                            let ecoMult = 1.0;
                            if (overgrazed) ecoMult = 0.5;
                            else if (starvingHerb || starvingCarn) ecoMult = 0.2;
                            else if (isPlant && hCount > 0) ecoMult = 1.25;

                            let vhLaborMult = 1.0;
                            let vhPopBonus = 1.0;
                            const vhRace = selectedNode.node_type === "HAIRY_MAN_SETTLEMENT" ? "HAIRY_MAN"
                              : (selectedNode.node_type === "FEATHERMAN_CANYON" ? "FEATHERMAN"
                              : (isMine || selectedNode.node_type === "ROCKMAN_TRIBE" ? "ROCKMAN" : ""));
                            const vh = vhRace ? variantHumans.find(v => v.race === vhRace) : undefined;
                            const vhPop = vh ? vh.population : 0;
                            const vhLabor = vh ? vh.labor_intensity : "NORMAL";
                            const laborLabels: Record<string, string> = { LOW: "x0.7 (Nhàn)", NORMAL: "x1.0", HIGH: "x1.5", EXPLOIT: "x2.0 (Bóc lột)" };

                            if (vhRace && vhPop > 0) {
                              vhPopBonus = 1.0 + vhPop * 0.001;
                              if (vhLabor === "LOW") vhLaborMult = 0.7;
                              if (vhLabor === "HIGH") vhLaborMult = 1.5;
                              if (vhLabor === "EXPLOIT") vhLaborMult = 2.0;
                            }

                            const getManaVal = (rank: number): number => getRankScale(rank);

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
                                  itemFoodPathMultiplier = 1.0 + itemPathMarks * 0.0001;
                                }
                              }
                            }

                            const combinedMultiplier = (areaMultiplier * daoMultiplier * conflictMult * ecoMult * vhLaborMult * vhPopBonus * itemFoodPathMultiplier * levelMultiplier) / itemRankDivisor;
                            const baseProd = selectedNode.production_amount || 0;
                            let rawExpected = baseProd > 0 ? Math.max(1, Math.floor(baseProd * combinedMultiplier)) : 0;
                            if (selectedNode.production_type === "Material" && rawExpected > 10000000) {
                              rawExpected = 10000000;
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
                                    Hiệu suất thực tế: {efficiencyPercent.toLocaleString("vi-VN")}% ({combinedMultiplier >= 1.0 ? `+${Math.round((combinedMultiplier - 1) * 100).toLocaleString("vi-VN")}%` : `-${Math.round((1 - combinedMultiplier) * 100).toLocaleString("vi-VN")}%`})
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
                                      {((displayMarks * 0.01) / itemRankDivisor) >= 10000
                                        ? `+${formatNumber((displayMarks * 0.01) / itemRankDivisor)}%`
                                        : `+${((displayMarks * 0.01) / itemRankDivisor).toFixed(2)}%`}
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
                                  {vhRace && vhPop > 0 && (
                                    <div className="space-y-1 bg-slate-900/60 p-2 rounded-lg border border-slate-850 text-[10px]">
                                      <div className="flex justify-between items-center">
                                        <span className="text-slate-300">{vhRace === "HAIRY_MAN" ? "🦁 Dân số Mao Nhân:" : (vhRace === "FEATHERMAN" ? "🪶 Dân số Vũ Nhân:" : "🪨 Dân số Người Đá:")}</span>
                                        <span className="text-cyan-400 font-bold">{vhPop.toLocaleString("vi-VN")} (+{(vhPop * 0.1).toFixed(1)}%)</span>
                                      </div>
                                      <div className="flex justify-between items-center">
                                        <span className="text-slate-300">⚙️ Cường độ lao động:</span>
                                        <span className="text-amber-400 font-bold">{laborLabels[vhLabor] || "x1.0"}</span>
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
                                <CountdownTimer node={selectedNode} timeRate={effectiveTimeRate} />
                              </span>
                            </div>
                          )}

                          {/* Collect timing indicator - only for resource nodes */}
                          {!selectedNode.is_destroyed && selectedNode.node_category !== "DEFENSE" && (
                            <div className="pt-2">
                              {canCollect(selectedNode, effectiveTimeRate) ? (
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
                                  setBuildCategory(cat as ApiResponseData);
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
                                  const parseItemPaths = (pathVal: ApiResponseData): string[] => {
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
                                      const itemType = (item.type || item.item_type || "").toString();
                                      if (["Currency", "Consumable"].includes(itemType)) return false;
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

        {/* TAB 3: DAO MARKS & DAO COMBAT PROFILE */}
        {activeTab === "dao" && (
          <div className="space-y-6">
            {/* Header & Controls Panel */}
            <div className="bg-slate-900/60 backdrop-blur-md border border-slate-700/40 rounded-2xl p-6 space-y-5">
              <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-slate-800 pb-4">
                <div>
                  <h2 className="text-xl font-bold text-slate-100 flex items-center gap-2">
                    <Layers className="w-5 h-5 text-amber-400" />
                    Bảng Đạo Ngân & Uy Lực Tiên Khiếu
                  </h2>
                  <p className="text-xs text-slate-400 mt-1">
                    Cơ chế tính toán toàn diện theo hệ thống Đạo Ngân, Xung Đột Lưu Phái, Cảnh Giới Chân Ý và Màng Kháng Bản Nguyên.
                  </p>
                </div>

                <div className="flex items-center gap-2">
                  <button
                    onClick={() => setShowDaoGuide(!showDaoGuide)}
                    className="flex items-center gap-1.5 px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-amber-400 border border-amber-500/30 rounded-lg text-xs font-semibold transition"
                  >
                    <BookOpen className="w-4 h-4" />
                    {showDaoGuide ? "Ẩn Cơ Chế" : "Cơ Chế Đạo Ngân"}
                  </button>
                  <button
                    onClick={() => {
                      fetchDaoMarks(true);
                      fetchDaoNgan(true);
                    }}
                    disabled={actionLoading || isReloading}
                    className="flex items-center gap-1.5 px-3 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-lg text-xs transition"
                  >
                    <RefreshCw className={`w-3.5 h-3.5 ${isReloading ? "animate-spin" : ""}`} />
                    Làm Mới
                  </button>
                </div>
              </div>

              {/* Explainer Box if showDaoGuide is open */}
              {showDaoGuide && (
                <div className="p-4 bg-slate-950/80 border border-amber-500/20 rounded-xl space-y-3 text-xs text-slate-300">
                  <div className="font-bold text-amber-300 flex items-center gap-2 text-sm">
                    <Info className="w-4 h-4 text-amber-400" />
                    Quy Tắc Vận Hành Đạo Ngân (Dao Combat Mechanics)
                  </div>
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3 pt-1">
                    <div className="p-3 bg-slate-900/80 border border-slate-800 rounded-lg space-y-1">
                      <div className="font-bold text-slate-100 flex items-center gap-1.5">
                        <TrendingUp className="w-3.5 h-3.5 text-emerald-400" />
                        1. Tăng Phúc Uy Lực
                      </div>
                      <p className="text-slate-400 leading-relaxed">
                        Mỗi <strong className="text-amber-400">100 Đạo Ngân</strong> tăng +100% (+1.0x) sát thương và phòng thủ tương ứng khi thi triển cổ chiêu cùng hệ.
                      </p>
                    </div>

                    <div className="p-3 bg-slate-900/80 border border-slate-800 rounded-lg space-y-1">
                      <div className="font-bold text-slate-100 flex items-center gap-1.5">
                        <ShieldAlert className="w-3.5 h-3.5 text-rose-400" />
                        2. Xung Đột & Ngưỡng An Toàn
                      </div>
                      <p className="text-slate-400 leading-relaxed">
                        Ngưỡng an toàn ngoại hệ là <strong className="text-cyan-400">{daoNgan?.grace_threshold ? daoNgan.grace_threshold.toLocaleString() : "1.000"} Đạo Ngân</strong>. Vượt qua ngưỡng này sẽ gây xung đột nội thể làm suy giảm bội số công kích.
                      </p>
                    </div>

                    <div className="p-3 bg-slate-900/80 border border-slate-800 rounded-lg space-y-1">
                      <div className="font-bold text-slate-100 flex items-center gap-1.5">
                        <Award className="w-3.5 h-3.5 text-purple-400" />
                        3. Cảnh Giới Chân Ý
                      </div>
                      <p className="text-slate-400 leading-relaxed">
                        Cảnh giới lưu phái (Đại Sư, Tông Sư, Đại Tông Sư) giúp <strong className="text-purple-300">điều hòa xung đột</strong> (+10%~50%) và <strong className="text-purple-300">nhân bội hiệu suất</strong> (1.05x ~ 1.30x).
                      </p>
                    </div>

                    <div className="p-3 bg-slate-900/80 border border-slate-800 rounded-lg space-y-1">
                      <div className="font-bold text-slate-100 flex items-center gap-1.5">
                        <ShieldCheck className="w-3.5 h-3.5 text-blue-400" />
                        4. Màng Kháng Tự Nhiên
                      </div>
                      <p className="text-slate-400 leading-relaxed">
                        Đạo Ngân hình thành lớp màng bảo hộ tự nhiên, giúp <strong className="text-blue-300">giảm trực tiếp sát thương</strong> nhận vào khi bị kẻ địch tấn công bằng chiêu thức cùng hệ.
                      </p>
                    </div>
                  </div>
                </div>
              )}

              {/* 3 Top Summary KPI Cards */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                {/* Total Marks & Power */}
                <div className="p-4 bg-gradient-to-br from-amber-950/40 to-slate-950/80 border border-amber-500/30 rounded-xl space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Tổng Đạo Ngân Tích Lũy</span>
                    <Layers className="w-4 h-4 text-amber-400" />
                  </div>
                  <div className="flex items-baseline gap-2">
                    <span className="text-2xl font-black text-amber-400">
                      <Num value={daoNgan?.total_marks ?? daoMarks.reduce((acc, m) => acc + m.amount, 0)} />
                    </span>
                    <span className="text-xs text-slate-400">Đạo Ngân</span>
                  </div>
                  <div className="pt-1 border-t border-slate-800 flex items-center justify-between text-xs text-slate-400">
                    <span>Điểm Uy Lực Tổng:</span>
                    <span className="font-bold text-amber-300">
                      <Num value={daoNgan?.total_dao_ngan ?? 0} />
                    </span>
                  </div>
                </div>

                {/* Dominant Path */}
                <div className="p-4 bg-gradient-to-br from-purple-950/40 to-slate-950/80 border border-purple-500/30 rounded-xl space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Lưu Phái Chủ Tu</span>
                    <Star className="w-4 h-4 text-yellow-400 fill-yellow-400" />
                  </div>
                  <div className="flex items-center gap-2">
                    {daoNgan?.dominant_path ? (
                      <>
                        <span className="text-xl font-black text-purple-200">
                          {translatePath(daoNgan.dominant_path)}
                        </span>
                        <span className="text-xs text-purple-400/80 font-mono">
                          ({daoNgan.dominant_path})
                        </span>
                      </>
                    ) : (
                      <span className="text-sm font-semibold text-slate-500">Chưa xác định</span>
                    )}
                  </div>
                  <div className="pt-1 border-t border-slate-800 flex items-center justify-between text-xs text-slate-400">
                    <span>Độ Tinh Thuần:</span>
                    <span className="font-bold text-purple-300">
                      {daoNgan?.paths?.find(p => p.path === daoNgan?.dominant_path)?.purity_pct.toFixed(1) ?? "0"}%
                    </span>
                  </div>
                </div>

                {/* Dao Body & Grace Threshold */}
                <div className="p-4 bg-gradient-to-br from-cyan-950/40 to-slate-950/80 border border-cyan-500/30 rounded-xl space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Đặc Tính Đạo Thể</span>
                    <Sparkles className="w-4 h-4 text-cyan-400" />
                  </div>
                  <div className="text-sm font-bold">
                    {daoNgan?.has_non_conflicting_dao ? (
                      <span className="text-cyan-300 flex items-center gap-1.5">
                        ✨ Vô Xung Đột (Chí Tôn)
                      </span>
                    ) : (
                      <span className="text-slate-300">Đạo Thể Tương Khắc Bình Thường</span>
                    )}
                  </div>
                  <div className="pt-1 border-t border-slate-800 flex items-center justify-between text-xs text-slate-400">
                    <span>Ngưỡng An Toàn Ngoại Hệ:</span>
                    <span className="font-bold text-cyan-300">
                      {daoNgan?.grace_threshold ? daoNgan.grace_threshold.toLocaleString() : "1.000"} Đạo
                    </span>
                  </div>
                </div>
              </div>

              {/* Group Filter & Search */}
              <div className="space-y-3 pt-2">
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                  <div className="flex items-center gap-2 overflow-x-auto pb-1 max-w-full scrollbar-thin">
                    {DAO_GROUPS.map((grp) => {
                      const count = grp.id === "ALL"
                        ? (daoNgan?.paths?.length ?? daoMarks.length)
                        : (daoNgan?.paths ?? []).filter(p => (PATH_TO_DAO_GROUP[p.path.toLowerCase()] || "ELEMENTAL") === grp.id).length;
                      
                      const isSelected = daoGroupFilter === grp.id;
                      return (
                        <button
                          key={grp.id}
                          onClick={() => setDaoGroupFilter(grp.id)}
                          className={`px-3 py-1.5 rounded-lg text-xs font-medium whitespace-nowrap transition flex items-center gap-1.5 border ${
                            isSelected
                              ? `${grp.color} border-current shadow-lg shadow-black/40`
                              : "bg-slate-950/50 text-slate-400 border-slate-800 hover:border-slate-700 hover:text-slate-200"
                          }`}
                        >
                          <span>{grp.icon}</span>
                          <span>{grp.name}</span>
                          {count > 0 && (
                            <span className={`px-1.5 py-0.2 rounded-full text-[10px] font-bold ${
                              isSelected ? "bg-white/20 text-white" : "bg-slate-800 text-slate-400"
                            }`}>
                              {count}
                            </span>
                          )}
                        </button>
                      );
                    })}
                  </div>

                  <div className="relative shrink-0 w-full sm:w-56">
                    <Search className="w-3.5 h-3.5 absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
                    <input
                      type="text"
                      placeholder="Tìm lưu phái..."
                      value={daoSearch}
                      onChange={(e) => setDaoSearch(e.target.value)}
                      className="w-full pl-9 pr-3 py-1.5 bg-slate-950/70 border border-slate-800 focus:border-amber-500/50 rounded-lg text-xs text-slate-200 placeholder:text-slate-600 outline-none transition"
                    />
                  </div>
                </div>
              </div>
            </div>

            {/* List of Path Cards */}
            {(() => {
              const pathsList = daoNgan?.paths && daoNgan.paths.length > 0
                ? daoNgan.paths
                : daoMarks.map(m => ({
                    path: m.path,
                    amount: m.amount,
                    multiplier: 1.0 + m.amount / 100.0,
                    base_multiplier: 1.0 + m.amount / 100.0,
                    power: m.amount * (1.0 + m.amount / 100.0),
                    attainment_level: 0,
                    attainment_name: "Phổ Thông",
                    realm_control_pct: 0,
                    realm_efficiency: 1.0,
                    internal_conflict_penalty_pct: 0,
                    natural_resistance_pct: 0,
                    purity_pct: daoMarks.length > 0 ? (m.amount / daoMarks.reduce((a, b) => a + b.amount, 0)) * 100 : 100,
                  }));

              const filteredPaths = pathsList.filter(p => {
                const group = PATH_TO_DAO_GROUP[p.path.toLowerCase()] || "ELEMENTAL";
                if (daoGroupFilter !== "ALL" && group !== daoGroupFilter) return false;
                if (daoSearch.trim()) {
                  const query = daoSearch.toLowerCase();
                  const vnName = translatePath(p.path).toLowerCase();
                  const enName = p.path.toLowerCase();
                  if (!vnName.includes(query) && !enName.includes(query)) return false;
                }
                return true;
              });

              if (filteredPaths.length === 0) {
                return (
                  <div className="p-12 text-center text-slate-500 border border-slate-800/60 rounded-2xl bg-slate-900/30">
                    <Layers className="w-10 h-10 mx-auto text-slate-600 mb-3 opacity-50" />
                    <p className="text-sm font-medium text-slate-400">Không tìm thấy Đạo Ngân phù hợp bộ lọc.</p>
                    <p className="text-xs text-slate-600 mt-1">Hãy thử đổi nhóm Đạo hoặc phòng thủ vượt qua kiếp nạn để nhận thêm Đạo Ngân!</p>
                  </div>
                );
              }

              return (
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                  {filteredPaths.map((detail) => {
                    const isDominant = (daoNgan?.dominant_path || "").toLowerCase() === detail.path.toLowerCase();
                    const groupKey = PATH_TO_DAO_GROUP[detail.path.toLowerCase()] || "ELEMENTAL";
                    const groupDef = DAO_GROUPS.find(g => g.id === groupKey) || DAO_GROUPS[1];
                    const attainmentBadge = getAttainmentBadge(detail.attainment_name, detail.attainment_level);
                    const pathIconColor = getPathIconAndColor(detail.path, undefined);

                    return (
                      <div
                        key={detail.path}
                        className={`p-5 bg-slate-950/70 border rounded-2xl transition hover:border-slate-700 space-y-4 relative overflow-hidden ${
                          isDominant
                            ? "border-amber-500/40 shadow-lg shadow-amber-500/5 ring-1 ring-amber-500/20"
                            : "border-slate-800/80"
                        }`}
                      >
                        {/* Background subtle accent */}
                        <div className="absolute -top-12 -right-12 w-32 h-32 bg-amber-500/5 rounded-full blur-2xl pointer-events-none" />

                        {/* Card Header */}
                        <div className="flex items-start justify-between gap-3">
                          <div className="flex items-center gap-3">
                            <div className={`w-11 h-11 rounded-xl flex items-center justify-center text-xl border ${pathIconColor.color}`}>
                              {pathIconColor.icon}
                            </div>
                            <div>
                              <div className="flex items-center gap-2">
                                <span className="font-extrabold text-base text-slate-100">
                                  {translatePath(detail.path)}
                                </span>
                                {isDominant && (
                                  <span className="px-2 py-0.5 rounded-md text-[10px] font-black bg-gradient-to-r from-amber-500/30 to-yellow-500/30 text-amber-300 border border-amber-400/40 flex items-center gap-1 shadow-sm">
                                    <Star className="w-3 h-3 fill-amber-300 text-amber-300" />
                                    CHỦ TU
                                  </span>
                                )}
                              </div>
                              <div className="flex items-center gap-2 mt-0.5">
                                <span className="text-[11px] text-slate-500 font-mono uppercase">{detail.path} Path</span>
                                <span className="text-slate-700">•</span>
                                <span className={`px-1.5 py-0.5 rounded text-[10px] font-medium border ${groupDef.badgeBg}`}>
                                  {groupDef.icon} {groupDef.name}
                                </span>
                              </div>
                            </div>
                          </div>

                          <div className="text-right shrink-0">
                            <span className={`inline-block px-2.5 py-1 rounded-lg text-xs font-bold border ${attainmentBadge.bg} ${attainmentBadge.textCol} ${attainmentBadge.border}`}>
                              {attainmentBadge.text}
                            </span>
                          </div>
                        </div>

                        {/* Marks & Purity Progress */}
                        <div className="space-y-1.5 p-3 bg-slate-900/60 rounded-xl border border-slate-850">
                          <div className="flex items-center justify-between text-xs">
                            <span className="text-slate-400">Tích lũy Đạo Ngân:</span>
                            <span className="font-bold text-amber-400">
                              <Num value={detail.amount} /> Đạo Ngân
                            </span>
                          </div>
                          <div className="flex items-center justify-between text-[11px] text-slate-500">
                            <span>Tỷ lệ tinh thuần:</span>
                            <span className="text-purple-300 font-medium">{detail.purity_pct.toFixed(1)}%</span>
                          </div>
                          <div className="w-full bg-slate-950 rounded-full h-1.5 overflow-hidden border border-slate-800">
                            <div
                              className="bg-gradient-to-r from-amber-500 to-purple-500 h-full rounded-full transition-all duration-500"
                              style={{ width: `${Math.min(100, Math.max(0, detail.purity_pct))}%` }}
                            />
                          </div>
                        </div>

                        {/* Combat Modifiers Grid (4 Tiles) */}
                        <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 pt-1">
                          {/* Multiplier */}
                          <div className="p-2.5 bg-slate-900/40 border border-slate-800/80 rounded-xl space-y-1">
                            <div className="text-[10px] text-slate-500 font-medium">Khuếch Đại Thực Chiến</div>
                            <div className="text-sm font-black text-emerald-400 flex items-center gap-0.5">
                              <TrendingUp className="w-3.5 h-3.5 shrink-0" />
                              x{detail.multiplier >= 10000 ? formatNumber(detail.multiplier) : detail.multiplier.toFixed(2)}
                            </div>
                            <div className="text-[9px] text-slate-500" title={`Gốc: x${(detail.base_multiplier || (1 + detail.amount / 100)).toFixed(1)}`}>
                              Gốc: x{formatNumber(detail.base_multiplier || (1 + detail.amount / 100))}
                            </div>
                          </div>

                          {/* Attainment Efficiency */}
                          <div className="p-2.5 bg-slate-900/40 border border-slate-800/80 rounded-xl space-y-1">
                            <div className="text-[10px] text-slate-500 font-medium">Hiệu Suất Chân Ý</div>
                            <div className="text-sm font-black text-purple-300">
                              x{(detail.realm_efficiency || 1.0).toFixed(2)}
                            </div>
                            <div className="text-[9px] text-slate-500">
                              Điều hòa: +{(detail.realm_control_pct || 0).toFixed(0)}%
                            </div>
                          </div>

                          {/* Conflict Penalty */}
                          <div className="p-2.5 bg-slate-900/40 border border-slate-800/80 rounded-xl space-y-1">
                            <div className="text-[10px] text-slate-500 font-medium">Phạt Xung Đột</div>
                            <div className={`text-sm font-black ${detail.internal_conflict_penalty_pct > 0 ? "text-rose-400" : "text-emerald-400"}`}>
                              {detail.internal_conflict_penalty_pct > 0 ? `-${detail.internal_conflict_penalty_pct.toFixed(1)}%` : "0%"}
                            </div>
                            <div className="text-[9px] text-slate-600">
                              {detail.internal_conflict_penalty_pct > 0 ? "Suy giảm do ngoại hệ" : "Hoàn toàn an toàn"}
                            </div>
                          </div>

                          {/* Natural Resistance */}
                          <div className="p-2.5 bg-slate-900/40 border border-slate-800/80 rounded-xl space-y-1">
                            <div className="text-[10px] text-slate-500 font-medium">Kháng Tự Nhiên</div>
                            <div className="text-sm font-black text-cyan-400">
                              +{(detail.natural_resistance_pct || 0).toFixed(1)}%
                            </div>
                            <div className="text-[9px] text-slate-600">Giảm sát thương nhận</div>
                          </div>
                        </div>

                        {/* Power Contribution Footer */}
                        <div className="pt-2 border-t border-slate-850 flex items-center justify-between text-xs text-slate-400">
                          <span>Uy lực đóng góp:</span>
                          <span className="font-bold text-amber-400">
                            <Num value={detail.power} /> Điểm
                          </span>
                        </div>
                      </div>
                    );
                  })}
                </div>
              );
            })()}
          </div>
        )}


        {/* TAB 5: VARIANT HUMANS — Dinh Nhan Tien Khieu */}
        {activeTab === "variant_humans" && (
          <div className="space-y-6">
            {/* Sub-tab Navigation */}
            <GameTabs
              items={[
                { id: "overview", label: "Tổng Quan", icon: <Users className="w-4 h-4 shrink-0" /> },
                { id: "settlements", label: "Thành Trì", icon: <MapIcon className="w-4 h-4 shrink-0" /> },
                { id: "npcs", label: "Cổ Sư", icon: <Sparkles className="w-4 h-4 shrink-0" /> },
                { id: "refine", label: "Luyện Cổ Tự Động", icon: <Flame className="w-4 h-4 shrink-0" /> },
                { id: "ecosystem", label: "Sinh Thái", icon: <Leaf className="w-4 h-4 shrink-0" /> },
              ]}
              activeId={variantSubTab}
              onChange={(id) => setVariantSubTab(id as ApiResponseData)}
              variant="pills"
              size="sm"
              scrollable={true}
            />

            {/* SUB-TAB: OVERVIEW — Race Cards */}
            {variantSubTab === "overview" && (
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
                        const prosperity = record ? record.prosperity : 50;
                        const guMasters = record ? record.gu_master_count : 0;

                        const settlementCount = apertureNodes.filter(
                          (n) => !n.is_destroyed && n.node_type === raceInfo.settlementType
                        ).length;
                        const maxPop = 200 * settlementCount;

                        const rebellionRate = (100.0 - happiness + oppression) / 4.0;
                        const showRebellionAlert = happiness < 30.0 || oppression > 70.0;

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
                            confirmText = `⚠️ CẢNH BÁO: Thiết lập cường độ 'Lao Dịch' (x1.5 sản lượng) sẽ giảm 10 Hạnh phúc và tăng 10 Áp bức của tộc này mỗi năm.`;
                          } else if (intensity === "EXPLOIT") {
                            confirmText = `💥 CẢNH BÁO CỰC KỲ NGUY HIỂM: Thiết lập cường độ 'Bóc Lột' (x2.0 sản lượng) sẽ khiến tộc nhân bị kiệt quệ, giảm 25 Hạnh phúc và tăng 25 Áp bức mỗi năm!`;
                          }

                          if (await showConfirm(confirmText)) {
                            await handleSetLabor(raceInfo.race, intensity);
                          }
                        };

                        return (
                          <div key={raceInfo.race} className="bg-slate-950/60 border border-slate-850 hover:border-slate-800 rounded-2xl p-5 flex flex-col justify-between space-y-5 transition-all">
                            <div className="space-y-3">
                              <div className="flex items-center gap-3">
                                <span className="text-3xl select-none">{raceInfo.icon}</span>
                                <div>
                                  <h3 className="font-extrabold text-sm text-slate-100">{raceInfo.name}</h3>
                                  <span className="text-[10px] text-slate-500">Định cư: {settlementCount} bộ lạc</span>
                                </div>
                              </div>
                              <p className="text-[10px] text-slate-400 leading-relaxed min-h-[40px]">{raceInfo.desc}</p>

                              <div className="space-y-1.5">
                                <div className="flex justify-between text-xs">
                                  <span className="text-slate-400">Dân số:</span>
                                  <span className="font-bold text-slate-200">{population} / {maxPop}</span>
                                </div>
                                <div className="w-full bg-slate-900 h-2 rounded-full overflow-hidden border border-slate-800">
                                  <div className="h-full bg-cyan-500 transition-all duration-500"
                                    style={{ width: `${maxPop > 0 ? Math.min(100, (population / maxPop) * 100) : 0}%` }} />
                                </div>
                              </div>

                              <div className="space-y-2 pt-1">
                                <div className="space-y-1">
                                  <div className="flex justify-between text-[10px]">
                                    <span className="text-slate-400">Hạnh Phúc:</span>
                                    <span className={`font-bold ${happiness < 30 ? "text-rose-400" : happiness > 60 ? "text-emerald-400" : "text-amber-400"}`}>
                                      {happiness.toFixed(1)}
                                    </span>
                                  </div>
                                  <div className="w-full bg-slate-900 h-1.5 rounded-full overflow-hidden">
                                    <div className={`h-full transition-all duration-500 ${happiness < 30 ? "bg-rose-500" : happiness > 60 ? "bg-emerald-500" : "bg-amber-500"}`}
                                      style={{ width: `${happiness}%` }} />
                                  </div>
                                </div>
                                <div className="space-y-1">
                                  <div className="flex justify-between text-[10px]">
                                    <span className="text-slate-400">Áp Bức:</span>
                                    <span className={`font-bold ${oppression > 70 ? "text-rose-400" : "text-slate-400"}`}>
                                      {oppression.toFixed(1)}
                                    </span>
                                  </div>
                                  <div className="w-full bg-slate-900 h-1.5 rounded-full overflow-hidden">
                                    <div className={`h-full transition-all duration-500 ${oppression > 70 ? "bg-rose-500" : "bg-slate-500"}`}
                                      style={{ width: `${oppression}%` }} />
                                  </div>
                                </div>
                              </div>

                              <div className="grid grid-cols-3 gap-2 text-center text-[9px]">
                                <div className="p-1.5 bg-slate-900/60 rounded-lg">
                                  <div className="text-slate-500">Thịnh Vượng</div>
                                  <div className="font-bold text-amber-400">{prosperity.toFixed(0)}</div>
                                </div>
                                <div className="p-1.5 bg-slate-900/60 rounded-lg">
                                  <div className="text-slate-500">Cổ Sư</div>
                                  <div className="font-bold text-purple-400">{guMasters}</div>
                                </div>
                                <div className="p-1.5 bg-slate-900/60 rounded-lg">
                                  <div className="text-slate-500">Nổi Loạn</div>
                                  <div className={`font-bold ${showRebellionAlert ? "text-rose-400" : "text-emerald-400"}`}>{rebellionRate.toFixed(0)}%</div>
                                </div>
                              </div>

                              <div className="p-2.5 bg-slate-900/60 border border-slate-800/80 rounded-lg flex items-center justify-between text-[10px]">
                                <span className="text-slate-400">Nguy cơ:</span>
                                <span className={`font-black ${showRebellionAlert ? "text-rose-400 animate-pulse" : "text-amber-400"}`}>
                                  {showRebellionAlert ? "Nguy hiểm" : "Ổn định"}
                                </span>
                              </div>
                            </div>

                            <div className="space-y-4 pt-3 border-t border-slate-850">
                              <div className="space-y-1.5">
                                <label className="text-[10px] text-slate-400 font-semibold block">Cường Độ Lao Động</label>
                                <div className="grid grid-cols-2 gap-1 text-[9px] font-bold">
                                  {["LOW", "NORMAL", "HIGH", "EXPLOIT"].map((mode) => (
                                    <button key={mode} onClick={() => handleLaborChange(mode)} disabled={actionLoading}
                                      className={`py-1.5 px-1 rounded transition border ${laborIntensity === mode ? "bg-amber-500/20 border-amber-500/40 text-amber-300" : "bg-slate-900 border-transparent text-slate-400 hover:text-slate-200"}`}>
                                      {mode === "LOW" ? "Nhàn Nhã" : mode === "NORMAL" ? "Bình Thường" : mode === "HIGH" ? "Lao Dịch" : "Bóc Lột"}
                                    </button>
                                  ))}
                                </div>
                              </div>
                              <div className="p-3 bg-slate-900/40 border border-slate-850 rounded-xl text-[10px] flex items-center justify-between gap-2">
                                <div>
                                  <div className="text-slate-400 font-medium">{raceInfo.buffLabel}</div>
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

            {/* SUB-TAB: SETTLEMENTS */}
            {variantSubTab === "settlements" && (
              <div className="bg-slate-900/60 backdrop-blur-md border border-slate-700/40 rounded-2xl p-6 space-y-6">
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-slate-800 pb-4">
                  <div>
                    <h2 className="text-xl font-bold text-slate-100 flex items-center gap-2">
                      <MapIcon className="w-5 h-5 text-cyan-400" />
                      Quản Lý Thành Trì & Định Cư Dị Nhân
                    </h2>
                    <p className="text-xs text-slate-400 mt-1">
                      Xây dựng và nâng cấp các cơ sở định cư cho từng chủng tộc để mở rộng sức chứa dân số, nhà ở và kho lương thực.
                    </p>
                  </div>
                  <button
                    onClick={() => setShowBuildSettlementModal(true)}
                    disabled={actionLoading}
                    className="px-4 py-2 bg-gradient-to-r from-cyan-600 to-blue-600 hover:from-cyan-500 hover:to-blue-500 text-white text-xs font-bold rounded-xl shadow-lg transition-all flex items-center gap-2 shrink-0 self-start sm:self-auto"
                  >
                    <Plus className="w-4 h-4" />
                    Xây Thành Trì Mới
                  </button>
                </div>

                {/* Summary Header */}
                <div className="grid grid-cols-2 md:grid-cols-4 gap-3 text-xs">
                  <div className="p-3 bg-slate-950/60 border border-slate-800 rounded-xl">
                    <div className="text-slate-400">Tổng Thành Trì</div>
                    <div className="text-base font-bold text-cyan-400 mt-0.5">{settlements.length} Cơ sở</div>
                  </div>
                  <div className="p-3 bg-slate-950/60 border border-slate-800 rounded-xl">
                    <div className="text-slate-400">Tổng Sức Chứa Dân Số</div>
                    <div className="text-base font-bold text-emerald-400 mt-0.5">
                      {settlements.reduce((acc, s) => acc + s.population_capacity, 0).toLocaleString()}
                    </div>
                  </div>
                  <div className="p-3 bg-slate-950/60 border border-slate-800 rounded-xl">
                    <div className="text-slate-400">Tổng Mức Nhà Ở</div>
                    <div className="text-base font-bold text-amber-400 mt-0.5">
                      {settlements.reduce((acc, s) => acc + s.housing, 0).toLocaleString()}
                    </div>
                  </div>
                  <div className="p-3 bg-slate-950/60 border border-slate-800 rounded-xl">
                    <div className="text-slate-400">Kho Lương Thực</div>
                    <div className="text-base font-bold text-purple-400 mt-0.5">
                      {settlements.reduce((acc, s) => acc + s.food_storage, 0).toLocaleString()}
                    </div>
                  </div>
                </div>

                {/* Settlements Grid */}
                {settlements.length === 0 ? (
                  <div className="text-center text-xs text-slate-500 py-12 bg-slate-950/40 rounded-xl border border-dashed border-slate-800">
                    Chưa có thành trì nào được xây dựng. Nhấn "Xây Thành Trì Mới" để tạo nơi định cư cho bộ tộc dị nhân.
                  </div>
                ) : (
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {settlements.map((s) => {
                      const raceIcons: Record<string, string> = {
                        HAIRY_MAN: "🧔",
                        ROCKMAN: "🗿",
                        FEATHERMAN: "🦅",
                        SNOWMAN: "❄️",
                        INKMAN: "📜",
                        MINIMAN: "🌱",
                        MERMAN: "🧜",
                        MUSHROOMMAN: "🍄",
                        DRAGONMAN: "🐉",
                      };
                      const raceNames: Record<string, string> = {
                        HAIRY_MAN: "Tộc Người Lông",
                        ROCKMAN: "Tộc Người Đá",
                        FEATHERMAN: "Tộc Người Lông Vũ",
                        SNOWMAN: "Tộc Người Tuyết",
                        INKMAN: "Tộc Mặc Nhân",
                        MINIMAN: "Tộc Tiểu Nhân",
                        MERMAN: "Tộc Giao Nhân",
                        MUSHROOMMAN: "Tộc Cô Nhân",
                        DRAGONMAN: "Tộc Long Nhân",
                      };
                      const typeLabels: Record<string, { label: string; color: string; nextType?: string; nextLabel?: string }> = {
                        VILLAGE: { label: "Bản Làng (Cấp 1)", color: "text-slate-300 bg-slate-800", nextType: "TOWN", nextLabel: "Thị Trấn (50k)" },
                        TOWN: { label: "Thị Trấn (Cấp 2)", color: "text-cyan-300 bg-cyan-950/80 border-cyan-800", nextType: "CITY", nextLabel: "Thành Thị (250k)" },
                        CITY: { label: "Thành Thị (Cấp 3)", color: "text-amber-300 bg-amber-950/80 border-amber-800", nextType: "FORTRESS", nextLabel: "Pháo Đài (1M)" },
                        FORTRESS: { label: "Đại Pháo Đài (Tối Đa)", color: "text-rose-300 bg-rose-950/80 border-rose-800" },
                      };

                      const typeInfo = typeLabels[s.settlement_type] || { label: s.settlement_type, color: "text-slate-300 bg-slate-800" };

                      const handleUpgrade = async () => {
                        if (!typeInfo.nextType) {
                          addToast("Thành trì đã đạt cấp tối đa!", "info");
                          return;
                        }
                        const confirmed = await showConfirm(`Xác nhận nâng cấp ${s.name || raceNames[s.race]} lên ${typeInfo.nextLabel}?`);
                        if (confirmed) {
                          await upgradeSettlement(s.id, { settlement_type: typeInfo.nextType });
                        }
                      };

                      return (
                        <div key={s.id} className="bg-slate-950/80 border border-slate-800 hover:border-cyan-800/60 rounded-xl p-4 space-y-4 transition-all flex flex-col justify-between">
                          <div className="space-y-3">
                            <div className="flex items-start justify-between gap-2">
                              <div className="flex items-center gap-2.5">
                                <span className="text-3xl">{raceIcons[s.race] || "🏘️"}</span>
                                <div>
                                  <h3 className="font-bold text-sm text-slate-100">{s.name || raceNames[s.race]}</h3>
                                  <span className="text-[10px] text-slate-400">{raceNames[s.race] || s.race}</span>
                                </div>
                              </div>
                              <span className={`text-[9px] font-bold px-2 py-0.5 rounded border ${typeInfo.color}`}>
                                {typeInfo.label}
                              </span>
                            </div>

                            <div className="space-y-2 text-xs bg-slate-900/60 p-3 rounded-lg border border-slate-850">
                              <div className="flex justify-between items-center">
                                <span className="text-slate-400">Sức chứa dân số:</span>
                                <span className="text-cyan-400 font-bold">{s.population_capacity.toLocaleString()}</span>
                              </div>
                              <div className="flex justify-between items-center">
                                <span className="text-slate-400">Nhà ở (Housing):</span>
                                <span className="text-amber-400 font-bold">{s.housing.toLocaleString()}</span>
                              </div>
                              <div className="flex justify-between items-center">
                                <span className="text-slate-400">Kho lương thực:</span>
                                <span className="text-purple-400 font-bold">{s.food_storage.toLocaleString()}</span>
                              </div>
                              <div className="flex justify-between items-center">
                                <span className="text-slate-400">Chỉ số an ninh:</span>
                                <span className="text-emerald-400 font-bold">{s.security.toFixed(0)}%</span>
                              </div>
                            </div>
                          </div>

                          <div className="pt-2 border-t border-slate-850">
                            {typeInfo.nextType ? (
                              <button
                                onClick={handleUpgrade}
                                disabled={actionLoading}
                                className="w-full py-2 bg-slate-900 hover:bg-cyan-950/60 border border-slate-700 hover:border-cyan-600 text-cyan-300 text-xs font-bold rounded-lg transition-all flex items-center justify-center gap-1.5"
                              >
                                <Sparkles className="w-3.5 h-3.5 text-cyan-400" />
                                Nâng cấp lên {typeInfo.nextLabel}
                              </button>
                            ) : (
                              <div className="w-full py-2 text-center text-[10px] font-bold text-amber-400 bg-amber-950/30 border border-amber-900/40 rounded-lg">
                                ⭐ Cấp độ thành trì tối đa
                              </div>
                            )}
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}

                {/* Build Settlement Modal */}
                {showBuildSettlementModal && (
                  <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4">
                    <div className="bg-slate-900 border border-slate-700 rounded-2xl max-w-md w-full p-6 space-y-5 shadow-2xl">
                      <div className="flex justify-between items-center border-b border-slate-800 pb-3">
                        <h3 className="text-base font-bold text-slate-100 flex items-center gap-2">
                          <MapIcon className="w-5 h-5 text-cyan-400" />
                          Xây Dựng Cơ Sở Định Cư Mới
                        </h3>
                        <button onClick={() => setShowBuildSettlementModal(false)} className="text-slate-400 hover:text-slate-200">
                          ✕
                        </button>
                      </div>

                      <div className="space-y-4 text-xs">
                        <div className="space-y-1.5">
                          <label className="text-slate-300 font-bold">Chọn Bộ Tộc Dị Nhân</label>
                          <select
                            value={buildRace}
                            onChange={(e) => setBuildRace(e.target.value)}
                            className="w-full bg-slate-950 border border-slate-700 rounded-xl p-2.5 text-slate-200 focus:outline-none focus:border-cyan-500"
                          >
                            <option value="HAIRY_MAN">🧔 Tộc Người Lông (Luyện Cổ)</option>
                            <option value="ROCKMAN">🗿 Tộc Người Đá (Khai Khoáng)</option>
                            <option value="FEATHERMAN">🦅 Tộc Người Lông Vũ (Vận Chuyển & Kiếp Nạn)</option>
                            <option value="SNOWMAN">❄️ Tộc Người Tuyết (Băng Tuyết)</option>
                            <option value="INKMAN">📜 Tộc Mặc Nhân (Nghiên Cứu)</option>
                            <option value="MINIMAN">🌱 Tộc Tiểu Nhân (Trồng Trọt)</option>
                            <option value="MERMAN">🧜 Tộc Giao Nhân (Thủy Vực)</option>
                            <option value="MUSHROOMMAN">🍄 Tộc Cô Nhân (Độc Thảo)</option>
                            <option value="DRAGONMAN">🐉 Tộc Long Nhân (Quân Sự)</option>
                          </select>
                        </div>

                        <div className="space-y-1.5">
                          <label className="text-slate-300 font-bold">Loại Định Cư Ban Đầu</label>
                          <select
                            value={buildType}
                            onChange={(e) => setBuildType(e.target.value)}
                            className="w-full bg-slate-950 border border-slate-700 rounded-xl p-2.5 text-slate-200 focus:outline-none focus:border-cyan-500"
                          >
                            <option value="VILLAGE">🏠 Bản Làng (Sức chứa: 10,000 người)</option>
                            <option value="TOWN">🏘️ Thị Trấn (Sức chứa: 50,000 người)</option>
                            <option value="CITY">🏰 Thành Thị (Sức chứa: 250,000 người)</option>
                            <option value="FORTRESS">🛡️ Pháo Đài (Sức chứa: 1,000,000 người)</option>
                          </select>
                        </div>

                        <div className="space-y-1.5">
                          <label className="text-slate-300 font-bold">Tên Thành Trì (Tùy chọn)</label>
                          <input
                            type="text"
                            placeholder="VD: Mao Gia Thôn, Thạch Sấm Động..."
                            value={buildName}
                            onChange={(e) => setBuildName(e.target.value)}
                            className="w-full bg-slate-950 border border-slate-700 rounded-xl p-2.5 text-slate-200 focus:outline-none focus:border-cyan-500"
                          />
                        </div>
                      </div>

                      <div className="flex gap-3 pt-2">
                        <button
                          onClick={() => setShowBuildSettlementModal(false)}
                          className="flex-1 py-2.5 bg-slate-800 hover:bg-slate-700 text-slate-300 font-bold rounded-xl text-xs transition-all"
                        >
                          Hủy Bỏ
                        </button>
                        <button
                          onClick={async () => {
                            const success = await buildSettlement(buildRace, buildName, buildType);
                            if (success) {
                              setShowBuildSettlementModal(false);
                              setBuildName("");
                            }
                          }}
                          disabled={actionLoading}
                          className="flex-1 py-2.5 bg-cyan-600 hover:bg-cyan-500 text-white font-bold rounded-xl text-xs transition-all"
                        >
                          Xác Nhận Xây Dựng
                        </button>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* SUB-TAB: NPCs */}
            {variantSubTab === "npcs" && (
              <div className="bg-slate-900/60 backdrop-blur-md border border-slate-700/40 rounded-2xl p-6 space-y-4">
                <h2 className="text-lg font-bold text-slate-100 flex items-center gap-2">
                  <Sparkles className="w-5 h-5 text-purple-400" />
                  Cổ Sư Dị Nhân
                </h2>
                <p className="text-xs text-slate-400">Quản lý các Cổ Sư được sinh ra từ dân số dị nhân. Phân công vai trò để tối ưu hiệu suất.</p>
                {variantNPCs.length === 0 ? (
                  <div className="text-center text-xs text-slate-500 py-8">Chưa có Cổ Sư nào. Dân số đủ lớn sẽ tự sinh ra Cổ Sư.</div>
                ) : (
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
                    {variantNPCs.map((npc) => {
                      const raceIcons: Record<string, string> = { HAIRY_MAN: "🧔", ROCKMAN: "🗿", FEATHERMAN: "🦅" };
                      const rankLabels: Record<number, string> = { 1: "Sơ Giai", 2: "Trung Giai", 3: "Hậu Giai", 4: "Đỉnh Phong", 5: "Siêu Việt" };
                      return (
                        <div key={npc.id} className="bg-slate-950/60 border border-slate-850 rounded-xl p-3 flex items-center gap-3">
                          <span className="text-2xl">{raceIcons[npc.race] || "👤"}</span>
                          <div className="flex-1 min-w-0">
                            <div className="font-bold text-xs text-slate-100 truncate">{npc.name}</div>
                            <div className="text-[9px] text-slate-500">Rank {npc.rank} · {rankLabels[npc.rank] || "?"} · {npc.path}</div>
                            <div className="text-[9px] text-slate-500">Loyal: {npc.loyalty.toFixed(0)} · Talent: {npc.talent.toFixed(0)}</div>
                          </div>
                          <span className={`text-[9px] font-bold px-1.5 py-0.5 rounded ${npc.status === "IDLE" ? "bg-slate-800 text-slate-400" : "bg-amber-500/20 text-amber-400"}`}>
                            {npc.role}
                          </span>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            )}

            {/* SUB-TAB: REFINEMENT QUEUE */}
            {variantSubTab === "refine" && (
              <div className="bg-slate-900/60 backdrop-blur-md border border-slate-700/40 rounded-2xl p-6 space-y-6">
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-slate-800 pb-4">
                  <div>
                    <h2 className="text-xl font-bold text-slate-100 flex items-center gap-2">
                      <Flame className="w-5 h-5 text-orange-400" />
                      Luyện Cổ Tự Động (Mao Dân)
                    </h2>
                    <p className="text-xs text-slate-400 mt-1">
                      Mao Dân sẽ tự động luyện Cổ theo hàng chờ. Hãy chuẩn bị nguyên liệu vào kho Tiên Khiếu để bắt đầu.
                    </p>
                  </div>
                  <button
                    onClick={() => setShowBuildRefineModal(true)}
                    disabled={actionLoading}
                    className="px-4 py-2 bg-gradient-to-r from-orange-600 to-amber-600 hover:from-orange-500 hover:to-amber-500 text-white text-xs font-bold rounded-xl shadow-lg transition-all flex items-center gap-2 shrink-0 self-start sm:self-auto"
                  >
                    <Plus className="w-4 h-4" />
                    Tạo Lệnh Mới
                  </button>
                </div>

                {refinementQueue.length === 0 ? (
                  <div className="text-center text-xs text-slate-500 py-12 bg-slate-950/40 rounded-xl border border-dashed border-slate-800 space-y-3">
                    <Flame className="w-10 h-10 mx-auto text-orange-500/30" />
                    <div>Chưa có lệnh luyện Cổ nào trong hàng đợi.</div>
                    <button
                      onClick={() => setShowBuildRefineModal(true)}
                      className="px-4 py-1.5 bg-orange-600/20 border border-orange-500/40 text-orange-300 font-bold rounded-lg text-xs hover:bg-orange-600/40 transition-all"
                    >
                      Tạo Lệnh Luyện Cổ Mới
                    </button>
                  </div>
                ) : (
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {refinementQueue.map((job) => {
                      const cleanGuId = job.recipe_id.replace(/^recipe_/, '').replace(/_\d+_\d+$/, '');
                      const guInfo = guDictionary?.[job.recipe_id] || guDictionary?.[cleanGuId];
                      const guName = guInfo?.name || cleanGuId || job.recipe_id;

                      const modeLabels: Record<string, { label: string; color: string }> = {
                        FIXED_COUNT: { label: "Số lượng cố định", color: "text-slate-300 bg-slate-800 border-slate-700" },
                        MAINTAIN_STOCK: { label: "Duy trì số lượng", color: "text-cyan-300 bg-cyan-950/80 border-cyan-800" },
                        UNTIL_SUCCESS: { label: "Luyện đến khi thành công", color: "text-emerald-300 bg-emerald-950/80 border-emerald-800" },
                        UNTIL_MATERIAL_EMPTY: { label: "Luyện hết nguyên liệu", color: "text-amber-300 bg-amber-950/80 border-amber-800" },
                      };

                      const modeInfo = modeLabels[job.mode] || { label: job.mode, color: "text-slate-300 bg-slate-800 border-slate-700" };
                      const progressPct = job.target_quantity > 0 ? Math.min(100, (job.current_quantity / job.target_quantity) * 100) : 0;

                      return (
                        <div key={job.id} className="bg-slate-950/80 border border-slate-800 rounded-xl p-4 space-y-3 flex flex-col justify-between">
                          <div className="space-y-2">
                            <div className="flex items-start justify-between gap-2">
                              <div className="flex items-center gap-2.5">
                                <div className="p-2 bg-orange-950/50 border border-orange-800/60 rounded-lg text-orange-400">
                                  <Flame className="w-5 h-5" />
                                </div>
                                <div>
                                  <div className="text-sm font-bold text-slate-100">{guName}</div>
                                  <div className="text-[10px] text-slate-400">Recipe: {job.recipe_id}</div>
                                </div>
                              </div>
                              <span className={`text-[9px] font-bold px-2 py-0.5 rounded border ${modeInfo.color}`}>
                                {modeInfo.label}
                              </span>
                            </div>

                            <div className="space-y-1.5 pt-1">
                              <div className="flex justify-between text-xs text-slate-400">
                                <span>Tiến độ luyện:</span>
                                <span className="font-bold text-amber-300">{job.current_quantity} / {job.target_quantity}</span>
                              </div>
                              <div className="w-full bg-slate-900 h-2 rounded-full overflow-hidden border border-slate-800">
                                <div className="h-full bg-gradient-to-r from-orange-500 to-amber-500 transition-all duration-500" style={{ width: `${progressPct}%` }} />
                              </div>
                            </div>
                          </div>

                          <div className="flex justify-between items-center pt-2 border-t border-slate-850">
                            <span className={`text-[10px] font-bold flex items-center gap-1 ${job.enabled ? "text-emerald-400" : "text-slate-500"}`}>
                              <span className={`w-1.5 h-1.5 rounded-full ${job.enabled ? "bg-emerald-400 animate-pulse" : "bg-slate-500"}`} />
                              {job.enabled ? "Đang luyện..." : "Tạm dừng"}
                            </span>
                            <button
                              onClick={async () => {
                                const confirmed = await showConfirm(`Xóa lệnh luyện Cổ ${guName}?`);
                                if (confirmed) {
                                  await deleteRefinementJob(job.id);
                                }
                              }}
                              disabled={actionLoading}
                              className="px-2.5 py-1 text-[10px] font-bold text-rose-400 hover:text-rose-300 bg-rose-950/40 border border-rose-900/60 hover:bg-rose-900/60 rounded-lg transition-all"
                            >
                              Xóa Lệnh
                            </button>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}

                {/* Create Refinement Job Modal */}
                {showBuildRefineModal && (
                  <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4">
                    <div className="bg-slate-900 border border-slate-700 rounded-2xl max-w-md w-full p-6 space-y-5 shadow-2xl">
                      <div className="flex justify-between items-center border-b border-slate-800 pb-3">
                        <h3 className="text-base font-bold text-slate-100 flex items-center gap-2">
                          <Flame className="w-5 h-5 text-orange-400" />
                          Tạo Lệnh Luyện Cổ Tự Động
                        </h3>
                        <button onClick={() => setShowBuildRefineModal(false)} className="text-slate-400 hover:text-slate-200">
                          ✕
                        </button>
                      </div>

                      <div className="space-y-4 text-xs">
                        {(() => {
                          const completedRecipes = (recipes || []).filter(
                            (r) => (r.completion_rate ?? 0) >= 100 || (r as { is_unlocked?: boolean }).is_unlocked
                          );

                          if (completedRecipes.length === 0) {
                            return (
                              <div className="p-3 bg-amber-950/40 border border-amber-800/60 rounded-xl text-amber-300 text-xs leading-relaxed space-y-1">
                                <div className="font-bold flex items-center gap-1.5 text-amber-200">
                                  <span>⚠️</span> Chưa có Bí Phương Hoàn Thiện
                                </div>
                                <div>
                                  Mao Dân chỉ có thể nhận luyện tự động các Cổ trùng mà đạo hữu đã <strong>hoàn thiện bí phương (đạt 100% tiến độ suy diễn)</strong>. Hãy vào mục Luyện Cổ để khai mở và hoàn thiện bí phương trước.
                                </div>
                              </div>
                            );
                          }

                          const currentSelectedId = completedRecipes.some((r) => r.id === refineRecipeId || r.target_gu_id === refineRecipeId)
                            ? (completedRecipes.find((r) => r.id === refineRecipeId || r.target_gu_id === refineRecipeId)?.id || refineRecipeId)
                            : (completedRecipes[0]?.id || completedRecipes[0]?.target_gu_id);

                          return (
                            <div className="space-y-1.5">
                              <label className="text-slate-300 font-bold">Chọn Bí Phương Hoàn Thiện</label>
                              <select
                                value={currentSelectedId}
                                onChange={(e) => setRefineRecipeId(e.target.value)}
                                className="w-full bg-slate-950 border border-slate-700 rounded-xl p-2.5 text-slate-200 focus:outline-none focus:border-orange-500"
                              >
                                {completedRecipes.map((r) => {
                                  const guName = guDictionary?.[r.target_gu_id]?.name || r.target_gu_id || r.id;
                                  const rank = r.target_rank || guDictionary?.[r.target_gu_id]?.rank || 1;
                                  return (
                                    <option key={r.id} value={r.id}>
                                      [{rank} Chuyển] Bí phương {guName} (Hoàn thiện 100%)
                                    </option>
                                  );
                                })}
                              </select>
                            </div>
                          );
                        })()}

                        <div className="space-y-1.5">
                          <label className="text-slate-300 font-bold">Chế Độ Luyện</label>
                          <select
                            value={refineMode}
                            onChange={(e) => setRefineMode(e.target.value)}
                            className="w-full bg-slate-950 border border-slate-700 rounded-xl p-2.5 text-slate-200 focus:outline-none focus:border-orange-500"
                          >
                            <option value="FIXED_COUNT">🔢 Số Lượng Cố Định (Luyện N lần)</option>
                            <option value="MAINTAIN_STOCK">📦 Duy Trì Số Lượng Trong Kho (Tự bổ sung khi thiếu)</option>
                            <option value="UNTIL_SUCCESS">🎯 Luyện Đến Khi Thành Công N Con</option>
                            <option value="UNTIL_MATERIAL_EMPTY">⚡ Luyện Hết Nguyên Liệu Trong Kho</option>
                          </select>
                        </div>

                        <div className="space-y-1.5">
                          <label className="text-slate-300 font-bold">Số Lượng Mục Tiêu</label>
                          <input
                            type="number"
                            min={1}
                            max={10000}
                            value={refineTargetQty}
                            onChange={(e) => setRefineTargetQty(Math.max(1, parseInt(e.target.value) || 1))}
                            className="w-full bg-slate-950 border border-slate-700 rounded-xl p-2.5 text-slate-200 focus:outline-none focus:border-orange-500"
                          />
                        </div>
                      </div>

                      <div className="flex gap-3 pt-2">
                        <button
                          onClick={() => setShowBuildRefineModal(false)}
                          className="flex-1 py-2.5 bg-slate-800 hover:bg-slate-700 text-slate-300 font-bold rounded-xl text-xs transition-all"
                        >
                          Hủy Bỏ
                        </button>
                        <button
                          onClick={async () => {
                            const completedRecipes = (recipes || []).filter(
                              (r) => (r.completion_rate ?? 0) >= 100 || (r as { is_unlocked?: boolean }).is_unlocked
                            );
                            const selectedRecipe = completedRecipes.find(r => r.id === refineRecipeId || r.target_gu_id === refineRecipeId) || completedRecipes[0];
                            const selectedId = selectedRecipe?.id || selectedRecipe?.target_gu_id || refineRecipeId;
                            if (!selectedId) {
                              addToast("Cần có bí phương hoàn thiện để tạo lệnh!", "error");
                              return;
                            }
                            const success = await createRefinementJob(selectedId, refineMode, refineTargetQty);
                            if (success) {
                              setShowBuildRefineModal(false);
                            }
                          }}
                          disabled={actionLoading}
                          className="flex-1 py-2.5 bg-orange-600 hover:bg-orange-500 text-white font-bold rounded-xl text-xs transition-all"
                        >
                          Xác Nhận Tạo Lệnh
                        </button>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* SUB-TAB: ECOSYSTEM */}
            {variantSubTab === "ecosystem" && (
              <div className="bg-slate-900/60 backdrop-blur-md border border-slate-700/40 rounded-2xl p-6 space-y-6">
                <div>
                  <h2 className="text-lg font-bold text-slate-100 flex items-center gap-2">
                    <Leaf className="w-5 h-5 text-emerald-400" />
                    Chuỗi Sinh Thái Tiên Khiếu
                  </h2>
                  <p className="text-xs text-slate-400 mt-1">Duy trì cân bằng thực vật → thú ăn cỏ → thú ăn thịt để tối ưu sản lượng.</p>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div className="bg-slate-950/60 p-4 rounded-xl border border-slate-850 text-center space-y-1">
                    <Leaf className="w-8 h-8 mx-auto text-emerald-500/40" />
                    <div className="text-xl font-black text-emerald-400">{plantCount}</div>
                    <div className="text-[10px] text-slate-400">Thực Vật</div>
                  </div>
                  <div className="bg-slate-950/60 p-4 rounded-xl border border-slate-850 text-center space-y-1">
                    <span className="text-2xl">🦌</span>
                    <div className="text-xl font-black text-amber-400">{herbCount}</div>
                    <div className="text-[10px] text-slate-400">Thú Ăn Cỏ</div>
                  </div>
                  <div className="bg-slate-950/60 p-4 rounded-xl border border-slate-850 text-center space-y-1">
                    <span className="text-2xl">🐺</span>
                    <div className="text-xl font-black text-rose-400">{carnCount}</div>
                    <div className="text-[10px] text-slate-400">Thú Ăn Thịt</div>
                  </div>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="bg-slate-950/50 p-4 rounded-xl border border-slate-800 space-y-2">
                    <h3 className="text-xs font-bold text-slate-200">Cung Cấp Lương Thực</h3>
                    <div className="space-y-1.5 text-[10px]">
                      <div className="flex justify-between"><span className="text-slate-400">Sản lượng:</span><span className="text-emerald-400 font-bold">{totalFoodProd} / năm</span></div>
                      <div className="flex justify-between"><span className="text-slate-400">Tiêu thụ ({totalVariantHumanPop} người):</span><span className="text-rose-400 font-bold">{foodNeeded.toFixed(0)} / năm</span></div>
                      <div className="flex justify-between pt-1 border-t border-slate-800">
                        <span className="text-slate-400">Cân đối:</span>
                        {foodSurplus ? (
                          <span className="text-emerald-400 font-bold">Thặng Dư ✓</span>
                        ) : (
                          <span className="text-rose-400 font-bold animate-pulse">Thiếu Thụt ✗</span>
                        )}
                      </div>
                    </div>
                  </div>
                  <div className="bg-slate-950/50 p-4 rounded-xl border border-slate-800 space-y-2">
                    <h3 className="text-xs font-bold text-slate-200 flex items-center gap-1"><AlertTriangle className="w-3.5 h-3.5 text-amber-500" /> Xung Đột Đạo Ngân</h3>
                    {(() => {
                      const conflictsList: { n1: ApertureResourceNode; n2: ApertureResourceNode }[] = [];
                      const checkedPairs = new Set<string>();
                      apertureNodes.forEach((n1) => {
                        if (n1.is_destroyed) return;
                        const conflictingOther = checkConflict(n1, apertureNodes);
                        if (conflictingOther) {
                          const pairKey = n1.id < conflictingOther.id ? `${n1.id}:${conflictingOther.id}` : `${conflictingOther.id}:${n1.id}`;
                          if (!checkedPairs.has(pairKey)) { checkedPairs.add(pairKey); conflictsList.push({ n1, n2: conflictingOther }); }
                        }
                      });
                      if (conflictsList.length === 0) return <div className="text-[10px] text-emerald-400 text-center py-2">Không có xung đột</div>;
                      return (
                        <div className="space-y-1 max-h-24 overflow-y-auto">
                          {conflictsList.map(({ n1, n2 }, idx) => (
                            <div key={idx} className="text-[9px] text-rose-300 flex justify-between">
                              <span>{n1.name} ❌ {n2.name}</span>
                              <span className="text-rose-500">Xung Đột</span>
                            </div>
                          ))}
                        </div>
                      );
                    })()}
                  </div>
                </div>
              </div>
            )}
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
