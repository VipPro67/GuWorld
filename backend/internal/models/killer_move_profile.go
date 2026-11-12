package models

import (
	"encoding/json"
	"fmt"

	"github.com/vippro67/GuWorld/backend/internal/gameplay/composition"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

// KillerMoveCombatProfile defines the unified tactical combat profile of a killer move.
// It is deterministically compiled from a Core Gu (hạch tâm) and auxiliary Gu (phụ trợ).
type KillerMoveCombatProfile struct {
	ProfileVersion       int                         `json:"profile_version,omitempty"`
	EffectLabel          string                      `json:"effect_label,omitempty"`
	EffectKey            string                      `json:"effect_key,omitempty"`
	EffectContext        string                      `json:"effect_context,omitempty"`
	EffectExecutor       string                      `json:"effect_executor,omitempty"`
	EffectGuID           string                      `json:"effect_gu_id,omitempty"`
	EffectOperation      *gameplayeffects.Operation  `json:"effect_operation,omitempty"`
	EffectTarget         *gameplayeffects.Target     `json:"effect_target,omitempty"`
	EffectConditions     []gameplayeffects.Condition `json:"effect_conditions,omitempty"`
	CompositionGraph     *composition.Graph          `json:"composition_graph,omitempty"`
	EffectBlocks         gameplayeffects.BlockSet    `json:"effect_blocks,omitempty"`
	ComponentGuInstances []string                    `json:"component_gu_instances,omitempty"`
	ID                   string                      `json:"id"`
	Name                 string                      `json:"name"`
	Rank                 int                         `json:"rank"`
	MainPath             string                      `json:"main_path"`
	AllPaths             []string                    `json:"all_paths"`
	TargetType           string                      `json:"target_type"` // SINGLE | ROW | COL | CROSS | ALL_ENEMIES
	Bypass               bool                        `json:"bypass"`      // Bypasses frontline / shields
	CooldownTurns        int                         `json:"cooldown_turns,omitempty"`
	WarmupTurns          int                         `json:"warmup_turns,omitempty"`
	Priority             int                         `json:"priority"`
	ManaCost             float64                     `json:"mana_cost"`
	HungerCost           float64                     `json:"hunger_cost"`
	BaseDamage           float64                     `json:"base_damage"`
	DamageMult           float64                     `json:"damage_mult"`
	BaseShield           float64                     `json:"base_shield"`
	BaseHealing          float64                     `json:"base_healing"`
	LifeStealPct         float64                     `json:"lifesteal_pct"`
	CritBonus            float64                     `json:"crit_bonus"`
	StatusOps            []statusfx.StatusOp         `json:"status_ops,omitempty"`
	SuccessChance        float64                     `json:"success_chance"`
	BackfireHPPercent    float64                     `json:"backfire_hp_pct"`
	BackfireStatus       string                      `json:"backfire_status"`
}

func (p *KillerMoveCombatProfile) UnmarshalJSON(raw []byte) error {
	type profileAlias KillerMoveCombatProfile
	var decoded profileAlias
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return err
	}
	*p = KillerMoveCombatProfile(decoded)
	p.upgradeLegacyEffectBlockIDs()
	return nil
}

// upgradeLegacyEffectBlockIDs migrates old composed-action snapshots whose
// block IDs were effect keys. Block v2 requires canonical IDs; the graph keeps
// the effect-key provenance, so update only its block-ID projection.
func (p *KillerMoveCombatProfile) upgradeLegacyEffectBlockIDs() {
	if p == nil || p.CompositionGraph == nil || len(p.EffectBlocks) == 0 || len(p.CompositionGraph.Nodes) == 0 {
		return
	}
	blockIndexByID := make(map[string]int, len(p.EffectBlocks))
	for index, block := range p.EffectBlocks {
		blockIndexByID[block.ID] = index
	}
	usedBlocks := make(map[int]bool, len(p.EffectBlocks))
	for nodeIndex := range p.CompositionGraph.Nodes {
		node := &p.CompositionGraph.Nodes[nodeIndex]
		blockIndex, found := blockIndexByID[node.BlockID]
		if !found && node.BlockID == "" {
			blockIndex, found = blockIndexByID[node.EffectKey]
		}
		if !found || usedBlocks[blockIndex] {
			continue
		}
		blockID := fmt.Sprintf("EFFECT_%02d", nodeIndex+1)
		p.EffectBlocks[blockIndex].ID = blockID
		node.BlockID = blockID
		usedBlocks[blockIndex] = true
	}
}

// GuHouseCombatProfile represents an Immortal Gu House (Tiên Cổ Ốc) in combat.
// It acts as a materialized immortal killer move entity / vehicle stance.
type GuHouseCombatProfile struct {
	ProfileVersion     int                       `json:"profile_version,omitempty"`
	ProductProfile     string                    `json:"product_profile,omitempty"`
	Context            string                    `json:"context,omitempty"`
	Executor           string                    `json:"executor,omitempty"`
	PreviewFingerprint string                    `json:"preview_fingerprint,omitempty"`
	ID                 string                    `json:"id"`
	Name               string                    `json:"name"`
	Rank               int                       `json:"rank"`
	CoreGuIDs          []string                  `json:"core_gu_ids"`
	AuxGuCount         int                       `json:"aux_gu_count"`
	MaxHouseHP         float64                   `json:"max_house_hp"`
	CurrentHouseHP     float64                   `json:"current_house_hp"`
	HouseDefense       float64                   `json:"house_defense"`
	DamageReduction    float64                   `json:"damage_reduction"` // % damage reduction (0..1)
	ImmuneDebuffs      bool                      `json:"immune_debuffs"`
	ZeroBackfire       bool                      `json:"zero_backfire"`
	Actions            []KillerMoveCombatProfile `json:"actions"`
}
