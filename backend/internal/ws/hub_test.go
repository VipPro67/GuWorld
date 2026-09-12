package ws

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHub_SubscribeAndBroadcast(t *testing.T) {
	hub := InitHub()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = hub.Stop(ctx)
	}()

	playerID := uuid.New()
	client := &Client{
		playerID:      playerID,
		send:          make(chan []byte, 10),
		hub:           hub,
		subscriptions: make(map[string]struct{}),
	}

	hub.register <- client
	time.Sleep(20 * time.Millisecond)

	hub.Subscribe(client, ChannelName("chat:world"))
	time.Sleep(20 * time.Millisecond)

	// Broadcast message to chat:world
	testPayload := []byte(`{"type":"event","channel":"chat:world","data":"hello"}`)
	hub.broadcast <- &BroadcastMsg{
		Channel: ChannelName("chat:world"),
		Payload: testPayload,
	}

	select {
	case msg := <-client.send:
		if string(msg) != string(testPayload) {
			t.Fatalf("expected payload %s, got %s", string(testPayload), string(msg))
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for broadcast message")
	}

	// Broadcast message to a different channel
	otherPayload := []byte(`{"type":"event","channel":"chat:sect","data":"sect message"}`)
	hub.broadcast <- &BroadcastMsg{
		Channel: ChannelName("chat:sect"),
		Payload: otherPayload,
	}

	select {
	case msg := <-client.send:
		t.Fatalf("unexpected message received: %s", string(msg))
	case <-time.After(100 * time.Millisecond):
		// Expected: no message
	}

	// Unregister client
	hub.unregister <- client
	time.Sleep(20 * time.Millisecond)

	hub.mu.RLock()
	if _, ok := hub.clients[playerID]; ok {
		t.Fatal("client should have been unregistered")
	}
	hub.mu.RUnlock()
}

func TestHub_IsConnected(t *testing.T) {
	hub := InitHub()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = hub.Stop(ctx)
	}()

	playerID := uuid.New()
	if hub.IsConnected(playerID) {
		t.Fatal("expected player to not be connected initially")
	}

	client := &Client{
		playerID:      playerID,
		send:          make(chan []byte, 10),
		hub:           hub,
		subscriptions: make(map[string]struct{}),
	}

	hub.register <- client
	time.Sleep(20 * time.Millisecond)

	if !hub.IsConnected(playerID) {
		t.Fatal("expected player to be connected after registration")
	}

	hub.unregister <- client
	time.Sleep(20 * time.Millisecond)

	if hub.IsConnected(playerID) {
		t.Fatal("expected player to not be connected after unregistration")
	}
}

func TestIsPlayerScopedChannel(t *testing.T) {
	for _, channel := range []string{"player:abc", "combat:abc", "trade:abc", "notif:abc"} {
		if !isPlayerScopedChannel(channel) {
			t.Fatalf("expected %q to be player-scoped", channel)
		}
	}
	for _, channel := range []string{"player", "combat", "chat:world"} {
		if isPlayerScopedChannel(channel) {
			t.Fatalf("expected %q not to be player-scoped", channel)
		}
	}
}

func TestShardedHub_TradeEventsStayPlayerScoped(t *testing.T) {
	hub := InitShardedHub()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = hub.Stop(ctx)
	}()

	firstPlayerID := uuid.New()
	secondPlayerID := uuid.New()
	firstClient := newTestClient(hub, firstPlayerID)
	secondClient := newTestClient(hub, secondPlayerID)
	hub.Register(firstClient)
	hub.Register(secondClient)
	time.Sleep(20 * time.Millisecond)

	firstTradeChannel := ChannelName("trade:" + firstPlayerID.String())
	secondTradeChannel := ChannelName("trade:" + secondPlayerID.String())
	hub.Subscribe(firstClient, firstTradeChannel)
	hub.Subscribe(secondClient, secondTradeChannel)

	payload := []byte(`{"type":"event","channel":"trade","data":{"trade":{"id":"trade-1"}}}`)
	hub.BroadcastToChannel(firstTradeChannel, payload, uuid.Nil)

	select {
	case message := <-firstClient.send:
		if string(message) != string(payload) {
			t.Fatalf("expected payload %s, got %s", string(payload), string(message))
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for trade event")
	}

	select {
	case message := <-secondClient.send:
		t.Fatalf("unexpected trade event received by another player: %s", string(message))
	case <-time.After(100 * time.Millisecond):
	}
}

func newTestClient(hub *ShardedHub, playerID uuid.UUID) *Client {
	return &Client{
		playerID:      playerID,
		send:          make(chan []byte, 10),
		done:          make(chan struct{}),
		hub:           hub.GetClientShard(playerID),
		shard:         hub,
		subscriptions: make(map[string]struct{}),
	}
}

func TestShardedHub_UnregisterAndBroadcast(t *testing.T) {
	shub := InitShardedHub()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = shub.Stop(ctx)
	}()

	playerID := uuid.New()
	client := &Client{
		playerID:      playerID,
		send:          make(chan []byte, 10),
		done:          make(chan struct{}),
		hub:           shub.GetClientShard(playerID),
		shard:         shub,
		subscriptions: make(map[string]struct{}),
	}

	shub.Register(client)
	time.Sleep(20 * time.Millisecond)

	channels := []ChannelName{
		"player:" + ChannelName(playerID.String()),
		"world:pos",
		"world:cell",
		"chat:world",
		"trade:" + ChannelName(playerID.String()),
	}

	for _, ch := range channels {
		shub.Subscribe(client, ch)
	}
	time.Sleep(20 * time.Millisecond)

	// Unregister client
	shub.Unregister(client)
	time.Sleep(50 * time.Millisecond)

	// Broadcasting to any of these channels should NEVER panic even though client disconnected
	for _, ch := range channels {
		payload := []byte(`{"type":"event","channel":"` + string(ch) + `","data":"test"}`)
		shub.BroadcastToChannel(ch, payload, uuid.Nil)
	}

	// Verify client is not connected
	if shub.IsConnected(playerID) {
		t.Fatal("expected player to be disconnected after unregistration")
	}

	// Verify all subscriptions were cleaned up
	for _, ch := range channels {
		shard := shub.GetChannelShard(ch)
		shard.mu.RLock()
		if subs, ok := shard.channels[ch]; ok {
			if _, exists := subs[playerID]; exists {
				t.Fatalf("expected player to be removed from channel %s", ch)
			}
		}
		shard.mu.RUnlock()
	}
}
