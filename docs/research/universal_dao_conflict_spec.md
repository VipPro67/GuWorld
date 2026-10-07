# Đặc Tả Kỹ Thuật: Hệ Thống Vạn Đạo Tương Xung & Trừng Phạt Đa Tu

**Mã Thiết Kế:** GuWorld-Spec-DaoConflict-2026  
**Liên kết Tracker:** [Map #176](https://github.com/VipPro67/GuWorld/issues/176)  
**Tiến độ Tickets:**
- [x] [#177: Khảo sát dữ liệu Đạo Ngân & Baseline](https://github.com/VipPro67/GuWorld/issues/177)
- [x] [#178: Mô hình toán học Xung đột Đa Tu & Độ thuần khiết](https://github.com/VipPro67/GuWorld/issues/178)
- [x] [#179: Cơ chế Mana Tax, Tỉ lệ thất bại & Phản phệ nội thương](https://github.com/VipPro67/GuWorld/issues/179)
- [ ] [#180: Vô Thượng Đại Tông Sư giải trừ 100% & Tương Sinh](https://github.com/VipPro67/GuWorld/issues/180)
- [ ] [#181: Dị Biến Tiên Khiếu & Kiếp Nạn Hỗn Loạn](https://github.com/VipPro67/GuWorld/issues/181)
- [ ] [#182: Thiết kế tính năng & kinh tế học Thanh Đạo Tẩy Thể](https://github.com/VipPro67/GuWorld/issues/182)

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
