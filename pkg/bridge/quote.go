package bridge

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// BridgeQuoteRequest describes an exact base-unit transfer preview.
type BridgeQuoteRequest struct {
	FromChainID        string `json:"fromChainId"`
	ToChainID          string `json:"toChainId"`
	FromTokenAddr      string `json:"fromTokenAddress"`
	ToTokenAddr        string `json:"toTokenAddress"`
	FromAmountBaseUnit string `json:"fromAmountBaseUnit"`
	RecipientAddress   string `json:"recipientAddress"`
}

// BridgeQuoteResponse preserves optional estimates as absent instead of zero.
type BridgeQuoteResponse struct {
	EstimatedCheckoutTimeMs  *int64              `json:"estCheckoutTimeMs"`
	EstimatedFeeBreakdown    *BridgeFeeBreakdown `json:"estFeeBreakdown"`
	EstimatedInputUSD        *decimal.Decimal    `json:"estInputUsd"`
	EstimatedOutputUSD       *decimal.Decimal    `json:"estOutputUsd"`
	EstimatedToTokenBaseUnit *string             `json:"estToTokenBaseUnit"`
	QuoteID                  string              `json:"quoteId"`
}

// BridgeFeeBreakdown contains the optional individual cost estimates.
type BridgeFeeBreakdown struct {
	AppFeeLabel     string           `json:"appFeeLabel"`
	AppFeePercent   *decimal.Decimal `json:"appFeePercent"`
	AppFeeUSD       *decimal.Decimal `json:"appFeeUsd"`
	FillCostPercent *decimal.Decimal `json:"fillCostPercent"`
	FillCostUSD     *decimal.Decimal `json:"fillCostUsd"`
	GasUSD          *decimal.Decimal `json:"gasUsd"`
	MaxSlippage     *decimal.Decimal `json:"maxSlippage"`
	MinReceived     *decimal.Decimal `json:"minReceived"`
	SwapImpact      *decimal.Decimal `json:"swapImpact"`
	SwapImpactUSD   *decimal.Decimal `json:"swapImpactUsd"`
	TotalImpact     *decimal.Decimal `json:"totalImpact"`
	TotalImpactUSD  *decimal.Decimal `json:"totalImpactUsd"`
}

// WithdrawAddressRequest selects a source Polymarket wallet and destination asset.
type WithdrawAddressRequest struct {
	Address       string `json:"address"`
	ToChainID     string `json:"toChainId"`
	ToTokenAddr   string `json:"toTokenAddress"`
	RecipientAddr string `json:"recipientAddr"`
}

// WithdrawAddressResponse returns deposit addresses for a later explicit transfer.
type WithdrawAddressResponse struct {
	Address DepositAddresses `json:"address"`
	Note    string           `json:"note,omitempty"`
}

// Quote calls the corresponding typed API endpoint and preserves transport errors.
func (c *clientImpl) Quote(ctx context.Context, req *BridgeQuoteRequest) (BridgeQuoteResponse, error) {
	var resp BridgeQuoteResponse
	if req == nil {
		return resp, fmt.Errorf("bridge quote request is required")
	}
	if strings.TrimSpace(req.FromChainID) == "" || strings.TrimSpace(req.ToChainID) == "" || strings.TrimSpace(req.FromTokenAddr) == "" || strings.TrimSpace(req.ToTokenAddr) == "" || strings.TrimSpace(req.FromAmountBaseUnit) == "" || strings.TrimSpace(req.RecipientAddress) == "" {
		return resp, fmt.Errorf("bridge quote source, destination, amount and recipient are required")
	}
	err := c.httpClient.Post(ctx, "/quote", req, &resp)
	return resp, err
}

// WithdrawAddress calls the corresponding typed API endpoint and preserves transport errors.
func (c *clientImpl) WithdrawAddress(ctx context.Context, req *WithdrawAddressRequest) (WithdrawAddressResponse, error) {
	var resp WithdrawAddressResponse
	if req == nil {
		return resp, fmt.Errorf("bridge withdrawal request is required")
	}
	if strings.TrimSpace(req.Address) == "" || strings.TrimSpace(req.ToChainID) == "" || strings.TrimSpace(req.ToTokenAddr) == "" || strings.TrimSpace(req.RecipientAddr) == "" {
		return resp, fmt.Errorf("withdrawal source, destination and recipient are required")
	}
	err := c.httpClient.Post(ctx, "/withdraw", req, &resp)
	return resp, err
}
