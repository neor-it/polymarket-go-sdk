package clob

import (
	"context"
	"strings"
	"time"

	"github.com/neor-it/polymarket-go-sdk/pkg/clob/clobtypes"
)

const (
	defaultOrderTradeResolutionTimeout      = 30 * time.Second
	defaultOrderTradeResolutionPollInterval = 500 * time.Millisecond
	orderTradeLookupLimit                   = 1
	failedTradeStatus                       = "FAILED"
	failedTradeStatusV2                     = "TRADE_STATUS_FAILED"
)

func (c *clientImpl) resolveOrderTransactionHashes(ctx context.Context, resp *clobtypes.OrderResponse, deferExec bool) {
	if c == nil || resp == nil || deferExec || len(resp.TransactionHashes) > 0 || len(resp.TradeIDs) == 0 {
		return
	}

	trades := c.waitForResolvedTrades(ctx, resp.TradeIDs)
	for i := range trades {
		hash := strings.TrimSpace(trades[i].TransactionHash)
		if hash == "" {
			continue
		}
		resp.TransactionHashes = append(resp.TransactionHashes, hash)
	}
}

func (c *clientImpl) waitForResolvedTrades(ctx context.Context, tradeIDs []string) []clobtypes.Trade {
	remainingIDs := uniqueNonEmptyTradeIDs(tradeIDs)
	if len(remainingIDs) == 0 {
		return nil
	}

	deadline := time.NewTimer(defaultOrderTradeResolutionTimeout)
	defer deadline.Stop()

	ticker := time.NewTicker(defaultOrderTradeResolutionPollInterval)
	defer ticker.Stop()

	resolvedByID := make(map[string]clobtypes.Trade, len(remainingIDs))

	for {
		c.pollPendingTrades(ctx, remainingIDs, resolvedByID)
		if len(resolvedByID) == len(remainingIDs) {
			break
		}

		select {
		case <-ctx.Done():
			return orderedResolvedTrades(remainingIDs, resolvedByID)
		case <-deadline.C:
			return orderedResolvedTrades(remainingIDs, resolvedByID)
		case <-ticker.C:
		}
	}

	return orderedResolvedTrades(remainingIDs, resolvedByID)
}

func (c *clientImpl) pollPendingTrades(ctx context.Context, tradeIDs []string, resolvedByID map[string]clobtypes.Trade) {
	for _, tradeID := range tradeIDs {
		if _, ok := resolvedByID[tradeID]; ok {
			continue
		}

		resp, err := c.Trades(ctx, &clobtypes.TradesRequest{
			ID:    tradeID,
			Limit: orderTradeLookupLimit,
		})
		if err != nil {
			continue
		}

		for i := range resp.Data {
			trade := resp.Data[i]
			if strings.TrimSpace(trade.ID) != tradeID || !isResolvedTrade(trade) {
				continue
			}
			resolvedByID[tradeID] = trade
			break
		}
	}
}

func isResolvedTrade(trade clobtypes.Trade) bool {
	if strings.TrimSpace(trade.TransactionHash) != "" {
		return true
	}
	status := strings.ToUpper(strings.TrimSpace(trade.Status))
	return status == failedTradeStatus || status == failedTradeStatusV2
}

func uniqueNonEmptyTradeIDs(tradeIDs []string) []string {
	seenIDs := make(map[string]struct{}, len(tradeIDs))
	uniqueIDs := make([]string, 0, len(tradeIDs))
	for _, tradeID := range tradeIDs {
		trimmedTradeID := strings.TrimSpace(tradeID)
		if trimmedTradeID == "" {
			continue
		}
		if _, ok := seenIDs[trimmedTradeID]; ok {
			continue
		}
		seenIDs[trimmedTradeID] = struct{}{}
		uniqueIDs = append(uniqueIDs, trimmedTradeID)
	}
	return uniqueIDs
}

func orderedResolvedTrades(tradeIDs []string, resolvedByID map[string]clobtypes.Trade) []clobtypes.Trade {
	if len(resolvedByID) == 0 {
		return nil
	}

	orderedTrades := make([]clobtypes.Trade, 0, len(resolvedByID))
	for _, tradeID := range tradeIDs {
		trade, ok := resolvedByID[tradeID]
		if !ok {
			continue
		}
		orderedTrades = append(orderedTrades, trade)
	}
	return orderedTrades
}
