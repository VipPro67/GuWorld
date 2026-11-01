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

### Entity Companion
Thực thể đồng hành (`COMPANION_BEAST` hoặc `VARIANT_NPC`) trong mô phỏng chiến đấu realtime (`sim.World`), liên kết với `OwnerID` người chơi, kế thừa phe phái/tổ đội và có AI thực thi lệnh chiến thuật.
_Avoid_: EntityMonster, pet record

### Nô Thú Channeling (Enslavement Channeling)
Hành vi dẫn kênh thi triển Nô Thú Cổ lên quái hoang trong realtime khi HP dưới ngưỡng hoặc bị khống chế để bắt thú thành Actor Instance.
_Avoid_: Turnbase capture, instant catch

### Lệnh Chiến Thuật (Pet Command Stance)
Trạng thái chỉ huy realtime mà người chơi áp dụng cho toàn bộ Companion/Minion thuộc quyền sở hữu (`Hộ Vệ / Defend`, `Tập Trung Tiêu Diệt / Focus Target`, `Thu Hồi / Follow`).
_Avoid_: RTS micro-selection

### Sát Chiêu Bầy Đàn (Swarm Summoning)
Hành động triệu hồi một đàn thú phụ tá (Minions) tạm thời tham chiến trong khoảng thời gian có hạn, số lượng phụ thuộc vào quy mô Hồn Phách và đẳng cấp Nô Cổ.
_Avoid_: Permanent pet clone

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

## Universal Dao Conflict & Multi-Path Impurity (Vạn Đạo Tương Xung & Trừng Phạt Đa Tu)

### Đa Tu / Tạp Tu (Multi-Path Impurity)
Trạng thái Cổ Tiên tích lũy Đạo Ngân dàn trải trên nhiều lưu phái độc lập mà không đạt cảnh giới tương ứng hoặc không sở hữu thể chất đặc thù (như Chí Tôn Tiên Thai). Độ thuần khiết (`Purity`) sụt giảm và số lưu phái kích hoạt (`ActivePaths`) vượt ngưỡng an toàn sẽ kích hoạt hệ thống trừng phạt lũy tiến.

### Vạn Đạo Tương Xung (Universal Dao Conflict)
Cơ chế xung đột bản nguyên giữa các dòng quy tắc tự nhiên bên trong nhục thân và tiên khiếu Cổ Tiên. Xung đột tác động lên ba tầng chiến đấu: Suy giảm uy lực sát chiêu (Damage Penalty với sàn hạ sâu), Gia tăng hao tổn Tiên Nguyên (Mana Tax), và Nguy cơ phản phệ sát chiêu (Backfire / Instability gây nội thương, mất lượt).

### Độ Thuần Khiết Đạo Ngân (Dao Mark Purity)
Tỷ lệ giữa lượng Đạo Ngân của lưu phái chủ đạo so với tổng số Đạo Ngân trên toàn bộ cơ thể. Độ thuần khiết thấp kết hợp với số lượng phái ngoại lai sẽ nhân theo hàm lũy tiến trừng phạt hiệu năng toàn diện.

### Vô Thượng Hóa Giải (Supreme Grandmaster Absolute Mitigation)
Đặc quyền cảnh giới Vô Thượng Đại Tông Sư (Supreme Grandmaster - Level 4) đối với một lưu phái: triệt tiêu 100% xung đột nội tại mà các đạo ngân dị hệ gây ra cho lưu phái đó, biến mọi đạo ngân tạp nham thành công cụ phụ trợ hoặc loại bỏ hoàn toàn ma sát quy tắc.

### Dị Biến Tiên Khiếu (Aperture Environmental Derangement)
Tác động tiêu cực của tạp tu lên Tiên Khiếu: các đạo ngân xung khắc làm suy thoái sản lượng tài nguyên sinh thái và khiến tai kiếp định kỳ (Địa Tai, Thiên Kiếp) biến dị thành kiếp nạn lai tạp đa hệ hỗn loạn, gia tăng độ nguy hiểm khi độ kiếp.

### Thanh Đạo Tẩy Thể (Dao Mark Cleansing / Expulsion)
Cơ chế/vật phẩm cho phép Cổ Tiên chủ động phân giải, tẩy trừ hoặc dẫn lưu các đạo ngân tạp nham ra khỏi cơ thể để khôi phục độ thuần khiết và thoát khỏi trạng thái phản phệ.

## System Economy & Valuation (Kinh Tế Hệ Thống & Định Giá)

### Tỷ Lệ Thu Mua Thương Nhân (Merchant Buy-back Spread)
Tỷ lệ chiết khấu khi NPC (Thương đoàn Caravan, Thôn trang, Thương nhân lưu động) thu mua vật phẩm, nguyên liệu và Cổ trùng từ người chơi so với giá cơ sở (`base_price`). Tỷ lệ này phân tầng theo Rank và chủng loại để bảo đảm người chơi bán chiến lợi phẩm farm được luôn có lãi ròng sau chi phí, đồng thời chống đầu cơ kiếm lời từ việc mua đi bán lại với hệ thống.

### Kỳ Vọng Hợp Luyện (Refinement Expected Value)
Nguyên tắc định giá Cổ trùng: Giá trị Cổ bậc cao sau hợp luyện phải lớn hơn tổng chi phí nguyên liệu kỳ vọng chia cho xác suất thành công bình quân ($\text{Value}(Rank_{N+1}) > \frac{\text{TotalCost}}{\text{SuccessRate}}$). Đảm bảo người chơi chấp nhận rủi ro luyện Cổ khi thành công luôn có thặng dư kinh tế tương xứng.

### Tiên Phàm Giới Hạn Kinh Tế (Mortal-Immortal Economic Boundary)
Ranh giới phân tách triệt để giữa kinh tế Phàm trần (Rank 1–5, dùng Nguyên Thạch, giao dịch tại Thôn trang/Thương đoàn, quyên góp Tông môn) và Tiên giới (Rank 6–9, dùng Tiên Nguyên Thạch, giao dịch Tiên Cổ và Tiên Tài). Tông môn phàm giới không thu nạp Tiên Tài; quy đổi tiền tệ chỉ cho phép 1 chiều từ Nguyên Thạch lên Tiên Nguyên Thạch với tỷ giá cao để phục vụ Cổ Tiên sơ kỳ.

## Real-world Exploration & Inheritances (Mộng Cảnh, Bí Cảnh & Truyền Thừa)

### Mộng Cảnh (Dream Realm)
Không gian tiềm thức và ký ức tàn dư của tiền nhân hoặc thiên địa, vận hành thuần túy theo cơ chế kịch bản tình huống lựa chọn (Scenario-Choice Q&A). Cổ sư dùng thần hồn tiến vào, chịu rủi ro tổn hao Hồn Lực (Soul HP) nếu lựa chọn sai, có thể trợ lực bằng Cổ Mộng Đạo (Cổ Giải Mộng). Phần thưởng tối hậu là Chân Ý (Path Attainment) giúp thăng cấp cảnh giới lưu phái. Tuyệt đối không gắn với bản đồ di chuyển không gian.
_Avoid_: Bí cảnh, phó bản bản đồ, dungeon bản đồ

### Bí Cảnh Không Gian (Spatial Secret Realm)
Không gian tiểu giới hoặc phúc địa vô chủ tồn tại độc lập với Ngũ Vực nhưng có lối vào/lối ra cụ thể trên bản đồ thế giới. Mỗi phiên là một map riêng có scope `SHARED`, `PARTY` hoặc `SOLO`, thời hạn và luật cấu hình; hoạt động theo tài nguyên thật của nhân vật và đưa người chơi về vị trí đã lưu khi thoát hoặc hết giờ. Cổ sư tự do di chuyển theo tọa độ, khai thác tài nguyên, săn quái, tranh đoạt hoặc tương tác với môi trường theo luật bí cảnh.
_Avoid_: Mộng cảnh, dungeon trả lời câu hỏi

### Truyền Thừa Tiền Nhân (Spatial Inheritance Ground)
Không gian khảo nghiệm kín đáo do tiền nhân, ma đầu hoặc chính đạo cao thủ lưu lại để tuyển chọn truyền nhân. Vận hành theo mô hình phiên riêng biệt cho cá nhân hoặc tổ đội (Private/Party Spatial Instance). Gồm các ải thử thách thực chiến, giải mã cơ quan trận pháp, đánh bại Thủ Hộ Thú và kích hoạt Đài Truyền Thừa (Inheritance Altar) để kế thừa Cổ trùng độc bản, bí kíp, phương thuốc hợp luyện và di sản.
_Avoid_: Bí cảnh đại trà, mộng cảnh

### Đài Truyền Thừa (Inheritance Altar)
Điểm phong ấn cơ quan lưu giữ di sản của tiền nhân trên bản đồ không gian, yêu cầu đánh bại Thủ Hộ Giả (Guardian Boss) và thỏa mãn các điều kiện về Đạo Ngân lưu phái hoặc cảnh giới để lĩnh nhận. Di sản Tiên Cổ/Bậc cao (Rank 6+) là độc bản vĩnh viễn (chỉ nhận một lần trong đời); di sản phàm trần (Rank 1–5) có thể hỗ trợ chu kỳ làm mới tùy cấu hình khảo nghiệm.

### Quy Tắc Bí Cảnh (Spatial Realm Rules / Realm Rules)
Tập luật môi trường ràng buộc hành vi của mọi Cổ sư khi hiện diện bên trong một Bí cảnh hoặc Vùng truyền thừa. Bao gồm: Thời lượng đếm ngược (Duration Timer), Cơ chế PK (Vùng An Toàn / Hỗn Chiến Tự Do), Giới hạn cảnh giới tối đa & Áp chế tu vi (Rank Suppression), Quy tắc trừng phạt tử vong (trục xuất về thế giới hoặc hồi sinh tại cổng), và Cấm đoán truyền tống trận.

### Map Trận Đấu Đấu Trường (Arena Match Map)
Map realtime riêng cho một trận Đấu Trường, chỉ được chia sẻ bởi những người trong trận. Mỗi trận cách ly trạng thái và tài nguyên combat; ngắt kết nối có thời gian quay lại 30 giây trước khi bị xử thua.

### Map Tập Luyện (Training Map)
Map realtime riêng tư cho một phiên tập luyện, dùng tài nguyên combat tạm thời và có thể thoát hoặc đặt lại bất cứ lúc nào mà không ảnh hưởng trạng thái nhân vật thật.


## Nguyên Tuyền (Primeval Vein)

Điểm tài nguyên động trên bản đồ Ngũ Vực, tự spawn/hết hạn và tích lũy nguyên thạch (hoặc tiên nguyên thạch với Tiên Tuyền) vào một bể chung. Ai đứng đúng ô cũng thu hoạch được; thu xong bể reset. Mỏ có trần tích lũy 1 ngày sản lượng và trần sản lượng vòng đời theo phẩm chất.
_Avoid_: Mine slot, mỏ cá nhân, mỏ cố định

## Trạng Thái Người Chơi & Di Chuyển (Player State & Movement)

### Ảnh Chụp Người Chơi (Player Snapshot)
Bản đọc nhất quán của hàng `players` cùng các tập con được chọn (toàn bộ Gu, Gu đang trang bị, vật phẩm, đạo ngân) và revision bộ sưu tập; thứ tự cột, cách gắn tập con và lỗi không tìm thấy chỉ được định nghĩa ở một module.
_Avoid_: player DTO, player object

### Chuỗi Di Chuyển (Movement Chain)
Chuỗi kiểm tra và ghi đích dùng chung cho mọi cách người chơi đổi ô: quyền điều khiển realtime → biên giới → sống/chết → chiến đấu → trưởng tổ đội → chính sách vùng → đích → áp chế giới vực → ghi vị trí → kéo tổ đội → phát sự kiện.
_Avoid_: move handler, teleport logic

## Realtime Combat & Techniques (Chiến Đấu Thời Gian Thực & Chiêu Thức)

### Chiến Đấu Thời Gian Thực (Realtime Combat)
Mô hình chiến đấu trong đó người chơi trực tiếp điều khiển di chuyển và lựa chọn chiêu thức, còn đồng đội và đối thủ do AI hành động; kết quả được server quyết định liên tục theo thời gian thay vì theo lượt.
_Avoid_: auto-combat mô phỏng theo lượt

### Hiệu Ứng Chiến Đấu Realtime (Realtime Combat Effect)
Hiệu ứng chiến đấu cần có hành vi, mục tiêu và thời lượng realtime được định nghĩa rõ; số lượt không tự quy đổi thành giây. Hiệu ứng chưa có hành vi realtime chưa thể dùng trong chiến đấu realtime.

### Lướt (Dash)
Chiêu thức di chuyển nhân vật một quãng theo hướng đã khóa khi bắt đầu thi triển. Lướt thuần di chuyển có thể xuyên qua thực thể; lướt va chạm dừng ở kẻ địch đầu tiên và gây sát thương, còn địa hình chặn đường. ROOT/BIND ngăn displacement bắt đầu nhưng không dừng cú lướt đã khởi động; SLOW không làm chậm cú lướt đang chạy, còn hard CC chặn cast và ngắt cú lướt.
_Avoid_: tăng tốc, chạy nhanh

### Tăng Tốc (Haste)
Trạng thái tích cực tạm thời làm tăng tốc độ di chuyển nền. Thi triển lại cùng trạng thái làm mới thời hạn mà không cộng dồn cường độ; nhiều trạng thái tăng tốc chỉ lấy hệ số cao nhất và vẫn chịu ảnh hưởng của làm chậm, trói chân cùng trần di chuyển chung.
_Avoid_: lướt, displacement

### Dạng Phòng Thủ Tiên Cổ (Defensive Immortal Gu Form)
Trạng thái biến thân do Tiên Cổ phòng thủ kích hoạt, có thời gian biến thân riêng theo từng Cổ; khi hoàn tất, trạng thái phòng thủ có hiệu lực và tiêu hao mana liên tục cho đến khi tắt hoặc hết mana.

### Gián Đoạn Biến Thân (Transformation Interrupt)
Khống chế cứng có thể hủy quá trình biến thân trước khi hoàn tất; sát thương thường không hủy thi triển. Phòng thủ của Tiên Cổ chỉ có hiệu lực sau khi biến thân hoàn tất.

### Chiêu Thức Dạng Biến Thân (Form Skill)
Chiêu thức riêng của một dạng biến thân, chỉ có thể dùng khi trạng thái biến thân tương ứng còn hoạt động.

### Biến Thân Hoàn Toàn (Full Metamorphosis)
Trạng thái biến toàn bộ cơ thể thành thú hoặc ma thần (thường qua Sát Chiêu Biến Hóa Đạo), thay đổi hitbox, chỉ số thuộc tính và hoán đổi thanh kỹ năng chiến đấu sang bộ chiêu thức riêng của hình thái đó.
_Avoid_: shape-shift buff, avatar swap đơn thuần

### Biến Thân Bộ Phận (Partial Transformation)
Trạng thái biến đổi một phần cơ thể (như móng, cánh, vảy qua Cổ trùng biến hóa), duy trì bộ kỹ năng thông thường kèm theo chỉ số tăng cường và mở khóa hoặc cường hóa chiêu thức đặc thù tương ứng.
_Avoid_: morph buff

### Dung Nhập Tiên Cổ Phòng (Gu House Integration)
Hành vi người chơi dung nhập vào Tiên Cổ Phòng đang trang bị, chuyển quyền kiểm soát chiến đấu sang Tiên Cổ Phòng; độ bền của Cổ Phòng nhận 100% sát thương thay nhục thân cho đến khi vỡ phòng, tiêu hao Tiên Nguyên duy trì tỷ lệ nghịch với độ chênh lệch cảnh giới giữa Cổ Tiên và Rank Cổ Phòng, và kích hoạt bộ Trận Kích chuyên biệt.
_Avoid_: vehicle mount, robot pilot

### Trận Kích Toàn Diện (Full House Strike Set)
Hệ thống 6 chiêu thức chiến đấu chuẩn hóa của Tiên Cổ Phòng khi người chơi dung nhập, bao gồm đầy đủ các vai trò: Pháo kích công kích, Xung kích di chuyển, Đại trận phòng hộ, Hồi phục sửa chữa, Thần thông hủy diệt tối hậu, và Lệnh rời khỏi Cổ Phòng.
_Avoid_: generic killer moves

### Hao Hụt Duy Trì Biến Thân (Sustained Form Upkeep)
Cơ chế tiêu hao Chân Nguyên hoặc Tiên Nguyên liên tục theo nhịp realtime (tick) để duy trì hình thái biến thân, tự động giải trừ khi cạn kiệt năng lượng hoặc bị phá vỡ hình thái.
_Avoid_: mana drain debuff
