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
	return []StatusDef{
		{
			Key:  "severely_wounded",
			Name: "Trọng Thương",
			Effects: Effects{
				StatMods:             map[string]float64{"attack": -0.20, "defense": -0.20, "speed": -0.20},
				HealingReceivedBonus: -0.50,
			},
			DefaultDurationMinutes: 60,
			BlockedActions:         []string{ActionExplore},
			Description:            "Thân trúng trọng thương, công thủ tốc đều giảm mạnh.",
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
				RefineBonus: -0.25,
			},
			DefaultDurationMinutes: 10,
			Description:            "Luyện cổ khiến tâm thần mệt mỏi, giảm 25% tỉ lệ luyện cổ thành công.",
		},
		{
			Key:  "refine_backfire",
			Name: "Luyện Cổ Phản Phệ",
			Effects: Effects{
				RefineBonus: -0.50,
			},
			DefaultDurationMinutes: 20,
			Description:            "Bị cổ khí phản phệ do cố luyện cổ khi mệt mỏi, giảm 50% tỉ lệ luyện cổ thành công.",
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
	}
}

// EffectsForValue returns the status effects adjusted for the instance's
// severity Value. A negative Value (rank-scaled suppression statuses store
// their intensity there, e.g. regional_wall_suppression) replaces every
// stat_mod percentage with the Value itself.
func (def StatusDef) EffectsForValue(value float64) Effects {
	eff := def.Effects
	if value < 0 && len(eff.StatMods) > 0 {
		mods := make(map[string]float64, len(eff.StatMods))
		for k := range eff.StatMods {
			mods[k] = value
		}
		eff.StatMods = mods
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
