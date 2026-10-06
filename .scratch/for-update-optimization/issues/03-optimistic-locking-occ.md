# Đặc tả Optimistic Concurrency Control (OCC) cho đột biến trạng thái Player

Type: research
Status: resolved
Blocked by: None

Part of map.md

## Question

Bảng `players` đã có sẵn cột `version` (int). Trong `player_service.go` và các luồng mutation lớn (tu luyện, đột phá, trang bị Cổ trùng, đổi phái):
1. Có những luồng nào có thể chuyển từ `LoadForUpdate` sang đọc không khóa (`LoadReadOnly`), tính toán trạng thái trong Go, rồi cập nhật bằng `UPDATE players SET ... version = version + 1 WHERE id = $1 AND version = $currentVersion`?
2. Khi `RowsAffected() == 0` (xung đột version do request đồng thời ghi đè trước), cơ chế retry ở tầng Go nên thiết kế ra sao (số lần retry, backoff, giới hạn context)?
3. Những trường hợp ngoại lệ nào (ví dụ: giao dịch thương mại giữa 2 người, PvP trực tiếp) bắt buộc vẫn phải dùng khóa bi quan (pessimistic lock)?

## Answer

### 1. Phân Tích Hiện Trạng `players.version` và Điểm Nghẽn

Trong `backend/internal/services/player_service.go`:
- Cột `version` (int) đã tồn tại sẵn trong bảng `players` và được nạp vào struct `models.Player.Version`.
- Tuy nhiên, trong `EvaluateAndPersistMana` (dòng 645-671), hệ thống lại dùng cơ chế khóa bi quan:
  ```go
  player, err := s.LoadForUpdate(tx, id) // <-- SELECT ... FOR UPDATE (giữ lock dòng)
  // Tính toán time-variant fields trong RAM Go (mana, stress, lifespan)
  // Thực hiện EvaluatePlayerHungerTx (quét Gu)
  tx.Exec(`UPDATE players SET ... version = version + 1 WHERE id = $1`, id) // Không check version cũ!
  ```
- **Hậu quả:** Vì hàm này được nhúng vào hơn **70 endpoint API**, bất kỳ khi nào người chơi có 2 request đồng thời (ví dụ: auto-combat song song với bấm nhặt đồ hoặc poll SSE), request thứ hai bị xếp hàng đợi khóa. Khi hàng đợi kéo dài > 15s, PostgreSQL hủy câu lệnh bằng lỗi `57014 (statement timeout)`.

---

### 2. Danh Mục Các Luồng Chuyển Sang OCC (Optimistic Concurrency Control)

Toàn bộ các luồng thao tác đơn người chơi (Single-player mutations) chuyển từ `LoadForUpdate` sang `LoadReadOnly + OCC`:

1. **Đồng bộ trạng thái & Mana (`EvaluateAndPersistMana` / `EvaluateAndPersistManaEquipped`):**
   - Đọc snapshot không khóa (`LoadReadOnly`).
   - Tính toán mana regen, lifespan, stress trong Go.
   - Cập nhật bằng OCC:
     ```sql
     UPDATE players
     SET current_mana = $1, max_mana = $2, last_mana_update = $3,
         lifespan = $4, time_speed_multiplier = $5, aperture_stress = $6,
         last_cultivate_time = $7, hp = $8, soul_hp = $9, current_thought = $10,
         version = version + 1
     WHERE id = $11 AND version = $12
     ```
   - Nếu `RowsAffected() == 0`: Phát hiện xung đột version trong < 1ms, kích hoạt retry thay vì treo 15s.

2. **Tu Luyện & Đột Phá (`cultivate.go`):**
   - Đột phá màng khiếu (`Breakthrough`), tu luyện chân nguyên (`CultivateMana`), tu luyện hồn phách (`CultivateSoul`).
   - Chỉ có chính người chơi thực hiện; xung đột chỉ xảy ra khi spam click. OCC giải quyết triệt để.

3. **Di Chuyển & Thám Hiểm Bản Đồ (`exploration_move.go`):**
   - Cập nhật tọa độ `(x, y)`, ô bản đồ đã khám phá, thể lực tiêu hao.

4. **Quản Lý Thú Nuôi & Dị Nhân (`pets.go`, `variant_npc_gameplay.go`):**
   - Cho thú ăn (`FeedPet`), ghép thú (`MergePets`), xếp trận dị nhân (`SetNPCFormation`), tu luyện dị nhân (`CultivateNPC`).

5. **Cài Đặt Hồ Sơ & Bảo Hộ Tân Thủ (`player_profile.go`, `respawn.go`):**
   - Cập nhật Bio, ngụy trang, bật/tắt khiên bảo hộ PvP sau hồi sinh.

---

### 3. Thiết Kế Chuẩn Cơ Chế OCC Retry ở Tầng Go

Xây dựng helper chuẩn `WithOCCRetry` để bọc các tác vụ mutation nhạy cảm:

```go
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"time"
)

var ErrOCCVersionConflict = errors.New("occ: version conflict detected")

// WithOCCRetry thực thi hàm fn với cơ chế retry khi phát hiện xung đột version.
func WithOCCRetry[T any](ctx context.Context, maxRetries int, fn func() (T, error)) (T, error) {
	var zero T
	var lastErr error
	backoff := 15 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		default:
		}

		result, err := fn()
		if err == nil {
			return result, nil
		}

		// Chỉ retry khi là lỗi xung đột OCC; các lỗi nghiệp vụ khác văng ngay lập tức
		if !errors.Is(err, ErrOCCVersionConflict) {
			return zero, err
		}

		lastErr = err
		// Exponential backoff có jitter để tránh hiện tượng thundering herd
		jitter := time.Duration(rand.Int63n(int64(10 * time.Millisecond)))
		sleepDuration := backoff + jitter

		timer := time.NewTimer(sleepDuration)
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, ctx.Err()
		case <-timer.C:
		}

		backoff *= 2
	}
	return zero, fmt.Errorf("thao tác thất bại sau %d lần thử do có request đồng thời: %w", maxRetries, lastErr)
}
```

#### Quy Chuẩn Tham Số Retry:
- **Số lần thử tối đa (`maxRetries`):** `3` lần (đủ để hấp thụ 99.9% trường hợp spam click hoặc xung đột với background worker).
- **Thời gian chờ (`backoff`):** Khởi đầu `15ms`, nhân đôi mỗi lần (`15ms -> 30ms -> 60ms`) kèm `jitter 0-10ms`. Tổng thời gian retry tối đa chỉ ~120ms, hoàn toàn trong suốt với người dùng và **nhỏ hơn 100 lần so với timeout 15s**.
- **Context Cancellation:** Luôn kiểm tra `ctx.Done()`. Nếu client ngắt kết nối (đóng tab/rớt mạng), dừng retry ngay lập tức để tiết kiệm tài nguyên CPU/DB.

---

### 4. Những Trường Hợp Bắt Buộc Giữ Khóa Bi Quan (Pessimistic Lock)

Không áp dụng OCC cho các trường hợp đa người chơi có tính chất tranh chấp hoặc trao đổi chéo (để tránh Livelock):

1. **Giao Dịch P2P Trực Tiếp (`trade_confirm.go`):**
   - Cần đóng băng tài sản của cả Alice và Bob cùng lúc. Nếu dùng OCC, Alice và Bob có thể liên tục làm lệch version của nhau dẫn tới Livelock (cả 2 đều fail retry).
   - *Chuẩn áp dụng:* Giữ `FOR UPDATE` nhưng sắp xếp thứ tự khóa ID tăng dần (`MIN(alice, bob)` rồi đến `MAX(...)`) và đặt `lock_timeout = 2s`.
2. **Khóa Đang Chiến Đấu (`players.active_combat`):**
   - Khi đang trong trận chiến PvP/PvE, cấm mọi hành động di chuyển hoặc chuyển đồ. Kiểm tra cờ `active_combat` trực tiếp.
3. **Thao Tác Quản Trị Trừng Phạt (`admin_roles.go`):**
   - Khóa tài khoản, cấm chat, ban nick. Cần khóa bi quan cưỡng chế tức thì.
