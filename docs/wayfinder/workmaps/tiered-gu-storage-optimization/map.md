# Wayfinder Map: Kiến Trúc Phân Tầng Lưu Trữ Cổ Trùng & Đàn Cổ Tiên Khiếu (Tiered Gu Storage & Aperture Gu Swarms)

## Destination

Thiết kế kiến trúc phân tầng Cổ Trùng (Tiered Gu Storage & Scaling Architecture) giải quyết triệt để vấn đề bùng nổ dữ liệu khi người chơi sở hữu hàng vạn Phàm Cổ ở giai đoạn Cổ Tiên (Rank 6+), trong khi bảo toàn 100% trải nghiệm chăm sóc, nuôi dưỡng và dữ liệu cá thể của Cổ Sư Phàm Nhân (Rank 1–5):
- Phàm Nhân (Rank 1–5): Giữ nguyên bảng `player_gu` lưu cá thể (Instance: hunger, durability, subjugation, state). Áp đặt giới hạn Sức Chứa Không Khiếu Phàm Nhân (Mortal Aperture Capacity: 10–80 con) để vừa sát nguyên tác vừa chặn đứng nguy cơ phình DB trước khi Thăng Tiên.
- Cổ Tiên (Rank 6+): Tích hợp hệ thống "Đàn Phàm Cổ Tiên Khiếu (Immortal Aperture Gu Swarms)" lưu dạng Xếp chồng (Stackable/Fungible: `aperture_id, gu_id, rank, quantity`) trong Tiên Khiếu sinh thái; hệ sinh thái Tiên Khiếu tự động nuôi dưỡng không cần worker quét đói; cho phép Rút (Materialize) thành cá thể khi trang bị và Nhập đàn (Dematerialize) khi tháo ra; hỗ trợ Hợp Luyện & Sát Chiêu tiêu thụ hàng loạt (Batch consumption) trực tiếp từ đàn.
- Tiên Cổ (Rank 6–9) & Cổ Đang Trang Bị (Max 60 con): Luôn là cá thể duy nhất trong `player_gu`.

## Notes

- Domain: Tiered Gu Storage, Aperture Gu Swarms, Gu Materialization/Dematerialization, Mortal Aperture Capacity, Batch Refining & Killer Move Consumption.
- Skills to consult: `domain-modeling`, `codebase-design`, `prototype`, `research`.
- Standing preferences:
  - Tuân thủ nghiêm ngặt `AGENTS.md`: Gameplay logic giữ tại `backend/internal/handlers/`, DDL tại `backend/internal/db/schema_*.go`. Không tự ý tạo thêm tầng repository/service trừu tượng.
  - Tương thích ngược tuyệt đối (Zero Breaking Changes for Mortals): Không làm xáo trộn luồng tân thủ, code nuôi Cổ, giao dịch, hợp luyện của Phàm nhân Rank 1–5.
  - Phân tách rõ ràng giữa Phàm Cổ (Rank 1–5) và Tiên Cổ (Rank 6–9): Tiên Cổ tuyệt đối không bao giờ được phép stack; luôn tuân thủ ràng buộc duy nhất toàn server `uq_player_gu_immortal`.
  - Triệt tiêu gánh nặng I/O: Loại bỏ vòng lặp quét đói `EvaluatePlayerHungerTx` đối với Phàm Cổ trong Đàn Cổ Tiên Khiếu.

## Decisions so far

<!-- the index: one line per closed ticket, enough to judge relevance, then zoom the link for the detail the ticket holds -->

- [[01-schema-and-domain-model-mortal-gu-swarms](issues/01-schema-and-domain-model-mortal-gu-swarms.md)]: Chuẩn hóa DDL bảng `aperture_gu_swarms` gắn liền với `immortal_apertures(id) ON DELETE CASCADE`, ràng buộc chặt `gu_rank BETWEEN 1 AND 5` và `quantity >= 0`, cách ly hoàn toàn khỏi snapshot hotpath với 0ms overhead.
- [[02-materialization-and-dematerialization-lifecycle](issues/02-materialization-and-dematerialization-lifecycle.md)]: Chuẩn hóa quy trình 2 chiều Rút Cổ (Materialize ➔ Equip) và Nhập Đàn (Dematerialize ➔ Delete row), chốt chặn Cổ bị thương < 100% độ bền / Tiên Cổ / Bản Mệnh, và cơ chế thu hồi nhanh 1 chạm; kiểm chứng hoàn tất qua prototype demo HTML.
- [[03-batch-consumption-refining-and-killer-moves](issues/03-batch-consumption-refining-and-killer-moves.md)]: Chuẩn hóa cơ chế ký quỹ và khấu trừ hàng loạt Phàm Cổ từ Đàn Tiên Khiếu qua cột `swarms_deducted` trong `refine_sessions`, khấu trừ đạn Cổ O(1) cho Sát Chiêu, và logic hoàn trả 50% khi hủy phiên.
- [[04-aperture-capacity-and-mortal-safety-limits](issues/04-aperture-capacity-and-mortal-safety-limits.md)]: Khóa bảng định mức sức chứa Không Khiếu Phàm Nhân (Rank 1: 15 ➔ Rank 5: 80–100 con), quy chuẩn hàm kiểm tra dung lượng theo delta (cho phép hợp luyện khi đầy túi) và dỡ bỏ giới hạn khi Thăng Tiên; kiểm chứng hoàn tất qua prototype demo HTML.
- [[05-frontend-aperture-gu-swarm-ui-and-ux](issues/05-frontend-aperture-gu-swarm-ui-and-ux.md)]: Chuẩn hóa giao diện Quản lý Đàn Cổ Tiên Khiếu trên Client: Thẻ Đàn Cổ kèm badge số lượng lớn, bộ lọc Lưu phái & Rank, nút Rút ➔ Trang bị 1-chạm, và nút thu hồi toàn bộ Phàm Cổ lành lặn; kiểm chứng qua prototype demo HTML.





## Not yet specified

<!-- Fog of War: Những khía cạnh trong phạm vi nhưng cần chờ các quyết định trên Frontier làm rõ trước khi tạo ticket -->

- **Aperture Ecosystem Gu Breeding (Tiên Khiếu Tự Động Sinh Sản Phàm Cổ)**: Cơ chế tài nguyên sinh thái (Resource Nodes / Rừng Trúc, Đầm Lầy) trong Tiên Khiếu tự động sinh sản thêm Phàm Cổ theo thời gian sông Quang Âm trôi qua.
- **Gu Swarm Mortality in Calamities & Tribulations (Tổn Thất Đàn Cổ Khi Thiên Kiếp)**: Cơ chế tính toán tỷ lệ hao hụt % Đàn Phàm Cổ khi Tiên Khiếu gánh chịu Địa Tai / Thiên Kiếp hoặc chiến tranh xâm lược giữa các Cổ Tiên.
- **Connecting Heaven Batch Trade (Giao Dịch Sỉ Đàn Cổ Qua Bảo Hoàng Thiên)**: Hỗ trợ giao dịch sỉ hàng ngàn/hàng vạn Phàm Cổ giữa các Cổ Tiên qua Bảo Hoàng Thiên với mức phí Tiên Nguyên Thạch tương xứng.

## Out of scope

<!-- Công việc nằm ngoài phạm vi của nỗ lực này -->

- **Immortal Gu Stacking (Xếp Chồng Tiên Cổ)**: Tiên Cổ (Rank 6–9) là "Thiên Địa duy nhất", bắt buộc giữ nguyên 100% dạng cá thể độc bản, không thể stack.
- **Equipped Gu Stacking (Xếp Chồng Cổ Đang Trang Bị)**: Cổ đã trang bị vào Không Khiếu/Thanh kỹ năng (tối đa 60 con) bắt buộc là cá thể để tính toán độ bền trong giao tranh, cooldown và slot trang bị.
- **Overhauling Mortal Gu Survival Mechanics**: Không thay đổi cơ chế độ đói, độ bền, cho ăn cá thể đối với Cổ Sư Phàm Nhân (Rank 1–5).
