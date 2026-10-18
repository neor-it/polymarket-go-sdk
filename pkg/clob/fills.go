package clob

import (
	"math/big"
	"strings"

	"github.com/neor-it/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/pkg/errors"
)

const orderUint256Bits = 256

// FillAmounts keeps base-unit principal and fees separate for one signed maker fill.
type FillAmounts struct{ MakerAsset, TakerAsset, Collateral, Shares, Fee, CollateralDebit, CollateralCredit *big.Int }

// CalculateFill computes floor(makerAssetFill*takerAmount/makerAmount) using only integers.
// Maker asset is collateral for BUY and shares for SELL. BUY fees add to spend;
// SELL fees reduce proceeds. This function never reconstructs fees from a price.
func CalculateFill(order *clobtypes.Order, makerAssetFill, collateralFee *big.Int) (FillAmounts, error) {
	if order == nil {
		return FillAmounts{}, errors.New("signed maker order is required")
	}
	if err := validateOrderFinancials(order); err != nil {
		return FillAmounts{}, err
	}
	makerAmount, takerAmount := order.MakerAmount.BigInt(), order.TakerAmount.BigInt()
	if makerAssetFill == nil || makerAssetFill.Sign() < 0 || makerAssetFill.Cmp(makerAmount) > 0 {
		return FillAmounts{}, errors.New("maker fill must be nonnegative and within the signed maker amount")
	}
	if collateralFee == nil || collateralFee.Sign() < 0 || collateralFee.BitLen() > orderUint256Bits {
		return FillAmounts{}, errors.New("collateral fee must be a uint256")
	}
	counter := new(big.Int).Quo(new(big.Int).Mul(makerAssetFill, takerAmount), makerAmount)
	result := FillAmounts{MakerAsset: new(big.Int).Set(makerAssetFill), TakerAsset: counter, Fee: new(big.Int).Set(collateralFee), CollateralDebit: new(big.Int), CollateralCredit: new(big.Int)}
	switch strings.ToUpper(order.Side) {
	case "BUY":
		result.Collateral = new(big.Int).Set(makerAssetFill)
		result.Shares = new(big.Int).Set(counter)
		result.CollateralDebit.Add(result.Collateral, collateralFee)
	case "SELL":
		if collateralFee.Cmp(counter) > 0 {
			return FillAmounts{}, errors.New("sell fee exceeds collateral proceeds")
		}
		result.Collateral = new(big.Int).Set(counter)
		result.Shares = new(big.Int).Set(makerAssetFill)
		result.CollateralCredit.Sub(counter, collateralFee)
	default:
		return FillAmounts{}, errors.New("maker order side must be BUY or SELL")
	}
	return result, nil
}

func validateOrderFinancials(order *clobtypes.Order) error {
	if strings.ToUpper(order.Side) != "BUY" && strings.ToUpper(order.Side) != "SELL" {
		return errors.New("order side must be BUY or SELL")
	}
	if order.Salt.Int != nil && (order.Salt.Sign() < 0 || order.Salt.BitLen() > orderUint256Bits) {
		return errors.New("order salt must be a uint256")
	}
	if order.Timestamp < 0 {
		return errors.New("order timestamp must not be negative")
	}

	if order.TokenID.Int == nil || order.TokenID.Sign() < 0 || order.TokenID.BitLen() > orderUint256Bits {
		return errors.New("order token ID must be a uint256")
	}
	for _, amount := range []struct {
		name    string
		value   *big.Int
		integer bool
	}{{"maker amount", order.MakerAmount.BigInt(), order.MakerAmount.Truncate(0).Equal(order.MakerAmount)}, {"taker amount", order.TakerAmount.BigInt(), order.TakerAmount.Truncate(0).Equal(order.TakerAmount)}} {
		if !amount.integer || amount.value.Sign() <= 0 || amount.value.BitLen() > orderUint256Bits {
			return errors.Errorf("order %s must be a positive integer uint256", amount.name)
		}
	}
	return nil
}
