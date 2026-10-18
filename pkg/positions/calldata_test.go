package positions

import (
	"bytes"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestRouterCalldataPreservesConditionAndExplicitOutcomeAmount(t *testing.T) {
	native := "0x" + strings.Repeat("ab", 31)
	amount := big.NewInt(1000001)
	split, err := EncodeSplit(native, amount)
	if err != nil {
		t.Fatal(err)
	}
	merge, err := EncodeMerge(native+"00", amount)
	if err != nil {
		t.Fatal(err)
	}
	redeem, err := EncodeRedeem(native, 1, amount)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		payload   []byte
		signature string
	}{{split, "split(bytes31,uint256)"}, {merge, "merge(bytes31,uint256)"}, {redeem, "redeem(bytes31,uint256,uint256)"}} {
		selector := crypto.Keccak256([]byte(test.signature))[:4]
		if !bytes.Equal(test.payload[:4], selector) || !bytes.Equal(test.payload[4:35], bytes.Repeat([]byte{0xab}, 31)) || test.payload[35] != 0 {
			t.Fatalf("incorrect selector or condition padding: %x", test.payload)
		}
	}
	if new(big.Int).SetBytes(redeem[36:68]).Uint64() != 1 || new(big.Int).SetBytes(redeem[68:100]).Cmp(amount) != 0 {
		t.Fatalf("explicit outcome/amount changed: %x", redeem)
	}
	for _, condition := range []string{native + "01", "0x12"} {
		if _, err := EncodeSplit(condition, amount); err == nil {
			t.Fatal("bad condition accepted")
		}
	}
	if _, err := EncodeRedeem(native, 2, amount); err == nil {
		t.Fatal("CTF index set accepted as V2 outcome")
	}
	for _, invalid := range []*big.Int{nil, big.NewInt(-1), big.NewInt(0), new(big.Int).Lsh(big.NewInt(1), 256)} {
		if _, err := EncodeMerge(native, invalid); err == nil {
			t.Fatal("invalid amount accepted")
		}
	}
}

func TestMigrationOverloadsAndCallerSorting(t *testing.T) {
	conditions := []common.Hash{common.HexToHash("0x11"), common.HexToHash("0x11"), common.HexToHash("0x22")}
	outcomes := []uint64{0, 1, 0}
	amounts := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3)}
	own, err := EncodeMigratePositions(conditions, outcomes, amounts)
	if err != nil {
		t.Fatal(err)
	}
	operator, err := EncodeOperatorMigratePositions(common.HexToAddress("0x1"), conditions, outcomes, amounts)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(own[:4], crypto.Keccak256([]byte("migratePositions(bytes32[],uint256[],uint256[])"))[:4]) || !bytes.Equal(operator[:4], crypto.Keccak256([]byte("migratePositions(address,bytes32[],uint256[],uint256[])"))[:4]) {
		t.Fatal("migration overload selector changed")
	}
	if _, err := EncodeMigratePositions([]common.Hash{conditions[2], conditions[0]}, []uint64{0, 1}, amounts[:2]); err == nil {
		t.Fatal("unsorted migration silently reordered")
	}
	if _, err := EncodeMigratePositions(conditions, outcomes[:1], amounts); err == nil {
		t.Fatal("misaligned migration accepted")
	}
	if _, err := EncodeAutoRedeem([]common.Address{common.HexToAddress("0x1")}, []*big.Int{new(big.Int).Lsh(big.NewInt(1), 256)}); err == nil {
		t.Fatal("oversized automatic redemption accepted")
	}
}

func TestApprovalAndBatchReadUseExactContractSignatures(t *testing.T) {
	cfg := ConfigPolygon137()
	approval, err := EncodeSetApprovalForAll(cfg.Exchange, true)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(approval[:4], crypto.Keccak256([]byte("setApprovalForAll(address,bool)"))[:4]) {
		t.Fatal("operator approval selector")
	}
	amount := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	collateral, err := EncodeERC20Approval(cfg.Router, amount)
	if err != nil {
		t.Fatal(err)
	}
	if new(big.Int).SetBytes(collateral[36:68]).Cmp(amount) != 0 {
		t.Fatal("uint256 allowance truncated")
	}
	if _, err := EncodeBalanceOfBatch([]common.Address{common.HexToAddress("0x1")}, []*big.Int{amount}); err != nil {
		t.Fatal(err)
	}
	if _, err := EncodeBalanceOfBatch([]common.Address{common.HexToAddress("0x1")}, nil); err == nil {
		t.Fatal("misaligned batch accepted")
	}
}

func TestPositionIdentityUsesNativeWidths(t *testing.T) {
	var condition common.Hash
	condition[0] = 2
	condition[18] = 3
	condition[29] = 1
	condition[30] = 2
	id, err := PositionID(condition.Hex(), 1)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := DecodePositionID(id)
	if err != nil {
		t.Fatal(err)
	}
	if identity.ModuleID != 2 || identity.Arity != 3 || identity.OutcomeIndex != 1 || identity.ConditionIndex != 258 || identity.ConditionID != condition || identity.EventID[29] != 0 || identity.EventID[30] != 0 || identity.EventID[31] != 0 {
		t.Fatalf("identity %+v", identity)
	}
}
