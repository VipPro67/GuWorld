package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

// PartyFormationEntry is one lineup slot after the leader: a party member
// player, or one of the leader's non-player actors (pet / variant NPC).
// Pos is the 1..8 combat grid position (allySlots index); 0 = auto-fill
// lowest free cell in entry order.
type PartyFormationEntry struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Pos  int    `json:"pos,omitempty"`
}

// MaxPartyFollowers caps the lineup after the leader (9 grid slots total).
const MaxPartyFollowers = 8

// normalizePartyFormation validates and dedupes a leader-submitted lineup.
// The leader themself is always Allies[0] so their own player entry is dropped.
// Pure function (no DB) for unit testing.
func normalizePartyFormation(entries []PartyFormationEntry, leaderID string) ([]PartyFormationEntry, error) {
	seen := make(map[string]bool, len(entries))
	out := make([]PartyFormationEntry, 0, len(entries))
	for _, e := range entries {
		switch e.Kind {
		case "player", "pet", "npc":
		default:
			return nil, fmt.Errorf("kind %q không hợp lệ (player/pet/npc)", e.Kind)
		}
		if _, err := uuid.Parse(e.ID); err != nil {
			return nil, fmt.Errorf("id %q không phải UUID", e.ID)
		}
		if e.Kind == "player" && e.ID == leaderID {
			continue // leader luôn đứng đầu, khỏi xếp
		}
		key := e.Kind + ":" + e.ID
		if seen[key] {
			continue
		}
		if e.Pos < 0 || e.Pos > MaxPartyFollowers {
			return nil, fmt.Errorf("ô %d không hợp lệ (1-8, 0 = tự xếp)", e.Pos)
		}
		seen[key] = true
		out = append(out, e)
	}
	if len(out) > MaxPartyFollowers {
		return nil, fmt.Errorf("đội hình tối đa %d vị trí sau trưởng nhóm", MaxPartyFollowers)
	}
	used := make(map[int]bool, len(out))
	for _, e := range out {
		if e.Pos == 0 {
			continue
		}
		if used[e.Pos] {
			return nil, fmt.Errorf("ô %d bị xếp trùng", e.Pos)
		}
		used[e.Pos] = true
	}
	return out, nil
}

// loadPartyFormation reads the raw formation JSON of a party.
func loadPartyFormation(tx *sql.Tx, partyID string) ([]PartyFormationEntry, error) {
	var raw []byte
	if err := tx.QueryRow(`SELECT COALESCE(formation, '[]') FROM parties WHERE id = $1`, partyID).Scan(&raw); err != nil {
		return nil, err
	}
	var entries []PartyFormationEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return []PartyFormationEntry{}, nil
	}
	if entries == nil {
		return []PartyFormationEntry{}, nil
	}
	return entries, nil
}

// partyStatusPayload builds the party object returned by status/mutation
// endpoints, including the leader-arranged battle formation.
func partyStatusPayload(q db.Queryer, partyID string) (map[string]interface{}, error) {
	var leaderID string
	var membersRaw, formationRaw []byte
	if err := q.QueryRow(`SELECT leader_id, members, COALESCE(formation, '[]') FROM parties WHERE id = $1`, partyID).Scan(&leaderID, &membersRaw, &formationRaw); err != nil {
		return nil, err
	}
	var members []PartyMember
	_ = json.Unmarshal(membersRaw, &members)
	if members == nil {
		members = []PartyMember{}
	}
	var formation []PartyFormationEntry
	_ = json.Unmarshal(formationRaw, &formation)
	if formation == nil {
		formation = []PartyFormationEntry{}
	}
	return map[string]interface{}{
		"id":        partyID,
		"leader_id": leaderID,
		"members":   members,
		"formation": formation,
	}, nil
}

// mustPartyFormation returns the formation for payloads, tolerating errors
// with an empty lineup (payload paths must never 500 on formation reads).
func mustPartyFormation(tx *sql.Tx, partyID string) []PartyFormationEntry {
	entries, err := loadPartyFormation(tx, partyID)
	if err != nil {
		return []PartyFormationEntry{}
	}
	return entries
}

type saveFormationRequest struct {
	Order []PartyFormationEntry `json:"order" binding:"required"`
}

// SavePartyFormation lets the party leader arrange the battle lineup:
// member players plus the leader's own pets / variant NPCs, in join order.
// Empty order restores legacy auto behavior.
func SavePartyFormation(c *gin.Context) {
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req saveFormationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đội hình không hợp lệ"})
		return
	}
	if req.Order == nil {
		req.Order = []PartyFormationEntry{}
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải nhân vật"})
		return
	}
	if player.PartyID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn chưa ở trong tổ đội"})
		return
	}
	var leaderID string
	var membersRaw []byte
	if err := tx.QueryRow(`SELECT leader_id, members FROM parties WHERE id = $1 FOR UPDATE`, player.PartyID).Scan(&leaderID, &membersRaw); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tổ đội"})
		return
	}
	if leaderID != player.ID.String() {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ Trưởng nhóm mới được xếp đội hình"})
		return
	}

	entries, err := normalizePartyFormation(req.Order, player.ID.String())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var members []PartyMember
	_ = json.Unmarshal(membersRaw, &members)
	memberSet := make(map[string]bool, len(members))
	for _, m := range members {
		memberSet[m.ID] = true
	}
	for _, e := range entries {
		switch e.Kind {
		case "player":
			if !memberSet[e.ID] {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Người chơi %s không ở trong tổ đội", e.ID)})
				return
			}
		case "pet":
			var ok bool
			if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM non_player_actors WHERE id = $1 AND owner_player_id = $2 AND actor_kind = 'COMPANION_BEAST' AND lifecycle_state = 'ACTIVE')`, e.ID, player.ID).Scan(&ok); err != nil || !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Thú %s không thuộc về bạn", e.ID)})
				return
			}
		case "npc":
			var ok bool
			if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM non_player_actors WHERE id = $1 AND owner_player_id = $2 AND actor_kind = 'VARIANT_NPC' AND lifecycle_state = 'ACTIVE')`, e.ID, player.ID).Scan(&ok); err != nil || !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Dị nhân %s không thuộc về bạn", e.ID)})
				return
			}
		}
	}

	formationBytes, _ := json.Marshal(entries)
	if _, err := tx.Exec(`UPDATE parties SET formation = $1 WHERE id = $2`, formationBytes, player.PartyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu đội hình"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Đã lưu đội hình xuất trận", "formation": entries})
}
