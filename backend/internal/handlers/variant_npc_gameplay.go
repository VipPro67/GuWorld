package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

const maxMortalNPCRank = 5

func releaseActorGu(tx *sql.Tx, actorID string) error {
	rows, err := tx.Query(`SELECT player_gu_id FROM actor_gu_equipment WHERE actor_id=$1`, actorID)
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err = rows.Close(); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM actor_gu_equipment WHERE actor_id=$1`, actorID); err != nil {
		return err
	}
	if len(ids) > 0 {
		_, err = tx.Exec(`UPDATE player_gu SET location='INVENTORY',updated_at=NOW() WHERE id=ANY($1::text[])`, pq.Array(ids))
	}
	return err
}

func loadActorGu(q db.Queryer, actorID string) (models.GuInventory, error) {
	rows, err := q.Query(`
		SELECT pg.id, pg.gu_id, pg.gu_rank, pg.hunger, pg.state, true, false,
		       pg.integration_time, pg.equipped_at, pg.growth_hours
		FROM actor_gu_equipment e JOIN player_gu pg ON pg.id=e.player_gu_id
		WHERE e.actor_id=$1 ORDER BY e.slot`, actorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := models.GuInventory{}
	for rows.Next() {
		var gu models.GuInstance
		var hunger float64
		var equippedAt sql.NullTime
		if err := rows.Scan(&gu.InstanceID, &gu.GuID, &gu.Rank, &hunger, &gu.Status,
			&gu.Equipped, &gu.IsVital, &gu.IntegrationTime, &equippedAt, &gu.GrowthHours); err != nil {
			return nil, err
		}
		gu.Hunger = int(hunger)
		if equippedAt.Valid {
			t := equippedAt.Time
			gu.EquippedAt = &t
		}
		result = append(result, gu)
	}
	return result, rows.Err()
}

type npcFormationRequest struct {
	Slot int `json:"slot" binding:"min=0,max=8"`
}

func SetNPCFormation(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req npcFormationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Slot không hợp lệ"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(500, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()
	var playerRank int
	if err = tx.QueryRow(`SELECT rank FROM players WHERE id=$1 FOR UPDATE`, playerID).Scan(&playerRank); err != nil {
		c.JSON(500, gin.H{"error": "Database error"})
		return
	}
	maxSlots := 1 + playerRank
	if maxSlots > 8 {
		maxSlots = 8
	}
	if req.Slot > maxSlots {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Tối đa %d ô đội hình", maxSlots)})
		return
	}
	if req.Slot > 0 {
		var occupied bool
		if err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM non_player_actors WHERE owner_player_id=$1 AND lifecycle_state='ACTIVE' AND formation_slot=$2 AND id<>$3)`, playerID, req.Slot, c.Param("npc_id")).Scan(&occupied); err != nil {
			c.JSON(500, gin.H{"error": "Database error"})
			return
		}
		if occupied {
			c.JSON(http.StatusConflict, gin.H{"error": "Ô đội hình đã được sử dụng"})
			return
		}
	}
	result, err := tx.Exec(`UPDATE non_player_actors SET formation_slot=$3,updated_at=NOW(),version=version+1
		WHERE id=$1 AND owner_player_id=$2 AND actor_kind='VARIANT_NPC' AND lifecycle_state='ACTIVE'
		  AND role='NONE' AND execution_state='STORED'`, c.Param("npc_id"), playerID, req.Slot)
	if err != nil {
		c.JSON(500, gin.H{"error": "Database error"})
		return
	}
	if n, _ := result.RowsAffected(); n != 1 {
		c.JSON(http.StatusConflict, gin.H{"error": "NPC phải nhàn rỗi để vào đội hình"})
		return
	}
	if err = tx.Commit(); err != nil {
		c.JSON(500, gin.H{"error": "Database error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"formation_slot": req.Slot})
}

type npcEquipGuRequest struct {
	GuInstanceID string `json:"gu_instance_id" binding:"required"`
	Slot         int    `json:"slot" binding:"min=0,max=9"`
}

func EquipNPCGu(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req npcEquipGuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(500, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()
	var actorRank int
	err = tx.QueryRow(`SELECT rank FROM non_player_actors WHERE id=$1 AND owner_player_id=$2 AND actor_kind='VARIANT_NPC' AND lifecycle_state='ACTIVE' FOR UPDATE`, c.Param("npc_id"), playerID).Scan(&actorRank)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy NPC"})
		return
	}
	var guRank int
	err = tx.QueryRow(`SELECT gu_rank FROM player_gu WHERE id=$1 AND player_id=$2 AND location='INVENTORY' AND state='ACTIVE' AND hunger>0 AND NOT equipped AND NOT is_vital AND NOT is_locked FOR UPDATE`, req.GuInstanceID, playerID).Scan(&guRank)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Cổ không khả dụng để giao cho NPC"})
		return
	}
	if guRank > actorRank {
		c.JSON(http.StatusBadRequest, gin.H{"error": "NPC không thể dùng Cổ cao hơn cảnh giới"})
		return
	}
	if req.Slot == 0 {
		for slot := 1; slot <= actorRank; slot++ {
			var used bool
			if err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM actor_gu_equipment WHERE actor_id=$1 AND slot=$2)`, c.Param("npc_id"), slot).Scan(&used); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
				return
			}
			if !used {
				req.Slot = slot
				break
			}
		}
		if req.Slot == 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "NPC đã đầy ô trang bị"})
			return
		}
	}
	if req.Slot > actorRank {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("NPC hiện chỉ có %d ô trang bị", actorRank)})
		return
	}
	if _, err = tx.Exec(`UPDATE player_gu SET location='ACTOR',equipped=false,is_vital=false,updated_at=NOW() WHERE id=$1`, req.GuInstanceID); err != nil {
		c.JSON(500, gin.H{"error": "Database error"})
		return
	}
	if _, err = tx.Exec(`INSERT INTO actor_gu_equipment(actor_id,player_gu_id,slot) VALUES($1,$2,$3)`, c.Param("npc_id"), req.GuInstanceID, req.Slot); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Cổ hoặc ô trang bị đã được sử dụng"})
		return
	}
	if err = tx.Commit(); err != nil {
		c.JSON(500, gin.H{"error": "Database error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"equipped": true})
}

func UnequipNPCGu(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(500, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()
	var guID string
	err = tx.QueryRow(`DELETE FROM actor_gu_equipment e USING non_player_actors a
		WHERE e.actor_id=a.id AND a.id=$1 AND a.owner_player_id=$2 AND e.player_gu_id=$3 RETURNING e.player_gu_id`, c.Param("npc_id"), playerID, c.Param("gu_id")).Scan(&guID)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Trang bị không tồn tại"})
		return
	}
	if err != nil {
		c.JSON(500, gin.H{"error": "Database error"})
		return
	}
	if _, err = tx.Exec(`UPDATE player_gu SET location='INVENTORY',updated_at=NOW() WHERE id=$1`, guID); err != nil {
		c.JSON(500, gin.H{"error": "Database error"})
		return
	}
	if err = tx.Commit(); err != nil {
		c.JSON(500, gin.H{"error": "Database error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"equipped": false})
}

func CultivateNPC(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(500, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()
	var rank, subRank int
	var experience int64
	var talent float64
	err = tx.QueryRow(`SELECT rank,sub_rank,experience,talent FROM non_player_actors
		WHERE id=$1 AND owner_player_id=$2 AND actor_kind='VARIANT_NPC' AND lifecycle_state='ACTIVE'
		  AND execution_state='STORED' AND role='NONE' AND formation_slot=0 FOR UPDATE`, c.Param("npc_id"), playerID).Scan(&rank, &subRank, &experience, &talent)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "NPC phải nhàn rỗi để tu luyện"})
		return
	}
	if rank >= maxMortalNPCRank && subRank >= 4 {
		c.JSON(http.StatusConflict, gin.H{"error": "NPC đã đạt đỉnh Ngũ Chuyển"})
		return
	}
	cost := int64(rank * subRank * 100)
	gain := int64(math.Max(10, 20+talent/2))
	threshold := int64(rank * subRank * 100)
	res, err := tx.Exec(`UPDATE players SET primeval_stones=primeval_stones-$2 WHERE id=$1 AND primeval_stones >= $2`, playerID, cost)
	if err != nil {
		c.JSON(500, gin.H{"error": "Database error"})
		return
	}
	if n, _ := res.RowsAffected(); n != 1 {
		c.JSON(http.StatusConflict, gin.H{"error": "Không đủ nguyên thạch"})
		return
	}
	experience += gain
	for experience >= threshold && !(rank >= maxMortalNPCRank && subRank >= 4) {
		experience -= threshold
		if subRank < 4 {
			subRank++
		} else {
			rank++
			subRank = 1
		}
		threshold = int64(rank * subRank * 100)
	}
	maxHP := 80 + float64(rank*40) + talent
	attack := 8 + float64(rank*6) + talent*.12
	defense := 5 + float64(rank*4) + talent*.08
	speed := 8 + float64(rank) + talent*.02
	_, err = tx.Exec(`UPDATE non_player_actors SET rank=$2,sub_rank=$3,experience=$4,max_hp=$5,hp=LEAST($5,hp+($5-max_hp)),base_attack=$6,base_defense=$7,speed=$8,updated_at=NOW(),version=version+1 WHERE id=$1`, c.Param("npc_id"), rank, subRank, experience, maxHP, attack, defense, speed)
	if err != nil {
		c.JSON(500, gin.H{"error": "Database error"})
		return
	}
	if err = tx.Commit(); err != nil {
		c.JSON(500, gin.H{"error": "Database error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rank": rank, "sub_rank": subRank, "experience": experience, "next_experience": threshold, "gained": gain, "cost": cost})
}
