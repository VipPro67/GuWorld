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

// probeConnector hands out connections that count queries and report an empty
// sect list, so the maintenance pass runs without touching a real database.
type probeConnector struct{ queries *int32 }

func (c probeConnector) Driver() driver.Driver { return probeDriver{} }
func (c probeConnector) Connect(context.Context) (driver.Conn, error) {
	return probeConn{c.queries}, nil
}

type probeDriver struct{}

func (probeDriver) Open(string) (driver.Conn, error) { return probeConn{new(int32)}, nil }

type probeConn struct{ queries *int32 }

func (probeConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (probeConn) Close() error                        { return nil }
func (probeConn) Begin() (driver.Tx, error)           { return nil, errors.New("no transaction") }

func (c probeConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	atomic.AddInt32(c.queries, 1)
	return emptySectRows{}, nil
}

type emptySectRows struct{}

func (emptySectRows) Columns() []string {
	return []string{"id", "level", "funds", "buildings", "last_maintenance_at"}
}
func (emptySectRows) Close() error              { return nil }
func (emptySectRows) Next([]driver.Value) error { return io.EOF }

// The daily fee is gated by sects.last_maintenance_at (rolling 24h), so a tick
// that only fires after a full interval of uptime loses a day's charge whenever
// the backend restarts first. The worker must therefore run one pass at startup.
func TestSectMaintenanceRunsAtStartupNotOnlyAfterInterval(t *testing.T) {
	var queries int32
	db := sql.OpenDB(probeConnector{queries: &queries})
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// The interval is far longer than this test: only the startup pass can fire.
	StartSectMaintenanceWorker(ctx, db, time.Hour)

	if got := atomic.LoadInt32(&queries); got == 0 {
		t.Fatal("sect maintenance did not run at startup; sects would only be charged after a full interval of uptime")
	}
}
