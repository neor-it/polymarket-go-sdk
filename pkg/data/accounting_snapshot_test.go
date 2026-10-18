package data

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/neor-it/polymarket-go-sdk/pkg/transport"
	"github.com/neor-it/polymarket-go-sdk/pkg/types"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAccountingSnapshotBinaryAndErrors(t *testing.T) {
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	for _, name := range []string{"positions.csv", "equity.csv"} {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = fmt.Fprint(file, "amount\n0\n")
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	user := common.HexToAddress("0x1111111111111111111111111111111111111111")
	failed := false
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/v1/accounting/snapshot" || r.URL.Query().Get("user") != user.Hex() || r.Header.Get("Accept") != "application/zip" {
			t.Fatalf("incorrect snapshot request %s", r.URL.String())
		}
		if failed {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprint(w, `{"error":"invalid wallet"}`)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(archive.Bytes())
	}))
	defer server.Close()
	c := NewClient(transport.NewClient(server.Client(), server.URL))
	req := &AccountingSnapshotRequest{User: user}
	result, err := c.AccountingSnapshot(context.Background(), req)
	if err != nil || !bytes.Equal(result, archive.Bytes()) {
		t.Fatalf("ZIP changed: %v", err)
	}
	failed = true
	_, err = c.AccountingSnapshot(context.Background(), req)
	var apiErr *types.Error
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusBadRequest {
		t.Fatalf("typed API error lost: %v", err)
	}
	if _, err = c.AccountingSnapshot(context.Background(), nil); err == nil {
		t.Fatal("missing user accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = c.AccountingSnapshot(ctx, req)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	const expectedCalls = 2
	if calls != expectedCalls {
		t.Fatalf("unexpected requests %d", calls)
	}
}
