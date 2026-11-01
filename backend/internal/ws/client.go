package ws

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/handlers"
	"github.com/vippro67/GuWorld/backend/internal/monitor"
	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
	"github.com/vippro67/GuWorld/backend/internal/realtime/protocol"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	writeWait               = 10 * time.Second
	pongWait                = 60 * time.Second
	pingPeriod              = (pongWait * 9) / 10
	maxMessageSize          = 1048576 // 1 MB
	maxConnectionsPerPlayer = 3
	realtimeInputQueueSize  = 64
	realtimeInputsPerSecond = 30
	snapshotBaselineTicks   = 40
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
	playerID            uuid.UUID
	conn                *websocket.Conn
	send                chan []byte
	snapshots           chan []byte
	done                chan struct{}
	hub                 *Hub
	shard               *ShardedHub
	clientIP            string
	userAgent           string
	connectedAt         time.Time
	wsMessagesIn        atomic.Int64
	wsMessagesOut       atomic.Int64
	wsBytesIn           atomic.Int64
	wsBytesOut          atomic.Int64
	wsActionMu          sync.Mutex
	wsActions           map[string]*wsActionAccumulator
	commands            chan WSMessage
	worldInputs         chan WSMessage
	inputRateMu         sync.Mutex
	inputWindowStart    time.Time
	inputWindowCount    int
	snapshotMu          sync.Mutex
	snapshotCancel      context.CancelFunc
	snapshotUnsubscribe func()
	snapshotGeneration  atomic.Uint64
	snapshotStateMu     sync.Mutex
	deltaTracker        *SnapshotDeltaTracker
	controllerID        uuid.UUID
	realtimeAdmission   RealtimeWorldAdmission
	lastTransferReject  atomic.Int64

	mu            sync.Mutex
	realtimeMu    sync.Mutex
	subscriptions map[string]struct{}

	closeOnce sync.Once
	closeErr  error
}

type wsActionAccumulator struct {
	metric          monitor.WSActionMetric
	totalDurationMs float64
	durationSamples int64
}

// ServeWs handles websocket requests from the peer.
func ServeWs(shub *ShardedHub, w http.ResponseWriter, r *http.Request, playerID uuid.UUID, clientIP string, admission ...RealtimeWorldAdmission) {
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

	var realtimeAdmission RealtimeWorldAdmission
	if len(admission) > 0 {
		realtimeAdmission = admission[0]
	}
	client := &Client{
		playerID:          playerID,
		controllerID:      uuid.New(),
		realtimeAdmission: realtimeAdmission,
		conn:              conn,
		send:              make(chan []byte, 1024),
		snapshots:         make(chan []byte, 1),
		done:              make(chan struct{}),
		hub:               shub.GetClientShard(playerID),
		subscriptions:     make(map[string]struct{}),
		clientIP:          clientIP,
		userAgent:         r.UserAgent(),
		connectedAt:       time.Now(),
		shard:             shub,
		commands:          make(chan WSMessage, 16),
		worldInputs:       make(chan WSMessage, realtimeInputQueueSize),
		deltaTracker:      NewSnapshotDeltaTracker(),
	}

	client.hub.register <- client
	handlers.TouchPlayerActivity(client.playerID)
	handlers.EnsureWorldPosLoaded(client.playerID.String())

	isWorldOnly := r != nil && r.URL.Query().Get("purpose") == "world"

	if !isWorldOnly {
		// Auto-subscribe to player-specific and global channels for web UI
		shub.Subscribe(client, ChannelName("player:"+playerID.String()))
		shub.Subscribe(client, ChannelName("refine:"+playerID.String()))
		shub.Subscribe(client, ChannelName("trade:"+playerID.String()))
		shub.Subscribe(client, ChannelName("notif:"+playerID.String()))
		shub.Subscribe(client, ChannelName("combat:"+playerID.String()))
		shub.Subscribe(client, ChannelName("world:pos"))
		shub.Subscribe(client, ChannelName("world:cell"))
		shub.Subscribe(client, ChannelName("world:combat"))
		shub.Subscribe(client, ChannelName("world:cells:"+playerID.String()))
		shub.Subscribe(client, ChannelName("chat:world"))
		shub.Subscribe(client, ChannelName("chat:private:"+playerID.String()))
		shub.Subscribe(client, ChannelName("death_arena"))
		shub.Subscribe(client, ChannelName("martial_arena"))
		shub.Subscribe(client, ChannelName("catalog"))

		// Check Sect membership
		var sectID string
		err = db.DB.QueryRow("SELECT sect_id FROM sect_members WHERE player_id = $1", playerID).Scan(&sectID)
		if err == nil && sectID != "" {
			shub.Subscribe(client, ChannelName("chat:sect:"+sectID))
		}
	}

	go client.writePump()
	go client.commandPump()
	go client.realtimeInputPump()
	go client.readPump()

	if !isWorldOnly {
		// Send initial snapshot to web UI store
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
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	var closeErr error
	defer func() {
		c.recordWebSocketSession(closeErr)
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
			closeErr = err
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived) {
				logger.S.Warnf("Unexpected close error for %s: %v", c.playerID, err)
			}
			break
		}
		c.wsMessagesIn.Add(1)
		c.wsBytesIn.Add(int64(len(message)))

		var wsMsg WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			logger.S.Warnf("Invalid message format from %s: %v", c.playerID, err)
			continue
		}
		wsMsg.Raw = append([]byte(nil), message...)
		c.beginWSAction(&wsMsg)

		c.handleMessage(wsMsg)
	}
}

func (c *Client) recordWebSocketSession(closeErr error) {
	closeCode := websocket.CloseAbnormalClosure
	closeReason := "connection ended without a WebSocket close frame"
	if closeErr != nil {
		var wsCloseErr *websocket.CloseError
		if errors.As(closeErr, &wsCloseErr) {
			closeCode = wsCloseErr.Code
			closeReason = wsCloseErr.Text
		} else {
			closeReason = closeErr.Error()
		}
	}
	trace := monitor.RequestTrace{
		Timestamp:     c.connectedAt,
		Method:        "WS",
		Path:          "/api/ws",
		Status:        http.StatusSwitchingProtocols,
		DurationMs:    roundDurationMs(time.Since(c.connectedAt)),
		ClientIP:      c.clientIP,
		PlayerID:      c.playerID.String(),
		UserAgent:     c.userAgent,
		WSMessagesIn:  c.wsMessagesIn.Load(),
		WSMessagesOut: c.wsMessagesOut.Load(),
		WSBytesIn:     c.wsBytesIn.Load(),
		WSBytesOut:    c.wsBytesOut.Load(),
		WSCloseCode:   closeCode,
		WSCloseReason: closeReason,
		WSActions:     c.snapshotWSActions(),
	}
	if closeCode != websocket.CloseNormalClosure && closeCode != websocket.CloseGoingAway {
		trace.Error = closeReason
	}
	monitor.GlobalTracker.RecordTraceOnly(trace)
}

func (c *Client) beginWSAction(msg *WSMessage) {
	action := wsActionName(*msg)
	if action == "" {
		return
	}
	msg.Action = action
	msg.ReceivedAt = time.Now()
	c.wsActionMu.Lock()
	if c.wsActions == nil {
		c.wsActions = make(map[string]*wsActionAccumulator)
	}
	entry := c.wsActions[action]
	if entry == nil {
		entry = &wsActionAccumulator{metric: monitor.WSActionMetric{Action: action}}
		c.wsActions[action] = entry
	}
	entry.metric.Received++
	c.wsActionMu.Unlock()
}

func wsActionName(msg WSMessage) string {
	switch msg.Type {
	case "command":
		switch msg.Channel {
		case "world.leave", "world.cells_rect", "world.drops", "world.objects", "match.input", "chat.send":
			return "command:" + msg.Channel
		default:
			return "command:unknown"
		}
	case protocol.TypeJoin:
		return "world.join"
	case protocol.TypeInput:
		return "world.input"
	case protocol.TypeMoveTo:
		return "world.move_to"
	case protocol.TypeCast:
		return "world.cast"
	case protocol.TypeAttack:
		return "world.attack"
	case "subscribe", "unsubscribe", "ping":
		return msg.Type
	default:
		if strings.HasPrefix(msg.Type, "world.") {
			return "world.unsupported"
		}
		return ""
	}
}

func (c *Client) completeWSAction(msg WSMessage, status int, duration time.Duration, data json.RawMessage) {
	if msg.Action == "" {
		return
	}
	if !msg.ReceivedAt.IsZero() {
		duration = time.Since(msg.ReceivedAt)
	}
	var response struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(data, &response)
	durationMs := roundDurationMs(duration)
	c.wsActionMu.Lock()
	defer c.wsActionMu.Unlock()
	entry := c.wsActions[msg.Action]
	if entry == nil {
		return
	}
	entry.metric.LastStatus = status
	if status >= 500 {
		entry.metric.Errors++
	} else if status >= 400 {
		entry.metric.Rejected++
	} else {
		entry.metric.Accepted++
	}
	if status >= 400 && response.Code != "" {
		entry.metric.LastErrorCode = response.Code
	}
	entry.totalDurationMs += durationMs
	entry.durationSamples++
	entry.metric.AvgDurationMs = entry.totalDurationMs / float64(entry.durationSamples)
	if durationMs > entry.metric.MaxDurationMs {
		entry.metric.MaxDurationMs = durationMs
	}
}

func (c *Client) snapshotWSActions() []monitor.WSActionMetric {
	c.wsActionMu.Lock()
	defer c.wsActionMu.Unlock()
	if len(c.wsActions) == 0 {
		return nil
	}
	actions := make([]monitor.WSActionMetric, 0, len(c.wsActions))
	for _, entry := range c.wsActions {
		actions = append(actions, entry.metric)
	}
	sort.Slice(actions, func(i, j int) bool { return actions[i].Action < actions[j].Action })
	return actions
}

func (c *Client) recordWSGameplayRejection(action, reason string) {
	c.wsActionMu.Lock()
	defer c.wsActionMu.Unlock()
	if c.wsActions == nil {
		c.wsActions = make(map[string]*wsActionAccumulator)
	}
	entry := c.wsActions[action]
	if entry == nil {
		entry = &wsActionAccumulator{metric: monitor.WSActionMetric{Action: action}}
		c.wsActions[action] = entry
	}
	entry.metric.GameplayRejections++
	entry.metric.LastGameplayReason = reason
}

func roundDurationMs(duration time.Duration) float64 {
	return float64(duration.Microseconds()) / 1000
}

// handleMessage processes incoming messages from the client.
func (c *Client) handleMessage(msg WSMessage) {
	handlers.TouchPlayerActivity(c.playerID)
	switch msg.Type {
	case "subscribe":
		var sub SubscribeMsg
		if err := json.Unmarshal(msg.Payload, &sub); err != nil {
			logger.S.Warnf("Invalid subscribe payload from %s: %v", c.playerID, err)
			c.completeWSAction(msg, http.StatusBadRequest, time.Since(msg.ReceivedAt), json.RawMessage(`{"code":"invalid_subscription"}`))
			return
		}
		if strings.HasPrefix(sub.Channel, "chat:sect:") {
			requestedSectID := strings.TrimPrefix(sub.Channel, "chat:sect:")
			var memberSectID string
			err := db.DB.QueryRow("SELECT sect_id FROM sect_members WHERE player_id = $1", c.playerID).Scan(&memberSectID)
			if err != nil || requestedSectID == "" || requestedSectID != memberSectID {
				logger.S.Warnf("Rejected sect chat subscription for %s to %s", c.playerID, requestedSectID)
				c.completeWSAction(msg, http.StatusForbidden, time.Since(msg.ReceivedAt), json.RawMessage(`{"code":"subscription_forbidden"}`))
				return
			}
		}
		if strings.HasPrefix(sub.Channel, "chat:region:") {
			requestedScope := strings.TrimPrefix(sub.Channel, "chat:")
			currentScope, err := handlers.CurrentRegionChatScope(c.playerID.String())
			if err != nil || requestedScope != "region:"+currentScope {
				logger.S.Warnf("Rejected region chat subscription for %s to %s", c.playerID, requestedScope)
				c.completeWSAction(msg, http.StatusForbidden, time.Since(msg.ReceivedAt), json.RawMessage(`{"code":"subscription_forbidden"}`))
				return
			}
		}
		if strings.HasPrefix(sub.Channel, "chat:private:") && strings.TrimPrefix(sub.Channel, "chat:private:") != c.playerID.String() {
			logger.S.Warnf("Rejected private chat subscription for %s to %s", c.playerID, sub.Channel)
			c.completeWSAction(msg, http.StatusForbidden, time.Since(msg.ReceivedAt), json.RawMessage(`{"code":"subscription_forbidden"}`))
			return
		}
		c.shard.Subscribe(c, ChannelName(sub.Channel))
		c.completeWSAction(msg, http.StatusOK, time.Since(msg.ReceivedAt), nil)
	case "unsubscribe":
		var sub SubscribeMsg
		if err := json.Unmarshal(msg.Payload, &sub); err != nil || sub.Channel == "" {
			c.completeWSAction(msg, http.StatusBadRequest, time.Since(msg.ReceivedAt), json.RawMessage(`{"code":"invalid_unsubscription"}`))
			return
		}
		c.shard.Unsubscribe(c, ChannelName(sub.Channel))
		c.completeWSAction(msg, http.StatusOK, time.Since(msg.ReceivedAt), nil)
	case "ping":
		// handled by pong handler/ping loop, but if client sends application-level ping:
		pong, _ := json.Marshal(WSMessage{Type: "pong"})
		select {
		case c.send <- pong:
			c.completeWSAction(msg, http.StatusOK, time.Since(msg.ReceivedAt), nil)
		default:
			c.completeWSAction(msg, http.StatusTooManyRequests, time.Since(msg.ReceivedAt), json.RawMessage(`{"code":"websocket_send_queue_full"}`))
		}
	case "command":
		if msg.RequestID == "" {
			c.completeWSAction(msg, http.StatusBadRequest, time.Since(msg.ReceivedAt), json.RawMessage(`{"code":"request_id_required"}`))
			c.sendCommandResponse(msg, http.StatusBadRequest, json.RawMessage(`{"error":"request_id is required"}`))
			return
		}
		switch msg.Channel {
		case "world.leave", "world.cells_rect", "world.drops", "world.objects", "match.input", "chat.send":
			select {
			case c.commands <- msg:
			default:
				c.completeWSAction(msg, http.StatusTooManyRequests, time.Since(msg.ReceivedAt), json.RawMessage(`{"code":"command_queue_full"}`))
				c.sendCommandResponse(msg, http.StatusTooManyRequests, json.RawMessage(`{"error":"WebSocket command queue is full"}`))
			}
		default:
			c.completeWSAction(msg, http.StatusNotFound, time.Since(msg.ReceivedAt), json.RawMessage(`{"code":"unknown_command"}`))
			c.sendCommandResponse(msg, http.StatusNotFound, json.RawMessage(`{"error":"Unknown command"}`))
		}
	case protocol.TypeJoin:
		if msg.RequestID == "" {
			c.completeWSAction(msg, http.StatusBadRequest, time.Since(msg.ReceivedAt), json.RawMessage(`{"code":"request_id_required"}`))
			c.sendWorldReject("", "request_id_required", "request_id is required", false)
			return
		}
		_, decoded, err := protocol.DecodeClientMessage(msg.Raw)
		if err != nil {
			c.rejectWorldAction(msg, "invalid_world_join", err.Error(), false)
			return
		}
		joinRequest, ok := decoded.(protocol.JoinRequest)
		if !ok {
			c.rejectWorldAction(msg, "invalid_world_join", "Invalid join request", false)
			return
		}
		payload, _ := json.Marshal(joinRequest)
		msg.Payload = payload
		msg.Channel = "world.join"
		select {
		case c.commands <- msg:
		default:
			c.rejectWorldAction(msg, "world_command_queue_full", "WebSocket command queue is full", true)
		}
	case protocol.TypeInput:
		_, decoded, err := protocol.DecodeClientMessage(msg.Raw)
		if err != nil {
			c.rejectWorldAction(msg, "invalid_world_input", err.Error(), false)
			return
		}
		_, ok := decoded.(protocol.Input)
		if !ok {
			c.rejectWorldAction(msg, "invalid_world_input", "Invalid movement input", false)
			return
		}
		msg.Channel = protocol.TypeInput
		if !c.allowRealtimeInput(time.Now()) {
			c.rejectWorldAction(msg, "realtime_input_rate_limited", "Movement input rate exceeded", true)
			return
		}
		select {
		case c.worldInputs <- msg:
		default:
			c.rejectWorldAction(msg, "realtime_input_queue_full", "Realtime input queue is full", true)
		}
	case protocol.TypeMoveTo:
		_, decoded, err := protocol.DecodeClientMessage(msg.Raw)
		if err != nil {
			c.rejectWorldAction(msg, "invalid_world_move_target", err.Error(), false)
			return
		}
		if _, ok := decoded.(protocol.MoveTarget); !ok {
			c.rejectWorldAction(msg, "invalid_world_move_target", "Invalid move target", false)
			return
		}
		msg.Channel = protocol.TypeMoveTo
		if !c.allowRealtimeInput(time.Now()) {
			c.rejectWorldAction(msg, "realtime_input_rate_limited", "Movement input rate exceeded", true)
			return
		}
		select {
		case c.worldInputs <- msg:
		default:
			c.rejectWorldAction(msg, "realtime_input_queue_full", "Realtime input queue is full", true)
		}
	case protocol.TypeCast:
		_, decoded, err := protocol.DecodeClientMessage(msg.Raw)
		if err != nil {
			c.rejectWorldAction(msg, "invalid_world_cast", err.Error(), false)
			return
		}
		if _, ok := decoded.(protocol.Cast); !ok {
			c.rejectWorldAction(msg, "invalid_world_cast", "Invalid cast", false)
			return
		}
		msg.Channel = protocol.TypeCast
		if !c.allowRealtimeInput(time.Now()) {
			c.rejectWorldAction(msg, "realtime_input_rate_limited", "Movement input rate exceeded", true)
			return
		}
		select {
		case c.worldInputs <- msg:
		default:
			c.rejectWorldAction(msg, "realtime_input_queue_full", "Realtime input queue is full", true)
		}
	case protocol.TypeAttack:
		_, decoded, err := protocol.DecodeClientMessage(msg.Raw)
		if err != nil {
			c.rejectWorldAction(msg, "invalid_world_attack", err.Error(), false)
			return
		}
		if _, ok := decoded.(protocol.Attack); !ok {
			c.rejectWorldAction(msg, "invalid_world_attack", "Invalid basic attack", false)
			return
		}
		msg.Channel = protocol.TypeAttack
		if !c.allowRealtimeInput(time.Now()) {
			c.rejectWorldAction(msg, "realtime_input_rate_limited", "Combat input rate exceeded", true)
			return
		}
		select {
		case c.worldInputs <- msg:
		default:
			c.rejectWorldAction(msg, "realtime_input_queue_full", "Realtime input queue is full", true)
		}
	default:
		if strings.HasPrefix(msg.Type, "world.") {
			c.rejectWorldAction(msg, "world_message_not_supported", "Realtime world message is not supported yet", false)
		}
	}
}

func (c *Client) commandPump() {
	for {
		select {
		case <-c.done:
			return
		case msg := <-c.commands:
			start := time.Now()
			if msg.Type == protocol.TypeJoin {
				status, data := c.handleRealtimeLifecycle(msg)
				c.logWSCommand(msg, status, data, time.Since(start))
				if status >= 200 && status < 300 {
					c.sendWorldEnvelope(protocol.TypeWelcome, msg.RequestID, data)
					var welcome protocol.Welcome
					if err := json.Unmarshal(data, &welcome); err != nil {
						c.Close()
						continue
					}
					if err := c.startRealtimeSnapshotStream(welcome); err != nil {
						c.sendWorldReject(msg.RequestID, "realtime_snapshot_unavailable", err.Error(), true)
						c.Close()
					}
				} else {
					var payload struct {
						Code  string `json:"code"`
						Error string `json:"error"`
					}
					_ = json.Unmarshal(data, &payload)
					c.sendWorldReject(msg.RequestID, payload.Code, payload.Error, status >= 500 || status == http.StatusTooManyRequests)
				}
				continue
			}
			if msg.Channel == "world.leave" {
				status, data := c.handleRealtimeLifecycle(msg)
				if status >= 200 && status < 300 {
					c.stopRealtimeSnapshotStream()
				}
				c.logWSCommand(msg, status, data, time.Since(start))
				c.sendCommandResponse(msg, status, data)
				continue
			}
			status, data := handlers.HandleWorldWSCommand(c.playerID, msg.Channel, msg.Payload)
			duration := time.Since(start)
			c.logWSCommand(msg, status, data, duration)
			c.sendCommandResponse(msg, status, data)
		}
	}
}

// realtimeInputPump drains movement and combat actions on an isolated loop
// so slow transactional database RPCs in commandPump do not induce input lag.
func (c *Client) realtimeInputPump() {
	for {
		select {
		case <-c.done:
			return
		case msg := <-c.worldInputs:
			start := time.Now()
			status, data := c.handleRealtimeInput(msg)
			c.logWSCommand(msg, status, data, time.Since(start))
			if status >= 400 {
				var payload struct {
					Code  string `json:"code"`
					Error string `json:"error"`
				}
				_ = json.Unmarshal(data, &payload)
				c.sendWorldReject(msg.RequestID, payload.Code, payload.Error, status == http.StatusTooManyRequests || status >= 500)
			}
		}
	}
}

func (c *Client) allowRealtimeInput(now time.Time) bool {
	c.inputRateMu.Lock()
	defer c.inputRateMu.Unlock()
	if c.inputWindowStart.IsZero() || now.Sub(c.inputWindowStart) >= time.Second || now.Before(c.inputWindowStart) {
		c.inputWindowStart = now
		c.inputWindowCount = 0
	}
	if c.inputWindowCount >= realtimeInputsPerSecond {
		return false
	}
	c.inputWindowCount++
	return true
}

func (c *Client) handleRealtimeInput(msg WSMessage) (int, json.RawMessage) {
	c.realtimeMu.Lock()
	defer c.realtimeMu.Unlock()
	select {
	case <-c.done:
		return http.StatusGone, json.RawMessage(`{"code":"websocket_closed","error":"WebSocket is closed"}`)
	default:
	}
	if c.realtimeAdmission == nil {
		return http.StatusServiceUnavailable, json.RawMessage(`{"code":"realtime_world_unavailable","error":"Realtime world admission is unavailable"}`)
	}
	_, decoded, err := protocol.DecodeClientMessage(msg.Raw)
	if err != nil {
		return http.StatusBadRequest, json.RawMessage(`{"code":"invalid_world_input","error":"Invalid movement input"}`)
	}
	switch command := decoded.(type) {
	case protocol.Input:
		if err := c.realtimeAdmission.Input(c.playerID, c.controllerID, command); err != nil {
			return realtimeAdmissionErrorResponse(err)
		}
	case protocol.MoveTarget:
		if err := c.realtimeAdmission.MoveTarget(c.playerID, c.controllerID, command); err != nil {
			return realtimeAdmissionErrorResponse(err)
		}
	case protocol.Cast:
		if err := c.realtimeAdmission.Cast(c.playerID, c.controllerID, command); err != nil {
			return realtimeAdmissionErrorResponse(err)
		}
	case protocol.Attack:
		if err := c.realtimeAdmission.BasicAttack(c.playerID, c.controllerID, command); err != nil {
			return realtimeAdmissionErrorResponse(err)
		}
	case protocol.CaptureIntent:
		if err := c.realtimeAdmission.Capture(c.playerID, c.controllerID, command); err != nil {
			return realtimeAdmissionErrorResponse(err)
		}
	case protocol.CompanionStanceIntent:
		if err := c.realtimeAdmission.CompanionStance(c.playerID, c.controllerID, command); err != nil {
			return realtimeAdmissionErrorResponse(err)
		}
	default:
		return http.StatusBadRequest, json.RawMessage(`{"code":"invalid_world_input","error":"Invalid movement input"}`)
	}
	return http.StatusAccepted, json.RawMessage(`{"accepted":true}`)
}

func (c *Client) startRealtimeSnapshotStream(welcome protocol.Welcome) error {
	if c.realtimeAdmission == nil {
		return errors.New("realtime world admission is unavailable")
	}
	c.stopRealtimeSnapshotStream()
	feed, unsubscribe, err := c.realtimeAdmission.SubscribeSnapshots(c.playerID, c.controllerID)
	if err != nil {
		return err
	}
	if err := c.initializeSnapshotBaseline(welcome); err != nil {
		unsubscribe()
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.snapshotMu.Lock()
	select {
	case <-c.done:
		c.snapshotMu.Unlock()
		cancel()
		unsubscribe()
		return errors.New("WebSocket is closed")
	default:
	}
	oldCancel, oldUnsubscribe := c.snapshotCancel, c.snapshotUnsubscribe
	generation := c.snapshotGeneration.Load()
	c.snapshotCancel, c.snapshotUnsubscribe = cancel, unsubscribe
	c.snapshotMu.Unlock()
	if oldCancel != nil {
		oldCancel()
	}
	if oldUnsubscribe != nil {
		oldUnsubscribe()
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-c.done:
				return
			case snapshot, ok := <-feed:
				if !ok {
					return
				}
				select {
				case <-ctx.Done():
					return
				default:
				}
				if self, ok := snapshotSelf(snapshot, sim.CharacterID(c.playerID.String())); ok {
					transfer, crossing, transferErr := c.realtimeAdmission.TransferAtEdge(c.playerID, c.controllerID, self.Transform.Position)
					if transferErr != nil {
						if coded, ok := transferErr.(interface{ ErrorCode() string }); ok && coded.ErrorCode() == "realtime_transfer_denied" {
							now := time.Now().UnixNano()
							if now-c.lastTransferReject.Load() > int64(10*time.Second) {
								c.lastTransferReject.Store(now)
								c.sendWorldReject("", coded.ErrorCode(), transferErr.Error(), false)
							}
						}
					} else if crossing {
						payload, marshalErr := json.Marshal(transfer)
						if marshalErr == nil {
							c.sendWorldEnvelope(protocol.TypeTransfer, "", payload)
							logger.S.Infof("Realtime edge transfer %s: %s -> %s (local %.1f,%.1f legacy %d,%d)",
								c.playerID, transfer.FromZoneID, transfer.ToZoneID,
								transfer.TargetX, transfer.TargetY, transfer.LegacyX, transfer.LegacyY)
							// Give the write pump time to flush world.transfer; the
							// client normally closes this socket and reconnects.
							time.AfterFunc(2*time.Second, c.Close)
						}
						return
					}
				}
				c.sendWorldSnapshotGeneration(snapshot, generation)
			}
		}
	}()
	return nil
}

func snapshotSelf(snapshot sim.Snapshot, characterID sim.CharacterID) (sim.Entity, bool) {
	for _, entity := range snapshot.Entities {
		if entity.CharacterID == characterID {
			return entity, true
		}
	}
	return sim.Entity{}, false
}

func (c *Client) stopRealtimeSnapshotStream() {
	c.snapshotStateMu.Lock()
	c.snapshotGeneration.Add(1)
	if c.deltaTracker != nil {
		c.deltaTracker.Reset()
	}
	c.snapshotStateMu.Unlock()
	c.snapshotMu.Lock()
	cancel, unsubscribe := c.snapshotCancel, c.snapshotUnsubscribe
	c.snapshotCancel, c.snapshotUnsubscribe = nil, nil
	c.snapshotMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if unsubscribe != nil {
		unsubscribe()
	}
	for {
		select {
		case <-c.snapshots:
		default:
			return
		}
	}
}

type worldSnapshotEnvelope struct {
	Version int               `json:"v"`
	Type    string            `json:"type"`
	Payload protocol.Snapshot `json:"payload"`
}

type worldEventPayload struct {
	SkillID           string   `json:"skill_id,omitempty"`
	SkillName         string   `json:"skill_name,omitempty"`
	CastVFX           string   `json:"cast_vfx,omitempty"`
	ProjectileVFX     string   `json:"projectile_vfx,omitempty"`
	ImpactVFX         string   `json:"impact_vfx,omitempty"`
	X                 float64  `json:"x,omitempty"`
	Y                 float64  `json:"y,omitempty"`
	Amount            float64  `json:"amount,omitempty"`
	ShieldAbsorbed    float64  `json:"shield_absorbed,omitempty"`
	HPDamage          float64  `json:"hp_damage,omitempty"`
	HP                float64  `json:"hp,omitempty"`
	Shield            float64  `json:"shield,omitempty"`
	DurationTicks     uint64   `json:"duration_ticks,omitempty"`
	DashDirX          *float64 `json:"dash_dir_x,omitempty"`
	DashDirY          *float64 `json:"dash_dir_y,omitempty"`
	DashDurationTicks uint64   `json:"dash_duration_ticks,omitempty"`
	Reason            string   `json:"reason,omitempty"`
}

type typedWorldEvent struct {
	EventID  uint64             `json:"event_id"`
	Tick     uint64             `json:"tick"`
	Type     string             `json:"type"`
	SourceID string             `json:"source_id,omitempty"`
	TargetID string             `json:"target_id,omitempty"`
	Payload  *worldEventPayload `json:"payload,omitempty"`
}

type typedWorldEventBatch struct {
	FromEventID uint64            `json:"from_event_id"`
	Events      []typedWorldEvent `json:"events"`
}

type worldEventsEnvelope struct {
	Version int                  `json:"v"`
	Type    string               `json:"type"`
	Payload typedWorldEventBatch `json:"payload"`
}

func (c *Client) sendWorldSnapshot(snapshot sim.Snapshot) {
	c.sendWorldSnapshotGeneration(snapshot, c.snapshotGeneration.Load())
}

func (c *Client) sendWorldSnapshotGeneration(snapshot sim.Snapshot, generation uint64) {
	c.sendWorldEvents(snapshot.Events, generation)
	c.snapshotStateMu.Lock()
	defer c.snapshotStateMu.Unlock()
	if generation != c.snapshotGeneration.Load() {
		return
	}
	messageSnapshot := c.buildSnapshotDeltaLocked(snapshot)
	message, err := json.Marshal(worldSnapshotEnvelope{
		Version: protocol.Version,
		Type:    protocol.TypeSnapshot,
		Payload: messageSnapshot,
	})
	if err != nil {
		return
	}
	select {
	case <-c.done:
		return
	default:
	}
	if generation != c.snapshotGeneration.Load() {
		return
	}
	select {
	case c.snapshots <- message:
	default:
		select {
		case <-c.snapshots:
		default:
		}
		select {
		case c.snapshots <- message:
		default:
		}
	}
}

func (c *Client) sendWorldEvents(events []combat.Event, generation uint64) {
	if len(events) == 0 || generation != c.snapshotGeneration.Load() {
		return
	}
	batch := typedWorldEventBatch{
		FromEventID: events[0].ID,
		Events:      make([]typedWorldEvent, 0, len(events)),
	}
	for _, event := range events {
		if event.SourceID == c.playerID.String() {
			switch event.Type {
			case combat.EventCastRejected:
				action := "world.cast"
				if event.SkillID == "BASIC_ATTACK" {
					action = "world.attack"
				}
				c.recordWSGameplayRejection(action, event.Reason)
			case combat.EventMoveRejected:
				c.recordWSGameplayRejection("world.move_to", event.Reason)
			}
		}
		payload := &worldEventPayload{
			SkillID:           event.SkillID,
			SkillName:         event.SkillName,
			CastVFX:           event.CastVFX,
			ProjectileVFX:     event.ProjectileVFX,
			ImpactVFX:         event.ImpactVFX,
			X:                 event.X,
			Y:                 event.Y,
			Amount:            event.Amount,
			ShieldAbsorbed:    event.ShieldAbsorbed,
			HPDamage:          event.HPDamage,
			HP:                event.HP,
			Shield:            event.Shield,
			DurationTicks:     event.DurationTicks,
			DashDurationTicks: event.DashDurationTicks,
			Reason:            event.Reason,
		}
		if event.Type == combat.EventCastAccepted && event.DashDurationTicks > 0 {
			payload.DashDirX = &event.DashDirX
			payload.DashDirY = &event.DashDirY
		}
		batch.Events = append(batch.Events, typedWorldEvent{
			EventID:  event.ID,
			Tick:     event.Tick,
			Type:     string(event.Type),
			SourceID: event.SourceID,
			TargetID: event.TargetID,
			Payload:  payload,
		})
	}
	message, err := json.Marshal(worldEventsEnvelope{
		Version: protocol.Version,
		Type:    protocol.TypeEvents,
		Payload: batch,
	})
	if err != nil {
		return
	}
	select {
	case c.send <- message:
	default:
		// Preview limitation: reliable event replay is not implemented yet. A
		// slow command queue drops this batch rather than blocking the zone feed.
	}
}

func protocolEntityState(entity sim.Entity) protocol.EntityState {
	state := protocol.EntityState{
		EntityID: string(entity.ID), Revision: entity.Revision, Kind: string(entity.Kind), PartyID: entity.PartyID,
		PvPEnabled: entity.PvPEnabled, Radius: entity.Radius,
		X:          entity.Transform.Position.X, Y: entity.Transform.Position.Y,
		HP: entity.Vitals.HP, MaxHP: entity.Vitals.MaxHP, Shield: entity.Vitals.Shield,
	}
	if entity.Transformation.Active || entity.Transformation.IsWindup {
		state.Transformation = &protocol.TransformationState{
			Active:          entity.Transformation.Active,
			FormID:          entity.Transformation.FormID,
			FormName:        entity.Transformation.FormName,
			FormKind:        string(entity.Transformation.FormKind),
			ModelScale:      entity.Transformation.ModelScale,
			CollisionRadius: entity.Radius,
			MaxFormHP:       entity.Transformation.MaxFormHP,
			CurrentFormHP:   entity.Transformation.CurrentFormHP,
			IsWindup:        entity.Transformation.IsWindup,
		}
	}
	if entity.Kind == sim.EntityMonster {
		state.TemplateID = entity.AI.TemplateID
		state.Name = entity.AI.Name
		state.AIState = string(entity.AI.State)
		state.AggroTarget = string(entity.AI.Threat)
	} else if entity.Name != "" {
		state.Name = entity.Name
	}
	if entity.Combat.MaxMana > 0 {
		state.Mana = entity.Combat.Mana
		state.MaxMana = entity.Combat.MaxMana
		state.FormActive = entity.Combat.FormActive
		state.Cooldowns = make([]uint64, len(entity.Combat.Cooldowns))
		copy(state.Cooldowns, entity.Combat.Cooldowns[:])
		if entity.Combat.Pending != nil {
			state.CastSkillID = entity.Combat.Pending.Skill.ID
			state.CastEndTick = entity.Combat.Pending.ResolveTick
			state.CastWindup = entity.Combat.Pending.Skill.WindupTicks
		}
	}
	if len(entity.Combat.Statuses) > 0 {
		state.Statuses = make([]string, 0, len(entity.Combat.Statuses))
		for _, st := range entity.Combat.Statuses {
			state.Statuses = append(state.Statuses, st.ID)
			if st.ID == "STEALTH" || st.ID == "INVISIBILITY" {
				state.Stealthed = true
			}
			if st.ID == "FLYING" || st.ID == "FLIGHT" {
				state.Flying = true
			}
		}
	}
	return state
}

func protocolProjectiles(projectiles []sim.ProjectileState) []protocol.ProjectileState {
	if len(projectiles) == 0 {
		return nil
	}
	result := make([]protocol.ProjectileState, 0, len(projectiles))
	for _, projectile := range projectiles {
		result = append(result, protocol.ProjectileState{
			ID: projectile.ID, SourceID: projectile.SourceID, SkillID: projectile.SkillID,
			X: projectile.X, Y: projectile.Y, Radius: projectile.Radius,
		})
	}
	return result
}

func (c *Client) trackerLocked() *SnapshotDeltaTracker {
	if c.deltaTracker == nil {
		c.deltaTracker = NewSnapshotDeltaTracker()
	}
	return c.deltaTracker
}

func (c *Client) initializeSnapshotBaseline(welcome protocol.Welcome) error {
	c.snapshotStateMu.Lock()
	defer c.snapshotStateMu.Unlock()
	return c.trackerLocked().Initialize(welcome)
}

func (c *Client) buildSnapshotDelta(snapshot sim.Snapshot) protocol.Snapshot {
	c.snapshotStateMu.Lock()
	defer c.snapshotStateMu.Unlock()
	return c.buildSnapshotDeltaLocked(snapshot)
}

func (c *Client) buildSnapshotDeltaLocked(snapshot sim.Snapshot) protocol.Snapshot {
	return c.trackerLocked().BuildDelta(snapshot, c.playerID.String())
}

func (c *Client) handleRealtimeLifecycle(msg WSMessage) (int, json.RawMessage) {
	c.realtimeMu.Lock()
	defer c.realtimeMu.Unlock()
	select {
	case <-c.done:
		return http.StatusGone, json.RawMessage(`{"code":"websocket_closed","error":"WebSocket is closed"}`)
	default:
	}
	if c.realtimeAdmission == nil {
		return http.StatusServiceUnavailable, json.RawMessage(`{"code":"realtime_world_unavailable","error":"Realtime world admission is unavailable"}`)
	}

	var response any
	var err error
	switch msg.Channel {
	case "world.join":
		var request protocol.JoinRequest
		if len(msg.Payload) == 0 || json.Unmarshal(msg.Payload, &request) != nil || request.ZoneID == "" {
			return http.StatusBadRequest, json.RawMessage(`{"code":"invalid_world_join","error":"zone_id is required"}`)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		response, err = c.realtimeAdmission.Join(ctx, c.playerID, c.controllerID, request)
		cancel()
	case "world.leave":
		err = c.realtimeAdmission.Leave(c.playerID, c.controllerID)
		if err == nil {
			response = map[string]string{"type": "world.left"}
		}
	}

	select {
	case <-c.done:
		if msg.Channel == "world.join" && err == nil {
			c.realtimeAdmission.Disconnect(c.playerID, c.controllerID)
		}
		return http.StatusGone, json.RawMessage(`{"code":"websocket_closed","error":"WebSocket is closed"}`)
	default:
	}
	if err != nil {
		return realtimeAdmissionErrorResponse(err)
	}
	data, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		return http.StatusInternalServerError, json.RawMessage(`{"code":"realtime_response_failed","error":"Failed to encode realtime response"}`)
	}
	return http.StatusOK, json.RawMessage(data)
}

func realtimeAdmissionErrorResponse(err error) (int, json.RawMessage) {
	status := http.StatusServiceUnavailable
	code := "realtime_world_unavailable"
	type codedError interface {
		HTTPStatus() int
		ErrorCode() string
	}
	var coded codedError
	if errors.As(err, &coded) {
		status, code = coded.HTTPStatus(), coded.ErrorCode()
	}
	data, _ := json.Marshal(map[string]string{"code": code, "error": err.Error()})
	return status, json.RawMessage(data)
}

func (c *Client) sendWorldReject(requestID, code, message string, retryable bool) {
	if code == "" {
		code = "realtime_world_rejected"
	}
	payload, _ := json.Marshal(protocol.Reject{Code: code, Message: message, Retryable: retryable})
	c.sendWorldEnvelope(protocol.TypeReject, requestID, payload)
}

func (c *Client) rejectWorldAction(msg WSMessage, code, message string, retryable bool) {
	status := http.StatusBadRequest
	if retryable {
		status = http.StatusTooManyRequests
	}
	data, _ := json.Marshal(map[string]string{"code": code})
	c.completeWSAction(msg, status, time.Since(msg.ReceivedAt), data)
	c.sendWorldReject(msg.RequestID, code, message, retryable)
}

func (c *Client) sendWorldEnvelope(messageType, requestID string, payload json.RawMessage) {
	message, err := json.Marshal(protocol.Envelope{Version: protocol.Version, Type: messageType, RequestID: requestID, Payload: payload})
	if err != nil {
		return
	}
	select {
	case c.send <- message:
	default:
		c.Close()
	}
}

func (c *Client) logWSCommand(msg WSMessage, status int, data json.RawMessage, duration time.Duration) {
	c.completeWSAction(msg, status, duration, data)
	level := zapcore.InfoLevel
	switch {
	case status >= 500:
		level = zapcore.ErrorLevel
	case status >= 400:
		level = zapcore.WarnLevel
	}

	var reqPayload interface{}
	if len(msg.Payload) > 0 {
		_ = json.Unmarshal(msg.Payload, &reqPayload)
	}

	var respData interface{}
	var errMsg string
	if len(data) > 0 {
		if err := json.Unmarshal(data, &respData); err == nil {
			if m, ok := respData.(map[string]interface{}); ok {
				if e, ok := m["error"].(string); ok {
					errMsg = e
				}
			}
		} else {
			respData = string(data)
			errMsg = string(data)
		}
	}

	durMs := float64(duration.Microseconds()) / 1000

	if status >= 400 {
		if checked := logger.L.Check(level, "ws_command_error"); checked != nil {
			checked.Write(
				zap.String("player_id", c.playerID.String()),
				zap.String("client_ip", c.clientIP),
				zap.String("command", msg.Channel),
				zap.String("request_id", msg.RequestID),
				zap.Int("status", status),
				zap.Float64("duration_ms", durMs),
				zap.Any("request_body", reqPayload),
				zap.Any("response_body", respData),
				zap.String("error", errMsg),
			)
		}
	}

	if checked := logger.L.Check(level, "ws_command_completed"); checked != nil {
		fields := []zapcore.Field{
			zap.String("player_id", c.playerID.String()),
			zap.String("client_ip", c.clientIP),
			zap.String("command", msg.Channel),
			zap.String("request_id", msg.RequestID),
			zap.Int("status", status),
			zap.Float64("duration_ms", durMs),
			zap.Any("request_body", reqPayload),
		}
		if status >= 400 {
			fields = append(fields, zap.Any("response_body", respData), zap.String("error", errMsg))
		}
		checked.Write(fields...)
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
		c.wsMessagesOut.Add(int64(len(batch)))
		for _, msg := range batch {
			c.wsBytesOut.Add(int64(len(msg)))
		}
		batch = batch[:0]
		return nil
	}
	flushReliable := func(message []byte) error {
		batch = append(batch, message)
		for len(batch) < maxBatchSize {
			select {
			case next := <-c.send:
				batch = append(batch, next)
			default:
				return flushBatch()
			}
		}
		return flushBatch()
	}

	for {
		select {
		case <-c.done:
			return
		default:
		}
		// Preserve reliable welcome/command/event ordering ahead of replaceable snapshots.
		select {
		case message := <-c.send:
			if err := flushReliable(message); err != nil {
				return
			}
			continue
		default:
		}
		select {
		case <-c.done:
			return
		case message := <-c.send:
			if err := flushReliable(message); err != nil {
				return
			}
		case snapshot := <-c.snapshots:
			// A welcome may have been queued after the priority check above.
			// Since this snapshot is replaceable, send reliable messages first
			// and let the next zone tick provide a fresh baseline.
			select {
			case reliable := <-c.send:
				if err := flushReliable(reliable); err != nil {
					return
				}
				continue
			default:
			}
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, snapshot); err != nil {
				return
			}
			c.wsMessagesOut.Add(1)
			c.wsBytesOut.Add(int64(len(snapshot)))
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
		c.stopRealtimeSnapshotStream()
		c.realtimeMu.Lock()
		if c.realtimeAdmission != nil {
			c.realtimeAdmission.Disconnect(c.playerID, c.controllerID)
		}
		c.realtimeMu.Unlock()
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
