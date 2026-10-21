# Danh Mục Tài Nguyên Đồ Họa Pixel Art & Tiêu Chuẩn Kỹ Thuật (Prototype 0.1)

> **Mã công việc:** Ticket #242 (thuộc Map #241)  
> **Mục tiêu:** Khảo sát, tuyển chọn bộ asset đồ họa Pixel Art Top-Down phong cách cổ phong Á Đông / võ hiệp tiên hiệp (tương tự *Cổ Tiên Lộ*), có giấy phép mở (CC0 / Free Commercial Friendly), tối ưu cho engine đồ họa Phaser 3 và web browser.

---

## 1. Đánh giá Thẩm mỹ & Đối chiếu Phong cách "Cổ Tiên Lộ" (`cotienlo.com`)

Tựa game tham chiếu **Cổ Tiên Lộ** lấy bối cảnh Cổ Chân Nhân (Reverend Insanity) với đồ họa 2D Top-Down góc nhìn 3/4 (Orthographic Retro Pixel Art tương tự Game Boy Advance JRPG / *The Minish Cap*):
* **Môi trường & Kiến trúc:** Nhà sàn gỗ, mái ngói cong, tường đá rêu phong, bờ suối, lũy tre, ruộng vườn bậc thang, vách đá cheo leo (phù hợp tuyệt đối với hình thái các sơn trại như Cổ Nguyệt Sơn Trại, Bạch Gia Trại trên Thanh Mao Sơn).
* **Nhân vật:** Tỉ lệ Semi-Chibi (~2.5 - 3 đầu người), mặc đạo bào, áo cộc võ sinh, trang phục cổ trang Á Đông, di chuyển 4 hướng mượt mà.
* **Cổ trùng & Kỹ năng (VFX):** Cổ trùng có tạo hình côn trùng (bọ cánh cứng, ve sầu, rết, bướm, bọ cạp...). VFX kết hợp giữa chưởng khí kiếm hiệp, luồng độc ám đạo (xanh lục/tím), hỏa diễm, băng sương và ánh trăng (Nguyệt Quang Cổ).

---

## 2. Danh Mục Asset Tuyển Chọn Khả Dụng Cho Prototype

### 2.1. Tileset Bản Đồ & Địa Hình Cổ Phong

| Tên Bộ Asset | Tác Giả / Nguồn | Kích Thước | Giấy Phép (License) | Mô Tả & Điểm Phù Hợp |
| :--- | :--- | :--- | :--- | :--- |
| **Ninja Adventure Pack** | Pixel-boy & AAA ([itch.io](https://pixel-boy.itch.io/ninja-adventure)) | **16x16** | **CC0 (Public Domain)** | Bộ asset Á Đông toàn diện nhất: nhà gỗ, mái ngói cong cổ kính, cổng tam quan, cầu gỗ, lũy tre, ruộng đồng, suối nước, hang đá. Hỗ trợ autotile. **Lựa chọn số 1 cho Prototype.** |
| **Pixel Art Top Down - Basic & Village** | Cainos ([itch.io](https://cainos.itch.io/pixel-art-top-down-basic)) | **32x32** | Free / Commercial Friendly | Nhà gỗ, hàng rào tre gỗ, giếng làng, đồi cỏ hoa, tảng đá lớn rêu phong, bóng đổ tách lớp (separated shadows). Rất sắc nét. |
| **Ancient Chinese Wuxia Tileset** | Cute SCKR / willYEE ([itch.io](https://cutesckr.itch.io/ancient-chinese-wuxia-pixelart-tileset)) | **16x16 & 32x32** | Paid (Thương mại) | Chuẩn phong cách kiếm hiệp Trung Hoa: trà quán, khách điếm, tông môn điện, lầu các, sông nước Giang Nam. Khuyên dùng làm gói nâng cấp đồ họa (Upgrade Path) cho Phase thương mại/Beta. |
| **Medieval Chinese Buildings** | VectoRaith ([itch.io](https://vectoraith.itch.io/)) | **16x16** | Free / Commercial Friendly | Nhà cửa, tường thành, quán xá cổ trấn Á Đông thời phong kiến. |

### 2.2. Spritesheet Nhân Vật (4 Hướng: Idle, Walk, Attack, Hurt)

| Tên Bộ Asset | Tác Giả / Nguồn | Kích Thước | Giấy Phép | Tính Năng & Chuyển Động |
| :--- | :--- | :--- | :--- | :--- |
| **Ninja Adventure Characters** | Pixel-boy ([itch.io](https://pixel-boy.itch.io/ninja-adventure)) | Frame **16x16** (canvas 16x20) | **CC0** | 50+ nhân vật (kiếm sĩ nam, nữ đạo sĩ, võ sư, thích khách, thôn dân) với đủ 4 hướng (Down, Up, Left, Right). Animations: Idle, Walk (4 frames), Attack (3 frames), Hurt/Die. Kèm 30+ quái vật dã ngoại (lợn rừng, sói gai, nhện độc). |
| **Universal LPC Character Generator** | Liberated Pixel Cup ([OpenGameArt](https://sanderfrenken.github.io/Universal-LPC-Spritesheet-Character-Generator/)) | Frame **64x64** | **CC-BY-SA 3.0 / GPL 3.0** | Công cụ ghép tạo nhân vật tùy biến cực mạnh (nam/nữ, kiểu tóc, đạo bào, giáp trụ, vũ khí). Đầy đủ animation: Đi bộ, Vung kiếm, Bắn phép (Cast), Nhận đòn (Hurt), Ngã gục. |
| **PIPOYA FREE RPG Character Sprites** | Pipoya ([itch.io](https://pipoya.itch.io/pipoya-free-rpg-character-sprites-32x32)) | Frame **32x32** | Free Commercial Friendly | Nhân vật phong cách JRPG cổ điển, chuyển động 4 hướng (3-frame walk). Có bộ công cụ CharaMEL đi kèm để phối đồ. |

### 2.3. Hiệu Ứng Kỹ Năng / Phép Thuật (VFX & Projectiles)

| Tên Bộ Asset | Tác Giả / Nguồn | Kích Thước | Giấy Phép | Kỹ Năng Cổ Trùng Ứng Dụng |
| :--- | :--- | :--- | :--- | :--- |
| **Ninja Adventure FX** | Pixel-boy ([itch.io](https://pixel-boy.itch.io/ninja-adventure)) | 16x16 / 32x32 | **CC0** | Vệt chém kiếm khí, làn khói lướt né (Dash smoke), lốc xoáy (Phong đạo), tia sáng ngưng tụ (Quang đạo), bọt khí độc xanh (Độc đạo). |
| **Effect & Bullet Collection** | BDragon1727 ([itch.io](https://bdragon1727.itch.io/)) | 16x16 / 32x32 | Free Commercial Friendly | Đạn đạo ma pháp (Skillshots), vụ nổ va chạm (Impact VFX), cầu lửa, băng tiễn, chùm sét, quầng hào quang kích hoạt Cổ trùng (Aura). |
| **Poison Attacks Mini-Pack** | Sentient Dream Studio ([itch.io](https://itch.io/game-assets/free/tag-pixel-art/tag-vfx)) | 32x32 | Free | Sương độc AoE, đạn nước độc nhổ từ nhện/rết, bãi lầy độc ăn mòn mặt đất. |
| **Free Pixel Effects Packs** | XYEzawr ([itch.io](https://xyezawr.itch.io/)) | 16x16 & 32x32 | Free / CC0 | Vệt máu, tia năng lượng, quả cầu nguyên tố, đạn phép bắn theo đường thẳng. |

### 2.4. Icons Cổ Trùng, Đạo Tài, Dược Thảo, Khoáng Thạch

| Tên Bộ Asset | Tác Giả / Nguồn | Kích Thước | Giấy Phép | Ứng Dụng Cụ Thể Trong GuWorld |
| :--- | :--- | :--- | :--- | :--- |
| **Bugs / Beetles Pixel Pack** | Pixel Gnome / Admurin ([itch.io](https://itch.io/game-assets/tag-bugs/tag-pixel-art)) | **16x16** | Free / CC0 | Hình tượng trực quan cho các Cổ trùng bậc 1-5: Bọ cánh cứng (Thiềm Cổ, Hắc Bì Trư Cổ), Bướm (Tịnh Hồn Điệp), Ve sầu (Xuân Thu Thiền), Bọ cạp, Rết. |
| **Kyrise's Free RPG Icon Pack** | Kyrise ([itch.io](https://kyrise.itch.io/kyrise-free-16x16-rpg-icon-pack)) | **16x16** | Free Commercial Friendly | 350+ icons: Đan dược (Nguyên Thạch, Xá Lợi Cổ), thảo dược luyện cổ, cuộn bí kíp/công thức, khoáng thạch dã ngoại. |
| **Kenney Roguelike / RPG Pack** | Kenney ([kenney.nl](https://kenney.nl/assets/rpg-base)) | **16x16** | **CC0 1.0 Universal** | Hơn 1.700 icons và ô trang bị, hòm kho báu, chìa khóa, túi càn khôn, nguyên liệu cơ bản. |
| **16x16 Flowers & Herbs** | greenpixels_ ([itch.io](https://greenpixels.itch.io/)) | **16x16** | Free | Các loại linh thảo, hoa độc, rễ cây thu thập trên núi Thanh Mao. |

---

## 3. Khuyến Nghị Kỹ Thuật Cho Phaser 3 Canvas

### 3.1. Chuẩn Kích Thước Khuyến Nghị
* **Tile Size:** **16x16 pixel**.  
  * *Lý do:* Tương thích 100% với pack `Ninja Adventure` (pack đã đồng bộ cả Tileset + Nhân vật + VFX + Quái vật + SFX dưới chuẩn CC0). Đồng thời tải cực nhanh trên thiết bị di động / mạng yếu.
* **Độ Phân Giải Game (Viewport) & Scaling:**
  - Game canvas: `width: 480, height: 270` (tỉ lệ 16:9).
  - Tỷ lệ phóng đại Integer Scaling x4 = 1920x1080 (Full HD) không bị mờ vỡ pixel.
  - Bật `pixelArt: true` và `roundPixels: true`.

### 3.2. Body Hitbox & Depth Sorting (Z-Order)
* **Kích thước Hitbox nhân vật:** Frame hiển thị 16x20 px, nhưng thân vật lý (Arcade Body) chỉ đặt ở bàn chân `width: 10px, height: 6px, offset: (3, 14)` để nhân vật có thể đi xuyên/nấp sau mái nhà, thân cây hoặc bờ rào.
* **Depth Sorting:** Trong mỗi Game Loop `update()`, cập nhật `sprite.setDepth(sprite.y)` để vật thể phía dưới che vật thể phía trên một cách tự nhiên.

### 3.3. Tối Ưu Tải Tài Nguyên (Texture Atlas)
* Đóng gói toàn bộ Frame Animation của VFX và Icon vào Atlas (`vfx_atlas.png` + `vfx_atlas.json`).
* Giảm số lượng HTTP Requests khi vào game xuống dưới 5 requests chính.

---

## 4. Cấu Trúc Thư Mục Asset Đề Xuất Cho Dự Án Mới

```text
guworld-online/
├── client/
│   ├── public/
│   │   └── assets/
│   │       ├── maps/
│   │       │   ├── tilesets/
│   │       │   │   ├── wuxia_village_16x16.png
│   │       │   │   ├── nature_cliffs_16x16.png
│   │       │   │   └── water_interior_16x16.png
│   │       │   └── tmx/
│   │       │       ├── qingmao_spawn_village.json
│   │       │       └── wild_bamboo_forest.json
│   │       ├── characters/
│   │       │   ├── players/
│   │       │   │   ├── gu_master_male.png
│   │       │   │   └── gu_master_female.png
│   │       │   ├── npcs/
│   │       │   │   ├── elder_gu_yue.png
│   │       │   │   └── merchant.png
│   │       │   └── monsters/
│   │       │       ├── wild_boar.png
│   │       │       ├── jade_centipede.png
│   │       │       └── shadow_wolf.png
│   │       ├── vfx/
│   │       │   ├── vfx_atlas.png
│   │       │   ├── vfx_atlas.json
│   │       │   └── skills/
│   │       │       ├── moon_blade.png
│   │       │       ├── poison_spray.png
│   │       │       └── dash_smoke.png
│   │       ├── icons/
│   │       │   ├── gu_worms/
│   │       │   ├── items/
│   │       │   └── ui/
│   │       └── audio/
│   │           ├── sfx/
│   │           └── bgm/
```

---

## 5. Kết Luận & Lộ Trình Triển Khai (Action Items)

1. **Giai đoạn Prototype 0.1 (Ngay lập tức):**
   * Sử dụng trọn vẹn bộ **Ninja Adventure Pack (CC0)** làm nòng cốt: Vừa có bản đồ làng mạc Á Đông, nhân vật 4 hướng, quái vật rừng, vừa có animation chưởng khí/lướt né đồng nhất 16x16.
   * Bổ sung gói **Bugs Pack** và **Kyrise RPG Icons** để hiển thị giao diện Cổ trùng và túi đồ.
2. **Giai đoạn Alpha / Beta (Nâng cấp thẩm mỹ):**
   * Tích hợp **Cute SCKR Ancient Chinese Wuxia Tileset** để tăng độ hoành tráng cho Tông môn và thành trì lớn.
   * Dùng **Universal LPC Spritesheet Generator** để mở khóa tính năng thay đổi ngoại hình/trang bị theo thời gian thực (Paperdoll System).
