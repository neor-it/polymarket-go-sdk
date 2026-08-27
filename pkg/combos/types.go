package combos

import "encoding/json"

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

// UserComboPositionsRequest filters user Combo positions.
type UserComboPositionsRequest struct {
	User          string
	Status        string
	SortBy        string
	SortDirection string
	MarketID      string
	Limit         int
	Offset        int
	UpdatedAfter  string
	UpdatedBefore string
	Cursor        string
}

// UserComboPositionsResponse contains user Combo positions.
type UserComboPositionsResponse struct {
	Positions  []UserComboPosition `json:"positions"`
	NextCursor string              `json:"next_cursor,omitempty"`
}

// UserComboPosition describes a user Combo position from Data API.
type UserComboPosition struct {
	ComboConditionID string             `json:"combo_condition_id,omitempty"`
	ComboPositionID  string             `json:"combo_position_id,omitempty"`
	MarketID         string             `json:"market_id,omitempty"`
	Title            string             `json:"title,omitempty"`
	Status           string             `json:"status,omitempty"`
	Outcome          string             `json:"outcome,omitempty"`
	SizeE6           string             `json:"size_e6,omitempty"`
	ValueE6          string             `json:"value_e6,omitempty"`
	UpdatedAt        string             `json:"updated_at,omitempty"`
	Legs             []ComboPositionLeg `json:"legs,omitempty"`
	Raw              json.RawMessage    `json:"-"`
}

// ComboPositionLeg describes one leg inside a user Combo position.
type ComboPositionLeg struct {
	ConditionID string `json:"condition_id,omitempty"`
	PositionID  string `json:"position_id,omitempty"`
	MarketSlug  string `json:"market_slug,omitempty"`
	MarketTitle string `json:"market_title,omitempty"`
	Outcome     string `json:"outcome,omitempty"`
}

// UserComboActivityRequest filters user Combo activity.
type UserComboActivityRequest struct {
	User          string
	Type          string
	MarketID      string
	Limit         int
	Offset        int
	UpdatedAfter  string
	UpdatedBefore string
	Cursor        string
}

// UserComboActivityResponse contains user Combo activity entries.
type UserComboActivityResponse struct {
	Activity   []UserComboActivity `json:"activity"`
	NextCursor string              `json:"next_cursor,omitempty"`
}

// UserComboActivity describes a combo activity row.
type UserComboActivity struct {
	ID               string             `json:"id,omitempty"`
	Type             string             `json:"type,omitempty"`
	ComboConditionID string             `json:"combo_condition_id,omitempty"`
	ComboPositionID  string             `json:"combo_position_id,omitempty"`
	AmountE6         string             `json:"amount_e6,omitempty"`
	TxHash           string             `json:"tx_hash,omitempty"`
	Timestamp        string             `json:"timestamp,omitempty"`
	Legs             []ComboPositionLeg `json:"legs,omitempty"`
	Raw              json.RawMessage    `json:"-"`
}

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
