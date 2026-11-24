# ĐẶC TẢ KIẾN TRÚC BACKEND & DATABASE CHO LƯU VẾT CHẤP THUẬN VÀ QUYỀN XÓA DỮ LIỆU

**Mục tiêu**: Chuẩn hóa cấu trúc cơ sở dữ liệu PostgreSQL, luồng xử lý API Backend (Golang / Gin) để ghi nhận chứng cứ chấp thuận (Consent Audit Trail) theo **Nghị định 13/2023/NĐ-CP** và hỗ trợ quyền xóa/vô danh hóa dữ liệu (Right to be Forgotten) an toàn toàn vẹn dữ liệu trong GuWorld.

---

## 1. THIẾT KẾ CƠ SỞ DỮ LIỆU (DATABASE SCHEMA EXTENSIONS)

Căn cứ quy ước tại `AGENTS.md` và `backend/internal/db/schema_players.go`:

### 1.1. Mở Rộng Bảng `players`
Thêm các cột quản lý Chấp thuận Điều khoản & Trạng thái Xóa dữ liệu:

```sql
-- Thêm vào định nghĩa CREATE TABLE IF NOT EXISTS players:
accepted_terms_version VARCHAR(20) NOT NULL DEFAULT '',
accepted_privacy_version VARCHAR(20) NOT NULL DEFAULT '',
terms_accepted_at TIMESTAMPTZ DEFAULT NULL,
terms_accepted_ip VARCHAR(50) NOT NULL DEFAULT '',

is_deleted BOOLEAN NOT NULL DEFAULT false,
deleted_at TIMESTAMPTZ DEFAULT NULL,
deletion_reason TEXT NOT NULL DEFAULT ''
```

### 1.2. Khối DDL Tương Thích Ngược (Idempotent Compatibility Schema)
Đặt trong hàm `schemaPlayers()` tại `backend/internal/db/schema_players.go`:

```go
if err := execSchema("players_legal_consent_columns", `
ALTER TABLE players ADD COLUMN IF NOT EXISTS accepted_terms_version VARCHAR(20) NOT NULL DEFAULT '';
ALTER TABLE players ADD COLUMN IF NOT EXISTS accepted_privacy_version VARCHAR(20) NOT NULL DEFAULT '';
ALTER TABLE players ADD COLUMN IF NOT EXISTS terms_accepted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE players ADD COLUMN IF NOT EXISTS terms_accepted_ip VARCHAR(50) NOT NULL DEFAULT '';
ALTER TABLE players ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE players ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE players ADD COLUMN IF NOT EXISTS deletion_reason TEXT NOT NULL DEFAULT '';
`); err != nil {
    return err
}

if err := execSchema("players_is_deleted_idx", `
CREATE INDEX IF NOT EXISTS idx_players_is_deleted ON players (is_deleted) WHERE is_deleted = true;
`); err != nil {
    return err
}
```

---

## 2. CHUẨN HÓA LUỒNG ĐĂNG KÝ (CONSENT AUDIT ON REGISTRATION)

### 2.1. Cập Nhật Model Yêu Cầu Đăng Ký (`CreatePlayerRequest`)
Trong `backend/internal/handlers/player_auth_register.go`:

```go
const (
    CurrentTermsVersion   = "2024.1"
    CurrentPrivacyVersion = "2024.1"
)

type CreatePlayerRequest struct {
    Username       string `json:"username" binding:"required"`
    Password       string `json:"password" binding:"required"`
    Email          string `json:"email" binding:"required"`
    StartingRegion string `json:"starting_region"`
    AttackGuID     string `json:"attack_gu_id"`
    DefenseGuID    string `json:"defense_gu_id"`
    Gender         string `json:"gender"`
    Fingerprint    string `json:"fingerprint"`
    AgreeTerms     bool   `json:"agree_terms"` // Bắt buộc true
}
```

### 2.2. Kiểm Tra Chấp Thuận & Lưu Vết (Enforcement & Audit Insertion)
Trước khi khởi tạo giao dịch DB trong `RegisterPlayer`:

```go
// Xác thực bắt buộc chấp thuận Điều khoản & Chính sách theo NĐ 13/2023
if !req.AgreeTerms {
    c.JSON(http.StatusBadRequest, gin.H{
        "error": "Đạo hữu cần đồng ý với Điều Khoản Dịch Vụ và Chính Sách Bảo Mật để thức tỉnh Không Khiếu!",
    })
    return
}
```

Khi thực hiện lệnh `INSERT INTO players`:
- Bổ sung các cột: `accepted_terms_version`, `accepted_privacy_version`, `terms_accepted_at`, `terms_accepted_ip`.
- Giá trị truyền vào:
  - `accepted_terms_version` = `CurrentTermsVersion` (`"2024.1"`)
  - `accepted_privacy_version` = `CurrentPrivacyVersion` (`"2024.1"`)
  - `terms_accepted_at` = `time.Now().UTC()`
  - `terms_accepted_ip` = `c.ClientIP()`

---

## 3. THIẾT KẾ API QUYỀN XÓA TÀI KHOẢN (RIGHT TO BE FORGOTTEN)

Căn cứ **Điều 9 và Điều 14 Nghị định số 13/2023/NĐ-CP**: Người chơi có quyền yêu cầu xóa dữ liệu cá nhân.

### 3.1. Định Tuyến Endpoint
- **Phương thức**: `POST`
- **Đường dẫn**: `/api/player/request-deletion`
- **Xác thực**: Yêu cầu JWT Token (Bearer Auth qua middleware `AuthMiddleware`)

### 3.2. Payload Yêu Cầu
```json
{
  "password": "user_current_password",
  "reason": "Tôi không còn nhu cầu chơi tiếp"
}
```

### 3.3. Quy Trình Xử Lý Kỹ Thuật (Transaction & Anonymization)
Hệ thống áp dụng chiến lược **Vô danh hóa dữ liệu (Data Anonymization & Soft Delete)** nhằm vừa xóa sạch thông tin cá nhân của người chơi, vừa bảo đảm tính toàn vẹn khóa ngoại (FK) và đáp ứng luật kế toán lưu trữ giao dịch PayOS:

```go
func RequestAccountDeletion(c *gin.Context) {
    playerID := auth.GetPlayerID(c)
    var req struct {
        Password string `json:"password" binding:"required"`
        Reason   string `json:"reason"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Vui lòng nhập mật khẩu xác nhận!"})
        return
    }

    tx, err := db.GetDB().BeginTx(c.Request.Context(), nil)
    if err != nil {
        httperr.Respond(c, httperr.Internal("Không thể mở giao dịch", err))
        return
    }
    defer tx.Rollback()

    // 1. Kiểm tra mật khẩu hiện tại
    var currentHash, email string
    var isDeleted bool
    err = tx.QueryRowContext(c.Request.Context(), `
        SELECT password_hash, email, is_deleted FROM players WHERE id = $1 FOR UPDATE
    `, playerID).Scan(&currentHash, &email, &isDeleted)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tài khoản"})
        return
    }
    if isDeleted {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Tài khoản này đã bị xóa trước đó!"})
        return
    }

    if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.Password)); err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Mật khẩu xác nhận không chính xác!"})
        return
    }

    // 2. Thực hiện vô danh hóa (Anonymization) thông tin định danh cá nhân
    anonymizedEmail := fmt.Sprintf("deleted_%s@guworld.local", playerID.String()[:8])
    anonymizedIP := "0.0.0.0"

    _, err = tx.ExecContext(c.Request.Context(), `
        UPDATE players
        SET is_deleted = true,
            deleted_at = NOW(),
            deletion_reason = $1,
            email = $2,
            password_hash = '',
            register_ip = $3,
            last_login_ip = $3,
            register_fingerprint = '',
            last_login_fingerprint = '',
            active_session_id = ''
        WHERE id = $4
    `, req.Reason, anonymizedEmail, anonymizedIP, playerID)
    if err != nil {
        httperr.Respond(c, httperr.Internal("Lỗi cập nhật xóa tài khoản", err))
        return
    }

    // 3. Ghi vết Audit Log
    auditlog.Log(tx, auditlog.Entry{
        ActorID:   playerID,
        Action:    "player.account_deleted_gdpr",
        TargetID:  playerID.String(),
        Details:   map[string]any{"reason": req.Reason, "old_email_masked": maskEmail(email)},
        IPAddress: c.ClientIP(),
    })

    if err := tx.Commit(); err != nil {
        httperr.Respond(c, httperr.Internal("Lỗi xác nhận giao dịch xóa", err))
        return
    }

    // 4. Ngắt kết nối SSE / Realtime Session
    uiRealtimeManager.DisconnectPlayer(playerID.String())

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "message": "Tài khoản và toàn bộ dữ liệu định danh cá nhân đã được vô danh hóa thành công theo Nghị định 13/2023/NĐ-CP.",
    })
}
```

---

## 4. QUẢN LÝ PHIÊN BẢN ĐIỀU KHOẢN (TERMS VERSIONING RE-CONSENT)

Khi Ban Quản Trị nâng cấp phiên bản Điều khoản (ví dụ từ `"2024.1"` lên `"2025.1"`):
1. **Kiểm tra lúc Đăng Nhập (`LoginPlayer`)**:
   - So sánh `player.AcceptedTermsVersion` với `CurrentTermsVersion`.
   - Nếu `player.AcceptedTermsVersion < CurrentTermsVersion`: Response đăng nhập trả về thêm cờ:
     ```json
     {
       "token": "jwt_token...",
       "requires_terms_reconsent": true,
       "new_terms_version": "2025.1"
     }
     ```
2. **Endpoint Xác Nhận Điều Khoản Mới**:
   - `POST /api/player/reconsent-terms`
   - Nhận payload: `{"agree_terms": true, "version": "2025.1"}`.
   - Cập nhật lại `accepted_terms_version = "2025.1"`, `terms_accepted_at = NOW()`, `terms_accepted_ip = ClientIP()`.
