import React, { useState, useEffect, useMemo } from "react";
import { useGameStore, Sect, SectMission, GuInstance, SectMine, formatRecipeName, guSpecFor, SectRelocationEstimate } from "@/store/useGameStore";
import { translatePath, parsePaths } from "@/lib/translate";
import {
  Shield,
  Activity,
  Award,
  BookOpen,
  Plus,
  Coins,
  Hourglass,
  AlertTriangle,
  Sparkles,
  TrendingUp,
  Package,
  ArrowRight,
  LogOut,
  CheckCircle,
  HelpCircle,
  Lock,
  RefreshCw,
  Gift,
  Flame,
  Search,
  X,
  Check,
  Trash2,
  Ban,
  MapPin,
  Navigation,
  Crown,
} from "lucide-react";
import { GamePanel, GameButton, GameCard, GameTabs, GameModal, GameInput, GameSelect, GameBadge, GameProgressBar, GameSection } from "@/components/ui";
// UI migrated to shared components

const rankSalaryBonus: Record<string, number> = {
  NGOAI_MON: 0,
  NOI_MON: 50,
  TINH_ANH: 150,
  CHAN_TRUYEN: 400,
  TRUONG_LAO: 1000,
  HO_PHAP: 2000,
  CHUONG_MON: 5000,
};

interface StockSelection {
  item_type: "gu" | "material" | "recipe";
  item_id: string;
  name: string;
  unit_cost: number; // quỹ tông môn / đơn vị
  quantity: number;
}

export const SectView: React.FC = () => {
  const {
    player,
    sects,
    regions,
    mySectStatus,
    sectMissions,
    guDictionary,
    itemDictionary,
    monsterDictionary,
    actionLoading,
    fetchSects,
    joinSect,
    cancelSectJoinRequest,
    leaveSect,
    fetchMySectStatus,
    promoteSectRank,
    donateSectStones,
    donateSectMaterials,
    donateSectGu,
    donateSectBulk,
    donateSectRecipe,
    buyFromSectVault,
    fetchSectMissions,
    acceptSectMission,
    completeSectMission,
    deleteSectMission,
    abandonSectMission,
    sectMembers,
    createSect,
    fetchSectMembers,
    setMemberRank,
    kickMember,
    createCustomMission,
    stockSectVaultBulk,
    setView,
    sectJoinRequests,
    sectPromotionRequests,
    toggleSectApproval,
  toggleSectPromotionApproval,
  fetchJoinRequests,
  actionJoinRequest,
  fetchPromotionRequests,
  actionPromotionRequest,
  fetchPlayerProfile,
  sectMap,
  fetchSectsMap,
  attackSect,
  defendSect,
  undefendSect,
  toggleDefendSect,
  upgradeSectBuilding,
  addToast,
  showConfirm,
  recipes,
  paySectSalary,
  getRelocationEstimate,
  relocateSect,
  claimSectLeadership,
  } = useGameStore();

  const [activeTab, setActiveTab] = useState<"lobby" | "missions" | "vault" | "donate" | "territory" | "manage">("lobby");
  const [selectedSectId, setSelectedSectId] = useState<string | null>(null);

  // Search & Pagination States
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedRegion, setSelectedRegion] = useState("all");
  const [currentPage, setCurrentPage] = useState(1);
  const itemsPerPage = 6;

  // Sect Creation Form
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [createSectName, setCreateSectName] = useState("");
  const [createSectDesc, setCreateSectDesc] = useState("");
  const [createSectRegion, setCreateSectRegion] = useState("Central_Continent");

  // Vault state
  const [vaultTab, setVaultTab] = useState<"gu" | "material" | "recipe">("gu");
  const [vaultSearch, setVaultSearch] = useState("");
  const [vaultRankFilter, setVaultRankFilter] = useState<string>("All");
  const [buyQtys, setBuyQtys] = useState<Record<string, number>>({});

  const getBuyQty = (key: string): number => buyQtys[key] || 1;
  const setBuyQty = (key: string, n: number, max?: number): void => {
    const upper = max !== undefined ? Math.max(1, max) : Number.MAX_SAFE_INTEGER;
    const v = Math.max(1, Math.min(upper, Math.floor(n) || 1));
    setBuyQtys((prev) => ({ ...prev, [key]: v }));
  };

  const renderQtyStepper = (key: string, maxInVault: number, maxAffordable?: number) => {
    const curVal = getBuyQty(key);
    const effectiveAffordable = maxAffordable !== undefined ? maxAffordable : maxInVault;
    const smartMax = effectiveAffordable > 0 ? Math.min(maxInVault, effectiveAffordable) : maxInVault;

    return (
      <div className="flex items-center gap-1">
        <button
          type="button"
          onClick={() => setBuyQty(key, curVal - 1, maxInVault)}
          disabled={curVal <= 1}
          className="w-7 h-7 rounded-lg bg-zinc-800 hover:bg-zinc-700 disabled:opacity-40 disabled:cursor-not-allowed text-zinc-300 flex items-center justify-center cursor-pointer transition-colors text-sm font-bold"
        >
          −
        </button>
        <input
          type="number"
          min={1}
          max={maxInVault}
          value={curVal || ""}
          placeholder="1"
          onChange={(e) => {
            const raw = e.target.value;
            if (raw === "") {
              setBuyQtys((prev) => ({ ...prev, [key]: 0 }));
              return;
            }
            const n = parseInt(raw);
            if (!isNaN(n)) {
              setBuyQty(key, n, maxInVault);
            }
          }}
          onBlur={() => {
            if (getBuyQty(key) < 1) {
              setBuyQty(key, 1, maxInVault);
            }
          }}
          className="w-16 h-7 bg-zinc-950 border border-zinc-800 rounded-lg text-center text-xs font-mono font-bold text-amber-400 outline-none focus:border-amber-600"
        />
        <button
          type="button"
          onClick={() => setBuyQty(key, curVal + 1, maxInVault)}
          disabled={curVal >= maxInVault}
          className="w-7 h-7 rounded-lg bg-zinc-800 hover:bg-zinc-700 disabled:opacity-40 disabled:cursor-not-allowed text-zinc-300 flex items-center justify-center cursor-pointer transition-colors text-sm font-bold"
        >
          +
        </button>
        {maxInVault > 1 && (
          <div className="flex items-center gap-1">
            {maxInVault >= 10 && (
              <button
                type="button"
                onClick={() => setBuyQty(key, curVal + 10, maxInVault)}
                disabled={curVal + 10 > maxInVault}
                className="px-1.5 h-7 rounded-lg bg-zinc-800/80 hover:bg-zinc-700 text-[10px] font-mono font-semibold text-zinc-400 hover:text-zinc-200 disabled:opacity-30 disabled:cursor-not-allowed transition-colors cursor-pointer"
                title="Cộng thêm 10"
              >
                +10
              </button>
            )}
            <button
              type="button"
              onClick={() => setBuyQty(key, smartMax, maxInVault)}
              className="px-1.5 h-7 rounded-lg bg-amber-950/40 hover:bg-amber-900/60 border border-amber-800/40 text-[10px] font-mono font-bold text-amber-400 hover:text-amber-300 transition-colors cursor-pointer"
              title={effectiveAffordable < maxInVault ? `Mua tối đa khả năng (${smartMax})` : `Tối đa trong kho (${maxInVault})`}
            >
              Max
            </button>
          </div>
        )}
      </div>
    );
  };

  // Donation state
  const [donateType, setDonateType] = useState<"stones" | "materials" | "gu" | "recipes">("stones");
  const [donateStones, setDonateStones] = useState<number>(100);
  const [selectedMatId, setSelectedMatId] = useState<string>("");
  const [donateMatAmount, setDonateMatAmount] = useState<number>(1);
  const [selectedGuInstId, setSelectedGuInstId] = useState<string>("");
  const [donateSearch, setDonateSearch] = useState("");
  const [donateRankFilter, setDonateRankFilter] = useState<string>("All");
  const [donatePathFilter, setDonatePathFilter] = useState<string>("All");

  const [selectedDonateMaterials, setSelectedDonateMaterials] = useState<Record<string, {
    material_id: string;
    name: string;
    owned: number;
    valuePerItem: number;
    amountToDonate: number;
  }>>({});
  const [selectedDonateGus, setSelectedDonateGus] = useState<Record<string, {
    key: string;
    gu_id: string;
    rank: number;
    name: string;
    valuePerItem: number;
    ownedCount: number;
    amountToDonate: number;
    instances: GuInstance[];
  }>>({});
  const [selectedDonateRecipes, setSelectedDonateRecipes] = useState<Record<string, {
    recipe_id: string;
    name: string;
    completion_rate: number;
    value: number;
  }>>({});

  useEffect(() => {
    setDonateSearch("");
    setDonateRankFilter("All");
    setDonatePathFilter("All");
    setSelectedDonateMaterials({});
    setSelectedDonateGus({});
    setSelectedDonateRecipes({});
  }, [donateType, activeTab]);

  // Sect management sub-tab
  const [mgmtTab, setMgmtTab] = useState<"members" | "missions" | "stock" | "requests" | "salary" | "relocate">("members");
  const [relocationEstimate, setRelocationEstimate] = useState<SectRelocationEstimate | null>(null);
  const [loadingEstimate, setLoadingEstimate] = useState(false);

  // Custom mission creation form state
  const [mTitle, setMTitle] = useState("");
  const [mDesc, setMDesc] = useState("");
  const [mMissionType, setMMissionType] = useState("GATHER");
  const [mTargetId, setMTargetId] = useState("");
  const [mTargetSearch, setMTargetSearch] = useState("");
  const [mTargetAmount, setMTargetAmount] = useState(1);
  const [mRewardContribution, setMRewardContribution] = useState(100);
  const [mRequiredRank, setMRequiredRank] = useState("NGOAI_MON");
  const [mDuration, setMDuration] = useState(24); // giờ; < 0 => không hết hạn

  // Re-render countdowns for mission expiry periodically
  const [nowTs, setNowTs] = useState(() => Date.now());
  useEffect(() => {
    const id = setInterval(() => setNowTs(Date.now()), 30000);
    return () => clearInterval(id);
  }, []);

  // Territory defend modal state
  const [showDefendModal, setShowDefendModal] = useState(false);
  const [defendType, setDefendType] = useState<"member" | "beast">("member");
  const [defendPlayerId, setDefendPlayerId] = useState("");
  const [defendMonsterId, setDefendMonsterId] = useState("");
  const [defendMonsterRank, setDefendMonsterRank] = useState(1);

  // Territory attack modal state
  const [attackTargetSect, setAttackTargetSect] = useState<ApiResponseData | null>(null);
  const [attackPlayerId, setAttackPlayerId] = useState("");

  // Vault stocking form state (donation-style multi-select)
  const [stockItemType, setStockItemType] = useState<"gu" | "material" | "recipe">("material");
  const [stockSelections, setStockSelections] = useState<Record<string, StockSelection>>({});
  const [stockSearch, setStockSearch] = useState("");
  const [stockRankFilter, setStockRankFilter] = useState("All");
  const [stockPathFilter, setStockPathFilter] = useState("All");

  useEffect(() => {
    fetchSects();
    fetchMySectStatus();
  }, [fetchSects, fetchMySectStatus]);

  useEffect(() => {
    if (mySectStatus?.in_sect) {
      fetchSectMissions();
    }
  }, [mySectStatus?.in_sect, fetchSectMissions]);

  useEffect(() => {
    if (activeTab === "territory") {
      fetchSectsMap();
    }
  }, [activeTab, fetchSectsMap]);

  useEffect(() => {
    if (mySectStatus?.in_sect) {
      fetchSectMembers();
    }
  }, [mySectStatus?.in_sect, fetchSectMembers]);

  useEffect(() => {
    if (activeTab === "manage") {
      fetchSectMembers();
      const isLeaderOrElder = mySectStatus?.member?.member_rank === "CHUONG_MON" || mySectStatus?.member?.member_rank === "TRUONG_LAO" || mySectStatus?.member?.member_rank === "HO_PHAP";
      if (isLeaderOrElder) {
        fetchJoinRequests();
        fetchPromotionRequests();
      }
    }
  }, [activeTab, fetchSectMembers, fetchJoinRequests, fetchPromotionRequests, mySectStatus?.member?.member_rank]);

  useEffect(() => {
    if (activeTab === "manage" && mgmtTab === "relocate") {
      setLoadingEstimate(true);
      getRelocationEstimate().then((res) => {
        setRelocationEstimate(res);
        setLoadingEstimate(false);
      });
    }
  }, [activeTab, mgmtTab, getRelocationEstimate]);

  useEffect(() => {
    setCurrentPage(1);
  }, [searchTerm, selectedRegion]);

  const handleCreateSect = async () => {
    if (!createSectName.trim() || !createSectDesc.trim()) return;
    const ok = await createSect(createSectName.trim(), createSectDesc.trim(), createSectRegion);
    if (ok) {
      setShowCreateForm(false);
      setCreateSectName("");
      setCreateSectDesc("");
      setActiveTab("lobby");
    }
  };

  const handleCreateCustomMission = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!mTitle.trim() || !mDesc.trim() || !mTargetId.trim()) return;
    const ok = await createCustomMission({
      title: mTitle.trim(),
      description: mDesc.trim(),
      mission_type: mMissionType,
      target_id: mTargetId.trim(),
      target_amount: mTargetAmount,
      reward_contribution: mRewardContribution,
      required_member_rank: mRequiredRank,
      duration_hours: mDuration,
    });
    if (ok) {
      setMTitle("");
      setMDesc("");
      setMTargetId("");
      setMTargetAmount(1);
      setMRewardContribution(100);
      setMDuration(24);
    }
  };

  if (!player) {
    return (
      <div className="flex flex-col items-center justify-center p-8 bg-zinc-950/80 backdrop-blur-md rounded-2xl border border-zinc-900 max-w-lg mx-auto text-center">
        <div className="w-12 h-12 border-4 border-emerald-500 border-t-transparent rounded-full animate-spin mb-4" />
        <p className="text-zinc-400">Đang tải thông tin Cổ sư...</p>
      </div>
    );
  }

  // Vietnamese translator for Sect Member ranks
  const getRankName = (rank: string): string => {
    switch (rank) {
      case "NGOAI_MON": return "Đệ Tử Ngoại Môn";
      case "NOI_MON": return "Đệ Tử Nội Môn";
      case "TINH_ANH": return "Đệ Tử Tinh Anh";
      case "CHAN_TRUYEN": return "Đệ Tử Chân Truyền";
      case "TRUONG_LAO": return "Trưởng Lão";
      case "HO_PHAP": return "Hộ Pháp";
      case "CHUONG_MON": return "Chưởng Môn";
      default: return rank;
    }
  };

  const getRankColor = (rank: string): string => {
    switch (rank) {
      case "NGOAI_MON": return "text-zinc-400 bg-zinc-900/50 border-zinc-800";
      case "NOI_MON": return "text-emerald-400 bg-emerald-950/20 border-emerald-500/20";
      case "TINH_ANH": return "text-cyan-400 bg-cyan-950/20 border-cyan-500/20";
      case "CHAN_TRUYEN": return "text-purple-400 bg-purple-950/20 border-purple-500/20";
      case "TRUONG_LAO": return "text-amber-400 bg-amber-950/20 border-amber-500/20";
      case "HO_PHAP": return "text-orange-400 bg-orange-950/25 border-orange-500/30";
      case "CHUONG_MON": return "text-rose-400 bg-rose-950/25 border-rose-500/30 font-bold";
      default: return "text-zinc-400";
    }
  };

  const rankValue = (rank?: string): number => {
    switch (rank) {
      case "NGOAI_MON": return 1;
      case "NOI_MON": return 2;
      case "TINH_ANH": return 3;
      case "CHAN_TRUYEN": return 4;
      case "TRUONG_LAO": return 5;
      case "HO_PHAP": return 6;
      case "CHUONG_MON": return 7;
      default: return 0;
    }
  };

  const getMissionTypeName = (type: string): string => {
    switch (type) {
      case "GATHER": return "Thu thập Linh tài";
      case "COMBAT": return "Chinh phạt quái";
      case "REFINE": return "Hỗ trợ Luyện Cổ";
      case "DONATE": return "Cống hiến Cổ trùng";
      default: return type;
    }
  };

  const formatMissionTime = (expiresAt?: string | null): string | null => {
    if (!expiresAt) return null;
    const target = new Date(expiresAt).getTime();
    if (Number.isNaN(target)) return null;
    const diff = target - nowTs;
    if (diff <= 0) return "Đã hết hạn";
    const h = Math.floor(diff / 3600000);
    const m = Math.floor((diff % 3600000) / 60000);
    const s = Math.floor((diff % 60000) / 1000);
    return `${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
  };

  const getRegionName = (region: string): string => {
    switch (region) {
      case "hub_trung_chau": return "Trung Châu";
      case "hub_nam_cuong": return "Nam Cương";
      case "hub_bac_nguyen": return "Bắc Nguyên";
      case "hub_tay_mac": return "Tây Mạc";
      case "hub_dong_hai": return "Đông Hải";
      default: return region;
    }
  };

  // Unit cost (quỹ tông môn) for stocking an item — mirrors backend pkg/pricing
  const getStockUnitCost = (type: "gu" | "material" | "recipe", id: string): number => {
    const guRankBasePrice = [0, 200, 500, 1200, 2500, 5000, 200, 500, 1200, 2500];
    if (type === "gu") {
      const spec = guDictionary[id];
      const rank = spec?.rank || 1;
      const basePrice = (spec?.base_price && spec.base_price > 0) ? spec.base_price : (guRankBasePrice[rank] || guRankBasePrice[1]);
      return basePrice;
    }
    if (type === "recipe") {
      const info = recipes.find((r) => r.id === id);
      const targetGuSpec = info ? guDictionary[info.target_gu_id] : null;
      const rank = targetGuSpec?.rank || 1;
      return getRecipeCost(rank);
    }
    const spec = itemDictionary[id];
    const rank = spec?.rank || 1;
    return rank * 50; // backend: 50 quỹ / hạng
  };

  // Promotion requirements helper
  const getPromotionRequirements = (currentRank: string) => {
    switch (currentRank) {
      case "NGOAI_MON":
        return { next: "NOI_MON", reqRank: 2, points: 100 };
      case "NOI_MON":
        return { next: "TINH_ANH", reqRank: 3, points: 500 };
      case "TINH_ANH":
        return { next: "CHAN_TRUYEN", reqRank: 4, points: 2000 };
      case "CHAN_TRUYEN":
        return { next: "TRUONG_LAO", reqRank: 5, points: 10000 };
      case "TRUONG_LAO":
        return { next: "HO_PHAP", reqRank: 6, points: 30000 };
      default:
        return null;
    }
  };

  const promotionReq = mySectStatus?.member ? getPromotionRequirements(mySectStatus.member.member_rank) : null;
  const canPromote = promotionReq && player.rank >= promotionReq.reqRank && mySectStatus!.member!.contribution_points >= promotionReq.points;

  // Donation materials listing
  const playerMaterials = useMemo(() => {
    return Object.entries(player.materials_inventory || {}).filter(([_, qty]) => qty > 0);
  }, [player.materials_inventory]);

  // Donation Gu listing (Filter out Vital Gu and equipped Gu)
  const playerDonatableGus = useMemo(() => {
    return (player.gu_inventory || []).filter(g => !g.is_vital && !g.equipped && g.status !== "DEAD");
  }, [player.gu_inventory]);

  // Pricing helpers matching backend/pkg/pricing
  const guRankBasePrice = [0, 200, 500, 1200, 2500, 5000, 200, 500, 1200, 2500];
  const itemRankBasePrice = [0, 1, 3, 6, 13, 25, 10, 25, 60, 150];

  const getItemBasePrice = (matId: string): number => {
    const spec = itemDictionary[matId];
    if (!spec) return 5;
    if (spec.base_price && spec.base_price > 0) return spec.base_price;
    return itemRankBasePrice[spec.rank || 1] || 5;
  };

  const getGuBasePrice = (guId: string): number => {
    const spec = guDictionary[guId];
    if (!spec) return 200;
    if (spec.base_price && spec.base_price > 0) return spec.base_price;
    return guRankBasePrice[spec.rank || 1] || 200;
  };

  // DonatePoints = basePrice * 0.50 (DonateRatio), minimum 1
  const getDonatePoints = (basePrice: number): number => Math.max(1, Math.floor(basePrice * 0.50));

  // VaultCost = DonatePoints * 1.50 (VaultMarkup), minimum 1
  const getVaultCost = (basePrice: number): number => Math.max(1, Math.floor(getDonatePoints(basePrice) * 1.50));

  // RecipeCostByGuRank = GuBasePriceByRank(rank) / 10, minimum 20
  const getRecipeCost = (targetGuRank: number): number => {
    const p = Math.floor(guRankBasePrice[targetGuRank] / 10);
    return Math.max(20, p);
  };

  // Get material contribution points value (DonateRatio = 50%)
  const getMaterialValue = (matId: string): number => getDonatePoints(getItemBasePrice(matId));

  // Get Gu contribution points value (DonateRatio = 50%)
  const getGuValue = (guId: string): number => getDonatePoints(getGuBasePrice(guId));

  // Filter sects
  const filteredSects = (sects || []).filter((sect) => {
    const matchesSearch =
      sect.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      sect.description.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesRegion = selectedRegion === "all" || sect.region === selectedRegion;
    return matchesSearch && matchesRegion;
  });

  // Pagination calculation
  const totalPages = Math.ceil(filteredSects.length / itemsPerPage);
  const startIndex = (currentPage - 1) * itemsPerPage;
  const paginatedSects = filteredSects.slice(startIndex, startIndex + itemsPerPage);

  // 1. STATE: NOT IN SECT
  if (!mySectStatus?.in_sect) {
    return (
      <div className="w-full mx-auto px-2 py-4 flex flex-col gap-6 animate-fade-in">
        {/* Welcome Header */}
        <div className="text-center py-6 bg-zinc-950/60 border border-zinc-900 rounded-3xl p-6 shadow-xl backdrop-blur-md relative overflow-hidden">
          <div className="absolute top-0 left-1/2 -translate-x-1/2 w-64 h-64 bg-blue-500/10 rounded-full blur-3xl -z-10" />
          <Award className="w-16 h-16 text-blue-400 mx-auto mb-4 animate-pulse" />
          <h2 className="text-2xl sm:text-3xl font-serif font-bold text-zinc-100 bg-gradient-to-r from-blue-200 to-indigo-400 bg-clip-text text-transparent">
            Gia Nhập Tông Môn
          </h2>
          <p className="text-sm text-zinc-400 max-w-xl mx-auto mt-2 leading-relaxed">
            Nguyên Thủy Tiên Tôn sáng lập Tông môn chế độ, phá vỡ xiềng xích gia tộc huyết thống.
            Nơi đây lấy cống hiến đo tài nguyên, bái sư học nghệ, mượn cổ trùng quý hiếm, cùng tranh phong thiên hạ!
          </p>
          {mySectStatus?.has_pending_join && (
            <div className="mt-4 p-3 bg-amber-950/40 border border-amber-500/25 rounded-2xl text-xs text-amber-400 font-mono max-w-md mx-auto flex flex-col items-center gap-2">
              <span>⏳ Yêu cầu gia nhập tông môn <strong className="text-amber-200 font-bold">{mySectStatus.pending_sect_name}</strong> đang chờ duyệt!</span>
              <button
                onClick={async () => {
                  if (await showConfirm("Đạo hữu có chắc chắn muốn hủy đơn xin gia nhập tông môn này?")) {
                    await cancelSectJoinRequest();
                  }
                }}
                disabled={actionLoading}
                className="px-3 py-1.5 bg-zinc-900 hover:bg-red-950/40 border border-zinc-800 hover:border-red-500/40 text-zinc-300 hover:text-red-300 rounded-lg text-xs font-semibold transition-colors cursor-pointer"
              >
                Hủy Đơn Xin
              </button>
            </div>
          )}
        </div>

        {/* Search & Filter Controls */}
        <div className="flex flex-col sm:flex-row gap-4 bg-zinc-950/40 p-4 border border-zinc-900 rounded-2xl backdrop-blur-md">
          <div className="flex-1 relative">
            <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-500" />
            <input
              type="text"
              placeholder="Tìm kiếm tông môn theo tên hoặc mô tả..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full bg-zinc-950 border border-zinc-900 rounded-xl pl-10 pr-4 py-2 text-xs text-zinc-200 focus:outline-none focus:border-blue-500/50"
            />
          </div>
          <div className="w-full sm:w-48">
            <select
              value={selectedRegion}
              onChange={(e) => setSelectedRegion(e.target.value)}
              className="w-full bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-blue-500/50 cursor-pointer"
            >
              <option value="all">Tất cả Vực Giới</option>
              <option value="hub_trung_chau">Trung Châu</option>
              <option value="hub_nam_cuong">Nam Cương</option>
              <option value="hub_bac_nguyen">Bắc Nguyên</option>
              <option value="hub_tay_mac">Tây Mạc</option>
              <option value="hub_dong_hai">Đông Hải</option>
            </select>
          </div>
        </div>

        {/* Sects Directory */}
        {paginatedSects.length > 0 ? (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {paginatedSects.map((sect) => (
              <div
                key={sect.id}
                onClick={() => setSelectedSectId(sect.id)}
                className={`group bg-zinc-950/40 hover:bg-zinc-950/80 border ${selectedSectId === sect.id ? "border-blue-500/50 shadow-[0_0_20px_rgba(59,130,246,0.15)]" : "border-zinc-900 hover:border-zinc-800"
                  } rounded-3xl p-5 flex flex-col justify-between transition-all duration-300 cursor-pointer backdrop-blur-md`}
              >
                <div>
                  <div className="flex justify-between items-start mb-4">
                    <div className="px-3 py-1 text-[10px] font-mono font-semibold uppercase tracking-wider rounded-lg bg-zinc-900 text-zinc-400 border border-zinc-800">
                      {getRegionName(sect.region)}
                    </div>
                    {sect.require_approval ? (
                      <span className="text-[9px] font-mono bg-amber-950/40 text-amber-400 border border-amber-500/20 rounded-md px-1.5 py-0.5">
                        Cần Duyệt
                      </span>
                    ) : (
                      <span className="text-[9px] font-mono bg-emerald-950/40 text-emerald-400 border border-emerald-500/20 rounded-md px-1.5 py-0.5">
                        Tự Do
                      </span>
                    )}
                  </div>

                  <h3 className="text-lg font-serif font-bold text-zinc-100 mb-2 group-hover:text-blue-400 transition-colors">
                    {sect.name}
                  </h3>
                  <p className="text-xs text-zinc-400 leading-relaxed line-clamp-3 mb-4">
                    {sect.description}
                  </p>
                </div>

                <div className="pt-4 border-t border-zinc-900/60 flex items-center justify-between mt-auto">
                  <span className="text-[10px] text-zinc-500 font-mono">Bảo Khố: Cực Phẩm</span>
                  <span className="text-xs text-blue-400 flex items-center gap-1 font-semibold group-hover:translate-x-1 transition-transform">
                    Chi tiết <ArrowRight className="w-3.5 h-3.5" />
                  </span>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="py-12 bg-zinc-950/30 border border-zinc-900 rounded-3xl text-center text-xs text-zinc-500 font-mono">
            Không tìm thấy tông môn nào phù hợp điều kiện tìm kiếm.
          </div>
        )}

        {/* Pagination Controls */}
        {totalPages > 1 && (
          <div className="flex justify-center items-center gap-2 mt-2">
            <button
              onClick={() => setCurrentPage(prev => Math.max(1, prev - 1))}
              disabled={currentPage === 1}
              className="px-3 py-1.5 bg-zinc-900 border border-zinc-800 disabled:opacity-30 disabled:hover:bg-zinc-900 hover:bg-zinc-800 text-zinc-300 rounded-xl text-xs font-semibold transition-colors cursor-pointer"
            >
              Trang trước
            </button>

            {Array.from({ length: totalPages }).map((_, i) => (
              <button
                key={i}
                onClick={() => setCurrentPage(i + 1)}
                className={`w-8 h-8 rounded-xl text-xs font-mono font-bold border transition-colors cursor-pointer ${currentPage === i + 1
                    ? "bg-blue-600 border-blue-500 text-white"
                    : "bg-zinc-900 border-zinc-800 text-zinc-400 hover:bg-zinc-800 hover:text-zinc-200"
                  }`}
              >
                {i + 1}
              </button>
            ))}

            <button
              onClick={() => setCurrentPage(prev => Math.min(totalPages, prev + 1))}
              disabled={currentPage === totalPages}
              className="px-3 py-1.5 bg-zinc-900 border border-zinc-800 disabled:opacity-30 disabled:hover:bg-zinc-900 hover:bg-zinc-800 text-zinc-300 rounded-xl text-xs font-semibold transition-colors cursor-pointer"
            >
              Trang sau
            </button>
          </div>
        )}

        {/* Selected Sect Detail Popup Modal */}
        {selectedSectId && (() => {
          const sect = sects.find(s => s.id === selectedSectId);
          if (!sect) return null;
          return (
            <div
              onClick={() => setSelectedSectId(null)}
              className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm transition-opacity animate-fade-in cursor-pointer"
            >
              <div
                onClick={(e) => e.stopPropagation()}
                className="w-full max-w-2xl bg-zinc-950 border border-zinc-900 rounded-3xl p-6 shadow-[0_0_30px_rgba(59,130,246,0.2)] flex flex-col gap-6 animate-scale-up cursor-default relative overflow-hidden max-h-[90vh] overflow-y-auto"
              >
                <div className="absolute top-0 right-0 w-64 h-64 bg-blue-500/5 rounded-full blur-3xl -z-10" />

                {/* Modal Header */}
                <div className="flex justify-between items-start border-b border-zinc-900 pb-4">
                  <div>
                    <h3 className="text-xl font-serif font-bold text-zinc-100 flex items-center gap-2">
                      <Award className="w-6 h-6 text-blue-400" />
                      {sect.name}
                    </h3>
                    <p className="text-xs text-zinc-500 font-mono mt-1">
                      Vực giới quản hạt: {getRegionName(sect.region)}
                    </p>
                  </div>
                  <button
                    onClick={() => setSelectedSectId(null)}
                    className="p-1.5 rounded-lg bg-zinc-900/60 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-100 transition-colors border border-zinc-900/60 cursor-pointer"
                    title="Đóng"
                  >
                    <X className="w-4 h-4" />
                  </button>
                </div>

                {/* Modal Body */}
                <div className="space-y-4">
                  <div>
                    <h4 className="text-xs font-semibold text-zinc-400 uppercase tracking-widest mb-2 font-serif">Truyền thuyết tông môn</h4>
                    <p className="text-sm text-zinc-300 leading-relaxed font-mono whitespace-pre-wrap">{sect.description}</p>
                  </div>

                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-2">
                    <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-4">
                      <h5 className="text-xs font-semibold text-zinc-400 mb-2 font-serif">Đặc trưng bảo khố</h5>
                      <div className="flex flex-wrap gap-2 max-h-32 overflow-y-auto">
                        {sect.vault_gu && (sect.vault_gu as ApiResponseData[]).filter((g: ApiResponseData) => g.quantity === undefined || g.quantity > 0).map((g: ApiResponseData) => {
                          const spec = guDictionary[g.id];
                          return (
                            <span key={g.id} className="text-xs bg-zinc-950 px-2.5 py-1 rounded-lg border border-zinc-800 text-zinc-300">
                              {spec?.name || g.id}
                            </span>
                          );
                        })}
                        {sect.vault_materials && Object.entries(sect.vault_materials).filter(([_, qty]: ApiResponseData) => qty > 0).map(([matId]) => {
                          const spec = itemDictionary[matId];
                          return (
                            <span key={matId} className="text-xs bg-zinc-950 px-2.5 py-1 rounded-lg border border-zinc-800 text-zinc-300">
                              {spec?.name || matId}
                            </span>
                          );
                        })}
                        {(!sect.vault_gu || (sect.vault_gu as ApiResponseData[]).length === 0) && (!sect.vault_materials || Object.keys(sect.vault_materials).length === 0) && (
                          <span className="text-xs text-zinc-500 font-mono">Bảo khố trống</span>
                        )}
                      </div>
                    </div>

                    <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-4">
                      <h5 className="text-xs font-semibold text-zinc-400 mb-2 font-serif">Đệ tử Quy chế</h5>
                      <ul className="text-xs text-zinc-400 space-y-1.5 list-disc list-inside">
                        <li>Gia nhập nhận thân phận <strong className="text-zinc-300">Ngoại môn đệ tử</strong>.</li>
                        <li>Có thể thăng cấp qua các kỳ khảo hạch tu vi.</li>
                        <li>Rời khỏi tông môn sẽ <strong className="text-red-400">mất sạch</strong> điểm cống hiến.</li>
                      </ul>
                    </div>
                  </div>
                </div>

                {/* Modal Footer */}
                <div className="flex justify-end gap-3 border-t border-zinc-900 pt-4 mt-2">
                  <button
                    onClick={() => setSelectedSectId(null)}
                    className="px-5 py-2.5 bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 text-zinc-400 hover:text-zinc-200 rounded-xl text-sm font-semibold transition-colors cursor-pointer"
                  >
                    Hủy
                  </button>
                  {mySectStatus?.has_pending_join && mySectStatus.pending_sect_id === sect.id ? (
                    <button
                      onClick={async () => {
                        if (await showConfirm("Đạo hữu có chắc chắn muốn hủy đơn xin gia nhập tông môn này?")) {
                          const ok = await cancelSectJoinRequest();
                          if (ok) setSelectedSectId(null);
                        }
                      }}
                      disabled={actionLoading}
                      className="px-6 py-2.5 bg-gradient-to-r from-red-600 to-rose-600 hover:from-red-500 hover:to-rose-500 disabled:from-zinc-900 disabled:to-zinc-950 disabled:border-zinc-800 disabled:text-zinc-600 disabled:opacity-50 border border-transparent text-white rounded-xl font-semibold shadow-lg shadow-red-500/20 hover:shadow-red-500/35 transition-all text-sm cursor-pointer"
                    >
                      {actionLoading ? "Đang hủy đơn..." : "Hủy Đơn Xin"}
                    </button>
                  ) : (
                    <button
                      onClick={async () => {
                        const ok = await joinSect(sect.id);
                        if (ok) setSelectedSectId(null);
                      }}
                      disabled={actionLoading || mySectStatus?.has_pending_join}
                      className="px-6 py-2.5 bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 disabled:from-zinc-900 disabled:to-zinc-950 disabled:border-zinc-800 disabled:text-zinc-600 disabled:opacity-50 border border-transparent text-white rounded-xl font-semibold shadow-lg shadow-blue-500/20 hover:shadow-blue-500/35 transition-all text-sm cursor-pointer"
                    >
                      {actionLoading
                        ? "Đang gia nhập..."
                        : mySectStatus?.has_pending_join
                          ? mySectStatus.pending_sect_id === sect.id
                            ? "Đang Chờ Duyệt..."
                            : "Đã Đăng Ký Phái Khác"
                          : sect.require_approval
                            ? "Xin Gia Nhập (Cần duyệt)"
                            : "Gia Nhập Môn Phái"}
                    </button>
                  )}
                </div>
              </div>
            </div>
          );
        })()}

        {/* Khai Tông Lập Phái Form */}
        <div className="bg-zinc-950/60 border border-zinc-900 rounded-3xl p-6 backdrop-blur-md relative overflow-hidden flex flex-col gap-4">
          <div className="flex items-center gap-3">
            <Plus className="w-6 h-6 text-yellow-500 animate-pulse" />
            <h3 className="text-lg font-serif font-bold text-zinc-100 bg-gradient-to-r from-yellow-200 to-amber-400 bg-clip-text text-transparent">
              Khai Tông Lập Phái
            </h3>
          </div>
          <p className="text-xs text-zinc-400 leading-relaxed">
            Trở thành thủy tổ của một phái, tự định quy chế, thu nhận đệ tử, tự thiết lập bảng nhiệm vụ và thống trị giang sơn!
            <br />
            <strong className="text-yellow-500 font-mono">Yêu cầu: Tu vi Tam Chuyển trở lên & Chi phí lập phái: 30,000 Nguyên Thạch.</strong>
          </p>

          {(() => {
            const isPlayerInHub = player.current_region === "hub_nam_cuong" || player.current_region === "hub_trung_chau" || player.current_region === "Lobby" || regions.find((r) => r.id === player.current_region)?.zone_type === "HUB" || regions.find((r) => r.id === player.current_region)?.zone_type === "SAFE_ZONE";

            if (player.rank < 3) {
              return (
                <div className="p-3 bg-red-950/15 border border-red-500/20 rounded-xl text-xs text-red-400 font-mono">
                  ⚠️ Đạo hữu chưa đủ tu vi! Cần đạt Tam Chuyển để khai sơn lập tổ (Tu vi hiện tại: Cấp {player.rank}).
                </div>
              );
            }
            if (player.primeval_stones < 30000) {
              return (
                <div className="p-3 bg-red-950/15 border border-red-500/20 rounded-xl text-xs text-red-400 font-mono">
                  ⚠️ Đạo hữu chưa đủ tài lực! Cần 30,000 Nguyên Thạch (Hiện có: {player.primeval_stones.toLocaleString()} Đá).
                </div>
              );
            }
            if (!showCreateForm) {
              return (
                <button
                  onClick={() => setShowCreateForm(true)}
                  className="py-2.5 px-6 bg-gradient-to-r from-yellow-600 to-amber-600 hover:from-yellow-500 hover:to-amber-500 text-white rounded-xl font-serif font-bold text-xs shadow-md transition-all self-start cursor-pointer"
                >
                  Khai Sáng Thế Lực Mới
                </button>
              );
            }

            return (
              <div className="bg-zinc-900/10 border border-zinc-900 rounded-2xl p-4 flex flex-col gap-4 animate-slide-up">
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div className="flex flex-col gap-1.5">
                    <label className="text-xs text-zinc-400 font-semibold font-serif">Tên Tông Môn / Thế Lực</label>
                    <input
                      type="text"
                      placeholder="Ví dụ: Cổ Nguyệt Phái, Vạn Thú Môn..."
                      value={createSectName}
                      onChange={(e) => setCreateSectName(e.target.value)}
                      className="bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-yellow-500/50"
                    />
                  </div>

                  <div className="flex flex-col gap-1.5">
                    <label className="text-xs text-zinc-400 font-semibold font-serif">Vị Trí Khởi Tạo Lãnh Địa</label>
                    <div className="bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs font-mono text-zinc-300 flex items-center justify-between">
                      <span>📍 {getRegionName(player.current_region)} ({player.x}, {player.y})</span>
                      {isPlayerInHub ? (
                        <span className="text-[10px] text-rose-400 font-bold bg-rose-950/40 border border-rose-500/30 px-2 py-0.5 rounded">Trấn (An Toàn Khu)</span>
                      ) : (
                        <span className="text-[10px] text-emerald-400 font-bold bg-emerald-950/40 border border-emerald-500/30 px-2 py-0.5 rounded">Dã Ngoại</span>
                      )}
                    </div>
                  </div>

                  {isPlayerInHub && (
                    <div className="sm:col-span-2 p-3 bg-red-950/20 border border-red-500/30 rounded-xl text-xs text-red-400 font-mono">
                      ⚠️ Không thể khai tông lập phái tại Trấn / An Toàn Khu! Đạo hữu hãy di chuyển ra bản đồ dã ngoại trước khi lập phái.
                    </div>
                  )}

                  <div className="sm:col-span-2 flex flex-col gap-1.5">
                    <label className="text-xs text-zinc-400 font-semibold font-serif">Mô Tả Truyền Thuyết</label>
                    <textarea
                      placeholder="Mô tả tôn chỉ thế lực, lịch sử sáng lập hoặc đặc sắc luyện cổ..."
                      value={createSectDesc}
                      onChange={(e) => setCreateSectDesc(e.target.value)}
                      rows={3}
                      className="bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-yellow-500/50 resize-none"
                    />
                  </div>
                </div>

                <div className="flex gap-3 mt-2">
                  <button
                    onClick={handleCreateSect}
                    disabled={actionLoading || isPlayerInHub || !createSectName.trim() || !createSectDesc.trim()}
                    className="py-2 px-4 bg-gradient-to-r from-yellow-600 to-amber-600 hover:from-yellow-500 hover:to-amber-500 disabled:opacity-50 text-white rounded-xl font-semibold text-xs shadow-md transition-all cursor-pointer"
                  >
                    Xác Nhận Lập Phái (-30,000 Đá)
                  </button>
                  <button
                    onClick={() => setShowCreateForm(false)}
                    className="py-2 px-4 bg-zinc-900 hover:bg-zinc-800 text-zinc-400 rounded-xl font-semibold text-xs transition-colors cursor-pointer"
                  >
                    Hủy
                  </button>
                </div>
              </div>
            );
          })()}
        </div>
      </div>
    );
  }

  const { member, sect, leaderboard } = mySectStatus;
  const isLeaderOrElder = member?.member_rank === "CHUONG_MON" || member?.member_rank === "TRUONG_LAO" || member?.member_rank === "HO_PHAP";

  // Check if sect leader is missing or inactive for >= 5 days
  const currentLeader = (sectMembers || []).find((m) => m.member_rank === "CHUONG_MON");
  const nowMs = Date.now();
  const fiveDaysMs = 5 * 24 * 60 * 60 * 1000;
  const isLeaderInactive = !currentLeader || (
    currentLeader.last_activity ? (nowMs - new Date(currentLeader.last_activity).getTime() >= fiveDaysMs) : true
  );

  // Find candidate with top seniority
  const nonLeaderCandidates = (sectMembers || [])
    .filter((m) => m.member_rank !== "CHUONG_MON")
    .sort((a, b) => {
      const diff = rankValue(b.member_rank) - rankValue(a.member_rank);
      if (diff !== 0) return diff;
      return b.contribution_points - a.contribution_points;
    });
  const topCandidate = nonLeaderCandidates[0];
  const isEligibleToClaim = isLeaderInactive && topCandidate && topCandidate.player_id === player?.id;

  // 2. STATE: MEMBER PANEL
  return (
    <GamePanel variant="default" className="w-full mx-auto px-2 py-4 flex flex-col gap-6 animate-fade-in">
      {/* Sect Banner HUD */}
      <div className="bg-zinc-950/60 border border-zinc-900 rounded-3xl p-5 flex flex-col md:flex-row justify-between items-start md:items-center gap-5 shadow-xl backdrop-blur-md relative overflow-hidden">
        <div className="absolute top-0 right-0 w-32 h-32 bg-blue-500/5 rounded-full blur-2xl -z-10" />

        <div className="flex items-center gap-4">
          <div className="w-12 h-12 rounded-2xl bg-gradient-to-br from-blue-600 to-indigo-600 flex items-center justify-center border border-blue-400/20 shadow-lg">
            <Shield className="w-6 h-6 text-white" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-lg sm:text-xl font-serif font-bold text-zinc-100">
                {sect?.name}
              </h2>
              {sect?.level !== undefined && (
                <span className="text-[10px] bg-blue-950/40 text-blue-400 border border-blue-800/40 px-2 py-0.5 rounded-lg font-bold font-mono">
                  Cấp {sect.level}
                </span>
              )}
            </div>
            <p className="text-xs text-zinc-500 font-mono mt-0.5 flex items-center flex-wrap gap-1">
              <span>Địa Khu: {getRegionName(sect?.region || "")}{sect?.map_x !== undefined && sect?.map_y !== undefined ? ` (${sect.map_x}, ${sect.map_y})` : ""} | {getRankName(member?.member_rank || "")}</span>
              {member?.member_rank === "CHUONG_MON" && (
                <button
                  onClick={() => { setActiveTab("manage"); setMgmtTab("relocate"); }}
                  className="text-[11px] text-amber-400 hover:text-amber-300 underline font-mono cursor-pointer ml-1"
                >
                  [Di dời sơn môn]
                </button>
              )}
            </p>
          </div>
        </div>

        {/* Contribution Badge & Sect Funds */}
        <div className="flex items-center gap-6 w-full md:w-auto justify-between border-t md:border-t-0 pt-3 md:pt-0 border-zinc-900">
          {sect?.funds !== undefined && (
            <div className="flex flex-col">
              <span className="text-[10px] text-zinc-500 font-mono uppercase tracking-wider">Quỹ Tông Môn</span>
              <span className="text-lg font-mono font-bold text-cyan-400 flex items-center gap-1.5">
                <Coins className="w-4.5 h-4.5 text-cyan-400" />
                {sect.funds.toLocaleString()}
              </span>
              {mySectStatus?.estimated_daily_maintenance !== undefined && (
                <span className="text-[10px] font-mono text-zinc-400 mt-0.5" title="Phí duy trì mỗi 24h khấu trừ vào Quỹ tông môn">
                  Duy trì: <span className="text-amber-400">-{mySectStatus.estimated_daily_maintenance.toLocaleString()}</span>/ngày
                </span>
              )}
            </div>
          )}

          <div className="flex flex-col">
            <span className="text-[10px] text-zinc-500 font-mono uppercase tracking-wider">Điểm Cống Hiến</span>
            <span className="text-lg font-mono font-bold text-yellow-400 flex items-center gap-1.5">
              <Coins className="w-4.5 h-4.5 text-yellow-500" />
              {member?.contribution_points || 0}
            </span>
          </div>

          <div className="flex items-center gap-1 bg-zinc-900/80 px-3 py-1.5 rounded-xl border border-zinc-800 font-mono text-[10px] text-zinc-400">
            <span>Tham nhập:</span>
            <span>{member?.joined_at ? new Date(member.joined_at).toLocaleDateString() : ""}</span>
          </div>
        </div>
      </div>

      <GameTabs
      items={[
        { id: "lobby", label: "Sảnh", disabled: false },
        { id: "missions", label: "Nhiệm Vụ", disabled: false },
        { id: "vault", label: "Bảo Khố", disabled: false },
        { id: "donate", label: "Quyên Góp", disabled: false },
        { id: "territory", label: "Lãnh Địa", disabled: false },
        ...(isLeaderOrElder ? [{ id: "manage", label: "Quản Lý", disabled: false }] : []),
      ]}
      activeId={activeTab}
      onChange={(id) => setActiveTab(id as ApiResponseData)}
      variant="pills"
      size="md"
      className=""
      scrollable={true}
      fullWidth={false}
    >
</GameTabs>
      {/* Tab Panels */}
      {/* Succession Banner when leader is inactive or missing */}
      {isLeaderInactive && (
        <div className="bg-amber-950/30 border border-amber-500/40 rounded-3xl p-5 flex flex-col md:flex-row justify-between items-start md:items-center gap-4 shadow-lg backdrop-blur-md">
          <div className="flex items-start gap-3">
            <div className="p-2.5 rounded-2xl bg-amber-500/10 border border-amber-500/30 text-amber-400 shrink-0">
              <Crown className="w-6 h-6" />
            </div>
            <div>
              <h4 className="text-sm font-serif font-bold text-amber-200 flex items-center gap-2">
                Sơn Môn Đại Sự: Kế Vị Chưởng Môn
                <span className="text-[10px] font-mono px-2 py-0.5 rounded-full bg-amber-500/20 text-amber-300 border border-amber-500/40">
                  {currentLeader ? "Chưởng Môn Vắng Bóng 5+ Ngày" : "Khuyết Chưởng Môn"}
                </span>
              </h4>
              <p className="text-xs text-amber-300/80 font-mono mt-1 leading-relaxed">
                {currentLeader
                  ? `Chưởng Môn (${currentLeader.username}) đã không đăng nhập hoặc vận công trong hơn 5 ngày. Tông môn đang đình trệ!`
                  : "Tông môn hiện không có ai đảm đương chức vụ Chưởng Môn!"}
                {" "}Người có tư cách kế vị ưu tiên cao nhất: <strong className="text-amber-200 underline font-bold">{topCandidate ? `${topCandidate.username} (${getRankName(topCandidate.member_rank)}, ${topCandidate.contribution_points} Pts)` : "Không có"}</strong>.
              </p>
            </div>
          </div>

          <div className="w-full md:w-auto shrink-0 flex justify-end">
            {isEligibleToClaim ? (
              <button
                type="button"
                onClick={async () => {
                  if (await showConfirm("Đạo hữu có tư cách cao nhất hiện tại! Đạo hữu có chắc chắn muốn tiến hành đại điển kế vị, chính thức đăng cơ Chưởng Môn tông môn?")) {
                    await claimSectLeadership();
                  }
                }}
                disabled={actionLoading}
                className="w-full md:w-auto px-5 py-2.5 bg-gradient-to-r from-amber-600 to-yellow-600 hover:from-amber-500 hover:to-yellow-500 text-white font-serif font-bold text-xs rounded-xl shadow-lg shadow-amber-500/20 transition-all cursor-pointer flex items-center justify-center gap-2 border border-amber-400/30"
              >
                <Crown className="w-4 h-4 text-yellow-200" />
                Tiếp Quản Chưởng Môn
              </button>
            ) : (
              <div className="text-[11px] font-mono text-zinc-500 bg-zinc-900/60 border border-zinc-800 px-3 py-2 rounded-xl">
                Chưa đủ tư cách kế vị (Cần thứ bậc/cống hiến cao nhất)
              </div>
            )}
          </div>
        </div>
      )}

      {activeTab === "lobby" && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 animate-slide-up">
          {/* Main Info & Promotion Exam */}
          <div className="lg:col-span-2 flex flex-col gap-6">
            <div className="bg-zinc-950/40 border border-zinc-900 rounded-3xl p-5 backdrop-blur-md">
              <h3 className="text-base font-serif font-bold text-zinc-200 mb-3 flex items-center gap-2">
                <BookOpen className="w-5 h-5 text-blue-400" />
                Nội Tình Tông Môn
              </h3>
              <p className="text-sm text-zinc-400 leading-relaxed mb-4">{sect?.description}</p>
              <div className="p-3.5 bg-zinc-900/30 border border-zinc-900 rounded-2xl flex items-center justify-between text-xs text-zinc-500 font-mono">
                <span>Tu vi thực tế của đạo hữu:</span>
                <span className="text-zinc-300 font-semibold">Bậc {player.rank}</span>
              </div>
            </div>

            {/* Sect Maintenance Status Card */}
            {mySectStatus?.maintenance_breakdown && (
              <div className="bg-zinc-950/40 border border-zinc-900 rounded-3xl p-5 backdrop-blur-md flex flex-col gap-3">
                <div className="flex justify-between items-center border-b border-zinc-900/60 pb-2.5">
                  <h3 className="text-base font-serif font-bold text-zinc-200 flex items-center gap-2">
                    <Hourglass className="w-5 h-5 text-amber-400" />
                    Phí Duy Trì Sơn Môn (Mỗi 24 Giờ)
                  </h3>
                  <span className={`text-xs font-mono font-bold px-2.5 py-1 rounded-full border ${
                    (sect?.funds || 0) >= (mySectStatus.maintenance_breakdown.total_fee || 0)
                      ? "text-emerald-400 bg-emerald-950/30 border-emerald-500/30"
                      : "text-red-400 bg-red-950/30 border-red-500/30 animate-pulse"
                  }`}>
                    {(sect?.funds || 0) >= (mySectStatus.maintenance_breakdown.total_fee || 0) ? "Đủ Ngân Quỹ" : "Nguy Cơ Giáng Cấp!"}
                  </span>
                </div>

                <p className="text-xs text-zinc-400 font-mono leading-relaxed">
                  Định kỳ mỗi 24 giờ, hệ thống sẽ tự động trích Quỹ Tông Môn để bảo trì sơn môn theo quy mô thực tế. Nếu không đủ quỹ, tông môn sẽ bị giáng 1 cấp.
                </p>

                <div className="grid grid-cols-2 sm:grid-cols-3 gap-2 text-xs font-mono mt-1">
                  <div className="bg-zinc-900/40 border border-zinc-900 p-2.5 rounded-xl flex flex-col gap-1">
                    <span className="text-[10px] text-zinc-500">Cấp Tông Môn (Cấp {mySectStatus.maintenance_breakdown.sect_level}):</span>
                    <span className="text-zinc-200 font-bold">{mySectStatus.maintenance_breakdown.level_fee.toLocaleString()} Quỹ</span>
                  </div>
                  <div className="bg-zinc-900/40 border border-zinc-900 p-2.5 rounded-xl flex flex-col gap-1">
                    <span className="text-[10px] text-zinc-500">Môn nhân ({mySectStatus.maintenance_breakdown.member_count} TV):</span>
                    <span className="text-zinc-200 font-bold">{mySectStatus.maintenance_breakdown.member_fee.toLocaleString()} Quỹ</span>
                  </div>
                  <div className="bg-zinc-900/40 border border-zinc-900 p-2.5 rounded-xl flex flex-col gap-1">
                    <span className="text-[10px] text-zinc-500">Linh tài ({mySectStatus.maintenance_breakdown.materials_count} món):</span>
                    <span className="text-zinc-200 font-bold">{mySectStatus.maintenance_breakdown.materials_fee.toLocaleString()} Quỹ</span>
                  </div>
                  <div className="bg-zinc-900/40 border border-zinc-900 p-2.5 rounded-xl flex flex-col gap-1">
                    <span className="text-[10px] text-zinc-500">Phàm Cổ ({mySectStatus.maintenance_breakdown.mortal_gu_count} con):</span>
                    <span className="text-zinc-200 font-bold">{mySectStatus.maintenance_breakdown.mortal_gu_fee.toLocaleString()} Quỹ</span>
                  </div>
                  <div className="bg-zinc-900/40 border border-zinc-900 p-2.5 rounded-xl flex flex-col gap-1">
                    <span className="text-[10px] text-zinc-500">Tiên Cổ ({mySectStatus.maintenance_breakdown.immortal_gu_count} con):</span>
                    <span className="text-zinc-200 font-bold">{mySectStatus.maintenance_breakdown.immortal_gu_fee.toLocaleString()} Quỹ</span>
                  </div>
                  <div className="bg-zinc-900/40 border border-zinc-900 p-2.5 rounded-xl flex flex-col gap-1">
                    <span className="text-[10px] text-zinc-500">Bí phương ({mySectStatus.maintenance_breakdown.recipes_count} quyển):</span>
                    <span className="text-zinc-200 font-bold">{mySectStatus.maintenance_breakdown.recipes_fee.toLocaleString()} Quỹ</span>
                  </div>
                </div>

                <div className="flex justify-between items-center bg-emerald-950/20 border border-emerald-900/50 p-3 rounded-xl mt-1 text-emerald-400">
                  <span className="text-xs">Bảo Khố cấp {mySectStatus.maintenance_breakdown.treasury_level} đã tiết kiệm:</span>
                  <span className="text-sm font-mono font-bold">-{mySectStatus.maintenance_breakdown.treasury_discount.toLocaleString()} Quỹ</span>
                </div>

                <div className="flex justify-between items-center bg-amber-950/20 border border-amber-500/30 p-3 rounded-xl mt-1">
                  <span className="text-xs font-serif font-bold text-amber-200">Dự kiến phí bảo trì ngày:</span>
                  <span className="text-sm font-mono font-bold text-amber-400">
                    {mySectStatus.maintenance_breakdown.total_fee.toLocaleString()} Quỹ Tông Môn
                  </span>
                </div>
              </div>
            )}

            {/* Promotion Exam Panel */}
            <div className="bg-zinc-950/40 border border-zinc-900 rounded-3xl p-5 backdrop-blur-md flex flex-col gap-4">
              <h3 className="text-base font-serif font-bold text-zinc-200 flex items-center gap-2">
                <Award className="w-5 h-5 text-blue-400" />
                Khảo Hạch Thăng Cấp Đệ Tử
              </h3>

              {promotionReq ? (
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div className="bg-zinc-900/20 border border-zinc-900 rounded-2xl p-4 flex flex-col gap-3">
                    <span className="text-xs text-zinc-500 font-mono">Thăng tiến tiếp theo:</span>
                    <div className="flex items-center gap-2.5">
                      <span className={`px-2 py-0.5 text-[10px] font-mono rounded-lg border ${getRankColor(member?.member_rank || "")}`}>
                        {getRankName(member?.member_rank || "")}
                      </span>
                      <ArrowRight className="w-3.5 h-3.5 text-zinc-500" />
                      <span className={`px-2 py-0.5 text-[10px] font-mono rounded-lg border ${getRankColor(promotionReq.next)}`}>
                        {getRankName(promotionReq.next)}
                      </span>
                    </div>
                  </div>

                  <div className="bg-zinc-900/20 border border-zinc-900 rounded-2xl p-4 flex flex-col gap-2">
                    <span className="text-xs text-zinc-500 font-mono">Điều kiện khảo hạch:</span>
                    <div className="space-y-1 text-xs">
                      <div className="flex justify-between font-mono">
                        <span>Linh thạch cống hiến:</span>
                        <span className={member!.contribution_points >= promotionReq.points ? "text-emerald-400 font-semibold" : "text-red-400"}>
                          {member!.contribution_points} / {promotionReq.points}
                        </span>
                      </div>
                      <div className="flex justify-between font-mono">
                        <span>Tu vi của Cổ sư:</span>
                        <span className={player.rank >= promotionReq.reqRank ? "text-emerald-400 font-semibold" : "text-red-400"}>
                          Rank {player.rank} / Rank {promotionReq.reqRank}
                        </span>
                      </div>
                    </div>
                  </div>

                  {mySectStatus?.has_pending_promotion && (
                    <div className="sm:col-span-2 p-3 bg-amber-950/40 border border-amber-500/25 rounded-2xl text-xs text-amber-400 font-mono text-center">
                      ⏳ Đơn xin thăng cấp lên <strong className="text-amber-200 font-bold">{getRankName(mySectStatus.pending_promotion_target || "")}</strong> của đạo hữu đang chờ duyệt!
                    </div>
                  )}

                  <button
                    onClick={promoteSectRank}
                    disabled={!canPromote || actionLoading || mySectStatus?.has_pending_promotion}
                    className="sm:col-span-2 py-3 bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 disabled:from-zinc-900 disabled:to-zinc-950 disabled:border-zinc-800 disabled:text-zinc-600 border border-transparent disabled:opacity-50 text-white rounded-2xl font-serif font-bold text-sm shadow-lg shadow-blue-500/10 hover:shadow-blue-500/25 transition-all cursor-pointer"
                  >
                    {actionLoading
                      ? "Đang tiến hành khảo hạch..."
                      : mySectStatus?.has_pending_promotion
                        ? "Yêu Cầu Thăng Cấp Đang Chờ Duyệt..."
                        : canPromote
                          ? "Bắt Đầu Khảo Hạch Thăng Cấp"
                          : "Chưa Đạt Yêu Cầu Khảo Hạch"}
                  </button>
                </div>
              ) : (
                <div className="py-6 bg-zinc-900/10 border border-zinc-900 border-dashed rounded-2xl text-center text-xs text-amber-400 font-serif">
                  🎉 Đạo hữu đã đạt đến cấp bậc cao nhất của tông môn (Hộ Pháp)!
                </div>
              )}
            </div>

            {/* Leave Sect Button */}
            <div className="border border-red-950/20 bg-red-950/5 rounded-3xl p-5 flex flex-col sm:flex-row items-center justify-between gap-4">
              <div>
                <h4 className="text-sm font-serif font-bold text-red-200">Trục Xuất / Phản Xuất Tông Môn</h4>
                <p className="text-xs text-red-400/70 mt-0.5">
                  Rời khỏi tông môn sẽ thu hồi lệnh bài tông môn và xóa bỏ toàn bộ {member?.contribution_points} điểm cống hiến hiện tại.
                </p>
              </div>
              <button
                onClick={async () => {
                  if (await showConfirm("Đạo hữu có chắc chắn muốn rời khỏi tông môn? Điểm cống hiến sẽ hoàn toàn biến mất!")) {
                    await leaveSect();
                  }
                }}
                disabled={actionLoading}
                className="w-full sm:w-auto px-4 py-2 border border-red-500/30 hover:bg-red-950/20 text-red-400 hover:text-red-300 font-semibold text-xs rounded-xl transition-all cursor-pointer flex items-center gap-1.5 justify-center"
              >
                <LogOut className="w-3.5 h-3.5" />
                Rời Tông Môn
              </button>
            </div>
          </div>

          {/* Leaderboard inside the sect */}
          <div className="bg-zinc-950/40 border border-zinc-900 rounded-3xl p-5 backdrop-blur-md flex flex-col gap-4">
            <h3 className="text-base font-serif font-bold text-zinc-200 flex items-center gap-2">
              <TrendingUp className="w-5 h-5 text-blue-400" />
              Đông Đường Cổ Tín
            </h3>
            <span className="text-[10px] text-zinc-500 font-mono -mt-2 uppercase tracking-wider">
              Bảng xếp hạng cống hiến
            </span>

            <div className="flex flex-col gap-2 max-h-[360px] overflow-y-auto pr-1">
              {leaderboard && leaderboard.length > 0 ? (
                leaderboard.map((entry, idx) => (
                  <div
                    key={idx}
                    className={`flex justify-between items-center p-3 rounded-2xl border text-xs ${entry.username === player.username
                        ? "bg-blue-950/20 border-blue-500/20 text-blue-100"
                        : "bg-zinc-900/20 border-zinc-900 text-zinc-300"
                      }`}
                  >
                    <div className="flex items-center gap-2.5 min-w-0">
                      <span className="font-mono font-bold text-[10px] text-zinc-500 w-4">
                        {idx + 1}.
                      </span>
                      <div className="w-6 h-6 rounded-full shrink-0 overflow-hidden bg-zinc-800 flex items-center justify-center text-[8px] font-bold text-zinc-500">
                        {entry.avatar_url ? (
                          <img src={entry.avatar_url} alt="" className="w-full h-full object-cover" />
                        ) : (
                          entry.username.slice(0, 2).toUpperCase()
                        )}
                      </div>
                      <div className="flex flex-col min-w-0">
                        <span className="font-semibold truncate">{entry.username}</span>
                        <span className="text-[9px] text-zinc-500 font-mono">
                          {getRankName(entry.rank)}
                        </span>
                      </div>
                    </div>
                    <span className="font-mono font-bold text-yellow-500 shrink-0">
                      {entry.contribution} Pts
                    </span>
                  </div>
                ))
              ) : (
                <div className="text-center py-6 text-zinc-600 text-xs font-mono">Chưa có thành viên nào xếp hạng</div>
              )}
            </div>
          </div>
        </div>
      )}

      {activeTab === "territory" && (
        <div className="flex flex-col gap-6 animate-slide-up">
          {/* Section 0: Trấn Thủ Tông Môn Toggle */}
          <div className="bg-zinc-950/40 border border-zinc-900 rounded-3xl p-4 backdrop-blur-md">
            <div className="flex justify-between items-center border-b border-zinc-900/60 pb-2">
              <h3 className="text-sm font-serif font-bold text-zinc-200">
                <Shield className="w-4 h-4 text-amber-400" />
                Trấn Thủ Tông Môn
              </h3>
              <span className="text-xs font-mono text-amber-400 bg-amber-950/30 border border-amber-500/30 px-2 py-1 rounded-full">
                {mySectStatus?.member?.is_defending ? "BẬT" : "TẮT"}
              </span>
            </div>
            <p className="text-xs text-zinc-400 leading-relaxed">
              Đạo hữu sẽ tham chiến khi Tông Môn bị tấn công. Bạn phải đứng tại tọa độ Tông Môn mới có thể bật chế độ.
            </p>
            <div className="flex gap-2 pt-3">
              <button
                onClick={async () => {
                  const ok = await toggleDefendSect();
                  if (ok) {
                    // Refresh the UI state
                    fetchMySectStatus();
                    fetchSectsMap();
                  }
                }}
                disabled={!mySectStatus?.member || actionLoading}
                className={`w-full py-2.5 bg-gradient-to-r from-emerald-600 to-green-600 hover:from-emerald-500 hover:to-green-500 disabled:opacity-50 text-white rounded-xl font-serif font-bold text-xs shadow-md transition-all cursor-pointer ${mySectStatus?.member?.is_defending ? "animate-pulse" : ""}`}
              >
                {mySectStatus?.member?.is_defending ? "Tắt Trấn Thủ" : "Bật Trấn Thủ"}
              </button>
            </div>
          </div>

          {/* Section 1: Sect Buildings & Upgrades */}
          <div className="bg-zinc-950/40 border border-zinc-900 rounded-3xl p-5 backdrop-blur-md flex flex-col gap-4">
            <div className="flex justify-between items-center border-b border-zinc-900/60 pb-3">
              <h3 className="text-base font-serif font-bold text-zinc-200 flex items-center gap-2">
                <Sparkles className="w-5 h-5 text-amber-400" />
                Kiến Trúc Tông Môn
              </h3>
              <span className="text-xs font-mono text-amber-400 bg-amber-950/30 border border-amber-500/30 px-3 py-1 rounded-full">
                Cấp Tông Môn: {mySectStatus?.sect?.buildings?.main_hall || mySectStatus?.sect?.level || 1}
              </span>
            </div>
            <p className="text-xs text-zinc-400 leading-relaxed">
              Nâng cấp kiến trúc giúp gia tăng sản lượng tài nguyên tự động, mở rộng bán kính Buff Hộ Sơn và gia cố Hộ Sơn Đại Trận bảo vệ Tông môn.
            </p>

            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 mt-2">
              {(() => {
                const bMap = mySectStatus?.sect?.buildings || {};
                const currentFunds = mySectStatus?.sect?.funds || 0;
                const canManage = mySectStatus?.member?.member_rank === "CHUONG_MON" || mySectStatus?.member?.member_rank === "TRUONG_LAO" || mySectStatus?.member?.member_rank === "HO_PHAP";
                const mainHallLvl = bMap.main_hall || 1;

                const buildingDefs = [
                  {
                    key: "main_hall",
                    name: "Chính Điện",
                    desc: "Trọng địa Tông môn. Cấp Chính Điện quyết định Cấp Tông Môn & Bán kính Buff Hộ Sơn.",
                    baseFunds: 5000,
                    icon: "🏛️",
                  },
                  {
                    key: "spirit_vein",
                    name: "Linh Mạch",
                    desc: "Tự động sản sinh Quỹ tông môn và Linh tài mỗi phút cho Tông môn.",
                    baseFunds: 3000,
                    icon: "💎",
                  },
                  {
                    key: "defensive_array",
                    name: "Hộ Sơn Đại Trận",
                    desc: "Tăng sức mạnh thủ hộ. Khi hết hồi 12 giờ, Đại Trận chặn đợt tấn công đầu tiên và mở 10 phút để môn nhân tập hợp.",
                    baseFunds: 2000,
                    icon: "🛡️",
                  },
                  {
                    key: "treasury",
                    name: "Bảo Khố",
                    desc: "Giảm tỷ lệ từng tài nguyên/Cổ/Bí phương rơi khi thất thủ, đồng thời giảm mạnh phí duy trì và di dời tài sản trong kho.",
                    baseFunds: 1500,
                    icon: "🏛️",
                  },
                  {
                    key: "training_ground",
                    name: "Diễn Võ Trường",
                    desc: "Tăng tỷ lệ tích lũy Điểm Cống Hiến tự động cho tất cả đệ tử Tông môn.",
                    baseFunds: 1500,
                    icon: "⚔️",
                  },
                ];

                return buildingDefs.map((b) => {
                  const lvl = bMap[b.key] || 1;
                  const targetLvl = lvl + 1;
                  const fundCost = b.baseFunds * targetLvl * targetLvl;
                  const canAfford = currentFunds >= fundCost;
                  const isMainHall = b.key === "main_hall";
                  const levelCapped = !isMainHall && lvl >= mainHallLvl;

                  return (
                    <div key={b.key} className="bg-zinc-900/20 border border-zinc-900 rounded-2xl p-4 flex flex-col justify-between gap-3 relative overflow-hidden">
                      <div className="flex flex-col gap-2">
                        <div className="flex justify-between items-start">
                          <div className="flex items-center gap-2">
                            <span className="text-lg">{b.icon}</span>
                            <h4 className="text-sm font-bold font-serif text-zinc-200">{b.name}</h4>
                          </div>
                          <span className="text-xs font-mono font-bold px-2.5 py-0.5 rounded bg-zinc-900 text-amber-400 border border-amber-500/20">
                            Cấp {lvl}
                          </span>
                        </div>
                        <p className="text-[11px] text-zinc-400 leading-snug">{b.desc}</p>
                      </div>

                      <div className="flex flex-col gap-2 pt-2 border-t border-zinc-900/60 mt-2">
                        <div className="flex justify-between items-center text-[11px] font-mono">
                          <span className="text-zinc-500">Phí nâng cấp:</span>
                          <span className={canAfford ? "text-amber-400 font-bold" : "text-rose-400"}>
                            {fundCost.toLocaleString()} Quỹ
                          </span>
                        </div>

                        {canManage ? (
                          <button
                            onClick={async () => {
                              if (levelCapped) {
                                addToast("Cấp kiến trúc không được vượt quá cấp Chính Điện!", "error");
                                return;
                              }
                              await upgradeSectBuilding(b.key);
                            }}
                            disabled={actionLoading || !canAfford || levelCapped}
                            className="w-full py-2 bg-amber-600/20 hover:bg-amber-600/30 text-amber-300 border border-amber-500/30 disabled:opacity-40 disabled:border-zinc-800 disabled:bg-zinc-900 disabled:text-zinc-600 rounded-xl text-xs font-bold transition-colors cursor-pointer"
                          >
                            {levelCapped ? "Cần Nâng Chính Điện" : !canAfford ? "Không Đủ Quỹ" : `Nâng Lên Cấp ${targetLvl}`}
                          </button>
                        ) : (
                          <div className="text-[10px] text-center text-zinc-600 font-mono py-1">Chỉ cấp quản lý mới được nâng cấp</div>
                        )}
                      </div>
                    </div>
                  );
                });
              })()}
            </div>
          </div>

          {/* Defend Modal */}
          {showDefendModal && (
            <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
              <div className="w-full max-w-md bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4">
                <div className="flex items-center justify-between">
                  <h3 className="text-sm font-bold text-zinc-100 font-serif">Bố Trí Phòng Thủ Tông Môn</h3>
                  <button onClick={() => setShowDefendModal(false)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer"><X className="w-4 h-4 text-zinc-500" /></button>
                </div>

                <div className="flex bg-zinc-900/50 p-1 rounded-xl border border-zinc-800/60 w-fit">
                  <button
                    onClick={() => { setDefendType("member"); fetchSectMembers(); }}
                    className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${defendType === "member" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"}`}
                  >
                    Đệ Tử
                  </button>
                  <button
                    onClick={() => setDefendType("beast")}
                    className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${defendType === "beast" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"}`}
                  >
                    Thần Thú
                  </button>
                </div>

                {defendType === "member" ? (
                  <div className="flex flex-col gap-1.5">
                    <label className="text-[11px] font-bold text-zinc-500 font-mono">Chọn đệ tử phòng thủ</label>
                    <select
                      value={defendPlayerId}
                      onChange={(e) => setDefendPlayerId(e.target.value)}
                      className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-cyan-500/50"
                    >
                      <option value="">-- Chọn đệ tử --</option>
                      {sectMembers
                        .filter((m) => m.player_id !== player.id)
                        .map((m) => (
                          <option key={m.player_id} value={m.player_id}>
                            {m.username} ({getRankName(m.member_rank)})
                          </option>
                        ))}
                    </select>
                  </div>
                ) : (
                  <div className="flex flex-col gap-1.5">
                    <label className="text-[11px] font-bold text-zinc-500 font-mono">Chọn thần thú canh giữ</label>
                    <select
                      value={defendMonsterId}
                      onChange={(e) => {
                        const id = e.target.value;
                        setDefendMonsterId(id);
                        const spec = monsterDictionary[id];
                        if (spec) setDefendMonsterRank(spec.rank || 1);
                      }}
                      className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-cyan-500/50"
                    >
                      <option value="">-- Chọn thần thú --</option>
                      {Object.entries(monsterDictionary || {}).map(([id, spec]) => (
                        <option key={id} value={id}>{spec.name} (Hạng {spec.rank || 1})</option>
                      ))}
                    </select>

                    <div className="flex flex-col gap-1.5">
                      <label className="text-[11px] font-bold text-zinc-500 font-mono">Hạng thần thú</label>
                      <input
                        type="number"
                        min={1}
                        max={9}
                        value={defendMonsterRank}
                        onChange={(e) => setDefendMonsterRank(Math.max(1, Math.min(9, parseInt(e.target.value) || 1)))}
                        className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 mt-1 focus:outline-none focus:border-cyan-500/50 font-mono"
                      />
                      <span className="text-[10px] text-amber-500/80 font-mono">
                        Phí thuê: {(defendMonsterRank || 1) * 500} Quỹ tông môn.
                      </span>
                    </div>
                  </div>
                )}

                <button
                  disabled={actionLoading || (defendType === "member" ? !defendPlayerId : !defendMonsterId)}
                  onClick={async () => {
                    const ok = await defendSect(
                      defendType,
                      defendType === "member" ? defendPlayerId : undefined,
                      defendType === "beast" ? defendMonsterId : undefined,
                      defendType === "beast" ? defendMonsterRank : undefined
                    );
                    if (ok) setShowDefendModal(false);
                  }}
                  className="w-full py-2.5 bg-gradient-to-r from-cyan-600 to-sky-600 hover:from-cyan-500 hover:to-sky-500 disabled:opacity-50 text-white rounded-xl font-serif font-bold text-xs shadow-md transition-all cursor-pointer"
                >
                  Xác Nhận Bố Trí
                </button>
              </div>
            </div>
          )}
        </div>
      )}

      {activeTab === "missions" && (
        <div className="flex flex-col gap-6 animate-slide-up">
          {/* Active Mission */}
          <div className="bg-zinc-950/40 border border-zinc-900 rounded-3xl p-5 backdrop-blur-md flex flex-col gap-4">
            <h3 className="text-base font-serif font-bold text-zinc-200 flex items-center gap-2 border-b border-zinc-900/60 pb-3">
              <Activity className="w-5 h-5 text-blue-400" />
              Nhiệm Vụ Đang Thực Hiện
            </h3>

            {mySectStatus.has_active && mySectStatus.active_mission ? (() => {
              const am = mySectStatus.active_mission;
              const progress = mySectStatus.active_progress || 0;

              // Find if gather requirements are met
              let isReadyToComplete = false;
              if (am.mission_type === "GATHER") {
                const owned = player.materials_inventory[am.target_id] || 0;
                isReadyToComplete = owned >= am.target_amount;
              } else if (am.mission_type === "DONATE") {
                const matchingGus = (player.gu_inventory || []).filter(g => g.gu_id === am.target_id && !g.is_vital && !g.equipped);
                isReadyToComplete = matchingGus.length > 0;
              }

              return (
                <div className="bg-blue-950/15 border border-blue-500/20 rounded-2xl p-4 flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
                  <div className="flex flex-col gap-1.5">
                    <span className="text-[10px] text-blue-400 font-mono uppercase tracking-wider flex items-center gap-1">
                      <Hourglass className="w-3.5 h-3.5 animate-spin" />
                      {getMissionTypeName(am.mission_type)}
                    </span>
                    <h4 className="text-base font-serif font-bold text-zinc-100">{am.title}</h4>
                    <p className="text-xs text-zinc-400 leading-relaxed max-w-xl">{am.description}</p>

                    {formatMissionTime(am.expires_at) && (
                      <span className="text-[10px] font-mono text-amber-400/90 flex items-center gap-1">
                        <Hourglass className="w-3 h-3" />
                        Hết hạn sau: {formatMissionTime(am.expires_at)}
                      </span>
                    )}

                    {/* Mission Requirements Display */}
                    <div className="mt-2 text-xs font-mono text-zinc-300">
                      {am.mission_type === "GATHER" && (() => {
                        const owned = player.materials_inventory[am.target_id] || 0;
                        const matSpec = itemDictionary[am.target_id];
                        return (
                          <span>
                            Tiến độ: {owned >= am.target_amount ? "✅" : "❌"} {owned} / {am.target_amount}x {matSpec?.name || am.target_id}
                          </span>
                        );
                      })()}
                      {am.mission_type === "DONATE" && (() => {
                        const matchingGus = (player.gu_inventory || []).filter(g => g.gu_id === am.target_id && !g.is_vital && !g.equipped);
                        const guSpec = guDictionary[am.target_id];
                        return (
                          <span>
                            Tiến độ: {matchingGus.length > 0 ? "✅ Sẵn sàng quyên góp" : `❌ Cần Cổ trùng: ${guSpec?.name || am.target_id} trong túi (không trang bị / không bản mệnh)`}
                          </span>
                        );
                      })()}
                    </div>
                  </div>

                  <div className="flex flex-col items-end gap-2.5 w-full sm:w-auto border-t sm:border-t-0 pt-3 sm:pt-0 border-zinc-900">
                    <div className="text-right">
                      <span className="text-[10px] text-zinc-500 font-mono block">Thưởng cống hiến</span>
                      <span className="text-sm font-mono font-bold text-yellow-400">+{am.reward_contribution} Pts</span>
                    </div>
                    <div className="flex items-center gap-2 w-full sm:w-auto">
                      <button
                        onClick={async () => {
                          if (!(await showConfirm("Đạo hữu có chắc chắn muốn hủy nhiệm vụ đang thực hiện này?"))) return;
                          await abandonSectMission(am.id);
                        }}
                        disabled={actionLoading}
                        className="flex-1 sm:flex-none px-3 py-2 bg-zinc-900/60 hover:bg-red-950/40 hover:text-red-300 border border-zinc-800 hover:border-red-500/40 text-zinc-400 rounded-xl font-semibold text-xs transition-colors cursor-pointer flex items-center justify-center gap-1.5"
                        title="Hủy nhiệm vụ"
                      >
                        <Ban className="w-4 h-4" />
                        Hủy Nhiệm Vụ
                      </button>
                      <button
                        onClick={async () => {
                          await completeSectMission(am.id);
                        }}
                        disabled={!isReadyToComplete || actionLoading}
                        className="flex-1 sm:flex-none px-5 py-2 bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 disabled:from-zinc-900 disabled:to-zinc-950 disabled:border-zinc-800 disabled:text-zinc-600 disabled:opacity-50 text-white rounded-xl font-semibold text-xs shadow-lg transition-all cursor-pointer flex items-center justify-center gap-1.5"
                      >
                        <CheckCircle className="w-4 h-4" />
                        Nộp Nhiệm Vụ
                      </button>
                    </div>
                  </div>
                </div>
              );
            })() : (
              <div className="py-8 bg-zinc-900/10 border border-zinc-900 border-dashed rounded-2xl text-center text-xs text-zinc-500 font-mono">
                Chưa nhận nhiệm vụ nào. Nhận nhiệm vụ môn phái phía dưới để kiếm cống hiến điểm!
              </div>
            )}
          </div>

          {/* Open Missions list */}
          <div className="bg-zinc-950/40 border border-zinc-900 rounded-3xl p-5 backdrop-blur-md flex flex-col gap-4">
            <div className="flex justify-between items-center border-b border-zinc-900/60 pb-3">
              <h3 className="text-base font-serif font-bold text-zinc-200">
                Nhiệm Vụ Môn Phái Có Sẵn
              </h3>
              <button
                onClick={() => fetchSectMissions()}
                className="p-1 rounded-lg border border-zinc-800 bg-zinc-950 text-zinc-500 hover:text-zinc-300 transition-colors"
                title="Làm mới bảng"
              >
                <RefreshCw className="w-3.5 h-3.5" />
              </button>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {sectMissions && sectMissions.length > 0 ? (
                sectMissions.map((mission) => {
                  const rankMap: Record<string, number> = {
                    "NGOAI_MON": 1,
                    "NOI_MON": 2,
                    "TINH_ANH": 3,
                    "CHAN_TRUYEN": 4,
                    "TRUONG_LAO": 5,
                    "HO_PHAP": 6,
                  };
                  const currentMemberRankVal = rankMap[member?.member_rank || "NGOAI_MON"] || 1;
                  const reqRankVal = rankMap[mission.required_member_rank] || 1;
                  const rankTooLow = currentMemberRankVal < reqRankVal;

                  return (
                    <div
                      key={mission.id}
                      className={`bg-zinc-900/20 border ${rankTooLow ? "border-zinc-950 opacity-60" : "border-zinc-900 hover:border-zinc-800"
                        } rounded-2xl p-4 flex flex-col justify-between gap-4 transition-all duration-300`}
                    >
                      <div>
                        <div className="flex justify-between items-start mb-2">
                          <span className="text-[9px] font-mono px-2 py-0.5 rounded bg-zinc-950 border border-zinc-800 text-zinc-400">
                            {getMissionTypeName(mission.mission_type)}
                          </span>
                          <span className={`text-[9px] font-mono px-2 py-0.5 rounded border ${getRankColor(mission.required_member_rank)}`}>
                            Yêu cầu: {getRankName(mission.required_member_rank)}
                          </span>
                        </div>
                        <h4 className="text-sm font-serif font-bold text-zinc-100 mb-1 flex items-start justify-between gap-2">
                          <span>{mission.title}</span>
                          {(member?.member_rank === "CHUONG_MON" || member?.member_rank === "TRUONG_LAO" || member?.member_rank === "HO_PHAP") && (
                            <button
                              onClick={async () => {
                                if (!(await showConfirm(`Đạo hữu có chắc chắn muốn xóa nhiệm vụ "${mission.title}"? Đệ tử đang thực hiện nhiệm vụ này sẽ bị hủy bỏ!`))) return;
                                await deleteSectMission(mission.id);
                              }}
                              disabled={actionLoading}
                              className="shrink-0 p-1.5 rounded-lg border border-zinc-800 text-zinc-500 hover:text-red-400 hover:border-red-500/40 bg-zinc-950/60 transition-colors cursor-pointer"
                              title="Xóa nhiệm vụ"
                            >
                              <Trash2 className="w-3.5 h-3.5" />
                            </button>
                          )}
                        </h4>
                        <p className="text-xs text-zinc-500 leading-relaxed">{mission.description}</p>
                        {formatMissionTime(mission.expires_at) ? (
                          <span className="mt-2 inline-flex items-center gap-1 text-[9px] font-mono text-amber-400/80">
                            <Hourglass className="w-3 h-3" />
                            Hết hạn sau: {formatMissionTime(mission.expires_at)}
                          </span>
                        ) : (
                          <span className="mt-2 inline-flex items-center gap-1 text-[9px] font-mono text-zinc-600">
                            <Hourglass className="w-3 h-3" />
                            Không hết hạn
                          </span>
                        )}
                      </div>

                      <div className="pt-3 border-t border-zinc-950 flex items-center justify-between mt-auto">
                        <span className="text-[10px] font-mono text-yellow-400 font-semibold">
                          +{mission.reward_contribution} Pts
                        </span>

                        {rankTooLow ? (
                          <span className="text-[10px] text-red-400 flex items-center gap-1 font-mono">
                            <Lock className="w-3 h-3" /> Cấp bậc chưa đủ
                          </span>
                        ) : (
                          <button
                            onClick={async () => {
                              await acceptSectMission(mission.id);
                            }}
                            disabled={mySectStatus.has_active || actionLoading}
                            className="px-3 py-1.5 bg-blue-600 hover:bg-blue-500 disabled:bg-zinc-950 disabled:border-zinc-900 disabled:text-zinc-600 border border-transparent disabled:opacity-50 text-white font-semibold text-xs rounded-xl shadow-md transition-colors cursor-pointer"
                          >
                            Nhận Nhiệm Vụ
                          </button>
                        )}
                      </div>
                    </div>
                  );
                })
              ) : (
                <div className="col-span-2 text-center py-8 text-zinc-600 text-xs font-mono">Không tìm thấy nhiệm vụ nào</div>
              )}
            </div>
          </div>
        </div>
      )}

      {activeTab === "vault" && (
        <div className="bg-zinc-950/40 border border-zinc-900 rounded-3xl p-5 backdrop-blur-md flex flex-col gap-6 animate-slide-up">
          {/* Header Vault & Sub-tabs */}
          <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 border-b border-zinc-900 pb-4">
            <div>
              <h3 className="text-base font-serif font-bold text-zinc-200">
                Tàng Kinh Các & Bảo Khố Tông Môn
              </h3>
              <p className="text-xs text-zinc-500 mt-0.5">
                Dùng điểm cống hiến môn phái để đổi lấy thiên tài địa bảo, cổ trùng hay công thức bí truyền.
              </p>
            </div>

            {/* Sub Tabs */}
            <div className="flex bg-zinc-900/50 p-1 rounded-xl border border-zinc-800/60 self-start sm:self-auto">
              <button
                onClick={() => setVaultTab("gu")}
                className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${vaultTab === "gu" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"
                  }`}
              >
                Cổ Trùng
              </button>
              <button
                onClick={() => setVaultTab("material")}
                className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${vaultTab === "material" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"
                  }`}
              >
                Linh Tài
              </button>
              <button
                onClick={() => setVaultTab("recipe")}
                className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${vaultTab === "recipe" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"
                  }`}
              >
                Cổ Phương
              </button>
            </div>
          </div>

          {/* Vault Top Bar: Contribution Point balance and Filter/Search */}
          <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 bg-zinc-900/30 border border-zinc-800/60 rounded-2xl p-3.5 mb-2">
            <div className="flex items-center gap-2">
              <span className="text-xs text-zinc-400 font-serif font-bold">Điểm Cống Hiến:</span>
              <span className="text-sm font-mono font-bold text-yellow-400 flex items-center gap-1">
                <Coins className="w-4 h-4 text-yellow-500" />
                {(member?.contribution_points || 0).toLocaleString()} Pts
              </span>
              <button
                type="button"
                onClick={() => setActiveTab("donate")}
                className="ml-2 text-[10px] font-mono text-blue-400 hover:text-blue-300 underline decoration-dotted cursor-pointer"
              >
                + Dâng nộp thêm
              </button>
            </div>

            <div className="flex items-center gap-2 flex-wrap">
              <input
                type="text"
                value={vaultSearch}
                onChange={(e) => setVaultSearch(e.target.value)}
                placeholder="Tìm kiếm vật phẩm..."
                className="bg-zinc-950 border border-zinc-800 rounded-xl px-3 py-1.5 text-xs font-mono text-zinc-200 focus:outline-none focus:border-blue-500/50 min-w-[160px]"
              />

              {(vaultTab === "gu" || vaultTab === "material") && (
                <div className="flex items-center gap-1 bg-zinc-950/60 p-1 rounded-xl border border-zinc-800 text-[10px] font-mono">
                  {["All", "1", "2", "3", "4", "5", "6+"].map((r) => (
                    <button
                      key={r}
                      type="button"
                      onClick={() => setVaultRankFilter(r)}
                      className={`px-2 py-0.5 rounded-lg transition-colors cursor-pointer ${
                        vaultRankFilter === r
                          ? "bg-blue-600 text-white font-bold"
                          : "text-zinc-400 hover:text-zinc-200"
                      }`}
                    >
                      {r === "All" ? "Tất cả" : r === "6+" ? "Tiên Cổ" : `R${r}`}
                    </button>
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* Vault Inventory Render */}
          <div>
            {vaultTab === "gu" && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {sect?.vault_gu && (sect.vault_gu as ApiResponseData[]).length > 0 ? (
                  (() => {
                    // Deduplicate by id + rank (merge quantities), then sort by rank ascending
                    const merged: Record<string, ApiResponseData> = {};
                    for (const vg of sect.vault_gu as ApiResponseData[]) {
                      const key = `${vg.id}_${vg.rank || 0}`;
                      if (!merged[key]) {
                        merged[key] = { ...vg };
                      } else {
                        merged[key].quantity = (merged[key].quantity || 0) + (vg.quantity || 0);
                      }
                    }
                    const sorted = Object.values(merged).sort((a: ApiResponseData, b: ApiResponseData) => {
                      const rankA = a.rank || guDictionary[a.id]?.rank || 1;
                      const rankB = b.rank || guDictionary[b.id]?.rank || 1;
                      return rankA - rankB;
                    }).filter((vg: ApiResponseData) => {
                      if (vg.quantity !== undefined && vg.quantity <= 0) return false;
                      const spec = guDictionary[vg.id];
                      const displayRank = vg.rank || spec?.rank || 1;
                      if (vaultRankFilter !== "All") {
                        if (vaultRankFilter === "6+") {
                          if (displayRank < 6) return false;
                        } else {
                          if (displayRank !== parseInt(vaultRankFilter)) return false;
                        }
                      }
                      if (!vaultSearch) return true;
                      const name = spec?.name || vg.id;
                      return name.toLowerCase().includes(vaultSearch.toLowerCase());
                    });
                    return sorted.map((vg: ApiResponseData) => {
                      const spec = guDictionary[vg.id];
                      const outOfStock = vg.quantity !== undefined && vg.quantity <= 0;
                      const uKey = `${vg.id}_${vg.rank || 0}`;
                      const qtyKey = `gu-${uKey}`;
                      const qty = getBuyQty(qtyKey);
                      const isImmortal = (vg.rank && vg.rank >= 6) || (spec?.rank && spec.rank >= 6);
                      const maxQty = isImmortal ? 1 : (vg.quantity !== undefined ? vg.quantity : 1);
                      const totalCost = vg.cost * qty;
                      const memberPts = member?.contribution_points || 0;
                      const maxAffordable = Math.floor(memberPts / (vg.cost || 1));
                      const unaffordable = memberPts < totalCost;
                      const displayRank = vg.rank || spec?.rank || 1;
                      const ownedCount = player?.gu_inventory?.filter((g: ApiResponseData) => g.gu_id === vg.id && g.rank === displayRank && g.status !== "DEAD").length || 0;
                      const vgMaxHunger = spec?.max_hunger || 100;
                      return (
                        <div
                          key={uKey}
                          className="bg-zinc-900/20 border border-zinc-900 hover:border-zinc-800/80 rounded-2xl p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-4 transition-all duration-300"
                        >
                          <div className="flex flex-col min-w-0">
                            <h4 className="text-sm font-serif font-bold text-zinc-100">
                              {spec?.name || vg.id}
                            </h4>
                            <span className="text-[10px] text-zinc-500 font-mono mt-0.5">
                              Phân loại: {spec?.gu_type || "Cổ"} | Hạng: {displayRank} | Số lượng: {vg.quantity !== undefined ? `${vg.quantity} con` : "Vô hạn"}
                              {player && (
                                <>
                                  {" | "}
                                  Bạn có: <span className="text-emerald-500 font-semibold">{ownedCount} con</span>
                                </>
                              )}
                            </span>
                            {(spec?.hunger_rate ?? 0) > 0 && (
                              <div className="flex items-center gap-1.5 mt-1">
                                <span className="text-[9px] text-zinc-500 font-mono">Độ no khi rút:</span>
                                <span className="text-[9px] font-mono text-amber-400 font-semibold">
                                  1/{vgMaxHunger}
                                </span>
                              </div>
                            )}
                            <p className="text-xs text-zinc-500 leading-snug line-clamp-2 mt-1.5">
                              {spec?.description || "Không có mô tả bí thuật..."}
                            </p>
                          </div>

                          <div className="flex flex-row items-center justify-between w-full sm:w-auto sm:flex-col sm:items-end gap-3 shrink-0 pt-3 sm:pt-0 border-t border-zinc-800/40 sm:border-0">
                            {!outOfStock && (
                              <div className="flex items-center gap-2">
                                {renderQtyStepper(qtyKey, maxQty, maxAffordable)}
                                <div className="text-right">
                                  <span className="text-[9px] text-zinc-500 font-mono block">Cống hiến cần</span>
                                  <span className="text-xs font-mono font-bold text-yellow-500">
                                    {totalCost.toLocaleString()} Pts
                                  </span>
                                  {member && (
                                    <span className={`text-[9px] font-mono block ${unaffordable ? "text-red-400 font-semibold" : "text-zinc-500"}`}>
                                      {unaffordable ? `(Thiếu ${(totalCost - memberPts).toLocaleString()} Pts)` : `(Còn ${(memberPts - totalCost).toLocaleString()} Pts)`}
                                    </span>
                                  )}
                                </div>
                              </div>
                            )}
                            <button
                              onClick={async () => {
                                await buyFromSectVault("gu", vg.id, qty, vg.rank);
                              }}
                              disabled={unaffordable || outOfStock || actionLoading}
                              className="px-3 py-1.5 bg-blue-600 hover:bg-blue-500 disabled:bg-zinc-950 disabled:border-zinc-900 disabled:text-zinc-600 disabled:opacity-50 text-white font-semibold text-xs rounded-xl shadow-md transition-colors cursor-pointer"
                            >
                              {outOfStock ? "Hết Hàng" : qty > 1 ? `Đổi ${qty} Cổ` : "Đổi Cổ"}
                            </button>
                          </div>
                        </div>
                      );
                    });
                  })()
                ) : (
                  <div className="col-span-2 text-center py-6 text-zinc-600 text-xs font-mono">Bảo khố hiện tại không còn Cổ trùng nào bán</div>
                )}
              </div>
            )}

            {vaultTab === "material" && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {sect?.vault_materials && Object.entries(sect.vault_materials).length > 0 ? (
                  Object.entries(sect.vault_materials)
                  .filter(([matId, qty]: ApiResponseData) => {
                    if (qty <= 0) return false;
                    const spec = itemDictionary[matId];
                    const rank = spec?.rank || 1;
                    if (vaultRankFilter !== "All") {
                      if (vaultRankFilter === "6+") {
                        if (rank < 6) return false;
                      } else {
                        if (rank !== parseInt(vaultRankFilter)) return false;
                      }
                    }
                    if (!vaultSearch) return true;
                    const name = spec?.name || matId;
                    return name.toLowerCase().includes(vaultSearch.toLowerCase());
                  })
                  .map(([matId, qty]) => {
                    const spec = itemDictionary[matId];
                    const cost = getVaultCost(getItemBasePrice(matId));
                    const qtyKey = `material-${matId}`;
                    const buyQty = getBuyQty(qtyKey);
                    const totalCost = cost * buyQty;
                    const memberPts = member?.contribution_points || 0;
                    const maxAffordable = Math.floor(memberPts / (cost || 1));
                    const unaffordable = memberPts < totalCost;

                    const ownedCount = player?.materials_inventory?.[matId] || 0;

                    return (
                      <div
                        key={matId}
                        className="bg-zinc-900/20 border border-zinc-900 hover:border-zinc-800/80 rounded-2xl p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-4 transition-all duration-300"
                      >
                        <div className="flex flex-col min-w-0">
                          <h4 className="text-sm font-serif font-bold text-zinc-100">
                            {spec?.name || matId}
                          </h4>
                          <span className="text-[10px] text-zinc-500 font-mono mt-0.5">
                            Còn lại bảo khố: {qty}x
                            {player && (
                              <>
                                {" | "}
                                Bạn có: <span className="text-emerald-500 font-semibold">{ownedCount}x</span>
                              </>
                            )}
                          </span>
                          <p className="text-xs text-zinc-500 leading-snug line-clamp-2 mt-1.5">
                            {spec?.description || "Nguyên liệu chế tác / luyện cổ trùng..."}
                          </p>
                        </div>

                        <div className="flex flex-row items-center justify-between w-full sm:w-auto sm:flex-col sm:items-end gap-3 shrink-0 pt-3 sm:pt-0 border-t border-zinc-800/40 sm:border-0">
                          {qty > 0 && (
                            <div className="flex items-center gap-2">
                              {renderQtyStepper(qtyKey, qty, maxAffordable)}
                              <div className="text-right">
                                <span className="text-[9px] text-zinc-500 font-mono block">Cống hiến cần</span>
                                <span className="text-xs font-mono font-bold text-yellow-500">
                                  {totalCost.toLocaleString()} Pts
                                </span>
                                {member && (
                                  <span className={`text-[9px] font-mono block ${unaffordable ? "text-red-400 font-semibold" : "text-zinc-500"}`}>
                                    {unaffordable ? `(Thiếu ${(totalCost - memberPts).toLocaleString()} Pts)` : `(Còn ${(memberPts - totalCost).toLocaleString()} Pts)`}
                                  </span>
                                )}
                              </div>
                            </div>
                          )}
                          <button
                            onClick={async () => {
                              await buyFromSectVault("material", matId, buyQty);
                            }}
                            disabled={unaffordable || qty <= 0 || actionLoading}
                            className="px-3 py-1.5 bg-blue-600 hover:bg-blue-500 disabled:bg-zinc-950 disabled:border-zinc-900 disabled:text-zinc-600 disabled:opacity-50 text-white font-semibold text-xs rounded-xl shadow-md transition-colors cursor-pointer"
                          >
                            {qty <= 0 ? "Hết Hàng" : buyQty > 1 ? `Đổi ${buyQty} Linh Tài` : "Đổi Linh Tài"}
                          </button>
                        </div>
                      </div>
                    );
                  })
                ) : (
                  <div className="col-span-2 text-center py-6 text-zinc-600 text-xs font-mono">Linh tài tại bảo khố đã cạn kiệt</div>
                )}
              </div>
            )}

            {vaultTab === "recipe" && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {sect?.vault_recipes && sect.vault_recipes.length > 0 ? (
                  sect.vault_recipes
                  .filter((vaultRecipe) => {
                    if (!vaultSearch) return true;
                    const name = formatRecipeName(vaultRecipe.recipe_id);
                    return name.toLowerCase().includes(vaultSearch.toLowerCase());
                  })
                  .map((vaultRecipe) => {
                    const recipeId = vaultRecipe.recipe_id;
                    const recipeInfo = recipes?.find(r => r.id === recipeId);
                    const targetGuSpec = recipeInfo ? guDictionary[recipeInfo.target_gu_id] : null;
                    const targetGuName = targetGuSpec?.name || recipeId.replace("recipe_", "").toUpperCase();
                    const cost = getRecipeCost(targetGuSpec?.rank || 1);
                    const unaffordable = (member?.contribution_points || 0) < cost;

                    const isUnlocked = player?.unlocked_recipes?.some(ur => ur.recipe_id === recipeId) || false;

                    return (
                      <div
                        key={recipeId}
                        className="bg-zinc-900/20 border border-zinc-900 hover:border-zinc-800/80 rounded-2xl p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-4 transition-all duration-300"
                      >
                        <div className="flex flex-col min-w-0">
                          <h4 className="text-sm font-serif font-bold text-zinc-100">
                            {formatRecipeName(recipeId)}
                          </h4>
                          <span className="text-[10px] text-zinc-500 font-mono mt-0.5">
                            Độ hoàn thiện: {vaultRecipe.completion_rate.toFixed(0)}%{" | "}
                            Bao gồm: Nguyên liệu & Tỷ lệ luyện chế thành công môn phái đúc rút.
                            {player && (
                              <>
                                {" | "}
                                Trạng thái: <span className={isUnlocked ? "text-emerald-500 font-semibold" : "text-zinc-400"}>
                                  {isUnlocked ? "Đã học" : "Chưa học"}
                                </span>
                              </>
                            )}
                          </span>
                          <p className="text-xs text-zinc-500 leading-snug line-clamp-2 mt-1.5">
                            Giải khóa công thức hợp thành tại lò luyện Cổ trùng của bạn.
                          </p>
                        </div>

                        <div className="flex flex-row items-center justify-between w-full sm:w-auto sm:flex-col sm:items-end gap-3 shrink-0 pt-3 sm:pt-0 border-t border-zinc-800/40 sm:border-0">
                          <div className="text-right">
                            <span className="text-[9px] text-zinc-500 font-mono block">Cống hiến cần</span>
                            <span className="text-xs font-mono font-bold text-yellow-500">{cost.toLocaleString()} Pts</span>
                            {member && !isUnlocked && (
                              <span className={`text-[9px] font-mono block ${unaffordable ? "text-red-400 font-semibold" : "text-zinc-500"}`}>
                                {unaffordable ? `(Thiếu ${(cost - member.contribution_points).toLocaleString()} Pts)` : `(Còn ${(member.contribution_points - cost).toLocaleString()} Pts)`}
                              </span>
                            )}
                          </div>
                          <button
                            onClick={async () => {
                              await buyFromSectVault("recipe", recipeId);
                            }}
                            disabled={unaffordable || isUnlocked || actionLoading}
                            className="px-3 py-1.5 bg-blue-600 hover:bg-blue-500 disabled:bg-zinc-950 disabled:border-zinc-900 disabled:text-zinc-600 disabled:opacity-50 text-white font-semibold text-xs rounded-xl shadow-md transition-colors cursor-pointer"
                          >
                            {isUnlocked ? "Đã Học" : "Học Công Thức"}
                          </button>
                        </div>
                      </div>
                    );
                  })
                ) : (
                  <div className="col-span-2 text-center py-6 text-zinc-600 text-xs font-mono">Tông môn hiện không có Cổ phương truyền thừa nào bán</div>
                )}
              </div>
            )}
          </div>
        </div>
      )}

      {activeTab === "donate" && (
        <div className="bg-zinc-950/40 border border-zinc-900 rounded-3xl p-5 backdrop-blur-md flex flex-col gap-6 animate-slide-up">
          {/* Header & Donate Category Tabs */}
          <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 border-b border-zinc-900 pb-4">
            <div>
              <h3 className="text-base font-serif font-bold text-zinc-200">
                Quyên Góp Cho Môn Phái
              </h3>
              <p className="text-xs text-zinc-500 mt-0.5">
                Đóng góp tài sản cá nhân cho tông môn bồi dưỡng thế lực và nhận về Điểm cống hiến môn phái.
              </p>
            </div>

            <div className="flex bg-zinc-900/50 p-1 rounded-xl border border-zinc-800/60 self-start sm:self-auto">
              <button
                onClick={() => setDonateType("stones")}
                className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${donateType === "stones" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"
                  }`}
              >
                Nguyên Thạch
              </button>
              <button
                onClick={() => setDonateType("materials")}
                className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${donateType === "materials" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"
                  }`}
              >
                Linh Tài
              </button>
              <button
                onClick={() => setDonateType("gu")}
                className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${donateType === "gu" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"
                  }`}
              >
                Cổ Trùng
              </button>
              <button
                onClick={() => setDonateType("recipes")}
                className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${donateType === "recipes" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"
                  }`}
              >
                Bí Phương
              </button>
            </div>
          </div>

          {/* Donate Stone Panel */}
          {donateType === "stones" && (
            <div className="bg-zinc-950/20 border border-zinc-900 rounded-2xl p-6 flex flex-col gap-6 max-w-xl mx-auto w-full shadow-lg relative overflow-hidden">
              <div className="absolute top-0 right-0 w-24 h-24 bg-yellow-500/5 rounded-full blur-xl -z-10" />

              <div className="flex justify-between items-center bg-zinc-900/40 px-4 py-3 rounded-xl border border-zinc-800/80">
                <span className="text-zinc-400 font-serif font-bold text-xs">Tài sản hiện có:</span>
                <span className="font-mono font-bold text-amber-500 flex items-center gap-1 text-sm">
                  <Coins className="w-4 h-4 text-amber-500" />
                  {player.primeval_stones.toLocaleString()} Nguyên Thạch
                </span>
              </div>

              <div className="flex flex-col gap-2">
                <label className="text-xs text-zinc-350 font-serif font-bold">Số lượng dâng nộp</label>
                <div className="relative">
                  <input
                    type="number"
                    value={donateStones}
                    onChange={(e) => setDonateStones(Math.max(1, parseInt(e.target.value) || 0))}
                    className="w-full bg-zinc-950 border border-zinc-800 rounded-xl pl-4 pr-20 py-3 text-sm font-mono text-zinc-100 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500/30"
                  />
                  <button
                    type="button"
                    onClick={() => setDonateStones(player.primeval_stones)}
                    className="absolute right-2.5 top-1/2 -translate-y-1/2 text-[10px] bg-blue-600/10 hover:bg-blue-600/20 text-blue-400 border border-blue-800/30 px-3 py-1.5 rounded-lg transition-all font-semibold font-mono"
                  >
                    TỐI ĐA
                  </button>
                </div>

                {/* Preset shortcuts */}
                <div className="grid grid-cols-4 gap-2 mt-1">
                  {[100, 1000, 5000, 10000].map((preset) => (
                    <button
                      key={preset}
                      type="button"
                      onClick={() => setDonateStones(Math.min(player.primeval_stones, preset))}
                      className="py-1.5 text-[10px] font-mono text-zinc-400 hover:text-zinc-200 bg-zinc-900 border border-zinc-800 hover:border-zinc-700 rounded-lg transition-all"
                    >
                      +{preset.toLocaleString()}
                    </button>
                  ))}
                </div>

                <span className="text-[10px] text-zinc-500 font-mono mt-1 flex items-center gap-1">
                  ℹ️ Tỷ giá quyên góp: 1 Nguyên Thạch = 1 Điểm cống hiến.
                </span>
              </div>

              <button
                onClick={async () => {
                  if (donateStones <= 0 || donateStones > player.primeval_stones) {
                    addToast("Số lượng đá quyên góp không hợp lệ hoặc vượt quá tài sản hiện có!", "error");
                    return;
                  }
                  await donateSectStones(donateStones);
                }}
                disabled={player.primeval_stones <= 0 || actionLoading}
                className="w-full py-3 bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 disabled:opacity-50 text-white rounded-xl font-bold text-sm shadow-md transition-all cursor-pointer flex items-center justify-center gap-2 tracking-wide"
              >
                <Gift className="w-4 h-4" />
                DÂNG NỘP LINH THẠCH (Nhận +{donateStones.toLocaleString()} Pts)
              </button>
            </div>
          )}

          {/* Donate Material Panel */}
          {donateType === "materials" && (() => {
            const availableDonateMatRanks = Array.from(
              new Set(playerMaterials.map(([matId]) => itemDictionary[matId]?.rank || 1))
            ).sort((a, b) => a - b);

            const availableDonateMatPaths = Array.from(
              new Set(playerMaterials.flatMap(([matId]) => parsePaths(itemDictionary[matId]?.path)))
            ).filter(Boolean).sort((a, b) => translatePath(a).localeCompare(translatePath(b), "vi"));

            const filteredMaterials = playerMaterials.filter(([matId]) => {
              const spec = itemDictionary[matId];
              const name = spec?.name || matId;
              const rank = spec?.rank || 1;
              const paths = parsePaths(spec?.path);

              const matchesSearch = !donateSearch || name.toLowerCase().includes(donateSearch.toLowerCase()) || matId.toLowerCase().includes(donateSearch.toLowerCase());
              const matchesRank = donateRankFilter === "All" || rank.toString() === donateRankFilter;
              const matchesPath = donatePathFilter === "All" || paths.includes(donatePathFilter);

              return matchesSearch && matchesRank && matchesPath;
            });

            const selectedCount = Object.keys(selectedDonateMaterials).length;
            const totalGainedPoints = Object.values(selectedDonateMaterials).reduce(
              (sum, item) => sum + item.amountToDonate * item.valuePerItem,
              0
            );

            return (
              <div className="grid grid-cols-1 md:grid-cols-3 gap-6 w-full">
                {/* Left side: Selector List */}
                <div className="md:col-span-2 bg-zinc-950/20 border border-zinc-900 rounded-2xl p-5 flex flex-col gap-4">
                  <div className="flex justify-between items-center border-b border-zinc-900 pb-3">
                    <div>
                      <h4 className="text-sm font-bold text-zinc-200 font-serif">Danh Sách Linh Tài</h4>
                      <p className="text-[10px] text-zinc-500 font-mono mt-0.5">Tích chọn linh tài và số lượng cần quyên hiến.</p>
                    </div>
                    <span className="text-[10px] font-mono text-zinc-400 bg-zinc-900 px-2.5 py-1 rounded-lg border border-zinc-800">
                      Tìm thấy: {filteredMaterials.length} loại
                    </span>
                  </div>

                  {/* Search & Filters */}
                  <div className="flex flex-col sm:flex-row gap-2">
                    <input
                      type="text"
                      value={donateSearch}
                      onChange={(e) => setDonateSearch(e.target.value)}
                      placeholder="Tìm kiếm linh tài..."
                      className="flex-1 bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs font-mono text-zinc-200 focus:outline-none focus:border-blue-500/50"
                    />
                    <div className="flex gap-2 flex-wrap">
                      <select
                        value={donateRankFilter}
                        onChange={(e) => setDonateRankFilter(e.target.value)}
                        className="bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs font-mono text-zinc-300 focus:outline-none focus:border-blue-500/50 cursor-pointer outline-none"
                      >
                        <option value="All">Tất cả Hạng</option>
                        {availableDonateMatRanks.map((rank) => (
                          <option key={rank} value={rank.toString()}>Hạng {rank}</option>
                        ))}
                      </select>
                      <select
                        value={donatePathFilter}
                        onChange={(e) => setDonatePathFilter(e.target.value)}
                        className="bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs font-mono text-zinc-300 focus:outline-none focus:border-blue-500/50 cursor-pointer outline-none"
                      >
                        <option value="All">Tất cả Phái</option>
                        {availableDonateMatPaths.map((path) => (
                          <option key={path} value={path}>{translatePath(path)}</option>
                        ))}
                      </select>
                    </div>
                  </div>

                  {/* Quick select bar */}
                  <div className="flex items-center gap-1.5 flex-wrap">
                    <span className="text-[9px] font-mono text-zinc-500">Chọn nhanh:</span>
                    {availableDonateMatRanks.map((rank) => (
                      <button
                        key={rank}
                        type="button"
                        onClick={() => {
                          setSelectedDonateMaterials(prev => {
                            const next = { ...prev };
                            filteredMaterials.filter(([matId]) => (itemDictionary[matId]?.rank || 1) === rank).forEach(([matId, qty]) => {
                              if (!next[matId]) {
                                const spec = itemDictionary[matId];
                                next[matId] = {
                                  material_id: matId,
                                  name: spec?.name || matId,
                                  owned: qty,
                                  valuePerItem: getMaterialValue(matId),
                                  amountToDonate: qty
                                };
                              } else {
                                next[matId] = { ...next[matId], amountToDonate: next[matId].owned };
                              }
                            });
                            return next;
                          });
                        }}
                        className="px-2 py-0.5 text-[9px] font-mono rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-400 border border-zinc-800 transition-colors cursor-pointer"
                      >
                        Rank {rank}
                      </button>
                    ))}
                    <button
                      type="button"
                      onClick={() => {
                        setSelectedDonateMaterials(prev => {
                          const next = { ...prev };
                          filteredMaterials.forEach(([matId, qty]) => {
                            if (!next[matId]) {
                              const spec = itemDictionary[matId];
                              next[matId] = {
                                material_id: matId,
                                name: spec?.name || matId,
                                owned: qty,
                                valuePerItem: getMaterialValue(matId),
                                amountToDonate: qty
                              };
                            } else {
                              next[matId] = { ...next[matId], amountToDonate: next[matId].owned };
                            }
                          });
                          return next;
                        });
                      }}
                      className="px-2 py-0.5 text-[9px] font-mono rounded bg-blue-950/20 hover:bg-blue-950/30 text-blue-400 border border-blue-900/30 transition-colors cursor-pointer"
                    >
                      Tất cả tab này
                    </button>
                    <button
                      type="button"
                      onClick={() => setSelectedDonateMaterials({})}
                      className="px-2 py-0.5 text-[9px] font-mono rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-500 hover:text-zinc-300 border border-zinc-800 transition-colors cursor-pointer"
                    >
                      Bỏ chọn hết
                    </button>
                  </div>

                  {/* Grid layout for items */}
                  <div className="max-h-[480px] overflow-y-auto pr-1 grid grid-cols-1 sm:grid-cols-2 gap-3 scrollbar-thin">
                    {filteredMaterials.length === 0 ? (
                      <div className="col-span-full text-center py-12 text-xs text-zinc-600 font-mono">Không tìm thấy linh tài nào phù hợp</div>
                    ) : (
                      filteredMaterials.map(([matId, qty]) => {
                        const spec = itemDictionary[matId];
                        const valPerItem = getMaterialValue(matId);
                        const isSelected = !!selectedDonateMaterials[matId];
                        const selectedItem = selectedDonateMaterials[matId];

                        const toggleSelect = () => {
                          setSelectedDonateMaterials(prev => {
                            const next = { ...prev };
                            if (next[matId]) {
                              delete next[matId];
                            } else {
                              next[matId] = {
                                material_id: matId,
                                name: spec?.name || matId,
                                owned: qty,
                                valuePerItem: valPerItem,
                                amountToDonate: 1
                              };
                            }
                            return next;
                          });
                        };

                        const updateAmount = (val: number) => {
                          const amount = Math.max(1, Math.min(qty, val));
                          setSelectedDonateMaterials(prev => ({
                            ...prev,
                            [matId]: {
                              ...prev[matId],
                              amountToDonate: amount
                            }
                          }));
                        };

                        return (
                          <div
                            key={matId}
                            onClick={toggleSelect}
                            className={`bg-zinc-950/40 border rounded-xl p-3 flex flex-col justify-between gap-3.5 transition-all cursor-pointer select-none ${
                              isSelected
                                ? "border-blue-600/50 bg-blue-900/5 shadow-md shadow-blue-950/10"
                                : "border-zinc-900 hover:border-zinc-800"
                            }`}
                          >
                            <div className="flex gap-2.5 min-w-0 items-start">
                              {/* Checkbox */}
                              <div className={`w-4.5 h-4.5 rounded border flex items-center justify-center shrink-0 mt-0.5 transition-colors ${
                                isSelected ? "bg-blue-600 border-blue-500 text-white" : "bg-zinc-900 border-zinc-800"
                              }`}>
                                {isSelected && <Check className="w-3.5 h-3.5 text-white stroke-[3px]" />}
                              </div>

                              <div className="min-w-0">
                                <h5 className="text-xs font-bold text-zinc-200 truncate">{spec?.name || matId}</h5>
                                <p className="text-[10px] text-zinc-500 font-mono mt-1">
                                  Hạng {spec?.rank || 1} • Giá trị: <span className="text-zinc-300 font-semibold">{valPerItem} Pts</span>
                                </p>
                                <p className="text-[10px] text-zinc-400 font-mono mt-0.5">
                                  Đang có: <span className="text-zinc-200 font-semibold">{qty}x</span>
                                </p>
                              </div>
                            </div>

                            {/* Quantity control */}
                            {isSelected && (
                              <div className="flex items-center justify-between border-t border-zinc-900/80 pt-2" onClick={e => e.stopPropagation()}>
                                <span className="text-[10px] text-zinc-500 font-mono">Quyên góp:</span>
                                <div className="flex items-center gap-1 shrink-0">
                                  <button
                                    type="button"
                                    onClick={() => updateAmount(selectedItem.amountToDonate - 1)}
                                    className="w-5 h-5 flex items-center justify-center bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 rounded text-zinc-400 font-bold transition-all text-[10px]"
                                  >
                                    -
                                  </button>
                                  <input
                                    type="number"
                                    min={1}
                                    max={qty}
                                    value={selectedItem.amountToDonate}
                                    onChange={(e) => updateAmount(parseInt(e.target.value, 10) || 1)}
                                    className="w-8 text-center bg-zinc-950 border border-zinc-900 rounded py-0.5 text-[10px] font-mono font-bold text-zinc-200 focus:outline-none"
                                  />
                                  <button
                                    type="button"
                                    onClick={() => updateAmount(selectedItem.amountToDonate + 1)}
                                    className="w-5 h-5 flex items-center justify-center bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 rounded text-zinc-400 font-bold transition-all text-[10px]"
                                  >
                                    +
                                  </button>
                                  <button
                                    type="button"
                                    onClick={() => updateAmount(qty)}
                                    className="ml-1 px-1.5 py-0.5 bg-blue-950 text-blue-400 border border-blue-900/30 rounded text-[9px] font-mono transition-all font-semibold"
                                  >
                                    Max
                                  </button>
                                </div>
                              </div>
                            )}
                          </div>
                        );
                      })
                    )}
                  </div>
                </div>

                {/* Right side: Summary & Submit Action */}
                <div className="bg-zinc-900/10 border border-zinc-900 rounded-2xl p-5 flex flex-col gap-4 h-fit md:sticky md:top-4">
                  <h4 className="text-xs font-bold text-zinc-350 font-serif border-b border-zinc-900 pb-2.5">
                    Tóm Tắt Quyên Góp ({selectedCount} loại)
                  </h4>

                  {selectedCount === 0 ? (
                    <div className="flex flex-col items-center justify-center py-10 text-center gap-2">
                      <Package className="w-8 h-8 text-zinc-700 stroke-[1.5]" />
                      <span className="text-[11px] text-zinc-500 font-mono">
                        Chọn linh tài từ danh sách để dâng nộp.
                      </span>
                    </div>
                  ) : (
                    <>
                      {/* Selected Items Itemized List */}
                      <div className="max-h-48 overflow-y-auto space-y-2 pr-1 scrollbar-thin">
                        {Object.values(selectedDonateMaterials).map((item) => (
                          <div key={item.material_id} className="flex justify-between items-center text-[10px] font-mono bg-zinc-950/20 border border-zinc-950 rounded-lg p-2">
                            <span className="text-zinc-300 truncate max-w-[120px]">{item.name}</span>
                            <span className="text-zinc-500">
                              x{item.amountToDonate} → <span className="text-yellow-500/90 font-bold">+{item.amountToDonate * item.valuePerItem} Pts</span>
                            </span>
                          </div>
                        ))}
                      </div>

                      <div className="border-t border-zinc-900 pt-3.5 space-y-3.5">
                        <div className="flex justify-between items-center text-xs font-mono">
                          <span className="text-zinc-400 font-serif font-bold">Tổng cống hiến:</span>
                          <span className="text-yellow-500 font-bold text-sm">+{totalGainedPoints} Pts</span>
                        </div>
                        <button
                          onClick={async () => {
                            const materialsPayload = Object.values(selectedDonateMaterials).map(item => ({
                              material_id: item.material_id,
                              amount: item.amountToDonate
                            }));
                            const ok = await donateSectBulk({ materials: materialsPayload, gus: [], recipes: [] });
                            if (ok) setSelectedDonateMaterials({});
                          }}
                          disabled={actionLoading}
                          className="w-full py-3 bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 disabled:opacity-50 text-white rounded-xl font-semibold text-sm shadow-md transition-all cursor-pointer flex items-center justify-center gap-1.5"
                        >
                          <Gift className="w-4 h-4" />
                          QUYÊN GÓP HÀNG LOẠT
                        </button>
                      </div>
                    </>
                  )}
                </div>
              </div>
            );
          })()}

          {/* Donate Gu Panel */}
          {donateType === "recipes" && (() => {
            const availableRecipes = (player?.unlocked_recipes || [])
              .map((ur) => {
                const info = recipes.find((r) => r.id === ur.recipe_id);
                const name = formatRecipeName(ur.recipe_id);
                const rank = info?.target_rank || 1;
                const value = getRecipeCost(rank);
                return {
                  recipe_id: ur.recipe_id,
                  name,
                  completion_rate: ur.completion_rate,
                  value,
                };
              })
              .filter((r) => {
                const matchesSearch = !donateSearch || r.name.toLowerCase().includes(donateSearch.toLowerCase()) || r.recipe_id.toLowerCase().includes(donateSearch.toLowerCase());
                const matchesRank = donateRankFilter === "All" || r.value.toString() === donateRankFilter;
                return matchesSearch && matchesRank;
              });

            const selectedCount = Object.keys(selectedDonateRecipes).length;
            const totalGainedPoints = Object.values(selectedDonateRecipes).reduce(
              (sum, item) => sum + item.value,
              0
            );

            return (
              <div className="grid grid-cols-1 md:grid-cols-3 gap-6 w-full">
                {/* Left side: Selector List */}
                <div className="md:col-span-2 bg-zinc-950/20 border border-zinc-900 rounded-2xl p-5 flex flex-col gap-4">
                  <div className="flex justify-between items-center border-b border-zinc-900 pb-3">
                    <div>
                      <h4 className="text-sm font-bold text-zinc-200 font-serif">Danh Sách Bí Phương</h4>
                      <p className="text-[10px] text-zinc-500 font-mono mt-0.5">Tích chọn Bí phương muốn hiến tế cho tông môn.</p>
                    </div>
                    <span className="text-[10px] font-mono text-zinc-400 bg-zinc-900 px-2.5 py-1 rounded-lg border border-zinc-800">
                      Có thể quyên hiến: {availableRecipes.length} bản
                    </span>
                  </div>

                  {/* Search */}
                  <div className="flex flex-col sm:flex-row gap-2">
                    <input
                      type="text"
                      value={donateSearch}
                      onChange={(e) => setDonateSearch(e.target.value)}
                      placeholder="Tìm kiếm bí phương..."
                      className="flex-1 bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs font-mono text-zinc-200 focus:outline-none focus:border-blue-500/50"
                    />
                  </div>

                  {/* Quick select bar */}
                  <div className="flex items-center gap-1.5 flex-wrap">
                    <span className="text-[9px] font-mono text-zinc-500">Chọn nhanh:</span>
                    <button
                      type="button"
                      onClick={() => {
                        setSelectedDonateRecipes(prev => {
                          const next = { ...prev };
                          availableRecipes.forEach(r => {
                            if (!next[r.recipe_id]) {
                              next[r.recipe_id] = {
                                recipe_id: r.recipe_id,
                                name: r.name,
                                completion_rate: r.completion_rate,
                                value: r.value
                              };
                            }
                          });
                          return next;
                        });
                      }}
                      className="px-2 py-0.5 text-[9px] font-mono rounded bg-blue-950/20 hover:bg-blue-950/30 text-blue-400 border border-blue-900/30 transition-colors cursor-pointer"
                    >
                      Tất cả tab này
                    </button>
                    <button
                      type="button"
                      onClick={() => setSelectedDonateRecipes({})}
                      className="px-2 py-0.5 text-[9px] font-mono rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-500 hover:text-zinc-300 border border-zinc-800 transition-colors cursor-pointer"
                    >
                      Bỏ chọn hết
                    </button>
                  </div>

                  {/* Recipe List */}
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 max-h-96 overflow-y-auto pr-1 scrollbar-thin">
                    {availableRecipes.length === 0 ? (
                      <p className="text-xs text-zinc-600 italic text-center py-4 col-span-full">Không có Bí phương nào có thể quyên hiến</p>
                    ) : (
                      availableRecipes.map((r) => {
                        const isSelected = !!selectedDonateRecipes[r.recipe_id];
                        return (
                          <button
                            key={r.recipe_id}
                            onClick={() => {
                              setSelectedDonateRecipes((prev) => {
                                const next = { ...prev };
                                if (isSelected) {
                                  delete next[r.recipe_id];
                                } else {
                                  next[r.recipe_id] = r;
                                }
                                return next;
                              });
                            }}
                            className={`text-left p-3 rounded-xl border text-xs font-mono flex justify-between items-center transition-all cursor-pointer ${
                              isSelected
                                ? "bg-pink-950/20 border-pink-500/40 text-pink-300 shadow-md shadow-pink-950/15"
                                : "bg-zinc-950 border-zinc-900 hover:border-zinc-850 text-zinc-300"
                            }`}
                          >
                            <div className="flex flex-col min-w-0">
                              <span className="font-bold truncate">{r.name}</span>
                              <span className="text-[10px] text-zinc-500">{r.recipe_id.replace("recipe_", "").toUpperCase()}</span>
                            </div>
                            <div className="text-right shrink-0">
                              <span className="text-[10px] block text-zinc-500">{r.completion_rate.toFixed(0)}%</span>
                              <span className="text-[10px] text-yellow-500 font-bold">+{r.value} Pts</span>
                            </div>
                          </button>
                        );
                      })
                    )}
                  </div>
                </div>

                {/* Right side: Summary & Submit Action */}
                <div className="bg-zinc-900/10 border border-zinc-900 rounded-2xl p-5 flex flex-col gap-4 h-fit md:sticky md:top-4">
                  <h4 className="text-xs font-bold text-zinc-350 font-serif border-b border-zinc-900 pb-2.5">
                    Tóm Tắt Quyên Góp ({selectedCount} bản)
                  </h4>

                  {selectedCount === 0 ? (
                    <div className="flex flex-col items-center justify-center py-10 text-center gap-2">
                      <BookOpen className="w-8 h-8 text-zinc-700 stroke-[1.5]" />
                      <span className="text-[11px] text-zinc-500 font-mono">
                        Chọn Bí phương để hiến tế tông môn.
                      </span>
                    </div>
                  ) : (
                    <>
                      <div className="max-h-48 overflow-y-auto space-y-2 pr-1 scrollbar-thin">
                        {Object.values(selectedDonateRecipes).map((item) => (
                          <div key={item.recipe_id} className="flex justify-between items-center text-[10px] font-mono bg-zinc-950/20 border border-zinc-950 rounded-lg p-2">
                            <span className="text-zinc-300 truncate max-w-[120px]">{item.name}</span>
                            <span className="text-zinc-500">
                              {item.completion_rate.toFixed(0)}% → <span className="text-yellow-500/90 font-bold">+{item.value} Pts</span>
                            </span>
                          </div>
                        ))}
                      </div>

                      <div className="border-t border-zinc-900 pt-3.5 space-y-3.5">
                        <div className="flex justify-between items-center text-xs font-mono">
                          <span className="text-zinc-400 font-serif font-bold">Tổng cống hiến:</span>
                          <span className="text-yellow-500 font-bold text-sm">+{totalGainedPoints} Pts</span>
                        </div>
                        <button
                          onClick={async () => {
                            const recipePayload = Object.values(selectedDonateRecipes).map((r) => ({
                              recipe_id: r.recipe_id,
                            }));
                            if (recipePayload.length === 0) return;

                            const ok = await donateSectBulk({
                              materials: [],
                              gus: [],
                              recipes: recipePayload,
                            });

                            if (ok) setSelectedDonateRecipes({});
                          }}
                          disabled={actionLoading}
                          className="w-full py-3 bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 disabled:opacity-50 text-white rounded-xl font-semibold text-sm shadow-md transition-all cursor-pointer flex items-center justify-center gap-1.5"
                        >
                          <Gift className="w-4 h-4" />
                          HIẾN TẾ BÍ PHƯƠNG
                        </button>
                      </div>
                    </>
                  )}
                </div>
              </div>
            );
          })()}

          {donateType === "gu" && (() => {
            // Group by gu_id and rank
            const groupedGusMap: Record<string, {
              gu_id: string;
              rank: number;
              name: string;
              value: number;
              instances: GuInstance[];
            }> = {};

            playerDonatableGus.forEach((g) => {
              const spec = guSpecFor(g.gu_id, g.rank);
              const rank = g.rank || spec?.rank || 1;
              const name = spec?.name || g.gu_id;
              const val = getGuValue(g.gu_id);
              const key = `${g.gu_id}_rank_${rank}`;

              if (!groupedGusMap[key]) {
                groupedGusMap[key] = {
                  gu_id: g.gu_id,
                  rank,
                  name,
                  value: val,
                  instances: []
                };
              }
              groupedGusMap[key].instances.push(g);
            });

            const groupedGusList = Object.values(groupedGusMap);

            const availableDonateGuRanks = Array.from(
              new Set(groupedGusList.map((g) => g.rank))
            ).sort((a, b) => a - b);

            const availableDonateGuPaths = Array.from(
              new Set(groupedGusList.flatMap((g) => parsePaths(guSpecFor(g.gu_id, g.rank)?.path)))
            ).filter(Boolean).sort((a, b) => translatePath(a).localeCompare(translatePath(b), "vi"));

            const filteredGus = groupedGusList.filter((g) => {
              const matchesSearch = !donateSearch || g.name.toLowerCase().includes(donateSearch.toLowerCase()) || g.gu_id.toLowerCase().includes(donateSearch.toLowerCase());
              const matchesRank = donateRankFilter === "All" || g.rank.toString() === donateRankFilter;

              const spec = guSpecFor(g.gu_id, g.rank);
              const paths = parsePaths(spec?.path);
              const matchesPath = donatePathFilter === "All" || paths.includes(donatePathFilter);

              return matchesSearch && matchesRank && matchesPath;
            });

            const selectedCount = Object.keys(selectedDonateGus).length;
            const totalGainedPoints = Object.values(selectedDonateGus).reduce(
              (sum, item) => sum + item.amountToDonate * item.valuePerItem,
              0
            );

            return (
              <div className="grid grid-cols-1 md:grid-cols-3 gap-6 w-full">
                {/* Left side: Selector List */}
                <div className="md:col-span-2 bg-zinc-950/20 border border-zinc-900 rounded-2xl p-5 flex flex-col gap-4">
                  <div className="flex justify-between items-center border-b border-zinc-900 pb-3">
                    <div>
                      <h4 className="text-sm font-bold text-zinc-200 font-serif">Danh Sách Cổ Trùng</h4>
                      <p className="text-[10px] text-zinc-500 font-mono mt-0.5">Tích chọn các Cổ trùng muốn hiến tế cho tông môn.</p>
                    </div>
                    <span className="text-[10px] font-mono text-zinc-400 bg-zinc-900 px-2.5 py-1 rounded-lg border border-zinc-800">
                      Có thể quyên hiến: {playerDonatableGus.length} con
                    </span>
                  </div>

                  {/* Search & Filters */}
                  <div className="flex flex-col sm:flex-row gap-2">
                    <input
                      type="text"
                      value={donateSearch}
                      onChange={(e) => setDonateSearch(e.target.value)}
                      placeholder="Tìm kiếm cổ trùng..."
                      className="flex-1 bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs font-mono text-zinc-200 focus:outline-none focus:border-blue-500/50"
                    />
                    <div className="flex gap-2">
                      <select
                        value={donateRankFilter}
                        onChange={(e) => setDonateRankFilter(e.target.value)}
                        className="bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs font-mono text-zinc-300 focus:outline-none focus:border-blue-500/50 cursor-pointer outline-none"
                      >
                        <option value="All">Tất cả Hạng</option>
                        {availableDonateGuRanks.map((rank) => (
                          <option key={rank} value={rank.toString()}>Hạng {rank}</option>
                        ))}
                      </select>
                      <select
                        value={donatePathFilter}
                        onChange={(e) => setDonatePathFilter(e.target.value)}
                        className="bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs font-mono text-zinc-300 focus:outline-none focus:border-blue-500/50 cursor-pointer outline-none"
                      >
                        <option value="All">Tất cả Phái</option>
                        {availableDonateGuPaths.map((path) => (
                          <option key={path} value={path}>{translatePath(path)}</option>
                        ))}
                      </select>
                    </div>
                  </div>

                  {/* Quick select bar */}
                  <div className="flex items-center gap-1.5 flex-wrap">
                    <span className="text-[9px] font-mono text-zinc-500">Chọn nhanh:</span>
                    {availableDonateGuRanks.map((rank) => (
                      <button
                        key={rank}
                        type="button"
                        onClick={() => {
                          setSelectedDonateGus(prev => {
                            const next = { ...prev };
                            filteredGus.filter(g => g.rank === rank).forEach(g => {
                              const key = `${g.gu_id}_rank_${g.rank}`;
                              if (!next[key]) {
                                next[key] = {
                                  key,
                                  gu_id: g.gu_id,
                                  rank: g.rank,
                                  name: g.name,
                                  valuePerItem: g.value,
                                  ownedCount: g.instances.length,
                                  amountToDonate: g.instances.length,
                                  instances: g.instances
                                };
                              } else {
                                next[key] = { ...next[key], amountToDonate: next[key].ownedCount };
                              }
                            });
                            return next;
                          });
                        }}
                        className="px-2 py-0.5 text-[9px] font-mono rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-400 border border-zinc-800 transition-colors cursor-pointer"
                      >
                        Rank {rank}
                      </button>
                    ))}
                    <button
                      type="button"
                      onClick={() => {
                        setSelectedDonateGus(prev => {
                          const next = { ...prev };
                          filteredGus.forEach(g => {
                            const key = `${g.gu_id}_rank_${g.rank}`;
                            if (!next[key]) {
                              next[key] = {
                                key,
                                gu_id: g.gu_id,
                                rank: g.rank,
                                name: g.name,
                                valuePerItem: g.value,
                                ownedCount: g.instances.length,
                                amountToDonate: g.instances.length,
                                instances: g.instances
                              };
                            } else {
                              next[key] = { ...next[key], amountToDonate: next[key].ownedCount };
                            }
                          });
                          return next;
                        });
                      }}
                      className="px-2 py-0.5 text-[9px] font-mono rounded bg-blue-950/20 hover:bg-blue-950/30 text-blue-400 border border-blue-900/30 transition-colors cursor-pointer"
                    >
                      Tất cả tab này
                    </button>
                    <button
                      type="button"
                      onClick={() => setSelectedDonateGus({})}
                      className="px-2 py-0.5 text-[9px] font-mono rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-500 hover:text-zinc-300 border border-zinc-800 transition-colors cursor-pointer"
                    >
                      Bỏ chọn hết
                    </button>
                  </div>

                  {/* Grid layout for Gu items */}
                  <div className="max-h-[480px] overflow-y-auto pr-1 grid grid-cols-1 sm:grid-cols-2 gap-3 scrollbar-thin">
                    {filteredGus.length === 0 ? (
                      <div className="col-span-full text-center py-12 text-xs text-zinc-600 font-mono">Không tìm thấy cổ trùng phù hợp</div>
                    ) : (
                      filteredGus.map((g) => {
                        const spec = guSpecFor(g.gu_id, g.rank);
                        const key = `${g.gu_id}_rank_${g.rank}`;
                        const isSelected = !!selectedDonateGus[key];
                        const selectedItem = selectedDonateGus[key];

                        const toggleSelect = () => {
                          setSelectedDonateGus(prev => {
                            const next = { ...prev };
                            if (next[key]) {
                              delete next[key];
                            } else {
                              next[key] = {
                                key,
                                gu_id: g.gu_id,
                                rank: g.rank,
                                name: g.name,
                                valuePerItem: g.value,
                                ownedCount: g.instances.length,
                                amountToDonate: 1,
                                instances: g.instances
                              };
                            }
                            return next;
                          });
                        };

                        const updateAmount = (val: number) => {
                          const amount = Math.max(1, Math.min(g.instances.length, val));
                          setSelectedDonateGus(prev => ({
                            ...prev,
                            [key]: {
                              ...prev[key],
                              amountToDonate: amount
                            }
                          }));
                        };

                        return (
                          <div
                            key={key}
                            onClick={toggleSelect}
                            className={`bg-zinc-950/40 border rounded-xl p-3 flex flex-col justify-between gap-3.5 transition-all cursor-pointer select-none ${
                              isSelected
                                ? "border-blue-600/50 bg-blue-900/5 shadow-md shadow-blue-950/10"
                                : "border-zinc-900 hover:border-zinc-800"
                            }`}
                          >
                            <div className="flex items-start gap-2.5 min-w-0 flex-1 pr-2">
                              {/* Checkbox */}
                              <div className={`w-4.5 h-4.5 rounded border flex items-center justify-center shrink-0 mt-0.5 transition-colors ${
                                isSelected ? "bg-blue-600 border-blue-500 text-white" : "bg-zinc-900 border-zinc-800"
                              }`}>
                                {isSelected && <Check className="w-3.5 h-3.5 text-white stroke-[3px]" />}
                              </div>

                              <div className="min-w-0">
                                <h5 className="text-xs font-bold text-zinc-200 truncate">{g.name}</h5>
                                <p className="text-[10px] text-zinc-500 font-mono mt-1">
                                  Hạng {g.rank} • Lưu Phái: <span className="text-zinc-300 font-medium">{parsePaths(spec?.path).map(translatePath).join(", ") || "Không"}</span>
                                </p>
                                <p className="text-[10px] text-zinc-500 font-mono mt-0.5">
                                  Đang có: <span className="text-zinc-300 font-semibold">{g.instances.length} con</span>
                                </p>
                                <p className="text-[10px] text-yellow-500/90 font-mono mt-0.5 font-semibold">
                                  Trị giá: +{g.value} Pts/Con
                                </p>
                              </div>
                            </div>

                            {/* Quantity control */}
                            {isSelected && (
                              <div className="flex items-center justify-between border-t border-zinc-900/80 pt-2" onClick={e => e.stopPropagation()}>
                                <span className="text-[10px] text-zinc-500 font-mono">Hiến tế:</span>
                                <div className="flex items-center gap-1 shrink-0">
                                  <button
                                    type="button"
                                    onClick={() => updateAmount(selectedItem.amountToDonate - 1)}
                                    className="w-5 h-5 flex items-center justify-center bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 rounded text-zinc-400 font-bold transition-all text-[10px]"
                                  >
                                    -
                                  </button>
                                  <input
                                    type="number"
                                    min={1}
                                    max={g.instances.length}
                                    value={selectedItem.amountToDonate}
                                    onChange={(e) => updateAmount(parseInt(e.target.value, 10) || 1)}
                                    className="w-8 text-center bg-zinc-950 border border-zinc-900 rounded py-0.5 text-[10px] font-mono font-bold text-zinc-200 focus:outline-none"
                                  />
                                  <button
                                    type="button"
                                    onClick={() => updateAmount(selectedItem.amountToDonate + 1)}
                                    className="w-5 h-5 flex items-center justify-center bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 rounded text-zinc-400 font-bold transition-all text-[10px]"
                                  >
                                    +
                                  </button>
                                  <button
                                    type="button"
                                    onClick={() => updateAmount(g.instances.length)}
                                    className="ml-1 px-1.5 py-0.5 bg-blue-950 text-blue-400 border border-blue-900/30 rounded text-[9px] font-mono transition-all font-semibold"
                                  >
                                    Max
                                  </button>
                                </div>
                              </div>
                            )}
                          </div>
                        );
                      })
                    )}
                  </div>

                  <span className="text-[9.5px] text-zinc-500 font-mono leading-relaxed mt-1 flex items-start gap-1">
                    ⚠️ Lưu ý: Bản Mệnh Cổ và Cổ Trùng đang trang bị không thể quyên hiến cho tông môn.
                  </span>
                </div>

                {/* Right side: Summary & Submit Action */}
                <div className="bg-zinc-900/10 border border-zinc-900 rounded-2xl p-5 flex flex-col gap-4 h-fit md:sticky md:top-4">
                  <h4 className="text-xs font-bold text-zinc-350 font-serif border-b border-zinc-900 pb-2.5">
                    Tóm Tắt Quyên Góp ({selectedCount} loại)
                  </h4>

                  {selectedCount === 0 ? (
                    <div className="flex flex-col items-center justify-center py-10 text-center gap-2">
                      <Sparkles className="w-8 h-8 text-zinc-700 stroke-[1.5]" />
                      <span className="text-[11px] text-zinc-500 font-mono">
                        Chọn cổ trùng để hiến tế tông môn.
                      </span>
                    </div>
                  ) : (
                    <>
                      {/* Selected Gu Itemized List */}
                      <div className="max-h-48 overflow-y-auto space-y-2 pr-1 scrollbar-thin">
                        {Object.values(selectedDonateGus).map((item) => (
                          <div key={item.key} className="flex justify-between items-center text-[10px] font-mono bg-zinc-950/20 border border-zinc-950 rounded-lg p-2">
                            <span className="text-zinc-300 truncate max-w-[120px]">{item.name}</span>
                            <span className="text-zinc-500">
                              x{item.amountToDonate} → <span className="text-yellow-500/90 font-bold">+{item.amountToDonate * item.valuePerItem} Pts</span>
                            </span>
                          </div>
                        ))}
                      </div>

                      <div className="border-t border-zinc-900 pt-3.5 space-y-3.5">
                        <div className="flex justify-between items-center text-xs font-mono">
                          <span className="text-zinc-400 font-serif font-bold">Tổng cống hiến:</span>
                          <span className="text-yellow-500 font-bold text-sm">+{totalGainedPoints} Pts</span>
                        </div>
                        <button
                          onClick={async () => {
                            const gusPayload = Object.values(selectedDonateGus).flatMap(item => {
                              return item.instances.slice(0, item.amountToDonate).map(gi => ({
                                gu_instance_id: gi.instance_id
                              }));
                            });
                            const ok = await donateSectBulk({ materials: [], gus: gusPayload, recipes: [] });
                            if (ok) setSelectedDonateGus({});
                          }}
                          disabled={actionLoading}
                          className="w-full py-3 bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 disabled:opacity-50 text-white rounded-xl font-semibold text-sm shadow-md transition-all cursor-pointer flex items-center justify-center gap-1.5"
                        >
                          <Gift className="w-4 h-4" />
                          HIẾN TẾ HÀNG LOẠT
                        </button>
                      </div>
                    </>
                  )}
                </div>
              </div>
            );
          })()}
      </div>
      )}

      {activeTab === "manage" && (
        <div className="bg-zinc-950/40 border border-zinc-900 rounded-3xl p-5 backdrop-blur-md flex flex-col gap-6 animate-slide-up">
          {/* Sub tabs for Management */}
          <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 border-b border-zinc-900 pb-4">
            <div className="flex flex-col gap-1">
              <h3 className="text-base font-serif font-bold text-zinc-200">
                Môn Phái Quản Sự
              </h3>
              <p className="text-xs text-zinc-500 mt-0.5 font-mono">
                Chưởng môn hoặc Trưởng lão thiết lập môn quy, phân phong chức vụ và phát hành nhiệm vụ cho đệ tử.
              </p>
              {(mySectStatus?.member?.member_rank === "CHUONG_MON" || mySectStatus?.member?.member_rank === "TRUONG_LAO" || mySectStatus?.member?.member_rank === "HO_PHAP") && (
                <div className="flex items-center gap-2 mt-2 bg-zinc-900/40 border border-zinc-800/80 px-3 py-1.5 rounded-xl w-fit">
                  <span className="text-[11px] text-zinc-300 font-semibold">Duyệt Đệ Tử Khi Gia Nhập:</span>
                  <button
                    type="button"
                    onClick={async () => {
                      await toggleSectApproval(!mySectStatus?.sect?.require_approval);
                    }}
                    disabled={actionLoading || (mySectStatus?.sect?.level || 1) < 10}
                    className={`w-9 h-5 flex items-center rounded-full p-1 cursor-pointer transition-colors duration-300 ${(mySectStatus?.sect?.level || 1) < 10 ? "opacity-50 cursor-not-allowed" : mySectStatus?.sect?.require_approval ? "bg-emerald-600" : "bg-zinc-800"
                      }`}
                    title={(mySectStatus?.sect?.level || 1) < 10 ? "Tông môn phải đạt cấp 10 mới có thể bật duyệt thành viên" : ""}
                  >
                    <div
                      className={`bg-white w-3 h-3 rounded-full shadow-md transform transition-transform duration-300 ${mySectStatus?.sect?.require_approval ? "translate-x-4" : "translate-x-0"
                        }`}
                    />
                  </button>
                  <span className="w-px h-4 bg-zinc-800 mx-1" />
                  <span className="text-[11px] text-zinc-300 font-semibold" title="Khi bật, mọi thăng cấp đều cần Trưởng Lão/Chưởng Môn duyệt">Duyệt Lên Cấp:</span>
                  <button
                    type="button"
                    onClick={async () => {
                      await toggleSectPromotionApproval(!mySectStatus?.sect?.require_promotion_approval);
                    }}
                    disabled={actionLoading}
                    className={`w-9 h-5 flex items-center rounded-full p-1 cursor-pointer transition-colors duration-300 ${mySectStatus?.sect?.require_promotion_approval ? "bg-emerald-600" : "bg-zinc-800"
                      }`}
                  >
                    <div
                      className={`bg-white w-3 h-3 rounded-full shadow-md transform transition-transform duration-300 ${mySectStatus?.sect?.require_promotion_approval ? "translate-x-4" : "translate-x-0"
                        }`}
                    />
                  </button>
                </div>
              )}
            </div>

            <div className="flex flex-wrap bg-zinc-900/50 p-1 rounded-xl border border-zinc-800/60 self-start sm:self-auto gap-1">
              <button
                onClick={() => setMgmtTab("members")}
                className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${mgmtTab === "members" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"
                  }`}
              >
                Đệ Tử Danh Sách
              </button>
              <button
                onClick={() => setMgmtTab("missions")}
                className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${mgmtTab === "missions" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"
                  }`}
              >
                Tạo Nhiệm Vụ
              </button>
              <button
                onClick={() => setMgmtTab("stock")}
                className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${mgmtTab === "stock" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"
                  }`}
              >
                Nhập Kho & Tăng Cấp
              </button>
              {(mySectStatus?.member?.member_rank === "CHUONG_MON" || mySectStatus?.member?.member_rank === "TRUONG_LAO" || mySectStatus?.member?.member_rank === "HO_PHAP") && (
                <button
                  onClick={() => setMgmtTab("requests")}
                  className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer relative ${mgmtTab === "requests" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"
                    }`}
                >
                  Duyệt Yêu Cầu
                  {((sectJoinRequests?.length || 0) + (sectPromotionRequests?.length || 0)) > 0 && (
                    <span className="absolute -top-1 -right-1 bg-red-500 text-white text-[8px] px-1.5 py-0.5 rounded-full font-mono font-bold leading-none scale-90">
                      {(sectJoinRequests?.length || 0) + (sectPromotionRequests?.length || 0)}
                    </span>
                  )}
                </button>
              )}
              {(mySectStatus?.member?.member_rank === "CHUONG_MON" || mySectStatus?.member?.member_rank === "TRUONG_LAO" || mySectStatus?.member?.member_rank === "HO_PHAP") && (
                <button
                  onClick={() => setMgmtTab("salary")}
                  className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${mgmtTab === "salary" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"
                    }`}
                >
                  Phát Lương
                </button>
              )}
              {mySectStatus?.member?.member_rank === "CHUONG_MON" && (
                <button
                  onClick={() => setMgmtTab("relocate")}
                  className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer flex items-center gap-1.5 ${mgmtTab === "relocate" ? "bg-amber-900/60 border border-amber-600/50 text-amber-200" : "text-zinc-400 hover:text-zinc-200"
                    }`}
                >
                  <Navigation className="w-3.5 h-3.5 text-amber-400" />
                  Di Dời Sơn Môn
                </button>
              )}
            </div>
          </div>

          {/* Members Management Tab */}
          {mgmtTab === "members" && (
            <div className="flex flex-col gap-3">
              <div className="flex justify-between items-center text-xs text-zinc-400 font-mono">
                <span>Danh sách đệ tử tông môn:</span>
                <button
                  type="button"
                  onClick={() => fetchSectMembers()}
                  className="p-1 rounded bg-zinc-900 border border-zinc-800 hover:bg-zinc-800 transition-colors cursor-pointer"
                >
                  <RefreshCw className="w-3.5 h-3.5" />
                </button>
              </div>

              <div className="flex flex-col gap-2 max-h-[400px] overflow-y-auto pr-1">
                {sectMembers && sectMembers.length > 0 ? (
                  sectMembers.map((mInfo) => {
                    const isSelf = mInfo.player_id === player.id;
                    const callerIsLeader = member?.member_rank === "CHUONG_MON";

                    return (
                      <div
                        key={mInfo.player_id}
                        className="bg-zinc-900/20 border border-zinc-900 rounded-2xl p-4 flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 text-xs"
                      >
                        <div className="flex flex-col gap-1 min-w-0">
                          <div className="flex items-center gap-2">
                            <div className="w-6 h-6 rounded-full shrink-0 overflow-hidden bg-zinc-800 flex items-center justify-center text-[8px] font-bold text-zinc-500">
                              {mInfo.avatar_url ? (
                                <img src={mInfo.avatar_url} alt="" className="w-full h-full object-cover" />
                              ) : (
                                mInfo.username.slice(0, 2).toUpperCase()
                              )}
                            </div>
                            <span
                              onClick={() => !isSelf && fetchPlayerProfile(mInfo.username)}
                              className={`font-semibold text-zinc-100 truncate ${!isSelf ? "cursor-pointer hover:text-blue-400 hover:underline decoration-dotted" : ""}`}
                              title={!isSelf ? "Bấm để trinh thám cổ sư" : undefined}
                            >
                              {mInfo.username}
                            </span>
                            <span className={`px-2 py-0.5 text-[9px] font-mono rounded border ${getRankColor(mInfo.member_rank)}`}>
                              {getRankName(mInfo.member_rank)}
                            </span>
                            {mInfo.is_online ? (
                              <span className="flex items-center gap-1 text-[9px] font-mono text-emerald-400 bg-emerald-950/40 border border-emerald-500/30 px-1.5 py-0.5 rounded-full">
                                <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
                                Online
                              </span>
                            ) : (
                              <span className="text-[9px] font-mono text-zinc-500 bg-zinc-950 px-1.5 py-0.5 rounded border border-zinc-800">
                                {mInfo.last_activity
                                  ? (() => {
                                      const days = Math.floor((nowMs - new Date(mInfo.last_activity).getTime()) / (24 * 3600 * 1000));
                                      return days > 0 ? `Offline ${days} ngày` : "Offline";
                                    })()
                                  : "Offline"}
                              </span>
                            )}
                            {isSelf && <span className="text-[9px] text-blue-400 font-mono font-semibold">(Đạo hữu)</span>}
                          </div>
                          <div className="text-[10px] text-zinc-500 font-mono">
                            Tu vi: Rank {mInfo.player_rank} | Cống hiến tích lũy: <strong className="text-yellow-500">{mInfo.contribution_points} Pts</strong>
                          </div>
                        </div>

                        {/* Action buttons */}
                        {isSelf && isEligibleToClaim && (
                          <div className="flex items-center gap-2.5 w-full sm:w-auto justify-end">
                            <button
                              type="button"
                              onClick={async () => {
                                if (await showConfirm("Đạo hữu có tư cách cao nhất hiện tại! Đạo hữu có chắc chắn muốn tiến hành đại điển kế vị, chính thức đăng cơ Chưởng Môn tông môn?")) {
                                  await claimSectLeadership();
                                }
                              }}
                              disabled={actionLoading}
                              className="px-3 py-1 bg-amber-600 hover:bg-amber-500 text-white font-serif font-bold text-xs rounded-xl shadow-md transition-all cursor-pointer flex items-center gap-1.5"
                            >
                              <Crown className="w-3.5 h-3.5" />
                              Tiếp Quản Chưởng Môn
                            </button>
                          </div>
                        )}
                        {!isSelf && (
                          <div className="flex items-center gap-2.5 w-full sm:w-auto justify-end">
                            {/* Promote rank select (Chưởng môn only) */}
                            {callerIsLeader && (
                              <select
                                value={mInfo.member_rank}
                                onChange={async (e) => {
                                  const targetRank = e.target.value;
                                  if (targetRank === "CHUONG_MON" && !await showConfirm("⚠️ CẢNH BÁO: Phong đệ tử này làm Chưởng Môn sẽ nhường lại ngôi vị của đạo hữu và giáng đạo hữu xuống Trưởng Lão! Xác nhận?")) {
                                    return;
                                  }
                                  await setMemberRank(mInfo.player_id, targetRank);
                                }}
                                className="max-w-[180px] bg-zinc-950 border border-zinc-900 rounded-xl px-2.5 py-1 text-[11px] text-zinc-300 focus:outline-none focus:border-yellow-500/50"
                              >
                                <option value="NGOAI_MON">Ngoại Môn</option>
                                <option value="NOI_MON">Nội Môn</option>
                                <option value="TINH_ANH">Tinh Anh</option>
                                <option value="CHAN_TRUYEN">Chân Truyền</option>
                                <option value="TRUONG_LAO">Trưởng Lão</option>
                                <option value="HO_PHAP">Hộ Pháp</option>
                                <option value="CHUONG_MON">Nhường Ngôi Chưởng Môn</option>
                              </select>
                            )}

                            {/* Expel button */}
                            {(callerIsLeader || rankValue(member?.member_rank) > rankValue(mInfo.member_rank)) && (
                              <button
                                type="button"
                                onClick={async () => {
                                  if (await showConfirm(`Xác nhận trục xuất đệ tử "${mInfo.username}" khỏi tông môn?`)) {
                                    await kickMember(mInfo.player_id);
                                  }
                                }}
                                className="px-2.5 py-1 bg-red-950/40 hover:bg-red-900/20 text-red-400 border border-red-500/20 rounded-xl transition-all cursor-pointer font-semibold"
                              >
                                Trục Xuất
                              </button>
                            )}
                          </div>
                        )}
                      </div>
                    );
                  })
                ) : (
                  <div className="text-center py-6 text-zinc-600 font-mono">Chưa có thành viên nào khác</div>
                )}
              </div>
            </div>
          )}

          {/* Requests Management Tab */}
          {mgmtTab === "requests" && (
            <div className="flex flex-col gap-6">
              {/* Join Requests Section */}
              <div className="flex flex-col gap-3">
                <div className="flex justify-between items-center border-b border-zinc-900 pb-2">
                  <h4 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2">
                    <Shield className="w-4 h-4 text-emerald-400" />
                    Yêu Cầu Gia Nhập Tông Môn
                  </h4>
                  <button
                    type="button"
                    onClick={() => fetchJoinRequests()}
                    className="p-1 rounded bg-zinc-900 border border-zinc-800 hover:bg-zinc-800 transition-colors cursor-pointer text-zinc-400 hover:text-zinc-200"
                  >
                    <RefreshCw className="w-3.5 h-3.5" />
                  </button>
                </div>

                <div className="flex flex-col gap-2 max-h-[300px] overflow-y-auto pr-1">
                  {sectJoinRequests && sectJoinRequests.length > 0 ? (
                    sectJoinRequests.map((reqInfo) => (
                      <div
                        key={reqInfo.id}
                        className="bg-zinc-900/20 border border-zinc-900 rounded-2xl p-4 flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 text-xs"
                      >
                        <div className="flex items-center gap-2 min-w-0">
                          <div className="w-6 h-6 rounded-full shrink-0 overflow-hidden bg-zinc-800 flex items-center justify-center text-[8px] font-bold text-zinc-500">
                            {reqInfo.avatar_url ? (
                              <img src={reqInfo.avatar_url} alt="" className="w-full h-full object-cover" />
                            ) : (
                              reqInfo.username?.slice(0, 2).toUpperCase() || "??"
                            )}
                          </div>
                          <div className="flex flex-col gap-1 min-w-0">
                            <span
                              onClick={() => reqInfo.username && fetchPlayerProfile(reqInfo.username)}
                              className="font-semibold text-zinc-100 truncate cursor-pointer hover:text-blue-400 hover:underline decoration-dotted"
                              title="Bấm để trinh thám cổ sư"
                            >
                              {reqInfo.username}
                            </span>
                            <div className="text-[10px] text-zinc-500 font-mono">
                              Tu vi: Rank {reqInfo.player_rank} | Gửi lúc: {new Date(reqInfo.created_at).toLocaleString()}
                            </div>
                          </div>
                        </div>

                        <div className="flex items-center gap-2 w-full sm:w-auto justify-end">
                          <button
                            type="button"
                            onClick={async () => {
                              await actionJoinRequest(reqInfo.id, "approve");
                            }}
                            disabled={actionLoading}
                            className="px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl transition-all cursor-pointer font-semibold text-[11px]"
                          >
                            Đồng Ý
                          </button>
                          <button
                            type="button"
                            onClick={async () => {
                              await actionJoinRequest(reqInfo.id, "reject");
                            }}
                            disabled={actionLoading}
                            className="px-3 py-1.5 bg-red-950/40 hover:bg-red-900/20 text-red-400 border border-red-500/20 rounded-xl transition-all cursor-pointer font-semibold text-[11px]"
                          >
                            Từ Chối
                          </button>
                        </div>
                      </div>
                    ))
                  ) : (
                    <div className="text-center py-6 text-zinc-600 font-mono">Chưa có yêu cầu gia nhập nào</div>
                  )}
                </div>
              </div>

              {/* Promotion Requests Section */}
              <div className="flex flex-col gap-3">
                <div className="flex justify-between items-center border-b border-zinc-900 pb-2">
                  <h4 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2">
                    <Award className="w-4 h-4 text-yellow-400" />
                    Yêu Cầu Thăng Chức
                  </h4>
                  <button
                    type="button"
                    onClick={() => fetchPromotionRequests()}
                    className="p-1 rounded bg-zinc-900 border border-zinc-800 hover:bg-zinc-800 transition-colors cursor-pointer text-zinc-400 hover:text-zinc-200"
                  >
                    <RefreshCw className="w-3.5 h-3.5" />
                  </button>
                </div>

                <div className="flex flex-col gap-2 max-h-[300px] overflow-y-auto pr-1">
                  {sectPromotionRequests && sectPromotionRequests.length > 0 ? (
                    sectPromotionRequests.map((reqInfo) => (
                      <div
                        key={reqInfo.id}
                        className="bg-zinc-900/20 border border-zinc-900 rounded-2xl p-4 flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 text-xs"
                      >
                        <div className="flex items-center gap-2 min-w-0">
                          <div className="w-6 h-6 rounded-full shrink-0 overflow-hidden bg-zinc-800 flex items-center justify-center text-[8px] font-bold text-zinc-500">
                            {reqInfo.avatar_url ? (
                              <img src={reqInfo.avatar_url} alt="" className="w-full h-full object-cover" />
                            ) : (
                              reqInfo.username?.slice(0, 2).toUpperCase() || "??"
                            )}
                          </div>
                          <div className="flex flex-col gap-1 min-w-0">
                            <span
                              onClick={() => reqInfo.username && fetchPlayerProfile(reqInfo.username)}
                              className="font-semibold text-zinc-100 truncate cursor-pointer hover:text-blue-400 hover:underline decoration-dotted"
                              title="Bấm để trinh thám cổ sư"
                            >
                              {reqInfo.username}
                            </span>
                            <div className="text-[10px] text-zinc-300 font-mono mt-0.5">
                              Cấp bậc: <span className="text-red-400 font-semibold">{getRankName(reqInfo.current_rank)}</span> &rarr;{" "}
                              <span className="text-emerald-400 font-semibold">{getRankName(reqInfo.target_rank)}</span>
                            </div>
                            <div className="text-[10px] text-zinc-500 font-mono">
                              Tu vi: Rank {reqInfo.player_rank} | Gửi lúc: {new Date(reqInfo.created_at).toLocaleString()}
                            </div>
                          </div>
                        </div>

                        <div className="flex items-center gap-2 w-full sm:w-auto justify-end">
                          <button
                            type="button"
                            onClick={async () => {
                              await actionPromotionRequest(reqInfo.id, "approve");
                            }}
                            disabled={actionLoading}
                            className="px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-xl transition-all cursor-pointer font-semibold text-[11px]"
                          >
                            Duyệt Cho Lên
                          </button>
                          <button
                            type="button"
                            onClick={async () => {
                              await actionPromotionRequest(reqInfo.id, "reject");
                            }}
                            disabled={actionLoading}
                            className="px-3 py-1.5 bg-red-950/40 hover:bg-red-900/20 text-red-400 border border-red-500/20 rounded-xl transition-all cursor-pointer font-semibold text-[11px]"
                          >
                            Bác Bỏ
                          </button>
                        </div>
                      </div>
                    ))
                  ) : (
                    <div className="text-center py-6 text-zinc-600 font-mono">Chưa có yêu cầu thăng chức nào</div>
                  )}
                </div>
              </div>
            </div>
          )}

          {/* Salary Management Tab */}
          {mgmtTab === "salary" && (
            <div className="bg-zinc-950/20 border border-zinc-900 rounded-2xl p-5 flex flex-col gap-5">
              <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 border-b border-zinc-900 pb-4">
                <div>
                  <h4 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-1.5">
                    <Coins className="w-4 h-4 text-yellow-400" />
                    Phát Lương Thường Nhất
                  </h4>
                  <p className="text-xs text-zinc-500 mt-0.5">
                    Trả lương cho toàn thể đệ tử từ quỹ tông môn. Lương = (Cấp tông môn × 100) + Phụ cấp chức vụ.
                  </p>
                </div>
                <div className="bg-zinc-900/40 px-4 py-3 rounded-xl border border-zinc-800/80 self-start sm:self-auto">
                  <span className="text-[10px] text-zinc-500 font-mono block">Quỹ tông môn hiện có:</span>
                  <span className="font-mono font-bold text-cyan-400 text-sm flex items-center gap-1">
                    <Coins className="w-4 h-4 text-yellow-500" />
                    {(mySectStatus?.sect?.funds || 0).toLocaleString()} Quỹ
                  </span>
                </div>
              </div>

              <div className="bg-zinc-900/30 border border-zinc-800/60 rounded-xl p-4 flex flex-col gap-3">
                <div className="grid grid-cols-2 gap-4 text-xs">
                  <div className="bg-zinc-950/50 rounded-lg p-3">
                    <span className="text-zinc-400 font-mono block">Cấp Tông Môn</span>
                    <span className="font-bold text-cyan-400 text-lg font-mono">{mySectStatus?.sect?.level || 1}</span>
                  </div>
                  <div className="bg-zinc-950/50 rounded-lg p-3">
                    <span className="text-zinc-400 font-mono block">Lương Cơ Bản / Người</span>
                    <span className="font-bold text-emerald-400 text-lg font-mono">{(mySectStatus?.sect?.level || 1) * 100} Linh Thạch</span>
                  </div>
                </div>
                <div className="text-[10px] text-zinc-500 font-mono">
                  Phụ cấp chức vụ: NGOẠI MÔN: 0 | NỘI MÔN: 50 | TINH ANH: 150 | CHÂN TRUYỀN: 400 | TRƯỞNG LÃO: 1.000 | HỘ PHÁP: 2.000 | CHƯƠNG MÔN: 5.000
                </div>
                {sectMembers && sectMembers.length > 0 && (
                  <div className="border-t border-zinc-900 pt-3">
                    <span className="text-xs text-zinc-300 font-semibold">Danh sách nhận lương dự kiến:</span>
                    <div className="flex flex-col gap-1 max-h-[200px] overflow-y-auto mt-2 pr-1">
                      {sectMembers.map((m) => {
                        const baseSalary = (mySectStatus?.sect?.level || 1) * 100;
                        const rankBonus = rankSalaryBonus[m.member_rank] || 0;
                        const totalSalary = baseSalary + rankBonus;
                        return (
                          <div key={m.player_id} className="flex justify-between items-center text-[11px] bg-zinc-950/30 rounded-lg px-2 py-1.5">
                            <div className="flex items-center gap-2">
                              <span className="font-mono text-zinc-400">{getRankName(m.member_rank)}</span>
                              <span className="text-zinc-100 truncate max-w-[120px]">{m.username}</span>
                            </div>
                            <span className="font-bold text-emerald-400 font-mono">{totalSalary.toLocaleString()} Linh Thạch</span>
                          </div>
                        );
                      })}
                    </div>
                  </div>
                )}
                <div className="bg-zinc-950/50 rounded-lg p-3 text-center text-xs">
                  <span className="text-zinc-400 font-mono">Tổng lương dự kiến: </span>
                  <span className="font-bold text-emerald-400 text-lg font-mono">
                    {(() => {
                      if (!sectMembers) return 0;
                      const base = (mySectStatus?.sect?.level || 1) * 100;
                      return sectMembers.reduce((sum, m) => sum + base + (rankSalaryBonus[m.member_rank] || 0), 0);
                    })().toLocaleString()} Linh Thạch
                  </span>
                </div>
                <button
                  type="button"
                  onClick={async () => {
                    await paySectSalary();
                  }}
                  disabled={actionLoading}
                  className="w-full py-3 bg-emerald-600/20 hover:bg-emerald-600/30 border border-emerald-500/30 text-emerald-400 rounded-xl font-semibold transition-all cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {actionLoading ? "Đang phát lương..." : "Xác Nhận Phát Lương"}
                </button>
              </div>
            </div>
          )}

          {/* Create Mission Tab */}
          {mgmtTab === "missions" && (
            <form onSubmit={handleCreateCustomMission} className="bg-zinc-900/10 border border-zinc-900 rounded-2xl p-5 flex flex-col gap-4 mx-auto w-full">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div className="sm:col-span-2 flex flex-col gap-1.5">
                  <label className="text-xs text-zinc-400 font-serif font-bold">Tiêu Đề Nhiệm Vụ</label>
                  <input
                    type="text"
                    required
                    placeholder="Ví dụ: Đóng Góp Linh Chi Cáp, Săn Hoang thú Trảm Thiết Thử..."
                    value={mTitle}
                    onChange={(e) => setMTitle(e.target.value)}
                    className="w-full bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2.5 text-xs text-zinc-100 focus:outline-none focus:border-blue-500/50"
                  />
                </div>

                <div className="sm:col-span-2 flex flex-col gap-1.5">
                  <label className="text-xs text-zinc-400 font-serif font-bold">Mô Tả Yêu Cầu</label>
                  <textarea
                    required
                    placeholder="Mô tả kỹ nhiệm vụ cho đệ tử dễ hiểu và thực hiện..."
                    value={mDesc}
                    onChange={(e) => setMDesc(e.target.value)}
                    rows={2}
                    className="w-full bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2.5 text-xs text-zinc-100 focus:outline-none focus:border-blue-500/50 resize-none"
                  />
                </div>

                <div className="flex flex-col gap-1.5">
                  <label className="text-xs text-zinc-400 font-serif font-bold">Thể Loại</label>
                  <select
                    value={mMissionType}
                    onChange={(e) => setMMissionType(e.target.value)}
                    className="w-full bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs text-zinc-100 focus:outline-none"
                  >
                    <option value="GATHER">Thu thập Linh tài</option>
                    <option value="DONATE">Cống hiến Cổ trùng</option>
                  </select>
                </div>

                <div className="sm:col-span-2 flex flex-col gap-1.5">
                  <label className="text-xs text-zinc-400 font-serif font-bold">Mục Tiêu</label>
                  <div className="flex flex-col gap-1">
                    <input
                      type="text"
                      placeholder="Lọc nhanh..."
                      value={mTargetSearch}
                      onChange={e => { setMTargetSearch(e.target.value); setMTargetId(""); }}
                      className="w-full bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs text-zinc-100 focus:outline-none"
                    />
                    <select
                      required
                      value={mTargetId}
                      onChange={(e) => setMTargetId(e.target.value)}
                      className="w-full bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs text-zinc-100 focus:outline-none"
                    >
                      <option value="">-- Chọn --</option>
                      {(mMissionType === "DONATE" ? Object.entries(guDictionary) : Object.entries(itemDictionary))
                        .filter(([id]) => !mTargetSearch || id.toLowerCase().includes(mTargetSearch.toLowerCase()))
                        .sort(([, a], [, b]) => ((a as ApiResponseData)?.rank || 99) - ((b as ApiResponseData)?.rank || 99))
                        .map(([id, spec]) => (
                          <option key={id} value={id}>{(spec as ApiResponseData)?.name || id} ({id})</option>
                        ))}
                    </select>
                  </div>
                </div>

                <div className="flex flex-col gap-1.5">
                  <label className="text-xs text-zinc-400 font-serif font-bold">Số lượng yêu cầu</label>
                  <input
                    type="number"
                    required
                    min={1}
                    value={mTargetAmount}
                    onChange={(e) => setMTargetAmount(Math.max(1, parseInt(e.target.value) || 1))}
                    className="bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs text-zinc-100 focus:outline-none font-mono"
                  />
                </div>

                <div className="flex flex-col gap-1.5">
                  <label className="text-xs text-zinc-400 font-serif font-bold">Điểm đóng góp thưởng</label>
                  <input
                    type="number"
                    required
                    min={1}
                    value={mRewardContribution}
                    onChange={(e) => setMRewardContribution(Math.max(1, parseInt(e.target.value) || 1))}
                    className="bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs text-zinc-100 focus:outline-none font-mono"
                  />
                </div>

                <div className="flex flex-col gap-1.5">
                  <label className="text-xs text-zinc-400 font-serif font-bold">Thời hạn (giờ)</label>
                  <input
                    type="number"
                    required
                    min={-1}
                    value={mDuration}
                    onChange={(e) => setMDuration(parseInt(e.target.value) || 0)}
                    className="bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs text-zinc-100 focus:outline-none font-mono"
                  />
                  <span className="text-[9px] text-zinc-600 font-mono leading-tight">Nhập 0 để dùng mặc định 24 giờ. Nhập -1 để nhiệm vụ không hết hạn.</span>
                </div>

                <div className="sm:col-span-2 flex flex-col gap-1.5">
                  <label className="text-xs text-zinc-400 font-serif font-bold">Cấp bậc đệ tử yêu cầu nhận</label>
                  <select
                    value={mRequiredRank}
                    onChange={(e) => setMRequiredRank(e.target.value)}
                    className="w-full bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs text-zinc-100 focus:outline-none"
                  >
                    <option value="NGOAI_MON">Đệ Tử Ngoại Môn</option>
                    <option value="NOI_MON">Đệ Tử Nội Môn</option>
                    <option value="TINH_ANH">Đệ Tử Tinh Anh</option>
                    <option value="CHAN_TRUYEN">Đệ Tử Chân Truyền</option>
                    <option value="TRUONG_LAO">Trưởng Lão</option>
                    <option value="HO_PHAP">Hộ Pháp</option>
                  </select>
                </div>
              </div>

              <button
                type="submit"
                disabled={actionLoading}
                className="w-full py-2.5 bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 disabled:opacity-50 text-white rounded-xl font-serif font-bold text-sm shadow-md transition-all cursor-pointer flex items-center justify-center gap-1.5"
              >
                <Plus className="w-4 h-4" />
                Phát Hành Nhiệm Vụ Môn Phái
              </button>
            </form>
          )}

          {/* Stock Vault & Upgrade Sect Tab */}
          {mgmtTab === "stock" && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6 w-full">
              {/* Stock Vault Card */}

              {<div className="bg-zinc-950/20 border border-zinc-900 rounded-2xl p-5 flex flex-col gap-5 md:col-span-2">
                <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 border-b border-zinc-900 pb-4">
                  <div>
                    <h4 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-1.5">
                      <Plus className="w-4 h-4 text-cyan-400" />
                      Mua Sắm Vật Phẩm (Nhập Kho Bảo Khố)
                    </h4>
                    <p className="text-xs text-zinc-500 mt-0.5">
                      Dùng quỹ tông môn để mua thêm Cổ trùng hoặc Linh tài đưa vào bảo khố để đệ tử trao đổi.
                    </p>
                  </div>
                  <div className="bg-zinc-900/40 px-4 py-3 rounded-xl border border-zinc-800/80 self-start sm:self-auto">
                    <span className="text-[10px] text-zinc-500 font-mono block">Quỹ tông môn hiện có:</span>
                    <span className="font-mono font-bold text-cyan-400 text-sm flex items-center gap-1">
                      <Coins className="w-4 h-4 text-yellow-500" />
                      {(sect?.funds || 0).toLocaleString()} Quỹ
                    </span>
                  </div>
                </div>

                {/* Tabs hidden: Only material deposit is allowed */}

                {(() => {
                  const sectLvl = sect?.level || 1;
                  const recipeEntries: [string, { name: string; rank: number; rarity: number; path?: string[] | string }][] = recipes
                    ? (recipes.map((r) => {
                        const targetGuSpec = guDictionary[r.target_gu_id];
                        return [
                          r.id,
                          {
                            name: targetGuSpec?.name || r.id.replace("recipe_", "").toUpperCase(),
                            rank: targetGuSpec?.rank || 1,
                            rarity: targetGuSpec?.rarity || 0,
                            path: targetGuSpec?.path,
                          },
                        ] as [string, { name: string; rank: number; rarity: number; path?: string[] | string }];
                      }))
                    : [];

                  const allEntries = stockItemType === "gu"
                    ? Object.entries(guDictionary || {}).filter(([, spec]) => Math.max(spec.rank || 1, spec.rarity || 0) < 6)
                    : stockItemType === "recipe"
                    ? recipeEntries.filter(([, spec]) => Math.max(spec.rank || 1, spec.rarity || 0) < 6)
                    : Object.entries(itemDictionary || {}).filter(([, spec]) => Math.max(spec.rank || 1, spec.rarity || 0) < 6);

                  const availableRanks = Array.from(new Set(allEntries.map(([, spec]) => (spec.rank || 1)))).sort((a, b) => a - b);

                  const availablePaths = Array.from(new Set(allEntries.flatMap(([, spec]): string[] => {
                    return parsePaths(spec.path);
                  }))).filter(Boolean).sort((a, b) => translatePath(a).localeCompare(translatePath(b), "vi"));

                  const filtered = allEntries.filter(([id, spec]) => {
                    const name = spec.name || id;
                    const rank = spec.rank || 1;
                    const q = stockSearch.trim().toLowerCase();
                    const matchesSearch = !q || name.toLowerCase().includes(q) || id.toLowerCase().includes(q);
                    const matchesRank = stockRankFilter === "All" || rank.toString() === stockRankFilter;
                    const paths = parsePaths(spec.path);
                    const matchesPath = stockPathFilter === "All" || paths.includes(stockPathFilter);
                    return matchesSearch && matchesRank && matchesPath;
                  });

                  const selectedCount = Object.keys(stockSelections).length;
                  const totalCost = Object.values(stockSelections).reduce((sum, it) => sum + it.unit_cost * it.quantity, 0);
                  const canAfford = (sect?.funds || 0) >= totalCost;

                  return (
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-6 w-full">
                      <div className="md:col-span-2 bg-zinc-900/10 border border-zinc-900 rounded-2xl p-5 flex flex-col gap-4">
                        <div className="flex justify-between items-center border-b border-zinc-900 pb-3">
                          <div>
                            <h4 className="text-sm font-bold text-zinc-200 font-serif">
                              {stockItemType === "gu" ? "Danh Sách Cổ Trùng" : stockItemType === "recipe" ? "Danh Sách Bí Phương" : "Danh Sách Linh Tài"}
                            </h4>
                            <p className="text-[10px] text-zinc-500 font-mono mt-0.5">
                              {stockItemType === "gu"
                                ? "Tích chọn Cổ trùng (Hạng 1–5) để nhập vào bảo khố."
                                : stockItemType === "recipe"
                                ? "Tích chọn Bí phương (Hạng 1–5) để nhập vào bảo khố."
                                : "Tích chọn Linh tài và số lượng cần nhập kho."}
                            </p>
                          </div>
                          <span className="text-[10px] font-mono text-zinc-400 bg-zinc-900 px-2.5 py-1 rounded-lg border border-zinc-800">
                            Tìm thấy: {filtered.length} loại
                          </span>
                        </div>

                        <div className="flex flex-col sm:flex-row gap-2">
                          <input
                            type="text"
                            value={stockSearch}
                            onChange={(e) => setStockSearch(e.target.value)}
                            placeholder={stockItemType === "gu" ? "Tìm kiếm cổ trùng..." : stockItemType === "recipe" ? "Tìm kiếm bí phương..." : "Tìm kiếm linh tài..."}
                            className="flex-1 bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs font-mono text-zinc-200 focus:outline-none focus:border-cyan-500/50"
                          />
                          <div className="flex gap-2">
                            <select
                              value={stockRankFilter}
                              onChange={(e) => setStockRankFilter(e.target.value)}
                              className="bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs font-mono text-zinc-300 focus:outline-none focus:border-cyan-500/50 cursor-pointer outline-none"
                            >
                              <option value="All">Tất cả Hạng</option>
                              {availableRanks.map((rank) => (
                                <option key={rank} value={rank.toString()}>Hạng {rank}</option>
                              ))}
                            </select>
                            <select
                              value={stockPathFilter}
                              onChange={(e) => setStockPathFilter(e.target.value)}
                              className="bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs font-mono text-zinc-300 focus:outline-none focus:border-cyan-500/50 cursor-pointer outline-none"
                            >
                              <option value="All">Tất cả Phái</option>
                              {availablePaths.map((path) => (
                                <option key={path} value={path}>{translatePath(path)}</option>
                              ))}
                            </select>
                          </div>
                        </div>

                        <div className="max-h-[480px] overflow-y-auto pr-1 grid grid-cols-1 sm:grid-cols-2 gap-3 scrollbar-thin">
                          {filtered.length === 0 ? (
                            <div className="col-span-full text-center py-12 text-xs text-zinc-600 font-mono">Không tìm thấy vật phẩm phù hợp</div>
                          ) : (
                            filtered.map(([id, spec]) => {
                              const name = spec.name || id;
const rank = spec.rank || 1;
                               const maxItemLevel = Math.min(5, Math.floor(sectLvl / 10) + 1);
                               const locked = rank > maxItemLevel;
                              const unitCost = getStockUnitCost(stockItemType, id);
                              const isSelected = !!stockSelections[id];
                              const sel = stockSelections[id];

                              const toggleSelect = () => {
                                setStockSelections(prev => {
                                  const next = { ...prev };
                                  if (next[id]) {
                                    delete next[id];
                                  } else {
                                    next[id] = { item_type: stockItemType, item_id: id, name, unit_cost: unitCost, quantity: 1 };
                                  }
                                  return next;
                                });
                              };

                              const updateQty = (val: number) => {
                                const qty = Math.max(1, Math.floor(val) || 1);
                                setStockSelections(prev => ({
                                  ...prev,
                                  [id]: { ...prev[id], quantity: qty },
                                }));
                              };

                              return (
                                <div
                                  key={id}
                                  onClick={locked ? undefined : toggleSelect}
                                  className={`bg-zinc-950/40 border rounded-xl p-3 flex flex-col justify-between gap-3.5 transition-all ${
                                    locked ? "opacity-50 cursor-not-allowed" : "cursor-pointer select-none"
                                  } ${
                                    isSelected
                                      ? "border-cyan-600/50 bg-cyan-900/5 shadow-md shadow-cyan-950/10"
                                      : "border-zinc-900 hover:border-zinc-800"
                                  }`}
                                >
                                  <div className="flex gap-2.5 min-w-0 items-start">
                                    <div className={`w-4.5 h-4.5 rounded border flex items-center justify-center shrink-0 mt-0.5 transition-colors ${
                                      isSelected ? "bg-cyan-600 border-cyan-500 text-white" : "bg-zinc-900 border-zinc-800"
                                    }`}>
                                      {isSelected && <Check className="w-3.5 h-3.5 text-white stroke-[3px]" />}
                                    </div>

                                    <div className="min-w-0">
                                      <h5 className="text-xs font-bold text-zinc-200 truncate">{locked ? "🔒 " : ""}{name}</h5>
                                      <p className="text-[10px] text-zinc-500 font-mono mt-1">
                                        Hạng {rank} • Giá nhập: <span className="text-cyan-400 font-semibold">{unitCost} Quỹ</span>
                                      </p>
                                      {locked && (
                                        <p className="text-[10px] text-red-400/80 font-mono mt-0.5">
                                          {(stockItemType === "gu" || stockItemType === "recipe") && rank >= 6 ? "Hạng 6+ không thể nhập" : `Cần Tông Môn Cấp ${(rank - 1) * 10}`}
                                        </p>
                                      )}
                                    </div>
                                  </div>

                                  {isSelected && !locked && (
                                    <div className="flex items-center justify-between border-t border-zinc-900/80 pt-2" onClick={e => e.stopPropagation()}>
                                      <span className="text-[10px] text-zinc-500 font-mono">Nhập kho:</span>
                                      <div className="flex items-center gap-1 shrink-0">
                                        <button
                                          type="button"
                                          onClick={() => updateQty(sel.quantity - 1)}
                                          className="w-5 h-5 flex items-center justify-center bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 rounded text-zinc-400 font-bold transition-all text-[10px]"
                                        >
                                          -
                                        </button>
                                        <input
                                          type="number"
                                          min={1}
                                          value={sel.quantity}
                                          onChange={(e) => updateQty(parseInt(e.target.value, 10) || 1)}
                                          className="w-8 text-center bg-zinc-950 border border-zinc-900 rounded py-0.5 text-[10px] font-mono font-bold text-zinc-200 focus:outline-none"
                                        />
                                        <button
                                          type="button"
                                          onClick={() => updateQty(sel.quantity + 1)}
                                          className="w-5 h-5 flex items-center justify-center bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 rounded text-zinc-400 font-bold transition-all text-[10px]"
                                        >
                                          +
                                        </button>
                                      </div>
                                    </div>
                                  )}
                                </div>
                              );
                            })
                          )}
                        </div>
                      </div>

                      <div className="bg-zinc-900/10 border border-zinc-900 rounded-2xl p-5 flex flex-col gap-4 h-fit md:sticky md:top-4">
                        <h4 className="text-xs font-bold text-zinc-300 font-serif border-b border-zinc-900 pb-2.5">
                          Tóm Tắt Nhập Kho ({selectedCount} loại)
                        </h4>

                        {selectedCount === 0 ? (
                          <div className="flex flex-col items-center justify-center py-10 text-center gap-2">
                            <Package className="w-8 h-8 text-zinc-700 stroke-[1.5]" />
                            <span className="text-[11px] text-zinc-500 font-mono">
                              Chọn vật phẩm từ danh sách để nhập kho.
                            </span>
                          </div>
                        ) : (
                          <>
                            <div className="max-h-48 overflow-y-auto space-y-2 pr-1 scrollbar-thin">
                              {Object.values(stockSelections).map((item) => (
                                <div key={item.item_id} className="flex justify-between items-center text-[10px] font-mono bg-zinc-950/20 border border-zinc-950 rounded-lg p-2">
                                  <span className="text-zinc-300 truncate max-w-[120px]">{item.name}</span>
                                  <span className="text-zinc-500">
                                    x{item.quantity} → <span className="text-cyan-400/90 font-bold">{item.unit_cost * item.quantity} Quỹ</span>
                                  </span>
                                </div>
                              ))}
                            </div>

                            <div className="border-t border-zinc-900 pt-3.5 space-y-3.5">
                              <div className="flex justify-between items-center text-xs font-mono">
                                <span className="text-zinc-400 font-serif font-bold">Tổng chi phí:</span>
                                <span className={`font-bold text-sm ${canAfford ? "text-cyan-400" : "text-red-400"}`}>
                                  {totalCost.toLocaleString()} Quỹ
                                </span>
                              </div>
                              <button
                                onClick={async () => {
                                  const payload = Object.values(stockSelections).map(it => ({
                                    item_type: it.item_type,
                                    item_id: it.item_id,
                                    quantity: it.quantity,
                                  }));
                                  const ok = await stockSectVaultBulk(payload);
                                  if (ok) setStockSelections({});
                                }}
                                disabled={!canAfford || actionLoading}
                                className="w-full py-3 bg-gradient-to-r from-cyan-600 to-sky-600 hover:from-cyan-500 hover:to-sky-500 disabled:opacity-50 text-white rounded-xl font-semibold text-sm shadow-md transition-all cursor-pointer flex items-center justify-center gap-1.5"
                              >
                                <Package className="w-4 h-4" />
                                NHẬP KHO HÀNG LOẠT
                              </button>
                              {!canAfford && (
                                <p className="text-[10px] text-red-400 font-mono text-center">Quỹ tông môn không đủ!</p>
                              )}
                            </div>
                          </>
                        )}
                      </div>
                    </div>
                  );
                })()}
              </div>}

            </div>
          )}
          {mgmtTab === "relocate" && (
            <div className="flex flex-col gap-5">
              <div className="flex justify-between items-center bg-zinc-900/40 p-4 rounded-2xl border border-zinc-800">
                <div className="flex items-center gap-3">
                  <div className="p-2.5 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-400">
                    <Navigation className="w-5 h-5" />
                  </div>
                  <div>
                    <h4 className="text-sm font-bold font-serif text-zinc-100 flex items-center gap-2">
                      Đại Điển Di Dời Sơn Môn
                    </h4>
                    <p className="text-xs text-zinc-400 font-mono mt-0.5">
                      Chưởng Môn xuất hành đến tọa độ dã ngoại mới để dời toàn bộ căn cứ Tông Môn.
                    </p>
                  </div>
                </div>
                <button
                  onClick={async () => {
                    setLoadingEstimate(true);
                    const res = await getRelocationEstimate();
                    setRelocationEstimate(res);
                    setLoadingEstimate(false);
                  }}
                  disabled={loadingEstimate || actionLoading}
                  className="px-3 py-1.5 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-xl text-xs font-mono flex items-center gap-1.5 cursor-pointer transition-colors"
                >
                  <RefreshCw className={`w-3.5 h-3.5 ${loadingEstimate ? "animate-spin" : ""}`} />
                  Cập Nhật Dự Toán
                </button>
              </div>

              {loadingEstimate && !relocationEstimate && (
                <div className="flex items-center justify-center p-12 text-zinc-500 font-mono text-xs">
                  <RefreshCw className="w-5 h-5 animate-spin mr-2" />
                  Đang tính toán chi phí và kiểm tra phong thủy địa lý...
                </div>
              )}

              {relocationEstimate && (
                <div className="flex flex-col gap-4">
                  {/* Status Banner */}
                  {relocationEstimate.can_relocate ? (
                    <div className="bg-emerald-950/40 border border-emerald-500/30 p-3.5 rounded-2xl flex items-center gap-3">
                      <CheckCircle className="w-5 h-5 text-emerald-400 flex-shrink-0" />
                      <div className="text-xs text-emerald-200">
                        <span className="font-bold">Địa thế hợp lệ!</span> Tọa độ dã ngoại hiện tại của Chưởng Môn không bị trùng lấn lãnh địa, không thuộc Khu An Toàn và Quỹ Tông Môn đã sẵn sàng.
                      </div>
                    </div>
                  ) : (
                    <div className="bg-red-950/40 border border-red-500/30 p-3.5 rounded-2xl flex items-center gap-3">
                      <AlertTriangle className="w-5 h-5 text-red-400 flex-shrink-0" />
                      <div className="text-xs text-red-200">
                        <span className="font-bold">Chưa thể di dời: </span>
                        {relocationEstimate.reason || "Vị trí hoặc điều kiện chưa thỏa mãn."}
                      </div>
                    </div>
                  )}

                  {/* Coordinates Comparison Cards */}
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    <div className="bg-zinc-900/30 border border-zinc-800/80 p-3.5 rounded-2xl flex flex-col gap-2">
                      <div className="text-[11px] font-mono text-zinc-400 flex items-center gap-1.5">
                        <MapPin className="w-3.5 h-3.5 text-zinc-500" />
                        Vị Trí Sơn Môn Hiện Tại
                      </div>
                      <div className="text-base font-serif font-bold text-zinc-200">
                        {relocationEstimate.current_sect_location.region}
                      </div>
                      <div className="text-xs font-mono text-zinc-400">
                        Tọa độ: <span className="text-amber-400 font-bold">({relocationEstimate.current_sect_location.x}, {relocationEstimate.current_sect_location.y})</span>
                      </div>
                    </div>

                    <div className="bg-zinc-900/30 border border-amber-500/20 p-3.5 rounded-2xl flex flex-col gap-2">
                      <div className="text-[11px] font-mono text-amber-400/90 flex items-center gap-1.5">
                        <Navigation className="w-3.5 h-3.5 text-amber-400" />
                        Tọa Độ Mới (Vị Trí Chưởng Môn)
                      </div>
                      <div className="text-base font-serif font-bold text-zinc-100">
                        {relocationEstimate.target_location.region}
                      </div>
                      <div className="text-xs font-mono text-zinc-300">
                        Tọa độ: <span className="text-emerald-400 font-bold">({relocationEstimate.target_location.x}, {relocationEstimate.target_location.y})</span>
                        <span className="text-zinc-500 ml-2">(Bán kính {relocationEstimate.target_location.radius} ô)</span>
                      </div>
                    </div>
                  </div>

                  {/* Cooldown Information */}
                  {relocationEstimate.cooldown_remaining_seconds > 0 && (
                    <div className="bg-zinc-900/40 border border-zinc-800 p-3.5 rounded-2xl flex items-center justify-between text-xs font-mono">
                      <span className="text-zinc-400 flex items-center gap-2">
                        <Hourglass className="w-4 h-4 text-amber-400 animate-pulse" />
                        Thời gian hồi di dời (3 ngày / 72h):
                      </span>
                      <span className="text-amber-400 font-bold">
                        {Math.floor(relocationEstimate.cooldown_remaining_seconds / 3600)} giờ {Math.floor((relocationEstimate.cooldown_remaining_seconds % 3600) / 60)} phút
                      </span>
                    </div>
                  )}

                  {/* Cost Breakdown */}
                  <div className="bg-zinc-900/30 border border-zinc-800 rounded-2xl p-4 flex flex-col gap-3">
                    <div className="flex justify-between items-center border-b border-zinc-800 pb-2">
                      <span className="text-xs font-bold text-zinc-300 font-serif">Chi Tiết Dự Toán Quỹ Di Dời</span>
                      <span className="text-xs font-mono text-zinc-400">
                        Quỹ hiện có: <span className={relocationEstimate.current_funds >= relocationEstimate.funds_needed ? "text-emerald-400 font-bold" : "text-red-400 font-bold"}>
                          {relocationEstimate.current_funds.toLocaleString()}
                        </span> Quỹ
                      </span>
                    </div>

                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 text-xs font-mono">
                      <div className="flex justify-between p-2 rounded-xl bg-zinc-950/40 border border-zinc-900">
                        <span className="text-zinc-400">Khởi tạo đại điển:</span>
                        <span className="text-zinc-200 font-bold">{relocationEstimate.cost_breakdown.base_cost.toLocaleString()} Quỹ</span>
                      </div>
                      <div className="flex justify-between p-2 rounded-xl bg-zinc-950/40 border border-zinc-900">
                        <span className="text-zinc-400">Di dời môn nhân ({relocationEstimate.cost_breakdown.member_count} TV):</span>
                        <span className="text-zinc-200 font-bold">{relocationEstimate.cost_breakdown.member_cost.toLocaleString()} Quỹ</span>
                      </div>
                      <div className="flex justify-between p-2 rounded-xl bg-zinc-950/40 border border-zinc-900">
                        <span className="text-zinc-400">Linh tài kho ({relocationEstimate.cost_breakdown.materials_count} món):</span>
                        <span className="text-zinc-200 font-bold">{relocationEstimate.cost_breakdown.materials_cost.toLocaleString()} Quỹ</span>
                      </div>
                      <div className="flex justify-between p-2 rounded-xl bg-zinc-950/40 border border-zinc-900">
                        <span className="text-zinc-400">Phàm Cổ kho ({relocationEstimate.cost_breakdown.mortal_gu_count} con):</span>
                        <span className="text-zinc-200 font-bold">{relocationEstimate.cost_breakdown.mortal_gu_cost.toLocaleString()} Quỹ</span>
                      </div>
                      <div className="flex justify-between p-2 rounded-xl bg-zinc-950/40 border border-zinc-900">
                        <span className="text-zinc-400">Tiên Cổ bảo khố ({relocationEstimate.cost_breakdown.immortal_gu_count} con):</span>
                        <span className="text-zinc-200 font-bold">{relocationEstimate.cost_breakdown.immortal_gu_cost.toLocaleString()} Quỹ</span>
                      </div>
                      <div className="flex justify-between p-2 rounded-xl bg-zinc-950/40 border border-zinc-900">
                        <span className="text-zinc-400">Bí phương ({relocationEstimate.cost_breakdown.recipes_count} quyển):</span>
                        <span className="text-zinc-200 font-bold">{relocationEstimate.cost_breakdown.recipes_cost.toLocaleString()} Quỹ</span>
                      </div>
                      <div className="flex justify-between p-2 rounded-xl bg-emerald-950/20 border border-emerald-900/50 text-emerald-400">
                        <span>Bảo Khố cấp {relocationEstimate.cost_breakdown.treasury_level} tiết kiệm:</span>
                        <span className="font-bold">-{relocationEstimate.cost_breakdown.treasury_discount.toLocaleString()} Quỹ</span>
                      </div>
                    </div>

                    <div className="flex justify-between items-center p-3 rounded-xl bg-amber-950/20 border border-amber-500/30 mt-1">
                      <span className="text-xs font-serif font-bold text-amber-200">Tổng Chi Phí Di Dời:</span>
                      <span className="text-sm font-mono font-bold text-amber-400">
                        {relocationEstimate.cost_breakdown.total_cost.toLocaleString()} Quỹ Tông Môn
                      </span>
                    </div>
                  </div>

                  {/* Action Button */}
                  <button
                    onClick={async () => {
                      const confirmed = await showConfirm(
                        `Đạo hữu có chắc chắn muốn dời căn cứ Tông Môn đến tọa độ (${relocationEstimate.target_location.x}, ${relocationEstimate.target_location.y}) thuộc ${relocationEstimate.target_location.region}? Hành động này tiêu hao ${relocationEstimate.cost_breakdown.total_cost.toLocaleString()} Quỹ Tông Môn và kích hoạt thời gian hồi 3 ngày (72 giờ).`
                      );
                      if (!confirmed) return;
                      const ok = await relocateSect();
                      if (ok) {
                        setLoadingEstimate(true);
                        const res = await getRelocationEstimate();
                        setRelocationEstimate(res);
                        setLoadingEstimate(false);
                      }
                    }}
                    disabled={!relocationEstimate.can_relocate || actionLoading}
                    className="w-full py-3.5 bg-gradient-to-r from-amber-600 via-yellow-600 to-amber-700 hover:from-amber-500 hover:to-yellow-500 disabled:opacity-40 disabled:cursor-not-allowed text-white rounded-2xl font-serif font-bold text-sm shadow-lg shadow-amber-950/40 transition-all cursor-pointer flex items-center justify-center gap-2 mt-2"
                  >
                    <Navigation className="w-4 h-4 text-amber-200" />
                    TIẾN HÀNH ĐẠI ĐIỂN DI DỜI SƠN MÔN
                  </button>
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </GamePanel>
  );
};
