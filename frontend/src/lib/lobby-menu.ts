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

export interface FeatureUnlockRule {
  minRank: number;
  chapter: number;
  lockLabel: string;
  desc: string;
}

export const FEATURE_UNLOCK_RULES: Record<string, FeatureUnlockRule> = {
  refine: { minRank: 2, chapter: 2, lockLabel: "Mở ở Chặng 2 (Nhị Chuyển)", desc: "Hợp thành Cổ trùng mới từ công thức và nguyên liệu." },
  pets: { minRank: 2, chapter: 2, lockLabel: "Mở ở Chặng 2 (Nhị Chuyển)", desc: "Thuần hóa và xuất chiến cùng dị thú trợ thủ." },
  "infinite-tower": { minRank: 2, chapter: 2, lockLabel: "Mở ở Chặng 2 (Nhị Chuyển)", desc: "Thử thách tháp chiến thuật leo tầng nhận thưởng." },
  utilities: { minRank: 2, chapter: 2, lockLabel: "Mở ở Chặng 2 (Nhị Chuyển)", desc: "Phong ấn và quản lý Cổ trùng đặc biệt." },
  gacha: { minRank: 2, chapter: 2, lockLabel: "Mở ở Chặng 2 (Nhị Chuyển)", desc: "Vòng quay cơ duyên và giải trí." },
  "premium-store": { minRank: 2, chapter: 2, lockLabel: "Mở ở Chặng 2 (Nhị Chuyển)", desc: "Cát Bảo Các — cửa hàng vật phẩm cao cấp." },
  sect: { minRank: 3, chapter: 3, lockLabel: "Mở ở Chặng 3 (Tam Chuyển)", desc: "Gia nhập hoặc lập Tông Môn, làm nhiệm vụ tông và tranh đoạt lãnh địa." },
  arena: { minRank: 3, chapter: 3, lockLabel: "Mở ở Chặng 3 (Tam Chuyển)", desc: "Giao đấu PvP xếp hạng cùng các Cổ Sư khác." },
  "martial-arena": { minRank: 3, chapter: 3, lockLabel: "Mở ở Chặng 3 (Tam Chuyển)", desc: "Đại Hội Võ Thuật định kỳ toàn Cổ Giới." },
  "dream-realm": { minRank: 3, chapter: 3, lockLabel: "Mở ở Chặng 3 (Tam Chuyển)", desc: "Thám hiểm Mộng Cảnh để tăng cảnh giới lưu phái." },
  "spatial-realms": { minRank: 3, chapter: 3, lockLabel: "Mở ở Chặng 3 (Tam Chuyển)", desc: "Khám phá Bí Cảnh & Truyền Thừa cổ xưa." },
  divination: { minRank: 3, chapter: 3, lockLabel: "Mở ở Chặng 3 (Tam Chuyển)", desc: "Thiên Cơ Các — bói toán và suy diễn cơ duyên." },
  auction: { minRank: 3, chapter: 3, lockLabel: "Mở ở Chặng 3 (Tam Chuyển)", desc: "Bảo Hoàng Thiên — chợ và đấu giá toàn cõi." },
  "death-arena": { minRank: 4, chapter: 4, lockLabel: "Mở ở Chặng 4 (Tứ Chuyển)", desc: "Đấu Trường Sinh Tử khốc liệt dành cho cao thủ." },
  aperture: { minRank: 6, chapter: 5, lockLabel: "Mở ở Chặng 5 (Lục Chuyển)", desc: "Thế giới riêng của Cổ Tiên (Rank 6+): khai thác mỏ, Đạo Ngân và độ kiếp." },
};

export function getFeatureLockInfo(
  featureId: string,
  playerRank: number,
  unlockedFeatures: string[] = []
): { locked: boolean; rule?: FeatureUnlockRule } {
  const rule = FEATURE_UNLOCK_RULES[featureId];
  if (!rule) return { locked: false };
  const unlocked = playerRank >= rule.minRank || unlockedFeatures.includes(featureId);
  return { locked: !unlocked, rule };
}
