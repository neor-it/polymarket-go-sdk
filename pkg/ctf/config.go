package ctf

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

// Chain IDs.
const (
	PolygonChainID int64 = 137
	AmoyChainID    int64 = 80002
)

type contractConfig struct {
	ConditionalTokens common.Address
	NegRiskAdapter    *common.Address
}

var contractConfigs = map[int64]contractConfig{
	PolygonChainID: {
		ConditionalTokens: common.HexToAddress("0x4D97DCd97eC945f40cF65F87097ACe5EA0476045"),
	},
	AmoyChainID: {
		ConditionalTokens: common.HexToAddress("0x69308FB512518e39F9b16112fA8d994F4e2Bf8bB"),
	},
}

var negRiskConfigs = map[int64]contractConfig{
	PolygonChainID: {
		ConditionalTokens: common.HexToAddress("0x4D97DCd97eC945f40cF65F87097ACe5EA0476045"),
		NegRiskAdapter:    ptrAddress(common.HexToAddress("0xd91E80cF2E7be2e162c6513ceD06f1dD0dA35296")),
	},
	AmoyChainID: {
		ConditionalTokens: common.HexToAddress("0x69308FB512518e39F9b16112fA8d994F4e2Bf8bB"),
		NegRiskAdapter:    ptrAddress(common.HexToAddress("0xd91E80cF2E7be2e162c6513ceD06f1dD0dA35296")),
	},
}

func resolveConfig(chainID int64, negRisk bool) (contractConfig, bool) {
	if negRisk {
		cfg, ok := negRiskConfigs[chainID]
		return cfg, ok
	}
	cfg, ok := contractConfigs[chainID]
	return cfg, ok
}

func ptrAddress(addr common.Address) *common.Address {
	return &addr
}

// CollateralAdapterForChain returns the published pUSD CTF split/merge/redeem target.
// Negative-risk markets use their collateral adapter, not the legacy NegRiskAdapter.
// Only the published Polygon deployment is supported; testnet targets are not inferred.
func CollateralAdapterForChain(chainID int64, negRisk bool) (common.Address, error) {
	if chainID != PolygonChainID {
		return common.Address{}, errors.Wrapf(ErrConfigNotFound, "resolve pUSD CTF collateral adapter for chain %d", chainID)
	}
	if negRisk {
		return common.HexToAddress(polygonNegRiskCollateralAdapter), nil
	}
	return common.HexToAddress(polygonCollateralAdapter), nil
}
