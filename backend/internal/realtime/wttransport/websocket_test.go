package wttransport

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/vippro67/GuWorld/backend/internal/realtime/protocol"
	v2 "github.com/vippro67/GuWorld/backend/internal/realtime/protocol/v2"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
)

type pipeListener struct {
	conn net.Conn
	once sync.Once
	done chan struct{}
}

func (p *pipeListener) Accept() (net.Conn, error) {
	var conn net.Conn
	p.once.Do(func() { conn = p.conn })
	if conn != nil {
		return conn, nil
	}
	<-p.done
	return nil, net.ErrClosed
}
func (p *pipeListener) Close() error   { close(p.done); return nil }
func (p *pipeListener) Addr() net.Addr { return &net.TCPAddr{} }

func TestWebSocketDatagramQueueKeepsNewestAndForcesBaseline(t *testing.T) {
	session := &websocketSession{ctx: context.Background(), outbound: make(chan []byte, 1)}
	if err := session.SendDatagram([]byte{1}); err != nil {
		t.Fatal(err)
	}
	if err := session.SendDatagram([]byte{2}); err != nil {
		t.Fatal(err)
	}
	if !session.needsResync.Load() {
		t.Fatal("dropped delta must trigger a full baseline")
	}
	got := <-session.outbound
	if len(got) != 2 || got[0] != wsDgram || got[1] != 2 {
		t.Fatalf("queued datagram = %v, want newest", got)
	}
}

func TestWebSocketMultiplexesChunkedWorldAndUI(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	listener := &pipeListener{conn: serverConn, done: make(chan struct{})}
	playerID := uuid.New()
	fake := &fakeAdmission{feed: make(chan sim.Snapshot, 4), inputs: make(chan protocol.Input, 4)}
	server := New(fake, nil).WithUIOptions(UIOptions{Hub: newFakeUIHub()})
	httpServer := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = server.HandleWebSocket(w, r, playerID, "tab-a", func(*http.Request) bool { return true })
	})}
	go func() { _ = httpServer.Serve(listener) }()
	t.Cleanup(func() { _ = clientConn.Close(); _ = serverConn.Close(); _ = httpServer.Close() })
	dialer := websocket.Dialer{NetDialContext: func(context.Context, string, string) (net.Conn, error) { return clientConn, nil }}
	conn, _, err := dialer.Dial("ws://example/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	join, err := v2.EncodeJoin(v2.JoinRequest{ZoneID: "forest", MapVersion: "map-1", ContentVersion: "content-1"})
	if err != nil {
		t.Fatal(err)
	}
	frame, err := v2.EncodeFrame(v2.OpJoin, join)
	if err != nil {
		t.Fatal(err)
	}
	world := append([]byte{v2.StreamWorld, 'G', 'W', 'T', v2.Version}, frame...)
	for _, part := range [][]byte{append([]byte{wsWorld, 7}, world[:3]...), {wsUI, 9, v2.StreamUI}, append([]byte{wsWorld, 7}, world[3:]...), append([]byte{wsUI, 9}, []byte("{\"type\":\"ping\"}\n")...)} {
		if err := conn.WriteMessage(websocket.BinaryMessage, part); err != nil {
			t.Fatal(err)
		}
	}
	welcome, pong := false, false
	for i := 0; i < 8 && (!welcome || !pong); i++ {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, message, err := conn.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		if len(message) < 3 {
			continue
		}
		if message[0] == wsWorld && message[1] == 7 {
			op, _, err := readFrame(bytes.NewReader(message[2:]))
			if err == nil && op == v2.OpWelcome {
				welcome = true
			}
		}
		if message[0] == wsUI && message[1] == 9 && bytes.Contains(message[2:], []byte(`"pong"`)) {
			pong = true
		}
	}
	if !welcome || !pong {
		t.Fatalf("welcome=%v pong=%v", welcome, pong)
	}
	if err := conn.WriteMessage(websocket.BinaryMessage, []byte{wsClose, wsUI, 9}); err != nil {
		t.Fatal(err)
	}
	input, err := v2.EncodeInput(v2.Input{Seq: 42, MoveX: 127})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteMessage(websocket.BinaryMessage, append([]byte{wsDgram}, input...)); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-fake.inputs:
		if got.Seq != 42 {
			t.Fatalf("input seq=%d", got.Seq)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("world input stopped after UI close")
	}
	closedUI := false
	for i := 0; i < 4 && !closedUI; i++ {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, message, err := conn.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("after UI close: %v", message)
		closedUI = bytes.Equal(message, []byte{wsClose, wsUI, 9})
	}
	if !closedUI {
		t.Fatal("server did not close UI stream")
	}
	oversized := make([]byte, v2.MaxFrameSize+4)
	oversized[0], oversized[1] = wsWorld, 7
	if err := conn.WriteMessage(websocket.BinaryMessage, oversized); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(7 * time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("oversized WebSocket message was accepted")
	}
}
