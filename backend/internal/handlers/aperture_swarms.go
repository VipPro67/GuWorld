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
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/playerstate"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// GetApertureGuSwarms: GET /api/immortal-aperture/swarms
// Trả về danh sách toàn bộ các Đàn Phàm Cổ (Rank 1-5) sinh sống trong Tiên Khiếu của Cổ Tiên.
func GetApertureGuSwarms(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var rank int
	var apertureID string
	err := db.DB.QueryRow(`
		SELECT p.rank, COALESCE(a.id::text, '')
		FROM players p
		LEFT JOIN immortal_apertures a ON a.player_id = p.id
		WHERE p.id = $1
	`, playerID).Scan(&rank, &apertureID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ sư"})
		return
	}

	if rank < 6 || apertureID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa thăng tiên (Rank 6+), chưa có Tiên Khiếu."})
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT s.id, s.aperture_id, s.gu_id, s.gu_rank, s.quantity, s.created_at, s.updated_at,
		       COALESCE(gd.name, s.gu_id), COALESCE(gd.type, ''), COALESCE(gd.path, ARRAY[]::varchar[]),
		       COALESCE(gd.description, ''), COALESCE(gd.food, '')
		FROM aperture_gu_swarms s
		LEFT JOIN gu_dictionary gd ON gd.id = s.gu_id AND gd.rank = s.gu_rank
		WHERE s.aperture_id = $1 AND s.quantity > 0
		ORDER BY s.gu_rank DESC, s.quantity DESC
	`, apertureID)
	if err != nil {
		logger.S.Errorf("GetApertureGuSwarms query error: %v", err)
		httperr.Respond(c, httperr.Internal("Lỗi truy vấn Đàn Cổ", err))
		return
	}
	defer rows.Close()

	swarms := make([]models.ApertureGuSwarmView, 0)
	for rows.Next() {
		var sw models.ApertureGuSwarmView
		var paths []string
		if err := rows.Scan(
			&sw.ID, &sw.ApertureID, &sw.GuID, &sw.GuRank, &sw.Quantity, &sw.CreatedAt, &sw.UpdatedAt,
			&sw.GuName, &sw.GuType, pq.Array(&paths), &sw.Description, &sw.Food,
		); err != nil {
			logger.S.Errorf("GetApertureGuSwarms scan error: %v", err)
			httperr.Respond(c, httperr.Internal("Lỗi đọc dữ liệu Đàn Cổ", err))
			return
		}
		sw.Paths = paths
		swarms = append(swarms, sw)
	}

	c.JSON(http.StatusOK, gin.H{
		"aperture_id": apertureID,
		"swarms":      swarms,
	})
}

type MaterializeGuRequest struct {
	GuID   string `json:"gu_id" binding:"required"`
	GuRank int    `json:"gu_rank" binding:"required"`
	Equip  bool   `json:"equip"`
}

// MaterializeGuFromSwarm: POST /api/immortal-aperture/swarms/materialize
// Cụ thể hóa 1 Phàm Cổ từ Đàn Tiên Khiếu thành thực thể cá thể trong player_gu (và trang bị nếu chọn).
func MaterializeGuFromSwarm(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req MaterializeGuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu yêu cầu không hợp lệ"})
		return
	}

	if req.GuRank < 1 || req.GuRank > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ có thể rút Phàm Cổ (Rank 1-5) từ Đàn Cổ"})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi khởi tạo giao dịch", err))
		return
	}
	defer tx.Rollback()

	var rank int
	var apertureID string
	var equippedGusJSON []byte
	err = tx.QueryRow(`
		SELECT p.rank, COALESCE(a.id::text, ''), COALESCE(p.equipped_gus, '[]'::jsonb)
		FROM players p
		LEFT JOIN immortal_apertures a ON a.player_id = p.id
		WHERE p.id = $1
		FOR UPDATE OF p
	`, playerID).Scan(&rank, &apertureID, &equippedGusJSON)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ sư"})
		return
	}

	if rank < 6 || apertureID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa có Tiên Khiếu"})
		return
	}

	// 1. Kiểm tra số lượng trong đàn và khóa hàng
	var currentQty int64
	err = tx.QueryRow(`
		SELECT quantity
		FROM aperture_gu_swarms
		WHERE aperture_id = $1 AND gu_id = $2 AND gu_rank = $3
		FOR UPDATE
	`, apertureID, req.GuID, req.GuRank).Scan(&currentQty)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Loài Cổ này chưa được nuôi dưỡng trong Tiên Khiếu"})
			return
		}
		httperr.Respond(c, httperr.Internal("Lỗi kiểm tra số lượng Đàn Cổ", err))
		return
	}

	if currentQty < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Số lượng Cổ trong đàn không đủ để rút"})
		return
	}

	// 2. Nếu chọn trang bị, kiểm tra giới hạn MaxEquippedGus
	var equippedGus models.EquippedGusList
	if err := json.Unmarshal(equippedGusJSON, &equippedGus); err != nil {
		equippedGus = models.EquippedGusList{}
	}

	if req.Equip && len(equippedGus) >= db.MaxEquippedGus {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không Khiếu đã trang bị tối đa %d Cổ trùng", db.MaxEquippedGus)})
		return
	}

	// 3. Khấu trừ 1 con trong Đàn Cổ
	var remainingQty int64
	err = tx.QueryRow(`
		UPDATE aperture_gu_swarms
		SET quantity = quantity - 1, updated_at = NOW()
		WHERE aperture_id = $1 AND gu_id = $2 AND gu_rank = $3 AND quantity >= 1
		RETURNING quantity
	`, apertureID, req.GuID, req.GuRank).Scan(&remainingQty)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể khấu trừ Cổ từ đàn"})
		return
	}

	// 4. Sinh Instance mới vào player_gu
	newInstanceID := uuid.New().String()
	now := time.Now()
	_, err = tx.Exec(`
		INSERT INTO player_gu (
			id, player_id, gu_id, gu_rank, rank, level, hunger, refinement_progress,
			durability, state, location, is_locked, equipped, is_vital, created_at, updated_at, last_fed_at
		) VALUES (
			$1, $2, $3, $4, $4, 1, 100, 100.0,
			100.0, 'ACTIVE', 'APERTURE', false, $5, false, $6, $6, $6
		)
	`, newInstanceID, playerID, req.GuID, req.GuRank, req.Equip, now)
	if err != nil {
		logger.S.Errorf("MaterializeGuFromSwarm insert player_gu error: %v", err)
		httperr.Respond(c, httperr.Internal("Lỗi tạo thực thể Cổ cá thể", err))
		return
	}

	// 5. Cập nhật mảng equipped_gus nếu trang bị
	if req.Equip {
		equippedGus = append(equippedGus, newInstanceID)
		newEquippedJSON, _ := json.Marshal(equippedGus)
		if _, err := tx.Exec(`UPDATE players SET equipped_gus = $1 WHERE id = $2`, newEquippedJSON, playerID); err != nil {
			httperr.Respond(c, httperr.Internal("Lỗi cập nhật danh sách trang bị", err))
			return
		}
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi cam kết giao dịch", err))
		return
	}

	// Nạp lại snapshot để phát sự kiện
	player, err := playerstate.Load(db.DB, playerID, playerstate.Options{Gu: playerstate.GuFull})
	if err == nil {
		PublishPlayerSnapshot(&player, PlayerEventUpdated, PlayerDomain, GuInventoryDomain, CombatStatsDomain)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Đã rút Cổ trùng từ Tiên Khiếu thành công!",
		"instance_id":     newInstanceID,
		"gu_id":           req.GuID,
		"gu_rank":         req.GuRank,
		"equipped":        req.Equip,
		"remaining_swarm": remainingQty,
	})
}

type DematerializeGuRequest struct {
	InstanceID string `json:"instance_id" binding:"required"`
}

// DematerializeGuToSwarm: POST /api/immortal-aperture/swarms/dematerialize
// Giải phóng một Phàm Cổ lành lặn trong player_gu trở về Đàn Tiên Khiếu (xóa cá thể, tăng số lượng đàn).
func DematerializeGuToSwarm(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req DematerializeGuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu yêu cầu không hợp lệ"})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi khởi tạo giao dịch", err))
		return
	}
	defer tx.Rollback()

	var rank int
	var apertureID string
	var equippedGusJSON []byte
	err = tx.QueryRow(`
		SELECT p.rank, COALESCE(a.id::text, ''), COALESCE(p.equipped_gus, '[]'::jsonb)
		FROM players p
		LEFT JOIN immortal_apertures a ON a.player_id = p.id
		WHERE p.id = $1
		FOR UPDATE OF p
	`, playerID).Scan(&rank, &apertureID, &equippedGusJSON)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ sư"})
		return
	}

	if rank < 6 || apertureID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa có Tiên Khiếu"})
		return
	}

	// 1. Khóa và kiểm tra cá thể Cổ trong player_gu
	var (
		guID       string
		guRank     int
		durability float64
		isVital    bool
		isLocked   bool
		equipped   bool
	)
	err = tx.QueryRow(`
		SELECT gu_id, gu_rank, durability, is_vital, is_locked, equipped
		FROM player_gu
		WHERE id = $1 AND player_id = $2
		FOR UPDATE
	`, req.InstanceID, playerID).Scan(&guID, &guRank, &durability, &isVital, &isLocked, &equipped)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ trùng trong túi thân"})
			return
		}
		httperr.Respond(c, httperr.Internal("Lỗi kiểm tra Cổ trùng", err))
		return
	}

	// 2. Chốt chặn an toàn
	if guRank > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tiên Cổ (Rank 6+) là độc nhất thiên địa, không thể hòa nhập đàn!"})
		return
	}
	if isVital {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bản Mệnh Cổ gắn liền sinh mệnh, không thể giải phóng vào đàn!"})
		return
	}
	if isLocked {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng đang bị khóa trong phiên luyện hoặc giao dịch!"})
		return
	}
	reserved, reservationErr := guHasActiveProductReservation(tx, req.InstanceID)
	if reservationErr != nil {
		httperr.Respond(c, httperr.Internal("Không thể kiểm tra Cổ đang được công trình giữ chỗ", reservationErr))
		return
	}
	if reserved {
		c.JSON(http.StatusConflict, gin.H{"error": "Cổ đang được Sát Chiêu hoặc Tiên Cổ Ốc active giữ chỗ; hãy tháo công trình trước"})
		return
	}
	if durability < 100.0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Cổ trùng đang bị thương tổn (độ bền %.1f/100)! Cần tu bổ lành lặn trước khi thả vào Tiên Khiếu.", durability),
		})
		return
	}

	// 3. Nếu đang trang bị, gỡ khỏi equipped_gus
	if equipped {
		var equippedGus models.EquippedGusList
		if err := json.Unmarshal(equippedGusJSON, &equippedGus); err == nil {
			newEquipped := make(models.EquippedGusList, 0, len(equippedGus))
			for _, id := range equippedGus {
				if id != req.InstanceID {
					newEquipped = append(newEquipped, id)
				}
			}
			newJSON, _ := json.Marshal(newEquipped)
			if _, err := tx.Exec(`UPDATE players SET equipped_gus = $1 WHERE id = $2`, newJSON, playerID); err != nil {
				httperr.Respond(c, httperr.Internal("Lỗi gỡ Cổ trang bị", err))
				return
			}
		}
	}

	// 4. Xóa bản ghi cá thể
	if _, err := tx.Exec(`DELETE FROM player_gu WHERE id = $1 AND player_id = $2`, req.InstanceID, playerID); err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi xóa bản ghi Cổ cá thể", err))
		return
	}

	// 5. Cộng dồn số lượng vào Đàn Cổ
	var newSwarmQty int64
	err = tx.QueryRow(`
		INSERT INTO aperture_gu_swarms (id, aperture_id, gu_id, gu_rank, quantity, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, 1, NOW(), NOW())
		ON CONFLICT (aperture_id, gu_id, gu_rank)
		DO UPDATE SET quantity = aperture_gu_swarms.quantity + 1, updated_at = NOW()
		RETURNING quantity
	`, apertureID, guID, guRank).Scan(&newSwarmQty)
	if err != nil {
		logger.S.Errorf("DematerializeGuToSwarm upsert swarm error: %v", err)
		httperr.Respond(c, httperr.Internal("Lỗi cộng dồn vào Đàn Cổ", err))
		return
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi cam kết giao dịch", err))
		return
	}

	// Nạp lại snapshot để phát sự kiện
	player, err := playerstate.Load(db.DB, playerID, playerstate.Options{Gu: playerstate.GuFull})
	if err == nil {
		PublishPlayerSnapshot(&player, PlayerEventUpdated, PlayerDomain, GuInventoryDomain, CombatStatsDomain)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Đã hòa nhập Cổ trùng vào Đàn Tiên Khiếu thành công!",
		"gu_id":         guID,
		"gu_rank":       guRank,
		"new_swarm_qty": newSwarmQty,
	})
}

// DepositAllHealthyGus: POST /api/immortal-aperture/swarms/deposit-all
// Thu gom nhanh toàn bộ Phàm Cổ lành lặn trong túi đồ cá nhân vào Đàn Tiên Khiếu (1-click batch deposit).
func DepositAllHealthyGus(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi khởi tạo giao dịch", err))
		return
	}
	defer tx.Rollback()

	var rank int
	var apertureID string
	err = tx.QueryRow(`
		SELECT p.rank, COALESCE(a.id::text, '')
		FROM players p
		LEFT JOIN immortal_apertures a ON a.player_id = p.id
		WHERE p.id = $1
	`, playerID).Scan(&rank, &apertureID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ sư"})
		return
	}

	if rank < 6 || apertureID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa có Tiên Khiếu"})
		return
	}

	// Quét các Phàm Cổ thỏa mãn: rank 1-5, độ bền 100%, không vital, không lock, không equipped
	rows, err := tx.Query(`
		SELECT id, gu_id, gu_rank
		FROM player_gu
		WHERE player_id = $1 AND gu_rank <= 5 AND durability >= 100.0
		  AND is_vital = false AND is_locked = false AND equipped = false
		  AND NOT EXISTS (SELECT 1 FROM player_killer_move_gu_reservations AS reservation WHERE reservation.gu_instance_id=player_gu.id)
		  AND NOT EXISTS (SELECT 1 FROM player_gu_house_core_reservations AS reservation WHERE reservation.gu_instance_id=player_gu.id)
		  AND NOT EXISTS (SELECT 1 FROM teleport_array_gu_reservations AS reservation WHERE reservation.gu_instance_id=player_gu.id)
		FOR UPDATE
	`, playerID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi quét danh sách Cổ lành lặn", err))
		return
	}
	defer rows.Close()

	type eligibleGu struct {
		id     string
		guID   string
		guRank int
	}

	var eligible []eligibleGu
	idsToDelete := make([]string, 0)
	counts := make(map[string]int64) // key: guID_guRank

	for rows.Next() {
		var eg eligibleGu
		if err := rows.Scan(&eg.id, &eg.guID, &eg.guRank); err == nil {
			eligible = append(eligible, eg)
			idsToDelete = append(idsToDelete, eg.id)
			key := fmt.Sprintf("%s_%d", eg.guID, eg.guRank)
			counts[key]++
		}
	}
	rows.Close()

	if len(eligible) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":         "Không có Phàm Cổ nào lành lặn chưa trang bị cần thu hồi.",
			"deposited_count": 0,
		})
		return
	}

	// Cộng dồn vào aperture_gu_swarms
	for key, qty := range counts {
		parts := strings.Split(key, "_")
		if len(parts) != 2 {
			continue
		}
		gID := parts[0]
		var gRank int
		fmt.Sscanf(parts[1], "%d", &gRank)

		_, err := tx.Exec(`
			INSERT INTO aperture_gu_swarms (id, aperture_id, gu_id, gu_rank, quantity, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW(), NOW())
			ON CONFLICT (aperture_id, gu_id, gu_rank)
			DO UPDATE SET quantity = aperture_gu_swarms.quantity + EXCLUDED.quantity, updated_at = NOW()
		`, apertureID, gID, gRank, qty)
		if err != nil {
			logger.S.Errorf("DepositAllHealthyGus upsert error: %v", err)
			httperr.Respond(c, httperr.Internal("Lỗi cập nhật Đàn Cổ", err))
			return
		}
	}

	// Xóa các bản ghi cá thể
	if _, err := tx.Exec(`DELETE FROM player_gu WHERE id = ANY($1::text[])`, pq.Array(idsToDelete)); err != nil {
		logger.S.Errorf("DepositAllHealthyGus delete error: %v", err)
		httperr.Respond(c, httperr.Internal("Lỗi xóa bản ghi Cổ sau thu hồi", err))
		return
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi cam kết giao dịch", err))
		return
	}

	// Nạp lại snapshot để phát sự kiện
	player, err := playerstate.Load(db.DB, playerID, playerstate.Options{Gu: playerstate.GuFull})
	if err == nil {
		PublishPlayerSnapshot(&player, PlayerEventUpdated, PlayerDomain, GuInventoryDomain, CombatStatsDomain)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         fmt.Sprintf("Đã thu hồi thành công %d Phàm Cổ lành lặn vào Đàn Tiên Khiếu!", len(eligible)),
		"deposited_count": len(eligible),
	})
}
