package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/util"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

type ExploreRequest struct {
	PlayerID string `json:"player_id" binding:"required"`
	NodeID   string `json:"node_id" binding:"required"`
}

func ExploreNode(c *gin.Context) {
	var req ExploreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	playerUUID, err := uuid.Parse(req.PlayerID)
	if err != nil || playerUUID != authID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Aperture mismatch"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lock player Aperture"})
		return
	}

	if errLeader := checkPartyLeaderOrSolo(tx, &player); errLeader != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": errLeader.Error()})
		return
	}

	if player.HP <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đang trong trạng thái trọng thương, không thể khám phá!"})
		return
	}

	if player.ActiveCombat != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đang trong chiến đấu! Hãy dứt điểm kẻ địch trước."})
		return
	}
	destinationZone := ZoneByID(req.NodeID)
	if destinationZone == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Khu vực du ngoạn không tồn tại."})
		return
	}
	var mapInstanceID sql.NullString
	if err := tx.QueryRow(`SELECT map_instance_id::text FROM players WHERE id=$1`, player.ID).Scan(&mapInstanceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra bản đồ hiện tại"})
		return
	}
	if mapInstanceID.Valid && mapInstanceID.String != "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Hãy rời bí cảnh bằng cổng thoát trước khi du ngoạn sang khu vực khác"})
		return
	}

	if player.CurrentRegion != req.NodeID {
		if err := checkRegionAction(player.CurrentRegion, "teleport"); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if err := checkRegionAction(req.NodeID, "teleport"); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if err := checkRegionEntry(req.NodeID, player.Rank, player.GuInventory); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if errLimit := checkAndIncrementExploreLimit(tx, &player); errLimit != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": errLimit.Error()})
			return
		}
	}

	exploreEnergyCost := 5.0
	if util.IsImmortalPlayer(&player) {
		exploreEnergyCost = 0.25
	}
	if player.CurrentMana < exploreEnergyCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": insufficientEnergyMessage(&player)})
		return
	}
	player.CurrentMana -= exploreEnergyCost

	nodeName := "Khu Vực Bí Ẩn"
	nodeName = destinationZone.Name

	roll := rand.Intn(100)
	// DISGUISE shrinks the random-monster window (70% threshold)
	// while leaving the resource window (<40%) untouched.
	monsterThreshold := 70 - int(PlayerDisguiseLevel(&player))
	if monsterThreshold < 40 {
		monsterThreshold = 40
	}
	var logMsg string
	combatTriggered := false

	if roll < 40 {
		stonesFound := 1 + rand.Intn(3)
		player.PrimevalStones += float64(stonesFound)
		logMsg = fmt.Sprintf("Khám phá hoàn tất tại %s. Đạo hữu may mắn phát hiện thấy %d viên Nguyên Thạch!", nodeName, stonesFound)
	} else if roll < monsterThreshold {
		combatTriggered = true

		monsterID, minRank, maxRank, err := db.PickRandomMonsterFromRegion(tx, req.NodeID)
		if err != nil || monsterID == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể chọn quái vật từ cơ sở dữ liệu"})
			return
		}
		monTemplate, err := db.LoadMonsterTemplate(tx, monsterID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải quái vật từ cơ sở dữ liệu"})
			return
		}

		rank := minRank
		if maxRank > minRank {
			rank += rand.Intn(maxRank - minRank + 1)
		}
		subRank := rand.Intn(5) + 1

		enemy := models.BuildCombatStateFromMonster(monTemplate, rank, subRank)

		player.ActiveCombat = StartCombat(tx, &enemy, &player)

		// Check if player has skip_battle enabled
		if player.CombatSettings.SkipBattle {
			// Auto-resolve combat using saved settings
			var afterCombatCommit func()
			winner, _, skipSummary, pendingNotifications, err := AutoResolveCombat(tx, &player, &afterCombatCommit)
			if err != nil {
				logger.S.Errorf("Auto-resolve combat failed: %v", err)
				_ = tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Tự động chiến đấu thất bại: " + err.Error()})
				return
			} else {
				combatTriggered = false // Combat was auto-resolved, no need for frontend combat screen
				if winner == "Phe Ta" {
					logMsg = fmt.Sprintf("Tự động chiến thắng %s tại %s! Nhân Khí +3.", enemy.EnemyName, nodeName)
				} else {
					logMsg = fmt.Sprintf("Tự động chiến đấu với %s tại %s thất bại! Đạo hữu bị truyền tống về an toàn.", enemy.EnemyName, nodeName)
				}
				// Add loot info to log message if available
				if skipSummary != nil {
					if loot, ok := skipSummary["loot"].([]string); ok && len(loot) > 0 {
						logMsg += " " + strings.Join(loot, " ")
					}
					if mineOccupied, ok := skipSummary["mine_occupied"].(int); ok && mineOccupied > 0 {
						logMsg += fmt.Sprintf(" Đã chiếm mỏ %d!", mineOccupied)
					}
				}

				// Return early with skip result
				zone := ZoneByID(req.NodeID)
				if zone == nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Khu vực du ngoạn không tồn tại."})
					return
				}
				player.CurrentRegion = zone.ID
				player.X = zone.EntryX
				player.Y = zone.EntryY
				recordVisitedSubZone(&player, SubZoneKey(player.X, player.Y))

				qiGained := AddHumanQi(&player, 3.0)

				updateQuery := `
					UPDATE players
					SET current_mana = $1, primeval_stones = $2, active_combat = NULL, current_region = $3, x = $4, y = $5,
					    explore_count = $6, last_explore_reset = $7, human_qi = $8, visited_subzones = $9
					WHERE id = $10
				`
				_, err = tx.Exec(updateQuery, player.CurrentMana, player.PrimevalStones, player.CurrentRegion, player.X, player.Y,
					player.ExploreCount, player.LastExploreReset, player.HumanQi, player.VisitedSubZones, player.ID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save exploration progress"})
					return
				}

				if err = tx.Commit(); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
					return
				}
				if afterCombatCommit != nil {
					afterCombatCommit()
				}
				for _, notification := range pendingNotifications {
					notify.Publish(notification)
				}

				setWorldPosFromPlayer(&player)

				respondActionWithInventory(c, http.StatusOK, player, gin.H{
					"log":              logMsg + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
					"combat_triggered": false,
					"skip_result":      skipSummary,
				})
				return
			}
		} else {
			logMsg = fmt.Sprintf("Nguy hiểm! Đột nhiên từ bụi cỏ rậm rạp tại %s, một con %s lao ra tấn công đạo hữu!", nodeName, enemy.EnemyName)
		}

		hasHeavenSecret := false
		for _, gu := range player.GuInventory {
			if models.InventoryHasGuCapability(models.GuInventory{gu}, "LOOT_PREVIEW", true) {
				hasHeavenSecret = true
				break
			}
		}
		if hasHeavenSecret && monTemplate.LootTableID != "" {
			logMsg += heavenSecretLootPreview(tx, monTemplate.LootTableID, rank)
		}
	} else if roll < 90 {
		foods := []string{"moon_orchid_petal", "green_bamboo_wine", "jade_stone", "boar_meat"}
		foodNames := []string{"Cánh hoa Nguyệt Lan", "Rượu Thanh Trúc", "Ngọc Thạch", "Thịt Lợn Rừng"}
		idx := rand.Intn(len(foods))
		foodID := foods[idx]
		foodName := foodNames[idx]

		if err := InventorySvc.AddItem(tx, player.ID.String(), foodID, "materials", 1); err != nil {
			logger.S.Errorf("Failed to add explore food %s to player %s: %v", foodID, player.ID, err)
			_ = tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thêm nguyên liệu vào túi"})
			return
		}
		logMsg = fmt.Sprintf("Đi qua khu rừng rậm ở %s, đạo hữu nhặt được nguyên liệu trân quý: 1x %s.", nodeName, foodName)
	} else {
		logMsg = fmt.Sprintf("Đạo hữu cẩn thận thăm dò địa khu %s nhưng cảnh vật hoang tàn, không tìm thấy thứ gì có giá trị.", nodeName)
	}

	zone := ZoneByID(req.NodeID)
	if zone == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Khu vực du ngoạn không tồn tại."})
		return
	}
	player.CurrentRegion = zone.ID
	player.X = zone.EntryX
	player.Y = zone.EntryY
	recordVisitedSubZone(&player, SubZoneKey(player.X, player.Y))

	qiGained := AddHumanQi(&player, 3.0)

	updateQuery := `
		UPDATE players
		SET current_mana = $1, primeval_stones = $2, active_combat = $3, current_region = $4, x = $5, y = $6,
		    explore_count = $7, last_explore_reset = $8, human_qi = $9, visited_subzones = $10
		WHERE id = $11
	`
	_, err = tx.Exec(updateQuery, player.CurrentMana, player.PrimevalStones, player.ActiveCombat, player.CurrentRegion, player.X, player.Y,
		player.ExploreCount, player.LastExploreReset, player.HumanQi, player.VisitedSubZones, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save exploration progress"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	setWorldPosFromPlayer(&player)

	respondActionWithInventory(c, http.StatusOK, player, gin.H{
		"log":              logMsg + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
		"combat_triggered": combatTriggered,
	})
}

func checkAndIncrementExploreLimit(tx *sql.Tx, player *models.Player) error {
	now := time.Now()

	resetIfNeeded := func(pCount *int, pReset *time.Time, pID uuid.UUID) error {
		y1, m1, d1 := now.Date()
		y2, m2, d2 := pReset.Date()
		if y1 != y2 || m1 != m2 || d1 != d2 {
			*pCount = 0
			*pReset = now
			_, err := tx.Exec("UPDATE players SET explore_count = 0, last_explore_reset = $1 WHERE id = $2", now, pID)
			return err
		}
		return nil
	}

	if err := resetIfNeeded(&player.ExploreCount, &player.LastExploreReset, player.ID); err != nil {
		return err
	}
	if player.ExploreCount >= 20 {
		return fmt.Errorf("Đạo hữu đã hết lượt khám phá hôm nay (tối đa 20 lượt/ngày).")
	}

	if player.PartyID != nil {
		var membersRaw []byte
		errPart := tx.QueryRow("SELECT members FROM parties WHERE id = $1", player.PartyID).Scan(&membersRaw)
		if errPart == nil {
			var members []PartyMember
			json.Unmarshal(membersRaw, &members)
			for _, m := range members {
				mUUID, errParse := uuid.Parse(m.ID)
				if errParse == nil && mUUID != player.ID {
					var tCount int
					var tReset time.Time
					errLoad := tx.QueryRow("SELECT explore_count, last_explore_reset FROM players WHERE id = $1 FOR UPDATE", mUUID).Scan(&tCount, &tReset)
					if errLoad == nil {
						y1, m1, d1 := now.Date()
						y2, m2, d2 := tReset.Date()
						if y1 != y2 || m1 != m2 || d1 != d2 {
							tCount = 0
							tReset = now
							_, errUpd := tx.Exec("UPDATE players SET explore_count = 0, last_explore_reset = $1 WHERE id = $2", now, mUUID)
							if errUpd != nil {
								return errUpd
							}
						}
						if tCount >= 20 {
							return fmt.Errorf("Thành viên trong tổ đội (%s) đã hết lượt khám phá hôm nay (tối đa 20 lượt/ngày).", m.Username)
						}
					}
				}
			}

			for _, m := range members {
				mUUID, errParse := uuid.Parse(m.ID)
				if errParse == nil && mUUID != player.ID {
					_, errInc := tx.Exec("UPDATE players SET explore_count = explore_count + 1 WHERE id = $1", mUUID)
					if errInc != nil {
						return errInc
					}
				}
			}
		}
	}

	player.ExploreCount++
	player.LastExploreReset = now
	return nil
}
