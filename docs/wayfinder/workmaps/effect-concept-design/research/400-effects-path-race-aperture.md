# Research 400: Tiền lệ nguyên tác và ranh giới code cho effect lưu phái, chủng tộc, Tiên Khiếu

Type: research  
Status: findings captured; design decisions remain open  
Related: [Wayfinder #399](https://github.com/VipPro67/GuWorld/issues/399), [research ticket #400](https://github.com/VipPro67/GuWorld/issues/400)

## Phạm vi và cách đọc

Báo cáo ghi nhận những tiền lệ có thể đối chiếu với văn bản *Cổ Chân Nhân / Reverend Insanity* và code GuWorld tại thời điểm khảo sát. “Canon” dưới đây nghĩa là nội dung được kể/giải thích trong các chương được dẫn, không khẳng định đó là quy luật áp dụng đồng nhất cho mọi tình huống. “Diễn giải” là khái quát cơ chế để thảo luận thiết kế. “Còn mở” chỉ những luật mà nguồn không xác lập hoặc code chưa có owner/consumer tương ứng. Link chapter dẫn bản tiếng Anh được WebNovel lưu trữ; số chương là định danh để tra lại trong bản dịch.

## Kết quả chính

### 1. Đạo Ngân, lưu phái và độ tương thích effect

**Canon.** Đạo Ngân khuếch đại effect của Cổ trùng/sát chiêu cùng lưu phái; Đạo Ngân khác lưu phái có thể xung khắc và làm yếu effect. Chương 1033 nêu ví dụ Cổ Hỏa Đạo bị người có nhiều Đạo Ngân Thủy Đạo làm yếu, và nói Cổ Tiên thường chuyên một lưu phái. Chương 1102 nêu Đạo Ngân các lưu phái kháng/cản trở lẫn nhau, đồng thời cho ví dụ một vật mang Đạo Ngân Thông Tin có thể đặt trên vật chứa Đạo Ngân Kiếm để tránh tương tác trực tiếp. [Chương 1033](https://www.webnovel.com/ru/book/32605619408374705/88383382802714704) · [Chương 1102](https://www.webnovel.com/book/reverend-insanity%28english%29_32605619408374705/chapter-1102-planning_88427397761781506)

**Canon.** Có trường hợp ngoại lệ đặc biệt: Chí Tôn Tiên Thai Cổ tạo thân thể mà Đạo Ngân không xung đột, cho phép dùng effect nhiều lưu phái mà không bị suy giảm theo luật thông thường (chương 1033). Đây là đặc tính riêng được nêu cho thể chất đó, không phải mặc định cho mọi người chơi đa lưu phái. [Chương 1033](https://www.webnovel.com/ru/book/32605619408374705/88383382802714704)

**Diễn giải.** Nên xem “đạo ngân/độ tương hợp” là bộ modifier phụ thuộc ngữ cảnh và cặp lưu phái, chứ không đồng nhất với một bonus phẳng do `path_masteries` hoặc chủng tộc cấp. Nguồn hỗ trợ sự khuếch đại và xung khắc; không hỗ trợ một ma trận số học tổng quát cố định cho mọi hai lưu phái.

**Code hiện có.** `backend/internal/gamefx/dao_combat_profile.go` chứa 40 đường lưu phái chuẩn, nhóm đường, ma trận tương thích mặc định và các override cặp đường. `BuildDaoCombatProfile`/`BuildDaoCombatProfileWithReduction` tạo profile cho simulator; các call sites có thể thấy ở `backend/internal/handlers/combat_effects.go` và `backend/internal/handlers/aperture_collect.go`. Profile này đang mô hình hóa tương thích/xung đột cho combat và một số phép tính Tiên Khiếu. Nó là luật game hiện hành, không phải số liệu được suy ra từ nguyên tác.

**Còn mở.** Cần quyết định effect nào đọc profile này ngoài các call sites hiện tại, cách tương tác với effect theo đường/chủng tộc, và liệu ngoại lệ “không xung đột” có thể áp dụng ở đâu. Không có căn cứ để nâng các giá trị trong `groupCompatibilityMatrix` hay `pairOverrides` thành canon.

### 2. Thể chất và chủng tộc không phải cùng một trục

**Canon.** Mười Tuyệt Thể là các thiên phú/căn cốt cực hạn gắn với aperture và Chân Nguyên; chương 135 mô tả Bắc Ám Băng Hồn Thể có 100% Chân Nguyên nhưng aperture chịu áp lực, tiềm năng chiến đấu cao đi cùng nguy cơ tự hủy. Chương 939 nhắc tới quan hệ giữa Tuyệt Thể, cấp bậc phúc địa và tai kiếp. Các ví dụ này cho thấy thể chất có thể tác động mạnh tới năng lực và rủi ro tu luyện, không chỉ là tên chủng tộc. [Chương 135](https://www.webnovel.com/book/reverend-insanity%28english%29_32605619408374705/chapter-135-northern-dark-ice-soul-ten-extreme-talents_87702418422294734) · [Chương 939](https://www.webnovel.com/book/reverend-insanity%28english%29_32605619408374705/chapter-939-dual-ten-extreme-physique-rank-nine-demon-venerable_88335917240085406)

**Canon.** Dị nhân có thể có tài nguyên/đặc tính riêng theo tộc; chương 1489 liệt kê tài nguyên đặc hữu của Tuyết Nhân, Thạch Nhân, Mao Dân và Vũ Dân trong bối cảnh phát triển Tiên Khiếu. Đây là ví dụ về khác biệt sinh thái/kinh tế; không khẳng định mọi thành viên chủng tộc đều có cùng chỉ số chiến đấu cố định. [Chương 1489](https://www.webnovel.com/pt/book/32605619408374705/90267048084567334)

**Diễn giải.** Nên giữ riêng các trục “chủng tộc”, “thể chất/constitution”, “Đạo Ngân/lưu phái”, “Cổ trùng và trạng thái trang bị”. Một chủng tộc có thể gợi ý affinity hoặc điều kiện sinh thái; nó không tự chứng minh mọi cá thể cùng tu một lưu phái hay có cùng bộ modifier.

**Code hiện có.** `backend/internal/gamefx/races.go` nạp `RaceDef` từ `race_dictionary`/`Race.csv`; trường `Effects` dùng cùng dạng effect có cấu trúc. `backend/internal/gamefx/fx.go` giải mã/tổng hợp các effect và có getter race; `Race.csv` hiện khai báo một số effect như `DAO_MARK_MODIFIER`, `STAT_MODS`, `HP_REGEN`, `REFINE_SUCCESS`. Constitution có registry riêng ở `backend/internal/gamefx/constitutions.go`. `players.race` và `players.constitution` là các trường riêng (`backend/internal/db/schema_players.go`). Đây là hợp đồng dữ liệu GuWorld; nội dung CSV hiện tại là quyết định gameplay, không phải canon.

**Còn mở.** Không có phép ánh xạ phổ quát trong nguồn từ từng chủng tộc sang một lưu phái, hay từ một chủng tộc sang một bộ stat bất biến. Cần chốt riêng quy tắc cộng dồn race/constitution/status/Gu và cách migration nếu phân tách định nghĩa effect.

### 3. Passive và effect ngoài combat

**Canon.** Nguyên tác có effect tồn tại ngoài một lượt đánh: Đạo Ngân có thể ảnh hưởng độ mạnh của Cổ trùng; Đạo Ngân từ tai kiếp có thể biến đổi môi trường và hệ sinh thái Tiên Khiếu. Chương 826 mô tả Đạo Ngân Nô Đạo tăng khả năng quản lý/sinh sản sinh vật trong Hu Tiên Phúc Địa, Đạo Ngân Thổ Đạo làm đất và môi trường phù hợp hơn, còn Đạo Ngân Huyết Đạo làm đổi loài thực vật và tài nguyên. [Chương 826](https://www.webnovel.com/id/book/reverend-insanity%28english%29_32605619408374705/chapter-826-new-dao-marks-deeper-foundation_88240709089412764)

**Diễn giải.** Effect cần được phân theo consumer/context (combat, tu luyện, luyện Cổ, scouting, aperture simulation/world action) ngoài trigger và operation. Một passive có giá trị ngoài combat chỉ hoạt động nếu subsystem tương ứng thực sự đọc nó; “catalog chấp nhận” không đồng nghĩa “mọi runtime đều thực thi”.

**Code hiện có.** Effect contract và parser ở `backend/internal/gamefx/fx.go`, manifest/capability ở `backend/internal/effectcatalog/`. `frontend/src/lib/wiki/effect-glossary.ts` cũng ghi rõ một số operation cần executor world riêng. Simulator dựng `DaoCombatProfile` trong `backend/internal/simulator/engine.go`; do đó profile combat là một consumer rõ ràng, không phải engine dùng chung cho mọi action. Các handler/worker riêng điều khiển việc tu luyện, scouting và Tiên Khiếu.

**Còn mở.** Chưa có bằng chứng rằng effect của registry có một dispatcher toàn cục bao phủ mọi hành động world. Mỗi trigger mới cần chỉ rõ state owner, consumer, thứ tự áp dụng, hết hạn, và phản hồi khi không có context thích hợp.

### 4. Dò tìm, che giấu và xâm nhập Tiên Khiếu

**Canon.** Tiên Khiếu là một tiểu thế giới; một phương pháp truy tung có thể mất hiệu lực khi đối tượng đi vào Tiên Khiếu của người khác, còn Đạo Ngân của khách và chủ khiếu có thể xung đột/tiêu hao nhau. Chương 1071 cũng nói việc cho người khác vào khiếu là hiếm vì tầm quan trọng và rủi ro; nếu chênh lệch lực/Đạo Ngân quá lớn, người vào có thể làm khiếu bất ổn hoặc nổ. [Chương 1071](https://www.webnovel.com/book/reverend-insanity%28english%29_32605619408374705/chapter-1071-selling-immortal-gu_88392910919233788)

**Canon.** Tiên Khiếu đã đặt ra ngoài có thể mở cửa để tiếp nhận thiên địa nhị khí; chương 1096 mô tả chủ khiếu chủ động mở cửa và dòng khí đi vào. Chương 1183 mô tả một khiếu/phúc địa ẩn trong không gian được phát hiện bằng sát chiêu dò tìm, sau đó dùng Thượng Cực Thiên Ưng nhận biết/cưỡng nhập cửa khiếu. Các ví dụ này cho thấy dò tìm, mở cửa và xâm nhập là các bước/công cụ riêng biệt, không phải một hành động duy nhất. [Chương 1096](https://www.webnovel.com/ru/book/32605619408374705/88407578769416845) · [Chương 1183](https://www.webnovel.com/ru/book/32605619408374705/88476344282832620)

**Canon.** Bản thân khiếu được đặt/ẩn ở một nơi có thể khó cảm nhận; chương 1605 nói điều đó không bất thường, đồng thời nêu các lưu phái/phương pháp có thiên hướng che giấu. Đây là phát biểu trong đối thoại về một trường hợp cụ thể, không phải quy tắc mọi Tiên Khiếu luôn vô hình. [Chương 1605](https://www.webnovel.com/ru/book/32605619408374705/90406963623550116)

**Diễn giải.** Một hệ thống gameplay có thể tách riêng: dấu vết/đầu mối ngoài thế giới, kiểm tra dò tìm, che giấu/đánh lạc hướng, xác định vị trí cửa khiếu, quyền mở cửa/được mời, kiểm tra xâm nhập, hậu quả đối với chủ/khách. Canon hỗ trợ các loại tương tác này nhưng không quy định tỷ lệ thành công, UI, quyền sở hữu game hay cách cân bằng đối kháng.

**Code hiện có.** Aperture state nằm trong `immortal_apertures` và `models.ImmortalAperture`; portal/guest roster ở `backend/internal/handlers/aperture_portal.go` và route `/api/immortal-aperture/...` trong `backend/internal/server/server.go` quản lý cổng, vào/ra, khách, vai trò và trục xuất. Đây là đường vào dựa trên portal/permission hiện hành. Scouting có `backend/internal/gamefx/scouting_engine.go`, handler liên quan và schema `player_tracking_marks`/`world_residual_traces` trong `backend/internal/db/schema_scouting.go`; dữ liệu dấu vết ghi vị trí, loại hành động, lưu phái trội, cường độ và thời điểm hết hạn. Aperture placement/state được lưu riêng trong `immortal_apertures` và các handler aperture. Các owner này không tạo sẵn một luật tìm kiếm hay đột nhập Tiên Khiếu.

**Còn mở.** Cần quyết định khi nào dấu vết lộ ra, ai nhìn thấy, độ chính xác vị trí, cách so sánh dò tìm/che giấu, cửa khiếu có thể bị tìm hoặc mở trong điều kiện nào, và hệ quả của xâm nhập trái phép. Không có căn cứ canon cho một hành động “tìm Tiên Khiếu” tự động bỏ qua portal/permission hiện có.

## Ranh giới thiết kế rút ra

| Chủ đề | Nguồn hỗ trợ | Không được coi là đã chốt |
| --- | --- | --- |
| Đạo Ngân và lưu phái | Khuếch đại cùng đường; xung khắc khác đường; ngoại lệ đặc biệt có tiền lệ | Hệ số code hiện tại là canon; một công thức xung đột áp dụng mọi context |
| Chủng tộc/thể chất | Dị nhân có khác biệt tài nguyên; một số thể chất có tác động aperture/tu luyện rõ rệt | Chủng tộc xác định lưu phái hoặc stat duy nhất cho mọi cá thể |
| Passive/world effects | Effect có thể tác động tu luyện, sinh thái và sản lượng ngoài combat | Effect contract hiện tại tự chạy ở mọi subsystem |
| Trinh sát Tiên Khiếu | Truy tung có thể bị ngắt/che; cửa vào và khiếu ẩn có tiền lệ; xâm nhập có rủi ro Đạo Ngân | Canon cung cấp luật PvP, quyền xâm nhập, tỉ lệ dò hoặc counterplay số |

Đây là đầu vào nghiên cứu cho các quyết định ở #399, không phải đặc tả implementation-ready hay đề xuất balance. Chưa chốt bất kỳ cơ chế gameplay mới nào.

## Mốc code đã đối chiếu

- `backend/internal/gamefx/dao_combat_profile.go` — canonical paths, compatibility matrix, `DaoCombatProfile`.
- `backend/internal/gamefx/fx.go`, `backend/internal/gamefx/races.go`, `backend/internal/gamefx/constitutions.go` — effect decoding/aggregation và registry race/constitution.
- `backend/internal/effectcatalog/` — block/trigger/capability manifest và validator.
- `backend/internal/simulator/engine.go` — combat profile integration.
- `backend/internal/gamefx/scouting_engine.go`, `backend/internal/db/schema_scouting.go` — scouting inputs và dấu vết/tracking state.
- `backend/internal/models/aperture.go`, `backend/internal/db/schema_aperture.go`, `backend/internal/handlers/aperture_portal.go`, `backend/internal/server/server.go` — aperture persistence, guest/portal flow và routes.
- `backend/data/Race.csv` — dữ liệu race effects hiện hành.
