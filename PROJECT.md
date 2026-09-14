# GuWorld Project Status

Tài liệu này mô tả trạng thái codebase hiện tại, không phải backlog phase cũ.

## Tổng quan

GuWorld là web RPG text-based có backend Go/Gin và frontend Next.js. Gameplay xoay quanh người chơi tu luyện, quản lý Cổ trùng, khám phá map, chiến đấu PvE, giao dịch, gia nhập môn phái và phát triển lên Cổ Tiên.

## Kiến trúc hiện tại

- `backend/cmd/server/main.go`: entrypoint backend, load `.env`, kết nối PostgreSQL/Redis, khởi tạo schema, load các registry từ DB, chạy worker và Gin router/WebSocket hub.
- `backend/internal/server/server.go`: khai báo toàn bộ route `/api`.
- `backend/internal/db/db.go` + `backend/internal/db/schema_*.go`: schema PostgreSQL, patch tương thích dạng `CREATE/ALTER IF NOT EXISTS` và seed hệ thống tối thiểu. Dictionary gameplay được import qua Admin.
- `backend/internal/models`: struct domain cho player, Cổ, item, recipe, combat, aperture, monster, cell, feedback, killer move.
- `backend/internal/handlers`: điều phối HTTP và phần lớn luật gameplay theo từng domain.
- `backend/internal/services`: các module dùng chung đã được tách cho player, inventory, map, combat session, recipe, quest, achievement và đạo ngân; không phải mọi handler đều đi qua service.
- `backend/internal/worker`: polling/batch worker cho đói/tăng trưởng, persistence, môn phái và mô phỏng xã hội tiên khiếu.
- `backend/internal/cache`: Redis client (bắt buộc).
- `frontend/src/app`: Next.js App Router pages/layout.
- `frontend/src/store/useGameStore.ts`: Zustand store, API client phía frontend và state gameplay.
- `frontend/src/components`: UI module gameplay.

## Module backend đang có

### Auth, player và profile

- Đăng ký, đăng nhập, đăng xuất, `/auth/me`.
- Google OAuth callback.
- Hồ sơ công khai, chỉnh username/bio/mật khẩu, ngụy trang thông tin, avatar upload.
- Referral, gift code, notification, audit log cá nhân, ví và tick regen.

### Tu luyện và sinh tồn

- Phá vách (`break-wall`) cho Rank 1-5.
- Dùng Xá Lợi Cổ.
- Thăng Tiên và Độ Kiếp cơ bản.
- Recharge bằng nguyên thạch, clear stress, quest completion.
- Hồi mana/HP dựa trên thời gian và hệ số tốc độ.

### Cổ, vật phẩm, luyện Cổ, sát chiêu

- Dictionary Cổ/vật phẩm public.
- Inventory: feed, feed group, equip, sleep/active/dead status, set vital Gu, consume/support/drop.
- Item effects: hồi mana/HP, soul, đạo ngân, xóa trạng thái xấu và điều kiện mortal/immortal.
- Refining: luyện theo recipe và suy diễn recipe.
- Killer move: preview, tạo, liệt kê, xem, xóa, cast trong combat.

### Khám phá, combat, phó bản

- Grid map theo vùng, di chuyển, explore node, retreat.
- Combat PvE/PvP dùng Cổ, charge, target theo hàng/cột/vị trí, status effects, loot vào túi và ground drop khi tử vong.
- Bắt quái vật làm sủng vật (Monster Capture): rank check, hpFactor, typeFactor, capture bonus.
- Sủng vật (Pets): CRUD, feed, formation, sleep, merge, hunger worker, chiến đấu tự động.
- Party tạo/mời/phản hồi lời mời/rời; combat luyện tập và lịch sử trận đấu.
- Instance template/session/action.

### Tiên khiếu và độ kiếp nâng cao

- Immortal aperture, node tài nguyên, thu hoạch/gỡ/xóa/sửa node.
- Time rate, nâng cấp/sửa node, phân công lao động dị nhân, hàng đợi luyện chế, khu định cư, xã hội và NPC dị nhân.
- Dao marks.
- Active tribulation, enemy grid, attack/resolve, worker tự trigger/process.

### Economy và social

- Shop NPC buy/sell.
- Bảo Hoàng Thiên: listing market/auction/trade, bid, buy, cancel, settle expired auctions.
- P2P trade: initiate/accept/decline/cancel/offer/lock/confirm + cập nhật realtime qua WebSocket.
- Chat/mail private, Bảo Hoàng chat và sự kiện realtime qua WebSocket.
- Wallet, topup package/order, PayOS webhook, premium package shop, admin wallet/topup listing.

### Môn phái, mỏ, arena, feedback

- Môn phái: list/join/leave/create, cấp bậc, đóng góp, kho, nhiệm vụ, thành viên, duyệt đơn, nâng cấp.
- Mỏ cá nhân: chiếm, claim, bỏ, tấn công; worker gán quái cho mỏ.
- Mỏ môn phái: claim/tấn công.
- Võ đài leaderboard/challenge/shop, Đấu Trường Sinh Tử theo giải, gacha, thành tựu và bảng xếp hạng nhiều hạng mục.
- Cược linh thú, thông báo, lịch sử chiến đấu và nhật ký hành động.
- Feedback: submit/list/vote/update status.

## Frontend đang có

- Lobby/dashboard chính và điều hướng view.
- `Aperture` cho trạng thái người chơi, tu luyện, quest/thành tựu, hồ sơ, referral và gift code.
- `GuInventory`, `RefiningFurnace`, `KillerMoveBuilder`.
- `ExplorationMap`, `ExploreCard`, `CombatScreen`, `DungeonView`.
- `ImmortalApertureView`, `MineView`, `SectView`, `ArenaView`, `DeathArenaView` và bảng xếp hạng.
- `BaoHoangMarket`, `TradeRoomModal`, `CaravanShop`.
- `PremiumWallet`, `PremiumShopView`, `TopupView`.
- `FeedbackView`, `PlayerProfileModal`, toast/log UI.
- `PetPanel`: quản lý sủng vật (formation, feed, sleep, release, merge).

## Dữ liệu gameplay hiện tại

Nguồn chuẩn nằm trong `backend/data/`, gồm dictionary Cổ/vật phẩm/công thức/quái/loot, vùng và ô bản đồ, spawn, chủng tộc/thể chất/trạng thái, quest, môn phái, các shop, gói nạp, node tiên khiếu và mộng cảnh. Không ghi số dòng tại đây vì dataset thay đổi thường xuyên.

`InitSchema()` chỉ tạo schema và seed các bản ghi bắt buộc để server hoạt động (system player, hub, sentinel dictionary, mỏ, wallet/leaderboard backfill). Quản trị viên import/export dictionary ở tab **Dữ liệu**. Registry trong bộ nhớ được load từ DB lúc startup, vì vậy cần restart sau các import ảnh hưởng registry.

## Tài liệu liên quan

- API: `docs/API.md`.
- Kiến trúc: `docs/ARCHITECTURE.md`.
- Dữ liệu: `docs/GAME_DATA.md`.
- Hướng dẫn phát triển: `docs/DEVELOPMENT.md`.
- Wiki gameplay: `WIKI.md`.
