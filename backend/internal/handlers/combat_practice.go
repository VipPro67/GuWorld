package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

type PracticeStartRequest struct {
	Loadout     simulator.CombatLoadout `json:"loadout"`
	DummyHP     float64                 `json:"dummy_hp"`
	DummyShield float64                 `json:"dummy_shield"`
	DummyAtk    float64                 `json:"dummy_atk"`
	DummyDef    float64                 `json:"dummy_def"`
	DummySpeed  float64                 `json:"dummy_speed"`
}

// PracticeCombatStart initializes and simulates a practice combat against a custom dummy.
func PracticeCombatStart(c *gin.Context) {
	var req PracticeStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTxEquipped(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	// Always true for practice mode to avoid consuming real resources
	isVirtual := true
	seed := time.Now().UnixNano()
	simState := &simulator.CombatState{
		Turn:        0,
		Seed:        seed,
		BossPhase:   0,
		VirtualMode: isVirtual,
	}

	var loadoutPtr *simulator.CombatLoadout
	if len(req.Loadout.GuIds) > 0 || req.Loadout.KillerMoveIds != nil {
		loadoutPtr = &req.Loadout
	}

	simPlayer, err := buildSimCombatant(tx, &player, isVirtual, true, loadoutPtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize initiator stats"})
		return
	}
	simPlayer.GridX = 2
	simPlayer.GridY = 2
	simState.Allies = append(simState.Allies, simPlayer)

	// Create the training dummy
	dummyHP := req.DummyHP
	if dummyHP <= 0 {
		dummyHP = 100000
	}
	dummy := &simulator.CombatantAuto{
		ID:            "practice_dummy",
		Name:          "Hình Nhân Gỗ",
		Kind:          "Enemy",
		Rank:          1,
		SubRank:       1,
		GridX:         1,
		GridY:         2,
		MaxHP:         dummyHP,
		HP:            dummyHP,
		MaxMana:       0,
		Mana:          0,
		BaseDmg:       req.DummyAtk,
		BaseDef:       req.DummyDef,
		Speed:         req.DummySpeed,
		VirtualShield: req.DummyShield,
		Actions: []simulator.SimAction{{
			ActionType: "BASIC_ATTACK",
			Name:       "Đánh thường",
			Damage:     req.DummyAtk,
			ManaCost:   0,
		}},
	}
	simState.Enemies = append(simState.Enemies, dummy)

	// Simulate the combat
	logs := simulator.Simulate(simState)

	// Determine result (whether player won)
	playerWon := false
	for _, e := range simState.Enemies {
		if e.HP <= 0 || e.IsDead {
			playerWon = true
			break
		}
	}

	// Add END event
	endEvent := simulator.CombatLog{
		Turn:   simState.Turn + 1,
		Action: "BATTLE_END",
		Detail: "Trận chiến kết thúc.",
	}
	if playerWon {
		endEvent.Detail = "Thắng lợi! Hình nhân đã bị phá hủy."
	} else {
		endEvent.Detail = "Thất bại! Bạn không thể đánh bại hình nhân."
	}
	logs = append(logs, endEvent)

	// Commit transaction just in case (though no real data changed, but evaluateManaTx might have ticked something)
	tx.Commit()

	// Return logs directly to client for playback
	c.JSON(http.StatusOK, gin.H{
		"logs":        logs,
		"player_won":  playerWon,
		"is_practice": true,
	})
}
