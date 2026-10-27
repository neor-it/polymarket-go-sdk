package positions_test

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/neor-it/polymarket-go-sdk/pkg/positions"
)

func splitEventWord(value *big.Int) []byte {
	return value.FillBytes(make([]byte, common.HashLength))
}

func ctfSplitEvent(parent common.Hash) ethtypes.Log {
	cfg := positions.ConfigPolygon137()
	// Official CTF event: collateral, partition offset, amount, partition length/items.
	data := common.LeftPadBytes(common.HexToAddress(splitEventCollateral).Bytes(), common.HashLength)
	data = append(data, splitEventWord(big.NewInt(splitEventHeadWords*common.HashLength))...)
	data = append(data, splitEventWord(big.NewInt(splitEventAmount))...)
	data = append(data, splitEventWord(big.NewInt(splitEventTwo))...)
	data = append(data, splitEventWord(big.NewInt(splitEventOne))...)
	data = append(data, splitEventWord(big.NewInt(splitEventTwo))...)
	return ethtypes.Log{
		Address: cfg.ConditionalTokens,
		Topics: []common.Hash{
			crypto.Keccak256Hash([]byte(splitEventCTFSignature)),
			common.BytesToHash(common.HexToAddress(splitEventAdapter).Bytes()),
			parent,
			common.HexToHash("0xabcdef"),
		},
		Data:   data,
		TxHash: common.HexToHash("0x1234"),
	}
}

func TestCTFSplitEventPreservesCollateralParentAndPartition(t *testing.T) {
	for _, parent := range []common.Hash{{}, common.HexToHash("0x5678")} {
		log := ctfSplitEvent(parent)
		event, err := positions.DecodeLog(log, positions.ConfigPolygon137())
		if err != nil {
			t.Fatal(err)
		}
		if event == nil || event.OperationName != "split" || event.Operation == nil {
			t.Fatalf("CTF split event missing: %+v", event)
		}
		operation := event.Operation
		if operation.CollateralToken != common.HexToAddress(splitEventCollateral) || operation.ParentCollectionID != parent || operation.ConditionID != common.HexToHash("0xabcdef") || operation.Amount.Int64() != splitEventAmount {
			t.Fatalf("CTF split identity or amount changed: %+v", operation)
		}
		if len(operation.Partition) != splitEventTwo || operation.Partition[0].Int64() != splitEventOne || operation.Partition[splitEventOne].Int64() != splitEventTwo {
			t.Fatalf("CTF split partition changed: %v", operation.Partition)
		}
		// Adapter context must not be rewritten to the end user's wallet.
		if operation.Initiator != common.HexToAddress(splitEventAdapter) || operation.Recipient != operation.Initiator || event.Raw.TxHash != log.TxHash {
			t.Fatalf("CTF split caller or raw provenance changed: %+v", event)
		}
	}
}

func TestCTFSplitEventRejectsMalformedLogsAndIgnoresUnrelatedEmitters(t *testing.T) {
	cfg := positions.ConfigPolygon137()
	for _, malformed := range []func(*ethtypes.Log){
		func(log *ethtypes.Log) { log.Data = log.Data[:len(log.Data)-splitEventOne] },
		func(log *ethtypes.Log) { log.Topics = log.Topics[:len(log.Topics)-splitEventOne] },
		func(log *ethtypes.Log) { log.Topics[splitEventOne][0] = splitEventOne },
	} {
		log := ctfSplitEvent(common.Hash{})
		malformed(&log)
		if _, err := positions.DecodeLog(log, cfg); err == nil {
			t.Fatal("malformed CTF split event accepted")
		}
	}
	for _, source := range []common.Address{{}, cfg.Router, cfg.PositionManager, common.HexToAddress(splitEventAdapter)} {
		log := ctfSplitEvent(common.Hash{})
		log.Address = source
		event, err := positions.DecodeLog(log, cfg)
		if err != nil || event != nil {
			t.Fatalf("CTF split accepted from unrelated emitter %s: %+v, %v", source.Hex(), event, err)
		}
	}
}

func TestV2SplitEventRemainsDistinctFromCTF(t *testing.T) {
	cfg := positions.ConfigPolygon137()
	owner := common.HexToAddress(splitEventOwner)
	condition := common.HexToHash("0x" + strings.Repeat("ab", common.HashLength-splitEventOne) + "00")
	data := common.LeftPadBytes(owner.Bytes(), common.HashLength)
	data = append(data, splitEventWord(big.NewInt(splitEventAmount))...)
	event, err := positions.DecodeLog(ethtypes.Log{
		Address: cfg.BinaryModule,
		Topics: []common.Hash{
			crypto.Keccak256Hash([]byte(splitEventV2Signature)),
			common.BytesToHash(owner.Bytes()),
			condition,
			common.BytesToHash(owner.Bytes()),
		},
		Data: data,
	}, cfg)
	if err != nil || event == nil || event.Operation == nil {
		t.Fatalf("V2 split regression: %+v, %v", event, err)
	}
	operation := event.Operation
	if event.OperationName != "split" || operation.Initiator != owner || operation.ConditionID != condition || operation.Amount.Int64() != splitEventAmount || len(operation.Recipients) != splitEventTwo {
		t.Fatalf("V2 split identity changed: %+v", event)
	}
	if operation.CollateralToken != (common.Address{}) || operation.ParentCollectionID != (common.Hash{}) || operation.Partition != nil {
		t.Fatalf("V2 split unexpectedly inherited CTF fields: %+v", operation)
	}
}
