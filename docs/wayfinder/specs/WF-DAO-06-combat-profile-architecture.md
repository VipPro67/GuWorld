# [SPECIFICATION] Kiến Trúc Cache Dao Combat Profile O(1), Data Models & API Contract

**Ticket:** [WF-DAO-06: Kiến trúc Cache Dao Combat Profile O(1), Data Models & API Contract](../tickets/WF-DAO-06-architecture-combat-profile-cache.md)  
**Tác giả:** Antigravity  
**Ngày hoàn thành:** 2026-09-20  

---

## 1. Cấu Trúc Dữ Liệu Dao Combat Profile (`gamefx/dao_combat_profile.go`)

Để giải quyết bài toán hiệu năng khi mở rộng lên 50 lưu phái trong hàng chục ngàn trận đấu tự động của Simulator, toàn bộ hệ số xung đột và phòng thủ được tiền tính toán một lần thành `DaoCombatProfile`:

```go
package gamefx

// PathCombatModifiers chứa toàn bộ hệ số chiến đấu đã tính sẵn cho 1 lưu phái.
type PathCombatModifiers struct {
    Path                 string  `json:"path"`
    DaoMarks             int64   `json:"dao_marks"`
    DaoAmplification     float64 `json:"dao_amplification"`      // DaoAmp(M) theo công thức 2 đoạn
    AttainmentLevel      int     `json:"attainment_level"`       // 0-4 (Phàm Thai -> Vô Thượng)
    RealmControl         float64 `json:"realm_control"`          // 0.0 -> 0.80 (Tỷ lệ triệt tiêu xung đột)
    RealmEfficiency      float64 `json:"realm_efficiency"`       // 1.00 -> 1.45 (Hiệu suất khai thác Đạo ngân)
    InternalConflictMult float64 `json:"internal_conflict_mult"` // 0.60 -> 1.00 (Hệ số xung đột khi ra chiêu)
    NaturalResistance    float64 `json:"natural_resistance"`     // 0.00 -> 0.30 (Kháng tự nhiên khi bị phái này đánh)
    Purity               float64 `json:"purity"`                 // Đạo ngân phái / Tổng đạo ngân
}

// DaoCombatProfile là profile chiến đấu đạo ngân của Combatant.
type DaoCombatProfile struct {
    TotalDaoMarks        int64                          `json:"total_dao_marks"`
    GraceThreshold       int64                          `json:"grace_threshold"`
    HasNonConflictingDao bool                           `json:"has_non_conflicting_dao"`
    DominantPath         string                         `json:"dominant_path"`
    Paths                map[string]PathCombatModifiers `json:"paths"`
}
```

---

## 2. Thuật Toán Xây Dựng Profile (`gamefx.BuildDaoCombatProfile`)

```go
func BuildDaoCombatProfile(
    daoMarks map[string]int64,
    pathMasteries map[string]int,
    hasNonConflictingDao bool,
    daoBonus map[string]float64,
) *DaoCombatProfile {
    // 1. Tính tổng Đạo ngân & Grace Threshold
    var totalMarks int64
    for _, amt := range daoMarks {
        if amt > 0 {
            totalMarks += amt
        }
    }
    graceThreshold := int64(1000)
    if pct := int64(float64(totalMarks) * 0.02); pct > graceThreshold {
        graceThreshold = pct
    }

    // 2. Lọc các phái có đạo ngân vượt ngưỡng an toàn
    activeForeign := make(map[string]int64)
    for p, amt := range daoMarks {
        if amt > graceThreshold {
            activeForeign[p] = amt
        }
    }

    profile := &DaoCombatProfile{
        TotalDaoMarks:        totalMarks,
        GraceThreshold:       graceThreshold,
        HasNonConflictingDao: hasNonConflictingDao,
        Paths:                make(map[string]PathCombatModifiers),
    }

    // 3. Tiền tính toán cho toàn bộ 49 lưu phái chuẩn
    for _, p := range CanonicalPaths {
        amt := daoMarks[p]
        points := pathMasteries[p]
        lvl := PathMasteryLevel(points)
        realmCtrl := RealmControlForLevel(lvl)
        realmEff := RealmEfficiencyForLevel(lvl)
        daoAmp := DaoAmplification(amt) + daoBonus[p]

        // Tính Raw Conflict khi công kích phái p
        var rawConflict float64
        for otherP, otherAmt := range activeForeign {
            if otherP == p {
                continue
            }
            comp := GetCompatibility(p, otherP)
            rawConflict += float64(otherAmt) * (1.0 - comp)
        }
        effConflict := rawConflict * (1.0 - realmCtrl)
        conflictRatio := effConflict / (float64(amt) + 50000.0)
        conflictMult := 1.0 / (1.0 + conflictRatio*0.8*2.0)
        if conflictMult < 0.60 {
            conflictMult = 0.60
        }

        // Tính Natural Resistance khi bị phái p tấn công
        var defConflict float64
        for defP, defAmt := range daoMarks {
            if defAmt <= 0 {
                continue
            }
            comp := GetCompatibility(p, defP)
            defConflict += float64(defAmt) * (1.0 - comp)
        }
        natResist := 0.0
        if !hasNonConflictingDao && defConflict > 0 {
            natResist = 0.30 * (defConflict / (defConflict + 50000.0))
        }

        purity := 0.0
        if totalMarks > 0 {
            purity = float64(amt) / float64(totalMarks)
        }

        profile.Paths[p] = PathCombatModifiers{
            Path:                 p,
            DaoMarks:             amt,
            DaoAmplification:     daoAmp,
            AttainmentLevel:      lvl,
            RealmControl:         realmCtrl,
            RealmEfficiency:      realmEff,
            InternalConflictMult: conflictMult,
            NaturalResistance:    natResist,
            Purity:               purity,
        }
    }

    return profile
}
```

---

## 3. Tích Hợp Vào Combat Engine (`simulator/engine.go`)

### 3.1. Truy Xuất Công Kích Trong Vòng Lặp Combat ($O(1)$):
```go
func (p *DaoCombatProfile) GetOffensiveFactors(path string) (daoAmp float64, realmEff float64, conflictMult float64) {
    if p == nil {
        return 1.0, 1.0, 1.0
    }
    clean := strings.TrimSuffix(strings.ToLower(path), " path")
    mod, ok := p.Paths[clean]
    if !ok {
        return 1.0, 1.0, 1.0
    }
    if p.HasNonConflictingDao {
        return mod.DaoAmplification, mod.RealmEfficiency, 1.00
    }
    return mod.DaoAmplification, mod.RealmEfficiency, mod.InternalConflictMult
}
```

### 3.2. Truy Xuất Phòng Thủ Trong Vòng Lặp Combat ($O(1)$):
```go
func (p *DaoCombatProfile) GetDefensiveResistance(attackPath string) float64 {
    if p == nil || p.HasNonConflictingDao {
        return 0.0
    }
    clean := strings.TrimSuffix(strings.ToLower(attackPath), " path")
    if mod, ok := p.Paths[clean]; ok {
        return mod.NaturalResistance
    }
    return 0.0
}
```

---

## 4. API Response Contract Cho UI (`GET /api/player/daongan`)

Mở rộng DTO trả về để Frontend hiển thị trực quan các chỉ số cân bằng:

```json
{
  "total_dao_marks": 70000,
  "total_dao_ngan": 18270000.0,
  "grace_threshold": 1400,
  "dominant_path": "fire",
  "has_non_conflicting_dao": false,
  "paths": [
    {
      "path": "fire",
      "amount": 50000,
      "multiplier": 261.0,
      "power": 13050000.0,
      "attainment_level": 3,
      "attainment_name": "Đại Tông Sư",
      "realm_control_pct": 45.0,
      "realm_efficiency": 1.25,
      "internal_conflict_penalty_pct": 16.5,
      "natural_resistance_pct": 3.2,
      "purity_pct": 71.4
    },
    {
      "path": "water",
      "amount": 20000,
      "multiplier": 141.0,
      "power": 2820000.0,
      "attainment_level": 1,
      "attainment_name": "Đại Sư",
      "realm_control_pct": 10.0,
      "realm_efficiency": 1.05,
      "internal_conflict_penalty_pct": 28.4,
      "natural_resistance_pct": 14.8,
      "purity_pct": 28.6
    }
  ]
}
```
