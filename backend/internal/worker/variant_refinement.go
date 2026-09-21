package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

var errRefinementBlocked = errors.New("auto refinement requirements not met")

type autoRecipe struct {
	targetGuID   string
	targetRank   int
	requiredGus  json.RawMessage
	materials    map[string]int64
	successRate  float64
	primevalCost float64
	immortalCost float64
}

type refinementCandidate struct {
	id   string
	rank int
}

// processApertureRefinement performs at most one attempt per queued job and
// production tick. Each attempt is atomic, so multiple server processes cannot
// consume the same ingredients or advance the same job twice.
func processApertureRefinement(ctx context.Context, database *sql.DB, apertureID string) {
	conn, err := database.Conn(ctx)
	if err != nil {
		logger.S.Errorf("VariantRefinement: reserve connection %s: %v", apertureID, err)
		return
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock(hashtext($1))`, apertureID); err != nil {
		logger.S.Errorf("VariantRefinement: lock aperture %s: %v", apertureID, err)
		return
	}
	defer conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock(hashtext($1))`, apertureID)

	var availableHairy int
	if err := conn.QueryRowContext(ctx, `
		SELECT GREATEST(v.population-COALESCE(SUM(w.assigned_population),0),0)
		FROM aperture_variant_humans v
		LEFT JOIN aperture_node_workers w ON w.variant_human_id=v.id
		WHERE v.aperture_id=$1 AND v.race='HAIRY_MAN'
		GROUP BY v.population`, apertureID).Scan(&availableHairy); err != nil || availableHairy <= 0 {
		return
	}
	rows, err := conn.QueryContext(ctx, `
		SELECT q.id, COALESCE(r.target_rank, 1)
		FROM aperture_refinement_queue q
		LEFT JOIN refining_recipes r
		  ON ('recipe_' || r.target_gu_id || '_' || r.target_rank::text || '_' || r.variant::text)=q.recipe_id
		WHERE q.aperture_id=$1 AND q.enabled=true
		ORDER BY q.priority DESC, q.created_at ASC`, apertureID)
	if err != nil {
		logger.S.Errorf("VariantRefinement: load queue %s: %v", apertureID, err)
		return
	}
	var candidates []refinementCandidate
	for rows.Next() {
		var c refinementCandidate
		if rows.Scan(&c.id, &c.rank) == nil {
			candidates = append(candidates, c)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		logger.S.Errorf("VariantRefinement: scan queue %s: %v", apertureID, err)
		return
	}
	rows.Close()
	for _, candidate := range selectRefinementCandidates(candidates, availableHairy) {
		if err := processRefinementJob(ctx, database, candidate.id, rand.Float64()); err != nil && !errors.Is(err, errRefinementBlocked) {
			logger.S.Errorf("VariantRefinement: process job %s: %v", candidate.id, err)
		}
	}
}

func selectRefinementCandidates(candidates []refinementCandidate, availableHairy int) []refinementCandidate {
	selected := make([]refinementCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		required := requiredHairyMen(candidate.rank)
		if required <= availableHairy {
			selected = append(selected, candidate)
			availableHairy -= required
		}
	}
	return selected
}

// requiredHairyMen is the workforce reserved for one refinement attempt.
// Each additional Gu rank reserves another 100 idle Hairy Men for the tick.
func requiredHairyMen(rank int) int {
	if rank < 1 {
		rank = 1
	}
	if rank > 9 {
		rank = 9
	}
	return rank * 100
}

func processRefinementJob(ctx context.Context, database *sql.DB, jobID string, roll float64) error {
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var apertureID, playerID, recipeID, mode string
	var target, current int
	err = tx.QueryRowContext(ctx, `
		SELECT q.aperture_id, a.player_id, q.recipe_id, q.mode, q.target_quantity, q.current_quantity
		FROM aperture_refinement_queue q JOIN immortal_apertures a ON a.id=q.aperture_id
		WHERE q.id=$1 AND q.enabled=true FOR UPDATE OF q`, jobID).
		Scan(&apertureID, &playerID, &recipeID, &mode, &target, &current)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	var recipe autoRecipe
	var materialsJSON []byte
	err = tx.QueryRowContext(ctx, `
		SELECT target_gu_id,target_rank,required_gus,required_materials,success_rate,
		       COALESCE(primeval_stones_cost,0),COALESCE(immortal_essence_stones_cost,0)
		FROM refining_recipes
		WHERE ('recipe_' || target_gu_id || '_' || target_rank::text || '_' || variant::text)=$1`, recipeID).
		Scan(&recipe.targetGuID, &recipe.targetRank, &recipe.requiredGus, &materialsJSON,
			&recipe.successRate, &recipe.primevalCost, &recipe.immortalCost)
	if err != nil {
		return blockRefinement(ctx, tx, jobID, "Bí phương không còn tồn tại")
	}
	if len(recipe.requiredGus) > 0 && string(recipe.requiredGus) != "{}" && string(recipe.requiredGus) != "[]" {
		return blockRefinement(ctx, tx, jobID, "Bí phương cần Cổ trùng nguyên liệu")
	}
	if err := json.Unmarshal(materialsJSON, &recipe.materials); err != nil {
		return err
	}

	var hairyManID string
	var hairyPopulation, assignedHairy int
	if err = tx.QueryRowContext(ctx, `
		SELECT id, population FROM aperture_variant_humans
		WHERE aperture_id=$1 AND race='HAIRY_MAN' FOR UPDATE`, apertureID).Scan(&hairyManID, &hairyPopulation); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return pauseRefinement(ctx, tx, jobID, "Không còn Mao Dân khả dụng")
		}
		return err
	}
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(assigned_population),0) FROM aperture_node_workers WHERE variant_human_id=$1`, hairyManID).Scan(&assignedHairy); err != nil {
		return err
	}
	availableHairy := hairyPopulation - assignedHairy
	if availableHairy < 0 {
		availableHairy = 0
	}
	requiredWorkforce := requiredHairyMen(recipe.targetRank)
	if availableHairy < requiredWorkforce {
		return pauseRefinement(ctx, tx, jobID, fmt.Sprintf("Cần %d Mao Dân chưa được phân công để luyện Cổ cấp %d", requiredWorkforce, recipe.targetRank))
	}

	if mode == models.RefineModeMaintainStock {
		var stock int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM player_gu WHERE player_id=$1 AND gu_id=$2 AND gu_rank=$3`, playerID, recipe.targetGuID, recipe.targetRank).Scan(&stock); err != nil {
			return err
		}
		if stock >= target {
			_, err = tx.ExecContext(ctx, `UPDATE aperture_refinement_queue SET last_error='',last_processed_at=NOW(),updated_at=NOW() WHERE id=$1`, jobID)
			if err != nil {
				return err
			}
			return tx.Commit()
		}
	}

	for materialID, quantity := range recipe.materials {
		var available int64
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(quantity),0) FROM player_inventory_items WHERE player_id=$1 AND item_id=$2`, playerID, materialID).Scan(&available); err != nil {
			return err
		}
		if available < quantity {
			return pauseRefinement(ctx, tx, jobID, "Không đủ nguyên liệu")
		}
	}
	var canPay bool
	if err := tx.QueryRowContext(ctx, `SELECT primeval_stones >= $2 AND immortal_stones >= $3 FROM players WHERE id=$1 FOR UPDATE`, playerID, recipe.primevalCost, recipe.immortalCost).Scan(&canPay); err != nil {
		return err
	}
	if !canPay {
		return pauseRefinement(ctx, tx, jobID, "Không đủ nguyên thạch")
	}

	for materialID, quantity := range recipe.materials {
		if err := deductRefinementMaterial(ctx, tx, playerID, materialID, quantity); err != nil {
			return err
		}
	}
	result, err := tx.ExecContext(ctx, `UPDATE players SET primeval_stones=primeval_stones-$2,immortal_stones=immortal_stones-$3,updated_at=NOW() WHERE id=$1 AND primeval_stones >= $2 AND immortal_stones >= $3`, playerID, recipe.primevalCost, recipe.immortalCost)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n != 1 {
		return errors.New("player currency changed during locked refinement")
	}

	succeeded := roll < recipe.successRate
	if succeeded {
		gu := models.GuInstance{InstanceID: uuid.NewString(), GuID: recipe.targetGuID, Rank: recipe.targetRank, Hunger: 100, Status: "ACTIVE"}
		if err := db.AddPlayerGu(tx, playerID, gu); err != nil {
			if db.IsImmortalConflict(err) {
				return err
			}
			return err
		}
		current++
	}

	disable := (mode == models.RefineModeFixedCount && current >= target) || (mode == models.RefineModeUntilSuccess && succeeded)
	_, err = tx.ExecContext(ctx, `UPDATE aperture_refinement_queue SET current_quantity=$2,enabled=$3,last_error='',last_processed_at=NOW(),updated_at=NOW() WHERE id=$1`, jobID, current, !disable)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func deductRefinementMaterial(ctx context.Context, tx *sql.Tx, playerID, itemID string, quantity int64) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT category, quantity FROM player_inventory_items
		WHERE player_id=$1 AND item_id=$2 AND quantity>0
		ORDER BY quantity DESC FOR UPDATE`, playerID, itemID)
	if err != nil {
		return err
	}
	type inventoryRow struct {
		category string
		quantity int64
	}
	var entries []inventoryRow
	var available int64
	for rows.Next() {
		var entry inventoryRow
		if err := rows.Scan(&entry.category, &entry.quantity); err != nil {
			rows.Close()
			return err
		}
		entries = append(entries, entry)
		available += entry.quantity
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if available < quantity {
		return sql.ErrNoRows
	}
	remaining := quantity
	for _, entry := range entries {
		if remaining == 0 {
			break
		}
		deduct := entry.quantity
		if deduct > remaining {
			deduct = remaining
		}
		if _, err := tx.ExecContext(ctx, `UPDATE player_inventory_items SET quantity=quantity-$4,updated_at=NOW() WHERE player_id=$1 AND item_id=$2 AND category=$3`, playerID, itemID, entry.category, deduct); err != nil {
			return err
		}
		remaining -= deduct
	}
	return nil
}

func blockRefinement(ctx context.Context, tx *sql.Tx, jobID, reason string) error {
	_, err := tx.ExecContext(ctx, `UPDATE aperture_refinement_queue SET enabled=false,last_error=$2,last_processed_at=NOW(),updated_at=NOW() WHERE id=$1`, jobID, reason)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return errRefinementBlocked
}

func pauseRefinement(ctx context.Context, tx *sql.Tx, jobID, reason string) error {
	_, err := tx.ExecContext(ctx, `UPDATE aperture_refinement_queue SET last_error=$2,last_processed_at=NOW(),updated_at=NOW() WHERE id=$1`, jobID, reason)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return errRefinementBlocked
}
