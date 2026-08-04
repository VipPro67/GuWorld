package models

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

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

	SummonGrowthPhantoms bool    `json:"summon_growth_phantoms,omitempty"`
	SummonCountPerTick   float64 `json:"summon_count_per_tick,omitempty"`
	PhantomDamageScale   float64 `json:"phantom_damage_scale,omitempty"`
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

	// Khởi tạo Other là mảng rỗng thay vì nil để dễ dùng
	if err := json.Unmarshal(b, g); err != nil {
		return err
	}
	if g.Other == nil {
		g.Other = []SpecialMechanic{}
	}
	return nil
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
	return nil
}

// GetThienNguyenMultiplier returns the multiplier for Thien Nguyen Bao Lien series effects.
// Same rank = x2. Lower player rank = higher boost (high-rank gu helps low-rank cultivator more).
// Formula follows energy cost logic: multiplier = 2.0 * costFactor(guRank) / costFactor(playerRank).
// Immortal gu (6+) blocked for mortals (<6) in EquipGu already.
func GetThienNguyenMultiplier(playerRank, guRank int) float64 {
	if playerRank == guRank {
		return 2.0
	}
	immortalCost := func(diff int) float64 {
		return math.Pow(100, float64(diff))
	}
	mortalCost := map[int]float64{1: 1, 2: 2, 3: 5, 4: 20, 5: 100}
	if playerRank >= 6 && guRank >= 6 {
		diff := playerRank - guRank
		if diff > 0 {
			return 2.0 / immortalCost(diff)
		}
		return 2.0 * immortalCost(-diff)
	}
	if playerRank >= 6 {
		return 2.0 / immortalCost(playerRank-guRank)
	}
	if guRank >= 6 {
		return 0
	}
	playerCost := mortalCost[playerRank]
	guCost := mortalCost[guRank]
	if playerCost == 0 || guCost == 0 {
		return 0
	}
	return 2.0 * guCost / playerCost
}

// GetPlayerManaRegenTickBonus calculates the total mana regeneration bonus from equipped active Gus with MANA_REGEN_TICK.
func GetPlayerManaRegenTickBonus(playerRank int, inventory []GuInstance) float64 {
	var totalBonus float64
	for _, gu := range inventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			spec, found := GetGuAny(gu.GuID)
			if found && spec.Effects.Other != nil {
				tnMultiplier := 1.0
				if strings.HasPrefix(spec.ID, "GU_HEAVENLY_ESSENCE_LOTUS") {
					tnMultiplier = GetThienNguyenMultiplier(playerRank, spec.Rank)
				}
				for _, mech := range spec.Effects.Other {
					if mech.EventTrigger == "ON_TICK" && mech.ActionCode == "MANA_REGEN_TICK" {
						var params map[string]interface{}
						if len(mech.Params) > 0 {
							_ = json.Unmarshal(mech.Params, &params)
						}
						val, _ := params["value"].(float64)
						if val <= 0 {
							val = 0.5
						}
						totalBonus += val * tnMultiplier
					}
				}
			}
		}
	}
	return totalBonus
}
