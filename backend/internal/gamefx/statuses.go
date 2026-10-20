package gamefx

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

// Canonical blocked-action keys used by status definitions. Handlers consult
// them via HasBlockedAction before executing the corresponding gameplay flow.
// Only explore and cultivate are currently enforced by handlers; combat,
// refine and trade keys are accepted in data but no handler gates on them yet
// (adding enforcement is a gameplay change, not a data fix).
const (
	ActionExplore   = "explore"
	ActionCultivate = "cultivate"
	ActionCombat    = "combat"
	ActionRefine    = "refine"
	ActionTrade     = "trade"
	// ActionBreakthrough gates every rank-up path (BreakWall / Ascension /
	// UseRelicGu and RANK_PROGRESS / MAN_BEAST_BURIAL / FORCE_RANK_UP).
	ActionBreakthrough = "breakthrough"
)

// StatusDef describes a single persistent player status (trạng thái). The
// in-memory registry is loaded from the status_dictionary table (seeded from
// data/Status.csv) at startup and after admin imports.
type StatusDef struct {
	Key     string  `json:"key"`
	Name    string  `json:"name"`
	Effects Effects `json:"effects"`
	// DefaultDurationMinutes is applied when a status is granted without an
	// explicit duration. 0 means permanent until removed.
	DefaultDurationMinutes int      `json:"default_duration_minutes"`
	BlockedActions         []string `json:"blocked_actions"`
	Description            string   `json:"description"`
}

// StatusRegistry holds the in-memory cache of status definitions.
var StatusRegistry = make(map[string]StatusDef)

// defaultStatuses returns the built-in statuses used as a fallback when the
// status_dictionary table is empty (e.g. in tests or before a CSV import).
// CSV/DB rows are the source of truth once loaded.
func defaultStatuses() []StatusDef {
	defs := []StatusDef{
		{
			Key:  "severely_wounded",
			Name: "Trọng Thương",
			Effects: Effects{
				StatMods:             map[string]float64{"attack": -0.20, "defense": -0.20, "speed": -0.20},
				HealingReceivedBonus: -0.50,
				RefineBonus:          -0.20,
			},
			DefaultDurationMinutes: 60,
			BlockedActions:         []string{ActionExplore},
			Description:            "Thân trúng trọng thương, công thủ tốc và tỉ lệ luyện cổ đều suy giảm theo cường độ phản phệ.",
		},
		{
			Key:  "poisoned",
			Name: "Trúng Độc",
			Effects: Effects{
				StatMods: map[string]float64{"attack": -0.10, "speed": -0.10},
			},
			DefaultDurationMinutes: 30,
			Description:            "Độc tố xâm nhập cơ thể, suy giảm công kích và tốc độ.",
		},
		{
			Key:  "exhausted",
			Name: "Kiệt Sức",
			Effects: Effects{
				StatMods:                   map[string]float64{"speed": -0.30},
				CultivationEfficiencyBonus: -0.50,
			},
			DefaultDurationMinutes: 10,
			BlockedActions:         []string{ActionCultivate},
			Description:            "Lực kiệt sức, tạm thời không thể tu luyện.",
		},
		{
			Key:  "blessed",
			Name: "Phúc Trạch",
			Effects: Effects{
				LuckBonus:                  5,
				CultivationEfficiencyBonus: 0.10,
			},
			DefaultDurationMinutes: 60,
			Description:            "Nhận phúc trạch, may mắn và tu luyện hiệu quả hơn.",
		},
		{
			Key:  "death_penalty",
			Name: "Hình Phạt Tử Vong",
			Effects: Effects{
				StatMods: map[string]float64{"attack": -0.50, "defense": -0.50, "speed": -0.50},
			},
			DefaultDurationMinutes: 30,
			Description:            "Sát khí chưa tan sau khi chết, toàn bộ chỉ số suy giảm.",
		},
		{
			Key:  "refine_fatigue",
			Name: "Tâm Thần Mệt Mỏi",
			Effects: Effects{
				RefineBonus: -0.50,
			},
			DefaultDurationMinutes: 10,
			Description:            "Luyện cổ khiến tâm thần mệt mỏi, giảm 50% tỉ lệ luyện cổ thành công.",
		},
		{
			Key:  "refine_backfire",
			Name: "Luyện Cổ Phản Phệ",
			Effects: Effects{
				RefineBonus: -0.95,
			},
			DefaultDurationMinutes: 20,
			Description:            "Bị cổ khí phản phệ do cố luyện cổ khi mệt mỏi, giảm 95% tỉ lệ luyện cổ thành công.",
		},
		{
			Key:  "regional_wall_suppression",
			Name: "Áp Chế Giới Vực",
			Effects: Effects{
				StatMods: map[string]float64{
					"attack":  -0.20,
					"defense": -0.20,
					"speed":   -0.20,
				},
			},
			DefaultDurationMinutes: 15,
			Description:            "Chịu áp chế khi xuyên qua Giới Vực Bích giữa các Đại Vực. Công thủ tốc đều suy giảm (tu vi rank càng cao bị áp chế càng nặng).",
		},
		{
			// Không Khiếu Hóa Đá: Thạch Khiếu Cổ đẩy tu vi lên Đỉnh Phong nhưng
			// hóa đá Không Khiếu một thời gian (12h × 2^(rank-1)) — không tự hồi
			// Chân Nguyên, không thể đột phá lên cảnh giới cao hơn.
			Key:  StatusAperturePetrified,
			Name: "Không Khiếu Hóa Đá",
			Effects: Effects{
				DisableManaRegen: true,
			},
			DefaultDurationMinutes: 720,
			BlockedActions:         []string{ActionBreakthrough},
			Description:            "Không Khiếu hóa đá: không thể tự hồi Chân Nguyên và không thể đột phá lên tu vi cao hơn. Tự hồi phục sau 12 giờ × 2^(rank-1) (R1 12h, R2 24h, R3 48h, ...).",
		},
		{
			// Sinh Khí Tàn: persistent PvP debuff do Sát Sinh Khí Cổ (GU_KILL_ENERGY) gây ra.
			// Cường độ nằm ở StatusInstance.Value (âm) và được EffectsForValue dùng để
			// ghi đè toàn bộ stat_mods — cùng cơ chế với Áp Chế Giới Vực.
			// Giá trị mặc định -0.10 tương ứng tầng I. Duration tạm 300 phút (5h);
			// spec gốc 50h (3000 phút) đang tạm giảm để cân bằng, mở lại bằng cách
			// đổi DefaultDurationMinutes và SinhKhiTanDuration.
			Key:  StatusSinhKhiTan,
			Name: "Sinh Khí Tàn",
			Effects: Effects{
				StatMods: map[string]float64{
					"attack":  -0.10,
					"defense": -0.10,
					"speed":   -0.10,
				},
			},
			DefaultDurationMinutes: 300,
			Description:            "Trúng Sát Sinh Khí Cổ, sinh cơ tàn lụi theo tầng (I -10%, II -25%, III -35%, IV -50%, V -99% công/thủ/tốc). Mỗi lần đánh trúng +1 tầng, thanh tẩy -1 tầng.",
		},
	}
	for rank := 1; rank <= 9; rank++ {
		defs = append(defs, StatusDef{
			Key:                    RefineInsightKey(rank),
			Name:                   "Luyện Cổ Tâm Đắc",
			Effects:                Effects{},
			DefaultDurationMinutes: 0,
			Description:            "Mỗi lần luyện cổ thất bại tích 1 tầng (+1% tỉ lệ cho đúng rank đó, không giới hạn). Luyện thành công rank đó thì reset về 0.",
		})
	}
	return defs
}

// SinhKhiTan là key persistent của trạng thái Sinh Khí Tàn (Sát Sinh Khí Cổ).
const StatusSinhKhiTan = "sinh_khi_tan"

// StatusAperturePetrified là key persistent của trạng thái Không Khiếu Hóa Đá
// (Thạch Khiếu Cổ và họ Cổ Thổ đạo cùng cơ chế): không tự hồi Chân Nguyên và
// không thể đột phá lên tu vi cao hơn.
const StatusAperturePetrified = "aperture_petrified"

// AperturePetrifiedDuration trả thời hạn hóa đá theo rank: R1 12 giờ, mỗi
// rank sau gấp đôi (R2 24h, R3 48h, ... R9 3072h).
func AperturePetrifiedDuration(rank int) time.Duration {
	if rank < 1 {
		rank = 1
	}
	if rank > 9 {
		rank = 9
	}
	return 12 * time.Hour << uint(rank-1)
}

// SinhKhiTanDuration là thời hạn mỗi lần đánh trúng refresh (tạm 5h, spec gốc 50h).
const SinhKhiTanDurationMinutes = 300

// SinhKhiTanTierValues maps tầng 1..5 sang cường độ âm (dùng cho StatusInstance.Value).
var SinhKhiTanTierValues = map[int]float64{
	1: -0.10,
	2: -0.25,
	3: -0.35,
	4: -0.50,
	5: -0.99,
}

// SinhKhiTanMaxTier là số tầng tối đa.
const SinhKhiTanMaxTier = 5

// SinhKhiTanTierFromValue maps cường độ Value hiện tại về tầng 0..5 (0 = chưa có).
// So sánh theo ngưỡng giữa các tầng để chịu được sai số float.
func SinhKhiTanTierFromValue(value float64) int {
	if value >= 0 {
		return 0
	}
	v := -value
	switch {
	case v >= 0.99:
		return 5
	case v >= 0.50:
		return 4
	case v >= 0.35:
		return 3
	case v >= 0.25:
		return 2
	case v >= 0.10:
		return 1
	default:
		return 1
	}
}

// SinhKhiTanValueForTier trả cường độ của tầng, clamp 1..5.
func SinhKhiTanValueForTier(tier int) float64 {
	if tier < 1 {
		tier = 1
	}
	if tier > SinhKhiTanMaxTier {
		tier = SinhKhiTanMaxTier
	}
	return SinhKhiTanTierValues[tier]
}

// SinhKhiTanTierName trả tên hiển thị La Mã của tầng.
func SinhKhiTanTierName(tier int) string {
	switch tier {
	case 1:
		return "Sinh Khí Tàn I"
	case 2:
		return "Sinh Khí Tàn II"
	case 3:
		return "Sinh Khí Tàn III"
	case 4:
		return "Sinh Khí Tàn IV"
	case 5:
		return "Sinh Khí Tàn V"
	default:
		return "Sinh Khí Tàn"
	}
}

// RefineInsightKey returns the per-rank pity status key.
// Each rank has its own stack: fail rank R only buffs rank R.
func RefineInsightKey(rank int) string {
	if rank < 1 {
		rank = 1
	}
	if rank > 9 {
		rank = 9
	}
	return fmt.Sprintf("refine_insight_%d", rank)
}

// EffectsForValue returns the status effects adjusted for the instance's
// severity Value. A negative Value (rank-scaled suppression statuses store
// their intensity there, e.g. regional_wall_suppression) replaces every
// stat_mod percentage with the Value itself. For rank-scaled severe wounds,
// the same intensity also scales the Refinement penalty.
func (def StatusDef) EffectsForValue(value float64) Effects {
	eff := def.Effects
	if value < 0 && len(eff.StatMods) > 0 {
		mods := make(map[string]float64, len(eff.StatMods))
		for k := range eff.StatMods {
			mods[k] = value
		}
		eff.StatMods = mods
	}
	if value < 0 && def.Key == "severely_wounded" {
		eff.RefineBonus = value
	}
	return eff
}

// effectiveStatuses returns the registry, falling back to the built-in
// defaults when it has not been loaded from the database yet.
func effectiveStatuses() map[string]StatusDef {
	if len(StatusRegistry) > 0 {
		return StatusRegistry
	}
	fallback := make(map[string]StatusDef, len(defaultStatuses()))
	for _, def := range defaultStatuses() {
		fallback[def.Key] = def
	}
	return fallback
}

// LoadStatusRegistry queries the database and populates the in-memory cache.
// When the table is empty the built-in defaults are used.
func LoadStatusRegistry(db *sql.DB) error {
	rows, err := db.Query(`SELECT key, name, COALESCE(effects::text,'{}'), COALESCE(default_duration_minutes,0), COALESCE(blocked_actions::text,'[]'), COALESCE(description,'') FROM status_dictionary ORDER BY key`)
	if err != nil {
		return err
	}
	defer rows.Close()

	newRegistry := make(map[string]StatusDef)
	for rows.Next() {
		var def StatusDef
		var effectsRaw, blockedRaw string
		if err := rows.Scan(&def.Key, &def.Name, &effectsRaw, &def.DefaultDurationMinutes, &blockedRaw, &def.Description); err != nil {
			return fmt.Errorf("scan status row: %w", err)
		}
		if err := json.Unmarshal([]byte(effectsRaw), &def.Effects); err != nil {
			return fmt.Errorf("status %s effects: %w", def.Key, err)
		}
		if err := json.Unmarshal([]byte(blockedRaw), &def.BlockedActions); err != nil {
			return fmt.Errorf("status %s blocked_actions: %w", def.Key, err)
		}
		newRegistry[def.Key] = def
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(newRegistry) == 0 {
		for _, def := range defaultStatuses() {
			newRegistry[def.Key] = def
		}
	}
	StatusRegistry = newRegistry
	return nil
}

// AllStatuses returns the full list of status definitions, ordered by key.
func AllStatuses() []StatusDef {
	defs := effectiveStatuses()
	out := make([]StatusDef, 0, len(defs))
	for _, def := range defs {
		out = append(out, def)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// ValidStatuses returns the set of accepted status keys.
func ValidStatuses() map[string]bool {
	defs := effectiveStatuses()
	out := make(map[string]bool, len(defs))
	for key := range defs {
		out[key] = true
	}
	return out
}

// StatusDefaultDuration returns the registry duration for a status key
// (0 when unknown or permanent).
func StatusDefaultDuration(key string) int {
	if def, ok := effectiveStatuses()[key]; ok {
		return def.DefaultDurationMinutes
	}
	return 0
}

// BlockingStatuses returns the active (non-expired) status definitions that
// block the given action, so callers can build precise error messages.
func BlockingStatuses(p *models.Player, action string) []StatusDef {
	out := []StatusDef{}
	if p == nil || len(p.Statuses) == 0 {
		return out
	}
	now := time.Now()
	for _, st := range p.Statuses {
		if st.ExpiresAt != nil && !st.ExpiresAt.After(now) {
			continue
		}
		def, ok := effectiveStatuses()[st.ID]
		if !ok {
			continue
		}
		for _, a := range def.BlockedActions {
			if a == action {
				out = append(out, def)
				break
			}
		}
	}
	return out
}
