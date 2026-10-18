package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
)

const (
	upstreamQAWSAsset       = "123"
	upstreamQAWSMarket      = "qa-market"
	upstreamQAWSExtra       = "extra-subscription"
	upstreamQAWSEventID     = "immediate-retry-order"
	upstreamQAWSTimeout     = 2 * time.Second
	upstreamQAWSSingleEntry = 1
)

func TestUpstreamQAWSFailedConnectCleansReceiverAndAllowsRetry(t *testing.T) {
	for _, channel := range []Channel{ChannelMarket, ChannelUser} {
		t.Run(string(channel), func(t *testing.T) {
			var clientPointer atomic.Pointer[clientImpl]
			var allowConnection atomic.Bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				client := clientPointer.Load()
				client.subMu.Lock()
				registered := len(client.orderbookSubs) == upstreamQAWSSingleEntry && client.marketRefs[upstreamQAWSAsset] == upstreamQAWSSingleEntry
				if channel == ChannelUser {
					registered = len(client.orderSubs) == upstreamQAWSSingleEntry && client.userRefs[upstreamQAWSMarket] == upstreamQAWSSingleEntry
				}
				client.subMu.Unlock()
				if !registered {
					t.Error("receiver and references must exist before either successful or failed connection setup")
				}
				if !allowConnection.Load() {
					http.Error(w, "dial rejected", http.StatusForbidden)
					return
				}
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					t.Error(err)
					return
				}
				defer func() { _ = conn.Close() }()
				if err := conn.SetReadDeadline(time.Now().Add(upstreamQAWSTimeout)); err != nil {
					t.Error(err)
					return
				}
				var request SubscriptionRequest
				if err := conn.ReadJSON(&request); err != nil {
					t.Error(err)
					return
				}
				expectedType := ChannelMarket
				if channel == ChannelUser {
					expectedType = ChannelSubscribe
				}
				if request.Type != expectedType || request.Operation != OperationSubscribe {
					t.Errorf("retry did not send a new subscription: %+v", request)
				}
				var event map[string]string
				if channel == ChannelMarket {
					event = map[string]string{"event_type": "book", "asset_id": upstreamQAWSAsset, "market": upstreamQAWSMarket}
				} else {
					if request.Auth == nil {
						t.Error("user retry lost authentication")
					}
					event = map[string]string{"event_type": "order", "id": upstreamQAWSEventID, "market": upstreamQAWSMarket}
				}
				// Reply immediately after the wire request, never after a client-side ready signal.
				if err := conn.WriteJSON(event); err != nil {
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
			client := upstreamQANewWSClient(server.URL)
			clientPointer.Store(client)
			defer func() { _ = client.Close() }()
			ids := []string{upstreamQAWSAsset}
			if channel == ChannelUser {
				ids = []string{upstreamQAWSMarket}
			}
			if _, _, err := upstreamQASubscribeWS(t, client, channel, ids); err == nil {
				t.Fatal("failed handshake unexpectedly returned a stream")
			}
			client.subMu.Lock()
			entries := len(client.orderbookSubs) + len(client.orderSubs)
			refs := len(client.marketRefs) + len(client.userRefs)
			client.subMu.Unlock()
			if entries != 0 || refs != 0 {
				t.Fatalf("failed connect leaked receiver/ref: entries=%d refs=%d", entries, refs)
			}
			allowConnection.Store(true)
			closeStream, receive, err := upstreamQASubscribeWS(t, client, channel, ids)
			if err != nil {
				t.Fatal(err)
			}
			receive(t)
			if err := closeStream(); err != nil {
				t.Fatal(err)
			}
			if err := closeStream(); err != nil {
				t.Fatal(err)
			}
			client.subMu.Lock()
			entries = len(client.orderbookSubs) + len(client.orderSubs)
			refs = len(client.marketRefs) + len(client.userRefs)
			client.subMu.Unlock()
			if entries != 0 || refs != 0 {
				t.Fatalf("successful retry close leaked receiver/ref: entries=%d refs=%d", entries, refs)
			}
		})
	}
}

func TestUpstreamQAWSFailedWritePreservesOverlappingSibling(t *testing.T) {
	for _, channel := range []Channel{ChannelMarket, ChannelUser} {
		t.Run(string(channel), func(t *testing.T) {
			server := mockWSServer(t, func(conn *websocket.Conn) { _, _, _ = conn.ReadMessage() })
			defer server.Close()
			client := upstreamQANewWSClient(server.URL)
			defer func() { _ = client.Close() }()
			conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
			if response != nil {
				_ = response.Body.Close()
			}
			if err != nil {
				t.Fatal(err)
			}
			// Retain a closed socket without starting a read loop. ensureConn succeeds,
			// then the subscription write deterministically fails on that exact socket.
			if err := conn.Close(); err != nil {
				t.Fatal(err)
			}
			var siblingClosed func() bool
			var siblingID string
			ids := []string{upstreamQAWSAsset, upstreamQAWSExtra}
			if channel == ChannelMarket {
				client.setMarketConn(conn)
				entry := newSubscriptionEntry[OrderbookEvent](client, channel, Orderbook, ids[:upstreamQAWSSingleEntry], nil)
				client.orderbookSubs[entry.id] = entry
				siblingID = entry.id
				client.addMarketRefs(ids[:upstreamQAWSSingleEntry], false)
				siblingClosed = func() bool { entry.mu.RLock(); defer entry.mu.RUnlock(); return entry.closed }
			} else {
				client.setUserConn(conn)
				ids = []string{upstreamQAWSMarket, upstreamQAWSExtra}
				entry := newSubscriptionEntry[OrderEvent](client, channel, UserOrders, nil, ids[:upstreamQAWSSingleEntry])
				client.orderSubs[entry.id] = entry
				siblingID = entry.id
				client.addUserRefs(ids[:upstreamQAWSSingleEntry], client.resolveAuth(nil))
				siblingClosed = func() bool { entry.mu.RLock(); defer entry.mu.RUnlock(); return entry.closed }
			}
			if _, _, err := upstreamQASubscribeWS(t, client, channel, ids); err == nil {
				t.Fatal("closed-socket subscription did not fail")
			}
			client.subMu.Lock()
			entries := len(client.orderbookSubs) + len(client.orderSubs)
			siblingPresent := client.orderbookSubs[siblingID] != nil
			refCount, extraRefCount := client.marketRefs[ids[0]], client.marketRefs[upstreamQAWSExtra]
			if channel == ChannelUser {
				siblingPresent = client.orderSubs[siblingID] != nil
				refCount, extraRefCount = client.userRefs[ids[0]], client.userRefs[upstreamQAWSExtra]
			}
			client.subMu.Unlock()
			if entries != upstreamQAWSSingleEntry || refCount != upstreamQAWSSingleEntry || extraRefCount != 0 || !siblingPresent || siblingClosed() {
				t.Fatalf("failed setup damaged sibling or leaked own state: entries=%d shared=%d extra=%d siblingPresent=%t siblingClosed=%t", entries, refCount, extraRefCount, siblingPresent, siblingClosed())
			}
		})
	}
}

func upstreamQANewWSClient(url string) *clientImpl {
	cfg := DefaultClientConfig()
	cfg.Reconnect = false
	cfg.DisablePing = true
	return newClientImpl("ws"+strings.TrimPrefix(url, "http"), nil, &auth.APIKey{Key: "qa-key", Secret: "qa-secret", Passphrase: "qa-pass"}, cfg)
}

func upstreamQASubscribeWS(t *testing.T, client *clientImpl, channel Channel, ids []string) (func() error, func(*testing.T), error) {
	t.Helper()
	if channel == ChannelMarket {
		stream, err := client.SubscribeOrderbookStream(t.Context(), ids)
		if err != nil {
			if stream != nil {
				t.Fatal("failed market subscription returned a non-nil stream")
			}
			return nil, nil, err
		}
		return stream.Close, func(t *testing.T) {
			t.Helper()
			select {
			case event, ok := <-stream.C:
				if !ok || event.AssetID != upstreamQAWSAsset {
					t.Fatalf("retry lost or corrupted immediate book: %+v open=%t", event, ok)
				}
			case <-time.After(upstreamQAWSTimeout):
				t.Fatal("retry did not receive immediate market event")
			}
		}, nil
	}
	stream, err := client.SubscribeUserOrdersStream(t.Context(), ids)
	if err != nil {
		if stream != nil {
			t.Fatal("failed user subscription returned a non-nil stream")
		}
		return nil, nil, err
	}
	return stream.Close, func(t *testing.T) {
		t.Helper()
		select {
		case event, ok := <-stream.C:
			if !ok || event.ID != upstreamQAWSEventID || event.Market != upstreamQAWSMarket {
				t.Fatalf("retry lost or corrupted immediate user event: %+v open=%t", event, ok)
			}
		case <-time.After(upstreamQAWSTimeout):
			t.Fatal("retry did not receive immediate user event")
		}
	}, nil
}
