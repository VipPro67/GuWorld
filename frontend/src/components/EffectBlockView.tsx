import React from "react";
import type { EffectBlock, EffectEnvelope, EffectLike, EffectOperation, EffectTarget } from "@/lib/effect-blocks";
import { translatePath } from "@/lib/translate";
import { formatEffectTarget, renderableEffectBlocks } from "@/lib/effect-presentation";
import { COMBAT_STATUSES, NATIVE_HOOKS } from "@/lib/wiki/effect-glossary";

const TRIGGER_CONFIG: Record<string, { label: string; color: string; icon: string }> = {
  ALWAYS_ON: { label: "Nội Tại", color: "bg-purple-950/60 text-purple-300 border-purple-800/50", icon: "💎" },
  PASSIVE: { label: "Nội Tại", color: "bg-purple-950/60 text-purple-300 border-purple-800/50", icon: "💎" },
  COMBAT_PASSIVE: { label: "Nội Tại Chiến Đấu", color: "bg-purple-950/60 text-purple-300 border-purple-800/50", icon: "⚡" },
  ON_ACTION: { label: "Khi Ra Chiêu", color: "bg-amber-950/60 text-amber-300 border-amber-800/50", icon: "⚔️" },
  ON_HIT: { label: "Khi Đánh Trúng", color: "bg-rose-950/60 text-rose-300 border-rose-800/50", icon: "🎯" },
  BEFORE_HIT: { label: "Trước Khi Bị Đánh", color: "bg-cyan-950/60 text-cyan-300 border-cyan-800/50", icon: "🛡️" },
  ON_TURN_START: { label: "Đầu Lượt", color: "bg-emerald-950/60 text-emerald-300 border-emerald-800/50", icon: "⏳" },
  ON_BATTLE_START: { label: "Vào Trận", color: "bg-yellow-950/60 text-yellow-300 border-yellow-800/50", icon: "🚩" },
  BEFORE_ENEMY_ATTACK: { label: "Trước Khi Địch Đánh", color: "bg-orange-950/60 text-orange-300 border-orange-800/50", icon: "👁️" },
  ON_DEATH: { label: "Khi Tử Vong", color: "bg-red-950/70 text-red-300 border-red-800/60", icon: "☠️" },
  ON_TICK: { label: "Mỗi Chu Kỳ", color: "bg-teal-950/60 text-teal-300 border-teal-800/50", icon: "🔄" },
  ON_CAST: { label: "Khi Niệm", color: "bg-indigo-950/60 text-indigo-300 border-indigo-800/50", icon: "✨" },
  CAPABILITY: { label: "Năng Lực", color: "bg-sky-950/60 text-sky-300 border-sky-800/50", icon: "🌐" },
  ON_CONSUME: { label: "Khi Dùng", color: "bg-emerald-950/60 text-emerald-300 border-emerald-800/50", icon: "🧪" },
  ON_USE: { label: "Khi Kích Hoạt", color: "bg-amber-950/60 text-amber-300 border-amber-800/50", icon: "📜" },
};

const STAT_NAMES: Record<string, string> = {
  ATTACK: "Công",
  DEFENSE: "Thủ",
  HP: "Sinh Lực",
  SPEED: "Tốc Độ",
  CRIT_RATE: "Bạo Kích",
  TENACITY: "Kiên Cường",
  CC_RESIST: "Kháng Khống Chế",
  LUCK: "Vận Khí",
  ALL: "Toàn Chỉ Số",
  DAMAGE_DEALT: "Sát Thương Gây Ra",
  MOVEMENT: "Di Chuyển (ô)",
  INVESTIGATION: "Điều Tra",
  HEARING: "Thính Giác",
  DISGUISE: "Ngụy Trang",
  TIME_SPEED: "Tốc Độ Thời Gian",
  REFINE_SUCCESS: "Tỷ Lệ Luyện Cổ",
  REFINE_MANA_TIER: "Tinh Luyện Chân Nguyên",
  STORAGE: "Sức Chứa",
  DAO_MARK: "Đạo Ngân",
  PATH_CONFLICT_REDUCTION: "Giảm Xung Đột Đạo",
  PATH_RESISTANCE: "Kháng Lưu Phái",
};

const STATUS_NAMES: Record<string, string> = {
  BLEED: "Chảy Máu",
  POISON: "Trúng Độc",
  BURN: "Thiêu Đốt",
  STUN: "Choáng",
  FREEZE: "Đóng Băng",
  SILENCE: "Câm Lặng",
  GU_SEAL: "Khóa Cổ",
  HASTE: "Tăng Tốc",
  SLOW: "Làm Chậm",
  SLOW1: "Làm Chậm Cấp 1",
  SLOW_DAMAGE: "Sát Thương Chậm",
  WEAK: "Suy Yếu",
  ATK_UP: "Tăng Công",
  DEF_UP: "Tăng Thủ",
  FLYING: "Bay Lượn",
  PHI_HANH: "Phi Hành",
  REGEN: "Hồi Phục",
  THORNS: "Gai Phản Đòn",
  TAUNT: "Khiêu Khích",
  INVINCIBLE: "Bất Hoại",
  LAST_STAND: "Bất Tử",
  EVASION: "Né Tránh",
  CONFUSE: "Hỗn Loạn",
  CONFUSION: "Hỗn Loạn",
  FEAR: "Hoảng Sợ",
  STEALTH: "Tàng Hình",
  SLEEP: "Ngủ Say",
  BLIND: "Mù Lòa",
  CHARM: "Mê Hoặc",
  BIND: "Trói Buộc",
  ROOT: "Bất Động",
  DECAY: "Hủ Bại",
  FORGET_EGO: "Lãng Quên Bản Ngã",
  FORGET_REALITY: "Lãng Quên Thực Tại",
  ARMOR_BREAK: "Phá Giáp",
  DAMAGE_REDUCTION_UP: "Giảm Thương",
  REVIVE_LOCK: "Khóa Hồi Sinh",
  ESSENCE_LEAK: "Thất Thoát Chân Nguyên",
  SOUL_WOUND: "Tổn Thương Hồn",
  THOUGHT_CHAOS: "Tạp Niệm",
  DAO_CONFLICT: "Xung Đột Đạo",
  BACKLASH: "Phản Phệ",
  DREAM_PRISON: "Mộng Cảnh Giam Cầm",
  DREAM_ILLUSION: "Mộng Ảnh Ẩn Thân",
  LUCK_SIPHON: "Đoạt Vận",
  LUCK_CURSE_REVERSE: "Họa Hề Phúc Sở Ỷ",
  BERSERK: "Cuồng Bạo",
  CALCULATED: "Động Tất",
  BLOOD_SACRIFICE: "Huyết Tế",
  BEAR_FORM: "Hóa Thân Cự Hùng",
  EAGLE_FORM: "Hóa Thân Kim Điêu",
  ADAPTIVE_RESIST: "Thích Ứng Đạo Pháp",
  BONE_ARMOR: "Bạch Cốt Giáp",
  METAL_AEGIS: "Cương Thiết Hộ Thể",
  CLOUD_MIST: "Vân Vụ Ảo Mộng",
  REVERSE_FLOW_SEAL: "Nghịch Lưu Hộ Thân Quỹ",
  FAMILIAR_FACE: "Gặp Mặt Từng Quen Biết",
  TIME_SPEED_SURGE: "Phi Thuyền Lưu Tốc",
  FIVE_REGIONAL_SUPPRESSION: "Ngũ Giới Áp Chế",
  NINETY_NINE_SHIELD: "Cửu Cửu Liên Hoàn Giáp",
  YAMA_EMPEROR: "Diêm Đế Thể",
  BLOOD_ROBE: "Huyết Nhiễm Chinh Bào",
  FUTURE_SELF: "Vị Lai Thân",
  EATING_LOSS: "Cật Khuy Thị Phúc",
  FAREWELL_CURSE: "Tống Hữu Chi Phong",
  SWORD_MARK: "Kiếm Ngân Tác Mệnh",
  REFINE_SELF: "Luyện Kỷ Thể",
  ALL_PEOPLE_UNITED: "Vạn Chúng Nhất Tâm",
  DRY_WOOD_SPRING: "Khô Mộc Phùng Xuân",
  ICE_COFFIN: "Cực Băng Trấn Tiên Quan",
  TOTEM_GOLDEN_BELL: "Đồ Đằng Kim Chung",
  GREAT_NIGHT: "Đại Dạ Di Thiên",
};

const UNIQUE_NAMES: Record<string, string> = {
  REBORN: "Hồi Sinh Khi Chết",
  BYPASS_SHIELD: "Xuyên Hộ Thể (Bỏ Qua Khiên)",
  SUMMON_PHANTOM: "Triệu Hồi Hư Ảnh",
  SUMMON_WOLF_FRONTLINE: "Triệu Hồi Sói Tiền Hàng",
  SWORD_DRAGON_TRANSFORM: "Kiếm Long Biến",
  FULL_SOUL_HEAL: "Đầy Hồn Phục",
  BORROW_PAST_POWER: "Mượn Lực Quá Khứ",
  UNRAVEL_RIDDLE: "Giải Đoán Câu Đố",
  CHAOS_GAMBLE: "Hỗn Độn Đánh Cược",
  SEIZE_INITIATIVE: "Đoạt Tiên Cơ (Đánh Trước)",
  INSTANT_WIN: "Chiến Thắng Tuyệt Đối",
  FIRST_TURN_EVADE: "Né Tuyệt Đối Lượt Đầu",
  FORGET_INTENT_INTERRUPT: "Đoán Ý & Ngắt Chiêu",
  FORGET_WILL_CONFUSE: "Ý Niệm Hỗn Loạn",
  DISGUISE: "Ngụy Trang Khí Tức",
  CHANGE_FORM_DISGUISE: "Biến Hóa Ngụy Trang",
  PERFECT_DISGUISE: "Ngụy Trang Hoàn Mỹ",
  REMOTE_TRADE: "Giao Dịch Từ Xa",
  MARKETPLACE_ACCESS: "Truy Cập Chợ Trời",
  CHAT_CONNECT: "Kết Nối Thiên Cơ",
  CHAT_PERCEIVE: "Cảm Ứng Thiên Cơ",
  IGNORE_MOVEMENT_RANGE: "Vượt Giới Hạn Di Chuyển",
  LOOT_PREVIEW: "Thấu Thị Chiến Lợi Phẩm",
  POEM_INTENT: "Thi Ý",
  DOUBLE_DAMAGE_DEALT: "Nhân Đôi Sát Thương",
  MANA_REGEN_TICK: "Hồi Chân Nguyên Định Kỳ",
  THOUGHT_REGEN_TICK: "Hồi Niệm Lực Định Kỳ",
  SOUL_REGEN_TICK: "Hồi Hồn Phách Định Kỳ",
  GENERATE_STONES_TICK: "Ngưng Tụ Nguyên Thạch",
  GENERATE_IMMORTAL_STONES_TICK: "Ngưng Tụ Tiên Nguyên Thạch",
  ADD_LIFESPAN_YEARS: "Tăng Thọ Nguyên",
  RANK_PROGRESS: "Tiến Cảnh Giới",
  SACRIFICE_APTITUDE: "Tế Lễ Tư Chất",
  PORTAL_CREATE: "Tạo Cổng Dịch Chuyển",
  RANDOM_PORTAL: "Cổng Dịch Chuyển Ngẫu Nhiên",
  CAPTURE_BONUS: "Tăng Tỷ Lệ Thu Phục",
  DEDUCTION_INSTANT_FINISH: "Đốn Ngộ Thôi Diễn Bí Phương",
  REVERSE_CALAMITY: "Nghịch Chuyển Càn Khôn",
  SUPPRESS_PRIORITY: "Cấm Tuyệt Tiên Thủ",
  GROWTH_DURATION_HOURS: "Thời Gian Trưởng Thành Cổ",
  IMMUNE_PATH_STATUS: "Miễn Nhiễm Trạng Thái Lưu Phái",
  SOVEREIGN_BODY_TRANSFORM: "Cải Tạo Tiên Thể Chí Tôn",
  LETHAL_LUCK_WARD: "Cẩu Thỉ Vận (Miễn Tử & Khiên)",
  LUCK_ROULETTE: "Cát Hung Chi Biến (Xúc Xắc Vận)",
  DREAM_UNRAVEL: "Giải Mộng (Thanh Tẩy Khống Chế)",
  CALCULATED_STRIKE: "Toán Kế Phá Trận (Tất Trúng)",
  SOUL_SHOCK: "Đãng Hồn Kích (Xuyên Khiên)",
  SOUL_DEVOUR: "Phệ Hồn Hóa Lực (Hút Hồn Tăng Công)",
  BEAST_PHANTOM_STRIKE: "Thú Lực Hư Ảnh (Bồi Kích)",
  BERSERK_STATE: "Cuồng Man Bá Thể (Máu Thấp Tăng Công)",
  HEAVY_LIFESTEAL: "Huyết Thao Thiết (Siêu Hút Máu)",
  BLOOD_SACRIFICE: "Huyết Tế Cuồng Bạo (Hiến Máu x2 Sát Thương)",
  BLEED_BURST: "Huyết Bạo Trùng Điệp (Kích Nổ Xuất Huyết)",
  MANA_STEAL: "Thâu Chân Nguyên (Trộm Chân Nguyên Địch)",
  BUFF_STEAL: "Thâu Thiên Hoán Nhật (Trộm Buff Địch)",
  TIME_REWIND: "Quang Âm Đảo Lưu (Đảo Ngược Thời Gian)",
  PACK_FRENZY: "Quân Đoàn Hợp Kích (Bầy Đàn Tăng Công)",
  FORM_BEAR: "Hóa Thân Cự Hùng (+Thủ & Hùng Giáp)",
  FORM_EAGLE: "Hóa Thân Kim Điêu (+Tốc & Né)",
  DAO_ADAPTIVE_RESIST: "Vạn Vật Thích Ứng (Giảm Thương)",
  POISON_DETONATE: "Vạn Độc Bộc Phát (Kích Nổ Độc)",
  SWORD_ARMOR_PIERCE: "Vô Kiên Bất Tồi (Xuyên Giáp)",
  EXECUTE_STRIKE: "Trảm Diệt Đao Ý (Trảm Sát <35% HP)",
  BONE_SPIKE: "Cốt Thứ Xuyên Tâm (Xuất Huyết & Phá Giáp)",
  BONE_THORNS: "Bạch Cốt Gai Khôi (Phản Thương)",
  MANA_EAT: "Thôn Phệ Bổ Khiếu (Nuốt Chân Nguyên)",
  METAL_AEGIS: "Cương Thiết Kim Thân (Kim Giáp)",
  ARMOR_BREAK_HEAVY: "Phá Giáp Kim Mang (Phá 40% Giáp)",
  WOOD_REGEN_BURST: "Sinh Sinh Bất Tức (Hồi Máu & Chân Nguyên)",
  WOOD_BIND: "Mộc Lao Trói Buộc (Trói Chân Khóa Né)",
  WATER_CLEANSE: "Thủy Lưu Triều Tịch (Thanh Tẩy Toàn Bộ Debuff)",
  WATER_MIRROR_SHIELD: "Thủy Kính Hóa Giải (Khiên & Phản Chấn)",
  FIRE_INFERNO: "Liệt Hỏa Liêu Nguyên (Biển Lửa Thiêu Đốt)",
  IGNITE_BURST: "Nộ Hỏa Bạo Phát (Kích Nổ Thiêu Đốt)",
  EARTH_SHIELD: "Đại Địa Hộ Thuẫn (Thổ Nham Giáp)",
  EARTH_GRAVITY: "Địa Chấn Trọng Lực (Làm Chậm Cực Đại)",
  WIND_HASTE_BURST: "Tật Phong Tốc Độ (Bộc Phá Tốc & Né)",
  LIGHTNING_STUN_BURST: "Lôi Đình Vạn Quân (Choáng Váng)",
  CHAIN_LIGHTNING: "Lôi Điện Truyền Dẫn (Sét Dây Chuyền)",
  ICE_DEEP_FREEZE: "Băng Phong Vạn Lý (Đóng Băng Mất Lượt)",
  ICE_FROSTBITE: "Hàn Băng Tê Cóng (Cấu Rỉa & Chậm Tốc)",
  LIGHT_BLINDING_FLASH: "Manh Mục Cực Quang (Gây Mù 60%)",
  LIGHT_PURIFY: "Thánh Quang Phục Thể (Hồi Máu & Giải Khống Chế)",
  DARK_STEALTH: "Ám Dạ Ẩn Nấp (Tàng Hình & Crit)",
  DARK_CURSE_WEAK: "Ám Ảnh Hủ Thực (Nguyền Rủa Suy Yếu)",
  STAR_FIRE_RAIN: "Tinh Quang Lạc Trần (Mưa Sao Băng & Khiên)",
  STAR_THOUGHT_RESTORE: "Tinh Niệm Tụ Hồn (Hồi Phục Chân Nguyên)",
  SOUND_DISRUPT_SILENCE: "Ma Âm Quán Nhĩ (Câm Lặng 2 Lượt)",
  SOUND_RESONANCE: "Ba Động Chấn Thần (Xuyên Khiên)",
  CLOUD_MIST: "Vân Vụ Ảo Mộng (+Né & Giảm Thương)",
  HEAVEN_SUPPRESSION: "Thiên Ý Khó Dò (Xóa Buff Địch)",
  TIEP_THIEN_DIA: "Tiếp Thiên Tiếp Địa (Giải Khống Chế & Hồi Phục)",
  HUMAN_WILL: "Nhân Định Thắng Thiên (Máu Thấp Bộc Phá)",
  VAN_DAN_DONG_TAM: "Vạn Dân Đồng Tâm (Khiên Nhân Đạo)",
  QI_WALL: "Khí Tường Hộ Thể (Nguyên Khí Giáp)",
  QI_EXPLOSION: "Khí Bạo Trảm Thiên (+40% Sát Thương)",
  SPATIAL_SHIFT: "Hư Không Thuấn Di (Né Tránh 75%)",
  DIMENSION_SLASH: "Thứ Nguyên Trảm (Xuyên Phòng Tuyến)",
  RULE_IMPOSITION: "Định Quy Đặt Củ (Xóa Toàn Bộ Buff Địch)",
  JUSTICE_VERDICT: "Phán Quyết Công Lý (+35% Sát Thương)",
  FORMATION_DOMAIN: "Trận Địa Khống Chế (Lĩnh Vực Trận Giáp)",
  BLINDING_MIRAGE: "Huyễn Ảnh Mê Trận (Mê Loạn & Mù)",
  PAINTING_IMPRISONMENT: "Họa Địa Vi Lao (Phong Ấn Vào Tranh)",
  PASSION_OVERLOAD: "Tình Cổ Vô Hối (Máu Thấp Bộc Phá)",
  GU_DETONATION: "Tự Bạo Cổ Trùng (Xung Kích Tàn Phá)",
  REVERSE_FLOW_SEAL: "Nghịch Lưu Hộ Thân Quỹ (Phản Phệ 100% Sát Thương)",
  GHOST_HAND_STEAL: "Đâu Thiên Thâu Thủ (Cướp 30% Chân Nguyên & Bóc Khiên)",
  LEAD_SOUL_TO_DREAM: "Dẫn Hồn Nhập Mộng (Sát Thương Hồn & Giam Cầm Mộng Cảnh)",
  MYRIAD_SELF_GIANT_HAND: "Vạn Ngã Đại Thủ Ấn (Đập Vỡ Khiên & Tăng 75% Uy Lực)",
  FAMILIAR_FACE: "Gặp Mặt Từng Quen Biết (Né Tránh 50% & Bạo Kích Tuyệt Đối)",
  FIVE_FINGER_SWORD: "Ngũ Chỉ Quyền Tâm Kiếm (Xuyên 50% Giáp & Tăng 65% Sát Thương)",
  TIME_CUTTER: "Tuế Nguyệt Như Đao (Chém Thọ Mệnh & Làm Chậm Tuyệt Đối)",
  SPRING_SCISSORS_SUMMER_FAN: "Xuân Tiễn Hạ Phiến (Phá 25% Giáp & Nổ Nhiệt Lượng)",
  DRAGON_BREATH_SWORD: "Long Tức Kiếm (Đóng Băng & Tăng 45% Sát Thương)",
  MYRIAD_YEAR_WARSHIP: "Vạn Niên Đấu Phi Thuyền (Toàn Đội +50% Tốc Độ & Bắn Phi Kiếm)",
  DRAGON_PALACE_DOMINION: "Long Cung Ngự Hồn (Khói Mộng Điệp Gây Mê Loạn)",
  DIVINE_BEAN_ABUNDANCE: "Thần Đậu Cung Sinh Cơ (Thanh Tẩy Toàn Đội & Hồi 10% HP)",
  EIGHT_EXTREMITIES_LUCK: "Bát Cực Quán Trấn Vận (Giảm 20% Bạo Kích & 25% Né Của Địch)",
  THUNDER_GOD_CITY: "Lôi Thần Thành Phạt Tội (Oanh Kích Địch Máu Yếu Nhất)",
  FIVE_REGIONAL_BOUNDARY: "Ngũ Giới Phân Tranh Trận (Áp Chế Giới Vực Bích & Tự Tổn)",
  NINETY_NINE_ARRAY: "Cửu Cửu Liên Hoàn Trận (Hấp Thụ 50% Sát Thương Nhận Vào)",
  SOUL_CONCEALMENT_ARRAY: "Yểm Hồn Ma Trận (Toàn Đội Ẩn Thân & +35% Né Tránh)",
  SIX_EXTREMITIES_ARRAY: "Lục Cực Tuyệt Sát Trận (Trận Nhãn Sát Phạt Ramping Mỗi Hiệp)",
  SONG_OF_SEPARATION: "Ly Ca (Phá 2 Buff & Hủy 30% Giáp)",
  SONG_OF_DESTINY: "Vận Mệnh Ca (Buff Toàn Đội +40% Công, Giảm 30% Địch)",
  FAREWELL_FRIEND_WIND: "Tống Hữu Phong (Xuyên Giáp & Nguyền Rủa 4 Hiệp)",
  INFINITE_WIND: "Vô Hạn Phong (+60% Tốc, +35% Né & x1.75 Uy Lực)",
  BLOOD_STAINED_ROBE: "Huyết Nhiễm Chinh Bào (Chuyển 40% Sát Thương Thành Huyết Thuẫn)",
  BLOOD_DEMON_DISINTEGRATION: "Huyết Ma Giải Thể (Hy Sinh 20% Máu Bộc Phát x2.2 & 50% Hút Máu)",
  YAMA_EMPEROR: "Diêm Đế Quỷ Quan Y (+50% Thủ & Diêm La Tử Nuốt Hồn)",
  IMMEASURABLE_QI_SEA: "Vô Lượng Khí Hải (AoE Toàn Trường Giảm 30% Công & Tốc)",
  THREE_QI_RETURN: "Tam Khí Quy Lai (Hồi 30% HP, Đầy Chân Nguyên & Xuyên 50% Giáp)",
  THREE_MEN_MAKE_TIGER: "Tam Nhân Thành Hổ (Hung Hổ Áp Đặt Sợ Hãi & Suy Yếu)",
  WORDS_ARE_FRIGHTENING: "Nhân Ngôn Khả Úy (Sát Thương Tăng Theo Debuff & Câm Lặng)",
  FATE_VANQUISHED: "Mệnh Bại (Phá Vỡ Bất Tử, Bất Hoại & Xé Nát Giáp Ảo)",
  HEAVEN_NET_RESTRICTION: "Lưới Trời Tuy Thưa (Khóa Né Tránh & Làm Chậm 40%)",
  FUTURE_SELF: "Vị Lai Thân (Mượn Sức Mạnh Tương Lai +80% Công & +50% Tốc)",
  LANDSCAPE_AS_BEFORE: "Giang Sơn Như Cố (Khôi Phục 40% Giáp & Tẩy Toàn Bộ Debuff)",
  EATING_LOSS_BLESSING: "Cật Khuy Thị Phúc (Tích Tầng Chịu Đòn Quy Đổi Sát Thương & Hồi Máu)",
  MYRIAD_DISMANTLE: "Vạn Giải (Xóa Sạch Mọi Buff, Hủy Khiên & Xuyên 100% Giáp)",
  YOU_DU_STRENGTH_TIGER: "Đồ Đằng U Đô Lực Bưu (Sát Thương Khủng, Xuất Huyết Nặng & Choáng)",
  NIRVANA_FIRE: "Niết Bàn Hỏa (Dục Hỏa Trùng Sinh Hồi Sinh 40% HP & Nổ Lửa Toàn Sàn)",
  SWORD_WAVE_TRIPLE: "Kiếm Lãng Tam Điệp (Ba Đợt Sóng Kiếm x1.8 & Xuất Huyết)",
  SWORD_MARK_DEATH: "Kiếm Ngân Tác Mệnh (Khắc Kiếm Ngân & Trảm Sát Dưới 25% HP)",
  SOUL_EXPLOSION_DANG_HUN: "Hồn Bạo Đãng Hồn Sơn (Nổ Sát Thương Hồn & Choáng Toàn Sàn)",
  SOUL_BEAST_SUMMON: "Hồn Thú Triệu Lai (Thái Cổ Hồn Thú Hộ Vệ +35% Khiên & +30% Công)",
  STEAL_LIFE: "Thâu Sinh (Hút Thẳng 15% MaxHP Kẻ Địch Hồi Cho Bản Thân)",
  CHAOS_CAVE_DRILL: "Hỗn Động Toản (Mũi Khoan Không Gian Hủy Khiên & Xuyên 100% Giáp)",
  NO_EXTREME_SPACE: "Vô Gian Đạo (Hòa Tan Không Gian Vô Tận +50% Né Tránh)",
  LIGHT_TIME_FLYING_BLADE: "Quang Âm Phi Nhận (Tước 30% Chân Nguyên & Làm Chậm 50%)",
  ALL_PEOPLE_UNITED: "Vạn Chúng Nhất Tâm (Khiên Nhân Đạo 30% HP & +35% Sát Thương)",
  HERO_AMONG_MEN: "Nhân Trung Hào Kiệt (Máu Càng Thấp Sát Thương Càng Cao)",
  QI_JUE_RETURN_LIFE: "Khí Tuyệt Phùng Sinh (Máu Dưới 30% Hồi 40% HP & Thanh Tẩy)",
  RAGING_ANGRY_BIRD: "Phẫn Nộ Đích Tiểu Điểu (Hỏa Điểu Phát Nổ & Thiêu Đốt Nặng)",
  EXTREME_ICE_CRYSTAL_COFFIN: "Cực Băng Trấn Tiên Quan (Đóng Băng & Câm Lặng 2 Hiệp)",
  THIRTY_THREE_HEAVEN_LIGHT: "Tam Thập Tam Thiên Quang (Thanh Tẩy Toàn Đội & Mù Lòa Phe Địch)",
  GREAT_NIGHT: "Đại Dạ Di Thiên (Toàn Đội Ẩn Thân, +50% Bạo Kích & Giảm Né Địch)",
  DRY_WOOD_SPRING: "Khô Mộc Phùng Xuân (Tái Sinh Liên Tục Hồi 15% MaxHP Mỗi Hiệp)",
  IMMORTAL_BINDING_VINE: "Khổn Tiên Đằng (Trói Chặt 2 Hiệp & Hút 20% Chân Nguyên)",
  BLOOD_GOD_CHILD: "Huyết Thần Tử (Huyết Thần Tử Trợ Chiến +50% Sát Thương & 40% Hút Máu)",
  TOTEM_KUN_PENG: "Đồ Đằng Côn Bằng (Cuồng Phong Bạt Sơn +75% Sát Thương & +50% Tốc)",
  TOTEM_GOLDEN_BELL: "Đồ Đằng Kim Chung (Phản Phệ 35% Sát Thương & Kháng Bạo Kích)",
  REFINE_SELF: "Luyện Kỷ (Lấy Thân Làm Phôi Tăng Vĩnh Viễn 20% Công Thủ)",
  MYRIAD_STAR_FIREFLY: "Vạn Tinh Phi Huỳnh (Đom Đóm Tinh Tú Thiêu Rụi 40% Chân Nguyên)",
  FIVE_RESTRICTION_DEEP_LIGHT: "Ngũ Cấm Huyền Quang Khí (Khóa Ngũ Hành Phong Ấn Cổ Trùng 3 Hiệp)",
  CHARM_GHOST_COME: "Tử Quỷ Khoái Lai (Quyến Rũ Mê Hoặc Thần Hồn Điên Đảo 2 Hiệp)",
  DEVOUR_HEAVEN_EARTH: "Thôn Thiên Địa (Cắn Nuốt 50% Khiên Kẻ Địch Chuyển Hóa Thành HP)",
  ADD_SOUL_MAX: "Tăng Giới Hạn Hồn Phách",
  ADD_THOUGHT_MAX: "Tăng Giới Hạn Tạp Niệm",
  CONCEAL_GU: "Ẩn Giấu Khí Tức Cổ Trùng",
  DECREASE_TIME_SPEED: "Làm Chậm Dòng Thời Gian Khiếu",
  GUARANTEED_HIT: "Tất Trúng (Không Thể Né)",
  IGNORE_MONSTER_ENCOUNTER: "Tránh Né Dị Thú Hoang Dã",
  INCREASE_TIME_SPEED: "Gia Tốc Dòng Thời Gian Khiếu",
  KNOCKBACK: "Đẩy Lùi Mục Tiêu",
  MAN_BEAST_BURIAL: "Nhân Thú Táng (Huyết Tế Căn Cốt)",
  PERM_STEAL_GU: "Đoạt Cổ Vĩnh Viễn",
  POISON_VOW: "Độc Thệ (Lời Thề Độc)",
  PULL_TARGET: "Kéo Mục Tiêu Lên Tiền Tuyến",
  RECORD_MESSAGE: "Lưu Giữ Thần Niệm / Tin Nhắn",
  REDUCE_STRESS: "Giải Tỏa Áp Lực Tiên Khiếu",
  RESTORE_MENTAL: "Hồi Phục Tinh Thần",
  RESTORE_SOUL_HP: "Hồi Phục Sinh Lực Hồn Phách",
  RETREAT_SELF: "Rút Lui Về Tuyến Sau",
  SAT_KHI_GUARANTEE: "Sát Khí Tuyệt Sát",
  SEVER_CLUES: "Cắt Đứt Thiên Cơ Manh Mối",
  SPRING_EGG_HATCH: "Ấp Trứng Khai Sinh Nguyên Tuyền",
  STONE_APERTURE: "Thạch Khiếu (Hóa Đá Không Khiếu)",
  SUMMON_MONSTER: "Triệu Hồi Dị Thú Trợ Chiến",
  SWAP_TARGET: "Hoán Đổi Vị Trí Mục Tiêu",
  TEMP_STEAL_GU: "Tước Đoạt Cổ Trùng Tạm Thời",
  THIRD_WATCH: "Tam Canh (Ngừng Đọng Thời Gian)",
};

const STATUS_BEHAVIORS: Record<string, string> = {};
if (typeof COMBAT_STATUSES !== "undefined" && Array.isArray(COMBAT_STATUSES)) {
  for (const s of COMBAT_STATUSES) {
    if (!STATUS_NAMES[s.id]) STATUS_NAMES[s.id] = s.label;
    if (s.behavior) STATUS_BEHAVIORS[s.id] = s.behavior;
  }
}

const HOOK_BEHAVIORS: Record<string, string> = {};
if (typeof NATIVE_HOOKS !== "undefined" && Array.isArray(NATIVE_HOOKS)) {
  for (const h of NATIVE_HOOKS) {
    if (!UNIQUE_NAMES[h.code]) UNIQUE_NAMES[h.code] = h.label;
    if (h.behavior) HOOK_BEHAVIORS[h.code] = h.behavior;
  }
}

export const formatTarget = formatEffectTarget;

export function formatOperation(op: EffectOperation): { text: string; color: string; icon: string; tooltip?: string } {
  const opType = (op.op || "").toUpperCase();
  switch (opType) {
    case "BONUS_STRIKE":
      return {
        text: `Đòn phụ ${Math.round((op.chance ?? 0) * 100)}% · ×${op.value ?? 0} (sàn ${op.min_damage ?? 0})`,
        color: "text-amber-300",
        icon: "🐾",
        tooltip: "Có xác suất tung đòn sát thương trực tiếp theo action; không đi qua lớp khiên và phòng thủ.",
      };
    case "CHAIN_DAMAGE":
      return {
        text: `Sát thương lan ×${op.value ?? 0} · ${op.target_count ?? 0} mục tiêu phụ (sàn ${op.min_damage ?? 0})`,
        color: "text-amber-300",
        icon: "⚡",
        tooltip: "Gây direct HP theo tỷ lệ sát thương action hiện tại lên các địch còn sống tiếp theo, bỏ qua khiên.",
      };
    case "DAMAGE": {
      const pathStr = op.path ? ` [${translatePath(op.path)}]` : "";
      const tooltip = "Gây sát thương lên mục tiêu (bị giảm bởi phòng thủ, khiên và chênh lệch cảnh giới).";
      if (op.sub_type === "MULTIPLIER") {
        return { text: `Sát thương ×${op.value}${pathStr}`, color: "text-rose-300", icon: "⚔️", tooltip };
      }
      return { text: `Sát thương ${op.value ?? 0}${pathStr}`, color: "text-rose-300", icon: "⚔️", tooltip };
    }
    case "TARGET_DAMAGE_MODIFIER":
      return {
        text: `Giảm ${Math.round((op.value ?? 0) * 100)}% sát thương mục tiêu (sàn ×${op.min_multiplier ?? 0})`,
        color: "text-amber-300",
        icon: "📉",
        tooltip: "Giảm DamageMultiplier của mục tiêu đến hết trận; không phải status nên không tự hết hạn hoặc bị cleanse.",
      };
    case "DEFENSE_PENETRATION":
      return {
        text: `Xuyên ${Math.round((op.value ?? 0) * 100)}% phòng thủ`,
        color: "text-amber-300",
        icon: "🗡️",
        tooltip: "Bỏ qua tỷ lệ phòng thủ tương ứng của mục tiêu trong đòn đánh.",
      };
    case "SHIELD": {
      const tooltip = "Tạo lớp khiên ảo hấp thụ sát thương trước khi trừ vào Sinh Lực.";
      if (op.sub_type === "MULTIPLIER") {
        return { text: `Khiên ×${op.value}`, color: "text-sky-300", icon: "🛡️", tooltip };
      }
      return { text: `Khiên ảo +${op.value ?? 0}`, color: "text-sky-300", icon: "🛡️", tooltip };
    }
    case "HEAL": {
      if (op.sub_type === "RESOURCE" && op.resource?.toUpperCase() === "MANA") {
        const amount = op.unit === "PERCENT" ? `${op.value ?? 0}% Max Chân Nguyên` : `${op.value ?? 0} Chân Nguyên`;
        return { text: `Hồi ${amount}`, color: "text-cyan-300", icon: "💧", tooltip: "Hồi Chân Nguyên cho mục tiêu đã chọn; phần trăm được tính theo Max Chân Nguyên của từng mục tiêu." };
      }
      if (op.sub_type === "MANA") {
        return { text: `Hồi ${op.value ?? 0} Chân Nguyên`, color: "text-cyan-300", icon: "💧", tooltip: "Hồi phục Chân Nguyên cho Không Khiếu." };
      }
      if (op.sub_type === "LIFESTEAL") {
        const value = op.value ?? 0;
        const percent = value > 1 ? value : value * 100;
        return { text: `Hút máu ${Math.round(percent)}%`, color: "text-emerald-300", icon: "🩸", tooltip: "Hút máu chuyển hóa phần trăm sát thương gây ra thành sinh lực." };
      }
      return { text: `Hồi +${op.value ?? 0} HP`, color: "text-emerald-300", icon: "💚", tooltip: "Hồi phục Sinh Lực cho bản thân." };
    }
    case "NOTIFY":
      return { text: op.message || "Ghi log chiến đấu", color: "text-zinc-300", icon: "🗨️", tooltip: "Ghi thông điệp cấu hình vào nhật ký chiến đấu." };
    case "TENACITY_MODIFIER":
      return { text: `Kiên Cường +${op.value ?? 0} + ${op.per_missing_hp ?? 0} × HP đã mất`, color: "text-amber-300", icon: "🛡️", tooltip: "Passive combat cộng Kiên Cường theo tỷ lệ HP đã mất của chủ thể." };
    case "STATUS_IMMUNITY": {
      const status = op.sub_type === "FREEZE" ? "Đóng Băng" : "Choáng";
      return { text: `Miễn nhiễm ${status} theo điều kiện`, color: "text-indigo-300", icon: "✨", tooltip: "Khi block condition đúng, trạng thái tương ứng không khóa hành động." };
    }
    case "SLOT_INTERACTION":
      return {
        text: op.sub_type === "TEMPORARY_STEAL" ? "Đoạt Cổ Tạm Thời (Tước Cổ Trong Trận)" : "Đoạt Cổ Vĩnh Viễn (Chiếm Cổ Sau Trận)",
        color: "text-amber-300 font-semibold",
        icon: "🪝",
        tooltip: "Tước đoạt hoặc chiếm giữ Cổ trùng đối phương theo cơ chế PvP.",
      };
    case "REVIVE":
      return { text: "Hồi sinh khi chịu đòn chí tử", color: "text-rose-300", icon: "🔁", tooltip: "Tự hồi sinh theo chance, mode và giới hạn sử dụng trong params; xử lý tại ON_DEATH." };
    case "EFFECT_IMMUNITY":
      return { text: "Miễn nhiễm hiệu ứng thù địch", color: "text-indigo-300", icon: "↩️", tooltip: "Miễn mọi hiệu ứng từ địch; sát thương thô vẫn áp dụng." };
    case "RANK_SCALED_DEBUFF":
      return { text: "Áp debuff theo rank mục tiêu", color: "text-amber-300", icon: "📉", tooltip: "Cường độ theo công thức Regional Wall và hệ số strength_factor; có thể áp trạng thái PvP theo thời lượng rank." };
    case "GU_SEAL":
      return { text: "Phong ấn Cổ phản ứng", color: "text-indigo-300", icon: "🔒", tooltip: "Phản ứng trước đòn địch; params cấu hình xác suất, thời lượng, giới hạn kích hoạt và phản phệ." };
    case "DIRECT_HP": {
      const tooltip = "Sát thương thực đánh thẳng vào Sinh Lực, bỏ qua 100% phòng thủ và xuyên khiên ảo.";
      if (op.sub_type === "MAX_HP_PERCENT") {
        return { text: `Sát thương trực tiếp ${Math.round((op.value ?? 0) * 100)}% Max HP (xuyên khiên)`, color: "text-purple-300 font-semibold", icon: "⚡", tooltip };
      }
      if (op.sub_type === "SHIELD_STRIP") {
        return { text: `Hủy ${Math.round((op.value ?? 0) * 100)}% khiên mục tiêu`, color: "text-amber-300 font-semibold", icon: "💥", tooltip: "Bóc đi phần trăm khiên ảo hiện có của mục tiêu." };
      }
      return { text: `Sát thương trực tiếp vào máu`, color: "text-purple-300", icon: "⚡", tooltip };
    }
    case "HP_COST": {
      const percent = Math.round((op.value ?? 0) * 100);
      const minimum = op.min_cost ? ` (tối thiểu ${op.min_cost})` : "";
      const floor = op.min_remaining ? `, giữ lại ${op.min_remaining} HP` : "";
      return {
        text: `Trả ${percent}% HP hiện tại${minimum}${floor}`,
        color: "text-rose-300",
        icon: "🩸",
        tooltip: "Trừ HP người thi triển theo HP hiện tại, áp dụng mức trừ tối thiểu và giữ lại ngưỡng HP đã cấu hình.",
      };
    }
    case "TRANSFER_RESOURCE": {
      if (op.resource?.toUpperCase() === "HP" && op.sub_type === "PERCENT_OF_TARGET_SHIELD") {
        return {
          text: `Hấp thụ ${Math.round((op.value ?? 0) * 100)}% Khiên mục tiêu, hồi HP cho bản thân`,
          color: "text-emerald-300",
          icon: "🛡️",
          tooltip: "Hấp thụ tỷ lệ Khiên Ảo hiện tại của mục tiêu và hồi cùng lượng HP cho người thi triển.",
        };
      }
      if (op.resource?.toUpperCase() === "HP" && op.sub_type === "PERCENT_OF_MAX_HP") {
        const minimum = op.min_amount ? ` (tối thiểu ${op.min_amount})` : "";
        return {
          text: `Rút ${Math.round((op.value ?? 0) * 100)}% Max HP${minimum}, hồi HP cho bản thân`,
          color: "text-emerald-300",
          icon: "🩸",
          tooltip: "Trừ trực tiếp HP mục tiêu theo Max HP đã cấu hình và hồi cùng lượng HP cho người thi triển.",
        };
      }
      if (op.resource?.toUpperCase() === "MANA" && op.sub_type === "PERCENT_OF_TARGET") {
        return {
          text: `Chuyển ${Math.round((op.value ?? 0) * 100)}% Chân Nguyên mục tiêu`,
          color: "text-cyan-300",
          icon: "💧",
          tooltip: "Chuyển tỷ lệ Chân Nguyên hiện có từ mục tiêu sang người thi triển.",
        };
      }
      return { text: `Chuyển tài nguyên ${op.resource || ""}`, color: "text-cyan-300", icon: "💧" };
    }
    case "RESOURCE_CONVERSION": {
      if (op.sub_type === "DISPEL_TO_RESOURCES") {
        return {
          text: "Giải Mộng (thanh tẩy debuff & chuyển hóa thành Chân Nguyên, Khiên)",
          color: "text-cyan-300",
          icon: "🔮",
          tooltip: "Gỡ bỏ các hiệu ứng khống chế và đổi số lượng hiệu ứng gỡ được thành Chân Nguyên và Mộng Giáp.",
        };
      }
      if (op.sub_type === "CURRENT_HP_TO_SHIELD") {
        const percent = Math.round((op.value ?? 0) * 100);
        return {
          text: `Chuyển ${percent}% HP hiện tại thành Khiên`,
          color: "text-sky-300",
          icon: "🛡️",
          tooltip: "Tiêu hao một phần HP hiện tại để chuyển hóa thành Khiên Ảo.",
        };
      }
      return { text: "Chuyển đổi tài nguyên", color: "text-cyan-300", icon: "🔄" };
    }
    case "TARGET_REDIRECTION":
      return {
        text: "Chuyển hướng đòn địch (Ý Niệm Hỗn Loạn)",
        color: "text-amber-300",
        icon: "🌀",
        tooltip: "Khi kẻ địch chuẩn bị tấn công, có tỷ lệ làm rối loạn mục tiêu khiến địch đánh hụt hoặc tự đánh chính mình.",
      };
    case "OUTCOME_ROULETTE": {
      if (op.sub_type === "CHAOS") {
        return {
          text: "Hỗn Độn Đánh Cược (chọn ngẫu nhiên hiệu ứng & trả giá)",
          color: "text-purple-300 font-semibold",
          icon: "🎲",
          tooltip: "Ngẫu nhiên kích hoạt hiệu ứng cực mạnh trong trận, sau trận gánh chịu cái giá vĩnh viễn.",
        };
      }
      return {
        text: "Cát Hung Chi Biến (vòng quay vận khí)",
        color: "text-yellow-300",
        icon: "🍀",
        tooltip: "Rút thăm ngẫu nhiên kết quả cát hung theo bảng tỷ lệ để định đoạt sát thương và hiệu ứng phụ.",
      };
    }
    case "HIT_RULE": {
      if (op.sub_type === "LETHAL_LUCK_WARD") {
        return { text: "Cẩu Thỉ Vận (miễn tử & nhận khiên)", color: "text-yellow-300", icon: "🍀", tooltip: "Khi nhận đòn chí tử, có xác suất giữ lại 1 HP và nhận khiên bảo mệnh." };
      }
      if (op.sub_type === "GUARANTEED") {
        return { text: "Tất Trúng (không thể né)", color: "text-amber-300", icon: "🎯", tooltip: "Đòn đánh chắc chắn trúng mục tiêu, không thể né tránh." };
      }
      if (op.sub_type === "COUNTER_THRESHOLD") {
        const paramKey = op.params?.key != null ? String(op.params.key) : "";
        const keyName = UNIQUE_NAMES[paramKey.toUpperCase()] || paramKey || "tầng đếm";
        const thresh = typeof op.params?.threshold === "number" ? op.params.threshold : 0;
        return {
          text: `Tích lũy [${keyName}] ≥ ${thresh}: Tất Trúng & Xuyên Khiên`,
          color: "text-amber-300 font-semibold",
          icon: "🎯",
          tooltip: "Khi số tầng tích lũy đạt ngưỡng, đòn đánh chắc chắn trúng đích và xuyên qua mọi khiên/miễn nhiễm.",
        };
      }
      if (op.sub_type === "MULTI_HIT") {
        const count = op.params?.count || op.value || 2;
        return {
          text: `Liên Hoàn Kích (${count} lần đánh liên tiếp)`,
          color: "text-orange-400 font-semibold",
          icon: "⚡",
          tooltip: "Tung ra nhiều đợt tấn công liên tiếp trong cùng một lượt hành động.",
        };
      }
      if (op.sub_type === "FIRST_TURN_EVADE") {
        return {
          text: "Tuyệt Đối Né Tránh (Lượt 1)",
          color: "text-indigo-300 font-semibold",
          icon: "💨",
          tooltip: "Hoàn toàn né tránh mọi đòn tấn công trong lượt thi đấu đầu tiên.",
        };
      }
      return { text: `Quy tắc tấn công: ${op.sub_type || ""}`, color: "text-amber-300", icon: "🎯" };
    }
    case "STAT_MOD": {
      const statName = STAT_NAMES[op.stat?.toUpperCase() || ""] || op.stat || "Chỉ số";
      const isMult = op.sub_type === "MULTIPLY_PERCENT";
      const val = op.value ?? 0;
      const isPathConflictReduction = op.stat?.toUpperCase() === "PATH_CONFLICT_REDUCTION";
      const text = isPathConflictReduction
        ? `${val >= 0 ? "+" : ""}${val}% ${statName}`
        : isMult
        ? `${val >= 0 ? "+" : ""}${Math.round(val * 100)}% ${statName}`
        : `${val >= 0 ? "+" : ""}${val} ${statName}`;
      const tooltip = "Cộng dồn vĩnh viễn vào thuộc tính khi Cổ đang trang bị hoặc thể chất/chủng tộc sở hữu.";
      return { text, color: val >= 0 ? "text-amber-300" : "text-rose-400", icon: "📊", tooltip };
    }
    case "STATUS": {
      const sId = (op.status_id || "").toUpperCase();
      const sName = STATUS_NAMES[sId] || sId;
      const parts: string[] = [sName];
      if (op.value && op.value > 0) parts.push(`[${op.value}]`);
      if (op.turns && op.turns > 0) parts.push(`(${op.turns} lượt)`);
      if (op.chance && op.chance > 0 && op.chance < 1) parts.push(`[${Math.round(op.chance * 100)}%]`);
      const tooltip = STATUS_BEHAVIORS[sId];
      return { text: `Trạng thái: ${parts.join(" ")}`, color: "text-indigo-300", icon: "✨", tooltip };
    }
    case "BARRIER": {
      const turnsStr = op.turns ? ` (${op.turns} lượt)` : "";
      const tooltip = "Lớp kết giới bảo hộ toàn diện trong số lượt quy định.";
      return { text: `Bảo hộ: ${op.sub_type || "Bất tử"}${turnsStr}`, color: "text-yellow-300 font-semibold", icon: "🌟", tooltip };
    }
    case "CLEANSE": {
      const tooltip = "Hóa giải và thanh tẩy các trạng thái bất lợi / khống chế.";
      return { text: `Thanh tẩy: ${op.sub_type || "Giải trừ hiệu ứng"}`, color: "text-cyan-300", icon: "🍃", tooltip };
    }
    case "REVOKE_STATUS": {
      const statusID = op.status_id || "trạng thái";
      return { text: `Gỡ trạng thái người chơi: ${statusID}`, color: "text-cyan-300", icon: "🍃", tooltip: "Xóa một persistent player status khi bắt đầu trận, trước khi tính chỉ số." };
    }
    case "UNIQUE": {
      const code = (op.code || "").toUpperCase();
      const val = op.params?.value !== undefined ? op.params.value : op.value;
      const tooltip = HOOK_BEHAVIORS[code];

      if (code === "STAT_MODS" && val && typeof val === "object") {
        const parts = Object.entries(val as Record<string, number>).map(([k, v]) => {
          const sName = STAT_NAMES[k.toUpperCase()] || k;
          return `${sName} ${v >= 0 ? "+" : ""}${Math.round(v * 100)}%`;
        });
        const isPositive = Object.values(val as Record<string, number>).every((v) => v >= 0);
        return { text: parts.join(", "), color: isPositive ? "text-amber-300" : "text-rose-400", icon: "📊", tooltip };
      }
      if (code === "DAO_MARK_MODIFIER" && val && typeof val === "object") {
        const parts = Object.entries(val as Record<string, number>).map(([path, v]) => {
          return `Đạo Ngân [${translatePath(path)}] ${v >= 0 ? "+" : ""}${Math.round(v * 100)}%`;
        });
        return { text: parts.join(", "), color: "text-indigo-300", icon: "⚡", tooltip };
      }
      if (code === "PATH_DAMAGE_MODIFIER" && val && typeof val === "object") {
        const parts = Object.entries(val as Record<string, number>).map(([path, v]) => {
          return `Sát Thương [${translatePath(path)}] ${v >= 0 ? "+" : ""}${Math.round(v * 100)}%`;
        });
        return { text: parts.join(", "), color: "text-rose-300", icon: "⚔️", tooltip };
      }
      if (code === "PATH_RESISTANCE_MODIFIER" && val && typeof val === "object") {
        const parts = Object.entries(val as Record<string, number>).map(([path, v]) => {
          return `Kháng [${translatePath(path)}] ${v >= 0 ? "+" : ""}${Math.round(v * 100)}%`;
        });
        return { text: parts.join(", "), color: "text-sky-300", icon: "🛡️", tooltip };
      }
      if (code === "MANA_REGEN") {
        const num = Number(val) || 0;
        return { text: `Hồi Chân Nguyên ${num >= 0 ? "+" : ""}${num}`, color: "text-cyan-300", icon: "💧", tooltip };
      }
      if (code === "MAX_MANA") {
        const num = Number(val) || 0;
        return { text: `Chân Nguyên Tối Đa +${Math.round(num * 100)}%`, color: "text-cyan-300", icon: "💧", tooltip };
      }
      if (code === "HEALING_RECEIVED") {
        const num = Number(val) || 0;
        return { text: `Hiệu Quả Hồi Máu ${num >= 0 ? "+" : ""}${Math.round(num * 100)}%`, color: num >= 0 ? "text-emerald-300" : "text-rose-400", icon: "💚", tooltip };
      }
      if (code === "HP_REGEN") {
        const num = Number(val) || 0;
        return { text: `Hồi Sinh Lực ${num >= 0 ? "+" : ""}${num}/giờ`, color: "text-emerald-300", icon: "💚", tooltip };
      }
      if (code === "REFINE_SUCCESS") {
        const num = Number(val) || 0;
        return { text: `Tỷ Lệ Luyện Cổ ${num >= 0 ? "+" : ""}${Math.round(num * 100)}%`, color: num >= 0 ? "text-amber-300" : "text-rose-400", icon: "🧪", tooltip };
      }
      if (code === "LUCK") {
        const num = Number(val) || 0;
        return { text: `Vận Khí ${num >= 0 ? "+" : ""}${num}`, color: "text-yellow-300", icon: "🍀", tooltip };
      }
      if (code === "CULTIVATION_EFFICIENCY") {
        const num = Number(val) || 0;
        return { text: `Hiệu Suất Tu Luyện ${num >= 0 ? "+" : ""}${Math.round(num * 100)}%`, color: num >= 0 ? "text-purple-300" : "text-rose-400", icon: "📈", tooltip };
      }
      if (code === "DAO_MARK_GAIN") {
        const num = Number(val) || 0;
        return { text: `Thu Hoạch Đạo Ngân +${Math.round(num * 100)}%`, color: "text-violet-300", icon: "✨", tooltip };
      }
      if (code === "MENTAL_RESISTANCE") {
        const num = Number(val) || 0;
        return { text: `Kháng Tâm Trí / Khống Chế +${Math.round(num * 100)}%`, color: "text-teal-300", icon: "🧘", tooltip };
      }
      if (code === "SOUL_DEFENSE") {
        const num = Number(val) || 0;
        return { text: `Phòng Ngự Hồn Phách +${Math.round(num * 100)}%`, color: "text-indigo-300", icon: "👻", tooltip };
      }
      if (code === "SOUL_REGEN") {
        const num = Number(val) || 0;
        return { text: `Dưỡng Hồn +${Math.round(num * 100)}%`, color: "text-indigo-300", icon: "👻", tooltip };
      }
      if (code === "TENACITY") {
        const num = Number(val) || 0;
        return { text: `Kiên Cường +${Math.round(num * 100)}%`, color: "text-blue-300", icon: "🛡️", tooltip };
      }
      if (code === "CC_RESIST") {
        const num = Number(val) || 0;
        return { text: `Kháng Khống Chế +${Math.round(num * 100)}%`, color: "text-blue-300", icon: "🛡️", tooltip };
      }
      if (code === "APERTURE_STRESS") {
        const num = Number(val) || 0;
        return { text: `Áp Lực Không Khiếu +${Math.round(num * 100)}%`, color: "text-red-300", icon: "💥", tooltip };
      }
      if (code === "NON_CONFLICTING_DAO_MARKS") {
        return { text: `Đạo Ngân Không Xung Đột`, color: "text-amber-300", icon: "☯️", tooltip };
      }
      if (code === "IMMORTAL") {
        return { text: `Thọ Nguyên Vô Tận (Bất Tử)`, color: "text-yellow-300 font-semibold", icon: "♾️", tooltip };
      }
      if (code === "DISABLE_HP_REGEN") {
        return { text: `Không Thể Hồi Sinh Lực Tự Nhiên`, color: "text-rose-400", icon: "🚫", tooltip };
      }
      if (code === "DISABLE_MANA_REGEN") {
        return { text: `Không Thể Hồi Chân Nguyên Tự Nhiên`, color: "text-rose-400", icon: "🚫", tooltip };
      }
      if (code === "DISABLE_RESOURCE_REGEN") {
        return { text: `Không Thể Hồi Tiên/Chân Nguyên`, color: "text-rose-400", icon: "🚫", tooltip };
      }
      if (code === "DISABLE_SOUL_REGEN") {
        return { text: `Không Thể Phục Hồi Hồn Phách`, color: "text-rose-400", icon: "🚫", tooltip };
      }
      if (code === "DEDUCTION_INSTANT_FINISH") {
        const p = op.params || {};
        const base = Number(p.base_chance ?? 0.05);
        const step = Number(p.per_rank_step ?? 0.02);
        return { text: `Đốn ngộ hoàn thành thôi diễn ngay (${Math.round(base * 100)}% + ${Math.round(step * 100)}%/rank chênh lệch)`, color: "text-amber-300", icon: "💡", tooltip };
      }
      if (code === "REVERSE_CALAMITY") {
        return { text: `Đảo nghịch mọi trạng thái bất lợi về hư vô (chỉ nhận sát thương)`, color: "text-slate-200 font-semibold", icon: "☯️", tooltip };
      }
      if (code === "SUPPRESS_PRIORITY") {
        return { text: `Trấn áp ưu tiên hành động của địch — trấn áp thành công thì đi trước Lượt 1`, color: "text-red-300 font-semibold", icon: "⛔", tooltip };
      }

      const codeName = UNIQUE_NAMES[code] || code;
      const paramStr = op.params && Object.keys(op.params).length > 0 ? ` ${JSON.stringify(op.params)}` : "";
      return { text: `${codeName}${paramStr}`, color: "text-fuchsia-300", icon: "🔮", tooltip };
    }
    case "END_BATTLE": {
      const summary = op.message ? op.message.replace(/\{actor\}/g, "Bản thân").replace(/\{target\}/g, "Mục tiêu") : "Đoạt Thắng (Toàn diệt địch nhân)";
      return {
        text: `Đoạt Thắng: ${summary}`,
        color: "text-amber-400 font-bold",
        icon: "👑",
        tooltip: "Kết thúc chiến đấu ngay lập tức — giành chiến thắng tuyệt đối trước toàn bộ quân địch.",
      };
    }
    case "GRANT_CAPABILITY": {
      const sub = (op.sub_type || "").toUpperCase();
      if (sub === "CAPTURE_BONUS") {
        const val = Number(op.params?.value ?? op.value ?? 0);
        return {
          text: `Thu Phục Sủng Vật +${Math.round(val * 100)}%`,
          color: "text-purple-300 font-semibold",
          icon: "🐾",
          tooltip: "Gia tăng đáng kể tỷ lệ bắt giữ và thuần phục dã thú / dị thú.",
        };
      }
      if (sub === "PORTAL_CREATE") {
        const range = Number(op.params?.range ?? op.value ?? 0);
        return {
          text: `Mở Cổng Truyền Tống${range > 0 ? ` (${range} ô)` : " (Vô hạn cự ly)"}`,
          color: "text-sky-300",
          icon: "🌀",
          tooltip: "Mở cổng không gian di chuyển đến các khu vực hoặc hub đã kích hoạt.",
        };
      }
      if (sub === "RANDOM_PORTAL") {
        const range = Number(op.params?.range ?? op.value ?? 0);
        return {
          text: `Truyền Tống Ngẫu Nhiên${range > 0 ? ` (${range} ô)` : ""}`,
          color: "text-sky-300",
          icon: "🎲",
          tooltip: "Dịch chuyển ngẫu nhiên đến một vị trí trong phạm vi bản đồ.",
        };
      }
      if (sub === "IGNORE_MOVEMENT_RANGE") {
        return {
          text: "Vượt Cự Ly Giới Hạn (Vô hạn di chuyển)",
          color: "text-sky-300 font-semibold",
          icon: "🚀",
          tooltip: "Bỏ qua mọi giới hạn khoảng cách khi di chuyển hoặc truyền tống qua giới vực.",
        };
      }
      const name = UNIQUE_NAMES[sub] || sub;
      return {
        text: `Năng Lực: ${name}`,
        color: "text-sky-300",
        icon: "🌐",
        tooltip: HOOK_BEHAVIORS[sub] || "Năng lực đặc thù kích hoạt ngoài bản đồ thế giới.",
      };
    }
    case "DAMAGE_MODIFIER": {
      if (op.sub_type === "MINION_COUNT") {
        const perMinion = Math.round((op.per_minion ?? 0.15) * 100);
        return {
          text: `Quân Đoàn Hợp Kích (+${perMinion}% ST/dị thú)`,
          color: "text-rose-300 font-semibold",
          icon: "🐾",
          tooltip: op.message || "Tăng uy lực sát thương theo số lượng dị thú/sủng vật trợ chiến phe ta.",
        };
      }
      if (op.sub_type === "MISSING_HP") {
        const percent = Math.round(((op.value ?? 1) - 1) * 100) || Math.round((op.value ?? 0) * 100);
        return {
          text: `Máu Thấp Bộc Phá (+${percent}% ST theo HP mất)`,
          color: "text-rose-300 font-semibold",
          icon: "🩸",
          tooltip: op.message || "Gia tăng sát thương khi sinh lực người thi triển bị hao hụt.",
        };
      }
      if (op.sub_type === "TARGET_DEBUFF_COUNT") {
        const perDebuff = Math.round((op.per_target_debuff ?? op.value ?? 0.1) * 100);
        return {
          text: `Khắc Chế Tàn Phế (+${perDebuff}% ST/debuff mục tiêu)`,
          color: "text-rose-300",
          icon: "📉",
          tooltip: "Gia tăng sát thương theo tổng số lượng trạng thái bất lợi trên người mục tiêu.",
        };
      }
      if (op.sub_type === "TARGET_HAS_STATUS") {
        const sName = STATUS_NAMES[op.status_id?.toUpperCase() || ""] || op.status_id || "Dị trạng";
        const percent = Math.round((op.value ?? 0) * 100);
        return {
          text: `Bồi Kích Mục Tiêu ${sName} (+${percent}% ST)`,
          color: "text-rose-300",
          icon: "🎯",
          tooltip: `Tăng mạnh sát thương khi mục tiêu đang dính ${sName}.`,
        };
      }
      if (op.sub_type === "COUNTER_COUNT") {
        return {
          text: `Tăng sát thương theo tích lũy [${op.key || "Chiến ý"}]`,
          color: "text-rose-300",
          icon: "🪙",
          tooltip: "Sát thương tăng tiến dựa trên số tầng đếm tích lũy.",
        };
      }
      return {
        text: `Khuếch Đại Sát Thương: ${op.sub_type || ""}`,
        color: "text-rose-300",
        icon: "⚔️",
        tooltip: op.message,
      };
    }
    case "DISPEL": {
      if (op.sub_type === "ALL_BUFFS") {
        return { text: "Xóa Toàn Bộ Buff Của Mục Tiêu", color: "text-amber-300 font-semibold", icon: "💨", tooltip: "Hóa giải toàn bộ các hiệu ứng có lợi trên người mục tiêu." };
      }
      if (op.sub_type === "BUFFS") {
        const count = op.value ? ` (${op.value} hiệu ứng)` : "";
        return { text: `Tước Đoạt Buff${count}`, color: "text-amber-300", icon: "💨", tooltip: "Xóa bớt các trạng thái có lợi của đối phương." };
      }
      if (op.sub_type === "PROTECTION") {
        return { text: "Phá Vỡ Lớp Bảo Vệ & Khiên", color: "text-amber-300 font-semibold", icon: "💥", tooltip: "Xuyên thủng và xóa bỏ các lớp khiên chắn hộ thể." };
      }
      return { text: `Giải trừ: ${op.sub_type || "hiệu ứng"}`, color: "text-amber-300", icon: "💨" };
    }
    case "DISPLACE": {
      if (op.sub_type === "KNOCKBACK") {
        return { text: `Đẩy Lùi ${op.value ?? 1} Hàng`, color: "text-indigo-300", icon: "🔄", tooltip: "Đẩy mục tiêu lùi về hàng sau trên chiến trận 3x3." };
      }
      if (op.sub_type === "PULL") {
        return { text: `Kéo Lại Gần ${op.value ?? 1} Hàng`, color: "text-indigo-300", icon: "🧲", tooltip: "Kéo mục tiêu lên hàng trước để dễ bề tấn công." };
      }
      if (op.sub_type === "RETREAT") {
        return { text: "Rút Về Hàng Sau", color: "text-indigo-300", icon: "↩️", tooltip: "Lùi bước về hàng sau để được tuyến trước che chở." };
      }
      if (op.sub_type === "SWAP") {
        return { text: "Hoán Đổi Vị Trí Mục Tiêu", color: "text-indigo-300", icon: "🔀", tooltip: "Thay đổi vị trí của mục tiêu trong đội hình chiến đấu." };
      }
      return { text: `Chuyển dịch vị trí (${op.sub_type || ""})`, color: "text-indigo-300", icon: "🔄" };
    }
    case "MITIGATE_DAMAGE": {
      const percent = Math.round((op.value ?? 0) * 100);
      return {
        text: `Chuyển ${percent}% Sát Thương Nhận Vào Thành Khiên`,
        color: "text-sky-300 font-semibold",
        icon: "🛡️",
        tooltip: op.message || "Hấp thụ sát thương nhận vào và chuyển hóa thành lớp Khiên Ảo mới.",
      };
    }
    case "TURN_ORDER_RULE": {
      if (op.sub_type === "SEIZE_INITIATIVE") {
        return {
          text: "Đoạt Tiên Cơ (Luôn đi trước ở Lượt 1)",
          color: "text-yellow-300 font-semibold",
          icon: "⚡",
          tooltip: "Chiếm thế chủ động, luôn được ra tay trước tiên ở lượt mở đầu bất kể tốc độ.",
        };
      }
      if (op.sub_type === "SUPPRESS_ENEMY_PRIORITY") {
        return {
          text: "Trấn Áp Tiên Thủ (Khóa quyền đi trước của địch)",
          color: "text-red-300 font-semibold",
          icon: "⛔",
          tooltip: "Vô hiệu hóa quyền ưu tiên hành động và tốc độ xuất chiêu của đối phương.",
        };
      }
      return { text: `Quy tắc lượt: ${op.sub_type || ""}`, color: "text-yellow-300", icon: "⚡" };
    }
    case "SET_RULE": {
      if (op.sub_type === "IMMUNE_DREAM_STATUS") {
        return {
          text: "Miễn Nhiễm Toàn Bộ Trạng Thái Mộng Cảnh",
          color: "text-amber-300 font-semibold",
          icon: "☯️",
          tooltip: "Không thể bị kéo vào Mộng Cảnh hoặc dính các trạng thái mộng đạo.",
        };
      }
      return { text: `Quy tắc đặc thù: ${op.sub_type || ""}`, color: "text-amber-300", icon: "☯️" };
    }
    case "SUMMON": {
      const pos = op.params?.position === "frontline" ? "tiền tuyến" : "hậu phương";
      return {
        text: `Triệu Hồi Hộ Vệ Trợ Chiến (${pos})`,
        color: "text-emerald-300 font-semibold",
        icon: "🐾",
        tooltip: "Gọi linh thú/con rối xuất hiện trên chiến trường chia sẻ sát thương và tấn công đối phương.",
      };
    }
    case "DETONATE_STATUS": {
      const sName = STATUS_NAMES[op.status_id?.toUpperCase() || ""] || op.status_id || "dị trạng";
      return {
        text: `Kích Nổ ${sName} (Bộc phát sát thương)`,
        color: "text-rose-400 font-semibold",
        icon: "💥",
        tooltip: op.message || `Kích hoạt toàn bộ các tầng ${sName} trên mục tiêu cùng lúc để tạo sát thương bộc phá.`,
      };
    }
    case "COUNTER_ADD": {
      const delta = op.delta ?? op.value ?? 1;
      const keyName = UNIQUE_NAMES[op.key?.toUpperCase() || ""] || op.key || "tầng đếm";
      const maxStr = typeof op.max === "number" ? ` (tối đa ${op.max})` : "";
      return {
        text: `Tích lũy ${delta >= 0 ? "+" : ""}${delta} [${keyName}]${maxStr}`,
        color: "text-amber-300",
        icon: "🪙",
        tooltip: "Tích lũy tầng số liệu phục vụ tính toán các đòn đánh hoặc sát chiêu tiếp theo.",
      };
    }
    case "THRESHOLD_RECOVERY": {
      const hpThresh = Math.round((op.hp_threshold ?? op.value ?? 0.3) * 100);
      const healPct = Math.round((op.heal_percent ?? 0.4) * 100);
      const cleanseStr = op.cleanse_debuffs ? " & Thanh tẩy dị trạng" : "";
      return {
        text: `Máu < ${hpThresh}%: Hồi Phục ${healPct}% HP${cleanseStr}`,
        color: "text-emerald-300 font-semibold",
        icon: "💚",
        tooltip: op.message || "Khi sinh lực giảm xuống dưới ngưỡng báo động, lập tức hồi phục cấp tốc.",
      };
    }
    case "THRESHOLD_CHECK": {
      const key = op.key || "Tầng đếm";
      const thresh = op.threshold ?? op.value ?? 0;
      return {
        text: `Kiểm tra ngưỡng [${UNIQUE_NAMES[key.toUpperCase()] || key}] ≥ ${thresh}`,
        color: "text-amber-300",
        icon: "⚖️",
        tooltip: "Kiểm tra điều kiện tầng tích lũy để kích hoạt các hiệu ứng kế tiếp.",
      };
    }
    case "APPLY_STACKING_STATUS": {
      const key = op.status_key || op.status_id || "Dị trạng";
      const maxStr = op.max_stacks ? ` (tối đa ${op.max_stacks} tầng)` : "";
      return {
        text: `Áp Đặt Dị Trạng Cộng Dồn: ${STATUS_NAMES[key.toUpperCase()] || key}${maxStr}`,
        color: "text-orange-300",
        icon: "🔥",
        tooltip: "Áp đặt trạng thái có khả năng tích tầng, uy lực tăng tiến theo số tầng dồn.",
      };
    }
    case "PERIODIC_RESOURCE": {
      const res = op.resource === "MANA" ? "Chân Nguyên" : op.resource || "Tài nguyên";
      return {
        text: `Sản sinh định kỳ: +${op.value ?? 0} ${res}/giờ`,
        color: "text-teal-300",
        icon: "⏳",
        tooltip: "Tự động sinh ra tài nguyên theo chu kỳ thời gian thực.",
      };
    }
    case "MODIFY_APERTURE": {
      if (op.sub_type === "AREA_SIZE") {
        const val = op.value ? Number(op.value).toLocaleString() : "0";
        return {
          text: `Mở Rộng Không Khiếu +${val} Mẫu`,
          color: "text-purple-300 font-semibold",
          icon: "🌌",
          tooltip: op.message || "Gia tăng vĩnh viễn diện tích Không Gian Tiên Khiếu.",
        };
      }
      if (op.sub_type === "TIME_RATE" || op.sub_type === "TIME_RATE_ADD") {
        return { text: `Gia Tốc Dòng Thời Gian Không Khiếu +${op.value ?? 0}x`, color: "text-purple-300", icon: "⏳", tooltip: "Thời gian trong Không Khiếu trôi nhanh hơn, thúc đẩy tài nguyên phát triển." };
      }
      if (op.sub_type === "TIME_RATE_SUBTRACT") {
        return { text: `Làm Chậm Dòng Thời Gian Không Khiếu -${op.value ?? 0}x`, color: "text-purple-300", icon: "⏳", tooltip: "Kéo dài thọ nguyên bằng cách làm chậm dòng thời gian trong Khiếu." };
      }
      if (op.sub_type === "STRESS_REDUCE") {
        return { text: `Giảm Áp Lực Không Khiếu ${op.value ?? 0}`, color: "text-purple-300", icon: "🛡️", tooltip: "Giảm tải áp lực không gian và sức ép của Tiên Khiếu." };
      }
      return { text: `Cải tạo Không Khiếu: ${op.sub_type || ""}`, color: "text-purple-300", icon: "🌌" };
    }
    case "MODIFY_APTITUDE": {
      const val = op.value ?? 0;
      return { text: `Nâng Cao Tư Chất Không Khiếu +${val}%`, color: "text-amber-300 font-bold", icon: "🌟", tooltip: "Gia tăng vĩnh viễn tư chất Không Khiếu (Giáp, Ất, Bính, Đinh)." };
    }
    case "PETRIFY_APERTURE": {
      return { text: "Hóa Thạch Không Khiếu (Đóng băng cảnh giới & thọ nguyên)", color: "text-stone-400 font-semibold", icon: "🗿", tooltip: "Khiến Không Khiếu hóa đá để ngừng tiêu hao thọ mệnh và thoát khỏi thiên kiếp." };
    }
    case "REPAIR_APERTURE": {
      const val = op.value ? Number(op.value).toLocaleString() : "0";
      return {
        text: `Hàn Gắn & Hồi Phục Khiếu +${val}`,
        color: "text-teal-300 font-semibold",
        icon: "🛡️",
        tooltip: op.message || "Chữa lành tổn thương và hàn gắn các vết rạn nứt trong Không Khiếu.",
      };
    }
    case "OPEN_REWARD_CONTAINER": {
      if (op.sub_type === "IMMORTAL_MINE") {
        return {
          text: "Mở Rương Tiên Mạch (Tài nguyên Tiên & Bí phương)",
          color: "text-amber-300 font-semibold",
          icon: "💎",
          tooltip: op.message || "Rương chứa tài nguyên quý từ Tiên Mạch với tỉ lệ bí phương và vật phẩm Chuyển 6-9 cao hơn hẳn.",
        };
      }
      if (op.sub_type === "MORTAL_MINE") {
        return {
          text: "Mở Rương Phàm Mỏ (Linh thạch & Linh tài)",
          color: "text-amber-300 font-semibold",
          icon: "📦",
          tooltip: op.message || "Rương chứa nhiều tài nguyên quý giá từ mỏ linh thạch.",
        };
      }
      return {
        text: `Mở Rương Chứa Thưởng: ${op.sub_type || ""}`,
        color: "text-amber-300 font-semibold",
        icon: "🎁",
        tooltip: op.message || "Mở nhận vật phẩm và phần thưởng ngẫu nhiên bên trong.",
      };
    }
    case "RANDOMIZE_TRAIT": {
      if (op.sub_type === "CONSTITUTION") {
        return {
          text: "Hoán Cải Thể Chất Ngẫu Nhiên (Nghịch thiên cải mệnh)",
          color: "text-rose-300 font-semibold",
          icon: "🧬",
          tooltip: op.message || "Đan dược nghịch thiên cải mệnh, tái tạo thể chất ngẫu nhiên cho Cổ sư.",
        };
      }
      if (op.sub_type === "RACE") {
        return {
          text: "Hoán Chuyển Dị Tộc / Chủng Tộc (Đổi huyết mạch)",
          color: "text-rose-300 font-semibold",
          icon: "🩸",
          tooltip: op.message || "Đan dược hoán đổi huyết mạch chủng tộc ngẫu nhiên.",
        };
      }
      return {
        text: `Hoán Cải Đặc Tính: ${op.sub_type || ""}`,
        color: "text-rose-300 font-semibold",
        icon: "🎲",
        tooltip: op.message || "Hoán đổi đặc tính thuộc tính bản thân ngẫu nhiên.",
      };
    }
    case "REQUIRE": {
      if (op.sub_type === "IMMORTAL") {
        return {
          text: "Yêu Cầu: Đẳng cấp Cổ Tiên (Rank 6+)",
          color: "text-red-400 font-semibold",
          icon: "⚡",
          tooltip: "Chỉ Cổ Tiên mới có thể sử dụng hoặc luyện hóa vật phẩm này.",
        };
      }
      return {
        text: `Yêu Cầu Tiên Quyết: ${op.sub_type || ""}`,
        color: "text-red-400 font-semibold",
        icon: "⚠️",
        tooltip: op.message || "Điều kiện tiên quyết bắt buộc phải thỏa mãn để sử dụng.",
      };
    }
    case "PLACE_WORLD_OBJECT": {
      const name = op.sub_type === "PRIMEVAL_SPRING" ? "Nguyên Tuyền (Mắt Nguồn Nguyên Thạch)" : op.sub_type || "Công Trình";
      return { text: `Khởi Tạo Thực Thể: ${name}`, color: "text-emerald-300 font-semibold", icon: "🏛️", tooltip: "Kiến tạo tài nguyên địa mạch trường tồn trên thế giới." };
    }
    case "ADVANCE_PROGRESSION": {
      return { text: "Thăng Tiến Tiểu Cảnh Giới Tu Vi", color: "text-yellow-300 font-bold", icon: "📈", tooltip: "Lập tức đột phá lên tiểu cảnh giới kế tiếp." };
    }
    case "ADVANCE_MAJOR_REALM": {
      return { text: "Đột Phá Đại Cảnh Giới Chuyển Bậc", color: "text-yellow-300 font-bold", icon: "⚡", tooltip: "Đột phá chuyển bậc đại cảnh giới." };
    }
    case "GRANT_BODY_MOD": {
      return { text: "Cải Tạo Tiên Thể: Thập Tuyệt Thể / Chí Tôn Tiên Thể", color: "text-amber-300 font-bold", icon: "✨", tooltip: "Tái tạo căn cốt thể chất tối thượng, xóa bỏ xung đột đạo ngân." };
    }
    case "GRANT_RESOURCE": {
      if (op.resource === "LIFESPAN") {
        return { text: `Ban Tặng: +${op.value ?? 0} Năm Thọ Nguyên`, color: "text-emerald-300 font-bold", icon: "🎁", tooltip: "Gia tăng vĩnh viễn lượng tài nguyên hoặc tuổi thọ." };
      }
      if (op.resource === "HUMAN_QI") {
        const val = op.value ? Number(op.value).toLocaleString() : "0";
        return { text: `Tăng Nhân Khí: +${val}`, color: "text-amber-300 font-bold", icon: "👤", tooltip: op.message || "Tăng lượng Nhân Khí cần thiết để thăng Tiên hoặc luyện đan." };
      }
      if (op.resource === "DAO_MARK") {
        const pathStr = op.path ? ` [${op.path}]` : "";
        return { text: `Ban Tặng: +${op.value ?? 0} Đạo Ngân${pathStr}`, color: "text-emerald-300 font-bold", icon: "💠", tooltip: op.message || "Gia tăng số lượng đạo ngân lưu phái vĩnh viễn." };
      }
      return { text: `Ban Tặng: +${op.value ?? 0} ${op.resource || ""}`, color: "text-emerald-300 font-bold", icon: "🎁", tooltip: "Gia tăng vĩnh viễn lượng tài nguyên hoặc tuổi thọ." };
    }
    case "RESTORE_RESOURCE": {
      if (op.resource === "EARTH_QI") {
        const val = op.value ? Number(op.value).toLocaleString() : "0";
        return { text: `Hồi Phục Địa Khí: +${val}`, color: "text-yellow-400 font-semibold", icon: "🌍", tooltip: op.message || "Hồi phục tài nguyên Địa Khí trong Không Khiếu." };
      }
      if (op.resource === "HEAVEN_QI") {
        const val = op.value ? Number(op.value).toLocaleString() : "0";
        return { text: `Hồi Phục Thiên Khí: +${val}`, color: "text-sky-300 font-semibold", icon: "☁️", tooltip: op.message || "Hồi phục tài nguyên Thiên Khí trong Không Khiếu." };
      }
      const res = op.resource === "SOUL" ? "Hồn Phách" : op.resource || "Tài nguyên";
      return { text: `Hồi Phục ${op.value ?? 0} ${res}`, color: "text-cyan-300", icon: "💚", tooltip: "Hồi phục tài nguyên tương ứng cho bản thân." };
    }
    case "MODIFY_MAX_RESOURCE": {
      const res = op.resource === "SOUL" ? "Giới Hạn Hồn Phách" : op.resource || "Giới Hạn Tài Nguyên";
      return { text: `Tăng ${res} +${op.value ?? 0}`, color: "text-indigo-300", icon: "📈", tooltip: "Mở rộng ngưỡng chứa tối đa của tài nguyên." };
    }
    case "GROWTH_DURATION": {
      return { text: `Thời gian trưởng thành: ${op.value ?? 0} giờ`, color: "text-teal-300", icon: "🌱", tooltip: "Thời gian nuôi dưỡng cần thiết để Cổ đạt đến trạng thái phát huy tối đa sức mạnh." };
    }
    case "DEDUCTION_FINISH_CHANCE": {
      return { text: "Tăng Xác Suất Đốn Ngộ Thôi Diễn", color: "text-amber-300", icon: "💡", tooltip: "Gia tăng tỷ lệ hoàn thành tức thì khi thôi diễn phương thuốc hoặc bí pháp." };
    }
    default: {
      const summary = op.message ? op.message.replace(/\{actor\}/g, "Bản thân").replace(/\{target\}/g, "Mục tiêu") : (op.sub_type || op.code || opType);
      return { text: summary, color: "text-zinc-200", icon: "✨", tooltip: JSON.stringify(op) };
    }
  }
}

const operationMeta = (op: EffectOperation): string[] => {
  const parts: string[] = [];
  const target = formatTarget(op.target);
  if (target) parts.push(`mục tiêu: ${target}`);
  if (op.unit) parts.push(`đơn vị: ${op.unit}`);
  if (typeof op.chance === "number") parts.push(`tỉ lệ: ${Math.round(op.chance * 100)}%`);
  if (typeof op.turns === "number") parts.push(`${op.turns} lượt`);
  if (op.stack_policy) parts.push(`cộng dồn: ${op.stack_policy}`);
  return parts;
};

export const EffectBlockItem: React.FC<{ block: EffectBlock; index?: number }> = ({ block }) => {
  const triggerKey = (block.trigger || "").toUpperCase();
  const cfg = TRIGGER_CONFIG[triggerKey] || {
    label: block.trigger || "Hiệu ứng",
    color: "bg-zinc-900/60 text-zinc-300 border-zinc-800/50",
    icon: "•",
  };
  const targetLabel = formatTarget(block.target);

  return (
    <div className="flex flex-wrap items-center gap-1.5 p-1.5 rounded-lg bg-zinc-950/60 border border-zinc-900/80 text-[10px]">
      <span className={`px-1.5 py-0.5 rounded border text-[9px] font-mono font-medium flex items-center gap-1 ${cfg.color}`}>
        <span>{cfg.icon}</span>
        <span>{cfg.label}</span>
      </span>

      {targetLabel && (
        <span className="px-1.5 py-0.5 rounded bg-zinc-900/80 text-zinc-400 border border-zinc-800/60 text-[9px] font-mono">
          🎯 {targetLabel}
        </span>
      )}

      {block.conditions?.map((condition, index) => (
        <span key={`condition-${index}`} className="px-1.5 py-0.5 rounded bg-slate-900/70 text-slate-300 border border-slate-700/60 text-[9px] font-mono">
          nếu {condition.field} {condition.operator} {condition.value}
        </span>
      ))}

      {block.operations && block.operations.map((op, i) => {
        const { text, color, icon, tooltip } = formatOperation(op);
        const metadata = operationMeta(op);
        const fullTitle = tooltip ? `${text} — ${tooltip}` : text;
        return (
          <span
            key={i}
            title={fullTitle}
            className={`px-1.5 py-0.5 rounded bg-zinc-900/40 border border-zinc-800/40 text-[9.5px] font-mono flex items-center gap-1 cursor-help hover:border-zinc-700 transition-colors ${color}`}
          >
            <span className="opacity-80 text-[9px]">{icon}</span>
            <span>{text}</span>
            {metadata.length > 0 && <span className="text-zinc-400">· {metadata.join(" · ")}</span>}
          </span>
        );
      })}
    </div>
  );
};

export const EffectBlocksList: React.FC<{ blocks?: EffectBlock[] | null }> = ({ blocks }) => {
  if (!blocks || !Array.isArray(blocks) || blocks.length === 0) return null;
  return (
    <div className="flex flex-col gap-1 w-full">
      {blocks.map((b, i) => (
        <EffectBlockItem key={b.id || i} block={b} index={i} />
      ))}
    </div>
  );
};

/** Canonical renderer. Unknown envelopes never throw and remain inspectable. */
export const EffectEnvelopeView: React.FC<{ effects?: EffectLike | null }> = ({ effects }) => {
  const presentation = renderableEffectBlocks(effects);
  if (presentation.unsupportedSchema !== undefined) {
    return <div className="text-[10px] text-amber-400">Effect schema không được hỗ trợ: {String(presentation.unsupportedSchema)}</div>;
  }
  return <EffectBlocksList blocks={presentation.blocks} />;
};
