package data

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neor-it/polymarket-go-sdk/pkg/transport"
)

func TestEndpointEnvelopeCannotInventMissingValue(t *testing.T) {
	for _, payload := range []string{`{}`, `null`, `{"data":null}`, `{"data":{}}`} {
		t.Run(payload, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if _, err := w.Write([]byte(payload)); err != nil {
					t.Errorf("write test response: %v", err)
				}
			}))
			defer server.Close()
			_, err := NewClient(transport.NewClient(server.Client(), server.URL)).Value(context.Background(), &ValueRequest{})
			if err == nil {
				t.Fatalf("unavailable portfolio value invented zero: %s", payload)
			}
		})
	}
}

func TestEndpointPageRequiresArrayAndCompletePagination(t *testing.T) {
	for _, payload := range []string{
		`{}`, `null`, `{"data":null}`, `{"data":[]}`, `{"data":[],"pagination":null}`, `{"data":[],"pagination":{}}`,
		`{"data":[],"pagination":{"limit":2,"offset":0}}`,
		`{"data":[],"pagination":{"limit":2,"offset":0,"has_more":true}}`,
		`{"data":[],"pagination":{"limit":2,"offset":0,"has_more":true,"next_cursor":""}}`,
		`{"data":[],"pagination":{"limit":2,"offset":-1,"has_more":false}}`,
		`{"data":[],"pagination":{"limit":2,"offset":0,"has_more":false,"next_cursor":"unexpected"}}`,
	} {
		t.Run(payload, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if _, err := w.Write([]byte(payload)); err != nil {
					t.Errorf("write test response: %v", err)
				}
			}))
			defer server.Close()
			_, err := NewClient(transport.NewClient(server.Client(), server.URL)).Trades(context.Background(), &TradesRequest{})
			if err == nil {
				t.Fatalf("incomplete page became empty success: %s", payload)
			}
		})
	}
}

func TestNullableEnvelopeNeedsExplicitData(t *testing.T) {
	for _, payload := range []string{`{}`, `null`, `{"data":null}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := w.Write([]byte(payload)); err != nil {
				t.Errorf("write test response: %v", err)
			}
		}))
		client := NewClient(transport.NewClient(server.Client(), server.URL))
		stats, err := client.UserStats(context.Background(), &UserStatsRequest{})
		server.Close()
		if payload == `{"data":null}` {
			if err != nil || stats.Data != nil {
				t.Fatalf("documented missing user rejected %+v %v", stats, err)
			}
			continue
		}
		if err == nil {
			t.Fatalf("missing envelope accepted: %s", payload)
		}
	}
}
