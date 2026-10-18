package data

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/neor-it/polymarket-go-sdk/pkg/types"
	"github.com/pkg/errors"
	"net/url"
	"strconv"
	"strings"
)

func addString[T ~string](q url.Values, key string, value *T) {
	if value != nil {
		q.Set(key, string(*value))
	}
}
func addInt(q url.Values, key string, value *int) {
	if value != nil {
		q.Set(key, strconv.Itoa(*value))
	}
}
func addInt64(q url.Values, key string, value *int64) {
	if value != nil {
		q.Set(key, strconv.FormatInt(*value, 10))
	}
}
func addBool(q url.Values, key string, value *bool) {
	if value != nil {
		q.Set(key, strconv.FormatBool(*value))
	}
}
func addDecimal(q url.Values, key string, value *types.Decimal) {
	if value != nil {
		q.Set(key, value.String())
	}
}
func addAddress(q url.Values, key string, value *common.Address) {
	if value != nil {
		q.Set(key, value.Hex())
	}
}
func addConditions(q url.Values, values []common.Hash) error {
	seen := make(map[common.Hash]struct{}, len(values))
	conditions := make([]string, 0, len(values))
	for _, value := range values {
		if value == (common.Hash{}) {
			return errors.New("condition ID must not be zero")
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		conditions = append(conditions, value.Hex())
	}
	if len(conditions) > maxConditions {
		return errors.Errorf("at most %d distinct conditions are supported", maxConditions)
	}
	if len(conditions) > 0 {
		q.Set("condition", strings.Join(conditions, ","))
	}
	return nil
}
func addEventIDs(q url.Values, values []int64) {
	if len(values) == 0 {
		return
	}
	ids := make([]string, 0, len(values))
	for _, value := range values {
		ids = append(ids, strconv.FormatInt(value, 10))
	}
	q.Set("event_id", strings.Join(ids, ","))
}
func addActivityTypes(q url.Values, values []ActivityType) {
	if len(values) == 0 {
		return
	}
	kinds := make([]string, 0, len(values))
	for _, value := range values {
		kinds = append(kinds, string(value))
	}
	q.Set("type", strings.Join(kinds, ","))
}
func applyMarketFilter(q url.Values, filter *MarketFilter) error {
	if filter == nil {
		return nil
	}
	if len(filter.Markets) > 0 && len(filter.EventIDs) > 0 {
		return errors.New("condition and event_id filters are mutually exclusive")
	}
	if err := addConditions(q, filter.Markets); err != nil {
		return errors.Wrap(err, "market filter")
	}
	addEventIDs(q, filter.EventIDs)
	return nil
}
func applyTradeFilter(q url.Values, filter *TradeFilter) error {
	if filter == nil {
		return nil
	}
	if filter.FilterType != FilterCash && filter.FilterType != FilterTokens {
		return errors.New("filter type must be CASH or TOKENS")
	}
	if filter.FilterAmount.Sign() < 0 {
		return errors.New("filter amount must not be negative")
	}
	q.Set("filter_type", string(filter.FilterType))
	q.Set("filter_amount", filter.FilterAmount.String())
	return nil
}
