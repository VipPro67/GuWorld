package services

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/lib/pq"
)

// ErrOCCVersionConflict là sentinel error khi UPDATE ... WHERE version = $old không cập nhật dòng nào,
// báo hiệu cần retry do có request khác đã ghi đè trước.
var ErrOCCVersionConflict = errors.New("occ: version conflict — row updated by concurrent request")

// isOCCConflict kiểm tra xem lỗi có phải là xung đột OCC hay không.
// Bao gồm cả ErrOCCVersionConflict lẫn các Postgres transient errors có thể retry.
func isOCCConflict(err error) bool {
	if errors.Is(err, ErrOCCVersionConflict) {
		return true
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "40001", // serialization_failure
			"40P01": // deadlock_detected — safe to retry after jittered sleep
			return true
		}
	}
	return false
}

// WithOCCRetry thực thi fn tối đa maxRetries lần, retry tự động khi phát hiện
// xung đột version OCC hoặc lỗi transient serialization/deadlock.
// Sử dụng exponential backoff với jitter để tránh thundering-herd.
//
// fn nhận attempt (bắt đầu từ 0) và trả về (result, error).
// Nếu error là ErrOCCVersionConflict, isOCCConflict trả về true và retry.
// Các lỗi nghiệp vụ khác văng ngay lập tức không retry.
func WithOCCRetry[T any](ctx context.Context, maxRetries int, fn func(attempt int) (T, error)) (T, error) {
	var zero T
	if maxRetries <= 0 {
		maxRetries = 3
	}

	base := 15 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return zero, err
		}

		result, err := fn(attempt)
		if err == nil {
			return result, nil
		}
		if !isOCCConflict(err) {
			// Lỗi nghiệp vụ hoặc DB thực sự — không retry
			return zero, err
		}

		if attempt == maxRetries-1 {
			// Hết lần thử — trả về lỗi
			return zero, fmt.Errorf("thao tác thất bại sau %d lần thử do xung đột đồng thời: %w", maxRetries, err)
		}

		// Exponential backoff với jitter để tránh thundering herd
		sleep := base*(1<<uint(attempt)) + time.Duration(rand.Int63n(int64(10*time.Millisecond)))
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(sleep):
		}
	}
	return zero, fmt.Errorf("occ retry: unreachable")
}
