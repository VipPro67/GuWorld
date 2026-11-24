package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/worker"
)

type travelJourney struct {
	JourneyID    uuid.UUID  `json:"journey_id"`
	Revision     int64      `json:"revision"`
	Status       string     `json:"status"`
	TargetRegion string     `json:"target_region"`
	TargetX      int        `json:"target_x"`
	TargetY      int        `json:"target_y"`
	NextStepAt   *time.Time `json:"next_step_at,omitempty"`
	LastStepAt   *time.Time `json:"last_step_at,omitempty"`
	LastEvent    string     `json:"last_event"`
	ReasonCode   string     `json:"reason_code"`
}

func travelPlayerID(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get("player_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Chưa xác thực"})
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Mã người chơi không hợp lệ"})
		return uuid.Nil, false
	}
	return id, true
}

func loadTravelJourney(q interface{ QueryRow(string, ...any) *sql.Row }, playerID uuid.UUID) (*travelJourney, error) {
	j := &travelJourney{}
	var next, last sql.NullTime
	err := q.QueryRow(`SELECT journey_id,revision,status,target_region,target_x,target_y,next_step_at,last_step_at,last_event,reason_code FROM player_travel_journeys WHERE player_id=$1`, playerID).Scan(&j.JourneyID, &j.Revision, &j.Status, &j.TargetRegion, &j.TargetX, &j.TargetY, &next, &last, &j.LastEvent, &j.ReasonCode)
	if err != nil {
		return nil, err
	}
	if next.Valid {
		j.NextStepAt = &next.Time
	}
	if last.Valid {
		j.LastStepAt = &last.Time
	}
	return j, nil
}

func GetTravelJourney(c *gin.Context) {
	id, ok := travelPlayerID(c)
	if !ok {
		return
	}
	j, err := loadTravelJourney(db.DB, id)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, gin.H{"journey": nil})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải hành trình"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"journey": j})
}

type travelTargetRequest struct {
	Region string `json:"region_id"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
}

func PreviewTravelJourney(c *gin.Context) {
	id, ok := travelPlayerID(c)
	if !ok {
		return
	}
	var req travelTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đích hành trình không hợp lệ"})
		return
	}
	req.Region = strings.TrimSpace(req.Region)
	player, err := PlayerSvc.LoadEquipped(db.DB, id)
	if err != nil {
		c.JSON(500, gin.H{"error": "Không thể tải trạng thái người chơi"})
		return
	}
	if err := validateTravelLeader(db.DB, &player, id); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	var instance sql.NullString
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT map_instance_id::text FROM players WHERE id=$1`, id).Scan(&instance); err != nil {
		c.JSON(500, gin.H{"error": "Không thể kiểm tra bản đồ hiện tại"})
		return
	}
	if player.ActiveCombat != nil || instance.Valid {
		c.JSON(http.StatusConflict, gin.H{"error": "Không thể lập hành trình khi đang chiến đấu hoặc ở trong bản đồ instance"})
		return
	}
	if err := validateTravelTarget(player.CurrentRegion, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := checkRegionEntry(req.Region, player.Rank, player.GuInventory); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ điều kiện vào khu vực đích: " + err.Error()})
		return
	}
	if player.CurrentRegion == req.Region && player.X == req.X && player.Y == req.Y {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đích đang là vị trí hiện tại"})
		return
	}
	segments, regionIDs, ok := estimateTravelRoute(&player, player.CurrentRegion, req.Region, player.X, player.Y, req.X, req.Y)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có tuyến đi bộ giữa hai khu vực này."})
		return
	}
	movement := CalculatePlayerStats(&player)["movement"]
	var etaSeconds *int
	if movement > 0 {
		var eta time.Duration
		for _, length := range segments {
			eta += travelStepDuration(movement, length)
		}
		seconds := int(math.Ceil(eta.Seconds()))
		etaSeconds = &seconds
	}
	firstStep, _, routeOK := travelNextStep(player.CurrentRegion, req.Region, player.X, player.Y, req.X, req.Y, &player)
	if !routeOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm được bước đầu tiên tới đích."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"target_region": req.Region, "target_x": req.X, "target_y": req.Y, "region_path": regionIDs, "distance": math.Round(sumTravelDistance(segments)*100) / 100, "eta_seconds": etaSeconds, "movement": movement, "first_step": gin.H{"x": firstStep[0], "y": firstStep[1]}})
}

func estimateTravelRoute(player *models.Player, startRegion, targetRegion string, x, y, targetX, targetY int) ([]float64, []string, bool) {
	start, target := ZoneByID(startRegion), ZoneByID(targetRegion)
	regions := travelRegionRouteForPlayer(start, target, player)
	if len(regions) == 0 {
		return nil, nil, false
	}
	var lengths []float64
	regionIDs := make([]string, 0, len(regions))
	for _, region := range regions {
		regionIDs = append(regionIDs, region.ID)
	}
	for i := 0; i < len(regions); i++ {
		current := regions[i]
		if i == len(regions)-1 {
			lengths = append(lengths, travelPathLengths(x, y, targetX, targetY)...)
			break
		}
		next := regions[i+1]
		dirX, dirY := 0, 0
		if next.AtlasX > current.AtlasX {
			dirX = 1
		} else if next.AtlasX < current.AtlasX {
			dirX = -1
		}
		if next.AtlasY > current.AtlasY {
			dirY = 1
		} else if next.AtlasY < current.AtlasY {
			dirY = -1
		}
		goalX, goalY := x, y
		if dirX > 0 {
			goalX = current.MaxX
		} else if dirX < 0 {
			goalX = current.MinX
		}
		if dirY > 0 {
			goalY = current.MaxY
		} else if dirY < 0 {
			goalY = current.MinY
		}
		if dirX != 0 && dirY == 0 {
			goalY = clampTravelCoordinate(y, current.MinY, current.MaxY)
		}
		if dirY != 0 && dirX == 0 {
			goalX = clampTravelCoordinate(x, current.MinX, current.MaxX)
		}
		lengths = append(lengths, travelPathLengths(x, y, goalX, goalY)...)
		lengths = append(lengths, 1)
		x, y = travelEdgeEntry(next, dirX, dirY, goalX, goalY)
	}
	return lengths, regionIDs, true
}

func travelPathLengths(x, y, targetX, targetY int) []float64 {
	path := lineCells(x, y, targetX, targetY)
	lengths := make([]float64, 0, len(path))
	for _, cell := range path {
		length := 1.0
		if cell[0] != x && cell[1] != y {
			length = math.Sqrt2
		}
		lengths = append(lengths, length)
		x, y = cell[0], cell[1]
	}
	return lengths
}

func sumTravelDistance(lengths []float64) float64 {
	var total float64
	for _, length := range lengths {
		total += length
	}
	return total
}

func travelEdgeEntry(region *models.Region, dirX, dirY, x, y int) (int, int) {
	const margin = 4
	if dirX > 0 {
		x = margin
	} else if dirX < 0 {
		x = clampTravelCoordinate(region.MaxX-margin, region.MinX, region.MaxX)
	}
	if dirY > 0 {
		y = margin
	} else if dirY < 0 {
		y = clampTravelCoordinate(region.MaxY-margin, region.MinY, region.MaxY)
	}
	return x, y
}

func validateTravelTarget(playerRegion string, target travelTargetRequest) error {
	if target.Region == "" {
		return errTravelTarget
	}
	zone := ZoneByID(target.Region)
	current := ZoneByID(playerRegion)
	if zone == nil || current == nil || zone.ZoneType == ZoneTypeHub || zone.ZoneType == ZoneTypeSecret || zone.ZoneType == ZoneTypeAperture || zone.ZoneType == ZoneTypeDungeon || !zoneContains(zone, target.X, target.Y) {
		return errTravelTarget
	}
	if greatRegionID(target.Region) != greatRegionID(playerRegion) || zoneLayerID(zone) != zoneLayerID(current) {
		return errTravelTarget
	}
	return nil
}

var errTravelTarget = &movementError{status: http.StatusBadRequest, message: "Ô đích không thuộc khu vực đi bộ hợp lệ hiện tại."}

func StartTravelJourney(c *gin.Context) {
	id, ok := travelPlayerID(c)
	if !ok {
		return
	}
	var req travelTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đích hành trình không hợp lệ"})
		return
	}
	req.Region = strings.TrimSpace(req.Region)
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(500, gin.H{"error": "Không thể bắt đầu hành trình"})
		return
	}
	defer tx.Rollback()
	var region string
	var x, y int
	var combat sql.NullString
	var mapInstance sql.NullString
	if err = tx.QueryRowContext(c.Request.Context(), `SELECT current_region,x,y,active_combat::text,map_instance_id::text FROM players WHERE id=$1 FOR UPDATE`, id).Scan(&region, &x, &y, &combat, &mapInstance); err != nil {
		c.JSON(500, gin.H{"error": "Không thể tải vị trí"})
		return
	}
	if combat.Valid && combat.String != "null" {
		c.JSON(http.StatusConflict, gin.H{"error": "Không thể bắt đầu hành trình khi đang chiến đấu"})
		return
	}
	if mapInstance.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể bắt đầu hành trình trong bản đồ instance"})
		return
	}
	if err = validateTravelTarget(region, req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	player, err := PlayerSvc.LoadEquipped(tx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tính tốc độ hành trình"})
		return
	}
	if err := validateTravelLeader(tx, &player, id); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if err := checkRegionEntry(req.Region, player.Rank, player.GuInventory); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ điều kiện vào khu vực đích: " + err.Error()})
		return
	}
	if region == req.Region && x == req.X && y == req.Y {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đích đang là vị trí hiện tại"})
		return
	}
	now := time.Now().UTC()
	_, firstStepLength, routeOK := travelNextStep(region, req.Region, player.X, player.Y, req.X, req.Y, &player)
	if !routeOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm được bước đầu tiên tới đích."})
		return
	}
	movement := CalculatePlayerStats(&player)["movement"]
	status, reason, event := "running", "", "Hành trình đã bắt đầu"
	var nextStepAt any = now.Add(travelStepDuration(movement, firstStepLength))
	if movement <= 0 {
		status, reason, event, nextStepAt = "paused_condition", "movement_unavailable", "Hành trình tạm dừng vì không thể di chuyển.", nil
	}
	_, err = tx.ExecContext(c.Request.Context(), `INSERT INTO player_travel_journeys(player_id,journey_id,revision,status,target_region,target_x,target_y,next_step_at,last_step_at,last_event,reason_code,updated_at) VALUES($1,gen_random_uuid(),1,$2,$3,$4,$5,$6,NULL,$7,$8,$9) ON CONFLICT(player_id) DO UPDATE SET journey_id=gen_random_uuid(),revision=player_travel_journeys.revision+1,status=EXCLUDED.status,target_region=EXCLUDED.target_region,target_x=EXCLUDED.target_x,target_y=EXCLUDED.target_y,next_step_at=EXCLUDED.next_step_at,last_step_at=NULL,last_event=EXCLUDED.last_event,reason_code=EXCLUDED.reason_code,updated_at=EXCLUDED.updated_at`, id, status, req.Region, req.X, req.Y, nextStepAt, event, reason, now)
	if err != nil {
		c.JSON(500, gin.H{"error": "Không thể lưu hành trình"})
		return
	}
	if err = tx.Commit(); err != nil {
		c.JSON(500, gin.H{"error": "Không thể xác nhận hành trình"})
		return
	}
	PublishPlayerRealtimeEvent(id, PlayerEventUpdated, TravelJourneyDomain)
	j, err := loadTravelJourney(db.DB, id)
	if err != nil {
		c.JSON(500, gin.H{"error": "Không thể tải hành trình"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"journey": j})
}

func validateTravelLeader(q interface{ QueryRow(string, ...any) *sql.Row }, player *models.Player, playerID uuid.UUID) error {
	if player.PartyID == nil {
		return nil
	}
	var leaderID uuid.UUID
	if err := q.QueryRow(`SELECT leader_id FROM parties WHERE id=$1`, player.PartyID).Scan(&leaderID); err != nil {
		return fmt.Errorf("không thể xác nhận quyền trưởng nhóm")
	}
	if leaderID != playerID {
		return fmt.Errorf("chỉ đội trưởng mới có thể bắt đầu hành trình")
	}
	return nil
}

func travelStepDuration(movement, stepLength float64) time.Duration {
	if movement <= 0 || stepLength <= 0 {
		return 0
	}
	seconds := 10 * stepLength / movement
	if seconds < 1 {
		seconds = 1
	}
	if seconds > 60 {
		seconds = 60
	}
	return time.Duration(seconds * float64(time.Second))
}

func calculatePlayerStatsAt(player *models.Player, at time.Time) map[string]float64 {
	projected := *player
	projected.Statuses = make(models.StatusList, 0, len(player.Statuses))
	for _, status := range player.Statuses {
		if status.ExpiresAt != nil && !status.ExpiresAt.After(at) {
			continue
		}
		status.ExpiresAt = nil
		projected.Statuses = append(projected.Statuses, status)
	}
	return CalculatePlayerStats(&projected)
}

func persistTravelJourneyStep(tx *sql.Tx, player *models.Player, journeyID uuid.UUID, targetRegion string, targetX, targetY int, scheduledAt time.Time, fromRegion string, fromX, fromY int, manualCombat, pvp bool, combatWinner, event string) error {
	status, reason := "running", ""
	var nextStepAt any
	switch {
	case pvp:
		status, reason = "failed", "pvp_encounter"
		event = "PvP đã bắt đầu; hành trình kết thúc để bạn xử lý chiến đấu thủ công."
	case manualCombat:
		status, reason = "paused_manual_combat", "combat"
		event = "Đang chờ xử lý chiến đấu."
	case combatWinner != "" && combatWinner != "Phe Ta":
		status, reason = "failed", "combat_loss"
		event = "Nhân vật thất bại trong chiến đấu tự động; hành trình kết thúc."
	case player.CurrentRegion == fromRegion && player.X == fromX && player.Y == fromY:
		status, reason = "failed", "step_blocked"
		event = "Bước tiếp theo bị chặn; hành trình dừng tại vị trí hiện tại."
	case travelAtTarget(player.CurrentRegion, targetRegion, player.X, player.Y, targetX, targetY):
		status, reason = "completed", "arrived"
		event = "Đã tới đích."
	default:
		movement := calculatePlayerStatsAt(player, scheduledAt)["movement"]
		_, stepLength, routeOK := travelNextStep(player.CurrentRegion, targetRegion, player.X, player.Y, targetX, targetY, player)
		if !routeOK {
			status, reason = "failed", "no_route"
			event = "Không còn tuyến đi bộ hợp lệ tới đích."
			break
		}
		delay := travelStepDuration(movement, stepLength)
		if delay <= 0 {
			status, reason = "paused_condition", "movement_unavailable"
			event = "Hành trình tạm dừng vì không thể di chuyển."
		} else {
			due := scheduledAt.Add(delay)
			nextStepAt = due
			if event == "" {
				event = "Đã hoàn thành một bước."
			}
		}
	}
	res, err := tx.Exec(`UPDATE player_travel_journeys SET status=$3,revision=revision+1,next_step_at=$4,last_step_at=NOW(),reason_code=$5,last_event=$6,updated_at=NOW() WHERE player_id=$1 AND journey_id=$2 AND status='running'`, player.ID, journeyID, status, nextStepAt, reason, event)
	if err != nil {
		return err
	}
	if count, err := res.RowsAffected(); err != nil || count != 1 {
		return fmt.Errorf("travel journey changed during step commit")
	}
	return nil
}

func StopTravelJourney(c *gin.Context) {
	id, ok := travelPlayerID(c)
	if !ok {
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(500, gin.H{"error": "Không thể dừng hành trình"})
		return
	}
	defer tx.Rollback()
	var lockedID uuid.UUID
	if err := tx.QueryRowContext(c.Request.Context(), `SELECT id FROM players WHERE id=$1 FOR UPDATE`, id).Scan(&lockedID); err != nil {
		c.JSON(500, gin.H{"error": "Không thể khóa trạng thái người chơi"})
		return
	}
	res, err := tx.ExecContext(c.Request.Context(), `UPDATE player_travel_journeys SET status='cancelled',revision=revision+1,next_step_at=NULL,last_event='Hành trình đã dừng',reason_code='cancelled_by_player',updated_at=NOW() WHERE player_id=$1 AND status IN ('running','paused_manual_combat','paused_condition')`, id)
	if err != nil {
		c.JSON(500, gin.H{"error": "Không thể dừng hành trình"})
		return
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		if err := tx.Commit(); err != nil {
			c.JSON(500, gin.H{"error": "Không thể xác nhận dừng hành trình"})
			return
		}
	}
	j, _ := loadTravelJourney(db.DB, id)
	if n == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Không có hành trình đang chạy"})
		return
	}
	PublishPlayerRealtimeEvent(id, PlayerEventUpdated, TravelJourneyDomain)
	c.JSON(http.StatusOK, gin.H{"journey": j})
}

func ResumeTravelJourney(c *gin.Context) {
	id, ok := travelPlayerID(c)
	if !ok {
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(500, gin.H{"error": "Không thể tiếp tục hành trình"})
		return
	}
	defer tx.Rollback()
	player, err := PlayerSvc.LoadForUpdateEquipped(tx, id)
	if err != nil {
		c.JSON(500, gin.H{"error": "Không thể tải trạng thái người chơi"})
		return
	}
	if player.ActiveCombat != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Hãy kết thúc chiến đấu trước khi tiếp tục"})
		return
	}
	var targetRegion string
	var targetX, targetY int
	if err := tx.QueryRowContext(c.Request.Context(), `SELECT target_region,target_x,target_y FROM player_travel_journeys WHERE player_id=$1 AND status IN ('paused_manual_combat','paused_condition') FOR UPDATE`, id).Scan(&targetRegion, &targetX, &targetY); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Hành trình hiện không thể tiếp tục"})
		return
	}
	movement := CalculatePlayerStats(&player)["movement"]
	status, reason, event := "running", "", "Hành trình tiếp tục"
	_, firstStepLength, routeOK := travelNextStep(player.CurrentRegion, targetRegion, player.X, player.Y, targetX, targetY, &player)
	if !routeOK {
		c.JSON(http.StatusConflict, gin.H{"error": "Không tìm được đường hiện tại tới đích"})
		return
	}
	var due any = time.Now().UTC().Add(travelStepDuration(movement, firstStepLength))
	if movement <= 0 {
		status, reason, event, due = "paused_condition", "movement_unavailable", "Hành trình vẫn tạm dừng vì không thể di chuyển.", nil
	}
	res, err := tx.ExecContext(c.Request.Context(), `UPDATE player_travel_journeys SET status=$2,revision=revision+1,next_step_at=$3,reason_code=$4,last_event=$5,updated_at=NOW() WHERE player_id=$1 AND status IN ('paused_manual_combat','paused_condition')`, id, status, due, reason, event)
	if err != nil {
		c.JSON(500, gin.H{"error": "Không thể tiếp tục hành trình"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Hành trình hiện không thể tiếp tục"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(500, gin.H{"error": "Không thể xác nhận tiếp tục hành trình"})
		return
	}
	PublishPlayerRealtimeEvent(id, PlayerEventUpdated, TravelJourneyDomain)
	j, _ := loadTravelJourney(db.DB, id)
	c.JSON(http.StatusOK, gin.H{"journey": j})
}

// StartTravelJourneyWorker advances due journeys one authoritative move request
// at a time. Position is reloaded before every step, so a committed move is the
// recovery cursor after a process restart.
func StartTravelJourneyWorker(ctx context.Context) {
	worker.RunEvery(ctx, time.Second, "Travel Journey Worker", func(ctx context.Context) {
		advanceDueTravelJourneys(ctx)
	})
}

func advanceDueTravelJourneys(ctx context.Context) {
	autoResumeConditionJourneys(ctx)
	rows, err := db.DB.QueryContext(ctx, `SELECT player_id FROM player_travel_journeys WHERE status='running' AND next_step_at<=NOW() ORDER BY next_step_at LIMIT 32`)
	if err != nil {
		return
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		advanceTravelJourney(ctx, id)
	}
}

func autoResumeConditionJourneys(ctx context.Context) {
	rows, err := db.DB.QueryContext(ctx, `SELECT player_id,journey_id,target_region,target_x,target_y FROM player_travel_journeys WHERE status='paused_condition' ORDER BY updated_at LIMIT 32`)
	if err != nil {
		return
	}
	type pausedJourney struct {
		playerID     uuid.UUID
		journeyID    uuid.UUID
		targetRegion string
		targetX      int
		targetY      int
	}
	var paused []pausedJourney
	for rows.Next() {
		var item pausedJourney
		if rows.Scan(&item.playerID, &item.journeyID, &item.targetRegion, &item.targetX, &item.targetY) == nil {
			paused = append(paused, item)
		}
	}
	rows.Close()
	for _, item := range paused {
		player, err := PlayerSvc.LoadEquipped(db.DB, item.playerID)
		if err != nil || CalculatePlayerStats(&player)["movement"] <= 0 {
			continue
		}
		_, firstStepLength, routeOK := travelNextStep(player.CurrentRegion, item.targetRegion, player.X, player.Y, item.targetX, item.targetY, &player)
		if !routeOK {
			finishTravelJourney(item.playerID, item.journeyID, "failed", "no_route", "Không tìm được đường hiện tại tới đích.")
			continue
		}
		due := time.Now().UTC().Add(travelStepDuration(CalculatePlayerStats(&player)["movement"], firstStepLength))
		res, err := db.DB.ExecContext(ctx, `UPDATE player_travel_journeys SET status='running',revision=revision+1,next_step_at=$3,reason_code='',last_event='Điều kiện di chuyển đã phục hồi; hành trình tiếp tục',updated_at=NOW() WHERE player_id=$1 AND journey_id=$2 AND status='paused_condition'`, item.playerID, item.journeyID, due)
		if err == nil {
			if n, _ := res.RowsAffected(); n > 0 {
				PublishPlayerRealtimeEvent(item.playerID, PlayerEventUpdated, TravelJourneyDomain)
			}
		}
	}
}

func advanceTravelJourney(ctx context.Context, playerID uuid.UUID) {
	conn, err := db.DB.Conn(ctx)
	if err != nil {
		return
	}
	defer conn.Close()
	var locked bool
	if err = conn.QueryRowContext(ctx, `SELECT pg_try_advisory_lock(hashtextextended($1,0))`, playerID.String()).Scan(&locked); err != nil || !locked {
		return
	}
	defer conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock(hashtextextended($1,0))`, playerID.String())
	var region, targetRegion string
	var partyID sql.NullString
	var scheduledAt time.Time
	var journeyID uuid.UUID
	var x, y, tx, ty int
	err = conn.QueryRowContext(ctx, `SELECT p.current_region,p.x,p.y,p.party_id::text,j.target_region,j.target_x,j.target_y,j.journey_id,j.next_step_at FROM players p JOIN player_travel_journeys j ON j.player_id=p.id WHERE p.id=$1 AND j.status='running' AND j.next_step_at<=NOW()`, playerID).Scan(&region, &x, &y, &partyID, &targetRegion, &tx, &ty, &journeyID, &scheduledAt)
	if err != nil {
		return
	}
	if travelAtTarget(region, targetRegion, x, y, tx, ty) {
		finishTravelJourney(playerID, journeyID, "completed", "arrived", "Đã tới đích.")
		return
	}
	statsPlayer, err := PlayerSvc.LoadEquipped(db.DB, playerID)
	if err != nil {
		return
	}
	if statsPlayer.HP <= 0 {
		finishTravelJourney(playerID, journeyID, "failed", "player_dead", "Hành trình kết thúc vì nhân vật đã tử trận.")
		return
	}
	if partyID.Valid {
		var leaderID uuid.UUID
		if err := db.DB.QueryRowContext(ctx, `SELECT leader_id FROM parties WHERE id=$1`, partyID.String).Scan(&leaderID); err != nil || leaderID != playerID {
			finishTravelJourney(playerID, journeyID, "failed", "not_party_leader", "Hành trình kết thúc vì nhân vật không còn là đội trưởng.")
			return
		}
	}
	stepMovement := calculatePlayerStatsAt(&statsPlayer, scheduledAt)["movement"]
	if stepMovement <= 0 {
		updateTravelJourney(playerID, journeyID, "paused_condition", nil, "movement_unavailable", "Hành trình tạm dừng vì không thể di chuyển.")
		return
	}
	step, _, ok := travelNextStep(region, targetRegion, x, y, tx, ty, &statsPlayer)
	if !ok {
		finishTravelJourney(playerID, journeyID, "failed", "no_route", "Không tìm được đường tới đích.")
		return
	}
	_, payload := ExecuteMovePlayerPath(ctx, playerID, MoveRequest{X: step[0], Y: step[1], TravelJourney: true, TravelJourneyID: journeyID})
	if payload == nil {
		return
	}
	if _, failed := payload["error"]; failed {
		finishTravelJourney(playerID, journeyID, "failed", "movement_denied", "Hành trình dừng vì bước tiếp theo không hợp lệ.")
	}
	return
}

func travelAtTarget(regionID, targetRegionID string, x, y, targetX, targetY int) bool {
	return regionID == targetRegionID && x == targetX && y == targetY
}

func travelNextStep(regionID, targetRegionID string, x, y, targetX, targetY int, players ...*models.Player) ([2]int, float64, bool) {
	goalX, goalY := targetX, targetY
	if regionID != targetRegionID {
		current, target := ZoneByID(regionID), ZoneByID(targetRegionID)
		if current == nil || target == nil {
			return [2]int{}, 0, false
		}
		route := travelRegionRoute(current, target)
		if len(players) > 0 && players[0] != nil {
			route = travelRegionRouteForPlayer(current, target, players[0])
		}
		if len(route) < 2 {
			return [2]int{}, 0, false
		}
		next := route[1]
		dirX, dirY := 0, 0
		if next.AtlasX > current.AtlasX {
			dirX = 1
		} else if next.AtlasX < current.AtlasX {
			dirX = -1
		}
		if next.AtlasY > current.AtlasY {
			dirY = 1
		} else if next.AtlasY < current.AtlasY {
			dirY = -1
		}
		if dirX == 0 && dirY == 0 {
			return [2]int{}, 0, false
		}
		goalX, goalY = x, y
		if dirX > 0 {
			goalX = current.MaxX
		} else if dirX < 0 {
			goalX = current.MinX
		}
		if dirY > 0 {
			goalY = current.MaxY
		} else if dirY < 0 {
			goalY = current.MinY
		}
		if dirX != 0 && dirY == 0 {
			goalY = clampTravelCoordinate(y, current.MinY, current.MaxY)
		}
		if dirY != 0 && dirX == 0 {
			goalX = clampTravelCoordinate(x, current.MinX, current.MaxX)
		}
		if x == goalX && y == goalY {
			return [2]int{x + dirX, y + dirY}, 1, true
		}
	}
	path := lineCells(x, y, goalX, goalY)
	if len(path) == 0 {
		return [2]int{}, 0, false
	}
	step := path[0]
	length := 1.0
	if step[0] != x && step[1] != y {
		length = 1.41421356237
	}
	return step, length, true
}

func travelRegionRoute(start, target *models.Region) []*models.Region {
	return travelRegionRouteBy(start, target, nil)
}

func travelRegionRouteForPlayer(start, target *models.Region, player *models.Player) []*models.Region {
	if player == nil {
		return travelRegionRoute(start, target)
	}
	return travelRegionRouteBy(start, target, func(zone *models.Region) bool {
		return checkRegionEntry(zone.ID, player.Rank, player.GuInventory) == nil
	})
}

func travelRegionRouteBy(start, target *models.Region, canEnter func(*models.Region) bool) []*models.Region {
	if start == nil || target == nil {
		return nil
	}
	if start.ID == target.ID {
		return []*models.Region{start}
	}
	type routeNode struct {
		zone *models.Region
		path []*models.Region
	}
	queue := []routeNode{{zone: start, path: []*models.Region{start}}}
	seen := map[string]bool{start.ID: true}
	directions := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, direction := range directions {
			next := FindNeighborZoneOnAtlas(current.zone, direction[0], direction[1])
			if next == nil || seen[next.ID] || (canEnter != nil && !canEnter(next)) {
				continue
			}
			path := append(append([]*models.Region(nil), current.path...), next)
			if next.ID == target.ID {
				return path
			}
			seen[next.ID] = true
			queue = append(queue, routeNode{zone: next, path: path})
		}
	}
	return nil
}

func clampTravelCoordinate(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func updateTravelJourney(id, journeyID uuid.UUID, status string, nextDue *time.Time, reason, event string) {
	res, err := db.DB.Exec(`UPDATE player_travel_journeys SET status=$3,revision=revision+1,next_step_at=$4,last_step_at=NOW(),reason_code=$5,last_event=$6,updated_at=NOW() WHERE player_id=$1 AND journey_id=$2 AND status IN ('running','paused_manual_combat','paused_condition')`, id, journeyID, status, nextDue, reason, event)
	if err == nil {
		if n, _ := res.RowsAffected(); n > 0 {
			PublishPlayerRealtimeEvent(id, PlayerEventUpdated, TravelJourneyDomain)
		}
	}
}

func finishTravelJourney(id, journeyID uuid.UUID, status, reason, event string) {
	res, err := db.DB.Exec(`UPDATE player_travel_journeys SET status=$3,revision=revision+1,next_step_at=NULL,last_step_at=NOW(),reason_code=$4,last_event=$5,updated_at=NOW() WHERE player_id=$1 AND journey_id=$2 AND status IN ('running','paused_condition','paused_manual_combat')`, id, journeyID, status, reason, event)
	if err == nil {
		if n, _ := res.RowsAffected(); n > 0 {
			PublishPlayerRealtimeEvent(id, PlayerEventUpdated, TravelJourneyDomain)
		}
	}
}
