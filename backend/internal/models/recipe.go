package models

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
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
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("unsupported Scan, expected []byte or string, got %T", value)
	}
	return json.Unmarshal(b, ur)
}

// Value implements the driver.Valuer interface for UnlockedRecipesList
func (ur UnlockedRecipesList) Value() (driver.Value, error) {
	if ur == nil {
		return "[]", nil
	}
	b, err := json.Marshal(ur)
	if err != nil {
		return "[]", err
	}
	return string(b), nil
}

// Scan implements the sql.Scanner interface for RequiredGusMap
func (rg *RequiredGusMap) Scan(value interface{}) error {
	if value == nil {
		*rg = RequiredGusMap{}
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("unsupported Scan, expected []byte or string, got %T", value)
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
	b, err := json.Marshal(rg)
	if err != nil {
		return "{}", err
	}
	return string(b), nil
}

// Scan implements the sql.Scanner interface for RequiredMaterialsMap
func (rm *RequiredMaterialsMap) Scan(value interface{}) error {
	if value == nil {
		*rm = RequiredMaterialsMap{}
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("unsupported Scan, expected []byte or string, got %T", value)
	}
	return json.Unmarshal(b, rm)
}

// Value implements the driver.Valuer interface for RequiredMaterialsMap
func (rm RequiredMaterialsMap) Value() (driver.Value, error) {
	if rm == nil {
		return "{}", nil
	}
	b, err := json.Marshal(rm)
	if err != nil {
		return "{}", err
	}
	return string(b), nil
}

// ParseRecipeID parses recipe_GU_ID_RANK_VARIANT into components.
// E.g., recipe_GU_BIG_BELLY_TOAD_1_1 -> target_gu_id = "GU_BIG_BELLY_TOAD", target_rank = 1, variant = 1
func ParseRecipeID(recipeID string) (string, int, int, error) {
	s := strings.TrimPrefix(recipeID, "recipe_")
	parts := strings.Split(s, "_")
	if len(parts) < 3 {
		return "", 0, 0, fmt.Errorf("invalid recipe id format: %s", recipeID)
	}
	variantStr := parts[len(parts)-1]
	rankStr := parts[len(parts)-2]
	targetGuID := strings.Join(parts[:len(parts)-2], "_")

	var rank, variant int
	if _, err := fmt.Sscanf(rankStr, "%d", &rank); err != nil {
		return "", 0, 0, fmt.Errorf("invalid rank in recipe id: %s", recipeID)
	}
	if _, err := fmt.Sscanf(variantStr, "%d", &variant); err != nil {
		return "", 0, 0, fmt.Errorf("invalid variant in recipe id: %s", recipeID)
	}
	return targetGuID, rank, variant, nil
}

var (
	recipeMu       sync.RWMutex
	RecipeRegistry = make(map[string]RefiningRecipe)
	RecipeByTarget = make(map[string]map[int][]RefiningRecipe)
	recipeList     = make([]RefiningRecipe, 0)
)

// LoadRecipeRegistry loads all refining_recipes from the database into memory cache.
func LoadRecipeRegistry(db *sql.DB) error {
	if db == nil {
		return nil
	}
	rows, err := db.Query(`
		SELECT target_gu_id, target_rank, variant, required_gus, required_materials,
		       success_rate, primeval_stones_cost, immortal_essence_stones_cost
		FROM refining_recipes
		ORDER BY target_gu_id, target_rank, variant`)
	if err != nil {
		return err
	}
	defer rows.Close()

	newRegistry := make(map[string]RefiningRecipe)
	newByTarget := make(map[string]map[int][]RefiningRecipe)
	newList := make([]RefiningRecipe, 0)

	for rows.Next() {
		var r RefiningRecipe
		if err := rows.Scan(&r.TargetGuID, &r.TargetRank, &r.Variant,
			&r.RequiredGus, &r.RequiredMaterials, &r.SuccessRate,
			&r.PrimevalStonesCost, &r.ImmortalEssenceStonesCost); err != nil {
			return err
		}
		r.ID = fmt.Sprintf("recipe_%s_%d_%d", r.TargetGuID, r.TargetRank, r.Variant)
		newRegistry[r.ID] = r

		if _, ok := newByTarget[r.TargetGuID]; !ok {
			newByTarget[r.TargetGuID] = make(map[int][]RefiningRecipe)
		}
		newByTarget[r.TargetGuID][r.TargetRank] = append(newByTarget[r.TargetGuID][r.TargetRank], r)
		newList = append(newList, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	recipeMu.Lock()
	RecipeRegistry = newRegistry
	RecipeByTarget = newByTarget
	recipeList = newList
	recipeMu.Unlock()

	logger.S.Infof("LoadRecipeRegistry: loaded %d refining recipes into memory", len(newRegistry))
	return nil
}

// GetRecipe returns a recipe by recipe_id from memory cache.
func GetRecipe(recipeID string) (RefiningRecipe, bool) {
	recipeMu.RLock()
	defer recipeMu.RUnlock()
	r, ok := RecipeRegistry[recipeID]
	return r, ok
}

// GetRecipeByTarget returns the recipe for targetGuID, targetRank, and variant from memory cache.
func GetRecipeByTarget(targetGuID string, targetRank, variant int) (RefiningRecipe, bool) {
	recipeMu.RLock()
	defer recipeMu.RUnlock()
	if variants, ok := RecipeByTarget[targetGuID]; ok {
		if list, ok := variants[targetRank]; ok {
			for _, r := range list {
				if r.Variant == variant {
					return r, true
				}
			}
		}
	}
	return RefiningRecipe{}, false
}

// GetAllRecipes returns a copy of all recipes from memory cache.
func GetAllRecipes() []RefiningRecipe {
	recipeMu.RLock()
	defer recipeMu.RUnlock()
	res := make([]RefiningRecipe, len(recipeList))
	copy(res, recipeList)
	return res
}

