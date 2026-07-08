package clob

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

func TestVerifyClobAuthSignature(t *testing.T) {
	key, err := crypto.HexToECDSA("59c6995e998f97a5a0044966f094538f7a6592a9b37b7d01bd2636b2b56ff7cf")
	if err != nil {
		t.Fatalf("HexToECDSA: %v", err)
	}
	address := crypto.PubkeyToAddress(key.PublicKey).Hex()

	rawTypedData, err := BuildClobAuthTypedData(address, 1782378115, 0, 137)
	if err != nil {
		t.Fatalf("BuildClobAuthTypedData: %v", err)
	}
	var typedData apitypes.TypedData
	if err := json.Unmarshal(rawTypedData, &typedData); err != nil {
		t.Fatalf("unmarshal typed data: %v", err)
	}
	digest, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		t.Fatalf("TypedDataAndHash: %v", err)
	}

	signature, err := crypto.Sign(digest, key)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	signature[64] += 27

	if err := VerifyClobAuthSignature(address, 1782378115, 0, 137, "0x"+hex.EncodeToString(signature)); err != nil {
		t.Fatalf("VerifyClobAuthSignature: %v", err)
	}
	if err := VerifyClobAuthSignature("0x0000000000000000000000000000000000000001", 1782378115, 0, 137, "0x"+hex.EncodeToString(signature)); err == nil {
		t.Fatal("VerifyClobAuthSignature accepted a signature for the wrong address")
	}
}

func TestClobAuthDigestMatchesViem(t *testing.T) {
	const (
		address        = "0x8686dEE12B17dA1a170a8f7a53B4d51a1A4f88bc"
		timestamp      = int64(1782423056)
		nonce          = int64(0)
		chainID        = int64(137)
		expectedDigest = "0x79d0bfd0246eb05e10a91692de454b1971b5b2ab72bbda63ae4d7ccedd35926d"
	)

	rawTypedData, err := BuildClobAuthTypedData(address, timestamp, nonce, chainID)
	if err != nil {
		t.Fatalf("BuildClobAuthTypedData: %v", err)
	}
	var typedData apitypes.TypedData
	if err := json.Unmarshal(rawTypedData, &typedData); err != nil {
		t.Fatalf("unmarshal typed data: %v", err)
	}
	digest, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		t.Fatalf("TypedDataAndHash: %v", err)
	}

	if got := hexutil.Encode(digest); got != expectedDigest {
		t.Fatalf("unexpected ClobAuth digest: got %s want %s", got, expectedDigest)
	}
}

func TestDepositWalletClobAuthEnvelopeOwnerSignature(t *testing.T) {
	key, err := crypto.HexToECDSA("59c6995e998f97a5a0044966f094538f7a6592a9b37b7d01bd2636b2b56ff7cf")
	if err != nil {
		t.Fatalf("HexToECDSA: %v", err)
	}
	ownerAddress := crypto.PubkeyToAddress(key.PublicKey).Hex()
	const walletAddress = "0x3Ce6cB0d5fe5aa5c1ccE04529f11BcBF5AB5298e"

	envelope, err := BuildDepositWalletClobAuthEnvelope(walletAddress, 1782378115, 0, 137)
	if err != nil {
		t.Fatalf("BuildDepositWalletClobAuthEnvelope: %v", err)
	}
	var typedData apitypes.TypedData
	if err := json.Unmarshal(envelope.TypedData, &typedData); err != nil {
		t.Fatalf("unmarshal typed data: %v", err)
	}
	digest, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		t.Fatalf("TypedDataAndHash: %v", err)
	}

	signature, err := crypto.Sign(digest, key)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	signature[64] += 27
	signatureHex := "0x" + hex.EncodeToString(signature)

	if err := VerifyTypedDataSignEnvelopeOwnerSignature(ownerAddress, envelope, signatureHex); err != nil {
		t.Fatalf("VerifyTypedDataSignEnvelopeOwnerSignature: %v", err)
	}
	wrappedSignature, err := AssembleTypedDataSignSignature(signatureHex, envelope)
	if err != nil {
		t.Fatalf("AssembleTypedDataSignSignature: %v", err)
	}
	if len(wrappedSignature) <= len(signatureHex) {
		t.Fatalf("wrapped signature length = %d, want longer than %d", len(wrappedSignature), len(signatureHex))
	}
	if err := VerifyTypedDataSignEnvelopeOwnerSignature(walletAddress, envelope, signatureHex); err == nil {
		t.Fatal("VerifyTypedDataSignEnvelopeOwnerSignature accepted the deposit wallet as owner")
	}
}
