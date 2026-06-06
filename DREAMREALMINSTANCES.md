Để triển khai tính năng "Phó bản" (Dungeon/Instance) vào tựa game RPG Text-based *Cổ Chân Nhân* của bạn, chúng ta cần gạt bỏ tư duy phó bản kiểu truyền thống (chỉ vào đánh quái, diệt Boss, rớt đồ). 

Trong thế giới Cổ Chân Nhân, phó bản được gọi bằng những cái tên mang đậm tính triết lý và cốt truyện: **Truyền Thừa (Inheritances)**, **Mộng Cảnh (Dream Realms)**, **Phúc Địa Vô Chủ (Ownerless Blessed Lands)**, và **Bí Cảnh Thiên Địa (Heaven and Earth Secluded Domains)**.

Dưới đây là thiết kế chi tiết các loại Phó bản và cơ chế hoạt động để bạn triển khai vào Game:

### 1. PHÂN LOẠI PHÓ BẢN (INSTANCE TYPES)

Dựa trên cốt truyện, bạn nên chia Phó bản thành 4 loại chính, phục vụ cho các cấp độ và mục đích khác nhau:

#### A. Truyền Thừa (Inheritances) - Phó bản Solo giải đố
*   **Lore:** Đây là nơi các cổ sư đời trước trước khi chết lập ra để tìm người kế vị, chia làm Chính đạo và Ma đạo. Truyền thừa Chính đạo thường khảo nghiệm phẩm đức, thất bại không mất mạng. Truyền thừa Ma đạo tàn khốc, câu đố tầng tầng lớp lớp, thất bại là chết. (Ví dụ: Truyền thừa Hoa Tửu Hành Giả).
*   **Gameplay:** Thích hợp cho người chơi Phàm nhân (< Rank 6). Thay vì chỉ Combat, người chơi phải đối mặt với **Bẫy rập** và **Giải đố**. Đôi khi hệ thống yêu cầu người chơi phải dùng đúng loại Cổ trùng nhất định mới mở được cửa (ví dụ dùng sức mạnh Bạch Thỉ Cổ đẩy tảng đá).

#### B. Mộng Cảnh (Dream Realms) - Phó bản Sinh tồn Tinh thần
*   **Lore:** Mộng cảnh là hiện thực ảo mô phỏng lại kiếp sống của các vị tiên hiền. 
*   **Gameplay:** Khi vào Mộng cảnh, Cổ trùng và Tu vi hiện tại của người chơi bị khóa. Hệ thống ép người chơi nhập vai vào một nhân vật (ví dụ: một đứa trẻ đang học bày Trận đạo,). 
*   **Cơ chế cốt lõi:** Thăm dò Mộng cảnh không trừ HP vật lý mà trừ **Nội tình Hồn Phách (Soul Foundation)**. Nếu chọn sai hoặc đánh thua trong mộng, Hồn phách giảm mạnh. Nếu có sát chiêu *Giải Mộng*, người chơi có thể "Skip/Hint" qua ải. Hoàn thành phó bản sẽ được thưởng **Chân Ý (Tăng trực tiếp Cảnh giới Lưu phái)**,.

#### C. Phúc Địa/Động Thiên Vô Chủ - Phó bản Raid Đội (GvG/Party)
*   **Lore:** Khi Cổ Tiên chết đi, tiên khiếu rơi xuống đất hóa thành Phúc địa. Bên trong sinh ra Địa Linh (Land Spirit) mang theo chấp niệm của người chết,.
*   **Gameplay:** Dành cho Rank 6+. Nhiều nhóm người chơi cùng tiến vào tranh đoạt. Để "Clear" phó bản này, không phải là giết Địa Linh, mà phải **hoàn thành Chấp niệm của Địa Linh để nó nhận chủ**. (Ví dụ: Địa linh Bá Quy yêu cầu người chơi phải luyện thành công cổ Không Khiếu Thứ Hai,).

#### D. Siêu Phó Bản Sự Kiện (Mega Dungeons)
*   **Lore:** Những công trình vĩ đại như *Lầu Chân Dương 88 Góc*, hoặc *Bia Công Đức*.
*   **Gameplay:** Hoạt động theo mô hình Tháp (Tower) hoặc Bảng nhiệm vụ. Ví dụ ở Bia Công Đức, người chơi không được phép PvP, mà phải thi đua làm nhiệm vụ (như thu thập kỳ nhông, sửa chữa cổ trận) để tích lũy điểm công đức đổi Tiên Cổ,. Tại Lầu Chân Dương, cần có "Lệnh bài" (Khách lệnh, Chủ lâu lệnh) mới được chọn tầng để vượt ải.

---

### 2. CƠ CHẾ HOẠT ĐỘNG TRONG PHÓ BẢN (MECHANICS)

Để Phó bản hấp dẫn, bạn cần áp dụng các cơ chế khắc nghiệt sau:

*   **Sương mù chiến tranh (Fog of War) & Lưới độc lập:** Khi bước vào Phó bản, người chơi bị chuyển sang một Grid Map (Bản đồ lưới) hoàn toàn tách biệt. Tầm nhìn bị hạn chế, phải dùng Cổ Trinh Sát để mở map.
*   **Tiêu hao Lượt (Turn/Stamina) hoặc Vật Phẩm:** Đi dạo trong phó bản làm giảm Độ no của Cổ trùng và tốn Chân Nguyên,.
*   **Cơ chế "Mượn xác hoàn hồn" (Scenario Play):** Giống như Mộng cảnh, đôi khi Phó bản tước hết đồ của người chơi, ném cho họ 4 con phàm cổ và yêu cầu họ phải tự tìm cách phối hợp chúng để chiến thắng Boss,,. Điều này thử thách tư duy build đồ của người chơi.
*   **Permadeath (Chết vĩnh viễn):** Nếu chết trong truyền thừa Ma đạo hoặc Hồn phách cạn kiệt trong Mộng cảnh, nhân vật bị xóa sổ (Game Over), trừ khi có các cổ trùng bảo mệnh đặc thù.

---

### 3. KIẾN TRÚC KỸ THUẬT (DÀNH CHO AI AGENT)

Dưới đây là phần Prompt để bạn giao cho AI (Golang/Next.js) triển khai cấu trúc Database và Backend Logic cho tính năng Phó Bản:

```markdown
# FEATURE IMPLEMENTATION: INHERITANCE & DREAM REALM INSTANCES (PHÓ BẢN)

We need to implement a Dungeon/Instance system for our Reverend Insanity text-based RPG. The instances are categorized into "Inheritance" (Truyền Thừa) and "Dream Realm" (Mộng Cảnh).

### 1. Database Schema (Golang/GORM)

Create models to handle Instances and Player Progress:

```go
type InstanceTemplate struct {
    ID              string `json:"id"`
    Name            string `json:"name"`
    Type            string `json:"type"` // "INHERITANCE_RIGHTEOUS", "INHERITANCE_DEMONIC", "DREAM_REALM", "BLESSED_LAND"
    RequiredRank    int    `json:"required_rank"`
    EntryItem       string `json:"entry_item"` // Item ID needed to enter (e.g., Token)
    MapData         JSONB  `json:"map_data"` // 2D Grid data for the instance
    Rewards         JSONB  `json:"rewards"` // Gu recipes, specific Gu, or Attainment points
}

type PlayerInstanceSession struct {
    ID              string `json:"id"`
    PlayerID        string `json:"player_id"`
    InstanceID      string `json:"instance_id"`
    CurrentX        int    `json:"current_x"`
    CurrentY        int    `json:"current_y"`
    Status          string `json:"status"` // "ACTIVE", "COMPLETED", "FAILED"
    TempInventory   JSONB  `json:"temp_inventory"` // Used if the instance restricts player's own Gu (like Dream Realms)
}
```

### 2. Backend Logic (Golang)

**A. Entry Logic (`POST /api/instance/enter`):**
- Check if the player is at the correct Global Map (X,Y) coordinates where the instance is located.
- Check `RequiredRank` and consume `EntryItem`.
- If `Type == "DREAM_REALM"`, verify if `Player.SoulFoundation > MinimumThreshold`.
- Create a `PlayerInstanceSession` and switch the player's active view to the Instance Map.

**B. Dream Realm Penalty (`POST /api/instance/dream-action`):**
- In Dream Realms, every action/combat turn MUST consume `Player.SoulFoundation` instead of normal HP.
- If `SoulFoundation <= 0`, trigger Permadeath logic (or kick them out with a severe coma debuff).

**C. Blessed Land Ownership (`POST /api/instance/claim-blessed-land`):**
- For Blessed Land instances, standard combat isn't enough. The player must submit the required "Land Spirit Obsession" condition (e.g., providing a specific Gu, answering a lore question correctly).
- If successful, transfer ownership of the instance to the Player.

### 3. Frontend UI (Next.js)
- When inside an instance, change the UI theme (e.g., Dark Red for Demonic Inheritances, Ethereal Purple for Dream Realms).
- Display a distinct "Instance Event Log" box. Instead of just combat text, render narrative text (e.g., "Bạn bước vào một mật thất, trên tường khắc một bài thơ... Hãy chọn cửa Trái hay Phải?").
```

### Tóm tắt chiến lược cho bạn:
Để bắt đầu, bạn hãy **làm loại Phó bản "Truyền Thừa Ma Đạo" trước**. Hãy thiết kế một phó bản giống *Truyền thừa Hoa Tửu Hành Giả*, nơi người chơi phải dùng sức mạnh phá đá, hoặc giải đố bằng text. Sau khi luồng đi ải này mượt mà, bạn mới đắp thêm cơ chế trừ Hồn Phách phức tạp của **Mộng Cảnh** vào. 