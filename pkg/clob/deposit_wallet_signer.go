package clob

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
)

// DepositWalletSigner signs Polymarket CLOB payloads for a Deposit Wallet
// through its owner EOA key. Address returns the Deposit Wallet address so CLOB
// orders use signatureType=3 / POLY_1271 correctly. CLOB L1/L2 auth should use
// the owner EOA signer, not this order signer.
type DepositWalletSigner struct {
	owner         *auth.PrivateKeySigner
	walletAddress common.Address
	chainID       *big.Int
}

// NewDepositWalletSigner creates a Deposit Wallet signer from the owner EOA
// private key and the deployed Deposit Wallet address.
func NewDepositWalletSigner(ownerPrivateKeyHex, walletAddress string, chainID int64) (*DepositWalletSigner, error) {
	owner, err := auth.NewPrivateKeySigner(ownerPrivateKeyHex, chainID)
	if err != nil {
		return nil, fmt.Errorf("create owner signer: %w", err)
	}
	if !common.IsHexAddress(walletAddress) {
		return nil, fmt.Errorf("invalid deposit wallet address: %q", walletAddress)
	}

	return &DepositWalletSigner{
		owner:         owner,
		walletAddress: common.HexToAddress(walletAddress),
		chainID:       big.NewInt(chainID),
	}, nil
}

// Address returns the Deposit Wallet address, not the owner EOA address.
func (s *DepositWalletSigner) Address() common.Address {
	return s.walletAddress
}

// ChainID returns the configured chain ID.
func (s *DepositWalletSigner) ChainID() *big.Int {
	return new(big.Int).Set(s.chainID)
}

// OwnerAddress returns the EOA address that controls the Deposit Wallet.
func (s *DepositWalletSigner) OwnerAddress() common.Address {
	return s.owner.Address()
}

// SignDigest signs a precomputed digest with the owner EOA key.
func (s *DepositWalletSigner) SignDigest(digest []byte) ([]byte, error) {
	return s.owner.SignDigest(digest)
}

// SignTypedData signs typed data through an ERC-7739 TypedDataSign envelope and
// returns the full Deposit Wallet ERC-1271 wire signature.
func (s *DepositWalletSigner) SignTypedData(domain *apitypes.TypedDataDomain, typesDef apitypes.Types, message apitypes.TypedDataMessage, primaryType string) ([]byte, error) {
	envelope, err := BuildTypedDataSignEnvelope(s.walletAddress.Hex(), s.chainID.Int64(), domain, typesDef, message, primaryType)
	if err != nil {
		return nil, err
	}

	var typedData apitypes.TypedData
	if err := json.Unmarshal(envelope.TypedData, &typedData); err != nil {
		return nil, fmt.Errorf("parse typed data sign envelope: %w", err)
	}
	digest, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		return nil, fmt.Errorf("hash typed data sign envelope: %w", err)
	}

	signature, err := s.owner.SignDigest(digest)
	if err != nil {
		return nil, fmt.Errorf("sign typed data sign envelope: %w", err)
	}
	signatureBlob, err := AssembleTypedDataSignSignature(hexutil.Encode(signature), envelope)
	if err != nil {
		return nil, err
	}
	return hexutil.Decode(signatureBlob)
}
