import { getManifestIconUrl, isValidGameImageUrl } from "./game-icon-manifest.ts";

export const WORLD_RESOURCE_ASSETS = {
  herb: "/assets/world/spirit-herb.webp",
  crystal: "/assets/items/icons/crystal.webp",
  food: "/assets/items/icons/food.webp",
  gourd: "/assets/items/icons/gourd.webp",
} as const;

export function getWorldResourceGroup(itemId: string, itemType = ""): keyof typeof WORLD_RESOURCE_ASSETS {
  const id = itemId.toLowerCase();
  const type = itemType.toLowerCase();
  if (/herb|flower|orchid|petal|ginseng|root|leaf/.test(id)) return "herb";
  if (/wine|liquor|potion|elixir/.test(id) || type === "consumable") return "gourd";
  if (/meat|food/.test(id) || type === "food") return "food";
  if (/stone|ore|crystal|jade|gem/.test(id) || type === "material") return "crystal";
  return "herb";
}

export function getWorldResourceIconUrl(itemId: string, itemType = "", imageUrl?: string): string {
  if (isValidGameImageUrl(imageUrl)) return imageUrl.trim();
  return getManifestIconUrl("item", itemId) ?? WORLD_RESOURCE_ASSETS[getWorldResourceGroup(itemId, itemType)];
}

export function getResourcePointIconUrl(pointType?: string): string | undefined {
  return getManifestIconUrl("resourcePoint", pointType);
}
