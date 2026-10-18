"use client";



import React, { useState, useMemo, useEffect, useCallback } from "react";
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
  Map as MapIcon,
  RotateCcw,
} from "lucide-react";
import { type GuDictionaryItem, type MonsterTemplate, type LootTable, type WikiRegion, type WikiMonsterSpawn, type WikiRegionLoot, reqGuQty, reqGuRank } from "@/lib/gu-dict";
import { indexGus, guAt } from "@/lib/gu-index";
import { type ItemSpec, type RefiningRecipe } from "@/store/useGameStore";
import { getPathColor } from "@/lib/utils";
import { translatePath } from "@/lib/translate";
import { EffectEnvelopeView } from "@/components/EffectBlockView";
import {
  EFFECT_CATEGORIES,
  extractGuSearchableText,
  matchesGuEffect,
  extractGuEffectBadges,
  removeAccents,
} from "@/lib/gu-effect-search";
import WikiPagination from "@/components/WikiPagination";
import { getPaginationInfo, type PageSize } from "@/lib/pagination";

// Types
type TabType = "guides" | "gus" | "items" | "recipes" | "monsters" | "regions";

const DEFAULT_PAGE_SIZES: Record<TabType, PageSize> = {
  guides: "all",
  gus: 36,
  items: 36,
  recipes: 30,
  monsters: 24,
  regions: 20,
};

const TAB_PAGE_SIZE_OPTIONS: Record<TabType, PageSize[]> = {
  guides: ["all"],
  gus: [24, 36, 60, 120, "all"],
  items: [24, 36, 60, 120, "all"],
  recipes: [20, 30, 50, 100, "all"],
  monsters: [24, 48, 96, "all"],
  regions: [10, 20, 40, "all"],
};

const TAB_ITEM_LABELS: Record<TabType, string> = {
  guides: "hướng dẫn",
  gus: "Cổ trùng",
  items: "Vật phẩm",
  recipes: "Cổ phương",
  monsters: "Quái vật",
  regions: "Khu vực",
};

interface Props {
  initialGus: GuDictionaryItem[];
  initialItems: ItemSpec[];
  initialRecipes: RefiningRecipe[];
  initialMonsters: MonsterTemplate[];
  initialLootTables: LootTable[];
  initialRegions: WikiRegion[];
  initialMonsterSpawns: WikiMonsterSpawn[];
  initialRegionLoot?: WikiRegionLoot[];
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

export default function WikiClientView({
  initialGus,
  initialItems,
  initialRecipes,
  initialMonsters,
  initialLootTables,
  initialRegions,
  initialMonsterSpawns,
  initialRegionLoot = [],
  guideSections,
  roadmap,
  subByParent,
  sectionIcon,
}: Props) {
  // Tabs State
  const [activeTab, setActiveTab] = useState<TabType>("guides");

  // Pagination State
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState<PageSize>(DEFAULT_PAGE_SIZES["guides"]);

  // Search State
  const [searchQuery, setSearchQuery] = useState("");
  const [showFilters, setShowFilters] = useState(false);

  // Gu Filters State
  const [guRank, setGuRank] = useState<string>("all");
  const [guPath, setGuPath] = useState<string>("all");
  const [guRarity, setGuRarity] = useState<string>("all");
  const [guType, setGuType] = useState<string>("all");
  const [guEffect, setGuEffect] = useState<string>("all");

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

  const resetFilters = () => {
    setSearchQuery("");
    setGuRank("all");
    setGuPath("all");
    setGuRarity("all");
    setGuType("all");
    setGuEffect("all");
    setItemRank("all");
    setItemType("all");
    setRecipeRank("all");
    setRecipeSort("none");
    setMonsterType("all");
    setMonsterRank("all");
    setRegionZoneType("all");
    setRegionRank("all");
    setCurrentPage(1);
  };

  // Reset current page when filters or search query change
  useEffect(() => {
    setCurrentPage(1);
  }, [
    searchQuery,
    guRank,
    guPath,
    guRarity,
    guType,
    guEffect,
    itemRank,
    itemType,
    recipeRank,
    recipeSort,
    monsterType,
    monsterRank,
    regionZoneType,
    regionRank,
  ]);

  // Read URL parameters on mount (e.g. /wiki?tab=recipes&search=xyz)
  useEffect(() => {
    if (typeof window === "undefined") return;
    const params = new URLSearchParams(window.location.search);
    const tabParam = params.get("tab") as TabType | null;
    const searchParam = params.get("search");
    if (tabParam && ["guides", "gus", "items", "recipes", "monsters", "regions"].includes(tabParam)) {
      setActiveTab(tabParam);
      setPageSize(DEFAULT_PAGE_SIZES[tabParam] || 36);
    }
    if (searchParam) {
      setSearchQuery(searchParam);
    }
  }, []);

  const hasActiveFilters = useMemo(() => {
    if (searchQuery.trim()) return true;
    if (activeTab === "gus") {
      return guRank !== "all" || guPath !== "all" || guRarity !== "all" || guType !== "all" || guEffect !== "all";
    }
    if (activeTab === "items") {
      return itemRank !== "all" || itemType !== "all";
    }
    if (activeTab === "recipes") {
      return recipeRank !== "all" || recipeSort !== "none";
    }
    if (activeTab === "monsters") {
      return monsterType !== "all" || monsterRank !== "all";
    }
    if (activeTab === "regions") {
      return regionZoneType !== "all" || regionRank !== "all";
    }
    return false;
  }, [activeTab, searchQuery, guRank, guPath, guRarity, guType, guEffect, itemRank, itemType, recipeRank, recipeSort, monsterType, monsterRank, regionZoneType, regionRank]);

  // Expand state for long lists in region and monster cards
  const [expandedRegions, setExpandedRegions] = useState<Record<string, boolean>>({});
  const [expandedMonsters, setExpandedMonsters] = useState<Record<string, boolean>>({});
  const [expandedRegionLoot, setExpandedRegionLoot] = useState<Record<string, boolean>>({});
  const [regionLootTab, setRegionLootTab] = useState<Record<string, "region" | "monster">>({});
  const [monsterDropTab, setMonsterDropTab] = useState<Record<string, "monster" | "region">>({});

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

  const toggleRegionLootExpand = (regionId: string) => {
    setExpandedRegionLoot(prev => ({
      ...prev,
      [regionId]: !prev[regionId]
    }));
  };

  const setRegionDropTab = (regionId: string, tab: "region" | "monster") => {
    setRegionLootTab(prev => ({
      ...prev,
      [regionId]: tab
    }));
  };

  const setMonsterLootTab = (monsterId: string, tab: "monster" | "region") => {
    setMonsterDropTab(prev => ({
      ...prev,
      [monsterId]: tab
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

  // Map Gu IDs (+ rank) to specs for Recipe lookup. Ids repeat across ranks.
  const guIndex = useMemo(() => indexGus(initialGus), [initialGus]);
  const guMap = guIndex.byId;
  const guFor = useCallback(
    (guId: string, rank?: number | null) => guAt(guIndex, guId, rank),
    [guIndex]
  );

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

  // Region loot map: region_id -> WikiRegionLoot[]
  const regionLootMap = useMemo(() => {
    const map = new Map<string, WikiRegionLoot[]>();
    initialRegionLoot.forEach((rl) => {
      if (!map.has(rl.region_id)) map.set(rl.region_id, []);
      map.get(rl.region_id)!.push(rl);
    });
    return map;
  }, [initialRegionLoot]);

  // Source lookup: item -> { gatherRegions, lootRegions, monsters }
  const itemSourceMap = useMemo(() => {
    const map = new Map<string, { gatherRegions: string[]; lootRegions: string[]; monsters: string[] }>();
    const ensure = (id: string) => {
      if (!map.has(id)) map.set(id, { gatherRegions: [], lootRegions: [], monsters: [] });
      return map.get(id)!;
    };

    // 1. From region pool_weights.resources (Thu thập/Khai thác)
    initialRegions.forEach((r) => {
      (r.pool_weights?.resources || []).forEach((itemId) => {
        const src = ensure(itemId);
        if (!src.gatherRegions.includes(r.id)) src.gatherRegions.push(r.id);
      });
    });

    // 2. From region loot tables (Loot khu vực)
    initialRegionLoot.forEach((rl) => {
      const lt = lootTableMap.get(rl.loot_table_id);
      if (!lt) return;
      lt.entries.forEach((e) => {
        if (e.item_id) {
          const src = ensure(e.item_id);
          if (!src.lootRegions.includes(rl.region_id)) src.lootRegions.push(rl.region_id);
        }
      });
    });

    // 3. From monster loot table entries (Loot từ quái)
    initialMonsters.forEach((m) => {
      if (!m.loot_table_id) return;
      const lt = lootTableMap.get(m.loot_table_id);
      if (!lt) return;
      lt.entries.forEach((e) => {
        if (e.item_id) {
          const src = ensure(e.item_id);
          if (!src.monsters.includes(m.id)) src.monsters.push(m.id);
        }
      });
    });

    return map;
  }, [initialRegions, initialRegionLoot, initialMonsters, lootTableMap]);

  // Source lookup: gu -> { wildRegions, lootRegions, dropMonsters }
  const guSourceMap = useMemo(() => {
    const map = new Map<string, { wildRegions: string[]; lootRegions: string[]; dropMonsters: string[] }>();
    const ensure = (id: string) => {
      if (!map.has(id)) map.set(id, { wildRegions: [], lootRegions: [], dropMonsters: [] });
      return map.get(id)!;
    };

    // 1. From region pool_weights.wild_gus (Hoang dã)
    initialRegions.forEach((r) => {
      (r.pool_weights?.wild_gus || []).forEach((guId) => {
        const src = ensure(guId);
        if (!src.wildRegions.includes(r.id)) src.wildRegions.push(r.id);
      });
    });

    // 2. From region loot tables (Loot khu vực)
    initialRegionLoot.forEach((rl) => {
      const lt = lootTableMap.get(rl.loot_table_id);
      if (!lt) return;
      lt.entries.forEach((e) => {
        if (e.gu_id) {
          const src = ensure(e.gu_id);
          if (!src.lootRegions.includes(rl.region_id)) src.lootRegions.push(rl.region_id);
        }
      });
    });

    // 3. From monster loot table entries (Loot từ quái)
    initialMonsters.forEach((m) => {
      if (!m.loot_table_id) return;
      const lt = lootTableMap.get(m.loot_table_id);
      if (!lt) return;
      lt.entries.forEach((e) => {
        if (e.gu_id) {
          const src = ensure(e.gu_id);
          if (!src.dropMonsters.includes(m.id)) src.dropMonsters.push(m.id);
        }
      });
    });

    return map;
  }, [initialRegions, initialRegionLoot, initialMonsters, lootTableMap]);

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

  // Search index for Gus including effects, food item names, tags, paths, and stats
  const guSearchIndex = useMemo(() => {
    const map = new Map<string, string>();
    initialGus.forEach((g) => {
      map.set(`${g.id}_${g.rank}`, extractGuSearchableText(g, itemMap));
    });
    return map;
  }, [initialGus, itemMap]);

  // Reset filters when changing tabs
  const handleTabChange = (tab: TabType) => {
    setActiveTab(tab);
    setSearchQuery("");
    setCurrentPage(1);
    setPageSize(DEFAULT_PAGE_SIZES[tab] || 36);
  };

  // Filtered lists
  const filteredGus = useMemo(() => {
    return initialGus.filter((gu) => {
      // Search check across name, id, description, effects, tags, food names
      if (cleanSearch) {
        const indexed = guSearchIndex.get(`${gu.id}_${gu.rank}`);
        if (indexed) {
          if (!indexed.includes(cleanSearch)) return false;
        } else {
          const fallback = extractGuSearchableText(gu, itemMap);
          if (!fallback.includes(cleanSearch)) return false;
        }
      }
      // Rank check
      if (guRank !== "all" && gu.rank !== Number(guRank)) return false;
      // Path check
      if (guPath !== "all" && !gu.path?.includes(guPath)) return false;
      // Rarity check
      if (guRarity !== "all" && gu.rarity !== Number(guRarity)) return false;
      // Type check
      if (guType !== "all" && gu.gu_type !== guType) return false;
      // Effect category check
      if (guEffect !== "all" && !matchesGuEffect(gu, guEffect)) return false;

      return true;
    }).sort((a, b) => (a.name || "").localeCompare(b.name || ""));
  }, [initialGus, cleanSearch, guRank, guPath, guRarity, guType, guEffect, guSearchIndex, itemMap]);

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
      const targetGu = guFor(recipe.target_gu_id, recipe.target_rank);
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
        const nameA = guFor(a.target_gu_id, a.target_rank)?.name || a.target_gu_id;
        const nameB = guFor(b.target_gu_id, b.target_rank)?.name || b.target_gu_id;
        return nameA.localeCompare(nameB);
      });
    }

    return list;
  }, [initialRecipes, cleanSearch, recipeRank, recipeSort, guFor]);

  const filteredMonsters = useMemo(() => {
    return initialMonsters.filter((monster) => {
      if (cleanSearch) {
        const nameClean = removeAccents((monster.name || "").toLowerCase());
        const idClean = removeAccents((monster.id || "").toLowerCase());
        const typeClean = removeAccents((monster.type || "").toLowerCase());
        let matches = nameClean.includes(cleanSearch) || idClean.includes(cleanSearch) || typeClean.includes(cleanSearch);

        if (!matches && monster.loot_table_id) {
          const lt = lootTableMap.get(monster.loot_table_id);
          if (lt) {
            matches = lt.entries.some((e) => {
              const targetId = e.item_id || e.gu_id || "";
              const targetName = e.gu_id ? guMap.get(targetId)?.name : itemMap.get(targetId)?.name;
              return removeAccents(targetId.toLowerCase()).includes(cleanSearch) ||
                (targetName && removeAccents(targetName.toLowerCase()).includes(cleanSearch));
            });
          }
        }

        if (!matches) return false;
      }
      if (monsterType !== "all" && monster.type !== monsterType) return false;
      return true;
    }).sort((a, b) => a.name.localeCompare(b.name));
  }, [initialMonsters, cleanSearch, monsterType, lootTableMap, itemMap, guMap]);

  const filteredRegions = useMemo(() => {
    return initialRegions.filter((region) => {
      if (cleanSearch) {
        const nameClean = removeAccents((region.name || "").toLowerCase());
        const idClean = removeAccents((region.id || "").toLowerCase());
        const descClean = removeAccents((region.description || "").toLowerCase());
        let matches = nameClean.includes(cleanSearch) || idClean.includes(cleanSearch) || descClean.includes(cleanSearch);

        if (!matches) {
          const rules = [
            ...(regionLootMap.get(region.id) || []),
            ...(region.parent_id ? (regionLootMap.get(region.parent_id) || []) : []),
          ];
          matches = rules.some((rl) => {
            const lt = lootTableMap.get(rl.loot_table_id);
            if (!lt) return false;
            return lt.entries.some((e) => {
              const targetId = e.item_id || e.gu_id || "";
              const targetName = e.gu_id ? guMap.get(targetId)?.name : itemMap.get(targetId)?.name;
              return removeAccents(targetId.toLowerCase()).includes(cleanSearch) ||
                (targetName && removeAccents(targetName.toLowerCase()).includes(cleanSearch));
            });
          });
        }

        if (!matches) {
          const res = region.pool_weights?.resources || [];
          const wild = region.pool_weights?.wild_gus || [];
          matches = res.some((rId) => {
            const itemName = itemMap.get(rId)?.name;
            return removeAccents(rId.toLowerCase()).includes(cleanSearch) ||
              (itemName && removeAccents(itemName.toLowerCase()).includes(cleanSearch));
          }) || wild.some((gId) => {
            const guName = guMap.get(gId)?.name;
            return removeAccents(gId.toLowerCase()).includes(cleanSearch) ||
              (guName && removeAccents(guName.toLowerCase()).includes(cleanSearch));
          });
        }

        if (!matches) return false;
      }
      if (regionZoneType !== "all" && region.zone_type !== regionZoneType) return false;
      return true;
    }).sort((a, b) => a.name.localeCompare(b.name));
  }, [initialRegions, cleanSearch, regionZoneType, regionLootMap, lootTableMap, itemMap, guMap]);

  // Pagination metadata and sliced items
  const currentTotalItems = useMemo(() => {
    switch (activeTab) {
      case "gus": return filteredGus.length;
      case "items": return filteredItems.length;
      case "recipes": return filteredRecipes.length;
      case "monsters": return filteredMonsters.length;
      case "regions": return filteredRegions.length;
      default: return 0;
    }
  }, [activeTab, filteredGus.length, filteredItems.length, filteredRecipes.length, filteredMonsters.length, filteredRegions.length]);

  const paginationInfo = useMemo(() => {
    return getPaginationInfo(currentTotalItems, currentPage, pageSize);
  }, [currentTotalItems, currentPage, pageSize]);

  const paginatedGus = useMemo(() => {
    if (activeTab !== "gus") return [];
    if (pageSize === "all") return filteredGus;
    return filteredGus.slice(paginationInfo.startIndex, paginationInfo.endIndex);
  }, [activeTab, pageSize, filteredGus, paginationInfo.startIndex, paginationInfo.endIndex]);

  const paginatedItems = useMemo(() => {
    if (activeTab !== "items") return [];
    if (pageSize === "all") return filteredItems;
    return filteredItems.slice(paginationInfo.startIndex, paginationInfo.endIndex);
  }, [activeTab, pageSize, filteredItems, paginationInfo.startIndex, paginationInfo.endIndex]);

  const paginatedRecipes = useMemo(() => {
    if (activeTab !== "recipes") return [];
    if (pageSize === "all") return filteredRecipes;
    return filteredRecipes.slice(paginationInfo.startIndex, paginationInfo.endIndex);
  }, [activeTab, pageSize, filteredRecipes, paginationInfo.startIndex, paginationInfo.endIndex]);

  const paginatedMonsters = useMemo(() => {
    if (activeTab !== "monsters") return [];
    if (pageSize === "all") return filteredMonsters;
    return filteredMonsters.slice(paginationInfo.startIndex, paginationInfo.endIndex);
  }, [activeTab, pageSize, filteredMonsters, paginationInfo.startIndex, paginationInfo.endIndex]);

  const paginatedRegions = useMemo(() => {
    if (activeTab !== "regions") return [];
    if (pageSize === "all") return filteredRegions;
    return filteredRegions.slice(paginationInfo.startIndex, paginationInfo.endIndex);
  }, [activeTab, pageSize, filteredRegions, paginationInfo.startIndex, paginationInfo.endIndex]);

  // Navigate to Gu tab and filter by name
  const locateGu = (guId: string, rank?: number | null) => {
    setActiveTab("gus");
    setPageSize(DEFAULT_PAGE_SIZES["gus"]);
    setCurrentPage(1);
    const target = guFor(guId, rank);
    if (target) {
      setSearchQuery(target.name);
    } else {
      setSearchQuery(guId);
    }
  };

  const locateItem = (itemId: string) => {
    setActiveTab("items");
    setPageSize(DEFAULT_PAGE_SIZES["items"]);
    setCurrentPage(1);
    const target = itemMap.get(itemId);
    if (target) {
      setSearchQuery(target.name);
    } else {
      setSearchQuery(itemId);
    }
  };

  const locateMonster = (monsterId: string) => {
    setActiveTab("monsters");
    setPageSize(DEFAULT_PAGE_SIZES["monsters"]);
    setCurrentPage(1);
    const target = monsterMap.get(monsterId);
    if (target) {
      setSearchQuery(target.name);
    } else {
      setSearchQuery(monsterId);
    }
  };

  const locateRegion = (regionId: string) => {
    setActiveTab("regions");
    setPageSize(DEFAULT_PAGE_SIZES["regions"]);
    setCurrentPage(1);
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
                      ? "Tìm Cổ trùng theo tên, hiệu ứng (choáng, hút máu, độc...), thức ăn, ID..."
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
                {hasActiveFilters && (
                  <span className="w-2 h-2 rounded-full bg-indigo-500 animate-pulse ml-0.5" />
                )}
              </button>
            </div>

            {/* Quick Effect Filters for Gus */}
            {activeTab === "gus" && (
              <div className="flex items-center gap-1.5 overflow-x-auto pb-1 pt-0.5 no-scrollbar">
                <span className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider shrink-0 flex items-center gap-1 mr-1">
                  <Sparkles className="w-3 h-3 text-indigo-400" />
                  Hiệu ứng:
                </span>
                {EFFECT_CATEGORIES.map((cat) => {
                  const isActive = guEffect === cat.id;
                  return (
                    <button
                      key={cat.id}
                      onClick={() => setGuEffect(cat.id)}
                      className={`px-2.5 py-1 rounded-lg text-xs font-mono shrink-0 transition-all flex items-center gap-1 cursor-pointer border ${
                        isActive
                          ? "bg-indigo-600/30 text-indigo-200 border-indigo-500/60 shadow-sm font-bold"
                          : "bg-zinc-900/40 text-zinc-400 border-zinc-800/60 hover:text-zinc-200 hover:border-zinc-700/80 hover:bg-zinc-800/30"
                      }`}
                    >
                      <span>{cat.icon}</span>
                      <span>{cat.label}</span>
                    </button>
                  );
                })}
              </div>
            )}

            {/* Filters Collapse Drawer */}
            {showFilters && (
              <div className="p-4 rounded-xl bg-zinc-950/30 border border-[var(--border)] grid grid-cols-1 sm:grid-cols-2 md:grid-cols-5 gap-3 animate-fadeIn">
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

                    {/* Gu Effect */}
                    <div className="space-y-1">
                      <label className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider">Hiệu ứng</label>
                      <select
                        value={guEffect}
                        onChange={(e) => setGuEffect(e.target.value)}
                        className="w-full bg-[var(--surface)] border border-[var(--border)] rounded-lg px-2.5 py-1.5 text-xs text-[var(--text-primary)] focus:outline-none"
                      >
                        {EFFECT_CATEGORIES.map((cat) => (
                          <option key={cat.id} value={cat.id}>
                            {cat.icon} {cat.label}
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

            {/* Results count, Quick reset, and Compact Pagination */}
            <div id="wiki-results-anchor" className="flex flex-wrap items-center justify-between gap-3 text-xs font-mono text-[var(--text-muted)] pt-1">
              <div className="flex flex-wrap items-center gap-3">
                <div>
                  <span>Hiển thị </span>
                  <span className="font-bold text-[var(--text-primary)]">
                    {activeTab === "gus"
                      ? filteredGus.length.toLocaleString()
                      : activeTab === "items"
                      ? filteredItems.length.toLocaleString()
                      : activeTab === "recipes"
                      ? filteredRecipes.length.toLocaleString()
                      : activeTab === "monsters"
                      ? filteredMonsters.length.toLocaleString()
                      : filteredRegions.length.toLocaleString()}
                  </span>
                  <span> / </span>
                  <span>
                    {activeTab === "gus"
                      ? initialGus.length.toLocaleString()
                      : activeTab === "items"
                      ? initialItems.length.toLocaleString()
                      : activeTab === "recipes"
                      ? initialRecipes.length.toLocaleString()
                      : activeTab === "monsters"
                      ? initialMonsters.length.toLocaleString()
                      : initialRegions.length.toLocaleString()}
                  </span>
                  <span> {TAB_ITEM_LABELS[activeTab]}</span>
                </div>
                {hasActiveFilters && (
                  <button
                    onClick={resetFilters}
                    className="inline-flex items-center gap-1.5 text-xs text-rose-400 hover:text-rose-300 font-mono transition-colors cursor-pointer"
                  >
                    <RotateCcw className="w-3 h-3" />
                    <span>Đặt lại bộ lọc</span>
                  </button>
                )}
              </div>

              {/* Compact pagination controls at top */}
              <WikiPagination
                compact
                currentPage={paginationInfo.currentPage}
                totalPages={paginationInfo.totalPages}
                totalItems={paginationInfo.totalItems}
                pageSize={pageSize}
                startIndex={paginationInfo.startIndex}
                endIndex={paginationInfo.endIndex}
                onPageChange={setCurrentPage}
                onPageSizeChange={setPageSize}
                pageSizeOptions={TAB_PAGE_SIZE_OPTIONS[activeTab]}
                itemLabel={TAB_ITEM_LABELS[activeTab]}
                scrollTargetId="wiki-results-anchor"
              />
            </div>
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
              <>
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                {paginatedGus.map((gu) => {
                  const effectBadges = extractGuEffectBadges(gu, 3);
                  return (
                    <Link
                      key={gu.id + "_" + gu.rank}
                      href={`/wiki/gu/${gu.id}?rank=${gu.rank}`}
                      className="group flex flex-col p-4 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)] hover:border-indigo-500/40 transition-all hover:bg-[var(--bg-elevated)] hover:-translate-y-0.5 duration-200 shadow-sm hover:shadow-md cursor-pointer"
                    >
                      <div className="flex items-start justify-between gap-2">
                        <div className="flex items-center gap-3 min-w-0">
                          <span className="text-xl shrink-0 p-2 rounded-xl bg-zinc-950/60 border border-zinc-800/40 group-hover:border-indigo-500/30 transition-colors">
                            {getGuTypeIcon(gu.gu_type)}
                          </span>
                          <div className="min-w-0">
                            <h3 className="text-sm font-serif font-bold text-[var(--text-primary)] group-hover:text-indigo-400 transition-colors truncate">
                              {gu.name}
                            </h3>
                            <div className="flex items-center gap-1.5 mt-0.5">
                              <span className="text-[10px] text-zinc-400 font-mono">
                                Chuyển {gu.rank}
                              </span>
                              <span className="text-zinc-600">·</span>
                              <span className="text-[10px] text-[var(--text-muted)] font-mono">
                                {gu.gu_type}
                              </span>
                              {gu.rank >= 6 && (
                                <span className="px-1.5 py-0.2 text-[9px] font-mono font-bold text-amber-400 bg-amber-950/40 border border-amber-500/30 rounded">
                                  Tiên Cổ
                                </span>
                              )}
                            </div>
                          </div>
                        </div>
                        {gu.rarity > 0 && (
                          <span className={`px-2 py-0.5 rounded-md text-[9px] font-mono font-bold shrink-0 border ${rarityColors[gu.rarity] || "text-[var(--text-muted)] border-zinc-800/60"}`}>
                            {rarityLabels[gu.rarity] || `Độ hiếm ${gu.rarity}`}
                          </span>
                        )}
                      </div>

                      {gu.description && (
                        <p className="text-[11px] text-[var(--text-muted)] leading-relaxed mt-3 line-clamp-2 italic font-serif">
                          &ldquo;{gu.description}&rdquo;
                        </p>
                      )}

                      {/* Effects Preview */}
                      {effectBadges.length > 0 && (
                        <div className="flex flex-wrap gap-1 mt-2.5">
                          {effectBadges.map((badge, idx) => (
                            <span
                              key={idx}
                              className={`px-1.5 py-0.5 rounded text-[8px] font-mono border flex items-center gap-1 ${badge.color}`}
                            >
                              <span>{badge.icon}</span>
                              <span className="truncate max-w-[130px]">{badge.text}</span>
                            </span>
                          ))}
                        </div>
                      )}

                      {/* Food Types with Item Names */}
                      {gu.food_types && gu.food_types.length > 0 && (
                        <div className="flex items-center gap-1.5 mt-2.5 pt-2 border-t border-zinc-900/60 text-[9px] font-mono text-[var(--text-muted)]">
                          <span className="text-rose-400 shrink-0 font-semibold">🍖 Thức ăn:</span>
                          <div className="flex flex-wrap gap-1 min-w-0">
                            {gu.food_types.slice(0, 2).map((foodId) => {
                              const foodItem = itemMap.get(foodId);
                              const foodName = foodItem ? foodItem.name : foodId;
                              return (
                                <span
                                  key={foodId}
                                  className="px-1.5 py-0.2 rounded bg-rose-950/20 text-rose-300 border border-rose-500/15 text-[8px] truncate max-w-[110px]"
                                  title={foodName}
                                >
                                  {foodName}
                                </span>
                              );
                            })}
                            {gu.food_types.length > 2 && (
                              <span className="text-[8px] text-[var(--text-muted)] self-center">
                                +{gu.food_types.length - 2}
                              </span>
                            )}
                          </div>
                        </div>
                      )}

                      <div className="flex flex-wrap gap-1 mt-auto pt-2.5 border-t border-zinc-900/60">
                        {gu.path && gu.path.map((pathName) => {
                          const pathColor = getPathColor(pathName);
                          return (
                            <span
                              key={pathName}
                              className={`px-1.5 py-0.5 rounded text-[8px] font-mono font-semibold border ${pathColor.bg} ${pathColor.text} ${pathColor.border}`}
                            >
                              {translatePath(pathName) || pathName}
                            </span>
                          );
                        })}
                        {gu.combat_tags && gu.combat_tags.slice(0, 2).map((tag) => (
                          <span
                            key={tag}
                            className="px-1.5 py-0.5 rounded bg-zinc-900/50 text-[8px] text-zinc-500 border border-zinc-800/40 font-mono"
                          >
                            #{tag}
                          </span>
                        ))}
                      </div>

                      {/* Source Locations */}
                      {guSourceMap.has(gu.id) && (
                        (guSourceMap.get(gu.id)!.wildRegions.length > 0 ||
                         guSourceMap.get(gu.id)!.lootRegions.length > 0 ||
                         guSourceMap.get(gu.id)!.dropMonsters.length > 0)
                      ) && (
                        <div className="flex flex-wrap gap-1 mt-2 pt-2 border-t border-zinc-900/40">
                          {guSourceMap.get(gu.id)!.dropMonsters.slice(0, 2).map((mId) => {
                            const monster = monsterMap.get(mId);
                            return (
                              <button
                                key={`m-${mId}`}
                                type="button"
                                onClick={(ev) => {
                                  ev.preventDefault();
                                  ev.stopPropagation();
                                  locateMonster(mId);
                                }}
                                className="px-1.5 py-0.5 rounded bg-red-950/30 text-red-400 border border-red-500/20 text-[8px] font-mono flex items-center gap-1 hover:border-red-500/40 cursor-pointer"
                                title={`Rơi từ quái: ${monster ? monster.name : mId}`}
                              >
                                <Skull className="w-2.5 h-2.5" />
                                <span className="truncate max-w-[85px]">{monster ? monster.name : mId}</span>
                              </button>
                            );
                          })}
                          {guSourceMap.get(gu.id)!.lootRegions.slice(0, 2).map((rId) => {
                            const region = regionMap.get(rId);
                            return (
                              <button
                                key={`lr-${rId}`}
                                type="button"
                                onClick={(ev) => {
                                  ev.preventDefault();
                                  ev.stopPropagation();
                                  locateRegion(rId);
                                }}
                                className="px-1.5 py-0.5 rounded bg-amber-950/30 text-amber-400 border border-amber-500/20 text-[8px] font-mono flex items-center gap-1 hover:border-amber-500/40 cursor-pointer"
                                title={`Loot khu vực: ${region ? region.name : rId}`}
                              >
                                <Layers className="w-2.5 h-2.5" />
                                <span className="truncate max-w-[85px]">{region ? region.name : rId}</span>
                              </button>
                            );
                          })}
                          {guSourceMap.get(gu.id)!.wildRegions.slice(0, 2).map((rId) => {
                            const region = regionMap.get(rId);
                            return (
                              <button
                                key={`w-${rId}`}
                                type="button"
                                onClick={(ev) => {
                                  ev.preventDefault();
                                  ev.stopPropagation();
                                  locateRegion(rId);
                                }}
                                className="px-1.5 py-0.5 rounded bg-orange-950/30 text-orange-400 border border-orange-500/20 text-[8px] font-mono flex items-center gap-1 hover:border-orange-500/40 cursor-pointer"
                                title={`Hoang dã tại: ${region ? region.name : rId}`}
                              >
                                <MapPin className="w-2.5 h-2.5" />
                                <span className="truncate max-w-[85px]">{region ? region.name : rId}</span>
                              </button>
                            );
                          })}
                        </div>
                      )}
                    </Link>
                  );
                })}
              </div>
              <WikiPagination
                currentPage={paginationInfo.currentPage}
                totalPages={paginationInfo.totalPages}
                totalItems={paginationInfo.totalItems}
                pageSize={pageSize}
                startIndex={paginationInfo.startIndex}
                endIndex={paginationInfo.endIndex}
                onPageChange={setCurrentPage}
                onPageSizeChange={setPageSize}
                pageSizeOptions={TAB_PAGE_SIZE_OPTIONS["gus"]}
                itemLabel="Cổ trùng"
                scrollTargetId="wiki-results-anchor"
              />
            </>
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
              <>
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                {paginatedItems.map((item) => (
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
                    {itemSourceMap.has(item.id) && (
                      (itemSourceMap.get(item.id)!.monsters.length > 0 ||
                       itemSourceMap.get(item.id)!.lootRegions.length > 0 ||
                       itemSourceMap.get(item.id)!.gatherRegions.length > 0)
                    ) && (
                      <div className="flex flex-wrap gap-1 mt-2">
                        {itemSourceMap.get(item.id)!.monsters.slice(0, 2).map((mId) => {
                          const monster = monsterMap.get(mId);
                          return (
                            <button
                              key={`m-${mId}`}
                              type="button"
                              onClick={() => locateMonster(mId)}
                              className="px-1.5 py-0.5 rounded bg-red-950/30 text-red-400 border border-red-500/20 text-[8px] font-mono flex items-center gap-1 hover:border-red-500/40 cursor-pointer"
                              title={`Rơi từ quái: ${monster ? monster.name : mId}`}
                            >
                              <Skull className="w-2.5 h-2.5" />
                              <span className="truncate max-w-[85px]">{monster ? monster.name : mId}</span>
                            </button>
                          );
                        })}
                        {itemSourceMap.get(item.id)!.lootRegions.slice(0, 2).map((rId) => {
                          const region = regionMap.get(rId);
                          return (
                            <button
                              key={`lr-${rId}`}
                              type="button"
                              onClick={() => locateRegion(rId)}
                              className="px-1.5 py-0.5 rounded bg-amber-950/30 text-amber-400 border border-amber-500/20 text-[8px] font-mono flex items-center gap-1 hover:border-amber-500/40 cursor-pointer"
                              title={`Loot khu vực: ${region ? region.name : rId}`}
                            >
                              <Layers className="w-2.5 h-2.5" />
                              <span className="truncate max-w-[85px]">{region ? region.name : rId}</span>
                            </button>
                          );
                        })}
                        {itemSourceMap.get(item.id)!.gatherRegions.slice(0, 2).map((rId) => {
                          const region = regionMap.get(rId);
                          return (
                            <button
                              key={`gr-${rId}`}
                              type="button"
                              onClick={() => locateRegion(rId)}
                              className="px-1.5 py-0.5 rounded bg-orange-950/30 text-orange-400 border border-orange-500/20 text-[8px] font-mono flex items-center gap-1 hover:border-orange-500/40 cursor-pointer"
                              title={`Thu thập tại: ${region ? region.name : rId}`}
                            >
                              <MapPin className="w-2.5 h-2.5" />
                              <span className="truncate max-w-[85px]">{region ? region.name : rId}</span>
                            </button>
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
              <WikiPagination
                currentPage={paginationInfo.currentPage}
                totalPages={paginationInfo.totalPages}
                totalItems={paginationInfo.totalItems}
                pageSize={pageSize}
                startIndex={paginationInfo.startIndex}
                endIndex={paginationInfo.endIndex}
                onPageChange={setCurrentPage}
                onPageSizeChange={setPageSize}
                pageSizeOptions={TAB_PAGE_SIZE_OPTIONS["items"]}
                itemLabel="Vật phẩm"
                scrollTargetId="wiki-results-anchor"
              />
            </>
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
              <>
                <div className="space-y-3">
                {paginatedRecipes.map((recipe) => {
                  const targetGu = guFor(recipe.target_gu_id, recipe.target_rank);
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
                            onClick={() => locateGu(recipe.target_gu_id, recipe.target_rank)}
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
                                const qty = reqGuQty(value);
                                const rank = reqGuRank(value);
                                const reqGu = guFor(guId, rank || null);
                                const reqName = reqGu ? reqGu.name : guId;
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
              <WikiPagination
                currentPage={paginationInfo.currentPage}
                totalPages={paginationInfo.totalPages}
                totalItems={paginationInfo.totalItems}
                pageSize={pageSize}
                startIndex={paginationInfo.startIndex}
                endIndex={paginationInfo.endIndex}
                onPageChange={setCurrentPage}
                onPageSizeChange={setPageSize}
                pageSizeOptions={TAB_PAGE_SIZE_OPTIONS["recipes"]}
                itemLabel="Cổ phương"
                scrollTargetId="wiki-results-anchor"
              />
            </>
          )}
        </div>
        )}

        {/* 5. MONSTERS TAB */}
        {activeTab === "monsters" && (
          <div>
            {filteredMonsters.length === 0 ? (
              <div className="text-center py-20 bg-zinc-900/10 rounded-2xl border border-[var(--border)] border-dashed">
                <ShieldAlert className="w-8 h-8 mx-auto text-[var(--text-muted)] mb-2" />
                <p className="text-[var(--text-muted)] font-mono text-xs">Không tìm thấy quái vật nào trùng khớp với bộ lọc</p>
              </div>
            ) : (
              <>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {paginatedMonsters.map((monster) => {
                  const regions = monsterRegionMap.get(monster.id) || [];
                  const lootTable = lootTableMap.get(monster.loot_table_id);
                  const monsterLootEntries = lootTable?.entries || [];

                  // Region loot that this monster can drop when fought in its spawning regions
                  const monsterRegionLootEntries: Array<{
                    id: string;
                    isGu: boolean;
                    name: string;
                    chance: number;
                    minQty: number;
                    maxQty: number;
                    regionId: string;
                    regionName: string;
                  }> = [];

                  const seenRegionDropKeys = new Set<string>();
                  regions.forEach((rId) => {
                    const regObj = regionMap.get(rId);
                    const rules = [
                      ...(regionLootMap.get(rId) || []),
                      ...(regObj?.parent_id ? (regionLootMap.get(regObj.parent_id) || []) : []),
                    ];
                    const regionName = regObj?.name || rId;
                    rules.forEach((rl) => {
                      const mType = (monster.type || "Normal").toLowerCase();
                      const rlType = (rl.monster_type || "Normal").toLowerCase();
                      const matches = rlType === mType
                        || (mType === "boss" && rlType === "boss")
                        || (mType === "elite" && rlType === "elite")
                        || (mType !== "boss" && mType !== "elite" && rlType === "normal");

                      if (!matches) return;
                      const lt = lootTableMap.get(rl.loot_table_id);
                      if (!lt) return;
                      lt.entries.forEach((e) => {
                        const isGu = !!e.gu_id;
                        const targetId = e.item_id || e.gu_id || "";
                        if (!targetId) return;
                        const dedupKey = `${rId}_${targetId}`;
                        if (seenRegionDropKeys.has(dedupKey)) return;
                        seenRegionDropKeys.add(dedupKey);

                        const targetName = isGu
                          ? (guMap.get(targetId)?.name || targetId)
                          : (itemMap.get(targetId)?.name || targetId);

                        monsterRegionLootEntries.push({
                          id: targetId,
                          isGu,
                          name: targetName,
                          chance: e.chance,
                          minQty: e.min_qty || 1,
                          maxQty: e.max_qty || 1,
                          regionId: rId,
                          regionName,
                        });
                      });
                    });
                  });

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
                                    type="button"
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

                        {/* Drops: Show both Monster Loot and Region Loot */}
                        <div>
                          <div className="flex items-center justify-between mb-2">
                            <h4 className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider flex items-center gap-1.5">
                              <Layers className="w-3 h-3 text-amber-500" />
                              Vật phẩm rơi (Loot)
                            </h4>
                            {monsterRegionLootEntries.length > 0 && (
                              <div className="flex items-center gap-1">
                                <button
                                  type="button"
                                  onClick={() => setMonsterLootTab(monster.id, "monster")}
                                  className={`px-1.5 py-0.5 rounded text-[9px] font-mono transition-colors cursor-pointer ${
                                    (monsterDropTab[monster.id] || "monster") === "monster"
                                      ? "bg-amber-500/20 text-amber-300 border border-amber-500/40"
                                      : "text-zinc-500 hover:text-zinc-300 border border-transparent"
                                  }`}
                                >
                                  Loot quái ({monsterLootEntries.length})
                                </button>
                                <button
                                  type="button"
                                  onClick={() => setMonsterLootTab(monster.id, "region")}
                                  className={`px-1.5 py-0.5 rounded text-[9px] font-mono transition-colors cursor-pointer ${
                                    monsterDropTab[monster.id] === "region"
                                      ? "bg-orange-500/20 text-orange-300 border border-orange-500/40"
                                      : "text-zinc-500 hover:text-zinc-300 border border-transparent"
                                  }`}
                                >
                                  Loot khu vực ({monsterRegionLootEntries.length})
                                </button>
                              </div>
                            )}
                          </div>

                          {(monsterDropTab[monster.id] || "monster") === "monster" ? (
                            monsterLootEntries.length === 0 ? (
                              <p className="text-[10px] text-zinc-600 italic font-mono">Không có vật phẩm rơi riêng</p>
                            ) : (
                              <div className="space-y-1.5">
                                {(expandedMonsters[monster.id] ? monsterLootEntries : monsterLootEntries.slice(0, 5)).map((e, idx) => {
                                  const targetId = e.item_id || e.gu_id || "";
                                  const isGu = !!e.gu_id;
                                  const targetName = isGu
                                    ? (guMap.get(targetId)?.name || targetId)
                                    : (itemMap.get(targetId)?.name || targetId);

                                  return (
                                    <div key={idx} className="flex items-center justify-between text-[10px] font-mono p-1.5 rounded-lg bg-zinc-950/30 border border-zinc-900/40">
                                      <button
                                        type="button"
                                        onClick={() => isGu ? locateGu(targetId) : locateItem(targetId)}
                                        className="text-[var(--text-primary)] hover:text-amber-400 transition-colors truncate max-w-[130px] text-left cursor-pointer"
                                        title={`Xem ${isGu ? "Cổ trùng" : "Vật phẩm"}: ${targetName}`}
                                      >
                                        {targetName}
                                        {(e.min_qty && e.max_qty && (e.min_qty > 1 || e.max_qty > 1)) ? (
                                          <span className="text-[9px] text-zinc-500 ml-1">
                                            (x{e.min_qty === e.max_qty ? e.min_qty : `${e.min_qty}-${e.max_qty}`})
                                          </span>
                                        ) : null}
                                      </button>
                                      <span className="text-zinc-500 shrink-0">{(e.chance * 100).toFixed(1)}%</span>
                                    </div>
                                  );
                                })}
                                {monsterLootEntries.length > 5 && (
                                  <button
                                    type="button"
                                    onClick={() => toggleMonsterExpand(monster.id)}
                                    className="text-[9px] text-amber-500 hover:text-amber-400 italic text-center mt-1 cursor-pointer w-full hover:bg-amber-500/10 rounded py-0.5 transition-colors"
                                  >
                                    {expandedMonsters[monster.id] ? "Thu gọn ▴" : `+ ${monsterLootEntries.length - 5} mục khác ▾`}
                                  </button>
                                )}
                              </div>
                            )
                          ) : (
                            <div className="space-y-1.5">
                              {(expandedMonsters[monster.id] ? monsterRegionLootEntries : monsterRegionLootEntries.slice(0, 5)).map((e, idx) => (
                                <div key={idx} className="flex items-center justify-between text-[10px] font-mono p-1.5 rounded-lg bg-zinc-950/30 border border-zinc-900/40">
                                  <div className="flex items-center gap-1 min-w-0">
                                    <button
                                      type="button"
                                      onClick={() => e.isGu ? locateGu(e.id) : locateItem(e.id)}
                                      className="text-[var(--text-primary)] hover:text-orange-400 transition-colors truncate max-w-[110px] text-left cursor-pointer"
                                      title={`Xem: ${e.name}`}
                                    >
                                      {e.name}
                                    </button>
                                    <button
                                      type="button"
                                      onClick={() => locateRegion(e.regionId)}
                                      className="px-1 py-0.2 rounded bg-orange-950/30 text-orange-400 border border-orange-500/20 text-[8px] truncate max-w-[75px] hover:border-orange-500/40 cursor-pointer"
                                      title={`Khu vực: ${e.regionName}`}
                                    >
                                      {e.regionName}
                                    </button>
                                  </div>
                                  <span className="text-zinc-500 shrink-0">{(e.chance * 100).toFixed(1)}%</span>
                                </div>
                              ))}
                              {monsterRegionLootEntries.length > 5 && (
                                <button
                                  type="button"
                                  onClick={() => toggleMonsterExpand(monster.id)}
                                  className="text-[9px] text-orange-500 hover:text-orange-400 italic text-center mt-1 cursor-pointer w-full hover:bg-orange-500/10 rounded py-0.5 transition-colors"
                                >
                                  {expandedMonsters[monster.id] ? "Thu gọn ▴" : `+ ${monsterRegionLootEntries.length - 5} mục khác ▾`}
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
              <WikiPagination
                currentPage={paginationInfo.currentPage}
                totalPages={paginationInfo.totalPages}
                totalItems={paginationInfo.totalItems}
                pageSize={pageSize}
                startIndex={paginationInfo.startIndex}
                endIndex={paginationInfo.endIndex}
                onPageChange={setCurrentPage}
                onPageSizeChange={setPageSize}
                pageSizeOptions={TAB_PAGE_SIZE_OPTIONS["monsters"]}
                itemLabel="Quái vật"
                scrollTargetId="wiki-results-anchor"
              />
            </>
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
              <>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {paginatedRegions.map((region) => {
                  const wildGus = region.pool_weights?.wild_gus || [];
                  const resources = region.pool_weights?.resources || [];
                  const monstersInRegion = regionMonsterMap.get(region.id) || [];

                  // 1. Region loot from RegionLootTables for this region
                  const regionLootRules = [
                    ...(regionLootMap.get(region.id) || []),
                    ...(region.parent_id ? (regionLootMap.get(region.parent_id) || []) : []),
                  ];
                  const regionLootEntries: Array<{
                    id: string;
                    isGu: boolean;
                    name: string;
                    chance: number;
                    minQty: number;
                    maxQty: number;
                    monsterType: string;
                  }> = [];

                  const seenRegionDropKeys = new Set<string>();
                  regionLootRules.forEach((rl) => {
                    const lt = lootTableMap.get(rl.loot_table_id);
                    if (!lt) return;
                    lt.entries.forEach((e) => {
                      const isGu = !!e.gu_id;
                      const targetId = e.item_id || e.gu_id || "";
                      if (!targetId) return;
                      const dedupKey = `${rl.monster_type}_${targetId}`;
                      if (seenRegionDropKeys.has(dedupKey)) return;
                      seenRegionDropKeys.add(dedupKey);

                      const name = isGu
                        ? (guMap.get(targetId)?.name || targetId)
                        : (itemMap.get(targetId)?.name || targetId);

                      regionLootEntries.push({
                        id: targetId,
                        isGu,
                        name,
                        chance: e.chance,
                        minQty: e.min_qty || 1,
                        maxQty: e.max_qty || 1,
                        monsterType: rl.monster_type,
                      });
                    });
                  });

                  // 2. Monster loot from monsters that inhabit this region
                  const regionMonsterLootEntries: Array<{
                    id: string;
                    isGu: boolean;
                    name: string;
                    chance: number;
                    minQty: number;
                    maxQty: number;
                    monsterName: string;
                    monsterId: string;
                  }> = [];

                  const seenMonsterDropKeys = new Set<string>();
                  monstersInRegion.forEach((mId) => {
                    const m = monsterMap.get(mId);
                    if (!m || !m.loot_table_id) return;
                    const lt = lootTableMap.get(m.loot_table_id);
                    if (!lt) return;
                    lt.entries.forEach((e) => {
                      const isGu = !!e.gu_id;
                      const targetId = e.item_id || e.gu_id || "";
                      if (!targetId) return;
                      const dedupKey = `${mId}_${targetId}`;
                      if (seenMonsterDropKeys.has(dedupKey)) return;
                      seenMonsterDropKeys.add(dedupKey);

                      const name = isGu
                        ? (guMap.get(targetId)?.name || targetId)
                        : (itemMap.get(targetId)?.name || targetId);

                      regionMonsterLootEntries.push({
                        id: targetId,
                        isGu,
                        name,
                        chance: e.chance,
                        minQty: e.min_qty || 1,
                        maxQty: e.max_qty || 1,
                        monsterName: m.name,
                        monsterId: m.id,
                      });
                    });
                  });

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
                                    type="button"
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
                                  type="button"
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
                                  <button
                                    key={`gu-${gId}`}
                                    type="button"
                                    onClick={() => locateGu(gId)}
                                    className="px-1.5 py-0.5 rounded bg-indigo-950/30 text-indigo-400 border border-indigo-500/20 text-[9px] font-mono truncate max-w-full hover:border-indigo-500/40 cursor-pointer"
                                    title={`Cổ trùng: ${gu ? gu.name : gId}`}
                                  >
                                    {gu ? gu.name : gId}
                                  </button>
                                );
                              })}
                              {(expandedRegions[region.id] ? resources : resources.slice(0, 4)).map((rId) => {
                                const item = itemMap.get(rId);
                                return (
                                  <button
                                    key={`res-${rId}`}
                                    type="button"
                                    onClick={() => locateItem(rId)}
                                    className="px-1.5 py-0.5 rounded bg-sky-950/30 text-sky-400 border border-sky-500/20 text-[9px] font-mono truncate max-w-full hover:border-sky-500/40 cursor-pointer"
                                    title={`Vật phẩm: ${item ? item.name : rId}`}
                                  >
                                    {item ? item.name : rId}
                                  </button>
                                );
                              })}
                              {(wildGus.length + resources.length > 8) && !expandedRegions[region.id] && (
                                <button
                                  type="button"
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

                      {/* Region Drops & Loot: Show both Region Loot and Monster Loot */}
                      <div className="mt-4 pt-4 border-t border-zinc-900/60">
                        <div className="flex items-center justify-between mb-2">
                          <h4 className="text-[10px] text-[var(--text-muted)] font-mono uppercase tracking-wider flex items-center gap-1.5">
                            <Layers className="w-3 h-3 text-amber-500" />
                            Chiến lợi phẩm (Loot)
                          </h4>
                          <div className="flex items-center gap-1">
                            <button
                              type="button"
                              onClick={() => setRegionDropTab(region.id, "region")}
                              className={`px-1.5 py-0.5 rounded text-[9px] font-mono transition-colors cursor-pointer ${
                                (regionLootTab[region.id] || "region") === "region"
                                  ? "bg-amber-500/20 text-amber-300 border border-amber-500/40"
                                  : "text-zinc-500 hover:text-zinc-300 border border-transparent"
                              }`}
                            >
                              Loot khu vực ({regionLootEntries.length})
                            </button>
                            <button
                              type="button"
                              onClick={() => setRegionDropTab(region.id, "monster")}
                              className={`px-1.5 py-0.5 rounded text-[9px] font-mono transition-colors cursor-pointer ${
                                regionLootTab[region.id] === "monster"
                                  ? "bg-red-500/20 text-red-300 border border-red-500/40"
                                  : "text-zinc-500 hover:text-zinc-300 border border-transparent"
                              }`}
                            >
                              Loot từ quái ({regionMonsterLootEntries.length})
                            </button>
                          </div>
                        </div>

                        {(regionLootTab[region.id] || "region") === "region" ? (
                          regionLootEntries.length === 0 ? (
                            <p className="text-[10px] text-zinc-600 italic font-mono">Chưa ghi nhận rơi đồ đặc trưng khu vực</p>
                          ) : (
                            <div className="space-y-1.5">
                              {(expandedRegionLoot[region.id] ? regionLootEntries : regionLootEntries.slice(0, 5)).map((e, idx) => (
                                <div key={idx} className="flex items-center justify-between text-[10px] font-mono p-1.5 rounded-lg bg-zinc-950/30 border border-zinc-900/40">
                                  <div className="flex items-center gap-1.5 min-w-0">
                                    <button
                                      type="button"
                                      onClick={() => e.isGu ? locateGu(e.id) : locateItem(e.id)}
                                      className="text-[var(--text-primary)] hover:text-amber-400 transition-colors truncate max-w-[120px] text-left cursor-pointer"
                                      title={`Xem: ${e.name}`}
                                    >
                                      {e.name}
                                      {(e.minQty > 1 || e.maxQty > 1) ? (
                                        <span className="text-[9px] text-zinc-500 ml-1">
                                          (x{e.minQty === e.maxQty ? e.minQty : `${e.minQty}-${e.maxQty}`})
                                        </span>
                                      ) : null}
                                    </button>
                                    <span className={`text-[8px] px-1 py-0.2 rounded font-mono ${
                                      e.monsterType === "Boss" ? "bg-red-950/30 text-red-400 border border-red-500/20" :
                                      e.monsterType === "Elite" ? "bg-purple-950/30 text-purple-400 border border-purple-500/20" :
                                      "bg-zinc-900 text-zinc-400 border border-zinc-800"
                                    }`}>
                                      {e.monsterType}
                                    </span>
                                  </div>
                                  <span className="text-zinc-500 shrink-0">{(e.chance * 100).toFixed(1)}%</span>
                                </div>
                              ))}
                              {regionLootEntries.length > 5 && (
                                <button
                                  type="button"
                                  onClick={() => toggleRegionLootExpand(region.id)}
                                  className="text-[9px] text-amber-500 hover:text-amber-400 italic text-center mt-1 cursor-pointer w-full hover:bg-amber-500/10 rounded py-0.5 transition-colors"
                                >
                                  {expandedRegionLoot[region.id] ? "Thu gọn ▴" : `+ ${regionLootEntries.length - 5} mục khác ▾`}
                                </button>
                              )}
                            </div>
                          )
                        ) : (
                          regionMonsterLootEntries.length === 0 ? (
                            <p className="text-[10px] text-zinc-600 italic font-mono">Không có quái rơi đồ tại khu vực này</p>
                          ) : (
                            <div className="space-y-1.5">
                              {(expandedRegionLoot[region.id] ? regionMonsterLootEntries : regionMonsterLootEntries.slice(0, 5)).map((e, idx) => (
                                <div key={idx} className="flex items-center justify-between text-[10px] font-mono p-1.5 rounded-lg bg-zinc-950/30 border border-zinc-900/40">
                                  <div className="flex items-center gap-1.5 min-w-0">
                                    <button
                                      type="button"
                                      onClick={() => e.isGu ? locateGu(e.id) : locateItem(e.id)}
                                      className="text-[var(--text-primary)] hover:text-red-400 transition-colors truncate max-w-[110px] text-left cursor-pointer"
                                      title={`Xem: ${e.name}`}
                                    >
                                      {e.name}
                                      {(e.minQty > 1 || e.maxQty > 1) ? (
                                        <span className="text-[9px] text-zinc-500 ml-1">
                                          (x{e.minQty === e.maxQty ? e.minQty : `${e.minQty}-${e.maxQty}`})
                                        </span>
                                      ) : null}
                                    </button>
                                    <button
                                      type="button"
                                      onClick={() => locateMonster(e.monsterId)}
                                      className="text-[8px] px-1 py-0.2 rounded bg-red-950/30 text-red-400 border border-red-500/20 truncate max-w-[80px] hover:border-red-500/40 cursor-pointer"
                                      title={`Quái: ${e.monsterName}`}
                                    >
                                      {e.monsterName}
                                    </button>
                                  </div>
                                  <span className="text-zinc-500 shrink-0">{(e.chance * 100).toFixed(1)}%</span>
                                </div>
                              ))}
                              {regionMonsterLootEntries.length > 5 && (
                                <button
                                  type="button"
                                  onClick={() => toggleRegionLootExpand(region.id)}
                                  className="text-[9px] text-red-500 hover:text-red-400 italic text-center mt-1 cursor-pointer w-full hover:bg-red-500/10 rounded py-0.5 transition-colors"
                                >
                                  {expandedRegionLoot[region.id] ? "Thu gọn ▴" : `+ ${regionMonsterLootEntries.length - 5} mục khác ▾`}
                                </button>
                              )}
                            </div>
                          )
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
              <WikiPagination
                currentPage={paginationInfo.currentPage}
                totalPages={paginationInfo.totalPages}
                totalItems={paginationInfo.totalItems}
                pageSize={pageSize}
                startIndex={paginationInfo.startIndex}
                endIndex={paginationInfo.endIndex}
                onPageChange={setCurrentPage}
                onPageSizeChange={setPageSize}
                pageSizeOptions={TAB_PAGE_SIZE_OPTIONS["regions"]}
                itemLabel="Khu vực"
                scrollTargetId="wiki-results-anchor"
              />
            </>
          )}
        </div>
        )}
      </div>
    </div>
  );
}
