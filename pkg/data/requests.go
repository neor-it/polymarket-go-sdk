package data

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/neor-it/polymarket-go-sdk/pkg/types"
)

// ActivityRequest configures GET /v2/activity.
type ActivityRequest struct {
	Filter                     *MarketFilter
	User                       common.Address
	Limit                      *int
	Cursor                     *string
	ActivityTypes              []ActivityType
	Side                       *Side
	Start                      *int64
	End                        *int64
	SortBy                     *ActivitySortBy
	SortDirection              *SortDirection
	ExcludeDepositsWithdrawals *bool
}

// ActivityResponse is the documented GET /v2/activity envelope.
type ActivityResponse = Page[Activity]

// ApprovalsRequest configures GET /v2/approvals.
type ApprovalsRequest struct {
	User common.Address
}

// ApprovalsResponse is the documented GET /v2/approvals envelope.
type ApprovalsResponse = Envelope[Approvals]

// BiggestWinnersRequest configures GET /v2/biggest-winners.
type BiggestWinnersRequest struct {
	TimePeriod *TimePeriod
	Category   *LeaderboardCategory
	Limit      *int
	Cursor     *string
}

// BiggestWinnersResponse is the documented GET /v2/biggest-winners envelope.
type BiggestWinnersResponse = Page[BiggestWinner]

// BuildersLeaderboardRequest configures GET /v2/builders/leaderboard.
type BuildersLeaderboardRequest struct {
	TimePeriod *TimePeriod
	Limit      *int
	Cursor     *string
}

// BuildersLeaderboardResponse is the documented GET /v2/builders/leaderboard envelope.
type BuildersLeaderboardResponse = Page[BuilderStanding]

// BuildersVolumeRequest configures GET /v2/builders/volume.
type BuildersVolumeRequest struct {
	Interval *string
	Limit    *int
}

// BuildersVolumeResponse is the documented GET /v2/builders/volume envelope.
type BuildersVolumeResponse = Envelope[[]BuilderVolumePoint]

// HoldersRequest configures GET /v2/holders.
type HoldersRequest struct {
	Conditions []common.Hash
	Limit      *int
	Cursor     *string
	MinBalance *types.Decimal
	IncludePnl *bool
}

// HoldersResponse is the documented GET /v2/holders envelope.
type HoldersResponse = Page[MetaHolder]

// LeaderboardRequest configures GET /v2/leaderboard.
type LeaderboardRequest struct {
	TimePeriod *TimePeriod
	Category   *LeaderboardCategory
	SortBy     *LeaderboardOrderBy
	Limit      *int
	Cursor     *string
}

// LeaderboardResponse is the documented GET /v2/leaderboard envelope.
type LeaderboardResponse = Page[LeaderboardEntry]

// LiveVolumeRequest configures GET /v2/live-volume.
type LiveVolumeRequest struct {
	EventIDs []int64
}

// LiveVolumeResponse is the documented GET /v2/live-volume envelope.
type LiveVolumeResponse = Envelope[LiveVolume]

// OpenInterestRequest configures GET /v2/oi.
type OpenInterestRequest struct {
	Conditions []common.Hash
}

// OpenInterestResponse is the documented GET /v2/oi envelope.
type OpenInterestResponse = Envelope[[]OpenInterest]

// PositionsRequest configures GET /v2/positions.
type PositionsRequest struct {
	Filter          *MarketFilter
	TradeFilter     *TradeFilter
	User            common.Address
	Limit           *int
	Cursor          *string
	Status          *PositionStatus
	Title           *string
	IncludeArchived *bool
	SortBy          *PositionSortBy
	Start           *int64
	End             *int64
	SortDirection   *SortDirection
}

// PositionsResponse is the documented GET /v2/positions envelope.
type PositionsResponse = Page[Position]

// PricesHistoryRequest configures GET /v2/prices-history.
type PricesHistoryRequest struct {
	TokenID       *string
	Start         *int64
	End           *int64
	Interval      *string
	BucketSeconds *int64
	AsOf          *int64
	Limit         *int
	Cursor        *string
}

// PricesHistoryResponse is the documented GET /v2/prices-history envelope.
type PricesHistoryResponse = Page[PricePoint]

// ResolutionsRequest configures GET /v2/resolutions.
type ResolutionsRequest struct {
	QuestionID *string
	Conditions []common.Hash
	EventIDs   []int64
}

// ResolutionsResponse is the documented GET /v2/resolutions envelope.
type ResolutionsResponse = Envelope[[]Resolution]

// StatusResponse is the documented GET /v2/status envelope.
type StatusResponse = Envelope[ServiceStatus]

// TokensRequest configures GET /v2/tokens.
type TokensRequest struct {
	TokenID    *string
	Conditions []common.Hash
}

// TokensResponse is the documented GET /v2/tokens envelope.
type TokensResponse = Envelope[[]TokenRef]

// TradesRequest configures GET /v2/trades.
type TradesRequest struct {
	Filter      *MarketFilter
	TradeFilter *TradeFilter
	User        *common.Address
	Limit       *int
	Cursor      *string
	TakerOnly   *bool
	Start       *int64
	End         *int64
	Side        *Side
}

// TradesResponse is the documented GET /v2/trades envelope.
type TradesResponse = Page[Trade]

// UserPnlRequest configures GET /v2/user-pnl.
type UserPnlRequest struct {
	User     common.Address
	Interval *string
	Fidelity *string
}

// UserPnlResponse is the documented GET /v2/user-pnl envelope.
type UserPnlResponse = Envelope[UserPnlSeries]

// UserStatsRequest configures GET /v2/user-stats.
type UserStatsRequest struct {
	User common.Address
}

// UserStatsResponse is the documented GET /v2/user-stats envelope.
type UserStatsResponse = Envelope[*UserStats]

// UserVolumeRequest configures GET /v2/user-volume.
type UserVolumeRequest struct {
	User  common.Address
	Start *int64
	End   *int64
}

// UserVolumeResponse is the documented GET /v2/user-volume envelope.
type UserVolumeResponse = Envelope[UserVolume]

// ValueRequest configures GET /v2/value.
type ValueRequest struct {
	User       common.Address
	Conditions []common.Hash
}

// ValueResponse is the documented GET /v2/value envelope.
type ValueResponse = Envelope[PortfolioValue]

// LeaderboardUserRequest selects a single user rather than a ranked page.
type LeaderboardUserRequest struct {
	User       common.Address
	TimePeriod *TimePeriod
	Category   *LeaderboardCategory
}

// LeaderboardUserResponse preserves a documented missing user's null value.
type LeaderboardUserResponse = Envelope[*LeaderboardUserEntry]
