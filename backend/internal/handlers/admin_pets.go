package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
)

func AdminListPlayerPets(c *gin.Context) {
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
	_ = SendSystemMessage("world", sql.NullString{}, fmt.Sprintf("[Hệ thống] Admin đã thu hồi Linh thú '%s' của đạo hữu %s", nickname, username))

	c.JSON(http.StatusOK, gin.H{"deleted": true, "pet_id": petID})
}
