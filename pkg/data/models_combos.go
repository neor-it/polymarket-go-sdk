package data

import "github.com/neor-it/polymarket-go-sdk/pkg/types"

// ComboActivity represents the documented Data API v2 ComboActivity response.
type ComboActivity struct {
	AmountUSDC       *types.Decimal `json:"amount_usdc"`
	BlockNumber      int64          `json:"block_number"`
	ComboConditionID string         `json:"combo_condition_id"`
	ComboPositionID  string         `json:"combo_position_id"`
	ID               string         `json:"id"`
	Legs             []ComboLeg     `json:"legs"`
	PayoutUSDC       *types.Decimal `json:"payout_usdc"`
	ProxyWallet      string         `json:"proxy_wallet"`
	Timestamp        int64          `json:"timestamp"`
	TransactionHash  string         `json:"transaction_hash"`
	Type             string         `json:"type"`
}

// ComboPosition represents the documented Data API v2 ComboPosition response.
type ComboPosition struct {
	ComboConditionID   string        `json:"combo_condition_id"`
	ComboPositionID    string        `json:"combo_position_id"`
	CurrentSize        types.Decimal `json:"current_size"`
	EntryAvgPriceUSDC  types.Decimal `json:"entry_avg_price_usdc"`
	EntryCostUSDC      types.Decimal `json:"entry_cost_usdc"`
	EntryFeesUSDC      types.Decimal `json:"entry_fees_usdc"`
	FirstEntryAt       string        `json:"first_entry_at"`
	FirstEntryAtMicros *int64        `json:"first_entry_at_micros"`
	GrossEntryCostUSDC types.Decimal `json:"gross_entry_cost_usdc"`
	Legs               []ComboLeg    `json:"legs"`
	LegsPending        int64         `json:"legs_pending"`
	LegsResolved       int64         `json:"legs_resolved"`
	LegsTotal          int64         `json:"legs_total"`
	OutcomeIndex       int64         `json:"outcome_index"`
	OutcomeLabel       string        `json:"outcome_label"`
	ProxyWallet        string        `json:"proxy_wallet"`
	RealizedPayoutUSDC types.Decimal `json:"realized_payout_usdc"`
	Redeemable         bool          `json:"redeemable"`
	ResolvedAt         *string       `json:"resolved_at"`
	Status             string        `json:"status"`
	UpdatedAt          string        `json:"updated_at"`
	UpdatedAtMicros    int64         `json:"updated_at_micros"`
}

// ComboLeg represents the documented Data API v2 ComboLeg response.
type ComboLeg struct {
	LegConditionID  string         `json:"leg_condition_id"`
	LegCurrentPrice types.Decimal  `json:"leg_current_price"`
	LegIndex        int64          `json:"leg_index"`
	LegOutcomeIndex int64          `json:"leg_outcome_index"`
	LegOutcomeLabel string         `json:"leg_outcome_label"`
	LegPositionID   string         `json:"leg_position_id"`
	LegResolvedAt   *string        `json:"leg_resolved_at"`
	LegStatus       string         `json:"leg_status"`
	Market          ComboLegMarket `json:"market"`
}

// ComboLegMarket represents the documented Data API v2 ComboLegMarket response.
type ComboLegMarket struct {
	Category         string         `json:"category"`
	EndDate          string         `json:"end_date"`
	Event            ComboLegEvent  `json:"event"`
	GroupItemTitle   string         `json:"group_item_title"`
	IconURL          string         `json:"icon_url"`
	ImageURL         string         `json:"image_url"`
	Line             *types.Decimal `json:"line"`
	MarketID         string         `json:"market_id"`
	Outcome          string         `json:"outcome"`
	Outcomes         []string       `json:"outcomes"`
	Question         string         `json:"question"`
	Slug             string         `json:"slug"`
	SportsMarketType string         `json:"sports_market_type"`
	Subcategory      string         `json:"subcategory"`
	Tags             []string       `json:"tags"`
	Title            string         `json:"title"`
}

// ComboLegEvent represents the documented Data API v2 ComboLegEvent response.
type ComboLegEvent struct {
	EventID    string `json:"event_id"`
	EventImage string `json:"event_image"`
	EventSlug  string `json:"event_slug"`
	EventTitle string `json:"event_title"`
}
