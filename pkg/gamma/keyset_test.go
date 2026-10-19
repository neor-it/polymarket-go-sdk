package gamma

import (
	"context"
	"fmt"
	"github.com/neor-it/polymarket-go-sdk/pkg/transport"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestKeysetUsesOfficialCursorAndTypedPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("offset") || r.URL.Query().Has("next_cursor") {
			t.Fatal("legacy pagination on keyset")
		}
		if r.URL.Path == "/teams/7" {
			_, _ = fmt.Fprint(w, `{"id":7,"name":"Team"}`)
			return
		}
		if r.URL.Query().Get("after_cursor") != "opaque+/=" {
			t.Fatal("cursor changed")
		}
		if r.URL.Path == "/events/keyset" {
			if r.URL.Query().Get("order") != "volume,id" {
				t.Fatal("incorrect event order")
			}
			_, _ = fmt.Fprint(w, `{"events":[{"id":"e"}],"next_cursor":"next"}`)
			return
		}
		_, _ = fmt.Fprint(w, `{"markets":[],"next_cursor":""}`)
	}))
	defer server.Close()
	c := NewClient(transport.NewClient(server.Client(), server.URL))
	ctx := context.Background()
	events, err := c.EventsKeyset(ctx, &EventsRequest{AfterCursor: "opaque+/=", Order: []string{"volume", "id"}})
	if err != nil || len(events.Events) != 1 || events.NextCursor != "next" {
		t.Fatalf("event page %v %v", events, err)
	}
	markets, err := c.MarketsKeyset(ctx, &MarketsRequest{NextCursor: "opaque+/="})
	if err != nil || len(markets.Markets) != 0 || markets.NextCursor != "" {
		t.Fatalf("market page %v %v", markets, err)
	}
	if _, err = c.TeamByID(ctx, "7"); err != nil {
		t.Fatal(err)
	}
	offset := 0
	if _, err = c.EventsKeyset(ctx, &EventsRequest{Offset: &offset}); err == nil {
		t.Fatal("explicit zero offset accepted")
	}
	if _, err = c.MarketsKeyset(ctx, &MarketsRequest{AfterCursor: "a", NextCursor: "b"}); err == nil {
		t.Fatal("ambiguous cursor accepted")
	}
}
