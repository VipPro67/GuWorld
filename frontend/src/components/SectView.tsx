import React, { useState, useEffect } from "react";
import { useGameStore, Sect, SectMission, GuInstance } from "@/store/useGameStore";
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
} from "lucide-react";

export const SectView: React.FC = () => {
  const {
    player,
    sects,
    mySectStatus,
    sectMissions,
    guDictionary,
    itemDictionary,
    actionLoading,
    fetchSects,
    joinSect,
    leaveSect,
    fetchMySectStatus,
    promoteSectRank,
    donateSectStones,
    donateSectMaterials,
    donateSectGu,
    buyFromSectVault,
    fetchSectMissions,
    acceptSectMission,
    completeSectMission,
    sectMembers,
    createSect,
    fetchSectMembers,
    setMemberRank,
    kickMember,
    createCustomMission,
    stockSectVault,
    upgradeSect,
    setView,
    sectJoinRequests,
    sectPromotionRequests,
    toggleSectApproval,
    fetchJoinRequests,
    actionJoinRequest,
    fetchPromotionRequests,
    actionPromotionRequest,
    fetchPlayerProfile,
    sectMines,
    fetchSectMines,
    attackSectMine,
    claimSectMineRewards,
    addToast,
    showConfirm,
    recipes,
  } = useGameStore();

  const [activeTab, setActiveTab] = useState<"lobby" | "missions" | "vault" | "donate" | "territory" | "manage">("lobby");
  const [selectedSectId, setSelectedSectId] = useState<string | null>(null);

  // Sect Creation Form
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [createSectName, setCreateSectName] = useState("");
  const [createSectDesc, setCreateSectDesc] = useState("");
  const [createSectRegion, setCreateSectRegion] = useState("Central_Continent");
  
  // Vault state
  const [vaultTab, setVaultTab] = useState<"gu" | "material" | "recipe">("gu");

  // Donation state
  const [donateType, setDonateType] = useState<"stones" | "materials" | "gu">("stones");
  const [donateStones, setDonateStones] = useState<number>(100);
  const [selectedMatId, setSelectedMatId] = useState<string>("");
  const [donateMatAmount, setDonateMatAmount] = useState<number>(1);
  const [selectedGuInstId, setSelectedGuInstId] = useState<string>("");

  // Sect management sub-tab
  const [mgmtTab, setMgmtTab] = useState<"members" | "missions" | "stock" | "requests">("members");

  // Custom mission creation form state
  const [mTitle, setMTitle] = useState("");
  const [mDesc, setMDesc] = useState("");
  const [mMissionType, setMMissionType] = useState("GATHER");
  const [mTargetId, setMTargetId] = useState("");
  const [mTargetAmount, setMTargetAmount] = useState(1);
  const [mRewardContribution, setMRewardContribution] = useState(100);
  const [mRequiredRank, setMRequiredRank] = useState("NGOAI_MON");

  // Vault stocking form state
  const [stockItemType, setStockItemType] = useState<"gu" | "material">("gu");
  const [stockItemId, setStockItemId] = useState("");
  const [stockQuantity, setStockQuantity] = useState(5);

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
      fetchSectMines();
    }
  }, [activeTab, fetchSectMines]);

  useEffect(() => {
    if (activeTab === "manage") {
      fetchSectMembers();
      const isLeaderOrElder = mySectStatus?.member?.member_rank === "CHUONG_MON" || mySectStatus?.member?.member_rank === "TRUONG_LAO";
      if (isLeaderOrElder) {
        fetchJoinRequests();
        fetchPromotionRequests();
      }
    }
  }, [activeTab, fetchSectMembers, fetchJoinRequests, fetchPromotionRequests, mySectStatus?.member?.member_rank]);

  useEffect(() => {
    if (mgmtTab === "stock" && !stockItemId) {
      if (stockItemType === "gu") {
        const guKeys = Object.keys(guDictionary || {});
        if (guKeys.length > 0) setStockItemId(guKeys[0]);
      } else {
        const itemKeys = Object.keys(itemDictionary || {});
        if (itemKeys.length > 0) setStockItemId(itemKeys[0]);
      }
    }
  }, [mgmtTab, stockItemType, guDictionary, itemDictionary, stockItemId]);

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
    });
    if (ok) {
      setMTitle("");
      setMDesc("");
      setMTargetId("");
      setMTargetAmount(1);
      setMRewardContribution(100);
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
      case "CHUONG_MON": return "text-rose-400 bg-rose-950/25 border-rose-500/30 font-bold";
      default: return "text-zinc-400";
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

  const getRegionName = (region: string): string => {
    switch (region) {
      case "Central_Continent": return "Trung Châu";
      case "Southern_Border": return "Nam Cương";
      case "Northern_Plains": return "Bắc Nguyên";
      case "Western_Desert": return "Tây Mạc";
      case "Eastern_Sea": return "Đông Hải";
      default: return region;
    }
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
      default:
        return null;
    }
  };

  const promotionReq = mySectStatus?.member ? getPromotionRequirements(mySectStatus.member.member_rank) : null;
  const canPromote = promotionReq && player.rank >= promotionReq.reqRank && mySectStatus!.member!.contribution_points >= promotionReq.points;

  // Donation materials listing
  const playerMaterials = Object.entries(player.materials_inventory || {}).filter(([_, qty]) => qty > 0);

  // Donation Gu listing (Filter out Vital Gu and equipped Gu)
  const playerDonatableGus = (player.gu_inventory || []).filter(g => !g.is_vital && !g.equipped && g.status !== "DEAD");

  // Pricing helpers matching backend/pkg/pricing
  const guRankBasePrice = [0, 200, 500, 1200, 2500, 5000, 10000, 25000, 60000, 150000];
  const itemRankBasePrice = [0, 5, 15, 30, 60, 120, 250, 500, 1000, 2500];

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

  // 1. STATE: NOT IN SECT
  if (!mySectStatus?.in_sect) {
    return (
      <div className="w-full max-w-4xl mx-auto px-2 py-4 flex flex-col gap-6 animate-fade-in">
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
            <div className="mt-4 p-3 bg-amber-950/40 border border-amber-500/25 rounded-2xl text-xs text-amber-400 font-mono max-w-md mx-auto">
              ⏳ Yêu cầu gia nhập tông môn <strong className="text-amber-200 font-bold">{mySectStatus.pending_sect_name}</strong> đang chờ duyệt!
            </div>
          )}
        </div>

        {/* Sects Directory */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {sects.map((sect) => (
            <div
              key={sect.id}
              onClick={() => setSelectedSectId(sect.id)}
              className={`group bg-zinc-950/40 hover:bg-zinc-950/80 border ${
                selectedSectId === sect.id ? "border-blue-500/50 shadow-[0_0_20px_rgba(59,130,246,0.15)]" : "border-zinc-900 hover:border-zinc-800"
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

        {/* Selected Sect Detail & Join Panel */}
        {selectedSectId && (() => {
          const sect = sects.find(s => s.id === selectedSectId);
          if (!sect) return null;
          return (
            <div className="bg-zinc-950/60 border border-zinc-900 rounded-3xl p-6 backdrop-blur-md animate-slide-up flex flex-col gap-6">
              <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 border-b border-zinc-900 pb-4">
                <div>
                  <h3 className="text-xl font-serif font-bold text-zinc-100 flex items-center gap-2">
                    <Award className="w-6 h-6 text-blue-400" />
                    {sect.name}
                  </h3>
                  <p className="text-xs text-zinc-500 font-mono mt-1">
                    Vực giới quản hạt: {getRegionName(sect.region)} | Trụ sở thế lực
                  </p>
                </div>
                <button
                  onClick={async () => {
                    const ok = await joinSect(sect.id);
                    if (ok) setSelectedSectId(null);
                  }}
                  disabled={actionLoading || mySectStatus?.has_pending_join}
                  className="w-full sm:w-auto px-6 py-2.5 bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 disabled:from-zinc-900 disabled:to-zinc-950 disabled:border-zinc-800 disabled:text-zinc-600 disabled:opacity-50 border border-transparent text-white rounded-xl font-semibold shadow-lg shadow-blue-500/20 hover:shadow-blue-500/35 transition-all text-sm cursor-pointer"
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
              </div>

              <div>
                <h4 className="text-xs font-semibold text-zinc-400 uppercase tracking-widest mb-2">Truyền thuyết tông môn</h4>
                <p className="text-sm text-zinc-300 leading-relaxed">{sect.description}</p>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-2">
                <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-4">
                  <h5 className="text-xs font-semibold text-zinc-400 mb-2">Đặc trưng bảo khố</h5>
                  <div className="flex flex-wrap gap-2">
                    {sect.vault_gu && (sect.vault_gu as any[]).map((g: any) => {
                      const spec = guDictionary[g.id];
                      return (
                        <span key={g.id} className="text-xs bg-zinc-950 px-2.5 py-1 rounded-lg border border-zinc-800 text-zinc-300">
                          {spec?.name || g.id}
                        </span>
                      );
                    })}
                    {sect.vault_materials && Object.keys(sect.vault_materials).map((matId) => {
                      const spec = itemDictionary[matId];
                      return (
                        <span key={matId} className="text-xs bg-zinc-950 px-2.5 py-1 rounded-lg border border-zinc-800 text-zinc-300">
                          {spec?.name || matId}
                        </span>
                      );
                    })}
                  </div>
                </div>

                <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-4">
                  <h5 className="text-xs font-semibold text-zinc-400 mb-2">Đệ tử Quy chế</h5>
                  <ul className="text-xs text-zinc-400 space-y-1.5 list-disc list-inside">
                    <li>Gia nhập nhận thân phận <strong className="text-zinc-300">Ngoại môn đệ tử</strong>.</li>
                    <li>Có thể thăng cấp qua các kỳ khảo hạch tu vi.</li>
                    <li>Rời khỏi tông môn sẽ <strong className="text-red-400">mất sạch</strong> điểm cống hiến.</li>
                  </ul>
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
            <strong className="text-yellow-500 font-mono">Yêu cầu: Tu vi Ngũ Chuyển trở lên & Chi phí lập phái: 50,000 Nguyên Thạch.</strong>
          </p>

          {player.rank < 5 ? (
            <div className="p-3 bg-red-950/15 border border-red-500/20 rounded-xl text-xs text-red-400 font-mono">
              ⚠️ Đạo hữu chưa đủ tu vi! Cần đạt Ngũ Chuyển để khai sơn lập tổ (Tu vi hiện tại: Cấp {player.rank}).
            </div>
          ) : player.primeval_stones < 50000 ? (
            <div className="p-3 bg-red-950/15 border border-red-500/20 rounded-xl text-xs text-red-400 font-mono">
              ⚠️ Đạo hữu chưa đủ tài lực! Cần 50,000 Nguyên Thạch (Hiện có: {player.primeval_stones} Đá).
            </div>
          ) : !showCreateForm ? (
            <button
              onClick={() => setShowCreateForm(true)}
              className="py-2.5 px-6 bg-gradient-to-r from-yellow-600 to-amber-600 hover:from-yellow-500 hover:to-amber-500 text-white rounded-xl font-serif font-bold text-xs shadow-md transition-all self-start cursor-pointer"
            >
              Khai Sáng Thế Lực Mới
            </button>
          ) : (
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
                  <label className="text-xs text-zinc-400 font-semibold font-serif">Vực Giới Trụ Sở</label>
                  <select
                    value={createSectRegion}
                    onChange={(e) => setCreateSectRegion(e.target.value)}
                    className="bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-yellow-500/50"
                  >
                    <option value="Central_Continent">Trung Châu</option>
                    <option value="Southern_Border">Nam Cương</option>
                    <option value="Northern_Plains">Bắc Nguyên</option>
                    <option value="Western_Desert">Tây Mạc</option>
                    <option value="Eastern_Sea">Đông Hải</option>
                  </select>
                </div>

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
                  disabled={actionLoading || !createSectName.trim() || !createSectDesc.trim()}
                  className="py-2 px-4 bg-gradient-to-r from-yellow-600 to-amber-600 hover:from-yellow-500 hover:to-amber-500 disabled:opacity-50 text-white rounded-xl font-semibold text-xs shadow-md transition-all cursor-pointer"
                >
                  Xác Nhận Lập Phái (-50,000 Đá)
                </button>
                <button
                  onClick={() => setShowCreateForm(false)}
                  className="py-2 px-4 bg-zinc-900 hover:bg-zinc-800 text-zinc-400 rounded-xl font-semibold text-xs transition-colors cursor-pointer"
                >
                  Hủy
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    );
  }

  const { member, sect, leaderboard } = mySectStatus;

  // 2. STATE: MEMBER PANEL
  return (
    <div className="w-full max-w-4xl mx-auto px-2 py-4 flex flex-col gap-6 animate-fade-in">
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
            <p className="text-xs text-zinc-500 font-mono mt-0.5">
              Địa Khu: {getRegionName(sect?.region || "")} | {getRankName(member?.member_rank || "")}
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

      {/* Tabs Menu */}
      <div className="flex bg-zinc-950/50 p-1.5 rounded-2xl border border-zinc-900/60 backdrop-blur-sm self-start w-full sm:w-auto">
        <button
          onClick={() => setActiveTab("lobby")}
          className={`flex-1 sm:flex-initial px-4 py-2 text-xs font-serif font-bold rounded-xl transition-all cursor-pointer ${
            activeTab === "lobby" ? "bg-blue-600 text-white shadow-md shadow-blue-600/10" : "text-zinc-400 hover:text-zinc-200"
          }`}
        >
          Môn Đường
        </button>
        <button
          onClick={() => setActiveTab("missions")}
          className={`flex-1 sm:flex-initial px-4 py-2 text-xs font-serif font-bold rounded-xl transition-all cursor-pointer ${
            activeTab === "missions" ? "bg-blue-600 text-white shadow-md shadow-blue-600/10" : "text-zinc-400 hover:text-zinc-200"
          }`}
        >
          Bảng Nhiệm Vụ
        </button>
        <button
          onClick={() => setActiveTab("vault")}
          className={`flex-1 sm:flex-initial px-4 py-2 text-xs font-serif font-bold rounded-xl transition-all cursor-pointer ${
            activeTab === "vault" ? "bg-blue-600 text-white shadow-md shadow-blue-600/10" : "text-zinc-400 hover:text-zinc-200"
          }`}
        >
          Bảo Khố Tông Môn
        </button>
        <button
          onClick={() => setActiveTab("donate")}
          className={`flex-1 sm:flex-initial px-4 py-2 text-xs font-serif font-bold rounded-xl transition-all cursor-pointer ${
            activeTab === "donate" ? "bg-blue-600 text-white shadow-md shadow-blue-600/10" : "text-zinc-400 hover:text-zinc-200"
          }`}
        >
          Quyên Góp
        </button>
        <button
          onClick={() => setActiveTab("territory")}
          className={`flex-1 sm:flex-initial px-4 py-2 text-xs font-serif font-bold rounded-xl transition-all cursor-pointer ${
            activeTab === "territory" ? "bg-blue-600 text-white shadow-md shadow-blue-600/10" : "text-zinc-400 hover:text-zinc-200"
          }`}
        >
          Lãnh Địa
        </button>
        {member?.member_rank && (member.member_rank === "CHUONG_MON" || member.member_rank === "TRUONG_LAO") && (
          <button
            onClick={() => setActiveTab("manage")}
            className={`flex-1 sm:flex-initial px-4 py-2 text-xs font-serif font-bold rounded-xl transition-all cursor-pointer ${
              activeTab === "manage" ? "bg-blue-600 text-white shadow-md shadow-blue-600/10" : "text-zinc-400 hover:text-zinc-200"
            }`}
          >
            Quản Lý
          </button>
        )}
      </div>

      {/* Tab Panels */}
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
                <span className="text-zinc-300 font-semibold">Rank {player.rank}</span>
              </div>
            </div>

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
                        <span>Tu vi của Cổ sư (Rank):</span>
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
                  🎉 Đạo hữu đã đạt đến cấp bậc cao nhất của tông môn (Trưởng Lão)!
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
                    className={`flex justify-between items-center p-3 rounded-2xl border text-xs ${
                      entry.username === player.username
                        ? "bg-blue-950/20 border-blue-500/20 text-blue-100"
                        : "bg-zinc-900/20 border-zinc-900 text-zinc-300"
                    }`}
                  >
                    <div className="flex items-center gap-2.5 min-w-0">
                      <span className="font-mono font-bold text-[10px] text-zinc-500 w-4">
                        {idx + 1}.
                      </span>
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
          <div className="bg-zinc-950/40 border border-zinc-900 rounded-3xl p-5 backdrop-blur-md flex flex-col gap-4">
            <h3 className="text-base font-serif font-bold text-zinc-200 flex items-center gap-2 border-b border-zinc-900/60 pb-3">
              <Shield className="w-5 h-5 text-rose-400" />
              Lãnh Địa Tông Môn (Tranh Đoạt)
            </h3>
            <p className="text-xs text-zinc-400 leading-relaxed mb-2">
              Lãnh địa Tông môn là các mỏ tài nguyên quý giá sinh ra Quỹ tông môn và Linh tài mỗi giờ. 
              Các đệ tử từ <strong>Nội Môn</strong> trở lên có thể khiêu chiến Hộ Phái Thần Thú để cướp đoạt mỏ cho Tông môn của mình.
            </p>

            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              {sectMines.length === 0 ? (
                <div className="sm:col-span-2 lg:col-span-3 text-center py-6 text-zinc-600 text-xs font-mono">Chưa có Lãnh địa nào xuất hiện.</div>
              ) : (
                sectMines.map((mine) => {
                  const isMine = mine.occupier_sect_id === mySectStatus.sect?.id;
                  const isShielded = mine.shield_until && new Date(mine.shield_until).getTime() > Date.now();
                  
                  return (
                    <div key={mine.id} className={`bg-zinc-900/20 border ${isMine ? "border-blue-500/30 shadow-[0_0_15px_rgba(59,130,246,0.1)]" : "border-zinc-900"} rounded-2xl p-4 flex flex-col gap-3 relative overflow-hidden`}>
                      {isMine && <div className="absolute top-0 right-0 w-16 h-16 bg-blue-500/10 rounded-bl-full -z-10" />}
                      
                      <div className="flex justify-between items-start">
                        <h4 className="text-sm font-bold font-serif text-zinc-200">{mine.name}</h4>
                        <span className={`text-[10px] font-mono px-2 py-0.5 rounded border ${
                          mine.rarity === "Truyền Thuyết" ? "bg-amber-950/30 text-amber-400 border-amber-500/30" :
                          mine.rarity === "Sử Thi" ? "bg-purple-950/30 text-purple-400 border-purple-500/30" :
                          mine.rarity === "Hiếm" ? "bg-blue-950/30 text-blue-400 border-blue-500/30" :
                          "bg-zinc-900/50 text-zinc-400 border-zinc-700"
                        }`}>{mine.rarity}</span>
                      </div>

                      <div className="flex flex-col gap-1 text-xs font-mono">
                        <div className="flex justify-between">
                          <span className="text-zinc-500">Chiếm giữ bởi:</span>
                          <span className={isMine ? "text-blue-400 font-bold" : "text-zinc-300"}>
                            {mine.occupier_name || "Vô Chủ"}
                          </span>
                        </div>
                        {mine.occupied_at && (
                          <div className="flex justify-between">
                            <span className="text-zinc-500">Từ lúc:</span>
                            <span className="text-zinc-400">{new Date(mine.occupied_at).toLocaleTimeString()}</span>
                          </div>
                        )}
                        {isShielded && (
                          <div className="flex justify-between text-rose-400">
                            <span>Đại Trận Bảo Hộ:</span>
                            <span>{new Date(mine.shield_until!).toLocaleTimeString()}</span>
                          </div>
                        )}
                      </div>

                      <div className="pt-3 mt-auto border-t border-zinc-900 flex gap-2">
                        {isMine ? (
                          (mySectStatus.member?.member_rank === "CHUONG_MON" || mySectStatus.member?.member_rank === "TRUONG_LAO") ? (
                            <button
                              onClick={() => claimSectMineRewards(mine.id)}
                              disabled={actionLoading}
                              className="w-full py-2 bg-emerald-600/20 hover:bg-emerald-600/30 text-emerald-400 border border-emerald-500/30 rounded-xl text-xs font-bold transition-colors cursor-pointer"
                            >
                              Thu Hoạch Mỏ
                            </button>
                          ) : (
                            <div className="w-full py-2 text-center text-emerald-500/70 text-xs font-mono border border-emerald-500/10 rounded-xl bg-emerald-950/10">
                              Đang Khai Thác
                            </div>
                          )
                        ) : (
                          <button
                            onClick={() => attackSectMine(mine.id)}
                            disabled={actionLoading || isShielded || player.active_combat !== null}
                            className="w-full py-2 bg-rose-600/20 hover:bg-rose-600/30 text-rose-400 border border-rose-500/30 disabled:opacity-50 disabled:border-zinc-800 disabled:bg-zinc-900 disabled:text-zinc-600 rounded-xl text-xs font-bold transition-colors cursor-pointer"
                          >
                            {isShielded ? "Đang Được Bảo Hộ" : player.active_combat ? "Đang Chiến Đấu" : "Công Kích Lãnh Địa"}
                          </button>
                        )}
                      </div>
                    </div>
                  );
                })
              )}
            </div>
          </div>
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
                    <button
                      onClick={async () => {
                        await completeSectMission(am.id);
                      }}
                      disabled={!isReadyToComplete || actionLoading}
                      className="w-full sm:w-auto px-5 py-2 bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 disabled:from-zinc-900 disabled:to-zinc-950 disabled:border-zinc-800 disabled:text-zinc-600 disabled:opacity-50 text-white rounded-xl font-semibold text-xs shadow-lg transition-all cursor-pointer flex items-center justify-center gap-1.5"
                    >
                      <CheckCircle className="w-4 h-4" />
                      Nộp Nhiệm Vụ
                    </button>
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
                  };
                  const currentMemberRankVal = rankMap[member?.member_rank || "NGOAI_MON"] || 1;
                  const reqRankVal = rankMap[mission.required_member_rank] || 1;
                  const rankTooLow = currentMemberRankVal < reqRankVal;

                  return (
                    <div
                      key={mission.id}
                      className={`bg-zinc-900/20 border ${
                        rankTooLow ? "border-zinc-950 opacity-60" : "border-zinc-900 hover:border-zinc-800"
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
                        <h4 className="text-sm font-serif font-bold text-zinc-100 mb-1">{mission.title}</h4>
                        <p className="text-xs text-zinc-500 leading-relaxed">{mission.description}</p>
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
                className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${
                  vaultTab === "gu" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"
                }`}
              >
                Cổ Trùng
              </button>
              <button
                onClick={() => setVaultTab("material")}
                className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${
                  vaultTab === "material" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"
                }`}
              >
                Linh Tài
              </button>
              <button
                onClick={() => setVaultTab("recipe")}
                className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${
                  vaultTab === "recipe" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"
                }`}
              >
                Cổ Phương
              </button>
            </div>
          </div>

          {/* Vault Inventory Render */}
          <div>
            {vaultTab === "gu" && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {sect?.vault_gu && (sect.vault_gu as any[]).length > 0 ? (
                  (sect.vault_gu as any[]).map((vg: any) => {
                    const spec = guDictionary[vg.id];
                    const outOfStock = vg.quantity !== undefined && vg.quantity <= 0;
                    const unaffordable = (member?.contribution_points || 0) < vg.cost;
                    
                    return (
                      <div
                        key={vg.id}
                        className="bg-zinc-900/20 border border-zinc-900 hover:border-zinc-800/80 rounded-2xl p-4 flex items-center justify-between gap-4 transition-all duration-300"
                      >
                        <div className="flex flex-col min-w-0">
                          <h4 className="text-sm font-serif font-bold text-zinc-100">
                            {spec?.name || vg.id}
                          </h4>
                          <span className="text-[10px] text-zinc-500 font-mono mt-0.5">
                            Phân loại: {spec?.gu_type || "Cổ"} | Rank: {spec?.rank || 1} | Số lượng: {vg.quantity !== undefined ? `${vg.quantity} con` : "Vô hạn"}
                          </span>
                          <p className="text-xs text-zinc-500 leading-snug line-clamp-2 mt-1.5">
                            {spec?.description || "Không có mô tả bí thuật..."}
                          </p>
                        </div>

                        <div className="flex flex-col items-end gap-2 shrink-0">
                          <div className="text-right">
                            <span className="text-[9px] text-zinc-500 font-mono block">Cống hiến cần</span>
                            <span className="text-xs font-mono font-bold text-yellow-500">{vg.cost} Pts</span>
                          </div>
                          <button
                            onClick={async () => {
                              await buyFromSectVault("gu", vg.id);
                            }}
                            disabled={unaffordable || outOfStock || actionLoading}
                            className="px-3 py-1.5 bg-blue-600 hover:bg-blue-500 disabled:bg-zinc-950 disabled:border-zinc-900 disabled:text-zinc-600 disabled:opacity-50 text-white font-semibold text-xs rounded-xl shadow-md transition-colors cursor-pointer"
                          >
                            {outOfStock ? "Hết Hàng" : "Đổi Cổ"}
                          </button>
                        </div>
                      </div>
                    );
                  })
                ) : (
                  <div className="col-span-2 text-center py-6 text-zinc-600 text-xs font-mono">Bảo khố hiện tại không còn Cổ trùng nào bán</div>
                )}
              </div>
            )}

            {vaultTab === "material" && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {sect?.vault_materials && Object.entries(sect.vault_materials).length > 0 ? (
                  Object.entries(sect.vault_materials).map(([matId, qty]) => {
                    const spec = itemDictionary[matId];
                    const cost = getVaultCost(getItemBasePrice(matId));
                    const unaffordable = (member?.contribution_points || 0) < cost;
                    
                    return (
                      <div
                        key={matId}
                        className="bg-zinc-900/20 border border-zinc-900 hover:border-zinc-800/80 rounded-2xl p-4 flex items-center justify-between gap-4 transition-all duration-300"
                      >
                        <div className="flex flex-col min-w-0">
                          <h4 className="text-sm font-serif font-bold text-zinc-100">
                            {spec?.name || matId}
                          </h4>
                          <span className="text-[10px] text-zinc-500 font-mono mt-0.5">
                            Còn lại bảo khố: {qty}x
                          </span>
                          <p className="text-xs text-zinc-500 leading-snug line-clamp-2 mt-1.5">
                            {spec?.description || "Nguyên liệu chế tác / luyện cổ trùng..."}
                          </p>
                        </div>

                        <div className="flex flex-col items-end gap-2 shrink-0">
                          <div className="text-right">
                            <span className="text-[9px] text-zinc-500 font-mono block">Cống hiến cần</span>
                            <span className="text-xs font-mono font-bold text-yellow-500">{cost} Pts</span>
                          </div>
                          <button
                            onClick={async () => {
                              await buyFromSectVault("material", matId);
                            }}
                            disabled={unaffordable || qty <= 0 || actionLoading}
                            className="px-3 py-1.5 bg-blue-600 hover:bg-blue-500 disabled:bg-zinc-950 disabled:border-zinc-900 disabled:text-zinc-600 disabled:opacity-50 text-white font-semibold text-xs rounded-xl shadow-md transition-colors cursor-pointer"
                          >
                            Đổi Linh Tài
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
                {sect?.vault_recipes && (sect.vault_recipes as string[]).length > 0 ? (
                  (sect.vault_recipes as string[]).map((recipeId) => {
                    const recipeInfo = recipes?.find(r => r.id === recipeId);
                    const targetGuSpec = recipeInfo ? guDictionary[recipeInfo.target_gu_id] : null;
                    const cost = getRecipeCost(targetGuSpec?.rank || 1);
                    const unaffordable = (member?.contribution_points || 0) < cost;
                    
                    return (
                      <div
                        key={recipeId}
                        className="bg-zinc-900/20 border border-zinc-900 hover:border-zinc-800/80 rounded-2xl p-4 flex items-center justify-between gap-4 transition-all duration-300"
                      >
                        <div className="flex flex-col min-w-0">
                          <h4 className="text-sm font-serif font-bold text-zinc-100">
                            Bí Phương: {recipeId.replace("recipe_", "").toUpperCase()}
                          </h4>
                          <span className="text-[10px] text-zinc-500 font-mono mt-0.5">
                            Bao gồm: Nguyên liệu & Tỷ lệ luyện chế thành công môn phái đúc rút.
                          </span>
                          <p className="text-xs text-zinc-500 leading-snug line-clamp-2 mt-1.5">
                            Giải khóa công thức hợp thành tại lò luyện Cổ trùng của bạn.
                          </p>
                        </div>

                        <div className="flex flex-col items-end gap-2 shrink-0">
                          <div className="text-right">
                            <span className="text-[9px] text-zinc-500 font-mono block">Cống hiến cần</span>
                            <span className="text-xs font-mono font-bold text-yellow-500">{cost} Pts</span>
                          </div>
                          <button
                            onClick={async () => {
                              await buyFromSectVault("recipe", recipeId);
                            }}
                            disabled={unaffordable || actionLoading}
                            className="px-3 py-1.5 bg-blue-600 hover:bg-blue-500 disabled:bg-zinc-950 disabled:border-zinc-900 disabled:text-zinc-600 disabled:opacity-50 text-white font-semibold text-xs rounded-xl shadow-md transition-colors cursor-pointer"
                          >
                            Học Công Thức
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
                className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${
                  donateType === "stones" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"
                }`}
              >
                Nguyên Thạch
              </button>
              <button
                onClick={() => setDonateType("materials")}
                className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${
                  donateType === "materials" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"
                }`}
              >
                Linh Tài
              </button>
              <button
                onClick={() => setDonateType("gu")}
                className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${
                  donateType === "gu" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"
                }`}
              >
                Cổ Trùng
              </button>
            </div>
          </div>

          {/* Donate Stone Panel */}
          {donateType === "stones" && (
            <div className="bg-zinc-900/10 border border-zinc-900 rounded-2xl p-5 flex flex-col gap-4 max-w-lg mx-auto w-full">
              <div className="flex justify-between items-center text-xs">
                <span className="text-zinc-500 font-mono">Nguyên thạch của đạo hữu:</span>
                <span className="font-mono font-bold text-yellow-500">{player.primeval_stones} Đá</span>
              </div>

              <div className="flex flex-col gap-1.5">
                <label className="text-xs text-zinc-400 font-serif font-bold">Số lượng quyên góp</label>
                <div className="relative">
                  <input
                    type="number"
                    value={donateStones}
                    onChange={(e) => setDonateStones(Math.max(1, parseInt(e.target.value) || 0))}
                    className="w-full bg-zinc-950 border border-zinc-900 rounded-xl px-4 py-2.5 text-sm font-mono text-zinc-100 focus:outline-none focus:border-blue-500/50"
                  />
                  <button
                    onClick={() => setDonateStones(player.primeval_stones)}
                    className="absolute right-2.5 top-1/2 -translate-y-1/2 text-[10px] bg-zinc-900 border border-zinc-800 text-zinc-400 hover:text-zinc-200 px-2 py-1 rounded-md transition-colors"
                  >
                    Tối đa
                  </button>
                </div>
                <span className="text-[10px] text-zinc-500 font-mono mt-0.5">
                  Tỷ giá quyên góp: 1 Nguyên thạch = 1 Điểm cống hiến
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
                className="w-full py-2.5 bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 disabled:opacity-50 text-white rounded-xl font-semibold text-sm shadow-md transition-all cursor-pointer flex items-center justify-center gap-1.5"
              >
                <Gift className="w-4 h-4" />
                Dâng Nộp Linh Thạch (Nhận +{donateStones} Pts)
              </button>
            </div>
          )}

          {/* Donate Material Panel */}
          {donateType === "materials" && (
            <div className="bg-zinc-900/10 border border-zinc-900 rounded-2xl p-5 flex flex-col gap-4 max-w-lg mx-auto w-full">
              <div className="flex flex-col gap-1.5">
                <label className="text-xs text-zinc-400 font-serif font-bold">Chọn linh tài quyên góp</label>
                <select
                  value={selectedMatId}
                  onChange={(e) => {
                    setSelectedMatId(e.target.value);
                    setDonateMatAmount(1);
                  }}
                  className="w-full bg-zinc-950 border border-zinc-900 rounded-xl px-4 py-2.5 text-sm text-zinc-100 focus:outline-none focus:border-blue-500/50"
                >
                  <option value="">-- Chọn một loại linh tài --</option>
                  {playerMaterials.map(([matId, qty]) => {
                    const spec = itemDictionary[matId];
                    return (
                      <option key={matId} value={matId}>
                        {spec?.name || matId} (Hiện có: {qty}x | Trị giá: {getMaterialValue(matId)} Pts/Cái)
                      </option>
                    );
                  })}
                </select>
              </div>

              {selectedMatId && (() => {
                const maxQty = player.materials_inventory[selectedMatId] || 0;
                const valuePerItem = getMaterialValue(selectedMatId);
                const totalGainedPoints = donateMatAmount * valuePerItem;

                return (
                  <>
                    <div className="flex flex-col gap-1.5">
                      <label className="text-xs text-zinc-400 font-serif font-bold">Số lượng nộp</label>
                      <div className="relative">
                        <input
                          type="number"
                          value={donateMatAmount}
                          onChange={(e) => setDonateMatAmount(Math.max(1, Math.min(maxQty, parseInt(e.target.value) || 1)))}
                          className="w-full bg-zinc-950 border border-zinc-900 rounded-xl px-4 py-2.5 text-sm font-mono text-zinc-100 focus:outline-none focus:border-blue-500/50"
                        />
                        <button
                          onClick={() => setDonateMatAmount(maxQty)}
                          className="absolute right-2.5 top-1/2 -translate-y-1/2 text-[10px] bg-zinc-900 border border-zinc-800 text-zinc-400 hover:text-zinc-200 px-2 py-1 rounded-md transition-colors"
                        >
                          Tối đa
                        </button>
                      </div>
                    </div>

                    <button
                      onClick={async () => {
                        if (donateMatAmount <= 0 || donateMatAmount > maxQty) {
                          addToast("Số lượng quyên góp không hợp lệ!", "error");
                          return;
                        }
                        const ok = await donateSectMaterials(selectedMatId, donateMatAmount);
                        if (ok) setSelectedMatId("");
                      }}
                      disabled={actionLoading}
                      className="w-full py-2.5 bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 disabled:opacity-50 text-white rounded-xl font-semibold text-sm shadow-md transition-all cursor-pointer flex items-center justify-center gap-1.5"
                    >
                      <Gift className="w-4 h-4" />
                      Quyên Góp Linh Tài (Nhận +{totalGainedPoints} Pts)
                    </button>
                  </>
                );
              })()}
            </div>
          )}

          {/* Donate Gu Panel */}
          {donateType === "gu" && (
            <div className="bg-zinc-900/10 border border-zinc-900 rounded-2xl p-5 flex flex-col gap-4 max-w-lg mx-auto w-full">
              <div className="flex flex-col gap-1.5">
                <label className="text-xs text-zinc-400 font-serif font-bold">Chọn Cổ trùng trong túi đồ</label>
                <select
                  value={selectedGuInstId}
                  onChange={(e) => setSelectedGuInstId(e.target.value)}
                  className="w-full bg-zinc-950 border border-zinc-900 rounded-xl px-4 py-2.5 text-sm text-zinc-100 focus:outline-none focus:border-blue-500/50"
                >
                  <option value="">-- Chọn Cổ trùng quyên hiến --</option>
                  {playerDonatableGus.map((g) => {
                    const spec = guDictionary[g.gu_id];
                    const rank = spec?.rank || 1;
                    const val = getGuValue(g.gu_id);
                    return (
                      <option key={g.instance_id} value={g.instance_id}>
                        {spec?.name || g.gu_id} (Rank: {rank} | Nhận: {val} Pts)
                      </option>
                    );
                  })}
                </select>
                <span className="text-[10px] text-zinc-500 font-mono mt-0.5 leading-relaxed">
                  ⚠️ Lưu ý: Bản Mệnh Cổ và Cổ Trùng đang trang bị không thể quyên góp môn phái.
                </span>
              </div>

              {selectedGuInstId && (() => {
                const targetGu = playerDonatableGus.find(g => g.instance_id === selectedGuInstId);
                if (!targetGu) return null;
                const spec = guDictionary[targetGu.gu_id];
                const value = getGuValue(targetGu.gu_id);

                return (
                  <button
                    onClick={async () => {
                      const ok = await donateSectGu(selectedGuInstId);
                      if (ok) setSelectedGuInstId("");
                    }}
                    disabled={actionLoading}
                    className="w-full py-2.5 bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 disabled:opacity-50 text-white rounded-xl font-semibold text-sm shadow-md transition-all cursor-pointer flex items-center justify-center gap-1.5"
                  >
                    <Gift className="w-4 h-4" />
                    Hiến Tế Cổ Trùng (Nhận +{value} Pts)
                  </button>
                );
              })()}
            </div>
          )}
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
              {(mySectStatus?.member?.member_rank === "CHUONG_MON" || mySectStatus?.member?.member_rank === "TRUONG_LAO") && (
                <div className="flex items-center gap-2 mt-2 bg-zinc-900/40 border border-zinc-800/80 px-3 py-1.5 rounded-xl w-fit">
                  <span className="text-[11px] text-zinc-300 font-semibold">Duyệt Đệ Tử Khi Gia Nhập:</span>
                  <button
                    type="button"
                    onClick={async () => {
                      await toggleSectApproval(!mySectStatus?.sect?.require_approval);
                    }}
                    disabled={actionLoading}
                    className={`w-9 h-5 flex items-center rounded-full p-1 cursor-pointer transition-colors duration-300 ${
                      mySectStatus?.sect?.require_approval ? "bg-emerald-600" : "bg-zinc-800"
                    }`}
                  >
                    <div
                      className={`bg-white w-3 h-3 rounded-full shadow-md transform transition-transform duration-300 ${
                        mySectStatus?.sect?.require_approval ? "translate-x-4" : "translate-x-0"
                      }`}
                    />
                  </button>
                </div>
              )}
            </div>

            <div className="flex flex-wrap bg-zinc-900/50 p-1 rounded-xl border border-zinc-800/60 self-start sm:self-auto gap-1">
              <button
                onClick={() => setMgmtTab("members")}
                className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${
                  mgmtTab === "members" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"
                }`}
              >
                Đệ Tử Danh Sách
              </button>
              <button
                onClick={() => setMgmtTab("missions")}
                className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${
                  mgmtTab === "missions" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"
                }`}
              >
                Tạo Nhiệm Vụ
              </button>
              <button
                onClick={() => setMgmtTab("stock")}
                className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer ${
                  mgmtTab === "stock" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"
                }`}
              >
                Nhập Kho & Tăng Cấp
              </button>
              {(mySectStatus?.member?.member_rank === "CHUONG_MON" || mySectStatus?.member?.member_rank === "TRUONG_LAO") && (
                <button
                  onClick={() => setMgmtTab("requests")}
                  className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all cursor-pointer relative ${
                    mgmtTab === "requests" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:text-zinc-200"
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
                    const targetIsElderOrLeader = mInfo.member_rank === "CHUONG_MON" || mInfo.member_rank === "TRUONG_LAO";
                    
                    return (
                      <div
                        key={mInfo.player_id}
                        className="bg-zinc-900/20 border border-zinc-900 rounded-2xl p-4 flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 text-xs"
                      >
                        <div className="flex flex-col gap-1 min-w-0">
                          <div className="flex items-center gap-2">
                            <span
                              onClick={() => !isSelf && fetchPlayerProfile(mInfo.username)}
                              className={`font-semibold text-zinc-100 truncate ${!isSelf ? "cursor-pointer hover:text-blue-450 hover:underline decoration-dotted" : ""}`}
                              title={!isSelf ? "Bấm để trinh thám cổ sư" : undefined}
                            >
                              {mInfo.username}
                            </span>
                            <span className={`px-2 py-0.5 text-[9px] font-mono rounded border ${getRankColor(mInfo.member_rank)}`}>
                              {getRankName(mInfo.member_rank)}
                            </span>
                            {isSelf && <span className="text-[9px] text-blue-400 font-mono font-semibold">(Đạo hữu)</span>}
                          </div>
                          <div className="text-[10px] text-zinc-500 font-mono">
                            Tu vi: Rank {mInfo.player_rank} | Quyên hiến tích lũy: <strong className="text-yellow-500">{mInfo.contribution_points} Pts</strong>
                          </div>
                        </div>

                        {/* Action buttons */}
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
                                className="bg-zinc-950 border border-zinc-900 rounded-xl px-2.5 py-1 text-[11px] text-zinc-300 focus:outline-none focus:border-yellow-500/50"
                              >
                                <option value="NGOAI_MON">Ngoại Môn</option>
                                <option value="NOI_MON">Nội Môn</option>
                                <option value="TINH_ANH">Tinh Anh</option>
                                <option value="CHAN_TRUYEN">Chân Truyền</option>
                                <option value="TRUONG_LAO">Trưởng Lão</option>
                                <option value="CHUONG_MON">Nhường Ngôi Chưởng Môn</option>
                              </select>
                            )}

                            {/* Expel button */}
                            {((callerIsLeader) || (member?.member_rank === "TRUONG_LAO" && !targetIsElderOrLeader)) && (
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
                        <div className="flex flex-col gap-1 min-w-0">
                          <span
                            onClick={() => reqInfo.username && fetchPlayerProfile(reqInfo.username)}
                            className="font-semibold text-zinc-100 truncate cursor-pointer hover:text-blue-450 hover:underline decoration-dotted"
                            title="Bấm để trinh thám cổ sư"
                          >
                            {reqInfo.username}
                          </span>
                          <div className="text-[10px] text-zinc-500 font-mono">
                            Tu vi: Rank {reqInfo.player_rank} | Gửi lúc: {new Date(reqInfo.created_at).toLocaleString()}
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
                        <div className="flex flex-col gap-1 min-w-0">
                          <span
                            onClick={() => reqInfo.username && fetchPlayerProfile(reqInfo.username)}
                            className="font-semibold text-zinc-100 truncate cursor-pointer hover:text-blue-450 hover:underline decoration-dotted"
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

          {/* Create Mission Tab */}
          {mgmtTab === "missions" && (
            <form onSubmit={handleCreateCustomMission} className="bg-zinc-900/10 border border-zinc-900 rounded-2xl p-5 flex flex-col gap-4 max-w-lg mx-auto w-full">
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
                    className="bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs text-zinc-100 focus:outline-none"
                  >
                    <option value="GATHER">Thu thập Linh tài (GATHER)</option>
                    <option value="DONATE">Cống hiến Cổ trùng (DONATE)</option>
                  </select>
                </div>

                <div className="flex flex-col gap-1.5">
                  <label className="text-xs text-zinc-400 font-serif font-bold">Mã Mục Tiêu (Target ID)</label>
                  <input
                    type="text"
                    required
                    placeholder="Ví dụ: guts_gu hoặc GU_WHITE_BOAR"
                    value={mTargetId}
                    onChange={(e) => setMTargetId(e.target.value)}
                    className="bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs text-zinc-100 focus:outline-none"
                  />
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

                <div className="sm:col-span-2 flex flex-col gap-1.5">
                  <label className="text-xs text-zinc-400 font-serif font-bold">Cấp bậc đệ tử yêu cầu nhận</label>
                  <select
                    value={mRequiredRank}
                    onChange={(e) => setMRequiredRank(e.target.value)}
                    className="bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs text-zinc-100 focus:outline-none"
                  >
                    <option value="NGOAI_MON">Đệ Tử Ngoại Môn</option>
                    <option value="NOI_MON">Đệ Tử Nội Môn</option>
                    <option value="TINH_ANH">Đệ Tử Tinh Anh</option>
                    <option value="CHAN_TRUYEN">Đệ Tử Chân Truyền</option>
                    <option value="TRUONG_LAO">Trưởng Lão</option>
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
              <div className="bg-zinc-900/10 border border-zinc-900 rounded-2xl p-5 flex flex-col gap-4">
                <h4 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-1.5">
                  <Plus className="w-4 h-4 text-cyan-400" />
                  Mua Sắm Vật Phẩm (Nhập Kho Bảo Khố)
                </h4>
                <p className="text-xs text-zinc-500">
                  Dùng quỹ tông môn để mua thêm Cổ trùng hoặc Linh tài đưa vào bảo khố để đệ tử trao đổi.
                </p>

                <div className="flex flex-col gap-3">
                  <div className="flex flex-col gap-1">
                    <label className="text-[10px] text-zinc-500 font-bold uppercase tracking-wider">Loại vật phẩm</label>
                    <select
                      value={stockItemType}
                      onChange={(e) => {
                        const type = e.target.value as "gu" | "material";
                        setStockItemType(type);
                        const sectLvl = sect?.level || 1;
                        if (type === "gu") {
                          const firstAvailable = Object.entries(guDictionary || {}).find(([, spec]) => (spec.rank || 1) <= sectLvl);
                          setStockItemId(firstAvailable ? firstAvailable[0] : "");
                        } else {
                          const firstAvailable = Object.entries(itemDictionary || {}).find(([, spec]) => (spec.rank || 1) <= sectLvl);
                          setStockItemId(firstAvailable ? firstAvailable[0] : "");
                        }
                      }}
                      className="bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs text-zinc-100 focus:outline-none cursor-pointer"
                    >
                      <option value="gu">Cổ Trùng</option>
                      <option value="material">Linh Tài (Material)</option>
                    </select>
                  </div>

                  <div className="flex flex-col gap-1">
                    <label className="text-[10px] text-zinc-500 font-bold uppercase tracking-wider">
                      Chọn vật phẩm <span className="text-cyan-500 normal-case">(Giới hạn: Hạng ≤ Cấp {sect?.level || 1})</span>
                    </label>
                    <select
                      value={stockItemId}
                      onChange={(e) => setStockItemId(e.target.value)}
                      className="bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs text-zinc-100 focus:outline-none cursor-pointer"
                    >
                      {stockItemType === "gu" ? (
                        Object.entries(guDictionary || {}).map(([id, spec]) => {
                          const locked = (spec.rank || 1) > (sect?.level || 1);
                          return (
                            <option key={id} value={id} disabled={locked}>
                              {locked ? "🔒 " : ""}{spec.name} (Hạng {spec.rank}){locked ? ` — Cần Cấp ${spec.rank}` : ""}
                            </option>
                          );
                        })
                      ) : (
                        Object.entries(itemDictionary || {}).map(([id, spec]) => {
                          const locked = (spec.rank || 1) > (sect?.level || 1);
                          return (
                            <option key={id} value={id} disabled={locked}>
                              {locked ? "🔒 " : ""}{spec.name} (Hạng {spec.rank || 1}){locked ? ` — Cần Cấp ${spec.rank || 1}` : ""}
                            </option>
                          );
                        })
                      )}
                    </select>
                    {stockItemType === "gu" && Object.values(guDictionary || {}).some(s => (s.rank || 1) > (sect?.level || 1)) && (
                      <p className="text-[10px] text-amber-500/80 mt-0.5">⚠ Một số Cổ trùng bị khóa do cấp tông môn chưa đủ. Quyên góp từ đệ tử sẽ không bị giới hạn.</p>
                    )}
                  </div>

                  <div className="flex flex-col gap-1">
                    <label className="text-[10px] text-zinc-500 font-bold uppercase tracking-wider">Số lượng nhập</label>
                    <input
                      type="number"
                      min={1}
                      value={stockQuantity}
                      onChange={(e) => setStockQuantity(Math.max(1, parseInt(e.target.value) || 1))}
                      className="bg-zinc-950 border border-zinc-900 rounded-xl px-3 py-2 text-xs text-zinc-100 focus:outline-none font-mono"
                    />
                  </div>

                  {(() => {
                    if (!stockItemId) return null;
                    const sectLvl = sect?.level || 1;
                    let unitCost = 0;
                    let isLocked = false;
                    if (stockItemType === "gu") {
                      const spec = guDictionary[stockItemId];
                      const rank = spec?.rank || 1;
                      isLocked = rank > sectLvl;
                      unitCost = rank * 100;
                    } else {
                      const spec = itemDictionary[stockItemId];
                      const rank = spec?.rank || 1;
                      isLocked = rank > sectLvl;
                      unitCost = rank * 50;
                    }
                    const totalCost = unitCost * stockQuantity;
                    const canAfford = (sect?.funds || 0) >= totalCost;

                    return (
                      <div className="flex flex-col gap-3 mt-2">
                        <div className="flex justify-between items-center text-xs border-t border-zinc-900/60 pt-3">
                          <span className="text-zinc-500 font-mono">Đơn giá:</span>
                          <span className="text-zinc-300 font-mono">{unitCost} Quỹ / chiếc</span>
                        </div>
                        <div className="flex justify-between items-center text-xs">
                          <span className="text-zinc-500 font-mono">Tổng chi phí:</span>
                          <span className={`font-mono font-bold ${canAfford && !isLocked ? "text-cyan-400" : "text-red-400"}`}>
                            {totalCost.toLocaleString()} Quỹ
                          </span>
                        </div>

                        {isLocked ? (
                          <div className="w-full py-2.5 bg-zinc-800 text-zinc-500 rounded-xl font-serif font-bold text-xs text-center">
                            🔒 Tông môn chưa đủ cấp để nhập vật phẩm này
                          </div>
                        ) : (
                          <button
                            type="button"
                            onClick={async () => {
                              if (!canAfford || isLocked) return;
                              await stockSectVault(stockItemType, stockItemId, stockQuantity);
                            }}
                            disabled={!canAfford || actionLoading || isLocked}
                            className="w-full py-2.5 bg-gradient-to-r from-cyan-600 to-sky-600 hover:from-cyan-500 hover:to-sky-500 disabled:opacity-50 text-white rounded-xl font-serif font-bold text-xs shadow-md transition-all cursor-pointer flex items-center justify-center gap-1.5"
                          >
                            <Plus className="w-3.5 h-3.5" />
                            Nhập Bảo Khố
                          </button>
                        )}
                      </div>
                    );
                  })()}
                </div>
              </div>

              {/* Upgrade Sect Card */}
              <div className="bg-zinc-900/10 border border-zinc-900 rounded-2xl p-5 flex flex-col justify-between gap-4">
                <div>
                  <h4 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-1.5">
                    <Sparkles className="w-4 h-4 text-yellow-500" />
                    Thăng Cấp Tông Môn
                  </h4>
                  <p className="text-xs text-zinc-500 mt-1">
                    Nâng cao vị thế và uy danh của thế lực tông môn trên tiên giới.
                  </p>

                  <div className="bg-zinc-950/40 border border-zinc-900 rounded-xl p-3.5 mt-4 flex items-center justify-between">
                    <div>
                      <span className="text-[10px] text-zinc-500 font-mono uppercase tracking-wider block">Cấp độ hiện tại</span>
                      <span className="text-xl font-serif font-bold text-zinc-200">Cấp {sect?.level || 1}</span>
                    </div>
                    <ArrowRight className="w-5 h-5 text-zinc-700" />
                    <div className="text-right">
                      <span className="text-[10px] text-zinc-500 font-mono uppercase tracking-wider block">Cấp độ tiếp theo</span>
                      <span className="text-xl font-serif font-bold text-yellow-500">Cấp {(sect?.level || 1) + 1}</span>
                    </div>
                  </div>
                </div>

                {(() => {
                  const currentLvl = sect?.level || 1;
                  const cost = currentLvl * 10000;
                  const canAfford = (sect?.funds || 0) >= cost;

                  return (
                    <div className="flex flex-col gap-3">
                      <div className="flex justify-between items-center text-xs">
                        <span className="text-zinc-500 font-mono">Chi phí nâng cấp:</span>
                        <span className={`font-mono font-bold ${canAfford ? "text-yellow-500" : "text-red-400"}`}>
                          {cost.toLocaleString()} Quỹ
                        </span>
                      </div>

                      <button
                        type="button"
                        onClick={async () => {
                          if (!canAfford) return;
                          await upgradeSect();
                        }}
                        disabled={!canAfford || actionLoading}
                        className="w-full py-2.5 bg-gradient-to-r from-yellow-600 to-amber-600 hover:from-yellow-500 hover:to-amber-500 disabled:opacity-50 text-white rounded-xl font-serif font-bold text-xs shadow-md transition-all cursor-pointer flex items-center justify-center gap-1.5"
                      >
                        <Sparkles className="w-3.5 h-3.5" />
                        Tăng Cấp Thế Lực
                      </button>
                    </div>
                  );
                })()}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
};
