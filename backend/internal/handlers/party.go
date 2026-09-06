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
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

type PartyMember struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

func CreateParty(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player status"})
		return
	}

	if player.PartyID != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn đã ở trong một tổ đội rồi!"})
		return
	}

	partyID := uuid.New()
	members := []PartyMember{
		{ID: player.ID.String(), Username: player.Username, AvatarURL: player.AvatarURL},
	}
	membersBytes, _ := json.Marshal(members)

	_, err = tx.Exec("INSERT INTO parties (id, leader_id, members) VALUES ($1, $2, $3)", partyID, player.ID, membersBytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create party"})
		return
	}

	_, err = tx.Exec("UPDATE players SET party_id = $1 WHERE id = $2", partyID, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player party"})
		return
	}

	_, err = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 2, $1) WHERE id = $2", MaxHumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Nhân Khí"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	auditlog.Logf(db.DB, &authID, "party.create", "partyId=%s", partyID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Tạo tổ đội thành công! Nhân Khí +2.",
		"party": gin.H{
			"id":        partyID.String(),
			"leader_id": player.ID.String(),
			"members":   members,
		},
	})
}

func GetPartyStatus(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var partyIDStr sql.NullString
	err := db.DB.QueryRowContext(c.Request.Context(), "SELECT party_id FROM players WHERE id = $1", authID).Scan(&partyIDStr)
	if err != nil || !partyIDStr.Valid || partyIDStr.String == "" {
		c.JSON(http.StatusOK, gin.H{"party": nil})
		return
	}

	var leaderID string
	var membersRaw []byte
	err = db.DB.QueryRowContext(c.Request.Context(), "SELECT leader_id, members FROM parties WHERE id = $1", partyIDStr.String).Scan(&leaderID, &membersRaw)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"party": nil})
		return
	}

	var members []PartyMember
	json.Unmarshal(membersRaw, &members)

	c.JSON(http.StatusOK, gin.H{
		"party": gin.H{
			"id":        partyIDStr.String,
			"leader_id": leaderID,
			"members":   members,
		},
	})
}

type InviteRequest struct {
	Username string `json:"username" binding:"required"`
}

func InvitePlayer(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req InviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username is required"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player status"})
		return
	}

	if player.PartyID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn phải ở trong tổ đội để mời người khác!"})
		return
	}

	var leaderID string
	var membersRaw []byte
	err = tx.QueryRow("SELECT leader_id, members FROM parties WHERE id = $1 FOR UPDATE", player.PartyID).Scan(&leaderID, &membersRaw)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tổ đội"})
		return
	}

	if leaderID != player.ID.String() {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Trưởng nhóm mới có quyền mời!"})
		return
	}

	var targetPlayer models.Player
	var targetPartyID sql.NullString
	err = tx.QueryRow("SELECT id, username, party_id FROM players WHERE username = $1 FOR UPDATE", req.Username).Scan(&targetPlayer.ID, &targetPlayer.Username, &targetPartyID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi này"})
		return
	}

	if targetPartyID.Valid && targetPartyID.String != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Người chơi này đã có tổ đội rồi!"})
		return
	}

	var members []PartyMember
	json.Unmarshal(membersRaw, &members)

	if len(members) >= 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tổ đội đã đầy (tối đa 4 người)!"})
		return
	}

	// Insert pending invitation
	_, err = tx.Exec(`
		INSERT INTO party_invitations (party_id, inviter_id, invitee_id, status)
		VALUES ($1, $2, $3, 'PENDING')
		ON CONFLICT (party_id, invitee_id) DO UPDATE SET status = 'PENDING', created_at = CURRENT_TIMESTAMP
	`, player.PartyID, player.ID, targetPlayer.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gửi lời mời thất bại"})
		return
	}

	notify.Send(db.DB, targetPlayer.ID, "PARTY_INVITE", "Lời Mời Tổ Đội", fmt.Sprintf("Cổ sư %s đã mời bạn tham gia tổ đội!", player.Username))

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Đã gửi lời mời tham gia tổ đội đến người chơi %s!", targetPlayer.Username),
		"party": gin.H{
			"id":        player.PartyID.String(),
			"leader_id": leaderID,
			"members":   members,
		},
	})
}

func LeaveParty(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player"})
		return
	}

	if player.PartyID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn không ở trong tổ đội nào."})
		return
	}

	partyID := player.PartyID.String()
	var leaderID string
	var membersRaw []byte
	err = tx.QueryRow("SELECT leader_id, members FROM parties WHERE id = $1 FOR UPDATE", partyID).Scan(&leaderID, &membersRaw)
	if err != nil {
		_, _ = tx.Exec("UPDATE players SET party_id = NULL WHERE id = $1", player.ID)
		tx.Commit()
		c.JSON(http.StatusOK, gin.H{"message": "Đã rời tổ đội"})
		return
	}

	var members []PartyMember
	json.Unmarshal(membersRaw, &members)

	newMembers := []PartyMember{}
	for _, m := range members {
		if m.ID != player.ID.String() {
			newMembers = append(newMembers, m)
		}
	}

	_, err = tx.Exec("UPDATE players SET party_id = NULL WHERE id = $1", player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database update failed"})
		return
	}

	if len(newMembers) == 0 {
		_, _ = tx.Exec("DELETE FROM parties WHERE id = $1", partyID)
	} else {
		newLeaderID := leaderID
		if leaderID == player.ID.String() {
			newLeaderID = newMembers[0].ID
		}
		newMembersBytes, _ := json.Marshal(newMembers)
		_, _ = tx.Exec("UPDATE parties SET leader_id = $1, members = $2 WHERE id = $3", newLeaderID, newMembersBytes, partyID)
	}

	for _, m := range newMembers {
		if mUUID, errM := uuid.Parse(m.ID); errM == nil {
			notify.Send(db.DB, mUUID, "PARTY_LEAVE", "Cập Nhật Tổ Đội", fmt.Sprintf("Cổ sư %s đã rời khỏi tổ đội.", player.Username))
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	auditlog.Logf(db.DB, &authID, "party.leave", "partyId=%s", partyID)

	c.JSON(http.StatusOK, gin.H{"message": "Rời tổ đội thành công!"})
}

func GetPendingInvitations(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT i.id, i.party_id, p.username AS inviter_name, p.avatar_url AS inviter_avatar
		FROM party_invitations i
		JOIN players p ON i.inviter_id = p.id
		WHERE i.invitee_id = $1 AND i.status = 'PENDING'
	`, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	type InvInfo struct {
		ID            string `json:"id"`
		PartyID       string `json:"party_id"`
		InviterName   string `json:"inviter_name"`
		InviterAvatar string `json:"inviter_avatar,omitempty"`
	}
	var list []InvInfo
	for rows.Next() {
		var inv InvInfo
		var avatar sql.NullString
		if errScan := rows.Scan(&inv.ID, &inv.PartyID, &inv.InviterName, &avatar); errScan == nil {
			if avatar.Valid {
				inv.InviterAvatar = avatar.String
			}
			list = append(list, inv)
		}
	}
	c.JSON(http.StatusOK, gin.H{"invitations": list})
}

type RespondInvitationRequest struct {
	Action string `json:"action" binding:"required"`
}

func RespondToInvitation(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	invID := c.Param("id")

	var req RespondInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Action is required"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var partyID uuid.UUID
	var status string
	err = tx.QueryRow("SELECT party_id, status FROM party_invitations WHERE id = $1 AND invitee_id = $2 FOR UPDATE", invID, authID).Scan(&partyID, &status)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy lời mời này"})
		return
	}
	if status != "PENDING" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lời mời này đã được xử lý"})
		return
	}

	if req.Action == "reject" {
		_, _ = tx.Exec("UPDATE party_invitations SET status = 'REJECTED' WHERE id = $1", invID)
		tx.Commit()
		c.JSON(http.StatusOK, gin.H{"message": "Đã từ chối lời mời"})
		return
	}

	if req.Action != "accept" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hành động không hợp lệ"})
		return
	}

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải thông tin nhân vật"})
		return
	}
	if player.PartyID != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đã có tổ đội rồi!"})
		return
	}

	var leaderID string
	var membersRaw []byte
	err = tx.QueryRow("SELECT leader_id, members FROM parties WHERE id = $1 FOR UPDATE", partyID).Scan(&leaderID, &membersRaw)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tổ đội"})
		return
	}

	var members []PartyMember
	json.Unmarshal(membersRaw, &members)
	if len(members) >= 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tổ đội đã đầy (tối đa 4 người)!"})
		return
	}

	members = append(members, PartyMember{ID: player.ID.String(), Username: player.Username, AvatarURL: player.AvatarURL})
	newMembersBytes, _ := json.Marshal(members)

	_, err = tx.Exec("UPDATE parties SET members = $1 WHERE id = $2", newMembersBytes, partyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật tổ đội"})
		return
	}

	var leaderRegion sql.NullString
	var leaderX, leaderY int
	_ = tx.QueryRow("SELECT current_region, x, y FROM players WHERE id = $1", leaderID).Scan(&leaderRegion, &leaderX, &leaderY)

	_, err = tx.Exec("UPDATE players SET party_id = $1, current_region = $2, x = $3, y = $4 WHERE id = $5", partyID, leaderRegion, leaderX, leaderY, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật thông tin nhân vật"})
		return
	}

	_, err = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 2, $1) WHERE id = $2", MaxHumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Nhân Khí"})
		return
	}

	// Update invitation status to ACCEPTED and clean up other pending ones
	_, _ = tx.Exec("UPDATE party_invitations SET status = 'ACCEPTED' WHERE id = $1", invID)
	_, _ = tx.Exec("DELETE FROM party_invitations WHERE invitee_id = $1 AND status = 'PENDING'", player.ID)

	if leaderUUID, errL := uuid.Parse(leaderID); errL == nil {
		notify.Send(db.DB, leaderUUID, "PARTY_JOIN", "Thành Viên Mới", fmt.Sprintf("Cổ sư %s đã chấp nhận lời mời và gia nhập tổ đội!", player.Username))
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database commit error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Đã chấp nhận lời mời tham gia tổ đội! Nhân Khí +2.",
		"party": gin.H{
			"id":        partyID.String(),
			"leader_id": leaderID,
			"members":   members,
		},
	})
}

func checkPartyLeaderOrSolo(tx *sql.Tx, player *models.Player) error {
	if player.PartyID == nil {
		return nil
	}
	var isLeader bool
	err := tx.QueryRow("SELECT (leader_id = $1) FROM parties WHERE id = $2", player.ID, player.PartyID).Scan(&isLeader)
	if err != nil {
		return fmt.Errorf("Không tìm thấy thông tin tổ đội")
	}
	if !isLeader {
		return fmt.Errorf("Chỉ có Trưởng nhóm mới có quyền thao tác!")
	}
	return nil
}

func SyncPartyMembersLocation(tx *sql.Tx, leader *models.Player) {
	if leader.PartyID == nil {
		return
	}
	var isLeader bool
	err := tx.QueryRow("SELECT (leader_id = $1) FROM parties WHERE id = $2", leader.ID, leader.PartyID).Scan(&isLeader)
	if err == nil && isLeader {
		_, _ = tx.Exec("UPDATE players SET current_region = $1, x = $2, y = $3 WHERE party_id = $4 AND id != $5", leader.CurrentRegion, leader.X, leader.Y, leader.PartyID, leader.ID)
	}
}

