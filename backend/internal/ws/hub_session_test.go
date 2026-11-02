package ws

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newSessionTestClient(hub *ShardedHub, playerID uuid.UUID, kind string) *Client {
	return &Client{
		playerID:      playerID,
		kind:          kind,
		send:          make(chan []byte, 10),
		done:          make(chan struct{}),
		hub:           hub.GetClientShard(playerID),
		shard:         hub,
		subscriptions: make(map[string]struct{}),
	}
}

func waitFor(t *testing.T, description string, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", description)
}

func isClosed(c *Client) bool {
	select {
	case <-c.done:
		return true
	default:
		return false
	}
}

// A player who reconnects (new tab, new network path) must not be blocked by the
// previous half-open world socket: the new world session takes over and the old
// one is closed so it releases the character lease.
func TestHub_WorldTakeoverClosesPreviousConnection(t *testing.T) {
	shub := InitShardedHub()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = shub.Stop(ctx)
	}()

	playerID := uuid.New()
	first := newSessionTestClient(shub, playerID, clientKindWorld)
	second := newSessionTestClient(shub, playerID, clientKindWorld)
	admission := &fakeRealtimeAdmission{}
	first.realtimeAdmission = admission

	shub.Register(first)
	waitFor(t, "first world client registered", func() bool { return shub.IsConnected(playerID) })

	shub.Register(second)
	waitFor(t, "previous world client closed by takeover", func() bool { return isClosed(first) })
	// Closing the superseded socket is what releases the character lease; that
	// is the acceptance criterion, not merely "the goroutine stopped".
	waitFor(t, "superseded controller disconnected from the lease", func() bool {
		return admission.disconnectCount() == 1
	})
	waitFor(t, "replacement world client registered", func() bool {
		shard := shub.GetClientShard(playerID)
		shard.mu.RLock()
		defer shard.mu.RUnlock()
		return shard.clients[clientKey{playerID: playerID, kind: clientKindWorld}] == second
	})

	// The stale connection's own unregister must not clear the replacement.
	shub.Unregister(first)
	waitFor(t, "stale unregister processed", func() bool {
		shard := shub.GetClientShard(playerID)
		shard.mu.RLock()
		defer shard.mu.RUnlock()
		return shard.clients[clientKey{playerID: playerID, kind: clientKindWorld}] == second
	})
	if !shub.IsConnected(playerID) {
		t.Fatal("player should stay connected through the replacement world socket")
	}
}

// Presence must count live sockets, not the session slot, so closing one of a
// player's two sockets does not flap IsConnected while the other is still open.
func TestHub_PresenceSurvivesClosingOneOfTwoSockets(t *testing.T) {
	shub := InitShardedHub()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = shub.Stop(ctx)
	}()

	playerID := uuid.New()
	ui := newSessionTestClient(shub, playerID, clientKindUI)
	world := newSessionTestClient(shub, playerID, clientKindWorld)

	shub.Register(ui)
	shub.Register(world)
	waitFor(t, "both sockets registered", func() bool {
		shard := shub.GetClientShard(playerID)
		shard.mu.RLock()
		defer shard.mu.RUnlock()
		return len(shard.presence[playerID]) == 2
	})

	shub.Unregister(ui)
	waitFor(t, "UI socket unregistered", func() bool {
		shard := shub.GetClientShard(playerID)
		shard.mu.RLock()
		defer shard.mu.RUnlock()
		return len(shard.presence[playerID]) == 1
	})
	if !shub.IsConnected(playerID) {
		t.Fatal("player must stay connected while the world socket is still open")
	}

	shub.Unregister(world)
	waitFor(t, "world socket unregistered", func() bool { return !shub.IsConnected(playerID) })
}

// UI sockets are intentionally not exclusive: multiple tabs may hold one, and
// evicting them would make two open tabs kick each other in a reconnect loop.
func TestHub_UITakeoverDoesNotEvict(t *testing.T) {
	shub := InitShardedHub()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = shub.Stop(ctx)
	}()

	playerID := uuid.New()
	first := newSessionTestClient(shub, playerID, clientKindUI)
	second := newSessionTestClient(shub, playerID, clientKindUI)

	shub.Register(first)
	waitFor(t, "first UI client registered", func() bool { return shub.IsConnected(playerID) })
	shub.Register(second)
	waitFor(t, "both UI clients registered", func() bool {
		shard := shub.GetClientShard(playerID)
		shard.mu.RLock()
		defer shard.mu.RUnlock()
		return len(shard.presence[playerID]) == 2
	})
	if isClosed(first) {
		t.Fatal("registering a second UI socket must not close the first one")
	}
}
