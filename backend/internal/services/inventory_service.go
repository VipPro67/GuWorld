package services

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// InventoryItem represents a single item entry in a player's inventory.
type InventoryItem struct {
	PlayerID  string    `json:"player_id"`
	ItemID    string    `json:"item_id"`
	Category  string    `json:"category"`
	Quantity  int       `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// InventoryService handles player inventory persistence.
type InventoryService struct {
	db *sql.DB
}

func NewInventoryService(db *sql.DB) *InventoryService {
	return &InventoryService{db: db}
}

// ListByPlayer returns all inventory items for a player.
func (s *InventoryService) ListByPlayer(playerID string) ([]InventoryItem, error) {
	rows, err := s.db.Query(`
		SELECT player_id, item_id, category, quantity, created_at, updated_at
		FROM player_inventory_items
		WHERE player_id = $1
		ORDER BY category, item_id
	`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []InventoryItem
	for rows.Next() {
		var it InventoryItem
		if err := rows.Scan(&it.PlayerID, &it.ItemID, &it.Category, &it.Quantity, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, it)
	}
	return results, rows.Err()
}

// GetQuantity returns the quantity of a specific item, or 0 if not found.
func (s *InventoryService) GetQuantity(q QueryRower, playerID, itemID, category string) (int, error) {
	var qty int
	err := q.QueryRow(`
		SELECT quantity FROM player_inventory_items
		WHERE player_id = $1 AND item_id = $2 AND category = $3
	`, playerID, itemID, category).Scan(&qty)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return qty, nil
}

// GetQuantityInAnyCategory checks multiple categories in a single query.
// Returns (quantity, foundCategory, error).
func (s *InventoryService) GetQuantityInAnyCategory(q QueryRower, playerID, itemID string, categories ...string) (int, string, error) {
	if len(categories) == 0 {
		return 0, "", nil
	}
	placeholders := make([]string, len(categories))
	args := []interface{}{playerID, itemID}
	for i, cat := range categories {
		placeholders[i] = fmt.Sprintf("$%d", i+3)
		args = append(args, cat)
	}
	query := fmt.Sprintf(`
		SELECT quantity, category FROM player_inventory_items
		WHERE player_id = $1 AND item_id = $2 AND category IN (%s)
		LIMIT 1
	`, strings.Join(placeholders, ","))

	var qty int
	var cat string
	err := q.QueryRow(query, args...).Scan(&qty, &cat)
	if err == sql.ErrNoRows {
		return 0, "", nil
	}
	if err != nil {
		return 0, "", err
	}
	return qty, cat, nil
}

// AddItem adds quantity to an existing item or inserts a new row.
// If tx is nil, operates directly on the DB.
func (s *InventoryService) AddItem(tx *sql.Tx, playerID, itemID, category string, quantity int) error {
	var err error
	if tx != nil {
		_, err = tx.Exec(`
			INSERT INTO player_inventory_items (player_id, item_id, category, quantity, updated_at)
			VALUES ($1, $2, $3, $4, NOW())
			ON CONFLICT (player_id, item_id, category)
			DO UPDATE SET quantity = player_inventory_items.quantity + EXCLUDED.quantity, updated_at = NOW()
		`, playerID, itemID, category, quantity)
	} else {
		_, err = s.db.Exec(`
			INSERT INTO player_inventory_items (player_id, item_id, category, quantity, updated_at)
			VALUES ($1, $2, $3, $4, NOW())
			ON CONFLICT (player_id, item_id, category)
			DO UPDATE SET quantity = player_inventory_items.quantity + EXCLUDED.quantity, updated_at = NOW()
		`, playerID, itemID, category, quantity)
	}
	return err
}

// RemoveItem decrements quantity and deletes the row if it reaches 0.
func (s *InventoryService) RemoveItem(tx *sql.Tx, playerID, itemID, category string, quantity int) error {
	var current int
	err := tx.QueryRow(`
		SELECT quantity FROM player_inventory_items
		WHERE player_id = $1 AND item_id = $2 AND category = $3
		FOR UPDATE
	`, playerID, itemID, category).Scan(&current)
	if err != nil {
		return err
	}
	if current < quantity {
		return sql.ErrNoRows
	}
	if current == quantity {
		_, err = tx.Exec(`DELETE FROM player_inventory_items WHERE player_id = $1 AND item_id = $2 AND category = $3`, playerID, itemID, category)
	} else {
		_, err = tx.Exec(`UPDATE player_inventory_items SET quantity = quantity - $1, updated_at = NOW() WHERE player_id = $2 AND item_id = $3 AND category = $4`, quantity, playerID, itemID, category)
	}
	return err
}

// SetQuantity sets the exact quantity for an item (inserts or deletes if 0).
func (s *InventoryService) SetQuantity(tx *sql.Tx, playerID, itemID, category string, quantity int) error {
	if quantity <= 0 {
		_, err := tx.Exec(`DELETE FROM player_inventory_items WHERE player_id = $1 AND item_id = $2 AND category = $3`, playerID, itemID, category)
		return err
	}
	_, err := tx.Exec(`
		INSERT INTO player_inventory_items (player_id, item_id, category, quantity, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (player_id, item_id, category)
		DO UPDATE SET quantity = EXCLUDED.quantity, updated_at = NOW()
	`, playerID, itemID, category, quantity)
	return err
}

// DeleteAll removes all inventory rows for a player (e.g. on delete).
func (s *InventoryService) DeleteAll(tx *sql.Tx, playerID string) error {
	_, err := tx.Exec(`DELETE FROM player_inventory_items WHERE player_id = $1`, playerID)
	return err
}

// BulkAddItems upserts multiple items in a single query using unnest.
// items maps itemID → total quantity to add.
func (s *InventoryService) BulkAddItems(tx *sql.Tx, playerID, category string, items map[string]int) error {
	if len(items) == 0 {
		return nil
	}

	ids := make([]string, 0, len(items))
	qtys := make([]int, 0, len(items))
	for id, qty := range items {
		if qty <= 0 {
			continue
		}
		ids = append(ids, id)
		qtys = append(qtys, qty)
	}
	if len(ids) == 0 {
		return nil
	}

	// Build a VALUES list for INSERT ... ON CONFLICT
	valueStrings := make([]string, len(ids))
	valueArgs := make([]interface{}, 0, len(ids)*4)
	argIdx := 1
	for i, id := range ids {
		valueStrings[i] = fmt.Sprintf("($%d, $%d, $%d, $%d, NOW())", argIdx, argIdx+1, argIdx+2, argIdx+3)
		valueArgs = append(valueArgs, playerID, id, category, qtys[i])
		argIdx += 4
	}

	query := fmt.Sprintf(`
		INSERT INTO player_inventory_items (player_id, item_id, category, quantity, updated_at)
		VALUES %s
		ON CONFLICT (player_id, item_id, category)
		DO UPDATE SET quantity = player_inventory_items.quantity + EXCLUDED.quantity, updated_at = NOW()
	`, strings.Join(valueStrings, ", "))

	var err error
	if tx != nil {
		_, err = tx.Exec(query, valueArgs...)
	} else {
		_, err = s.db.Exec(query, valueArgs...)
	}
	return err
}
