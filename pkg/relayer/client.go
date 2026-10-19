// Package relayer provides a Go client for the Polymarket relayer API.
// It covers Deposit Wallet deployment and batch signing, typed SAFE/PROXY
// submissions, public nonce/deployment reads, authenticated history and API keys.
package relayer

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
	"github.com/neor-it/polymarket-go-sdk/pkg/transport"
)

// Client provides gasless deposit wallet management via the Polymarket relayer.
type Client interface {
	// WithRelayerCredentials returns a copy with address-scoped relayer credentials.
	// The credential value is copied; passing nil removes these credentials.
	WithRelayerCredentials(credentials *RelayerCredentials) Client

	// Submit submits an already-signed SAFE/PROXY transaction using builder or relayer authentication.
	Submit(ctx context.Context, request *SubmitRequest) (*SubmitResponse, error)

	// GetTransactions lists recent authenticated transactions. The API exposes no pagination contract.
	GetTransactions(ctx context.Context) ([]Transaction, error)

	// GetNonce reads the explicitly selected SAFE/PROXY signer nonce.
	GetNonce(ctx context.Context, request *NonceRequest) (*NonceResponse, error)

	// GetRelayPayload reads the relayer address and explicitly selected SAFE/PROXY nonce.
	GetRelayPayload(ctx context.Context, request *NonceRequest) (*RelayPayloadResponse, error)

	// GetDeployed checks a SAFE or Deposit Wallet address without changing existing wallet helpers.
	GetDeployed(ctx context.Context, request *DeployedRequest) (*DeployedResponse, error)

	// GetAPIKeys lists keys using Relayer API credentials, not CLOB L2 or builder headers.
	GetAPIKeys(ctx context.Context) ([]APIKeyResponse, error)

	// WithSigner returns a new Client configured to use the given signer for
	// EIP-712 Batch signing.  ChainID and factory address are updated to match.
	WithSigner(signer auth.Signer) Client

	// WithBuilderConfig returns a new Client that attaches POLY_BUILDER_* HMAC
	// headers to authenticated requests (POST /submit).
	WithBuilderConfig(cfg *auth.BuilderConfig) Client

	// DeployDepositWallet submits a WALLET-CREATE transaction to deploy a new
	// deposit wallet for owner.  No signature is required.
	// Poll the returned TransactionID with WaitForConfirmation.
	DeployDepositWallet(ctx context.Context, owner common.Address) (*SubmitResponse, error)

	// GetWalletNonce returns the current WALLET nonce for owner.
	// Fetched automatically inside ExecuteWalletBatch; exposed for inspection.
	GetWalletNonce(ctx context.Context, owner common.Address) (string, error)

	// ExecuteWalletBatch signs and submits a WALLET batch transaction.
	//   owner    – the EOA or session key address
	//   wallet   – the deployed deposit wallet address
	//   calls    – the on-chain calls to execute (e.g. ERC-20 approvals)
	//   deadline – Unix seconds after which the batch expires (must be > 0)
	// Requires WithSigner to have been called first.
	ExecuteWalletBatch(ctx context.Context, owner, wallet common.Address, calls []DepositWalletCall, deadline int64) (*SubmitResponse, error)

	// GetTransaction returns the current status record(s) for the given relayer
	// transaction ID.
	GetTransaction(ctx context.Context, id string) ([]Transaction, error)

	// IsWalletDeployed reports whether the deposit wallet contract is live on-chain.
	IsWalletDeployed(ctx context.Context, wallet common.Address) (bool, error)

	// WaitForConfirmation polls the relayer until the transaction reaches
	// STATE_CONFIRMED or a terminal error state (STATE_INVALID or STATE_FAILED).
	// Cancelling ctx stops the poll and returns ctx.Err().
	WaitForConfirmation(ctx context.Context, id string) (*Transaction, error)
}

// NewClient creates a relayer Client targeting the given transport.
// Pass nil to create a default transport pointed at DefaultURL.
// Chain defaults to Polygon mainnet (137); call WithSigner to update.
func NewClient(httpClient *transport.Client) Client {
	if httpClient == nil {
		httpClient = transport.NewClient(nil, DefaultURL)
	}
	return &clientImpl{
		httpClient: httpClient,
		chainID:    137,
		factory:    common.HexToAddress(FactoryPolygon),
	}
}
