package models

import (
	"time"

	"github.com/google/uuid"
)

// TrackingMark represents an active tracking impression on a target player
type TrackingMark struct {
	ID        uuid.UUID `json:"id"`
	TrackerID uuid.UUID `json:"tracker_id"`
	TargetID  uuid.UUID `json:"target_id"`
	MarkType  string    `json:"mark_type"` // "wisdom", "blood", "dao"
	Power     float64   `json:"power"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`

	// Enriched fields for client responses
	TargetUsername string `json:"target_username,omitempty"`
	TargetAvatar   string `json:"target_avatar,omitempty"`
	CurrentRegion  string `json:"current_region,omitempty"`
	X              int    `json:"x,omitempty"`
	Y              int    `json:"y,omitempty"`
}

// ResidualTrace represents residual spiritual scent/traces left at a grid cell
type ResidualTrace struct {
	ID               uuid.UUID `json:"id"`
	RegionID         string    `json:"region_id"`
	X                int       `json:"x"`
	Y                int       `json:"y"`
	PlayerID         uuid.UUID `json:"player_id"`
	ActionType       string    `json:"action_type"` // "move", "combat", "mine", "tribulation"
	DominantPath     string    `json:"dominant_path"`
	DaoPulseStrength float64   `json:"dao_pulse_strength"`
	ExpiresAt        time.Time `json:"expires_at"`
	CreatedAt        time.Time `json:"created_at"`
}

// InspectRequest is used for local Worldmap FOV inspection
type InspectRequest struct {
	TargetID       string `json:"target_id"`
	TargetUsername string `json:"target_username"`
}

// DivinationRequest is used for remote deduction
type DivinationRequest struct {
	TargetUsername  string  `json:"target_username"`
	SuspectedRegion string  `json:"suspected_region"`
	SuspectedSect   string  `json:"suspected_sect"`
	SuspectedGu     string  `json:"suspected_gu"`
	ItemClueID      string  `json:"item_clue_id"`
	ResourceType    string  `json:"resource_type"` // "mana" or "immortal_stone"
	ResourceAmount  float64 `json:"resource_amount"`
	TargetSectName  string  `json:"-"` // populated server-side for clue validation
}

// Coordinates represents (X, Y) on the world map
type Coordinates struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// DivinationReport contains deduced intelligence
type DivinationReport struct {
	TargetID      uuid.UUID    `json:"target_id"`
	Username      string       `json:"username"`
	Region        string       `json:"region"`
	Coordinates   *Coordinates `json:"coordinates,omitempty"` // null if partial/fogged
	ApproxRank    int          `json:"approx_rank"`
	SubRank       int          `json:"sub_rank,omitempty"`
	DominantPath  string       `json:"dominant_path"`
	DetectedGus   []string     `json:"detected_gus"`
	StatusSummary string       `json:"status_summary"`
	IsDisguised   bool         `json:"is_disguised"`
}

// DivinationResponse is the outcome of a remote deduction
type DivinationResponse struct {
	StatusType         string            `json:"status_type"` // "full", "partial", "fogged", "backfire"
	DeltaScore         float64           `json:"delta_score"`
	Report             *DivinationReport `json:"divination_report,omitempty"`
	BackfireDamage     float64           `json:"backfire_damage,omitempty"`
	BackfireSoulDamage float64           `json:"backfire_soul_damage,omitempty"`
	CanApplyMark       bool              `json:"can_apply_tracking_mark"`
	Message            string            `json:"message"`
}

// ApplyMarkRequest is used to apply a tracking mark
type ApplyMarkRequest struct {
	TargetID string `json:"target_id"`
	MarkType string `json:"mark_type"` // "wisdom" | "blood"
}

// SeverCluesRequest is used to purge marks and traces
type SeverCluesRequest struct {
	UseKillerMove bool `json:"use_killer_move"`
}

// SeverCluesResponse is returned after purging
type SeverCluesResponse struct {
	Success           bool   `json:"success"`
	PurgedMarksCount  int    `json:"purged_marks_count"`
	PurgedTracesCount int    `json:"purged_traces_count"`
	Message           string `json:"message"`
}

// ResonanceScanRequest is used to scan for Gu or Dao Mark pulses
type ResonanceScanRequest struct {
	TargetPath string `json:"target_path"`
	MinRank    int    `json:"min_rank"`
	ScanRegion string `json:"scan_region"`
}

// ResonancePing represents a detected Gu or node resonance on map
type ResonancePing struct {
	X          int    `json:"x"`
	Y          int    `json:"y"`
	Intensity  string `json:"intensity"` // "high", "medium", "low"
	GuID       string `json:"gu_id"`
	EntityType string `json:"entity_type"` // "wild_gu", "player", "mine"
}

// DaoPulsePing represents world-level tribulation/immortal dao pulses
type DaoPulsePing struct {
	Region           string `json:"region"`
	X                int    `json:"x"`
	Y                int    `json:"y"`
	Path             string `json:"path"`
	Event            string `json:"event"` // "tribulation", "immortal_clash", "aperture_open"
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

// ResonanceScanResponse is returned by resonance scan
type ResonanceScanResponse struct {
	Pings            []ResonancePing `json:"pings"`
	DaoPulses        []DaoPulsePing  `json:"dao_pulses"`
	CooldownSeconds  int             `json:"cooldown_seconds"`
}

// DivineGuRequest is used for deducing a Gu worm without needing a target player
type DivineGuRequest struct {
	GuQuery        string  `json:"gu_query"` // GuID or Gu Name (e.g. "xuan_bing_gu", "Xuân Thu Thiền")
	TargetRank     int     `json:"target_rank,omitempty"`
	ResourceType   string  `json:"resource_type"` // "mana" or "immortal_stone"
	ResourceAmount float64 `json:"resource_amount"`
}

// DivineGuHolderInfo represents intelligence on an entity holding or listing the Gu
type DivineGuHolderInfo struct {
	HolderType     string       `json:"holder_type"` // "player" | "sect_vault" | "auction_listing"
	HolderUsername string       `json:"holder_username,omitempty"`
	HolderName     string       `json:"holder_name,omitempty"`
	Region         string       `json:"region,omitempty"`
	Coordinates    *Coordinates `json:"coordinates,omitempty"`
	IsVital        bool         `json:"is_vital,omitempty"`
	Equipped       bool         `json:"equipped,omitempty"`
	LocationState  string       `json:"location_state,omitempty"` // "equipped", "inventory", "sleeping", "sect_vault", "auction_listing"
	Quantity       int          `json:"quantity"`
	PriceInfo      string       `json:"price_info,omitempty"`
	Concealed      bool         `json:"concealed"`
}

// DivineGuReport contains deduced intelligence on a Gu worm
type DivineGuReport struct {
	GuID           string               `json:"gu_id"`
	GuName         string               `json:"gu_name"`
	Rank           int                  `json:"rank"`
	Path           string               `json:"path"`
	IsImmortal     bool                 `json:"is_immortal"`
	IsRefined      bool                 `json:"is_refined"`
	WorldCount     int                  `json:"world_count"`
	PlayerCount    int                  `json:"player_count"`
	SectVaultCount int                  `json:"sect_vault_count"`
	MarketCount    int                  `json:"market_count"`
	Holders        []DivineGuHolderInfo `json:"holders,omitempty"`
	FoodHint       string               `json:"food_hint,omitempty"`
	RecipeHint     string               `json:"recipe_hint,omitempty"`
	NaturalHabitat string               `json:"natural_habitat,omitempty"`
	LoreStatus     string               `json:"lore_status"`
}

// DivineGuResponse is the outcome of a Gu deduction
type DivineGuResponse struct {
	StatusType         string          `json:"status_type"` // "full", "partial", "fogged", "backfire"
	DeltaScore         float64         `json:"delta_score"`
	Report             *DivineGuReport `json:"report,omitempty"`
	BackfireDamage     float64         `json:"backfire_damage,omitempty"`
	BackfireSoulDamage float64         `json:"backfire_soul_damage,omitempty"`
	Message            string          `json:"message"`
}

