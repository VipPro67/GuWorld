package handlers

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
)

const recipeCatalogCacheKey = "master_refining_recipes"

var (
	recipeCatalogMu       sync.RWMutex
	recipeCatalogMemory   []models.RefiningRecipe
	recipeMemoryRevision  uint64
	recipeCatalogRevision atomic.Uint64
)

func init() {
	recipeCatalogRevision.Store(1)
}

func loadMasterRecipes() ([]models.RefiningRecipe, error) {
	revision := currentRecipeCatalogRevision()
	recipeCatalogMu.RLock()
	if recipeCatalogMemory != nil && recipeMemoryRevision == revision {
		result := append([]models.RefiningRecipe(nil), recipeCatalogMemory...)
		recipeCatalogMu.RUnlock()
		return result, nil
	}
	recipeCatalogMu.RUnlock()

	var recipes []models.RefiningRecipe
	versionedCacheKey := fmt.Sprintf("%s:%d", recipeCatalogCacheKey, revision)
	if err := cache.GetJSON(versionedCacheKey, &recipes); err != nil {
		rows, queryErr := db.DB.Query(`
			SELECT target_gu_id, target_rank, variant, required_gus, required_materials,
			       success_rate, primeval_stones_cost, immortal_essence_stones_cost
			FROM refining_recipes
			ORDER BY target_gu_id, target_rank, variant`)
		if queryErr != nil {
			return nil, queryErr
		}
		defer rows.Close()

		for rows.Next() {
			var recipe models.RefiningRecipe
			if err := rows.Scan(&recipe.TargetGuID, &recipe.TargetRank, &recipe.Variant,
				&recipe.RequiredGus, &recipe.RequiredMaterials, &recipe.SuccessRate,
				&recipe.PrimevalStonesCost, &recipe.ImmortalEssenceStonesCost); err != nil {
				return nil, err
			}
			recipe.ID = fmt.Sprintf("recipe_%s_%d_%d", recipe.TargetGuID, recipe.TargetRank, recipe.Variant)
			recipes = append(recipes, recipe)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		_ = cache.SetJSON(versionedCacheKey, recipes, time.Hour)
	}

	recipeCatalogMu.Lock()
	recipeCatalogMemory = append([]models.RefiningRecipe(nil), recipes...)
	recipeMemoryRevision = revision
	recipeCatalogMu.Unlock()
	return append([]models.RefiningRecipe(nil), recipes...), nil
}

func recipeResponseETag(recipes []models.RefiningRecipe, revision uint64) string {
	raw, _ := json.Marshal(recipes)
	hash := md5.Sum(raw)
	return fmt.Sprintf(`"recipes-%d-%x"`, revision, hash)
}

// recipeCatalogInfo resolves a stored recipe id back to its authoritative
// catalog tuple. recipe_id is only a projection of
// (target_gu_id, target_rank, variant), so rank, rarity and display name must
// come from refining_recipes instead of parsing the id string: a Gu id that
// itself ends with a numeric suffix cannot be told apart from the rank/variant
// suffix by parsing alone.
//
// The boolean result is false when the id is unknown to the catalog (legacy or
// orphaned recipe row). Callers must decide whether to fall back to
// ParseRecipeID or to reject the operation.
func recipeCatalogInfo(q services.QueryRower, recipeID string) (targetGuID string, rank, variant, rarity int, name string, ok bool) {
	recipeID = strings.TrimSpace(recipeID)
	if recipeID == "" || q == nil {
		return "", 0, 0, 0, "", false
	}
	err := q.QueryRow(`
		SELECT rr.target_gu_id, rr.target_rank, rr.variant,
		       COALESCE(gd.rarity, 0), COALESCE(gd.name, '')
		FROM refining_recipes rr
		JOIN gu_dictionary gd ON gd.id = rr.target_gu_id AND gd.rank = rr.target_rank
		WHERE ('recipe_' || rr.target_gu_id || '_' || rr.target_rank::text || '_' || rr.variant::text) = $1
	`, recipeID).Scan(&targetGuID, &rank, &variant, &rarity, &name)
	if err != nil {
		return "", 0, 0, 0, "", false
	}
	return targetGuID, rank, variant, rarity, name, true
}

// recipeTargetGuRank resolves the target Gu rank of a recipe id from the
// catalog tuple, falling back to parsing the id suffix only when the row is not
// in refining_recipes (legacy/orphaned data). Returns 1 as the safe floor.
func recipeTargetGuRank(q services.QueryRower, recipeID string) int {
	if _, rank, _, _, _, ok := recipeCatalogInfo(q, recipeID); ok && rank > 0 {
		return rank
	}
	if _, rank, _, err := ParseRecipeID(recipeID); err == nil && rank > 0 {
		return rank
	}
	return 1
}

func currentRecipeCatalogRevision() uint64 {
	if cache.Client != nil {
		_, _ = cache.Client.SetNX(cache.Ctx, "recipe_catalog_revision", recipeCatalogRevision.Load(), 0).Result()
		if revision, err := cache.Client.Get(cache.Ctx, "recipe_catalog_revision").Uint64(); err == nil && revision > 0 {
			recipeCatalogRevision.Store(revision)
			return revision
		}
	}
	return recipeCatalogRevision.Load()
}

func invalidateRecipeCatalog() uint64 {
	oldRevision := currentRecipeCatalogRevision()
	recipeCatalogMu.Lock()
	recipeCatalogMemory = nil
	recipeMemoryRevision = 0
	recipeCatalogMu.Unlock()
	cache.Delete(recipeCatalogCacheKey, fmt.Sprintf("%s:%d", recipeCatalogCacheKey, oldRevision))
	revision := recipeCatalogRevision.Add(1)
	if cache.Client != nil {
		_, _ = cache.Client.SetNX(cache.Ctx, "recipe_catalog_revision", revision-1, 0).Result()
		if sharedRevision, err := cache.Client.Incr(cache.Ctx, "recipe_catalog_revision").Uint64(); err == nil {
			revision = sharedRevision
			recipeCatalogRevision.Store(revision)
		}
	}
	_ = cache.PublishEventQuick("catalog:updates", map[string]interface{}{
		"type":     "catalog.updated",
		"catalog":  "recipes",
		"revision": revision,
	})
	return revision
}
