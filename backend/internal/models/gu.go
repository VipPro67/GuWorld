package models

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// StatusApplication is one entry of active.apply_status: a statusfx status to
// inflict, with optional explicit intensity and duration. Omitted value/turns
// inherit the legacy top-level status_tick_damage / status_duration_ticks, so
// a bare string entry keeps its exact historical semantics.
type StatusApplication struct {
	ID    string  `json:"id"`
	Value float64 `json:"value,omitempty"` // intensity (DoT damage, percent); 0 = inherit
	Turns int     `json:"turns,omitempty"` // duration in combat ticks; 0 = inherit
}

// StatusApplications accepts three authoring forms for the active.apply_status
// key so one action can assemble several statuses:
//
//	"BIND"                                legacy single string
//	{"id": "BIND", "turns": 3}            single object
//	[{"id": "BLEED"}, {"id": "BIND"}]     array (the preferred form)
type StatusApplications []StatusApplication

func (s *StatusApplications) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		*s = nil
		return nil
	}
	switch trimmed[0] {
	case '"':
		var id string
		if err := json.Unmarshal(trimmed, &id); err != nil {
			return err
		}
		id = strings.ToUpper(strings.TrimSpace(id))
		if id == "" {
			*s = nil
			return nil
		}
		*s = StatusApplications{{ID: id}}
	case '{':
		var one StatusApplication
		if err := json.Unmarshal(trimmed, &one); err != nil {
			return err
		}
		one.ID = strings.ToUpper(strings.TrimSpace(one.ID))
		*s = StatusApplications{one}
	case '[':
		var list []StatusApplication
		if err := json.Unmarshal(trimmed, &list); err != nil {
			return err
		}
		for i := range list {
			list[i].ID = strings.ToUpper(strings.TrimSpace(list[i].ID))
		}
		*s = list
	default:
		return fmt.Errorf("apply_status phải là chuỗi, object hoặc mảng")
	}
	return nil
}

// HasStatus reports whether the action applies the given canonical status ID.
func (a *ActiveEffect) HasStatus(id string) bool {
	for _, entry := range a.ApplyStatus {
		if entry.ID == id {
			return true
		}
	}
	return false
}

// StatusOp returns the resolved statusfx op for the first apply_status entry
// matching id (e.g. overworld skill handling of REGEN/ATK_UP buffs).
func (a *ActiveEffect) StatusOp(id string) (statusfx.StatusOp, bool) {
	for _, op := range a.ResolveStatusApplications() {
		if op.ID == id {
			return op, true
		}
	}
	return statusfx.StatusOp{}, false
}

// effectiveStatusValue returns the intensity for an entry, falling back to the
// legacy top-level status_tick_damage.
func (a *ActiveEffect) effectiveStatusValue(entry StatusApplication) float64 {
	if entry.Value > 0 {
		return entry.Value
	}
	return a.StatusTickDamage
}

// effectiveStatusTurns returns the duration for an entry, falling back to the
// legacy top-level status_duration_ticks.
func (a *ActiveEffect) effectiveStatusTurns(entry StatusApplication) int {
	if entry.Turns > 0 {
		return entry.Turns
	}
	return a.StatusDurationTicks
}

// ResolveStatusApplications resolves every apply_status entry through the
// statusfx registry into combat StatusOps. Explicit entry values/turns are
// direct; the legacy tier pipeline (top-level status_duration_ticks ≤ 5 read
// as a tier) only applies to entries inheriting the old top-level fields.
func (a *ActiveEffect) ResolveStatusApplications() []statusfx.StatusOp {
	ops := make([]statusfx.StatusOp, 0, len(a.ApplyStatus))
	for _, entry := range a.ApplyStatus {
		value := a.effectiveStatusValue(entry)
		turns := a.effectiveStatusTurns(entry)
		var op statusfx.StatusOp
		var err error
		switch {
		case value > 0:
			op, err = statusfx.ResolveStatus(entry.ID, value, turns)
		case entry.Turns > 0:
			op, err = statusfx.ResolveStatus(entry.ID, 0, entry.Turns)
		case turns > 5:
			op, err = statusfx.ResolveStatus(entry.ID, 0, turns)
		case turns > 0:
			op, err = statusfx.ResolveStatusWithTier(entry.ID, turns)
		default:
			op, err = statusfx.ResolveStatus(entry.ID, 0, 0)
		}
		if err != nil {
			continue
		}
		// FORGET_REALITY intensity comes from target_damage_reduce_percent.
		if op.ID == statusfx.StatusForgetReality && a.TargetDamageReducePct > 0 {
			op.Value = a.TargetDamageReducePct * 100
		}
		ops = append(ops, op)
	}
	return ops
}

type ActiveEffect struct {
	DamageFlat          float64            `json:"damage_flat,omitempty"`
	HealFlat            float64            `json:"heal_flat,omitempty"`
	ShieldFlat          float64            `json:"shield_flat,omitempty"`
	ApplyStatus         StatusApplications `json:"apply_status,omitempty"`
	StatusTickDamage    float64            `json:"status_tick_damage,omitempty"`
	StatusDurationTicks int                `json:"status_duration_ticks,omitempty"`
	TargetSelector      string             `json:"target_selector,omitempty"`

	Range              float64 `json:"range,omitempty"`
	CurveAttack        bool    `json:"curve_attack,omitempty"`
	ManaRegen          float64 `json:"mana_regen,omitempty"`
	WoodTransformation bool    `json:"wood_transformation,omitempty"`
	BloodDrain         bool    `json:"blood_drain,omitempty"`
	Replicate          bool    `json:"replicate,omitempty"`
	LifeStealRate      float64 `json:"lifesteal_rate,omitempty"`
	RefineManaTier     float64 `json:"refine_mana_tier,omitempty"`
	DamageMult         float64 `json:"damage_mult,omitempty"`
	ShieldMult          float64 `json:"shield_mult,omitempty"`
	CleansingPower     float64 `json:"cleansing_power,omitempty"`
	SummonPhantom      bool    `json:"summon_phantom,omitempty"`
	GenerateGu         bool    `json:"generate_gu,omitempty"`
	CaptureBonus       float64 `json:"capture_bonus,omitempty"` // +% tỉ lệ bắt monster (0..1)
	PortalCreate       bool    `json:"portal_create,omitempty"`
	RandomPortal       bool    `json:"random_portal,omitempty"`

	SummonGrowthPhantoms bool    `json:"summon_growth_phantoms,omitempty"`
	SummonCountPerTick   float64 `json:"summon_count_per_tick,omitempty"`
	PhantomDamageScale   float64 `json:"phantom_damage_scale,omitempty"`

	// Combat modifier fields
	IgnoreDefense    float64 `json:"ignore_defense,omitempty"`         // flat defense ignored
	IgnoreDefensePct float64 `json:"ignore_defense_pct,omitempty"`     // % defense ignored (0..1)
	CritRateBoost    float64 `json:"crit_rate_boost,omitempty"`        // extra crit chance (0..1)
	AllStatsBuffPct  float64 `json:"all_stats_buff_percent,omitempty"` // % buff to all stats (0..1)
	SpeedBoostFlat   float64 `json:"speed_boost_flat,omitempty"`       // flat speed bonus

	// Percent-based direct effects (0..1). Generic: any Gu may set them.
	DamageHPPercent          float64 `json:"damage_hp_percent,omitempty"`          // % target MaxHP as direct HP damage (bypasses shield)
	ShieldFlatReducePercent  float64 `json:"shield_flat_reduce_percent,omitempty"` // % target current shield stripped
	DisableGuNextTurn        bool    `json:"disable_gu_next_turn,omitempty"`       // seal target Gu for 1 turn (GU_SEAL)
	ForgetRealityChance      float64 `json:"forget_reality_chance,omitempty"`      // proc gate (0..1) for the action's status + bonus effects
	SelfEvasionBoostPercent  float64 `json:"self_evasion_boost_percent,omitempty"` // self evasion buff (0..1)
	TargetDamageReducePct    float64 `json:"target_damage_reduce_percent,omitempty"`
	DamagePath               string  `json:"damage_path,omitempty"`      // overrides action dao path (e.g. "Dream Path")
	StatBaseAttackFlat       float64 `json:"stat_base_attack,omitempty"` // flat bonus added to action damage

	// Shorthand status effect fields — tier value (e.g. "freeze": 1 = speed T1)
	// Normalized to ApplyStatus + StatusDurationTicks during loading.
	Freeze  float64 `json:"freeze,omitempty"`
	Slow    float64 `json:"slow,omitempty"`
	Slow1   float64 `json:"slow1,omitempty"`
	Weak    float64 `json:"weak,omitempty"`
	Stun    float64 `json:"stun,omitempty"`
	Silence float64 `json:"silence,omitempty"`
	Bleed   float64 `json:"bleed,omitempty"`
	Poison  float64 `json:"poison,omitempty"`
	Thorns  float64 `json:"thorns,omitempty"`
	Regen   float64 `json:"regen,omitempty"`
	AtkUp   float64 `json:"atk_up,omitempty"`
	DefUp   float64 `json:"def_up,omitempty"`
	Taunt   float64 `json:"taunt,omitempty"`
}

type PassiveEffect struct {
	StatBaseAttack        float64 `json:"stat_base_attack,omitempty"`
	StatBaseDefense       float64 `json:"stat_base_defense,omitempty"`
	StatBaseHP            float64 `json:"stat_base_hp,omitempty"`
	StatBaseSpeed         float64 `json:"stat_base_speed,omitempty"`
	PathConflictReduction float64 `json:"path_conflict_reduction,omitempty"`

	DamageMultiplierOnHit float64 `json:"damage_multiplier_on_hit,omitempty"`
	SoulRefinement        float64 `json:"soul_refinement,omitempty"`
	StorageCapacity       float64 `json:"storage_capacity,omitempty"`
	Movement              float64 `json:"movement,omitempty"`
	Investigation         float64 `json:"investigation,omitempty"`
	Hearing               float64 `json:"hearing,omitempty"`
	Evasion               float64 `json:"evasion,omitempty"`
	RefineManaTier        float64 `json:"refine_mana_tier,omitempty"`
	CleansingPower        float64 `json:"cleansing_power,omitempty"`
	Luck                  float64 `json:"luck,omitempty"`
	SuccessRateBonus      float64 `json:"success_rate_bonus,omitempty"`
	TimeSpeedMultiplier   float64 `json:"time_speed_multiplier,omitempty"`
	Accuracy              float64 `json:"accuracy,omitempty"` // accuracy bonus (reduces enemy dodge)
	DisguiseLevel         float64 `json:"disguise_level,omitempty"`
	Tenacity              float64 `json:"tenacity,omitempty"`  // self-dispel chance + CC duration reduction (0..1)
	CCResist              float64 `json:"cc_resist,omitempty"` // generic proc resist vs all debuffs (0..1)

	// Conditional path counters (0..1 damage reduction when the attacker's
	// Gu path matches; resolved by Gu path, e.g. "Heaven Path" → "heaven").
	AntiHeavenPct            float64 `json:"anti_heaven_damage_reduction_percent,omitempty"`
	AntiEarthPct             float64 `json:"anti_earth_damage_reduction_percent,omitempty"`
	AntiDreamPct             float64 `json:"anti_dream_damage_reduction_percent,omitempty"`
	ImmuneDreamStatus        bool    `json:"immune_dream_status,omitempty"` // immune to dream-path statuses
	ForgetHeavenProtection   bool    `json:"forget_heaven_protection,omitempty"`
	ForgetEarthProtection    bool    `json:"forget_earth_protection,omitempty"`
}

type SpecialMechanic struct {
	EventTrigger string          `json:"event_trigger"`
	ActionCode   string          `json:"action_code"`
	Params       json.RawMessage `json:"params"`
}

type GrowthEffect struct {
	MaxAttack  float64 `json:"max_attack,omitempty"`
	MaxDefense float64 `json:"max_defense,omitempty"`
	MaxHP      float64 `json:"max_hp,omitempty"`
	MaxSpeed   float64 `json:"max_speed,omitempty"`
	HoursToMax float64 `json:"hours_to_max,omitempty"`
}

type GuEffects struct {
	Active  *ActiveEffect     `json:"active"`
	Passive *PassiveEffect    `json:"passive"`
	Growth  *GrowthEffect     `json:"growth,omitempty"`
	Other   []SpecialMechanic `json:"other"`
}

type GuDictionaryItem struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Rank         int       `json:"rank"`
	Path         []string  `json:"path"`
	Type         string    `json:"gu_type"`
	BaseManaCost float64   `json:"base_mana_cost"`
	Effects      GuEffects `json:"effects"`
	FoodTypes    []string  `json:"food_types"`
	HungerRate   int       `json:"hunger_rate"`
	MaxHunger    int       `json:"max_hunger"`
	Description  string    `json:"description"`
	Rarity       int       `json:"rarity"`
	IsImmortal   bool      `json:"is_immortal"`
	PrimaryRole  string    `json:"primary_role"`
	CombatRoles  []string  `json:"combat_roles"`
	CombatTags   []string  `json:"combat_tags"`
	SupportTags  []string  `json:"support_tags"`
	BasePrice    float64   `json:"base_price"`
	ImageURL     string    `json:"image_url"`
}

type Gu = GuDictionaryItem

// GuMechanic resolves a gameplay capability from the catalog effect column.
// Callers may still use GuID to locate the definition, but must not infer
// behavior from that identifier.
func GuMechanic(instance GuInstance, actionCode string) (SpecialMechanic, bool) {
	spec, ok := GetGu(instance.GuID, instance.Rank)
	if !ok {
		spec, ok = GetGuAny(instance.GuID)
	}
	if !ok {
		return SpecialMechanic{}, false
	}
	for _, mechanic := range spec.Effects.Other {
		if mechanic.ActionCode == actionCode {
			return mechanic, true
		}
	}
	return SpecialMechanic{}, false
}

func InventoryHasGuMechanic(inventory GuInventory, actionCode string, requireEquipped bool) bool {
	for _, instance := range inventory {
		if instance.Status == "DEAD" || (requireEquipped && (!instance.Equipped || instance.Status != "ACTIVE")) {
			continue
		}
		if _, ok := GuMechanic(instance, actionCode); ok {
			return true
		}
		if actionCode == "DISGUISE" {
			spec, ok := GetGu(instance.GuID, instance.Rank)
			if !ok {
				spec, ok = GetGuAny(instance.GuID)
			}
			if ok && spec.Effects.Passive != nil && spec.Effects.Passive.DisguiseLevel > 0 {
				return true
			}
		}
	}
	return false
}

func GuMechanicFloat(instance GuInstance, actionCode, key string) float64 {
	mechanic, ok := GuMechanic(instance, actionCode)
	if !ok {
		return 0
	}
	var params map[string]float64
	if json.Unmarshal(mechanic.Params, &params) != nil {
		return 0
	}
	return params[key]
}

type PlayerGu struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	PlayerID  string    `json:"player_id" gorm:"index;not null"`
	GuID      string    `json:"gu_id" gorm:"index;not null"`
	Rank      int       `json:"rank"`
	Level     int       `json:"level"`
	Hunger    float64   `json:"hunger"`
	State     string    `json:"state"`
	Location  string    `json:"location"`
	IsLocked  bool      `json:"is_locked"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (g *GuEffects) Scan(value interface{}) error {
	if value == nil {
		*g = GuEffects{}
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("unsupported Scan, expected []byte or string, got %T", value)
	}

	normalized, _, err := effectcatalog.NormalizeAndValidate(b)
	if err != nil {
		return fmt.Errorf("invalid Gu effects: %w", err)
	}
	if err := json.Unmarshal(normalized, g); err != nil {
		return err
	}
	if g.Active != nil {
		g.Active.normalize()
	}
	if g.Other == nil {
		g.Other = []SpecialMechanic{}
	}
	return nil
}

// normalize converts legacy shorthand status keys (freeze:1) into apply_status
// entries. Deprecated for authoring (CSV now writes apply_status directly);
// kept so existing database rows loaded before re-import keep working.
func (a *ActiveEffect) normalize() {
	if len(a.ApplyStatus) > 0 {
		return
	}
	addDuration := func(id string, shorthand float64) {
		a.ApplyStatus = append(a.ApplyStatus, StatusApplication{ID: id, Turns: int(shorthand)})
	}
	addTick := func(id string, shorthand float64) {
		a.ApplyStatus = append(a.ApplyStatus, StatusApplication{ID: id, Value: shorthand, Turns: int(shorthand)})
	}
	if a.Freeze > 0 {
		addDuration("FREEZE", a.Freeze)
		a.Freeze = 0
	}
	if a.Slow > 0 {
		addTick("SLOW", a.Slow)
		a.Slow = 0
	}
	if a.Slow1 > 0 {
		addTick("SLOW1", a.Slow1)
		a.Slow1 = 0
	}
	if a.Weak > 0 {
		addTick("WEAK", a.Weak)
		a.Weak = 0
	}
	if a.Stun > 0 {
		addDuration("STUN", a.Stun)
		a.Stun = 0
	}
	if a.Silence > 0 {
		addDuration("SILENCE", a.Silence)
		a.Silence = 0
	}
	if a.Bleed > 0 {
		addTick("BLEED", a.Bleed)
		a.Bleed = 0
	}
	if a.Poison > 0 {
		addTick("POISON", a.Poison)
		a.Poison = 0
	}
	if a.Thorns > 0 {
		addDuration("THORNS", a.Thorns)
		a.Thorns = 0
	}
	if a.Regen > 0 {
		addDuration("REGEN", a.Regen)
		a.Regen = 0
	}
	if a.AtkUp > 0 {
		addDuration("ATK_UP", a.AtkUp)
		a.AtkUp = 0
	}
	if a.DefUp > 0 {
		addDuration("DEF_UP", a.DefUp)
		a.DefUp = 0
	}
	if a.Taunt > 0 {
		addDuration("TAUNT", a.Taunt)
		a.Taunt = 0
	}
}

func (g GuEffects) Value() (driver.Value, error) {
	b, err := json.Marshal(g)
	if err != nil {
		return "{}", err
	}
	return string(b), nil
}

func GetFlatGuEffects(gu Gu) map[string]float64 {
	res := make(map[string]float64)
	if gu.Effects.Active != nil {
		if gu.Effects.Active.DamageFlat > 0 {
			res["damage"] = gu.Effects.Active.DamageFlat
		}
		if gu.Effects.Active.ShieldFlat > 0 {
			res["defense"] = gu.Effects.Active.ShieldFlat
		}
		if gu.Effects.Active.DamageMult > 0 {
			res["damage_multiplier"] = gu.Effects.Active.DamageMult
		}
		if gu.Effects.Active.ShieldMult > 0 {
			res["shield_multiplier"] = gu.Effects.Active.ShieldMult
		}
		// Apply_status entries → legacy flat keys (consumed by killer-move
		// compounding checks: damage+bleed, damage+stun).
		for _, entry := range gu.Effects.Active.ApplyStatus {
			value := gu.Effects.Active.effectiveStatusValue(entry)
			turns := gu.Effects.Active.effectiveStatusTurns(entry)
			switch entry.ID {
			case "BLEED":
				res["bleed"] = value
			case "POISON":
				res["poison"] = value
			case "STUN":
				res["stun"] = float64(turns)
			case "SLOW":
				res["slow"] = value
			case "SLOW1":
				res["slow1"] = value
			case "WEAK":
				res["weak"] = value
			case "FREEZE":
				res["freeze"] = float64(turns)
			case "SILENCE":
				res["silence"] = float64(turns)
			case "THORNS":
				res["thorns_buff"] = value
			case "REGEN":
				res["regen"] = value
			case "ATK_UP":
				res["atk_up"] = value
			case "DEF_UP":
				res["def_up"] = value
			case "TAUNT":
				res["taunt"] = float64(turns)
			}
		}
		if gu.Effects.Active.LifeStealRate > 0 {
			res["lifesteal"] = gu.Effects.Active.LifeStealRate
		}
		if gu.Effects.Active.HealFlat > 0 {
			res["heal"] = gu.Effects.Active.HealFlat
		}
		if gu.Effects.Active.BloodDrain {
			res["blood_drain"] = 1.0
		}
		if gu.Effects.Active.PortalCreate {
			res["portal_create"] = 1.0
		}
		if gu.Effects.Active.RandomPortal {
			res["random_portal"] = 1.0
		}
		if gu.Effects.Active.IgnoreDefense > 0 {
			res["ignore_defense"] = gu.Effects.Active.IgnoreDefense
		}
		if gu.Effects.Active.IgnoreDefensePct > 0 {
			res["ignore_defense_pct"] = gu.Effects.Active.IgnoreDefensePct
		}
		if gu.Effects.Active.CritRateBoost > 0 {
			res["crit_rate_boost"] = gu.Effects.Active.CritRateBoost
		}
		if gu.Effects.Active.AllStatsBuffPct > 0 {
			res["all_stats_buff_percent"] = gu.Effects.Active.AllStatsBuffPct
		}
		if gu.Effects.Active.SpeedBoostFlat > 0 {
			res["speed_boost_flat"] = gu.Effects.Active.SpeedBoostFlat
		}
		if gu.Effects.Active.CaptureBonus > 0 {
			res["capture_bonus"] = gu.Effects.Active.CaptureBonus
		}
		res["channel_time"] = 0 // default
	}
	if gu.Effects.Passive != nil {
		if gu.Effects.Passive.StatBaseAttack > 0 {
			res["flat_damage"] = gu.Effects.Passive.StatBaseAttack
		}
		if gu.Effects.Passive.StatBaseDefense > 0 {
			res["flat_defense"] = gu.Effects.Passive.StatBaseDefense
		}
		if gu.Effects.Passive.Investigation > 0 {
			res["success_rate_bonus"] = gu.Effects.Passive.Investigation
		}
		if gu.Effects.Passive.PathConflictReduction > 0 {
			res["path_conflict_reduction"] = gu.Effects.Passive.PathConflictReduction
		}
		if gu.Effects.Passive.Accuracy > 0 {
			res["accuracy"] = gu.Effects.Passive.Accuracy
		}
		if gu.Effects.Passive.Tenacity > 0 {
			res["tenacity"] = gu.Effects.Passive.Tenacity
		}
		if gu.Effects.Passive.CCResist > 0 {
			res["cc_resist"] = gu.Effects.Passive.CCResist
		}
	}
	return res
}

func rawEffectNumber(raw json.RawMessage) (float64, bool) {
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil {
		return number, true
	}
	var flag bool
	if err := json.Unmarshal(raw, &flag); err == nil {
		if flag {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

var GuRegistry = make(map[string]map[int]GuDictionaryItem)

// GuIDsByName returns all gu ids whose name or ID contains the given search term (case/accent-insensitive).
// Exact matches are returned first, followed by prefix and substring matches.
func GuIDsByName(search string) []string {
	if search == "" {
		return nil
	}
	cleanSearch := RemoveAccents(strings.TrimSpace(search))
	if cleanSearch == "" {
		return nil
	}
	var exactMatches []string
	var prefixMatches []string
	var substringMatches []string

	for id, ranks := range GuRegistry {
		for _, item := range ranks {
			cleanName := RemoveAccents(item.Name)
			cleanID := strings.ToLower(id)
			if cleanName == cleanSearch || cleanID == cleanSearch {
				exactMatches = append(exactMatches, id)
				break
			} else if strings.HasPrefix(cleanName, cleanSearch) || strings.HasPrefix(cleanID, cleanSearch) {
				prefixMatches = append(prefixMatches, id)
				break
			} else if strings.Contains(cleanName, cleanSearch) || strings.Contains(cleanID, cleanSearch) {
				substringMatches = append(substringMatches, id)
				break
			}
		}
	}

	result := append(exactMatches, prefixMatches...)
	result = append(result, substringMatches...)
	return result
}

func GetGu(id string, rank int) (GuDictionaryItem, bool) {
	ranks, ok := GuRegistry[id]
	if !ok {
		for k, v := range GuRegistry {
			if strings.EqualFold(k, id) {
				ranks = v
				ok = true
				break
			}
		}
	}
	if !ok {
		return GuDictionaryItem{}, false
	}
	item, found := ranks[rank]
	return item, found
}

func GetGuAny(id string) (GuDictionaryItem, bool) {
	ranks, ok := GuRegistry[id]
	if !ok {
		for k, v := range GuRegistry {
			if strings.EqualFold(k, id) {
				ranks = v
				ok = true
				break
			}
		}
	}
	if !ok {
		return GuDictionaryItem{}, false
	}
	var best GuDictionaryItem
	found := false
	min := 0
	for r, item := range ranks {
		if !found || r < min {
			min = r
			best = item
			found = true
		}
	}
	if !found {
		return GuDictionaryItem{}, false
	}
	return best, true
}

// FindGuByNameOrID resolves a Gu item from the dictionary by checking:
// 1. Exact Name match (case/accent-insensitive)
// 2. Exact ID match (case-insensitive)
// 3. Prefix Name / ID match
// 4. Substring Name / ID match
// Filters by targetRank if targetRank > 0.
func FindGuByNameOrID(query string, targetRank int) (GuDictionaryItem, bool) {
	query = strings.TrimSpace(query)
	if query == "" {
		return GuDictionaryItem{}, false
	}
	cleanQuery := RemoveAccents(query)

	// Phase 1: Exact Name match
	for _, ranks := range GuRegistry {
		for rank, item := range ranks {
			if targetRank > 0 && rank != targetRank {
				continue
			}
			if RemoveAccents(item.Name) == cleanQuery {
				return item, true
			}
		}
	}

	// Phase 2: Direct ID lookup
	if targetRank > 0 {
		if item, found := GetGu(query, targetRank); found {
			return item, true
		}
	}
	if item, found := GetGuAny(query); found {
		if targetRank == 0 || item.Rank == targetRank {
			return item, true
		}
	}

	// Phase 3: Prefix Name or ID match
	for id, ranks := range GuRegistry {
		for rank, item := range ranks {
			if targetRank > 0 && rank != targetRank {
				continue
			}
			cleanName := RemoveAccents(item.Name)
			cleanID := strings.ToLower(id)
			if strings.HasPrefix(cleanName, cleanQuery) || strings.HasPrefix(cleanID, cleanQuery) {
				return item, true
			}
		}
	}

	// Phase 4: Substring Name or ID match
	for id, ranks := range GuRegistry {
		for rank, item := range ranks {
			if targetRank > 0 && rank != targetRank {
				continue
			}
			cleanName := RemoveAccents(item.Name)
			cleanID := strings.ToLower(id)
			if strings.Contains(cleanName, cleanQuery) || strings.Contains(cleanID, cleanQuery) {
				return item, true
			}
		}
	}

	return GuDictionaryItem{}, false
}

// HasImmortalRank returns true if any rank variant of the given Gu ID is >= 6 (Tiên Cổ).
func HasImmortalRank(id string) bool {
	ranks, ok := GuRegistry[id]
	if !ok {
		return false
	}
	for r := range ranks {
		if r >= 6 {
			return true
		}
	}
	return false
}

func LoadGuRegistry(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT id, name, rank, path, gu_type, base_mana_cost, effects, food_types, hunger_rate, max_hunger, description, target_type, COALESCE(rarity, 0), is_immortal, primary_role, combat_roles, combat_tags, support_tags, COALESCE(base_price, 0), COALESCE(image_url, '')
		FROM gu_dictionary
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	newRegistry := make(map[string]map[int]GuDictionaryItem)
	count := 0

	for rows.Next() {
		var item GuDictionaryItem
		var pathRaw, foodTypesRaw, combatRolesRaw, combatTagsRaw, supportTagsRaw []byte
		var targetTypeRaw []byte
		var isImmortalRaw bool
		err = rows.Scan(
			&item.ID, &item.Name, &item.Rank, &pathRaw,
			&item.Type, &item.BaseManaCost,
			&item.Effects, &foodTypesRaw, &item.HungerRate,
			&item.MaxHunger, &item.Description,
			&targetTypeRaw, &item.Rarity, &isImmortalRaw,
			&item.PrimaryRole,
			&combatRolesRaw, &combatTagsRaw, &supportTagsRaw, &item.BasePrice,
			&item.ImageURL,
		)
		if err != nil {
			return fmt.Errorf("failed to scan Gu spec row: %w", err)
		}

		if err := json.Unmarshal(pathRaw, &item.Path); err != nil {
			item.Path = []string{string(pathRaw)}
		}
		_ = json.Unmarshal(foodTypesRaw, &item.FoodTypes)
		_ = json.Unmarshal(combatRolesRaw, &item.CombatRoles)
		_ = json.Unmarshal(combatTagsRaw, &item.CombatTags)
		_ = json.Unmarshal(supportTagsRaw, &item.SupportTags)

		item.IsImmortal = isImmortalRaw

		if _, ok := newRegistry[item.ID]; !ok {
			newRegistry[item.ID] = make(map[int]GuDictionaryItem)
		}
		newRegistry[item.ID][item.Rank] = item
		count++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate Gu registry rows: %w", err)
	}

	GuRegistry = newRegistry
	logger.S.Infof("Loaded %d Gu bugs from database registry into memory cache", count)
	_ = cache.Delete("game_dictionary_unified", "game_dictionary_gus", "game_dictionary_items", "game_dictionary_monsters", "game_dictionary_constitutions", "game_dictionary_races", "game_dictionary_loot_tables", "game_dictionary_regions", "game_dictionary_monster_spawns")

	allPathsCache = nil
	logger.S.Infof("Built all-paths cache: %d unique dao paths", len(AllGuPaths()))
	return nil
}

var allPathsCache []string

func AllGuPaths() []string {
	if allPathsCache != nil {
		return allPathsCache
	}
	pathSet := make(map[string]struct{})
	for _, ranks := range GuRegistry {
		for _, item := range ranks {
			for _, p := range item.Path {
				normalized := strings.TrimSpace(strings.TrimSuffix(strings.ToLower(p), " path"))
				if normalized != "" {
					pathSet[normalized] = struct{}{}
				}
			}
		}
	}
	allPathsCache = make([]string, 0, len(pathSet))
	for p := range pathSet {
		allPathsCache = append(allPathsCache, p)
	}
	return allPathsCache
}

// ManaRegenScale is the per-rank "cost" used to scale MANA_REGEN_TICK so a Gu
// gives its full value only when its rank matches the player's, and less when
// the player outranks it. Index 0 = rank 1.
var ManaRegenScale = []float64{1, 2, 5, 20, 100, 10000, 1000000, 100000000, 10000000000}

// EquippedTimeSpeedBonus sums the passive time_speed_multiplier contributed by
// every equipped ACTIVE Gu. The bonus is additive to the player's base
// time_speed_multiplier and applies to all time flows (mana/hp/soul/thought
// regen, lifespan decay, aperture internal years, Gu growth).
func EquippedTimeSpeedBonus(inventory []GuInstance) float64 {
	var total float64
	for _, gu := range inventory {
		if !gu.Equipped || gu.Status != "ACTIVE" {
			continue
		}
		spec, ok := GetGu(gu.GuID, gu.Rank)
		if !ok || spec.Effects.Passive == nil {
			continue
		}
		total += spec.Effects.Passive.TimeSpeedMultiplier
	}
	return total
}

// ManaRegenTickFactor returns the multiplier applied to a MANA_REGEN_TICK
// value: 1.0 when player and Gu ranks match, otherwise cost[guRank]/cost[playerRank].
func ManaRegenTickFactor(playerRank, guRank int) float64 {
	if playerRank == guRank {
		return 1.0
	}
	if playerRank < 1 || playerRank > len(ManaRegenScale) || guRank < 1 || guRank > len(ManaRegenScale) {
		return 1.0
	}
	return ManaRegenScale[guRank-1] / ManaRegenScale[playerRank-1]
}

// resourceRegenTickPerHour sums the per-hour contribution of every equipped
// ACTIVE Gu carrying an ON_TICK action in the given set. It is the shared
// engine behind mana/thought/soul regen; defaultValue applies when a mechanic
// omits params.value.
func resourceRegenTickPerHour(inventory []GuInstance, playerRank int, defaultValue float64, actions ...string) float64 {
	wanted := make(map[string]bool, len(actions))
	for _, action := range actions {
		wanted[action] = true
	}
	var total float64
	for _, gu := range inventory {
		if !gu.Equipped || gu.Status != "ACTIVE" {
			continue
		}
		spec, ok := GetGu(gu.GuID, gu.Rank)
		if !ok {
			continue
		}
		for _, mech := range spec.Effects.Other {
			if mech.EventTrigger != "ON_TICK" || !wanted[mech.ActionCode] {
				continue
			}
			var params map[string]interface{}
			if len(mech.Params) > 0 {
				json.Unmarshal(mech.Params, &params)
			}
			val, _ := params["value"].(float64)
			if val <= 0 {
				val = defaultValue
			}
			total += val * ManaRegenTickFactor(playerRank, gu.Rank)
		}
	}
	return total
}

// ManaRegenTickPerHour sums the per-hour mana contribution of every equipped
// ACTIVE Gu that carries an ON_TICK MANA_REGEN_TICK effect.
func ManaRegenTickPerHour(inventory []GuInstance, playerRank int) float64 {
	return resourceRegenTickPerHour(inventory, playerRank, 0.5, "MANA_REGEN_TICK")
}

// ThoughtRegenTickPerHour sums the per-hour thought (Niệm Lực) contribution of
// every equipped ACTIVE Gu that carries an ON_TICK THOUGHT_REGEN_TICK effect.
// It reuses the ManaRegenTickFactor rank scaling so a Gu gives its full value
// only when its rank matches the player's.
func ThoughtRegenTickPerHour(inventory []GuInstance, playerRank int) float64 {
	return resourceRegenTickPerHour(inventory, playerRank, 1.0, "THOUGHT_REGEN_TICK")
}

// SoulRegenTickPerHour sums the per-hour soul_hp contribution of every equipped
// ACTIVE Gu that carries an ON_TICK SOUL_REGEN_TICK effect. It reuses the
// ManaRegenTickFactor rank scaling so a Gu gives its full value only when its
// rank matches the player's.
func SoulRegenTickPerHour(inventory []GuInstance, playerRank int) float64 {
	return resourceRegenTickPerHour(inventory, playerRank, 1.0, "SOUL_REGEN_TICK")
}
