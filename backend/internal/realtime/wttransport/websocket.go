package wttransport

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/vippro67/GuWorld/backend/internal/realtime/protocol/v2"
)

// WebSocket envelopes preserve the existing world and UI stream protocols.
// A stream message is [kind, streamID, bytes...], a datagram is [3, bytes...],
// and a stream close is [4, kind, streamID].
const (
	wsWorld byte = 1
	wsUI    byte = 2
	wsDgram byte = 3
	wsClose byte = 4
)

type websocketSession struct {
	conn        *websocket.Conn
	ctx         context.Context
	mu          sync.Mutex
	datagrams   chan []byte
	outbound    chan []byte
	controls    chan []byte
	needsResync atomic.Bool
}

func (s *websocketSession) write(payload []byte, deadline time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.conn.SetWriteDeadline(deadline); err != nil {
		return err
	}
	return s.conn.WriteMessage(websocket.BinaryMessage, payload)
}

func (s *websocketSession) SendDatagram(data []byte) error {
	message := append([]byte{wsDgram}, data...)
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
	}
	select {
	case s.outbound <- message:
		return nil
	default:
		// A newer snapshot supersedes an unsent delta. The next baseline
		// repairs gaps if the displaced datagram carried unique state.
		select {
		case <-s.outbound:
			s.needsResync.Store(true)
		default:
		}
		select {
		case s.outbound <- message:
		default:
		}
		return nil
	}
}

func (s *websocketSession) writeDatagrams() {
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

func (s *websocketSession) ReceiveDatagram(ctx context.Context) ([]byte, error) {
	select {
	case data := <-s.datagrams:
		return data, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
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
	stream := &websocketStream{session: session, kind: kind, id: id, reader: r, writer: w, incoming: make(chan []byte, 8), done: make(chan struct{})}
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

// HandleWebSocket runs the same world and UI protocols over a single TCP
// connection. Authentication and the active-tab claim happen before upgrade.
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request, playerID uuid.UUID, tabID string, checkOrigin func(*http.Request) bool) error {
	if s.admission == nil {
		return errors.New("realtime admission unavailable")
	}
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
	if s.wsSessions == nil {
		s.wsSessions = make(map[*websocket.Conn]struct{})
	}
	s.wsSessions[conn] = struct{}{}
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.wsSessions, conn); s.mu.Unlock() }()
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer conn.Close()
	client := &websocketSession{conn: conn, ctx: ctx, datagrams: make(chan []byte, 64), outbound: make(chan []byte, 1), controls: make(chan []byte, 16)}
	streams := make(map[[2]byte]*websocketStream)
	var workers sync.WaitGroup
	workers.Add(1)
	go func() { defer workers.Done(); client.writeDatagrams() }()
	defer func() {
		for _, stream := range streams {
			_ = stream.Close()
		}
		cancel()
		workers.Wait()
	}()
	conn.SetReadLimit(v2.MaxFrameSize + 3)
	for {
		kind, message, err := conn.ReadMessage()
		if err != nil {
			return nil
		}
		if kind != websocket.BinaryMessage || len(message) == 0 {
			continue
		}
		switch message[0] {
		case wsDgram:
			if len(message) < 2 {
				continue
			}
			payload := append([]byte(nil), message[1:]...)
			select {
			case client.datagrams <- payload:
			default:
				select {
				case <-client.datagrams:
				default:
				}
				select {
				case client.datagrams <- payload:
				default:
				}
			}
		case wsClose:
			if len(message) != 3 {
				continue
			}
			key := [2]byte{message[1], message[2]}
			if stream := streams[key]; stream != nil {
				_ = stream.Close()
				delete(streams, key)
			}
		case wsWorld, wsUI:
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
				if key[0] == wsWorld {
					go func(st *websocketStream) {
						defer workers.Done()
						var header [1]byte
						if _, err := io.ReadFull(st, header[:]); err != nil || header[0] != v2.StreamWorld {
							_ = st.Close()
							return
						}
						s.serveWorldStream(ctx, client, st, playerID, uuid.New(), tabID)
					}(stream)
				} else {
					go func(st *websocketStream) {
						defer workers.Done()
						defer st.Close()
						var header [1]byte
						if _, err := io.ReadFull(st, header[:]); err == nil && header[0] == v2.StreamUI {
							s.ServeUIStream(ctx, st, playerID)
						}
					}(stream)
				}
			}
			if !stream.enqueue(message[2:]) {
				_ = stream.Close()
				delete(streams, key)
			}
		}
	}
}
