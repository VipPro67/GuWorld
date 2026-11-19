package auditlog_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type mockQuerier struct {
	execCalled bool
	lastQuery  string
	lastArgs   []interface{}
	returnErr  error
}

func TestLogContextPopulatesAndPreservesRequestID(t *testing.T) {
	ctx := logger.WithTraceID(context.Background(), "trace-ctx")
	querier := &mockQuerier{}
	auditlog.LogContext(ctx, querier, auditlog.Entry{Action: "player.test", Success: true})
	if got := querier.lastArgs[10]; got != "trace-ctx" {
		t.Fatalf("expected request ID from context, got %v", got)
	}

	querier = &mockQuerier{}
	auditlog.LogContext(ctx, querier, auditlog.Entry{Action: "player.test", RequestID: "operation-42", Success: true})
	if got := querier.lastArgs[10]; got != "operation-42" {
		t.Fatalf("expected explicit request ID to take precedence, got %v", got)
	}
}

func (m *mockQuerier) Exec(query string, args ...interface{}) (sql.Result, error) {
	m.execCalled = true
	m.lastQuery = query
	m.lastArgs = args
	return nil, m.returnErr
}

func TestLog_UsesPassedQuerier(t *testing.T) {
	mock := &mockQuerier{}
	pID := uuid.New()

	auditlog.Log(mock, auditlog.Entry{
		PlayerID:       &pID,
		Action:         "gu.starve_death",
		EntityType:     "gu_instance",
		EntityID:       "test-id",
		RequestSummary: "Test summary",
		Success:        true,
	})

	if !mock.execCalled {
		t.Fatalf("expected Exec to be called on passed Querier")
	}

	if len(mock.lastArgs) < 16 {
		t.Fatalf("expected at least 16 args, got %d", len(mock.lastArgs))
	}

	if mock.lastArgs[0] != pID {
		t.Errorf("expected player ID %v, got %v", pID, mock.lastArgs[0])
	}
	if mock.lastArgs[3] != "gu.starve_death" {
		t.Errorf("expected action gu.starve_death, got %v", mock.lastArgs[3])
	}
}

func TestLogf_UsesPassedQuerier(t *testing.T) {
	mock := &mockQuerier{}
	pID := uuid.New()

	auditlog.Logf(mock, &pID, "admin.test_action", "item_id=%s", "item-123")

	if !mock.execCalled {
		t.Fatalf("expected Exec to be called on passed Querier")
	}

	if mock.lastArgs[3] != "admin.test_action" {
		t.Errorf("expected action admin.test_action, got %v", mock.lastArgs[3])
	}
	if mock.lastArgs[11] != "item_id=item-123" {
		t.Errorf("expected summary item_id=item-123, got %v", mock.lastArgs[11])
	}
}

func TestLog_NilQuerierDoesNotPanic(t *testing.T) {
	pID := uuid.New()
	// Calling with nil querier when db.DB is nil should return gracefully without panic
	auditlog.Log(nil, auditlog.Entry{
		PlayerID: &pID,
		Action:   "nil_test",
	})
}

func TestLog_ExecErrorHandled(t *testing.T) {
	mock := &mockQuerier{returnErr: errors.New("db error")}
	pID := uuid.New()

	auditlog.Log(mock, auditlog.Entry{
		PlayerID: &pID,
		Action:   "err_test",
	})

	if !mock.execCalled {
		t.Fatalf("expected Exec to be called")
	}
}
