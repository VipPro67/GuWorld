package main

import (
	"encoding/json"
	"os"
)

// config is the optional JSON config for gudata. Missing fields fall back to
// the defaults below, so a bare `{}` (or no -config) is valid.
type config struct {
	Balance balanceConfig `json:"balance"`
	Recipe  recipeConfig  `json:"recipe"`
}

type balanceConfig struct {
	// Tol is the fractional band around the rank median power: a Gu is flagged
	// when its score is below median*(1-Tol) or above median*(1+Tol).
	Tol float64 `json:"tol"`
	// MinScoreFrac suppresses the "too weak" flag for Gu whose score is below
	// median*MinScoreFrac (utility/support Gu with no combat numbers).
	MinScoreFrac float64 `json:"min_score_frac"`
	// Weights scale each power component (defaults 1.0).
	Weights map[string]float64 `json:"weights"`
}

type recipeConfig struct {
	SuccessRate  map[string]float64 `json:"success_rate"`
	PrimevalCost map[string]float64 `json:"primeval_cost"`
	ImmortalCost map[string]float64 `json:"immortal_cost"`
	// MaterialQty is the base quantity per ingredient before scaling.
	MaterialQty float64 `json:"material_qty"`
	// MaterialQtyScale multiplies MaterialQty (the "tăng lượng tài nguyên" knob).
	MaterialQtyScale float64 `json:"material_qty_scale"`
	// MaterialCount is how many distinct ingredients to auto-select.
	MaterialCount int `json:"material_count"`
	// RequiredGuQty is the quantity of the predecessor Gu (the "tăng lượng cổ" knob).
	RequiredGuQty int `json:"required_gu_qty"`
	// RequiredGuRankDelta is the rank offset of the predecessor Gu (default -1).
	RequiredGuRankDelta int `json:"required_gu_rank_delta"`
	// RequiredGu maps a target Gu id to the explicit predecessor Gu id. Targets
	// absent from this map get an empty required_gus (author must fill it).
	RequiredGu map[string]string `json:"required_gu"`
}

func defaultConfig() config {
	return config{
		Balance: balanceConfig{
			Tol:          0.25,
			MinScoreFrac: 0.01,
			Weights: map[string]float64{
				"damage": 1, "shield": 1, "heal": 1, "direct_hp": 1, "stat": 1, "status": 1,
			},
		},
		Recipe: recipeConfig{
			SuccessRate:         map[string]float64{"1": 0.81, "2": 0.8, "3": 0.7, "4": 0.6, "5": 0.5, "6": 0.05, "7": 0.01, "8": 0.0001, "9": 0.00001},
			PrimevalCost:        map[string]float64{"1": 20, "2": 40, "3": 80, "4": 150, "5": 300, "6": 600, "7": 0, "8": 0, "9": 0},
			ImmortalCost:        map[string]float64{"1": 0, "2": 0, "3": 0, "4": 0, "5": 0, "6": 100, "7": 200, "8": 500, "9": 1600},
			MaterialQty:         10,
			MaterialQtyScale:    1.0,
			MaterialCount:       2,
			RequiredGuQty:       1,
			RequiredGuRankDelta: -1,
			RequiredGu:          map[string]string{},
		},
	}
}

func loadConfig(path string) (config, error) {
	cfg := defaultConfig()
	if path == "" {
		return cfg, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	// Guard against nil maps after partial decode.
	if cfg.Balance.Weights == nil {
		cfg.Balance.Weights = defaultConfig().Balance.Weights
	}
	if cfg.Recipe.SuccessRate == nil {
		cfg.Recipe.SuccessRate = defaultConfig().Recipe.SuccessRate
	}
	if cfg.Recipe.PrimevalCost == nil {
		cfg.Recipe.PrimevalCost = defaultConfig().Recipe.PrimevalCost
	}
	if cfg.Recipe.ImmortalCost == nil {
		cfg.Recipe.ImmortalCost = defaultConfig().Recipe.ImmortalCost
	}
	if cfg.Recipe.RequiredGu == nil {
		cfg.Recipe.RequiredGu = map[string]string{}
	}
	if cfg.Balance.Tol <= 0 {
		cfg.Balance.Tol = 0.25
	}
	if cfg.Balance.MinScoreFrac <= 0 {
		cfg.Balance.MinScoreFrac = 0.01
	}
	if cfg.Recipe.MaterialQty <= 0 {
		cfg.Recipe.MaterialQty = 10
	}
	if cfg.Recipe.MaterialCount <= 0 {
		cfg.Recipe.MaterialCount = 2
	}
	if cfg.Recipe.RequiredGuQty < 0 {
		cfg.Recipe.RequiredGuQty = 0
	}
	return cfg, nil
}
