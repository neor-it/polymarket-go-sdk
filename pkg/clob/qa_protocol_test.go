package clob_test

import (
	"bytes"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
	"github.com/neor-it/polymarket-go-sdk/pkg/clob"
	"github.com/neor-it/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/neor-it/polymarket-go-sdk/pkg/types"
	"github.com/shopspring/decimal"
)

const qaChain = 137
const qaBits = 256
const qaWordSize = 32
const qaTimestamp int64 = 1700000000123
const qaPrivateKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
const qaExchangeV3 = "0xe3333700cA9d93003F00f0F71f8515005F6c00Aa"
const qaDomainType = "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
const qaOrderType = "Order(uint256 salt,address maker,address signer,uint256 tokenId,uint256 makerAmount,uint256 takerAmount,uint8 side,uint8 signatureType,uint256 timestamp,bytes32 metadata,bytes32 builder)"

func qaABIWord(v *big.Int) []byte           { return v.FillBytes(make([]byte, qaWordSize)) }
func qaAddressWord(v common.Address) []byte { return qaABIWord(new(big.Int).SetBytes(v.Bytes())) }

// qaDigest independently implements the official EIP-712 field order without using SDK envelope/hash helpers.
func qaDigest(o clobtypes.Order, version string, exchange common.Address) []byte {
	domain := crypto.Keccak256([]byte(qaDomainType))
	domain = append(domain, crypto.Keccak256([]byte("Polymarket CTF Exchange"))...)
	domain = append(domain, crypto.Keccak256([]byte(version))...)
	domain = append(domain, qaABIWord(big.NewInt(qaChain))...)
	domain = append(domain, qaAddressWord(exchange)...)
	order := crypto.Keccak256([]byte(qaOrderType))
	order = append(order, qaABIWord(o.Salt.Int)...)
	order = append(order, qaAddressWord(o.Maker)...)
	order = append(order, qaAddressWord(o.Signer)...)
	order = append(order, qaABIWord(o.TokenID.Int)...)
	order = append(order, qaABIWord(o.MakerAmount.BigInt())...)
	order = append(order, qaABIWord(o.TakerAmount.BigInt())...)
	side := int64(0)
	if o.Side == "SELL" {
		side = 1
	}
	order = append(order, qaABIWord(big.NewInt(side))...)
	order = append(order, qaABIWord(big.NewInt(int64(*o.SignatureType)))...)
	order = append(order, qaABIWord(big.NewInt(o.Timestamp))...)
	order = append(order, o.Metadata[:]...)
	order = append(order, o.Builder[:]...)
	return crypto.Keccak256([]byte{0x19, 0x01}, crypto.Keccak256(domain), crypto.Keccak256(order))
}

func qaRecover(t *testing.T, digest []byte, signature string) common.Address {
	t.Helper()
	sig := common.FromHex(signature)
	if len(sig) != crypto.SignatureLength {
		t.Fatalf("invalid signature length %d", len(sig))
	}
	if sig[crypto.RecoveryIDOffset] >= 27 {
		sig[crypto.RecoveryIDOffset] -= 27
	}
	pub, err := crypto.SigToPub(digest, sig)
	if err != nil {
		t.Fatal(err)
	}
	return crypto.PubkeyToAddress(*pub)
}

func TestQAV2EOAAndSafeSignaturesRecoverOnlyOfficialExchangeDomain(t *testing.T) {
	signer, err := auth.NewPrivateKeySigner(qaPrivateKey, qaChain)
	if err != nil {
		t.Fatal(err)
	}
	for _, signatureType := range []int{int(auth.SignatureEOA), int(auth.SignatureGnosisSafe)} {
		for _, side := range []string{"BUY", "SELL"} {
			for _, negRisk := range []bool{false, true} {
				order := &clobtypes.Order{ProtocolVersion: types.ProtocolV2, Salt: types.U256{Int: big.NewInt(321)}, TokenID: types.U256{Int: new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), qaBits), big.NewInt(1))}, Maker: signer.Address(), Signer: signer.Address(), MakerAmount: decimal.NewFromInt(5000001), TakerAmount: decimal.NewFromInt(9999999), Side: side, SignatureType: &signatureType, Timestamp: qaTimestamp, NegRisk: &negRisk}
				if signatureType == int(auth.SignatureGnosisSafe) {
					order.Maker = common.HexToAddress("0x1234567890123456789012345678901234567890")
				}
				signed, err := clob.SignOrder(signer, &auth.APIKey{Key: "fixture"}, order)
				if err != nil {
					t.Fatal(err)
				}
				if got := qaRecover(t, qaDigest(signed.Order, "3", common.HexToAddress(qaExchangeV3)), signed.Signature); got != signer.Address() {
					t.Fatalf("signature %d %s negrisk=%t recovers %s", signatureType, side, negRisk, got)
				}
				if got := qaRecover(t, qaDigest(signed.Order, "2", common.HexToAddress(qaExchangeV3)), signed.Signature); got == signer.Address() {
					t.Fatal("domain2 signature accepted as V2")
				}
			}
		}
	}
}

func TestQAPoly1271DomainChangesDoNotChangeClobAuthOrDepositWalletVersion(t *testing.T) {
	signer, err := auth.NewPrivateKeySigner(qaPrivateKey, qaChain)
	if err != nil {
		t.Fatal(err)
	}
	wallet := common.HexToAddress("0x1234567890123456789012345678901234567890")
	authBefore, err := clob.BuildClobAuthTypedData(signer.Address().Hex(), qaTimestamp, 0, qaChain)
	if err != nil {
		t.Fatal(err)
	}
	signatureType := int(auth.SignaturePoly1271)
	order := &clobtypes.Order{ProtocolVersion: types.ProtocolV2, Salt: types.U256{Int: big.NewInt(1)}, TokenID: types.U256{Int: big.NewInt(123)}, Maker: wallet, Signer: wallet, MakerAmount: decimal.NewFromInt(1000000), TakerAmount: decimal.NewFromInt(2000000), Side: "SELL", SignatureType: &signatureType, Timestamp: qaTimestamp}
	envelope, err := clob.BuildPoly1271OrderEnvelope(order, qaChain)
	if err != nil {
		t.Fatal(err)
	}
	var typed apitypes.TypedData
	if err := json.Unmarshal(envelope.TypedData, &typed); err != nil {
		t.Fatal(err)
	}
	if typed.Domain.Version != "3" || common.HexToAddress(typed.Domain.VerifyingContract) != common.HexToAddress(qaExchangeV3) || typed.Message["version"] != "1" || typed.Message["verifyingContract"] != wallet.Hex() {
		t.Fatalf("domain mismatch %+v", typed)
	}
	digest, _, err := apitypes.TypedDataAndHash(typed)
	if err != nil {
		t.Fatal(err)
	}
	key, err := crypto.HexToECDSA(qaPrivateKey[2:])
	if err != nil {
		t.Fatal(err)
	}
	sig, err := crypto.Sign(digest, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := clob.VerifyTypedDataSignEnvelopeOwnerSignature(signer.Address().Hex(), envelope, common.Bytes2Hex(sig)); err == nil {
		t.Fatal("unprefixed invalid signature accepted")
	}
	if err := clob.VerifyTypedDataSignEnvelopeOwnerSignature(signer.Address().Hex(), envelope, "0x"+common.Bytes2Hex(sig)); err != nil {
		t.Fatal(err)
	}
	order.ProtocolVersion = types.ProtocolV1
	oldEnvelope, err := clob.BuildPoly1271OrderEnvelope(order, qaChain)
	if err != nil {
		t.Fatal(err)
	}
	if err := clob.VerifyTypedDataSignEnvelopeOwnerSignature(signer.Address().Hex(), oldEnvelope, "0x"+common.Bytes2Hex(sig)); err == nil {
		t.Fatal("cross-protocol replay accepted")
	}
	authAfter, err := clob.BuildClobAuthTypedData(signer.Address().Hex(), qaTimestamp, 0, qaChain)
	if err != nil || !bytes.Equal(authBefore, authAfter) {
		t.Fatal("ClobAuth changed with order protocol")
	}
}
