package gamma

import (
	"encoding/hex"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

const (
	conditionV1Bytes = 32
	conditionV2Bytes = 31
)

// ConditionHash normalizes a condition selector without changing its byte order.
// V2 native bytes31 IDs are right-padded with zero. Padded V2 bytes32 IDs must
// already end in zero. common.HexToHash must not be used on a native bytes31 ID:
// its left-padding produces a different condition.
func ConditionHash(version ProtocolVersion, value string) (common.Hash, error) {
	var condition common.Hash
	if !strings.HasPrefix(value, "0x") {
		return condition, errors.New("condition ID must be 0x-prefixed hexadecimal")
	}
	raw, err := hex.DecodeString(strings.TrimPrefix(value, "0x"))
	if err != nil {
		return condition, errors.Wrap(err, "decode condition ID")
	}
	switch version {
	case ProtocolV1:
		if len(raw) != conditionV1Bytes {
			return condition, errors.Errorf("CTF condition ID must contain %d bytes", conditionV1Bytes)
		}
	case ProtocolV2:
		if len(raw) != conditionV2Bytes && len(raw) != conditionV1Bytes {
			return condition, errors.Errorf("V2 condition ID must contain %d or %d bytes", conditionV2Bytes, conditionV1Bytes)
		}
		if len(raw) == conditionV1Bytes && raw[conditionV2Bytes] != 0 {
			return condition, errors.New("padded V2 condition ID must end in zero")
		}
	default:
		return condition, errors.Errorf("unsupported condition protocol version %q", version)
	}
	copy(condition[:], raw)
	if condition == (common.Hash{}) {
		return condition, errors.New("condition ID must not be zero")
	}
	return condition, nil
}
