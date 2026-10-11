package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

func InitializeCombatState(tx *sql.Tx, base *models.CombatState, player *models.Player) *models.CombatState {
	if base.CombatID == "" {
		base.CombatID = uuid.NewString()
		base.TitleEffectSnapshot = snapshotEquippedTitleEffect(tx, player.ID.String(), base.EnemyType)
	}
	if base.EnemyType == "PVP" && base.EnemyGuInventory != nil {
		var filteredGu []models.GuInstance
		for _, g := range base.EnemyGuInventory {
			if g.Equipped && g.Status == "ACTIVE" {
				filteredGu = append(filteredGu, g)
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
		pHP := gamefx.EffectiveMaxHP(player)
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

	// Load formation pets into Allies (soul cost applies on real combat, once per combat)
	chargeAttacker := base.CombatMode != "virtual" && !base.PetSoulCharged

	// Party battle formation (arranged by the leader): when set and non-empty,
	// followers join strictly in lineup order — member players plus the
	// leader's own pets/NPCs interleaved. Otherwise legacy auto behavior
	// (all same-region members, then pets and NPCs by formation_slot).
	formation := []PartyFormationEntry{}
	if player.PartyID != nil {
		if f, err := loadPartyFormation(tx, player.PartyID.String()); err != nil {
			logger.S.Errorf("load party formation: %v", err)
		} else {
			formation = f
		}
	}
	// Explicit grids for the lineup path (nil = legacy index-based grids).
	var lineupGrids [][2]int
	if len(formation) > 0 {
		attackerSoul, attackerSoulLoaded := -1, false
		if chargeAttacker {
			var soulHP int
			if errSoul := tx.QueryRow(`SELECT soul_hp FROM players WHERE id = $1`, player.ID).Scan(&soulHP); errSoul == nil {
				attackerSoul, attackerSoulLoaded = soulHP, true
			}
		}
		// Explicit grids: leader anchors cell 0, followers take their pos
		// (or lowest free cell). Applied over the index-based defaults below.
		lineupGrids = [][2]int{allySlots[0]}
		usedPos := [9]bool{true}
		if len(formation) > MaxPartyFollowers {
			formation = formation[:MaxPartyFollowers]
		}
		for _, e := range formation {
			switch e.Kind {
			case "player":
				if ally, ok := loadPartyMemberCombatant(tx, e.ID, player, base.CombatMode == "virtual"); ok {
					base.Allies = append(base.Allies, *ally)
					lineupGrids = append(lineupGrids, allySlots[takeLineupPos(&usedPos, e.Pos)])
				}
			case "pet":
				if pet, ok := loadSingleFormationPet(tx, e.ID, player.ID.String(), base.CombatMode == "virtual", chargeAttacker, &attackerSoul); ok {
					base.Allies = append(base.Allies, *pet)
					lineupGrids = append(lineupGrids, allySlots[takeLineupPos(&usedPos, e.Pos)])
				}
			case "npc":
				if npc, err := loadSingleFormationNPC(tx, e.ID, player.ID.String(), base.CombatMode == "virtual"); err != nil {
					logger.S.Errorf("load lineup NPC %s: %v", e.ID, err)
				} else if npc != nil {
					base.Allies = append(base.Allies, *npc)
					lineupGrids = append(lineupGrids, allySlots[takeLineupPos(&usedPos, e.Pos)])
				}
			}
		}
		if chargeAttacker && attackerSoulLoaded {
			player.SoulHP = attackerSoul
			_, _ = tx.Exec("UPDATE players SET soul_hp = $1 WHERE id = $2", attackerSoul, player.ID)
			base.PetSoulCharged = true
		}
	} else {
		if player.PartyID != nil {
			var membersRaw []byte
			errPart := tx.QueryRow("SELECT members FROM parties WHERE id = $1", player.PartyID).Scan(&membersRaw)
			if errPart == nil {
				var members []PartyMember
				json.Unmarshal(membersRaw, &members)
				for _, m := range members {
					if m.ID == player.ID.String() {
						continue
					}
					if ally, ok := loadPartyMemberCombatant(tx, m.ID, player, base.CombatMode == "virtual"); ok {
						base.Allies = append(base.Allies, *ally)
					}
				}
			}
		}

		attackerPets, attackerSoul, attackerSoulLoaded := loadFormationPetCombatants(tx, player.ID.String(), base.CombatMode == "virtual", chargeAttacker)
		base.Allies = append(base.Allies, attackerPets...)
		attackerNPCs, errNPC := loadFormationNPCCombatants(tx, player.ID.String(), base.CombatMode == "virtual")
		if errNPC != nil {
			logger.S.Errorf("load attacker variant NPCs: %v", errNPC)
		} else {
			base.Allies = append(base.Allies, attackerNPCs...)
		}
		if chargeAttacker && attackerSoulLoaded {
			player.SoulHP = attackerSoul
			_, _ = tx.Exec("UPDATE players SET soul_hp = $1 WHERE id = $2", attackerSoul, player.ID)
			base.PetSoulCharged = true
		}
	}
	for _, pet := range base.Allies {
		if pet.Kind == "Pet" {
			base.PetIDs = append(base.PetIDs, pet.ID)
		}
	}

	// Load challenged/defender formation pets into Enemies (PVP: mine, arena, map)
	if base.TargetPlayerID != "" && base.TargetPlayerID != "dummy" {
		defenderPets, defenderSoul, defenderSoulLoaded := loadFormationPetCombatants(tx, base.TargetPlayerID, base.CombatMode == "virtual", base.CombatMode != "virtual")
		base.Enemies = append(base.Enemies, defenderPets...)
		defenderNPCs, errNPC := loadFormationNPCCombatants(tx, base.TargetPlayerID, base.CombatMode == "virtual")
		if errNPC != nil {
			logger.S.Errorf("load defender variant NPCs: %v", errNPC)
		} else {
			base.Enemies = append(base.Enemies, defenderNPCs...)
		}
		if defenderSoulLoaded {
			_, _ = tx.Exec("UPDATE players SET soul_hp = $1 WHERE id = $2", defenderSoul, base.TargetPlayerID)
		}
	}

	if base.TargetPlayerID == "" && base.MineSlotID == 0 && base.TribulationEventID == "" && base.EnemyTemplateID != "monster_heavenly_tribulation" && base.EnemyHP < 1000 {
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

	if base.EnemyType == "MINE_PVE" || (base.MineSlotID > 0 && base.EnemyType != "PVP") {
		for i := 0; i < 2; i++ {
			minion := models.Combatant{
				ID:            fmt.Sprintf("minion_%d_%s", i, uuid.New().String()),
				Name:          fmt.Sprintf("Hộ Vệ %s", base.EnemyName),
				Kind:          "Normal",
				Rank:          base.EnemyRank,
				SubRank:       1,
				HP:            base.EnemyMaxHP * 0.5,
				MaxHP:         base.EnemyMaxHP * 0.5,
				CurrentMana:   base.EnemyMana * 0.5,
				MaxMana:       base.EnemyMaxMana * 0.5,
				BaseAttack:    base.EnemyDamage * 0.6,
				BaseDefense:   base.EnemyDefense * 0.6,
				AIProfile:     base.AIProfile,
				StatusEffects: []models.StatusEffect{},
			}
			base.Enemies = append(base.Enemies, minion)
		}
	}

	for idx := range base.Allies {
		if idx < len(allySlots) {
			base.Allies[idx].GridX = allySlots[idx][0]
			base.Allies[idx].GridY = allySlots[idx][1]
		}
		if base.Allies[idx].Speed == 0 {
			base.Allies[idx].Speed = 8.0 + float64(base.Allies[idx].Rank)*2.0 + float64(base.Allies[idx].SubRank)*0.5
		}
	}
	// Lineup path overrides with the leader-arranged cells.
	if len(lineupGrids) == len(base.Allies) {
		for i, g := range lineupGrids {
			base.Allies[i].GridX, base.Allies[i].GridY = g[0], g[1]
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

// takeLineupPos resolves a lineup entry's grid cell: explicit pos when free,
// otherwise the lowest free cell (1..8). 0 means the lineup overflowed.
func takeLineupPos(used *[9]bool, want int) int {
	if want >= 1 && want <= MaxPartyFollowers && !used[want] {
		used[want] = true
		return want
	}
	for p := 1; p <= MaxPartyFollowers; p++ {
		if !used[p] {
			used[p] = true
			return p
		}
	}
	return 0
}

// loadPartyMemberCombatant loads one party member as an Ally combatant with
// the same eligibility as the legacy auto-join: same region and alive
// (virtual mode revives). Shared by legacy and lineup paths.
func loadPartyMemberCombatant(tx *sql.Tx, memberID string, player *models.Player, virtual bool) (*models.Combatant, bool) {
	mUUID, errParse := uuid.Parse(memberID)
	if errParse != nil || mUUID == player.ID {
		return nil, false
	}
	mPlayer, errMember := evaluateManaTxEquippedReadonly(tx, mUUID)
	if errMember != nil || mPlayer.CurrentRegion != player.CurrentRegion || (mPlayer.HP <= 0 && !virtual) {
		return nil, false
	}
	mStats := CalculatePlayerStats(&mPlayer)
	mCombatant := models.CombatantFromPlayer(&mPlayer, mStats)
	if mCombatant.HP > mCombatant.MaxHP {
		mCombatant.HP = mCombatant.MaxHP
	}
	if virtual {
		mHP := gamefx.EffectiveMaxHP(&mPlayer)
		if mHP <= 0 {
			mHP = 100.0
		}
		mCombatant.HP = mHP
		mCombatant.MaxHP = mHP
	}
	mCombatant.Kind = "Ally"
	return &mCombatant, true
}

// loadSingleFormationPet loads one leader-owned pet for the party lineup.
// Unlike the formation_slot batch loader, the lineup itself is the
// arrangement, so no slot requirement — only STORED, hunger and soul.
func loadSingleFormationPet(tx *sql.Tx, petID, ownerID string, virtual, charge bool, remainingSoul *int) (*models.Combatant, bool) {
	var name string
	var petRank, petSubRank, petCount int
	var petHP, petMaxHP, petAtk, petDef, petSpeed float64
	if err := tx.QueryRow(`
		SELECT name, rank, sub_rank, hp, max_hp, base_attack, base_defense, speed, count
		FROM non_player_actors
		WHERE id = $1 AND owner_player_id = $2 AND actor_kind = 'COMPANION_BEAST'
		  AND lifecycle_state = 'ACTIVE' AND execution_state = 'STORED' AND hunger > 0
	`, petID, ownerID).Scan(&name, &petRank, &petSubRank, &petHP, &petMaxHP, &petAtk, &petDef, &petSpeed, &petCount); err != nil {
		return nil, false
	}
	if charge {
		if *remainingSoul < petCount {
			logger.S.Warnf("lineup pet %s skipped: insufficient soul (need=%d remaining=%d)", petID, petCount, *remainingSoul)
			return nil, false
		}
		*remainingSoul -= petCount
	}
	if virtual {
		petHP = petMaxHP
	}
	displayName := name
	if petCount > 1 {
		displayName = fmt.Sprintf("Bầy %s (%d)", name, petCount)
	}
	return &models.Combatant{
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
	}, true
}

// loadSingleFormationNPC loads one leader-owned variant NPC for the party
// lineup: ACTIVE, STORED, role NONE. Returns (nil, nil) when ineligible.
func loadSingleFormationNPC(tx *sql.Tx, npcID, ownerID string, virtual bool) (*models.Combatant, error) {
	var c models.Combatant
	var path string
	if err := tx.QueryRow(`
		SELECT id,name,rank,sub_rank,hp,max_hp,base_attack,base_defense,speed,path
		FROM non_player_actors WHERE id = $1 AND owner_player_id = $2 AND actor_kind = 'VARIANT_NPC'
		AND lifecycle_state = 'ACTIVE' AND execution_state = 'STORED' AND role = 'NONE'
	`, npcID, ownerID).Scan(&c.ID, &c.Name, &c.Rank, &c.SubRank, &c.HP, &c.MaxHP, &c.BaseAttack, &c.BaseDefense, &c.Speed, &path); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	c.Kind = "VariantNPC"
	c.MaxMana = float64(c.Rank * 100)
	c.CurrentMana = c.MaxMana
	if virtual {
		c.HP = c.MaxHP
	}
	c.PathMasteries = map[string]int{path: c.Rank * 100}
	c.DaoMarks = models.DaoMarksMap{}
	gu, err := loadActorGu(tx, c.ID)
	if err != nil {
		return nil, err
	}
	c.GuInventory = gu
	return &c, nil
}

func loadFormationNPCCombatants(tx *sql.Tx, playerID string, virtual bool) ([]models.Combatant, error) {
	rows, err := tx.Query(`SELECT id,name,rank,sub_rank,hp,max_hp,base_attack,base_defense,speed,formation_slot,path
		FROM non_player_actors WHERE owner_player_id=$1 AND actor_kind='VARIANT_NPC'
		AND lifecycle_state='ACTIVE' AND execution_state='STORED' AND role='NONE' AND formation_slot>0
		ORDER BY formation_slot FOR UPDATE`, playerID)
	if err != nil {
		return nil, err
	}
	// Buffer all rows BEFORE issuing follow-up queries: a Tx pins a single
	// connection, so querying actor_gu_equipment while these rows are still
	// open corrupts the protocol (pq: unexpected Parse response) and poisons
	// the whole transaction (every later Exec fails with bad connection).
	type npcRow struct {
		c    models.Combatant
		slot int
		path string
	}
	var buffered []npcRow
	for rows.Next() {
		var n npcRow
		if err := rows.Scan(&n.c.ID, &n.c.Name, &n.c.Rank, &n.c.SubRank, &n.c.HP, &n.c.MaxHP, &n.c.BaseAttack, &n.c.BaseDefense, &n.c.Speed, &n.slot, &n.path); err != nil {
			rows.Close()
			return nil, err
		}
		buffered = append(buffered, n)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	result := []models.Combatant{}
	for _, n := range buffered {
		c := n.c
		c.Kind = "VariantNPC"
		c.MaxMana = float64(c.Rank * 100)
		c.CurrentMana = c.MaxMana
		if virtual {
			c.HP = c.MaxHP
		}
		c.PathMasteries = map[string]int{n.path: c.Rank * 100}
		c.DaoMarks = models.DaoMarksMap{}
		c.GuInventory, err = loadActorGu(tx, c.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}

// loadFormationPetCombatants loads a player's formation pets as Combatants.
// virtual restores pets to full HP, mirroring arena's virtual-mode convention.
// When charge is true, each loaded pet costs soul_hp equal to its count
// (individuals); pets that cannot be afforded are skipped. It returns the
// loaded pets, the remaining soul_hp after deduction, and whether the soul
// value was successfully read (and thus should be persisted by the caller).
func loadFormationPetCombatants(tx *sql.Tx, playerID string, virtual bool, charge bool) ([]models.Combatant, int, bool) {
	remainingSoul := -1
	soulLoaded := false
	if charge {
		var soulHP int
		if errSoul := tx.QueryRow(`SELECT soul_hp FROM players WHERE id = $1`, playerID).Scan(&soulHP); errSoul == nil {
			remainingSoul = soulHP
			soulLoaded = true
		}
	}

	petRows, errPet := tx.Query(`
		SELECT id, name, rank, sub_rank, hp, max_hp, base_attack, base_defense, speed, count
		FROM non_player_actors
		WHERE owner_player_id = $1 AND actor_kind = 'COMPANION_BEAST' AND lifecycle_state = 'ACTIVE' AND formation_slot > 0 AND execution_state = 'STORED' AND hunger > 0
		ORDER BY formation_slot ASC
	`, playerID)
	if errPet != nil {
		return nil, remainingSoul, soulLoaded
	}
	defer petRows.Close()

	var pets []models.Combatant
	skippedSoul := 0
	for petRows.Next() {
		var petID, name string
		var petRank, petSubRank, petCount int
		var petHP, petMaxHP, petAtk, petDef, petSpeed float64
		if errScan := petRows.Scan(&petID, &name, &petRank, &petSubRank, &petHP, &petMaxHP, &petAtk, &petDef, &petSpeed, &petCount); errScan != nil {
			continue
		}
		if charge {
			if remainingSoul < petCount {
				// Insufficient soul to field this pack — skip loading it.
				skippedSoul++
				continue
			}
			remainingSoul -= petCount
		}
		if virtual {
			petHP = petMaxHP
		}
		displayName := name
		if petCount > 1 {
			displayName = fmt.Sprintf("Bầy %s (%d)", name, petCount)
		}
		pets = append(pets, models.Combatant{
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
		})
	}
	if skippedSoul > 0 {
		logger.S.Warnf("loadFormationPets player %s: skipped %d packs due to insufficient soul (remaining=%d)", playerID, skippedSoul, remainingSoul)
	}
	return pets, remainingSoul, soulLoaded
}

func StartCombat(tx *sql.Tx, base *models.CombatState, player *models.Player) *models.CombatState {
	if base.CombatMode != "virtual" && base.EnemyGuInventory != nil && base.EnemyType != "PVP" {
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

	// Execute pre-combat steal mechanics (Vô Tướng Thủ & Ám Thâu Cổ) for open-world PvP
	handlePreCombatSteal(tx, initialized, player)

	// Fire ON_COMBAT_START on equipped ACTIVE Gus
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok {
				HandleGuEvent(player, spec, gu, "ON_COMBAT_START")
			}
		}
	}

	for _, ally := range initialized.Allies {
		if ally.Kind != "Ally" && ally.Kind != "Player" {
			continue
		}
		allyUUID, errParse := uuid.Parse(ally.ID)
		if errParse == nil && allyUUID != player.ID {
			// notify.Send uses a savepoint, so a member deleted after the
			// combat snapshot cannot abort the caller's transaction.
			notify.Send(tx, allyUUID, "PARTY_COMBAT", "Tổ Đội Giao Chiến", fmt.Sprintf("Trưởng nhóm %s đã bắt đầu giao chiến! Đội hình của bạn đã tự động tham chiến.", player.Username))
		}
	}
	return initialized
}
