package bot

import (
	"context"

	"github.com/neor-it/polymarket-go-sdk/pkg/clob"
	"github.com/neor-it/polymarket-go-sdk/pkg/clob/clobtypes"
)

// stubClient is a test double for clob.Client. It embeds the interface so only
// the methods the bot actually exercises need to be implemented; any other call
// would panic (and would indicate the bot started depending on a new method).
// Each used method is backed by a function field so tests can inject behavior.
type stubClient struct {
	clob.Client

	marketsFn  func(ctx context.Context, req *clobtypes.MarketsRequest) (clobtypes.MarketsResponse, error)
	orderBook  func(ctx context.Context, req *clobtypes.BookRequest) (clobtypes.OrderBookResponse, error)
	ordersFn   func(ctx context.Context, req *clobtypes.OrdersRequest) (clobtypes.OrdersResponse, error)
	tickSizeFn func(ctx context.Context, req *clobtypes.TickSizeRequest) (clobtypes.TickSizeResponse, error)
	negRiskFn  func(ctx context.Context, req *clobtypes.NegRiskRequest) (clobtypes.NegRiskResponse, error)
	createFn   func(ctx context.Context, order *clobtypes.SignableOrder) (clobtypes.OrderResponse, error)
}

func (s *stubClient) Markets(ctx context.Context, req *clobtypes.MarketsRequest) (clobtypes.MarketsResponse, error) {
	return s.marketsFn(ctx, req)
}

func (s *stubClient) OrderBook(ctx context.Context, req *clobtypes.BookRequest) (clobtypes.OrderBookResponse, error) {
	return s.orderBook(ctx, req)
}

func (s *stubClient) Orders(ctx context.Context, req *clobtypes.OrdersRequest) (clobtypes.OrdersResponse, error) {
	return s.ordersFn(ctx, req)
}

func (s *stubClient) TickSize(ctx context.Context, req *clobtypes.TickSizeRequest) (clobtypes.TickSizeResponse, error) {
	return s.tickSizeFn(ctx, req)
}

func (s *stubClient) NegRisk(ctx context.Context, req *clobtypes.NegRiskRequest) (clobtypes.NegRiskResponse, error) {
	return s.negRiskFn(ctx, req)
}

func (s *stubClient) CreateOrderFromSignable(ctx context.Context, order *clobtypes.SignableOrder) (clobtypes.OrderResponse, error) {
	return s.createFn(ctx, order)
}
