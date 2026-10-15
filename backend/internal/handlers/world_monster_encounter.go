package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

func mapCellBossSpawn(spawnID string) (region string, x, y int, ok bool) {
	parts := strings.Split(spawnID, ":")
	if len(parts) != 4 || parts[0] != "mapcell" {
		return "", 0, 0, false
	}
	x, errX := strconv.Atoi(parts[2])
	y, errY := strconv.Atoi(parts[3])
	if errX != nil || errY != nil {
		return "", 0, 0, false
	}
	return parts[1], x, y, true
}

// startWorldMonsterEncounter promotes a proximity contact into the same
// turn-based combat state used by other PVE encounters.
func startWorldMonsterEncounter(ctx context.Context, monster WorldMonster, targetID string) error {
	if db.DB == nil {
		return nil
	}
	playerID, err := uuid.Parse(targetID)
	if err != nil {
		return err
	}
	tx, err := db.DB.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	player, err := evaluateManaTxEquipped(tx, playerID)
	if err != nil {
		return err
	}
	if player.ActiveCombat != nil || gamefx.IsDeadState(&player) || player.HP <= 0 {
		return nil
	}
	if PlayerIgnoresMonsters(&player) {
		// Flying players cannot be dragged into ground-monster combat.
		return nil
	}
	if err := checkPartyLeaderOrSolo(tx, &player); err != nil {
		return nil
	}
	if err := checkRegionAction(player.CurrentRegion, "combat"); err != nil {
		return nil
	}
	if err := checkRegionAction(player.CurrentRegion, "monster_combat"); err != nil {
		return nil
	}
	if err := checkRegionEquippedGus(player.CurrentRegion, player.GuInventory); err != nil {
		return nil
	}
	var mapInstance sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT map_instance_id::text FROM players WHERE id=$1`, player.ID).Scan(&mapInstance); err != nil {
		return err
	}
	if mapInstance.Valid && mapInstance.String != "" {
		return nil
	}
	if !isRegionWithinScope(monster.RegionID, player.CurrentRegion) || math.Hypot(float64(player.X)-monster.X, float64(player.Y)-monster.Y) > monster.AttackRange ||
		isWorldMonsterSafeZone(player.CurrentRegion, player.X, player.Y) {
		return nil
	}
	current, exists, err := cache.GetWorldMonster(ctx, monster.RegionID, monster.InstanceID)
	if err != nil || !exists || current.State == MonsterStateDefeated || current.State != MonsterStateAttack || current.TargetID != targetID {
		return err
	}
	if math.Hypot(float64(player.X)-current.X, float64(player.Y)-current.Y) > current.AttackRange {
		return nil
	}
	template, ok := models.MonsterRegistry[current.MonsterTemplateID]
	if !ok {
		template, err = db.LoadMonsterTemplate(tx, current.MonsterTemplateID)
		if err != nil {
			return err
		}
	}
	combat := models.BuildCombatStateFromMonsterWithRegion(template, current.Rank, current.SubRank, regionMonsterGuPool(current.RegionID))
	combat.WorldMonsterRegionID = current.RegionID
	combat.WorldMonsterID = current.InstanceID
	combat.WorldMonsterEncounterID = uuid.NewString()
	if current.IsBoss {
		combat.EnemyType = "BOSS"
		if region, x, y, ok := mapCellBossSpawn(current.SpawnID); ok {
			combat.WorldBossCellRegionID, combat.WorldBossCellX, combat.WorldBossCellY = region, x, y
		}
	}
	beforeGus := append(models.GuInventory(nil), player.GuInventory...)
	player.ActiveCombat = StartCombat(tx, &combat, &player)
	if err := db.SyncPlayerGus(tx, player.ID.String(), beforeGus, player.GuInventory); err != nil {
		return err
	}

	var winner string
	var pendingNotifications []notify.Notification
	var afterCombatCommit func()
	if player.CombatSettings.SkipBattle {
		winner, _, _, pendingNotifications, err = AutoResolveCombat(tx, &player, &afterCombatCommit)
		if err != nil {
			resetWorldMonsterAfterEncounter(ctx, current)
			return err
		}
	} else {
		combatJSON, err := json.Marshal(player.ActiveCombat)
		if err != nil {
			resetWorldMonsterAfterEncounter(ctx, current)
			return err
		}
		if err := savePlayerMovement(tx, &player, string(combatJSON)); err != nil {
			resetWorldMonsterAfterEncounter(ctx, current)
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		resetWorldMonsterAfterEncounter(ctx, current)
		return err
	}
	if afterCombatCommit != nil {
		afterCombatCommit()
	}
	for _, notification := range pendingNotifications {
		notify.Publish(notification)
	}
	if player.CombatSettings.SkipBattle {
		if winner == "Phe Ta" {
			finishWorldMonsterEncounter(ctx, current, combat.WorldMonsterEncounterID, player.ID.String())
		} else {
			resetWorldMonsterAfterEncounter(ctx, current)
		}
		PublishPlayerCombatDelta(player.ID, CombatEventEnded, nil)
		return nil
	}
	PublishPlayerCombatDelta(player.ID, CombatEventStarted, player.ActiveCombat)
	return nil
}

func finishWorldMonsterEncounter(ctx context.Context, encounter WorldMonster, encounterID, playerID string) {
	if encounterID == "" {
		return
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	var result cache.WorldMonsterHitResult
	var updated WorldMonster
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		result, updated, err = cache.DefeatWorldMonster(cleanupCtx, encounter.RegionID, encounter.InstanceID, encounterID, time.Now().UTC(),
			monsterRespawnDelay(encounter.RespawnSeconds), monsterTTL(encounter.RespawnSeconds))
		if err == nil || cleanupCtx.Err() != nil || attempt == 2 {
			break
		}
		timer := time.NewTimer(time.Duration(attempt+1) * 100 * time.Millisecond)
		select {
		case <-cleanupCtx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
	if err != nil {
		logger.S.Warnf("finish world monster encounter %s: %v", encounter.InstanceID, err)
		return
	}
	if result == cache.WorldMonsterFirstKill {
		worldMonsterMetrics.kills.Add(1)
		payload := map[string]any{"source_id": playerID, "hp": updated.HP, "max_hp": updated.MaxHP, "encounter_id": encounterID}
		broadcastWorldCombat(updated, "monster_defeated", payload)
		clearDefeatedWorldMonsterCell(updated, playerID)
		return
	}
	// Also finish static-cell cleanup when a prior call changed Redis state but
	// lost its response. DefeatWorldMonster then returns AlreadyDefeated.
	payload := map[string]any{"source_id": playerID, "hp": updated.HP, "max_hp": updated.MaxHP, "encounter_id": encounterID}
	broadcastWorldCombat(updated, "monster_defeated", payload)
	clearDefeatedWorldMonsterCell(updated, playerID)
}

// clearDefeatedWorldMonsterCell hides the static map-cell representation for
// the same respawn window as the runtime monster. Without this, removing the
// defeated runtime actor reveals the old static spawn and can trigger another
// encounter before the actor's respawn timer expires.
func clearDefeatedWorldMonsterCell(monster WorldMonster, playerID string) {
	region, x, y, procedural := proceduralMonsterCell(monster.SpawnID)
	if !procedural {
		region, x, y, procedural = mapCellBossSpawn(monster.SpawnID)
		if procedural {
			// Boss map cells use persistent map-cell respawn state rather than the
			// Redis cooldown used by generated and regular monster spawns.
			if err := MarkPersistentCellCleared(nil, uuid.Nil, region, x, y); err != nil {
				logger.S.Warnf("clear defeated world boss cell %s (%d,%d): %v", region, x, y, err)
			}
			return
		}
		region, x, y = monster.RegionID, int(math.Round(monster.SpawnX)), int(math.Round(monster.SpawnY))
	}

	ttl := monsterRespawnDelay(monster.RespawnSeconds)
	if err := cache.SetCellCleared(region, x, y, playerID, ttl); err != nil {
		logger.S.Warnf("clear defeated world monster cell %s (%d,%d): %v", region, x, y, err)
	}
	if template, ok := models.MonsterRegistry[monster.MonsterTemplateID]; ok {
		if err := cache.BroadcastMonsterKilled(x, y, template.Name, region); err != nil {
			logger.S.Warnf("broadcast defeated world monster cell %s (%d,%d): %v", region, x, y, err)
		}
	}
}

func completeWorldMonsterCombat(ctx context.Context, state *models.CombatState, winner, playerID string) {
	if state == nil || state.WorldMonsterRegionID == "" || state.WorldMonsterID == "" || state.WorldMonsterEncounterID == "" {
		return
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	monster, exists, err := cache.GetWorldMonster(cleanupCtx, state.WorldMonsterRegionID, state.WorldMonsterID)
	if err != nil || !exists {
		return
	}
	if winner == "Phe Ta" {
		finishWorldMonsterEncounter(cleanupCtx, monster, state.WorldMonsterEncounterID, playerID)
	} else {
		resetWorldMonsterAfterEncounter(cleanupCtx, monster)
	}
}

func resetWorldMonsterAfterEncounter(ctx context.Context, before WorldMonster) {
	current, exists, err := cache.GetWorldMonster(ctx, before.RegionID, before.InstanceID)
	if err != nil || !exists || current.State == MonsterStateDefeated {
		return
	}
	current.State, current.TargetID = MonsterStateReturn, ""
	current.Revision++
	current.LastUpdateAt = time.Now().UTC()
	oldBX, oldBY := monsterBucket(int(current.X), int(current.Y))
	if ok, _ := casWorldMonster(ctx, current, current, current.Revision-1, oldBX, oldBY, oldBX, oldBY, monsterTTL(current.RespawnSeconds)); ok {
		broadcastWorldCombat(current, "monster_returned", nil)
	}
}
