// Package gamefx resolves gameplay modifiers granted by a player's character
// attributes (race, gender, constitution, statuses). It is shared by the
// handler and service layers to avoid an import cycle.
package gamefx

import (
	"bytes"
	"encoding/json"
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
	// RawBlocks preserves the original block-v2 JSON array when loaded from DB or CSV.
	RawBlocks json.RawMessage `json:"-"`
}

// UnmarshalJSON unmarshals either a legacy Effects object or a pure Block v2 array [...].
func (e *Effects) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		*e = Effects{}
		return nil
	}

	// 1. If it's a JSON array `[...]`, parse blocks and preserve raw bytes
	if trimmed[0] == '[' {
		err := e.unmarshalBlocks(trimmed)
		if err == nil {
			e.RawBlocks = append([]byte(nil), trimmed...)
		}
		return err
	}

	// 2. If it's a JSON object `{...}`:
	// Check if it's an envelope `{"blocks": [...]}`
	var root map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &root); err != nil {
		return err
	}
	if b, ok := root["blocks"]; ok {
		trimmedB := bytes.TrimSpace(b)
		if len(trimmedB) > 0 && trimmedB[0] == '[' {
			err := e.unmarshalBlocks(trimmedB)
			if err == nil {
				e.RawBlocks = append([]byte(nil), trimmedB...)
			}
			return err
		}
	}

	// Legacy struct fields
	type alias Effects
	var a alias
	if err := json.Unmarshal(trimmed, &a); err != nil {
		return err
	}
	*e = Effects(a)
	return nil
}

// MarshalJSON marshals Effects to Block v2 array format [...].
func (e Effects) MarshalJSON() ([]byte, error) {
	if len(e.RawBlocks) > 0 {
		trimmed := bytes.TrimSpace(e.RawBlocks)
		if len(trimmed) > 0 && (trimmed[0] == '[' || bytes.Contains(trimmed, []byte(`"blocks"`))) {
			return trimmed, nil
		}
	}
	return json.Marshal(e.ToBlocks())
}

// ToBlocks converts the struct fields into a canonical Block v2 array.
func (e Effects) ToBlocks() []any {
	type target struct {
		Selector string `json:"selector"`
	}
	type op struct {
		Op     string         `json:"op"`
		Code   string         `json:"code"`
		Params map[string]any `json:"params"`
	}
	type blockItem struct {
		ID         string `json:"id"`
		Trigger    string `json:"trigger"`
		Target     target `json:"target"`
		Operations []op   `json:"operations"`
	}

	blocks := make([]any, 0)
	addBlock := func(id, code string, val any) {
		blocks = append(blocks, blockItem{
			ID:      id,
			Trigger: "ALWAYS_ON",
			Target:  target{Selector: "SELF"},
			Operations: []op{{
				Op:     "UNIQUE",
				Code:   code,
				Params: map[string]any{"value": val},
			}},
		})
	}

	if len(e.StatMods) > 0 {
		addBlock("STAT_MODS", "STAT_MODS", e.StatMods)
	}
	if len(e.DaoNganBonus) > 0 {
		addBlock("DAO_MARK_MODIFIER", "DAO_MARK_MODIFIER", e.DaoNganBonus)
	}
	if len(e.PathDamageBonus) > 0 {
		addBlock("PATH_DAMAGE_MODIFIER", "PATH_DAMAGE_MODIFIER", e.PathDamageBonus)
	}
	if len(e.PathResistanceBonus) > 0 {
		addBlock("PATH_RESISTANCE_MODIFIER", "PATH_RESISTANCE_MODIFIER", e.PathResistanceBonus)
	}
	if e.RefineBonus != 0 {
		addBlock("REFINE_SUCCESS", "REFINE_SUCCESS", e.RefineBonus)
	}
	if e.LuckBonus != 0 {
		addBlock("LUCK", "LUCK", e.LuckBonus)
	}
	if e.MaxManaBonus != 0 {
		addBlock("MAX_MANA", "MAX_MANA", e.MaxManaBonus)
	}
	if e.ManaRegenBonus != 0 {
		addBlock("MANA_REGEN", "MANA_REGEN", e.ManaRegenBonus)
	}
	if e.CultivationEfficiencyBonus != 0 {
		addBlock("CULTIVATION_EFFICIENCY", "CULTIVATION_EFFICIENCY", e.CultivationEfficiencyBonus)
	}
	if e.DaoMarkGainBonus != 0 {
		addBlock("DAO_MARK_GAIN", "DAO_MARK_GAIN", e.DaoMarkGainBonus)
	}
	if e.HealingReceivedBonus != 0 {
		addBlock("HEALING_RECEIVED", "HEALING_RECEIVED", e.HealingReceivedBonus)
	}
	if e.HPRegenBonus != 0 {
		addBlock("HP_REGEN", "HP_REGEN", e.HPRegenBonus)
	}
	if e.MentalResistanceBonus != 0 {
		addBlock("MENTAL_RESISTANCE", "MENTAL_RESISTANCE", e.MentalResistanceBonus)
	}
	if e.SoulDefenseBonus != 0 {
		addBlock("SOUL_DEFENSE", "SOUL_DEFENSE", e.SoulDefenseBonus)
	}
	if e.SoulRegenBonus != 0 {
		addBlock("SOUL_REGEN", "SOUL_REGEN", e.SoulRegenBonus)
	}
	if e.Tenacity != 0 {
		addBlock("TENACITY", "TENACITY", e.Tenacity)
	}
	if e.CCResist != 0 {
		addBlock("CC_RESIST", "CC_RESIST", e.CCResist)
	}
	if e.ApertureStressBonus != 0 {
		addBlock("APERTURE_STRESS", "APERTURE_STRESS", e.ApertureStressBonus)
	}
	if e.NonConflictingDaoMarks {
		addBlock("NON_CONFLICTING_DAO_MARKS", "NON_CONFLICTING_DAO_MARKS", true)
	}
	if e.Immortal {
		addBlock("IMMORTAL", "IMMORTAL", true)
	}
	if e.DisableHPRegen {
		addBlock("DISABLE_HP_REGEN", "DISABLE_HP_REGEN", true)
	}
	if e.DisableManaRegen {
		addBlock("DISABLE_MANA_REGEN", "DISABLE_MANA_REGEN", true)
	}
	if e.DisableResourceRegen {
		addBlock("DISABLE_RESOURCE_REGEN", "DISABLE_RESOURCE_REGEN", true)
	}
	if e.DisableSoulRegen {
		addBlock("DISABLE_SOUL_REGEN", "DISABLE_SOUL_REGEN", true)
	}

	return blocks
}

func (e *Effects) unmarshalBlocks(data []byte) error {
	*e = Effects{
		StatMods:            map[string]float64{},
		DaoNganBonus:        map[string]float64{},
		PathDamageBonus:     map[string]float64{},
		PathResistanceBonus: map[string]float64{},
	}

	type blockOp struct {
		Op      string          `json:"op"`
		Code    string          `json:"code"`
		Stat    string          `json:"stat"`
		SubType string          `json:"sub_type"`
		Path    string          `json:"path"`
		Unit    string          `json:"unit"`
		Value   any             `json:"value"`
		Params  json.RawMessage `json:"params"`
	}
	type blockItem struct {
		Operations []blockOp `json:"operations"`
	}

	var blocks []blockItem
	if err := json.Unmarshal(data, &blocks); err != nil {
		return err
	}

	asFloat := func(v any) float64 {
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		default:
			return 0
		}
	}
	asBool := func(v any) bool {
		if b, ok := v.(bool); ok {
			return b
		}
		return false
	}
	asMap := func(v any) map[string]float64 {
		m := make(map[string]float64)
		if rawMap, ok := v.(map[string]any); ok {
			for k, val := range rawMap {
				m[k] = asFloat(val)
			}
		}
		return m
	}

	for _, b := range blocks {
		for _, op := range b.Operations {
			var p struct {
				Value any `json:"value"`
			}
			if len(op.Params) > 0 {
				_ = json.Unmarshal(op.Params, &p)
			}
			val := p.Value
			if val == nil && op.Value != nil {
				val = op.Value
			}

			// 1. Direct STAT_MOD handling (canonical Block v2)
			if strings.EqualFold(op.Op, "STAT_MOD") && op.Stat != "" {
				statUpper := strings.ToUpper(op.Stat)
				switch statUpper {
				case "ATTACK", "DEFENSE", "SPEED", "HP":
					e.StatMods[strings.ToLower(statUpper)] += asFloat(val)
				case "DAO_MARK":
					if op.Path != "" {
						e.DaoNganBonus[strings.ToLower(op.Path)] += asFloat(val)
					}
				case "PATH_DAMAGE":
					if op.Path != "" {
						e.PathDamageBonus[strings.ToLower(op.Path)] += asFloat(val)
					}
				case "PATH_RESISTANCE":
					if op.Path != "" {
						e.PathResistanceBonus[strings.ToLower(op.Path)] += asFloat(val)
					}
				case "LUCK":
					e.LuckBonus += asFloat(val)
				case "REFINE_SUCCESS":
					e.RefineBonus += asFloat(val)
				case "CULTIVATION_EFFICIENCY":
					e.CultivationEfficiencyBonus += asFloat(val)
				case "DAO_MARK_GAIN":
					e.DaoMarkGainBonus += asFloat(val)
				case "APERTURE_STRESS":
					e.ApertureStressBonus += asFloat(val)
				case "TENACITY":
					e.Tenacity += asFloat(val)
				case "CC_RESIST":
					e.CCResist += asFloat(val)
				default:
					e.StatMods[strings.ToLower(statUpper)] += asFloat(val)
				}
				continue
			}

			// 2. Direct HEAL handling
			if strings.EqualFold(op.Op, "HEAL") {
				if strings.EqualFold(op.SubType, "MANA") {
					e.ManaRegenBonus += asFloat(val)
				} else if strings.EqualFold(op.SubType, "LIFESTEAL") {
					e.HealingReceivedBonus += asFloat(val)
				}
				continue
			}

			// 3. UNIQUE op or code-based mapping (authoring format & CSV block v2)
			code := op.Code
			if code == "" && op.Op != "" && !strings.EqualFold(op.Op, "UNIQUE") {
				code = op.Op
			}

			switch strings.ToUpper(code) {
			case "STAT_MODS":
				for k, v := range asMap(val) {
					e.StatMods[k] += v
				}
			case "REFINE_SUCCESS":
				e.RefineBonus += asFloat(val)
			case "DAO_MARK_MODIFIER":
				for k, v := range asMap(val) {
					e.DaoNganBonus[k] += v
				}
			case "PATH_DAMAGE_MODIFIER":
				for k, v := range asMap(val) {
					e.PathDamageBonus[k] += v
				}
			case "PATH_RESISTANCE_MODIFIER":
				for k, v := range asMap(val) {
					e.PathResistanceBonus[k] += v
				}
			case "LUCK":
				e.LuckBonus += asFloat(val)
			case "MAX_MANA":
				e.MaxManaBonus += asFloat(val)
			case "MANA_REGEN":
				e.ManaRegenBonus += asFloat(val)
			case "CULTIVATION_EFFICIENCY":
				e.CultivationEfficiencyBonus += asFloat(val)
			case "DAO_MARK_GAIN":
				e.DaoMarkGainBonus += asFloat(val)
			case "HEALING_RECEIVED":
				e.HealingReceivedBonus += asFloat(val)
			case "HP_REGEN":
				e.HPRegenBonus += asFloat(val)
			case "MENTAL_RESISTANCE":
				e.MentalResistanceBonus += asFloat(val)
			case "SOUL_DEFENSE":
				e.SoulDefenseBonus += asFloat(val)
			case "SOUL_REGEN":
				e.SoulRegenBonus += asFloat(val)
			case "TENACITY":
				e.Tenacity += asFloat(val)
			case "CC_RESIST":
				e.CCResist += asFloat(val)
			case "APERTURE_STRESS":
				e.ApertureStressBonus += asFloat(val)
			case "NON_CONFLICTING_DAO_MARKS":
				e.NonConflictingDaoMarks = asBool(val)
			case "IMMORTAL":
				e.Immortal = asBool(val)
			case "DISABLE_HP_REGEN":
				e.DisableHPRegen = asBool(val)
			case "DISABLE_MANA_REGEN":
				e.DisableManaRegen = asBool(val)
			case "DISABLE_RESOURCE_REGEN":
				e.DisableResourceRegen = asBool(val)
			case "DISABLE_SOUL_REGEN":
				e.DisableSoulRegen = asBool(val)
			}
		}
	}
	return nil
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
			eff := def.EffectsForValue(status.Value)
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
		if !ok {
			continue
		}
		addEffects(&fx, Effects{
			Tenacity:               models.GuStatValue(spec, "TENACITY"),
			CCResist:               models.GuStatValue(spec, "CC_RESIST"),
			NonConflictingDaoMarks: models.GuStatValue(spec, "PATH_CONFLICT_REDUCTION") >= 100,
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

// GuStatMultipliers returns the two multipliers a Gu passive's stat values are
// scaled by: the highest path-mastery attainment multiplier across the Gu's
// paths, and the dao-mark amplification (1 + marks/100 of the best-marked
// path). Shared by the handlers' stat pipeline and PassiveHPBonus so every Gu
// stat source scales identically.
func GuStatMultipliers(pathMasteries map[string]int, daoMarks map[string]int64, paths []string) (masteryMult, daoMult float64) {
	masteryMult = PathMasteryMultiplier(pathMasteries, paths)
	daoMult = 1.0
	if daoMarks != nil {
		for _, path := range paths {
			clean := CleanPath(path)
			if amt, ok := daoMarks[clean]; ok && amt > 0 {
				if pm := DaoAmplification(amt); pm > daoMult {
					daoMult = pm
				}
			}
		}
	}
	return masteryMult, daoMult
}

// PassiveHPBonus returns the total flat HP granted by equipped ACTIVE Gus
// (passive.stat_base_hp), scaled by path mastery and dao marks exactly like
// the "hp" key of the handlers' stat pipeline (GetPassiveStats), so
// EffectiveMaxHP always matches the MaxHP combat uses.
func PassiveHPBonus(p *models.Player) float64 {
	if p == nil {
		return 0
	}
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
		if !ok {
			continue
		}
		baseHP := models.GuStatValue(spec, "HP")
		if baseHP <= 0 {
			continue
		}
		masteryMult, daoMult := GuStatMultipliers(p.PathMasteries, p.DaoMarks, spec.Path)
		total += baseHP * masteryMult * daoMult
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

// IsDeadState reports whether the player has entered Cõi Chết. Lifespan is the
// single source of truth — the immortal race never decays lifespan, so a
// zero-lifespan immortal is not dead.
func IsDeadState(p *models.Player) bool {
	if p == nil || p.Lifespan > 0 {
		return false
	}
	// Vĩnh Sinh (Rank 10) is eternally alive: aging never depletes lifespan.
	if models.IsVinhSinhRank(p.Rank) {
		return false
	}
	dummyP := models.Player{}
	dummyP.Race = p.Race
	return !Get(&dummyP).Immortal
}

// HasBlockedAction reports whether any of the player's active (non-expired)
// statuses block the given action.
func HasBlockedAction(p *models.Player, action string) bool {
	if p == nil {
		return false
	}
	if IsDeadState(p) {
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
