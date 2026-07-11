package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/pricing"
)

type Sect struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	Region         string          `json:"region"`
	VaultGu        json.RawMessage `json:"vault_gu"`
	VaultMaterials json.RawMessage `json:"vault_materials"`
	VaultRecipes   json.RawMessage `json:"vault_recipes"`
	Funds          int             `json:"funds"`
	Level          int             `json:"level"`
	RequireApproval bool           `json:"require_approval"`
}

type SectJoinRequest struct {
	ID        uuid.UUID `json:"id"`
	SectID    string    `json:"sect_id"`
	PlayerID  uuid.UUID `json:"player_id"`
	Status    string    `json:"status"` // PENDING, APPROVED, REJECTED
	CreatedAt time.Time `json:"created_at"`
	Username   string `json:"username,omitempty"`
	PlayerRank int    `json:"player_rank,omitempty"`
}

type SectPromotionRequest struct {
	ID          uuid.UUID `json:"id"`
	SectID      string    `json:"sect_id"`
	PlayerID    uuid.UUID `json:"player_id"`
	CurrentRank string    `json:"current_rank"`
	TargetRank  string    `json:"target_rank"`
	Status      string    `json:"status"` // PENDING, APPROVED, REJECTED
	CreatedAt   time.Time `json:"created_at"`
	Username   string `json:"username,omitempty"`
	PlayerRank int    `json:"player_rank,omitempty"`
}

type SectMember struct {
	ID                 uuid.UUID `json:"id"`
	SectID             string    `json:"sect_id"`
	PlayerID           uuid.UUID `json:"player_id"`
	MemberRank         string    `json:"member_rank"` // NGOAI_MON, NOI_MON, TINH_ANH, CHAN_TRUYEN, TRUONG_LAO
	ContributionPoints int       `json:"contribution_points"`
	MasterID           *string   `json:"master_id"`
	JoinedAt           time.Time `json:"joined_at"`
}

type SectMission struct {
	ID                  uuid.UUID `json:"id"`
	SectID              string    `json:"sect_id"`
	Title               string    `json:"title"`
	Description         string    `json:"description"`
	MissionType         string    `json:"mission_type"` // GATHER, COMBAT, REFINE, DONATE
	TargetID            string    `json:"target_id"`
	TargetAmount        int       `json:"target_amount"`
	RewardContribution  int       `json:"reward_contribution"`
	RequiredMemberRank  string    `json:"required_member_rank"`
}

type PlayerSectMission struct {
	ID              uuid.UUID `json:"id"`
	PlayerID        uuid.UUID `json:"player_id"`
	MissionID       uuid.UUID `json:"mission_id"`
	CurrentProgress int       `json:"current_progress"`
	Status          string    `json:"status"` // IN_PROGRESS, COMPLETED, FAILED
	AssignedAt      time.Time `json:"assigned_at"`
}

type SectVaultGuItem struct {
	ID       string `json:"id"`
	Cost     int    `json:"cost"`
	Quantity int    `json:"quantity"`
}

type DonateStonesRequest struct {
	Amount int `json:"amount" binding:"required,min=1"`
}

type DonateMaterialsRequest struct {
	MaterialID string `json:"material_id" binding:"required"`
	Amount     int    `json:"amount" binding:"required,min=1"`
}

type DonateGuRequest struct {
	GuInstanceID string `json:"gu_instance_id" binding:"required"`
}

type VaultBuyRequest struct {
	ItemType string `json:"item_type" binding:"required"` // gu, material, recipe
	ItemID   string `json:"item_id" binding:"required"`
}

// GetSects lists all available sects
func GetSects(c *gin.Context) {
	rows, err := db.DB.Query(`SELECT id, name, description, region, vault_gu, vault_materials, vault_recipes, funds, level, require_approval FROM sects`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sects: " + err.Error()})
		return
	}
	defer rows.Close()

	var sects []Sect
	for rows.Next() {
		var s Sect
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Region, &s.VaultGu, &s.VaultMaterials, &s.VaultRecipes, &s.Funds, &s.Level, &s.RequireApproval); err == nil {
			sects = append(sects, s)
		}
	}

	c.JSON(http.StatusOK, sects)
}

// JoinSect joins a player to a sect
func JoinSect(c *gin.Context) {
	sectID := c.Param("id")
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

	// Check if already in a sect
	var existingSectID string
	err = tx.QueryRow(`SELECT sect_id FROM sect_members WHERE player_id = $1`, playerID).Scan(&existingSectID)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đã gia nhập tông môn \"" + existingSectID + "\" rồi!"})
		return
	}

	// Verify sect exists and get require_approval status
	var requireApproval bool
	err = tx.QueryRow(`SELECT require_approval FROM sects WHERE id = $1`, sectID).Scan(&requireApproval)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tông môn này!"})
		return
	}

	if requireApproval {
		// Check for existing pending request
		var pendingID string
		err = tx.QueryRow(`SELECT id FROM sect_join_requests WHERE player_id = $1 AND sect_id = $2 AND status = 'PENDING'`, playerID, sectID).Scan(&pendingID)
		if err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Yêu cầu gia nhập tông môn này của đạo hữu đang chờ duyệt!"})
			return
		}

		// Insert or update to pending request
		_, err = tx.Exec(`
			INSERT INTO sect_join_requests (sect_id, player_id, status)
			VALUES ($1, $2, 'PENDING')
			ON CONFLICT (sect_id, player_id) DO UPDATE SET status = 'PENDING', created_at = now()
		`, sectID, playerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit join request: " + err.Error()})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Gửi yêu cầu gia nhập thành công, vui lòng chờ duyệt!"})
		return
	}

	// Create member record (Immediate Join)
	_, err = tx.Exec(`
		INSERT INTO sect_members (sect_id, player_id, member_rank, contribution_points)
		VALUES ($1, $2, 'NGOAI_MON', 0)
	`, sectID, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to join sect: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Gia nhập tông môn thành công!"})
}

// LeaveSect removes a player from their sect
func LeaveSect(c *gin.Context) {
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

	// Delete membership
	_, err = tx.Exec(`DELETE FROM sect_members WHERE player_id = $1`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to leave sect"})
		return
	}

	// Delete active player missions
	_, _ = tx.Exec(`DELETE FROM player_sect_missions WHERE player_id = $1`, playerID)

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã rời khỏi tông môn, tổn thất toàn bộ điểm cống hiến."})
}

// GetMySectStatus gets current player's sect status
func GetMySectStatus(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var m SectMember
	err := db.DB.QueryRow(`
		SELECT id, sect_id, player_id, member_rank, contribution_points, master_id, joined_at
		FROM sect_members
		WHERE player_id = $1
	`, playerID).Scan(&m.ID, &m.SectID, &m.PlayerID, &m.MemberRank, &m.ContributionPoints, &m.MasterID, &m.JoinedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			// Check if they have a pending join request
			var pendingSectID, pendingSectName string
			err = db.DB.QueryRow(`
				SELECT r.sect_id, s.name FROM sect_join_requests r
				JOIN sects s ON r.sect_id = s.id
				WHERE r.player_id = $1 AND r.status = 'PENDING'
				LIMIT 1
			`, playerID).Scan(&pendingSectID, &pendingSectName)
			
			hasPendingJoin := false
			if err == nil {
				hasPendingJoin = true
			}
			c.JSON(http.StatusOK, gin.H{
				"in_sect":           false,
				"has_pending_join":  hasPendingJoin,
				"pending_sect_id":   pendingSectID,
				"pending_sect_name": pendingSectName,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		}
		return
	}

	var s Sect
	err = db.DB.QueryRow(`
		SELECT id, name, description, region, vault_gu, vault_materials, vault_recipes, funds, level, require_approval
		FROM sects
		WHERE id = $1
	`, m.SectID).Scan(&s.ID, &s.Name, &s.Description, &s.Region, &s.VaultGu, &s.VaultMaterials, &s.VaultRecipes, &s.Funds, &s.Level, &s.RequireApproval)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sect details"})
		return
	}

	// Active mission if any
	var activeMission SectMission
	var activeProgress int
	var activePMId uuid.UUID
	hasActive := false
	err = db.DB.QueryRow(`
		SELECT pm.id, sm.id, sm.title, sm.description, sm.mission_type, sm.target_id, sm.target_amount, sm.reward_contribution, pm.current_progress
		FROM player_sect_missions pm
		JOIN sect_missions sm ON pm.mission_id = sm.id
		WHERE pm.player_id = $1 AND pm.status = 'IN_PROGRESS'
		LIMIT 1
	`, playerID).Scan(&activePMId, &activeMission.ID, &activeMission.Title, &activeMission.Description, &activeMission.MissionType, &activeMission.TargetID, &activeMission.TargetAmount, &activeMission.RewardContribution, &activeProgress)
	if err == nil {
		hasActive = true
	}

	// Leaderboard inside the sect
	rows, err := db.DB.Query(`
		SELECT p.username, sm.member_rank, sm.contribution_points
		FROM sect_members sm
		JOIN players p ON sm.player_id = p.id
		WHERE sm.sect_id = $1
		ORDER BY sm.contribution_points DESC
		LIMIT 10
	`, m.SectID)
	
	type LeaderboardEntry struct {
		Username     string `json:"username"`
		Rank         string `json:"rank"`
		Contribution int    `json:"contribution"`
	}
	var leaderboard []LeaderboardEntry
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var entry LeaderboardEntry
			if err := rows.Scan(&entry.Username, &entry.Rank, &entry.Contribution); err == nil {
				leaderboard = append(leaderboard, entry)
			}
		}
	}

	// Check if this player has a pending promotion request
	var hasPendingPromotion bool
	var pendingPromoTarget string
	err = db.DB.QueryRow(`
		SELECT target_rank FROM sect_promotion_requests
		WHERE player_id = $1 AND status = 'PENDING'
	`, playerID).Scan(&pendingPromoTarget)
	if err == nil {
		hasPendingPromotion = true
	}

	c.JSON(http.StatusOK, gin.H{
		"in_sect":                  true,
		"member":                   m,
		"sect":                     s,
		"has_active":               hasActive,
		"active_mission":           activeMission,
		"active_progress":          activeProgress,
		"active_pm_id":             activePMId,
		"leaderboard":              leaderboard,
		"has_pending_promotion":    hasPendingPromotion,
		"pending_promotion_target": pendingPromoTarget,
	})
}

// PromoteSectRank upgrades player's rank in sect
func PromoteSectRank(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
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

	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}

	var m SectMember
	err = tx.QueryRow(`
		SELECT id, sect_id, player_id, member_rank, contribution_points
		FROM sect_members
		WHERE player_id = $1
	`, playerID).Scan(&m.ID, &m.SectID, &m.PlayerID, &m.MemberRank, &m.ContributionPoints)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	var nextRank string
	var reqRank int
	var costPoints int

	switch m.MemberRank {
	case "NGOAI_MON":
		nextRank = "NOI_MON"
		reqRank = 2
		costPoints = 100
	case "NOI_MON":
		nextRank = "TINH_ANH"
		reqRank = 3
		costPoints = 500
	case "TINH_ANH":
		nextRank = "CHAN_TRUYEN"
		reqRank = 4
		costPoints = 2000
	case "CHAN_TRUYEN":
		nextRank = "TRUONG_LAO"
		reqRank = 5
		costPoints = 10000
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cảnh giới tông môn đã đạt đỉnh phong (Trưởng Lão)!"})
		return
	}

	if player.Rank < reqRank {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tu vi thực tế của đạo hữu chưa đủ. Yêu cầu Rank " + string(rune('0'+reqRank)) + "!"})
		return
	}

	if m.ContributionPoints < costPoints {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không đủ Điểm cống hiến để tham gia khảo hạch thăng cấp!"})
		return
	}

	m.ContributionPoints -= costPoints

	// Manual check for high tier ranks: CHAN_TRUYEN and TRUONG_LAO
	if nextRank == "CHAN_TRUYEN" || nextRank == "TRUONG_LAO" {
		var leaderCount int
		err = tx.QueryRow(`SELECT COUNT(1) FROM sect_members WHERE sect_id = $1 AND member_rank = 'CHUONG_MON'`, m.SectID).Scan(&leaderCount)
		if err == nil && leaderCount > 0 {
			// Check for pending request
			var pendingID string
			err = tx.QueryRow(`SELECT id FROM sect_promotion_requests WHERE player_id = $1 AND sect_id = $2 AND status = 'PENDING'`, playerID, m.SectID).Scan(&pendingID)
			if err == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Yêu cầu thăng cấp lên " + nextRank + " của đạo hữu đang chờ duyệt!"})
				return
			}

			// Deduct points from database
			_, err = tx.Exec(`UPDATE sect_members SET contribution_points = $1 WHERE player_id = $2`, m.ContributionPoints, playerID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update contribution points: " + err.Error()})
				return
			}

			// Insert promotion request
			_, err = tx.Exec(`
				INSERT INTO sect_promotion_requests (sect_id, player_id, current_rank, target_rank, status)
				VALUES ($1, $2, $3, $4, 'PENDING')
				ON CONFLICT (sect_id, player_id) DO UPDATE SET current_rank = $3, target_rank = $4, status = 'PENDING', created_at = now()
			`, m.SectID, playerID, m.MemberRank, nextRank)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit promotion request: " + err.Error()})
				return
			}

			if err := tx.Commit(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "Gửi yêu cầu thăng cấp lên " + nextRank + " thành công, vui lòng chờ duyệt!",
				"member":  m,
			})
			return
		}
	}

	m.MemberRank = nextRank

	_, err = tx.Exec(`
		UPDATE sect_members
		SET member_rank = $1, contribution_points = $2
		WHERE player_id = $3
	`, m.MemberRank, m.ContributionPoints, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update member rank"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Vượt qua khảo hạch thành công! Đạo hữu thăng lên " + nextRank,
		"member":  m,
	})
}

// DonateStones converts player primeval stones to contribution points
func DonateStones(c *gin.Context) {
	var req DonateStonesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	if player.PrimevalStones < req.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không đủ Nguyên Thạch!"})
		return
	}

	var m SectMember
	err = tx.QueryRow(`
		SELECT id, sect_id, contribution_points
		FROM sect_members
		WHERE player_id = $1
	`, playerID).Scan(&m.ID, &m.SectID, &m.ContributionPoints)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	player.PrimevalStones -= req.Amount
	m.ContributionPoints += req.Amount // 1:1 ratio

	// Increase sect funds
	_, err = tx.Exec(`UPDATE sects SET funds = funds + $1 WHERE id = $2`, req.Amount, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect funds"})
		return
	}

	_, err = tx.Exec(`UPDATE players SET primeval_stones = $1 WHERE id = $2`, player.PrimevalStones, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct stones"})
		return
	}

	_, err = tx.Exec(`UPDATE sect_members SET contribution_points = $1 WHERE player_id = $2`, m.ContributionPoints, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to award contribution"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.donate_stones", EntityType: "sect",
		EntityID: m.SectID, RequestSummary: fmt.Sprintf("%d Nguyên Thạch → %d điểm cống hiến", req.Amount, req.Amount),
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{
		"message":             "Quyên góp thành công!",
		"primeval_stones":     player.PrimevalStones,
		"contribution_points": m.ContributionPoints,
	})
}

// DonateMaterials converts player materials to contribution points
func DonateMaterials(c *gin.Context) {
	var req DonateMaterialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	qty, ok := player.MaterialsInventory[req.MaterialID]
	if !ok || qty < req.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không có đủ linh tài yêu cầu!"})
		return
	}

	var m SectMember
	err = tx.QueryRow(`
		SELECT id, sect_id, contribution_points
		FROM sect_members
		WHERE player_id = $1
	`, playerID).Scan(&m.ID, &m.SectID, &m.ContributionPoints)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	// Calculate value dynamically from item_dictionary.base_price + rank (via pricing package)
	var dbBasePrice, matRank int
	if err = tx.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM item_dictionary WHERE id=$1`, req.MaterialID).Scan(&dbBasePrice, &matRank); err != nil {
		matRank = 1
		dbBasePrice = 0
	}
	pointsPerItem := pricing.DonatePoints(pricing.ItemBasePrice(dbBasePrice, matRank))

	gainedContribution := pointsPerItem * req.Amount
	player.MaterialsInventory[req.MaterialID] -= req.Amount
	if player.MaterialsInventory[req.MaterialID] == 0 {
		delete(player.MaterialsInventory, req.MaterialID)
	}

	m.ContributionPoints += gainedContribution

	// Increase sect funds
	_, err = tx.Exec(`UPDATE sects SET funds = funds + $1 WHERE id = $2`, gainedContribution, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect funds"})
		return
	}

	// Add to sect vault
	var vaultMaterialsRaw json.RawMessage
	err = tx.QueryRow(`SELECT vault_materials FROM sects WHERE id = $1`, m.SectID).Scan(&vaultMaterialsRaw)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sect vault"})
		return
	}
	
	vaultMats := make(map[string]int)
	if len(vaultMaterialsRaw) > 0 {
		_ = json.Unmarshal(vaultMaterialsRaw, &vaultMats)
	}
	
	vaultMats[req.MaterialID] += req.Amount
	newVaultMats, _ := json.Marshal(vaultMats)
	_, err = tx.Exec(`UPDATE sects SET vault_materials = $1 WHERE id = $2`, newVaultMats, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault materials"})
		return
	}

	_, err = tx.Exec(`UPDATE players SET materials_inventory = $1 WHERE id = $2`, player.MaterialsInventory, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update materials"})
		return
	}

	_, err = tx.Exec(`UPDATE sect_members SET contribution_points = $1 WHERE player_id = $2`, m.ContributionPoints, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update contribution"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	materialName := req.MaterialID
	_ = db.DB.QueryRow(`SELECT name FROM item_dictionary WHERE id = $1`, req.MaterialID).Scan(&materialName)
	if materialName == "" {
		materialName = req.MaterialID
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.donate_materials", EntityType: "sect",
		EntityID: m.SectID, RequestSummary: fmt.Sprintf("%s x%d → %d điểm", materialName, req.Amount, gainedContribution),
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{
		"message":             "Quyên góp linh tài thành công!",
		"materials_inventory": player.MaterialsInventory,
		"contribution_points": m.ContributionPoints,
	})
}

// DonateGu converts player Gu bug to contribution points
func DonateGu(c *gin.Context) {
	var req DonateGuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	targetIdx := -1
	for i, gi := range player.GuInventory {
		if gi.InstanceID == req.GuInstanceID {
			targetIdx = i
			break
		}
	}

	if targetIdx == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cổ trùng không tồn tại trong túi đồ!"})
		return
	}

	guInstance := player.GuInventory[targetIdx]
	if guInstance.IsVital {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể quyên góp Bản Mệnh Cổ!"})
		return
	}
	if guInstance.Equipped {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vui lòng tháo Cổ trùng ra trước khi quyên góp!"})
		return
	}

	var m SectMember
	err = tx.QueryRow(`
		SELECT id, sect_id, contribution_points
		FROM sect_members
		WHERE player_id = $1
	`, playerID).Scan(&m.ID, &m.SectID, &m.ContributionPoints)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	// Fetch Gu base_price + rank for contribution calculation
	var guRank, guBasePrice int
	err = tx.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM gu_dictionary WHERE id = $1`, guInstance.GuID).Scan(&guBasePrice, &guRank)
	if err != nil {
		guRank = 1
		guBasePrice = 0
	}

	effectiveBase := pricing.GuBasePrice(guBasePrice, guRank)
	gainedContribution := pricing.DonatePoints(effectiveBase)
	vaultItemCost := pricing.VaultCost(effectiveBase)

	// Add to sect vault
	var vaultGuRaw json.RawMessage
	err = tx.QueryRow(`SELECT vault_gu FROM sects WHERE id = $1`, m.SectID).Scan(&vaultGuRaw)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sect vault"})
		return
	}

	var vaultGus []SectVaultGuItem
	if len(vaultGuRaw) > 0 {
		_ = json.Unmarshal(vaultGuRaw, &vaultGus)
	}

	// Increment quantity if Gu type already exists in vault, otherwise add new entry
	donatedGuFound := false
	for i, v := range vaultGus {
		if v.ID == guInstance.GuID {
			vaultGus[i].Quantity++
			donatedGuFound = true
			break
		}
	}
	if !donatedGuFound {
		vaultGus = append(vaultGus, SectVaultGuItem{
			ID:       guInstance.GuID,
			Cost:     vaultItemCost,
			Quantity: 1,
		})
	}

	newVaultGus, _ := json.Marshal(vaultGus)
	_, err = tx.Exec(`UPDATE sects SET vault_gu = $1 WHERE id = $2`, newVaultGus, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault Gu list"})
		return
	}

	// Remove Gu
	player.GuInventory = append(player.GuInventory[:targetIdx], player.GuInventory[targetIdx+1:]...)
	m.ContributionPoints += gainedContribution

	// Increase sect funds
	_, err = tx.Exec(`UPDATE sects SET funds = funds + $1 WHERE id = $2`, gainedContribution, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect funds"})
		return
	}

	_, err = tx.Exec(`UPDATE players SET gu_inventory = $1 WHERE id = $2`, player.GuInventory, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update gu inventory"})
		return
	}

	_, err = tx.Exec(`UPDATE sect_members SET contribution_points = $1 WHERE player_id = $2`, m.ContributionPoints, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update contribution"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	guName := guInstance.GuID
	if spec, ok := models.GuRegistry[guInstance.GuID]; ok {
		guName = spec.Name
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.donate_gu", EntityType: "sect",
		EntityID: m.SectID, RequestSummary: fmt.Sprintf("%s → %d điểm", guName, gainedContribution),
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{
		"message":             "Quyên góp Cổ trùng thành công!",
		"gu_inventory":        player.GuInventory,
		"contribution_points": m.ContributionPoints,
	})
}

// BuyFromSectVault processes vault purchases
func BuyFromSectVault(c *gin.Context) {
	var req VaultBuyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	playerID, exists := middleware.GetPlayerID(c)
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

	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}

	var m SectMember
	err = tx.QueryRow(`
		SELECT id, sect_id, contribution_points
		FROM sect_members
		WHERE player_id = $1
	`, playerID).Scan(&m.ID, &m.SectID, &m.ContributionPoints)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	var s Sect
	err = tx.QueryRow(`
		SELECT id, vault_gu, vault_materials, vault_recipes
		FROM sects
		WHERE id = $1
	`, m.SectID).Scan(&s.ID, &s.VaultGu, &s.VaultMaterials, &s.VaultRecipes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sect details"})
		return
	}

	var cost int
	if strings.ToLower(req.ItemType) == "gu" {
		var vaultGus []SectVaultGuItem
		if err := json.Unmarshal(s.VaultGu, &vaultGus); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse vault Gu list"})
			return
		}

		targetIdx := -1
		for idx, item := range vaultGus {
			if item.ID == req.ItemID {
				targetIdx = idx
				break
			}
		}

		if targetIdx == -1 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Cổ trùng này không có sẵn trong Bảo khố!"})
			return
		}

		if vaultGus[targetIdx].Quantity <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng này đã hết hàng trong Bảo khố!"})
			return
		}

		cost = vaultGus[targetIdx].Cost
		if m.ContributionPoints < cost {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không đủ Điểm cống hiến!"})
			return
		}

		m.ContributionPoints -= cost
		vaultGus[targetIdx].Quantity--

		newGu := models.GuInstance{
			InstanceID: uuid.New().String(),
			GuID:       req.ItemID,
			Hunger:     100,
			Status:     "ACTIVE",
			Equipped:   false,
		}
		player.GuInventory = append(player.GuInventory, newGu)

		_, err = tx.Exec(`UPDATE players SET gu_inventory = $1 WHERE id = $2`, player.GuInventory, player.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add Gu bug"})
			return
		}

		newVaultGu, err := json.Marshal(vaultGus)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize updated Gu list"})
			return
		}
		_, err = tx.Exec(`UPDATE sects SET vault_gu = $1 WHERE id = $2`, newVaultGu, s.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault Gu list"})
			return
		}

	} else if strings.ToLower(req.ItemType) == "material" {
		var vaultMats map[string]int
		if err := json.Unmarshal(s.VaultMaterials, &vaultMats); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse vault materials"})
			return
		}

		qty, hasMat := vaultMats[req.ItemID]
		if !hasMat || qty <= 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Linh tài này đã hết hàng trong Bảo khố!"})
			return
		}

		// Dynamic cost from item_dictionary base_price + rank via pricing package
		{
			var matBasePrice, matRank int
			_ = tx.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM item_dictionary WHERE id=$1`, req.ItemID).Scan(&matBasePrice, &matRank)
			cost = pricing.VaultCost(pricing.ItemBasePrice(matBasePrice, matRank))
		}

		if m.ContributionPoints < cost {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không đủ Điểm cống hiến!"})
			return
		}

		m.ContributionPoints -= cost
		vaultMats[req.ItemID]--
		if vaultMats[req.ItemID] == 0 {
			delete(vaultMats, req.ItemID)
		}

		if player.MaterialsInventory == nil {
			player.MaterialsInventory = make(models.MaterialsInventory)
		}
		player.MaterialsInventory[req.ItemID]++

		newVaultMats, _ := json.Marshal(vaultMats)
		_, err = tx.Exec(`UPDATE sects SET vault_materials = $1 WHERE id = $2`, newVaultMats, s.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update vault materials"})
			return
		}

		_, err = tx.Exec(`UPDATE players SET materials_inventory = $1 WHERE id = $2`, player.MaterialsInventory, player.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update materials"})
			return
		}
	} else if strings.ToLower(req.ItemType) == "recipe" {
		var vaultRecipes []string
		if err := json.Unmarshal(s.VaultRecipes, &vaultRecipes); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse recipes list"})
			return
		}

		hasRecipe := false
		for _, recipeId := range vaultRecipes {
			if recipeId == req.ItemID {
				hasRecipe = true
				break
			}
		}

		if !hasRecipe {
			c.JSON(http.StatusNotFound, gin.H{"error": "Công thức này không có sẵn trong Bảo khố!"})
			return
		}

		// Dynamic recipe cost based on target Gu rank via pricing package
		{
			var targetGuRank int
			_ = tx.QueryRow(`SELECT COALESCE(gd.rank,1) FROM refining_recipes rr JOIN gu_dictionary gd ON gd.id=rr.target_gu_id WHERE rr.id=$1`, req.ItemID).Scan(&targetGuRank)
			cost = pricing.RecipeCostByGuRank(targetGuRank)
		}
		if m.ContributionPoints < cost {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không đủ Điểm cống hiến!"})
			return
		}

		m.ContributionPoints -= cost
		// Since recipe unlocking is currently free-form for crafting recipes catalog,
		// we just deduct the points and tell the player they learned it.
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại vật phẩm trao đổi không hợp lệ!"})
		return
	}

	_, err = tx.Exec(`UPDATE sect_members SET contribution_points = $1 WHERE player_id = $2`, m.ContributionPoints, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update contribution"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	itemName := req.ItemID
	if req.ItemType == "Gu" {
		if spec, ok := models.GuRegistry[req.ItemID]; ok {
			itemName = spec.Name
		}
	} else {
		_ = db.DB.QueryRow(`SELECT name FROM item_dictionary WHERE id = $1`, req.ItemID).Scan(&itemName)
	}
	if itemName == "" {
		itemName = req.ItemID
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.buy_vault", EntityType: "sect",
		EntityID: m.SectID,
		RequestSummary: fmt.Sprintf("%s | %s | giá %d điểm cống hiến", req.ItemType, itemName, cost),
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{
		"message": "Đổi vật phẩm thành công!",
		"player":  player,
		"member":  m,
	})
}

// GetSectMissions returns open missions
func GetSectMissions(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var m SectMember
	err := db.DB.QueryRow(`
		SELECT sect_id, member_rank
		FROM sect_members
		WHERE player_id = $1
	`, playerID).Scan(&m.SectID, &m.MemberRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	rows, err := db.DB.Query(`
		SELECT id, sect_id, title, description, mission_type, target_id, target_amount, reward_contribution, required_member_rank
		FROM sect_missions
		WHERE sect_id = $1
	`, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch missions"})
		return
	}
	defer rows.Close()

	var missions []SectMission
	for rows.Next() {
		var sm SectMission
		if err := rows.Scan(&sm.ID, &sm.SectID, &sm.Title, &sm.Description, &sm.MissionType, &sm.TargetID, &sm.TargetAmount, &sm.RewardContribution, &sm.RequiredMemberRank); err == nil {
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
		LIMIT 1
	`, playerID).Scan(&dummy)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đang thực hiện một nhiệm vụ khác!"})
		return
	}

	// Fetch mission details
	var sm SectMission
	err = tx.QueryRow(`
		SELECT id, sect_id, required_member_rank
		FROM sect_missions
		WHERE id = $1
	`, missionID).Scan(&sm.ID, &sm.SectID, &sm.RequiredMemberRank)
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
	}

	if rankMap[m.MemberRank] < rankMap[sm.RequiredMemberRank] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cấp bậc đệ tử của đạo hữu không đủ để nhận nhiệm vụ này!"})
		return
	}

	// Insert player mission
	_, err = tx.Exec(`
		INSERT INTO player_sect_missions (player_id, mission_id, current_progress, status)
		VALUES ($1, $2, 0, 'IN_PROGRESS')
	`, playerID, sm.ID)
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
		qty, hasMat := player.MaterialsInventory[sm.TargetID]
		if !hasMat || qty < sm.TargetAmount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa thu thập đủ linh tài yêu cầu!"})
			return
		}
		// Deduct materials from player
		player.MaterialsInventory[sm.TargetID] -= sm.TargetAmount
		if player.MaterialsInventory[sm.TargetID] == 0 {
			delete(player.MaterialsInventory, sm.TargetID)
		}
		_, err = tx.Exec(`UPDATE players SET materials_inventory = $1 WHERE id = $2`, player.MaterialsInventory, player.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct materials"})
			return
		}

		// Add gathered materials to sect vault
		var memberForVault SectMember
		err = tx.QueryRow(`SELECT sect_id FROM sect_members WHERE player_id = $1`, playerID).Scan(&memberForVault.SectID)
		if err == nil {
			var vaultMaterialsRaw json.RawMessage
			err = tx.QueryRow(`SELECT vault_materials FROM sects WHERE id = $1`, memberForVault.SectID).Scan(&vaultMaterialsRaw)
			if err == nil {
				vaultMats := make(map[string]int)
				if len(vaultMaterialsRaw) > 0 {
					_ = json.Unmarshal(vaultMaterialsRaw, &vaultMats)
				}
				vaultMats[sm.TargetID] += sm.TargetAmount
				newVaultMats, _ := json.Marshal(vaultMats)
				_, _ = tx.Exec(`UPDATE sects SET vault_materials = $1 WHERE id = $2`, newVaultMats, memberForVault.SectID)
			}
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
		player.GuInventory = append(player.GuInventory[:targetIdx], player.GuInventory[targetIdx+1:]...)
		_, err = tx.Exec(`UPDATE players SET gu_inventory = $1 WHERE id = $2`, player.GuInventory, player.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update gu inventory"})
			return
		}

		// Add donated Gu to sect vault
		var memberForVault SectMember
		err = tx.QueryRow(`SELECT sect_id FROM sect_members WHERE player_id = $1`, playerID).Scan(&memberForVault.SectID)
		if err == nil {
			var vaultGuRaw json.RawMessage
			err = tx.QueryRow(`SELECT vault_gu FROM sects WHERE id = $1`, memberForVault.SectID).Scan(&vaultGuRaw)
			if err == nil {
				var vaultGus []SectVaultGuItem
				if len(vaultGuRaw) > 0 {
					_ = json.Unmarshal(vaultGuRaw, &vaultGus)
				}
				// Look up Gu base_price + rank for vault cost calculation
				var donatedGuRank, donatedGuBase int
				_ = tx.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM gu_dictionary WHERE id = $1`, donatedGuID).Scan(&donatedGuBase, &donatedGuRank)
				if donatedGuRank == 0 {
					donatedGuRank = 1
				}
				missionGuFound := false
				for i, v := range vaultGus {
					if v.ID == donatedGuID {
						vaultGus[i].Quantity++
						missionGuFound = true
						break
					}
				}
				if !missionGuFound {
					vaultGus = append(vaultGus, SectVaultGuItem{
						ID:       donatedGuID,
						Cost:     pricing.VaultCost(pricing.GuBasePrice(donatedGuBase, donatedGuRank)),
						Quantity: 1,
					})
				}
				newVaultGus, _ := json.Marshal(vaultGus)
				_, _ = tx.Exec(`UPDATE sects SET vault_gu = $1 WHERE id = $2`, newVaultGus, memberForVault.SectID)
			}
		}
	}

	// Update mission status
	_, err = tx.Exec(`UPDATE player_sect_missions SET status = 'COMPLETED' WHERE id = $1`, psm.ID)
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

	m.ContributionPoints += sm.RewardContribution
	_, err = tx.Exec(`UPDATE sect_members SET contribution_points = $1 WHERE player_id = $2`, m.ContributionPoints, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add contribution points"})
		return
	}

	// Increase sect funds
	_, err = tx.Exec(`UPDATE sects SET funds = funds + $1 WHERE id = $2`, sm.RewardContribution, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect funds"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":             "Hoàn thành nhiệm vụ tông môn! Nhận được " + string(rune('0'+sm.RewardContribution)) + " Điểm cống hiến.",
		"player":              player,
		"contribution_points": m.ContributionPoints,
	})
}

type CreateSectRequest struct {
	Name        string `json:"name" binding:"required,min=3,max=50"`
	Description string `json:"description" binding:"required,max=500"`
	Region      string `json:"region" binding:"required"`
}

type SetMemberRankRequest struct {
	Rank string `json:"rank" binding:"required"`
}

type CreateCustomMissionRequest struct {
	Title              string `json:"title" binding:"required"`
	Description        string `json:"description" binding:"required"`
	MissionType        string `json:"mission_type" binding:"required"` // GATHER, DONATE
	TargetID           string `json:"target_id" binding:"required"`
	TargetAmount       int    `json:"target_amount" binding:"required,min=1"`
	RewardContribution int    `json:"reward_contribution" binding:"required,min=1"`
	RequiredMemberRank string `json:"required_member_rank" binding:"required"`
}

func slugify(s string) string {
	var res []rune
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			res = append(res, r)
		} else if r == ' ' || r == '-' || r == '_' {
			if len(res) > 0 && res[len(res)-1] != '_' {
				res = append(res, '_')
			}
		}
	}
	return strings.Trim(string(res), "_")
}

// CreateSect founds a new sect
func CreateSect(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CreateSectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player"})
		return
	}

	// Requirements: Rank >= 5 and 50,000 Primeval Stones
	if player.Rank < 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Khai tông lập phái yêu cầu tu vi ít nhất Ngũ Chuyển (Rank 5)!"})
		return
	}

	cost := 50000
	if player.PrimevalStones < cost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không có đủ 50,000 Nguyên Thạch làm chi phí lập phái!"})
		return
	}

	// Check if already in a sect
	var existingSectID string
	err = tx.QueryRow(`SELECT sect_id FROM sect_members WHERE player_id = $1`, playerID).Scan(&existingSectID)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đang thuộc tông môn \"" + existingSectID + "\" rồi, hãy phản môn trước!"})
		return
	}

	sectID := slugify(req.Name)
	if len(sectID) < 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tên tông môn không hợp lệ!"})
		return
	}

	// Verify uniqueness of name
	var dummy string
	err = tx.QueryRow(`SELECT id FROM sects WHERE id = $1`, sectID).Scan(&dummy)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tông môn tên \"" + req.Name + "\" đã tồn tại trên thế giới!"})
		return
	}

	// Deduct resources
	player.PrimevalStones -= cost
	_, err = tx.Exec(`UPDATE players SET primeval_stones = $1 WHERE id = $2`, player.PrimevalStones, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Deduct primeval stones failed"})
		return
	}

	// Insert new sect
	_, err = tx.Exec(`
		INSERT INTO sects (id, name, description, region, vault_gu, vault_materials, vault_recipes)
		VALUES ($1, $2, $3, $4, '[]'::jsonb, '{}'::jsonb, '[]'::jsonb)
	`, sectID, req.Name, req.Description, req.Region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create sect record: " + err.Error()})
		return
	}

	// Insert founder member record as CHUONG_MON
	_, err = tx.Exec(`
		INSERT INTO sect_members (sect_id, player_id, member_rank, contribution_points)
		VALUES ($1, $2, 'CHUONG_MON', 10000)
	`, sectID, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register founder record: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit transaction failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Khai tông lập phái thành công! Chào mừng Chưởng môn của \"" + req.Name + "\"!",
		"player":  player,
	})
}

// GetSectMembers lists all members of the player's sect
func GetSectMembers(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var sectID string
	err := db.DB.QueryRow(`SELECT sect_id FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	rows, err := db.DB.Query(`
		SELECT sm.player_id, p.username, sm.member_rank, sm.contribution_points, sm.joined_at, p.rank
		FROM sect_members sm
		JOIN players p ON sm.player_id = p.id
		WHERE sm.sect_id = $1
		ORDER BY 
			CASE sm.member_rank
				WHEN 'CHUONG_MON' THEN 1
				WHEN 'TRUONG_LAO' THEN 2
				WHEN 'CHAN_TRUYEN' THEN 3
				WHEN 'TINH_ANH' THEN 4
				WHEN 'NOI_MON' THEN 5
				ELSE 6
			END, sm.contribution_points DESC
	`, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sect members: " + err.Error()})
		return
	}
	defer rows.Close()

	type MemberInfo struct {
		PlayerID     uuid.UUID `json:"player_id"`
		Username     string    `json:"username"`
		MemberRank   string    `json:"member_rank"`
		Contribution int       `json:"contribution_points"`
		JoinedAt     time.Time `json:"joined_at"`
		PlayerRank   int       `json:"player_rank"`
	}

	var members []MemberInfo
	for rows.Next() {
		var mi MemberInfo
		if err := rows.Scan(&mi.PlayerID, &mi.Username, &mi.MemberRank, &mi.Contribution, &mi.JoinedAt, &mi.PlayerRank); err == nil {
			members = append(members, mi)
		}
	}

	c.JSON(http.StatusOK, members)
}

// SetMemberRank adjusts a member's rank in the sect
func SetMemberRank(c *gin.Context) {
	targetMemberIDStr := c.Param("memberId")
	targetMemberID, err := uuid.Parse(targetMemberIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member ID format"})
		return
	}

	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req SetMemberRankRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate target rank
	validRanks := map[string]bool{
		"NGOAI_MON":   true,
		"NOI_MON":     true,
		"TINH_ANH":    true,
		"CHAN_TRUYEN": true,
		"TRUONG_LAO":  true,
		"CHUONG_MON":  true,
	}
	if !validRanks[req.Rank] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rank không hợp lệ"})
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

	if callerRank != "CHUONG_MON" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Chưởng môn mới có quyền định đoạt chức vụ!"})
		return
	}

	// Verify target member is in the same sect
	var targetSectID, targetRank string
	err = tx.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, targetMemberID).Scan(&targetSectID, &targetRank)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy đệ tử này trong môn phái!"})
		return
	}

	if targetSectID != callerSectID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đệ tử này thuộc tông môn khác!"})
		return
	}

	if targetMemberID == playerID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chưởng môn không thể tự giáng chức chính mình! Vui lòng nhường ngôi trước."})
		return
	}

	// Update rank
	_, err = tx.Exec(`UPDATE sect_members SET member_rank = $1 WHERE player_id = $2`, req.Rank, targetMemberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update member rank"})
		return
	}

	// If transferring Chưởng môn, demote caller to Trưởng lão
	if req.Rank == "CHUONG_MON" {
		_, err = tx.Exec(`UPDATE sect_members SET member_rank = 'TRUONG_LAO' WHERE player_id = $1`, playerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to step down from Chưởng môn"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã điều chỉnh chức vụ thành công!"})
}

// KickMember expels a member from the sect
func KickMember(c *gin.Context) {
	targetMemberIDStr := c.Param("memberId")
	targetMemberID, err := uuid.Parse(targetMemberIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member ID format"})
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

	// Fetch caller's sect and rank
	var callerSectID, callerRank string
	err = tx.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&callerSectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if callerRank != "CHUONG_MON" && callerRank != "TRUONG_LAO" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu không đủ quyền hạn (cần Trưởng Lão hoặc Chưởng môn)!"})
		return
	}

	// Verify target member is in the same sect
	var targetSectID, targetRank string
	err = tx.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, targetMemberID).Scan(&targetSectID, &targetRank)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy thành viên này!"})
		return
	}

	if targetSectID != callerSectID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thành viên này thuộc tông môn khác!"})
		return
	}

	if targetMemberID == playerID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể tự trục xuất chính mình!"})
		return
	}

	// Hierarchical check: Elders cannot kick other elders or leaders
	if callerRank == "TRUONG_LAO" && (targetRank == "CHUONG_MON" || targetRank == "TRUONG_LAO") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Trưởng lão không thể trục xuất Trưởng lão khác hoặc Chưởng môn!"})
		return
	}

	// Delete member
	_, err = tx.Exec(`DELETE FROM sect_members WHERE player_id = $1`, targetMemberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to expel member"})
		return
	}

	// Delete target player's active missions
	_, _ = tx.Exec(`DELETE FROM player_sect_missions WHERE player_id = $1`, targetMemberID)

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã trục xuất thành viên ra khỏi tông môn!"})
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

	if callerRank != "CHUONG_MON" && callerRank != "TRUONG_LAO" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Trưởng Lão hoặc Chưởng Môn mới có quyền phân phát nhiệm vụ!"})
		return
	}

	// Add mission to sect_missions
	_, err = tx.Exec(`
		INSERT INTO sect_missions (sect_id, title, description, mission_type, target_id, target_amount, reward_contribution, required_member_rank)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, callerSectID, req.Title, req.Description, req.MissionType, req.TargetID, req.TargetAmount, req.RewardContribution, req.RequiredMemberRank)
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

type StockVaultRequest struct {
	ItemType string `json:"item_type" binding:"required"` // gu, material
	ItemID   string `json:"item_id" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,min=1"`
}

// StockSectVault allows leaders/elders to buy vault items using sect funds
func StockSectVault(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req StockVaultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	// Fetch caller's sect and rank
	var sectID, callerRank string
	err = tx.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if callerRank != "CHUONG_MON" && callerRank != "TRUONG_LAO" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Trưởng Lão hoặc Chưởng Môn mới có quyền mua thêm vật phẩm cho bảo khố!"})
		return
	}

	// Fetch sect details
	var s Sect
	err = tx.QueryRow(`
		SELECT id, vault_gu, vault_materials, funds, level
		FROM sects
		WHERE id = $1
	`, sectID).Scan(&s.ID, &s.VaultGu, &s.VaultMaterials, &s.Funds, &s.Level)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sect details: " + err.Error()})
		return
	}

	unitCost := 0
	itemType := strings.ToLower(req.ItemType)

	if itemType == "gu" {
		// Calculate Gu cost: lookup rank from gu_dictionary
		var guRank int
		err = tx.QueryRow(`SELECT rank FROM gu_dictionary WHERE id = $1`, req.ItemID).Scan(&guRank)
		if err != nil {
			guRank = 1 // default to 1
		}

		// Level gate: Gu rank must be <= sect level
		if guRank > s.Level {
			c.JSON(http.StatusForbidden, gin.H{
				"error": fmt.Sprintf("Tông môn chưa đạt cấp %d để nhập Cổ trùng hạng %d! (Hiện tại: Cấp %d)", guRank, guRank, s.Level),
			})
			return
		}

		unitCost = guRank * 100 // 100 funds per rank
	} else if itemType == "material" {
		// Check material rank for level gating
		var matRank int
		err = tx.QueryRow(`SELECT rank FROM item_dictionary WHERE id = $1`, req.ItemID).Scan(&matRank)
		if err != nil {
			matRank = 1
		}

		// Level gate: material rank must be <= sect level
		if matRank > s.Level {
			c.JSON(http.StatusForbidden, gin.H{
				"error": fmt.Sprintf("Tông môn chưa đạt cấp %d để nhập Linh tài hạng %d! (Hiện tại: Cấp %d)", matRank, matRank, s.Level),
			})
			return
		}

		// Calculate material cost based on rank
		unitCost = matRank * 50 // 50 funds per rank
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại vật phẩm không hợp lệ (chỉ hỗ trợ gu hoặc material)"})
		return
	}

	totalCost := unitCost * req.Quantity
	if s.Funds < totalCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quỹ tông môn không đủ! Cần " + string(rune('0'+totalCost)) + " quỹ tông môn"})
		return
	}

	// Deduct funds
	s.Funds -= totalCost

	// Update vault json
	if itemType == "gu" {
		var vaultGus []SectVaultGuItem
		if len(s.VaultGu) > 0 {
			_ = json.Unmarshal(s.VaultGu, &vaultGus)
		}

		found := false
		for i, v := range vaultGus {
			if v.ID == req.ItemID {
				vaultGus[i].Quantity += req.Quantity
				found = true
				break
			}
		}

		if !found {
			// Cost for members to buy from vault uses pricing package
			var guRank, guBasePrice int
			_ = tx.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM gu_dictionary WHERE id = $1`, req.ItemID).Scan(&guBasePrice, &guRank)
			if guRank == 0 {
				guRank = 1
			}
			vaultGus = append(vaultGus, SectVaultGuItem{
				ID:       req.ItemID,
				Cost:     pricing.VaultCost(pricing.GuBasePrice(guBasePrice, guRank)),
				Quantity: req.Quantity,
			})
		}

		newVaultGu, _ := json.Marshal(vaultGus)
		_, err = tx.Exec(`UPDATE sects SET vault_gu = $1, funds = $2 WHERE id = $3`, newVaultGu, s.Funds, s.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault: " + err.Error()})
			return
		}
	} else {
		vaultMats := make(map[string]int)
		if len(s.VaultMaterials) > 0 {
			_ = json.Unmarshal(s.VaultMaterials, &vaultMats)
		}

		vaultMats[req.ItemID] += req.Quantity

		newVaultMats, _ := json.Marshal(vaultMats)
		_, err = tx.Exec(`UPDATE sects SET vault_materials = $1, funds = $2 WHERE id = $3`, newVaultMats, s.Funds, s.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault: " + err.Error()})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Nhập thêm vật phẩm vào bảo khố thành công!",
		"new_funds": s.Funds,
	})
}

// UpgradeSect upgrades the sect's level
func UpgradeSect(c *gin.Context) {
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

	// Fetch caller's sect and rank
	var sectID, callerRank string
	err = tx.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if callerRank != "CHUONG_MON" && callerRank != "TRUONG_LAO" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Trưởng Lão hoặc Chưởng Môn mới có quyền tăng cấp tông môn!"})
		return
	}

	// Fetch level and funds
	var level, funds int
	err = tx.QueryRow(`SELECT level, funds FROM sects WHERE id = $1`, sectID).Scan(&level, &funds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sect details"})
		return
	}

	cost := level * 10000
	if funds < cost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quỹ tông môn không đủ để nâng cấp!"})
		return
	}

	funds -= cost
	level++

	_, err = tx.Exec(`UPDATE sects SET level = $1, funds = $2 WHERE id = $3`, level, funds, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade sect"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Nâng cấp tông môn thành công!",
		"new_level": level,
		"new_funds": funds,
	})
}

type ToggleSectApprovalRequest struct {
	RequireApproval bool `json:"require_approval"`
}

// ToggleSectApproval toggles the require_approval setting for a sect
func ToggleSectApproval(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req ToggleSectApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var sectID, callerRank string
	err = tx.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if callerRank != "CHUONG_MON" && callerRank != "TRUONG_LAO" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Trưởng Lão hoặc Chưởng Môn mới có quyền thay đổi thiết lập này!"})
		return
	}

	_, err = tx.Exec(`UPDATE sects SET require_approval = $1 WHERE id = $2`, req.RequireApproval, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect setting"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	msg := "Đã bật duyệt thành viên gia nhập!"
	if !req.RequireApproval {
		msg = "Đã tắt duyệt thành viên, cho phép gia nhập tự do!"
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          msg,
		"require_approval": req.RequireApproval,
	})
}

// GetJoinRequests fetches pending join requests for the sect
func GetJoinRequests(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var sectID, callerRank string
	err := db.DB.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if callerRank != "CHUONG_MON" && callerRank != "TRUONG_LAO" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Không đủ quyền hạn!"})
		return
	}

	rows, err := db.DB.Query(`
		SELECT r.id, r.sect_id, r.player_id, r.status, r.created_at, p.username, p.rank
		FROM sect_join_requests r
		JOIN players p ON r.player_id = p.id
		WHERE r.sect_id = $1 AND r.status = 'PENDING'
		ORDER BY r.created_at ASC
	`, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch join requests: " + err.Error()})
		return
	}
	defer rows.Close()

	var reqs []SectJoinRequest
	for rows.Next() {
		var r SectJoinRequest
		if err := rows.Scan(&r.ID, &r.SectID, &r.PlayerID, &r.Status, &r.CreatedAt, &r.Username, &r.PlayerRank); err == nil {
			reqs = append(reqs, r)
		}
	}

	c.JSON(http.StatusOK, reqs)
}

type ActionJoinRequestRequest struct {
	Action string `json:"action" binding:"required"` // approve, reject
}

// ActionJoinRequest handles join request approval/rejection
func ActionJoinRequest(c *gin.Context) {
	requestIDStr := c.Param("requestId")
	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID format"})
		return
	}

	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req ActionJoinRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var sectID, callerRank string
	err = tx.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if callerRank != "CHUONG_MON" && callerRank != "TRUONG_LAO" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Không đủ quyền hạn để duyệt yêu cầu!"})
		return
	}

	// Verify request belongs to this sect
	var targetPlayerID uuid.UUID
	var reqSectID string
	err = tx.QueryRow(`SELECT player_id, sect_id FROM sect_join_requests WHERE id = $1`, requestID).Scan(&targetPlayerID, &reqSectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Yêu cầu gia nhập không tồn tại!"})
		return
	}

	if reqSectID != sectID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Yêu cầu gia nhập này thuộc tông môn khác!"})
		return
	}

	if strings.ToLower(req.Action) == "approve" {
		// Check if player is already in a sect
		var existingSect string
		err = tx.QueryRow(`SELECT sect_id FROM sect_members WHERE player_id = $1`, targetPlayerID).Scan(&existingSect)
		if err == nil {
			// Delete their requests since they already joined somewhere else
			_, _ = tx.Exec(`DELETE FROM sect_join_requests WHERE player_id = $1`, targetPlayerID)
			_ = tx.Commit()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu này đã gia nhập tông môn khác!"})
			return
		}

		// Add as Ngoại Môn
		_, err = tx.Exec(`
			INSERT INTO sect_members (sect_id, player_id, member_rank, contribution_points)
			VALUES ($1, $2, 'NGOAI_MON', 0)
		`, sectID, targetPlayerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add member to sect: " + err.Error()})
			return
		}

		// Delete all pending requests for this player (since they joined this sect now)
		_, err = tx.Exec(`DELETE FROM sect_join_requests WHERE player_id = $1`, targetPlayerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clean join requests: " + err.Error()})
			return
		}
	} else if strings.ToLower(req.Action) == "reject" {
		// Just delete the request to allow them to apply again later
		_, err = tx.Exec(`DELETE FROM sect_join_requests WHERE id = $1`, requestID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject join request: " + err.Error()})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hành động không hợp lệ (chỉ chấp nhận approve hoặc reject)"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	msg := "Đã từ chối yêu cầu gia nhập!"
	if strings.ToLower(req.Action) == "approve" {
		msg = "Đã đồng ý cho đệ tử gia nhập tông môn!"
	}

	c.JSON(http.StatusOK, gin.H{"message": msg})
}

// GetPromotionRequests fetches pending promotion requests for the sect
func GetPromotionRequests(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var sectID, callerRank string
	err := db.DB.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if callerRank != "CHUONG_MON" && callerRank != "TRUONG_LAO" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Không đủ quyền hạn!"})
		return
	}

	rows, err := db.DB.Query(`
		SELECT r.id, r.sect_id, r.player_id, r.current_rank, r.target_rank, r.status, r.created_at, p.username, p.rank
		FROM sect_promotion_requests r
		JOIN players p ON r.player_id = p.id
		WHERE r.sect_id = $1 AND r.status = 'PENDING'
		ORDER BY r.created_at ASC
	`, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch promotion requests: " + err.Error()})
		return
	}
	defer rows.Close()

	var reqs []SectPromotionRequest
	for rows.Next() {
		var r SectPromotionRequest
		if err := rows.Scan(&r.ID, &r.SectID, &r.PlayerID, &r.CurrentRank, &r.TargetRank, &r.Status, &r.CreatedAt, &r.Username, &r.PlayerRank); err == nil {
			reqs = append(reqs, r)
		}
	}

	c.JSON(http.StatusOK, reqs)
}

type ActionPromotionRequestRequest struct {
	Action string `json:"action" binding:"required"` // approve, reject
}

// ActionPromotionRequest handles promotion request approval/rejection
func ActionPromotionRequest(c *gin.Context) {
	requestIDStr := c.Param("requestId")
	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID format"})
		return
	}

	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req ActionPromotionRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var sectID, callerRank string
	err = tx.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if callerRank != "CHUONG_MON" && callerRank != "TRUONG_LAO" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Không đủ quyền hạn để duyệt yêu cầu thăng chức!"})
		return
	}

	// Fetch promotion request details
	var targetPlayerID uuid.UUID
	var reqSectID, currentRank, targetRank string
	err = tx.QueryRow(`SELECT player_id, sect_id, current_rank, target_rank FROM sect_promotion_requests WHERE id = $1`, requestID).Scan(&targetPlayerID, &reqSectID, &currentRank, &targetRank)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Yêu cầu thăng chức không tồn tại!"})
		return
	}

	if reqSectID != sectID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Yêu cầu thăng chức này thuộc tông môn khác!"})
		return
	}

	if strings.ToLower(req.Action) == "approve" {
		// Update member rank in sect_members
		_, err = tx.Exec(`UPDATE sect_members SET member_rank = $1 WHERE player_id = $2`, targetRank, targetPlayerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update member rank: " + err.Error()})
			return
		}

		// Delete request
		_, err = tx.Exec(`DELETE FROM sect_promotion_requests WHERE id = $1`, requestID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete request: " + err.Error()})
			return
		}
	} else if strings.ToLower(req.Action) == "reject" {
		// Refund contribution points
		var costPoints int
		switch targetRank {
		case "NOI_MON":
			costPoints = 100
		case "TINH_ANH":
			costPoints = 500
		case "CHAN_TRUYEN":
			costPoints = 2000
		case "TRUONG_LAO":
			costPoints = 10000
		}

		if costPoints > 0 {
			_, err = tx.Exec(`UPDATE sect_members SET contribution_points = contribution_points + $1 WHERE player_id = $2`, costPoints, targetPlayerID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refund contribution points: " + err.Error()})
				return
			}
		}

		// Delete request
		_, err = tx.Exec(`DELETE FROM sect_promotion_requests WHERE id = $1`, requestID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete request: " + err.Error()})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hành động không hợp lệ"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	msg := "Đã bác bỏ yêu cầu thăng chức!"
	if strings.ToLower(req.Action) == "approve" {
		msg = "Đã đồng ý thăng chức cho đệ tử!"
	}

	c.JSON(http.StatusOK, gin.H{"message": msg})
}
