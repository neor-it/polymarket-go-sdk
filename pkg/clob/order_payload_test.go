package clob

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"

	"github.com/neor-it/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/neor-it/polymarket-go-sdk/pkg/types"
)

func TestBuildOrderPayloadCasingAndOptions(t *testing.T) {
	sigType := 0
	order := clobtypes.SignedOrder{
		Order: clobtypes.Order{
			Salt:          types.U256{Int: big.NewInt(1)},
			Maker:         common.HexToAddress("0x0000000000000000000000000000000000000001"),
			Signer:        common.HexToAddress("0x0000000000000000000000000000000000000002"),
			Taker:         common.HexToAddress("0x0000000000000000000000000000000000000000"),
			TokenID:       types.U256{Int: big.NewInt(123)},
			MakerAmount:   decimal.NewFromInt(100),
			TakerAmount:   decimal.NewFromInt(50),
			Side:          "BUY",
			Expiration:    types.U256{Int: big.NewInt(0)},
			FeeRateBps:    decimal.NewFromInt(0),
			Nonce:         types.U256{Int: big.NewInt(0)},
			SignatureType: &sigType,
			Timestamp:     1713398400000,
			Metadata:      common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000aa"),
			Builder:       common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000bb"),
		},
		Signature: "0xsig",
		Owner:     "builder-owner",
		OrderType: clobtypes.OrderTypeGTC,
		PostOnly:  boolPtr(true),
	}

	payload, err := buildOrderPayload(&order)
	if err != nil {
		t.Fatalf("buildOrderPayload failed: %v", err)
	}

	if payload["owner"] != "builder-owner" {
		t.Fatalf("owner mismatch: got %v", payload["owner"])
	}
	if got := payload["orderType"]; got != clobtypes.OrderTypeGTC {
		t.Fatalf("orderType mismatch: got %v", got)
	}

	orderMap, ok := payload["order"].(map[string]interface{})
	if !ok {
		t.Fatalf("order payload missing order map")
	}
	if orderMap["tokenId"] != "123" {
		t.Fatalf("tokenId mismatch: got %v", orderMap["tokenId"])
	}
	if orderMap["makerAmount"] == nil || orderMap["takerAmount"] == nil {
		t.Fatalf("maker/taker amounts missing in order payload")
	}
	if orderMap["signature"] != "0xsig" {
		t.Fatalf("signature mismatch: got %v", orderMap["signature"])
	}
	if orderMap["expiration"] != "0" {
		t.Fatalf("expiration mismatch: got %v", orderMap["expiration"])
	}
	if orderMap["timestamp"] != "1713398400000" {
		t.Fatalf("timestamp mismatch: got %v", orderMap["timestamp"])
	}
	if orderMap["metadata"] != "0x00000000000000000000000000000000000000000000000000000000000000aa" {
		t.Fatalf("metadata mismatch: got %v", orderMap["metadata"])
	}
	if orderMap["builder"] != "0x00000000000000000000000000000000000000000000000000000000000000bb" {
		t.Fatalf("builder mismatch: got %v", orderMap["builder"])
	}

	for _, legacyField := range []string{"taker", "nonce", "feeRateBps"} {
		if _, ok := orderMap[legacyField]; ok {
			t.Fatalf("legacy v1 field %q must not be included in order payload", legacyField)
		}
	}
	if _, ok := payload["fee_rate_bps"]; ok {
		t.Fatal("legacy top-level fee_rate_bps must not be included in order payload")
	}
}

func TestBuildOrderPayloadPostOnlyValidation(t *testing.T) {
	sigType := 0
	order := clobtypes.SignedOrder{
		Order: clobtypes.Order{
			Salt:          types.U256{Int: big.NewInt(1)},
			Maker:         common.HexToAddress("0x0000000000000000000000000000000000000001"),
			Signer:        common.HexToAddress("0x0000000000000000000000000000000000000002"),
			Taker:         common.HexToAddress("0x0000000000000000000000000000000000000000"),
			TokenID:       types.U256{Int: big.NewInt(123)},
			MakerAmount:   decimal.NewFromInt(100),
			TakerAmount:   decimal.NewFromInt(50),
			Side:          "BUY",
			Expiration:    types.U256{Int: big.NewInt(0)},
			FeeRateBps:    decimal.NewFromInt(0),
			Nonce:         types.U256{Int: big.NewInt(0)},
			SignatureType: &sigType,
		},
		Signature: "0xsig",
		Owner:     "builder-owner",
		OrderType: clobtypes.OrderTypeFAK,
		PostOnly:  boolPtr(true),
	}

	_, err := buildOrderPayload(&order)
	if err == nil || !strings.Contains(err.Error(), "postOnly") {
		t.Fatalf("expected postOnly validation error, got %v", err)
	}
}

func TestBuildOrderPayloadRequiresSignatureAndOwner(t *testing.T) {
	order := clobtypes.SignedOrder{
		Order: clobtypes.Order{
			Salt:        types.U256{Int: big.NewInt(1)},
			Maker:       common.HexToAddress("0x0000000000000000000000000000000000000001"),
			Signer:      common.HexToAddress("0x0000000000000000000000000000000000000002"),
			Taker:       common.HexToAddress("0x0000000000000000000000000000000000000000"),
			TokenID:     types.U256{Int: big.NewInt(123)},
			MakerAmount: decimal.NewFromInt(100),
			TakerAmount: decimal.NewFromInt(50),
			Side:        "BUY",
			Expiration:  types.U256{Int: big.NewInt(0)},
			FeeRateBps:  decimal.NewFromInt(0),
			Nonce:       types.U256{Int: big.NewInt(0)},
		},
	}

	_, err := buildOrderPayload(&order)
	if err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("expected signature validation error, got %v", err)
	}
}

func boolPtr(v bool) *bool {
	return &v
}
