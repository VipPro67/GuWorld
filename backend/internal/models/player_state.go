package models

import (
	"time"

	"github.com/google/uuid"
)

// Account groups authentication and identity fields.
type Account struct {
	ID              uuid.UUID `json:"id"`
	Username        string    `json:"username"`
	PasswordHash    string    `json:"-"`
	Email           string    `json:"email"`
	GoogleID        *string   `json:"google_id,omitempty"`
	RefCode         string    `json:"ref_code"`
	Bio             string    `json:"bio"`
	AvatarURL       string    `json:"avatar_url"`
	IsAdmin         bool      `json:"is_admin"`
	IsDisguised     bool      `json:"is_disguised"`
	IsDead          bool      `json:"is_dead"`
	FakeRank        int       `json:"fake_rank"`
	FakeSubRank     int       `json:"fake_sub_rank"`
	FakeVitalGu     string    `json:"fake_vital_gu"`
	NeedsOnboarding bool      `json:"needs_onboarding"`
	Version         int       `json:"version"`
	CreatedAt       time.Time `json:"created_at"`
}

// NewAccount extracts account data from a Player.
func NewAccount(p *Player) Account {
	return Account{
		ID:              p.ID,
		Username:        p.Username,
		PasswordHash:    p.PasswordHash,
		Email:           p.Email,
		GoogleID:        p.GoogleID,
		RefCode:         p.RefCode,
		Bio:             p.Bio,
		AvatarURL:       p.AvatarURL,
		IsAdmin:         p.IsAdmin,
		IsDisguised:     p.IsDisguised,
		IsDead:          p.IsDead,
		FakeRank:        p.FakeRank,
		FakeSubRank:     p.FakeSubRank,
		FakeVitalGu:     p.FakeVitalGu,
		NeedsOnboarding: p.NeedsOnboarding,
		Version:         p.Version,
		CreatedAt:       p.CreatedAt,
	}
}

// Apply writes the account fields back to a Player.
func (s Account) Apply(p *Player) {
	p.ID = s.ID
	p.Username = s.Username
	p.PasswordHash = s.PasswordHash
	p.Email = s.Email
	p.GoogleID = s.GoogleID
	p.RefCode = s.RefCode
	p.Bio = s.Bio
	p.AvatarURL = s.AvatarURL
	p.IsAdmin = s.IsAdmin
	p.IsDisguised = s.IsDisguised
	p.IsDead = s.IsDead
	p.FakeRank = s.FakeRank
	p.FakeSubRank = s.FakeSubRank
	p.FakeVitalGu = s.FakeVitalGu
	p.NeedsOnboarding = s.NeedsOnboarding
	p.Version = s.Version
	p.CreatedAt = s.CreatedAt
}

// CultivationState groups cultivation progression fields.
type CultivationState struct {
	Rank                int        `json:"rank"`
	SubRank             int        `json:"sub_rank"`
	AptitudeType        string     `json:"aptitude_type"`
	AptitudePercent     int        `json:"aptitude_percent"`
	AptitudeGrade       int        `json:"aptitude_grade"`
	WallProgress        float64    `json:"wall_progress"`
	HumanQi             float64    `json:"human_qi"`
	Lifespan            float64    `json:"lifespan"`
	ApertureStress      float64    `json:"aperture_stress"`
	LastCultivateTime   time.Time  `json:"last_cultivate_time"`
	TimeSpeedMultiplier float64    `json:"time_speed_multiplier"`
	NextTribulationTime *time.Time `json:"next_tribulation_time"`
}

// NewCultivationState extracts cultivation data from a Player.
func NewCultivationState(p *Player) CultivationState {
	return CultivationState{
		Rank:                p.Rank,
		SubRank:             p.SubRank,
		AptitudeType:        p.AptitudeType,
		AptitudePercent:     p.AptitudePercent,
		AptitudeGrade:       p.AptitudeGrade,
		WallProgress:        p.WallProgress,
		HumanQi:             p.HumanQi,
		Lifespan:            p.Lifespan,
		ApertureStress:      p.ApertureStress,
		LastCultivateTime:   p.LastCultivateTime,
		TimeSpeedMultiplier: p.TimeSpeedMultiplier,
		NextTribulationTime: p.NextTribulationTime,
	}
}

// Apply writes cultivation fields back to a Player.
func (s CultivationState) Apply(p *Player) {
	p.Rank = s.Rank
	p.SubRank = s.SubRank
	p.AptitudeType = s.AptitudeType
	p.AptitudePercent = s.AptitudePercent
	p.AptitudeGrade = s.AptitudeGrade
	p.WallProgress = s.WallProgress
	p.HumanQi = s.HumanQi
	p.Lifespan = s.Lifespan
	p.ApertureStress = s.ApertureStress
	p.LastCultivateTime = s.LastCultivateTime
	p.TimeSpeedMultiplier = s.TimeSpeedMultiplier
	p.NextTribulationTime = s.NextTribulationTime
}

// ResourceState groups mana and currency fields.
type ResourceState struct {
	CurrentMana      float64   `json:"current_mana"`
	MaxMana          float64   `json:"max_mana"`
	PrimevalStones   float64   `json:"primeval_stones"`
	ImmortalStones   float64   `json:"immortal_stones"`
	LastManaUpdate   time.Time `json:"last_mana_update"`
	LastHungerTickAt time.Time `json:"last_hunger_tick_at"`
}

// NewResourceState extracts resource data from a Player.
func NewResourceState(p *Player) ResourceState {
	return ResourceState{
		CurrentMana:      p.CurrentMana,
		MaxMana:          p.MaxMana,
		PrimevalStones:   p.PrimevalStones,
		ImmortalStones:   p.ImmortalStones,
		LastManaUpdate:   p.LastManaUpdate,
		LastHungerTickAt: p.LastHungerTickAt,
	}
}

// Apply writes resource fields back to a Player.
func (s ResourceState) Apply(p *Player) {
	p.CurrentMana = s.CurrentMana
	p.MaxMana = s.MaxMana
	p.PrimevalStones = s.PrimevalStones
	p.ImmortalStones = s.ImmortalStones
	p.LastManaUpdate = s.LastManaUpdate
	p.LastHungerTickAt = s.LastHungerTickAt
}

// BodyState groups HP and body modification fields.
type BodyState struct {
	HP             float64      `json:"hp"`
	BaseHP         float64      `json:"base_hp"`
	BodyMods       BodyModsList `json:"body_mods"`
	DaoMarkDamage  float64      `json:"dao_mark_damage"`
	ApertureDamage float64      `json:"aperture_damage"`
}

// NewBodyState extracts body data from a Player.
func NewBodyState(p *Player) BodyState {
	return BodyState{
		HP:             p.HP,
		BaseHP:         p.BaseHP,
		BodyMods:       p.BodyMods,
		DaoMarkDamage:  p.DaoMarkDamage,
		ApertureDamage: p.ApertureDamage,
	}
}

// Apply writes body fields back to a Player.
func (s BodyState) Apply(p *Player) {
	p.HP = s.HP
	p.BaseHP = s.BaseHP
	p.BodyMods = s.BodyMods
	p.DaoMarkDamage = s.DaoMarkDamage
	p.ApertureDamage = s.ApertureDamage
}

// CharacterState groups the player's race (chủng tộc), gender (giới tính),
// constitution (thể chất) and active statuses (trạng thái). Values are stored
// as English keys; effects are resolved by handlers.CharacterEffects.
type CharacterState struct {
	Race         string     `json:"race"`
	Gender       string     `json:"gender"`
	Constitution string     `json:"constitution"`
	Statuses     StatusList `json:"statuses"`
}

// NewCharacterState extracts character data from a Player.
func NewCharacterState(p *Player) CharacterState {
	return CharacterState{
		Race:         p.Race,
		Gender:       p.Gender,
		Constitution: p.Constitution,
		Statuses:     p.Statuses,
	}
}

// Apply writes character fields back to a Player.
func (s CharacterState) Apply(p *Player) {
	p.Race = s.Race
	p.Gender = s.Gender
	p.Constitution = s.Constitution
	p.Statuses = s.Statuses
}

// SoulState groups soul-related fields.
type SoulState struct {
	SoulHP          int    `json:"soul_hp"`
	MaxSoul         int    `json:"max_soul"`
	SoulTier        string `json:"soul_tier"`
	NeedsSoulRefine bool   `json:"needs_soul_refine"`
}

// NewSoulState extracts soul data from a Player.
func NewSoulState(p *Player) SoulState {
	return SoulState{
		SoulHP:          p.SoulHP,
		MaxSoul:         p.MaxSoul,
		SoulTier:        p.SoulTier,
		NeedsSoulRefine: p.NeedsSoulRefine,
	}
}

// Apply writes soul fields back to a Player.
func (s SoulState) Apply(p *Player) {
	p.SoulHP = s.SoulHP
	p.MaxSoul = s.MaxSoul
	p.SoulTier = s.SoulTier
	p.NeedsSoulRefine = s.NeedsSoulRefine
}

// ThoughtPool groups thought-related fields (replaces old Wisdom stat).
type ThoughtPool struct {
	CurrentThought float64 `json:"current_thought"`
	MaxThought     float64 `json:"max_thought"`
	RegenRate      float64 `json:"thought_regen_rate"`
}

// NewThoughtPool extracts thought data from a Player.
func NewThoughtPool(p *Player) ThoughtPool {
	return ThoughtPool{
		CurrentThought: p.CurrentThought,
		MaxThought:     p.MaxThought,
		RegenRate:      p.RegenRate,
	}
}

// Apply writes thought fields back to a Player.
func (s ThoughtPool) Apply(p *Player) {
	p.CurrentThought = s.CurrentThought
	p.MaxThought = s.MaxThought
	p.RegenRate = s.RegenRate
}

// DaoState groups dao marks and path mastery fields.
type DaoState struct {
	DaoMarks      DaoMarksMap      `json:"dao_marks"`
	PathMasteries PathMasteriesMap `json:"path_masteries"`
}

// NewDaoState extracts dao data from a Player.
func NewDaoState(p *Player) DaoState {
	return DaoState{
		DaoMarks:      p.DaoMarks,
		PathMasteries: p.PathMasteries,
	}
}

// Apply writes dao fields back to a Player.
func (s DaoState) Apply(p *Player) {
	p.DaoMarks = s.DaoMarks
	p.PathMasteries = s.PathMasteries
}

// WorldState groups world position and exploration fields.
type WorldState struct {
	X                int                 `json:"x"`
	Y                int                 `json:"y"`
	CurrentRegion    string              `json:"current_region"`
	ClearedCells     ClearedCellsMap     `json:"cleared_cells"`
	VisitedSubZones  VisitedSubZonesList `json:"visited_subzones"`
	ExploreCount     int                 `json:"explore_count"`
	LastExploreReset time.Time           `json:"last_explore_reset"`
}

// NewWorldState extracts world data from a Player.
func NewWorldState(p *Player) WorldState {
	return WorldState{
		X:                p.X,
		Y:                p.Y,
		CurrentRegion:    p.CurrentRegion,
		ClearedCells:     p.ClearedCells,
		VisitedSubZones:  p.VisitedSubZones,
		ExploreCount:     p.ExploreCount,
		LastExploreReset: p.LastExploreReset,
	}
}

// Apply writes world fields back to a Player.
func (s WorldState) Apply(p *Player) {
	p.X = s.X
	p.Y = s.Y
	p.CurrentRegion = s.CurrentRegion
	p.ClearedCells = s.ClearedCells
	p.VisitedSubZones = s.VisitedSubZones
	p.ExploreCount = s.ExploreCount
	p.LastExploreReset = s.LastExploreReset
}

// SocialState groups social, party, and arena fields.
type SocialState struct {
	PartyID   *uuid.UUID `json:"party_id"`
	ArenaRank *int       `json:"arena_rank"`
}

// NewSocialState extracts social data from a Player.
func NewSocialState(p *Player) SocialState {
	return SocialState{
		PartyID:   p.PartyID,
		ArenaRank: p.ArenaRank,
	}
}

// Apply writes social fields back to a Player.
func (s SocialState) Apply(p *Player) {
	p.PartyID = s.PartyID
	p.ArenaRank = s.ArenaRank
}

// CombatConfig groups combat equipment and auto-combat settings.
type CombatConfig struct {
	EquippedGus    EquippedGusList  `json:"equipped_gus"`
	CombatSettings AutoCombatConfig `json:"combat_settings"`
}

// NewCombatConfig extracts combat config from a Player.
func NewCombatConfig(p *Player) CombatConfig {
	return CombatConfig{
		EquippedGus:    p.EquippedGus,
		CombatSettings: p.CombatSettings,
	}
}

// Apply writes combat config fields back to a Player.
func (s CombatConfig) Apply(p *Player) {
	p.EquippedGus = s.EquippedGus
	p.CombatSettings = s.CombatSettings
}
