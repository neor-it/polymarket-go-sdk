# Polygaming deployment after SDK v0.4.0

Publishing the SDK, deploying the platform, and opening Protocol V2 trading are
three separate releases. SDK publication does not change a running application.
Deploy compatible code with V2 writes closed first; activate trading only after
the application, schema, historical holdings and live settlement are verified.
This guide records deployment instructions, not a production deployment.

## Audited application candidates

| Artifact | Audited identity | Status and boundary |
| --- | --- | --- |
| SDK | [PR #1](https://github.com/neor-it/polymarket-go-sdk/pull/1), release v0.4.0 | Retains v0.3.1 Data API v2 and Protocol V2 support; adds upstream capabilities and WS repairs |
| Backend | [PR #160](https://github.com/sumika-xyz/polygaming-platform/pull/160), `322d53c1149aa043a9a3ffd14e41eb2e96efaf2d` | Uses SDK v0.3.1; PostgreSQL CI passed on this SHA |
| Frontend | [PR #116](https://github.com/sumika-xyz/polygaming.xyz/pull/116), `4abede0f9598575c090d72f88dcc72e74ecfc90d` | Type, unit, SSR and production build CI passed on this SHA |
| Production schema | Last operator-confirmed migration: 035 | Verify the actual version and dirty state before executing any migration |

Both platform PRs target `feature/polypass`, not production. Their green checks
do not certify a future production merge. Freeze and test that actual merge set,
including Polypass. Frontend deployment also exposes `/earn`; there is no
separate Polypass cohort flag in these candidates.

For these candidates, the maintenance sequence below supersedes the older
runbook's suggestion to accept normal Backend B financial traffic on schema 036.
The temporary token-only primary key can still conflict with ledger-scoped
inventory inserts. Complete 037 before starting the new full financial runtime.

The official [Data API migration guide](https://docs.polymarket.com/migrate/data-api-v1-to-v2)
retires the listed v1 routes on **October 24, 2026**. Target the application's
Data API v2 deployment for **October 22**, reserving October 23 for repairs.
An SDK tag alone does not meet this deadline. Full V2 trading need not open to
complete the Data API migration. The retained accounting snapshot route is an
explicit exception to the retirement.

## Capability and activation matrix

| Capability | SDK | Audited platform implementation | Activation boundary |
| --- | --- | --- | --- |
| Data API v2 | Cursor pages, envelopes, exact values and token provenance | v2 trades/activity and reconciliation | Compatible backend deployment |
| CTF markets and holdings | Retained | Retained with verified historical identity | V1 regression and provenance checks |
| V2 trading | ExchangeV3/domain 3, position IDs, approvals and signing | BUY/SELL/claim, readiness, immutable execution identity, fees and recovery | Account and market cohorts plus independent operation flags |
| V2 position contracts | PositionManager and Router split/merge/redeem/payout | Claim lifecycle; no separate user split/merge flows | Claim gate; additional flows need product implementation |
| Position indexing | Typed ledger, module and migration events | Finalized observations, replay, external transfers/redemptions and reconciliation guards | Schema 037, compatible frontend and verified RPC/start block |
| AutoRedeemer | Approval, calldata and events | Approval/index observation support | No platform scheduler/operator workflow to enable as part of this rollout |
| CTF-to-V2 conversion | Registered migration operations and events | Detects external conversion and pauses affected lots for reconciliation | No automatic lot conversion or user migration-submission flow |
| Upstream API extensions | Generic Relayer/Bridge, Gamma keysets, accounting snapshots | New wrappers are not automatically used by existing backend services | Consumer adoption and tests when a feature needs them |

`POLYMARKET_V2_MIGRATION_ENABLED` is configuration in the audited backend, not
an implemented transaction-submission workflow. Do not promise conversion by
turning it on. `POSITION_OPERATIONS_ENABLED` gates platform claim; it does not
create split/merge UI. The platform's existing order execution is FOK; SDK order
type support is not evidence for platform GTC/GTD maker execution.

An externally converted holding is not automatically reassigned to a platform
lot. The frontend can display the confirmed destination and an unknown valuation;
the original position remains blocked for reconciliation until attribution is
established. Old signed operations retain their original identity.

## Why an unordered simultaneous deployment can fail

- New backend code on schema 035 accesses missing metadata/identity columns.
- `make migrate-up` applies every pending migration, including 037. Old workers
  must not run after the inventory keys become ledger/owner scoped.
- Backend B on 036 can also fail existing V1 SELL/reservation paths: a scoped
  inventory lookup misses an unclassified legacy row, and inserting the same
  asset then conflicts with the still token-only primary key. V2 flags alone do
  not make this temporary combination a normal financial serving state.
- The API binary also starts workers. Its sponsor EOA transactor requires
  **one active worker per sponsor key**. Overlapping old/new replicas can race
  on nonces; ordinary blue/green deployment is insufficient without worker control.
- Position indexing before 037, without a positive verified start block, or on
  the wrong RPC chain fails initialization and stops application startup.
  Missing finalized, historical-log or block-hash RPC support can instead fail
  scanner execution while the API remains live; verify both startup and catch-up.
- Migration 036 leaves historical market protocol versions unknown. New BUYs
  fail closed until Gamma discovery refreshes the correct metadata. Database
  migration success does not establish market readiness.
- Old frontend code does not understand V2 ledger identity and observations.
  Do not expose V2 writes or indexing observations before frontend cutover.

Use a controlled maintenance window for this candidate. Pause new financial
requests, drain and stop the old API/workers, complete the schema change, then
start the new runtime. The API binary starts financial workers automatically;
an ingress pause alone does not pause recovery work. Zero downtime would need
separately verified worker isolation and intermediate-schema compatibility.

## Step-by-step deployment

### 1. Publish and adopt a verified SDK

Merge the reviewed PR, wait for checks on the resulting main commit, tag that
exact commit and publish v0.4.0. Download the published module in a clean consumer.
Do not move earlier tags. In a separate backend change, pin the new SDK and
commit its resolved `go.mod`/`go.sum`; never ship a temporary local replacement.

Run backend PostgreSQL, financial recovery/race and browser E2E checks on the
actual SDK/backend/frontend merge set. Custom SDK clients and mocks must implement
new interface methods. Custom geth `bind.ContractBackend` implementations need
`TransactionByHash`. Keep the consuming application's compiler patched: the
dependency's toolchain directive does not upgrade it.

### 2. Close all V2 activation gates

Use the following initial backend settings:

```dotenv
POLYMARKET_V2_BUY_ENABLED=false
POLYMARKET_V2_SELL_ENABLED=false
POLYMARKET_V2_POSITION_OPERATIONS_ENABLED=false
POLYMARKET_V2_AUTO_REDEEMER_ENABLED=false
POLYMARKET_V2_MIGRATION_ENABLED=false
POLYMARKET_V2_ACCOUNT_ALLOWLIST=
POLYMARKET_V2_MARKET_ALLOWLIST=
POLYMARKET_POSITION_INDEXING_ENABLED=false
```

Empty account and market cohorts also deny V2 writes. V1 readiness is independent
of V2 approvals. Freeze environment configuration and rollback artifacts together
with the code SHA.

### 3. Back up and inventory production

Verify schema version 035 with no dirty migration. Back up the database and
record current inventory/reservations, pending intents, relayer/CLOB identifiers,
submitted transaction hashes, pUSD/share balances and reconciliation totals.
Measure DDL behavior and locks on a representative disposable database.

Identify the active sponsor worker and its graceful drain procedure. Do not
clear pending transactions, reuse their nonce blindly or restore a stale database
over financial actions to make deployment easier.

### 4. Apply only migration 036

From a **verified clean 035** database, use the reviewed migration directory:

```sh
migrate -path migrations -database "$DATABASE_URL" version
migrate -path migrations -database "$DATABASE_URL" up 1
migrate -path migrations -database "$DATABASE_URL" version
```

Confirm the resulting version is 036 and clean. These commands are operator
instructions, not an automated production action. Do not use an unrestricted
`make migrate-up` here. The application does not apply migrations on startup.

### 5. Drain and stop all old API/worker processes

Pause new financial requests and gracefully drain the old financial writer.
Record remaining pending operations and verify that every old API/worker process
has stopped. Do not start Backend B yet. No sponsor worker may overlap its
replacement. The tested replacement image and compatible rollback plan must
already be prepared before entering the maintenance window.

### 6. Apply migration 037, then start Backend B

With all old processes stopped and the database at **verified clean 036**, run
exactly one migration using the command sequence above and confirm clean 037.
The new primary keys include chain, ledger and owner identity. Control DDL locks
using the window measured on representative data.

Start the compatible Backend B image on 037 with V2 writes and indexing off.
Ensure one active transactor per sponsor key. Check health, authentication,
wallet bootstrap, history, Data API v2 reads and pending recovery. Keep new
financial requests controlled until the historical inventory and market gates
below pass; a successful startup is not financial acceptance.

### 7. Refresh metadata and deploy the compatible frontend

Wait for successful Gamma refresh and verify protocol, condition and outcome
asset mappings for each market that will accept BUYs. Do not bulk-label unknown
rows as V1 to bypass fail-closed guards.

Deploy the compatible frontend. Verify large IDs as strings, operation-specific
readiness and closed V2 actions. Missing readiness must remain unavailable, not
a zero balance or successful approval. A new frontend can use legacy V1 backend
fields during transition. An old frontend must not be used for V2 activation.

### 8. Complete historical reconciliation

Enumerate unclassified global inventory, user holdings, reservations and pending
orders. Lazy user-position adoption alone does not prove that every legacy
global inventory row has been attributed. Bind identity
only to confirmed execution snapshots and authoritative token provenance.
Reconcile before opening affected financial actions; unknown/conflicting rows
remain unavailable rather than being guessed as CTF.

The candidate has lazy verified adoption and bounded account/asset recovery, but
no ready tracked CLI for a production-wide offline backfill. If actual accounts
exceed the bounded candidate limit, a reviewed batch procedure and per-batch
report are required before release to that cohort. This is a deployment gate,
not a request to invent identities or change financial amounts with bulk SQL.

Only then accept V1 financial traffic: verify existing approvals, historical
SELL/reservations, claim/settlement and recovery against reconciled holdings.
Keep unresolved affected cohorts closed. This acceptance is after 037, not an
intermediate 036 check.

### 9. Activate finalized indexing

After frontend cutover and schema 037, verify RPC chain ID, finalized headers,
historical logs and EIP-1898 block-hash calls. Choose a confirmed deployment block
or reconciled checkpoint as `POLYMARKET_POSITION_SCAN_FROM_BLOCK`; do not guess
the current head. Enable `POLYMARKET_POSITION_INDEXING_ENABLED`, wait for catch-up,
and reconcile pUSD/shares, history, source/destination holdings and cursor age.

### 10. Open V2 only after a live canary

With separate authorization, use an isolated wallet and budget. Verify approvals,
confirmation/CLOB allowance refresh, BUY, SELL, cancellation and redemption of an
available resolved position. Reconcile actual shares, pUSD, fees, persisted
settlement and Polypass effects. Mock/fork tests do not prove live matching or
relayer behavior.

Keep BUY closed until SELL and redemption are proven. Open only the selected
account/market allowlists, observe one day, extend to a small cohort, then observe
another day before broader access. AutoRedeemer scheduling and migration
submission are separate product/fee-policy decisions and remain closed.

## Remaining test and operational gates

The frontend's Global E2E workflow builds PostgreSQL, backend and frontend on
the GitHub runner. It does not require a deployed staging environment. Its
`E2E_BACKEND_CHECKOUT_TOKEN` secret is needed only to read the private backend
repository (`Contents: read`); its variables must pin the final backend SHA.
The audited run failed credential preflight before tests. Run it again on the
final consumer set; a successful standalone frontend build does not replace it.

The previous Polygon fork verification was blocked by archive RPC HTTP 403.
Mainnet CLOB/relayer canary has not been executed. They are unverified levels,
not successful checks. Do not claim all migration behavior is live-proven.

## Monitoring and rollback

Watch provider error/latency, pending operation age, indexing lag, approval and
signature failures, pUSD/share discrepancies, duplicate settlements and Polypass
awards. An unexplained financial discrepancy stops cohort expansion and new
V2 BUYs. Preserve reads and recovery; gate SELL/claim separately if affected.

After 037, use only a backend compatible with ledger-scoped identity. The down
migration rejects colliding holdings instead of merging or deleting them. After
Data API retirement, rollback must remain v2 compatible. Restore neither an old
token-only schema nor an old database snapshot over executed financial actions.
Use a compatible rollback artifact or reviewed forward repair.

For implementation details and source-specific operating instructions, use the
[backend release runbook](https://github.com/sumika-xyz/polygaming-platform/blob/322d53c1149aa043a9a3ffd14e41eb2e96efaf2d/docs/polymarket-v2-release-runbook.md),
[backend worker startup](https://github.com/sumika-xyz/polygaming-platform/blob/322d53c1149aa043a9a3ffd14e41eb2e96efaf2d/cmd/server/main.go),
and [frontend E2E workflow](https://github.com/sumika-xyz/polygaming.xyz/blob/4abede0f9598575c090d72f88dcc72e74ecfc90d/.github/workflows/global-e2e.yml).
