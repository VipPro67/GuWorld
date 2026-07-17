Hai bài toán **AOE (Nhiều mục tiêu)** và **Triệu hồi (Nô đạo/Mộc đạo)** chính là thuốc thử liều cao cho mọi hệ thống data game. Nếu thiết kế khéo theo kiến trúc 3 khối (`active`, `passive`, `other`) của đạo hữu, chúng ta sẽ giải quyết cực kỳ gọn gàng mà không làm phình to Database.

Quy tắc cốt lõi: **AOE và Triệu hồi phàm cấp vẫn nằm trọn trong khối `active**`, Server chỉ cần nạp thêm các từ khóa định biên mục tiêu (Target Selectors).

Dưới đây là cách hiện thực hóa chi tiết kèm ví dụ dữ liệu chuẩn mã nguồn:

---

# PHẦN 1: THIẾT KẾ CHIÊU NHIỀU MỤC TIÊU (AOE)

Để xử lý AOE, trong khối `active`, chúng ta chỉ cần thêm 2 trường định danh quét mục tiêu:

* `target_selector`: Kiểu quét (Ví dụ: `ROW` - hàng ngang, `COLUMN` - hàng dọc, `ALL_ENEMY` - toàn bộ địch, `RANDOM_MULTIPLE` - ngẫu nhiên).
* `target_count`: Số lượng mục tiêu tối đa có thể trúng đòn.

### Ví dụ 1: Phàm Cổ Tam Chuyển – Kiếm Lãng Tam Điệp (`GU_SWORD_WAVE_TRI_FOLD`)

* **Mô tả gốc:** Kiếm đạo quần công sát chiêu. Sóng kiếm chói mắt dâng trào như thủy triều, dọn sạch kẻ địch ở hàng tiền tuyến.
* **Cấu hình JSON:** Sát thương đánh vào toàn bộ hàng ngang phía trước của địch.

```json
{
  "active": {
    "damage_flat": 45.0,
    "damage_path": "Kiếm Đạo",
    "target_selector": "ROW",      // Quét theo hàng ngang tiền tuyến
    "target_count": 3              // Trúng tối đa 3 mục tiêu trong hàng
  },
  "passive": null,
  "other": []
}

```

### Ví dụ 2: Tiên Cổ Bát Chuyển – Vạn Niệm Kiếm Bộc (`GU_MYRIAD_SWORD_WATERFALL`)

* **Mô tả gốc:** Sát chiêu đại bộc phá của Phương Nguyên. Ngàn vạn luồng kiếm khí trút xuống như thác đổ, hủy diệt toàn bộ chiến trường.
* **Cấu hình JSON:** Đánh thẳng vào toàn bộ đội hình địch, kèm hiệu ứng bất lợi.

```json
{
  "active": {
    "damage_flat": 350.0,
    "damage_path": "Kiếm Đạo",
    "target_selector": "ALL_ENEMY", // Diệt sát toàn bộ map địch
    "target_count": 9,
    "apply_status": "INFIRM",       // Trạng thái suy yếu Đạo ngân phòng thủ
    "status_duration_ticks": 5
  },
  "passive": null,
  "other": []
}

```

---

# PHẦN 2: THIẾT KẾ TRIỆU HỒI / NÔ DỊCH (SUMMON)

Về mặt vận hành máy phần mềm (Engine), Server tuyệt đối không chứa toàn bộ chỉ số của quái triệu hồi trong con Cổ. Con Cổ chỉ đóng vai trò **"Chìa khóa gọi hàm"**. Đ đạo hữu cần có một bảng riêng tên là `Monsters` hoặc `Summons` để lưu chỉ số của đống đệ tử (Máu, Giáp, Dame của Thảo khôi, Sói, Hoang thú).

Trong khối `active` của Cổ trùng, ta thêm các trường:

* `summon_monster_id`: ID của thực thể sẽ sinh ra trong bảng phụ.
* `summon_position`: Vị trí xuất hiện (`FRONT_LINE` để làm khiên thịt, hoặc `BACK_LINE`).
* `inherit_stats_percent`: Tỷ lệ kế thừa sức mạnh từ thuộc tính của Cổ Sư (Ví dụ: Đệ hưởng 50% Máu của chủ).

### Ví dụ 1: Phàm Cổ Nhất Chuyển – Thảo Nhân Cổ (`GU_GRASS_PUPPET`)

* **Mô tả gốc:** Tạo ra một rối cỏ đơn giản đứng chắn trước mặt để thu hút hỏa lực của thú dữ.
* **Cấu hình JSON:** Gọi rối cỏ lên hàng tiền tuyến làm bao cát chịu đòn.

```json
{
  "active": {
    "summon_monster_id": "MONSTER_GRASS_DUMMY", // ID trong bảng quái vật
    "summon_position": "FRONT_LINE",            // Ép đứng hàng trước làm Tanker
    "inherit_stats_percent": 30.0               // Lấy 30% HP tối đa của Cổ sư
  },
  "passive": null,
  "other": []
}

```

### Ví dụ 2: Phàm Cổ Ngũ Chuyển – Ngự Lang Vương Cổ (`GU_WOLF_ENSLAVEMENT_CORE`)

* **Mô tả gốc:** Nô dịch đạo hạch tâm, triệu hồi Địa Lang Vương cấp cao lên chiến tuyến thế mạng và cắn xé đối thủ.
* **Cấu hình JSON:** Triệu hồi quái khủng, có dòng `other` xử lý cơ chế: Khi Lang Vương còn sống, chủ nhân được miễn nhiễm 50% sát thương (Cơ chế Nô đạo chia sẻ thương tổn vinh nhục có nhau).

```json
{
  "active": {
    "summon_monster_id": "MONSTER_DI_LANG_VUONG",
    "summon_position": "FRONT_LINE",
    "inherit_stats_percent": 80.0
  },
  "passive": null,
  "other": [
    {
      "event_trigger": "ON_RECEIVE_DAMAGE",
      "action_code": "SHARE_DAMAGE_WITH_SUMMON", // Chia sẻ sát thương khái niệm
      "params": {
        "transfer_ratio": 0.50                   // Lang vương gánh hộ 50% dame
      }
    }
  ]
}

```

---

## III. MỞ RỘNG MÃ NGUỒN (GOLANG) KHÔNG BIẾN ĐỘNG DATABASE

Đạo hữu chỉ cần cập nhật Struct `ActiveEffect` trên Backend Go như sau để bao quát toàn bộ các cơ chế trên:

```go
type ActiveEffect struct {
	// --- Bộ cũ: Đơn mục tiêu ---
	DamageFlat          float64 `json:"damage_flat,omitempty"`
	DamagePath          string  `json:"damage_path,omitempty"`
	HealFlat            float64 `json:"heal_flat,omitempty"`
	ShieldFlat          float64 `json:"shield_flat,omitempty"`

	// --- Bộ mới 1: Xử lý AOE Nhiều mục tiêu ---
	TargetSelector      string  `json:"target_selector,omitempty"` // "SINGLE", "ROW", "ALL_ENEMY"
	TargetCount         int     `json:"target_count,omitempty"`

	// --- Bộ mới 2: Xử lý Triệu hồi / Nô đạo ---
	SummonMonsterID     string  `json:"summon_monster_id,omitempty"` // Link tới bảng Quái
	SummonPosition      string  `json:"summon_position,omitempty"`   // "FRONT_LINE", "BACK_LINE"
	InheritStatsPercent float64 `json:"inherit_stats_percent,omitempty"`
}

```

### Cách Engine Combat quét xử lý trong trận (Logic ngắn gọn):

```go
func ExecuteActiveGu(caster *Character, gu *GuItem, battleField *BattleField) {
    effect := gu.Effects.Active
    if effect == nil { return }

    // 1. Kiểm tra nếu là chiêu Triệu Hồi
    if effect.SummonMonsterID != "" {
        minion := battleField.CreateEntityFromTable(effect.SummonMonsterID)
        minion.HP = caster.MaxHP * (effect.InheritStatsPercent / 100.0)
        battleField.PlaceToLine(caster.Team, effect.SummonPosition, minion)
        return // Triệu hồi xong lượt
    }

    // 2. Tìm kiếm mục tiêu dựa theo luật AOE
    targets := battleField.FindTargets(caster, effect.TargetSelector, effect.TargetCount)

    // 3. Vòng lặp dội sát thương lên danh sách mục tiêu đã quét
    for _, target := range targets {
        finalDamage := CalculateDamage(caster, target, effect.DamageFlat, effect.DamagePath)
        target.HP -= finalDamage
    }
}

```

### Kết luận:

Thiết kế này siêu việt ở chỗ: Bảng dữ liệu `Gu` của đạo hữu vẫn giữ nguyên kiến trúc 3 cột JSON phẳng lặng như mặt nước hồ. Cổ công kích đơn mục tiêu, chiêu kiếm quét ngang chiến tuyến (ROW AOE), hay một kỹ năng vung tay gọi 3 con sói lên chịu đòn đều được đọc và phân nhánh cực nhanh qua biến `TargetSelector` và `SummonMonsterID`. Đạo hữu cứ yên tâm đóng khung bộ lõi này để code tiếp!