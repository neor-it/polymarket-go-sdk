package positions

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

// EncodeAutoRedeem submits full V2 holdings to the operator-only AutoRedeemer.
// Holder approval is necessary but does not schedule execution; outputs return to each holder.
func EncodeAutoRedeem(owners []common.Address, positionIDs []*big.Int) ([]byte, error) {
	if len(owners) == 0 || len(owners) != len(positionIDs) {
		return nil, errors.New("automatic redemption owners and positions must be nonempty and aligned")
	}
	for index, owner := range owners {
		if owner == (common.Address{}) {
			return nil, errors.New("automatic redemption owner must not be zero")
		}
		if err := validateUint(positionIDs[index], false, "position ID"); err != nil {
			return nil, err
		}
	}
	return pack(AutoRedeemerABI, "redeem", owners, positionIDs)
}
