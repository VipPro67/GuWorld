package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type RequiredGusMap map[string]int
type RequiredMaterialsMap map[string]int

type RefiningRecipe struct {
	ID                       string               `json:"id"`
	TargetGuID               string               `json:"target_gu_id"`
	RequiredGus              RequiredGusMap       `json:"required_gus"`
	RequiredMaterials        RequiredMaterialsMap `json:"required_materials"`
	SuccessRate              float64              `json:"success_rate"`
	PrimevalStonesCost       int                  `json:"primeval_stones_cost"`
	ImmortalEssenceStonesCost float64             `json:"immortal_essence_stones_cost"`
}

// Scan implements the sql.Scanner interface for RequiredGusMap
func (rg *RequiredGusMap) Scan(value interface{}) error {
	if value == nil {
		*rg = RequiredGusMap{}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte, got %T", value)
	}
	return json.Unmarshal(b, rg)
}

// Value implements the driver.Valuer interface for RequiredGusMap
func (rg RequiredGusMap) Value() (driver.Value, error) {
	if rg == nil {
		return "{}", nil
	}
	return json.Marshal(rg)
}

// Scan implements the sql.Scanner interface for RequiredMaterialsMap
func (rm *RequiredMaterialsMap) Scan(value interface{}) error {
	if value == nil {
		*rm = RequiredMaterialsMap{}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte, got %T", value)
	}
	return json.Unmarshal(b, rm)
}

// Value implements the driver.Valuer interface for RequiredMaterialsMap
func (rm RequiredMaterialsMap) Value() (driver.Value, error) {
	if rm == nil {
		return "{}", nil
	}
	return json.Marshal(rm)
}
