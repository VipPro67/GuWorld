package db

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/lib/pq"
)

type schemaRetryExecutor struct {
	failures int
	calls    int
}

func (e *schemaRetryExecutor) Exec(string, ...interface{}) (sql.Result, error) {
	e.calls++
	if e.calls <= e.failures {
		return nil, &pq.Error{Code: "55P03", Message: "canceling statement due to lock timeout"}
	}
	return nil, nil
}
func (*schemaRetryExecutor) Query(string, ...interface{}) (*sql.Rows, error) {
	return nil, errors.New("unexpected Query")
}
func (*schemaRetryExecutor) QueryRow(string, ...interface{}) *sql.Row { return nil }

func TestExecSchemaRetriesLockTimeout(t *testing.T) {
	executor := &schemaRetryExecutor{failures: 2}
	var waits []time.Duration
	err := execSchemaWithRetry(executor, "locked_table", "ALTER TABLE locked_table ADD COLUMN value int", 3, time.Millisecond, func(wait time.Duration) {
		waits = append(waits, wait)
	})
	if err != nil {
		t.Fatalf("execSchemaWithRetry() error = %v", err)
	}
	if executor.calls != 3 {
		t.Fatalf("Exec calls = %d, want 3", executor.calls)
	}
	if len(waits) != 2 || waits[0] != time.Millisecond || waits[1] != 2*time.Millisecond {
		t.Fatalf("retry waits = %v, want [1ms 2ms]", waits)
	}
}

type permanentSchemaExecutor struct{ calls int }

func (e *permanentSchemaExecutor) Exec(string, ...interface{}) (sql.Result, error) {
	e.calls++
	return nil, errors.New("syntax error")
}
func (*permanentSchemaExecutor) Query(string, ...interface{}) (*sql.Rows, error) {
	return nil, errors.New("unexpected Query")
}
func (*permanentSchemaExecutor) QueryRow(string, ...interface{}) *sql.Row { return nil }

func TestExecSchemaDoesNotRetryPermanentError(t *testing.T) {
	executor := &permanentSchemaExecutor{}
	err := execSchemaWithRetry(executor, "bad_sql", "invalid", 3, time.Millisecond, func(time.Duration) {
		t.Fatal("permanent error must not sleep")
	})
	if err == nil {
		t.Fatal("execSchemaWithRetry() error = nil, want permanent error")
	}
	if executor.calls != 1 {
		t.Fatalf("Exec calls = %d, want 1", executor.calls)
	}
}

func TestMigrationTimeoutMsRejectsNegativeValue(t *testing.T) {
	t.Setenv("TEST_MIGRATION_TIMEOUT_MS", "-1")
	if got := migrationTimeoutMs("TEST_MIGRATION_TIMEOUT_MS", "1234"); got != "1234" {
		t.Fatalf("migrationTimeoutMs() = %q, want fallback %q", got, "1234")
	}
}

func TestCappedRetryDelayDoesNotOverflow(t *testing.T) {
	if got := cappedRetryDelay(time.Duration(1<<62), 1000); got != 10*time.Second {
		t.Fatalf("cappedRetryDelay() = %s, want 10s", got)
	}
}

func TestMigrationPositiveIntRejectsValuesAboveLimit(t *testing.T) {
	t.Setenv("TEST_MIGRATION_RETRIES", "999999999")
	if got := migrationPositiveInt("TEST_MIGRATION_RETRIES", 6, 20); got != 6 {
		t.Fatalf("migrationPositiveInt() = %d, want fallback 6", got)
	}
}
