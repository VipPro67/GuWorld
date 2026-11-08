# Kế hoạch sắp xếp tài liệu và thư mục Markdown

## Mục tiêu

- Nhìn tên thư mục là biết tài liệu dùng cho ai và dùng vào việc gì.
- Có một điểm vào ngắn gọn để tìm tài liệu chuẩn đang mô tả hệ thống.
- Tách tài liệu sử dụng lâu dài khỏi nghiên cứu, kế hoạch và gói công việc tạm.
- Giảm rủi ro khi di chuyển: không làm mất lịch sử, link, hay ngữ cảnh công việc đang chạy.

## Trạng thái triển khai

- **Đã làm:** tạo `docs/README.md`, `docs/guides/README.md`, `docs/architecture/README.md`.
- **Đã chuyển:** API, development, game data, production deploy và ba tài liệu Admin vào `docs/guides/`; architecture overview và Player Domain guide vào `docs/architecture/`.
- **Đã cập nhật:** điểm vào trong root README, hướng dẫn agent/project và các tham chiếu đường dẫn trong tài liệu/code.
- **Đã dọn thư mục ẩn:** chuyển work map/ticket từ `.scratch/` sang `docs/wayfinder/workmaps/`; chuyển sơ đồ HTML Archify sang `docs/architecture/diagrams/` và xóa JSON receipt có thể tái tạo.
- **Còn lại:** phân nhóm tài liệu gameplay/design cấp trực tiếp; thêm mục lục cho MMO/Wayfinder; gắn trạng thái cho plan/research.

## Hiện trạng

- Root có các điểm vào và trạng thái (`README.md`, `PROJECT.md`, `CONTEXT.md`, `WIKI.md`) nhưng cũng có kế hoạch/tiến độ dài (`OPTIMIZATION_PLAN.md`, `OPTIMIZATION_PROGRESS.md`, `GuWorld_MMO_Realtime_Phaser4_Master_Plan.md`).
- `docs/` chứa hướng dẫn vận hành và API, tài liệu gameplay/thiết kế, nghiên cứu, roadmap, ADR, và các nhóm tài liệu Wayfinder/MMO. Có nhiều file cùng cấp, tên file chưa thống nhất hoa thường/ngôn ngữ.
- `docs/wayfinder/workmaps/` giữ lại ba bộ map/ticket trước đây nằm trong `.scratch/`, cùng prototype HTML; đây là hồ sơ thiết kế, không phải contract hiện hành.
- Sơ đồ kiến trúc sinh bởi Archify được lưu ở `docs/architecture/diagrams/`; các JSON receipt có thể tái tạo đã xóa.
- Các tài liệu được liên kết trực tiếp từ README, AGENTS, WIKI và nhiều tài liệu khác. Việc đổi đường dẫn hàng loạt có thể làm hỏng liên kết và các quy trình hiện tại.

## Quy ước đích

Giữ tên `docs/` làm kho tài liệu chính (không đổi thành `.docs`; thư mục ẩn sẽ khó khám phá và không có lợi cho tài liệu cần đọc thường xuyên). Dùng cấu trúc theo mục đích:

```text
/
├── README.md                 # giới thiệu ngắn + lối vào
├── PROJECT.md                # trạng thái và phạm vi hiện tại
├── CONTEXT.md                # ngữ cảnh miền cho agent/người phát triển
├── WIKI.md                   # wiki gameplay
├── docs/
│   ├── README.md             # mục lục tài liệu, phân loại và quy tắc cập nhật
│   ├── guides/               # phát triển, API, dữ liệu, triển khai/vận hành
│   ├── architecture/         # kiến trúc hiện hành và hợp đồng runtime
│   ├── gameplay/             # luật chơi, hệ thống, cân bằng và thiết kế gameplay
│   ├── plans/                # kế hoạch được duyệt/đang hoạt động, có chủ sở hữu và trạng thái
│   ├── research/             # khảo sát, audit, phân tích; không mặc định là hành vi hiện hành
│   ├── adr/                  # quyết định kiến trúc đã chốt, đánh số và bất biến
│   ├── mmo/                  # tài liệu MMO theo miền; giữ nhóm nếu vẫn là một bộ tài liệu
│   ├── wayfinder/            # hồ sơ initiative Wayfinder (spec, ticket, research, notes)
│   ├── wayfinder/workmaps/   # hồ sơ work map/ticket, không phải contract hiện hành
│   ├── architecture/diagrams/# sơ đồ kiến trúc được giữ lại
│   └── archive/              # tài liệu đã hết hiệu lực nhưng cần giữ lịch sử
```

`docs/mmo/` và `docs/wayfinder/` là nhóm miền được giữ nguyên trong giai đoạn đầu; chỉ chuyển các tài liệu khi có người sở hữu và biết rõ điểm vào/link cần cập nhật. Không tạo thêm tầng thư mục chỉ để phân loại vài file.

## Phân loại và chuyển đổi đề xuất

| Hiện tại | Đích/việc cần làm | Điều kiện |
| --- | --- | --- |
| `README.md`, `PROJECT.md`, `CONTEXT.md`, `WIKI.md` | Giữ ở root; README liên kết sang `docs/README.md` | Không đổi vai trò; nội dung dài chuyên biệt chuyển theo từng đợt sau |
| `docs/guides/API.md`, `DEVELOPMENT.md`, `GAME_DATA.md`, `PRODUCTION_DEPLOY.md`, `ADMIN_*` | `docs/guides/` | Cập nhật link và bảng tài liệu README trong cùng thay đổi |
| `docs/architecture/ARCHITECTURE.md`, `WEBSOCKET_PLAN.md`, `WORLDMAP_REALTIME.md`, các contract/protocol hiện hành | `docs/architecture/` | Xác nhận tài liệu còn hiệu lực; phân biệt contract hiện hành với ý tưởng tương lai |
| `docs/*_SYSTEM.md`, tài liệu combat/tribulation/sect/companion/capture | `docs/gameplay/` | Gom theo hệ thống; tránh đổi tên nội dung nếu chỉ cần chuyển thư mục |
| `docs/OPTIMIZATION_*`, `docs/plans/*`, các plan ở root | `docs/plans/` | Gắn trạng thái (proposed/active/done/superseded), chủ sở hữu hoặc ngày rà soát |
| `docs/research/*`, `docs/architecture_review/*` | Giữ ở `docs/research/` hoặc đưa báo cáo review vào `docs/archive/` | Thêm dòng trạng thái/ngày; không dùng báo cáo cũ làm mô tả hiện hành |
| `docs/adr/*` | Giữ nguyên | ADR đã chấp nhận không sửa nội dung; quyết định mới thêm số tiếp theo |
| `docs/mmo/*`, `docs/wayfinder/*` | Giữ nguyên trong đợt đầu | Thêm README mục lục ở cấp nhóm; xử lý dọn trùng lặp sau kiểm kê riêng |
| `.scratch/<initiative>/` | Đã chuyển nội dung hiện có sang `docs/wayfinder/workmaps/` và bỏ thư mục ẩn | Map, ticket và prototype được giữ; không suy ra implementation đã hoàn tất |
| `.archify/<run>/` | Đã chuyển HTML sơ đồ vào `docs/architecture/diagrams/` và xóa JSON receipt tái tạo được | Giữ đầu ra có giá trị tham khảo, không giữ trạng thái công cụ cục bộ |
| `GuWorld_MMO_Realtime_Phaser4_Master_Plan.md` và các MD dài ở root | Phân loại vào `docs/plans/` hoặc `docs/archive/` | Xác minh quyền sở hữu, mức độ còn hiệu lực và link trước khi di chuyển; không xóa |

## Các bước thực hiện

### 1. Lập danh mục và gắn trạng thái

- Liệt kê mọi Markdown, phân loại: current/reference, guide, gameplay, plan, research, ADR, initiative, temporary.
- Ghi chủ sở hữu hoặc người xác nhận, ngày rà soát, trạng thái; đánh dấu tài liệu trùng lặp hoặc trái nhau.
- Xác định tài liệu được tạo tự động và tài liệu cần giữ vì lịch sử.

**Hoàn tất khi:** mỗi file có một nhóm đích và không có tài liệu chưa rõ trạng thái bị chuyển/xóa.

### 2. Tạo lối vào dễ đọc

- Thêm `docs/README.md` làm mục lục ngắn, mô tả người đọc và mục đích từng nhóm.
- Rút gọn mục “Tài liệu đang dùng” trong root README để phân biệt tài liệu hiện hành với kế hoạch/tham khảo.
- Thêm README riêng cho nhóm work map nếu chuyển nội dung từ scratch; phân biệt hồ sơ thiết kế với contract hiện hành.

**Hoàn tất khi:** người mới tìm được cách chạy, API, kiến trúc, dữ liệu, gameplay và kế hoạch từ tối đa hai lần nhấp.

### 3. Di chuyển theo nhóm nhỏ

- Tạo các thư mục đích cần thiết; ưu tiên `git mv` theo nhóm có cùng mục đích.
- Mỗi đợt chỉ chuyển một nhóm; cập nhật mọi tham chiếu Markdown, AGENTS, README và đường dẫn trong mã/config.
- Tránh đổi tên file cùng lúc với đổi thư mục; chuẩn hóa tên file thành chữ thường/kebab-case chỉ khi có thể cập nhật toàn bộ link.
- Không thay đổi tài liệu thuộc thay đổi gameplay/feature đang dở dang nếu chưa tách được khỏi công việc đó.

### 4. Xử lý kế hoạch và tài liệu hết hiệu lực

- Mỗi plan có trạng thái và ngày cập nhật; kế hoạch hoàn tất được tóm tắt kết quả rồi lưu ở `docs/archive/plans/` hoặc giữ trong lịch sử issue.
- Tài liệu thay thế tài liệu khác phải ghi “superseded by” cùng link; không xóa bản cũ ngay.
- Hồ sơ trong `docs/wayfinder/workmaps/` cần đối chiếu tracker/code trước khi tiếp tục; chúng được chuyển từ local work maps.

### 5. Hoàn tất và giữ cấu trúc ổn định

- Tìm các link Markdown tương đối, đường dẫn `docs/...` trong source/CI/scripts và các chỉ dẫn agent.
- Kiểm tra đường dẫn đích tồn tại, không còn link cũ, không có tài liệu bị mất; xem diff để chắc không sửa nội dung ngoài phạm vi.
- Thêm quy tắc ngắn vào `AGENTS.md`: tài liệu chuẩn ở `docs/`, work maps có trạng thái riêng, ADR bất biến, cập nhật link cùng lúc di chuyển.
- Không chạy formatter hàng loạt cho toàn bộ Markdown trong đợt di chuyển.

## Tiêu chí hoàn thành

- Có `docs/README.md` làm mục lục và README root dẫn tới nó.
- Tên nhóm thể hiện mục đích; không có thư mục ẩn mới để chứa tài liệu đọc thường xuyên.
- Các file được chuyển có trạng thái/đích rõ, link nội bộ cập nhật và không còn tham chiếu đường dẫn cũ.
- Không còn `.scratch/` hoặc `.archify/`; các work map và sơ đồ cần giữ có vị trí hiển thị trong `docs/`.
- Không mất lịch sử; file cũ được lưu trữ hoặc giữ nguyên cho tới khi xác nhận không còn giá trị.

## Phạm vi không làm trong kế hoạch này

- Không đổi nội dung gameplay, API, schema hoặc hợp đồng runtime.
- Không xóa hồ sơ work map/sơ đồ chỉ vì chúng được tạo trong công cụ hoặc thư mục tạm.
- Không di chuyển tài liệu của initiative đang triển khai trước khi có điểm bàn giao.
- Không coi số lượng thư mục ít hơn là mục tiêu; ưu tiên tìm nhanh và sở hữu rõ ràng.

## Kết quả quét Markdown

Đợt quét ban đầu ghi nhận **228 file `.md`** trong repo, gồm thư mục ẩn; bỏ qua `.git/`, `node_modules/` và `vendor/`. Đã thêm bốn README mục lục (docs, guides, architecture, workmaps), nên tổng hiện tại là **232 file**. Việc di chuyển không đổi tổng; JSON receipt Archify đã xóa không phải Markdown. Đây là kiểm kê theo đường dẫn và tiêu đề/nội dung mở đầu, không phải đánh giá từng tuyên bố kỹ thuật là còn đúng với code hiện tại.

### Phân bố

| Khu vực | Số file | Nội dung chính / nhận xét |
| --- | ---: | --- |
| `.agents/skills/` | 59 | Hướng dẫn skill cho agent và các mẫu/format đi kèm. Đây là tài liệu cấu hình quy trình; giữ cạnh skill, không chuyển vào `docs/` sản phẩm. Có các bộ hướng dẫn issue tracker GitHub/GitLab/local nên cần giữ rõ lựa chọn đang áp dụng. |
| `docs/wayfinder/workmaps/` | 20 | Ba initiative: tối ưu `FOR UPDATE` (map + 6 issue), mở rộng Tiên Khiếu (map + 5 issue), Tiên Cổ Phòng (map + 5 issue), cộng README nhóm. Có thêm 5 prototype HTML; cần đối chiếu code/tracker trước khi coi là công việc hiện hành. |
| Root | 8 | `AGENTS.md`, `README.md`, `PROJECT.md`, `CONTEXT.md`, `WIKI.md` là điểm vào/ngữ cảnh; `OPTIMIZATION_PLAN.md`, `OPTIMIZATION_PROGRESS.md`, `GuWorld_MMO_Realtime_Phaser4_Master_Plan.md` là plan/log dài, cần gắn trạng thái và chủ sở hữu trước khi chuyển. |
| `backend/cmd/wtspike/` | 1 | README cho spike/prototype WebTransport; để cạnh prototype/backend command. |
| `docs/` trực tiếp | 30 | Nhiều spec/gameplay vẫn cùng cấp; tiếp tục phân loại theo các đợt riêng sau khi kiểm tra trạng thái và link. |
| `docs/guides/` | 8 | Bảy tài liệu phát triển/vận hành đã chuyển trong đợt đầu, cộng README nhóm. |
| `docs/architecture/` | 3 | Architecture overview, Player Domain guide và README mục lục của nhóm. |
| `docs/mmo/` | 26 | Bộ tài liệu MMO: audit/baseline, kiến trúc/protocol, gameplay và vận hành như benchmark, rollout, recovery, scaling, SLO, security, assets, loot, party. Nên thêm mục lục và trạng thái từng tài liệu; tránh tách tiếp nếu chưa cần. |
| `docs/plans/` | 6 | Kế hoạch nhánh realtime, viết lại effect/killer moves, nguồn vị trí player, whitelist spawn, parity combat và kế hoạch sắp xếp tài liệu này. Cần trạng thái nhất quán. |
| `docs/research/` | 17 | Audit/khảo sát API/UI, player state, MMO, spatial realm, Dao, economy, assets, transformation. Phần lớn là snapshot theo thời điểm; cần ngày khảo sát và trạng thái current/superseded. |
| `docs/wayfinder/` | 70 | Hồ sơ initiative lớn: 5 tài liệu map/progress, 21 ticket, 17 spec, 6 research, 1 note và 19 Markdown trong work maps. README nhóm work maps giúp điều hướng. |
| `docs/adr/`, `docs/agents/`, `docs/specs/` | 5 | Một ADR, ba hướng dẫn agent (domain/tracker/labels), một spec pricing economy. Các nhóm này đã có ý nghĩa riêng; không trộn ADR với plan/research. |
| `frontend/` | 3 | `README.md`, `AGENTS.md`, `CLAUDE.md`; hướng dẫn framework và quy tắc frontend nên ở gần frontend. `CLAUDE.md` hiện chuyển tiếp sang AGENTS. |
| `frontend/public/assets/` | 2 | README mô tả bộ icon item và art world; giữ cạnh tài sản để ghi nguồn/phạm vi sử dụng. |
| `scripts/` | 1 | README mô tả script quản lý dữ liệu game; giữ cạnh script. |

### Bản đồ nội dung `docs/` cấp trực tiếp

- **Vận hành/phát triển:** `API.md`, `DEVELOPMENT.md`, `PRODUCTION_DEPLOY.md`, `ADMIN_OPERATIONS_RUNBOOK.md`, `ADMIN_RESOURCE_MATRIX.md`, `ADMIN_RBAC_TEST_MATRIX.md`, `GAME_DATA.md`.
- **Kiến trúc/realtime:** `ARCHITECTURE.md`, `WEBSOCKET_PLAN.md`, `WORLDMAP_REALTIME.md`, `REALTIME_TRANSFORMATION_PROTOCOL_SPEC.md`, `PLAYER_DOMAIN.md`.
- **Gameplay/design:** `CANH_GIOI_DAO_NGAN.md`, `DaoMarks_Attainment_Design.md`, `dao_mark_conflict_design.md`, `he-thong-di-nhan-tien-khieu.md`, `DESIGN_TIEN_KHAU.md`, `GU_HOUSE_INTEGRATION_SPEC.md`, `COMBAT_PROFILES_TEMPLATES.md`, `GuRankScaling.md`, `KILLER_MOVE_TRANSFORMATION_SPEC.md`, `MONSTER_CAPTURE_DESIGN.md`, `COMPANION_CASUALTY_SPEC.md`, `STATUS_EFFECTS.md`, `TRIBULATION_BALANCE.md`.
- **Kế hoạch/audit tối ưu:** `OPTIMIZATION_ROADMAP_2026-09.md`, `PLAYER_PERSISTENCE_OPTIMIZATION_PLAN.md`, `REQUEST_PERFORMANCE_OPTIMIZATION_GUIDE.md`, `UI_REFACTOR.md`.

### Điểm cần xử lý trong đợt phân loại

- Có tài liệu cùng chủ đề cần xác định quan hệ canonical/tham khảo: `CANH_GIOI_DAO_NGAN.md` và `DaoMarks_Attainment_Design.md`; nhiều thiết kế Dao Mark ở root `docs/`, `docs/research/` và `docs/wayfinder/`.
- Tài liệu thiết kế, kế hoạch và hành vi đã triển khai đang đứng cạnh nhau; tiêu đề “spec” chưa đảm bảo đây là contract đang chạy. Cần thêm trạng thái, không tự kết luận chỉ từ tên file.
- `OPTIMIZATION_PROGRESS.md` tự ghi đây là log lịch sử; nên liên kết tới tài liệu hiện hành thay vì xem như bảng trạng thái live.
- `.agents/skills/setup-matt-pocock-skills/` có hướng dẫn tracker cho nhiều nền tảng; xác nhận nền tảng đang dùng trước khi dọn (repo hướng dẫn hiện ghi GitHub Issues).
- `.archify/` không có Markdown; sơ đồ HTML đã chuyển vào `docs/architecture/diagrams/`, JSON receipt tái tạo được đã xóa.

### Phạm vi kiểm kê theo nhóm

Danh mục ban đầu bao quát 228 file Markdown theo thư mục và mục đích; hiện có thêm bốn README mục lục. Các nhóm 21 ticket và 17 spec Wayfinder, 59 skill agent, cùng các work map đã chuyển được kiểm tra tiêu đề và phần mở đầu để phân nhóm; nội dung kỹ thuật bên trong từng ticket/skill không được diễn giải lại ở đây vì chúng là hồ sơ đơn vị riêng, không phải một bộ tài liệu dự án thống nhất.
