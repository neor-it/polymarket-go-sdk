package bot

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
	"github.com/neor-it/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/shopspring/decimal"
)

func testSigner(t *testing.T) auth.Signer {
	t.Helper()
	s, err := auth.NewPrivateKeySigner("0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", 137)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	return s
}

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// ---- config.go ----

func TestConfig_Validate(t *testing.T) {
	base := DefaultConfig()
	tests := []struct {
		name    string
		mutate  func(c *Config)
		wantErr bool
	}{
		{"default is valid", func(c *Config) {}, false},
		{"scan limit zero", func(c *Config) { c.ScanLimit = 0 }, true},
		{"top-n zero", func(c *Config) { c.TopN = 0 }, true},
		{"default amount zero", func(c *Config) { c.DefaultAmountUSDC = decimal.Zero }, true},
		{"default amount negative", func(c *Config) { c.DefaultAmountUSDC = dec("-1") }, true},
		{"max per trade zero", func(c *Config) { c.MaxPerTradeUSDC = decimal.Zero }, true},
		{"max daily loss negative", func(c *Config) { c.MaxDailyLossUSDC = dec("-1") }, true},
		{"max daily loss zero is ok", func(c *Config) { c.MaxDailyLossUSDC = decimal.Zero }, false},
		{"max open trades zero", func(c *Config) { c.MaxOpenTrades = 0 }, true},
		{"max slippage zero", func(c *Config) { c.MaxSlippageBps = decimal.Zero }, true},
		{"min confidence negative", func(c *Config) { c.MinConfidenceBps = dec("-1") }, true},
		{"min confidence zero is ok", func(c *Config) { c.MinConfidenceBps = decimal.Zero }, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := base
			tc.mutate(&c)
			err := c.Validate()
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestConfig_MergeEnv(t *testing.T) {
	t.Setenv("BOT_SCAN_LIMIT", "200")
	t.Setenv("BOT_TOP_N", "3")
	t.Setenv("BOT_DEFAULT_AMOUNT_USDC", "12.5")
	t.Setenv("BOT_MAX_PER_TRADE_USDC", "77")
	t.Setenv("BOT_MAX_SLIPPAGE_BPS", "40")
	t.Setenv("BOT_DRY_RUN", "false")

	c := DefaultConfig().MergeEnv()
	if c.ScanLimit != 200 {
		t.Errorf("ScanLimit = %d, want 200", c.ScanLimit)
	}
	if c.TopN != 3 {
		t.Errorf("TopN = %d, want 3", c.TopN)
	}
	if !c.DefaultAmountUSDC.Equal(dec("12.5")) {
		t.Errorf("DefaultAmountUSDC = %s, want 12.5", c.DefaultAmountUSDC)
	}
	if !c.MaxPerTradeUSDC.Equal(dec("77")) {
		t.Errorf("MaxPerTradeUSDC = %s, want 77", c.MaxPerTradeUSDC)
	}
	if !c.MaxSlippageBps.Equal(dec("40")) {
		t.Errorf("MaxSlippageBps = %s, want 40", c.MaxSlippageBps)
	}
	if c.DryRun {
		t.Errorf("DryRun = true, want false")
	}
}

func TestConfig_MergeEnv_IgnoresInvalidAndEmpty(t *testing.T) {
	t.Setenv("BOT_SCAN_LIMIT", "not-a-number")
	t.Setenv("BOT_TOP_N", "-5")              // non-positive ignored
	t.Setenv("BOT_DEFAULT_AMOUNT_USDC", "0") // non-positive ignored
	t.Setenv("BOT_DRY_RUN", "yes")

	base := DefaultConfig()
	c := base.MergeEnv()
	if c.ScanLimit != base.ScanLimit {
		t.Errorf("ScanLimit changed on invalid input: %d", c.ScanLimit)
	}
	if c.TopN != base.TopN {
		t.Errorf("TopN changed on non-positive input: %d", c.TopN)
	}
	if !c.DefaultAmountUSDC.Equal(base.DefaultAmountUSDC) {
		t.Errorf("DefaultAmountUSDC changed on non-positive input: %s", c.DefaultAmountUSDC)
	}
	if !c.DryRun {
		t.Errorf("DryRun = false, want true for 'yes'")
	}
}

// ---- engine constructor ----

func newTestEngine(t *testing.T, client *stubClient, cfg Config) *Engine {
	t.Helper()
	e, err := NewEngine(client, testSigner(t), cfg)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	return e
}

func TestNewEngine_RejectsInvalidConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ScanLimit = 0
	if _, err := NewEngine(&stubClient{}, testSigner(t), cfg); err == nil {
		t.Fatal("expected NewEngine to reject invalid config")
	}
}

// ---- risk.go ----

func TestEvaluateRisk(t *testing.T) {
	tests := []struct {
		name         string
		openOrders   int
		ordersErr    error
		maxOpen      int
		wantCanTrade bool
		wantErr      bool
	}{
		{"under cap can trade", 2, nil, 5, true, false},
		{"at cap blocked", 5, nil, 5, false, false},
		{"over cap blocked", 9, nil, 5, false, false},
		{"orders error", 0, errors.New("boom"), 5, false, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &stubClient{
				ordersFn: func(ctx context.Context, req *clobtypes.OrdersRequest) (clobtypes.OrdersResponse, error) {
					if tc.ordersErr != nil {
						return clobtypes.OrdersResponse{}, tc.ordersErr
					}
					data := make([]clobtypes.OrderResponse, tc.openOrders)
					return clobtypes.OrdersResponse{Data: data}, nil
				},
			}
			cfg := DefaultConfig()
			cfg.MaxOpenTrades = tc.maxOpen
			e := newTestEngine(t, client, cfg)

			snap, err := e.EvaluateRisk(context.Background())
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if snap.CanTrade != tc.wantCanTrade {
				t.Fatalf("CanTrade = %v, want %v (reason=%q)", snap.CanTrade, tc.wantCanTrade, snap.Reason)
			}
			if snap.OpenOrders != tc.openOrders {
				t.Fatalf("OpenOrders = %d, want %d", snap.OpenOrders, tc.openOrders)
			}
		})
	}
}

func TestValidatePlanAgainstRisk_Cases(t *testing.T) {
	cfg := DefaultConfig()
	e := &Engine{cfg: cfg}
	good := &TradePlan{Side: "BUY", AmountUSDC: dec("10"), MaxAcceptedPrice: dec("0.51")}

	tests := []struct {
		name    string
		plan    *TradePlan
		risk    RiskSnapshot
		wantErr bool
	}{
		{"nil plan", nil, RiskSnapshot{CanTrade: true}, true},
		{"risk blocked", good, RiskSnapshot{CanTrade: false, Reason: "blocked"}, true},
		{"valid", good, RiskSnapshot{CanTrade: true}, false},
		{"sell side ok", &TradePlan{Side: "sell", AmountUSDC: dec("10"), MaxAcceptedPrice: dec("0.49")}, RiskSnapshot{CanTrade: true}, false},
		{"unsupported side", &TradePlan{Side: "HOLD", AmountUSDC: dec("10"), MaxAcceptedPrice: dec("0.5")}, RiskSnapshot{CanTrade: true}, true},
		{"price non-positive", &TradePlan{Side: "BUY", AmountUSDC: dec("10"), MaxAcceptedPrice: decimal.Zero}, RiskSnapshot{CanTrade: true}, true},
		{"amount over cap", &TradePlan{Side: "BUY", AmountUSDC: cfg.MaxPerTradeUSDC.Add(dec("1")), MaxAcceptedPrice: dec("0.5")}, RiskSnapshot{CanTrade: true}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := e.ValidatePlanAgainstRisk(tc.plan, tc.risk)
			if tc.wantErr != (err != nil) {
				t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

// ---- analyze.go ----

func bookWith(bids, asks []clobtypes.PriceLevel) clobtypes.OrderBookResponse {
	return clobtypes.OrderBookResponse{Bids: bids, Asks: asks}
}

func TestAnalyzeToken(t *testing.T) {
	market := clobtypes.Market{ID: "m1", Question: "q?"}
	token := clobtypes.MarketToken{TokenID: "t1", Outcome: "YES"}

	tests := []struct {
		name        string
		book        clobtypes.OrderBookResponse
		bookErr     error
		wantErr     bool
		wantRecomm  string
		checkSignal bool
	}{
		{
			name:       "bid-heavy book recommends BUY",
			book:       bookWith([]clobtypes.PriceLevel{{Price: "0.50", Size: "1000"}}, []clobtypes.PriceLevel{{Price: "0.55", Size: "100"}}),
			wantRecomm: "BUY",
		},
		{
			name:       "ask-heavy book recommends SELL",
			book:       bookWith([]clobtypes.PriceLevel{{Price: "0.50", Size: "100"}}, []clobtypes.PriceLevel{{Price: "0.55", Size: "1000"}}),
			wantRecomm: "SELL",
		},
		{
			name:    "empty book errors",
			book:    bookWith(nil, nil),
			wantErr: true,
		},
		{
			name:    "crossed book errors",
			book:    bookWith([]clobtypes.PriceLevel{{Price: "0.60", Size: "100"}}, []clobtypes.PriceLevel{{Price: "0.55", Size: "100"}}),
			wantErr: true,
		},
		{
			name:    "order book lookup error",
			bookErr: errors.New("down"),
			wantErr: true,
		},
		{
			name:    "bad price string errors",
			book:    bookWith([]clobtypes.PriceLevel{{Price: "abc", Size: "100"}}, []clobtypes.PriceLevel{{Price: "0.55", Size: "100"}}),
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := &stubClient{
				orderBook: func(ctx context.Context, req *clobtypes.BookRequest) (clobtypes.OrderBookResponse, error) {
					if tc.bookErr != nil {
						return clobtypes.OrderBookResponse{}, tc.bookErr
					}
					return tc.book, nil
				},
			}
			e := newTestEngine(t, client, DefaultConfig())
			op, err := e.analyzeToken(context.Background(), market, token)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if op.Recommended != tc.wantRecomm {
				t.Fatalf("Recommended = %q, want %q", op.Recommended, tc.wantRecomm)
			}
			if op.TokenID != "t1" {
				t.Fatalf("TokenID = %q, want t1", op.TokenID)
			}
			if op.Mid.LessThanOrEqual(decimal.Zero) {
				t.Fatalf("Mid should be positive, got %s", op.Mid)
			}
		})
	}
}

// ---- engine.go: BuildTradePlan ----

func TestBuildTradePlan(t *testing.T) {
	t.Run("rejects low confidence", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MinConfidenceBps = dec("100")
		e := &Engine{cfg: cfg}
		_, err := e.BuildTradePlan(Opportunity{ConfidenceBps: dec("10")})
		if err == nil {
			t.Fatal("expected confidence rejection")
		}
	})

	t.Run("amount capped by max-per-trade", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.DefaultAmountUSDC = dec("100")
		cfg.MaxPerTradeUSDC = dec("25") // smaller cap should win
		cfg.MinConfidenceBps = decimal.Zero
		e := &Engine{cfg: cfg}
		op := Opportunity{Recommended: "BUY", Mid: dec("0.50"), ConfidenceBps: dec("50"), TokenID: "t1"}
		plan, err := e.BuildTradePlan(op)
		if err != nil {
			t.Fatalf("BuildTradePlan: %v", err)
		}
		if !plan.AmountUSDC.Equal(dec("25")) {
			t.Fatalf("AmountUSDC = %s, want 25", plan.AmountUSDC)
		}
	})

	t.Run("BUY guard is above mid", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MinConfidenceBps = decimal.Zero
		cfg.MaxSlippageBps = dec("100")
		e := &Engine{cfg: cfg}
		op := Opportunity{Recommended: "BUY", Mid: dec("0.50"), ConfidenceBps: dec("50")}
		plan, err := e.BuildTradePlan(op)
		if err != nil {
			t.Fatalf("BuildTradePlan: %v", err)
		}
		if plan.Side != "BUY" {
			t.Fatalf("Side = %q, want BUY", plan.Side)
		}
		if !plan.MaxAcceptedPrice.GreaterThan(op.Mid) {
			t.Fatalf("BUY guard %s should be > mid %s", plan.MaxAcceptedPrice, op.Mid)
		}
	})

	t.Run("SELL guard is below mid", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MinConfidenceBps = decimal.Zero
		cfg.MaxSlippageBps = dec("100")
		e := &Engine{cfg: cfg}
		op := Opportunity{Recommended: "SELL", Mid: dec("0.50"), ConfidenceBps: dec("50")}
		plan, err := e.BuildTradePlan(op)
		if err != nil {
			t.Fatalf("BuildTradePlan: %v", err)
		}
		if plan.Side != "SELL" {
			t.Fatalf("Side = %q, want SELL", plan.Side)
		}
		if !plan.MaxAcceptedPrice.LessThan(op.Mid) {
			t.Fatalf("SELL guard %s should be < mid %s", plan.MaxAcceptedPrice, op.Mid)
		}
	})

	t.Run("SELL guard floors at 0.01 when slippage would push non-positive", func(t *testing.T) {
		// mid 0.001 with 100% slippage -> guard would be 0 -> must floor to 0.01.
		op := Opportunity{Recommended: "SELL", Mid: dec("0.001")}
		guard := slippageGuardPrice(op, dec("10000"))
		if !guard.Equal(dec("0.01")) {
			t.Fatalf("expected 0.01 floor, got %s", guard)
		}
	})
}

// ---- engine.go: ExecutePlan ----

func TestExecutePlan_NilPlan(t *testing.T) {
	e := newTestEngine(t, &stubClient{}, DefaultConfig())
	if _, err := e.ExecutePlan(context.Background(), nil); err == nil {
		t.Fatal("expected error for nil plan")
	}
}

func TestExecutePlan_DryRunGating(t *testing.T) {
	plan := &TradePlan{TokenID: "1", Side: "BUY", AmountUSDC: dec("10"), MaxAcceptedPrice: dec("0.50")}

	tests := []struct {
		name           string
		dryRun         bool
		allowExecution bool
		wantBlocked    bool
	}{
		{"dry-run blocks", true, true, true},
		{"execution disallowed blocks", false, false, true},
		{"dry-run and disallowed blocks", true, false, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.DryRun = tc.dryRun
			cfg.AllowExecution = tc.allowExecution
			called := false
			client := &stubClient{
				createFn: func(ctx context.Context, order *clobtypes.SignableOrder) (clobtypes.OrderResponse, error) {
					called = true
					return clobtypes.OrderResponse{}, nil
				},
			}
			e := newTestEngine(t, client, cfg)
			_, err := e.ExecutePlan(context.Background(), plan)
			if tc.wantBlocked {
				if err == nil {
					t.Fatal("expected execution to be blocked")
				}
				if called {
					t.Fatal("client should not be called when blocked")
				}
			}
		})
	}
}

func TestExecutePlan_AllowExecutionSubmits(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DryRun = false
	cfg.AllowExecution = true

	var submitted *clobtypes.SignableOrder
	client := &stubClient{
		tickSizeFn: func(ctx context.Context, req *clobtypes.TickSizeRequest) (clobtypes.TickSizeResponse, error) {
			return clobtypes.TickSizeResponse{MinimumTickSize: 0.01}, nil
		},
		negRiskFn: func(ctx context.Context, req *clobtypes.NegRiskRequest) (clobtypes.NegRiskResponse, error) {
			return clobtypes.NegRiskResponse{NegRisk: false}, nil
		},
		createFn: func(ctx context.Context, order *clobtypes.SignableOrder) (clobtypes.OrderResponse, error) {
			submitted = order
			return clobtypes.OrderResponse{ID: "order-1", Status: "live"}, nil
		},
	}
	e := newTestEngine(t, client, cfg)

	plan := &TradePlan{
		TokenID:          "1234",
		Side:             "BUY",
		AmountUSDC:       dec("10"),
		MaxAcceptedPrice: dec("0.55"),
	}
	resp, err := e.ExecutePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("ExecutePlan: %v", err)
	}
	if resp.ID != "order-1" {
		t.Fatalf("resp.ID = %q, want order-1", resp.ID)
	}
	if submitted == nil || submitted.Order == nil {
		t.Fatal("expected a signable order to be submitted")
	}
	if submitted.Order.Side != "BUY" {
		t.Fatalf("submitted side = %q, want BUY", submitted.Order.Side)
	}
	if submitted.OrderType != clobtypes.OrderTypeFAK {
		t.Fatalf("submitted order type = %q, want FAK", submitted.OrderType)
	}
}

// ---- engine.go: ScanOpportunities end-to-end ----

func TestScanOpportunities(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ScanLimit = 10
	cfg.TopN = 5
	cfg.RequestTimeout = 2 * time.Second
	// Loosen filters so the constructed opportunity survives.
	cfg.MinSpreadBps = dec("1")
	cfg.MinBookDepthShares = dec("10")
	cfg.MinImbalance = dec("0.01")

	market := clobtypes.Market{
		ID:       "m1",
		Question: "Will it rain?",
		Tokens:   []clobtypes.MarketToken{{TokenID: "t1", Outcome: "YES"}},
	}

	client := &stubClient{
		marketsFn: func(ctx context.Context, req *clobtypes.MarketsRequest) (clobtypes.MarketsResponse, error) {
			if req.Active == nil || !*req.Active {
				t.Errorf("expected Active=true filter")
			}
			return clobtypes.MarketsResponse{Data: []clobtypes.Market{market}}, nil
		},
		orderBook: func(ctx context.Context, req *clobtypes.BookRequest) (clobtypes.OrderBookResponse, error) {
			// Strongly bid-imbalanced, wide spread.
			return bookWith(
				[]clobtypes.PriceLevel{{Price: "0.50", Size: "5000"}},
				[]clobtypes.PriceLevel{{Price: "0.60", Size: "100"}},
			), nil
		},
	}
	e := newTestEngine(t, client, cfg)

	opps, err := e.ScanOpportunities(context.Background())
	if err != nil {
		t.Fatalf("ScanOpportunities: %v", err)
	}
	if len(opps) != 1 {
		t.Fatalf("expected 1 opportunity, got %d", len(opps))
	}
	if opps[0].Recommended != "BUY" {
		t.Fatalf("Recommended = %q, want BUY", opps[0].Recommended)
	}
}

func TestScanOpportunities_MarketsError(t *testing.T) {
	client := &stubClient{
		marketsFn: func(ctx context.Context, req *clobtypes.MarketsRequest) (clobtypes.MarketsResponse, error) {
			return clobtypes.MarketsResponse{}, errors.New("markets down")
		},
	}
	e := newTestEngine(t, client, DefaultConfig())
	if _, err := e.ScanOpportunities(context.Background()); err == nil {
		t.Fatal("expected markets error to propagate")
	}
}

func TestScanOpportunities_FiltersByThresholds(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MinSpreadBps = dec("100000") // impossibly high spread requirement
	market := clobtypes.Market{ID: "m1", Tokens: []clobtypes.MarketToken{{TokenID: "t1"}}}
	client := &stubClient{
		marketsFn: func(ctx context.Context, req *clobtypes.MarketsRequest) (clobtypes.MarketsResponse, error) {
			return clobtypes.MarketsResponse{Data: []clobtypes.Market{market}}, nil
		},
		orderBook: func(ctx context.Context, req *clobtypes.BookRequest) (clobtypes.OrderBookResponse, error) {
			return bookWith(
				[]clobtypes.PriceLevel{{Price: "0.50", Size: "5000"}},
				[]clobtypes.PriceLevel{{Price: "0.51", Size: "100"}},
			), nil
		},
	}
	e := newTestEngine(t, client, cfg)
	opps, err := e.ScanOpportunities(context.Background())
	if err != nil {
		t.Fatalf("ScanOpportunities: %v", err)
	}
	if len(opps) != 0 {
		t.Fatalf("expected 0 opportunities after spread filter, got %d", len(opps))
	}
}
