package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/playerstate"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type EquipGuRequest struct {
	InstanceID string `json:"instance_id" binding:"required"`
	Equip      bool   `json:"equip"`
}

var (
	getPlayerStatsEvaluate = evaluateManaTx
	getPlayerStatsRespond  = responseWithPlayer
)

// CalculatePlayerStats computes dynamic stats based on equipped active Gu bugs and passive mods

func flushGrowthOnUnequip(player *models.Player, gu *models.GuInstance, spec *models.GuDictionaryItem) {
	growth, hasGrowth := models.GrowthProfile(*spec)
	if !hasGrowth || gu.EquippedAt == nil {
		return
	}
	tsm := player.TimeSpeedMultiplier + models.EquippedTimeSpeedBonus(player.GuInventory)
	if tsm <= 0 {
		tsm = 1.0
	}
	elapsed := time.Since(*gu.EquippedAt).Hours() * tsm
	if elapsed <= 0 {
		return
	}
	atk, def, hp, spd := ApplyGrowthProgress(gu, spec, elapsed)
	if atk != 0 || def != 0 || hp != 0 || spd != 0 {
		player.GrowthData.Boosts.Attack += atk
		player.GrowthData.Boosts.Defense += def
		player.GrowthData.Boosts.HP += hp
		player.GrowthData.Boosts.Speed += spd
		player.BaseHP += hp
	}
	if gu.GrowthHours >= growth.HoursToMax {
		player.GrowthData.FullyGrown = append(player.GrowthData.FullyGrown, models.FullyGrownInfo{
			GuID:   gu.GuID,
			Rank:   gu.Rank,
			Attack: growth.MaxAttack, Defense: growth.MaxDefense,
			HP: growth.MaxHP, Speed: growth.MaxSpeed,
		})
	}
}

// cleanupInvalidEquipped unequips every non-vital ACTIVE equipped Gu that is no
// longer valid to wear: unknown spec, Consumable type, or invalid tier rank
// (e.g. Mortal user equipped with Immortal Gu). Returns the removed instances
// so callers can persist and audit the cleanup.
func cleanupInvalidEquipped(player *models.Player) []models.GuInstance {
	cleaned := []models.GuInstance{}
	for i := range player.GuInventory {
		gu := &player.GuInventory[i]
		if !gu.Equipped || gu.Status != "ACTIVE" || gu.IsVital {
			continue
		}
		spec, ok := models.GetGu(gu.GuID, gu.Rank)
		isOverRankPassive := ok && strings.EqualFold(spec.Type, "PASSIVE") && gu.Rank > player.Rank
		if ok && spec.Type != "Consumable" && gamefx.CanEquipGu(player.Rank, spec.Rank) && !isOverRankPassive {
			continue
		}
		gu.Equipped = false
		if ok {
			flushGrowthOnUnequip(player, gu, &spec)
		}
		gu.EquippedAt = nil
		cleaned = append(cleaned, *gu)
	}
	return cleaned
}

// EquipGu toggles whether a Gu instance is equipped
func EquipGu(c *gin.Context) {
	var req EquipGuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var action string
	var statusMsg string
	var guName string
	var cleaned []models.GuInstance

	player, err := MutatePlayer(authID, PlayerMutationOptions{
		Scope:        playerstate.GuFull,
		EventDomains: []PlayerEventDomain{PlayerDomain, GuInventoryDomain, CombatStatsDomain},
	}, func(tx *sql.Tx, player *models.Player) error {
		targetIdx := -1
		for i, gu := range player.GuInventory {
			if gu.InstanceID == req.InstanceID {
				targetIdx = i
			}
		}

		if targetIdx == -1 {
			return NewMutationError(http.StatusNotFound, "Gu bug not found in inventory")
		}

		targetGu := &player.GuInventory[targetIdx]
		if targetGu.Status == "DEAD" {
			return NewMutationError(http.StatusBadRequest, "Cannot equip a DEAD Gu bug")
		}
		if req.Equip && (targetGu.Status == "DAMAGED" || targetGu.Durability <= 0) {
			return NewMutationError(http.StatusBadRequest, "Cổ trùng đã bị trọng thương hư hại (độ bền 0), cần tu bổ trước khi trang bị.")
		}
		if req.Equip && targetGu.RefinementProgress < 100 {
			return NewMutationError(http.StatusBadRequest, "Cổ trùng chưa được luyện hóa hoàn tất (tiến độ cần đạt 100%).")
		}

		spec, specOk := models.GetGu(targetGu.GuID, targetGu.Rank)
		if req.Equip && targetGu.Rank > player.Rank && (targetGu.IsVital || (specOk && strings.EqualFold(spec.Type, "PASSIVE"))) {
			guKind := "Cổ trùng bị động"
			if targetGu.IsVital {
				guKind = "Bản Mệnh Cổ"
			}
			return NewMutationError(http.StatusBadRequest, fmt.Sprintf("Không thể trang bị %s Rank %d vì vượt quá tu vi hiện tại (Rank %d).", guKind, targetGu.Rank, player.Rank))
		}

		// Vital Gu is always equipped and active — cannot be unequipped
		if targetGu.IsVital && !req.Equip {
			return NewMutationError(http.StatusBadRequest, "Bản Mệnh Cổ luôn luôn được trang bị, không thể tháo ra.")
		}

		// Re-validate the whole equipped list on every equip/unequip. Any non-vital
		// Gu that is no longer valid to wear (rank above the player\x27s rank after a
		// cultivation drop, Consumable type, or unknown spec) is auto-unequipped.
		cleaned = cleanupInvalidEquipped(player)

		targetCleaned := false
		for _, gu := range cleaned {
			if gu.InstanceID == req.InstanceID {
				targetCleaned = true
				break
			}
		}

		guName = targetGu.GuID
		if req.Equip && specOk {
			if err := checkRegionGu(player.CurrentRegion, targetGu.GuID, targetGu.Rank, spec.Path); err != nil {
				return NewMutationError(http.StatusForbidden, err.Error())
			}
		}
		if specOk {
			guName = spec.Name
		}

		isFullyGrown := false
		_, specHasGrowth := models.GrowthProfile(spec)
		if specOk && specHasGrowth {
			for _, fg := range player.GrowthData.FullyGrown {
				if fg.GuID == targetGu.GuID && fg.Rank == targetGu.Rank {
					isFullyGrown = true
					break
				}
			}
		}

		action = "gu.unequip"
		statusMsg = fmt.Sprintf("Tháo %s thành công!", guName)
		if req.Equip {
			action = "gu.equip"
			if targetCleaned {
				statusMsg = fmt.Sprintf("%s không hợp lệ với tu vi hiện tại, đã tự động tháo khỏi trang bị.", guName)
			} else if targetGu.Equipped {
				statusMsg = "Gu bug is already equipped"
			} else {
				equippedCount := models.CountEquippedActive(player.GuInventory)
				if equippedCount >= db.MaxEquippedGus {
					return NewMutationError(http.StatusBadRequest, fmt.Sprintf("Aperture capacity full! Cannot equip more than %d Gu bugs.", db.MaxEquippedGus))
				}

				if specOk && spec.Type == "Consumable" {
					return NewMutationError(http.StatusBadRequest, "Không thể trang bị Cổ trùng loại Consumable.")
				}

				if specOk && !gamefx.CanEquipGu(player.Rank, spec.Rank) {
					return NewMutationError(http.StatusBadRequest, fmt.Sprintf("Phàm Cổ Sư (%s) không thể trang bị Tiên Cổ %s (Rank %d)!", models.GetRankName(player.Rank), spec.Name, spec.Rank))
				}

				targetGu.Equipped = true
				statusMsg = fmt.Sprintf("Trang bị %s thành công!", guName)
				if specOk {
					if msg, err := HandleGuEvent(player, spec, *targetGu, "ON_EQUIP"); err == nil && msg != "" {
						statusMsg += " " + msg
					}
				}
				if specOk && specHasGrowth && !isFullyGrown {
					now := time.Now()
					targetGu.EquippedAt = &now
				}
			}
		} else {
			targetGu.Equipped = false
			if specOk {
				flushGrowthOnUnequip(player, targetGu, &spec)
			}
		}

		RebuildEquippedGus(player)

		if len(cleaned) > 0 {
			statusMsg += fmt.Sprintf(" Đã tự động tháo %d Cổ trùng không hợp lệ.", len(cleaned))
		}

		if err := db.UpdatePlayerGu(tx, *targetGu); err != nil {
			return fmt.Errorf("failed to update player state: %w", err)
		}
		for _, gu := range cleaned {
			if gu.InstanceID == req.InstanceID {
				continue
			}
			if err := db.UpdatePlayerGu(tx, gu); err != nil {
				return fmt.Errorf("failed to update player state: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		HandleMutationError(c, err)
		return
	}

	auditlog.Logf(db.DB, &authID, action, "gu=%s instance=%s", guName, req.InstanceID)
	if len(cleaned) > 0 {
		cleanedIDs := make([]string, 0, len(cleaned))
		for _, gu := range cleaned {
			cleanedIDs = append(cleanedIDs, gu.InstanceID)
		}
		auditlog.Logf(db.DB, &authID, "gu.unequip_invalid", "count=%d instances=%s", len(cleaned), strings.Join(cleanedIDs, ","))
	}

	respondActionGuOnly(c, http.StatusOK, *player, gin.H{"message": statusMsg})
}

// UnequipAllGu unequips all equipped Gu except the Vital Gu.
func UnequipAllGu(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var unequippedCount int

	player, err := MutatePlayer(authID, PlayerMutationOptions{
		Scope:        playerstate.GuFull,
		EventDomains: []PlayerEventDomain{PlayerDomain, GuInventoryDomain, CombatStatsDomain},
	}, func(tx *sql.Tx, player *models.Player) error {
		var toUpdate models.GuInventory
		for i := range player.GuInventory {
			gu := &player.GuInventory[i]
			if gu.Equipped && !gu.IsVital {
				gu.Equipped = false
				unequippedCount++

				// Unequip — flush pending growth time
				spec, specOk := models.GetGu(gu.GuID, gu.Rank)
				growth, hasGrowth := models.GrowthProfile(spec)
				hasGrowth = specOk && hasGrowth
				if hasGrowth && gu.EquippedAt != nil {
					tsm := player.TimeSpeedMultiplier + models.EquippedTimeSpeedBonus(player.GuInventory)
					if tsm <= 0 {
						tsm = 1.0
					}
					elapsed := time.Since(*gu.EquippedAt).Hours() * tsm
					if elapsed > 0 {
						atk, def, hp, spd := ApplyGrowthProgress(gu, &spec, elapsed)
						if atk != 0 || def != 0 || hp != 0 || spd != 0 {
							player.GrowthData.Boosts.Attack += atk
							player.GrowthData.Boosts.Defense += def
							player.GrowthData.Boosts.HP += hp
							player.GrowthData.Boosts.Speed += spd
							player.BaseHP += hp
						}
						if gu.GrowthHours >= growth.HoursToMax {
							player.GrowthData.FullyGrown = append(player.GrowthData.FullyGrown, models.FullyGrownInfo{
								GuID:   gu.GuID,
								Rank:   gu.Rank,
								Attack: growth.MaxAttack, Defense: growth.MaxDefense,
								HP: growth.MaxHP, Speed: growth.MaxSpeed,
							})
						}
					}
				}

				toUpdate = append(toUpdate, *gu)
			}
		}

		if unequippedCount == 0 {
			return nil
		}

		if len(toUpdate) > 0 {
			if err := db.UpdatePlayerGusBulk(tx, toUpdate); err != nil {
				return fmt.Errorf("failed to update Gu state: %w", err)
			}
		}

		RebuildEquippedGus(player)
		return nil
	})
	if err != nil {
		HandleMutationError(c, err)
		return
	}

	if unequippedCount == 0 {
		respondActionGuOnly(c, http.StatusOK, *player, gin.H{"message": "Không có Cổ trùng nào có thể tháo."})
		return
	}

	auditlog.Logf(db.DB, &authID, "gu.unequip_all", "count=%d", unequippedCount)

	respondActionGuOnly(c, http.StatusOK, *player, gin.H{"message": fmt.Sprintf("Đã tháo thành công %d Cổ trùng!", unequippedCount)})
}

// getSealingStoneCostByRank returns required sealing_stone quantity according to Rank P factor
func getSealingStoneCostByRank(rank int) int64 {
	return gamefx.SealingStoneCostByRank(rank)
}

// UpdateGuStatusRequest defines body to change status of owned Gu instance
type UpdateGuStatusRequest struct {
	InstanceID string `json:"instance_id" binding:"required"`
	Status     string `json:"status" binding:"required"` // "ACTIVE", "SLEEPING", "SEALED"
}

// UpdateGuStatus changes Gu status between ACTIVE, SLEEPING, SEALED
func UpdateGuStatus(c *gin.Context) {
	var req UpdateGuStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Status != "ACTIVE" && req.Status != "SLEEPING" && req.Status != "SEALED" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Trạng thái không hợp lệ. Chỉ chấp nhận ACTIVE, SLEEPING hoặc SEALED."})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var neededStones int64
	var targetGu models.GuInstance

	player, err := MutatePlayer(authID, PlayerMutationOptions{
		Scope:        playerstate.GuFull,
		EventDomains: []PlayerEventDomain{PlayerDomain, GuInventoryDomain, CombatStatsDomain},
	}, func(tx *sql.Tx, player *models.Player) error {
		targetIdx := -1
		for idx, inst := range player.GuInventory {
			if inst.InstanceID == req.InstanceID {
				targetIdx = idx
				break
			}
		}

		if targetIdx == -1 {
			return NewMutationError(http.StatusNotFound, "Gu bug not found in inventory")
		}

		target := &player.GuInventory[targetIdx]
		if target.Status == "DEAD" {
			return NewMutationError(http.StatusBadRequest, "Không thể thay đổi trạng thái của Cổ trùng ĐÃ CHẾT.")
		}

		// Vital Gu is always ACTIVE — cannot be put to sleep or sealed
		if target.IsVital && req.Status != "ACTIVE" {
			return NewMutationError(http.StatusBadRequest, "Bản Mệnh Cổ luôn luôn ở trạng thái ACTIVE, không thể thay đổi.")
		}

		// If transitioning away from ACTIVE (e.g. to SLEEPING or SEALED), force unequip if equipped.
		// Flush pending growth time into permanent boosts first, mirroring EquipGu.
		if req.Status != "ACTIVE" && target.Equipped {
			target.Equipped = false
			if spec, ok := models.GetGu(target.GuID, target.Rank); ok {
				flushGrowthOnUnequip(player, target, &spec)
			}
			target.EquippedAt = nil
		}

		if req.Status == "SEALED" && target.Status != "SEALED" {
			if target.Equipped {
				return NewMutationError(http.StatusBadRequest, "Không thể phong ấn Cổ trùng đang được trang bị. Hãy tháo ra trước.")
			}
			neededStones = getSealingStoneCostByRank(target.Rank)
			qty, err := InventorySvc.GetQuantity(tx, player.ID.String(), "sealing_stone")
			if err != nil || qty < neededStones {
				return NewMutationError(http.StatusBadRequest, fmt.Sprintf("Không đủ sealing_stone để phong ấn Cổ trùng (Cần %d Đá Phong Ấn).", neededStones))
			}
			if err := InventorySvc.RemoveItem(tx, player.ID.String(), "sealing_stone", "materials", neededStones); err != nil {
				return fmt.Errorf("failed to consume sealing_stone: %w", err)
			}
		}

		target.Status = req.Status

		// Recalculate equipped_gus list
		RebuildEquippedGus(player)

		if err := db.UpdatePlayerGu(tx, *target); err != nil {
			return fmt.Errorf("failed to update Gu inventory status: %w", err)
		}

		targetGu = *target
		return nil
	})
	if err != nil {
		HandleMutationError(c, err)
		return
	}

	if req.Status == "SEALED" {
		if neededStones < 1 {
			neededStones = getSealingStoneCostByRank(targetGu.Rank)
		}
		auditlog.Log(db.DB, auditlog.Entry{
			PlayerID:       &authID,
			Action:         "gu.seal",
			EntityType:     "gu_instance",
			EntityID:       targetGu.InstanceID,
			RequestSummary: fmt.Sprintf("Dùng %dx Phong Ấn Thạch phong ấn Cổ trùng %s (%s)", neededStones, targetGu.GuID, targetGu.InstanceID),
			Metadata: map[string]interface{}{
				"gu_instance_id":  targetGu.InstanceID,
				"gu_id":           targetGu.GuID,
				"item_consumed":   "sealing_stone",
				"stones_consumed": neededStones,
			},
			IPAddress: c.ClientIP(),
			Success:   true,
		})
	}

	respondActionWithInventory(c, http.StatusOK, *player, gin.H{
		"message": fmt.Sprintf("Cập nhật trạng thái Cổ trùng thành %s thành công!", req.Status),
	})
}

// GetPlayerStatsHandler returns the calculated stats for the current player
func GetPlayerStatsHandler(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := getPlayerStatsEvaluate(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player Aperture"})
		return
	}

	stats := CalculatePlayerStats(&player)
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	getPlayerStatsRespond(c, http.StatusOK, player, gin.H{
		"stats": stats,
	})
}
