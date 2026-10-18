package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/neor-it/polymarket-go-sdk/pkg/transport"
	"net/http"
	"net/http/httptest"
	"testing"
)

const quoteBaseUnits = "9007199254740993000001"

func TestBridgeQuoteAndWithdrawalOfficialWire(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var fields map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
			t.Fatal(err)
		}
		if r.Method != http.MethodPost {
			t.Fatal("incorrect method")
		}
		if r.URL.Path == "/quote" {
			for _, key := range []string{"fromAmountBaseUnit", "recipientAddress", "fromChainId", "toChainId", "fromTokenAddress", "toTokenAddress"} {
				if _, ok := fields[key]; !ok {
					t.Fatalf("missing %s", key)
				}
			}
			if _, ok := fields["fromAmountRaw"]; ok {
				t.Fatal("obsolete quote wire")
			}
			_, _ = fmt.Fprintf(w, `{"estInputUsd":0,"estOutputUsd":1.000000000000000001,"estToTokenBaseUnit":%q,"quoteId":"quote","estFeeBreakdown":{"gasUsd":0}}`, quoteBaseUnits)
			return
		}
		for _, key := range []string{"address", "recipientAddr", "toChainId", "toTokenAddress"} {
			if _, ok := fields[key]; !ok {
				t.Fatalf("missing %s", key)
			}
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, `{"address":{"evm":"destination"}}`)
	}))
	defer server.Close()
	c := NewClient(transport.NewClient(server.Client(), server.URL))
	ctx := context.Background()
	value, err := c.Quote(ctx, &BridgeQuoteRequest{FromChainID: "137", ToChainID: "1", FromTokenAddr: "source", ToTokenAddr: "token", FromAmountBaseUnit: quoteBaseUnits, RecipientAddress: "recipient"})
	if err != nil {
		t.Fatal(err)
	}
	if value.EstimatedInputUSD == nil || !value.EstimatedInputUSD.IsZero() || value.EstimatedOutputUSD.String() != "1.000000000000000001" || *value.EstimatedToTokenBaseUnit != quoteBaseUnits || value.EstimatedFeeBreakdown.GasUSD == nil {
		t.Fatal("estimate precision/zero lost")
	}
	withdrawal, err := c.WithdrawAddress(ctx, &WithdrawAddressRequest{Address: "wallet", ToChainID: "1", ToTokenAddr: "token", RecipientAddr: "recipient"})
	if err != nil || withdrawal.Address.EVM != "destination" {
		t.Fatalf("withdrawal %v %v", withdrawal, err)
	}
	if _, err = c.Quote(ctx, nil); err == nil {
		t.Fatal("nil quote accepted")
	}
	if _, err = c.WithdrawAddress(ctx, &WithdrawAddressRequest{}); err == nil {
		t.Fatal("missing withdrawal inputs accepted")
	}
}
