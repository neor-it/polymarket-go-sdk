package transport

import (
	"context"
	"net/http"
	"net/url"
)

type rawDestination struct{ body []byte }

// GetRaw retrieves bytes through the normal authentication, retry, rate-limit and error pipeline.
// Headers may override Accept for a binary endpoint; error statuses still return typed API errors.
func (c *Client) GetRaw(ctx context.Context, path string, query url.Values, headers map[string]string) ([]byte, error) {
	dest := rawDestination{}
	if err := c.CallWithHeaders(ctx, http.MethodGet, path, query, nil, &dest, headers); err != nil {
		return nil, err
	}
	return dest.body, nil
}
