package clob

import (
	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
	"github.com/neor-it/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/neor-it/polymarket-go-sdk/pkg/types"
	"github.com/shopspring/decimal"
	"math/big"
	"testing"
)

const defaultsTimestamp = int64(1700000000000)

type zeroMakerSigner struct{ auth.Signer }

func (s zeroMakerSigner) Address() types.Address { return types.Address{} }
func TestSignOrderDefaultsSignerAcrossWalletsAndProtocols(t *testing.T) {
	signer := mustSigner(t)
	for _, protocol := range []types.ProtocolVersion{types.ProtocolV1, types.ProtocolV2} {
		for _, kind := range []auth.SignatureType{auth.SignatureEOA, auth.SignatureProxy, auth.SignatureGnosisSafe} {
			kindValue := int(kind)
			order := &clobtypes.Order{ProtocolVersion: protocol, TokenID: types.U256{Int: big.NewInt(1)}, Salt: types.U256{Int: big.NewInt(1)}, Side: "BUY", MakerAmount: decimal.NewFromInt(1), TakerAmount: decimal.NewFromInt(1), Timestamp: defaultsTimestamp, SignatureType: &kindValue}
			signed, err := SignOrder(signer, &auth.APIKey{Key: "key"}, order)
			if err != nil {
				t.Fatal(err)
			}
			if signed.Order.Signer != signer.Address() || signed.Order.Maker == (types.Address{}) || signed.Order.ProtocolVersion != protocol {
				t.Fatalf("default signer/domain lost: %+v", signed.Order)
			}
		}
	}
	order := &clobtypes.Order{TokenID: types.U256{Int: big.NewInt(1)}, Side: "BUY", MakerAmount: decimal.NewFromInt(1), TakerAmount: decimal.NewFromInt(1)}
	if _, err := SignOrder(zeroMakerSigner{signer}, &auth.APIKey{Key: "key"}, order); err == nil {
		t.Fatal("zero derived maker accepted")
	}
}
