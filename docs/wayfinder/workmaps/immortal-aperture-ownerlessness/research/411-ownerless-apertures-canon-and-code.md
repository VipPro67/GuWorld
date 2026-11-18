# Research 411: Tiên Khiếu vô chủ, địa linh và thôn phệ

Type: research  
Status: bằng chứng ghi nhận; luật gameplay còn mở  
Scope: tiền lệ trong *Cổ Chân Nhân / Reverend Insanity* và hiện trạng GuWorld; không đề xuất quyết định thiết kế.

## Kết quả

### Canon

- **Địa linh hình thành sau khi chủ nhân chết, nhưng gắn với một tiên khiếu cụ thể.** Chương 570 mô tả chấp niệm còn lại của Cổ Tiên kết hợp thiên địa lực trong phúc địa để hình thành địa linh; chương 409 nhắc lại trường hợp Hồ Tiên và xem địa linh như sự tiếp nối một phần sinh mệnh. Đây không phải bằng chứng rằng mọi cái chết đều tạo địa linh: các ví dụ gắn với chấp niệm và tiên khiếu còn tồn tại. [Ch. 570](https://www.webnovel.com/book/32605619408374705/chapter-570-compromise_88031368088443139) · [Ch. 409](https://www.webnovel.com/book/7996858406002505/30050075942124186)
- **Địa linh giữ chấp niệm/điều kiện kế thừa, và người khác có thể trở thành chủ nhân khi thỏa điều kiện.** Ba Vương Phúc Địa yêu cầu hoàn thành tâm nguyện của chủ cũ; Tiểu Hồn Phúc Địa chấp nhận Hắc Lâu Lan qua tín vật “tình thân” gắn với điều kiện của địa linh. Các trường hợp cho thấy điều kiện có thể riêng cho từng phúc địa, không phải một nghi thức nhận chủ phổ quát. [Ch. 570](https://www.webnovel.com/book/32605619408374705/chapter-570-compromise_88031368088443139) · [Ch. 953](https://www.webnovel.com/book/32605619408374705/chapter-953-late-by-one-step_88346255494796148)
- **Động thiên là cấp/loại tiên khiếu khác phúc địa.** Chương 795 nói tiên khiếu thường của Cổ Tiên trở thành động thiên khi đạt rank 8; chương 713 gọi khiếu rank 8 hoặc 9 là động thiên. Động thiên Tinh Túc là một tiền lệ cụ thể: chủ nhân Thất Tinh Tử đã chết, và trong dòng thời gian Fang Yuan nhớ lại, động thiên về sau rơi vỡ thành nhiều mảnh tiên khiếu. Điều này cho thấy cái chết không nhất thiết xóa động thiên ngay, nhưng không xác lập thời hạn hay kết cục chung. [Ch. 795](https://www.webnovel.com/ru/book/32605619408374705/88182063458478361) · [Ch. 713](https://www.webnovel.com/ru/book/32605619408374705/88147310327939695)
- **Thôn phệ tiên khiếu có điều kiện về trạng thái, kích thước, tu vi và cảnh giới thành tựu.** Chương 735 nói tiên khiếu chết không thể thôn phệ tiên khiếu khác và sẽ sụp dần; khiếu nhỏ không nuốt khiếu lớn; cảnh giới thành tựu cần tăng theo cấp mục tiêu (rank 6/master, rank 7/grandmaster, rank 8/great grandmaster). Đây là lời giải thích trong bối cảnh của Fang Yuan, không nên tự động biến thành luật số học đầy đủ cho mọi tình huống. [Ch. 735](https://www.webnovel.com/book/32605619408374705/88159579304678553)
- **“Hấp thu” cũng có nghĩa khác: đặt tiên khiếu ra ngoài để hút thiên địa khí.** Chương 865 mô tả chủ nhân đặt khiếu ra ngoài, vào bên trong và hấp thu thiên địa khí; đó không phải thôn phệ một phúc địa khác. [Ch. 865](https://www.webnovel.com/book/32605619408374705/chapter-865-heavenly-wish_88264094381041044)

### Chấp niệm, ý chí và địa linh

Canon tách **chấp niệm** của chủ nhân đã chết kết hợp thiên địa lực để hình thành **địa linh**, khỏi **ý chí (will)** như một hiện tượng tư tưởng được Trí Đạo nghiên cứu/phát triển thành nhiều loại cổ trùng. Chương 650 nói rõ địa linh khác với ý chí và không nói dối. Vì vậy không nên dùng “ý chí sót lại” như tên gọi đồng nghĩa với địa linh: chấp niệm là thành phần dẫn đến linh thể, còn ý chí là khái niệm riêng. [Ch. 570](https://www.webnovel.com/book/32605619408374705/chapter-570-compromise_88031368088443139) · [Ch. 650](https://www.webnovel.com/book/32605619408374705/chapter-650-return-to-hu-immortal-blessed-land_88081532098506983)

### GuWorld hiện tại

- `immortal_apertures.player_id` là `NOT NULL UNIQUE REFERENCES players(id) ON DELETE CASCADE`; schema hiện không cho bản ghi Tiên Khiếu tồn tại độc lập sau khi xóa nhân vật. Bảng `aperture_land_spirits` có các trường chấp niệm/điều kiện claim, và comment nói linh xuất hiện khi host chết hoặc khiếu vô chủ, nhưng tìm kiếm toàn backend chỉ thấy schema và model `ApertureLandSpirit`, không thấy code tạo/đọc/claim hay route sử dụng bảng đó. [schema_aperture.go](../../../../../backend/internal/db/schema_aperture.go#L8) · [model aperture.go](../../../../../backend/internal/models/aperture.go#L793)
- Tạo khiếu hiện gắn trực tiếp với player: `createAperture` ghi `player_id`; `GetImmortalAperture` truy vấn theo player ID. [aperture_core.go](../../../../../backend/internal/handlers/aperture_core.go#L24) · [aperture_core.go](../../../../../backend/internal/handlers/aperture_core.go#L210)
- Các route aperture người chơi hiện hành nằm dưới `/api/immortal-aperture` và dùng handler sở hữu theo người chơi; route `refinement-queue` quản lý công việc trong kinh tế tiên khiếu, không phải thôn phệ tiên khiếu. Không thấy handler/route cho chuyển chủ, nhận phúc địa vô chủ, hoặc annex/absorb aperture. [server.go](../../../../../backend/internal/server/server.go#L1048) · [aperture_points.go](../../../../../backend/internal/handlers/aperture_points.go#L1048)
- Cái chết trong `death_penalty.go` xử lý tổn thất tài nguyên/Gu và ground drops; code khảo sát không chuyển Tiên Khiếu thành vô chủ. Đây là death penalty trong gameplay, không đồng nghĩa xóa nhân vật hoặc permadeath. [death_penalty.go](../../../../../backend/internal/handlers/death_penalty.go#L232)

### Còn chưa quyết định

- “Chết” trong issue là thua trận/áp dụng death penalty, xóa nhân vật, hay một trạng thái permadeath mới? Canon có một tiền lệ động thiên còn tồn tại sau cái chết chủ nhân rồi về sau vỡ mảnh, nhưng không đủ để khái quát vận mệnh hoặc thời hạn của mọi phúc địa/động thiên.
- Khi chủ chết, tiên khiếu còn nguyên, suy thoái/sụp đổ, hay chịu một cơ chế riêng? Khiếu đang đặt ngoài và đang có khách/worker được xử lý ra sao?
- Địa linh là vật thể kế thừa cho khiếu vô chủ, hay chỉ xuất hiện ở một số loại khiếu/chấp niệm? Ai tạo điều kiện claim và điều kiện có thể thay đổi không?
- “Refine”/“absorb” chỉ ra hành động nào: chuyển quyền sở hữu qua điều kiện địa linh, đặt khiếu vào khiếu khác, hay một quy trình riêng? Phạm vi canon thôn phệ trong chương 735 không giải quyết lịch sử/điều kiện của một khiếu vô chủ.
- Bản ghi nào là nguồn sở hữu chuẩn khi có địa linh/không có player chủ? Schema hiện ép mỗi aperture có đúng một player chủ, nên câu trả lời ảnh hưởng mô hình dữ liệu và các truy vấn owner hiện có.

## Ghi chú nguồn

Canon dẫn chương WebNovel của *Reverend Insanity*; mô tả là tóm lược ngắn, không trích nguyên văn. Code paths là nguồn hiện hành tại repo khi khảo sát. “Không thấy” có nghĩa tìm kiếm source backend/route hiện tại không tìm ra consumer, không khẳng định không có ý tưởng hoặc issue khác mô tả tính năng tương lai.
