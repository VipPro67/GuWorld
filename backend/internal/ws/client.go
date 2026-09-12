package ws

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/handlers"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

const (
	writeWait               = 10 * time.Second
	pongWait                = 60 * time.Second
	pingPeriod              = (pongWait * 9) / 10
	maxMessageSize          = 1048576 // 1 MB
	maxConnectionsPerPlayer = 3
)

// GetAllowedOrigins returns the list of allowed WebSocket origins from env var
func GetAllowedOrigins() []string {
	origins := os.Getenv("WS_ALLOWED_ORIGINS")
	if origins == "" {
		// Default to localhost development
		return []string{
			"http://localhost:3000",
			"http://localhost:3001",
			"http://127.0.0.1:3000",
		}
	}
	return strings.Split(origins, ",")
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // Allow same-origin requests
		}
		for _, allowed := range GetAllowedOrigins() {
			if strings.TrimSpace(allowed) == origin {
				return true
			}
		}
		return false
	},
}

// Client represents a connected WebSocket client.
type Client struct {
	playerID uuid.UUID
	conn     *websocket.Conn
	send     chan []byte
	done     chan struct{}
	hub      *Hub
	shard    *ShardedHub
	clientIP string
	commands chan WSMessage

	mu            sync.Mutex
	subscriptions map[string]struct{}

	closeOnce sync.Once
	closeErr  error
}

// ServeWs handles websocket requests from the peer.
func ServeWs(shub *ShardedHub, w http.ResponseWriter, r *http.Request, playerID uuid.UUID, clientIP string) {
	if clientIP == "" {
		// Fallback to extracting client IP from request
		clientIP = r.Header.Get("X-Forwarded-For")
		if clientIP != "" {
			parts := strings.Split(clientIP, ",")
			clientIP = strings.TrimSpace(parts[0])
		} else {
			clientIP = r.RemoteAddr
			// Strip port from IP
			if idx := strings.LastIndex(clientIP, ":"); idx != -1 {
				clientIP = clientIP[:idx]
			}
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.S.Errorf("Failed to upgrade websocket: %v", err)
		ReleaseConnection(playerID, clientIP)
		return
	}

	client := &Client{
		playerID:      playerID,
		conn:          conn,
		send:          make(chan []byte, 1024),
		done:          make(chan struct{}),
		hub:           shub.GetClientShard(playerID),
		subscriptions: make(map[string]struct{}),
		clientIP:      clientIP,
		shard:         shub,
		commands:      make(chan WSMessage, 16),
	}

	client.hub.register <- client
	handlers.TouchPlayerActivity(client.playerID)

	// Auto-subscribe to player-specific and global channels
	shub.Subscribe(client, ChannelName("player:"+playerID.String()))
	shub.Subscribe(client, ChannelName("trade:"+playerID.String()))
	shub.Subscribe(client, ChannelName("notif:"+playerID.String()))
	shub.Subscribe(client, ChannelName("combat:"+playerID.String()))
	shub.Subscribe(client, ChannelName("world:pos"))
	shub.Subscribe(client, ChannelName("world:cell"))
	shub.Subscribe(client, ChannelName("world:cells:"+playerID.String()))
	shub.Subscribe(client, ChannelName("chat:world"))
	shub.Subscribe(client, ChannelName("death_arena"))

	// Check Sect membership
	var sectID string
	err = db.DB.QueryRow("SELECT sect_id FROM sect_members WHERE player_id = $1", playerID).Scan(&sectID)
	if err == nil && sectID != "" {
		shub.Subscribe(client, ChannelName("chat:sect:"+sectID))
	}

	go client.writePump()
	go client.commandPump()
	go client.readPump()

	// Send initial snapshot
	go func() {
		snapshot, err := handlers.LoadPlayerRealtimeSnapshot(playerID)
		if err == nil {
			msg, _ := json.Marshal(struct {
				Type    string                          `json:"type"`
				Channel string                          `json:"channel"`
				Data    handlers.PlayerRealtimeSnapshot `json:"data"`
			}{
				Type:    "event",
				Channel: "player",
				Data:    snapshot,
			})
			select {
			case client.send <- msg:
			default:
			}
		}

		trade, err := handlers.LoadActiveTrade(playerID)
		if err == nil && trade != nil {
			msg, _ := json.Marshal(struct {
				Type    string `json:"type"`
				Channel string `json:"channel"`
				Data    any    `json:"data"`
			}{
				Type:    "event",
				Channel: "trade",
				Data:    map[string]any{"trade": trade},
			})
			select {
			case client.send <- msg:
			default:
			}
		}
	}()
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.shard.Unregister(c)
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		handlers.TouchPlayerActivity(c.playerID)
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived) {
				logger.S.Warnf("Unexpected close error for %s: %v", c.playerID, err)
			}
			break
		}

		var wsMsg WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			logger.S.Warnf("Invalid message format from %s: %v", c.playerID, err)
			continue
		}

		c.handleMessage(wsMsg)
	}
}

// handleMessage processes incoming messages from the client.
func (c *Client) handleMessage(msg WSMessage) {
	handlers.TouchPlayerActivity(c.playerID)
	switch msg.Type {
	case "subscribe":
		var sub SubscribeMsg
		if err := json.Unmarshal(msg.Payload, &sub); err != nil {
			logger.S.Warnf("Invalid subscribe payload from %s: %v", c.playerID, err)
			return
		}
		if strings.HasPrefix(sub.Channel, "chat:sect:") {
			requestedSectID := strings.TrimPrefix(sub.Channel, "chat:sect:")
			var memberSectID string
			err := db.DB.QueryRow("SELECT sect_id FROM sect_members WHERE player_id = $1", c.playerID).Scan(&memberSectID)
			if err != nil || requestedSectID == "" || requestedSectID != memberSectID {
				logger.S.Warnf("Rejected sect chat subscription for %s to %s", c.playerID, requestedSectID)
				return
			}
		}
		c.shard.Subscribe(c, ChannelName(sub.Channel))
	case "ping":
		// handled by pong handler/ping loop, but if client sends application-level ping:
		pong, _ := json.Marshal(WSMessage{Type: "pong"})
		select {
		case c.send <- pong:
		default:
		}
	case "command":
		if msg.RequestID == "" {
			c.sendCommandResponse(msg, http.StatusBadRequest, json.RawMessage(`{"error":"request_id is required"}`))
			return
		}
		switch msg.Channel {
		case "world.move_path", "world.cells_rect", "world.objects", "chat.send":
			select {
			case c.commands <- msg:
			default:
				c.sendCommandResponse(msg, http.StatusTooManyRequests, json.RawMessage(`{"error":"WebSocket command queue is full"}`))
			}
		default:
			c.sendCommandResponse(msg, http.StatusNotFound, json.RawMessage(`{"error":"Unknown command"}`))
		}
	}
}

func (c *Client) commandPump() {
	for {
		select {
		case <-c.done:
			return
		case msg := <-c.commands:
			status, data := handlers.HandleWorldWSCommand(c.playerID, msg.Channel, msg.Payload)
			c.sendCommandResponse(msg, status, data)
		}
	}
}

func (c *Client) sendCommandResponse(request WSMessage, status int, data json.RawMessage) {
	response, err := json.Marshal(struct {
		Type      string          `json:"type"`
		Channel   string          `json:"channel"`
		RequestID string          `json:"request_id"`
		OK        bool            `json:"ok"`
		Status    int             `json:"status"`
		Data      json.RawMessage `json:"data"`
	}{"command.result", request.Channel, request.RequestID, status >= 200 && status < 300, status, data})
	if err != nil {
		return
	}
	select {
	case c.send <- response:
	default:
		c.Close()
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	const maxBatchSize = 64
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.shard.Unregister(c)
	}()

	batch := make([][]byte, 0, maxBatchSize)

	flushBatch := func() error {
		if len(batch) == 0 {
			return nil
		}
		c.conn.SetWriteDeadline(time.Now().Add(writeWait))
		w, err := c.conn.NextWriter(websocket.TextMessage)
		if err != nil {
			batch = batch[:0]
			return err
		}
		for i, msg := range batch {
			if i > 0 {
				w.Write([]byte{'\n'})
			}
			w.Write(msg)
		}
		if err := w.Close(); err != nil {
			batch = batch[:0]
			return err
		}
		batch = batch[:0]
		return nil
	}

	for {
		select {
		case <-c.done:
			return
		case message := <-c.send:
			batch = append(batch, message)
		drainLoop:
			for len(batch) < maxBatchSize {
				select {
				case m := <-c.send:
					batch = append(batch, m)
				default:
					break drainLoop
				}
			}
			if err := flushBatch(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Close ensures connection is only closed once.
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		if c.done != nil {
			close(c.done)
		}
		if c.conn != nil {
			c.closeErr = c.conn.Close()
		}
		ReleaseConnection(c.playerID, c.clientIP)
	})
}

// GetSubscriptions returns a copy of current channel subscriptions.
func (c *Client) GetSubscriptions() []ChannelName {
	c.mu.Lock()
	defer c.mu.Unlock()
	subs := make([]ChannelName, 0, len(c.subscriptions))
	for ch := range c.subscriptions {
		subs = append(subs, ChannelName(ch))
	}
	return subs
}
