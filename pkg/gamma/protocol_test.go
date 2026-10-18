package gamma

import (
	"strings"
	"testing"
)

func TestTradingTokensExplicitLedger(t *testing.T) {
	tests := []struct {
		name      string
		version   ProtocolVersion
		ctf       string
		positions []string
		outcomes  string
		want      string
		invalid   bool
	}{
		{name: "V1 selects CTF", version: ProtocolV1, ctf: `["11","12"]`, positions: []string{"21", "22"}, outcomes: `["Yes","No"]`, want: "11"},
		{name: "V2 selects PositionManager", version: ProtocolV2, ctf: `["11","12"]`, positions: []string{"21", "22"}, outcomes: `["Yes","No"]`, want: "21"},
		{name: "missing version", ctf: `["11","12"]`, outcomes: `["Yes","No"]`, invalid: true},
		{name: "unknown version", version: "v3", positions: []string{"21", "22"}, outcomes: `["Yes","No"]`, invalid: true},
		{name: "V2 does not fallback", version: ProtocolV2, ctf: `["11","12"]`, outcomes: `["Yes","No"]`, invalid: true},
		{name: "mismatched outcomes", version: ProtocolV2, positions: []string{"21"}, outcomes: `["Yes","No"]`, invalid: true},
		{name: "missing outcome label", version: ProtocolV2, positions: []string{"21", "22"}, outcomes: `["","No"]`, invalid: true},
		{name: "hex token", version: ProtocolV2, positions: []string{"0x21", "22"}, outcomes: `["Yes","No"]`, invalid: true},
		{name: "negative token", version: ProtocolV2, positions: []string{"-1", "22"}, outcomes: `["Yes","No"]`, invalid: true},
		{name: "oversized token", version: ProtocolV2, positions: []string{strings.Repeat("9", 79), "22"}, outcomes: `["Yes","No"]`, invalid: true},
		{name: "duplicate numerical token", version: ProtocolV2, positions: []string{"021", "21"}, outcomes: `["Yes","No"]`, invalid: true},
		{name: "full uint256", version: ProtocolV2, positions: []string{"115792089237316195423570985008687907853269984665640564039457584007913129639935", "22"}, outcomes: `["Yes","No"]`, want: "115792089237316195423570985008687907853269984665640564039457584007913129639935"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			market := Market{Version: test.version, ClobTokenIds: test.ctf, PositionIDs: test.positions, Outcomes: test.outcomes, Tokens: []Token{{TokenID: "untrusted-preparsed"}}}
			tokens, err := market.TradingTokens()
			if test.invalid {
				if err == nil || market.ParsedTokens() != nil {
					t.Fatal("invalid market accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(tokens) != 2 || tokens[0].TokenID != test.want || tokens[0].Outcome != "Yes" {
				t.Fatalf("tokens %+v", tokens)
			}
		})
	}
}
