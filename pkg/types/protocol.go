package types

// ProtocolVersion identifies the outcome ledger and exchange signing domain.
type ProtocolVersion string

const (
	// ProtocolV1 identifies legacy ConditionalTokens positions.
	ProtocolV1 ProtocolVersion = "v1"
	// ProtocolV2 identifies PositionManager positions.
	ProtocolV2 ProtocolVersion = "v2"
)
