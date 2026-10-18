package combos_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/neor-it/polymarket-go-sdk/pkg/combos"
	"github.com/shopspring/decimal"
)

const (
	qaComboUser         = "0x0000000000000000000000000000000000000001"
	qaComboToken        = "115792089237316195423570985008687907853269984665640564039457584007913129639935"
	qaComboLimit        = 2
	qaComboMaximumLimit = 1000
)

func TestQAComboV2ReadsPreserveNestedLegPrecisionAndOpaqueCursor(t *testing.T) {
	const cursor = "opaque+/combo=cursor"
	condition := common.HexToHash("0x12")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("user") != qaComboUser || query.Get("cursor") != cursor || query.Get("condition") != condition.Hex() || query.Has("offset") || query.Has("market") {
			t.Errorf("wrong query: %s", r.URL.RawQuery)
		}
		var body string
		switch r.URL.Path {
		case "/v2/positions/combos":
			body = `{"data":[{"combo_position_id":"` + qaComboToken + `","current_size":"0.000001","entry_cost_usdc":"9007199254.740993","entry_avg_price_usdc":"0.5","entry_fees_usdc":"0","gross_entry_cost_usdc":"9007199254.740993","realized_payout_usdc":"0","legs":[{"leg_position_id":"` + qaComboToken + `","leg_current_price":"0.1234567890123456789"}]}],"pagination":{"limit":2,"offset":0,"has_more":true,"next_cursor":"next"}}`
		case "/v2/activity/combos":
			body = `{"data":[{"combo_position_id":"` + qaComboToken + `","amount_usdc":null,"payout_usdc":"0.000001"}],"pagination":{"limit":2,"offset":0,"has_more":false,"next_cursor":null}}`
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	client := combos.NewClient(combos.Config{PublicBaseURL: server.URL, DataBaseURL: server.URL, HTTPClient: server.Client()})
	positions, err := client.ListUserComboPositions(context.Background(), &combos.UserComboPositionsRequest{User: qaComboUser, Cursor: cursor, Limit: qaComboLimit, Conditions: []common.Hash{condition}})
	if err != nil {
		t.Fatal(err)
	}
	if len(positions.Data) != 1 || positions.Data[0].ComboPositionID != qaComboToken || positions.Data[0].Legs[0].LegPositionID != qaComboToken || !positions.Data[0].Legs[0].LegCurrentPrice.Equal(decimal.RequireFromString("0.1234567890123456789")) {
		t.Fatalf("combo precision lost: %+v", positions)
	}
	activities, err := client.ListUserComboActivity(context.Background(), &combos.UserComboActivityRequest{User: qaComboUser, Cursor: cursor, Limit: qaComboLimit, Conditions: []common.Hash{condition}})
	if err != nil {
		t.Fatal(err)
	}
	if len(activities.Data) != 1 || activities.Data[0].AmountUSDC != nil || activities.Data[0].PayoutUSDC == nil || activities.Data[0].PayoutUSDC.String() != "0.000001" {
		t.Fatalf("nullable combo payout lost: %+v", activities)
	}
}

func TestQAComboInvalidSelectorsFailBeforeTransport(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	client := combos.NewClient(combos.Config{PublicBaseURL: server.URL, DataBaseURL: server.URL, HTTPClient: server.Client()})
	for _, request := range []*combos.UserComboActivityRequest{
		nil,
		{User: "not-an-address"},
		{User: qaComboUser, Limit: qaComboMaximumLimit + qaComboLimit},
		{User: qaComboUser, Conditions: []common.Hash{{}}},
	} {
		if _, err := client.ListUserComboActivity(context.Background(), request); err == nil {
			t.Fatalf("invalid request accepted: %+v", request)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("invalid input reached network: %d", calls.Load())
	}
}
