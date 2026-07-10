export interface GuEffects {
  active: Record<string, number>;
  support: Record<string, number>;
  passive: Record<string, number>;
  house: Record<string, number>;
}

export interface GuDictionaryItem {
  id: string;
  name: string;
  rank: number;
  path: string[];
  gu_type: string;
  base_mana_cost: number;
  effects: GuEffects;
  food_types: string[];
  hunger_rate: number;
  max_hunger: number;
  description: string;
  rarity: string;
  is_immortal: boolean;
  primary_role: string;
  combat_roles: string[];
  combat_tags: string[];
  support_tags: string[];
  target_type?: string;
}

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

export async function fetchAllGus(): Promise<GuDictionaryItem[]> {
  try {
    const res = await fetch(`${BACKEND_URL}/api/dictionary`, {
      cache: "force-cache",
      next: { revalidate: 3600 },
    });
    if (!res.ok) return [];
    const data = await res.json();
    const gu: Record<string, GuDictionaryItem> = data.gu || {};
    return Object.values(gu);
  } catch {
    return [];
  }
}

export async function fetchGuById(id: string): Promise<GuDictionaryItem | null> {
  const gus = await fetchAllGus();
  return gus.find((g) => g.id === id) || null;
}
