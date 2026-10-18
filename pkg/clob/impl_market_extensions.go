package clob

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/neor-it/polymarket-go-sdk/pkg/clob/clobtypes"
)

const maxBatchHistoryAssets = 20

// MarketByToken calls the corresponding typed API endpoint and preserves transport errors.
func (c *clientImpl) MarketByToken(ctx context.Context, tokenID string) (clobtypes.MarketByTokenResponse, error) {
	var resp clobtypes.MarketByTokenResponse
	if strings.TrimSpace(tokenID) == "" {
		return resp, fmt.Errorf("token ID is required")
	}
	err := c.httpClient.Get(ctx, "/markets-by-token/"+url.PathEscape(tokenID), nil, &resp)
	return resp, mapError(err)
}

// MarketsLiveActivity calls the corresponding typed API endpoint and preserves transport errors.
func (c *clientImpl) MarketsLiveActivity(ctx context.Context, req *clobtypes.LiveActivityRequest) (clobtypes.LiveActivityResponse, error) {
	var resp clobtypes.LiveActivityResponse
	if req == nil || len(req.ConditionIDs) == 0 {
		return resp, fmt.Errorf("condition IDs are required")
	}
	err := c.httpClient.Post(ctx, "/markets/live-activity", req.ConditionIDs, &resp)
	return resp, mapError(err)
}

// MarketLiveActivity calls the corresponding typed API endpoint and preserves transport errors.
func (c *clientImpl) MarketLiveActivity(ctx context.Context, conditionID string) (clobtypes.LiveActivityItem, error) {
	var resp clobtypes.LiveActivityItem
	if strings.TrimSpace(conditionID) == "" {
		return resp, fmt.Errorf("condition ID is required")
	}
	err := c.httpClient.Get(ctx, "/markets/live-activity/"+url.PathEscape(conditionID), nil, &resp)
	return resp, mapError(err)
}

// BatchPricesHistory calls the corresponding typed API endpoint and preserves transport errors.
func (c *clientImpl) BatchPricesHistory(ctx context.Context, req *clobtypes.BatchPricesHistoryRequest) (clobtypes.BatchPricesHistoryResponse, error) {
	var resp clobtypes.BatchPricesHistoryResponse
	if req == nil || len(req.Markets) == 0 || len(req.Markets) > maxBatchHistoryAssets {
		return resp, fmt.Errorf("history requires 1 to %d asset IDs", maxBatchHistoryAssets)
	}
	err := c.httpClient.Post(ctx, "/batch-prices-history", req, &resp)
	return resp, mapError(err)
}

// OrderBooksQuery calls the corresponding typed API endpoint and preserves transport errors.
func (c *clientImpl) OrderBooksQuery(ctx context.Context, req *clobtypes.BooksQueryRequest) (clobtypes.OrderBooksResponse, error) {
	var resp clobtypes.OrderBooksResponse
	if req == nil || len(req.TokenIDs) == 0 {
		return resp, fmt.Errorf("token IDs are required")
	}
	q := url.Values{"token_ids": {strings.Join(req.TokenIDs, ",")}}
	err := c.httpClient.Get(ctx, "/books", q, &resp)
	return resp, mapError(err)
}

// NegRiskByPath calls the corresponding typed API endpoint and preserves transport errors.
func (c *clientImpl) NegRiskByPath(ctx context.Context, tokenID string) (clobtypes.NegRiskResponse, error) {
	var resp clobtypes.NegRiskResponse
	if strings.TrimSpace(tokenID) == "" {
		return resp, fmt.Errorf("token ID is required")
	}
	err := c.httpClient.Get(ctx, "/neg-risk/"+url.PathEscape(tokenID), nil, &resp)
	return resp, mapError(err)
}
