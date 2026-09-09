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
