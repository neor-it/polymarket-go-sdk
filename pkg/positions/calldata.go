package positions

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/neor-it/polymarket-go-sdk/pkg/gamma"
	"github.com/pkg/errors"
)

const (
	uint256Bits        = 256
	binaryOutcomeCount = 2
)

// ConditionID normalizes a native or right-padded V2 condition to the Router's bytes31.
func ConditionID(value string) ([31]byte, error) {
	var result [31]byte
	canonical, err := gamma.ConditionHash(gamma.ProtocolV2, value)
	if err != nil {
		return result, errors.Wrap(err, "normalize V2 condition")
	}
	copy(result[:], canonical[:])
	return result, nil
}

func validateUint(value *big.Int, positive bool, label string) error {
	if value == nil || value.Sign() < 0 || value.BitLen() > uint256Bits {
		return errors.Errorf("%s must be a uint256", label)
	}
	if positive && value.Sign() == 0 {
		return errors.Errorf("%s must be positive", label)
	}
	return nil
}

func pack(contractABI, method string, values ...any) ([]byte, error) {
	parsed, err := contractDefinition(contractABI)
	if err != nil {
		return nil, errors.Wrap(err, "parse contract ABI")
	}
	payload, err := parsed.Pack(method, values...)
	if err != nil {
		return nil, errors.Wrapf(err, "encode %s", method)
	}
	return payload, nil
}

// EncodeSplit spends pUSD from the caller and returns both V2 outcomes to that caller atomically.
func EncodeSplit(condition string, amount *big.Int) ([]byte, error) {
	id, err := ConditionID(condition)
	if err != nil {
		return nil, err
	}
	if err := validateUint(amount, true, "split amount"); err != nil {
		return nil, err
	}
	return pack(RouterABI, "split", id, amount)
}

// EncodeMerge spends an explicit amount of both outcomes and returns pUSD to the caller.
func EncodeMerge(condition string, amount *big.Int) ([]byte, error) {
	id, err := ConditionID(condition)
	if err != nil {
		return nil, err
	}
	if err := validateUint(amount, true, "merge amount"); err != nil {
		return nil, err
	}
	return pack(RouterABI, "merge", id, amount)
}

// EncodeRedeem redeems one explicit V2 outcome amount; YES is zero and NO is one.
func EncodeRedeem(condition string, outcomeIndex uint64, amount *big.Int) ([]byte, error) {
	if outcomeIndex >= binaryOutcomeCount {
		return nil, errors.New("redemption outcome must be zero or one")
	}
	id, err := ConditionID(condition)
	if err != nil {
		return nil, err
	}
	if err := validateUint(amount, true, "redeem amount"); err != nil {
		return nil, err
	}
	return pack(RouterABI, "redeem", id, new(big.Int).SetUint64(outcomeIndex), amount)
}

// EncodeERC20Approval grants the specified spender an exact pUSD allowance, including zero revocation.
func EncodeERC20Approval(spender common.Address, amount *big.Int) ([]byte, error) {
	if spender == (common.Address{}) {
		return nil, errors.New("approval spender must not be zero")
	}
	if err := validateUint(amount, false, "approval amount"); err != nil {
		return nil, err
	}
	return pack(CollateralABI, "approve", spender, amount)
}

// EncodeSetApprovalForAll grants or revokes a PositionManager operator permission.
func EncodeSetApprovalForAll(operator common.Address, approved bool) ([]byte, error) {
	if operator == (common.Address{}) {
		return nil, errors.New("approval operator must not be zero")
	}
	return pack(PositionManagerABI, "setApprovalForAll", operator, approved)
}

// EncodeTransfer produces a safe ERC-1155 V2 transfer, with explicit receiver data.
func EncodeTransfer(from, to common.Address, positionID, amount *big.Int, data []byte) ([]byte, error) {
	if from == (common.Address{}) || to == (common.Address{}) {
		return nil, errors.New("transfer owners must not be zero")
	}
	if err := validateUint(positionID, false, "position ID"); err != nil {
		return nil, err
	}
	if err := validateUint(amount, true, "transfer amount"); err != nil {
		return nil, err
	}
	return pack(PositionManagerABI, "safeTransferFrom", from, to, positionID, amount, data)
}

// EncodeBalanceOfBatch builds a standard ERC-1155 batch read for an externally pinned block hash.
func EncodeBalanceOfBatch(owners []common.Address, positionIDs []*big.Int) ([]byte, error) {
	if len(owners) == 0 || len(owners) != len(positionIDs) {
		return nil, errors.New("batch owners and position IDs must be nonempty and aligned")
	}
	for index, owner := range owners {
		if owner == (common.Address{}) {
			return nil, errors.Errorf("batch owner %d must not be zero", index)
		}
		if err := validateUint(positionIDs[index], false, "position ID"); err != nil {
			return nil, err
		}
	}
	return pack(PositionManagerABI, "balanceOfBatch", owners, positionIDs)
}
