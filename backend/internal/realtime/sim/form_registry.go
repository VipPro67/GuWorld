package sim

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
)

// TransformationFormRecord represents a parsed row from TransformationForm.csv.
type TransformationFormRecord struct {
	FormID              string             `json:"form_id"`
	Name                string             `json:"name"`
	FormKind            TransformationKind `json:"form_kind"`
	ModelScale          float64            `json:"model_scale"`
	CollisionRadiusMult float64            `json:"collision_radius_mult"`
	MaxFormHP           float64            `json:"max_form_hp"`
	FormDefense         float64            `json:"form_defense"`
	DamageReduction     float64            `json:"damage_reduction"`
	ManaUpkeepSec       float64            `json:"mana_upkeep_sec"`
	WindupSec           float64            `json:"windup_sec"`
	SkillsJSON          string             `json:"skills_json"`
	Description         string             `json:"description"`
}

var (
	formRegistryMu sync.RWMutex
	formRegistry   = make(map[string]TransformationFormRecord)
)

// LoadTransformationForms loads and parses TransformationForm.csv from the given path or default location.
func LoadTransformationForms(csvPath string) error {
	if csvPath == "" {
		candidates := []string{
			"data/TransformationForm.csv",
			"backend/data/TransformationForm.csv",
			"../data/TransformationForm.csv",
			"../../data/TransformationForm.csv",
			"../../../data/TransformationForm.csv",
		}
		for _, cand := range candidates {
			if _, err := os.Stat(cand); err == nil {
				csvPath = cand
				break
			}
		}
	}

	file, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("open transformation forms csv: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("read transformation forms header: %w", err)
	}

	colIdx := make(map[string]int)
	for i, col := range header {
		colIdx[strings.TrimSpace(col)] = i
	}

	requiredCols := []string{"key", "name", "form_kind", "model_scale", "collision_radius_mult", "max_form_hp", "form_defense", "damage_reduction", "mana_upkeep_sec", "windup_sec", "skills_json", "description"}
	for _, col := range requiredCols {
		if _, ok := colIdx[col]; !ok {
			return fmt.Errorf("missing required column %q in TransformationForm.csv", col)
		}
	}

	newRegistry := make(map[string]TransformationFormRecord)
	line := 1
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read line %d: %w", line, err)
		}
		line++

		key := strings.TrimSpace(record[colIdx["key"]])
		if key == "" {
			continue
		}

		modelScale, _ := strconv.ParseFloat(record[colIdx["model_scale"]], 64)
		if modelScale <= 0 {
			modelScale = 1.0
		}
		radiusMult, _ := strconv.ParseFloat(record[colIdx["collision_radius_mult"]], 64)
		if radiusMult <= 0 {
			radiusMult = 1.0
		}
		maxHP, _ := strconv.ParseFloat(record[colIdx["max_form_hp"]], 64)
		defense, _ := strconv.ParseFloat(record[colIdx["form_defense"]], 64)
		damageRed, _ := strconv.ParseFloat(record[colIdx["damage_reduction"]], 64)
		upkeep, _ := strconv.ParseFloat(record[colIdx["mana_upkeep_sec"]], 64)
		windup, _ := strconv.ParseFloat(record[colIdx["windup_sec"]], 64)

		formRecord := TransformationFormRecord{
			FormID:              key,
			Name:                strings.TrimSpace(record[colIdx["name"]]),
			FormKind:            TransformationKind(strings.TrimSpace(record[colIdx["form_kind"]])),
			ModelScale:          modelScale,
			CollisionRadiusMult: radiusMult,
			MaxFormHP:           maxHP,
			FormDefense:         defense,
			DamageReduction:     damageRed,
			ManaUpkeepSec:       upkeep,
			WindupSec:           windup,
			SkillsJSON:          record[colIdx["skills_json"]],
			Description:         strings.TrimSpace(record[colIdx["description"]]),
		}

		// Validate skills_json is valid JSON
		if formRecord.SkillsJSON != "" {
			var testJSON interface{}
			if err := json.Unmarshal([]byte(formRecord.SkillsJSON), &testJSON); err != nil {
				return fmt.Errorf("form %s has invalid skills_json: %w", key, err)
			}
		}

		newRegistry[key] = formRecord
	}

	formRegistryMu.Lock()
	formRegistry = newRegistry
	formRegistryMu.Unlock()

	return nil
}

// GetTransformationForm retrieves a cached transformation form record.
func GetTransformationForm(formID string) (TransformationFormRecord, bool) {
	formRegistryMu.RLock()
	defer formRegistryMu.RUnlock()
	rec, ok := formRegistry[formID]
	return rec, ok
}

// CountTransformationForms returns the number of registered forms.
func CountTransformationForms() int {
	formRegistryMu.RLock()
	defer formRegistryMu.RUnlock()
	return len(formRegistry)
}
