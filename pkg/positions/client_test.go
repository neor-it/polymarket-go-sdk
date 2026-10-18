package positions

import (
	"context"
	stderrors "errors"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type rpcReadFunc func(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error)

func (fn rpcReadFunc) CallContract(ctx context.Context, call ethereum.CallMsg, block *big.Int) ([]byte, error) {
	return fn(ctx, call, block)
}

func TestPayoutReadKeepsUnresolvedDistinctFromExplicitZero(t *testing.T) {
	sentinel := stderrors.New("execution reverted: unresolved")
	for _, test := range []struct {
		name   string
		output []byte
		err    error
		valid  bool
	}{{"losing zero", make([]byte, 32), nil, true}, {"unresolved", nil, sentinel, false}, {"unavailable payload", nil, nil, false}} {
		t.Run(test.name, func(t *testing.T) {
			cfg := ConfigPolygon137()
			caller := rpcReadFunc(func(ctx context.Context, call ethereum.CallMsg, block *big.Int) ([]byte, error) {
				if call.To == nil || *call.To != cfg.PositionManager || string(call.Data[:4]) != string(crypto.Keccak256([]byte("getPayout(uint256,uint256)"))[:4]) {
					t.Fatal("wrong payout read")
				}
				return test.output, test.err
			})
			client, err := NewClient(caller, cfg)
			if err != nil {
				t.Fatal(err)
			}
			value, err := client.GetPayout(context.Background(), big.NewInt(123), big.NewInt(1000000))
			if test.valid {
				if err != nil || value == nil || value.Sign() != 0 {
					t.Fatalf("losing payout %v %v", value, err)
				}
				return
			}
			if err == nil || value != nil {
				t.Fatalf("unavailable payout invented zero %v %v", value, err)
			}
			if test.err != nil && !stderrors.Is(err, sentinel) {
				t.Fatalf("RPC cause lost %v", err)
			}
		})
	}
}

func TestReadBalancesPermissionsAndDerivedModuleResult(t *testing.T) {
	cfg := ConfigPolygon137()
	owner := common.HexToAddress("0x123")
	id := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	amount := big.NewInt(1000001)
	caller := rpcReadFunc(func(ctx context.Context, call ethereum.CallMsg, block *big.Int) ([]byte, error) {
		contract := PositionManagerABI
		if call.To != nil && *call.To == cfg.Collateral {
			contract = CollateralABI
		}
		if call.To != nil && *call.To == cfg.NegRiskModule {
			contract = ModuleABI
		}
		parsed, err := contractDefinition(contract)
		if err != nil {
			return nil, err
		}
		method, err := parsed.MethodById(call.Data[:4])
		if err != nil {
			return nil, err
		}
		switch method.RawName {
		case "balanceOf":
			if new(big.Int).SetBytes(call.Data[36:]).Cmp(id) != 0 {
				t.Fatal("full position ID lost")
			}
			return method.Outputs.Pack(amount)
		case "isApprovedForAll":
			return method.Outputs.Pack(true)
		case "allowance":
			return method.Outputs.Pack(amount)
		case "getResult":
			return method.Outputs.Pack([]*big.Int{big.NewInt(0), big.NewInt(1000000)})
		default:
			t.Fatalf("unexpected read %s", method.RawName)
			return nil, nil
		}
	})
	client, err := NewClient(caller, cfg)
	if err != nil {
		t.Fatal(err)
	}
	value, err := client.BalanceOf(context.Background(), owner, id)
	if err != nil || value.Cmp(amount) != 0 {
		t.Fatalf("balance %v %v", value, err)
	}
	approved, err := client.IsApprovedForAll(context.Background(), owner, cfg.Exchange)
	if err != nil || !approved {
		t.Fatalf("permission %t %v", approved, err)
	}
	value, err = client.Allowance(context.Background(), owner, cfg.Exchange)
	if err != nil || value.Cmp(amount) != 0 {
		t.Fatalf("allowance %v %v", value, err)
	}
	result, err := client.GetResult(context.Background(), cfg.NegRiskModule, "0x"+strings.Repeat("ab", 31))
	if err != nil || len(result) != 2 || result[1].Int64() != 1000000 {
		t.Fatalf("derived result %v %v", result, err)
	}
}
