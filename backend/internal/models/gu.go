package models

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type ActiveEffect struct {
	DamageFlat          float64 `json:"damage_flat,omitempty"`
	DamagePath          string  `json:"damage_path,omitempty"`
	HealFlat            float64 `json:"heal_flat,omitempty"`
	ShieldFlat          float64 `json:"shield_flat,omitempty"`
	ApplyStatus         string  `json:"apply_status,omitempty"`
	StatusTickDamage    float64 `json:"status_tick_damage,omitempty"`
	StatusDurationTicks int     `json:"status_duration_ticks,omitempty"`
	TargetSelector      string  `json:"target_selector,omitempty"`
	TargetCount         int     `json:"target_count,omitempty"`
	SummonMonsterID     string  `json:"summon_monster_id,omitempty"`
	SummonPosition      string  `json:"summon_position,omitempty"`
	InheritStatsPercent float64 `json:"inherit_stats_percent,omitempty"`

	Range              float64 `json:"range,omitempty"`
	CurveAttack        bool    `json:"curve_attack,omitempty"`
	ManaRegen          float64 `json:"mana_regen,omitempty"`
	WoodTransformation bool    `json:"wood_transformation,omitempty"`
	BloodDrain         bool    `json:"blood_drain,omitempty"`
	Replicate          bool    `json:"replicate,omitempty"`
	LifeStealRate      float64 `json:"lifesteal_rate,omitempty"`
	RefineManaTier     float64 `json:"refine_mana_tier,omitempty"`
	DamageMult         float64 `json:"damage_mult,omitempty"`
	CleansingPower     float64 `json:"cleansing_power,omitempty"`
	SummonPhantom      bool    `json:"summon_phantom,omitempty"`
	GenerateGu         bool    `json:"generate_gu,omitempty"`
	CaptureBonus       float64 `json:"capture_bonus,omitempty"` // +% tỉ lệ bắt monster (0..1)

	SummonGrowthPhantoms bool    `json:"summon_growth_phantoms,omitempty"`
	SummonCountPerTick   float64 `json:"summon_count_per_tick,omitempty"`
	PhantomDamageScale   float64 `json:"phantom_damage_scale,omitempty"`

	// Shorthand status effect fields — tier value (e.g. "freeze": 1 = speed T1)
	// Normalized to ApplyStatus + StatusDurationTicks during loading.
	Freeze  float64 `json:"freeze,omitempty"`
	Slow    float64 `json:"slow,omitempty"`
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
	ManaRegenMult         float64 `json:"mana_regen_mult,omitempty"`
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
	ControlPower          float64 `json:"control_power,omitempty"`
	SuccessRateBonus      float64 `json:"success_rate_bonus,omitempty"`
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
	Rarity       string    `json:"rarity"`
	IsImmortal   bool      `json:"is_immortal"`
	PrimaryRole  string    `json:"primary_role"`
	CombatRoles  []string  `json:"combat_roles"`
	CombatTags   []string  `json:"combat_tags"`
	SupportTags  []string  `json:"support_tags"`
	BasePrice    int       `json:"base_price"`
	ImageURL     string    `json:"image_url"`
}

type Gu = GuDictionaryItem

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
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte, got %T", value)
	}

	if err := json.Unmarshal(b, g); err != nil {
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

// normalize converts shorthand status keys (freeze:1) to ApplyStatus format.
func (a *ActiveEffect) normalize() {
	if a.Freeze > 0 && a.ApplyStatus == "" {
		a.ApplyStatus = "FREEZE"
		a.StatusDurationTicks = int(a.Freeze)
		a.Freeze = 0
	}
	if a.Slow > 0 && a.ApplyStatus == "" {
		a.ApplyStatus = "SLOW"
		a.StatusTickDamage = a.Slow
		a.StatusDurationTicks = int(a.Slow)
		a.Slow = 0
	}
	if a.Weak > 0 && a.ApplyStatus == "" {
		a.ApplyStatus = "WEAK"
		a.StatusTickDamage = a.Weak
		a.StatusDurationTicks = int(a.Weak)
		a.Weak = 0
	}
	if a.Stun > 0 && a.ApplyStatus == "" {
		a.ApplyStatus = "STUN"
		a.StatusDurationTicks = int(a.Stun)
		a.Stun = 0
	}
	if a.Silence > 0 && a.ApplyStatus == "" {
		a.ApplyStatus = "SILENCE"
		a.StatusDurationTicks = int(a.Silence)
		a.Silence = 0
	}
	if a.Bleed > 0 && a.ApplyStatus == "" {
		a.ApplyStatus = "BLEED"
		a.StatusTickDamage = a.Bleed
		a.StatusDurationTicks = int(a.Bleed)
		a.Bleed = 0
	}
	if a.Poison > 0 && a.ApplyStatus == "" {
		a.ApplyStatus = "POISON"
		a.StatusTickDamage = a.Poison
		a.StatusDurationTicks = int(a.Poison)
		a.Poison = 0
	}
	if a.Thorns > 0 && a.ApplyStatus == "" {
		a.ApplyStatus = "THORNS"
		a.StatusDurationTicks = int(a.Thorns)
		a.Thorns = 0
	}
	if a.Regen > 0 && a.ApplyStatus == "" {
		a.ApplyStatus = "REGEN"
		a.StatusDurationTicks = int(a.Regen)
		a.Regen = 0
	}
	if a.AtkUp > 0 && a.ApplyStatus == "" {
		a.ApplyStatus = "ATK_UP"
		a.StatusDurationTicks = int(a.AtkUp)
		a.AtkUp = 0
	}
	if a.DefUp > 0 && a.ApplyStatus == "" {
		a.ApplyStatus = "DEF_UP"
		a.StatusDurationTicks = int(a.DefUp)
		a.DefUp = 0
	}
	if a.Taunt > 0 && a.ApplyStatus == "" {
		a.ApplyStatus = "TAUNT"
		a.StatusDurationTicks = int(a.Taunt)
		a.Taunt = 0
	}
}

func (g GuEffects) Value() (driver.Value, error) {
	return json.Marshal(g)
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
		if gu.Effects.Active.ApplyStatus == "BLEED" {
			res["bleed"] = gu.Effects.Active.StatusTickDamage
		}
		if gu.Effects.Active.ApplyStatus == "POISON" {
			res["poison"] = gu.Effects.Active.StatusTickDamage
		}
		if gu.Effects.Active.ApplyStatus == "STUN" {
			res["stun"] = float64(gu.Effects.Active.StatusDurationTicks)
		}
		if gu.Effects.Active.ApplyStatus == "SLOW" {
			res["slow"] = gu.Effects.Active.StatusTickDamage
		}
		if gu.Effects.Active.ApplyStatus == "WEAK" {
			res["weak"] = gu.Effects.Active.StatusTickDamage
		}
		if gu.Effects.Active.ApplyStatus == "FREEZE" {
			res["freeze"] = float64(gu.Effects.Active.StatusDurationTicks)
		}
		if gu.Effects.Active.ApplyStatus == "SILENCE" {
			res["silence"] = float64(gu.Effects.Active.StatusDurationTicks)
		}
		if gu.Effects.Active.ApplyStatus == "THORNS" {
			res["thorns_buff"] = gu.Effects.Active.StatusTickDamage
		}
		if gu.Effects.Active.ApplyStatus == "REGEN" {
			res["regen"] = gu.Effects.Active.StatusTickDamage
		}
		if gu.Effects.Active.ApplyStatus == "ATK_UP" {
			res["atk_up"] = gu.Effects.Active.StatusTickDamage
		}
		if gu.Effects.Active.ApplyStatus == "DEF_UP" {
			res["def_up"] = gu.Effects.Active.StatusTickDamage
		}
		if gu.Effects.Active.ApplyStatus == "TAUNT" {
			res["taunt"] = float64(gu.Effects.Active.StatusDurationTicks)
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
		if gu.Effects.Active.CaptureBonus > 0 {
			res["capture_bonus"] = gu.Effects.Active.CaptureBonus
		}
		if gu.Effects.Passive.PathConflictReduction > 0 {
			res["path_conflict_reduction"] = gu.Effects.Passive.PathConflictReduction
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
func GuIDsByName(search string) []string {
	if search == "" {
		return nil
	}
	cleanSearch := RemoveAccents(search)
	var ids []string
	for id, ranks := range GuRegistry {
		for _, item := range ranks {
			if strings.Contains(RemoveAccents(item.Name), cleanSearch) || strings.Contains(strings.ToLower(id), cleanSearch) {
				ids = append(ids, id)
				break
			}
		}
	}
	return ids
}

func GetGu(id string, rank int) (GuDictionaryItem, bool) {
	ranks, ok := GuRegistry[id]
	if !ok {
		return GuDictionaryItem{}, false
	}
	item, found := ranks[rank]
	return item, found
}

func GetGuAny(id string) (GuDictionaryItem, bool) {
	ranks, ok := GuRegistry[id]
	if !ok {
		return GuDictionaryItem{}, false
	}
	var best GuDictionaryItem
	max := 0
	for r, item := range ranks {
		if r > max {
			max = r
			best = item
		}
	}
	if max == 0 {
		return GuDictionaryItem{}, false
	}
	return best, true
}

func LoadGuRegistry(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT id, name, rank, path, gu_type, base_mana_cost, effects, food_types, hunger_rate, max_hunger, description, target_type, rarity, is_immortal, primary_role, combat_roles, combat_tags, support_tags, COALESCE(base_price, 0), COALESCE(image_url, '')
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
		var targetTypeRaw, rarityRaw []byte
		var isImmortalRaw bool
		err = rows.Scan(
			&item.ID, &item.Name, &item.Rank, &pathRaw,
			&item.Type, &item.BaseManaCost,
			&item.Effects, &foodTypesRaw, &item.HungerRate,
			&item.MaxHunger, &item.Description,
			&targetTypeRaw, &rarityRaw, &isImmortalRaw,
			&item.PrimaryRole,
			&combatRolesRaw, &combatTagsRaw, &supportTagsRaw, &item.BasePrice,
			&item.ImageURL,
		)
		if err != nil {
			logger.S.Warnf("failed to scan Gu spec row: %v", err)
			continue
		}

		if err := json.Unmarshal(pathRaw, &item.Path); err != nil {
			item.Path = []string{string(pathRaw)}
		}
		_ = json.Unmarshal(foodTypesRaw, &item.FoodTypes)
		_ = json.Unmarshal(combatRolesRaw, &item.CombatRoles)
		_ = json.Unmarshal(combatTagsRaw, &item.CombatTags)
		_ = json.Unmarshal(supportTagsRaw, &item.SupportTags)

		if len(rarityRaw) > 0 {
			_ = json.Unmarshal(rarityRaw, &item.Rarity)
			if item.Rarity == "" {
				item.Rarity = string(rarityRaw)
			}
		}
		item.IsImmortal = isImmortalRaw

		if _, ok := newRegistry[item.ID]; !ok {
			newRegistry[item.ID] = make(map[int]GuDictionaryItem)
		}
		newRegistry[item.ID][item.Rank] = item
		count++
	}

	GuRegistry = newRegistry
	logger.S.Infof("Loaded %d Gu bugs from database registry into memory cache", count)
	_ = cache.Delete("game_dictionary_unified")

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

