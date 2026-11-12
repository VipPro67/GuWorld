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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()
	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải nhân vật"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải Tiên Cổ Ốc"})
		return
	}
	if durability <= 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Tiên Cổ Ốc đã hư hại, cần tu sửa trước"})
		return
	}
	if err := validateGuHouseActivationProfile(profileRaw); err != nil {
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác minh Cổ hạch tâm"})
		return
	}
	if !reserved {
		c.JSON(http.StatusConflict, gin.H{"error": "Cổ hạch tâm không còn được Tiên Cổ Ốc giữ chỗ; hãy tháo và xuất chiến lại"})
		return
	}
	owned, err := loadOwnedKillerMoveGus(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải Cổ hạch tâm"})
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
	if player.CurrentMana < cost {
		c.JSON(http.StatusBadRequest, gin.H{"error": insufficientEnergyMessage(&player)})
		return
	}
	snapshot := gameplayeffects.Snapshot{ID: player.ID.String(), HP: player.HP, MaxHP: gamefx.EffectiveMaxHP(&player), Mana: player.CurrentMana, MaxMana: gamefx.EffectiveMaxMana(&player)}
	result, err := compiled.Evaluate(gameplayeffects.Context{Actor: snapshot, Rank: gu.Rank}, gameplayeffects.Event{Trigger: gameplayeffects.TriggerOnUse})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Không thể đánh giá effect ON_USE"})
		return
	}
	player.CurrentMana -= cost
	healed, mana, flight, stealth, haste, err := applyWorldUseCommands(&player, result.Commands, gu.Rank, houseName)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Executor ON_USE từ chối effect không hỗ trợ"})
		return
	}
	if healed == 0 && mana == 0 && flight == 0 && stealth == 0 && haste == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Điều kiện effect không thỏa hoặc không tạo tác động"})
		return
	}
	if _, err := tx.Exec(`UPDATE player_gu SET hunger=GREATEST(0,hunger-1),updated_at=NOW() WHERE id=$1 AND player_id=$2`, req.CoreGuInstanceID, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể trừ độ no Cổ hạch tâm"})
		return
	}
	for i := range player.GuInventory {
		if player.GuInventory[i].InstanceID == req.CoreGuInstanceID {
			player.GuInventory[i].Hunger = max(0, player.GuInventory[i].Hunger-1)
		}
	}
	if _, err := tx.Exec(`UPDATE players SET hp=$1,current_mana=$2,statuses=$3 WHERE id=$4`, player.HP, player.CurrentMana, player.Statuses, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu trạng thái nhân vật"})
		return
	}
	auditlog.Log(tx, auditlog.Entry{PlayerID: &playerID, Action: "gu_house.use", EntityType: "gu_house", EntityID: houseID, RequestSummary: "Dùng effect ON_USE từ Tiên Cổ Ốc " + houseName, Metadata: map[string]interface{}{"house_id": houseID, "core_gu_instance_id": gu.InstanceID, "effect_key": selected.Key, "mana_cost": cost}, IPAddress: c.ClientIP(), Success: true})
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hoàn tất effect Tiên Cổ Ốc"})
		return
	}
	respondActionWithInventory(c, http.StatusOK, player, gin.H{"message": "Tiên Cổ Ốc [" + houseName + "] đã thi triển effect ngoài combat", "gu_house_id": houseID, "success": true})
}
