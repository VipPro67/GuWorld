package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
	realtimeworld "github.com/vippro67/GuWorld/backend/internal/realtime/world"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// HandleRealtimePlayerDeath applies the ordinary world death settlement after
// the realtime authority has reported a player's defeat. Private combat maps
// keep their own result handlers and never use the persistent-world penalty.
func HandleRealtimePlayerDeath(ctx context.Context, death realtimeworld.DeathEvent) error {
	playerID, err := uuid.Parse(string(death.TargetID))
	if err != nil {
		return nil
	}
	if private, err := privateRealtimeCombatDeath(ctx, death.ZoneKey.InstanceID, playerID); err != nil {
		return err
	} else if private {
		return nil
	}

	handedOff, err := middleware.BeginRealtimeHandoff(playerID)
	if err != nil {
		return fmt.Errorf("checkpoint realtime character before death settlement: %w", err)
	}
	if handedOff {
		defer middleware.AbortRealtimeHandoff(playerID)
	}
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin realtime death settlement: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	player, err := evaluateManaTxEquipped(tx, playerID)
	if err != nil {
		return fmt.Errorf("load defeated character: %w", err)
	}
	deathRegion, deathX, deathY := player.CurrentRegion, player.X, player.Y
	if deathRegion == "" {
		return errors.New("defeated character has no persisted region")
	}
	var winnerID *uuid.UUID
	killerName := ""
	if sourceID, parseErr := uuid.Parse(string(death.SourceID)); parseErr == nil && sourceID != playerID {
		winnerID = &sourceID
		_ = tx.QueryRowContext(ctx, `SELECT username FROM players WHERE id=$1`, sourceID).Scan(&killerName)
	} else if template, ok := models.MonsterRegistry[death.TemplateID]; ok {
		killerName = template.Name
	}

	beforeGus := append(models.GuInventory(nil), player.GuInventory...)
	dropSummary := applyDeathDrop(tx, &player, winnerID, false, DeathPos{
		Region: deathRegion, X: deathX, Y: deathY, KillerName: killerName,
	})
	reduceSatKhiOnDeath(tx, playerID)
	respawn := ResolveDeathRespawn(deathRegion, deathX, deathY)
	placePlayerAt(&player, respawn.Region, respawn.X, respawn.Y)
	player.HP = 1
	if err := db.SyncPlayerGus(tx, playerID.String(), beforeGus, player.GuInventory); err != nil {
		return fmt.Errorf("persist defeated character Gu inventory: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE players SET hp=1, current_mana=$1, primeval_stones=$2, immortal_stones=$3,
			current_region=$4, world_zone_id=$4, world_instance_id='', map_instance_id=NULL,
			x=$5, y=$6, world_x=$5, world_y=$6, world_safe_x=$5, world_safe_y=$6,
			world_location_revision=world_location_revision+1,
			version=version+1
		WHERE id=$7
	`, player.CurrentMana, player.PrimevalStones, player.ImmortalStones,
		player.CurrentRegion, player.X, player.Y, playerID); err != nil {
		return fmt.Errorf("persist realtime death respawn: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO player_world_state_v2 (player_id,hp,shield,map_version,content_version,updated_at)
		VALUES ($1,1,0,'','',NOW())
		ON CONFLICT (player_id) DO UPDATE SET hp=1,shield=0,updated_at=NOW()
	`, playerID); err != nil {
		return fmt.Errorf("persist realtime death checkpoint: %w", err)
	}
	if winnerID != nil {
		if err := recordRealtimePlayerCombatHistory(ctx, tx, *winnerID, playerID, killerName, player.Username, death.ZoneKey.MapID, death.Tick); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit realtime death settlement: %w", err)
	}
	if handedOff {
		middleware.FinishRealtimeHandoff(playerID)
	}
	setWorldPosFromPlayer(&player)
	PublishPlayerRealtimeEvent(playerID, PlayerEventUpdated)
	message := "Đạo hữu đã bị hạ gục. " + respawn.Message()
	if dropSummary != "" {
		message = fmt.Sprintf("Đạo hữu đã bị hạ gục và rơi rớt mất: %s. %s", dropSummary, respawn.Message())
	}
	_ = notify.Send(db.DB, playerID, "death_drop", "Tử đạo - Bị hạ gục", message)
	return nil
}

// HandleRealtimeCombatInstanceDeath finalizes the ascension calamity without
// routing private-map deaths through open-world drops or automatic combat.
func HandleRealtimeCombatInstanceDeath(ctx context.Context, death realtimeworld.DeathEvent) (bool, error) {
	instanceID, err := uuid.Parse(strings.TrimSpace(death.ZoneKey.InstanceID))
	if err != nil {
		return false, nil
	}
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin private combat death settlement: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var mode string
	var sessionPayload []byte
	err = tx.QueryRowContext(ctx, `
		SELECT mode,payload FROM realtime_combat_sessions WHERE id=$1 AND status='active' FOR UPDATE
	`, instanceID).Scan(&mode, &sessionPayload)
	if errors.Is(err, sql.ErrNoRows) {
		// Multiple deaths may already be queued from the same tick when the
		// first one completes a tournament match. Never send those late virtual
		// events through ordinary world death settlement.
		var tournamentSession bool
		if lookupErr := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM realtime_combat_sessions WHERE id=$1 AND mode='tournament')`, instanceID).Scan(&tournamentSession); lookupErr == nil && tournamentSession {
			return true, nil
		}
		var sectWarSession bool
		if lookupErr := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM realtime_combat_sessions WHERE id=$1 AND mode='sect_war')`, instanceID).Scan(&sectWarSession); lookupErr == nil && sectWarSession {
			return true, nil
		}
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("load private combat death session: %w", err)
	}
	if mode == "arena" {
		winnerID, loserID, handled, arenaErr := finalizeRealtimeArenaDeath(ctx, tx, instanceID, sessionPayload, death)
		if arenaErr != nil {
			return true, arenaErr
		}
		if !handled {
			return true, nil
		}
		if err := tx.Commit(); err != nil {
			return true, fmt.Errorf("commit realtime arena result: %w", err)
		}
		PublishPlayerRealtimeEvent(winnerID, PlayerEventUpdated)
		PublishPlayerRealtimeEvent(loserID, PlayerEventUpdated)
		return true, nil
	}
	if mode == "tournament" {
		restored, tournamentErr := settleRealtimeTournamentDeath(ctx, tx, instanceID, sessionPayload, death)
		if tournamentErr != nil {
			return true, tournamentErr
		}
		if len(restored) == 0 {
			return true, nil
		}
		var handoffs []uuid.UUID
		for _, id := range restored {
			handed, handoffErr := middleware.BeginRealtimeHandoff(id)
			if handoffErr != nil {
				for _, previous := range handoffs {
					middleware.AbortRealtimeHandoff(previous)
				}
				return true, fmt.Errorf("checkpoint tournament participant %s: %w", id, handoffErr)
			}
			if handed {
				handoffs = append(handoffs, id)
			}
		}
		if err := tx.Commit(); err != nil {
			for _, id := range handoffs {
				middleware.AbortRealtimeHandoff(id)
			}
			return true, fmt.Errorf("commit realtime tournament result: %w", err)
		}
		for _, id := range handoffs {
			middleware.FinishRealtimeHandoff(id)
		}
		for _, id := range restored {
			PublishPlayerRealtimeEvent(id, PlayerEventUpdated)
		}
		var tournament realtimeTournamentSessionMetadata
		if json.Unmarshal(sessionPayload, &tournament) == nil {
			if tournament.Competition == "death_arena" {
				invalidateTournamentCache(tournament.TournamentID)
				BroadcastDeathArenaEvent("match_updated", tournament.TournamentID, map[string]interface{}{"match_id": tournament.MatchID})
			} else if tournament.Competition == "martial_arena" {
				invalidateMartialCache(tournament.TournamentID)
				BroadcastMartialArenaEvent("match_updated", tournament.TournamentID, map[string]interface{}{"match_id": tournament.MatchID})
			}
		}
		return true, nil
	}
	if mode == "sect_war" {
		rows, queryErr := tx.QueryContext(ctx, `SELECT player_id FROM realtime_combat_session_participants WHERE session_id=$1 AND status IN ('active','eliminated') ORDER BY player_id`, instanceID)
		if queryErr != nil {
			return true, fmt.Errorf("list sect war participants for handoff: %w", queryErr)
		}
		var allIDs []uuid.UUID
		var handed []uuid.UUID
		for rows.Next() {
			var id uuid.UUID
			if scanErr := rows.Scan(&id); scanErr != nil {
				rows.Close()
				return true, scanErr
			}
			allIDs = append(allIDs, id)
		}
		if rowsErr := rows.Err(); rowsErr != nil {
			rows.Close()
			return true, rowsErr
		}
		rows.Close()
		for _, id := range allIDs {
			wasHanded, handoffErr := middleware.BeginRealtimeHandoff(id)
			if handoffErr != nil {
				for _, previous := range handed {
					middleware.AbortRealtimeHandoff(previous)
				}
				return true, fmt.Errorf("checkpoint sect war participant %s: %w", id, handoffErr)
			}
			if wasHanded {
				handed = append(handed, id)
			}
		}
		restored, settlementErr := settleRealtimeSectWarDeath(ctx, tx, instanceID, sessionPayload, death)
		if settlementErr != nil {
			for _, id := range handed {
				middleware.AbortRealtimeHandoff(id)
			}
			return true, settlementErr
		}
		if err := tx.Commit(); err != nil {
			for _, id := range handed {
				middleware.AbortRealtimeHandoff(id)
			}
			return true, fmt.Errorf("commit realtime sect war settlement: %w", err)
		}
		restoredSet := make(map[uuid.UUID]struct{}, len(restored))
		for _, id := range restored {
			restoredSet[id] = struct{}{}
			PublishPlayerRealtimeEvent(id, PlayerEventUpdated)
		}
		for _, id := range handed {
			if _, shouldLeave := restoredSet[id]; shouldLeave {
				middleware.FinishRealtimeHandoff(id)
			} else {
				middleware.AbortRealtimeHandoff(id)
			}
		}
		return true, nil
	}
	if mode == "secret_realm" {
		playerID, parseErr := uuid.Parse(string(death.TargetID))
		handedOff := false
		if death.TargetKind == string(sim.EntityPlayer) && parseErr == nil {
			var handoffErr error
			handedOff, handoffErr = middleware.BeginRealtimeHandoff(playerID)
			if handoffErr != nil {
				return true, fmt.Errorf("checkpoint spatial realm participant before death settlement: %w", handoffErr)
			}
			if handedOff {
				defer middleware.AbortRealtimeHandoff(playerID)
			}
		}
		playerID, handled, realmErr := finalizeRealtimeSpatialRealmDeath(ctx, tx, instanceID, death)
		if realmErr != nil {
			return true, realmErr
		}
		if !handled {
			return false, nil
		}
		if err := tx.Commit(); err != nil {
			return true, fmt.Errorf("commit realtime spatial realm death: %w", err)
		}
		if handedOff {
			middleware.FinishRealtimeHandoff(playerID)
		}
		ClearPendingPlayerQueues(playerID.String())
		PublishPlayerRealtimeEvent(playerID, PlayerEventUpdated)
		return true, nil
	}
	var playerID uuid.UUID
	var originZone string
	var payload []byte
	err = tx.QueryRowContext(ctx, `
		SELECT p.player_id,p.payload,s.origin_zone_id
		FROM realtime_combat_sessions s
		JOIN realtime_combat_session_participants p ON p.session_id=s.id
		WHERE s.id=$1 AND s.status='active' AND p.status='active'
		ORDER BY p.player_id LIMIT 1 FOR UPDATE OF p
	`, instanceID).Scan(&playerID, &payload, &originZone)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("load private combat participant: %w", err)
	}
	if mode != "ascension" {
		return false, nil
	}
	var config struct {
		TemplateID  string  `json:"template_id"`
		CurrentHP   float64 `json:"current_hp"`
		CurrentMana float64 `json:"current_mana"`
	}
	if err := json.Unmarshal(payload, &config); err != nil {
		return true, fmt.Errorf("decode ascension combat checkpoint: %w", err)
	}
	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		return true, fmt.Errorf("load ascension combatant: %w", err)
	}
	player.CurrentMana = config.CurrentMana
	if death.TargetKind == string(sim.EntityPlayer) && death.TargetID == sim.EntityID(playerID.String()) {
		session, err := loadAscensionSession(tx, &player)
		if err != nil {
			return true, err
		}
		if session != nil && session.Step == AscensionStepCalamity {
			if err := failAscensionSession(tx, &player, session, "Đại bại trước Địa Tai Bản Yếu"); err != nil {
				return true, err
			}
		}
		player.HP = 1
	} else if death.TemplateID == config.TemplateID {
		session, err := loadAscensionSession(tx, &player)
		if err != nil {
			return true, fmt.Errorf("load ascension session after realtime victory: %w", err)
		}
		if session == nil {
			return true, fmt.Errorf("ascension session missing during realtime victory")
		}
		if session.Step == AscensionStepCalamity {
			if err := completeAscensionCalamity(tx, playerID); err != nil {
				return true, fmt.Errorf("advance ascension after realtime victory: %w", err)
			}
		} else if session.Step != AscensionStepFailed {
			return true, fmt.Errorf("ascension session is no longer awaiting calamity resolution")
		}
		if config.CurrentHP > 0 {
			player.HP = config.CurrentHP
		}
	} else {
		return true, nil
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE players SET hp=$1,current_mana=$2,world_zone_id=$3,world_instance_id='',
			world_location_revision=world_location_revision+1
		WHERE id=$4
	`, player.HP, player.CurrentMana, originZone, playerID); err != nil {
		return true, fmt.Errorf("restore ascension combatant to origin map: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO player_world_state_v2 (player_id,hp,shield,map_version,content_version,updated_at)
		VALUES ($1,$2,0,'','',NOW())
		ON CONFLICT (player_id) DO UPDATE SET hp=EXCLUDED.hp,shield=0,updated_at=NOW()
	`, playerID, player.HP); err != nil {
		return true, fmt.Errorf("persist ascension combatant checkpoint: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE realtime_combat_session_participants SET status='completed' WHERE session_id=$1 AND player_id=$2`, instanceID, playerID); err != nil {
		return true, fmt.Errorf("complete ascension combat participant: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE realtime_combat_sessions SET status='completed',ended_at=NOW() WHERE id=$1`, instanceID); err != nil {
		return true, fmt.Errorf("complete ascension combat session: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return true, fmt.Errorf("commit ascension combat result: %w", err)
	}
	ClearPendingPlayerQueues(playerID.String())
	PublishPlayerRealtimeEvent(playerID, PlayerEventUpdated)
	return true, nil
}

func finalizeRealtimeSpatialRealmDeath(ctx context.Context, tx *sql.Tx, instanceID uuid.UUID, death realtimeworld.DeathEvent) (uuid.UUID, bool, error) {
	if death.TargetKind != string(sim.EntityPlayer) {
		return uuid.Nil, false, nil
	}
	playerID, err := uuid.Parse(string(death.TargetID))
	if err != nil {
		return uuid.Nil, false, nil
	}
	var returnRegion, realmZone, deathPenalty, realmRegion string
	var returnX, returnY, entryX, entryY int
	var memberStatus string
	err = tx.QueryRowContext(ctx, `
		SELECT m.return_region_id,m.return_x,m.return_y,p.status,s.zone_id,i.region_id,
			COALESCE(t.rules->>'death_penalty','EVICT'),COALESCE(r.entry_x,0),COALESCE(r.entry_y,0)
		FROM realtime_combat_sessions s
		JOIN realtime_combat_session_participants p ON p.session_id=s.id AND p.player_id=$2
		JOIN shared_spatial_instances i ON i.id=s.id
		JOIN shared_spatial_instance_members m ON m.instance_id=i.id AND m.player_id=p.player_id
		JOIN instance_templates t ON t.id=i.template_id
		JOIN regions r ON r.id=i.region_id
		WHERE s.id=$1 AND s.mode='secret_realm' AND s.status='active'
		FOR UPDATE OF p,m
	`, instanceID, playerID).Scan(&returnRegion, &returnX, &returnY, &memberStatus, &realmZone, &realmRegion, &deathPenalty, &entryX, &entryY)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("load realtime spatial realm death state: %w", err)
	}
	if memberStatus != "active" {
		return uuid.Nil, false, nil
	}
	deathPenalty = strings.ToUpper(strings.TrimSpace(deathPenalty))
	// Older Region.csv imports and seeded templates used this name for the
	// same behavior. Normalize it at read time so existing installations work.
	if deathPenalty == "EXPEL_TO_ENTRANCE" {
		deathPenalty = "EVICT"
	}
	if deathPenalty == "SAFE_RESPAWN" {
		if _, err := tx.ExecContext(ctx, `
			UPDATE players SET hp=1,current_region=$1,x=$2,y=$3,world_zone_id=$4,world_instance_id=$5,
				map_instance_id=$6,world_x=$2,world_y=$3,world_safe_x=$2,world_safe_y=$3,
				world_location_revision=world_location_revision+1,version=version+1
			WHERE id=$7
		`, realmRegion, entryX, entryY, realmZone, "realm:"+instanceID.String()+":respawn:"+uuid.NewString(), instanceID, playerID); err != nil {
			return uuid.Nil, false, fmt.Errorf("respawn spatial realm participant: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO player_world_state_v2 (player_id,hp,shield,map_version,content_version,updated_at)
			VALUES ($1,1,0,'','',NOW()) ON CONFLICT (player_id)
			DO UPDATE SET hp=1,shield=0,updated_at=NOW()
		`, playerID); err != nil {
			return uuid.Nil, false, fmt.Errorf("persist spatial realm respawn checkpoint: %w", err)
		}
		return playerID, true, nil
	}
	if deathPenalty != "EVICT" {
		return uuid.Nil, false, fmt.Errorf("unsupported spatial realm death penalty %q", deathPenalty)
	}
	var returnZoneType, returnMap string
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(zone_type,''),COALESCE((SELECT world_map_id FROM region_world_maps WHERE region_id=$1),'') FROM regions WHERE id=$1`, returnRegion).Scan(&returnZoneType, &returnMap); err != nil {
		return uuid.Nil, false, fmt.Errorf("resolve spatial realm death return map: %w", err)
	}
	returnZone := realtimeMapID(returnRegion, returnZoneType, returnMap)
	if _, err := tx.ExecContext(ctx, `
		UPDATE players SET hp=1,current_region=$1,x=$2,y=$3,map_instance_id=NULL,world_zone_id=$4,world_instance_id='',
			world_x=$2,world_y=$3,world_safe_x=$2,world_safe_y=$3,world_location_revision=world_location_revision+1,
			version=version+1 WHERE id=$5
	`, returnRegion, returnX, returnY, returnZone, playerID); err != nil {
		return uuid.Nil, false, fmt.Errorf("evict defeated spatial realm participant: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE shared_spatial_instance_members SET exited_at=NOW(),exit_reason='DEATH' WHERE instance_id=$1 AND player_id=$2 AND exited_at IS NULL`, instanceID, playerID); err != nil {
		return uuid.Nil, false, fmt.Errorf("complete spatial realm membership after death: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE realtime_combat_session_participants SET status='completed' WHERE session_id=$1 AND player_id=$2 AND status='active'`, instanceID, playerID); err != nil {
		return uuid.Nil, false, fmt.Errorf("complete spatial realm participant after death: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO player_world_state_v2 (player_id,hp,shield,map_version,content_version,updated_at)
		VALUES ($1,1,0,'','',NOW()) ON CONFLICT (player_id)
		DO UPDATE SET hp=1,shield=0,updated_at=NOW()
	`, playerID); err != nil {
		return uuid.Nil, false, fmt.Errorf("persist spatial realm death checkpoint: %w", err)
	}
	return playerID, true, nil
}

type realtimeArenaMetadata struct {
	ChallengerID   string `json:"challenger_id"`
	ChallengedID   string `json:"challenged_id"`
	ChallengerName string `json:"challenger_name"`
	ChallengedName string `json:"challenged_name"`
	ChallengerRank int    `json:"challenger_rank"`
	ChallengedRank int    `json:"challenged_rank"`
	Region         string `json:"region"`
}

func finalizeRealtimeArenaDeath(ctx context.Context, tx *sql.Tx, sessionID uuid.UUID, rawMetadata []byte, death realtimeworld.DeathEvent) (uuid.UUID, uuid.UUID, bool, error) {
	var metadata realtimeArenaMetadata
	if err := json.Unmarshal(rawMetadata, &metadata); err != nil {
		return uuid.Nil, uuid.Nil, false, fmt.Errorf("decode arena session metadata: %w", err)
	}
	challengerID, err := uuid.Parse(metadata.ChallengerID)
	if err != nil {
		return uuid.Nil, uuid.Nil, false, fmt.Errorf("invalid arena challenger id: %w", err)
	}
	challengedID, err := uuid.Parse(metadata.ChallengedID)
	if err != nil {
		return uuid.Nil, uuid.Nil, false, fmt.Errorf("invalid arena opponent id: %w", err)
	}
	if death.TargetKind != string(sim.EntityPlayer) {
		return uuid.Nil, uuid.Nil, false, nil
	}
	loserID, err := uuid.Parse(string(death.TargetID))
	if err != nil || (loserID != challengerID && loserID != challengedID) {
		return uuid.Nil, uuid.Nil, false, nil
	}
	winnerID, err := uuid.Parse(string(death.SourceID))
	if err != nil || winnerID == loserID || (winnerID != challengerID && winnerID != challengedID) {
		return uuid.Nil, uuid.Nil, false, fmt.Errorf("arena death event does not identify the opposing participant as winner")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT player_id,payload FROM realtime_combat_session_participants
		WHERE session_id=$1 AND status='active' ORDER BY player_id FOR UPDATE
	`, sessionID)
	if err != nil {
		return uuid.Nil, uuid.Nil, false, fmt.Errorf("load arena participants: %w", err)
	}
	checkpoints := make(map[uuid.UUID]RealtimeCombatInstanceVitals, 2)
	for rows.Next() {
		var id uuid.UUID
		var raw []byte
		if err := rows.Scan(&id, &raw); err != nil {
			rows.Close()
			return uuid.Nil, uuid.Nil, false, fmt.Errorf("scan arena participant: %w", err)
		}
		var state RealtimeCombatInstanceVitals
		if err := json.Unmarshal(raw, &state); err != nil {
			rows.Close()
			return uuid.Nil, uuid.Nil, false, fmt.Errorf("decode arena checkpoint: %w", err)
		}
		checkpoints[id] = state
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return uuid.Nil, uuid.Nil, false, fmt.Errorf("read arena participants: %w", err)
	}
	rows.Close()
	if len(checkpoints) != 2 || checkpoints[challengerID].OriginZoneID == "" || checkpoints[challengedID].OriginZoneID == "" {
		return uuid.Nil, uuid.Nil, false, fmt.Errorf("arena session is missing a participant checkpoint")
	}
	challengerWon := winnerID == challengerID
	qiReward := 2.0
	coinReward := int64(1)
	if challengerWon {
		qiReward = combatHumanQiReward(metadata.ChallengedRank)
		coinReward = 50
	}
	if InventorySvc == nil {
		return uuid.Nil, uuid.Nil, false, fmt.Errorf("inventory service unavailable for arena reward")
	}
	if err := InventorySvc.AddItem(tx, challengerID.String(), "arena_coin", "materials", coinReward); err != nil {
		return uuid.Nil, uuid.Nil, false, fmt.Errorf("award arena coins: %w", err)
	}
	if challengerWon {
		if metadata.ChallengerRank > metadata.ChallengedRank {
			if _, err := tx.ExecContext(ctx, `UPDATE arena_rankings SET rank=-1 WHERE player_id=$1`, challengerID); err != nil {
				return uuid.Nil, uuid.Nil, false, err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE arena_rankings SET rank=$1,updated_at=NOW() WHERE player_id=$2`, metadata.ChallengerRank, challengedID); err != nil {
				return uuid.Nil, uuid.Nil, false, err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE arena_rankings SET rank=$1,updated_at=NOW() WHERE player_id=$2`, metadata.ChallengedRank, challengerID); err != nil {
				return uuid.Nil, uuid.Nil, false, err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE players SET arena_rank=$1 WHERE id=$2`, metadata.ChallengerRank, challengedID); err != nil {
				return uuid.Nil, uuid.Nil, false, err
			}
			if _, err := tx.ExecContext(ctx, `UPDATE players SET arena_rank=$1 WHERE id=$2`, metadata.ChallengedRank, challengerID); err != nil {
				return uuid.Nil, uuid.Nil, false, err
			}
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE players SET human_qi=LEAST(human_qi+$1,$2) WHERE id=$3`, qiReward, MaxHumanQi, challengerID); err != nil {
		return uuid.Nil, uuid.Nil, false, fmt.Errorf("settle arena human qi: %w", err)
	}
	for id, state := range checkpoints {
		if _, err := tx.ExecContext(ctx, `
			UPDATE players SET world_zone_id=$2,world_instance_id='',world_location_revision=world_location_revision+1
			WHERE id=$1
		`, id, state.OriginZoneID); err != nil {
			return uuid.Nil, uuid.Nil, false, fmt.Errorf("restore arena participant location: %w", err)
		}
	}
	logs, err := realtimeCombatHistoryJSON(ctx, tx, sessionID)
	if err != nil {
		return uuid.Nil, uuid.Nil, false, err
	}
	if string(logs) == "[]" {
		logs, _ = json.Marshal([]map[string]interface{}{{
			"type": "REALTIME_PLAYER_DEATH", "winner_id": winnerID.String(), "loser_id": loserID.String(), "tick": death.Tick,
		}})
	}
	summary, _ := json.Marshal(map[string]interface{}{
		"winner_id": winnerID.String(), "loser_id": loserID.String(), "arena_coins_awarded": coinReward,
		"rank_swapped": challengerWon && metadata.ChallengerRank > metadata.ChallengedRank,
	})
	challengerCombatants, _ := json.Marshal([]map[string]interface{}{
		{"id": challengerID.String(), "name": metadata.ChallengerName, "kind": "ally"},
		{"id": challengedID.String(), "name": metadata.ChallengedName, "kind": "enemy"},
	})
	challengedCombatants, _ := json.Marshal([]map[string]interface{}{
		{"id": challengedID.String(), "name": metadata.ChallengedName, "kind": "ally"},
		{"id": challengerID.String(), "name": metadata.ChallengerName, "kind": "enemy"},
	})
	challengerWinner, challengedWinner := "Phe Địch", "Phe Ta"
	if challengerWon {
		challengerWinner, challengedWinner = "Phe Ta", "Phe Địch"
	}
	for _, entry := range []struct {
		id, winner, enemyName string
		combatants            []byte
	}{
		{challengerID.String(), challengerWinner, metadata.ChallengedName, challengerCombatants},
		{challengedID.String(), challengedWinner, metadata.ChallengerName, challengedCombatants},
	} {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO combat_history (player_id,winner,logs,summary,combatants,enemy_name,enemy_type,region)
			VALUES ($1,$2,$3,$4,$5,$6,'PVP',$7)
		`, entry.id, entry.winner, logs, summary, entry.combatants, entry.enemyName, metadata.Region); err != nil {
			return uuid.Nil, uuid.Nil, false, fmt.Errorf("persist realtime arena history: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE realtime_combat_session_participants SET status='completed' WHERE session_id=$1 AND status='active'`, sessionID); err != nil {
		return uuid.Nil, uuid.Nil, false, fmt.Errorf("complete arena participants: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE realtime_combat_sessions SET status='completed',ended_at=NOW() WHERE id=$1 AND status='active'`, sessionID); err != nil {
		return uuid.Nil, uuid.Nil, false, fmt.Errorf("complete arena session: %w", err)
	}
	return winnerID, loserID, true, nil
}

// privateRealtimeCombatDeath identifies session-scoped combat maps so a
// virtual defeat cannot run persistent-world inventory drops or respawning.
func privateRealtimeCombatDeath(ctx context.Context, instanceID string, playerID uuid.UUID) (bool, error) {
	id, err := uuid.Parse(strings.TrimSpace(instanceID))
	if err != nil {
		return false, nil
	}
	var mode string
	err = db.DB.QueryRowContext(ctx, `
		SELECT mode FROM realtime_combat_instances WHERE id=$1 AND player_id=$2
	`, id, playerID).Scan(&mode)
	if errors.Is(err, sql.ErrNoRows) {
		err = db.DB.QueryRowContext(ctx, `
			SELECT s.mode FROM realtime_combat_sessions s
			JOIN realtime_combat_session_participants p ON p.session_id=s.id
			WHERE s.id=$1 AND p.player_id=$2
		`, id, playerID).Scan(&mode)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("resolve realtime combat death mode: %w", err)
	}
	if mode == "training" {
		return true, nil
	}
	return true, fmt.Errorf("realtime death settlement for private mode %q is not configured", mode)
}
