package ws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func mockWSServer(t *testing.T, handler func(*websocket.Conn)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade failed: %v", err)
			return
		}
		defer func() { _ = conn.Close() }()
		handler(conn)
	}))
}

func TestClientConnection(t *testing.T) {
	// Simple test: connect and receive one message
	s := mockWSServer(t, func(c *websocket.Conn) {
		// The peer must receive the actual asset subscription before emitting an event.
		var subscription SubscriptionRequest
		if err := c.ReadJSON(&subscription); err != nil {
			t.Errorf("read subscription: %v", err)
			return
		}
		if subscription.Type != ChannelMarket || subscription.Operation != OperationSubscribe || len(subscription.AssetIDs) != 1 || subscription.AssetIDs[0] != "123" {
			t.Errorf("unexpected subscription: %+v", subscription)
			return
		}

		// Send a dummy event with proper price_changes structure
		err := c.WriteJSON(map[string]interface{}{
			"event_type": "price",
			"market":     "m1",
			"price_changes": []map[string]string{
				{"asset_id": "123", "price": "0.5"},
			},
		})
		if err != nil {
			return
		}
		// Keep the socket open until client cleanup, without timing assumptions.
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	})
	defer s.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(s.URL, "http")

	client, err := NewClient(wsURL, nil, nil)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	// NewClient establishes the connection before returning.
	if client.ConnectionState(ChannelMarket) != ConnectionConnected {
		t.Errorf("expected connected, got %v", client.ConnectionState(ChannelMarket))
	}

	sub, err := client.SubscribePrices(context.Background(), []string{"123"})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	select {
	case event := <-sub:
		if event.AssetID != "123" {
			t.Errorf("expected asset 123, got %s", event.AssetID)
		}
	case <-time.After(1 * time.Second):
		t.Error("timeout waiting for event")
	}
}

func TestClientReadTimeout(t *testing.T) {
	connections := make(chan struct{}, 10)

	s := mockWSServer(t, func(c *websocket.Conn) {
		connections <- struct{}{}
		// Hang: don't send anything. Client should timeout after 100ms.
		select {}
	})
	defer s.Close()

	wsURL := "ws" + strings.TrimPrefix(s.URL, "http")

	// Set reconnect max to allow multiple reconnects
	// We can't easily set reconnect options via NewClient without env vars,
	// but defaults are usually fine (reconnect=true).

	client, err := NewClient(wsURL, nil, nil)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	// Set a short read timeout for testing
	if impl, ok := client.(*clientImpl); ok {
		impl.setReadTimeout(100 * time.Millisecond)
	}

	// 1. First connection
	select {
	case <-connections:
		// OK
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for first connection")
	}

	// 2. Client should timeout (100ms) + reconnect delay (default is 2s, which is too long for this test)
	// We need to override reconnect delay?
	// The clientImpl reads CLOB_WS_RECONNECT_DELAY_MS from env.
	// But it reads it in NewClient. We can't set it easily now.
	// However, we can verify that the connection drops.

	time.Sleep(200 * time.Millisecond) // Wait for timeout

	// The client should have closed the connection by now.
	// We check if it reconnects.
	// Since default reconnect delay is 2s, we might need to wait > 2s.
	// That's acceptable for a test.

	select {
	case <-connections:
		// Reconnected!
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for reconnection")
	}
}

func TestClientStreamRegisteredBeforeConnection(t *testing.T) {
	const assetID = "123"
	const marketID = "m1"
	const eventID = "immediate-order"
	const eventTimeout = time.Second
	for _, channel := range []Channel{ChannelMarket, ChannelUser} {
		t.Run(string(channel), func(t *testing.T) {
			var client *clientImpl
			ready := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				<-ready
				client.subMu.Lock()
				registered := len(client.priceSubs) == 1 && client.marketRefs[assetID] == 1
				if channel == ChannelUser {
					registered = len(client.orderSubs) == 1 && client.userRefs[marketID] == 1
				}
				client.subMu.Unlock()
				if !registered {
					t.Error("stream and reference must be registered before connection can deliver its first event")
				}
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					t.Error(err)
					return
				}
				defer func() { _ = conn.Close() }()
				if err := conn.SetReadDeadline(time.Now().Add(eventTimeout)); err != nil {
					t.Error(err)
					return
				}
				var request SubscriptionRequest
				if err := conn.ReadJSON(&request); err != nil {
					t.Error(err)
					return
				}
				if channel == ChannelMarket {
					if request.Type != ChannelMarket || len(request.AssetIDs) != 1 || request.AssetIDs[0] != assetID {
						t.Errorf("wrong market subscription: %+v", request)
					}
					err = conn.WriteJSON(map[string]interface{}{"event_type": "price", "price_changes": []map[string]string{{"asset_id": assetID, "price": "0.5"}}})
				} else {
					if len(request.Markets) != 1 || request.Markets[0] != marketID || request.Auth == nil {
						t.Errorf("wrong authenticated user subscription: %+v", request)
					}
					err = conn.WriteJSON(map[string]string{"event_type": "order", "id": eventID, "market": marketID})
				}
				if err != nil {
					t.Error(err)
					return
				}
				for {
					if _, _, err := conn.ReadMessage(); err != nil {
						return
					}
				}
			}))
			defer server.Close()
			cfg := DefaultClientConfig()
			cfg.DisablePing = true
			cfg.Reconnect = false
			client = newClientImpl("ws"+strings.TrimPrefix(server.URL, "http"), nil, &auth.APIKey{Key: "test-key", Secret: "test-secret", Passphrase: "test-pass"}, cfg)
			close(ready)
			defer func() { _ = client.Close() }()
			if channel == ChannelMarket {
				stream, err := client.SubscribePricesStream(context.Background(), []string{assetID})
				if err != nil {
					t.Fatal(err)
				}
				select {
				case event := <-stream.C:
					if event.AssetID != assetID {
						t.Errorf("wrong immediate price: %+v", event)
					}
				case <-time.After(eventTimeout):
					t.Error("timeout waiting for immediate market event")
				}
			} else {
				stream, err := client.SubscribeUserOrdersStream(context.Background(), []string{marketID})
				if err != nil {
					t.Fatal(err)
				}
				select {
				case event := <-stream.C:
					if event.ID != eventID {
						t.Errorf("wrong immediate order: %+v", event)
					}
				case <-time.After(eventTimeout):
					t.Error("timeout waiting for immediate user event")
				}
			}
		})
	}
}
