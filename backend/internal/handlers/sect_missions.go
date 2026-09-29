package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/pricing"
)

// GetSectMissions returns open missions
func GetSectMissions(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var m SectMember
	err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT sect_id, member_rank
		FROM sect_members
		WHERE player_id = $1
	`, playerID).Scan(&m.SectID, &m.MemberRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT id, sect_id, title, description, mission_type, target_id, target_amount, reward_contribution, required_member_rank, expires_at
		FROM sect_missions
		WHERE sect_id = $1 AND (expires_at IS NULL OR expires_at > now())
		ORDER BY required_member_rank, title
	`, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch missions"})
		return
	}
	defer rows.Close()

	var missions []SectMission
	for rows.Next() {
		var sm SectMission
		if err := rows.Scan(&sm.ID, &sm.SectID, &sm.Title, &sm.Description, &sm.MissionType, &sm.TargetID, &sm.TargetAmount, &sm.RewardContribution, &sm.RequiredMemberRank, &sm.ExpiresAt); err == nil {
			missions = append(missions, sm)
		}
	}

	c.JSON(http.StatusOK, missions)
}

// AcceptSectMission accepts a mission
func AcceptSectMission(c *gin.Context) {
	missionIDStr := c.Param("missionId")
	missionID, err := uuid.Parse(missionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mission ID format"})
		return
	}

	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Check if already has active mission
	var dummy uuid.UUID
	err = tx.QueryRow(`
		SELECT id FROM player_sect_missions
		WHERE player_id = $1 AND status = 'IN_PROGRESS'
		  AND (expires_at IS NULL OR expires_at > now())
		LIMIT 1
	`, playerID).Scan(&dummy)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đang thực hiện một nhiệm vụ khác!"})
		return
	}

	// Fetch mission details
	var sm SectMission
	err = tx.QueryRow(`
		SELECT id, sect_id, required_member_rank, expires_at
		FROM sect_missions
		WHERE id = $1
	`, missionID).Scan(&sm.ID, &sm.SectID, &sm.RequiredMemberRank, &sm.ExpiresAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy nhiệm vụ yêu cầu!"})
		return
	}

	// Verify player belongs to this sect
	var m SectMember
	err = tx.QueryRow(`
		SELECT member_rank
		FROM sect_members
		WHERE player_id = $1 AND sect_id = $2
	`, playerID, sm.SectID).Scan(&m.MemberRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nhiệm vụ này thuộc về tông môn khác!"})
		return
	}

	// Verify rank requirement
	rankMap := map[string]int{
		"NGOAI_MON":   1,
		"NOI_MON":     2,
		"TINH_ANH":    3,
		"CHAN_TRUYEN": 4,
		"TRUONG_LAO":  5,
		"HO_PHAP":     6,
	}

	if rankMap[m.MemberRank] < rankMap[sm.RequiredMemberRank] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cấp bậc đệ tử của đạo hữu không đủ để nhận nhiệm vụ này!"})
		return
	}

	// Insert player mission
	_, err = tx.Exec(`
		INSERT INTO player_sect_missions (player_id, mission_id, current_progress, status, expires_at)
		VALUES ($1, $2, 0, 'IN_PROGRESS', $3)
	`, playerID, sm.ID, sm.ExpiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to accept mission: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Nhận nhiệm vụ tông môn thành công!"})
}

// CompleteSectMission completes active mission
func CompleteSectMission(c *gin.Context) {
	missionIDStr := c.Param("missionId")
	missionID, err := uuid.Parse(missionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mission ID format"})
		return
	}

	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player"})
		return
	}

	var psm PlayerSectMission
	err = tx.QueryRow(`
		SELECT id, current_progress, status
		FROM player_sect_missions
		WHERE player_id = $1 AND mission_id = $2 AND status = 'IN_PROGRESS'
		  AND (expires_at IS NULL OR expires_at > now())
	`, playerID, missionID).Scan(&psm.ID, &psm.CurrentProgress, &psm.Status)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không có nhiệm vụ này ở trạng thái đang thực hiện!"})
		return
	}

	var sm SectMission
	err = tx.QueryRow(`
		SELECT id, title, mission_type, target_id, target_amount, reward_contribution
		FROM sect_missions
		WHERE id = $1
	`, missionID).Scan(&sm.ID, &sm.Title, &sm.MissionType, &sm.TargetID, &sm.TargetAmount, &sm.RewardContribution)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load mission parameters"})
		return
	}

	// Verify completion requirements
	if sm.MissionType == "GATHER" {
		qty, _ := InventorySvc.GetQuantity(tx, player.ID.String(), sm.TargetID)
		if qty < sm.TargetAmount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa thu thập đủ linh tài yêu cầu!"})
			return
		}
		// Deduct materials from player
		InventorySvc.RemoveItem(tx, player.ID.String(), sm.TargetID, "materials", sm.TargetAmount)

		// Add gathered materials to sect vault
		var memberForVault SectMember
		err = tx.QueryRow(`SELECT sect_id FROM sect_members WHERE player_id = $1`, playerID).Scan(&memberForVault.SectID)
		if err == nil {
			_, _ = tx.Exec(`
				INSERT INTO sect_vault_materials (sect_id, item_id, quantity)
				VALUES ($1, $2, $3)
				ON CONFLICT (sect_id, item_id) DO UPDATE SET
					quantity = sect_vault_materials.quantity + EXCLUDED.quantity,
					updated_at = NOW()
			`, memberForVault.SectID, sm.TargetID, sm.TargetAmount)
		}
	} else if sm.MissionType == "DONATE" {
		// Donate a Gu bug
		targetIdx := -1
		for idx, gi := range player.GuInventory {
			if gi.GuID == sm.TargetID {
				targetIdx = idx
				break
			}
		}

		if targetIdx == -1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không có Cổ trùng \"" + sm.TargetID + "\" yêu cầu trong túi!"})
			return
		}

		if player.GuInventory[targetIdx].IsVital || player.GuInventory[targetIdx].Equipped {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể nộp Cổ trùng đang trang bị hoặc Bản Mệnh Cổ!"})
			return
		}

		// Remove Gu from player
		donatedGuID := player.GuInventory[targetIdx].GuID
		donatedGuInstRank := player.GuInventory[targetIdx].Rank
		donatedGuHunger := float64(player.GuInventory[targetIdx].Hunger)
		donatedInstanceID := player.GuInventory[targetIdx].InstanceID
		player.GuInventory = append(player.GuInventory[:targetIdx], player.GuInventory[targetIdx+1:]...)
		if err := db.RemovePlayerGu(tx, donatedInstanceID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update gu inventory"})
			return
		}

		// Add donated Gu to sect vault
		var memberForVault SectMember
		err = tx.QueryRow(`SELECT sect_id FROM sect_members WHERE player_id = $1`, playerID).Scan(&memberForVault.SectID)
		if err == nil {
			var donatedGuRank int
			var donatedGuBase int64
			_ = tx.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM gu_dictionary WHERE id = $1 ORDER BY rank DESC LIMIT 1`, donatedGuID).Scan(&donatedGuBase, &donatedGuRank)
			if donatedGuRank == 0 {
				donatedGuRank = 1
			}
			vaultItemCost := int64(pricing.VaultCost(pricing.GuBasePrice(float64(donatedGuBase), donatedGuRank)))
			itemInstID := donatedInstanceID
			if itemInstID == "" {
				itemInstID = uuid.New().String()
			}
			rowID := fmt.Sprintf("%s:%s", memberForVault.SectID, itemInstID)
			instIDToStore := itemInstID
			if donatedGuInstRank < 6 {
				rowID = fmt.Sprintf("%s:%s:%d", memberForVault.SectID, donatedGuID, donatedGuInstRank)
				instIDToStore = ""
			}
			normHunger := normalizeVaultGuHunger(donatedGuID, donatedGuInstRank, donatedGuHunger)
			_, _ = tx.Exec(`
				INSERT INTO sect_vault_gu (id, sect_id, gu_id, rank, quantity, cost, hunger, instance_id)
				VALUES ($1, $2, $3, $4, 1, $5, $6, $7)
				ON CONFLICT (id) DO UPDATE SET
					quantity = sect_vault_gu.quantity + 1,
					cost = EXCLUDED.cost,
					hunger = GREATEST(sect_vault_gu.hunger, EXCLUDED.hunger),
					updated_at = NOW()
			`, rowID, memberForVault.SectID, donatedGuID, donatedGuInstRank, vaultItemCost, normHunger, instIDToStore)
		}
	}

	// Update mission status
	_, err = tx.Exec(`DELETE FROM player_sect_missions WHERE id = $1`, psm.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete mission"})
		return
	}

	// Add contribution points
	var m SectMember
	err = tx.QueryRow(`
		SELECT sect_id, contribution_points
		FROM sect_members
		WHERE player_id = $1
	`, playerID).Scan(&m.SectID, &m.ContributionPoints)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get contribution info"})
		return
	}

	m.ContributionPoints += int64(sm.RewardContribution)
	_, err = tx.Exec(`UPDATE sect_members SET contribution_points = $1 WHERE player_id = $2`, m.ContributionPoints, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add contribution points"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &playerID,
		Action:         "sect.mission_complete",
		EntityType:     "sect_mission",
		EntityID:       missionID.String(),
		RequestSummary: fmt.Sprintf("Hoàn thành nhiệm vụ tông môn: %s (%s) | Nộp: %d %s | Nhận %d Cống hiến", sm.Title, sm.MissionType, sm.TargetAmount, sm.TargetID, sm.RewardContribution),
		Metadata: map[string]interface{}{
			"mission_id":    missionID.String(),
			"mission_title": sm.Title,
			"mission_type":  sm.MissionType,
			"target_id":     sm.TargetID,
			"target_amount": sm.TargetAmount,
			"reward_points": sm.RewardContribution,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	respondActionWithInventory(c, http.StatusOK, player, gin.H{
		"message":             "Hoàn thành nhiệm vụ tông môn! Nhận được " + strconv.FormatInt(sm.RewardContribution, 10) + " Điểm cống hiến.",
		"contribution_points": m.ContributionPoints,
	})
}

// DeleteSectMission lets a sect leader/elder delete a mission posted on the board.
// Accepted missions referencing it are removed via ON DELETE CASCADE.
func DeleteSectMission(c *gin.Context) {
	missionIDStr := c.Param("missionId")
	missionID, err := uuid.Parse(missionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mission ID format"})
		return
	}

	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var sm SectMission
	err = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT id, sect_id, title, reward_contribution
		FROM sect_missions
		WHERE id = $1
	`, missionID).Scan(&sm.ID, &sm.SectID, &sm.Title, &sm.RewardContribution)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy nhiệm vụ yêu cầu!"})
		return
	}

	var m SectMember
	err = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT sect_id, member_rank
		FROM sect_members
		WHERE player_id = $1 AND sect_id = $2
	`, playerID, sm.SectID).Scan(&m.SectID, &m.MemberRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nhiệm vụ này thuộc về tông môn khác!"})
		return
	}

	if !canManageSect(m.MemberRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Trưởng Lão hoặc Chưởng Môn mới có quyền xóa nhiệm vụ!"})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM sect_missions WHERE id = $1`, missionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa nhiệm vụ"})
		return
	}

	// Refund sect funds
	_, err = tx.Exec(`UPDATE sects SET funds = funds + $1 WHERE id = $2`, sm.RewardContribution, sm.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hoàn quỹ bang"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã thu hồi nhiệm vụ tông môn và hoàn lại " + strconv.FormatInt(sm.RewardContribution, 10) + " quỹ bang!"})
}

// AbandonSectMission lets a player give up their own in-progress mission.
func AbandonSectMission(c *gin.Context) {
	missionIDStr := c.Param("missionId")
	missionID, err := uuid.Parse(missionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mission ID format"})
		return
	}

	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	res, err := db.DB.ExecContext(c.Request.Context(), `
		DELETE FROM player_sect_missions
		WHERE player_id = $1 AND mission_id = $2 AND status = 'IN_PROGRESS'
	`, playerID, missionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hủy nhiệm vụ"})
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không có nhiệm vụ đang thực hiện này!"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.abandon_mission", EntityType: "sect_mission",
		EntityID:       missionID.String(),
		RequestSummary: "Hủy nhiệm vụ tông môn đang thực hiện",
		IPAddress:      c.ClientIP(), Success: true,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Đã hủy nhiệm vụ tông môn!"})
}

// CreateCustomMission registers a new mission for the sect
func CreateCustomMission(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CreateCustomMissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check mission type: only GATHER or DONATE supported for simplicity
	if req.MissionType != "GATHER" && req.MissionType != "DONATE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại nhiệm vụ không hợp lệ (chỉ hỗ trợ GATHER hoặc DONATE)"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Fetch caller's sect and rank
	var callerSectID, callerRank string
	err = tx.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&callerSectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if !canManageSect(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Trưởng Lão hoặc Chưởng Môn mới có quyền phân phát nhiệm vụ!"})
		return
	}

	// Check if sect has enough funds
	var currentFunds int64
	err = tx.QueryRow(`SELECT funds FROM sects WHERE id = $1 FOR UPDATE`, callerSectID).Scan(&currentFunds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	missionCost := req.RewardContribution
	if currentFunds < missionCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quỹ bang không đủ để ban bố nhiệm vụ này! (Cần tối thiểu " + strconv.FormatInt(missionCost, 10) + " quỹ bang)"})
		return
	}

	// Deduct sect funds
	_, err = tx.Exec(`UPDATE sects SET funds = funds - $1 WHERE id = $2`, missionCost, callerSectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct sect funds: " + err.Error()})
		return
	}

	// Add mission to sect_missions
	var expiresAt *time.Time
	switch {
	case req.DurationHours > 0:
		t := time.Now().Add(time.Duration(req.DurationHours) * time.Hour)
		expiresAt = &t
	case req.DurationHours < 0:
		expiresAt = nil // no expiry
	default:
		t := time.Now().Add(24 * time.Hour)
		expiresAt = &t
	}

	_, err = tx.Exec(`
		INSERT INTO sect_missions (sect_id, title, description, mission_type, target_id, target_amount, reward_contribution, required_member_rank, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, callerSectID, req.Title, req.Description, req.MissionType, req.TargetID, req.TargetAmount, req.RewardContribution, req.RequiredMemberRank, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create mission record: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã phát hành nhiệm vụ tông môn thành công!"})
}
