package clob

import (
	"context"
	"fmt"
	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
	"github.com/neor-it/polymarket-go-sdk/pkg/clob/clobtypes"
	"net/http"
	"net/url"
	"strconv"
)

// PutBalanceAllowance refreshes the chain permission cache; an empty response is an acknowledgement, not a balance.
func (c *clientImpl) PutBalanceAllowance(ctx context.Context, req *clobtypes.PutBalanceAllowanceRequest) error {
	if req == nil {
		return fmt.Errorf("allowance update request is required")
	}
	switch req.AssetType {
	case clobtypes.AssetTypeCollateral:
	case clobtypes.AssetTypeConditional, clobtypes.AssetTypeConditionalV2:
		if req.TokenID == "" {
			return fmt.Errorf("conditional allowance update requires a token ID")
		}
	default:
		return fmt.Errorf("unsupported allowance asset type %q", req.AssetType)
	}
	sig := int(c.signatureType)
	if req.SignatureType != nil {
		sig = *req.SignatureType
	}
	if sig < int(auth.SignatureEOA) || sig > int(auth.SignaturePoly1271) {
		return fmt.Errorf("unsupported allowance signature type %d", sig)
	}
	q := url.Values{"asset_type": {string(req.AssetType)}, "signature_type": {strconv.Itoa(sig)}}
	if req.TokenID != "" {
		q.Set("token_id", req.TokenID)
	}
	return mapError(c.httpClient.Call(ctx, http.MethodPut, "/balance-allowance", q, nil, nil, nil))
}
