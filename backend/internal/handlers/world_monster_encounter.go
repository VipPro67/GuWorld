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
	combat := models.BuildCombatStateFromMonster(template, current.Rank, current.SubRank)
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
	if player.CombatSettings.SkipBattle {
		winner, _, _, err = AutoResolveCombat(tx, &player)
		if err != nil {
			return err
		}
	} else {
		combatJSON, err := json.Marshal(player.ActiveCombat)
		if err != nil {
			return err
		}
		if err := savePlayerMovement(tx, &player, string(combatJSON)); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
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
	result, updated, err := cache.DefeatWorldMonster(ctx, encounter.RegionID, encounter.InstanceID, encounterID, time.Now().UTC(),
		time.Duration(encounter.RespawnSeconds)*time.Second, monsterTTL(encounter.RespawnSeconds))
	if err != nil {
		logger.S.Warnf("finish world monster encounter %s: %v", encounter.InstanceID, err)
		return
	}
	if result == cache.WorldMonsterFirstKill {
		worldMonsterMetrics.kills.Add(1)
		payload := map[string]any{"source_id": playerID, "hp": updated.HP, "max_hp": updated.MaxHP, "encounter_id": encounterID}
		broadcastWorldCombat(updated, "monster_defeated", payload)
		if region, x, y, procedural := proceduralMonsterCell(updated.SpawnID); procedural {
			_ = cache.SetCellCleared(region, x, y, playerID, clearedCellCooldown)
			if template, ok := models.MonsterRegistry[updated.MonsterTemplateID]; ok {
				_ = cache.BroadcastMonsterKilled(x, y, template.Name, region)
			}
		}
		return
	}
	if result == cache.WorldMonsterAlreadyDefeated {
		worldMonsterMetrics.preventedDoubleKills.Add(1)
	}
}

func completeWorldMonsterCombat(ctx context.Context, state *models.CombatState, winner, playerID string) {
	if state == nil || state.WorldMonsterRegionID == "" || state.WorldMonsterID == "" || state.WorldMonsterEncounterID == "" {
		return
	}
	monster, exists, err := cache.GetWorldMonster(ctx, state.WorldMonsterRegionID, state.WorldMonsterID)
	if err != nil || !exists {
		return
	}
	if winner == "Phe Ta" {
		finishWorldMonsterEncounter(ctx, monster, state.WorldMonsterEncounterID, playerID)
	} else {
		resetWorldMonsterAfterEncounter(ctx, monster)
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
