package handlers

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

// settleArenaRankSwap swaps Võ Đài ranks when the player beats a higher-ranked
// opponent. It is the only place combat settlement writes arena_rankings, so the
// orchestration never embeds ranking SQL. Returns the rank-up log line, or nil
// when no swap happened (target not ranked, or the player was already better).
func settleArenaRankSwap(tx *sql.Tx, playerID, targetID uuid.UUID, turn int) *simulator.CombatLog {
	var playerRank, targetRank int
	if err := tx.QueryRow("SELECT rank FROM arena_rankings WHERE player_id = $1", playerID).Scan(&playerRank); err != nil {
		return nil
	}
	if err := tx.QueryRow("SELECT rank FROM arena_rankings WHERE player_id = $1", targetID).Scan(&targetRank); err != nil {
		return nil
	}
	if playerRank <= targetRank {
		return nil
	}

	_, _ = tx.Exec("UPDATE arena_rankings SET rank = -1 WHERE player_id = $1", playerID)
	_, _ = tx.Exec("UPDATE arena_rankings SET rank = $1 WHERE player_id = $2", playerRank, targetID)
	_, _ = tx.Exec("UPDATE arena_rankings SET rank = $1 WHERE player_id = $2", targetRank, playerID)
	_, _ = tx.Exec("UPDATE players SET arena_rank = $1 WHERE id = $2", playerRank, targetID)
	_, _ = tx.Exec("UPDATE players SET arena_rank = $1 WHERE id = $2", targetRank, playerID)

	return &simulator.CombatLog{
		Turn:   turn,
		Action: "ARENA_RANK_UP",
		Detail: fmt.Sprintf("Chúc mừng! Đạo hữu đã vượt cấp khiêu chiến thành công, thăng lên Hạng %d Võ Đài!", targetRank),
	}
}
