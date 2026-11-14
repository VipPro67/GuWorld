package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
	"github.com/vippro67/GuWorld/backend/internal/worker"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

const combatDecisionTimeout = 45 * time.Second

type combatSessionRow struct {
	ID            uuid.UUID
	PlayerID      uuid.UUID
	ParticipantID uuid.NullUUID
	Revision      int64
	Status        string
	InitialState  []byte
	Loadout       []byte
	Commands      []byte
	PendingActor  string
	DeadlineAt    sql.NullTime
}

func StartCombatSessionWorker(ctx context.Context) {
	worker.RunEvery(ctx, time.Second*5, "Combat Session Timeout Worker", func(ctx context.Context) {
		processExpiredCombatSessions(ctx)
	})
}

func processExpiredCombatSessions(ctx context.Context) {
	if db.DB == nil {
		return
	}
	rows, err := db.DB.QueryContext(ctx, `SELECT id, player_id FROM combat_sessions WHERE status='active' AND deadline_at <= NOW() ORDER BY deadline_at LIMIT 20`)
	if err != nil {
		logger.S.Errorf("combat session timeout: query: %v", err)
		return
	}
	type expired struct{ id, player uuid.UUID }
	var sessions []expired
	for rows.Next() {
		var item expired
		if rows.Scan(&item.id, &item.player) == nil {
			sessions = append(sessions, item)
		}
	}
	rows.Close()
	for _, item := range sessions {
		if err := autoCombatSessionCommand(ctx, item.id, item.player); err != nil {
			logger.S.Errorf("combat session timeout: resolve %s: %v", item.id, err)
		}
	}
	processExpiredPvpPreparation(ctx)
}

func processExpiredPvpPreparation(ctx context.Context) {
	rows, err := db.DB.QueryContext(ctx, `SELECT p.id FROM players p
		WHERE p.active_combat->>'enemy_type'='PVP'
		  AND p.active_combat->>'combat_mode'='real'
		  AND p.active_combat->>'combat_owner_id'=p.id::text
		  AND COALESCE(NULLIF(p.active_combat->>'combat_started_at','')::timestamptz, NOW()-INTERVAL '1 minute') <= NOW()-($1 * INTERVAL '1 millisecond')
		  AND NOT EXISTS (SELECT 1 FROM combat_sessions s WHERE s.player_id=p.id AND s.status='active')
		ORDER BY p.id LIMIT 20`, combatDecisionTimeout.Milliseconds())
	if err != nil {
		logger.S.Errorf("combat PvP preparation timeout: query: %v", err)
		return
	}
	var playerIDs []uuid.UUID
	for rows.Next() {
		var playerID uuid.UUID
		if rows.Scan(&playerID) == nil {
			playerIDs = append(playerIDs, playerID)
		}
	}
	rows.Close()
	for _, playerID := range playerIDs {
		if err := autoResolveExpiredPvpPreparation(ctx, playerID); err != nil {
			logger.S.Errorf("combat PvP preparation timeout: resolve %s: %v", playerID, err)
		}
	}
}

func autoResolveExpiredPvpPreparation(ctx context.Context, playerID uuid.UUID) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := loadCombatSession(tx, playerID, nil, true); err == nil {
		return nil
	} else if err != sql.ErrNoRows {
		return err
	}
	player, err := evaluateManaTxEquipped(tx, playerID)
	if err != nil {
		return err
	}
	if player.ActiveCombat == nil || player.ActiveCombat.EnemyType != "PVP" || player.ActiveCombat.CombatMode != "real" || player.ActiveCombat.CombatOwnerID != playerID.String() {
		return nil
	}
	if !player.ActiveCombat.CombatStartedAt.IsZero() && time.Since(player.ActiveCombat.CombatStartedAt) < combatDecisionTimeout {
		return nil
	}
	var sessionExists bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM combat_sessions WHERE player_id=$1 AND status='active')`, playerID).Scan(&sessionExists); err != nil {
		return err
	}
	if sessionExists {
		return nil
	}
	resolution, err := ResolveCombat(tx, &player, nil, nil)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	PublishPlayerCombatDelta(player.ID, CombatEventEnded, nil)
	if resolution.AfterCommit != nil {
		resolution.AfterCommit()
	}
	for _, notification := range resolution.Notifications {
		notify.Publish(notification)
	}
	setWorldPosFromPlayer(&player)
	return nil
}

func autoCombatSessionCommand(ctx context.Context, sessionID, playerID uuid.UUID) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	row, err := loadCombatSession(tx, playerID, &sessionID, true)
	if err != nil {
		return err
	}
	if !row.DeadlineAt.Valid || row.DeadlineAt.Time.After(time.Now()) {
		return nil
	}
	state, _, _, err := replayCombatSession(row, nil)
	if err != nil {
		return err
	}
	actorID := state.Interactive.PendingActor
	if actorID == "" {
		return errors.New("expired combat session has no pending actor")
	}
	var actor *simulator.CombatantAuto
	for _, side := range [][]*simulator.CombatantAuto{state.Allies, state.Enemies} {
		for _, combatant := range side {
			if combatant.ID == actorID {
				actor = combatant
				break
			}
		}
	}
	if actor == nil {
		return errors.New("pending combat actor not found")
	}
	// Persist AUTO as the command so the simulator chooses a currently legal
	// action from this actor's loadout/tactics. Replaying the command makes the
	// timeout result deterministic and avoids getting stuck when a basic attack
	// is unavailable because of status or target rules.
	command := simulator.CombatCommand{ActorID: actorID, ActionType: "AUTO"}
	commands, err := decodeCombatCommands(row.Commands)
	if err != nil {
		return err
	}
	commands = append(commands, command)
	state, _, _, err = replayCombatSession(row, &command)
	if err != nil || state.Interactive.Error != "" || state.Interactive.Consumed != len(commands) {
		return errors.New("auto-selected combat action was rejected")
	}
	commandsJSON, err := json.Marshal(commands)
	if err != nil {
		return err
	}
	commandID := uuid.New()
	row.Revision++
	var result *CombatResolution
	var player models.Player
	if state.Interactive.PendingActor == "" {
		var loadout simulator.CombatLoadout
		if err := json.Unmarshal(row.Loadout, &loadout); err != nil {
			return err
		}
		player, err = evaluateManaTxEquipped(tx, playerID)
		if err != nil {
			return err
		}
		var initial simulator.CombatState
		if err := json.Unmarshal(row.InitialState, &initial); err != nil {
			return err
		}
		initial.Interactive.Commands = commands
		result, err = ResolveCombatFromSession(tx, &player, &loadout, &initial)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`UPDATE combat_sessions SET commands=$1, revision=$2, status='complete', pending_actor='', deadline_at=NULL, updated_at=NOW() WHERE id=$3`, commandsJSON, row.Revision, row.ID)
		row.Status = "complete"
	} else {
		deadline := time.Now().Add(combatDecisionTimeout)
		_, err = tx.Exec(`UPDATE combat_sessions SET commands=$1, revision=$2, pending_actor=$3, deadline_at=$4, updated_at=NOW() WHERE id=$5`, commandsJSON, row.Revision, state.Interactive.PendingActor, deadline, row.ID)
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO combat_session_commands(session_id, command_id, revision) VALUES($1,$2,$3)`, row.ID, commandID, row.Revision); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if result != nil {
		publishCombatSessionEvent(row, "complete", "")
		PublishPlayerCombatDelta(playerID, CombatEventEnded, nil)
		if result.AfterCommit != nil {
			result.AfterCommit()
		}
		for _, notification := range result.Notifications {
			notify.Publish(notification)
		}
		setWorldPosFromPlayer(&player)
	} else {
		publishCombatSessionEvent(row, "active", state.Interactive.PendingActor)
	}
	return nil
}

func decodeCombatCommands(raw []byte) ([]simulator.CombatCommand, error) {
	var commands []simulator.CombatCommand
	if err := json.Unmarshal(raw, &commands); err != nil {
		return nil, err
	}
	return commands, nil
}

type combatSessionCommandRequest struct {
	CommandID        uuid.UUID               `json:"command_id" binding:"required"`
	ExpectedRevision int64                   `json:"expected_revision"`
	Command          simulator.CombatCommand `json:"command" binding:"required"`
}

func loadCombatSession(tx *sql.Tx, playerID uuid.UUID, sessionID *uuid.UUID, lock bool) (combatSessionRow, error) {
	var row combatSessionRow
	query := `SELECT id, player_id, participant_id, revision, status, initial_state, loadout, commands, deadline_at
		FROM combat_sessions WHERE (player_id = $1 OR participant_id = $1)`
	args := []interface{}{playerID}
	if sessionID != nil {
		query += ` AND id = $2`
		args = append(args, *sessionID)
	} else {
		query += ` AND status = 'active'`
	}
	if lock {
		query += ` FOR UPDATE`
	}
	err := tx.QueryRow(query, args...).Scan(&row.ID, &row.PlayerID, &row.ParticipantID, &row.Revision, &row.Status, &row.InitialState, &row.Loadout, &row.Commands, &row.DeadlineAt)
	return row, err
}

func replayCombatSession(row combatSessionRow, extra *simulator.CombatCommand) (*simulator.CombatState, []simulator.CombatLog, []simulator.CombatCommand, error) {
	var state simulator.CombatState
	if err := json.Unmarshal(row.InitialState, &state); err != nil {
		return nil, nil, nil, err
	}
	var commands []simulator.CombatCommand
	if err := json.Unmarshal(row.Commands, &commands); err != nil {
		return nil, nil, nil, err
	}
	if extra != nil {
		commands = append(commands, *extra)
	}
	if state.Interactive == nil {
		return nil, nil, nil, errors.New("session has no interactive controller")
	}
	state.Interactive.Commands = commands
	logs := simulator.Simulate(&state)
	return &state, logs, commands, nil
}

func combatSessionResponse(row combatSessionRow, state *simulator.CombatState, logs []simulator.CombatLog) gin.H {
	var snapshot *simulator.CombatSnapshot
	for i := len(logs) - 1; i >= 0; i-- {
		if logs[i].Snapshot != nil {
			snapshot = logs[i].Snapshot
			break
		}
	}
	response := gin.H{
		"session_id": row.ID, "revision": row.Revision, "status": row.Status,
		"snapshot": snapshot, "logs": logs,
	}
	if state != nil && state.Interactive != nil {
		response["pending_actor"] = state.Interactive.PendingActor
		response["command_error"] = state.Interactive.Error
		if state.Interactive.TribulationEventID != "" {
			response["tribulation_event_id"] = state.Interactive.TribulationEventID
		}
		// Send the current combat roster and the pending actor's prepared actions
		// so clients can render legal choices without rebuilding simulator rules.
		response["allies"] = combatSessionSide(state.Allies, state.Interactive.PendingActor, state.Enemies)
		response["enemies"] = combatSessionSide(state.Enemies, state.Interactive.PendingActor, state.Allies)
	}
	if row.DeadlineAt.Valid {
		response["deadline_at"] = row.DeadlineAt.Time
	}
	return response
}

func combatSessionSide(side []*simulator.CombatantAuto, pendingActor string, opposingSide []*simulator.CombatantAuto) []gin.H {
	result := make([]gin.H, 0, len(side))
	for _, combatant := range side {
		if combatant == nil {
			continue
		}
		entry := gin.H{"id": combatant.ID, "name": combatant.Name, "kind": combatant.Kind,
			"hp": combatant.HP, "max_hp": combatant.MaxHP, "mana": combatant.Mana,
			"max_mana": combatant.MaxMana, "dead": combatant.IsDead}
		if combatant.ID == pendingActor {
			actions := make([]gin.H, 0, len(combatant.Actions)+2)
			fallbackBasicAttack := simulator.SimAction{ActionType: "BASIC_ATTACK", Name: "Đánh thường", TargetType: "SINGLE"}
			basicAttack := gin.H{"action_type": "BASIC_ATTACK", "name": "Đánh thường",
				"target_type": fallbackBasicAttack.TargetType,
				"target_ids":  simulator.InteractiveActionTargetIDs(combatant, opposingSide, &fallbackBasicAttack)}
			hasBasicAttack := false
			for _, action := range combatant.Actions {
				if action.ActionType == "BASIC_ATTACK" {
					if !hasBasicAttack {
						basicAttack = gin.H{"action_type": action.ActionType, "source_id": action.SourceID,
							"instance_id": action.InstanceID, "name": action.Name, "target_type": action.TargetType,
							"mana_cost": action.ManaCost, "rank": action.Rank, "path": action.Path,
							"damage": action.Damage, "defense": action.Defense, "healing": action.Healing,
							"reason": action.Reason, "target_ids": simulator.InteractiveActionTargetIDs(combatant, opposingSide, &action)}
						hasBasicAttack = true
					}
					continue
				}
				actions = append(actions, gin.H{"action_type": action.ActionType, "source_id": action.SourceID,
					"instance_id": action.InstanceID, "name": action.Name, "target_type": action.TargetType,
					"mana_cost": action.ManaCost, "rank": action.Rank, "path": action.Path,
					"damage": action.Damage, "defense": action.Defense, "healing": action.Healing,
					"reason": action.Reason, "target_ids": simulator.InteractiveActionTargetIDs(combatant, opposingSide, &action)})
			}
			actions = append([]gin.H{basicAttack}, actions...)
			actions = append(actions, gin.H{"action_type": "FLEE", "name": "Bỏ chạy"})
			entry["actions"] = actions
		}
		result = append(result, entry)
	}
	return result
}

func publishCombatSessionEvent(row combatSessionRow, status, pendingActor string) {
	players := []uuid.UUID{row.PlayerID}
	if row.ParticipantID.Valid {
		players = append(players, row.ParticipantID.UUID)
	}
	event := PlayerDeltaEvent{
		Type: "combat.session.updated", Revision: nextPlayerEventRevision(),
		ChangedDomains: []PlayerEventDomain{PlayerDomain},
		Data: PlayerDeltaData{CombatSession: &CombatSessionDelta{
			SessionID: row.ID.String(), Revision: row.Revision, Status: status, PendingActor: pendingActor,
		}},
	}
	for _, playerID := range players {
		stateRevision, err := loadPlayerStateRevision(db.DB, playerID)
		if err != nil {
			continue
		}
		event.StateRevision = stateRevision
		_ = cache.PublishEventQuick("player:"+playerID.String(), event)
	}
}

// StartCombatSession snapshots the prepared loadout and pauses the simulator at
// the first player decision. Existing auto combat remains on /combat/start.
func StartCombatSession(c *gin.Context) {
	var req AutoCombatStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()
	if _, existingErr := loadCombatSession(tx, playerID, nil, true); existingErr == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "An interactive combat session is already active"})
		return
	} else if existingErr != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to inspect combat session"})
		return
	}
	player, err := evaluateManaTxEquipped(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player"})
		return
	}
	if player.ActiveCombat == nil || !combatOwnedByPlayer(player.ActiveCombat, player.ID.String()) {
		c.JSON(http.StatusConflict, gin.H{"error": "No player-controlled encounter is ready"})
		return
	}
	if player.ActiveCombat.CombatOwnerID != "" && player.ActiveCombat.CombatOwnerID != player.ID.String() {
		c.JSON(http.StatusConflict, gin.H{"error": "The attacking player starts the shared PvP session"})
		return
	}
	if requiresRegionCombatPolicy(player.ActiveCombat) {
		if err := checkRegionAction(player.CurrentRegion, "combat"); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
	}
	if player.ActiveCombat.EnemyType != "TOWER" {
		if err := checkRegionEquippedGus(player.CurrentRegion, player.GuInventory); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
	}
	isVirtual := player.ActiveCombat.CombatMode == "virtual"
	if isVirtual {
		req.Loadout.AutoHealHP = false
		req.Loadout.AutoRestoreMana = false
	}
	state, err := buildSimCombatState(tx, &player, &req.Loadout, isVirtual)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare combat"})
		return
	}
	state.CaptureTurnSnapshots = true
	manualActors := map[string]bool{playerID.String(): true}
	var participantID uuid.NullUUID
	if player.ActiveCombat.EnemyType == "PVP" && player.ActiveCombat.CombatMode == "real" && player.ActiveCombat.TargetPlayerID != "" {
		if targetID, parseErr := uuid.Parse(player.ActiveCombat.TargetPlayerID); parseErr == nil && targetID != playerID {
			participantID = uuid.NullUUID{UUID: targetID, Valid: true}
			manualActors[targetID.String()] = true
		}
	}
	tribulationEventID := ""
	if player.ActiveCombat != nil {
		tribulationEventID = player.ActiveCombat.TribulationEventID
	}
	state.Interactive = &simulator.InteractiveCombat{
		ManualActors:       manualActors,
		TribulationEventID: tribulationEventID,
	}
	initialJSON, err := json.Marshal(state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode combat"})
		return
	}
	loadoutJSON, err := json.Marshal(req.Loadout)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode loadout"})
		return
	}
	row := combatSessionRow{PlayerID: playerID, ParticipantID: participantID, InitialState: initialJSON, Loadout: loadoutJSON, Commands: []byte("[]"), Revision: 0, Status: "active"}
	state, logs, _, err := replayCombatSession(row, nil)
	if err != nil || state.Interactive.Error != "" || state.Interactive.PendingActor == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Combat did not reach a player decision"})
		return
	}
	deadline := time.Now().Add(combatDecisionTimeout)
	err = tx.QueryRow(`INSERT INTO combat_sessions (player_id, participant_id, mode, initial_state, loadout, commands, pending_actor, deadline_at)
		VALUES ($1, $2, $3, $4, $5, '[]', $6, $7) RETURNING id`, playerID, participantID, player.ActiveCombat.CombatMode, initialJSON, loadoutJSON, state.Interactive.PendingActor, deadline).Scan(&row.ID)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Could not create combat session"})
		return
	}
	row.DeadlineAt = sql.NullTime{Time: deadline, Valid: true}
	row.PendingActor = state.Interactive.PendingActor
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit combat session"})
		return
	}
	PublishPlayerCombatDelta(player.ID, CombatEventStarted, player.ActiveCombat)
	publishCombatSessionEvent(row, "active", state.Interactive.PendingActor)
	c.JSON(http.StatusCreated, combatSessionResponse(row, state, logs))
}

func GetCombatSession(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()
	row, err := loadCombatSession(tx, playerID, nil, false)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "No active combat session"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load combat session"})
		return
	}
	state, logs, _, err := replayCombatSession(row, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to replay combat session"})
		return
	}
	c.JSON(http.StatusOK, combatSessionResponse(row, state, logs))
}

func GetCombatSessionByID(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()
	row, err := loadCombatSession(tx, playerID, &sessionID, false)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Combat session not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load combat session"})
		return
	}
	state, logs, _, err := replayCombatSession(row, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to replay combat session"})
		return
	}
	response := combatSessionResponse(row, state, logs)
	if row.Status == "complete" {
		response["winner"] = combatSessionWinner(state)
	}
	c.JSON(http.StatusOK, response)
}

func combatSessionWinner(state *simulator.CombatState) string {
	if state == nil {
		return "Phe Địch"
	}
	if state.FledBy != "" {
		for sideIndex, side := range [][]*simulator.CombatantAuto{state.Allies, state.Enemies} {
			for _, actor := range side {
				if actor != nil && actor.ID == state.FledBy {
					if sideIndex == 0 {
						return "Phe Địch"
					}
					return "Phe Ta"
				}
			}
		}
	}
	alliesAlive, enemiesAlive := false, false
	for _, actor := range state.Allies {
		alliesAlive = alliesAlive || actor != nil && actor.HP > 0 && !actor.IsDead
	}
	for _, actor := range state.Enemies {
		enemiesAlive = enemiesAlive || actor != nil && actor.HP > 0 && !actor.IsDead
	}
	if alliesAlive && !enemiesAlive {
		return "Phe Ta"
	}
	return "Phe Địch"
}

func SubmitCombatSessionCommand(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}
	var req combatSessionCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.CommandID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid combat command"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()
	row, err := loadCombatSession(tx, playerID, &sessionID, true)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "No active combat session"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lock combat session"})
		return
	}
	if row.Status != "active" {
		state, logs, _, replayErr := replayCombatSession(row, nil)
		if replayErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to replay completed combat"})
			return
		}
		response := combatSessionResponse(row, state, logs)
		if row.Status == "complete" {
			response["winner"] = combatSessionWinner(state)
		}
		c.JSON(http.StatusConflict, response)
		return
	}
	if req.Command.ActorID != playerID.String() {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the active player can submit this turn"})
		return
	}
	var seen bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM combat_session_commands WHERE session_id=$1 AND command_id=$2)`, row.ID, req.CommandID).Scan(&seen); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check command ID"})
		return
	}
	if seen || req.ExpectedRevision != row.Revision {
		state, logs, _, replayErr := replayCombatSession(row, nil)
		if replayErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to replay combat session"})
			return
		}
		c.JSON(http.StatusConflict, combatSessionResponse(row, state, logs))
		return
	}
	state, logs, commands, err := replayCombatSession(row, &req.Command)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to replay combat command"})
		return
	}
	if state.Interactive.Error != "" || state.Interactive.Consumed != len(commands) {
		current, currentLogs, _, replayErr := replayCombatSession(row, nil)
		if replayErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to replay combat session"})
			return
		}
		response := combatSessionResponse(row, current, currentLogs)
		response["error"] = state.Interactive.Error
		c.JSON(http.StatusUnprocessableEntity, response)
		return
	}
	commandsJSON, err := json.Marshal(commands)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode commands"})
		return
	}
	row.Revision++
	var result *CombatResolution
	var player models.Player
	if state.Interactive.PendingActor != "" {
		deadline := time.Now().Add(combatDecisionTimeout)
		_, err = tx.Exec(`UPDATE combat_sessions SET commands=$1, revision=$2, pending_actor=$3, deadline_at=$4, updated_at=NOW() WHERE id=$5`, commandsJSON, row.Revision, state.Interactive.PendingActor, deadline, row.ID)
		row.DeadlineAt = sql.NullTime{Time: deadline, Valid: true}
	} else {
		var playerLoadout simulator.CombatLoadout
		if err = json.Unmarshal(row.Loadout, &playerLoadout); err == nil {
			var playerErr error
			player, playerErr = evaluateManaTxEquipped(tx, row.PlayerID)
			if playerErr != nil {
				err = playerErr
			} else {
				var initial simulator.CombatState
				if err = json.Unmarshal(row.InitialState, &initial); err == nil {
					initial.Interactive.Commands = commands
					result, err = ResolveCombatFromSession(tx, &player, &playerLoadout, &initial)
				}
			}
		}
		if err == nil {
			_, err = tx.Exec(`UPDATE combat_sessions SET commands=$1, revision=$2, status='complete', pending_actor='', deadline_at=NULL, updated_at=NOW() WHERE id=$3`, commandsJSON, row.Revision, row.ID)
			row.Status = "complete"
			row.DeadlineAt.Valid = false
		}
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save combat command"})
		return
	}
	if _, err := tx.Exec(`INSERT INTO combat_session_commands(session_id, command_id, revision) VALUES($1,$2,$3)`, row.ID, req.CommandID, row.Revision); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record command ID"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit combat command"})
		return
	}
	if row.Status == "complete" {
		publishCombatSessionEvent(row, "complete", "")
		PublishPlayerCombatDelta(row.PlayerID, CombatEventEnded, nil)
		if result != nil && result.AfterCommit != nil {
			result.AfterCommit()
		}
		if result != nil {
			for _, n := range result.Notifications {
				notify.Publish(n)
			}
		}
		setWorldPosFromPlayer(&player)
	} else {
		publishCombatSessionEvent(row, "active", state.Interactive.PendingActor)
	}
	if result != nil {
		response := combatSessionResponse(row, state, result.Logs)
		response["winner"] = result.Winner
		c.JSON(http.StatusOK, response)
		return
	}
	c.JSON(http.StatusOK, combatSessionResponse(row, state, logs))
}
