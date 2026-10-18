package rfq

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/neor-it/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/neor-it/polymarket-go-sdk/pkg/types"
	"math/big"
	"testing"
)

const currentTimestamp = int64(1700000000000)

func TestCurrentRFQOrdersKeepSignedFieldsWithoutDeprecatedNonce(t *testing.T) {
	signed := &clobtypes.SignedOrder{Order: clobtypes.Order{TokenID: types.U256{Int: big.NewInt(123)}, Salt: types.U256{Int: big.NewInt(1)}, Timestamp: currentTimestamp, Metadata: common.HexToHash("0x01"), Builder: common.HexToHash("0x02")}, Signature: "0xsignature", Owner: "key"}
	accept, err := BuildRFQAcceptRequestFromSignedOrder("request", "quote", signed)
	if err != nil {
		t.Fatal(err)
	}
	approve, err := BuildRFQApproveQuoteFromSignedOrder("request", "quote", signed)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []any{accept, approve} {
		body, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]json.RawMessage
		if err = json.Unmarshal(body, &fields); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"nonce", "taker", "feeRateBps"} {
			if _, ok := fields[name]; ok {
				t.Fatalf("obsolete signed field %s", name)
			}
		}
		for _, name := range []string{"timestamp", "metadata", "builder"} {
			if _, ok := fields[name]; !ok {
				t.Fatalf("missing signed field %s", name)
			}
		}
	}
	if accept.Timestamp != currentTimestamp || accept.Metadata != signed.Order.Metadata.Hex() || approve.Builder != signed.Order.Builder.Hex() {
		t.Fatal("signed order fields changed")
	}
}
