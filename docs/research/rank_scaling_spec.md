# Đặc Tả Toán Học: Dynamic Proc Chance & Áp Chế Cảnh Giới 2 Chiều
*(Mathematical Specification: Dynamic Proc Chance, Caster Attenuation & Target Suppression)*

- **Ticket**: [#70](https://github.com/VipPro67/GuWorld/issues/70)
- **Tham Chiếu Nguồn**: `backend/internal/gamefx/energy.go` (`HigherRankManaMultiplier`), `backend/internal/statusfx/spec.go`

---

## 1. Bảng Trọng Số Sức Mạnh Theo Rank (Rank Power Scale Table)

Dựa trên bảng phân cấp tiêu hao và bước nhảy cảnh giới hiện hữu tại `HigherRankManaMultiplier`:

| Rank | Tên Cảnh Giới | Trọng Số Sức Mạnh $W(\text{Rank})$ | Phân Loại |
| :---: | :--- | :---: | :---: |
| **1** | Nhất Chuyển | $1$ | Phàm Cổ Sư |
| **2** | Nhị Chuyển | $2$ | Phàm Cổ Sư |
| **3** | Tam Chuyển | $5$ | Phàm Cổ Sư |
| **4** | Tứ Chuyển | $20$ | Phàm Cổ Sư |
| **5** | Ngũ Chuyển | $100$ | Đỉnh Phong Phàm Giới |
| **6** | Lục Chuyển | $1,000$ | Hạ Tiên (Bước Nhảy Tiên Phàm $\times 10$) |
| **7** | Thất Chuyển | $10,000$ | Trung Tiên |
| **8** | Bát Chuyển | $100,000$ | Thượng Tiên |
| **9** | Cửu Chuyển | $1,000,000$ | Tôn Giả Tiên |

---

## 2. Công Thức Caster Attenuation (Suy Hao Buff Bản Thân)

Khi Cổ sư có cảnh giới cao hơn Rank của Cổ trùng đang sử dụng để buff cho chính mình hoặc đồng đội (`CasterRank > GuRank`):

### 2.1 Hệ Số Suy Hao Caster ($K_{\text{caster}}$)
$$K_{\text{caster}}(\text{GuRank}, \text{CasterRank}) = \frac{W(\text{GuRank})}{W(\text{CasterRank})}$$

*(Giới hạn sàn: $\min K_{\text{caster}} = 0.0001$, tức tối thiểu $0.01\%$ để tránh lỗi chia số không).*

### 2.2 Áp Dụng Lên Uy Lực & Tỉ Lệ Kích Hoạt
1. **Chỉ số cộng thêm (Shield Flat, Heal Flat, StatMod)**:
   $$\text{EffectiveValue} = \text{BaseValue} \times K_{\text{caster}}$$
2. **Hiệu ứng đặc biệt (Bất Tử / INVINCIBLE, Miễn Dịch / IMMUNITY)**:
   - Giữ nguyên bản chất hiệu ứng nhưng **giảm mạnh tỉ lệ kích hoạt**:
     $$\text{EffectiveChance} = \text{BaseChance} \times K_{\text{caster}}$$
   - *Ví dụ minh họa:*
     - Cổ Bất Tử Rank 4 (BaseChance = 100%) khi Cổ Tiên Rank 9 sử dụng:
       $$K_{\text{caster}} = \frac{20}{1,000,000} = 0.00002 = 0.002\%$$
       $\rightarrow$ Tỉ lệ kích hoạt rơi về $0.002\%$, gần như không thể xuất hiện trong thực chiến ("coi như ai may thì được").

---

## 3. Công Thức Target Suppression (Áp Chế Cảnh Giới Khi Tấn Công Đối Thủ)

Khi Cổ trùng có đẳng cấp thấp hơn đẳng cấp của mục tiêu đối phương (`TargetRank > GuRank`):

### 3.1 Khoảng Chênh Lệch Cảnh Giới
$$\Delta R = \text{TargetRank} - \text{GuRank}$$

### 3.2 Tỉ Lệ Kích Hoạt Thực Tế (Effective Proc Chance)
$$\text{SuppressionFactor} = \max\left(0,\; 1.0 - 0.20 \times \Delta R\right)$$
$$\text{EffectiveChance} = \text{BaseChance} \times \text{SuppressionFactor}$$

- $\Delta R = 1$: Giảm $20\%$ tỉ lệ trúng ($\text{Factor} = 0.8$).
- $\Delta R = 2$: Giảm $40\%$ tỉ lệ trúng ($\text{Factor} = 0.6$).
- $\Delta R = 3$: Giảm $60\%$ tỉ lệ trúng ($\text{Factor} = 0.4$).
- $\Delta R = 4$: Giảm $80\%$ tỉ lệ trúng ($\text{Factor} = 0.2$).
- $\Delta R \ge 5$: Giảm $100\%$ tỉ lệ trúng ($\text{Factor} = 0.0$) $\rightarrow$ **Mục tiêu miễn dịch hoàn toàn với hiệu ứng bất lợi từ Cổ trùng này**.

### 3.3 Rút Ngắn Thời Lượng Khống Chế (Duration Penalty)
Nếu hiệu ứng trúng đích ($\text{Roll} \le \text{EffectiveChance}$):
$$\text{DurationPenalty} = \left\lfloor \frac{\Delta R}{2} \right\rfloor$$
$$\text{EffectiveTurns} = \max\left(1,\; \text{BaseTurns} - \text{DurationPenalty}\right)$$

---

## 4. Công Thức Dynamic Proc Chance Tổng Hợp

Tỉ lệ kích hoạt cuối cùng của một `AtomicOperation` được tính qua đường ống (pipeline):

$$\text{ProcChance}_{\text{final}} = \left(\text{BaseChance} + 0.05 \times (\text{GuRank} - 1)\right) \times \text{Modifier}$$

Trong đó:
- Với Target là `SELF` hoặc `ALL_ALLIES`:
  $$\text{Modifier} = K_{\text{caster}}$$
- Với Target là `TARGET` hoặc `ALL_ENEMIES`:
  $$\text{Modifier} = \text{SuppressionFactor} \times (1.0 - \text{CCResist}_{\text{target}})$$
  *(có xét thêm Kháng tính tâm thần nếu là debuff có `ResistGate: true`).*
