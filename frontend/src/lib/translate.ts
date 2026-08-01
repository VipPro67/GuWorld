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
  Fire: "Viêm", Water: "Thủy", Sword: "Kiếm", Time: "Trụ", Space: "Vũ",
  Qi: "Khí", Blood: "Huyết", Strength: "Lực", Wisdom: "Trí",
  Transformation: "Biến Hóa", Soul: "Hồn", Illusion: "Huyễn",
  Refinement: "Luyện", Rule: "Luật", Wood: "Mộc", Earth: "Thổ",
  Wind: "Phong", Light: "Quang", Dark: "Ám", Ice: "Băng", Lightning: "Lôi",
  Poison: "Độc", Sound: "Âm", Star: "Tinh", Cloud: "Vân", Dream: "Mộng",
  Theft: "Trộm", Information: "Tín", Bone: "Cốt", Phantom: "Hư", Luck: "Vận",
  Food: "Thực", Painting: "Họa", Heaven: "Thiên", Human: "Nhân", Weapon: "Binh",
  Formation: "Trận", Yin: "Âm", Yang: "Dương", Metal: "Kim", Moon: "Nguyệt", Sun: "Nhật",
  Enslavement: "Nô"
};

export function translatePath(path: string | undefined): string {
  if (!path) return "?";
  
  // Normalize by stripping " Path" suffix if it exists
  const base = path.endsWith(" Path") ? path.slice(0, -5) : path;
  
  if (PATH_MAP[base]) {
    return PATH_MAP[base] + " Đạo";
  }
  
  return PATH_MAP[path] || path;
}
