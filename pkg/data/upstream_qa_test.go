package data

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/neor-it/polymarket-go-sdk/pkg/transport"
	"github.com/neor-it/polymarket-go-sdk/pkg/types"
)

const upstreamQASnapshotUser = "0x1234567890123456789012345678901234567890"
const upstreamQASnapshotDeadline = 30 * time.Millisecond
const upstreamQASnapshotCSV = "asset_id,amount\n115792089237316195423570985008687907853269984665640564039457584007913129639935,0.000001\n"

func TestUpstreamQAAccountingSnapshotPreservesZIPAndTransportFailures(t *testing.T) {
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	entry, err := writer.Create("positions.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = io.WriteString(entry, upstreamQASnapshotCSV); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	const (
		valid = iota
		malformed
		forbidden
		throttled
		hanging
	)
	var mode, calls atomic.Int32
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/v1/accounting/snapshot" || r.URL.Query().Get("user") != common.HexToAddress(upstreamQASnapshotUser).Hex() || r.Header.Get("Accept") != "application/zip" {
			t.Errorf("accounting wire mismatch: %s %s %v", r.Method, r.URL, r.Header)
		}
		switch mode.Load() {
		case malformed:
			_, _ = io.WriteString(w, `{"error":"not an archive"}`)
		case forbidden:
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"error":"unauthorized"}`)
		case throttled:
			w.Header().Set("Retry-After", "3600")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(w, `{"error":"rate limited","code":"rate_limited","retryable":true}`)
		case hanging:
			close(started)
			<-r.Context().Done()
		default:
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(archive.Bytes())
		}
	}))
	defer server.Close()
	client := NewClient(transport.NewClient(server.Client(), server.URL))
	request := &AccountingSnapshotRequest{User: common.HexToAddress(upstreamQASnapshotUser)}
	body, err := client.AccountingSnapshot(t.Context(), request)
	if err != nil || !bytes.Equal(body, archive.Bytes()) {
		t.Fatalf("ZIP bytes changed: %v", err)
	}
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	file, err := reader.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	csv, err := io.ReadAll(file)
	_ = file.Close()
	if err != nil || string(csv) != upstreamQASnapshotCSV {
		t.Fatalf("binary CSV payload corrupted: %q %v", csv, err)
	}
	before := calls.Load()
	for _, invalid := range []*AccountingSnapshotRequest{nil, {}} {
		if _, err := client.AccountingSnapshot(t.Context(), invalid); err == nil {
			t.Fatal("missing required user was accepted")
		}
	}
	if calls.Load() != before {
		t.Fatal("missing-user requests reached HTTP")
	}
	mode.Store(malformed)
	if _, err := client.AccountingSnapshot(t.Context(), request); err == nil {
		t.Fatal("HTTP200 JSON error was accepted as an accounting ZIP")
	}
	mode.Store(forbidden)
	_, err = client.AccountingSnapshot(t.Context(), request)
	var apiError *types.Error
	if !errors.As(err, &apiError) || apiError.Status != http.StatusUnauthorized || apiError.Message != "unauthorized" {
		t.Fatalf("binary path swallowed typed HTTP error: %v", err)
	}
	mode.Store(throttled)
	before = calls.Load()
	ctx, cancel := context.WithTimeout(t.Context(), upstreamQASnapshotDeadline)
	_, err = client.AccountingSnapshot(ctx, request)
	cancel()
	if !errors.Is(err, context.DeadlineExceeded) || calls.Load() != before+1 {
		t.Fatalf("Retry-After/cancellation was bypassed by binary path: calls=%d error=%v", calls.Load()-before, err)
	}
	mode.Store(hanging)
	ctx, cancel = context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { _, err := client.AccountingSnapshot(ctx, request); done <- err }()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("in-flight binary request cancellation lost: %v", err)
	}
}
