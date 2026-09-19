package models

type TradeOfferData struct {
	PrimevalStones float64            `json:"primeval_stones"`
	ImmortalStones float64            `json:"immortal_stones"`
	Gus            []string           `json:"gus"`       // slice of Gu instance IDs (for offered) or Gu IDs (for requested)
	Materials      map[string]int64   `json:"materials"` // map of itemID -> quantity
	Recipes        map[string]float64 `json:"recipes"`   // map of recipeID -> completion_rate
}
