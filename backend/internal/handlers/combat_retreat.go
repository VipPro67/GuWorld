package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// isTimeoutOrBusy reports whether an error is a context deadline/cancel, a
// Postgres lock timeout (55P03), a canceled statement (57014) or a deadlock
// (40P01) — all of which the client should retry rather than treat as fatal.
func isTimeoutOrBusy(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "55P03", "57014", "40P01":
			return true
		}
	}
	return false
}

// beginCombatTx opens a bounded transaction whose lock waits cannot hang
// indefinitely: the context caps the whole request and SET LOCAL lock_timeout
// aborts any row-lock wait after 5 seconds. The shared busy message is used by
// combat mutation handlers so clients surface a retriable error.
// On failure it responds itself and returns (nil, nil).
func beginCombatTx(c *gin.Context) (*sql.Tx, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		cancel()
		if isTimeoutOrBusy(err) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Hệ thống đang bận — quá nhiều thao tác cùng lúc, đạo hữu thử lại chút nữa nhé."})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		}
		return nil, nil
	}
	if _, errSet := tx.Exec("SET LOCAL lock_timeout = '5000'"); errSet != nil {
		logger.S.Warnf("beginCombatTx: failed to SET LOCAL lock_timeout: %v", errSet)
	}
	// The default 15s statement_timeout (from the connection string) can abort
	// large bulk Gu inventory writes (e.g. a retreat that harvests many Gus).
	// Raise it for combat mutation flows; the surrounding context still caps the
	// whole request at 30s.
	if _, errSet := tx.Exec("SET LOCAL statement_timeout = '60000'"); errSet != nil {
		logger.S.Warnf("beginCombatTx: failed to SET LOCAL statement_timeout: %v", errSet)
	}
	return tx, cancel
}

// respondCombatTxErr maps transaction errors to a friendly client response.
func respondCombatTxErr(c *gin.Context, err error, fallback string) {
	if err == nil {
		return
	}
	if isTimeoutOrBusy(err) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Hệ thống đang bận — quá nhiều thao tác cùng lúc, đạo hữu thử lại chút nữa nhé."})
		return
	}
	msg := fallback
	if msg == "" {
		msg = "Giao dịch thất bại, vui lòng thử lại."
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
}

func Retreat(c *gin.Context) {
	playerIDStr := c.Param("id")
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	playerUUID, err := uuid.Parse(playerIDStr)
	if err != nil || playerUUID != authID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Aperture mismatch"})
		return
	}

	tx, cancel := beginCombatTx(c)
	if tx == nil {
		return
	}
	defer cancel()
	defer tx.Rollback()
	if _, sessionErr := loadCombatSession(tx, authID, nil, true); sessionErr == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Đạo hữu cần dùng action Bỏ chạy trong lượt chiến đấu."})
		return
	} else if sessionErr != sql.ErrNoRows {
		respondCombatTxErr(c, sessionErr, "Không thể kiểm tra phiên chiến đấu.")
		return
	}

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		respondCombatTxErr(c, err, "Failed to lock player Aperture")
		return
	}

	if errLeader := checkPartyLeaderOrSolo(tx, &player); errLeader != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": errLeader.Error()})
		return
	}
	if player.ActiveCombat != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Đang chiến đấu; hãy dùng lượt Bỏ chạy hoặc kết thúc trận theo luật của mode này."})
		return
	}
	before := append(models.GuInventory(nil), player.GuInventory...)
	inventoryBefore := len(before)

	logMsg := "Đạo hữu trở về an toàn từ hành trình."

	player.CurrentRegion, player.X, player.Y = ResolveRespawnPosition(tx, authID, player.CurrentRegion, player.X, player.Y)

	qiGained := AddHumanQi(&player, 2.0)

	updateQuery := `
		UPDATE players
		SET hp = $1, current_mana = $2, active_combat = $3, x = $4, y = $5, primeval_stones = $6,
		    current_region = $7, human_qi = $8
		WHERE id = $9
	`
	_, err = tx.Exec(updateQuery, player.HP, player.CurrentMana, player.ActiveCombat, player.X, player.Y, player.PrimevalStones,
		player.CurrentRegion, player.HumanQi, player.ID)
	if err != nil {
		respondCombatTxErr(c, err, "Failed to save player status after retreat")
		return
	}

	var dbBefore int
	_ = tx.QueryRow(`SELECT COUNT(*) FROM player_gu WHERE player_id = $1`, player.ID).Scan(&dbBefore)

	if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
		respondCombatTxErr(c, err, "Failed to save player Gus after retreat")
		return
	}

	inventoryAfter := len(player.GuInventory)
	var dbAfterTx int
	_ = tx.QueryRow(`SELECT COUNT(*) FROM player_gu WHERE player_id = $1`, player.ID).Scan(&dbAfterTx)

	if err = tx.Commit(); err != nil {
		respondCombatTxErr(c, err, "Transaction commit failed")
		return
	}
	ClearPendingPlayerQueues(player.ID.String())
	_ = cache.Delete("player_snapshot:" + player.ID.String())
	PublishPlayerCombatDelta(player.ID, CombatEventEnded, nil)

	setWorldPosFromPlayer(&player)

	var dbAfter int
	_ = db.DB.QueryRow(`SELECT COUNT(*) FROM player_gu WHERE player_id = $1`, player.ID).Scan(&dbAfter)
	logger.S.Infof("Retreat player=%s invBefore=%d invAfter=%d added=%d dbBefore=%d dbAfterTx=%d dbAfter=%d",
		player.ID, inventoryBefore, inventoryAfter, inventoryAfter-inventoryBefore, dbBefore, dbAfterTx, dbAfter)
	respondActionOnly(c, http.StatusOK, player, gin.H{
		"log": logMsg + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
	})
}

func Flee(c *gin.Context) {
	playerUUID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid player ID"})
		return
	}
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	if playerUUID != authID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Aperture mismatch"})
		return
	}

	tx, cancel := beginCombatTx(c)
	if tx == nil {
		return
	}
	defer cancel()
	defer tx.Rollback()

	_, sessionErr := loadCombatSession(tx, authID, nil, true)
	if sessionErr != nil && sessionErr != sql.ErrNoRows {
		respondCombatTxErr(c, sessionErr, "Không thể kiểm tra phiên chiến đấu.")
		return
	}
	player, err := evaluateManaTxEquipped(tx, authID)
	if err != nil {
		respondCombatTxErr(c, err, "Không thể đọc trạng thái nhân vật.")
		return
	}
	if player.ActiveCombat == nil {
		respondActionOnly(c, http.StatusOK, player, gin.H{"log": "Không có trận chiến nào đang diễn ra."})
		return
	}
	if !combatOwnedByPlayer(player.ActiveCombat, player.ID.String()) {
		c.JSON(http.StatusConflict, gin.H{"error": "Trận chiến tổ đội do trưởng nhóm điều khiển."})
		return
	}
	if sessionErr == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Bỏ chạy phải được gửi như action trong lượt chiến đấu."})
		return
	}
	c.JSON(http.StatusConflict, gin.H{"error": "Hãy bắt đầu phiên chiến đấu turn-based để bỏ chạy trong lượt."})
}
