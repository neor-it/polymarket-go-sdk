package clob

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// MarshalTypedDataForSigning renders typedData as the wire JSON sent to a
// browser wallet's eth_signTypedData_v4. It differs from a naive
// json.Marshal(typedData) in two ways that some wallets validate strictly —
// observed: Phantom's EVM signer rejects both with "Missing or invalid
// parameters." (JSON-RPC -32602), even though MetaMask silently tolerates
// them:
//
//  1. domain.chainId is emitted as a plain decimal JSON number (e.g. 137)
//     instead of go-ethereum's default quoted hex string (e.g. "0x89").
//     apitypes.TypedDataDomain.ChainId is *math.HexOrDecimal256, which only
//     implements MarshalText (common/math/big.go), so encoding/json always
//     quotes it — there is no way to get a bare number out of the struct's
//     own MarshalJSON.
//  2. Domain fields the payload's own EIP712Domain type doesn't declare
//     (e.g. an empty verifyingContract/salt on a domain that only declares
//     name/version/chainId) are omitted, via TypedData.Domain.Map()'s
//     existing non-empty filtering, instead of go-ethereum's raw struct
//     marshaling always including every field at its Go zero value.
//
// This only changes the wire representation sent for signing. Hash
// computation (TypedDataAndHash) and our own signature verification always
// operate on typedData directly or reconstruct it from the same numeric
// chainID input — never on this marshaled JSON — so this has no effect on
// signature validity. Round-tripping back through Go is also unaffected:
// HexOrDecimal256.UnmarshalJSON already accepts a bare decimal number, not
// just a quoted hex string.
func MarshalTypedDataForSigning(typedData apitypes.TypedData) (json.RawMessage, error) {
	dataMap := typedData.Map()

	if domain, ok := dataMap["domain"].(map[string]interface{}); ok {
		if _, hasChainID := domain["chainId"]; hasChainID {
			domain["chainId"] = chainIDAsDecimal(typedData.Domain.ChainId)
		}
	}

	// The ERC-7739 TypedDataSign wrapper (see BuildTypedDataSignEnvelope) embeds
	// a mirrored chainId directly in the message body, not just the domain;
	// normalize that copy too when present.
	if message, ok := dataMap["message"].(map[string]interface{}); ok {
		if hex, ok := message["chainId"].(*math.HexOrDecimal256); ok {
			message["chainId"] = chainIDAsDecimal(hex)
		}
	}

	return json.Marshal(dataMap)
}

// chainIDAsDecimal returns chainID as a json.Number so encoding/json emits it
// as a bare, unquoted decimal integer.
func chainIDAsDecimal(chainID *math.HexOrDecimal256) json.Number {
	if chainID == nil {
		return "0"
	}
	return json.Number((*big.Int)(chainID).String())
}
