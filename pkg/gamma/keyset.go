package gamma

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

const maxKeysetPageSize = 100

// EventsKeysetResponse retains the opaque continuation cursor.
type EventsKeysetResponse struct {
	Events     []Event `json:"events"`
	NextCursor string  `json:"next_cursor,omitempty"`
}

// MarketsKeysetResponse retains the opaque continuation cursor.
type MarketsKeysetResponse struct {
	Markets    []Market `json:"markets"`
	NextCursor string   `json:"next_cursor,omitempty"`
}

// EventsKeyset calls the corresponding typed API endpoint and preserves transport errors.
func (c *clientImpl) EventsKeyset(ctx context.Context, req *EventsRequest) (EventsKeysetResponse, error) {
	var resp EventsKeysetResponse
	if req != nil && req.Offset != nil {
		return resp, fmt.Errorf("event keyset does not accept offset")
	}
	if req != nil && req.Limit != nil && (*req.Limit <= 0 || *req.Limit > maxKeysetPageSize) {
		return resp, fmt.Errorf("event keyset limit must be 1 to %d", maxKeysetPageSize)
	}
	q := buildEventsQuery(req)
	if req != nil && len(req.Order) > 0 {
		q.Set("order", strings.Join(req.Order, ","))
	}
	if req != nil {
		if req.AfterCursor != "" && req.NextCursor != "" {
			return resp, fmt.Errorf("specify only one event cursor")
		}
		cursor := req.AfterCursor
		if cursor == "" {
			cursor = req.NextCursor
		}
		addString(q, "after_cursor", cursor)
	}
	err := c.httpClient.Get(ctx, "/events/keyset", q, &resp)
	return resp, err
}

// MarketsKeyset calls the corresponding typed API endpoint and preserves transport errors.
func (c *clientImpl) MarketsKeyset(ctx context.Context, req *MarketsRequest) (MarketsKeysetResponse, error) {
	var resp MarketsKeysetResponse
	if req != nil && req.Offset != nil {
		return resp, fmt.Errorf("market keyset does not accept offset")
	}
	if req != nil && req.Limit != nil && (*req.Limit <= 0 || *req.Limit > maxKeysetPageSize) {
		return resp, fmt.Errorf("market keyset limit must be 1 to %d", maxKeysetPageSize)
	}
	q := buildMarketsQuery(req)
	if req != nil {
		if req.AfterCursor != "" && req.NextCursor != "" {
			return resp, fmt.Errorf("specify only one market cursor")
		}
		cursor := req.AfterCursor
		if cursor == "" {
			cursor = req.NextCursor
		}
		addString(q, "after_cursor", cursor)
	}
	err := c.httpClient.Get(ctx, "/markets/keyset", q, &resp)
	return resp, err
}

// TeamByID calls the corresponding typed API endpoint and preserves transport errors.
func (c *clientImpl) TeamByID(ctx context.Context, id string) (*Team, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("team ID is required")
	}
	var resp Team
	if err := c.httpClient.Get(ctx, "/teams/"+url.PathEscape(id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
