package relayer

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

const (
	// HeaderRelayerAPIKey authenticates the Relayer API key mechanism.
	HeaderRelayerAPIKey = "RELAYER_API_KEY"
	// HeaderRelayerAPIKeyAddress identifies the address owning the relayer key.
	HeaderRelayerAPIKeyAddress = "RELAYER_API_KEY_ADDRESS"
)

func (c *clientImpl) WithRelayerCredentials(credentials *RelayerCredentials) Client {
	next := *c
	next.relayerCredentials = nil
	if credentials != nil {
		copied := *credentials
		next.relayerCredentials = &copied
	}
	return &next
}

func (c *clientImpl) callAuthenticated(ctx context.Context, method, path string, body interface{}, dest interface{}, allowBuilder bool) error {
	var serialized string
	var wireBody interface{}
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return errors.Wrap(err, "encode authenticated relayer payload")
		}
		serialized = string(raw)
		wireBody = json.RawMessage(raw)
	}
	headers := make(map[string]string)
	if credentials := c.relayerCredentials; credentials != nil {
		if strings.TrimSpace(credentials.APIKey) == "" {
			return errors.New("relayer API key is required")
		}
		if !common.IsHexAddress(credentials.Address) || !strings.HasPrefix(credentials.Address, "0x") {
			return errors.New("relayer API key owner must be an Ethereum address")
		}
		headers[HeaderRelayerAPIKey] = credentials.APIKey
		headers[HeaderRelayerAPIKeyAddress] = credentials.Address
	} else {
		if !allowBuilder || c.builderCfg == nil || !c.builderCfg.UseHMACBuilderHeaders() {
			return errors.New("authenticated relayer request requires relayer API credentials or supported builder credentials")
		}
		builderHeaders, err := c.builderCfg.Headers(ctx, method, path, &serialized, 0)
		if err != nil {
			return errors.Wrap(err, "sign authenticated relayer request")
		}
		for key, values := range builderHeaders {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}
	}
	if err := c.httpClient.CallWithHeaders(ctx, method, path, nil, wireBody, dest, headers); err != nil {
		return errors.Wrap(err, "send authenticated relayer request")
	}
	return nil
}
