// Package data provides typed, cursor-paginated Polymarket Data API v2 reads.
package data

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/neor-it/polymarket-go-sdk/pkg/types"
)

// Enum types.
type (
	Side                string
	ActivityType        string
	PositionSortBy      string
	ActivitySortBy      string
	SortDirection       string
	FilterType          string
	TimePeriod          string
	LeaderboardCategory string
	LeaderboardOrderBy  string
)

const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

const (
	ActivityTrade       ActivityType = "TRADE"
	ActivitySplit       ActivityType = "SPLIT"
	ActivityMerge       ActivityType = "MERGE"
	ActivityRedeem      ActivityType = "REDEEM"
	ActivityReward      ActivityType = "REWARD"
	ActivityConversion  ActivityType = "CONVERSION"
	ActivityYield       ActivityType = "YIELD"
	ActivityMakerRebate ActivityType = "MAKER_REBATE"
)

const (
	PositionSortCurrentValue  PositionSortBy = "CURRENT_VALUE"
	PositionSortPrice         PositionSortBy = "PRICE"
	PositionSortTokens        PositionSortBy = "TOKENS"
	PositionSortUnrealizedPnl PositionSortBy = "UNREALIZED_PNL"
	PositionSortRealizedPnl   PositionSortBy = "REALIZED_PNL"
	PositionSortTotalPnl      PositionSortBy = "TOTAL_PNL"
	PositionSortTimestamp     PositionSortBy = "TIMESTAMP"
)
const ActivitySortTimestamp ActivitySortBy = "TIMESTAMP"

const (
	SortAsc  SortDirection = "ASC"
	SortDesc SortDirection = "DESC"
)

const (
	FilterCash   FilterType = "CASH"
	FilterTokens FilterType = "TOKENS"
)

const (
	TimePeriodDay   TimePeriod = "day"
	TimePeriodWeek  TimePeriod = "week"
	TimePeriodMonth TimePeriod = "month"
	TimePeriodAll   TimePeriod = "all"
)

const (
	LeaderboardOverall   LeaderboardCategory = "overall"
	LeaderboardPolitics  LeaderboardCategory = "politics"
	LeaderboardSports    LeaderboardCategory = "sports"
	LeaderboardCrypto    LeaderboardCategory = "crypto"
	LeaderboardCulture   LeaderboardCategory = "culture"
	LeaderboardMentions  LeaderboardCategory = "mentions"
	LeaderboardWeather   LeaderboardCategory = "weather"
	LeaderboardEconomics LeaderboardCategory = "economics"
	LeaderboardTech      LeaderboardCategory = "tech"
	LeaderboardFinance   LeaderboardCategory = "finance"
)

const (
	OrderByPnl LeaderboardOrderBy = "PNL"
	OrderByVol LeaderboardOrderBy = "VOLUME"
)

// PositionStatus selects a position lifecycle state.
type PositionStatus string

const (
	PositionOpen           PositionStatus = "OPEN"
	PositionRedeemable     PositionStatus = "REDEEMABLE"
	PositionRedeemableLost PositionStatus = "REDEEMABLE_LOST"
	PositionMergeable      PositionStatus = "MERGEABLE"
	PositionClosed         PositionStatus = "CLOSED"
)

// MarketFilter restricts a read to condition IDs or Gamma event IDs.
// Markets is serialized as condition, never as the retired market parameter.
type MarketFilter struct {
	Markets  []common.Hash
	EventIDs []int64
}

// MarketFilterByMarkets selects condition IDs.
func MarketFilterByMarkets(markets []common.Hash) *MarketFilter {
	return &MarketFilter{Markets: markets}
}

// MarketFilterByEventIDs selects Gamma event IDs.
func MarketFilterByEventIDs(eventIDs []int64) *MarketFilter { return &MarketFilter{EventIDs: eventIDs} }

// TradeFilter restricts rows to a minimum cash or share quantity.
type TradeFilter struct {
	FilterType   FilterType
	FilterAmount types.Decimal
}

// NewTradeFilter constructs a minimum-quantity filter.
func NewTradeFilter(kind FilterType, amount types.Decimal) *TradeFilter {
	return &TradeFilter{FilterType: kind, FilterAmount: amount}
}

// TradeFilterCash selects a minimum cash quantity.
func TradeFilterCash(amount types.Decimal) *TradeFilter { return NewTradeFilter(FilterCash, amount) }

// TradeFilterTokens selects a minimum share quantity.
func TradeFilterTokens(amount types.Decimal) *TradeFilter {
	return NewTradeFilter(FilterTokens, amount)
}

// Pagination carries a server-issued cursor bound to the original query.
// Offset is diagnostic response metadata, not a request parameter.
type Pagination struct {
	HasMore    bool    `json:"has_more"`
	Limit      int64   `json:"limit"`
	NextCursor *string `json:"next_cursor"`
	Offset     int64   `json:"offset"`
}

// Page is one cursor page. Follow NextCursor until nil; no automatic truncation occurs.
type Page[T any] struct {
	Data       []T        `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// Envelope contains a non-paginated v2 response, including documented null results.
type Envelope[T any] struct {
	Data T `json:"data"`
}
