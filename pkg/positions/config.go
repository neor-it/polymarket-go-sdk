// Package positions encodes and reads Polymarket Protocol V2 position contracts.
// It never submits transactions automatically or converts existing CTF holdings.
package positions

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

const (
	// PolygonChainID is the supported chain for the published Protocol V2 deployment.
	PolygonChainID int64 = 137
	// ExchangeV3Address is the published Polygon exchange proxy, not its implementation.
	ExchangeV3Address          = "0xe3333700cA9d93003F00f0F71f8515005F6c00Aa"
	positionManagerAddress     = "0x006F54F7f9A22e0000CC2AB60031000000ae9fEF"
	routerAddress              = "0x12121212006e4CD160D18e3f00711DA5c3372600"
	collateralAddress          = "0xC011a7E12a19f7B1f670d46F03B03f3342E82DFB"
	binaryModuleAddress        = "0x1000008dD9001B968442c1000017eaE6E0dA00Ba"
	negRiskModuleAddress       = "0x200000900045e3B6259600682756002200028933"
	combinatorialModuleAddress = "0x30000034706C7d8e12009DAB006Be20000c031A8"
	conditionalTokensAddress   = "0x4D97DCd97eC945f40cF65F87097ACe5EA0476045"
	autoRedeemerAddress        = "0xa1200000d0002264C9a1698e001292D00E1b00af"
)

// Config contains proxy addresses for one V2 deployment.
type Config struct {
	ConditionalTokens   common.Address
	ChainID             int64
	PositionManager     common.Address
	Router              common.Address
	Collateral          common.Address
	Exchange            common.Address
	BinaryModule        common.Address
	NegRiskModule       common.Address
	CombinatorialModule common.Address
	AutoRedeemer        common.Address
}

// ConfigPolygon137 returns the published Polygon proxies from the official contract catalog.
func ConfigPolygon137() Config {
	return Config{ConditionalTokens: common.HexToAddress(conditionalTokensAddress), ChainID: PolygonChainID, PositionManager: common.HexToAddress(positionManagerAddress), Router: common.HexToAddress(routerAddress), Collateral: common.HexToAddress(collateralAddress), Exchange: common.HexToAddress(ExchangeV3Address), BinaryModule: common.HexToAddress(binaryModuleAddress), NegRiskModule: common.HexToAddress(negRiskModuleAddress), CombinatorialModule: common.HexToAddress(combinatorialModuleAddress), AutoRedeemer: common.HexToAddress(autoRedeemerAddress)}
}

func (cfg Config) validate() error {
	if cfg.ChainID != PolygonChainID {
		return errors.Errorf("Protocol V2 deployment is unsupported on chain %d", cfg.ChainID)
	}
	for _, address := range []common.Address{cfg.PositionManager, cfg.Router, cfg.Collateral, cfg.Exchange, cfg.BinaryModule, cfg.NegRiskModule, cfg.CombinatorialModule, cfg.AutoRedeemer} {
		if address == (common.Address{}) {
			return errors.New("Protocol V2 deployment contains a zero contract address")
		}
	}
	return nil
}

// ConfigForChain returns a supported Protocol V2 deployment or an explicit error.
func ConfigForChain(chainID int64) (Config, error) {
	if chainID != PolygonChainID {
		return Config{}, errors.Errorf("Protocol V2 deployment is unsupported on chain %d", chainID)
	}
	return ConfigPolygon137(), nil
}
