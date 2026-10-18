package bot

import (
	"context"
	"testing"

	"github.com/neor-it/polymarket-go-sdk/pkg/clob/clobtypes"
	"github.com/neor-it/polymarket-go-sdk/pkg/types"
)

func TestExecutionPreservesSelectedProtocol(t *testing.T) {
	for _, protocol := range []types.ProtocolVersion{types.ProtocolV1, types.ProtocolV2, "", "unknown"} {
		t.Run(string(protocol), func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.DryRun, cfg.AllowExecution = false, true
			cfg.MaxSlippageBps = dec("2000")
			submitted := false
			client := &stubClient{
				tickSizeFn: func(context.Context, *clobtypes.TickSizeRequest) (clobtypes.TickSizeResponse, error) {
					return clobtypes.TickSizeResponse{MinimumTickSize: 0.01}, nil
				},
				negRiskFn: func(context.Context, *clobtypes.NegRiskRequest) (clobtypes.NegRiskResponse, error) {
					return clobtypes.NegRiskResponse{}, nil
				},
				createFn: func(_ context.Context, order *clobtypes.SignableOrder) (clobtypes.OrderResponse, error) {
					submitted = true
					expected := protocol
					if expected == "" {
						expected = types.ProtocolV1
					}
					if order.Order.ProtocolVersion != expected {
						t.Fatalf("submitted protocol %q, want %q", order.Order.ProtocolVersion, expected)
					}
					return clobtypes.OrderResponse{}, nil
				},
			}
			engine := newTestEngine(t, client, cfg)
			plan, err := engine.BuildTradePlan(Opportunity{ProtocolVersion: protocol, TokenID: "1234", Recommended: "BUY", Mid: dec("0.5"), ConfidenceBps: cfg.MinConfidenceBps})
			if err != nil {
				t.Fatal(err)
			}
			if plan.ProtocolVersion != protocol {
				t.Fatalf("plan changed selected protocol: %q", plan.ProtocolVersion)
			}
			// Protocol propagation is independent of the strategy's price rounding.
			plan.MaxAcceptedPrice = dec("0.55")
			_, err = engine.ExecutePlan(context.Background(), plan)
			if protocol == "unknown" {
				if err == nil || submitted {
					t.Fatalf("unknown protocol reached submission: err=%v submitted=%t", err, submitted)
				}
				return
			}
			if err != nil || !submitted {
				t.Fatalf("execution failed: err=%v submitted=%t", err, submitted)
			}
		})
	}
}
