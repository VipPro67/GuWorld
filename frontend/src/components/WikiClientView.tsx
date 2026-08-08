"use client";

/* eslint-disable @typescript-eslint/no-explicit-any */

import React, { useState, useMemo } from "react";
import Link from "next/link";
import { 
  Search, 
  BookOpen, 
  ShieldAlert, 
  Layers, 
  Cpu, 
  Sparkles, 
  Coins, 
  ChevronRight, 
  SlidersHorizontal,
  ArrowLeft
} from "lucide-react";
import { type GuDictionaryItem } from "@/lib/gu-dict";
import { type ItemSpec, type RefiningRecipe } from "@/store/useGameStore";
import { getPathColor } from "@/lib/utils";

// Types
type TabType = "guides" | "gus" | "items" | "recipes";

interface Props {
  initialGus: GuDictionaryItem[];
  initialItems: ItemSpec[];
  initialRecipes: RefiningRecipe[];
  guideSections: any[];
  roadmap: any;
  subByParent: Record<string, any[]>;
  sectionIcon: Record<string, string>;
}

// Translations and Mappings
const rarityColors: Record<string, string> = {
  COMMON: "text-zinc-400 border-zinc-700/50 bg-zinc-950/20",
  UNCOMMON: "text-emerald-400 border-emerald-500/20 bg-emerald-950/10",
  RARE: "text-sky-400 border-sky-500/20 bg-sky-950/10",
  EPIC: "text-purple-400 border-purple-500/20 bg-purple-950/10",
  LEGENDARY: "text-amber-400 border-amber-500/20 bg-amber-950/10",
  MYTHIC: "text-rose-400 border-rose-500/20 bg-rose-950/10",
};

const rarityLabels: Record<string, string> = {
  COMMON: "Phàm",
  UNCOMMON: "Hiếm",
  RARE: "Quý",
  EPIC: "Sử thi",
  LEGENDARY: "Huyền thoại",
  MYTHIC: "Thần thoại",
};

const pathLabels: Record<string, string> = {
  "Blade Path": "Đao Đạo",
  "Blood Path": "Huyết Đạo",
  "Bone Path": "Cốt Đạo",
  "Cloud Path": "Vân Đạo",
  "Concealment Path": "Ẩn Nặc Đạo",
  "Creation Path": "Tạo Hóa Đạo",
  "Dark Path": "Hắc Ám Đạo",
  "Death Path": "Tử Vong Đạo",
  "Destruction Path": "Hủy Diệt Đạo",
  "Dream Path": "Mộng Đạo",
  "Earth Path": "Thổ Đạo",
  "Enchantment Path": "Phù Chú Đạo",
  "Enslavement Path": "Nô Dịch Đạo",
  "Fire Path": "Hỏa Đạo",
  "Food Path": "Thực Đạo",
  "Formation Path": "Trận Pháp Đạo",
  "Heaven Path": "Thiên Đạo",
  "Human Path": "Nhân Đạo",
  "Ice Path": "Hàn Băng Đạo",
  "Illusion Path": "Ảo Giác Đạo",
  "Information Path": "Trí Đạo",
  "Investigation Path": "Trinh Sát Đạo",
  "Life Path": "Sinh Đạo",
  "Light Path": "Quang Minh Đạo",
  "Lightning Path": "Lôi Đạo",
  "Luck Path": "Khí Vận Đạo",
  "Metal Path": "Kim Đạo",
  "Moon Path": "Nguyệt Đạo",
  "Movement Path": "Di Động Đạo",
  "Poison Path": "Độc Đạo",
  "Qi Path": "Khí Đạo",
  "Refinement Path": "Tinh Luyện Đạo",
  "Rule Path": "Quy Tắc Đạo",
  "Shadow Path": "Ảnh Đạo",
  "Smoke Path": "Yên Vụ Đạo",
  "Soul Path": "Hồn Đạo",
  "Sound Path": "Âm Thanh Đạo",
  "Space Path": "Không Gian Đạo",
  "Strength Path": "Lực Đạo",
  "Sun Path": "Nhật Đạo",
  "Sword Path": "Kiếm Đạo",
  "Theft Path": "Trộm Đạo",
  "Theme Path": "Chủ Đề Đạo",
  "Time Path": "Thời Gian Đạo",
  "Transformation Path": "Biến Hóa Đạo",
  "Water Path": "Thủy Đạo",
  "Wind Path": "Phong Đạo",
  "Wisdom Path": "Trí Tuệ Đạo",
  "Wood Path": "Mộc Đạo",
};

const itemTypeLabels: Record<string, string> = {
  Currency: "Tiền tệ",
  Consumable: "Tiêu hao",
  Food: "Thức ăn",
  Material: "Nguyên liệu",
  Equipment: "Trang bị",
};

const guTypeIcons: Record<string, string> = {
  Movement: "🦅",
  Storage: "📦",
  Healing: "❤️",
  Passive: "🔋",
  Support: "🔧",
  Defense: "🛡️",
  Consumable: "💊",
  Active: "⚡",
  Detection: "👁️",
  Illusion: "🌀",
  Summon: "👻",
};

function getGuTypeIcon(type: string): string {
  return guTypeIcons[type] || "🔮";
}

function removeAccents(str: string): string {
  return str
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/đ/g, "d")
    .replace(/Đ/g, "D");
}

export default function WikiClientView({
  initialGus,
  initialItems,
  initialRecipes,
  guideSections,
  roadmap,
  subByParent,
  sectionIcon,
}: Props) {
  // Tabs State
  const [activeTab, setActiveTab] = useState<TabType>("guides");

  // Search State
  const [searchQuery, setSearchQuery] = useState("");
  const [showFilters, setShowFilters] = useState(false);

  // Gu Filters State
  const [guRank, setGuRank] = useState<string>("all");
  const [guPath, setGuPath] = useState<string>("all");
  const [guRarity, setGuRarity] = useState<string>("all");
  const [guType, setGuType] = useState<string>("all");

  // Item Filters State
  const [itemRank, setItemRank] = useState<string>("all");
  const [itemType, setItemType] = useState<string>("all");

  // Recipe Filters State
  const [recipeRank, setRecipeRank] = useState<string>("all");
  const [recipeSort, setRecipeSort] = useState<string>("none");

  // Dynamic filter lists
  const allPaths = useMemo(() => {
    const paths = new Set<string>();
    initialGus.forEach((g) => g.path?.forEach((p) => paths.add(p)));
    return Array.from(paths).sort();
  }, [initialGus]);

  const allGuTypes = useMemo(() => {
    const types = new Set<string>();
    initialGus.forEach((g) => {
      if (g.gu_type) types.add(g.gu_type);
    });
    return Array.from(types).sort();
  }, [initialGus]);

  const allItemTypes = useMemo(() => {
    const types = new Set<string>();
    initialItems.forEach((i) => {
      if (i.item_type) types.add(i.item_type);
    });
    return Array.from(types).sort();
  }, [initialItems]);

  // Clean and filter helper
  const cleanSearch = useMemo(() => removeAccents(searchQuery.toLowerCase().trim()), [searchQuery]);

  // Map Gu IDs to Names for Recipe lookup
  const guMap = useMemo(() => {
    const map = new Map<string, GuDictionaryItem>();
    initialGus.forEach((g) => map.set(g.id, g));
    return map;
  }, [initialGus]);

  // Map Item IDs to Names for Recipe lookup
  const itemMap = useMemo(() => {
    const map = new Map<string, ItemSpec>();
    initialItems.forEach((i) => map.set(i.id, i));
    return map;
  }, [initialItems]);

  // Reset filters when changing tabs
  const handleTabChange = (tab: TabType) => {
    setActiveTab(tab);
    setSearchQuery("");
  };

  // Filtered lists
  const filteredGus = useMemo(() => {
    return initialGus.filter((gu) => {
      // Search check
      if (cleanSearch) {
        const nameClean = removeAccents(gu.name.toLowerCase());
        const idClean = removeAccents(gu.id.toLowerCase());
        const descClean = removeAccents(gu.description.toLowerCase());
        const matchText = nameClean.includes(cleanSearch) || idClean.includes(cleanSearch) || descClean.includes(cleanSearch);
        if (!matchText) return false;
      }
      // Rank check
      if (guRank !== "all" && gu.rank !== Number(guRank)) return false;
      // Path check
      if (guPath !== "all" && !gu.path?.includes(guPath)) return false;
      // Rarity check
      if (guRarity !== "all" && gu.rarity !== guRarity) return false;
      // Type check
      if (guType !== "all" && gu.gu_type !== guType) return false;

      return true;
    }).sort((a, b) => a.name.localeCompare(b.name));
  }, [initialGus, cleanSearch, guRank, guPath, guRarity, guType]);

  const filteredItems = useMemo(() => {
    return initialItems.filter((item) => {
      // Search check
      if (cleanSearch) {
        const nameClean = removeAccents(item.name.toLowerCase());
        const idClean = removeAccents(item.id.toLowerCase());
        const descClean = removeAccents((item.description || "").toLowerCase());
        const matchText = nameClean.includes(cleanSearch) || idClean.includes(cleanSearch) || descClean.includes(cleanSearch);
        if (!matchText) return false;
      }
      // Rank check
      if (itemRank !== "all" && item.rank !== Number(itemRank)) return false;
      // Type check
      if (itemType !== "all" && item.item_type !== itemType) return false;

      return true;
    }).sort((a, b) => a.name.localeCompare(b.name));
  }, [initialItems, cleanSearch, itemRank, itemType]);

  const filteredRecipes = useMemo(() => {
    let list = initialRecipes.filter((recipe) => {
      const targetGu = guMap.get(recipe.target_gu_id);
      const targetName = targetGu ? targetGu.name : recipe.target_gu_id;

      // Search check
      if (cleanSearch) {
        const nameClean = removeAccents(targetName.toLowerCase());
        const idClean = removeAccents(recipe.target_gu_id.toLowerCase());
        const matchText = nameClean.includes(cleanSearch) || idClean.includes(cleanSearch);
        if (!matchText) return false;
      }
      // Target Rank check
      if (recipeRank !== "all" && recipe.target_rank !== Number(recipeRank)) return false;

      return true;
    });

    // Sorting
    if (recipeSort === "high-to-low") {
      list = [...list].sort((a, b) => b.success_rate - a.success_rate);
    } else if (recipeSort === "low-to-high") {
      list = [...list].sort((a, b) => a.success_rate - b.success_rate);
    } else {
      // Default sort by target Gu Name
      list = [...list].sort((a, b) => {
        const nameA = guMap.get(a.target_gu_id)?.name || a.target_gu_id;
        const nameB = guMap.get(b.target_gu_id)?.name || b.target_gu_id;
        return nameA.localeCompare(nameB);
      });
    }

    return list;
  }, [initialRecipes, cleanSearch, recipeRank, recipeSort, guMap]);

  // Navigate to Gu tab and filter by name
  const locateGu = (guId: string) => {
    setActiveTab("gus");
    const target = guMap.get(guId);
    if (target) {
      setSearchQuery(target.name);
    } else {
      setSearchQuery(guId);
    }
  };

  return (
    <div className="min-h-screen bg-[var(--bg-primary)] text-[var(--text-primary)]">
      <div className="max-w-6xl mx-auto px-4 py-8">
        
        {/* Header Section */}
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-8">
          <div className="flex flex-wrap items-center gap-4">
            <Link
              href="/"
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-zinc-900/60 border border-zinc-850 hover:border-emerald-500/30 text-zinc-400 hover:text-emerald-400 transition-all font-mono text-xs cursor-pointer"
            >
              <ArrowLeft className="w-3.5 h-3.5" />
              Quay về Game
            </Link>
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl bg-indigo-950/60 border border-indigo-500/20 flex items-center justify-center">
                <span className="text-lg">📖</span>
              </div>
              <div>
                <h1 className="text-2xl font-serif font-bold tracking-wider bg-gradient-to-r from-zinc-100 to-zinc-400 bg-clip-text text-transparent">
                  Cổ Giới Thư Viện
                </h1>
                <p className="text-xs text-[var(--text-muted)] font-mono mt-0.5">
                  Tra cứu Bách khoa Toàn thư của Cổ Giới
                </p>
              </div>
            </div>
          </div>
          
          {/* Quick Stats Grid */}
          <div className="grid grid-cols-4 gap-2 bg-zinc-900/40 p-2.5 rounded-2xl border border-[var(--border)] max-w-md">
            <div className="text-center px-2 py-1">
              <p className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider">H.Dẫn</p>
              <p className="text-sm font-bold font-mono text-emerald-400">{guideSections.length + (roadmap ? 1 : 0)}</p>
            </div>
            <div className="text-center px-2 py-1 border-l border-zinc-800">
              <p className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider">Cổ</p>
              <p className="text-sm font-bold font-mono text-indigo-400">{initialGus.length}</p>
            </div>
            <div className="text-center px-2 py-1 border-l border-zinc-800">
              <p className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider">Vật phẩm</p>
              <p className="text-sm font-bold font-mono text-sky-400">{initialItems.length}</p>
            </div>
            <div className="text-center px-2 py-1 border-l border-zinc-800">
              <p className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider">Cổ phương</p>
              <p className="text-sm font-bold font-mono text-amber-400">{initialRecipes.length}</p>
            </div>
          </div>
        </div>

        {/* Tab Buttons */}
        <div className="flex flex-wrap gap-1 border-b border-[var(--border)] pb-px mb-6">
          <button
            onClick={() => handleTabChange("guides")}
            className={`flex items-center gap-2 px-4 py-3 text-xs font-bold font-serif uppercase tracking-wider border-b-2 transition-all cursor-pointer ${
              activeTab === "guides"
                ? "border-emerald-500 text-emerald-400 bg-emerald-950/5"
                : "border-transparent text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-zinc-800/10"
            }`}
          >
            <BookOpen className="w-4 h-4" />
            Hướng Dẫn
          </button>
          <button
            onClick={() => handleTabChange("gus")}
            className={`flex items-center gap-2 px-4 py-3 text-xs font-bold font-serif uppercase tracking-wider border-b-2 transition-all cursor-pointer ${
              activeTab === "gus"
                ? "border-indigo-500 text-indigo-400 bg-indigo-950/5"
                : "border-transparent text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-zinc-800/10"
            }`}
          >
            <Sparkles className="w-4 h-4" />
            Cổ Trùng
          </button>
          <button
            onClick={() => handleTabChange("items")}
            className={`flex items-center gap-2 px-4 py-3 text-xs font-bold font-serif uppercase tracking-wider border-b-2 transition-all cursor-pointer ${
              activeTab === "items"
                ? "border-sky-500 text-sky-400 bg-sky-950/5"
                : "border-transparent text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-zinc-800/10"
            }`}
          >
            <Layers className="w-4 h-4" />
            Vật Phẩm
          </button>
          <button
            onClick={() => handleTabChange("recipes")}
            className={`flex items-center gap-2 px-4 py-3 text-xs font-bold font-serif uppercase tracking-wider border-b-2 transition-all cursor-pointer ${
              activeTab === "recipes"
                ? "border-amber-500 text-amber-400 bg-amber-950/5"
                : "border-transparent text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-zinc-800/10"
            }`}
          >
            <Cpu className="w-4 h-4" />
            Cổ Phương
          </button>
        </div>

        {/* Search Bar & Filters Panel */}
        {activeTab !== "guides" && (
          <div className="mb-6 space-y-3">
            <div className="flex gap-2">
              <div className="relative flex-1">
                <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-muted)]" />
                <input
                  type="text"
                  placeholder={
                    activeTab === "gus"
                      ? "Tìm kiếm Cổ trùng theo tên, ID, mô tả..."
                      : activeTab === "items"
                      ? "Tìm kiếm Vật phẩm theo tên, ID, mô tả..."
                      : "Tìm kiếm công thức luyện Cổ theo tên Cổ đích..."
                  }
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="w-full pl-10 pr-4 py-2 rounded-xl bg-[var(--surface)] border border-[var(--border)] focus:border-indigo-500 focus:outline-none text-xs font-mono placeholder-zinc-500 text-[var(--text-primary)]"
                />
                {searchQuery && (
                  <button
                    onClick={() => setSearchQuery("")}
                    className="absolute right-3.5 top-1/2 -translate-y-1/2 text-xs text-[var(--text-muted)] hover:text-[var(--text-primary)] cursor-pointer"
                  >
                    Xóa
                  </button>
                )}
              </div>
              <button
                onClick={() => setShowFilters(!showFilters)}
                className={`flex items-center gap-1.5 px-4 py-2 rounded-xl border text-xs font-bold cursor-pointer transition-all ${
                  showFilters
                    ? "bg-zinc-800 border-zinc-700 text-zinc-200"
                    : "bg-[var(--surface)] border-[var(--border)] text-[var(--text-muted)] hover:text-[var(--text-primary)]"
                }`}
              >
                <SlidersHorizontal className="w-3.5 h-3.5" />
                <span>Bộ lọc</span>
              </button>
            </div>

            {/* Filters Collapse Drawer */}
            {showFilters && (
              <div className="p-4 rounded-xl bg-zinc-950/30 border border-[var(--border)] grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-3 animate-fadeIn">
                {activeTab === "gus" && (
                  <>
                    {/* Gu Path */}
                    <div className="space-y-1">
                      <label className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider">Hệ phái</label>
                      <select
                        value={guPath}
                        onChange={(e) => setGuPath(e.target.value)}
                        className="w-full bg-[var(--surface)] border border-[var(--border)] rounded-lg px-2.5 py-1.5 text-xs text-[var(--text-primary)] focus:outline-none"
                      >
                        <option value="all">Tất cả hệ phái</option>
                        {allPaths.map((p) => (
                          <option key={p} value={p}>
                            {pathLabels[p] || p}
                          </option>
                        ))}
                      </select>
                    </div>

                    {/* Gu Rank */}
                    <div className="space-y-1">
                      <label className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider">Tu vi (Chuyển)</label>
                      <select
                        value={guRank}
                        onChange={(e) => setGuRank(e.target.value)}
                        className="w-full bg-[var(--surface)] border border-[var(--border)] rounded-lg px-2.5 py-1.5 text-xs text-[var(--text-primary)] focus:outline-none"
                      >
                        <option value="all">Tất cả Chuyển</option>
                        {[1, 2, 3, 4, 5, 6, 7, 8, 9].map((r) => (
                          <option key={r} value={r}>
                            Chuyển {r}
                          </option>
                        ))}
                      </select>
                    </div>

                    {/* Gu Rarity */}
                    <div className="space-y-1">
                      <label className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider">Phẩm chất</label>
                      <select
                        value={guRarity}
                        onChange={(e) => setGuRarity(e.target.value)}
                        className="w-full bg-[var(--surface)] border border-[var(--border)] rounded-lg px-2.5 py-1.5 text-xs text-[var(--text-primary)] focus:outline-none"
                      >
                        <option value="all">Tất cả phẩm chất</option>
                        {Object.entries(rarityLabels).map(([k, v]) => (
                          <option key={k} value={k}>
                            {v}
                          </option>
                        ))}
                      </select>
                    </div>

                    {/* Gu Type */}
                    <div className="space-y-1">
                      <label className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider">Phân loại</label>
                      <select
                        value={guType}
                        onChange={(e) => setGuType(e.target.value)}
                        className="w-full bg-[var(--surface)] border border-[var(--border)] rounded-lg px-2.5 py-1.5 text-xs text-[var(--text-primary)] focus:outline-none"
                      >
                        <option value="all">Tất cả loại</option>
                        {allGuTypes.map((t) => (
                          <option key={t} value={t}>
                            {t}
                          </option>
                        ))}
                      </select>
                    </div>
                  </>
                )}

                {activeTab === "items" && (
                  <>
                    {/* Item Rank */}
                    <div className="space-y-1">
                      <label className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider">Cấp bậc</label>
                      <select
                        value={itemRank}
                        onChange={(e) => setItemRank(e.target.value)}
                        className="w-full bg-[var(--surface)] border border-[var(--border)] rounded-lg px-2.5 py-1.5 text-xs text-[var(--text-primary)] focus:outline-none"
                      >
                        <option value="all">Tất cả cấp</option>
                        {[1, 2, 3, 4, 5, 6, 7, 8, 9].map((r) => (
                          <option key={r} value={r}>
                            Cấp {r}
                          </option>
                        ))}
                      </select>
                    </div>

                    {/* Item Type */}
                    <div className="space-y-1">
                      <label className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider">Phân loại</label>
                      <select
                        value={itemType}
                        onChange={(e) => setItemType(e.target.value)}
                        className="w-full bg-[var(--surface)] border border-[var(--border)] rounded-lg px-2.5 py-1.5 text-xs text-[var(--text-primary)] focus:outline-none"
                      >
                        <option value="all">Tất cả loại</option>
                        {allItemTypes.map((t) => (
                          <option key={t} value={t}>
                            {itemTypeLabels[t] || t}
                          </option>
                        ))}
                      </select>
                    </div>
                  </>
                )}

                {activeTab === "recipes" && (
                  <>
                    {/* Recipe Target Rank */}
                    <div className="space-y-1">
                      <label className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider">Tu vi Cổ đích</label>
                      <select
                        value={recipeRank}
                        onChange={(e) => setRecipeRank(e.target.value)}
                        className="w-full bg-[var(--surface)] border border-[var(--border)] rounded-lg px-2.5 py-1.5 text-xs text-[var(--text-primary)] focus:outline-none"
                      >
                        <option value="all">Tất cả Chuyển</option>
                        {[1, 2, 3, 4, 5, 6, 7, 8, 9].map((r) => (
                          <option key={r} value={r}>
                            Luyện Cổ Chuyển {r}
                          </option>
                        ))}
                      </select>
                    </div>

                    {/* Recipe Sorting */}
                    <div className="space-y-1">
                      <label className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider">Sắp xếp</label>
                      <select
                        value={recipeSort}
                        onChange={(e) => setRecipeSort(e.target.value)}
                        className="w-full bg-[var(--surface)] border border-[var(--border)] rounded-lg px-2.5 py-1.5 text-xs text-[var(--text-primary)] focus:outline-none"
                      >
                        <option value="none">Mặc định (Tên Cổ)</option>
                        <option value="high-to-low">Tỉ lệ thành công: Cao → Thấp</option>
                        <option value="low-to-high">Tỉ lệ thành công: Thấp → Cao</option>
                      </select>
                    </div>
                  </>
                )}
              </div>
            )}
          </div>
        )}

        {/* Tab Content Rendering */}

        {/* 1. GUIDES TAB */}
        {activeTab === "guides" && (
          <section className="p-5 rounded-2xl bg-zinc-900/20 border border-[var(--border)]">
            <div className="flex items-center gap-2 mb-4">
              <span className="text-sm">📚</span>
              <h2 className="text-sm font-bold uppercase tracking-wider text-[var(--text-secondary)] font-mono">Hướng Dẫn Cổ Giới</h2>
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
              {roadmap && (
                <Link
                  href={`/wiki/${roadmap.id}`}
                  className="flex items-center gap-3 p-4 rounded-xl bg-[var(--surface)] border border-[var(--border)] hover:border-emerald-500/30 hover:bg-[var(--bg-elevated)] transition-all group shadow-sm cursor-pointer"
                >
                  <span className="text-xl shrink-0 p-2 rounded-lg bg-zinc-950/40 border border-zinc-800/30">🗺️</span>
                  <div className="min-w-0">
                    <span className="block text-xs font-serif font-bold text-[var(--text-primary)] group-hover:text-emerald-400 transition-colors">
                      {roadmap.title}
                    </span>
                    <span className="block text-[10px] text-[var(--text-muted)] font-mono mt-0.5">Lộ trình phát triển</span>
                  </div>
                </Link>
              )}
              {guideSections.map((page) => {
                const subs = subByParent[page.id];
                return (
                  <div key={page.id} className="p-4 rounded-xl bg-[var(--surface)] border border-[var(--border)] space-y-2.5 shadow-sm">
                    <Link
                      href={`/wiki/${page.id}`}
                      className="flex items-center gap-3 group cursor-pointer"
                    >
                      <span className="text-xl shrink-0 p-2 rounded-lg bg-zinc-950/40 border border-zinc-800/30">
                        {sectionIcon[page.category] || "📄"}
                      </span>
                      <div className="min-w-0">
                        <span className="block text-xs font-serif font-bold text-[var(--text-primary)] group-hover:text-emerald-400 transition-colors">
                          {page.title}
                        </span>
                        <span className="block text-[10px] text-[var(--text-muted)] font-mono mt-0.5">Chủ đề chính</span>
                      </div>
                    </Link>
                    {subs && subs.length > 0 && (
                      <div className="pl-12 space-y-1.5 border-l border-zinc-800/80 ml-5">
                        {subs.map((sub) => (
                          <Link
                            key={sub.id}
                            href={`/wiki/${sub.id}`}
                            className="block text-[11px] text-[var(--text-muted)] hover:text-emerald-400 font-mono transition-colors cursor-pointer"
                          >
                            └ {sub.title}
                          </Link>
                        ))}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </section>
        )}

        {/* 2. GUS TAB */}
        {activeTab === "gus" && (
          <div>
            {filteredGus.length === 0 ? (
              <div className="text-center py-20 bg-zinc-900/10 rounded-2xl border border-[var(--border)] border-dashed">
                <ShieldAlert className="w-8 h-8 mx-auto text-[var(--text-muted)] mb-2" />
                <p className="text-[var(--text-muted)] font-mono text-xs">Không tìm thấy Cổ trùng nào trùng khớp với bộ lọc</p>
              </div>
            ) : (
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                {filteredGus.map((gu) => (
                  <Link
                    key={gu.id}
                    href={`/wiki/gu/${gu.id}`}
                    className="group flex flex-col p-4 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)] hover:border-indigo-500/30 transition-all hover:bg-[var(--bg-elevated)] shadow-sm hover:shadow-md cursor-pointer"
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div className="flex items-center gap-3 min-w-0">
                        <span className="text-xl shrink-0 p-1.5 rounded-lg bg-zinc-950/50 border border-zinc-800/40">
                          {getGuTypeIcon(gu.gu_type)}
                        </span>
                        <div className="min-w-0">
                          <h3 className="text-xs font-serif font-bold text-[var(--text-primary)] group-hover:text-indigo-400 transition-colors truncate">
                            {gu.name}
                          </h3>
                          <p className="text-[9px] text-[var(--text-muted)] font-mono mt-0.5">
                            Rank {gu.rank} · {gu.gu_type}
                          </p>
                        </div>
                      </div>
                      <span className={`px-2 py-0.5 rounded-md text-[9px] font-mono font-bold shrink-0 border border-zinc-800/60 ${rarityColors[gu.rarity] || "text-[var(--text-muted)]"}`}>
                        {rarityLabels[gu.rarity] || gu.rarity}
                      </span>
                    </div>

                    <p className="text-[11px] text-[var(--text-muted)] leading-relaxed mt-3 line-clamp-2 italic font-serif">
                      &ldquo;{gu.description}&rdquo;
                    </p>

                    <div className="flex flex-wrap gap-1 mt-auto pt-3 border-t border-zinc-900/60">
                      {gu.path && gu.path.map((pathName) => {
                        const pathColor = getPathColor(pathName);
                        return (
                          <span
                            key={pathName}
                            className={`px-1.5 py-0.5 rounded text-[8px] font-mono font-semibold border ${pathColor.bg} ${pathColor.text} ${pathColor.border}`}
                          >
                            {pathLabels[pathName] || pathName}
                          </span>
                        );
                      })}
                      {gu.combat_tags.slice(0, 2).map((tag) => (
                        <span
                          key={tag}
                          className="px-1.5 py-0.5 rounded bg-zinc-900/50 text-[8px] text-zinc-500 border border-zinc-800/40 font-mono"
                        >
                          #{tag}
                        </span>
                      ))}
                    </div>
                  </Link>
                ))}
              </div>
            )}
          </div>
        )}

        {/* 3. ITEMS TAB */}
        {activeTab === "items" && (
          <div>
            {filteredItems.length === 0 ? (
              <div className="text-center py-20 bg-zinc-900/10 rounded-2xl border border-[var(--border)] border-dashed">
                <ShieldAlert className="w-8 h-8 mx-auto text-[var(--text-muted)] mb-2" />
                <p className="text-[var(--text-muted)] font-mono text-xs">Không tìm thấy vật phẩm nào trùng khớp với bộ lọc</p>
              </div>
            ) : (
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                {filteredItems.map((item) => (
                  <div
                    key={item.id}
                    className="group flex flex-col p-4 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)] hover:border-sky-500/20 transition-all bg-gradient-to-br from-[var(--bg-card)] to-zinc-900/10 shadow-sm"
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div className="flex items-center gap-3 min-w-0">
                        {item.image_url ? (
                          <img
                            src={item.image_url}
                            alt={item.name}
                            className="w-10 h-10 rounded-lg object-cover bg-zinc-950 border border-zinc-800 shrink-0"
                          />
                        ) : (
                          <span className="text-xl shrink-0 p-1.5 rounded-lg bg-zinc-950/50 border border-zinc-800/40">
                            📦
                          </span>
                        )}
                        <div className="min-w-0">
                          <h3 className="text-xs font-serif font-bold text-[var(--text-primary)] truncate">
                            {item.name}
                          </h3>
                          <p className="text-[9px] text-[var(--text-muted)] font-mono mt-0.5">
                            ID: {item.id}
                          </p>
                        </div>
                      </div>
                      <div className="flex flex-col items-end gap-1 shrink-0">
                        <span className="px-1.5 py-0.5 rounded-md bg-sky-950/20 text-sky-400 border border-sky-500/10 text-[9px] font-mono font-bold">
                          {itemTypeLabels[item.item_type] || item.item_type}
                        </span>
                        {item.rank && item.rank > 0 && (
                          <span className="text-[9px] text-[var(--text-muted)] font-mono">
                            Cấp {item.rank}
                          </span>
                        )}
                      </div>
                    </div>

                    <p className="text-[11px] text-[var(--text-muted)] leading-relaxed mt-3 line-clamp-2">
                      {item.description || "Không có mô tả chi tiết."}
                    </p>

                    {/* Stats & Price */}
                    <div className="flex items-center justify-between mt-auto pt-3 border-t border-zinc-900/60 text-[10px] font-mono text-[var(--text-muted)]">
                      {item.effects && Object.keys(item.effects).length > 0 ? (
                        <div className="flex flex-wrap gap-1">
                          {Object.entries(item.effects).slice(0, 2).map(([k, v]) => (
                            <span key={k} className="px-1 py-0.5 rounded bg-zinc-900/60 text-zinc-400 text-[8px]">
                              {k}: +{v}
                            </span>
                          ))}
                        </div>
                      ) : (
                        <span className="text-zinc-600 italic text-[9px]">Vật phẩm phổ thông</span>
                      )}
                      
                      {item.base_price ? (
                        <div className="flex items-center gap-1 text-amber-500 font-bold text-[10px]">
                          <Coins className="w-3.5 h-3.5" />
                          <span>{item.base_price} Đá</span>
                        </div>
                      ) : null}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* 4. RECIPES TAB */}
        {activeTab === "recipes" && (
          <div>
            {filteredRecipes.length === 0 ? (
              <div className="text-center py-20 bg-zinc-900/10 rounded-2xl border border-[var(--border)] border-dashed">
                <ShieldAlert className="w-8 h-8 mx-auto text-[var(--text-muted)] mb-2" />
                <p className="text-[var(--text-muted)] font-mono text-xs">Không tìm thấy công thức nào trùng khớp với bộ lọc</p>
              </div>
            ) : (
              <div className="space-y-3">
                {filteredRecipes.map((recipe) => {
                  const targetGu = guMap.get(recipe.target_gu_id);
                  const targetName = targetGu ? targetGu.name : recipe.target_gu_id;
                  
                  return (
                    <div
                      key={recipe.id}
                      className="p-5 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)] hover:border-amber-500/20 transition-all flex flex-col md:flex-row gap-5 items-start md:items-stretch shadow-sm"
                    >
                      {/* Left Block: Target Gu Info */}
                      <div className="md:w-1/4 shrink-0 flex flex-col justify-between border-b md:border-b-0 md:border-r border-zinc-900/80 pb-4 md:pb-0 md:pr-5">
                        <div>
                          <div className="flex items-center gap-2">
                            <span className="text-xl p-1 bg-zinc-950/60 border border-zinc-800/40 rounded-lg">
                              {targetGu ? getGuTypeIcon(targetGu.gu_type) : "🔮"}
                            </span>
                            <div>
                              <h3 className="text-xs font-serif font-bold text-amber-400 transition-colors truncate max-w-[150px]">
                                {targetName}
                              </h3>
                              <p className="text-[9px] text-[var(--text-muted)] font-mono mt-0.5">
                                Cổ trùng mục tiêu
                              </p>
                            </div>
                          </div>
                          
                          {targetGu && (
                            <div className="flex flex-wrap gap-1 mt-3">
                              <span className="px-1.5 py-0.5 rounded bg-zinc-950/40 border border-zinc-800 text-[8px] text-[var(--text-secondary)] font-mono">
                                Chuyển {recipe.target_rank || targetGu.rank}
                              </span>
                              <span className={`px-1.5 py-0.5 rounded bg-zinc-950/40 border border-zinc-800 text-[8px] font-mono font-bold ${rarityColors[targetGu.rarity] || "text-[var(--text-muted)]"}`}>
                                {rarityLabels[targetGu.rarity] || targetGu.rarity}
                              </span>
                            </div>
                          )}
                        </div>

                        {targetGu && (
                          <button
                            onClick={() => locateGu(recipe.target_gu_id)}
                            className="flex items-center gap-1 text-[10px] text-indigo-400 hover:text-indigo-300 font-mono mt-4 text-left cursor-pointer"
                          >
                            <span>Xem chỉ số Cổ</span>
                            <ChevronRight className="w-3 h-3" />
                          </button>
                        )}
                      </div>

                      {/* Middle Block: Ingredients (Gus & Materials) */}
                      <div className="flex-1 grid grid-cols-1 sm:grid-cols-2 gap-4">
                        {/* Required Gus */}
                        <div>
                          <h4 className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider mb-2 flex items-center gap-1.5">
                            <span className="w-1.5 h-1.5 rounded-full bg-indigo-500"></span>
                            Cổ trùng yêu cầu
                          </h4>
                          {Object.keys(recipe.required_gus).length === 0 ? (
                            <p className="text-[10px] text-zinc-600 italic font-mono">Không yêu cầu Cổ khác</p>
                          ) : (
                            <div className="space-y-1">
                              {Object.entries(recipe.required_gus).map(([guId, qty]) => {
                                const reqGu = guMap.get(guId);
                                const reqName = reqGu ? reqGu.name : guId;
                                return (
                                  <div
                                    key={guId}
                                    onClick={() => locateGu(guId)}
                                    className="flex items-center justify-between text-[11px] font-mono p-1.5 px-2.5 rounded-lg bg-zinc-950/40 border border-zinc-900/60 hover:border-indigo-500/20 cursor-pointer transition-colors"
                                    title="Click để tìm Cổ trùng này"
                                  >
                                    <span className="text-[var(--text-primary)] hover:text-indigo-400 transition-colors truncate">
                                      {reqName}
                                    </span>
                                    <span className="text-zinc-500 font-bold shrink-0 ml-2">x{qty}</span>
                                  </div>
                                );
                              })}
                            </div>
                          )}
                        </div>

                        {/* Required Materials */}
                        <div>
                          <h4 className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider mb-2 flex items-center gap-1.5">
                            <span className="w-1.5 h-1.5 rounded-full bg-sky-500"></span>
                            Nguyên liệu cần có
                          </h4>
                          {Object.keys(recipe.required_materials).length === 0 ? (
                            <p className="text-[10px] text-zinc-600 italic font-mono">Không yêu cầu nguyên liệu</p>
                          ) : (
                            <div className="space-y-1">
                              {Object.entries(recipe.required_materials).map(([matId, qty]) => {
                                const reqMat = itemMap.get(matId);
                                const reqName = reqMat ? reqMat.name : matId;
                                return (
                                  <div
                                    key={matId}
                                    className="flex items-center justify-between text-[11px] font-mono p-1.5 px-2.5 rounded-lg bg-zinc-950/40 border border-zinc-900/60"
                                  >
                                    <span className="text-[var(--text-primary)] truncate">
                                      {reqName}
                                    </span>
                                    <span className="text-zinc-500 font-bold shrink-0 ml-2">x{qty}</span>
                                  </div>
                                );
                              })}
                            </div>
                          )}
                        </div>
                      </div>

                      {/* Right Block: Stats & Price Cost */}
                      <div className="md:w-1/4 shrink-0 flex flex-col justify-between items-end border-t md:border-t-0 md:border-l border-zinc-900/80 pt-4 md:pt-0 md:pl-5">
                        <div className="text-right">
                          <p className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider">Tỉ lệ thành công</p>
                          <p className="text-lg font-bold font-mono text-emerald-400 mt-1">{(recipe.success_rate * 100).toFixed(0)}%</p>
                        </div>

                        <div className="space-y-1 mt-4 text-right">
                          {recipe.primeval_stones_cost > 0 && (
                            <div className="flex items-center justify-end gap-1.5 text-[10px] font-mono text-[var(--text-secondary)]">
                              <span>{recipe.primeval_stones_cost.toLocaleString()} Nguyên thạch</span>
                            </div>
                          )}
                          {recipe.immortal_essence_stones_cost > 0 && (
                            <div className="flex items-center justify-end gap-1.5 text-[10px] font-mono text-amber-500 font-bold">
                              <span>{recipe.immortal_essence_stones_cost.toLocaleString()} Tiên nguyên thạch</span>
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
