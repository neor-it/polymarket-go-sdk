package combos

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
	"github.com/neor-it/polymarket-go-sdk/pkg/clob"
)

const (
	zeroBytes32 = "0x0000000000000000000000000000000000000000000000000000000000000000"
)

// ComboOrderParams contains the Exchange v3 order fields needed for Combo RFQ accept.
type ComboOrderParams struct {
	Salt              string
	Maker             string
	Signer            string
	TokenID           string
	MakerAmount       string
	TakerAmount       string
	Side              Direction
	SignatureType     int
	Timestamp         int64
	Metadata          string
	Builder           string
	ChainID           int64
	VerifyingContract string
}

// BuildComboOrder constructs the unsigned Exchange v3 order for Combo RFQ accept.
func BuildComboOrder(params ComboOrderParams) (*ComboOrder, error) {
	if strings.TrimSpace(params.Salt) == "" {
		return nil, fmt.Errorf("salt is required")
	}
	if !common.IsHexAddress(params.Maker) {
		return nil, fmt.Errorf("invalid maker address: %q", params.Maker)
	}
	if !common.IsHexAddress(params.Signer) {
		return nil, fmt.Errorf("invalid signer address: %q", params.Signer)
	}
	if strings.TrimSpace(params.TokenID) == "" {
		return nil, fmt.Errorf("token id is required")
	}
	if strings.TrimSpace(params.MakerAmount) == "" {
		return nil, fmt.Errorf("maker amount is required")
	}
	if strings.TrimSpace(params.TakerAmount) == "" {
		return nil, fmt.Errorf("taker amount is required")
	}
	timestamp := params.Timestamp
	if timestamp < 0 {
		return nil, fmt.Errorf("timestamp must be non-negative")
	}
	metadata := params.Metadata
	if strings.TrimSpace(metadata) == "" {
		metadata = zeroBytes32
	}
	builder := params.Builder
	if strings.TrimSpace(builder) == "" {
		builder = zeroBytes32
	}
	side, err := comboSideInt(params.Side)
	if err != nil {
		return nil, err
	}
	return &ComboOrder{
		Salt:          params.Salt,
		Maker:         common.HexToAddress(params.Maker).Hex(),
		Signer:        common.HexToAddress(params.Signer).Hex(),
		TokenID:       params.TokenID,
		MakerAmount:   params.MakerAmount,
		TakerAmount:   params.TakerAmount,
		Side:          side,
		SignatureType: params.SignatureType,
		Timestamp:     strconv.FormatInt(timestamp, 10),
		Metadata:      metadata,
		Builder:       builder,
	}, nil
}

// BuildComboOrderEnvelope returns the ERC-7739 envelope for a Deposit Wallet Combo order.
func BuildComboOrderEnvelope(order *ComboOrder, chainID int64, verifyingContract string) (*clob.TypedDataSignEnvelope, error) {
	if order == nil {
		return nil, fmt.Errorf("combo order is required")
	}
	if !common.IsHexAddress(order.Signer) {
		return nil, fmt.Errorf("invalid signer address: %q", order.Signer)
	}
	domain, typesDef, message, err := comboOrderTypedDataParts(order, chainID, verifyingContract)
	if err != nil {
		return nil, err
	}
	return clob.BuildTypedDataSignEnvelope(order.Signer, chainID, domain, typesDef, message, "Order")
}

// BuildComboOrderTypedData returns the plain Exchange v3 typed-data payload.
func BuildComboOrderTypedData(order *ComboOrder, chainID int64, verifyingContract string) (json.RawMessage, error) {
	if order == nil {
		return nil, fmt.Errorf("combo order is required")
	}
	domain, typesDef, message, err := comboOrderTypedDataParts(order, chainID, verifyingContract)
	if err != nil {
		return nil, err
	}
	typedData := apitypes.TypedData{
		Types:       typesDef,
		PrimaryType: "Order",
		Domain:      *domain,
		Message:     message,
	}
	return clob.MarshalTypedDataForSigning(typedData)
}

// WithSignature returns a signed-order payload preserving the original order fields.
func (o ComboOrder) WithSignature(signature string) ComboSignedOrder {
	return ComboSignedOrder{
		Salt:          o.Salt,
		Maker:         o.Maker,
		Signer:        o.Signer,
		TokenID:       o.TokenID,
		MakerAmount:   o.MakerAmount,
		TakerAmount:   o.TakerAmount,
		Side:          o.Side,
		SignatureType: o.SignatureType,
		Timestamp:     o.Timestamp,
		Metadata:      o.Metadata,
		Builder:       o.Builder,
		Signature:     signature,
	}
}

func comboOrderTypedDataParts(order *ComboOrder, chainID int64, verifyingContract string) (*apitypes.TypedDataDomain, apitypes.Types, apitypes.TypedDataMessage, error) {
	contract := strings.TrimSpace(verifyingContract)
	if contract == "" {
		contract = DefaultExchangeV3Contract
	}
	if !common.IsHexAddress(contract) {
		return nil, nil, nil, fmt.Errorf("invalid verifying contract: %q", contract)
	}
	if chainID == 0 {
		chainID = auth.PolygonChainID
	}
	salt, err := parseUnsignedDecimal(order.Salt, "salt")
	if err != nil {
		return nil, nil, nil, err
	}
	tokenID, err := parseUnsignedDecimal(order.TokenID, "token id")
	if err != nil {
		return nil, nil, nil, err
	}
	makerAmount, err := parseUnsignedDecimal(order.MakerAmount, "maker amount")
	if err != nil {
		return nil, nil, nil, err
	}
	takerAmount, err := parseUnsignedDecimal(order.TakerAmount, "taker amount")
	if err != nil {
		return nil, nil, nil, err
	}
	timestamp, err := parseUnsignedDecimal(order.Timestamp, "timestamp")
	if err != nil {
		return nil, nil, nil, err
	}

	domain := &apitypes.TypedDataDomain{
		Name:              exchangeDomainName,
		Version:           exchangeDomainVersion,
		ChainId:           math.NewHexOrDecimal256(chainID),
		VerifyingContract: common.HexToAddress(contract).Hex(),
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
		"salt":          (*math.HexOrDecimal256)(salt),
		"maker":         common.HexToAddress(order.Maker).Hex(),
		"signer":        common.HexToAddress(order.Signer).Hex(),
		"tokenId":       (*math.HexOrDecimal256)(tokenID),
		"makerAmount":   (*math.HexOrDecimal256)(makerAmount),
		"takerAmount":   (*math.HexOrDecimal256)(takerAmount),
		"side":          (*math.HexOrDecimal256)(big.NewInt(int64(order.Side))),
		"signatureType": (*math.HexOrDecimal256)(big.NewInt(int64(order.SignatureType))),
		"timestamp":     (*math.HexOrDecimal256)(timestamp),
		"metadata":      order.Metadata,
		"builder":       order.Builder,
	}
	return domain, typesDef, message, nil
}

func comboSideInt(direction Direction) (int, error) {
	switch Direction(strings.ToUpper(string(direction))) {
	case DirectionBuy:
		return 0, nil
	case DirectionSell:
		return 1, nil
	default:
		return 0, fmt.Errorf("invalid combo order side: %q", direction)
	}
}

func parseUnsignedDecimal(value string, field string) (*big.Int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, fmt.Errorf("%s is required", field)
	}
	parsed, ok := new(big.Int).SetString(trimmed, 10)
	if !ok || parsed.Sign() < 0 {
		return nil, fmt.Errorf("%s must be an unsigned decimal string", field)
	}
	return parsed, nil
}
