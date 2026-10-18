package relayer

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
	"github.com/neor-it/polymarket-go-sdk/pkg/transport"
)

const (
	apiFixtureAddress = "0x0000000000000000000000000000000000000011"
	apiFixtureNonce   = "115792089237316195423570985008687907853269984665640564039457584007913129639935"
)

func TestRelayerGenericAPIUsesOfficialWireAndAuthentication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/submit":
			if r.Method != http.MethodPost {
				t.Errorf("submit method %s", r.Method)
			}
			if r.Header.Get(HeaderRelayerAPIKey) != "fixture-key" || r.Header.Get(HeaderRelayerAPIKeyAddress) != apiFixtureAddress || r.Header.Get(auth.HeaderPolyAPIKey) != "" {
				t.Errorf("wrong relayer authentication: %v", r.Header)
			}
			var payload map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			for _, key := range []string{"from", "to", "proxyWallet", "data", "nonce", "signature", "signatureParams", "type"} {
				if _, ok := payload[key]; !ok {
					t.Errorf("missing flat submit field %s", key)
				}
			}
			if _, ok := payload["transaction"]; ok {
				t.Error("unsupported upstream transaction wrapper sent")
			}
			if string(payload["nonce"]) != `"`+apiFixtureNonce+`"` {
				t.Errorf("nonce precision lost: %s", payload["nonce"])
			}
			_, _ = io.WriteString(w, `{"transactionID":"tx-fixture","state":"STATE_NEW"}`)
		case "/nonce", "/relay-payload":
			if r.URL.Query().Get("address") != apiFixtureAddress || r.URL.Query().Get("type") != string(TransactionProxy) || r.URL.Query().Has("signer") {
				t.Errorf("wrong nonce query: %s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"address":"`+apiFixtureAddress+`","nonce":"`+apiFixtureNonce+`"}`)
		case "/deployed":
			if r.URL.Query().Get("address") != apiFixtureAddress || r.URL.Query().Get("type") != string(TransactionWallet) {
				t.Errorf("wrong deployment query: %s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"deployed":true}`)
		case "/transaction", "/transactions":
			if r.URL.Path == "/transactions" && r.Header.Get(HeaderRelayerAPIKey) != "fixture-key" {
				t.Error("recent transactions missing authentication")
			}
			_, _ = io.WriteString(w, `[{"transactionID":"tx-fixture","transactionHash":"0xabc","nonce":"`+apiFixtureNonce+`","data":"0x12","value":"0","signature":"0xab","owner":"`+apiFixtureAddress+`","metadata":"fixture","state":"STATE_NEW"}]`)
		case "/relayer/api/keys":
			if r.Header.Get(HeaderRelayerAPIKey) != "fixture-key" || r.Header.Get(HeaderRelayerAPIKeyAddress) != apiFixtureAddress {
				t.Error("API key endpoint missing relayer credential headers")
			}
			_, _ = io.WriteString(w, `[{"apiKey":"listed-key","address":"`+apiFixtureAddress+`","createdAt":"created","updatedAt":"updated"}]`)
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
		}
	}))
	defer server.Close()
	credentials := &RelayerCredentials{APIKey: "fixture-key", Address: apiFixtureAddress}
	client := NewClient(transport.NewClient(server.Client(), server.URL)).WithRelayerCredentials(credentials)
	credentials.APIKey = "mutated"
	ctx := context.Background()
	submitted, err := client.Submit(ctx, validAPISubmit())
	if err != nil || submitted.TransactionID != "tx-fixture" || submitted.State != StateNew {
		t.Fatalf("submit response %v, %v", submitted, err)
	}
	nonce, err := client.GetNonce(ctx, &NonceRequest{Address: apiFixtureAddress, Type: TransactionProxy})
	if err != nil || nonce.Nonce != apiFixtureNonce {
		t.Fatalf("nonce response %v, %v", nonce, err)
	}
	payload, err := client.GetRelayPayload(ctx, &NonceRequest{Address: apiFixtureAddress, Type: TransactionProxy})
	if err != nil || payload.Address != apiFixtureAddress || payload.Nonce != apiFixtureNonce {
		t.Fatalf("relay payload response %v, %v", payload, err)
	}
	deployed, err := client.GetDeployed(ctx, &DeployedRequest{Address: apiFixtureAddress, Type: TransactionWallet})
	if err != nil || !deployed.Deployed {
		t.Fatalf("deployment response %v, %v", deployed, err)
	}
	txs, err := client.GetTransactions(ctx)
	if err != nil || len(txs) != 1 || txs[0].Nonce != apiFixtureNonce || txs[0].TransactionHash != "0xabc" || txs[0].Metadata != "fixture" {
		t.Fatalf("transactions response %v, %v", txs, err)
	}
	txs, err = client.GetTransaction(ctx, "tx-fixture")
	if err != nil || len(txs) != 1 || txs[0].State != StateNew {
		t.Fatalf("existing array response changed: %v, %v", txs, err)
	}
	keys, err := client.GetAPIKeys(ctx)
	if err != nil || len(keys) != 1 || keys[0].APIKey != "listed-key" || keys[0].Address != apiFixtureAddress {
		t.Fatalf("keys response %v, %v", keys, err)
	}
}

func validAPISubmit() *SubmitRequest {
	return &SubmitRequest{From: apiFixtureAddress, To: apiFixtureAddress, ProxyWallet: apiFixtureAddress, Data: "0x", Nonce: apiFixtureNonce, Signature: "0x12", SignatureParams: &SignatureParams{GasPrice: "0", Operation: "0"}, Type: TransactionSafe}
}

func TestRelayerBuilderAuthenticationSignsExactGenericPayload(t *testing.T) {
	builder := &auth.BuilderConfig{Local: &auth.BuilderCredentials{Key: "builder-key", Secret: "c2VjcmV0", Passphrase: "builder-pass"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		signature, err := auth.SignHMAC(builder.Local.Secret, r.Header.Get(auth.HeaderPolyBuilderTimestamp)+r.Method+r.URL.Path+string(body))
		if err != nil || signature != r.Header.Get(auth.HeaderPolyBuilderSignature) || r.Header.Get(auth.HeaderPolyBuilderAPIKey) != builder.Local.Key || r.Header.Get(HeaderRelayerAPIKey) != "" {
			t.Errorf("builder HMAC mismatch: %v", err)
		}
		if r.Method == http.MethodGet {
			_, _ = io.WriteString(w, `[]`)
			return
		}
		_, _ = io.WriteString(w, `{"transactionID":"builder-tx","state":"STATE_NEW"}`)
	}))
	defer server.Close()
	client := NewClient(transport.NewClient(server.Client(), server.URL)).WithBuilderConfig(builder)
	if _, err := client.Submit(context.Background(), validAPISubmit()); err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetTransactions(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetAPIKeys(context.Background()); err == nil {
		t.Fatal("key management must not substitute builder credentials for relayer credentials")
	}
}

func TestRelayerRejectsAmbiguousDomainAndMalformedSubmitBeforeHTTP(t *testing.T) {
	client := NewClient(transport.NewClient(apiRejectDoer{t: t}, "https://fixture.invalid")).WithRelayerCredentials(&RelayerCredentials{APIKey: "key", Address: apiFixtureAddress})
	ctx := context.Background()
	for _, request := range []*NonceRequest{nil, {Address: apiFixtureAddress}, {Address: apiFixtureAddress, Type: TransactionWallet}, {Address: "bad", Type: TransactionSafe}} {
		if _, err := client.GetNonce(ctx, request); err == nil {
			t.Fatal("invalid nonce domain accepted")
		}
		if _, err := client.GetRelayPayload(ctx, request); err == nil {
			t.Fatal("invalid relay payload domain accepted")
		}
	}
	for _, mutate := range []func(*SubmitRequest){
		func(r *SubmitRequest) { r.From = "bad" },
		func(r *SubmitRequest) { r.Type = TransactionWallet },
		func(r *SubmitRequest) { r.SignatureParams = nil },
		func(r *SubmitRequest) { r.Nonce = "-1" },
		func(r *SubmitRequest) { r.Nonce = "01" },
		func(r *SubmitRequest) { r.Nonce += "0" },
		func(r *SubmitRequest) { r.Signature = "0x" },
		func(r *SubmitRequest) { r.Data = "not hex" },
	} {
		request := validAPISubmit()
		mutate(request)
		if _, err := client.Submit(ctx, request); err == nil {
			t.Fatalf("invalid submit accepted: %+v", request)
		}
	}
	if _, err := client.WithRelayerCredentials(nil).Submit(ctx, validAPISubmit()); err == nil || !strings.Contains(err.Error(), "credentials") {
		t.Fatalf("missing authentication accepted: %v", err)
	}
}

type apiRejectDoer struct{ t *testing.T }

func (d apiRejectDoer) Do(*http.Request) (*http.Response, error) {
	d.t.Fatal("invalid request reached HTTP boundary")
	return nil, nil
}
