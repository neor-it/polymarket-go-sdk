package clob

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// TestMarshalTypedDataForSigningChainIDIsBareNumber is the regression test for
// the actual bug this investigation found: a wallet (Phantom) rejected
// eth_signTypedData_v4 payloads with "Missing or invalid parameters." because
// domain.chainId — via go-ethereum's default json.Marshal(apitypes.TypedData)
// — comes out as a quoted hex string ("0x89"), which MetaMask tolerates but
// Phantom's stricter validator does not.
func TestMarshalTypedDataForSigningChainIDIsBareNumber(t *testing.T) {
	td := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
			},
			"ClobAuth": {
				{Name: "address", Type: "address"},
			},
		},
		PrimaryType: "ClobAuth",
		Domain: apitypes.TypedDataDomain{
			Name:    "ClobAuth",
			Version: "1",
			ChainId: math.NewHexOrDecimal256(137),
		},
		Message: apitypes.TypedDataMessage{
			"address": "0x8686dEE12B17dA1a170a8f7a53B4d51a1A4f88bc",
		},
	}

	// Sanity check: go-ethereum's own marshaling really does quote chainId —
	// otherwise this test would not be exercising the bug it claims to guard.
	naive, err := json.Marshal(td)
	if err != nil {
		t.Fatalf("naive marshal: %v", err)
	}
	var naiveDoc struct {
		Domain struct {
			ChainID json.RawMessage `json:"chainId"`
		} `json:"domain"`
	}
	if err := json.Unmarshal(naive, &naiveDoc); err != nil {
		t.Fatalf("unmarshal naive: %v", err)
	}
	if string(naiveDoc.Domain.ChainID) != `"0x89"` {
		t.Fatalf("expected go-ethereum's default marshaling to quote chainId as a hex string, got %s — this test's premise may be stale", naiveDoc.Domain.ChainID)
	}

	fixed, err := MarshalTypedDataForSigning(td)
	if err != nil {
		t.Fatalf("MarshalTypedDataForSigning: %v", err)
	}

	var fixedDoc struct {
		Domain struct {
			ChainID json.RawMessage `json:"chainId"`
		} `json:"domain"`
	}
	if err := json.Unmarshal(fixed, &fixedDoc); err != nil {
		t.Fatalf("unmarshal fixed: %v", err)
	}
	if string(fixedDoc.Domain.ChainID) != "137" {
		t.Fatalf("domain.chainId = %s, want bare decimal 137", fixedDoc.Domain.ChainID)
	}
}

// TestMarshalTypedDataForSigningOmitsUndeclaredEmptyDomainFields checks the
// secondary fix: domain fields the EIP712Domain type doesn't declare (here,
// verifyingContract/salt on a domain that only declares name/version/chainId)
// must not appear in the wire payload at their Go zero value.
func TestMarshalTypedDataForSigningOmitsUndeclaredEmptyDomainFields(t *testing.T) {
	td := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
			},
			"ClobAuth": {
				{Name: "address", Type: "address"},
			},
		},
		PrimaryType: "ClobAuth",
		Domain: apitypes.TypedDataDomain{
			Name:    "ClobAuth",
			Version: "1",
			ChainId: math.NewHexOrDecimal256(137),
			// VerifyingContract and Salt left at their Go zero value ("") and
			// are NOT in the EIP712Domain type array above.
		},
		Message: apitypes.TypedDataMessage{
			"address": "0x8686dEE12B17dA1a170a8f7a53B4d51a1A4f88bc",
		},
	}

	payload, err := MarshalTypedDataForSigning(td)
	if err != nil {
		t.Fatalf("MarshalTypedDataForSigning: %v", err)
	}

	var doc struct {
		Domain map[string]json.RawMessage `json:"domain"`
	}
	if err := json.Unmarshal(payload, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, undeclared := range []string{"verifyingContract", "salt"} {
		if _, present := doc.Domain[undeclared]; present {
			t.Errorf("domain has undeclared field %q = %s, want it omitted", undeclared, doc.Domain[undeclared])
		}
	}
	if _, present := doc.Domain["chainId"]; !present {
		t.Errorf("domain is missing chainId, which is declared and non-empty")
	}
}

// TestMarshalTypedDataForSigningNormalizesEnvelopeMessageChainID covers the
// ERC-7739 TypedDataSign wrapper (BuildTypedDataSignEnvelope), which mirrors
// chainId a second time directly inside the message body, not just the
// top-level domain.
func TestMarshalTypedDataForSigningNormalizesEnvelopeMessageChainID(t *testing.T) {
	td := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "chainId", Type: "uint256"},
			},
			"TypedDataSign": {
				{Name: "chainId", Type: "uint256"},
			},
		},
		PrimaryType: "TypedDataSign",
		Domain: apitypes.TypedDataDomain{
			Name:    "DepositWallet",
			ChainId: math.NewHexOrDecimal256(137),
		},
		Message: apitypes.TypedDataMessage{
			"chainId": math.NewHexOrDecimal256(137),
		},
	}

	payload, err := MarshalTypedDataForSigning(td)
	if err != nil {
		t.Fatalf("MarshalTypedDataForSigning: %v", err)
	}

	var doc struct {
		Message struct {
			ChainID json.RawMessage `json:"chainId"`
		} `json:"message"`
	}
	if err := json.Unmarshal(payload, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(doc.Message.ChainID) != "137" {
		t.Fatalf("message.chainId = %s, want bare decimal 137", doc.Message.ChainID)
	}
}
