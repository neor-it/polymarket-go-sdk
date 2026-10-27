package ctf_test

import (
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/neor-it/polymarket-go-sdk/pkg/ctf"
)

func splitRequest() *ctf.SplitPositionRequest {
	return &ctf.SplitPositionRequest{
		CollateralToken: common.HexToAddress(testCollateralToken),
		ConditionID:     common.HexToHash("0xabcdef"),
		Partition:       []*big.Int{big.NewInt(testOne), big.NewInt(testTwo)},
		Amount:          big.NewInt(testSplitAmount),
	}
}

func ctfWord(value *big.Int) []byte {
	return value.FillBytes(make([]byte, common.HashLength))
}

func TestEncodeSplitPositionMatchesIndependentABI(t *testing.T) {
	for _, parent := range []common.Hash{{}, common.HexToHash("0x1234")} {
		for _, amount := range []*big.Int{
			big.NewInt(testSplitAmount),
			new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(testOne), testUint256Bits), big.NewInt(testOne)),
		} {
			req := splitRequest()
			req.ParentCollectionID, req.Amount = parent, amount
			expectedAmount := amount.String()
			got, err := ctf.EncodeSplitPosition(req)
			if err != nil {
				t.Fatal(err)
			}
			// Assemble the official five-argument ABI independently of the SDK ABI.
			want := append([]byte{}, crypto.Keccak256([]byte(testSplitMethodSignature))[:testSelectorBytes]...)
			want = append(want, common.LeftPadBytes(req.CollateralToken.Bytes(), common.HashLength)...)
			want = append(want, parent.Bytes()...)
			want = append(want, req.ConditionID.Bytes()...)
			want = append(want, ctfWord(big.NewInt(testSplitHeadWords*common.HashLength))...)
			want = append(want, ctfWord(amount)...)
			want = append(want, ctfWord(big.NewInt(testTwo))...)
			want = append(want, ctfWord(big.NewInt(testOne))...)
			want = append(want, ctfWord(big.NewInt(testTwo))...)
			if !bytes.Equal(got, want) {
				t.Fatalf("split calldata mismatch\ngot  %x\nwant %x", got, want)
			}
			if req.Amount.String() != expectedAmount || req.Partition[0].Int64() != testOne || req.Partition[testOne].Int64() != testTwo {
				t.Fatal("encoder changed request amount or partition")
			}
		}
	}
}

func TestEncodeSplitPositionRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name   string
		change func(*ctf.SplitPositionRequest)
	}{
		{"missing amount", func(req *ctf.SplitPositionRequest) { req.Amount = nil }},
		{"zero amount", func(req *ctf.SplitPositionRequest) { req.Amount = new(big.Int) }},
		{"negative amount", func(req *ctf.SplitPositionRequest) { req.Amount = big.NewInt(-testOne) }},
		{"oversized amount", func(req *ctf.SplitPositionRequest) {
			req.Amount = new(big.Int).Lsh(big.NewInt(testOne), testUint256Bits)
		}},
		{"zero collateral", func(req *ctf.SplitPositionRequest) { req.CollateralToken = common.Address{} }},
		{"missing partition", func(req *ctf.SplitPositionRequest) { req.Partition = nil }},
		{"trivial partition", func(req *ctf.SplitPositionRequest) { req.Partition = []*big.Int{big.NewInt(testOne)} }},
		{"nil index set", func(req *ctf.SplitPositionRequest) { req.Partition[testOne] = nil }},
		{"zero index set", func(req *ctf.SplitPositionRequest) { req.Partition[testOne] = new(big.Int) }},
		{"negative index set", func(req *ctf.SplitPositionRequest) { req.Partition[testOne] = big.NewInt(-testOne) }},
		{"oversized index set", func(req *ctf.SplitPositionRequest) {
			req.Partition[testOne] = new(big.Int).Lsh(big.NewInt(testOne), testUint256Bits)
		}},
		{"overlapping index sets", func(req *ctf.SplitPositionRequest) { req.Partition[testOne] = big.NewInt(testThree) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := splitRequest()
			test.change(req)
			if _, err := ctf.EncodeSplitPosition(req); err == nil {
				t.Fatal("invalid split accepted")
			}
		})
	}
	if _, err := ctf.EncodeSplitPosition(nil); !errors.Is(err, ctf.ErrMissingRequest) {
		t.Fatalf("missing request lost its sentinel: %v", err)
	}
	req := splitRequest()
	req.Amount = nil
	if _, err := ctf.EncodeSplitPosition(req); !errors.Is(err, ctf.ErrMissingU256Value) {
		t.Fatalf("missing amount lost its sentinel: %v", err)
	}
}

func TestCollateralAdapterForChainMatchesPublishedTargets(t *testing.T) {
	for _, test := range []struct {
		negRisk bool
		target  string
	}{{false, testStandardAdapter}, {true, testNegRiskAdapter}} {
		got, err := ctf.CollateralAdapterForChain(ctf.PolygonChainID, test.negRisk)
		if err != nil || got != common.HexToAddress(test.target) {
			t.Fatalf("incorrect adapter for negRisk=%v: %s, %v", test.negRisk, got.Hex(), err)
		}
	}
	for _, chainID := range []int64{0, ctf.AmoyChainID, testUnsupportedChainID} {
		for _, negRisk := range []bool{false, true} {
			got, err := ctf.CollateralAdapterForChain(chainID, negRisk)
			if !errors.Is(err, ctf.ErrConfigNotFound) || got != (common.Address{}) {
				t.Fatalf("unsupported chain received an adapter: %s, %v", got.Hex(), err)
			}
		}
	}
}
