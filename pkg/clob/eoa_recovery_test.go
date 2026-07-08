package clob

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/shopspring/decimal"

	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
	"github.com/neor-it/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/neor-it/polymarket-go-sdk/pkg/types"
)

// eoaOrderDigest rebuilds the EIP-712 digest the EOA order path signs, by
// reconstructing the exact apitypes.TypedData that signOrderWithCreds builds for
// a signed order. Recovering against this digest proves the EOA signature is
// bound to the order's contents.
func eoaOrderDigest(t *testing.T, signer auth.Signer, o clobtypes.Order) []byte {
	t.Helper()
	negRisk := false
	if o.NegRisk != nil {
		negRisk = *o.NegRisk
	}
	sideInt := 0
	if strings.ToUpper(o.Side) == "SELL" {
		sideInt = 1
	}
	sigTypeVal := int(auth.SignatureEOA)
	if o.SignatureType != nil {
		sigTypeVal = *o.SignatureType
	}

	domain := apitypes.TypedDataDomain{
		Name:              "Polymarket CTF Exchange",
		Version:           "2",
		ChainId:           (*math.HexOrDecimal256)(signer.ChainID()),
		VerifyingContract: verifyingContractV2(negRisk),
	}
	typesDef := apitypes.Types{
		"EIP712Domain": {
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"},
			{Name: "verifyingContract", Type: "address"},
		},
		"Order": {
			{Name: "salt", Type: "uint256"},
			{Name: "maker", Type: "address"},
			{Name: "signer", Type: "address"},
			{Name: "tokenId", Type: "uint256"},
			{Name: "makerAmount", Type: "uint256"},
			{Name: "takerAmount", Type: "uint256"},
			{Name: "side", Type: "uint8"},
			{Name: "signatureType", Type: "uint8"},
			{Name: "timestamp", Type: "uint256"},
			{Name: "metadata", Type: "bytes32"},
			{Name: "builder", Type: "bytes32"},
		},
	}
	message := apitypes.TypedDataMessage{
		"salt":          (*math.HexOrDecimal256)(o.Salt.Int),
		"maker":         o.Maker.String(),
		"signer":        o.Signer.String(),
		"tokenId":       (*math.HexOrDecimal256)(o.TokenID.Int),
		"makerAmount":   (*math.HexOrDecimal256)(o.MakerAmount.BigInt()),
		"takerAmount":   (*math.HexOrDecimal256)(o.TakerAmount.BigInt()),
		"side":          (*math.HexOrDecimal256)(big.NewInt(int64(sideInt))),
		"signatureType": (*math.HexOrDecimal256)(big.NewInt(int64(sigTypeVal))),
		"timestamp":     (*math.HexOrDecimal256)(big.NewInt(o.Timestamp)),
		"metadata":      o.Metadata.Hex(),
		"builder":       o.Builder.Hex(),
	}
	td := apitypes.TypedData{Types: typesDef, PrimaryType: "Order", Domain: domain, Message: message}
	digest, _, err := apitypes.TypedDataAndHash(td)
	if err != nil {
		t.Fatalf("TypedDataAndHash: %v", err)
	}
	return digest
}

func recoverEOA(t *testing.T, digest []byte, sigHex string) common.Address {
	t.Helper()
	sig := common.FromHex(sigHex)
	if len(sig) != 65 {
		t.Fatalf("recoverEOA: sig length %d, want 65", len(sig))
	}
	normalized := make([]byte, 65)
	copy(normalized, sig)
	if normalized[64] >= 27 {
		normalized[64] -= 27
	}
	pub, err := crypto.SigToPub(digest, normalized)
	if err != nil {
		t.Fatalf("SigToPub: %v", err)
	}
	return crypto.PubkeyToAddress(*pub)
}

// TestEOAOrder_RecoversToSigner builds and signs an EOA order, then recovers the
// signature against the reconstructed order digest and asserts the recovered
// address equals the signer's address. This is real correctness verification of
// the EIP-712 signing path (not just signature shape).
func TestEOAOrder_RecoversToSigner(t *testing.T) {
	for _, side := range []string{"BUY", "SELL"} {
		t.Run(side, func(t *testing.T) {
			signer, err := auth.NewPrivateKeySigner("0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", 137)
			if err != nil {
				t.Fatalf("create signer: %v", err)
			}
			apiKey := &auth.APIKey{Key: "k1", Secret: "s1", Passphrase: "p1"}

			negRisk := false
			eoa := int(auth.SignatureEOA)
			order := &clobtypes.Order{
				Salt:          types.U256{Int: big.NewInt(987654321)},
				TokenID:       types.U256{Int: new(big.Int).SetBytes(common.FromHex("0xabcd"))},
				MakerAmount:   decimal.NewFromInt(5_000_000),
				TakerAmount:   decimal.NewFromInt(10_000_000),
				Side:          side,
				Signer:        signer.Address(),
				SignatureType: &eoa,
				Timestamp:     1700000000123,
				NegRisk:       &negRisk,
			}

			signed, err := signOrderWithCreds(signer, apiKey, order, nil, nil, nil, nil)
			if err != nil {
				t.Fatalf("signOrderWithCreds: %v", err)
			}
			if signed.Order.SignatureType == nil || *signed.Order.SignatureType != int(auth.SignatureEOA) {
				t.Fatalf("expected EOA signature type, got %+v", signed.Order.SignatureType)
			}

			digest := eoaOrderDigest(t, signer, signed.Order)
			recovered := recoverEOA(t, digest, signed.Signature)
			if recovered != signer.Address() {
				t.Fatalf("recovered %s, want signer %s", recovered.Hex(), signer.Address().Hex())
			}
		})
	}
}

// TestEOAOrder_NegRiskRecoversToSigner ensures the neg-risk verifying contract
// path also produces a recoverable signature bound to the signer.
func TestEOAOrder_NegRiskRecoversToSigner(t *testing.T) {
	signer, err := auth.NewPrivateKeySigner("0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", 137)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}
	apiKey := &auth.APIKey{Key: "k1", Secret: "s1", Passphrase: "p1"}

	negRisk := true
	order := &clobtypes.Order{
		Salt:        types.U256{Int: big.NewInt(42)},
		TokenID:     types.U256{Int: big.NewInt(7)},
		MakerAmount: decimal.NewFromInt(1_000_000),
		TakerAmount: decimal.NewFromInt(2_000_000),
		Side:        "BUY",
		Signer:      signer.Address(),
		Timestamp:   1700000000999,
		NegRisk:     &negRisk,
	}

	signed, err := signOrderWithCreds(signer, apiKey, order, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("signOrderWithCreds: %v", err)
	}

	digest := eoaOrderDigest(t, signer, signed.Order)
	if recoverEOA(t, digest, signed.Signature) != signer.Address() {
		t.Fatal("neg-risk EOA order did not recover to signer")
	}

	// Sanity: a standard-exchange digest must NOT recover to the signer, proving
	// the neg-risk verifying contract is actually part of what was signed.
	std := signed.Order
	stdNeg := false
	std.NegRisk = &stdNeg
	if recoverEOA(t, eoaOrderDigest(t, signer, std), signed.Signature) == signer.Address() {
		t.Fatal("standard-exchange digest unexpectedly matched a neg-risk signature")
	}
}
