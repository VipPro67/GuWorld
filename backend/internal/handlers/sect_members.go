package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
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
	if player.PrimevalStones < float64(cost) {
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

	// Verify territory does not overlap with any existing sect (initial radius = 5 + 1 = 6)
	hasOverlap, conflictingSect, errOverlap := CheckTerritoryOverlap(tx, currentRegion, player.X, player.Y, 6, "")
	if errOverlap != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kiểm tra lãnh địa thất bại: " + errOverlap.Error()})
		return
	}
	if hasOverlap {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Vị trí này quá gần tông môn \"%s\"! Lãnh địa sơn môn bị chồng lấn, không thể lập phái tại đây.", conflictingSect)})
		return
	}

	// Insert new sect at player's current location (region, map_x, map_y)
	_, err = tx.Exec(`
		INSERT INTO sects (id, name, description, region, map_x, map_y)
		VALUES ($1, $2, $3, $4, $5, $6)
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

	var sectID, sectRegion string
	var sectX, sectY, mainHallLevel int
	err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT sm.sect_id, s.region, s.map_x, s.map_y,
		       COALESCE((s.buildings->>'main_hall')::int, 1)
		FROM sect_members sm
		JOIN sects s ON s.id = sm.sect_id
		WHERE sm.player_id = $1
	`, playerID).Scan(&sectID, &sectRegion, &sectX, &sectY, &mainHallLevel)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT sm.player_id, p.username, p.avatar_url, sm.member_rank, sm.contribution_points, sm.joined_at, p.rank,
		       p.last_activity, p.last_mana_update
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
	type MemberInfo struct {
		PlayerID     uuid.UUID  `json:"player_id"`
		Username     string     `json:"username"`
		AvatarURL    string     `json:"avatar_url"`
		MemberRank   string     `json:"member_rank"`
		Contribution int64      `json:"contribution_points"`
		JoinedAt     time.Time  `json:"joined_at"`
		PlayerRank   int        `json:"player_rank"`
		Power        float64    `json:"power"`
		LastActivity *time.Time `json:"last_activity,omitempty"`
		IsOnline     bool       `json:"is_online"`
	}

	var members []MemberInfo
	now := time.Now()
	memberIDs := make([]string, 0)
	for rows.Next() {
		var mi MemberInfo
		var la, lmu *time.Time
		if err := rows.Scan(&mi.PlayerID, &mi.Username, &mi.AvatarURL, &mi.MemberRank, &mi.Contribution, &mi.JoinedAt, &mi.PlayerRank, &la, &lmu); err == nil {
			mi.LastActivity = la
			if la != nil && now.Sub(*la) <= 5*time.Minute {
				mi.IsOnline = true
			} else if lmu != nil && now.Sub(*lmu) <= 5*time.Minute {
				mi.IsOnline = true
			}
			memberIDs = append(memberIDs, mi.PlayerID.String())
			members = append(members, mi)
		}
	}
	_ = rows.Close()
	if len(memberIDs) == 0 {
		c.JSON(http.StatusOK, members)
		return
	}

	// Load only the inputs needed by CalculatePlayerStats in one batch. The
	// generic PlayerSvc.LoadEquipped path also reads materials and combat state,
	// neither of which contributes to sect member power.
	type memberPowerInput struct {
		player    models.Player
		territory SectTerritoryInfo
	}
	powerInputs := make(map[uuid.UUID]memberPowerInput, len(memberIDs))
	statRows, statErr := db.DB.QueryContext(c.Request.Context(), `
		SELECT p.id, p.rank, p.sub_rank, p.base_hp, p.hp, p.growth_data,
		       COALESCE(p.path_masteries, '{}'::jsonb), COALESCE(p.statuses, '[]'::jsonb),
		       COALESCE(p.race, ''), COALESCE(p.constitution, ''), COALESCE(p.body_mods, '[]'::jsonb),
		       COALESCE(p.current_region, ''), p.x, p.y,
		       CASE WHEN p.rank >= 6 THEN ia.dao_marks::text ELSE '{"strength":0}' END,
		       EXISTS (SELECT 1 FROM sect_members sm WHERE sm.sect_id = $2 AND sm.player_id = p.id)
		FROM players p
		LEFT JOIN immortal_apertures ia ON ia.player_id = p.id
		WHERE p.id = ANY($1::uuid[])
	`, pq.Array(memberIDs), sectID)
	if statErr != nil {
		logger.S.Warnf("sect members: batch load power inputs failed: %v", statErr)
	} else {
		for statRows.Next() {
			var player models.Player
			var daoMarksJSON sql.NullString
			var stillMember bool
			if err := statRows.Scan(
				&player.ID, &player.Rank, &player.SubRank, &player.BaseHP, &player.HP,
				&player.GrowthData, &player.PathMasteries, &player.Statuses,
				&player.Race, &player.Constitution, &player.BodyMods,
				&player.CurrentRegion, &player.X, &player.Y, &daoMarksJSON, &stillMember,
			); err != nil {
				continue
			}
			player.DaoMarks = models.DaoMarksMap{"strength": 0}
			if player.Rank >= 6 && daoMarksJSON.Valid {
				var marks models.DaoMarksMap
				if err := marks.Scan(daoMarksJSON.String); err == nil {
					player.DaoMarks = marks
				}
			}
			territory := SectTerritoryInfo{MainHallLvl: mainHallLevel}
			if stillMember && player.CurrentRegion == sectRegion {
				territory.InTerritory = IsInsideSquareTerritory(player.X, player.Y, sectX, sectY, 5+mainHallLevel)
			}
			powerInputs[player.ID] = memberPowerInput{player: player, territory: territory}
		}
		if err := statRows.Err(); err != nil {
			logger.S.Warnf("sect members: batch load power inputs incomplete: %v", err)
		}
		_ = statRows.Close()
	}

	// Power only uses equipped ACTIVE Gus. Loading these together avoids a
	// per-player inventory query; a failed Gu query mirrors LoadEquipped's
	// existing behavior of returning power from the remaining player stats.
	guRows, guErr := db.DB.QueryContext(c.Request.Context(), `
		SELECT player_id::text, gu_id, gu_rank
		FROM player_gu
		WHERE player_id = ANY($1::uuid[])
		  AND location <> 'ACTOR' AND equipped = TRUE AND state = 'ACTIVE'
		ORDER BY player_id, created_at, id
	`, pq.Array(memberIDs))
	if guErr != nil {
		logger.S.Warnf("sect members: batch load equipped Gus failed: %v", guErr)
	} else {
		for guRows.Next() {
			var playerID string
			var gu models.GuInstance
			if err := guRows.Scan(&playerID, &gu.GuID, &gu.Rank); err != nil {
				continue
			}
			gu.Status = "ACTIVE"
			gu.Equipped = true
			id, err := uuid.Parse(playerID)
			if err != nil {
				continue
			}
			input, ok := powerInputs[id]
			if !ok {
				continue
			}
			input.player.GuInventory = append(input.player.GuInventory, gu)
			powerInputs[id] = input
		}
		if err := guRows.Err(); err != nil {
			logger.S.Warnf("sect members: batch load equipped Gus incomplete: %v", err)
		}
		_ = guRows.Close()
	}

	for i := range members {
		if input, ok := powerInputs[members[i].PlayerID]; ok {
			members[i].Power = computeMemberPowerWithTerritory(&input.player, input.territory)
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

// ClaimSectLeadership allows a qualified senior member to succeed as leader
// when the current CHUONG_MON has been inactive for 5+ days or is missing.
func ClaimSectLeadership(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể khởi tạo giao dịch"})
		return
	}
	defer tx.Rollback()

	// 1. Get caller's membership
	var callerSectID string
	var callerRank string
	var callerContrib int64
	var callerUsername string
	err = tx.QueryRow(`
		SELECT sm.sect_id, sm.member_rank, sm.contribution_points, p.username
		FROM sect_members sm
		JOIN players p ON sm.player_id = p.id
		WHERE sm.player_id = $1
		FOR UPDATE OF sm
	`, playerID).Scan(&callerSectID, &callerRank, &callerContrib, &callerUsername)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa gia nhập tông môn nào!"})
		return
	}

	if callerRank == "CHUONG_MON" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu hiện đang là Chưởng Môn rồi, không cần tiếp quản!"})
		return
	}

	// 2. Check current CHUONG_MON status
	type LeaderRow struct {
		PlayerID     uuid.UUID
		Username     string
		LastActivity *time.Time
		LastMana     *time.Time
	}
	var leaders []LeaderRow
	rows, err := tx.Query(`
		SELECT sm.player_id, p.username, p.last_activity, p.last_mana_update
		FROM sect_members sm
		JOIN players p ON sm.player_id = p.id
		WHERE sm.sect_id = $1 AND sm.member_rank = 'CHUONG_MON'
		FOR UPDATE OF sm
	`, callerSectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn Chưởng Môn: " + err.Error()})
		return
	}
	for rows.Next() {
		var lr LeaderRow
		if err := rows.Scan(&lr.PlayerID, &lr.Username, &lr.LastActivity, &lr.LastMana); err == nil {
			leaders = append(leaders, lr)
		}
	}
	rows.Close()

	now := time.Now()
	inactiveThreshold := 5 * 24 * time.Hour

	if len(leaders) > 0 {
		leader := leaders[0]
		isInactive := true

		if leader.LastActivity != nil && now.Sub(*leader.LastActivity) < inactiveThreshold {
			isInactive = false
		}
		if leader.LastMana != nil && now.Sub(*leader.LastMana) < inactiveThreshold {
			isInactive = false
		}

		if !isInactive {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Chưởng Môn %s vẫn đang tại vị và hoạt động trong vòng 5 ngày qua! Không thể tiếp quản vị trí.", leader.Username),
			})
			return
		}
	}

	// 3. Find top candidate among non-leaders
	// Priority: Rank Seniority DESC (HO_PHAP > TRUONG_LAO > CHAN_TRUYEN > TINH_ANH > NOI_MON > NGOAI_MON)
	// Then: contribution_points DESC
	// Then: joined_at ASC
	type Candidate struct {
		PlayerID     uuid.UUID
		Username     string
		MemberRank   string
		Contribution int64
		JoinedAt     time.Time
	}

	candidateRows, err := tx.Query(`
		SELECT sm.player_id, p.username, sm.member_rank, sm.contribution_points, sm.joined_at
		FROM sect_members sm
		JOIN players p ON sm.player_id = p.id
		WHERE sm.sect_id = $1 AND sm.member_rank != 'CHUONG_MON'
		ORDER BY 
			CASE sm.member_rank
				WHEN 'HO_PHAP' THEN 1
				WHEN 'TRUONG_LAO' THEN 2
				WHEN 'CHAN_TRUYEN' THEN 3
				WHEN 'TINH_ANH' THEN 4
				WHEN 'NOI_MON' THEN 5
				WHEN 'NGOAI_MON' THEN 6
				ELSE 7
			END ASC,
			sm.contribution_points DESC,
			sm.joined_at ASC
		FOR UPDATE OF sm
	`, callerSectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn danh sách kế vị: " + err.Error()})
		return
	}

	var candidates []Candidate
	for candidateRows.Next() {
		var cand Candidate
		if err := candidateRows.Scan(&cand.PlayerID, &cand.Username, &cand.MemberRank, &cand.Contribution, &cand.JoinedAt); err == nil {
			candidates = append(candidates, cand)
		}
	}
	candidateRows.Close()

	if len(candidates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy đệ tử nào đủ điều kiện kế vị!"})
		return
	}

	topCandidate := candidates[0]
	if topCandidate.PlayerID != playerID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": fmt.Sprintf("Đạo hữu chưa đủ tư cách kế vị! Quyền ưu tiên cao nhất hiện tại thuộc về %s (%s, %d Điểm cống hiến).",
				topCandidate.Username, topCandidate.MemberRank, topCandidate.Contribution),
		})
		return
	}

	// 4. Perform succession
	// Demote existing leaders to TRUONG_LAO
	for _, oldLeader := range leaders {
		_, err = tx.Exec(`UPDATE sect_members SET member_rank = 'TRUONG_LAO' WHERE player_id = $1`, oldLeader.PlayerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi giáng chức Chưởng Môn cũ: " + err.Error()})
			return
		}
	}

	// Promote caller to CHUONG_MON
	_, err = tx.Exec(`UPDATE sect_members SET member_rank = 'CHUONG_MON' WHERE player_id = $1 AND sect_id = $2`, playerID, callerSectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tấn phong Chưởng Môn: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Hoàn tất giao dịch thất bại: " + err.Error()})
		return
	}

	_ = cache.Delete(sectsListCacheKey)

	// 5. Broadcast notification to sect chat
	var oldLeaderNote string
	if len(leaders) > 0 {
		oldLeaderNote = fmt.Sprintf("Do tiền nhiệm Chưởng Môn %s vắng bóng lâu ngày, ", leaders[0].Username)
	} else {
		oldLeaderNote = "Trước tình thế sơn môn vô chủ, "
	}
	chatMsg := fmt.Sprintf("[Sơn Môn] %sđạo hữu %s (%s) đã chính thức kế vị, tiếp quản chức vụ Chưởng Môn tông môn!",
		oldLeaderNote, callerUsername, callerRank)
	_ = SendSystemMessage("sect", sql.NullString{String: callerSectID, Valid: true}, chatMsg)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Đạo hữu đã tiếp quản chức vụ Chưởng Môn thành công!",
		"new_rank": "CHUONG_MON",
	})
}
