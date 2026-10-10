package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func GetPlayerStatus(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid player UUID"})
		return
	}

	// AUTH SECURITY CHECK: Verify token matches the requested resource
	authID, exists := middleware.GetPlayerID(c)
	if !exists || authID != playerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: You do not own this cultivation Aperture"})
		return
	}

	player, err := projectPlayerStatusTx(db.DB, playerID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to project player status: " + err.Error()})
		}
		return
	}

	// player.MaterialsInventory is already populated by loadRow() inside ProjectStatus.
	// No need to call loadMaterialsInventory() again here.
	player.BaseHP = gamefx.EffectiveMaxHP(&player)
	writeTrimmedPlayerStatus(c, player)
}

type playerStatusPayload models.Player

// playerStatusResponse shadows fields the status client never reads: auth
// identifiers (referral code comes from /player/referrals), internal tick /
// damage bookkeeping, and disguise values only meaningful to other viewers.
// The omitempty fields override their promoted Player fields during encoding,
// preserving the response shape without a marshal/unmarshal/map/marshal pass.
type playerStatusResponse struct {
	playerStatusPayload
	Email            string     `json:"email,omitempty"`
	GoogleID         *string    `json:"google_id,omitempty"`
	RefCode          string     `json:"ref_code,omitempty"`
	LastHungerTickAt *time.Time `json:"last_hunger_tick_at,omitempty"`
	DaoMarkDamage    float64    `json:"dao_mark_damage,omitempty"`
	ApertureDamage   float64    `json:"aperture_damage,omitempty"`
	BodyMods         []string   `json:"body_mods,omitempty"`
	FakeRank         int        `json:"fake_rank,omitempty"`
	FakeSubRank      int        `json:"fake_sub_rank,omitempty"`
	FakeVitalGu      string     `json:"fake_vital_gu,omitempty"`
}

// writeTrimmedPlayerStatus omits server-only fields while serializing the
// projected player once, instead of serializing, decoding to a map, and
// serializing again.
func writeTrimmedPlayerStatus(c *gin.Context, player models.Player) {
	payload, err := json.Marshal(playerStatusResponse{playerStatusPayload: playerStatusPayload(player)})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize player status"})
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", payload)
}

// GetPlayerInventoryGus returns the full Gu inventory of the authenticated player.
// Uses a lightweight query against player_gu directly instead of loading the full player row.

type playerQueryRower interface {
	QueryRow(query string, args ...any) *sql.Row
}

// loadPlayerForStatus delegates to the service layer.
func loadPlayerForStatus(q playerQueryRower, id uuid.UUID, forUpdate bool) (models.Player, error) {
	return PlayerSvc.Load(q, id, forUpdate)
}

func setDefaultPlayerStatusFields(player *models.Player) {
	// Unlocked recipes are now stored in player_recipe_progress table
	if player.Lifespan < 0 {
		player.Lifespan = 0
	}
	if player.TimeSpeedMultiplier <= 0 {
		if player.Rank >= 6 {
			player.TimeSpeedMultiplier = 10.0
		} else {
			player.TimeSpeedMultiplier = 1.0
		}
	}
}

// projectPlayerStatusTx delegates to the service layer.
func projectPlayerStatusTx(q playerQueryRower, id uuid.UUID) (models.Player, error) {
	return PlayerSvc.ProjectStatus(q, id)
}

// GrantRace sets a player's race after validating the key. Dùng cho các hoạt động
// trong game (quest, item, kỳ ngộ...) khi người chơi nhận được chủng tộc mới.
func GrantRace(tx *sql.Tx, playerID uuid.UUID, race string) error {
	if race != "" && !gamefx.ValidRaces()[race] {
		return fmt.Errorf("chủng tộc không hợp lệ: %s", race)
	}
	_, err := tx.Exec(`UPDATE players SET race = $1 WHERE id = $2`, race, playerID)
	return err
}

// EquipGuRequest defines request to equip/unequip Gu bugs
