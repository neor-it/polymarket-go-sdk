package combos

import (
	"net/http"
	"testing"

	"github.com/neor-it/polymarket-go-sdk/pkg/auth"
)

func TestBuildHMACHeaders(t *testing.T) {
	headers, err := buildAccountHeaders(AccountAuth{
		Address: "0x0000000000000000000000000000000000000001",
		Credentials: Credentials{
			Key:        "key",
			Secret:     "c2VjcmV0",
			Passphrase: "pass",
		},
	}, http.MethodPost, "/v1/builder/rfq/requests", `{"x":"y"}`, 1700000000)
	if err != nil {
		t.Fatalf("buildAccountHeaders returned error: %v", err)
	}

	if got := headers.Get(auth.HeaderPolySignature); got != "7cbkDOF6fe1vX6eswxdGvx8NnhVuf_iOmVTCqilTXKc=" {
		t.Fatalf("signature = %q", got)
	}
	if got := headers.Get(auth.HeaderPolyAddress); got != "0x0000000000000000000000000000000000000001" {
		t.Fatalf("address = %q", got)
	}
	if got := headers.Get(auth.HeaderPolyTimestamp); got != "1700000000" {
		t.Fatalf("timestamp = %q", got)
	}
}

func TestBuildBuilderHeaders(t *testing.T) {
	headers, err := buildBuilderHeaders(BuilderAuth{
		Credentials: Credentials{
			Key:        "builder-key",
			Secret:     "c2VjcmV0",
			Passphrase: "builder-pass",
		},
	}, http.MethodPost, "/v1/builder/rfq/requests", `{"x":"y"}`, 1700000000)
	if err != nil {
		t.Fatalf("buildBuilderHeaders returned error: %v", err)
	}

	if got := headers.Get(auth.HeaderPolyBuilderSignature); got != "7cbkDOF6fe1vX6eswxdGvx8NnhVuf_iOmVTCqilTXKc=" {
		t.Fatalf("builder signature = %q", got)
	}
	if got := headers.Get(auth.HeaderPolyBuilderAPIKey); got != "builder-key" {
		t.Fatalf("builder key = %q", got)
	}
}
