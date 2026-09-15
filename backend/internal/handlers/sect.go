package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
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
	LastVaultDecayAt         *time.Time      `json:"last_vault_decay_at,omitempty"`
	MapX                     int             `json:"map_x"`
	MapY                     int             `json:"map_y"`
	ShieldUntil              *time.Time      `json:"shield_until,omitempty"`
	Buildings                json.RawMessage `json:"buildings,omitempty"`
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
	IsDefending        bool      `json:"is_defending"`
}

type SectMission struct {
	ID                 uuid.UUID  `json:"id"`
	SectID             string     `json:"sect_id"`
	Title              string     `json:"title"`
	Description        string     `json:"description"`
	MissionType        string     `json:"mission_type"` // GATHER, COMBAT, REFINE, DONATE
	TargetID           string     `json:"target_id"`
	TargetAmount       int64      `json:"target_amount"`
	RewardContribution int64      `json:"reward_contribution"`
	RequiredMemberRank string     `json:"required_member_rank"`
	ExpiresAt          *time.Time `json:"expires_at"`
}

type PlayerSectMission struct {
	ID              uuid.UUID  `json:"id"`
	PlayerID        uuid.UUID  `json:"player_id"`
	MissionID       uuid.UUID  `json:"mission_id"`
	CurrentProgress int64      `json:"current_progress"`
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
	ID         string  `json:"id"`
	InstanceID string  `json:"instance_id,omitempty"`
	Rank       int     `json:"rank,omitempty"`
	Cost       int64   `json:"cost"`
	Quantity   int64   `json:"quantity"`
	Hunger     float64 `json:"hunger,omitempty"`
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

// createVaultWithdrawnGu creates a new player GuInstance withdrawn from the sect vault.
// As per game rules, Gus with hunger rate > 0 start with hunger = 1.
// Gus with hunger rate <= 0 (e.g. consumables) keep full max hunger.
func createVaultWithdrawnGu(guID string, rank int, guSpec models.Gu) models.GuInstance {
	initialHunger := 1
	if guSpec.HungerRate <= 0 {
		initialHunger = guSpec.MaxHunger
		if initialHunger <= 0 {
			initialHunger = 100
		}
	}
	return models.GuInstance{
		InstanceID: uuid.New().String(),
		GuID:       guID,
		Rank:       rank,
		Hunger:     initialHunger,
		Status:     "ACTIVE",
		Equipped:   false,
	}
}

type DonateStonesRequest struct {
	Amount int64 `json:"amount" binding:"required,min=1"`
}

type DonateMaterialsRequest struct {
	MaterialID string `json:"material_id" binding:"required"`
	Amount     int64  `json:"amount" binding:"required,min=1"`
}

type DonateGuRequest struct {
	GuInstanceID string `json:"gu_instance_id" binding:"required"`
}

type DonateRecipeRequest struct {
	RecipeID string `json:"recipe_id" binding:"required"`
}

type ExchangeVaultItemRequest struct {
	ItemType string `json:"item_type" binding:"required"`
	ItemID   string `json:"item_id" binding:"required"`
	Rank     int    `json:"rank,omitempty"`
	Quantity int64  `json:"quantity" binding:"min=1"`
}

type VaultBuyRequest = ExchangeVaultItemRequest

// GetSects lists all available sects
func GetSects(c *gin.Context) {
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT id, name, description, region, vault_gu, vault_materials, vault_recipes, funds, level, require_approval, require_promotion_approval, map_x, map_y, shield_until, buildings FROM sects`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sects: " + err.Error()})
		return
	}
	defer rows.Close()

	var sects []Sect
	for rows.Next() {
		var s Sect
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Region, &s.VaultGu, &s.VaultMaterials, &s.VaultRecipes, &s.Funds, &s.Level, &s.RequireApproval, &s.RequirePromotionApproval, &s.MapX, &s.MapY, &s.ShieldUntil, &s.Buildings); err == nil {
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

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	var username string
	_ = db.DB.QueryRow(`SELECT username FROM players WHERE id = $1`, playerID).Scan(&username)
	if username != "" {
		_ = SendSystemMessage("sect", sql.NullString{String: sectID, Valid: true}, fmt.Sprintf("Chào mừng đạo hữu %s đã gia nhập tông môn!", username))
	}

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
		SELECT id, sect_id, player_id, member_rank, contribution_points, master_id, joined_at, is_defending
		FROM sect_members
		WHERE player_id = $1
	`, playerID).Scan(&m.ID, &m.SectID, &m.PlayerID, &m.MemberRank, &m.ContributionPoints, &m.MasterID, &m.JoinedAt, &m.IsDefending)
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
		SELECT id, name, description, region, vault_gu, vault_materials, vault_recipes, funds, level, require_approval, require_promotion_approval, map_x, map_y, shield_until, buildings
		FROM sects
		WHERE id = $1
	`, m.SectID).Scan(&s.ID, &s.Name, &s.Description, &s.Region, &s.VaultGu, &s.VaultMaterials, &s.VaultRecipes, &s.Funds, &s.Level, &s.RequireApproval, &s.RequirePromotionApproval, &s.MapX, &s.MapY, &s.ShieldUntil, &s.Buildings)
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

// BuyFromSectVault processes vault purchases
