package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func AdminListPlayers(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	var total int
	countQuery := `SELECT COUNT(*) FROM players`
	args := []interface{}{}
	if search != "" {
		countQuery += ` WHERE username ILIKE $1 OR email ILIKE $1`
		args = append(args, "%"+search+"%")
	}
	if err := db.DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đếm số lượng người chơi"})
		return
	}

	query := `SELECT id, username, COALESCE(email, ''), rank, sub_rank, current_mana, max_mana, hp, base_hp, primeval_stones, immortal_stones, wisdom, COALESCE(is_admin, false), created_at, COALESCE(current_region, ''), COALESCE(aptitude_type, ''), wall_progress, human_qi, lifespan, max_lifespan FROM players`
	if search != "" {
		query += ` WHERE username ILIKE $1 OR email ILIKE $1`
	}
	query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)

	dataArgs := append(args, limit, offset)
	if search == "" {
		dataArgs = []interface{}{limit, offset}
	}

	rows, err := db.DB.Query(query, dataArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách người chơi"})
		return
	}
	defer rows.Close()

	players := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var username, email, currentRegion, aptitudeType string
		var rank, subRank, primevalStones int
		var currentMana, maxMana, hp, baseHP, immortalStones, wisdom, wallProgress, humanQi, lifespan, maxLifespan float64
		var isAdmin bool
		var createdAt time.Time

		if err := rows.Scan(&id, &username, &email, &rank, &subRank, &currentMana, &maxMana, &hp, &baseHP, &primevalStones, &immortalStones, &wisdom, &isAdmin, &createdAt, &currentRegion, &aptitudeType, &wallProgress, &humanQi, &lifespan, &maxLifespan); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc dữ liệu người chơi"})
			return
		}
		players = append(players, gin.H{
			"id":              id,
			"username":        username,
			"email":           email,
			"rank":            rank,
			"sub_rank":        subRank,
			"current_mana":    currentMana,
			"max_mana":        maxMana,
			"hp":              hp,
			"base_hp":         baseHP,
			"primeval_stones": primevalStones,
			"immortal_stones": immortalStones,
			"wisdom":          wisdom,
			"is_admin":        isAdmin,
			"created_at":      createdAt,
			"current_region":  currentRegion,
			"aptitude_type":   aptitudeType,
			"wall_progress":   wallProgress,
			"human_qi":        humanQi,
			"lifespan":        lifespan,
			"max_lifespan":    maxLifespan,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  players,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func AdminGetPlayerDetail(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	player, err := loadPlayerForStatus(db.DB, playerID, false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                   player.ID,
		"username":             player.Username,
		"email":                player.Email,
		"rank":                 player.Rank,
		"sub_rank":             player.SubRank,
		"aptitude_type":        player.AptitudeType,
		"aptitude_percent":     player.AptitudePercent,
		"aptitude_grade":       player.AptitudeGrade,
		"current_mana":         player.CurrentMana,
		"max_mana":             player.MaxMana,
		"primeval_stones":      player.PrimevalStones,
		"immortal_stones":      player.ImmortalStones,
		"wisdom":               player.Wisdom,
		"hp":                   player.HP,
		"base_hp":              player.BaseHP,
		"wall_progress":        player.WallProgress,
		"human_qi":             player.HumanQi,
		"current_region":       player.CurrentRegion,
		"is_admin":             player.IsAdmin,
		"created_at":           player.CreatedAt,
		"gu_inventory":         player.GuInventory,
		"materials_inventory":  player.MaterialsInventory,
		"lifespan":             player.Lifespan,
		"max_lifespan":         player.MaxLifespan,
		"soul_hp":              player.SoulHP,
		"max_soul":             player.MaxSoul,
		"soul_tier":            player.SoulTier,
		"explore_count":        player.ExploreCount,
		"aperture_stress":      player.ApertureStress,
		"avatar_url":           player.AvatarURL,
		"bio":                  player.Bio,
	})
}

type AdminItemRequest struct {
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
}

func AdminAddPlayerItems(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	var req AdminItemRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ItemID == "" || req.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	if _, exists := models.ItemRegistry[req.ItemID]; !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm không tồn tại"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	player, err := loadPlayerForStatus(tx, playerID, true)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}

	player.MaterialsInventory[req.ItemID] += req.Quantity

	if _, err := tx.Exec(`UPDATE players SET materials_inventory=$1 WHERE id=$2`, player.MaterialsInventory, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật vật phẩm"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Đã thêm vật phẩm thành công", "item_id": req.ItemID, "quantity": req.Quantity})
}

func AdminRemovePlayerItems(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	var req AdminItemRequest
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

	player, err := loadPlayerForStatus(tx, playerID, true)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}

	currentQty := player.MaterialsInventory[req.ItemID]
	if currentQty < req.Quantity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Người chơi không có đủ vật phẩm"})
		return
	}

	player.MaterialsInventory[req.ItemID] -= req.Quantity
	if player.MaterialsInventory[req.ItemID] <= 0 {
		delete(player.MaterialsInventory, req.ItemID)
	}

	if _, err := tx.Exec(`UPDATE players SET materials_inventory=$1 WHERE id=$2`, player.MaterialsInventory, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật vật phẩm"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa vật phẩm thành công", "item_id": req.ItemID, "quantity": req.Quantity})
}

type AdminAddGuRequest struct {
	GuID     string `json:"gu_id"`
	Quantity int    `json:"quantity"`
}

func AdminAddPlayerGus(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	var req AdminAddGuRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.GuID == "" || req.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	if _, exists := models.GuRegistry[req.GuID]; !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng không tồn tại"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	player, err := loadPlayerForStatus(tx, playerID, true)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}

	for i := 0; i < req.Quantity; i++ {
		player.GuInventory = append(player.GuInventory, models.GuInstance{
			InstanceID: uuid.New().String(),
			GuID:       req.GuID,
			Hunger:     100,
			Status:     "ACTIVE",
			Equipped:   false,
		})
	}

	if _, err := tx.Exec(`UPDATE players SET gu_inventory=$1 WHERE id=$2`, player.GuInventory, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật cổ trùng"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Đã thêm cổ trùng thành công", "gu_id": req.GuID, "quantity": req.Quantity})
}

func AdminRemovePlayerGu(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	instanceID := c.Param("instance_id")
	if instanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instance ID không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	player, err := loadPlayerForStatus(tx, playerID, true)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}

	found := false
	newInv := models.GuInventory{}
	for _, gu := range player.GuInventory {
		if gu.InstanceID == instanceID {
			found = true
			continue
		}
		newInv = append(newInv, gu)
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy cổ trùng này"})
		return
	}

	if _, err := tx.Exec(`UPDATE players SET gu_inventory=$1 WHERE id=$2`, newInv, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật cổ trùng"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa cổ trùng thành công"})
}

func AdminChangePlayerStones(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	var req struct {
		PrimevalStones *int     `json:"primeval_stones"`
		ImmortalStones *float64 `json:"immortal_stones"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	if req.PrimevalStones == nil && req.ImmortalStones == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phải nhập ít nhất một giá trị"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	sets := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.PrimevalStones != nil {
		sets = append(sets, "primeval_stones=$"+strconv.Itoa(argIdx))
		args = append(args, *req.PrimevalStones)
		argIdx++
	}
	if req.ImmortalStones != nil {
		sets = append(sets, "immortal_stones=$"+strconv.Itoa(argIdx))
		args = append(args, *req.ImmortalStones)
		argIdx++
	}
	args = append(args, playerID)

	if _, err := tx.Exec(`UPDATE players SET `+strings.Join(sets, ", ")+` WHERE id=$`+strconv.Itoa(argIdx), args...); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật tài nguyên"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật tài nguyên thành công"})
}

// -------- Admin: Sect Management --------

func AdminListSects(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	rows, err := db.DB.Query(`
		SELECT s.id, s.name, s.description, s.region, s.level, s.funds, s.require_approval,
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
		ID              string `json:"id"`
		Name            string `json:"name"`
		Description     string `json:"description"`
		Region          string `json:"region"`
		Level           int    `json:"level"`
		Funds           int    `json:"funds"`
		RequireApproval bool   `json:"require_approval"`
		MemberCount     int    `json:"member_count"`
		LeaderName      string `json:"leader_name,omitempty"`
	}

	var sects []SectInfo
	for rows.Next() {
		var s SectInfo
		var leaderName *string
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Region, &s.Level, &s.Funds, &s.RequireApproval, &s.MemberCount, &leaderName); err != nil {
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
		ID              string          `json:"id"`
		Name            string          `json:"name"`
		Description     string          `json:"description"`
		Region          string          `json:"region"`
		Level           int             `json:"level"`
		Funds           int             `json:"funds"`
		RequireApproval bool            `json:"require_approval"`
		VaultGu         json.RawMessage `json:"vault_gu"`
		VaultMaterials  json.RawMessage `json:"vault_materials"`
		VaultRecipes    json.RawMessage `json:"vault_recipes"`
	}
	var s sectVault
	err := db.DB.QueryRow(`SELECT id, name, description, region, level, funds, require_approval, vault_gu, vault_materials, vault_recipes FROM sects WHERE id = $1`, sectID).Scan(&s.ID, &s.Name, &s.Description, &s.Region, &s.Level, &s.Funds, &s.RequireApproval, &s.VaultGu, &s.VaultMaterials, &s.VaultRecipes)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tông môn"})
		return
	}

	rows, err := db.DB.Query(`
		SELECT sm.player_id, p.username, sm.member_rank, sm.contribution_points, sm.joined_at, p.rank
		FROM sect_members sm
		JOIN players p ON p.id = sm.player_id
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải thành viên"})
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
	Name            *string `json:"name"`
	Description     *string `json:"description"`
	Region          *string `json:"region"`
	Level           *int    `json:"level"`
	Funds           *int    `json:"funds"`
	RequireApproval *bool   `json:"require_approval"`
}

func AdminUpdateSect(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
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

	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật tông môn thành công"})
}

func AdminSetSectLeader(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
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

	c.JSON(http.StatusOK, gin.H{"message": "Đã bổ nhiệm tân Chưởng môn thành công"})
}

func AdminKickSectMember(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
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
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa thành viên khỏi tông môn"})
}

type AdminSectAddMemberRequest struct {
	PlayerID   string `json:"player_id"`
	MemberRank string `json:"member_rank"`
}

func AdminAddToSect(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
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
	validRanks := map[string]bool{"NGOAI_MON": true, "NOI_MON": true, "TINH_ANH": true, "CHAN_TRUYEN": true, "TRUONG_LAO": true, "CHUONG_MON": true}
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
	c.JSON(http.StatusOK, gin.H{"message": "Đã thêm " + username + " vào tông môn thành công"})
}

// -------- Admin: Sect Vault --------

type AdminStockVaultRequest struct {
	ItemType string `json:"item_type"` // gu, material, recipe
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
}

func AdminStockSectVault(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
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
			if item.ID == req.ItemID {
				guItems[i].Quantity += req.Quantity
				found = true
				break
			}
		}
		if !found {
			guItems = append(guItems, SectVaultGuItem{ID: req.ItemID, Cost: 0, Quantity: req.Quantity})
		}
		updated, _ := json.Marshal(guItems)
		_, err = tx.Exec(`UPDATE sects SET vault_gu=$1 WHERE id=$2`, updated, sectID)

	case "material":
		var mats map[string]int
		json.Unmarshal(vaultMatsJSON, &mats)
		if mats == nil {
			mats = map[string]int{}
		}
		mats[req.ItemID] += req.Quantity
		updated, _ := json.Marshal(mats)
		_, err = tx.Exec(`UPDATE sects SET vault_materials=$1 WHERE id=$2`, updated, sectID)

	case "recipe":
		var recipes []string
		json.Unmarshal(vaultRecipesJSON, &recipes)
		if recipes == nil {
			recipes = []string{}
		}
		recipes = append(recipes, req.ItemID)
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
	c.JSON(http.StatusOK, gin.H{"message": "Đã thêm vào bảo khố thành công"})
}

func AdminRemoveSectVaultItem(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
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
			if item.ID == req.ItemID {
				if item.Quantity < req.Quantity {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ số lượng trong bảo khố"})
					return
				}
				guItems[i].Quantity -= req.Quantity
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
		var mats map[string]int
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

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại vật phẩm không hợp lệ"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật bảo khố"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa khỏi bảo khố thành công"})
}

// -------- Admin: Sect Missions --------

func AdminListSectMissions(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	sectID := c.Param("id")

	rows, err := db.DB.Query(`SELECT id, sect_id, title, description, mission_type, target_id, target_amount, reward_contribution, required_member_rank FROM sect_missions WHERE sect_id = $1 ORDER BY required_member_rank, title`, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải nhiệm vụ"})
		return
	}
	defer rows.Close()

	type Mission struct {
		ID                  string `json:"id"`
		SectID              string `json:"sect_id"`
		Title               string `json:"title"`
		Description         string `json:"description"`
		MissionType         string `json:"mission_type"`
		TargetID            string `json:"target_id"`
		TargetAmount        int    `json:"target_amount"`
		RewardContribution  int    `json:"reward_contribution"`
		RequiredMemberRank  string `json:"required_member_rank"`
	}

	var missions []Mission
	for rows.Next() {
		var m Mission
		if err := rows.Scan(&m.ID, &m.SectID, &m.Title, &m.Description, &m.MissionType, &m.TargetID, &m.TargetAmount, &m.RewardContribution, &m.RequiredMemberRank); err == nil {
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
	TargetAmount       int    `json:"target_amount"`
	RewardContribution int    `json:"reward_contribution"`
	RequiredMemberRank string `json:"required_member_rank"`
}

func AdminCreateSectMission(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
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

	_, err := db.DB.Exec(`INSERT INTO sect_missions (sect_id, title, description, mission_type, target_id, target_amount, reward_contribution, required_member_rank) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		sectID, req.Title, req.Description, req.MissionType, req.TargetID, req.TargetAmount, req.RewardContribution, req.RequiredMemberRank)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo nhiệm vụ"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã tạo nhiệm vụ thành công"})
}

func AdminDeleteSectMission(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	missionID := c.Param("missionId")

	_, err := db.DB.Exec(`DELETE FROM sect_missions WHERE id = $1`, missionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa nhiệm vụ"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa nhiệm vụ thành công"})
}

func AdminDeleteSect(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
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

	c.JSON(http.StatusOK, gin.H{"message": "Đã giải tán tông môn " + name})
}

// AdminImportData seeds all dictionary tables from CSV files.
// POST /api/admin/data/import
func AdminImportData(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	if err := db.SeedFromCSV(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Import thất bại: " + err.Error()})
		return
	}

	// Reload registries
	if err := models.LoadGuRegistry(db.DB); err != nil {
		log.Printf("Warning: failed to reload Gu registry after import: %v", err)
	}
	if err := models.LoadItemRegistry(db.DB); err != nil {
		log.Printf("Warning: failed to reload Item registry after import: %v", err)
	}
	if err := models.LoadMonsterRegistry(db.DB); err != nil {
		log.Printf("Warning: failed to reload Monster registry after import: %v", err)
	}
	if err := LoadQuestRegistry(); err != nil {
		log.Printf("Warning: failed to reload Quest registry after import: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Import dữ liệu từ CSV thành công"})
}

// AdminExportData exports all dictionary tables to CSV files.
// POST /api/admin/data/export
func AdminExportData(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	if err := db.ExportToCSV(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Export thất bại: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Export dữ liệu ra CSV thành công"})
}
