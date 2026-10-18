# SDK v0.3.0: Protocol V2 alongside CTF

This release adds an explicit PositionManager/ExchangeV3 path while retaining
CTF holdings, clients and domain version `2`. Updating the SDK never converts
existing positions or submits transactions automatically.

## Identity and order signing

`types.ProtocolVersion` is shared by Gamma and orders; `gamma.ProtocolVersion`
remains a source-compatible alias. Select assets using Gamma's explicit version.
`OrderBuilder.ProtocolVersion(types.ProtocolV2)` and `Order.ProtocolVersion`
bind an order to the PositionManager ledger. Missing versions on live Gamma
markets remain invalid. An empty version on a historical order preserves its
legacy V1 interpretation; new builders explicitly default to V1.

`clob.SigningConfiguration(chainID, protocol, negRisk)` returns the protocol,
exchange proxy and EIP-712 domain version. V2 is supported on Polygon chain 137
only, uses ExchangeV3 and domain version `3`, and uses the same exchange for
binary and neg-risk positions. CTF retains its existing standard/neg-risk
exchange selection and domain version `2`.

EOA, Safe, internal POLY_1271 and external `BuildPoly1271OrderEnvelope` use that
configuration. Exchange domain version `3` does not change ClobAuth or the
DepositWallet's own domain version `1`. Internal and external POLY_1271 signing
are tested to produce the same complete signature for the same order.

`Order.ProtocolVersion` is local metadata and is excluded from the CLOB wire.
Persist protocol, chain, exchange/domain, condition, asset and outcome in the
application's immutable prepared-operation snapshot, and restore the protocol
before signing/recovering that order. Do not reselect its ledger from refreshed
market metadata. Reject explicit unknown order versions.

## Balances, permissions and integer fills

Use `clobtypes.AssetTypeConditionalV2` (`CONDITIONAL-V2`) for V2 positions, retain
`CONDITIONAL` for CTF and `COLLATERAL` for pUSD. After confirmed approvals, call
CLOB allowance update/read with the appropriate maker signature type and verify
the exact ExchangeV3 spender entry.

V2 BUY requires pUSD permission for ExchangeV3, covering principal plus fees.
SELL requires PositionManager operator permission for ExchangeV3. Existing CTF
approvals do not carry over.

`clob.CalculateFill` uses integer base units and the signed maker/taker amounts:
`floor(makerAssetFill * takerAmount / makerAmount)`. It reports principal, shares,
fee, collateral debit and collateral credit separately. BUY fees add to spend;
SELL fees reduce proceeds. Inputs are not mutated. Signed order amounts must be
positive integer uint256 values, not fractional base units. BUY GTC/GTD targets
remain shares; BUY FOK/FAK targets remain collateral through the existing builder.

## Position contracts

`positions.ConfigForChain(137)` / `ConfigPolygon137` provide the documented proxy
addresses, including legacy ConditionalTokens for indexing. No implementation
address is used as a spender or transaction target. Callers may provide an
explicit deployment config, which is validated before constructing a client.

`positions.NewClient(caller, config)` accepts the read-only `CallContract`
interface implemented by `ethclient.Client`. Reads include balances, exact
pUSD allowances, operator approvals, payout for an explicit amount and native
or derived Binary/NegRisk module results. RPC failures and malformed output are
errors; a legitimate losing payout is an explicit zero. An empty module result
vector remains unresolved.

The standalone calldata encoders return bytes for an application-owned wallet
or relayer transaction:

- `EncodeSplit` / `EncodeMerge` / `EncodeRedeem` target Router. Amounts use six
  decimals; redeem uses outcome `0` or `1` and an explicit amount.
- `EncodeERC20Approval` targets pUSD; `EncodeSetApprovalForAll` targets the
  appropriate position ledger. Spend/permission targets must match the operation.
- `EncodeTransfer` creates a safe ERC-1155 transfer, preserving receiver data.
- `EncodeMigratePositions` / `EncodeOperatorMigratePositions` target the correct
  registered Binary/NegRisk module. Conditions are legacy bytes32 IDs in
  nondecreasing order, with matching outcome/amount arrays. The contract enforces
  actual registration and authority; the SDK never guesses a mapping.
- `EncodeAutoRedeem` targets AutoRedeemer and is operator-only. It redeems full
  submitted balances for their original holders. Approval does not schedule it.

Router pulls from and returns outputs to the calling holder in one transaction.
Do not pre-transfer assets to a module in a separate transaction. Protocol mints
and unsafe transfers do not invoke receiver callbacks; receiving contracts must
refresh or index their balances explicitly rather than relying on callbacks.

`ConditionID` validates native bytes31 or right-padded bytes32 before narrowing.
`PositionID` adds an explicit binary outcome; `DecodePositionID` separates the
published module/condition/outcome layout without using it to infer a ledger.
Asset IDs remain uint256 throughout.

## Indexing and reorgs

`positions.DecodeLog(log, config)` returns typed transfers, approvals, module/
Router operations, migration registrations/conversions, results and AutoRedeemer
redemptions. Only configured emitters are accepted; unknown topics/emitters are
ignored, recognized malformed logs return errors. Native fixed-bytes identifiers
are right-padded only at the bytes32 compatibility boundary.

`LogEvent.Raw` retains transaction, block, log index and `Removed` identity.
Single/batch ERC-1155 transfers are the canonical source for holdings; lifecycle
operations are context, not additional credits. Module `initiator` is the
immediate caller. Attribute owners and recipients using transfers and payouts.
AutoRedeemer payouts return to each holder and may bypass application fee logic.

`RemainingConditionsDerivableAsNo` and `SyntheticConditionDerivableAsYes` trigger
fresh module result or PositionManager payout reads; individual condition events
are not required for native derivations. Registered migrated conditions retain
their CTF-backed result rules. Preserve `MigrationConditionRegistered` as the
authoritative old/new condition mapping.

For reorg-safe absolute balances at a block hash, use the exported
`PositionManagerABI` / `EncodeBalanceOfBatch` with the caller's block-hash-pinned
RPC. The default client reads latest state; it is not an indexer checkpoint.
Static ABIs are parsed once and shared immutably across concurrent readers.

## Sources and verification

ABI declarations are derived from the official
[Protocol V2 repository](https://github.com/Polymarket/polymarket-v2-external/tree/741f8bbe88c3f29a1c55d1248a3dd411f622f111),
including Router, PositionManager, BaseModule, NegRiskModule,
CombinatorialModule, BaseMigrationMixin and AutoRedeemer. The event contract
snapshot is pinned in `pkg/positions/testdata/events.json`.

- [API migration](https://docs.polymarket.com/migrate/polymarket-v2/api-integrations)
- [Contract migration](https://docs.polymarket.com/migrate/polymarket-v2/contract-integrations)
- [Proxy contract catalog](https://docs.polymarket.com/resources/contracts)
- [On-chain position indexing](https://docs.polymarket.com/resources/onchain-position-data)

Run `go test ./...` and focused race checks on signing/contracts. Tests cover
V1/V2 domain selection, internal/external POLY_1271 equivalence, calldata and
condition widths, integer fills, registered-migration overloads, exact RPC
reads, zero/unresolved payouts and canonical event attribution.

No live trading or relayer/mainnet settlement success is claimed. A pinned local
Polygon fork obtained contract code, but archival contract reads were blocked
by the provider; those checks remain unverified. A real integration must keep
V2 writes gated until its separately authorized end-to-end canary succeeds.

The optional bot `Opportunity`/`TradePlan` carry explicit protocol metadata into
execution. Its existing CLOB-only scanner is not a Gamma V2 discovery workflow;
V2 candidates must be supplied with verified Gamma protocol identity.

Legacy CTF lifecycle context (`PositionSplit`, `PositionsMerge`,
`PayoutRedemption`) is decoded only from ConditionalTokens. Those declarations
are pinned to the official
[ConditionalTokens source](https://github.com/gnosis/conditional-tokens-contracts/blob/eeefca66eb46c800a9aaab88db2064a99026fde5/contracts/ConditionalTokens.sol).
CTF redemption and merge proceeds must not be reclassified as deposits.
