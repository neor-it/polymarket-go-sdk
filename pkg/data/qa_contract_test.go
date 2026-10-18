package data_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/neor-it/polymarket-go-sdk/pkg/data"
	"github.com/neor-it/polymarket-go-sdk/pkg/gamma"
	"github.com/neor-it/polymarket-go-sdk/pkg/transport"
	"github.com/shopspring/decimal"
)

const (
	qaUser              = "0x0000000000000000000000000000000000000001"
	qaLimit             = 2
	qaCancellationDelay = 30 * time.Millisecond
	qaHugeToken         = "115792089237316195423570985008687907853269984665640564039457584007913129639935"
)

func TestQAV2TradeBoundaryPreservesSelectorsCursorAndDuplicateFills(t *testing.T) {
	native := "0x" + strings.Repeat("ab", 31)
	condition, err := gamma.ConditionHash(gamma.ProtocolV2, native)
	if err != nil {
		t.Fatal(err)
	}
	const cursor = "signed+/cursor=with&selectors"
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		q := r.URL.Query()
		if r.URL.Path != "/v2/trades" || q.Get("condition") != native+"00" || q.Get("cursor") != cursor || q.Get("user") != common.HexToAddress(qaUser).Hex() || q.Get("taker_only") != "false" || q.Get("side") != "BUY" || q.Has("offset") || q.Has("market") {
			t.Errorf("unexpected request: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		// Two identical fills can legitimately occur within one transaction.
		row := map[string]any{"token_id": qaHugeToken, "size": "0.000001", "price": "0.1234567890123456789", "proxy_wallet": qaUser, "transaction_hash": "same", "side": "BUY"}
		if err := json.NewEncoder(w).Encode(map[string]any{"data": []any{row, row}, "pagination": map[string]any{"limit": qaLimit, "offset": 0, "has_more": false, "next_cursor": nil}}); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	client := data.NewClient(transport.NewClient(server.Client(), server.URL))
	limit, takerOnly, side, user := qaLimit, false, data.SideBuy, common.HexToAddress(qaUser)
	response, err := client.Trades(context.Background(), &data.TradesRequest{User: &user, Filter: data.MarketFilterByMarkets([]common.Hash{condition}), Cursor: qaPointer(cursor), Limit: &limit, TakerOnly: &takerOnly, Side: &side})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Data) != qaLimit || calls.Load() != 1 {
		t.Fatalf("fills truncated or deduplicated: %+v", response)
	}
	if response.Data[0].TokenID != qaHugeToken || !response.Data[0].Price.Equal(decimal.RequireFromString("0.1234567890123456789")) {
		t.Fatal("asset or decimal precision lost")
	}
}

func TestQAV2RetryAfterCancellationDoesNotRetryAndPreservesContext(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "3600")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	client := data.NewClient(transport.NewClient(server.Client(), server.URL))
	ctx, cancel := context.WithTimeout(context.Background(), qaCancellationDelay)
	defer cancel()
	_, err := client.Trades(ctx, &data.TradesRequest{})
	if !errors.Is(err, context.DeadlineExceeded) || calls.Load() != 1 {
		t.Fatalf("context or Retry-After lost: %v calls=%d", err, calls.Load())
	}
}

func TestQAV2MalformedPayloadAndClientErrorsNeverBecomeEmptySuccess(t *testing.T) {
	for _, test := range []struct {
		name, body string
		status     int
	}{
		{"malformed", "{bad", http.StatusOK},
		{"legacy-array", "[]", http.StatusOK},
		{"bad-decimal", `{"data":[{"price":"not-a-number"}]}`, http.StatusOK},
		{"client-rejection", `{"error":"invalid cursor"}`, http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.WriteHeader(test.status)
				if _, err := w.Write([]byte(test.body)); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			client := data.NewClient(transport.NewClient(server.Client(), server.URL))
			_, err := client.Trades(context.Background(), &data.TradesRequest{})
			if err == nil || calls.Load() != 1 {
				t.Fatalf("failure silently converted/retried: %v calls=%d", err, calls.Load())
			}
		})
	}
}

func TestQAV2ConditionPaddingAndExplicitMarketVersion(t *testing.T) {
	native := "0x" + strings.Repeat("ab", 31)
	for _, test := range []struct {
		version gamma.ProtocolVersion
		id      string
		valid   bool
	}{
		{gamma.ProtocolV2, native, true}, {gamma.ProtocolV2, native + "00", true}, {gamma.ProtocolV2, native + "01", false},
		{gamma.ProtocolV1, native, false}, {gamma.ProtocolV1, native + "01", true}, {"", native + "00", false},
	} {
		_, err := gamma.ConditionHash(test.version, test.id)
		if (err == nil) != test.valid {
			t.Fatalf("condition validation %s %s: %v", test.version, test.id, err)
		}
	}
	market := gamma.Market{Version: gamma.ProtocolV2, ClobTokenIds: `["11","12"]`, PositionIDs: []string{qaHugeToken, "22"}, Outcomes: `["YES","NO"]`}
	tokens, err := market.TradingTokens()
	if err != nil || tokens[0].TokenID != qaHugeToken {
		t.Fatalf("wrong ledger %+v %v", tokens, err)
	}
	market.Version = ""
	if _, err := market.TradingTokens(); err == nil {
		t.Fatal("missing version inferred from IDs")
	}
}

func qaPointer[T any](value T) *T { return &value }

func TestQAActivityUnavailablePayoutCannotBecomeConfirmedZero(t *testing.T) {
	for _, test := range []struct {
		name, body string
		valid      bool
	}{
		{"missing", `{"data":[{"price":0,"size":0}]}`, false},
		{"null", `{"data":[{"price":0,"size":0,"usdc_size":null}]}`, false},
		{"zero", `{"data":[{"price":0,"size":0,"usdc_size":0}],"pagination":{"limit":2,"offset":0,"has_more":false,"next_cursor":null}}`, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if _, err := w.Write([]byte(test.body)); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			client := data.NewClient(transport.NewClient(server.Client(), server.URL))
			response, err := client.Activity(context.Background(), &data.ActivityRequest{})
			if (err == nil) != test.valid {
				t.Fatalf("availability silently changed: %+v %v", response, err)
			}
			if test.valid && (len(response.Data) != 1 || !response.Data[0].USDCSize.IsZero()) {
				t.Fatalf("explicit losing zero rejected: %+v", response)
			}
		})
	}
}

func TestQAV2UnavailableEnvelopeCannotBecomeEmptyHistory(t *testing.T) {
	for _, payload := range []string{
		`{}`, `null`, `{"pagination":{"limit":2,"offset":0,"has_more":false}}`,
		`{"data":null,"pagination":{"limit":2,"offset":0,"has_more":false}}`,
		`{"data":[]}`, `{"data":[],"pagination":{"limit":2,"offset":0,"has_more":true,"next_cursor":null}}`,
	} {
		t.Run(payload, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if _, err := w.Write([]byte(payload)); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			client := data.NewClient(transport.NewClient(server.Client(), server.URL))
			if _, err := client.Trades(context.Background(), &data.TradesRequest{}); err == nil {
				t.Fatal("unavailable page became empty success")
			}
		})
	}
}
