package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/lib/pq"
)

// TxTimeoutOptions điều chỉnh timeout cục bộ (SET LOCAL) bên trong một transaction.
type TxTimeoutOptions struct {
	LockTimeoutMs      int // Mặc định 2000ms — fail-fast khi đợi lock
	StatementTimeoutMs int // Mặc định 5000ms — chặn câu lệnh chạy quá lâu
}

// DefaultTxTimeout là cấu hình mặc định cho các transaction gameplay thông thường.
var DefaultTxTimeout = &TxTimeoutOptions{
	LockTimeoutMs:      2000,
	StatementTimeoutMs: 5000,
}

// LongTxTimeout dùng cho các luồng nặng có chủ đích (chiến tranh tông môn, nhập data).
var LongTxTimeout = &TxTimeoutOptions{
	LockTimeoutMs:      3000,
	StatementTimeoutMs: 30000,
}

// BeginTxWithTimeout mở transaction và áp đặt fail-fast timeouts cục bộ.
// Khi xác nhận (commit) hoặc rollback transaction, cấu hình cục bộ tự hoàn nguyên.
func BeginTxWithTimeout(ctx context.Context, dbConn *sql.DB, opts *TxTimeoutOptions) (*sql.Tx, error) {
	if opts == nil {
		opts = DefaultTxTimeout
	}
	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	lockMs := opts.LockTimeoutMs
	if lockMs <= 0 {
		lockMs = 2000
	}
	stmtMs := opts.StatementTimeoutMs
	if stmtMs <= 0 {
		stmtMs = 5000
	}
	setQ := fmt.Sprintf(
		"SELECT set_config('lock_timeout', '%d', true), set_config('statement_timeout', '%d', true)",
		lockMs, stmtMs,
	)
	if _, err := tx.ExecContext(ctx, setQ); err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("BeginTxWithTimeout: set timeouts: %w", err)
	}
	return tx, nil
}

// ErrorClass mô tả phân loại của một lỗi PostgreSQL.
type ErrorClass struct {
	IsTransient bool   // Có thể retry hoặc thông báo thân thiện với user
	StatusCode  int    // HTTP status code phù hợp
	ErrorCode   string // Mã nội bộ gửi về frontend
	Message     string // Thông báo tiếng Việt thân thiện
}

// ClassifyPostgresError phân loại pq errors thành HTTP status và thông báo phù hợp.
// Trả về classification với IsTransient=false và StatusCode=500 khi không nhận diện được.
func ClassifyPostgresError(err error) ErrorClass {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "55P03": // lock_not_available — do NOWAIT hoặc lock_timeout
			return ErrorClass{
				IsTransient: true,
				StatusCode:  http.StatusConflict,
				ErrorCode:   "RESOURCE_LOCKED",
				Message:     "Tài nguyên đang có người thao tác, vui lòng thử lại sau giây lát!",
			}
		case "40P01": // deadlock_detected
			return ErrorClass{
				IsTransient: true,
				StatusCode:  http.StatusConflict,
				ErrorCode:   "TRANSACTION_DEADLOCK",
				Message:     "Xung đột giao dịch đồng thời, vui lòng thử lại thao tác!",
			}
		case "57014": // query_canceled / statement_timeout
			return ErrorClass{
				IsTransient: true,
				StatusCode:  http.StatusGatewayTimeout,
				ErrorCode:   "STATEMENT_TIMEOUT",
				Message:     "Máy chủ đang bận xử lý nhiều yêu cầu, thao tác bị quá hạn. Vui lòng thử lại!",
			}
		case "40001": // serialization_failure
			return ErrorClass{
				IsTransient: true,
				StatusCode:  http.StatusConflict,
				ErrorCode:   "SERIALIZATION_FAILURE",
				Message:     "Dữ liệu đã bị thay đổi đồng thời, vui lòng thử lại!",
			}
		case "23505": // unique_violation — bổ sung để handler có thể dùng
			return ErrorClass{
				IsTransient: false,
				StatusCode:  http.StatusConflict,
				ErrorCode:   "DUPLICATE_KEY",
				Message:     "Dữ liệu đã tồn tại, không thể thêm trùng lặp!",
			}
		}
	}
	return ErrorClass{
		IsTransient: false,
		StatusCode:  http.StatusInternalServerError,
		ErrorCode:   "INTERNAL_DB_ERROR",
		Message:     "Lỗi cơ sở dữ liệu nội bộ",
	}
}

// IsLockNotAvailable báo xem lỗi có phải do NOWAIT kích hoạt (55P03) hay không.
// Dùng trong các handlers đã chuyển sang FOR UPDATE NOWAIT để bắt và xử lý nhanh.
func IsLockNotAvailable(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "55P03"
}

// IsStatementTimeout báo xem lỗi có phải do statement_timeout hay query cancel (57014).
func IsStatementTimeout(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "57014"
}

// IsDeadlock báo xem lỗi có phải do deadlock (40P01).
func IsDeadlock(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "40P01"
}
