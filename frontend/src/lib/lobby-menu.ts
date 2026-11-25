export interface LobbyMenuGroupSpec {
  id: string;
  label: string;
  desc: string;
  children: readonly string[];
}

export const CORE_LOBBY_FEATURE_IDS = [
  "character",
  "inventory",
  "refine",
  "explore",
  "aperture",
  "pets",
  "sect",
] as const;

export const LOBBY_MENU_GROUPS: readonly LobbyMenuGroupSpec[] = [
  {
    id: "combat-hub",
    label: "Đấu Trường & Thí Luyện",
    desc: "Phòng tập, võ đài, tháp chiến thuật và đại hội",
    children: ["practice-room", "arena", "infinite-tower", "death-arena", "martial-arena"],
  },
  {
    id: "commerce-hub",
    label: "Thương Hội",
    desc: "Cửa hàng, Cát Bảo Các và chợ đấu giá",
    children: ["shop", "premium-store", "auction"],
  },
  {
    id: "adventure-hub",
    label: "Bí Cảnh & Cơ Duyên",
    desc: "Mộng cảnh, bí cảnh truyền thừa, thiên cơ và giải trí",
    children: ["dream-realm", "spatial-realms", "divination", "gacha"],
  },
  {
    id: "system-hub",
    label: "Tiện Ích & Ký Sự",
    desc: "Nhiệm vụ, danh bảng, phong ấn cổ và nhật ký",
    children: ["achievements", "leaderboard", "utilities", "combat-history", "audit-logs", "feedback"],
  },
] as const;

export const LOBBY_FEATURE_IDS: readonly string[] = [
  ...CORE_LOBBY_FEATURE_IDS,
  ...LOBBY_MENU_GROUPS.flatMap((group) => group.children),
];

export const DEFAULT_LOBBY_GRID_ITEMS: readonly string[] = [
  ...CORE_LOBBY_FEATURE_IDS,
  ...LOBBY_MENU_GROUPS.map((group) => group.id),
];

export const VALID_LOBBY_GRID_IDS = new Set<string>([
  ...LOBBY_FEATURE_IDS,
  ...LOBBY_MENU_GROUPS.map((group) => group.id),
]);

export const LOBBY_GRID_LIMIT_OPTIONS = [6, 8, 12, 0] as const;
export type LobbyGridLimit = (typeof LOBBY_GRID_LIMIT_OPTIONS)[number];
export const DEFAULT_LOBBY_GRID_LIMIT: LobbyGridLimit = 12;
export const LOBBY_MENU_VERSION = 2;

export function isLobbyGroupId(id: string): boolean {
  return LOBBY_MENU_GROUPS.some((group) => group.id === id);
}

export const LOBBY_ENTRY_LABELS: Record<string, string> = {
  "combat-hub": "Đấu Trường & Thí Luyện",
  "commerce-hub": "Thương Hội",
  "adventure-hub": "Bí Cảnh & Cơ Duyên",
  "system-hub": "Tiện Ích & Ký Sự",
  character: "Nhân Vật",
  inventory: "Túi Đồ",
  refine: "Luyện Cổ",
  explore: "Khám Phá Dã Ngoại",
  aperture: "Tiên Khiếu",
  pets: "Nô Thú",
  sect: "Tông Môn",
  "practice-room": "Phòng Tập",
  arena: "Võ Đài",
  "infinite-tower": "Tháp Chiến Thuật",
  "death-arena": "Đấu Trường Sinh Tử",
  "martial-arena": "Đại Hội Võ Thuật",
  shop: "Cửa Hàng",
  "premium-store": "Cát Bảo Các",
  auction: "Chợ, Đấu Giá",
  "dream-realm": "Mộng Cảnh",
  "spatial-realms": "Bí Cảnh & Truyền Thừa",
  divination: "Thiên Cơ Các",
  gacha: "Giải Trí",
  achievements: "Nhiệm Vụ & Thành Tựu",
  leaderboard: "Bảng Xếp Hạng",
  utilities: "Phong Ấn Cổ Trùng",
  "combat-history": "Lịch Sử Chiến Đấu",
  "audit-logs": "Ký Sự Tu Hành",
  feedback: "Thỉnh Thiên Đạo",
};
