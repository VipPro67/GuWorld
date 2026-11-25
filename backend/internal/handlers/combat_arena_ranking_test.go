package handlers

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

var (
	registerArenaRankDriver sync.Once
	arenaRankSinkRef        *arenaRankTestSink
)

type arenaRankTestSink struct {
	mu    sync.Mutex
	execs []string
}

func (s *arenaRankTestSink) record(query string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.execs = append(s.execs, query)
}

func (s *arenaRankTestSink) count() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.execs)
}

type arenaRankDriver struct{}
type arenaRankConn struct {
	ranks []int
	query int
	sink  *arenaRankTestSink
}
type arenaRankTx struct{}
type arenaRankRows struct {
	rank int
	read bool
}

func (arenaRankDriver) Open(dsn string) (driver.Conn, error) {
	parts := strings.Split(dsn, ",")
	ranks := make([]int, 0, len(parts))
	for _, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, err
		}
		ranks = append(ranks, n)
	}
	return &arenaRankConn{ranks: ranks, sink: arenaRankSinkRef}, nil
}

func (*arenaRankConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (*arenaRankConn) Close() error              { return nil }
func (*arenaRankConn) Begin() (driver.Tx, error) { return arenaRankTx{}, nil }
func (c *arenaRankConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	rank := 0
	if c.query < len(c.ranks) {
		rank = c.ranks[c.query]
	}
	c.query++
	return &arenaRankRows{rank: rank}, nil
}
func (c *arenaRankConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	c.sink.record(query)
	return driver.RowsAffected(1), nil
}
func (arenaRankTx) Commit() error        { return nil }
func (arenaRankTx) Rollback() error      { return nil }
func (*arenaRankRows) Columns() []string { return []string{"rank"} }
func (*arenaRankRows) Close() error      { return nil }
func (r *arenaRankRows) Next(values []driver.Value) error {
	if r.read {
		return io.EOF
	}
	r.read = true
	values[0] = int64(r.rank)
	return nil
}

func arenaRankTxFor(t *testing.T, ranks string, sink *arenaRankTestSink) *sql.Tx {
	t.Helper()
	registerArenaRankDriver.Do(func() {
		sql.Register("arena-rank-test", arenaRankDriver{})
	})
	arenaRankSinkRef = sink
	database, err := sql.Open("arena-rank-test", ranks)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	tx, err := database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })
	return tx
}

func TestSettleArenaRankSwapSwapsWhenPlayerIsWorse(t *testing.T) {
	sink := &arenaRankTestSink{}
	tx := arenaRankTxFor(t, "5,2", sink)

	log := settleArenaRankSwap(tx, uuid.New(), uuid.New(), 7)
	if log == nil {
		t.Fatal("expected a rank-up log when a worse-ranked player wins")
	}
	if log.Action != "ARENA_RANK_UP" || log.Turn != 7 {
		t.Fatalf("log = %+v, want ARENA_RANK_UP at turn 7", log)
	}
	if !strings.Contains(log.Detail, "Hạng 2") {
		t.Fatalf("detail = %q, want the target rank", log.Detail)
	}
	if got := sink.count(); got != 5 {
		t.Fatalf("exec count = %d, want 5 rank writes", got)
	}
}

func TestSettleArenaRankSwapKeepsRanksWhenPlayerIsBetter(t *testing.T) {
	sink := &arenaRankTestSink{}
	tx := arenaRankTxFor(t, "1,4", sink)

	if log := settleArenaRankSwap(tx, uuid.New(), uuid.New(), 3); log != nil {
		t.Fatalf("expected no rank-up log, got %+v", log)
	}
	if got := sink.count(); got != 0 {
		t.Fatalf("exec count = %d, want 0 writes", got)
	}
}

func TestSettleArenaRankSwapKeepsRanksWhenEqual(t *testing.T) {
	sink := &arenaRankTestSink{}
	tx := arenaRankTxFor(t, "5,5", sink)

	if log := settleArenaRankSwap(tx, uuid.New(), uuid.New(), 1); log != nil {
		t.Fatalf("expected no rank-up log for equal ranks, got %+v", log)
	}
	if got := sink.count(); got != 0 {
		t.Fatalf("exec count = %d, want 0 writes", got)
	}
}

func TestSettleCombatRequiresTransaction(t *testing.T) {
	if _, err := settleCombat(nil, &models.Player{}, nil, nil, nil, nil, nil); err == nil || !strings.Contains(err.Error(), "transaction is required") {
		t.Fatalf("error = %v, want a transaction-required error", err)
	}
}

func TestSettleCombatRequiresActiveCombat(t *testing.T) {
	if _, err := settleCombat(&sql.Tx{}, &models.Player{}, nil, nil, nil, nil, nil); err == nil || !strings.Contains(err.Error(), "no active combat") {
		t.Fatalf("error = %v, want a no-active-combat error", err)
	}
}
