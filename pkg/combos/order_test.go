package combos

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildComboOrderAndEnvelope(t *testing.T) {
	order, err := BuildComboOrder(ComboOrderParams{
		Salt:          "1",
		Maker:         "0x0000000000000000000000000000000000000001",
		Signer:        "0x0000000000000000000000000000000000000001",
		TokenID:       "123",
		MakerAmount:   "1000000",
		TakerAmount:   "250000",
		Side:          DirectionBuy,
		SignatureType: 3,
		Timestamp:     1700000000,
	})
	if err != nil {
		t.Fatalf("BuildComboOrder returned error: %v", err)
	}
	if order.Side != 0 {
		t.Fatalf("side = %d", order.Side)
	}
	if order.Metadata != zeroBytes32 || order.Builder != zeroBytes32 {
		t.Fatalf("default bytes32 fields not set: %+v", order)
	}

	typedData, err := BuildComboOrderTypedData(order, 137, "")
	if err != nil {
		t.Fatalf("BuildComboOrderTypedData returned error: %v", err)
	}
	if !strings.Contains(string(typedData), `"version":"3"`) {
		t.Fatalf("typed data does not use exchange v3: %s", string(typedData))
	}

	envelope, err := BuildComboOrderEnvelope(order, 137, "")
	if err != nil {
		t.Fatalf("BuildComboOrderEnvelope returned error: %v", err)
	}
	if len(envelope.TypedData) == 0 || envelope.AppDomainSeparator == "" || envelope.ContentsHash == "" {
		t.Fatalf("invalid envelope: %+v", envelope)
	}
}

func TestComboSignedOrderJSON(t *testing.T) {
	order := ComboOrder{
		Salt: "1", Maker: "0x0000000000000000000000000000000000000001", Signer: "0x0000000000000000000000000000000000000001",
		TokenID: "123", MakerAmount: "1000000", TakerAmount: "250000", Side: 0, SignatureType: 3, Timestamp: "1700000000",
		Metadata: zeroBytes32, Builder: zeroBytes32,
	}
	signed := order.WithSignature("0xsig")
	raw, err := json.Marshal(signed)
	if err != nil {
		t.Fatalf("marshal signed order: %v", err)
	}
	if !strings.Contains(string(raw), `"tokenId":"123"`) || !strings.Contains(string(raw), `"signature":"0xsig"`) {
		t.Fatalf("unexpected signed order json: %s", string(raw))
	}
}

func TestBuildComboOrderValidation(t *testing.T) {
	_, err := BuildComboOrder(ComboOrderParams{
		Salt: "1", Maker: "bad", Signer: "0x0000000000000000000000000000000000000001",
		TokenID: "123", MakerAmount: "1", TakerAmount: "1", Side: DirectionBuy,
	})
	if err == nil {
		t.Fatal("expected invalid maker error")
	}

	_, err = BuildComboOrder(ComboOrderParams{
		Salt: "1", Maker: "0x0000000000000000000000000000000000000001", Signer: "0x0000000000000000000000000000000000000001",
		TokenID: "123", MakerAmount: "1", TakerAmount: "1", Side: Direction("HOLD"),
	})
	if err == nil {
		t.Fatal("expected invalid side error")
	}
}
