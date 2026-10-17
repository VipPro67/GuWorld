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
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

// -------- Admin: Sect Management --------

func AdminListSects(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT s.id, s.name, s.description, s.region, s.level, s.funds, s.require_approval, s.require_promotion_approval,
			COALESCE(m.member_count, 0) AS member_count,
			l.leader_name
		FROM sects s
		LEFT JOIN (
			SELECT sect_id, COUNT(*) AS member_count FROM sect_members GROUP BY sect_id
		) m ON m.sect_id = s.id
		LEFT JOIN (
			SELECT sm.sect_id, p.username AS leader_name
			FROM sect_members sm
			JOIN players p ON p.id = sm.player_id
			WHERE sm.member_rank = 'CHUONG_MON'
		) l ON l.sect_id = s.id
		ORDER BY s.name
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách tông môn"})
		return
	}
	defer rows.Close()

	type SectInfo struct {
		ID                       string `json:"id"`
		Name                     string `json:"name"`
		Description              string `json:"description"`
		Region                   string `json:"region"`
		Level                    int    `json:"level"`
		Funds                    int64  `json:"funds"`
		RequireApproval          bool   `json:"require_approval"`
		RequirePromotionApproval bool   `json:"require_promotion_approval"`
		MemberCount              int    `json:"member_count"`
		LeaderName               string `json:"leader_name,omitempty"`
	}

	var sects []SectInfo
	for rows.Next() {
		var s SectInfo
		var leaderName *string
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Region, &s.Level, &s.Funds, &s.RequireApproval, &s.RequirePromotionApproval, &s.MemberCount, &leaderName); err != nil {
			continue
		}
		if leaderName != nil {
			s.LeaderName = *leaderName
		}
		sects = append(sects, s)
	}

	c.JSON(http.StatusOK, sects)
}

func AdminGetSectDetail(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	sectID := c.Param("id")

	type sectVault struct {
		ID                       string          `json:"id"`
		Name                     string          `json:"name"`
		Description              string          `json:"description"`
		Region                   string          `json:"region"`
		Level                    int             `json:"level"`
		Funds                    int64           `json:"funds"`
		RequireApproval          bool            `json:"require_approval"`
		RequirePromotionApproval bool            `json:"require_promotion_approval"`
		VaultGu                  json.RawMessage `json:"vault_gu"`
		VaultMaterials           json.RawMessage `json:"vault_materials"`
		VaultRecipes             json.RawMessage `json:"vault_recipes"`
	}
	var s sectVault
	err := db.DB.QueryRowContext(c.Request.Context(), `SELECT id, name, description, region, level, funds, require_approval, require_promotion_approval FROM sects WHERE id = $1`, sectID).Scan(&s.ID, &s.Name, &s.Description, &s.Region, &s.Level, &s.Funds, &s.RequireApproval, &s.RequirePromotionApproval)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tông môn"})
		return
	}
	s.VaultGu, s.VaultMaterials, s.VaultRecipes, _ = LoadSectVaultJSON(c.Request.Context(), db.DB, s.ID)

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT sm.player_id, p.username, sm.member_rank, sm.contribution_points, sm.joined_at, p.rank
		FROM sect_members sm
		JOIN players p ON p.id = sm.player_id
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải thành viên"})
		return
	}
	defer rows.Close()

	type MemberInfo struct {
		PlayerID     uuid.UUID `json:"player_id"`
		Username     string    `json:"username"`
		MemberRank   string    `json:"member_rank"`
		Contribution int64     `json:"contribution_points"`
		JoinedAt     time.Time `json:"joined_at"`
		PlayerRank   int       `json:"player_rank"`
	}

	var members []MemberInfo
	for rows.Next() {
		var m MemberInfo
		if err := rows.Scan(&m.PlayerID, &m.Username, &m.MemberRank, &m.Contribution, &m.JoinedAt, &m.PlayerRank); err == nil {
			members = append(members, m)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"sect":    s,
		"members": members,
	})
}

type AdminSectUpdateRequest struct {
	Name                     *string `json:"name"`
	Description              *string `json:"description"`
	Region                   *string `json:"region"`
	Level                    *int    `json:"level"`
	Funds                    *int64  `json:"funds"`
	RequireApproval          *bool   `json:"require_approval"`
	RequirePromotionApproval *bool   `json:"require_promotion_approval"`
}

func AdminUpdateSect(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	sectID := c.Param("id")

	var req AdminSectUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	sets := []string{}
	args := []interface{}{}
	idx := 1

	if req.Name != nil {
		sets = append(sets, "name=$"+strconv.Itoa(idx))
		args = append(args, *req.Name)
		idx++
	}
	if req.Description != nil {
		sets = append(sets, "description=$"+strconv.Itoa(idx))
		args = append(args, *req.Description)
		idx++
	}
	if req.Region != nil {
		sets = append(sets, "region=$"+strconv.Itoa(idx))
		args = append(args, *req.Region)
		idx++
	}
	if req.Level != nil {
		sets = append(sets, "level=$"+strconv.Itoa(idx))
		args = append(args, *req.Level)
		idx++
	}
	if req.Funds != nil {
		sets = append(sets, "funds=$"+strconv.Itoa(idx))
		args = append(args, *req.Funds)
		idx++
	}
	if req.RequireApproval != nil {
		sets = append(sets, "require_approval=$"+strconv.Itoa(idx))
		args = append(args, *req.RequireApproval)
		idx++
	}
	if req.RequirePromotionApproval != nil {
		sets = append(sets, "require_promotion_approval=$"+strconv.Itoa(idx))
		args = append(args, *req.RequirePromotionApproval)
		idx++
	}

	if len(sets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có dữ liệu cập nhật"})
		return
	}

	args = append(args, sectID)
	query := `UPDATE sects SET ` + strings.Join(sets, ", ") + ` WHERE id=$` + strconv.Itoa(idx)

	if _, err := db.DB.Exec(query, args...); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật tông môn"})
		return
	}

	var sectName string
	_ = db.DB.QueryRow(`SELECT name FROM sects WHERE id = $1`, sectID).Scan(&sectName)
	_ = SendSystemMessage("world", sql.NullString{}, fmt.Sprintf("Admin đã cập nhật thông tin của tông môn %s: %s", sectName, formatChangedFields(sets, args)))

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.update_sect",
		EntityType: "sect", EntityID: sectID,
		RequestSummary: fmt.Sprintf("sets=%d", len(sets)),
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật tông môn thành công"})
}

func AdminSetSectLeader(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	sectID := c.Param("id")
	memberIDStr := c.Param("memberId")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID thành viên không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	// Verify member belongs to the sect
	var currentRank string
	err = tx.QueryRow(`SELECT member_rank FROM sect_members WHERE player_id = $1 AND sect_id = $2`, memberID, sectID).Scan(&currentRank)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Thành viên không thuộc tông môn này"})
		return
	}

	// Demote all existing leaders
	_, err = tx.Exec(`UPDATE sect_members SET member_rank = 'TRUONG_LAO' WHERE sect_id = $1 AND member_rank = 'CHUONG_MON'`, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thay đổi chức vụ"})
		return
	}

	// Set new leader
	_, err = tx.Exec(`UPDATE sect_members SET member_rank = 'CHUONG_MON' WHERE player_id = $1 AND sect_id = $2`, memberID, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể bổ nhiệm tân Chưởng môn"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.set_sect_leader",
		EntityType: "sect", EntityID: sectID,
		RequestSummary: fmt.Sprintf("memberId=%s", memberIDStr),
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã bổ nhiệm tân Chưởng môn thành công"})
}

func AdminKickSectMember(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	sectID := c.Param("id")
	memberIDStr := c.Param("memberId")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID thành viên không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	var currentRank string
	err = tx.QueryRow(`SELECT member_rank FROM sect_members WHERE player_id = $1 AND sect_id = $2`, memberID, sectID).Scan(&currentRank)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Thành viên không thuộc tông môn này"})
		return
	}

	// Delete active missions for this member
	_, _ = tx.Exec(`DELETE FROM player_sect_missions WHERE player_id = $1 AND mission_id IN (SELECT id FROM sect_missions WHERE sect_id = $2)`, memberID, sectID)

	// Delete member
	_, err = tx.Exec(`DELETE FROM sect_members WHERE player_id = $1 AND sect_id = $2`, memberID, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa thành viên"})
		return
	}

	tx.Commit()
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.kick_sect_member",
		EntityType: "sect", EntityID: sectID,
		RequestSummary: fmt.Sprintf("memberId=%s", memberIDStr),
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa thành viên khỏi tông môn"})
}

type AdminSectAddMemberRequest struct {
	PlayerID   string `json:"player_id"`
	MemberRank string `json:"member_rank"`
}

func AdminAddToSect(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	sectID := c.Param("id")

	var req AdminSectAddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	playerUUID, err := uuid.Parse(req.PlayerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	if req.MemberRank == "" {
		req.MemberRank = "NGOAI_MON"
	}
	validRanks := map[string]bool{"NGOAI_MON": true, "NOI_MON": true, "TINH_ANH": true, "CHAN_TRUYEN": true, "TRUONG_LAO": true, "HO_PHAP": true, "CHUONG_MON": true}
	if !validRanks[req.MemberRank] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chức vụ không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	// Check player exists
	var username string
	err = tx.QueryRow(`SELECT username FROM players WHERE id = $1`, playerUUID).Scan(&username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}

	// Check if already in a sect
	var existingSectID string
	err = tx.QueryRow(`SELECT sect_id FROM sect_members WHERE player_id = $1`, playerUUID).Scan(&existingSectID)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Người chơi đã ở trong tông môn " + existingSectID})
		return
	}

	// Verify sect exists
	var sectExists bool
	err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM sects WHERE id = $1)`, sectID).Scan(&sectExists)
	if err != nil || !sectExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tông môn"})
		return
	}

	// If being added as leader, demote existing leader first
	if req.MemberRank == "CHUONG_MON" {
		_, _ = tx.Exec(`UPDATE sect_members SET member_rank = 'TRUONG_LAO' WHERE sect_id = $1 AND member_rank = 'CHUONG_MON'`, sectID)
	}

	_, err = tx.Exec(`INSERT INTO sect_members (sect_id, player_id, member_rank, contribution_points) VALUES ($1, $2, $3, 0)`, sectID, playerUUID, req.MemberRank)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thêm thành viên"})
		return
	}

	tx.Commit()
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.add_to_sect",
		EntityType: "sect", EntityID: sectID,
		RequestSummary: fmt.Sprintf("player=%s rank=%s", req.PlayerID, req.MemberRank),
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã thêm " + username + " vào tông môn thành công"})
}

// -------- Admin: Sect Vault --------

type AdminStockVaultRequest struct {
	ItemType string `json:"item_type"` // gu, material, recipe
	ItemID   string `json:"item_id"`
	Rank     int    `json:"rank,omitempty"`
	Quantity int64  `json:"quantity"`
}

func AdminStockSectVault(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	sectID := c.Param("id")
	var req AdminStockVaultRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ItemID == "" || req.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`SELECT 1 FROM sects WHERE id = $1 FOR UPDATE`, sectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tông môn"})
		return
	}

	switch strings.ToLower(req.ItemType) {
	case "consumable", "currency":
		c.JSON(http.StatusForbidden, gin.H{"error": "Không thể nhập vật phẩm loại Tiêu hao (Consumable) hoặc Tiền tệ (Currency) vào bảo khố tông môn"})
		return

	case "gu":
		if req.Rank <= 0 {
			req.Rank = 1
		}
		var stockInstID string
		var rowID string
		if req.Rank >= 6 {
			stockInstID = uuid.New().String()
			rowID = fmt.Sprintf("%s:%s", sectID, stockInstID)
		} else {
			rowID = fmt.Sprintf("%s:%s:%d", sectID, req.ItemID, req.Rank)
		}
		_, err = tx.Exec(`
			INSERT INTO sect_vault_gu (id, sect_id, gu_id, rank, quantity, instance_id)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO UPDATE SET
				quantity = sect_vault_gu.quantity + EXCLUDED.quantity,
				updated_at = NOW()
		`, rowID, sectID, req.ItemID, req.Rank, req.Quantity, stockInstID)

	case "material":
		var dbItemType string
		err := tx.QueryRow(`SELECT COALESCE(item_type, '') FROM item_dictionary WHERE id = $1`, req.ItemID).Scan(&dbItemType)
		if err != nil {
			if spec, ok := models.ItemRegistry[req.ItemID]; ok {
				dbItemType = spec.Type
				if dbItemType == "" {
					dbItemType = spec.ItemType
				}
			}
		}
		if strings.EqualFold(dbItemType, "Consumable") || strings.EqualFold(dbItemType, "Currency") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Không thể nhập vật phẩm loại Tiêu hao (Consumable) hoặc Tiền tệ (Currency) vào bảo khố tông môn"})
			return
		}

		_, err = tx.Exec(`
			INSERT INTO sect_vault_materials (sect_id, item_id, quantity)
			VALUES ($1, $2, $3)
			ON CONFLICT (sect_id, item_id) DO UPDATE SET
				quantity = sect_vault_materials.quantity + EXCLUDED.quantity,
				updated_at = NOW()
		`, sectID, req.ItemID, req.Quantity)

	case "recipe":
		_, err = tx.Exec(`
			INSERT INTO sect_vault_recipes (sect_id, recipe_id, completion_rate)
			VALUES ($1, $2, 100.0)
			ON CONFLICT (sect_id, recipe_id) DO UPDATE SET
				completion_rate = 100.0, unlocked_at = NOW()
		`, sectID, req.ItemID)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại vật phẩm không hợp lệ (gu, material, recipe)"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật bảo khố: " + err.Error()})
		return
	}

	tx.Commit()
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.stock_sect_vault",
		EntityType: "sect", EntityID: sectID,
		RequestSummary: fmt.Sprintf("type=%s item=%s qty=%d", req.ItemType, req.ItemID, req.Quantity),
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã thêm vào bảo khố thành công"})
}

func AdminRemoveSectVaultItem(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	sectID := c.Param("id")
	var req AdminStockVaultRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ItemID == "" || req.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	switch req.ItemType {
	case "gu":
		rows, err := tx.Query(`
			SELECT id, quantity FROM sect_vault_gu
			WHERE sect_id = $1 AND (gu_id = $2 OR LOWER(gu_id) = LOWER($2) OR id = $2 OR instance_id = $2) AND ($3 = 0 OR rank = $3) AND quantity > 0
			ORDER BY id ASC FOR UPDATE
		`, sectID, req.ItemID, req.Rank)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn bảo khố"})
			return
		}
		type guRemoval struct {
			id       string
			quantity int64
		}
		var removals []guRemoval
		var totalQty int64
		for rows.Next() {
			var r guRemoval
			if err := rows.Scan(&r.id, &r.quantity); err != nil {
				rows.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đọc dữ liệu"})
				return
			}
			removals = append(removals, r)
			totalQty += r.quantity
		}
		rows.Close()

		if len(removals) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy vật phẩm trong bảo khố"})
			return
		}
		if totalQty < req.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ số lượng trong bảo khố"})
			return
		}

		remaining := req.Quantity
		for _, r := range removals {
			if remaining <= 0 {
				break
			}
			toDeduct := r.quantity
			if toDeduct > remaining {
				toDeduct = remaining
			}
			if toDeduct >= r.quantity {
				_, err = tx.Exec(`DELETE FROM sect_vault_gu WHERE id = $1`, r.id)
			} else {
				_, err = tx.Exec(`UPDATE sect_vault_gu SET quantity = quantity - $1, updated_at = NOW() WHERE id = $2`, toDeduct, r.id)
			}
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật bảo khố: " + err.Error()})
				return
			}
			remaining -= toDeduct
		}

	case "material":
		res, err := tx.Exec(`
			UPDATE sect_vault_materials 
			SET quantity = quantity - $1, updated_at = NOW() 
			WHERE sect_id = $2 AND item_id = $3 AND quantity >= $1
		`, req.Quantity, sectID, req.ItemID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật bảo khố: " + err.Error()})
			return
		}
		if rows, _ := res.RowsAffected(); rows == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ số lượng trong bảo khố hoặc không tìm thấy"})
			return
		}
		_, _ = tx.Exec(`DELETE FROM sect_vault_materials WHERE sect_id = $1 AND item_id = $2 AND quantity <= 0`, sectID, req.ItemID)

	case "recipe":
		res, err := tx.Exec(`DELETE FROM sect_vault_recipes WHERE sect_id = $1 AND recipe_id = $2`, sectID, req.ItemID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật bảo khố: " + err.Error()})
			return
		}
		if rows, _ := res.RowsAffected(); rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Bí phương trong bảo khố"})
			return
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại vật phẩm không hợp lệ"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật bảo khố: " + err.Error()})
		return
	}

	tx.Commit()
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.remove_sect_vault",
		EntityType: "sect", EntityID: sectID,
		RequestSummary: fmt.Sprintf("type=%s item=%s qty=%d", req.ItemType, req.ItemID, req.Quantity),
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa khỏi bảo khố thành công"})
}

// -------- Admin: Sect Missions --------

func AdminListSectMissions(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	sectID := c.Param("id")

	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT id, sect_id, title, description, mission_type, target_id, target_amount, reward_contribution, required_member_rank, expires_at FROM sect_missions WHERE sect_id = $1 ORDER BY required_member_rank, title`, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải nhiệm vụ"})
		return
	}
	defer rows.Close()

	type Mission struct {
		ID                 string     `json:"id"`
		SectID             string     `json:"sect_id"`
		Title              string     `json:"title"`
		Description        string     `json:"description"`
		MissionType        string     `json:"mission_type"`
		TargetID           string     `json:"target_id"`
		TargetAmount       int64      `json:"target_amount"`
		RewardContribution int64      `json:"reward_contribution"`
		RequiredMemberRank string     `json:"required_member_rank"`
		ExpiresAt          *time.Time `json:"expires_at"`
	}

	var missions []Mission
	for rows.Next() {
		var m Mission
		if err := rows.Scan(&m.ID, &m.SectID, &m.Title, &m.Description, &m.MissionType, &m.TargetID, &m.TargetAmount, &m.RewardContribution, &m.RequiredMemberRank, &m.ExpiresAt); err == nil {
			missions = append(missions, m)
		}
	}

	c.JSON(http.StatusOK, missions)
}

type AdminCreateMissionRequest struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	MissionType        string `json:"mission_type"` // GATHER, DONATE, COMBAT, REFINE
	TargetID           string `json:"target_id"`
	TargetAmount       int64  `json:"target_amount"`
	RewardContribution int64  `json:"reward_contribution"`
	RequiredMemberRank string `json:"required_member_rank"`
	DurationHours      int    `json:"duration_hours"` // 0 => default 24h, < 0 => no expiry
}

func AdminCreateSectMission(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	sectID := c.Param("id")

	var req AdminCreateMissionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Title == "" || req.MissionType == "" || req.TargetID == "" || req.TargetAmount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	if req.RequiredMemberRank == "" {
		req.RequiredMemberRank = "NGOAI_MON"
	}

	var expiresAt *time.Time
	switch {
	case req.DurationHours > 0:
		t := time.Now().Add(time.Duration(req.DurationHours) * time.Hour)
		expiresAt = &t
	case req.DurationHours < 0:
		expiresAt = nil
	default:
		t := time.Now().Add(24 * time.Hour)
		expiresAt = &t
	}

	_, err := db.DB.Exec(`INSERT INTO sect_missions (sect_id, title, description, mission_type, target_id, target_amount, reward_contribution, required_member_rank, expires_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		sectID, req.Title, req.Description, req.MissionType, req.TargetID, req.TargetAmount, req.RewardContribution, req.RequiredMemberRank, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo nhiệm vụ"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.create_sect_mission",
		EntityType: "sect", EntityID: sectID,
		RequestSummary: fmt.Sprintf("title=%s type=%s target=%s qty=%d", req.Title, req.MissionType, req.TargetID, req.TargetAmount),
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã tạo nhiệm vụ thành công"})
}

func AdminDeleteSectMission(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	missionID := c.Param("missionId")

	_, err := db.DB.Exec(`DELETE FROM sect_missions WHERE id = $1`, missionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa nhiệm vụ"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.delete_sect_mission",
		EntityType: "sect_mission", EntityID: missionID,
		RequestSummary: "",
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa nhiệm vụ thành công"})
}

func AdminDeleteSect(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	sectID := c.Param("id")

	// Verify sect exists before deleting
	var name string
	err := db.DB.QueryRow(`SELECT name FROM sects WHERE id = $1`, sectID).Scan(&name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tông môn"})
		return
	}

	// CASCADE will delete sect_members, sect_missions, player_sect_missions, etc
	if _, err := db.DB.Exec(`DELETE FROM sects WHERE id = $1`, sectID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa tông môn"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.delete_sect",
		EntityType: "sect", EntityID: sectID,
		RequestSummary: fmt.Sprintf("name=%s", name),
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã giải tán tông môn " + name})
}
