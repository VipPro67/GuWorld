# Môn phái

## Tính năng làm gì?

Môn phái là tổ chức người chơi có thành viên, đóng góp, kho, nhiệm vụ, lãnh địa và hoạt động chiến sự.

## Điều kiện mở và quyền truy cập

Tạo môn phái yêu cầu Rank 3 trở lên và 30,000 Stones. Thao tác quản lý phụ thuộc vai trò/quyền thành viên; gia nhập có thể yêu cầu lời mời hoặc phê duyệt theo trạng thái môn phái.

## Cách sử dụng

Tạo hoặc tìm môn phái, nộp yêu cầu gia nhập, rồi dùng các tab thành viên, đóng góp, kho và nhiệm vụ. Lãnh đạo/người có quyền quản lý xử lý thành viên, hoạt động và lãnh địa.

## Luật, chi phí và giới hạn

- Tạo môn phái: Rank tối thiểu 3 và phí 30,000 Primeval Stones.
- Radius lãnh địa ban đầu là base radius 5 cộng 1, tổng 6; placement còn kiểm tra địa hình/ownership nên không phải mọi ô trong bán kính đều claim được.
- Rank tối thiểu 3, phí tạo 30,000 Stones.
- Các hành động kho/đóng góp/nhiệm vụ kiểm tra quyền, giới hạn và số dư trên server.
- Chiến sự/lãnh địa có lifecycle riêng; trạng thái hiển thị không đồng nghĩa action luôn khả dụng.

## Kết quả và trường hợp thất bại

Thiếu rank/currency, không đủ quyền hoặc trạng thái membership không phù hợp sẽ từ chối thao tác. Contribution và kho chỉ cập nhật sau khi server xác nhận.

## Liên hệ với hệ thống khác

Membership ảnh hưởng party scouting, quyền truy cập một số aperture và hoạt động nhóm. Lãnh địa dùng world/combat systems.

## Dữ liệu/cấu hình ảnh hưởng

Môn phái, membership, contribution, vault, mission và territory được lưu trong các bảng domain; admin có công cụ quản trị tương ứng.

## Mã nguồn liên quan

- Backend: `backend/internal/handlers/sect.go`, `sect_members.go`, `sect_vault.go`, `sect_missions.go`, `sect_warfare.go`, `admin_sects.go`.
- Frontend: `frontend/src/components/SectView.tsx`, `frontend/src/store/slices/sectSlice.ts`.

## API liên quan

Nhóm `/sects/*`; tra router `backend/internal/server/server.go` và [API guide](../guides/API.md).

## Ví dụ flow

```text
Rank 3 + 30,000 Stones → tạo môn phái → mời thành viên
→ thành viên gia nhập → đóng góp/kho/nhiệm vụ theo quyền được cấp
```
