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
  Blade: "Đao",
  Blood: "Huyết",
  Bone: "Cốt",
  Cloud: "Vân",
  Concealment: "Ẩn",
  Creation: "Tạo Hóa",
  Dark: "Hắc Ám",
  Death: "Tử Vong",
  Destruction: "Hủy Diệt",
  Dream: "Mộng",
  Earth: "Thổ",
  Enchantment: "Chú",
  Enslavement: "Nô",
  Fire: "Hỏa",
  Food: "Thực",
  Formation: "Trận",
  Heaven: "Thiên",
  Human: "Nhân",
  Ice: "Băng",
  Illusion: "Ảo Giác",
  Information: "Tín",
  Investigation: "Trinh Sát",
  Life: "Sinh",
  Light: "Quang",
  Lightning: "Lôi",
  Luck: "Khí Vận",
  Metal: "Kim",
  Moon: "Nguyệt",
  Movement: "Di",
  Painting: "Họa",
  Phantom: "Hư",
  Poison: "Độc",
  Qi: "Khí",
  Refinement: "Luyện",
  Rule: "Quy Tắc",
  Shadow: "Ảnh",
  Smoke: "Yên Vụ",
  Soul: "Hồn",
  Sound: "Âm Thanh",
  Space: "Không Gian",
  Star: "Tinh",
  Strength: "Lực",
  Sun: "Nhật",
  Sword: "Kiếm",
  Theft: "Trộm",
  Theme: "Chủ Đề",
  Time: "Thời Gian",
  Transformation: "Biến Hóa",
  Water: "Thủy",
  Weapon: "Binh",
  Wind: "Phong",
  Wisdom: "Trí",
  Wood: "Mộc",
  Yang: "Dương",
  Yin: "Âm",
};

const pathLabels: Record<string, string> = {
  "Blade Path": "Đao Đạo",
  "Blood Path": "Huyết Đạo",
  "Bone Path": "Cốt Đạo",
  "Cloud Path": "Vân Đạo",
  "Concealment Path": "Ẩn Nặc Đạo",
  "Creation Path": "Tạo Hóa Đạo",
  "Dark Path": "Hắc Ám Đạo",
  "Death Path": "Tử Vong Đạo",
  "Destruction Path": "Hủy Diệt Đạo",
  "Dream Path": "Mộng Đạo",
  "Earth Path": "Thổ Đạo",
  "Enchantment Path": "Phù Chú Đạo",
  "Enslavement Path": "Nô Dịch Đạo",
  "Fire Path": "Hỏa Đạo",
  "Food Path": "Thực Đạo",
  "Formation Path": "Trận Pháp Đạo",
  "Heaven Path": "Thiên Đạo",
  "Human Path": "Nhân Đạo",
  "Ice Path": "Hàn Băng Đạo",
  "Illusion Path": "Ảo Giác Đạo",
  "Information Path": "Tín Đạo",
  "Investigation Path": "Trinh Sát Đạo",
  "Life Path": "Sinh Đạo",
  "Light Path": "Quang Minh Đạo",
  "Lightning Path": "Lôi Đạo",
  "Luck Path": "Khí Vận Đạo",
  "Metal Path": "Kim Đạo",
  "Moon Path": "Nguyệt Đạo",
  "Movement Path": "Di Động Đạo",
  "Poison Path": "Độc Đạo",
  "Qi Path": "Khí Đạo",
  "Refinement Path": "Tinh Luyện Đạo",
  "Rule Path": "Quy Tắc Đạo",
  "Shadow Path": "Ảnh Đạo",
  "Smoke Path": "Yên Vụ Đạo",
  "Soul Path": "Hồn Đạo",
  "Sound Path": "Âm Thanh Đạo",
  "Space Path": "Không Gian Đạo",
  "Strength Path": "Lực Đạo",
  "Sun Path": "Nhật Đạo",
  "Sword Path": "Kiếm Đạo",
  "Theft Path": "Trộm Đạo",
  "Theme Path": "Chủ Đề Đạo",
  "Time Path": "Thời Gian Đạo",
  "Transformation Path": "Biến Hóa Đạo",
  "Water Path": "Thủy Đạo",
  "Wind Path": "Phong Đạo",
  "Wisdom Path": "Trí Đạo",
  "Wood Path": "Mộc Đạo",
};


export function translatePath(path: string | undefined): string {
  if (!path) return "?";
  
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

// Path mastery milestones (mirror backend gamefx.PathMasteryLevel).
export const PATH_MASTERY_MILESTONES = [100, 500, 2500, 10000];

// masteryLevelForPoints maps accumulated path mastery points to the attained
// realm level (0-4). Mirrors gamefx.PathMasteryLevel.
export function masteryLevelForPoints(points: number): number {
  if (points >= PATH_MASTERY_MILESTONES[3]) return 4;
  if (points >= PATH_MASTERY_MILESTONES[2]) return 3;
  if (points >= PATH_MASTERY_MILESTONES[1]) return 2;
  if (points >= PATH_MASTERY_MILESTONES[0]) return 1;
  return 0;
}

export const MASTERY_LEVEL_KEYS = ["Ordinary", "Master", "Grandmaster", "Great Grandmaster", "Supreme Grandmaster"];

// masteryKeyForPoints returns the English level key for path mastery points.
export function masteryKeyForPoints(points: number): string {
  return MASTERY_LEVEL_KEYS[masteryLevelForPoints(points)];
}

// masteryLevelRequiredForRank returns the minimum path-mastery level required
// to advance to a rank (Rank 7 = Tông Sư, Rank 8 = Đại Tông Sư, Rank 9 = Vô
// Thượng Đại Tông Sư). Returns 0 when no requirement. Mirrors gamefx.
export function masteryLevelRequiredForRank(rank: number): number {
  if (rank === 7) return 2;
  if (rank === 8) return 3;
  if (rank === 9) return 4;
  return 0;
}
