package data

import "github.com/neor-it/polymarket-go-sdk/pkg/types"

// Position represents the documented Data API v2 Position response.
type Position struct {
	Archived           bool          `json:"archived"`
	AvgPrice           types.Decimal `json:"avg_price"`
	ConditionID        string        `json:"condition_id"`
	CurrentPrice       types.Decimal `json:"current_price"`
	CurrentSize        types.Decimal `json:"current_size"`
	CurrentValue       types.Decimal `json:"current_value"`
	EndDate            string        `json:"end_date"`
	EntryCostUSDC      types.Decimal `json:"entry_cost_usdc"`
	EntryFeesUSDC      types.Decimal `json:"entry_fees_usdc"`
	EventID            string        `json:"event_id"`
	EventSlug          string        `json:"event_slug"`
	FirstEntryAt       int64         `json:"first_entry_at"`
	Icon               string        `json:"icon"`
	LastEventAt        int64         `json:"last_event_at"`
	Mergeable          bool          `json:"mergeable"`
	Name               string        `json:"name"`
	NegativeRisk       bool          `json:"negative_risk"`
	OppositeOutcome    string        `json:"opposite_outcome"`
	OppositeTokenID    string        `json:"opposite_token_id"`
	Outcome            string        `json:"outcome"`
	OutcomeIndex       int64         `json:"outcome_index"`
	PercentPnl         types.Decimal `json:"percent_pnl"`
	PercentRealizedPnl types.Decimal `json:"percent_realized_pnl"`
	ProfileImage       string        `json:"profile_image"`
	ProxyWallet        string        `json:"proxy_wallet"`
	RealizedPnl        types.Decimal `json:"realized_pnl"`
	Redeemable         bool          `json:"redeemable"`
	Slug               string        `json:"slug"`
	Status             string        `json:"status"`
	Title              string        `json:"title"`
	TokenID            string        `json:"token_id"`
	TotalCostUSDC      types.Decimal `json:"total_cost_usdc"`
	TotalPnl           types.Decimal `json:"total_pnl"`
	TotalSize          types.Decimal `json:"total_size"`
	UnrealizedPnl      types.Decimal `json:"unrealized_pnl"`
	Verified           bool          `json:"verified"`
}

// Holder represents the documented Data API v2 Holder response.
type Holder struct {
	Amount                types.Decimal  `json:"amount"`
	AvgPrice              *types.Decimal `json:"avg_price"`
	Bio                   string         `json:"bio"`
	CurrentPrice          *types.Decimal `json:"current_price"`
	CurrentValue          *types.Decimal `json:"current_value"`
	DisplayUsernamePublic bool           `json:"display_username_public"`
	EntryCostUSDC         *types.Decimal `json:"entry_cost_usdc"`
	Name                  string         `json:"name"`
	OutcomeIndex          int64          `json:"outcome_index"`
	ProfileImage          string         `json:"profile_image"`
	ProfileImageOptimized string         `json:"profile_image_optimized"`
	ProxyWallet           string         `json:"proxy_wallet"`
	Pseudonym             string         `json:"pseudonym"`
	RealizedPnl           *types.Decimal `json:"realized_pnl"`
	TokenID               string         `json:"token_id"`
	TotalPnl              *types.Decimal `json:"total_pnl"`
	UnrealizedPnl         *types.Decimal `json:"unrealized_pnl"`
	Verified              bool           `json:"verified"`
}

// MetaHolder represents the documented Data API v2 MetaHolder response.
type MetaHolder struct {
	Holders []Holder `json:"holders"`
	TokenID string   `json:"token_id"`
}

// PortfolioValue represents the documented Data API v2 PortfolioValue response.
type PortfolioValue struct {
	ProxyWallet string        `json:"proxy_wallet"`
	Value       types.Decimal `json:"value"`
}

// OpenInterest represents the documented Data API v2 OpenInterest response.
type OpenInterest struct {
	ConditionID string        `json:"condition_id"`
	Value       types.Decimal `json:"value"`
}

// LiveVolume represents the documented Data API v2 LiveVolume response.
type LiveVolume struct {
	Conditions       []ConditionVolume `json:"conditions"`
	TakerVolumeTotal types.Decimal     `json:"taker_volume_total"`
}

// ConditionVolume represents the documented Data API v2 ConditionVolume response.
type ConditionVolume struct {
	ConditionID string        `json:"condition_id"`
	TakerVolume types.Decimal `json:"taker_volume"`
}
