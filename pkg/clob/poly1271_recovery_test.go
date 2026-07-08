package clob

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/shopspring/decimal"

	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
	"github.com/neor-it/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/neor-it/polymarket-go-sdk/pkg/types"
)

// poly1271FinalDigest recomputes the ERC-7739 (POLY_1271) digest that
// signPoly1271Order signs, so a test can recover the signing address from the
// inner ECDSA signature carried in the wrapped envelope.
func poly1271FinalDigest(t *testing.T, signer auth.Signer, order *clobtypes.Order) []byte {
	t.Helper()
	negRisk := false
	if order.NegRisk != nil {
		negRisk = *order.NegRisk
	}
	domainSep := poly1271ExchangeDomainSeparator(signer.ChainID(), verifyingContractV2(negRisk))

	sideInt := 0
	if strings.ToUpper(order.Side) == "SELL" {
		sideInt = 1
	}
	sigTypeVal := 3
	if order.SignatureType != nil {
		sigTypeVal = *order.SignatureType
	}
	contentsHash := poly1271OrderStructHash(order, sideInt, sigTypeVal)
	depositWallet := common.Address(order.Signer)
	typedDataSignHash := poly1271TypedDataSignStructHash(signer.ChainID(), depositWallet, contentsHash)

	buf := make([]byte, 66)
	buf[0] = 0x19
	buf[1] = 0x01
	copy(buf[2:34], domainSep[:])
	copy(buf[34:66], typedDataSignHash[:])
	return crypto.Keccak256(buf)
}

func recoverAddr(t *testing.T, digest, sig []byte) common.Address {
	t.Helper()
	if len(sig) != 65 {
		t.Fatalf("recoverAddr: sig length %d, want 65", len(sig))
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

func newPoly1271Order(funder common.Address, side string) *clobtypes.Order {
	sigType := int(auth.SignaturePoly1271)
	return &clobtypes.Order{
		Salt:          types.U256{Int: big.NewInt(123)},
		Maker:         funder,
		Signer:        funder,
		TokenID:       types.U256{Int: new(big.Int).SetBytes(common.FromHex("0x1234"))},
		MakerAmount:   decimal.NewFromInt(5_000_000),
		TakerAmount:   decimal.NewFromInt(10_000_000),
		Side:          side,
		SignatureType: &sigType,
		Timestamp:     1700000000123,
	}
}

// TestPoly1271_RecoversToSigner proves the inner ECDSA signature of a POLY_1271
// wrapped order recovers to the signer's EOA address over the hand-rolled
// ERC-7739 digest — i.e. the digest being signed is the one we expect.
func TestPoly1271_RecoversToSigner(t *testing.T) {
	for _, side := range []string{"BUY", "SELL"} {
		t.Run(side, func(t *testing.T) {
			// Foundry default key #0 — deterministic ECDSA via RFC 6979.
			signer, err := auth.NewPrivateKeySigner("0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", 137)
			if err != nil {
				t.Fatalf("create signer: %v", err)
			}
			apiKey := &auth.APIKey{Key: "k1", Secret: "s1", Passphrase: "p1"}
			funder := common.HexToAddress("0x9c90cad2e22a1E9b4a9aB3F95f7f14d08Ce78ade")

			order := newPoly1271Order(funder, side)
			signed, err := signOrderWithCreds(signer, apiKey, order, nil, nil, nil, nil)
			if err != nil {
				t.Fatalf("signOrderWithCreds: %v", err)
			}

			wrapped := common.FromHex(signed.Signature)
			if len(wrapped) < 65 {
				t.Fatalf("wrapped signature too short: %d", len(wrapped))
			}
			innerSig := wrapped[:65]

			digest := poly1271FinalDigest(t, signer, &signed.Order)
			recovered := recoverAddr(t, digest, innerSig)
			if recovered != signer.Address() {
				t.Fatalf("recovered %s, want signer %s", recovered.Hex(), signer.Address().Hex())
			}

			// Verify the wrapped envelope tail: domainSep || contentsHash || type || len.
			domainSep := poly1271ExchangeDomainSeparator(signer.ChainID(), verifyingContractV2(false))
			gotDomainSep := wrapped[65:97]
			if hexutil.Encode(gotDomainSep) != hexutil.Encode(domainSep[:]) {
				t.Fatalf("wrapped domainSep mismatch:\n got = %s\n want= %s", hexutil.Encode(gotDomainSep), hexutil.Encode(domainSep[:]))
			}
		})
	}
}

func TestBuildPoly1271OrderEnvelopeMatchesWrappedSignature(t *testing.T) {
	signer, err := auth.NewPrivateKeySigner("0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", 137)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}
	apiKey := &auth.APIKey{Key: "k1", Secret: "s1", Passphrase: "p1"}
	funder := common.HexToAddress("0x9c90cad2e22a1E9b4a9aB3F95f7f14d08Ce78ade")
	order := newPoly1271Order(funder, "BUY")

	signed, err := signOrderWithCreds(signer, apiKey, order, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("signOrderWithCreds: %v", err)
	}

	envelope, err := BuildPoly1271OrderEnvelope(&signed.Order, 137)
	if err != nil {
		t.Fatalf("BuildPoly1271OrderEnvelope: %v", err)
	}
	var envelopeTD apitypes.TypedData
	if err := json.Unmarshal(envelope.TypedData, &envelopeTD); err != nil {
		t.Fatalf("unmarshal envelope typed data: %v", err)
	}
	digest, _, err := apitypes.TypedDataAndHash(envelopeTD)
	if err != nil {
		t.Fatalf("hash envelope: %v", err)
	}
	innerSig, err := signer.SignDigest(digest)
	if err != nil {
		t.Fatalf("sign envelope digest: %v", err)
	}
	wrapped, err := AssembleTypedDataSignSignature(hexutil.Encode(innerSig), envelope)
	if err != nil {
		t.Fatalf("AssembleTypedDataSignSignature: %v", err)
	}
	if wrapped != signed.Signature {
		t.Fatalf("wrapped signature mismatch:\n got = %s\n want= %s", wrapped, signed.Signature)
	}
}

// TestPoly1271_ExchangeDomainSeparator_Golden locks the CTF Exchange V2 domain
// separators (standard and neg-risk) plus the EIP-712 type strings to fixed
// golden vectors, so an accidental change to the signing domain is caught.
func TestPoly1271_ExchangeDomainSeparator_Golden(t *testing.T) {
	std := poly1271ExchangeDomainSeparator(big.NewInt(137), verifyingContractV2(false))
	const goldenStd = "0x3264e159346253e26a64e00b69032db0e7d32f94628de3e6eecb50304d7af3d2"
	if hexutil.Encode(std[:]) != goldenStd {
		t.Fatalf("standard exchange domain separator drift:\n got    = %s\n golden = %s", hexutil.Encode(std[:]), goldenStd)
	}

	nr := poly1271ExchangeDomainSeparator(big.NewInt(137), verifyingContractV2(true))
	const goldenNR = "0x9b858f53327b0bd13af8ec14cfb35234fb9eb7b0504d1a4e61f433840d30e81a"
	if hexutil.Encode(nr[:]) != goldenNR {
		t.Fatalf("neg-risk exchange domain separator drift:\n got    = %s\n golden = %s", hexutil.Encode(nr[:]), goldenNR)
	}

	const goldenOrderType = "Order(uint256 salt,address maker,address signer,uint256 tokenId,uint256 makerAmount,uint256 takerAmount,uint8 side,uint8 signatureType,uint256 timestamp,bytes32 metadata,bytes32 builder)"
	if orderTypeStr != goldenOrderType {
		t.Fatalf("Order type string drift: %q", orderTypeStr)
	}
	const goldenDomainType = "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
	if eip712DomainTypeStr != goldenDomainType {
		t.Fatalf("EIP712Domain type string drift: %q", eip712DomainTypeStr)
	}
	const goldenTypedDataSign = "TypedDataSign(Order contents,string name,string version,uint256 chainId,address verifyingContract,bytes32 salt)" + goldenOrderType
	if typedDataSignTypeStr != goldenTypedDataSign {
		t.Fatalf("TypedDataSign type string drift: %q", typedDataSignTypeStr)
	}
}

// TestPoly1271_SignerMismatchFailsRecovery is a negative control: recovering the
// POLY_1271 signature against a digest built for a DIFFERENT deposit wallet must
// NOT yield the signer address. This guards against the recovery test passing
// for the wrong reason (e.g. if the digest were independent of the order).
func TestPoly1271_SignerMismatchFailsRecovery(t *testing.T) {
	signer, err := auth.NewPrivateKeySigner("0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", 137)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}
	apiKey := &auth.APIKey{Key: "k1", Secret: "s1", Passphrase: "p1"}
	funder := common.HexToAddress("0x9c90cad2e22a1E9b4a9aB3F95f7f14d08Ce78ade")

	order := newPoly1271Order(funder, "BUY")
	signed, err := signOrderWithCreds(signer, apiKey, order, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("signOrderWithCreds: %v", err)
	}
	innerSig := common.FromHex(signed.Signature)[:65]

	// Tamper the deposit wallet used to rebuild the digest.
	tampered := signed.Order
	tampered.Signer = common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	wrongDigest := poly1271FinalDigest(t, signer, &tampered)

	if recoverAddr(t, wrongDigest, innerSig) == signer.Address() {
		t.Fatal("recovery against a tampered digest unexpectedly matched the signer")
	}
}
