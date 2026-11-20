# Choose the asset production and delivery pipeline

Type: grilling
Status: resolved
Blocked by: [Choose a visual direction and icon readability rules](https://github.com/VipPro67/GuWorld/issues/434), [Define gameplay asset groups and coverage boundaries](https://github.com/VipPro67/GuWorld/issues/435)

Part of [Wayfinder: Thư viện hình ảnh gameplay GuWorld](https://github.com/VipPro67/GuWorld/issues/433); canonical ticket: [Choose the asset production and delivery pipeline](https://github.com/VipPro67/GuWorld/issues/436).

## Question

Given the agreed visual rules and coverage, how should assets be authored or generated, reviewed, stored, and served? Decide individual files versus atlases/sprite sheets (or a mix), source/editable artifact expectations, output formats and dimensions, transparency, naming/manifest conventions, licensing/provenance requirements, and any tooling needed to produce a maintainable library that fits current frontend usage.

## Resolution

- Gu và marker phát hành thành file riêng theo ID ổn định; item có manifest record riêng theo ID nhưng được phép trỏ nhiều ID cùng archetype vào một artwork. WebP là định dạng runtime, PNG là master lossless để tái xuất và sửa.
- Hai atlas được chia sẻ chỉ dùng làm tham chiếu phong cách. Tạo artwork mới; không crop hay tái sử dụng sprite của atlas trực tiếp.
- Duyệt mẫu đại diện theo từng nhóm asset trước khi mở rộng sản xuất. Với mỗi batch, tự kiểm tra tên/ID, kích thước chuẩn và alpha/nền trong suốt; lỗi khiến batch cần sửa trước khi tích hợp.
- Tên đầu ra dựa trên ID ổn định trong catalog, không dựa vào tên hiển thị có thể đổi. Manifest đi kèm lưu ID, đường dẫn master/runtime và nguồn tạo/prompt để truy xuất, kiểm tra coverage và tái tạo batch.

## Implementation notes

- Runtime manifest: `frontend/src/data/game-icon-manifest.json`; every entry carries `runtime`, `master`, `source`, and `prompt`.
- PNG masters: `art/game-icons/masters/{gu,item,resource-point}/{ID}.png`; shared item archetypes use a stable archetype filename.
- WebP runtime files: `frontend/public/assets/game-icons/{gu,item,resource-point}/{ID-or-archetype}.webp`, built at 256×256 with transparency from each PNG master. Current item manifest covers 44 IDs with 25 distinct artworks; 20 common beast-material IDs share one archetype, and rank variants resolve to the same canonical art.
- Run `cd frontend && npm run build:game-icons` to produce runtime files, then `npm run validate:game-icons` to check CSV IDs, stable paths, metadata, file existence, dimensions, and alpha. The validator intentionally fails while the manifest has no delivered icons.
- Grouped source sheets and per-output provenance are recorded in `art/game-icons/sources/manifest.json`; reference art paths identify the existing Gu and item icon libraries used for style only.
- The first batch is delivered and `npm run validate:game-icons:batch` passes all 126 planned IDs. Temporary individual 24×24 QA previews and the superseded sample directory were removed after review; production PNG masters, source sheets, and runtime WebPs remain.
