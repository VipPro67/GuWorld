package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
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
		SELECT id, owner_player_id, template_id, name, rank, sub_rank,
		       hp, max_hp, base_attack, base_defense, speed, count,
		       hunger, max_hunger, upkeep_rate,
		       CASE WHEN execution_state = 'RECOVERING' THEN 'SLEEPING' ELSE 'ACTIVE' END,
		       formation_slot, created_at, updated_at
		FROM non_player_actors WHERE owner_player_id = $1
		  AND actor_kind = 'COMPANION_BEAST' AND lifecycle_state = 'ACTIVE'
		ORDER BY formation_slot ASC, created_at ASC
	`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query pets"})
		return
	}
	defer pets.Close()

	var results []map[string]interface{}
	for pets.Next() {
		var id, plID, templateID, name, state string
		var rank, subRank, count, formationSlot int
		var hp, maxHP, atk, def, speed, hunger, maxHunger, hungerRate float64
		var capturedAt, updatedAt interface{}
		if errScan := pets.Scan(
			&id, &plID, &templateID, &name, &rank, &subRank,
			&hp, &maxHP, &atk, &def, &speed, &count,
			&hunger, &maxHunger, &hungerRate, &state, &formationSlot,
			&capturedAt, &updatedAt,
		); errScan != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"id": id, "player_id": plID, "monster_template_id": templateID,
			"nickname": name, "rank": rank, "sub_rank": subRank,
			"hp": hp, "max_hp": maxHP, "base_attack": atk, "base_defense": def,
			"speed": speed, "count": count, "hunger": hunger, "max_hunger": maxHunger,
			"pet_hunger_per_hour": hungerRate, "state": state, "formation_slot": formationSlot,
			"captured_at": capturedAt, "updated_at": updatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"pets": results})
}

func AdminDeletePlayerPet(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	playerID := c.Param("id")
	petID := c.Param("pet_id")
	if playerID == "" || petID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing player ID or pet ID"})
		return
	}

	var name string
	_ = db.DB.QueryRow(`SELECT name FROM non_player_actors WHERE id = $1`, petID).Scan(&name)
	if name == "" {
		name = petID
	}

	result, err := db.DB.Exec(`DELETE FROM non_player_actors WHERE id = $1 AND owner_player_id = $2 AND actor_kind = 'COMPANION_BEAST'`, petID, playerID)
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
	_ = SendSystemMessage("world", sql.NullString{}, fmt.Sprintf("Admin đã thu hồi Linh thú '%s' của đạo hữu %s", name, username))

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
	if _, ok := requireAdmin(c); !ok {
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
		INSERT INTO non_player_actors (actor_kind, owner_player_id, template_source, template_id, species, name, rank, sub_rank, hp, max_hp, base_attack, base_defense, speed, count, hunger, max_hunger, upkeep_rate, lifecycle_state, execution_state, behavior_profile, capabilities, formation_slot)
		VALUES ('COMPANION_BEAST', $1, 'MONSTER_TEMPLATE', $2, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, 'ACTIVE', CASE WHEN UPPER($15) = 'SLEEPING' THEN 'RECOVERING' ELSE 'STORED' END, 'DEFENSIVE_FOLLOWER', '{"MOVE":true,"FOLLOW":true,"GUARD":true,"COMBAT":true,"FEEDABLE":true,"CAPTURABLE":true,"STACKABLE":true,"MORTAL":true}'::jsonb, $16)`,
		playerID, req.MonsterTemplateID, req.Nickname, req.Rank, req.SubRank, req.HP, req.MaxHP, req.BaseAttack, req.BaseDefense, req.Speed, req.Count, req.Hunger, req.MaxHunger, req.PetHungerPerHour, req.State, req.FormationSlot)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo linh thú: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã cấp linh thú thành công"})
}

func AdminUpdatePlayerPet(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
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
		UPDATE non_player_actors SET name=$1, rank=$2, sub_rank=$3, hp=$4, max_hp=$5, base_attack=$6, base_defense=$7, speed=$8, count=$9, hunger=$10, max_hunger=$11, upkeep_rate=$12, execution_state=CASE WHEN UPPER($13) = 'SLEEPING' THEN 'RECOVERING' ELSE 'STORED' END, formation_slot=$14, updated_at=NOW(), version=version+1
		WHERE id=$15 AND owner_player_id=$16 AND actor_kind='COMPANION_BEAST' AND lifecycle_state='ACTIVE'`,
		req.Nickname, req.Rank, req.SubRank, req.HP, req.MaxHP, req.BaseAttack, req.BaseDefense, req.Speed, req.Count, req.Hunger, req.MaxHunger, req.PetHungerPerHour, req.State, req.FormationSlot, petID, playerID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật linh thú: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật linh thú thành công"})
}
