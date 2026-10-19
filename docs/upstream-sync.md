# Upstream compatibility update

Version `v0.4.0` compares our published `v0.3.1` commit
`145ffecac81b18664c85c2547535e43d5faa7104` with
[GoPolymarket main at 15d4d7c](https://github.com/GoPolymarket/polymarket-go-sdk/tree/15d4d7c076107a5700ec03cff34dc53233305958).
The histories share commit `b536e7b32b808e165aed22e9bed9c31d178e653a`;
27 upstream commits are absent from our Git history. Import-path changes were
normalized before comparing behavior. This is a selective integration of missing
capabilities, not a replacement of our protocol implementation.

## Added capabilities

| Domain | Additions |
| --- | --- |
| CLOB market data | Market lookup by token, single/batch live activity, batch price history, query-based books and path-based neg-risk reads |
| CLOB account data | PUT balance/allowance refresh; additional compact market-information fields |
| Gamma | Team lookup by ID and typed event/market keyset pages |
| Bridge | Quote and withdrawal-address requests with documented request/response fields |
| Data | Binary accounting snapshot download on the retained `/v1/accounting/snapshot` route |
| Relayer | Typed generic submission, transaction history, nonce, relay payload, deployment and API-key reads with Relayer-specific credentials |
| Signing | Exact uint256 salt JSON numbers for every signature type; per-order timestamp, metadata and builder overrides |
| Wire compatibility | Single-request server-time decoding, exact number/string earnings decoding and current RFQ signing fields |
| WebSocket lifecycle | Register market/user consumers before connection/submission and remove failed subscription state |

The existing `GetClobMarketInfo` API is extended rather than duplicated.
Missing signer addresses on supported ordinary orders are populated from the
configured signer; invalid maker identity still fails before submission.
Async order responses retain trade IDs, exact making/taking amounts, deferred
execution and bounded transaction-hash resolution.

## Corrected upstream wire contracts

New endpoints follow the official API specifications where upstream's
implementation differs:

- Gamma keyset requests use `after_cursor`, reject offsets and decode
  `{events, next_cursor}` or `{markets, next_cursor}` pages.
- Query-based order books use comma-separated `token_ids`. Batch live activity
  sends an array; single-market live activity returns an object.
- Market-by-token responses preserve both primary and secondary token IDs.
- PUT allowance refresh uses query parameters and acknowledges the update;
  callers must read balances separately.
- Bridge quotes preserve `fromAmountBaseUnit`, `recipientAddress` and the
  documented estimated amounts/fee breakdown. Withdrawal-address requests use
  `address` and `recipientAddr`.
- Accounting snapshots are ZIP bytes, not JSON-encoded byte slices.
- Generic relayer submission uses the documented flat signed payload. Nonce and
  relay-payload requests identify `address` and an explicit transaction type.
  Transaction lookup remains an array and nonces remain decimal strings.
- Relayer API keys use `RELAYER_API_KEY` and `RELAYER_API_KEY_ADDRESS` headers;
  CLOB L2 credentials are not substituted for that authentication mechanism.

See the official [CLOB](https://docs.polymarket.com/api-spec/clob-openapi.yaml),
[Gamma](https://docs.polymarket.com/api-spec/gamma-openapi.yaml),
[Bridge](https://docs.polymarket.com/api-spec/bridge-openapi.yaml) and
[Relayer](https://docs.polymarket.com/api-spec/relayer-openapi.yaml) specifications.

## Existing Data API equivalents

Retired upstream methods are represented by our cursor-based Data API v2
contract. Use `Status` instead of `Health`, `UserStats` instead of `Traded`, and
`Positions` with lifecycle filters instead of `ClosedPositions`. Condition-filtered
`Positions` replaces legacy market-position reads. Combo position/activity reads
remain on the v2-backed `Combos` client. The accounting snapshot route is the
documented exception that remains on v1.

The Data API v2 envelopes, exact decimal economics, required-field validation,
cursor continuity and strict token provenance from `v0.3.1` are retained. See the
[Data API migration guide](data-api-v2-migration.md) for the complete contract.

## Preserved fork behavior

Two upstream routes are not imported: Gamma `/combo-markets` and CLOB
`/markets/keyset`. Neither appears in its official OpenAPI specification and
read-only requests returned HTTP 404. Existing `Combos` supports the documented
Combo RFQ API, and existing CLOB `Markets` already supports cursor pagination.
Gamma's documented event and market keyset endpoints are included.

The module remains `github.com/neor-it/polymarket-go-sdk`. Protocol V2 retains
ExchangeV3/domain `3`; CTF retains its existing exchanges/domain `2`. External
POLY_1271 signatures, address-only signers, KMS digest signing, Deposit Wallet
derivation/batches and builder-code attribution remain supported. ClobAuth and
Deposit Wallet domains are unchanged. See the
[Protocol V2 guide](protocol-v2-migration.md).

Linux CI exposed an inherited WebSocket ordering race: an immediate server reply
could arrive before the local stream was registered. Market and user streams now
register before connecting/submitting and remove local state after setup failure.
Regression checks cover immediate replies, failed setup and shared subscriptions.

Transport retains `Retry-After`, cancellation and structured errors after
exhausted retries. No balances are automatically converted and no transaction
or redemption scheduler starts merely because the SDK is upgraded.

Upstream's `/v2` Go module rename, legacy Data API implementation, governance
instruction files and checked-in trading executable are not imported. Source
examples continue to build from this module. Earlier published tags and releases
remain unchanged. The integration is delivered as `v0.4.0`.

## Dependencies and CI

Direct dependencies advance to go-ethereum `v1.17.5` and AWS KMS `v1.61.0`, with
their compatible minimum dependency graph. The Go language requirement remains
`1.24.0`. SDK development and CI use toolchain `go1.26.9`: vulnerability analysis
found called standard-library advisories with the old pinned `go1.24.13`.
Applications select their own compiler; a dependency's toolchain directive does
not upgrade a consuming application's toolchain.

CI reads the configured toolchain from `go.mod`, pins golangci-lint `v2.12.2`
and govulncheck `v1.9.0`, checks the root package and examples, and runs focused
race checks. Reachable vulnerability failures fail CI. Debricked also requires
its repository scan credential; its container disables Go VCS stamping for
dependency resolution only. A separate
compile-only job verifies Go 1.24 source compatibility; it is not a security
claim about the older runtime.

The manual **Compatibility Evidence** workflow runs WebSocket schema regression
tests and the package compile gate, retaining logs, source SHA, dirty-worktree
status and SHA-256 checksums in an archive. Run it locally with
`scripts/compatibility-evidence.sh --out-dir artifacts/compatibility`. These two
checks supplement the full SDK suite and consumer verification. No recurring
schedule is created.

Public client interfaces gain methods, so custom implementations and mocks may
need updates. The go-ethereum update also adds `TransactionByHash` to custom
`bind.ContractBackend` implementations. Keep the consumer's compiler patched
and test the actual dependency graph selected by its `go.mod`.

The [Polygaming rollout guide](polygaming-rollout.md) distinguishes SDK
publication from application deployment and V2 write activation.
