package uihub

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	wsWorld      byte = 1
	wsUI         byte = 2
	wsDgram      byte = 3
	wsClose      byte = 4
	StreamUI     byte = 2
	MaxFrameSize      = 1 << 20
)

// UIOptions configures optional domain hooks for the UI websocket stream.
type UIOptions struct {
	Command       func(playerID uuid.UUID, command string, payload json.RawMessage) (int, json.RawMessage)
	SectLookup    func(playerID uuid.UUID) string
	RegionScope   func(playerID uuid.UUID) (string, error)
	InitialEvents func(playerID uuid.UUID) [][]byte
	TouchActivity func(playerID uuid.UUID)
}

// Server handles UI event streaming over WebSocket.
type Server struct {
	hub        *ShardedHub
	opts       UIOptions
	mu         sync.Mutex
	wsSessions map[*websocket.Conn]struct{}
	draining   atomic.Bool
}

func NewServer(hub *ShardedHub, opts UIOptions) *Server {
	return &Server{
		hub:        hub,
		opts:       opts,
		wsSessions: make(map[*websocket.Conn]struct{}),
	}
}

func (s *Server) Drain() int {
	s.draining.Store(true)
	s.mu.Lock()
	defer s.mu.Unlock()
	count := len(s.wsSessions)
	for conn := range s.wsSessions {
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(4010, "server draining"), time.Now().Add(time.Second))
		_ = conn.Close()
	}
	return count
}

func (s *Server) ActiveSessions() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.wsSessions)
}

type websocketSession struct {
	conn     *websocket.Conn
	ctx      context.Context
	mu       sync.Mutex
	outbound chan []byte
	controls chan []byte
}

func (s *websocketSession) write(payload []byte, deadline time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.conn.SetWriteDeadline(deadline); err != nil {
		return err
	}
	return s.conn.WriteMessage(websocket.BinaryMessage, payload)
}

func (s *websocketSession) writeLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case message := <-s.controls:
			if err := s.write(message, time.Now().Add(5*time.Second)); err != nil {
				return
			}
		default:
		}
		select {
		case <-s.ctx.Done():
			return
		case message := <-s.controls:
			if err := s.write(message, time.Now().Add(5*time.Second)); err != nil {
				return
			}
		case message := <-s.outbound:
			if err := s.write(message, time.Now().Add(5*time.Second)); err != nil {
				return
			}
		}
	}
}

type websocketStream struct {
	session  *websocketSession
	kind, id byte
	reader   *io.PipeReader
	writer   *io.PipeWriter
	mu       sync.Mutex
	deadline time.Time
	timer    *time.Timer
	closed   bool
	incoming chan []byte
	done     chan struct{}
}

func newWebsocketStream(session *websocketSession, kind, id byte) *websocketStream {
	r, w := io.Pipe()
	stream := &websocketStream{session: session, kind: kind, id: id, reader: r, writer: w, incoming: make(chan []byte, 256), done: make(chan struct{})}
	go stream.feed()
	return stream
}

func (s *websocketStream) feed() {
	for {
		select {
		case <-s.done:
			return
		case data := <-s.incoming:
			if _, err := s.writer.Write(data); err != nil {
				return
			}
		}
	}
}

func (s *websocketStream) enqueue(data []byte) bool {
	copyData := append([]byte(nil), data...)
	select {
	case <-s.done:
		return false
	case s.incoming <- copyData:
		return true
	default:
		return false
	}
}

func (s *websocketStream) Read(p []byte) (int, error) { return s.reader.Read(p) }
func (s *websocketStream) Write(p []byte) (int, error) {
	s.mu.Lock()
	deadline, closed := s.deadline, s.closed
	s.mu.Unlock()
	if closed {
		return 0, io.ErrClosedPipe
	}
	if deadline.IsZero() {
		deadline = time.Now().Add(5 * time.Second)
	}
	message := make([]byte, 2+len(p))
	message[0], message[1] = s.kind, s.id
	copy(message[2:], p)
	if err := s.session.write(message, deadline); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (s *websocketStream) SetWriteDeadline(deadline time.Time) error {
	s.mu.Lock()
	s.deadline = deadline
	s.mu.Unlock()
	return nil
}

func (s *websocketStream) SetReadDeadline(deadline time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	if !deadline.IsZero() && !s.closed {
		s.timer = time.AfterFunc(time.Until(deadline), func() { _ = s.reader.CloseWithError(context.DeadlineExceeded) })
	}
	return nil
}

func (s *websocketStream) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	close(s.done)
	if s.timer != nil {
		s.timer.Stop()
	}
	s.mu.Unlock()
	_ = s.reader.Close()
	_ = s.writer.Close()
	select {
	case s.session.controls <- []byte{wsClose, s.kind, s.id}:
	default:
	}
	return nil
}

// HandleWebSocket upgrades HTTP to WebSocket and services UI events.
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request, playerID uuid.UUID, tabID string, checkOrigin func(*http.Request) bool) error {
	if s.draining.Load() {
		http.Error(w, "server draining", http.StatusServiceUnavailable)
		return nil
	}
	upgrader := websocket.Upgrader{CheckOrigin: checkOrigin, ReadBufferSize: 4096, WriteBufferSize: 4096}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.wsSessions[conn] = struct{}{}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.wsSessions, conn)
		s.mu.Unlock()
	}()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer conn.Close()

	client := &websocketSession{conn: conn, ctx: ctx, outbound: make(chan []byte, 1), controls: make(chan []byte, 16)}
	streams := make(map[[2]byte]*websocketStream)
	var workers sync.WaitGroup
	workers.Add(1)
	go func() {
		defer workers.Done()
		client.writeLoop()
	}()
	defer func() {
		for _, stream := range streams {
			_ = stream.Close()
		}
		cancel()
		workers.Wait()
	}()

	conn.SetReadLimit(MaxFrameSize + 3)
	for {
		kind, message, err := conn.ReadMessage()
		if err != nil {
			return nil
		}
		if kind != websocket.BinaryMessage || len(message) == 0 {
			continue
		}
		switch message[0] {
		case wsClose:
			if len(message) != 3 {
				continue
			}
			key := [2]byte{message[1], message[2]}
			if stream := streams[key]; stream != nil {
				_ = stream.Close()
				delete(streams, key)
			}
		case wsUI:
			if len(message) < 3 || message[1] == 0 {
				continue
			}
			key := [2]byte{message[0], message[1]}
			stream := streams[key]
			if stream == nil {
				if len(streams) >= 8 {
					return errors.New("too many active websocket streams")
				}
				stream = newWebsocketStream(client, key[0], key[1])
				streams[key] = stream
				workers.Add(1)
				go func(st *websocketStream) {
					defer workers.Done()
					defer st.Close()
					var header [1]byte
					if _, err := io.ReadFull(st, header[:]); err == nil && header[0] == StreamUI {
						s.ServeUIStream(ctx, st, playerID)
					}
				}(stream)
			}
			if !stream.enqueue(message[2:]) {
				_ = stream.Close()
				delete(streams, key)
			}
		default:
			// Ignore unsupported streams or datagrams
		}
	}
}

type uiStreamSink struct {
	playerID uuid.UUID
	hub      *ShardedHub
	out      chan []byte
	done     chan struct{}
	mu       sync.Mutex
	subs     map[ChannelName]*uiChannelSink
}

type uiChannelSink struct {
	stream  *uiStreamSink
	channel string
}

func (s *uiChannelSink) Deliver(payload []byte) {
	if s.stream != nil {
		s.stream.deliverChannel(s.channel, payload)
	}
}

func newUIStreamSink(playerID uuid.UUID, hub *ShardedHub) *uiStreamSink {
	return &uiStreamSink{
		playerID: playerID,
		hub:      hub,
		out:      make(chan []byte, 256),
		done:     make(chan struct{}),
		subs:     make(map[ChannelName]*uiChannelSink),
	}
}

func (u *uiStreamSink) Deliver(payload []byte) {
	u.deliverChannel("", payload)
}

func (u *uiStreamSink) deliverChannel(channel string, payload []byte) {
	if len(payload) == 0 {
		return
	}
	line := ensureEventEnvelope(channel, payload)
	select {
	case <-u.done:
	case u.out <- line:
	default:
	}
}

func (u *uiStreamSink) sendRaw(line []byte) {
	if len(line) == 0 {
		return
	}
	if line[len(line)-1] != '\n' {
		buf := make([]byte, len(line)+1)
		copy(buf, line)
		buf[len(line)] = '\n'
		line = buf
	}
	select {
	case <-u.done:
	case u.out <- line:
	default:
	}
}

func (u *uiStreamSink) subscribe(channel ChannelName) {
	if u.hub == nil || channel == "" {
		return
	}
	u.mu.Lock()
	chSink, exists := u.subs[channel]
	if !exists {
		chSink = &uiChannelSink{stream: u, channel: string(channel)}
		u.subs[channel] = chSink
	}
	u.mu.Unlock()
	u.hub.SubscribeSink(u.playerID, channel, chSink)
}

func (u *uiStreamSink) unsubscribe(channel ChannelName) {
	if u.hub == nil || channel == "" {
		return
	}
	u.mu.Lock()
	chSink := u.subs[channel]
	delete(u.subs, channel)
	u.mu.Unlock()
	if chSink != nil {
		u.hub.UnsubscribeSinkIf(u.playerID, channel, chSink)
	} else {
		u.hub.UnsubscribeSink(u.playerID, channel)
	}
}

func (u *uiStreamSink) closeAll() {
	close(u.done)
	if u.hub == nil {
		return
	}
	u.mu.Lock()
	subs := u.subs
	u.subs = make(map[ChannelName]*uiChannelSink)
	u.mu.Unlock()
	for ch, chSink := range subs {
		u.hub.UnsubscribeSinkIf(u.playerID, ch, chSink)
	}
	u.hub.UnsubscribeAllSinks(u.playerID)
}

type uiClientMessage struct {
	Type      string          `json:"type"`
	Channel   string          `json:"channel,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
}

// ServeUIStream processes NDJSON UI protocol on rw for playerID.
func (s *Server) ServeUIStream(ctx context.Context, rw io.ReadWriter, playerID uuid.UUID) {
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	sink := newUIStreamSink(playerID, s.hub)
	defer sink.closeAll()

	var writerWG sync.WaitGroup
	writerWG.Add(1)
	go func() {
		defer writerWG.Done()
		for {
			select {
			case <-streamCtx.Done():
				return
			case <-sink.done:
				return
			case line := <-sink.out:
				if deadliner, ok := rw.(interface{ SetWriteDeadline(time.Time) error }); ok {
					_ = deadliner.SetWriteDeadline(time.Now().Add(5 * time.Second))
				}
				if _, err := rw.Write(line); err != nil {
					cancel()
					return
				}
			}
		}
	}()
	defer writerWG.Wait()

	if s.opts.TouchActivity != nil {
		s.opts.TouchActivity(playerID)
	}

	idStr := playerID.String()
	for _, ch := range []string{
		"player:" + idStr,
		"refine:" + idStr,
		"trade:" + idStr,
		"notif:" + idStr,
		"combat:" + idStr,
		"chat:world",
		"chat:general",
		"chat:private:" + idStr,
		"death_arena",
		"martial_arena",
		"catalog",
		"world:cell",
		"world:drops",
		"world:points",
	} {
		sink.subscribe(ChannelName(ch))
	}
	if s.opts.SectLookup != nil {
		if sectID := strings.TrimSpace(s.opts.SectLookup(playerID)); sectID != "" {
			sink.subscribe(ChannelName("chat:sect:" + sectID))
		}
	}

	if s.opts.InitialEvents != nil {
		go func() {
			for _, ev := range s.opts.InitialEvents(playerID) {
				sink.sendRaw(ev)
			}
		}()
	}

	scanner := bufio.NewScanner(rw)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var msg uiClientMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		if s.opts.TouchActivity != nil {
			s.opts.TouchActivity(playerID)
		}
		switch msg.Type {
		case "subscribe":
			ch := extractUIChannel(msg)
			if ch == "" || !s.allowUISubscribe(playerID, ch) {
				continue
			}
			for _, expanded := range expandUIChannels(playerID, ch) {
				sink.subscribe(expanded)
			}
		case "unsubscribe":
			ch := extractUIChannel(msg)
			if ch == "" {
				continue
			}
			for _, expanded := range expandUIChannels(playerID, ch) {
				sink.unsubscribe(expanded)
			}
		case "ping":
			sink.sendRaw([]byte(`{"type":"pong"}`))
		case "command":
			if strings.TrimSpace(msg.RequestID) == "" {
				continue
			}
			status := http.StatusNotImplemented
			result := json.RawMessage(`{"error":"UI commands are not configured"}`)
			if s.opts.Command != nil {
				payload := msg.Payload
				if len(payload) == 0 {
					payload = msg.Data
				}
				if len(payload) == 0 {
					payload = json.RawMessage(`{}`)
				}
				status, result = s.opts.Command(playerID, msg.Channel, payload)
			}
			if status < 100 || status > 599 {
				status = http.StatusInternalServerError
				result = json.RawMessage(`{"error":"Invalid command response"}`)
			}
			if len(result) == 0 || !json.Valid(result) {
				result = json.RawMessage(`{"error":"Invalid command response"}`)
			}
			resp, err := json.Marshal(struct {
				Type      string          `json:"type"`
				RequestID string          `json:"request_id,omitempty"`
				OK        bool            `json:"ok"`
				Status    int             `json:"status"`
				Data      json.RawMessage `json:"data"`
			}{
				Type:      "command.result",
				RequestID: msg.RequestID,
				OK:        status >= http.StatusOK && status < http.StatusMultipleChoices,
				Status:    status,
				Data:      result,
			})
			if err == nil {
				sink.sendRaw(resp)
			}
		}
	}
}

func extractUIChannel(msg uiClientMessage) string {
	if ch := strings.TrimSpace(msg.Channel); ch != "" {
		return ch
	}
	for _, raw := range []json.RawMessage{msg.Data, msg.Payload} {
		if len(raw) == 0 {
			continue
		}
		var sub struct {
			Channel string `json:"channel"`
		}
		if err := json.Unmarshal(raw, &sub); err == nil && strings.TrimSpace(sub.Channel) != "" {
			return strings.TrimSpace(sub.Channel)
		}
	}
	return ""
}

func expandUIChannels(playerID uuid.UUID, channel string) []ChannelName {
	idStr := playerID.String()
	switch channel {
	case "player":
		return []ChannelName{ChannelName("player:" + idStr)}
	case "refine":
		return []ChannelName{ChannelName("refine:" + idStr)}
	case "trade":
		return []ChannelName{ChannelName("trade:" + idStr)}
	case "notification":
		return []ChannelName{ChannelName("notif:" + idStr)}
	case "combat":
		return []ChannelName{ChannelName("combat:" + idStr)}
	default:
		return []ChannelName{ChannelName(channel)}
	}
}

func (s *Server) allowUISubscribe(playerID uuid.UUID, channel string) bool {
	if channel == "chat:private" {
		return false
	}
	idStr := playerID.String()
	for _, prefix := range []string{"player:", "refine:", "trade:", "notif:", "combat:", "chat:private:"} {
		if strings.HasPrefix(channel, prefix) {
			return strings.TrimPrefix(channel, prefix) == idStr
		}
	}
	if strings.HasPrefix(channel, "chat:sect:") && s.opts.SectLookup != nil {
		requested := strings.TrimPrefix(channel, "chat:sect:")
		actual := strings.TrimSpace(s.opts.SectLookup(playerID))
		return requested != "" && requested == actual
	}
	if strings.HasPrefix(channel, "chat:region:") && s.opts.RegionScope != nil {
		requested := strings.TrimPrefix(channel, "chat:region:")
		actual, err := s.opts.RegionScope(playerID)
		return err == nil && requested != "" && requested == actual
	}
	return true
}

var eventEnvelopePrefix = []byte(`{"type":"event","channel":"`)

func ensureEventEnvelope(channel string, payload []byte) []byte {
	trimmed := bytes.TrimSpace(payload)
	if bytes.HasPrefix(trimmed, eventEnvelopePrefix) && json.Valid(trimmed) {
		out := make([]byte, len(trimmed)+1)
		copy(out, trimmed)
		out[len(trimmed)] = '\n'
		return out
	}
	var probe struct {
		Type    string          `json:"type"`
		Channel string          `json:"channel"`
		Data    json.RawMessage `json:"data"`
	}
	if bytes.Contains(trimmed, []byte(`"type"`)) && json.Unmarshal(trimmed, &probe) == nil && probe.Type == "event" && probe.Channel != "" {
		out := make([]byte, len(trimmed)+1)
		copy(out, trimmed)
		out[len(trimmed)] = '\n'
		return out
	}
	data := json.RawMessage(trimmed)
	if !json.Valid(trimmed) {
		marshaled, _ := json.Marshal(string(trimmed))
		data = marshaled
	}
	wrapped, err := json.Marshal(struct {
		Type    string          `json:"type"`
		Channel string          `json:"channel"`
		Data    json.RawMessage `json:"data"`
	}{
		Type:    "event",
		Channel: channel,
		Data:    data,
	})
	if err != nil {
		out := make([]byte, len(trimmed)+1)
		copy(out, trimmed)
		out[len(trimmed)] = '\n'
		return out
	}
	return append(wrapped, '\n')
}
