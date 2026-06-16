package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func InitializeCombatState(tx *sql.Tx, base *models.CombatState, player *models.Player) *models.CombatState {
	playerStats := CalculatePlayerStats(player)
	playerCombatant := models.CombatantFromPlayer(player, playerStats)
	playerCombatant.HP = player.HP
	playerCombatant.MaxHP = 100.0
	if playerCombatant.HP > playerCombatant.MaxHP {
		playerCombatant.HP = playerCombatant.MaxHP
	}
	if base.EnemyType == "PVP" {
		pHP := player.BaseHP
		if pHP <= 0 {
			pHP = 100.0
		}
		playerCombatant.HP = pHP
	}
	base.Allies = []models.Combatant{playerCombatant}

	enemyCombatant := models.CombatantFromActiveCombat(base)
	if enemyCombatant.Speed == 0 {
		enemyCombatant.Speed = 8.0 + float64(base.EnemyRank)*2.0 + float64(base.EnemySubRank)*0.5
	}
	base.Enemies = []models.Combatant{enemyCombatant}

	if player.PartyID != nil {
		var membersRaw []byte
		errPart := tx.QueryRow("SELECT members FROM parties WHERE id = $1", player.PartyID).Scan(&membersRaw)
		if errPart == nil {
			var members []PartyMember
			json.Unmarshal(membersRaw, &members)
			for _, m := range members {
				if m.ID != player.ID.String() {
					mUUID, errParse := uuid.Parse(m.ID)
					if errParse == nil {
						var mPlayer models.Player
						errMember := tx.QueryRow(`
							SELECT id, username, rank, sub_rank, hp, max_mana, current_mana, path_masteries, gu_inventory, dao_marks, body_mods, base_hp
							FROM players 
							WHERE id = $1 AND current_region = $2`,
							mUUID, player.CurrentRegion).Scan(
							&mPlayer.ID, &mPlayer.Username, &mPlayer.Rank, &mPlayer.SubRank,
							&mPlayer.HP, &mPlayer.MaxMana, &mPlayer.CurrentMana, &mPlayer.PathMasteries, &mPlayer.GuInventory, &mPlayer.DaoMarks, &mPlayer.BodyMods, &mPlayer.BaseHP,
						)
						if errMember == nil && (mPlayer.HP > 0 || base.EnemyType == "PVP") {
							mStats := CalculatePlayerStats(&mPlayer)
							mCombatant := models.CombatantFromPlayer(&mPlayer, mStats)
							mCombatant.MaxHP = 100.0
							if mCombatant.HP > mCombatant.MaxHP {
								mCombatant.HP = mCombatant.MaxHP
							}
							if base.EnemyType == "PVP" {
								mHP := mPlayer.BaseHP
								if mHP <= 0 {
									mHP = 100.0
								}
								mCombatant.HP = mHP
							}
							mCombatant.Kind = "Ally"
							base.Allies = append(base.Allies, mCombatant)
						}
					}
				}
			}
		}
	}

	if base.TargetPlayerID == "" && base.EnemyTemplateID != "monster_heavenly_tribulation" && base.EnemyHP < 1000 {
		numMinions := rand.Intn(3)
		for i := 0; i < numMinions; i++ {
			minion := models.Combatant{
				ID:            fmt.Sprintf("minion_%d_%s", i, uuid.New().String()),
				Name:          fmt.Sprintf("Hộ Vệ %s", base.EnemyName),
				Kind:          "Normal",
				Rank:          base.EnemyRank,
				SubRank:       1,
				HP:            base.EnemyMaxHP * 0.4,
				MaxHP:         base.EnemyMaxHP * 0.4,
				CurrentMana:   base.EnemyMana * 0.4,
				MaxMana:       base.EnemyMaxMana * 0.4,
				BaseAttack:    base.EnemyDamage * 0.5,
				BaseDefense:   base.EnemyDefense * 0.5,
				AIProfile:     base.AIProfile,
				StatusEffects: []models.StatusEffect{},
			}
			base.Enemies = append(base.Enemies, minion)
		}
	}

	allySlots := [][2]int{
		{2, 2},
		{2, 1},
		{2, 3},
		{3, 2},
		{3, 1},
		{3, 3},
		{1, 2},
		{1, 1},
		{1, 3},
	}
	for idx := range base.Allies {
		if idx < len(allySlots) {
			base.Allies[idx].GridX = allySlots[idx][0]
			base.Allies[idx].GridY = allySlots[idx][1]
		}
		if base.Allies[idx].Speed == 0 {
			base.Allies[idx].Speed = 10.0 + float64(base.Allies[idx].Rank)*2.0 + float64(base.Allies[idx].SubRank)*0.5
		}
	}

	enemySlots := [][2]int{
		{1, 2},
		{1, 1},
		{1, 3},
		{2, 2},
		{2, 1},
		{2, 3},
		{3, 2},
		{3, 1},
		{3, 3},
	}
	for idx := range base.Enemies {
		if idx < len(enemySlots) {
			base.Enemies[idx].GridX = enemySlots[idx][0]
			base.Enemies[idx].GridY = enemySlots[idx][1]
		}
		if base.Enemies[idx].Speed == 0 {
			base.Enemies[idx].Speed = 8.0 + float64(base.Enemies[idx].Rank)*2.0 + float64(base.Enemies[idx].SubRank)*0.5
		}
	}

	return base
}

func StartCombat(tx *sql.Tx, base *models.CombatState, player *models.Player) *models.CombatState {
	if base.EnemyType != "PVP" && base.EnemyGuInventory != nil {
		var filteredGu []models.GuInstance
		for _, g := range base.EnemyGuInventory {
			spec, ok := models.GuRegistry[g.GuID]
			if ok && spec.Rank >= 6 {
				exists, _ := db.CheckImmortalGuExists(tx, g.GuID)
				if exists {
					continue
				}
			}
			filteredGu = append(filteredGu, g)
		}
		base.EnemyGuInventory = filteredGu
	}

	initialized := InitializeCombatState(tx, base, player)

	combatBytes, err := json.Marshal(initialized)
	if err == nil {
		for _, ally := range initialized.Allies {
			allyUUID, errParse := uuid.Parse(ally.ID)
			if errParse == nil && allyUUID != player.ID {
				_, _ = tx.Exec("UPDATE players SET active_combat = $1 WHERE id = $2", combatBytes, allyUUID)
			}
		}
	}
	return initialized
}
