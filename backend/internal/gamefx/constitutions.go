package gamefx

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// ConstitutionDef describes a single constitution (thể chất) definition.
// The in-memory registry is loaded from the constitution_dictionary table
// (seeded from data/Constitution.csv) at startup and after admin imports.
type ConstitutionDef struct {
	Tier    int     `json:"tier"`
	Key     string  `json:"key"`
	Name    string  `json:"name"`
	Effects Effects `json:"effects"`
}

// ConstitutionRegistry holds the in-memory cache of constitution definitions.
var ConstitutionRegistry = make(map[string]ConstitutionDef)

// fallbackConstitutions loads constitution definitions from Constitution.csv
// when the in-memory registry is unpopulated (e.g. during standalone unit tests).
func fallbackConstitutions() map[string]ConstitutionDef {
	fallback := make(map[string]ConstitutionDef)
	path := csvPath("Constitution.csv")
	f, err := os.Open(path)
	if err != nil {
		return fallback
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil {
		return fallback
	}
	keyIdx, nameIdx, effectsIdx, tierIdx := -1, -1, -1, -1
	for i, col := range header {
		switch strings.TrimSpace(strings.ToLower(col)) {
		case "key":
			keyIdx = i
		case "name":
			nameIdx = i
		case "effects":
			effectsIdx = i
		case "tier":
			tierIdx = i
		}
	}
	if keyIdx == -1 || nameIdx == -1 {
		return fallback
	}

	for {
		rec, err := r.Read()
		if err != nil {
			break
		}
		if len(rec) <= keyIdx {
			continue
		}
		key := strings.TrimSpace(rec[keyIdx])
		if key == "" {
			continue
		}
		name := ""
		if nameIdx >= 0 && nameIdx < len(rec) {
			name = strings.TrimSpace(rec[nameIdx])
		}
		tier := 0
		if tierIdx >= 0 && tierIdx < len(rec) {
			if v, err := strconv.Atoi(strings.TrimSpace(rec[tierIdx])); err == nil {
				tier = v
			}
		}
		var eff Effects
		if effectsIdx >= 0 && effectsIdx < len(rec) {
			raw := strings.TrimSpace(rec[effectsIdx])
			if raw != "" {
				_ = json.Unmarshal([]byte(raw), &eff)
			}
		}
		fallback[key] = ConstitutionDef{
			Key:     key,
			Name:    name,
			Tier:    tier,
			Effects: eff,
		}
	}
	return fallback
}

// effectiveConstitutions returns the registry, falling back to the CSV dataset
// when it has not been loaded from the database yet.
func effectiveConstitutions() map[string]ConstitutionDef {
	if len(ConstitutionRegistry) > 0 {
		return ConstitutionRegistry
	}
	return fallbackConstitutions()
}

// LoadConstitutionRegistry queries the database and populates the in-memory
// cache. When the table is empty the CSV dataset is used as fallback.
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
		newRegistry = fallbackConstitutions()
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
	dst.Tenacity += src.Tenacity
	dst.CCResist += src.CCResist
	dst.SoulRegenBonus += src.SoulRegenBonus
	dst.ApertureStressBonus += src.ApertureStressBonus
	if dst.NonConflictingDaoMarks || src.NonConflictingDaoMarks {
		dst.NonConflictingDaoMarks = true
		dst.DaoConflictReductionPct = 100
	} else {
		dst.DaoConflictReductionPct = CombineDaoConflictReduction(dst.DaoConflictReductionPct, src.DaoConflictReductionPct)
	}
	if src.Immortal {
		dst.Immortal = true
	}
	if src.DisableHPRegen {
		dst.DisableHPRegen = true
	}
	if src.DisableManaRegen {
		dst.DisableManaRegen = true
	}
	if src.DisableResourceRegen {
		dst.DisableResourceRegen = true
	}
	if src.DisableSoulRegen {
		dst.DisableSoulRegen = true
	}
	if dst.BlockedActions == nil {
		dst.BlockedActions = map[string]bool{}
	}
	for action := range src.BlockedActions {
		dst.BlockedActions[action] = true
	}
}
