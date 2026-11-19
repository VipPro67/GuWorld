# Define gameplay asset groups and coverage boundaries

Type: grilling
Status: resolved
Blocked by: none

Part of [Wayfinder: Thư viện hình ảnh gameplay GuWorld](https://github.com/VipPro67/GuWorld/issues/433); canonical ticket: [Define gameplay asset groups and coverage boundaries](https://github.com/VipPro67/GuWorld/issues/435).

## Question

Which concrete asset groups and coverage targets belong in this library? Inspect current Gu/item catalogs and exploration/resource UI, then decide the granularity for Gu identity/family, item and material classes, mine/resource deposits, and exploration markers. Clarify when assets should be unique versus shared category fallbacks, and whether rank, element, biome, or state variants are required. Keep portraits, map backgrounds, and VFX outside this scope unless the destination is explicitly changed.

## Resolution

- Cổ trùng: độ phủ hình ảnh theo từng `Gu ID`; các cấp rank của cùng một ID dùng chung icon. Không gom khác Gu ID vào một icon family chỉ vì cùng họ/đạo.
- Item/nguyên liệu: tạo icon riêng cho item tài nguyên/vật phẩm đặc trưng; các item còn lại dùng fallback theo nhóm vật thể/tài nguyên để giữ độ phủ toàn catalog mà không cần minh họa riêng mọi dòng.
- Mỏ và khám phá: có marker riêng theo loại mỏ/tài nguyên. Trạng thái như đang khai thác, cạn hoặc chưa sẵn sàng được UI biểu diễn bằng overlay/state treatment, không nhân bản ảnh gốc theo từng trạng thái.
- Phạm vi vẫn là icon và hình minh họa nhỏ trong gameplay hiện tại; không mở rộng sang chân dung, ảnh nền biome hay VFX.
