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
Hành vi dẫn kênh thi triển Nô Thú Cổ lên quái hoang còn sống trong realtime để bắt thú thành Actor Instance; HP thấp hoặc khống chế làm tăng xác suất nhưng không bắt buộc để bắt đầu.
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
Quy tắc gameplay mô tả sự kiện kích hoạt, điều kiện áp dụng, mục tiêu và những thay đổi trạng thái được tạo ra. Chi phí, hồi chiêu, thời gian tụ lực và cách kích hoạt thuộc hồ sơ sử dụng của từng sản phẩm hoặc ngữ cảnh, không thuộc bản thân effect.
_Avoid_: Kỹ năng chiến đấu mặc định, hồ sơ thi triển

### Suy Hao Cảnh Giới Tự Thân (Caster Rank Attenuation)
Cơ chế suy giảm uy lực và thời lượng của hiệu ứng buff/bảo hộ khi Cổ sư cấp cao sử dụng Cổ trùng có Rank thấp hơn cảnh giới của bản thân áp lên chính mình (ví dụ Cổ Rank 4 tạo hiệu ứng Bất Tử cho Cổ Tiên Rank 9). Hiệu quả bảo vệ suy giảm mạnh theo hàm mũ hoặc chuyển đổi thành giá trị hữu hạn để tránh phá vỡ cân bằng cảnh giới.

### Áp Chế Cảnh Giới Mục Tiêu (Target Realm Suppression)
Cơ chế bảo hộ và kháng cự tự nhiên của mục tiêu cấp cao trước các hiệu ứng bất lợi (debuff, khống chế, sát thương theo % máu) từ Cổ trùng cấp thấp. Tỉ lệ kích hoạt thành công (Proc Chance) và thời gian duy trì (Duration) của debuff bị suy giảm theo độ chênh lệch giữa Rank Cổ và Rank Mục Tiêu (`TargetRank - GuRank`).

### Xác Suất Kích Hoạt Động (Dynamic Proc Chance)
Tỉ lệ kích hoạt thực tế của một hiệu ứng trong chiến đấu, được suy ra từ Xác suất cơ bản (`BaseChance` theo phân loại độ mạnh của hiệu ứng), cộng dồn tăng tiến theo Rank Cổ trùng (`+delta * (GuRank - 1)`), và điều chỉnh sau cùng thông qua Áp Chế Cảnh Giới, Kháng Tính Tâm Thần (Mental Resistance) và Vận Số (Luck).

## Universal Dao Conflict & Multi-Path Impurity (Vạn Đạo Tương Xung & Trừng Phạt Đa Tu)

### Khuếch Đại Đạo Ngân Cơ Sở (Base Dao Mark Amplification)
Hệ số thô do đạo ngân cùng lưu phái cung cấp cho một hành động, trước khi áp dụng hiệu suất Chân Ý và xung đột đạo ngân.
_Avoid_: Hệ số Đạo Ngân Hiệu Quả, tổng đạo ngân

### Hệ Số Đạo Ngân Hiệu Quả (Effective Dao Mark Multiplier)
Hệ số cuối cùng mà hành động nhận từ khuếch đại cơ sở sau khi Chân Ý và xung đột của lưu phái hành động được tính đến.
_Avoid_: Khuếch Đại Đạo Ngân Cơ Sở, tổng đạo ngân

### Ma Trận Tương Thích Lưu Phái (Path Compatibility Matrix)
Quan hệ tương thích được định nghĩa cho từng cặp lưu phái, dùng để biểu diễn mức độ đạo ngân của phái này ma sát với hành động thuộc phái kia. Các nhóm phái cung cấp giá trị mặc định; cặp đặc thù có thể ghi đè.
_Avoid_: Tương khắc nguyên tác cố định

### Đạo Ngân Ngoại Phái Đang Hoạt Động (Active Foreign Dao Marks)
Đạo ngân thuộc phái khác với path đang được đối chiếu (path hành động hoặc hit) và vượt ngưỡng dung nạp, nên được đưa vào tính xung đột tương ứng.
_Avoid_: Tổng đạo ngân, đạo ngân chủ phái

### Tương Tác Đạo Ngân Trên Mục Tiêu (Target-Side Dao Mark Interaction)
Quan hệ giữa path của hit và từng nhóm đạo ngân trên người trúng; đạo ngân xung đột làm suy yếu hit, còn đạo ngân đồng phái có thể khuếch đại sát thương nhận.
_Avoid_: Chỉ kháng từ đạo ngân ngoại phái

### Cộng Hưởng Đạo Ngân Đồng Phái (Same-Path Dao Resonance)
Hiệu ứng khuếch đại hit khi path của hit trùng với đạo ngân trên mục tiêu; nó tách biệt với khuếch đại do đạo ngân của người xuất chiêu.
_Avoid_: Kháng tự nhiên, xung đột nội tại

### Đạo Thương Ngoại Phái (Foreign Dao Wound)
Vết thương combat do một hit mang path xung đột gây ra, làm suy giảm khả năng tiếp nhận trị liệu trong thời gian hiệu lực.
_Avoid_: Trọng Thương, giảm trị liệu chung

### Hệ Số Xung Đột Đạo Ngân (Dao Conflict Multiplier)
Hệ số chung mô tả mức uy lực còn lại của hành động sau ma sát giữa lưu phái thi triển và đạo ngân ngoại phái đang hoạt động; cùng mức xung đột cũng điều chỉnh chi phí và khả năng bất ổn theo ngưỡng riêng của từng hệ quả.
_Avoid_: Độ thuần khiết, kháng Đạo Ngân

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

### Phiên Đấu Trường (Arena Match)
Trận đấu PvP giải quyết theo thể thức theo lượt (Turn-based), hai bên sử dụng đội hình, sát chiêu và Cổ trùng đã trang bị để tranh tài.

### Phiên Tập Luyện (Training Session)
Phiên đấu tập turn-based với hình nhân (dummy), dùng tài nguyên combat tạm thời để thử nghiệm sát chiêu mà không ảnh hưởng trạng thái nhân vật thật.


## Nguyên Tuyền (Primeval Vein)

Điểm tài nguyên động trên bản đồ Ngũ Vực, tự spawn/hết hạn và tích lũy nguyên thạch (hoặc tiên nguyên thạch với Tiên Tuyền) vào một bể chung. Ai đứng đúng ô cũng thu hoạch được; thu xong bể reset. Mỏ có trần tích lũy 1 ngày sản lượng và trần sản lượng vòng đời theo phẩm chất.
_Avoid_: Mine slot, mỏ cá nhân, mỏ cố định

## Trạng Thái Người Chơi & Di Chuyển (Player State & Movement)

### Ảnh Chụp Người Chơi (Player Snapshot)
Bản đọc nhất quán của hàng `players` cùng các tập con được chọn (toàn bộ Gu, Gu đang trang bị, vật phẩm, đạo ngân) và revision bộ sưu tập; thứ tự cột, cách gắn tập con và lỗi không tìm thấy chỉ được định nghĩa ở một module.
_Avoid_: player DTO, player object

### Chuỗi Di Chuyển (Movement Chain / Spatial Transition)
Module chuyển dịch không gian nguyên tử và nhất quán dùng chung cho mọi cách người chơi đổi ô: biên giới → sống/chết → kiểm tra chiến đấu → trưởng tổ đội → chính sách vùng → đích → áp chế giới vực → cập nhật vị trí đồng bộ (`players`, `world_location_revision`) → kéo tổ đội → phát sự kiện post-commit.
_Avoid_: move handler, teleport logic, ad-hoc coordinate update

### Chuỗi Đột Biến Người Chơi (Player Mutation Pipeline)
Module thực thi các đột biến trạng thái người chơi nguyên tử và nhất quán dùng chung cho mọi hành vi gameplay làm thay đổi chỉ số hoặc tài sản nhân vật: mở transaction `sql.Tx` & khóa dòng `FOR UPDATE` → chiếu nạp Player Snapshot → tính toán thụ động (mana / ôn dưỡng độ bền) → thực thi closure đột biến nghiệp vụ → cập nhật đầy đủ các cột khả biến với bước tăng đơn điệu `version` và `state_revision` → commit nguyên tử → phát sự kiện đồng bộ post-commit qua Redis / SSE.
_Avoid_: ad-hoc SQL update, unversioned mutation, manual snapshot broadcast

## Strategic Combat & Turn-Based Engine (Chiến Đấu Theo Lượt & Mô Phỏng Chiến Thuật)

### Chiến Đấu Theo Lượt (Turn-Based Combat)
Mô hình chiến đấu tất định (deterministic) theo lượt tính toán từ phía máy chủ (`backend/internal/simulator/engine.go`), dựa trên snapshot chiến đấu của các bên tham gia (Cổ sư, Hung thú, Phân thân Đạo phái), tính toán hiệu ứng kích hoạt của Cổ trùng, Sát chiêu, Đạo ngân và Buff/Debuff theo từng hiệp. Người chơi có thể lựa chọn điều khiển chiêu thức / theo dõi diễn biến trận chiến qua `CombatScreen.tsx` hoặc Quét trận tức thì (Fast Auto-resolve) qua `combat_auto_resolve.go`.
_Avoid_: 20Hz continuous tick, physics hitbox

### Hiệu Ứng Chiến Đấu Theo Lượt (Turn-Based Combat Effect)
Hiệu ứng chiến đấu diễn ra và tính toán theo số hiệp (turns) hoặc thời điểm kích hoạt (`ON_ACTION`, `ON_HIT`, `ON_TURN_START`, `ON_DEATH`), tương tác trực tiếp với Chân Nguyên, Sinh Lực, Giáp Ảo và Đạo Ngân lưu phái.
_Avoid_: thời lượng giây liên tục, continuous damage tick

### Sát Chiêu Hợp Thành (Composite Killer Move)
Hành động chiến đấu do Cổ sư tạo từ một Cổ chính và các Cổ thành phần; một sát chiêu có thể kết hợp nhiều thao tác hiệu ứng do người chơi chọn, còn Cổ chính định danh sát chiêu và lưu phái chủ đạo.
_Avoid_: effect đơn lẻ, chiêu tự ghép không chọn lọc

### Hệ Số ATK Người Chơi của Cổ (Gu Player-ATK Coefficient)
Trọng số riêng của một Cổ nhân với ATK người chơi để tính phần sát thương đóng góp vào Sát Chiêu Hợp Thành. Hệ số thuộc cấu hình của từng Cổ, không được suy ra từ Path và không đổi theo rank.
_Avoid_: hệ số do người chơi nhập khi ghép chiêu, hệ số chung của Path

### Thám Hiểm Lưới Chiến Thuật (Tactical Grid Exploration)
Mô hình di chuyển và khám phá thế giới từng bước theo tọa độ ô vuông nguyên bản ($x, y$ số nguyên) kế thừa từ `map_cells`. Mỗi bước đi tiêu hao Chân Nguyên / Thể Lực, giải tỏa sương mù chiến tranh dựa trên chỉ số Thám Tra (`investigation`), phát hiện tài nguyên ẩn, mỏ nguyên tuyền và kích hoạt trận chiến theo lượt khi bước vào lãnh địa quái vật.
_Avoid_: continuous float coordinates, WASD vector movement, 20Hz zone sim

### Lướt (Dash)
Chiêu thức di chuyển nhân vật một quãng theo hướng đã khóa khi bắt đầu thi triển. Lướt thuần di chuyển có thể xuyên qua thực thể; lướt va chạm dừng ở kẻ địch đầu tiên và gây sát thương, còn địa hình chặn đường. ROOT/BIND ngăn displacement bắt đầu nhưng không dừng cú lướt đã khởi động; SLOW không làm chậm cú lướt đang chạy, còn hard CC chặn cast và ngắt cú lướt. Không thể bắt đầu cú lướt thứ hai trước khi cú hiện tại kết thúc; Tăng Tốc không đổi cú lướt đang chạy.
_Avoid_: tăng tốc, chạy nhanh

### Đẩy Lùi (Knockback)
Xung lực dịch chuyển mục tiêu theo đường thẳng ra xa nguồn phát lực, sử dụng cùng thuật toán quét bước va chạm (sweep substepping & bisection) với Lướt, dừng lại an toàn tại mép chướng ngại vật thay vì dịch chuyển tức thời hay xuyên qua tường mỏng.
_Avoid_: instant teleport, blink

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

### Vỡ Phòng Tiên Cổ Phòng (Gu House Structural Breach)
Trạng thái sụp đổ hoàn toàn kết cấu trận pháp của Tiên Cổ Phòng khi Form HP cạn kiệt trong giao tranh; Tiên Cổ Phòng hấp thu 100% đòn đánh mà không tràn sát thương sang nhục thân, nhưng kích phát phản phệ trận pháp làm Cổ Tiên bị văng ra chịu Choáng 3 giây cùng tổn thương 30% sinh mệnh tối đa, độ bền Cổ Phòng về mức 0% buộc tháo trang bị, và truyền 30% xung kích chấn thương làm suy giảm độ bền của toàn bộ Tiên Cổ hạch tâm.
_Avoid_: armor break debuff, vehicle wreck

### Hao Hụt Duy Trì Biến Thân (Sustained Form Upkeep)
Cơ chế tiêu hao Chân Nguyên hoặc Tiên Nguyên liên tục theo nhịp realtime (tick) để duy trì hình thái biến thân, tự động giải trừ khi cạn kiệt năng lượng hoặc bị phá vỡ hình thái.
_Avoid_: mana drain debuff

## Gu Will Subjugation & Durability (Luyện Hóa Ý Chí & Độ Bền Cổ Trùng)

### Luyện Hóa Ý Chí (Gu Will Subjugation)
Hành vi Cổ sư vận chuyển Chân Nguyên / Tiên Nguyên để thanh tẩy ý chí hoang dã hoặc ý chí ngoại lai bên trong Cổ trùng, in dấu ấn tâm niệm của bản thân. Chỉ khi tiến độ luyện hóa đạt mức toàn vẹn (100%), Cổ trùng mới thần phục hoàn toàn và cho phép trang bị vào Không Khiếu.
_Avoid_: Gu Crafting, Hợp Luyện Cổ

### Áp Chế Bản Mệnh Cổ (Vital Gu Will Suppression)
Quy luật uy áp đẳng cấp trong Cổ Chân Nhân: khi Cổ sư sở hữu Cổ Bản Mệnh có phẩm giai cao hơn phẩm giai Cổ cần luyện từ 2 Rank trở lên, khí tức bản mệnh lập tức trấn áp và bức lui hoàn toàn ý chí dã Cổ, hoàn tất luyện hóa tức khắc mà không cần tiêu hao Chân Nguyên theo đợt.
_Avoid_: Instant tame cheat

### Suy Thoái Ý Chí Hoang Dã (Wild Will Resurgence)
Hiện tượng phản phệ tự nhiên xảy ra khi Cổ trùng chưa được luyện hóa hoàn tất (tiến độ dưới 100%) bị bỏ quên trong thời gian dài: ý chí hoang dã trỗi dậy dần dần làm tiêu biến tiến độ luyện hóa đã đạt được, tính toán theo cơ chế lazy evaluation dựa trên thời gian trôi qua.
_Avoid_: Progress decay debuff

### Độ Bền Cổ Trùng (Gu Durability & Integrity)
Chỉ số toàn vẹn thể chất và sinh mệnh của Cổ trùng (thang đo 0-100). Bị hao tổn do thi triển kỹ năng cường độ cao, chịu phản chấn giao tranh, chấn động khi Tiên Cổ Phòng nhận sát thương, hoặc khi Cổ sư bị tử vong. Cổ trùng cạn kiệt độ bền sẽ mất khả năng thi triển và rơi vào trạng thái trọng thương.
_Avoid_: Equipment durability đơn thuần

### Nội Dưỡng Tiêu Hao Độ No (Healing Nourishment Burn)
Cơ chế tự hồi phục độ bền tự nhiên khi Cổ trùng bị thương nhẹ (độ bền từ 20 trở lên) được nuôi dưỡng trong Không Khiếu không lâm trận; Cổ trùng tăng cường đốt độ no (thức ăn) với tốc độ cao hơn bình thường để tái tạo thương tổn, đánh đổi độ no lấy độ bền.
_Avoid_: Passive HP regen

### Tu Bổ Cổ Trùng (Gu Restoration & Mending)
Hành động chủ động dùng Chân Nguyên ôn dưỡng (với chấn thương nhẹ) hoặc phối hợp Dược Thảo, Tiên Nguyên Thạch, nguyên liệu luyện Cổ đồng hệ (khi chấn thương nặng hoặc độ bền dưới 20) để chữa lành kết cấu Cổ trùng, khôi phục linh tính thi triển.
_Avoid_: Blacksmith repair

## Tiered Gu Storage & Aperture Gu Swarms (Kiến Trúc Phân Tầng Cổ Trùng & Đàn Cổ Tiên Khiếu)

### Đàn Phàm Cổ Tiên Khiếu (Immortal Aperture Gu Swarm)
Mô hình lưu trữ xếp chồng số lượng (Stackable/Fungible) dành riêng cho Phàm Cổ (Rank 1–5) cư ngụ trong Tiên Khiếu (Phúc Địa/Động Thiên) của Cổ Tiên (Rank 6+). Phàm Cổ trong Đàn Cổ được hệ sinh thái Tiên Khiếu tự động chu cấp dưỡng khí và thức ăn, không chịu cơ chế suy giảm độ đói cá thể, cho phép tích trữ hàng vạn con phục vụ Hợp Luyện, Sát Chiêu và xây dựng Tiên Cổ Phòng. Tiên Cổ (Rank 6–9) và Cổ đang trang bị (Equipped) vĩnh viễn không thể nhập đàn.
_Avoid_: Generic item stack, boundless mortal bag, single giant instance table

### Chuyển Hóa Cá Thể Cổ Trùng (Gu Materialization & Dematerialization)
Vòng đời chuyển đổi hai chiều giữa Đàn Cổ Tiên Khiếu (Stackable) và Cổ Cá Thể (Instance trong `player_gu`):
- **Cụ Thể Hóa (Materialization)**: Rút 1 con Phàm Cổ từ Đàn Cổ để tạo thành 1 thực thể Cổ cá thể đầy đủ sinh mệnh (100% hunger, 100% durability, state ACTIVE) để trang bị vào Không Khiếu hoặc cài làm Bản Mệnh Cổ.
- **Hòa Nhập Đàn Cổ (Dematerialization)**: Giải phóng một Phàm Cổ đang trang bị khi tháo ra trở về Tiên Khiếu; chỉ chấp nhận Cổ hoàn toàn lành lặn (100% durability, không bị khóa, không phải Bản Mệnh) để xóa bản ghi cá thể và cộng dồn số lượng vào Đàn Cổ. Cổ bị thương duy trì dạng cá thể để chữa trị.
_Avoid_: Lossy item conversion, unequip item drop

### Sức Chứa Không Khiếu Phàm Nhân (Mortal Aperture Capacity)
Hạn mức dung nạp Cổ trùng tối đa của Không Khiếu Cổ Sư giai đoạn Phàm trần (Rank 1–5), tăng dần theo cảnh giới tiểu cảnh giới và đại cảnh giới. Phản ánh áp lực chân nguyên và gánh nặng thức ăn nuôi dưỡng Cổ trùng của người phàm; đồng thời ngăn chặn triệt để hành vi tích trữ bùng nổ dữ liệu trước khi đắc đạo Thăng Tiên.
_Avoid_: Infinite inventory, weight limit RPG thông thường

### Tiêu Thụ Hàng Loạt Từ Đàn (Direct Gu Swarm Batch Consumption)
Cơ chế cho phép các nghi thức Hợp Luyện Tiên giai và kích hoạt Sát Chiêu quy mô lớn (như Vạn Kiếm Quy Tông, Phong Nhận Loạn Vũ) khấu trừ trực tiếp số lượng Phàm Cổ cần thiết từ Đàn Cổ Tiên Khiếu qua một thao tác trừ số học nguyên tử, không yêu cầu phân giải hay hủy diệt từng mã định danh cá thể.
_Avoid_: Looped single gu deletion, N+1 query consumption


## Observability, Tracing & Diagnostics (Hệ Thống Quan Sát, Truy Vết & Chẩn Đoán)

### Khối Thu Nạp Lỗi (Error Intake Module)
Module sâu nằm tại seam của HTTP handler, chịu trách nhiệm chuẩn hóa phân loại lỗi (domain error vs system error), ghi nhận c.Error(err) cho middleware, phát log có cấu trúc mang ngữ cảnh đến Zap, và trả về phản hồi an toàn cho client kèm mã Trace ID. Ngăn chặn triệt để việc nuốt lỗi hoặc lộ cấu trúc DB nội bộ ra ngoài.
_Avoid_: Ad-hoc c.JSON(500), generic raw string error

### Mã Truy Vết Ngữ Cảnh (Contextual Trace Identifier / Trace ID)
Chuỗi định danh duy nhất cho một chu trình xử lý (HTTP request, WebTransport frame, worker tick), được tạo lập ngay tại intake seam, tiêm vào Go context.Context, đính kèm vào header phản hồi X-Trace-ID (được expose qua CORS), truyền tải đến client toast và lưu trong bộ nhớ đệm chẩn đoán của server. Cho phép liên kết 1-1 giữa thông báo lỗi trên UI của người chơi với log chi tiết và trace trên bảng điều khiển Quản trị.
_Avoid_: Client-side generated random error, untraceable exception

## Immortal Aperture Ecosystem & Co-op Tribulation (Hệ Sinh Thái Tiên Khiếu & Trợ Trận Độ Kiếp)

### Môn Hộ Tiên Khiếu (Aperture World Gateway)
Thực thể cổng không gian vật lý duy nhất xuất hiện trên bản đồ thế giới Ngũ Vực tại tọa độ `(X, Y, Region)` khi Cổ Tiên thực hiện Hạ Khiếu (`AnchorAperture`). Cổ Tiên và Tiên Khiếu là nhất thể không thể tách rời: khi không hạ khiếu, Tiên Khiếu ngự trị trong thân thể Cổ Tiên (không có mặt trên bản đồ thế giới); khi Hạ Khiếu cắm rễ vào địa mạch, Cổ Tiên cư ngụ BÊN TRONG Tiên Khiếu để tọa trấn và không thể xuất hiện hay di chuyển tự do ngoài thế giới thực. Môn Hộ là lối liên kết duy nhất cho phép ngoại nhân bái phỏng theo các chính sách kiểm soát (`OPEN`, `PARTY`, `SECT`, `KEY`, `HIDDEN`). Muốn trở lại thế giới thực chu du, Cổ Tiên bắt buộc phải Thu Khiếu (`UnanchorAperture`).
_Avoid_: Private UI teleport, instant invisible portal, phân thân đứng ngoài thế giới canh khiếu

### Tiên Khiếu Giới Vực (Aperture Domain Instance)
Không gian tiểu thế giới độc lập thuộc quyền sở hữu của Cổ Tiên trong Zone Sim, có hệ thống thời gian sông Quang Âm riêng và lưới tài nguyên sinh thái. Khách được cấp quyền bước vào thông qua Môn Hộ chịu sự giám sát của ma trận phân quyền (`GUEST_ROLE`: `VISITOR`, `ASSISTANT`, `PRISONER`).
_Avoid_: Generic dungeon map, public open zone

### Áp Lực Đạo Ngân Tiên Khiếu (Aperture Dao Stress)
Ứng lực ma sát và tải trọng quy tắc đè nặng lên vách ngăn Tiên Khiếu khi dung nạp một Cổ Tiên ngoại lai có phẩm cấp Rank hoặc mật độ Đạo Ngân vượt trội hơn phẩm cấp Tiên Khiếu bản địa. Nếu không thi triển sát chiêu thu liễm Đạo Ngân, áp lực này sẽ gây rạn nứt cấu trúc và suy giảm sinh lực Tiên Khiếu (`aperture_hp`).
_Avoid_: Generic level restriction, soft level cap

### Trợ Trận Độ Kiếp (Co-op Tribulation Assistance)
Hành vi mời gọi hoặc cho phép các Cổ Tiên khác cùng hiện diện trong Tiên Khiếu Giới Vực để hiệp đồng tác chiến chống lại tai kiếp (Địa Tai, Thiên Kiếp, Hạo Kiếp) khi đợt độ kiếp được kích hoạt.
_Avoid_: Raid party thông thường, public world boss

### Thiên Đạo Thích Ứng & Biến Dị (Reactive Heaven's Will Mutation)
Cơ chế tự động tăng cường độ nguy hiểm của Thiên Đạo khi phát hiện nhiều Cổ Tiên cùng độ kiếp trong một Tiên Khiếu: nhân hệ số HP/Công/Thủ của quái tai kiếp theo tổng thực lực tổ đội, đồng thời kích phát đợt kiếp nạn hỗn loạn (`Chaos Calamity Wave`) mang Đạo Ngân tương khắc với lưu phái của người trợ trận.
_Avoid_: Static boss scaling, flat damage buff

### Duy Ngã Độc Tôn (Aperture Lord Sovereignty)
Quyền uy tối thượng và bùa lợi toàn diện của Chủ Khiếu khi giao tranh ngay bên trong Tiên Khiếu của chính mình: giảm tiêu hao Tiên Nguyên khi vận chuyển chiêu thức, tăng tốc độ hồi phục chân nguyên/tiên nguyên, gia tăng sát thương và sở hữu thẩm quyền trục xuất khách vi phạm giao kèo.
_Avoid_: Admin godmode, simple realm buff

### Phúc Địa Vô Chủ (Unowned Blessed Land)
Trạng thái tiểu thế giới còn sót lại sau khi Chủ Khiếu tử vong hoặc vẫn lạc trong lúc Hạ Khiếu: nhục thân tan rã, cổng Môn Hộ cố định vĩnh viễn ngoài thế giới, quyền quản lý chuyển giao cho Địa Linh chấp niệm bảo vệ di sản và đưa ra điều kiện thử thách để tìm kiếm chủ nhân mới.
_Avoid_: Expired instance, despawned zone


