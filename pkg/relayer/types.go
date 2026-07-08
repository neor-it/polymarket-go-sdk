package relayer

import "math/big"

// DepositWalletCall is a single call inside a wallet batch.
type DepositWalletCall struct {
	Target string `json:"target"` // hex-checksummed address
	Value  string `json:"value"`  // decimal string, e.g. "0"
	Data   string `json:"data"`   // 0x-prefixed hex calldata
}

// SubmitResponse is returned by POST /submit.
type SubmitResponse struct {
	TransactionID string `json:"transactionID"`
	State         string `json:"state"`
}

// Transaction is one record from GET /transaction.
type Transaction struct {
	TransactionID   string `json:"transactionID"`
	TransactionHash string `json:"transactionHash"`
	State           string `json:"state"`
	Type            string `json:"type"`
	From            string `json:"from"`
	To              string `json:"to"`
	ProxyAddress    string `json:"proxyAddress"`
	Nonce           string `json:"nonce"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

// Transaction state values returned by the relayer.
const (
	StateNew       = "STATE_NEW"
	StateExecuted  = "STATE_EXECUTED"
	StateMined     = "STATE_MINED"
	StateConfirmed = "STATE_CONFIRMED" // terminal success
	StateInvalid   = "STATE_INVALID"   // terminal error
	StateFailed    = "STATE_FAILED"    // terminal error
)

const (
	// DefaultURL is the Polymarket production relayer.
	DefaultURL = "https://relayer-v2.polymarket.com"

	// FactoryPolygon is the deposit wallet factory on Polygon mainnet (chain 137).
	FactoryPolygon = "0x00000000000Fb5C9ADea0298D729A0CB3823Cc07"
)

// MaxUint256 is 2^256-1, the standard "unlimited" approval amount for ERC-20 tokens.
var MaxUint256 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
