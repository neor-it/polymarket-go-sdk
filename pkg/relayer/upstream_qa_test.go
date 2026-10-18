package relayer

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
	"github.com/neor-it/polymarket-go-sdk/pkg/transport"
	"github.com/neor-it/polymarket-go-sdk/pkg/types"
)

const upstreamQARelayerOwner = "0x6e0c80c90ea6c15917308f820eac91ce2724b5b5"
const upstreamQARelayerNonce = "115792089237316195423570985008687907853269984665640564039457584007913129639935"
const upstreamQARelayerKey = "qa-relayer-key"

func TestUpstreamQARelayerOfficialArraysQueriesAndFlatSubmit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/nonce", "/relay-payload":
			if r.URL.Query().Get("address") != upstreamQARelayerOwner || r.URL.Query().Get("type") != "SAFE" || r.URL.Query().Has("signer") {
				t.Errorf("wrong official query: %s", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"address":"`+upstreamQARelayerOwner+`","nonce":"`+upstreamQARelayerNonce+`"}`)
		case "/submit":
			if r.Method != http.MethodPost {
				t.Errorf("submit method %s", r.Method)
			}
			var body map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if _, exists := body["transaction"]; exists {
				t.Error("upstream's unsupported nested transaction body was sent")
			}
			if string(body["nonce"]) != `"`+upstreamQARelayerNonce+`"` || string(body["type"]) != `"SAFE"` || body["signatureParams"] == nil {
				t.Errorf("flat request does not match official schema: %v", body)
			}
			if r.Header.Get(HeaderRelayerAPIKey) != upstreamQARelayerKey || r.Header.Get(HeaderRelayerAPIKeyAddress) != upstreamQARelayerOwner || r.Header.Get(auth.HeaderPolyAPIKey) != "" {
				t.Errorf("wrong relayer authentication: %v", r.Header)
			}
			_, _ = io.WriteString(w, `{"transactionID":"qa-id","state":"STATE_NEW"}`)
		case "/transaction", "/transactions":
			if r.Method != http.MethodGet {
				t.Error("history is not a GET")
			}
			if r.URL.Path == "/transaction" && r.URL.Query().Get("id") != "qa-id" {
				t.Error("transaction lookup ID is missing")
			}
			if r.URL.Path == "/transactions" && (r.URL.RawQuery != "" || r.Header.Get(HeaderRelayerAPIKey) != upstreamQARelayerKey) {
				t.Error("unsupported pagination or missing authenticated history")
			}
			_, _ = io.WriteString(w, `[{"transactionID":"qa-id","transactionHash":"0x38cbfbeae8fffa4e2b187ee5978d3ee9cafc53af0363ed90a35b7ea9016535d8","nonce":"`+upstreamQARelayerNonce+`","proxyAddress":"`+upstreamQARelayerOwner+`","state":"STATE_CONFIRMED","type":"SAFE"}]`)
		case "/relayer/api/keys":
			if r.Header.Get(HeaderRelayerAPIKey) != upstreamQARelayerKey || r.Header.Get(HeaderRelayerAPIKeyAddress) != upstreamQARelayerOwner {
				t.Error("key-list request used the wrong auth mechanism")
			}
			_, _ = io.WriteString(w, `[{"apiKey":"listed-key","address":"`+upstreamQARelayerOwner+`","createdAt":"2026-02-24T18:20:11.237485Z","updatedAt":"2026-02-24T18:20:11.237485Z"}]`)
		default:
			t.Errorf("unexpected relayer path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	credentials := &RelayerCredentials{APIKey: upstreamQARelayerKey, Address: upstreamQARelayerOwner}
	original := NewClient(transport.NewClient(server.Client(), server.URL))
	client := original.WithRelayerCredentials(credentials)
	credentials.APIKey = "mutated"
	request := &SubmitRequest{From: upstreamQARelayerOwner, To: upstreamQARelayerOwner, ProxyWallet: upstreamQARelayerOwner, Data: "0x12", Nonce: upstreamQARelayerNonce, Signature: "0xab", SignatureParams: &SignatureParams{}, Type: TransactionSafe}
	response, err := client.Submit(t.Context(), request)
	if err != nil || response.TransactionID != "qa-id" {
		t.Fatalf("submit: %+v %v", response, err)
	}
	payload, err := client.GetRelayPayload(t.Context(), &NonceRequest{Address: upstreamQARelayerOwner, Type: TransactionSafe})
	if err != nil || payload.Address != upstreamQARelayerOwner || payload.Nonce != upstreamQARelayerNonce {
		t.Fatalf("relay payload: %+v %v", payload, err)
	}
	nonce, err := client.GetNonce(t.Context(), &NonceRequest{Address: upstreamQARelayerOwner, Type: TransactionSafe})
	if err != nil || nonce.Nonce != upstreamQARelayerNonce {
		t.Fatalf("nonce: %+v %v", nonce, err)
	}
	for _, read := range []func(context.Context) ([]Transaction, error){client.GetTransactions, func(ctx context.Context) ([]Transaction, error) { return client.GetTransaction(ctx, "qa-id") }} {
		txs, err := read(t.Context())
		if err != nil || len(txs) != 1 || txs[0].Nonce != upstreamQARelayerNonce || txs[0].ProxyAddress != upstreamQARelayerOwner {
			t.Fatalf("transaction array: %+v %v", txs, err)
		}
	}
	keys, err := client.GetAPIKeys(t.Context())
	if err != nil || len(keys) != 1 || keys[0].APIKey != "listed-key" || keys[0].Address != upstreamQARelayerOwner {
		t.Fatalf("API keys: %+v %v", keys, err)
	}
	if _, err := original.GetTransactions(t.Context()); err == nil {
		t.Fatal("credential clone mutated the original client")
	}
	if _, err := client.WithRelayerCredentials(nil).GetAPIKeys(t.Context()); err == nil {
		t.Fatal("removed credentials still authorize key reads")
	}
}

func TestUpstreamQARelayerBuilderSignsExactBodyAndDoesNotAuthorizeAPIKeys(t *testing.T) {
	const secret = "qa-builder-secret"
	const builderKey = "qa-builder-key"
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write([]byte(r.Header.Get(auth.HeaderPolyBuilderTimestamp) + r.Method + r.URL.Path + string(body)))
		if r.Header.Get(auth.HeaderPolyBuilderAPIKey) != builderKey || r.Header.Get(auth.HeaderPolyBuilderSignature) != base64.URLEncoding.EncodeToString(mac.Sum(nil)) {
			t.Error("builder HMAC did not cover the exact transmitted bytes")
		}
		if r.Header.Get(HeaderRelayerAPIKey) != "" || r.Header.Get(auth.HeaderPolyAPIKey) != "" {
			t.Error("authentication mechanisms were mixed")
		}
		_, _ = io.WriteString(w, `[]`)
	}))
	defer server.Close()
	client := NewClient(transport.NewClient(server.Client(), server.URL)).WithBuilderConfig(&auth.BuilderConfig{Local: &auth.BuilderCredentials{Key: builderKey, Secret: base64.StdEncoding.EncodeToString([]byte(secret)), Passphrase: "qa-pass"}})
	if _, err := client.GetTransactions(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetAPIKeys(t.Context()); err == nil {
		t.Fatal("builder credentials cannot authorize relayer API-key enumeration")
	}
	if calls.Load() != 1 {
		t.Fatalf("unexpected HTTP requests: %d", calls.Load())
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := client.GetTransactions(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("caller cancellation lost: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatal("cancelled request reached the server")
	}
}

func TestUpstreamQARelayerForbiddenResponseRemainsTypedAndIsNotRetried(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"error":"not allowed"}`)
	}))
	defer server.Close()
	client := NewClient(transport.NewClient(server.Client(), server.URL)).WithRelayerCredentials(&RelayerCredentials{APIKey: upstreamQARelayerKey, Address: upstreamQARelayerOwner})
	_, err := client.GetAPIKeys(t.Context())
	var apiError *types.Error
	if !errors.As(err, &apiError) || apiError.Status != http.StatusForbidden || apiError.Message != "not allowed" {
		t.Fatalf("provider403 contract lost: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("forbidden request retried %d times", calls.Load())
	}
}
