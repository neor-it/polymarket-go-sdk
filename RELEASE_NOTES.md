# Release Notes

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
