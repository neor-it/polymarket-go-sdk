package clob

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	exchangeV2Address        = "0xE111180000d2663C0091e4f400237545B87B996B"
	exchangeV2NegRiskAddress = "0xe2222d279d744050d28e00520010520000310F59"
)

func verifyingContractV2(negRisk bool) string {
	if negRisk {
		return exchangeV2NegRiskAddress
	}
	return exchangeV2Address
}

func parseBuilderCodeString(code string) (common.Hash, error) {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return common.Hash{}, nil
	}
	if !strings.HasPrefix(trimmed, "0x") {
		trimmed = "0x" + trimmed
	}
	decoded, err := hexutil.Decode(trimmed)
	if err != nil {
		return common.Hash{}, fmt.Errorf("builder code: %w", err)
	}
	if len(decoded) != common.HashLength {
		return common.Hash{}, fmt.Errorf("builder code: want 32 bytes, got %d", len(decoded))
	}
	var hash common.Hash
	copy(hash[:], decoded)
	return hash, nil
}
