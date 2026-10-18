package positions

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

// ContractCaller is the read-only RPC boundary implemented by ethclient.Client.
type ContractCaller interface {
	CallContract(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error)
}

// Client reads V2 position state without broadcasting transactions.
type Client interface {
	// BalanceOf reads shares for one owner and full position ID.
	BalanceOf(context.Context, common.Address, *big.Int) (*big.Int, error)
	// GetPayout returns base-unit proceeds; unresolved state remains an error.
	GetPayout(context.Context, *big.Int, *big.Int) (*big.Int, error)
	// IsApprovedForAll reads the PositionManager owner/operator pair.
	IsApprovedForAll(context.Context, common.Address, common.Address) (bool, error)
	// Allowance reads pUSD permission for one exact spender.
	Allowance(context.Context, common.Address, common.Address) (*big.Int, error)
	// GetResult reads BinaryModule or NegRiskModule native/derived payouts.
	GetResult(context.Context, common.Address, string) ([]*big.Int, error)
}

type positionClient struct {
	caller ContractCaller
	config Config
}

// NewClient creates a read-only V2 client; all approvals and writes remain explicit calldata.
func NewClient(caller ContractCaller, cfg Config) (Client, error) {
	if caller == nil {
		return nil, errors.New("position contract caller is required")
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &positionClient{caller: caller, config: cfg}, nil
}

func (c *positionClient) read(ctx context.Context, to common.Address, contractABI, method string, args ...any) ([]any, error) {
	parsed, err := contractDefinition(contractABI)
	if err != nil {
		return nil, errors.Wrap(err, "parse position read ABI")
	}
	input, err := parsed.Pack(method, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "encode %s read", method)
	}
	output, err := c.caller.CallContract(ctx, ethereum.CallMsg{To: &to, Data: input}, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "call %s", method)
	}
	values, err := parsed.Unpack(method, output)
	if err != nil {
		return nil, errors.Wrapf(err, "decode %s output", method)
	}
	if len(values) != 1 {
		return nil, errors.Errorf("%s returned %d values, expected one", method, len(values))
	}
	return values, nil
}

// BalanceOf reads a PositionManager holding using its full uint256 position ID.
func (c *positionClient) BalanceOf(ctx context.Context, owner common.Address, positionID *big.Int) (*big.Int, error) {
	if owner == (common.Address{}) {
		return nil, errors.New("position owner must not be zero")
	}
	if err := validateUint(positionID, false, "position ID"); err != nil {
		return nil, err
	}
	values, err := c.read(ctx, c.config.PositionManager, PositionManagerABI, "balanceOf", owner, positionID)
	if err != nil {
		return nil, err
	}
	return new(big.Int).Set(values[0].(*big.Int)), nil
}

// GetPayout preserves unresolved RPC reverts as errors and valid losing payouts as zero.
func (c *positionClient) GetPayout(ctx context.Context, positionID, amount *big.Int) (*big.Int, error) {
	if err := validateUint(positionID, false, "position ID"); err != nil {
		return nil, err
	}
	if err := validateUint(amount, false, "payout amount"); err != nil {
		return nil, err
	}
	values, err := c.read(ctx, c.config.PositionManager, PositionManagerABI, "getPayout", positionID, amount)
	if err != nil {
		return nil, err
	}
	return new(big.Int).Set(values[0].(*big.Int)), nil
}

// IsApprovedForAll reads the permission on PositionManager, independent of CTF approvals.
func (c *positionClient) IsApprovedForAll(ctx context.Context, owner, operator common.Address) (bool, error) {
	if owner == (common.Address{}) || operator == (common.Address{}) {
		return false, errors.New("approval owner and operator must not be zero")
	}
	values, err := c.read(ctx, c.config.PositionManager, PositionManagerABI, "isApprovedForAll", owner, operator)
	if err != nil {
		return false, err
	}
	return values[0].(bool), nil
}

// Allowance reads the pUSD allowance for the exact spender required by the operation.
func (c *positionClient) Allowance(ctx context.Context, owner, spender common.Address) (*big.Int, error) {
	if owner == (common.Address{}) || spender == (common.Address{}) {
		return nil, errors.New("allowance owner and spender must not be zero")
	}
	values, err := c.read(ctx, c.config.Collateral, CollateralABI, "allowance", owner, spender)
	if err != nil {
		return nil, err
	}
	return new(big.Int).Set(values[0].(*big.Int)), nil
}

// GetResult reads native or derived module results; an empty vector remains unresolved.
func (c *positionClient) GetResult(ctx context.Context, module common.Address, condition string) ([]*big.Int, error) {
	if module != c.config.BinaryModule && module != c.config.NegRiskModule {
		return nil, errors.New("result module is not part of the configured deployment")
	}
	id, err := ConditionID(condition)
	if err != nil {
		return nil, err
	}
	values, err := c.read(ctx, module, ModuleABI, "getResult", id)
	if err != nil {
		return nil, err
	}
	return values[0].([]*big.Int), nil
}
