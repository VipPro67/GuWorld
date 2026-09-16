package models

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// RequiredGuItem describes one material Gu required by a refining recipe.
// Rank 0 means "any rank" (used for legacy recipes that predate ranked
// materials). Qty is the number of such Gu instances required.
type RequiredGuItem struct {
	Rank int `json:"rank"`
	Qty  int `json:"qty"`
}

// RequiredGusMap maps a material Gu id to its requirement. It accepts both the
// ranked JSON form {"GU_ID": {"rank": N, "qty": Q}} and the legacy form
// {"GU_ID": Q} (parsed as {rank: 0, qty: Q}).
type RequiredGusMap map[string]RequiredGuItem
type RequiredMaterialsMap map[string]int64

type RefiningRecipe struct {
	ID                        string               `json:"id"`
	TargetGuID                string               `json:"target_gu_id"`
	TargetRank                int                  `json:"target_rank"`
	Variant                   int                  `json:"variant"`
	RequiredGus               RequiredGusMap       `json:"required_gus"`
	RequiredMaterials         RequiredMaterialsMap `json:"required_materials"`
	SuccessRate               float64              `json:"success_rate"`
	PrimevalStonesCost        float64              `json:"primeval_stones_cost"`
	ImmortalEssenceStonesCost float64              `json:"immortal_essence_stones_cost"`
	CompletionRate            float64              `json:"completion_rate"`
}

type UnlockedRecipe struct {
	RecipeID       string  `json:"recipe_id"`
	CompletionRate float64 `json:"completion_rate"`
}

type UnlockedRecipesList []UnlockedRecipe

// SectVaultRecipe represents a recipe stored in a sect vault, carrying its
// completion rate so donated/purchased recipes retain their progress value.
type SectVaultRecipe struct {
	RecipeID       string  `json:"recipe_id"`
	CompletionRate float64 `json:"completion_rate"`
}

// Scan implements the sql.Scanner interface for UnlockedRecipesList
func (ur *UnlockedRecipesList) Scan(value interface{}) error {
	if value == nil {
		*ur = UnlockedRecipesList{}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte, got %T", value)
	}
	return json.Unmarshal(b, ur)
}

// Value implements the driver.Valuer interface for UnlockedRecipesList
func (ur UnlockedRecipesList) Value() (driver.Value, error) {
	if ur == nil {
		return "[]", nil
	}
	return json.Marshal(ur)
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
	return rg.UnmarshalJSON(b)
}

// UnmarshalJSON accepts both the ranked object form and the legacy numeric form.
func (rg *RequiredGusMap) UnmarshalJSON(data []byte) error {
	raw := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &raw); err != nil {
		*rg = RequiredGusMap{}
		return err
	}
	out := RequiredGusMap{}
	for k, v := range raw {
		trimmed := bytes.TrimSpace(v)
		if len(trimmed) > 0 && trimmed[0] == '{' {
			var item RequiredGuItem
			if err := json.Unmarshal(trimmed, &item); err != nil {
				continue
			}
			if item.Qty <= 0 {
				item.Qty = 1
			}
			out[k] = item
			continue
		}
		var qty int
		if err := json.Unmarshal(trimmed, &qty); err != nil {
			continue
		}
		if qty <= 0 {
			qty = 1
		}
		out[k] = RequiredGuItem{Qty: qty}
	}
	*rg = out
	return nil
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
