package transport

import (
	"context"
	"errors"
	"github.com/neor-it/polymarket-go-sdk/pkg/types"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRetryAfterParsing(t *testing.T) {
	now := time.Date(2026, time.October, 18, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		header string
		want   time.Duration
	}{{"2", 2 * time.Second}, {"-1", 0}, {"invalid", 0}, {now.Add(3 * time.Second).Format(http.TimeFormat), 3 * time.Second}, {now.Add(-time.Second).Format(http.TimeFormat), 0}}
	for _, test := range tests {
		if got := parseRetryAfter(test.header, now); got != test.want {
			t.Errorf("%q: %s != %s", test.header, got, test.want)
		}
	}
}
func TestRetryAfterHonorsContextCancellation(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Retry-After", "3600")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := NewClient(server.Client(), server.URL).Get(ctx, "/v2/trades", nil, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error %v", err)
	}
	if calls != 1 {
		t.Fatalf("server Retry-After ignored: %d calls", calls)
	}
}

func TestDataV2RetryExhaustionPreservesStructuredError(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusTooManyRequests)
		if _, err := w.Write([]byte(`{"error":"rate limited","code":"rate_limited","retryable":true,"trace_id":"retry-trace","parameter":null}`)); err != nil {
			t.Errorf("write test response: %v", err)
		}
	}))
	defer server.Close()
	var apiError *types.Error
	err := NewClient(server.Client(), server.URL).Get(context.Background(), "/v2/trades", nil, nil)
	if !errors.As(err, &apiError) || apiError.Message != "rate limited" || apiError.Status != http.StatusTooManyRequests || apiError.TraceID != "retry-trace" || apiError.Retryable == nil || !*apiError.Retryable {
		t.Fatalf("lost retry metadata: %+v (%v)", apiError, err)
	}
	if calls != defaultMaxRetries+1 {
		t.Fatalf("retry budget changed: %d", calls)
	}
}
