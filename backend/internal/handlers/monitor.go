package handlers

import (
	"context"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/monitor"
)

var (
	serverStartTime  = time.Now()
	wsStatsMu        sync.RWMutex
	wsStatsFn        func() (clients int, channels int)
	realtimeStatsMu  sync.RWMutex
	realtimeStatsFn  func() any
	realtimeStatsErr string
)

// SetWSStatsProvider sets the provider function for WebSocket statistics.
func SetWSStatsProvider(fn func() (int, int)) {
	wsStatsMu.Lock()
	defer wsStatsMu.Unlock()
	wsStatsFn = fn
}

func getWSStats() (int, int) {
	wsStatsMu.RLock()
	fn := wsStatsFn
	wsStatsMu.RUnlock()
	if fn == nil {
		return 0, 0
	}
	return fn()
}

// SetRealtimeStatsProvider exposes per-zone realtime telemetry to the admin
// monitor endpoint. It is set once during startup.
func SetRealtimeStatsProvider(fn func() any) {
	realtimeStatsMu.Lock()
	defer realtimeStatsMu.Unlock()
	realtimeStatsFn = fn
}

// SetRealtimeRuntimeError records why startup did not install the realtime
// manager. The admin monitor can then distinguish a disabled rollout from a
// map/configuration startup failure without exposing this detail to players.
func SetRealtimeRuntimeError(reason string) {
	realtimeStatsMu.Lock()
	defer realtimeStatsMu.Unlock()
	realtimeStatsErr = reason
}

// AdminGetRealtimeStats returns per-zone tick/entity/player/subscriber stats.
func AdminGetRealtimeStats(c *gin.Context) {
	realtimeStatsMu.RLock()
	fn := realtimeStatsFn
	reason := realtimeStatsErr
	realtimeStatsMu.RUnlock()
	if fn == nil {
		response := gin.H{"error": "realtime runtime is not configured"}
		if reason != "" {
			response["reason"] = reason
		}
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}
	c.JSON(http.StatusOK, gin.H{"zones": fn()})
}

// DBStatsResponse represents database connection pool diagnostics.
type DBStatsResponse struct {
	MaxOpenConnections int    `json:"max_open_connections"`
	OpenConnections    int    `json:"open_connections"`
	InUse              int    `json:"in_use"`
	Idle               int    `json:"idle"`
	WaitCount          int64  `json:"wait_count"`
	WaitDuration       string `json:"wait_duration"`
	MaxIdleClosed      int64  `json:"max_idle_closed"`
	MaxIdleTimeClosed  int64  `json:"max_idle_time_closed"`
	MaxLifetimeClosed  int64  `json:"max_lifetime_closed"`
}

// RedisStatsResponse represents Redis connection health and performance.
type RedisStatsResponse struct {
	Connected     bool    `json:"connected"`
	PingLatencyMs float64 `json:"ping_latency_ms"`
	Error         string  `json:"error,omitempty"`
}

// ServerMetadata contains host and runtime process metadata.
type ServerMetadata struct {
	PID       int    `json:"pid"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	StartTime string `json:"start_time"`
	Uptime    string `json:"uptime"`
	UptimeSec int64  `json:"uptime_sec"`
}

// WebSocketStatsResponse represents WebSocket hub metrics.
type WebSocketStatsResponse struct {
	ConnectedClients int `json:"connected_clients"`
	ActiveChannels   int `json:"active_channels"`
}

// FullSystemStats aggregates all subsystems into a single status report.
type FullSystemStats struct {
	Server    ServerMetadata         `json:"server"`
	CPU       monitor.CPUMetrics     `json:"cpu"`
	Memory    monitor.MemoryMetrics  `json:"memory"`
	Database  DBStatsResponse        `json:"database"`
	Redis     RedisStatsResponse     `json:"redis"`
	WebSocket WebSocketStatsResponse `json:"websocket"`
}

// collectSystemStats builds the complete system diagnostic state.
func collectSystemStats() FullSystemStats {
	now := time.Now()
	uptime := now.Sub(serverStartTime)

	serverMeta := ServerMetadata{
		PID:       os.Getpid(),
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		StartTime: serverStartTime.Format(time.RFC3339),
		Uptime:    uptime.Round(time.Second).String(),
		UptimeSec: int64(uptime.Seconds()),
	}

	cpuMetrics := monitor.GetCPUMetrics()
	memMetrics := monitor.GetMemoryMetrics()

	// DB stats
	var dbResp DBStatsResponse
	if db.DB != nil {
		dbStats := db.DB.Stats()
		dbResp = DBStatsResponse{
			MaxOpenConnections: dbStats.MaxOpenConnections,
			OpenConnections:    dbStats.OpenConnections,
			InUse:              dbStats.InUse,
			Idle:               dbStats.Idle,
			WaitCount:          dbStats.WaitCount,
			WaitDuration:       dbStats.WaitDuration.String(),
			MaxIdleClosed:      dbStats.MaxIdleClosed,
			MaxIdleTimeClosed:  dbStats.MaxIdleTimeClosed,
			MaxLifetimeClosed:  dbStats.MaxLifetimeClosed,
		}
	}

	// Redis stats
	redisResp := RedisStatsResponse{Connected: false}
	if cache.Client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		pingStart := time.Now()
		err := cache.Client.Ping(ctx).Err()
		cancel()
		if err != nil {
			redisResp.Error = err.Error()
		} else {
			redisResp.Connected = true
			redisResp.PingLatencyMs = float64(time.Since(pingStart).Nanoseconds()) / 1e6
		}
	} else {
		redisResp.Error = "Redis client not initialized"
	}

	// WebSocket stats
	wsClients, wsChannels := getWSStats()
	wsResp := WebSocketStatsResponse{
		ConnectedClients: wsClients,
		ActiveChannels:   wsChannels,
	}

	return FullSystemStats{
		Server:    serverMeta,
		CPU:       cpuMetrics,
		Memory:    memMetrics,
		Database:  dbResp,
		Redis:     redisResp,
		WebSocket: wsResp,
	}
}

// AdminGetSystemStats handles GET /api/admin/monitor/system.
func AdminGetSystemStats(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	c.JSON(http.StatusOK, collectSystemStats())
}

// AdminGetRequestSummary handles GET /api/admin/monitor/requests/summary.
func AdminGetRequestSummary(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	summary := monitor.GlobalTracker.GetSummary()
	c.JSON(http.StatusOK, summary)
}

// AdminGetOperationMetrics returns counters for privileged authorization,
// mutation, import, and audit events since process start or the last reset.
func AdminGetOperationMetrics(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	c.JSON(http.StatusOK, monitor.GlobalAdminOperations.Snapshot())
}

// AdminGetRequestTraces handles GET /api/admin/monitor/requests/traces.
func AdminGetRequestTraces(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	paging := parseAdminPageQuery(c, 50, map[string]string{"timestamp": "timestamp", "duration_ms": "duration_ms", "status": "status", "method": "method", "path": "path"}, "timestamp")
	statusStr := strings.TrimSpace(c.Query("status"))
	method := c.Query("method")
	path := c.Query("path")
	search := c.Query("search")
	minDur, _ := strconv.ParseFloat(c.Query("min_duration_ms"), 64)

	var status int
	if statusStr == "errors" || statusStr == "error" || statusStr == "-1" {
		status = -1
	} else if statusStr != "" {
		status, _ = strconv.Atoi(statusStr)
	}

	opts := monitor.FilterOptions{
		Limit:         paging.PageSize,
		Offset:        paging.Offset,
		Method:        method,
		Status:        status,
		Path:          path,
		MinDurationMs: minDur,
		Search:        search,
		Sort:          paging.Sort,
		Direction:     paging.Direction,
	}

	result := monitor.GlobalTracker.FilterTraces(opts)
	c.JSON(http.StatusOK, gin.H{"data": result.Traces, "traces": result.Traces, "total": result.Total, "page": paging.Page, "page_size": paging.PageSize})
}

// AdminGetRequestTraceDetail handles GET /api/admin/monitor/requests/traces/:id.
func AdminGetRequestTraceDetail(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	id := c.Param("id")
	trace, ok := monitor.GlobalTracker.GetTraceByID(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Trace not found"})
		return
	}
	c.JSON(http.StatusOK, trace)
}

// AdminResetRequestMetrics handles POST /api/admin/monitor/requests/reset.
func AdminResetRequestMetrics(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	monitor.GlobalTracker.Reset()
	monitor.GlobalAdminOperations.Reset()
	c.JSON(http.StatusOK, gin.H{"message": "Request metrics and traces reset successfully"})
}
