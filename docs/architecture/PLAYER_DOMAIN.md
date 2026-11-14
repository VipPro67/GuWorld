# Miền Player

Tài liệu này mô tả ranh giới sở hữu dữ liệu player để tránh các handler tự đọc/ghi cùng một trạng thái theo cách không nhất quán.

## Thành phần sở hữu

- Route đăng ký tại `backend/internal/server/server.go`.
- `backend/internal/playerstate/` định nghĩa projection chuẩn cho toàn bộ hàng `players`, projection theo nhu cầu, revision bộ sưu tập và một số lazy repair.
- `PlayerService` là service dùng chung cho thao tác player được nối trong `handlers/services.go`; không phải mọi handler đều đi qua service.
- Luật gameplay và nhiều mutation vẫn ở `backend/internal/handlers/`.
- Các bảng domain như inventory, Gu, quest, achievement, aperture và combat giữ dữ liệu chuyên biệt của chúng; không nhồi dữ liệu đó vào projection `players` nếu schema/domain đã có chủ sở hữu riêng.

## Quy tắc đọc

- Dùng projection chuẩn khi cần trạng thái player đầy đủ hoặc khi serialization phải nhất quán với snapshot.
- Projection hẹp vẫn phù hợp nếu caller chỉ cần một nhóm field xác định; không mở rộng mọi query chỉ để dùng một kiểu chung.
- Cần phân biệt trạng thái lưu bền vững với giá trị tính toán từ thời gian như regen, tuổi thọ hoặc trạng thái hết hạn.
- Collection revision dùng để phát hiện/cập nhật thay đổi bộ sưu tập. Không tự tăng revision ở nơi không sở hữu mutation tương ứng.

## Quy tắc ghi

1. Xác định bảng/domain sở hữu field và đọc schema tương ứng trong `backend/internal/db/schema_*.go`.
2. Giữ các cập nhật liên quan trong cùng transaction khi có nhiều bản ghi cần đồng bộ.
3. Tôn trọng compare-and-swap/revision và khóa hiện có; player state được cập nhật thường xuyên.
4. Sau mutation, phát snapshot/event theo flow domain đang dùng để frontend không phải tự suy đoán trạng thái server.
5. Thêm hoặc cập nhật test ở package sở hữu hành vi.

Player đổi vị trí phải đi qua `handlers/movement.go`, dùng chung kiểm tra quyền/đích, ghi vị trí và phát sự kiện. Không tự viết một nhánh cập nhật tọa độ riêng trong handler khác.

## Frontend và đồng bộ

`useGameStore` nhận response và player events rồi cập nhật state thông qua `setPlayerFromServer` cùng các helper hiện có. Tái sử dụng luồng này khi API trả snapshot player. `authFetch` xử lý 401 tập trung; slice không thêm chính sách logout riêng.

## Trước khi đổi

- Đọc các handler/route liên quan và schema domain.
- Kiểm tra caller cần full-row projection hay projection hẹp.
- Kiểm tra mutation có ảnh hưởng event, collection revision, inventory hoặc snapshot frontend không.
- Chạy test backend phù hợp; dùng `go test -race ./...` khi thay đổi concurrency.
