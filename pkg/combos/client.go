package combos

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/neor-it/polymarket-go-sdk/pkg/transport"
)

// Client defines public and authenticated Polymarket Combo operations.
type Client interface {
	// ListComboMarkets retrieves public Combo RFQ markets.
	ListComboMarkets(ctx context.Context, req *ComboMarketListRequest) (ComboMarketListResponse, error)
	// CreateComboRFQ requests a maker quote for a Combo from the Builder Gateway.
	CreateComboRFQ(ctx context.Context, req *CreateRFQRequest) (CreateRFQResponse, error)
	// AcceptComboRFQ accepts a quote by submitting a signed Exchange v3 order.
	AcceptComboRFQ(ctx context.Context, req *AcceptRFQRequest) (AcceptRFQResponse, error)
	// GetComboRFQ returns Builder RFQ lifecycle status after requester acceptance.
	GetComboRFQ(ctx context.Context, rfqID string) (GetRFQResponse, error)
	// ListUserComboPositions retrieves user Combo positions from Data API.
	ListUserComboPositions(ctx context.Context, req *UserComboPositionsRequest) (UserComboPositionsResponse, error)
	// ListUserComboActivity retrieves user Combo activity from Data API.
	ListUserComboActivity(ctx context.Context, req *UserComboActivityRequest) (UserComboActivityResponse, error)
	// ReturnCollateral starts collateral return for an eligible Combo position.
	ReturnCollateral(ctx context.Context, req *CollateralReturnRequest) (CollateralReturnResponse, error)
}

// Config configures Combo API transports and credentials.
type Config struct {
	PublicBaseURL  string
	DataBaseURL    string
	BuilderBaseURL string
	HTTPClient     transport.Doer
	AccountAuth    *AccountAuth
	BuilderAuth    *BuilderAuth
	NowUnix        func() int64
}

type clientImpl struct {
	publicTransport  *transport.Client
	dataTransport    *transport.Client
	builderTransport *transport.Client
	accountAuth      *AccountAuth
	builderAuth      *BuilderAuth
	nowUnix          func() int64
}

// NewClient creates a Combo API client.
func NewClient(config Config) Client {
	publicBaseURL := strings.TrimRight(config.PublicBaseURL, "/")
	if publicBaseURL == "" {
		publicBaseURL = DefaultPublicBaseURL
	}
	dataBaseURL := strings.TrimRight(config.DataBaseURL, "/")
	if dataBaseURL == "" {
		dataBaseURL = DefaultDataBaseURL
	}
	builderBaseURL := strings.TrimRight(config.BuilderBaseURL, "/")
	if builderBaseURL == "" {
		builderBaseURL = publicBaseURL
	}

	return &clientImpl{
		publicTransport:  transport.NewClient(config.HTTPClient, publicBaseURL),
		dataTransport:    transport.NewClient(config.HTTPClient, dataBaseURL),
		builderTransport: transport.NewClient(config.HTTPClient, builderBaseURL),
		accountAuth:      config.AccountAuth,
		builderAuth:      config.BuilderAuth,
		nowUnix:          config.NowUnix,
	}
}

func (c *clientImpl) ListComboMarkets(ctx context.Context, req *ComboMarketListRequest) (ComboMarketListResponse, error) {
	var resp ComboMarketListResponse
	query := url.Values{}
	if req != nil {
		if req.Limit > 0 {
			query.Set("limit", strconv.Itoa(req.Limit))
		}
		if req.Cursor != "" {
			query.Set("cursor", req.Cursor)
		}
		if len(req.Exclude) > 0 {
			query.Set("exclude", strings.Join(req.Exclude, ","))
		}
	}
	err := c.publicTransport.Get(ctx, comboMarketsPath, query, &resp)
	return resp, err
}

func (c *clientImpl) CreateComboRFQ(ctx context.Context, req *CreateRFQRequest) (CreateRFQResponse, error) {
	var resp CreateRFQResponse
	if req == nil {
		return resp, fmt.Errorf("create combo rfq request is required")
	}
	body, err := marshalExactBody(req)
	if err != nil {
		return resp, err
	}
	headers, err := c.authHeaders(http.MethodPost, builderRFQPath, string(body), true)
	if err != nil {
		return resp, err
	}
	err = c.builderTransport.CallWithHeaders(ctx, http.MethodPost, builderRFQPath, nil, body, &resp, headers)
	return resp, err
}

func (c *clientImpl) AcceptComboRFQ(ctx context.Context, req *AcceptRFQRequest) (AcceptRFQResponse, error) {
	var resp AcceptRFQResponse
	if req == nil {
		return resp, fmt.Errorf("accept combo rfq request is required")
	}
	rfqID := strings.TrimSpace(req.RFQID)
	if rfqID == "" {
		return resp, fmt.Errorf("rfq id is required")
	}
	path := builderRFQPath + "/" + url.PathEscape(rfqID) + "/accept"
	body, err := marshalExactBody(req)
	if err != nil {
		return resp, err
	}
	headers, err := c.authHeaders(http.MethodPost, path, string(body), true)
	if err != nil {
		return resp, err
	}
	err = c.builderTransport.CallWithHeaders(ctx, http.MethodPost, path, nil, body, &resp, headers)
	return resp, err
}

func (c *clientImpl) GetComboRFQ(ctx context.Context, rfqID string) (GetRFQResponse, error) {
	var resp GetRFQResponse
	rfqID = strings.TrimSpace(rfqID)
	if rfqID == "" {
		return resp, fmt.Errorf("rfq id is required")
	}
	path := builderRFQPath + "/" + url.PathEscape(rfqID)
	headers, err := c.authHeaders(http.MethodGet, path, "", false)
	if err != nil {
		return resp, err
	}
	err = c.builderTransport.CallWithHeaders(ctx, http.MethodGet, path, nil, nil, &resp, headers)
	return resp, err
}

func (c *clientImpl) ListUserComboPositions(ctx context.Context, req *UserComboPositionsRequest) (UserComboPositionsResponse, error) {
	var resp UserComboPositionsResponse
	query := userComboQuery(req)
	err := c.dataTransport.Get(ctx, comboPositionsPath, query, &resp)
	return resp, err
}

func (c *clientImpl) ListUserComboActivity(ctx context.Context, req *UserComboActivityRequest) (UserComboActivityResponse, error) {
	var resp UserComboActivityResponse
	query := userComboActivityQuery(req)
	err := c.dataTransport.Get(ctx, comboActivityPath, query, &resp)
	return resp, err
}

func (c *clientImpl) ReturnCollateral(ctx context.Context, req *CollateralReturnRequest) (CollateralReturnResponse, error) {
	var resp CollateralReturnResponse
	if req == nil {
		return resp, fmt.Errorf("collateral return request is required")
	}
	body, err := marshalExactBody(req)
	if err != nil {
		return resp, err
	}
	headers, err := c.authHeaders(http.MethodPost, collateralReturnPath, string(body), true)
	if err != nil {
		return resp, err
	}
	err = c.builderTransport.CallWithHeaders(ctx, http.MethodPost, collateralReturnPath, nil, body, &resp, headers)
	return resp, err
}

func (c *clientImpl) authHeaders(method, fullPath, body string, includeBuilder bool) (map[string]string, error) {
	if c.accountAuth == nil {
		return nil, fmt.Errorf("account auth is required")
	}
	timestamp := int64(0)
	if c.nowUnix != nil {
		timestamp = c.nowUnix()
	}
	accountHeaders, err := buildAccountHeaders(*c.accountAuth, method, fullPath, body, timestamp)
	if err != nil {
		return nil, fmt.Errorf("build account headers: %w", err)
	}
	headers := map[string]string{}
	mergeHeaders(headers, accountHeaders)
	if !includeBuilder {
		return headers, nil
	}
	if c.builderAuth == nil {
		return nil, fmt.Errorf("builder auth is required")
	}
	builderHeaders, err := buildBuilderHeaders(*c.builderAuth, method, fullPath, body, timestamp)
	if err != nil {
		return nil, fmt.Errorf("build builder headers: %w", err)
	}
	mergeHeaders(headers, builderHeaders)
	return headers, nil
}

func marshalExactBody(value any) ([]byte, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal combo body: %w", err)
	}
	return body, nil
}

func userComboQuery(req *UserComboPositionsRequest) url.Values {
	query := url.Values{}
	if req == nil {
		return query
	}
	setString(&query, "user", req.User)
	setString(&query, "status", req.Status)
	setString(&query, "sort", req.SortBy)
	setString(&query, "sort_dir", req.SortDirection)
	setString(&query, "market_id", req.MarketID)
	setString(&query, "updatedAfter", req.UpdatedAfter)
	setString(&query, "updatedBefore", req.UpdatedBefore)
	setString(&query, "cursor", req.Cursor)
	if req.Limit > 0 {
		query.Set("limit", strconv.Itoa(req.Limit))
	}
	if req.Offset > 0 {
		query.Set("offset", strconv.Itoa(req.Offset))
	}
	return query
}

func userComboActivityQuery(req *UserComboActivityRequest) url.Values {
	query := url.Values{}
	if req == nil {
		return query
	}
	setString(&query, "user", req.User)
	setString(&query, "type", req.Type)
	setString(&query, "market_id", req.MarketID)
	setString(&query, "updatedAfter", req.UpdatedAfter)
	setString(&query, "updatedBefore", req.UpdatedBefore)
	setString(&query, "cursor", req.Cursor)
	if req.Limit > 0 {
		query.Set("limit", strconv.Itoa(req.Limit))
	}
	if req.Offset > 0 {
		query.Set("offset", strconv.Itoa(req.Offset))
	}
	return query
}

func setString(query *url.Values, key, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	query.Set(key, value)
}
