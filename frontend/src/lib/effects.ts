// Helper để format effect từ dictionary (thể chất, chủng tộc...)
// sang mô tả tiếng Việt hiển thị cho người chơi.

const PATH_NAMES: Record<string, string> = {
  blade: "Đao Đạo",
  blood: "Huyết Đạo",
  bone: "Cốt Đạo",
  cloud: "Vân Đạo",
  concealment: "Ẩn Nặc Đạo",
  creation: "Tạo Hóa Đạo",
  dark: "Hắc Ám Đạo",
  death: "Tử Vong Đạo",
  destruction: "Hủy Diệt Đạo",
  dream: "Mộng Đạo",
  earth: "Thổ Đạo",
  enchantment: "Phù Chú Đạo",
  enslavement: "Nô Dịch Đạo",
  fire: "Hỏa Đạo",
  food: "Thực Đạo",
  formation: "Trận Pháp Đạo",
  heaven: "Thiên Đạo",
  human: "Nhân Đạo",
  ice: "Hàn Băng Đạo",
  illusion: "Ảo Giác Đạo",
  information: "Tín Đạo",
  investigation: "Trinh Sát Đạo",
  life: "Sinh Đạo",
  light: "Quang Minh Đạo",
  lightning: "Lôi Đạo",
  luck: "Khí Vận Đạo",
  metal: "Kim Đạo",
  moon: "Nguyệt Đạo",
  movement: "Di Động Đạo",
  painting: "Họa Đạo",
  phantom: "Huyễn Đạo",
  poison: "Độc Đạo",
  qi: "Khí Đạo",
  refinement: "Tinh Luyện Đạo",
  rule: "Quy Tắc Đạo",
  shadow: "Ảnh Đạo",
  smoke: "Yên Vụ Đạo",
  soul: "Hồn Đạo",
  sound: "Âm Thanh Đạo",
  space: "Không Gian Đạo",
  strength: "Lực Đạo",
  sun: "Nhật Đạo",
  sword: "Kiếm Đạo",
  theft: "Trộm Đạo",
  theme: "Chủ Đề Đạo",
  time: "Thời Gian Đạo",
  transformation: "Biến Hóa Đạo",
  water: "Thủy Đạo",
  wind: "Phong Đạo",
  wisdom: "Trí Đạo",
  wood: "Mộc Đạo",
};

const STAT_NAMES: Record<string, string> = {
  attack: "Tấn công",
  defense: "Phòng thủ",
  speed: "Tốc độ",
  healing: "Hồi phục",
  movement: "Di chuyển",
  investigation: "Trinh sát",
  storage: "Kho chứa",
};

const pct = (v: number): string => {
  const p = Math.round(v * 100);
  return `${p > 0 ? "+" : ""}${p}%`;
};

const pathName = (p: string): string => PATH_NAMES[p] || p.charAt(0).toUpperCase() + p.slice(1) + " Đạo";

export interface GameEffects {
  stat_mods?: Record<string, number>;
  dao_mark?: Record<string, number>;
  path_damage?: Record<string, number>;
  path_resistance?: Record<string, number>;
  mana_regen?: number;
  max_mana?: number;
  cultivation_efficiency?: number;
  dao_mark_gain?: number;
  luck?: number;
  refine?: number;
  healing_received?: number;
  mental_resistance?: number;
  soul_defense?: number;
  aperture_stress_bonus?: number;
}

// Định nghĩa một mục dictionary (thể chất / chủng tộc) trả về từ API.
export interface EffectDictionaryEntry {
  key: string;
  name: string;
  effects?: GameEffects;
}

// Trả về mảng các mô tả hiệu ứng bằng tiếng Việt.
export function formatGameEffects(effects?: GameEffects): string[] {
  if (!effects) return [];
  const out: string[] = [];

  if (effects.stat_mods) {
    for (const [key, v] of Object.entries(effects.stat_mods)) {
      const label = STAT_NAMES[key] || key;
      out.push(`${pct(v)} ${label}`);
    }
  }
  if (effects.dao_mark) {
    for (const [path, v] of Object.entries(effects.dao_mark)) {
      out.push(`${pathName(path)}: đạo ngân ×${Math.round((1 + v) * 100)}%`);
    }
  }
  if (effects.path_damage) {
    for (const [path, v] of Object.entries(effects.path_damage)) {
      out.push(`${pct(v)} sát thương ${pathName(path)}`);
    }
  }
  if (effects.path_resistance) {
    for (const [path, v] of Object.entries(effects.path_resistance)) {
      out.push(`Giảm ${pct(Math.abs(v)).replace("+", "")} sát thương ${pathName(path)} nhận`);
    }
  }
  if (effects.mana_regen) out.push(`${pct(effects.mana_regen)} hồi Chân Nguyên`);
  if (effects.max_mana) out.push(`${pct(effects.max_mana)} Chân Nguyên tối đa`);
  if (effects.cultivation_efficiency) out.push(`${pct(effects.cultivation_efficiency)} hiệu quả tu luyện`);
  if (effects.dao_mark_gain) out.push(`${pct(effects.dao_mark_gain)} Đạo Ngân nhận được`);
  if (effects.luck) out.push(`+${effects.luck} Vận`);
  if (effects.refine) out.push(`${pct(effects.refine)} tỉ lệ Luyện Cổ`);
  if (effects.healing_received) out.push(`${pct(effects.healing_received)} hồi phục nhận`);
  if (effects.mental_resistance) out.push(`${pct(effects.mental_resistance)} kháng hiệu ứng tâm lý`);
  if (effects.soul_defense) out.push(`${pct(effects.soul_defense)} phòng thủ hồn`);
  if (effects.aperture_stress_bonus && effects.aperture_stress_bonus > 0) {
    out.push(`Áp lực Không Khiếu tăng ×${Math.round((1 + effects.aperture_stress_bonus) * 10) / 10}`);
  }

  return out;
}

// Alias giữ tên cũ cho các file đã import formatConstitutionEffects.
export const formatConstitutionEffects = formatGameEffects;
