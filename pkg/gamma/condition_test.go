package gamma

import (
	"github.com/ethereum/go-ethereum/common"
	"strings"
	"testing"
)

func TestConditionHashKeepsV2ByteOrder(t *testing.T) {
	native := "0x" + strings.Repeat("12", 31)
	padded := native + "00"
	fromNative, err := ConditionHash(ProtocolV2, native)
	if err != nil {
		t.Fatal(err)
	}
	fromPadded, err := ConditionHash(ProtocolV2, padded)
	if err != nil {
		t.Fatal(err)
	}
	if fromNative != fromPadded || fromNative.Hex() != padded {
		t.Fatalf("condition altered %s %s", fromNative.Hex(), fromPadded.Hex())
	}
	if fromNative == common.HexToHash(native) {
		t.Fatal("native ID was incorrectly left-padded")
	}
	for _, value := range []string{native + "01", "0x12", strings.TrimPrefix(native, "0x"), "0x" + strings.Repeat("00", 32), "0xgg"} {
		if _, err := ConditionHash(ProtocolV2, value); err == nil {
			t.Errorf("invalid condition accepted %s", value)
		}
	}
	if _, err := ConditionHash(ProtocolV1, native); err == nil {
		t.Fatal("CTF accepted native bytes31")
	}
	if _, err := ConditionHash("", padded); err == nil {
		t.Fatal("missing version accepted")
	}
}
