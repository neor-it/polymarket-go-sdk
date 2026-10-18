package combos

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc struct {
	fn    func(req *http.Request) (*http.Response, error)
	calls []*http.Request
}

func (r *roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	r.calls = append(r.calls, req)
	return r.fn(req)
}

func TestListComboMarkets(t *testing.T) {
	doer := &roundTripFunc{
		fn: func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != comboMarketsPath {
				t.Fatalf("path = %q", req.URL.Path)
			}
			if got := req.URL.Query().Get("exclude"); got != "a,b" {
				t.Fatalf("exclude = %q", got)
			}
			return jsonResponse(`{"markets":[{"id":"combo-1","condition_id":"0xabc","position_ids":["1","2"],"slug":"combo","title":"Combo","outcomes":["YES","NO"],"outcome_prices":["0.25","0.75"]}],"next_cursor":"next"}`)
		},
	}
	client := NewClient(Config{HTTPClient: doer})

	resp, err := client.ListComboMarkets(context.Background(), &ComboMarketListRequest{Limit: 10, Cursor: "cur", Exclude: []string{"a", "b"}})
	if err != nil {
		t.Fatalf("ListComboMarkets returned error: %v", err)
	}
	if len(resp.Markets) != 1 || resp.Markets[0].ID != "combo-1" {
		t.Fatalf("unexpected markets: %+v", resp.Markets)
	}
	if resp.NextCursor != "next" {
		t.Fatalf("next cursor = %q", resp.NextCursor)
	}
}

func TestCreateAcceptAndGetComboRFQ(t *testing.T) {
	call := 0
	doer := &roundTripFunc{
		fn: func(req *http.Request) (*http.Response, error) {
			call++
			if req.Header.Get("POLY_SIGNATURE") == "" {
				t.Fatalf("missing account signature on call %d", call)
			}
			switch call {
			case 1:
				if req.URL.Path != builderRFQPath {
					t.Fatalf("create path = %q", req.URL.Path)
				}
				if req.Header.Get("POLY_BUILDER_SIGNATURE") == "" {
					t.Fatal("missing builder signature")
				}
				return jsonResponse(`{"rfq_id":"rfq-1","status":"AWAITING_REQUESTER_ACCEPTANCE","expires_at":"2026-01-01T00:00:00Z","builder_code":"0x00","request":{"yes_position_id":"123"},"quote":{"quote_id":"quote-1","maker_amount_e6":"1000000","taker_amount_e6":"250000"}}`)
			case 2:
				if req.URL.Path != builderRFQPath+"/rfq-1/accept" {
					t.Fatalf("accept path = %q", req.URL.Path)
				}
				if req.Header.Get("POLY_BUILDER_SIGNATURE") == "" {
					t.Fatal("missing builder signature")
				}
				return jsonResponse(`{"rfq_id":"rfq-1","status":"EXECUTING"}`)
			case 3:
				if req.URL.Path != builderRFQPath+"/rfq-1" {
					t.Fatalf("get path = %q", req.URL.Path)
				}
				if req.Header.Get("POLY_BUILDER_SIGNATURE") != "" {
					t.Fatal("get should use account auth only")
				}
				return jsonResponse(`{"rfq_id":"rfq-1","status":"FILLED","tx_hash":"0xabc"}`)
			default:
				t.Fatalf("unexpected call %d", call)
				return nil, nil
			}
		},
	}
	client := NewClient(Config{
		HTTPClient: doer,
		AccountAuth: &AccountAuth{
			Address:     "0x0000000000000000000000000000000000000001",
			Credentials: Credentials{Key: "key", Secret: "c2VjcmV0", Passphrase: "pass"},
		},
		BuilderAuth: &BuilderAuth{
			Credentials: Credentials{Key: "builder-key", Secret: "c2VjcmV0", Passphrase: "builder-pass"},
		},
		NowUnix: func() int64 { return 1700000000 },
	})

	createResp, err := client.CreateComboRFQ(context.Background(), &CreateRFQRequest{
		SignerAddress:  "0x0000000000000000000000000000000000000001",
		MakerAddress:   "0x0000000000000000000000000000000000000001",
		SignatureType:  3,
		LegPositionIDs: []string{"11", "22"},
		Direction:      DirectionBuy,
		Side:           ComboSideYes,
		RequestedSize:  RequestedSize{Unit: RequestedSizeUnitNotional, ValueE6: "1000000"},
	})
	if err != nil {
		t.Fatalf("CreateComboRFQ returned error: %v", err)
	}
	if createResp.RFQID != "rfq-1" || createResp.Status != RFQStatusAwaitingRequesterAcceptance {
		t.Fatalf("unexpected create response: %+v", createResp)
	}

	acceptResp, err := client.AcceptComboRFQ(context.Background(), &AcceptRFQRequest{
		RFQID:   "rfq-1",
		QuoteID: "quote-1",
		SignedOrder: ComboSignedOrder{
			Salt: "1", Maker: "0x0000000000000000000000000000000000000001", Signer: "0x0000000000000000000000000000000000000001",
			TokenID: "123", MakerAmount: "1000000", TakerAmount: "250000", Side: 0, SignatureType: 3, Timestamp: "1700000000",
			Metadata: zeroBytes32, Builder: zeroBytes32, Signature: "0xsig",
		},
	})
	if err != nil {
		t.Fatalf("AcceptComboRFQ returned error: %v", err)
	}
	if acceptResp.Status != RFQStatusExecuting {
		t.Fatalf("accept status = %q", acceptResp.Status)
	}

	getResp, err := client.GetComboRFQ(context.Background(), "rfq-1")
	if err != nil {
		t.Fatalf("GetComboRFQ returned error: %v", err)
	}
	if getResp.Status != RFQStatusFilled || getResp.TxHash != "0xabc" {
		t.Fatalf("unexpected get response: %+v", getResp)
	}
}

func TestListUserComboPositionsAndActivity(t *testing.T) {
	call := 0
	doer := &roundTripFunc{
		fn: func(req *http.Request) (*http.Response, error) {
			call++
			switch call {
			case 1:
				if req.URL.Path != comboPositionsPath {
					t.Fatalf("positions path = %q", req.URL.Path)
				}
				if got := req.URL.Query().Get("user"); got != "0x1111111111111111111111111111111111111111" {
					t.Fatalf("positions user = %q", got)
				}
				return jsonResponse(`{"data":[{"combo_condition_id":"0x1","combo_position_id":"10","status":"OPEN","current_size":1.000001,"entry_avg_price_usdc":0.5,"entry_cost_usdc":0.5,"entry_fees_usdc":0,"gross_entry_cost_usdc":0.5,"realized_payout_usdc":0,"legs":[{"leg_position_id":"1","leg_current_price":0.5,"leg_outcome_label":"YES"}]}],"pagination":{"has_more":false,"offset":0,"limit":2,"next_cursor":null}}`)
			case 2:
				if req.URL.Path != comboActivityPath {
					t.Fatalf("activity path = %q", req.URL.Path)
				}
				return jsonResponse(`{"data":[{"id":"act-1","type":"SPLIT","amount_usdc":1.000001}],"pagination":{"offset":0,"limit":2,"next_cursor":"next","has_more":true}}`)
			default:
				t.Fatalf("unexpected call %d", call)
				return nil, nil
			}
		},
	}
	client := NewClient(Config{HTTPClient: doer})

	positions, err := client.ListUserComboPositions(context.Background(), &UserComboPositionsRequest{User: "0x1111111111111111111111111111111111111111", Limit: 5})
	if err != nil {
		t.Fatalf("ListUserComboPositions returned error: %v", err)
	}
	if len(positions.Data) != 1 || positions.Data[0].Status != "OPEN" {
		t.Fatalf("unexpected positions: %+v", positions)
	}

	activity, err := client.ListUserComboActivity(context.Background(), &UserComboActivityRequest{User: "0x1111111111111111111111111111111111111111"})
	if err != nil {
		t.Fatalf("ListUserComboActivity returned error: %v", err)
	}
	if len(activity.Data) != 1 || activity.Pagination.NextCursor == nil || *activity.Pagination.NextCursor != "next" {
		t.Fatalf("unexpected activity: %+v", activity)
	}
}

func TestCreateComboRFQBusinessFailureStatus(t *testing.T) {
	doer := &roundTripFunc{
		fn: func(req *http.Request) (*http.Response, error) {
			return jsonResponse(`{"status":"FAILED","error":"no quote"}`)
		},
	}
	client := NewClient(Config{
		HTTPClient: doer,
		AccountAuth: &AccountAuth{
			Address:     "0x0000000000000000000000000000000000000001",
			Credentials: Credentials{Key: "key", Secret: "c2VjcmV0", Passphrase: "pass"},
		},
		BuilderAuth: &BuilderAuth{
			Credentials: Credentials{Key: "builder-key", Secret: "c2VjcmV0", Passphrase: "builder-pass"},
		},
	})

	resp, err := client.CreateComboRFQ(context.Background(), &CreateRFQRequest{
		SignerAddress:  "0x0000000000000000000000000000000000000001",
		MakerAddress:   "0x0000000000000000000000000000000000000001",
		SignatureType:  3,
		LegPositionIDs: []string{"11", "22"},
		Direction:      DirectionBuy,
		Side:           ComboSideYes,
		RequestedSize:  RequestedSize{Unit: RequestedSizeUnitNotional, ValueE6: "1000000"},
	})
	if err != nil {
		t.Fatalf("CreateComboRFQ returned transport error: %v", err)
	}
	if resp.Status != RFQStatusFailed || resp.Error != "no quote" {
		t.Fatalf("unexpected business failure: %+v", resp)
	}
}

func TestGetComboRFQPreservesUnknownStatus(t *testing.T) {
	doer := &roundTripFunc{
		fn: func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != builderRFQPath+"/rfq-unknown" {
				t.Fatalf("get path = %q", req.URL.Path)
			}
			return jsonResponse(`{"rfq_id":"rfq-unknown","status":"PROVIDER_SIDEWAYS"}`)
		},
	}
	client := NewClient(Config{
		HTTPClient: doer,
		AccountAuth: &AccountAuth{
			Address:     "0x0000000000000000000000000000000000000001",
			Credentials: Credentials{Key: "key", Secret: "c2VjcmV0", Passphrase: "pass"},
		},
		NowUnix: func() int64 { return 1700000000 },
	})

	resp, err := client.GetComboRFQ(context.Background(), "rfq-unknown")
	if err != nil {
		t.Fatalf("GetComboRFQ returned error: %v", err)
	}
	if resp.Status != RFQStatus("PROVIDER_SIDEWAYS") {
		t.Fatalf("unknown status was not preserved: %q", resp.Status)
	}
}

func jsonResponse(body string) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}
