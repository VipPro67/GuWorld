package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

func AdminListPlayerPets(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	playerID := c.Param("id")
	if playerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing player ID"})
		return
	}

	pets, err := db.DB.Query(`
		SELECT id, player_id, monster_template_id, nickname, rank, sub_rank,
		       hp, max_hp, base_attack, base_defense, speed, count,
		       hunger, max_hunger, pet_hunger_per_hour, state, formation_slot,
		       captured_at, updated_at
		FROM player_pets WHERE player_id = $1
		ORDER BY formation_slot ASC, captured_at ASC
	`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query pets"})
		return
	}
	defer pets.Close()

	var results []map[string]interface{}
	for pets.Next() {
		var id, plID, templateID, nickname, state string
		var rank, subRank, count, formationSlot int
		var hp, maxHP, atk, def, speed, hunger, maxHunger, hungerRate float64
		var capturedAt, updatedAt interface{}
		if errScan := pets.Scan(
			&id, &plID, &templateID, &nickname, &rank, &subRank,
			&hp, &maxHP, &atk, &def, &speed, &count,
			&hunger, &maxHunger, &hungerRate, &state, &formationSlot,
			&capturedAt, &updatedAt,
		); errScan != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"id": id, "player_id": plID, "monster_template_id": templateID,
			"nickname": nickname, "rank": rank, "sub_rank": subRank,
			"hp": hp, "max_hp": maxHP, "base_attack": atk, "base_defense": def,
			"speed": speed, "count": count, "hunger": hunger, "max_hunger": maxHunger,
			"pet_hunger_per_hour": hungerRate, "state": state, "formation_slot": formationSlot,
			"captured_at": capturedAt, "updated_at": updatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"pets": results})
}

func AdminDeletePlayerPet(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	playerID := c.Param("id")
	petID := c.Param("pet_id")
	if playerID == "" || petID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing player ID or pet ID"})
		return
	}

	var nickname string
	_ = db.DB.QueryRow(`SELECT nickname FROM player_pets WHERE id = $1`, petID).Scan(&nickname)
	if nickname == "" {
		nickname = petID
	}

	result, err := db.DB.Exec(`DELETE FROM player_pets WHERE id = $1 AND player_id = $2`, petID, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete pet"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pet not found"})
		return
	}

	var username string
	_ = db.DB.QueryRow(`SELECT username FROM players WHERE id = $1`, playerID).Scan(&username)
	_ = SendSystemMessage("world", sql.NullString{}, fmt.Sprintf("Admin đã thu hồi Linh thú '%s' của đạo hữu %s", nickname, username))

	playerUUID, _ := uuid.Parse(playerID)
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.delete_pet",
		EntityType: "player_pet", EntityID: petID,
		RequestSummary: fmt.Sprintf("player=%s pet=%s nickname=%s", playerID, petID, nickname),
		IPAddress:      c.ClientIP(), Success: true,
	})
	_ = playerUUID

	c.JSON(http.StatusOK, gin.H{"deleted": true, "pet_id": petID})
}

type AdminPetInput struct {
	MonsterTemplateID string  `json:"monster_template_id"`
	Nickname          string  `json:"nickname"`
	Rank              int     `json:"rank"`
	SubRank           int     `json:"sub_rank"`
	HP                float64 `json:"hp"`
	MaxHP             float64 `json:"max_hp"`
	BaseAttack        float64 `json:"base_attack"`
	BaseDefense       float64 `json:"base_defense"`
	Speed             float64 `json:"speed"`
	Count             int     `json:"count"`
	Hunger            float64 `json:"hunger"`
	MaxHunger         float64 `json:"max_hunger"`
	PetHungerPerHour  float64 `json:"pet_hunger_per_hour"`
	State             string  `json:"state"`
	FormationSlot     int     `json:"formation_slot"`
}

func AdminCreatePlayerPet(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	playerID := c.Param("id")
	if playerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing player ID"})
		return
	}

	var req AdminPetInput
	if err := c.ShouldBindJSON(&req); err != nil || req.MonsterTemplateID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID mẫu quái vật là bắt buộc"})
		return
	}

	if req.Nickname == "" {
		req.Nickname = req.MonsterTemplateID
	}
	if req.Rank < 1 {
		req.Rank = 1
	}
	if req.MaxHP <= 0 {
		req.MaxHP = 100
	}
	if req.HP <= 0 {
		req.HP = req.MaxHP
	}
	if req.Count <= 0 {
		req.Count = 1
	}
	if req.State == "" {
		req.State = "active"
	}

	_, err := db.DB.ExecContext(c.Request.Context(), `
		INSERT INTO player_pets (player_id, monster_template_id, nickname, rank, sub_rank, hp, max_hp, base_attack, base_defense, speed, count, hunger, max_hunger, pet_hunger_per_hour, state, formation_slot)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		playerID, req.MonsterTemplateID, req.Nickname, req.Rank, req.SubRank, req.HP, req.MaxHP, req.BaseAttack, req.BaseDefense, req.Speed, req.Count, req.Hunger, req.MaxHunger, req.PetHungerPerHour, req.State, req.FormationSlot)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo linh thú: " + err.Error()})
		return
	}

	playerUUID, _ := uuid.Parse(playerID)
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.create_pet",
		EntityType: "player_pet", EntityID: playerID,
		RequestSummary: fmt.Sprintf("player=%s template=%s nickname=%s", playerID, req.MonsterTemplateID, req.Nickname),
		IPAddress:      c.ClientIP(), Success: true,
	})
	_ = playerUUID

	c.JSON(http.StatusOK, gin.H{"message": "Đã cấp linh thú thành công"})
}

func AdminUpdatePlayerPet(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	playerID := c.Param("id")
	petID := c.Param("pet_id")
	if playerID == "" || petID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing player ID or pet ID"})
		return
	}

	var req AdminPetInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.DB.ExecContext(c.Request.Context(), `
		UPDATE player_pets SET nickname=$1, rank=$2, sub_rank=$3, hp=$4, max_hp=$5, base_attack=$6, base_defense=$7, speed=$8, count=$9, hunger=$10, max_hunger=$11, pet_hunger_per_hour=$12, state=$13, formation_slot=$14, updated_at=NOW()
		WHERE id=$15 AND player_id=$16`,
		req.Nickname, req.Rank, req.SubRank, req.HP, req.MaxHP, req.BaseAttack, req.BaseDefense, req.Speed, req.Count, req.Hunger, req.MaxHunger, req.PetHungerPerHour, req.State, req.FormationSlot, petID, playerID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật linh thú: " + err.Error()})
		return
	}

	playerUUID, _ := uuid.Parse(playerID)
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.update_pet",
		EntityType: "player_pet", EntityID: petID,
		RequestSummary: fmt.Sprintf("player=%s pet=%s nickname=%s", playerID, petID, req.Nickname),
		IPAddress:      c.ClientIP(), Success: true,
	})
	_ = playerUUID

	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật linh thú thành công"})
}
