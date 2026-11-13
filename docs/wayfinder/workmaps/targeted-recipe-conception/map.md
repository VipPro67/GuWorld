## Destination

Một bản đặc tả kỹ thuật (Spec) và kiến trúc hệ thống hoàn chỉnh cho tính năng "Định Hướng Thôi Diễn Bí Phương / Khai Sáng Tàn Phương", cho phép Cổ sư vận dụng Niệm Lực, Nguyên Thạch, Cảnh Giới Chân Ý (Trí Đạo, Luyện Đạo, Bản Lưu Phái) và Vật Phẩm Dẫn Dắt để suy tính tàn phương cho Cổ trùng mong muốn thay vì phụ thuộc vào rơi ngẫu nhiên.

## Notes

- Domain: Cổ Chân Nhân / Luyện Cổ / Thôi Diễn Trí Đạo.
- Skills: `grilling`, `domain-modeling`, `codebase-design`.
- Standing preferences: 
  - Tận dụng cấu trúc bảng dữ liệu `refining_recipes` và `player_recipe_progress` hiện có.
  - Tận dụng bộ cân bằng `DeductionRealmMultiplier` và `gamefx.PathMasteryLevel`.
  - Giữ vững quy tắc: Cổ có sẵn trong từ điển `gu_dictionary` (4,713 Cổ), không tự sinh Cổ tùy tiện.
  - Phản phệ nghiêm khắc khi vượt cấp tu vi hoặc phàm nhân cưỡng cầu Tiên Cổ.

## Decisions so far

<!-- the index: one line per closed ticket, enough to judge relevance, then zoom the link for the detail the ticket holds -->

- [Mô hình toán cân bằng và cơ chế lệch hướng](issues/01-math-balance-and-deviation-formula.md): Chốt công thức tỉ lệ trúng đích (15%-88%, trần 88% thiên cơ), thuật toán lệch hướng sang Cổ cùng hệ rank/rank-1, điều kiện phản phệ vượt cấp (mất 50% HP/Mana, 0 niệm lực), và độ hoàn thiện khởi điểm (15%-35%).
- [Quy chuẩn Cổ phôi và Vật dẫn nghiên cứu](issues/02-catalyst-specimen-rules.md): Chốt điều kiện hiến tế Cổ phôi (không bản mệnh, không trang bị, cùng hệ, rank <= target), định lượng vật liệu thay thế (5-50 đơn vị), và mức bonus tỉ lệ trúng đích (+8% đến +20%), giữ nguyên chi phí niệm lực.
- [Hợp đồng API Backend và Tối ưu Database](issues/03-backend-api-and-db-contracts.md): Chốt 2 endpoints GET /api/refine/conceive/candidates và POST /api/refine/conceive, DTO request/response, truy vấn lọc loại trừ bí phương 100%, quy trình transaction và audit/event snapshot.
- [Giao diện và luồng tương tác trong Refine Modal](issues/04-frontend-ui-flow-in-refine-modal.md): Chốt bố cục tab Khai Sáng trong RefiningFurnace.tsx, bảng dự toán thiên cơ (progress bar tỉ lệ, rủi ro phản phệ, vật dẫn), modal thông báo kết quả sống động (thành công/lệch nhánh/linh quang 100%/phản phệ) và nút tắt từ Cổ Phổ Codex.

## Not yet specified

<!-- see "Fog of war": in-scope fog you can't ticket yet; graduates as the frontier advances -->

- Tương tác với Cổ trùng chuyên dụng Trí Đạo (như Định Trí Cổ, Đa Niệm Cổ, Trí Tuệ Cổ) và Sát chiêu Trí Đạo hỗ trợ khai sáng.
- Khả năng xuất bản/phong ấn Tàn Phương thành Ngọc Giản để buôn bán trên Chợ Đen (Marketplace) hoặc cống hiến Tàng Kinh Các Tông Môn (Sect Vault).
- Chuỗi Thành Tựu (Achievements) và Nhiệm Vụ (Quests) vinh danh đại năng khai sáng cổ phương.

## Out of scope

<!-- see "Out of scope": work ruled beyond the destination; closed, never graduates -->

- Tự thiết kế Cổ hoàn toàn mới ngoài `gu_dictionary` (custom procedurally generated stats/names).
- Suy tính bí phương Tiên Cổ (Rank 6+) khi người chơi vẫn đang ở cảnh giới Phàm nhân (Rank 1-5).
- Miễn phí Niệm Lực / Chân Nguyên khi khai sáng.
