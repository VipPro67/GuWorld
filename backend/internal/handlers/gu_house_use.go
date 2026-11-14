package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type useGuHouseRequest struct {
	CoreGuInstanceID string `json:"core_gu_instance_id" binding:"required"`
	EffectKey        string `json:"effect_key" binding:"required"`
}

// UseGuHouse executes one explicitly selected ON_USE effect from a core Gu
// reserved by an active Gu House. It shares the world ON_USE executor used by
// Killer Moves and teleport formations.
func UseGuHouse(c *gin.Context) {
	playerID, ok := killerMoveOwner(c)
	if !ok {
		return
	}
	var req useGuHouseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cần chọn Cổ hạch tâm và effect muốn dùng"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	defer tx.Rollback()
	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể tải nhân vật", err))
		return
	}
	if player.ActiveCombat != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Không thể dùng Tiên Cổ Ốc khi đang trong combat"})
		return
	}
	var houseID, houseName string
	var profileRaw, coreRaw []byte
	var durability float64
	err = tx.QueryRow(`SELECT id::text,name,house_profile,core_gu_instances,current_durability
		FROM player_gu_houses WHERE id=$1 AND player_id=$2 AND is_equipped=true FOR UPDATE`, c.Param("id"), playerID).
		Scan(&houseID, &houseName, &profileRaw, &coreRaw, &durability)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tiên Cổ Ốc chưa active hoặc không tồn tại"})
		return
	}
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể tải Tiên Cổ Ốc", err))
		return
	}
	if durability <= 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Tiên Cổ Ốc đã hư hại, cần tu sửa trước"})
		return
	}
	if err := validateGuHouseActivationProfileMode(profileRaw, false); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	var coreIDs []string
	if err := json.Unmarshal(coreRaw, &coreIDs); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Danh sách Cổ hạch tâm không hợp lệ"})
		return
	}
	if !containsString(coreIDs, req.CoreGuInstanceID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cổ đã chọn không thuộc Tiên Cổ Ốc này"})
		return
	}
	var reserved bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM player_gu_house_core_reservations WHERE gu_instance_id=$1 AND gu_house_id=$2 AND player_id=$3)`, req.CoreGuInstanceID, houseID, playerID).Scan(&reserved); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể xác minh Cổ hạch tâm", err))
		return
	}
	if !reserved {
		c.JSON(http.StatusConflict, gin.H{"error": "Cổ hạch tâm không còn được Tiên Cổ Ốc giữ chỗ; hãy tháo và xuất chiến lại"})
		return
	}
	owned, err := loadOwnedKillerMoveGus(tx, playerID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể tải Cổ hạch tâm", err))
		return
	}
	gu, exists := owned[req.CoreGuInstanceID]
	if !exists || !strings.EqualFold(gu.State, "ACTIVE") || gu.Location != "INVENTORY" || gu.Hunger < 1 {
		c.JSON(http.StatusConflict, gin.H{"error": "Cổ hạch tâm không còn hoạt động hoặc đã kiệt sức"})
		return
	}
	spec, found := models.GetGu(gu.GuID, gu.Rank)
	if !found {
		c.JSON(http.StatusConflict, gin.H{"error": "Không tìm thấy dữ liệu Cổ hạch tâm"})
		return
	}
	var selected *killerMoveEffectOption
	options, err := getKillerMoveEffectOptions(gu)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	for i := range options {
		if options[i].Key == req.EffectKey && options[i].TravelSupported && options[i].SourceID == req.CoreGuInstanceID {
			selected = &options[i]
			break
		}
	}
	if selected == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Effect ON_USE này không được executor thế giới hỗ trợ"})
		return
	}
	block := gameplayeffects.Block{ID: selected.SourceBlockID, Trigger: gameplayeffects.TriggerOnUse, Target: selected.Target, Conditions: selected.Conditions, Operations: []gameplayeffects.Operation{selected.Operation}}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{block})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Effect ON_USE không compile được"})
		return
	}
	cost := gamefx.EnergyCost(&player, &spec)
	outcome, execErr := executeWorldUseEffect(tx, &player, compiled, gameplayeffects.Context{Rank: gu.Rank}, gu.Rank, cost, houseName, []string{req.CoreGuInstanceID})
	if execErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": execErr.Error()})
		return
	}
	if _, err := tx.Exec(`UPDATE players SET hp=$1,current_mana=$2,statuses=$3,version=version+1,state_revision=state_revision+1 WHERE id=$4`, player.HP, player.CurrentMana, player.Statuses, playerID); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể lưu trạng thái nhân vật", err))
		return
	}
	auditlog.Log(tx, auditlog.Entry{PlayerID: &playerID, Action: "gu_house.use", EntityType: "gu_house", EntityID: houseID, RequestSummary: "Dùng effect ON_USE từ Tiên Cổ Ốc " + houseName + ": " + outcome.EffectsSummary, Metadata: map[string]interface{}{"house_id": houseID, "core_gu_instance_id": gu.InstanceID, "effect_key": selected.Key, "mana_cost": cost}, IPAddress: c.ClientIP(), Success: true})
	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể hoàn tất effect Tiên Cổ Ốc", err))
		return
	}
	respondActionWithInventory(c, http.StatusOK, player, gin.H{"message": "Tiên Cổ Ốc [" + houseName + "] đã thi triển: " + outcome.EffectsSummary, "gu_house_id": houseID, "success": true})
}
