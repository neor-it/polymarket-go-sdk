package clob

import (
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"

	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
	"github.com/neor-it/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/neor-it/polymarket-go-sdk/pkg/transport"
	"github.com/neor-it/polymarket-go-sdk/pkg/types"
)

func TestOrderManagementMethods(t *testing.T) {
	signer, _ := auth.NewPrivateKeySigner("0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318", 137)
	apiKey := &auth.APIKey{Key: "k1", Secret: "s1", Passphrase: "p1"}
	ctx := context.Background()

	t.Run("PostOrder", func(t *testing.T) {
		doer := &staticDoer{
			responses: map[string]string{"/order": `{"orderID":"o1","status":"OK"}`},
		}
		client := &clientImpl{
			httpClient: transport.NewClient(doer, "http://example"),
			signer:     signer,
			apiKey:     apiKey,
		}
		order := &clobtypes.SignedOrder{
			Order:     clobtypes.Order{Side: "BUY"},
			Signature: "0x123",
			Owner:     "0xabc",
		}
		resp, err := client.PostOrder(ctx, order)
		if err != nil || resp.ID != "o1" {
			t.Errorf("PostOrder failed: %v", err)
		}
	})

	t.Run("PostOrderResolvesTradeIDs", func(t *testing.T) {
		doer := &staticDoer{
			responses: map[string]string{
				"/order":                          `{"orderID":"o1","status":"matched","tradeIDs":["trade-1"]}`,
				"/data/trades?id=trade-1&limit=1": `{"data":[{"id":"trade-1","status":"TRADE_STATUS_CONFIRMED","transaction_hash":"0xabc"}]}`,
			},
		}
		client := &clientImpl{
			httpClient: transport.NewClient(doer, "http://example"),
			signer:     signer,
			apiKey:     apiKey,
		}
		order := &clobtypes.SignedOrder{
			Order:     clobtypes.Order{Side: "BUY"},
			Signature: "0x123",
			Owner:     "0xabc",
			OrderType: clobtypes.OrderTypeFOK,
		}
		resp, err := client.PostOrder(ctx, order)
		if err != nil {
			t.Fatalf("PostOrder failed: %v", err)
		}
		if len(resp.TransactionHashes) != 1 || resp.TransactionHashes[0] != "0xabc" {
			t.Fatalf("TransactionHashes = %v, want [0xabc]", resp.TransactionHashes)
		}
		if len(resp.TradeIDs) != 1 || resp.TradeIDs[0] != "trade-1" {
			t.Fatalf("TradeIDs = %v, want [trade-1]", resp.TradeIDs)
		}
	})

	t.Run("PostOrderKeepsInlineTransactionHashes", func(t *testing.T) {
		doer := &staticDoer{
			responses: map[string]string{
				"/order": `{"orderID":"o1","status":"matched","transactionsHashes":["0xinline"],"tradeIDs":["trade-1"]}`,
			},
		}
		client := &clientImpl{
			httpClient: transport.NewClient(doer, "http://example"),
			signer:     signer,
			apiKey:     apiKey,
		}
		order := &clobtypes.SignedOrder{
			Order:     clobtypes.Order{Side: "BUY"},
			Signature: "0x123",
			Owner:     "0xabc",
			OrderType: clobtypes.OrderTypeFOK,
		}
		resp, err := client.PostOrder(ctx, order)
		if err != nil {
			t.Fatalf("PostOrder failed: %v", err)
		}
		if len(resp.TransactionHashes) != 1 || resp.TransactionHashes[0] != "0xinline" {
			t.Fatalf("TransactionHashes = %v, want [0xinline]", resp.TransactionHashes)
		}
	})

	t.Run("PostOrderSkipsResolutionForDeferExec", func(t *testing.T) {
		deferExec := true
		doer := &staticDoer{
			responses: map[string]string{
				"/order": `{"orderID":"o1","status":"matched","tradeIDs":["trade-1"]}`,
			},
		}
		client := &clientImpl{
			httpClient: transport.NewClient(doer, "http://example"),
			signer:     signer,
			apiKey:     apiKey,
		}
		order := &clobtypes.SignedOrder{
			Order:     clobtypes.Order{Side: "BUY"},
			Signature: "0x123",
			Owner:     "0xabc",
			OrderType: clobtypes.OrderTypeFOK,
			DeferExec: &deferExec,
		}
		resp, err := client.PostOrder(ctx, order)
		if err != nil {
			t.Fatalf("PostOrder failed: %v", err)
		}
		if len(resp.TransactionHashes) != 0 {
			t.Fatalf("TransactionHashes = %v, want empty", resp.TransactionHashes)
		}
	})

	t.Run("PostOrderDoesNotFailWhenTradePollingFails", func(t *testing.T) {
		pollCtx, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
		defer cancel()
		doer := &staticDoer{
			responses: map[string]string{
				"/order": `{"orderID":"o1","status":"matched","tradeIDs":["trade-1"]}`,
			},
		}
		client := &clientImpl{
			httpClient: transport.NewClient(doer, "http://example"),
			signer:     signer,
			apiKey:     apiKey,
		}
		order := &clobtypes.SignedOrder{
			Order:     clobtypes.Order{Side: "BUY"},
			Signature: "0x123",
			Owner:     "0xabc",
			OrderType: clobtypes.OrderTypeFOK,
		}
		resp, err := client.PostOrder(pollCtx, order)
		if err != nil {
			t.Fatalf("PostOrder must keep successful submit response: %v", err)
		}
		if resp.ID != "o1" {
			t.Fatalf("ID = %s, want o1", resp.ID)
		}
		if len(resp.TransactionHashes) != 0 {
			t.Fatalf("TransactionHashes = %v, want empty", resp.TransactionHashes)
		}
	})

	t.Run("PostOrdersResolvesTradeIDsPerOrder", func(t *testing.T) {
		doer := &staticDoer{
			responses: map[string]string{
				"/orders":                         `[{"orderID":"o1","status":"matched","tradeIDs":["trade-1"]},{"orderID":"o2","status":"live"}]`,
				"/data/trades?id=trade-1&limit=1": `{"data":[{"id":"trade-1","status":"TRADE_STATUS_CONFIRMED","transaction_hash":"0xabc"}]}`,
			},
		}
		client := &clientImpl{
			httpClient: transport.NewClient(doer, "http://example"),
			signer:     signer,
			apiKey:     apiKey,
		}
		resp, err := client.PostOrders(ctx, &clobtypes.SignedOrders{
			Orders: []clobtypes.SignedOrder{
				{
					Order:     clobtypes.Order{Side: "BUY"},
					Signature: "0x123",
					Owner:     "0xabc",
					OrderType: clobtypes.OrderTypeFOK,
				},
				{
					Order:     clobtypes.Order{Side: "SELL"},
					Signature: "0x456",
					Owner:     "0xabc",
					OrderType: clobtypes.OrderTypeGTC,
				},
			},
		})
		if err != nil {
			t.Fatalf("PostOrders failed: %v", err)
		}
		if len(resp) != 2 {
			t.Fatalf("len(resp) = %d, want 2", len(resp))
		}
		if len(resp[0].TransactionHashes) != 1 || resp[0].TransactionHashes[0] != "0xabc" {
			t.Fatalf("resp[0].TransactionHashes = %v, want [0xabc]", resp[0].TransactionHashes)
		}
		if len(resp[1].TransactionHashes) != 0 {
			t.Fatalf("resp[1].TransactionHashes = %v, want empty", resp[1].TransactionHashes)
		}
	})

	t.Run("PostOrderExcludesFailedTradesFromHashes", func(t *testing.T) {
		doer := &staticDoer{
			responses: map[string]string{
				"/order":                          `{"orderID":"o1","status":"matched","tradeIDs":["trade-1"]}`,
				"/data/trades?id=trade-1&limit=1": `{"data":[{"id":"trade-1","status":"TRADE_STATUS_FAILED"}]}`,
			},
		}
		client := &clientImpl{
			httpClient: transport.NewClient(doer, "http://example"),
			signer:     signer,
			apiKey:     apiKey,
		}
		order := &clobtypes.SignedOrder{
			Order:     clobtypes.Order{Side: "BUY"},
			Signature: "0x123",
			Owner:     "0xabc",
			OrderType: clobtypes.OrderTypeFOK,
		}
		resp, err := client.PostOrder(ctx, order)
		if err != nil {
			t.Fatalf("PostOrder failed: %v", err)
		}
		if len(resp.TransactionHashes) != 0 {
			t.Fatalf("TransactionHashes = %v, want empty", resp.TransactionHashes)
		}
	})

	t.Run("CancelAll", func(t *testing.T) {
		doer := &staticDoer{
			responses: map[string]string{"/cancel-all": `{"canceled":["o1","o2"]}`},
		}
		client := &clientImpl{
			httpClient: transport.NewClient(doer, "http://example"),
		}
		resp, err := client.CancelAll(ctx)
		if err != nil {
			t.Errorf("CancelAll failed: %v", err)
		}
		if len(resp.Canceled) != 2 {
			t.Errorf("expected 2 canceled, got %d", len(resp.Canceled))
		}
	})

	t.Run("CancelOrder", func(t *testing.T) {
		doer := &staticDoer{
			responses: map[string]string{"/order": `{"canceled":["o1"]}`},
		}
		client := &clientImpl{
			httpClient: transport.NewClient(doer, "http://example"),
		}
		resp, err := client.CancelOrder(ctx, &clobtypes.CancelOrderRequest{OrderID: "o1"})
		if err != nil {
			t.Errorf("CancelOrder failed: %v", err)
		}
		if len(resp.Canceled) != 1 || resp.Canceled[0] != "o1" {
			t.Errorf("expected canceled [o1], got %v", resp.Canceled)
		}
	})

	t.Run("CancelOrders", func(t *testing.T) {
		doer := &staticDoer{
			responses: map[string]string{"/orders": `{"canceled":["o1"]}`},
		}
		client := &clientImpl{
			httpClient: transport.NewClient(doer, "http://example"),
		}
		resp, err := client.CancelOrders(ctx, &clobtypes.CancelOrdersRequest{OrderIDs: []string{"o1"}})
		if err != nil {
			t.Errorf("CancelOrders failed: %v", err)
		}
		if len(resp.Canceled) != 1 || resp.Canceled[0] != "o1" {
			t.Errorf("expected canceled [o1], got %v", resp.Canceled)
		}
	})

	t.Run("CancelMarketOrders", func(t *testing.T) {
		doer := &staticDoer{
			responses: map[string]string{"/cancel-market-orders": `{"canceled":["o1"]}`},
		}
		client := &clientImpl{
			httpClient: transport.NewClient(doer, "http://example"),
		}
		resp, err := client.CancelMarketOrders(ctx, &clobtypes.CancelMarketOrdersRequest{Market: "m1"})
		if err != nil {
			t.Errorf("CancelMarketOrders failed: %v", err)
		}
		if len(resp.Canceled) != 1 || resp.Canceled[0] != "o1" {
			t.Errorf("expected canceled [o1], got %v", resp.Canceled)
		}
	})

	t.Run("BuilderTrades", func(t *testing.T) {
		doer := &staticDoer{
			responses: map[string]string{"/builder/trades": `{"data":[]}`},
		}
		client := &clientImpl{
			httpClient: transport.NewClient(doer, "http://example"),
		}
		resp, err := client.BuilderTrades(ctx, nil)
		if err != nil {
			t.Errorf("BuilderTrades failed: %v", err)
		}
		if resp.Data == nil {
			t.Errorf("expected empty slice")
		}
	})

	t.Run("OrderLookup", func(t *testing.T) {
		doer := &staticDoer{
			responses: map[string]string{"/data/order/o1": `{"orderID":"o1","status":"OK"}`},
		}
		client := &clientImpl{
			httpClient: transport.NewClient(doer, "http://example"),
		}
		resp, err := client.Order(ctx, "o1")
		if err != nil || resp.ID != "o1" {
			t.Errorf("Order lookup failed: %v", err)
		}
	})

	t.Run("OrdersList", func(t *testing.T) {
		doer := &staticDoer{
			responses: map[string]string{"/data/orders": `{"data":[{"id":"o1"}],"next_cursor":"LTE="}`},
		}
		client := &clientImpl{
			httpClient: transport.NewClient(doer, "http://example"),
		}
		resp, err := client.Orders(ctx, nil)
		if err != nil {
			t.Fatalf("Orders list failed: %v", err)
		}
		if len(resp.Data) == 0 {
			t.Fatal("Orders list returned no data")
		}
		if resp.Data[0].ID != "o1" {
			t.Errorf("Orders list ID = %s, want o1", resp.Data[0].ID)
		}
	})

	t.Run("OrdersListNumericCreatedAt", func(t *testing.T) {
		doer := &staticDoer{
			responses: map[string]string{"/data/orders": `{"data":[{"orderID":"o1","created_at":1700000000,"timestamp":1700000001}],"next_cursor":"LTE="}`},
		}
		client := &clientImpl{
			httpClient: transport.NewClient(doer, "http://example"),
		}
		resp, err := client.Orders(ctx, nil)
		if err != nil {
			t.Fatalf("Orders list failed: %v", err)
		}
		if len(resp.Data) != 1 {
			t.Fatalf("len(resp.Data) = %d, want 1", len(resp.Data))
		}
		if resp.Data[0].CreatedAt != "1700000000" {
			t.Errorf("CreatedAt = %s, want 1700000000", resp.Data[0].CreatedAt)
		}
		if resp.Data[0].Timestamp != "1700000001" {
			t.Errorf("Timestamp = %s, want 1700000001", resp.Data[0].Timestamp)
		}
	})

	t.Run("OrderScoring", func(t *testing.T) {
		doer := &staticDoer{
			responses: map[string]string{"/order-scoring?order_id=o1": `{"scoring":true}`},
		}
		client := &clientImpl{
			httpClient: transport.NewClient(doer, "http://example"),
		}
		resp, err := client.OrderScoring(ctx, &clobtypes.OrderScoringRequest{ID: "o1"})
		if err != nil || !resp.Scoring {
			t.Errorf("OrderScoring failed: %v", err)
		}
	})

	t.Run("OrdersScoring", func(t *testing.T) {
		doer := &staticDoer{
			responses: map[string]string{"/orders-scoring": `{"o1":true,"o2":false}`},
		}
		client := &clientImpl{
			httpClient: transport.NewClient(doer, "http://example"),
		}
		resp, err := client.OrdersScoring(ctx, &clobtypes.OrdersScoringRequest{IDs: []string{"o1", "o2"}})
		if err != nil || resp["o1"] != true || resp["o2"] != false {
			t.Errorf("OrdersScoring failed: %v", err)
		}
	})
}

func TestSignOrderDefaults(t *testing.T) {
	signer, _ := auth.NewPrivateKeySigner("0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318", 137)
	apiKey := &auth.APIKey{Key: "k1", Secret: "s1", Passphrase: "p1"}

	funder := common.HexToAddress("0x3333333333333333333333333333333333333333")
	client := &clientImpl{
		signer:        signer,
		apiKey:        apiKey,
		signatureType: auth.SignatureProxy,
		funder:        &funder,
		saltGenerator: func() (*big.Int, error) { return big.NewInt(7), nil },
		builderCfg: &auth.BuilderConfig{
			Code: "0x0000000000000000000000000000000000000000000000000000000000000001",
		},
	}

	order := &clobtypes.Order{
		Side:        "BUY",
		TokenID:     types.U256{Int: big.NewInt(1)},
		MakerAmount: decimal.NewFromInt(10),
		TakerAmount: decimal.NewFromInt(5),
		FeeRateBps:  decimal.NewFromInt(0),
		Nonce:       types.U256{Int: big.NewInt(1)},
		Expiration:  types.U256{Int: big.NewInt(0)},
		Taker:       common.Address{},
		Signer:      signer.Address(),
	}

	signed, err := client.signOrder(order)
	if err != nil {
		t.Fatalf("signOrder failed: %v", err)
	}
	if signed.Order.SignatureType == nil || *signed.Order.SignatureType != 1 {
		t.Fatalf("signature type mismatch: %+v", signed.Order.SignatureType)
	}
	if signed.Order.Maker != funder {
		t.Fatalf("maker mismatch: got %s want %s", signed.Order.Maker.Hex(), funder.Hex())
	}
	if signed.Order.Salt.Int == nil || signed.Order.Salt.Int.Int64() != 7 {
		t.Fatalf("salt mismatch: got %v", signed.Order.Salt.Int)
	}
	if signed.Order.Timestamp == 0 {
		t.Fatal("timestamp must be set for CLOB v2 orders")
	}
	if signed.Order.Builder != common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001") {
		t.Fatalf("builder mismatch: got %s", signed.Order.Builder.Hex())
	}
}

func TestSignOrderRejectsInvalidBuilderCode(t *testing.T) {
	signer, _ := auth.NewPrivateKeySigner("0x4c0883a69102937d6231471b5dbb6204fe5129617082792ae468d01a3f362318", 137)
	apiKey := &auth.APIKey{Key: "k1", Secret: "s1", Passphrase: "p1"}

	client := &clientImpl{
		signer:     signer,
		apiKey:     apiKey,
		builderCfg: &auth.BuilderConfig{Code: "0x1234"},
	}
	order := &clobtypes.Order{
		Side:        "BUY",
		TokenID:     types.U256{Int: big.NewInt(1)},
		MakerAmount: decimal.NewFromInt(10),
		TakerAmount: decimal.NewFromInt(5),
		Signer:      signer.Address(),
	}

	_, err := client.signOrder(order)
	if err == nil || !strings.Contains(err.Error(), "builder code") {
		t.Fatalf("expected builder code validation error, got %v", err)
	}
}

func TestPostOrders_BatchSizeValidation(t *testing.T) {
	ctx := context.Background()
	client := &clientImpl{
		httpClient: transport.NewClient(&staticDoer{responses: map[string]string{}}, "http://example"),
	}

	// Exactly at the limit should not error (would error from server, but not from validation)
	atLimit := &clobtypes.SignedOrders{
		Orders: make([]clobtypes.SignedOrder, clobtypes.MaxPostOrdersBatchSize),
	}
	// This will fail at the HTTP level but should NOT fail at validation
	_, err := client.PostOrders(ctx, atLimit)
	if err != nil && strings.Contains(err.Error(), "batch size") {
		t.Errorf("expected no batch size error at limit, got: %v", err)
	}

	// Over the limit should error immediately
	overLimit := &clobtypes.SignedOrders{
		Orders: make([]clobtypes.SignedOrder, clobtypes.MaxPostOrdersBatchSize+1),
	}
	_, err = client.PostOrders(ctx, overLimit)
	if err == nil {
		t.Fatal("expected error for exceeding batch size")
	}
	if !strings.Contains(err.Error(), "batch size") {
		t.Errorf("expected batch size error, got: %v", err)
	}
}

func TestSignOrderPoly1271WrappedSignature(t *testing.T) {
	// Foundry default key #0 — deterministic ECDSA via RFC 6979.
	signer, err := auth.NewPrivateKeySigner("0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", 137)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}
	apiKey := &auth.APIKey{Key: "k1", Secret: "s1", Passphrase: "p1"}
	funder := common.HexToAddress("0x9c90cad2e22a1E9b4a9aB3F95f7f14d08Ce78ade")
	sigType := int(auth.SignaturePoly1271)

	order := &clobtypes.Order{
		Salt:          types.U256{Int: big.NewInt(123)},
		Maker:         funder,
		Signer:        funder,
		TokenID:       types.U256{Int: new(big.Int).SetBytes(common.FromHex("0x1234"))},
		MakerAmount:   decimal.NewFromInt(5_000_000),
		TakerAmount:   decimal.NewFromInt(10_000_000),
		Side:          "BUY",
		SignatureType: &sigType,
		Timestamp:     1700000000123,
	}

	signed, err := signOrderWithCreds(signer, apiKey, order, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("signOrderWithCreds failed: %v", err)
	}

	if signed.Order.SignatureType == nil || *signed.Order.SignatureType != 3 {
		t.Fatalf("signature type mismatch: %+v", signed.Order.SignatureType)
	}
	if signed.Order.Maker != funder {
		t.Fatalf("maker mismatch: got %s", signed.Order.Maker.Hex())
	}
	if signed.Order.Signer != funder {
		t.Fatalf("signer mismatch: got %s, want deposit wallet %s", signed.Order.Signer.Hex(), funder.Hex())
	}
	if !strings.HasPrefix(signed.Signature, "0x") {
		t.Fatalf("signature must start with 0x")
	}

	// Wrapped signature length: 65 (ECDSA) + 32 (domainSep) + 32 (contentsHash) + len(orderTypeStr) + 2 (uint16)
	sigBytes := common.FromHex(signed.Signature)
	expectedLen := 65 + 32 + 32 + len(orderTypeStr) + 2
	if len(sigBytes) != expectedLen {
		t.Fatalf("wrapped signature length = %d, want %d", len(sigBytes), expectedLen)
	}
}

func TestCancelOrders_BatchSizeValidation(t *testing.T) {
	ctx := context.Background()
	client := &clientImpl{
		httpClient: transport.NewClient(&staticDoer{responses: map[string]string{}}, "http://example"),
	}

	// Over the limit should error immediately
	ids := make([]string, clobtypes.MaxCancelOrdersBatchSize+1)
	for i := range ids {
		ids[i] = "order-" + strings.Repeat("x", 5)
	}
	_, err := client.CancelOrders(ctx, &clobtypes.CancelOrdersRequest{OrderIDs: ids})
	if err == nil {
		t.Fatal("expected error for exceeding cancel batch size")
	}
	if !strings.Contains(err.Error(), "batch size") {
		t.Errorf("expected batch size error, got: %v", err)
	}
}
