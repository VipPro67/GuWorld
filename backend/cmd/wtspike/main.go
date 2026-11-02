// Command wtspike is a throwaway WebTransport-vs-WebSocket latency probe for
// ticket #310. It is intentionally outside the server build path: no auth, no
// game state, self-signed TLS, synthetic snapshots.
//
// Run: cd backend && go run ./cmd/wtspike -addr :9443 -rate 20 -size 220
// Open https://localhost:9443 and accept the self-signed certificate.
package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/gorilla/websocket"
	"github.com/libdns/cloudflare"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
)

//go:embed index.html
var indexHTML []byte

const (
	opSnapshot byte = 1
	opPong     byte = 2
	opPing     byte = 3
)

func main() {
	addr := flag.String("addr", ":9443", "TCP (HTTPS/WS) + UDP (QUIC/WT) listen address")
	rate := flag.Int("rate", 20, "synthetic snapshot rate in Hz")
	size := flag.Int("size", 220, "datagram/frame payload size in bytes")
	certFile := flag.String("cert", "", "TLS certificate file (PEM); empty = self-signed")
	keyFile := flag.String("key", "", "TLS private key file (PEM); empty = self-signed")
	acmeDomain := flag.String("acme-domain", "", "issue a Let's Encrypt cert for this domain via Cloudflare DNS-01 (requires CF_DNS_API_TOKEN)")
	acmeEmail := flag.String("acme-email", "", "ACME account email")
	acmeStorage := flag.String("acme-storage", "/tmp/wtspike-acme", "ACME certificate storage directory")
	acmeStaging := flag.Bool("acme-staging", false, "use Let's Encrypt staging (browsers will NOT trust it)")
	smoke := flag.Bool("smoke", false, "run an in-process WT client for a few seconds and exit")
	flag.Parse()

	var tcpTLS, h3TLS *tls.Config
	if *acmeDomain != "" {
		issuedTCP, issuedH3, err := acmeTLS(*acmeDomain, *acmeEmail, *acmeStorage, *acmeStaging)
		if err != nil {
			log.Fatalf("acme: %v", err)
		}
		tcpTLS, h3TLS = issuedTCP, issuedH3
		log.Printf("issued/loaded certificate for %s (open https://%s:%s)", *acmeDomain, *acmeDomain, strings.TrimPrefix(*addr, ":"))
	} else {
		cert := selfSignedCert()
		if *certFile != "" || *keyFile != "" {
			if *certFile == "" || *keyFile == "" {
				log.Fatal("-cert and -key must be provided together")
			}
			loaded, err := tls.LoadX509KeyPair(*certFile, *keyFile)
			if err != nil {
				log.Fatalf("load certificate: %v", err)
			}
			cert = loaded
		}
		tcpTLS = &tls.Config{Certificates: []tls.Certificate{cert}, NextProtos: []string{"h2", "http/1.1"}}
		h3TLS = http3.ConfigureTLSConfig(&tls.Config{Certificates: []tls.Certificate{cert}})
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexHTML)
	})
	mux.HandleFunc("/ws", wsHandler(*rate, *size))
	mux.HandleFunc("/delay", func(w http.ResponseWriter, r *http.Request) {
		ms, _ := strconv.Atoi(r.URL.Query().Get("ms"))
		if ms < 0 || ms > 10000 {
			ms = 1000
		}
		time.Sleep(time.Duration(ms) * time.Millisecond)
		w.Header().Set("Content-Type", "image/gif")
		_, _ = w.Write([]byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"))
	})

	h3 := &http3.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("H3 request: %s %s proto=%s", r.Method, r.URL.Path, r.Proto)
			mux.ServeHTTP(w, r)
		}),
		TLSConfig: h3TLS,
		QUICConfig: &quic.Config{
			MaxIdleTimeout:                   90 * time.Second,
			KeepAlivePeriod:                  20 * time.Second,
			MaxIncomingStreams:               16,
			MaxIncomingUniStreams:            16,
			MaxStreamReceiveWindow:           256 << 10,
			MaxConnectionReceiveWindow:       2 << 20,
			EnableDatagrams:                  true,
			EnableStreamResetPartialDelivery: true,
		},
	}
	wt := &webtransport.Server{
		H3:          h3,
		CheckOrigin: func(*http.Request) bool { return true },
	}
	mux.HandleFunc("/wt", func(w http.ResponseWriter, r *http.Request) {
		session, err := wt.Upgrade(w, r)
		if err != nil {
			log.Printf("WT upgrade failed: %v (method=%s proto=%s origin=%q ua=%q)", err, r.Method, r.Proto, r.Header.Get("Origin"), r.Header.Get("User-Agent"))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("WT session accepted from %s", r.RemoteAddr)
		reqRate, reqSize := requestTuning(r, *rate, *size)
		go serveWT(session, reqRate, reqSize)
	})

	udpAddr, err := net.ResolveUDPAddr("udp", *addr)
	if err != nil {
		log.Fatalf("resolve udp: %v", err)
	}
	udp, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatalf("listen udp: %v", err)
	}
	tcpSrv := &http.Server{Addr: *addr, Handler: mux, TLSConfig: tcpTLS}

	errCh := make(chan error, 2)
	go func() { errCh <- wt.Serve(udp) }()
	go func() { errCh <- tcpSrv.ListenAndServeTLS("", "") }()
	log.Printf("wtspike on %s (WT /wt, WS /ws, rate=%dHz size=%dB)", *addr, *rate, *size)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if *smoke {
		time.Sleep(500 * time.Millisecond)
		if err := runSmoke(*addr); err != nil {
			log.Printf("smoke FAILED: %v", err)
		} else {
			log.Printf("smoke OK")
		}
	} else {
		select {
		case <-ctx.Done():
		case err := <-errCh:
			log.Printf("server stopped: %v", err)
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = tcpSrv.Shutdown(shutdownCtx)
	_ = wt.Close()
	_ = udp.Close()
}

func serveWT(session *webtransport.Session, rate, size int) {
	defer session.CloseWithError(0, "done")
	ctx := session.Context()
	var seq uint32
	ticker := time.NewTicker(time.Second / time.Duration(max(rate, 1)))
	defer ticker.Stop()

	go func() {
		for {
			payload, err := session.ReceiveDatagram(ctx)
			if err != nil {
				return
			}
			if len(payload) >= 9 && payload[0] == opPing {
				clientMs := binary.LittleEndian.Uint64(payload[1:9])
				pong := make([]byte, 17)
				pong[0] = opPong
				binary.LittleEndian.PutUint64(pong[1:9], clientMs)
				binary.LittleEndian.PutUint64(pong[9:17], uint64(time.Now().UnixMilli()))
				_ = session.SendDatagram(pong)
			}
		}
	}()

	eventTicker := time.NewTicker(2 * time.Second)
	defer eventTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-eventTicker.C:
			go sendEvent(session, seq)
		case <-ticker.C:
			seq++
			_ = session.SendDatagram(snapshotPayload(seq, size))
		}
	}
}

func sendEvent(session *webtransport.Session, seq uint32) {
	ctx, cancel := context.WithTimeout(session.Context(), 2*time.Second)
	defer cancel()
	stream, err := session.OpenUniStreamSync(ctx)
	if err != nil {
		return
	}
	line, _ := json.Marshal(map[string]any{"type": "spike_event", "seq": seq, "at": time.Now().UnixMilli()})
	_, _ = stream.Write(append(line, '\n'))
	_ = stream.Close()
}

func snapshotPayload(seq uint32, size int) []byte {
	if size < 13 {
		size = 13
	}
	buf := make([]byte, size)
	buf[0] = opSnapshot
	binary.LittleEndian.PutUint32(buf[1:5], seq)
	binary.LittleEndian.PutUint64(buf[5:13], uint64(time.Now().UnixMilli()))
	return buf
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(*http.Request) bool { return true },
}

func wsHandler(rate, size int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		reqRate, reqSize := requestTuning(r, rate, size)
		var seq uint32
		var pongs atomic.Int64
		go func() {
			for {
				_, payload, err := conn.ReadMessage()
				if err != nil {
					return
				}
				if len(payload) >= 9 && payload[0] == opPing {
					pong := make([]byte, 17)
					pong[0] = opPong
					copy(pong[1:9], payload[1:9])
					binary.LittleEndian.PutUint64(pong[9:17], uint64(time.Now().UnixMilli()))
					if err := conn.WriteMessage(websocket.BinaryMessage, pong); err != nil {
						return
					}
					pongs.Add(1)
				}
			}
		}()

		ticker := time.NewTicker(time.Second / time.Duration(max(reqRate, 1)))
		defer ticker.Stop()
		eventTicker := time.NewTicker(2 * time.Second)
		defer eventTicker.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-eventTicker.C:
				line, _ := json.Marshal(map[string]any{"type": "spike_event", "seq": seq, "at": time.Now().UnixMilli()})
				if err := conn.WriteMessage(websocket.TextMessage, line); err != nil {
					return
				}
			case <-ticker.C:
				seq++
				if err := conn.WriteMessage(websocket.BinaryMessage, snapshotPayload(seq, reqSize)); err != nil {
					return
				}
			}
		}
	}
}

func runSmoke(addr string) error {
	host := addr
	if strings.HasPrefix(host, ":") {
		host = "localhost" + host
	}
	t := &webtransport.Transport{
		ApplicationProtocols: []string{"gu.spike.v1"},
		TLSClientConfig:      &tls.Config{InsecureSkipVerify: true},
		QUICConfig:           &quic.Config{EnableDatagrams: true, EnableStreamResetPartialDelivery: true},
	}
	defer t.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	resp, session, err := t.Dial(ctx, "https://"+host+"/wt?rate=20&size=220", nil)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	defer session.CloseWithError(0, "smoke done")
	for i := 0; i < 5; i++ {
		payload, err := session.ReceiveDatagram(ctx)
		if err != nil {
			return err
		}
		if len(payload) != 220 || payload[0] != opSnapshot {
			return fmt.Errorf("unexpected datagram: len=%d op=%d", len(payload), payload[0])
		}
	}
	ping := make([]byte, 9)
	ping[0] = opPing
	binary.LittleEndian.PutUint64(ping[1:], uint64(time.Now().UnixMilli()))
	if err := session.SendDatagram(ping); err != nil {
		return err
	}
	for {
		payload, err := session.ReceiveDatagram(ctx)
		if err != nil {
			return err
		}
		if payload[0] == opPong {
			return nil
		}
	}
}

func requestTuning(r *http.Request, defaultRate, defaultSize int) (int, int) {
	rate, size := defaultRate, defaultSize
	if value := r.URL.Query().Get("rate"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed >= 1 && parsed <= 120 {
			rate = parsed
		}
	}
	if value := r.URL.Query().Get("size"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed >= 64 && parsed <= 1400 {
			size = parsed
		}
	}
	return rate, size
}

func acmeTLS(domain, email, storage string, staging bool) (*tls.Config, *tls.Config, error) {
	token := os.Getenv("CF_DNS_API_TOKEN")
	if token == "" {
		return nil, nil, fmt.Errorf("CF_DNS_API_TOKEN is required for -acme-domain")
	}
	if domain == "" {
		return nil, nil, fmt.Errorf("-acme-domain is required")
	}
	var magic *certmagic.Config
	cache := certmagic.NewCache(certmagic.CacheOptions{
		GetConfigForCert: func(certmagic.Certificate) (*certmagic.Config, error) { return magic, nil },
	})
	magic = certmagic.New(cache, certmagic.Config{Storage: &certmagic.FileStorage{Path: storage}})
	ca := certmagic.LetsEncryptProductionCA
	if staging {
		ca = certmagic.LetsEncryptStagingCA
	}
	issuer := certmagic.NewACMEIssuer(magic, certmagic.ACMEIssuer{
		CA:                      ca,
		Email:                   email,
		Agreed:                  true,
		DisableHTTPChallenge:    true,
		DisableTLSALPNChallenge: true,
		DNS01Solver: &certmagic.DNS01Solver{
			DNSManager: certmagic.DNSManager{
				DNSProvider: &cloudflare.Provider{APIToken: token},
			},
		},
	})
	magic.Issuers = []certmagic.Issuer{issuer}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if err := magic.ManageSync(ctx, []string{domain}); err != nil {
		return nil, nil, err
	}
	tcp := magic.TLSConfig()
	tcp.NextProtos = []string{"h2", "http/1.1"}
	return tcp, http3.ConfigureTLSConfig(magic.TLSConfig()), nil
}

func selfSignedCert() tls.Certificate {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("generate key: %v", err)
	}
	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "wt-spike"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(7 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		log.Fatalf("create cert: %v", err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

func init() {
	// Keep flag errors readable without importing a second logging stack.
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "wtspike: WebTransport vs WebSocket latency probe (#310)\n")
		flag.PrintDefaults()
	}
}
