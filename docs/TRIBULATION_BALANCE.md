# Cân Bằng Quái Kiếp Nạn & Thưởng Đạo Ngân

Nguồn: `backend/internal/handlers/tribulation_trigger.go` (`spawnTribulationEventWorker`), `backend/internal/models/monster.go` (`ScaleMonsterStats`), `backend/internal/models/aperture.go` (`GetTribulationRankAndSubRank`, `CalculateTribulationReward`), `backend/data/Monster.csv`.

## 1. Công thức sức mạnh

```
HP/ATK/DEF = base × RankScale(rank quái) × typeMult × subRankMult × calamityMult
```

- `base`: chỉ số gốc trong `Monster.csv` (rank-independent).
- `RankScale`: R6 = 10⁴, R7 = 10⁶, R8 = 10⁸, R9 = 10¹⁰; quái rank 10 (Hỗn Độn/Hỗn Loạn) dùng 10¹³.
- `typeMult`: Boss kiếp = **10**, Minion kiếp = **3**.
- Hệ số hạ công/thủ: Boss ATK ×0.05, DEF ×0.02; Minion ATK ×0.1, DEF ×0.1.
- `subRankMult = 1 + 0.2 × (subRank − 1)`. Chỉ Boss có subRank theo loại kiếp; Minion luôn subRank 1.
- `calamityMult`: **chỉ Kiếp Nạn Hỗn Loạn (CHAOTIC_TRIBULATION)**, áp vào HP + ATK (không áp DEF):
  `1 + (1 − purity) × 0.75` → **1.30×** (purity 0.60) đến **1.675×** (purity 0.10).
- **Không còn** scale theo độ khó (`difficulty`) hay theo lượng Đạo Ngân của người chơi.

Rank quái theo **loại kiếp**; riêng Hỗn Loạn lấy đúng **rank người chơi**:

| Loại kiếp | Boss template | Rank | SubRank |
|---|---|---|---|
| Địa Tai (EARTH_CALAMITY) | `monster_earth_calamity_boss` | 6 | 1 |
| Thiên Kiếp (HEAVENLY_TRIBULATION) | `monster_heaven_tribulation_boss` | 7 | 2 |
| Hạo Kiếp (GRAND_TRIBULATION) | `monster_grand_calamity_boss` | 8 | 3 |
| Vạn Kiếp (MYRIAD_TRIBULATION) | `monster_myriad_tribulation_boss` | 9 | 4 |
| Hỗn Độn (CHAOS_CALAMITY) | `monster_chaos_calamity_boss` | 10 | 4 |
| Hỗn Loạn (CHAOTIC_TRIBULATION) | `monster_chaos_calamity_boss` | = rank người chơi (6–10) | 4 |

## 2. Sức mạnh quái (chưa tính calamityMult)

### Boss

| Kiếp | Base HP / ATK / DEF | HP | ATK | DEF |
|---|---|---|---|---|
| Địa Tai | 247 / 29.9 / 0.34 | 24.700.000 | 149.500 | 680 |
| Thiên Kiếp | 221 / 32.5 / 0.28 | 2.652.000.000 | 19.500.000 | 67.200 |
| Hạo Kiếp | 221 / 32.5 / 0.28 | 309.400.000.000 | 2.275.000.000 | 7.840.000 |
| Vạn Kiếp | 247 / 29.9 / 0.34 | 39.520.000.000.000 | 239.200.000.000 | 1.088.000.000 |
| Hỗn Độn | 247 / 29.9 / 0.34 | 39.520.000.000.000.000 | 239.200.000.000.000 | 1.088.000.000.000 |

**Hỗn Loạn** dùng template Hỗn Độn nhưng scale theo rank người chơi:

| Rank người chơi | HP | ATK | DEF |
|---|---|---|---|
| 6 | 39.520.000 | 239.200 | 1.088 |
| 7 | 3.952.000.000 | 23.920.000 | 108.800 |
| 8 | 395.200.000.000 | 2.392.000.000 | 10.880.000 |
| 9 | 39.520.000.000.000 | 239.200.000.000 | 1.088.000.000 |
| 10 | 39.520.000.000.000.000 | 239.200.000.000.000 | 1.088.000.000.000 |

### Minion (`monster_tribulation_beast`, 160 / 16 / 0.25)

| Kiếp | Rank minion | HP | ATK | DEF |
|---|---|---|---|---|
| Địa Tai | 6 | 4.800.000 | 48.000 | 750 |
| Thiên Kiếp | 7 | 480.000.000 | 4.800.000 | 75.000 |
| Hạo Kiếp | 8 | 48.000.000.000 | 480.000.000 | 7.500.000 |
| Vạn Kiếp | 9 | 4.800.000.000.000 | 48.000.000.000 | 750.000.000 |
| Hỗn Độn | 10 | 4.800.000.000.000.000 | 48.000.000.000.000 | 750.000.000.000 |

- Minion Hỗn Loạn lấy rank = rank người chơi: tra bảng trên theo rank tương ứng (R6 → dòng Địa Tai, R7 → Thiên Kiếp, R8 → Hạo Kiếp, R9 → Vạn Kiếp, R10 → Hỗn Độn).

- Số minion: `1 + rand.Intn(playerRank − 3)` → rank 6: 1–3, rank 7: 1–4, rank 8: 1–5, rank 9: 1–6.
- Hỗn Loạn nhân thêm `calamityMult` vào HP và ATK của cả boss lẫn minion.

## 3. Thưởng Đạo Ngân (cố định)

Tính theo **rank người chơi** + loại kiếp. Không scale theo số lần độ kiếp, độ khó, hay Đạo Ngân tích lũy.

| Rank người chơi | Địa Tai | Thiên Kiếp | Hạo Kiếp | Vạn Kiếp | Hỗn Độn | Hỗn Loạn |
|---|---|---|---|---|---|---|
| 6 | 250 | 750 | — | — | — | 250 |
| 7 | 1.500 | 5.000 | 20.000 | — | — | 1.500 |
| 8 | — | 10.000 | 40.000 | 150.000 | — | 10.000 |
| 9 | — | — | 50.000 | 200.000 | 1.500.000 | 1.500.000 |
| 10 | — | — | — | — | 1.500.000 | 1.500.000 |

- `—`: loại kiếp không nằm trong lịch của rank đó (`GetTribulationTypesForRank`).
- Hỗn Loạn dùng bảng thưởng Hỗn Độn của rank người chơi (`CalculateTribulationReward` map `CHAOTIC_TRIBULATION` → `CHAOS_CALAMITY`).
- Thưởng thực nhận còn nhân `rewardMult` (hiện luôn = 1.0) và hệ số `DaoMarkGainBonus` của thể chất khi cộng vào Tiên Khiếu.

## 4. Ghi chú cân bằng

- Quái rank 10 (Hỗn Độn) mạnh hơn rank 9 **1000×** (`10¹³ / 10¹⁰`).
- Hỗn Loạn giờ scale theo rank người chơi, nên không còn cảnh rank 6–8 phải đánh boss rank 10; thưởng cũng tính theo rank người chơi nên khớp độ khó.
- Không còn cơ chế nào khiến quái mạnh lên theo số lần độ kiếp; kiếp thứ 1 và kiếp thứ N của cùng một loại có cùng chỉ số.
