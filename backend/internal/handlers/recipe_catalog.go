package handlers

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
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
