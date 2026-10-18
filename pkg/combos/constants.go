// Package combos provides a client for Polymarket Combo markets and Builder RFQ flows.
package combos

const (
	// DefaultPublicBaseURL is the production endpoint for public Combo RFQ market data.
	DefaultPublicBaseURL = "https://combos-rfq-api.polymarket.com"
	// DefaultDataBaseURL is the production endpoint for user Combo positions and activity.
	DefaultDataBaseURL = "https://data-api.polymarket.com"
	// DefaultExchangeV3Contract is the Polymarket Exchange v3 verifying contract used for Combo RFQ orders.
	DefaultExchangeV3Contract = "0xe3333700cA9d93003F00f0F71f8515005F6c00Aa"
)

const (
	comboMarketsPath     = "/v1/rfq/combo-markets"
	comboPositionsPath   = "/v2/positions/combos"
	comboActivityPath    = "/v2/activity/combos"
	builderRFQPath       = "/v1/builder/rfq/requests"
	collateralReturnPath = "/v1/builder/rfq/collateral-return"
)

const (
	exchangeDomainName    = "Polymarket CTF Exchange"
	exchangeDomainVersion = "3"
)
