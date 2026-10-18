package data

import "github.com/neor-it/polymarket-go-sdk/pkg/types"

// ApprovalContract represents the documented Data API v2 ApprovalContract response.
type ApprovalContract struct {
	Amount   *string `json:"amount"`
	Approved bool    `json:"approved"`
	Feature  string  `json:"feature"`
	ID       string  `json:"id"`
	Spender  string  `json:"spender"`
	Standard string  `json:"standard"`
	Token    string  `json:"token"`
}

// Approvals represents the documented Data API v2 Approvals response.
type Approvals struct {
	Address   string             `json:"address"`
	ChainID   int64              `json:"chain_id"`
	CheckedAt string             `json:"checked_at"`
	Contracts []ApprovalContract `json:"contracts"`
}

// Resolution represents the documented Data API v2 Resolution response.
type Resolution struct {
	ConditionID            *string `json:"condition_id"`
	ExpectedSettlementTime *string `json:"expected_settlement_time"`
	ExtendedReview         bool    `json:"extended_review"`
	LastUpdateTimestamp    string  `json:"last_update_timestamp"`
	LogIndex               string  `json:"log_index"`
	MarketType             *string `json:"market_type"`
	NewVersionQ            bool    `json:"new_version_q"`
	Payouts                []int64 `json:"payouts"`
	Price                  *string `json:"price"`
	ProposedPrice          *string `json:"proposed_price"`
	QuestionID             *string `json:"question_id"`
	Reporter               *string `json:"reporter"`
	ReproposedPrice        *string `json:"reproposed_price"`
	ResolutionSource       *string `json:"resolution_source"`
	ResolvedAt             *string `json:"resolved_at"`
	ResolvedBlock          *int64  `json:"resolved_block"`
	SettlementTimeBasis    *string `json:"settlement_time_basis"`
	Status                 string  `json:"status"`
	TransactionHash        string  `json:"transaction_hash"`
	WasArbitrated          *bool   `json:"was_arbitrated"`
	WasDisputed            bool    `json:"was_disputed"`
}

// TokenRef represents the documented Data API v2 TokenRef response.
type TokenRef struct {
	ClobIndex             *int64         `json:"clob_index"`
	Closed                *bool          `json:"closed"`
	ConditionID           string         `json:"condition_id"`
	EventID               *int64         `json:"event_id"`
	EventSlug             *string        `json:"event_slug"`
	FinalPrice            *types.Decimal `json:"final_price"`
	MarketSlug            *string        `json:"market_slug"`
	Module                TokenModule    `json:"module"`
	NegRisk               *bool          `json:"neg_risk"`
	NegRiskMarketID       *string        `json:"neg_risk_market_id"`
	OppositeTokenID       *string        `json:"opposite_token_id"`
	Outcome               *string        `json:"outcome"`
	OutcomeIndex          int64          `json:"outcome_index"`
	QuestionIndex         *int64         `json:"question_index"`
	Resolved              bool           `json:"resolved"`
	StructuralConditionID *string        `json:"structural_condition_id"`
	Title                 *string        `json:"title"`
	TokenID               string         `json:"token_id"`
}

// ServiceStatus represents the documented Data API v2 ServiceStatus response.
type ServiceStatus struct {
	AgeSeconds int64              `json:"age_seconds"`
	ComputedAt string             `json:"computed_at"`
	Ingestion  IngestionFreshness `json:"ingestion"`
	Serving    ServingFreshness   `json:"serving"`
}

// ServingFreshness represents the documented Data API v2 ServingFreshness response.
type ServingFreshness struct {
	LagSeconds *int64             `json:"lag_seconds"`
	Mechanisms []ServingMechanism `json:"mechanisms"`
	Worst      *string            `json:"worst"`
}

// ServingMechanism represents the documented Data API v2 ServingMechanism response.
type ServingMechanism struct {
	AgeSeconds   int64  `json:"age_seconds"`
	BlocksBehind *int64 `json:"blocks_behind"`
	Name         string `json:"name"`
}

// IngestionFreshness represents the documented Data API v2 IngestionFreshness response.
type IngestionFreshness struct {
	ChainID        int64       `json:"chain_id"`
	Cursors        int64       `json:"cursors"`
	Lagging        []CursorLag `json:"lagging"`
	MaxSyncedBlock *int64      `json:"max_synced_block"`
	MinSyncedBlock *int64      `json:"min_synced_block"`
	MostLagged     *CursorLag  `json:"most_lagged"`
	Network        *string     `json:"network"`
}

// CursorLag represents the documented Data API v2 CursorLag response.
type CursorLag struct {
	BehindMax int64  `json:"behind_max"`
	Block     int64  `json:"block"`
	Source    string `json:"source"`
}

// TokenModule identifies the ledger/module that issued an outcome.
type TokenModule string

// ModuleV1Ctf identifies the v1_ctf module.
const ModuleV1Ctf TokenModule = "v1_ctf"

// ModuleBinary identifies the binary module.
const ModuleBinary TokenModule = "binary"

// ModuleNegRisk identifies the neg_risk module.
const ModuleNegRisk TokenModule = "neg_risk"

// ModuleCombo identifies the combo module.
const ModuleCombo TokenModule = "combo"
