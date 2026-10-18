package data

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/neor-it/polymarket-go-sdk/pkg/types"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/neor-it/polymarket-go-sdk/pkg/transport"
	"github.com/shopspring/decimal"
)

func ptr[T any](value T) *T { return &value }

func TestPositionsCursorLifecycleAndExactEconomics(t *testing.T) {
	const tokenID = "115792089237316195423570985008687907853269984665640564039457584007913129639935"
	condition := common.HexToHash("0x1234")
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/v2/positions" {
			t.Errorf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("condition") != condition.Hex() || r.URL.Query().Get("status") != "CLOSED" {
			t.Errorf("query %s", r.URL.RawQuery)
		}
		if r.URL.Query().Has("offset") || r.URL.Query().Has("market") {
			t.Errorf("retired query %s", r.URL.RawQuery)
		}
		switch r.URL.Query().Get("cursor") {
		case "":
			w.Write([]byte(`{"data":[{"token_id":"` + tokenID + `","avg_price":0.5,"current_price":0,"current_value":0,"percent_pnl":0,"percent_realized_pnl":0,"realized_pnl":0,"total_pnl":0,"unrealized_pnl":0,"current_size":0.000001,"entry_cost_usdc":9007199254.740993,"entry_fees_usdc":0.000001,"total_cost_usdc":9007199254.740994,"total_size":3,"status":"CLOSED"}],"pagination":{"has_more":true,"limit":1,"offset":0,"next_cursor":"opaque+/="}}`))
		case "opaque+/=":
			w.Write([]byte(`{"data":[],"pagination":{"has_more":false,"limit":1,"offset":1,"next_cursor":null}}`))
		default:
			t.Errorf("unexpected cursor")
		}
	}))
	defer server.Close()
	client := NewClient(transport.NewClient(server.Client(), server.URL))
	request := &PositionsRequest{Filter: MarketFilterByMarkets([]common.Hash{condition}), Status: ptr(PositionClosed), Limit: ptr(1)}
	first, err := client.Positions(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Data) != 1 || first.Data[0].TokenID != tokenID || first.Data[0].CurrentSize.String() != "0.000001" || first.Data[0].EntryCostUSDC.String() != "9007199254.740993" {
		t.Fatalf("imprecise position %+v", first)
	}
	if !first.Data[0].EntryCostUSDC.Add(first.Data[0].EntryFeesUSDC).Equal(first.Data[0].TotalCostUSDC) {
		t.Fatal("cost basis identity changed")
	}
	request.Cursor = first.Pagination.NextCursor
	last, err := client.Positions(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(last.Data) != 0 || last.Pagination.NextCursor != nil || last.Pagination.HasMore || calls != 2 {
		t.Fatalf("pagination %+v calls=%d", last, calls)
	}
}

func TestAllV2RoutesAndNullUserStats(t *testing.T) {
	cases := []struct {
		path string
		body string
		call func(Client) error
	}{
		{"/v2/activity", `{"data":[],"pagination":{"has_more":false,"offset":0,"limit":2,"next_cursor":null}}`, func(c Client) error { _, e := c.Activity(context.Background(), &ActivityRequest{}); return e }},
		{"/v2/approvals", `{"data":{"contracts":[]}}`, func(c Client) error { _, e := c.Approvals(context.Background(), &ApprovalsRequest{}); return e }},
		{"/v2/biggest-winners", `{"data":[],"pagination":{"has_more":false,"offset":0,"limit":2,"next_cursor":null}}`, func(c Client) error {
			_, e := c.BiggestWinners(context.Background(), &BiggestWinnersRequest{})
			return e
		}},
		{"/v2/builders/leaderboard", `{"data":[],"pagination":{"has_more":false,"offset":0,"limit":2,"next_cursor":null}}`, func(c Client) error {
			_, e := c.BuildersLeaderboard(context.Background(), &BuildersLeaderboardRequest{})
			return e
		}},
		{"/v2/builders/volume", `{"data":[]}`, func(c Client) error {
			_, e := c.BuildersVolume(context.Background(), &BuildersVolumeRequest{})
			return e
		}},
		{"/v2/holders", `{"data":[],"pagination":{"has_more":false,"offset":0,"limit":2,"next_cursor":null}}`, func(c Client) error { _, e := c.Holders(context.Background(), &HoldersRequest{}); return e }},
		{"/v2/leaderboard", `{"data":[],"pagination":{"has_more":false,"offset":0,"limit":2,"next_cursor":null}}`, func(c Client) error { _, e := c.Leaderboard(context.Background(), &LeaderboardRequest{}); return e }},
		{"/v2/live-volume", `{"data":{"conditions":[],"taker_volume_total":0}}`, func(c Client) error { _, e := c.LiveVolume(context.Background(), &LiveVolumeRequest{}); return e }},
		{"/v2/oi", `{"data":[]}`, func(c Client) error { _, e := c.OpenInterest(context.Background(), &OpenInterestRequest{}); return e }},
		{"/v2/prices-history", `{"data":[],"pagination":{"has_more":false,"offset":0,"limit":2,"next_cursor":null}}`, func(c Client) error { _, e := c.PricesHistory(context.Background(), &PricesHistoryRequest{}); return e }},
		{"/v2/resolutions", `{"data":[]}`, func(c Client) error { _, e := c.Resolutions(context.Background(), &ResolutionsRequest{}); return e }},
		{"/v2/status", `{"data":{}}`, func(c Client) error { _, e := c.Status(context.Background()); return e }},
		{"/v2/tokens", `{"data":[]}`, func(c Client) error { _, e := c.Tokens(context.Background(), &TokensRequest{}); return e }},
		{"/v2/trades", `{"data":[],"pagination":{"has_more":false,"offset":0,"limit":2,"next_cursor":null}}`, func(c Client) error { _, e := c.Trades(context.Background(), &TradesRequest{}); return e }},
		{"/v2/user-pnl", `{"data":{"points":[]}}`, func(c Client) error { _, e := c.UserPnl(context.Background(), &UserPnlRequest{}); return e }},
		{"/v2/user-stats", `{"data":null}`, func(c Client) error {
			v, e := c.UserStats(context.Background(), &UserStatsRequest{})
			if v.Data != nil {
				t.Errorf("null user stats became a user")
			}
			return e
		}},
		{"/v2/user-volume", `{"data":{"volume":0,"volume_usdc":0}}`, func(c Client) error { _, e := c.UserVolume(context.Background(), &UserVolumeRequest{}); return e }},
		{"/v2/value", `{"data":{"proxy_wallet":"0x123","value":0.000001}}`, func(c Client) error {
			v, e := c.Value(context.Background(), &ValueRequest{})
			if v.Data.Value.String() != "0.000001" {
				t.Errorf("value precision")
			}
			return e
		}},
	}
	for _, test := range cases {
		t.Run(test.path, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != test.path {
					t.Errorf("path %s", r.URL.Path)
				}
				w.Write([]byte(test.body))
			}))
			defer server.Close()
			if err := test.call(NewClient(transport.NewClient(server.Client(), server.URL))); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestTradesActivityWireShape(t *testing.T) {
	var trade TradesResponse
	if err := json.Unmarshal([]byte(`{"data":[{"proxy_wallet":"0x123","token_id":"123456789012345678901234567890","condition_id":"0x1","size":0.000001,"price":0.123456,"transaction_hash":"0x2"}],"pagination":{"has_more":true,"offset":0,"limit":2,"next_cursor":"next"}}`), &trade); err != nil {
		t.Fatal(err)
	}
	if trade.Data[0].Size.String() != "0.000001" || trade.Data[0].TokenID != "123456789012345678901234567890" {
		t.Fatalf("trade %+v", trade)
	}
	var activity ActivityResponse
	if err := json.Unmarshal([]byte(`{"data":[{"type":"REDEEM","token_id":"4","condition_id":"0x1","outcome_index":1,"size":2.000001,"price":0,"usdc_size":1.000001,"transaction_hash":"0x2"}],"pagination":{"has_more":false,"offset":0,"limit":2,"next_cursor":null}}`), &activity); err != nil {
		t.Fatal(err)
	}
	if activity.Data[0].USDCSize.String() != "1.000001" || activity.Data[0].OutcomeIndex != 1 {
		t.Fatalf("activity %+v", activity)
	}
}

func TestV2ValidationAndCanonicalQueries(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		q := r.URL.Query()
		if q.Get("taker_only") != "false" || q.Get("filter_type") != "CASH" || q.Get("filter_amount") != "0.000001" || q.Get("start") != "1" || q.Get("end") != "2" || q.Get("side") != "BUY" {
			t.Errorf("query %s", r.URL.RawQuery)
		}
		w.Write([]byte(`{"data":[],"pagination":{"has_more":false,"offset":0,"limit":2,"next_cursor":null}}`))
	}))
	defer server.Close()
	client := NewClient(transport.NewClient(server.Client(), server.URL))
	if _, err := client.Trades(context.Background(), nil); err == nil {
		t.Fatal("nil request accepted")
	}
	if _, err := client.Trades(context.Background(), &TradesRequest{Limit: ptr(1001)}); err == nil {
		t.Fatal("oversized limit accepted")
	}
	if _, err := client.Trades(context.Background(), &TradesRequest{TradeFilter: TradeFilterCash(decimal.RequireFromString("-1"))}); err == nil {
		t.Fatal("negative cash accepted")
	}
	if _, err := client.Trades(context.Background(), &TradesRequest{Filter: MarketFilterByMarkets([]common.Hash{{}})}); err == nil {
		t.Fatal("zero condition accepted")
	}
	ids := make([]common.Hash, 21)
	for index := range ids {
		ids[index] = common.BigToHash(decimal.NewFromInt(int64(index + 1)).BigInt())
	}
	if _, err := client.Trades(context.Background(), &TradesRequest{Filter: MarketFilterByMarkets(ids)}); err == nil {
		t.Fatal("too many conditions accepted")
	}
	_, err := client.Trades(context.Background(), &TradesRequest{TakerOnly: ptr(false), TradeFilter: TradeFilterCash(decimal.RequireFromString("0.000001")), Start: ptr(int64(1)), End: ptr(int64(2)), Side: ptr(SideBuy)})
	if err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("validation sent %d requests", requests)
	}
}

func TestV2ErrorsPreserveContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"cursor does not match query","code":"invalid_request","retryable":false,"trace_id":"trace-123","parameter":"cursor"}`))
	}))
	defer server.Close()
	_, err := NewClient(transport.NewClient(server.Client(), server.URL)).Positions(context.Background(), &PositionsRequest{Cursor: ptr("bad")})
	if err == nil || !strings.Contains(err.Error(), "/v2/positions") || !strings.Contains(err.Error(), "cursor") {
		t.Fatalf("error %v", err)
	}
	var apiError *types.Error
	if !errors.As(err, &apiError) || apiError.Message != "cursor does not match query" || apiError.TraceID != "trace-123" || apiError.Retryable == nil || *apiError.Retryable || apiError.Parameter == nil || *apiError.Parameter != "cursor" {
		t.Fatalf("lost error metadata: %+v", apiError)
	}

}

func TestLeaderboardUserHasObjectOrNullShape(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !r.URL.Query().Has("user") {
			t.Error("user missing")
		}
		w.Write([]byte(`{"data":{"user_id":"0x123","rank_pnl":null,"rank_volume":2,"pnl":0.000001,"volume":4}}`))
	}))
	defer server.Close()
	response, err := NewClient(transport.NewClient(server.Client(), server.URL)).LeaderboardUser(context.Background(), &LeaderboardUserRequest{User: common.HexToAddress("0x123")})
	if err != nil {
		t.Fatal(err)
	}
	if response.Data == nil || response.Data.RankPnl != nil || response.Data.RankVolume == nil || *response.Data.RankVolume != 2 || response.Data.Pnl.String() != "0.000001" {
		t.Fatalf("standing %+v", response)
	}
}

func TestActivityPayoutUnavailableIsNotLosingZero(t *testing.T) {
	for _, body := range []string{`{"data":[{"type":"REDEEM","size":1,"price":0,"usdc_size":null}],"pagination":{"has_more":false,"offset":0,"limit":2,"next_cursor":null}}`, `{"data":[{"type":"REDEEM","size":1,"price":0}],"pagination":{"has_more":false,"offset":0,"limit":2,"next_cursor":null}}`} {
		var response ActivityResponse
		if err := json.Unmarshal([]byte(body), &response); err == nil {
			t.Fatalf("unavailable payout became a losing zero: %s", body)
		}
	}
	var response ActivityResponse
	if err := json.Unmarshal([]byte(`{"data":[{"type":"REDEEM","size":1,"price":0,"usdc_size":0}],"pagination":{"has_more":false,"offset":0,"limit":2,"next_cursor":null}}`), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Data) != 1 || !response.Data[0].USDCSize.IsZero() {
		t.Fatalf("losing zero rejected %+v", response)
	}
}
