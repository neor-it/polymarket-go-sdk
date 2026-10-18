package types

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAPIErrorSupportsCLOBAndDataV2(t *testing.T) {
	tests := []struct {
		body    string
		message string
	}{
		{`{"message":"CLOB order rejected","code":"bad_order"}`, "CLOB order rejected"},
		{`{"error":"cursor does not match query","code":"invalid_request","retryable":false,"trace_id":"opaque","parameter":"cursor"}`, "cursor does not match query"},
	}
	for _, test := range tests {
		var decoded Error
		if err := json.Unmarshal([]byte(test.body), &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.Message != test.message || !strings.Contains(decoded.Error(), test.message) {
			t.Fatalf("lost message %+v", decoded)
		}
	}
}
