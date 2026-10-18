package clob

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/neor-it/polymarket-go-sdk/pkg/positions"
	"github.com/neor-it/polymarket-go-sdk/pkg/types"
	"github.com/pkg/errors"
)

const (
	ctfDomainVersion      = "2"
	positionDomainVersion = "3"
)

// SigningConfig binds an order to the selected protocol's exchange and EIP-712 domain.
type SigningConfig struct {
	ProtocolVersion types.ProtocolVersion
	Exchange        common.Address
	DomainVersion   string
}

// SigningConfiguration resolves signing identity without inferring a protocol from token IDs.
// Empty protocol is accepted only for historical CTF order compatibility. Gamma
// market selection must require an explicit supported version before building an order.
func SigningConfiguration(chainID int64, protocol types.ProtocolVersion, negRisk bool) (SigningConfig, error) {
	if protocol == "" {
		protocol = types.ProtocolV1
	}
	switch protocol {
	case types.ProtocolV1:
		return SigningConfig{ProtocolVersion: protocol, Exchange: common.HexToAddress(verifyingContractV2(negRisk)), DomainVersion: ctfDomainVersion}, nil
	case types.ProtocolV2:
		if chainID != positions.PolygonChainID {
			return SigningConfig{}, errors.Errorf("Protocol V2 signing is unsupported on chain %d", chainID)
		}
		return SigningConfig{ProtocolVersion: protocol, Exchange: common.HexToAddress(positions.ExchangeV3Address), DomainVersion: positionDomainVersion}, nil
	default:
		return SigningConfig{}, errors.Errorf("unsupported order protocol version %q", protocol)
	}
}
