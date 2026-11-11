# Ticket 01: Thiết kế cấu trúc dữ liệu và DDL cho Đàn Phàm Cổ Tiên Khiếu (Aperture Gu Swarms)

Type: research
Status: resolved
Blocked by:

## Question

Làm thế nào để thiết kế cấu trúc lưu trữ cho Đàn Phàm Cổ (Mortal Gu Swarms) trong Tiên Khiếu đảm bảo tối ưu hóa triệt để I/O, không phình DB, và bảo vệ ràng buộc nghiệp vụ (chỉ Phàm Cổ Rank 1–5 mới được stack)?

Cụ thể cần nghiên cứu và giải quyết:
1. **Lựa chọn Schema**:
   - Phương án A: Tạo bảng mới `aperture_gu_swarms (aperture_id UUID, gu_id VARCHAR(50), rank INT, quantity BIGINT, created_at, updated_at, PRIMARY KEY (aperture_id, gu_id, rank))` gắn liền với `immortal_apertures(id)`.
   - Phương án B: Tái sử dụng bảng `player_inventory_items` với category `mortal_gu_swarm`.
   - Đánh giá ưu nhược điểm giữa việc gắn với `aperture_id` (khiến Đàn Cổ gắn liền với vận mệnh Tiên Khiếu) vs `player_id`.
2. **Ràng buộc toàn vẹn dữ liệu (Constraints & Checks)**:
   - Ràng buộc `CHECK (rank >= 1 AND rank <= 5)` để ngăn chặn tuyệt đối việc Tiên Cổ (Rank 6+) bị lọt vào stack.
   - Ràng buộc `CHECK (quantity >= 0)`.
   - Khóa ngoại liên kết `FOREIGN KEY (gu_id, rank) REFERENCES gu_dictionary(id, rank)`.
3. **Cơ chế Indexing**:
   - Cần đánh index nào để việc truy vấn toàn bộ danh sách Đàn Cổ của một Tiên Khiếu và tra cứu số lượng của 1 loại Cổ đạt tốc độ O(1)?
4. **Tác động đến `LoadPlayerGus` & Snapshot**:
   - Khi load snapshot của Cổ Tiên, Đàn Cổ có nằm trong `player.GuInventory` không, hay được tách thành một trường riêng `player.ApertureGuSwarms` (hoặc gắn vào `ApertureState`) để tránh làm phình JSON của các API status thông thường?

## Answer

Đã hoàn thành nghiên cứu kỹ thuật chuyên sâu và chốt phương án DDL cho Đàn Phàm Cổ Tiên Khiếu:

### 1. Quyết định Schema: Chọn Phương Án A (`aperture_gu_swarms`)
* **Lý do bắt buộc:** 
  - Trong `backend/internal/playerstate/snapshot.go` (`loadMaterials`), hệ thống hiện tại `SELECT item_id, quantity FROM player_inventory_items WHERE player_id = $1` mà **không lọc category**. Nếu dùng Phương án B, mọi đàn Phàm Cổ sẽ bị quét và nhồi vào `MaterialsInventory` của người chơi mỗi khi di chuyển hoặc đánh nhau, gây sụp đổ hiệu năng snapshot.
  - Về domain, Đàn Phàm Cổ là sinh thái sống của Tiên Khiếu, phải chịu chung số phận với `immortal_apertures(id)` khi Hạ Khiếu, Thu Khiếu, hoặc vỡ khiếu.
* **Cấu trúc DDL hoàn chỉnh:**
  ```sql
  CREATE TABLE IF NOT EXISTS aperture_gu_swarms (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      aperture_id UUID NOT NULL REFERENCES immortal_apertures(id) ON DELETE CASCADE,
      gu_id VARCHAR(50) NOT NULL,
      gu_rank INTEGER NOT NULL CHECK (gu_rank >= 1 AND gu_rank <= 5),
      quantity BIGINT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
      created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
      updated_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
      FOREIGN KEY (gu_id, gu_rank) REFERENCES gu_dictionary(id, rank) ON UPDATE CASCADE ON DELETE RESTRICT,
      CONSTRAINT uq_aperture_gu_swarms UNIQUE (aperture_id, gu_id, gu_rank)
  );

  CREATE INDEX IF NOT EXISTS idx_aperture_gu_swarms_gu ON aperture_gu_swarms (gu_id, gu_rank);
  ```

### 2. Ràng buộc toàn vẹn dữ liệu
* `CHECK (gu_rank >= 1 AND gu_rank <= 5)`: Ngăn chặn triệt để Tiên Cổ (Rank 6+) bị đưa vào stack; bảo toàn nguyên lý "Tiên Cổ độc nhất vô nhị".
* `CHECK (quantity >= 0)`: Chống lỗi số âm trong các thao tác trừ số lượng đồng thời.
* `FOREIGN KEY (gu_id, gu_rank) REFERENCES gu_dictionary(id, rank)`: Hợp lệ 100% vì `schemaDictionaries` chạy ở Bước 1 trước `schemaAperture` (Bước 17).

### 3. Cách ly hoàn toàn khỏi Snapshot Hotpath (0ms Overhead)
* Tuyệt đối không nhồi `aperture_gu_swarms` vào `playerstate.Load` hay `db.LoadPlayerGus`.
* Chỉ nạp on-demand qua endpoint chuyên biệt `GET /api/immortal-aperture/swarms` khi người chơi mở giao diện Quản lý Tiên Khiếu. Các tác vụ di chuyển, chiến đấu 20Hz, kiểm tra mana của Cổ Tiên hoàn toàn không chịu thêm bất kỳ chi phí I/O nào.

