package relayer

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
	"github.com/neor-it/polymarket-go-sdk/pkg/transport"
)

var factoryByChain = map[int64]common.Address{
	137: common.HexToAddress(FactoryPolygon),
}

type clientImpl struct {
	httpClient *transport.Client
	signer     auth.Signer
	chainID    int64
	factory    common.Address
	builderCfg *auth.BuilderConfig
}

func (c *clientImpl) WithSigner(signer auth.Signer) Client {
	next := *c
	next.signer = signer
	if signer != nil && signer.ChainID() != nil {
		next.chainID = signer.ChainID().Int64()
		if factory, ok := factoryByChain[next.chainID]; ok {
			next.factory = factory
		}
	}
	return &next
}

func (c *clientImpl) WithBuilderConfig(cfg *auth.BuilderConfig) Client {
	next := *c
	next.builderCfg = cfg
	return &next
}

// postWithBuilderAuth serializes body, optionally computes POLY_BUILDER_* HMAC headers,
// and calls POST path.
func (c *clientImpl) postWithBuilderAuth(ctx context.Context, path string, body interface{}, dest interface{}) error {
	if c.builderCfg == nil || !c.builderCfg.UseHMACBuilderHeaders() {
		return c.httpClient.Post(ctx, path, body, dest)
	}

	// Pre-serialize body so the HMAC covers the exact bytes sent over the wire.
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}
	bodyStr := string(bodyBytes)

	hdrs, err := c.builderCfg.Headers(ctx, "POST", path, &bodyStr, 0)
	if err != nil {
		return fmt.Errorf("builder headers: %w", err)
	}
	extraHdrs := make(map[string]string, len(hdrs))
	for k, vs := range hdrs {
		if len(vs) > 0 {
			extraHdrs[k] = vs[0]
		}
	}

	return c.httpClient.CallWithHeaders(ctx, "POST", path, nil, json.RawMessage(bodyBytes), dest, extraHdrs)
}

func (c *clientImpl) DeployDepositWallet(ctx context.Context, owner common.Address) (*SubmitResponse, error) {
	if owner == (common.Address{}) {
		return nil, fmt.Errorf("relayer: owner address is required")
	}
	body := map[string]string{
		"type": "WALLET-CREATE",
		"from": owner.Hex(),
		"to":   c.factory.Hex(),
	}
	var resp SubmitResponse
	if err := c.postWithBuilderAuth(ctx, "/submit", body, &resp); err != nil {
		return nil, fmt.Errorf("deploy deposit wallet: %w", err)
	}
	return &resp, nil
}

func (c *clientImpl) GetWalletNonce(ctx context.Context, owner common.Address) (string, error) {
	if owner == (common.Address{}) {
		return "", fmt.Errorf("relayer: owner address is required")
	}
	q := url.Values{}
	q.Set("address", owner.Hex())
	q.Set("type", "WALLET")
	var resp struct {
		Nonce string `json:"nonce"`
	}
	if err := c.httpClient.Get(ctx, "/nonce", q, &resp); err != nil {
		return "", fmt.Errorf("get wallet nonce: %w", err)
	}
	return resp.Nonce, nil
}

func (c *clientImpl) ExecuteWalletBatch(ctx context.Context, owner, wallet common.Address, calls []DepositWalletCall, deadline int64) (*SubmitResponse, error) {
	if c.signer == nil {
		return nil, fmt.Errorf("relayer: signer is required for wallet batch execution; call WithSigner first")
	}
	if owner == (common.Address{}) {
		return nil, fmt.Errorf("relayer: owner address is required")
	}
	if wallet == (common.Address{}) {
		return nil, fmt.Errorf("relayer: wallet address is required")
	}
	if len(calls) == 0 {
		return nil, fmt.Errorf("relayer: at least one call is required")
	}
	if deadline <= 0 {
		return nil, fmt.Errorf("relayer: deadline must be a positive Unix timestamp")
	}

	nonceStr, err := c.GetWalletNonce(ctx, owner)
	if err != nil {
		return nil, err
	}
	nonce := new(big.Int)
	if _, ok := nonce.SetString(nonceStr, 10); !ok {
		return nil, fmt.Errorf("relayer: invalid nonce %q from relayer", nonceStr)
	}

	sig, err := signWalletBatch(c.signer, wallet, nonce, deadline, calls)
	if err != nil {
		return nil, fmt.Errorf("relayer: sign wallet batch: %w", err)
	}

	type callJSON struct {
		Target string `json:"target"`
		Value  string `json:"value"`
		Data   string `json:"data"`
	}
	callsJSON := make([]callJSON, len(calls))
	for i, call := range calls {
		v := call.Value
		if v == "" {
			v = "0"
		}
		d := call.Data
		if d == "" {
			d = "0x"
		}
		callsJSON[i] = callJSON{Target: call.Target, Value: v, Data: d}
	}

	body := map[string]interface{}{
		"type":      "WALLET",
		"from":      owner.Hex(),
		"to":        c.factory.Hex(),
		"wallet":    wallet.Hex(),
		"nonce":     nonceStr,
		"deadline":  fmt.Sprintf("%d", deadline),
		"calls":     callsJSON,
		"signature": hexutil.Encode(sig),
	}

	var resp SubmitResponse
	if err := c.postWithBuilderAuth(ctx, "/submit", body, &resp); err != nil {
		return nil, fmt.Errorf("execute wallet batch: %w", err)
	}
	return &resp, nil
}

func (c *clientImpl) GetTransaction(ctx context.Context, id string) ([]Transaction, error) {
	if id == "" {
		return nil, fmt.Errorf("relayer: transaction ID is required")
	}
	q := url.Values{}
	q.Set("id", id)
	var resp []Transaction
	if err := c.httpClient.Get(ctx, "/transaction", q, &resp); err != nil {
		return nil, fmt.Errorf("get transaction: %w", err)
	}
	return resp, nil
}

func (c *clientImpl) IsWalletDeployed(ctx context.Context, wallet common.Address) (bool, error) {
	if wallet == (common.Address{}) {
		return false, fmt.Errorf("relayer: wallet address is required")
	}
	q := url.Values{}
	q.Set("address", wallet.Hex())
	q.Set("type", "WALLET")
	var resp struct {
		Deployed bool `json:"deployed"`
	}
	if err := c.httpClient.Get(ctx, "/deployed", q, &resp); err != nil {
		return false, fmt.Errorf("check wallet deployed: %w", err)
	}
	return resp.Deployed, nil
}

func (c *clientImpl) WaitForConfirmation(ctx context.Context, id string) (*Transaction, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			txs, err := c.GetTransaction(ctx, id)
			if err != nil {
				return nil, err
			}
			if len(txs) == 0 {
				continue
			}
			tx := txs[0]
			switch tx.State {
			case StateConfirmed:
				return &tx, nil
			case StateInvalid, StateFailed:
				return &tx, fmt.Errorf("relayer: transaction %s reached terminal state %s", id, tx.State)
			}
		}
	}
}

// ERC20ApproveCall returns a DepositWalletCall for ERC-20 approve(spender, amount).
// Use MaxUint256 for an unlimited approval.
func ERC20ApproveCall(token, spender common.Address, amount *big.Int) DepositWalletCall {
	// approve(address,uint256) selector = 0x095ea7b3
	var spenderPadded [32]byte
	copy(spenderPadded[12:], spender[:])

	var amountPadded [32]byte
	if amount != nil {
		b := amount.Bytes()
		if len(b) <= 32 {
			copy(amountPadded[32-len(b):], b)
		}
	}

	calldata := make([]byte, 68)
	calldata[0] = 0x09
	calldata[1] = 0x5e
	calldata[2] = 0xa7
	calldata[3] = 0xb3
	copy(calldata[4:36], spenderPadded[:])
	copy(calldata[36:68], amountPadded[:])

	return DepositWalletCall{
		Target: token.Hex(),
		Value:  "0",
		Data:   hexutil.Encode(calldata),
	}
}
