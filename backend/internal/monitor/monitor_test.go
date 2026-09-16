package monitor

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCaptureRequestBodyRedactsAndPreservesJSON(t *testing.T) {
	body := `{"username":"alice","password":"secret","nested":{"access_token":"abc"}}`
	request := httptest.NewRequest("POST", "/api/player/action?token=secret&view=full", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	captured, truncated := captureRequestBody(request)
	if truncated {
		t.Fatal("small body should not be truncated")
	}
	result := captured.(map[string]interface{})
	if result["password"] != "[REDACTED]" || result["nested"].(map[string]interface{})["access_token"] != "[REDACTED]" {
		t.Fatalf("sensitive values were not redacted: %#v", result)
	}
	replayed, err := io.ReadAll(request.Body)
	if err != nil || string(replayed) != body {
		t.Fatalf("request body was not preserved: %q, %v", replayed, err)
	}
	if got := sanitizeQuery(request.URL.RawQuery); got != "token=%5BREDACTED%5D&view=full" {
		t.Fatalf("unexpected sanitized query: %s", got)
	}
}

func TestCaptureRequestBodySkipsSensitiveRoutes(t *testing.T) {
	request := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"password":"secret"}`))
	request.Header.Set("Content-Type", "application/json")
	if captured, _ := captureRequestBody(request); captured != nil {
		t.Fatalf("sensitive route body should not be captured: %#v", captured)
	}
}

func TestCaptureRequestBodyLimitsAndSkipsNonJSON(t *testing.T) {
	large := `{"value":"` + strings.Repeat("x", maxRequestBody) + `"}`
	request := httptest.NewRequest("POST", "/api/test", strings.NewReader(large))
	request.Header.Set("Content-Type", "application/json")
	captured, truncated := captureRequestBody(request)
	if !truncated || captured != "[TRUNCATED JSON OMITTED]" {
		t.Fatalf("expected safely omitted truncated capture, got %#v", captured)
	}

	binary := httptest.NewRequest("POST", "/api/upload", strings.NewReader("secret"))
	binary.Header.Set("Content-Type", "multipart/form-data")
	if captured, _ := captureRequestBody(binary); captured != nil {
		t.Fatalf("non-JSON body should not be captured: %#v", captured)
	}
}

func TestRequestTracker_RecordAndSummary(t *testing.T) {
	tracker := NewRequestTracker()

	tracker.RecordCompleted(RequestTrace{
		ID:         "t1",
		Timestamp:  time.Now(),
		Method:     "GET",
		Path:       "/api/player/profile",
		Status:     200,
		DurationMs: 12.5,
		ClientIP:   "127.0.0.1",
		PlayerID:   "p1",
		Username:   "PlayerOne",
	})

	tracker.RecordCompleted(RequestTrace{
		ID:         "t2",
		Timestamp:  time.Now(),
		Method:     "POST",
		Path:       "/api/combat/attack",
		Status:     500,
		DurationMs: 45.0,
		ClientIP:   "127.0.0.1",
		PlayerID:   "p2",
		Username:   "PlayerTwo",
		Error:      "server timeout",
	})

	tracker.RecordCompleted(RequestTrace{
		ID:         "t3",
		Timestamp:  time.Now(),
		Method:     "GET",
		Path:       "/api/player/profile",
		Status:     200,
		DurationMs: 15.0,
		ClientIP:   "127.0.0.2",
		PlayerID:   "p1",
		Username:   "PlayerOne",
	})

	summary := tracker.GetSummary()
	if summary.TotalRequests != 3 {
		t.Fatalf("expected 3 total requests, got %d", summary.TotalRequests)
	}
	if summary.Status2xx != 2 {
		t.Errorf("expected 2 2xx statuses, got %d", summary.Status2xx)
	}
	if summary.Status5xx != 1 {
		t.Errorf("expected 1 5xx status, got %d", summary.Status5xx)
	}
	if summary.LatencyMinMs != 12.5 {
		t.Errorf("expected min latency 12.5, got %f", summary.LatencyMinMs)
	}
	if summary.LatencyMaxMs != 45.0 {
		t.Errorf("expected max latency 45.0, got %f", summary.LatencyMaxMs)
	}

	// Test endpoint aggregation
	if len(summary.TopEndpoints) == 0 {
		t.Fatalf("expected top endpoints to be populated")
	}
	if summary.TopEndpoints[0].Path != "/api/player/profile" || summary.TopEndpoints[0].Count != 2 {
		t.Errorf("expected top endpoint /api/player/profile with count 2, got %+v", summary.TopEndpoints[0])
	}
}

func TestRequestTracker_FilterTraces(t *testing.T) {
	tracker := NewRequestTracker()

	tracker.RecordCompleted(RequestTrace{
		ID:         "trace-1",
		Timestamp:  time.Now().Add(-2 * time.Second),
		Method:     "GET",
		Path:       "/api/inventory",
		Status:     200,
		DurationMs: 5.0,
		ClientIP:   "10.0.0.1",
		PlayerID:   "user-a",
		Username:   "Alice",
	})
	tracker.RecordCompleted(RequestTrace{
		ID:         "trace-2",
		Timestamp:  time.Now().Add(-1 * time.Second),
		Method:     "POST",
		Path:       "/api/market/buy",
		Status:     400,
		DurationMs: 120.0,
		ClientIP:   "10.0.0.2",
		PlayerID:   "user-b",
		Username:   "Bob",
		Error:      "Insufficient funds",
	})

	// 1. Filter by status errors
	resErrors := tracker.FilterTraces(FilterOptions{Status: -1})
	if resErrors.Total != 1 || resErrors.Traces[0].ID != "trace-2" {
		t.Fatalf("expected 1 error trace with ID trace-2, got %+v", resErrors)
	}

	// 2. Filter by min duration
	resSlow := tracker.FilterTraces(FilterOptions{MinDurationMs: 50.0})
	if resSlow.Total != 1 || resSlow.Traces[0].ID != "trace-2" {
		t.Fatalf("expected 1 slow trace with ID trace-2, got %+v", resSlow)
	}

	// 3. Filter by search (username)
	resSearch := tracker.FilterTraces(FilterOptions{Search: "Alice"})
	if resSearch.Total != 1 || resSearch.Traces[0].Username != "Alice" {
		t.Fatalf("expected Alice, got %+v", resSearch)
	}

	// 4. Test GetTraceByID
	trace, ok := tracker.GetTraceByID("trace-1")
	if !ok || trace.Path != "/api/inventory" {
		t.Fatalf("failed to find trace by ID trace-1")
	}

	// 5. Test Reset
	tracker.Reset()
	summaryAfterReset := tracker.GetSummary()
	if summaryAfterReset.TotalRequests != 0 {
		t.Fatalf("expected 0 requests after reset, got %d", summaryAfterReset.TotalRequests)
	}
	resEmpty := tracker.FilterTraces(FilterOptions{})
	if resEmpty.Total != 0 {
		t.Fatalf("expected 0 traces after reset, got %d", resEmpty.Total)
	}
}

func TestCPUMetrics(t *testing.T) {
	metrics := GetCPUMetrics()
	if metrics.NumCPU <= 0 {
		t.Errorf("expected NumCPU > 0, got %d", metrics.NumCPU)
	}
	if metrics.NumGoroutine <= 0 {
		t.Errorf("expected NumGoroutine > 0, got %d", metrics.NumGoroutine)
	}

	mem := GetMemoryMetrics()
	if mem.Sys == 0 {
		t.Errorf("expected Sys memory > 0")
	}
	if mem.AllocHuman == "" {
		t.Errorf("expected formatted AllocHuman string")
	}
}
