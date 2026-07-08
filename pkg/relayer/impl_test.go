package relayer

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
	"github.com/neor-it/polymarket-go-sdk/pkg/transport"
)

// mockRelayer is a minimal HTTP server that records incoming requests.
type mockRelayer struct {
	responses map[string]interface{}
	reqs      []*http.Request
	bodies    []string
}

func (m *mockRelayer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.reqs = append(m.reqs, r)
	if r.Body != nil {
		b, _ := io.ReadAll(r.Body)
		m.bodies = append(m.bodies, string(b))
	}
	key := r.URL.Path
	if resp, ok := m.responses[key]; ok {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(`{"error":"not found"}`))
}

func newTestClient(t *testing.T, mock *mockRelayer) (Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(mock)
	t.Cleanup(srv.Close)
	tc := transport.NewClient(srv.Client(), srv.URL)
	signer, err := auth.NewPrivateKeySigner(testPrivKey, 137)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}
	return NewClient(tc).WithSigner(signer), srv
}

func TestDeployDepositWallet(t *testing.T) {
	mock := &mockRelayer{
		responses: map[string]interface{}{
			"/submit": SubmitResponse{TransactionID: "tx123", State: StateNew},
		},
	}
	client, _ := newTestClient(t, mock)
	owner := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")

	resp, err := client.DeployDepositWallet(context.Background(), owner)
	if err != nil {
		t.Fatalf("DeployDepositWallet failed: %v", err)
	}
	if resp.TransactionID != "tx123" {
		t.Fatalf("got TransactionID %q, want tx123", resp.TransactionID)
	}

	if len(mock.bodies) == 0 {
		t.Fatal("expected a POST body")
	}
	var body map[string]string
	if err := json.Unmarshal([]byte(mock.bodies[0]), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["type"] != "WALLET-CREATE" {
		t.Fatalf("type = %q, want WALLET-CREATE", body["type"])
	}
	if !strings.EqualFold(body["from"], owner.Hex()) {
		t.Fatalf("from = %q, want %s", body["from"], owner.Hex())
	}
}

func TestDeployDepositWallet_RequiresOwner(t *testing.T) {
	mock := &mockRelayer{responses: map[string]interface{}{}}
	client, _ := newTestClient(t, mock)

	_, err := client.DeployDepositWallet(context.Background(), common.Address{})
	if err == nil || !strings.Contains(err.Error(), "owner address is required") {
		t.Fatalf("expected owner validation error, got %v", err)
	}
}

func TestGetWalletNonce(t *testing.T) {
	mock := &mockRelayer{
		responses: map[string]interface{}{
			"/nonce": map[string]string{"nonce": "7"},
		},
	}
	client, _ := newTestClient(t, mock)
	owner := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")

	nonce, err := client.GetWalletNonce(context.Background(), owner)
	if err != nil {
		t.Fatalf("GetWalletNonce failed: %v", err)
	}
	if nonce != "7" {
		t.Fatalf("nonce = %q, want 7", nonce)
	}
}

func TestExecuteWalletBatch(t *testing.T) {
	mock := &mockRelayer{
		responses: map[string]interface{}{
			"/nonce":  map[string]string{"nonce": "0"},
			"/submit": SubmitResponse{TransactionID: "batch-tx1", State: StateNew},
		},
	}
	client, _ := newTestClient(t, mock)
	owner := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	wallet := common.HexToAddress("0x9c90cad2e22a1E9b4a9aB3F95f7f14d08Ce78ade")
	token := common.HexToAddress("0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174") // pUSDC
	spender := common.HexToAddress("0x4bFb41d5B3570DeFd03C39a9A4D8dE6Bd8B8982E")

	calls := []DepositWalletCall{ERC20ApproveCall(token, spender, MaxUint256)}
	deadline := time.Now().Add(24 * time.Hour).Unix()

	resp, err := client.ExecuteWalletBatch(context.Background(), owner, wallet, calls, deadline)
	if err != nil {
		t.Fatalf("ExecuteWalletBatch failed: %v", err)
	}
	if resp.TransactionID != "batch-tx1" {
		t.Fatalf("transactionID = %q, want batch-tx1", resp.TransactionID)
	}

	// Verify the batch body contains a signature
	var submitBody map[string]interface{}
	for _, b := range mock.bodies {
		if err := json.Unmarshal([]byte(b), &submitBody); err == nil {
			if submitBody["type"] == "WALLET" {
				break
			}
		}
	}
	if submitBody["type"] != "WALLET" {
		t.Fatal("expected WALLET submit body")
	}
	sig, _ := submitBody["signature"].(string)
	if !strings.HasPrefix(sig, "0x") || len(sig) < 10 {
		t.Fatalf("signature looks wrong: %q", sig)
	}
}

func TestExecuteWalletBatch_RequiresSigner(t *testing.T) {
	mock := &mockRelayer{responses: map[string]interface{}{}}
	srv := httptest.NewServer(mock)
	defer srv.Close()

	tc := transport.NewClient(srv.Client(), srv.URL)
	client := NewClient(tc) // no signer

	_, err := client.ExecuteWalletBatch(
		context.Background(),
		common.HexToAddress("0x1"),
		common.HexToAddress("0x2"),
		[]DepositWalletCall{{Target: "0x3", Value: "0", Data: "0x"}},
		time.Now().Add(time.Hour).Unix(),
	)
	if err == nil || !strings.Contains(err.Error(), "signer is required") {
		t.Fatalf("expected signer error, got %v", err)
	}
}

func TestIsWalletDeployed(t *testing.T) {
	mock := &mockRelayer{
		responses: map[string]interface{}{
			"/deployed": map[string]bool{"deployed": true},
		},
	}
	client, _ := newTestClient(t, mock)
	wallet := common.HexToAddress("0x9c90cad2e22a1E9b4a9aB3F95f7f14d08Ce78ade")

	deployed, err := client.IsWalletDeployed(context.Background(), wallet)
	if err != nil {
		t.Fatalf("IsWalletDeployed failed: %v", err)
	}
	if !deployed {
		t.Fatal("expected deployed = true")
	}
}

func TestGetTransaction(t *testing.T) {
	mock := &mockRelayer{
		responses: map[string]interface{}{
			"/transaction": []Transaction{
				{TransactionID: "tx1", State: StateConfirmed, TransactionHash: "0xabc"},
			},
		},
	}
	client, _ := newTestClient(t, mock)

	txs, err := client.GetTransaction(context.Background(), "tx1")
	if err != nil {
		t.Fatalf("GetTransaction failed: %v", err)
	}
	if len(txs) != 1 || txs[0].TransactionID != "tx1" {
		t.Fatalf("unexpected transactions: %+v", txs)
	}
}

func TestWaitForConfirmation_ImmediateConfirm(t *testing.T) {
	mock := &mockRelayer{
		responses: map[string]interface{}{
			"/transaction": []Transaction{
				{TransactionID: "tx1", State: StateConfirmed},
			},
		},
	}
	client, _ := newTestClient(t, mock)

	tx, err := client.WaitForConfirmation(context.Background(), "tx1")
	if err != nil {
		t.Fatalf("WaitForConfirmation failed: %v", err)
	}
	if tx.State != StateConfirmed {
		t.Fatalf("state = %q, want STATE_CONFIRMED", tx.State)
	}
}

func TestWaitForConfirmation_ContextCancel(t *testing.T) {
	mock := &mockRelayer{
		responses: map[string]interface{}{
			"/transaction": []Transaction{{TransactionID: "tx1", State: StateNew}},
		},
	}
	client, _ := newTestClient(t, mock)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.WaitForConfirmation(ctx, "tx1")
	if err == nil {
		t.Fatal("expected context timeout error")
	}
}

func TestWithSigner_UpdatesChainID(t *testing.T) {
	mock := &mockRelayer{responses: map[string]interface{}{}}
	srv := httptest.NewServer(mock)
	defer srv.Close()

	tc := transport.NewClient(srv.Client(), srv.URL)
	base := NewClient(tc)

	signer, _ := auth.NewPrivateKeySigner(testPrivKey, 80002) // Amoy
	updated := base.WithSigner(signer)

	impl, ok := updated.(*clientImpl)
	if !ok {
		t.Fatal("expected *clientImpl")
	}
	if impl.chainID != 80002 {
		t.Fatalf("chainID = %d, want 80002", impl.chainID)
	}
	// Factory should remain Polygon factory since Amoy is not in factoryByChain
	if impl.factory != common.HexToAddress(FactoryPolygon) {
		t.Fatalf("factory should remain Polygon factory for unknown chain")
	}
}
