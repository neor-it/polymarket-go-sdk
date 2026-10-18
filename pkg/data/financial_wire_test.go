package data

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestRequiredFinancialFieldsDoNotInventZero(t *testing.T) {
	snapshot, err := os.ReadFile("testdata/openapi-v2.json")
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Components struct {
			Schemas map[string]struct {
				Required   []string `json:"required"`
				Properties map[string]struct {
					Type json.RawMessage `json:"type"`
				} `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(snapshot, &document); err != nil {
		t.Fatal(err)
	}
	rows := []any{Activity{}, BiggestWinner{}, BuilderStanding{}, BuilderVolumePoint{}, ComboLeg{}, ComboPosition{}, ConditionVolume{}, Holder{}, LeaderboardEntry{}, LeaderboardUserEntry{}, LiveVolume{}, OpenInterest{}, PortfolioValue{}, Position{}, PricePoint{}, Trade{}, UserPnlPoint{}, UserStats{}, UserVolume{}}
	for _, row := range rows {
		typ := reflect.TypeOf(row)
		t.Run(typ.Name(), func(t *testing.T) {
			schema := document.Components.Schemas[typ.Name()]
			fields := []string{}
			baseline := map[string]any{}
			for _, field := range schema.Required {
				if string(schema.Properties[field].Type) == `"number"` {
					fields = append(fields, field)
					baseline[field] = 0
				}
			}
			if len(fields) == 0 {
				t.Fatal("no financial assertions")
			}
			payload, err := json.Marshal(baseline)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(payload, reflect.New(typ).Interface()); err != nil {
				t.Fatalf("explicit financial zeros rejected: %v", err)
			}
			for _, field := range fields {
				for _, missing := range []bool{false, true} {
					candidate := make(map[string]any, len(baseline))
					for key, value := range baseline {
						candidate[key] = value
					}
					if missing {
						delete(candidate, field)
					} else {
						candidate[field] = nil
					}
					payload, err := json.Marshal(candidate)
					if err != nil {
						t.Fatal(err)
					}
					if err := json.Unmarshal(payload, reflect.New(typ).Interface()); err == nil {
						t.Errorf("%s missing=%t invented zero", field, missing)
					}
				}
			}
		})
	}
}
