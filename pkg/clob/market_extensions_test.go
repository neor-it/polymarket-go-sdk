package clob

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/neor-it/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/neor-it/polymarket-go-sdk/pkg/transport"
	"net/http"
	"net/http/httptest"
	"testing"
)

const extensionToken = "115792089237316195423570985008687907853269984665640564039457584007913129639935"
const extensionTimestamp = 1700000000

func TestMarketExtensionOfficialWire(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch r.URL.Path {
		case "/markets-by-token/" + extensionToken:
			_, _ = fmt.Fprintf(w, `{"condition_id":"condition","primary_token_id":%q,"secondary_token_id":"2"}`, extensionToken)
		case "/markets/live-activity":
			var ids []string
			if err := json.NewDecoder(r.Body).Decode(&ids); err != nil || len(ids) != 1 || ids[0] != "condition" {
				t.Fatalf("batch requires bare array: %v %v", ids, err)
			}
			_, _ = fmt.Fprint(w, `[{"condition_id":"condition","id":7,"market_slug":"market","tags":["sports"]}]`)
		case "/markets/live-activity/condition":
			_, _ = fmt.Fprint(w, `{"condition_id":"condition","id":7,"question":"Q","event_slug":"event"}`)
		case "/batch-prices-history":
			var req clobtypes.BatchPricesHistoryRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Markets[0] != extensionToken {
				t.Fatal("asset lost")
			}
			_, _ = fmt.Fprintf(w, `{"history":{%q:[{"t":%d,"p":0.5}]}}`, extensionToken, extensionTimestamp)
		case "/books":
			if r.URL.Query().Get("token_ids") != extensionToken+",2" || r.URL.Query().Has("token_id") {
				t.Fatal("incorrect books query")
			}
			_, _ = fmt.Fprint(w, `[]`)
		case "/neg-risk/" + extensionToken:
			_, _ = fmt.Fprint(w, `{"neg_risk":false}`)
		case "/balance-allowance":
			if r.Method != http.MethodPut || r.URL.Query().Get("asset_type") != "CONDITIONAL-V2" || r.URL.Query().Get("token_id") != extensionToken || r.ContentLength > 0 {
				t.Fatalf("PUT query wire: %s %s", r.Method, r.URL.String())
			}
			_, _ = fmt.Fprint(w, `{}`)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client := NewClient(transport.NewClient(server.Client(), server.URL))
	ctx := context.Background()

	market, err := client.MarketByToken(ctx, extensionToken)
	if err != nil || market.PrimaryTokenID != extensionToken || market.SecondaryTokenID != "2" {
		t.Fatalf("market %v %v", market, err)
	}
	batch, err := client.MarketsLiveActivity(ctx, &clobtypes.LiveActivityRequest{ConditionIDs: []string{"condition"}})
	if err != nil || batch[0].MarketSlug != "market" {
		t.Fatalf("batch %v %v", batch, err)
	}
	single, err := client.MarketLiveActivity(ctx, "condition")
	if err != nil || single.EventSlug != "event" {
		t.Fatalf("single %v %v", single, err)
	}
	history, err := client.BatchPricesHistory(ctx, &clobtypes.BatchPricesHistoryRequest{Markets: []string{extensionToken}})
	if err != nil || history.History[extensionToken][0].T != extensionTimestamp {
		t.Fatalf("history %v %v", history, err)
	}
	if _, err = client.OrderBooksQuery(ctx, &clobtypes.BooksQueryRequest{TokenIDs: []string{extensionToken, "2"}}); err != nil {
		t.Fatal(err)
	}
	if _, err = client.NegRiskByPath(ctx, extensionToken); err != nil {
		t.Fatal(err)
	}
	if err = client.PutBalanceAllowance(ctx, &clobtypes.PutBalanceAllowanceRequest{AssetType: clobtypes.AssetTypeConditionalV2, TokenID: extensionToken}); err != nil {
		t.Fatal(err)
	}
	const expectedCalls = 7
	if calls != expectedCalls {
		t.Fatalf("expected %d requests, got %d", expectedCalls, calls)
	}
}
func TestTimeResponseUsesSingleRequest(t *testing.T) {
	for _, body := range []string{`1700000000`, `{"timestamp":1700000000}`} {
		t.Run(body, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; _, _ = fmt.Fprint(w, body) }))
			defer server.Close()
			client := NewClient(transport.NewClient(server.Client(), server.URL))
			value, err := client.Time(context.Background())
			if err != nil || value.Timestamp != extensionTimestamp || calls != 1 {
				t.Fatalf("time=%v err=%v calls=%d", value, err, calls)
			}
		})
	}
}
func TestMarketExtensionRejectsMissingInputs(t *testing.T) {
	client := NewClient(nil)
	ctx := context.Background()
	if _, err := client.BatchPricesHistory(ctx, &clobtypes.BatchPricesHistoryRequest{Markets: make([]string, maxBatchHistoryAssets+1)}); err == nil {
		t.Fatal("oversized history accepted")
	}
	if _, err := client.OrderBooksQuery(ctx, nil); err == nil {
		t.Fatal("nil book query accepted")
	}
	if err := client.PutBalanceAllowance(ctx, nil); err == nil {
		t.Fatal("nil allowance update accepted")
	}
}
