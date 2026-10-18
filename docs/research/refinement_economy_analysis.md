# Nghiên Cứu & Khảo Sát: Kinh Tế Hợp Luyện (Recipe.csv) vs Định Giá Cổ Trùng (Gu.csv / pricing.go)

- **Ticket**: [#217](https://github.com/VipPro67/GuWorld/issues/217)
- **Map Issue**: [#216 - Cân bằng toàn bộ cơ chế giá cả hệ thống, giá Cổ, Item, Bí phương & Kinh tế farm quái](https://github.com/VipPro67/GuWorld/issues/216)
- **Files liên quan**: `backend/data/Recipe.csv`, `backend/data/Gu.csv`, `backend/data/Item.csv`, `backend/pkg/pricing/pricing.go`, `backend/internal/handlers/refine_gu.go`, `backend/internal/handlers/refine_core.go`.

---

## 1. Tóm Tắt Kết Quả (Executive Summary)

Khảo sát toàn diện **4,479 công thức** trong `Recipe.csv` và đối chiếu với bảng giá base Cổ trùng trong `pricing.go` (`guRankBasePrice`) cùng `Gu.csv` cho thấy:

1. **Sụp đổ tỷ suất lợi nhuận (Systemic Negative ROI)**:
   - Ở mọi Rank từ 1 đến 9, tỷ suất lợi nhuận kỳ vọng của người chơi khi hợp luyện $E[\text{ROI}]$ đều **bị âm nặng nề, dao động từ $-88.0\%$ đến $-99.98\%$**.
   - Nếu tính cả giá thị trường nguyên liệu trong `Item.csv` và mất phôi Cổ khi thất bại, **người chơi càng luyện Cổ cấp cao thì càng tán gia bại sản**.
   - Thậm chí, ngay cả khi giả định **toàn bộ nguyên liệu và phôi Cổ là miễn phí ($0$ đồng)**, chỉ riêng **chi phí tiền đá trực tiếp (Nguyên thạch / Tiên nguyên thạch)** của số lần thử kỳ vọng $E[N]$ đã **vượt gấp từ $1.11$ lần đến $4,000$ lần** so với giá base của con Cổ tạo ra (từ Rank 3 trở lên).

2. **Nguyên nhân cốt lõi (Root Cause)**:
   - `Recipe.csv` được thiết kế theo tư duy kinh tế bậc thang lũy thừa nghiêm ngặt: tỷ lệ thành công giảm dốc đứng ($90\% \to 75\% \to 60\% \to 45\% \to 30\% \to 8\% \to 3\% \to 1\% \to 0.2\%$), số lần thử kỳ vọng $E[N]$ tăng từ $1.11$ lên $500$ lần, phí đá tăng theo cấp số nhân (từ $100$ NT lên $20,000$ TNT).
   - Ngược lại, bảng giá `guRankBasePrice` trong `pricing.go` chỉ là một mảng giá trị tượng trưng được copy-paste đơn giản:
     - Phàm Cổ (R1-5): $200 \to 500 \to 1,200 \to 2,500 \to 5,000$ NT (tỷ lệ giãn cách chỉ $\approx 2.0\times - 2.5\times$).
     - Tiên Cổ (R6-9): $200 \to 500 \to 1,200 \to 2,500$ TNT (lặp lại y hệt 4 bậc đầu của Phàm Cổ).
   - Sự **lệch pha hoàn toàn (Complete Decoupling)** giữa chi phí sản xuất kỳ vọng $E[\text{Cost}]$ và giá trị định giá $BasePrice$ đã biến tính năng Hợp Luyện thành một "hố đen tiêu hao" vô lý.

3. **Đề xuất giải pháp**:
   - Tái cấu trúc bảng giá base $GuBasePriceByRank$ theo mô hình giãn cách đa tầng (Multi-tier Rank Gap) đảm bảo biên lợi nhuận kỳ vọng $E[\text{ROI}] \in [+20\%, +50\%]$.
   - Tách biệt rõ ràng 2 tầng tiền tệ: Phàm Giới (Rank 1-5, đơn vị **Nguyên Thạch**) và Tiên Giới (Rank 6-9, đơn vị **Tiên Nguyên Thạch**).
   - Điều chỉnh hoặc chuẩn hóa dữ liệu nguyên liệu trong `Item.csv` và cơ chế bảo toàn phôi Cổ Tiên duy nhất (`uq_player_gu_immortal`).

---

## 2. Khảo Sát Chi Tiết Công Thức Hợp Luyện Theo Rank (Recipe.csv)

### 2.1 Bảng tổng hợp thông số trung bình từng Rank (1 - 9)

Thống kê trên toàn bộ $4,479$ công thức trong `backend/data/Recipe.csv`:

| Rank | Số lượng công thức | Tỉ lệ thành công ($p$) | Số lần thử kỳ vọng $E[N] = 1/p$ | Phí Nguyên Thạch / lần | Phí Tiên Nguyên Thạch / lần | Phôi Cổ yêu cầu (`required_gus`) | Tổng SL nguyên liệu (`materials`) |
| :---: | :---: | :---: | :---: | :---: | :---: | :--- | :---: |
| **1** | 491 | $90.0\%$ | **1.11** | 100 NT | 0 | Không yêu cầu phôi | 13 |
| **2** | 482 | $75.0\%$ | **1.33** | 300 NT | 0 | $1\times$ Cổ Rank 1 | 26 |
| **3** | 535 | $60.0\%$ | **1.67** | 800 NT | 0 | $1\times$ Cổ Rank 2 | 53.5 |
| **4** | 522 | $45.0\%$ | **2.22** | 2,500 NT | 0 | $1\times$ R2 + $1\times$ R3 | 91.1 |
| **5** | 558 | $30.0\%$ | **3.33** | 10,000 NT | 0 | $1\times$ R3 + $2\times$ R4 | 172.8 |
| **6** | 543 | $8.0\%$ | **12.50** | 50,000 NT | 200 TNT | $2\times$ R4 + $2\times$ R5 | 291.5 |
| **7** | 486 | $3.0\%$ | **33.33** | 200,000 NT | 800 TNT | $1\times$ R4 + $2\times$ R5 + $2\times$ R6 | 553.3 |
| **8** | 446 | $1.0\%$ | **100.00** | 1,000,000 NT | 4,000 TNT | $1\times$ R5 + $2\times$ R6 + $2\times$ R7 | 977.3 |
| **9** | 416 | $0.2\%$ | **500.00** | 5,000,000 NT | 20,000 TNT | $1\times$ R5 + $1\times$ R6 + $2\times$ R7 + $2\times$ R8 | 1,689.2 |

### 2.2 Cơ cấu chi phí 1 lần thử ($Cost_{\text{attempt}}$)

Theo cơ chế xử lý tại `backend/internal/handlers/refine_gu.go`:
- **Phí đá trực tiếp**: Bị trừ ngay lập tức khi bắt đầu quá trình luyện (`player.PrimevalStones` và `player.ImmortalStones`).
- **Nguyên liệu phụ (`required_materials`)**:
  - Khi thành công: Tiêu thụ $100\%$.
  - Khi thất bại an toàn (`OutcomeFailSafe` - chỉ kích hoạt khi có cảnh giới Luyện Đạo cao): Mất $50\%$.
  - Khi thất bại thông thường hoặc phản phệ (`OutcomeFailNormal` / `OutcomeFailBackfire`): Tiêu thụ $100\%$ thành tro bụi.
- **Phôi Cổ (`required_gus`)**:
  - Khi thành công: Tiêu thụ toàn bộ phôi đã chọn.
  - Khi thất bại thông thường: Toàn bộ phôi Cổ **bị tiêu hủy** (trừ Cổ bản mệnh sống sót ở trạng thái `DYING`).

Do đó, ở baseline (người chơi bình thường chưa có cảnh giới Tông Sư/Vô Thượng), chi phí tiêu hao cho mỗi lần thử bao gồm toàn bộ:
$$Cost_{\text{attempt}} = Fee_{\text{stones}} + Value_{\text{phôi}} + Value_{\text{materials}}$$

---

## 3. So Sánh Chi Phí Kỳ Vọng $E[\text{Cost}]$ vs Giá Base Hiện Tại

### 3.1 Bảng so sánh tổng hợp

Dưới đây là bảng đối chiếu giữa chi phí kỳ vọng để luyện thành công 1 con Cổ ($E[\text{Cost}] = E[N] \times Cost_{\text{attempt}}$) so với giá base hiện hành trong `pricing.go` (`guRankBasePrice`):

> **Ghi chú về đơn vị**:
> - Rank 1 - 5: Đơn vị là **Nguyên Thạch (NT)**.
> - Rank 6 - 9: Giá Cổ tính bằng **Tiên Nguyên Thạch (TNT)**. Chi phí thực tế gồm cả TNT và phần NT phụ trợ.

| Rank | Giá Base Hiện Tại | Phí Đá Thuần Túy Kỳ Vọng ($E[N] \times Fee$) | Phôi Cổ Kỳ Vọng (Tính theo Base) | Chi Phí Nguyên Liệu Kỳ Vọng (`Item.csv`) | Tổng Chi Phí Kỳ Vọng $E[\text{Cost}]$ | $E[\text{ROI}]$ so với Giá Base | $E[\text{ROI}]$ khi Bán Shop (10%) |
| :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **R1** | 200 NT | 111 NT | 0 NT | 1,556 NT | **1,667 NT** | **-88.00%** | -98.80% |
| **R2** | 500 NT | 400 NT | 267 NT | 11,067 NT | **11,733 NT** | **-95.74%** | -99.57% |
| **R3** | 1,200 NT | 1,333 NT | 833 NT | 74,302 NT | **76,469 NT** | **-98.43%** | -99.84% |
| **R4** | 2,500 NT | 5,556 NT | 3,778 NT | 433,119 NT | **442,452 NT** | **-99.43%** | -99.94% |
| **R5** | 5,000 NT | 33,333 NT | 20,667 NT | 3,126,792 NT | **3,180,792 NT** | **-99.84%** | -99.98% |
| **R6** | 200 TNT | 2,500 TNT *(+625k NT)* | 187,500 NT | 59,291,436 NT | **2,500 TNT + 60.1M NT** | **-92.00%** *(chỉ tính TNT)* | -99.20% |
| **R7** | 500 TNT | 26,667 TNT *(+6.67M NT)* | 13,333 TNT *(+417k NT)* | 746,790,123 NT | **40,000 TNT + 754M NT** | **-98.75%** *(chỉ tính TNT)* | -99.88% |
| **R8** | 1,200 TNT | 400,000 TNT *(+100M NT)* | 140,000 TNT *(+500k NT)* | 9,751,569,507 NT | **540,000 TNT + 9.85B NT** | **-99.78%** *(chỉ tính TNT)* | -99.98% |
| **R9** | 2,500 TNT | 10,000,000 TNT *(+2.5B NT)* | 1,800,000 TNT *(+2.5M NT)* | 226,356,971,154 NT | **11.8M TNT + 229B NT** | **-99.98%** *(chỉ tính TNT)* | -100.00% |

---

## 4. Năm Điểm Bất Hợp Lý Cốt Lõi (Critical Economic Regressions)

### 4.1 Tử huyệt 1: Phí đá 1 lần thử vượt qua / áp sát giá trị toàn bộ con Cổ
- Tại **Rank 4**: Phí đá của **1 lần bấm** là $2,500$ NT = đúng bằng $100\%$ giá base của Cổ Rank 4 ($2,500$ NT). Với tỷ lệ $45\%$ ($E[N] = 2.22$), chỉ riêng tiền đá đã là $5,556$ NT (gấp $2.22$ lần giá Cổ).
- Tại **Rank 5**: Phí đá của 1 lần bấm là $10,000$ NT = **gấp 2 lần** giá base Cổ Rank 5 ($5,000$ NT). Kỳ vọng tốn $33,333$ NT tiền đá (gấp $6.67$ lần giá Cổ).
- Tại **Rank 6 (Tiên Cổ)**: Phí đá 1 lần bấm đòi hỏi $200$ TNT + $50,000$ NT. Giá base Cổ Rank 6 chỉ có $200$ TNT. Với tỷ lệ $8\%$ ($E[N] = 12.5$), tiền đá tốn $2,500$ TNT (gấp $12.5$ lần giá Cổ).
- Tại **Rank 7, 8, 9**: Tình trạng càng nghiêm trọng hơn:
  - Rank 7: Phí đá $800$ TNT/lần vs Giá base $500$ TNT $\rightarrow$ Kỳ vọng $26,667$ TNT (gấp $53.3$ lần giá Cổ).
  - Rank 8: Phí đá $4,000$ TNT/lần vs Giá base $1,200$ TNT $\rightarrow$ Kỳ vọng $400,000$ TNT (gấp $333$ lần giá Cổ).
  - Rank 9: Phí đá $20,000$ TNT/lần vs Giá base $2,500$ TNT $\rightarrow$ Kỳ vọng $10,000,000$ TNT (gấp $4,000$ lần giá Cổ).

### 4.2 Tử huyệt 2: Nghịch lý hàm mũ vs hàm cộng (Decoupled Scaling Curves)
- Chi phí sản xuất trong `Recipe.csv` tăng theo **hàm mũ (Exponential)**:
  - Số lần thử kỳ vọng: $1.11 \to 1.33 \to 1.67 \to 2.22 \to 3.33 \to 12.5 \to 33.3 \to 100 \to 500$ (tăng gấp $450$ lần).
  - Phí đá trực tiếp: $100 \to 300 \to 800 \to 2,500 \to 10,000$ NT, sau đó nhảy vọt sang $200 \to 800 \to 4,000 \to 20,000$ TNT.
- Trong khi đó, bảng giá base `guRankBasePrice` lại tăng theo **cấp số nhân cực nhỏ hoặc tuyến tính yếu**:
  - $200 \to 500$ ($\times 2.5$) $\to 1,200$ ($\times 2.4$) $\to 2,500$ ($\times 2.08$) $\to 5,000$ ($\times 2.0$).
  - Sang Tiên Cổ lại lặp lại y hệt: $200 \to 500 \to 1,200 \to 2,500$.
- Hệ quả: Khoảng cách giữa chi phí kỳ vọng và giá trị thu lại càng lên cao càng bị xé rách, khiến $E[\text{ROI}]$ rơi tự do từ $-88\%$ xuống $-99.98\%$.

### 4.3 Tử huyệt 3: Mất trắng phôi Cổ bậc cao và xung đột với tính duy nhất của Tiên Cổ
- Để luyện Tiên Cổ Rank 7 cần $2\times$ Tiên Cổ Rank 6 làm phôi.
- Với tỷ lệ thành công $3\%$ ($E[N] = 33.33$), mỗi lần thất bại sẽ tiêu hủy cả 2 con Cổ Rank 6.
- Kỳ vọng để luyện được **1 con Cổ Rank 7**, người chơi phải đốt cháy:
  $$33.33 \times 2 = 66.7 \text{ con Tiên Cổ Rank 6!}$$
- Tương tự, để luyện **1 con Cổ Rank 8** ($1\%$), kỳ vọng tiêu tốn:
  $$100 \times 2 = 200 \text{ con Cổ Rank 6} + 200 \text{ con Cổ Rank 7!}$$
- Và để luyện **1 con Cổ Rank 9** ($0.2\%$), kỳ vọng tiêu tốn:
  $$500 \times 2 = 1,000 \text{ con Cổ Rank 7} + 1,000 \text{ con Cổ Rank 8!}$$
- **Xung đột nghiêm trọng với hệ thống**: Trong game, Tiên Cổ (Rank $\ge 6$) là **DUY NHẤT TOÀN SERVER** (ràng buộc DB `uq_player_gu_immortal`, xem `docs/GAME_DATA.md`). Nếu toàn server chỉ tồn tại tối đa vài chục cá thể Tiên Cổ, việc đòi hỏi đốt cháy hàng trăm con Tiên Cổ làm phôi khiến việc hợp luyện Tiên Cổ bậc 8, 9 trên thực tế là **bất khả thi về mặt toán học** nếu không có tỷ lệ bảo tồn phôi gần như tuyệt đối (`OutcomeFailSafe` hoặc vật phẩm bảo vệ phôi).

### 4.4 Tử huyệt 4: Sự bất hợp lý trong định giá nguyên liệu tại `Item.csv`
- Trong `backend/data/Item.csv`, toàn bộ $792$ nguyên liệu (`Material`) được gán `base_price` theo dải:
  - Rank 1: $\approx 110$ NT
  - Rank 2: $\approx 325$ NT
  - Rank 3: $\approx 850$ NT
  - Rank 4: $\approx 2,200$ NT
  - Rank 5: $\approx 5,500$ NT
  - Rank 6: $\approx 16,500$
  - Rank 7: $\approx 45,000$
  - Rank 8: $\approx 110,000$
  - Rank 9: $\approx 325,000$
- Một công thức Rank 5 đòi hỏi $173$ nguyên liệu Rank 5. Với giá $5,500$ NT/món, chỉ riêng tiền nguyên liệu cho 1 lần bấm đã là $\approx 950,000$ NT! Qua $3.33$ lần thử kỳ vọng, chi phí nguyên liệu lên tới **$3.1$ triệu Nguyên Thạch**. Trong khi một con Cổ Rank 5 chỉ đáng giá $5,000$ NT.
- Ở Rank 6+, các nguyên liệu này nếu bị `pricing.StoneTierByRank` quy định thành Tiên Nguyên Thạch thì sẽ tạo ra con số hàng chục triệu TNT cho một lần bấm; còn nếu coi là Nguyên Thạch thì lại đòi hỏi hàng trăm triệu NT ở giai đoạn tu tiên vốn giao dịch bằng TNT.

### 4.5 Tử huyệt 5: Hai tầng tiền tệ không đồng bộ (Dual-Currency Disconnect)
- Hệ thống có 2 loại tiền tệ: Nguyên Thạch (`primeval_stones`) và Tiên Nguyên Thạch (`immortal_stones`).
- Tỷ giá đổi tại ngân hàng/sổ cái: $1\text{ TNT} = 100\text{ NT}$ (chiều bán ra), nhưng $100,000\text{ NT} = 1\text{ TNT}$ (chiều mua vào).
- Tuy nhiên, trong công thức luyện Cổ Tiên (Rank 6-9), hệ thống bắt người chơi vừa phải trả hàng triệu Nguyên Thạch vừa phải trả hàng nghìn Tiên Nguyên Thạch. Việc này gây lúng túng cho người chơi Cổ Tiên vốn chủ yếu thu thập và quản lý tài nguyên trong Tiên Khiếu bằng Tiên Nguyên Thạch.

---

## 5. Đề Xuất Bảng Giá Base Mới & Độ Giãn Cách (Gu Rank Price Gap)

Để đưa nền kinh tế hợp luyện trở về trạng thái lành mạnh, đảm bảo người chơi luyện Cổ thành công có tỷ suất lợi nhuận kỳ vọng dương hợp lý ($E[\text{ROI}] \in [+20\%, +50\%]$), đề xuất bảng giá mới và độ giãn cách như sau.

### 5.1 Nguyên tắc xác định giá base mới
1. **Giá trị bù đắp rủi ro (Risk-adjusted Value)**: Giá base của Cổ phải lớn hơn tổng chi phí kỳ vọng gồm: Phí đá kỳ vọng ($E[N] \times Fee$) + Giá trị phôi Cổ hao hụt kỳ vọng + Chi phí nguyên liệu cơ bản.
2. **Biên lợi nhuận mục tiêu (Target ROI)**: Người chơi thành công đạt lợi nhuận ròng $+25\%$ đến $+40\%$ so với tổng vốn đầu tư kỳ vọng khi định giá hệ thống (và khi bán lại thương nhân ở mức SellBackRatio phù hợp).
3. **Phân tầng tiền tệ triệt để**:
   - Rank 1 - 5: Tính thuần túy bằng **Nguyên Thạch (NT)**.
   - Rank 6 - 9: Tính thuần túy bằng **Tiên Nguyên Thạch (TNT)**.

---

### 5.2 Bảng giá base đề xuất (Khuyến nghị chuẩn cho Ticket #219)

Mô hình định giá dựa trên:
- **Phàm Cổ (R1-5)**: Chi phí đá trực tiếp kỳ vọng + Giá trị phôi Cổ lũy kế + Biên lợi nhuận $+25\% \dots +35\%$.
- **Tiên Cổ (R6-9)**: Chi phí đá Tiên trực tiếp kỳ vọng + Giá trị phôi Cổ Tiên cơ sở + Biên lợi nhuận $+30\% \dots +40\%$.

| Rank | Phân Loại Cảnh Giới | Tiền Tệ | Giá Base Hiện Tại | **Giá Base Đề Xuất Mới** | Hệ Số Tăng vs Hiện Tại | Hệ Số Giãn Cách (Gap vs Rank trước) | Chi Phí Đá Kỳ Vọng | $E[\text{ROI}]$ Dự Kiến vs Chi Phí Đá |
| :---: | :--- | :---: | :---: | :---: | :---: | :---: | :---: |
| **R1** | Nhất Chuyển (Sơ giai) | NT | 200 | **250** | $1.25\times$ | — | 111 NT | **+125%** *(+25% nếu gồm NL)* |
| **R2** | Nhị Chuyển | NT | 500 | **1,000** | $2.00\times$ | **$4.0\times$** | 400 NT | **+50%** *(vs Đá+Phôi R1)* |
| **R3** | Tam Chuyển | NT | 1,200 | **4,000** | $3.33\times$ | **$4.0\times$** | 1,333 NT | **+33%** *(vs Đá+Phôi R2)* |
| **R4** | Tứ Chuyển (Cường giả) | NT | 2,500 | **22,000** | $8.80\times$ | **$5.5\times$** | 5,556 NT | **+32%** *(vs Đá+Phôi R2,R3)* |
| **R5** | Ngũ Chuyển (Phàm đỉnh) | NT | 5,000 | **250,000** | $50.00\times$ | **$11.4\times$** | 33,333 NT | **+29%** *(vs Đá+Phôi R3,R4)* |
| **R6** | Lục Chuyển (Hạ Tiên) | TNT | 200 | **3,500** | $17.50\times$ | **Bước nhảy Tiên Phàm** | 2,500 TNT | **+40%** *(vs Đá TNT)* |
| **R7** | Thất Chuyển (Trung Tiên) | TNT | 500 | **35,000** | $70.00\times$ | **$10.0\times$** | 26,667 TNT | **+31%** *(vs Đá TNT)* |
| **R8** | Bát Chuyển (Thượng Tiên) | TNT | 1,200 | **550,000** | $458.33\times$ | **$15.7\times$** | 400,000 TNT | **+37.5%** *(vs Đá TNT)* |
| **R9** | Cửu Chuyển (Chí Tôn Tiên) | TNT | 2,500 | **14,000,000** | $5,600.00\times$ | **$25.5\times$** | 10,000,000 TNT | **+40%** *(vs Đá TNT)* |

---

### 5.3 Đánh giá độ khả thi và tính logic theo cốt truyện (Lore & Gameplay)

1. **Giai đoạn Phàm Giới (Rank 1 - 5)**:
   - Rank 1 ($250$ NT) đến Rank 3 ($4,000$ NT) hoàn toàn phù hợp với túi tiền của Cổ sư phàm nhân thông qua việc làm nhiệm vụ gia tộc, săn quái và buôn bán thương đội.
   - Rank 4 ($22,000$ NT) và Rank 5 ($250,000$ NT) phản ánh đúng vị thế của Tộc trưởng, Trưởng lão ngũ chuyển đỉnh phong, nơi một con Cổ ngũ chuyển là bảo vật trấn tộc của cả một sơn trại.
2. **Bước nhảy Tiên - Phàm (Rank 5 $\to$ Rank 6)**:
   - Chuyển giao hoàn toàn từ đơn vị Nguyên Thạch sang Tiên Nguyên Thạch.
   - $3,500$ TNT cho một con Tiên Cổ Lục Chuyển phản ánh đúng cái giá mà một vị Cổ Tiên mới thăng tiên phải bỏ ra để sở hữu Tiên Cổ đầu tiên (qua độ kiếp hoặc luyện chế gian khổ).
3. **Đỉnh phong Tiên Giới (Rank 8 - 9)**:
   - Cổ Bát Chuyển ($550,000$ TNT) và Cổ Cửu Chuyển ($14,000,000$ TNT) là những thần vật hiếm hoi trong trời đất, tương xứng với tài sản của cự đầu Bát Chuyển và Tôn Giả Vô Địch Cửu Chuyển.

---

## 6. Kiến Nghị Cho Các Ticket Tiếp Theo Trong Map #216

Để hoàn thiện toàn bộ bản thiết kế kinh tế hệ thống, khuyến nghị triển khai các bước tiếp theo như sau:

1. **Ticket #219 (Decision: Hệ số khoảng cách giá Cổ & Ma trận giá)**:
   - Áp dụng bảng giá base đề xuất trên vào `guRankBasePrice` trong `backend/pkg/pricing/pricing.go`.
   - Cập nhật lại cột `base_price` trong `backend/data/Gu.csv`.

2. **Ticket #220 (Decision: Định giá Item, Nguyên liệu & Biểu giá thương nhân)**:
   - **Tái cân bằng giá nguyên liệu trong `Item.csv`**: Nguyên liệu rác và nguyên liệu phụ luyện cổ cần đưa về đúng thang giá hợp lý (ví dụ: rank 1 khoảng $1-5$ NT, rank 2 khoảng $5-15$ NT... thay vì $100 - 5,000$ NT như hiện tại) để tổng tiền nguyên liệu không vượt quá $30\%$ giá trị con Cổ.
   - Thống nhất đơn vị tiền tệ cho nguyên liệu Rank 6-9 trong `Item.csv`.

3. **Cải tiến cơ chế Hợp Luyện Tiên Cổ (`refine_gu.go` / `Recipe.csv`)**:
   - **Bảo toàn phôi Tiên Cổ**: Do Tiên Cổ là duy nhất toàn server, khi luyện thất bại Cổ Rank 7-9, cần có cơ chế **bảo tồn phôi Cổ Tiên** (chỉ làm Cổ bị đói/DYING hoặc tiêu hao nguyên liệu phụ + đá) thay vì xóa sổ vĩnh viễn cá thể Tiên Cổ phôi, tránh tình trạng tuyệt chủng Cổ Tiên trên toàn server.
   - **Bỏ yêu cầu Nguyên Thạch ở công thức Tiên Cổ**: Công thức Tiên Cổ (Rank 6+) chỉ nên yêu cầu Tiên Nguyên Thạch (`immortal_essence_stones_cost`), loại bỏ khoản trừ `primeval_stones_cost` để đồng bộ tiền tệ.
