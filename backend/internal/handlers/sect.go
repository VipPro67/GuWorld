package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/pricing"
)

type Sect struct {
	ID                       string          `json:"id"`
	Name                     string          `json:"name"`
	Description              string          `json:"description"`
	Region                   string          `json:"region"`
	VaultGu                  json.RawMessage `json:"vault_gu"`
	VaultMaterials           json.RawMessage `json:"vault_materials"`
	VaultRecipes             json.RawMessage `json:"vault_recipes"`
	Funds                    int64           `json:"funds"`
	Level                    int             `json:"level"`
	RequireApproval          bool            `json:"require_approval"`
	RequirePromotionApproval bool            `json:"require_promotion_approval"`
}

type SectJoinRequest struct {
	ID         uuid.UUID `json:"id"`
	SectID     string    `json:"sect_id"`
	PlayerID   uuid.UUID `json:"player_id"`
	Status     string    `json:"status"` // PENDING, APPROVED, REJECTED
	CreatedAt  time.Time `json:"created_at"`
	Username   string    `json:"username,omitempty"`
	AvatarURL  string    `json:"avatar_url,omitempty"`
	PlayerRank int       `json:"player_rank,omitempty"`
}

type SectPromotionRequest struct {
	ID          uuid.UUID `json:"id"`
	SectID      string    `json:"sect_id"`
	PlayerID    uuid.UUID `json:"player_id"`
	CurrentRank string    `json:"current_rank"`
	TargetRank  string    `json:"target_rank"`
	Status      string    `json:"status"` // PENDING, APPROVED, REJECTED
	CreatedAt   time.Time `json:"created_at"`
	Username    string    `json:"username,omitempty"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	PlayerRank  int       `json:"player_rank,omitempty"`
}

type SectMember struct {
	ID                 uuid.UUID `json:"id"`
	SectID             string    `json:"sect_id"`
	PlayerID           uuid.UUID `json:"player_id"`
	MemberRank         string    `json:"member_rank"` // NGOAI_MON, NOI_MON, TINH_ANH, CHAN_TRUYEN, TRUONG_LAO, HO_PHAP, CHUONG_MON
	ContributionPoints int64     `json:"contribution_points"`
	MasterID           *string   `json:"master_id"`
	JoinedAt           time.Time `json:"joined_at"`
}

type SectMission struct {
	ID                 uuid.UUID  `json:"id"`
	SectID             string     `json:"sect_id"`
	Title              string     `json:"title"`
	Description        string     `json:"description"`
	MissionType        string     `json:"mission_type"` // GATHER, COMBAT, REFINE, DONATE
	TargetID           string     `json:"target_id"`
	TargetAmount       int        `json:"target_amount"`
	RewardContribution int        `json:"reward_contribution"`
	RequiredMemberRank string     `json:"required_member_rank"`
	ExpiresAt          *time.Time `json:"expires_at"`
}

type PlayerSectMission struct {
	ID              uuid.UUID  `json:"id"`
	PlayerID        uuid.UUID  `json:"player_id"`
	MissionID       uuid.UUID  `json:"mission_id"`
	CurrentProgress int        `json:"current_progress"`
	Status          string     `json:"status"` // IN_PROGRESS, COMPLETED, FAILED
	AssignedAt      time.Time  `json:"assigned_at"`
	ExpiresAt       *time.Time `json:"expires_at"`
}

// sectRankValue ranks member ranks by seniority (lower = junior).
// CHUONG_MON is reserved for the founder/transferred leader, never reached by promotion.
func sectRankValue(rank string) int {
	switch rank {
	case "NGOAI_MON":
		return 1
	case "NOI_MON":
		return 2
	case "TINH_ANH":
		return 3
	case "CHAN_TRUYEN":
		return 4
	case "TRUONG_LAO":
		return 5
	case "HO_PHAP":
		return 6
	case "CHUONG_MON":
		return 7
	default:
		return 0
	}
}

// canManageSect reports whether the rank may manage sect operations (missions, vault, settings, kicks).
func canManageSect(rank string) bool {
	return rank == "CHUONG_MON" || rank == "TRUONG_LAO" || rank == "HO_PHAP"
}

type SectVaultGuItem struct {
	ID       string  `json:"id"`
	Rank     int     `json:"rank,omitempty"`
	Cost     int     `json:"cost"`
	Quantity int     `json:"quantity"`
	Hunger   float64 `json:"hunger,omitempty"`
}

// vaultGuMaxHunger returns the effective max hunger for a vault Gu group,
// falling back to 100 when the dictionary entry is missing or lacks a max.
func vaultGuMaxHunger(id string, rank int) float64 {
	if spec, ok := models.GetGu(id, rank); ok && spec.MaxHunger > 0 {
		return float64(spec.MaxHunger)
	}
	if spec, ok := models.GetGuAny(id); ok && spec.MaxHunger > 0 {
		return float64(spec.MaxHunger)
	}
	return 100
}

// normalizeVaultGuHunger backfills legacy entries (missing hunger) to the
// effective max and clamps the value into [0, max].
func normalizeVaultGuHunger(id string, rank int, h float64) float64 {
	max := vaultGuMaxHunger(id, rank)
	if h <= 0 {
		return max
	}
	if h > max {
		return max
	}
	return h
}

// mergeVaultGuHunger raises the group hunger to the donated Gu's hunger when
// the incoming Gu is fresher (max of existing vs donated), clamped to [0, max].
func mergeVaultGuHunger(id string, rank int, current, donated float64) float64 {
	max := vaultGuMaxHunger(id, rank)
	merged := current
	if donated > merged {
		merged = donated
	}
	if merged <= 0 {
		return max
	}
	if merged > max {
		return max
	}
	return merged
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

type DonateRecipeRequest struct {
	RecipeID string `json:"recipe_id" binding:"required"`
}

type VaultBuyRequest struct {
	ItemType string `json:"item_type" binding:"required"` // gu, material, recipe
	ItemID   string `json:"item_id" binding:"required"`
	Rank     int    `json:"rank,omitempty"`
	Quantity int    `json:"quantity" binding:"min=1"`
}

// GetSects lists all available sects
func GetSects(c *gin.Context) {
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT id, name, description, region, vault_gu, vault_materials, vault_recipes, funds, level, require_approval, require_promotion_approval FROM sects`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sects: " + err.Error()})
		return
	}
	defer rows.Close()

	var sects []Sect
	for rows.Next() {
		var s Sect
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Region, &s.VaultGu, &s.VaultMaterials, &s.VaultRecipes, &s.Funds, &s.Level, &s.RequireApproval, &s.RequirePromotionApproval); err == nil {
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

		_, err = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 2, $1) WHERE id = $2", MaxHumanQi, playerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Nhân Khí"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Gửi yêu cầu gia nhập thành công, vui lòng chờ duyệt! Nhân Khí +2."})
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

	_, err = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 3, $1) WHERE id = $2", MaxHumanQi, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Nhân Khí"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	var username string
	_ = db.DB.QueryRow(`SELECT username FROM players WHERE id = $1`, playerID).Scan(&username)
	if username != "" {
		_ = SendSystemMessage("sect", sql.NullString{String: sectID, Valid: true}, fmt.Sprintf("Chào mừng đạo hữu %s đã gia nhập tông môn!", username))
	}

	auditlog.Logf(db.DB, &playerID, "sect.join", "sectId=%s username=%s", sectID, username)
	c.JSON(http.StatusOK, gin.H{"message": "Gia nhập tông môn thành công! Nhân Khí +3."})
}

// CancelSectJoinRequest withdraws a player's pending join request
func CancelSectJoinRequest(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	_, err := db.DB.Exec(`DELETE FROM sect_join_requests WHERE player_id = $1 AND status = 'PENDING'`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel join request: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã hủy đơn xin gia nhập tông môn."})
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

	// Get player username and sect ID before deleting
	var sectID, username string
	err = tx.QueryRow(`
		SELECT m.sect_id, p.username 
		FROM sect_members m
		JOIN players p ON m.player_id = p.id
		WHERE m.player_id = $1
	`, playerID).Scan(&sectID, &username)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	// Delete membership
	_, err = tx.Exec(`DELETE FROM sect_members WHERE player_id = $1`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to leave sect"})
		return
	}

	// Delete active player missions
	_, _ = tx.Exec(`DELETE FROM player_sect_missions WHERE player_id = $1`, playerID)

	// Delete pending promotion requests
	_, _ = tx.Exec(`DELETE FROM sect_promotion_requests WHERE player_id = $1 AND sect_id = $2`, playerID, sectID)

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	// Trigger System Chat Notification on Sect channel
	_ = SendSystemMessage("sect", sql.NullString{String: sectID, Valid: true}, fmt.Sprintf("Đạo hữu %s đã rời khỏi tông môn.", username))

	auditlog.Logf(db.DB, &playerID, "sect.leave", "sectId=%s username=%s", sectID, username)
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
	err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT id, sect_id, player_id, member_rank, contribution_points, master_id, joined_at
		FROM sect_members
		WHERE player_id = $1
	`, playerID).Scan(&m.ID, &m.SectID, &m.PlayerID, &m.MemberRank, &m.ContributionPoints, &m.MasterID, &m.JoinedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			// Check if they have a pending join request
			var pendingSectID, pendingSectName string
			err = db.DB.QueryRowContext(c.Request.Context(), `
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
	err = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT id, name, description, region, vault_gu, vault_materials, vault_recipes, funds, level, require_approval, require_promotion_approval
		FROM sects
		WHERE id = $1
	`, m.SectID).Scan(&s.ID, &s.Name, &s.Description, &s.Region, &s.VaultGu, &s.VaultMaterials, &s.VaultRecipes, &s.Funds, &s.Level, &s.RequireApproval, &s.RequirePromotionApproval)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sect details"})
		return
	}

	// Active mission if any
	var activeMission SectMission
	var activeProgress int
	var activePMId uuid.UUID
	hasActive := false
	err = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT pm.id, sm.id, sm.title, sm.description, sm.mission_type, sm.target_id, sm.target_amount, sm.reward_contribution, pm.current_progress, pm.expires_at
		FROM player_sect_missions pm
		JOIN sect_missions sm ON pm.mission_id = sm.id
		WHERE pm.player_id = $1 AND pm.status = 'IN_PROGRESS'
		  AND (pm.expires_at IS NULL OR pm.expires_at > now())
		LIMIT 1
	`, playerID).Scan(&activePMId, &activeMission.ID, &activeMission.Title, &activeMission.Description, &activeMission.MissionType, &activeMission.TargetID, &activeMission.TargetAmount, &activeMission.RewardContribution, &activeProgress, &activeMission.ExpiresAt)
	if err == nil {
		hasActive = true
	}

	// Leaderboard inside the sect
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT p.username, p.avatar_url, sm.member_rank, sm.contribution_points
		FROM sect_members sm
		JOIN players p ON sm.player_id = p.id
		WHERE sm.sect_id = $1
		ORDER BY sm.contribution_points DESC
		LIMIT 10
	`, m.SectID)

	type LeaderboardEntry struct {
		Username     string `json:"username"`
		AvatarURL    string `json:"avatar_url"`
		Rank         string `json:"rank"`
		Contribution int    `json:"contribution"`
	}
	var leaderboard []LeaderboardEntry
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var entry LeaderboardEntry
			if err := rows.Scan(&entry.Username, &entry.AvatarURL, &entry.Rank, &entry.Contribution); err == nil {
				leaderboard = append(leaderboard, entry)
			}
		}
	}

	// Check if this player has a pending promotion request
	var hasPendingPromotion bool
	var pendingPromoTarget string
	err = db.DB.QueryRowContext(c.Request.Context(), `
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
	var costPoints int64

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
	case "TRUONG_LAO":
		nextRank = "HO_PHAP"
		reqRank = 6
		costPoints = 30000
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cảnh giới tông môn đã đạt đỉnh phong (Hộ Pháp)!"})
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

	// Whether this promotion must be approved by sect leadership first
	requiresApproval := false

	// Option "duyệt lên cấp": khi bật, MỌI thăng cấp đều phải qua duyệt
	var requirePromotionApproval bool
	err = tx.QueryRow(`SELECT require_promotion_approval FROM sects WHERE id = $1`, m.SectID).Scan(&requirePromotionApproval)
	if err == nil && requirePromotionApproval {
		requiresApproval = true
	} else {
		// Manual check for high tier ranks: CHAN_TRUYEN, TRUONG_LAO and HO_PHAP
		if nextRank == "CHAN_TRUYEN" || nextRank == "TRUONG_LAO" || nextRank == "HO_PHAP" {
			var leaderCount int
			err = tx.QueryRow(`SELECT COUNT(1) FROM sect_members WHERE sect_id = $1 AND member_rank = 'CHUONG_MON'`, m.SectID).Scan(&leaderCount)
			if err == nil && leaderCount > 0 {
				requiresApproval = true
			}
		}
	}

	if requiresApproval {
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

	if player.PrimevalStones < float64(req.Amount) {
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

	player.PrimevalStones -= float64(req.Amount)
	m.ContributionPoints += int64(req.Amount)

	// Increase sect funds
	_, err = tx.Exec(`UPDATE sects SET funds = funds + $1 WHERE id = $2`, req.Amount, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect funds"})
		return
	}

	qiGained := AddHumanQi(&player, 5.0)

	_, err = tx.Exec(`UPDATE players SET primeval_stones = $1, human_qi = $2 WHERE id = $3`, player.PrimevalStones, player.HumanQi, player.ID)
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

	// Trigger System Chat Notification on Sect channel
	_ = SendSystemMessage("sect", sql.NullString{String: m.SectID, Valid: true}, fmt.Sprintf("Đạo hữu %s đã quyên góp %d Nguyên Thạch cho tông môn.", player.Username, req.Amount))

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.donate_stones", EntityType: "sect",
		EntityID: m.SectID, RequestSummary: fmt.Sprintf("%d Nguyên Thạch → %d điểm cống hiến", req.Amount, req.Amount),
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{
		"message":             fmt.Sprintf("Quyên góp thành công! Nhân Khí +%.0f.", qiGained),
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

	rows, err := tx.Query(`SELECT category, quantity FROM player_inventory_items WHERE player_id = $1 AND item_id = $2`, playerID.String(), req.MaterialID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking inventory"})
		return
	}

	type invItem struct {
		category string
		quantity int
	}
	var invs []invItem
	totalQty := 0
	for rows.Next() {
		var iv invItem
		if err := rows.Scan(&iv.category, &iv.quantity); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error scanning inventory"})
			return
		}
		invs = append(invs, iv)
		totalQty += iv.quantity
	}
	rows.Close()

	if totalQty < req.Amount {
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
	var itemType string
	if err = tx.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1), COALESCE(item_type,'') FROM item_dictionary WHERE id=$1`, req.MaterialID).Scan(&dbBasePrice, &matRank, &itemType); err != nil {
		matRank = 1
		dbBasePrice = 0
		itemType = ""
	}
	pointsPerItem := pricing.DonatePoints(pricing.ItemBasePrice(dbBasePrice, matRank))

	gainedContribution := pointsPerItem * req.Amount

	preferredCat := "materials"
	if itemType == "Consumable" {
		preferredCat = "consumables"
	}

	remainingDeduct := req.Amount
	for _, iv := range invs {
		if iv.category == preferredCat {
			deduct := iv.quantity
			if deduct > remainingDeduct {
				deduct = remainingDeduct
			}
			if err := InventorySvc.RemoveItem(tx, playerID.String(), req.MaterialID, iv.category, deduct); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa vật phẩm khỏi túi đồ"})
				return
			}
			remainingDeduct -= deduct
			if remainingDeduct <= 0 {
				break
			}
		}
	}
	if remainingDeduct > 0 {
		for _, iv := range invs {
			if iv.category != preferredCat {
				deduct := iv.quantity
				if deduct > remainingDeduct {
					deduct = remainingDeduct
				}
				if err := InventorySvc.RemoveItem(tx, playerID.String(), req.MaterialID, iv.category, deduct); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa vật phẩm khỏi túi đồ"})
					return
				}
				remainingDeduct -= deduct
				if remainingDeduct <= 0 {
					break
				}
			}
		}
	}

	m.ContributionPoints += int64(gainedContribution)
	qiGained := AddHumanQi(&player, 5.0)

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

	_, err = tx.Exec(`UPDATE sect_members SET contribution_points = $1 WHERE player_id = $2`, m.ContributionPoints, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update contribution"})
		return
	}

	_, err = tx.Exec(`UPDATE players SET human_qi = $1 WHERE id = $2`, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update human qi"})
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

	// Trigger System Chat Notification on Sect channel
	_ = SendSystemMessage("sect", sql.NullString{String: m.SectID, Valid: true}, fmt.Sprintf("Đạo hữu %s đã quyên góp %d %s cho tông môn.", player.Username, req.Amount, materialName))

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.donate_materials", EntityType: "sect",
		EntityID: m.SectID, RequestSummary: fmt.Sprintf("%s x%d → %d điểm", materialName, req.Amount, gainedContribution),
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{
		"message":             fmt.Sprintf("Quyên góp linh tài thành công! Nhân Khí +%.0f.", qiGained),
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
	err = tx.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM gu_dictionary WHERE id = $1 AND rank = $2`, guInstance.GuID, guInstance.Rank).Scan(&guBasePrice, &guRank)
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

	// Increment quantity if Gu type already exists in vault, otherwise add new
	// entry. Raise the group hunger to the donated Gu's hunger when fresher.
	donatedGuFound := false
	for i, v := range vaultGus {
		if v.ID == guInstance.GuID && v.Rank == guInstance.Rank {
			vaultGus[i].Quantity++
			vaultGus[i].Hunger = mergeVaultGuHunger(guInstance.GuID, guInstance.Rank, v.Hunger, float64(guInstance.Hunger))
			donatedGuFound = true
			break
		}
	}
	if !donatedGuFound {
		vaultGus = append(vaultGus, SectVaultGuItem{
			ID:       guInstance.GuID,
			Rank:     guInstance.Rank,
			Cost:     vaultItemCost,
			Quantity: 1,
			Hunger:   normalizeVaultGuHunger(guInstance.GuID, guInstance.Rank, float64(guInstance.Hunger)),
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
	m.ContributionPoints += int64(gainedContribution)
	qiGained := AddHumanQi(&player, 5.0)

	// Increase sect funds
	_, err = tx.Exec(`UPDATE sects SET funds = funds + $1 WHERE id = $2`, gainedContribution, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect funds"})
		return
	}

	if err := db.RemovePlayerGu(tx, guInstance.InstanceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update gu inventory"})
		return
	}
	_, err = tx.Exec(`UPDATE players SET human_qi = $1 WHERE id = $2`, player.HumanQi, player.ID)
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
	if spec, ok := models.GetGu(guInstance.GuID, guInstance.Rank); ok {
		guName = spec.Name
	}

	// Trigger System Chat Notification on Sect channel
	_ = SendSystemMessage("sect", sql.NullString{String: m.SectID, Valid: true}, fmt.Sprintf("Đạo hữu %s đã quyên góp Cổ trùng %s cho tông môn.", player.Username, guName))

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.donate_gu", EntityType: "sect",
		EntityID: m.SectID, RequestSummary: fmt.Sprintf("%s → %d điểm", guName, gainedContribution),
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{
		"message":             fmt.Sprintf("Quyên góp Cổ trùng thành công! Nhân Khí +%.0f.", qiGained),
		"gu_inventory":        player.GuInventory,
		"contribution_points": m.ContributionPoints,
	})
}

// DonateRecipe converts a player recipe to contribution points and adds it to
// the sect vault so other members can learn it.
func DonateRecipe(c *gin.Context) {
	var req DonateRecipeRequest
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

	progress, err := RecipeSvc.Get(tx, playerID.String(), req.RecipeID)
	if err != nil || progress == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bí phương không tồn tại trong túi đồ!"})
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

	// Resolve target Gu rank and compute contribution points scaled by completion rate.
	var targetGuRank int = 1
	if _, tr, _, err := ParseRecipeID(req.RecipeID); err == nil {
		targetGuRank = tr
	} else {
		_ = tx.QueryRow(`
			SELECT COALESCE(gd.rank,1)
			FROM refining_recipes rr
			JOIN gu_dictionary gd ON gd.id=rr.target_gu_id AND gd.rank=rr.target_rank
			WHERE ('recipe_' || rr.target_gu_id || '_' || rr.target_rank::text || '_' || rr.variant::text)=$1
		`, req.RecipeID).Scan(&targetGuRank)
	}
	basePoints := pricing.RecipeDonatePoints(targetGuRank)
	gainedContribution := int(float64(basePoints) * progress.CompletionRate / 100.0)
	if gainedContribution < 1 {
		gainedContribution = 1
	}

	if err := RecipeSvc.Delete(tx, playerID.String(), req.RecipeID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa Bí phương khỏi túi đồ"})
		return
	}

	m.ContributionPoints += int64(gainedContribution)
	qiGained := AddHumanQi(&player, 5.0)

	// Increase sect funds
	_, err = tx.Exec(`UPDATE sects SET funds = funds + $1 WHERE id = $2`, gainedContribution, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect funds"})
		return
	}

	// Add recipe to sect vault, keeping the higher completion rate.
	var vaultRecipes []models.SectVaultRecipe
	_ = tx.QueryRow(`SELECT vault_recipes FROM sects WHERE id = $1`, m.SectID).Scan(&vaultRecipes)
	if vaultRecipes == nil {
		vaultRecipes = []models.SectVaultRecipe{}
	}
	found := false
	for i := range vaultRecipes {
		if vaultRecipes[i].RecipeID == req.RecipeID {
			if progress.CompletionRate > vaultRecipes[i].CompletionRate {
				vaultRecipes[i].CompletionRate = progress.CompletionRate
			}
			found = true
			break
		}
	}
	if !found {
		vaultRecipes = append(vaultRecipes, models.SectVaultRecipe{
			RecipeID:       req.RecipeID,
			CompletionRate: progress.CompletionRate,
		})
	}
	newVaultRecipes, _ := json.Marshal(vaultRecipes)
	_, err = tx.Exec(`UPDATE sects SET vault_recipes = $1 WHERE id = $2`, newVaultRecipes, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault recipes"})
		return
	}

	_, err = tx.Exec(`UPDATE sect_members SET contribution_points = $1 WHERE player_id = $2`, m.ContributionPoints, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update contribution"})
		return
	}

	_, err = tx.Exec(`UPDATE players SET human_qi = $1 WHERE id = $2`, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update human qi"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	itemName := req.RecipeID
	if spec, ok := models.GetGuAny(req.RecipeID); ok {
		itemName = spec.Name
	}

	// Trigger System Chat Notification on Sect channel
	_ = SendSystemMessage("sect", sql.NullString{String: m.SectID, Valid: true}, fmt.Sprintf("Đạo hữu %s đã quyên góp Bí phương %s cho tông môn.", player.Username, itemName))

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.donate_recipe", EntityType: "sect",
		EntityID: m.SectID, RequestSummary: fmt.Sprintf("%s (%.0f%%) → %d điểm", itemName, progress.CompletionRate, gainedContribution),
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{
		"message":             fmt.Sprintf("Quyên góp Bí phương thành công! Nhân Khí +%.0f.", qiGained),
		"contribution_points": m.ContributionPoints,
	})
}

// BulkDonateRequest represents a collection of materials, Gus and recipes to donate
type BulkDonateRequest struct {
	Materials []DonateMaterialsRequest `json:"materials"`
	Gus       []DonateGuRequest        `json:"gus"`
	Recipes   []DonateRecipeRequest    `json:"recipes"`
}

// DonateBulk converts multiple player materials and Gu bugs to contribution points in a single transaction
func DonateBulk(c *gin.Context) {
	var req BulkDonateRequest
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

	totalGainedContribution := 0
	var auditSummaries []string

	// 1. Process Materials
	if len(req.Materials) > 0 {
		// Fetch sect vault materials
		var vaultMaterialsRaw json.RawMessage
		err = tx.QueryRow(`SELECT vault_materials FROM sects WHERE id = $1`, m.SectID).Scan(&vaultMaterialsRaw)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sect vault materials"})
			return
		}
		vaultMats := make(map[string]int)
		if len(vaultMaterialsRaw) > 0 {
			_ = json.Unmarshal(vaultMaterialsRaw, &vaultMats)
		}

		for _, matReq := range req.Materials {
			rows, err := tx.Query(`SELECT category, quantity FROM player_inventory_items WHERE player_id = $1 AND item_id = $2`, playerID.String(), matReq.MaterialID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking inventory"})
				return
			}

			type invItem struct {
				category string
				quantity int
			}
			var invs []invItem
			totalQty := 0
			for rows.Next() {
				var iv invItem
				if err := rows.Scan(&iv.category, &iv.quantity); err != nil {
					rows.Close()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error scanning inventory"})
					return
				}
				invs = append(invs, iv)
				totalQty += iv.quantity
			}
			rows.Close()

			if totalQty < matReq.Amount {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Đạo hữu không có đủ linh tài: %s", matReq.MaterialID)})
				return
			}

			// Calculate contribution points dynamically
			var dbBasePrice, matRank int
			var itemType string
			if err = tx.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1), COALESCE(item_type,'') FROM item_dictionary WHERE id=$1`, matReq.MaterialID).Scan(&dbBasePrice, &matRank, &itemType); err != nil {
				matRank = 1
				dbBasePrice = 0
				itemType = ""
			}
			pointsPerItem := pricing.DonatePoints(pricing.ItemBasePrice(dbBasePrice, matRank))
			gained := pointsPerItem * matReq.Amount
			totalGainedContribution += gained

			preferredCat := "materials"
			if itemType == "Consumable" {
				preferredCat = "consumables"
			}

			// Update Inventory
			remainingDeduct := matReq.Amount
			for _, iv := range invs {
				if iv.category == preferredCat {
					deduct := iv.quantity
					if deduct > remainingDeduct {
						deduct = remainingDeduct
					}
					if err := InventorySvc.RemoveItem(tx, playerID.String(), matReq.MaterialID, iv.category, deduct); err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa vật phẩm khỏi túi đồ"})
						return
					}
					remainingDeduct -= deduct
					if remainingDeduct <= 0 {
						break
					}
				}
			}
			if remainingDeduct > 0 {
				for _, iv := range invs {
					if iv.category != preferredCat {
						deduct := iv.quantity
						if deduct > remainingDeduct {
							deduct = remainingDeduct
						}
						if err := InventorySvc.RemoveItem(tx, playerID.String(), matReq.MaterialID, iv.category, deduct); err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa vật phẩm khỏi túi đồ"})
							return
						}
						remainingDeduct -= deduct
						if remainingDeduct <= 0 {
							break
						}
					}
				}
			}

			// Update Sect Vault map
			vaultMats[matReq.MaterialID] += matReq.Amount

			// Get friendly name
			materialName := matReq.MaterialID
			_ = tx.QueryRow(`SELECT name FROM item_dictionary WHERE id = $1`, matReq.MaterialID).Scan(&materialName)
			if materialName == "" {
				materialName = matReq.MaterialID
			}
			auditSummaries = append(auditSummaries, fmt.Sprintf("%s x%d", materialName, matReq.Amount))
		}

		// Save updated materials vault
		newVaultMats, _ := json.Marshal(vaultMats)
		_, err = tx.Exec(`UPDATE sects SET vault_materials = $1 WHERE id = $2`, newVaultMats, m.SectID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault materials"})
			return
		}
	}

	// 2. Process Gu Instances
	if len(req.Gus) > 0 {
		// Fetch sect vault Gu
		var vaultGuRaw json.RawMessage
		err = tx.QueryRow(`SELECT vault_gu FROM sects WHERE id = $1`, m.SectID).Scan(&vaultGuRaw)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sect vault gu"})
			return
		}
		var vaultGus []SectVaultGuItem
		if len(vaultGuRaw) > 0 {
			_ = json.Unmarshal(vaultGuRaw, &vaultGus)
		}

		// Keep track of instance IDs we want to remove
		gusToRemove := make(map[string]bool)
		removedIDs := make([]string, 0, len(req.Gus))
		for _, guReq := range req.Gus {
			gusToRemove[guReq.GuInstanceID] = true
			removedIDs = append(removedIDs, guReq.GuInstanceID)
		}

		// Process each target Gu instance
		var newGuInventory []models.GuInstance
		for _, gi := range player.GuInventory {
			if gusToRemove[gi.InstanceID] {
				if gi.IsVital {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể quyên góp Bản Mệnh Cổ!"})
					return
				}
				if gi.Equipped {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Vui lòng tháo Cổ trùng ra trước khi quyên góp!"})
					return
				}

				// Fetch base price
				var guRank, guBasePrice int
				err = tx.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM gu_dictionary WHERE id = $1 AND rank = $2`, gi.GuID, gi.Rank).Scan(&guBasePrice, &guRank)
				if err != nil {
					guRank = 1
					guBasePrice = 0
				}
				effectiveBase := pricing.GuBasePrice(guBasePrice, guRank)
				gained := pricing.DonatePoints(effectiveBase)
				vaultItemCost := pricing.VaultCost(effectiveBase)

				totalGainedContribution += gained

				// Update Vault Gu count, raising group hunger when the donated
				// Gu is fresher.
				donatedGuFound := false
				for idx, v := range vaultGus {
					if v.ID == gi.GuID && v.Rank == gi.Rank {
						vaultGus[idx].Quantity++
						vaultGus[idx].Hunger = mergeVaultGuHunger(gi.GuID, gi.Rank, v.Hunger, float64(gi.Hunger))
						donatedGuFound = true
						break
					}
				}
				if !donatedGuFound {
					vaultGus = append(vaultGus, SectVaultGuItem{
						ID:       gi.GuID,
						Rank:     gi.Rank,
						Cost:     vaultItemCost,
						Quantity: 1,
						Hunger:   normalizeVaultGuHunger(gi.GuID, gi.Rank, float64(gi.Hunger)),
					})
				}

				guName := gi.GuID
				if spec, ok := models.GetGu(gi.GuID, gi.Rank); ok {
					guName = spec.Name
				}
				auditSummaries = append(auditSummaries, guName)
			} else {
				newGuInventory = append(newGuInventory, gi)
			}
		}

		// Verify that all requested Gus were actually found and processed
		if len(player.GuInventory)-len(newGuInventory) != len(req.Gus) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Một hoặc nhiều Cổ trùng không tồn tại trong túi đồ!"})
			return
		}

		player.GuInventory = newGuInventory

		// Save player Gu inventory
		if err := db.RemovePlayerGus(tx, removedIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update gu inventory"})
			return
		}

		// Save updated Gu vault
		newVaultGus, _ := json.Marshal(vaultGus)
		_, err = tx.Exec(`UPDATE sects SET vault_gu = $1 WHERE id = $2`, newVaultGus, m.SectID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault Gu list"})
			return
		}
	}

	// 3. Process Recipes
	if len(req.Recipes) > 0 {
		var vaultRecipes []models.SectVaultRecipe
		var vaultRecipesRaw json.RawMessage
		err = tx.QueryRow(`SELECT vault_recipes FROM sects WHERE id = $1`, m.SectID).Scan(&vaultRecipesRaw)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sect vault recipes"})
			return
		}
		if len(vaultRecipesRaw) > 0 {
			_ = json.Unmarshal(vaultRecipesRaw, &vaultRecipes)
		}
		if vaultRecipes == nil {
			vaultRecipes = []models.SectVaultRecipe{}
		}

		donatedRecipeIDs := make(map[string]bool)
		for _, r := range req.Recipes {
			donatedRecipeIDs[r.RecipeID] = true
		}

		for _, recipeReq := range req.Recipes {
			progress, err := RecipeSvc.Get(tx, playerID.String(), recipeReq.RecipeID)
			if err != nil || progress == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Bí phương không tồn tại trong túi đồ: %s", recipeReq.RecipeID)})
				return
			}

			var targetGuRank int = 1
			if _, tr, _, err := ParseRecipeID(recipeReq.RecipeID); err == nil {
				targetGuRank = tr
			} else {
				_ = tx.QueryRow(`
					SELECT COALESCE(gd.rank,1)
					FROM refining_recipes rr
					JOIN gu_dictionary gd ON gd.id=rr.target_gu_id AND gd.rank=rr.target_rank
					WHERE ('recipe_' || rr.target_gu_id || '_' || rr.target_rank::text || '_' || rr.variant::text)=$1
				`, recipeReq.RecipeID).Scan(&targetGuRank)
			}
			basePoints := pricing.RecipeDonatePoints(targetGuRank)
			gained := int(float64(basePoints) * progress.CompletionRate / 100.0)
			if gained < 1 {
				gained = 1
			}
			totalGainedContribution += gained

			if err := RecipeSvc.Delete(tx, playerID.String(), recipeReq.RecipeID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Không thể xóa Bí phương %s", recipeReq.RecipeID)})
				return
			}

			found := false
			for i := range vaultRecipes {
				if vaultRecipes[i].RecipeID == recipeReq.RecipeID {
					if progress.CompletionRate > vaultRecipes[i].CompletionRate {
						vaultRecipes[i].CompletionRate = progress.CompletionRate
					}
					found = true
					break
				}
			}
			if !found {
				vaultRecipes = append(vaultRecipes, models.SectVaultRecipe{
					RecipeID:       recipeReq.RecipeID,
					CompletionRate: progress.CompletionRate,
				})
			}

			recipeName := recipeReq.RecipeID
			if spec, ok := models.GetGuAny(recipeReq.RecipeID); ok {
				recipeName = spec.Name
			}
			auditSummaries = append(auditSummaries, fmt.Sprintf("%s (%.0f%%)", recipeName, progress.CompletionRate))
		}

		newVaultRecipes, _ := json.Marshal(vaultRecipes)
		_, err = tx.Exec(`UPDATE sects SET vault_recipes = $1 WHERE id = $2`, newVaultRecipes, m.SectID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault recipes"})
			return
		}
	}

	if totalGainedContribution <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có vật phẩm nào được quyên góp."})
		return
	}

	// Update Human Qi (5.0 per unique item type/instance donated)
	numItems := len(req.Materials) + len(req.Gus) + len(req.Recipes)
	qiGained := AddHumanQi(&player, 5.0*float64(numItems))

	m.ContributionPoints += int64(totalGainedContribution)

	// Update Sect funds
	_, err = tx.Exec(`UPDATE sects SET funds = funds + $1 WHERE id = $2`, totalGainedContribution, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect funds"})
		return
	}

	// Update player points
	_, err = tx.Exec(`UPDATE sect_members SET contribution_points = $1 WHERE player_id = $2`, m.ContributionPoints, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update contribution"})
		return
	}

	// Update player Human Qi
	_, err = tx.Exec(`UPDATE players SET human_qi = $1 WHERE id = $2`, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update human qi"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	// Trigger System Chat Notification on Sect channel
	summaryStr := strings.Join(auditSummaries, ", ")
	_ = SendSystemMessage("sect", sql.NullString{String: m.SectID, Valid: true}, fmt.Sprintf("Đạo hữu %s đã quyên góp hàng loạt vật phẩm: %s cho tông môn.", player.Username, summaryStr))

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.donate_bulk", EntityType: "sect",
		EntityID: m.SectID, RequestSummary: fmt.Sprintf("Bulk: %s → %d điểm", summaryStr, totalGainedContribution),
		IPAddress: c.ClientIP(), Success: true,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":             fmt.Sprintf("Quyên góp hàng loạt thành công! Nhân Khí +%.0f.", qiGained),
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

	quantity := req.Quantity
	if quantity <= 0 {
		quantity = 1
	}
	if quantity > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mỗi lần chỉ được trao đổi tối đa 100 đơn vị!"})
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
	before := append(models.GuInventory(nil), player.GuInventory...)

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

	var cost int64
	if strings.ToLower(req.ItemType) == "gu" {
		var vaultGus []SectVaultGuItem
		if err := json.Unmarshal(s.VaultGu, &vaultGus); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse vault Gu list"})
			return
		}

		targetIdx := -1
		for idx, item := range vaultGus {
			if item.ID == req.ItemID && (req.Rank == 0 || item.Rank == req.Rank) {
				targetIdx = idx
				break
			}
		}

		if targetIdx == -1 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Cổ trùng này không có sẵn trong Bảo khố!"})
			return
		}

		if vaultGus[targetIdx].Quantity < quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Bảo khố chỉ còn %d con Cổ trùng này!", vaultGus[targetIdx].Quantity)})
			return
		}

		cost = int64(vaultGus[targetIdx].Cost * quantity)
		if m.ContributionPoints < cost {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không đủ Điểm cống hiến!"})
			return
		}

		m.ContributionPoints -= cost
		vaultGus[targetIdx].Quantity -= quantity

		guSpec, _ := models.GetGuAny(req.ItemID)
		actualRank := vaultGus[targetIdx].Rank
		if actualRank == 0 {
			actualRank = guSpec.Rank
		}
		buyHunger := normalizeVaultGuHunger(req.ItemID, actualRank, vaultGus[targetIdx].Hunger)

		if vaultGus[targetIdx].Quantity <= 0 {
			vaultGus = append(vaultGus[:targetIdx], vaultGus[targetIdx+1:]...)
		}
		for i := 0; i < quantity; i++ {
			newGu := models.GuInstance{
				InstanceID: uuid.New().String(),
				GuID:       req.ItemID,
				Rank:       actualRank,
				Hunger:     int(buyHunger),
				Status:     "ACTIVE",
				Equipped:   false,
			}
			player.GuInventory = append(player.GuInventory, newGu)
		}

		if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save Gu inventory"})
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
		if !hasMat || qty < quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Linh tài này không đủ số lượng trong Bảo khố!"})
			return
		}

		// Dynamic cost from item_dictionary base_price + rank via pricing package
		{
			var matBasePrice, matRank int
			_ = tx.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM item_dictionary WHERE id=$1`, req.ItemID).Scan(&matBasePrice, &matRank)
			cost = int64(pricing.VaultCost(pricing.ItemBasePrice(matBasePrice, matRank)) * quantity)
		}

		if m.ContributionPoints < cost {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không đủ Điểm cống hiến!"})
			return
		}

		m.ContributionPoints -= cost
		vaultMats[req.ItemID] -= quantity
		if vaultMats[req.ItemID] <= 0 {
			delete(vaultMats, req.ItemID)
		}

		InventorySvc.AddItem(tx, player.ID.String(), req.ItemID, "materials", quantity)

		newVaultMats, _ := json.Marshal(vaultMats)
		_, err = tx.Exec(`UPDATE sects SET vault_materials = $1 WHERE id = $2`, newVaultMats, s.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update vault materials"})
			return
		}
	} else if strings.ToLower(req.ItemType) == "recipe" {
		var vaultRecipes []models.SectVaultRecipe
		if err := json.Unmarshal(s.VaultRecipes, &vaultRecipes); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse recipes list"})
			return
		}

		var recipeEntry *models.SectVaultRecipe
		for i := range vaultRecipes {
			if vaultRecipes[i].RecipeID == req.ItemID {
				recipeEntry = &vaultRecipes[i]
				break
			}
		}

		if recipeEntry == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Công thức này không có sẵn trong Bảo khố!"})
			return
		}

		if quantity > 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Công thức mỗi lần chỉ có thể học một lần!"})
			return
		}

		// Dynamic recipe cost based on target Gu rank via pricing package
		{
			var targetGuRank int = 1
			if _, tr, _, err := ParseRecipeID(req.ItemID); err == nil {
				targetGuRank = tr
			} else {
				_ = tx.QueryRow(`SELECT COALESCE(gd.rank,1) FROM refining_recipes rr JOIN gu_dictionary gd ON gd.id=rr.target_gu_id AND gd.rank=rr.target_rank WHERE ('recipe_' || rr.target_gu_id || '_' || rr.target_rank::text || '_' || rr.variant::text)=$1`, req.ItemID).Scan(&targetGuRank)
			}
			cost = int64(pricing.RecipeCostByGuRank(targetGuRank))
		}
		if m.ContributionPoints < cost {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không đủ Điểm cống hiến!"})
			return
		}

		m.ContributionPoints -= cost

		// Remove recipe from vault
		for i := range vaultRecipes {
			if vaultRecipes[i].RecipeID == req.ItemID {
				vaultRecipes = append(vaultRecipes[:i], vaultRecipes[i+1:]...)
				break
			}
		}
		newVaultRecipes, err := json.Marshal(vaultRecipes)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize updated recipes list"})
			return
		}
		_, err = tx.Exec(`UPDATE sects SET vault_recipes = $1 WHERE id = $2`, newVaultRecipes, s.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault recipes list"})
			return
		}

		// Grant the recipe to the buyer, keeping the higher completion rate.
		grantedRate := recipeEntry.CompletionRate
		if existing, err := RecipeSvc.Get(tx, player.ID.String(), req.ItemID); err == nil && existing != nil {
			if existing.CompletionRate > grantedRate {
				grantedRate = existing.CompletionRate
			}
		}
		if grantedRate > 100.0 {
			grantedRate = 100.0
		}
		if err := RecipeSvc.Upsert(tx, &services.RecipeProgress{
			PlayerID:       player.ID.String(),
			RecipeID:       req.ItemID,
			CompletionRate: grantedRate,
			IsUnlocked:     true,
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể học công thức từ bảo khố"})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại vật phẩm trao đổi không hợp lệ!"})
		return
	}

	qiGained := AddHumanQi(&player, 5.0)

	_, err = tx.Exec(`UPDATE sect_members SET contribution_points = $1 WHERE player_id = $2`, m.ContributionPoints, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update contribution"})
		return
	}

	_, err = tx.Exec(`UPDATE players SET human_qi = $1 WHERE id = $2`, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update human qi"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	itemName := req.ItemID
	if req.ItemType == "Gu" {
		if spec, ok := models.GetGuAny(req.ItemID); ok {
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
		EntityID:       m.SectID,
		RequestSummary: fmt.Sprintf("%s | %s x%d | giá %d điểm cống hiến", req.ItemType, itemName, quantity, cost),
		IPAddress:      c.ClientIP(), Success: true,
	})
	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"message": fmt.Sprintf("Đổi thành công %d %s! Nhân Khí +%.0f.", quantity, itemName, qiGained),
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

	auditlog.Logf(db.DB, &playerID, "sect.mission_accept", "sectId=%s missionId=%s", sm.SectID, missionID)

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
			var vaultGuRaw json.RawMessage
			err = tx.QueryRow(`SELECT vault_gu FROM sects WHERE id = $1`, memberForVault.SectID).Scan(&vaultGuRaw)
			if err == nil {
				var vaultGus []SectVaultGuItem
				if len(vaultGuRaw) > 0 {
					_ = json.Unmarshal(vaultGuRaw, &vaultGus)
				}
				// Look up Gu base_price + rank for vault cost calculation
				var donatedGuRank, donatedGuBase int
				_ = tx.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM gu_dictionary WHERE id = $1 ORDER BY rank DESC LIMIT 1`, donatedGuID).Scan(&donatedGuBase, &donatedGuRank)
				if donatedGuRank == 0 {
					donatedGuRank = 1
				}
				missionGuFound := false
				for i, v := range vaultGus {
					if v.ID == donatedGuID {
						vaultGus[i].Quantity++
						vaultGus[i].Hunger = mergeVaultGuHunger(donatedGuID, donatedGuInstRank, v.Hunger, donatedGuHunger)
						missionGuFound = true
						break
					}
				}
				if !missionGuFound {
					vaultGus = append(vaultGus, SectVaultGuItem{
						ID:       donatedGuID,
						Cost:     pricing.VaultCost(pricing.GuBasePrice(donatedGuBase, donatedGuRank)),
						Quantity: 1,
						Hunger:   normalizeVaultGuHunger(donatedGuID, donatedGuInstRank, donatedGuHunger),
					})
				}
				newVaultGus, _ := json.Marshal(vaultGus)
				_, _ = tx.Exec(`UPDATE sects SET vault_gu = $1 WHERE id = $2`, newVaultGus, memberForVault.SectID)
			}
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

	auditlog.Logf(db.DB, &playerID, "sect.mission_complete", "sectId=%s missionId=%s reward=%d", m.SectID, missionID, sm.RewardContribution)

	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"message":             "Hoàn thành nhiệm vụ tông môn! Nhận được " + strconv.Itoa(sm.RewardContribution) + " Điểm cống hiến.",
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

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.delete_mission", EntityType: "sect_mission",
		EntityID:       missionID.String(),
		RequestSummary: fmt.Sprintf("Xóa nhiệm vụ %s của tông môn %s (Hoàn %d quỹ)", sm.Title, sm.SectID, sm.RewardContribution),
		IPAddress:      c.ClientIP(), Success: true,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Đã thu hồi nhiệm vụ tông môn và hoàn lại " + strconv.Itoa(sm.RewardContribution) + " quỹ bang!"})
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
	DurationHours      int    `json:"duration_hours"` // 0 => default 24h, < 0 => no expiry
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

	// Requirements: Rank >= 3 and 30,000 Primeval Stones
	if player.Rank < 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Khai tông lập phái yêu cầu tu vi ít nhất Tam Chuyển (Rank 3)!"})
		return
	}

	cost := 30000.0
	if player.PrimevalStones < cost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không có đủ 30,000 Nguyên Thạch làm chi phí lập phái!"})
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

	auditlog.Logf(db.DB, &playerID, "sect.create", "sectId=%s name=%s", sectID, req.Name)

	// Trigger System Chat Notification on Sect channel
	_ = SendSystemMessage("sect", sql.NullString{String: sectID, Valid: true}, fmt.Sprintf("Chúc mừng đạo hữu %s đã lập phái thành công! Tông môn %s chính thức được thành lập.", player.Username, req.Name))

	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"message": "Khai tông lập phái thành công! Chào mừng Chưởng môn của \"" + req.Name + "\"!",
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
	err := db.DB.QueryRowContext(c.Request.Context(), `SELECT sect_id FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT sm.player_id, p.username, p.avatar_url, sm.member_rank, sm.contribution_points, sm.joined_at, p.rank
		FROM sect_members sm
		JOIN players p ON sm.player_id = p.id
		WHERE sm.sect_id = $1
		ORDER BY 
			CASE sm.member_rank
				WHEN 'CHUONG_MON' THEN 1
				WHEN 'HO_PHAP' THEN 2
				WHEN 'TRUONG_LAO' THEN 3
				WHEN 'CHAN_TRUYEN' THEN 4
				WHEN 'TINH_ANH' THEN 5
				WHEN 'NOI_MON' THEN 6
				ELSE 7
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
		AvatarURL    string    `json:"avatar_url"`
		MemberRank   string    `json:"member_rank"`
		Contribution int64     `json:"contribution_points"`
		JoinedAt     time.Time `json:"joined_at"`
		PlayerRank   int       `json:"player_rank"`
		Power        float64   `json:"power"`
	}

	var members []MemberInfo
	for rows.Next() {
		var mi MemberInfo
		if err := rows.Scan(&mi.PlayerID, &mi.Username, &mi.AvatarURL, &mi.MemberRank, &mi.Contribution, &mi.JoinedAt, &mi.PlayerRank); err == nil {
			if p, err := PlayerSvc.LoadEquipped(db.DB, mi.PlayerID); err == nil {
				mi.Power = computeMemberPower(&p)
			}
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
		"HO_PHAP":     true,
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

	if !canManageSect(callerRank) {
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

	// Hierarchical check: cannot kick a member of equal or higher rank
	if callerRank != "CHUONG_MON" && sectRankValue(callerRank) <= sectRankValue(targetRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu không thể trục xuất thành viên ngang hàng hoặc cấp cao hơn!"})
		return
	}

	var targetUsername string
	_ = tx.QueryRow(`SELECT username FROM players WHERE id = $1`, targetMemberID).Scan(&targetUsername)

	// Delete member
	_, err = tx.Exec(`DELETE FROM sect_members WHERE player_id = $1`, targetMemberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to expel member"})
		return
	}

	// Delete target player's active missions
	_, _ = tx.Exec(`DELETE FROM player_sect_missions WHERE player_id = $1`, targetMemberID)

	// Delete pending promotion requests
	_, _ = tx.Exec(`DELETE FROM sect_promotion_requests WHERE player_id = $1 AND sect_id = $2`, targetMemberID, targetSectID)

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	auditlog.Logf(db.DB, &playerID, "sect.kick", "sectId=%s targetId=%s target=%s", targetSectID, targetMemberID, targetUsername)

	if targetUsername != "" {
		_ = SendSystemMessage("sect", sql.NullString{String: targetSectID, Valid: true}, fmt.Sprintf("Đạo hữu %s đã bị trục xuất khỏi tông môn.", targetUsername))
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

	if !canManageSect(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Trưởng Lão hoặc Chưởng Môn mới có quyền phân phát nhiệm vụ!"})
		return
	}

	// Check if sect has enough funds
	var currentFunds int
	err = tx.QueryRow(`SELECT funds FROM sects WHERE id = $1`, callerSectID).Scan(&currentFunds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	missionCost := req.RewardContribution
	if currentFunds < missionCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quỹ bang không đủ để ban bố nhiệm vụ này! (Cần tối thiểu " + strconv.Itoa(missionCost) + " quỹ bang)"})
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

type StockVaultRequest struct {
	ItemType string `json:"item_type" binding:"required"` // gu, material, recipe
	ItemID   string `json:"item_id" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,min=1"`
}

// stockItemUnitCost computes the sect-funds unit cost for stocking an item and
// enforces the rank gates (Gu: rank 1-5 & <= sect level; material: rank <= sect level).
// Returns (unitCost, httpStatusOK, "") on success; otherwise (0, status, errorMessage).
func stockItemUnitCost(tx *sql.Tx, sectLevel int, itemType, itemID string) (int, int, string) {
	itemType = strings.ToLower(itemType)
	if itemType == "gu" {
		return 0, http.StatusForbidden, "Không thể nhập Cổ trùng vào bảo khố."
	}

	if itemType == "material" {
		// calculate max allowed item level based on sect level (10 sect levels = 1 item level, max 5)
		maxLevel := sectLevel/10 + 1
		if maxLevel > 5 {
			maxLevel = 5
		}
		var matRank, matRarity int
		err := tx.QueryRow(`SELECT COALESCE(rank, 1), COALESCE(rarity, 0) FROM item_dictionary WHERE id = $1`, itemID).Scan(&matRank, &matRarity)
		if err != nil {
			matRank = 1
			matRarity = 0
		}
		effectiveLevel := matRank
		if matRarity > effectiveLevel {
			effectiveLevel = matRarity
		}

		if effectiveLevel >= 6 {
			return 0, http.StatusForbidden, "Không thể nhập Tiên phẩm Linh tài vào bảo khố tông môn (chỉ hỗ trợ phàm phẩm cấp 1-5)."
		}

		if effectiveLevel > maxLevel {
			return 0, http.StatusForbidden, fmt.Sprintf("Tông môn chưa đạt cấp %d để nhập Linh tài hạng %d! (Hiện tại: Cấp %d)", maxLevel, effectiveLevel, sectLevel)
		}

		return effectiveLevel * 50, http.StatusOK, ""
	}

	if itemType == "recipe" {
		var targetGuRank int = 1
		var targetGuRarity int = 0
		err := tx.QueryRow(`
			SELECT COALESCE(gd.rank, 1), COALESCE(gd.rarity, 0)
			FROM refining_recipes rr
			JOIN gu_dictionary gd ON gd.id = rr.target_gu_id AND gd.rank = rr.target_rank
			WHERE ('recipe_' || rr.target_gu_id || '_' || rr.target_rank::text || '_' || rr.variant::text) = $1
		`, itemID).Scan(&targetGuRank, &targetGuRarity)
		if err != nil {
			return 0, http.StatusBadRequest, "Bí phương không tồn tại trong tàng kinh các"
		}

		effectiveLevel := targetGuRank
		if targetGuRarity > effectiveLevel {
			effectiveLevel = targetGuRarity
		}

		if effectiveLevel >= 6 {
			return 0, http.StatusForbidden, "Không thể nhập Tiên cấp Bí phương vào bảo khố tông môn (chỉ hỗ trợ phàm cấp 1-5)."
		}

		// calculate max allowed item level based on sect level (10 sect levels = 1 item level, max 5)
		maxLevel := sectLevel/10 + 1
		if maxLevel > 5 {
			maxLevel = 5
		}
		if effectiveLevel > maxLevel {
			return 0, http.StatusForbidden, fmt.Sprintf("Tông môn chưa đạt cấp %d để nhập Bí phương hạng %d! (Hiện tại: Cấp %d)", maxLevel, effectiveLevel, sectLevel)
		}
		return pricing.RecipeCostByGuRank(effectiveLevel), http.StatusOK, ""
	}

	return 0, http.StatusBadRequest, "Loại vật phẩm không hợp lệ (chỉ hỗ trợ gu, material hoặc recipe)"
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

	if !canManageSect(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Trưởng Lão hoặc Chưởng Môn mới có quyền mua thêm vật phẩm cho bảo khố!"})
		return
	}

	// Fetch sect details
	var s Sect
	err = tx.QueryRow(`
		SELECT id, vault_gu, vault_materials, vault_recipes, funds, level
		FROM sects
		WHERE id = $1
	`, sectID).Scan(&s.ID, &s.VaultGu, &s.VaultMaterials, &s.VaultRecipes, &s.Funds, &s.Level)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sect details: " + err.Error()})
		return
	}

	unitCost, code, msg := stockItemUnitCost(tx, s.Level, req.ItemType, req.ItemID)
	if code != http.StatusOK {
		c.JSON(code, gin.H{"error": msg})
		return
	}
	itemType := strings.ToLower(req.ItemType)

	totalCost := int64(unitCost) * int64(req.Quantity)
	if s.Funds < totalCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Quỹ tông môn không đủ! Cần %d quỹ tông môn", totalCost)})
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

		// Recalculate cost for this Gu using pricing package
		var stockGuRank, stockGuBasePrice int
		_ = tx.QueryRow(`SELECT COALESCE(rank,1), COALESCE(base_price,0) FROM gu_dictionary WHERE id = $1 ORDER BY rank DESC LIMIT 1`, req.ItemID).Scan(&stockGuRank, &stockGuBasePrice)
		if stockGuRank == 0 {
			stockGuRank = 1
		}
		stockGuCost := pricing.VaultCost(pricing.GuBasePrice(stockGuBasePrice, stockGuRank))

		found := false
		for i, v := range vaultGus {
			if v.ID == req.ItemID {
				vaultGus[i].Quantity += req.Quantity
				vaultGus[i].Cost = stockGuCost // refresh cost to current pricing
				vaultGus[i].Hunger = vaultGuMaxHunger(req.ItemID, stockGuRank)
				found = true
				break
			}
		}

		if !found {
			vaultGus = append(vaultGus, SectVaultGuItem{
				ID:       req.ItemID,
				Rank:     stockGuRank,
				Cost:     stockGuCost,
				Quantity: req.Quantity,
				Hunger:   vaultGuMaxHunger(req.ItemID, stockGuRank),
			})
		}

		newVaultGu, _ := json.Marshal(vaultGus)
		_, err = tx.Exec(`UPDATE sects SET vault_gu = $1, funds = $2 WHERE id = $3`, newVaultGu, s.Funds, s.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault: " + err.Error()})
			return
		}
	} else if itemType == "recipe" {
		var vaultRecipes []models.SectVaultRecipe
		if len(s.VaultRecipes) > 0 {
			_ = json.Unmarshal(s.VaultRecipes, &vaultRecipes)
		}
		if vaultRecipes == nil {
			vaultRecipes = []models.SectVaultRecipe{}
		}

		found := false
		for i := range vaultRecipes {
			if vaultRecipes[i].RecipeID == req.ItemID {
				if vaultRecipes[i].CompletionRate < 100.0 {
					vaultRecipes[i].CompletionRate = 100.0
				}
				found = true
				break
			}
		}
		if !found {
			vaultRecipes = append(vaultRecipes, models.SectVaultRecipe{
				RecipeID:       req.ItemID,
				CompletionRate: 100.0,
			})
		}

		newVaultRecipes, _ := json.Marshal(vaultRecipes)
		_, err = tx.Exec(`UPDATE sects SET vault_recipes = $1, funds = $2 WHERE id = $3`, newVaultRecipes, s.Funds, s.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault recipes: " + err.Error()})
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

	// Trigger System Chat Notification on Sect channel
	rankName := "Chưởng Môn"
	if callerRank == "TRUONG_LAO" {
		rankName = "Trưởng Lão"
	} else if callerRank == "HO_PHAP" {
		rankName = "Hộ Pháp"
	}
	var username string
	_ = db.DB.QueryRow(`SELECT username FROM players WHERE id = $1`, playerID).Scan(&username)
	itemName := req.ItemID
	if itemType == "gu" {
		if spec, ok := models.GetGuAny(req.ItemID); ok {
			itemName = spec.Name
		}
	} else if itemType == "material" {
		if spec, ok := models.ItemRegistry[req.ItemID]; ok {
			itemName = spec.Name
		}
	} else if itemType == "recipe" {
		if targetGuID, _, _, err := ParseRecipeID(req.ItemID); err == nil {
			if spec, ok := models.GetGuAny(targetGuID); ok {
				itemName = spec.Name
			}
		}
		itemName = "Bí phương " + itemName
	}
	if username != "" {
		_ = SendSystemMessage("sect", sql.NullString{String: sectID, Valid: true}, fmt.Sprintf("%s %s đã nhập thêm %d %s vào bảo khố tông môn.", rankName, username, req.Quantity, itemName))
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Nhập thêm vật phẩm vào bảo khố thành công!",
		"new_funds": s.Funds,
	})
}

type StockVaultBulkItem struct {
	ItemType string `json:"item_type" binding:"required"` // gu, material, recipe
	ItemID   string `json:"item_id" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,min=1"`
}

type StockVaultBulkRequest struct {
	Items []StockVaultBulkItem `json:"items" binding:"required,min=1"`
}

// StockSectVaultBulk stocks multiple vault items in a single atomic transaction.
func StockSectVaultBulk(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req StockVaultBulkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Danh sách vật phẩm rỗng!"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	var sectID, callerRank string
	err = tx.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if !canManageSect(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Trưởng Lão hoặc Chưởng Môn mới có quyền mua thêm vật phẩm cho bảo khố!"})
		return
	}

	var s Sect
	err = tx.QueryRow(`SELECT id, vault_gu, vault_materials, vault_recipes, funds, level FROM sects WHERE id = $1`, sectID).Scan(&s.ID, &s.VaultGu, &s.VaultMaterials, &s.VaultRecipes, &s.Funds, &s.Level)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sect details: " + err.Error()})
		return
	}

	type stockLine struct {
		itemType string
		itemID   string
		quantity int
		unitCost int
	}
	lines := make([]stockLine, 0, len(req.Items))
	var totalCost int64
	for _, it := range req.Items {
		unitCost, code, msg := stockItemUnitCost(tx, s.Level, it.ItemType, it.ItemID)
		if code != http.StatusOK {
			c.JSON(code, gin.H{"error": msg})
			return
		}
		lines = append(lines, stockLine{strings.ToLower(it.ItemType), it.ItemID, it.Quantity, unitCost})
		totalCost += int64(unitCost) * int64(it.Quantity)
	}

	if s.Funds < totalCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Quỹ tông môn không đủ! Cần %d quỹ tông môn", totalCost)})
		return
	}
	s.Funds -= totalCost

	var vaultGus []SectVaultGuItem
	if len(s.VaultGu) > 0 {
		_ = json.Unmarshal(s.VaultGu, &vaultGus)
	}
	vaultMats := make(map[string]int)
	if len(s.VaultMaterials) > 0 {
		_ = json.Unmarshal(s.VaultMaterials, &vaultMats)
	}
	var vaultRecipes []models.SectVaultRecipe
	if len(s.VaultRecipes) > 0 {
		_ = json.Unmarshal(s.VaultRecipes, &vaultRecipes)
	}
	if vaultRecipes == nil {
		vaultRecipes = []models.SectVaultRecipe{}
	}

	for _, line := range lines {
		if line.itemType == "gu" {
			// Recalculate cost for this Gu using pricing package
			var stockGuRank, stockGuBasePrice int
			_ = tx.QueryRow(`SELECT COALESCE(rank,1), COALESCE(base_price,0) FROM gu_dictionary WHERE id = $1 ORDER BY rank DESC LIMIT 1`, line.itemID).Scan(&stockGuRank, &stockGuBasePrice)
			if stockGuRank == 0 {
				stockGuRank = 1
			}
			stockGuCost := pricing.VaultCost(pricing.GuBasePrice(stockGuBasePrice, stockGuRank))

			found := false
			for i := range vaultGus {
				if vaultGus[i].ID == line.itemID {
					vaultGus[i].Quantity += line.quantity
					vaultGus[i].Cost = stockGuCost
					vaultGus[i].Hunger = vaultGuMaxHunger(line.itemID, stockGuRank)
					found = true
					break
				}
			}
			if !found {
				vaultGus = append(vaultGus, SectVaultGuItem{ID: line.itemID, Rank: stockGuRank, Cost: stockGuCost, Quantity: line.quantity, Hunger: vaultGuMaxHunger(line.itemID, stockGuRank)})
			}
		} else if line.itemType == "recipe" {
			found := false
			for i := range vaultRecipes {
				if vaultRecipes[i].RecipeID == line.itemID {
					if vaultRecipes[i].CompletionRate < 100.0 {
						vaultRecipes[i].CompletionRate = 100.0
					}
					found = true
					break
				}
			}
			if !found {
				vaultRecipes = append(vaultRecipes, models.SectVaultRecipe{
					RecipeID:       line.itemID,
					CompletionRate: 100.0,
				})
			}
		} else {
			vaultMats[line.itemID] += line.quantity
		}
	}

	newVaultGu, _ := json.Marshal(vaultGus)
	newVaultMats, _ := json.Marshal(vaultMats)
	newVaultRecipes, _ := json.Marshal(vaultRecipes)
	_, err = tx.Exec(`UPDATE sects SET vault_gu = $1, vault_materials = $2, vault_recipes = $3, funds = $4 WHERE id = $5`, newVaultGu, newVaultMats, newVaultRecipes, s.Funds, s.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	// Trigger System Chat Notification on Sect channel
	rankName := "Chưởng Môn"
	if callerRank == "TRUONG_LAO" {
		rankName = "Trưởng Lão"
	} else if callerRank == "HO_PHAP" {
		rankName = "Hộ Pháp"
	}
	var username string
	_ = db.DB.QueryRow(`SELECT username FROM players WHERE id = $1`, playerID).Scan(&username)
	if username != "" {
		_ = SendSystemMessage("sect", sql.NullString{String: sectID, Valid: true}, fmt.Sprintf("%s %s đã nhập thêm %d mặt hàng vào bảo khố tông môn.", rankName, username, len(lines)))
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

	if !canManageSect(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Trưởng Lão hoặc Chưởng Môn mới có quyền tăng cấp tông môn!"})
		return
	}

	// Fetch level and funds
	var level int
	var funds int64
	err = tx.QueryRow(`SELECT level, funds FROM sects WHERE id = $1`, sectID).Scan(&level, &funds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sect details"})
		return
	}

	cost := int64(level) * int64(level) * 5000
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
	var sectLevel int
	err = tx.QueryRow(`SELECT sect_id, member_rank, (SELECT level FROM sects WHERE id = sect_members.sect_id) FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank, &sectLevel)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if !canManageSect(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Trưởng Lão hoặc Chưởng Môn mới có quyền thay đổi thiết lập này!"})
		return
	}

	if req.RequireApproval && sectLevel < 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tông môn phải đạt cấp 10 mới có thể bật duyệt thành viên!"})
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

type ToggleSectPromotionApprovalRequest struct {
	RequirePromotionApproval bool `json:"require_promotion_approval"`
}

// ToggleSectPromotionApproval toggles whether all rank-ups require leadership approval.
func ToggleSectPromotionApproval(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req ToggleSectPromotionApprovalRequest
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

	if !canManageSect(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Trưởng Lão hoặc Chưởng Môn mới có quyền thay đổi thiết lập này!"})
		return
	}

	_, err = tx.Exec(`UPDATE sects SET require_promotion_approval = $1 WHERE id = $2`, req.RequirePromotionApproval, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect setting"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	msg := "Đã bật duyệt lên cấp: mọi thăng cấp đều cần Trưởng Lão/Chưởng Môn duyệt!"
	if !req.RequirePromotionApproval {
		msg = "Đã tắt duyệt lên cấp: đệ tử tự thăng cấp theo tiêu chuẩn (trừ Chân Truyền/Trưởng Lão)."
	}

	c.JSON(http.StatusOK, gin.H{
		"message":                    msg,
		"require_promotion_approval": req.RequirePromotionApproval,
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
	err := db.DB.QueryRowContext(c.Request.Context(), `SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if !canManageSect(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Không đủ quyền hạn!"})
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT r.id, r.sect_id, r.player_id, r.status, r.created_at, p.username, p.avatar_url, p.rank
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
		if err := rows.Scan(&r.ID, &r.SectID, &r.PlayerID, &r.Status, &r.CreatedAt, &r.Username, &r.AvatarURL, &r.PlayerRank); err == nil {
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

	if !canManageSect(callerRank) {
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
		var username string
		_ = db.DB.QueryRow(`SELECT username FROM players WHERE id = $1`, targetPlayerID).Scan(&username)
		if username != "" {
			_ = SendSystemMessage("sect", sql.NullString{String: sectID, Valid: true}, fmt.Sprintf("Chào mừng đạo hữu %s đã gia nhập tông môn!", username))
		}
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
	err := db.DB.QueryRowContext(c.Request.Context(), `SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if !canManageSect(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Không đủ quyền hạn!"})
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT r.id, r.sect_id, r.player_id, r.current_rank, r.target_rank, r.status, r.created_at, p.username, p.avatar_url, p.rank
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
		if err := rows.Scan(&r.ID, &r.SectID, &r.PlayerID, &r.CurrentRank, &r.TargetRank, &r.Status, &r.CreatedAt, &r.Username, &r.AvatarURL, &r.PlayerRank); err == nil {
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

	if !canManageSect(callerRank) {
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

	if sectRankValue(callerRank) <= sectRankValue(targetRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu không thể duyệt thăng chức lên cấp bậc ngang hoặc cao hơn mình!"})
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
		case "HO_PHAP":
			costPoints = 30000
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
