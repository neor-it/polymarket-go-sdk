package relayer

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
)

// Foundry default account #0 private key — deterministic ECDSA via RFC 6979.
const testPrivKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

func mustTestSigner(t *testing.T) auth.Signer {
	t.Helper()
	s, err := auth.NewPrivateKeySigner(testPrivKey, 137)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}
	return s
}

func TestSignWalletBatch_SignatureShape(t *testing.T) {
	signer := mustTestSigner(t)
	wallet := common.HexToAddress("0x9c90cad2e22a1E9b4a9aB3F95f7f14d08Ce78ade")
	calls := []DepositWalletCall{
		{
			Target: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
			Value:  "0",
			Data:   "0x095ea7b3000000000000000000000000E111180000d2663C0091e4f400237545B87B996B0000000000000000000000000000000000000000033b2e3c9fd0803ce8000000",
		},
	}

	sig, err := signWalletBatch(signer, wallet, big.NewInt(0), 1760000000, calls)
	if err != nil {
		t.Fatalf("signWalletBatch failed: %v", err)
	}
	if len(sig) != 65 {
		t.Fatalf("signature length = %d, want 65", len(sig))
	}
	v := sig[64]
	if v != 27 && v != 28 {
		t.Fatalf("signature V = %d, want 27 or 28", v)
	}
}

func TestSignWalletBatch_Deterministic(t *testing.T) {
	signer := mustTestSigner(t)
	wallet := common.HexToAddress("0x9c90cad2e22a1E9b4a9aB3F95f7f14d08Ce78ade")

	sig1, err := signWalletBatch(signer, wallet, big.NewInt(5), 1760000000, []DepositWalletCall{})
	if err != nil {
		t.Fatalf("sign1: %v", err)
	}
	sig2, err := signWalletBatch(signer, wallet, big.NewInt(5), 1760000000, []DepositWalletCall{})
	if err != nil {
		t.Fatalf("sign2: %v", err)
	}
	for i := range sig1 {
		if sig1[i] != sig2[i] {
			t.Fatalf("signatures not deterministic at byte %d", i)
		}
	}
}

func TestSignWalletBatch_NonceChangesSignature(t *testing.T) {
	signer := mustTestSigner(t)
	wallet := common.HexToAddress("0x9c90cad2e22a1E9b4a9aB3F95f7f14d08Ce78ade")

	sig0, _ := signWalletBatch(signer, wallet, big.NewInt(0), 1760000000, nil)
	sig1, _ := signWalletBatch(signer, wallet, big.NewInt(1), 1760000000, nil)

	same := true
	for i := range sig0 {
		if sig0[i] != sig1[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("different nonces must produce different signatures")
	}
}

func TestSignWalletBatch_RequiresSignDigest(t *testing.T) {
	// A Signer that does NOT implement SignDigest.
	type minimalSigner struct{ auth.Signer }

	signer := mustTestSigner(t)
	noDigest := minimalSigner{signer}

	_, err := signWalletBatch(noDigest, common.Address{}, big.NewInt(0), 1, nil)
	if err == nil {
		t.Fatal("expected error when signer lacks SignDigest")
	}
}

func TestERC20ApproveCall_Encoding(t *testing.T) {
	token := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	spender := common.HexToAddress("0x4bFb41d5B3570DeFd03C39a9A4D8dE6Bd8B8982E")

	call := ERC20ApproveCall(token, spender, MaxUint256)

	if call.Target != token.Hex() {
		t.Fatalf("target mismatch: got %s", call.Target)
	}
	if call.Value != "0" {
		t.Fatalf("value mismatch: got %s", call.Value)
	}
	// calldata must be 68 bytes: 4 selector + 32 address + 32 amount = 68 bytes → 138 hex chars with "0x"
	if len(call.Data) != 138 {
		t.Fatalf("calldata length = %d, want 138 (4+32+32 bytes as 0x hex)", len(call.Data))
	}
	// First 4 bytes must be approve selector 0x095ea7b3
	if call.Data[:10] != "0x095ea7b3" {
		t.Fatalf("wrong selector: %s", call.Data[:10])
	}
}
