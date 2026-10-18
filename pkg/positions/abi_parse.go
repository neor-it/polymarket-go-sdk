package positions

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/pkg/errors"
)

type parsedContract struct {
	value abi.ABI
	err   error
}

// The static ABI registry is initialized once and remains immutable for concurrent readers.
var contractDefinitions = map[string]parsedContract{
	PositionManagerABI: parseContract(PositionManagerABI),
	RouterABI:          parseContract(RouterABI),
	CollateralABI:      parseContract(CollateralABI),
	ModuleABI:          parseContract(ModuleABI),
	AutoRedeemerABI:    parseContract(AutoRedeemerABI),
	EventsABI:          parseContract(EventsABI),
}

func parseContract(source string) parsedContract {
	value, err := abi.JSON(strings.NewReader(source))
	return parsedContract{value: value, err: err}
}
func contractDefinition(source string) (abi.ABI, error) {
	definition, exists := contractDefinitions[source]
	if !exists {
		return abi.ABI{}, errors.New("unregistered position contract ABI")
	}
	if definition.err != nil {
		return abi.ABI{}, errors.Wrap(definition.err, "parse published position ABI")
	}
	return definition.value, nil
}
