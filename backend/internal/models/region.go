package models

import "time"

// ZonePools defines procedural content pools for a zone. All fields are
// data-driven via the regions.pool_weights JSONB column.
type ZonePools struct {
	CellWeights map[string]int `json:"cell_type_weights"` // Monster / Resource / WildGu / Empty
	Resources   []string       `json:"resources"`         // resource item ids
	WildGus     []string       `json:"wild_gus"`          // low-tier Gu ids
}

// Region represents a zone in the shared continuous world map. Regions are
// hierarchical: HUB (safe lobby), DAI_VUC (one of the 3x2 large zones,
// 10000x10000 each) and NGU_VUC (named areas that live inside a DAI_VUC).
// Coordinates are global world coordinates.
type Region struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	MinRank     int       `json:"min_rank"`
	MaxRank     int       `json:"max_rank"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	ParentID    string    `json:"parent_id,omitempty"`
	ZoneType    string    `json:"zone_type"` // "HUB" | "DAI_VUC" | "NGU_VUC"
	MinX        int       `json:"min_x"`
	MinY        int       `json:"min_y"`
	MaxX        int       `json:"max_x"`
	MaxY        int       `json:"max_y"`
	EntryX      int       `json:"entry_x"`
	EntryY      int       `json:"entry_y"`
	PoolWeights ZonePools `json:"pool_weights"`
}
