package handlers

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"sync"
	"testing"
)

var registerSectMembershipDriver sync.Once

type sectMembershipDriver struct{}
type sectMembershipConn struct{ dsn string }
type sectMembershipTx struct{}
type sectMembershipRows struct {
	sectID      string
	rank        string
	isDefending bool
	noRows      bool
	read        bool
}

func (sectMembershipDriver) Open(dsn string) (driver.Conn, error) {
	return &sectMembershipConn{dsn: dsn}, nil
}
func (*sectMembershipConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (*sectMembershipConn) Close() error              { return nil }
func (*sectMembershipConn) Begin() (driver.Tx, error) { return &sectMembershipTx{}, nil }
func (*sectMembershipTx) Commit() error               { return nil }
func (*sectMembershipTx) Rollback() error             { return nil }
func (c *sectMembershipConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	if c.dsn == "missing" {
		return &sectMembershipRows{noRows: true}, nil
	}
	return &sectMembershipRows{
		sectID:      c.dsn,
		rank:        "TRUONG_LAO",
		isDefending: true,
	}, nil
}
func (*sectMembershipRows) Columns() []string {
	return []string{"sect_id", "member_rank", "is_defending"}
}
func (*sectMembershipRows) Close() error { return nil }
func (r *sectMembershipRows) Next(values []driver.Value) error {
	if r.noRows || r.read {
		return io.EOF
	}
	r.read = true
	values[0] = r.sectID
	values[1] = r.rank
	values[2] = r.isDefending
	return nil
}

func sectMembershipDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	registerSectMembershipDriver.Do(func() {
		sql.Register("sect-membership-test", sectMembershipDriver{})
	})
	database, err := sql.Open("sect-membership-test", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func TestLoadSectMembershipReadsSectRankAndDefending(t *testing.T) {
	database := sectMembershipDB(t, "sect-a")
	m, err := LoadSectMembership(database, "player-1")
	if err != nil {
		t.Fatal(err)
	}
	if m.SectID != "sect-a" || m.Rank != "TRUONG_LAO" || !m.IsDefending {
		t.Fatalf("membership = %+v, want sect-a/TRUONG_LAO/defending", m)
	}
}

func TestLoadSectMembershipReturnsNoRowsForNonMember(t *testing.T) {
	database := sectMembershipDB(t, "missing")
	if _, err := LoadSectMembership(database, "player-1"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("error = %v, want sql.ErrNoRows", err)
	}
}

func TestLoadSectIDReturnsOnlyTheSect(t *testing.T) {
	database := sectMembershipDB(t, "sect-b")
	sectID, err := LoadSectID(database, "player-1")
	if err != nil {
		t.Fatal(err)
	}
	if sectID != "sect-b" {
		t.Fatalf("sect id = %q, want sect-b", sectID)
	}
}

func TestLoadSectMembershipContextReadsMembership(t *testing.T) {
	database := sectMembershipDB(t, "sect-c")
	m, err := LoadSectMembershipContext(context.Background(), database, "player-1")
	if err != nil {
		t.Fatal(err)
	}
	if m.SectID != "sect-c" {
		t.Fatalf("sect id = %q, want sect-c", m.SectID)
	}
}
