package clobtypes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/shopspring/decimal"
)

// UnmarshalJSON accepts a bare server timestamp or an object without a second HTTP request.
func (t *TimeResponse) UnmarshalJSON(data []byte) error {
	raw := bytes.TrimSpace(data)
	if bytes.Equal(raw, []byte("null")) {
		return fmt.Errorf("server time cannot be null")
	}
	if len(raw) > 0 && raw[0] == '{' {
		var value struct {
			Timestamp json.RawMessage `json:"timestamp"`
		}
		if err := json.Unmarshal(raw, &value); err != nil {
			return err
		}
		if len(value.Timestamp) == 0 || bytes.Equal(bytes.TrimSpace(value.Timestamp), []byte("null")) {
			return fmt.Errorf("server timestamp is required")
		}
		raw = value.Timestamp
	}
	var timestamp int64
	if err := json.Unmarshal(raw, &timestamp); err != nil {
		return fmt.Errorf("decode server timestamp: %w", err)
	}
	if timestamp < 0 {
		return fmt.Errorf("server timestamp must not be negative")
	}
	t.Timestamp = timestamp
	return nil
}

func decodeEarnings[T any](data []byte, target *T, requiredRate bool) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for _, key := range []string{"earnings", "asset_rate"} {
		raw, present := fields[key]
		if !present {
			if key == "earnings" || requiredRate {
				return fmt.Errorf("%s is required", key)
			}
			continue
		}
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return fmt.Errorf("%s cannot be null", key)
		}
		var number NumericString
		if err := json.Unmarshal(raw, &number); err != nil {
			return fmt.Errorf("decode %s: %w", key, err)
		}
		if _, err := decimal.NewFromString(number.String()); err != nil {
			return fmt.Errorf("invalid %s: %w", key, err)
		}
		encoded, err := json.Marshal(number.String())
		if err != nil {
			return err
		}
		fields[key] = encoded
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, target)
}

// UnmarshalJSON preserves exact reward amounts from either number or string JSON.
func (e *Earning) UnmarshalJSON(data []byte) error {
	type plain Earning
	var next plain
	if err := decodeEarnings(data, &next, false); err != nil {
		return err
	}
	*e = Earning(next)
	return nil
}

// UnmarshalJSON preserves exact daily earnings without changing the public string fields.
func (e *UserEarning) UnmarshalJSON(data []byte) error {
	type plain UserEarning
	var next plain
	if err := decodeEarnings(data, &next, false); err != nil {
		return err
	}
	*e = UserEarning(next)
	return nil
}

// UnmarshalJSON preserves exact total earnings and requires the asset rate.
func (e *TotalUserEarning) UnmarshalJSON(data []byte) error {
	type plain TotalUserEarning
	var next plain
	if err := decodeEarnings(data, &next, true); err != nil {
		return err
	}
	*e = TotalUserEarning(next)
	return nil
}
