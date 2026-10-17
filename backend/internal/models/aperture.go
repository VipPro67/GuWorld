package models

import (
	"math"
	"time"
)

type ImmortalAperture struct {
	ID                       string      `json:"id"`
	PlayerID                 string      `json:"player_id"`
	Grade                    string      `json:"grade"` // 'LOW' | 'MID' | 'HIGH' | 'EXTREME'
	TimeRate                 float64     `json:"time_rate"`
	ApertureHP               float64     `json:"aperture_hp"`
	MaxApertureHP            float64     `json:"max_aperture_hp"`
	AreaSize                 int64       `json:"area_size"`
	ImmortalEssence          int         `json:"immortal_essence"`
	EssenceProductionPerYear int         `json:"essence_production_per_year"`
	InternalYearsPassed      float64     `json:"internal_years_passed"`
	NextTribulationYear      float64     `json:"next_tribulation_year"`
	NextTribulationAt        *time.Time  `json:"next_tribulation_at,omitempty"`
	TribulationCount         int         `json:"tribulation_count"`
	RankStartYear            float64     `json:"rank_start_year"`
	EarthCalamityCount       int         `json:"earth_calamity_count"`
	HeavenlyTribulationCount int         `json:"heavenly_tribulation_count"`
	GrandTribulationCount    int         `json:"grand_tribulation_count"`
	MyriadTribulationCount   int         `json:"myriad_tribulation_count"`
	ChaosCalamityCount       int         `json:"chaos_calamity_count"`
	State                    string      `json:"state"` // 'STABLE' | 'TRIBULATION_PENDING' | 'TRIBULATION_ACTIVE' | 'DAMAGED' | 'DESTROYED'
	DaoMarks                 DaoMarksMap `json:"dao_marks"`
	CreatedAt                time.Time   `json:"created_at"`
	UpdatedAt                time.Time   `json:"updated_at"`
	Temperature              float64     `json:"temperature,omitempty"`
	Humidity                 float64     `json:"humidity,omitempty"`
	Qi                       float64     `json:"qi,omitempty"`
	Terrain                  string      `json:"terrain,omitempty"`
}

func (a ImmortalAperture) TotalTribulationCount() int {
	return a.EarthCalamityCount + a.HeavenlyTribulationCount + a.GrandTribulationCount + a.MyriadTribulationCount + a.ChaosCalamityCount
}

type DaoMark struct {
	ID        string    `json:"id"`
	PlayerID  string    `json:"player_id"`
	Path      string    `json:"path"`
	Amount    int64     `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ApertureResourceNode struct {
	ID                      string             `json:"id"`
	ApertureID              string             `json:"aperture_id"`
	NodeType                string             `json:"node_type"`
	Name                    string             `json:"name"`
	Path                    string             `json:"path"`
	Level                   int                `json:"level"`
	HP                      int                `json:"hp"`
	MaxHP                   int                `json:"max_hp"`
	X                       int                `json:"x"`
	Y                       int                `json:"y"`
	ProductionType          string             `json:"production_type"`
	ProductionAmount        int64              `json:"production_amount"`
	ProductionIntervalHours int                `json:"production_interval_hours"`
	LastCollectedAt         *time.Time         `json:"last_collected_at"`
	IsDestroyed             bool               `json:"is_destroyed"`
	CreatedAt               time.Time          `json:"created_at"`
	UpdatedAt               time.Time          `json:"updated_at"`
	NodeCategory            string             `json:"node_category"`
	DefensePower            int                `json:"defense_power"`
	AttackRange             int                `json:"attack_range"`
	HPBonus                 int                `json:"hp_bonus"`
	DamageReduction         int                `json:"damage_reduction"`
	BlocksMovement          bool               `json:"blocks_movement"`
	RequiredGuID            string             `json:"required_gu_id"`
	RequiredGuRank          int                `json:"required_gu_rank"`
	ZoneType                string             `json:"zone_type"`
	FoodPath                string             `json:"food_path"`
	ItemTier                string             `json:"item_tier"`
	ProductionKey           string             `json:"production_key"`
	Cells                   []ApertureGridCell `json:"cells,omitempty"`
	DaoConflictMultiplier   float64            `json:"dao_conflict_multiplier,omitempty"`
	YieldPreview            *ApertureYieldPreview `json:"yield_preview,omitempty"`
}

// ApertureYieldFactor is one multiplier of the harvest breakdown (gamefx
// ComputeApertureYield). Keys: level, dao, area, conflict, ecology, eco, labor, pop.
type ApertureYieldFactor struct {
	Key   string  `json:"key"`
	Value float64 `json:"value"`
}

// ApertureYieldPreview mirrors gamefx.ComputeApertureYield so the client never
// has to re-derive the harvest formula.
type ApertureYieldPreview struct {
	Base      float64               `json:"base"`
	Expected  float64               `json:"expected"`
	Whole     int64                 `json:"whole"`
	Remainder float64               `json:"remainder"`
	Bonus     float64               `json:"bonus"`
	RawBonus  float64               `json:"raw_bonus"`
	Capped    bool                  `json:"capped"`
	PerHour   float64               `json:"per_hour"`
	Factors   []ApertureYieldFactor `json:"factors"`
}

type ApertureGridCell struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type TribulationEvent struct {
	ID               string     `json:"id"`
	ApertureID       string     `json:"aperture_id"`
	PlayerID         string     `json:"player_id"`
	TribulationType  string     `json:"tribulation_type"` // 'EARTH_CALAMITY' | 'HEAVENLY_TRIBULATION' | 'GRAND_TRIBULATION' | 'MYRIAD_TRIBULATION' | 'CHAOS_CALAMITY'
	Path             string     `json:"path"`
	Difficulty       int        `json:"difficulty"`
	Status           string     `json:"status"` // 'ACTIVE' | 'SUCCESS' | 'FAILED'
	ApertureHPBefore float64    `json:"aperture_hp_before"`
	ApertureHPAfter  *float64   `json:"aperture_hp_after"`
	RewardDaoMarks   int64      `json:"reward_dao_marks"`
	StartedAt        time.Time  `json:"started_at"`
	EndedAt          *time.Time `json:"ended_at"`
}

type TribulationEnemy struct {
	ID           string    `json:"id"`
	EventID      string    `json:"event_id"`
	Name         string    `json:"name"`
	EnemyType    string    `json:"enemy_type"`
	Path         string    `json:"path"`
	HP           float64   `json:"hp"`
	MaxHP        float64   `json:"max_hp"`
	Attack       float64   `json:"attack"`
	Defense      float64   `json:"defense"`
	TargetNodeID *string   `json:"target_node_id"`
	X            int       `json:"x"`
	Y            int       `json:"y"`
	IsBoss       bool      `json:"is_boss"`
	IsDead       bool      `json:"is_dead"`
	Rank         int       `json:"rank"`
	SubRank      int       `json:"sub_rank"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ApertureVariantHuman struct {
	ID              string    `json:"id"`
	ApertureID      string    `json:"aperture_id"`
	Race            string    `json:"race"` // 'HAIRY_MAN', 'ROCKMAN', 'FEATHERMAN'
	Population      int       `json:"population"`
	Happiness       float64   `json:"happiness"`
	Oppression      float64   `json:"oppression"`
	RebellionRate   float64   `json:"rebellion_rate"`
	LaborIntensity  string    `json:"labor_intensity"` // 'LOW', 'NORMAL', 'HIGH', 'EXPLOIT'
	Prosperity      float64   `json:"prosperity"`
	GuMasterCount   int       `json:"gu_master_count"`
	GuImmortalCount int       `json:"gu_immortal_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ApertureNodeWorker represents variant human workers assigned to a resource node.
type ApertureNodeWorker struct {
	ID                 string    `json:"id"`
	NodeID             string    `json:"node_id"`
	VariantHumanID     string    `json:"variant_human_id"`
	AssignedPopulation int       `json:"assigned_population"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// RefinementMode constants for aperture_refinement_queue.
const (
	RefineModeFixedCount    = "FIXED_COUNT"
	RefineModeUntilSuccess  = "UNTIL_SUCCESS"
	RefineModeUntilMaterial = "UNTIL_MATERIAL_EMPTY"
	RefineModeMaintainStock = "MAINTAIN_STOCK"
)

// ApertureRefinementQueue represents an auto-refinement job for variant humans.
type ApertureRefinementQueue struct {
	ID              string     `json:"id"`
	ApertureID      string     `json:"aperture_id"`
	RecipeID        string     `json:"recipe_id"`
	Mode            string     `json:"mode"` // FIXED_COUNT, UNTIL_SUCCESS, UNTIL_MATERIAL_EMPTY, MAINTAIN_STOCK
	TargetQuantity  int        `json:"target_quantity"`
	CurrentQuantity int        `json:"current_quantity"`
	Priority        int        `json:"priority"`
	Enabled         bool       `json:"enabled"`
	LastError       string     `json:"last_error"`
	LastProcessedAt *time.Time `json:"last_processed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ApertureVariantSettlement represents a settlement for a variant human race.
type ApertureVariantSettlement struct {
	ID                 string    `json:"id"`
	ApertureID         string    `json:"aperture_id"`
	Race               string    `json:"race"`
	Name               string    `json:"name"`
	SettlementType     string    `json:"settlement_type"` // VILLAGE, TOWN, CITY, TRIBE
	PopulationCapacity int       `json:"population_capacity"`
	Housing            int       `json:"housing"`
	FoodStorage        int       `json:"food_storage"`
	Security           float64   `json:"security"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// ApertureVariantEvent represents a society event (rebellion, genius birth, etc.).
type ApertureVariantEvent struct {
	ID          string     `json:"id"`
	ApertureID  string     `json:"aperture_id"`
	EventType   string     `json:"event_type"` // REBELLION, GENIUS_BIRTH, IMMORTAL_ASCENSION, MIGRATION, DISASTER
	Race        string     `json:"race"`
	Severity    string     `json:"severity"` // LOW, MEDIUM, HIGH, CRITICAL
	Description string     `json:"description"`
	Effects     string     `json:"effects"` // JSONB
	Resolved    bool       `json:"resolved"`
	CreatedAt   time.Time  `json:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// Society event type constants.
const (
	EventTypeRebellion      = "REBELLION"
	EventTypeGeniusBirth    = "GENIUS_BIRTH"
	EventTypeImmortalAscend = "IMMORTAL_ASCENSION"
	EventTypeMigration      = "MIGRATION"
	EventTypeDisaster       = "DISASTER"
	EventTypeFoodShortage   = "FOOD_SHORTAGE"
)

// NPC role/path constants.
const (
	NPCRoleNone     = "NONE"
	NPCRoleRefiner  = "REFINER"
	NPCRoleMiner    = "MINER"
	NPCRoleFarmer   = "FARMER"
	NPCRoleScholar  = "SCHOLAR"
	NPCRoleGeneral  = "GENERAL"
	NPCRoleGovernor = "GOVERNOR"
)

// ApertureVariantNPC represents a named Gu Master / Gu Immortal NPC within an aperture.
type ApertureVariantNPC struct {
	ID            string       `json:"id"`
	ApertureID    string       `json:"aperture_id"`
	Race          string       `json:"race"`
	Name          string       `json:"name"`
	Rank          int          `json:"rank"`
	SubRank       int          `json:"sub_rank"`
	Path          string       `json:"path"`
	Role          string       `json:"role"`
	Loyalty       float64      `json:"loyalty"`
	Talent        float64      `json:"talent"`
	Experience    int64        `json:"experience"`
	HP            float64      `json:"hp"`
	MaxHP         float64      `json:"max_hp"`
	FormationSlot int          `json:"formation_slot"`
	Version       int64        `json:"version"`
	EquippedGu    []GuInstance `json:"equipped_gu"`
	Status        string       `json:"status"` // IDLE, REFINING, MINING, TRAINING, DEAD
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

type TribTypeConfig struct {
	Type     string
	Interval float64
}

func GetTribulationInterval(rank int, tribType string) float64 {
	switch rank {
	case 6:
		switch tribType {
		case "EARTH_CALAMITY":
			return 10
		case "HEAVENLY_TRIBULATION":
			return 100
		}
	case 7:
		switch tribType {
		case "EARTH_CALAMITY":
			return 10
		case "HEAVENLY_TRIBULATION":
			return 50
		case "GRAND_TRIBULATION":
			return 100
		}
	case 8:
		switch tribType {
		case "HEAVENLY_TRIBULATION":
			return 10
		case "GRAND_TRIBULATION":
			return 50
		case "MYRIAD_TRIBULATION":
			return 100
		}
	case 9:
		switch tribType {
		case "GRAND_TRIBULATION":
			return 10
		case "MYRIAD_TRIBULATION":
			return 50
		case "CHAOS_CALAMITY":
			return 100
		}
	case 10:
		if tribType == "CHAOS_CALAMITY" {
			return 10
		}
	}
	return 0
}

func GetTribulationTypesForRank(rank int) []TribTypeConfig {
	switch rank {
	case 6:
		return []TribTypeConfig{
			{"EARTH_CALAMITY", 10},
			{"HEAVENLY_TRIBULATION", 100},
		}
	case 7:
		return []TribTypeConfig{
			{"EARTH_CALAMITY", 10},
			{"HEAVENLY_TRIBULATION", 50},
			{"GRAND_TRIBULATION", 100},
		}
	case 8:
		return []TribTypeConfig{
			{"HEAVENLY_TRIBULATION", 10},
			{"GRAND_TRIBULATION", 50},
			{"MYRIAD_TRIBULATION", 100},
		}
	case 9:
		return []TribTypeConfig{
			{"GRAND_TRIBULATION", 10},
			{"MYRIAD_TRIBULATION", 50},
			{"CHAOS_CALAMITY", 100},
		}
	case 10:
		return []TribTypeConfig{{"CHAOS_CALAMITY", 10}}
	}
	return nil
}

func SanitizeTribulationTypeForRank(rank int, tribType string) string {
	// Chaotic/Hybrid tribulation is a data-driven override (spec §5.2); it is not
	// part of any rank's normal rotation but must survive sanitization.
	if tribType == "CHAOTIC_TRIBULATION" {
		return tribType
	}
	types := GetTribulationTypesForRank(rank)
	if len(types) == 0 {
		return "EARTH_CALAMITY"
	}
	for _, t := range types {
		if t.Type == tribType {
			return tribType
		}
	}
	return types[0].Type
}

func tribulationPriority(t string) int {
	switch t {
	case "CHAOS_CALAMITY":
		return 5
	case "MYRIAD_TRIBULATION":
		return 4
	case "GRAND_TRIBULATION":
		return 3
	case "HEAVENLY_TRIBULATION":
		return 2
	case "EARTH_CALAMITY":
		return 1
	}
	return 0
}

// nextRoundMultiple returns the smallest multiple of interval that is strictly
// greater than yearsAtRank.  When yearsAtRank ≤ 0 the first interval is returned.
// This ensures tribulation milestones always land on round numbers (10, 20, 30 …
// for interval-10 types; 50, 100, 150 … for interval-50, etc.).
func nextRoundMultiple(yearsAtRank, interval float64) float64 {
	if yearsAtRank <= 0 {
		return interval
	}
	n := math.Floor(yearsAtRank/interval) + 1
	return n * interval
}

// GetNextTribulationInfo returns the year and type of the next tribulation for
// the given rank.  The next year for each type is the next round multiple of
// its interval relative to rankStartYear; the type with the earliest year wins
// (ties broken by rarity priority).  currentYear is the player's internal
// years passed (used to determine position within the current rank cycle).
func GetNextTribulationInfo(rank int, rankStartYear float64, currentYear float64) (nextYear float64, nextType string) {
	cleanRankStartYear := math.Round(rankStartYear)
	yearsAtRank := math.Max(0, currentYear-cleanRankStartYear)
	types := GetTribulationTypesForRank(rank)

	nextYear = math.MaxFloat64
	nextType = ""

	for _, t := range types {
		if t.Interval <= 0 {
			continue
		}
		nextOffset := nextRoundMultiple(yearsAtRank, t.Interval)
		year := cleanRankStartYear + nextOffset
		priority := tribulationPriority(t.Type)

		if year < nextYear || (math.Abs(year-nextYear) < 0.001 && priority > tribulationPriority(nextType)) {
			nextYear = year
			nextType = t.Type
		}
	}

	if nextYear != math.MaxFloat64 {
		nextYear = math.Round(nextYear)
	}

	return nextYear, nextType
}

func GetNextTribulationInfoWithCounts(rank int, rankStartYear float64, counts map[string]int) (nextYear float64, nextType string) {
	nextYear = math.MaxFloat64
	cleanRankStartYear := math.Round(rankStartYear)

	for _, t := range GetTribulationTypesForRank(rank) {
		occurrence := counts[t.Type] + 1
		year := cleanRankStartYear + float64(occurrence)*t.Interval
		priority := tribulationPriority(t.Type)
		if year < nextYear || (math.Abs(year-nextYear) < 0.001 && priority > tribulationPriority(nextType)) {
			nextYear = year
			nextType = t.Type
		}
	}

	if nextYear != math.MaxFloat64 {
		nextYear = math.Round(nextYear)
	}
	return nextYear, nextType
}

func GetTribulationTypeAtYear(rank int, rankStartYear float64, year float64) string {
	offset := math.Round(year) - math.Round(rankStartYear)
	if offset <= 0 {
		return ""
	}

	tribType := ""
	for _, t := range GetTribulationTypesForRank(rank) {
		if math.Mod(offset, t.Interval) == 0 && tribulationPriority(t.Type) > tribulationPriority(tribType) {
			tribType = t.Type
		}
	}
	return tribType
}

func CheckImmortalBreakthrough(rank int, earthCnt, heavenCnt, grandCnt, myriadCnt, chaosCnt int) (bool, int) {
	switch rank {
	case 6:
		if heavenCnt >= 3 {
			return true, 7
		}
	case 7:
		if grandCnt >= 3 {
			return true, 8
		}
	case 8:
		if myriadCnt >= 3 {
			return true, 9
		}
	}
	return false, 0
}

// GetTribulationRankAndSubRank returns the monster rank and subRank based on
// the tribulation type. The player rank controls which types can be scheduled,
// but it does not change the rank of the monster spawned for that type.
func GetTribulationRankAndSubRank(_ int, tribType string) (rank int, subRank int) {
	switch tribType {
	case "EARTH_CALAMITY":
		return 6, 1
	case "HEAVENLY_TRIBULATION":
		return 7, 2
	case "GRAND_TRIBULATION":
		return 8, 3
	case "MYRIAD_TRIBULATION":
		return 9, 4
	case "CHAOS_CALAMITY":
		return 10, 4
	case "CHAOTIC_TRIBULATION":
		return 10, 4
	default:
		return 6, 1
	}
}

// CalculateTribulationReward calculates the Dao Marks reward based on player rank, tribulation type, and difficulty.
// Scales smoothly with the target Dao Marks for each realm (Rank 6: ~40k, Rank 7: ~220k, Rank 8: ~1.4M, Rank 9: ~8.5M).
func CalculateTribulationReward(rank int, tribType string, difficulty int) int64 {
	if rank < 6 {
		rank = 6
	} else if rank > 10 {
		rank = 10
	}

	// Chaotic/Hybrid tribulations reap the highest-tier reward of their rank.
	if tribType == "CHAOTIC_TRIBULATION" {
		tribType = "CHAOS_CALAMITY"
	}

	diff := int64(difficulty)
	switch rank {
	case 6:
		switch tribType {
		case "HEAVENLY_TRIBULATION":
			return 750 + diff*150
		default:
			return 250 + diff*50
		}
	case 7:
		switch tribType {
		case "GRAND_TRIBULATION":
			return 20000 + diff*2000
		case "HEAVENLY_TRIBULATION":
			return 5000 + diff*500
		default:
			return 1500 + diff*200
		}
	case 8:
		switch tribType {
		case "MYRIAD_TRIBULATION":
			return 150000 + diff*10000
		case "GRAND_TRIBULATION":
			return 40000 + diff*3000
		default:
			return 10000 + diff*1000
		}
	case 9:
		switch tribType {
		case "CHAOS_CALAMITY":
			return 1500000 + diff*50000
		case "MYRIAD_TRIBULATION":
			return 200000 + diff*15000
		default:
			return 50000 + diff*5000
		}
	case 10:
		return 1500000 + diff*50000
	default:
		return 250 + diff*50
	}
}
