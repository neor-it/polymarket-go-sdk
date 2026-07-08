package clob

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
	"github.com/neor-it/polymarket-go-sdk/pkg/clob/clobtypes"
)

const (
	typedDataSignPrimaryType = "TypedDataSign"
	signatureLengthBytes     = 65
	hashLengthBytes          = 32
	maxTypeDescriptorBytes   = 0xFFFF
)

// TypedDataSignEnvelope is the ERC-7739 signing payload for a Deposit Wallet.
// TypedData can be sent directly to an EVM wallet via eth_signTypedData_v4.
type TypedDataSignEnvelope struct {
	TypedData          json.RawMessage `json:"typed_data"`
	AppDomainSeparator string          `json:"app_domain_separator"`
	ContentsHash       string          `json:"contents_hash"`
	ContentsDescr      string          `json:"contents_descr"`
}

// BuildTypedDataSignEnvelope wraps application EIP-712 typed data in the
// ERC-7739 TypedDataSign envelope expected by Polymarket Deposit Wallets.
func BuildTypedDataSignEnvelope(walletAddress string, chainID int64, domain *apitypes.TypedDataDomain, typesDef apitypes.Types, message apitypes.TypedDataMessage, primaryType string) (*TypedDataSignEnvelope, error) {
	if !common.IsHexAddress(walletAddress) {
		return nil, fmt.Errorf("invalid deposit wallet address: %q", walletAddress)
	}
	if domain == nil {
		return nil, fmt.Errorf("domain is required")
	}

	typedData := apitypes.TypedData{
		Types:       typesDef,
		PrimaryType: primaryType,
		Domain:      *domain,
		Message:     message,
	}

	contentsHash, err := typedData.HashStruct(primaryType, message)
	if err != nil {
		return nil, fmt.Errorf("hash contents struct: %w", err)
	}
	appDomainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return nil, fmt.Errorf("hash app domain: %w", err)
	}

	contentsDescr := string(typedData.EncodeType(primaryType))
	if err := validateTypeDescriptor(contentsDescr); err != nil {
		return nil, err
	}

	envelopeTypes := apitypes.Types{}
	for name, fields := range typesDef {
		if name == "EIP712Domain" {
			continue
		}
		envelopeTypes[name] = fields
	}
	envelopeTypes["EIP712Domain"] = typesDef["EIP712Domain"]
	envelopeTypes[typedDataSignPrimaryType] = []apitypes.Type{
		{Name: "contents", Type: primaryType},
		{Name: "name", Type: "string"},
		{Name: "version", Type: "string"},
		{Name: "chainId", Type: "uint256"},
		{Name: "verifyingContract", Type: "address"},
		{Name: "salt", Type: "bytes32"},
	}

	envelope := apitypes.TypedData{
		Types:       envelopeTypes,
		PrimaryType: typedDataSignPrimaryType,
		Domain:      *domain,
		Message: apitypes.TypedDataMessage{
			"contents":          map[string]interface{}(message),
			"name":              depositWalletName,
			"version":           depositWalletVersion,
			"chainId":           (*math.HexOrDecimal256)(big.NewInt(chainID)),
			"verifyingContract": common.HexToAddress(walletAddress).Hex(),
			"salt":              "0x0000000000000000000000000000000000000000000000000000000000000000",
		},
	}
	if _, _, err := apitypes.TypedDataAndHash(envelope); err != nil {
		return nil, fmt.Errorf("hash typed data sign envelope: %w", err)
	}

	payload, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("marshal envelope typed data: %w", err)
	}

	return &TypedDataSignEnvelope{
		TypedData:          payload,
		AppDomainSeparator: hexutil.Encode(appDomainSeparator),
		ContentsHash:       hexutil.Encode(contentsHash),
		ContentsDescr:      contentsDescr,
	}, nil
}

// AssembleTypedDataSignSignature combines a 65-byte wallet signature with the
// captured ERC-7739 envelope for Deposit Wallet ERC-1271 validation.
func AssembleTypedDataSignSignature(signatureHex string, envelope *TypedDataSignEnvelope) (string, error) {
	if envelope == nil {
		return "", fmt.Errorf("envelope is required")
	}

	signature, err := decodeFixedHex(signatureHex, signatureLengthBytes, "signature")
	if err != nil {
		return "", err
	}
	if signature[64] < 27 {
		signature[64] += 27
	}

	appDomain, err := decodeFixedHex(envelope.AppDomainSeparator, hashLengthBytes, "app domain separator")
	if err != nil {
		return "", err
	}
	contentsHash, err := decodeFixedHex(envelope.ContentsHash, hashLengthBytes, "contents hash")
	if err != nil {
		return "", err
	}
	if err := validateTypeDescriptor(envelope.ContentsDescr); err != nil {
		return "", err
	}

	descriptor := []byte(envelope.ContentsDescr)
	out := make([]byte, 0, len(signature)+len(appDomain)+len(contentsHash)+len(descriptor)+2)
	out = append(out, signature...)
	out = append(out, appDomain...)
	out = append(out, contentsHash...)
	out = append(out, descriptor...)
	lenBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBytes, uint16(len(descriptor)))
	out = append(out, lenBytes...)
	return hexutil.Encode(out), nil
}

// BuildPoly1271OrderEnvelope returns the ERC-7739 envelope for a POLY_1271 CLOB order.
func BuildPoly1271OrderEnvelope(order *clobtypes.Order, chainID int64) (*TypedDataSignEnvelope, error) {
	if order == nil {
		return nil, fmt.Errorf("order is required")
	}
	depositWallet := common.Address(order.Signer)
	if depositWallet == (common.Address{}) {
		return nil, fmt.Errorf("order signer deposit wallet is required")
	}

	negRisk := false
	if order.NegRisk != nil {
		negRisk = *order.NegRisk
	}
	sideInt := 0
	if strings.ToUpper(order.Side) == "SELL" {
		sideInt = 1
	}
	sigTypeVal := int(auth.SignaturePoly1271)
	if order.SignatureType != nil {
		sigTypeVal = *order.SignatureType
	}

	domain := &apitypes.TypedDataDomain{
		Name:              "Polymarket CTF Exchange",
		Version:           "2",
		ChainId:           math.NewHexOrDecimal256(chainID),
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
		"salt":          (*math.HexOrDecimal256)(order.Salt.Int),
		"maker":         order.Maker.String(),
		"signer":        order.Signer.String(),
		"tokenId":       (*math.HexOrDecimal256)(order.TokenID.Int),
		"makerAmount":   (*math.HexOrDecimal256)(order.MakerAmount.BigInt()),
		"takerAmount":   (*math.HexOrDecimal256)(order.TakerAmount.BigInt()),
		"side":          (*math.HexOrDecimal256)(big.NewInt(int64(sideInt))),
		"signatureType": (*math.HexOrDecimal256)(big.NewInt(int64(sigTypeVal))),
		"timestamp":     (*math.HexOrDecimal256)(big.NewInt(order.Timestamp)),
		"metadata":      order.Metadata.Hex(),
		"builder":       order.Builder.Hex(),
	}
	return BuildTypedDataSignEnvelope(depositWallet.Hex(), chainID, domain, typesDef, message, "Order")
}

// BuildClobAuthTypedData returns the plain ClobAuth typed-data payload used for
// L1 authentication.
func BuildClobAuthTypedData(address string, timestamp, nonce, chainID int64) (json.RawMessage, error) {
	typedData, err := clobAuthTypedData(address, timestamp, nonce, chainID)
	if err != nil {
		return nil, err
	}
	if _, _, err := apitypes.TypedDataAndHash(*typedData); err != nil {
		return nil, fmt.Errorf("hash clob auth typed data: %w", err)
	}
	payload, err := json.Marshal(typedData)
	if err != nil {
		return nil, fmt.Errorf("marshal clob auth typed data: %w", err)
	}
	return payload, nil
}

// VerifyClobAuthSignature recovers the EIP-712 ClobAuth signer and checks that
// it matches the owner address used in L1 CLOB authentication headers.
func VerifyClobAuthSignature(address string, timestamp, nonce, chainID int64, signatureHex string) error {
	typedData, err := clobAuthTypedData(address, timestamp, nonce, chainID)
	if err != nil {
		return err
	}
	digest, _, err := apitypes.TypedDataAndHash(*typedData)
	if err != nil {
		return fmt.Errorf("hash clob auth typed data: %w", err)
	}
	signature, err := hexutil.Decode(strings.TrimSpace(signatureHex))
	if err != nil || len(signature) != signatureLengthBytes {
		return fmt.Errorf("signature must be 65 bytes of 0x hex")
	}
	recovery := make([]byte, signatureLengthBytes)
	copy(recovery, signature)
	if recovery[64] >= 27 {
		recovery[64] -= 27
	}
	if recovery[64] > 1 {
		return fmt.Errorf("invalid signature recovery id")
	}
	pub, err := ethcrypto.SigToPub(digest, recovery)
	if err != nil {
		return fmt.Errorf("recover clob auth signer: %w", err)
	}
	recovered := ethcrypto.PubkeyToAddress(*pub).Hex()
	expected := common.HexToAddress(address).Hex()
	if !strings.EqualFold(recovered, expected) {
		return fmt.Errorf("clob auth signature does not recover to owner address: recovered %s expected %s", recovered, expected)
	}
	return nil
}

// BuildDepositWalletClobAuthEnvelope returns the ERC-7739 envelope for a ClobAuth
// payload. Deposit-wallet API credentials must be bound to the deposit wallet,
// so the L1 auth signature uses the same POLY_1271 wrapper as CLOB orders.
func BuildDepositWalletClobAuthEnvelope(walletAddress string, timestamp, nonce, chainID int64) (*TypedDataSignEnvelope, error) {
	typedData, err := clobAuthTypedData(walletAddress, timestamp, nonce, chainID)
	if err != nil {
		return nil, err
	}
	return BuildTypedDataSignEnvelope(walletAddress, chainID, &typedData.Domain, typedData.Types, typedData.Message, typedData.PrimaryType)
}

// VerifyTypedDataSignEnvelopeOwnerSignature recovers the owner signer for a
// browser-signed ERC-7739 envelope before the signature is wrapped for CLOB.
func VerifyTypedDataSignEnvelopeOwnerSignature(ownerAddress string, envelope *TypedDataSignEnvelope, signatureHex string) error {
	if !common.IsHexAddress(ownerAddress) {
		return fmt.Errorf("invalid owner address: %q", ownerAddress)
	}
	if envelope == nil {
		return fmt.Errorf("envelope is required")
	}
	var typedData apitypes.TypedData
	if err := json.Unmarshal(envelope.TypedData, &typedData); err != nil {
		return fmt.Errorf("unmarshal typed data sign envelope: %w", err)
	}
	digest, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		return fmt.Errorf("hash typed data sign envelope: %w", err)
	}
	signature, err := decodeFixedHex(signatureHex, signatureLengthBytes, "signature")
	if err != nil {
		return err
	}
	recovery := make([]byte, signatureLengthBytes)
	copy(recovery, signature)
	if recovery[64] >= 27 {
		recovery[64] -= 27
	}
	if recovery[64] > 1 {
		return fmt.Errorf("invalid signature recovery id")
	}
	pub, err := ethcrypto.SigToPub(digest, recovery)
	if err != nil {
		return fmt.Errorf("recover typed data sign signer: %w", err)
	}
	recovered := ethcrypto.PubkeyToAddress(*pub).Hex()
	expected := common.HexToAddress(ownerAddress).Hex()
	if !strings.EqualFold(recovered, expected) {
		return fmt.Errorf("typed data sign signature does not recover to owner address: recovered %s expected %s", recovered, expected)
	}
	return nil
}

func clobAuthTypedData(address string, timestamp, nonce, chainID int64) (*apitypes.TypedData, error) {
	if !common.IsHexAddress(address) {
		return nil, fmt.Errorf("invalid auth address: %q", address)
	}

	return &apitypes.TypedData{
		Types:       auth.ClobAuthTypes,
		PrimaryType: "ClobAuth",
		Domain: apitypes.TypedDataDomain{
			Name:    auth.ClobAuthDomain.Name,
			Version: auth.ClobAuthDomain.Version,
			ChainId: math.NewHexOrDecimal256(chainID),
		},
		Message: apitypes.TypedDataMessage{
			"address":   common.HexToAddress(address).Hex(),
			"timestamp": fmt.Sprintf("%d", timestamp),
			"nonce":     (*math.HexOrDecimal256)(big.NewInt(nonce)),
			"message":   "This message attests that I control the given wallet",
		},
	}, nil
}

func decodeFixedHex(value string, expectedBytes int, label string) ([]byte, error) {
	decoded, err := hexutil.Decode(strings.TrimSpace(value))
	if err != nil || len(decoded) != expectedBytes {
		return nil, fmt.Errorf("%s must be %d bytes of 0x hex", label, expectedBytes)
	}
	return decoded, nil
}

func validateTypeDescriptor(contentsDescr string) error {
	if contentsDescr == "" {
		return fmt.Errorf("contents type descriptor is empty")
	}
	if len(contentsDescr) > maxTypeDescriptorBytes {
		return fmt.Errorf("contents type descriptor too long: %d bytes", len(contentsDescr))
	}
	return nil
}
