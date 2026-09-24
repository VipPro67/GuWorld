export const GU_TYPE_MAP: Record<string, string> = {
  Active: "Chủ động",
  Buff: "Cường hoá",
  Consumable: "Tiêu hao",
  Control: "Khống chế",
  Defense: "Phòng thủ",
  Healing: "Hồi phục",
  Investigation: "Trinh sát",
  Movement: "Di chuyển",
  Passive: "Bị động",
  Refinement: "Luyện cổ",
  Special: "Đặc thù",
  Storage: "Lưu trữ",
  Summon: "Triệu hồi",
  Support: "Hỗ trợ",
  Transformation: "Biến hoá",
  Vital: "Bản mệnh",
};

export const ITEM_TYPE_MAP: Record<string, string> = {
  Material: "Vật liệu",
  Consumable: "Tiêu hao",
  Food: "Thức ăn",
  Currency: "Tiền tệ",
  Recipe: "Cổ phương",
};

export const TARGET_TYPE_MAP: Record<string, string> = {
  SINGLE: "Đơn",
  ALL: "Toàn thể",
};

export const STAT_MAP: Record<string, string> = {
  HP: "Sinh lực",
  MP: "Chân nguyên",
  Mana: "Chân nguyên",
  ATK: "Tấn công",
  DEF: "Phòng thủ",
  SPD: "Tốc độ",
  Rank: "Bậc",
};

export function translateGuType(type: string | undefined): string {
  if (!type) return "?";
  return GU_TYPE_MAP[type] || type;
}

export function translateItemType(type: string | undefined): string {
  if (!type) return "?";
  return ITEM_TYPE_MAP[type] || type;
}

export function translateTargetType(type: string | undefined): string {
  if (!type) return "?";
  return TARGET_TYPE_MAP[type] || type;
}

export function translateStat(stat: string | undefined): string {
  if (!stat) return "?";
  return STAT_MAP[stat] || stat;
}
export const PATH_MAP: Record<string, string> = {
  // Ngũ hành & Tự nhiên
  Metal: "Kim",
  Wood: "Mộc",
  Water: "Thủy",
  Fire: "Hỏa",
  Earth: "Thổ",
  Wind: "Phong",
  Lightning: "Lôi",
  Ice: "Băng Tuyết",
  Cloud: "Vân",
  Light: "Quang",
  Dark: "Ám",
  Star: "Tinh",
  Sound: "Âm",

  // Không - Thời gian & Vũ trụ
  Space: "Vũ",
  Time: "Trụ",
  Heaven: "Thiên",

  // Quy tắc & Kỹ nghệ
  Rule: "Luật",
  Refinement: "Luyện",
  Formation: "Trận",
  Qi: "Khí",

  // Cơ thể, Tinh thần & Xã hội
  Strength: "Lực",
  Soul: "Hồn",
  Wisdom: "Trí",
  Emotion: "Tình",
  Transformation: "Biến Hóa",
  Enslavement: "Nô",
  Information: "Tín",
  Blood: "Huyết",
  Poison: "Độc",
  Bone: "Cốt",
  Human: "Nhân",
  Dream: "Mộng",
  Food: "Thực",
  Luck: "Vận",

  // Vũ khí, Sát phạt & Đặc thù
  Sword: "Kiếm",
  Blade: "Đao",
  Theft: "Thâu",
  Phantom: "Huyễn",
  Painting: "Họa",
  Weapon: "Binh",
  Killing: "Sát",
  Kill: "Sát",
  Death: "Tử",
  Creation: "Tạo Hóa",
  Destruction: "Hủy Diệt",
  Investigation: "Trinh Sát",
  Restriction: "Cấm",
  Pill: "Đan",
};

export const pathLabels: Record<string, string> = {
  // Ngũ hành & Tự nhiên
  "Metal Path": "Kim Đạo",
  "Wood Path": "Mộc Đạo",
  "Water Path": "Thủy Đạo",
  "Fire Path": "Hỏa Đạo",
  "Earth Path": "Thổ Đạo",
  "Wind Path": "Phong Đạo",
  "Lightning Path": "Lôi Đạo",
  "Ice Path": "Băng Tuyết Đạo",
  "Cloud Path": "Vân Đạo",
  "Light Path": "Quang Đạo",
  "Dark Path": "Ám Đạo",
  "Star Path": "Tinh Đạo",
  "Sound Path": "Âm Đạo",

  // Không - Thời gian & Vũ trụ
  "Space Path": "Vũ Đạo",
  "Time Path": "Trụ Đạo",
  "Heaven Path": "Thiên Đạo",

  // Quy tắc & Kỹ nghệ
  "Rule Path": "Luật Đạo",
  "Refinement Path": "Luyện Đạo",
  "Formation Path": "Trận Đạo",
  "Qi Path": "Khí Đạo",
  "Death Path": "Tử Đạo",
  "Creation Path": "Tạo Hóa Đạo",
  "Destruction Path": "Hủy Diệt Đạo",
  "Investigation Path": "Trinh Sát Đạo",

  // Cơ thể, Tinh thần & Xã hội
  "Strength Path": "Lực Đạo",
  "Soul Path": "Hồn Đạo",
  "Wisdom Path": "Trí Đạo",
  "Emotion Path": "Tình Đạo",
  "Transformation Path": "Biến Hóa Đạo",
  "Enslavement Path": "Nô Đạo",
  "Information Path": "Tín Đạo",
  "Blood Path": "Huyết Đạo",
  "Poison Path": "Độc Đạo",
  "Bone Path": "Cốt Đạo",
  "Human Path": "Nhân Đạo",
  "Dream Path": "Mộng Đạo",
  "Food Path": "Thực Đạo",
  "Luck Path": "Vận Đạo",

  // Vũ khí, Sát phạt & Đặc thù
  "Sword Path": "Kiếm Đạo",
  "Blade Path": "Đao Đạo",
  "Theft Path": "Thâu Đạo",
  "Phantom Path": "Huyễn Đạo",
  "Painting Path": "Họa Đạo",
  "Weapon Path": "Binh Đạo",
  "Killing Path": "Sát Đạo",
  "Kill Path": "Sát Đạo",
  "Restriction Path": "Cấm Đạo",
  "Pill Path": "Đan Đạo",
};


export function parsePaths(path: unknown): string[] {
  if (!path) return [];
  if (Array.isArray(path)) {
    return path.flatMap((p) => parsePaths(p));
  }
  if (typeof path === "string") {
    const trimmed = path.trim();
    if (!trimmed) return [];
    if (trimmed.startsWith("[") && trimmed.endsWith("]")) {
      try {
        const parsed = JSON.parse(trimmed);
        if (Array.isArray(parsed)) {
          return parsed.flatMap((p) => parsePaths(p));
        }
      } catch {
        // Fallback to single string if JSON parsing fails
      }
    }
    return [trimmed];
  }
  return [];
}

export function translatePath(path: string | undefined): string {
  if (!path) return "?";

  if (typeof path === "string" && path.trim().startsWith("[") && path.trim().endsWith("]")) {
    const parsed = parsePaths(path);
    if (parsed.length > 0) {
      return parsed.map(translatePath).join(", ");
    }
  }

  // Normalize by stripping " Path" suffix if it exists
  const base = path.endsWith(" Path") ? path.slice(0, -5) : path;

  // Capitalize first letter to match keys in PATH_MAP
  const capitalizedBase = base.charAt(0).toUpperCase() + base.slice(1).toLowerCase();

  if (PATH_MAP[capitalizedBase]) {
    return PATH_MAP[capitalizedBase] + " Đạo";
  }

  return PATH_MAP[path] || path;
}

export const MASTERY_MAP: Record<string, string> = {
  Ordinary: "Phổ Thông",
  Master: "Đại Sư",
  Grandmaster: "Tông Sư",
  "Great Grandmaster": "Đại Tông Sư",
  "Supreme Grandmaster": "Vô Thượng Tông Sư",
};

export function translateMastery(mastery: string | undefined): string {
  if (!mastery) return "?";
  return MASTERY_MAP[mastery] || mastery;
}

export {
  PATH_MASTERY_MILESTONES,
  PATH_MASTERY_LEVEL_KEYS,
  masteryLevelForPoints,
  masteryKeyForPoints,
  masteryLevelRequiredForRank,
} from "./balance";
export type { PathMasteryLevelKey } from "./balance";
export const MASTERY_LEVEL_KEYS = [
  "Ordinary",
  "Master",
  "Grandmaster",
  "Great Grandmaster",
  "Supreme Grandmaster",
];
