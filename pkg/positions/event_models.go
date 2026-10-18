package positions

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// EventKind is the canonical deployed event name, including ERC-1155 events.
type EventKind string

const (
	KindTransferSingle                   EventKind = "TransferSingle"
	KindTransferBatch                    EventKind = "TransferBatch"
	KindApprovalForAll                   EventKind = "ApprovalForAll"
	KindMigrationConditionRegistered     EventKind = "MigrationConditionRegistered"
	KindPositionMigrated                 EventKind = "PositionMigrated"
	KindMigrationResolved                EventKind = "MigrationResolved"
	KindConditionResolved                EventKind = "ConditionResolved"
	KindRemainingConditionsDerivableAsNo EventKind = "RemainingConditionsDerivableAsNo"
	KindSyntheticConditionDerivableAsYes EventKind = "SyntheticConditionDerivableAsYes"
)

// LogEvent contains one validated event and its original reorg/transaction identity.
// Only Transfer changes holdings; lifecycle amounts are contextual and must not be applied twice.
type LogEvent struct {
	Kind           EventKind
	OperationName  string
	Raw            ethtypes.Log
	Transfer       *PositionTransfer
	Approval       *OperatorApproval
	Operation      *PositionOperation
	Migration      *PositionMigration
	Preparation    *EventPreparation
	Registration   *MigrationRegistration
	Resolution     *ConditionResolution
	AutoRedemption *AutoRedemption
}

// PositionTransfer is one ERC-1155 single or batch transfer, preserving all item indices.
type PositionTransfer struct {
	Operator, From, To   common.Address
	PositionIDs, Amounts []*big.Int
}

// OperatorApproval records one owner/operator permission change on the emitting ledger.
type OperatorApproval struct {
	Owner, Operator common.Address
	Approved        bool
}

// PositionOperation describes module/router context without mutating indexed balances.
type PositionOperation struct {
	Initiator, Recipient       common.Address
	Recipients                 []common.Address
	ConditionID, EventID       common.Hash
	PositionID, Amount, Payout *big.Int
}

// PositionMigration links a converted holding to its registered structural condition.
type PositionMigration struct {
	Owner                            common.Address
	ConditionID                      common.Hash
	PositionID, OutcomeIndex, Amount *big.Int
}

// MigrationRegistration is the authoritative old/new condition mapping emitted by a module.
type MigrationRegistration struct{ ConditionID, LegacyConditionID common.Hash }

// ConditionResolution records a reported, resolved or derivable result; derivation triggers require a fresh result read.
type ConditionResolution struct {
	ConditionID, EventID, LegacyConditionID common.Hash
	Result, LegacyPayouts                   []*big.Int
}

// AutoRedemption returns collateral to the same holder; approval alone does not schedule it.
type AutoRedemption struct {
	Owner       common.Address
	PositionID  *big.Int
	ConditionID common.Hash
	Payout      *big.Int
}

// EventPreparation links a structured event to legacy migration provenance or Combo legs.
type EventPreparation struct {
	EventID, ConditionID, LegacyEventID common.Hash
	ConditionCount                      *big.Int
	Legs                                []*big.Int
}
