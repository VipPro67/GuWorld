package ws

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/handlers"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1048576 // 1 MB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Rely on CORS middleware or implement tighter logic here
		return true
	},
}

// Client represents a connected WebSocket client.
type Client struct {
	playerID uuid.UUID
	conn     *websocket.Conn
	send     chan []byte
	hub      *Hub

	mu            sync.Mutex
	subscriptions map[string]struct{}

	closeOnce sync.Once
	closeErr  error
}

// ServeWs handles websocket requests from the peer.
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request, playerID uuid.UUID) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.S.Errorf("Failed to upgrade websocket: %v", err)
		return
	}

	client := &Client{
		playerID:      playerID,
		conn:          conn,
		send:          make(chan []byte, 256),
		hub:           hub,
		subscriptions: make(map[string]struct{}),
	}

	client.hub.register <- client
	handlers.TouchPlayerActivity(client.playerID)

	// Auto-subscribe to player-specific and global channels
	hub.Subscribe(client, ChannelName("player:"+playerID.String()))
	hub.Subscribe(client, ChannelName("trade:"+playerID.String()))
	hub.Subscribe(client, ChannelName("notif:"+playerID.String()))
	hub.Subscribe(client, ChannelName("combat:"+playerID.String()))
	hub.Subscribe(client, ChannelName("world:pos"))
	hub.Subscribe(client, ChannelName("world:cell"))
	hub.Subscribe(client, ChannelName("chat:world"))

	// Check Sect membership
	var sectID string
	err = db.DB.QueryRow("SELECT sect_id FROM sect_members WHERE player_id = $1", playerID).Scan(&sectID)
	if err == nil && sectID != "" {
		hub.Subscribe(client, ChannelName("chat:sect:"+sectID))
		hub.Subscribe(client, ChannelName("chat:sect"))
	}

	go client.writePump()
	go client.readPump()

	// Send initial snapshot
	go func() {
		snapshot, err := handlers.LoadPlayerRealtimeSnapshot(playerID)
		if err == nil {
			msg, _ := json.Marshal(struct {
				Type    string                        `json:"type"`
				Channel string                        `json:"channel"`
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
		c.hub.unregister <- c
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
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.S.Errorf("Unexpected close error: %v", err)
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
		c.hub.Subscribe(c, ChannelName(sub.Channel))
	case "ping":
		// handled by pong handler/ping loop, but if client sends application-level ping:
		pong, _ := json.Marshal(WSMessage{Type: "pong"})
		c.send <- pong
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.hub.unregister <- c
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
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
		close(c.send)
		if c.conn != nil {
			c.closeErr = c.conn.Close()
		}
	})
}
