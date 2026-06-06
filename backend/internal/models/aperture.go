package models

import (
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
	State                    string    `json:"state"` // 'STABLE' | 'TRIBULATION_PENDING' | 'TRIBULATION_ACTIVE' | 'DAMAGED' | 'DESTROYED'
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
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
	TribulationType  string     `json:"tribulation_type"` // 'EARTH_CALAMITY' | 'HEAVENLY_TRIBULATION' | 'GRAND_TRIBULATION' | 'MYRIAD_TRIBULATION'
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
