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
	if base.EnemyType == "PVP" && base.EnemyGuInventory != nil {
		var filteredGu []models.GuInstance
		for _, g := range base.EnemyGuInventory {
			if g.Equipped && g.Status == "ACTIVE" {
				spec, ok := models.GetGu(g.GuID, g.Rank)
				if ok && hasCombatActiveEffects(&spec) {
					filteredGu = append(filteredGu, g)
				}
			}
		}
		base.EnemyGuInventory = filteredGu
	}

	playerStats := CalculatePlayerStats(player)
	playerCombatant := models.CombatantFromPlayer(player, playerStats)
	playerCombatant.HP = player.HP
	if playerCombatant.HP > playerCombatant.MaxHP {
		playerCombatant.HP = playerCombatant.MaxHP
	}
	if base.CombatMode == "virtual" {
		pHP := player.BaseHP
		if pHP <= 0 {
			pHP = 100.0
		}
		playerCombatant.HP = pHP
		playerCombatant.MaxHP = pHP
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
							SELECT p.id, p.username, p.rank, p.sub_rank, p.hp, p.max_mana, p.current_mana, p.path_masteries, `+db.GuInventoryAggregate("p.id")+`, COALESCE(ia.dao_marks, '{}'::jsonb), p.body_mods, p.base_hp, p.avatar_url
							FROM players p 
							LEFT JOIN immortal_apertures ia ON p.id = ia.player_id
							WHERE p.id = $1 AND p.current_region = $2`,
							mUUID, player.CurrentRegion).Scan(
							&mPlayer.ID, &mPlayer.Username, &mPlayer.Rank, &mPlayer.SubRank,
							&mPlayer.HP, &mPlayer.MaxMana, &mPlayer.CurrentMana, &mPlayer.PathMasteries, &mPlayer.GuInventory, &mPlayer.DaoMarks, &mPlayer.BodyMods, &mPlayer.BaseHP, &mPlayer.AvatarURL,
						)
						if errMember == nil && (mPlayer.HP > 0 || base.CombatMode == "virtual") {
							mStats := CalculatePlayerStats(&mPlayer)
							mCombatant := models.CombatantFromPlayer(&mPlayer, mStats)
							mCombatant.MaxHP = 100.0
							if mCombatant.HP > mCombatant.MaxHP {
								mCombatant.HP = mCombatant.MaxHP
							}
							if base.CombatMode == "virtual" {
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

	// Load formation pets into Allies
	petRows, errPet := tx.Query(`
		SELECT id, nickname, rank, sub_rank, hp, max_hp, base_attack, base_defense, speed, count
		FROM player_pets
		WHERE player_id = $1 AND formation_slot > 0 AND state = 'ACTIVE' AND hunger > 0
		ORDER BY formation_slot ASC
	`, player.ID)
	if errPet == nil {
		defer petRows.Close()
		for petRows.Next() {
			var petID, nickname string
			var petRank, petSubRank, petCount int
			var petHP, petMaxHP, petAtk, petDef, petSpeed float64
			if errScan := petRows.Scan(&petID, &nickname, &petRank, &petSubRank, &petHP, &petMaxHP, &petAtk, &petDef, &petSpeed, &petCount); errScan != nil {
				continue
			}
			displayName := nickname
			if petCount > 1 {
				displayName = fmt.Sprintf("Bầy %s (%d)", nickname, petCount)
			}
			petCombatant := models.Combatant{
				ID:            petID,
				Name:          displayName,
				Kind:          "Pet",
				Rank:          petRank,
				SubRank:       petSubRank,
				HP:            petHP,
				MaxHP:         petMaxHP,
				BaseAttack:    petAtk,
				BaseDefense:   petDef,
				Speed:         petSpeed,
				StatusEffects: []models.StatusEffect{},
			}
			base.Allies = append(base.Allies, petCombatant)
			base.PetIDs = append(base.PetIDs, petID)
		}
	}

	if base.TargetPlayerID == "" && base.MineSlotID == 0 && base.EnemyTemplateID != "monster_heavenly_tribulation" && base.EnemyHP < 1000 {
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
		{2, 3}, // index 0 (player) -> visual center-back (middle row, leftmost column)
		{2, 1}, // index 1 (pet slot 1) -> visual center-front (middle row, rightmost column)
		{1, 1}, // index 2 (pet slot 2) -> visual top-front (top row, rightmost column)
		{3, 1}, // index 3 (pet slot 3) -> visual bottom-front (bottom row, rightmost column)
		{2, 2}, // index 4 (pet slot 4) -> visual center-mid (middle row, middle column)
		{1, 2}, // index 5 (pet slot 5) -> visual top-mid (top row, middle column)
		{3, 2}, // index 6 (pet slot 6) -> visual bottom-mid (bottom row, middle column)
		{1, 3}, // index 7 (pet slot 7) -> visual top-back (top row, leftmost column)
		{3, 3}, // index 8 (pet slot 8) -> visual bottom-back (bottom row, leftmost column)
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
		{2, 3}, // index 0 (boss/main enemy) -> visual center-back (middle row, rightmost column)
		{2, 1}, // index 1 -> visual center-front (middle row, leftmost column)
		{1, 1}, // index 2 -> visual top-front (top row, leftmost column)
		{3, 1}, // index 3 -> visual bottom-front (bottom row, leftmost column)
		{2, 2}, // index 4 -> visual center-mid (middle row, middle column)
		{1, 2}, // index 5 -> visual top-mid (top row, middle column)
		{3, 2}, // index 6 -> visual bottom-mid (bottom row, middle column)
		{1, 3}, // index 7 -> visual top-back (top row, rightmost column)
		{3, 3}, // index 8 -> visual bottom-back (bottom row, rightmost column)
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
	if base.CombatMode != "virtual" && base.EnemyGuInventory != nil {
		var filteredGu []models.GuInstance
		for _, g := range base.EnemyGuInventory {
			spec, ok := models.GetGu(g.GuID, g.Rank)
			if ok && spec.Rank >= 6 {
				exists, _ := db.CheckImmortalGuExists(tx, g.GuID, spec.Rank, nil)
				if exists {
					continue
				}
			}
			filteredGu = append(filteredGu, g)
		}
		base.EnemyGuInventory = filteredGu
	}

	initialized := InitializeCombatState(tx, base, player)

	// Fire ON_COMBAT_START on equipped ACTIVE Gus
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok {
				HandleGuEvent(player, spec, gu, "ON_COMBAT_START")
			}
		}
	}

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
