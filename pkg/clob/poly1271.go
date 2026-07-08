package clob

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
	"github.com/neor-it/polymarket-go-sdk/pkg/clob/clobtypes"
)

const (
	depositWalletName    = "DepositWallet"
	depositWalletVersion = "1"
)

// eip712DomainTypeStr is the EIP-712 domain type string for the CTF Exchange V2 domain.
const eip712DomainTypeStr = "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"

// orderTypeStr is the EIP-712 type string for the CLOB V2 Order struct.
const orderTypeStr = "Order(uint256 salt,address maker,address signer,uint256 tokenId,uint256 makerAmount,uint256 takerAmount,uint8 side,uint8 signatureType,uint256 timestamp,bytes32 metadata,bytes32 builder)"

// typedDataSignTypeStr is the ERC-7739 TypedDataSign type with the Order type appended.
// The Order type definition is concatenated to produce the full type hash string.
const typedDataSignTypeStr = "TypedDataSign(Order contents,string name,string version,uint256 chainId,address verifyingContract,bytes32 salt)" + orderTypeStr

// digestSigner extends Signer to support raw-digest signing needed for POLY_1271.
// PrivateKeySigner implements this via its SignDigest method.
type digestSigner interface {
	auth.Signer
	SignDigest(digest []byte) ([]byte, error)
}

// signPoly1271Order signs an order using the ERC-7739 wrapped POLY_1271 scheme.
// Before calling this, order.Signer must be set to the deposit wallet address
// (same as order.Maker).  The owner EOA's key signs the wrapped digest.
func signPoly1271Order(signer auth.Signer, order *clobtypes.Order) ([]byte, error) {
	ds, ok := signer.(digestSigner)
	if !ok {
		return nil, fmt.Errorf("signer does not implement SignDigest (required for POLY_1271)")
	}

	negRisk := false
	if order.NegRisk != nil {
		negRisk = *order.NegRisk
	}

	// Exchange domain separator for the CTF Exchange V2.
	domainSep := poly1271ExchangeDomainSeparator(signer.ChainID(), verifyingContractV2(negRisk))

	// Order struct hash, used as the ERC-7739 "contents" hash.
	sideInt := 0
	if strings.ToUpper(order.Side) == "SELL" {
		sideInt = 1
	}
	sigTypeVal := 3
	if order.SignatureType != nil {
		sigTypeVal = *order.SignatureType
	}
	contentsHash := poly1271OrderStructHash(order, sideInt, sigTypeVal)

	// TypedDataSign struct hash — deposit wallet is the ERC-7739 verifying contract.
	depositWallet := common.Address(order.Signer)
	typedDataSignHash := poly1271TypedDataSignStructHash(signer.ChainID(), depositWallet, contentsHash)

	// Final EIP-191 prefix: keccak256(0x1901 || domainSep || typedDataSignHash)
	buf := make([]byte, 66)
	buf[0] = 0x19
	buf[1] = 0x01
	copy(buf[2:34], domainSep[:])
	copy(buf[34:66], typedDataSignHash[:])
	finalDigest := crypto.Keccak256(buf)

	sig, err := ds.SignDigest(finalDigest)
	if err != nil {
		return nil, fmt.Errorf("POLY_1271 digest sign failed: %w", err)
	}

	// ERC-7739 wrapping: sig || domainSep || contentsHash || contentsType || uint16BE(len(contentsType))
	return wrapPoly1271Signature(sig, domainSep[:], contentsHash[:], orderTypeStr), nil
}

// poly1271ExchangeDomainSeparator computes the EIP-712 domain separator for the
// CTF Exchange V2 contract.
func poly1271ExchangeDomainSeparator(chainID *big.Int, verifyingContract string) [32]byte {
	chainIDBuf := abiEncodeUint256(chainID)
	contractBuf := abiEncodeAddress(common.HexToAddress(verifyingContract))

	var combined []byte
	combined = append(combined, crypto.Keccak256([]byte(eip712DomainTypeStr))...)
	combined = append(combined, crypto.Keccak256([]byte("Polymarket CTF Exchange"))...)
	combined = append(combined, crypto.Keccak256([]byte("2"))...)
	combined = append(combined, chainIDBuf[:]...)
	combined = append(combined, contractBuf[:]...)

	var out [32]byte
	copy(out[:], crypto.Keccak256(combined))
	return out
}

// poly1271OrderStructHash computes the EIP-712 struct hash for a CLOB V2 order.
func poly1271OrderStructHash(order *clobtypes.Order, sideInt, sigTypeVal int) [32]byte {
	saltBuf := abiEncodeUint256(order.Salt.Int)
	makerBuf := abiEncodeAddress(common.Address(order.Maker))
	signerBuf := abiEncodeAddress(common.Address(order.Signer))
	tokenIDBuf := abiEncodeUint256(order.TokenID.Int)
	makerAmtBuf := abiEncodeUint256(order.MakerAmount.BigInt())
	takerAmtBuf := abiEncodeUint256(order.TakerAmount.BigInt())
	sideBuf := abiEncodeUint256(big.NewInt(int64(sideInt)))
	sigTypeBuf := abiEncodeUint256(big.NewInt(int64(sigTypeVal)))
	tsBuf := abiEncodeUint256(big.NewInt(order.Timestamp))

	var combined []byte
	combined = append(combined, crypto.Keccak256([]byte(orderTypeStr))...)
	combined = append(combined, saltBuf[:]...)
	combined = append(combined, makerBuf[:]...)
	combined = append(combined, signerBuf[:]...)
	combined = append(combined, tokenIDBuf[:]...)
	combined = append(combined, makerAmtBuf[:]...)
	combined = append(combined, takerAmtBuf[:]...)
	combined = append(combined, sideBuf[:]...)
	combined = append(combined, sigTypeBuf[:]...)
	combined = append(combined, tsBuf[:]...)
	combined = append(combined, order.Metadata[:]...)
	combined = append(combined, order.Builder[:]...)

	var out [32]byte
	copy(out[:], crypto.Keccak256(combined))
	return out
}

// poly1271TypedDataSignStructHash computes the ERC-7739 TypedDataSign struct hash.
// The deposit wallet is the ERC-7739 verifying contract (verifyingContract field).
func poly1271TypedDataSignStructHash(chainID *big.Int, depositWallet common.Address, contentsHash [32]byte) [32]byte {
	chainIDBuf := abiEncodeUint256(chainID)
	walletBuf := abiEncodeAddress(depositWallet)
	var walletSalt [32]byte // always zero per the POLY_1271 spec

	var combined []byte
	combined = append(combined, crypto.Keccak256([]byte(typedDataSignTypeStr))...)
	combined = append(combined, contentsHash[:]...)
	combined = append(combined, crypto.Keccak256([]byte(depositWalletName))...)
	combined = append(combined, crypto.Keccak256([]byte(depositWalletVersion))...)
	combined = append(combined, chainIDBuf[:]...)
	combined = append(combined, walletBuf[:]...)
	combined = append(combined, walletSalt[:]...)

	var out [32]byte
	copy(out[:], crypto.Keccak256(combined))
	return out
}

// wrapPoly1271Signature builds the ERC-7739 outer signature envelope:
//
//	innerSig || domainSep || contentsHash || contentsTypeStr || uint16BE(len(contentsTypeStr))
func wrapPoly1271Signature(sig, domainSep, contentsHash []byte, contentsTypeStr string) []byte {
	typeBytes := []byte(contentsTypeStr)
	typeLenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(typeLenBuf, uint16(len(typeBytes)))

	result := make([]byte, 0, len(sig)+len(domainSep)+len(contentsHash)+len(typeBytes)+2)
	result = append(result, sig...)
	result = append(result, domainSep...)
	result = append(result, contentsHash...)
	result = append(result, typeBytes...)
	result = append(result, typeLenBuf...)
	return result
}

// abiEncodeUint256 left-pads n to 32 bytes for EIP-712 ABI encoding.
func abiEncodeUint256(n *big.Int) [32]byte {
	var buf [32]byte
	if n == nil {
		return buf
	}
	b := n.Bytes()
	if len(b) > 32 {
		copy(buf[:], b[len(b)-32:])
	} else {
		copy(buf[32-len(b):], b)
	}
	return buf
}

// abiEncodeAddress pads the 20-byte address to 32 bytes for EIP-712 ABI encoding.
func abiEncodeAddress(addr common.Address) [32]byte {
	var buf [32]byte
	copy(buf[12:], addr[:])
	return buf
}
