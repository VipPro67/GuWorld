# Kế hoạch sắp xếp tài liệu và thư mục Markdown

## Mục tiêu

- Nhìn tên thư mục là biết tài liệu dùng cho ai và dùng vào việc gì.
- Có một điểm vào ngắn gọn để tìm tài liệu chuẩn đang mô tả hệ thống.
- Tách tài liệu sử dụng lâu dài khỏi nghiên cứu, kế hoạch và gói công việc tạm.
- Giảm rủi ro khi di chuyển: không làm mất lịch sử, link, hay ngữ cảnh công việc đang chạy.

## Hiện trạng

- Root có các điểm vào và trạng thái (`README.md`, `PROJECT.md`, `CONTEXT.md`, `WIKI.md`) nhưng cũng có kế hoạch/tiến độ dài (`OPTIMIZATION_PLAN.md`, `OPTIMIZATION_PROGRESS.md`, `GuWorld_MMO_Realtime_Phaser4_Master_Plan.md`).
- `docs/` chứa hướng dẫn vận hành và API, tài liệu gameplay/thiết kế, nghiên cứu, roadmap, ADR, và các nhóm tài liệu Wayfinder/MMO. Có nhiều file cùng cấp, tên file chưa thống nhất hoa thường/ngôn ngữ.
- `.scratch/` chứa các thư mục theo initiative, mỗi thư mục có `map.md` và `issues/`. Đây là hồ sơ công việc đang triển khai, không phải tài liệu chuẩn cho người đọc codebase.
- `.archify/` chứa JSON trạng thái/gói bàn giao và HTML kiến trúc được sinh bởi công cụ; nội dung này không phải nguồn tài liệu dự án chính.
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
│   └── archive/              # tài liệu đã hết hiệu lực nhưng cần giữ lịch sử
├── .scratch/                 # work-in-progress chưa phải tài liệu chuẩn
└── .archify/                 # đầu ra cục bộ của Archify, có thể tái tạo
```

`docs/mmo/` và `docs/wayfinder/` là nhóm miền được giữ nguyên trong giai đoạn đầu; chỉ chuyển các tài liệu khi có người sở hữu và biết rõ điểm vào/link cần cập nhật. Không tạo thêm tầng thư mục chỉ để phân loại vài file.

## Phân loại và chuyển đổi đề xuất

| Hiện tại | Đích/việc cần làm | Điều kiện |
| --- | --- | --- |
| `README.md`, `PROJECT.md`, `CONTEXT.md`, `WIKI.md` | Giữ ở root; README liên kết sang `docs/README.md` | Không đổi vai trò; nội dung dài chuyên biệt chuyển theo từng đợt sau |
| `docs/API.md`, `DEVELOPMENT.md`, `GAME_DATA.md`, `PRODUCTION_DEPLOY.md`, `ADMIN_*` | `docs/guides/` | Cập nhật link và bảng tài liệu README trong cùng thay đổi |
| `docs/ARCHITECTURE.md`, `WEBSOCKET_PLAN.md`, `WORLDMAP_REALTIME.md`, các contract/protocol hiện hành | `docs/architecture/` | Xác nhận tài liệu còn hiệu lực; phân biệt contract hiện hành với ý tưởng tương lai |
| `docs/*_SYSTEM.md`, tài liệu combat/tribulation/sect/companion/capture | `docs/gameplay/` | Gom theo hệ thống; tránh đổi tên nội dung nếu chỉ cần chuyển thư mục |
| `docs/OPTIMIZATION_*`, `docs/plans/*`, các plan ở root | `docs/plans/` | Gắn trạng thái (proposed/active/done/superseded), chủ sở hữu hoặc ngày rà soát |
| `docs/research/*`, `docs/architecture_review/*` | Giữ ở `docs/research/` hoặc đưa báo cáo review vào `docs/archive/` | Thêm dòng trạng thái/ngày; không dùng báo cáo cũ làm mô tả hiện hành |
| `docs/adr/*` | Giữ nguyên | ADR đã chấp nhận không sửa nội dung; quyết định mới thêm số tiếp theo |
| `docs/mmo/*`, `docs/wayfinder/*` | Giữ nguyên trong đợt đầu | Thêm README mục lục ở cấp nhóm; xử lý dọn trùng lặp sau kiểm kê riêng |
| `.scratch/<initiative>/` | Giữ tại chỗ khi đang hoạt động; sau khi hoàn tất chuyển phần có giá trị sang `docs/plans/` hoặc `docs/archive/` | Chỉ đóng gói khi issue/initiative hoàn tất; `map.md` phải trỏ tới hồ sơ đích |
| `.archify/<run>/` | Giữ là artifact cục bộ; cân nhắc ignore theo cấu hình công cụ | Chỉ xuất bản HTML/diagram được duyệt vào `docs/architecture/diagrams/`; không đưa JSON trạng thái sinh tự động vào docs |
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
- Thêm README riêng cho `.scratch`/initiative chỉ khi có nhu cầu dùng chung; không biến scratch thành tài liệu chuẩn.

**Hoàn tất khi:** người mới tìm được cách chạy, API, kiến trúc, dữ liệu, gameplay và kế hoạch từ tối đa hai lần nhấp.

### 3. Di chuyển theo nhóm nhỏ

- Tạo các thư mục đích cần thiết; ưu tiên `git mv` theo nhóm có cùng mục đích.
- Mỗi đợt chỉ chuyển một nhóm; cập nhật mọi tham chiếu Markdown, AGENTS, README và đường dẫn trong mã/config.
- Tránh đổi tên file cùng lúc với đổi thư mục; chuẩn hóa tên file thành chữ thường/kebab-case chỉ khi có thể cập nhật toàn bộ link.
- Không thay đổi tài liệu thuộc thay đổi gameplay/feature đang dở dang nếu chưa tách được khỏi công việc đó.

### 4. Xử lý kế hoạch và tài liệu hết hiệu lực

- Mỗi plan có trạng thái và ngày cập nhật; kế hoạch hoàn tất được tóm tắt kết quả rồi lưu ở `docs/archive/plans/` hoặc giữ trong lịch sử issue.
- Tài liệu thay thế tài liệu khác phải ghi “superseded by” cùng link; không xóa bản cũ ngay.
- Initiative trong `.scratch/` chỉ được đóng khi map/issues đã được giải quyết hoặc chuyển sang tracker chính thức.

### 5. Hoàn tất và giữ cấu trúc ổn định

- Tìm các link Markdown tương đối, đường dẫn `docs/...` trong source/CI/scripts và các chỉ dẫn agent.
- Kiểm tra đường dẫn đích tồn tại, không còn link cũ, không có tài liệu bị mất; xem diff để chắc không sửa nội dung ngoài phạm vi.
- Thêm quy tắc ngắn vào `AGENTS.md`: tài liệu chuẩn ở `docs/`, scratch là tạm, ADR bất biến, cập nhật link cùng lúc di chuyển.
- Không chạy formatter hàng loạt cho toàn bộ Markdown trong đợt di chuyển.

## Tiêu chí hoàn thành

- Có `docs/README.md` làm mục lục và README root dẫn tới nó.
- Tên nhóm thể hiện mục đích; không có thư mục ẩn mới để chứa tài liệu đọc thường xuyên.
- Các file được chuyển có trạng thái/đích rõ, link nội bộ cập nhật và không còn tham chiếu đường dẫn cũ.
- `.scratch/` chỉ chứa công việc chưa hoàn tất; `.archify/` được xác định rõ là artifact sinh/lưu cục bộ hay đầu ra được duyệt.
- Không mất lịch sử; file cũ được lưu trữ hoặc giữ nguyên cho tới khi xác nhận không còn giá trị.

## Phạm vi không làm trong kế hoạch này

- Không đổi nội dung gameplay, API, schema hoặc hợp đồng runtime.
- Không xóa `.scratch/`, `.archify/` hay tài liệu chỉ vì tên/tuổi của chúng.
- Không di chuyển tài liệu của initiative đang triển khai trước khi có điểm bàn giao.
- Không coi số lượng thư mục ít hơn là mục tiêu; ưu tiên tìm nhanh và sở hữu rõ ràng.

## Kết quả quét Markdown

Đã quét **228 file `.md`** trong repo, gồm thư mục ẩn; bỏ qua `.git/`, `node_modules/` và `vendor/`. Đây là kiểm kê theo đường dẫn và tiêu đề/nội dung mở đầu, không phải đánh giá từng tuyên bố kỹ thuật là còn đúng với code hiện tại.

### Phân bố

| Khu vực | Số file | Nội dung chính / nhận xét |
| --- | ---: | --- |
| `.agents/skills/` | 59 | Hướng dẫn skill cho agent và các mẫu/format đi kèm. Đây là tài liệu cấu hình quy trình; giữ cạnh skill, không chuyển vào `docs/` sản phẩm. Có các bộ hướng dẫn issue tracker GitHub/GitLab/local nên cần giữ rõ lựa chọn đang áp dụng. |
| `.scratch/` | 19 | Ba initiative: tối ưu `FOR UPDATE` (map + 6 issue), mở rộng Tiên Khiếu (map + 5 issue), Tiên Cổ Phòng (map + 5 issue). Là tài liệu công việc chưa chuẩn hóa/trạng thái cần xác minh trước khi đóng gói. |
| Root | 8 | `AGENTS.md`, `README.md`, `PROJECT.md`, `CONTEXT.md`, `WIKI.md` là điểm vào/ngữ cảnh; `OPTIMIZATION_PLAN.md`, `OPTIMIZATION_PROGRESS.md`, `GuWorld_MMO_Realtime_Phaser4_Master_Plan.md` là plan/log dài, cần gắn trạng thái và chủ sở hữu trước khi chuyển. |
| `backend/cmd/wtspike/` | 1 | README cho spike/prototype WebTransport; để cạnh prototype/backend command. |
| `docs/` trực tiếp | 31 | Tài liệu API, phát triển, dữ liệu, deploy/admin, architecture và các spec/gameplay. Đây là nhóm quá đông và nhiều loại đang cùng cấp; ưu tiên phân loại ở đợt đầu. |
| `docs/mmo/` | 26 | Bộ tài liệu MMO: audit/baseline, kiến trúc/protocol, gameplay và vận hành như benchmark, rollout, recovery, scaling, SLO, security, assets, loot, party. Nên thêm mục lục và trạng thái từng tài liệu; tránh tách tiếp nếu chưa cần. |
| `docs/plans/` | 6 | Kế hoạch nhánh realtime, viết lại effect/killer moves, nguồn vị trí player, whitelist spawn, parity combat và kế hoạch sắp xếp tài liệu này. Cần trạng thái nhất quán. |
| `docs/research/` | 17 | Audit/khảo sát API/UI, player state, MMO, spatial realm, Dao, economy, assets, transformation. Phần lớn là snapshot theo thời điểm; cần ngày khảo sát và trạng thái current/superseded. |
| `docs/wayfinder/` | 50 | Hồ sơ initiative lớn: 5 tài liệu map/progress, 21 ticket, 17 spec, 6 research, 1 note. Nên giữ theo initiative; README mục lục giúp điều hướng. |
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
- `.archify/` không có Markdown; qua kiểm kê ban đầu, nó là tập HTML/JSON artifact, nên quản lý riêng khỏi số liệu trên.

### Phạm vi kiểm kê theo nhóm

Danh mục trên bao quát toàn bộ 228 file Markdown theo thư mục và mục đích. Các nhóm 21 ticket và 17 spec Wayfinder, 59 skill agent, cùng 19 file scratch được kiểm tra tiêu đề và phần mở đầu để phân nhóm; nội dung kỹ thuật bên trong từng ticket/skill không được diễn giải lại ở đây vì chúng là hồ sơ đơn vị riêng, không phải một bộ tài liệu dự án thống nhất.
