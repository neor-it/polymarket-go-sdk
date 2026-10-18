package positions

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

// EncodeMigratePositions converts only registered legacy conditions for the caller.
// It preserves the caller's nondecreasing legacy-condition order and explicit amounts.
func EncodeMigratePositions(conditions []common.Hash, outcomes []uint64, amounts []*big.Int) ([]byte, error) {
	indices, err := validateMigration(conditions, outcomes, amounts)
	if err != nil {
		return nil, err
	}
	return packMigration("migratePositions(bytes32[],uint256[],uint256[])", conditions, indices, amounts)
}

// EncodeOperatorMigratePositions encodes the authorized-operator overload for one holder.
func EncodeOperatorMigratePositions(owner common.Address, conditions []common.Hash, outcomes []uint64, amounts []*big.Int) ([]byte, error) {
	if owner == (common.Address{}) {
		return nil, errors.New("migration owner must not be zero")
	}
	indices, err := validateMigration(conditions, outcomes, amounts)
	if err != nil {
		return nil, err
	}
	return packMigration("migratePositions(address,bytes32[],uint256[],uint256[])", owner, conditions, indices, amounts)
}
func validateMigration(conditions []common.Hash, outcomes []uint64, amounts []*big.Int) ([]*big.Int, error) {
	if len(conditions) == 0 || len(conditions) != len(outcomes) || len(conditions) != len(amounts) {
		return nil, errors.New("migration arrays must be nonempty and aligned")
	}
	indices := make([]*big.Int, len(outcomes))
	for index, condition := range conditions {
		if condition == (common.Hash{}) {
			return nil, errors.New("legacy migration condition must not be zero")
		}
		if index > 0 && bytes.Compare(condition[:], conditions[index-1][:]) < 0 {
			return nil, errors.New("legacy migration conditions must be sorted")
		}
		if outcomes[index] >= binaryOutcomeCount {
			return nil, errors.New("migration outcome must be zero or one")
		}
		if err := validateUint(amounts[index], true, "migration amount"); err != nil {
			return nil, err
		}
		indices[index] = new(big.Int).SetUint64(outcomes[index])
	}
	return indices, nil
}
func packMigration(signature string, values ...any) ([]byte, error) {
	parsed, err := contractDefinition(ModuleABI)
	if err != nil {
		return nil, errors.Wrap(err, "parse migration ABI")
	}
	for name, method := range parsed.Methods {
		if method.Sig == signature {
			payload, err := parsed.Pack(name, values...)
			if err != nil {
				return nil, errors.Wrap(err, "encode registered migration")
			}
			return payload, nil
		}
	}
	return nil, errors.Errorf("migration method %s not found", signature)
}
