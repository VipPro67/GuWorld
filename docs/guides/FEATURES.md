# Mục lục tính năng

Tài liệu người chơi bắt đầu từ [WIKI.md](../../WIKI.md). Các trang bên dưới đi sâu vào flow và nối từng tính năng với UI, API, handler, schema hoặc dataset hiện tại.

| Domain | Tài liệu | Tính năng |
| --- | --- | --- |
| Tài khoản và tu luyện | [accounts-and-cultivation.md](../features/accounts-and-cultivation.md) | Đăng ký, tutorial/onboarding, hồ sơ, phá vách, Thăng Tiên |
| Nhiệm vụ và phần thưởng | [progression-and-rewards.md](../features/progression-and-rewards.md) | Quest, nhiệm vụ ngày, thành tựu, referral, gift code |
| Cổ trùng và inventory | [gu-and-inventory.md](../features/gu-and-inventory.md) | Túi đồ, feed/equip, trạng thái, durability, bản mệnh |
| Luyện Cổ | [refining.md](../features/refining.md) | Cổ phương, suy diễn/cảm ngộ, phiên luyện và minigame |
| Sản phẩm chiến đấu | [killer-moves-and-gu-houses.md](../features/killer-moves-and-gu-houses.md) | Sát Chiêu, Tiên Cổ Ốc, composition/reservation |
| Thế giới | [world-and-exploration.md](../features/world-and-exploration.md) | Map, movement, portal, ground drop, tài nguyên, teleport |
| Thiên Cơ | [scouting.md](../features/scouting.md) | Inspect, divination, tracking marks, sever clues, resonance |
| Combat | [combat.md](../features/combat.md) | Encounter, auto/session, PvP, action plans, flee, capture, history |
| Party và sủng vật | [party-and-pets.md](../features/party-and-pets.md) | Party, formation, capture/pet lifecycle, actor roster |
| Tiên Khiếu | [immortal-aperture.md](../features/immortal-aperture.md) | Node, resource points, workers, NPC, swarms, portal, tribulation |
| Realm | [realms-and-instances.md](../features/realms-and-instances.md) | Mộng Cảnh, Bí Cảnh Không Gian, truyền thừa, instance |
| Môn phái | [sects.md](../features/sects.md) | Thành viên, đóng góp, kho, nhiệm vụ, lãnh địa và chiến sự |
| Kinh tế | [economy-and-trading.md](../features/economy-and-trading.md) | Cửa hàng, merchant, marketplace, auction, buyback, P2P trade, wallet |
| Đấu trường và sự kiện | [arenas-and-events.md](../features/arenas-and-events.md) | Võ Đài, Tháp, hai giải đấu, gacha, cược linh thú |
| Xã hội và tiện ích | [social-and-account-tools.md](../features/social-and-account-tools.md) | Chat/mail, notification, leaderboard, feedback, profile, audit |

## Quy ước đọc

- Route/API là bề mặt gọi; handler/service/simulator quyết định luật thực tế.
- `backend/internal/handlers/...` là gameplay HTTP; `backend/internal/db/schema_*.go` là persistence; `backend/data/*.csv` là dataset được quản trị qua Admin.
- Nhiều nội dung và giới hạn phụ thuộc DB/config runtime. Trang feature ghi rõ khi một giá trị do dataset quyết định thay vì hard-code.
- Công thức trong mục “Công thức” được chép theo backend hiện tại; xem phần source liên quan trong cùng trang. Phân biệt constants, CSV/config và giá trị trả về động; giữ nguyên đơn vị, clamp và thứ tự áp dụng khi cập nhật.
- Tên feature trên lobby là nhãn UI, không đảm bảo action mở được trong mọi rank/trạng thái.

## Cập nhật đồng bộ

Khi sửa route, rule, điều kiện mở, trạng thái UI, schema, worker hoặc dataset, cập nhật trang feature bị ảnh hưởng trong cùng thay đổi. Nếu docs mô tả behavior mới, code phải implement behavior đó; nếu chỉ mô tả rõ behavior đang có thì kiểm tra source và cập nhật docs. Thay đổi nội dung hướng dẫn người chơi cần đối chiếu [WIKI.md](../../WIKI.md).
