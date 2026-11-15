package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
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
			httperr.Respond(c, httperr.Internal("Database transaction failure", err))
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
	httperr.Respond(c, httperr.Internal(msg, err))
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
	if err := lockPvPCombatPreparation(tx, authID); err != nil {
		respondCombatTxErr(c, err, "Không thể khóa trạng thái chuẩn bị PvP.")
		return
	}

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		respondCombatTxErr(c, err, "Failed to lock player Aperture")
		return
	}
	if _, sessionErr := loadCombatSession(tx, authID, nil, true); sessionErr == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Đạo hữu cần dùng action Bỏ chạy trong lượt chiến đấu."})
		return
	} else if sessionErr != sql.ErrNoRows {
		respondCombatTxErr(c, sessionErr, "Không thể kiểm tra phiên chiến đấu.")
		return
	}

	if errLeader := checkPartyLeaderOrSolo(tx, &player); errLeader != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": errLeader.Error()})
		return
	}
	if player.ActiveCombat == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Không còn trận PvP nào đang chuẩn bị."})
		return
	}
	if player.ActiveCombat.EnemyType != "PVP" {
		c.JSON(http.StatusConflict, gin.H{"error": "Encounter PvE này không thể rút lui ở bước chuẩn bị; hãy bắt đầu trận để đánh hoặc bỏ chạy trong lượt."})
		return
	}

	combat := player.ActiveCombat
	opponentID, hasOpponent := pvpPreparationOpponentID(combat, player.ID)
	if combat.CombatMode == "real" && !hasOpponent {
		c.JSON(http.StatusConflict, gin.H{"error": "Không thể xác minh đối phương để xin đồng ý thoát PvP."})
		return
	}
	if hasOpponent {
		var opponentRaw []byte
		if err := tx.QueryRow(`SELECT active_combat FROM players WHERE id = $1 FOR UPDATE`, opponentID).Scan(&opponentRaw); err != nil {
			respondCombatTxErr(c, err, "Không thể đọc trạng thái PvP của đối phương.")
			return
		}
		if len(opponentRaw) == 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Đối phương không còn ở bước chuẩn bị trận PvP này."})
			return
		}
		var opponentCombat models.CombatState
		if err := json.Unmarshal(opponentRaw, &opponentCombat); err != nil {
			respondCombatTxErr(c, err, "Không thể đọc trạng thái PvP của đối phương.")
			return
		}
		if id, ok := pvpPreparationOpponentID(&opponentCombat, opponentID); !ok || id != player.ID {
			c.JSON(http.StatusConflict, gin.H{"error": "Đối phương không còn ở bước chuẩn bị trận PvP này."})
			return
		}
		if opponentCombat.RetreatRequestedBy != combat.RetreatRequestedBy {
			c.JSON(http.StatusConflict, gin.H{"error": "Trạng thái đề nghị thoát PvP giữa hai bên không đồng bộ."})
			return
		}

		if combat.RetreatRequestedBy != opponentID.String() {
			if combat.RetreatRequestedBy == "" {
				combat.RetreatRequestedBy = player.ID.String()
				opponentCombat.RetreatRequestedBy = player.ID.String()
				player.ActiveCombat = combat
				if _, err := tx.Exec(`UPDATE players SET active_combat=jsonb_set(active_combat, '{retreat_requested_by}', to_jsonb($2::text), true), version=version+1, state_revision=state_revision+1 WHERE id=$1`, player.ID, player.ID.String()); err != nil {
					respondCombatTxErr(c, err, "Không thể gửi đề nghị thoát PvP.")
					return
				}
				if _, err := tx.Exec(`UPDATE players SET active_combat=jsonb_set(active_combat, '{retreat_requested_by}', to_jsonb($2::text), true), version=version+1, state_revision=state_revision+1 WHERE id=$1`, opponentID, player.ID.String()); err != nil {
					respondCombatTxErr(c, err, "Không thể gửi đề nghị thoát PvP.")
					return
				}
				if err := tx.Commit(); err != nil {
					respondCombatTxErr(c, err, "Không thể gửi đề nghị thoát PvP.")
					return
				}
				_ = cache.Delete("player_snapshot:" + player.ID.String())
				_ = cache.Delete("player_snapshot:" + opponentID.String())
				PublishPlayerCombatDelta(player.ID, CombatEventStarted, combat)
				PublishPlayerCombatDelta(opponentID, CombatEventStarted, &opponentCombat)
				respondActionOnly(c, http.StatusOK, player, gin.H{"retreat_pending": true, "active_combat": combat, "log": "Đã gửi đề nghị thoát PvP; chờ đối phương đồng ý."})
				return
			}
			// Repeating one's own request is idempotent; only the opponent can confirm it.
			if combat.RetreatRequestedBy == player.ID.String() {
				c.JSON(http.StatusConflict, gin.H{"error": "Đang chờ đối phương đồng ý thoát PvP."})
				return
			}
			c.JSON(http.StatusConflict, gin.H{"error": "Đề nghị thoát PvP không hợp lệ."})
			return
		}
	}
	player.ActiveCombat = nil
	if hasOpponent {
		if _, err := tx.Exec(`UPDATE players
			SET active_combat = NULL, version = version + 1, state_revision = state_revision + 1
			WHERE id = $1 AND active_combat->>'enemy_type' = 'PVP'
			  AND active_combat->>'combat_mode' = 'real'
			  AND active_combat->>'combat_owner_id' = $2
			  AND active_combat->>'target_player_id' IN ($3, $2)`, opponentID, combat.CombatOwnerID, player.ID.String()); err != nil {
			respondCombatTxErr(c, err, "Không thể hủy trận PvP chuẩn bị.")
			return
		}
	}
	if _, err := tx.Exec(`UPDATE players
		SET active_combat = NULL, version = version + 1, state_revision = state_revision + 1
		WHERE id = $1`, player.ID); err != nil {
		respondCombatTxErr(c, err, "Không thể hủy trận PvP chuẩn bị.")
		return
	}
	if err = tx.Commit(); err != nil {
		respondCombatTxErr(c, err, "Không thể hoàn tất việc hủy trận PvP.")
		return
	}
	ClearPendingPlayerQueues(player.ID.String())
	_ = cache.Delete("player_snapshot:" + player.ID.String())
	PublishPlayerCombatDelta(player.ID, CombatEventEnded, nil)
	if hasOpponent {
		ClearPendingPlayerQueues(opponentID.String())
		_ = cache.Delete("player_snapshot:" + opponentID.String())
		PublishPlayerCombatDelta(opponentID, CombatEventEnded, nil)
	}
	respondActionOnly(c, http.StatusOK, player, gin.H{"log": "Đã rời trận PvP trước khi bắt đầu."})
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
