# Đặc Tả Kỹ Thuật: Hệ Thống Vạn Đạo Tương Xung & Trừng Phạt Đa Tu

**Mã Thiết Kế:** GuWorld-Spec-DaoConflict-2026  
**Liên kết Tracker:** [Map #176](https://github.com/VipPro67/GuWorld/issues/176)  
**Tiến độ Tickets:**
- [x] [#177: Khảo sát dữ liệu Đạo Ngân & Baseline](https://github.com/VipPro67/GuWorld/issues/177)
- [x] [#178: Mô hình toán học Xung đột Đa Tu & Độ thuần khiết](https://github.com/VipPro67/GuWorld/issues/178)
- [x] [#179: Cơ chế Mana Tax, Tỉ lệ thất bại & Phản phệ nội thương](https://github.com/VipPro67/GuWorld/issues/179)
- [x] [#180: Vô Thượng Đại Tông Sư giải trừ 100% & Tương Sinh](https://github.com/VipPro67/GuWorld/issues/180)
- [x] [#181: Dị Biến Tiên Khiếu & Kiếp Nạn Hỗn Loạn](https://github.com/VipPro67/GuWorld/issues/181)
- [x] [#182: Thiết kế tính năng & kinh tế học Thanh Đạo Tẩy Thể](https://github.com/VipPro67/GuWorld/issues/182)

---

## 1. Mục Tiêu & Triết Lý Cốt Lõi

1. **Khẳng định chân lý lore Cổ Chân Nhân:** "Chuyên tu là vương đạo, kiêm tu cần đại ngộ tương sinh, tạp tu ắt tự bạo diệt vong".
2. **Khắc phục toàn diện lỗ hổng cũ:** Loại bỏ hằng số làm phẳng cố định $+50,000$ và sàn bảo hộ $0.60$ khiến người chơi farm bừa bãi mọi loại đạo ngân mà không chịu trừng phạt.
3. **Trừng phạt đa chiều trong Combat Simulator:** Không chỉ suy giảm uy lực đòn đánh, người chơi tạp tu sẽ đối mặt với:
   - Sát thương suy giảm sâu (chạm sàn $0.08$ - mất tới $92\%$ uy lực).
   - Hao phí Tiên Nguyên gia tăng (Mana Tax).
   - Nguy cơ thôi động thất bại (Cast Failure / Fizzle).
   - Phản phệ nội thương (Internal Backfire: tự mất máu xuyên giáp, rối loạn chân nguyên).
4. **Bảo tồn cân bằng PVE:** Boss và Hoang Thú thiên sinh sở hữu đặc tính "Dị Thú Đạo Thể" (`HasNonConflictingDao = true`) để không tự bóp uy lực.

---

## 2. Mô Hình Toán Học Xung Đột (Quyết Định #178)

### 2.1. Chuẩn Hóa Mẫu Số Động (`Denominator`)
Thay thế $+50,000$ cũ bằng mẫu số phụ thuộc vào quy mô đạo ngân thực tế của nhân vật:
$$\text{Denominator} = \text{Amount}(p) + \max(1\,000, 0.1 \times \text{TotalMarks})$$
*Trong đó:*
- $\text{Amount}(p)$: Số Đạo Ngân của lưu phái thi triển.
- $\text{TotalMarks}$: Tổng toàn bộ Đạo Ngân trên nhân vật.

### 2.2. Hệ Số Phạt Đa Tu Lũy Tiến ($F_{\text{Multi}}$)
Với $N$ là số lưu phái ngoại lai vượt ngưỡng an toàn ($> 2\%$ tổng Đạo Ngân) và $P = \frac{\text{Amount}(p)}{\text{TotalMarks}}$ là Độ Thuần Khiết:
$$F_{\text{Multi}} = 1.0 + (N - 1)^{1.3} \times (1.0 - P)^2 \times 4.0$$

*Tính chất:*
- **Đơn tu thuần ($N=1, P=1.0$):** $F_{\text{Multi}} = 1.0$ (0% phạt).
- **Song tu tương sinh ($N=2, P=0.70$):** $F_{\text{Multi}} \approx 1.36$ (Phạt nhẹ, hóa giải được bằng Tông Sư).
- **Tạp tu 10 phái ($N=10, P=0.10$):** $F_{\text{Multi}} \approx 1.0 + 17.15 \times 0.81 \times 4.0 = 56.57$ (Xung đột bị khuếch đại gấp 56 lần).

### 2.3. Tỷ Lệ Xung Đột & Hệ Số Uy Lực (`conflictMult`)
$$\text{effConflict} = \text{rawConflict} \times F_{\text{Multi}} \times (1.0 - \text{RealmControl})$$
$$\text{conflictRatio} = \frac{\text{effConflict}}{\text{Denominator}}$$
$$\text{conflictMult} = \max\left(0.08, \frac{1.0}{1.0 + \text{conflictRatio} \times 0.8 \times 2.0}\right)$$

---

## 3. Đặc Tả Tác Động Trong Combat Simulator (Quyết Định #179)

### 3.1. Hệ Số Hao Phí Tiên Nguyên (Mana Tax)
Khi Cổ Tiên thôi động sát chiêu thuộc lưu phái $p$:
$$\text{ManaTaxMultiplier} = 1.0 + (1.0 - \text{conflictMult}) \times 1.5$$
$$\text{EffectiveManaCost} = \text{BaseManaCost} \times \text{ManaTaxMultiplier}$$
- Đơn tu (`conflictMult = 1.0`): $1.0\times$ (Không tăng).
- Song tu tương khắc (`conflictMult = 0.50`): $1.75\times$ ($+75\%$).
- Tạp tu cực đoan (`conflictMult = 0.08`): $2.38\times$ ($+138\%$).

### 3.2. Tỉ Lệ Thôi Động Thất Bại (Cast Failure / Skill Fizzle)
- **Điều kiện kích hoạt:** $\text{conflictMult} < 0.70$.
- **Công thức xác suất:**
  $$\text{CastFailureChance} = \min\left(0.25, \frac{0.70 - \text{conflictMult}}{0.70 - 0.08} \times 0.25\right)$$
- **Hành vi:** Tiên Nguyên vẫn bị trừ theo `EffectiveManaCost`, nhưng sát chiêu hoàn toàn không phát động ra ngoài (0 sát thương, 0 hiệu ứng debuff/buff).
- **Log:** `[Thôi Động Thất Bại] Đạo ngân dị chủng nhiễu loạn chân nguyên, sát chiêu [%s] tán loạn tiêu biến!`

### 3.3. Phản Phệ Nội Thương (Internal Backfire)
- **Điều kiện kích hoạt:** $\text{conflictMult} < 0.60$.
- **Công thức xác suất:**
  $$\text{BackfireChance} = \min\left(0.30, \frac{0.60 - \text{conflictMult}}{0.60 - 0.08} \times 0.30\right)$$
- **Hành vi khi kích hoạt:**
  1. Người thi triển tự nhận lượng sát thương chuẩn (True Damage):
     $$\text{BackfireDamage} = \text{MaxHP} \times \text{Random}(0.08, 0.12)$$
     *(Xuyên qua Khiên Ảo và Phòng ngự, trừ trực tiếp vào HP).*
  2. Nếu đòn đánh không bị gãy chiêu (vượt qua kiểm tra Cast Failure), uy lực sát chiêu vẫn bị giảm thêm $50\%$.
- **Log:** `[Phản Phệ Nội Thương] Quy tắc dị hệ xung đột dữ dội! %s chịu %d sát thương nội thương do phản phệ!`

---

## 4. Đặc Tả Vô Thượng Đại Tông Sư & Quy Tắc Tương Sinh (Quyết Định #180)

### 4.1. Hóa Giải Tuyệt Đối 100% Cục Bộ (Vô Thượng Đại Tông Sư)
- **Định nghĩa:** Khi Cổ Tiên đạt cảnh giới Vô Thượng Đại Tông Sư (Level 4, `AttainmentLevel = 4`) tại lưu phái $p$:
  $$\text{RealmControlForLevel}(4) = 1.00 \quad (100\%)$$
- **Quy tắc thi triển:**
  - Khi xuất chiêu thuộc lưu phái $p$:
    $$\text{effConflict} = \text{rawConflict} \times F_{\text{Multi}} \times (1.0 - 1.00) = 0.0$$
    $$\implies \text{conflictRatio} = 0.0 \implies \text{conflictMult} = 1.00$$
    *(Hoàn toàn miễn nhiễm xung đột: Sát thương bảo lưu 100%, Mana Tax = 1.0x, Tỉ lệ thất bại = 0%, Phản phệ = 0%).*
  - Khi xuất chiêu thuộc một lưu phái khác $q$ mà nhân vật chưa đạt Vô Thượng: Vẫn chịu xung đột và ma sát quy tắc theo cấp bậc Cảnh giới thực tế của phái $q$.
  *(Chỉ có thể chất Chí Tôn Tiên Thai mới có đặc tính Đạo Ngân Bất Tương Xung toàn diện cho tất cả lưu phái).*

### 4.2. Quy Tắc Tương Sinh & Số Phái Bất Khả Dung Hiệu Dụng ($N_{\text{eff}}$)
Thay thế số đếm nguyên thô $N$ bằng biến số thực liên tục $N_{\text{eff}}$ dựa trên ma trận tương thích 49 lưu phái:
$$N_{\text{eff}} = 1.0 + \sum_{q \neq p, \, \text{amt}(q) > \text{grace}} 2.0 \times (1.0 - \text{Compatibility}(p, q)) \times \text{MasterySynergyDiscount}$$

*Trong đó:*
- $\text{Compatibility}(p, q)$: Điểm tương thích giữa 2 phái (từ `0.55` đến `0.98`).
  - Cặp Tương Sinh / Đồng Nguyên ($\text{Comp} \ge 0.95$, e.g. Mộc - Hỏa, Thủy - Mộc, Không Gian - Thời Gian):
    $(1.0 - 0.95) \times 2.0 = 0.1 \implies$ mỗi phái phụ chỉ đóng góp $+0.1$ vào $N_{\text{eff}}$.
  - Cặp Tương Khắc ($\text{Comp} \le 0.60$, e.g. Thủy - Hỏa, Quang - Ám):
    $(1.0 - 0.55) \times 2.0 = 0.9 \implies$ đóng góp trọn vẹn $+0.9$ vào $N_{\text{eff}}$.
- **Hệ số "Vạn Đạo Tương Thông" ($\text{MasterySynergyDiscount}$):**
  - Cảnh giới Phổ thông (L0) hoặc Đại Sư (L1) ở phái chính: $\text{MasterySynergyDiscount} = 1.0$ (chưa đủ khả năng dung hòa tương sinh).
  - Cảnh giới Tông Sư (L2) trở lên ở phái chính: $\text{MasterySynergyDiscount} = 0.50$ (đối với các phái có $\text{Comp} \ge 0.90$, ma sát giảm thêm 50%, giúp $N_{\text{eff}} \approx 1.0$).

---

## 5. Đặc Tả Dị Biến Tiên Khiếu & Kiếp Nạn Hỗn Loạn (Quyết Định #181)

### 5.1. Suy Thoái Sinh Thái Tiên Khiếu (`EcologyPurityFactor`)
- **Ngưỡng an toàn:** Độ Thuần Khiết $P = \frac{\text{DominantMarks}}{\text{TotalMarks}} \ge 0.70 \implies \text{EcologyPurityFactor} = 1.00$ (Sản lượng $100\%$, sinh thái ổn định).
- **Công thức suy thoái khi $P < 0.70$:**
  $$\text{EcologyPurityFactor} = \max\left(0.25, \, 1.0 - (0.70 - P) \times 1.25\right)$$
  - $P = 0.50$ (Song tu tạp): Hệ số $= 0.75$ (Giảm $25\%$ sản lượng).
  - $P = 0.30$ (Tứ tu): Hệ số $= 0.50$ (Giảm $50\%$ sản lượng).
  - $P = 0.10$ (Tạp tu 10 phái): Hệ số $= 0.25$ (Sàn giảm $75\%$ sản lượng: linh khí cạn kiệt, khoáng mạch đứt gãy).
- **Phạm vi tác động:**
  1. Sản lượng thu hoạch từ các mỏ tài nguyên (`computeApertureNodeRewardWithCtx`).
  2. Sản lượng Tiên Nguyên tự nhiên sinh ra hàng năm (`essence_production_per_year` / `applyApertureProjection`).

### 5.2. Kích Hoạt Kiếp Nạn Hỗn Loạn (Chaotic / Hybrid Tribulation)
- **Điều kiện kích hoạt:** $P < 0.60$ VÀ $N_{\text{eff}} \ge 2.5$.
- **Đặc điểm tai kiếp biến dị:**
  1. **Lai tạp đa thuộc tính (Hybrid Elements):** Tai kiếp kết hợp 2-3 lưu phái tương khắc mạnh nhất hiện có trong Tiên Khiếu (ví dụ: *Băng Hỏa Lôi Kiếp*, *Ám Độc Hồn Kiếp*).
  2. **Khuếch đại uy lực Quái Kiếp Nạn (Calamity Monster Power):**
     $$\text{CalamityPowerMultiplier} = 1.0 + (1.0 - P) \times 0.75 \quad (+30\% \text{ đến } +67.5\%)$$
     *(Áp dụng trực tiếp vào HP và Công kích của quái vật tai kiếp).*
  3. **Tàn phá Tiên Khiếu gia tăng:** Quái tai kiếp ưu tiên dồn hỏa lực phá hủy các node tài nguyên và gây sát thương x2 lên độ bền Tiên Khiếu (`aperture_hp`).

### 5.3. Quy Tắc Vòng Xoáy Tha Hóa (Degeneracy Spiral)
- Vượt qua kiếp nạn hỗn loạn, phần thưởng Đạo Ngân rải đều theo các hệ cấu thành tai kiếp (thay vì dồn 70% cho phái chính).
- Càng tạp tu $\implies$ kiếp nạn càng hỗn loạn $\implies$ phần thưởng nhận về càng nhiều tạp đạo $\implies$ càng lún sâu vào vũng lầy suy thoái, trừ khi Cổ Tiên chủ động sử dụng cơ chế Thanh Đạo Tẩy Thể.

### 5.4. Đặc Quyền Thể Chất Chí Tôn Tiên Thai
- Miễn nhiễm hoàn toàn với suy thoái sản lượng sinh thái ($\text{EcologyPurityFactor} = 1.00$ bất kể quy mô và số lượng phái).
- Vẫn phải đối mặt với kiếp nạn hỗn loạn siêu cấp (đúng chuẩn lore nguyên tác: kiếp nạn của Chí Tôn Tiên Khiếu mang quy mô vạn đạo cuồng bạo).

---

## 6. Đặc Tả Cơ Chế & Kinh Tế Học Thanh Đạo Tẩy Thể (Quyết Định #182)

### 6.1. Nguyên Mẫu Trực Quan (Interactive Prototype)
Đã triển khai và lưu trữ tại:
📄 [`docs/prototypes/dao_cleansing_prototype.html`](file:///home/vippro67/GuWorld/docs/prototypes/dao_cleansing_prototype.html)
*(Mô phỏng đầy đủ phân bổ Đạo Ngân, tính toán thời gian thực $P, N_{\text{eff}}, F_{\text{Multi}}$, Mana Tax, Fizzle, Backfire, và giao diện bóc tách tẩy rửa Đạo Ngân).*

### 6.2. Mô Hình Chi Phí & Thu Hồi Kinh Tế
1. **Chi phí bóc tách Đạo Ngân tạp:**
   - Tỷ lệ chuẩn hóa: **$1$ Tiên Nguyên Thạch cho mỗi $100$ Đạo Ngân tạp**.
   - Công thức: $\text{StoneCost} = \lceil \frac{\text{CleansedMarks}}{100} \rceil$.
   - Ví dụ: Tẩy $1,000$ Đạo Ngân = $10$ Tiên Nguyên Thạch; Tẩy $10,000$ Đạo Ngân = $100$ Tiên Nguyên Thạch.
2. **Thu hồi tài nguyên (Resource Refund):**
   - Hoàn trả **$30\%$ giá trị** Đạo Ngân bị bóc tách dưới dạng **Đạo Tài (Dao Materials)** thuộc lưu phái tương ứng:
     $$\text{RefundMaterials} = \lfloor \text{CleansedMarks} \times 0.30 \rfloor$$
   - Đạo Tài thu hồi được dùng trực tiếp để nuôi Cổ trùng, luyện Cổ, hoặc bán đổi lấy tài nguyên bồi dưỡng cho lưu phái chính.
3. **Nhịp độ & Giới hạn bế quan:**
   - Cooldown: **$12$ Giờ** giữa mỗi lần đại tẩy thể.
   - Hạn mức mỗi lần tẩy: Tối đa **$10,000$ Đạo Ngân / lần**.
   - Cho phép tiêu hao thêm Tiên Nguyên Thạch để hoàn thành bế quan ngay lập tức nếu cần khẩn cấp.

### 6.3. Giao Diện & Cảnh Báo Trực Quan (UI/UX)
1. **Thanh Trạng Thái Độ Thuần Khiết Tiên Khiếu (Aperture Purity Badge):**
   - 🟢 **Xanh lục ($P \ge 70\%$):** "Bản Nguyên Thuần Khiết (An Toàn)" — Hệ số xung đột $1.0\times$, sinh thái tối ưu.
   - 🟡 **Vàng ($50\% \le P < 70\%$):** "Quy Tắc Tiên Khiếu Nhiễu Loạn" — Cảnh báo suy giảm sản lượng sinh thái và gia tăng hao phí Tiên Nguyên.
   - 🔴 **Đỏ ($P < 50\%$):** "⚠️ VẠN ĐẠO TƯƠNG XUNG NGUY CẤP" — Cảnh báo đỏ rực: Sát chiêu nguy cơ gãy vụn ($25\%$), phản phệ nội thương ($30\%$), và Kiếp Nạn Hỗn Loạn đe dọa phá hủy khiếu!
2. **Modal "Thanh Đạo Tẩy Thể":**
   - Cho phép chọn danh sách lưu phái dị chủng (tự động loại trừ phái chính).
   - Thanh trượt Slider chọn số lượng Đạo Ngân cần bóc tách.
   - Bảng tính toán trước (Live Preview): Chi phí Tiên Nguyên Thạch, số Đạo Tài nhận lại, và mức tăng của Độ Thuần Khiết $P$ sau khi xác nhận.

---

## 7. Lộ Trình Triển Khai Kỹ Thuật (Implementation Roadmap)

1. **Phase 1 (Backend Core - GameFX & Profile):**
   - Cập nhật `dao_combat_profile.go`: Chuẩn hóa mẫu số động, tích hợp $N_{\text{eff}}, F_{\text{Multi}}$, hạ sàn `0.08`, cập nhật `RealmControlForLevel(4) = 1.00`, và thêm `HasNonConflictingDao = true` cho Boss.
2. **Phase 2 (Simulator Engine):**
   - Cập nhật `engine.go`: Áp dụng `ManaTaxMultiplier`, kiểm tra `CastFailureChance` (gãy chiêu), và kích hoạt `BackfireChance` (sát thương chuẩn % MaxHP).
3. **Phase 3 (Immortal Aperture & Tribulations):**
   - Cập nhật `aperture_collect.go`: Áp dụng `EcologyPurityFactor` vào sản lượng tài nguyên và Tiên Nguyên.
   - Cập nhật `tribulation_worker.go`: Logic sinh kiếp nạn hỗn loạn và phân tán đạo ngân thưởng khi $P < 0.60$.
4. **Phase 4 (Thanh Đạo Tẩy Thể Handler & UI):**
   - Thêm handler `POST /api/aperture/cleanse-dao-marks`.
   - Cập nhật giao diện `ImmortalApertureView.tsx` với Purity Badge và Cleansing Modal.

---

## 8. Trạng Thái Triển Khai (Implementation Status)

- [x] **Phase 1:** `gamefx/dao_combat_profile.go` — mẫu số động, $N_{\text{eff}}$, $F_{\text{Multi}}$, sàn `0.08`, `RealmControlForLevel(4)=1.00`, helper `ManaTaxMultiplier`/`CastFailureChance`/`BackfireChance`/`DaoPurity`/`EcologyPurityFactor`/`EffectivePathCount`/`IsChaoticTribulation`; Boss/Hoang Thú `HasNonConflictingDao=true` (`combat_auto.go`).
- [x] **Phase 2:** `simulator/engine.go` — Mana Tax, Fizzle (`DAO_CAST_FAILED`), Backfire (`DAO_BACKFIRE`, true damage + giảm 50% uy lực), log tương ứng.
- [x] **Phase 3:** `aperture_collect.go` + `services/player_service.go` + `tribulation_trigger.go` — `EcologyPurityFactor` áp vào sản lượng node và Tiên Nguyên; kiếp nạn `CHAOTIC_TRIBULATION` (khuếch đại HP/Công quái, path lai tạp, rải đều Đạo Ngân thưởng).
- [x] **Phase 4:** `handlers/aperture_cleanse.go` + route `POST /api/immortal-aperture/cleanse-dao-marks` + cột `immortal_apertures.last_cleanse_at` + UI Purity Badge & Cleansing Modal (`ImmortalApertureView.tsx`, `apertureSlice.ts`).
- [ ] **Hoãn:** tuỳ chọn "hoàn thành bế quan ngay" bằng Tiên Nguyên Thạch (chưa cần thiết cho v1).




