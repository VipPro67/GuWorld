package models

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type ActorKind string

const (
	ActorKindMonster        ActorKind = "MONSTER"
	ActorKindCompanionBeast ActorKind = "COMPANION_BEAST"
	ActorKindVariantNPC     ActorKind = "VARIANT_NPC"
	ActorKindWorldNPC       ActorKind = "WORLD_NPC"
	ActorKindSectNPC        ActorKind = "SECT_NPC"
	ActorKindConstruct      ActorKind = "CONSTRUCT"
	ActorKindBoss           ActorKind = "BOSS"
)

type TemplateSource string

const (
	TemplateSourceMonsterTemplate TemplateSource = "MONSTER_TEMPLATE"
	TemplateSourceVariantRace     TemplateSource = "VARIANT_RACE"
	TemplateSourceNPCTemplate     TemplateSource = "NPC_TEMPLATE"
	TemplateSourceCustom          TemplateSource = "CUSTOM"
)

type ActorLifecycleState string

const (
	ActorLifecycleActive    ActorLifecycleState = "ACTIVE"
	ActorLifecycleDead      ActorLifecycleState = "DEAD"
	ActorLifecycleDismissed ActorLifecycleState = "DISMISSED"
)

type ActorExecutionState string

const (
	ActorStateStored     ActorExecutionState = "STORED"
	ActorStateDeploying  ActorExecutionState = "DEPLOYING"
	ActorStateTraining   ActorExecutionState = "TRAINING"
	ActorStateWorking    ActorExecutionState = "WORKING"
	ActorStateRecovering ActorExecutionState = "RECOVERING"
	ActorStateDead       ActorExecutionState = "DEAD"
	ActorStateDismissed  ActorExecutionState = "DISMISSED"
	ActorStateWorldIdle  ActorExecutionState = "WORLD_IDLE"
	ActorStateMoving     ActorExecutionState = "MOVING"
	ActorStateFollowing  ActorExecutionState = "FOLLOWING"
	ActorStateGuarding   ActorExecutionState = "GUARDING"
	ActorStatePatrolling ActorExecutionState = "PATROLLING"
	ActorStateGathering  ActorExecutionState = "GATHERING"
	ActorStateCombat     ActorExecutionState = "COMBAT"
	ActorStateReturning  ActorExecutionState = "RETURNING"
	ActorStateInjured    ActorExecutionState = "INJURED"
)

type ActorCapability string

const (
	CapabilityMove             ActorCapability = "MOVE"
	CapabilityFollow           ActorCapability = "FOLLOW"
	CapabilityPatrol           ActorCapability = "PATROL"
	CapabilityGuard            ActorCapability = "GUARD"
	CapabilityCombat           ActorCapability = "COMBAT"
	CapabilityProduce          ActorCapability = "PRODUCE"
	CapabilityRefine           ActorCapability = "REFINE"
	CapabilityGather           ActorCapability = "GATHER"
	CapabilityTrain            ActorCapability = "TRAIN"
	CapabilityFeedable         ActorCapability = "FEEDABLE"
	CapabilityCapturable       ActorCapability = "CAPTURABLE"
	CapabilityStackable        ActorCapability = "STACKABLE"
	CapabilityEquipGu          ActorCapability = "EQUIP_GU"
	CapabilityUseSkill         ActorCapability = "USE_SKILL"
	CapabilityTrade            ActorCapability = "TRADE"
	CapabilityQuestInteraction ActorCapability = "QUEST_INTERACTION"
	CapabilityMortal           ActorCapability = "MORTAL"
	CapabilityRevivable        ActorCapability = "REVIVABLE"
)

type ActorCommand string

const (
	ActorCommandFollow     ActorCommand = "FOLLOW"
	ActorCommandStay       ActorCommand = "STAY"
	ActorCommandMoveTo     ActorCommand = "MOVE_TO"
	ActorCommandGuardArea  ActorCommand = "GUARD_AREA"
	ActorCommandPatrol     ActorCommand = "PATROL"
	ActorCommandAttack     ActorCommand = "ATTACK"
	ActorCommandAssist     ActorCommand = "ASSIST"
	ActorCommandGather     ActorCommand = "GATHER"
	ActorCommandReturnHome ActorCommand = "RETURN_HOME"
	ActorCommandRetreat    ActorCommand = "RETREAT"
)

// ActorCapabilities is deliberately a boolean map: actor_kind supplies only
// defaults, while this persisted policy is authoritative for execution.
type ActorCapabilities map[ActorCapability]bool

func (c *ActorCapabilities) Scan(value interface{}) error {
	if value == nil {
		*c = ActorCapabilities{}
		return nil
	}
	data, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("scan actor capabilities: unsupported %T", value)
	}
	if len(data) == 0 {
		*c = ActorCapabilities{}
		return nil
	}
	return json.Unmarshal(data, c)
}

func (c ActorCapabilities) Value() (driver.Value, error) {
	if c == nil {
		return []byte(`{}`), nil
	}
	return json.Marshal(c)
}

type NonPlayerActor struct {
	ID              string                 `json:"id"`
	ActorKind       ActorKind              `json:"actor_kind"`
	OwnerPlayerID   *string                `json:"owner_player_id,omitempty"`
	FactionID       *string                `json:"faction_id,omitempty"`
	ApertureID      *string                `json:"aperture_id,omitempty"`
	TemplateSource  TemplateSource         `json:"template_source"`
	TemplateID      string                 `json:"template_id"`
	Species         string                 `json:"species"`
	Race            string                 `json:"race"`
	Name            string                 `json:"name"`
	Rank            int                    `json:"rank"`
	SubRank         int                    `json:"sub_rank"`
	Experience      int64                  `json:"experience"`
	Talent          float64                `json:"talent"`
	Loyalty         float64                `json:"loyalty"`
	HP              float64                `json:"hp"`
	MaxHP           float64                `json:"max_hp"`
	BaseAttack      float64                `json:"base_attack"`
	BaseDefense     float64                `json:"base_defense"`
	Speed           float64                `json:"speed"`
	Count           int                    `json:"count"`
	Hunger          float64                `json:"hunger"`
	MaxHunger       float64                `json:"max_hunger"`
	UpkeepRate      float64                `json:"upkeep_rate"`
	HungerUpdatedAt time.Time              `json:"hunger_updated_at"`
	LifecycleState  ActorLifecycleState    `json:"lifecycle_state"`
	ExecutionState  ActorExecutionState    `json:"execution_state"`
	BehaviorProfile string                 `json:"behavior_profile"`
	AIProfile       string                 `json:"ai_profile"`
	Capabilities    ActorCapabilities      `json:"capabilities"`
	Traits          map[string]interface{} `json:"traits"`
	Progression     map[string]interface{} `json:"progression"`
	LegacySource    *string                `json:"legacy_source,omitempty"`
	LegacyID        *string                `json:"legacy_id,omitempty"`
	Version         int64                  `json:"version"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

const actorSelectColumns = `
	id, actor_kind, owner_player_id, faction_id, aperture_id,
	template_source, template_id, species, race, name,
	rank, sub_rank, experience, talent, loyalty,
	hp, max_hp, base_attack, base_defense, speed,
	count, hunger, max_hunger, upkeep_rate, hunger_updated_at,
	lifecycle_state, execution_state, behavior_profile, ai_profile,
	capabilities, traits, progression, legacy_source, legacy_id,
	version, created_at, updated_at`

type actorScanner interface {
	Scan(dest ...interface{}) error
}

func scanNonPlayerActor(scanner actorScanner) (NonPlayerActor, error) {
	var actor NonPlayerActor
	var ownerID, factionID, apertureID, legacySource, legacyID sql.NullString
	var traits, progression []byte
	err := scanner.Scan(
		&actor.ID, &actor.ActorKind, &ownerID, &factionID, &apertureID,
		&actor.TemplateSource, &actor.TemplateID, &actor.Species, &actor.Race, &actor.Name,
		&actor.Rank, &actor.SubRank, &actor.Experience, &actor.Talent, &actor.Loyalty,
		&actor.HP, &actor.MaxHP, &actor.BaseAttack, &actor.BaseDefense, &actor.Speed,
		&actor.Count, &actor.Hunger, &actor.MaxHunger, &actor.UpkeepRate, &actor.HungerUpdatedAt,
		&actor.LifecycleState, &actor.ExecutionState, &actor.BehaviorProfile, &actor.AIProfile,
		&actor.Capabilities, &traits, &progression, &legacySource, &legacyID,
		&actor.Version, &actor.CreatedAt, &actor.UpdatedAt,
	)
	if err != nil {
		return NonPlayerActor{}, err
	}
	assignNullableString := func(value sql.NullString) *string {
		if !value.Valid {
			return nil
		}
		v := value.String
		return &v
	}
	actor.OwnerPlayerID = assignNullableString(ownerID)
	actor.FactionID = assignNullableString(factionID)
	actor.ApertureID = assignNullableString(apertureID)
	actor.LegacySource = assignNullableString(legacySource)
	actor.LegacyID = assignNullableString(legacyID)
	actor.Traits = map[string]interface{}{}
	actor.Progression = map[string]interface{}{}
	if err := json.Unmarshal(traits, &actor.Traits); err != nil {
		return NonPlayerActor{}, fmt.Errorf("decode actor traits: %w", err)
	}
	if err := json.Unmarshal(progression, &actor.Progression); err != nil {
		return NonPlayerActor{}, fmt.Errorf("decode actor progression: %w", err)
	}
	return actor, nil
}

func LoadOwnedNonPlayerActors(ctx context.Context, db *sql.DB, playerID string) ([]NonPlayerActor, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+actorSelectColumns+`
		FROM non_player_actors WHERE owner_player_id = $1
		ORDER BY actor_kind, created_at, id`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	actors := make([]NonPlayerActor, 0)
	for rows.Next() {
		actor, scanErr := scanNonPlayerActor(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		actors = append(actors, actor)
	}
	return actors, rows.Err()
}

func LoadOwnedNonPlayerActor(ctx context.Context, db *sql.DB, actorID, playerID string) (NonPlayerActor, error) {
	return scanNonPlayerActor(db.QueryRowContext(ctx, `SELECT `+actorSelectColumns+`
		FROM non_player_actors WHERE id = $1 AND owner_player_id = $2`, actorID, playerID))
}

var actorTransitions = map[ActorExecutionState]map[ActorExecutionState]bool{
	ActorStateStored:    {ActorStateDeploying: true, ActorStateTraining: true, ActorStateWorking: true, ActorStateRecovering: true, ActorStateDead: true, ActorStateDismissed: true},
	ActorStateDeploying: {ActorStateWorldIdle: true},
	ActorStateWorldIdle: {ActorStateMoving: true, ActorStateFollowing: true, ActorStateGuarding: true, ActorStatePatrolling: true, ActorStateGathering: true, ActorStateCombat: true, ActorStateReturning: true, ActorStateInjured: true},
	ActorStateCombat:    {ActorStateWorldIdle: true, ActorStateReturning: true, ActorStateInjured: true, ActorStateDead: true},
}

func CanTransition(actor NonPlayerActor, from, to ActorExecutionState) bool {
	if actor.ExecutionState != from || actor.LifecycleState != ActorLifecycleActive || from == to {
		return false
	}
	return actorTransitions[from][to]
}

func CanExecute(actor NonPlayerActor, capability ActorCapability) bool {
	return actor.LifecycleState == ActorLifecycleActive && actor.Capabilities[capability]
}

var commandCapabilities = map[ActorCommand]ActorCapability{
	ActorCommandFollow: CapabilityFollow, ActorCommandStay: CapabilityMove,
	ActorCommandMoveTo: CapabilityMove, ActorCommandGuardArea: CapabilityGuard,
	ActorCommandPatrol: CapabilityPatrol, ActorCommandAttack: CapabilityCombat,
	ActorCommandAssist: CapabilityCombat, ActorCommandGather: CapabilityGather,
	ActorCommandReturnHome: CapabilityMove, ActorCommandRetreat: CapabilityMove,
}

func CanReceiveCommand(actor NonPlayerActor, playerID string, command ActorCommand) error {
	if actor.OwnerPlayerID == nil || *actor.OwnerPlayerID != playerID {
		return errors.New("player does not control actor")
	}
	capability, ok := commandCapabilities[command]
	if !ok {
		return errors.New("unknown actor command")
	}
	if !CanExecute(actor, capability) {
		return fmt.Errorf("actor cannot execute capability %s", capability)
	}
	return nil
}
