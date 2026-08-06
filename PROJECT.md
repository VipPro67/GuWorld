# GuWorld Project Status

Tài liệu này mô tả trạng thái codebase hiện tại, không phải backlog phase cũ.

## Tổng quan

GuWorld là web RPG text-based có backend Go/Gin và frontend Next.js. Gameplay xoay quanh người chơi tu luyện, quản lý Cổ trùng, khám phá map, chiến đấu PvE, giao dịch, gia nhập môn phái và phát triển lên Cổ Tiên.

## Kiến trúc hiện tại

- `backend/cmd/server/main.go`: entrypoint backend, load `.env`, kết nối PostgreSQL, khởi tạo schema/seed, kết nối Redis nếu có, load registry Cổ/vật phẩm, chạy worker và Gin router.
- `backend/internal/server/server.go`: khai báo toàn bộ route `/api`.
- `backend/internal/db/db.go`: schema PostgreSQL, migration dạng `CREATE/ALTER IF NOT EXISTS`, seed CSV.
- `backend/internal/models`: struct domain cho player, Cổ, item, recipe, combat, aperture, monster, cell, feedback, killer move.
- `backend/internal/handlers`: logic API theo từng module gameplay.
- `backend/internal/worker`: worker đói Cổ, worker đói sủng vật.
- `backend/internal/cache`: Redis client optional.
- `frontend/src/app`: Next.js App Router pages/layout.
- `frontend/src/store/useGameStore.ts`: Zustand store, API client phía frontend và state gameplay.
- `frontend/src/components`: UI module gameplay.

## Module backend đang có

### Auth, player và profile

- Đăng ký, đăng nhập, đăng xuất, `/auth/me`.
- Google OAuth callback.
- Hồ sơ công khai, chỉnh username/bio/mật khẩu, ngụy trang thông tin, avatar upload.
- Referral, mô phỏng deposit, SSE player events, tick regen.

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
- Combat PvE dùng Cổ, charge, target theo hàng/cột/vị trí, status effects, loot tạm.
- Bắt quái vật làm sủng vật (Monster Capture): rank check, hpFactor, typeFactor, capture bonus.
- Sủng vật (Pets): CRUD, feed, formation, sleep, merge, hunger worker, chiến đấu tự động.
- Party tạo/mời/rời.
- Instance template/session/action.

### Tiên khiếu và độ kiếp nâng cao

- Immortal aperture, node tài nguyên, thu hoạch/gỡ/xóa/sửa node.
- Time rate, lao động dị nhân, conflict/ecosystem node.
- Dao marks.
- Active tribulation, enemy grid, attack/resolve, worker tự trigger/process.

### Economy và social

- Shop NPC buy/sell.
- Bảo Hoàng Thiên: listing market/auction/trade, bid, buy, cancel, settle expired auctions.
- P2P trade: initiate/accept/decline/cancel/offer/lock/confirm + SSE events.
- Chat/mail private, Bảo Hoàng chat + SSE.
- Wallet, topup package/order, PayOS webhook, premium package shop, admin wallet/topup listing.

### Môn phái, mỏ, arena, feedback

- Môn phái: list/join/leave/create, cấp bậc, đóng góp, kho, nhiệm vụ, thành viên, duyệt đơn, nâng cấp.
- Mỏ cá nhân: chiếm, claim, bỏ, tấn công; worker gán quái cho mỏ.
- Mỏ môn phái: claim/tấn công.
- Arena leaderboard/challenge.
- Feedback: submit/list/vote/update status.

## Frontend đang có

- Lobby/dashboard chính và điều hướng view.
- `Aperture` cho trạng thái người chơi, tu luyện, quest, hồ sơ, referral.
- `GuInventory`, `RefiningFurnace`, `KillerMoveBuilder`.
- `ExplorationMap`, `ExploreCard`, `CombatScreen`, `DungeonView`.
- `ImmortalApertureView`, `MineView`, `SectView`, `ArenaView`.
- `BaoHoangMarket`, `TradeRoomModal`, `CaravanShop`.
- `PremiumWallet`, `PremiumShopView`, `TopupView`.
- `FeedbackView`, `PlayerProfileModal`, toast/log UI.
- `PetPanel`: quản lý sủng vật (formation, feed, sleep, release, merge).

## Dữ liệu seed hiện tại

Dữ liệu runtime được seed từ CSV trong `backend/`:

- `Gu.csv`: 227 Cổ (bổ sung 13 Cổ trùng đặc biệt).
- `Item.csv`: 136 vật phẩm/nguyên liệu/tiền tệ.
- `Recipe.csv`: 31 công thức luyện.
- `Monster.csv`: 13 template quái.
- `LootTable.csv`: 8 bảng loot.
- `Region.csv`: 3 vùng.
- `MonsterSpawn.csv`: 1 spawn rule.
- `ShopItem.csv`: 15 mặt hàng shop NPC.
- `PremiumShopPackage.csv`: 4 gói premium shop.
- `TopupPackage.csv`: 5 gói nạp.

## Tài liệu liên quan

- API: `docs/API.md`.
- Kiến trúc: `docs/ARCHITECTURE.md`.
- Dữ liệu: `docs/GAME_DATA.md`.
- Hướng dẫn phát triển: `docs/DEVELOPMENT.md`.
- Wiki gameplay: `WIKI.md`.
