# Domain Glossary (CONTEXT.md)

## Non-player Actors

### Actor Template
Dữ liệu tĩnh mô tả loài hoặc kiểu của một actor; không đại diện cho một cá thể tồn tại lâu dài.
_Avoid_: Actor Instance, cá thể NPC

### Actor Instance
Cá thể không phải người chơi có định danh và trạng thái dài hạn riêng, dù hiện có xuất hiện trên bản đồ hay không.
_Avoid_: Pet record, NPC record, monster record

### World Presence
Sự hiện diện duy nhất của một Actor Instance trên bản đồ thế giới tại một thời điểm; có thể được tạo khi triển khai và xóa khi thu hồi mà không xóa Actor Instance.
_Avoid_: Actor Instance, spawn definition

### Capability
Năng lực dữ liệu hóa xác định một Actor Instance được phép thực hiện loại hành động nào; `actor_kind` chỉ mô tả danh tính/lore và policy mặc định.
_Avoid_: Actor type check

### Behavior Profile
Policy dữ liệu hóa mô tả thứ tự ưu tiên ra quyết định của actor, nhưng không trực tiếp thay đổi domain state.
_Avoid_: Capability, AI writer

### Assignment
Công việc dài hạn duy nhất mà một Actor Instance đang thực hiện.
_Avoid_: Command, Behavior Profile

### Combat Snapshot
Bản sao bất biến của trạng thái chiến đấu actor tại thời điểm trận đấu bắt đầu; simulator chỉ đọc snapshot này và không truy vấn Actor Instance.
_Avoid_: Actor Instance, live combat state

## Scouting, Divination & Concealment (Trinh Sát, Suy Diễn & Ẩn Thân)

### Trinh Sát Cục Bộ (Local Inspection)
Hành động quan sát trực tiếp thông tin và tu vi của đối phương khi cả hai cùng hiện diện tại một tọa độ hoặc trong phạm vi tầm nhìn cục bộ trên bản đồ Ngũ Vực. Thay thế hoàn toàn cơ chế xem thông tin tự do từ xa.

### Suy Diễn Thiên Cơ (Divination / Deduction)
Hành vi vận dụng Đạo Ngân (Dao Marks) Trí Đạo / Tín Đạo, Cảnh Giới Chân Ý (Path Attainment - Tông Sư/Đại Tông Sư) kết hợp Cổ trùng và Sát chiêu để suy đoán vị trí, tu vi, hoặc cổ trùng của mục tiêu từ xa mà không cần tiếp xúc vật lý. Điểm 3 trụ (Cổ trinh sát / cảnh giới+đạo ngân / manh mối) + xúc xắc Vận đạo, **chỉ so điểm tổng cuối**; client chỉ thấy `my_score` của mình.

### Ấn Ký Truy Tung (Tracking Mark)
Dấu vết đặc thù (Trí Niệm Ấn, Huyết Mạch Ấn, Khí Tức Ấn) được gieo vào đối phương khi va chạm giao tranh hoặc qua sát chiêu trinh sát, cho phép theo dõi tọa độ chính xác của mục tiêu trong một khoảng thời gian giới hạn.

### Cảm Ứng Cổ Trùng (Gu Resonance / Detection)
Năng lực trinh sát phát hiện sự tồn tại, chủng loại hoặc bậc phẩm của Cổ trùng (hoang dã hoặc do Cổ sư khác mang theo) trong một vùng lãnh thổ dựa trên dao động Đạo Ngân.

### Che Giấu Thiên Cơ (Concealment / Disguise)
Năng lực phòng thủ, ngụy trang tu vi, thân phận, vị trí và cổ trùng dựa trên Cổ trùng ngụy trang (Thái Độ Cổ, Biến Hóa Cổ, Vô Tướng Cổ), Sát Chiêu Ẩn Thân, cùng Đạo Ngân / Cảnh Giới thuộc Ám Đạo, Biến Hóa Đạo, Huyễn Đạo hoặc Trí Đạo. Vai 6 đạo: Trí đối kháng trực tiếp, Ám che thân/vị/khí, Tín bóp méo manh mối, Vận làm lệch kết quả, Thiên che thiên cơ (Tiên nhân), Biến hóa đổi dấu vết/thân phận. Cổ có chủ mang che giấu của chủ (trang bị full, nằm kho/ngủ một nửa).

### Chém Đứt Manh Mối (Sever Clues)
Hành vi chủ động thanh tẩy ấn ký truy tung, chém đứt tàn dư khí tức và manh mối thiên cơ còn lưu lại sau giao tranh hoặc di chuyển, ngăn chặn kẻ thù lần theo vết tích.

### Thiên Cơ Phản Phệ (Backfire / Retaliation)
Hiệu ứng trừng phạt xảy ra khi Cổ sư cố gắng suy diễn một mục tiêu có năng lực che giấu / phòng ngự trí đạo vượt trội hoặc có bố trí Sát Chiêu Phản Suy Diễn. Kẻ suy diễn chịu tổn thương linh hồn, thất thoát chân/tiên nguyên và bị lộ ngược lại vị trí cho mục tiêu.

## Modular Effect System & Realm Scaling (Hệ Thống Hiệu Ứng Lắp Ráp & Thang Đo Cảnh Giới)

### Khối Hiệu Ứng Lắp Ráp (Composable Effect Block)
Cấu trúc nguyên tử hóa **đang ở mức prototype dữ liệu**, mô tả hiệu ứng theo mô hình ECA (Event-Condition-Action): gồm `Trigger` (thời điểm xảy ra sự kiện), `Conditions` (điều kiện lọc ngữ cảnh), `Target` (mục tiêu thụ hưởng/chịu đòn) và `Operations` (hành động/hiệu ứng thành phần như Damage, Shield, Status, StatMod). Backend gameplay hiện tại chưa thực thi `blocks`; frontend mới chỉ đọc một phần để hiển thị, chỉnh sửa và phân loại Cổ khi dựng đội. Các hành vi gameplay vẫn phụ thuộc trực tiếp vào `active`, `passive`, `growth` và `other`. Vì vậy không được coi `blocks` là nguồn dữ liệu chuẩn hoặc xóa các namespace cũ cho tới khi có executor, chuyển đổi đủ dữ liệu và kiểm thử tương đương hành vi.

### Suy Hao Cảnh Giới Tự Thân (Caster Rank Attenuation)
Cơ chế suy giảm uy lực và thời lượng của hiệu ứng buff/bảo hộ khi Cổ sư cấp cao sử dụng Cổ trùng có Rank thấp hơn cảnh giới của bản thân áp lên chính mình (ví dụ Cổ Rank 4 tạo hiệu ứng Bất Tử cho Cổ Tiên Rank 9). Hiệu quả bảo vệ suy giảm mạnh theo hàm mũ hoặc chuyển đổi thành giá trị hữu hạn để tránh phá vỡ cân bằng cảnh giới.

### Áp Chế Cảnh Giới Mục Tiêu (Target Realm Suppression)
Cơ chế bảo hộ và kháng cự tự nhiên của mục tiêu cấp cao trước các hiệu ứng bất lợi (debuff, khống chế, sát thương theo % máu) từ Cổ trùng cấp thấp. Tỉ lệ kích hoạt thành công (Proc Chance) và thời gian duy trì (Duration) của debuff bị suy giảm theo độ chênh lệch giữa Rank Cổ và Rank Mục Tiêu (`TargetRank - GuRank`).

### Xác Suất Kích Hoạt Động (Dynamic Proc Chance)
Tỉ lệ kích hoạt thực tế của một hiệu ứng trong chiến đấu, được suy ra từ Xác suất cơ bản (`BaseChance` theo phân loại độ mạnh của hiệu ứng), cộng dồn tăng tiến theo Rank Cổ trùng (`+delta * (GuRank - 1)`), và điều chỉnh sau cùng thông qua Áp Chế Cảnh Giới, Kháng Tính Tâm Thần (Mental Resistance) và Vận Số (Luck).
