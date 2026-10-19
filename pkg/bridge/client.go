package bridge

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Client defines the Bridge API interface.
type Client interface {
	// Quote previews bridge costs without transferring assets.
	Quote(context.Context, *BridgeQuoteRequest) (BridgeQuoteResponse, error)
	// WithdrawAddress creates destination-specific bridge addresses without sending funds.
	WithdrawAddress(context.Context, *WithdrawAddressRequest) (WithdrawAddressResponse, error)
	Deposit(ctx context.Context, amount *big.Int, asset common.Address) (*types.Transaction, error)
	Withdraw(ctx context.Context, amount *big.Int, asset common.Address) (*types.Transaction, error)
	WithdrawTo(ctx context.Context, req *WithdrawRequest) (*types.Transaction, error)
	SupportedAssets(ctx context.Context) ([]common.Address, error)

	DepositAddress(ctx context.Context, req *DepositRequest) (DepositResponse, error)
	SupportedAssetsInfo(ctx context.Context) (SupportedAssetsResponse, error)
	Status(ctx context.Context, req *StatusRequest) (StatusResponse, error)
}
