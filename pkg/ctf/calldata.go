// Package ctf encodes Conditional Tokens calls and provides optional direct-chain clients.
package ctf

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

var splitContractABI, splitContractABIError = abi.JSON(strings.NewReader(conditionalTokensABI))

// EncodeSplitPosition encodes a CTF split without signing or submitting a transaction.
// The caller selects the execution target, collateral and partition. Prepared-condition
// slot membership remains an onchain check; zero parent collections are valid.
func EncodeSplitPosition(req *SplitPositionRequest) ([]byte, error) {
	if req == nil {
		return nil, errors.Wrap(ErrMissingRequest, "encode CTF split")
	}
	if req.Amount == nil {
		return nil, errors.Wrap(ErrMissingU256Value, "encode CTF split amount")
	}
	if req.Amount.Sign() <= 0 || req.Amount.BitLen() > splitUint256Bits {
		return nil, errors.Errorf("encode CTF split: amount must be a positive uint256")
	}
	if req.CollateralToken == (common.Address{}) {
		return nil, errors.Errorf("encode CTF split: collateral token must not be zero")
	}
	if len(req.Partition) < splitMinimumPartitionLength {
		return nil, errors.Errorf("encode CTF split: partition must contain disjoint outcome sets")
	}
	usedIndexSets := new(big.Int)
	for index, indexSet := range req.Partition {
		if indexSet == nil {
			return nil, errors.Wrapf(ErrMissingU256Value, "encode CTF split partition item %d", index)
		}
		if indexSet.Sign() <= 0 || indexSet.BitLen() > splitUint256Bits {
			return nil, errors.Errorf("encode CTF split: partition item %d must be a positive uint256", index)
		}
		if new(big.Int).And(usedIndexSets, indexSet).Sign() != 0 {
			return nil, errors.Errorf("encode CTF split: partition item %d overlaps another outcome set", index)
		}
		usedIndexSets.Or(usedIndexSets, indexSet)
	}
	if splitContractABIError != nil {
		return nil, errors.Wrap(splitContractABIError, "parse CTF split ABI")
	}
	payload, err := splitContractABI.Pack("splitPosition", req.CollateralToken, req.ParentCollectionID, req.ConditionID, req.Partition, req.Amount)
	if err != nil {
		return nil, errors.Wrap(err, "encode CTF split calldata")
	}
	return payload, nil
}
