package positions

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

const (
	moduleShift         = 248
	conditionIndexShift = 8
	conditionIndexMask  = 0xffff
	arityShift          = 104
	outcomeByteIndex    = 31
	eventSuffixBytes    = 3
)

// PositionIdentity preserves the structured on-chain identity of a V2 position.
type PositionIdentity struct {
	ModuleID, OutcomeIndex uint8
	ConditionIndex, Arity  uint16
	ConditionID, EventID   common.Hash
}

// PositionID adds an explicit binary outcome byte to a validated V2 condition ID.
func PositionID(condition string, outcomeIndex uint64) (*big.Int, error) {
	if outcomeIndex >= binaryOutcomeCount {
		return nil, errors.New("position outcome must be zero or one")
	}
	id, err := ConditionID(condition)
	if err != nil {
		return nil, err
	}
	var padded common.Hash
	copy(padded[:], id[:])
	padded[outcomeByteIndex] = byte(outcomeIndex)
	return new(big.Int).SetBytes(padded[:]), nil
}

// DecodePositionID separates module, condition and outcome without losing uint256 bits.
// Ledger selection must still come from explicit protocol metadata, never from this layout alone.
func DecodePositionID(positionID *big.Int) (PositionIdentity, error) {
	if err := validateUint(positionID, false, "position ID"); err != nil {
		return PositionIdentity{}, err
	}
	bytes := common.BigToHash(positionID)
	identity := PositionIdentity{ModuleID: uint8(new(big.Int).Rsh(new(big.Int).Set(positionID), moduleShift).Uint64()), OutcomeIndex: bytes[outcomeByteIndex], ConditionIndex: uint16(new(big.Int).Rsh(new(big.Int).Set(positionID), conditionIndexShift).Uint64() & conditionIndexMask), Arity: uint16(new(big.Int).Rsh(new(big.Int).Set(positionID), arityShift).Uint64() & conditionIndexMask), ConditionID: bytes, EventID: bytes}
	identity.ConditionID[outcomeByteIndex] = 0
	for index := common.HashLength - eventSuffixBytes; index < common.HashLength; index++ {
		identity.EventID[index] = 0
	}
	return identity, nil
}
