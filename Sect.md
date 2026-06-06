Triển khai tính năng **Tông môn (Sect)** là một bước đi cực kỳ xuất sắc để mở rộng nội dung game của bạn, đặc biệt là giai đoạn người chơi ở Trung Châu hoặc khi thành lập thế lực riêng. 

Theo nguyên tác, Tông môn do Nguyên Thủy Tiên Tôn sáng lập, phá vỡ chế độ Gia tộc (huyết thống). Tông môn lấy **năng lực, phẩm đức và sự trung thành** làm thước đo, dùng quan hệ Sư - Đồ để thay thế quan hệ ruột thịt. 

Dưới đây là chi tiết các tính năng nhỏ (Sub-features) cần có để xây dựng một hệ thống Tông môn chuẩn xác với *Cổ Chân Nhân*, kèm theo hướng dẫn triển khai Backend (Golang):

### 1. Hệ thống Phân cấp & Khảo hạch (Hierarchy & Exams)
Khác với bang hội game bình thường, Tông môn có giai cấp cực kỳ nghiêm ngặt và số lượng giới hạn ở cấp cao.
*   **Hệ thống Cấp bậc:** Đệ tử ngoại môn $\rightarrow$ Đệ tử nội môn $\rightarrow$ Đệ tử tinh anh $\rightarrow$ Đệ tử chân truyền $\rightarrow$ Trưởng lão $\rightarrow$ Chưởng môn $\rightarrow$ Thái thượng trưởng lão.
*   **Cơ chế Thăng cấp (Khảo hạch):** Không phải cứ đủ EXP là lên cấp, người chơi phải tham gia các kỳ thi định kỳ:
    *   *Tiểu khảo (3 năm/lần trong game):* Lên Nội môn.
    *   *Trung khảo (8 năm/lần):* Giành chiến thắng để lên Tinh anh.
    *   *Đại khảo (50 năm/lần):* Sống sót và chiến thắng để lên Chân truyền.
    *   *Logic Game:* Tổ chức các giải đấu (Tournament) PvP hoặc PvE theo chu kỳ thời gian thực (ví dụ: mỗi tuần 1 lần). Top X người chơi mới được thăng cấp.

### 2. Hệ thống Điểm Cống Hiến (Contribution Economy)
Đây là "huyết mạch" kinh tế của Tông môn. Tông môn không phát lương miễn phí, mọi thứ đều quy ra Điểm cống hiến môn phái.
*   **Kiếm điểm:** Người chơi quyên góp Cổ trùng, Nguyên thạch, Tiên tài, Cổ phương cho Tông môn, hoặc hoàn thành Nhiệm vụ.
*   **Tiêu điểm:** Sử dụng điểm để mua Cổ trùng quý hiếm, thuê người khác luyện cổ giúp, mượn Tiên Cổ, hoặc mở khóa các tầng Tàng Kinh Các để đọc bí phương.
*   **Bảng xếp hạng (Cổ Tín Đạo):** Mỗi người chơi được phát một phàm cổ Tín đạo chứa danh sách Điểm cống hiến của toàn bộ thành viên. Đây chính là Bảng xếp hạng (Leaderboard) tạo động lực cày cuốc.

### 3. Hệ thống Bảng Nhiệm Vụ (Mission Board)
Nhiệm vụ tông môn không chỉ là đánh quái. Nó bao gồm:
*   **Nhiệm vụ Bắt buộc:** Mỗi tháng cổ sư bắt buộc phải hoàn thành ít nhất 1 nhiệm vụ. Nếu không hoàn thành sẽ bị rớt đánh giá, giáng cấp hoặc trục xuất. Chỉ có Trưởng lão hoặc tu vi cao mới có quyền "bỏ qua" nhiệm vụ.
*   **Phân loại Nhiệm vụ:**
    *   *Thu thập:* Đi tìm tài liệu luyện cổ đặc thù.
    *   *Bồi luyện:* Làm "quân xanh" đấu tập với các đệ tử khác hoặc người Lông để rèn luyện kỹ năng chiến đấu.
    *   *Chinh phạt:* Tiêu diệt thế lực thù địch hoặc dọn dẹp Hoang thú quanh trụ sở.
    *   *Trấn thủ:* Bảo vệ các điểm tài nguyên của môn phái.

### 4. Tàng Kinh Các & Bảo Khố (Sect Vault)
Môn phái có nội tình vô cùng thâm hậu, là kho tàng khổng lồ thu hút người chơi cống hiến.
*   **Hệ thống Mượn Cổ:** Tiên cổ hoặc Cổ cấp cao cực kỳ đắt. Tông môn cho phép "thuê/mượn" cổ trùng theo thời gian và trừ dần điểm cống hiến.
*   **Đổi Cổ Phương & Sát Chiêu:** Tàng thư vô số, người chơi muốn học Sát chiêu mới hay xem Cổ phương (ví dụ Cổ Cốt Nhục Đoàn Viên, hay các sát chiêu Lực đạo) phải đạt đủ Cấp bậc (Đệ tử chân truyền/Trưởng lão) và tiêu hao điểm cống hiến.

### 5. Cơ chế Sư - Đồ & Trợ giúp Luyện Cổ (Master-Disciple System)
Trong Tông môn, quan hệ Sư - Đồ thay thế huyết thống gia tộc.
*   **Bái Sư:** Đệ tử cấp thấp có thể bái Trưởng lão (người chơi cấp cao hoặc NPC) làm thầy. 
*   **Truyền công & Hỗ trợ:** Sư phụ có thể share (chia sẻ) Cổ phương hoặc trợ giúp đồ đệ Luyện cổ để tăng tỷ lệ thành công. Đổi lại, sư phụ nhận được điểm cống hiến hoặc trích phần trăm tài nguyên từ đồ đệ.

---

### 💻 Gợi ý triển khai Database (Golang/GORM)

Để triển khai hệ thống này, bạn cần thiết kế cấu trúc Dữ liệu cho Tông Môn vô cùng chặt chẽ. Dưới đây là phác thảo Model Backend cho AI Agent của bạn:

```go
// 1. Model Tông Môn
type Sect struct {
    ID                string    `json:"id"`
    Name              string    `json:"name"`
    Level             int       `json:"level"` // Cấp độ tông môn (Môn phái nhỏ, Mười đại cổ phái...)
    VaultResources    JSONB     `json:"vault_resources"` // Kho tài nguyên chung
    AvailableGu       JSONB     `json:"available_gu"` // Danh sách cổ trùng có thể đổi/mượn
    AvailableRecipes  JSONB     `json:"available_recipes"` // Danh sách cổ phương
}

// 2. Model Thành Viên Tông Môn (Mapping Player with Sect)
type SectMember struct {
    ID                string    `json:"id"`
    SectID            string    `json:"sect_id"`
    PlayerID          string    `json:"player_id"`
    Rank              string    `json:"rank"` // ENUM: NGOAI_MON, NOI_MON, TINH_ANH, CHAN_TRUYEN, TRUONG_LAO, CHUONG_MON
    ContributionPts   int       `json:"contribution_points"` // Điểm cống hiến
    MasterID          *string   `json:"master_id"` // ID của sư phụ (nếu có)
    JoinedAt          time.Time `json:"joined_at"`
}

// 3. Model Nhiệm Vụ Tông Môn
type SectMission struct {
    ID                string    `json:"id"`
    SectID            string    `json:"sect_id"`
    Title             string    `json:"title"`
    Type              string    `json:"type"` // "GATHER", "SPAR", "COMBAT", "REFINE"
    Requirement       JSONB     `json:"requirement"` // Cụ thể cần nộp vật phẩm gì, hoặc giết quái nào
    RewardPoints      int       `json:"reward_points"` // Điểm cống hiến thưởng
    RequiredRank      string    `json:"required_rank"` // Cấp bậc tối thiểu để nhận
    AssigneeID        *string   `json:"assignee_id"` // Ai đang nhận nhiệm vụ này
    Status            string    `json:"status"` // "OPEN", "IN_PROGRESS", "COMPLETED", "FAILED"
    Deadline          time.Time `json:"deadline"`
}
```

### 💡 Lời khuyên thiết kế UI/UX (Next.js):
Khi người chơi gia nhập Tông môn, UI nên có một Tab riêng là **"Lệnh Bài Tông Môn"** (Tín đạo cổ trùng). Trong Tab này:
*   Trang chủ hiển thị số Điểm cống hiến và Rank hiện tại.
*   Bảng xếp hạng (Leaderboard) Điểm cống hiến của các đệ tử khác để kích thích sự ganh đua.
*   Khu vực Nhận nhiệm vụ và Đổi điểm lấy Cổ trùng/Bí phương. 

Triển khai tính năng này sẽ giữ chân người chơi lâu dài hơn rất nhiều, chuyển game từ giai đoạn sinh tồn cá nhân sang các cuộc cạnh tranh thế lực diện rộng!