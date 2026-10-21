package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/worker"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// Primeval veins ("Nguyên Tuyền") are world-map resource points that produce
// primeval stones (or immortal stones) into a shared pool. Anyone standing on
// the tile may harvest the pool; harvesting resets it. Veins expire after a
// month and disappear early once their lifetime output cap is reached.
const (
	veinLifetime        = 30 * 24 * time.Hour
	veinDailyCapMinutes = 24 * 60
	veinMortalCap       = 31
	veinImmortalCap     = 5
	veinMinImmortalRank = 5
)

type veinRarityRate struct {
	Weight       int
	StonesPerMin float64
	QiPerMin     float64
	ChestChance  float64
	CapDays      int
}

// Same production curve as the old mines; total lifetime output scales with
// rarity (Q19): 7/10/14/21/30 days of production.
var veinRarityRates = map[string]veinRarityRate{
	"Thường":        {Weight: 50, StonesPerMin: 15, QiPerMin: 3.0, ChestChance: 0.08, CapDays: 7},
	"Hiếm":          {Weight: 30, StonesPerMin: 22, QiPerMin: 5.5, ChestChance: 0.12, CapDays: 10},
	"Sử Thi":        {Weight: 14, StonesPerMin: 25, QiPerMin: 6.25, ChestChance: 0.15, CapDays: 14},
	"Truyền Thuyết": {Weight: 5, StonesPerMin: 30, QiPerMin: 7.5, ChestChance: 0.2, CapDays: 21},
	"Siêu Cấp":      {Weight: 1, StonesPerMin: 60, QiPerMin: 15, ChestChance: 0.5, CapDays: 30},
}

// VeinInfo is the per-cell payload the world map renders and the detail panel
// reads. Accumulated values are computed lazily from accumulated_since.
type VeinInfo struct {
	ID                string    `json:"id"`
	VeinType          string    `json:"vein_type"`
	Rarity            string    `json:"rarity"`
	AccumulatedStones int64     `json:"accumulated_stones"`
	AccumulatedQi     float64   `json:"accumulated_qi"`
	AccumulatedChests int       `json:"accumulated_chests"`
	CapacityStones    int64     `json:"capacity_stones"`
	ExpiresAt         time.Time `json:"expires_at"`
}

type veinRow struct {
	ID               string
	RegionID         string
	X                int
	Y                int
	VeinType         string
	Rarity           string
	AccumulatedSince time.Time
	ProducedMinutes  int
	ExpiresAt        time.Time
}

func veinRate(rarity string) veinRarityRate {
	if rate, ok := veinRarityRates[rarity]; ok {
		return rate
	}
	return veinRarityRates["Thường"]
}

func veinCapMinutes(rarity string) int {
	return veinRate(rarity).CapDays * veinDailyCapMinutes
}

// veinAccrual computes the unharvested pool. It never exceeds one day of
// production (daily cap) nor the vein's remaining lifetime output.
func veinAccrual(v veinRow, now time.Time) (minutes int, stones int64, qi float64, chests int) {
	elapsed := int(now.Sub(v.AccumulatedSince).Minutes())
	if elapsed < 0 {
		elapsed = 0
	}
	remaining := veinCapMinutes(v.Rarity) - v.ProducedMinutes
	if remaining < 0 {
		remaining = 0
	}
	minutes = elapsed
	if minutes > veinDailyCapMinutes {
		minutes = veinDailyCapMinutes
	}
	if minutes > remaining {
		minutes = remaining
	}
	if minutes <= 0 {
		return 0, 0, 0, 0
	}

	rate := veinRate(v.Rarity)
	stones = int64(float64(minutes) * rate.StonesPerMin)
	qi = float64(minutes) * rate.QiPerMin
	chests = veinChestRoll(v.ID, v.AccumulatedSince, minutes, rate.ChestChance, v.VeinType == "immortal")
	return minutes, stones, qi, chests
}

// veinChestRoll is deterministic for a given (vein, accumulation window) so
// every map read shows the same chest count until somebody harvests.
func veinChestRoll(id string, since time.Time, minutes int, chance float64, immortal bool) int {
	var seed int64
	for _, b := range []byte(id) {
		seed = seed*31 + int64(b)
	}
	r := rand.New(rand.NewSource(seed + since.Unix()))
	chests := 0
	for i := 0; i < minutes; i++ {
		c := chance
		if immortal {
			c *= veinChestDecay(i)
		}
		if r.Float64() < c {
			chests++
		}
	}
	return chests
}

// veinChestDecay mirrors the old anti-AFK immortal-mine decay: -50% every 2h,
// floor 20% of the base chance.
func veinChestDecay(minute int) float64 {
	decay := math.Pow(0.5, float64(minute)/120.0)
	if decay < 0.2 {
		return 0.2
	}
	return decay
}

func veinChestItemID(veinType string) string {
	if veinType == "immortal" {
		return "mine_chest_immortal"
	}
	return "mine_chest"
}

func veinInfo(v veinRow, now time.Time) VeinInfo {
	_, stones, qi, chests := veinAccrual(v, now)
	rate := veinRate(v.Rarity)
	return VeinInfo{
		ID:                v.ID,
		VeinType:          v.VeinType,
		Rarity:            v.Rarity,
		AccumulatedStones: stones,
		AccumulatedQi:     qi,
		AccumulatedChests: chests,
		CapacityStones:    int64(float64(veinDailyCapMinutes) * rate.StonesPerMin),
		ExpiresAt:         v.ExpiresAt,
	}
}

func scanVeinRow(scan func(dest ...any) error) (veinRow, error) {
	var v veinRow
	err := scan(&v.ID, &v.RegionID, &v.X, &v.Y, &v.VeinType, &v.Rarity, &v.AccumulatedSince, &v.ProducedMinutes, &v.ExpiresAt)
	return v, err
}

// veinCellsInBounds returns map cells for every live vein inside the rectangle.
// The world map polls /explore/cells/rect, so veins need no registry reload.
func veinCellsInBounds(minX, minY, maxX, maxY int) []NearbyCell {
	query := `
		SELECT v.id::text, v.region_id, v.x, v.y, v.vein_type, v.rarity, v.accumulated_since, v.produced_minutes, v.expires_at
		FROM primeval_veins v
		JOIN region_world_maps rwm ON rwm.region_id = v.region_id
		WHERE rwm.world_map_id = 'world_map'
		  AND v.expires_at > NOW()
		  AND v.x BETWEEN $1 AND $2
		  AND v.y BETWEEN $3 AND $4`
	rows, err := db.DB.Query(query, minX, maxX, minY, maxY)
	if err != nil {
		logger.S.Warnf("Failed to load primeval veins: %v", err)
		return nil
	}
	defer rows.Close()

	now := time.Now()
	out := make([]NearbyCell, 0)
	for rows.Next() {
		v, errScan := scanVeinRow(rows.Scan)
		if errScan != nil {
			continue
		}
		info := veinInfo(v, now)
		out = append(out, NearbyCell{
			X:       v.X,
			Y:       v.Y,
			Type:    "PrimevalVein",
			Content: v.ID,
			Rank:    1,
			Region:  v.RegionID,
			Vein:    &info,
		})
	}
	return out
}

// HarvestVein collects the shared pool. The player must stand on the vein tile.
func HarvestVein(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	veinID := strings.TrimSpace(c.Param("id"))
	if _, err := uuid.Parse(veinID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nguyên Tuyền không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}

	var v veinRow
	err = tx.QueryRow(`
		SELECT id::text, region_id, x, y, vein_type, rarity, accumulated_since, produced_minutes, expires_at
		FROM primeval_veins WHERE id = $1 FOR UPDATE NOWAIT
	`, veinID).Scan(&v.ID, &v.RegionID, &v.X, &v.Y, &v.VeinType, &v.Rarity, &v.AccumulatedSince, &v.ProducedMinutes, &v.ExpiresAt)
	if err != nil {
		if db.IsLockNotAvailable(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Nguyên Tuyền đang có người thu hoạch, thử lại sau giây lát!", "code": "RESOURCE_BUSY"})
			return
		}
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Nguyên Tuyền không còn tồn tại"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load vein"})
		}
		return
	}

	now := time.Now()
	if !v.ExpiresAt.After(now) {
		_, _ = tx.Exec(`DELETE FROM primeval_veins WHERE id = $1`, v.ID)
		_ = tx.Commit()
		c.JSON(http.StatusNotFound, gin.H{"error": "Nguyên Tuyền đã cạn và biến mất"})
		return
	}
	standZone := ""
	if zone := FindZoneAt(player.X, player.Y); zone != nil {
		standZone = zone.ID
	}
	if player.X != v.X || player.Y != v.Y || (player.CurrentRegion != v.RegionID && standZone != v.RegionID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu phải đứng ngay tại Nguyên Tuyền mới thu hoạch được!"})
		return
	}

	minutes, stones, qi, chests := veinAccrual(v, now)
	if minutes <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nguyên Tuyền chưa tích tụ đủ nguyên khí, hãy chờ thêm!"})
		return
	}

	var stonesAdd, immortalAdd float64
	if v.VeinType == "immortal" {
		immortalAdd = float64(stones)
	} else {
		stonesAdd = float64(stones)
	}
	if _, err = tx.Exec(fmt.Sprintf(`
		UPDATE players
		SET primeval_stones = primeval_stones + $1,
		    immortal_stones = immortal_stones + $2,
		    human_qi = LEAST(human_qi + $3, %d)
		WHERE id = $4
	`, int(MaxHumanQi)), stonesAdd, immortalAdd, qi, authID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add vein rewards"})
		return
	}
	if chests > 0 {
		if err = InventorySvc.AddItem(tx, authID.String(), veinChestItemID(v.VeinType), "materials", int64(chests)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add vein chests"})
			return
		}
	}

	produced := v.ProducedMinutes + minutes
	depleted := produced >= veinCapMinutes(v.Rarity)
	if depleted {
		_, err = tx.Exec(`DELETE FROM primeval_veins WHERE id = $1`, v.ID)
	} else {
		_, err = tx.Exec(`UPDATE primeval_veins SET accumulated_since = $1, produced_minutes = $2 WHERE id = $3`, now, produced, v.ID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update vein"})
		return
	}

	auditlog.Log(tx, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "vein.harvest",
		EntityType:     "primeval_vein",
		EntityID:       v.ID,
		RequestSummary: fmt.Sprintf("Thu hoạch Nguyên Tuyền %s tại %s (%d,%d): %d thạch, %.1f Nhân Khí, %d rương", v.Rarity, v.RegionID, v.X, v.Y, stones, qi, chests),
		Metadata: map[string]interface{}{
			"vein_id":   v.ID,
			"region_id": v.RegionID,
			"rarity":    v.Rarity,
			"vein_type": v.VeinType,
			"minutes":   minutes,
			"stones":    stones,
			"qi":        qi,
			"chests":    chests,
			"depleted":  depleted,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	msg := fmt.Sprintf("Thu hoạch Nguyên Tuyền %s: +%d %s, +%.0f Nhân Khí", v.Rarity, stones, map[bool]string{true: "Tiên Nguyên Thạch", false: "Nguyên Thạch"}[v.VeinType == "immortal"], qi)
	if chests > 0 {
		msg += fmt.Sprintf(", +%d rương", chests)
	}
	if depleted {
		msg += ". Nguyên Tuyền đã cạn!"
	}
	respondActionWithInventory(c, http.StatusOK, player, gin.H{"message": msg})
}

// rollVeinRarity returns a weighted rarity. Eggs never produce immortal veins;
// the caller chooses the vein_type separately.
func rollVeinRarity() string {
	total := 0
	for _, rate := range veinRarityRates {
		total += rate.Weight
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	n := r.Intn(total)
	for _, rarity := range []string{"Thường", "Hiếm", "Sử Thi", "Truyền Thuyết", "Siêu Cấp"} {
		n -= veinRarityRates[rarity].Weight
		if n < 0 {
			return rarity
		}
	}
	return "Thường"
}

type veinQueryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

func veinTypeCount(q veinQueryer, veinType string) (int, error) {
	var count int
	err := q.QueryRow(`SELECT COUNT(*) FROM primeval_veins WHERE vein_type = $1 AND expires_at > NOW()`, veinType).Scan(&count)
	return count, err
}

// veinGreatRegionCap keeps roughly 1/5 of the global cap in each đại vực.
func veinGreatRegionCap(veinType string) int {
	if veinType == "immortal" {
		return (veinImmortalCap + 4) / 5
	}
	return (veinMortalCap + 4) / 5
}

func veinGreatRegionCounts(q veinQueryer, veinType string) (map[string]int, error) {
	rows, err := q.Query(`SELECT region_id, COUNT(*) FROM primeval_veins WHERE vein_type = $1 AND expires_at > NOW() GROUP BY region_id`, veinType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := make(map[string]int)
	for rows.Next() {
		var regionID string
		var count int
		if errScan := rows.Scan(&regionID, &count); errScan == nil {
			counts[greatRegionID(regionID)] += count
		}
	}
	return counts, rows.Err()
}

// spawnVeinTx inserts a vein at an explicit tile. Used by SPRING_EGG_HATCH.
func spawnVeinTx(tx *sql.Tx, regionID string, x, y int, veinType, rarity string) error {
	var zoneType string
	var minRank int
	if err := tx.QueryRow(`SELECT UPPER(zone_type), min_rank FROM regions WHERE id = $1 AND is_active`, regionID).Scan(&zoneType, &minRank); err != nil {
		return fmt.Errorf("vùng không tồn tại")
	}
	switch zoneType {
	case ZoneTypeHub, ZoneTypeSafe, ZoneTypeSecret, ZoneTypeDungeon:
		return fmt.Errorf("không thể cấy Nguyên Tuyền trong vùng an toàn hoặc bí cảnh")
	}
	if veinType == "immortal" && minRank < veinMinImmortalRank {
		return fmt.Errorf("vùng này không đủ linh khí cho Tiên Tuyền")
	}

	var occupied bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM map_cells WHERE region_id = $1 AND x = $2 AND y = $3 AND is_active)`, regionID, x, y).Scan(&occupied); err != nil {
		return err
	}
	if occupied {
		return fmt.Errorf("ô này đã có địa điểm khác")
	}

	res, err := tx.Exec(`
		INSERT INTO primeval_veins (region_id, x, y, vein_type, rarity, accumulated_since, expires_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), $6)
		ON CONFLICT (region_id, x, y) DO NOTHING
	`, regionID, x, y, veinType, rarity, time.Now().Add(veinLifetime))
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("ô này đã có Nguyên Tuyền")
	}
	return nil
}

// spawnRandomVein places one vein in a random eligible world-map region.
func spawnRandomVein(ctx context.Context, veinType string) error {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT r.id, r.min_x, r.min_y, r.max_x, r.max_y
		FROM regions r
		JOIN region_world_maps rwm ON rwm.region_id = r.id
		WHERE r.is_active
		  AND rwm.world_map_id = 'world_map'
		  AND UPPER(r.zone_type) IN ('NGU_VUC','KHU_VUC','PVP_ZONE','BOSS_ZONE')
	`)
	if err != nil {
		return err
	}
	type regionBounds struct {
		ID                     string
		MinX, MinY, MaxX, MaxY int
	}
	regions := make([]regionBounds, 0)
	for rows.Next() {
		var r regionBounds
		if errScan := rows.Scan(&r.ID, &r.MinX, &r.MinY, &r.MaxX, &r.MaxY); errScan == nil && r.MaxX > r.MinX && r.MaxY > r.MinY {
			regions = append(regions, r)
		}
	}
	rows.Close()
	if len(regions) == 0 {
		return fmt.Errorf("no eligible region")
	}

	counts, errCounts := veinGreatRegionCounts(db.DB, veinType)
	if errCounts != nil {
		return errCounts
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for attempt := 0; attempt < 60; attempt++ {
		r := regions[rng.Intn(len(regions))]
		x := r.MinX + rng.Intn(r.MaxX-r.MinX+1)
		y := r.MinY + rng.Intn(r.MaxY-r.MinY+1)

		zone := FindZoneAt(x, y)
		if zone == nil || worldMapIDForRegion(zone.ID) != "world_map" {
			continue
		}
		switch strings.ToUpper(zone.ZoneType) {
		case ZoneTypeHub, ZoneTypeSafe, ZoneTypeSecret:
			continue
		}
		if veinType == "immortal" && zone.MinRank < veinMinImmortalRank {
			continue
		}
		great := greatRegionID(zone.ID)
		if counts[great] >= veinGreatRegionCap(veinType) {
			continue
		}

		var occupied bool
		if errScan := db.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM map_cells WHERE region_id = $1 AND x = $2 AND y = $3 AND is_active)`, zone.ID, x, y).Scan(&occupied); errScan != nil || occupied {
			continue
		}
		res, errExec := db.DB.ExecContext(ctx, `
			INSERT INTO primeval_veins (region_id, x, y, vein_type, rarity, accumulated_since, expires_at)
			VALUES ($1, $2, $3, $4, $5, NOW(), $6)
			ON CONFLICT (region_id, x, y) DO NOTHING
		`, zone.ID, x, y, veinType, rollVeinRarity(), time.Now().Add(veinLifetime))
		if errExec != nil {
			continue
		}
		if n, _ := res.RowsAffected(); n > 0 {
			counts[great]++
			return nil
		}
	}
	return fmt.Errorf("no free tile found")
}

// ensureVeinPopulation removes dead veins and tops the world up to its caps.
func ensureVeinPopulation(ctx context.Context, fillAll bool) {
	if _, err := db.DB.ExecContext(ctx, `
		DELETE FROM primeval_veins WHERE expires_at <= NOW()
	`); err != nil {
		logger.S.Warnf("Primeval vein cleanup failed: %v", err)
		return
	}

	for _, spec := range []struct {
		Type string
		Cap  int
	}{
		{"mortal", veinMortalCap},
		{"immortal", veinImmortalCap},
	} {
		count, err := veinTypeCount(db.DB, spec.Type)
		if err != nil {
			logger.S.Warnf("Primeval vein count failed: %v", err)
			continue
		}
		toSpawn := spec.Cap - count
		if toSpawn <= 0 {
			continue
		}
		if !fillAll && toSpawn > 1 {
			toSpawn = 1
		}
		for i := 0; i < toSpawn; i++ {
			if errSpawn := spawnRandomVein(ctx, spec.Type); errSpawn != nil {
				logger.S.Warnf("Primeval vein spawn (%s) failed: %v", spec.Type, errSpawn)
				break
			}
		}
	}
}

// StartPrimevalVeinWorker fills the map once at boot, then tops up one vein per
// type every 30 minutes.
func StartPrimevalVeinWorker(ctx context.Context) {
	worker.SafeRun("Primeval Vein initial fill", ctx, func(ctx context.Context) {
		ensureVeinPopulation(ctx, true)
	})
	worker.RunEvery(ctx, 30*time.Minute, "Primeval Vein Worker", func(ctx context.Context) {
		ensureVeinPopulation(ctx, false)
	})
}

// springEggHatchEffect implements SPRING_EGG_HATCH: plant a mortal vein on the
// player's current tile. Returns an error (and therefore does NOT consume the
// egg) when the tile is taken, the region is safe, or the global cap is full.
func springEggHatchEffect(tx *sql.Tx, player *models.Player, _ models.ItemSpec, _ float64) (string, error) {
	count, err := veinTypeCount(tx, "mortal")
	if err != nil {
		return "", err
	}
	if count >= veinMortalCap {
		return "", fmt.Errorf("thiên hạ đã đủ %d Nguyên Tuyền, không thể cấy thêm", veinMortalCap)
	}
	var instanceID string
	if err := tx.QueryRow(`SELECT COALESCE(map_instance_id::text, '') FROM players WHERE id = $1`, player.ID).Scan(&instanceID); err != nil {
		return "", err
	}
	if instanceID != "" {
		return "", fmt.Errorf("không thể cấy Nguyên Tuyền trong bí cảnh")
	}
	regionID := player.CurrentRegion
	if regionID == "" {
		zone := FindZoneAt(player.X, player.Y)
		if zone == nil {
			return "", fmt.Errorf("không xác định được vùng hiện tại")
		}
		regionID = zone.ID
	}
	if zone := FindZoneAt(player.X, player.Y); zone != nil {
		regionID = zone.ID
	}
	greatCounts, err := veinGreatRegionCounts(tx, "mortal")
	if err != nil {
		return "", err
	}
	if greatCounts[greatRegionID(regionID)] >= veinGreatRegionCap("mortal") {
		return "", fmt.Errorf("đại vực này đã đủ Nguyên Tuyền, không thể cấy thêm")
	}
	rarity := rollVeinRarity()
	if err := spawnVeinTx(tx, regionID, player.X, player.Y, "mortal", rarity); err != nil {
		return "", err
	}
	return fmt.Sprintf("Trứng nứt! Một Nguyên Tuyền phẩm chất %s xuất hiện ngay dưới chân đạo hữu.", rarity), nil
}
