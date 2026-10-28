package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

const (
	guHouseMaxCoreGus = 5
	guHouseMinCoreGus = 2
	guHouseMaxSaved   = 4
)

type guHouseDraftRequest struct {
	Name              string   `json:"name"`
	CoreGuInstanceIDs []string `json:"core_gu_instance_ids"`
	AuxGuCount        int      `json:"aux_gu_count"`
}

type guHouseRecord struct {
	ID                string                      `json:"id"`
	PlayerID          string                      `json:"player_id"`
	Name              string                      `json:"name"`
	Rank              int                         `json:"rank"`
	CoreGuInstances   []string                    `json:"core_gu_instances"`
	AuxGuCount        int                         `json:"aux_gu_count"`
	HouseProfile      models.GuHouseCombatProfile `json:"house_profile"`
	CurrentDurability float64                     `json:"current_durability"`
	IsEquipped        bool                        `json:"is_equipped"`
	CreatedAt         string                      `json:"created_at"`
}

func calculateGuHousePreview(tx *sql.Tx, playerID uuid.UUID, req guHouseDraftRequest) (models.GuHouseCombatProfile, int, error) {
	if len(req.CoreGuInstanceIDs) < guHouseMinCoreGus || len(req.CoreGuInstanceIDs) > guHouseMaxCoreGus {
		return models.GuHouseCombatProfile{}, 0, fmt.Errorf("Tiên Cổ Ốc cần từ %d đến %d Tiên Cổ hạch tâm", guHouseMinCoreGus, guHouseMaxCoreGus)
	}
	if req.AuxGuCount < 0 {
		req.AuxGuCount = 0
	}
	if req.AuxGuCount > 200 {
		req.AuxGuCount = 200
	}

	owned, err := loadOwnedKillerMoveGus(tx, playerID)
	if err != nil {
		return models.GuHouseCombatProfile{}, 0, err
	}

	highestRank := 6
	coreGuIDs := make([]string, 0, len(req.CoreGuInstanceIDs))
	seen := map[string]bool{}

	actions := make([]models.KillerMoveCombatProfile, 0, len(req.CoreGuInstanceIDs))

	for _, id := range req.CoreGuInstanceIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		gu, ok := owned[id]
		if !ok || strings.EqualFold(gu.State, "DEAD") {
			return models.GuHouseCombatProfile{}, 0, fmt.Errorf("Cổ hạch tâm không hợp lệ hoặc đã kiệt sức")
		}
		if gu.Rank < 6 {
			return models.GuHouseCombatProfile{}, 0, fmt.Errorf("Hạch tâm Tiên Cổ Ốc phải là Tiên Cổ (Rank 6 trở lên)")
		}
		if gu.Rank > highestRank {
			highestRank = gu.Rank
		}
		coreGuIDs = append(coreGuIDs, gu.GuID)

		spec, found := models.GetGu(gu.GuID, gu.Rank)
		if found {
			actionPath := "strength"
			if len(spec.Path) > 0 {
				actionPath = gamefx.CleanPath(spec.Path[0])
			}
			actions = append(actions, models.KillerMoveCombatProfile{
				ID:            id,
				Name:          fmt.Sprintf("Trận Kích: %s", spec.Name),
				Rank:          gu.Rank,
				MainPath:      actionPath,
				AllPaths:      []string{actionPath},
				TargetType:    getGuTargetType(spec),
				CooldownTurns: 2,
				Priority:      1,
				ManaCost:      spec.BaseManaCost * 1.5,
				BaseDamage:    float64(gu.Rank * 120),
				DamageMult:    1.2,
				SuccessChance: 0.95, // Gu House provides high casting stability
			})
		}
	}

	baseHP := float64(highestRank*500 + len(coreGuIDs)*250 + req.AuxGuCount*35)
	baseDef := float64(highestRank*40 + len(coreGuIDs)*25 + req.AuxGuCount*4)
	reduction := math.Min(0.35, 0.10+float64(len(coreGuIDs))*0.05)

	profile := models.GuHouseCombatProfile{
		Name:            req.Name,
		Rank:            highestRank,
		CoreGuIDs:       coreGuIDs,
		AuxGuCount:      req.AuxGuCount,
		MaxHouseHP:      baseHP,
		CurrentHouseHP:  baseHP,
		HouseDefense:    baseDef,
		DamageReduction: reduction,
		ImmuneDebuffs:   true,
		ZeroBackfire:    true,
		Actions:         actions,
	}

	return profile, highestRank, nil
}

func ListGuHouses(c *gin.Context) {
	playerID, ok := killerMoveOwner(c)
	if !ok {
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT id::text, player_id::text, name, rank, core_gu_instances, aux_gu_count, house_profile, current_durability, is_equipped, to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM player_gu_houses
		WHERE player_id = $1
		ORDER BY created_at DESC
	`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách Tiên Cổ Ốc"})
		return
	}
	defer rows.Close()

	list := make([]guHouseRecord, 0)
	for rows.Next() {
		var r guHouseRecord
		var coreBytes, profBytes []byte
		if err := rows.Scan(&r.ID, &r.PlayerID, &r.Name, &r.Rank, &coreBytes, &r.AuxGuCount, &profBytes, &r.CurrentDurability, &r.IsEquipped, &r.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đọc dữ liệu Tiên Cổ Ốc"})
			return
		}
		_ = json.Unmarshal(coreBytes, &r.CoreGuInstances)
		_ = json.Unmarshal(profBytes, &r.HouseProfile)
		list = append(list, r)
	}

	c.JSON(http.StatusOK, gin.H{
		"gu_houses":    list,
		"max_saved":    guHouseMaxSaved,
		"max_core_gus": guHouseMaxCoreGus,
		"min_core_gus": guHouseMinCoreGus,
	})
}

func PreviewGuHouse(c *gin.Context) {
	playerID, ok := killerMoveOwner(c)
	if !ok {
		return
	}
	var req guHouseDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	profile, _, err := calculateGuHousePreview(tx, playerID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func CreateGuHouse(c *gin.Context) {
	playerID, ok := killerMoveOwner(c)
	if !ok {
		return
	}
	var req guHouseDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len([]rune(name)) > 80 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tên Tiên Cổ Ốc phải từ 1 đến 80 ký tự"})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM player_gu_houses WHERE player_id = $1`, playerID).Scan(&count); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra giới hạn"})
		return
	}
	if count >= guHouseMaxSaved {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Mỗi người chơi tối đa %d Tiên Cổ Ốc", guHouseMaxSaved)})
		return
	}

	profile, highestRank, err := calculateGuHousePreview(tx, playerID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	coreBytes, _ := json.Marshal(req.CoreGuInstanceIDs)
	profBytes, _ := json.Marshal(profile)

	var houseID string
	err = tx.QueryRow(`
		INSERT INTO player_gu_houses (player_id, name, rank, core_gu_instances, aux_gu_count, house_profile, current_durability, is_equipped)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6::jsonb, 100, false)
		RETURNING id::text
	`, playerID, name, highestRank, coreBytes, req.AuxGuCount, profBytes).Scan(&houseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu Tiên Cổ Ốc"})
		return
	}

	auditlog.Log(tx, auditlog.Entry{
		PlayerID:   &playerID,
		Action:     "gu_house.create",
		EntityType: "gu_house",
		EntityID:   houseID,
		Metadata: map[string]interface{}{
			"name":         name,
			"rank":         highestRank,
			"core_count":   len(req.CoreGuInstanceIDs),
			"aux_gu_count": req.AuxGuCount,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xác nhận giao dịch"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": houseID, "message": "Đã chế tạo thành công Tiên Cổ Ốc!"})
}

func EquipGuHouse(c *gin.Context) {
	playerID, ok := killerMoveOwner(c)
	if !ok {
		return
	}
	houseID := c.Param("id")
	if houseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu mã Tiên Cổ Ốc"})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var isEquipped bool
	err = tx.QueryRow(`SELECT is_equipped FROM player_gu_houses WHERE id=$1 AND player_id=$2 FOR UPDATE`, houseID, playerID).Scan(&isEquipped)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Tiên Cổ Ốc"})
		return
	}

	newState := !isEquipped
	if newState {
		// Unequip any currently equipped house
		_, _ = tx.Exec(`UPDATE player_gu_houses SET is_equipped=false WHERE player_id=$1`, playerID)
	}

	_, err = tx.Exec(`UPDATE player_gu_houses SET is_equipped=$1, updated_at=NOW() WHERE id=$2 AND player_id=$3`, newState, houseID, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật trạng thái trang bị"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xác nhận"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"is_equipped": newState, "message": "Cập nhật trang bị thành công"})
}

func RepairGuHouse(c *gin.Context) {
	playerID, ok := killerMoveOwner(c)
	if !ok {
		return
	}
	houseID := c.Param("id")

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var currentDurability float64
	var rank int
	err = tx.QueryRow(`SELECT current_durability, rank FROM player_gu_houses WHERE id=$1 AND player_id=$2 FOR UPDATE`, houseID, playerID).Scan(&currentDurability, &rank)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Tiên Cổ Ốc"})
		return
	}

	if currentDurability >= 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tiên Cổ Ốc vẫn nguyên vẹn (100% độ bền)"})
		return
	}

	cost := int64(math.Ceil((100.0 - currentDurability) * float64(rank) * 10))

	// Deduct primeval / immortal stones from player
	var currentStones int64
	err = tx.QueryRow(`SELECT primeval_stones FROM players WHERE id=$1 FOR UPDATE`, playerID).Scan(&currentStones)
	if err != nil || currentStones < cost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Nguyên Thạch tu sửa (cần %d)", cost)})
		return
	}

	_, _ = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones - $1 WHERE id=$2`, cost, playerID)
	_, _ = tx.Exec(`UPDATE player_gu_houses SET current_durability = 100, updated_at = NOW() WHERE id=$1`, houseID)

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xác nhận"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"current_durability": 100.0,
		"cost":               cost,
		"message":            "Đã tu sửa Tiên Cổ Ốc về trạng thái hoàn hảo!",
	})
}
