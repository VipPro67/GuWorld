package models

import (
	"encoding/json"
	"time"
)

const (
	KillerMoveSlotCore    = "CORE"
	KillerMoveSlotSupport = "SUPPORT"
)

type KillerMove struct {
	ID             string           `json:"id" gorm:"primaryKey"`
	PlayerID       string           `json:"player_id" gorm:"index;not null"`
	Name           string           `json:"name" gorm:"not null"`
	Path           string           `json:"path" gorm:"index"`
	MoveType       string           `json:"move_type"`
	RankRequired   int              `json:"rank_required"`
	CorePlayerGuID string           `json:"core_player_gu_id" gorm:"not null"`
	TotalManaCost  float64          `json:"total_mana_cost"`
	SuccessRate    float64          `json:"success_rate"`
	DamageValue    float64          `json:"damage_value"`
	DefenseValue   float64          `json:"defense_value"`
	Cooldown       int              `json:"cooldown"`
	ChannelTime    int              `json:"channel_time"`
	Effects        json.RawMessage  `json:"effects"`
	IsMastered     bool             `json:"is_mastered"`
	IsReliable     bool             `json:"is_reliable"`
	IsInstantCast  bool             `json:"is_instant_cast"`
	UsageCount     int              `json:"usage_count"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
	Slots          []KillerMoveSlot `json:"slots,omitempty"`
}

type KillerMoveSlot struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	KillerMoveID string    `json:"killer_move_id" gorm:"index;not null"`
	PlayerGuID   string    `json:"player_gu_id" gorm:"not null"`
	SlotType     string    `json:"slot_type"`
	SlotIndex    int       `json:"slot_index"`
	CreatedAt    time.Time `json:"created_at"`
}
