package handlers

import (
	"database/sql"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func loadKillerMoves(playerID uuid.UUID) ([]models.KillerMove, error) {
	rows, err := db.DB.Query(`SELECT id, player_id, name, path, move_type, rank_required, core_player_gu_id, total_mana_cost, success_rate, damage_value, defense_value, cooldown, channel_time, effects, is_mastered, is_reliable, is_instant_cast, created_at, updated_at FROM killer_moves WHERE player_id = $1 ORDER BY updated_at DESC`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	moves := []models.KillerMove{}
	for rows.Next() {
		m, err := scanKillerMove(rows)
		if err != nil {
			return nil, err
		}
		moves = append(moves, m)
	}
	for i := range moves {
		moves[i].Slots, _ = loadKillerMoveSlots(moves[i].ID)
	}
	return moves, rows.Err()
}

func loadKillerMove(playerID uuid.UUID, id string) (models.KillerMove, error) {
	row := db.DB.QueryRow(`SELECT id, player_id, name, path, move_type, rank_required, core_player_gu_id, total_mana_cost, success_rate, damage_value, defense_value, cooldown, channel_time, effects, is_mastered, is_reliable, is_instant_cast, created_at, updated_at FROM killer_moves WHERE id = $1 AND player_id = $2`, id, playerID)
	m, err := scanKillerMove(row)
	if err != nil {
		return m, err
	}
	m.Slots, _ = loadKillerMoveSlots(m.ID)
	return m, nil
}

func loadKillerMoveTx(tx *sql.Tx, playerID uuid.UUID, id string) (models.KillerMove, []models.KillerMoveSlot, error) {
	row := tx.QueryRow(`SELECT id, player_id, name, path, move_type, rank_required, core_player_gu_id, total_mana_cost, success_rate, damage_value, defense_value, cooldown, channel_time, effects, is_mastered, is_reliable, is_instant_cast, created_at, updated_at FROM killer_moves WHERE id = $1 AND player_id = $2 FOR UPDATE`, id, playerID)
	m, err := scanKillerMove(row)
	if err != nil {
		return m, nil, err
	}
	rows, err := tx.Query(`SELECT id, killer_move_id, player_gu_id, slot_type, slot_index, created_at FROM killer_move_slots WHERE killer_move_id = $1 ORDER BY slot_index ASC`, m.ID)
	if err != nil {
		return m, nil, err
	}
	defer rows.Close()
	slots, err := scanKillerMoveSlots(rows)
	return m, slots, err
}

type rowScanner interface{ Scan(dest ...any) error }

func scanKillerMove(row rowScanner) (models.KillerMove, error) {
	var m models.KillerMove
	err := row.Scan(&m.ID, &m.PlayerID, &m.Name, &m.Path, &m.MoveType, &m.RankRequired, &m.CorePlayerGuID, &m.TotalManaCost, &m.SuccessRate, &m.DamageValue, &m.DefenseValue, &m.Cooldown, &m.ChannelTime, &m.Effects, &m.IsMastered, &m.IsReliable, &m.IsInstantCast, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}

func loadKillerMoveSlots(moveID string) ([]models.KillerMoveSlot, error) {
	rows, err := db.DB.Query(`SELECT id, killer_move_id, player_gu_id, slot_type, slot_index, created_at FROM killer_move_slots WHERE killer_move_id = $1 ORDER BY slot_index ASC`, moveID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKillerMoveSlots(rows)
}

func scanKillerMoveSlots(rows *sql.Rows) ([]models.KillerMoveSlot, error) {
	slots := []models.KillerMoveSlot{}
	for rows.Next() {
		var s models.KillerMoveSlot
		if err := rows.Scan(&s.ID, &s.KillerMoveID, &s.PlayerGuID, &s.SlotType, &s.SlotIndex, &s.CreatedAt); err != nil {
			return nil, err
		}
		slots = append(slots, s)
	}
	return slots, rows.Err()
}

func insertKillerMoveSlotsTx(tx *sql.Tx, moveID, coreID string, supportIDs []string) error {
	_, err := tx.Exec(`INSERT INTO killer_move_slots (id, killer_move_id, player_gu_id, slot_type, slot_index) VALUES ($1,$2,$3,$4,0)`, uuid.New().String(), moveID, coreID, models.KillerMoveSlotCore)
	if err != nil {
		return err
	}
	for idx, id := range supportIDs {
		_, err = tx.Exec(`INSERT INTO killer_move_slots (id, killer_move_id, player_gu_id, slot_type, slot_index) VALUES ($1,$2,$3,$4,$5)`, uuid.New().String(), moveID, id, models.KillerMoveSlotSupport, idx+1)
		if err != nil {
			return err
		}
	}
	return nil
}
