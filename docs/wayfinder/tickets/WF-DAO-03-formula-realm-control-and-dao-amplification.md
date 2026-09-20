# [WF-DAO-03] Thiết kế công thức Điều Hòa Cảnh Giới (Realm Control) & Khuếch Đại Đạo Ngân

**Parent Map:** [[WAYFINDER MAP] Dao Mark Conflict & Path Balance](../dao-mark-conflict-map.md)  
**Label:** `wayfinder:grilling`  
**Status:** Closed  
**Assignee:** Antigravity  

---

## Question

1. Công thức tính **Raw Conflict Score** từ các đạo ngân ngoại hệ (sau khi đã lọc qua Grace Threshold từ WF-DAO-02).
2. Cơ chế **Điều Hòa Cảnh Giới (Realm Control)**:
   - Các mức giảm trừ xung đột theo Cảnh Giới Chân Ý: $0\% \to 80\%$.
3. Công thức tính **Tỷ Lệ Xung Đột (Conflict Ratio)** & **Hệ Số Xung Đột Nội Tại (Internal Conflict Multiplier)** với sàn bảo hộ $\ge 0.60$.
4. Công thức **Khuếch Đại Đạo Ngân (Dao Amplification)** 2 đoạn giải quyết bài toán lạm phát số mũ.
5. Cơ chế **Hiệu Suất Cảnh Giới (Realm Efficiency)** ($\times 1.00 \to \times 1.45$).

---

## Resolution

Đã hoàn thành thiết kế toán học chi tiết tại [WF-DAO-03 Spec](../specs/WF-DAO-03-realm-control-dao-amplification.md):
- **Raw Conflict:** $\sum_{P_{\text{other}}} M(P_{\text{other}}) \times (1 - \text{Comp})$ (bỏ qua đạo ngân $\le T_{\text{grace}}$).
- **Realm Control:** Giảm trừ xung đột theo bậc Chân Ý: Ordinary $0\%$, Master $10\%$, Grandmaster $25\%$, Great Grandmaster $45\%$, Supreme $80\%$.
- **Conflict Multiplier:** $\frac{1}{1 + \text{Ratio} \times \text{Sensitivity} \times K}$ (với $K=2.0$, $S_{\text{scale}}=50.000$, sàn bảo hộ $\ge 0.60$).
- **Dao Amplification:** Piecewise Linear: $1 + M/100$ ($M \le 10.000$) và $101 + (M-10.000)/250$ ($M > 10.000$).
- **Realm Efficiency:** Nhân thêm $\times 1.00 \to \times 1.45$ dựa trên mức độ ngộ đạo.
- **Stress Tests:** Đã kiểm thử 3 kịch bản thực tế (Chuyên tu, Song tu sơ khởi, Song tu đại thành).

