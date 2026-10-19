# Release Notes

## Version 0.4.0 (2026-10-19)

- Integrate missing capabilities from GoPolymarket SDK commit `15d4d7c` while
  retaining this module's Data API v2, Protocol V2, CTF, external-signature and
  wallet lifecycle implementations. Existing published releases are unchanged.
- Add CLOB market/account endpoints, Gamma typed keyset pages and team lookup,
  Bridge quote/withdrawal APIs, binary
  accounting snapshots and official typed Relayer APIs/authentication.
- Preserve complete uint256 salt precision for all signature types, add per-order
  timestamp/metadata/builder overrides, update RFQ signing fields and accept
  compatible server-time and exact numeric earnings representations.
- Fix an inherited WebSocket subscription race that could drop immediate market
  or user events; failed setup removes the pending consumer and reference counts.
- Advance go-ethereum to `v1.17.5` and AWS KMS to `v1.61.0`. Keep the Go 1.24
  language floor and use patched toolchain `go1.26.9` for development/CI.
- Pin lint tooling, add race/source-compatibility checks and a manual checksummed
  compatibility-evidence exporter. See the [comparison and upgrade notes](docs/upstream-sync.md).
- Make reachable-vulnerability scanning a required CI step with pinned
  govulncheck `v1.9.0`; allow containerized Debricked dependency resolution without
  Go VCS stamping. Debricked still requires its own scan credential.
- Public client interfaces gain methods. Custom implementations and mocks may
  need updates; custom go-ethereum `bind.ContractBackend` implementations must
  implement `TransactionByHash`. The module path is unchanged.
- SDK publication does not update an already built application or enable V2
  operations. Follow the [Polygaming rollout guide](docs/polygaming-rollout.md)
  for schema, consumer, worker and activation gates.

## Version 0.3.1 (2026-10-18)

- Reject missing or null Data API token-reference identity fields instead of
  interpreting unavailable `outcome_index` as outcome zero. The published int32
  range is enforced; explicit zero, unresolved `false` and nullable enrichment
  fields remain supported. Consumers can safely distinguish unavailable
  provenance from a verified YES slot.

## Version 0.3.0 (2026-10-18)

### Explicit Protocol V2 support

- Added shared protocol identity, version-aware order builders and centralized
  exchange/domain selection. V2 uses Polygon ExchangeV3/domain `3`; CTF keeps
  its current exchanges/domain `2`. EOA, Safe and internal/external POLY_1271
  paths share the configuration. ClobAuth and DepositWallet domains are unchanged.
- Added `CONDITIONAL-V2`, full-width uint256 order validation and integer maker
  fill calculations with separate collateral fees.
- Added the standalone `positions` contract client and calldata encoders for
  balances, payouts, approvals, transfers, Router split/merge/redeem, registered
  migrations and operator-only AutoRedeemer submissions.
- Added typed, emitter-validated ERC-1155, module, Router, migration and automatic
  redemption events, preserving reorg identity and native identifier widths.
  Published ABIs are parsed once for concurrent readers.

### Upgrade and verification boundaries

Persist the protocol in immutable application operation snapshots. Local order
protocol metadata is excluded from CLOB wire payloads. Unknown live-market
versions fail closed; empty historical order versions retain V1 compatibility.
See [Protocol V2 migration](docs/protocol-v2-migration.md) for approvals, exact
amounts, indexing and source provenance.

No automatic balance conversion, redemption scheduling or transaction broadcast
is enabled. Mainnet matching/relayer behavior and archival fork reads are not
claimed as verified. Applications must keep V2 writes gated until their own
separately authorized financial canary passes.


## Version 0.2.0 (2026-10-18)

### Data API v2 — breaking Go API changes

- All retiring Data API reads use `/v2`, including the existing Combos feeds from
  `main`. Responses expose typed `Data` and explicit cursor `Pagination`; request
  offsets are removed. Follow `NextCursor` with the original filters until nil.
- Position lifecycle reads use `PositionsRequest.Status`, replacing the separate
  closed-position route. `UserStats` replaces `Traded`; an unknown user remains
  `Data == nil`. Position basis, fees, current holdings and lifetime bought shares
  retain their distinct v2 semantics.
- Missing response envelopes and incomplete cursor metadata are rejected instead
  of becoming empty portfolios or truncated history. Explicit null payloads remain
  valid only for endpoints whose documented result is nullable.
- Financial values decode directly into decimals. Missing/null required financial
  values are rejected; explicit zero is valid, and documented nullable fields
  remain pointers. Asset IDs retain their complete decimal-string representation.
- Added typed token lookup, resolution, approvals, freshness, price history and
  user analytics reads. Trader leaderboard pages and single-user standings have
  separate typed methods because the upstream response shapes differ.
- Gamma market versions select `clobTokenIds` for V1 and `positionIds` for V2.
  Missing/unsupported versions and malformed outcome alignment fail closed.
  `ConditionHash` right-pads native V2 bytes31 IDs and validates padded bytes32 IDs.
- Shared transport honors `Retry-After`, bounded retry attempts and cancellation.
  Official Data API `error` bodies preserve their message, retryability, trace ID
  and parameter through `errors.As`, including exhausted retries; CLOB `message`
  errors remain supported.

### Combos and migration guidance

This release includes the existing root Combos client, authentication and order
implementation from `main`. Combo positions/activity now use the canonical v2
rows and cursor envelope instead of synthetic legacy response fields.

See [Data API v2 migration](docs/data-api-v2-migration.md) for the complete Go
upgrade guide and economics mapping. Upstream Data API v1 retires on October 24,
2026. The accounting snapshot endpoint remains outside that retirement.

### Validation and release boundary

All SDK packages and examples compile under `go test ./...`; focused race checks
cover Data API, Gamma, Combos, transport and numeric/error types. Official OpenAPI
schema tests and independent SDK/backend tests cover exact decimals, full-width
IDs, cursor continuity, null handling and error contracts.

This release recognizes Protocol V2 metadata but does not enable ExchangeV3
signing or PositionManager writes. No live trading or mainnet settlement success
is claimed. Existing CTF trading remains on its current signing configuration.


## Version 0.1.5 (2026-07-23)

### CLOB Order Response Compatibility

- Added `OrderResponse` support for `success`, `errorMsg`, `makingAmount`, `takingAmount`, `transactionsHashes`, and `tradeIDs`.
- `PostOrder` and `PostOrders` now resolve settlement transaction hashes from `tradeIDs` when Polymarket returns matched orders without inline hashes.
- Trade polling is best-effort: a successful order submit is never failed because trade lookup times out or returns transient errors.
- `deferExec` orders skip hash polling because execution is intentionally deferred.

### Upgrade Notes

- This release is backward compatible with pre-async-commit responses that still include inline `transactionsHashes`.
- Custom integrations that read `OrderResponse.TransactionHashes` can keep doing so; SDK users that need raw trade IDs can read `OrderResponse.TradeIDs`.

## Version 0.x.x (2026-02-10)

### 🔧 Critical Bug Fixes

This release addresses 6 critical and high-priority concurrency and performance issues that could impact production deployments.

#### WebSocket Client Improvements

**Fixed: Goroutine Leaks in WebSocket Connections**
- Added per-connection context cancellation to properly manage goroutine lifecycle
- Implemented `createGoroutineContext()`, `cancelGoroutines()`, and `getGoroutineContext()` helper methods
- Ensured old `pingLoop` and `readLoop` goroutines are properly cleaned up on reconnection
- **Impact**: Prevents memory leaks and resource exhaustion in long-running applications with frequent reconnections

**Fixed: Race Conditions in Connection State Management**
- Connection pointers are now set to `nil` after closing to prevent use-after-close errors
- Added proper nil checks after acquiring connection references
- **Impact**: Eliminates potential crashes and undefined behavior in concurrent scenarios

**Fixed: Subscription Panic Risks**
- Removed panic recovery from `trySend()` method - now uses clean non-blocking send
- Added 10ms grace period before closing channels to allow pending sends to complete
- Fixed TOCTOU (time-of-check-time-of-use) race condition between closed check and channel send
- **Impact**: Improves stability and prevents runtime panics in high-throughput scenarios

#### Stream Processing Improvements

**Fixed: Context Cancellation in Stream Functions**
- Made `StreamDataWithCursor` fully respect context cancellation
- Added context checks before each fetch operation
- Made channel sends cancellable using select with `ctx.Done()`
- **Impact**: Enables proper cleanup and resource management when operations are cancelled

#### Heartbeat Management

**Fixed: Heartbeat Goroutine Accumulation**
- Added proper cleanup of old heartbeat goroutines in `startHeartbeats()`
- Implemented 50ms delay to allow old goroutines to exit gracefully before starting new ones
- **Impact**: Prevents goroutine accumulation when heartbeat intervals are changed

#### Performance Optimization

**Optimized: Rate Limiter Implementation**
- Complete refactor from ticker-based to timestamp-based token calculation
- **Eliminated background goroutine** - tokens are now calculated on-demand
- Simplified internal structure: removed channels, replaced with float64 token counter
- Added `stopped` flag for backward compatibility with `Stop()` behavior
- **Impact**: Reduced resource consumption and improved efficiency in high-throughput scenarios

### 📊 Test Coverage

Added comprehensive test suites to ensure reliability:
- `pkg/clob/ws/goroutine_leak_test.go` - Goroutine leak detection using goleak
- `pkg/clob/ws/race_condition_test.go` - Concurrent access pattern testing
- `pkg/clob/ws/subscription_panic_test.go` - Subscription lifecycle and panic prevention tests

**Test Results**: all packages passing (100%)
- Rate Limiter: 6/6 tests passing (100%)
- WebSocket: all tests passing

### 🔄 Breaking Changes

**None** - All changes are backward compatible. Existing code will continue to work without modifications.

### 📦 Dependencies

- Added `go.uber.org/goleak v1.3.0` for goroutine leak detection in tests

### 🔍 Files Modified

**Core Implementation:**
- `pkg/transport/ratelimit.go` - Complete refactor to timestamp-based implementation
- `pkg/clob/ws/impl.go` - Goroutine lifecycle management and race condition fixes
- `pkg/clob/ws/subscription_manager.go` - Subscription safety improvements
- `pkg/clob/stream.go` - Context cancellation enhancements
- `pkg/clob/impl.go` - Heartbeat goroutine management

**Test Files:**
- `pkg/clob/ws/goroutine_leak_test.go` (new)
- `pkg/clob/ws/race_condition_test.go` (new)
- `pkg/clob/ws/subscription_panic_test.go` (new)

### 📈 Performance Impact

- **Memory**: Reduced memory footprint by eliminating unnecessary background goroutines
- **CPU**: More efficient rate limiting with on-demand token calculation
- **Stability**: Eliminated goroutine leaks and race conditions for improved long-term stability

### 🎯 Upgrade Recommendation

**Highly Recommended** for all production deployments, especially:
- Applications with long-running WebSocket connections
- High-frequency trading systems
- Market making bots
- Any service experiencing memory growth over time

### 🙏 Acknowledgments

This release was made possible through comprehensive code review and optimization efforts using AI-assisted development tools.

---

**Full Changelog**: https://github.com/neor-it/polymarket-go-sdk/compare/a9a3cc8...89b9bed
