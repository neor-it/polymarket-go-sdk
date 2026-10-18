package data

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/neor-it/polymarket-go-sdk/pkg/transport"
	"github.com/pkg/errors"
	"net/url"
)

// BaseURL is the public Data API origin.
const BaseURL = "https://data-api.polymarket.com"
const (
	maxConditions         = 20
	maxPageLimit          = 1000
	maxPriceHistoryLimit  = 10000
	maxBuilderVolumeLimit = 90
)

type clientImpl struct{ httpClient *transport.Client }

// NewClient creates a v2 client using the supplied transport.
func NewClient(httpClient *transport.Client) Client {
	if httpClient == nil {
		httpClient = transport.NewClient(nil, BaseURL)
	}
	return &clientImpl{httpClient: httpClient}
}

// Activity reads the corresponding Data API v2 endpoint.
func (c *clientImpl) Activity(ctx context.Context, req *ActivityRequest) (ActivityResponse, error) {
	var resp ActivityResponse
	if req == nil {
		return resp, errors.Errorf("activity request is required")
	}
	q := url.Values{}
	if req.Limit != nil && (*req.Limit < 0 || *req.Limit > maxPageLimit) {
		return resp, errors.Errorf("activity limit must be between 0 and 1000")
	}
	if err := applyMarketFilter(q, req.Filter); err != nil {
		return resp, errors.Wrap(err, "activity filter")
	}
	if req.User != (common.Address{}) {
		q.Set("user", req.User.Hex())
	}
	addInt(q, "limit", req.Limit)
	addString(q, "cursor", req.Cursor)
	addActivityTypes(q, req.ActivityTypes)
	addString(q, "side", req.Side)
	addInt64(q, "start", req.Start)
	addInt64(q, "end", req.End)
	addString(q, "sort_by", req.SortBy)
	addString(q, "sort_direction", req.SortDirection)
	addBool(q, "exclude_deposits_withdrawals", req.ExcludeDepositsWithdrawals)
	if err := c.httpClient.Get(ctx, "/v2/activity", q, &resp); err != nil {
		return resp, errors.Wrap(err, "get /v2/activity")
	}
	return resp, nil
}

// Approvals reads the corresponding Data API v2 endpoint.
func (c *clientImpl) Approvals(ctx context.Context, req *ApprovalsRequest) (ApprovalsResponse, error) {
	var resp ApprovalsResponse
	if req == nil {
		return resp, errors.Errorf("approvals request is required")
	}
	q := url.Values{}
	if req.User != (common.Address{}) {
		q.Set("user", req.User.Hex())
	}
	if err := c.httpClient.Get(ctx, "/v2/approvals", q, &resp); err != nil {
		return resp, errors.Wrap(err, "get /v2/approvals")
	}
	return resp, nil
}

// BiggestWinners reads the corresponding Data API v2 endpoint.
func (c *clientImpl) BiggestWinners(ctx context.Context, req *BiggestWinnersRequest) (BiggestWinnersResponse, error) {
	var resp BiggestWinnersResponse
	if req == nil {
		return resp, errors.Errorf("biggestwinners request is required")
	}
	q := url.Values{}
	if req.Limit != nil && (*req.Limit < 0 || *req.Limit > maxPageLimit) {
		return resp, errors.Errorf("biggestwinners limit must be between 0 and 1000")
	}
	addString(q, "time_period", req.TimePeriod)
	addString(q, "category", req.Category)
	addInt(q, "limit", req.Limit)
	addString(q, "cursor", req.Cursor)
	if err := c.httpClient.Get(ctx, "/v2/biggest-winners", q, &resp); err != nil {
		return resp, errors.Wrap(err, "get /v2/biggest-winners")
	}
	return resp, nil
}

// BuildersLeaderboard reads the corresponding Data API v2 endpoint.
func (c *clientImpl) BuildersLeaderboard(ctx context.Context, req *BuildersLeaderboardRequest) (BuildersLeaderboardResponse, error) {
	var resp BuildersLeaderboardResponse
	if req == nil {
		return resp, errors.Errorf("buildersleaderboard request is required")
	}
	q := url.Values{}
	if req.Limit != nil && (*req.Limit < 0 || *req.Limit > maxPageLimit) {
		return resp, errors.Errorf("buildersleaderboard limit must be between 0 and 1000")
	}
	addString(q, "time_period", req.TimePeriod)
	addInt(q, "limit", req.Limit)
	addString(q, "cursor", req.Cursor)
	if err := c.httpClient.Get(ctx, "/v2/builders/leaderboard", q, &resp); err != nil {
		return resp, errors.Wrap(err, "get /v2/builders/leaderboard")
	}
	return resp, nil
}

// BuildersVolume reads the corresponding Data API v2 endpoint.
func (c *clientImpl) BuildersVolume(ctx context.Context, req *BuildersVolumeRequest) (BuildersVolumeResponse, error) {
	var resp BuildersVolumeResponse
	if req == nil {
		return resp, errors.Errorf("buildersvolume request is required")
	}
	q := url.Values{}
	if req.Limit != nil && (*req.Limit < 0 || *req.Limit > maxBuilderVolumeLimit) {
		return resp, errors.Errorf("buildersvolume limit must be between 0 and 90")
	}
	addString(q, "interval", req.Interval)
	addInt(q, "limit", req.Limit)
	if err := c.httpClient.Get(ctx, "/v2/builders/volume", q, &resp); err != nil {
		return resp, errors.Wrap(err, "get /v2/builders/volume")
	}
	return resp, nil
}

// Holders reads the corresponding Data API v2 endpoint.
func (c *clientImpl) Holders(ctx context.Context, req *HoldersRequest) (HoldersResponse, error) {
	var resp HoldersResponse
	if req == nil {
		return resp, errors.Errorf("holders request is required")
	}
	q := url.Values{}
	if req.Limit != nil && (*req.Limit < 0 || *req.Limit > maxPageLimit) {
		return resp, errors.Errorf("holders limit must be between 0 and 1000")
	}
	if err := addConditions(q, req.Conditions); err != nil {
		return resp, errors.Wrap(err, "holders conditions")
	}
	addInt(q, "limit", req.Limit)
	addString(q, "cursor", req.Cursor)
	addDecimal(q, "min_balance", req.MinBalance)
	addBool(q, "include_pnl", req.IncludePnl)
	if err := c.httpClient.Get(ctx, "/v2/holders", q, &resp); err != nil {
		return resp, errors.Wrap(err, "get /v2/holders")
	}
	return resp, nil
}

// Leaderboard reads the corresponding Data API v2 endpoint.
func (c *clientImpl) Leaderboard(ctx context.Context, req *LeaderboardRequest) (LeaderboardResponse, error) {
	var resp LeaderboardResponse
	if req == nil {
		return resp, errors.Errorf("leaderboard request is required")
	}
	q := url.Values{}
	if req.Limit != nil && (*req.Limit < 0 || *req.Limit > maxPageLimit) {
		return resp, errors.Errorf("leaderboard limit must be between 0 and 1000")
	}
	addString(q, "time_period", req.TimePeriod)
	addString(q, "category", req.Category)
	addString(q, "sort_by", req.SortBy)
	addInt(q, "limit", req.Limit)
	addString(q, "cursor", req.Cursor)
	if err := c.httpClient.Get(ctx, "/v2/leaderboard", q, &resp); err != nil {
		return resp, errors.Wrap(err, "get /v2/leaderboard")
	}
	return resp, nil
}

// LiveVolume reads the corresponding Data API v2 endpoint.
func (c *clientImpl) LiveVolume(ctx context.Context, req *LiveVolumeRequest) (LiveVolumeResponse, error) {
	var resp LiveVolumeResponse
	if req == nil {
		return resp, errors.Errorf("livevolume request is required")
	}
	q := url.Values{}
	addEventIDs(q, req.EventIDs)
	if err := c.httpClient.Get(ctx, "/v2/live-volume", q, &resp); err != nil {
		return resp, errors.Wrap(err, "get /v2/live-volume")
	}
	return resp, nil
}

// OpenInterest reads the corresponding Data API v2 endpoint.
func (c *clientImpl) OpenInterest(ctx context.Context, req *OpenInterestRequest) (OpenInterestResponse, error) {
	var resp OpenInterestResponse
	if req == nil {
		return resp, errors.Errorf("openinterest request is required")
	}
	q := url.Values{}
	if err := addConditions(q, req.Conditions); err != nil {
		return resp, errors.Wrap(err, "openinterest conditions")
	}
	if err := c.httpClient.Get(ctx, "/v2/oi", q, &resp); err != nil {
		return resp, errors.Wrap(err, "get /v2/oi")
	}
	return resp, nil
}

// Positions reads the corresponding Data API v2 endpoint.
func (c *clientImpl) Positions(ctx context.Context, req *PositionsRequest) (PositionsResponse, error) {
	var resp PositionsResponse
	if req == nil {
		return resp, errors.Errorf("positions request is required")
	}
	q := url.Values{}
	if req.Limit != nil && (*req.Limit < 0 || *req.Limit > maxPageLimit) {
		return resp, errors.Errorf("positions limit must be between 0 and 1000")
	}
	if err := applyMarketFilter(q, req.Filter); err != nil {
		return resp, errors.Wrap(err, "positions filter")
	}
	if err := applyTradeFilter(q, req.TradeFilter); err != nil {
		return resp, errors.Wrap(err, "positions quantity filter")
	}
	if req.User != (common.Address{}) {
		q.Set("user", req.User.Hex())
	}
	addInt(q, "limit", req.Limit)
	addString(q, "cursor", req.Cursor)
	addString(q, "status", req.Status)
	addString(q, "title", req.Title)
	addBool(q, "include_archived", req.IncludeArchived)
	addString(q, "sort_by", req.SortBy)
	addInt64(q, "start", req.Start)
	addInt64(q, "end", req.End)
	addString(q, "sort_direction", req.SortDirection)
	if err := c.httpClient.Get(ctx, "/v2/positions", q, &resp); err != nil {
		return resp, errors.Wrap(err, "get /v2/positions")
	}
	return resp, nil
}

// PricesHistory reads the corresponding Data API v2 endpoint.
func (c *clientImpl) PricesHistory(ctx context.Context, req *PricesHistoryRequest) (PricesHistoryResponse, error) {
	var resp PricesHistoryResponse
	if req == nil {
		return resp, errors.Errorf("priceshistory request is required")
	}
	q := url.Values{}
	if req.Limit != nil && (*req.Limit < 0 || *req.Limit > maxPriceHistoryLimit) {
		return resp, errors.Errorf("priceshistory limit must be between 0 and 10000")
	}
	addString(q, "token_id", req.TokenID)
	addInt64(q, "start", req.Start)
	addInt64(q, "end", req.End)
	addString(q, "interval", req.Interval)
	addInt64(q, "bucket_seconds", req.BucketSeconds)
	addInt64(q, "as_of", req.AsOf)
	addInt(q, "limit", req.Limit)
	addString(q, "cursor", req.Cursor)
	if err := c.httpClient.Get(ctx, "/v2/prices-history", q, &resp); err != nil {
		return resp, errors.Wrap(err, "get /v2/prices-history")
	}
	return resp, nil
}

// Resolutions reads the corresponding Data API v2 endpoint.
func (c *clientImpl) Resolutions(ctx context.Context, req *ResolutionsRequest) (ResolutionsResponse, error) {
	var resp ResolutionsResponse
	if req == nil {
		return resp, errors.Errorf("resolutions request is required")
	}
	q := url.Values{}
	addString(q, "question_id", req.QuestionID)
	if err := addConditions(q, req.Conditions); err != nil {
		return resp, errors.Wrap(err, "resolutions conditions")
	}
	addEventIDs(q, req.EventIDs)
	if err := c.httpClient.Get(ctx, "/v2/resolutions", q, &resp); err != nil {
		return resp, errors.Wrap(err, "get /v2/resolutions")
	}
	return resp, nil
}

// Status reads the corresponding Data API v2 endpoint.
func (c *clientImpl) Status(ctx context.Context) (StatusResponse, error) {
	var resp StatusResponse
	q := url.Values{}

	if err := c.httpClient.Get(ctx, "/v2/status", q, &resp); err != nil {
		return resp, errors.Wrap(err, "get /v2/status")
	}
	return resp, nil
}

// Tokens reads the corresponding Data API v2 endpoint.
func (c *clientImpl) Tokens(ctx context.Context, req *TokensRequest) (TokensResponse, error) {
	var resp TokensResponse
	if req == nil {
		return resp, errors.Errorf("tokens request is required")
	}
	q := url.Values{}
	addString(q, "token_id", req.TokenID)
	if err := addConditions(q, req.Conditions); err != nil {
		return resp, errors.Wrap(err, "tokens conditions")
	}
	if err := c.httpClient.Get(ctx, "/v2/tokens", q, &resp); err != nil {
		return resp, errors.Wrap(err, "get /v2/tokens")
	}
	return resp, nil
}

// Trades reads the corresponding Data API v2 endpoint.
func (c *clientImpl) Trades(ctx context.Context, req *TradesRequest) (TradesResponse, error) {
	var resp TradesResponse
	if req == nil {
		return resp, errors.Errorf("trades request is required")
	}
	q := url.Values{}
	if req.Limit != nil && (*req.Limit < 0 || *req.Limit > maxPageLimit) {
		return resp, errors.Errorf("trades limit must be between 0 and 1000")
	}
	if err := applyMarketFilter(q, req.Filter); err != nil {
		return resp, errors.Wrap(err, "trades filter")
	}
	if err := applyTradeFilter(q, req.TradeFilter); err != nil {
		return resp, errors.Wrap(err, "trades quantity filter")
	}
	addAddress(q, "user", req.User)
	addInt(q, "limit", req.Limit)
	addString(q, "cursor", req.Cursor)
	addBool(q, "taker_only", req.TakerOnly)
	addInt64(q, "start", req.Start)
	addInt64(q, "end", req.End)
	addString(q, "side", req.Side)
	if err := c.httpClient.Get(ctx, "/v2/trades", q, &resp); err != nil {
		return resp, errors.Wrap(err, "get /v2/trades")
	}
	return resp, nil
}

// UserPnl reads the corresponding Data API v2 endpoint.
func (c *clientImpl) UserPnl(ctx context.Context, req *UserPnlRequest) (UserPnlResponse, error) {
	var resp UserPnlResponse
	if req == nil {
		return resp, errors.Errorf("userpnl request is required")
	}
	q := url.Values{}
	if req.User != (common.Address{}) {
		q.Set("user", req.User.Hex())
	}
	addString(q, "interval", req.Interval)
	addString(q, "fidelity", req.Fidelity)
	if err := c.httpClient.Get(ctx, "/v2/user-pnl", q, &resp); err != nil {
		return resp, errors.Wrap(err, "get /v2/user-pnl")
	}
	return resp, nil
}

// UserStats reads the corresponding Data API v2 endpoint.
func (c *clientImpl) UserStats(ctx context.Context, req *UserStatsRequest) (UserStatsResponse, error) {
	var resp UserStatsResponse
	if req == nil {
		return resp, errors.Errorf("userstats request is required")
	}
	q := url.Values{}
	if req.User != (common.Address{}) {
		q.Set("user", req.User.Hex())
	}
	if err := c.httpClient.Get(ctx, "/v2/user-stats", q, &resp); err != nil {
		return resp, errors.Wrap(err, "get /v2/user-stats")
	}
	return resp, nil
}

// UserVolume reads the corresponding Data API v2 endpoint.
func (c *clientImpl) UserVolume(ctx context.Context, req *UserVolumeRequest) (UserVolumeResponse, error) {
	var resp UserVolumeResponse
	if req == nil {
		return resp, errors.Errorf("uservolume request is required")
	}
	q := url.Values{}
	if req.User != (common.Address{}) {
		q.Set("user", req.User.Hex())
	}
	addInt64(q, "start", req.Start)
	addInt64(q, "end", req.End)
	if err := c.httpClient.Get(ctx, "/v2/user-volume", q, &resp); err != nil {
		return resp, errors.Wrap(err, "get /v2/user-volume")
	}
	return resp, nil
}

// Value reads the corresponding Data API v2 endpoint.
func (c *clientImpl) Value(ctx context.Context, req *ValueRequest) (ValueResponse, error) {
	var resp ValueResponse
	if req == nil {
		return resp, errors.Errorf("value request is required")
	}
	q := url.Values{}
	if req.User != (common.Address{}) {
		q.Set("user", req.User.Hex())
	}
	if err := addConditions(q, req.Conditions); err != nil {
		return resp, errors.Wrap(err, "value conditions")
	}
	if err := c.httpClient.Get(ctx, "/v2/value", q, &resp); err != nil {
		return resp, errors.Wrap(err, "get /v2/value")
	}
	return resp, nil
}

// LeaderboardUser reads the corresponding Data API v2 endpoint.
func (c *clientImpl) LeaderboardUser(ctx context.Context, req *LeaderboardUserRequest) (LeaderboardUserResponse, error) {
	var resp LeaderboardUserResponse
	if req == nil || req.User == (common.Address{}) {
		return resp, errors.New("leaderboard user address is required")
	}
	q := url.Values{"user": {req.User.Hex()}}
	addString(q, "time_period", req.TimePeriod)
	addString(q, "category", req.Category)
	if err := c.httpClient.Get(ctx, "/v2/leaderboard", q, &resp); err != nil {
		return resp, errors.Wrap(err, "get user leaderboard")
	}
	return resp, nil
}
