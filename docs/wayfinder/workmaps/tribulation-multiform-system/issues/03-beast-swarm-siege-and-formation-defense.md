# Ticket 03: Cơ Chế Thú Triều, Trùng Họa & Thủ Hộ Trận Pháp Đa Đợt

Type: prototype
Status: resolved
Blocked by: none
Assignee: Antigravity

## Question

Làm thế nào để xây dựng cơ chế **Thú Triều Xâm Lăng (`BEAST_SWARM`)**, biến việc độ kiếp thành một trận chiến phòng thủ cứ điểm (Tower Defense / Siege Defense) sinh động trên lưới ô Tiên Khiếu với sự tham gia của Tháp Trận Pháp, Quân Đội Dị Nhân và Cổ Tiên?

Cụ thể cần giải quyết:
1. **Quy luật Tiến Công Theo Đợt (Wave Spawning & Pathfinding)**:
   - Thú triều xuất hiện từ các cổng hư không ở rìa bản đồ, chia làm 3–5 đợt (Waves) có quy mô tăng dần, dẫn đầu bởi Hoang thú / Dị chủng thú vương.
   - Cơ chế nhắm mục tiêu (Target Priority): Quái vật ưu tiên di chuyển tấn công các Mỏ Điểm Tài Nguyên (`ApertureResourcePoint`) và Khu Dân Cư Dị Nhân (`settlements`) trước khi hướng tới Tiên Khiếu Hạch Tâm (`Core Node`).
2. **Cơ chế Tác Chiến Phòng Thủ Đa Tầng**:
   - **Tầng 1 - Tháp Trận Pháp**: Tự động bắn tỉa hoặc phát ra sóng xung kích làm chậm/sát thương quái vật khi chúng tiến vào tầm bắn.
   - **Tầng 2 - Lực Lượng Dị Nhân / Hộ Vệ (`aperture_npcs`)**: Đứng tại các chốt chặn để cản bước và giao tranh tiêu hao sinh lực thú triều.
   - **Tầng 3 - Cổ Tiên / Tiên Cổ Phòng Chặn Đánh**: Người chơi có thể tự do xuất kích chặn đầu, kích hoạt trận chiến nhanh hoặc nã pháo Trận Kích từ Tiên Cổ Phòng để tiêu diệt các cụm quái chủ lực.
3. **Kết toán Thiệt Hại & Bắt Sống Thú Hoang**:
   - Nếu để quái vật tràn vào tài nguyên: điểm tài nguyên bị giảm trữ lượng hoặc mất vĩnh viễn sản lượng mùa vụ.
   - Cơ chế Nô Đạo / Thuần Hóa: Cổ Tiên chuyên tu Nô Đạo có thể dùng Cổ Trùng thu phục một phần dị thú làm hộ vệ hoặc vật nuôi cho Tiên Khiếu.

## Answer

Đã chuẩn hóa toàn diện cơ chế vận hành của dạng tai kiếp **Thú Triều Xâm Lăng (`BEAST_SWARM`)**, kết hợp giữa thuật toán tìm đường nhắm mục tiêu có tôn ti, hệ thống phòng thủ cứ điểm đa tầng và cơ chế Nô Đạo bắt sống thú hoang:

### 1. Quy Luật Tiến Công Theo Đợt (Wave Spawning & Pathfinding AI)
- **Cổng Xuất Hiện (Spawn Ingress Points)**:
  - Xuất hiện từ 4 vết nứt không gian tại 4 góc biên bản đồ $20 \times 20$: Tây Bắc $(0, 0)$, Đông Bắc $(19, 0)$, Tây Nam $(0, 19)$, Đông Nam $(19, 19)$.
- **Cấu Trúc Các Đợt Tấn Công (Wave Hierarchy)**:
  - *Rank 6 (Địa Tai)*: 3 Waves.
    - Wave 1: 4–6 Dị Thú Tiên Phong (Thiết Bì Lang, Cuồng Đao Trư).
    - Wave 2: 8 Minion + 2 Thú Đầu Đàn (Cuồng Huyết Báo, Huyết Giáp Trùng).
    - Wave 3: Thượng Cổ Hoang Thú Boss (Thiết Bì Tê Vương) dẫn đầu bầy thú tinh nhuệ.
  - *Rank 7 (Thiên Kiếp)*: 4 Waves (kèm thú bay phớt lờ chướng ngại vật mặt đất và quái đào đất phá hoại ngầm).
  - *Rank 8–9 (Hạo Kiếp & Vạn Kiếp)*: 5 Waves (Thái Cổ Hoang Thú, Niên Thú, Thao Thiết nuốt chửng linh thổ).
- **Thuật Toán Nhắm Mục Tiêu Có Tôn Ti (Target Priority Hierarchy)**:
  1. *Ưu Tiên 1*: Mỏ Điểm Tài Nguyên (`ApertureResourcePoint`) gần nhất nằm trên hướng di chuyển (thèm khát cắn nuốt tiên khoáng, linh thảo để tiến hóa).
  2. *Ưu Tiên 2*: Khu Dân Cư Dị Nhân (`settlements`) (huyết thú cắn xé sinh linh).
  3. *Ưu Tiên Tối Hậu*: Tiên Khiếu Hạch Tâm (`Core Node` tại trung tâm) - nếu thú triều tràn vào Hạch Tâm, gây trừ trực tiếp $-200$ HP vách khiếu mỗi lượt.
- **Quy Tắc Di Chuyển**: Mỗi tick bước (5–10s), quái tiến 1 ô theo hướng ngắn nhất (Manhattan distance / A*). Nếu gặp Tháp Trận, Hộ Vệ hoặc Cổ Tiên cản đường, quái vật dừng lại giao chiến phá hủy chướng ngại vật trước khi đi tiếp.

---

### 2. Hệ Thống Tác Chiến Phòng Thủ Đa Tầng (Multi-tier Defense)
- **Tầng 1 - Tháp Trận Pháp Phòng Thủ (`DEFENSE` nodes trên grid)**:
  - Bán kính tầm bắn: $R = 2$ đến $3$ ô.
  - Tự động tìm mục tiêu có HP cao nhất trong tầm, xả tiễn/sét trừ $-150$ HP quái vật mỗi lượt.
  - Tiêu hao Tiên Nguyên Thạch tích trữ trong kho Tiên Khiếu để duy trì đạn dược.
- **Tầng 2 - Quân Đội Dị Nhân & NPC Hộ Vệ (`aperture_npcs`, `aperture_variant_humans`)**:
  - Huấn luyện Dị nhân (Thạch Nhân, Mao Dân, Đản Dân) đóng tại các chốt chặn hiểm yếu.
  - Cản bước di chuyển của quái vật (Movement Blocking), trao đổi sát thương cận chiến.
  - Nếu NPC Hộ Vệ tử trận: tổn thất $-50$ dân số của bản dị nhân, giảm điểm hạnh phúc và tăng áp bức.
- **Tầng 3 - Cổ Tiên & Tiên Cổ Phòng Oanh Kích Chủ Động**:
  - *Chặn đánh trực tiếp*: Cổ Tiên di chuyển đến ô có cụm quái, kích hoạt giao tranh nhanh để quét sạch mục tiêu.
  - *Tiên Cổ Phòng Pháo Kích (Artillery Bombardment)*: Ra lệnh Tiên Cổ Phòng nã Trận Kích diện rộng $3 \times 3$ từ xa, tiêu diệt ngay lập tức các cụm quái thú mà không cần rời vị trí tọa trấn trung tâm.

---

### 3. Kết Toán Thiệt Hại & Cơ Chế Bắt Sống Nô Đạo (Enslavement Capture)
- **Tổn Thất Tài Nguyên Nếu Bị Tràn Vào**:
  - Mỗi lượt quái vật đứng trên ô tài nguyên, nó cắn nuốt trữ lượng (`stock`) và trừ $-20$ điểm `vitality`. Nếu vitality về 0, mỏ sụp đổ ngừng sản xuất.
- **Cơ Chế Bắt Sống Thú Hoang (Nô Đạo Thuần Hóa)**:
  - Dành cho Cổ Tiên tu luyện Nô Đạo hoặc mang theo Cổ Nô Dịch phù hợp trong túi đồ.
  - Khi quái vật hoặc Hoang thú bị đánh hạ xuống ngưỡng sinh lực $HP \le 30\%$, xuất hiện hành động: **Thi Triển Nô Dịch**.
  - Bắt sống thành công: Quái vật quy phục, không biến mất mà được đưa vào hệ thống **Thú Nuôi Hộ Khiếu (Aperture Guardian Beast)** hoặc bổ sung vào các Mỏ Chăn Nuôi (`khu_nuoi`) để thu hoạch tài nguyên dài hạn.
- **Rơi Rớt Kiếp Tài Hậu Chiến**:
  - Xác quái vật bị tiêu diệt kết tinh thành các nguyên liệu da lông, cốt tủy quý hiếm (`wolf_king_fang`, `beast_bone_shard`, `metal_ore_rank6`, `blood_crystal_rank7`...).

---

### 4. Kiểm Chứng Qua Prototype HTML
- Đã xây dựng hoàn chỉnh bản thử nghiệm tương tác độc lập tại:
  [`prototypes/03_beast_swarm_defense_demo.html`](../prototypes/03_beast_swarm_defense_demo.html).
- Kiểm chứng trơn tru: Spawning 3 đợt thú triều từ 4 góc, AI di chuyển theo tôn ti mục tiêu, Tháp Trận tự động bắn tỉa, Hộ Vệ chặn cửa giáp lá cà, Tiên Cổ Phòng nã pháo diện rộng $3 \times 3$, và cơ chế Nô Đạo bắt sống thú hoang khi $HP \le 30\%$.
