import type { GuDictionaryItem } from "./gu-dict";

// A Gu id is NOT unique: the same id ships as several rank variants that often
// have different names (e.g. GU_GOLDEN_SUPPORT_I is "Kim Hỗ Trợ Tiên Cổ" at rank
// 6-8 but "Cửu Chuyển Kim Hỗ Trợ Tiên Cổ" at rank 9). Resolving a name from the
// id alone collapses them, so lookups must key on (id, rank).
export interface GuIndex {
  /** Highest-rank variant per id — use only when the rank is genuinely unknown. */
  byId: Map<string, GuDictionaryItem>;
  byRank: Map<string, Map<number, GuDictionaryItem>>;
}

export function indexGus(gus: GuDictionaryItem[]): GuIndex {
  const byId = new Map<string, GuDictionaryItem>();
  const byRank = new Map<string, Map<number, GuDictionaryItem>>();
  for (const g of gus) {
    let ranks = byRank.get(g.id);
    if (!ranks) byRank.set(g.id, (ranks = new Map()));
    ranks.set(g.rank, g);
    const best = byId.get(g.id);
    if (!best || g.rank > best.rank) byId.set(g.id, g);
  }
  return { byId, byRank };
}

/** Resolve a Gu by id + rank, falling back to the highest-rank variant when the rank is unknown or absent. */
export function guAt(index: GuIndex, id: string, rank?: number | null): GuDictionaryItem | undefined {
  if (rank != null) {
    const hit = index.byRank.get(id)?.get(rank);
    if (hit) return hit;
  }
  return index.byId.get(id);
}
