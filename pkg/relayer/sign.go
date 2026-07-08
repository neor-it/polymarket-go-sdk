package relayer

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
)

const (
	walletBatchDomainTypeStr = "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
	walletCallTypeStr        = "Call(address target,uint256 value,bytes data)"
	// The Call type appended after Batch is required by EIP-712 for referenced types.
	walletBatchTypeStr = "Batch(address wallet,uint256 nonce,uint256 deadline,Call[] calls)Call(address target,uint256 value,bytes data)"
)

// walletDigestSigner extends auth.Signer with raw 32-byte digest signing.
// PrivateKeySigner satisfies this; KMS implementations would need updating.
type walletDigestSigner interface {
	auth.Signer
	SignDigest(digest []byte) ([]byte, error)
}

// signWalletBatch computes the EIP-712 Batch digest over the DepositWallet domain
// and signs it, returning the 65-byte ECDSA signature.
func signWalletBatch(signer auth.Signer, wallet common.Address, nonce *big.Int, deadline int64, calls []DepositWalletCall) ([]byte, error) {
	ds, ok := signer.(walletDigestSigner)
	if !ok {
		return nil, fmt.Errorf("signer does not implement SignDigest (required for wallet batch signing)")
	}

	domainSep := walletBatchDomainSeparator(signer.ChainID(), wallet)

	batchHash, err := walletBatchStructHash(wallet, nonce, deadline, calls)
	if err != nil {
		return nil, err
	}

	// EIP-191: 0x1901 || domainSep || structHash
	buf := make([]byte, 66)
	buf[0] = 0x19
	buf[1] = 0x01
	copy(buf[2:34], domainSep[:])
	copy(buf[34:66], batchHash[:])
	digest := crypto.Keccak256(buf)

	return ds.SignDigest(digest)
}

func walletBatchDomainSeparator(chainID *big.Int, wallet common.Address) [32]byte {
	chainIDBuf := walletABIUint256(chainID)
	walletBuf := walletABIAddress(wallet)

	var combined []byte
	combined = append(combined, crypto.Keccak256([]byte(walletBatchDomainTypeStr))...)
	combined = append(combined, crypto.Keccak256([]byte("DepositWallet"))...)
	combined = append(combined, crypto.Keccak256([]byte("1"))...)
	combined = append(combined, chainIDBuf[:]...)
	combined = append(combined, walletBuf[:]...)

	var out [32]byte
	copy(out[:], crypto.Keccak256(combined))
	return out
}

func walletCallStructHash(call DepositWalletCall) ([32]byte, error) {
	targetBuf := walletABIAddress(common.HexToAddress(call.Target))

	rawValue := strings.TrimSpace(call.Value)
	if rawValue == "" {
		rawValue = "0"
	}
	valueInt := new(big.Int)
	if _, ok := valueInt.SetString(rawValue, 10); !ok {
		return [32]byte{}, fmt.Errorf("invalid call value %q", call.Value)
	}
	valueBuf := walletABIUint256(valueInt)

	rawData := call.Data
	if rawData == "" {
		rawData = "0x"
	}
	dataBytes, err := hexutil.Decode(rawData)
	if err != nil {
		return [32]byte{}, fmt.Errorf("decode call data: %w", err)
	}
	dataHash := crypto.Keccak256(dataBytes)

	var combined []byte
	combined = append(combined, crypto.Keccak256([]byte(walletCallTypeStr))...)
	combined = append(combined, targetBuf[:]...)
	combined = append(combined, valueBuf[:]...)
	combined = append(combined, dataHash...)

	var out [32]byte
	copy(out[:], crypto.Keccak256(combined))
	return out, nil
}

func walletBatchStructHash(wallet common.Address, nonce *big.Int, deadline int64, calls []DepositWalletCall) ([32]byte, error) {
	walletBuf := walletABIAddress(wallet)
	nonceBuf := walletABIUint256(nonce)
	deadlineBuf := walletABIUint256(big.NewInt(deadline))

	// Call[] array encoded as keccak256(hash(call1) || hash(call2) || ...)
	var callsEncoded []byte
	for _, c := range calls {
		h, err := walletCallStructHash(c)
		if err != nil {
			return [32]byte{}, err
		}
		callsEncoded = append(callsEncoded, h[:]...)
	}
	callsHash := crypto.Keccak256(callsEncoded)

	var combined []byte
	combined = append(combined, crypto.Keccak256([]byte(walletBatchTypeStr))...)
	combined = append(combined, walletBuf[:]...)
	combined = append(combined, nonceBuf[:]...)
	combined = append(combined, deadlineBuf[:]...)
	combined = append(combined, callsHash...)

	var out [32]byte
	copy(out[:], crypto.Keccak256(combined))
	return out, nil
}

// walletABIUint256 left-pads n to 32 bytes for EIP-712 ABI encoding.
func walletABIUint256(n *big.Int) [32]byte {
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

// walletABIAddress pads the 20-byte address to 32 bytes for EIP-712 ABI encoding.
func walletABIAddress(addr common.Address) [32]byte {
	var buf [32]byte
	copy(buf[12:], addr[:])
	return buf
}
