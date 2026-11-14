# Realm và instance

## Tính năng làm gì?

Realm/instance tạo hoạt động có session và giới hạn entry riêng, gồm Mộng Cảnh, Bí Cảnh Không Gian và truyền thừa.

## Điều kiện mở và quyền truy cập

Template có thể yêu cầu rank, entry item, cooldown hoặc vùng/portal cụ thể. Player đang có instance session không thể mở session khác.

## Cách sử dụng

Chọn hoạt động khả dụng, đáp ứng entry requirements rồi tạo session. Với spatial realm, đi tới đúng ENTRY_PORTAL tile để vào map riêng; altar chỉ dùng khi guardian đã bị hạ và player đứng đủ gần.

## Luật, chi phí và giới hạn

### Bonus Mộng Cảnh

Trong ActionInstance, Dream Dao marks giảm damage tối đa `min(30%, dreamMarks/1000*10%)`; Gu path Dream đang ACTIVE/equipped cộng thêm 25% reduction; tổng reduction cap 50%. Bonus reward Mộng Đạo hiện là `int(baseReward * (25% + dreamMarks/2000))` nếu reward có path Dream hoặc player có Dream marks dương. Template vẫn sở hữu entry item, rank, cooldown và reward chính.

- DREAM_REALM và SECRET_REALM dùng chung quota 5 lần/ngày, không phải 5 lần mỗi loại.
- Một active session chặn session instance khác.
- Entry item, cooldown, rank và template rules được backend xác thực.
- Spatial realm entry cần đúng portal tile; altar ở khoảng Chebyshev 2 và guardian đã clear.

## Kết quả và trường hợp thất bại

Quota hết, thiếu vật phẩm, sai rank/cooldown hoặc đang có session sẽ từ chối entry. Session lưu tiến độ/kết quả và áp dụng reward theo template.

## Liên hệ với hệ thống khác

World movement đưa player tới portal; combat xử lý guardian/encounter; quest và progression nhận reward. Party có thể bị hạn chế theo template.

## Dữ liệu/cấu hình ảnh hưởng

Instance templates, entry requirements, cooldown, map/altar và reward tables quyết định behavior. Backend quota là chuẩn; UI fallback không phải luật gameplay.

## Mã nguồn liên quan

- Backend: `backend/internal/handlers/instance.go`, `tower.go` và spatial realm handlers.
- Frontend: `frontend/src/store/slices/instanceSlice.ts` và realm UI.

## API liên quan

Instance, dream realm, secret realm và spatial realm routes; xem `backend/internal/server/server.go` và [API guide](../guides/API.md).

## Ví dụ flow

```text
kiểm tra template/rank/item/quota → tạo session → vào portal đúng tile
→ đánh guardian → tới altar trong bán kính → nhận kết quả/reward
```
