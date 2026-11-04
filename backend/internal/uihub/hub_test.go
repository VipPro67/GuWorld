package uihub

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func waitForPayload(t *testing.T, ch <-chan []byte) []byte {
	t.Helper()
	select {
	case got := <-ch:
		return append([]byte(nil), got...)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for sink payload")
		return nil
	}
}

func TestHub_SinkSubscribeBroadcastAndUnsubscribe(t *testing.T) {
	hub := InitHub()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = hub.Stop(ctx)
	}()

	playerA := uuid.New()
	playerB := uuid.New()
	receivedA := make(chan []byte, 4)
	receivedB := make(chan []byte, 4)

	hub.SubscribeSink(playerA, ChannelName("chat:world"), SinkFunc(func(p []byte) { receivedA <- p }))
	hub.SubscribeSink(playerB, ChannelName("chat:world"), SinkFunc(func(p []byte) { receivedB <- p }))
	hub.SubscribeSink(playerA, ChannelName("notif:"+playerA.String()), SinkFunc(func(p []byte) { receivedA <- p }))

	if !hub.IsConnected(playerA) {
		t.Fatal("expected subscribed player to report IsConnected=true")
	}

	msg := []byte(`{"type":"event","channel":"chat:world","data":{"text":"hi"}}`)
	hub.BroadcastToChannel(ChannelName("chat:world"), msg, playerB)
	if got := waitForPayload(t, receivedA); string(got) != string(msg) {
		t.Fatalf("got %s, want %s", got, msg)
	}
	select {
	case got := <-receivedB:
		t.Fatalf("excluded player received %s", got)
	case <-time.After(100 * time.Millisecond):
	}

	hub.UnsubscribeSink(playerA, ChannelName("chat:world"))
	hub.BroadcastToChannel(ChannelName("chat:world"), msg, uuid.Nil)
	select {
	case got := <-receivedA:
		t.Fatalf("unsubscribed player received %s", got)
	case <-time.After(100 * time.Millisecond):
	}
	if !hub.IsConnected(playerA) {
		t.Fatal("player still subscribed to notification channel")
	}
	hub.UnsubscribeAllSinks(playerA)
	if hub.IsConnected(playerA) {
		t.Fatal("expected IsConnected=false after UnsubscribeAllSinks")
	}
}

func TestShardedHub_TargetsAndStats(t *testing.T) {
	shub := InitShardedHub()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = shub.Stop(ctx)
	}()

	playerA := uuid.New()
	playerB := uuid.New()
	receivedA := make(chan []byte, 4)
	receivedB := make(chan []byte, 4)
	shub.SubscribeSink(playerA, ChannelName("world:events"), SinkFunc(func(p []byte) { receivedA <- p }))
	shub.SubscribeSink(playerB, ChannelName("world:events"), SinkFunc(func(p []byte) { receivedB <- p }))

	targets := map[uuid.UUID]struct{}{playerA: {}}
	msg := []byte(`{"type":"event","channel":"world:events","data":{}}`)
	shub.BroadcastToTargets(ChannelName("world:events"), msg, targets)
	if got := waitForPayload(t, receivedA); string(got) != string(msg) {
		t.Fatalf("got %s, want %s", got, msg)
	}
	select {
	case got := <-receivedB:
		t.Fatalf("non-target player received %s", got)
	case <-time.After(100 * time.Millisecond):
	}

	clients, channels := shub.GetStats()
	if clients != 2 {
		t.Fatalf("connected clients = %d, want 2", clients)
	}
	if channels != 1 {
		t.Fatalf("active channels = %d, want 1", channels)
	}
}

func TestFormatClientEvent_ScopesPrivateVsGlobalChannels(t *testing.T) {
	playerID := uuid.New()
	for _, tc := range []struct {
		redisCh       string
		wantClientCh  string
		wantBroadcast bool
	}{
		{redisCh: "player:" + playerID.String(), wantClientCh: "player", wantBroadcast: false},
		{redisCh: "refine:" + playerID.String(), wantClientCh: "refine", wantBroadcast: false},
		{redisCh: "trade:" + playerID.String(), wantClientCh: "trade", wantBroadcast: false},
		{redisCh: "notif:" + playerID.String(), wantClientCh: "notification", wantBroadcast: false},
		{redisCh: "combat:" + playerID.String(), wantClientCh: "combat", wantBroadcast: false},
		{redisCh: "death_arena:match-1", wantClientCh: "death_arena", wantBroadcast: true},
		{redisCh: "martial_arena:match-2", wantClientCh: "martial_arena", wantBroadcast: true},
		{redisCh: "catalog:items", wantClientCh: "catalog", wantBroadcast: true},
		{redisCh: "chat:world", wantClientCh: "chat:world", wantBroadcast: false},
	} {
		clientCh, _, broadcastNormalized := formatClientEvent(tc.redisCh, []byte(`{"ok":true}`))
		if clientCh != tc.wantClientCh || broadcastNormalized != tc.wantBroadcast {
			t.Fatalf("formatClientEvent(%q) = (%q, %v), want (%q, %v)",
				tc.redisCh, clientCh, broadcastNormalized, tc.wantClientCh, tc.wantBroadcast)
		}
	}
}
