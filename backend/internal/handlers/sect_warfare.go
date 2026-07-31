package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

type SectMineResponse struct {
	ID             int        `json:"id"`
	Name           string     `json:"name"`
	Rarity         string     `json:"rarity"`
	OccupierSectID *string    `json:"occupier_sect_id"`
	OccupierName   *string    `json:"occupier_name"`
	OccupiedAt     *time.Time `json:"occupied_at"`
	ShieldUntil    *time.Time `json:"shield_until"`
	LastClaimedAt  *time.Time `json:"last_claimed_at"`
}

func GetSectMines(c *gin.Context) {
	rows, err := db.DB.Query(`
		SELECT sm.id, sm.name, sm.rarity, sm.occupier_sect_id, s.name, sm.occupied_at, sm.shield_until, sm.last_claimed_at
		FROM sect_mines sm
		LEFT JOIN sects s ON sm.occupier_sect_id = s.id
		ORDER BY sm.id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sect mines"})
		return
	}
	defer rows.Close()

	var mines []SectMineResponse
	for rows.Next() {
		var m SectMineResponse
		var oName sql.NullString
		err = rows.Scan(&m.ID, &m.Name, &m.Rarity, &m.OccupierSectID, &oName, &m.OccupiedAt, &m.ShieldUntil, &m.LastClaimedAt)
		if err == nil {
			if oName.Valid {
				m.OccupierName = &oName.String
			}
			mines = append(mines, m)
		}
	}
	c.JSON(http.StatusOK, mines)
}

func ClaimSectMineRewards(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		MineID int `json:"mine_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var sectID, memberRank string
	err := db.DB.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &memberRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if memberRank != "CHUONG_MON" && memberRank != "TRUONG_LAO" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Trưởng Lão hoặc Chưởng Môn mới có thể nhận thưởng!"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var occupierSectID string
	var lastClaimedAt sql.NullTime
	var rarity string
	err = tx.QueryRow(`SELECT occupier_sect_id, last_claimed_at, rarity FROM sect_mines WHERE id = $1 FOR UPDATE`, req.MineID).Scan(&occupierSectID, &lastClaimedAt, &rarity)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy mỏ khoáng!"})
		return
	}

	if occupierSectID != sectID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tông môn của bạn không chiếm đóng mỏ khoáng này!"})
		return
	}

	now := time.Now()
	var startTime time.Time
	if lastClaimedAt.Valid {
		startTime = lastClaimedAt.Time
	} else {
		var occupiedAt sql.NullTime
		_ = tx.QueryRow(`SELECT occupied_at FROM sect_mines WHERE id = $1`, req.MineID).Scan(&occupiedAt)
		if occupiedAt.Valid {
			startTime = occupiedAt.Time
		} else {
			startTime = now
		}
	}

	elapsedMinutes := now.Sub(startTime).Minutes()
	if elapsedMinutes < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chưa có đủ tài nguyên để nhận (cần ít nhất 1 phút)!"})
		return
	}

	var sectLevel int
	err = tx.QueryRow(`SELECT level FROM sects WHERE id = $1`, sectID).Scan(&sectLevel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get sect level"})
		return
	}

	baseStonesPerMin := 10.0
	switch rarity {
	case "Truyền Thuyết":
		baseStonesPerMin = 100.0
	case "Sử Thi":
		baseStonesPerMin = 50.0
	case "Hiếm":
		baseStonesPerMin = 25.0
	case "Thường":
		baseStonesPerMin = 10.0
	}

	stonesGained := int(elapsedMinutes * baseStonesPerMin * float64(sectLevel))

	if stonesGained <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chưa có đủ tài nguyên để nhận!"})
		return
	}

	_, err = tx.Exec(`UPDATE sects SET funds = funds + $1 WHERE id = $2`, stonesGained, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect funds"})
		return
	}

	_, err = tx.Exec(`UPDATE sect_mines SET last_claimed_at = $1 WHERE id = $2`, now, req.MineID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update mine status"})
		return
	}

	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Nhận thưởng thành công!",
		"stones":  stonesGained,
	})
}

func AttackSectMine(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		MineID int `json:"mine_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var sectID, memberRank string
	err := db.DB.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &memberRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if memberRank == "NGOAI_MON" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Ngoại môn đệ tử không thể tham gia tranh đoạt mỏ khoáng!"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var occupierSectID sql.NullString
	var shieldUntil sql.NullTime
	err = tx.QueryRow(`SELECT occupier_sect_id, shield_until FROM sect_mines WHERE id = $1 FOR UPDATE`, req.MineID).Scan(&occupierSectID, &shieldUntil)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy mỏ khoáng!"})
		return
	}

	if occupierSectID.Valid && occupierSectID.String == sectID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tông môn của bạn đã chiếm đóng mỏ khoáng này rồi!"})
		return
	}

	if shieldUntil.Valid && time.Now().Before(shieldUntil.Time) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mỏ khoáng này đang trong thời gian bảo hộ!"})
		return
	}

	now := time.Now()
	newShieldUntil := now.Add(4 * time.Hour) // 4 hours shield

	_, err = tx.Exec(`
		UPDATE sect_mines 
		SET occupier_sect_id = $1, occupied_at = $2, shield_until = $3, last_claimed_at = $2 
		WHERE id = $4
	`, sectID, now, newShieldUntil, req.MineID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update mine occupier"})
		return
	}

	_, err = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 5, $1) WHERE id = $2", MaxHumanQi, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Nhân Khí"})
		return
	}

	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã chiếm đóng mỏ khoáng thành công! Nhân Khí +5."})
}
