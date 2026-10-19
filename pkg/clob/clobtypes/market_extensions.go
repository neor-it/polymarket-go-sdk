package clobtypes

// MarketByTokenResponse identifies the complete binary outcome pair.
type MarketByTokenResponse struct {
	ConditionID      string `json:"condition_id"`
	PrimaryTokenID   string `json:"primary_token_id"`
	SecondaryTokenID string `json:"secondary_token_id"`
}

// LiveActivityRequest selects conditions for the batch widget endpoint.
type LiveActivityRequest struct {
	ConditionIDs []string `json:"-"`
}

// LiveActivityItem contains minimal market widget metadata.
type LiveActivityItem struct {
	ConditionID string   `json:"condition_id"`
	ID          int64    `json:"id"`
	Question    string   `json:"question"`
	MarketSlug  string   `json:"market_slug"`
	EventSlug   string   `json:"event_slug"`
	SeriesSlug  string   `json:"series_slug"`
	Icon        string   `json:"icon"`
	Image       string   `json:"image"`
	Tags        []string `json:"tags"`
}

// LiveActivityResponse is the batch widget response.
type LiveActivityResponse []LiveActivityItem

// BatchPricesHistoryRequest selects asset histories, not condition IDs.
type BatchPricesHistoryRequest struct {
	Markets  []string `json:"markets"`
	StartTs  *float64 `json:"start_ts,omitempty"`
	EndTs    *float64 `json:"end_ts,omitempty"`
	Interval string   `json:"interval,omitempty"`
	Fidelity int      `json:"fidelity,omitempty"`
}

// BatchPricesHistoryResponse maps assets to timestamped prices.
type BatchPricesHistoryResponse struct {
	History map[string][]MarketPrice `json:"history"`
}

// MarketPrice contains a Unix timestamp and price.
type MarketPrice struct {
	T int64   `json:"t"`
	P float64 `json:"p"`
}

// BooksQueryRequest selects books for the GET endpoint.
type BooksQueryRequest struct {
	TokenIDs []string `json:"token_ids"`
}

// PutBalanceAllowanceRequest selects an on-chain permission cache refresh.
type PutBalanceAllowanceRequest struct {
	AssetType     AssetType `json:"asset_type"`
	TokenID       string    `json:"token_id,omitempty"`
	SignatureType *int      `json:"signature_type,omitempty"`
}
