package wttransport

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/quic-go/webtransport-go"
	"github.com/vippro67/GuWorld/backend/internal/uihub"
)

// UIHub is the channel subscription surface required by the WebTransport UI
// stream. Both *uihub.Hub and *uihub.ShardedHub implement it.
type UIHub interface {
	SubscribeSink(playerID uuid.UUID, channel uihub.ChannelName, sink uihub.MessageSink)
	UnsubscribeSink(playerID uuid.UUID, channel uihub.ChannelName)
	UnsubscribeAllSinks(playerID uuid.UUID)
}

type conditionalUIHub interface {
	UnsubscribeSinkIf(playerID uuid.UUID, channel uihub.ChannelName, sink uihub.MessageSink)
}

// UIOptions configures optional domain hooks for the WebTransport UI stream.
type UIOptions struct {
	Hub           UIHub
	SectLookup    func(playerID uuid.UUID) string
	RegionScope   func(playerID uuid.UUID) (string, error)
	InitialEvents func(playerID uuid.UUID) [][]byte
	TouchActivity func(playerID uuid.UUID)
}

// WithUIHub attaches a channel hub so stream 0x02 (StreamUI) can subscribe and
// receive broadcast events.
func (s *Server) WithUIHub(hub UIHub) *Server {
	s.ui.Hub = hub
	return s
}

// WithUIOptions attaches domain callbacks for the WebTransport UI stream.
func (s *Server) WithUIOptions(opts UIOptions) *Server {
	s.ui = opts
	return s
}

type uiClientMessage struct {
	Type      string          `json:"type"`
	Channel   string          `json:"channel,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
}

type uiChannelSink struct {
	stream  *uiStreamSink
	channel string
}

func (c *uiChannelSink) Deliver(payload []byte) {
	c.stream.deliverChannel(c.channel, payload)
}

type uiStreamSink struct {
	playerID uuid.UUID
	hub      UIHub
	out      chan []byte
	done     chan struct{}
	mu       sync.Mutex
	subs     map[uihub.ChannelName]*uiChannelSink
}

func newUIStreamSink(playerID uuid.UUID, hub UIHub) *uiStreamSink {
	return &uiStreamSink{
		playerID: playerID,
		hub:      hub,
		out:      make(chan []byte, 256),
		done:     make(chan struct{}),
		subs:     make(map[uihub.ChannelName]*uiChannelSink),
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

func (u *uiStreamSink) subscribe(channel uihub.ChannelName) {
	if u.hub == nil || channel == "" {
		return
	}
	u.mu.Lock()
	chSink, exists := u.subs[channel]
	if !exists {
		chSink = &uiChannelSink{stream: u, channel: normalizeClientChannel(string(channel))}
		u.subs[channel] = chSink
	}
	u.mu.Unlock()
	u.hub.SubscribeSink(u.playerID, channel, chSink)
}

func (u *uiStreamSink) unsubscribe(channel uihub.ChannelName) {
	if u.hub == nil || channel == "" {
		return
	}
	u.mu.Lock()
	chSink := u.subs[channel]
	delete(u.subs, channel)
	u.mu.Unlock()
	if cond, ok := u.hub.(conditionalUIHub); ok && chSink != nil {
		cond.UnsubscribeSinkIf(u.playerID, channel, chSink)
		return
	}
	u.hub.UnsubscribeSink(u.playerID, channel)
}

func (u *uiStreamSink) closeAll() {
	close(u.done)
	if u.hub == nil {
		return
	}
	u.mu.Lock()
	subs := u.subs
	u.subs = make(map[uihub.ChannelName]*uiChannelSink)
	u.mu.Unlock()
	if cond, ok := u.hub.(conditionalUIHub); ok {
		for ch, chSink := range subs {
			cond.UnsubscribeSinkIf(u.playerID, ch, chSink)
		}
		return
	}
	for ch := range subs {
		u.hub.UnsubscribeSink(u.playerID, ch)
	}
	u.hub.UnsubscribeAllSinks(u.playerID)
}

func (s *Server) serveUIStream(ctx context.Context, stream *webtransport.Stream, playerID uuid.UUID) {
	defer func() { _ = stream.Close() }()
	s.ServeUIStream(ctx, stream, playerID)
}

// ServeUIStream runs the NDJSON UI protocol on an already-opened stream after
// the initial 0x02 StreamUI byte has been consumed.
func (s *Server) ServeUIStream(ctx context.Context, rw io.ReadWriter, playerID uuid.UUID) {
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	sink := newUIStreamSink(playerID, s.ui.Hub)
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

	if s.ui.TouchActivity != nil {
		s.ui.TouchActivity(playerID)
	}

	// Auto-subscribe to the standard player-scoped and global UI channels.
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
		sink.subscribe(uihub.ChannelName(ch))
	}
	if s.ui.SectLookup != nil {
		if sectID := strings.TrimSpace(s.ui.SectLookup(playerID)); sectID != "" {
			sink.subscribe(uihub.ChannelName("chat:sect:" + sectID))
		}
	}

	if s.ui.InitialEvents != nil {
		go func() {
			for _, ev := range s.ui.InitialEvents(playerID) {
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
		if s.ui.TouchActivity != nil {
			s.ui.TouchActivity(playerID)
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
			resp, err := json.Marshal(struct {
				Type      string `json:"type"`
				RequestID string `json:"request_id,omitempty"`
				Message   string `json:"message"`
			}{
				Type:      "error",
				RequestID: msg.RequestID,
				Message:   "use HTTP",
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

func expandUIChannels(playerID uuid.UUID, channel string) []uihub.ChannelName {
	idStr := playerID.String()
	switch channel {
	case "player":
		return []uihub.ChannelName{uihub.ChannelName("player:" + idStr)}
	case "refine":
		return []uihub.ChannelName{uihub.ChannelName("refine:" + idStr)}
	case "trade":
		return []uihub.ChannelName{uihub.ChannelName("trade:" + idStr)}
	case "notification":
		return []uihub.ChannelName{uihub.ChannelName("notif:" + idStr)}
	case "combat":
		return []uihub.ChannelName{uihub.ChannelName("combat:" + idStr)}
	default:
		return []uihub.ChannelName{uihub.ChannelName(channel)}
	}
}

func normalizeClientChannel(redisChannel string) string {
	ch, _ := uihub.NormalizeClientChannel(redisChannel)
	return ch
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
	if strings.HasPrefix(channel, "chat:sect:") && s.ui.SectLookup != nil {
		requested := strings.TrimPrefix(channel, "chat:sect:")
		actual := strings.TrimSpace(s.ui.SectLookup(playerID))
		return requested != "" && requested == actual
	}
	if strings.HasPrefix(channel, "chat:region:") && s.ui.RegionScope != nil {
		requested := strings.TrimPrefix(channel, "chat:region:")
		actual, err := s.ui.RegionScope(playerID)
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
