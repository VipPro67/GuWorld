package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"testing"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

// effectParityCase freezes the legacy native definition of an ON_ACTION hook
// next to the typed Block v2 definition that replaces it. The legacy JSON is
// frozen here on purpose: once Gu.csv is migrated the UNIQUE op no longer
// exists in the shipped data, so the CSV cannot be the baseline any more.
type effectParityCase struct {
	code   string
	native string // legacy UNIQUE block
	typed  string // typed replacement block(s)
}

// parityTerminator is a one-damage attack shared by both sides. It only makes
// the probe battle advance; it is not part of the migration under test.
const parityTerminator = `{"op":"DAMAGE","value":1,"sub_type":"FLAT"}`

func legacyHookBlock(code string) string {
	return `[{"id":"ACTIVE_SKILL","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[` +
		parityTerminator + `,{"op":"UNIQUE","code":"` + code + `","params":{}}]}]`
}

func typedBlock(ops string) string {
	return `[{"id":"ACTIVE_SKILL","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[` +
		parityTerminator + `,` + ops + `]}]`
}

// combatEffectParityCases is the conversion table. Every entry must produce the
// same battle state on both sides; add an entry before converting a code.
var combatEffectParityCases = []effectParityCase{
	{
		code:   "SWORD_DRAGON_TRANSFORM",
		native: `[{"id":"SWORD_DRAGON_TRANSFORM","trigger":"ON_CAST","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"SWORD_DRAGON_TRANSFORM","params":{}}]}]`,
		typed:  `[{"id":"SWORD_DRAGON_TRANSFORM","trigger":"ON_CAST","target":{"selector":"SELF"},"operations":[{"op":"RESOURCE_CONVERSION","sub_type":"CURRENT_HP_TO_SHIELD","value":1,"min_remaining":1,"log_action":"TRANSFORM","message":"{actor} hóa thân thành Thượng Cổ Kiếm Long! Chuyển {amount} HP thành Khiên Ảo, tăng 50% sát thương trong 3 lượt!"},{"op":"STATUS","status_id":"ATK_UP","value":50,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK","target":{"selector":"SELF"},"suppress_log":true}]}]`,
	},
	{
		code:   "FORMATION_DOMAIN",
		native: `[{"id":"ACTIVE_SKILL","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DAMAGE","value":1,"sub_type":"FLAT"}]},{"id":"FORMATION_DOMAIN","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"FORMATION_DOMAIN","params":{}}]}]`,
		typed:  `[{"id":"ACTIVE_SKILL","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DAMAGE","value":1,"sub_type":"FLAT"}]},{"id":"FORMATION_DOMAIN","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"SHIELD","sub_type":"PERCENT","value":20,"target":{"selector":"SELF"}},{"op":"STATUS","status_id":"DEF_UP","value":25,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK","target":{"selector":"SELF"}}]}]`,
	},
	{
		code:   "IGNITE_BURST",
		native: legacyHookBlock("IGNITE_BURST"),
		typed:  typedBlock(`{"op":"DAMAGE_MODIFIER","sub_type":"TARGET_HAS_STATUS","value":1.6,"status_id":"BURN","log_action":"IGNITE_BURST","message":"[Hỏa Đạo - Nộ Hỏa Bạo Phát] Mục tiêu {target} đang bốc cháy dữ dội! Đòn đánh phát nổ cộng hưởng tăng vọt {damage_percent}% sát thương!"}`),
	},
	{
		code:   "FIRE_INFERNO",
		native: legacyHookBlock("FIRE_INFERNO"),
		typed:  typedBlock(`{"op":"STATUS","status_id":"BURN","value":8,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK"}`),
	},
	{
		code:   "EARTH_GRAVITY",
		native: legacyHookBlock("EARTH_GRAVITY"),
		typed:  typedBlock(`{"op":"STATUS","status_id":"SLOW","value":40,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK","phase":"PRE_DAMAGE","log_action":"EARTH_GRAVITY","message":"[Thổ Đạo - Địa Chấn Trọng Lực] Trọng lực đại địa giáng xuống đè nặng {target}, làm chậm 40% tốc độ!"}`),
	},
	{
		code:   "WIND_HASTE_BURST",
		native: legacyHookBlock("WIND_HASTE_BURST"),
		typed: typedBlock(`{"op":"STATUS","status_id":"HASTE","value":40,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK","target":{"selector":"SELF"},"log_action":"WIND_HASTE_BURST","message":"[Phong Đạo - Tật Phong Tốc Độ] {actor} ngự phong phi hành, thân pháp mau lẹ tăng 40% Tốc độ và 25% Né tránh!"},` +
			`{"op":"STATUS","status_id":"EVASION","value":25,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK","target":{"selector":"SELF"},"suppress_log":true}`),
	},
	{
		code:   "RAGING_ANGRY_BIRD",
		native: legacyHookBlock("RAGING_ANGRY_BIRD"),
		typed: typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.5},` +
			`{"op":"STATUS","status_id":"BURN","value":15,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK","phase":"PRE_DAMAGE","log_action":"RAGING_ANGRY_BIRD","message":"[Viêm Đạo - Phẫn Nộ Đích Tiểu Điểu] Hỏa điểu phẫn nộ phát nổ kinh hoàng! Tăng 50% sát thương và thiêu đốt {target} trong 3 hiệp!"}`),
	},
	{
		code:   "ARMOR_BREAK_HEAVY",
		native: legacyHookBlock("ARMOR_BREAK_HEAVY"),
		typed:  typedBlock(`{"op":"STATUS","status_id":"ARMOR_BREAK","value":40,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK","log_action":"ARMOR_BREAK_HEAVY","message":"[Kim Đạo - Phá Giáp Kim Mang] Đòn đánh xé toạc hộ giáp của {target}, giảm 40% phòng thủ trong 3 lượt!"}`),
	},
	{
		code:   "EARTH_SHIELD",
		native: legacyHookBlock("EARTH_SHIELD"),
		typed:  typedBlock(`{"op":"SHIELD","sub_type":"PERCENT","value":30,"target":{"selector":"SELF"}},{"op":"STATUS","status_id":"DEF_UP","value":25,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK","target":{"selector":"SELF"},"log_action":"EARTH_SHIELD","message":"[Thổ Đạo - Đại Địa Hộ Thuẫn] {actor} dẫn động địa khí, tạo 30% Thổ Nham Giáp và tăng 25% phòng thủ!"}`),
	},
	{
		code:   "WOOD_REGEN_BURST",
		native: legacyHookBlock("WOOD_REGEN_BURST"),
		typed:  typedBlock(`{"op":"HEAL","sub_type":"RESOURCE","resource":"MANA","unit":"PERCENT","value":15,"target":{"selector":"SELF"}},{"op":"STATUS","status_id":"REGEN","value":10,"turns":4,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK","target":{"selector":"SELF"},"log_action":"WOOD_REGEN_BURST","message":"[Mộc Đạo - Sinh Sinh Bất Tức] {actor} vận khởi linh mộc sinh cơ, hồi phục 15% Chân Nguyên và kích hoạt hồi máu liên tục 4 lượt!"}`),
	},
	{
		code:   "QI_EXPLOSION",
		native: legacyHookBlock("QI_EXPLOSION"),
		typed:  typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.4}`),
	},
	{
		code:   "STAR_FIRE_RAIN",
		native: legacyHookBlock("STAR_FIRE_RAIN"),
		typed:  typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.3},{"op":"SHIELD","sub_type":"PERCENT","value":15,"target":{"selector":"SELF"}}`),
	},
	{
		code:   "WATER_CLEANSE",
		native: legacyHookBlock("WATER_CLEANSE"),
		typed:  typedBlock(`{"op":"CLEANSE","sub_type":"DEBUFF","target":{"selector":"SELF"}}`),
	},
	{
		code:   "TOTEM_KUN_PENG",
		native: legacyHookBlock("TOTEM_KUN_PENG"),
		typed:  typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.75},{"op":"STATUS","status_id":"HASTE","value":50,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK","target":{"selector":"SELF"}}`),
	},
	{
		code:   "SOUL_EXPLOSION_DANG_HUN",
		native: legacyHookBlock("SOUL_EXPLOSION_DANG_HUN"),
		typed:  typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.6},{"op":"STATUS","status_id":"STUN","turns":1,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK","target":{"selector":"ALL_ENEMIES"}}`),
	},
	{
		code:   "BLOOD_GOD_CHILD",
		native: legacyHookBlock("BLOOD_GOD_CHILD"),
		typed:  typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.5},{"op":"HEAL","sub_type":"LIFESTEAL","value":40}`),
	},
	{
		code:   "BONE_SPIKE",
		native: legacyHookBlock("BONE_SPIKE"),
		typed:  typedBlock(`{"op":"STATUS","status_id":"BLEED","value":30,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK"},{"op":"STATUS","status_id":"ARMOR_BREAK","value":30,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK"}`),
	},
	{
		code:   "SWORD_MARK_DEATH",
		native: legacyHookBlock("SWORD_MARK_DEATH"),
		typed:  typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.5},{"op":"STATUS","status_id":"SWORD_MARK","turns":4,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK"},{"op":"STATUS","status_id":"SWORD_MARK","turns":4,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK"}`),
	},
	{
		code:   "THREE_MEN_MAKE_TIGER",
		native: legacyHookBlock("THREE_MEN_MAKE_TIGER"),
		typed:  typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.5},{"op":"STATUS","status_id":"FEAR","turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK"},{"op":"STATUS","status_id":"WEAK","value":25,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK"}`),
	},
	{
		code:   "IMMORTAL_BINDING_VINE",
		native: legacyHookBlock("IMMORTAL_BINDING_VINE"),
		typed:  typedBlock(`{"op":"STATUS","status_id":"BIND","turns":2,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK"},{"op":"TRANSFER_RESOURCE","resource":"MANA","sub_type":"PERCENT_OF_TARGET","value":0.2}`),
	},
	{
		code:   "EXTREME_ICE_CRYSTAL_COFFIN",
		native: legacyHookBlock("EXTREME_ICE_CRYSTAL_COFFIN"),
		typed:  typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.4},{"op":"STATUS","status_id":"FREEZE","turns":2,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK"},{"op":"STATUS","status_id":"SILENCE","turns":2,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK"}`),
	},
	{
		code:   "INFINITE_WIND",
		native: legacyHookBlock("INFINITE_WIND"),
		typed:  typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.75},{"op":"STATUS","status_id":"HASTE","value":60,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK","target":{"selector":"SELF"}},{"op":"STATUS","status_id":"EVASION","value":35,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK","target":{"selector":"SELF"}}`),
	},
	{
		code:   "MYRIAD_STAR_FIREFLY",
		native: legacyHookBlock("MYRIAD_STAR_FIREFLY"),
		typed:  typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.6},{"op":"TRANSFER_RESOURCE","resource":"MANA","sub_type":"PERCENT_OF_TARGET","value":0.4}`),
	},
	{
		code:   "YOU_DU_STRENGTH_TIGER",
		native: legacyHookBlock("YOU_DU_STRENGTH_TIGER"),
		typed:  typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.85},{"op":"STATUS","status_id":"BLEED","value":20,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK"},{"op":"STATUS","status_id":"STUN","turns":1,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK"}`),
	},
	{
		code:   "FUTURE_SELF",
		native: legacyHookBlock("FUTURE_SELF"),
		typed:  typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.8},{"op":"STATUS","status_id":"FUTURE_SELF","value":80,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK","target":{"selector":"SELF"}},{"op":"STATUS","status_id":"HASTE","value":50,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK","target":{"selector":"SELF"}}`),
	},
	{
		code:   "FAREWELL_FRIEND_WIND",
		native: legacyHookBlock("FAREWELL_FRIEND_WIND"),
		typed:  typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.5},{"op":"HIT_RULE","sub_type":"BYPASS_SHIELD"},{"op":"STATUS","status_id":"FAREWELL_CURSE","value":10,"turns":4,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK"}`),
	},
	{
		code:   "DIMENSION_SLASH",
		native: legacyHookBlock("DIMENSION_SLASH"),
		typed:  typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.25}`),
	},
	{
		code:   "SPATIAL_SHIFT",
		native: legacyHookBlock("SPATIAL_SHIFT"),
		typed:  typedBlock(`{"op":"STATUS","status_id":"EVASION","value":75,"turns":1,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK","target":{"selector":"SELF"}}`),
	},
	{
		code:   "JUSTICE_VERDICT",
		native: legacyHookBlock("JUSTICE_VERDICT"),
		typed:  typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.35}`),
	},
	{
		code:   "HERO_AMONG_MEN",
		native: legacyHookBlock("HERO_AMONG_MEN"),
		typed:  typedBlock(`{"op":"DAMAGE_MODIFIER","sub_type":"MISSING_HP","value":1.3,"per_missing_hp":1.5,"log_action":"HERO_AMONG_MEN","message":"[Nhân Đạo - Nhân Trung Hào Kiệt] Hào kiệt nghịch cảnh chuyển thương thành dũng! Máu hao hụt {missing_hp_percent}% bộc phát tăng {damage_percent}% sát thương!"}`),
	},
	{
		code:   "NO_EXTREME_SPACE",
		native: legacyHookBlock("NO_EXTREME_SPACE"),
		typed:  typedBlock(`{"op":"STATUS","status_id":"EVASION","value":50,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK","target":{"selector":"SELF"}}`),
	},
	{
		code:   "HEAVEN_NET_RESTRICTION",
		native: legacyHookBlock("HEAVEN_NET_RESTRICTION"),
		typed:  typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.5},{"op":"HIT_RULE","sub_type":"GUARANTEED"},{"op":"STATUS","status_id":"SLOW1","value":40,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK"}`),
	},
	{
		code:   "CALCULATED_STRIKE",
		native: legacyHookBlock("CALCULATED_STRIKE"),
		typed:  typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.35}`),
	},
	{
		code:   "VAN_DAN_DONG_TAM",
		native: legacyHookBlock("VAN_DAN_DONG_TAM"),
		typed:  typedBlock(`{"op":"SHIELD","sub_type":"PERCENT","value":35,"target":{"selector":"SELF"}}`),
	},
	{
		code:   "TIME_REWIND",
		native: legacyHookBlock("TIME_REWIND"),
		typed:  typedBlock(`{"op":"HEAL","sub_type":"PERCENT","value":35,"target":{"selector":"SELF"}},{"op":"HEAL","sub_type":"RESOURCE","resource":"MANA","unit":"PERCENT","value":40,"target":{"selector":"SELF"}}`),
	},
	{
		code:   "TIEP_THIEN_DIA",
		native: legacyHookBlock("TIEP_THIEN_DIA"),
		typed:  typedBlock(`{"op":"HEAL","sub_type":"RESOURCE","resource":"MANA","unit":"PERCENT","value":25,"target":{"selector":"SELF"}},{"op":"CLEANSE","sub_type":"ALL","target":{"selector":"SELF"}}`),
	},
	{
		code:   "PAINTING_IMPRISONMENT",
		native: legacyHookBlock("PAINTING_IMPRISONMENT"),
		typed:  typedBlock(`{"op":"STATUS","status_id":"BIND","turns":1,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK"}`),
	},
	{
		code:   "LIGHT_PURIFY",
		native: legacyHookBlock("LIGHT_PURIFY"),
		typed:  typedBlock(`{"op":"HEAL","sub_type":"PERCENT","value":25,"target":{"selector":"SELF"}},{"op":"CLEANSE","sub_type":"DEBUFF","target":{"selector":"SELF"}}`),
	},
	{
		code:   "SOUL_BEAST_SUMMON",
		native: legacyHookBlock("SOUL_BEAST_SUMMON"),
		typed:  typedBlock(`{"op":"SHIELD","sub_type":"PERCENT","value":35,"target":{"selector":"SELF"}},{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.3}`),
	},
	{
		code:   "CHAOS_CAVE_DRILL",
		native: legacyHookBlock("CHAOS_CAVE_DRILL"),
		typed:  typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.8},{"op":"DIRECT_HP","sub_type":"SHIELD_STRIP","value":100},{"op":"HIT_RULE","sub_type":"BYPASS_SHIELD"}`),
	},
	{
		code:   "LANDSCAPE_AS_BEFORE",
		native: legacyHookBlock("LANDSCAPE_AS_BEFORE"),
		typed:  typedBlock(`{"op":"SHIELD","sub_type":"PERCENT","value":40,"target":{"selector":"SELF"}},{"op":"CLEANSE","sub_type":"DEBUFF","target":{"selector":"SELF"}}`),
	},
	{
		code:   "THIRTY_THREE_HEAVEN_LIGHT",
		native: legacyHookBlock("THIRTY_THREE_HEAVEN_LIGHT"),
		typed:  `[{"id":"ACTIVE_SKILL","trigger":"ON_ACTION","target":{"selector":"ALL_ENEMIES"},"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":1},{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.45},{"op":"STATUS","status_id":"BLIND","value":35,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK"},{"op":"CLEANSE","sub_type":"DEBUFF","target":{"selector":"ALL_ALLIES"}}]}]`,
	},
	{
		code:   "WORDS_ARE_FRIGHTENING",
		native: legacyHookBlock("WORDS_ARE_FRIGHTENING"),
		typed: typedBlock(`{"op":"DAMAGE_MODIFIER","sub_type":"TARGET_DEBUFF_COUNT","value":1.4,"per_target_debuff":0.25,"log_action":"WORDS_ARE_FRIGHTENING","message":"[Tín Đạo - Nhân Ngôn Khả Úy] Lời đồn thế gian như dao nhọn xé lòng! Kích hoạt {debuff_count} debuff tăng {damage_percent}% sát thương và Câm Lặng {target} trong 2 hiệp!"},` +
			`{"op":"STATUS","status_id":"SILENCE","value":1,"turns":2,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK","suppress_log":true}`),
	},
	{
		code:   "SONG_OF_SEPARATION",
		native: legacyHookBlock("SONG_OF_SEPARATION"),
		typed: typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.65},` +
			`{"op":"DIRECT_HP","sub_type":"SHIELD_STRIP","value":30,"rank_suppression":false},` +
			`{"op":"DISPEL","sub_type":"BUFFS","value":2,"log_action":"SONG_OF_SEPARATION","message":"[Âm Đạo - Ly Ca] Khúc ca ly tán vạn vật vút lên! Tăng 65% sát thương, giải trừ {dispelled} buff, phá hủy {shield_stripped} Giáp và khiến {target} chịu trạng thái Suy Yếu (giảm 25% sức chống cự)!"},` +
			`{"op":"STATUS","status_id":"WEAK","value":25,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK","suppress_log":true}`),
	},
	{
		code:   "SWORD_WAVE_TRIPLE",
		native: legacyHookBlock("SWORD_WAVE_TRIPLE"),
		typed: typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.8},` +
			`{"op":"HIT_RULE","sub_type":"MULTI_HIT","params":{"count":3}},` +
			`{"op":"STATUS","status_id":"BLEED","value":25,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK","suppress_log":true}`),
	},
	{
		code:   "MANA_EAT",
		native: legacyHookBlock("MANA_EAT"),
		typed:  typedBlock(`{"op":"HEAL","sub_type":"RESOURCE","resource":"MANA","unit":"PERCENT","value":25,"target":{"selector":"SELF"}}`),
	},
	{
		code:   "FORM_BEAR",
		native: legacyHookBlock("FORM_BEAR"),
		typed: typedBlock(`{"op":"STATUS","status_id":"BEAR_FORM","value":30,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK","target":{"selector":"SELF"}},` +
			`{"op":"SHIELD","sub_type":"PERCENT","value":25,"target":{"selector":"SELF"}}`),
	},
	{
		code:   "HEAVEN_SUPPRESSION",
		native: legacyHookBlock("HEAVEN_SUPPRESSION"),
		typed:  typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.35},{"op":"DISPEL","sub_type":"BUFFS","value":2}`),
	},
	{
		code:   "BLINDING_MIRAGE",
		native: legacyHookBlock("BLINDING_MIRAGE"),
		typed:  typedBlock(`{"op":"STATUS","status_id":"CONFUSE","turns":2,"chance":0,"application_mode":"DIRECT","stack_policy":"STACK"},{"op":"STATUS","status_id":"BLIND","value":40,"turns":2,"chance":0,"application_mode":"DIRECT","stack_policy":"STACK"}`),
	},
	{
		code:   "RULE_IMPOSITION",
		native: legacyHookBlock("RULE_IMPOSITION"),
		typed:  typedBlock(`{"op":"DISPEL","sub_type":"ALL_BUFFS"}`),
	},
	{
		code:   "DAO_ADAPTIVE_RESIST",
		native: `[ {"id":"ACTIVE_SKILL","target":{"selector":"TARGET"},"trigger":"ON_ACTION","operations":[{"op":"DAMAGE","sub_type":"FLAT","value":1}]}, {"id":"DAO_ADAPTIVE_RESIST","target":{"selector":"SELF"},"trigger":"ON_ACTION","operations":[{"op":"UNIQUE","code":"DAO_ADAPTIVE_RESIST","params":{}}]} ]`,
		typed:  `[ {"id":"ACTIVE_SKILL","target":{"selector":"TARGET"},"trigger":"ON_ACTION","operations":[{"op":"DAMAGE","sub_type":"FLAT","value":1}]}, {"id":"DAO_ADAPTIVE_RESIST","target":{"selector":"SELF"},"trigger":"ON_ACTION","operations":[{"op":"STATUS","status_id":"ADAPTIVE_RESIST","value":25,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK"}]} ]`,
	},
	{
		code:   "SOUL_SHOCK",
		native: legacyHookBlock("SOUL_SHOCK"),
		typed:  typedBlock(`{"op":"HIT_RULE","sub_type":"BYPASS_SHIELD"}`),
	},
	{
		code:   "DEVOUR_HEAVEN_EARTH",
		native: legacyHookBlock("DEVOUR_HEAVEN_EARTH"),
		typed:  typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.5},{"op":"TRANSFER_RESOURCE","resource":"HP","sub_type":"PERCENT_OF_TARGET_SHIELD","value":0.5,"phase":"EARLY_RIDER","log_action":"DEVOUR_HEAVEN_EARTH","message":"Hút {amount} khiên của {target}."}`),
	},
	{
		code:   "FATE_VANQUISHED",
		native: legacyHookBlock("FATE_VANQUISHED"),
		typed: typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":2.2},` +
			`{"op":"DISPEL","sub_type":"PROTECTION","log_action":"FATE_VANQUISHED","message":"[Thiên Đạo - Mệnh Bại] Hồng Liên xé rách số mệnh! Phá vỡ hoàn toàn Bất Tử, Bất Hoại và Giáp ảo, giáng đòn chí mạng x2.2 sát thương lên {target}!"}`),
	},
	{
		code:   "MYRIAD_DISMANTLE",
		native: legacyHookBlock("MYRIAD_DISMANTLE"),
		typed:  typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.6},{"op":"DIRECT_HP","sub_type":"SHIELD_STRIP","value":100,"rank_suppression":false},{"op":"DISPEL","sub_type":"ALL_BUFFS","log_action":"MYRIAD_DISMANTLE","message":"[Luật Đạo - Vạn Giải] Vạn vật quy nhất, pháp tắc tiêu giải! Hủy toàn bộ Hộ Thuẫn, xóa sạch mọi Buff và đánh xuyên 100% phòng ngự của {target}!"},{"op":"DEFENSE_PENETRATION","value":1}`),
	},
	{
		code:   "BEAST_PHANTOM_STRIKE",
		native: legacyHookBlock("BEAST_PHANTOM_STRIKE"),
		typed:  typedBlock(`{"op":"BONUS_STRIKE","chance":0.45,"value":0.75,"min_damage":50,"log_action":"BEAST_PHANTOM_STRIKE","message":"[Lực Đạo - Thú Lực Hư Ảnh] Tiếng thú gầm thét kinh thiên! Hư Ảnh bồi thêm một đòn bạt sơn bạt hải lên {target}, gây thêm {damage} sát thương!"}`),
	},
	{
		code:   "SONG_OF_DESTINY",
		native: legacyHookBlock("SONG_OF_DESTINY"),
		typed: `[
			{"id":"ACTIVE_SKILL","trigger":"ON_ACTION","target":{"selector":"ALL_ENEMIES"},"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":1},{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.5},{"op":"STATUS","status_id":"SONG_OF_DESTINY_WEAK","value":30,"turns":1,"chance":1,"application_mode":"DIRECT","stack_policy":"REPLACE","suppress_log":true}]},
			{"id":"DESTINY_ALLY_SUPPORT","trigger":"ON_ACTION","target":{"selector":"ALL_ALLIES"},"operations":[{"op":"STATUS","status_id":"SONG_OF_DESTINY","value":40,"turns":1,"chance":1,"application_mode":"DIRECT","stack_policy":"REPLACE","suppress_log":true},{"op":"HEAL","sub_type":"RESOURCE","resource":"MANA","unit":"PERCENT","value":20,"suppress_log":true}]},
			{"id":"DESTINY_LOG","trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[{"op":"NOTIFY","log_action":"SONG_OF_DESTINY","message":"[Âm Vận Đạo - Vận Mệnh Ca] Khúc ca vận mệnh chấn động thương khung! Tăng 40% sát thương và hồi 20% Chân Nguyên cho toàn đội, giảm 30% sát thương của toàn bộ địch thủ!"}]}
		]`,
	},
	{
		code:   "IMMEASURABLE_QI_SEA",
		native: legacyHookBlock("IMMEASURABLE_QI_SEA"),
		typed: `[{"id":"ACTIVE_SKILL","trigger":"ON_ACTION","target":{"selector":"ALL_ENEMIES"},"operations":[` +
			`{"op":"DAMAGE","sub_type":"FLAT","value":1},{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.7},` +
			`{"op":"TARGET_DAMAGE_MODIFIER","sub_type":"REDUCE","value":0.3,"min_multiplier":0.2,"log_action":"IMMEASURABLE_QI_SEA","message":"[Khí Đạo - Vô Lượng Khí Hải] Khí Hải Lão Tổ thôi động khí triều mênh mông như biển vô lượng! Tăng 70% sát thương toàn diện, giảm 30% Tấn Công và làm chậm 30% toàn bộ quân địch!"},` +
			`{"op":"STATUS","status_id":"SLOW1","value":30,"turns":3,"chance":1,"application_mode":"DIRECT","stack_policy":"STACK","phase":"PRE_DAMAGE","suppress_log":true}]}]`,
	},
	{
		code:   "THREE_QI_RETURN",
		native: legacyHookBlock("THREE_QI_RETURN"),
		typed: typedBlock(`{"op":"DAMAGE","sub_type":"MULTIPLIER","value":1.85},` +
			`{"op":"DEFENSE_PENETRATION","value":0.5},` +
			`{"op":"HEAL","sub_type":"PERCENT","value":30,"target":{"selector":"SELF"}},` +
			`{"op":"HEAL","sub_type":"RESOURCE","resource":"MANA","unit":"PERCENT","value":100,"target":{"selector":"SELF"}}`),
	},
}

// TestCombatEffectMigrationParity proves a typed replacement produces the same
// combat outcome as the legacy native hook it replaces. HP, shield, mana and
// the applied status multiset are compared; log phrasing is not, because the
// two authoring paths word their entries differently.
func TestCombatEffectMigrationParity(t *testing.T) {
	for _, tc := range combatEffectParityCases {
		t.Run(tc.code, func(t *testing.T) {
			nativeAction, err := parityAction(tc.native, false)
			if err != nil {
				t.Fatalf("native %s: %v", tc.code, err)
			}
			typedAction, err := parityAction(tc.typed, true)
			if err != nil {
				t.Fatalf("typed %s: %v", tc.code, err)
			}
			seedCleanseDebuff := tc.code == "LANDSCAPE_AS_BEFORE" || tc.code == "THIRTY_THREE_HEAVEN_LIGHT"
			seedEnemyDebuffs := tc.code == "WORDS_ARE_FRIGHTENING"
			seedTargetBuffs := tc.code == "SONG_OF_SEPARATION" || tc.code == "MYRIAD_DISMANTLE"
			seedLowMana := tc.code == "MANA_EAT" || tc.code == "SONG_OF_DESTINY" || tc.code == "THREE_QI_RETURN"
			seedEnemyShield := tc.code == "SOUL_SHOCK"
			seedBurningTarget := tc.code == "IGNITE_BURST"
			seed := int64(7)
			if tc.code == "BEAST_PHANTOM_STRIKE" {
				seed = 9 // First bonus-strike roll succeeds, so both damage and log parity are covered.
			}
			nativeState, nativeLogs := simulateParityAction(nativeAction, seedCleanseDebuff, seedEnemyDebuffs, seedTargetBuffs, seedLowMana, seedEnemyShield, seedBurningTarget, seed)
			typedState, typedLogs := simulateParityAction(typedAction, seedCleanseDebuff, seedEnemyDebuffs, seedTargetBuffs, seedLowMana, seedEnemyShield, seedBurningTarget, seed)
			if diff := compareParityState(nativeState, typedState); diff != "" {
				t.Fatalf("typed block diverges from native hook:\n%s\nnative logs: %+v\ntyped logs: %+v\ntyped action: %+v", diff, nativeLogs, typedLogs, typedAction)
			}
			if tc.code == "FATE_VANQUISHED" {
				nativeLog, typedLog := findParityLog(nativeLogs, tc.code), findParityLog(typedLogs, tc.code)
				if nativeLog == nil || typedLog == nil || nativeLog.Detail != typedLog.Detail {
					t.Fatalf("protection-break log differs: native=%+v typed=%+v native logs=%+v typed logs=%+v", nativeLog, typedLog, nativeLogs, typedLogs)
				}
			}
			if tc.code == "SWORD_DRAGON_TRANSFORM" {
				nativeLog, typedLog := findParityLog(nativeLogs, tc.code), findParityLog(typedLogs, tc.code)
				if nativeLog == nil || typedLog == nil || nativeLog.Detail != typedLog.Detail || nativeLog.Action != typedLog.Action {
					t.Fatalf("transform log differs: native=%+v typed=%+v", nativeLog, typedLog)
				}
			}
			if tc.code == "IMMEASURABLE_QI_SEA" {
				if math.Abs(nativeState.Enemies[0].DamageMultiplier-typedState.Enemies[0].DamageMultiplier) > 1e-9 || math.Abs(typedState.Enemies[0].DamageMultiplier-0.4) > 1e-9 {
					t.Fatalf("persistent target damage multiplier differs: native=%v typed=%v", nativeState.Enemies[0].DamageMultiplier, typedState.Enemies[0].DamageMultiplier)
				}
				nativeLog, typedLog := findParityLog(nativeLogs, tc.code), findParityLog(typedLogs, tc.code)
				if nativeLog == nil || typedLog == nil || nativeLog.Detail != typedLog.Detail {
					t.Fatalf("aggregate combat log differs: native=%+v typed=%+v", nativeLog, typedLog)
				}
			}
			if tc.code == "THREE_QI_RETURN" && (nativeState.Allies[0].IgnoreDefensePct != 0.5 || typedState.Allies[0].IgnoreDefensePct != 0.5) {
				t.Fatalf("defense penetration differs: native=%v typed=%v", nativeState.Allies[0].IgnoreDefensePct, typedState.Allies[0].IgnoreDefensePct)
			}
			if tc.code == "MYRIAD_DISMANTLE" {
				if nativeState.Allies[0].IgnoreDefensePct != 1 || typedState.Allies[0].IgnoreDefensePct != 1 {
					t.Fatalf("full defense penetration differs: native=%v typed=%v", nativeState.Allies[0].IgnoreDefensePct, typedState.Allies[0].IgnoreDefensePct)
				}
				nativeLog, typedLog := findParityLog(nativeLogs, tc.code), findParityLog(typedLogs, tc.code)
				const detail = "Vạn vật quy nhất, pháp tắc tiêu giải! Hủy toàn bộ Hộ Thuẫn, xóa sạch mọi Buff và đánh xuyên 100% phòng ngự của Địch!"
				if nativeLog == nil || typedLog == nil || !strings.Contains(nativeLog.Detail, detail) || !strings.Contains(typedLog.Detail, detail) {
					t.Fatalf("aggregate combat log differs: native=%+v typed=%+v", nativeLog, typedLog)
				}
			}
			if tc.code == "BEAST_PHANTOM_STRIKE" {
				nativeLog, typedLog := findParityLog(nativeLogs, tc.code), findParityLog(typedLogs, tc.code)
				if nativeLog == nil || typedLog == nil || nativeLog.Detail != typedLog.Detail || nativeLog.Damage != typedLog.Damage {
					t.Fatalf("bonus strike did not proc identically: native=%+v typed=%+v; native action=%+v typed action=%+v; logs native=%+v typed=%+v", nativeLog, typedLog, nativeAction, typedAction, nativeLogs, typedLogs)
				}
			}
			if tc.code == "RAGING_ANGRY_BIRD" || tc.code == "EARTH_GRAVITY" || tc.code == "WIND_HASTE_BURST" || tc.code == "HERO_AMONG_MEN" || tc.code == "IGNITE_BURST" {
				nativeLog, typedLog := findParityLog(nativeLogs, tc.code), findParityLog(typedLogs, tc.code)
				logsMatch := nativeLog != nil && typedLog != nil && nativeLog.Detail == typedLog.Detail
				if tc.code == "HERO_AMONG_MEN" {
					const detail = "[Nhân Đạo - Nhân Trung Hào Kiệt] Hào kiệt nghịch cảnh chuyển thương thành dũng! Máu hao hụt 100% bộc phát tăng 180% sát thương!"
					logsMatch = false
					for _, log := range nativeLogs {
						if strings.Contains(log.Detail, detail) {
							for _, typed := range typedLogs {
								if strings.Contains(typed.Detail, detail) {
									logsMatch = true
									break
								}
							}
						}
					}
				}
				if tc.code == "IGNITE_BURST" {
					const detail = "[Hỏa Đạo - Nộ Hỏa Bạo Phát] Mục tiêu Địch đang bốc cháy dữ dội! Đòn đánh phát nổ cộng hưởng tăng vọt 60% sát thương!"
					logsMatch = nativeLog != nil && typedLog != nil && strings.Contains(nativeLog.Detail, detail) && strings.Contains(typedLog.Detail, detail)
				}
				if tc.code == "WIND_HASTE_BURST" && nativeLog != nil && typedLog != nil {
					const detail = "[Phong Đạo - Tật Phong Tốc Độ] Người Chơi ngự phong phi hành, thân pháp mau lẹ tăng 40% Tốc độ và 25% Né tránh!"
					logsMatch = strings.Contains(nativeLog.Detail, detail) && typedLog.Detail == "Người Chơi dùng [Cổ Đối Chiếu], "+detail
					count := 0
					nativeCount := 0
					for _, log := range typedLogs {
						if strings.Contains(log.Detail, "Tật Phong Tốc Độ") {
							count++
						}
					}
					for _, log := range nativeLogs {
						if strings.Contains(log.Detail, "Tật Phong Tốc Độ") {
							nativeCount++
						}
					}
					logsMatch = logsMatch && count == nativeCount
				}
				if !logsMatch {
					t.Fatalf("authored status log differs: native=%+v typed=%+v; native logs=%+v; typed logs=%+v", nativeLog, typedLog, nativeLogs, typedLogs)
				}
			}
			if tc.code == "WORDS_ARE_FRIGHTENING" {
				nativeLog, typedLog := findParityLog(nativeLogs, tc.code), findParityLog(typedLogs, tc.code)
				if nativeLog == nil || typedLog == nil || nativeLog.Detail != typedLog.Detail {
					t.Fatalf("damage modifier log differs: native=%+v typed=%+v; native logs=%+v; typed logs=%+v", nativeLog, typedLog, nativeLogs, typedLogs)
				}
			}
			if tc.code == "SONG_OF_SEPARATION" || tc.code == "SONG_OF_DESTINY" {
				nativeLog, typedLog := findParityLog(nativeLogs, tc.code), findParityLog(typedLogs, tc.code)
				logsMatch := nativeLog != nil && typedLog != nil && nativeLog.Detail == typedLog.Detail
				if tc.code == "SONG_OF_DESTINY" && nativeLog != nil && typedLog != nil {
					logsMatch = strings.Contains(nativeLog.Detail, "Khúc ca vận mệnh chấn động thương khung!") && strings.Contains(typedLog.Detail, "Khúc ca vận mệnh chấn động thương khung!")
				}
				if !logsMatch {
					t.Fatalf("dispel log differs: native=%+v typed=%+v; native logs=%+v; typed logs=%+v", nativeLog, typedLog, nativeLogs, typedLogs)
				}
			}
		})
	}
}

func TestIgniteBurstRequiresBurnBeforeTheAction(t *testing.T) {
	nativeAction, err := parityAction(legacyHookBlock("IGNITE_BURST"), false)
	if err != nil {
		t.Fatalf("native IGNITE_BURST: %v", err)
	}
	typedAction, err := parityAction(typedBlock(`{"op":"DAMAGE_MODIFIER","sub_type":"TARGET_HAS_STATUS","value":1.6,"status_id":"BURN"}`), true)
	if err != nil {
		t.Fatalf("typed IGNITE_BURST: %v", err)
	}
	nativeState, _ := simulateParityAction(nativeAction, false, false, false, false, false, false, 7)
	typedState, _ := simulateParityAction(typedAction, false, false, false, false, false, false, 7)
	if diff := compareParityState(nativeState, typedState); diff != "" {
		t.Fatalf("modifier should not activate without pre-existing BURN:\n%s", diff)
	}
}

func parityGuSpec() *models.GuDictionaryItem {
	return &models.GuDictionaryItem{ID: "GU_PARITY", Name: "Cổ Đối Chiếu", Path: []string{"fire"}, Rank: 3}
}

func parityActor() *models.Combatant {
	return &models.Combatant{
		ID: "GU_PARITY", Name: "Cổ Đối Chiếu", Rank: 3,
		HP: 2000, MaxHP: 2000, CurrentMana: 2000, MaxMana: 2000,
		BaseAttack: 100, BaseDefense: 50, Speed: 30,
		GridX: 1, GridY: 1,
	}
}

// parityAction compiles one authored block. Native blocks feed SimAction
// Mechanics (the legacy hook channel); typed blocks feed OperationEffects (the
// Block v2 channel).
func parityAction(blockJSON string, typed bool) (*simulator.SimAction, error) {
	var blocks []gameplayeffects.Block
	if err := json.Unmarshal([]byte(blockJSON), &blocks); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	compiled, err := (gameplayeffects.Compiler{}).Compile(blocks)
	if err != nil {
		return nil, fmt.Errorf("compile: %w", err)
	}
	operations := append(compiled.Operations(gameplayeffects.TriggerOnAction), compiled.Operations(gameplayeffects.TriggerOnCast)...)
	if !typed {
		filtered := operations[:0]
		for _, operation := range operations {
			if operation.Trigger == gameplayeffects.TriggerOnCast && operation.Type == gameplayeffects.OpUnique {
				continue // legacy ON_CAST hooks enter through the compatibility mechanic carrier
			}
			filtered = append(filtered, operation)
		}
		operations = filtered
	}
	adapted, err := adaptBlockCombatOperations(parityActor(), parityGuSpec(), 1.0, operations)
	if err != nil {
		return nil, fmt.Errorf("adapt: %w", err)
	}
	action := simulator.SimAction{
		ActionType: "GU_SKILL",
		Name:       "Cổ Đối Chiếu",
		SourceID:   "GU_PARITY",
		Path:       []string{"fire"},
		Rank:       3,
		Damage:     1,
	}
	if typed {
		action.OperationEffects = adapted
	} else {
		for _, operation := range compiled.Operations(gameplayeffects.TriggerOnCast) {
			if operation.Type == gameplayeffects.OpUnique {
				action.Mechanics = append(action.Mechanics, simulator.CombatMechanic{Trigger: string(operation.Trigger), ActionCode: operation.Code, Params: append([]byte(nil), operation.Params...), Values: operation.Values})
			}
		}
		for _, operation := range adapted {
			if operation.Action == nil {
				continue
			}
			action.Mechanics = append(action.Mechanics, operation.Action.Mechanics...)
		}
	}
	return &action, nil
}

func simulateParityAction(action *simulator.SimAction, seedCleanseDebuff, seedEnemyDebuffs, seedTargetBuffs, seedLowMana, seedEnemyShield, seedBurningTarget bool, seed int64) (*simulator.CombatState, []simulator.CombatLog) {
	ally := &simulator.CombatantAuto{
		ID: "ally", Name: "Người Chơi", Rank: 3,
		HP: 1, MaxHP: 2000, Mana: 2000, MaxMana: 2000,
		BaseDmg: 100, BaseDef: 50, Speed: 30, GridX: 1, GridY: 1,
		Actions: []simulator.SimAction{*action},
	}
	for _, mechanic := range action.Mechanics {
		if mechanic.ActionCode == "SWORD_DRAGON_TRANSFORM" {
			ally.HP = 1000
		}
	}
	for _, effect := range action.OperationEffects {
		if effect.Action != nil && len(effect.Action.ResourceConversions) > 0 {
			ally.HP = 1000
		}
	}
	if seedLowMana {
		ally.Mana = 0
	}
	if seedCleanseDebuff {
		ally.ActiveEffects = []simulator.Effect{
			{ID: "parity_debuff", Type: "debuff", Duration: 2},
			{ID: "parity_buff", Type: "buff", Duration: 2},
		}
	}
	enemy := &simulator.CombatantAuto{
		ID: "enemy", Name: "Địch", Rank: 3,
		// The target survives the probe action, then ends combat on its reply.
		// This isolates one hook dispatch while allowing typed post-damage status
		// applications to land on a living target.
		HP: 1000, MaxHP: 1000, Mana: 100, MaxMana: 100,
		BaseDmg: 0, BaseDef: 0, Speed: 5, GridX: 1, GridY: 1,
		Actions: []simulator.SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 100}},
	}
	if seedEnemyDebuffs {
		enemy.ActiveEffects = []simulator.Effect{
			{ID: "WEAK", Type: "debuff", Value: 20, Duration: 3},
			{ID: "BURN", Type: "debuff", Value: 15, Duration: 3},
		}
	}
	if seedTargetBuffs {
		// Song's native shield strip is rank independent; this catches a typed
		// migration that accidentally inherits normal rank suppression.
		enemy.Rank = 8
		enemy.BaseDef = 500
		enemy.VirtualShield = 100
		enemy.ActiveEffects = []simulator.Effect{
			{ID: "ATK_UP", Type: "buff", Value: 20, Duration: 2},
			{ID: "DEF_UP", Type: "buff", Value: 20, Duration: 2},
			{ID: "HASTE", Type: "buff", Value: 20, Duration: 2},
		}
	}
	formBearProbe := false
	for _, mechanic := range action.Mechanics {
		formBearProbe = formBearProbe || mechanic.ActionCode == "FORM_BEAR"
	}
	for _, effect := range action.OperationEffects {
		formBearProbe = formBearProbe || effect.Compiled.Status != nil && effect.Compiled.Status.ID == "BEAR_FORM"
	}
	if formBearProbe {
		enemy.HP = 2
	}
	if seedEnemyShield {
		enemy.VirtualShield = 100
	}
	if seedBurningTarget {
		enemy.ActiveEffects = append(enemy.ActiveEffects, simulator.Effect{ID: "BURN", Type: "debuff", Value: 8, Duration: 3})
	}
	fateProbe := false
	for _, mechanic := range action.Mechanics {
		fateProbe = fateProbe || mechanic.ActionCode == "FATE_VANQUISHED"
	}
	for _, effect := range action.OperationEffects {
		fateProbe = fateProbe || effect.Action != nil && effect.Action.DispelProtections
	}
	if fateProbe {
		enemy.VirtualShield = 500
		enemy.IsInvincible = true
		enemy.IsLastStand = true
		enemy.DamageTakenReduction = 1
		enemy.ActiveEffects = append(enemy.ActiveEffects,
			simulator.Effect{ID: "invulnerable", Type: "buff", Duration: 3},
			simulator.Effect{ID: "bat_tu", Type: "buff", Duration: 3},
			simulator.Effect{ID: "bat_hoai", Type: "buff", Duration: 3},
			simulator.Effect{ID: "ATK_UP", Type: "buff", Value: 20, Duration: 3},
		)
	}
	state := &simulator.CombatState{Seed: seed, Allies: []*simulator.CombatantAuto{ally}, Enemies: []*simulator.CombatantAuto{enemy}}
	logs := simulator.Simulate(state)
	return state, logs
}

func findParityLog(logs []simulator.CombatLog, action string) *simulator.CombatLog {
	marker := "Phẫn Nộ Đích Tiểu Điểu"
	if action == "SWORD_DRAGON_TRANSFORM" {
		marker = "hóa thân thành Thượng Cổ Kiếm Long"
	}
	if action == "BEAST_PHANTOM_STRIKE" {
		marker = "Thú Lực Hư Ảnh"
	}
	if action == "WORDS_ARE_FRIGHTENING" {
		marker = "Nhân Ngôn Khả Úy"
	}
	if action == "WIND_HASTE_BURST" {
		marker = "Tật Phong Tốc Độ"
	}
	if action == "EARTH_GRAVITY" {
		marker = "Địa Chấn Trọng Lực"
	}
	if action == "IGNITE_BURST" {
		marker = "Nộ Hỏa Bạo Phát"
	}
	if action == "SONG_OF_SEPARATION" {
		marker = "Khúc ca ly tán vạn vật"
	}
	if action == "SONG_OF_DESTINY" {
		marker = "Khúc ca vận mệnh chấn động thương khung"
	}
	if action == "IMMEASURABLE_QI_SEA" {
		marker = "Vô Lượng Khí Hải"
	}
	if action == "MYRIAD_DISMANTLE" {
		marker = "Vạn vật quy nhất, pháp tắc tiêu giải!"
	}
	if action == "FATE_VANQUISHED" {
		marker = "Mệnh Bại"
	}
	for index := range logs {
		if logs[index].Action == action || strings.Contains(logs[index].Detail, marker) {
			return &logs[index]
		}
	}
	return nil
}

func compareParityState(native, typed *simulator.CombatState) string {
	diff := ""
	for side := 0; side < 2; side++ {
		nativeSide, typedSide := native.Allies, typed.Allies
		if side == 1 {
			nativeSide, typedSide = native.Enemies, typed.Enemies
		}
		for i := range nativeSide {
			if i >= len(typedSide) {
				diff += fmt.Sprintf("side %d missing combatant %d in typed run\n", side, i)
				continue
			}
			diff += compareParityCombatant(nativeSide[i], typedSide[i])
		}
	}
	return diff
}

func compareParityCombatant(native, typed *simulator.CombatantAuto) string {
	diff := ""
	label := native.ID
	if native.HP != typed.HP {
		diff += fmt.Sprintf("%s HP native=%v typed=%v\n", label, native.HP, typed.HP)
	}
	if native.VirtualShield != typed.VirtualShield {
		diff += fmt.Sprintf("%s shield native=%v typed=%v\n", label, native.VirtualShield, typed.VirtualShield)
	}
	if native.Mana != typed.Mana {
		diff += fmt.Sprintf("%s mana native=%v typed=%v\n", label, native.Mana, typed.Mana)
	}
	if native.IsInvincible != typed.IsInvincible || native.IsLastStand != typed.IsLastStand || native.DamageTakenReduction != typed.DamageTakenReduction {
		diff += fmt.Sprintf("%s protections native=(invincible=%v last_stand=%v reduction=%v) typed=(invincible=%v last_stand=%v reduction=%v)\n", label, native.IsInvincible, native.IsLastStand, native.DamageTakenReduction, typed.IsInvincible, typed.IsLastStand, typed.DamageTakenReduction)
	}
	if got, want := parityStatuses(typed), parityStatuses(native); got != want {
		diff += fmt.Sprintf("%s statuses native=%s typed=%s\n", label, want, got)
	}
	return diff
}

// parityStatuses merges ActiveEffects and ActiveDoT into one multiset so a
// status that is a legacy marker on one side and a typed DoT on the other is
// still recognised as the same applied status. Duplicates are kept: STACK
// policy must reproduce the legacy append-per-application behaviour exactly.
func normalizeParityValue(id string, val int) int {
	switch strings.ToUpper(id) {
	case "STUN", "FREEZE", "SILENCE", "BIND", "FEAR", "SLEEP":
		if val == 0 {
			return 1
		}
	case "SWORD_MARK":
		if val == 0 {
			return 10
		}
	}
	return val
}

func parityStatuses(c *simulator.CombatantAuto) string {
	entries := make([]string, 0, len(c.ActiveEffects)+len(c.ActiveDoT))
	for _, effect := range c.ActiveEffects {
		id := strings.ToUpper(effect.ID)
		if id == "SWORD_DRAGON_FORM" {
			id = "ATK_UP"
		}
		entries = append(entries, fmt.Sprintf("%s:%d:%d", id, normalizeParityValue(id, effect.Value), effect.Duration))
	}
	for _, dot := range c.ActiveDoT {
		entries = append(entries, fmt.Sprintf("%s:%d:%d", strings.ToUpper(dot.Type), normalizeParityValue(dot.Type, int(dot.Value)), dot.Turns))
	}
	sort.Strings(entries)
	return fmt.Sprint(entries)
}
