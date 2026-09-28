// Package effects owns the typed gameplay-effect seam. Definitions are mutable
// authoring DTOs; Compiled is the immutable representation used at runtime.
package effects

import "encoding/json"

type Trigger string

const (
	TriggerAlwaysOn          Trigger = "ALWAYS_ON"
	TriggerOnAction          Trigger = "ON_ACTION"
	TriggerOnHit             Trigger = "ON_HIT"
	TriggerBeforeHit         Trigger = "BEFORE_HIT"
	TriggerOnTurnStart       Trigger = "ON_TURN_START"
	TriggerOnDeath           Trigger = "ON_DEATH"
	TriggerOnTick            Trigger = "ON_TICK"
	TriggerCapability        Trigger = "CAPABILITY"
	TriggerCombatPassive     Trigger = "COMBAT_PASSIVE"
	TriggerPassive           Trigger = "PASSIVE"
	TriggerOnBattleStart     Trigger = "ON_BATTLE_START"
	TriggerBeforeEnemyAttack Trigger = "BEFORE_ENEMY_ATTACK"
	TriggerOnCast            Trigger = "ON_CAST"
	TriggerOnConsume         Trigger = "ON_CONSUME"
	TriggerOnUse             Trigger = "ON_USE"
)

type Selector string

const (
	TargetSelf       Selector = "SELF"
	TargetSingle     Selector = "TARGET"
	TargetRow        Selector = "ROW"
	TargetCol        Selector = "COL"
	TargetAllEnemies Selector = "ALL_ENEMIES"
	TargetAllAllies  Selector = "ALL_ALLIES"
	TargetAll        Selector = "ALL"
)

type Target struct {
	Selector        Selector `json:"selector"`
	BypassFrontline bool     `json:"bypass_frontline,omitempty"`
}
type Condition struct {
	Field    string  `json:"field"`
	Operator string  `json:"operator"`
	Value    float64 `json:"value"`
}
type OperationType string

const (
	OpDamage   OperationType = "DAMAGE"
	OpShield   OperationType = "SHIELD"
	OpHeal     OperationType = "HEAL"
	OpDirectHP OperationType = "DIRECT_HP"
	OpStatMod  OperationType = "STAT_MOD"
	OpStatus   OperationType = "STATUS"
	OpBarrier  OperationType = "BARRIER"
	OpCleanse  OperationType = "CLEANSE"
	OpUnique   OperationType = "UNIQUE"
)

type Operation struct {
	Op          OperationType   `json:"op"`
	SubType     string          `json:"sub_type,omitempty"`
	Target      *Target         `json:"target,omitempty"`
	Value       float64         `json:"value,omitempty"`
	Turns       int             `json:"turns,omitempty"`
	Chance      float64         `json:"chance,omitempty"`
	Path        string          `json:"path,omitempty"`
	StatusID    string          `json:"status_id,omitempty"`
	Stat        string          `json:"stat,omitempty"`
	Unit        string          `json:"unit,omitempty"`
	Resource    string          `json:"resource,omitempty"`
	StackPolicy string          `json:"stack_policy,omitempty"`
	Code        string          `json:"code,omitempty"`
	Params      json.RawMessage `json:"params,omitempty"`
}
type Block struct {
	ID         string      `json:"id,omitempty"`
	Trigger    Trigger     `json:"trigger"`
	Target     Target      `json:"target"`
	Conditions []Condition `json:"conditions,omitempty"`
	Operations []Operation `json:"operations"`
}
