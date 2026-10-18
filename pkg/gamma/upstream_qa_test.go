package gamma

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/neor-it/polymarket-go-sdk/pkg/transport"
)

const upstreamQACursor = "opaque+/= ?cursor"

func TestUpstreamQAGammaKeysetUsesAfterCursorAndObjectEnvelope(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodGet || r.URL.Query().Has("offset") || r.URL.Query().Has("next_cursor") {
			t.Errorf("unsupported keyset query: %s %s", r.Method, r.URL)
		}
		after := r.URL.Query().Get("after_cursor")
		if after != "" && after != upstreamQACursor {
			t.Errorf("opaque cursor changed: %q", after)
		}
		switch r.URL.Path {
		case "/markets/keyset":
			if after == "" {
				_, _ = io.WriteString(w, `{"markets":[{"id":"qa-market","question":"Exact envelope"}],"next_cursor":"`+upstreamQACursor+`"}`)
			} else {
				_, _ = io.WriteString(w, `{"markets":[]}`)
			}
		case "/events/keyset":
			if after == "" {
				_, _ = io.WriteString(w, `{"events":[{"id":"qa-event","title":"Exact event envelope"}],"next_cursor":"`+upstreamQACursor+`"}`)
			} else {
				_, _ = io.WriteString(w, `{"events":[]}`)
			}
		default:
			t.Errorf("unexpected Gamma route %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client := NewClient(transport.NewClient(server.Client(), server.URL))
	markets, err := client.MarketsKeyset(t.Context(), nil)
	if err != nil || len(markets.Markets) != 1 || markets.Markets[0].ID != "qa-market" || markets.NextCursor != upstreamQACursor {
		t.Fatalf("market envelope: %+v %v", markets, err)
	}
	lastMarkets, err := client.MarketsKeyset(t.Context(), &MarketsRequest{AfterCursor: markets.NextCursor})
	if err != nil || lastMarkets.Markets == nil || len(lastMarkets.Markets) != 0 || lastMarkets.NextCursor != "" {
		t.Fatalf("last market page: %+v %v", lastMarkets, err)
	}
	events, err := client.EventsKeyset(t.Context(), nil)
	if err != nil || len(events.Events) != 1 || events.Events[0].ID != "qa-event" || events.NextCursor != upstreamQACursor {
		t.Fatalf("event envelope: %+v %v", events, err)
	}
	lastEvents, err := client.EventsKeyset(t.Context(), &EventsRequest{NextCursor: events.NextCursor})
	if err != nil || lastEvents.Events == nil || len(lastEvents.Events) != 0 || lastEvents.NextCursor != "" {
		t.Fatalf("last event page: %+v %v", lastEvents, err)
	}
	const expectedCalls = 4
	if requests.Load() != expectedCalls {
		t.Fatalf("unexpected pagination calls: %d", requests.Load())
	}
	zero := 0
	before := requests.Load()
	for _, req := range []*MarketsRequest{{Offset: &zero}, {AfterCursor: "a", NextCursor: "b"}} {
		if _, err := client.MarketsKeyset(t.Context(), req); err == nil {
			t.Fatal("unsupported offset/ambiguous market cursor was accepted")
		}
	}
	for _, req := range []*EventsRequest{{Offset: &zero}, {AfterCursor: "a", NextCursor: "b"}} {
		if _, err := client.EventsKeyset(t.Context(), req); err == nil {
			t.Fatal("unsupported offset/ambiguous event cursor was accepted")
		}
	}
	if requests.Load() != before {
		t.Fatal("invalid keyset requests reached the provider")
	}
}
