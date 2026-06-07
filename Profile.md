Ý tưởng cho phép người chơi **làm giả thông tin Profile (Hồ sơ cá nhân)** là một nước đi thiết kế cực kỳ xuất sắc và bám sát 100% tinh thần "lừa gạt, giấu bài" của *Cổ Chân Nhân*. Trong nguyên tác, Phương Nguyên sống sót và vươn lên đỉnh cao chính là nhờ vô số thân phận giả (Thường Sơn Âm, Vũ Di Hải, Trí Đa Tinh, Toán Bất Tẫn, Liễu Quán Nhất) và thủ đoạn che giấu khí tức.

Để triển khai tính năng này vào hệ thống Web Game (Golang/Next.js) của bạn sao cho không bị lạm dụng và mang tính chiến thuật cao, bạn không thể cho phép người chơi "sửa Text miễn phí". Mọi sự ngụy trang đều phải **trả giá bằng Cổ trùng, Sát chiêu và Chân nguyên/Tiên nguyên**.

Dưới đây là chi tiết thiết kế và cách lập trình (Implementation) cho tính năng Profile:

### 1. Cơ chế Ngụy Trang (Disguise Mechanics)
Người chơi có 2 bộ hồ sơ trong Database: `Real_Profile` (Sự thật) và `Public_Profile` (Hiển thị cho người khác).
Để thay đổi `Public_Profile`, người chơi bắt buộc phải trang bị các Cổ trùng thuộc nhóm **Ẩn nấp / Ngụy trang / Trí Đạo / Tín Đạo**.

*   **DisplayName (Tên hiển thị):** 
    *   Mặc định là tên thật.
    *   Muốn đổi tên và ngoại hình: Phải dùng **Cổ Da Người** (Phàm đạo) hoặc **Cổ Đổi Mặt**. Ở cấp cao, phải dùng sát chiêu **Gặp Mặt Từng Quen Biết** (Cần Cổ Thái Độ làm hạch tâm). Khi trang bị các cổ này, UI sẽ mở khóa một ô Input để gõ tên giả.
*   **Level (Tu vi - Rank/Sub-rank):** 
    *   Mặc định hiển thị tu vi thật.
    *   Giấu tu vi (Giả yếu): Bắt buộc phải trang bị **Cổ Liễm Tức**. Người chơi Rank 4 nếu dùng Cổ Liễm Tức có thể kéo thanh trượt trên UI để hiển thị mình là Rank 1, 2 hoặc 3 trên Profile.
    *   Nâng tu vi (Giả mạnh): Cần dùng Cổ Khí Tức giả tạo để dọa người.
*   **Vital Gu (Bản mệnh Cổ):**
    *   *Tuyệt đối ẩn (Hidden by default):* Bản mệnh cổ là nhược điểm chí mạng của cổ sư. Nếu kẻ địch biết, chúng sẽ tìm cách khắc chế lưu phái đó. 
    *   *Cố tình hiển thị giả:* Người chơi có thể dùng một con phàm cổ rác rưởi đưa lên Profile giả làm Bản Mệnh Cổ để "giăng bẫy" lừa kẻ địch mang sai đồ khắc chế.
*   **Lưu Phái (Path) & Đạo Ngân (Dao Marks):**
    *   Đạo ngân là con số ẩn. Người khác muốn xem Profile của bạn có bao nhiêu Đạo Ngân thì họ phải dùng **Cổ Đạo Khả Đạo** (Tín Đạo) để soi.
    *   Bạn muốn giấu Đạo Ngân: Phải dùng **Tiên Cổ Ám Độ**. Tiên cổ này che giấu toàn bộ Đạo ngân và Lưu phái thật, kẻ địch bấm soi Profile của bạn sẽ chỉ thấy "???".

---

### 2. Cuộc chiến "Trinh Sát vs Ngụy Trang" (Backend Logic)
Khi Player A click vào Avatar của Player B để xem Profile, Backend không đơn thuần trả về dữ liệu, mà phải tính toán một bài toán **Va chạm Chỉ số (Stat Clash)**.

**Công thức Backend:**
`Điểm Trinh Sát của A (Investigation)` vs `Điểm Ngụy Trang của B (Concealment)`

*   **Nếu Trinh Sát (A) < Ngụy Trang (B):** 
    *   Backend chỉ trả về `Public_Profile` (Dữ liệu giả). Player A hoàn toàn tin rằng Player B là một kẻ yếu Rank 2 tên là "Giáp Ất", dùng Nguyệt Quang Cổ.
*   **Nếu Trinh Sát (A) >= Ngụy Trang (B):** 
    *   Backend trả về `Real_Profile`. 
    *   *Hiệu ứng UI cực ngầu:* Trên màn hình của Player A, Profile của Player B sẽ có hiệu ứng "Kính vỡ" (Shatter Effect) hoặc Text bị "Nhiễu loạn" (Glitch), tên "Giáp Ất" bị rớt xuống, lòi ra chữ "Phương Nguyên - Rank 5 Đỉnh Phong". Lớp ngụy trang bị phá vỡ!

---

### 3. Cấu trúc Database (Golang / GORM)

Bạn cung cấp đoạn Model này cho AI Agent (Backend) để triển khai:

```go
type PlayerProfile struct {
    PlayerID         string  `json:"player_id"`
    
    // --- SỰ THẬT (Chỉ Backend và bản thân người chơi thấy) ---
    RealName         string  `json:"real_name"`
    RealRank         int     `json:"real_rank"`
    RealSubRank      int     `json:"real_sub_rank"`
    RealVitalGuID    string  `json:"real_vital_gu_id"`
    RealPaths        JSONB   `json:"real_paths"` // Dao marks
    
    // --- LỚP NGỤY TRANG (Public View) ---
    IsDisguised      bool    `json:"is_disguised"`
    DisguiseGuID     string  `json:"disguise_gu_id"` // Ví dụ: ID của Cổ Liễm Tức / Thái Độ Cổ
    ConcealmentScore float64 `json:"concealment_score"` // Tính từ Cổ Ngụy Trang
    
    // Dữ liệu do người chơi tự Gõ/Set nếu IsDisguised = true
    FakeName         string  `json:"fake_name"`
    FakeRank         int     `json:"fake_rank"`
    FakeVitalGuID    string  `json:"fake_vital_gu_id"`
}
```

**Logic API Fetch Profile (`GET /api/players/:target_id/profile`):**
*Lưu ý bảo mật (Security):* Backend **tuyệt đối không bao giờ** gửi cả Real và Fake data về Frontend cùng lúc rồi dùng React để giấu. Dân IT F12 sẽ soi được Network tab. Backend phải là người quyết định gửi data thật hay giả!

```go
func GetPlayerProfile(viewer *Player, target *PlayerProfile) ProfileResponse {
    viewerInvestigationScore := CalculateInvestigation(viewer) // Dựa vào Cổ Trinh Sát đang mặc
    
    if target.IsDisguised && viewerInvestigationScore < target.ConcealmentScore {
        // Trinh sát thất bại -> Trả về Dữ liệu Giả
        return ProfileResponse{
            Name: target.FakeName,
            Rank: target.FakeRank,
            VitalGu: target.FakeVitalGuID,
            IsTruth: false,
        }
    }
    
    // Trinh sát thành công hoặc Target không ngụy trang -> Trả về Dữ liệu Thật
    return ProfileResponse{
        Name: target.RealName,
        Rank: target.RealRank,
        VitalGu: target.RealVitalGuID,
        IsTruth: true,
    }
}
```

---

### 4. Giao diện UI/UX (Next.js / Frontend)

**Màn hình Cài đặt của Bản thân (Profile Editor):**
1.  Bình thường, các ô Name, Rank, Vital Gu bị **Disable (Màu xám)**, không thể sửa.
2.  Nếu người chơi kéo **Cổ Liễm Tức** vào Slot Hỗ Trợ, ô `FakeRank` sáng lên (Enable). Giao diện hiện thông báo: *"Khí tức đã được che giấu. Đạo hữu muốn hiển thị tu vi là Nhất Chuyển hay Nhị Chuyển?"*.
3.  Nếu trang bị **Cổ Da Người / Sát chiêu Gặp Mặt Từng Quen Biết**, ô `FakeName` sáng lên. Cho phép gõ tên mới.

**Màn hình Xem người khác (Inspect Others):**
1.  Thêm một nút **[ SỬ DỤNG TRINH SÁT ]** (Tốn Mana / Lượt).
2.  Khi bấm vào, hệ thống chạy thanh Progress Bar (như đang soi Data). Nếu trinh sát thành công, thẻ Profile của đối phương sẽ đổi từ màu tối sang màu rực rỡ, Avatar giả bị xé rách, các thông số giả bốc cháy và hiện ra thông số thật.
3.  *Lưu ý:* Việc dùng Cổ Trinh sát soi người khác ở nơi công cộng sẽ sinh ra "Địch Ý" (Ác ý). Hệ thống có thể báo cho người bị soi biết: *"Có kẻ đang dùng Cổ Đạo Khả Đạo dòm ngó Đạo ngân của bạn!"* để tạo sự thù hằn PvP.

Với thiết kế này, Game của bạn sẽ tái hiện hoàn hảo những pha "giả trư ăn thịt hổ" (giấu tu vi để dụ kẻ địch tấn công rồi phản sát) kinh điển của Cổ Chân Nhân!