package models

import (
	"math"
	"time"
)

type ImmortalAperture struct {
	ID                       string    `json:"id"`
	PlayerID                 string    `json:"player_id"`
	Grade                    string    `json:"grade"` // 'LOW' | 'MID' | 'HIGH' | 'EXTREME'
	TimeRate                 float64   `json:"time_rate"`
	ApertureHP               int       `json:"aperture_hp"`
	MaxApertureHP            int       `json:"max_aperture_hp"`
	AreaSize                 int64     `json:"area_size"`
	ImmortalEssence          int       `json:"immortal_essence"`
	EssenceProductionPerYear int       `json:"essence_production_per_year"`
	InternalYearsPassed      float64   `json:"internal_years_passed"`
	NextTribulationYear      float64   `json:"next_tribulation_year"`
	TribulationCount         int       `json:"tribulation_count"`
	RankStartYear            float64   `json:"rank_start_year"`
	EarthCalamityCount       int       `json:"earth_calamity_count"`
	HeavenlyTribulationCount  int       `json:"heavenly_tribulation_count"`
	GrandTribulationCount     int       `json:"grand_tribulation_count"`
	MyriadTribulationCount    int       `json:"myriad_tribulation_count"`
	ChaosCalamityCount        int       `json:"chaos_calamity_count"`
	State                    string    `json:"state"` // 'STABLE' | 'TRIBULATION_PENDING' | 'TRIBULATION_ACTIVE' | 'DAMAGED' | 'DESTROYED'
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
	Temperature              float64   `json:"temperature,omitempty"`
	Humidity                 float64   `json:"humidity,omitempty"`
	Qi                       float64   `json:"qi,omitempty"`
	Terrain                  string    `json:"terrain,omitempty"`
}

type DaoMark struct {
	ID        string    `json:"id"`
	PlayerID  string    `json:"player_id"`
	Path      string    `json:"path"`
	Amount    int       `json:"amount"`
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
	ProductionAmount        int        `json:"production_amount"`
	ProductionIntervalHours int        `json:"production_interval_hours"`
	LastCollectedAt         *time.Time `json:"last_collected_at"`
	IsDestroyed             bool       `json:"is_destroyed"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

type TribulationEvent struct {
	ID               string     `json:"id"`
	ApertureID       string     `json:"aperture_id"`
	PlayerID         string     `json:"player_id"`
	TribulationType  string     `json:"tribulation_type"` // 'EARTH_CALAMITY' | 'HEAVENLY_TRIBULATION' | 'GRAND_TRIBULATION' | 'MYRIAD_TRIBULATION' | 'CHAOS_CALAMITY'
	Path             string     `json:"path"`
	Difficulty       int        `json:"difficulty"`
	Status           string     `json:"status"` // 'ACTIVE' | 'SUCCESS' | 'FAILED'
	ApertureHPBefore int        `json:"aperture_hp_before"`
	ApertureHPAfter  *int       `json:"aperture_hp_after"`
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
	HP           int       `json:"hp"`
	MaxHP        int       `json:"max_hp"`
	Attack       int       `json:"attack"`
	Defense      int       `json:"defense"`
	TargetNodeID *string   `json:"target_node_id"`
	X            int       `json:"x"`
	Y            int       `json:"y"`
	IsBoss       bool      `json:"is_boss"`
	IsDead       bool      `json:"is_dead"`
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

func GetTribTypeCount(a *ImmortalAperture, tribType string) int {
	switch tribType {
	case "EARTH_CALAMITY":
		return a.EarthCalamityCount
	case "HEAVENLY_TRIBULATION":
		return a.HeavenlyTribulationCount
	case "GRAND_TRIBULATION":
		return a.GrandTribulationCount
	case "MYRIAD_TRIBULATION":
		return a.MyriadTribulationCount
	case "CHAOS_CALAMITY":
		return a.ChaosCalamityCount
	}
	return 0
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

func GetNextTribulationInfo(rank int, earthCnt, heavenCnt, grandCnt, myriadCnt, chaosCnt int) (nextYear float64, nextType string) {
	types := []struct {
		Type     string
		Interval float64
		Count    int
	}{
		{"EARTH_CALAMITY", GetTribulationInterval(rank, "EARTH_CALAMITY"), earthCnt},
		{"HEAVENLY_TRIBULATION", GetTribulationInterval(rank, "HEAVENLY_TRIBULATION"), heavenCnt},
		{"GRAND_TRIBULATION", GetTribulationInterval(rank, "GRAND_TRIBULATION"), grandCnt},
		{"MYRIAD_TRIBULATION", GetTribulationInterval(rank, "MYRIAD_TRIBULATION"), myriadCnt},
		{"CHAOS_CALAMITY", GetTribulationInterval(rank, "CHAOS_CALAMITY"), chaosCnt},
	}

	nextYear = math.MaxFloat64
	nextType = ""

	for _, t := range types {
		if t.Interval <= 0 {
			continue
		}
		year := t.Interval * float64(t.Count+1)
		priority := tribulationPriority(t.Type)

		if year < nextYear || (math.Abs(year-nextYear) < 0.001 && priority > tribulationPriority(nextType)) {
			nextYear = year
			nextType = t.Type
		}
	}

	return nextYear, nextType
}

func CheckImmortalBreakthrough(rank int, earthCnt, heavenCnt, grandCnt, myriadCnt, chaosCnt int, yearsAtRank float64) (bool, int) {
	switch rank {
	case 6:
		if heavenCnt >= 3 && yearsAtRank >= 300 {
			return true, 7
		}
	case 7:
		if grandCnt >= 3 && yearsAtRank >= 300 {
			return true, 8
		}
	case 8:
		if myriadCnt >= 3 {
			return true, 9
		}
	}
	return false, 0
}
