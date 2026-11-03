package wttransport

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
	"github.com/vippro67/GuWorld/backend/internal/realtime/protocol"
	v2 "github.com/vippro67/GuWorld/backend/internal/realtime/protocol/v2"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
	"github.com/vippro67/GuWorld/backend/internal/uihub"
)

type fakeUIHub struct {
	mu    sync.RWMutex
	sinks map[uihub.ChannelName]map[uuid.UUID]uihub.MessageSink
}

func newFakeUIHub() *fakeUIHub {
	return &fakeUIHub{sinks: make(map[uihub.ChannelName]map[uuid.UUID]uihub.MessageSink)}
}

func (f *fakeUIHub) SubscribeSink(playerID uuid.UUID, channel uihub.ChannelName, sink uihub.MessageSink) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.sinks[channel] == nil {
		f.sinks[channel] = make(map[uuid.UUID]uihub.MessageSink)
	}
	f.sinks[channel][playerID] = sink
}

func (f *fakeUIHub) UnsubscribeSink(playerID uuid.UUID, channel uihub.ChannelName) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if subs, ok := f.sinks[channel]; ok {
		delete(subs, playerID)
		if len(subs) == 0 {
			delete(f.sinks, channel)
		}
	}
}

func (f *fakeUIHub) UnsubscribeAllSinks(playerID uuid.UUID) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for ch, subs := range f.sinks {
		delete(subs, playerID)
		if len(subs) == 0 {
			delete(f.sinks, ch)
		}
	}
}

func (f *fakeUIHub) Broadcast(channel uihub.ChannelName, payload []byte) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, sink := range f.sinks[channel] {
		sink.Deliver(payload)
	}
}

func (f *fakeUIHub) HasSubscriber(playerID uuid.UUID, channel uihub.ChannelName) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	_, ok := f.sinks[channel][playerID]
	return ok
}

func TestUIStreamSubscribeUnsubscribePingAndCommand(t *testing.T) {
	hub := newFakeUIHub()
	playerID := uuid.New()
	otherPlayerID := uuid.New()

	srv := New(nil, nil).WithUIOptions(UIOptions{
		Hub:        hub,
		SectLookup: func(uuid.UUID) string { return "sect-lotus" },
	})

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.ServeUIStream(ctx, serverConn, playerID)
		_ = serverConn.Close()
	}()

	reader := bufio.NewReader(clientConn)
	readLine := func() map[string]any {
		t.Helper()
		_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		line, err := reader.ReadBytes('\n')
		if err != nil {
			t.Fatalf("read line: %v", err)
		}
		var out map[string]any
		if err := json.Unmarshal(line, &out); err != nil {
			t.Fatalf("unmarshal %q: %v", string(line), err)
		}
		return out
	}

	// Wait until auto-subscriptions are registered.
	deadline := time.Now().Add(time.Second)
	for !hub.HasSubscriber(playerID, uihub.ChannelName("chat:sect:sect-lotus")) {
		if time.Now().After(deadline) {
			t.Fatal("auto-subscription did not complete")
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Ping -> Pong.
	if _, err := clientConn.Write([]byte(`{"type":"ping"}` + "\n")); err != nil {
		t.Fatal(err)
	}
	if got := readLine(); got["type"] != "pong" {
		t.Fatalf("pong = %+v", got)
	}

	// Command -> error use HTTP.
	if _, err := clientConn.Write([]byte(`{"type":"command","channel":"world.drops","request_id":"req-42"}` + "\n")); err != nil {
		t.Fatal(err)
	}
	cmdResp := readLine()
	if cmdResp["type"] != "error" || cmdResp["request_id"] != "req-42" || cmdResp["message"] != "use HTTP" {
		t.Fatalf("command response = %+v", cmdResp)
	}

	// Subscribe custom channel -> receives broadcast.
	if _, err := clientConn.Write([]byte(`{"type":"subscribe","channel":"custom:test"}` + "\n")); err != nil {
		t.Fatal(err)
	}
	for !hub.HasSubscriber(playerID, uihub.ChannelName("custom:test")) {
		if time.Now().After(deadline) {
			t.Fatal("custom:test subscription did not complete")
		}
		time.Sleep(5 * time.Millisecond)
	}
	hub.Broadcast(uihub.ChannelName("custom:test"), []byte(`{"hello":"world"}`))
	ev := readLine()
	if ev["type"] != "event" || ev["channel"] != "custom:test" {
		t.Fatalf("event = %+v", ev)
	}

	// Unauthorized private channel for another player is ignored.
	if _, err := clientConn.Write([]byte(`{"type":"subscribe","channel":"chat:private:` + otherPlayerID.String() + `"}` + "\n")); err != nil {
		t.Fatal(err)
	}

	// Unsubscribe custom:test -> stops receiving events on custom:test, still receives auto-subscribed notif.
	if _, err := clientConn.Write([]byte(`{"type":"unsubscribe","channel":"custom:test"}` + "\n")); err != nil {
		t.Fatal(err)
	}
	unsubDeadline := time.Now().Add(time.Second)
	for hub.HasSubscriber(playerID, uihub.ChannelName("custom:test")) {
		if time.Now().After(unsubDeadline) {
			t.Fatal("custom:test unsubscription did not complete")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if hub.HasSubscriber(playerID, uihub.ChannelName("chat:private:"+otherPlayerID.String())) {
		t.Fatal("should not allow subscribing to another player's private chat channel")
	}

	hub.Broadcast(uihub.ChannelName("custom:test"), []byte(`{"ignored":true}`))
	hub.Broadcast(uihub.ChannelName("notif:"+playerID.String()), []byte(`{"type":"event","channel":"notification","data":{"title":"Tiên báo"}}`))
	notifEv := readLine()
	if notifEv["type"] != "event" || notifEv["channel"] != "notification" {
		t.Fatalf("expected notification event, got %+v", notifEv)
	}

	// Closing client stream unsubscribes all sinks.
	_ = clientConn.Close()
	<-done
	if hub.HasSubscriber(playerID, uihub.ChannelName("notif:"+playerID.String())) {
		t.Fatal("expected all sinks to be unsubscribed after stream close")
	}
}

func TestWebTransportSharedSessionWorldAndUIStreams(t *testing.T) {
	shub := uihub.InitShardedHub()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = shub.Stop(ctx)
	}()

	playerID := uuid.New()
	fake := &fakeAdmission{playerID: playerID, feed: make(chan sim.Snapshot, 4), inputs: make(chan protocol.Input, 4)}
	cert := testCertificate(t)
	mux := http.NewServeMux()

	h3 := &http3.Server{
		Handler:   mux,
		TLSConfig: http3.ConfigureTLSConfig(&tls.Config{Certificates: []tls.Certificate{cert}}),
		QUICConfig: &quic.Config{
			EnableDatagrams:                  true,
			EnableStreamResetPartialDelivery: true,
		},
	}
	wtServer := &webtransport.Server{H3: h3, CheckOrigin: func(*http.Request) bool { return true }}
	handler := New(fake, wtServer).WithUIHub(shub)
	mux.HandleFunc("/wt", func(w http.ResponseWriter, r *http.Request) {
		handler.Handle(w, r, playerID, "tab-shared")
	})

	udp, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = udp.Close(); _ = wtServer.Close() })
	go func() { _ = wtServer.Serve(udp) }()
	port := udp.LocalAddr().(*net.UDPAddr).Port

	transport := &webtransport.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		QUICConfig: &quic.Config{
			EnableDatagrams:                  true,
			EnableStreamResetPartialDelivery: true,
		},
	}
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, session, err := transport.Dial(ctx, "https://127.0.0.1:"+itoa(port)+"/wt", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer session.CloseWithError(0, "test done")

	// 1. Open UI stream (0x02) on the session.
	uiStream, err := session.OpenStreamSync(ctx)
	if err != nil {
		t.Fatalf("open ui stream: %v", err)
	}
	if _, err := uiStream.Write([]byte{v2.StreamUI}); err != nil {
		t.Fatalf("write ui stream header: %v", err)
	}
	if _, err := uiStream.Write([]byte(`{"type":"ping"}` + "\n")); err != nil {
		t.Fatalf("write ui ping: %v", err)
	}
	uiReader := bufio.NewReader(uiStream)
	line, err := uiReader.ReadBytes('\n')
	if err != nil {
		t.Fatalf("read ui pong: %v", err)
	}
	var pong map[string]any
	if err := json.Unmarshal(line, &pong); err != nil || pong["type"] != "pong" {
		t.Fatalf("unexpected ui pong: %s (%v)", string(line), err)
	}

	// 2. Open World stream (0x01) on the SAME session.
	worldStream, err := session.OpenStreamSync(ctx)
	if err != nil {
		t.Fatalf("open world stream: %v", err)
	}
	joinPayload, err := v2.EncodeJoin(v2.JoinRequest{ZoneID: "forest", MapVersion: "map-1", ContentVersion: "content-1"})
	if err != nil {
		t.Fatal(err)
	}
	joinFrame, err := v2.EncodeFrame(v2.OpJoin, joinPayload)
	if err != nil {
		t.Fatal(err)
	}
	handshakeAndJoin := append([]byte{v2.StreamWorld, 'G', 'W', 'T', v2.Version}, joinFrame...)
	if _, err := worldStream.Write(handshakeAndJoin); err != nil {
		t.Fatalf("write world join: %v", err)
	}
	op, _, err := readFrame(worldStream)
	if err != nil || op != v2.OpWelcome {
		t.Fatalf("world welcome op=%d err=%v", op, err)
	}

	// 3. Broadcast chat & trade via ShardedHub -> received over UI stream.
	shub.BroadcastToChannel(uihub.ChannelName("chat:world"), []byte(`{"type":"event","channel":"chat:world","data":{"text":"Chào đạo hữu"}}`), uuid.Nil)
	chatLine, err := uiReader.ReadBytes('\n')
	if err != nil {
		t.Fatalf("read chat event: %v", err)
	}
	var chatEv struct {
		Type    string `json:"type"`
		Channel string `json:"channel"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(chatLine, &chatEv); err != nil || chatEv.Channel != "chat:world" || chatEv.Data.Text != "Chào đạo hữu" {
		t.Fatalf("unexpected chat event: %s (%v)", string(chatLine), err)
	}

	shub.BroadcastToChannel(uihub.ChannelName("trade:"+playerID.String()), []byte(`{"type":"event","channel":"trade","data":{"id":"trade-99"}}`), uuid.Nil)
	tradeLine, err := uiReader.ReadBytes('\n')
	if err != nil {
		t.Fatalf("read trade event: %v", err)
	}
	var tradeEv struct {
		Type    string `json:"type"`
		Channel string `json:"channel"`
	}
	if err := json.Unmarshal(tradeLine, &tradeEv); err != nil || tradeEv.Channel != "trade" {
		t.Fatalf("unexpected trade event: %s (%v)", string(tradeLine), err)
	}
}
