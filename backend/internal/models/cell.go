package models

// Cell represents a cell on the grid map
type Cell struct {
	X       int    `json:"x"`
	Y       int    `json:"y"`
	Type    string `json:"cell_type"` // "Empty", "Resource", "Monster", "WildGu", "PVP", "Boss"
	Content string `json:"content"`   // name/ID of item, beast, or Gu
}
