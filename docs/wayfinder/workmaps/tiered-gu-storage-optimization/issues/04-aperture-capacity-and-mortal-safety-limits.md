# Ticket 04: Quy chuẩn Giới hạn Sức chứa Không Khiếu Phàm Nhân (Mortal Aperture Capacity Limits)

Type: prototype
Status: resolved
Blocked by:

## Question

Làm thế nào để áp đặt giới hạn Sức Chứa Không Khiếu (Aperture Capacity) cho Cổ Sư Phàm Nhân (Rank 1–5) để vừa bám sát nguyên tác tiểu thuyết (Không Khiếu phàm nhân không thể chứa vô hạn Cổ) vừa chặn đứng triệt để nguy cơ người chơi spam phình to bảng `player_gu` trước khi Thăng Tiên?

Cụ thể cần giải quyết:
1. **Bảng Định Mức Sức Chứa (Capacity Matrix by Rank & Sub-rank)**:
   - Rank 1: Tối đa 10 Cổ trùng.
   - Rank 2: Tối đa 20 Cổ trùng.
   - Rank 3: Tối đa 35 Cổ trùng.
   - Rank 4: Tối đa 55 Cổ trùng.
   - Rank 5: Tối đa 80 Cổ trùng (hoặc 100 Cổ trùng đỉnh phong).
   - Rank 6+ (Cổ Tiên): Tiên Khiếu khai mở, không giới hạn sức chứa cho Đàn Phàm Cổ; riêng Cổ mang trên người (equipped) vẫn giới hạn 60 con.
2. **Các Chốt Chặn Kiểm Tra (Validation Guard Points)**:
   - Nhặt Cổ dã ngoại (`ground_drops.go` / `movement.go`).
   - Mua Cổ từ Thương đoàn / Chợ (`marketplace.go`, `caravan`).
   - Rút Cổ từ Kho Tông Môn (`sect_vault.go`).
   - Kết quả Hợp Luyện tạo Cổ mới (`refine_session_resolve.go`).
   - Nhận thưởng nhiệm vụ / Gacha / Hộp quà.
3. **Trải Nghiệm Khi Đầy Túi (Inventory Full UX)**:
   - Trả lời mã lỗi rõ ràng: `APERTURE_CAPACITY_EXCEEDED` ("Không Khiếu đã quá tải, chân nguyên không đủ dung nạp thêm Cổ trùng, hãy thả bớt Cổ hoặc tu luyện đột phá").
   - Đảm bảo prototype kiểm thử các điểm chặn không để lọt khe hở cho người chơi vượt rào dung lượng.

## Answer

Đã hoàn thiện quy chuẩn định mức sức chứa Không Khiếu và xây dựng prototype kiểm chứng logic tại [`prototypes/04-mortal-aperture-capacity-limits.html`](../prototypes/04-mortal-aperture-capacity-limits.html):

### 1. Bảng Định Mức Sức Chứa Chuẩn Hóa
Định mức dung lượng phản ánh đúng năng lực chân nguyên nuôi dưỡng của Cổ Sư theo từng bậc tu vi:
* **Rank 1 (Nhất Chuyển)**: Tối đa **15** Cổ trùng (rất thoải mái cho tân thủ thường chỉ sở hữu 2–5 con).
* **Rank 2 (Nhị Chuyển)**: Tối đa **25** Cổ trùng.
* **Rank 3 (Tam Chuyển)**: Tối đa **40** Cổ trùng.
* **Rank 4 (Tứ Chuyển)**: Tối đa **60** Cổ trùng.
* **Rank 5 (Ngũ Chuyển)**: Tối đa **80** Cổ trùng (Tiểu cảnh giới 4 Đỉnh Phong đạt **100** Cổ trùng).
* **Rank 6+ (Cổ Tiên)**: Tiên Khiếu khai mở, dung lượng Đàn Phàm Cổ là **Vô Hạn** (`capacity = -1`); riêng Cổ mang trên người (`equipped`) vẫn chịu trần 60 con theo `MaxEquippedGus`.

### 2. Thuật Toán Kiểm Tra Sức Chứa (Go Logic Specification)
Hàm kiểm tra tính toán theo số lượng tăng ròng (`delta`):
```go
// MortalApertureCapacity trả về số lượng Cổ tối đa của Không Khiếu phàm trần. Cổ Tiên trả về -1 (vô hạn).
func MortalApertureCapacity(rank int, subRank int) int {
	switch rank {
	case 1:
		return 15
	case 2:
		return 25
	case 3:
		return 40
	case 4:
		return 60
	case 5:
		if subRank >= 4 {
			return 100
		}
		return 80
	default:
		if rank >= 6 {
			return -1
		}
		return 15
	}
}

// CheckApertureCapacity kiểm tra xem việc thêm delta Cổ có vượt ngưỡng Không Khiếu không.
func CheckApertureCapacity(tx Queryer, playerID string, rank, subRank, delta int) error {
	cap := MortalApertureCapacity(rank, subRank)
	if cap < 0 || delta <= 0 {
		return nil // Cổ Tiên hoặc thao tác không làm tăng số Cổ (như Hợp Luyện tiêu hao)
	}
	var currentCount int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM player_gu WHERE player_id = $1 AND location <> 'ACTOR'`, playerID).Scan(&currentCount); err != nil {
		return err
	}
	if currentCount+delta > cap {
		return fmt.Errorf("không khiếu đã đạt giới hạn sức chứa (%d/%d cổ trùng), chân nguyên không đủ dung nạp thêm", currentCount, cap)
	}
	return nil
}
```

### 3. Xử Lý Tình Huống Góc: Hợp Luyện Khi Đầy Túi (Net-Negative Logic)
* **Vấn đề phát hiện trong prototype**: Nếu người chơi đang ở ngưỡng tối đa (ví dụ 15/15 ở Rank 1), thao tác Hợp Luyện tiêu hao 2 Cổ cũ để tạo 1 Cổ mới -> Số lượng Cổ ròng sau khi hợp luyện là `15 - 2 + 1 = 14 <= 15`.
* **Giải pháp**: Kiểm tra dung lượng theo `delta = producedCount - consumedCount`. Vì `delta = -1 <= 0`, hệ thống **cho phép hợp luyện bình thường**, không làm nghẽn tiến trình của người chơi.

### 4. Vị Trí Chốt Chặn Kiểm Tra Trong Codebase
1. `backend/internal/db/player_gu.go` (`AddPlayerGu` / `AddPlayerGusBulk`): Chốt chặn tầng dưới cùng.
2. `backend/internal/handlers/ground_drops.go`: Chặn nhặt Cổ rơi khi đầy túi.
3. `backend/internal/handlers/marketplace.go`: Chặn mua Cổ từ chợ người chơi.
4. `backend/internal/handlers/sect_vault.go`: Chặn rút Cổ từ kho tông môn.
5. `backend/internal/handlers/refine_session_resolve.go`: Cho phép khi `net_delta <= 0`.
6. Client nhận mã lỗi `APERTURE_CAPACITY_EXCEEDED` hiển thị toast cảnh báo: *"Không Khiếu đã quá tải, chân nguyên không đủ dung nạp thêm Cổ trùng, hãy thả bớt Cổ hoặc tu luyện đột phá!"*.

