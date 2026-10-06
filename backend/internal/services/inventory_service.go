package services

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

// InventoryItem represents a single item entry in a player's inventory.
type InventoryItem struct {
	PlayerID  string    `json:"player_id"`
	ItemID    string    `json:"item_id"`
	Category  string    `json:"category"`
	Quantity  int64     `json:"quantity"`
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

// GetQuantity returns the total quantity of a specific item across categories, or 0 if not found.
func (s *InventoryService) GetQuantity(q QueryRower, playerID, itemID string) (int64, error) {
	var qty int64
	err := q.QueryRow(`
		SELECT COALESCE(SUM(quantity), 0) FROM player_inventory_items
		WHERE player_id = $1 AND item_id = $2
	`, playerID, itemID).Scan(&qty)
	if err != nil {
		return 0, err
	}
	return qty, nil
}

// GetQuantityInAnyCategory checks multiple categories in a single query.
// Returns (quantity, foundCategory, error).
func (s *InventoryService) GetQuantityInAnyCategory(q QueryRower, playerID, itemID string, categories ...string) (int64, string, error) {
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

	var qty int64
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

// ItemBatchEntry holds information for batch insertion.
type ItemBatchEntry struct {
	ItemID   string
	Category string
	Quantity int64
}

// AddItemsBatch inserts or increments multiple items for a player in a single SQL query.
func (s *InventoryService) AddItemsBatch(tx *sql.Tx, playerID string, items []ItemBatchEntry) error {
	if len(items) == 0 {
		return nil
	}
	if len(items) == 1 {
		return s.AddItem(tx, playerID, items[0].ItemID, items[0].Category, items[0].Quantity)
	}

	valueStrings := make([]string, 0, len(items))
	valueArgs := make([]any, 0, len(items)*3+1)
	valueArgs = append(valueArgs, playerID)

	validCount := 0
	for _, entry := range items {
		if entry.Quantity <= 0 {
			continue
		}
		baseIdx := validCount*3 + 2
		valueStrings = append(valueStrings, fmt.Sprintf("($1, $%d, $%d, $%d, NOW())", baseIdx, baseIdx+1, baseIdx+2))
		valueArgs = append(valueArgs, entry.ItemID, entry.Category, entry.Quantity)
		validCount++
	}

	if len(valueStrings) == 0 {
		return nil
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

// execWithSavepoint runs fn on tx inside a PostgreSQL SAVEPOINT so a failed
// statement rolls back only itself instead of poisoning the whole transaction
// with "current transaction is aborted" (25P02) on the next statement. Many
// callers treat inventory writes as best-effort (log and continue), so the
// error is still returned but the tx remains usable. No-op when tx is nil.
func execWithSavepoint(tx *sql.Tx, fn func() error) error {
	if tx == nil {
		return fn()
	}
	if _, err := tx.Exec("SAVEPOINT sp_inventory"); err != nil {
		return err
	}
	if err := fn(); err != nil {
		if _, rbErr := tx.Exec("ROLLBACK TO SAVEPOINT sp_inventory"); rbErr != nil {
			return rbErr
		}
		if _, relErr := tx.Exec("RELEASE SAVEPOINT sp_inventory"); relErr != nil {
			return relErr
		}
		return err
	}
	_, err := tx.Exec("RELEASE SAVEPOINT sp_inventory")
	return err
}

func (s *InventoryService) AddItem(tx *sql.Tx, playerID, itemID, category string, quantity int64) error {
	return execWithSavepoint(tx, func() error {
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
	})
}

// RemoveItem decrements quantity and deletes the row if it reaches 0.
func (s *InventoryService) RemoveItem(tx *sql.Tx, playerID, itemID, category string, quantity int64) error {
	if tx == nil {
		return sql.ErrTxDone
	}
	return execWithSavepoint(tx, func() error {
		var current int64
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
			_, err = tx.Exec(`UPDATE player_inventory_items SET quantity = quantity - $1, updated_at = NOW() WHERE player_id = $2 AND item_id = $3 AND category = $4 AND quantity >= $1`, quantity, playerID, itemID, category)
		}
		return err
	})
}

// DeductItem decrements quantity across any inventory categories for a player in order of quantity descending.
// If total available quantity across all categories is less than required quantity, returns sql.ErrNoRows.
func (s *InventoryService) DeductItem(tx *sql.Tx, playerID, itemID string, quantity int64) error {
	if quantity <= 0 {
		return nil
	}

	var rows *sql.Rows
	var err error
	if tx != nil {
		rows, err = tx.Query(`
			SELECT category, quantity FROM player_inventory_items
			WHERE player_id = $1 AND item_id = $2 AND quantity > 0
			ORDER BY quantity DESC
			FOR UPDATE
		`, playerID, itemID)
	} else {
		rows, err = s.db.Query(`
			SELECT category, quantity FROM player_inventory_items
			WHERE player_id = $1 AND item_id = $2 AND quantity > 0
			ORDER BY quantity DESC
		`, playerID, itemID)
	}
	if err != nil {
		return err
	}
	defer rows.Close()

	type itemCategoryRow struct {
		category string
		quantity int64
	}
	var entries []itemCategoryRow
	var totalAvail int64
	for rows.Next() {
		var entry itemCategoryRow
		if err := rows.Scan(&entry.category, &entry.quantity); err != nil {
			return err
		}
		entries = append(entries, entry)
		totalAvail += entry.quantity
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	if totalAvail < quantity {
		return sql.ErrNoRows
	}

	remaining := quantity
	for _, entry := range entries {
		if remaining <= 0 {
			break
		}
		if entry.quantity <= remaining {
			if tx != nil {
				_, err = tx.Exec(`DELETE FROM player_inventory_items WHERE player_id = $1 AND item_id = $2 AND category = $3`, playerID, itemID, entry.category)
			} else {
				_, err = s.db.Exec(`DELETE FROM player_inventory_items WHERE player_id = $1 AND item_id = $2 AND category = $3`, playerID, itemID, entry.category)
			}
			remaining -= entry.quantity
		} else {
			if tx != nil {
				_, err = tx.Exec(`UPDATE player_inventory_items SET quantity = quantity - $1, updated_at = NOW() WHERE player_id = $2 AND item_id = $3 AND category = $4`, remaining, playerID, itemID, entry.category)
			} else {
				_, err = s.db.Exec(`UPDATE player_inventory_items SET quantity = quantity - $1, updated_at = NOW() WHERE player_id = $2 AND item_id = $3 AND category = $4`, remaining, playerID, itemID, entry.category)
			}
			remaining = 0
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// SetQuantity sets the exact quantity for an item (inserts or deletes if 0).
func (s *InventoryService) SetQuantity(tx *sql.Tx, playerID, itemID, category string, quantity int64) error {
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
// items maps itemID → total quantity to add. Large maps are chunked so one
// request never holds a long multi-row INSERT that blocks concurrent removes.
func (s *InventoryService) BulkAddItems(tx *sql.Tx, playerID, category string, items map[string]int64) error {
	if len(items) == 0 {
		return nil
	}

	ids := make([]string, 0, len(items))
	qtys := make([]int64, 0, len(items))
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

	const chunkSize = 50
	for start := 0; start < len(ids); start += chunkSize {
		end := start + chunkSize
		if end > len(ids) {
			end = len(ids)
		}
		if err := s.bulkAddChunk(tx, playerID, category, ids[start:end], qtys[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func (s *InventoryService) bulkAddChunk(tx *sql.Tx, playerID, category string, ids []string, qtys []int64) error {

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

// BulkRemoveItems decrements multiple items in a single query using unnest.
// items maps itemID → total quantity to remove. Rows that would drop to zero
// are deleted. It requires tx to guarantee the read-modify-write is atomic.
func (s *InventoryService) BulkRemoveItems(tx *sql.Tx, playerID, category string, items map[string]int64) error {
	if len(items) == 0 {
		return nil
	}

	ids := make([]string, 0, len(items))
	qtys := make([]int64, 0, len(items))
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

	// First lock and verify quantities
	rows, err := tx.Query(`
		SELECT item_id, quantity FROM player_inventory_items
		WHERE player_id = $1 AND category = $2 AND item_id = ANY($3)
		ORDER BY item_id ASC
		FOR UPDATE
	`, playerID, category, pq.Array(ids))
	if err != nil {
		return err
	}
	defer rows.Close()

	avail := make(map[string]int64)
	for rows.Next() {
		var id string
		var q int64
		if err := rows.Scan(&id, &q); err != nil {
			return err
		}
		avail[id] = q
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for i, id := range ids {
		if avail[id] < qtys[i] {
			return sql.ErrNoRows
		}
	}

	// Decrement each row; rows with quantity <= 0 afterwards are deleted.
	_, err = tx.Exec(`
		UPDATE player_inventory_items AS p
		SET quantity = p.quantity - u.qty,
		    updated_at = NOW()
		FROM (
			SELECT unnest($1::text[]) AS item_id, unnest($2::bigint[]) AS qty
		) AS u(item_id, qty)
		WHERE p.player_id = $3 AND p.item_id = u.item_id AND p.category = $4 AND p.quantity >= u.qty
	`, pq.Array(ids), pq.Array(qtys), playerID, category)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		DELETE FROM player_inventory_items
		WHERE player_id = $1 AND category = $2 AND item_id = ANY($3) AND quantity <= 0
	`, playerID, category, pq.Array(ids))
	return err
}
