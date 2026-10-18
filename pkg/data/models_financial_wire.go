package data

import "github.com/pkg/errors"

// UnmarshalJSON preserves explicit zero while rejecting unavailable required financial values.
func (value *Activity) UnmarshalJSON(payload []byte) error {
	type wire Activity
	var decoded wire
	if err := decodeFinancialFields(payload, &decoded, "price", "size", "usdc_size"); err != nil {
		return errors.Wrap(err, "decode Activity financial fields")
	}
	*value = Activity(decoded)
	return nil
}

// UnmarshalJSON preserves explicit zero while rejecting unavailable required financial values.
func (value *BiggestWinner) UnmarshalJSON(payload []byte) error {
	type wire BiggestWinner
	var decoded wire
	if err := decodeFinancialFields(payload, &decoded, "final_value", "initial_value", "pnl"); err != nil {
		return errors.Wrap(err, "decode BiggestWinner financial fields")
	}
	*value = BiggestWinner(decoded)
	return nil
}

// UnmarshalJSON preserves explicit zero while rejecting unavailable required financial values.
func (value *BuilderStanding) UnmarshalJSON(payload []byte) error {
	type wire BuilderStanding
	var decoded wire
	if err := decodeFinancialFields(payload, &decoded, "volume"); err != nil {
		return errors.Wrap(err, "decode BuilderStanding financial fields")
	}
	*value = BuilderStanding(decoded)
	return nil
}

// UnmarshalJSON preserves explicit zero while rejecting unavailable required financial values.
func (value *BuilderVolumePoint) UnmarshalJSON(payload []byte) error {
	type wire BuilderVolumePoint
	var decoded wire
	if err := decodeFinancialFields(payload, &decoded, "volume"); err != nil {
		return errors.Wrap(err, "decode BuilderVolumePoint financial fields")
	}
	*value = BuilderVolumePoint(decoded)
	return nil
}

// UnmarshalJSON preserves explicit zero while rejecting unavailable required financial values.
func (value *ComboLeg) UnmarshalJSON(payload []byte) error {
	type wire ComboLeg
	var decoded wire
	if err := decodeFinancialFields(payload, &decoded, "leg_current_price"); err != nil {
		return errors.Wrap(err, "decode ComboLeg financial fields")
	}
	*value = ComboLeg(decoded)
	return nil
}

// UnmarshalJSON preserves explicit zero while rejecting unavailable required financial values.
func (value *ComboPosition) UnmarshalJSON(payload []byte) error {
	type wire ComboPosition
	var decoded wire
	if err := decodeFinancialFields(payload, &decoded, "current_size", "entry_avg_price_usdc", "entry_cost_usdc", "entry_fees_usdc", "gross_entry_cost_usdc", "realized_payout_usdc"); err != nil {
		return errors.Wrap(err, "decode ComboPosition financial fields")
	}
	*value = ComboPosition(decoded)
	return nil
}

// UnmarshalJSON preserves explicit zero while rejecting unavailable required financial values.
func (value *ConditionVolume) UnmarshalJSON(payload []byte) error {
	type wire ConditionVolume
	var decoded wire
	if err := decodeFinancialFields(payload, &decoded, "taker_volume"); err != nil {
		return errors.Wrap(err, "decode ConditionVolume financial fields")
	}
	*value = ConditionVolume(decoded)
	return nil
}

// UnmarshalJSON preserves explicit zero while rejecting unavailable required financial values.
func (value *Holder) UnmarshalJSON(payload []byte) error {
	type wire Holder
	var decoded wire
	if err := decodeFinancialFields(payload, &decoded, "amount"); err != nil {
		return errors.Wrap(err, "decode Holder financial fields")
	}
	*value = Holder(decoded)
	return nil
}

// UnmarshalJSON preserves explicit zero while rejecting unavailable required financial values.
func (value *LeaderboardEntry) UnmarshalJSON(payload []byte) error {
	type wire LeaderboardEntry
	var decoded wire
	if err := decodeFinancialFields(payload, &decoded, "pnl", "volume"); err != nil {
		return errors.Wrap(err, "decode LeaderboardEntry financial fields")
	}
	*value = LeaderboardEntry(decoded)
	return nil
}

// UnmarshalJSON preserves explicit zero while rejecting unavailable required financial values.
func (value *LeaderboardUserEntry) UnmarshalJSON(payload []byte) error {
	type wire LeaderboardUserEntry
	var decoded wire
	if err := decodeFinancialFields(payload, &decoded, "pnl", "volume"); err != nil {
		return errors.Wrap(err, "decode LeaderboardUserEntry financial fields")
	}
	*value = LeaderboardUserEntry(decoded)
	return nil
}

// UnmarshalJSON preserves explicit zero while rejecting unavailable required financial values.
func (value *LiveVolume) UnmarshalJSON(payload []byte) error {
	type wire LiveVolume
	var decoded wire
	if err := decodeFinancialFields(payload, &decoded, "taker_volume_total"); err != nil {
		return errors.Wrap(err, "decode LiveVolume financial fields")
	}
	*value = LiveVolume(decoded)
	return nil
}

// UnmarshalJSON preserves explicit zero while rejecting unavailable required financial values.
func (value *OpenInterest) UnmarshalJSON(payload []byte) error {
	type wire OpenInterest
	var decoded wire
	if err := decodeFinancialFields(payload, &decoded, "value"); err != nil {
		return errors.Wrap(err, "decode OpenInterest financial fields")
	}
	*value = OpenInterest(decoded)
	return nil
}

// UnmarshalJSON preserves explicit zero while rejecting unavailable required financial values.
func (value *PortfolioValue) UnmarshalJSON(payload []byte) error {
	type wire PortfolioValue
	var decoded wire
	if err := decodeFinancialFields(payload, &decoded, "value"); err != nil {
		return errors.Wrap(err, "decode PortfolioValue financial fields")
	}
	*value = PortfolioValue(decoded)
	return nil
}

// UnmarshalJSON preserves explicit zero while rejecting unavailable required financial values.
func (value *Position) UnmarshalJSON(payload []byte) error {
	type wire Position
	var decoded wire
	if err := decodeFinancialFields(payload, &decoded, "avg_price", "current_price", "current_size", "current_value", "entry_cost_usdc", "entry_fees_usdc", "percent_pnl", "percent_realized_pnl", "realized_pnl", "total_cost_usdc", "total_pnl", "total_size", "unrealized_pnl"); err != nil {
		return errors.Wrap(err, "decode Position financial fields")
	}
	*value = Position(decoded)
	return nil
}

// UnmarshalJSON preserves explicit zero while rejecting unavailable required financial values.
func (value *PricePoint) UnmarshalJSON(payload []byte) error {
	type wire PricePoint
	var decoded wire
	if err := decodeFinancialFields(payload, &decoded, "price"); err != nil {
		return errors.Wrap(err, "decode PricePoint financial fields")
	}
	*value = PricePoint(decoded)
	return nil
}

// UnmarshalJSON preserves explicit zero while rejecting unavailable required financial values.
func (value *Trade) UnmarshalJSON(payload []byte) error {
	type wire Trade
	var decoded wire
	if err := decodeFinancialFields(payload, &decoded, "price", "size"); err != nil {
		return errors.Wrap(err, "decode Trade financial fields")
	}
	*value = Trade(decoded)
	return nil
}

// UnmarshalJSON preserves explicit zero while rejecting unavailable required financial values.
func (value *UserPnlPoint) UnmarshalJSON(payload []byte) error {
	type wire UserPnlPoint
	var decoded wire
	if err := decodeFinancialFields(payload, &decoded, "realized_combo_pnl", "realized_lp_pnl", "realized_market_pnl", "realized_pnl", "volume", "volume_usdc"); err != nil {
		return errors.Wrap(err, "decode UserPnlPoint financial fields")
	}
	*value = UserPnlPoint(decoded)
	return nil
}

// UnmarshalJSON preserves explicit zero while rejecting unavailable required financial values.
func (value *UserStats) UnmarshalJSON(payload []byte) error {
	type wire UserStats
	var decoded wire
	if err := decodeFinancialFields(payload, &decoded, "biggest_win"); err != nil {
		return errors.Wrap(err, "decode UserStats financial fields")
	}
	*value = UserStats(decoded)
	return nil
}

// UnmarshalJSON preserves explicit zero while rejecting unavailable required financial values.
func (value *UserVolume) UnmarshalJSON(payload []byte) error {
	type wire UserVolume
	var decoded wire
	if err := decodeFinancialFields(payload, &decoded, "volume", "volume_usdc"); err != nil {
		return errors.Wrap(err, "decode UserVolume financial fields")
	}
	*value = UserVolume(decoded)
	return nil
}
