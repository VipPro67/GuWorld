package handlers

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// BanSpec describes one moderation application. Until.Valid=false means the ban
// is permanent; a valid time is enforced lazily at request/login time (no cron).
type BanSpec struct {
	Reason      string
	Until       sql.NullTime
	BannedBy    uuid.UUID // uuid.Nil for system-generated bans
	BlockDevice bool
	IP          string
	Fingerprint string
}

const banActiveMessage = "Tài khoản đã bị khóa. Vui lòng liên hệ bộ phận hỗ trợ."
const chatMutedMessage = "Đạo hữu đang bị hạn chế gửi tin nhắn."

// banIsActive reports whether a ban/expiry pair is currently in force.
func banIsActive(isBanned bool, until sql.NullTime) bool {
	return isBanned && (!until.Valid || until.Time.After(time.Now()))
}

// ApplyPlayerBan is the single write path for bans. Admin handlers and the
// automatic anti-abuse rules both call it so session rotation, device blocks
// and notifications cannot drift between the two.
func ApplyPlayerBan(ctx context.Context, playerID uuid.UUID, spec BanSpec) error {
	if playerID == uuid.Nil {
		return errors.New("ban: player id required")
	}
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var until interface{}
	if spec.Until.Valid {
		until = spec.Until.Time
	}
	var bannedBy interface{}
	if spec.BannedBy != uuid.Nil {
		bannedBy = spec.BannedBy
	}

	res, err := tx.ExecContext(ctx, `
		UPDATE players
		SET is_banned = true, ban_reason = $2, banned_until = $3, banned_by = $4, banned_at = NOW(),
		    active_session_id = gen_random_uuid()::text
		WHERE id = $1`, playerID, spec.Reason, until, bannedBy)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}

	if spec.BlockDevice && (spec.IP != "" || spec.Fingerprint != "") {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO player_access_blocks (ip, fingerprint, reason, banned_by, banned_until)
			VALUES ($1, $2, $3, $4, $5)`, spec.IP, spec.Fingerprint, spec.Reason, bannedBy, until); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	middleware.InvalidateSessionCache(playerID)
	return nil
}

// notifyPlayerBanned pushes the kick notification so an online player is told
// why they were disconnected (the rotated session does the actual ejecting).
func notifyPlayerBanned(playerID uuid.UUID, reason string) {
	content := "Tài khoản của đạo hữu đã bị khóa."
	if reason != "" {
		content += " Lý do: " + reason
	}
	_ = notify.Send(db.DB, playerID, "session.kicked", "Tài khoản đã bị khóa", content)
	_ = cache.PublishEventQuick("player:"+playerID.String(), PlayerRealtimeEvent{
		Type:           SessionEventKicked,
		Revision:       nextPlayerEventRevision(),
		ChangedDomains: []PlayerEventDomain{PlayerDomain},
	})
}

// playerBanStatus reads the current ban state for a login attempt.
func playerBanStatus(q playerQueryRower, playerID uuid.UUID) (bool, string, sql.NullTime, error) {
	var banned bool
	var reason string
	var until sql.NullTime
	err := q.QueryRow(`SELECT is_banned, COALESCE(ban_reason, ''), banned_until FROM players WHERE id=$1`, playerID).Scan(&banned, &reason, &until)
	return banned, reason, until, err
}

// deviceBlocked reports whether an IP/fingerprint is blocked from registering.
func deviceBlocked(q playerQueryRower, ip, fingerprint string) (bool, error) {
	ip = strings.TrimSpace(ip)
	fingerprint = strings.TrimSpace(fingerprint)
	if ip == "" && fingerprint == "" {
		return false, nil
	}
	var blocked bool
	err := q.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM player_access_blocks
			WHERE (banned_until IS NULL OR banned_until > NOW())
			  AND (($1 <> '' AND fingerprint = $1) OR ($2 <> '' AND ip = $2))
		)`, fingerprint, ip).Scan(&blocked)
	return blocked, err
}

// playerChatMuted reports whether the player is currently muted.
func playerChatMuted(q playerQueryRower, playerID uuid.UUID) (bool, error) {
	var muted bool
	var until sql.NullTime
	if err := q.QueryRow(`SELECT is_chat_muted, chat_muted_until FROM players WHERE id=$1`, playerID).Scan(&muted, &until); err != nil {
		return false, err
	}
	return banIsActive(muted, until), nil
}

func applyChatMute(ctx context.Context, playerID uuid.UUID, muted bool, until sql.NullTime) error {
	var untilVal interface{}
	if until.Valid {
		untilVal = until.Time
	}
	_, err := db.DB.ExecContext(ctx, `
		UPDATE players SET is_chat_muted=$2, chat_muted_until=$3 WHERE id=$1`, playerID, muted, untilVal)
	return err
}

// banUntilFromRequest converts the admin duration/permanent flags to a deadline.
// Permanent => invalid NullTime (no expiry). Temporary => now + hours.
func banUntilFromRequest(durationHours int, permanent bool) (sql.NullTime, bool) {
	if permanent {
		return sql.NullTime{}, true
	}
	if durationHours <= 0 {
		return sql.NullTime{}, false
	}
	return sql.NullTime{Time: time.Now().Add(time.Duration(durationHours) * time.Hour), Valid: true}, true
}

type adminBanRequest struct {
	Reason        string `json:"reason"`
	DurationHours int    `json:"duration_hours"`
	Permanent     bool   `json:"permanent"`
	ChatMute      bool   `json:"chat_mute"`
	ChatMuteHours int    `json:"chat_mute_hours"`
	BlockDevice   bool   `json:"block_device"`
}

// AdminBanPlayer locks an account (temporary or permanent), optionally mutes
// chat and blocks the last-seen device.
func AdminBanPlayer(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}
	var req adminBanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu khóa tài khoản không hợp lệ"})
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lý do khóa là bắt buộc"})
		return
	}
	if adminID == playerID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể tự khóa tài khoản của mình"})
		return
	}
	until, ok := banUntilFromRequest(req.DurationHours, req.Permanent)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cần chọn thời hạn hoặc khóa vĩnh viễn"})
		return
	}
	if !req.Permanent && req.DurationHours > 24*365*10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thời hạn khóa quá lớn"})
		return
	}

	var ip, fingerprint string
	if err := db.DB.QueryRowContext(c.Request.Context(),
		`SELECT COALESCE(last_login_ip, ''), COALESCE(last_login_fingerprint, '') FROM players WHERE id=$1`, playerID).Scan(&ip, &fingerprint); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải thông tin thiết bị"})
		}
		return
	}

	spec := BanSpec{Reason: req.Reason, Until: until, BannedBy: adminID, BlockDevice: req.BlockDevice, IP: ip, Fingerprint: fingerprint}
	if err := ApplyPlayerBan(c.Request.Context(), playerID, spec); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể khóa tài khoản"})
		return
	}

	if req.ChatMute {
		muteUntil, _ := banUntilFromRequest(req.ChatMuteHours, req.Permanent)
		_ = applyChatMute(c.Request.Context(), playerID, true, muteUntil)
	}
	notifyPlayerBanned(playerID, req.Reason)

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &adminID,
		Action:         "admin.ban_player",
		EntityType:     "player",
		EntityID:       playerID.String(),
		RequestSummary: "ban: " + req.Reason,
		IPAddress:      c.ClientIP(),
		Success:        true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã khóa tài khoản người chơi"})
}

// AdminUnbanPlayer lifts an account ban and clears device blocks created for
// the account's last-seen device.
func AdminUnbanPlayer(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	var ip, fingerprint string
	if err := db.DB.QueryRowContext(c.Request.Context(),
		`SELECT COALESCE(last_login_ip, ''), COALESCE(last_login_fingerprint, '') FROM players WHERE id=$1`, playerID).Scan(&ip, &fingerprint); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải thông tin người chơi"})
		}
		return
	}

	if _, err := db.DB.ExecContext(c.Request.Context(), `
		UPDATE players
		SET is_banned=false, ban_reason='', banned_until=NULL, banned_by=NULL, banned_at=NULL
		WHERE id=$1`, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể mở khóa tài khoản"})
		return
	}
	if ip != "" || fingerprint != "" {
		_, _ = db.DB.ExecContext(c.Request.Context(),
			`DELETE FROM player_access_blocks WHERE ($1 <> '' AND fingerprint = $1) OR ($2 <> '' AND ip = $2)`,
			fingerprint, ip)
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:   &adminID,
		Action:     "admin.unban_player",
		EntityType: "player",
		EntityID:   playerID.String(),
		IPAddress:  c.ClientIP(),
		Success:    true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã mở khóa tài khoản người chơi"})
}

type adminMuteRequest struct {
	DurationHours int  `json:"duration_hours"`
	Permanent     bool `json:"permanent"`
}

// AdminMutePlayer restricts a player from sending chat/mail.
func AdminMutePlayer(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}
	var req adminMuteRequest
	_ = c.ShouldBindJSON(&req)
	until, ok := banUntilFromRequest(req.DurationHours, req.Permanent)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cần chọn thời hạn hoặc hạn chế vĩnh viễn"})
		return
	}
	var exists bool
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM players WHERE id=$1)`, playerID).Scan(&exists); err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}
	if err := applyChatMute(c.Request.Context(), playerID, true, until); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hạn chế chat"})
		return
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:   &adminID,
		Action:     "admin.mute_player",
		EntityType: "player",
		EntityID:   playerID.String(),
		IPAddress:  c.ClientIP(),
		Success:    true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã hạn chế chat người chơi"})
}

// AdminUnmutePlayer lifts a chat restriction.
func AdminUnmutePlayer(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}
	if err := applyChatMute(c.Request.Context(), playerID, false, sql.NullTime{}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể bỏ hạn chế chat"})
		return
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:   &adminID,
		Action:     "admin.unmute_player",
		EntityType: "player",
		EntityID:   playerID.String(),
		IPAddress:  c.ClientIP(),
		Success:    true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã bỏ hạn chế chat người chơi"})
}

// --- Automatic anti-abuse rules ---------------------------------------------

func autoBanEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return fallback
}

func autoBanEnvBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(strings.TrimSpace(v)); err == nil {
			return b
		}
	}
	return fallback
}

func autoBanFingerprintLimit() int { return autoBanEnvInt("AUTO_BAN_FINGERPRINT_LIMIT", 5) }
func autoBanReferralLimit() int    { return autoBanEnvInt("AUTO_BAN_REFERRAL_LIMIT", 5) }
func autoBanDurationHours() int    { return autoBanEnvInt("AUTO_BAN_DURATION_HOURS", 72) }
func autoBanBlockDevice() bool     { return autoBanEnvBool("AUTO_BAN_BLOCK_DEVICE", true) }

// autoBanForFingerprint bans a device-sharing account once it exceeds the
// configured distinct-account limit. Called from the login guards.
func autoBanForFingerprint(ctx context.Context, playerID uuid.UUID, username, fingerprint, ip string, otherAccounts int) {
	limit := autoBanFingerprintLimit()
	if limit <= 0 || otherAccounts < limit {
		return
	}
	until, _ := banUntilFromRequest(autoBanDurationHours(), autoBanDurationHours() <= 0)
	spec := BanSpec{
		Reason:      "Hệ thống: vượt giới hạn đăng nhập nhiều tài khoản trên cùng thiết bị",
		Until:       until,
		BlockDevice: autoBanBlockDevice(),
		IP:          ip,
		Fingerprint: fingerprint,
	}
	if err := ApplyPlayerBan(ctx, playerID, spec); err != nil {
		return
	}
	notifyPlayerBanned(playerID, spec.Reason)
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &playerID,
		Action:         "system.auto_ban_fingerprint",
		EntityType:     "player",
		EntityID:       playerID.String(),
		RequestSummary: "auto-ban fingerprint limit: " + username + " accounts=" + strconv.Itoa(otherAccounts),
		IPAddress:      ip,
		Success:        true,
	})
}

// autoBanForReferral bans an account that farms referrals from one device.
func autoBanForReferral(ctx context.Context, referrerID uuid.UUID, suspiciousCount int) {
	limit := autoBanReferralLimit()
	if limit <= 0 || suspiciousCount < limit {
		return
	}
	until, _ := banUntilFromRequest(autoBanDurationHours(), autoBanDurationHours() <= 0)
	spec := BanSpec{
		Reason: "Hệ thống: lạm dụng giới thiệu (referral) từ cùng một thiết bị",
		Until:  until,
	}
	if err := ApplyPlayerBan(ctx, referrerID, spec); err != nil {
		return
	}
	notifyPlayerBanned(referrerID, spec.Reason)
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &referrerID,
		Action:         "system.auto_ban_referral",
		EntityType:     "player",
		EntityID:       referrerID.String(),
		RequestSummary: "auto-ban referral abuse: suspicious=" + strconv.Itoa(suspiciousCount),
		Success:        true,
	})
}
