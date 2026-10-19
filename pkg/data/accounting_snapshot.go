package data

import (
	"archive/zip"
	"bytes"
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"net/url"
)

// AccountingSnapshotRequest identifies the wallet whose accounting CSVs are requested.
type AccountingSnapshotRequest struct {
	User common.Address `json:"user"`
}

// AccountingSnapshot downloads the ZIP without routing its bytes through JSON decoding.
func (c *clientImpl) AccountingSnapshot(ctx context.Context, req *AccountingSnapshotRequest) ([]byte, error) {
	if req == nil || req.User == (common.Address{}) {
		return nil, errors.Errorf("accounting snapshot user is required")
	}
	body, err := c.httpClient.GetRaw(ctx, "/v1/accounting/snapshot", url.Values{"user": {req.User.Hex()}}, map[string]string{"Accept": "application/zip"})
	if err != nil {
		return nil, errors.Wrap(err, "download accounting snapshot")
	}
	if _, err := zip.NewReader(bytes.NewReader(body), int64(len(body))); err != nil {
		return nil, errors.Wrap(err, "decode accounting ZIP archive")
	}
	return body, nil
}
