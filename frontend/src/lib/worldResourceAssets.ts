export const WORLD_RESOURCE_ASSETS = {
  herb: "/assets/world/spirit-herb.webp",
  crystal: "/assets/world/items/crystal.svg",
  food: "/assets/world/items/food.svg",
  gourd: "/assets/world/items/gourd.svg",
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
