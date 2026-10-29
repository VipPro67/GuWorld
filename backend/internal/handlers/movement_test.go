package handlers

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

var registerMovementPolicyDriver sync.Once

type movementPolicyDriver struct{}
type movementPolicyConn struct{ isLeader bool }
type movementPolicyTx struct{}
type movementPolicyRows struct {
	isLeader bool
	read     bool
}

func (movementPolicyDriver) Open(name string) (driver.Conn, error) {
	return &movementPolicyConn{isLeader: name == "leader"}, nil
}
func (*movementPolicyConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (*movementPolicyConn) Close() error              { return nil }
func (*movementPolicyConn) Begin() (driver.Tx, error) { return movementPolicyTx{}, nil }
func (c *movementPolicyConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return &movementPolicyRows{isLeader: c.isLeader}, nil
}
func (movementPolicyTx) Commit() error        { return nil }
func (movementPolicyTx) Rollback() error      { return nil }
func (*movementPolicyRows) Columns() []string { return []string{"is_leader"} }
func (*movementPolicyRows) Close() error      { return nil }
func (r *movementPolicyRows) Next(values []driver.Value) error {
	if r.read {
		return io.EOF
	}
	r.read = true
	values[0] = r.isLeader
	return nil
}

func movementPolicyTxFor(t *testing.T, name string) *sql.Tx {
	t.Helper()
	registerMovementPolicyDriver.Do(func() {
		sql.Register("movement-policy-test", movementPolicyDriver{})
	})
	database, err := sql.Open("movement-policy-test", name)
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

func movementTestPlayer() *models.Player {
	player := &models.Player{}
	player.HP = 100
	player.Lifespan = 100
	player.Rank = 1
	player.Gender = "male"
	return player
}

func TestValidateMovementState(t *testing.T) {
	cases := []struct {
		name       string
		mutate     func(*models.Player)
		spec       movementSpec
		owned      bool
		wantStatus int
		wantCode   string
	}{
		{
			name:       "realtime ownership wins",
			mutate:     func(p *models.Player) { p.Lifespan = 0 },
			spec:       movementSpec{hasTarget: true, x: -1, y: -1},
			owned:      true,
			wantStatus: http.StatusConflict,
			wantCode:   "realtime_world_owns_character",
		},
		{
			name:       "bounds before alive checks",
			spec:       movementSpec{hasTarget: true, x: -1, y: 0},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "dead state is forbidden",
			mutate:     func(p *models.Player) { p.Lifespan = 0 },
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "wounded cannot move",
			mutate:     func(p *models.Player) { p.HP = 0 },
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "healthy player passes",
			wantStatus: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			player := movementTestPlayer()
			if tc.mutate != nil {
				tc.mutate(player)
			}
			err := validateMovementState(player, tc.spec, tc.owned)
			if tc.wantStatus == 0 {
				if err != nil {
					t.Fatalf("validateMovementState = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("validateMovementState = nil, want a movement error")
			}
			if err.status != tc.wantStatus {
				t.Fatalf("status = %d, want %d (%v)", err.status, tc.wantStatus, err)
			}
			if tc.wantCode != "" && err.code != tc.wantCode {
				t.Fatalf("code = %q, want %q", err.code, tc.wantCode)
			}
		})
	}
}

func TestCheckMovementUsesTransactionPartyLeadership(t *testing.T) {
	for _, test := range []struct {
		name     string
		isLeader bool
		wantErr  bool
	}{
		{name: "leader can move party", isLeader: true},
		{name: "member cannot move party", wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			player := movementTestPlayer()
			player.ID = uuid.New()
			partyID := uuid.New()
			player.PartyID = &partyID
			connection := "member"
			if test.isLeader {
				connection = "leader"
			}
			err := checkMovement(movementPolicyTxFor(t, connection), player, movementSpec{kind: movementWalk})
			if (err != nil) != test.wantErr {
				t.Fatalf("checkMovement error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestMovementPolicyTableCoversEveryKind(t *testing.T) {
	kinds := []movementKind{
		movementWalk,
		movementPortal,
		movementTeleport,
		movementGuTeleport,
		movementPartyJoin,
	}
	for _, kind := range kinds {
		if _, ok := movementPolicies[kind]; !ok {
			t.Fatalf("movement kind %d has no policy: add it to movementPolicies", kind)
		}
	}
	if !movementPolicies[movementWalk].requirePartyLeader {
		t.Fatal("walk must stay party-leader only")
	}
	if movementPolicies[movementWalk].requireRegionEntry {
		t.Fatal("walk resolves its own region per step; the chain must not pre-check entry")
	}
	if got := movementPolicies[movementPortal].regionEntryStatus; got != http.StatusBadRequest {
		t.Fatalf("portal entry status = %d, want 400", got)
	}
	if got := movementPolicies[movementGuTeleport].regionEntryStatus; got != http.StatusForbidden {
		t.Fatalf("Gu teleport entry status = %d, want 403", got)
	}
	if movementPolicies[movementTeleport].targetAction != "teleport" {
		t.Fatal("teleport arrays must keep the destination teleport ban")
	}
	if got := movementPolicies[movementTeleport].actionStatus; got != http.StatusBadRequest {
		t.Fatalf("teleport action status = %d, want 400", got)
	}
	if !movementPolicies[movementPartyJoin].requirePartyLeader && movementPolicies[movementPartyJoin].requireRegionEntry {
		t.Fatal("party join must not require region entry")
	}
}

func TestValidateMovementTarget(t *testing.T) {
	if err := validateMovementTarget(movementSpec{hasTarget: true, x: 0, y: 0}); err != nil {
		t.Fatalf("origin should be walkable: %v", err)
	}
	if err := validateMovementTarget(movementSpec{hasTarget: true, x: -1, y: 0}); err == nil {
		t.Fatal("negative x should be rejected")
	}
	if err := validateMovementTarget(movementSpec{hasTarget: false, x: -5, y: -5}); err != nil {
		t.Fatalf("server-chosen destination should skip the bounds check: %v", err)
	}
}

func TestMovementErrorPayload(t *testing.T) {
	status, payload, ok := movementErrorPayload(&movementError{
		status:  http.StatusConflict,
		code:    "realtime_world_owns_character",
		message: "Character is controlled by the realtime world",
	})
	if !ok || status != http.StatusConflict {
		t.Fatalf("payload = (%d, %v, %v), want 409", status, payload, ok)
	}
	if payload["code"] != "realtime_world_owns_character" {
		t.Fatalf("payload code = %v", payload["code"])
	}
	if _, _, ok := movementErrorPayload(nil); ok {
		t.Fatal("nil error should not produce a movement payload")
	}
}
