package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
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
		c.JSON(http.StatusForbidden, gin.H{"error": errLeader.Error()})
		return
	}

	if player.ActiveCombat != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đang trong chiến đấu! Hãy dứt điểm kẻ địch trước."})
		return
	}

	if player.CurrentRegion != req.NodeID {
		if errLimit := checkAndIncrementExploreLimit(tx, &player); errLimit != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": errLimit.Error()})
			return
		}
	}

	exploreEnergyCost := 5.0
	if isImmortalCultivator(&player) {
		exploreEnergyCost = 0.25
	}
	if player.CurrentMana < exploreEnergyCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": insufficientEnergyMessage(&player)})
		return
	}
	player.CurrentMana -= exploreEnergyCost

	nodeName := "Khu Vực Bí Ẩn"
	switch req.NodeID {
	case "qing_mao_mountain":
		nodeName = "Thanh Mao Sơn (Southern Border)"
	case "gu_luan_slope":
		nodeName = "Cổ Loạn Sườn (Southern Border)"
	case "three_kings_inheritance":
		nodeName = "Tam Vương Truyền Thừa (Southern Border)"
	}

	roll := rand.Intn(100)
	var logMsg string
	combatTriggered := false

	if roll < 40 {
		stonesFound := 1 + rand.Intn(3)
		player.TempLoot.PrimevalStones += float64(stonesFound)
		logMsg = fmt.Sprintf("Khám phá hoàn tất tại %s. Đạo hữu may mắn phát hiện thấy %d viên Nguyên Thạch (được giữ tạm thời trong bản đồ)!", nodeName, stonesFound)
	} else if roll < 70 {
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
		logMsg = fmt.Sprintf("Nguy hiểm! Đột nhiên từ bụi cỏ rậm rạp tại %s, một con %s lao ra tấn công đạo hữu!", nodeName, enemy.EnemyName)

		hasHeavenSecret := false
		for _, gu := range player.GuInventory {
			if gu.GuID == "GU_HEAVEN_SECRET" && gu.Equipped && gu.Status == "ACTIVE" {
				hasHeavenSecret = true
				break
			}
		}
		if hasHeavenSecret && monTemplate.LootTableID != "" {
			lt, errLt := db.LoadLootTable(tx, monTemplate.LootTableID)
			if errLt == nil {
				extraLootMsg := " [Thiên Cơ hiển lộ - Tỉ lệ rơi đồ: "
				first := true
				for _, entry := range lt.Entries {
					if !first {
						extraLootMsg += ", "
					}
					if entry.ItemID != "" {
						extraLootMsg += fmt.Sprintf("%s (%.1f%%)", entry.ItemID, entry.Chance*100)
					} else if entry.GuID != "" {
						extraLootMsg += fmt.Sprintf("%s (%.1f%%)", entry.GuID, entry.Chance*100)
					}
					first = false
				}
				extraLootMsg += "]"
				logMsg += extraLootMsg
			}
		}
	} else if roll < 90 {
		foods := []string{"moon_orchid_petal", "green_bamboo_wine", "jade_stone", "boar_meat"}
		foodNames := []string{"Cánh hoa Nguyệt Lan", "Rượu Thanh Trúc", "Ngọc Thạch", "Thịt Lợn Rừng"}
		idx := rand.Intn(len(foods))
		foodID := foods[idx]
		foodName := foodNames[idx]

		if player.TempLoot.MaterialsInventory == nil {
			player.TempLoot.MaterialsInventory = make(models.MaterialsInventory)
		}
		player.TempLoot.MaterialsInventory[foodID]++
		logMsg = fmt.Sprintf("Đi qua khu rừng rậm ở %s, đạo hữu nhặt được nguyên liệu trân quý: 1x %s (được giữ tạm thời trong bản đồ).", nodeName, foodName)
	} else {
		logMsg = fmt.Sprintf("Đạo hữu cẩn thận thăm dò địa khu %s nhưng cảnh vật hoang tàn, không tìm thấy thứ gì có giá trị.", nodeName)
	}

	if player.CurrentRegion != req.NodeID {
		player.ExploredCells = models.ExploredCellsList{"50,50"}
	}
	player.CurrentRegion = req.NodeID
	player.X = 50
	player.Y = 50

	exploreRadius := 1
	for _, gu := range player.GuInventory {
		if gu.GuID == "GU_HEAVEN_SECRET" && gu.Equipped && gu.Status == "ACTIVE" {
			exploreRadius = 2
			break
		}
	}
	addExploredCells(&player, exploreRadius)

	qiGained := AddHumanQi(&player, 3.0)

	updateQuery := `
		UPDATE players
		SET current_mana = $1, primeval_stones = $2, active_combat = $3, current_region = $4, x = $5, y = $6, explored_cells = $7,
		    temp_loot = $8, explore_count = $9, last_explore_reset = $10, human_qi = $11
		WHERE id = $12
	`
	_, err = tx.Exec(updateQuery, player.CurrentMana, player.PrimevalStones, player.ActiveCombat, player.CurrentRegion, player.X, player.Y, player.ExploredCells,
		player.TempLoot, player.ExploreCount, player.LastExploreReset, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save exploration progress"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	responseWithPlayer(c, http.StatusOK, player, gin.H{
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
	if player.ExploreCount >= 10 {
		return fmt.Errorf("Đạo hữu đã hết lượt khám phá hôm nay (tối đa 10 lượt/ngày).")
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
						if tCount >= 5 {
							return fmt.Errorf("Thành viên trong tổ đội (%s) đã hết lượt khám phá hôm nay (tối đa 5 lượt/ngày).", m.Username)
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
