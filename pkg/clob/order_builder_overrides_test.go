package clob

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
	"github.com/neor-it/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/neor-it/polymarket-go-sdk/pkg/types"
)

const overrideTimestamp = int64(1700000000123)

func TestOrderBuilderOverridesPreserveProtocolAndWalletDefaults(t *testing.T) {
	metadata := "0x" + strings.Repeat("ab", common.HashLength)
	configured := "0x" + strings.Repeat("cd", common.HashLength)
	override := "0x" + strings.Repeat("ef", common.HashLength)
	client := &clientImpl{builderCfg: &auth.BuilderConfig{Code: configured}}
	for _, protocol := range []types.ProtocolVersion{types.ProtocolV1, types.ProtocolV2} {
		for _, signature := range []auth.SignatureType{auth.SignatureEOA, auth.SignatureProxy, auth.SignatureGnosisSafe, auth.SignaturePoly1271} {
			t.Run(string(protocol)+"/"+signatureName(signature), func(t *testing.T) {
				for _, market := range []bool{false, true} {
					walletClient := clientImpl{builderCfg: client.builderCfg}
					walletClient.signatureType = signature
					if signature == auth.SignaturePoly1271 {
						wallet := common.HexToAddress("0x0000000000000000000000000000000000000011")
						walletClient.funder = &wallet
					}
					builder := NewOrderBuilder(&walletClient, mustSigner(t)).ProtocolVersion(protocol).TokenID("123").Side("BUY").Price(0.5).Size(10).TickSize(0.01).Timestamp(overrideTimestamp).Metadata(metadata).Builder(override)
					var order *clobtypes.Order
					var err error
					if market {
						var signable *clobtypes.SignableOrder
						signable, err = builder.AmountUSDC(5).BuildMarketWithContext(context.Background())
						if err == nil {
							order = signable.Order
						}
					} else {
						order, err = builder.BuildWithContext(context.Background())
					}
					if err != nil {
						t.Fatal(err)
					}
					if order.ProtocolVersion != protocol || order.Timestamp != overrideTimestamp || order.Metadata.Hex() != metadata || order.Builder.Hex() != override || *order.SignatureType != int(signature) {
						t.Fatalf("order overrides/domain changed: %+v", order)
					}
					if signature == auth.SignaturePoly1271 && order.Signer != order.Maker {
						t.Fatal("deposit wallet signer no longer equals maker")
					}
					signed, err := SignOrder(mustSigner(t), &auth.APIKey{Key: "fixture-key"}, order)
					if err != nil {
						t.Fatal(err)
					}
					if signed.Order.Timestamp != overrideTimestamp || signed.Order.Builder.Hex() != override || signed.Order.Metadata.Hex() != metadata || signed.Order.ProtocolVersion != protocol {
						t.Fatalf("signing changed per-order overrides: %+v", signed.Order)
					}
					if _, err := buildOrderPayload(signed); err != nil {
						t.Fatalf("signed override order cannot be serialized: %v", err)
					}
				}
			})
		}
	}
	before := time.Now().UnixMilli()
	order, err := NewOrderBuilder(client, mustSigner(t)).TokenID("123").Side("BUY").Price(0.5).Size(10).TickSize(0.01).Build()
	if err != nil {
		t.Fatal(err)
	}
	if order.Builder.Hex() != configured || order.Metadata != (common.Hash{}) || order.Timestamp < before {
		t.Fatalf("configured builder or automatic timestamp changed: %+v", order)
	}
}

func TestOrderBuilderRejectsInvalidOverrides(t *testing.T) {
	for _, configure := range []func(*OrderBuilder){
		func(b *OrderBuilder) { b.Timestamp(-1) },
		func(b *OrderBuilder) { b.Metadata("0x12") },
		func(b *OrderBuilder) { b.Metadata("not-hex") },
		func(b *OrderBuilder) { b.Builder("0x12") },
	} {
		for _, market := range []bool{false, true} {
			builder := NewOrderBuilder(nil, mustSigner(t)).TokenID("123").Side("BUY").Price(0.5).Size(10).TickSize(0.01)
			configure(builder)
			var err error
			if market {
				_, err = builder.AmountUSDC(5).BuildMarket()
			} else {
				_, err = builder.Build()
			}
			if err == nil {
				t.Fatal("invalid override accepted")
			}
		}
	}
}
