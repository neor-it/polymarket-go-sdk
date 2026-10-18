package combos

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/neor-it/polymarket-go-sdk/pkg/data"
)

// Direction is the requester trade direction in a Combo RFQ.
type Direction string

const (
	// DirectionBuy requests buying Combo YES exposure.
	DirectionBuy Direction = "BUY"
	// DirectionSell requests selling Combo YES exposure.
	DirectionSell Direction = "SELL"
)

// ComboSide identifies the Combo outcome side. Builder RFQ currently accepts YES.
type ComboSide string

const (
	// ComboSideYes is the conjunction of all selected legs.
	ComboSideYes ComboSide = "YES"
	// ComboSideNo is the complement of the conjunction.
	ComboSideNo ComboSide = "NO"
)

// RequestedSizeUnit is the unit for requested_size.value_e6.
type RequestedSizeUnit string

const (
	// RequestedSizeUnitNotional means value_e6 is collateral budget in 6-decimal units.
	RequestedSizeUnitNotional RequestedSizeUnit = "notional"
	// RequestedSizeUnitShares means value_e6 is Combo shares in 6-decimal units.
	RequestedSizeUnitShares RequestedSizeUnit = "shares"
)

// RFQStatus is a Builder RFQ lifecycle status. Unknown values are preserved.
type RFQStatus string

const (
	RFQStatusCreated                     RFQStatus = "CREATED"
	RFQStatusCollectingQuotes            RFQStatus = "COLLECTING_QUOTES"
	RFQStatusAwaitingRequesterAcceptance RFQStatus = "AWAITING_REQUESTER_ACCEPTANCE"
	RFQStatusAwaitingMakerConfirmation   RFQStatus = "AWAITING_MAKER_CONFIRMATION"
	RFQStatusExecuting                   RFQStatus = "EXECUTING"
	RFQStatusMined                       RFQStatus = "MINED"
	RFQStatusRetrying                    RFQStatus = "RETRYING"
	RFQStatusConfirmed                   RFQStatus = "CONFIRMED"
	RFQStatusFilled                      RFQStatus = "FILLED"
	RFQStatusFailed                      RFQStatus = "FAILED"
	RFQStatusExpired                     RFQStatus = "EXPIRED"
	RFQStatusCanceled                    RFQStatus = "CANCELED"
	RFQStatusRejected                    RFQStatus = "REJECTED"
)

// ComboMarketListRequest filters public Combo market discovery.
type ComboMarketListRequest struct {
	Limit   int
	Cursor  string
	Exclude []string
}

// ComboMarketListResponse contains public Combo markets and the pagination cursor.
type ComboMarketListResponse struct {
	Markets    []ComboMarket `json:"markets"`
	NextCursor string        `json:"next_cursor,omitempty"`
}

// ComboMarket describes a public Combo market.
type ComboMarket struct {
	ID            string          `json:"id"`
	ConditionID   string          `json:"condition_id"`
	PositionIDs   []string        `json:"position_ids"`
	Slug          string          `json:"slug"`
	Title         string          `json:"title"`
	Outcomes      []string        `json:"outcomes"`
	OutcomePrices []string        `json:"outcome_prices"`
	Image         string          `json:"image,omitempty"`
	Volume        string          `json:"volume,omitempty"`
	Tags          []ComboTag      `json:"tags,omitempty"`
	Raw           json.RawMessage `json:"-"`
}

// ComboTag is a tag attached to a public Combo market.
type ComboTag struct {
	ID    string `json:"id,omitempty"`
	Label string `json:"label,omitempty"`
	Slug  string `json:"slug,omitempty"`
}

// RequestedSize is the size object sent to Builder RFQ.
type RequestedSize struct {
	Unit    RequestedSizeUnit `json:"unit"`
	ValueE6 string            `json:"value_e6"`
}

// CreateRFQRequest requests a Combo quote from the Builder Gateway.
type CreateRFQRequest struct {
	SignerAddress  string        `json:"signer_address"`
	MakerAddress   string        `json:"maker_address"`
	SignatureType  int           `json:"signature_type"`
	LegPositionIDs []string      `json:"leg_position_ids"`
	Direction      Direction     `json:"direction"`
	Side           ComboSide     `json:"side"`
	RequestedSize  RequestedSize `json:"requested_size"`
}

// CreateRFQResponse is returned by Builder RFQ create.
type CreateRFQResponse struct {
	RFQID       string          `json:"rfq_id,omitempty"`
	Status      RFQStatus       `json:"status"`
	ExpiresAt   string          `json:"expires_at,omitempty"`
	BuilderCode string          `json:"builder_code,omitempty"`
	Request     RFQRequestInfo  `json:"request,omitempty"`
	Quote       ComboQuote      `json:"quote,omitempty"`
	Error       string          `json:"error,omitempty"`
	Raw         json.RawMessage `json:"-"`
}

// RFQRequestInfo describes the normalized request returned by Builder RFQ.
type RFQRequestInfo struct {
	RFQID          string        `json:"rfq_id,omitempty"`
	YesPositionID  string        `json:"yes_position_id,omitempty"`
	NoPositionID   string        `json:"no_position_id,omitempty"`
	LegPositionIDs []string      `json:"leg_position_ids,omitempty"`
	Direction      Direction     `json:"direction,omitempty"`
	Side           ComboSide     `json:"side,omitempty"`
	RequestedSize  RequestedSize `json:"requested_size,omitempty"`
}

// ComboQuote is the maker quote selected by the Builder Gateway.
type ComboQuote struct {
	QuoteID         string `json:"quote_id,omitempty"`
	BlendedPriceE6  string `json:"blended_price_e6,omitempty"`
	MakerAmountE6   string `json:"maker_amount_e6,omitempty"`
	TakerAmountE6   string `json:"taker_amount_e6,omitempty"`
	TotalRequiredE6 string `json:"total_required_e6,omitempty"`
	NetReceiveE6    string `json:"net_receive_e6,omitempty"`
}

// AcceptRFQRequest accepts a quoted Combo RFQ.
type AcceptRFQRequest struct {
	RFQID       string           `json:"-"`
	QuoteID     string           `json:"quote_id"`
	SignedOrder ComboSignedOrder `json:"signed_order"`
}

// AcceptRFQResponse contains the accepted RFQ state.
type AcceptRFQResponse struct {
	RFQID  string    `json:"rfq_id,omitempty"`
	Status RFQStatus `json:"status"`
	Error  string    `json:"error,omitempty"`
}

// GetRFQResponse contains current RFQ execution status.
type GetRFQResponse struct {
	RFQID     string          `json:"rfq_id,omitempty"`
	Status    RFQStatus       `json:"status"`
	Error     string          `json:"error,omitempty"`
	TxHash    string          `json:"tx_hash,omitempty"`
	OrderHash string          `json:"order_hash,omitempty"`
	Quote     ComboQuote      `json:"quote,omitempty"`
	Request   RFQRequestInfo  `json:"request,omitempty"`
	Raw       json.RawMessage `json:"-"`
}

// ComboOrder is the Exchange v3 order shape used by Combo RFQ accept.
type ComboOrder struct {
	Salt          string `json:"salt"`
	Maker         string `json:"maker"`
	Signer        string `json:"signer"`
	TokenID       string `json:"tokenId"`
	MakerAmount   string `json:"makerAmount"`
	TakerAmount   string `json:"takerAmount"`
	Side          int    `json:"side"`
	SignatureType int    `json:"signatureType"`
	Timestamp     string `json:"timestamp"`
	Metadata      string `json:"metadata"`
	Builder       string `json:"builder"`
}

// ComboSignedOrder is the signed Exchange v3 order sent in RFQ accept.
type ComboSignedOrder struct {
	Salt          string `json:"salt"`
	Maker         string `json:"maker"`
	Signer        string `json:"signer"`
	TokenID       string `json:"tokenId"`
	MakerAmount   string `json:"makerAmount"`
	TakerAmount   string `json:"takerAmount"`
	Side          int    `json:"side"`
	SignatureType int    `json:"signatureType"`
	Timestamp     string `json:"timestamp"`
	Metadata      string `json:"metadata"`
	Builder       string `json:"builder"`
	Signature     string `json:"signature"`
}

// UserComboPositionsRequest filters cursor-paginated Data API v2 Combo positions.
type UserComboPositionsRequest struct {
	User          string
	Status        string
	SortBy        string
	SortDirection string
	Conditions    []common.Hash
	Limit         int
	UpdatedAfter  *int64
	UpdatedBefore *int64
	Cursor        string
}

// UserComboPositionsResponse is the canonical v2 Combo positions page.
type UserComboPositionsResponse = data.Page[data.ComboPosition]

// UserComboActivityRequest filters cursor-paginated Data API v2 Combo activity.
type UserComboActivityRequest struct {
	User       string
	Conditions []common.Hash
	Limit      int
	Cursor     string
}

// UserComboActivityResponse is the canonical v2 Combo activity page.
type UserComboActivityResponse = data.Page[data.ComboActivity]

// CollateralReturnRequest starts a collateral return action for eligible Combo positions.
type CollateralReturnRequest struct {
	UserAddress      string `json:"user_address"`
	ComboConditionID string `json:"combo_condition_id"`
	ComboPositionID  string `json:"combo_position_id"`
	AmountE6         string `json:"amount_e6"`
}

// CollateralReturnResponse contains the collateral return execution result.
type CollateralReturnResponse struct {
	Status string `json:"status"`
	TxHash string `json:"tx_hash,omitempty"`
	Error  string `json:"error,omitempty"`
}
