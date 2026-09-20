// Package gamefx resolves gameplay modifiers granted by a player's character
// attributes (race, gender, constitution, statuses). It is shared by the
// handler and service layers to avoid an import cycle.
package gamefx

import (
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

// Effects aggregates the gameplay modifiers granted by a player's
// race (chủng tộc), constitution (thể chất) and active statuses (trạng thái).
type Effects struct {
	// StatMods maps a passive stat key ("attack", "defense", "speed", ...) to a
	// percentage modifier applied on the stat total. E.g. 0.10 => +10%.
	StatMods map[string]float64 `json:"stat_mods"`
	// RefineBonus is an additive success-rate bonus for luyện cổ (refining),
	// in the 0..1 range.
	RefineBonus float64 `json:"refine"`
	// DaoNganBonus maps a dao path to an additive multiplier for that path's
	// đạo ngân contribution in combat. E.g. {"ice": 0.2} => ×1.2 for ice.
	DaoNganBonus map[string]float64 `json:"dao_mark"`
	// PathDamageBonus maps a dao path to a percentage damage multiplier for
	// abilities of that path. E.g. {"dream": 0.15} => +15% dream-path damage.
	PathDamageBonus map[string]float64 `json:"path_damage"`
	// PathResistanceBonus maps a dao path to a percentage reduction of damage
	// taken from that path. E.g. {"fire": 0.10} => -10% fire damage taken.
	PathResistanceBonus map[string]float64 `json:"path_resistance"`
	// LuckBonus is a flat increase of the luck stat.
	LuckBonus float64 `json:"luck"`
	// MaxManaBonus is a percentage increase of the mana pool capacity.
	MaxManaBonus float64 `json:"max_mana"`
	// ManaRegenBonus is a percentage increase of mana regeneration rate.
	ManaRegenBonus float64 `json:"mana_regen"`
	// CultivationEfficiencyBonus is a percentage increase of cultivation
	// gains (phá vách wall progress, ...).
	CultivationEfficiencyBonus float64 `json:"cultivation_efficiency"`
	// DaoMarkGainBonus is a percentage increase of Đạo Ngân earned.
	DaoMarkGainBonus float64 `json:"dao_mark_gain"`
	// HealingReceivedBonus is a percentage increase of healing the player
	// receives (and HP regeneration).
	HealingReceivedBonus float64 `json:"healing_received"`
	// HPRegenBonus is a flat HP-per-hour bonus added to passive out-of-combat
	// HP regeneration (before the HealingReceivedBonus multiplier).
	HPRegenBonus float64 `json:"hp_regen"`
	// MentalResistanceBonus is a percentage increase of resistance to
	// control/mental effects.
	MentalResistanceBonus float64 `json:"mental_resistance"`
	// SoulDefenseBonus is a percentage reduction of soul damage taken.
	SoulDefenseBonus float64 `json:"soul_defense"`
	// SoulRegenBonus is a percentage increase of passive soul_hp regeneration
	// (dưỡng hồn). E.g. 0.50 => +50% soul_hp recovered per hour.
	SoulRegenBonus float64 `json:"soul_regen"`
	// Tenacity is the generic self-dispel chance rolled at turn start (0..1)
	// and the CC duration reduction. Aggregated from Gu passive, race,
	// constitution and status effects — never hardcoded per Gu ID.
	Tenacity float64 `json:"tenacity"`
	// CCResist is the generic proc resist vs all debuffs (0..1). Stacks
	// multiplicatively with MentalResistanceBonus in the proc formula.
	CCResist float64 `json:"cc_resist"`
	// ApertureStressBonus is an additive multiplier on the aperture stress
	// (áp lực không khiếu) gained per trùng kích attempt. 1.0 => stress ×2.
	ApertureStressBonus float64 `json:"aperture_stress_bonus"`
	// NonConflictingDaoMarks indicates whether character has the non-conflicting dao marks trait.
	NonConflictingDaoMarks bool `json:"non_conflicting_dao_marks,omitempty"`
	// Immortal indicates if the character's lifespan never decreases.
	Immortal bool `json:"immortal"`
	// DisableHPRegen disables HP regeneration.
	DisableHPRegen bool `json:"disable_hp_regen"`
	// DisableManaRegen disables Mana regeneration.
	DisableManaRegen bool `json:"disable_mana_regen"`
	// DisableResourceRegen disables Tiên/Chân Nguyên recovery.
	DisableResourceRegen bool `json:"disable_resource_regen"`
	// DisableSoulRegen disables Soul/Thần Hồn regeneration.
	DisableSoulRegen bool `json:"disable_soul_regen"`
	// BlockedActions is the set of gameplay actions currently gated by active
	// statuses (keys: gamefx.Action* constants, e.g. "explore", "cultivate").
	// Handlers consult it via HasBlockedAction before executing a flow.
	BlockedActions map[string]bool `json:"blocked_actions,omitempty"`
}

// ApplyToEnemyCombatState populates all defender character effect properties into a CombatState.
func (e Effects) ApplyToEnemyCombatState(combat *models.CombatState) {
	if combat == nil {
		return
	}
	combat.EnemyPathResistance = e.PathResistanceBonus
	combat.EnemyMentalResistance = e.MentalResistanceBonus
	combat.EnemyTenacity = e.Tenacity
	combat.EnemyCCResist = e.CCResist
	combat.EnemyDaoBonus = e.DaoNganBonus
	combat.EnemyPathDamageBonus = e.PathDamageBonus
	combat.EnemyHealingReceived = e.HealingReceivedBonus
	combat.EnemySoulDefenseBonus = e.SoulDefenseBonus
	combat.EnemyLuckBonus = e.LuckBonus
}

// ApplyEnemyEffects evaluates character effects from defender and applies them to combat.
func ApplyEnemyEffects(combat *models.CombatState, defender *models.Player) {
	if combat == nil || defender == nil {
		return
	}
	Get(defender).ApplyToEnemyCombatState(combat)
}

// ValidRaces returns the set of accepted race keys.
func ValidRaces() map[string]bool {
	defs := effectiveRaces()
	out := make(map[string]bool, len(defs))
	for key := range defs {
		out[key] = true
	}
	return out
}

// ValidGenders returns the set of accepted gender keys.
func ValidGenders() map[string]bool {
	return map[string]bool{
		"male":   true,
		"female": true,
	}
}

// ValidConstitutions returns the set of accepted constitution keys.
func ValidConstitutions() map[string]bool {
	defs := effectiveConstitutions()
	out := make(map[string]bool, len(defs))
	for key := range defs {
		out[key] = true
	}
	return out
}

// Get returns all effects from a player's character fields.
func Get(p *models.Player) Effects {
	fx := Effects{
		StatMods:            map[string]float64{},
		DaoNganBonus:        map[string]float64{},
		PathDamageBonus:     map[string]float64{},
		PathResistanceBonus: map[string]float64{},
	}
	if p == nil {
		return fx
	}

	if def, ok := effectiveRaces()[p.Race]; ok {
		addEffects(&fx, def.Effects)
	}

	for _, status := range p.Statuses.Active() {
		if def, ok := effectiveStatuses()[status.ID]; ok {
			eff := def.Effects
			if status.Value < 0 && len(eff.StatMods) > 0 {
				mods := make(map[string]float64, len(eff.StatMods))
				for k := range eff.StatMods {
					mods[k] = status.Value
				}
				eff.StatMods = mods
			}
			addEffects(&fx, eff)
			if fx.BlockedActions == nil {
				fx.BlockedActions = map[string]bool{}
			}
			for _, action := range def.BlockedActions {
				fx.BlockedActions[action] = true
			}
		}
	}

	if def, ok := effectiveConstitutions()[p.Constitution]; ok {
		addEffects(&fx, def.Effects)
	}

	// Gu passive effects (equipped ACTIVE only, best rank per GuID).
	// Generic source for tenacity / cc_resist — no per-Gu-ID branching.
	bestRank := map[string]int{}
	for _, gu := range p.GuInventory {
		if !gu.Equipped || gu.Status != "ACTIVE" {
			continue
		}
		if r, ok := bestRank[gu.GuID]; !ok || gu.Rank > r {
			bestRank[gu.GuID] = gu.Rank
		}
	}
	applied := map[string]bool{}
	for _, gu := range p.GuInventory {
		if !gu.Equipped || gu.Status != "ACTIVE" {
			continue
		}
		if gu.Rank != bestRank[gu.GuID] || applied[gu.GuID] {
			continue
		}
		applied[gu.GuID] = true
		spec, ok := models.GetGu(gu.GuID, gu.Rank)
		if !ok || spec.Effects.Passive == nil {
			continue
		}
		addEffects(&fx, Effects{
			Tenacity: spec.Effects.Passive.Tenacity,
			CCResist: spec.Effects.Passive.CCResist,
		})
	}

	return fx
}

// CombinedResist merges mental + generic CC resist multiplicatively, capped at 0.75.
func CombinedResist(mental, cc float64) float64 {
	r := 1.0 - (1.0-mental)*(1.0-cc)
	if r > 0.75 {
		r = 0.75
	}
	if r < 0 {
		r = 0
	}
	return r
}

// CappedTenacity clamps tenacity to the 0..0.75 playable range.
func CappedTenacity(t float64) float64 {
	if t > 0.75 {
		return 0.75
	}
	if t < 0 {
		return 0
	}
	return t
}

// MaxManaBonus returns the constitution percentage bonus on max mana.
func MaxManaBonus(p *models.Player) float64 {
	if p == nil {
		return 0
	}
	return Get(p).MaxManaBonus
}

// EffectiveMaxMana returns the player's mana pool capacity including the
// constitution MaxManaBonus.
func EffectiveMaxMana(p *models.Player) float64 {
	if p == nil {
		return 0
	}
	return p.MaxMana * (1.0 + MaxManaBonus(p))
}

// PassiveHPBonus returns the total flat HP granted by equipped ACTIVE Gus
// (passive.stat_base_hp), scaled by the unified DaoCombatProfile. Mirrors the
// "hp" key computed in GetPassiveStats.
func PassiveHPBonus(p *models.Player) float64 {
	if p == nil {
		return 0
	}
	fx := Get(p)
	profile := BuildDaoCombatProfile(p.DaoMarks, p.PathMasteries, fx.NonConflictingDaoMarks, fx.DaoNganBonus)
	total := 0.0
	bestRank := map[string]int{}
	for _, gu := range p.GuInventory {
		if !gu.Equipped || gu.Status != "ACTIVE" {
			continue
		}
		if r, ok := bestRank[gu.GuID]; !ok || gu.Rank > r {
			bestRank[gu.GuID] = gu.Rank
		}
	}
	applied := map[string]bool{}
	for _, gu := range p.GuInventory {
		if !gu.Equipped || gu.Status != "ACTIVE" {
			continue
		}
		if gu.Rank != bestRank[gu.GuID] || applied[gu.GuID] {
			continue
		}
		applied[gu.GuID] = true
		spec, ok := models.GetGu(gu.GuID, gu.Rank)
		if !ok || spec.Effects.Passive == nil || spec.Effects.Passive.StatBaseHP <= 0 {
			continue
		}
		pathKey := "strength"
		if len(spec.Path) > 0 {
			pathKey = spec.Path[0]
		}
		daoAmp, realmEff, conflictMult := profile.GetOffensiveFactors(pathKey)
		daoMult := (1.0 + (daoAmp-1.0)*realmEff) * conflictMult
		total += spec.Effects.Passive.StatBaseHP * daoMult
	}
	return total
}

// EffectiveMaxHP returns the player's max HP including passive HP bonuses
// from equipped Gus. This is the authoritative max HP for HUD, regen caps and
// combat.
func EffectiveMaxHP(p *models.Player) float64 {
	if p == nil {
		return 0
	}
	base := p.BaseHP + PassiveHPBonus(p)
	for _, st := range p.Statuses.Active() {
		if st.ID == "buff_hp" {
			base += st.Value
		}
	}
	return base
}

// RaceDisplayName returns the Vietnamese label for a race key.
func RaceDisplayName(race string) string {
	if def, ok := effectiveRaces()[race]; ok {
		return def.Name
	}
	return race
}

// GenderDisplayName returns the Vietnamese label for a gender key.
func GenderDisplayName(gender string) string {
	switch gender {
	case "male":
		return "Nam"
	case "female":
		return "Nữ"
	default:
		return gender
	}
}

// ConstitutionDisplayName returns the Vietnamese label for a constitution key.
func ConstitutionDisplayName(constitution string) string {
	if def, ok := effectiveConstitutions()[constitution]; ok {
		return def.Name
	}
	return constitution
}

// StatusDisplayName returns the Vietnamese label for a status key.
func StatusDisplayName(status string) string {
	if def, ok := effectiveStatuses()[status]; ok {
		return def.Name
	}
	return status
}

// HasBlockedAction reports whether any of the player's active (non-expired)
// statuses block the given action.
func HasBlockedAction(p *models.Player, action string) bool {
	if p == nil {
		return false
	}
	if p.IsDead {
		return true
	}
	return Get(p).BlockedActions[action]
}

// PathDamageMultiplier returns the percentage damage multiplier for a path
// from pathDamage (e.g. PathDamageBonus). 1.0 means no bonus.
func PathDamageMultiplier(path string, pathDamage map[string]float64) float64 {
	if pathDamage == nil {
		return 1.0
	}
	cleanPath := strings.TrimSuffix(strings.ToLower(path), " path")
	return 1.0 + pathDamage[cleanPath]
}

// PathDamageMultiplierForPaths returns the maximum path-damage multiplier
// across the given paths. 1.0 means no bonus.
func PathDamageMultiplierForPaths(paths []string, pathDamage map[string]float64) float64 {
	if pathDamage == nil {
		return 1.0
	}
	if len(paths) == 0 {
		paths = []string{"strength"}
	}
	maxMult := 1.0
	for _, path := range paths {
		if mult := PathDamageMultiplier(path, pathDamage); mult > maxMult {
			maxMult = mult
		}
	}
	return maxMult
}
