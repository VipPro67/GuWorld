package worker

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"
)

type mockReleaseConnector struct {
	queries *int32
}

func (c mockReleaseConnector) Driver() driver.Driver { return mockReleaseDriver{} }
func (c mockReleaseConnector) Connect(context.Context) (driver.Conn, error) {
	return mockReleaseConn{c.queries}, nil
}

type mockReleaseDriver struct{}

func (mockReleaseDriver) Open(string) (driver.Conn, error) {
	return mockReleaseConn{new(int32)}, nil
}

type mockReleaseConn struct {
	queries *int32
}

func (mockReleaseConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (mockReleaseConn) Close() error                        { return nil }
func (mockReleaseConn) Begin() (driver.Tx, error)           { return mockReleaseTx{}, nil }

func (c mockReleaseConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	atomic.AddInt32(c.queries, 1)
	return emptyReleaseRows{}, nil
}

func (c mockReleaseConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return driver.RowsAffected(0), nil
}

type mockReleaseTx struct{}

func (mockReleaseTx) Commit() error   { return nil }
func (mockReleaseTx) Rollback() error { return nil }

type emptyReleaseRows struct{}

func (emptyReleaseRows) Columns() []string {
	return []string{"id", "player_id", "gu_id", "gu_rank", "name", "username", "rank", "release_reason"}
}
func (emptyReleaseRows) Close() error              { return nil }
func (emptyReleaseRows) Next([]driver.Value) error { return io.EOF }

func TestStartImmortalGuReleaseWorkerIdempotent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	var queries int32
	dbConn := sql.OpenDB(mockReleaseConnector{queries: &queries})
	defer dbConn.Close()

	StartImmortalGuReleaseWorker(ctx, dbConn, 1*time.Hour)
	// Calling a second time should be a no-op
	StartImmortalGuReleaseWorker(ctx, dbConn, 1*time.Hour)
}

func TestProcessImmortalGuReleaseTickEmpty(t *testing.T) {
	ctx := context.Background()
	var queries int32
	dbConn := sql.OpenDB(mockReleaseConnector{queries: &queries})
	defer dbConn.Close()

	ProcessImmortalGuReleaseTick(ctx, dbConn)
	if atomic.LoadInt32(&queries) < 2 {
		t.Errorf("expected at least 2 queries (one for players, one for sects), got %d", atomic.LoadInt32(&queries))
	}
}

func TestReleaseStarvedOrAbandonedPlayerImmortalGusHandlesTxError(t *testing.T) {
	var queries int32
	dbConn := sql.OpenDB(mockReleaseConnector{queries: &queries})
	dbConn.Close() // closed connection

	err := releaseStarvedOrAbandonedPlayerImmortalGus(context.Background(), dbConn)
	if err == nil {
		t.Error("expected error with closed db, got nil")
	}
}

func TestReleaseAbandonedSectImmortalGusHandlesTxError(t *testing.T) {
	var queries int32
	dbConn := sql.OpenDB(mockReleaseConnector{queries: &queries})
	dbConn.Close() // closed connection

	err := releaseAbandonedSectImmortalGus(context.Background(), dbConn)
	if err == nil {
		t.Error("expected error with closed db, got nil")
	}
}

// Compile-time check for errors
var _ = errors.New
