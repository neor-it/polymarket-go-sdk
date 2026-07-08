package kms

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"math/big"
	"testing"

	awskms "github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
)

// Foundry default account #0 private key — deterministic ECDSA via RFC 6979.
const testKMSPrivKey = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

// Compile-time assertions that *AWSSigner satisfies the digest-signing
// interfaces required by the POLY_1271 and relayer batch code paths. These are
// structurally identical to clob.digestSigner and relayer.walletDigestSigner.
var (
	_ auth.Signer = (*AWSSigner)(nil)
	_ interface {
		auth.Signer
		SignDigest(digest []byte) ([]byte, error)
	} = (*AWSSigner)(nil)
)

// fakeKMSClient implements KMSClient backed by an in-process ECDSA key.
// Sign() returns an ASN.1 DER (r, s) pair exactly like AWS KMS, so the
// production ASN.1-unwrap → low-S → V-recovery path in signHash is exercised
// end-to-end. The secp256k1 key is signed via go-ethereum's crypto.Sign.
type fakeKMSClient struct {
	signKey *ecdsa.PrivateKey // secp256k1 key used to produce signatures
	// pubKeyDER is the SubjectPublicKeyInfo returned by GetPublicKey.
	pubKeyDER []byte
	// forceSignErr, when set, makes Sign return an error.
	forceSignErr error
	// highS, when true, returns a non-canonical (high) S to exercise the
	// low-S normalization branch.
	highS bool

	signCalls int
}

func (f *fakeKMSClient) Sign(ctx context.Context, params *awskms.SignInput, optFns ...func(*awskms.Options)) (*awskms.SignOutput, error) {
	f.signCalls++
	if f.forceSignErr != nil {
		return nil, f.forceSignErr
	}
	// crypto.Sign yields a canonical low-S [R||S||V] recoverable signature.
	raw, err := crypto.Sign(params.Message, f.signKey)
	if err != nil {
		return nil, err
	}
	r := new(big.Int).SetBytes(raw[:32])
	s := new(big.Int).SetBytes(raw[32:64])
	if f.highS {
		// Flip S to its high-half counterpart: KMS may return either parity,
		// and signHash must canonicalize it back down.
		s = new(big.Int).Sub(crypto.S256().Params().N, s)
	}
	der, err := asn1.Marshal(struct{ R, S *big.Int }{r, s})
	if err != nil {
		return nil, err
	}
	return &awskms.SignOutput{Signature: der}, nil
}

func (f *fakeKMSClient) GetPublicKey(ctx context.Context, params *awskms.GetPublicKeyInput, optFns ...func(*awskms.Options)) (*awskms.GetPublicKeyOutput, error) {
	return &awskms.GetPublicKeyOutput{PublicKey: f.pubKeyDER}, nil
}

// newSecp256k1Signer builds an AWSSigner wired to a fake KMS client backed by
// the well-known secp256k1 test key. The struct is constructed directly (rather
// than via NewAWSSigner) because Go's crypto/x509 cannot parse a secp256k1
// SubjectPublicKeyInfo; constructing directly still exercises the full signHash
// signing/recovery path, which is the behavior under test.
func newSecp256k1Signer(t *testing.T, highS bool) (*AWSSigner, common.Address) {
	t.Helper()
	key, err := crypto.HexToECDSA(testKMSPrivKey)
	if err != nil {
		t.Fatalf("parse test key: %v", err)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)
	signer := &AWSSigner{
		client:  &fakeKMSClient{signKey: key, highS: highS},
		keyID:   "test-key-id",
		chainID: big.NewInt(137),
		pubKey:  &key.PublicKey,
		address: addr,
		timeout: defaultKMSTimeout,
	}
	return signer, addr
}

func TestAWSSigner_AddressAndChainID(t *testing.T) {
	signer, addr := newSecp256k1Signer(t, false)
	if signer.Address() != addr {
		t.Fatalf("Address() = %s, want %s", signer.Address().Hex(), addr.Hex())
	}
	if signer.ChainID().Int64() != 137 {
		t.Fatalf("ChainID() = %d, want 137", signer.ChainID().Int64())
	}
}

func TestAWSSigner_SignDigest_RecoversToSigner(t *testing.T) {
	for _, highS := range []bool{false, true} {
		name := "lowS"
		if highS {
			name = "highS"
		}
		t.Run(name, func(t *testing.T) {
			signer, addr := newSecp256k1Signer(t, highS)
			digest := crypto.Keccak256([]byte("polymarket deposit wallet order"))

			sig, err := signer.SignDigest(digest)
			if err != nil {
				t.Fatalf("SignDigest: %v", err)
			}
			if len(sig) != 65 {
				t.Fatalf("signature length = %d, want 65", len(sig))
			}
			if v := sig[64]; v != 27 && v != 28 {
				t.Fatalf("signature V = %d, want 27 or 28", v)
			}

			recovered := recoverAddress(t, digest, sig)
			if recovered != addr {
				t.Fatalf("recovered %s, want signer %s", recovered.Hex(), addr.Hex())
			}

			// Low-S invariant: S must be in the lower half of the curve order.
			s := new(big.Int).SetBytes(sig[32:64])
			halfOrder := new(big.Int).Rsh(crypto.S256().Params().N, 1)
			if s.Cmp(halfOrder) > 0 {
				t.Fatalf("S is not canonical (high-S): %s", s.String())
			}
		})
	}
}

func TestAWSSigner_SignDigest_RejectsBadLength(t *testing.T) {
	signer, _ := newSecp256k1Signer(t, false)
	if _, err := signer.SignDigest(make([]byte, 31)); err == nil {
		t.Fatal("expected error for 31-byte digest")
	}
	if _, err := signer.SignDigest(make([]byte, 33)); err == nil {
		t.Fatal("expected error for 33-byte digest")
	}
}

func TestAWSSigner_SignTypedData_RecoversToSigner(t *testing.T) {
	signer, addr := newSecp256k1Signer(t, false)

	domain := &apitypes.TypedDataDomain{
		Name:    "ClobAuthDomain",
		Version: "1",
		ChainId: (*math.HexOrDecimal256)(big.NewInt(137)),
	}
	message := apitypes.TypedDataMessage{
		"address":   addr.Hex(),
		"timestamp": "1700000000",
		"nonce":     (*math.HexOrDecimal256)(big.NewInt(0)),
		"message":   "This message attests that I control the given wallet",
	}

	sig, err := signer.SignTypedData(domain, auth.ClobAuthTypes, message, "ClobAuth")
	if err != nil {
		t.Fatalf("SignTypedData: %v", err)
	}
	if len(sig) != 65 {
		t.Fatalf("signature length = %d, want 65", len(sig))
	}

	// Recompute the EIP-712 digest the same way the signer did and recover.
	typedData := apitypes.TypedData{
		Types:       auth.ClobAuthTypes,
		PrimaryType: "ClobAuth",
		Domain:      *domain,
		Message:     message,
	}
	digest, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		t.Fatalf("TypedDataAndHash: %v", err)
	}
	if recovered := recoverAddress(t, digest, sig); recovered != addr {
		t.Fatalf("recovered %s, want signer %s", recovered.Hex(), addr.Hex())
	}
}

func TestAWSSigner_SignDigest_PropagatesKMSError(t *testing.T) {
	signer, _ := newSecp256k1Signer(t, false)
	signer.client.(*fakeKMSClient).forceSignErr = errors.New("kms unavailable")
	if _, err := signer.SignDigest(crypto.Keccak256([]byte("x"))); err == nil {
		t.Fatal("expected KMS error to propagate")
	}
}

// TestAWSSigner_PrivateKeyCrossCheck verifies that, for a fixed digest, the
// mocked-KMS AWSSigner.SignDigest and the canonical PrivateKeySigner.SignDigest
// recover to the SAME address. This locks the KMS ASN.1/low-S/V-recovery path
// to the reference implementation.
func TestAWSSigner_PrivateKeyCrossCheck(t *testing.T) {
	kmsSigner, kmsAddr := newSecp256k1Signer(t, true) // high-S forces canonicalization
	pkSigner, err := auth.NewPrivateKeySigner(testKMSPrivKey, 137)
	if err != nil {
		t.Fatalf("NewPrivateKeySigner: %v", err)
	}
	if pkSigner.Address() != kmsAddr {
		t.Fatalf("test setup: PK address %s != KMS address %s", pkSigner.Address().Hex(), kmsAddr.Hex())
	}

	digest := crypto.Keccak256([]byte("fixed digest for cross-check"))

	kmsSig, err := kmsSigner.SignDigest(digest)
	if err != nil {
		t.Fatalf("kms SignDigest: %v", err)
	}
	pkSig, err := pkSigner.SignDigest(digest)
	if err != nil {
		t.Fatalf("pk SignDigest: %v", err)
	}

	kmsRecovered := recoverAddress(t, digest, kmsSig)
	pkRecovered := recoverAddress(t, digest, pkSig)
	if kmsRecovered != pkRecovered {
		t.Fatalf("recovered addresses differ: kms=%s pk=%s", kmsRecovered.Hex(), pkRecovered.Hex())
	}
	if kmsRecovered != kmsAddr {
		t.Fatalf("kms recovered %s, want %s", kmsRecovered.Hex(), kmsAddr.Hex())
	}

	// Both signatures recover identically; for this RFC-6979 key over the same
	// digest the ECDSA (r, s) pair is identical too, so the 64-byte bodies match.
	for i := 0; i < 64; i++ {
		if kmsSig[i] != pkSig[i] {
			t.Fatalf("signature body differs at byte %d (kms=%02x pk=%02x)", i, kmsSig[i], pkSig[i])
		}
	}
}

// TestNewAWSSigner_DerivesAddressFromPublicKey covers the NewAWSSigner
// constructor: it fetches the SubjectPublicKeyInfo via GetPublicKey, parses it,
// and derives the Ethereum address. Go's crypto/x509 only parses NIST curves,
// so a P-256 SPKI is used here purely to exercise the parse + address-derivation
// branch (the signing path is covered separately with the real secp256k1 key).
func TestNewAWSSigner_DerivesAddressFromPublicKey(t *testing.T) {
	p256Key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate P-256 key: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&p256Key.PublicKey)
	if err != nil {
		t.Fatalf("marshal PKIX: %v", err)
	}
	fake := &fakeKMSClient{pubKeyDER: der}

	signer, err := NewAWSSigner(context.Background(), fake, "key-id", 137)
	if err != nil {
		t.Fatalf("NewAWSSigner: %v", err)
	}
	want := crypto.PubkeyToAddress(p256Key.PublicKey)
	if signer.Address() != want {
		t.Fatalf("Address() = %s, want %s", signer.Address().Hex(), want.Hex())
	}
	if signer.chainID.Int64() != 137 {
		t.Fatalf("chainID = %d, want 137", signer.chainID.Int64())
	}
}

func TestNewAWSSigner_RejectsNonECDSAKey(t *testing.T) {
	// A malformed/empty SPKI must fail parsing rather than panic.
	fake := &fakeKMSClient{pubKeyDER: []byte{0x30, 0x00}}
	if _, err := NewAWSSigner(context.Background(), fake, "key-id", 137); err == nil {
		t.Fatal("expected error for invalid public key DER")
	}
}

func recoverAddress(t *testing.T, digest, sig []byte) common.Address {
	t.Helper()
	if len(sig) != 65 {
		t.Fatalf("recoverAddress: sig length %d, want 65", len(sig))
	}
	// crypto.SigToPub expects V in {0,1}; the produced signature uses 27/28.
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
