package wttransport

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"math/big"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
	"github.com/vippro67/GuWorld/backend/internal/realtime/protocol"
	v2 "github.com/vippro67/GuWorld/backend/internal/realtime/protocol/v2"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
)

type fakeAdmission struct {
	playerID uuid.UUID
	feed     chan sim.Snapshot
	inputs   chan protocol.Input
	joined   bool
}

func (f *fakeAdmission) JoinWithTab(_ context.Context, playerID, _ uuid.UUID, _ string, _ protocol.JoinRequest) (any, error) {
	f.joined = true
	snapshot := protocol.Snapshot{
		Tick: 10, BaselineID: 1, BaseTick: 10, IsBaseline: true,
		Entities: []protocol.EntityState{{
			EntityID: playerID.String(), Name: "Tester", Kind: "player",
			X: 10, Y: 20, HP: 100, MaxHP: 100, Mana: 50, MaxMana: 100, Revision: 1,
		}},
	}
	raw, _ := json.Marshal(snapshot)
	return protocol.Welcome{
		SessionID: "session-1", ZoneID: "forest#realtime-preview",
		MapVersion: "map-1", ContentVersion: "content-1", TickRate: 20,
		MovementSpeed: 5, ServerTick: 10, EntityID: playerID.String(),
		EntityRevision: 1, BaselineID: 1, Snapshot: raw,
	}, nil
}

func (f *fakeAdmission) Input(_ uuid.UUID, _ uuid.UUID, input protocol.Input) error {
	select {
	case f.inputs <- input:
	default:
	}
	return nil
}

func (f *fakeAdmission) MoveTarget(uuid.UUID, uuid.UUID, protocol.MoveTarget) error { return nil }
func (f *fakeAdmission) Cast(uuid.UUID, uuid.UUID, protocol.Cast) error             { return nil }
func (f *fakeAdmission) BasicAttack(uuid.UUID, uuid.UUID, protocol.Attack) error    { return nil }
func (f *fakeAdmission) Capture(uuid.UUID, uuid.UUID, protocol.CaptureIntent) error { return nil }
func (f *fakeAdmission) CompanionStance(uuid.UUID, uuid.UUID, protocol.CompanionStanceIntent) error {
	return nil
}
func (f *fakeAdmission) SubscribeSnapshots(uuid.UUID, uuid.UUID) (<-chan sim.Snapshot, func(), error) {
	return f.feed, func() {}, nil
}
func (f *fakeAdmission) Leave(uuid.UUID, uuid.UUID) error { return nil }
func (f *fakeAdmission) Disconnect(uuid.UUID, uuid.UUID)  {}

func testCertificate(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "wt-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

func TestWebTransportJoinWelcomeInputAndSnapshot(t *testing.T) {
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
	server := &webtransport.Server{H3: h3, CheckOrigin: func(*http.Request) bool { return true }}
	mux.HandleFunc("/wt", func(w http.ResponseWriter, r *http.Request) {
		New(fake, server).Handle(w, r, playerID, "tab-a")
	})
	udp, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = udp.Close(); _ = server.Close() })
	go func() { _ = server.Serve(udp) }()
	port := udp.LocalAddr().(*net.UDPAddr).Port

	transport := &webtransport.Transport{
		ApplicationProtocols: []string{"gu.realtime.v1"},
		TLSClientConfig:      &tls.Config{InsecureSkipVerify: true},
		QUICConfig: &quic.Config{
			EnableDatagrams:                  true,
			EnableStreamResetPartialDelivery: true,
		},
	}
	defer transport.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	url := "https://127.0.0.1:" + itoa(port) + "/wt"
	response, session, err := transport.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", response.StatusCode)
	}
	defer session.CloseWithError(0, "test done")

	stream, err := session.OpenStreamSync(ctx)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	joinPayload, err := v2.EncodeJoin(v2.JoinRequest{ZoneID: "forest", MapVersion: "map-1", ContentVersion: "content-1"})
	if err != nil {
		t.Fatal(err)
	}
	joinFrame, err := v2.EncodeFrame(v2.OpJoin, joinPayload)
	if err != nil {
		t.Fatal(err)
	}
	handshakeAndJoin := append(append([]byte{}, []byte{v2.StreamWorld, 'G', 'W', 'T', v2.Version}...), joinFrame...)
	written, err := stream.Write(handshakeAndJoin)
	if err != nil {
		t.Fatalf("join write: %v", err)
	}
	if written != len(handshakeAndJoin) {
		t.Fatalf("short write: %d/%d", written, len(handshakeAndJoin))
	}

	op, payload, err := readFrame(stream)
	if err != nil {
		t.Fatalf("welcome frame: %v", err)
	}
	if op != v2.OpWelcome {
		t.Fatalf("op = %d, want welcome", op)
	}
	welcome, err := v2.DecodeWelcome(payload)
	if err != nil {
		t.Fatalf("welcome decode: %v", err)
	}
	if welcome.ZoneID != "forest#realtime-preview" || welcome.TickRate != 20 || welcome.EntityID != playerID.String() {
		t.Fatalf("welcome = %+v", welcome)
	}
	full, err := v2.DecodeSnapshotFull(welcome.Snapshot)
	if err != nil {
		t.Fatalf("baseline decode: %v", err)
	}
	if len(full.Spawns) != 1 || full.Spawns[0].EntityID != playerID.String() || full.Spawns[0].NetID == 0 {
		t.Fatalf("baseline spawns = %+v", full.Spawns)
	}

	// Input datagram reaches admission.
	input, err := v2.EncodeInput(v2.Input{Seq: 7, MoveX: 127, MoveY: 0})
	if err != nil {
		t.Fatal(err)
	}
	if err := session.SendDatagram(input); err != nil {
		t.Fatalf("send input: %v", err)
	}
	select {
	case got := <-fake.inputs:
		if got.Seq != 7 || got.Move == nil || got.Move.X != 1 {
			t.Fatalf("input = %+v", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("admission did not receive input")
	}

	// Snapshot feed produces a datagram delta.
	fake.feed <- sim.Snapshot{
		Tick: 11,
		Entities: []sim.Entity{{
			ID: sim.EntityID(playerID.String()), CharacterID: sim.CharacterID(playerID.String()),
			Kind: sim.EntityPlayer, Name: "Tester", Revision: 2,
			Transform: sim.Transform{Position: sim.Vec2{X: 11, Y: 20}},
			Vitals:    sim.Vitals{HP: 90, MaxHP: 100, Shield: 5},
			Combat:    sim.CombatState{Mana: 40, MaxMana: 100, Thought: 5, MaxThought: 10},
			AckSeq:    7,
		}},
	}
	datagram, err := session.ReceiveDatagram(ctx)
	if err != nil {
		t.Fatalf("receive datagram: %v", err)
	}
	opcode, body, err := v2.DecodeDatagram(datagram)
	if err != nil || opcode != v2.OpSnapshotDelta {
		t.Fatalf("datagram op=%d err=%v", opcode, err)
	}
	delta, err := v2.DecodeSnapshotDelta(body)
	if err != nil {
		t.Fatalf("delta decode: %v", err)
	}
	if delta.Tick != 11 || delta.AckSeq != 7 || len(delta.Entities) != 1 {
		t.Fatalf("delta = %+v", delta)
	}
	entity := delta.Entities[0]
	if entity.HP != 90 || entity.Shield != 5 || entity.Mana != 40 || dequantized(entity.X) != 11 {
		t.Fatalf("entity delta = %+v", entity)
	}
}

func dequantized(value uint16) float64 { return v2.Dequantize(value) }

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	buf := make([]byte, 0, 6)
	for value > 0 {
		buf = append([]byte{byte('0' + value%10)}, buf...)
		value /= 10
	}
	return string(buf)
}
