# Wayfinder Map: Concept effect chiến đấu lấy cảm hứng từ hình tượng Cổ giới

## Trạng thái

**Tài liệu thiết kế, chưa triển khai.** Các concept bên dưới không mô tả tính năng hiện hành và không phải cam kết về chỉ số, cân bằng hay lịch phát hành.

## Đích đến

Tạo một nhóm concept effect có khác biệt chiến thuật cho Cổ trùng, Sát Chiêu, Trận Pháp và Tiên Cổ Ốc. Chỉ đưa vào gameplay sau khi concept được duyệt, consumer runtime được xác định, và cách khắc chế/cân bằng được chốt.

## Cách dùng tài liệu tham khảo

Google Doc [Tổng hợp thơ ca nhân vật và hoang thú](https://docs.google.com/document/d/1C7FAPHDOKaEO04NDfuN8Crox10sRPuCol7PYIFHh8Ng/edit?tab=t.e8auqb20l2vn) được dùng làm nguồn hình tượng và sắc thái. Phần đọc được có thơ gắn với nhân vật cùng danh sách hoang thú; không thấy mô tả luật effect, trigger, chỉ số hoặc quy tắc chiến đấu. Bản export bị timeout nên nguồn đọc được có thể chưa đầy đủ.

Các concept chỉ mượn mô-típ rộng như ý chí độc hành, biến chuyển cát hung, tích lũy qua thời gian và sức ép của sinh vật hoang dã. Tên, mô tả và cơ chế dưới đây được tạo riêng cho GuWorld; không chép thơ, tên nhân vật hay tên sinh vật từ tài liệu.

## Nguyên tắc thiết kế

- Mỗi concept tạo một quyết định chiến thuật, không chỉ tăng sát thương hoặc đổi tên status.
- Mỗi concept có cách đối phó, giới hạn hoặc đánh đổi rõ ràng.
- Phân biệt “catalog cho phép” với “runtime thực sự thực thi”; không đưa effect vào data nếu context chưa có consumer.
- Ưu tiên một concept Gu/Sát Chiêu có thể dùng runtime hiện hành làm lát cắt đầu tiên. Trận Pháp và các Trận Kích Tiên Cổ Ốc cần xác nhận/thiết kế executor riêng trước khi triển khai.
- Chưa gán trị số. Chỉ định lượng sau khi xác định rank, resource budget, thời lượng trận và tiêu chuẩn cân bằng.

## Bộ concept

### Cổ trùng

#### 1. Tích Lũy Linh Giáp

- **Vai trò:** Gu phòng thủ biến việc chịu áp lực thành cơ hội phản công.
- **Vòng chơi:** Khi chủ nhân bị đánh trúng, tích một tầng Linh Giáp; lần ra chiêu kế tiếp tiêu tầng để nhận lá chắn ngắn hạn. Tầng bị xóa khi hết trận.
- **Đối sách:** Đối thủ có thể dồn nhiều đòn nhỏ trước khi chủ nhân kịp dùng tầng, hoặc ép chủ nhân tiêu tầng vào thời điểm không thuận lợi.
- **Khả năng triển khai:** Có thể ghép từ trigger nhận đòn, status/tầng và SHIELD nếu combat consumer cho phép cùng một effect chain. Cần kiểm tra đường `ON_HIT` và quy tắc stack trong simulator trước khi chốt data; nếu không thể tiêu tầng có điều kiện, cần hook nhỏ thay vì giả lập bằng nhiều block độc lập.

#### 2. Hoang Tức Truy Ảnh

- **Vai trò:** Gu săn đuổi mục tiêu cơ động, đổi khả năng dồn sát thương lấy áp lực liên tục.
- **Vòng chơi:** Đòn đầu đánh dấu một mục tiêu. Các lần chủ nhân đánh trúng mục tiêu đó gây thêm một lượng sát thương phụ có giới hạn; đổi mục tiêu làm mất dấu.
- **Đối sách:** Mục tiêu có thể dùng thanh tẩy, né giao tranh hoặc buộc chủ nhân đổi mục tiêu.
- **Khả năng triển khai:** STATUS trên TARGET và DAMAGE có sẵn ở cấp catalog, nhưng cần xác nhận status đánh dấu có thể được truy vấn khi tính damage và mục tiêu mark có định danh ổn định trong combat state. Nếu chưa có, đây là một hook combat mới.

### Sát Chiêu

#### 3. Hồi Phong Hộ Mệnh

- **Vai trò:** Sát Chiêu phòng thủ khẩn cấp, hy sinh nhịp tấn công để chống một lượt bùng nổ.
- **Vòng chơi:** Dùng hết hành động trong lượt để gỡ debuff trên bản thân và tạo lá chắn ngắn hạn; người chơi chọn nhịp phòng thủ thay cho gây sát thương.
- **Đối sách:** Đối thủ có thể nhử Sát Chiêu trước đợt sát thương lớn hoặc dùng phá khiên.
- **Khả năng triển khai:** CLEANSE và SHIELD đã có trong bộ operation; cần xác minh composition/context của Sát Chiêu cho phép ghép cả hai trong cùng action. Có thể cân bằng bằng mana và chi phí lượt hiện hành, chưa cần thêm condition hoặc cooldown mới.

#### 4. Thiên Kích Liên Hoàn

- **Vai trò:** Sát Chiêu kết liễu có thể chuyển nhịp tiến công sang mục tiêu kế tiếp.
- **Vòng chơi:** Gây một đòn chính; nếu đòn hạ gục mục tiêu, phần uy lực dư chuyển thành một đòn yếu hơn lên mục tiêu địch còn sống tiếp theo. Chỉ nối tối đa một lần mỗi lượt.
- **Đối sách:** Khiên hoặc giảm sát thương có thể ngăn đòn đầu hạ gục; đội hình địch rộng làm giảm độ tin cậy của chuỗi.
- **Khả năng triển khai:** Đòn DAMAGE thông thường được hỗ trợ, nhưng kiểm tra kết liễu, chọn mục tiêu kế tiếp, chuyển sát thương dư và giới hạn kích hoạt là hành vi simulator mới. Chưa thể biểu diễn an toàn chỉ bằng Gu.csv.

### Trận Pháp

#### 5. Tam Mạch Trấn Vực

- **Vai trò:** Trận pháp kiểm soát tuyến, thưởng cho việc giữ đủ vị trí trong một đội hình.
- **Vòng chơi:** Ba đồng minh ở các vị trí hợp lệ tạo thành ba mạch. Khi đủ mạch, địch ở tuyến đối diện bị giảm tốc; khi một vị trí mất hiệu lực, trận suy yếu thay vì tiếp tục áp toàn bộ hiệu ứng.
- **Đối sách:** Hạ gục, khống chế hoặc buộc một mắt trận rời vị trí sẽ phá cấu hình.
- **Khả năng triển khai:** Cần mô hình vị trí/ô chiến trường, điều kiện đủ mắt trận và event khi đội hình thay đổi. Simulator hiện có target ROW/COL và một số mechanic Trận Đạo, nhưng chưa đủ để suy ra có hệ thống đội hình ba mắt trận tổng quát.

#### 6. Hồi Âm Liên Tâm

- **Vai trò:** Trận pháp hỗ trợ chia sẻ phòng thủ, đổi khả năng bảo hộ cả nhóm lấy nguy cơ sụp dây chuyền.
- **Vòng chơi:** Một mắt trận chịu một phần sát thương thay cho đồng minh trong vùng; phần chuyển hướng làm suy yếu mắt trận. Khi mắt trận cạn sức, chia sẻ dừng đến lượt sau.
- **Đối sách:** Tập trung đánh mắt trận hoặc dùng nhiều đòn nhỏ để làm cạn khả năng chia sẻ.
- **Khả năng triển khai:** Cần phạm vi/quan hệ vị trí và bước chuyển mục tiêu sát thương trong damage pipeline. Không mô phỏng bằng SHIELD lên tất cả đồng minh vì sẽ thay đổi ý nghĩa sát thương và tương tác khiên.

### Tiên Cổ Ốc

#### 7. Tụ Tinh Trận Kích

- **Vai trò:** Trận Kích tích lực, buộc người chơi chọn giữa áp lực tức thời và đòn đánh diện rộng.
- **Vòng chơi:** Dành một lượt tụ lực; lượt sau bắn một đòn mạnh vào một tuyến. Bị choáng hoặc rời Cổ Phòng trước khi khai hỏa làm mất phần tích lực.
- **Đối sách:** Phá nhịp tụ lực bằng khống chế, hoặc tản đội hình khỏi tuyến dự kiến.
- **Khả năng triển khai:** Gu House hiện có action theo lượt và composition riêng, nhưng cần trạng thái charge gắn với action, target tuyến và quy tắc hủy charge. Cần mở rộng action profile/simulator; không đặt thành DAMAGE block đơn giản.

#### 8. Huyền Môn Phản Chấn

- **Vai trò:** Trận Kích phản chế, biến đòn đánh của địch vào Cổ Phòng thành một lần trả đòn có giới hạn.
- **Vòng chơi:** Sau khi Cổ Phòng chịu một đòn đủ mạnh, tích một ấn; Trận Kích kế tiếp tiêu ấn để phản chấn một mục tiêu. Tối đa một ấn và không kích hoạt từ sát thương môi trường.
- **Đối sách:** Dùng đòn nhỏ để tránh ngưỡng tích ấn hoặc ép người chơi phản chấn vào mục tiêu ít giá trị.
- **Khả năng triển khai:** Cần phân loại nguồn sát thương, ngưỡng kích hoạt và state của Gu House gắn với simulator turn. Damage absorption/Gu House action đã tồn tại, nhưng chuỗi phản ứng này cần hook riêng.

## Thứ tự thử nghiệm đề xuất

1. Kiểm tra **Hồi Phong Hộ Mệnh** trước: phần lớn operation đã có, nên có thể xác minh nhanh các giới hạn context và conditions của Sát Chiêu.
2. Kiểm tra **Tích Lũy Linh Giáp** tiếp theo nếu simulator hỗ trợ lưu, tăng và tiêu tầng trong đúng các trigger cần thiết.
3. Chỉ mở rộng sang Trận Pháp sau khi chốt mô hình vị trí và vòng đời mắt trận.
4. Chỉ mở rộng Trận Kích Tiên Cổ Ốc sau khi chốt state charge/phản ứng và action contract của Gu House.

## Điều kiện chuyển từ thiết kế sang triển khai

- Chọn một concept đầu tiên và chốt rank, chi phí, thời lượng, số lần kích hoạt, stacking, counterplay.
- Chỉ rõ source data và executor chịu trách nhiệm cho từng trigger/operation.
- Thêm hoặc cập nhật test ở simulator/adapter để xác minh hành vi, context không hỗ trợ và deterministic combat.
- Nếu thêm trigger, operation, status hoặc native hook, cập nhật effect catalog/glossary và docs gameplay bắt buộc; nội dung vẫn được đánh dấu kế hoạch cho đến khi source thực thi tồn tại.

## Liên hệ với mã nguồn hiện tại

- Effect schema, validation và context adapters: `backend/internal/effectcatalog/`, `backend/internal/gameplay/effects/`.
- Combat turn-based: `backend/internal/simulator/`.
- Composition và executor profile cho Sát Chiêu/Gu House: `backend/internal/gameplay/composition/`.
- Tài liệu hành vi hiện tại: [Danh mục effect](../../../guides/EFFECTS.md), [Dữ liệu gameplay](../../../guides/GAME_DATA.md), [Sát Chiêu và Tiên Cổ Ốc](../../../features/killer-moves-and-gu-houses.md).
