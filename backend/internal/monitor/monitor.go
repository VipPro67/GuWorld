package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	maxTraces         = 500
	maxLatencySamples = 2000
	maxEndpoints      = 300
	rpsWindowSeconds  = 60
	maxRequestBody    = 8 * 1024
)

var traceSeq uint64

func generateTraceID() string {
	seq := atomic.AddUint64(&traceSeq, 1)
	now := time.Now().UnixNano()
	return fmt.Sprintf("%08x%04x", uint32(now), uint16(seq))
}

// RequestTrace stores individual request execution details for diagnosis.
type RequestTrace struct {
	ID            string      `json:"id"`
	Timestamp     time.Time   `json:"timestamp"`
	Method        string      `json:"method"`
	Path          string      `json:"path"`
	Route         string      `json:"route,omitempty"`
	Query         string      `json:"query,omitempty"`
	RequestBody   interface{} `json:"request_body,omitempty"`
	BodyTruncated bool        `json:"body_truncated,omitempty"`
	ContentLength int64       `json:"content_length,omitempty"`
	Status        int         `json:"status"`
	DurationMs    float64     `json:"duration_ms"`
	ClientIP      string      `json:"client_ip"`
	PlayerID      string      `json:"player_id,omitempty"`
	Username      string      `json:"username,omitempty"`
	UserAgent     string      `json:"user_agent,omitempty"`
	Error         string      `json:"error,omitempty"`
}

var sensitiveKeys = map[string]struct{}{
	"authorization": {}, "cookie": {}, "password": {}, "pass": {}, "secret": {},
	"token": {}, "access_token": {}, "refresh_token": {}, "otp": {}, "api_key": {},
	"credit_card": {}, "card_number": {}, "cvv": {},
}

func isSensitiveKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	if _, ok := sensitiveKeys[key]; ok {
		return true
	}
	return strings.Contains(key, "password") || strings.HasSuffix(key, "_token") || strings.HasSuffix(key, "_secret")
}

func redactJSON(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		for key, item := range typed {
			if isSensitiveKey(key) {
				typed[key] = "[REDACTED]"
			} else {
				typed[key] = redactJSON(item)
			}
		}
	case []interface{}:
		for index, item := range typed {
			typed[index] = redactJSON(item)
		}
	}
	return value
}

func sanitizeQuery(raw string) string {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return "[INVALID QUERY]"
	}
	for key := range values {
		if isSensitiveKey(key) {
			values.Set(key, "[REDACTED]")
		}
	}
	return values.Encode()
}

type replayBody struct {
	io.Reader
	closer io.Closer
}

func (body *replayBody) Close() error { return body.closer.Close() }

func captureRequestBody(request *http.Request) (interface{}, bool) {
	sensitivePrefixes := []string{"/api/auth", "/api/webhooks", "/api/topup", "/api/chat"}
	for _, prefix := range sensitivePrefixes {
		if strings.HasPrefix(request.URL.Path, prefix) {
			return nil, false
		}
	}
	if request.Body == nil || !strings.Contains(strings.ToLower(request.Header.Get("Content-Type")), "application/json") {
		return nil, false
	}
	prefix, err := io.ReadAll(io.LimitReader(request.Body, maxRequestBody+1))
	request.Body = &replayBody{Reader: io.MultiReader(bytes.NewReader(prefix), request.Body), closer: request.Body}
	if err != nil {
		return nil, false
	}
	truncated := len(prefix) > maxRequestBody
	captured := prefix
	if truncated {
		captured = prefix[:maxRequestBody]
	}
	if truncated {
		return "[TRUNCATED JSON OMITTED]", true
	}
	var parsed interface{}
	if json.Unmarshal(captured, &parsed) != nil {
		return "[INVALID JSON OMITTED]", false
	}
	return redactJSON(parsed), false
}

// EndpointSummary gives aggregated performance metrics for a specific endpoint.
type EndpointSummary struct {
	Method        string  `json:"method"`
	Path          string  `json:"path"`
	Count         int64   `json:"count"`
	ErrorCount    int64   `json:"error_count"`
	AvgDurationMs float64 `json:"avg_duration_ms"`
	MaxDurationMs float64 `json:"max_duration_ms"`
}

// RequestSummary aggregates server-wide HTTP metrics.
type RequestSummary struct {
	TotalRequests     int64             `json:"total_requests"`
	ActiveRequests    int64             `json:"active_requests"`
	RequestsPerSecond float64           `json:"requests_per_second"`
	Status2xx         int64             `json:"status_2xx"`
	Status3xx         int64             `json:"status_3xx"`
	Status4xx         int64             `json:"status_4xx"`
	Status5xx         int64             `json:"status_5xx"`
	LatencyMinMs      float64           `json:"latency_min_ms"`
	LatencyMaxMs      float64           `json:"latency_max_ms"`
	LatencyAvgMs      float64           `json:"latency_avg_ms"`
	LatencyP50Ms      float64           `json:"latency_p50_ms"`
	LatencyP90Ms      float64           `json:"latency_p90_ms"`
	LatencyP99Ms      float64           `json:"latency_p99_ms"`
	TopEndpoints      []EndpointSummary `json:"top_endpoints"`
	SlowestEndpoints  []EndpointSummary `json:"slowest_endpoints"`
}

// FilterOptions specifies query parameters for searching request traces.
type FilterOptions struct {
	Limit         int
	Offset        int
	Method        string
	Status        int // 0 = all, -1 = errors (status >= 400), or exact status like 200, 500
	Path          string
	MinDurationMs float64
	Search        string // Matches player_id, username, or client_ip
}

// FilterResult holds the paginated traces query response.
type FilterResult struct {
	Total  int            `json:"total"`
	Traces []RequestTrace `json:"traces"`
}

type endpointStatInternal struct {
	method        string
	path          string
	count         int64
	errorCount    int64
	totalDuration float64
	maxDuration   float64
}

// RequestTracker tracks incoming and completed requests in-memory.
type RequestTracker struct {
	totalRequests  int64
	activeRequests int64
	status2xx      int64
	status3xx      int64
	status4xx      int64
	status5xx      int64

	// RPS sliding window
	rpsMu      sync.Mutex
	rpsBuckets [rpsWindowSeconds]int64
	rpsTimes   [rpsWindowSeconds]int64

	// Latency reservoir
	latencyMu      sync.RWMutex
	latencySamples []float64
	latencyMin     float64
	latencyMax     float64
	latencySum     float64
	latencyCount   int64

	// Endpoint stats (bounded)
	endpointsMu sync.RWMutex
	endpoints   map[string]*endpointStatInternal

	// Fixed circular ring buffer for traces
	tracesMu   sync.RWMutex
	tracesRing [maxTraces]RequestTrace
	traceHead  int // next write position
	traceCount int // valid count <= maxTraces
}

var GlobalTracker = NewRequestTracker()

// NewRequestTracker initializes a new RequestTracker.
func NewRequestTracker() *RequestTracker {
	return &RequestTracker{
		endpoints:      make(map[string]*endpointStatInternal),
		latencySamples: make([]float64, 0, maxLatencySamples),
		latencyMin:     -1,
	}
}

// RecordCompleted registers a finished HTTP request.
func (t *RequestTracker) RecordCompleted(trace RequestTrace) {
	atomic.AddInt64(&t.totalRequests, 1)

	// Status category counters
	switch {
	case trace.Status >= 200 && trace.Status < 300:
		atomic.AddInt64(&t.status2xx, 1)
	case trace.Status >= 300 && trace.Status < 400:
		atomic.AddInt64(&t.status3xx, 1)
	case trace.Status >= 400 && trace.Status < 500:
		atomic.AddInt64(&t.status4xx, 1)
	case trace.Status >= 500:
		atomic.AddInt64(&t.status5xx, 1)
	}

	// RPS bucket update
	nowSec := time.Now().Unix()
	bucketIdx := nowSec % rpsWindowSeconds
	t.rpsMu.Lock()
	if t.rpsTimes[bucketIdx] != nowSec {
		t.rpsTimes[bucketIdx] = nowSec
		t.rpsBuckets[bucketIdx] = 1
	} else {
		t.rpsBuckets[bucketIdx]++
	}
	t.rpsMu.Unlock()

	dur := trace.DurationMs

	// Latency recording
	t.latencyMu.Lock()
	if t.latencyMin < 0 || dur < t.latencyMin {
		t.latencyMin = dur
	}
	if dur > t.latencyMax {
		t.latencyMax = dur
	}
	t.latencySum += dur
	t.latencyCount++
	if len(t.latencySamples) < maxLatencySamples {
		t.latencySamples = append(t.latencySamples, dur)
	} else {
		// Reservoir replacement using synchronized latencyCount
		idx := int((t.latencyCount - 1) % maxLatencySamples)
		t.latencySamples[idx] = dur
	}
	t.latencyMu.Unlock()

	// Endpoint statistics (bounded to avoid memory leak on dynamic paths)
	epPath := trace.Route
	if epPath == "" {
		epPath = trace.Path
	}
	epKey := trace.Method + " " + epPath

	t.endpointsMu.Lock()
	stat, exists := t.endpoints[epKey]
	if !exists {
		if len(t.endpoints) < maxEndpoints {
			stat = &endpointStatInternal{
				method: trace.Method,
				path:   epPath,
			}
			t.endpoints[epKey] = stat
		}
	}
	if stat != nil {
		stat.count++
		if trace.Status >= 400 {
			stat.errorCount++
		}
		stat.totalDuration += dur
		if dur > stat.maxDuration {
			stat.maxDuration = dur
		}
	}
	t.endpointsMu.Unlock()

	// Circular ring buffer insertion
	t.tracesMu.Lock()
	t.tracesRing[t.traceHead] = trace
	t.traceHead = (t.traceHead + 1) % maxTraces
	if t.traceCount < maxTraces {
		t.traceCount++
	}
	t.tracesMu.Unlock()
}

// GetSummary returns an aggregated overview of HTTP request stats.
func (t *RequestTracker) GetSummary() RequestSummary {
	total := atomic.LoadInt64(&t.totalRequests)
	active := atomic.LoadInt64(&t.activeRequests)
	s2xx := atomic.LoadInt64(&t.status2xx)
	s3xx := atomic.LoadInt64(&t.status3xx)
	s4xx := atomic.LoadInt64(&t.status4xx)
	s5xx := atomic.LoadInt64(&t.status5xx)

	// Calculate RPS over the last 60 seconds
	nowSec := time.Now().Unix()
	var rpsCount int64
	var validBuckets float64
	t.rpsMu.Lock()
	for i := 0; i < rpsWindowSeconds; i++ {
		sec := t.rpsTimes[i]
		if sec > 0 && (nowSec-sec) < rpsWindowSeconds && (nowSec-sec) >= 0 {
			rpsCount += t.rpsBuckets[i]
			validBuckets++
		}
	}
	t.rpsMu.Unlock()

	rps := 0.0
	if validBuckets > 0 {
		rps = round2(float64(rpsCount) / 60.0)
	}

	// Calculate latency percentiles
	t.latencyMu.RLock()
	minLat := t.latencyMin
	if minLat < 0 {
		minLat = 0
	}
	maxLat := t.latencyMax
	avgLat := 0.0
	if t.latencyCount > 0 {
		avgLat = round2(t.latencySum / float64(t.latencyCount))
	}

	samplesCopy := make([]float64, len(t.latencySamples))
	copy(samplesCopy, t.latencySamples)
	t.latencyMu.RUnlock()

	sort.Float64s(samplesCopy)
	p50 := 0.0
	p90 := 0.0
	p99 := 0.0
	n := len(samplesCopy)
	if n > 0 {
		p50 = samplesCopy[int(float64(n)*0.50)]
		p90 = samplesCopy[int(float64(n)*0.90)]
		idx99 := int(float64(n) * 0.99)
		if idx99 >= n {
			idx99 = n - 1
		}
		p99 = samplesCopy[idx99]
	}

	// Get Top endpoints and Slowest endpoints
	t.endpointsMu.RLock()
	allEndpoints := make([]EndpointSummary, 0, len(t.endpoints))
	for _, ep := range t.endpoints {
		avg := 0.0
		if ep.count > 0 {
			avg = round2(ep.totalDuration / float64(ep.count))
		}
		allEndpoints = append(allEndpoints, EndpointSummary{
			Method:        ep.method,
			Path:          ep.path,
			Count:         ep.count,
			ErrorCount:    ep.errorCount,
			AvgDurationMs: avg,
			MaxDurationMs: round2(ep.maxDuration),
		})
	}
	t.endpointsMu.RUnlock()

	// Top by count
	topByCount := make([]EndpointSummary, len(allEndpoints))
	copy(topByCount, allEndpoints)
	sort.Slice(topByCount, func(i, j int) bool {
		return topByCount[i].Count > topByCount[j].Count
	})
	if len(topByCount) > 10 {
		topByCount = topByCount[:10]
	}

	// Top by slowest average
	topBySlow := make([]EndpointSummary, len(allEndpoints))
	copy(topBySlow, allEndpoints)
	sort.Slice(topBySlow, func(i, j int) bool {
		return topBySlow[i].AvgDurationMs > topBySlow[j].AvgDurationMs
	})
	if len(topBySlow) > 10 {
		topBySlow = topBySlow[:10]
	}

	return RequestSummary{
		TotalRequests:     total,
		ActiveRequests:    active,
		RequestsPerSecond: rps,
		Status2xx:         s2xx,
		Status3xx:         s3xx,
		Status4xx:         s4xx,
		Status5xx:         s5xx,
		LatencyMinMs:      round2(minLat),
		LatencyMaxMs:      round2(maxLat),
		LatencyAvgMs:      avgLat,
		LatencyP50Ms:      round2(p50),
		LatencyP90Ms:      round2(p90),
		LatencyP99Ms:      round2(p99),
		TopEndpoints:      topByCount,
		SlowestEndpoints:  topBySlow,
	}
}

// FilterTraces searches and filters recent request traces in reverse chronological order.
func (t *RequestTracker) FilterTraces(opts FilterOptions) FilterResult {
	t.tracesMu.RLock()
	count := t.traceCount
	head := t.traceHead
	ordered := make([]RequestTrace, 0, count)

	// Iterate circular buffer from newest to oldest
	for i := 0; i < count; i++ {
		idx := (head - 1 - i + maxTraces) % maxTraces
		ordered = append(ordered, t.tracesRing[idx])
	}
	t.tracesMu.RUnlock()

	var filtered []RequestTrace
	methodUpper := strings.ToUpper(strings.TrimSpace(opts.Method))
	pathLower := strings.ToLower(strings.TrimSpace(opts.Path))
	searchLower := strings.ToLower(strings.TrimSpace(opts.Search))

	for _, trace := range ordered {
		if methodUpper != "" && strings.ToUpper(trace.Method) != methodUpper {
			continue
		}
		if pathLower != "" && !strings.Contains(strings.ToLower(trace.Path), pathLower) {
			continue
		}
		if opts.Status == -1 {
			if trace.Status < 400 {
				continue
			}
		} else if opts.Status > 0 && trace.Status != opts.Status {
			continue
		}
		if opts.MinDurationMs > 0 && trace.DurationMs < opts.MinDurationMs {
			continue
		}
		if searchLower != "" {
			match := strings.Contains(strings.ToLower(trace.PlayerID), searchLower) ||
				strings.Contains(strings.ToLower(trace.Username), searchLower) ||
				strings.Contains(strings.ToLower(trace.ClientIP), searchLower) ||
				strings.Contains(strings.ToLower(trace.Path), searchLower)
			if !match {
				continue
			}
		}
		filtered = append(filtered, trace)
	}

	total := len(filtered)
	if opts.Offset > total {
		return FilterResult{Total: total, Traces: []RequestTrace{}}
	}

	filtered = filtered[opts.Offset:]
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return FilterResult{
		Total:  total,
		Traces: filtered,
	}
}

// GetTraceByID retrieves a single request trace by its unique ID.
func (t *RequestTracker) GetTraceByID(id string) (RequestTrace, bool) {
	t.tracesMu.RLock()
	defer t.tracesMu.RUnlock()

	for i := 0; i < t.traceCount; i++ {
		idx := (t.traceHead - 1 - i + maxTraces) % maxTraces
		if t.tracesRing[idx].ID == id {
			return t.tracesRing[idx], true
		}
	}
	return RequestTrace{}, false
}

// Reset clears all recorded metrics and traces.
func (t *RequestTracker) Reset() {
	atomic.StoreInt64(&t.totalRequests, 0)
	atomic.StoreInt64(&t.status2xx, 0)
	atomic.StoreInt64(&t.status3xx, 0)
	atomic.StoreInt64(&t.status4xx, 0)
	atomic.StoreInt64(&t.status5xx, 0)

	t.rpsMu.Lock()
	t.rpsBuckets = [rpsWindowSeconds]int64{}
	t.rpsTimes = [rpsWindowSeconds]int64{}
	t.rpsMu.Unlock()

	t.latencyMu.Lock()
	t.latencySamples = make([]float64, 0, maxLatencySamples)
	t.latencyMin = -1
	t.latencyMax = 0
	t.latencySum = 0
	t.latencyCount = 0
	t.latencyMu.Unlock()

	t.endpointsMu.Lock()
	t.endpoints = make(map[string]*endpointStatInternal)
	t.endpointsMu.Unlock()

	t.tracesMu.Lock()
	t.tracesRing = [maxTraces]RequestTrace{}
	t.traceHead = 0
	t.traceCount = 0
	t.tracesMu.Unlock()
}

// Middleware creates a Gin middleware that records request performance and traces.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		atomic.AddInt64(&GlobalTracker.activeRequests, 1)
		defer atomic.AddInt64(&GlobalTracker.activeRequests, -1)

		// Fast, non-blocking trace ID generation
		traceID := generateTraceID()
		c.Set("trace_id", traceID)
		c.Header("X-Trace-ID", traceID)
		requestBody, bodyTruncated := captureRequestBody(c.Request)
		c.Set("request_body", requestBody)
		c.Set("request_body_truncated", bodyTruncated)

		c.Next()

		duration := time.Since(start)

		method := c.Request.Method
		if method == http.MethodOptions {
			return
		}

		path := c.Request.URL.Path
		route := c.FullPath()
		query := sanitizeQuery(c.Request.URL.RawQuery)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		var playerIDStr string
		var usernameStr string
		if pid, exists := c.Get("player_id"); exists {
			playerIDStr = fmt.Sprintf("%v", pid)
		}
		if u, exists := c.Get("username"); exists {
			usernameStr = fmt.Sprintf("%v", u)
		}

		var errMsg string
		if len(c.Errors) > 0 {
			errMsg = c.Errors.String()
		}

		durMs := float64(duration.Nanoseconds()) / 1e6

		GlobalTracker.RecordCompleted(RequestTrace{
			ID:            traceID,
			Timestamp:     start,
			Method:        method,
			Path:          path,
			Route:         route,
			Query:         query,
			RequestBody:   requestBody,
			BodyTruncated: bodyTruncated,
			ContentLength: c.Request.ContentLength,
			Status:        status,
			DurationMs:    round2(durMs),
			ClientIP:      clientIP,
			PlayerID:      playerIDStr,
			Username:      usernameStr,
			UserAgent:     userAgent,
			Error:         errMsg,
		})
	}
}
