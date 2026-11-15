# Kinh tế và giao dịch

## Tính năng làm gì?

Kinh tế gồm shop luân phiên, merchant cung cầu động, marketplace/Bảo Hoàng, auction, buyback, giao dịch trực tiếp và wallet.

## Điều kiện mở và quyền truy cập

Các thao tác cần đăng nhập, tài sản hợp lệ và currency phù hợp. Marketplace dùng Primeval Stones dưới Rank 6 và Immortal Stones từ Rank 6. Listing/bid/trade còn bị giới hạn bởi ownership, trạng thái Gu và quy tắc thị trường.

## Cách sử dụng

Mua/bán tại shop hoặc merchant; dùng marketplace để đăng listing, bid, cancel hoặc settle; dùng P2P trade để tạo phòng, đưa offer, lock tài sản rồi hai bên confirm. Trên bản đồ, chọn người chơi cùng khu vực trong phạm vi 10 ô để mở giao dịch tại chỗ mà không cần Cổ liên lạc. Khởi tạo giao dịch từ chat là giao dịch từ xa và yêu cầu cả hai người chơi có Thông Thiên Cổ cùng Thần Niệm Cổ. BaoHoang hỗ trợ listing/bid/cancel/settlement theo lifecycle riêng.

## Luật, chi phí và giới hạn

### Giá merchant và phí marketplace

Merchant buy ratio giảm tuyến tính theo lượng đã mua: nếu quota dương, `ratio = start + (floor-start)*clamp(bought/quota,0,1)`; quota <=0 thì không giảm. Unit price là base price nhân ratio, quantize theo precision currency, tối thiểu 0.001. `start` và `floor` đến từ merchant profile, nên không có một tỷ lệ chung.

Bảo Hoàng MARKET thu phí listing 4% tổng giá và commission 6%; AUCTION commission 15%. Auction request hiện giới hạn duration 1–24 giờ, còn helper có các tier phí upfront cho thời hạn dài hơn nhưng validation khiến các tier đó không thể đạt. TRADE không thu upfront fee/commission theo helper hiện tại. Async offers có duration 1–168 giờ, tối đa 10 offer active/player. Số tiền cuối cùng dùng theo preview/response server.

- Shop có rotation theo giờ.
- Merchant thay đổi supply/demand theo thời gian và khoảng cách.
- **Thương Nhân Tân Thủ (Tân Thủ Cốc):** Các vùng SAFE_ZONE Tân Thủ Cốc (`tan_thu_coc_*`) được bố trí thương nhân cố định ("Thương Nhân Tân Thủ", profile `tan_thu_merchant`) tại tọa độ trung tâm an toàn `(500, 500)`. Thương nhân này cung cấp đan dược hồi phục sơ cấp, thức ăn Cổ trùng các loại, nguyên liệu cấp 1 và Cổ trùng khởi đầu Chuyển 1; đồng thời thu mua tài nguyên, vật phẩm, công thức và Cổ trùng đến Chuyển 3 với cơ chế giá động công bằng. Tọa độ tương tác thương nhân tại Tân Thủ Cốc được mở rộng bán kính 15 ô để Cổ sư trong khu vực an toàn dễ dàng giao dịch, đồng thời hiển thị biểu tượng thương nhân tương tác được trên bản đồ thám hiểm.
- P2P trade phải đi qua state machine initiate/accept/offer/lock/unlock/confirm; không coi lời mời đơn lẻ là giao dịch hoàn tất.
- `POST /trade/initiate` nhận `source: "map"` để áp dụng điều kiện cùng khu vực, tối đa 10 ô; `source: "chat"` (mặc định nếu client cũ không gửi) yêu cầu hai Cổ giao liên lạc ở cả hai phía.
- Gu bị reserved, dead, equipped hoặc không thuộc người chơi có thể không được đưa vào offer/listing.
- Phí, giá, quota và trạng thái cuối do server response xác định.

## Kết quả và trường hợp thất bại

Giá/hàng thay đổi, hết hạn, không đủ currency, ownership mismatch hoặc trade state conflict sẽ từ chối thao tác. Settlement chỉ chuyển tài sản khi validation cuối thành công.

## Liên hệ với hệ thống khác

Inventory/Gu lifecycle quyết định tài sản giao dịch được; Rank quyết định currency; realtime chỉ hỗ trợ đồng bộ trade room, không thay thế xác nhận server.

## Dữ liệu/cấu hình ảnh hưởng

Shop stock, merchant demand/pricing, marketplace listings, auction state và wallet ledger được lưu/cấu hình theo domain.

## Mã nguồn liên quan

- Backend: `backend/internal/handlers/merchant.go`, `marketplace*.go`, `async_trade_handlers.go`, `trade_*.go`, `buyback_handlers.go`, `premium.go`.
- Frontend: `frontend/src/store/slices/marketplaceSlice.ts`, `tradeSlice.ts`; `ExplorationMap.tsx`, `GlobalChatOverlay.tsx`, `PlayerProfileModal.tsx`, `MerchantTradeView.tsx`, `BaoHoangMarket.tsx`.

## API liên quan

Shop, merchant, marketplace, auction, trade và wallet routes được đăng ký tại `backend/internal/server/server.go`; xem [API guide](../guides/API.md).

## Ví dụ flow

```text
mở trade room → hai bên accept → chọn offer → lock
→ kiểm tra tài sản/số dư → cả hai confirm → server settle và đồng bộ inventory
```
