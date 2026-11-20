import manifest from "../data/game-icon-manifest.json";

export type GameIconGroup = "gu" | "item" | "resourcePoint";

interface GameIconRecord {
  runtime: string;
  master: string;
  source: string;
  prompt: string;
  artwork_id?: string;
  archetype?: string;
}

const records = manifest as {
  schema_version: number;
} & Record<GameIconGroup, Record<string, GameIconRecord>>;

export function getManifestIconUrl(group: GameIconGroup, id?: string | null): string | undefined {
  if (!id) return undefined;
  const normalized = id.trim();
  const rankSharedId = group === "item" ? normalized.replace(/_rank\d+$/i, "") : normalized;
  const record = records[group][normalized] ?? records[group][rankSharedId];
  const path = record?.runtime;
  return path?.startsWith("/assets/game-icons/") ? path : undefined;
}

export function isValidGameImageUrl(value?: string | null): value is string {
  const url = value?.trim();
  return Boolean(url && (/^https?:\/\//i.test(url) || (url.startsWith("/") && !url.startsWith("//"))));
}
