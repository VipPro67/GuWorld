package gamefx

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
)

// ConstitutionDef describes a single constitution (thể chất) definition.
// The in-memory registry is loaded from the constitution_dictionary table
// (seeded from data/Constitution.csv) at startup and after admin imports.
type ConstitutionDef struct {
    Tier int `json:"tier"`
	Key     string  `json:"key"`
	Name    string  `json:"name"`
	Effects Effects `json:"effects"`
}

// ConstitutionRegistry holds the in-memory cache of constitution definitions.
var ConstitutionRegistry = make(map[string]ConstitutionDef)

// defaultConstitutions returns the built-in constitutions used as a fallback
// when the constitution_dictionary table is empty (e.g. in tests or before a
// CSV import). CSV/DB rows are the source of truth once loaded.
func defaultConstitutions() []ConstitutionDef {
	return []ConstitutionDef{
		{Key: "solar_sun_moon_body", Name: "Thái Nhật Dương Mãng Thể", Tier: 3, Effects: Effects{DaoNganBonus: map[string]float64{"light": 0.50}, PathDamageBonus: map[string]float64{"light": 0.20}, MaxManaBonus: 0.50, ManaRegenBonus: 1}},
		{Key: "ancient_moon_yin_desolate_body", Name: "Cổ Nguyệt Âm Hoang Thể", Tier: 3, Effects: Effects{DaoNganBonus: map[string]float64{"dark": 0.50}, MaxManaBonus: 0.50, ManaRegenBonus: 1, MentalResistanceBonus: 0.20}},
		{Key: "northern_dark_ice_soul_body", Name: "Bắc Minh Băng Phách Thể", Tier: 3, Effects: Effects{DaoNganBonus: map[string]float64{"ice": 0.50}, PathResistanceBonus: map[string]float64{"ice": 0.10}, MaxManaBonus: 0.50, ManaRegenBonus: 1}},
		{Key: "forest_sea_reincarnation_body", Name: "Sâm Hải Luân Hồi Thể", Tier: 3, Effects: Effects{DaoNganBonus: map[string]float64{"water": 0.50}, HealingReceivedBonus: 0.50, MaxManaBonus: 0.50, ManaRegenBonus: 1}},
		{Key: "myriad_gold_brilliant_flower_body", Name: "Vạn Kim Diệu Hoa Thể", Tier: 3, Effects: Effects{DaoNganBonus: map[string]float64{"metal": 0.50}, StatMods: map[string]float64{"attack": 0.30, "defense": 0.30}, MaxManaBonus: 0.50, ManaRegenBonus: 1}},
		{Key: "blazing_glory_lightning_brilliance_body", Name: "Viêm Hoàng Lôi Trạch Thể", Tier: 3, Effects: Effects{DaoNganBonus: map[string]float64{"lightning": 0.50}, StatMods: map[string]float64{"speed": 0.20}, MaxManaBonus: 0.50, ManaRegenBonus: 1}},
		{Key: "great_strength_true_martial_body", Name: "Đại Lực Chân Võ Thể", Tier: 3, Effects: Effects{DaoNganBonus: map[string]float64{"strength": 0.50}, StatMods: map[string]float64{"attack": 0.30, "defense": 0.30}, MaxManaBonus: 0.50, ManaRegenBonus: 1}},
		{Key: "carefree_wisdom_heart_body", Name: "Tiêu Dao Trí Tâm Thể", Tier: 3, Effects: Effects{DaoNganBonus: map[string]float64{"wisdom": 0.50}, MentalResistanceBonus: 0.30, MaxManaBonus: 0.50, ManaRegenBonus: 1}},
		{Key: "great_earth_origin_body", Name: "Hậu Thổ Nguyên Ương Thể", Tier: 3, Effects: Effects{DaoNganBonus: map[string]float64{"earth": 0.50}, StatMods: map[string]float64{"defense": 0.45}, MentalResistanceBonus: 0.45, MaxManaBonus: 0.50, ManaRegenBonus: 1}},
		{Key: "universe_great_evolution_body", Name: "Vũ Trụ Đại Diễn Thể", Tier: 3, Effects: Effects{DaoNganBonus: map[string]float64{"space": 0.50}, StatMods: map[string]float64{"attack": 0.15, "defense": 0.15, "speed": 0.15}, MaxManaBonus: 0.50, ManaRegenBonus: 1}},
		{Key: "pure_dream_reality_seeker_body", Name: "Thuần Mộng Cầu Chân Thể", Tier: 3, Effects: Effects{DaoNganBonus: map[string]float64{"dream": 0.40}, PathDamageBonus: map[string]float64{"dream": 0.10}, MaxManaBonus: 0.40, ManaRegenBonus: 1}},
	}
}

// effectiveConstitutions returns the registry, falling back to the built-in
// defaults when it has not been loaded from the database yet.
func effectiveConstitutions() map[string]ConstitutionDef {
	if len(ConstitutionRegistry) > 0 {
		return ConstitutionRegistry
	}
	fallback := make(map[string]ConstitutionDef, len(defaultConstitutions()))
	for _, def := range defaultConstitutions() {
		fallback[def.Key] = def
	}
	return fallback
}

// LoadConstitutionRegistry queries the database and populates the in-memory
// cache. When the table is empty the built-in defaults are used.
func LoadConstitutionRegistry(db *sql.DB) error {
	// Empty-key sentinel rows exist only to keep the players.constitution FK
	// satisfiable for the '' value; they must never surface in the registry.
	rows, err := db.Query(`SELECT key, name, COALESCE(effects::text,'{}'), tier FROM constitution_dictionary WHERE key <> '' ORDER BY key`)
	if err != nil {
		return err
	}
	defer rows.Close()

	newRegistry := make(map[string]ConstitutionDef)
	for rows.Next() {
		var def ConstitutionDef
		var effectsRaw string
		var tier int
if err := rows.Scan(&def.Key, &def.Name, &effectsRaw, &tier); err != nil {
			return fmt.Errorf("scan constitution row: %w", err)
		}
		if err := json.Unmarshal([]byte(effectsRaw), &def.Effects); err != nil {
			return fmt.Errorf("constitution %s effects: %w", def.Key, err)
		}
		newRegistry[def.Key] = def
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(newRegistry) == 0 {
		for _, def := range defaultConstitutions() {

		newRegistry[def.Key] = def
		}
	}
	ConstitutionRegistry = newRegistry
	return nil
}

// AllConstitutions returns the full list of constitution definitions,
// ordered by key.
func AllConstitutions() []ConstitutionDef {
	defs := effectiveConstitutions()
	out := make([]ConstitutionDef, 0, len(defs))
	for _, def := range defs {
		out = append(out, def)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// addEffects merges src into dst, adding every modifier.
func addEffects(dst *Effects, src Effects) {
	if dst.StatMods == nil {
		dst.StatMods = map[string]float64{}
	}
	for k, v := range src.StatMods {
		dst.StatMods[k] += v
	}
	dst.RefineBonus += src.RefineBonus
	if dst.DaoNganBonus == nil {
		dst.DaoNganBonus = map[string]float64{}
	}
	for k, v := range src.DaoNganBonus {
		dst.DaoNganBonus[k] += v
	}
	if dst.PathDamageBonus == nil {
		dst.PathDamageBonus = map[string]float64{}
	}
	for k, v := range src.PathDamageBonus {
		dst.PathDamageBonus[k] += v
	}
	if dst.PathResistanceBonus == nil {
		dst.PathResistanceBonus = map[string]float64{}
	}
	for k, v := range src.PathResistanceBonus {
		dst.PathResistanceBonus[k] += v
	}
	dst.LuckBonus += src.LuckBonus
	dst.MaxManaBonus += src.MaxManaBonus
	dst.ManaRegenBonus += src.ManaRegenBonus
	dst.CultivationEfficiencyBonus += src.CultivationEfficiencyBonus
	dst.DaoMarkGainBonus += src.DaoMarkGainBonus
	dst.HealingReceivedBonus += src.HealingReceivedBonus
	dst.HPRegenBonus += src.HPRegenBonus
	dst.MentalResistanceBonus += src.MentalResistanceBonus
	dst.SoulDefenseBonus += src.SoulDefenseBonus
	dst.SoulRegenBonus += src.SoulRegenBonus
	dst.ApertureStressBonus += src.ApertureStressBonus
	if dst.BlockedActions == nil {
		dst.BlockedActions = map[string]bool{}
	}
	for action := range src.BlockedActions {
		dst.BlockedActions[action] = true
	}
}
