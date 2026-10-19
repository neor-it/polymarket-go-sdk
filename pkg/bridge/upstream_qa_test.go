package bridge

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/neor-it/polymarket-go-sdk/pkg/transport"
)

const upstreamQAAmount = "115792089237316195423570985008687907853269984665640564039457584007913129639935"
const upstreamQASolanaRecipient = "CrvTBvzryYxBHbWu2TiQpcqD5M7Le7iBKzVmEj3f36Jb"
const upstreamQAEVM = "0x9156dd10bea4c8d7e2d591b633d1694b1d764756"

func TestUpstreamQABridgeQuoteAndWithdrawalUseOfficialCamelCaseSchema(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodPost {
			t.Errorf("unexpected bridge method %s", r.Method)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/quote":
			if len(body) != 6 || body["fromAmountBaseUnit"] != upstreamQAAmount || body["recipientAddress"] != upstreamQASolanaRecipient || body["toChainId"] != "1151111081099710" {
				t.Errorf("wrong official quote fields: %v", body)
			}
			_, _ = io.WriteString(w, `{"estCheckoutTimeMs":25000,"estFeeBreakdown":{"gasUsd":0.003854,"appFeeUsd":0},"estInputUsd":14.488305,"estOutputUsd":14.488305,"estToTokenBaseUnit":"14491203","quoteId":"qa-quote"}`)
		case "/withdraw":
			if len(body) != 4 || body["address"] != upstreamQAEVM || body["recipientAddr"] != upstreamQASolanaRecipient || body["toTokenAddress"] != "qa-token" {
				t.Errorf("wrong official withdrawal fields: %v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"address":{"evm":"`+upstreamQAEVM+`","svm":"`+upstreamQASolanaRecipient+`","btc":"qa-bitcoin-address"},"note":"explicit transfer remains required"}`)
		default:
			t.Errorf("unexpected bridge path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client := NewClient(transport.NewClient(server.Client(), server.URL))
	request := BridgeQuoteRequest{FromChainID: "137", ToChainID: "1151111081099710", FromTokenAddr: upstreamQAEVM, ToTokenAddr: "qa-token", FromAmountBaseUnit: upstreamQAAmount, RecipientAddress: upstreamQASolanaRecipient}
	quote, err := client.Quote(t.Context(), &request)
	if err != nil || quote.QuoteID != "qa-quote" || quote.EstimatedToTokenBaseUnit == nil || *quote.EstimatedToTokenBaseUnit != "14491203" || quote.EstimatedInputUSD == nil || quote.EstimatedInputUSD.String() != "14.488305" || quote.EstimatedFeeBreakdown == nil || quote.EstimatedFeeBreakdown.GasUSD == nil || quote.EstimatedFeeBreakdown.GasUSD.String() != "0.003854" {
		t.Fatalf("quote response did not match exact schema: %+v %v", quote, err)
	}
	withdrawal := WithdrawAddressRequest{Address: upstreamQAEVM, ToChainID: "1151111081099710", ToTokenAddr: "qa-token", RecipientAddr: upstreamQASolanaRecipient}
	addresses, err := client.WithdrawAddress(t.Context(), &withdrawal)
	if err != nil || addresses.Address.SVM != upstreamQASolanaRecipient || addresses.Address.EVM != upstreamQAEVM {
		t.Fatalf("withdrawal address envelope: %+v %v", addresses, err)
	}
	before := requests.Load()
	for _, clear := range []func(*BridgeQuoteRequest){
		func(r *BridgeQuoteRequest) { r.FromChainID = "" }, func(r *BridgeQuoteRequest) { r.ToChainID = "" }, func(r *BridgeQuoteRequest) { r.FromTokenAddr = "" }, func(r *BridgeQuoteRequest) { r.ToTokenAddr = "" }, func(r *BridgeQuoteRequest) { r.FromAmountBaseUnit = "" }, func(r *BridgeQuoteRequest) { r.RecipientAddress = "" },
	} {
		invalid := request
		clear(&invalid)
		if _, err := client.Quote(t.Context(), &invalid); err == nil {
			t.Fatal("missing OpenAPI-required quote field accepted")
		}
	}
	for _, clear := range []func(*WithdrawAddressRequest){
		func(r *WithdrawAddressRequest) { r.Address = "" }, func(r *WithdrawAddressRequest) { r.ToChainID = "" }, func(r *WithdrawAddressRequest) { r.ToTokenAddr = "" }, func(r *WithdrawAddressRequest) { r.RecipientAddr = "" },
	} {
		invalid := withdrawal
		clear(&invalid)
		if _, err := client.WithdrawAddress(t.Context(), &invalid); err == nil {
			t.Fatal("missing OpenAPI-required withdrawal field accepted")
		}
	}
	if requests.Load() != before {
		t.Fatal("invalid required fields reached bridge HTTP")
	}
}
