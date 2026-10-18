package data

import (
	"bytes"
	"encoding/json"
	"math"

	"github.com/pkg/errors"
)

// UnmarshalJSON requires published token-reference identity facts before accepting
// an outcome slot. Explicit slot zero and unresolved false remain valid values.
func (value *TokenRef) UnmarshalJSON(payload []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		return errors.Wrap(err, "decode token reference row")
	}
	for _, field := range []string{"token_id", "condition_id", "module", "outcome_index", "resolved"} {
		raw, exists := fields[field]
		if !exists || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return errors.Errorf("token reference field %s is required and must not be null", field)
		}
	}
	type wire TokenRef
	var decoded wire
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return errors.Wrap(err, "decode token reference identity")
	}
	if decoded.OutcomeIndex < math.MinInt32 || decoded.OutcomeIndex > math.MaxInt32 {
		return errors.Errorf("token reference outcome_index exceeds the published int32 range")
	}
	*value = TokenRef(decoded)
	return nil
}
