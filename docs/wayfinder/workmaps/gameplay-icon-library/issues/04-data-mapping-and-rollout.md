# Specify in-game mapping and rollout strategy

Type: grilling
Status: resolved
Blocked by: none

Part of [Wayfinder: Thư viện hình ảnh gameplay GuWorld](https://github.com/VipPro67/GuWorld/issues/433); canonical ticket: [Specify in-game mapping and rollout strategy](https://github.com/VipPro67/GuWorld/issues/437).

## Question

How should the approved assets be mapped into Gu and item records and into mining/exploration UI? Decide which existing CSV `image_url` values are authoritative, where resolver fallbacks or manifests should own mappings, how unknown/new records behave, and how assets roll out without breaking existing image URLs or cached registry data. Identify the frontend entry points and documentation that implementation must update.

## Resolution

- Thứ tự nguồn ảnh: `image_url` hợp lệ khai báo trên record là override ưu tiên cao nhất; nếu không có thì tra manifest asset theo ID ổn định; nếu manifest cũng chưa có thì giữ fallback hiện tại. Resolver item và Gu cùng áp dụng thứ tự này; item ID có hậu tố `_rankN` dùng chung artwork của ID gốc.
- Shared resolvers are wired to Gu inventory, utilities, combat, shops, sect vault, trade, world resource cells, and aperture resource-point cards. Old custom/family/category/emoji fallbacks remain active when no manifest image exists.
- Tích hợp qua resolver dùng chung, không gắn đường dẫn ad-hoc vào từng component. Các điểm nối hiện tại là `frontend/src/lib/gu-icons.ts`, `frontend/src/lib/item-icons.ts`, `frontend/src/lib/worldResourceAssets.ts` và marker render trong `frontend/src/components/ExplorationMap.tsx`.
- Giữ nguyên các URL và ảnh cũ trong quá trình chuyển đổi; asset mới được thêm theo ID vào manifest và resolver. Không cần sửa dữ liệu người chơi. Nếu sau này chỉnh CSV `image_url`, import dataset bằng Admin và khởi động lại backend theo quy tắc registry hiện tại.
- Đường đi dữ liệu hiện tại: `backend/data/Gu.csv` và `backend/data/Item.csv` chứa `image_url`; item UI gọi `getItemIconUrl`, còn Gu UI gọi `getGuIconUrl`. Tài nguyên trên bản đồ hiện gom bốn nhóm qua `getWorldResourceGroup`.
- Khi triển khai, cập nhật mô tả đường đi player-facing trong `docs/guides/FEATURES.md` và tài liệu liên quan đến asset/data; đối chiếu `WIKI.md` và `docs/guides/GAME_DATA.md` nếu thay đổi hành vi hoặc cách quản lý image data.
