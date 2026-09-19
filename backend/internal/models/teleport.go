package models

import "time"

// Teleport array constants
const (
	TeleportTier1 = "TIER_1" // Vi Trận - Bán kính ngắn (1000 toạ độ)
	TeleportTier2 = "TIER_2" // Đại Vực Trận - Toàn bộ khu vực/vực
	TeleportTier3 = "TIER_3" // Ngũ Vực Tiên Trận - Xuyên Ngũ Vực

	TeleportDirectionTwoWay        = "TWO_WAY"
	TeleportDirectionOneWaySend    = "ONE_WAY_SEND"
	TeleportDirectionOneWayReceive = "ONE_WAY_RECEIVE"

	TeleportAccessPersonal = "PERSONAL"
	TeleportAccessSect     = "SECT"
	TeleportAccessPublic   = "PUBLIC"
)

// TeleportArray represents a placed teleportation formation on the map
type TeleportArray struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	OwnerID           string    `json:"owner_id"`
	OwnerName         string    `json:"owner_name,omitempty"`
	SectID            *string   `json:"sect_id,omitempty"`
	SectName          *string   `json:"sect_name,omitempty"`
	RegionID          string    `json:"region_id"`
	RegionName        string    `json:"region_name,omitempty"`
	X                 int       `json:"x"`
	Y                 int       `json:"y"`
	Tier              string    `json:"tier"`
	MaxRange          int       `json:"max_range"`
	IsCrossRegion     bool      `json:"is_cross_region"`
	DirectionType     string    `json:"direction_type"`
	AccessPolicy      string    `json:"access_policy"`
	FeeEnabled        bool      `json:"fee_enabled"`
	FeeCurrency       string    `json:"fee_currency"`
	FeeAmount         int64     `json:"fee_amount"`
	VaultBalance      int64     `json:"vault_balance"`
	CurrentDurability int       `json:"current_durability"`
	MaxDurability     int       `json:"max_durability"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// TeleportTemplate defines construction requirements and stats for each tier
type TeleportTemplate struct {
	Tier               string         `json:"tier"`
	Name               string         `json:"name"`
	Description        string         `json:"description"`
	MinRank            int            `json:"min_rank"`
	MinFormationRealm  int            `json:"min_formation_realm"` // 0=Ordinary, 1=Master, 2=Grandmaster, 3=Great Grandmaster, 4=Supreme
	BaseSuccessRate    float64        `json:"base_success_rate"`   // Percentage: e.g. 70.0 for 70%
	MaxRange           int            `json:"max_range"`
	IsCrossRegion      bool           `json:"is_cross_region"`
	MaxDurability      int            `json:"max_durability"`
	CostPrimevalStones float64        `json:"cost_primeval_stones"`
	CostImmortalStones float64        `json:"cost_immortal_stones"`
	RequiredGu         []string       `json:"required_gu"`
	RequiredItems      map[string]int `json:"required_items"`
}

// TeleportDirectoryItem is a destination node returned in the directory query
type TeleportDirectoryItem struct {
	Array             TeleportArray `json:"array"`
	Distance          int           `json:"distance"`
	IsCrossRegionJump bool          `json:"is_cross_region_jump"`
	CanUse            bool          `json:"can_use"`
	Reason            string        `json:"reason,omitempty"`
	FeeToPay          int64         `json:"fee_to_pay"`
	FeeCurrency       string        `json:"fee_currency"`
}

// BuildTeleportArrayResp is returned after attempting to build a teleport array
type BuildTeleportArrayResp struct {
	Success            bool           `json:"success"`
	Message            string         `json:"message"`
	Array              *TeleportArray `json:"array,omitempty"`
	SuccessRate        float64        `json:"success_rate"`
	FormationRealm     string         `json:"formation_realm"`
	FormationLevel     int            `json:"formation_level"`
	DiscountPercent    float64        `json:"discount_percent"`
	MasteryGained      int            `json:"mastery_gained"`
	PaidPrimevalStones float64        `json:"paid_primeval_stones"`
	PaidImmortalStones float64        `json:"paid_immortal_stones"`
}

// RepairTeleportArrayResp is returned after repairing a teleport array
type RepairTeleportArrayResp struct {
	Success            bool    `json:"success"`
	Message            string  `json:"message"`
	CurrentDurability  int     `json:"current_durability"`
	MaxDurability      int     `json:"max_durability"`
	IsCriticalRepair   bool    `json:"is_critical_repair"`
	CriticalBonus      int     `json:"critical_bonus"`
	DiscountPercent    float64 `json:"discount_percent"`
	MasteryGained      int     `json:"mastery_gained"`
	PaidPrimevalStones float64 `json:"paid_primeval_stones"`
}

// Request types
type BuildTeleportArrayReq struct {
	Name          string `json:"name" binding:"required"`
	Tier          string `json:"tier" binding:"required"`
	DirectionType string `json:"direction_type"`
	AccessPolicy  string `json:"access_policy"`
	FeeEnabled    bool   `json:"fee_enabled"`
	FeeCurrency   string `json:"fee_currency"`
	FeeAmount     int64  `json:"fee_amount"`
}

type TravelTeleportReq struct {
	TargetArrayID string `json:"target_array_id" binding:"required"`
}

type ManageTeleportArrayReq struct {
	ArrayID       string  `json:"array_id" binding:"required"`
	Name          *string `json:"name,omitempty"`
	DirectionType *string `json:"direction_type,omitempty"`
	AccessPolicy  *string `json:"access_policy,omitempty"`
	FeeEnabled    *bool   `json:"fee_enabled,omitempty"`
	FeeCurrency   *string `json:"fee_currency,omitempty"`
	FeeAmount     *int64  `json:"fee_amount,omitempty"`
}

type RepairTeleportArrayReq struct {
	ArrayID string `json:"array_id" binding:"required"`
	Amount  int    `json:"amount"` // Durability points to repair
}

type WithdrawTeleportVaultReq struct {
	ArrayID string `json:"array_id" binding:"required"`
	Amount  int64  `json:"amount"` // 0 or empty means withdraw all
}

type DismantleTeleportArrayReq struct {
	ArrayID string `json:"array_id" binding:"required"`
}
