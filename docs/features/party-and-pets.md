# Party và sủng vật

## Tính năng làm gì?

Party ghép người chơi trong world; formation có thể bao gồm player, pet và NPC. Pet là actor có lifecycle riêng, có thể bắt từ encounter khi điều kiện cho phép.

## Điều kiện mở và quyền truy cập

Party tối đa 4 người chơi. Lời mời cần được chấp nhận khi ở cùng map và trong phạm vi Chebyshev cho phép tính từ leader. Không thể gia nhập party trong instance. Capture phụ thuộc loại encounter và trạng thái mục tiêu.

## Cách sử dụng

Leader tạo/invite, người chơi nhận lời khi đủ điều kiện, sau đó sắp xếp formation. Trong encounter, roster hợp lệ được đưa vào actor list. Pet được quản lý qua các thao tác bắt, xem, thả hoặc sử dụng theo UI/API.

## Luật, chi phí và giới hạn

- Invite range là 100 ô; acceptance so sánh Chebyshev distance, nên cần `max(|dx|,|dy|) <= 100`.
- Party cap hiện tại là 4 player.
- Chấp nhận lời mời yêu cầu cùng map và khoảng cách giới hạn.
- Party membership/formation không vượt qua giới hạn instance hoặc combat.
- Pet/NPC cần actor state hợp lệ; capture không áp dụng với mọi monster.

## Kết quả và trường hợp thất bại

Invite hết hạn, khác map, quá xa hoặc đang ở instance sẽ bị từ chối. Actor không hợp lệ bị loại khỏi formation/encounter theo validation server.

## Liên hệ với hệ thống khác

Party ảnh hưởng inspect, world encounter, scouting và quyền vào một số nội dung. Pets tham gia combat thông qua simulator và actor roster.

## Dữ liệu/cấu hình ảnh hưởng

Capture profile, monster data, pet state và formation rules quyết định loại actor có thể tham gia.

## Mã nguồn liên quan

- Backend: `backend/internal/handlers/party.go`, `pet*.go`, combat actor/roster handlers.
- Frontend: `frontend/src/store/slices/partySlice.ts`, `petsSlice.ts`.

## API liên quan

Party và pets routes được đăng ký trong `backend/internal/server/server.go`; xem [API guide](../guides/API.md).

## Ví dụ flow

```text
leader gửi invite → thành viên cùng map/đủ gần accept → formation cập nhật
→ world encounter mở → backend tạo roster player + pet hợp lệ
```
