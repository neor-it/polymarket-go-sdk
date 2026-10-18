package clob

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
	"github.com/neor-it/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/neor-it/polymarket-go-sdk/pkg/transport"
	"github.com/neor-it/polymarket-go-sdk/pkg/types"
	"github.com/shopspring/decimal"
)

const upstreamQAMaxUint256 = "115792089237316195423570985008687907853269984665640564039457584007913129639935"
const upstreamQABits = 256
const upstreamQATimestamp = int64(1700000000123)
const upstreamQAOwner = "0x1234567890123456789012345678901234567890"

func TestUpstreamQAPostOrderPreservesUint256JSONNumbersAcrossWallets(t *testing.T) {
	for _, protocol := range []types.ProtocolVersion{types.ProtocolV1, types.ProtocolV2} {
		for _, signatureType := range []auth.SignatureType{auth.SignatureEOA, auth.SignatureProxy, auth.SignatureGnosisSafe, auth.SignaturePoly1271} {
			t.Run(string(protocol)+"/"+big.NewInt(int64(signatureType)).String(), func(t *testing.T) {
				var requests atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					if r.Method != http.MethodPost || r.URL.Path != "/order" {
						t.Errorf("unexpected order request %s %s", r.Method, r.URL.Path)
					}
					raw, err := io.ReadAll(r.Body)
					if err != nil {
						t.Error(err)
						return
					}
					var payload struct {
						Order map[string]json.RawMessage `json:"order"`
					}
					if err := json.Unmarshal(raw, &payload); err != nil {
						t.Error(err)
						return
					}
					if string(payload.Order["salt"]) != upstreamQAMaxUint256 {
						t.Errorf("salt must be a full-width JSON number, got %s", payload.Order["salt"])
					}
					if string(payload.Order["tokenId"]) != `"`+upstreamQAMaxUint256+`"` {
						t.Errorf("asset ID lost its exact string: %s", payload.Order["tokenId"])
					}
					if _, exists := payload.Order["protocol_version"]; exists {
						t.Error("internal protocol selection leaked into order wire")
					}
					if string(payload.Order["timestamp"]) != `"`+big.NewInt(upstreamQATimestamp).String()+`"` {
						t.Errorf("timestamp changed: %s", payload.Order["timestamp"])
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, `{"success":true,"orderID":"qa-order","status":"live"}`)
				}))
				defer server.Close()
				salt, _ := new(big.Int).SetString(upstreamQAMaxUint256, 10)
				token := new(big.Int).Set(salt)
				sig := int(signatureType)
				order := &clobtypes.SignedOrder{Owner: "qa-owner", Signature: "0x01", Order: clobtypes.Order{ProtocolVersion: protocol, Salt: types.U256{Int: salt}, Maker: common.HexToAddress(upstreamQAOwner), Signer: common.HexToAddress(upstreamQAOwner), TokenID: types.U256{Int: token}, MakerAmount: decimal.NewFromInt(1), TakerAmount: decimal.NewFromInt(1), Side: "BUY", Timestamp: upstreamQATimestamp, SignatureType: &sig}}
				client := NewClient(transport.NewClient(server.Client(), server.URL))
				if _, err := client.PostOrder(context.Background(), order); err != nil {
					t.Fatal(err)
				}
				if requests.Load() != 1 {
					t.Fatalf("unexpected HTTP request count: %d", requests.Load())
				}
				if salt.String() != upstreamQAMaxUint256 || token.String() != upstreamQAMaxUint256 {
					t.Fatal("wire serialization mutated uint256 identity")
				}
			})
		}
	}
}

func TestUpstreamQAInvalidSaltNeverReachesHTTP(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); _, _ = io.WriteString(w, `{}`) }))
	defer server.Close()
	client := NewClient(transport.NewClient(server.Client(), server.URL))
	for _, salt := range []*big.Int{big.NewInt(-1), new(big.Int).Lsh(big.NewInt(1), upstreamQABits)} {
		order := &clobtypes.SignedOrder{Owner: "qa-owner", Signature: "0x01", Order: clobtypes.Order{Side: "BUY", Salt: types.U256{Int: salt}}}
		if _, err := client.PostOrder(t.Context(), order); err == nil || !strings.Contains(err.Error(), "uint256") {
			t.Fatalf("invalid salt accepted: %v", err)
		}
	}
	if requests.Load() != 0 {
		t.Fatal("invalid uint256 was submitted")
	}
}

func TestUpstreamQACLOBReadExtensionsMatchOfficialShapes(t *testing.T) {
	const condition = "qa-condition"
	const secondAsset = "42"
	const historyTimestamp = int64(1700000000)
	const historyPrice = 0.25
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/books":
			if r.Method != http.MethodGet || r.URL.Query().Get("token_ids") != upstreamQAMaxUint256+","+secondAsset {
				t.Errorf("GET books query: %s %s", r.Method, r.URL)
			}
			_, _ = io.WriteString(w, `[{"market":"qa-condition","asset_id":"`+upstreamQAMaxUint256+`","bids":[],"asks":[]}]`)
		case "/markets-by-token/" + upstreamQAMaxUint256:
			_, _ = io.WriteString(w, `{"condition_id":"qa-condition","primary_token_id":"`+upstreamQAMaxUint256+`","secondary_token_id":"42"}`)
		case "/markets/live-activity":
			var conditions []string
			if err := json.NewDecoder(r.Body).Decode(&conditions); err != nil {
				t.Error(err)
			}
			if r.Method != http.MethodPost || len(conditions) != 1 || conditions[0] != condition {
				t.Errorf("live activity requires bare condition array: %v", conditions)
			}
			_, _ = io.WriteString(w, `[{"condition_id":"qa-condition","id":1,"question":"official activity","tags":["sports"]}]`)
		case "/markets/live-activity/" + condition:
			_, _ = io.WriteString(w, `{"condition_id":"qa-condition","id":1,"question":"official activity","tags":["sports"]}`)
		case "/batch-prices-history":
			var payload struct {
				Markets  []string `json:"markets"`
				Interval string   `json:"interval"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			if r.Method != http.MethodPost || len(payload.Markets) != 1 || payload.Markets[0] != upstreamQAMaxUint256 || payload.Interval != "1d" {
				t.Errorf("history input uses assets, not conditions: %+v", payload)
			}
			_, _ = io.WriteString(w, `{"history":{"`+upstreamQAMaxUint256+`":[{"t":1700000000,"p":0.25}]}}`)
		case "/balance-allowance":
			if r.Method != http.MethodPut || r.URL.Query().Get("asset_type") != "CONDITIONAL-V2" || r.URL.Query().Get("token_id") != upstreamQAMaxUint256 || r.URL.Query().Get("signature_type") != "3" {
				t.Errorf("allowance refresh query %s %s", r.Method, r.URL)
			}
			// This is only an acknowledgement; no amount field is returned by the API.
			_, _ = io.WriteString(w, `{}`)
		default:
			t.Errorf("unexpected extension route %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client := NewClient(transport.NewClient(server.Client(), server.URL))
	books, err := client.OrderBooksQuery(t.Context(), &clobtypes.BooksQueryRequest{TokenIDs: []string{upstreamQAMaxUint256, secondAsset}})
	if err != nil || len(books) != 1 || books[0].AssetID != upstreamQAMaxUint256 {
		t.Fatalf("books result: %+v %v", books, err)
	}
	market, err := client.MarketByToken(t.Context(), upstreamQAMaxUint256)
	if err != nil || market.PrimaryTokenID != upstreamQAMaxUint256 || market.SecondaryTokenID != secondAsset || market.ConditionID != condition {
		t.Fatalf("token mapping: %+v %v", market, err)
	}
	batch, err := client.MarketsLiveActivity(t.Context(), &clobtypes.LiveActivityRequest{ConditionIDs: []string{condition}})
	if err != nil || len(batch) != 1 || batch[0].ConditionID != condition {
		t.Fatalf("batch activity: %+v %v", batch, err)
	}
	single, err := client.MarketLiveActivity(t.Context(), condition)
	if err != nil || single.ConditionID != condition {
		t.Fatalf("single activity object: %+v %v", single, err)
	}
	history, err := client.BatchPricesHistory(t.Context(), &clobtypes.BatchPricesHistoryRequest{Markets: []string{upstreamQAMaxUint256}, Interval: "1d"})
	if err != nil || len(history.History[upstreamQAMaxUint256]) != 1 || history.History[upstreamQAMaxUint256][0].T != historyTimestamp || history.History[upstreamQAMaxUint256][0].P != historyPrice {
		t.Fatalf("batch history: %+v %v", history, err)
	}
	signature := int(auth.SignaturePoly1271)
	if err := client.PutBalanceAllowance(t.Context(), &clobtypes.PutBalanceAllowanceRequest{AssetType: clobtypes.AssetTypeConditionalV2, TokenID: upstreamQAMaxUint256, SignatureType: &signature}); err != nil {
		t.Fatalf("empty allowance ACK is not a balance: %v", err)
	}
	before := requests.Load()
	signature = -1
	for _, req := range []*clobtypes.PutBalanceAllowanceRequest{nil, {AssetType: clobtypes.AssetTypeConditionalV2}, {AssetType: "unknown"}, {AssetType: clobtypes.AssetTypeCollateral, SignatureType: &signature}} {
		if err := client.PutBalanceAllowance(t.Context(), req); err == nil {
			t.Fatal("invalid allowance request accepted")
		}
	}
	if requests.Load() != before {
		t.Fatal("invalid permission refresh reached HTTP")
	}
}

func TestUpstreamQAResponseTimeAndRewardsRemainExact(t *testing.T) {
	const exact = "9007199254740993.000001"
	for _, raw := range []string{exact, `"` + exact + `"`} {
		var reward clobtypes.TotalUserEarning
		if err := json.Unmarshal([]byte(`{"earnings":`+raw+`,"asset_rate":0.000001}`), &reward); err != nil || reward.Earnings != exact || reward.AssetRate != "0.000001" {
			t.Fatalf("reward amount lost precision: %+v %v", reward, err)
		}
	}
	for _, raw := range []string{`{}`, `{"earnings":null,"asset_rate":1}`, `{"earnings":0,"asset_rate":null}`} {
		var reward clobtypes.TotalUserEarning
		if err := json.Unmarshal([]byte(raw), &reward); err == nil {
			t.Fatalf("unknown reward became zero: %s", raw)
		}
	}
	for _, raw := range []string{`1700000000`, `{"timestamp":1700000000}`} {
		var timestamp clobtypes.TimeResponse
		if err := json.Unmarshal([]byte(raw), &timestamp); err != nil || timestamp.Timestamp != 1700000000 {
			t.Fatalf("time response: %+v %v", timestamp, err)
		}
	}
}
