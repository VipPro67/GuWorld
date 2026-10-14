package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

const maxGroundDropQueryRadius = 127

type GroundDropResponse struct {
	ID             string         `json:"id"`
	RegionID       string         `json:"region_id"`
	X              int            `json:"x"`
	Y              int            `json:"y"`
	LoserID        string         `json:"loser_id"`
	LoserName      string         `json:"loser_name"`
	KillerID       string         `json:"killer_id,omitempty"`
	KillerName     string         `json:"killer_name,omitempty"`
	LootData       GroundLootData `json:"loot_data"`
	IsProtected    bool           `json:"is_protected"`
	ProtectedUntil *time.Time     `json:"protected_until,omitempty"`
	ExpiresAt      time.Time      `json:"expires_at"`
	CreatedAt      time.Time      `json:"created_at"`
}

// GetTileGroundDrops returns unexpired ground drops in the current viewport.
// Optional query params: 'x', 'y', and 'radius'. Without them, it uses the
// player's position and the standard map vision radius.
func GetTileGroundDrops(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var currentRegion sql.NullString
	var playerX, playerY int
	var partyID sql.NullString
	err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT current_region, x, y, party_id FROM players WHERE id = $1
	`, authID).Scan(&currentRegion, &playerX, &playerY, &partyID)
	if err != nil || !currentRegion.Valid || currentRegion.String == "" {
		c.JSON(http.StatusOK, gin.H{"drops": []GroundDropResponse{}})
		return
	}

	qX := c.Query("x")
	qY := c.Query("y")
	qRadius := c.Query("radius")

	var rows *sql.Rows
	if qX != "" || qY != "" || qRadius != "" {
		if qX == "" || qY == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "x and y must be provided together"})
			return
		}
		targetX, errX := strconv.Atoi(qX)
		targetY, errY := strconv.Atoi(qY)
		radius := 0
		var errRadius error
		if qRadius != "" {
			radius, errRadius = strconv.Atoi(qRadius)
		}
		if errX != nil || errY != nil || errRadius != nil || radius < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid drop viewport"})
			return
		}
		if radius > maxGroundDropQueryRadius {
			radius = maxGroundDropQueryRadius
		}
		if targetX < -maxGroundDropQueryRadius || targetX > WorldWidth-1+maxGroundDropQueryRadius ||
			targetY < -maxGroundDropQueryRadius || targetY > WorldHeight-1+maxGroundDropQueryRadius ||
			targetX+radius < 0 || targetX-radius >= WorldWidth ||
			targetY+radius < 0 || targetY-radius >= WorldHeight {
			c.JSON(http.StatusOK, gin.H{"drops": []GroundDropResponse{}})
			return
		}
		rows, err = db.DB.QueryContext(c.Request.Context(), `
			SELECT id, region_id, x, y, loser_id, loser_name, killer_id, killer_name,
			       loot_data, protected_until, expires_at, created_at
			FROM map_ground_drops
			WHERE x BETWEEN $1 AND $2 AND y BETWEEN $3 AND $4
			  AND expires_at > NOW()
			ORDER BY created_at DESC
		`, targetX-radius, targetX+radius, targetY-radius, targetY+radius)
	} else {
		rows, err = db.DB.QueryContext(c.Request.Context(), `
			SELECT id, region_id, x, y, loser_id, loser_name, killer_id, killer_name,
			       loot_data, protected_until, expires_at, created_at
			FROM map_ground_drops
			WHERE x BETWEEN $1 AND $2 AND y BETWEEN $3 AND $4
			  AND expires_at > NOW()
			ORDER BY created_at DESC
		`, playerX-MaxMapVisionRadius, playerX+MaxMapVisionRadius,
			playerY-MaxMapVisionRadius, playerY+MaxMapVisionRadius)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch ground drops"})
		return
	}
	defer rows.Close()

	drops := make([]GroundDropResponse, 0)
	now := time.Now()

	for rows.Next() {
		var d GroundDropResponse
		var killerID sql.NullString
		var killerName sql.NullString
		var lootBytes []byte
		var protectedUntil sql.NullTime

		if errScan := rows.Scan(
			&d.ID, &d.RegionID, &d.X, &d.Y, &d.LoserID, &d.LoserName,
			&killerID, &killerName, &lootBytes, &protectedUntil, &d.ExpiresAt, &d.CreatedAt,
		); errScan != nil {
			continue
		}

		if IsSafeZone(d.RegionID, d.X, d.Y) {
			continue
		}

		if killerID.Valid {
			d.KillerID = killerID.String
		}
		if killerName.Valid {
			d.KillerName = killerName.String
		}
		if protectedUntil.Valid {
			d.ProtectedUntil = &protectedUntil.Time
		}

		json.Unmarshal(lootBytes, &d.LootData)

		// Protection check: protected if protected_until > NOW() and authID != killer_id
		isProtected := false
		if protectedUntil.Valid && protectedUntil.Time.After(now) {
			if killerID.Valid && killerID.String != authID.String() {
				isProtected = true
			}
		}
		d.IsProtected = isProtected

		drops = append(drops, d)
	}

	c.JSON(http.StatusOK, gin.H{"drops": drops})
}

// LootGroundDrop picks up the specified ground drop if player is at the same location and allowed.
func LootGroundDrop(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	dropIDStr := c.Param("id")
	dropUUID, err := uuid.Parse(dropIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid drop ID"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var dropRegion string
	var dropX, dropY int
	var killerID sql.NullString
	var killerName sql.NullString
	var loserID uuid.UUID
	var loserName string
	var lootBytes []byte
	var protectedUntil sql.NullTime
	var expiresAt time.Time

	err = tx.QueryRow(`
		SELECT region_id, x, y, killer_id, killer_name, loser_id, loser_name, loot_data, protected_until, expires_at
		FROM map_ground_drops
		WHERE id = $1 FOR UPDATE NOWAIT
	`, dropUUID).Scan(
		&dropRegion, &dropX, &dropY, &killerID, &killerName, &loserID, &loserName, &lootBytes, &protectedUntil, &expiresAt,
	)
	if err != nil {
		if db.IsLockNotAvailable(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Vật phẩm đang có người nhặt, vui lòng thử lại sau giây lát!", "code": "RESOURCE_BUSY"})
			return
		}
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Chiến lợi phẩm không tồn tại hoặc đã biến mất"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query error"})
		return
	}

	if time.Now().After(expiresAt) {
		_, _ = tx.Exec("DELETE FROM map_ground_drops WHERE id = $1", dropUUID)
		_ = tx.Commit()
		c.JSON(http.StatusGone, gin.H{"error": "Chiến lợi phẩm đã bị thời gian làm hư hại hoàn toàn"})
		return
	}

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player"})
		return
	}

	// Verify player location matches drop location
	if player.CurrentRegion != dropRegion || player.X != dropX || player.Y != dropY {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn phải đứng cùng vị trí với chiến lợi phẩm để nhặt"})
		return
	}

	// Protection check
	now := time.Now()
	if protectedUntil.Valid && protectedUntil.Time.After(now) {
		if killerID.Valid && killerID.String != player.ID.String() {
			c.JSON(http.StatusForbidden, gin.H{
				"error": fmt.Sprintf("Chiến lợi phẩm đang được bảo hộ dành cho %s!", killerName.String),
			})
			return
		}
	}

	var loot GroundLootData
	if err := json.Unmarshal(lootBytes, &loot); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse loot data"})
		return
	}

	var logs []string

	// 1. Primeval & Immortal Stones
	if loot.PrimevalStones > 0 || loot.ImmortalStones > 0 {
		player.PrimevalStones += loot.PrimevalStones
		player.ImmortalStones += loot.ImmortalStones
		if _, errStones := tx.Exec(`
			UPDATE players SET
				primeval_stones = primeval_stones + $1,
				immortal_stones = immortal_stones + $2
			WHERE id = $3
		`, loot.PrimevalStones, loot.ImmortalStones, player.ID); errStones != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add stones"})
			return
		}
		if loot.PrimevalStones > 0 {
			logs = append(logs, fmt.Sprintf("+%.3f Nguyên Thạch", loot.PrimevalStones))
		}
		if loot.ImmortalStones > 0 {
			logs = append(logs, fmt.Sprintf("+%.3f Tiên Nguyên Thạch", loot.ImmortalStones))
		}
	}

	// 2. Materials & Items
	if len(loot.Items) > 0 {
		batchEntries := make([]services.ItemBatchEntry, 0, len(loot.Items))
		for _, item := range loot.Items {
			batchEntries = append(batchEntries, services.ItemBatchEntry{
				ItemID:   item.ItemID,
				Category: item.Category,
				Quantity: int64(item.Quantity),
			})
			itemName := item.ItemID
			if spec, ok := models.ItemRegistry[item.ItemID]; ok && spec.Name != "" {
				itemName = spec.Name
			}
			logs = append(logs, fmt.Sprintf("%dx %s", item.Quantity, itemName))
		}
		if errAdd := InventorySvc.AddItemsBatch(tx, player.ID.String(), batchEntries); errAdd != nil {
			logger.S.Errorf("Failed to add ground drop items to player %s: %v", player.ID, errAdd)
		}
	}

	// 3. Gus
	if len(loot.Gus) > 0 {
		newGus := make([]models.GuInstance, 0, len(loot.Gus))
		for _, g := range loot.Gus {
			newGu := models.GuInstance{
				InstanceID: "gu-" + uuid.New().String(),
				GuID:       g.GuID,
				Rank:       g.Rank,
				Hunger:     100,
				Status:     "ACTIVE",
				Equipped:   false,
			}
			newGus = append(newGus, newGu)
			guName := g.GuID
			if spec, ok := models.GetGuAny(g.GuID); ok && spec.Name != "" {
				guName = spec.Name
			}
			logs = append(logs, fmt.Sprintf("Cổ trùng [%s]", guName))
		}
		if errGu := db.AddPlayerGusBulk(tx, player.ID.String(), newGus); errGu != nil {
			logger.S.Errorf("Failed to add looted Gus to player %s: %v", player.ID, errGu)
		}
	}

	// 4. Recipes
	if len(loot.Recipes) > 0 {
		recipesToUpsert := make([]*services.RecipeProgress, 0, len(loot.Recipes))
		for _, rID := range loot.Recipes {
			rec := &services.RecipeProgress{
				PlayerID:   player.ID.String(),
				RecipeID:   rID,
				IsUnlocked: true,
			}
			recipesToUpsert = append(recipesToUpsert, rec)
			recName := rID
			if guDef, ok := models.GetGuAny(rID); ok && guDef.Name != "" {
				recName = guDef.Name
			}
			logs = append(logs, fmt.Sprintf("Bí phương [%s]", recName))
		}
		if errRec := RecipeSvc.UpsertBatch(tx, recipesToUpsert); errRec != nil {
			logger.S.Errorf("Failed to add looted recipes to player %s: %v", player.ID, errRec)
		}
	}

	// Xóa đống đồ sau khi nhặt
	if _, errDel := tx.Exec("DELETE FROM map_ground_drops WHERE id = $1", dropUUID); errDel != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear ground drop"})
		return
	}

	if errCommit := tx.Commit(); errCommit != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	summaryMsg := joinDropList(logs)

	// Notify looter
	notify.Send(db.DB, player.ID, "ground_drop_looted", "Thu hoạch chiến lợi phẩm",
		fmt.Sprintf("Bạn đã thu thập thành công chiến lợi phẩm của %s: %s.", loserName, summaryMsg))

	// Notify loser if looted by another player
	if loserID != player.ID {
		notify.Send(db.DB, loserID, "ground_drop_looted_by_other", "Chiến lợi phẩm bị thu thập",
			fmt.Sprintf("Chiến lợi phẩm của bạn tại %s (%d, %d) đã bị %s nhặt mất: %s.", dropRegion, dropX, dropY, player.Username, summaryMsg))
	}

	// Notify killer if drop was looted by someone else (third party)
	if killerID.Valid && killerID.String != "" && killerID.String != player.ID.String() {
		if kUUID, errK := uuid.Parse(killerID.String); errK == nil && kUUID != loserID {
			notify.Send(db.DB, kUUID, "ground_drop_looted_by_third_party", "Chiến lợi phẩm chiến trường",
				fmt.Sprintf("Chiến lợi phẩm của kẻ thù [%s] do bạn hạ gục tại %s (%d, %d) đã bị Cổ Sư [%s] thu thập: %s.", loserName, dropRegion, dropX, dropY, player.Username, summaryMsg))
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":             fmt.Sprintf("Thu thập thành công chiến lợi phẩm của %s: %s", loserName, summaryMsg),
		"looted":              logs,
		"materials_inventory": loadMaterialsInventory(player.ID.String()),
		"primeval_stones":     player.PrimevalStones,
		"immortal_stones":     player.ImmortalStones,
	})
}
