"use client";



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
  ArrowLeft,
  MapPin,
  Skull,
  Map as MapIcon
} from "lucide-react";
import { type GuDictionaryItem, type MonsterTemplate, type LootTable, type WikiRegion, type WikiMonsterSpawn, reqGuQty, reqGuRank } from "@/lib/gu-dict";
import { type ItemSpec, type RefiningRecipe } from "@/store/useGameStore";
import { getPathColor } from "@/lib/utils";
import { translatePath, translateMastery, masteryKeyForPoints, masteryLevelForPoints, PATH_MASTERY_MILESTONES } from "@/lib/translate";
import { EffectEnvelopeView } from "@/components/EffectBlockView";


// Types
type TabType = "guides" | "gus" | "items" | "recipes" | "monsters" | "regions";

interface Props {
  initialGus: GuDictionaryItem[];
  initialItems: ItemSpec[];
  initialRecipes: RefiningRecipe[];
  initialMonsters: MonsterTemplate[];
  initialLootTables: LootTable[];
  initialRegions: WikiRegion[];
  initialMonsterSpawns: WikiMonsterSpawn[];
  guideSections: ApiResponseData[];
  roadmap: ApiResponseData;
  subByParent: Record<string, ApiResponseData[]>;
  sectionIcon: Record<string, string>;
}

// Translations and Mappings
const rarityColors: Record<number, string> = {
  1: "text-zinc-400 border-zinc-700/50 bg-zinc-950/20",
  2: "text-emerald-400 border-emerald-500/20 bg-emerald-950/10",
  3: "text-sky-400 border-sky-500/20 bg-sky-950/10",
  4: "text-purple-400 border-purple-500/20 bg-purple-950/10",
  5: "text-amber-400 border-amber-500/20 bg-amber-950/10",
  6: "text-rose-400 border-rose-500/20 bg-rose-950/10",
  7: "text-red-400 border-red-500/20 bg-red-950/10",
  8: "text-orange-400 border-orange-500/20 bg-orange-950/10",
  9: "text-fuchsia-400 border-fuchsia-500/20 bg-fuchsia-950/10",
};

const rarityLabels: Record<number, string> = {
  1: "Phàm",
  2: "Tinh Phẩm",
  3: "Hiếm",
  4: "Quý",
  5: "Sử Thi",
  6: "Huyền Thoại",
  7: "Thần Thoại",
  8: "Chí Bảo",
  9: "Tuyệt Thế",
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
  initialMonsters,
  initialLootTables,
  initialRegions,
  initialMonsterSpawns,
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

  // Monster Filters State
  const [monsterType, setMonsterType] = useState<string>("all");
  const [monsterRank, setMonsterRank] = useState<string>("all");

  // Region Filters State
  const [regionZoneType, setRegionZoneType] = useState<string>("all");
  const [regionRank, setRegionRank] = useState<string>("all");

  // Expand state for long lists in region and monster cards
  const [expandedRegions, setExpandedRegions] = useState<Record<string, boolean>>({});
  const [expandedMonsters, setExpandedMonsters] = useState<Record<string, boolean>>({});

  const toggleRegionExpand = (regionId: string) => {
    setExpandedRegions(prev => ({
      ...prev,
      [regionId]: !prev[regionId]
    }));
  };

  const toggleMonsterExpand = (monsterId: string) => {
    setExpandedMonsters(prev => ({
      ...prev,
      [monsterId]: !prev[monsterId]
    }));
  };

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

  const allMonsterTypes = useMemo(() => {
    const types = new Set<string>();
    initialMonsters.forEach((m) => { if (m.type) types.add(m.type); });
    return Array.from(types).sort();
  }, [initialMonsters]);

  const allZoneTypes = useMemo(() => {
    const types = new Set<string>();
    initialRegions.forEach((r) => { if (r.zone_type) types.add(r.zone_type); });
    return Array.from(types).sort();
  }, [initialRegions]);

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

  // Monster map
  const monsterMap = useMemo(() => {
    const map = new Map<string, MonsterTemplate>();
    initialMonsters.forEach((m) => map.set(m.id, m));
    return map;
  }, [initialMonsters]);

  // Region map
  const regionMap = useMemo(() => {
    const map = new Map<string, WikiRegion>();
    initialRegions.forEach((r) => map.set(r.id, r));
    return map;
  }, [initialRegions]);

  // Loot table map
  const lootTableMap = useMemo(() => {
    const map = new Map<string, LootTable>();
    initialLootTables.forEach((lt) => map.set(lt.id, lt));
    return map;
  }, [initialLootTables]);

  // Source lookup: item -> { regions, monsters }
  const itemSourceMap = useMemo(() => {
    const map = new Map<string, { regions: string[]; monsters: string[] }>();
    // From region pool_weights.resources
    initialRegions.forEach((r) => {
      (r.pool_weights?.resources || []).forEach((itemId) => {
        if (!map.has(itemId)) map.set(itemId, { regions: [], monsters: [] });
        map.get(itemId)!.regions.push(r.id);
      });
    });
    // From loot table entries
    initialMonsters.forEach((m) => {
      if (!m.loot_table_id) return;
      const lt = lootTableMap.get(m.loot_table_id);
      if (!lt) return;
      lt.entries.forEach((e) => {
        if (e.item_id) {
          if (!map.has(e.item_id)) map.set(e.item_id, { regions: [], monsters: [] });
          const src = map.get(e.item_id)!;
          if (!src.monsters.includes(m.id)) src.monsters.push(m.id);
        }
      });
    });
    return map;
  }, [initialRegions, initialMonsters, lootTableMap]);

  // Source lookup: gu -> { wildRegions, dropMonsters }
  const guSourceMap = useMemo(() => {
    const map = new Map<string, { wildRegions: string[]; dropMonsters: string[] }>();
    // From region pool_weights.wild_gus
    initialRegions.forEach((r) => {
      (r.pool_weights?.wild_gus || []).forEach((guId) => {
        if (!map.has(guId)) map.set(guId, { wildRegions: [], dropMonsters: [] });
        map.get(guId)!.wildRegions.push(r.id);
      });
    });
    // From loot table entries
    initialMonsters.forEach((m) => {
      if (!m.loot_table_id) return;
      const lt = lootTableMap.get(m.loot_table_id);
      if (!lt) return;
      lt.entries.forEach((e) => {
        if (e.gu_id) {
          if (!map.has(e.gu_id)) map.set(e.gu_id, { wildRegions: [], dropMonsters: [] });
          const src = map.get(e.gu_id)!;
          if (!src.dropMonsters.includes(m.id)) src.dropMonsters.push(m.id);
        }
      });
    });
    return map;
  }, [initialRegions, initialMonsters, lootTableMap]);

  // Monster -> regions map
  const monsterRegionMap = useMemo(() => {
    const map = new Map<string, string[]>();
    initialMonsterSpawns.forEach((s) => {
      if (!map.has(s.monster_id)) map.set(s.monster_id, []);
      const arr = map.get(s.monster_id)!;
      if (!arr.includes(s.region_id)) arr.push(s.region_id);
    });
    return map;
  }, [initialMonsterSpawns]);

  // Region -> monsters map
  const regionMonsterMap = useMemo(() => {
    const map = new Map<string, string[]>();
    initialMonsterSpawns.forEach((s) => {
      if (!map.has(s.region_id)) map.set(s.region_id, []);
      const arr = map.get(s.region_id)!;
      if (!arr.includes(s.monster_id)) arr.push(s.monster_id);
    });
    return map;
  }, [initialMonsterSpawns]);

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
        const nameClean = removeAccents((gu.name || "").toLowerCase());
        const idClean = removeAccents((gu.id || "").toLowerCase());
        const descClean = removeAccents((gu.description || "").toLowerCase());
        const matchText = nameClean.includes(cleanSearch) || idClean.includes(cleanSearch) || descClean.includes(cleanSearch);
        if (!matchText) return false;
      }
      // Rank check
      if (guRank !== "all" && gu.rank !== Number(guRank)) return false;
      // Path check
      if (guPath !== "all" && !gu.path?.includes(guPath)) return false;
      // Rarity check
      if (guRarity !== "all" && gu.rarity !== Number(guRarity)) return false;
      // Type check
      if (guType !== "all" && gu.gu_type !== guType) return false;

      return true;
    }).sort((a, b) => (a.name || "").localeCompare(b.name || ""));
  }, [initialGus, cleanSearch, guRank, guPath, guRarity, guType]);

  const filteredItems = useMemo(() => {
    return initialItems.filter((item) => {
      // Search check
      if (cleanSearch) {
        const nameClean = removeAccents((item.name || "").toLowerCase());
        const idClean = removeAccents((item.id || "").toLowerCase());
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
        const nameClean = removeAccents((targetName || "").toLowerCase());
        const idClean = removeAccents((recipe.target_gu_id || "").toLowerCase());
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

  const filteredMonsters = useMemo(() => {
    return initialMonsters.filter((monster) => {
      if (cleanSearch) {
        const nameClean = removeAccents((monster.name || "").toLowerCase());
        const idClean = removeAccents((monster.id || "").toLowerCase());
        const typeClean = removeAccents((monster.type || "").toLowerCase());
        if (!nameClean.includes(cleanSearch) && !idClean.includes(cleanSearch) && !typeClean.includes(cleanSearch)) return false;
      }
      if (monsterType !== "all" && monster.type !== monsterType) return false;
      return true; // We don't have rank in MonsterTemplate for this simple filter currently, unless we add it
    }).sort((a, b) => a.name.localeCompare(b.name));
  }, [initialMonsters, cleanSearch, monsterType]);

  const filteredRegions = useMemo(() => {
    return initialRegions.filter((region) => {
      if (cleanSearch) {
        const nameClean = removeAccents((region.name || "").toLowerCase());
        const descClean = removeAccents((region.description || "").toLowerCase());
        if (!nameClean.includes(cleanSearch) && !descClean.includes(cleanSearch)) return false;
      }
      if (regionZoneType !== "all" && region.zone_type !== regionZoneType) return false;
      return true;
    }).sort((a, b) => a.name.localeCompare(b.name));
  }, [initialRegions, cleanSearch, regionZoneType]);

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

  const locateMonster = (monsterId: string) => {
    setActiveTab("monsters");
    const target = monsterMap.get(monsterId);
    if (target) {
      setSearchQuery(target.name);
    } else {
      setSearchQuery(monsterId);
    }
  };

  const locateRegion = (regionId: string) => {
    setActiveTab("regions");
    const target = regionMap.get(regionId);
    if (target) {
      setSearchQuery(target.name);
    } else {
      setSearchQuery(regionId);
    }
  };

  return (
    <div className="min-h-screen bg-[var(--bg-primary)] text-[var(--text-primary)]">
      <div className="w-full mx-auto px-2 sm:px-4 py-8">

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
              <p className="text-sm font-bold font-mono text-emerald-400">
                {guideSections.length + Object.values(subByParent).flat().length + (roadmap ? 1 : 0)}
              </p>
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
          <button
            onClick={() => handleTabChange("monsters")}
            className={`flex items-center gap-2 px-4 py-3 text-xs font-bold font-serif uppercase tracking-wider border-b-2 transition-all cursor-pointer ${
              activeTab === "monsters"
                ? "border-red-500 text-red-400 bg-red-950/5"
                : "border-transparent text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-zinc-800/10"
            }`}
          >
            <Skull className="w-4 h-4" />
            Quái Vật
          </button>
          <button
            onClick={() => handleTabChange("regions")}
            className={`flex items-center gap-2 px-4 py-3 text-xs font-bold font-serif uppercase tracking-wider border-b-2 transition-all cursor-pointer ${
              activeTab === "regions"
                ? "border-orange-500 text-orange-400 bg-orange-950/5"
                : "border-transparent text-[var(--text-muted)] hover:text-[var(--text-primary)] hover:bg-zinc-800/10"
            }`}
          >
            <MapIcon className="w-4 h-4" />
            Khu Vực
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
                      : activeTab === "recipes"
                      ? "Tìm kiếm công thức luyện Cổ theo tên Cổ đích..."
                      : activeTab === "monsters"
                      ? "Tìm kiếm Quái vật theo tên, ID, loại..."
                      : "Tìm kiếm Khu vực theo tên, mô tả..."
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
                            {translatePath(p)}
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

                {activeTab === "monsters" && (
                  <>
                    <div className="space-y-1">
                      <label className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider">Phân loại (Type)</label>
                      <select
                        value={monsterType}
                        onChange={(e) => setMonsterType(e.target.value)}
                        className="w-full bg-[var(--surface)] border border-[var(--border)] rounded-lg px-2.5 py-1.5 text-xs text-[var(--text-primary)] focus:outline-none"
                      >
                        <option value="all">Tất cả loại quái</option>
                        {allMonsterTypes.map((t) => (
                          <option key={t} value={t}>
                            {t}
                          </option>
                        ))}
                      </select>
                    </div>
                  </>
                )}

                {activeTab === "regions" && (
                  <>
                    <div className="space-y-1">
                      <label className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider">Loại khu vực</label>
                      <select
                        value={regionZoneType}
                        onChange={(e) => setRegionZoneType(e.target.value)}
                        className="w-full bg-[var(--surface)] border border-[var(--border)] rounded-lg px-2.5 py-1.5 text-xs text-[var(--text-primary)] focus:outline-none"
                      >
                        <option value="all">Tất cả loại khu vực</option>
                        {allZoneTypes.map((t) => (
                          <option key={t} value={t}>
                            {t}
                          </option>
                        ))}
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
                    key={gu.id + gu.rank}
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
                      {gu.rarity > 0 && (
                        <span className={`px-2 py-0.5 rounded-md text-[9px] font-mono font-bold shrink-0 border border-zinc-800/60 ${rarityColors[gu.rarity] || "text-[var(--text-muted)]"}`}>
                          {rarityLabels[gu.rarity] || `Độ hiếm ${gu.rarity}`}
                        </span>
                      )}
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
                            pathName
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

                    {/* Source Locations */}
                    {guSourceMap.has(gu.id) && (guSourceMap.get(gu.id)!.wildRegions.length > 0 || guSourceMap.get(gu.id)!.dropMonsters.length > 0) && (
                      <div className="flex flex-wrap gap-1 mt-2 pt-2 border-t border-zinc-900/40">
                        {guSourceMap.get(gu.id)!.wildRegions.slice(0, 2).map((rId) => {
                          const region = regionMap.get(rId);
                          return (
                            <span key={`w-${rId}`} className="px-1.5 py-0.5 rounded bg-orange-950/30 text-orange-400 border border-orange-500/20 text-[8px] font-mono flex items-center gap-1">
                              <MapPin className="w-2.5 h-2.5" />
                              {region ? region.name : rId}
                            </span>
                          );
                        })}
                        {guSourceMap.get(gu.id)!.dropMonsters.slice(0, 2).map((mId) => {
                          const monster = monsterMap.get(mId);
                          return (
                            <span key={`m-${mId}`} className="px-1.5 py-0.5 rounded bg-red-950/30 text-red-400 border border-red-500/20 text-[8px] font-mono flex items-center gap-1">
                              <Skull className="w-2.5 h-2.5" />
                              {monster ? monster.name : mId}
                            </span>
                          );
                        })}
                      </div>
                    )}
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

                    {/* Source Locations */}
                    {itemSourceMap.has(item.id) && (itemSourceMap.get(item.id)!.regions.length > 0 || itemSourceMap.get(item.id)!.monsters.length > 0) && (
                      <div className="flex flex-wrap gap-1 mt-2">
                        {itemSourceMap.get(item.id)!.regions.slice(0, 2).map((rId) => {
                          const region = regionMap.get(rId);
                          return (
                            <span key={`r-${rId}`} className="px-1.5 py-0.5 rounded bg-orange-950/30 text-orange-400 border border-orange-500/20 text-[8px] font-mono flex items-center gap-1">
                              <MapPin className="w-2.5 h-2.5" />
                              {region ? region.name : rId}
                            </span>
                          );
                        })}
                        {itemSourceMap.get(item.id)!.monsters.slice(0, 2).map((mId) => {
                          const monster = monsterMap.get(mId);
                          return (
                            <span key={`m-${mId}`} className="px-1.5 py-0.5 rounded bg-red-950/30 text-red-400 border border-red-500/20 text-[8px] font-mono flex items-center gap-1">
                              <Skull className="w-2.5 h-2.5" />
                              {monster ? monster.name : mId}
                            </span>
                          );
                        })}
                      </div>
                    )}

                    {/* Stats & Price */}
                    <div className="flex items-center justify-between mt-auto pt-3 border-t border-zinc-900/60 text-[10px] font-mono text-[var(--text-muted)]">
                      {item.effects ? (
                        <EffectEnvelopeView effects={item.effects} />
                      ) : (
                        <span className="text-zinc-600 italic text-[9px]">Vật phẩm phổ thông</span>
                      )}

                      {item.base_price ? (
                        <div className="flex items-center gap-1 text-amber-500 font-bold text-[10px]">
                          <Coins className="w-3.5 h-3.5" />
                          <span>
                            {item.base_price.toLocaleString()}{" "}
                            {item.rank && item.rank >= 6 ? "Tiên nguyên thạch" : "Nguyên thạch"}
                          </span>
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
                              {targetGu.rarity > 0 && (
                                <span className={`px-1.5 py-0.5 rounded bg-zinc-950/40 border border-zinc-800 text-[8px] font-mono font-bold ${rarityColors[targetGu.rarity] || "text-[var(--text-muted)]"}`}>
                                  {rarityLabels[targetGu.rarity] || `Độ hiếm ${targetGu.rarity}`}
                                </span>
                              )}
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
                              {Object.entries(recipe.required_gus).map(([guId, value]) => {
                                const reqGu = guMap.get(guId);
                                const reqName = reqGu ? reqGu.name : guId;
                                const qty = reqGuQty(value);
                                const rank = reqGuRank(value);
                                return (
                                  <div
                                    key={guId}
                                    onClick={() => locateGu(guId)}
                                    className="flex items-center justify-between text-[11px] font-mono p-1.5 px-2.5 rounded-lg bg-zinc-950/40 border border-zinc-900/60 hover:border-indigo-500/20 cursor-pointer transition-colors"
                                    title="Click để tìm Cổ trùng này"
                                  >
                                    <span className="text-[var(--text-primary)] hover:text-indigo-400 transition-colors truncate">
                                      {reqName}{rank > 0 ? ` (Bậc ${rank})` : ""}
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

        {/* 5. MONSTERS TAB */}
        {activeTab === "monsters" && (
          <div>
            {filteredMonsters.length === 0 ? (
              <div className="text-center py-20 bg-zinc-900/10 rounded-2xl border border-[var(--border)] border-dashed">
      wwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwww          <ShieldAlert className="w-8 h-8 mx-auto text-[var(--text-muted)] mb-2" />
                <p className="text-[var(--text-muted)] font-mono text-xs">Không tìm thấy quái vật nào trùng khớp với bộ lọc</p>
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {filteredMonsters.map((monster) => {
                  const regions = monsterRegionMap.get(monster.id) || [];
                  const lootTable = lootTableMap.get(monster.loot_table_id);

                  return (
                    <div
                      key={monster.id}
                      className="flex flex-col p-5 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)] shadow-sm hover:border-red-500/20 transition-all"
                    >
                      <div className="flex items-start justify-between gap-3 border-b border-zinc-900/80 pb-4 mb-4">
                        <div className="flex items-center gap-3">
                          <span className="text-2xl p-2 rounded-xl bg-zinc-950/60 border border-zinc-800/40">
                            {monster.type === "Boss" ? "👹" : monster.type === "Elite" ? "👺" : "🐺"}
                          </span>
                          <div>
                            <h3 className="text-sm font-serif font-bold text-[var(--text-primary)]">
                              {monster.name}
                            </h3>
                            <div className="flex items-center gap-2 mt-1">
                              <span className={`text-[9px] px-1.5 py-0.5 rounded font-mono font-bold border ${
                                monster.type === "Boss" ? "bg-red-950/30 text-red-400 border-red-500/20" :
                                monster.type === "Elite" ? "bg-orange-950/30 text-orange-400 border-orange-500/20" :
                                "bg-zinc-900/50 text-zinc-400 border-zinc-800"
                              }`}>
                                {monster.type}
                              </span>
                              <span className="text-[10px] font-mono text-[var(--text-muted)]">
                                ID: {monster.id}
                              </span>
                            </div>
                          </div>
                        </div>
                        <div className="text-right flex flex-col gap-1 items-end">
                          <span className="text-[10px] font-mono text-rose-400 px-2 py-0.5 rounded bg-rose-950/20 border border-rose-500/10 flex items-center gap-1">
                            ❤️ {monster.max_hp.toLocaleString()}
                          </span>
                          <div className="flex gap-1">
                            <span className="text-[9px] font-mono text-amber-400 px-1.5 py-0.5 rounded bg-amber-950/20">⚔️ {monster.base_attack}</span>
                            <span className="text-[9px] font-mono text-sky-400 px-1.5 py-0.5 rounded bg-sky-950/20">🛡️ {monster.base_defense}</span>
                          </div>
                        </div>
                      </div>

                      <p className="text-[11px] text-[var(--text-muted)] leading-relaxed mb-4 italic font-serif">
                        &ldquo;{monster.description}&rdquo;
                      </p>

                      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mt-auto pt-4 border-t border-zinc-900/60">
                        {/* Spawns */}
                        <div>
                          <h4 className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider mb-2 flex items-center gap-1.5">
                            <MapPin className="w-3 h-3 text-emerald-500" />
                            Khu vực xuất hiện
                          </h4>
                          {regions.length === 0 ? (
                            <p className="text-[10px] text-zinc-600 italic font-mono">Không rõ khu vực</p>
                          ) : (
                            <div className="flex flex-wrap gap-1">
                              {regions.map((rId) => {
                                const region = regionMap.get(rId);
                                return (
                                  <button
                                    key={rId}
                                    onClick={() => locateRegion(rId)}
                                    className="px-2 py-1 rounded bg-zinc-950/40 text-[10px] text-[var(--text-secondary)] border border-zinc-900/60 hover:border-emerald-500/30 hover:text-emerald-400 transition-colors cursor-pointer text-left"
                                  >
                                    {region ? region.name : rId}
                                  </button>
                                );
                              })}
                            </div>
                          )}
                        </div>

                        {/* Drops */}
                        <div>
                          <h4 className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider mb-2 flex items-center gap-1.5">
                            <Layers className="w-3 h-3 text-amber-500" />
                            Vật phẩm rơi (Loot)
                          </h4>
                          {!lootTable || lootTable.entries.length === 0 ? (
                            <p className="text-[10px] text-zinc-600 italic font-mono">Không có vật phẩm rơi</p>
                          ) : (
                            <div className="space-y-1.5">
                              {(expandedMonsters[monster.id] ? lootTable.entries : lootTable.entries.slice(0, 5)).map((e, idx) => {
                                const targetName = e.item_id
                                  ? (itemMap.get(e.item_id)?.name || e.item_id)
                                  : e.gu_id
                                    ? (guMap.get(e.gu_id)?.name || e.gu_id)
                                    : "Unknown";

                                return (
                                  <div key={idx} className="flex items-center justify-between text-[10px] font-mono p-1.5 rounded-lg bg-zinc-950/30 border border-zinc-900/40">
                                    <span className="text-[var(--text-primary)] truncate max-w-[120px]">
                                      {targetName}
                                    </span>
                                    <span className="text-zinc-500 shrink-0">{(e.chance * 100).toFixed(1)}%</span>
                                  </div>
                                );
                              })}
                              {lootTable.entries.length > 5 && (
                                <button
                                  onClick={() => toggleMonsterExpand(monster.id)}
                                  className="text-[9px] text-amber-500 hover:text-amber-400 italic text-center mt-1 cursor-pointer w-full hover:bg-amber-500/10 rounded py-0.5 transition-colors"
                                >
                                  {expandedMonsters[monster.id] ? "Thu gọn ▴" : `+ ${lootTable.entries.length - 5} mục khác ▾`}
                                </button>
                              )}
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

        {/* 6. REGIONS TAB */}
        {activeTab === "regions" && (
          <div>
            {filteredRegions.length === 0 ? (
              <div className="text-center py-20 bg-zinc-900/10 rounded-2xl border border-[var(--border)] border-dashed">
                <ShieldAlert className="w-8 h-8 mx-auto text-[var(--text-muted)] mb-2" />
                <p className="text-[var(--text-muted)] font-mono text-xs">Không tìm thấy khu vực nào trùng khớp với bộ lọc</p>
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {filteredRegions.map((region) => {
                  const wildGus = region.pool_weights?.wild_gus || [];
                  const resources = region.pool_weights?.resources || [];
                  const monstersInRegion = regionMonsterMap.get(region.id) || [];

                  return (
                    <div
                      key={region.id}
                      className="flex flex-col p-5 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)] shadow-sm hover:border-orange-500/20 transition-all"
                    >
                      <div className="flex items-start justify-between gap-3 border-b border-zinc-900/80 pb-4 mb-4">
                        <div className="flex items-center gap-3">
                          <span className="text-2xl p-2 rounded-xl bg-zinc-950/60 border border-zinc-800/40">
                            {region.zone_type === "Danger" ? "🌋" : region.zone_type === "Safe" ? "🏛️" : "🌲"}
                          </span>
                          <div>
                            <h3 className="text-sm font-serif font-bold text-[var(--text-primary)]">
                              {region.name}
                            </h3>
                            <div className="flex flex-wrap items-center gap-2 mt-1">
                              <span className={`text-[9px] px-1.5 py-0.5 rounded font-mono font-bold border ${
                                region.zone_type === "Danger" ? "bg-red-950/30 text-red-400 border-red-500/20" :
                                region.zone_type === "Safe" ? "bg-emerald-950/30 text-emerald-400 border-emerald-500/20" :
                                "bg-zinc-900/50 text-zinc-400 border-zinc-800"
                              }`}>
                                {region.zone_type}
                              </span>
                              <span className="text-[10px] font-mono text-[var(--text-muted)]">
                                Cấp {region.min_rank} - {region.max_rank}
                              </span>
                              {(region.min_x !== undefined && region.min_y !== undefined) && (
                                <span className="text-[10px] font-mono text-zinc-400 bg-zinc-900/50 px-1.5 py-0.5 rounded border border-zinc-800 flex items-center gap-1">
                                  <MapPin className="w-2.5 h-2.5" />
                                  {region.min_x}, {region.min_y} → {region.max_x}, {region.max_y}
                                </span>
                              )}
                            </div>
                          </div>
                        </div>
                      </div>

                      <p className="text-[11px] text-[var(--text-muted)] leading-relaxed mb-4 italic font-serif">
                        &ldquo;{region.description}&rdquo;
                      </p>

                      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mt-auto pt-4 border-t border-zinc-900/60">
                        {/* Spawns */}
                        <div>
                          <h4 className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider mb-2 flex items-center gap-1.5">
                            <Skull className="w-3 h-3 text-red-500" />
                            Quái vật xuất hiện
                          </h4>
                          {monstersInRegion.length === 0 ? (
                            <p className="text-[10px] text-zinc-600 italic font-mono">Khu vực an toàn</p>
                          ) : (
                            <div className="flex flex-col gap-1.5">
                              {(expandedRegions[region.id] ? monstersInRegion : monstersInRegion.slice(0, 5)).map((mId) => {
                                const monster = monsterMap.get(mId);
                                return (
                                  <button
                                    key={mId}
                                    onClick={() => locateMonster(mId)}
                                    className="px-2 py-1 rounded-lg bg-zinc-950/30 text-[10px] text-[var(--text-secondary)] border border-zinc-900/40 hover:border-red-500/30 hover:text-red-400 transition-colors cursor-pointer text-left truncate flex items-center justify-between"
                                  >
                                    <span className="truncate">{monster ? monster.name : mId}</span>
                                    {monster?.type === "Boss" && <span className="text-[8px] text-red-500 bg-red-950/40 px-1 rounded ml-1">BOSS</span>}
                                  </button>
                                );
                              })}
                              {monstersInRegion.length > 5 && (
                                <button
                                  onClick={() => toggleRegionExpand(region.id)}
                                  className="text-[9px] text-emerald-500 hover:text-emerald-400 italic text-center mt-1 cursor-pointer w-full hover:bg-emerald-500/10 rounded py-0.5 transition-colors"
                                >
                                  {expandedRegions[region.id] ? "Thu gọn ▴" : `+ ${monstersInRegion.length - 5} quái khác ▾`}
                                </button>
                              )}
                            </div>
                          )}
                        </div>

                        {/* Resources */}
                        <div>
                          <h4 className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider mb-2 flex items-center gap-1.5">
                            <Sparkles className="w-3 h-3 text-indigo-500" />
                            Tài nguyên & Cổ
                          </h4>
                          {resources.length === 0 && wildGus.length === 0 ? (
                            <p className="text-[10px] text-zinc-600 italic font-mono">Vùng đất cằn cỗi</p>
                          ) : (
                            <div className="flex flex-wrap gap-1">
                              {(expandedRegions[region.id] ? wildGus : wildGus.slice(0, 4)).map((gId) => {
                                const gu = guMap.get(gId);
                                return (
                                  <span key={`gu-${gId}`} className="px-1.5 py-0.5 rounded bg-indigo-950/30 text-indigo-400 border border-indigo-500/20 text-[9px] font-mono truncate max-w-full">
                                    {gu ? gu.name : gId}
                                  </span>
                                );
                              })}
                              {(expandedRegions[region.id] ? resources : resources.slice(0, 4)).map((rId) => {
                                const item = itemMap.get(rId);
                                return (
                                  <span key={`res-${rId}`} className="px-1.5 py-0.5 rounded bg-sky-950/30 text-sky-400 border border-sky-500/20 text-[9px] font-mono truncate max-w-full">
                                    {item ? item.name : rId}
                                  </span>
                                );
                              })}
                              {(wildGus.length + resources.length > 8) && !expandedRegions[region.id] && (
                                <button
                                  onClick={() => toggleRegionExpand(region.id)}
                                  className="px-1.5 py-0.5 rounded bg-emerald-950/30 text-emerald-400 border border-emerald-500/20 text-[9px] font-mono hover:bg-emerald-900/50 transition-colors cursor-pointer"
                                >
                                  +{wildGus.length + resources.length - 8} hiển thị tất cả
                                </button>
                              )}
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
