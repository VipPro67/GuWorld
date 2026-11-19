package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

// AdminListPlayerGuHouses returns all Gu houses owned by a player.
func AdminListPlayerGuHouses(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT id, name, rank, core_gu_instances, aux_gu_count, house_profile, current_durability, is_equipped, created_at, updated_at
		FROM player_gu_houses
		WHERE player_id = $1
		ORDER BY is_equipped DESC, rank DESC, created_at DESC
	`, playerID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể đọc danh sách Cổ Ốc", err))
		return
	}
	defer rows.Close()

	houses := make([]gin.H, 0)
	for rows.Next() {
		var id uuid.UUID
		var name string
		var rank, auxCount int
		var currentDurability float64
		var isEquipped bool
		var coreInstancesRaw, houseProfileRaw []byte
		var createdAt, updatedAt interface{}

		if err := rows.Scan(&id, &name, &rank, &coreInstancesRaw, &auxCount, &houseProfileRaw, &currentDurability, &isEquipped, &createdAt, &updatedAt); err != nil {
			httperr.Respond(c, httperr.Internal("Lỗi đọc dữ liệu Cổ Ốc", err))
			return
		}

		var coreInstances interface{} = []interface{}{}
		if len(coreInstancesRaw) > 0 {
			_ = json.Unmarshal(coreInstancesRaw, &coreInstances)
		}
		var houseProfile interface{} = map[string]interface{}{}
		if len(houseProfileRaw) > 0 {
			_ = json.Unmarshal(houseProfileRaw, &houseProfile)
		}

		houses = append(houses, gin.H{
			"id":                 id.String(),
			"name":               name,
			"rank":               rank,
			"core_gu_instances":  coreInstances,
			"aux_gu_count":       auxCount,
			"house_profile":      houseProfile,
			"current_durability": currentDurability,
			"is_equipped":        isEquipped,
			"created_at":         createdAt,
			"updated_at":         updatedAt,
		})
	}

	c.JSON(http.StatusOK, houses)
}

// AdminDeletePlayerGuHouse deletes a player's Gu house.
func AdminDeletePlayerGuHouse(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	houseID, err := uuid.Parse(c.Param("house_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID Cổ Ốc không hợp lệ"})
		return
	}

	var name string
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT name FROM player_gu_houses WHERE id = $1 AND player_id = $2`, houseID, playerID).Scan(&name); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ Ốc"})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể mở transaction", err))
		return
	}
	defer tx.Rollback()

	_, _ = tx.ExecContext(c.Request.Context(), `DELETE FROM player_gu_house_core_reservations WHERE gu_house_id = $1 AND player_id = $2`, houseID, playerID)
	if _, err := tx.ExecContext(c.Request.Context(), `DELETE FROM player_gu_houses WHERE id = $1 AND player_id = $2`, houseID, playerID); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể xóa Cổ Ốc", err))
		return
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể commit transaction", err))
		return
	}

	auditlog.LogContext(c.Request.Context(), db.DB, auditlog.Entry{
		PlayerID:       &adminID,
		EntityType:     "player_gu_houses",
		EntityID:       houseID.String(),
		Action:         "admin.delete_player_gu_house",
		RequestSummary: fmt.Sprintf("player_id=%s house_name=%s", playerID.String(), name),
		Reason:         c.GetHeader("X-Admin-Reason"),
		IPAddress:      c.ClientIP(),
		Success:        true,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa Cổ Ốc thành công"})
}

// AdminListPlayerKillerMoves returns all killer moves registered by a player.
func AdminListPlayerKillerMoves(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT id, name, main_gu_instance_id, main_gu_id, main_gu_rank, components, selected_effects, compiled_effects, path, harmony, power_budget, power_load, is_equipped, created_at, updated_at
		FROM player_killer_moves
		WHERE player_id = $1
		ORDER BY is_equipped DESC, created_at DESC
	`, playerID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể đọc danh sách Sát Chiêu", err))
		return
	}
	defer rows.Close()

	moves := make([]gin.H, 0)
	for rows.Next() {
		var id uuid.UUID
		var name, mainGuInstanceID, mainGuID, path string
		var mainGuRank int
		var harmony, powerBudget, powerLoad float64
		var isEquipped bool
		var componentsRaw, selectedEffectsRaw, compiledEffectsRaw []byte
		var createdAt, updatedAt interface{}

		if err := rows.Scan(&id, &name, &mainGuInstanceID, &mainGuID, &mainGuRank, &componentsRaw, &selectedEffectsRaw, &compiledEffectsRaw, &path, &harmony, &powerBudget, &powerLoad, &isEquipped, &createdAt, &updatedAt); err != nil {
			httperr.Respond(c, httperr.Internal("Lỗi đọc dữ liệu Sát Chiêu", err))
			return
		}

		var components interface{} = []interface{}{}
		if len(componentsRaw) > 0 {
			_ = json.Unmarshal(componentsRaw, &components)
		}
		var selectedEffects interface{} = []interface{}{}
		if len(selectedEffectsRaw) > 0 {
			_ = json.Unmarshal(selectedEffectsRaw, &selectedEffects)
		}
		var compiledEffects interface{} = map[string]interface{}{}
		if len(compiledEffectsRaw) > 0 {
			_ = json.Unmarshal(compiledEffectsRaw, &compiledEffects)
		}

		moves = append(moves, gin.H{
			"id":                  id.String(),
			"name":                name,
			"main_gu_instance_id": mainGuInstanceID,
			"main_gu_id":          mainGuID,
			"main_gu_rank":        mainGuRank,
			"components":          components,
			"selected_effects":    selectedEffects,
			"compiled_effects":    compiledEffects,
			"path":                path,
			"harmony":             harmony,
			"power_budget":        powerBudget,
			"power_load":          powerLoad,
			"is_equipped":         isEquipped,
			"created_at":          createdAt,
			"updated_at":          updatedAt,
		})
	}

	c.JSON(http.StatusOK, moves)
}

// AdminDeletePlayerKillerMove deletes a player's killer move.
func AdminDeletePlayerKillerMove(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	moveID, err := uuid.Parse(c.Param("move_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID Sát Chiêu không hợp lệ"})
		return
	}

	var name string
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT name FROM player_killer_moves WHERE id = $1 AND player_id = $2`, moveID, playerID).Scan(&name); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Sát Chiêu"})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể mở transaction", err))
		return
	}
	defer tx.Rollback()

	_, _ = tx.ExecContext(c.Request.Context(), `DELETE FROM player_killer_move_gu_reservations WHERE killer_move_id = $1 AND player_id = $2`, moveID, playerID)
	if _, err := tx.ExecContext(c.Request.Context(), `DELETE FROM player_killer_moves WHERE id = $1 AND player_id = $2`, moveID, playerID); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể xóa Sát Chiêu", err))
		return
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể commit transaction", err))
		return
	}

	auditlog.LogContext(c.Request.Context(), db.DB, auditlog.Entry{
		PlayerID:       &adminID,
		EntityType:     "player_killer_moves",
		EntityID:       moveID.String(),
		Action:         "admin.delete_player_killer_move",
		RequestSummary: fmt.Sprintf("player_id=%s move_name=%s", playerID.String(), name),
		Reason:         c.GetHeader("X-Admin-Reason"),
		IPAddress:      c.ClientIP(),
		Success:        true,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa Sát Chiêu thành công"})
}
