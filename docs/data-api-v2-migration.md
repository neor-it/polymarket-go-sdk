# SDK v0.2.0: Data API v2 migration

This release includes the existing Combos implementation from `main` and moves
all retiring Data API reads, including Combo feeds, to `/v2`.

## Breaking Go API changes

- List reads return typed pages: `response.Data` and `response.Pagination`.
  Keep the original query unchanged, set `request.Cursor` to
  `response.Pagination.NextCursor`, and stop when that cursor is nil. A single
  request returns exactly one page. There is no offset emulation or hidden
  page cap. Re-send title, conditions, user and time-window filters on every page.
- Requests no longer have `Offset`. `Pagination.Offset` is diagnostic metadata
  from the server. Condition IDs are sent as `condition`, at most 20 distinct IDs.
- Positions use `PositionsRequest.Status`, including `CLOSED`, `REDEEMABLE_LOST`
  and `MERGEABLE`. The old `ClosedPositions` endpoint is removed. Use
  `TradeFilterTokens`/`TradeFilterCash` instead of `SizeThreshold`.
- Position economics follow the v2 contract: `CurrentSize` is the holding,
  `TotalSize` is lifetime purchased shares, `EntryCostUSDC` is net entry basis,
  and `TotalCostUSDC` includes `EntryFeesUSDC`. These are not interchangeable.
- IDs remain decimal strings (`TokenID`), and monetary numbers decode directly
  into decimal values without a binary floating-point intermediate. Addresses
  and condition/transaction IDs in response rows are strings exactly as served.
- `Traded` is replaced by `UserStats`. `response.Data` is nil for an unknown user;
  `response.Data.Trades` is the distinct-market count, not fill count.
- `Value` returns one `PortfolioValue` object. `LiveVolume` returns the event
  aggregate and per-condition volumes. Builder and trader economics match the
  v2 schema rather than preserving old field aliases.
- `Leaderboard` returns a ranked page. `LeaderboardUser` returns one object or
  nil, because the endpoint changes response shape when `user` is supplied.
- Combo reads return `Data` and `Pagination` too. Position/activity fields use
  the canonical typed Data API models, including decimal cash/share quantities;
  old synthetic `size_e6`, `amount_e6`, `positions` and `activity` response shapes
  are removed. Unsupported legacy query parameters are removed.
- Data health/freshness is read through `Status`; it no longer probes a legacy
  root response. User analytics, token lookup, resolutions, approvals and price
  history are available as typed v2 reads.
- Position sorting uses `CURRENT_VALUE`, `UNREALIZED_PNL`, `REALIZED_PNL`,
  `TOTAL_PNL`, `TOKENS`, `PRICE` or `TIMESTAMP`. Activity sorts only by timestamp.
  Leaderboard volume sorting uses `VOLUME`. Category/time-period constants use
  the documented lower-case values.

A successful response must contain its documented `data` key. Only nullable
pointer payloads accept explicit `data: null`. Cursor pages require array data
and pagination with nonnegative `limit`/`offset`, boolean `has_more` and a
consistent continuation cursor. Missing envelopes or pagination are errors,
not empty portfolios or completed history scans.

Every schema-required financial number must be present and non-null. A malformed
provider row with unavailable financial data returns a decode error; it never
becomes an invented zero balance or losing payout. Explicit numeric zero remains
valid. Documented nullable economics remain pointers and preserve null.

Data API errors normalize the official `error` message into `types.Error.Message`
while retaining `Retryable`, `TraceID` and `Parameter`. CLOB `message` errors stay
compatible. Structured errors remain accessible through `errors.As`, including
after the retry budget is exhausted.

The shared transport preserves bounded retry attempts, respects `Retry-After`
in seconds or HTTP-date form, and aborts the delay when the context expires.
Use a deadline on external reads; a provider error never becomes an empty page.

## Gamma protocol identity

`Market.Version` is `gamma.ProtocolV1` or `gamma.ProtocolV2`. `TradingTokens`
selects `clobTokenIds` for V1 and `positionIds` for V2, validates decimal uint256
IDs and outcome alignment, and returns an error for missing/unsupported versions.
It does not trust pre-parsed `Tokens` or infer a ledger from `negRisk`.
`ParsedTokens` returns nil for rejected markets; use `TradingTokens` to inspect
the error. V2 `ResolutionStatus` and V1 `UmaResolutionStatus` are separate fields.

Recognizing V2 metadata does not enable V2 trading in this release. Existing
CTF signing remains unchanged until the Protocol V2 SDK release.

## Verification and source contract

Run `go test ./...` to verify all packages and compile examples. Focused tests
cover exact decimal economics, full-width IDs, lifecycle cursor continuation,
null user stats, canonical filters, every new route, Combo wire shapes, and
explicit Gamma ledger selection. Transport tests cover retry delay cancellation.

The official OpenAPI snapshot in `pkg/data/testdata/openapi-v2.json` was fetched
on 2026-10-18 from <https://data-api.polymarket.com/v2/openapi.json>. Schema
coverage tests ensure typed row fields and nullability remain aligned with it.

Official references:

- <https://docs.polymarket.com/migrate/data-api-v1-to-v2>
- <https://docs.polymarket.com/migrate/polymarket-v2/api-integrations>

The upstream retirement date for Data API v1 is 2026-10-24. The accounting
snapshot endpoint remains on its existing route and is outside that retirement.

Condition request filters take canonical `common.Hash` selectors. Construct V2
selectors using `gamma.ConditionHash(gamma.ProtocolV2, id)`: native bytes31 IDs
are **right-padded** with zero; padded bytes32 IDs are checked for a zero final
byte. Calling `common.HexToHash` on a native bytes31 ID left-pads it and selects
a different condition. `ConditionHash` also strictly validates V1 bytes32 IDs.
