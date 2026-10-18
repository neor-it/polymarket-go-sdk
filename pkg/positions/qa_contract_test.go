package positions_test

import (
	"bytes"
	"context"
	stderrors "errors"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/neor-it/polymarket-go-sdk/pkg/positions"
)

const qaWordBytes = 32
const qaSelectorBytes = 4
const qaPositionBits = 256
const qaUnit = 1000000

type qaCaller func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error)

func (fn qaCaller) CallContract(ctx context.Context, msg ethereum.CallMsg, block *big.Int) ([]byte, error) {
	return fn(ctx, msg, block)
}

func qaWord(value *big.Int) []byte { return value.FillBytes(make([]byte, qaWordBytes)) }
func qaSelector(signature string) []byte {
	return crypto.Keccak256([]byte(signature))[:qaSelectorBytes]
}

func TestQAOfficialRouterWireIsRightPaddedAndExplicit(t *testing.T) {
	condition := "0x" + strings.Repeat("ab", qaWordBytes-1)
	amount := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), qaPositionBits), big.NewInt(1))
	conditionWord := append(bytes.Repeat([]byte{0xab}, qaWordBytes-1), byte(0))
	for _, test := range []struct {
		name, signature string
		encode          func(string, *big.Int) ([]byte, error)
	}{
		{"split", "split(bytes31,uint256)", positions.EncodeSplit},
		{"merge", "merge(bytes31,uint256)", positions.EncodeMerge},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.encode(condition, amount)
			if err != nil {
				t.Fatal(err)
			}
			want := append(append(qaSelector(test.signature), conditionWord...), qaWord(amount)...)
			if !bytes.Equal(got, want) {
				t.Fatalf("ABI mismatch got %x want %x", got, want)
			}
			padded, err := test.encode(condition+"00", amount)
			if err != nil || !bytes.Equal(padded, want) {
				t.Fatalf("padded32 mismatch %x %v", padded, err)
			}
			if _, err := test.encode(condition+"01", amount); err == nil {
				t.Fatal("nonzero terminal byte accepted")
			}
		})
	}
	for _, outcome := range []uint64{0, 1} {
		got, err := positions.EncodeRedeem(condition, outcome, amount)
		if err != nil {
			t.Fatal(err)
		}
		want := append(append(append(qaSelector("redeem(bytes31,uint256,uint256)"), conditionWord...), qaWord(new(big.Int).SetUint64(outcome))...), qaWord(amount)...)
		if !bytes.Equal(got, want) {
			t.Fatalf("outcome %d wire mismatch %x", outcome, got)
		}
	}
	if _, err := positions.EncodeRedeem(condition, 2, amount); err == nil {
		t.Fatal("CTF index set 2 accepted as V2 outcome")
	}
	for _, bad := range []*big.Int{nil, big.NewInt(0), big.NewInt(-1), new(big.Int).Lsh(big.NewInt(1), qaPositionBits)} {
		if _, err := positions.EncodeRedeem(condition, 0, bad); err == nil {
			t.Fatalf("invalid amount accepted %v", bad)
		}
	}
}

func TestQAPositionRPCSeparatesErrorZeroAndFullPrecision(t *testing.T) {
	cfg := positions.ConfigPolygon137()
	owner := common.HexToAddress("0x1234567890123456789012345678901234567890")
	id := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), qaPositionBits), big.NewInt(1))
	rpcErr := stderrors.New("unresolved execution reverted")
	for _, test := range []struct {
		name   string
		output []byte
		err    error
		want   *big.Int
	}{
		{"losing", qaWord(big.NewInt(0)), nil, big.NewInt(0)},
		{"fractional", qaWord(big.NewInt(qaUnit / 2)), nil, big.NewInt(qaUnit / 2)},
		{"maximum", qaWord(id), nil, id},
		{"missing", nil, nil, nil},
		{"truncated", []byte{0}, nil, nil},
		{"unresolved", nil, rpcErr, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client, err := positions.NewClient(qaCaller(func(ctx context.Context, msg ethereum.CallMsg, block *big.Int) ([]byte, error) {
				calls++
				if msg.To == nil || *msg.To != cfg.PositionManager || block != nil || msg.Value != nil {
					t.Fatalf("unexpected contract call %+v", msg)
				}
				want := append(append(qaSelector("getPayout(uint256,uint256)"), qaWord(id)...), qaWord(big.NewInt(qaUnit))...)
				if !bytes.Equal(msg.Data, want) {
					t.Fatalf("payout request truncated %x", msg.Data)
				}
				return test.output, test.err
			}), cfg)
			if err != nil {
				t.Fatal(err)
			}
			got, err := client.GetPayout(context.Background(), id, big.NewInt(qaUnit))
			if calls != 1 {
				t.Fatalf("calls %d", calls)
			}
			if test.want != nil {
				if err != nil || got == nil || got.Cmp(test.want) != 0 {
					t.Fatalf("value %v error %v", got, err)
				}
				return
			}
			if err == nil || got != nil {
				t.Fatalf("unavailable converted to payout %v %v", got, err)
			}
			if test.err != nil && !stderrors.Is(err, rpcErr) {
				t.Fatalf("RPC cause lost %v", err)
			}
		})
	}
	client, err := positions.NewClient(qaCaller(func(ctx context.Context, msg ethereum.CallMsg, block *big.Int) ([]byte, error) {
		if !bytes.Equal(msg.Data[:qaSelectorBytes], qaSelector("balanceOf(address,uint256)")) || !bytes.Equal(msg.Data[qaSelectorBytes+qaWordBytes:], qaWord(id)) {
			t.Fatalf("wrong full-width balance call %x", msg.Data)
		}
		return qaWord(id), nil
	}), cfg)
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.BalanceOf(context.Background(), owner, id)
	if err != nil || got.Cmp(id) != 0 {
		t.Fatalf("balance %v %v", got, err)
	}
}

func TestQARPCPropagatesCancellationAndSpenderIdentity(t *testing.T) {
	cfg := positions.ConfigPolygon137()
	owner := common.HexToAddress("0x1234567890123456789012345678901234567890")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client, err := positions.NewClient(qaCaller(func(ctx context.Context, msg ethereum.CallMsg, block *big.Int) ([]byte, error) {
		if msg.To == nil || *msg.To != cfg.Collateral {
			t.Fatal("allowance read on wrong ledger")
		}
		want := append(append(qaSelector("allowance(address,address)"), qaWord(new(big.Int).SetBytes(owner.Bytes()))...), qaWord(new(big.Int).SetBytes(cfg.Exchange.Bytes()))...)
		if !bytes.Equal(msg.Data, want) {
			t.Fatal("spender collapsed or wrong allowance owner")
		}
		return nil, ctx.Err()
	}), cfg)
	if err != nil {
		t.Fatal(err)
	}
	value, err := client.Allowance(ctx, owner, cfg.Exchange)
	if value != nil || !stderrors.Is(err, context.Canceled) {
		t.Fatalf("cancel lost %v %v", value, err)
	}
}
