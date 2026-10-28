# GuWorld Frontend

Frontend là ứng dụng Next.js App Router dùng React, TypeScript, Zustand và Tailwind/PostCSS.

## Chạy local

```bash
npm install
npm run dev
```

Mở `http://localhost:3000`.

## Scripts

| Script | Mục đích |
| --- | --- |
| `npm run dev` | Chạy Next.js dev server. |
| `npm run build` | Build production. |
| `npm run start` | Chạy production server sau khi build. |
| `npm run lint` | Chạy ESLint. |

## Backend URL

Frontend gọi backend qua:

```text
NEXT_PUBLIC_BACKEND_URL=http://localhost:8080
```

Nếu biến này không tồn tại, store dùng mặc định `http://localhost:8080`.

## Cấu trúc chính

| Đường dẫn | Vai trò |
| --- | --- |
| `src/app/page.tsx` | Shell chính, login/register và điều hướng các view gameplay. |
| `src/app/topup/page.tsx` | Trang nạp tiền. |
| `src/app/topup/success/page.tsx` | Trang kết quả nạp thành công. |
| `src/store/useGameStore.ts` | Zustand store, API calls, SSE subscriptions, derived mana/HP estimate. |
| `src/components/Aperture.tsx` | View nhân vật/không khiếu, tu luyện, profile, quest. |
| `src/components/GuInventory.tsx` | Inventory Cổ. |
| `src/components/RefiningFurnace.tsx` | Luyện Cổ/suy diễn recipe. |
| `src/components/ExplorationMap.tsx` | Bản đồ lưới đơn giản (vị trí hiện tại + bấm ô di chuyển, chế độ Cài Đặt Chung). |
| `src/components/DungeonView.tsx` | Instance/phó bản. |
| `src/components/ImmortalApertureView.tsx` | Tiên khiếu, node, tai kiếp. |
| `src/components/SectView.tsx` | Môn phái. |
| `src/components/BaoHoangMarket.tsx` | Marketplace/auction/trade listings. |
| `src/components/TradeRoomModal.tsx` | Giao dịch P2P. |
| `src/components/ArenaView.tsx` | Arena. |
| `src/components/PremiumShopView.tsx` / `TopupView.tsx` / `PremiumWallet.tsx` | Ví, premium shop, topup. |
| `src/components/FeedbackView.tsx` | Feedback board. |

## Ghi chú phát triển

- API calls dùng `credentials: "include"`, backend cần CORS đúng origin frontend.
- View mới nên đi qua `currentView` trong Zustand store nếu là tab chính.
- Khi thêm route backend mới, cập nhật `docs/API.md` ở root repo.
- Khi thêm state/action lớn trong store, cập nhật README này và `PROJECT.md` nếu là feature người dùng nhìn thấy.
