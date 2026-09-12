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
	ID                      string     `json:"id"`
	ApertureID              string     `json:"aperture_id"`
	NodeType                string     `json:"node_type"`
	Name                    string     `json:"name"`
	Path                    string     `json:"path"`
	Level                   int        `json:"level"`
	HP                      int        `json:"hp"`
	MaxHP                   int        `json:"max_hp"`
	X                       int        `json:"x"`
	Y                       int        `json:"y"`
	ProductionType          string     `json:"production_type"`
	ProductionAmount        int64      `json:"production_amount"`
	ProductionIntervalHours int        `json:"production_interval_hours"`
	LastCollectedAt         *time.Time `json:"last_collected_at"`
	IsDestroyed             bool       `json:"is_destroyed"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
	NodeCategory            string     `json:"node_category"`
	DefensePower            int        `json:"defense_power"`
	AttackRange             int        `json:"attack_range"`
	HPBonus                 int        `json:"hp_bonus"`
	DamageReduction         int        `json:"damage_reduction"`
	BlocksMovement          bool       `json:"blocks_movement"`
	RequiredGuID            string     `json:"required_gu_id"`
	RequiredGuRank          int        `json:"required_gu_rank"`
	ZoneType                string     `json:"zone_type"`
	FoodPath                string     `json:"food_path"`
	ItemTier                string     `json:"item_tier"`
	ProductionKey           string     `json:"production_key"`
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
	RewardDaoMarks   int        `json:"reward_dao_marks"`
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
	ID             string    `json:"id"`
	ApertureID     string    `json:"aperture_id"`
	Race           string    `json:"race"` // 'HAIRY_MAN', 'ROCKMAN', 'FEATHERMAN'
	Population     int       `json:"population"`
	Happiness      float64   `json:"happiness"`
	Oppression     float64   `json:"oppression"`
	RebellionRate  float64   `json:"rebellion_rate"`
	LaborIntensity string    `json:"labor_intensity"` // 'LOW', 'NORMAL', 'HIGH', 'EXPLOIT'
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
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

// GetTribulationRankAndSubRank returns the monster rank and subRank for a tribulation event
// matching the player's realm rank and the calamity type according to lore:
// - Rank 6: Địa Tai (10y - Sơ Giai), Thiên Kiếp (100y - Đỉnh Phong)
// - Rank 7: Địa Tai (10y - Sơ Giai), Thiên Kiếp (50y - Trung Giai), Hạo Kiếp (100y - Đỉnh Phong)
// - Rank 8: Thiên Kiếp (10y - Sơ Giai), Hạo Kiếp (50y - Trung Giai), Vạn Kiếp (100y - Đỉnh Phong)
// - Rank 9: Hạo Kiếp (10y - Sơ Giai), Vạn Kiếp (50y - Trung Giai), Hỗn Độn Đại Nạn (100y - Đỉnh Phong)
func GetTribulationRankAndSubRank(playerRank int, tribType string) (rank int, subRank int) {
	rank = playerRank
	if rank < 6 {
		rank = 6
	} else if rank > 10 {
		rank = 10
	}

	subRank = 1
	switch rank {
	case 6:
		switch tribType {
		case "EARTH_CALAMITY":
			subRank = 1
		case "HEAVENLY_TRIBULATION":
			subRank = 4
		}
	case 7:
		switch tribType {
		case "EARTH_CALAMITY":
			subRank = 1
		case "HEAVENLY_TRIBULATION":
			subRank = 2
		case "GRAND_TRIBULATION":
			subRank = 4
		}
	case 8:
		switch tribType {
		case "HEAVENLY_TRIBULATION":
			subRank = 1
		case "GRAND_TRIBULATION":
			subRank = 2
		case "MYRIAD_TRIBULATION":
			subRank = 4
		}
	case 9:
		switch tribType {
		case "GRAND_TRIBULATION":
			subRank = 1
		case "MYRIAD_TRIBULATION":
			subRank = 2
		case "CHAOS_CALAMITY":
			subRank = 4
		}
	case 10:
		if tribType == "CHAOS_CALAMITY" {
			subRank = 4
		}
	}
	return rank, subRank
}

// CalculateTribulationReward calculates the Dao Marks reward based on player rank, tribulation type, and difficulty.
// Scales smoothly with the target Dao Marks for each realm (Rank 6: ~40k, Rank 7: ~220k, Rank 8: ~1.4M, Rank 9: ~8.5M).
func CalculateTribulationReward(rank int, tribType string, difficulty int) int {
	if rank < 6 {
		rank = 6
	} else if rank > 10 {
		rank = 10
	}

	switch rank {
	case 6:
		switch tribType {
		case "HEAVENLY_TRIBULATION":
			return 750 + difficulty*150
		default:
			return 250 + difficulty*50
		}
	case 7:
		switch tribType {
		case "GRAND_TRIBULATION":
			return 20000 + difficulty*2000
		case "HEAVENLY_TRIBULATION":
			return 5000 + difficulty*500
		default:
			return 1500 + difficulty*200
		}
	case 8:
		switch tribType {
		case "MYRIAD_TRIBULATION":
			return 150000 + difficulty*10000
		case "GRAND_TRIBULATION":
			return 40000 + difficulty*3000
		default:
			return 10000 + difficulty*1000
		}
	case 9:
		switch tribType {
		case "CHAOS_CALAMITY":
			return 1500000 + difficulty*50000
		case "MYRIAD_TRIBULATION":
			return 200000 + difficulty*15000
		default:
			return 50000 + difficulty*5000
		}
	case 10:
		return 1500000 + difficulty*50000
	default:
		return 250 + difficulty*50
	}
}
