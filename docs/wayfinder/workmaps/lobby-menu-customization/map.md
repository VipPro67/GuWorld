# Wayfinder Map: Tái cấu trúc Menu Lobby & Tùy biến Lưới Lobby

## Destination

Bản đặc tả thiết kế hoàn chỉnh (ready-to-implement spec) cho hệ thống **Nhóm Menu** và **Lưới Lobby** tùy biến theo thiết bị (`localStorage`), giúp thu gọn màn hình Lobby từ 25 mục hiện tại thành các nhóm gọn gàng và cho phép người chơi tùy chỉnh hiển thị, số lượng, thứ tự các mục ngoài Lobby.

## Notes

- Domain: frontend Lobby UI (`frontend/src/components/LobbyView.tsx`, `LobbySettingsPanel.tsx`, `UtilitiesView.tsx`), local UI preferences (`frontend/src/lib/ui-preferences.ts`), và responsive layout (mobile + desktop).
- Current state: `LobbyView` đang render 25 mục chia làm `managementItems` (16) và `activityItems` (9). `ui-preferences.ts` đã có sẵn `lobbyItems` (thứ tự) và `hiddenLobbyItems` (ẩn/hiện) trong `localStorage` (`gu_ui_prefs_v1`), và `UtilitiesView` đã tồn tại.
- Decisions from initial charting:
  - Gộp thành **1 Lưới Lobby thống nhất** (thay cho 2 cụm cứng hiện tại ở màn hình chính Lobby).
  - **Nhóm Menu** mở dạng **Modal / Bottom Sheet** ngay tại Lobby để chọn nhanh tính năng con.
  - Đưa 1 **Mục tính năng** lẻ ra **Lưới Lobby** hoạt động như ghim/shortcut; mục đó vẫn giữ trong **Nhóm Menu** để tránh người chơi bị "mất" tính năng.
  - Cấu hình lưu **theo thiết bị (`localStorage`)** qua `ui-preferences.ts`, không thay đổi DB/API backend.
- Skills to consult: `grilling`, `domain-modeling`, `prototype`.
- Standing preferences: giữ phong cách tiên hiệp/cổ phong hiện tại (`lobbyArtwork`), giữ logic khóa theo cấp (`Tiên Khiếu` rank >= 6) và nhãn động (`explore` khi ở `aperture_interior`).
- Canonical tracker: [Wayfinder: Tái cấu trúc Menu Lobby & Tùy biến Lưới Lobby](https://github.com/VipPro67/GuWorld/issues/455). Local files are context copies; GitHub issues are canonical.

## Decisions so far

- [Chốt danh sách Nhóm Menu mặc định và phân bổ 25 mục tính năng](https://github.com/VipPro67/GuWorld/issues/456): Giữ 7 mục cốt lõi ngoài Lưới Lobby và gom 18 mục còn lại vào 4 Nhóm Menu mặc định (tổng 11 ô), đổi nhãn `utilities` thành "Phong Ấn Cổ Trùng".
- [Chốt quy tắc hiển thị Lưới Lobby, giới hạn số lượng và nút Xem Tất Cả](https://github.com/VipPro67/GuWorld/issues/457): Giới hạn 4 mức (`6`/`8`/`12`/`Tất cả`, mặc định `12`), cắt theo thứ tự top `N`, nút "Xem tất cả" ở header `"Điểm Đến Cổ Giới"`, giữ nguyên lưới responsive hiện tại.
- [Thiết kế UX và prototype cho Modal Nhóm Menu và Cài đặt Lưới Lobby](https://github.com/VipPro67/GuWorld/issues/458): Duyệt prototype HTML (`03_lobby_menu_ux_prototype.html`) với ô nhóm có badge `{N} mục`, Modal/Bottom Sheet nhóm hỗ trợ ghim nhanh, Modal "Xem tất cả (25)" có tìm kiếm, và Cài đặt Lưới dùng nút `▲`/`▼` + đánh dấu `Ngoài top N`.
- [Chốt mô hình dữ liệu ui-preferences và migration từ gu_ui_prefs_v1](https://github.com/VipPro67/GuWorld/issues/459): Dùng `lobbyGridItems`, `lobbyGridLimit`, `lobbyMenuVersion: 2` trong `gu_ui_prefs_v1`; tự nâng cấp sang 11 ô mặc định nếu chưa từng tùy biến ẩn mục, cho phép rỗng và fallback limit về `12`.

## Not yet specified

- Không còn câu hỏi nào trong phạm vi cần quyết định trước khi triển khai.

## Out of scope

- Đồng bộ cấu hình menu Lobby theo tài khoản lên server / PostgreSQL.
- Thay đổi logic nghiệp vụ bên trong từng màn hình tính năng con (Võ Đài, Chợ, Túi Đồ, v.v.).
- Thay đổi thanh điều hướng ngoài Lobby (World HUD / Combat HUD).
