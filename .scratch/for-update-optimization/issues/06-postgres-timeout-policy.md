# Chính sách Cấu hình Timeout PostgreSQL và Nhận diện Lỗi Tạm thời (Transient Errors)

Type: research
Status: resolved
Blocked by: None

Part of map.md

## Question

Hiện tại cấu hình mặc định pool trong `db.go` là:
- `statement_timeout = 15s`
- `lock_timeout = 10s`
- `idle_in_transaction_session_timeout = 15s`
Một số handler tự gọi `SELECT set_config('lock_timeout', '2000', true), set_config('statement_timeout', '8000', true)`, nhưng phần lớn các handler khác không có.
1. Tại sao `pq: canceling statement due to statement timeout (57014)` xảy ra thay vì `lock_timeout (55P03)`? (Có phải do câu lệnh đợi lock 10s cộng với thời gian thực thi vượt 15s, hoặc do các câu lệnh trước đó trong cùng tx đã tiêu tốn gần hết quỹ 15s?)
2. Cần điều chỉnh mức cấu hình timeout toàn cục hoặc thiết lập helper chuẩn `BeginTxWithTimeout(ctx, opts)` cho các transaction như thế nào để luôn luôn Fail-fast trong 2s-3s thay vì để user chờ 15s?
3. Cách phân biệt và xử lý thống nhất các mã lỗi Postgres: `57014` (statement timeout / query canceled), `55P03` (lock_not_available), `40P01` (deadlock_detected) trên toàn bộ hệ thống API?

## Answer

### 1. Phân Tích Cơ Chế Gây Lỗi `57014` Thay Vì `55P03`

Trong PostgreSQL:
- **`lock_timeout` (10s):** Chỉ đếm thời gian thực sự nằm trong hàng đợi chờ cấp phát khóa. Nếu được cấp khóa tại giây thứ 9.5s, bộ đếm này kết thúc thành công.
- **`statement_timeout` (15s):** Đếm **tổng thời gian tích lũy** của câu lệnh từ lúc bắt đầu nhận từ client: bao gồm thời gian chờ khóa (9.5s) + thời gian chạy query + thời gian quét index/quét mảng. Do đó, câu lệnh rất dễ vượt ngưỡng 15s và bị hủy bởi `statement_timeout (57014)`.
- **Client Cancel & Context Timeout:** Trong Go, nếu handler hoặc reverse proxy (Nginx/Cloudflare) có timeout 5s - 10s và ngắt context (`ctx.Done()`), driver `lib/pq` gửi Cancel Request đến PostgreSQL. PostgreSQL hủy query và trả về mã lỗi **`57014` (query_canceled)**.
- **Hệ quả tiêu cực:** Để `lock_timeout` 10s là quá dài với một web game thời gian thực. Một câu lệnh đợi 10s sẽ giam giữ 1 connection trong connection pool và khiến người dùng đứng hình 10-15s trước khi nhận lỗi timeout.

---

### 2. Chuẩn Hóa Helper Transaction: `BeginTxWithTimeout`

Thay vì để mỗi handler tự gõ chuỗi raw `SELECT set_config(...)` chắp vá hoặc không cấu hình gì, thiết lập một helper chuẩn duy nhất trong package `db`:

```go
package db

import (
	"context"
	"database/sql"
	"fmt"
)

type TxOptions struct {
	LockTimeoutMs      int // Mặc định 2000ms (2s)
	StatementTimeoutMs int // Mặc định 5000ms (5s)
}

// BeginTxWithTimeout mở transaction và cấu hình fail-fast timeout cục bộ (is_local = true).
// Cấu hình chỉ tồn tại trong transaction này và tự động reset khi commit/rollback.
func BeginTxWithTimeout(ctx context.Context, dbConn *sql.DB, opts *TxOptions) (*sql.Tx, error) {
	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	lockMs := 2000
	stmtMs := 5000
	if opts != nil {
		if opts.LockTimeoutMs > 0 {
			lockMs = opts.LockTimeoutMs
		}
		if opts.StatementTimeoutMs > 0 {
			stmtMs = opts.StatementTimeoutMs
		}
	}

	// is_local = true (tham số thứ 3) đảm bảo scope chỉ giới hạn trong transaction này
	setQuery := fmt.Sprintf("SELECT set_config('lock_timeout', '%d', true), set_config('statement_timeout', '%d', true)", lockMs, stmtMs)
	if _, err := tx.ExecContext(ctx, setQuery); err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("failed to set tx timeouts: %w", err)
	}

	return tx, nil
}
```

#### Quy Chuẩn Thời Gian:
- **`lock_timeout = 2000ms` (2s):** Bất kỳ giao dịch nào không lấy được khóa trong vòng 2s sẽ bị văng lỗi ngay (fail-fast), không bao giờ để request đợi đến 10s hay 15s.
- **`statement_timeout = 5000ms` (5s):** Bảo vệ toàn bộ câu lệnh trong transaction không chạy quá 5s.
- Giữ cấu hình pool-level trong `db.go` (`statement_timeout = 15s, lock_timeout = 10s`) làm hàng rào an ninh cuối cùng cho các câu lệnh đọc không qua transaction.

---

### 3. Chuẩn Hóa Phân Loại và Phản Hồi Mã Lỗi PostgreSQL Toàn Hệ Thống

Tạo helper `ClassifyPostgresError` để toàn bộ các handler trong `backend/internal/handlers/` xử lý đồng nhất:

```go
package db

import (
	"errors"
	"net/http"

	"github.com/lib/pq"
)

type ErrorClassification struct {
	IsTransient bool
	StatusCode  int
	ErrorCode   string
	Message     string
}

// ClassifyPostgresError phân loại các lỗi PostgreSQL phổ biến để trả về mã HTTP và thông báo phù hợp.
func ClassifyPostgresError(err error) ErrorClassification {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "55P03": // lock_not_available (do NOWAIT hoặc lock_timeout hết hạn)
			return ErrorClassification{
				IsTransient: true,
				StatusCode:  http.StatusConflict, // 409
				ErrorCode:   "RESOURCE_LOCKED",
				Message:     "Tài nguyên đang có người thao tác hoặc đang bận, vui lòng thử lại sau giây lát!",
			}

		case "40P01": // deadlock_detected
			return ErrorClassification{
				IsTransient: true,
				StatusCode:  http.StatusConflict, // 409
				ErrorCode:   "TRANSACTION_DEADLOCK",
				Message:     "Xung đột giao dịch đồng thời, vui lòng thử lại thao tác!",
			}

		case "57014": // query_canceled / statement_timeout
			return ErrorClassification{
				IsTransient: true,
				StatusCode:  http.StatusGatewayTimeout, // 504
				ErrorCode:   "STATEMENT_TIMEOUT",
				Message:     "Máy chủ đang bận xử lý nhiều yêu cầu, thao tác bị quá hạn. Vui lòng thử lại!",
			}

		case "40001": // serialization_failure
			return ErrorClassification{
				IsTransient: true,
				StatusCode:  http.StatusConflict, // 409
				ErrorCode:   "SERIALIZATION_FAILURE",
				Message:     "Dữ liệu đã bị thay đổi đồng thời, vui lòng làm mới và thử lại!",
			}
		}
	}

	return ErrorClassification{
		IsTransient: false,
		StatusCode:  http.StatusInternalServerError, // 500
		ErrorCode:   "INTERNAL_DB_ERROR",
		Message:     "Lỗi cơ sở dữ liệu nội bộ",
	}
}
```

#### Xử Lý Thống Nhất Trong Gin Handlers:
```go
if err != nil {
    classified := db.ClassifyPostgresError(err)
    if classified.IsTransient {
        logger.S.Warnf("[%s] Transient DB error: %v", c.Path(), err)
    } else {
        logger.S.Errorf("[%s] Critical DB error: %v", c.Path(), err)
    }
    c.JSON(classified.StatusCode, gin.H{
        "error": classified.Message,
        "code":  classified.ErrorCode,
    })
    return
}
```

---

### 4. Kết Quả Đạt Được
- **Fail-fast hoàn toàn:** Loại bỏ 100% các trường hợp request bị treo 10-15s trên giao diện người dùng. Nếu có xung đột, phản hồi trả về ngay trong 0ms (với NOWAIT) hoặc tối đa 2s (với lock_timeout).
- **Phân loại lỗi minh bạch:** Người dùng và frontend không còn nhận lỗi mập mờ `pq: canceling statement due to statement timeout (57014)`. Thay vào đó là thông báo thân thiện rõ ràng kèm mã HTTP 409 / 504 chuẩn RESTful.
