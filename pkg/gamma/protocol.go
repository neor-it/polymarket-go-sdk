package gamma

import (
	"encoding/json"
	"github.com/neor-it/polymarket-go-sdk/pkg/types"
	"github.com/pkg/errors"
	"math/big"
	"strings"
)

// ProtocolVersion identifies the position ledger and signing configuration.
type ProtocolVersion = types.ProtocolVersion

const (
	// ProtocolV1 identifies legacy CTF positions.
	ProtocolV1 = types.ProtocolV1
	// ProtocolV2 identifies PositionManager positions.
	ProtocolV2 = types.ProtocolV2
)

const (
	positionIDBits = 256
)

// TradingTokens selects IDs by the explicit market version and validates outcome alignment.
// It never infers a ledger from an ID field or a neg-risk flag.
func (m *Market) TradingTokens() ([]Token, error) {
	var ids []string
	switch m.Version {
	case ProtocolV1:
		if err := json.Unmarshal([]byte(m.ClobTokenIds), &ids); err != nil {
			return nil, errors.Wrap(err, "decode CTF token IDs")
		}
	case ProtocolV2:
		ids = m.PositionIDs
	default:
		return nil, errors.Errorf("unsupported market protocol version %q", m.Version)
	}
	var outcomes []string
	if err := json.Unmarshal([]byte(m.Outcomes), &outcomes); err != nil {
		return nil, errors.Wrap(err, "decode market outcomes")
	}
	if len(ids) == 0 || len(ids) != len(outcomes) {
		return nil, errors.New("outcome and asset ID counts must match and must not be empty")
	}
	tokens := make([]Token, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for index, id := range ids {
		if id == "" || strings.IndexFunc(id, func(char rune) bool { return char < '0' || char > '9' }) >= 0 {
			return nil, errors.Errorf("outcome %d has a non-decimal asset ID", index)
		}
		parsed, ok := new(big.Int).SetString(id, 10)
		if !ok || parsed.Sign() < 0 || parsed.BitLen() > positionIDBits {
			return nil, errors.Errorf("outcome %d asset ID exceeds uint256", index)
		}
		if _, exists := seen[parsed.String()]; exists {
			return nil, errors.Errorf("outcome %d has a duplicate asset ID", index)
		}
		if strings.TrimSpace(outcomes[index]) == "" {
			return nil, errors.Errorf("outcome %d has no label", index)
		}
		seen[parsed.String()] = struct{}{}
		tokens[index] = Token{TokenID: id, Outcome: outcomes[index]}
	}
	return tokens, nil
}

// ParsedTokens returns validated trading tokens, or nil for an unsupported or invalid market.
// Use TradingTokens when the reason for rejection must be surfaced to the caller.
func (m *Market) ParsedTokens() []Token {
	tokens, err := m.TradingTokens()
	if err != nil {
		return nil
	}
	return tokens
}
