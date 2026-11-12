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
  ON_CONSUME: { label: "Khi Nuốt", color: "bg-emerald-950/60 text-emerald-300 border-emerald-800/50", icon: "🧪" },
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
  ARENA_IDENTITY_MASK: "Mặt Nạ Đấu Trường",
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
    case "DAMAGE": {
      const pathStr = op.path ? ` [${translatePath(op.path)}]` : "";
      const tooltip = "Gây sát thương lên mục tiêu (bị giảm bởi phòng thủ, khiên và chênh lệch cảnh giới).";
      if (op.sub_type === "MULTIPLIER") {
        return { text: `Sát thương ×${op.value}${pathStr}`, color: "text-rose-300", icon: "⚔️", tooltip };
      }
      return { text: `Sát thương ${op.value ?? 0}${pathStr}`, color: "text-rose-300", icon: "⚔️", tooltip };
    }
    case "SHIELD": {
      const tooltip = "Tạo lớp khiên ảo hấp thụ sát thương trước khi trừ vào Sinh Lực.";
      if (op.sub_type === "MULTIPLIER") {
        return { text: `Khiên ×${op.value}`, color: "text-sky-300", icon: "🛡️", tooltip };
      }
      return { text: `Khiên ảo +${op.value ?? 0}`, color: "text-sky-300", icon: "🛡️", tooltip };
    }
    case "HEAL": {
      if (op.sub_type === "MANA") {
        return { text: `Hồi ${op.value ?? 0} Chân Nguyên`, color: "text-cyan-300", icon: "💧", tooltip: "Hồi phục Chân Nguyên cho Không Khiếu." };
      }
      if (op.sub_type === "LIFESTEAL") {
        return { text: `Hút máu ${Math.round((op.value ?? 0) * 100)}%`, color: "text-emerald-300", icon: "🩸", tooltip: "Hút máu chuyển hóa phần trăm sát thương gây ra thành sinh lực." };
      }
      return { text: `Hồi +${op.value ?? 0} HP`, color: "text-emerald-300", icon: "💚", tooltip: "Hồi phục Sinh Lực cho bản thân." };
    }
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
    default:
      return { text: `${opType}: ${JSON.stringify(op)}`, color: "text-zinc-300", icon: "•" };
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
            {metadata.length > 0 && <span className="text-zinc-500">· {metadata.join(" · ")}</span>}
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
