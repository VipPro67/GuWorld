package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// Combat Loadout module: the single seam over the two realtime combat loadout
// storage columns (players.equipped_gus for hotbar slots 1-6 and
// players.realtime_skill_loadout for slots 9-12). REST handlers, the join
// state projection and gateway admission all consume the compiled snapshot so
// slot ordering and validation rules live in exactly one place.

// CombatLoadoutSnapshot is the compiled realtime combat projection: the
// ordered 1-6 active Gu hotbar and the ordered 9-12 killer move / house
// strike pool, already validated against the supported-compiler rules.
type CombatLoadoutSnapshot struct {
	EquippedGu  []RealtimeEquippedGu
	KillerMoves []RealtimeKillerMove
}

// LoadCombatLoadout reads the hotbar composition and the stored 9-12 pins,
// then compiles both into one snapshot. `hotbarOrder` is the
// players.equipped_gus JSONB read by the caller's character query (equipped
// fallback order), `hotbarSetting` is the explicit players.realtime_hotbar
// assignment (slot pins, "" = empty slot). The skill loadout column is
// interpreted inside the module.
func LoadCombatLoadout(ctx context.Context, q combatLoadoutQueryer, playerID uuid.UUID, hotbarOrder models.EquippedGusList, hotbarSetting models.RealtimeHotbarList) (CombatLoadoutSnapshot, error) {
	equipped, err := loadRealtimeEquipped(ctx, q, playerID, hotbarOrder, hotbarSetting)
	if err != nil {
		return CombatLoadoutSnapshot{}, err
	}
	loadout, configured, err := loadPlayerCombatLoadout(ctx, q, playerID)
	if err != nil {
		return CombatLoadoutSnapshot{}, err
	}
	_, killerMoves, err := resolveCombatSkills(ctx, q, playerID, loadout, configured)
	if err != nil {
		return CombatLoadoutSnapshot{}, err
	}
	return CombatLoadoutSnapshot{EquippedGu: equipped, KillerMoves: killerMoves}, nil
}

// validateCombatLoadoutEntries checks the requested 9-12 pin order against the
// freshly compiled eligible pool so stale clients cannot persist dead slots.
// The returned entries are the sanitized storage payload.
func validateCombatLoadoutEntries(entries []combatLoadoutEntry, candidates []combatSkillCandidate) ([]combatLoadoutEntry, error) {
	eligible := make(map[string]combatSkillCandidate, len(candidates))
	for _, candidate := range candidates {
		eligible[candidate.Kind+"\x00"+candidate.ID] = candidate
	}
	seen := map[string]bool{}
	clean := make([]combatLoadoutEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.ID == "" || (entry.Kind != "move" && entry.Kind != "house") {
			return nil, &loadoutEntryError{kind: errLoadoutInvalidEntry}
		}
		if seen[entry.Kind+"\x00"+entry.ID] {
			return nil, &loadoutEntryError{kind: errLoadoutDuplicateEntry}
		}
		candidate, exists := eligible[entry.Kind+"\x00"+entry.ID]
		if !exists {
			return nil, &loadoutEntryError{kind: errLoadoutUnknownEntry}
		}
		if !candidate.Supported {
			return nil, &loadoutEntryError{kind: errLoadoutUnsupportedEntry, Detail: candidate.Reason}
		}
		seen[entry.Kind+"\x00"+entry.ID] = true
		clean = append(clean, combatLoadoutEntry{Kind: entry.Kind, ID: entry.ID})
	}
	return clean, nil
}

var (
	errLoadoutInvalidEntry     = errors.New("skill loadout entry is invalid")
	errLoadoutDuplicateEntry   = errors.New("skill loadout entry is pinned twice")
	errLoadoutUnknownEntry     = errors.New("skill loadout entry is not equipped or missing")
	errLoadoutUnsupportedEntry = errors.New("skill loadout entry is not realtime supported")
)

// loadoutEntryError carries the player-facing compiler reason for a rejected
// slot so HTTP handlers map it directly onto their response messages.
type loadoutEntryError struct {
	kind   error
	Detail string
}

func (e *loadoutEntryError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s: %s", e.kind, e.Detail)
	}
	return e.kind.Error()
}

func (e *loadoutEntryError) Unwrap() error { return e.kind }

// EncodeCombatLoadout serializes the storage payload for the players table.
func EncodeCombatLoadout(entries []combatLoadoutEntry) (string, error) {
	raw, err := json.Marshal(entries)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
