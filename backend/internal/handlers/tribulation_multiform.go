package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

// SpoilDropSpec defines item info for spoils drops
type SpoilDropSpec struct {
	ItemID   string
	ItemName string
	Quantity int
}

func getSpoilDropsForTribulation(path string, rank int, tribulationType string) []SpoilDropSpec {
	normPath := strings.ToLower(strings.TrimSpace(path))
	drops := []SpoilDropSpec{}

	switch normPath {
	case "fire":
		drops = append(drops,
			SpoilDropSpec{ItemID: "fire_crystal_rank6", ItemName: "Hỏa Diễm Tinh Hạch (Rank 6)", Quantity: 2 + rand.Intn(3)},
			SpoilDropSpec{ItemID: "fire_rat_fur_rank6", ItemName: "Lông Chuột Lửa Viêm Thú (Rank 6)", Quantity: 3 + rand.Intn(3)},
		)
	case "ice", "water":
		drops = append(drops,
			SpoilDropSpec{ItemID: "ice_crystal_rank6", ItemName: "Vạn Niên Hàn Băng Tinh (Rank 6)", Quantity: 2 + rand.Intn(3)},
			SpoilDropSpec{ItemID: "ice_cicada_wing_rank6", ItemName: "Cánh Thiền Băng Phách (Rank 6)", Quantity: 2 + rand.Intn(2)},
		)
	case "poison":
		drops = append(drops,
			SpoilDropSpec{ItemID: "poison_sand_rank6", ItemName: "U Độc Tử Sa (Rank 6)", Quantity: 3 + rand.Intn(3)},
			SpoilDropSpec{ItemID: "poison_sac_rank6", ItemName: "Túi Độc Cổ Hạt (Rank 6)", Quantity: 2 + rand.Intn(2)},
		)
	case "lightning":
		drops = append(drops,
			SpoilDropSpec{ItemID: "lightning_crystal_rank7", ItemName: "Thần Tiêu Lôi Tinh (Rank 7)", Quantity: 2 + rand.Intn(2)},
			SpoilDropSpec{ItemID: "heaven_tribulation_shard_rank7", ItemName: "Mảnh Vỡ Thiên Kiếp (Rank 7)", Quantity: 1 + rand.Intn(2)},
		)
	case "metal", "strength":
		drops = append(drops,
			SpoilDropSpec{ItemID: "metal_ore_rank6", ItemName: "Huyền Thiết Bì Cương (Rank 6)", Quantity: 4 + rand.Intn(4)},
			SpoilDropSpec{ItemID: "wild_boar_tusk_rank6", ItemName: "Nanh Heo Rừng Thiết Bì (Rank 6)", Quantity: 2 + rand.Intn(2)},
		)
	case "blood":
		drops = append(drops,
			SpoilDropSpec{ItemID: "blood_crystal_rank7", ItemName: "Huyết Hải Ma Tinh (Rank 7)", Quantity: 2 + rand.Intn(2)},
			SpoilDropSpec{ItemID: "blood_bat_wing_rank7", ItemName: "Cánh Dơi Huyết Bức (Rank 7)", Quantity: 2 + rand.Intn(2)},
		)
	default:
		if rank >= 8 {
			drops = append(drops,
				SpoilDropSpec{ItemID: "heaven_sand_rank8", ItemName: "Cửu Thiên Tinh Sa (Rank 8)", Quantity: 2 + rand.Intn(2)},
				SpoilDropSpec{ItemID: "heaven_tribulation_shard_rank8", ItemName: "Mảnh Vỡ Hạo Kiếp (Rank 8)", Quantity: 1 + rand.Intn(2)},
			)
		} else if rank >= 7 {
			drops = append(drops,
				SpoilDropSpec{ItemID: "heaven_tribulation_shard_rank7", ItemName: "Mảnh Vỡ Thiên Kiếp (Rank 7)", Quantity: 2 + rand.Intn(2)},
				SpoilDropSpec{ItemID: "earth_soil_rank7", ItemName: "Tức Thổ Tiên Khiếu (Rank 7)", Quantity: 3 + rand.Intn(3)},
			)
		} else {
			drops = append(drops,
				SpoilDropSpec{ItemID: "heaven_tribulation_shard_rank6", ItemName: "Mảnh Vỡ Địa Tai (Rank 6)", Quantity: 2 + rand.Intn(2)},
				SpoilDropSpec{ItemID: "earth_soil_rank6", ItemName: "Linh Thổ Địa Mạch (Rank 6)", Quantity: 3 + rand.Intn(3)},
			)
		}
	}

	return drops
}

func getBeastCaptureDrop(enemyPath string, rank int) SpoilDropSpec {
	switch strings.ToLower(strings.TrimSpace(enemyPath)) {
	case "fire":
		return SpoilDropSpec{ItemID: "fire_rat_fur_rank6", ItemName: "Lông Chuột Lửa Viêm Thú (Rank 6)", Quantity: 2}
	case "ice", "water":
		return SpoilDropSpec{ItemID: "ice_cicada_wing_rank6", ItemName: "Cánh Thiền Băng Phách (Rank 6)", Quantity: 2}
	case "poison":
		return SpoilDropSpec{ItemID: "poison_sac_rank6", ItemName: "Túi Độc Cổ Hạt (Rank 6)", Quantity: 2}
	case "lightning":
		return SpoilDropSpec{ItemID: "lightning_crystal_rank7", ItemName: "Thần Tiêu Lôi Tinh (Rank 7)", Quantity: 1}
	case "blood":
		return SpoilDropSpec{ItemID: "blood_bat_wing_rank7", ItemName: "Cánh Dơi Huyết Bức (Rank 7)", Quantity: 2}
	default:
		return SpoilDropSpec{ItemID: "wild_boar_tusk_rank6", ItemName: "Nanh Heo Rừng Thiết Bì (Rank 6)", Quantity: 2}
	}
}

// GenerateTribulationSpoilsAndFalloutTx generates spoils nodes on grid and marks contaminated soil
func GenerateTribulationSpoilsAndFalloutTx(tx *sql.Tx, eventID, apertureID, playerID, path string, rank int, tribulationType string) error {
	// Idempotency: skip if spoils were already generated for this event
	var existingSpoilsCount int
	_ = tx.QueryRow(`SELECT COUNT(1) FROM aperture_tribulation_spoils WHERE event_id = $1`, eventID).Scan(&existingSpoilsCount)
	if existingSpoilsCount > 0 {
		_, err := tx.Exec(`UPDATE tribulation_events SET stage = 'AFTERMATH' WHERE id = $1`, eventID)
		return err
	}

	// 1. Generate 3 to 5 spoils items
	drops := getSpoilDropsForTribulation(path, rank, tribulationType)
	usedCoords := make(map[string]bool)

	for _, drop := range drops {
		var x, y int
		for attempts := 0; attempts < 20; attempts++ {
			x = rand.Intn(20)
			y = rand.Intn(20)
			key := fmt.Sprintf("%d,%d", x, y)
			if !usedCoords[key] {
				usedCoords[key] = true
				break
			}
		}

		spoilID := uuid.New()
		_, err := tx.Exec(`
			INSERT INTO aperture_tribulation_spoils (id, event_id, aperture_id, player_id, item_id, item_name, quantity, x, y, claimed, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, false, NOW())
		`, spoilID, eventID, apertureID, playerID, drop.ItemID, drop.ItemName, drop.Quantity, x, y)
		if err != nil {
			return fmt.Errorf("insert spoil: %w", err)
		}
	}

	// 2. Generate 2 to 4 soil contaminations
	contamType := "fire"
	switch strings.ToLower(path) {
	case "ice", "water":
		contamType = "ice"
	case "poison":
		contamType = "poison"
	case "lightning":
		contamType = "lightning"
	default:
		contamType = "fire"
	}

	contamCount := 2 + rand.Intn(3)
	for i := 0; i < contamCount; i++ {
		var cx, cy int
		for attempts := 0; attempts < 20; attempts++ {
			cx = rand.Intn(20)
			cy = rand.Intn(20)
			key := fmt.Sprintf("%d,%d", cx, cy)
			if !usedCoords[key] {
				usedCoords[key] = true
				break
			}
		}

		_, _ = tx.Exec(`
			INSERT INTO aperture_soil_contaminations (aperture_id, x, y, contam_type, path, severity, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW())
			ON CONFLICT (aperture_id, x, y) DO UPDATE SET severity = aperture_soil_contaminations.severity + 1
		`, apertureID, cx, cy, contamType, path, 1+rand.Intn(2))
	}

	// 3. Mark event stage as AFTERMATH
	_, err := tx.Exec(`
		UPDATE tribulation_events
		SET stage = 'AFTERMATH'
		WHERE id = $1
	`, eventID)
	return err
}

// AdvanceTribulationStage transitions HARBINGER to ORDEAL_ACTIVE immediately
func AdvanceTribulationStage(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	eventIDStr := c.Param("eventId")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã đợt tai kiếp không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database transaction failure", err))
		return
	}
	defer tx.Rollback()

	var stage, status string
	err = tx.QueryRow(`
		SELECT stage, status FROM tribulation_events WHERE id = $1 AND player_id = $2
	`, eventID, playerID).Scan(&stage, &status)
	if err == sql.ErrNoRows || status != "ACTIVE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đợt tai kiếp không tồn tại hoặc đã kết thúc"})
		return
	} else if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi kiểm tra tai kiếp", err))
		return
	}

	if stage != "HARBINGER" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tai kiếp đã bước vào giai đoạn nghênh chiến"})
		return
	}

	_, err = tx.Exec(`
		UPDATE tribulation_events
		SET stage = 'ORDEAL_ACTIVE'
		WHERE id = $1
	`, eventID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi cập nhật giai đoạn tai kiếp", err))
		return
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Commit thất bại", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cổ Tiên đã chủ động xuất trận nghênh kiếp sớm! Giai đoạn tác chiến chính thức bắt đầu.",
		"stage":   "ORDEAL_ACTIVE",
	})
}

// MitigateEnvironmentalHazard resolves hazard cells using Sát Chiêu
func MitigateEnvironmentalHazard(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	eventIDStr := c.Param("eventId")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã đợt tai kiếp không hợp lệ"})
		return
	}

	var req struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tọa độ không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database transaction failure", err))
		return
	}
	defer tx.Rollback()

	var archetype, stage, status, path, tribulationType, apertureID string
	var rank int
	var hazardRaw []byte
	err = tx.QueryRow(`
		SELECT ev.archetype, ev.stage, ev.status, ev.path, ev.tribulation_type, ev.aperture_id, ev.hazard_metadata, p.rank
		FROM tribulation_events ev
		JOIN players p ON p.id = ev.player_id
		WHERE ev.id = $1 AND ev.player_id = $2
	`, eventID, playerID).Scan(&archetype, &stage, &status, &path, &tribulationType, &apertureID, &hazardRaw, &rank)
	if err == sql.ErrNoRows || status != "ACTIVE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đợt tai kiếp không tồn tại hoặc đã kết thúc"})
		return
	} else if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi kiểm tra tai kiếp", err))
		return
	}

	if archetype != "ENVIRONMENTAL" || stage != "ORDEAL_ACTIVE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tai kiếp này không ở giai đoạn hóa giải môi trường"})
		return
	}

	var meta map[string]interface{}
	if len(hazardRaw) > 0 {
		_ = json.Unmarshal(hazardRaw, &meta)
	} else {
		meta = make(map[string]interface{})
	}

	cells, _ := meta["cells"].([]interface{})
	allExtinguished := true
	foundCell := false

	for _, rawCell := range cells {
		if cellMap, ok := rawCell.(map[string]interface{}); ok {
			cx := int(cellMap["x"].(float64))
			cy := int(cellMap["y"].(float64))
			extinguished, _ := cellMap["extinguished"].(bool)

			if cx == req.X && cy == req.Y {
				if extinguished {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Vùng hiểm họa này đã được dập tắt trước đó"})
					return
				}
				cellMap["extinguished"] = true
				foundCell = true
			} else if !extinguished {
				allExtinguished = false
			}
		}
	}

	if !foundCell {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy vùng hiểm họa tại tọa độ này"})
		return
	}

	// Update metadata
	updatedMeta, _ := json.Marshal(meta)
	_, err = tx.Exec(`
		UPDATE tribulation_events SET hazard_metadata = $1 WHERE id = $2
	`, updatedMeta, eventID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi lưu trạng thái hiểm họa", err))
		return
	}

	transitioned := false
	if allExtinguished {
		// All environmental hazards eliminated -> transition to AFTERMATH and drop spoils
		if err := GenerateTribulationSpoilsAndFalloutTx(tx, eventID.String(), apertureID, playerID.String(), path, rank, tribulationType); err != nil {
			httperr.Respond(c, httperr.Internal("Lỗi sinh kiếp tài hậu kiếp", err))
			return
		}
		transitioned = true
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Commit thất bại", err))
		return
	}

	msg := fmt.Sprintf("Thi triển Tiên Đạo Sát Chiêu hóa giải thành công vùng hiểm họa tại tọa độ (%d, %d)!", req.X, req.Y)
	if transitioned {
		msg += " Toàn bộ thiên tai đã bị dập tắt! Tiên Khiếu bước vào giai đoạn [DƯ BA & THU HOẠCH KIẾP TÀI]."
	}

	c.JSON(http.StatusOK, gin.H{
		"message":                msg,
		"cell_extinguished":      true,
		"all_extinguished":       allExtinguished,
		"transitioned_aftermath": transitioned,
	})
}

// CaptureBeast handles Enslavement Path capture of tribulation beasts when HP <= 30%
func CaptureBeast(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	eventIDStr := c.Param("eventId")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã đợt tai kiếp không hợp lệ"})
		return
	}

	var req struct {
		EnemyID string `json:"enemy_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã quái không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database transaction failure", err))
		return
	}
	defer tx.Rollback()

	// 1. Check event ownership, active status and stage
	var evStatus, evStage, apID, pPath, tType string
	var pRank int
	err = tx.QueryRow(`
		SELECT ev.status, ev.stage, ev.aperture_id, ev.path, ev.tribulation_type, p.rank
		FROM tribulation_events ev
		JOIN players p ON p.id = ev.player_id
		WHERE ev.id = $1 AND ev.player_id = $2
	`, eventID, playerID).Scan(&evStatus, &evStage, &apID, &pPath, &tType, &pRank)
	if err == sql.ErrNoRows || evStatus != "ACTIVE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đợt tai kiếp không tồn tại hoặc đã kết thúc"})
		return
	} else if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi kiểm tra tai kiếp", err))
		return
	}

	if evStage != "ORDEAL_ACTIVE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tai kiếp chưa bước vào giai đoạn tác chiến hoặc đã kết thúc"})
		return
	}

	// 2. Lock enemy row for update
	var hp, maxHP float64
	var enemyName, enemyPath string
	err = tx.QueryRow(`
		SELECT hp, max_hp, name, path
		FROM tribulation_enemies
		WHERE id = $1 AND event_id = $2 AND is_dead = false
		FOR UPDATE
	`, req.EnemyID, eventID).Scan(&hp, &maxHP, &enemyName, &enemyPath)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quái thú không tồn tại hoặc đã bị tiêu diệt"})
		return
	} else if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi kiểm tra quái thú", err))
		return
	}

	if hp > maxHP*0.3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quái thú còn quá khỏe (HP > 30%)! Phải đánh suy yếu trước khi dùng Nô Đạo Cổ Trùng thu phục."})
		return
	}

	// 3. Atomically mark enemy as captured (is_dead = true)
	res, err := tx.Exec(`
		UPDATE tribulation_enemies SET is_dead = true WHERE id = $1 AND is_dead = false
	`, req.EnemyID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi cập nhật quái thú", err))
		return
	}
	if rowsAff, _ := res.RowsAffected(); rowsAff == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quái thú này đã bị tiêu diệt hoặc thu phục trước đó"})
		return
	}

	// 4. Reward beast material based on enemy path
	drop := getBeastCaptureDrop(enemyPath, pRank)
	if InventorySvc != nil {
		_ = InventorySvc.AddItem(tx, playerID.String(), drop.ItemID, "materials", int64(drop.Quantity))
	}

	// 5. Check if all enemies in this event are now dead
	var remainingEnemies int
	_ = tx.QueryRow(`
		SELECT COUNT(1) FROM tribulation_enemies WHERE event_id = $1 AND is_dead = false
	`, eventID).Scan(&remainingEnemies)

	transitioned := false
	if remainingEnemies == 0 {
		_ = GenerateTribulationSpoilsAndFalloutTx(tx, eventID.String(), apID, playerID.String(), pPath, pRank, tType)
		transitioned = true
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Commit thất bại", err))
		return
	}

	msg := fmt.Sprintf("Thu phục thành công quái thú [%s] bằng Nô Đạo Tiên Thuật! Thu được %dx [%s].", enemyName, drop.Quantity, drop.ItemName)
	if transitioned {
		msg += " Toàn bộ thú triều đã bị đẩy lùi! Tiên Khiếu bước vào giai đoạn [DƯ BA & THU HOẠCH KIẾP TÀI]."
	}

	c.JSON(http.StatusOK, gin.H{
		"message":                msg,
		"captured":               true,
		"loot":                   drop.ItemName,
		"remaining_enemies":      remainingEnemies,
		"transitioned_aftermath": transitioned,
	})
}

// ClaimTribulationSpoil claims a single spoils node on the grid
func ClaimTribulationSpoil(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	eventIDStr := c.Param("eventId")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã đợt tai kiếp không hợp lệ"})
		return
	}

	var req struct {
		SpoilID string `json:"spoil_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã kiếp tài không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database transaction failure", err))
		return
	}
	defer tx.Rollback()

	// Atomically claim the spoil item
	var itemID, itemName string
	var quantity int
	err = tx.QueryRow(`
		UPDATE aperture_tribulation_spoils
		SET claimed = true
		WHERE id = $1 AND event_id = $2 AND player_id = $3 AND claimed = false
		RETURNING item_id, item_name, quantity
	`, req.SpoilID, eventID, playerID).Scan(&itemID, &itemName, &quantity)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Điểm Kiếp Tài này đã được thu hoạch hoặc không tồn tại"})
		return
	} else if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi thu hoạch kiếp tài", err))
		return
	}

	if InventorySvc != nil {
		if err := InventorySvc.AddItem(tx, playerID.String(), itemID, "materials", int64(quantity)); err != nil {
			httperr.Respond(c, httperr.Internal("Lỗi nhập kho kiếp tài", err))
			return
		}
	}

	// Check if all spoils claimed
	var remainingSpoils int
	_ = tx.QueryRow(`
		SELECT COUNT(1) FROM aperture_tribulation_spoils WHERE event_id = $1 AND claimed = false
	`, eventID).Scan(&remainingSpoils)

	if remainingSpoils == 0 {
		_, _ = tx.Exec(`UPDATE tribulation_events SET spoils_collected = true WHERE id = $1`, eventID)
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Commit thất bại", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          fmt.Sprintf("Thu hoạch thành công %dx [%s] vào túi đồ!", quantity, itemName),
		"item_id":          itemID,
		"quantity":         quantity,
		"remaining_spoils": remainingSpoils,
	})
}

// ClaimAllTribulationSpoils allows variant humans to gather all spoils at once
func ClaimAllTribulationSpoils(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	eventIDStr := c.Param("eventId")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã đợt tai kiếp không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database transaction failure", err))
		return
	}
	defer tx.Rollback()

	// Atomically mark all unclaimed spoils as claimed and return their details
	rows, err := tx.Query(`
		UPDATE aperture_tribulation_spoils
		SET claimed = true
		WHERE event_id = $1 AND player_id = $2 AND claimed = false
		RETURNING id, item_id, item_name, quantity
	`, eventID, playerID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi truy vấn kiếp tài", err))
		return
	}
	defer rows.Close()

	type SpoilItem struct {
		ID       string
		ItemID   string
		ItemName string
		Quantity int
	}
	items := []SpoilItem{}
	for rows.Next() {
		var it SpoilItem
		if err := rows.Scan(&it.ID, &it.ItemID, &it.ItemName, &it.Quantity); err == nil {
			items = append(items, it)
		}
	}

	if len(items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không còn Kiếp Tài nào chưa thu hoạch"})
		return
	}

	if InventorySvc != nil {
		for _, it := range items {
			_ = InventorySvc.AddItem(tx, playerID.String(), it.ItemID, "materials", int64(it.Quantity))
		}
	}

	_, _ = tx.Exec(`UPDATE tribulation_events SET spoils_collected = true WHERE id = $1`, eventID)

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Commit thất bại", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       fmt.Sprintf("Phái bộ tộc Dị Nhân thu gom thành công toàn bộ %d điểm Kiếp Tài!", len(items)),
		"claimed_count": len(items),
	})
}

// CleanseSoilContamination purifies a contaminated soil cell back to normal
func CleanseSoilContamination(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tọa độ không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database transaction failure", err))
		return
	}
	defer tx.Rollback()

	// 1. Check immortal essence >= 50
	const cleanseCost = 50
	var essence int
	var apertureID string
	err = tx.QueryRow(`
		SELECT id, immortal_essence
		FROM immortal_apertures
		WHERE player_id = $1
		FOR UPDATE
	`, playerID).Scan(&apertureID, &essence)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi kiểm tra Tiên Khiếu", err))
		return
	}

	if essence < cleanseCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Tiên Nguyên Thạch để thanh tẩy! (Cần %d, hiện có %d)", cleanseCost, essence)})
		return
	}

	// 2. Check if cell is contaminated
	var contamID string
	err = tx.QueryRow(`
		SELECT id FROM aperture_soil_contaminations
		WHERE aperture_id = $1 AND x = $2 AND y = $3
	`, apertureID, req.X, req.Y).Scan(&contamID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ô đất này không bị ô nhiễm Đạo Ngân"})
		return
	} else if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi kiểm tra ô nhiễm", err))
		return
	}

	// Deduct essence and remove contamination
	_, _ = tx.Exec(`UPDATE immortal_apertures SET immortal_essence = immortal_essence - $1 WHERE id = $2`, cleanseCost, apertureID)
	_, _ = tx.Exec(`DELETE FROM aperture_soil_contaminations WHERE id = $1`, contamID)

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Commit thất bại", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Thanh tẩy thành công ô đất tại (%d, %d)! Thổ nhưỡng đã hồi phục màu mỡ.", req.X, req.Y),
		"x":       req.X,
		"y":       req.Y,
	})
}

// PivotSoilContamination turns contaminated soil into a brand-new specialty resource mine
func PivotSoilContamination(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tọa độ không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database transaction failure", err))
		return
	}
	defer tx.Rollback()

	// 1. Check immortal essence >= 120
	const pivotCost = 120
	var essence int
	var apertureID string
	err = tx.QueryRow(`
		SELECT id, immortal_essence
		FROM immortal_apertures
		WHERE player_id = $1
		FOR UPDATE
	`, playerID).Scan(&apertureID, &essence)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi kiểm tra Tiên Khiếu", err))
		return
	}

	if essence < pivotCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Tiên Nguyên Thạch để Biến Nguy Thành Cơ! (Cần %d, hiện có %d)", pivotCost, essence)})
		return
	}

	// 2. Check contamination
	var contamID, contamType, contamPath string
	err = tx.QueryRow(`
		SELECT id, contam_type, path FROM aperture_soil_contaminations
		WHERE aperture_id = $1 AND x = $2 AND y = $3
	`, apertureID, req.X, req.Y).Scan(&contamID, &contamType, &contamPath)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ô đất này không bị ô nhiễm Đạo Ngân"})
		return
	} else if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi kiểm tra ô nhiễm", err))
		return
	}

	// 3. Check collision with existing resource points
	var existingPoints int
	err = tx.QueryRow(`
		SELECT COUNT(1) FROM aperture_resource_points
		WHERE aperture_id = $1 AND x = $2 AND y = $3 AND state != 'DESTROYED'
	`, apertureID, req.X, req.Y).Scan(&existingPoints)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi kiểm tra điểm tài nguyên", err))
		return
	}
	if existingPoints > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tọa độ này đã có điểm tài nguyên tồn tại, không thể kiến tạo thêm mỏ mới"})
		return
	}

	// 4. Convert into specialty resource point
	mineType := "fire_crystal_mine"
	mineName := "Mỏ Viêm Tinh Khoáng Mạch"
	switch contamType {
	case "ice":
		mineType = "ice_pool"
		mineName = "Hàn Băng Thần Trì"
	case "poison":
		mineType = "toxic_herb_field"
		mineName = "U Độc Dược Điền"
	case "lightning":
		mineType = "lightning_peak"
		mineName = "Lôi Phong Dục Lôi Đài"
	}

	pointID := uuid.New()
	_, err = tx.Exec(`
		INSERT INTO aperture_resource_points (id, scope, aperture_id, point_type, name, path, grade, point_class, level, vitality, x, y, state)
		VALUES ($1, 'APERTURE', $2, $3, $4, $5, 'LOW', 'NORMAL', 1, 100, $6, $7, 'ACTIVE')
	`, pointID, apertureID, mineType, mineName, contamPath, req.X, req.Y)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi tạo mỏ tài nguyên mới", err))
		return
	}

	// Deduct essence and remove contamination
	_, _ = tx.Exec(`UPDATE immortal_apertures SET immortal_essence = immortal_essence - $1 WHERE id = $2`, pivotCost, apertureID)
	_, _ = tx.Exec(`DELETE FROM aperture_soil_contaminations WHERE id = $1`, contamID)

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Commit thất bại", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   fmt.Sprintf("Biến Nguy Thành Cơ thành công! Đã chuyển hóa ô đất thành [%s]!", mineName),
		"point_id":  pointID.String(),
		"mine_name": mineName,
		"x":         req.X,
		"y":         req.Y,
	})
}

// TribulationSettlementResult holds the outcome of a finalized tribulation
type TribulationSettlementResult struct {
	Rewards         map[string]int64
	UpdatedDaoMarks models.DaoMarksMap
	DidBreakthrough bool
	BreakthroughMsg string
	Message         string
	State           string
}

// finalizeTribulationSettlementTx executes the shared successful settlement logic:
// awards Dao marks, increments calamity counts, checks breakthrough and path mastery requirements,
// resets aperture HP to max and state to STABLE, marks event SUCCESS, and updates human qi.
func finalizeTribulationSettlementTx(tx *sql.Tx, eventID uuid.UUID, playerID uuid.UUID) (*TribulationSettlementResult, error) {
	// 1. Lock the event before settlement
	var ev models.TribulationEvent
	err := tx.QueryRow(`
		SELECT id, aperture_id, player_id, tribulation_type, path, reward_dao_marks, started_at, status, COALESCE(stage, 'ORDEAL_ACTIVE')
		FROM tribulation_events
		WHERE id = $1 AND player_id = $2
		FOR UPDATE
	`, eventID, playerID).Scan(&ev.ID, &ev.ApertureID, &ev.PlayerID, &ev.TribulationType, &ev.Path, &ev.RewardDaoMarks, &ev.StartedAt, &ev.Status, &ev.Stage)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("đợt tai kiếp này đã được giải quyết hoặc không tồn tại")
	} else if err != nil {
		return nil, fmt.Errorf("lỗi truy vấn tai kiếp: %w", err)
	}
	if ev.Status != "ACTIVE" {
		return nil, fmt.Errorf("tai kiếp này đã được kết toán trước đó")
	}

	// 2. Lock and fetch aperture
	var ap models.ImmortalAperture
	err = tx.QueryRow(`
		SELECT id, aperture_hp, max_aperture_hp, next_tribulation_year,
			rank_start_year, earth_calamity_count, heavenly_tribulation_count, grand_tribulation_count, myriad_tribulation_count, chaos_calamity_count, internal_years_passed
		FROM immortal_apertures WHERE id = $1
		FOR UPDATE
	`, ev.ApertureID).Scan(&ap.ID, &ap.ApertureHP, &ap.MaxApertureHP, &ap.NextTribulationYear,
		&ap.RankStartYear, &ap.EarthCalamityCount, &ap.HeavenlyTribulationCount, &ap.GrandTribulationCount, &ap.MyriadTribulationCount, &ap.ChaosCalamityCount, &ap.InternalYearsPassed)
	if err != nil {
		return nil, fmt.Errorf("lỗi lấy thông tin Tiên Khiếu: %w", err)
	}

	// 3. Distribute Dao marks
	rewards := distributeDaoMarks(ev.Path, ev.RewardDaoMarks)
	updatedDaoMarks, errDao := addMultipleDaoMarksTx(tx, playerID.String(), rewards)
	if errDao != nil {
		return nil, fmt.Errorf("lỗi cộng Đạo Ngân: %w", errDao)
	}

	// 4. Update calamity counts
	switch ev.TribulationType {
	case "EARTH_CALAMITY":
		ap.EarthCalamityCount++
	case "HEAVENLY_TRIBULATION":
		ap.HeavenlyTribulationCount++
	case "GRAND_TRIBULATION":
		ap.GrandTribulationCount++
	case "MYRIAD_TRIBULATION":
		ap.MyriadTribulationCount++
	case "CHAOS_CALAMITY", "CHAOTIC_TRIBULATION":
		ap.ChaosCalamityCount++
	}

	var playerRank int
	var username string
	_ = tx.QueryRow(`SELECT rank, username FROM players WHERE id = $1`, playerID).Scan(&playerRank, &username)

	// Recalculate next tribulation year based on rank & counters
	nextYear, _ := models.GetNextTribulationInfo(playerRank, ap.RankStartYear, ap.InternalYearsPassed)
	ap.NextTribulationYear = guardNextTribulationYear(nextYear, ap.InternalYearsPassed)

	didBreakthrough := false
	breakthroughMsg := ""

	// Check breakthrough conditions
	canBreak, newRank := models.CheckImmortalBreakthrough(playerRank, ap.EarthCalamityCount, ap.HeavenlyTribulationCount, ap.GrandTribulationCount, ap.MyriadTribulationCount, ap.ChaosCalamityCount)
	if canBreak {
		var pathMasteries models.PathMasteriesMap
		_ = tx.QueryRow(`SELECT COALESCE(path_masteries, '{}'::jsonb) FROM players WHERE id = $1`, playerID).Scan(&pathMasteries)
		reqLevel := gamefx.PathMasteryLevelRequiredForRank(newRank)
		if !gamefx.HasPathMasteryAtLeast(pathMasteries, reqLevel) {
			blockedMsg := fmt.Sprintf(" Chưa đủ điều kiện đột phá lên %s: cần ít nhất 1 lưu phái đạt %s (%d điểm).", models.GetRankName(newRank), gamefx.PathMasteryLevelName(reqLevel), gamefx.PathMasteryPointsForLevel(reqLevel))
			_, _ = tx.Exec(`INSERT INTO messages (sender_id, receiver_id, content, channel) VALUES ($1, $1, $2, 'system')`, playerID, "Đột phá bị chặn: "+blockedMsg)
			breakthroughMsg = blockedMsg
		} else {
			_, _ = tx.Exec(`UPDATE players SET rank = $1 WHERE id = $2`, newRank, playerID)
			ap.RankStartYear = math.Floor(ap.InternalYearsPassed)
			ap.EarthCalamityCount = 0
			ap.HeavenlyTribulationCount = 0
			ap.GrandTribulationCount = 0
			ap.MyriadTribulationCount = 0
			ap.ChaosCalamityCount = 0
			didBreakthrough = true
			breakthroughMsg = fmt.Sprintf(" Đột phá! Đạo hữu đã tiến vào %s!", models.GetRankName(newRank))

			if newRank == 7 || newRank == 8 || newRank == 9 {
				_ = SendSystemMessage("world", sql.NullString{}, fmt.Sprintf("Chúc mừng đạo hữu %s đã đột phá thành công lên %s!", username, models.GetRankName(newRank)))
				auditlog.Log(tx, auditlog.Entry{
					PlayerID:       &playerID,
					Action:         "player.breakthrough_success",
					EntityType:     "player",
					EntityID:       playerID.String(),
					RequestSummary: fmt.Sprintf("Thăng rank %d (%s) thành công", newRank, models.GetRankName(newRank)),
					Success:        true,
				})
			}

			nextYear, _ = models.GetNextTribulationInfo(newRank, ap.RankStartYear, ap.InternalYearsPassed)
			ap.NextTribulationYear = guardNextTribulationYear(nextYear, ap.InternalYearsPassed)
		}
	}

	now := time.Now()
	ap.ApertureHP = ap.MaxApertureHP
	ap.State = "STABLE"

	// 5. Update event status = SUCCESS, stage = RESOLVED
	_, err = tx.Exec(`
		UPDATE tribulation_events
		SET status = 'SUCCESS', stage = 'RESOLVED', aperture_hp_after = $1, ended_at = $2
		WHERE id = $3
	`, ap.ApertureHP, now, ev.ID)
	if err != nil {
		return nil, fmt.Errorf("lỗi cập nhật sự kiện tai kiếp: %w", err)
	}

	// 6. Clean up any remaining enemies
	_, _ = tx.Exec(`UPDATE tribulation_enemies SET is_dead = true WHERE event_id = $1`, ev.ID)

	// 7. Update aperture
	_, err = tx.Exec(`
		UPDATE immortal_apertures 
		SET state = $1, next_tribulation_year = $2, aperture_hp = $3,
			rank_start_year = $4, earth_calamity_count = $5, heavenly_tribulation_count = $6,
			grand_tribulation_count = $7, myriad_tribulation_count = $8, chaos_calamity_count = $9,
			updated_at = $10
		WHERE id = $11
	`, ap.State, ap.NextTribulationYear, ap.ApertureHP,
		ap.RankStartYear, ap.EarthCalamityCount, ap.HeavenlyTribulationCount,
		ap.GrandTribulationCount, ap.MyriadTribulationCount, ap.ChaosCalamityCount, now, ap.ID)
	if err != nil {
		return nil, fmt.Errorf("lỗi cập nhật Tiên Khiếu: %w", err)
	}

	// 8. Human Qi +10
	_, err = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 10, $1) WHERE id = $2", MaxHumanQi, playerID)
	if err != nil {
		return nil, fmt.Errorf("lỗi cập nhật Nhân Khí: %w", err)
	}

	if didBreakthrough {
		_, _ = tx.Exec(`
			INSERT INTO messages (sender_id, receiver_id, content, channel)
			VALUES ('11111111-1111-1111-1111-111111111111', $1, $2, 'private')
		`, playerID, breakthroughMsg)
	}

	rewardStrings := []string{}
	for path, amount := range rewards {
		rewardStrings = append(rewardStrings, fmt.Sprintf("%d Đạo Ngân hệ %s", amount, strings.ToUpper(path)))
	}
	rewardText := strings.Join(rewardStrings, ", ")
	msg := fmt.Sprintf("Chúc mừng đạo hữu vượt kiếp nạn thành công! Khai quang nhận %s. Tiên Khiếu của bạn đã ổn định trở lại.%s", rewardText, breakthroughMsg)

	return &TribulationSettlementResult{
		Rewards:         rewards,
		UpdatedDaoMarks: updatedDaoMarks,
		DidBreakthrough: didBreakthrough,
		BreakthroughMsg: breakthroughMsg,
		Message:         msg + " Nhân Khí +10.",
		State:           ap.State,
	}, nil
}

// FinalizeTribulationAftermath completes the aftermath phase and restores stability
func FinalizeTribulationAftermath(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	eventIDStr := c.Param("eventId")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã đợt tai kiếp không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database transaction failure", err))
		return
	}
	defer tx.Rollback()

	// Verify stage is AFTERMATH
	var stage string
	err = tx.QueryRow(`
		SELECT COALESCE(stage, 'ORDEAL_ACTIVE')
		FROM tribulation_events
		WHERE id = $1 AND player_id = $2 AND status = 'ACTIVE'
	`, eventID, playerID).Scan(&stage)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đợt tai kiếp không tồn tại hoặc đã kết thúc"})
		return
	} else if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi truy vấn tai kiếp", err))
		return
	}

	if stage != "AFTERMATH" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tai kiếp chưa bước vào giai đoạn kết toán hậu độ kiếp"})
		return
	}

	res, err := finalizeTribulationSettlementTx(tx, eventID, playerID)
	if err != nil {
		httperr.Respond(c, httperr.Internal(err.Error(), err))
		return
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Commit thất bại", err))
		return
	}

	// Check if the player is immediately due for another tribulation
	var apertureID string
	_ = db.DB.QueryRow(`SELECT id FROM immortal_apertures WHERE player_id = $1`, playerID).Scan(&apertureID)
	if apertureID != "" {
		_ = checkAndTriggerSingleAperture(apertureID)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   res.Message,
		"stage":     "RESOLVED",
		"status":    "SUCCESS",
		"state":     res.State,
		"dao_marks": res.UpdatedDaoMarks,
	})
}
