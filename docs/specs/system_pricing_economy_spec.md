# Đặc Tả Cân Bằng Toàn Diện Cơ Chế Giá Cả & Kinh Tế Hệ Thống (GuWorld)
*Document ID: SPEC-ECONOMY-001*  
*Part of Wayfinder Map #216*  
*Status: Approved Specification*

---

## 1. Mục Tiêu & Tôn Chỉ Thiết Kế (Design Tenets)

Hệ thống kinh tế trong thế giới Cổ Chân Nhân xoay quanh tài nguyên khan hiếm, rủi ro tu luyện và sự cách biệt thăm thẳm giữa các cảnh giới:

1. **Chỉ Tập Trung Vào Kinh Tế Hệ Thống (System Faucets & Sinks)**:
   - Chuẩn hóa toàn bộ các giao dịch giữa Người Chơi và Hệ Thống (Thương đoàn Caravan Shop, Thôn trang, Thương nhân lưu động, Kho Tàng Bảo Tông Môn, Chi phí Luyện Cổ, Phần thưởng Săn Quái).
   - Loại trừ hoàn toàn thị trường tự do giữa người chơi (P2P Marketplace, Player-to-Player Trade) để bảo toàn tính độc lập của cung cầu thị trường người chơi.

2. **Farm Quái Không Lỗ (Monster Hunting Viability)**:
   - Gỡ bỏ hoàn toàn cơ chế tự động nghiền nát Nguyên Thạch / Tiên Nguyên Thạch trong chiến đấu khi cạn Chân Nguyên (nguyên nhân cốt lõi gây "cháy túi" khi auto farm).
   - Nâng tỷ lệ sàn thu mua chiến lợi phẩm cho Cổ sư sơ kỳ (Rank 1–3) để bảo đảm thu nhập bán rác luôn vượt chi phí nuôi Cổ và hồi phục sinh lực ($Profit > 0$).

3. **Luyện Cổ Có Lời Tương Xứng Rủi Ro (Refinement Profit Margin)**:
   - Nới rộng khoảng cách giá (gap) theo hàm mũ giữa các Rank Cổ.
   - Bảo đảm giá trị định giá của Cổ bậc cao luôn vượt tổng chi phí nguyên liệu kỳ vọng chia cho tỷ lệ thành công bình quân ($\mathbb{E}[ROI] \in [+25\%, +40\%]$). Luyện Cổ là một nghề có thể làm giàu nếu thành công.

4. **Tiên Phàm Phân Minh (Mortal-Immortal Economic Boundary)**:
   - Tách biệt tuyệt đối giữa kinh tế Phàm Giới (Rank 1–5, tiền tệ Nguyên Thạch - NT) và Tiên Giới (Rank 6–9, tiền tệ Tiên Nguyên Thạch - TNT).
   - Tông môn phàm giới không thu nạp Tiên Tài, Tiên Cổ hay Tiên Cổ Bí Phương (Rank 6+).
   - Tỷ giá quy đổi tiền tệ được khóa ở mức cao ($100.000\text{ NT} = 1\text{ TNT}$) để phục vụ tích lũy độ kiếp thăng tiên, ngăn chặn việc phá vỡ kinh tế Phàm.

---

## 2. Ma Trận Giá Base & Khoảng Cách Giá Cổ Trùng (Gu Base Price Matrix)

Giá cơ sở (`GuBasePriceByRank`) được điều chỉnh nhằm giải quyết bài toán $E[ROI]$ âm nặng (-88% đến -99.98%) trước đây:

| Rank | Cảnh Giới Cổ Sư | Tiền Tệ | Giá Base Cũ | **Giá Base Mới** | Hệ Số Giãn Cách (Gap vs Rank trước) | Chi Phí Đá Kỳ Vọng ($\mathbb{E}[\text{Cost}]$) | Lợi Nhuận Kỳ Vọng ($\mathbb{E}[\text{ROI}]$) |
| :---: | :--- | :---: | :---: | :---: | :---: | :---: | :---: |
| **R1** | Nhất Chuyển | NT | 200 | **250** | — | $111\text{ NT}$ | **$+125\%$** |
| **R2** | Nhị Chuyển | NT | 500 | **1.000** | **$4.0\times$** | $667\text{ NT}$ | **$+50\%$** |
| **R3** | Tam Chuyển | NT | 1.200 | **4.000** | **$4.0\times$** | $3.000\text{ NT}$ | **$+33\%$** |
| **R4** | Tứ Chuyển | NT | 2.500 | **22.000** | **$5.5\times$** | $16.667\text{ NT}$ | **$+32\%$** |
| **R5** | Ngũ Chuyển | NT | 5.000 | **250.000** | **$11.4\times$** | $194.000\text{ NT}$ | **$+29\%$** |
| **R6** | Lục Chuyển (Hạ Tiên) | TNT | 200 | **3.500** | *Bước nhảy Tiên Phàm* | $2.500\text{ TNT}$ | **$+40\%$** |
| **R7** | Thất Chuyển (Trung Tiên) | TNT | 500 | **35.000** | **$10.0\times$** | $26.667\text{ TNT}$ | **$+31\%$** |
| **R8** | Bát Chuyển (Thượng Tiên) | TNT | 1.200 | **550.000** | **$15.7\times$** | $400.000\text{ TNT}$ | **$+37.5\%$** |
| **R9** | Cửu Chuyển (Chí Tôn) | TNT | 2.500 | **14.000.000** | **$25.5\times$** | $10.000.000\text{ TNT}$ | **$+40\%$** |

### Quy Tắc Thu Mua Cổ Trùng (Sell-back Ratio):
- **Cố định 10% (0.10)** khi bán lại cho Thương nhân NPC (Thương đoàn Caravan, Thôn trang, Lưu động).
- **Mục đích**: Triệt tiêu hành vi lướt sóng buôn bán Cổ với NPC (mua 100% bán 10%). Thúc đẩy người chơi giữ Cổ lại để **hợp luyện lên bậc cao hơn** hoặc **quyên góp vào Tông Môn** (nhận 50% điểm cống hiến, gấp 5 lần bán cho NPC).

### Quy Tắc Bảo Toàn Phôi Cổ Khi Hợp Luyện Thất Bại:
- **Cổ Bản Mệnh (`IsVital`)**: Không bị phá hủy khi luyện xịt. Cổ rơi vào trạng thái hấp hối (`DYING`, đói 1, ngừng hoạt động) và Cổ sư chịu nội thương phản phệ.
- **Cổ Thường (Không phải Bản Mệnh)**: Tiêu biến thành tro bụi theo đúng quy luật tàn khốc của Cổ Đạo.

---

## 3. Ma Trận Giá Item, Nguyên Liệu & Chiến Lợi Phẩm (Item Price Matrix)

Tái cấu trúc toàn diện bảng giá Item fallback trong `pricing.go` và chuẩn hóa lại file `Item.csv` (loại bỏ hoàn toàn lỗi gán nhầm đơn vị tiền tệ ở Rank 6–9):

### Phàm Giới (Rank 1 – 5, đơn vị: Nguyên Thạch - NT):
*Gap giãn cách $1.000\times$ từ Nhất Chuyển lên Ngũ Chuyển:*

| Rank | Phân Hạng Phàm Giới | Giá Base Cũ (`pricing.go`) | **Giá Base Mới** | Hệ Số Giãn Cách (Gap) | Sàn Thu Mua NPC | Giá Bán Cho NPC |
| :---: | :--- | :---: | :---: | :---: | :---: | :---: |
| **R1** | Nhất Chuyển (Thường) | 1 NT | **2 NT** | — | **25%** | **0.5 NT** |
| **R2** | Nhị Chuyển (Khá) | 3 NT | **8 NT** | **$4.0\times$** | **20%** | **1.6 NT** |
| **R3** | Tam Chuyển (Hiếm) | 6 NT | **35 NT** | **$4.4\times$** | **20%** | **7.0 NT** |
| **R4** | Tứ Chuyển (Quý) | 13 NT | **200 NT** | **$5.7\times$** | **15%** | **30.0 NT** |
| **R5** | Ngũ Chuyển (Cực Phẩm Phàm) | 25 NT | **2.000 NT** | **$10.0\times$** | **15%** | **300.0 NT** |

### Tiên Giới (Rank 6 – 9, đơn vị: Tiên Nguyên Thạch - TNT):
*Gap giãn cách $3.000\times$ từ Lục Chuyển lên Cửu Chuyển (Sửa lỗi Item.csv cũ bị gán nhầm 15.000 - 350.000 TNT):*

| Rank | Phân Hạng Tiên Tài | Giá CSV Lỗi Cũ | **Giá Base Mới** | Hệ Số Giãn Cách (Gap) | Sàn Thu Mua NPC | Giá Bán Cho NPC |
| :---: | :--- | :---: | :---: | :---: | :---: | :---: |
| **R6** | Lục Chuyển (Hạ Phẩm Tiên Tài) | *15.000 TNT* | **5 TNT** | *Bước nhảy Tiên Phàm* | **10%** | **0.5 TNT** |
| **R7** | Thất Chuyển (Trung Phẩm Tiên Tài) | *40.000 TNT* | **50 TNT** | **$10.0\times$** | **10%** | **5.0 TNT** |
| **R8** | Bát Chuyển (Thượng Phẩm Tiên Tài) | *110.000 TNT* | **750 TNT** | **$15.0\times$** | **10%** | **75.0 TNT** |
| **R9** | Cửu Chuyển (Tuyệt Phẩm Tiên Tài) | *350.000 TNT* | **15.000 TNT** | **$20.0\times$** | **10%** | **1.500 TNT** |

### Thức Ăn Nuôi Cổ Trùng (`ShopItem.csv`):
- Toàn bộ thức ăn cơ bản cho Cổ Phàm (`boar_meat`, `copper_ore`, `bitter_wine`, etc.) giữ nguyên mức **$1\text{ NT/đơn vị}$** để bảo đảm người mới không bị thâm hụt tài nguyên khi nuôi Cổ.

---

## 4. Biên Độ Lợi Nhuận Farm Quái (Monster Drop Margins)

Sau khi gỡ bỏ cơ chế tự nghiền đá trong combat (`engine.go`), chi phí vận hành mỗi trận đánh quái được đưa về mức thực tế:
- **Chi phí vận hành (OpCost)**: $\approx 0.075\text{ NT}$ (độ đói Cổ) $+ 0.500\text{ NT}$ (hồi phục máu) $\approx \mathbf{0.575\text{ NT/trận}}$.
- **Kết quả kinh tế tại Rank 1**:
  - Tại sàn thu mua 10% cũ: Thu $0.455\text{ NT}$ vs chi phí $0.575\text{ NT}$ $\implies$ **Lỗ $-0.12\text{ NT/trận}$ (Quái thường lỗ $-0.29\text{ NT}$)**.
  - Tại sàn thu mua **25% mới**: Thu $1.14\text{ NT}$ vs chi phí $0.575\text{ NT}$ $\implies$ **Lãi ròng $+0.565\text{ NT/trận}$ (Quái tinh anh lãi $+2.2\text{ NT}$)**.
- **Kết quả kinh tế từ Rank 2 trở lên**:
  - Nhờ hệ số nhân số lượng rơi (`qtyMul` từ $2.0\times$ đến $5.0\times$), người chơi luôn đạt lãi ròng bền vững từ $+14\text{ NT/trận}$ (Rank 2) đến $+268\text{ NT/trận}$ (Rank 5).

---

## 5. Kinh Tế Tông Môn & Bí Phương (Sect & Recipes)

1. **Kho Tàng Bảo Tông Môn (Sect Vault)**:
   - **Đóng góp (Donate)**: Nhận `DonateRatio = 0.50` (50% giá base) điểm cống hiến.
   - **Đổi đồ (Vault Exchange)**: Tốn `VaultCost = 1.00` (100% giá base) điểm cống hiến.
   - **Chống đầu cơ (Arbitrage-Proof)**: Tỷ lệ đổi đồ ($1.00$) luôn cao hơn tỷ lệ đóng góp ($0.50$) và tỷ lệ bán cho NPC ($0.10 \dots 0.25$), triệt tiêu mọi vòng lặp farm điểm cống hiến vô tận.
2. **Chặn Toàn Diện Tiên Tài Nhập Tông Môn Phàm Giới**:
   - `DonateMaterials`: Chặn vật phẩm `matRank >= 6`.
   - `DonateGu`: Chặn Cổ trùng `guInstance.Rank >= 6`.
   - `DonateRecipe`: Chặn Bí phương `targetGuRank >= 6`.
   - `DonateBulk`: Chặn hàng loạt mọi vật phẩm, Cổ và Bí phương Rank 6+.
3. **Định Giá Bí Phương Luyện Cổ**:
   - `RecipeBasePrice(targetGuRank) = GuBasePriceByRank(targetGuRank)`.
   - Bán lại Bí phương cho Thương nhân: Nhân với tỷ lệ hoàn thành (`RecipeCompletionFactor(completionRate)`) và áp dụng sàn thu mua Cổ (10%).

---

## 6. Lộ Trình Triển Khai Mã Nguồn & Dữ Liệu

| Hạng Mục | Trạng Thái | Mô Tả |
| :--- | :---: | :--- |
| **Gỡ hao đá combat** | ✅ **Đã Xong** | Xóa vòng lặp auto-restore mana bằng đá tại `backend/internal/simulator/engine.go` và cập nhật `combat_sim_build.go`. |
| **Chặn Tiên tài Tông môn** | ✅ **Đã Xong** | Thêm validator Rank 6+ cho toàn bộ các hàm quyên góp tại `backend/internal/handlers/sect_donations.go`. |
| **Unit Tests Backend** | ✅ **Đã Xong** | `go test ./...` đạt 100% pass trên toàn bộ gói handler, simulator và server. |
| **Cập nhật `pricing.go`** | 🔄 *Sẵn sàng nạp* | Cập nhật `guRankBasePrice`, `itemRankBasePrice`, và hàm `SellBackRatioByRank(rank, itemType)`. |
| **Chuẩn hóa CSV Data** | 🔄 *Sẵn sàng nạp* | Đồng bộ lại cột `base_price` trong `backend/data/Gu.csv` và `backend/data/Item.csv`. |
