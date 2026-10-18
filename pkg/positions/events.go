package positions

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"
)

const eventTopicOffset = 1

// DecodeLog decodes only known events from the configured contract family.
// Unknown topics or unrelated emitters return nil without error. Recognized but
// malformed events return errors so an indexer cannot silently skip financial state.
func DecodeLog(log ethtypes.Log, cfg Config) (*LogEvent, error) {
	if len(log.Topics) == 0 {
		return nil, nil
	}
	parsed, err := contractDefinition(EventsABI)
	if err != nil {
		return nil, errors.Wrap(err, "parse position event ABI")
	}
	event, err := parsed.EventByID(log.Topics[0])
	if err != nil {
		return nil, nil
	}
	if !eventEmitterAllowed(event.RawName, log.Address, cfg) {
		return nil, nil
	}
	indexed := abi.Arguments{}
	for _, input := range event.Inputs {
		if input.Indexed {
			indexed = append(indexed, input)
		}
	}
	if len(log.Topics) != len(indexed)+eventTopicOffset {
		return nil, errors.Errorf("%s has an invalid topic count", event.RawName)
	}
	for index, input := range indexed {
		topic := log.Topics[index+eventTopicOffset]
		if input.Type.T == abi.FixedBytesTy {
			for _, value := range topic[input.Type.Size:] {
				if value != 0 {
					return nil, errors.Errorf("%s has noncanonical fixed-bytes topic padding", event.RawName)
				}
			}
		}
		if input.Type.T == abi.AddressTy {
			for _, value := range topic[:common.HashLength-common.AddressLength] {
				if value != 0 {
					return nil, errors.Errorf("%s has noncanonical address topic padding", event.RawName)
				}
			}
		}
	}
	values := make(map[string]any, len(event.Inputs))
	if err := event.Inputs.NonIndexed().UnpackIntoMap(values, log.Data); err != nil {
		return nil, errors.Wrapf(err, "decode %s event data", event.RawName)
	}
	if err := abi.ParseTopicsIntoMap(values, indexed, log.Topics[eventTopicOffset:]); err != nil {
		return nil, errors.Wrapf(err, "decode %s event topics", event.RawName)
	}
	result := &LogEvent{Kind: EventKind(event.RawName), Raw: log}
	switch result.Kind {
	case KindTransferSingle:
		result.Transfer = &PositionTransfer{Operator: eventAddress(values, "operator"), From: eventAddress(values, "from"), To: eventAddress(values, "to"), PositionIDs: []*big.Int{eventUint(values, "id")}, Amounts: []*big.Int{eventUint(values, "value")}}
	case KindTransferBatch:
		ids, _ := values["ids"].([]*big.Int)
		amounts, _ := values["values"].([]*big.Int)
		if len(ids) != len(amounts) {
			return nil, errors.New("ERC-1155 transfer batch IDs and amounts are misaligned")
		}
		result.Transfer = &PositionTransfer{Operator: eventAddress(values, "operator"), From: eventAddress(values, "from"), To: eventAddress(values, "to"), PositionIDs: ids, Amounts: amounts}
	case KindApprovalForAll:
		approved, _ := values["approved"].(bool)
		result.Approval = &OperatorApproval{Owner: eventAddress(values, "owner"), Operator: eventAddress(values, "operator"), Approved: approved}
	case "EventPrepared", "CombinatorialConditionPrepared":
		legs, _ := values["legs"].([]*big.Int)
		result.Preparation = &EventPreparation{EventID: eventHash(values, "eventId"), ConditionID: eventHash(values, "conditionId"), LegacyEventID: eventHash(values, "legacyEventId"), ConditionCount: eventUint(values, "conditionCount"), Legs: legs}
		result.OperationName = "prepare"
	case KindMigrationConditionRegistered:
		result.Registration = &MigrationRegistration{ConditionID: eventHash(values, "v2ConditionId"), LegacyConditionID: eventHash(values, "legacyConditionId")}
		result.OperationName = "migration"
	case KindPositionMigrated:
		result.Migration = &PositionMigration{Owner: eventAddress(values, "from"), ConditionID: eventHash(values, "conditionId"), PositionID: eventUint(values, "positionId"), OutcomeIndex: eventUint(values, "outcomeIndex"), Amount: eventUint(values, "amount")}
		result.OperationName = "migration"
	case KindMigrationResolved:
		result.Resolution = &ConditionResolution{ConditionID: eventHash(values, "conditionId"), LegacyConditionID: eventHash(values, "legacyConditionId"), Result: []*big.Int{eventUint(values, "result0"), eventUint(values, "result1")}, LegacyPayouts: []*big.Int{eventUint(values, "legacyPayout0"), eventUint(values, "legacyPayout1")}}
		result.OperationName = "resolution"
	case KindConditionResolved, "ResultReported", KindRemainingConditionsDerivableAsNo, KindSyntheticConditionDerivableAsYes:
		payouts, _ := values["result"].([]*big.Int)
		result.Resolution = &ConditionResolution{ConditionID: eventHash(values, "conditionId"), EventID: eventHash(values, "eventId"), Result: payouts}
		result.OperationName = "resolution"
	case "Redemption", "BinaryRedemption", "NegRiskRedemption":
		result.AutoRedemption = &AutoRedemption{Owner: eventAddress(values, "from"), PositionID: eventUint(values, "positionId"), ConditionID: eventHash(values, "conditionId"), Payout: eventUint(values, "payout")}
		result.OperationName = "redeem"
	default:
		initiator := eventAddress(values, "initiator")
		if initiator == (common.Address{}) {
			initiator = eventAddress(values, "user")
		}
		if initiator == (common.Address{}) {
			initiator = eventAddress(values, "redeemer")
		}
		if initiator == (common.Address{}) {
			initiator = eventAddress(values, "stakeholder")
		}

		recipients := []common.Address{}
		for _, field := range []string{"recipient0", "recipient1"} {
			if recipient := eventAddress(values, field); recipient != (common.Address{}) {
				recipients = append(recipients, recipient)
			}
		}
		result.Operation = &PositionOperation{Initiator: initiator, Recipient: eventAddress(values, "recipient"), Recipients: recipients, ConditionID: eventHash(values, "conditionId"), EventID: eventHash(values, "eventId"), PositionID: eventUint(values, "positionId"), Amount: eventUint(values, "amount"), Payout: eventUint(values, "payout")}
		if log.Address == cfg.ConditionalTokens {
			result.Operation.Recipient = initiator
		}
		name := event.RawName
		switch {
		case name == "PayoutRedemption" || strings.Contains(name, "Redeem"):
			result.OperationName = "redeem"
		case strings.Contains(name, "Merg"):
			result.OperationName = "merge"
		case strings.Contains(name, "Split"):
			result.OperationName = "split"
		case strings.Contains(name, "Convert"):
			result.OperationName = "convert"
		case strings.Contains(name, "Prepared"):
			result.OperationName = "prepare"
		default:
			result.OperationName = name
		}
	}
	return result, nil
}

func eventEmitterAllowed(name string, source common.Address, cfg Config) bool {
	if source == (common.Address{}) {
		return false
	}
	switch name {
	case "PayoutRedemption", "PositionsMerge", "PositionSplit":
		return source == cfg.ConditionalTokens
	case "TransferSingle", "TransferBatch", "ApprovalForAll":
		return source == cfg.PositionManager || source == cfg.ConditionalTokens
	case "Redemption", "BinaryRedemption", "NegRiskRedemption":
		return source == cfg.AutoRedeemer
	case "PositionMigrated", "MigrationConditionRegistered", "MigrationResolved", "LegacyCollateralSettled":
		return source == cfg.BinaryModule || source == cfg.NegRiskModule
	}
	if strings.HasPrefix(name, "Router") || name == "CombinatorialCollateralReturned" {
		return source == cfg.Router
	}
	return source == cfg.BinaryModule || source == cfg.NegRiskModule || source == cfg.CombinatorialModule
}
func eventAddress(values map[string]any, key string) common.Address {
	value, _ := values[key].(common.Address)
	return value
}
func eventUint(values map[string]any, key string) *big.Int {
	value, _ := values[key].(*big.Int)
	return value
}
func eventHash(values map[string]any, key string) common.Hash {
	switch value := values[key].(type) {
	case [32]byte:
		return common.Hash(value)
	case [31]byte:
		var result common.Hash
		copy(result[:], value[:])
		return result
	case [29]byte:
		var result common.Hash
		copy(result[:], value[:])
		return result
	default:
		return common.Hash{}
	}
}
