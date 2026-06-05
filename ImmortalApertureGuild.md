Dưới đây là bản triển khai hợp lý hơn cho game của bạn, bám sát **Cổ Chân Nhân** nhưng vẫn dễ code bằng Go/Gin/Gorm + Next.js.

---

# 1. Core Loop Endgame

Sau khi player đạt **Rank 5 Đỉnh Phong**, phá Phàm Khiếu để lên **Rank 6**, hệ thống mở khóa:

1. Tạo **Tiên Khiếu**
2. Sinh **Tiên Nguyên**
3. Nuôi **Resource Node** trong Tiên Khiếu
4. Chờ **Địa Tai / Thiên Kiếp**
5. Vượt kiếp để nhận **Đạo Ngân**
6. Dùng Đạo Ngân buff Cổ Trùng / Sát Chiêu
7. Tăng Rank → tai kiếp mạnh hơn → reward lớn hơn

Vòng lặp chính:

```txt
Farm tài nguyên → Build Tiên Khiếu → Chuẩn bị phòng thủ → Tai kiếp tới → Combat/Boss Raid → Nhận Đạo Ngân → Mạnh hơn
```

---

# 2. Database Design

## immortal_apertures

Mỗi player chỉ có 1 Tiên Khiếu chính.

```sql
CREATE TABLE immortal_apertures (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id UUID NOT NULL UNIQUE,
    grade VARCHAR(20) NOT NULL,
    time_rate NUMERIC NOT NULL DEFAULT 10,
    aperture_hp INT NOT NULL DEFAULT 10000,
    max_aperture_hp INT NOT NULL DEFAULT 10000,
    area_size BIGINT NOT NULL DEFAULT 3000000,
    immortal_essence INT NOT NULL DEFAULT 0,
    essence_production_per_year INT NOT NULL DEFAULT 10,
    internal_years_passed NUMERIC NOT NULL DEFAULT 0,
    next_tribulation_year NUMERIC NOT NULL DEFAULT 10,
    tribulation_count INT NOT NULL DEFAULT 0,
    state VARCHAR(30) NOT NULL DEFAULT 'STABLE',
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);
```

`state` nên có:

```txt
STABLE
TRIBULATION_PENDING
TRIBULATION_ACTIVE
DAMAGED
DESTROYED
```

---

## dao_marks

Không nên lưu `map[string]int` trực tiếp trong DB nếu bạn muốn query/sort/ranking dễ.

```sql
CREATE TABLE dao_marks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id UUID NOT NULL,
    path VARCHAR(50) NOT NULL,
    amount INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now(),
    UNIQUE(player_id, path)
);
```

Ví dụ path:

```txt
fire
water
ice
blood
sword
strength
wisdom
time
space
soul
enslavement
luck
refinement
formation
```

---

## aperture_resource_nodes

Tài nguyên trong Tiên Khiếu: linh tuyền, rừng trúc, đàn hoang thú, mỏ nguyên thạch, hoa đạo ngân...

```sql
CREATE TABLE aperture_resource_nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aperture_id UUID NOT NULL,
    node_type VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    path VARCHAR(50),
    level INT NOT NULL DEFAULT 1,
    hp INT NOT NULL DEFAULT 1000,
    max_hp INT NOT NULL DEFAULT 1000,
    x INT NOT NULL,
    y INT NOT NULL,
    production_type VARCHAR(50),
    production_amount INT NOT NULL DEFAULT 0,
    production_interval_hours INT NOT NULL DEFAULT 24,
    last_collected_at TIMESTAMP,
    is_destroyed BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);
```

Ví dụ node:

```txt
Moon Orchid Field
Blood Beast Den
Ice Crystal Lake
Primeval Stone Mine
Soul Fog Forest
Time River Fragment
```

---

## tribulation_events

Lưu mỗi lần tai kiếp.

```sql
CREATE TABLE tribulation_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aperture_id UUID NOT NULL,
    player_id UUID NOT NULL,
    tribulation_type VARCHAR(30) NOT NULL,
    path VARCHAR(50) NOT NULL,
    difficulty INT NOT NULL DEFAULT 1,
    status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',
    aperture_hp_before INT NOT NULL,
    aperture_hp_after INT,
    reward_dao_marks INT NOT NULL DEFAULT 0,
    started_at TIMESTAMP DEFAULT now(),
    ended_at TIMESTAMP
);
```

`tribulation_type`:

```txt
EARTH_CALAMITY
HEAVENLY_TRIBULATION
GRAND_TRIBULATION
MYRIAD_TRIBULATION
```

---

## tribulation_enemies

Boss/quái sinh ra trong tai kiếp.

```sql
CREATE TABLE tribulation_enemies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    enemy_type VARCHAR(50) NOT NULL,
    path VARCHAR(50) NOT NULL,
    hp INT NOT NULL,
    max_hp INT NOT NULL,
    attack INT NOT NULL,
    defense INT NOT NULL,
    target_node_id UUID,
    x INT NOT NULL,
    y INT NOT NULL,
    is_boss BOOLEAN NOT NULL DEFAULT false,
    is_dead BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);
```

---

# 3. Go Model

```go
type ImmortalAperture struct {
	ID                       string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	PlayerID                 string    `gorm:"type:uuid;uniqueIndex;not null" json:"player_id"`
	Grade                    string    `gorm:"type:varchar(20);not null" json:"grade"`
	TimeRate                 float64   `gorm:"not null;default:10" json:"time_rate"`
	ApertureHP               int       `gorm:"not null;default:10000" json:"aperture_hp"`
	MaxApertureHP            int       `gorm:"not null;default:10000" json:"max_aperture_hp"`
	AreaSize                 int64     `gorm:"not null;default:3000000" json:"area_size"`
	ImmortalEssence          int       `gorm:"not null;default:0" json:"immortal_essence"`
	EssenceProductionPerYear int       `gorm:"not null;default:10" json:"essence_production_per_year"`
	InternalYearsPassed      float64   `gorm:"not null;default:0" json:"internal_years_passed"`
	NextTribulationYear      float64   `gorm:"not null;default:10" json:"next_tribulation_year"`
	TribulationCount         int       `gorm:"not null;default:0" json:"tribulation_count"`
	State                    string    `gorm:"type:varchar(30);not null;default:'STABLE'" json:"state"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}
```

```go
type DaoMark struct {
	ID        string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	PlayerID  string    `gorm:"type:uuid;not null;uniqueIndex:idx_player_path" json:"player_id"`
	Path      string    `gorm:"type:varchar(50);not null;uniqueIndex:idx_player_path" json:"path"`
	Amount    int       `gorm:"not null;default:0" json:"amount"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
```

```go
type TribulationEvent struct {
	ID                string     `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ApertureID        string     `gorm:"type:uuid;not null" json:"aperture_id"`
	PlayerID          string     `gorm:"type:uuid;not null" json:"player_id"`
	TribulationType   string     `gorm:"type:varchar(30);not null" json:"tribulation_type"`
	Path              string     `gorm:"type:varchar(50);not null" json:"path"`
	Difficulty        int        `gorm:"not null;default:1" json:"difficulty"`
	Status            string     `gorm:"type:varchar(30);not null;default:'ACTIVE'" json:"status"`
	ApertureHPBefore  int        `gorm:"not null" json:"aperture_hp_before"`
	ApertureHPAfter   *int       `json:"aperture_hp_after"`
	RewardDaoMarks   int        `gorm:"not null;default:0" json:"reward_dao_marks"`
	StartedAt         time.Time  `json:"started_at"`
	EndedAt           *time.Time `json:"ended_at"`
}
```

---

# 4. Tạo Tiên Khiếu khi thăng tiên

Điểm quyết định grade nên dựa vào:

```txt
humanQi
soulPower
rank5Foundation
pathAttainment
numberOfGu
specialPhysique
```

Công thức đơn giản:

```go
func CalculateApertureGrade(score int) string {
	if score >= 900 {
		return "EXTREME"
	}

	if score >= 700 {
		return "HIGH"
	}

	if score >= 400 {
		return "MID"
	}

	return "LOW"
}
```

```go
func BuildApertureByGrade(playerID string, grade string) ImmortalAperture {
	switch grade {
	case "EXTREME":
		return ImmortalAperture{
			PlayerID:                 playerID,
			Grade:                    grade,
			TimeRate:                 40,
			ApertureHP:               50000,
			MaxApertureHP:            50000,
			AreaSize:                 10000000,
			EssenceProductionPerYear: 50,
			NextTribulationYear:      10,
			State:                    "STABLE",
		}
	case "HIGH":
		return ImmortalAperture{
			PlayerID:                 playerID,
			Grade:                    grade,
			TimeRate:                 30,
			ApertureHP:               30000,
			MaxApertureHP:            30000,
			AreaSize:                 9000000,
			EssenceProductionPerYear: 30,
			NextTribulationYear:      10,
			State:                    "STABLE",
		}
	case "MID":
		return ImmortalAperture{
			PlayerID:                 playerID,
			Grade:                    grade,
			TimeRate:                 20,
			ApertureHP:               20000,
			MaxApertureHP:            20000,
			AreaSize:                 6000000,
			EssenceProductionPerYear: 20,
			NextTribulationYear:      10,
			State:                    "STABLE",
		}
	default:
		return ImmortalAperture{
			PlayerID:                 playerID,
			Grade:                    grade,
			TimeRate:                 10,
			ApertureHP:               10000,
			MaxApertureHP:            10000,
			AreaSize:                 3000000,
			EssenceProductionPerYear: 10,
			NextTribulationYear:      10,
			State:                    "STABLE",
		}
	}
}
```

---

# 5. Internal Time System

Không nên dùng `NextTribulationAt time.Time` làm nguồn chính. Vì Tiên Khiếu có thể bị tăng/giảm time rate.

Nên lưu:

```txt
internal_years_passed
next_tribulation_year
time_rate
```

Mỗi worker tick sẽ cộng năm nội bộ:

```go
func AdvanceApertureTime(aperture *ImmortalAperture, realSeconds float64) {
	realDays := realSeconds / 86400
	internalDays := realDays * aperture.TimeRate
	internalYears := internalDays / 365
	aperture.InternalYearsPassed += internalYears
}
```

Ví dụ:

```txt
time_rate = 30
1 ngày thật = 30 ngày trong Tiên Khiếu
12.16 ngày thật ≈ 1 năm trong Tiên Khiếu
121.6 ngày thật ≈ 10 năm trong Tiên Khiếu
```

Nếu muốn game nhanh hơn, bạn có thể scale lại:

```txt
1 giờ thật = 1 năm Tiên Khiếu ở time_rate 10
```

Như vậy tai kiếp sẽ tới trong vài giờ/ngày, hợp game hơn.

---

# 6. Xác định loại Tai Kiếp

Theo rank:

```go
func GetTribulationType(rank int, tribulationCount int) string {
	if rank == 6 {
		if tribulationCount > 0 && tribulationCount%10 == 0 {
			return "HEAVENLY_TRIBULATION"
		}

		return "EARTH_CALAMITY"
	}

	if rank == 7 {
		if tribulationCount > 0 && tribulationCount%10 == 0 {
			return "GRAND_TRIBULATION"
		}

		if tribulationCount > 0 && tribulationCount%5 == 0 {
			return "HEAVENLY_TRIBULATION"
		}

		return "EARTH_CALAMITY"
	}

	if rank == 8 {
		if tribulationCount > 0 && tribulationCount%10 == 0 {
			return "MYRIAD_TRIBULATION"
		}

		if tribulationCount > 0 && tribulationCount%5 == 0 {
			return "GRAND_TRIBULATION"
		}

		return "HEAVENLY_TRIBULATION"
	}

	return "EARTH_CALAMITY"
}
```

---

# 7. Worker check Tai Kiếp

```go
func CheckAndTriggerTribulations(db *gorm.DB) error {
	var apertures []ImmortalAperture

	err := db.Where("state = ?", "STABLE").Find(&apertures).Error
	if err != nil {
		return err
	}

	for i := range apertures {
		aperture := apertures[i]

		AdvanceApertureTime(&aperture, 60)

		if aperture.InternalYearsPassed >= aperture.NextTribulationYear {
			err := SpawnTribulationEvent(db, &aperture)
			if err != nil {
				continue
			}
		} else {
			db.Save(&aperture)
		}
	}

	return nil
}
```

Worker mỗi phút:

```go
func StartTribulationWorker(db *gorm.DB) {
	ticker := time.NewTicker(time.Minute)

	go func() {
		for range ticker.C {
			_ = CheckAndTriggerTribulations(db)
		}
	}()
}
```

---

# 8. Spawn Tai Kiếp

Thiên Ý nên chọn path khắc chế hoặc path player có nhiều tài nguyên nhất.

Ví dụ:

```go
func PickTribulationPath(daoMarks []DaoMark) string {
	if len(daoMarks) == 0 {
		return "fire"
	}

	sort.Slice(daoMarks, func(i, j int) bool {
		return daoMarks[i].Amount > daoMarks[j].Amount
	})

	mainPath := daoMarks[0].Path

	counter := map[string]string{
		"fire":       "water",
		"water":      "earth",
		"ice":        "fire",
		"blood":      "light",
		"sword":      "defense",
		"strength":   "soul",
		"soul":       "wisdom",
		"wisdom":     "strength",
		"time":       "space",
		"space":      "time",
		"enslavement": "soul",
	}

	if value, ok := counter[mainPath]; ok {
		return value
	}

	return mainPath
}
```

```go
func SpawnTribulationEvent(db *gorm.DB, aperture *ImmortalAperture) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var player Player
		if err := tx.First(&player, "id = ?", aperture.PlayerID).Error; err != nil {
			return err
		}

		var daoMarks []DaoMark
		if err := tx.Where("player_id = ?", aperture.PlayerID).Find(&daoMarks).Error; err != nil {
			return err
		}

		tribulationType := GetTribulationType(player.Rank, aperture.TribulationCount+1)
		path := PickTribulationPath(daoMarks)
		difficulty := CalculateTribulationDifficulty(aperture, daoMarks, tribulationType)

		event := TribulationEvent{
			ApertureID:       aperture.ID,
			PlayerID:         aperture.PlayerID,
			TribulationType:  tribulationType,
			Path:             path,
			Difficulty:       difficulty,
			Status:           "ACTIVE",
			ApertureHPBefore: aperture.ApertureHP,
			RewardDaoMarks:  CalculateDaoMarkReward(tribulationType, difficulty),
			StartedAt:        time.Now(),
		}

		if err := tx.Create(&event).Error; err != nil {
			return err
		}

		enemies := GenerateTribulationEnemies(event.ID, path, difficulty, tribulationType)
		if err := tx.Create(&enemies).Error; err != nil {
			return err
		}

		aperture.State = "TRIBULATION_ACTIVE"
		if err := tx.Save(aperture).Error; err != nil {
			return err
		}

		return nil
	})
}
```

---

# 9. Tính độ khó Tai Kiếp

Tai kiếp không nên chỉ scale theo rank. Nên scale theo:

```txt
Rank
Số lần đã vượt kiếp
Tổng Đạo Ngân
Số resource node
Grade Tiên Khiếu
TimeRate
```

```go
func CalculateTribulationDifficulty(aperture *ImmortalAperture, daoMarks []DaoMark, tribulationType string) int {
	totalDaoMarks := 0
	for _, mark := range daoMarks {
		totalDaoMarks += mark.Amount
	}

	base := 1
	base += aperture.TribulationCount
	base += totalDaoMarks / 1000
	base += int(aperture.TimeRate / 10)

	switch tribulationType {
	case "HEAVENLY_TRIBULATION":
		base *= 3
	case "GRAND_TRIBULATION":
		base *= 10
	case "MYRIAD_TRIBULATION":
		base *= 50
	}

	if aperture.Grade == "HIGH" {
		base += 5
	}

	if aperture.Grade == "EXTREME" {
		base += 15
	}

	if base < 1 {
		return 1
	}

	return base
}
```

---

# 10. Reward Đạo Ngân

```go
func CalculateDaoMarkReward(tribulationType string, difficulty int) int {
	base := 250

	switch tribulationType {
	case "EARTH_CALAMITY":
		base = 250
	case "HEAVENLY_TRIBULATION":
		base = 750
	case "GRAND_TRIBULATION":
		base = 7250
	case "MYRIAD_TRIBULATION":
		base = 86750
	}

	return base + difficulty*25
}
```

Khi thắng:

```go
func CompleteTribulationSuccess(db *gorm.DB, eventID string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var event TribulationEvent
		if err := tx.First(&event, "id = ?", eventID).Error; err != nil {
			return err
		}

		var aperture ImmortalAperture
		if err := tx.First(&aperture, "id = ?", event.ApertureID).Error; err != nil {
			return err
		}

		if err := AddDaoMarks(tx, event.PlayerID, event.Path, event.RewardDaoMarks); err != nil {
			return err
		}

		now := time.Now()
		hpAfter := aperture.ApertureHP

		event.Status = "SUCCESS"
		event.ApertureHPAfter = &hpAfter
		event.EndedAt = &now

		aperture.State = "STABLE"
		aperture.TribulationCount += 1
		aperture.NextTribulationYear += 10

		if err := tx.Save(&event).Error; err != nil {
			return err
		}

		if err := tx.Save(&aperture).Error; err != nil {
			return err
		}

		return nil
	})
}
```

```go
func AddDaoMarks(tx *gorm.DB, playerID string, path string, amount int) error {
	var mark DaoMark

	err := tx.Where("player_id = ? AND path = ?", playerID, path).First(&mark).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		mark = DaoMark{
			PlayerID: playerID,
			Path:     path,
			Amount:   amount,
		}

		return tx.Create(&mark).Error
	}

	if err != nil {
		return err
	}

	mark.Amount += amount
	return tx.Save(&mark).Error
}
```

---

# 11. Damage Formula với Đạo Ngân

Công thức nên dùng:

```txt
Final Damage = Base Damage × Dao Mark Multiplier × Rank Multiplier × Element Counter
```

Bản đơn giản:

```go
func CalculateDamage(baseDamage float64, path string, playerID string, db *gorm.DB) float64 {
	var mark DaoMark

	err := db.Where("player_id = ? AND path = ?", playerID, path).First(&mark).Error
	if err != nil {
		return baseDamage
	}

	multiplier := 1 + float64(mark.Amount)/100

	return baseDamage * multiplier
}
```

Nhưng để balance game tốt hơn, không nên để 100 đạo ngân = x2 mãi mãi, vì late game sẽ vỡ damage rất nhanh.

Nên dùng soft cap:

```go
func CalculateDaoMarkMultiplier(amount int) float64 {
	if amount <= 10000 {
		return 1 + float64(amount)/100
	}

	early := 100.0
	extra := float64(amount-10000) / 500

	return 1 + early + extra
}
```

Ví dụ:

```txt
1,000 đạo ngân = x11
10,000 đạo ngân = x101
20,000 đạo ngân = x121
100,000 đạo ngân = x281
```

Vẫn rất mạnh, nhưng không phá game quá sớm.

---

# 12. Combat Tai Kiếp

Tai kiếp nên có 3 loại gameplay:

## Loại 1: Boss Raid

Boss xuất hiện trong Tiên Khiếu, player phải giết boss trước khi nó phá lõi Tiên Khiếu.

Ví dụ:

```txt
Huyết Vũ Ma Kiếp
Boss: Blood Rain Calamity Beast
Path: Blood
Skill: gây bleed lên toàn bộ resource node
Reward: Blood Dao Marks
```

## Loại 2: Tower Defense

Nhiều quái spawn ở rìa map, đi tới phá resource node.

Resource node có thể đặt trận pháp phòng thủ:

```txt
Kiếm Trận
Hỏa Trận
Băng Phong Trận
Lôi Kích Trận
Nô Đạo Thú Đàn
```

## Loại 3: Disaster Event

Không có quái chính, nhưng có thiên tai:

```txt
Mưa lửa
Băng phong
Độc vụ
Hắc phong
Huyết triều
Sấm trời
Không gian liệt phùng
Thời gian hỗn loạn
```

Cách xử lý:

```txt
Player dùng Cổ/Sát Chiêu để triệt tiêu disaster
Hoặc hy sinh resource node để hấp thụ sát thương
```

---

# 13. API Design

## Lấy Tiên Khiếu

```txt
GET /api/immortal-aperture
```

Response:

```json
{
  "id": "uuid",
  "grade": "HIGH",
  "time_rate": 30,
  "aperture_hp": 30000,
  "max_aperture_hp": 30000,
  "area_size": 9000000,
  "immortal_essence": 12,
  "internal_years_passed": 8.5,
  "next_tribulation_year": 10,
  "state": "STABLE"
}
```

## Lấy Đạo Ngân

```txt
GET /api/dao-marks
```

## Lấy map Tiên Khiếu

```txt
GET /api/immortal-aperture/nodes
```

## Đặt resource node

```txt
POST /api/immortal-aperture/nodes
```

Body:

```json
{
  "node_type": "MOON_ORCHID_FIELD",
  "x": 5,
  "y": 8
}
```

## Collect tài nguyên

```txt
POST /api/immortal-aperture/nodes/:id/collect
```

## Lấy tai kiếp hiện tại

```txt
GET /api/tribulations/active
```

## Đánh quái tai kiếp

```txt
POST /api/tribulations/:eventId/attack
```

Body:

```json
{
  "enemy_id": "uuid",
  "gu_id": "GU_BLOOD_MOON"
}
```

## Kết toán tai kiếp

```txt
POST /api/tribulations/:eventId/resolve
```

---

# 14. WebSocket Event

Khi tai kiếp tới, backend gửi event:

```json
{
  "type": "TRIBULATION_STARTED",
  "event_id": "uuid",
  "tribulation_type": "HEAVENLY_TRIBULATION",
  "path": "fire",
  "difficulty": 12,
  "message": "Thiên Kiếp đã giáng xuống Tiên Khiếu của ngươi!"
}
```

Topic:

```txt
/topic/player.{playerId}.tribulation
```

Frontend nhận event → bật modal đỏ → ép player vào map Tiên Khiếu.

---

# 15. Frontend UI

Trang Tiên Khiếu nên có 5 tab:

```txt
Overview
Resource Map
Dao Marks
Tribulation
Formation Defense
```

## Overview

Hiển thị:

```txt
Grade: Thượng Đẳng Phúc Địa
Time Rate: x30
Aperture HP: 27000 / 30000
Immortal Essence: 12
Internal Years: 8.5 / 10
Next Calamity: 1.5 năm nội bộ
State: Stable
```

## Resource Map

Grid 20x20 hoặc 30x30.

Mỗi ô có thể chứa:

```txt
Resource Node
Formation
Beast Group
Terrain
Enemy
```

## Dao Marks

Hiển thị dạng bảng:

```txt
Blood Dao Marks: 1,250 → Blood skills x13.5
Sword Dao Marks: 800 → Sword skills x9
Fire Dao Marks: 300 → Fire skills x4
```

## Tribulation

Nếu chưa tới:

```txt
Địa Tai tiếp theo còn: 2 giờ 15 phút
```

Nếu đang active:

```txt
Huyết Vũ Địa Tai đang phá hủy Tiên Khiếu
Boss HP: 78%
Aperture HP: 65%
Nodes destroyed: 2
```

---

# 16. Game Balance quan trọng

Không nên để player offline là chết luôn, vì dễ gây ức chế.

Nên dùng cơ chế:

```txt
Tai kiếp tới → state = TRIBULATION_PENDING
Player có 24h real-time để vào xử lý
Sau 24h không xử lý → auto resolve với tỉ lệ thất bại cao
```

Ví dụ:

```txt
Online xử lý: chủ động combat, ít mất tài nguyên
Offline auto resolve: mất 30-70% resource node, có thể trọng thương
```

Permadeath chỉ nên dùng cho mode hardcore.

Normal mode nên phạt:

```txt
Mất 20-50% tài nguyên
Giảm HP Tiên Khiếu
Mất Tiên Nguyên
Random mất Resource Node
Có xác suất tụt sub-rank
```

---

# 17. Roadmap triển khai hợp lý

## Phase 1: Tiên Khiếu cơ bản

Làm trước:

```txt
immortal_apertures table
dao_marks table
API get aperture
API get dao marks
Tạo Tiên Khiếu khi rank 6
Damage multiplier theo đạo ngân
```

## Phase 2: Resource Node

Thêm:

```txt
aperture_resource_nodes
Grid map
Đặt node
Collect tài nguyên
Node bị phá hủy
```

## Phase 3: Tai Kiếp

Thêm:

```txt
tribulation_events
tribulation_enemies
Worker check tai kiếp
Spawn event
Combat boss
Resolve success/fail
Reward đạo ngân
```

## Phase 4: Tower Defense

Thêm:

```txt
Formation
Enemy movement
Node targeting
Wave system
Boss skill
WebSocket realtime
```

---

# 18. Kết luận thiết kế

Cơ chế nên chia thành 3 tầng:

```txt
Tiên Khiếu = căn cứ / thế giới riêng
Tai Kiếp = boss raid định kỳ
Đạo Ngân = progression vĩnh viễn
```

Đây là phần endgame rất hợp với game Cổ Chân Nhân vì nó tạo được 3 cảm giác:

```txt
Sợ tai kiếp
Tham tài nguyên
Phê khi stack đạo ngân
```

Quan trọng nhất khi code: **đừng làm ngay full tower defense realtime**. Làm bản turn-based/simple combat trước, sau đó mới nâng cấp thành realtime grid.
