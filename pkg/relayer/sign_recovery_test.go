package relayer

import (
	"crypto/ecdsa"
	"encoding/asn1"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
)

// batchDigest recomputes the EIP-712 digest that signWalletBatch signs, so a
// test can recover the signing address from the produced signature.
func batchDigest(t *testing.T, chainID, nonce *big.Int, wallet common.Address, deadline int64, calls []DepositWalletCall) []byte {
	t.Helper()
	domainSep := walletBatchDomainSeparator(chainID, wallet)
	batchHash, err := walletBatchStructHash(wallet, nonce, deadline, calls)
	if err != nil {
		t.Fatalf("walletBatchStructHash: %v", err)
	}
	buf := make([]byte, 66)
	buf[0] = 0x19
	buf[1] = 0x01
	copy(buf[2:34], domainSep[:])
	copy(buf[34:66], batchHash[:])
	return crypto.Keccak256(buf)
}

func recoverSigner(t *testing.T, digest, sig []byte) common.Address {
	t.Helper()
	if len(sig) != 65 {
		t.Fatalf("recoverSigner: sig length %d, want 65", len(sig))
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

// TestSignWalletBatch_RecoversToSigner proves the produced signature actually
// recovers to the signer's address — i.e. the hand-rolled EIP-712 digest is the
// one being signed, not merely that the signature is the right shape.
func TestSignWalletBatch_RecoversToSigner(t *testing.T) {
	signer := mustTestSigner(t)
	wallet := common.HexToAddress("0x9c90cad2e22a1E9b4a9aB3F95f7f14d08Ce78ade")
	calls := []DepositWalletCall{
		{
			Target: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
			Value:  "0",
			Data:   "0x095ea7b3000000000000000000000000E111180000d2663C0091e4f400237545B87B996B0000000000000000000000000000000000000000033b2e3c9fd0803ce8000000",
		},
		{Target: "0x4bFb41d5B3570DeFd03C39a9A4D8dE6Bd8B8982E", Value: "0", Data: "0x"},
	}
	nonce := big.NewInt(7)
	deadline := int64(1760000000)

	sig, err := signWalletBatch(signer, wallet, nonce, deadline, calls)
	if err != nil {
		t.Fatalf("signWalletBatch: %v", err)
	}

	digest := batchDigest(t, signer.ChainID(), nonce, wallet, deadline, calls)
	recovered := recoverSigner(t, digest, sig)
	if recovered != signer.Address() {
		t.Fatalf("recovered %s, want signer %s", recovered.Hex(), signer.Address().Hex())
	}
}

// TestSignWalletBatch_DigestMatchesApitypes locks the hand-rolled EIP-712
// encoding to go-ethereum's reference TypedData implementation for the same
// Batch struct, asserting byte-equality of the final 0x1901 digest.
func TestSignWalletBatch_DigestMatchesApitypes(t *testing.T) {
	wallet := common.HexToAddress("0x9c90cad2e22a1E9b4a9aB3F95f7f14d08Ce78ade")
	chainID := big.NewInt(137)
	nonce := big.NewInt(3)
	deadline := int64(1760000000)
	calls := []DepositWalletCall{
		{Target: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", Value: "0", Data: "0x095ea7b3"},
		{Target: "0x4bFb41d5B3570DeFd03C39a9A4D8dE6Bd8B8982E", Value: "1000000", Data: "0xdeadbeef"},
	}

	handDigest := batchDigest(t, chainID, nonce, wallet, deadline, calls)

	apiCalls := make([]interface{}, len(calls))
	for i, c := range calls {
		dataBytes, err := hexutil.Decode(c.Data)
		if err != nil {
			t.Fatalf("decode call data: %v", err)
		}
		value := new(big.Int)
		value.SetString(c.Value, 10)
		apiCalls[i] = map[string]interface{}{
			"target": common.HexToAddress(c.Target).Hex(),
			"value":  (*math.HexOrDecimal256)(value),
			"data":   dataBytes,
		}
	}

	td := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"Batch": {
				{Name: "wallet", Type: "address"},
				{Name: "nonce", Type: "uint256"},
				{Name: "deadline", Type: "uint256"},
				{Name: "calls", Type: "Call[]"},
			},
			"Call": {
				{Name: "target", Type: "address"},
				{Name: "value", Type: "uint256"},
				{Name: "data", Type: "bytes"},
			},
		},
		PrimaryType: "Batch",
		Domain: apitypes.TypedDataDomain{
			Name:              "DepositWallet",
			Version:           "1",
			ChainId:           (*math.HexOrDecimal256)(chainID),
			VerifyingContract: wallet.Hex(),
		},
		Message: apitypes.TypedDataMessage{
			"wallet":   wallet.Hex(),
			"nonce":    (*math.HexOrDecimal256)(nonce),
			"deadline": (*math.HexOrDecimal256)(big.NewInt(deadline)),
			"calls":    apiCalls,
		},
	}

	apiDigest, _, err := apitypes.TypedDataAndHash(td)
	if err != nil {
		t.Fatalf("apitypes.TypedDataAndHash: %v", err)
	}
	if !equalBytes(handDigest, apiDigest) {
		t.Fatalf("digest mismatch:\n hand = %x\n api  = %x", handDigest, apiDigest)
	}
}

// TestWalletBatchDomainSeparator_Golden locks the DepositWallet domain separator
// and the EIP-712 type strings to fixed golden vectors. A change here means the
// signing domain changed and signatures would be rejected on-chain.
func TestWalletBatchDomainSeparator_Golden(t *testing.T) {
	wallet := common.HexToAddress("0x9c90cad2e22a1E9b4a9aB3F95f7f14d08Ce78ade")
	got := walletBatchDomainSeparator(big.NewInt(137), wallet)
	const goldenSep = "0x0b1d8e43e13f45e52a70129f2e6e1646965628a0a78f7994c89dd591e7b36038"
	if hexutil.Encode(got[:]) != goldenSep {
		t.Fatalf("domain separator drift:\n got    = %s\n golden = %s", hexutil.Encode(got[:]), goldenSep)
	}

	if walletBatchTypeStr != "Batch(address wallet,uint256 nonce,uint256 deadline,Call[] calls)Call(address target,uint256 value,bytes data)" {
		t.Fatalf("Batch type string drift: %q", walletBatchTypeStr)
	}
	if walletCallTypeStr != "Call(address target,uint256 value,bytes data)" {
		t.Fatalf("Call type string drift: %q", walletCallTypeStr)
	}
}

// TestSignWalletBatch_KMSStyleAndPrivateKeyAgree cross-checks that a
// KMS-style digest signer (one that emits an ASN.1 DER (r,s) pair and relies on
// low-S canonicalization + V-recovery, exactly like pkg/auth/kms.AWSSigner) and
// a PrivateKeySigner produce batch signatures that recover to the same address
// through the relayer's hand-rolled path. This guards the digestSigner
// interface dispatch for non-PrivateKey signers end-to-end.
func TestSignWalletBatch_KMSStyleAndPrivateKeyAgree(t *testing.T) {
	pk := mustTestSigner(t)
	wallet := common.HexToAddress("0x9c90cad2e22a1E9b4a9aB3F95f7f14d08Ce78ade")
	nonce := big.NewInt(0)
	deadline := int64(1760000000)
	calls := []DepositWalletCall{{Target: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", Value: "0", Data: "0x"}}

	kms := newKMSLikeSigner(t)
	if kms.Address() != pk.Address() {
		t.Fatalf("test setup: KMS addr %s != PK addr %s", kms.Address().Hex(), pk.Address().Hex())
	}

	pkSig, err := signWalletBatch(pk, wallet, nonce, deadline, calls)
	if err != nil {
		t.Fatalf("pk signWalletBatch: %v", err)
	}
	kmsSig, err := signWalletBatch(kms, wallet, nonce, deadline, calls)
	if err != nil {
		t.Fatalf("kms signWalletBatch: %v", err)
	}

	digest := batchDigest(t, kms.ChainID(), nonce, wallet, deadline, calls)
	if recoverSigner(t, digest, pkSig) != recoverSigner(t, digest, kmsSig) {
		t.Fatal("KMS-style and PK batch signatures recover to different addresses")
	}
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// kmsLikeSigner mimics pkg/auth/kms.AWSSigner: it signs a raw digest, emits an
// ASN.1 DER (r,s) pair, canonicalizes S to the lower half-order, and recovers V.
// It exists in-test (rather than importing the kms package, whose AWSSigner
// fields are unexported) so the relayer's digestSigner dispatch is exercised by
// a second, independent signer implementation.
type kmsLikeSigner struct {
	key     *ecdsa.PrivateKey
	address common.Address
	chainID *big.Int
}

func newKMSLikeSigner(t *testing.T) *kmsLikeSigner {
	t.Helper()
	// Foundry default account #0 — same key as mustTestSigner, so addresses match.
	key, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}
	return &kmsLikeSigner{key: key, address: crypto.PubkeyToAddress(key.PublicKey), chainID: big.NewInt(137)}
}

func (s *kmsLikeSigner) Address() common.Address { return s.address }
func (s *kmsLikeSigner) ChainID() *big.Int       { return s.chainID }

func (s *kmsLikeSigner) SignTypedData(domain *apitypes.TypedDataDomain, types apitypes.Types, message apitypes.TypedDataMessage, primaryType string) ([]byte, error) {
	td := apitypes.TypedData{Types: types, PrimaryType: primaryType, Domain: *domain, Message: message}
	digest, _, err := apitypes.TypedDataAndHash(td)
	if err != nil {
		return nil, err
	}
	return s.SignDigest(digest)
}

func (s *kmsLikeSigner) SignDigest(digest []byte) ([]byte, error) {
	// Produce a raw signature, then re-encode as ASN.1 DER (like KMS) and run it
	// back through the same canonicalize/recover steps AWSSigner uses.
	raw, err := crypto.Sign(digest, s.key)
	if err != nil {
		return nil, err
	}
	r := new(big.Int).SetBytes(raw[:32])
	sv := new(big.Int).SetBytes(raw[32:64])
	der, err := asn1.Marshal(struct{ R, S *big.Int }{r, sv})
	if err != nil {
		return nil, err
	}

	var parsed struct{ R, S *big.Int }
	if _, err := asn1.Unmarshal(der, &parsed); err != nil {
		return nil, err
	}
	order := crypto.S256().Params().N
	half := new(big.Int).Rsh(order, 1)
	if parsed.S.Cmp(half) > 0 {
		parsed.S = new(big.Int).Sub(order, parsed.S)
	}
	sig := make([]byte, 65)
	rb := parsed.R.Bytes()
	sb := parsed.S.Bytes()
	copy(sig[32-len(rb):32], rb)
	copy(sig[64-len(sb):64], sb)
	for _, v := range []byte{0, 1} {
		sig[64] = v
		pub, err := crypto.Ecrecover(digest, sig)
		if err != nil {
			continue
		}
		rec, err := crypto.UnmarshalPubkey(pub)
		if err == nil && crypto.PubkeyToAddress(*rec) == s.address {
			sig[64] = v + 27
			return sig, nil
		}
	}
	return nil, errors.New("failed to recover V")
}

// ensure auth import is used even if helpers above are trimmed.
var _ = auth.SignatureEOA
