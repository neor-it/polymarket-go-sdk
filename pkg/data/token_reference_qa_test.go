package data_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neor-it/polymarket-go-sdk/pkg/data"
	"github.com/neor-it/polymarket-go-sdk/pkg/transport"
)

func TestQAHistoricalTokenLookupNeverInventsMissingYesOutcome(t *testing.T) {
	const token = "101"
	base := map[string]any{"token_id": token, "condition_id": "0x0000000000000000000000000000000000000000000000000000000000000001", "module": "v1_ctf", "outcome_index": 0, "resolved": false}
	for _, field := range []string{"token_id", "condition_id", "module", "outcome_index", "resolved"} {
		for _, null := range []bool{false, true} {
			t.Run(field+map[bool]string{false: "_missing", true: "_null"}[null], func(t *testing.T) {
				row := make(map[string]any, len(base))
				for key, value := range base {
					row[key] = value
				}
				if null {
					row[field] = nil
				} else {
					delete(row, field)
				}
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/v2/tokens" || r.URL.Query().Get("token_id") != token {
						t.Errorf("wrong lookup %s", r.URL)
					}
					if err := json.NewEncoder(w).Encode(map[string]any{"data": []any{row}}); err != nil {
						t.Error(err)
					}
				}))
				defer server.Close()
				client := data.NewClient(transport.NewClient(server.Client(), server.URL))
				_, err := client.Tokens(context.Background(), &data.TokensRequest{TokenID: qaPointer(token)})
				if err == nil {
					t.Fatal("incomplete provenance became authoritative token identity")
				}
			})
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(map[string]any{"data": []any{base}}); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	client := data.NewClient(transport.NewClient(server.Client(), server.URL))
	response, err := client.Tokens(context.Background(), &data.TokensRequest{TokenID: qaPointer(token)})
	if err != nil || len(response.Data) != 1 || response.Data[0].OutcomeIndex != 0 || response.Data[0].Resolved {
		t.Fatalf("explicit zero/false rejected: %+v %v", response, err)
	}
}
