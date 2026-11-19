# Wayfinder Map: Thư Viện Hình Ảnh Gameplay GuWorld

## Destination

Đặc tả có thể bàn giao để xây dựng và tích hợp thư viện hình ảnh gameplay cho cổ trùng, item/nguyên liệu, mỏ tài nguyên và biểu tượng khám phá. Đặc tả chốt phong cách hình ảnh, nhóm và độ phủ asset, cách tổ chức/tạo asset, và cách nối chúng vào dữ liệu cùng giao diện hiện có.

## Notes

- Domain: frontend gameplay assets, Gu/item dictionaries and CSV image pointers, resource/mining exploration UI.
- Reference: atlas `icon_v2_atlas.png`, atlas `o_tui_atlas.png`, plus assets and resolvers already in the repository.
- Current state observed: `frontend/public/assets/gu_icons/` has 2,010 images (438 root, 1,557 family, 15 custom); `frontend/public/assets/items/icons/` has 75 files. World assets include monster atlases and a small number of resource illustrations. Gu family sprite mappings and item icon resolution already exist.
- Skills to consult: `grilling`, `domain-modeling`, `prototype`, `research` if external asset/tool facts become necessary.
- Standing preferences: preserve the existing pixel-art game identity; prefer reusable, readable icons at actual UI size; respect existing CSV `image_url` and frontend resolver paths; do not replace unrelated player/world/VFX art.
- Tracker note: GitHub Issues is canonical. This repo's existing Wayfinder maps record child links and blockers in issue bodies because native sub-issue/dependency operations are unavailable through the current CLI/API workflow.
- Canonical tracker: [Wayfinder: Thư viện hình ảnh gameplay GuWorld](https://github.com/VipPro67/GuWorld/issues/433). Local files are context copies; GitHub issues are canonical.

## Decisions so far

<!-- One line per closed ticket; the ticket holds the decision. -->

- [Choose a visual direction and icon readability rules](https://github.com/VipPro67/GuWorld/issues/434): Chọn hybrid ưu tiên V2, giữ pixel-art với silhouette rõ ở 24 px, nền trong suốt và không đóng khung cố định trong asset.
- [Define gameplay asset groups and coverage boundaries](https://github.com/VipPro67/GuWorld/issues/435): Mỗi Gu ID có icon riêng; item đặc trưng có ảnh riêng và phần còn lại fallback theo nhóm; marker mỏ riêng theo loại, state thể hiện ở UI.
- [Choose the asset production and delivery pipeline](https://github.com/VipPro67/GuWorld/issues/436): Phát hành file WebP riêng theo ID, giữ PNG master; atlas chỉ làm tham chiếu phong cách; duyệt mẫu theo nhóm, kiểm tra tự động và manifest ghi nguồn/prompt.
- [Specify in-game mapping and rollout strategy](https://github.com/VipPro67/GuWorld/issues/437): CSV `image_url` hợp lệ ưu tiên, sau đó manifest ID rồi fallback hiện tại; tích hợp qua resolver dùng chung và giữ ảnh cũ trong quá trình chuyển đổi.
- [Inventory exact gameplay icon coverage and resource marker kinds](https://github.com/VipPro67/GuWorld/issues/439): Có 3.472 Gu ID (3.422 family, 15 custom, 35 legacy/heuristic); 897 item đang fallback về 50 nhóm; 47 điểm tài nguyên hiện dùng glyph.

## Open Tickets (Frontier & Blocked)

- **Frontier**
  - [Set the first asset rollout batch and acceptance gates](https://github.com/VipPro67/GuWorld/issues/438) (local context: [ticket](issues/05-first-rollout-batch-and-acceptance-gates.md))

- **Blocked**
  - None.

## Not yet specified

- Không còn câu hỏi mơ hồ đã nhận diện; kiểm kê catalog và chọn batch là các ticket đang mở.

## Out of scope

- Character portraits, world background/biome atlases, combat VFX, and UI frame redesign; this effort covers gameplay icons and small resource/exploration illustrations only.
- Gameplay balance, item/Gu catalog redesign, and new gameplay systems.
