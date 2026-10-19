package relayer

// TransactionType selects a relayer transaction or nonce domain.
type TransactionType string

const (
	// TransactionSafe selects a Gnosis Safe transaction.
	TransactionSafe TransactionType = "SAFE"
	// TransactionProxy selects a legacy proxy transaction.
	TransactionProxy TransactionType = "PROXY"
	// TransactionWallet selects a Deposit Wallet deployment lookup.
	TransactionWallet TransactionType = "WALLET"
)

// SignatureParams contains the optional fields of the official relayer signature schema.
// Numeric values are decimal strings; no floating-point conversion is performed.
type SignatureParams struct {
	GasPrice       string `json:"gasPrice,omitempty"`
	Operation      string `json:"operation,omitempty"`
	SafeTxnGas     string `json:"safeTxnGas,omitempty"`
	BaseGas        string `json:"baseGas,omitempty"`
	GasToken       string `json:"gasToken,omitempty"`
	RefundReceiver string `json:"refundReceiver,omitempty"`
}

// SubmitRequest is the flat, already-signed SAFE/PROXY payload accepted by POST /submit.
// Deposit Wallet batches continue to use ExecuteWalletBatch.
type SubmitRequest struct {
	From            string           `json:"from"`
	To              string           `json:"to"`
	ProxyWallet     string           `json:"proxyWallet"`
	Data            string           `json:"data"`
	Nonce           string           `json:"nonce"`
	Signature       string           `json:"signature"`
	SignatureParams *SignatureParams `json:"signatureParams"`
	Type            TransactionType  `json:"type"`
}

// NonceRequest explicitly selects the signer address and SAFE/PROXY nonce domain.
type NonceRequest struct {
	Address string
	Type    TransactionType
}

// DeployedRequest selects a wallet address and SAFE/WALLET deployment domain.
// Empty Type uses the API's SAFE default.
type DeployedRequest struct {
	Address string
	Type    TransactionType
}

// NonceResponse preserves the exact decimal nonce returned by the relayer.
type NonceResponse struct {
	Nonce string `json:"nonce"`
}

// RelayPayloadResponse contains the relayer address and current nonce.
type RelayPayloadResponse struct {
	Address string `json:"address"`
	Nonce   string `json:"nonce"`
}

// DeployedResponse reports whether the requested wallet is deployed.
type DeployedResponse struct {
	Deployed bool `json:"deployed"`
}

// APIKeyResponse represents one key returned by GET /relayer/api/keys.
type APIKeyResponse struct {
	APIKey    string `json:"apiKey"`
	Address   string `json:"address"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// RelayerCredentials authenticate the address-scoped Relayer API key mechanism.
// These are separate from CLOB L2 and builder credentials.
type RelayerCredentials struct {
	APIKey  string
	Address string
}
