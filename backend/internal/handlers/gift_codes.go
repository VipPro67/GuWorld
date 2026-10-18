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
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type GiftCodeReward struct {
	Type   string  `json:"type"`
	ID     string  `json:"id,omitempty"`
	Amount float64 `json:"amount"`
	Label  string  `json:"label"`
	Rank   int     `json:"rank,omitempty"`
}

type GiftCode struct {
	ID           uuid.UUID        `json:"id"`
	Code         string           `json:"code"`
	Rewards      []GiftCodeReward `json:"rewards"`
	MaxUses      int              `json:"max_uses"`
	UsedCount    int              `json:"used_count"`
	MaxPerPlayer int              `json:"max_per_player"`
	ExpiresAt    time.Time        `json:"expires_at"`
	IsActive     bool             `json:"is_active"`
	CreatedBy    *uuid.UUID       `json:"created_by,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
}

type AdminCreateGiftCodeRequest struct {
	Code         string           `json:"code" binding:"required"`
	Rewards      []GiftCodeReward `json:"rewards" binding:"required"`
	MaxUses      int              `json:"max_uses"`
	MaxPerPlayer int              `json:"max_per_player"`
	ExpiresIn    int              `json:"expires_in_days"`
}

type RedeemGiftCodeRequest struct {
	Code string `json:"code" binding:"required"`
}

func AdminCreateGiftCode(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	var req AdminCreateGiftCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Rewards) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	req.Code = strings.ToUpper(strings.TrimSpace(req.Code))
	if len(req.Code) < 4 || len(req.Code) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã quà phải từ 4-50 ký tự"})
		return
	}

	if req.MaxUses <= 0 {
		req.MaxUses = 1
	}
	if req.MaxPerPlayer <= 0 {
		req.MaxPerPlayer = 1
	}
	if req.ExpiresIn <= 0 {
		req.ExpiresIn = 30
	}

	rewardsJSON, err := json.Marshal(req.Rewards)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xử lý rewards"})
		return
	}

	var id uuid.UUID
	err = db.DB.QueryRow(`
		INSERT INTO gift_codes (code, rewards, max_uses, max_per_player, expires_at, created_by)
		VALUES ($1, $2, $3, $4, NOW() + ($5 || ' days')::INTERVAL, $6)
		RETURNING id
	`, req.Code, rewardsJSON, req.MaxUses, req.MaxPerPlayer, fmt.Sprintf("%d", req.ExpiresIn), adminID).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			c.JSON(http.StatusConflict, gin.H{"error": "Mã quà này đã tồn tại"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo mã quà: " + err.Error()})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.create_gift_code",
		EntityType: "gift_code", EntityID: id.String(),
		RequestSummary: fmt.Sprintf("code=%s uses=%d rewards=%s", req.Code, req.MaxUses, string(rewardsJSON)),
		IPAddress:      c.ClientIP(), Success: true,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Đã tạo mã quà thành công",
		"code_id": id,
		"code":    req.Code,
	})
}

func AdminListGiftCodes(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT id, code, rewards, max_uses, used_count, max_per_player, expires_at, is_active, created_by, created_at
		FROM gift_codes
		ORDER BY created_at DESC
		LIMIT 200
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách mã quà"})
		return
	}
	defer rows.Close()

	codes := []GiftCode{}
	for rows.Next() {
		var gc GiftCode
		var rewardsJSON []byte
		var createdBy sql.NullString
		if err := rows.Scan(&gc.ID, &gc.Code, &rewardsJSON, &gc.MaxUses, &gc.UsedCount, &gc.MaxPerPlayer, &gc.ExpiresAt, &gc.IsActive, &createdBy, &gc.CreatedAt); err != nil {
			continue
		}
		json.Unmarshal(rewardsJSON, &gc.Rewards)
		if createdBy.Valid {
			uid, _ := uuid.Parse(createdBy.String)
			gc.CreatedBy = &uid
		}
		codes = append(codes, gc)
	}

	c.JSON(http.StatusOK, gin.H{
		"codes": codes,
		"total": len(codes),
	})
}

func RedeemGiftCode(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req RedeemGiftCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if len(code) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã quà không được để trống"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	_, _ = tx.Exec(`SET LOCAL lock_timeout = '3000ms'`)

	var gc GiftCode
	var rewardsJSON []byte
	var createdBy sql.NullString

	// FOR UPDATE serialises concurrent redemptions of the same code: without it
	// the per-player COUNT(*) check and the used_count check below are racy and a
	// player could redeem a max_per_player=1 code several times in parallel.
	err = tx.QueryRow(`
		SELECT id, code, rewards, max_uses, used_count, max_per_player, expires_at, is_active, created_by, created_at
		FROM gift_codes
		WHERE code = $1
		FOR UPDATE
	`, code).Scan(&gc.ID, &gc.Code, &rewardsJSON, &gc.MaxUses, &gc.UsedCount, &gc.MaxPerPlayer, &gc.ExpiresAt, &gc.IsActive, &createdBy, &gc.CreatedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mã quà không tồn tại"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}

	json.Unmarshal(rewardsJSON, &gc.Rewards)

	if !gc.IsActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã quà này đã bị vô hiệu hóa"})
		return
	}

	if time.Now().After(gc.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã quà này đã hết hạn"})
		return
	}

	if gc.UsedCount >= gc.MaxUses {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã quà này đã hết lượt sử dụng"})
		return
	}

	var redemptionCount int
	tx.QueryRow(`SELECT COUNT(*) FROM gift_code_redemptions WHERE gift_code_id = $1 AND player_id = $2`, gc.ID, playerID).Scan(&redemptionCount)
	if redemptionCount >= gc.MaxPerPlayer {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn đã sử dụng mã quà này rồi"})
		return
	}

	player, err := loadPlayerForStatus(tx, playerID, true)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}

	newGus, err := applyGiftCodeRewardsTx(tx, &player, gc.Rewards)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi áp dụng phần thưởng: " + err.Error()})
		return
	}

	qiGained := 0.0

	updateQuery, updateArgs := buildGiftCodeUpdateQuery(&player)
	_, err = tx.Exec(updateQuery, updateArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật người chơi"})
		return
	}

	if player.Rank >= 6 {
		_, err = tx.Exec(`
			UPDATE immortal_apertures 
			SET dao_marks = $1 
			WHERE player_id = $2`,
			player.DaoMarks, player.ID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Dao Marks Tiên Khiếu"})
			return
		}
	}
	if err := db.AddPlayerGusBulk(tx, player.ID.String(), newGus); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật cổ trùng"})
		return
	}

	res, err := tx.Exec(`
		UPDATE gift_codes 
		SET used_count = used_count + 1 
		WHERE id = $1 AND is_active = true AND expires_at > NOW() AND used_count < max_uses
	`, gc.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật mã quà"})
		return
	}
	if rowsAffected, _ := res.RowsAffected(); rowsAffected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã quà này đã hết lượt sử dụng hoặc đã hết hạn"})
		return
	}

	_, err = tx.Exec(`INSERT INTO gift_code_redemptions (gift_code_id, player_id) VALUES ($1, $2)`, gc.ID, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi ghi nhận đổi thưởng"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi commit"})
		return
	}

	rewardStr := buildGiftCodeRewardLabel(gc.Rewards)

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &playerID,
		Action:         "gift_code.redeem",
		EntityType:     "gift_code",
		EntityID:       gc.ID.String(),
		RequestSummary: fmt.Sprintf("Nhập mã quà %s: nhận %s", gc.Code, rewardStr),
		Metadata: map[string]interface{}{
			"code":    gc.Code,
			"rewards": gc.Rewards,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	respondActionWithInventory(c, http.StatusOK, player, gin.H{
		"message": "Nhập mã quà thành công!" + rewardStr + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
	})
}

func applyGiftCodeRewardsTx(tx *sql.Tx, player *models.Player, rewards []GiftCodeReward) ([]models.GuInstance, error) {
	var newGus []models.GuInstance
	for _, r := range rewards {
		switch r.Type {
		case "human_qi":
			player.HumanQi += r.Amount
			if player.HumanQi > MaxHumanQi {
				player.HumanQi = MaxHumanQi
			}
		case "primeval_stones":
			player.PrimevalStones += float64(r.Amount)
		case "immortal_stones":
			player.ImmortalStones += float64(r.Amount)
		case "dao_mark":
			if player.DaoMarks == nil {
				player.DaoMarks = make(models.DaoMarksMap)
			}
			player.DaoMarks[r.ID] += int64(r.Amount)
		case "item":
			InventorySvc.AddItem(tx, player.ID.String(), r.ID, "materials", int64(r.Amount))
		case "gu":
			guSpec, _ := models.GetGuAny(r.ID)
			rank := r.Rank
			if rank <= 0 {
				rank = guSpec.Rank
			}
			for i := 0; i < int(r.Amount); i++ {
				newGu := models.GuInstance{
					InstanceID: "gu-" + uuid.New().String(),
					GuID:       r.ID,
					Rank:       rank,
					Hunger:     100,
					Status:     "ACTIVE",
					Equipped:   false,
				}
				player.GuInventory = append(player.GuInventory, newGu)
				newGus = append(newGus, newGu)
			}
		case "lifespan":
			player.Lifespan += r.Amount
		case "wall_progress":
			player.WallProgress += r.Amount
			if player.WallProgress > 1.0 {
				player.WallProgress = 1.0
			}
		case "recipe":
			err := RecipeSvc.Upsert(tx, &services.RecipeProgress{
				PlayerID:       player.ID.String(),
				RecipeID:       r.ID,
				CompletionRate: 100,
				IsUnlocked:     true,
			})
			if err != nil {
				return nil, err
			}
		}
	}
	return newGus, nil
}

func buildGiftCodeUpdateQuery(player *models.Player) (string, []interface{}) {
	baseFields := []string{"human_qi", "primeval_stones", "immortal_stones", "lifespan", "wall_progress"}
	baseVals := []interface{}{player.HumanQi, player.PrimevalStones, player.ImmortalStones, player.Lifespan, player.WallProgress}
	extraFields := []string{}
	extraVals := []interface{}{}

	// materials_inventory no longer stored on players table

	allFields := append(baseFields, extraFields...)
	allVals := append(baseVals, extraVals...)

	placeholders := make([]string, len(allFields))
	for i := range allFields {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	setClauses := make([]string, len(allFields))
	for i, f := range allFields {
		setClauses[i] = fmt.Sprintf("%s = %s", f, placeholders[i])
	}

	query := fmt.Sprintf("UPDATE players SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "),
		len(allFields)+1,
	)
	allVals = append(allVals, player.ID)

	return query, allVals
}

func buildGiftCodeRewardLabel(rewards []GiftCodeReward) string {
	parts := []string{}
	for _, r := range rewards {
		switch r.Type {
		case "human_qi":
			parts = append(parts, fmt.Sprintf("%.0f Nhân Khí", r.Amount))
		case "primeval_stones":
			parts = append(parts, fmt.Sprintf("%.0f Nguyên Thạch", r.Amount))
		case "immortal_stones":
			parts = append(parts, fmt.Sprintf("%.0f Tiên Nguyên Thạch", r.Amount))
		case "dao_mark":
			parts = append(parts, fmt.Sprintf("%.0f %s", r.Amount, r.ID))
		case "item":
			itemName := r.ID
			if item, ok := models.ItemRegistry[r.ID]; ok {
				itemName = item.Name
			}
			parts = append(parts, fmt.Sprintf("%.0fx %s", r.Amount, itemName))
		case "gu":
			guName := r.ID
			if gu, ok := models.GetGuAny(r.ID); ok {
				guName = gu.Name
			}
			parts = append(parts, fmt.Sprintf("%.0fx %s", r.Amount, guName))
		case "recipe":
			recipeName := r.Label
			if recipeName == "" {
				recipeName = r.ID
			}
			parts = append(parts, fmt.Sprintf("Cổ Phương %s", recipeName))
		case "lifespan":
			parts = append(parts, fmt.Sprintf("%.0f năm Thọ Nguyên", r.Amount))
		case "wall_progress":
			parts = append(parts, fmt.Sprintf("%.0f%% Phá Vách", r.Amount*100))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return " Nhận được " + strings.Join(parts, ", ") + "!"
}
