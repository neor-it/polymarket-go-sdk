package transport

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
)

const upstreamQATransportOwner = "0x1234567890123456789012345678901234567890"
const upstreamQAServerTimestamp = "1700000000"

func TestUpstreamQAServerTimeObjectWorksForAutomaticL2Authentication(t *testing.T) {
	for _, response := range []string{upstreamQAServerTimestamp, `{"timestamp":` + upstreamQAServerTimestamp + `}`} {
		t.Run(response, func(t *testing.T) {
			var authenticatedCalls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/time" {
					_, _ = io.WriteString(w, response)
					return
				}
				if r.URL.Path != "/private" || r.Header.Get(auth.HeaderPolyTimestamp) != upstreamQAServerTimestamp || r.Header.Get(auth.HeaderPolySignature) == "" {
					t.Errorf("wrong server-time authenticated request: %s %v", r.URL, r.Header)
				}
				authenticatedCalls.Add(1)
				_, _ = io.WriteString(w, `{}`)
			}))
			defer server.Close()
			signer, err := auth.NewAddressOnlySigner(upstreamQATransportOwner, auth.PolygonChainID)
			if err != nil {
				t.Fatal(err)
			}
			client := NewClient(server.Client(), server.URL)
			client.SetAuth(signer, &auth.APIKey{Key: "qa-key", Secret: "c2VjcmV0", Passphrase: "qa-pass"})
			client.SetUseServerTime(true)
			if err := client.Get(t.Context(), "/private", nil, nil); err != nil {
				t.Fatalf("supported /time response cannot authenticate request: %v", err)
			}
			if authenticatedCalls.Load() != 1 {
				t.Fatalf("authenticated request count: %d", authenticatedCalls.Load())
			}
		})
	}
}

func TestUpstreamQARawTransportPreservesNonJSONBytes(t *testing.T) {
	const binarySize = 256
	body := make([]byte, binarySize)
	for index := range body {
		body[index] = byte(index)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/octet-stream" {
			t.Errorf("binary Accept header overwritten: %v", r.Header)
		}
		_, _ = w.Write(body)
	}))
	defer server.Close()
	actual, err := NewClient(server.Client(), server.URL).GetRaw(t.Context(), "/binary", nil, map[string]string{"Accept": "application/octet-stream"})
	if err != nil || !bytes.Equal(actual, body) {
		t.Fatalf("raw transport decoded or changed binary data: %v", err)
	}
}
