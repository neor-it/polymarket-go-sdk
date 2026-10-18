package positions

import (
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func eventForTest(t *testing.T, name string, source common.Address, indexed []common.Hash, args ...any) ethtypes.Log {
	t.Helper()
	parsed, err := contractDefinition(EventsABI)
	if err != nil {
		t.Fatal(err)
	}
	event := parsed.Events[name]
	data, err := event.Inputs.NonIndexed().Pack(args...)
	if err != nil {
		t.Fatal(err)
	}
	return ethtypes.Log{Address: source, Topics: append([]common.Hash{event.ID}, indexed...), Data: data}
}
func addressTopic(address common.Address) common.Hash { return common.BytesToHash(address.Bytes()) }

func TestTransferSingleAndBatchAreCanonicalBalanceInputs(t *testing.T) {
	cfg := ConfigPolygon137()
	owner := common.HexToAddress("0x123")
	recipient := common.HexToAddress("0x456")
	id := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	amount := big.NewInt(1000001)
	for _, source := range []common.Address{cfg.PositionManager, cfg.ConditionalTokens} {
		single := eventForTest(t, "TransferSingle", source, []common.Hash{addressTopic(cfg.Router), addressTopic(owner), addressTopic(recipient)}, id, amount)
		decoded, err := DecodeLog(single, cfg)
		if err != nil || decoded == nil {
			t.Fatalf("decode %v", err)
		}
		if decoded.Transfer.From != owner || decoded.Transfer.To != recipient || decoded.Transfer.PositionIDs[0].Cmp(id) != 0 || decoded.Transfer.Amounts[0].Cmp(amount) != 0 {
			t.Fatalf("transfer %+v", decoded.Transfer)
		}
		batch := eventForTest(t, "TransferBatch", source, single.Topics[1:], []*big.Int{id, big.NewInt(1)}, []*big.Int{amount, big.NewInt(0)})
		decoded, err = DecodeLog(batch, cfg)
		if err != nil || len(decoded.Transfer.PositionIDs) != 2 || decoded.Transfer.Amounts[1].Sign() != 0 {
			t.Fatalf("batch %+v %v", decoded, err)
		}
		batch.Data = []byte{1}
		if _, err := DecodeLog(batch, cfg); err == nil {
			t.Fatal("malformed event swallowed")
		}
		single.Address = cfg.Collateral
		if result, err := DecodeLog(single, cfg); err != nil || result != nil {
			t.Fatal("unrelated emitter treated as position ledger")
		}
	}
}

func TestMigrationAndRedemptionKeepOwnersAndZeroPayout(t *testing.T) {
	cfg := ConfigPolygon137()
	owner := common.HexToAddress("0x123")
	var condition common.Hash
	condition[0] = 1
	legacy := common.HexToHash("0x123456")
	registration := eventForTest(t, "MigrationConditionRegistered", cfg.BinaryModule, []common.Hash{condition, legacy})
	decoded, err := DecodeLog(registration, cfg)
	if err != nil || decoded.Registration.ConditionID != condition || decoded.Registration.LegacyConditionID != legacy {
		t.Fatalf("mapping %+v %v", decoded, err)
	}
	migrated := eventForTest(t, "PositionMigrated", cfg.BinaryModule, []common.Hash{addressTopic(owner), condition, common.BigToHash(big.NewInt(123))}, big.NewInt(1), big.NewInt(1000001))
	decoded, err = DecodeLog(migrated, cfg)
	if err != nil || decoded.Migration.Owner != owner || decoded.Migration.OutcomeIndex.Int64() != 1 {
		t.Fatalf("migration %+v %v", decoded, err)
	}
	redeemed := eventForTest(t, "PositionRedeemed", cfg.NegRiskModule, []common.Hash{addressTopic(cfg.Router), common.BigToHash(big.NewInt(123)), addressTopic(owner)}, big.NewInt(1000001), big.NewInt(0))
	decoded, err = DecodeLog(redeemed, cfg)
	if err != nil || decoded.Operation.Initiator != cfg.Router || decoded.Operation.Recipient != owner || decoded.Operation.Payout == nil || decoded.Operation.Payout.Sign() != 0 || decoded.Transfer != nil || decoded.OperationName != "redeem" {
		t.Fatalf("redeem %+v %v", decoded, err)
	}
	automatic := eventForTest(t, "Redemption", cfg.AutoRedeemer, []common.Hash{addressTopic(owner), common.BigToHash(big.NewInt(123))}, big.NewInt(1))
	decoded, err = DecodeLog(automatic, cfg)
	if err != nil || decoded.AutoRedemption.Owner != owner || decoded.AutoRedemption.Payout.Int64() != 1 {
		t.Fatalf("automatic %+v %v", decoded, err)
	}
	registration.Topics[1][31] = 1
	if _, err := DecodeLog(registration, cfg); err == nil {
		t.Fatal("noncanonical condition padding accepted")
	}
}

func TestPinnedEventContractMatchesExportedABI(t *testing.T) {
	snapshot, err := os.ReadFile("testdata/events.json")
	if err != nil {
		t.Fatal(err)
	}
	pinned, err := abi.JSON(strings.NewReader(string(snapshot)))
	if err != nil {
		t.Fatal(err)
	}
	current, err := contractDefinition(EventsABI)
	if err != nil {
		t.Fatal(err)
	}
	if len(pinned.Events) != len(current.Events) {
		t.Fatal("deployed event coverage changed without updating contract fixture")
	}
	for name, event := range pinned.Events {
		actual, ok := current.Events[name]
		if !ok || actual.ID != event.ID {
			t.Fatalf("event signature drift: %s", name)
		}
		if len(actual.Inputs) != len(event.Inputs) {
			t.Fatalf("event arity drift: %s", name)
		}
		for index, input := range event.Inputs {
			if actual.Inputs[index].Name != input.Name || actual.Inputs[index].Indexed != input.Indexed {
				t.Fatalf("indexed payload drift: %s", name)
			}
		}
	}
}

func TestLegacyCTFLifecycleCannotBecomeDepositContext(t *testing.T) {
	cfg := ConfigPolygon137()
	holder := common.HexToAddress("0x123")
	condition := common.HexToHash("0x123456")
	redemption := eventForTest(t, "PayoutRedemption", cfg.ConditionalTokens, []common.Hash{addressTopic(holder), addressTopic(cfg.Collateral), {}}, [32]byte(condition), []*big.Int{big.NewInt(1), big.NewInt(2)}, big.NewInt(1000001))
	if redemption.Topics[0].Hex() != "0x2682012a4a4f1973119f1c9b90745d1bd91fa2bab387344f044cb3586864d18d" {
		t.Fatal("CTF redemption topic differs from deployed event")
	}
	decoded, err := DecodeLog(redemption, cfg)
	if err != nil || decoded.OperationName != "redeem" || decoded.Operation.Initiator != holder || decoded.Operation.Recipient != holder || decoded.Operation.ConditionID != condition || decoded.Operation.Payout.Int64() != 1000001 {
		t.Fatalf("CTF redeem %+v %v", decoded, err)
	}
	for _, name := range []string{"PositionSplit", "PositionsMerge"} {
		log := eventForTest(t, name, cfg.ConditionalTokens, []common.Hash{addressTopic(holder), {}, condition}, cfg.Collateral, []*big.Int{big.NewInt(1), big.NewInt(2)}, big.NewInt(1000000))
		decoded, err := DecodeLog(log, cfg)
		if err != nil || decoded.Operation.Initiator != holder || decoded.Operation.Recipient != holder {
			t.Fatalf("CTF operation %+v %v", decoded, err)
		}
	}
}
