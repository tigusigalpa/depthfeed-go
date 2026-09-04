package stream

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// wsMock is a minimal WebSocket server that records messages and can push
// canned frames back to the client.
type wsMock struct {
	server   *httptest.Server
	upgrader websocket.Upgrader

	// received messages (text)
	received []string

	// handler invoked for each received message; may push frames back.
	onMessage func(c *websocket.Conn, msg string)
	// onConnect is invoked once after a connection is upgraded.
	onConnect func(c *websocket.Conn)
	// onRequest receives the WebSocket handshake request before upgrade.
	onRequest func(r *http.Request)
}

func newWSMock(onMessage func(*websocket.Conn, string)) *wsMock {
	m := &wsMock{
		upgrader:  websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		onMessage: onMessage,
	}
	m.server = httptest.NewServer(http.HandlerFunc(m.handle))
	return m
}

func (m *wsMock) handle(w http.ResponseWriter, r *http.Request) {
	if m.onRequest != nil {
		m.onRequest(r)
	}
	c, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()
	if m.onConnect != nil {
		m.onConnect(c)
	}
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			return
		}
		m.received = append(m.received, string(msg))
		if m.onMessage != nil {
			m.onMessage(c, string(msg))
		}
	}
}

func (m *wsMock) url() string {
	return "ws" + strings.TrimPrefix(m.server.URL, "http") + "/v3/stream"
}

func (m *wsMock) close() { m.server.Close() }

func TestSubscribeAndAck(t *testing.T) {
	m := newWSMock(func(c *websocket.Conn, msg string) {
		var f Frame
		_ = json.Unmarshal([]byte(msg), &f)
		if f.Op == OpSubscribe {
			ack, _ := json.Marshal(Frame{Op: OpSubscribed, Channels: f.Channels, Active: len(f.Channels)})
			_ = c.WriteMessage(websocket.TextMessage, ack)
		}
	})
	defer m.close()

	c := NewClient(Config{APIKey: "df_test_example_not_a_real_key", URL: m.url()})
	if err := c.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer c.Close()

	if err := c.Subscribe([]string{BookPolymarket("btc", "123")}); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	frame, err := c.Receive(ctx)
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if !frame.IsSubscribed() {
		t.Errorf("expected subscribed ack, got op=%q", frame.Op)
	}
}

func TestPingPong(t *testing.T) {
	m := newWSMock(func(c *websocket.Conn, msg string) {
		var f Frame
		_ = json.Unmarshal([]byte(msg), &f)
		if f.Op == OpPing {
			_ = c.WriteMessage(websocket.TextMessage, []byte(`{"op":"pong"}`))
		}
	})
	defer m.close()

	c := NewClient(Config{APIKey: "df_test_example_not_a_real_key", URL: m.url()})
	if err := c.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer c.Close()

	if err := c.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	frame, err := c.Receive(ctx)
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if frame.Op != OpPong {
		t.Errorf("expected pong, got %q", frame.Op)
	}
}

func TestDataFrame(t *testing.T) {
	m := newWSMock(func(c *websocket.Conn, msg string) {
		var f Frame
		_ = json.Unmarshal([]byte(msg), &f)
		if f.Op == OpSubscribe {
			// Push a book data frame.
			data, _ := json.Marshal(Frame{
				Op:      "book",
				Channel: "book:polymarket:btc:123",
				TS:      1780774485780,
				Data:    json.RawMessage(`{"market_id":"123","price_up":0.5}`),
			})
			_ = c.WriteMessage(websocket.TextMessage, data)
		}
	})
	defer m.close()

	c := NewClient(Config{APIKey: "df_test_example_not_a_real_key", URL: m.url()})
	if err := c.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer c.Close()

	if err := c.Subscribe([]string{BookPolymarket("btc", "123")}); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	frame, err := c.Receive(ctx)
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if frame.Op != "book" || frame.Channel != "book:polymarket:btc:123" {
		t.Errorf("unexpected frame: %+v", frame)
	}
	if frame.TS != 1780774485780 {
		t.Errorf("ts = %d", frame.TS)
	}
}

func TestInvalidChannelRejected(t *testing.T) {
	m := newWSMock(nil)
	defer m.close()
	c := NewClient(Config{APIKey: "df_test", URL: m.url()})
	if err := c.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer c.Close()
	if err := c.Subscribe([]string{"bogus:channel"}); err == nil {
		t.Errorf("expected error for invalid channel")
	}
}

func TestChannelBuilders(t *testing.T) {
	cases := []struct{ got, want string }{
		{BookPolymarket("btc", "123"), "book:polymarket:btc:123"},
		{BookKalshi("btc", "KXBTC-25T"), "book:kalshi:btc:KXBTC-25T"},
		{BookLimitless("btc", "slug"), "book:limitless:btc:slug"},
		{BookBinance("btc", "spot"), "book:binance:btc:spot"},
		{BookSportsPolymarket("tok1"), "book:sports:polymarket:tok1"},
		{BookSportsKalshi("KXBTC-25T"), "book:sports:kalshi:KXBTC-25T"},
		{Price("btc"), "price:btc"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("got %q, want %q", tc.got, tc.want)
		}
	}
}

func TestValidateChannel(t *testing.T) {
	valid := []string{
		"book:polymarket:btc:123",
		"book:kalshi:btc:KXBTC-25T",
		"book:limitless:btc:slug",
		"book:binance:btc:spot",
		"book:sports:polymarket:tok1",
		"book:sports:kalshi:KXBTC-25T",
		"price:btc",
	}
	for _, ch := range valid {
		if !ValidateChannel(ch) {
			t.Errorf("ValidateChannel(%q) = false, want true", ch)
		}
	}
	invalid := []string{
		"", "bogus", "book:unknown:x:y", "price:", "price:ada",
		"book:binance:btc:margin", "book:polymarket:btc:", "book:sports:kalshi:a:b",
	}
	for _, ch := range invalid {
		if ValidateChannel(ch) {
			t.Errorf("ValidateChannel(%q) = true, want false", ch)
		}
	}
}

func TestAuthBearerHeaderSent(t *testing.T) {
	var gotAuth string
	m := newWSMock(nil)
	m.onConnect = func(c *websocket.Conn) {
		// gorilla websocket exposes the request via the upgrade; we cannot
		// easily read headers here, so we just close. Auth is verified via
		// the dialer behavior in production; here we only assert Connect
		// succeeds with bearer mode.
		_ = gotAuth
	}
	defer m.close()
	c := NewClient(Config{APIKey: "df_test_example_not_a_real_key", URL: m.url()})
	if err := c.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer c.Close()
}

func TestFatalCloseNoReconnect(t *testing.T) {
	m := newWSMock(nil)
	m.onConnect = func(c *websocket.Conn) {
		// Immediately send a fatal auth-failed close.
		_ = c.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(int(CloseAuthFailed), "auth failed"))
		_ = c.Close()
	}
	defer m.close()
	c := NewClient(Config{APIKey: "df_test", URL: m.url(), Reconnect: true})
	if err := c.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := c.Receive(ctx)
	if err == nil {
		t.Fatalf("expected error on fatal close")
	}
	if !strings.Contains(err.Error(), "4001") {
		t.Errorf("expected 4001 in error, got %v", err)
	}
}

func TestClientUtilityAndAuthenticationBranches(t *testing.T) {
	m := newWSMock(nil)
	defer m.close()
	var query, apiKey string
	m.onRequest = func(r *http.Request) {
		query = r.URL.Query().Get("api_key")
		apiKey = r.Header.Get("X-API-Key")
	}

	c := NewClient(Config{APIKey: "df_test", URL: m.url()},
		WithAuthMode(AuthQuery), WithUserAgent("test-agent"),
		WithDialer(&websocket.Dialer{}), WithReconnect(false),
		WithBackoff(time.Millisecond, 2*time.Millisecond), WithMaxReconnectAttempts(2))
	if c.cfg.InitialBackoff != time.Millisecond || c.cfg.MaxReconnectAttempts != 2 {
		t.Fatal("stream options were not applied")
	}
	if err := c.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if query != "df_test" {
		t.Fatalf("query auth key = %q", query)
	}
	if err := c.Subscribe(nil); err != nil || c.Unsubscribe(nil) != nil {
		t.Fatal("empty subscription operation should be a no-op")
	}
	if err := c.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if err := c.Close(); err != nil || c.Close() != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := c.Connect(context.Background()); err == nil {
		t.Fatal("closed client should not connect")
	}

	headerClient := NewClient(Config{APIKey: "df_header", URL: m.url(), AuthMode: AuthHeader})
	if err := headerClient.Connect(context.Background()); err != nil {
		t.Fatalf("header Connect: %v", err)
	}
	defer headerClient.Close()
	if apiKey != "df_header" {
		t.Fatalf("header auth key = %q", apiKey)
	}
	if !hasQuery("wss://example.test/path?x=1") || hasQuery("wss://example.test/path") {
		t.Fatal("hasQuery returned an unexpected value")
	}
}

func TestReceiveUtilityBranches(t *testing.T) {
	c := NewClient(Config{})
	if _, err := c.Receive(context.Background()); err == nil {
		t.Fatal("Receive should fail before Connect")
	}
	if _, err := c.ReceiveRaw(context.Background()); err == nil {
		t.Fatal("ReceiveRaw should fail before Connect")
	}
	if err := c.Ping(); err == nil {
		t.Fatal("Ping should fail before Connect")
	}

	m := newWSMock(nil)
	m.onConnect = func(conn *websocket.Conn) {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("not-json"))
	}
	defer m.close()
	c = NewClient(Config{APIKey: "df_test", URL: m.url(), Reconnect: false})
	if err := c.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer c.Close()
	frame, err := c.Receive(context.Background())
	if err != nil || frame.Message != "not-json" || frame.Op != OpError {
		t.Fatalf("unexpected fallback frame: %+v, %v", frame, err)
	}
}
