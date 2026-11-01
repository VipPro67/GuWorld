/**
 * Catalog and mapping for authored Gu World VFX Atlases.
 * Supports direct resolution by gu_id, rank, and skill identifiers.
 */

export interface VfxDefinition {
  key: string;
  textureUrl: string;
  atlasUrl: string;
  frameRate?: number;
  scale?: number;
  category: "attack" | "shield" | "status" | "gu" | "elemental" | "boss";
  description?: string;
  guId?: string;
}

/** Symbolic profile recipe keys mapped to reusable, authored frame atlases. */
const VFX_TEMPLATE_ATLASES: Record<string, string> = {
  "cast/fire": "fx_gu_hoa_du",
  "cast/water": "fx_thu_nuoc_dang_gpt",
  "cast/frost": "fx_gu_nguyet_quang",
  "cast/lightning": "fx_gu_dien_nhan_gpt_dau",
  "cast/poison": "fx_thu_bot_doc_gpt",
  "cast/earth": "fx_dat_nut",
  "cast/light": "fx_cot_sang",
  "cast/dark": "fx_gu_huyet_nguyet",
  "cast/sword": "fx_gu_nguyet_quang",
  "cast/wind": "fx_gu_thinh_phong_gpt_kich_hoat",
  "cast/strength": "fx_thu_giap_da_gpt",
  "cast/formation": "fx_dat_nut",
  "cast/sound": "fx_song_gam",
  "cast/heal": "fx_hoi_sinh",
  "cast/water_shield": "fx_khien_anh_kim",
  "cast/transformation": "fx_thoi_trang_xuan_thu_quyen",
  "projectiles/fireball": "fx_thu_cau_dung_nham_gpt",
  "projectiles/spark": "fx_thu_set_danh_gpt",
  "projectiles/poison_spit": "fx_thu_can_doc_gpt",
  "projectiles/dark_orb": "fx_gu_huyet_bao",
  "projectiles/moon_blade": "fx_gu_nguyet_quang",
  "projectiles/wind_blade": "fx_gu_thinh_phong_gpt_kich_hoat",
  "projectiles/water_arrow": "fx_thu_nuoc_dang_gpt",
  "impact/fire_burst": "fx_thu_vung_dung_nham_gpt",
  "impact/water": "fx_thu_nuoc_dang_gpt",
  "impact/frost": "fx_gu_nguyet_quang",
  "impact/lightning_strike": "fx_thu_set_danh_gpt",
  "impact/poison_cloud": "fx_thu_bot_doc_gpt",
  "impact/heavy_strike": "fx_hit_dam",
  "impact/light_slash": "fx_gu_nguyet_quang",
  "impact/dark_burst": "fx_gu_huyet_bao",
  "impact/sword_slash": "fx_hit_chem",
  "impact/wind_slash": "fx_bui_lao",
  "impact/shield": "fx_khien_anh_kim",
  "impact/heal": "fx_hoi_sinh",
  "impact/ground_burst": "fx_dat_nut",
  "impact/armor": "fx_thu_giap_da_gpt",
  "impact/sound": "fx_song_gam",
  "impact/slash": "fx_hit_chem",
  "impact/strength": "fx_hit_dam",
};

export function resolveVisualTemplate(template?: string | null): string | null {
  if (!template) return null;
  const key = VFX_TEMPLATE_ATLASES[template] ?? template;
  return GU_VFX_CATALOG[key] ? key : null;
}

export const GU_VFX_CATALOG: Record<string, VfxDefinition> = {
  // === CORE COMBAT & HIT IMPACTS ===
  fx_hit_chem: {
    key: "fx_hit_chem",
    textureUrl: "/assets/vfx/fx_hit_chem.png",
    atlasUrl: "/assets/vfx/fx_hit_chem.json",
    frameRate: 16,
    scale: 1.2,
    category: "attack",
    description: "Vệt chém năng lượng cận chiến cơ bản",
  },
  fx_hit_dam: {
    key: "fx_hit_dam",
    textureUrl: "/assets/vfx/fx_hit_dam.png",
    atlasUrl: "/assets/vfx/fx_hit_dam.json",
    frameRate: 16,
    scale: 1.1,
    category: "attack",
    description: "Đòn đánh va chạm / chấn kích",
  },
  fx_hit_dien: {
    key: "fx_hit_dien",
    textureUrl: "/assets/vfx/fx_hit_dien.png",
    atlasUrl: "/assets/vfx/fx_hit_dien.json",
    frameRate: 16,
    scale: 1.2,
    category: "attack",
    description: "Tia điện giật trúng đích",
  },

  // === SHIELDS & DEFENSE ===
  fx_khien_anh_kim: {
    key: "fx_khien_anh_kim",
    textureUrl: "/assets/vfx/fx_khien_anh_kim.png",
    atlasUrl: "/assets/vfx/fx_khien_anh_kim.json",
    frameRate: 14,
    scale: 1.15,
    category: "shield",
    description: "Khiên hào quang ánh kim bảo hộ (Bạch Ngọc Cổ / Kim Cương Cổ)",
    guId: "GU_WHITE_JADE",
  },
  fx_thu_giap_da_gpt: {
    key: "fx_thu_giap_da_gpt",
    textureUrl: "/assets/vfx/fx_thu_giap_da_gpt.png",
    atlasUrl: "/assets/vfx/fx_thu_giap_da_gpt.json",
    frameRate: 12,
    scale: 1.1,
    category: "shield",
    description: "Giáp đá thạch cự khiên",
    guId: "GU_STONE_ARMOR",
  },
  fx_thu_vo_giap_gpt: {
    key: "fx_thu_vo_giap_gpt",
    textureUrl: "/assets/vfx/fx_thu_vo_giap_gpt.webp",
    atlasUrl: "/assets/vfx/fx_thu_vo_giap_gpt.json",
    frameRate: 16,
    scale: 1.0,
    category: "shield",
    description: "Hiệu ứng khiên vỡ vụn",
  },

  // === STATUS & CONTROL ===
  fx_hieu_ung_choang: {
    key: "fx_hieu_ung_choang",
    textureUrl: "/assets/vfx/fx_hieu_ung_choang.png",
    atlasUrl: "/assets/vfx/fx_hieu_ung_choang.json",
    frameRate: 12,
    scale: 0.9,
    category: "status",
    description: "Vòng sao choáng váng trên đầu",
  },
  fx_hieu_ung_doc: {
    key: "fx_hieu_ung_doc",
    textureUrl: "/assets/vfx/fx_hieu_ung_doc.png",
    atlasUrl: "/assets/vfx/fx_hieu_ung_doc.json",
    frameRate: 12,
    scale: 1.0,
    category: "status",
    description: "Độc khí tím bao phủ",
  },
  fx_hieu_ung_troi: {
    key: "fx_hieu_ung_troi",
    textureUrl: "/assets/vfx/fx_hieu_ung_troi.png",
    atlasUrl: "/assets/vfx/fx_hieu_ung_troi.json",
    frameRate: 12,
    scale: 1.0,
    category: "status",
    description: "Dây leo trói chân",
  },
  fx_hoi_sinh: {
    key: "fx_hoi_sinh",
    textureUrl: "/assets/vfx/fx_hoi_sinh.png",
    atlasUrl: "/assets/vfx/fx_hoi_sinh.json",
    frameRate: 14,
    scale: 1.2,
    category: "status",
    description: "Cột hào quang hồi máu / hồi sinh",
  },

  // === GU SIGNATURE ABILITIES (WITH GU_ID RESOLUTION) ===
  fx_gu_nguyet_quang: {
    key: "fx_gu_nguyet_quang",
    textureUrl: "/assets/vfx/fx_gu_nguyet_quang.png",
    atlasUrl: "/assets/vfx/fx_gu_nguyet_quang.json",
    frameRate: 16,
    scale: 1.3,
    category: "gu",
    description: "Nguyệt Quang Cổ / Nguyệt Nhận kiếm khí trăng khuyết xanh ngọc",
    guId: "GU_MOONLIGHT",
  },
  fx_gu_huyet_nguyet: {
    key: "fx_gu_huyet_nguyet",
    textureUrl: "/assets/vfx/fx_gu_huyet_nguyet.png",
    atlasUrl: "/assets/vfx/fx_gu_huyet_nguyet.json",
    frameRate: 16,
    scale: 1.3,
    category: "gu",
    description: "Huyết Nguyệt Cổ kiếm khí trăng máu đỏ thẫm",
    guId: "GU_BLOOD_MOON",
  },
  fx_gu_huyet_bao: {
    key: "fx_gu_huyet_bao",
    textureUrl: "/assets/vfx/fx_gu_huyet_bao.png",
    atlasUrl: "/assets/vfx/fx_gu_huyet_bao.json",
    frameRate: 14,
    scale: 1.25,
    category: "gu",
    description: "Huyết Bạo / Huyết Đạo bạo phát tung toé",
    guId: "GU_BLOOD_EXPLOSION",
  },
  fx_gu_bach_cot: {
    key: "fx_gu_bach_cot",
    textureUrl: "/assets/vfx/fx_gu_bach_cot.png",
    atlasUrl: "/assets/vfx/fx_gu_bach_cot.json",
    frameRate: 14,
    scale: 1.25,
    category: "gu",
    description: "Bạch Cốt Mâu / Cốt Đạo đâm xuyên",
    guId: "GU_WHITE_BONE",
  },
  fx_thoi_trang_xuan_thu_quyen: {
    key: "fx_thoi_trang_xuan_thu_quyen",
    textureUrl: "/assets/vfx/fx_thoi_trang_xuan_thu_quyen.png",
    atlasUrl: "/assets/vfx/fx_thoi_trang_xuan_thu_quyen.json",
    frameRate: 15,
    scale: 1.25,
    category: "gu",
    description: "Xuân Thu Thiền / Thời Đạo quyền kích vặn xoắn thời gian",
    guId: "GU_SPRING_AUTUMN_CICADA",
  },
  fx_gu_hoa_du: {
    key: "fx_gu_hoa_du",
    textureUrl: "/assets/vfx/fx_gu_hoa_du.png",
    atlasUrl: "/assets/vfx/fx_gu_hoa_du.json",
    frameRate: 14,
    scale: 1.25,
    category: "gu",
    description: "Hỏa Dực Cổ bộc phát ngọn lửa",
    guId: "GU_FIRE_WINGS",
  },
  fx_gu_thiem_quang: {
    key: "fx_gu_thiem_quang",
    textureUrl: "/assets/vfx/fx_gu_thiem_quang.png",
    atlasUrl: "/assets/vfx/fx_gu_thiem_quang.json",
    frameRate: 16,
    scale: 1.3,
    category: "gu",
    description: "Thiểm Quang Cổ chớp sáng chói lòa",
    guId: "GU_FLASH_LIGHT",
  },
  fx_gu_dien_nhan_gpt_dau: {
    key: "fx_gu_dien_nhan_gpt_dau",
    textureUrl: "/assets/vfx/fx_gu_dien_nhan_gpt_dau.png",
    atlasUrl: "/assets/vfx/fx_gu_dien_nhan_gpt_dau.json",
    frameRate: 14,
    scale: 1.2,
    category: "gu",
    description: "Điện Nhãn Cổ phóng luồng điện quang",
    guId: "GU_LIGHTNING_EYE",
  },
  fx_gu_thinh_phong_gpt_kich_hoat: {
    key: "fx_gu_thinh_phong_gpt_kich_hoat",
    textureUrl: "/assets/vfx/fx_gu_thinh_phong_gpt_kich_hoat.png",
    atlasUrl: "/assets/vfx/fx_gu_thinh_phong_gpt_kich_hoat.json",
    frameRate: 14,
    scale: 1.15,
    category: "gu",
    description: "Thính Phong Cổ gió lốc cuộn quanh",
    guId: "GU_WIND_HEAR",
  },
  fx_gu_xa_tin_dau_gpt_giu: {
    key: "fx_gu_xa_tin_dau_gpt_giu",
    textureUrl: "/assets/vfx/fx_gu_xa_tin_dau_gpt_giu.png",
    atlasUrl: "/assets/vfx/fx_gu_xa_tin_dau_gpt_giu.json",
    frameRate: 14,
    scale: 1.2,
    category: "gu",
    description: "Xà Tín Cổ độc xà uốn lượn",
    guId: "GU_SNAKE_TONGUE",
  },

  // === ELEMENTAL SPELLS ===
  fx_thu_set_danh_gpt: {
    key: "fx_thu_set_danh_gpt",
    textureUrl: "/assets/vfx/fx_thu_set_danh_gpt.webp",
    atlasUrl: "/assets/vfx/fx_thu_set_danh_gpt.json",
    frameRate: 16,
    scale: 1.3,
    category: "elemental",
    description: "Lôi điện cuồng lôi trừng phạt",
  },
  fx_thu_set_giang_gpt: {
    key: "fx_thu_set_giang_gpt",
    textureUrl: "/assets/vfx/fx_thu_set_giang_gpt.png",
    atlasUrl: "/assets/vfx/fx_thu_set_giang_gpt.json",
    frameRate: 16,
    scale: 1.3,
    category: "elemental",
    description: "Thiên lôi giáng thế",
  },
  fx_thu_cau_dung_nham_gpt: {
    key: "fx_thu_cau_dung_nham_gpt",
    textureUrl: "/assets/vfx/fx_thu_cau_dung_nham_gpt.png",
    atlasUrl: "/assets/vfx/fx_thu_cau_dung_nham_gpt.json",
    frameRate: 14,
    scale: 1.2,
    category: "elemental",
    description: "Cầu dung nham rực lửa",
  },
  fx_thu_vung_dung_nham_gpt: {
    key: "fx_thu_vung_dung_nham_gpt",
    textureUrl: "/assets/vfx/fx_thu_vung_dung_nham_gpt.png",
    atlasUrl: "/assets/vfx/fx_thu_vung_dung_nham_gpt.json",
    frameRate: 12,
    scale: 1.25,
    category: "elemental",
    description: "Vũng nham thạch thiêu đốt mặt đất",
  },
  fx_thu_bot_doc_gpt: {
    key: "fx_thu_bot_doc_gpt",
    textureUrl: "/assets/vfx/fx_thu_bot_doc_gpt.png",
    atlasUrl: "/assets/vfx/fx_thu_bot_doc_gpt.json",
    frameRate: 14,
    scale: 1.15,
    category: "elemental",
    description: "Bọt độc phun trào",
  },
  fx_thu_can_doc_gpt: {
    key: "fx_thu_can_doc_gpt",
    textureUrl: "/assets/vfx/fx_thu_can_doc_gpt.png",
    atlasUrl: "/assets/vfx/fx_thu_can_doc_gpt.json",
    frameRate: 14,
    scale: 1.1,
    category: "elemental",
    description: "Vết cắn độc xà",
  },
  fx_thu_nuoc_dang_gpt: {
    key: "fx_thu_nuoc_dang_gpt",
    textureUrl: "/assets/vfx/fx_thu_nuoc_dang_gpt.png",
    atlasUrl: "/assets/vfx/fx_thu_nuoc_dang_gpt.json",
    frameRate: 14,
    scale: 1.2,
    category: "elemental",
    description: "Sóng nước dâng trào",
  },
  fx_thu_xoay_nuoc_gpt: {
    key: "fx_thu_xoay_nuoc_gpt",
    textureUrl: "/assets/vfx/fx_thu_xoay_nuoc_gpt.png",
    atlasUrl: "/assets/vfx/fx_thu_xoay_nuoc_gpt.json",
    frameRate: 14,
    scale: 1.25,
    category: "elemental",
    description: "Xoáy nước cuồn cuộn",
  },
  fx_song_gam: {
    key: "fx_song_gam",
    textureUrl: "/assets/vfx/fx_song_gam.png",
    atlasUrl: "/assets/vfx/fx_song_gam.json",
    frameRate: 14,
    scale: 1.3,
    category: "elemental",
    description: "Sóng âm gầm thét chấn động",
  },
  fx_dat_nut: {
    key: "fx_dat_nut",
    textureUrl: "/assets/vfx/fx_dat_nut.png",
    atlasUrl: "/assets/vfx/fx_dat_nut.json",
    frameRate: 12,
    scale: 1.2,
    category: "elemental",
    description: "Mặt đất nứt toác",
  },
  fx_da_vo: {
    key: "fx_da_vo",
    textureUrl: "/assets/vfx/fx_da_vo.png",
    atlasUrl: "/assets/vfx/fx_da_vo.json",
    frameRate: 14,
    scale: 1.2,
    category: "elemental",
    description: "Đá vỡ tung toé",
  },
  fx_cot_sang: {
    key: "fx_cot_sang",
    textureUrl: "/assets/vfx/fx_cot_sang.png",
    atlasUrl: "/assets/vfx/fx_cot_sang.json",
    frameRate: 14,
    scale: 1.2,
    category: "elemental",
    description: "Cột sáng thiên địa hội tụ",
  },
  fx_bui_lao: {
    key: "fx_bui_lao",
    textureUrl: "/assets/vfx/fx_bui_lao.png",
    atlasUrl: "/assets/vfx/fx_bui_lao.json",
    frameRate: 14,
    scale: 1.1,
    category: "elemental",
    description: "Lốc bụi phi nhanh",
  },
};

/**
 * Direct mapping dictionary from canonical gu_id or slug to VFX key.
 */
export const GU_ID_TO_VFX: Record<string, string> = {
  // Moonblade / Sword Path
  GU_MOONLIGHT: "fx_gu_nguyet_quang",
  nguyet_quang_gu: "fx_gu_nguyet_quang",
  nguyet_nhan_gu: "fx_gu_nguyet_quang",

  // Blood Path
  GU_BLOOD_MOON: "fx_gu_huyet_nguyet",
  huyet_nguyet_gu: "fx_gu_huyet_nguyet",
  GU_BLOOD_EXPLOSION: "fx_gu_huyet_bao",
  huyet_bao_gu: "fx_gu_huyet_bao",

  // Bone Path
  GU_WHITE_BONE: "fx_gu_bach_cot",
  bach_cot_gu: "fx_gu_bach_cot",
  bach_cot_mau: "fx_gu_bach_cot",

  // Time Path
  GU_SPRING_AUTUMN_CICADA: "fx_thoi_trang_xuan_thu_quyen",
  xuan_thu_thien: "fx_thoi_trang_xuan_thu_quyen",

  // Fire Path
  GU_FIRE_WINGS: "fx_gu_hoa_du",
  hoa_du_gu: "fx_gu_hoa_du",

  // Lightning Path
  GU_LIGHTNING_EYE: "fx_gu_dien_nhan_gpt_dau",
  dien_nhan_gu: "fx_gu_dien_nhan_gpt_dau",
  GU_FLASH_LIGHT: "fx_gu_thiem_quang",
  thiem_quang_gu: "fx_gu_thiem_quang",
  GU_THUNDER_STRIKE: "fx_thu_set_danh_gpt",
  loi_phat_gu: "fx_thu_set_danh_gpt",

  // Wind Path
  GU_WIND_HEAR: "fx_gu_thinh_phong_gpt_kich_hoat",
  thinh_phong_gu: "fx_gu_thinh_phong_gpt_kich_hoat",

  // Poison Path
  GU_SNAKE_TONGUE: "fx_gu_xa_tin_dau_gpt_giu",
  xa_tin_gu: "fx_gu_xa_tin_dau_gpt_giu",

  // Defense / Armor
  GU_STONE_ARMOR: "fx_thu_giap_da_gpt",
  giap_da_gu: "fx_thu_giap_da_gpt",
  GU_WHITE_JADE: "fx_khien_anh_kim",
  bach_ngoc_gu: "fx_khien_anh_kim",

  // Sound / Force
  GU_SOUND_ROAR: "fx_song_gam",
  song_gam_gu: "fx_song_gam",
};

export interface ResolvedGuVfx {
  vfxKey: string;
  scale: number;
  frameRate: number;
  category: VfxDefinition["category"];
}

/**
 * Resolves VFX based on gu_id and rank, falling back to skill name/id keywords.
 */
export function resolveGuVfx(
  guId?: string | null,
  rank?: number | null,
  skillId?: string | null,
  skillName?: string | null
): ResolvedGuVfx {
  let vfxKey = "fx_hit_chem";

  // 1. Direct guId lookup
  if (guId) {
    const rawId = guId.trim();
    const lowerId = rawId.toLowerCase();
    if (GU_ID_TO_VFX[rawId]) {
      vfxKey = GU_ID_TO_VFX[rawId];
    } else if (GU_ID_TO_VFX[lowerId]) {
      vfxKey = GU_ID_TO_VFX[lowerId];
    }
  }

  // 2. Keyword fallback if not resolved yet
  if (vfxKey === "fx_hit_chem" && (skillId || skillName)) {
    vfxKey = resolveSkillVfxKey(skillId ?? "", skillName ?? undefined);
  }

  const def = GU_VFX_CATALOG[vfxKey];
  const baseScale = def?.scale ?? 1.0;
  const baseFrameRate = def?.frameRate ?? 14;

  // Rank-based scaling (Rank 1-5 phàm cổ, Rank 6-9 tiên cổ)
  const r = rank ?? 1;
  const rankScaleMultiplier = r >= 6 ? 1.25 : r >= 4 ? 1.12 : 1.0;
  const rankFrameRateBonus = r >= 6 ? 2 : 0;

  return {
    vfxKey,
    scale: baseScale * rankScaleMultiplier,
    frameRate: baseFrameRate + rankFrameRateBonus,
    category: def?.category ?? "attack",
  };
}

/**
 * Returns the best-matching VFX key for a skill name or skill ID.
 */
export function resolveSkillVfxKey(skillId: string, skillName?: string): string {
  const normalized = `${skillId} ${skillName ?? ""}`.toLowerCase();

  // Moonlight / Sword
  if (normalized.includes("nguyệt quang") || normalized.includes("nguyệt nhận") || normalized.includes("kiếm")) {
    return "fx_gu_nguyet_quang";
  }

  // Blood
  if (normalized.includes("huyết nguyệt")) return "fx_gu_huyet_nguyet";
  if (normalized.includes("huyết") || normalized.includes("máu")) return "fx_gu_huyet_bao";

  // Bone
  if (normalized.includes("cốt") || normalized.includes("xương")) return "fx_gu_bach_cot";

  // Time / Spring Autumn Cicada
  if (normalized.includes("xuân thu") || normalized.includes("thời gian")) return "fx_thoi_trang_xuan_thu_quyen";

  // Fire / Flame
  if (normalized.includes("hỏa") || normalized.includes("lửa") || normalized.includes("hoa_du") || normalized.includes("diễm")) {
    return normalized.includes("vùng") || normalized.includes("mưa") ? "fx_thu_vung_dung_nham_gpt" : "fx_gu_hoa_du";
  }

  // Lightning / Thunder
  if (normalized.includes("lôi") || normalized.includes("sét") || normalized.includes("điện") || normalized.includes("thiểm")) {
    if (normalized.includes("thiểm") || normalized.includes("quang")) return "fx_gu_thiem_quang";
    if (normalized.includes("điện") || normalized.includes("nhãn")) return "fx_gu_dien_nhan_gpt_dau";
    return "fx_thu_set_danh_gpt";
  }

  // Wind / Gale
  if (normalized.includes("phong") || normalized.includes("gió") || normalized.includes("thính")) {
    return "fx_gu_thinh_phong_gpt_kich_hoat";
  }

  // Poison / Venom
  if (normalized.includes("độc") || normalized.includes("xà") || normalized.includes("tin")) {
    if (normalized.includes("xà") || normalized.includes("tin")) return "fx_gu_xa_tin_dau_gpt_giu";
    return "fx_thu_bot_doc_gpt";
  }

  // Water / Ice
  if (normalized.includes("thủy") || normalized.includes("nước") || normalized.includes("băng") || normalized.includes("tuyết")) {
    return normalized.includes("xoáy") ? "fx_thu_xoay_nuoc_gpt" : "fx_thu_nuoc_dang_gpt";
  }

  // Shield / Rock / Defense
  if (normalized.includes("khiên") || normalized.includes("giáp") || normalized.includes("hộ") || normalized.includes("kim cương") || normalized.includes("đá")) {
    return normalized.includes("đá") ? "fx_thu_giap_da_gpt" : "fx_khien_anh_kim";
  }

  // Sound / Roar / Force
  if (normalized.includes("gầm") || normalized.includes("hống") || normalized.includes("sóng") || normalized.includes("âm")) {
    return "fx_song_gam";
  }

  // Earth / Quake
  if (normalized.includes("đất") || normalized.includes("địa") || normalized.includes("nứt") || normalized.includes("chấn")) {
    return "fx_dat_nut";
  }

  // Heal / Life
  if (normalized.includes("hồi") || normalized.includes("trị") || normalized.includes("sinh")) {
    return "fx_hoi_sinh";
  }

  // Default basic slash
  return "fx_hit_chem";
}
