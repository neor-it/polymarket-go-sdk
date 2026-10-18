package data

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/neor-it/polymarket-go-sdk/pkg/transport"
)

const tokenReferenceBaseline = `{"token_id":"123","condition_id":"0x1234","module":"v1_ctf","outcome_index":0,"resolved":false,"clob_index":null,"outcome":null,"final_price":null}`

func TestTokenReferenceRequiresDocumentedIdentityWithoutInventingOutcomeZero(t *testing.T) {
	snapshot, err := os.ReadFile("testdata/openapi-v2.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Components struct {
			Schemas map[string]struct {
				Required []string `json:"required"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(snapshot, &schema); err != nil {
		t.Fatal(err)
	}
	var baseline map[string]json.RawMessage
	if err := json.Unmarshal([]byte(tokenReferenceBaseline), &baseline); err != nil {
		t.Fatal(err)
	}
	for _, field := range schema.Components.Schemas["TokenRef"].Required {
		for _, missing := range []bool{false, true} {
			t.Run(field+map[bool]string{false: " null", true: " missing"}[missing], func(t *testing.T) {
				candidate := make(map[string]json.RawMessage, len(baseline))
				for key, value := range baseline {
					candidate[key] = value
				}
				if missing {
					delete(candidate, field)
				} else {
					candidate[field] = json.RawMessage("null")
				}
				payload, err := json.Marshal(candidate)
				if err != nil {
					t.Fatal(err)
				}
				var ref TokenRef
				if err := json.Unmarshal(payload, &ref); err == nil || !strings.Contains(err.Error(), field) {
					t.Fatalf("unavailable %s accepted: ref=%+v err=%v", field, ref, err)
				}
			})
		}
	}
	for _, invalid := range []string{`null`, `{}`, `[]`, `"wrong"`} {
		var ref TokenRef
		if err := json.Unmarshal([]byte(invalid), &ref); err == nil {
			t.Fatalf("invalid row accepted: %s", invalid)
		}
	}
}

func TestTokensHTTPRejectsUnavailableOutcomeButAcceptsExplicitZero(t *testing.T) {
	for _, outcome := range []string{"0", "1", "null", "missing", `"0"`, "0.5", "2147483648"} {
		t.Run(outcome, func(t *testing.T) {
			row := strings.Replace(tokenReferenceBaseline, `"outcome_index":0`, `"outcome_index":`+outcome, 1)
			if outcome == "missing" {
				row = strings.Replace(tokenReferenceBaseline, `"outcome_index":0,`, "", 1)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v2/tokens" || r.URL.Query().Get("token_id") != "123" {
					t.Errorf("unexpected token query: %s", r.URL.String())
				}
				_, _ = w.Write([]byte(`{"data":[` + row + `]}`))
			}))
			defer server.Close()
			client := NewClient(transport.NewClient(server.Client(), server.URL))
			result, err := client.Tokens(context.Background(), &TokensRequest{TokenID: ptr("123")})
			if outcome != "0" && outcome != "1" {
				if err == nil {
					t.Fatalf("invalid outcome accepted: %+v", result)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Data) != 1 || result.Data[0].Resolved || result.Data[0].FinalPrice != nil || result.Data[0].Outcome != nil {
				t.Fatalf("explicit reference facts changed: %+v", result)
			}
		})
	}
}
