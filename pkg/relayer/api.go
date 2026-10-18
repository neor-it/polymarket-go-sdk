package relayer

import (
	"context"
	"math/big"
	"net/http"
	"net/url"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
)

const relayerUint256Bits = 256

func (c *clientImpl) Submit(ctx context.Context, request *SubmitRequest) (*SubmitResponse, error) {
	if request == nil {
		return nil, errors.New("relayer submit request is required")
	}
	for _, address := range []struct{ name, value string }{{"from", request.From}, {"to", request.To}, {"proxyWallet", request.ProxyWallet}} {
		if !common.IsHexAddress(address.value) || !strings.HasPrefix(address.value, "0x") {
			return nil, errors.Errorf("relayer submit %s must be an Ethereum address", address.name)
		}
	}
	if request.Type != TransactionSafe && request.Type != TransactionProxy {
		return nil, errors.New("relayer submit type must be SAFE or PROXY")
	}
	if request.SignatureParams == nil {
		return nil, errors.New("relayer submit signatureParams is required")
	}
	if _, err := hexutil.Decode(request.Data); err != nil {
		return nil, errors.Wrap(err, "decode relayer submit data")
	}
	if signature, err := hexutil.Decode(request.Signature); err != nil || len(signature) == 0 {
		return nil, errors.New("relayer submit signature must be nonempty hex bytes")
	}
	nonce, ok := new(big.Int).SetString(request.Nonce, 10)
	if !ok || nonce.Sign() < 0 || nonce.BitLen() > relayerUint256Bits || nonce.String() != request.Nonce {
		return nil, errors.New("relayer submit nonce must be a canonical decimal uint256")
	}
	var response SubmitResponse
	if err := c.callAuthenticated(ctx, http.MethodPost, "/submit", request, &response, true); err != nil {
		return nil, errors.Wrap(err, "submit signed relayer transaction")
	}
	return &response, nil
}

func (c *clientImpl) GetTransactions(ctx context.Context) ([]Transaction, error) {
	var response []Transaction
	if err := c.callAuthenticated(ctx, http.MethodGet, "/transactions", nil, &response, true); err != nil {
		return nil, errors.Wrap(err, "get recent relayer transactions")
	}
	return response, nil
}

func (c *clientImpl) GetNonce(ctx context.Context, request *NonceRequest) (*NonceResponse, error) {
	query, err := nonceQuery(request)
	if err != nil {
		return nil, errors.Wrap(err, "get relayer nonce")
	}
	var response NonceResponse
	if err := c.httpClient.Get(ctx, "/nonce", query, &response); err != nil {
		return nil, errors.Wrap(err, "get relayer nonce")
	}
	return &response, nil
}

func (c *clientImpl) GetRelayPayload(ctx context.Context, request *NonceRequest) (*RelayPayloadResponse, error) {
	query, err := nonceQuery(request)
	if err != nil {
		return nil, errors.Wrap(err, "get relayer payload")
	}
	var response RelayPayloadResponse
	if err := c.httpClient.Get(ctx, "/relay-payload", query, &response); err != nil {
		return nil, errors.Wrap(err, "get relayer payload")
	}
	return &response, nil
}

func nonceQuery(request *NonceRequest) (url.Values, error) {
	if request == nil || !common.IsHexAddress(request.Address) || !strings.HasPrefix(request.Address, "0x") {
		return nil, errors.New("relayer nonce signer must be an Ethereum address")
	}
	if request.Type != TransactionSafe && request.Type != TransactionProxy {
		return nil, errors.New("relayer nonce type must be SAFE or PROXY")
	}
	return url.Values{"address": {request.Address}, "type": {string(request.Type)}}, nil
}

func (c *clientImpl) GetDeployed(ctx context.Context, request *DeployedRequest) (*DeployedResponse, error) {
	if request == nil || !common.IsHexAddress(request.Address) || !strings.HasPrefix(request.Address, "0x") {
		return nil, errors.New("relayer deployment wallet must be an Ethereum address")
	}
	if request.Type != "" && request.Type != TransactionSafe && request.Type != TransactionWallet {
		return nil, errors.New("relayer deployment type must be SAFE or WALLET")
	}
	query := url.Values{"address": {request.Address}}
	if request.Type != "" {
		query.Set("type", string(request.Type))
	}
	var response DeployedResponse
	if err := c.httpClient.Get(ctx, "/deployed", query, &response); err != nil {
		return nil, errors.Wrap(err, "get relayer deployment status")
	}
	return &response, nil
}

func (c *clientImpl) GetAPIKeys(ctx context.Context) ([]APIKeyResponse, error) {
	var response []APIKeyResponse
	if err := c.callAuthenticated(ctx, http.MethodGet, "/relayer/api/keys", nil, &response, false); err != nil {
		return nil, errors.Wrap(err, "get relayer API keys")
	}
	return response, nil
}
