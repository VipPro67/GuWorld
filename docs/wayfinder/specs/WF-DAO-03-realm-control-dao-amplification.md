# [SPECIFICATION] Công Thức Điều Hòa Cảnh Giới, Xung Đột Nội Tại & Khuếch Đại Đạo Ngân

**Ticket:** [WF-DAO-03: Thiết kế công thức Điều Hòa Cảnh Giới (Realm Control) & Khuếch Đại Đạo Ngân](../tickets/WF-DAO-03-formula-realm-control-and-dao-amplification.md)  
**Tác giả:** Antigravity  
**Ngày hoàn thành:** 2026-09-20  

---

## 1. Công Thức Tính Xung Đột Thô (Raw Conflict Score)

Khi nhân vật thi triển sát chiêu/Cổ trùng thuộc lưu phái $P_{\text{main}}$:
- Tổng đạo ngân toàn thân: $M_{\text{total}} = \sum_{P} M(P)$
- Ngưỡng miễn trừ (Grace Threshold): $T_{\text{grace}} = \max\left(1000, M_{\text{total}} \times 0.02\right)$

$$\text{RawConflict}(P_{\text{main}}) = \sum_{\substack{P_{\text{other}} \ne P_{\text{main}} \\ M(P_{\text{other}}) > T_{\text{grace}}}} M(P_{\text{other}}) \times \left(1.0 - \text{Compatibility}(P_{\text{main}}, P_{\text{other}})\right)$$

Trong đó $\text{Compatibility}(P_A, P_B)$ lấy từ Danh sách Ghi đè (Pair Overrides) hoặc Ma trận Nhóm ($10 \times 10$) từ [WF-DAO-02 Spec](./WF-DAO-02-dao-groups-compatibility.md).

---

## 2. Cơ Chế Điều Hòa Cảnh Giới (Realm Control)

Cảnh giới $L \in \{0, 1, 2, 3, 4\}$ của $P_{\text{main}}$ (`players.path_masteries[mainPath]` thông qua `gamefx.PathMasteryLevel`) đại diện cho khả năng chế ngự và điều hòa thiên địa quy tắc:

| Cảnh Giới (Realm Level) | Tên Cảnh Giới | Chân Ý Tối Thiểu | Hệ Số Điều Hòa ($\text{RealmControl}$) |
|---|---|---|---|
| **Level 0** | Phàm Thai (Ordinary) | $0$ | **$0\%$** (Không thể điều hòa) |
| **Level 1** | Đại Sư (Master) | $100$ | **$10\%$** |
| **Level 2** | Tông Sư (Grandmaster) | $1.000$ | **$25\%$** |
| **Level 3** | Đại Tông Sư (Great Grandmaster) | $10.000$ | **$45\%$** |
| **Level 4** | Vô Thượng Đại Tông Sư (Supreme) | $100.000$ | **$80\%$** |

$$\text{EffectiveConflict} = \text{RawConflict} \times (1.0 - \text{RealmControl})$$

---

## 3. Tỷ Lệ Xung Đột & Hệ Số Giảm Trừ Nội Tại (Internal Conflict Multiplier)

### 3.1. Tỷ Lệ Xung Đột ($\text{ConflictRatio}$):
$$\text{ConflictRatio} = \frac{\text{EffectiveConflict}}{M(P_{\text{main}}) + S_{\text{conflict}}}$$
- Hằng số co giãn: $S_{\text{conflict}} = 50.000$ (tương đương quy mô Đạo ngân chuẩn của Tiên nhân Rank 7).

### 3.2. Hệ Số Xung Đột Nội Tại ($\text{ConflictMultiplier}$):
$$\text{ConflictMultiplier} = \max\left(0.60, \frac{1.0}{1.0 + \text{ConflictRatio} \times \text{SkillSensitivity} \times K}\right)$$
- $K = 2.0$ (Hệ số điều chỉnh độ dốc xung đột).
- $\text{SkillSensitivity} \in [0.3, 1.0]$:
  - Đòn đánh cơ bản / Cổ cấp thấp: $0.4$
  - Sát chiêu tiêu chuẩn: $0.8$
  - Sát chiêu phức hợp / Thần thông tối thượng: $1.0$
- **Sàn bảo hộ (Floor Cap):** $\text{ConflictMultiplier} \ge 0.60$ (Hình phạt xung đột không bao giờ vượt quá $40\%$).

---

## 4. Công Thức Khuếch Đại Đạo Ngân (Dao Amplification) Mới

Khắc phục tình trạng tăng tuyến tính vô hạn làm lạm phát số liệu ở late-game, áp dụng hàm tuyến tính 2 đoạn (Piecewise Linear Scaling) mượt mà:

$$\text{DaoAmp}(M) = \begin{cases} 
1.0 + \dfrac{M}{100.0} & \text{khi } M \le 10.000 \\[10pt]
101.0 + \dfrac{M - 10.000}{250.0} & \text{khi } M > 10.000 
\end{cases}$$

### Bảng Mốc So Sánh:
- $1.000$ Đạo ngân (Rank 6 Sơ kỳ): $\text{DaoAmp} = \mathbf{\times 11.0}$
- $10.000$ Đạo ngân (Rank 6 Đỉnh phong / Rank 7): $\text{DaoAmp} = \mathbf{\times 101.0}$
- $50.000$ Đạo ngân (Rank 7 Cao kỳ): $\text{DaoAmp} = \mathbf{\times 261.0}$
- $100.000$ Đạo ngân (Rank 8 Tiêu chuẩn): $\text{DaoAmp} = \mathbf{\times 461.0}$
- $500.000$ Đạo ngân (Rank 8 Đỉnh phong / Á Tôn): $\text{DaoAmp} = \mathbf{\times 2061.0}$

---

## 5. Hiệu Suất Cảnh Giới (Realm Efficiency)

Cảnh giới cao giúp khai thác sâu sắc hơn uy lực của từng Đạo ngân:

| Cảnh Giới (Realm Level) | Hiệu Suất Cảnh Giới ($\text{RealmEfficiency}$) |
|---|---|
| **Level 0 (Ordinary)** | $\times 1.00$ |
| **Level 1 (Master)** | $\times 1.05$ |
| **Level 2 (Grandmaster)** | $\times 1.12$ |
| **Level 3 (Great Grandmaster)** | $\times 1.25$ |
| **Level 4 (Supreme Grandmaster)** | $\times 1.45$ |

---

## 6. Tổng Hợp Công Thức Sát Thương / Hiệu Ứng Công Kích Cuối Cùng

$$\text{FinalPower} = (\text{BaseSkillPower} + \text{BaseAttack}) \times \text{DaoAmp}(P_{\text{main}}) \times \text{RealmEfficiency} \times \text{ConflictMultiplier} \times \text{PathDamageBonus}$$

---

## 7. Phân Tích Thực Tế Các Kịch Bản Build (Case Studies)

### Kịch bản 1: Chuyên Tu Hỏa Đạo (Pure Specialist)
- **Thông số:** $50.000$ Hỏa đạo ngân, $0$ đạo ngân khác. Cảnh giới: Đại Tông Sư (Level 3).
- **Tính toán:**
  - $\text{RawConflict} = 0 \implies \text{ConflictMultiplier} = 1.00$
  - $\text{DaoAmp} = 261.0$
  - $\text{RealmEfficiency} = 1.25$
  - $\implies \text{Final Multiplier} = 261.0 \times 1.25 \times 1.00 = \mathbf{326.25}$ (Sát thương thuần cực cao, tối đa hóa hệ chính).

### Kịch bản 2: Song Tu Hỏa - Thủy Chưa Có Cảnh Giới (Uncontrolled Dual Build)
- **Thông số:** $50.000$ Hỏa, $50.000$ Thủy. Cảnh giới: Phàm Thai (Level 0).
- **Tính toán:**
  - $\text{RawConflict} = 50.000 \times (1.0 - 0.55) = 22.500$
  - $\text{RealmControl} = 0\% \implies \text{EffectiveConflict} = 22.500$
  - $\text{ConflictRatio} = 22.500 / 100.000 = 0.225$
  - $\text{ConflictMultiplier} = 1.0 / (1.0 + 0.225 \times 0.8 \times 2.0) = \mathbf{0.735}$ (Mất $26.5\%$ uy lực)
  - $\implies \text{Final Multiplier} = 261.0 \times 1.00 \times 0.735 = \mathbf{191.83}$

### Kịch bản 3: Song Tu Hỏa - Thủy Đạt Đại Tông Sư Cả Hai Phái (Controlled Dual Build)
- **Thông số:** $50.000$ Hỏa, $50.000$ Thủy. Cảnh giới: Đại Tông Sư Hỏa (Level 3) & Đại Tông Sư Thủy (Level 3).
- **Tính toán:**
  - $\text{RawConflict} = 22.500$
  - $\text{RealmControl} = 45\% \implies \text{EffectiveConflict} = 22.500 \times 0.55 = 12.375$
  - $\text{ConflictRatio} = 12.375 / 100.000 = 0.12375$
  - $\text{ConflictMultiplier} = 1.0 / (1.0 + 0.12375 \times 0.8 \times 2.0) = \mathbf{0.835}$ (Chỉ giảm $16.5\%$)
  - $\text{RealmEfficiency} = 1.25$
  - $\implies \text{Final Multiplier} = 261.0 \times 1.25 \times 0.835 = \mathbf{272.41}$
- **Kết quả:** Tuy sát thương đơn hệ thấp hơn Chuyên Tu ($272.4$ vs $326.2$), nhưng người chơi có thể ra chiêu cả Hỏa lẫn Thủy đều đạt uy lực $272.4$, đồng thời đủ điều kiện kích hoạt Sát Chiêu Phức Hợp uy lực vượt trội.
