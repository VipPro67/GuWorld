package models

import "time"

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
}
