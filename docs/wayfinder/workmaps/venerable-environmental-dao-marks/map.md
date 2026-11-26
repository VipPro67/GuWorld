# Wayfinder Map: Đạo Ngân Môi Trường & Tôn Giả Luyện Hóa Đạo Ngân

## Destination

Bản đặc tả thiết kế gameplay & kiến trúc hoàn chỉnh (ready-to-implement spec) cho hệ thống **Đạo Ngân Môi Trường** (tác động hai chiều lên hiệu suất chiêu thức) và cơ chế **Tôn Giả Luyện Hóa Đạo Ngân Môi Trường** (luyện hóa, sở hữu Đạo Vực, tranh đoạt giữa các Cửu Chuyển Cổ Tôn và tương tác Tiên Khiếu), bám sát cốt truyện Cổ Chân Nhân và kiến trúc hiện tại của GuWorld.

## Notes

- Domain: World/Regions/Map Cells (`backend/internal/db/schema_world.go`, `models/region.go`, `handlers/region_policy.go`), Combat Simulator & Dao Profile (`backend/internal/gamefx/dao_combat_profile.go`, `backend/internal/simulator/`), Immortal Aperture (`backend/internal/db/schema_aperture.go`, `handlers/aperture_*.go`), Rank 9 / Tôn Giả (`models/rank.go`, `handlers/tribulation_multiform.go`).
- Decisions from initial charting:
  - Thiết kế thống nhất **2 lớp gắn kết**: Lớp nền (**Đạo Ngân Môi Trường** áp dụng cho mọi người chơi khi giao chiến) + Lớp đỉnh cao (**Luyện Hóa Đạo Ngân Tự Nhiên** mở khóa cho Cửu Chuyển Cổ Tôn).
  - Môi trường ngoài thế giới dùng mô hình **2 tầng kế thừa** (`Region` làm nền + `Map Cell` / điểm nóng ghi đè); giao chiến bên trong **Tiên Khiếu** dùng Đạo Ngân của chính Tiên Khiếu đó.
  - Đạo Ngân Môi Trường tác động **hai chiều** lên hiệu suất chiêu: cả hiệu lực chiêu (Sát thương / Khiên / Trị liệu) và độ ổn định/tiêu hao (Tỉ lệ thành công Sát Chiêu / Tiêu hao Chân Nguyên-Tiên Nguyên), có ngưỡng sàn/trần cân bằng.
  - Luyện hóa của Tôn Giả là **tiến trình tích lũy** (tiêu hao Tiên Nguyên/thời gian, phụ thuộc sức kháng cự vùng), mang lại lợi ích chiến đấu + vận hành tại chỗ, và cho phép Tôn Giả khác **luyện hóa đè / tranh đoạt**.
  - Khi **Hạ Khiếu** ra vùng có Tôn Giả kiểm soát, Tiên Khiếu chịu tương tác (cộng hưởng hoặc áp chế) quanh khu vực đặt khiếu.
- Skills to consult: `grilling`, `domain-modeling`, `prototype`, `research`.
- Canonical tracker: [Wayfinder: Đạo Ngân Môi Trường & Tôn Giả Luyện Hóa Đạo Ngân](https://github.com/VipPro67/GuWorld/issues/460). Local context copies live in `docs/wayfinder/workmaps/venerable-environmental-dao-marks/`.

## Decisions so far

<!-- one line per closed ticket -->
- [Khảo sát kiến trúc DaoCombatProfile, Simulator, Region và Hạ Khiếu hiện tại](https://github.com/VipPro67/GuWorld/issues/461): Khảo sát điểm neo kỹ thuật (`DaoCombatProfile`, `Simulator`, `regions`/`map_cells`/`worldfeed`, Hạ Khiếu & Tiên Khiếu công cộng) — [461-codebase-anchors.md](https://github.com/VipPro67/GuWorld/blob/research/461-codebase-anchors/docs/wayfinder/workmaps/venerable-environmental-dao-marks/research/461-codebase-anchors.md)
- [Chốt mô hình dữ liệu Đạo Ngân Môi Trường (Region, Map Cell, Tiên Khiếu)](https://github.com/VipPro67/GuWorld/issues/462): Kế thừa cộng dồn `regions.natural_dao_marks` + `map_cells.dao_marks`, bảng riêng `region_dao_dominions` cho Đạo Vực Tôn Giả (cấp `Region`, trừ `HUB`/`SAFE_ZONE`), và dùng `immortal_apertures.dao_marks` khi chiến đấu trong Tiên Khiếu.
- [Chốt công thức & quy tắc Đạo Ngân Môi Trường ảnh hưởng hiệu suất chiêu thức](https://github.com/VipPro67/GuWorld/issues/463): Cấu hình hóa các mốc trong `gamefx`: Đạo Ngân Tự Nhiên `[0.75x, 1.35x]` (bão hòa `M/(M+K)`), Đạo Vực `[0.60x, 1.60x]` theo chủ/đồng minh/địch, phạt tiêu hao & thành công khi `< 0.90x`, bảo vệ Rank 1–5, `HasNonConflictingDao` miễn tự nhiên nhưng vẫn chịu Đạo Vực địch.
- [Chốt vòng lặp Luyện Hóa Đạo Ngân và cơ chế Tranh Đoạt của Tôn Giả](https://github.com/VipPro67/GuWorld/issues/464): Rank 9 đứng tại `Region` + Level 4 lưu phái; bơm tiến độ theo lần (Tiên Nguyên + cooldown) đẩy `control_pct` `0%→100%`; giới hạn số Đạo Vực + decay; tranh đoạt 2 pha (bào mòn về `0%` có cảnh báo → chiếm mới); hồi Tiên Nguyên/Khí + cảm ứng xâm nhập.
- [Chốt tương tác giữa Đạo Vực Tôn Giả và Tiên Khiếu (Hạ Khiếu & Tiên Khiếu công cộng)](https://github.com/VipPro67/GuWorld/issues/465): Nhân hệ số Đạo Vực vào Thiên Địa Nhị Khí lúc Thu Khiếu (`+50%`/`+30%`/`-35%` sàn `0.65x`, configurable); ranh giới chuyển môi trường tại Môn Hộ; miễn trừ `HUB`/`SAFE_ZONE` và Tiên Khiếu công cộng; Đạo Vực đóng vai trò cứ điểm an toàn & bá chủ cho chủ nhân (tăng mốc áp chế mặc định trong `EnvDaoConfig`).
- [Thiết kế UI/UX & luồng thông báo Đạo Ngân Môi Trường, Đạo Vực và Luyện Hóa](https://github.com/VipPro67/GuWorld/issues/466): Chốt prototype 3 luồng (Bản đồ/Khu vực, Combat HUD banner, Bảng Tôn Giả + `world:change`) và chuyển sang triển khai thực tế.

## Not yet specified

- Cân bằng bảng thông số kinh tế chi tiết (chi phí Tiên Nguyên, thời lượng cooldown, tốc độ decay) và bộ dữ liệu Đạo Ngân tự nhiên ban đầu cho Ngũ Vực và các khu vực hiện có.

## Out of scope

- Viết code triển khai trực tiếp (backend Go / frontend Next.js / DB migration) trong phạm vi bản đồ Wayfinder này.
- Thay đổi điều kiện đột phá lên Cửu Chuyển (Rank 9) hiện có.
