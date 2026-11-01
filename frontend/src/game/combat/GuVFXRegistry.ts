/**
 * Catalog and mapping for authored Gu World VFX Atlases.
 */

export interface VfxDefinition {
  key: string;
  textureUrl: string;
  atlasUrl: string;
  frameRate?: number;
  scale?: number;
  category: "attack" | "shield" | "status" | "gu" | "elemental" | "boss";
  description?: string;
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
    description: "Khiên hào quang ánh kim bảo hộ",
  },
  fx_thu_giap_da_gpt: {
    key: "fx_thu_giap_da_gpt",
    textureUrl: "/assets/vfx/fx_thu_giap_da_gpt.png",
    atlasUrl: "/assets/vfx/fx_thu_giap_da_gpt.json",
    frameRate: 12,
    scale: 1.1,
    category: "shield",
    description: "Giáp đá thạch cự khiên",
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

  // === GU SIGNATURE ABILITIES ===
  fx_gu_hoa_du: {
    key: "fx_gu_hoa_du",
    textureUrl: "/assets/vfx/fx_gu_hoa_du.png",
    atlasUrl: "/assets/vfx/fx_gu_hoa_du.json",
    frameRate: 14,
    scale: 1.25,
    category: "gu",
    description: "Hỏa Dực Cổ bộc phát ngọn lửa",
  },
  fx_gu_thiem_quang: {
    key: "fx_gu_thiem_quang",
    textureUrl: "/assets/vfx/fx_gu_thiem_quang.png",
    atlasUrl: "/assets/vfx/fx_gu_thiem_quang.json",
    frameRate: 16,
    scale: 1.3,
    category: "gu",
    description: "Thiểm Quang Cổ chớp sáng chói lòa",
  },
  fx_gu_dien_nhan_gpt_dau: {
    key: "fx_gu_dien_nhan_gpt_dau",
    textureUrl: "/assets/vfx/fx_gu_dien_nhan_gpt_dau.png",
    atlasUrl: "/assets/vfx/fx_gu_dien_nhan_gpt_dau.json",
    frameRate: 14,
    scale: 1.2,
    category: "gu",
    description: "Điện Nhãn Cổ phóng luồng điện quang",
  },
  fx_gu_thinh_phong_gpt_kich_hoat: {
    key: "fx_gu_thinh_phong_gpt_kich_hoat",
    textureUrl: "/assets/vfx/fx_gu_thinh_phong_gpt_kich_hoat.png",
    atlasUrl: "/assets/vfx/fx_gu_thinh_phong_gpt_kich_hoat.json",
    frameRate: 14,
    scale: 1.15,
    category: "gu",
    description: "Thính Phong Cổ gió lốc cuộn quanh",
  },
  fx_gu_xa_tin_dau_gpt_giu: {
    key: "fx_gu_xa_tin_dau_gpt_giu",
    textureUrl: "/assets/vfx/fx_gu_xa_tin_dau_gpt_giu.png",
    atlasUrl: "/assets/vfx/fx_gu_xa_tin_dau_gpt_giu.json",
    frameRate: 14,
    scale: 1.2,
    category: "gu",
    description: "Xà Tín Cổ độc xà uốn lượn",
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
 * Returns the best-matching VFX key for a skill name or skill ID.
 */
export function resolveSkillVfxKey(skillId: string, skillName?: string): string {
  const normalized = `${skillId} ${skillName ?? ""}`.toLowerCase();

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
