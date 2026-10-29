package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	realtimeeffects "github.com/vippro67/GuWorld/backend/internal/realtime/effects"
)

const (
	combatLoadoutFirstSlot = realtimeeffects.KillerMoveFirstSlot // 9
	combatLoadoutSlots     = realtimeeffects.KillerMoveLastSlot - realtimeeffects.KillerMoveFirstSlot + 1
)

// combatLoadoutEntry is one ordered realtime hotbar entry. Slots 9-12 are
// shared by player-created killer moves and Trận Kích from the equipped
// Immortal Gu House; the array order is the Alt+1..4 order.
type combatLoadoutEntry struct {
	Kind string `json:"kind"` // "move" | "house"
	ID   string `json:"id"`
}

func (entry combatLoadoutEntry) key() string { return entry.Kind + "\x00" + entry.ID }

// combatSkillCandidate is the authoring-facing view of a slot candidate. Slot 0
// means the entry exists but is not currently bound to an Alt key.
type combatSkillCandidate struct {
	Kind      string `json:"kind"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Rank      int    `json:"rank"`
	Path      string `json:"path"`
	Slot      int    `json:"slot"`
	Supported bool   `json:"supported"`
	Reason    string `json:"reason,omitempty"`
}

type combatLoadoutQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type combatSkillComponentRef struct {
	InstanceID string `json:"instance_id"`
	GuID       string `json:"gu_id"`
	Rank       int    `json:"rank"`
}

type combatSkillResolution struct {
	entry     combatLoadoutEntry
	move      RealtimeKillerMove
	candidate combatSkillCandidate
}

func realtimeMoveForCompile(move RealtimeKillerMove) realtimeeffects.KillerMove {
	components := make([]realtimeeffects.KillerMoveComponent, 0, len(move.Components))
	for _, component := range move.Components {
		components = append(components, realtimeeffects.KillerMoveComponent{InstanceID: component.InstanceID, Hunger: component.Hunger})
	}
	return realtimeeffects.KillerMove{
		ID: move.ID, Name: move.Name, Rank: move.Rank, Components: components,
		ExpectedComponents: move.ExpectedComponents, Effects: move.Effects, ManaCost: move.ManaCost,
		SuccessChance: move.SuccessChance, BackfireHPPercent: move.BackfireHPPercent,
		BackfireStatus: move.BackfireStatus, Complexity: move.Complexity, DamageScale: move.DamageScale,
		WindupMS: move.RealtimeWindupMS, CooldownMS: move.RealtimeCooldownMS, HouseStrike: move.HouseStrike,
	}
}

func supportedKillerMove(move realtimeeffects.KillerMove) (bool, string) {
	if len(move.Components) == 0 {
		return false, "Cổ thành phần không khả dụng hoặc đã kiệt sức"
	}
	// Hunger is a runtime resource: a starving component must not block
	// re-ordering the loadout. Admission still rejects the cast until fed.
	probe := move
	probe.Components = make([]realtimeeffects.KillerMoveComponent, 0, len(move.Components))
	for _, component := range move.Components {
		hunger := component.Hunger
		if hunger < 1 {
			hunger = 1
		}
		probe.Components = append(probe.Components, realtimeeffects.KillerMoveComponent{
			InstanceID: component.InstanceID, Hunger: hunger,
		})
	}
	if probe.DamageScale <= 0 {
		probe.DamageScale = 1
	}
	compiled := realtimeeffects.BuildKillerMoveSkills([]realtimeeffects.KillerMove{probe})
	if len(compiled) != 1 {
		return false, "Chiêu chưa có executor realtime"
	}
	return true, ""
}

// resolveCombatSkills loads every equipped killer move plus the strike actions
// of the equipped Tiên Cổ Ốc, binds the stored loadout order into slots 9-12,
// and (when the player has not saved an explicit order yet) backfills the
// remaining slots from the equipped pool. It returns both the UI candidates and
// the ordered projection used by realtime admission.
func resolveCombatSkills(ctx context.Context, q combatLoadoutQueryer, playerID uuid.UUID, loadout []combatLoadoutEntry, configured bool) ([]combatSkillCandidate, []RealtimeKillerMove, error) {
	moves, err := loadCombatKillerMoves(ctx, q, playerID)
	if err != nil {
		return nil, nil, err
	}
	houses, err := loadCombatHouseStrikes(ctx, q, playerID)
	if err != nil {
		return nil, nil, err
	}
	resolutions := make([]combatSkillResolution, 0, len(moves)+len(houses))
	resolutions = append(resolutions, moves...)
	resolutions = append(resolutions, houses...)
	if err := resolveCombatSkillHunger(ctx, q, playerID, resolutions); err != nil {
		return nil, nil, err
	}
	for i := range resolutions {
		ok, reason := supportedKillerMove(realtimeMoveForCompile(resolutions[i].move))
		resolutions[i].candidate.Supported = ok
		if !ok && resolutions[i].candidate.Reason == "" {
			resolutions[i].candidate.Reason = reason
		}
	}

	ordered := orderCombatSkills(loadout, configured, resolutions)
	bind := map[string]int{}
	for index, resolution := range ordered {
		if index >= combatLoadoutSlots {
			break
		}
		bind[resolution.entry.key()] = combatLoadoutFirstSlot + index
	}
	// Move candidates keep their natural (equipped moves first, then house
	// strikes) order; the UI renders only candidates whose slot is bound.
	candidates := make([]combatSkillCandidate, 0, len(resolutions))
	for i := range resolutions {
		candidate := resolutions[i].candidate
		candidate.Slot = bind[resolutions[i].entry.key()]
		candidates = append(candidates, candidate)
	}
	orderedMoves := make([]RealtimeKillerMove, 0, len(ordered))
	for i := range ordered {
		if i >= combatLoadoutSlots {
			break
		}
		move := ordered[i].move
		orderedMoves = append(orderedMoves, move)
	}
	return candidates, orderedMoves, nil
}

func orderCombatSkills(loadout []combatLoadoutEntry, configured bool, resolutions []combatSkillResolution) []combatSkillResolution {
	byKey := make(map[string]combatSkillResolution, len(resolutions))
	for _, resolution := range resolutions {
		byKey[resolution.entry.key()] = resolution
	}
	ordered := make([]combatSkillResolution, 0, combatLoadoutSlots)
	used := map[string]bool{}
	for _, entry := range loadout {
		if len(ordered) >= combatLoadoutSlots {
			break
		}
		resolution, ok := byKey[entry.key()]
		if !ok || used[entry.key()] {
			continue
		}
		used[entry.key()] = true
		ordered = append(ordered, resolution)
	}
	if configured {
		// The player has saved an explicit order (including an empty one):
		// respect it exactly so clearing a slot is stable instead of being
		// immediately backfilled.
		return ordered
	}
	// No stored order (legacy or never configured): keep the old automatic
	// behaviour by filling slots with equipped moves, then house strikes.
	for _, resolution := range resolutions {
		if len(ordered) >= combatLoadoutSlots {
			break
		}
		if used[resolution.entry.key()] {
			continue
		}
		used[resolution.entry.key()] = true
		ordered = append(ordered, resolution)
	}
	return ordered
}

func loadCombatKillerMoves(ctx context.Context, q combatLoadoutQueryer, playerID uuid.UUID) ([]combatSkillResolution, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT id::text,name,main_gu_rank,path,components,compiled_effects,
			COALESCE((combat_profile->>'mana_cost')::double precision,0),
			success_chance,complexity,COALESCE(combat_profile->>'backfire_hp_pct','0'),
			COALESCE(combat_profile->>'backfire_status',''),
			COALESCE((combat_profile->>'realtime_windup_ms')::bigint,0),
			COALESCE((combat_profile->>'realtime_cooldown_ms')::bigint,0),
			creation_path_mastery_level,mastery_xp
		FROM player_killer_moves WHERE player_id=$1 AND is_equipped=true
		ORDER BY updated_at DESC,id
	`, playerID)
	if err != nil {
		return nil, fmt.Errorf("load equipped realtime killer moves: %w", err)
	}
	type pendingComponents struct {
		moveIndex  int
		components []combatSkillComponentRef
	}
	resolutions := make([]combatSkillResolution, 0, 4)
	pending := make([]pendingComponents, 0, 4)
	for rows.Next() {
		var move RealtimeKillerMove
		var rawComponents, rawEffects []byte
		if err := rows.Scan(&move.ID, &move.Name, &move.Rank, &move.Path, &rawComponents, &rawEffects,
			&move.ManaCost, &move.SuccessChance, &move.Complexity, &move.BackfireHPPercent, &move.BackfireStatus,
			&move.RealtimeWindupMS, &move.RealtimeCooldownMS, &move.CreationMasteryLevel, &move.MasteryXP); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("scan equipped realtime killer move: %w", err)
		}
		var components []combatSkillComponentRef
		if err := json.Unmarshal(rawComponents, &components); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("decode realtime killer move components: %w", err)
		}
		move.ExpectedComponents = len(components)
		if move.RealtimeWindupMS == 0 || move.RealtimeCooldownMS == 0 {
			move.RealtimeWindupMS, move.RealtimeCooldownMS = models.KillerMoveRealtimeTiming(move.Rank, move.Complexity)
		}
		if err := json.Unmarshal(rawEffects, &move.Effects); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("decode realtime killer move effects: %w", err)
		}
		pending = append(pending, pendingComponents{moveIndex: len(resolutions), components: components})
		resolutions = append(resolutions, combatSkillResolution{
			entry: combatLoadoutEntry{Kind: "move", ID: move.ID},
			move:  move,
			candidate: combatSkillCandidate{
				Kind: "move", ID: move.ID, Name: move.Name, Rank: move.Rank, Path: move.Path,
			},
		})
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("read equipped realtime killer moves: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close equipped realtime killer moves: %w", err)
	}
	// database/sql transactions use one connection. Resolve component rows only
	// after closing the loadout cursor so the follow-up queries cannot wait on it.
	for _, entry := range pending {
		move := &resolutions[entry.moveIndex].move
		for _, component := range entry.components {
			var hunger float64
			var state string
			componentErr := q.QueryRowContext(ctx, `
				SELECT hunger,state FROM player_gu WHERE id=$1 AND player_id=$2 AND gu_id=$3 AND gu_rank=$4
			`, component.InstanceID, playerID, component.GuID, component.Rank).Scan(&hunger, &state)
			if errors.Is(componentErr, sql.ErrNoRows) || strings.EqualFold(state, "DEAD") {
				move.Components = nil
				break
			}
			if componentErr != nil {
				return nil, fmt.Errorf("load realtime killer move component %s: %w", component.InstanceID, componentErr)
			}
			move.Components = append(move.Components, RealtimeKillerMoveComponent{
				InstanceID: component.InstanceID, GuID: component.GuID, Rank: component.Rank, Hunger: hunger,
			})
		}
	}
	return resolutions, nil
}

// loadCombatHouseStrikes projects the equipped Tiên Cổ Ốc Trận Kích actions into
// the same 9-12 slot pool. House actions are damage strikes authored by the
// house compiler, so they are synthesized into one ON_ACTION damage block that
// the strict realtime killer-move compiler can consume unchanged.
func loadCombatHouseStrikes(ctx context.Context, q combatLoadoutQueryer, playerID uuid.UUID) ([]combatSkillResolution, error) {
	var rawProfile []byte
	err := q.QueryRowContext(ctx, `
		SELECT house_profile FROM player_gu_houses WHERE player_id=$1 AND is_equipped=true LIMIT 1
	`, playerID).Scan(&rawProfile)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load equipped gu house: %w", err)
	}
	var profile models.GuHouseCombatProfile
	if err := json.Unmarshal(rawProfile, &profile); err != nil {
		return nil, fmt.Errorf("decode equipped gu house profile: %w", err)
	}
	resolutions := make([]combatSkillResolution, 0, len(profile.Actions))
	for _, action := range profile.Actions {
		if action.ID == "" {
			continue
		}
		move := RealtimeKillerMove{
			ID: action.ID, Name: action.Name, Rank: action.Rank, Path: action.MainPath,
			ExpectedComponents: 1, ManaCost: action.ManaCost, SuccessChance: action.SuccessChance,
			BackfireHPPercent: action.BackfireHPPercent, BackfireStatus: action.BackfireStatus,
			Complexity: 1, RealtimeWindupMS: action.RealtimeWindupMS, RealtimeCooldownMS: action.RealtimeCooldownMS,
			HouseStrike: true,
		}
		if move.RealtimeWindupMS == 0 || move.RealtimeCooldownMS == 0 {
			move.RealtimeWindupMS, move.RealtimeCooldownMS = models.KillerMoveRealtimeTiming(move.Rank, move.Complexity)
		}
		resolution := combatSkillResolution{
			entry: combatLoadoutEntry{Kind: "house", ID: action.ID},
			move:  move,
			candidate: combatSkillCandidate{
				Kind: "house", ID: action.ID, Name: action.Name, Rank: action.Rank, Path: action.MainPath,
			},
		}
		damage := action.BaseDamage * action.DamageMult
		if !strings.EqualFold(action.TargetType, "SINGLE") || damage <= 0 || action.SuccessChance <= 0 {
			resolution.candidate.Reason = "Trận Kích chưa có dạng mục tiêu đơn hỗ trợ realtime"
			resolutions = append(resolutions, resolution)
			continue
		}
		move.Effects = []gameplayeffects.Block{{
			Trigger: gameplayeffects.TriggerOnAction,
			Target:  gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
			Operations: []gameplayeffects.Operation{{
				Op: gameplayeffects.OpDamage, Value: damage, Path: action.MainPath,
			}},
		}}
		resolution.move = move
		resolutions = append(resolutions, resolution)
	}
	return resolutions, nil
}

func resolveCombatSkillHunger(ctx context.Context, q combatLoadoutQueryer, playerID uuid.UUID, resolutions []combatSkillResolution) error {
	for i := range resolutions {
		move := &resolutions[i].move
		if resolutions[i].entry.Kind != "house" {
			continue
		}
		var hunger float64
		var state string
		err := q.QueryRowContext(ctx, `SELECT hunger,state FROM player_gu WHERE id=$1 AND player_id=$2`,
			resolutions[i].entry.ID, playerID).Scan(&hunger, &state)
		if errors.Is(err, sql.ErrNoRows) || strings.EqualFold(state, "DEAD") {
			move.Components = nil
			continue
		}
		if err != nil {
			return fmt.Errorf("load gu house strike component: %w", err)
		}
		move.Components = []RealtimeKillerMoveComponent{{InstanceID: resolutions[i].entry.ID, Hunger: hunger}}
	}
	return nil
}

func loadPlayerCombatLoadout(ctx context.Context, q combatLoadoutQueryer, playerID uuid.UUID) ([]combatLoadoutEntry, bool, error) {
	var raw []byte
	var configured bool
	if err := q.QueryRowContext(ctx, `SELECT COALESCE(realtime_skill_loadout, '[]'::jsonb), realtime_skill_loadout_configured FROM players WHERE id=$1`, playerID).Scan(&raw, &configured); err != nil {
		return nil, false, err
	}
	var loadout []combatLoadoutEntry
	if err := json.Unmarshal(raw, &loadout); err != nil {
		return nil, false, fmt.Errorf("decode realtime skill loadout: %w", err)
	}
	out := make([]combatLoadoutEntry, 0, len(loadout))
	for _, entry := range loadout {
		if entry.Kind != "move" && entry.Kind != "house" {
			continue
		}
		if entry.ID == "" {
			continue
		}
		out = append(out, entry)
		if len(out) >= combatLoadoutSlots {
			break
		}
	}
	return out, configured, nil
}

func combatLoadoutResponse(candidates []combatSkillCandidate, maxSlots int) gin.H {
	slots := make([]combatSkillCandidate, 0, maxSlots)
	available := make([]combatSkillCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		available = append(available, candidate)
		if candidate.Slot > 0 {
			slots = append(slots, candidate)
		}
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i].Slot < slots[j].Slot })
	return gin.H{
		"slots":      slots,
		"available":  available,
		"first_slot": combatLoadoutFirstSlot,
		"max_slots":  maxSlots,
	}
}

// GetCombatSkillLoadout returns the player's skill hotbar for slots 9-12 plus
// every eligible equipped killer move / house strike.
func GetCombatSkillLoadout(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	ctx := c.Request.Context()
	loadout, configured, err := loadPlayerCombatLoadout(ctx, db.DB, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải hotbar chiêu thức"})
		return
	}
	candidates, _, err := resolveCombatSkills(ctx, db.DB, playerID, loadout, configured)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải hotbar chiêu thức"})
		return
	}
	c.JSON(http.StatusOK, combatLoadoutResponse(candidates, combatLoadoutSlots))
}

// UpdateCombatSkillLoadout stores the ordered slot bindings for slots 9-12.
// Moves and house strikes share the same pool; unknown, unequipped or
// unsupported entries are rejected so stale clients cannot persist dead slots.
func UpdateCombatSkillLoadout(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req struct {
		Entries []combatLoadoutEntry `json:"entries"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu hotbar chiêu thức không hợp lệ"})
		return
	}
	if len(req.Entries) > combatLoadoutSlots {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Hotbar chỉ có %d ô chiêu thức", combatLoadoutSlots)})
		return
	}
	ctx := c.Request.Context()
	candidates, _, err := resolveCombatSkills(ctx, db.DB, playerID, nil, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác thực hotbar chiêu thức"})
		return
	}
	eligible := make(map[string]combatSkillCandidate, len(candidates))
	for _, candidate := range candidates {
		eligible[candidate.Kind+"\x00"+candidate.ID] = candidate
	}
	seen := map[string]bool{}
	clean := make([]combatLoadoutEntry, 0, len(req.Entries))
	for _, entry := range req.Entries {
		if entry.ID == "" || (entry.Kind != "move" && entry.Kind != "house") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Hotbar chứa chiêu thức không hợp lệ"})
			return
		}
		key := entry.Kind + "\x00" + entry.ID
		if seen[key] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể ghim trùng một chiêu thức hai lần"})
			return
		}
		candidate, exists := eligible[key]
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Chiêu thức không còn trang bị hoặc không tồn tại"})
			return
		}
		if !candidate.Supported {
			message := candidate.Reason
			if message == "" {
				message = "Chiêu thức chưa hỗ trợ realtime"
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": message})
			return
		}
		seen[key] = true
		clean = append(clean, combatLoadoutEntry{Kind: entry.Kind, ID: entry.ID})
	}
	raw, err := json.Marshal(clean)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu hotbar chiêu thức không hợp lệ"})
		return
	}
	// Resolve first so a read failure cannot report failure after the write
	// already persisted the new order.
	resolved, _, err := resolveCombatSkills(ctx, db.DB, playerID, clean, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải hotbar chiêu thức"})
		return
	}
	if _, err := db.DB.ExecContext(ctx, `UPDATE players SET realtime_skill_loadout=$1::jsonb, realtime_skill_loadout_configured=true WHERE id=$2`, raw, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu hotbar chiêu thức"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Đã lưu hotbar chiêu thức",
		"loadout": combatLoadoutResponse(resolved, combatLoadoutSlots),
	})
}
