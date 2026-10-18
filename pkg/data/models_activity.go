package data

import "github.com/neor-it/polymarket-go-sdk/pkg/types"

// Trade represents the documented Data API v2 Trade response.
type Trade struct {
	Bio                   string        `json:"bio"`
	ConditionID           string        `json:"condition_id"`
	EventSlug             string        `json:"event_slug"`
	Icon                  string        `json:"icon"`
	Name                  string        `json:"name"`
	Outcome               string        `json:"outcome"`
	OutcomeIndex          int64         `json:"outcome_index"`
	Price                 types.Decimal `json:"price"`
	ProfileImage          string        `json:"profile_image"`
	ProfileImageOptimized string        `json:"profile_image_optimized"`
	ProxyWallet           string        `json:"proxy_wallet"`
	Pseudonym             string        `json:"pseudonym"`
	Side                  string        `json:"side"`
	Size                  types.Decimal `json:"size"`
	Slug                  string        `json:"slug"`
	Timestamp             int64         `json:"timestamp"`
	Title                 string        `json:"title"`
	TokenID               string        `json:"token_id"`
	TransactionHash       string        `json:"transaction_hash"`
}

// Activity represents the documented Data API v2 Activity response.
type Activity struct {
	Bio                   string        `json:"bio"`
	ConditionID           string        `json:"condition_id"`
	EventSlug             string        `json:"event_slug"`
	Icon                  string        `json:"icon"`
	IsCombo               bool          `json:"is_combo"`
	Name                  string        `json:"name"`
	Outcome               string        `json:"outcome"`
	OutcomeIndex          int64         `json:"outcome_index"`
	Price                 types.Decimal `json:"price"`
	ProfileImage          string        `json:"profile_image"`
	ProfileImageOptimized string        `json:"profile_image_optimized"`
	ProxyWallet           string        `json:"proxy_wallet"`
	Pseudonym             string        `json:"pseudonym"`
	Side                  string        `json:"side"`
	Size                  types.Decimal `json:"size"`
	Slug                  string        `json:"slug"`
	Timestamp             int64         `json:"timestamp"`
	Title                 string        `json:"title"`
	TokenID               string        `json:"token_id"`
	TransactionHash       string        `json:"transaction_hash"`
	Type                  string        `json:"type"`
	USDCSize              types.Decimal `json:"usdc_size"`
}
