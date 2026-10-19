package clob

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
	"github.com/neor-it/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/neor-it/polymarket-go-sdk/pkg/types"
)

func TestOrderPayloadPreservesUint256SaltForEveryWallet(t *testing.T) {
	maximum := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), orderUint256Bits), big.NewInt(1))
	for _, signature := range []auth.SignatureType{auth.SignatureEOA, auth.SignatureProxy, auth.SignatureGnosisSafe, auth.SignaturePoly1271} {
		t.Run(signatureName(signature), func(t *testing.T) {
			signatureType := int(signature)
			order := &clobtypes.SignedOrder{Order: clobtypes.Order{Salt: types.U256{Int: maximum}, Side: "BUY", SignatureType: &signatureType}, Owner: "fixture-key", Signature: "0x12"}
			payload, err := buildOrderPayload(order)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(raw), `"salt":`+maximum.String()+`,`) {
				t.Fatalf("salt must remain an exact unquoted JSON integer: %s", raw)
			}
			for _, invalid := range []*big.Int{big.NewInt(-1), new(big.Int).Add(maximum, big.NewInt(1))} {
				order.Order.Salt.Int = invalid
				if _, err := buildOrderPayload(order); err == nil {
					t.Fatalf("invalid salt %s accepted", invalid)
				}
			}
		})
	}
}

func signatureName(signature auth.SignatureType) string {
	switch signature {
	case auth.SignatureEOA:
		return "EOA"
	case auth.SignatureProxy:
		return "Proxy"
	case auth.SignatureGnosisSafe:
		return "Safe"
	default:
		return "POLY_1271"
	}
}
