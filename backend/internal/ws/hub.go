package ws

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// ChannelName provides stronger typing for channel keys.
type ChannelName string

// BroadcastMsg is the envelope used internally by the Hub to send a message to subscribers.
type BroadcastMsg struct {
	Channel ChannelName
	Payload []byte
	// Exclude is the player ID to skip (e.g., the sender).
	Exclude uuid.UUID
	// Targets restricts broadcast to only these player IDs.
	// If nil/empty, broadcasts to all subscribers of the channel.
	Targets map[uuid.UUID]struct{}
}

// Hub manages WebSocket clients, channels, and broadcasts.
type Hub struct {
	mu sync.RWMutex

	// All registered clients
	clients map[uuid.UUID]*Client

	// Map of channel name to map of client UUIDs
	channels map[ChannelName]map[uuid.UUID]*Client

	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMsg

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

var GlobalHub *Hub

// InitHub creates and starts the global WebSocket hub.
func InitHub() *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	h := &Hub{
		clients:    make(map[uuid.UUID]*Client),
		channels:   make(map[ChannelName]map[uuid.UUID]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMsg, 256),
		ctx:        ctx,
		cancel:     cancel,
	}
	GlobalHub = h

	h.wg.Add(1)
	go h.run()

	return h
}

// run is the main event loop for the Hub.
func (h *Hub) run() {
	defer h.wg.Done()
	for {
		select {
		case <-h.ctx.Done():
			// Hub is shutting down
			h.mu.Lock()
			for _, c := range h.clients {
				c.Close()
			}
			h.clients = make(map[uuid.UUID]*Client)
			h.channels = make(map[ChannelName]map[uuid.UUID]*Client)
			h.mu.Unlock()
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.playerID] = client
			h.mu.Unlock()
			logger.S.Debugf("Client registered: %s", client.playerID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.playerID]; ok {
				// Remove from all subscribed channels
				for ch := range client.subscriptions {
					if subs, ok := h.channels[ChannelName(ch)]; ok {
						delete(subs, client.playerID)
						if len(subs) == 0 {
							delete(h.channels, ChannelName(ch))
						}
					}
				}
				delete(h.clients, client.playerID)
				client.Close()
				logger.S.Debugf("Client unregistered: %s", client.playerID)
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			subs, ok := h.channels[msg.Channel]
			if !ok {
				h.mu.RUnlock()
				continue
			}
			
			for pid, client := range subs {
				if msg.Exclude != uuid.Nil && pid == msg.Exclude {
					continue
				}
				if len(msg.Targets) > 0 {
					if _, included := msg.Targets[pid]; !included {
						continue
					}
				}
				select {
				case client.send <- msg.Payload:
				default:
					// Send buffer is full; drop the message or kick the client
					// In a production system, you might unregister the client here
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Stop gracefully shuts down the hub and disconnects clients.
func (h *Hub) Stop(ctx context.Context) error {
	h.cancel()
	
	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Subscribe adds a client to a channel.
func (h *Hub) Subscribe(client *Client, channel ChannelName) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if h.channels[channel] == nil {
		h.channels[channel] = make(map[uuid.UUID]*Client)
	}
	h.channels[channel][client.playerID] = client
	
	client.mu.Lock()
	client.subscriptions[string(channel)] = struct{}{}
	client.mu.Unlock()
}

// BroadcastToChannel sends a message to all subscribers of a channel, excluding the given player.
func (h *Hub) BroadcastToChannel(channel ChannelName, payload []byte, exclude uuid.UUID) {
	select {
	case h.broadcast <- &BroadcastMsg{Channel: channel, Payload: payload, Exclude: exclude}:
	default:
	}
}

// BroadcastToTargets sends a message to specific players in a channel.
func (h *Hub) BroadcastToTargets(channel ChannelName, payload []byte, targets map[uuid.UUID]struct{}) {
	select {
	case h.broadcast <- &BroadcastMsg{Channel: channel, Payload: payload, Targets: targets}:
	default:
	}
}

// GetStats returns current connected clients count and active channels count.
func (h *Hub) GetStats() (int, int) {
	if h == nil {
		return 0, 0
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients), len(h.channels)
}

