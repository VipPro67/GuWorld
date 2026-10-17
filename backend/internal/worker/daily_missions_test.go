package worker

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"sync/atomic"
	"testing"
	"time"
)

type dummyConnector struct{ queries *int32 }

func (c dummyConnector) Driver() driver.Driver { return dummyDriver{} }
func (c dummyConnector) Connect(context.Context) (driver.Conn, error) {
	return dummyConn{c.queries}, nil
}

type dummyDriver struct{}

func (dummyDriver) Open(string) (driver.Conn, error) { return dummyConn{new(int32)}, nil }

type dummyConn struct{ queries *int32 }

func (dummyConn) Prepare(string) (driver.Stmt, error) { return dummyStmt{}, nil }
func (dummyConn) Close() error                        { return nil }
func (dummyConn) Begin() (driver.Tx, error)           { return dummyTx{}, nil }

func (c dummyConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	atomic.AddInt32(c.queries, 1)
	return emptyDailyMissionRows{}, nil
}

func (c dummyConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	atomic.AddInt32(c.queries, 1)
	return dummyResult{}, nil
}

type dummyStmt struct{}

func (dummyStmt) Close() error                                    { return nil }
func (dummyStmt) NumInput() int                                   { return -1 }
func (dummyStmt) Exec([]driver.Value) (driver.Result, error)      { return dummyResult{}, nil }
func (dummyStmt) Query([]driver.Value) (driver.Rows, error)       { return emptyDailyMissionRows{}, nil }

type dummyTx struct{}

func (dummyTx) Commit() error   { return nil }
func (dummyTx) Rollback() error { return nil }

type dummyResult struct{}

func (dummyResult) LastInsertId() (int64, error) { return 0, nil }
func (dummyResult) RowsAffected() (int64, error) { return 0, nil }

type emptyDailyMissionRows struct{}

func (emptyDailyMissionRows) Columns() []string              { return []string{"day_key"} }
func (emptyDailyMissionRows) Close() error                  { return nil }
func (emptyDailyMissionRows) Next([]driver.Value) error     { return io.EOF }

func TestStartDailyMissionRolloverWorkerNonBlocking(t *testing.T) {
	var queries int32
	db := sql.OpenDB(dummyConnector{queries: &queries})
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		StartDailyMissionRolloverWorker(ctx, db, time.Hour)
		close(done)
	}()

	select {
	case <-done:
		// Succeeded immediately without blocking caller
	case <-time.After(100 * time.Millisecond):
		t.Fatal("StartDailyMissionRolloverWorker blocked the caller")
	}
}
