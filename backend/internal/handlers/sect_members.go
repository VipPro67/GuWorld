package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

type CreateSectRequest struct {
	Name        string `json:"name" binding:"required,min=3,max=50"`
	Description string `json:"description" binding:"required,max=500"`
	Region      string `json:"region"`
}

type SetMemberRankRequest struct {
	Rank string `json:"rank" binding:"required"`
}

type CreateCustomMissionRequest struct {
	Title              string `json:"title" binding:"required"`
	Description        string `json:"description" binding:"required"`
	MissionType        string `json:"mission_type" binding:"required"` // GATHER, DONATE
	TargetID           string `json:"target_id" binding:"required"`
	TargetAmount       int64  `json:"target_amount" binding:"required,min=1"`
	RewardContribution int64  `json:"reward_contribution" binding:"required,min=1"`
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

	cost := int64(30000)
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
	res, err := tx.Exec(`UPDATE players SET primeval_stones = primeval_stones - $1 WHERE id = $2 AND primeval_stones >= $1`, cost, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Deduct primeval stones failed"})
		return
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Nguyên Thạch để lập Tông môn"})
		return
	}

	// Check location: player cannot create sect in HUB or SAFE_ZONE
	currentRegion := player.CurrentRegion
	if currentRegion == "" {
		currentRegion = req.Region
	}
	var zoneType string
	err = tx.QueryRow(`SELECT COALESCE(zone_type, 'KHU_VUC') FROM regions WHERE id = $1`, currentRegion).Scan(&zoneType)
	if err == nil {
		if zoneType == "HUB" || zoneType == "SAFE_ZONE" || currentRegion == "hub_nam_cuong" || currentRegion == "Lobby" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể khai tông lập phái trong An Toàn Khu / Trấn (HUB/SAFE_ZONE)! Đạo hữu hãy di chuyển ra dã ngoại."})
			return
		}
	}

	// Insert new sect at player's current location (region, map_x, map_y)
	_, err = tx.Exec(`
		INSERT INTO sects (id, name, description, region, map_x, map_y, vault_gu, vault_materials, vault_recipes)
		VALUES ($1, $2, $3, $4, $5, $6, '[]'::jsonb, '{}'::jsonb, '[]'::jsonb)
	`, sectID, req.Name, req.Description, currentRegion, player.X, player.Y)
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

	// Trigger System Chat Notification on Sect channel
	_ = SendSystemMessage("sect", sql.NullString{String: sectID, Valid: true}, fmt.Sprintf("Chúc mừng đạo hữu %s đã lập phái thành công! Tông môn %s chính thức được thành lập.", player.Username, req.Name))

	respondActionOnly(c, http.StatusOK, player, gin.H{
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

	if targetUsername != "" {
		_ = SendSystemMessage("sect", sql.NullString{String: targetSectID, Valid: true}, fmt.Sprintf("Đạo hữu %s đã bị trục xuất khỏi tông môn.", targetUsername))
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã trục xuất thành viên ra khỏi tông môn!"})
}
