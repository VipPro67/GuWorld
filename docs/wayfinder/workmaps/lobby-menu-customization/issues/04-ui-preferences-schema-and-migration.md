# Chốt mô hình dữ liệu ui-preferences và migration từ gu_ui_prefs_v1

Type: grilling
Status: resolved
Blocked by:
- [Chốt danh sách Nhóm Menu mặc định và phân bổ 25 mục tính năng](https://github.com/VipPro67/GuWorld/issues/456)
- [Chốt quy tắc hiển thị Lưới Lobby, giới hạn số lượng và nút Xem Tất Cả](https://github.com/VipPro67/GuWorld/issues/457)

Part of [Wayfinder: Tái cấu trúc Menu Lobby & Tùy biến Lưới Lobby](https://github.com/VipPro67/GuWorld/issues/455); canonical ticket: [Chốt mô hình dữ liệu ui-preferences và migration từ gu_ui_prefs_v1](https://github.com/VipPro67/GuWorld/issues/459).

## Question

Chốt mô hình dữ liệu trong `UiPreferences` (`frontend/src/lib/ui-preferences.ts`) và chiến lược migration từ `gu_ui_prefs_v1`:
- Cấu trúc schema mới cho danh sách mục/nhóm ở Lưới Lobby, thứ tự, trạng thái ẩn/hiện, và giới hạn số lượng.
- Giá trị mặc định cho người chơi mới.
- Cách migrate an toàn cho người chơi cũ đang có `lobbyItems` / `hiddenLobbyItems` trong `localStorage` để không bị lỗi hoặc mất cấu hình quan trọng.
