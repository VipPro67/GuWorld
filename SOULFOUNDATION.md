Dưới đây là bản thiết kế chi tiết (Game Design Document) bằng định dạng Markdown, bám sát 100% cốt truyện Cổ Chân Nhân và cung cấp các logic cụ thể để bạn triển khai vào hệ thống Web Game (Golang/Next.js) của mình.

***

```markdown
# THIẾT KẾ CHI TIẾT: HỆ THỐNG HỒN PHÁCH & CẢNH GIỚI LƯU PHÁI

Trong thế giới Cổ Chân Nhân, tu vi (Rank) và Chân Nguyên/Tiên Nguyên chỉ là năng lượng bề nổi. Lực chiến thực sự và tiềm năng phát triển của một người chơi được quyết định bởi hai hệ thống ngầm cốt lõi: **Nội tình Hồn Phách (Soul Foundation)** và **Cảnh giới Lưu phái (Path Attainment)**.

---

## PHẦN 1: HỆ THỐNG NỘI TÌNH HỒN PHÁCH (SOUL FOUNDATION)

Hồn phách đại diện cho giới hạn tinh thần của người chơi, quyết định khả năng thao túng thú vật (Nô đạo), và đặc biệt là sinh mệnh của người chơi khi thám hiểm Mộng Cảnh.

### 1. Phân cấp Nội tình Hồn phách
Dựa theo cốt truyện, hồn phách được chia thành các cột mốc khắt khe:
1.  **Nhân Hồn (1):** Người bình thường.
2.  **Thập Nhân Hồn (10):** Tương đương 10 người trưởng thành.
3.  **Bách Nhân Hồn (100):** Cực hạn của phàm nhân bình thường.
4.  **Thiên Nhân Hồn (1,000)**
5.  **Vạn Nhân Hồn (10,000)**
6.  **Bách Vạn Nhân Hồn (1,000,000)**
7.  **Thiên Vạn Nhân Hồn (10,000,000)**
8.  **Ức Nhân Hồn (100,000,000):** Hồn phách có thể từ hư hóa thực, ngưng tụ thành thực thể có sức mạnh kình thiên.

### 2. Cách tính toán và Quy trình Tu Hồn (Tráng - Luyện - An)
Để tăng hồn phách mà không bị nổ tung hay phù phiếm, người chơi bắt buộc phải trải qua quy trình 3 bước:
*   **Tráng Hồn (Tăng số lượng):** Sử dụng **Cổ Can Đảm** (đặc sản Núi Đãng Hồn) để tăng trực tiếp điểm số Hồn phách mà không có tác dụng phụ. (Ví dụ: 1 Cổ Can Đảm = +10 điểm Hồn phách).
*   **Luyện Hồn (Tăng chất lượng):** Khi đạt các mốc giới hạn (Ví dụ: 100, 1.000), hồn phách sẽ bị "phù nề". Người chơi phải vào **Cốc Lạc Phách** chịu sự cắt chém của Gió Lạc Phách và Sương Mù Mê Hồn để loại bỏ tạp chất, làm hồn phách ngưng thực lại. *(Game logic: Mở khóa giới hạn Max Soul)*.
*   **An Hồn (Ổn định):** Dùng Nước Hồ An Hồn hoặc đi ngủ để ổn định linh hồn sau khi bị tra tấn ở Cốc Lạc Phách, chuẩn bị cho vòng tuần hoàn tiếp theo.

### 3. Ảnh hưởng của Hồn Phách tới Cơ chế Game
*   **Thám hiểm Mộng Cảnh (Dream Realm HP):** Hồn phách chính là "Thanh HP" khi ở trong Mộng cảnh. Mỗi bước di chuyển hoặc sai lầm trong mộng cảnh sẽ trừ đi số lượng lớn Nhân hồn. Nếu hồn phách tụt xuống 0 trong Mộng cảnh, người chơi sẽ chết vĩnh viễn (Permadeath).
*   **Nô Đạo (Beast Enslavement):** Điểm Hồn phách quy định "Giới hạn quân đội". Ví dụ: Bách Nhân Hồn điều khiển được Bách Thú Vương, Vạn Nhân Hồn điều khiển được Vạn Thú Vương, Thiên Vạn Nhân Hồn mới gánh vác nổi Hoang Thú Thái Cổ.

---

## PHẦN 2: HỆ THỐNG CẢNH GIỚI LƯU PHÁI (PATH ATTAINMENT)

Cảnh giới là sự thấu hiểu đại đạo thiên địa, là yếu tố quyết định người chơi là "Gà mờ" hay "Đại sư" thực thụ.

### 1. Phân cấp Cảnh giới
1.  **Phổ Thông (Ordinary):** Mặc định khi mới chơi.
2.  **Chuẩn Đại Sư / Đại Sư (Master):** Sinh ra trực giác.
3.  **Chuẩn Tông Sư / Tông Sư (Grandmaster):** Có thể "Loại suy".
4.  **Chuẩn Đại Tông Sư / Đại Tông Sư (Great Grandmaster):** Lợi dụng đạo ngân tự nhiên.
5.  **Chuẩn Vô Thượng / Vô Thượng Đại Tông Sư (Supreme Grandmaster):** Sát chiêu tự động vận hành, vượt qua cực hạn thiên địa.

### 2. Cách Tăng Cảnh Giới
*   **Thực hành (Grind):** Sử dụng Cổ trùng, tự sáng tạo Sát chiêu nhiều lần trong chiến đấu. Tốn hàng chục đến hàng trăm năm (Trong game: Yêu cầu cực kỳ nhiều lượt dùng skill và may mắn).
*   **Thám hiểm Mộng Cảnh (The Shortcut):** Đây là cách cốt lõi. Mộng cảnh chứa đựng **Chân Ý (True Meaning)** của tiền nhân. Khi vượt qua mộng cảnh thành công, người chơi trực tiếp hấp thu Chân ý, cảnh giới lập tức nhảy vọt (Ví dụ: Từ Phổ thông lên thẳng Đại sư).

### 3. Ảnh hưởng của Cảnh giới tới Cơ chế Game
*   **Ghép Sát Chiêu & Luyện Cổ (Crafting):** Cảnh giới càng cao, tỷ lệ ghép thành công sát chiêu và luyện Cổ càng lớn. Tông sư có thể tự cải tiến Cổ phương.
*   **Hiệu ứng Mốc Cảnh Giới (Milestone Buffs):**
    *   *Đại sư (Intuition):* Khi đối phương dùng kỹ năng cùng hệ, hệ thống tự động hiển thị gợi ý né tránh hoặc nhược điểm của chiêu đó.
    *   *Tông sư (Mimicry - Loại suy):* Xóa bỏ hình phạt (Penalty) khi dùng chéo hệ. (VD: Tông sư Lực đạo dùng Cổ Thủy đạo vẫn nhận 100% sát thương/hiệu quả thay vì bị giảm sút).
    *   *Đại Tông Sư (Natural Dao):* Tự động cộng hưởng với Đạo ngân của môi trường Lưới 3x3. (VD: Đánh ở bản đồ Biển, Đại Tông Sư Thủy Đạo tự động được +50% Sát thương/Phòng thủ không cần dùng Cổ).
*   **Điều kiện Nuốt Tiên Khiếu (Endgame):** Để cướp đoạt Phúc địa/Động thiên của Cổ Tiên khác nhằm bỏ qua tai kiếp, bắt buộc lưu phái phải tương đồng và cảnh giới phải đạt Tông Sư (nuốt rank 7) hoặc Đại Tông Sư (nuốt rank 8) *(Dựa trên cơ chế gốc của nguyên tác)*.

---

## PHẦN 3: TRIỂN KHAI VÀO WEB GAME (DEVELOPER GUIDE)

Dưới đây là logic dành cho Backend (Golang) và Database (PostgreSQL).

### A. Database Models

```go
// 1. Model Soul Foundation
type PlayerSoul struct {
    PlayerID         string  `json:"player_id"`
    CurrentSoulHP    int     `json:"current_soul_hp"` // Thay đổi liên tục khi vào Mộng cảnh
    MaxSoulPoints    int     `json:"max_soul_points"` // Điểm nội tình (Ví dụ: 1,000,000)
    SoulTier         string  `json:"soul_tier"`       // ENUM: "NHAN_HON", "BACH_NHAN", "VAN_NHAN", "UC_NHAN"
    NeedsRefinement  bool    `json:"needs_refinement"`// Nếu = true, ăn Cổ Can Đảm vô tác dụng, phải vào Cốc Lạc Phách
}

// 2. Model Path Attainment
type PlayerAttainment struct {
    PlayerID    string         `json:"player_id"`
    Paths       map[string]int `json:"paths"` // e.g., {"Sword": 1, "Strength": 3, "Dream": 0}
    // Giá trị INT tương ứng: 
    // 0 = Phổ Thông
    // 1 = Đại Sư
    // 2 = Tông Sư
    // 3 = Đại Tông Sư
    // 4 = Vô Thượng
}
```

### B. Game Loop Logic (Vòng Lặp Thám Hiểm Mộng Cảnh)

**API Endpoint: `POST /api/dream-realm/explore`**
1.  **Check Input:** Yêu cầu người chơi truyền vào `DreamRealm_ID` và `Cổ Giải Mộng` (tùy chọn).
2.  **Trừ Soul HP (Hồn Phách):** Mỗi Turn trả lời câu hỏi/chiến đấu trong Mộng Cảnh trừ đi một lượng `CurrentSoulHP`. Nếu không có Cổ Giải Mộng (Cheat code), lượng trừ rất lớn. Nếu `CurrentSoulHP <= 0` -> Kích hoạt trạng thái `Coma/Death`.
3.  **Xử lý Kết quả (Win/Loss):**
    *   *Loss:* Mất vĩnh viễn Soul HP đã tiêu hao, phải tốn Cổ Can Đảm để hồi phục.
    *   *Win:* Nhận được **Chân Ý (True Meaning)** của mộng cảnh đó.
4.  **Tăng Cảnh Giới:** Nếu hấp thu Chân Ý hệ Kiếm, `PlayerAttainment.Paths["Sword"] += 1` (Lên cấp Đại sư).

### C. Combat & Formula Updates
Trong hàm `ResolveCombat(player, enemy)`:
```go
// Xử lý Buff Đại Tông Sư
if player.Attainments[skill.Path] >= 3 /* Đại Tông Sư */ {
    if currentMap.Element == skill.Path {
        finalDamage *= 1.5 // Nhờ sử dụng Đạo Ngân môi trường
    }
}

// Xử lý Buff Tông Sư (Loại Suy)
if player.Attainments["Strength"] >= 2 /* Tông Sư */ {
    // Không bị trừ penalty khi dùng sai hệ phái
    removeElementPenalty(skill)
}

// Xử lý Nô Đạo (Enslavement Limit)
maxBeasts := calculateMaxBeasts(player.Soul.MaxSoulPoints)
if len(player.SummonedBeasts) > maxBeasts {
    return error("Nội tình Hồn Phách không đủ để khống chế thêm thú!")
}
```

### D. Gợi ý UI/UX (Next.js)
*   **Soul Panel:** Tạo một thanh trạng thái linh hồn. Khi linh hồn đổi màu (bị đục/phù nề), nút dùng Cổ Can Đảm bị `disabled`, yêu cầu người chơi mở khóa tính năng "Cốc Lạc Phách".
*   **Attainment Badges:** Trong trang Profile, hiển thị các huy hiệu Lục giác cho từng hệ phái. Khi đạt "Tông Sư", huy hiệu phát sáng viền vàng, click vào hiện Tooltip: *"Tông Sư Lực Đạo: Bỏ qua hình phạt mâu thuẫn hệ phái."*
*   **Dream Realm Warning:** Khi ấn vào một Mộng Cảnh, hiện cảnh báo đỏ nếu `CurrentSoulHP` của người chơi quá thấp so với độ khó của Mộng Cảnh, tránh việc người chơi "chết trong mộng" do không đọc kỹ.
```