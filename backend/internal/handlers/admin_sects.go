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
	err := db.DB.QueryRowContext(c.Request.Context(), `SELECT id, name, description, region, level, funds, require_approval, require_promotion_approval, vault_gu, vault_materials, vault_recipes FROM sects WHERE id = $1`, sectID).Scan(&s.ID, &s.Name, &s.Description, &s.Region, &s.Level, &s.Funds, &s.RequireApproval, &s.RequirePromotionApproval, &s.VaultGu, &s.VaultMaterials, &s.VaultRecipes)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tông môn"})
		return
	}

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

	var vaultGuJSON, vaultMatsJSON, vaultRecipesJSON []byte
	err = tx.QueryRow(`SELECT vault_gu, vault_materials, vault_recipes FROM sects WHERE id = $1 FOR UPDATE`, sectID).Scan(&vaultGuJSON, &vaultMatsJSON, &vaultRecipesJSON)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tông môn"})
		return
	}

	switch req.ItemType {
	case "gu":
		var guItems []SectVaultGuItem
		json.Unmarshal(vaultGuJSON, &guItems)
		if guItems == nil {
			guItems = []SectVaultGuItem{}
		}
		found := false
		for i, item := range guItems {
			if item.ID == req.ItemID && (req.Rank == 0 || item.Rank == req.Rank) {
				guItems[i].Quantity += req.Quantity
				guItems[i].Hunger = vaultGuMaxHunger(req.ItemID, item.Rank)
				found = true
				break
			}
		}
		if !found {
			guItems = append(guItems, SectVaultGuItem{ID: req.ItemID, Rank: req.Rank, Cost: 0, Quantity: req.Quantity, Hunger: vaultGuMaxHunger(req.ItemID, req.Rank)})
		}
		updated, _ := json.Marshal(guItems)
		_, err = tx.Exec(`UPDATE sects SET vault_gu=$1 WHERE id=$2`, updated, sectID)

	case "material":
		var mats map[string]int64
		json.Unmarshal(vaultMatsJSON, &mats)
		if mats == nil {
			mats = map[string]int64{}
		}
		mats[req.ItemID] += req.Quantity
		updated, _ := json.Marshal(mats)
		_, err = tx.Exec(`UPDATE sects SET vault_materials=$1 WHERE id=$2`, updated, sectID)

	case "recipe":
		var recipes []models.SectVaultRecipe
		json.Unmarshal(vaultRecipesJSON, &recipes)
		if recipes == nil {
			recipes = []models.SectVaultRecipe{}
		}
		found := false
		for i := range recipes {
			if recipes[i].RecipeID == req.ItemID {
				if recipes[i].CompletionRate < 100.0 {
					recipes[i].CompletionRate = 100.0
				}
				found = true
				break
			}
		}
		if !found {
			recipes = append(recipes, models.SectVaultRecipe{RecipeID: req.ItemID, CompletionRate: 100.0})
		}
		updated, _ := json.Marshal(recipes)
		_, err = tx.Exec(`UPDATE sects SET vault_recipes=$1 WHERE id=$2`, updated, sectID)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại vật phẩm không hợp lệ (gu, material, recipe)"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật bảo khố"})
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

	var vaultGuJSON, vaultMatsJSON, vaultRecipesJSON []byte
	err = tx.QueryRow(`SELECT vault_gu, vault_materials, vault_recipes FROM sects WHERE id = $1 FOR UPDATE`, sectID).Scan(&vaultGuJSON, &vaultMatsJSON, &vaultRecipesJSON)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tông môn"})
		return
	}

	switch req.ItemType {
	case "gu":
		var guItems []SectVaultGuItem
		json.Unmarshal(vaultGuJSON, &guItems)
		found := false
		for i, item := range guItems {
			if item.ID == req.ItemID && (req.Rank == 0 || item.Rank == req.Rank) {
				if item.Quantity < req.Quantity {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ số lượng trong bảo khố"})
					return
				}
				guItems[i].Quantity -= req.Quantity
				if guItems[i].Quantity <= 0 {
					guItems = append(guItems[:i], guItems[i+1:]...)
				}
				found = true
				break
			}
		}
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy vật phẩm trong bảo khố"})
			return
		}
		updated, _ := json.Marshal(guItems)
		_, err = tx.Exec(`UPDATE sects SET vault_gu=$1 WHERE id=$2`, updated, sectID)

	case "material":
		var mats map[string]int64
		json.Unmarshal(vaultMatsJSON, &mats)
		if mats[req.ItemID] < req.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ số lượng trong bảo khố"})
			return
		}
		mats[req.ItemID] -= req.Quantity
		if mats[req.ItemID] <= 0 {
			delete(mats, req.ItemID)
		}
		updated, _ := json.Marshal(mats)
		_, err = tx.Exec(`UPDATE sects SET vault_materials=$1 WHERE id=$2`, updated, sectID)

	case "recipe":
		var recipes []models.SectVaultRecipe
		json.Unmarshal(vaultRecipesJSON, &recipes)
		removed := int64(0)
		newRecipes := make([]models.SectVaultRecipe, 0, len(recipes))
		for _, r := range recipes {
			if r.RecipeID == req.ItemID && removed < req.Quantity {
				removed++
				continue
			}
			newRecipes = append(newRecipes, r)
		}
		if removed == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Bí phương trong bảo khố"})
			return
		}
		updated, _ := json.Marshal(newRecipes)
		_, err = tx.Exec(`UPDATE sects SET vault_recipes=$1 WHERE id=$2`, updated, sectID)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại vật phẩm không hợp lệ"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật bảo khố"})
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
