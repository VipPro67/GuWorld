import { getRankScale } from "./balance";
import type { Region } from "@/store/useGameStore";
import type {
  RegionDetailsData,
  RegionMonsterDetail,
  RegionDropDetail,
  NamedResource,
} from "@/components/WorldAtlasModal";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let cachedWikiData: any = null;
function getCache() {
  if (cachedWikiData) return cachedWikiData;
  try {
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    cachedWikiData = require("@/data/wiki-cache.json");
    return cachedWikiData;
  } catch {
    return null;
  }
}

// Maps for O(1) lookups
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let monsterMap: Map<string, any> | null = null;
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let lootTableMap: Map<string, any> | null = null;
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let guMap: Map<string, any> | null = null;
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let itemMap: Map<string, any> | null = null;
// eslint-disable-next-line @typescript-eslint/no-explicit-any
let regionMap: Map<string, any> | null = null;

function ensureMaps() {
  if (monsterMap) return;
  const d = getCache();
  if (!d) return;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  monsterMap = new Map((d.monsters || []).map((m: any) => [m.id, m]));
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  lootTableMap = new Map((d.lootTables || []).map((lt: any) => [lt.id, lt]));
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  guMap = new Map((d.gus || []).map((g: any) => [g.id, g]));
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  itemMap = new Map((d.items || []).map((i: any) => [i.id, i]));
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  regionMap = new Map((d.regions || []).map((r: any) => [r.id, r]));
}

export function getStaticRegions(): Region[] {
  const d = getCache();
  return (d?.regions || []) as Region[];
}

function scaleMonsterStat(base: number, rank: number, type: string): number {
  const rankMultiplier = getRankScale(rank);
  let typeMultiplier = 1.0;
  const t = (type || "").toUpperCase();
  if (t === "ELITE") typeMultiplier = 1.5;
  else if (t === "BOSS") typeMultiplier = 3.0;
  else if (t.includes("WORLD") && t.includes("BOSS")) typeMultiplier = 25.0;
  return Math.round(base * rankMultiplier * typeMultiplier);
}

export function getRegionDetailsFromCache(
  regionId: string,
  propRegions?: Region[]
): RegionDetailsData | null {
  const d = getCache();
  if (!d) return null;
  ensureMaps();

  const reg =
    (propRegions && propRegions.find((r) => r.id === regionId)) ||
    regionMap?.get(regionId);
  if (!reg) return null;

  // 1. Monsters from monsterSpawns
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const spawns = (d.monsterSpawns || []).filter((s: any) => s.region_id === regionId);
  const monsterDetailMap = new Map<string, RegionMonsterDetail>();

  for (const s of spawns) {
    const m = monsterMap?.get(s.monster_id);
    const mID = s.monster_id;
    const name = m?.name || mID;
    const mType = s.is_boss_spawn ? "Boss" : m?.type || "Normal";
    const isBoss = s.is_boss_spawn || mType === "Boss";
    const minR = s.min_rank || 1;
    const maxR = s.max_rank || minR;
    const baseHP = m?.max_hp || 100;
    const baseAtk = m?.base_attack || 10;
    const baseDef = m?.base_defense || 5;

    if (!monsterDetailMap.has(mID)) {
      monsterDetailMap.set(mID, {
        id: mID,
        name,
        type: mType,
        min_rank: minR,
        max_rank: maxR,
        is_boss: isBoss,
        is_capturable: !!m?.is_capturable,
        max_hp: scaleMonsterStat(baseHP, minR, mType),
        base_attack: scaleMonsterStat(baseAtk, minR, mType),
        base_defense: scaleMonsterStat(baseDef, minR, mType),
        max_hp_max: scaleMonsterStat(baseHP, maxR, mType),
        attack_max: scaleMonsterStat(baseAtk, maxR, mType),
        defense_max: scaleMonsterStat(baseDef, maxR, mType),
        description: m?.description || "",
      });
    } else {
      const existing = monsterDetailMap.get(mID)!;
      if (minR < existing.min_rank) {
        existing.min_rank = minR;
        existing.max_hp = scaleMonsterStat(baseHP, minR, existing.type);
        existing.base_attack = scaleMonsterStat(baseAtk, minR, existing.type);
        existing.base_defense = scaleMonsterStat(baseDef, minR, existing.type);
      }
      if (maxR > existing.max_rank) {
        existing.max_rank = maxR;
        existing.max_hp_max = scaleMonsterStat(baseHP, maxR, existing.type);
        existing.attack_max = scaleMonsterStat(baseAtk, maxR, existing.type);
        existing.defense_max = scaleMonsterStat(baseDef, maxR, existing.type);
      }
      if (isBoss) {
        existing.is_boss = true;
        existing.type = "Boss";
      }
    }
  }

  const monstersList = Array.from(monsterDetailMap.values());

  // 2. Drops
  const dropsMap = new Map<string, RegionDropDetail>();

  // A) From monsters in this region
  for (const mDetail of monstersList) {
    const m = monsterMap?.get(mDetail.id);
    if (!m?.loot_table_id) continue;
    const lt = lootTableMap?.get(m.loot_table_id);
    if (!lt?.entries) continue;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    for (const e of lt.entries as any[]) {
      if (e.item_id) {
        const key = `item:${e.item_id}`;
        const item = itemMap?.get(e.item_id);
        const name = item?.name || e.item_id;
        const existing = dropsMap.get(key);
        if (existing) {
          if (e.chance > existing.chance) existing.chance = e.chance;
          if (e.max_qty > existing.max_qty) existing.max_qty = e.max_qty;
          if (!existing.sources?.includes(mDetail.name)) {
            existing.sources = [...(existing.sources || []), mDetail.name];
          }
        } else {
          dropsMap.set(key, {
            id: e.item_id,
            name,
            kind: "item",
            rank: e.rank || item?.rank,
            chance: e.chance,
            min_qty: e.min_qty || 1,
            max_qty: e.max_qty || 1,
            sources: [mDetail.name],
          });
        }
      }
      if (e.gu_id) {
        const key = `gu:${e.gu_id}`;
        const gu = guMap?.get(e.gu_id);
        const name = gu?.name || e.gu_id;
        const existing = dropsMap.get(key);
        if (existing) {
          if (e.chance > existing.chance) existing.chance = e.chance;
          if (!existing.sources?.includes(mDetail.name)) {
            existing.sources = [...(existing.sources || []), mDetail.name];
          }
        } else {
          dropsMap.set(key, {
            id: e.gu_id,
            name,
            kind: "gu",
            rank: e.rank || gu?.rank,
            chance: e.chance,
            min_qty: e.min_qty || 1,
            max_qty: e.max_qty || 1,
            sources: [mDetail.name],
          });
        }
      }
    }
  }

  // B) From Region Loot Tables (including parent region)
  const regionRules = (d.regionLoot || []).filter(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (rl: any) =>
      rl.region_id === regionId || (reg.parent_id && rl.region_id === reg.parent_id)
  );

  for (const rl of regionRules) {
    const lt = lootTableMap?.get(rl.loot_table_id);
    if (!lt?.entries) continue;
    const sourceLabel = `Khu vực (${rl.monster_type})`;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    for (const e of lt.entries as any[]) {
      if (e.item_id) {
        const key = `item:${e.item_id}`;
        const item = itemMap?.get(e.item_id);
        const name = item?.name || e.item_id;
        const existing = dropsMap.get(key);
        if (existing) {
          if (e.chance > existing.chance) existing.chance = e.chance;
          if (e.max_qty > existing.max_qty) existing.max_qty = e.max_qty;
          if (!existing.sources?.includes(sourceLabel)) {
            existing.sources = [...(existing.sources || []), sourceLabel];
          }
        } else {
          dropsMap.set(key, {
            id: e.item_id,
            name,
            kind: "item",
            rank: e.rank || item?.rank,
            chance: e.chance,
            min_qty: e.min_qty || 1,
            max_qty: e.max_qty || 1,
            sources: [sourceLabel],
          });
        }
      }
      if (e.gu_id) {
        const key = `gu:${e.gu_id}`;
        const gu = guMap?.get(e.gu_id);
        const name = gu?.name || e.gu_id;
        const existing = dropsMap.get(key);
        if (existing) {
          if (e.chance > existing.chance) existing.chance = e.chance;
          if (!existing.sources?.includes(sourceLabel)) {
            existing.sources = [...(existing.sources || []), sourceLabel];
          }
        } else {
          dropsMap.set(key, {
            id: e.gu_id,
            name,
            kind: "gu",
            rank: e.rank || gu?.rank,
            chance: e.chance,
            min_qty: e.min_qty || 1,
            max_qty: e.max_qty || 1,
            sources: [sourceLabel],
          });
        }
      }
    }
  }

  // 3. Wild Gus & Resources
  const wildGus: NamedResource[] = (reg.pool_weights?.wild_gus || []).map((gId: string) => ({
    id: gId,
    name: guMap?.get(gId)?.name || gId,
  }));

  const resources: NamedResource[] = (reg.pool_weights?.resources || []).map(
    (rId: string) => ({
      id: rId,
      name: itemMap?.get(rId)?.name || rId,
    })
  );

  return {
    region_id: reg.id,
    name: reg.name,
    zone_type: reg.zone_type,
    min_rank: reg.min_rank,
    max_rank: reg.max_rank,
    description: reg.description,
    monsters: monstersList,
    drops: Array.from(dropsMap.values()),
    wild_gus: wildGus,
    resources,
    has_boss: reg.zone_type === "BOSS_ZONE" || monstersList.some((m) => m.is_boss),
    monster_count: monstersList.length,
  };
}
