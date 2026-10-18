package data

import "github.com/neor-it/polymarket-go-sdk/pkg/types"

// BuilderStanding represents the documented Data API v2 BuilderStanding response.
type BuilderStanding struct {
	ActiveUsers  int64         `json:"active_users"`
	BuilderCode  string        `json:"builder_code"`
	BuilderName  string        `json:"builder_name"`
	ProfileImage string        `json:"profile_image"`
	Rank         int64         `json:"rank"`
	Verified     bool          `json:"verified"`
	Volume       types.Decimal `json:"volume"`
}

// BuilderVolumePoint represents the documented Data API v2 BuilderVolumePoint response.
type BuilderVolumePoint struct {
	ActiveUsers  int64         `json:"active_users"`
	BuilderCode  string        `json:"builder_code"`
	BuilderName  string        `json:"builder_name"`
	Date         string        `json:"date"`
	ProfileImage string        `json:"profile_image"`
	Rank         int64         `json:"rank"`
	Verified     bool          `json:"verified"`
	Volume       types.Decimal `json:"volume"`
}

// LeaderboardEntry represents the documented Data API v2 LeaderboardEntry response.
type LeaderboardEntry struct {
	Pnl          types.Decimal `json:"pnl"`
	ProfileImage string        `json:"profile_image"`
	Rank         int64         `json:"rank"`
	UserID       string        `json:"user_id"`
	UserName     string        `json:"user_name"`
	Verified     bool          `json:"verified"`
	Volume       types.Decimal `json:"volume"`
	XUsername    string        `json:"x_username"`
}

// LeaderboardUserEntry represents the documented Data API v2 LeaderboardUserEntry response.
type LeaderboardUserEntry struct {
	Pnl          types.Decimal `json:"pnl"`
	ProfileImage string        `json:"profile_image"`
	RankPnl      *int64        `json:"rank_pnl"`
	RankVolume   *int64        `json:"rank_volume"`
	UserID       string        `json:"user_id"`
	UserName     string        `json:"user_name"`
	Verified     bool          `json:"verified"`
	Volume       types.Decimal `json:"volume"`
	XUsername    string        `json:"x_username"`
}

// BiggestWinner represents the documented Data API v2 BiggestWinner response.
type BiggestWinner struct {
	ConditionID  string        `json:"condition_id"`
	EventID      int64         `json:"event_id"`
	EventSlug    string        `json:"event_slug"`
	EventTitle   string        `json:"event_title"`
	FinalValue   types.Decimal `json:"final_value"`
	InitialValue types.Decimal `json:"initial_value"`
	Kind         string        `json:"kind"`
	Pnl          types.Decimal `json:"pnl"`
	PositionID   string        `json:"position_id"`
	ProfileImage string        `json:"profile_image"`
	ResolvedAt   int64         `json:"resolved_at"`
	UserID       string        `json:"user_id"`
	UserName     string        `json:"user_name"`
	WinRank      int64         `json:"win_rank"`
}

// UserStats represents the documented Data API v2 UserStats response.
type UserStats struct {
	AllTimePnl  *UserPnlPoint `json:"all_time_pnl"`
	BiggestWin  types.Decimal `json:"biggest_win"`
	JoinDate    *int64        `json:"join_date"`
	ProxyWallet string        `json:"proxy_wallet"`
	Trades      int64         `json:"trades"`
	Views       int64         `json:"views"`
}

// UserPnlPoint represents the documented Data API v2 UserPnlPoint response.
type UserPnlPoint struct {
	CashflowNet       *types.Decimal `json:"cashflow_net"`
	Deposits          *types.Decimal `json:"deposits"`
	EconomicPnl       *types.Decimal `json:"economic_pnl"`
	Fees              *types.Decimal `json:"fees"`
	FeesPaid          *types.Decimal `json:"fees_paid"`
	FeesRefunded      *types.Decimal `json:"fees_refunded"`
	MakerRebate       *types.Decimal `json:"maker_rebate"`
	PositionPnl       *types.Decimal `json:"position_pnl"`
	RealizedComboPnl  types.Decimal  `json:"realized_combo_pnl"`
	RealizedLpPnl     types.Decimal  `json:"realized_lp_pnl"`
	RealizedMarketPnl types.Decimal  `json:"realized_market_pnl"`
	RealizedPnl       types.Decimal  `json:"realized_pnl"`
	ReferralIncome    *types.Decimal `json:"referral_income"`
	RewardIncome      *types.Decimal `json:"reward_income"`
	SettledPnl        *types.Decimal `json:"settled_pnl"`
	SourceBlock       int64          `json:"source_block"`
	SponsoredIncome   *types.Decimal `json:"sponsored_income"`
	TakerRebate       *types.Decimal `json:"taker_rebate"`
	Timestamp         int64          `json:"timestamp"`
	TradeCount        int64          `json:"trade_count"`
	TradePnl          *types.Decimal `json:"trade_pnl"`
	UnrealizedPnl     *types.Decimal `json:"unrealized_pnl"`
	Volume            types.Decimal  `json:"volume"`
	VolumeUSDC        types.Decimal  `json:"volume_usdc"`
	WalletIncome      *types.Decimal `json:"wallet_income"`
	Withdrawals       *types.Decimal `json:"withdrawals"`
	YieldIncome       *types.Decimal `json:"yield_income"`
}

// UserPnlSeries represents the documented Data API v2 UserPnlSeries response.
type UserPnlSeries struct {
	Fidelity       string         `json:"fidelity"`
	Interval       string         `json:"interval"`
	Points         []UserPnlPoint `json:"points"`
	ProxyWallet    string         `json:"proxy_wallet"`
	SourceFidelity string         `json:"source_fidelity"`
}

// UserVolume represents the documented Data API v2 UserVolume response.
type UserVolume struct {
	TradeCount int64         `json:"trade_count"`
	Volume     types.Decimal `json:"volume"`
	VolumeUSDC types.Decimal `json:"volume_usdc"`
}

// PricePoint represents the documented Data API v2 PricePoint response.
type PricePoint struct {
	Price             types.Decimal `json:"price"`
	ResolutionSeconds int64         `json:"resolution_seconds"`
	Timestamp         int64         `json:"timestamp"`
}
