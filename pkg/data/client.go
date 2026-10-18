package data

import "context"

// Client defines all supported Data API v2 reads.
type Client interface {
	// LeaderboardUser reads one user, whose response is an object or null.
	// LeaderboardUser reads the corresponding typed Data API v2 response.
	LeaderboardUser(ctx context.Context, req *LeaderboardUserRequest) (LeaderboardUserResponse, error)
	// Activity reads the corresponding typed Data API v2 response.
	Activity(ctx context.Context, req *ActivityRequest) (ActivityResponse, error)
	// Approvals reads the corresponding typed Data API v2 response.
	Approvals(ctx context.Context, req *ApprovalsRequest) (ApprovalsResponse, error)
	// BiggestWinners reads the corresponding typed Data API v2 response.
	BiggestWinners(ctx context.Context, req *BiggestWinnersRequest) (BiggestWinnersResponse, error)
	// BuildersLeaderboard reads the corresponding typed Data API v2 response.
	BuildersLeaderboard(ctx context.Context, req *BuildersLeaderboardRequest) (BuildersLeaderboardResponse, error)
	// BuildersVolume reads the corresponding typed Data API v2 response.
	BuildersVolume(ctx context.Context, req *BuildersVolumeRequest) (BuildersVolumeResponse, error)
	// Holders reads the corresponding typed Data API v2 response.
	Holders(ctx context.Context, req *HoldersRequest) (HoldersResponse, error)
	// Leaderboard reads the corresponding typed Data API v2 response.
	Leaderboard(ctx context.Context, req *LeaderboardRequest) (LeaderboardResponse, error)
	// LiveVolume reads the corresponding typed Data API v2 response.
	LiveVolume(ctx context.Context, req *LiveVolumeRequest) (LiveVolumeResponse, error)
	// OpenInterest reads the corresponding typed Data API v2 response.
	OpenInterest(ctx context.Context, req *OpenInterestRequest) (OpenInterestResponse, error)
	// Positions reads the corresponding typed Data API v2 response.
	Positions(ctx context.Context, req *PositionsRequest) (PositionsResponse, error)
	// PricesHistory reads the corresponding typed Data API v2 response.
	PricesHistory(ctx context.Context, req *PricesHistoryRequest) (PricesHistoryResponse, error)
	// Resolutions reads the corresponding typed Data API v2 response.
	Resolutions(ctx context.Context, req *ResolutionsRequest) (ResolutionsResponse, error)
	// Status reads the corresponding typed Data API v2 response.
	Status(ctx context.Context) (StatusResponse, error)
	// Tokens reads the corresponding typed Data API v2 response.
	Tokens(ctx context.Context, req *TokensRequest) (TokensResponse, error)
	// Trades reads the corresponding typed Data API v2 response.
	Trades(ctx context.Context, req *TradesRequest) (TradesResponse, error)
	// UserPnl reads the corresponding typed Data API v2 response.
	UserPnl(ctx context.Context, req *UserPnlRequest) (UserPnlResponse, error)
	// UserStats reads the corresponding typed Data API v2 response.
	UserStats(ctx context.Context, req *UserStatsRequest) (UserStatsResponse, error)
	// UserVolume reads the corresponding typed Data API v2 response.
	UserVolume(ctx context.Context, req *UserVolumeRequest) (UserVolumeResponse, error)
	// Value reads the corresponding typed Data API v2 response.
	Value(ctx context.Context, req *ValueRequest) (ValueResponse, error)
}
