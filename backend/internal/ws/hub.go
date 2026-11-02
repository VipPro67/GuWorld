package ws

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// ChannelName provides stronger typing for channel keys.
type ChannelName string

const (
	// clientKindWorld is the Phaser world socket. It owns the character lease,
	// so a player may only have one live world connection at a time.
	clientKindWorld = "world"
	// clientKindUI is the web UI socket. Multiple tabs may hold one each; they
	// only carry player/channel events, so they never evict each other.
	clientKindUI = "ui"
)

// clientKey identifies one realtime session slot for a player.
type clientKey struct {
	playerID uuid.UUID
	kind     string
}

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

	// Active session per (player, kind). Registering a world session closes the
	// superseded one so a half-open controller cannot keep the character lease.
	clients map[clientKey]*Client

	// Live connections per player, for presence. Keyed by client rather than by
	// session slot so an evicted socket cannot zero a player that still has a
	// live one.
	presence map[uuid.UUID]map[*Client]struct{}

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
		clients:    make(map[clientKey]*Client),
		presence:   make(map[uuid.UUID]map[*Client]struct{}),
		channels:   make(map[ChannelName]map[uuid.UUID]*Client),
		register:   make(chan *Client, 128),
		unregister: make(chan *Client, 128),
		broadcast:  make(chan *BroadcastMsg, 4096),
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
	defer func() {
		if r := recover(); r != nil {
			logger.S.Errorf("Hub run panic recovered: %v", r)
		}
		h.wg.Done()
	}()
	for {
		select {
		case <-h.ctx.Done():
			// Hub is shutting down. Close every live socket, not just the newest
			// session slot per player.
			h.mu.Lock()
			live := make([]*Client, 0, len(h.clients))
			for _, clients := range h.presence {
				for c := range clients {
					live = append(live, c)
				}
			}
			h.clients = make(map[clientKey]*Client)
			h.presence = make(map[uuid.UUID]map[*Client]struct{})
			h.channels = make(map[ChannelName]map[uuid.UUID]*Client)
			h.mu.Unlock()
			for _, c := range live {
				c.Close()
			}
			return

		case client := <-h.register:
			key := clientKey{playerID: client.playerID, kind: client.sessionKind()}
			h.mu.Lock()
			previous := h.clients[key]
			h.clients[key] = client
			if h.presence[client.playerID] == nil {
				h.presence[client.playerID] = make(map[*Client]struct{})
			}
			h.presence[client.playerID][client] = struct{}{}
			h.mu.Unlock()

			// Takeover: a new world controller supersedes the previous one.
			// Closing it releases the character lease so the next join is not
			// rejected with 409 realtime_character_already_owned. Run it off the
			// hub loop because settling the lease writes to the database.
			if previous != nil && previous != client && key.kind == clientKindWorld {
				logger.S.Infof("Realtime takeover for %s: closing superseded world connection", client.playerID)
				go previous.Close()
			}
			logger.S.Debugf("Client registered: %s (%s)", client.playerID, key.kind)

		case client := <-h.unregister:
			key := clientKey{playerID: client.playerID, kind: client.sessionKind()}
			h.mu.Lock()
			if h.clients[key] == client {
				delete(h.clients, key)
			}
			if live, ok := h.presence[client.playerID]; ok {
				delete(live, client)
				if len(live) == 0 {
					delete(h.presence, client.playerID)
					logger.S.Debugf("Client unregistered: %s", client.playerID)
				}
			}
			// Remove from all subscribed channels on this shard if it matches this client
			for _, ch := range client.GetSubscriptions() {
				if subs, ok := h.channels[ch]; ok {
					if subs[client.playerID] == client {
						delete(subs, client.playerID)
						if len(subs) == 0 {
							delete(h.channels, ch)
						}
					}
				}
			}
			h.mu.Unlock()
			// Close outside the hub lock and off the shard goroutine: settlement
			// flushes to the database and must not stall broadcasts for this shard.
			go client.Close()

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
				if msg.Targets != nil {
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

// Unsubscribe removes a client from a channel.
func (h *Hub) Unsubscribe(client *Client, channel ChannelName) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if subs, ok := h.channels[channel]; ok {
		if subs[client.playerID] == client {
			delete(subs, client.playerID)
			if len(subs) == 0 {
				delete(h.channels, channel)
			}
		}
	}

	client.mu.Lock()
	delete(client.subscriptions, string(channel))
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
	if targets != nil && len(targets) == 0 {
		return
	}
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
	return len(h.presence), len(h.channels)
}

// IsConnected reports whether a player currently has an active WebSocket connection.
func (h *Hub) IsConnected(playerID uuid.UUID) bool {
	if h == nil {
		return false
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.presence[playerID]) > 0
}
