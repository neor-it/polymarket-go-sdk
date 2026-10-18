package data

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
)

// decodeFinancialFields checks only documented required financial numbers.
// Missing or null source data must not acquire decimal.Decimal's zero value.
// Nullable financial fields retain their published pointer representation.
func decodeFinancialFields(payload []byte, destination any, requiredFields ...string) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		return errors.Wrap(err, "decode financial row")
	}
	for _, field := range requiredFields {
		raw, exists := fields[field]
		if !exists || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return errors.Errorf("financial field %s is required and must not be null", field)
		}
	}
	if err := json.Unmarshal(payload, destination); err != nil {
		return errors.Wrap(err, "decode financial values")
	}
	return nil
}
